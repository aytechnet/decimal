package decimal

import (
	"testing"
)

func TestWeightConversions(t *testing.T) {
	var w0 Weight

	if w0.String() != "0" {
		t.Errorf(`w0.String() should be equal to 0 but w0 = %v`, w0)
	}
	if w0.Bytes() != nil {
		t.Errorf(`w0.Bytes() should be equal to nil but w0 = %v`, w0.Bytes())
	}

	w1, err := NewWeightFromString("1mcg")
	if err != nil {
		t.Errorf(`NewWeightFromString("1mcg") has result = %v and error = %v`, w1, err)
	}
	if w1.String() != "1µg" {
		t.Errorf(`w1 should be equal to 1µg but w1 = %v`, w1)
	}

	w1, err = NewWeightFromString("0g")
	if err != nil {
		t.Errorf(`NewWeightFromString("0g") has result = %v and error = %v`, w1, err)
	}
	if w1.String() != "0g" {
		t.Errorf(`w1 should be equal to 0g but w1 = %v (%016x)`, w1, uint64(w1))
	}

	w2, err := NewWeightFromString("1ozt")
	if err != nil {
		t.Errorf(`NewWeightFromString("1ozt") has result = %v and error = %v`, w2, err)
	}
	if w2.String() != "1 oz t" {
		t.Errorf(`w2 should be equal to 1 oz t but w1 = %v (%016x)`, w2, uint64(w2))
	}

	w3 := w1.Add(w2)
	if w3.String() != "31.1034768g" {
		t.Errorf(`w3 should be equal to 31.1034768g (1 oz t) but w3 = %v (%016x)`, w3, uint64(w3))
	}

	w4 := w2.Sub(w3)
	if w4.String() != "0 oz t" {
		t.Errorf(`w4 should be equal to 0 oz t but w4 = %v (%016x)`, w4, uint64(w4))
	}

	w4, err = NewWeightFromString("11ozz")
	if err == nil {
		t.Errorf(`11ozz should have conversion error, error is not set`)
	}
}

func TestWeightAdd(t *testing.T) {
	w1, err := NewWeightFromString(".00123")
	if err != nil {
		t.Errorf(`NewWeightFromString(".00123") has result = %v and error = %v`, w1, err)
	}

	w2, err := NewWeightFromString("~101g")
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

func TestWeightHelpers(t *testing.T) {
	var null Weight
	zero, _ := NewWeight(0, 0, "kg")
	oneKg, _ := NewWeight(1, 0, "kg")
	negOneKg, _ := NewWeight(-1, 0, "kg")

	// IsNull
	if !null.IsNull() {
		t.Error("Null should be Null")
	}
	if zero.IsNull() {
		t.Error("Zero should not be Null")
	}

	// IsSet
	if null.IsSet() {
		t.Error("Null should not be Set")
	}
	if !zero.IsSet() {
		t.Error("Zero should be Set")
	}

	// IfNull
	if null.IfNull(zero) != zero {
		t.Error("IfNull should return default for Null")
	}
	if oneKg.IfNull(zero) != oneKg {
		t.Error("IfNull should return value for non-Null")
	}

	// IsExactlyZero
	if !zero.IsExactlyZero() {
		t.Error("Zero should be ExactlyZero")
	}
	if !null.IsExactlyZero() {
		t.Error("Null should be ExactlyZero")
	}
	if oneKg.IsExactlyZero() {
		t.Error("1kg should not be ExactlyZero")
	}

	// IsZero
	if !zero.IsZero() {
		t.Error("Zero should be Zero")
	}
	if !null.IsZero() {
		t.Error("Null should be Zero")
	}
	// Test NearZero logic if applicable to Weight (assuming similar bit structure)
	// ...

	// IsPositive
	if !oneKg.IsPositive() {
		t.Error("1kg should be Positive")
	}
	if negOneKg.IsPositive() {
		t.Error("-1kg should not be Positive")
	}
	if zero.IsPositive() {
		t.Error("Zero should not be Positive")
	}

	// IsNegative
	if !negOneKg.IsNegative() {
		t.Error("-1kg should be Negative")
	}
	if oneKg.IsNegative() {
		t.Error("1kg should not be Negative")
	}
	if zero.IsNegative() {
		t.Error("Zero should not be Negative")
	}

	// IsNaN
	// Construct a NaN weight manually if needed or use existing constant if compatible
	// Assuming NaN constant from decimal package works if casted, but Weight has different layout?
	// Weight uses 53 bits mantissa vs 57 for Decimal.
	// Let's verify IsNaN implementation details.
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
