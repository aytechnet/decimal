package decimal

import (
	"errors"

	"database/sql/driver"
	"encoding/binary"
	"math"
	"math/bits"
)

// Decimal represents a fixed-point decimal hold as a 64 bits integer
// integer value between -144115188075855871 and 144115188075855871 (or MaxInt) can safely be used as Decimal, example :
//
//	var a Decimal = -1001 // a is a Decimal of integer value -1001
//
//	if a.Div(1000).Sub(1).Div(5).Mul(14000).Add(14).Div(-28) == 1000 {
//	  fmt.Printf("ok: (((a/1000)-1)*14000+14)/-28 == 1000\n") // will print "ok: (((a/1000)-1)*14000+14)/-28 == 1000"
//	}
//
// Note 0 is unitialized Decimal and its value for calculation is 0 like Zero which is initialized 0
// Note Zero is now the same int64 value as types.IntZero, so casting an types.IntZero to Decimal is safe provided is absolute value is not too high
// Note you need to use Decimal method for calculation, you cannot use + - * / or any other operators unless Decimal is a real non-zero integer value
// Unitialized Decimal is useful when using JSON marshaling/unmarshaling.
//
// Note Decimal does not follow IEEE 754 decimal floating point 64 bits representation.
// This is because Decimal is compatible with a large range of int64 values as described above for ease of use
// and use extended mantissa of 57 bits instead of 50 bits for IEEE 754 decimal floating point, thus providing only 17 significant digits (in fact a little more than float64)
type Decimal int64

const (
	// Null constant is the default value of a decimal left unitialized.
	// A decimal can be directly initialized with 0 and it will be empty.
	// Null can be safely compared with == or != directly, you may use decimal.IsNull or any other decimal method to check if it is null.
	// Any operation on Decimal will never return a Null.
	// Null is seen as 0 for any operation
	Null = 0

	// MaxInt constant is the maximal int64 value that can be safely saved as Decimal with exponent still 0.
	// MaxInt is as well the maximum value of mantissa of Decimal and the bitmask to extract mantissa value of a Decimal.
	MaxInt = 0x01ffffffffffffff

	// Zero constant is not empty zero Decimal value.
	// A decimal can be directly initialized with Zero and it will be not empty, but zero.
	// Zero can be safely compared with == or != directly to check for not empty zero decimal value.
	// But you need to use d.IsExactlyZero() to test if decimal d is zero or null.
	Zero Decimal = math.MinInt64

	// NearZero represents a decimal value too close to 0 but not equal to zero, its sign is undefined.
	// NearPositiveZero and NearNegativeZero represents a signed decimal value too close to 0 but not equal to zero, its sign has been kept.
	// They can be safely compared with == or != directly but -NearZero may occurs and should be "seen" as NearZero
	NearZero         Decimal = Zero | loss
	NearPositiveZero Decimal = 0x6000000000000000
	NearNegativeZero Decimal = -NearPositiveZero

	// PositiveInfinity and NegativeInfinity are constants that represents decimal too big to be represented as a decimal value.
	// They can safely be compared with == or != directly to check for infinite value.
	PositiveInfinity Decimal = 0x5e00000000000000
	NegativeInfinity Decimal = -PositiveInfinity

	// NaN represent a decimal which do not represents any more a number.
	// NaN should never be compared with == or != directly as they are multiple representation of such "nan" internally (search for NaN boxing for more info).
	// Use IsNaN method to check if a decimal is not a number.
	NaN Decimal = 0x4200000000000000

	decimal_min_e     = -16
	decimal_max_e     = 15
	decimal_bit_e     = 57
	decimal_e_bitmask = 0x3e00000000000000
)

var (
	// ErrOutOfRange can occurs when converting a decimal to a int or int64 as integer may not hold the integer part of the decimal value.
	ErrOutOfRange = errors.New("out of range")

	// ErrSyntax can occurs when converting a string to a decimal.
	ErrSyntax = errors.New("invalid syntax")

	// ErrFormatcan occurs when decoding a binary to a decimal.
	ErrFormat = errors.New("invalid format")

	// DivisionPrecision has the number of decimal places in the result when it doesn't divide exactly.
	DivisionPrecision = 16
)

// Mantissa returns the mantissa of the decimal.
func (d Decimal) Mantissa() int64 {
	if d < 0 {
		return int64(-d) & MaxInt
	} else {
		return int64(d) & MaxInt
	}
}

// Exponent returns the exponent, or scale component of the decimal.
func (d Decimal) Exponent() int32 {
	var u uint64

	if d < 0 {
		u = uint64(-d)
	} else {
		u = uint64(d)
	}

	e := int64((u&decimal_e_bitmask)<<2) >> (2 + decimal_bit_e) // e is now fully signed exponent

	if u&MaxInt == 0 {
		if e == decimal_min_e {
			return math.MinInt32
		} else if e == decimal_max_e {
			return math.MaxInt32
		}
	}

	return int32(e)
}

// internal function to extract decimal into VME tuple : Value of sign, loss and possibly type, Mantissa and Exponent
func (d Decimal) vme() (v, m uint64, e int64) {
	var u uint64

	if d < 0 {
		u = uint64(-d)
		v = (u & loss) | sign
	} else {
		u = uint64(d)
		v = u & loss
	}

	e = int64((u&decimal_e_bitmask)<<2) >> (2 + decimal_bit_e) // e is now fully signed exponent

	m = u & MaxInt

	// take care of special number
	if m == 0 {
		if e == decimal_min_e {
			e = math.MinInt64
		} else if e == decimal_max_e {
			e = math.MaxInt64
		}
	}

	return
}

// internal function to define a decimal from a VME tuple : Value of sign, loss and possibly type, Mantissa and Exponent
func vmeAsDecimal(v, m uint64, e int64) Decimal {
	// FIXME: handle special case for null and zero
	if m == 0 && v&loss == 0 {
		if v == 0 && e == 0 {
			return Null
		} else {
			return Zero
		}
	} else {
		v, m, e = vmeNormalize(v, m, e, MaxInt, decimal_min_e, decimal_max_e)

		// FIXME: out-of-range cannot occurs as normalization has been done
		v |= m | uint64(e<<decimal_bit_e)&decimal_e_bitmask

		if v&sign != 0 {
			return -Decimal(v ^ sign)
		} else {
			return Decimal(v)
		}
	}
}

// Abs returns the absolute value of the decimal.
func (d Decimal) Abs() Decimal {
	if d < 0 {
		return -d
	} else {
		return d
	}
}

// Add returns d1 + d2.
func (d1 Decimal) Add(d2 Decimal) Decimal {
	v1, m1, e1 := d1.vme()
	v2, m2, e2 := d2.vme()

	return vmeAsDecimal(vmeAdd(v1, m1, e1, v2, m2, e2))
}

// Sub returns d1 - d2.
func (d1 Decimal) Sub(d2 Decimal) Decimal {
	return d1.Add(-d2)
}

// Mul returns d1 * d2.
func (d1 Decimal) Mul(d2 Decimal) Decimal {
	v1, m1, e1 := d1.vme()
	v2, m2, e2 := d2.vme()

	return vmeAsDecimal(vmeMul(v1, m1, e1, v2, m2, e2))
}

// Div returns d1 / d2. If it doesn't divide exactly, the result will have DivisionPrecision digits after the decimal point and loss bit will be set.
func (d1 Decimal) Div(d2 Decimal) Decimal {
	v1, m1, e1 := d1.vme()
	v2, m2, e2 := d2.vme()

	v, m, e, rem, _ := vmeDivRem(v1, m1, e1, v2, m2, e2, int32(DivisionPrecision))

	if rem != 0 {
		v |= loss

		// FIXME: fix m so that the result is the nearest, like shopspring/decimal
		if (rem << 1) >= m2 {
			m++
		}
	}

	return vmeAsDecimal(v, m, e)
}

// QuoRem does division with remainder
// d1.QuoRem(d2,precision) returns quotient q and remainder r such that
//
//	d1 = d2 * q + r, q an integer multiple of 10^(-precision)
//	0 <= r < abs(d2) * 10 ^(-precision) if d1 >= 0
//	0 >= r > -abs(d2) * 10 ^(-precision) if d1 < 0
//
// Note that precision<0 is allowed as input.
func (d1 Decimal) QuoRem(d2 Decimal, precision int32) (Decimal, Decimal) {
	v1, m1, e1 := d1.vme()
	v2, m2, e2 := d2.vme()

	v, m, e, rem, reme := vmeDivRem(v1, m1, e1, v2, m2, e2, precision)

	return vmeAsDecimal(v, m, e), vmeAsDecimal(v, rem, reme)
}

// Mod returns d1 % d2.
func (d1 Decimal) Mod(d2 Decimal) Decimal {
	_, r := d1.QuoRem(d2, 0)

	return r
}

// Neg returns -d.
func (d Decimal) Neg() Decimal {
	if d.IsExactlyZero() || d == NearZero {
		return d
	} else {
		return -d
	}
}

// Equal returns whether d1 == d2 without taking care of loss bit. The values Null, Zero, NearZero, NearPositiveZero and NearNegativeZero are equals.
func (d1 Decimal) Equal(d2 Decimal) bool {
	d := d1.Sub(d2)

	return d.IsZero()
}

// Compare compares the numbers represented by d1 and d2 without taking into account lost precision and returns:
//
//	-1 if d1 <  d2
//	 0 if d1 == d2
//	+1 if d1 >  d2
func (d1 Decimal) Compare(d2 Decimal) int {
	d := d1.Sub(d2)

	if d.IsZero() {
		return 0
	} else if d.IsPositive() {
		return 1
	} else {
		return -1
	}
}

// Cmp is a synonym of Compare.
func (d1 Decimal) Cmp(d2 Decimal) int {
	return d1.Compare(d2)
}

// GreaterThan returns true when d1 is greater than d2 (d1 > d2).
func (d1 Decimal) GreatherThan(d2 Decimal) bool {
	d := d1.Sub(d2)

	return d.IsPositive()
}

// GreaterThanOrEqual returns true when d1 is greater than or equal to d2 (d1 >= d2).
func (d1 Decimal) GreatherThanOrEqual(d2 Decimal) bool {
	d := d1.Sub(d2)

	return d.IsPositive() || d.IsZero()
}

// LessThan returns true when d1 is less than d2 (d1 < d2).
func (d1 Decimal) LessThan(d2 Decimal) bool {
	return d2.GreatherThan(d1)
}

// LessThanOrEqual returns true when d1 is less than or equal to d2 (d1 <= d2).
func (d1 Decimal) LessThanOrEqual(d2 Decimal) bool {
	return d2.GreatherThanOrEqual(d1)
}

// Round rounds the decimal to places decimal places. If places < 0, it will round the integer part to the nearest 10^(-places).
func (d Decimal) Round(places int32) Decimal {
	v, m, e := d.vme()

	return vmeAsDecimal(vmeRound(v, m, e, places))
}

// Ceil returns the nearest integer value greater than or equal to d.
func (d Decimal) Ceil() Decimal {
	return d.RoundCeil(0)
}

// RoundCeil rounds the decimal towards +infinity.
func (d Decimal) RoundCeil(places int32) Decimal {
	v, m, e := d.vme()

	return vmeAsDecimal(vmeRoundCeil(v, m, e, places))
}

// Floor returns the nearest integer value less than or equal to d.
func (d Decimal) Floor() Decimal {
	return d.RoundFloor(0)
}

// RoundFloor rounds the decimal towards -infinity.
//
// Example:
//
//	NewFromFloat(545).RoundFloor(-2).String()   // output: "500"
//	NewFromFloat(-500).RoundFloor(-2).String()   // output: "-500"
//	NewFromFloat(1.1001).RoundFloor(2).String() // output: "1.1"
//	NewFromFloat(-1.454).RoundFloor(1).String() // output: "-1.5"
func (d Decimal) RoundFloor(places int32) Decimal {
	v, m, e := d.vme()

	return vmeAsDecimal(vmeRoundFloor(v, m, e, places))
}

// RoundBank rounds the decimal to places decimal places.
// If the final digit to round is equidistant from the nearest two integers the
// rounded value is taken as the even number
//
// If places < 0, it will round the integer part to the nearest 10^(-places).
//
// Examples:
//
//	NewFromFloat(5.45).RoundBank(1).String() // output: "5.4"
//	NewFromFloat(545).RoundBank(-1).String() // output: "540"
//	NewFromFloat(5.46).RoundBank(1).String() // output: "5.5"
//	NewFromFloat(546).RoundBank(-1).String() // output: "550"
//	NewFromFloat(5.55).RoundBank(1).String() // output: "5.6"
//	NewFromFloat(555).RoundBank(-1).String() // output: "560"
func (d Decimal) RoundBank(places int32) Decimal {
	v, m, e := d.vme()

	return vmeAsDecimal(vmeRoundBank(v, m, e, places))
}

// IsNull return
//
//	true if d == Null
//	false if d == 0
//	false if d == ~0 or d == -~0 or d == +~0
//	false if d < 0
//	false if d > 0
func (d Decimal) IsNull() bool {
	return d == Null
}

// IfNull return
//
//	default_value if d == Null
//	d in any other cases
func (d Decimal) IfNull(default_value Decimal) Decimal {
	if d == Null {
		return default_value
	} else {
		return d
	}
}

// IsSet return
//
//	false if d == Null
//	true if d == 0
//	true if d == ~0 or d == -~0 or d == +~0
//	true if d < 0
//	true if d > 0
func (d Decimal) IsSet() bool {
	return d != Null
}

// IsExactlyZero return
//
//	true if d == Null or d == Zero
//	false if d == ~0 or d == -~0 or d == +~0
//	false if d < 0
//	false if d > 0
func (d Decimal) IsExactlyZero() bool {
	return ^uint64(sign)&uint64(d) == 0 // d == Null || d == Zero
}

// IsZero return
//
//	true if d == Null or d == Zero
//	true if d == ~0 or d == -~0 or d == +~0
//	false if d < 0
//	false if d > 0
func (d Decimal) IsZero() bool {
	return d.IsExactlyZero() || d == NearZero || d == -NearZero || d == NearPositiveZero || d == NearNegativeZero
}

// IsExact return true if a decimal has its loss bit not set, ie it has not lost its precision during computation or conversion.
func (d Decimal) IsExact() bool {
	return d.Abs()&loss == 0
}

// IsInteger return true only if d is zero or can be safely casted as int64
func (d Decimal) IsInteger() bool {
	return ^uint64(sign|MaxInt)&uint64(d.Abs()) == 0
}

// IsPositive return
//
//	true if d > 0 or d == ~+0
//	false if d == Null or d == Zero or d == ~0
//	false if d < 0 or d == ~-0
//	false if d is NaN
func (d Decimal) IsPositive() bool {
	return d > 0 && !d.IsNaN() // FIXME: Zero is negative so this case is not needed
}

// IsNegative return
//
//	true if d < 0 or d == ~-0
//	false if d == Null or d == Zero or d == ~0
//	false if d > 0
func (d Decimal) IsNegative() bool {
	return d != Zero && d != NearZero && d < 0
}

// IsInfinite return
//
//	true if a d == +Inf or d == -Inf
//	false in any other case
func (d Decimal) IsInfinite() bool {
	return d.Abs() == PositiveInfinity
}

// IsNaN return
//
//	true if d is not a a number (NaN)
//	false in any other case
func (d Decimal) IsNaN() bool {
	u := uint64(d.Abs())

	if u&MaxInt == 0 {
		u = u >> 56 // decimal_bit_e - 1 to match last byte (easier to read)

		// excluded as not nan :
		//   0x40 : near zero (exponant = 0)
		//   0x5e : positive infinity (exponant = 15)
		//   0x60 : near positive zero (exponant = -16)
		// nan numbers (nan boxing) :
		//   0x42 to 0x5c : exponant 1 to 14
		//   0x62 to 0x7e : exponant -15 to -1
		return u >= 0x42 && u < 0x5c || u >= 0x62 && u <= 0x7e
	}

	return false
}

// Sign return
//
//	0 if d == Null or d == Zero or d == ~0
//	1 if d > 0 or d == ~+0
//	-1 if d < 0 or d == ~-0
//	undefined (1 or -1) if d is NaN
func (d Decimal) Sign() int {
	if d.IsExactlyZero() || d == NearZero {
		return 0
	} else {
		return 1 - (int(uint64(d)>>63) << 1)
	}
}

// Int64 returns the integer component of the decimal, this method is a synonym of IntPart
func (d Decimal) Int64() (i int64) {
	i, _ = d.IntPartErr()

	return
}

// IntPart returns the integer component of the decimal.
func (d Decimal) IntPart() (i int64) {
	i, _ = d.IntPartErr()

	return
}

// IntPartErr return the integer component of the decimal and an eventual out-of-range error of conversion.
func (d Decimal) IntPartErr() (int64, error) {
	if d.IsInteger() {
		if d == Zero {
			return 0, nil
		} else {
			return int64(d), nil
		}
	}

	v, m, e := d.vme()

	if v&loss != 0 && m == 0 {
		if e == decimal_max_e {
			if d < 0 {
				return math.MinInt64, ErrOutOfRange
			} else {
				return math.MaxInt64, ErrOutOfRange
			}
		} else {
			return 0, ErrOutOfRange
		}
	}

	if e == 0 {
		if d < 0 {
			return -int64(m), nil
		} else {
			return int64(m), nil
		}
	} else if e > 0 {
		hi, lo := bits.Mul64(m, ten_pow[e])

		if hi == 0 && lo <= MaxInt {
			if d < 0 {
				return -int64(lo), nil
			} else {
				return int64(lo), nil
			}
		} else {
			if d < 0 {
				return math.MinInt64, ErrOutOfRange
			} else {
				return math.MaxInt64, ErrOutOfRange
			}
		}
	} else {
		m /= ten_pow[-e]

		if d < 0 {
			return -int64(m), nil
		} else {
			return int64(m), nil
		}
	}
}

// Float64 returns the nearest float64 value for d and a bool indicating whether f may represents d exactly.
func (d Decimal) Float64() (f float64, exact bool) {
	v, m, e := d.vme()

	exact = v&loss == 0

	// take care of special number
	if m == 0 {
		if v&loss != 0 {
			if e == math.MaxInt64 {
				s := 0
				if v&sign != 0 {
					s = -1
				}

				f = math.Inf(s)
			} else if e != 0 && e != math.MinInt64 {
				f = math.NaN()
			}
		}

		return
	}

	f = float64(m)
	if e == 0 {
		if m >= (1 << 54) {
			exact = false
		}
	} else if e > 0 {
		for e >= int64(len(ten_pow)) {
			f *= float64(ten_pow[len(ten_pow)-1])
			e -= int64(len(ten_pow) - 1)
			exact = false
		}
		f *= float64(ten_pow[e])
		if f > float64(1<<54) {
			exact = false
		}
	} else if e < 0 {
		for e <= -int64(len(ten_pow)) {
			f /= float64(ten_pow[len(ten_pow)-1])
			e += int64(len(ten_pow) - 1)
			exact = false
		}
		f /= float64(ten_pow[-e])
		// FIXME: compute exact more accurately
	}

	if v&sign != 0 {
		f = -f
	}

	return
}

// InexactFloat64 returns the nearest float64 value for d.
func (d Decimal) InexactFloat64() float64 {
	f, _ := d.Float64()

	return f
}

// Ln calculates natural logarithm of d. Precision argument specifies how precise the result must be (number of digits after decimal point). Negative precision is allowed.
func (d Decimal) Ln(precision int32) Decimal {
	f, x := d.Float64()

	return NewFromFloat64Exact(math.Log(f), x).Round(precision)
}

// Sqrt computes the (possibly rounded) square root of a decimal.
//
// Special cases are:
//
//	Sqrt(+Inf) = +Inf
//	Sqrt(±0) = ±0
//	Sqrt(x < 0) = NaN
//	Sqrt(NaN) = NaN
func (d Decimal) Sqrt() Decimal {
	f, x := d.Float64()

	return NewFromFloat64Exact(math.Sqrt(f), x)
}

// Pow returns d1**d2, the base-d1 exponential of d2.
func (d1 Decimal) Pow(d2 Decimal) Decimal {
	f1, x1 := d1.Float64()
	f2, x2 := d2.Float64()

	return NewFromFloat64Exact(math.Pow(f1, f2), x1 && x2)
}

// PowWithPrecision returns d to the power of d2. Precision parameter specifies minimum precision of the result (digits after decimal point). Returned decimal is not rounded to 'precision' places after decimal point.
func (d1 Decimal) PowWithPrecision(d2 Decimal, precision int32) (Decimal, error) {
	// FIXME: should return error like shopspring decimal
	return d1.Pow(d2), nil
}

// Atan returns the arctangent, in radians, of d.
func (d Decimal) Atan() Decimal {
	f, x := d.Float64()

	return NewFromFloat64Exact(math.Atan(f), x)
}

// Cos returns the cosine of the radian argument d.
func (d Decimal) Cos() Decimal {
	f, x := d.Float64()

	return NewFromFloat64Exact(math.Cos(f), x)
}

// Sin returns the sine of the radian argument d.
func (d Decimal) Sin() Decimal {
	f, x := d.Float64()

	return NewFromFloat64Exact(math.Sin(f), x)
}

// Tan returns the tangent of the radian argument x.
func (d Decimal) Tan() Decimal {
	f, x := d.Float64()

	return NewFromFloat64Exact(math.Tan(f), x)
}

// New returns a new fixed-point decimal, value * 10 ^ exp, compatible with shopspring/decimal New function.
func New(value int64, exp int32) Decimal {
	if value == 0 {
		// need to handle special case of 0 and speed up result
		return Zero
	} else if value < 0 {
		return vmeAsDecimal(sign, uint64(-value), int64(exp))
	} else {
		return vmeAsDecimal(0, uint64(value), int64(exp))
	}
}

// NewFromInt converts a int64 to Decimal.
func NewFromInt(value int64) Decimal {
	if value < 0 {
		if value >= -MaxInt {
			return Decimal(value)
		} else {
			return vmeAsDecimal(sign, uint64(-value), 0)
		}
	} else {
		if value <= MaxInt {
			if value == 0 {
				return Zero
			} else {
				return Decimal(value)
			}
		} else {
			return vmeAsDecimal(0, uint64(value), 0)
		}
	}
}

// NewFromUint64 converts uint64 to Decimal.
func NewFromUint64(value uint64) Decimal {
	if value <= MaxInt {
		if value == 0 {
			return Zero
		} else {
			return Decimal(value)
		}
	} else {
		return vmeAsDecimal(0, value, 0)
	}
}

// NewFromInt32 converts a int32 to Decimal.
func NewFromInt32(value int32) Decimal {
	if value == 0 {
		return Zero
	} else {
		// int32 value fit in mantissa directly without other conversion
		return Decimal(value)
	}
}

// NewFromFloat converts a float64 to Decimal.
func NewFromFloat(value float64) Decimal {
	return NewFromFloat64Exact(value, true)
}

// NewFromFloat converts a float64 to Decimal.
func NewFromFloat64Exact(value float64, exact bool) Decimal {
	b := math.Float64bits(value)
	e := int64((b >> 52) & 0x7ff)
	v := b & sign

	if !exact {
		v |= loss
	}

	switch e {
	case 2047: // infinite and NaNs
		if (b << 12) == 0 {
			if (b & sign) != 0 {
				return NegativeInfinity
			} else {
				return PositiveInfinity
			}
		} else {
			return NaN
		}
	case 0: // subnormal numbers and signed zeros
		return newFromFloat(v, (b<<11) & ^sign, -1022)
	default:
		return newFromFloat(v, (b<<11)|sign, e-1023)
	}
}

// NewFromFloat32 converts a float32 to Decimal.
func NewFromFloat32(value float32) Decimal {
	b := uint64(math.Float32bits(value))
	e := int64((b >> 23) & 0xff)

	switch e {
	case 255: // infinite and NaNs
		if (b << 9) == 0 {
			if (b & sign) != 0 {
				return NegativeInfinity
			} else {
				return PositiveInfinity
			}
		} else {
			return NaN
		}
	case 0: // subnormal numbers and signed zeros
		return newFromFloat((b<<32)&sign, (b<<40) & ^sign, -126)
	default:
		return newFromFloat((b<<32)&sign, (b<<40)|sign, e-127)
	}
}

// NewFromFloatWithExponent converts a float64 to Decimal, with an arbitrary number of fractional digits.
func NewFromFloatWithExponent(value float64, exp int32) Decimal {
	return NewFromFloat(value).Round(exp)
}

// Sum returns the combined total of the provided first and rest Decimals
// using improved Kahan–Babuška Neumaier algorithm, see https://en.wikipedia.org/wiki/Kahan_summation_algorithm
//
// Example:
//
//	d := Sum(1, RequireFromString("1e30"), 1, RequireFromString("-1e30"))
func Sum(first Decimal, rest ...Decimal) Decimal {
	sum := first
	c := Zero // A running compensation for lost low-order bits.

	for _, item := range rest {
		t := sum.Add(item)

		if sum.Abs().GreatherThanOrEqual(item.Abs()) {
			c = c.Add(sum.Sub(t).Add(item)) // If sum is bigger, low-order digits of item are lost.
		} else {
			c = c.Add(item.Sub(t).Add(sum)) // Else low-order digits of sum are lost.
		}

		sum = t
	}

	return sum.Add(c)
}

// Avg returns the average value of the provided first and rest Decimals
func Avg(first Decimal, rest ...Decimal) Decimal {
	return Sum(first, rest...).Div(Decimal(len(rest) + 1))
}

// Min returns the smallest Decimal that was passed in the arguments.
func Min(first Decimal, rest ...Decimal) Decimal {
	min := first

	for _, item := range rest {
		if min.GreatherThanOrEqual(item) {
			min = item
		}
	}

	return min
}

// Min returns the largest Decimal that was passed in the arguments.
func Max(first Decimal, rest ...Decimal) Decimal {
	max := first
	for _, item := range rest {
		if item.GreatherThanOrEqual(max) {
			max = item
		}
	}
	return max
}

// NewFromBytes returns a new Decimal from a slice of bytes representation.
func NewFromBytes(value []byte) (Decimal, error) {
	if v, m, e, err := vmeFromBytes(value); err == nil {
		return vmeAsDecimal(v, m, e), nil
	} else {
		return 0, err
	}
}

// NewFromString returns a new Decimal from a string representation.
//
// Example:
//
//	d, err := NewFromString("-123.45")
//	d2, err := NewFromString(".0001")
//	d3, err := NewFromString("1.47000")
//	d4, err := NewFromString("3.14e15")
func NewFromString(value string) (Decimal, error) {
	return NewFromBytes([]byte(value))
}

// RequireFromString returns a new Decimal from a string representation
// or panics if NewFromString would have returned an error.
//
// Example:
//
//	d := RequireFromString("-123.45")
//	d2 := RequireFromString(".0001")
func RequireFromString(value string) Decimal {
	if v, m, e, err := vmeFromBytes([]byte(value)); err == nil {
		return vmeAsDecimal(v, m, e)
	} else {
		panic(err)
	}
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
func (d *Decimal) UnmarshalJSON(b []byte) error {
	if v, m, e, err := vmeFromBytes(b); err == nil {
		*d = vmeAsDecimal(v, m, e)

		return nil
	} else {
		return err
	}
}

// String returns the string representation of the decimal with the fixed point.
//
// Example:
//
//	d := New(-12345, -3)
//	println(d.String())
//
// Output:
//
//	-12.345
func (d Decimal) String() string {
	if d == Null {
		return "0"
	} else {
		return string(d.Bytes())
	}
}

// Bytes returns the string representation of the decimal as a slice of byte, but nil if the decimal is Null.
func (d Decimal) Bytes() (b []byte) {
	if d == Null {
		return nil
	} else {
		v, m, e := d.vme()

		// the maximal length of decimal representation in bytes in such conditions is 20
		return vmeBytes(make([]byte, 0, 20), v, m, e, true, false)
	}
}

// MarshalJSON implements the json.Marshaler interface.
func (d Decimal) MarshalJSON() ([]byte, error) {
	v, m, e := d.vme()

	return vmeBytes(nil, v, m, e, false, false), nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
func (d *Decimal) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		return ErrFormat
	}

	u := uint64(data[0]) << (decimal_bit_e - 1)
	if data[0]&1 != 0 {
		// clear low bit
		u = u ^ (1 << (decimal_bit_e - 1))
		if m, n := binary.Uvarint(data[1:]); n <= 0 {
			return ErrFormat
		} else {
			u |= m
		}
	}

	if u&sign != 0 && Decimal(u) != Zero {
		*d = -Decimal(u ^ sign)
	} else {
		*d = Decimal(u)
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (d Decimal) MarshalBinary() (data []byte, err error) {
	var u uint64
	var x byte

	if d < 0 {
		u = uint64(-d)
		x = byte((u | sign) >> (decimal_bit_e - 1))
	} else {
		u = uint64(d)
		x = byte(u >> (decimal_bit_e - 1))
	}
	u = u & MaxInt

	if u == 0 {
		// bit 0 is already unset as u is zero
		data = []byte{x}
	} else {
		// bit 0 is on to indicate a non-zero mantissa
		buff := [10]byte{x | 1}

		n := binary.PutUvarint(buff[1:], u)
		data = buff[0 : n+1]
	}

	return
}

// UnmarshalText implements the encoding.TextUnmarshaler interface for XML deserialization.
func (d *Decimal) UnmarshalText(text []byte) error {
	if _d, err := NewFromBytes(text); err != nil {
		return err
	} else {
		*d = _d

		return nil
	}
}

// MarshalText implements the encoding.TextMarshaler interface for XML serialization.
func (d Decimal) MarshalText() (text []byte, err error) {
	return d.Bytes(), nil
}

// GobEncode implements the gob.GobEncoder interface for gob serialization.
func (d Decimal) GobEncode() ([]byte, error) {
	return d.MarshalBinary()
}

// GobDecode implements the gob.GobDecoder interface for gob serialization.
func (d *Decimal) GobDecode(data []byte) error {
	return d.UnmarshalBinary(data)
}

// Scan implements the sql.Scanner interface for database deserialization.
func (d *Decimal) Scan(value interface{}) (err error) {
	// first try to see if the data is stored in database as a Numeric datatype
	switch v := value.(type) {
	case float32:
		*d = NewFromFloat(float64(v))
		return nil

	case float64:
		// numeric in sqlite3 sends us float64
		*d = NewFromFloat(v)
		return nil

	case int64:
		// at least in sqlite3 when the value is 0 in db, the data is sent
		// to us as an int64 instead of a float64 ...
		*d = New(v, 0)
		return nil

	case uint64:
		// while clickhouse may send 0 in db as uint64
		*d = NewFromUint64(v)
		return nil

	case string:
		*d, err = NewFromString(v)
		return err

	case []byte:
		*d, err = NewFromBytes(v)
		return err

	default:
		return ErrFormat
	}
}

// Value implements the driver.Valuer interface for database serialization.
func (d Decimal) Value() (driver.Value, error) {
	return d.String(), nil
}
