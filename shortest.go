package decimal

import (
	"math/bits"
)

// shortestDecimal finds the shortest decimal m×10^e that converts back to the binary float c×2^q,
// the nearest to it when several are as short — what strconv.FormatFloat(value, 'e', -1, bitSize)
// writes — as integers, without text. closer tells a float whose lower neighbour is twice as close
// as the upper one: a power of two (c is then 2^52 for a float64, 2^23 for a float32). It is the
// Schubfach algorithm of Raffaello Giulietti ("The Schubfach way to render doubles", 2020), on the
// scales a Decimal can hold: q from -125 to 83 (to 54 for a float64), which the caller checks.
//
// The idea. Every real number between the middle of the lower neighbour of the float and the middle
// of its upper neighbour rounds to it: that is its rounding interval, [c-1/2, c+1/2]×2^q (the lower
// half is 1/4 when closer). This interval, not the value, is what makes a float32 and the float64
// of the same value differ: 0.1 is inside the wide interval of the float32, not inside the narrow
// one of the float64. The shortest decimal is a number of the interval with the fewest digits.
// Schubfach picks the one scale 10^k at which the interval is wider than 1 but not wider than 10
// units: it then holds at least one multiple of 10^k (the pigeonhole — "Schubfach" — principle)
// and at most one multiple of 10^(k+1). So only four candidates exist: the multiple of 10^(k+1)
// just under or just above the value, which is shorter and wins when it is inside, then the
// multiple of 10^k just under or just above, the nearest of the two when both are inside.
//
// The arithmetic. The interval bounds and the value, each times 4 to stay integers (cbl, cb, cbr),
// are scaled by 10^-k: a product by a power of ten kept on 128 bits, of which the high 64 bits are
// taken. Its low bits are not dropped blindly: any non-zero remainder sets the lowest bit of the
// result (rounding to odd), which is enough to compare a scaled bound with a candidate exactly, as
// if the product were infinitely precise.
//
// The powers of ten come from the tenPow table, no other data (shortestPow10). The general
// algorithm carries a table of 617 powers to cover every float64; here a float outside 10^-38 to
// 10^24 is not a Decimal anyway.
func shortestDecimal(c uint64, q int64, closer bool) (m uint64, e int64) {
	// the bounds of the rounding interval and the value, times 4; the bounds belong to the interval
	// when c is even (a tie rounds to even)
	cbl, cb, cbr := 4*c-2, 4*c, 4*c+2
	if closer {
		cbl++
	}

	// k = floor(log10(2^q)), or floor(log10(3/4 × 2^q)) for the narrower interval: the scale whose
	// unit 10^k is the largest power of ten not exceeding the width of the interval
	k := (q * 1262611) >> 22
	if closer {
		k = (q*1262611 - 524031) >> 22
	}

	ghi, glo := shortestPow10(int(-k))

	// h brings the products back to quarters of 10^k: 2^q × 10^-k = g × 2^(h-128), 1 <= h <= 4
	// ((n × 1741647) >> 19 is floor(log2(10^n)))
	h := uint(q + (-k*1741647)>>19 + 1)

	vbl := shortestRoundToOdd(ghi, glo, cbl<<h)
	vb := shortestRoundToOdd(ghi, glo, cb<<h)
	vbr := shortestRoundToOdd(ghi, glo, cbr<<h)

	lower, upper := vbl, vbr
	if c&1 != 0 {
		lower, upper = vbl+1, vbr-1
	}

	// s = floor(value / 10^k); vb, lower and upper are in quarters of 10^k
	s := vb / 4

	// one digit shorter: the multiples of 10^(k+1) around the value, sp and sp+1. They cannot both
	// be inside; the one that is wins
	if s >= 10 {
		sp := s / 10
		under := lower <= 40*sp
		above := 40*sp+40 <= upper
		if under != above {
			if above {
				sp++
			}
			return sp, k + 1
		}
	}

	// the multiples of 10^k around the value, s and s+1: the only one inside, or the nearest when
	// both are (the even one on a tie)
	under := lower <= 4*s
	above := 4*s+4 <= upper
	if under != above {
		if above {
			s++
		}
		return s, k
	}
	if mid := 4*s + 2; vb > mid || (vb == mid && s&1 != 0) {
		s++
	}
	return s, k
}

// shortestPow10 gives 10^n on 128 bits, shifted so that its highest bit is set, for n from -24 to 38,
// from the tenPow table.
//
// From 0 to 38, 10^n fits 128 bits — an entry, or the product of two — and is exact. A negative
// power is a quotient, rounded up as Schubfach wants its inexact powers: 1/10^j is 2^191 divided by
// 10^j shifted to its highest bit, plus one. Down to -17 that is all a float64 needs and the
// quotient is the floor of the true value, plus one, as in the published table. From -18 to -24,
// reached only by a float32 above 2^83, the first quotient is divided again by the rest of the
// power: it is then above the true value by a few units of its 128th bit, far more precise than
// the 64 bits a float32 needs (its products have 30 bits, not 59).
func shortestPow10(n int) (ghi, glo uint64) {
	if n >= 0 {
		if n < len(tenPow) {
			ghi = tenPow[n]
		} else {
			ghi, glo = bits.Mul64(tenPow[len(tenPow)-1], tenPow[n-len(tenPow)+1])
		}
		z := uint(bits.LeadingZeros64(ghi))
		return ghi<<z | glo>>(64-z), glo << z
	}

	j, rest := -n, 0
	if j > 17 {
		j, rest = 17, j-17
	}
	d := tenPow[j] << uint(bits.LeadingZeros64(tenPow[j]))
	ghi, r := bits.Div64(1<<63, 0, d)
	glo, _ = bits.Div64(r, 0, d)

	if rest > 0 {
		// (ghi, glo, 0) / 10^rest on 192 bits, shifted back to its highest bit
		var mid, low uint64
		ghi, r = bits.Div64(0, ghi, tenPow[rest])
		mid, r = bits.Div64(r, glo, tenPow[rest])
		low, _ = bits.Div64(r, 0, tenPow[rest])
		z := uint(bits.LeadingZeros64(ghi))
		ghi, glo = ghi<<z|mid>>(64-z), mid<<z|low>>(64-z)
	}

	glo, carry := bits.Add64(glo, 1, 0)
	return ghi + carry, glo
}

// shortestRoundToOdd gives the high 64 bits of the 192-bit product g × cp, its lowest bit set when
// the bits dropped below are not all zero.
func shortestRoundToOdd(ghi, glo, cp uint64) uint64 {
	xhi, _ := bits.Mul64(glo, cp)
	yhi, ylo := bits.Mul64(ghi, cp)
	ylo, carry := bits.Add64(ylo, xhi, 0)
	yhi += carry
	if ylo > 1 {
		yhi |= 1
	}
	return yhi
}
