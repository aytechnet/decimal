package decimal

// These tests exercise core.go branches that are not reachable via the current
// 8-byte Decimal/Weight public APIs but are kept in the core because core.go is
// designed to also serve a wider (16-byte) decimal type with a larger mantissa.
// They call the internal functions directly with hand-picked VME tuples.

import (
	"math"
	"testing"
)

func TestVeNormalizeMagicNoLoss(t *testing.T) {
	// when v has no loss bit, veNormalizeMagic returns (v, 0, 0)
	if v, m, e := veNormalizeMagic(0, 5, decimalMinE, decimalMaxE); v != 0 || m != 0 || e != 0 {
		t.Errorf(`veNormalizeMagic(0, 5, ...) should be (0,0,0), got (%d,%d,%d)`, v, m, e)
	}
	if v, m, e := veNormalizeMagic(sign, 5, decimalMinE, decimalMaxE); v != sign || m != 0 || e != 0 {
		t.Errorf(`veNormalizeMagic(sign, 5, ...) should be (sign,0,0), got (%d,%d,%d)`, v, m, e)
	}
}

func TestVmhmeReduceWideMantissa(t *testing.T) {
	// simulate a wide-mantissa multiplication where mh stays >= 10^19 after the first division
	// (impossible with Decimal/Weight 8-byte mantissas but covered for the future 16-byte type)
	const big = uint64(0xffffffffffffffff) // > 10^19
	v, m, e := vmhmeReduce(0, big, 0, 0)

	// Result must encode loss because precision was inevitably dropped.
	if v&loss == 0 {
		t.Errorf(`vmhmeReduce with mh > 10^19 should set loss bit, got v=0x%x`, v)
	}
	// e is shifted up by ~20 to absorb the high word
	if e <= 0 {
		t.Errorf(`vmhmeReduce should bump e, got e=%d`, e)
	}
	// m fits in uint64
	_ = m

	// also exercise the rounding-up path inside the second `if mh > 0` (rm >= 5)
	v2, _, _ := vmhmeReduce(0, big, 5, 0)
	if v2&loss == 0 {
		t.Errorf(`vmhmeReduce should still mark loss when low word has remainder`)
	}
}

func TestUnitHashMultibyte(t *testing.T) {
	// a rune >= 256 takes the primeUnicodeHi branch
	h1 := unitHash("kg")
	h2 := unitHash("kĀ") // U+0100 LATIN CAPITAL LETTER A WITH MACRON, > 256
	if h1 == h2 || h2 == 0 {
		t.Errorf(`unitHash with rune >= 256 must use a different prime, h1=%x h2=%x`, h1, h2)
	}
}

func TestVmetBytesToTinyExponent(t *testing.T) {
	// vmetBytesTo with e = -20 (one beyond decimalMinE) lets the main loop reach the position of the
	// decimal point at i = 19 with `output == false`, so the `if !output { append '0' }` branch fires.
	// This exponent is unreachable from a normalized Decimal (clamped to [-16, 15]) but a wider type
	// reusing core.go can produce it, which is what the branch defends against.
	b := vmetBytesTo(nil, 0, 1, -20, 0, nil, true, false)
	if string(b) != "0.00000000000000000001" {
		t.Errorf(`vmetBytesTo(0, 1, -20) should be "0.00000000000000000001", got %q`, string(b))
	}
	// negative sign on the same value
	b = vmetBytesTo(nil, sign, 1, -20, 0, nil, true, false)
	if string(b) != "-0.00000000000000000001" {
		t.Errorf(`vmetBytesTo(sign, 1, -20) should be "-0.00000000000000000001", got %q`, string(b))
	}
}

func TestVmetBytesToQuoted(t *testing.T) {
	// str=true wraps the output with double quotes — branch reachable only via internal callers
	b := vmetBytesTo(nil, 0, 12345, -2, 0, nil, true, true)
	if string(b) != `"123.45"` {
		t.Errorf(`vmetBytesTo str=true on 123.45 should be "123.45", got %q`, string(b))
	}
	// magic value with str=true (NaN) — exercises both quote insertions for the magic path
	b = vmetBytesTo(nil, loss, 0, 1, 0, nil, true, true)
	if string(b) != `"NaN"` {
		t.Errorf(`vmetBytesTo str=true on NaN should be "NaN", got %q`, string(b))
	}
	// integer with quote and a unit
	u := unit{u: "kg"}
	b = vmetBytesTo(nil, 0, 4, 0, 0, &u, true, true)
	if string(b) != `"4kg"` {
		t.Errorf(`vmetBytesTo str=true on 4kg should be "4kg", got %q`, string(b))
	}
}

func TestVeMagicBytesToCompact(t *testing.T) {
	// ext=false → JSON-friendly output: "0" for ~0 and ±~0; "null" for NaN and ±Inf
	if s := string(veMagicBytesTo(nil, sign|loss, 0, false)); s != "0" {
		t.Errorf(`veMagicBytesTo ext=false on ~0 should be "0", got %q`, s)
	}
	if s := string(veMagicBytesTo(nil, loss, math.MaxInt64, false)); s != "null" {
		t.Errorf(`veMagicBytesTo ext=false on +Inf should be "null", got %q`, s)
	}
	if s := string(veMagicBytesTo(nil, loss, 1, false)); s != "null" {
		t.Errorf(`veMagicBytesTo ext=false on NaN should be "null", got %q`, s)
	}
	if s := string(veMagicBytesTo(nil, loss, math.MinInt64, false)); s != "0" {
		t.Errorf(`veMagicBytesTo ext=false on +~0 should be "0", got %q`, s)
	}
}

func TestVmeAddDichotomyHighWord(t *testing.T) {
	// craft a vmeAdd call where the dichotomic search of `h2` traverses every "i = k" branch (h2 >= tenPow[k])
	// a wide-mantissa ratio (m2 close to 2^63) mimics what a 16-byte decimal would feed in
	bigM := uint64(1) << 62
	v, _, _ := vmeAdd(0, bigM, 0, 0, bigM, 19)
	// loss is expected because precision is dropped
	if v&loss == 0 {
		t.Errorf(`vmeAdd should mark loss for wide-mantissa addition`)
	}

	// also exercise the symmetric `j = k` branches: pick a small h2 (≈ 54) that falls below tenPow[5]
	// so each step of the dichotomy enters the "j = k" half
	v2, _, _ := vmeAdd(0, 1, 0, 0, 100, 19)
	if v2&loss == 0 {
		t.Errorf(`vmeAdd with small h2 should still loss (precision dropped)`)
	}
}

func TestVmeMulOverflowExponent(t *testing.T) {
	// e1+e2 overflow is unreachable from Decimal (|e| <= 15) but must be handled for wider types
	v, m, e := vmeMul(0, 1, math.MaxInt64-10, 0, 1, 100)
	if v&loss == 0 || m != 0 || e != math.MaxInt64 {
		t.Errorf(`vmeMul positive-exponent overflow should be (loss,0,MaxInt64), got (%x,%d,%d)`, v, m, e)
	}
	v, m, e = vmeMul(0, 1, math.MinInt64+10, 0, 1, -100)
	if v&loss == 0 || m != 0 || e != math.MinInt64 {
		t.Errorf(`vmeMul negative-exponent overflow should be (loss,0,MinInt64), got (%x,%d,%d)`, v, m, e)
	}
}

func TestVmeMulMagic1Paths(t *testing.T) {
	// d1 == ~0 (e1 == 0), d2 == 0 → return Zero
	if v, m, e := vmeMulMagic1(loss, 0, sign, 0, 0); v != sign || m != 0 || e != 0 {
		t.Errorf(`~0 * 0 should be Zero, got (%x,%d,%d)`, v, m, e)
	}
	// d1 == ~0, d2 == NaN → NaN
	if v, _, e := vmeMulMagic1(loss, 0, loss, 0, 1); v&loss == 0 || e != 1 {
		t.Errorf(`~0 * NaN should be NaN, got (%x, _, %d)`, v, e)
	}
	// d1 == +~0, d2 == NaN → NaN
	if v, _, e := vmeMulMagic1(loss, math.MinInt64, loss, 0, 1); v&loss == 0 || e != 1 {
		t.Errorf(`+~0 * NaN should be NaN, got (%x, _, %d)`, v, e)
	}
	// d1 == +~0, d2 == ~0 → ~0
	if v, _, e := vmeMulMagic1(loss, math.MinInt64, sign|loss, 0, 0); v != sign|loss || e != 0 {
		t.Errorf(`+~0 * ~0 should be ~0, got (%x, _, %d)`, v, e)
	}
	// d1 == +~0, d2 == 0 (no loss bit) → 0
	if v, _, _ := vmeMulMagic1(loss, math.MinInt64, sign, 0, 0); v != sign {
		t.Errorf(`+~0 * 0 should be Zero (sign,0,0), got v=%x`, v)
	}
	// d1 == +Inf, d2 == ~0 (NaN per IEEE-like rule) → NaN
	if v, _, e := vmeMulMagic1(loss, math.MaxInt64, sign|loss, 0, 0); v&loss == 0 || e != 1 {
		t.Errorf(`+Inf * ~0 should be NaN, got (%x, _, %d)`, v, e)
	}
	// d1 == +Inf, d2 == 0 (no loss bit) → NaN
	if v, _, e := vmeMulMagic1(loss, math.MaxInt64, sign, 0, 0); v&loss == 0 || e != 1 {
		t.Errorf(`+Inf * 0 should be NaN, got (%x, _, %d)`, v, e)
	}
	// d1 is NaN (default e1) → NaN
	if v, _, e := vmeMulMagic1(loss, 5, 0, 1, 0); v&loss == 0 || e != 1 {
		t.Errorf(`NaN * x should be NaN, got (%x, _, %d)`, v, e)
	}
}

func TestVmeAddMagic1InfPaths(t *testing.T) {
	// +Inf + ±~0 magic — d2 has loss bit, e2 == MinInt64 — returns d1 unchanged
	if v, _, e := vmeAddMagic1(loss, math.MaxInt64, loss, 0, math.MinInt64); v&loss == 0 || e != math.MaxInt64 {
		t.Errorf(`+Inf + ±~0 should keep +Inf, got (%x, _, %d)`, v, e)
	}
	// +Inf + NaN — d2 is a NaN-magic decimal (e2 != MaxInt64, != MinInt64)
	if v, _, e := vmeAddMagic1(loss, math.MaxInt64, loss, 0, 1); v&loss == 0 || e != 1 {
		t.Errorf(`+Inf + NaN should propagate NaN, got (%x, _, %d)`, v, e)
	}
}

func TestVmeDivRemMagic2NaNPaths(t *testing.T) {
	// d1 == ~0, d2 == ±~0 → NaN
	if v, _, e, _, _ := vmeDivRemMagic2(loss, 0, 0, loss, math.MinInt64); v&loss == 0 || e != 1 {
		t.Errorf(`~0 / ±~0 should be NaN, got (%x, _, %d)`, v, e)
	}
	// d1 == +Inf, d2 == ±~0 → ±Inf
	if v, _, e, _, _ := vmeDivRemMagic2(loss, 0, math.MaxInt64, loss, math.MinInt64); v&loss == 0 || e != math.MaxInt64 {
		t.Errorf(`+Inf / ±~0 should be Inf, got (%x, _, %d)`, v, e)
	}
	// d1 == ~0, d2 == ±Inf → ~0
	if v, _, e, _, _ := vmeDivRemMagic2(loss, 0, 0, loss, math.MaxInt64); v != (sign|loss) || e != 0 {
		t.Errorf(`~0 / ±Inf should be ~0, got (%x, _, %d)`, v, e)
	}
	// d1 == +~0, d2 == ±Inf → ±~0
	if v, _, e, _, _ := vmeDivRemMagic2(loss, 0, math.MinInt64, loss, math.MaxInt64); v&loss == 0 || e != math.MinInt64 {
		t.Errorf(`+~0 / ±Inf should be ~+0, got (%x, _, %d)`, v, e)
	}
	// d1 == +Inf, d2 == ±Inf → NaN
	if v, _, e, _, _ := vmeDivRemMagic2(loss, 0, math.MaxInt64, loss, math.MaxInt64); v&loss == 0 || e != 1 {
		t.Errorf(`+Inf / ±Inf should be NaN, got (%x, _, %d)`, v, e)
	}
	// d1 == 0 (no loss), d2 == ±Inf → ~0
	if v, _, e, _, _ := vmeDivRemMagic2(0, 0, 0, loss, math.MaxInt64); v != (sign|loss) || e != 0 {
		t.Errorf(`0 / ±Inf should be ~0, got (%x, _, %d)`, v, e)
	}
	// d1 == ordinary, d2 == ±Inf → ~+0
	if v, _, e, _, _ := vmeDivRemMagic2(0, 1, 0, loss, math.MaxInt64); v&loss == 0 || e != math.MinInt64 {
		t.Errorf(`x / ±Inf should be ~+0, got (%x, _, %d)`, v, e)
	}
	// d2 == ~0 (e2 == 0)
	if v, _, e, _, _ := vmeDivRemMagic2(0, 1, 0, loss, 0); v&loss == 0 || e != 1 {
		t.Errorf(`x / ~0 should be NaN, got (%x, _, %d)`, v, e)
	}
	// fall-through: d2 has loss but unrecognized e2 → defensive NaN at the bottom
	if v, _, e, _, _ := vmeDivRemMagic2(0, 1, 0, loss, 5); v&loss == 0 || e != 1 {
		t.Errorf(`fall-through default should be NaN, got (%x, _, %d)`, v, e)
	}
}

func TestVmeDivRemMagicNumerator(t *testing.T) {
	// d1 magic (NaN) divided by an ordinary d2: branch at line 888
	v, _, _, _, _ := vmeDivRem(loss, 0, 5, 0, 1, 0, 0)
	if v&loss == 0 {
		t.Errorf(`NaN / x should keep loss bit`)
	}
	// d2 == 0 without loss bit: branch at line 881
	v, m, e, _, _ := vmeDivRem(0, 1, 0, 0, 0, 0, 0)
	if v&loss == 0 || m != 0 || e != 1 {
		t.Errorf(`x / 0 (no loss) should be NaN, got (%x,%d,%d)`, v, m, e)
	}
}

func TestVmeRoundBankMagic(t *testing.T) {
	// vmeRoundBank on NaN/Inf magic values: m == 0 with e != 0, != MinInt64 → returns input unchanged
	if v, m, e := vmeRoundBank(loss, 0, 1, 2); v != loss || m != 0 || e != 1 {
		t.Errorf(`vmeRoundBank(NaN) should return input unchanged, got (%x,%d,%d)`, v, m, e)
	}
	if v, m, e := vmeRoundBank(loss, 0, math.MaxInt64, 2); v != loss || m != 0 || e != math.MaxInt64 {
		t.Errorf(`vmeRoundBank(+Inf) should return input unchanged, got (%x,%d,%d)`, v, m, e)
	}

	// vmeRoundBank with (m << 1) < p (places exceeds mantissa precision) → Zero
	if v, m, e := vmeRoundBank(0, 1, -3, 1); v != sign || m != 0 || e != 0 {
		t.Errorf(`vmeRoundBank(0.001, 1) should be Zero, got (%x,%d,%d)`, v, m, e)
	}
}

func TestVmeRoundFamilyPlacesUnderflow(t *testing.T) {
	// Round/RoundBank/RoundCeil/RoundFloor with -i >= len(tenPow) hit the defensive `else { return Zero }` branch
	// The current 8-byte Decimal can reach this via Round(places=-100) on a normal value.
	if v, m, e := vmeRound(0, 1, 0, -100); v != sign || m != 0 || e != 0 {
		t.Errorf(`vmeRound underflow should be Zero, got (%x,%d,%d)`, v, m, e)
	}
	if v, m, e := vmeRoundBank(0, 1, 0, -100); v != sign || m != 0 || e != 0 {
		t.Errorf(`vmeRoundBank underflow should be Zero, got (%x,%d,%d)`, v, m, e)
	}
	if v, m, e := vmeRoundCeil(sign, 1, 0, -100); v != sign || m != 0 || e != 0 {
		t.Errorf(`vmeRoundCeil underflow should be Zero, got (%x,%d,%d)`, v, m, e)
	}
	if v, m, e := vmeRoundFloor(0, 1, 0, -100); v != sign || m != 0 || e != 0 {
		t.Errorf(`vmeRoundFloor underflow should be Zero, got (%x,%d,%d)`, v, m, e)
	}

	// And the 1004-1006 / 992-994 branches in vmeRoundBank: m << 1 < p path
	if v, m, e := vmeRoundBank(0, 1, -3, 1); v != sign || m != 0 || e != 0 {
		t.Errorf(`vmeRoundBank(0.001, 1) should be Zero, got (%x,%d,%d)`, v, m, e)
	}

	// vmeRound 967-969: defensive overflow branch when -i >= len(tenPow)
	// Already covered by underflow tests above.
}

func TestVmeFromBytesEdgeCases(t *testing.T) {
	// a lone "-" must error — covers the i > j branch after parsing the sign
	if _, _, _, err := vmeFromBytes([]byte("-"), nil); err == nil {
		t.Errorf(`vmeFromBytes("-") should error`)
	}
	// a lone "+" must error
	if _, _, _, err := vmeFromBytes([]byte("+"), nil); err == nil {
		t.Errorf(`vmeFromBytes("+") should error`)
	}
	// a sign followed only by "~" then nothing
	if _, _, _, err := vmeFromBytes([]byte("-~"), nil); err == nil {
		t.Errorf(`vmeFromBytes("-~") should error`)
	}
	// a "~" then nothing
	if _, _, _, err := vmeFromBytes([]byte("~"), nil); err == nil {
		t.Errorf(`vmeFromBytes("~") should error`)
	}

	// integer mantissa overflow without a decimal point exercises the `doti < 0 && e > 0` increment path
	// using more digits than a uint64 can hold (20+ digits before the implicit point)
	if v, _, e, err := vmeFromBytes([]byte("99999999999999999999999"), nil); err != nil || e == 0 || v&loss == 0 {
		t.Errorf(`vmeFromBytes on 23-digit integer should bump exponent and mark loss, got v=%x e=%d err=%v`, v, e, err)
	}
}
