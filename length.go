package decimal

import (
	"encoding/binary"
	"math"
)

// Length represents a fixed-point decimal hold as a 64 bits integer including unit among 7 possible.
// integer value between -9007199254740991 and 9007199254740991 (or LengthMaxInt) can safely be used as Length using 'm' unit, example :
//
//	var a Length = 101 // a is a Length of value 101m
//
// Note 0 is unitialized Length and its value for calculation is 0.
// Note you need to use Length method for calculation, you cannot use + - * / or any other operators unless Length is a real non-zero integer value with 'm' unit.
// Unitialized Length is useful when using JSON marshaling/unmarshaling.
//
// Length has similar 64 bits representation like Decimal except 4 bits are used to encode length unit.
// Length mantissa has 53 bits instead of Decimal mantissa of 57 bits.
type Length int64

const (
	// LengthMaxInt constant is the maximal int64 value that can be safely saved as Length with exponent still 0.
	// LengthMaxInt is as well the maximum value of mantissa of Length and the bitmask to extract mantissa value of a Length.
	LengthMaxInt = 0x001fffffffffffff

	lengthMinE     = -16
	lengthMaxE     = 15
	lengthBitE     = 57
	lengthEBitmask = 0x3e00000000000000
	lengthBitT     = 53
	lengthTBitmask = 0x01e0000000000000
)

var (
	lengthUnits = [...]unit{
		// International System of Units where 'm' is the base unit
		// Note: Mm, Gm, Tm are intentionally omitted because unitHash is case-insensitive and they would collide with mm
		{u: "m", c: 0, v: 0},
		{u: "km", c: 3, v: 1 << lengthBitT},
		{u: "dm", c: -1, v: 2 << lengthBitT},
		{u: "cm", c: -2, v: 3 << lengthBitT},
		{u: "mm", c: -3, v: 4 << lengthBitT},
		{u: "µm", c: -6, v: 5 << lengthBitT},
		{u: "nm", c: -9, v: 6 << lengthBitT},
		{u: "pm", c: -12, v: 7 << lengthBitT},

		{}, //  8 is reserved for future use
		{}, //  9 is reserved for future use
		{}, // 10 is reserved for future use

		// Unité Astronomique
		{u: "au", c: 1495978707 + 2<<decimalBitE /* 1.495978707x10^11 m */, v: 11 << lengthBitT},

		// International Yard and Pound (NIST 1959, exact)
		{u: "in", c: 254 + 28<<decimalBitE /* 0.0254 m */, v: 12 << lengthBitT},
		{u: "ft", c: 3048 + 28<<decimalBitE /* 0.3048 m */, v: 13 << lengthBitT},
		{u: "yd", c: 9144 + 28<<decimalBitE /* 0.9144 m */, v: 14 << lengthBitT},
		{u: "mi", c: 1609344 + 29<<decimalBitE /* 1609.344 m */, v: 15 << lengthBitT},

		// aliases
		{u: "um", c: -6, v: 5 << lengthBitT},
		{u: "ua", c: 1495978707 + 2<<decimalBitE /* 1.495978707x10^11 m */, v: 11 << lengthBitT},
	}
)

// internal function to extract decimal into VME tuple : Value of sign, loss and possibly type, Mantissa and Exponent
func (l Length) vmet() (v, m uint64, e int64, t *unit) {
	var u uint64

	if l < 0 {
		u = uint64(-l)
		v = (u & loss) | sign
	} else {
		u = uint64(l)
		v = u & loss
	}

	e = int64((u&lengthEBitmask)<<2) >> (2 + lengthBitE) // e is now fully signed exponent

	m = u & LengthMaxInt

	t = &lengthUnits[(u&lengthTBitmask)>>lengthBitT]
	v |= u & lengthTBitmask // v keep unit

	// take care of special number
	if m == 0 {
		if e == lengthMinE {
			e = math.MinInt64
		} else if e == lengthMaxE {
			e = math.MaxInt64
		}
	}

	return
}

// internal function to define a decimal from a VME tuple : Value of sign, loss and possibly type, Mantissa and Exponent
func vmeAsLength(v, m uint64, e int64) Length {
	// handle special case for null and zero
	if m == 0 && v&loss == 0 {
		if v == 0 && e == 0 {
			return Null
		} else {
			if v&lengthTBitmask == 0 {
				return Length(math.MinInt64)
			} else {
				return Length(v & lengthTBitmask)
			}
		}
	} else {
		// FIXME: vmeNormalize does not try to change unit
		v, m, e = vmeNormalize(v, m, e, LengthMaxInt, lengthMinE, lengthMaxE)

		// FIXME: out-of-range cannot occurs as normalization has been done
		v |= m | uint64(e<<lengthBitE)&lengthEBitmask

		if v&sign != 0 {
			return -Length(v ^ sign)
		} else {
			return Length(v)
		}
	}
}

// NewLength returns a new fixed-point decimal length, value * 10 ^ exp using unit.
func NewLength(value int64, exp int32, unit string) (l Length, err error) {
	var v, m uint64
	var e int64

	if value <= 0 {
		v, m, e = sign, uint64(-value), int64(exp)
	} else {
		v, m, e = 0, uint64(value), int64(exp)
	}

	v, m, e, err = vmeUnitOrMagicFromBytes([]byte(unit), v, m, e, lengthUnits[:])
	l = vmeAsLength(v, m, e)

	return
}

// NewLengthFromDecimal converts a Decimal to Length using unit.
func NewLengthFromDecimal(value Decimal, unit string) (l Length, err error) {
	v, m, e := value.vme()

	v, m, e, err = vmeUnitOrMagicFromBytes([]byte(unit), v, m, e, lengthUnits[:])
	l = vmeAsLength(v, m, e)

	return
}

// NewLengthFromBytes returns a new Length from a slice of bytes representation.
//
// If no length unit is given, 'm' is assumed.
func NewLengthFromBytes(value []byte) (Length, error) {
	if v, m, e, err := vmeFromBytes(value, lengthUnits[:]); err == nil {
		return vmeAsLength(v, m, e), nil
	} else {
		return 0, err
	}
}

// NewLengthFromString returns a new Length from a string representation.
//
// If no length unit is given, 'm' is assumed.
//
// Example:
//
//	l, err := NewLengthFromString("-123.45")
//	l2, err := NewLengthFromString(".0001m")
//	l3, err := NewLengthFromString("1.47000mm")
//	l4, err := NewLengthFromString("3.14e15 km")
func NewLengthFromString(value string) (Length, error) {
	return NewLengthFromBytes([]byte(value))
}

// Unit returns unit string of l.
//
// Example:
//
//	l1, err := NewLengthFromString("100cm")
//	println(l1.Unit())
//
// Output:
//
//	cm
func (l Length) Unit() string {
	var u uint64

	if l < 0 {
		u = uint64(-l)
	} else {
		u = uint64(l)
	}

	return lengthUnits[(u&lengthTBitmask)>>lengthBitT].u
}

// Abs returns the absolute value of the length.
func (l Length) Abs() Length {
	if l < 0 {
		return -l
	} else {
		return l
	}
}

// Add returns l1 + l2 using l1 unit.
//
// Example:
//
//	l1, err := NewLengthFromString("123.45km")
//	l2, err := NewLengthFromString("550m")
//	l3 := l1.Add(l2)
//	println(l1.Add(l2))
//	println(l2.Add(l1))
//
// Output:
//
//	124km
//	124000m
func (l1 Length) Add(l2 Length) Length {
	v1, m1, e1, t1 := l1.vmet()
	v2, m2, e2, t2 := l2.vmet()

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

	return vmeAsLength(v, m, e)
}

// Sub returns l1 - l2 using l1 unit.
func (l1 Length) Sub(l2 Length) Length {
	return l1.Add(-l2)
}

// Mul returns l * d using l unit.
func (l Length) Mul(d Decimal) Length {
	v1, m1, e1, _ := l.vmet()
	v2, m2, e2 := d.vme()

	return vmeAsLength(vmeMul(v1, m1, e1, v2, m2, e2))
}

// Div returns l / d using l unit. If it doesn't divide exactly, the result will have DivisionPrecision digits after the decimal point and loss bit will be set.
func (l Length) Div(d Decimal) Length {
	v1, m1, e1, _ := l.vmet()
	v2, m2, e2 := d.vme()

	v, m, e, rem, _ := vmeDivRem(v1, m1, e1, v2, m2, e2, int32(DivisionPrecision))

	if rem != 0 {
		v |= loss

		// fix m so that the result is the nearest, like in shopspring/decimal
		if (rem << 1) >= m2 {
			m++
		}
	}
	return vmeAsLength(v, m, e)
}

// String returns the string representation of the length with the fixed point and unit.
//
// Example:
//
//	l, err := NewLength(-12345, -3, "m")
//	println(l.String())
//
// Output:
//
//	-12.345m
func (l Length) String() string {
	return string(l.BytesTo(nil))
}

// BytesTo appends the string representation of the decimal to a slice of byte, if the decimal is Null it appends 0.
func (l Length) BytesTo(b []byte) []byte {
	v, m, e, t := l.vmet()

	// the maximal length of decimal representation in bytes in such conditions is 20
	return vmetBytesTo(b, v, m, e, 0, t, true, false)
}

// MarshalJSON implements the json.Marshaler interface.
func (l Length) MarshalJSON() ([]byte, error) {
	v, m, e, t := l.vmet()

	return vmetBytesTo(nil, v, m, e, 0, t, false, false), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (l *Length) UnmarshalJSON(b []byte) error {
	if v, m, e, err := vmeFromBytes(b, lengthUnits[:]); err == nil {
		*l = vmeAsLength(v, m, e)

		return nil
	} else {
		return err
	}
}

// UnmarshalText implements the encoding.TextUnmarshaler interface for XML deserialization.
func (l *Length) UnmarshalText(text []byte) error {
	if _l, err := NewLengthFromBytes(text); err != nil {
		return err
	} else {
		*l = _l

		return nil
	}
}

// MarshalText implements the encoding.TextMarshaler interface for XML serialization.
func (l Length) MarshalText() (text []byte, err error) {
	return l.BytesTo(nil), nil
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
//
// When the unit is m (the default unit code 0) the encoding is identical to a Decimal of the same
// scalar value (1-10 bytes). For any other unit the v2 Length extension format is used (see
// BINARY_FORMAT.md). Magic values (NaN, ±Inf, NearZero variants) always use the v1 magic byte and
// lose the unit info.
func (l Length) MarshalBinary() (data []byte, err error) {
	v, m, e, _ := l.vmet()
	unit := (v & lengthTBitmask) >> lengthBitT

	if m == 0 || unit == 0 {
		return marshalBinaryV1(v, m, e), nil
	}

	return marshalBinaryV2Ext(binExpLength, v, m, e, unit), nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
//
// Accepts the v1 format (assumed to be in m), the v2 Decimal extension (assumed to be in m),
// and the v2 Length extension (with explicit unit). A v2 Weight extension is rejected with ErrFormat.
func (l *Length) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		return ErrFormat
	}

	if data[0]&1 != 0 {
		u := uint64(data[0]) << (decimalBitE - 1)
		u ^= 1 << (decimalBitE - 1)
		m, n := binary.Uvarint(data[1:])
		if n <= 0 {
			return ErrFormat
		}
		u |= m
		if u&sign != 0 && Length(u) != Length(math.MinInt64) {
			*l = -Length(u ^ sign)
		} else {
			*l = Length(u)
		}
		return nil
	}

	if len(data) == 1 {
		u := uint64(data[0]) << (decimalBitE - 1)
		if u&sign != 0 && Length(u) != Length(math.MinInt64) {
			*l = -Length(u ^ sign)
		} else {
			*l = Length(u)
		}
		return nil
	}

	typeMarker, signNeg, negE, lossSet, ok := binDecodeOpcode(data[0])
	if !ok {
		return ErrFormat
	}

	rest := data[1:]
	var unit uint64

	switch typeMarker {
	case binExpDecimal:
		unit = 0
	case binExpLength:
		var n int
		unit, n = binary.Uvarint(rest)
		if n <= 0 {
			return ErrFormat
		}
		if unit >= uint64(len(lengthUnits)) || lengthUnits[unit].u == "" {
			return ErrUnitSyntax
		}
		rest = rest[n:]
	default:
		return ErrFormat
	}

	expAbs, nE := binary.Uvarint(rest)
	if nE <= 0 {
		return ErrFormat
	}
	rest = rest[nE:]

	mAbs, nM := binary.Uvarint(rest)
	if nM <= 0 {
		return ErrFormat
	}

	var v uint64
	if signNeg {
		v |= sign
	}
	if lossSet {
		v |= loss
	}
	v |= unit << lengthBitT

	e := int64(expAbs)
	if negE {
		e = -e
	}

	*l = vmeAsLength(v, mAbs, e)
	return nil
}

// IsNull return
//
//	true if l == Null
//	false if l == 0
//	false if l == ~0 or l == -~0 or l == +~0
//	false if l < 0
//	false if l > 0
func (l Length) IsNull() bool {
	return l == Null
}

// IfNull return
//
//	defaultValue if l == Null
//	l in any other cases
func (l Length) IfNull(defaultValue Length) Length {
	if l == Null {
		return defaultValue
	} else {
		return l
	}
}

// IsSet return
//
//	false if l == Null
//	true if l == 0
//	true if l == ~0 or l == -~0 or l == +~0
//	true if l < 0
//	true if l > 0
func (l Length) IsSet() bool {
	return l != Null
}

// IsExactlyZero return
//
//	true if l == Null or l == Zero
//	false if l == ~0 or l == -~0 or l == +~0
//	false if l < 0
//	false if l > 0
func (l Length) IsExactlyZero() bool {
	return ^uint64(sign|lengthTBitmask)&uint64(l) == 0 // l == Null || l == Zero (ignoring unit)
}

// IsZero return
//
//	true if l == Null or l == Zero
//	true if l == ~0 or l == -~0 or l == +~0
//	false if l < 0
//	false if l > 0
func (l Length) IsZero() bool {
	return l.IsExactlyZero() || Length(uint64(l)&^sign&^lengthTBitmask) == Length(loss)
}

// IsExact return true if a length has its loss bit not set, ie it has not lost its precision during computation or conversion.
func (l Length) IsExact() bool {
	return l.Abs()&loss == 0
}

// IsPositive return
//
//	true if l > 0 or l == ~+0
//	false if l == Null or l == Zero or l == ~0
//	false if l < 0 or l == ~-0
//	false if l is NaN
func (l Length) IsPositive() bool {
	return l > 0 && !l.IsNaN()
}

// IsNegative return
//
//	true if l < 0 or l == ~-0
//	false if l == Null or l == Zero or l == ~0
//	false if l > 0
func (l Length) IsNegative() bool {
	return !l.IsZero() && l < 0
}

// IsInfinite return
//
//	true if a l == +Inf or l == -Inf
//	false in any other case
func (l Length) IsInfinite() bool {
	_, _, e, _ := l.vmet()
	return e == math.MaxInt64
}

// IsNaN return
//
//	true if l is not a number (NaN)
//	false in any other case
func (l Length) IsNaN() bool {
	v, m, e, _ := l.vmet()
	if m == 0 && v&loss != 0 {
		if e != 0 && e != math.MinInt64 && e != math.MaxInt64 {
			return true
		}
	}
	return false
}

// Sign return
//
//	0 if l == Null or l == Zero or l == ~0
//	1 if l > 0 or l == ~+0
//	-1 if l < 0 or l == ~-0
//	undefined (1 or -1) if l is NaN
func (l Length) Sign() int {
	if l.IsExactlyZero() || l.IsZero() {
		return 0
	} else {
		return 1 - (int(uint64(l)>>63) << 1)
	}
}

// Compare compares the numbers represented by l1 and l2 without taking into account lost precision and returns:
//
//	-1 if l1 <  l2
//	 0 if l1 == l2
//	+1 if l1 >  l2
func (l1 Length) Compare(l2 Length) int {
	l := l1.Sub(l2)

	if l.IsZero() {
		return 0
	} else if l.IsPositive() {
		return 1
	} else {
		return -1
	}
}

// GreaterThan returns true when l1 is greater than l2 (l1 > l2).
func (l1 Length) GreaterThan(l2 Length) bool {
	l := l1.Sub(l2)

	return l.IsPositive()
}

// GreaterThanOrEqual returns true when l1 is greater than or equal to l2 (l1 >= l2).
func (l1 Length) GreaterThanOrEqual(l2 Length) bool {
	l := l1.Sub(l2)

	return l.IsPositive() || l.IsZero()
}

// LessThan returns true when l1 is less than l2 (l1 < l2).
func (l1 Length) LessThan(l2 Length) bool {
	return l2.GreaterThan(l1)
}

// LessThanOrEqual returns true when l1 is less than or equal to l2 (l1 <= l2).
func (l1 Length) LessThanOrEqual(l2 Length) bool {
	return l2.GreaterThanOrEqual(l1)
}
