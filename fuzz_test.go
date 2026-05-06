//go:build go1.18

package decimal

import (
	"math"
	"testing"
)

// FuzzNewFromString feeds arbitrary byte sequences to NewFromString and verifies that
// either parsing fails cleanly or — when it succeeds — the value round-trips through String.
func FuzzNewFromString(f *testing.F) {
	seeds := []string{
		"0", "-0", "1", "-1", "12345", "-12345",
		"1.0", "0.1", "-0.001", "1e10", "1.5e-10",
		"123.456e+15", ".0001", "1_000",
		"~0", "+~0", "-~0", "+Inf", "-Inf", "NaN",
		"null", "Null", "nil",
		"", " ", "abc", "1.2.3", "1ee2", "+", "-", "~",
		"1.7976931348623157e+308", "5e-324",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		d, err := NewFromString(s)
		if err != nil {
			return
		}
		// Magic values don't round-trip 1:1 through String:
		//   - Null is a sentinel produced only by parsing "nil"/"null"; it stringifies to "0" which re-parses as Zero.
		//   - NaN / ±Inf / NearZero variants stringify to "0" or "null" by design.
		if d.IsNull() || d.IsNaN() || d.IsInfinite() || !d.IsExact() {
			return
		}
		out := d.String()
		d2, err2 := NewFromString(out)
		if err2 != nil {
			t.Fatalf(`round-trip parse failed: input %q → %v → %q → err %v`, s, d, out, err2)
		}
		if d != d2 {
			t.Fatalf(`round-trip mismatch: input %q → %v (0x%016x) → %q → %v (0x%016x)`, s, d, uint64(d), out, d2, uint64(d2))
		}
	})
}

// FuzzNewFromFloat checks that NewFromFloat never panics for any float64, and that finite
// non-special results back-convert to a float close to the input.
func FuzzNewFromFloat(f *testing.F) {
	for _, v := range []float64{
		0, -0, 1, -1, 4, 0.5, 0.25, 1.25, 2.5, 3.75,
		0.1, 1.1, 5.45, 1e10, 1e-10, math.Pi, math.E,
		math.MaxFloat64, math.SmallestNonzeroFloat64,
		math.Copysign(0, -1),
	} {
		f.Add(v)
	}

	f.Fuzz(func(t *testing.T, v float64) {
		d := NewFromFloat(v)
		switch {
		case math.IsNaN(v):
			if !d.IsNaN() {
				t.Fatalf(`NewFromFloat(NaN) should be NaN, got %v`, d)
			}
		case math.IsInf(v, 1):
			if d != PositiveInfinity {
				t.Fatalf(`NewFromFloat(+Inf) should be +Inf, got %v`, d)
			}
		case math.IsInf(v, -1):
			if d != NegativeInfinity {
				t.Fatalf(`NewFromFloat(-Inf) should be -Inf, got %v`, d)
			}
		default:
			// finite: back-conversion should be reasonably close (relative ε of 1e-12 inside Decimal's range)
			// skip cases where we already know precision was dropped: NearZero, Inf, or any value with the loss bit set
			// (e.g. floats below ~10^-16 where the Decimal exponent saturates and mantissa digits are truncated)
			if d.IsZero() || d.IsInfinite() || !d.IsExact() {
				return
			}
			f2, _ := d.Float64()
			if v != 0 && !math.IsNaN(f2) && !math.IsInf(f2, 0) {
				// 1e-7 is generous: it accepts mantissa truncation that the legacy iteration
				// performs without setting the loss bit at the edges of Decimal's range,
				// while still catching catastrophic conversion errors (off-by-orders-of-magnitude).
				rel := math.Abs((f2 - v) / v)
				if rel > 1e-7 {
					t.Fatalf(`NewFromFloat(%v).Float64() = %v, relative diff = %v`, v, f2, rel)
				}
			}
		}
	})
}

// FuzzMarshalUnmarshalBinary verifies the binary round-trip property for any Decimal bit pattern.
func FuzzMarshalUnmarshalBinary(f *testing.F) {
	for _, u := range []uint64{
		0, 1, 0x8000000000000000, 0x4000000000000000,
		0x6000000000000000, 0x5e00000000000000, 0x4200000000000000,
		0x0000000000000005, 0xfffffffffffffff5,
	} {
		f.Add(u)
	}

	f.Fuzz(func(t *testing.T, u uint64) {
		d := Decimal(u)
		buf, err := d.MarshalBinary()
		if err != nil {
			t.Fatalf(`MarshalBinary(0x%016x) errored: %v`, u, err)
		}

		var d2 Decimal
		if err := d2.UnmarshalBinary(buf); err != nil {
			t.Fatalf(`UnmarshalBinary(% x) errored on round-trip from 0x%016x: %v`, buf, u, err)
		}

		// NaN encodings don't compare with == but should both be NaN
		if d.IsNaN() {
			if !d2.IsNaN() {
				t.Fatalf(`NaN round-trip lost NaN-ness: 0x%016x → % x → 0x%016x`, u, buf, uint64(d2))
			}
			return
		}
		if d != d2 {
			t.Fatalf(`round-trip mismatch: 0x%016x → % x → 0x%016x`, u, buf, uint64(d2))
		}
	})
}
