package decimal

import (
	"testing"
)

func TestLengthConversions(t *testing.T) {
	var l0 Length

	if l0.String() != "0m" {
		t.Errorf(`l0.String() should be equal to 0m but l0 = %v`, l0)
	}

	l1, err := NewLengthFromString("10um")
	if err != nil {
		t.Errorf(`NewLengthFromString("10um") has result = %v and error = %v`, l1, err)
	}
	if l1.String() != "10µm" {
		t.Errorf(`l1 should be equal to 10µm but l1 = %v`, l1)
	}

	l1, err = NewLengthFromString("0cm")
	if err != nil {
		t.Errorf(`NewLengthFromString("0cm") has result = %v and error = %v`, l1, err)
	}
	if l1.String() != "0cm" {
		t.Errorf(`l1 should be equal to 0cm but l1 = %v (%016x)`, l1, uint64(l1))
	}

	l2, err := NewLengthFromString("-1km")
	if err != nil {
		t.Errorf(`NewLengthFromString("-1km") has result = %v and error = %v`, l2, err)
	}
	if l2.String() != "-1km" {
		t.Errorf(`l2 should be equal to -1km but l2 = %v (%016x)`, l2, uint64(l2))
	}

	// 1km = 100000cm
	l3 := l1.Add(l2.Abs())
	if l3.String() != "100000cm" {
		t.Errorf(`l3 should be equal to 100000cm (1km) but l3 = %v (%016x)`, l3, uint64(l3))
	}

	l4 := l2.Add(l3)
	if l4.String() != "0km" {
		t.Errorf(`l4 should be equal to 0km but l4 = %v (%016x)`, l4, uint64(l4))
	}

	_, err = NewLengthFromBytes([]byte("11mz"))
	if err == nil {
		t.Errorf(`11mz should have conversion error, error is not set`)
	}
}

func TestLengthAdd(t *testing.T) {
	l1, err := NewLengthFromString(".00123")
	if !l1.IsExact() {
		t.Errorf(`l1 should be exact but l1 = %v`, l1)
	}
	if err != nil {
		t.Errorf(`NewLengthFromString(".00123") has result = %v and error = %v`, l1, err)
	}

	ls, _ := NewLengthFromDecimal(1, "pm")
	l1 = l1.Add(ls.Div(10000000000000000))
	if l1.IsExact() {
		t.Errorf(`l1 should not be exact but l1 = %v`, l1)
	}

	l2, err := NewLengthFromString("101mm")
	if !l2.IsExact() {
		t.Errorf(`l2 should be exact but l2 = %v`, l2)
	}
	if err != nil {
		t.Errorf(`NewLengthFromString("101mm") has result = %v and error = %v`, l2, err)
	}

	l3 := l1.Add(l2)
	if l3.Unit() != "m" {
		t.Errorf(`l3 unit should be equal to m but l3 unit = %v`, l3.Unit())
	}

	l3 = l2.Add(l1)
	if l3.Unit() != "mm" {
		t.Errorf(`l3 unit should be equal to mm but l3 unit = %v`, l3.Unit())
	}

	l3 = l2.Sub(l1)
	if l3.Unit() != "mm" {
		t.Errorf(`l3 unit should be equal to mm but l3 unit = %v`, l3.Unit())
	}

	l4 := l3.Sub(l2)
	if l4.Unit() != "mm" {
		t.Errorf(`l4 unit should be equal to mm but l4 unit = %v`, l4.Unit())
	}
}

func TestLengthMul(t *testing.T) {
	l1, err := NewLengthFromString("11mm")
	if err != nil {
		t.Errorf(`NewLengthFromString("11mm") has result = %v and error = %v`, l1, err)
	}

	l2 := l1.Mul(11)
	if l2.Unit() != "mm" {
		t.Errorf(`l2 unit should be equal to mm but l2 unit = %v`, l2.Unit())
	}
	if l2.String() != "121mm" {
		t.Errorf(`l2 should be equal to 121mm but l2 = %v`, l2)
	}

	l3 := l2.Mul(100000000000000000).Mul(100000000000000000)
	if !l3.IsInfinite() {
		t.Errorf(`l3 should be infinite but l3 = %v`, l3)
	}
	if l3.String() != "+Inf" {
		t.Errorf(`l3 should be infinite but l3 = %v`, l3)
	}
}

func TestLengthDiv(t *testing.T) {
	l1, err := NewLengthFromString("121mm")
	if err != nil {
		t.Errorf(`NewLengthFromString("121mm") has result = %v and error = %v`, l1, err)
	}

	l2 := l1.Div(11)
	if l2.Unit() != "mm" {
		t.Errorf(`l2 unit should be equal to mm but l2 unit = %v`, l2.Unit())
	}
	if l2.String() != "11mm" {
		t.Errorf(`l2 should be equal to 11mm but l2 = %v`, l2)
	}

	l3 := l1.Div(3)
	if l3.String() != "~40.33333333333333mm" {
		t.Errorf(`l3 should be equal to ~40.33mm but l3 = %v`, l3)
	}

	l2 = l1.Mul(2).Div(3).Add(l3)
	if l2.String() != "~121mm" {
		t.Errorf(`l2 should be equal to ~121mm but l2 = %v`, l2)
	}

	l2 = l1.Div(0)
	if !l2.IsNaN() {
		t.Errorf(`l2 should be NaN but l2 = %v`, l2)
	}
}

func TestLengthJSONMarshaling(t *testing.T) {
	l, err := NewLengthFromString("11cm")
	if err != nil {
		t.Errorf(`NewLengthFromString("11cm") has result = %v and error = %v`, l, err)
	}

	if b, err := l.MarshalText(); err != nil {
		t.Errorf(`(%v).MarshalText() should be ok, error = %v`, l, err)
	} else if string(b) != `11cm` {
		t.Errorf(`(%v).MarshalText() should be '11cm', buff = '%s'`, l, b)
	}

	if b, err := l.MarshalJSON(); err != nil {
		t.Errorf(`(%v).MarshalJSON() should be ok, error = %v`, l, err)
	} else if string(b) != `11cm` {
		t.Errorf(`(%v).MarshalJSON() should be '11cm', buff = '%s'`, l, b)
	}

	l1, _ := NewLengthFromString("42mm")
	for _, b := range []string{`42mm`, `"42mm"`} {
		if err := l.UnmarshalText([]byte(b)); err != nil {
			t.Errorf(`().UnmarshalText(%s) should be ok, error = %v`, b, err)
		} else if l != l1 {
			t.Errorf(`().UnmarshalText(%s) should be 42mm, buff = '%s'`, b, l)
		}

		if err := l.UnmarshalJSON([]byte(b)); err != nil {
			t.Errorf(`().UnmarshalJSON(%s) should be ok, error = %v`, b, err)
		} else if l != l1 {
			t.Errorf(`().UnmarshalJSON(%s) should be 42mm, buff = '%s'`, b, l)
		}
	}
}

func TestLengthNull(t *testing.T) {
	var null Length
	l0, _ := NewLength(0, 0, "m")
	l1, _ := NewLengthFromDecimal(1, "mm")

	if !null.IsNull() {
		t.Error("Null should be Null")
	}
	if l0.IsNull() {
		t.Error("Zero should not be Null")
	}

	if null.IsSet() {
		t.Error("Null should not be Set")
	}
	if !l0.IsSet() {
		t.Error("Zero should be Set")
	}

	if null.IfNull(l0) != l0 {
		t.Error("IfNull should return default for Null")
	}
	if l1.IfNull(l0) != l1 {
		t.Error("IfNull should return value for non-Null")
	}
}

func TestLengthZero(t *testing.T) {
	var null Length
	l0, _ := NewLength(0, 0, "m")
	l1, _ := NewLengthFromDecimal(1, "mm")
	lNeg1, _ := NewLengthFromDecimal(-1, "m")

	if !l0.IsExactlyZero() {
		t.Error("Zero should be ExactlyZero")
	}
	if !null.IsExactlyZero() {
		t.Error("Null should be ExactlyZero")
	}
	if l1.IsExactlyZero() {
		t.Error("1mm should not be ExactlyZero")
	}
	if lNeg1.IsExactlyZero() {
		t.Error("-1m should not be ExactlyZero")
	}

	if !l0.IsZero() {
		t.Error("Zero should be Zero")
	}
	if !null.IsZero() {
		t.Error("Null should be Zero")
	}
	if l1.IsZero() {
		t.Error("1mm should not be Zero")
	}
	if lNeg1.IsZero() {
		t.Error("-1m should not be Zero")
	}
}

func TestLengthSign(t *testing.T) {
	var null Length
	l0, _ := NewLength(0, 0, "m")
	l1, _ := NewLengthFromDecimal(1, "mm")
	lNeg1, _ := NewLengthFromDecimal(-1, "m")

	if null.Sign() != 0 {
		t.Error("Null should have sign 0")
	}
	if l0.Sign() != 0 {
		t.Error("Zero should have sign 0")
	}
	if l1.Sign() != 1 {
		t.Error("1mm should have sign 1")
	}
	if lNeg1.Sign() != -1 {
		t.Error("-1m should have sign -1")
	}

	if !l1.IsPositive() {
		t.Error("1mm should be Positive")
	}
	if lNeg1.IsPositive() {
		t.Error("-1m should not be Positive")
	}
	if l0.IsPositive() {
		t.Error("Zero should not be Positive")
	}

	if !lNeg1.IsNegative() {
		t.Error("-1m should be Negative")
	}
	if l1.IsNegative() {
		t.Error("1mm should not be Negative")
	}
	if l0.IsNegative() {
		t.Error("Zero should not be Negative")
	}
}

func TestLengthCompare(t *testing.T) {
	l1m, _ := NewLengthFromString("1m")
	l100cm, _ := NewLengthFromString("100cm")
	l2m, _ := NewLengthFromString("2m")
	l50cm, _ := NewLengthFromString("50cm")

	if l1m.Compare(l100cm) != 0 {
		t.Errorf("1m should equal 100cm, got %d", l1m.Compare(l100cm))
	}
	if l1m.Compare(l2m) >= 0 {
		t.Error("1m should be less than 2m")
	}
	if l2m.Compare(l1m) <= 0 {
		t.Error("2m should be greater than 1m")
	}

	if !l2m.GreaterThan(l1m) {
		t.Error("2m should be greater than 1m")
	}
	if l1m.GreaterThan(l2m) {
		t.Error("1m should not be greater than 2m")
	}

	if !l1m.GreaterThanOrEqual(l100cm) {
		t.Error("1m should be greater than 100cm")
	}
	if !l2m.GreaterThanOrEqual(l1m) {
		t.Error("2m should be greater than or equal to 1m")
	}

	if !l50cm.LessThan(l1m) {
		t.Error("50cm should be less than 1m")
	}

	if !l50cm.LessThanOrEqual(l1m) {
		t.Error("50cm should be less than or equal to 1m")
	}
	if !l1m.LessThanOrEqual(l100cm) {
		t.Error("1m should be less than or equal to 100cm")
	}
}

func TestNewLengthPositive(t *testing.T) {
	// NewLength with strictly positive value goes through the v=0 branch
	l, err := NewLength(101, 0, "m")
	if err != nil {
		t.Errorf(`NewLength(101, 0, "m") should not error, got %v`, err)
	}
	if l.String() != "101m" {
		t.Errorf(`NewLength(101, 0, "m") should be 101m, got %v`, l)
	}
}

func TestLengthVmetMagic(t *testing.T) {
	// underflow → encoded as ±~0 (m=0, e=lengthMinE) — exercises vmet's e == lengthMinE branch
	l, err := NewLengthFromString("1e-50m")
	if err != nil {
		t.Errorf(`NewLengthFromString("1e-50m") should not error, got %v`, err)
	}
	if l.IsExact() {
		t.Errorf(`underflowed length should have loss bit set, got %v`, l)
	}
	_ = l.String()

	// overflow → encoded as +Inf (m=0, e=lengthMaxE) — exercises vmet's e == lengthMaxE branch
	l, err = NewLengthFromString("1e50m")
	if err != nil {
		t.Errorf(`NewLengthFromString("1e50m") should not error, got %v`, err)
	}
	if !l.IsInfinite() {
		t.Errorf(`overflowed length should be infinite, got %v`, l)
	}
}

func TestLengthAddNullPair(t *testing.T) {
	// (Null) + (Null) returns Null via vmeAsLength's `v == 0 && e == 0 → return Null` short-circuit
	var n1, n2 Length
	if r := n1.Add(n2); r != Null {
		t.Errorf(`Null + Null should be Null, got %v`, r)
	}
}

func TestLengthUnmarshalErrors(t *testing.T) {
	var l Length
	if err := l.UnmarshalJSON([]byte("not-a-length")); err == nil {
		t.Errorf(`UnmarshalJSON("not-a-length") should error`)
	}
	if err := l.UnmarshalText([]byte("not-a-length")); err == nil {
		t.Errorf(`UnmarshalText("not-a-length") should error`)
	}
}

func TestLengthAddImperial(t *testing.T) {
	// adding an imperial unit (in/ft/yd/mi) to a SI length forces the non-integer t.c branch in Add
	lin, _ := NewLengthFromString("12in")
	lft, _ := NewLengthFromString("1ft")
	lm, _ := NewLengthFromString("1m")

	// 12in + 1ft = 24in (1ft = 12in exactly)
	r := lin.Add(lft)
	if r.Unit() != "in" || r.String() != "24in" {
		t.Errorf(`12in + 1ft should be 24in, got %v (unit=%q)`, r, r.Unit())
	}

	// 1ft + 12in = 2ft
	r = lft.Add(lin)
	if r.Unit() != "ft" || r.String() != "2ft" {
		t.Errorf(`1ft + 12in should be 2ft, got %v (unit=%q)`, r, r.Unit())
	}

	// 1m + 1ft = 1.3048m exactly (NIST: 1ft = 0.3048m)
	r = lm.Add(lft)
	if r.Unit() != "m" || r.String() != "1.3048m" {
		t.Errorf(`1m + 1ft should be 1.3048m, got %v`, r)
	}

	// 1ft + 1m has loss bit set (1m = 3.28083... ft, non-terminating in decimal)
	r = lft.Add(lm)
	if r.Unit() != "ft" || r.IsExact() {
		t.Errorf(`1ft + 1m should be ft with loss bit, got %v (exact=%v)`, r, r.IsExact())
	}

	// 1ft + 2m specifically exercises the rounding-up branch (rem<<1) >= mc inside Add (rem=2192 with mc=3048)
	l2m, _ := NewLengthFromString("2m")
	r = lft.Add(l2m)
	if r.Unit() != "ft" || r.IsExact() {
		t.Errorf(`1ft + 2m should be ft with loss bit, got %v`, r)
	}

	// mile / yard
	lyd, _ := NewLengthFromString("1760yd")
	lmi, _ := NewLengthFromString("0mi")
	r = lmi.Add(lyd)
	if r.String() != "1mi" {
		t.Errorf(`0mi + 1760yd should be 1mi, got %v`, r)
	}
}

func TestLengthNaN(t *testing.T) {
	// 1m / 0 produces NaN
	l1, _ := NewLengthFromString("1m")
	nan := l1.Div(0)
	if !nan.IsNaN() {
		t.Errorf(`1m / 0 should be NaN, got %v`, nan)
	}
	if nan.IsInfinite() {
		t.Errorf(`NaN should not be infinite, got %v`, nan)
	}

	// non-NaN values
	if l1.IsNaN() {
		t.Errorf(`1m should not be NaN`)
	}
	var null Length
	if null.IsNaN() {
		t.Errorf(`Null should not be NaN`)
	}
}
