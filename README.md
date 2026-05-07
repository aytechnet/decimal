# decimal

[![ci](https://github.com/aytechnet/decimal/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/aytechnet/decimal/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/aytechnet/decimal.svg)](https://pkg.go.dev/github.com/aytechnet/decimal)
[![Go Coverage](https://img.shields.io/codecov/c/github/aytechnet/decimal/main?color=brightcolor)](https://codecov.io/gh/aytechnet/decimal)
[![Go Report Card](https://goreportcard.com/badge/github.com/aytechnet/decimal)](https://goreportcard.com/report/github.com/aytechnet/decimal)

High performance, zero-allocation, low memory usage (8 bytes), low precision (17 digits), partially compatible with int64 and [shopspring/decimal](https://github.com/shopspring/decimal)

## Features

 - the unitialized value is Null and is safe to use without initialization, intepreted as 0 which is different from 0, usefull in omitempty flag for JSON decoding/encoding.
 - **low-memory** usage as internal representation is **int64** and value between -144115188075855871 and 144115188075855871 can safely be used as Decimal.
 - **no heap allocations** to prevent garbage collector impact.
 - **high performance**, arithmetic operations are 2x to 41x faster than [shopspring/decimal](https://github.com/shopspring/decimal) (see Benchmarks below).
 - compact binary serialization from 1 to 10 bytes.
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

Methods around `math/big` (`NewFromBigInt`, `NewFromBigRat`, `BigInt`, `BigFloat`, `Rat`, `Coefficient`) are not supported by design — the whole point of the package is to avoid `big.Int` allocations.

## Benchmarks

Comparison against [shopspring/decimal](https://github.com/shopspring/decimal) v1.4.0 on a Ryzen 5 8540U (Go 1.24, two runs averaged):

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

## Why this package

This package has been created in 2022 and has been used internally for e-commerce related softwares at Aytechnet like [DyaPi](https://dyapi.io)
or [Velonomad](https://www.velonomad.com). At this time, almost only [shopspring/decimal](https://github.com/shopspring/decimal) was available.
I would like a decimal package with `omitempty` friendly interface to `encoding/json` and a small memory usage.

Since then, much more decimal package alternatives have been made available.

## License

The MIT License (MIT)

Some portion of this code inspired directly from [shopspring/decimal](https://github.com/shopspring/decimal), which is also released under the MIT Licence.
