package bench

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"

	ay "github.com/aytechnet/decimal"
	ss "github.com/shopspring/decimal"
)

// TestRoundParity checks rounding parity with shopspring/decimal: every rounding entry point must give the same value on a
// grid of 4-decimal values in [-3, 3], negative halves included (a credit note must round like its
// invoice). Run from this directory: go test -run RoundParity
func TestRoundParity(t *testing.T) {
	type op struct {
		name string
		a    func(ay.Decimal, int32) string
		s    func(ss.Decimal, int32) string
		num  bool
	}
	ops := []op{
		{"Round", func(d ay.Decimal, p int32) string { return d.Round(p).String() }, func(d ss.Decimal, p int32) string { return d.Round(p).String() }, true},
		{"StringFixed", func(d ay.Decimal, p int32) string { return d.StringFixed(p) }, func(d ss.Decimal, p int32) string { return d.StringFixed(p) }, false},
		{"RoundBank", func(d ay.Decimal, p int32) string { return d.RoundBank(p).String() }, func(d ss.Decimal, p int32) string { return d.RoundBank(p).String() }, true},
		{"StringFixedBank", func(d ay.Decimal, p int32) string { return d.StringFixedBank(p) }, func(d ss.Decimal, p int32) string { return d.StringFixedBank(p) }, false},
		{"RoundUp", func(d ay.Decimal, p int32) string { return d.RoundUp(p).String() }, func(d ss.Decimal, p int32) string { return d.RoundUp(p).String() }, true},
		{"RoundDown", func(d ay.Decimal, p int32) string { return d.RoundDown(p).String() }, func(d ss.Decimal, p int32) string { return d.RoundDown(p).String() }, true},
		{"RoundCeil", func(d ay.Decimal, p int32) string { return d.RoundCeil(p).String() }, func(d ss.Decimal, p int32) string { return d.RoundCeil(p).String() }, true},
		{"RoundFloor", func(d ay.Decimal, p int32) string { return d.RoundFloor(p).String() }, func(d ss.Decimal, p int32) string { return d.RoundFloor(p).String() }, true},
		{"Truncate", func(d ay.Decimal, p int32) string { return d.Truncate(p).String() }, func(d ss.Decimal, p int32) string { return d.Truncate(p).String() }, true},
		{"DivRound3", func(d ay.Decimal, p int32) string { return d.DivRound(ay.New(3, 0), p).String() }, func(d ss.Decimal, p int32) string { return d.DivRound(ss.New(3, 0), p).String() }, true},
		{"DivRound8", func(d ay.Decimal, p int32) string { return d.DivRound(ay.New(8, 0), p).String() }, func(d ss.Decimal, p int32) string { return d.DivRound(ss.New(8, 0), p).String() }, true},
		{"RoundCash", func(d ay.Decimal, p int32) string { return d.RoundCash(uint8(p)).String() }, func(d ss.Decimal, p int32) string { return d.RoundCash(uint8(p)).String() }, true},
	}
	for _, o := range ops {
		places := []int32{-1, 0, 1, 2, 3}
		switch o.name {
		case "RoundCash":
			places = []int32{5, 10, 25, 50, 100}
		}
		bad, ex := 0, ""
		for n := int64(-30000); n <= 30000; n += 1 {
			for _, p := range places {
				da, ds := ay.New(n, -4), ss.New(n, -4)
				ga, gs := o.a(da, p), o.s(ds, p)
				eq := ga == gs
				if !eq && o.num {
					x, err := ss.NewFromString(ga)
					eq = err == nil && x.Equal(ss.RequireFromString(gs))
				}
				if !eq {
					bad++
					if len(ex) < 300 {
						ex += fmt.Sprintf(" %s@%d: aytechnet %s, shopspring %s;", ds, p, ga, gs)
					}
				}
			}
		}
		if bad > 0 {
			t.Errorf("%s: %d mismatches with shopspring:%s", o.name, bad, ex)
		}
	}
}

// TestNewFromFloatWithExponentParity checks NewFromFloatWithExponent against shopspring on random
// float64 values: same exponent convention (exp -2 keeps 2 places) and same rounding of the exact
// binary value, half away from zero. Beyond the ~17 significant digits of the 57-bit mantissa the
// result keeps ~17 significant digits and must be within one unit of its last digit.
func TestNewFromFloatWithExponentParity(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	bad, ex := 0, ""
	for i := 0; i < 200000; i++ {
		var f float64
		switch i % 4 {
		case 0: // money-like values with a few decimals, often near a tie
			f = float64(rng.Int63n(2000000)-1000000) / 1000
		case 1:
			f = (rng.Float64() - 0.5) * math.Pow(10, float64(rng.Intn(16)-6))
		case 2: // exact binary ties
			f = float64(rng.Int63n(20000)-10000) / 8
		default:
			f = float64(rng.Int63n(1<<53)-(1<<52)) * math.Pow(2, float64(rng.Intn(60)-70))
		}
		for exp := int32(-12); exp <= 3; exp++ {
			da := ay.NewFromFloatWithExponent(f, exp)
			ga := da.String()
			gs := ss.NewFromFloatWithExponent(f, exp)
			x, err := ss.NewFromString(strings.TrimPrefix(ga, "~"))
			ok := err == nil && x.Equal(gs)
			if !ok && err == nil && (!da.IsExact() || gs.NumDigits() > 17) {
				// more digits than the 57-bit mantissa holds: aytechnet keeps ~17 significant
				// digits, so allow one unit of its last digit
				ok = x.Sub(gs).Abs().LessThanOrEqual(ss.New(1, da.Exponent()))
			}
			if !ok {
				bad++
				if len(ex) < 400 {
					ex += fmt.Sprintf(" %v@%d: aytechnet %s, shopspring %s;", f, exp, ga, gs)
				}
			}
		}
	}
	if bad > 0 {
		t.Errorf("NewFromFloatWithExponent: %d mismatches with shopspring:%s", bad, ex)
	}
}

// TestNewFromFloatParity checks NewFromFloat and NewFromFloat32 against shopspring: both use the
// shortest decimal that converts back to the same float, so money values stay exact (-178464.49).
// A Decimal holds at most 16 fractional digits (and ±~0 below): a shortest decimal with more is
// rounded to them with the loss bit set.
func TestNewFromFloatParity(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	bad, ex := 0, ""
	for i := 0; i < 1000000; i++ {
		var f float64
		switch i % 3 {
		case 0: // money-like values
			f = float64(rng.Int63n(2000000000)-1000000000) / 100
		case 1:
			f = (rng.Float64() - 0.5) * math.Pow(10, float64(rng.Intn(30)-12))
		default:
			f = float64(rng.Int63n(1<<53)-(1<<52)) * math.Pow(2, float64(rng.Intn(80)-80))
		}
		da, da32 := ay.NewFromFloat(f), ay.NewFromFloat32(float32(f))
		ds, ds32 := ss.NewFromFloat(f), ss.NewFromFloat32(float32(f))
		ga, gs, ga32, gs32 := da.String(), ds.String(), da32.String(), ds32.String()
		if !sameShortest(da, ds) || !sameShortest(da32, ds32) {
			bad++
			if len(ex) < 400 {
				ex += fmt.Sprintf(" %v: aytechnet %s/%s, shopspring %s/%s;", f, ga, ga32, gs, gs32)
			}
		}
	}
	if bad > 0 {
		t.Errorf("NewFromFloat: %d mismatches with shopspring:%s", bad, ex)
	}
}

// sameShortest reports whether a equals s exactly, or, when a carries the loss bit, equals s rounded
// to a's exponent (or a is a near zero for an s below its range).
func sameShortest(a ay.Decimal, s ss.Decimal) bool {
	x, err := ss.NewFromString(strings.TrimPrefix(a.String(), "~"))
	if err != nil {
		return a.IsExact() == false && s.Abs().LessThan(ss.New(1, -16))
	}
	if a.IsExact() {
		return x.Equal(s)
	}
	return x.Equal(s.Round(-a.Exponent()))
}
