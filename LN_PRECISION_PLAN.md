# Plan: extend `Ln()` precision to the full 57-bit mantissa (when `precision >= 16`)

> **STATUS: IMPLEMENTED (2026-05-30).** See `ln_highprec.go` and `ln_highprec_test.go`.
> Outcome vs this plan:
> - Adopted the **binary** reduction `value = a·2^K` (NOT the decimal-exponent split
>   `e·ln10 + ln(m)` originally sketched in step 1) — the decimal split caused catastrophic
>   cancellation for values near 1 (`ln(9999) − 4·ln10`). The binary path avoids it.
> - The series runs in a **2^63-scaled fixed point (full 64-bit mantissa)**, per the
>   "use v,m,e directly / full 64-bit mantissa" instruction — not a 10^18 scale.
> - **No `math.Log`/`math.Exp` seed at all** — the empirical probe confirmed a float-seeded
>   single correction collapses to 0 (float64 contaminates the residual). The from-scratch
>   atanh series sidesteps this.
> - **Honest result:** precision reaches the type's own ~1e-17 resolution (typically ~10×
>   tighter than float64), verified against an independent 240-bit `big.Float`. The *gain
>   over float64 is modest in absolute digits* (the type has only ~1.2 digits of headroom,
>   exactly the risk flagged below) but real and measured.
> - **Speed — the right baseline.** A clean same-input benchmark (precision 18 vs precision
>   10) gives ~55 ns (high-prec) vs ~51 ns (float64): the extra precision costs ~4 ns. An
>   earlier "~8.6 ns / 7500×" claim was a measurement artifact and has been retracted.
>   But comparing the two aytechnet paths is the wrong yardstick: the only *other* way to get
>   this 17th correct digit is an arbitrary-precision library (`math/big`, ~66 µs), so the
>   high-prec path is **both more precise and ~1200× faster than the precision-matching
>   alternative**. The ~4 ns over the float64 path is noise, and it means callers never fall
>   back to a slow big-number lib for the extra digit. Net: precision is effectively free.
> - Values whose result `|ln|` is tiny (`x` within ~1 % of 1) **fall back to float64**
>   (the 2^63 fixed-point scale loses relative precision there); `precision < 16` keeps the
>   byte-identical float64 path.
> - **Near-1 128-bit variant — attempted and dropped.** A dedicated `lnNear1` (scale 2^126,
>   `s = (x-1)/(x+1)` computed directly from `x`) was prototyped to cover the tiny-`|ln|`
>   band at full precision. It was abandoned: the version committed first did not converge
>   (infinite loop on `ln(1.5)`, caught only by a no-cache test run), and after capping the
>   series it was still numerically wrong (`ln(1.5)` off by ~2e-3). Rather than ship a
>   second-best approximation, the band keeps the accurate float64 fallback. The episode is
>   a good reminder that an independent, cold (`-count=1`, cache-cleared) test is what
>   actually catches a divergent/incorrect path — a cached green run hid both bugs.


## Goal

When the caller asks for `precision >= 16`, return `Ln` accurate to the **type's full
mantissa (57 bits ≈ 17.1 decimal digits)** instead of being capped at `float64`'s
~15.95 digits — by adding **one extra refinement series** seeded from `math.Log`, computed
entirely in `Decimal` arithmetic. For `precision < 16` keep the current fast float64 path
unchanged.

## Precision budget (why there is room to gain)

| Source | Mantissa | Decimal digits | Relative error |
|---|---:|---:|---:|
| `float64` (`math.Log`) | 53 bits | ~15.95 | ~1.1e-16 |
| this `Decimal` type | 57 bits | ~17.15 | ~1.5e-17 |
| **gap to recover** | 4 bits | **~1.2 digits** | ~10× |

`math.Log` itself is typically `< 1 ULP` accurate, so the current result is good to ~16
digits but the 17th printed digit is the float64 tail, **not** a correct digit of `ln(d)`.

## Current implementation

```go
func (d Decimal) Ln(precision int32) Decimal {
	f, x := d.Float64()
	return NewFromFloat64Exact(math.Log(f), x).Round(precision)
}
```

Goes `d -> float64 -> math.Log -> Decimal -> Round`. Hard-capped at float64 precision.

## Empirical finding — the *naive* single step gains nothing

Prototype tested (`y0 = log(f)`, `E = exp(y0)`, `u = d/E - 1`, `ln ≈ y0 + u - u²/2`):

```
x=2.5  cur=0.9162907318741549  refined=0.9162907318741549  u=0
x=7    cur=1.9459101490553131  refined=1.9459101490553131  u=0
```

`u` collapses to **exactly 0**. Two independent reasons, both must be fixed:

1. **Sub-precision residual.** `d/E - 1` is ~1e-16; formed by a single `Decimal` `Div`
   (17-digit) it rounds straight to `1`, so the residual is lost before it can be used.
2. **float64 contamination.** Even if kept, `E = math.Exp(y0)` carries ~1e-16 error — the
   same magnitude as the correction — so the correction would be noise, not signal.

**Conclusion: any refinement that relies on `math.Exp` for the residual cannot beat
float64.** The fix must (a) compute the series in a representation that holds the extra
digits, and (b) avoid float64 in the final digits.

## Recommended approach — argument reduction + `atanh` series, all in `Decimal`

This is the numerically stable, self-contained route. It uses `math.Log` only to *seed/verify*,
never for the final digits.

1. **Decompose** `d = m · 2^k` with `m ∈ [√½, √2)` (so `ln m` is small and the series
   converges fast). `k` and the reduction are exact in `Decimal`.
2. **High-precision constant** `LN2` stored to ≥19 significant digits as a `Decimal`
   literal (`0.6931471805599453094`).
3. **Single series** (one Taylor/`atanh` expansion — this is the "une seule série"):

   ```
   s = (m - 1) / (m + 1)          // |s| < 0.172, computed in Decimal
   ln(m) = 2·s·(1 + s²/3 + s⁴/5 + s⁶/7 + …)
   ```

   Sum terms in `Decimal` until a term is below `1e-18` (≈ 5 terms suffice for |s|<0.172,
   since 0.172^(2n)/(2n+1) drops past 1e-18 by n≈8 — bounded, allocation-free loop).
4. **Recombine** `ln(d) = k·LN2 + ln(m)`, then `Round(precision)`.

Because every step after the seed runs in `Decimal` and the series residual is held above
the truncation threshold, the result saturates the 57-bit mantissa instead of float64.

### Cheaper alternative if full reduction is too invasive

Keep `y0 = math.Log(f)` as seed but make the correction *carry the extra digits*:
- form the residual at scaled precision: `w = d - exp_decimal(y0)` where `exp_decimal` is a
  **double-double** evaluation (`math.Exp(y0)` hi + a Decimal correction), then one Newton
  step `y0 + w / exp_decimal(y0)`.
- This needs a `Decimal`-precision `exp` helper, so it is strictly more work than the atanh
  series. **Prefer the atanh series.**

## Implementation tasks

1. Add unexported helpers in `decimal.go` / `core.go`:
   - `lnReduce(d) (m Decimal, k int64)` — exact `m·2^k` split.
   - `lnSeries(m Decimal) Decimal` — the `atanh` series, bounded loop, zero-alloc.
   - `const lnTwo = ...` (19-digit `Decimal` literal for `ln 2`).
2. Branch in `Ln`: `if precision < 16 { /* current fast path */ } else { /* reduce + series */ }`.
3. Keep `NaN`/`±Inf`/`d<=0` behaviour identical (propagate, no error) — the new path must
   return `NaN` for `d <= 0` just like the float path.
4. Stay allocation-free on the hot path (`go test -bench=Ln -benchmem`, `allocs/op == 0`).

## Verification plan

- New test `TestLnHighPrecision`: build a `big.Float` reference at 60-bit-plus precision
  (implement `bigLn` via the same `atanh` series in `math/big`, or via `big.Float` from a
  string constant table), sweep `d ∈ {0.3, 0.5, 1.5, 2, e, 7, 10, 123.456, 1e6, 1e-6}`,
  assert agreement to **≤ 1e-17 relative**.
- Regression: `precision < 16` results must be byte-identical to the current implementation.
- Re-run `TestMathAccuracy` (already green to 1e-13) — must stay green and tighten where the
  new path applies.
- Benchmark: confirm the `precision >= 16` path stays zero-alloc and document the new ns/op
  (expect a small constant-factor cost over the float path for the extra series terms).

## Risks / open questions

- **Is the gain worth it?** The type holds only ~1.2 digits beyond float64. Confirm the
  recovered digit is *correct* (vs `big.Float`) before shipping — a wrong 17th digit is
  worse than an honestly-truncated 16th.
- **Rounding of `LN2`/series at the last digit** — validate against `big.Float` across the
  sweep, not just spot values.
- **`k·LN2` cancellation** for `d` near 1 (small `k`, `m` near 1): the `atanh` series handles
  `m≈1` well (`s≈0`), so this is the good case; the risk is large `k` where `k·LN2` dominates
  and its own rounding caps precision — store `LN2` with 2 guard digits.
