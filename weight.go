package decimal

import (
	"math"
)

type Weight int64

const (
	// MaxInt constant is the maximal int64 value that can be safely saved as Decimal with exponent still 0.
	// MaxInt is as well the maximum value of mantissa of Decimal and the bitmask to extract mantissa value of a Decimal.
	WeightMaxInt = 0x001fffffffffffff

	// Zero constant is not empty zero Decimal value.
	// A decimal can be directly initialized with Zero and it will be not empty, but zero.
	// Zero can be safely compared with == or != directly to check for not empty zero decimal value.
	// But you need to use d.IsExactlyZero() to test if decimal d is zero or null.
	WeightZero Weight = math.MinInt64

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
		unit{ u: "kg", c: 0,   v: 0, },
		unit{ u: "t",  c: 3,   v: 1 << weight_bit_t, },
		unit{ u: "kt", c: 6,   v: 2 << weight_bit_t, },
		unit{ u: "Mt", c: 9,   v: 3 << weight_bit_t, },
		unit{ u: "Gt", c: 12,  v: 4 << weight_bit_t, },
		unit{ u: "g",  c: -3,  v: 5 << weight_bit_t, },
		unit{ u: "mg", c: -6,  v: 6 << weight_bit_t, },
		unit{ u: "µg", c: -9,  v: 7 << weight_bit_t, },
		unit{ u: "ng", c: -12, v: 8 << weight_bit_t, },
		unit{ u: "pg", c: -15, v: 9 << weight_bit_t, },

		unit{}, // 10 is reserved for future use
		unit{}, // 11 is reserved for future use

		// International avoirdupois and troy
		unit{ u: "lb",     c: 45359237 + 24 << decimal_bit_e    /* 0.45359237 kg */,     v: 12 << weight_bit_t, },
		unit{ u: "oz",     c: 28349523125 + 20 << decimal_bit_e /* 0.028349523125 kg */, v: 13 << weight_bit_t, },
		unit{ u: " lb t",  c: 3732417216 + 22 << decimal_bit_e  /* 0.3732417216 kg */,   v: 14 << weight_bit_t, },
		unit{ u: " oz t",  c: 311034768 + 22 << decimal_bit_e   /* 0.0311034768 kg */,   v: 15 << weight_bit_t, },

		// aliases
		unit{ u: "mcg",    c: -9,                                                        v: 7 << weight_bit_t, },
		unit{ u: " lb av", c: 45359237 + 24 << decimal_bit_e    /* 0.45359237 kg */,     v: 12 << weight_bit_t, },
		unit{ u: " oz av", c: 28349523125 + 20 << decimal_bit_e /* 0.028349523125 kg */, v: 13 << weight_bit_t, },
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

	t = &weight_units[(u&weight_t_bitmask) >> weight_bit_t]
	v |= u&weight_t_bitmask // v keep unit

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

	return weight_units[(u&weight_t_bitmask) >> weight_bit_t].u
}

// Add returns w1 + w2 using w1 unit.
//
// Example:
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

	// check if both c of unit t1 and t2 are integers, meaning a conversion applied only to exponent
	if t1.c.IsInteger() && t2.c.IsInteger() {
		e2 += t2.c.Int64() - t1.c.Int64()

		v, m, e := vmeAdd(v1, m1, e1, v2, m2, e2)

		return vmeAsWeight(v, m, e)
	} else {
		if t2.c.IsInteger() {
			e2 += t2.c.Int64()
		} else {
			vc, mc, ec := t2.c.vme()
			v2, m2, e2 = vmeMul(v2, m2, e2, vc, mc, ec)
		}
		if t1.c.IsInteger() {
			e2 -= t1.c.Int64()
		} else {
			vc, mc, ec := t2.c.vme()

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
	//return vmeAsWeight(vmeAdd(v1, m1, e1, v2, m2, e2))
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
