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

	weight_min_e     = -16
	weight_max_e     = 15
	weight_bit_e     = 57
	weight_e_bitmask = 0x3e00000000000000
	weight_bit_t     = 53
	weight_t_bitmask = 0x01e0000000000000
)

var (
	weight_units = [...]unit{
		// International System of Units where 'kg' is the base unit
		{u: "kg", c: 0, v: 0},
		{u: "t", c: 3, v: 1 << weight_bit_t},
		{u: "kt", c: 6, v: 2 << weight_bit_t},
		{u: "Mt", c: 9, v: 3 << weight_bit_t},
		{u: "Gt", c: 12, v: 4 << weight_bit_t},
		{u: "g", c: -3, v: 5 << weight_bit_t},
		{u: "mg", c: -6, v: 6 << weight_bit_t},
		{u: "µg", c: -9, v: 7 << weight_bit_t},
		{u: "ng", c: -12, v: 8 << weight_bit_t},
		{u: "pg", c: -15, v: 9 << weight_bit_t},

		{}, // 10 is reserved for future use
		{}, // 11 is reserved for future use

		// International avoirdupois and troy
		{u: "lb", c: 45359237 + 24<<decimal_bit_e /* 0.45359237 kg */, v: 12 << weight_bit_t},
		{u: "oz", c: 28349523125 + 20<<decimal_bit_e /* 0.028349523125 kg */, v: 13 << weight_bit_t},
		{u: " lb t", c: 3732417216 + 22<<decimal_bit_e /* 0.3732417216 kg */, v: 14 << weight_bit_t},
		{u: " oz t", c: 311034768 + 22<<decimal_bit_e /* 0.0311034768 kg */, v: 15 << weight_bit_t},

		// aliases
		{u: "mcg", c: -9, v: 7 << weight_bit_t},
		{u: " lb av", c: 45359237 + 24<<decimal_bit_e /* 0.45359237 kg */, v: 12 << weight_bit_t},
		{u: " oz av", c: 28349523125 + 20<<decimal_bit_e /* 0.028349523125 kg */, v: 13 << weight_bit_t},
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

	e = int64((u&weight_e_bitmask)<<2) >> (2 + weight_bit_e) // e is now fully signed exponent

	m = u & WeightMaxInt

	t = &weight_units[(u&weight_t_bitmask)>>weight_bit_t]
	v |= u & weight_t_bitmask // v keep unit

	// take care of special number
	if m == 0 {
		if e == weight_min_e {
			e = math.MinInt64
		} else if e == weight_max_e {
			e = math.MaxInt64
		}
	}

	return
}

// internal function to define a decimal from a VME tuple : Value of sign, loss and possibly type, Mantissa and Exponent
func vmeAsWeight(v, m uint64, e int64) Weight {
	// FIXME: handle special case for null
	if v == 0 && m == 0 && e == 0 {
		return Null
	} else {
		// FIXME: vmeNormalize does not try to change unit
		v, m, e = vmeNormalize(v, m, e, WeightMaxInt, weight_min_e, weight_max_e)

		// FIXME: out-of-range cannot occurs as normalization has been done
		v |= m | uint64(e<<weight_bit_e)&weight_e_bitmask

		if v&sign != 0 {
			return -Weight(v ^ sign)
		} else {
			return Weight(v)
		}
	}
}

// NewWeightFromBytes returns a new Weight from a slice of bytes representation.
//
// If no weight unit is given, 'kg' is assumed.
func NewWeightFromBytes(value []byte) (Weight, error) {
	if v, m, e, err := vmeFromBytes(value, weight_units[:]); err == nil {
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

	return weight_units[(u&weight_t_bitmask)>>weight_bit_t].u
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
		return vmetBytes(make([]byte, 0, 22), v, m, e, t, true, false)
	}
}

// MarshalJSON implements the json.Marshaler interface.
func (w Weight) MarshalJSON() ([]byte, error) {
	v, m, e, t := w.vmet()

	return vmetBytes(nil, v, m, e, t, false, false), nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
func (w *Weight) UnmarshalJSON(b []byte) error {
	if v, m, e, err := vmeFromBytes(b, weight_units[:]); err == nil {
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
