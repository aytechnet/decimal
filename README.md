# decimal

[![ci](https://github.com/aytechnet/decimal/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/aytechnet/decimal/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/aytechnet/decimal.svg)](https://pkg.go.dev/github.com/aytechnet/decimal)
[![Go Coverage](https://img.shields.io/codecov/c/github/aytechnet/decimal/main?color=brightcolor)](https://codecov.io/gh/aytechnet/decimal)
[![Go Report Card](https://goreportcard.com/badge/github.com/aytechnet/decimal)](https://goreportcard.com/report/github.com/aytechnet/decimal)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge-flat.svg)](https://github.com/avelino/awesome-go)

High performance, zero-allocation, low memory usage (8 bytes), low precision (17 digits), partially compatible with int64 and [shopspring/decimal](https://github.com/shopspring/decimal)

## Features

 - the unitialized value is Null and is safe to use without initialization, intepreted as 0 which is different from 0, usefull in omitempty flag for JSON decoding/encoding.
 - **low-memory** usage as internal representation is **int64** and value between -144115188075855871 and 144115188075855871 can safely be used as Decimal.
 - **no heap allocations** to prevent garbage collector impact.
 - **high performance**, fractional arithmetic is 5× to 40× faster than [shopspring/decimal](https://github.com/shopspring/decimal), pure-integer arithmetic 18× to 35×, trigonometric functions ~200×, and `Ln` up to ~1 200× — all allocation-free (see Benchmarks below).
 - compact binary serialization from 1 to 10 bytes — **open, documented, interoperable** format (see [BINARY_FORMAT.md](BINARY_FORMAT.md); same format covers `Weight` and `Length` with explicit unit support).
 - 57 bits mantissa to limit rounding errors compared to float64 (50 bits mantissa) for common operation like additions, multiplications, divisions.
 - **loss** flag available so if a rounding error occurs information is not lost.
 - support infinity and NaN decimal as well as near zero value, near negative zero and near positive zero (loss bit always set in such cases).
 - since **int64** is used internally, Decimal are **immutable** as no internal pointer is used.
 - **unique representation** for a given decimal, suitable for use as a key in hash table or by using == or != operator directly.
 - support Weight and Length decimal using 53 bits mantissa and 4 bits of type unit.
 - **JSON, XML** - compatible with [encoding/json] and [encoding/xml].
 - compatible with [shopspring/decimal](https://github.com/shopspring/decimal) except for BigInt and BigRat methods not supported.

## Install

Run `go get github.com/aytechnet/decimal`

## Requirements

Decimal library requires Go version `>=1.13`.

## Documentation

http://godoc.org/github.com/aytechnet/decimal


## Usage

Usage taken from [shopspring/decimal](https://github.com/shopspring/decimal) but updated for this package and constant compatibility with int64 :

```go
package main

import (
	"fmt"
	"github.com/aytechnet/decimal"
)

func main() {
	price, err := decimal.NewFromString("136.02")
	if err != nil {
		panic(err)
	}

	var quantity decimal.Decimal = 3

	fee := decimal.NewFromFloat(.035)
	taxRate, _ := decimal.NewFromString(".08875")

	subtotal := price.Mul(quantity)

	preTax := subtotal.Mul(fee.Add(1))

	total := preTax.Mul(taxRate.Add(1))

	fmt.Println("Subtotal:", subtotal)                      // Subtotal: 408.06
	fmt.Println("Pre-tax:", preTax)                         // Pre-tax: 422.3421
	fmt.Println("Taxes:", total.Sub(preTax))                // Taxes: 37.482861375
	fmt.Println("Total:", total)                            // Total: 459.824961375
	fmt.Println("Tax rate:", total.Sub(preTax).Div(preTax)) // Tax rate: 0.08875
}
```

## Weight and Length

`Weight` and `Length` are companion fixed-point types with the same 8-byte layout as `Decimal` but with 4 bits reserved for a unit code (53-bit mantissa instead of 57).

```go
w1, _ := decimal.NewWeightFromString("123.45kg")
w2, _ := decimal.NewWeightFromString("550g")
fmt.Println(w1.Add(w2)) // 124kg
fmt.Println(w2.Add(w1)) // 124000g — w2 unit (g) is preserved
```

`Weight` units: SI multiples of `kg` (`t`, `kt`, `Mt`, `Gt`, `g`, `mg`, `µg`, `ng`, `pg`) plus avoirdupois and troy (`lb`, `oz`, `lb t`, `oz t`, with `mcg`/`lb av`/`oz av` aliases).

```go
l1, _ := decimal.NewLengthFromString("1ft")
l2, _ := decimal.NewLengthFromString("12in")
fmt.Println(l1.Add(l2)) // 2ft (NIST 1959: 1 ft = 12 in exact)
fmt.Println(decimal.NewLengthFromString("1au")) // 149597870700m (UAI 2012)
```

`Length` units: `m`, `km`, `dm`, `cm`, `mm`, `µm` (alias `um`), `nm`, `pm`, `au` (alias `ua`), `in`, `ft`, `yd`, `mi`.

## shopspring/decimal compatibility

The public API mirrors [shopspring/decimal](https://github.com/shopspring/decimal). Methods added for compatibility include `DivRound`, `PowInt32`, `Shift`, `Truncate`, `RoundUp`, `RoundDown`, `RoundCash`, `StringFixedCash`, `NumDigits`, `Copy`, and `NewFromFormattedString`. JSON output is **unquoted** by default (raw number) — incompatible with shopspring's quoted-string default; route values through `MarshalText` / `UnmarshalText` if you need cross-package interop.

### `Ln` signature is intentionally NOT compatible

shopspring returns `Ln(precision int32) (Decimal, error)`; this package returns
`Ln(precision int32) Decimal` — **no error**. This is a deliberate design choice, not an
oversight. Because the type natively supports `NaN` and `±Inf` (every special value is a
real, propagating member of the set), an out-of-domain input simply yields a `NaN`/`-Inf`
result instead of an error:

```go
// aytechnet: chains naturally, NaN propagates if d <= 0
result := d.Ln(16).Add(k).Mul(rate)

// shopspring: the (Decimal, error) return breaks the expression
l, err := d.Ln(16)
if err != nil { /* handle */ }
result := l.Add(k).Mul(rate)
```

The same rationale applies to the other float-backed transcendental/trig methods (`Sqrt`,
`Sin`, `Cos`, `Tan`, `Atan`, `Pow`): they never return an error, so they compose inside a
single expression. Check `IsNaN()` / `IsInfinite()` on the final value when the domain is
uncertain, exactly as you would inspect a `float64` result. If you need shopspring's
error-returning shape, wrap the call: `func ln(d Decimal) (Decimal, error) { r := d.Ln(16); if r.IsNaN() { return r, errLn }; return r, nil }`.

Methods around `math/big` (`NewFromBigInt`, `NewFromBigRat`, `BigInt`, `BigFloat`, `Rat`, `Coefficient`) are not supported by design — the whole point of the package is to avoid `big.Int` allocations.

## Benchmarks

All figures: vs [shopspring/decimal](https://github.com/shopspring/decimal) v1.4.0, Ryzen 5 8540U, Go 1.26, `-count=6 -benchtime=100ms` averaged. aytechnet is **allocation-free on every operation below**.

### Synthesis (speedup ranges by category)

| Category | Speedup range | aytechnet allocs | Notes |
|---|---|---|---:|
| Fractional arithmetic (`Add`/`Mul`/`Div`/`Pow`) | **5×–40×** | 0 | `Div` 38×, `Mul` 5× |
| Conversion (`NewFromString`/`Float`/`String`) | **2×–21×** | 0–1 | string paths ~2× |
| Pure-integer arithmetic | **18×–35×** | 0 | shopspring allocates `big.Int` |
| Integer construction (`NewFromInt`/`Int32`) | **23×–~108×** | 0 | `Int32` near measurement floor |
| Running sum `Σ i` (convert + add) | **~20×** | 0 | true data dependency, most defensible |
| Transcendental / trig (`Sin`/`Cos`/`Tan`/`Atan`/`Ln`) | **~200×–~1300×** | 0 | **NOT iso-precision** — see caveat below |

Detailed per-operation tables follow.

Comparison against [shopspring/decimal](https://github.com/shopspring/decimal) v1.4.0 on a Ryzen 5 8540U (Go 1.26, 6 runs averaged):

| Operation | aytechnet | shopspring | Speedup | aytechnet allocs | shopspring allocs |
|---|---:|---:|---:|---:|---:|
| `Add` | 9.1 ns/op | 212 ns/op | **23×** | 0 | 8 (272 B) |
| `Mul` | 10.2 ns/op | 53 ns/op | **5×** | 0 | 2 (80 B) |
| `Div` | 8.1 ns/op | 332 ns/op | **41×** | 0 | 12 (328 B) |
| `Pow(1.1, 60)` | 38 ns/op | 702 ns/op | **18×** | 0 | 26 (912 B) |
| `NewFromString` | 37 ns/op | 92 ns/op | **2×** | 0 | 2 (40 B) |
| `NewFromFloat` | 16 ns/op | 318 ns/op | **20×** | 0 | 2 (40 B) |
| `String` | 59 ns/op | 126 ns/op | **2×** | 1 (24 B) | 4 (56 B) |

Reproduce: `cd bench && go test -bench=. -benchmem`. The `bench/` sub-module has its own `go.mod` so the main package keeps zero external dependencies.

### Pure integers

On operands with no fractional part — counts, quantities, integer money in minor units — `aytechnet/decimal` stays pure `int64` arithmetic with **zero allocations**, while `shopspring/decimal` still goes through `math/big`. Ryzen 5 8540U, Go 1.26, 6 runs averaged (every benchmark writes its result to a sink and derives constructor inputs from the loop index, so the compiler cannot eliminate or constant-fold the call):

| Operation | aytechnet | shopspring | Speedup | aytechnet allocs | shopspring allocs |
|---|---:|---:|---:|---:|---:|
| `Add` (int) | 1.7 ns/op | 53 ns/op | **32×** | 0 | 2 (80 B) |
| `Sub` (int) | 1.7 ns/op | 43 ns/op | **25×** | 0 | 2 (80 B) |
| `Mul` (int) | 2.4 ns/op | 53 ns/op | **22×** | 0 | 2 (80 B) |
| `Div` (exact, 620/31) | 7.5 ns/op | 203 ns/op | **27×** | 0 | 7 (184 B) |
| `Div` (inexact, 1e15/7) | 14 ns/op | 270 ns/op | **19×** | 0 | 10 (288 B) |
| `NewFromInt` | 1.4 ns/op | 31 ns/op | **22×** | 0 | 2 (40 B) |
| `NewFromInt32` | 0.3 ns/op | 31 ns/op | **~100×** † | 0 | 2 (40 B) |

† `NewFromInt32` inlines to little more than a sign-extension, because an in-range integer *is* the internal encoding (`type Decimal int64`); at ~1 CPU cycle this figure sits near the measurement floor and is inlining-sensitive — the arithmetic core's honest range is **~20–32×**.

The most representative figure is a **running sum** (`acc = 0 + 1 + 2 + …`, one int→Decimal conversion plus one `Add` per term, accumulator carried across iterations so the work is a genuine data dependency — no compiler trickery possible): **2.4 ns/term, 0 allocations** vs **53 ns/term, 2 allocations** for shopspring (**~21×**). This mirrors summing a column of integer amounts on an invoice. Note: the running total stays exact only while it fits aytechnet's 57-bit mantissa (`sum ≈ b.N²/2 < 1.44e17`, i.e. `b.N < ~5.4e8`, which covers the default `-benchtime=1s`); past that the per-term cost rises to ~6 ns as adds take the rounding path, so pin a short `-benchtime` (or `-benchtime=300000000x`) for the headline figure.

Reproduce: `cd bench && go test -bench='Int|Sum' -benchmem -count=6 -benchtime=200ms`.

### Transcendental / trigonometric

`Ln`, `Sin`, `Cos`, `Tan`, `Atan` (and `Sqrt`, which shopspring lacks) are iterative
functions, not hot-path. These figures are **indicative, not iso-precision**:
aytechnet computes to its ~17-digit `int64` mantissa, while shopspring computes to a
higher internal precision with `math/big` — so part of the gap is aytechnet doing
less work. aytechnet stays allocation-free throughout; correctness is checked against
the stdlib `math` package in `TestMathAccuracy` (all within 1e-13).

| Operation | aytechnet | shopspring | Speedup | aytechnet allocs | shopspring allocs |
|---|---:|---:|---:|---:|---:|
| `Sin(0.5)` | 35 ns/op | 7199 ns/op | **~205×** | 0 | 124 (6979 B) |
| `Cos(0.5)` | 37 ns/op | 7727 ns/op | **~208×** | 0 | 137 (7756 B) |
| `Tan(0.5)` | 37 ns/op | 7734 ns/op | **~207×** | 0 | 137 (6979 B) |
| `Atan(0.5)` | 33 ns/op | 9552 ns/op | **~290×** | 0 | 151 (4954 B) |
| `Ln(2.5)` (precision 16) | 55 ns/op | 66500 ns/op | **~1200×** | 0 | 1313 (82 kB) |
| `Sqrt(2)` | 23 ns/op | *n/a* | — | 0 | — |

These large multipliers are dominated by the precision mismatch: aytechnet's trig are
backed by hardware `float64` (≈15–16 significant digits, the most this 17-digit type can
hold) and round once, whereas shopspring computes to arbitrary precision with `math/big`
(e.g. `Ln` does 1313 allocations / 82 kB). The honest reading is *"if 16 digits is enough,
aytechnet gives it allocation-free for two orders of magnitude less work"* — not that the
algorithms are intrinsically 200×–1000× better.

`Ln` (precision >= 16) is special: it does **not** use `float64`/`math.Log`. It reduces the
value in binary to `a·2^K`, `a ∈ [1,2)`, and sums the `atanh` series
`ln(a) = 2(s + s³/3 + …)` in a 2^63-scaled fixed point (the full 64-bit mantissa). This
fills the type's 57-bit precision — verified against an independent 240-bit `big.Float`
reference (`TestLnHighPrecision`) to < 1e-16 relative, typically ~1e-17, i.e. ~10× tighter
than the float64 path. The right way to read its speed: getting that 17th correct digit any
other way means an arbitrary-precision library (`math/big`), which is ~66 µs here — so this
path is **both more precise and ~1200× faster than the only alternative that matches its
precision**. The ~4 ns it costs over aytechnet's own float64 path is noise by comparison, and
it means you never have to recompute in a slower big-number library for the extra digit.
Values whose result `|ln|` is tiny (`x` within ~1 % of 1) fall back to the float64 path:
there the 2^63 fixed-point scale loses relative precision and `math.Log` is more accurate.
Any `precision < 16` call also takes the plain float64 path. (A dedicated 128-bit near-1
variant was prototyped to cover that band at full precision but did not reach the required
accuracy in testing, so it was dropped in favour of the float64 fallback.)

Reproduce: `cd bench && go test -bench='Ln|Sin|Cos|Tan|Atan|Sqrt' -benchmem -count=6 -benchtime=100ms`.

## Why this package

This package has been created in 2022 and has been used internally for e-commerce related softwares at Aytechnet like [DyaPi](https://dyapi.io)
or [Velonomad](https://www.velonomad.com). At this time, almost only [shopspring/decimal](https://github.com/shopspring/decimal) was available.
I would like a decimal package with `omitempty` friendly interface to `encoding/json` and a small memory usage.

Since then, much more decimal package alternatives have been made available.

## License

The MIT License (MIT)

Some portion of this code inspired directly from [shopspring/decimal](https://github.com/shopspring/decimal), which is also released under the MIT Licence.
