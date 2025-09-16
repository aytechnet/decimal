# decimal

[![ci](https://github.com/aytechnet/decimal/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/aytechnet/decimal/actions/workflows/go.yml)
[![Go Doc](https://godoc.org/github.com/aytechnet/decimal?status.svg)](https://godoc.org/github.com/aytechnet/decimal) 
[![Go Coverage](https://img.shields.io/codecov/c/github/aytechnet/decimal/main?color=brightcolor)](https://codecov.io/gh/aytechnet/decimal)
[![Go Report Card](https://goreportcard.com/badge/github.com/aytechnet/decimal)](https://goreportcard.com/report/github.com/aytechnet/decimal)

High performance, zero-allocation, low precision (17 digits), partially compatible with int64 and [shopspring/decimal](https://github.com/shopspring/decimal)

## Features

 - the unitialized value is Null and is safe to use without initialization, intepreted as 0 which is different from 0, usefull in omitempty flag for JSON decoding/encoding.
 - **low-memory** usage as internal representation is **int64** and value between -144115188075855871 and 144115188075855871 can safely be used as Decimal.
 - **no heap allocations** to prevent garbage collector impact.
 - **high performance**, arithmetic operations are 5x to 150x faster than [shopspring/decimal](https://github.com/shopspring/decimal) package.
 - compact binary serialization from 1 to 10 bytes.
 - 57 bits mantissa to limit rounding errors compared to float64 (50 bits mantissa) for common operation like additions, multiplications, divisions.
 - **loss** flag available so if a rounding error occurs information is not lost.
 - support infinity and NaN decimal as well as near zero value, near negative zero and near positive zero (loss bit always set in such cases).
 - since **int64** is used internally, Decimal are **immutable** as no internal pointer are used.
 - **unique representation** for a given decimal, suitable for use as a key in hash table or by using == or != operator directly.
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

## Why this package

This package has been created in 2022 and has been used internally for e-commerce related softwares at Aytechnet like [DyaPi](https://dyapi.io)
or [Velonomad](https://www.velonomad.com). At this time, almost only [shopspring/decimal](https://github.com/shopspring/decimal) was available.
I would like a decimal package with `omitempty` friendly interface to `encoding/json` and a small memory usage.

Since then, much more decimal package alternatives have been made available.

## License

The MIT License (MIT)

Some portion of this code inspired directly from [shopspring/decimal](https://github.com/shopspring/decimal), which is also released under the MIT Licence.
