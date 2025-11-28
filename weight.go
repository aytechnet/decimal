package decimal

import (
	"math"
)

// Weight represents a fixed-point decimal hold as a 64 bits integer including unit among 14 possible.
// integer value between -9007199254740991 and 9007199254740991 (or WeightMaxInt) can safely be used as Weight using 'kg' unit, example :
//
//	var a Weight = 101 // a is a Weight of value 101Kg
//
// Note 0 is unitialized Weight and its value for calculation is 0.
// Note you need to use Weight method for calculation, you cannot use + - * / or any other operators unless Weight is a real non-zero integer value with 'kg' unit.
// Unitialized Weight is useful when using JSON marshaling/unmarshaling.
//
// Weight has similar 64 bits representation like Decimal except 4 bits are used to encode weight unit.
// Weight mantissa has 53 bits instead of Decimal mantissa of 57 bits.
type Weight int64

const (
	// WeightMaxInt constant is the maximal int64 value that can be safely saved as Weight with exponent still 0.
	// WeightMaxInt is as well the maximum value of mantissa of Weight and the bitmask to extract mantissa value of a Weight.
	WeightMaxInt = 0x001fffffffffffff

	weightMinE     = -16
	weightMaxE     = 15
	weightBitE     = 57
	weightEBitmask = 0x3e00000000000000
	weightBitT     = 53
	weightTBitmask = 0x01e0000000000000
)

var (
	weightUnits = [...]unit{
		// International System of Units where 'kg' is the base unit
		{u: "kg", c: 0, v: 0},
		{u: "t", c: 3, v: 1 << weightBitT},
		{u: "kt", c: 6, v: 2 << weightBitT},
		{u: "Mt", c: 9, v: 3 << weightBitT},
		{u: "Gt", c: 12, v: 4 << weightBitT},
		{u: "g", c: -3, v: 5 << weightBitT},
		{u: "mg", c: -6, v: 6 << weightBitT},
		{u: "µg", c: -9, v: 7 << weightBitT},
		{u: "ng", c: -12, v: 8 << weightBitT},
		{u: "pg", c: -15, v: 9 << weightBitT},

		{}, // 10 is reserved for future use
		{}, // 11 is reserved for future use

		// International avoirdupois and troy
		{u: "lb", c: 45359237 + 24<<decimalBitE /* 0.45359237 kg */, v: 12 << weightBitT},
		{u: "oz", c: 28349523125 + 20<<decimalBitE /* 0.028349523125 kg */, v: 13 << weightBitT},
		{u: " lb t", c: 3732417216 + 22<<decimalBitE /* 0.3732417216 kg */, v: 14 << weightBitT},
		{u: " oz t", c: 311034768 + 22<<decimalBitE /* 0.0311034768 kg */, v: 15 << weightBitT},

		// aliases
		{u: "mcg", c: -9, v: 7 << weightBitT},
		{u: " lb av", c: 45359237 + 24<<decimalBitE /* 0.45359237 kg */, v: 12 << weightBitT},
		{u: " oz av", c: 28349523125 + 20<<decimalBitE /* 0.028349523125 kg */, v: 13 << weightBitT},
	}
)

// internal function to extract decimal into VME tuple : Value of sign, loss and possibly type, Mantissa and Exponent
func (w Weight) vmet() (v, m uint64, e int64, t *unit) {
	var u uint64

	if w < 0 {
		u = uint64(-w)
		v = (u & loss) | sign
	} else {
		u = uint64(w)
		v = u & loss
	}

	e = int64((u&weightEBitmask)<<2) >> (2 + weightBitE) // e is now fully signed exponent

	m = u & WeightMaxInt

	t = &weightUnits[(u&weightTBitmask)>>weightBitT]
	v |= u & weightTBitmask // v keep unit

	// take care of special number
	if m == 0 {
		if e == weightMinE {
			e = math.MinInt64
		} else if e == weightMaxE {
			e = math.MaxInt64
		}
	}

	return
}

// internal function to define a decimal from a VME tuple : Value of sign, loss and possibly type, Mantissa and Exponent
func vmeAsWeight(v, m uint64, e int64) Weight {
	// handle special case for null and zero
	if m == 0 && v&loss == 0 {
		if v == 0 && e == 0 {
			return Null
		} else {
			if v&weightTBitmask == 0 {
				return Weight(math.MinInt64)
			} else {
				return Weight(v & weightTBitmask)
			}
		}
	} else {
		// FIXME: vmeNormalize does not try to change unit
		v, m, e = vmeNormalize(v, m, e, WeightMaxInt, weightMinE, weightMaxE)

		// FIXME: out-of-range cannot occurs as normalization has been done
		v |= m | uint64(e<<weightBitE)&weightEBitmask

		if v&sign != 0 {
			return -Weight(v ^ sign)
		} else {
			return Weight(v)
		}
	}
}

// NewWeight returns a new fixed-point decimal weight, value * 10 ^ exp using unit.
func NewWeight(value int64, exp int32, unit string) (w Weight, err error) {
	var v, m uint64
	var e int64

	if value <= 0 {
		v, m, e = sign, uint64(-value), int64(exp)
	} else {
		v, m, e = 0, uint64(value), int64(exp)
	}

	v, m, e, err = vmeUnitOrMagicFromBytes([]byte(unit), v, m, e, weightUnits[:])
	w = vmeAsWeight(v, m, e)

	return
}

// NewWeightFromDecimal converts a Decimal to Weight using unit.
func NewWeightFromDecimal(value Decimal, unit string) (w Weight, err error) {
	v, m, e := value.vme()

	v, m, e, err = vmeUnitOrMagicFromBytes([]byte(unit), v, m, e, weightUnits[:])
	w = vmeAsWeight(v, m, e)

	return
}

// NewWeightFromBytes returns a new Weight from a slice of bytes representation.
//
// If no weight unit is given, 'kg' is assumed.
func NewWeightFromBytes(value []byte) (Weight, error) {
	if v, m, e, err := vmeFromBytes(value, weightUnits[:]); err == nil {
		return vmeAsWeight(v, m, e), nil
	} else {
		return 0, err
	}
}

// NewWeightFromString returns a new Weight from a string representation.
//
// If no weight unit is given, 'kg' is assumed.
//
// Example:
//
//	w, err := NewFromString("-123.45")
//	w2, err := NewFromString(".0001g")
//	w3, err := NewFromString("1.47000mg")
//	w4, err := NewFromString("3.14e15 t")
func NewWeightFromString(value string) (Weight, error) {
	return NewWeightFromBytes([]byte(value))
}

// Unit returns unit string of w.
//
// Example:
//
//	w1, err := NewWeightFromString("100g")
//	println(w1.Unit())
//
// Output:
//
//	g
func (w Weight) Unit() string {
	var u uint64

	if w < 0 {
		u = uint64(-w)
	} else {
		u = uint64(w)
	}

	return weightUnits[(u&weightTBitmask)>>weightBitT].u
}

// Add returns w1 + w2 using w1 unit.
//
// Example:
//
//	w1, err := NewWeightFromString("123.45kg")
//	w2, err := NewWeightFromString("550g")
//	w3 := w1.Add(w2)
//	println(w1.Add(w2))
//	println(w2.Add(w1))
//
// Output:
//
//	124kg
//	124000g
func (w1 Weight) Add(w2 Weight) Weight {
	v1, m1, e1, t1 := w1.vmet()
	v2, m2, e2, t2 := w2.vmet()

	if t2.c.IsInteger() {
		e2 += t2.c.Int64()
	} else {
		vc, mc, ec := t2.c.vme()
		v2, m2, e2 = vmeMul(v2, m2, e2, vc, mc, ec)
	}
	if t1.c.IsInteger() {
		e2 -= t1.c.Int64()
	} else {
		vc, mc, ec := t1.c.vme()

		var rem uint64
		v2, m2, e2, rem, _ = vmeDivRem(v2, m2, e2, vc, mc, ec, int32(DivisionPrecision))

		if rem != 0 {
			v2 |= loss

			// FIXME: fix m so that the result is the nearest, like shopspring/decimal
			if (rem << 1) >= mc {
				m2++
			}
		}
	}

	v, m, e := vmeAdd(v1, m1, e1, v2, m2, e2)

	return vmeAsWeight(v, m, e)
}

// Sub returns w1 - w2 using w1 unit.
func (w1 Weight) Sub(w2 Weight) Weight {
	return w1.Add(-w2)
}

// Mul returns w * d using w unit.
func (w Weight) Mul(d Decimal) Weight {
	v1, m1, e1, _ := w.vmet()
	v2, m2, e2 := d.vme()

	return vmeAsWeight(vmeMul(v1, m1, e1, v2, m2, e2))
}

// String returns the string representation of the weight with the fixed point and unit.
//
// Example:
//
//	d := NewWeight(-12345, -3, "kg")
//	println(d.String())
//
// Output:
//
//	-12.345kg
func (w Weight) String() string {
	if w == Null {
		return "0"
	} else {
		return string(w.Bytes())
	}
}

// Bytes returns the string representation of the decimal as a slice of byte, but nil if the decimal is Null.
func (w Weight) Bytes() (b []byte) {
	if w == Null {
		return nil
	} else {
		v, m, e, t := w.vmet()

		// the maximal length of decimal representation in bytes in such conditions is 20
		return vmetBytes(make([]byte, 0, 22), v, m, e, 0, t, true, false)
	}
}

// MarshalJSON implements the json.Marshaler interface.
func (w Weight) MarshalJSON() ([]byte, error) {
	v, m, e, t := w.vmet()

	return vmetBytes(nil, v, m, e, 0, t, false, false), nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
func (w *Weight) UnmarshalJSON(b []byte) error {
	if v, m, e, err := vmeFromBytes(b, weightUnits[:]); err == nil {
		*w = vmeAsWeight(v, m, e)

		return nil
	} else {
		return err
	}
}

// UnmarshalText implements the encoding.TextUnmarshaler interface for XML deserialization.
func (w *Weight) UnmarshalText(text []byte) error {
	if _w, err := NewWeightFromBytes(text); err != nil {
		return err
	} else {
		*w = _w

		return nil
	}
}

// MarshalText implements the encoding.TextMarshaler interface for XML serialization.
func (w Weight) MarshalText() (text []byte, err error) {
	return w.Bytes(), nil
}

// IsNull return
//
//	true if w == Null
//	false if w == 0
//	false if w == ~0 or w == -~0 or w == +~0
//	false if w < 0
//	false if w > 0
func (w Weight) IsNull() bool {
	return w == Null
}

// IfNull return
//
//	defaultValue if w == Null
//	w in any other cases
func (w Weight) IfNull(defaultValue Weight) Weight {
	if w == Null {
		return defaultValue
	} else {
		return w
	}
}

// IsSet return
//
//	false if w == Null
//	true if w == 0
//	true if w == ~0 or w == -~0 or w == +~0
//	true if w < 0
//	true if w > 0
func (w Weight) IsSet() bool {
	return w != Null
}

// IsExactlyZero return
//
//	true if w == Null or w == Zero
//	false if w == ~0 or w == -~0 or w == +~0
//	false if w < 0
//	false if w > 0
func (w Weight) IsExactlyZero() bool {
	return ^uint64(sign|weightTBitmask)&uint64(w) == 0 // w == Null || w == Zero (ignoring unit)
}

// IsZero return
//
//	true if w == Null or w == Zero
//	true if w == ~0 or w == -~0 or w == +~0
//	false if w < 0
//	false if w > 0
func (w Weight) IsZero() bool {
	// Check for NearZero equivalents in Weight context if they exist.
	// Assuming Weight uses same loss bit mechanism.
	// NearZero is Zero | loss.
	return w.IsExactlyZero() || Weight(uint64(w)&^sign&^weightTBitmask) == Weight(loss)
}

// IsExact return true if a weight has its loss bit not set, ie it has not lost its precision during computation or conversion.
func (w Weight) IsExact() bool {
	return uint64(w)&loss == 0
}

// IsPositive return
//
//	true if w > 0 or w == ~+0
//	false if w == Null or w == Zero or w == ~0
//	false if w < 0 or w == ~-0
//	false if w is NaN
func (w Weight) IsPositive() bool {
	return w > 0 && !w.IsNaN()
}

// IsNegative return
//
//	true if w < 0 or w == ~-0
//	false if w == Null or w == Zero or w == ~0
//	false if w > 0
func (w Weight) IsNegative() bool {
	return !w.IsZero() && w < 0
}

// IsInfinite return
//
//	true if a w == +Inf or w == -Inf
//	false in any other case
func (w Weight) IsInfinite() bool {
	// Check exponent for max value
	_, _, e, _ := w.vmet()
	return e == math.MaxInt64
}

// IsNaN return
//
//	true if w is not a a number (NaN)
//	false in any other case
func (w Weight) IsNaN() bool {
	// Check if exponent is special (NaN range)
	// Weight has 53 bits mantissa, 4 bits unit, 57 bits total for value part?
	// Actually vmet() extracts e.
	// Let's use vmet() to check for NaN condition which is usually e=1 and v=loss?
	// Or check raw bits like Decimal.IsNaN.

	// Decimal IsNaN checks:
	// u >= 0x42 && u < 0x5c || u >= 0x62 && u <= 0x7e (after shifting)

	// Weight layout:
	// e = int64((u&weightEBitmask)<<2) >> (2 + weightBitE)
	// weightBitE = 57

	// Let's rely on checking if it's not a valid number via properties if possible,
	// or replicate bit check.
	// Simpler: check if e is in NaN range?
	// In core.go, NaN has e=1, v=loss.

	v, m, e, _ := w.vmet()
	if m == 0 && v&loss != 0 {
		if e != 0 && e != math.MinInt64 && e != math.MaxInt64 {
			return true
		}
	}
	return false
}

// Sign return
//
//	0 if w == Null or w == Zero or w == ~0
//	1 if w > 0 or w == ~+0
//	-1 if w < 0 or w == ~-0
//	undefined (1 or -1) if w is NaN
func (w Weight) Sign() int {
	if w.IsExactlyZero() || w.IsZero() {
		return 0
	} else {
		return 1 - (int(uint64(w)>>63) << 1)
	}
}

// Compare compares the numbers represented by w1 and w2 without taking into account lost precision and returns:
//
//	-1 if w1 <  w2
//	 0 if w1 == w2
//	+1 if w1 >  w2
func (w1 Weight) Compare(w2 Weight) int {
	w := w1.Sub(w2)

	if w.IsZero() {
		return 0
	} else if w.IsPositive() {
		return 1
	} else {
		return -1
	}
}

// GreaterThan returns true when w1 is greater than w2 (w1 > w2).
func (w1 Weight) GreaterThan(w2 Weight) bool {
	w := w1.Sub(w2)

	return w.IsPositive()
}

// GreaterThanOrEqual returns true when w1 is greater than or equal to w2 (w1 >= w2).
func (w1 Weight) GreaterThanOrEqual(w2 Weight) bool {
	w := w1.Sub(w2)

	return w.IsPositive() || w.IsZero()
}

// LessThan returns true when w1 is less than w2 (w1 < w2).
func (w1 Weight) LessThan(w2 Weight) bool {
	return w2.GreaterThan(w1)
}

// LessThanOrEqual returns true when w1 is less than or equal to w2 (w1 <= w2).
func (w1 Weight) LessThanOrEqual(w2 Weight) bool {
	return w2.GreaterThanOrEqual(w1)
}
