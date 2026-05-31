// Pure-integer benchmarks: operands with no fractional part. This is the case
// where aytechnet/decimal stays pure int64 arithmetic while shopspring/decimal
// still funnels everything through math/big. Run from this directory:
//
//	go test -bench=Int -benchmem -count=6
//
// Every benchmark writes its result into a package-level sink after the loop so
// the compiler cannot eliminate the call (dead-code elimination would otherwise
// make construction benchmarks report sub-nanosecond, meaningless timings).
package bench

import (
	"testing"

	ay "github.com/aytechnet/decimal"
	ss "github.com/shopspring/decimal"
)

// Sinks: written after each loop to defeat dead-code elimination.
var (
	sinkAy ay.Decimal
	sinkSS ss.Decimal
)

// =============================================================================
// Add — small pure integers (212 + 31)
// =============================================================================

func BenchmarkIntAddAytechnet(b *testing.B) {
	d1, d2 := ay.New(212, 0), ay.New(31, 0)
	var r ay.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d1.Add(d2)
	}
	sinkAy = r
}

func BenchmarkIntAddShopspring(b *testing.B) {
	d1, d2 := ss.New(212, 0), ss.New(31, 0)
	var r ss.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d1.Add(d2)
	}
	sinkSS = r
}

// =============================================================================
// Sub — small pure integers (212 - 31)
// =============================================================================

func BenchmarkIntSubAytechnet(b *testing.B) {
	d1, d2 := ay.New(212, 0), ay.New(31, 0)
	var r ay.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d1.Sub(d2)
	}
	sinkAy = r
}

func BenchmarkIntSubShopspring(b *testing.B) {
	d1, d2 := ss.New(212, 0), ss.New(31, 0)
	var r ss.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d1.Sub(d2)
	}
	sinkSS = r
}

// =============================================================================
// Mul — small pure integers (212 * 31)
// =============================================================================

func BenchmarkIntMulAytechnet(b *testing.B) {
	d1, d2 := ay.New(212, 0), ay.New(31, 0)
	var r ay.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d1.Mul(d2)
	}
	sinkAy = r
}

func BenchmarkIntMulShopspring(b *testing.B) {
	d1, d2 := ss.New(212, 0), ss.New(31, 0)
	var r ss.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d1.Mul(d2)
	}
	sinkSS = r
}

// =============================================================================
// Div (exact) — 620 / 31 == 20
// =============================================================================

func BenchmarkIntDivAytechnet(b *testing.B) {
	d1, d2 := ay.New(620, 0), ay.New(31, 0)
	var r ay.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d1.Div(d2)
	}
	sinkAy = r
}

func BenchmarkIntDivShopspring(b *testing.B) {
	d1, d2 := ss.New(620, 0), ss.New(31, 0)
	var r ss.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d1.Div(d2)
	}
	sinkSS = r
}

// =============================================================================
// Div (inexact) — 1_000_000_000_000_000 / 7, full rounding path
// =============================================================================

func BenchmarkIntDivInexactAytechnet(b *testing.B) {
	d1, d2 := ay.New(1000000000000000, 0), ay.New(7, 0)
	var r ay.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d1.Div(d2)
	}
	sinkAy = r
}

func BenchmarkIntDivInexactShopspring(b *testing.B) {
	d1, d2 := ss.New(1000000000000000, 0), ss.New(7, 0)
	var r ss.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d1.Div(d2)
	}
	sinkSS = r
}

// =============================================================================
// Large pure integers near aytechnet's 57-bit mantissa limit
// (MaxInt = 144115188075855871). Stresses shopspring's big.Int allocations.
// =============================================================================

func BenchmarkBigIntAddAytechnet(b *testing.B) {
	d1, d2 := ay.New(100000000000000000, 0), ay.New(44115188075855871, 0)
	var r ay.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d1.Add(d2)
	}
	sinkAy = r
}

func BenchmarkBigIntAddShopspring(b *testing.B) {
	d1, d2 := ss.New(100000000000000000, 0), ss.New(44115188075855871, 0)
	var r ss.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d1.Add(d2)
	}
	sinkSS = r
}

func BenchmarkBigIntMulAytechnet(b *testing.B) {
	d1, d2 := ay.New(123456789, 0), ay.New(987654321, 0)
	var r ay.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d1.Mul(d2)
	}
	sinkAy = r
}

func BenchmarkBigIntMulShopspring(b *testing.B) {
	d1, d2 := ss.New(123456789, 0), ss.New(987654321, 0)
	var r ss.Decimal
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r = d1.Mul(d2)
	}
	sinkSS = r
}

// =============================================================================
// Construction from native integer types (sink-protected against DCE)
// =============================================================================

// The argument is derived from the loop index (int64(i)) so the compiler cannot
// constant-fold the call into a compile-time value — otherwise the loop measures
// nothing. The conversion cost is identical on both sides, so the comparison
// stays fair.

func BenchmarkNewFromIntAytechnet(b *testing.B) {
	var r ay.Decimal
	for i := 0; i < b.N; i++ {
		r = ay.NewFromInt(int64(i))
	}
	sinkAy = r
}

func BenchmarkNewFromIntShopspring(b *testing.B) {
	var r ss.Decimal
	for i := 0; i < b.N; i++ {
		r = ss.NewFromInt(int64(i))
	}
	sinkSS = r
}

func BenchmarkNewFromInt32Aytechnet(b *testing.B) {
	var r ay.Decimal
	for i := 0; i < b.N; i++ {
		r = ay.NewFromInt32(int32(i))
	}
	sinkAy = r
}

func BenchmarkNewFromInt32Shopspring(b *testing.B) {
	var r ss.Decimal
	for i := 0; i < b.N; i++ {
		r = ss.NewFromInt32(int32(i))
	}
	sinkSS = r
}

// NewFromUint64 has no shopspring counterpart in v1.4.0 — aytechnet only.
func BenchmarkNewFromUint64Aytechnet(b *testing.B) {
	var r ay.Decimal
	for i := 0; i < b.N; i++ {
		r = ay.NewFromUint64(uint64(i))
	}
	sinkAy = r
}

// =============================================================================
// Running sum — acc = 0 + 1 + 2 + 3 + ... using the loop index as the addend.
//
// This is the most honest microbenchmark shape: the accumulator is carried from
// one iteration to the next, so each Add is a genuine data dependency the
// compiler can neither hoist out of the loop nor eliminate — no sink needed.
// It also mirrors a real fiscal workload: summing a column of integer amounts.
// Each iteration measures one int->Decimal conversion plus one Add.
//
// No artificial bound is applied, but mind the operating range. Go caps b.N at
// 1e9 and ramps up to fit -benchtime; the running total is sum(0..b.N-1) ≈
// b.N²/2, so it stays exact (inside aytechnet's 57-bit mantissa, 1.44e17) only
// while b.N < ~5.4e8. Measured per-op cost:
//
//	b.N = 3e8  (sum 4.5e16, exact)            2.4 ns/op   ~21× vs shopspring
//	b.N = 5e8  (sum 1.25e17, exact)           2.4 ns/op   ~21×
//	b.N = 1e9  (sum 5e17, EXCEEDS 2^57)       6.0 ns/op   ~9×
//
// The 2.4 -> 6 ns jump is NOT magnitude: it is aytechnet switching to the
// rounding/loss code path once the accumulator can no longer represent every
// integer exactly. shopspring stays ~53 ns throughout (single-word big.Int below
// 9.2e18). The default -benchtime=1s lands around b.N≈4e8 → still exact → ~21×.
// For a stable headline, keep b.N below ~5e8 (e.g. -benchtime=200ms or shorter)
// or pin it explicitly (-benchtime=300000000x). Running past 2^57 measures the
// inexact regime — valid timing, but a different operation.
// =============================================================================

func BenchmarkSumIntAytechnet(b *testing.B) {
	acc := ay.Zero
	for i := 0; i < b.N; i++ {
		acc = acc.Add(ay.NewFromInt(int64(i)))
	}
	sinkAy = acc
}

func BenchmarkSumIntShopspring(b *testing.B) {
	acc := ss.Zero
	for i := 0; i < b.N; i++ {
		acc = acc.Add(ss.NewFromInt(int64(i)))
	}
	sinkSS = acc
}
