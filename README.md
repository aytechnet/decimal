# decimal

[![ci](https://github.com/aytechnet/decimal/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/aytechnet/decimal/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/aytechnet/decimal.svg)](https://pkg.go.dev/github.com/aytechnet/decimal)
[![Go Coverage](https://img.shields.io/codecov/c/github/aytechnet/decimal/main?color=brightcolor)](https://codecov.io/gh/aytechnet/decimal)
[![lint](https://github.com/aytechnet/decimal/actions/workflows/lint.yml/badge.svg?branch=main)](https://github.com/aytechnet/decimal/actions/workflows/lint.yml)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge-flat.svg)](https://github.com/avelino/awesome-go)

High performance, zero-allocation, low memory usage (8 bytes), low precision (17 digits), partially compatible with int64 and [shopspring/decimal](https://github.com/shopspring/decimal)

## Features

 - the unitialized value is Null and is safe to use without initialization, intepreted as 0 which is different from 0, usefull in omitempty flag for JSON decoding/encoding.
 - **low-memory** usage as internal representation is **int64** and value between -144115188075855871 and 144115188075855871 can safely be used as Decimal.
 - **no heap allocations** to prevent garbage collector impact.
 - **high performance**, fractional arithmetic is 4× to 19× faster than [shopspring/decimal](https://github.com/shopspring/decimal) v1.6, pure-integer arithmetic 11× to 26×, trigonometric functions ~40× to ~95×, `Ln` ~45× and `Sqrt` ~15× — all allocation-free, where shopspring allocates on every operation (see Benchmarks below).
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

The public API mirrors [shopspring/decimal](https://github.com/shopspring/decimal). Methods added for compatibility include `DivRound`, `PowInt32`, `Shift`, `Truncate`, `RoundUp`, `RoundDown`, `RoundCash`, `StringFixedCash`, `NumDigits`, `Copy`, `Normalize` (a no-op: the representation is already unique) and `NewFromFormattedString`. `Decimal` implements `fmt.Formatter` like shopspring v1.6: `%.2f` rounds like `StringFixed`, with width and the `-`, `+`, ` ` and `0` flags, `%d` prints integer decimals and `%#v` prints Go syntax such as `decimal.New(-314, -2)`. JSON output is **unquoted** by default (raw number) — incompatible with shopspring's quoted-string default; route values through `MarshalText` / `UnmarshalText` if you need cross-package interop.

**Rounding** matches shopspring on every entry point (`Round`, `StringFixed`, `DivRound`, `RoundCash`, `RoundBank`, `RoundUp`/`Down`/`Ceil`/`Floor`, `Truncate`), checked value by value against shopspring v1.4.0 and v1.6.0 by `bench/compare_round_test.go`: a half rounds away from zero, symmetrically, so `-2.5` rounds to `-3` and a credit note rounds like its invoice. Up to v1.0.2 a negative half rounded toward +∞ (`-2.5` → `-2`). Since v1.1.0, `Truncate` with a negative precision truncates the integer part like shopspring v1.5+ (`5432` → `5400` at `-2`; shopspring v1.4 and aytechnet v1.0.2 returned the value unchanged).

`NewFromFloat` and `NewFromFloat32` follow shopspring since v1.1.0: they convert the **shortest decimal** that gives the same float back (`strconv.FormatFloat(f, 'g', -1, 64)`), so `NewFromFloat(-178464.49)` is exactly `-178464.49`. Up to v1.0.2 they converted the binary value with a precision bug from the 16th digit (`~-178464.49000000009`, 97 % of 2-3 decimal amounts differed). Checked against shopspring on 1,000,000 random floats by `bench/compare_round_test.go`; a value with more than 16 fractional digits is rounded to the 16 a `Decimal` holds (loss bit set). `NewFromFloat64Exact` still converts the binary value (correctly rounded to the mantissa since v1.1.0, `~-178464.48999999999`): the math functions use it.

`NewFromFloatWithExponent(value, exp)` follows shopspring since v1.1.0: `exp` is an **exponent** (`-2` keeps 2 decimal places; up to v1.0.2 it was a number of places, `2`), and the exact binary value of the float is rounded half away from zero, so `NewFromFloatWithExponent(0.015, -2)` is `0.01` (0.015 is 0.01499999… as a `float64`). It is checked against shopspring v1.4.0 and v1.6.0 on 200,000 random floats by `bench/compare_round_test.go`, to within one unit of the 17th significant digit. Unlike shopspring it returns `NaN`/`±Inf` instead of panicking.

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

### `Sqrt`, `SqrtRound`, `Root` and `RootRound` signatures are intentionally NOT compatible

shopspring v1.6.0 added `Sqrt() (Decimal, error)`, `SqrtRound(places int32) (Decimal, error)`,
`Root(n int64) (Decimal, error)` and `RootRound(n int64, places int32) (Decimal, error)`. This package
keeps its existing `Sqrt() Decimal`, on purpose and for the same reason as `Ln`: changing it would break
every caller of the v1 API, and a negative input already yields `NaN` instead of an error, so the call
composes in an expression. `SqrtRound(places int32) Decimal`, `Root(n int64) Decimal` and
`RootRound(n int64, places int32) Decimal` follow the same rule: no error, `NaN` for `n < 1` or an even
root of a negative number. They are computed from `float64` (~16 significant digits, exact integer roots
such as `Root(32, 5) = 2` stay exact), where shopspring computes `DivisionPrecision` places exactly with
`big.Int`. Code written for shopspring's versions therefore does not compile against this package as is;
drop the error, or wrap the call as shown for `Ln` above.

Methods around `math/big` (`NewFromBigInt`, `NewFromBigRat`, `BigInt`, `BigFloat`, `Rat`, `Coefficient`) are not supported by design — the whole point of the package is to avoid `big.Int` allocations.

## Benchmarks

All figures: vs [shopspring/decimal](https://github.com/shopspring/decimal) v1.6.0, Ryzen 5 8540U, Go 1.26, `-count=6 -benchtime=100ms` averaged. aytechnet is **allocation-free on every operation below**. shopspring v1.5/v1.6 roughly halved its own costs (fewer `big.Int` allocations, faster `Ln`, `NewFromFloat` and trig), so the gaps are narrower than with v1.4.0; aytechnet's own timings are unchanged.

### Synthesis (speedup ranges by category)

| Category | Speedup range | aytechnet allocs | Notes |
|---|---|---|---:|
| Fractional arithmetic (`Add`/`Mul`/`Div`/`Pow`) | **4×–19×** | 0 | `Div` 19×, `Mul` 4× |
| Conversion (`NewFromString`/`Float`/`String`) | **1.3×–2×** | 0–1 | shortest-decimal `NewFromFloat` 1.3× |
| Pure-integer arithmetic | **11×–26×** | 0 | shopspring allocates `big.Int` |
| Integer construction (`NewFromInt`/`Int32`) | **17×–~50×** | 0 | `Int32` near measurement floor |
| Running sum `Σ i` (convert + add) | **~19×** | 0 | true data dependency, most defensible |
| Transcendental / trig (`Sin`/`Cos`/`Tan`/`Atan`/`Ln`) | **~40×–~95×** | 0 | **NOT iso-precision** — see caveat below |
| `Sqrt` / `Root` | **~15× / ~31×** | 0 | close to iso-precision (16 places both) |

Detailed per-operation tables follow.

Comparison against [shopspring/decimal](https://github.com/shopspring/decimal) v1.6.0 on a Ryzen 5 8540U (Go 1.26, 6 runs averaged):

| Operation | aytechnet | shopspring | Speedup | aytechnet allocs | shopspring allocs |
|---|---:|---:|---:|---:|---:|
| `Add` | 9.4 ns/op | 57 ns/op | **6×** | 0 | 2 (80 B) |
| `Mul` | 10.2 ns/op | 44 ns/op | **4×** | 0 | 2 (80 B) |
| `Div` | 8.3 ns/op | 161 ns/op | **19×** | 0 | 8 (232 B) |
| `Pow(1.1, 60)` | 39 ns/op | 217 ns/op | **6×** | 0 | 7 (224 B) |
| `NewFromString` | 37 ns/op | 81 ns/op | **2×** | 0 | 2 (40 B) |
| `NewFromFloat` | 52 ns/op | 65 ns/op | **1.3×** | 0 | 2 (40 B) |
| `String` | 54 ns/op | 106 ns/op | **2×** | 1 (24 B) | 2 (32 B) |

Reproduce: `cd bench && go test -bench=. -benchmem`. The `bench/` sub-module has its own `go.mod` so the main package keeps zero external dependencies.

### Pure integers

On operands with no fractional part — counts, quantities, integer money in minor units — `aytechnet/decimal` stays pure `int64` arithmetic with **zero allocations**, while `shopspring/decimal` still goes through `math/big`. Ryzen 5 8540U, Go 1.26, 6 runs averaged (every benchmark writes its result to a sink and derives constructor inputs from the loop index, so the compiler cannot eliminate or constant-fold the call):

| Operation | aytechnet | shopspring | Speedup | aytechnet allocs | shopspring allocs |
|---|---:|---:|---:|---:|---:|
| `Add` (int) | 1.7 ns/op | 44 ns/op | **26×** | 0 | 2 (80 B) |
| `Sub` (int) | 1.7 ns/op | 35 ns/op | **21×** | 0 | 2 (40 B) |
| `Mul` (int) | 2.2 ns/op | 47 ns/op | **21×** | 0 | 2 (80 B) |
| `Div` (exact, 620/31) | 7.5 ns/op | 93 ns/op | **12×** | 0 | 4 (120 B) |
| `Div` (inexact, 1e15/7) | 14 ns/op | 149 ns/op | **11×** | 0 | 7 (224 B) |
| `NewFromInt` | 1.3 ns/op | 23 ns/op | **17×** | 0 | 2 (40 B) |
| `NewFromInt32` | 0.4 ns/op | 22 ns/op | **~50×** † | 0 | 2 (40 B) |

† `NewFromInt32` inlines to little more than a sign-extension, because an in-range integer *is* the internal encoding (`type Decimal int64`); at ~1 CPU cycle this figure sits near the measurement floor and is inlining-sensitive — the arithmetic core's honest range is **~11–26×** (division ~11–12×, add/sub/mul ~21–26×).

The most representative figure is a **running sum** (`acc = 0 + 1 + 2 + …`, one int→Decimal conversion plus one `Add` per term, accumulator carried across iterations so the work is a genuine data dependency — no compiler trickery possible): **2.4 ns/term, 0 allocations** vs **46 ns/term, 2 allocations** for shopspring (**~19×**). This mirrors summing a column of integer amounts on an invoice. Note: the running total stays exact only while it fits aytechnet's 57-bit mantissa (`sum ≈ b.N²/2 < 1.44e17`, i.e. `b.N < ~5.4e8`, which covers the default `-benchtime=1s`); past that the per-term cost rises to ~6 ns as adds take the rounding path, so pin a short `-benchtime` (or `-benchtime=300000000x`) for the headline figure.

Reproduce: `cd bench && go test -bench='Int|Sum' -benchmem -count=6 -benchtime=200ms`.

### Transcendental / trigonometric

`Ln`, `Sin`, `Cos`, `Tan`, `Atan` and `Sqrt` (added to shopspring in v1.6.0) are iterative
functions, not hot-path. These figures are **indicative, not iso-precision**:
aytechnet computes to its ~17-digit `int64` mantissa, while shopspring computes to a
higher internal precision with `math/big` — so part of the gap is aytechnet doing
less work. aytechnet stays allocation-free throughout; correctness is checked against
the stdlib `math` package in `TestMathAccuracy` (all within 1e-13).

| Operation | aytechnet | shopspring | Speedup | aytechnet allocs | shopspring allocs |
|---|---:|---:|---:|---:|---:|
| `Sin(0.5)` | 38 ns/op | 3382 ns/op | **~88×** | 0 | 71 (4920 B) |
| `Cos(0.5)` | 40 ns/op | 3760 ns/op | **~94×** | 0 | 77 (5448 B) |
| `Tan(0.5)` | 38 ns/op | 2731 ns/op | **~71×** | 0 | 62 (3954 B) |
| `Atan(0.5)` | 34 ns/op | 1469 ns/op | **~43×** | 0 | 51 (2136 B) |
| `Ln(2.5)` (precision 16) | 57 ns/op | 2541 ns/op | **~45×** | 0 | 51 (2288 B) |
| `Sqrt(2)` | 23 ns/op | 336 ns/op | **~15×** | 0 | 11 (248 B) |
| `Root(2, 3)` | 51 ns/op | 1567 ns/op | **~31×** | 0 | 39 (1296 B) |

These large multipliers are dominated by the precision mismatch: aytechnet's trig are
backed by hardware `float64` (≈15–16 significant digits, the most this 17-digit type can
hold) and round once, whereas shopspring computes to arbitrary precision with `math/big`
(e.g. `Ln` does 51 allocations / 2.3 kB). The honest reading is *"if 16 digits is enough,
aytechnet gives it allocation-free for one to two orders of magnitude less work"* — not that
the algorithms are intrinsically 40×–95× better.

`Ln` (precision >= 16) is special: it does **not** use `float64`/`math.Log`. It reduces the
value in binary to `a·2^K`, `a ∈ [1,2)`, and sums the `atanh` series
`ln(a) = 2(s + s³/3 + …)` in a 2^63-scaled fixed point (the full 64-bit mantissa). This
fills the type's 57-bit precision — verified against an independent 240-bit `big.Float`
reference (`TestLnHighPrecision`) to < 1e-16 relative, typically ~1e-17, i.e. ~10× tighter
than the float64 path. The right way to read its speed: getting that 17th correct digit any
other way means an arbitrary-precision library (`math/big`), which is ~2.5 µs here with
shopspring v1.6 — so this path is **both more precise and ~45× faster than the only
alternative that matches its precision**. The ~4 ns it costs over aytechnet's own float64 path is noise by comparison, and
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
