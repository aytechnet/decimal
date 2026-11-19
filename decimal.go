// Package decimal provides a high-performance, zero-allocation fixed-point decimal type
// (and companion Weight / Length types with unit support) packed in a single 8-byte int64.
// See README.md for an overview and BINARY_FORMAT.md for the open wire-format specification.
package decimal

import (
	"errors"

	"database/sql/driver"
	"encoding/binary"
	"math"
	"math/bits"
	"regexp"
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

	decimalMinE     = -16
	decimalMaxE     = 15
	decimalBitE     = 57
	decimalEBitmask = 0x3e00000000000000
)

var (
	// ErrOutOfRange can occurs when converting a decimal to a int or int64 as integer may not hold the integer part of the decimal value.
	ErrOutOfRange = errors.New("out of range")

	// ErrSyntax can occurs when converting a string to a decimal.
	ErrSyntax = errors.New("invalid syntax")

	// ErrUnitSyntax occurs when unit is not recognized.
	ErrUnitSyntax = errors.New("invalid unit syntax")

	// ErrFormatcan occurs when decoding a binary to a decimal.
	ErrFormat = errors.New("invalid format")

	// DivisionPrecision has the number of decimal places in the result when it doesn't divide exactly.
	DivisionPrecision = 16

	// PowPrecisionNegativeExponent has the maximum precision (digits after the decimal point) of the result of PowInt32 when the exponent is negative.
	PowPrecisionNegativeExponent = 16
)

// Mantissa returns the mantissa of the decimal.
func (d Decimal) Mantissa() int64 {
	if d < 0 {
		return int64(-d) & MaxInt
	} else {
		return int64(d) & MaxInt
	}
}

// NumDigits returns the number of digits of the decimal mantissa in base 10.
// Mantissa is only normalized (trailing zeros dropped) when its absolute value would otherwise exceed MaxInt
// or its exponent would be out of range, so 100 is kept as mantissa=100 (NumDigits=3) while 10^18 is normalized to mantissa=1, exp=18 (NumDigits=1).
// Null, Zero, NearZero, NearPositiveZero, NearNegativeZero, +Inf, -Inf and NaN return 1.
func (d Decimal) NumDigits() int {
	m := uint64(d.Mantissa())
	if m == 0 {
		return 1
	}

	// mantissa is always <= MaxInt (~1.44e17 < 10^18), so iterating from 19 down to 1 always finds the first tenPow[i] <= m or falls through to a single-digit mantissa
	for i := len(tenPow) - 1; i > 0; i-- {
		if m >= tenPow[i] {
			return i + 1
		}
	}

	return 1
}

// Exponent returns the exponent, or scale component of the decimal.
func (d Decimal) Exponent() int32 {
	var u uint64

	if d < 0 {
		u = uint64(-d)
	} else {
		u = uint64(d)
	}

	e := int64((u&decimalEBitmask)<<2) >> (2 + decimalBitE) // e is now fully signed exponent

	if u&MaxInt == 0 {
		if e == decimalMinE {
			return math.MinInt32
		} else if e == decimalMaxE {
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

	e = int64((u&decimalEBitmask)<<2) >> (2 + decimalBitE) // e is now fully signed exponent

	m = u & MaxInt

	// take care of special number
	if m == 0 {
		if e == decimalMinE {
			e = math.MinInt64
		} else if e == decimalMaxE {
			e = math.MaxInt64
		}
	}

	return
}

// internal function to define a decimal from a VME tuple : Value of sign, loss and possibly type, Mantissa and Exponent
func vmeAsDecimal(v, m uint64, e int64) Decimal {
	// handle special case for null and zero
	if m == 0 && v&loss == 0 {
		if v == 0 && e == 0 {
			return Null
		} else {
			return Zero
		}
	} else {
		v, m, e = vmeNormalize(v, m, e, MaxInt, decimalMinE, decimalMaxE)

		// out-of-range cannot occurs as normalization has been done
		v |= m | uint64(e<<decimalBitE)&decimalEBitmask

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
	// Fast path: both operands are int64-encoded integers in [-MaxInt, MaxInt] (Null included as 0).
	// Magic encodings (Zero, NaN, Inf, ~0, …) all sit outside this range, so the comparison alone filters them out.
	if -MaxInt <= int64(d1) && int64(d1) <= MaxInt && -MaxInt <= int64(d2) && int64(d2) <= MaxInt {
		// |d1| + |d2| ≤ 2*MaxInt < 2^58 — no int64 overflow possible.
		sum := int64(d1) + int64(d2)
		if -MaxInt <= sum && sum <= MaxInt {
			if sum == 0 {
				// preserve the Null + Null = Null invariant; any other zero result is an explicit Zero
				if d1 == 0 && d2 == 0 {
					return Null
				}
				return Zero
			}
			return Decimal(sum)
		}
	}

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
	// Fast path: both operands are int64-encoded integers in [-MaxInt, MaxInt] and their product fits.
	// bits.Mul64 on the absolute values catches overflow exactly, so the full integer range is usable
	// (not just the conservative sqrt(MaxInt) gate).
	if -MaxInt <= int64(d1) && int64(d1) <= MaxInt && -MaxInt <= int64(d2) && int64(d2) <= MaxInt {
		n1, n2 := int64(d1), int64(d2)
		negative := (n1 < 0) != (n2 < 0)

		a1 := uint64(n1)
		if n1 < 0 {
			a1 = uint64(-n1)
		}
		a2 := uint64(n2)
		if n2 < 0 {
			a2 = uint64(-n2)
		}

		hi, lo := bits.Mul64(a1, a2)
		if hi == 0 && lo <= MaxInt {
			if lo == 0 {
				return Zero
			}
			if negative {
				return -Decimal(lo)
			}
			return Decimal(lo)
		}
	}

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

		// fix m so that the result is the nearest, like in shopspring/decimal
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

// DivRound divides d1 by d2 and rounds the result to a given precision (an integer multiple of 10^(-precision)).
//
// Rounding follows the package Round semantics. Negative precision is allowed.
func (d1 Decimal) DivRound(d2 Decimal, precision int32) Decimal {
	p := precision + 1
	if dp := int32(DivisionPrecision); p < dp {
		p = dp
	}

	v1, m1, e1 := d1.vme()
	v2, m2, e2 := d2.vme()

	v, m, e, rem, _ := vmeDivRem(v1, m1, e1, v2, m2, e2, p)

	if rem != 0 {
		v |= loss

		// fix m so that the result is the nearest, like in shopspring/decimal
		if (rem << 1) >= m2 {
			m++
		}
	}

	return vmeAsDecimal(vmeRound(v, m, e, precision))
}

// Neg returns -d.
func (d Decimal) Neg() Decimal {
	if d.IsExactlyZero() || d == NearZero {
		return d
	} else {
		return -d
	}
}

// Copy returns a copy of the decimal. As Decimal is an immutable int64, the returned value has the same bit pattern.
// Provided for API compatibility with shopspring/decimal.
func (d Decimal) Copy() Decimal {
	return d
}

// Normalize returns d unchanged: a Decimal already has a unique representation (1.50, 1.5 and New(15, -1)
// are the same int64, and every exact zero is Zero), so == and reflect.DeepEqual compare values.
// Provided for API compatibility with shopspring/decimal, where it strips the trailing zeros of the coefficient.
func (d Decimal) Normalize() Decimal {
	return d
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
func (d1 Decimal) GreaterThan(d2 Decimal) bool {
	d := d1.Sub(d2)

	return d.IsPositive()
}

// GreaterThanOrEqual returns true when d1 is greater than or equal to d2 (d1 >= d2).
func (d1 Decimal) GreaterThanOrEqual(d2 Decimal) bool {
	d := d1.Sub(d2)

	return d.IsPositive() || d.IsZero()
}

// LessThan returns true when d1 is less than d2 (d1 < d2).
func (d1 Decimal) LessThan(d2 Decimal) bool {
	return d2.GreaterThan(d1)
}

// LessThanOrEqual returns true when d1 is less than or equal to d2 (d1 <= d2).
func (d1 Decimal) LessThanOrEqual(d2 Decimal) bool {
	return d2.GreaterThanOrEqual(d1)
}

// Round rounds the decimal to places decimal places. If places < 0, it will round the integer part to the nearest 10^(-places).
// A half rounds away from zero, like shopspring/decimal: 2.5 -> 3 and -2.5 -> -3.
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

// RoundDown rounds the decimal towards zero. If places < 0, it will round the integer part to the nearest 10^(-places).
//
// Examples:
//
//	NewFromFloat(545).RoundDown(-2).String()    // "500"
//	NewFromFloat(-500).RoundDown(-2).String()   // "-500"
//	NewFromFloat(1.1001).RoundDown(2).String()  // "1.1"
//	NewFromFloat(-1.454).RoundDown(1).String()  // "-1.4"
func (d Decimal) RoundDown(places int32) Decimal {
	if d.IsNegative() {
		return d.RoundCeil(places)
	}

	return d.RoundFloor(places)
}

// RoundUp rounds the decimal away from zero. If places < 0, it will round the integer part to the nearest 10^(-places).
//
// Examples:
//
//	NewFromFloat(545).RoundUp(-2).String()    // "600"
//	NewFromFloat(500).RoundUp(-2).String()    // "500"
//	NewFromFloat(1.1001).RoundUp(2).String()  // "1.11"
//	NewFromFloat(-1.454).RoundUp(1).String()  // "-1.5"
func (d Decimal) RoundUp(places int32) Decimal {
	if d.IsNegative() {
		return d.RoundFloor(places)
	}

	return d.RoundCeil(places)
}

// Truncate truncates digits from the decimal without rounding (towards zero), keeping precision digits
// after the decimal point. A negative precision truncates the integer part, like shopspring/decimal v1.5+.
//
// Example:
//
//	d, _ := NewFromString("123.456")
//	d.Truncate(2).String()  // "123.45"
//	New(5432, 0).Truncate(-2).String()  // "5400"
//	New(-5432, 0).Truncate(-2).String() // "-5400"
func (d Decimal) Truncate(precision int32) Decimal {
	return d.RoundDown(precision)
}

// Shift shifts the decimal in base 10. Positive shift moves left (multiply by 10^shift), negative shift moves right.
// In other words, the value of shift is added to the exponent of the decimal.
//
// Examples:
//
//	d, _ := NewFromString("123.45")
//	d.Shift(1).String()   // "1234.5"
//	d.Shift(-1).String()  // "12.345"
func (d Decimal) Shift(shift int32) Decimal {
	v, m, e := d.vme()

	// magic values (Null, Zero, NearZero, +Inf, -Inf, NaN) are returned as-is
	if m == 0 {
		return d
	}

	return vmeAsDecimal(v, m, e+int64(shift))
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

// RoundCash rounds the decimal to the nearest multiple of the given Cash interval (in units of 10^(-2), or hundredths).
// Valid intervals are 5, 10, 25, 50 and 100 (Swedish/cash rounding). Panics for any other interval.
//
// Examples:
//
//	NewFromFloat(3.43).RoundCash(5)   // 3.45
//	NewFromFloat(3.45).RoundCash(10)  // 3.5
//	NewFromFloat(3.41).RoundCash(25)  // 3.5
//	NewFromFloat(3.75).RoundCash(50)  // 4
//	NewFromFloat(3.50).RoundCash(100) // 4
func (d Decimal) RoundCash(interval uint8) Decimal {
	var factor Decimal

	switch interval {
	case 5:
		factor = 20
	case 10:
		factor = 10
	case 25:
		factor = 4
	case 50:
		factor = 2
	case 100:
		factor = 1
	default:
		panic("Decimal does not support this Cash rounding interval, valid intervals are 5, 10, 25, 50 and 100")
	}

	return d.Mul(factor).Round(0).Div(factor)
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
func (d Decimal) IfNull(defaultValue Decimal) Decimal {
	if d == Null {
		return defaultValue
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
	return d > 0 && !d.IsNaN() // Zero is negative so this case is not needed
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
		// vme() rewrites e to math.MaxInt64 / math.MinInt64 / [1..14] for magic values when m == 0,
		// so e == decimalMaxE is unreachable here — only the catch-all "out of range" path runs.
		return 0, ErrOutOfRange
	}

	if e == 0 {
		if d < 0 {
			return -int64(m), nil
		} else {
			return int64(m), nil
		}
	} else if e > 0 {
		hi, lo := bits.Mul64(m, tenPow[e])

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
		m /= tenPow[-e]

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
	// e is bounded by [-16, 15] for non-magic Decimals, so the cascading loops in core.go's wider-mantissa cousins are not needed here
	if e == 0 {
		if m >= (1 << 54) {
			exact = false
		}
	} else if e > 0 {
		f *= float64(tenPow[e])
		if f > float64(1<<54) {
			exact = false
		}
	} else {
		f /= float64(tenPow[-e])
		// may compute exact more accurately
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
//
// For precision >= 16 and a normal strictly-positive finite operand, Ln uses a
// fixed-point binary-reduction + atanh-series path (ln_highprec.go) that fills
// the type's full 57-bit mantissa, avoiding math.Log entirely. It is ~10× more
// accurate at the last digit than the float64 path; getting that precision any
// other way needs an arbitrary-precision library (~1200× slower), so the few ns
// it costs over the float64 path is negligible. Values whose result |ln| is tiny
// (x within ~1% of 1) fall back to the float64 path, which math.Log computes
// accurately in that regime; any precision < 16 call also uses the float64 path.
// NaN for d <= 0.
func (d Decimal) Ln(precision int32) Decimal {
	f, x := d.Float64()

	// High-precision path: for a normal, strictly-positive, finite operand and
	// precision >= 16, compute ln to the full 57-bit mantissa instead of being
	// capped at float64. See ln_highprec.go / LN_PRECISION_PLAN.md.
	if precision >= lnHighPrecMin && f > 0 && !math.IsInf(f, 0) {
		if v, m, e := d.vme(); m != 0 && v&sign == 0 {
			if r, ok := lnHighPrec(m, e, precision); ok {
				return r
			}
		}
	}

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
//
// Intentionally NOT compatible with shopspring/decimal v1.6+, whose Sqrt returns (Decimal, error):
// the existing v1 signature is kept, a negative input yields NaN instead of an error (see README).
func (d Decimal) Sqrt() Decimal {
	f, x := d.Float64()

	return NewFromFloat64Exact(math.Sqrt(f), x)
}

// SqrtRound returns the square root of d rounded half away from zero to places digits after the decimal
// point, like Round. It is computed from float64, so beyond ~16 significant digits the digits are those
// of Sqrt.
//
// Intentionally NOT compatible with shopspring/decimal v1.6+, whose SqrtRound returns (Decimal, error):
// a negative input yields NaN instead of an error (see README).
func (d Decimal) SqrtRound(places int32) Decimal {
	return d.Sqrt().Round(places)
}

// Root returns the n-th root of d. Odd roots of negative numbers are negative.
//
// Special cases are:
//
//	Root(n < 1) = NaN
//	Root(x < 0) = NaN for an even n
//	Root(±0) = ±0
//	Root(+Inf) = +Inf, Root(-Inf) = -Inf for an odd n
//	Root(NaN) = NaN
//
// Intentionally NOT compatible with shopspring/decimal v1.6+, whose Root returns (Decimal, error): an
// out-of-domain input yields NaN instead of an error, and the result has the precision of a float64
// (~16 significant digits) rather than DivisionPrecision places (see README).
func (d Decimal) Root(n int64) Decimal {
	if n < 1 {
		return NaN
	}
	if n == 1 {
		return d
	}
	if n == 2 {
		return d.Sqrt()
	}

	f, x := d.Float64()
	neg := f < 0
	if neg {
		if n%2 == 0 {
			return NaN
		}
		f = -f
	}

	var r float64
	switch {
	case n == 3:
		r = math.Cbrt(f)
	case f == 0 || math.IsInf(f, 0) || math.IsNaN(f):
		r = f
	default:
		r = math.Pow(f, 1/float64(n))
		// one Newton step corrects the error of 1/n not being exact in binary
		r -= (math.Pow(r, float64(n)) - f) / (float64(n) * math.Pow(r, float64(n-1)))
	}
	// an exact integer root (Root(32, 5) = 2) must not come out as 1.9999999999999998
	if i := math.Round(r); i != r && math.Pow(i, float64(n)) == f {
		r = i
	}

	if neg {
		r = -r
	}
	return NewFromFloat64Exact(r, x)
}

// RootRound returns the n-th root of d rounded half away from zero to places digits after the decimal
// point, like Round. Same special cases and float64 precision as Root.
//
// Intentionally NOT compatible with shopspring/decimal v1.6+, whose RootRound returns (Decimal, error).
func (d Decimal) RootRound(n int64, places int32) Decimal {
	return d.Root(n).Round(places)
}

// Pow returns d1**d2, the base-d1 exponential of d2.
func (d1 Decimal) Pow(d2 Decimal) Decimal {
	f1, x1 := d1.Float64()
	f2, x2 := d2.Float64()

	return NewFromFloat64Exact(math.Pow(f1, f2), x1 && x2)
}

// PowWithPrecision returns d to the power of d2. Precision parameter specifies minimum precision of the result (digits after decimal point). Returned decimal is not rounded to 'precision' places after decimal point.
func (d1 Decimal) PowWithPrecision(d2 Decimal, precision int32) (Decimal, error) {
	// compatibility issue as this code do not return error like shopspring decimal
	return d1.Pow(d2), nil
}

// PowInt32 returns d to the power of exp using fast exponentiation by squaring (so without going through float64 like Pow).
// When exp is negative, the result is rounded to PowPrecisionNegativeExponent digits after the decimal point.
//
// Returns an error only when d is zero and exp is zero (indeterminate form 0**0).
//
// Examples:
//
//	NewFromFloat(4).PowInt32(4)    // 256, nil
//	NewFromFloat(3.13).PowInt32(5) // 300.4150512793, nil
//	NewFromFloat(15.2).PowInt32(-2) // 0.0043282548476454, nil
func (d Decimal) PowInt32(exp int32) (Decimal, error) {
	if exp == 0 {
		if d.IsExactlyZero() {
			return Null, errors.New("indeterminate form: 0**0")
		}

		return 1, nil
	}

	n := int64(exp)
	if n < 0 {
		n = -n
	}

	// fast exponentiation by squaring
	result := Decimal(1)
	base := d

	for n > 0 {
		if n&1 == 1 {
			result = result.Mul(base)
		}

		n >>= 1
		if n > 0 {
			base = base.Mul(base)
		}
	}

	if exp < 0 {
		result = Decimal(1).DivRound(result, int32(PowPrecisionNegativeExponent))
	}

	return result, nil
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

// NewFromFloat converts a float64 to Decimal using the shortest decimal that converts back to the same
// float64, like shopspring/decimal and strconv.FormatFloat(value, 'g', -1, 64): NewFromFloat(-178464.49)
// is exactly -178464.49, not the binary value -178464.48999999999068... It has at most 17 significant
// digits, so it always fits the 57-bit mantissa and is exact (no loss bit). -0 gives NearNegativeZero,
// NaN and ±Inf give NaN and ±Inf.
//
// NewFromFloat64Exact converts the binary value instead, as the math functions do.
func NewFromFloat(value float64) Decimal {
	return newFromShortestFloat(value, 64)
}

// newFromShortestFloat parses the shortest decimal representation of value (a float64, or a float32
// when bitSize is 32) without allocating.
func newFromShortestFloat(value float64, bitSize int) Decimal {
	switch {
	case value == 0:
		if math.Signbit(value) {
			return NearNegativeZero
		}
		return Zero
	case math.IsNaN(value):
		return NaN
	case math.IsInf(value, 1):
		return PositiveInfinity
	case math.IsInf(value, -1):
		return NegativeInfinity
	}

	if d, ok := newFromShortFloat(value, bitSize); ok {
		return d
	}

	// the general path: Schubfach, on the binary fields of the float
	if bitSize == 32 {
		b := math.Float32bits(float32(value))
		frac, biased := uint64(b&(1<<23-1)), int64(b>>23)&0xff
		if biased == 0 {
			return newFromBinaryFloat(value < 0, frac, -149, false) // subnormal
		}
		return newFromBinaryFloat(value < 0, frac|1<<23, biased-150, frac == 0 && biased > 1)
	}
	b := math.Float64bits(value)
	frac, biased := b&(1<<52-1), int64(b>>52)&0x7ff
	if biased == 0 {
		return newFromBinaryFloat(value < 0, frac, -1074, false) // subnormal
	}
	return newFromBinaryFloat(value < 0, frac|1<<52, biased-1075, frac == 0 && biased > 1)
}

// newFromBinaryFloat converts the float c×2^q to the Decimal of its shortest decimal (shortest.go),
// rounded by vmeAsDecimal like any other decimal when its digits go under 10^-16. A float out of
// the scales shortestDecimal covers needs no digit: from 2^107 on it is beyond the largest Decimal
// (about 1.44×10^32), an infinity; under 2^-73 it is far under the smallest (10^-16), a near zero.
func newFromBinaryFloat(negative bool, c uint64, q int64, closer bool) Decimal {
	var v uint64
	if negative {
		v = sign
	}

	switch {
	case int64(bits.Len64(c))+q > 107:
		return vmeAsDecimal(v|loss, 0, math.MaxInt64)
	case q < -125:
		return vmeAsDecimal(v|loss, 0, math.MinInt64)
	}

	m, e := shortestDecimal(c, q, closer)
	return vmeAsDecimal(v, m, e)
}

// float64Pow10 are the powers of ten a float64 holds exactly.
var float64Pow10 = [...]float64{1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9, 1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19, 1e20, 1e21, 1e22}

// newFromShortFloat is the fast path of newFromShortestFloat, without text: it finds the shortest
// decimal of a float64 that has at most 15 significant digits — an amount, a rate, a weight — and of
// an integer; ok is false for any other value, left to newFromFloatText.
//
// The value is scaled by the largest power of ten 10^k that keeps it under 10^15 and rounded to the
// integer m. Both m and 10^k (k <= 22) are exact float64 and an IEEE division is correctly rounded,
// so float64(m)/10^k is the float64 nearest to m×10^-k, what strconv.ParseFloat would give: when it
// is the value, m×10^-k converts back to it. It is then THE shortest decimal: a decimal of at most
// 15 significant digits survives a round trip through float64 (DBL_DIG), so two of them never share
// a float64, and the shortest one — which cannot be longer than this one — is this one, written with
// trailing zeros that vmeAsDecimal drops. Conversely when the shortest decimal has at most 15 digits,
// it is found: the product is off by less than 10^15 × 2^-52 < 0.25, so it rounds to the right m.
func newFromShortFloat(value float64, bitSize int) (Decimal, bool) {
	var v uint64

	abs := value
	if abs < 0 {
		abs, v = -abs, sign
	}

	// an integer a float holds exactly, its neighbours being at least 1 apart, is its own shortest
	// decimal (2^53 for a float64, 2^24 for a float32)
	if bitSize == 32 {
		if m := uint64(abs); abs < 1<<24 && float64(m) == abs {
			return vmeAsDecimal(v, m, 0), true
		}
		return 0, false
	}
	if abs >= 1e15 {
		if m := uint64(abs); abs < 1<<53 && float64(m) == abs {
			return vmeAsDecimal(v, m, 0), true
		}
		return 0, false
	}

	// k: the largest power of ten keeping abs×10^k under 10^15. The binary exponent gives it or one
	// too many (log10(2) ~ 1233/4096, rounded down), never one too few
	k := 14 - ((int(math.Float64bits(abs)>>52)-1023)*1233)>>12
	if k > 22 {
		k = 22
	}
	s := abs * float64Pow10[k]
	if s >= 1e15 {
		k--
		s = abs * float64Pow10[k]
	}

	m := uint64(s + 0.5)
	if m == 0 || float64(m)/float64Pow10[k] != abs {
		return 0, false
	}

	// drop the trailing zeros here (at most 14, m being under 10^15): the exponent must fit a Decimal
	// before vmeAsDecimal
	e := int64(-k)
	if m%100000000 == 0 {
		m, e = m/100000000, e+8
	}
	if m%10000 == 0 {
		m, e = m/10000, e+4
	}
	if m%100 == 0 {
		m, e = m/100, e+2
	}
	if m%10 == 0 {
		m, e = m/10, e+1
	}
	if e < -16 {
		return 0, false
	}

	return vmeAsDecimal(v, m, e), true
}

// NewFromFloat64Exact converts the binary value of a float64 to Decimal, rounded to the 57-bit mantissa
// with the loss bit set when rounded (0.1 is ~0.1000000000000000055…); when exact is false, the loss bit
// is forced on the result. The math functions (Sqrt, Sin, Atan…) use it. NewFromFloat converts the
// shortest decimal instead, which is what a float literal or a parsed amount means.
func NewFromFloat64Exact(value float64, exact bool) Decimal {
	return newFromFloat64Binary(value, exact)
}

// newFromFloat64Binary converts the binary value of a float64, rounded to the 57-bit mantissa (loss bit
// set when rounded). When exact is false, the loss bit is forced on the result.
func newFromFloat64Binary(value float64, exact bool) Decimal {
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

// NewFromFloat32 converts a float32 to Decimal using the shortest decimal that converts back to the
// same float32, like shopspring/decimal: NewFromFloat32(54.33) is exactly 54.33.
func NewFromFloat32(value float32) Decimal {
	return newFromShortestFloat(float64(value), 32)
}

// NewFromFloatWithExponent converts a float64 to Decimal with -exp fractional digits, like
// shopspring/decimal: exp is an exponent, so -2 keeps 2 decimal places and 1 rounds to the tens.
// The exact binary value of the float is rounded half away from zero, so 0.015, which is
// 0.01499999999999999944... as a float64, gives 0.01 at exp -2 (NewFromFloat(0.015).Round(2) gives 0.02).
// Unlike shopspring it does not panic on NaN or ±Inf, it returns them.
//
// It is computed with exact integer arithmetic, on 128 bits, without text. When the rounded value
// has more digits than a Decimal holds — more than 16 places, or a mantissa beyond 57 bits — the
// exact value is rounded once, at the finest exponent a Decimal has for it, and the loss bit is
// set. A value beyond the largest Decimal gives an infinity.
//
// Example:
//
//	NewFromFloatWithExponent(123.456, -2).String() // output: "123.46"
func NewFromFloatWithExponent(value float64, exp int32) Decimal {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return NewFromFloat(value)
	}
	if value == 0 {
		return Zero
	}

	// |value| = mant × 2^e2, exactly
	var v uint64
	b := math.Float64bits(value)
	if b&sign != 0 {
		v = sign
	}
	mant := b & (1<<52 - 1)
	e2 := int64((b >> 52) & 0x7ff)
	if e2 == 0 {
		e2++ // subnormal
	} else {
		mant |= 1 << 52
	}
	e2 -= 1023 + 52

	// beyond the largest Decimal, (2^57-1) × 10^15 — between 2^106 and 2^107 — the value is an
	// infinity, whatever the exponent asked
	size := int64(bits.Len64(mant)) + e2 // 2^(size-1) <= |value| < 2^size
	if size >= 107 {
		maxhi, maxlo := bits.Mul64(MaxInt, tenPow[-decimalMinE-1])
		hi, lo := floatShiftLeft(0, mant, uint(e2))
		if size > 107 || hi > maxhi || hi == maxhi && lo > maxlo {
			return vmeAsDecimal(v|loss, 0, math.MaxInt64)
		}
	}

	// the value rounded at the asked exponent, as an integer N: it is the result when N × 10^exp is
	// a Decimal, that is when N without its trailing zeros fits the mantissa, with at most 16 places
	places := -int64(exp)
	if places < -38 {
		return Zero // |value| < 2^107 < 10^39/2
	}
	rounded := false
	if places <= floatMaxPlaces {
		if hi, lo, _, ok := floatRoundScaled(mant, e2, places); ok {
			if hi|lo == 0 {
				return Zero
			}
			e := -places
			for hi != 0 || lo > MaxInt || e < decimalMinE {
				qhi, r := bits.Div64(0, hi, 10)
				qlo, r := bits.Div64(r, lo, 10)
				if r != 0 {
					break
				}
				hi, lo, e = qhi, qlo, e+1
			}
			if hi == 0 && lo <= MaxInt && e >= decimalMinE {
				return vmeAsDecimal(v, lo, e)
			}
			rounded = true
		}
	}

	// more digits than a Decimal holds, either asked or because the value is large: the value is
	// rounded once more, from its exact binary value and half away from zero, at the finest
	// exponent a Decimal has for it — at most 16 places, and a mantissa that fits. The loss bit
	// tells that digits of the asked rounding were dropped: always when that rounding was computed
	// above; when it was out of reach (more than 27 places, or beyond 128 bits), only when it is
	// not the same number as this one (floatSameRounding).
	// 10^d <= 2^(size-1) <= |value|: starting from 17-d places, one too many at most is tried
	p := 17 - ((size-1)*1233)>>12
	if p > -decimalMinE {
		p = -decimalMinE
	}
	if p > places {
		p = places
	}
	for {
		hi, lo, inexact, ok := floatRoundScaled(mant, e2, p)
		if !ok || hi != 0 || lo > MaxInt {
			p--
			continue
		}
		if rounded || inexact && (p < 0 || !floatSameRounding(mant, e2, p, places)) {
			v |= loss
		}
		if lo == 0 {
			// under half of the last place of a Decimal: zero when the asked rounding is zero too,
			// else a near zero of its sign
			if v&loss == 0 {
				return Zero
			}
			return vmeAsDecimal(v, 0, math.MinInt64)
		}
		return vmeAsDecimal(v, lo, -p)
	}
}

// floatSameRounding tells whether mant × 2^e2 rounds to the same number at places decimal places
// as at p of them, 0 <= p < places, where it was rounded as N: the distance from the value to
// N × 10^-p must be under half of 10^-places. Times 10^p, that distance is the fraction dropped by
// floatRoundScaled, or its complement to 1 when N was rounded up.
func floatSameRounding(mant uint64, e2, p, places int64) bool {
	hi, lo := bits.Mul64(mant, floatPow5(p))
	t := -(e2 + p) // bits dropped; not negative: the product was not an integer
	if t > 128 {
		return floatUnderHalf(hi, lo, -t, places-p) // N was zero
	}
	hi, lo = floatShiftLeft(hi, lo, uint(128-t)) // the fraction, times 2^128
	if hi>>63 != 0 {
		var borrow uint64
		lo, borrow = bits.Sub64(0, lo, 0)
		hi, _ = bits.Sub64(0, hi, borrow)
	}
	return floatUnderHalf(hi, lo, -128, places-p)
}

// floatUnderHalf tells whether x × 2^e2 is under half of 10^-places, x a non-zero 128-bit integer
// and places at least 1: x × 5^places < 2^(-e2-1-places), an exact comparison of the bit length of
// a product that takes up to 16 words. Past 344 places no x × 2^e2 met here is that small (the
// smallest are a float64, at least 2^-1074, and a fraction of 128 bits).
func floatUnderHalf(hi, lo uint64, e2, places int64) bool {
	if places >= 344 {
		return false
	}

	var p [16]uint64
	p[0], p[1] = lo, hi
	n := 2
	for left := places; left > 0; left -= floatMaxPlaces {
		m := floatPow5(floatMaxPlaces)
		if left < floatMaxPlaces {
			m = floatPow5(left)
		}
		var carry uint64
		for i := 0; i < n; i++ {
			h, l := bits.Mul64(p[i], m)
			l, c := bits.Add64(l, carry, 0)
			p[i], carry = l, h+c
		}
		p[n] = carry
		n++
	}
	for p[n-1] == 0 {
		n--
	}

	return int64(64*(n-1)+bits.Len64(p[n-1])) <= -e2-1-places
}

// floatPow5 gives 5^p, p from 0 to floatMaxPlaces, from the tenPow table: 10^p shifted by p bits,
// or the product of two of them.
func floatPow5(p int64) uint64 {
	if last := int64(len(tenPow)) - 1; p > last {
		return (tenPow[last] >> uint(last)) * (tenPow[p-last] >> uint(p-last))
	}
	return tenPow[p] >> uint(p)
}

// floatMaxPlaces is the largest power of five a uint64 holds: 5^27 < 2^64.
const floatMaxPlaces = 27

// floatRoundScaled gives round(mant × 2^e2 × 10^p), half away from zero, as a 128-bit integer, with
// exact integer arithmetic: p goes from -38 to floatMaxPlaces, mant × 2^e2 is under 2^107. inexact
// tells that the product was not an integer; ok is false when the result does not fit 128 bits.
func floatRoundScaled(mant uint64, e2, p int64) (hi, lo uint64, inexact, ok bool) {
	var up bool

	if p >= 0 {
		// 10^p = 5^p × 2^p: mant × 5^p fits 128 bits, the powers of two are a shift
		hi, lo = bits.Mul64(mant, floatPow5(p))

		shift := e2 + p
		if shift >= 0 {
			size := int64(bits.Len64(lo))
			if hi != 0 {
				size = 64 + int64(bits.Len64(hi))
			}
			if size+shift > 128 {
				return 0, 0, false, false
			}
			hi, lo = floatShiftLeft(hi, lo, uint(shift))
			return hi, lo, false, true
		}

		// dropping t bits: the highest one is the half, any of them makes the product inexact
		t := uint64(-shift)
		if t > 128 {
			return 0, 0, true, true
		}
		dhi, dlo := floatShiftLeft(hi, lo, uint(128-t)) // the dropped bits, at the top
		inexact = dhi|dlo != 0
		up = dhi>>63 != 0
		hi, lo = floatShiftRight(hi, lo, uint(t))
	} else {
		// the integer part of mant × 2^e2, and whether a fraction is left under it
		var sticky bool
		if e2 >= 0 {
			hi, lo = floatShiftLeft(0, mant, uint(e2))
		} else if t := uint(-e2); t < 64 {
			lo, sticky = mant>>t, mant&(1<<t-1) != 0
		} else {
			sticky = true
		}

		// divided by 10^j, in two steps beyond 10^19; with r the remainder, the quotient is rounded
		// up when r + fraction >= 10^j/2, that is r >= 10^j/2: 10^j/2 is an integer, the fraction
		// is under 1
		var r, r0 uint64
		j, d := -p, tenPow[len(tenPow)-1]
		if j >= int64(len(tenPow)) {
			hi, r0 = bits.Div64(0, hi, d)
			lo, r0 = bits.Div64(r0, lo, d)
			j -= int64(len(tenPow)) - 1
		}
		d = tenPow[j]
		hi, r = bits.Div64(0, hi, d)
		lo, r = bits.Div64(r, lo, d)
		inexact = sticky || r|r0 != 0
		up = r >= d/2
	}

	if up {
		var carry uint64
		lo, carry = bits.Add64(lo, 1, 0)
		hi += carry
	}
	return hi, lo, inexact, true
}

// floatShiftLeft and floatShiftRight shift a 128-bit integer by 0 to 128 bits.
func floatShiftLeft(hi, lo uint64, n uint) (uint64, uint64) {
	if n >= 64 {
		return lo << (n - 64), 0
	}
	return hi<<n | lo>>(64-n), lo << n
}

func floatShiftRight(hi, lo uint64, n uint) (uint64, uint64) {
	if n >= 64 {
		return 0, hi >> (n - 64)
	}
	return hi >> n, lo>>n | hi<<(64-n)
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

		if sum.Abs().GreaterThanOrEqual(item.Abs()) {
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
		if min.GreaterThanOrEqual(item) {
			min = item
		}
	}

	return min
}

// Max returns the largest Decimal that was passed in the arguments.
func Max(first Decimal, rest ...Decimal) Decimal {
	max := first
	for _, item := range rest {
		if item.GreaterThanOrEqual(max) {
			max = item
		}
	}
	return max
}

// NewFromBytes returns a new Decimal from a slice of bytes representation.
func NewFromBytes(value []byte) (Decimal, error) {
	if v, m, e, err := vmeFromBytes(value, nil); err == nil {
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

// NewFromFormattedString returns a new Decimal from a formatted string representation.
// Characters matching replRegexp are stripped from value before parsing.
//
// Useful to consume locale-formatted numbers, e.g. "$1,234.56" or "  9 876.54 €":
//
//	r := regexp.MustCompile(`[$,€\s]`)
//	d, err := NewFromFormattedString("$1,234.56", r) // d = 1234.56
//
// Note: the function only removes characters; if you need to swap a comma decimal separator for a dot,
// preprocess value yourself before calling.
func NewFromFormattedString(value string, replRegexp *regexp.Regexp) (Decimal, error) {
	return NewFromString(replRegexp.ReplaceAllString(value, ""))
}

// RequireFromString returns a new Decimal from a string representation
// or panics if NewFromString would have returned an error.
//
// Example:
//
//	d := RequireFromString("-123.45")
//	d2 := RequireFromString(".0001")
func RequireFromString(value string) Decimal {
	if v, m, e, err := vmeFromBytes([]byte(value), nil); err == nil {
		return vmeAsDecimal(v, m, e)
	} else {
		panic(err)
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *Decimal) UnmarshalJSON(b []byte) error {
	if v, m, e, err := vmeFromBytes(b, nil); err == nil {
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
		return string(d.BytesTo(nil))
	}
}

// BytesTo appends the string representation of the decimal to a slice of byte, if the decimal is Null it appends 0.
func (d Decimal) BytesTo(b []byte) []byte {
	v, m, e := d.vme()

	if b == nil {
		b = make([]byte, 0, 20)
	}

	// the maximal length of decimal representation in bytes in such conditions is 20
	return vmetBytesTo(b, v, m, e, 0, nil, true, false)
}

// StringFixed returns a rounded fixed-point string with places digits after
// the decimal point.
//
// Example:
//
//	NewFromFloat(0).StringFixed(2) // output: "0.00"
//	NewFromFloat(0).StringFixed(0) // output: "0"
//	NewFromFloat(5.45).StringFixed(0) // output: "5"
//	NewFromFloat(5.45).StringFixed(1) // output: "5.5"
//	NewFromFloat(5.45).StringFixed(2) // output: "5.45"
//	NewFromFloat(5.45).StringFixed(3) // output: "5.450"
//	NewFromFloat(545).StringFixed(-1) // output: "550"
func (d Decimal) StringFixed(places int32) string {
	return string(d.IfNull(Zero).BytesToFixed(nil, places))
}

func (d Decimal) BytesToFixed(b []byte, places int32) []byte {
	v, m, e := d.vme()

	v, m, e = vmeRound(v, m, e, places)

	if b == nil {
		b = make([]byte, 0, 20)
	}
	if places < 0 {
		places = 0
	}

	return vmetBytesTo(b, v, m, e, places, nil, true, false)
}

func (d Decimal) BytesToFixedBank(b []byte, places int32) []byte {
	v, m, e := d.vme()

	v, m, e = vmeRoundBank(v, m, e, places)

	// the maximal length of decimal representation in bytes in such conditions is 20
	if b == nil {
		b = make([]byte, 0, 20)
	}

	return vmetBytesTo(b, v, m, e, places, nil, true, false)
}

// StringFixedCash returns a Cash-rounded fixed-point string with 2 digits after the decimal point. See RoundCash for the interval semantics.
//
// Examples:
//
//	NewFromFloat(3.43).StringFixedCash(5)   // "3.45"
//	NewFromFloat(3.75).StringFixedCash(50)  // "4.00"
func (d Decimal) StringFixedCash(interval uint8) string {
	return d.RoundCash(interval).StringFixed(2)
}

// StringFixedBank returns a banker rounded fixed-point string with places digits
// after the decimal point.
//
// Example:
//
//	NewFromFloat(0).StringFixedBank(2) // output: "0.00"
//	NewFromFloat(0).StringFixedBank(0) // output: "0"
//	NewFromFloat(5.45).StringFixedBank(0) // output: "5"
//	NewFromFloat(5.45).StringFixedBank(1) // output: "5.4"
//	NewFromFloat(5.45).StringFixedBank(2) // output: "5.45"
//	NewFromFloat(5.45).StringFixedBank(3) // output: "5.450"
//	NewFromFloat(545).StringFixedBank(-1) // output: "540"
func (d Decimal) StringFixedBank(places int32) string {
	v, m, e := d.vme()

	v, m, e = vmeRoundBank(v, m, e, places)

	if places < 0 {
		return string(vmetBytesTo(make([]byte, 0, 20), v, m, e, 0, nil, true, false))
	} else {
		return string(vmetBytesTo(make([]byte, 0, 20), v, m, e, places, nil, true, false))
	}
}

// MarshalJSON implements the json.Marshaler interface.
func (d Decimal) MarshalJSON() ([]byte, error) {
	v, m, e := d.vme()

	return vmetBytesTo(nil, v, m, e, 0, nil, false, false), nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
//
// Accepts the v1 format (header byte + optional uvarint mantissa) and the v2 extension
// formats for Decimal, Weight and Length (see BINARY_FORMAT.md). When fed a Weight or
// Length extension stream, the unit is read but discarded and only the scalar value
// (m × 10^exp) is returned.
func (d *Decimal) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		return ErrFormat
	}

	// v1 normal: bit 0 of header set, uvarint mantissa follows
	if data[0]&1 != 0 {
		u := uint64(data[0]) << (decimalBitE - 1)
		u ^= 1 << (decimalBitE - 1) // clear bit 56 (was force-set to 1 by the encoder)
		m, n := binary.Uvarint(data[1:])
		if n <= 0 {
			return ErrFormat
		}
		u |= m
		if u&sign != 0 && Decimal(u) != Zero {
			*d = -Decimal(u ^ sign)
		} else {
			*d = Decimal(u)
		}
		return nil
	}

	// v1 magic: a single byte encodes the whole value (Null, Zero, NaN, ±Inf, ~0…)
	if len(data) == 1 {
		u := uint64(data[0]) << (decimalBitE - 1)
		if u&sign != 0 && Decimal(u) != Zero {
			*d = -Decimal(u ^ sign)
		} else {
			*d = Decimal(u)
		}
		return nil
	}

	// v2 extension: opcode + (optional unit uvarint for Weight/Length) + uvarint(|exp|) + uvarint(|m|)
	typeMarker, signNeg, negE, lossSet, ok := binDecodeOpcode(data[0])
	if !ok {
		return ErrFormat
	}

	rest := data[1:]

	// For Weight/Length extensions, consume the unit uvarint and ignore it (we only keep the scalar)
	if typeMarker == binExpWeight || typeMarker == binExpLength {
		_, n := binary.Uvarint(rest)
		if n <= 0 {
			return ErrFormat
		}
		rest = rest[n:]
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
	e := int64(expAbs)
	if negE {
		e = -e
	}

	*d = vmeAsDecimal(v, mAbs, e)
	return nil
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
func (d Decimal) MarshalBinary() (data []byte, err error) {
	var u uint64
	var x byte

	if d < 0 {
		u = uint64(-d)
		x = byte((u | sign) >> (decimalBitE - 1))
	} else {
		u = uint64(d)
		x = byte(u >> (decimalBitE - 1))
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
	return d.BytesTo(nil), nil
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
