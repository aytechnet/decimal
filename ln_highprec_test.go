package decimal

import (
	"math"
	"math/big"
	"testing"
)

// bigLn is an independent high-precision reference for ln(x), x > 0, computed
// with math/big at 240-bit precision via the same atanh identity but with no
// shared code with the implementation under test. ln(2) is obtained from
// atanh(1/3) (= ½·ln2... actually 2·atanh(1/3) = ln2) so there is no circularity.
func bigLn(x float64) *big.Float {
	const prec = 240
	bx := new(big.Float).SetPrec(prec).SetFloat64(x)
	one := new(big.Float).SetPrec(prec).SetInt64(1)
	two := new(big.Float).SetPrec(prec).SetInt64(2)

	// reduce bx to a in [2/3, 4/3) by halving/doubling, tracking k (ln2 count)
	k := 0
	for bx.Cmp(new(big.Float).SetPrec(prec).SetFloat64(1.3333333333333333)) >= 0 {
		bx.Quo(bx, two)
		k++
	}
	for bx.Cmp(new(big.Float).SetPrec(prec).SetFloat64(0.6666666666666666)) < 0 {
		bx.Mul(bx, two)
		k--
	}

	lnA := atanhSeries(bx, prec) // ln(a)
	// ln2 via 2·atanh(1/3)
	third := new(big.Float).SetPrec(prec).Quo(one, new(big.Float).SetPrec(prec).SetInt64(3))
	ln2 := atanhLn(third, prec) // = ln((1+1/3)/(1-1/3)) = ln 2

	res := new(big.Float).SetPrec(prec).Mul(new(big.Float).SetPrec(prec).SetInt64(int64(k)), ln2)
	res.Add(res, lnA)
	return res
}

// atanhLn returns ln((1+s)/(1-s)) = 2·atanh(s) for |s| < 1.
func atanhLn(s *big.Float, prec uint) *big.Float {
	sum := new(big.Float).SetPrec(prec).Set(s)
	term := new(big.Float).SetPrec(prec).Set(s)
	s2 := new(big.Float).SetPrec(prec).Mul(s, s)
	limit := new(big.Float).SetPrec(prec).SetFloat64(math.Pow(2, -230))
	for n := int64(3); ; n += 2 {
		term.Mul(term, s2)
		t := new(big.Float).SetPrec(prec).Quo(term, new(big.Float).SetPrec(prec).SetInt64(n))
		sum.Add(sum, t)
		abs := new(big.Float).SetPrec(prec).Abs(t)
		if abs.Cmp(limit) < 0 {
			break
		}
	}
	return sum.Mul(sum, new(big.Float).SetPrec(prec).SetInt64(2))
}

// atanhSeries returns ln(a) for a near 1 via s = (a-1)/(a+1).
func atanhSeries(a *big.Float, prec uint) *big.Float {
	one := new(big.Float).SetPrec(prec).SetInt64(1)
	num := new(big.Float).SetPrec(prec).Sub(a, one)
	den := new(big.Float).SetPrec(prec).Add(a, one)
	s := new(big.Float).SetPrec(prec).Quo(num, den)
	return atanhLn(s, prec)
}

func TestLnHighPrecision(t *testing.T) {
	inputs := []string{
		"0.001", "0.5", "1.1", "1.5", "2", "2.5", "3", "7", "10",
		"42", "100", "123.456", "0.3", "0.9999", "1.0001", "9999999",
		"0.0000001", "98765432.1",
	}
	// The Decimal type resolves ~1.5e-17 relative (57-bit mantissa). The high-precision
	// path is typically correct to < 5e-17 (near 1, via the 128-bit lnNear1 path, down to
	// ~1e-18); for large |ln| the single-word ln2 constant caps it near ~6e-17 (~55 bits).
	// We assert 1e-16 — consistently ~10× better than the float64 path on the same inputs,
	// which is the point. Crucially we parse got.String() into big.Float (NOT got.Float64(),
	// which would collapse the result back to float64 and hide the precision we are proving).
	const tol = 1e-16
	for _, in := range inputs {
		d := RequireFromString(in)
		got := d.Ln(18)

		ref := bigLn(mustFloat(in))
		gotBig, _, perr := new(big.Float).SetPrec(240).Parse(got.String(), 10)
		if perr != nil {
			t.Fatalf("parse %q: %v", got.String(), perr)
		}
		rel := relErr(gotBig, ref)

		// float64 path, parsed the same way, for context
		f64 := NewFromFloat64Exact(math.Log(mustFloat(in)), true).Round(18)
		f64Big, _, _ := new(big.Float).SetPrec(240).Parse(f64.String(), 10)
		f64rel := relErr(f64Big, ref)

		status := "OK"
		if rel > tol {
			status = "** TOO COARSE **"
			t.Errorf("Ln(%s): rel err %.2e > tol %.0e (got=%s)", in, rel, tol, got.String())
		}
		t.Logf("Ln(%-12s) hp-rel=%.2e  float64path-rel=%.2e  %s", in, rel, f64rel, status)
	}
}

// relErr returns |got-ref| / max(1,|ref|) as a float64.
func relErr(got, ref *big.Float) float64 {
	diff := new(big.Float).SetPrec(240).Sub(got, ref)
	diff.Abs(diff)
	denom := new(big.Float).SetPrec(240).Abs(ref)
	if denom.Cmp(new(big.Float).SetInt64(1)) < 0 {
		denom.SetInt64(1)
	}
	r, _ := new(big.Float).SetPrec(240).Quo(diff, denom).Float64()
	return r
}

func mustFloat(s string) float64 {
	f, err := new(big.Float).SetPrec(240).SetString(s)
	_ = err
	v, _ := f.Float64()
	return v
}

// regression: precision < 16 must keep the exact float64-backed behaviour.
func TestLnLowPrecisionUnchanged(t *testing.T) {
	for _, in := range []string{"2", "2.5", "7", "0.5", "123.456"} {
		d := RequireFromString(in)
		f, x := d.Float64()
		want := NewFromFloat64Exact(math.Log(f), x).Round(15)
		if got := d.Ln(15); got != want {
			t.Errorf("Ln(%s) precision 15: got %s, want %s (low path must be unchanged)", in, got.String(), want.String())
		}
	}
}

// TestNorm128Branches exercises all three normalization branches of norm128
// directly: hi==0 (value fits in the low word), hi already top-bit-set (lz==0),
// and the general split across the word boundary (lz in 1..63).
func TestNorm128Branches(t *testing.T) {
	// hi == 0: lo gets left-shifted to fill bit 63, exponent goes negative.
	if F, G := norm128(0, 1); F != 1<<63 || G != -63 {
		t.Errorf("norm128(0,1) = (%#x, %d), want (2^63, -63)", F, G)
	}
	// hi already normalized (top bit set): returned as-is with G == 64.
	if F, G := norm128(1<<63, 0); F != 1<<63 || G != 64 {
		t.Errorf("norm128(2^63,0) = (%#x, %d), want (2^63, 64)", F, G)
	}
	// general case: hi nonzero with leading zeros, low bits shifted in.
	if F, G := norm128(1, 0); F != 1<<63 || G != 1 {
		t.Errorf("norm128(1,0) = (%#x, %d), want (2^63, 1)", F, G)
	}
}
