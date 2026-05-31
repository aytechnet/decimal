// Transcendental / trigonometric comparison: Ln, Sin, Cos, Tan, and the inverse
// Atan (the only inverse trig both packages expose), plus aytechnet-only Sqrt.
// Run from this directory:
//
//	go test -bench='Ln|Sin|Cos|Tan|Atan|Sqrt' -benchmem -count=6
//
// Unlike the arithmetic benchmarks these are ITERATIVE functions, not hot-path,
// and the two libraries do NOT share a precision model: aytechnet caps at its
// ~17-digit int64 mantissa, while shopspring computes to the requested precision
// with math/big. So treat these as indicative ("verify the functions exist and
// roughly where they land"), not as an apples-to-apples speed claim. Results are
// written to the package-level sinks (declared in compare_int_test.go) to defeat
// dead-code elimination.
package bench

import (
	"math"
	"testing"

	ay "github.com/aytechnet/decimal"
	ss "github.com/shopspring/decimal"
)

// TestMathAccuracy verifies aytechnet's transcendental/trig results against the
// stdlib math package — a fast function is worthless if it is wrong. aytechnet is
// fixed-point ~17 digits, so we allow a tolerance a little looser than float64.
func TestMathAccuracy(t *testing.T) {
	const tol = 1e-13
	cases := []struct {
		name string
		got  float64
		want float64
	}{
		{"Ln(2.5)", ay.New(25, -1).Ln(mathPrec).InexactFloat64(), math.Log(2.5)},
		{"Sin(0.5)", ay.New(5, -1).Sin().InexactFloat64(), math.Sin(0.5)},
		{"Cos(0.5)", ay.New(5, -1).Cos().InexactFloat64(), math.Cos(0.5)},
		{"Tan(0.5)", ay.New(5, -1).Tan().InexactFloat64(), math.Tan(0.5)},
		{"Atan(0.5)", ay.New(5, -1).Atan().InexactFloat64(), math.Atan(0.5)},
		{"Sqrt(2)", ay.New(2, 0).Sqrt().InexactFloat64(), math.Sqrt(2)},
	}
	for _, c := range cases {
		if diff := math.Abs(c.got - c.want); diff > tol {
			t.Errorf("%s = %.16g, want %.16g (diff %.2e > tol %.0e)", c.name, c.got, c.want, diff, tol)
		} else {
			t.Logf("%s = %.16g (ref %.16g, diff %.2e) OK", c.name, c.got, c.want, diff)
		}
	}
}

// Precision requested from the precision-parameterised functions (Ln).
const mathPrec = 16

// =============================================================================
// Ln(2.5) — aytechnet returns Decimal, shopspring returns (Decimal, error)
// =============================================================================

func BenchmarkLnAytechnet(b *testing.B) {
	d := ay.New(25, -1) // 2.5
	var r ay.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d.Ln(mathPrec)
	}
	sinkAy = r
}

func BenchmarkLnShopspring(b *testing.B) {
	d := ss.New(25, -1) // 2.5
	var r ss.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, _ = d.Ln(mathPrec)
	}
	sinkSS = r
}

// =============================================================================
// Sin(0.5)
// =============================================================================

func BenchmarkSinAytechnet(b *testing.B) {
	d := ay.New(5, -1) // 0.5 rad
	var r ay.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d.Sin()
	}
	sinkAy = r
}

func BenchmarkSinShopspring(b *testing.B) {
	d := ss.New(5, -1)
	var r ss.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d.Sin()
	}
	sinkSS = r
}

// =============================================================================
// Cos(0.5)
// =============================================================================

func BenchmarkCosAytechnet(b *testing.B) {
	d := ay.New(5, -1)
	var r ay.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d.Cos()
	}
	sinkAy = r
}

func BenchmarkCosShopspring(b *testing.B) {
	d := ss.New(5, -1)
	var r ss.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d.Cos()
	}
	sinkSS = r
}

// =============================================================================
// Tan(0.5)
// =============================================================================

func BenchmarkTanAytechnet(b *testing.B) {
	d := ay.New(5, -1)
	var r ay.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d.Tan()
	}
	sinkAy = r
}

func BenchmarkTanShopspring(b *testing.B) {
	d := ss.New(5, -1)
	var r ss.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d.Tan()
	}
	sinkSS = r
}

// =============================================================================
// Atan(0.5) — inverse trig (the only one both packages provide)
// =============================================================================

func BenchmarkAtanAytechnet(b *testing.B) {
	d := ay.New(5, -1)
	var r ay.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d.Atan()
	}
	sinkAy = r
}

func BenchmarkAtanShopspring(b *testing.B) {
	d := ss.New(5, -1)
	var r ss.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d.Atan()
	}
	sinkSS = r
}

// =============================================================================
// Sqrt(2) — aytechnet only (shopspring v1.4.0 has no Sqrt method)
// =============================================================================

func BenchmarkSqrtAytechnet(b *testing.B) {
	d := ay.New(2, 0) // 2
	var r ay.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d.Sqrt()
	}
	sinkAy = r
}
