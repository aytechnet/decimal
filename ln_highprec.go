package decimal

import (
	"math/bits"
)

// High-precision Ln path.
//
// When precision >= lnHighPrecMin, Ln routes a normal strictly-positive finite
// decimal here to fill the type's full 57-bit mantissa instead of being capped at
// float64's ~16 digits (see LN_PRECISION_PLAN.md).
//
// Method — everything past the seed runs in a 2^63-scaled fixed point held in
// raw uint64 / 128-bit integers, i.e. the FULL 64-bit mantissa, ~19 significant
// digits, two more than the public 57-bit Decimal:
//
//  1. binary reduction of the *value* (not the decimal exponent, which would
//     cause catastrophic cancellation for values near 1): m·10^e = a·2^K with
//     a in [1, 2), so s = (a-1)/(a+1) has |s| < 1/3.
//  2. atanh series: ln(a) = 2·(s + s³/3 + s⁵/5 + …).
//  3. recombine ln = K·ln2 + ln(a), all 2^63-scaled, then round.
//
// Values whose result |ln| is tiny (x within ~1% of 1) fall back to the float64
// path: there the 2^63 fixed-point scale loses relative precision and math.Log is
// more accurate. (A dedicated 128-bit near-1 variant was prototyped but did not
// reach the required accuracy, so it was dropped in favour of the float64
// fallback — see LN_PRECISION_PLAN.md.)
const (
	lnHighPrecMin = 16
	lnHalf        = uint64(1) << 63 // 2^63, the fixed-point scale (a = F/2^63)
	// ln(2) scaled by 2^63, round-to-nearest (= round(ln2 · 2^63)):
	//   ln 2 = 0.69314718055994530941723212...
	lnTwoS = 6393154322601327829
)

// lnSmallScaled is the smallest result magnitude (2^63-scaled) for which the
// fixed-point path keeps full relative precision. The 2^63 fixed point has
// absolute resolution 2^-63 ≈ 1.08e-19; a result R retains ~17 relative digits
// only while 2^-63 / R <= 1e-17, i.e. |ln| >= ~0.0108. Below that the float64
// path (math.Log) is more accurate, so the caller falls back. Set at ~0.01.
const lnSmallScaled = (uint64(1) << 63) / 100 // 0.01 · 2^63 ≈ 9.2e16

// div128 divides the 128-bit value (hi:lo) by d, returning the 128-bit quotient
// (qhi:qlo) and the remainder. Requires d > 0.
func div128(hi, lo, d uint64) (qhi, qlo, rem uint64) {
	qhi = hi / d
	qlo, rem = bits.Div64(hi%d, lo, d)
	return
}

// norm128 normalizes the non-zero 128-bit value (hi:lo) to F·2^G with F in
// [2^63, 2^64). Truncating (no rounding) — the dropped bits are below the 64-bit
// mantissa, i.e. < ~1e-19 relative, negligible for an 18-digit result.
func norm128(hi, lo uint64) (F uint64, G int64) {
	if hi == 0 {
		lz := bits.LeadingZeros64(lo) // lo != 0
		return lo << lz, int64(-lz)
	}
	lz := bits.LeadingZeros64(hi)
	if lz == 0 {
		return hi, 64
	}
	return hi<<lz | lo>>(64-lz), int64(64 - lz)
}

// mulS multiplies two 2^63-scaled fixed-point values, rounding to nearest.
func mulS(x, y uint64) uint64 {
	hi, lo := bits.Mul64(x, y)
	r := hi<<1 | lo>>63 // (x*y) >> 63
	if lo&(uint64(1)<<62) != 0 {
		r++
	}
	return r
}

// lnHighPrec computes ln(m·10^e) for m > 0 and e in the Decimal exponent range,
// rounded to `precision` decimal places. The caller guarantees a normal,
// strictly-positive, finite operand. The bool result is false when the result is
// too small for the fixed-point scale to beat float64 (value near 1) — the caller
// must then use the float64 path.
func lnHighPrec(m uint64, e int64, precision int32) (Decimal, bool) {
	// --- 1. reduce value = m·10^e to F·2^G, F in [2^63, 2^64) ---
	var F uint64
	var G int64
	if e >= 0 {
		hi, lo := bits.Mul64(m, tenPow[e]) // m·10^e, exact (m<2^57, 10^15<2^50)
		F, G = norm128(hi, lo)
	} else {
		// Normalize m to 64 bits FIRST, then divide, so the dividend (mn:0) is a
		// full 128-bit number and the quotient keeps full precision. Dividing the
		// un-normalized m·2^64 would drop the low quotient bits for small m
		// (e.g. ln(1e-7) lost ~13 bits → error ~3e-14).
		lz := bits.LeadingZeros64(m)
		mn := m << lz                            // mn in [2^63, 2^64)
		qhi, qlo, _ := div128(mn, 0, tenPow[-e]) // mn·2^64 / 10^|e|
		nf, ng := norm128(qhi, qlo)
		// value = (mn/10^|e|)·2^-lz = (mn·2^64/10^|e|)·2^(-64-lz) = nf·2^(ng-64-lz)
		F, G = nf, ng-64-int64(lz)
	}
	K := G + 63 // a = F/2^63 in [1,2), value = a·2^K

	// --- 2. s = (a-1)/(a+1) at scale 2^63 ---
	// s = (F-2^63)·2^62 / ((F+2^63)>>1)
	num := F - lnHalf
	dlo, c := bits.Add64(F, lnHalf, 0)
	d2 := dlo>>1 | c<<63
	nhi, nlo := bits.Mul64(num, uint64(1)<<62)
	_, s, _ := div128(nhi, nlo, d2) // s at scale 2^63, in [0, 2^62)

	// --- 3. atanh series: ln(a) = 2·(s + s³/3 + s⁵/5 + …) ---
	s2 := mulS(s, s)
	sum := s
	t := s
	for n := uint64(3); n < 200; n += 2 {
		t = mulS(t, s2)
		term := t / n
		if term == 0 {
			break
		}
		sum += term
	}
	lnAS := 2 * sum // ln(a) at scale 2^63, in [0, 0.4·2^63)

	// --- 4. total = K·ln2 + ln(a) at scale 2^63 (signed 128-bit magnitude) ---
	kAbs := uint64(K)
	if K < 0 {
		kAbs = uint64(-K)
	}
	khi, klo := bits.Mul64(kAbs, lnTwoS) // |K|·ln2

	var thi, tlo uint64
	var resNeg bool
	if K >= 0 {
		// total = K·ln2 + lnAS  (>= 0)
		tlo, c = bits.Add64(klo, lnAS, 0)
		thi = khi + c
	} else {
		// total = lnAS - |K|·ln2  (< 0 since |K|·ln2 >= ln2 > lnAS for K<=-1)
		resNeg = true
		blo, b := bits.Sub64(klo, lnAS, 0)
		thi, tlo = khi-b, blo
	}

	// Fall back to the float64 path when the result magnitude is too small for the
	// 2^63 fixed point to keep full relative precision. Catches values near 1 from
	// either side: K==0 with tiny s, and K==-1 with a≈2 where K·ln2 and ln(a) nearly
	// cancel (e.g. ln(0.9999)). math.Log is more accurate in that tiny-|ln| regime.
	if thi == 0 && tlo < lnSmallScaled {
		return 0, false
	}

	// --- 5. scale-2^63 magnitude (thi:tlo) -> Decimal ---
	q := thi<<1 | tlo>>63          // integer part = total >> 63
	r := tlo & (uint64(1)<<63 - 1) // fractional part, /2^63
	fhi, flo := bits.Mul64(r, 1_000_000_000_000_000_000)
	frac := fhi<<1 | flo>>63 // round(r/2^63 · 10^18), < 10^18
	if flo&(uint64(1)<<62) != 0 {
		frac++
	}

	res := New(int64(q), 0).Add(New(int64(frac), -18))
	if resNeg {
		res = -res
	}
	return res.Round(precision), true
}
