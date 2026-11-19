package decimal

import (
	"encoding/json"
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
	} else if string(b) != `"11lb"` {
		t.Errorf(`(%v).MarshalJSON() should be '"11lb"', buff = '%s'`, w, b)
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

func TestNewWeightPositive(t *testing.T) {
	// NewWeight with strictly positive value goes through the v=0 branch (the negative branch is already covered by NewWeight(0,...))
	w, err := NewWeight(101, 0, "kg")
	if err != nil {
		t.Errorf(`NewWeight(101, 0, "kg") should not error, got %v`, err)
	}
	if w.String() != "101kg" {
		t.Errorf(`NewWeight(101, 0, "kg") should be 101kg, got %v`, w)
	}
}

func TestWeightVmetMagic(t *testing.T) {
	// underflow → encoded as ±~0 (m=0, e=weightMinE) — exercises vmet's e == weightMinE branch
	w, err := NewWeightFromString("1e-50kg")
	if err != nil {
		t.Errorf(`NewWeightFromString("1e-50kg") should not error, got %v`, err)
	}
	if w.IsExact() {
		t.Errorf(`underflowed weight should have loss bit set, got %v`, w)
	}
	// trigger vmet on it (via String) — this is what runs the `e == weightMinE` branch
	_ = w.String()

	// overflow → encoded as +Inf (m=0, e=weightMaxE) — exercises vmet's e == weightMaxE branch
	w, err = NewWeightFromString("1e50kg")
	if err != nil {
		t.Errorf(`NewWeightFromString("1e50kg") should not error, got %v`, err)
	}
	if !w.IsInfinite() {
		t.Errorf(`overflowed weight should be infinite, got %v`, w)
	}
}

func TestWeightAddNullPair(t *testing.T) {
	// (Null) + (Null) returns Null via vmeAsWeight's `v == 0 && e == 0 → return Null` short-circuit
	var n1, n2 Weight
	if r := n1.Add(n2); r != Null {
		t.Errorf(`Null + Null should be Null, got %v`, r)
	}
}

func TestWeightAddAvoirdupois(t *testing.T) {
	// addition between an avoirdupois unit (lb / oz) and a kg weight forces the non-integer t.c branch
	// (where t.c is itself a Decimal that is not an integer, exercising lines around the rem != 0 fix-up)
	wlb, _ := NewWeightFromString("1lb")
	wkg, _ := NewWeightFromString("1kg")

	r := wlb.Add(wkg)
	if r.IsNull() || r.Unit() != "lb" {
		t.Errorf(`1lb + 1kg should keep lb unit, got %v (unit=%q)`, r, r.Unit())
	}

	// also the symmetric direction
	r = wkg.Add(wlb)
	if r.IsNull() || r.Unit() != "kg" {
		t.Errorf(`1kg + 1lb should keep kg unit, got %v (unit=%q)`, r, r.Unit())
	}
}

func TestWeightUnmarshalErrors(t *testing.T) {
	// UnmarshalJSON with invalid input must return an error (covers the `else { return err }` branch)
	var w Weight
	if err := w.UnmarshalJSON([]byte("not-a-weight")); err == nil {
		t.Errorf(`UnmarshalJSON("not-a-weight") should error`)
	}
	if err := w.UnmarshalText([]byte("not-a-weight")); err == nil {
		t.Errorf(`UnmarshalText("not-a-weight") should error`)
	}
}

// TestWeightJSONEncoding goes through encoding/json, as a caller does: a Weight must be a valid JSON
// value in every unit and in every position (field, omitempty field, pointer, slice, map), and
// must read back equal. Calling MarshalJSON directly does not see an invalid output.
func TestWeightJSONEncoding(t *testing.T) {
	for _, tu := range weightUnits {
		for _, n := range []string{"0", "1", "-2.5", "0.001", "1234567.89"} {
			in := n + tu.u
			w, err := NewWeightFromString(in)
			if err != nil {
				t.Errorf(`NewWeightFromString(%q) error = %v`, in, err)
				continue
			}

			b, err := json.Marshal(w)
			if err != nil {
				t.Errorf(`json.Marshal(%v) error = %v`, w, err)
				continue
			}
			if !json.Valid(b) || len(b) < 2 || b[0] != '"' || b[len(b)-1] != '"' {
				t.Errorf(`json.Marshal(%v) should be a JSON string, buff = '%s'`, w, b)
			}
			if want := `"` + w.String() + `"`; string(b) != want {
				t.Errorf(`json.Marshal(%v) should be '%s', buff = '%s'`, w, want, b)
			}

			var back Weight
			if err := json.Unmarshal(b, &back); err != nil {
				t.Errorf(`json.Unmarshal(%s) error = %v`, b, err)
			} else if back != w {
				t.Errorf(`json.Unmarshal(%s) should be %v, got %v`, b, w, back)
			}
		}
	}

	type doc struct {
		Set  Weight            `json:"set"`
		Opt  Weight            `json:"opt,omitempty"`
		Ptr  *Weight           `json:"ptr"`
		List []Weight          `json:"list"`
		Map  map[string]Weight `json:"map"`
	}
	x, _ := NewWeightFromString("11lb")
	y, _ := NewWeightFromString("0.25")
	zero, _ := NewWeight(0, 0, "kg")

	in := doc{Set: x, Ptr: &y, List: []Weight{x, y, zero}, Map: map[string]Weight{"k": x}}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf(`json.Marshal(doc) error = %v`, err)
	}
	if want := `{"set":"11lb","ptr":"0.25kg","list":["11lb","0.25kg","0kg"],"map":{"k":"11lb"}}`; string(b) != want {
		t.Errorf("json.Marshal(doc) should be\n%s\ngot\n%s", want, b)
	}

	var out doc
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf(`json.Unmarshal(doc) error = %v`, err)
	}
	if out.Set != x || !out.Opt.IsNull() || out.Ptr == nil || *out.Ptr != y || len(out.List) != 3 || out.List[2] != zero || out.Map["k"] != x {
		t.Errorf(`doc should read back as written, got %+v`, out)
	}

	// a Null Weight is left out by omitempty, and is a valid JSON string without it
	if b, err := json.Marshal(struct {
		A Weight `json:"a"`
		B Weight `json:"b,omitempty"`
	}{}); err != nil || string(b) != `{"a":"0kg"}` {
		t.Errorf(`json.Marshal(null Weight) should be '{"a":"0kg"}', buff = '%s', error = %v`, b, err)
	}

	// infinity and not-a-number have no unit: "+Inf", "-Inf" and "NaN", read back as such
	inf, _ := NewWeightFromString("+Inf")
	ninf, _ := NewWeightFromString("-Infinity")
	nan := x.Div(Zero)
	if !inf.IsInfinite() || !inf.IsPositive() || !ninf.IsInfinite() || !ninf.IsNegative() || !nan.IsNaN() {
		t.Errorf(`+Inf, -Infinity and x/0 should be +Inf, -Inf and NaN, got %v, %v and %v`, inf, ninf, nan)
	}
	for want, magic := range map[string]Weight{`"+Inf"`: inf, `"-Inf"`: ninf, `"NaN"`: nan} {
		b, err := json.Marshal(magic)
		if err != nil || string(b) != want {
			t.Errorf(`json.Marshal(%v) should be '%s', buff = '%s', error = %v`, magic, want, b, err)
		}
		var back Weight
		if err := json.Unmarshal(b, &back); err != nil || back.IsNaN() != magic.IsNaN() || back.IsInfinite() != magic.IsInfinite() || back.Sign() != magic.Sign() {
			t.Errorf(`json.Unmarshal(%s) should be %v, got %v, error = %v`, b, magic, back, err)
		}
	}

	// what was accepted before is still read: a bare number is in the default unit
	for src, want := range map[string]string{`{"set":0.25}`: "0.25kg", `{"set":"0.25"}`: "0.25kg", `{"set":"250 lb"}`: "250lb", `{"set":null}`: "0kg"} {
		var d doc
		if err := json.Unmarshal([]byte(src), &d); err != nil || d.Set.String() != want {
			t.Errorf(`json.Unmarshal(%s) should give %s, got %v, error = %v`, src, want, d.Set, err)
		}
	}
}
