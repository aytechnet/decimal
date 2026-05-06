# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

- Build: `go build -v ./...`
- Run all tests with coverage: `go test -v -coverprofile=coverage.txt` (this is what CI runs; coverage output already exists as `c.out`)
- Run a single test: `go test -run TestName -v` (e.g. `go test -run TestWeightConversions -v`)
- Run a single subtest / table case: `go test -run TestName/case_name -v`
- Benchmarks: `go test -bench=. -benchmem`

The module targets `go 1.13` (`go.mod`), but CI builds with Go 1.20. There are no external runtime dependencies.

## Architecture

This package implements fixed-point decimals where the entire value (sign, loss flag, mantissa, exponent, and optional unit) is packed into a single `int64`. The fundamental design choice — and the constraint everything else follows from — is that the types are literally `type Decimal int64` and `type Weight int64`. There are no pointers, no heap allocations, and an uninitialized variable (`Null = 0`) is a meaningful, JSON-`omitempty`-friendly sentinel distinct from `Zero`.

### Bit layout (the load-bearing detail)

Every type is decoded into a **VME tuple** `(v, m, e)` and re-encoded from one:
- `v` (value): high bits — `sign` (`0x8000…`), `loss` (`0x4000…`), and for `Weight` the 4-bit unit field
- `m` (mantissa): the low bits, 57 bits for `Decimal` (`MaxInt = 0x01ffffffffffffff`), 53 bits for `Weight` (`WeightMaxInt = 0x001fff…`) — the extra 4 bits encode the unit
- `e` (exponent): a 5-bit signed exponent in range `[-16, 15]`

Special values are encoded as `m == 0` with specific `v`/`e` combinations ("NaN-boxing"): `Null`, `Zero`, `NearZero`, `NearPositiveZero`, `NearNegativeZero`, `PositiveInfinity`, `NegativeInfinity`, and a range of NaN encodings. See the comment block above `vmeNormalize` in `core.go` for the canonical table.

Negative decimals are stored as the **negation** of the unsigned bit pattern (not two's complement of the encoded form). This is why `vme()` first checks `d < 0` and negates before extracting fields, and why `vmeAsDecimal` re-applies the sign at the end. Do **not** bit-twiddle a `Decimal` directly — always go through `vme()` / `vmeAsDecimal()`.

### File layout

- `core.go` — VME-tuple primitives: `vmeNormalize`, `vmeAdd`, `vmeMul`, `vmeDivRem`, `vmeRound*`, `vmeFromBytes` (parsing), `vmetBytesTo` (formatting), unit hashing. All arithmetic for all three types funnels through here.
- `decimal.go` — the `Decimal` type: public API (`Add`, `Sub`, `Mul`, `Div`, `Round*`, comparisons, `IsZero`/`IsNull`/`IsExact`/`IsNaN`/...), constructors (`New`, `NewFromInt`, `NewFromFloat*`, `NewFromString`, `RequireFromString`), and (un)marshalers for JSON, XML/text, binary (varint-packed, 1–10 bytes), gob, and `database/sql` (`Scan`/`Value`).
- `weight.go` — the `Weight` type: same shape as `Decimal` but with a unit table (`weightUnits`) covering SI (`kg`, `t`, `g`, `mg`, `µg`, `ng`, `pg`, …) and avoirdupois/troy (`lb`, `oz`, `lb t`, `oz t`, plus aliases `mcg`, `lb av`, `oz av`). Arithmetic auto-converts to a common unit. Unit codes 10 and 11 are reserved.
- `decimal_test.go` / `weight_test.go` — unit tests (the canonical specification of edge-case behavior — start here when changing semantics).

### Invariants to preserve

- **Unique representation**: after `vmeNormalize`, a given numeric value has exactly one `int64` encoding. This is what makes `==` and `!=` valid as equality (and usable as map keys). Any new code path that produces a `Decimal`/`Weight` must go through `vmeAsDecimal`/`vmeAsWeight` (which call `vmeNormalize`), not raw bit assembly.
- **`Null` (= 0) vs `Zero` (= `math.MinInt64`)**: `Null` is "unset" and only produced by leaving a value uninitialized — no operation should ever return `Null`. `Zero` is "explicit zero". Constructors that take a literal `0` return `Zero`; arithmetic on `Null` treats it as `0` but returns `Zero`-family results. `IsExactlyZero` covers both; `IsZero` also covers `NearZero` variants.
- **`loss` bit**: set whenever precision is dropped (rounding, division with non-zero remainder, float conversion of an inexact value). Never clear it implicitly. `IsExact()` is the public predicate.
- **Operator overload trap**: because the types are `int64`, `+ - * /` compile silently but produce garbage for any non-trivial value. Use `Add`/`Sub`/`Mul`/`Div`. The exception is integer literals in `[-MaxInt, MaxInt]` for `Decimal` (or `[-WeightMaxInt, WeightMaxInt]` kg for `Weight`) — those have the same bit pattern as the encoded form and can be assigned directly (`var a Decimal = -1001`).
- **Compatibility with `shopspring/decimal`**: the public API mirrors it deliberately. When adding methods, match the shopspring signature where one exists. Methods involving `*big.Int` / `*big.Rat` are intentionally not implemented.

### Performance posture

The README's headline claim is 5×–150× faster than `shopspring/decimal` and zero allocations. Hot-path code (anything in `core.go`, plus `Add`/`Sub`/`Mul`/`Div` and the (Un)Marshal paths) must stay allocation-free. When in doubt, run `go test -bench=. -benchmem` and check `allocs/op` is 0. `BytesTo`/`BytesToFixed` exist as the alloc-free counterparts of `String`/`StringFixed` — prefer extending those when adding format variants.
