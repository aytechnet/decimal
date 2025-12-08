package decimal

import (
	"testing"
)

func TestWeightConversions(t *testing.T) {
	var w0 Weight

	if w0.String() != "0kg" {
		t.Errorf(`w0.String() should be equal to 0 but w0 = %v`, w0)
	}

	w1, err := NewWeightFromString("10mcg")
	if err != nil {
		t.Errorf(`NewWeightFromString("10mcg") has result = %v and error = %v`, w1, err)
	}
	if w1.String() != "10µg" {
		t.Errorf(`w1 should be equal to 10µg but w1 = %v`, w1)
	}

	w1, err = NewWeightFromString("0g")
	if err != nil {
		t.Errorf(`NewWeightFromString("0g") has result = %v and error = %v`, w1, err)
	}
	if w1.String() != "0g" {
		t.Errorf(`w1 should be equal to 0g but w1 = %v (%016x)`, w1, uint64(w1))
	}

	w2, err := NewWeightFromString("-1ozt")
	if err != nil {
		t.Errorf(`NewWeightFromString("-1ozt") has result = %v and error = %v`, w2, err)
	}
	if w2.String() != "-1 oz t" {
		t.Errorf(`w2 should be equal to -1 oz t but w1 = %v (%016x)`, w2, uint64(w2))
	}

	w3 := w1.Add(w2.Abs())
	if w3.String() != "31.1034768g" {
		t.Errorf(`w3 should be equal to 31.1034768g (1 oz t) but w3 = %v (%016x)`, w3, uint64(w3))
	}

	w4 := w2.Add(w3)
	if w4.String() != "0 oz t" {
		t.Errorf(`w4 should be equal to 0 oz t but w4 = %v (%016x)`, w4, uint64(w4))
	}

	_, err = NewWeightFromBytes([]byte("11ozz"))
	if err == nil {
		t.Errorf(`11ozz should have conversion error, error is not set`)
	}
}

func TestWeightAdd(t *testing.T) {
	w1, err := NewWeightFromString(".00123")
	if !w1.IsExact() {
		t.Errorf(`w1 should be exact but w1 = %v`, w1)
	}
	if err != nil {
		t.Errorf(`NewWeightFromString(".00123") has result = %v and error = %v`, w1, err)
	}

	ws, _ := NewWeightFromDecimal(1, "pg")
	w1 = w1.Add(ws.Div(10000000000000000))
	if w1.IsExact() {
		t.Errorf(`w1 should be exact but w1 = %v`, w1)
	}

	w2, err := NewWeightFromString("101g")
	if !w2.IsExact() {
		t.Errorf(`w2 should be exact but w2 = %v`, w2)
	}
	if err != nil {
		t.Errorf(`NewFromString("~101g") has result = %v and error = %v`, w2, err)
	}

	w3 := w1.Add(w2)
	if w3.Unit() != "kg" {
		t.Errorf(`w3 unit should be equal to kg but w3 unit = %v`, w3.Unit())
	}
	if w3.String() != "~0.10223kg" {
		t.Errorf(`w3 should be equal to ~0.10223kg but w3 = %v`, w3)
	}

	w3 = w2.Add(w1)
	if w3.Unit() != "g" {
		t.Errorf(`w3 unit should be equal to g but w3 unit = %v`, w3.Unit())
	}
	if w3.String() != "~102.23g" {
		t.Errorf(`w3 should be equal to ~102.23g but w3 = %v`, w3)
	}

	w3 = w2.Sub(w1)
	if w3.Unit() != "g" {
		t.Errorf(`w3 unit should be equal to g but w3 unit = %v`, w3.Unit())
	}
	if w3.String() != "~99.77g" {
		t.Errorf(`w3 should be equal to ~99.77g but w3 = %v`, w3)
	}

	w4 := w3.Sub(w2)
	if w4.Unit() != "g" {
		t.Errorf(`w4 unit should be equal to g but w4 unit = %v`, w4.Unit())
	}
	if w4.String() != "~-1.23g" {
		t.Errorf(`w4 should be equal to ~-1.23g but w4 = %v`, w4)
	}
}

func TestWeightMul(t *testing.T) {
	w1, err := NewWeightFromString("11mg")
	if err != nil {
		t.Errorf(`NewWeightFromString("11mg") has result = %v and error = %v`, w1, err)
	}

	w2 := w1.Mul(11)
	if w2.Unit() != "mg" {
		t.Errorf(`w2 unit should be equal to mg but w2 unit = %v`, w2.Unit())
	}
	if w2.String() != "121mg" {
		t.Errorf(`w2 should be equal to 121mg but w2 = %v`, w2)
	}

	w3 := w2.Mul(100000000000000000).Mul(100000000000000000)
	if !w3.IsInfinite() {
		t.Errorf(`w3 should be infinite but w3 = %v`, w3)
	}
	if w3.String() != "+Inf" {
		t.Errorf(`w3 should be infinite but w3 = %v`, w3)
	}
}

func TestWeightDiv(t *testing.T) {
	w1, err := NewWeightFromString("121mg")
	if err != nil {
		t.Errorf(`NewWeightFromString("121mg") has result = %v and error = %v`, w1, err)
	}

	w2 := w1.Div(11)
	if w2.Unit() != "mg" {
		t.Errorf(`w2 unit should be equal to mg but w2 unit = %v`, w2.Unit())
	}
	if w2.String() != "11mg" {
		t.Errorf(`w2 should be equal to 11mg but w2 = %v`, w2)
	}

	w3 := w1.Div(3)
	if w3.String() != "~40.33333333333333mg" {
		t.Errorf(`w3 should be equal to ~40.33mg but w3 = %v`, w3)
	}

	w2 = w1.Mul(2).Div(3).Add(w3)
	if w2.String() != "~121mg" {
		t.Errorf(`w2 should be equal to ~121mg but w2 = %v`, w2)
	}

	w2 = w1.Div(0)
	if !w2.IsNaN() {
		t.Errorf(`w2 should be NaN but w2 = %v`, w2)
	}
}

func TestWeightJSONMarshaling(t *testing.T) {
	w, err := NewWeightFromString("11lb")
	if err != nil {
		t.Errorf(`NewWeightFromString("11lb") has result = %v and error = %v`, w, err)
	}

	if b, err := w.MarshalText(); err != nil {
		t.Errorf(`(%v).MarshalText() should be ok, error = %v`, w, err)
	} else if string(b) != `11lb` {
		t.Errorf(`(%v).MarshalText() should be '11lb', buff = '%s'`, w, b)
	}

	if b, err := w.MarshalJSON(); err != nil {
		t.Errorf(`(%v).MarshalJSON() should be ok, error = %v`, w, err)
	} else if string(b) != `11lb` {
		t.Errorf(`(%v).MarshalJSON() should be '11lb', buff = '%s'`, w, b)
	}

	w1, _ := NewWeightFromString("42g")
	for _, b := range []string{`42g`, `"42g"`} {
		if err := w.UnmarshalText([]byte(b)); err != nil {
			t.Errorf(`().UnmarshalText(%s) should be ok, error = %v`, b, err)
		} else if w != w1 {
			t.Errorf(`().UnmarshalText(%s) should be '456.123', buff = '%s'`, b, w)
		}

		if err := w.UnmarshalJSON([]byte(b)); err != nil {
			t.Errorf(`().UnmarshalJSON(%s) should be ok, error = %v`, b, err)
		} else if w != w1 {
			t.Errorf(`().UnmarshalJSON(%s) should be '456.123', buff = '%s'`, b, w)
		}
	}
}

func TestWeighNull(t *testing.T) {
	var null Weight
	w0, _ := NewWeight(0, 0, "kg")
	w1, _ := NewWeightFromDecimal(1, "g")

	// IsNull
	if !null.IsNull() {
		t.Error("Null should be Null")
	}
	if w0.IsNull() {
		t.Error("Zero should not be Null")
	}

	// IsSet
	if null.IsSet() {
		t.Error("Null should not be Set")
	}
	if !w0.IsSet() {
		t.Error("Zero should be Set")
	}

	// IfNull
	if null.IfNull(w0) != w0 {
		t.Error("IfNull should return default for Null")
	}
	if w1.IfNull(w0) != w1 {
		t.Error("IfNull should return value for non-Null")
	}
}

func TestWeighZero(t *testing.T) {
	var null Weight
	w0, _ := NewWeight(0, 0, "kg")
	w1, _ := NewWeightFromDecimal(1, "g")
	wNeg1, _ := NewWeightFromDecimal(-1, "kg")

	// IsExactlyZero
	if !w0.IsExactlyZero() {
		t.Error("Zero should be ExactlyZero")
	}
	if !null.IsExactlyZero() {
		t.Error("Null should be ExactlyZero")
	}
	if w1.IsExactlyZero() {
		t.Error("1kg should not be ExactlyZero")
	}
	if wNeg1.IsExactlyZero() {
		t.Error("-1kg should not be ExactlyZero")
	}

	// IsZero
	if !w0.IsZero() {
		t.Error("Zero should be Zero")
	}
	if !null.IsZero() {
		t.Error("Null should be Zero")
	}
	if w1.IsZero() {
		t.Error("1kg should not be Zero")
	}
	if wNeg1.IsZero() {
		t.Error("-1kg should not be Zero")
	}
}

func TestWeighSign(t *testing.T) {
	var null Weight
	w0, _ := NewWeight(0, 0, "kg")
	w1, _ := NewWeightFromDecimal(1, "g")
	wNeg1, _ := NewWeightFromDecimal(-1, "kg")

	// Sign
	if null.Sign() != 0 {
		t.Error("Null should have sign 0")
	}
	if w0.Sign() != 0 {
		t.Error("Zero should have sign 0")
	}
	if w1.Sign() != 1 {
		t.Error("1kg should have sign 1")
	}
	if wNeg1.Sign() != -1 {
		t.Error("-1kg should have sign -1")
	}

	// IsPositive
	if !w1.IsPositive() {
		t.Error("1kg should be Positive")
	}
	if wNeg1.IsPositive() {
		t.Error("-1kg should not be Positive")
	}
	if w0.IsPositive() {
		t.Error("Zero should not be Positive")
	}

	// IsNegative
	if !wNeg1.IsNegative() {
		t.Error("-1kg should be Negative")
	}
	if w1.IsNegative() {
		t.Error("1kg should not be Negative")
	}
	if w0.IsNegative() {
		t.Error("Zero should not be Negative")
	}
}

func TestWeightCompare(t *testing.T) {
	w1kg, _ := NewWeightFromString("1kg")
	w1000g, _ := NewWeightFromString("1000g")
	w2kg, _ := NewWeightFromString("2kg")
	w500g, _ := NewWeightFromString("500g")

	// Compare
	if w1kg.Compare(w1000g) != 0 {
		t.Errorf("1kg should equal 1000g, got %d", w1kg.Compare(w1000g))
	}
	if w1kg.Compare(w2kg) >= 0 {
		t.Error("1kg should be less than 2kg")
	}
	if w2kg.Compare(w1kg) <= 0 {
		t.Error("2kg should be greater than 1kg")
	}

	// GreaterThan
	if !w2kg.GreaterThan(w1kg) {
		t.Error("2kg should be greater than 1kg")
	}
	if w1kg.GreaterThan(w2kg) {
		t.Error("1kg should not be greater than 2kg")
	}

	// GreaterThanOrEqual
	if !w1kg.GreaterThanOrEqual(w1000g) {
		t.Error("1kg should be greater than 1000g")
	}
	if !w2kg.GreaterThanOrEqual(w1kg) {
		t.Error("2kg should be greater than or equal to 1kg")
	}

	// LessThan
	if !w500g.LessThan(w1kg) {
		t.Error("500g should be less than 1kg")
	}

	// LessThanOrEqual
	if !w500g.LessThanOrEqual(w1kg) {
		t.Error("500g should be less than or equal to 1kg")
	}
	if !w1kg.LessThanOrEqual(w1000g) {
		t.Error("1kg should be less than or equal to 1000g")
	}
}
