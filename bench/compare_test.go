// Package bench compares aytechnet/decimal against shopspring/decimal on representative arithmetic
// and conversion benchmarks. Run from this directory:
//
//	go test -bench=. -benchmem
package bench

import (
	"testing"

	ay "github.com/aytechnet/decimal"
	ss "github.com/shopspring/decimal"
)

// =============================================================================
// Add
// =============================================================================

func BenchmarkAddAytechnet(b *testing.B) {
	d1 := ay.New(212, -2)
	d2 := ay.New(31, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Add(d2)
	}
}

func BenchmarkAddShopspring(b *testing.B) {
	d1 := ss.New(212, -2)
	d2 := ss.New(31, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Add(d2)
	}
}

// =============================================================================
// Mul
// =============================================================================

func BenchmarkMulAytechnet(b *testing.B) {
	d1 := ay.New(212, -2)
	d2 := ay.New(31, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Mul(d2)
	}
}

func BenchmarkMulShopspring(b *testing.B) {
	d1 := ss.New(212, -2)
	d2 := ss.New(31, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Mul(d2)
	}
}

// =============================================================================
// Div (inexact division)
// =============================================================================

func BenchmarkDivAytechnet(b *testing.B) {
	d1 := ay.New(212, -2)
	d2 := ay.New(31, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Div(d2)
	}
}

func BenchmarkDivShopspring(b *testing.B) {
	d1 := ss.New(212, -2)
	d2 := ss.New(31, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Div(d2)
	}
}

// =============================================================================
// NewFromString
// =============================================================================

func BenchmarkNewFromStringAytechnet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ay.NewFromString("100020003000400050e-17")
	}
}

func BenchmarkNewFromStringShopspring(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ss.NewFromString("100020003000400050e-17")
	}
}

// =============================================================================
// String
// =============================================================================

func BenchmarkStringAytechnet(b *testing.B) {
	d := ay.NewFromFloat(1.000020003000400050)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.String()
	}
}

func BenchmarkStringShopspring(b *testing.B) {
	d := ss.NewFromFloat(1.000020003000400050)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.String()
	}
}

// =============================================================================
// NewFromFloat
// =============================================================================

func BenchmarkNewFromFloatAytechnet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ay.NewFromFloat(123.456789)
	}
}

func BenchmarkNewFromFloatShopspring(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ss.NewFromFloat(123.456789)
	}
}

// =============================================================================
// Pow
// =============================================================================

func BenchmarkPow60Aytechnet(b *testing.B) {
	d1 := ay.New(11, -1)
	d2 := ay.New(60, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Pow(d2)
	}
}

func BenchmarkPow60Shopspring(b *testing.B) {
	d1 := ss.New(11, -1)
	d2 := ss.New(60, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d1.Pow(d2)
	}
}
