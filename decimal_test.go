package decimal

import (
	"testing"

	"log"
	"math"
	"strconv"
)

func TestDoc(t *testing.T) {
	var a Decimal = -1001 // a is a Decimal of integer value -1001

	if d := a.Div(1000).Sub(1).Mul(14000).Add(14).Div(-28); d == 1000 {
		log.Printf("(((a/1000)-1)*14000+14)/-28 == 1000\n")
	} else {
		t.Errorf(`(((a/1000)-1)*14000+14)/-28 != 1000 with d = %v`, d)
	}
}

func TestIsNull(t *testing.T) {
	var d Decimal

	if !d.IsNull() {
		t.Error(`d.IsNull() = false with d not initialized`)
	}

	if d.IfNull(3) != 3 {
		t.Error(`d.IfNull() return bad value with d set to 3`)
	}

	d = Zero
	if d.IsNull() {
		t.Error(`d.IsNull() = true with d set to Zero`)
	}

	if d.IfNull(3) == 3 {
		t.Error(`d.IfNull() return bad value with d set to Zero`)
	}
}

func TestIsZero(t *testing.T) {
	var d Decimal

	if !d.IsZero() {
		t.Error(`d.IsZero() = false with d not initialized`)
	}

	if d.IsNaN() {
		t.Error(`d.IsNaN() = true with d not initialized`)
	}

	d = Zero

	if !d.IsZero() {
		t.Error(`d.IsZero() = false with d set to Zero`)
	}

	if d.IsNaN() {
		t.Error(`d.IsNaN() = true with d set to Zero`)
	}

	d = 1
	if d.IsZero() {
		t.Error(`d.IsZero() = true with d set to 1`)
	}

	if d.IsNaN() {
		t.Error(`d.IsNaN() = true with d set to 1`)
	}

	if d, err := NewFromString(`"0"`); err != nil {
		t.Errorf(`NewFromString("0") has result = %v and error = %v`, d, err)
	} else {
		if !d.IsZero() {
			t.Errorf(`NewFromString("0") has result = %v (%x) mismatch with 0 int64`, d, uint64(d))
		}
	}

	if d, err := NewFromString("6.000000"); err != nil {
		t.Errorf(`NewFromString("6.000000") has result = %v and error = %v`, d, err)
	} else {
		if d != 6 {
			t.Errorf(`NewFromString("6.000000") has result = %v (%x) mismatch with 6 int64`, d, uint64(d))
		}
	}
}

func TestIsNearZero(t *testing.T) {
	d, err := NewFromString("~0")
	if err != nil {
		t.Errorf(`NewFromString("~0") has result = %v and error = %v`, d, err)
	}

	if !d.IsZero() {
		t.Errorf(`d.IsZero() = false with d = %v (%x)`, d, uint64(d))
	}

	if d.IsNaN() {
		t.Errorf(`d.IsNaN() = true with d = %v (%x)`, d, uint64(d))
	}

	d, err = NewFromString("1.234e-40")
	if err != nil {
		t.Errorf(`NewFromString("1.234e-40") has result = %v and error = %v`, d, err)
	}

	if !d.IsZero() {
		t.Errorf(`d.IsZero() = false with d = %v (%x)`, d, uint64(d))
	}

	if d.IsNaN() {
		t.Errorf(`d.IsNaN() = true with d = %v (0x%016x)`, d, uint64(d))
	}
}

func TestIsExact(t *testing.T) {
	d := Zero

	if !d.IsExact() {
		t.Error(`d.IsExact() = false with d set to decimal.Zero`)
	}

	d = 1
	if !d.IsExact() {
		t.Error(`d.IsExact() = false with d set to 1`)
	}

	d = d.Div(3)
	if d.IsExact() {
		t.Error(`d.IsExact() = true with d set to 1/3`)
	}
}

func TestFloat64(t *testing.T) {
	d := New(14411518, 0)

	if f, exact := d.Float64(); f != 14411518 || !exact {
		t.Errorf(`d.Float64() does not return right float64 14411518 with d = %v, d.Float64() = %v, %v`, d, f, exact)
	}

	d = New(144115188075855871, 3)

	if f := d.InexactFloat64(); f != 144115188075855871000 {
		t.Errorf(`d.InexactFloat64() does not return right float64 144115188075855871000 with d = %v, d.InexactFloat64() = %v`, d, f)
	}

	d = New(144115188075855871, -3)

	if f := d.InexactFloat64(); f != 144115188075855.871 {
		t.Errorf(`d.InexactFloat64() does not return right float64 144115188075855.871 with d = %v, d.InexactFloat64() = %v`, d, f)
	}

	d = PositiveInfinity
	if f := d.InexactFloat64(); f != math.Inf(0) {
		t.Errorf(`d.InexactFloat64() does not return +Inf with d = %v, d.InexactFloat64() = %v`, d, f)
	}

	d = -d
	if f := d.InexactFloat64(); f != math.Inf(-1) {
		t.Errorf(`d.InexactFloat64() does not return -Inf with d = %v, d.InexactFloat64() = %v`, d, f)
	}

	d = NaN
	if f := d.InexactFloat64(); !math.IsNaN(f) {
		t.Errorf(`d.InexactFloat64() does not return NaN with d = %v, d.InexactFloat64() = %v`, d, f)
	}
}

func TestIntPartErr(t *testing.T) {
	d := New(144115188075855871, 3)

	if i, err := d.IntPartErr(); err == nil {
		t.Errorf(`d.IntPartErr() does not return error with out of range integer conversion d = %v, d.IntPartErr() = %v`, d, i)
	}

	d = PositiveInfinity
	if i, err := d.IntPartErr(); err == nil {
		t.Errorf(`d.IntPartErr() does not return error with out of range integer conversion d = %v, d.IntPartErr() = %v`, d, i)
	}

	d = -d
	if i, err := d.IntPartErr(); err == nil {
		t.Errorf(`d.IntPartErr() does not return error with out of range integer conversion d = %v, d.IntPartErr() = %v`, d, i)
	}

	d = NaN
	if i, err := d.IntPartErr(); err == nil {
		t.Errorf(`d.IntPartErr() does not return error with out of range integer conversion d = %v, d.IntPartErr() = %v`, d, i)
	}
}

func TestNewNilOrNullFromString(t *testing.T) {
	nils := [...]string{"", "nil", "niL", "nIl", "nIL", "Nil", "NiL", "NIl", "NIL", "null", "nulL", "nuLl", "nuLL", "nUll", "nUlL", "nULl", "nULL", "Null", "NulL", "NuLl", "NuLL", "NUll", "NUlL", "NULl", "NULL"}
	for _, s := range nils {
		if d, err := NewFromString(s); err != nil {
			t.Errorf(`NewFromString("%s") returns err = %s`, s, err)
		} else {
			if d != 0 {
				t.Errorf(`d should be unitialized for %s, d = %v, d.IsNull() = %t`, s, d, d.IsNull())
			}

		}
	}
}

func TestIsPositiveOrNegative(t *testing.T) {
	v := [...]Decimal{Null, Zero, NearZero}
	for _, d := range v {
		if d.IsPositive() {
			t.Errorf(`d should not be positive, d = %v, d.IsPositive() = %t`, d, d.IsPositive())
		}

		if d.IsNegative() {
			t.Errorf(`d should not be negative, d = %v, d.IsNegative() = %t`, d, d.IsNegative())
		}
	}
}

func TestNewFromString(t *testing.T) {
	d, err := NewFromString("0.00123")
	if err != nil {
		t.Errorf(`NewFromString("0.00123") has result = %v and error = %v`, d, err)
	}

	if !d.IsExact() {
		t.Error(`d.IsExact() = false with d set to 0.00123`)
	}

	d, err = NewFromString("~123")
	if err != nil {
		t.Errorf(`NewFromString("~123") has result = %v and error = %v`, d, err)
	}

	if d.IsExact() {
		t.Errorf(`d.IsExact() = true with d = %v (%x)`, d, uint64(d))
	}
}

func TestNewFromStringZeros(t *testing.T) {
	zeros := [...]string{"0", "00", "000", "0.0", ".0", ".00", ".000", "0.0e10", "no", "No", "nO", "off", "Off", "OFf", "OfF", "oFF", "oFf", "ofF", "OFF"}
	for _, s := range zeros {
		if d, err := NewFromString(s); err != nil {
			t.Errorf(`NewFromString("%s") returns err = %s`, s, err)
		} else {
			if d != Zero {
				t.Errorf(`s=%s should be exact 0, d = %v (%x), d == 0 = %t`, s, d, uint64(d), d == Zero)
			}

			if d = RequireFromString(s); d != Zero {
				t.Errorf(`RequireFromString("%s") returns non Zero value %v`, s, d)
			}

			if d, err = NewFromString("-" + s); err != nil {
				t.Errorf(`NewFromString("%s") returns err = %s`, s, err)
			}

			if d != Zero {
				t.Errorf(`d = NewFromString("-%s") should be near 0, d = %v (%x) , d.Equal(0) = %t`, s, d, uint64(d), d.Equal(0))
			}

			if d, err = NewFromString("~" + s); err != nil {
				t.Errorf(`NewFromString("%s") returns err = %s`, s, err)
			}

			if !d.Equal(0) {
				t.Errorf(`d = NewFromString("~%s") should be near 0, d = %v (%x) , d.Equal(0) = %t`, s, d, uint64(d), d.Equal(0))
			}

			if d.String() != "~0" {
				t.Errorf(`d = NewFromString("~%s") should be ~0 back in string, d = %v (%x) , d.String() == "~0" is %t`, s, d, uint64(d), d.String() == "~0")
			}

			if d, err = NewFromString("~-" + s); err != nil {
				t.Errorf(`NewFromString("%s") returns err = %s`, s, err)
			}

			if !d.Equal(0) {
				t.Errorf(`d = NewFromString("~-%s") should be near 0, d = %v (%x) , d.Equal(0) = %t`, s, d, uint64(d), d.Equal(0))
			}

			if d.String() != "-~0" {
				t.Errorf(`d = NewFromString("-~%s") should be -~0 back in string, d = %v (%x) , d.String() == "-~0" is %t`, s, d, uint64(d), d.String() == "-~0")
			}
		}
	}
}

func TestNewFromStringNans(t *testing.T) {
	nans := [...]string{"nan", "naN", "nAn", "nAN", "Nan", "NaN", "NAn", "NAN"}
	for _, s := range nans {
		if d, err := NewFromString(s); err != nil {
			t.Errorf(`NewFromString("%s") returns err = %s`, s, err)
		} else {
			if d != NaN {
				t.Errorf(`d should be exact NaN, d = %v, d.IsNaN() = %t`, d, d.IsNaN())
			}

			if !d.IsNaN() {
				t.Errorf(`d should be NaN, d = %v, d.IsNaN() = %t`, d, d.IsNaN())
			}
		}
	}
}

func TestNew(t *testing.T) {
	if d := New(0, 0); d != Zero {
		t.Errorf(`New(0, 0) should be Zero, d = %v`, d)
	}

	if d := New(0, 42); d != Zero {
		t.Errorf(`New(0, 42) should be Zero, d = %v`, d)
	}

	if d := New(42, 0); d != 42 {
		t.Errorf(`New(42, 0) should be 42, d = %v`, d)
	}

	if d := New(-42, 0); d != -42 {
		t.Errorf(`New(-42, 0) should be -42, d = %v`, d)
	}
}

func TestNewFromInt(t *testing.T) {
	if d := NewFromInt(0); d != Zero {
		t.Errorf(`NewFromInt(0) should be Zero, d = %v`, d)
	}

	if d := NewFromInt(42); d != 42 {
		t.Errorf(`NewFromInt(42) should be 42, d = %v`, d)
	}

	if d := NewFromInt32(0); d != Zero {
		t.Errorf(`NewFromInt32(0) should be Zero, d = %v`, d)
	}

	if d := NewFromInt32(42); d != 42 {
		t.Errorf(`NewFromInt32(42) should be 42, d = %v`, d)
	}
}

func TestNewFromFloat(t *testing.T) {
	if d := NewFromFloat(0); d != Zero {
		t.Errorf(`NewFromFloat(0) should be Zero, d = %v`, d)
	}

	if d := NewFromFloat32(0); d != Zero {
		t.Errorf(`NewFromFloat32(0) should be Zero, d = %v`, d)
	}

	if d := NewFromFloat32(-14.999).Round(3); d != New(-14999, -3) {
		t.Errorf(`NewFromFloat(-14.999) should be -14.999, d = %v`, d)
	}

	if d := NewFromFloat32(123456); d != 123456 {
		t.Errorf(`NewFromFloat(123456) should be 123456, d = %v`, d)
	}

	if d := NewFromFloat32(123.456).Round(3); d != New(123456, -3) {
		t.Errorf(`NewFromFloat(123.456) should be 123.456, d = %v`, d)
	}

	if d := NewFromFloat32(0.01).Round(2); d != New(1, -2) {
		t.Errorf(`NewFromFloat(0.01) should be 0.01, d = %v`, d)
	}

	if d := NewFromFloat(-14.999); d != New(-14999, -3) {
		t.Errorf(`NewFromFloat(-14.999) should be -14.999, d = %v`, d)
	}

	if d := NewFromFloat(123456); d != 123456 {
		t.Errorf(`NewFromFloat(123456) should be 123456, d = %v`, d)
	}

	if d := NewFromFloat(123.456789).Round(6); d != New(123456789, -6) {
		t.Errorf(`NewFromFloat(123.456) should be ~= 123.456789, d = %v`, d)
	}

	if d := NewFromFloat(0.01); d != New(1, -2) {
		t.Errorf(`NewFromFloat(0.01) should be 0.01, d = %v`, d)
	}

	if d := NewFromFloat(1.123e-10); !d.Equal(New(1123, -13)) {
		t.Errorf(`NewFromFloat(1.123e-10) should be 1.123e-10, d = %v`, d)
	}

	if d := NewFromFloat(1.1e-70); d != NearPositiveZero {
		t.Errorf(`NewFromFloat(1.1e-70) should be near +0 as too small, d = %v`, d)
	}
}

func TestNewFromFloatWithExponent(t *testing.T) {
	if d := NewFromFloatWithExponent(0, -10); d != Zero {
		t.Errorf(`NewFromFloatWithExponent(0) should be Zero, d = %v`, d)
	}
	if d := NewFromFloatWithExponent(-14.999, 3); d != New(-14999, -3) {
		t.Errorf(`NewFromFloatWithExponent(-14.999, 3) should be -14.999, d = %v`, d)
	}
	if d := NewFromFloatWithExponent(-14.999, 5); d != New(-14999, -3) {
		t.Errorf(`NewFromFloatWithExponent(-14.999, 5) should be -14.999, d = %v`, d)
	}
	if d := NewFromFloatWithExponent(-14.999, 2); d != -15 {
		t.Errorf(`NewFromFloatWithExponent(-14.999, 2) should be -15, d = %v`, d)
	}
	if d := NewFromFloatWithExponent(0.025, 2); d != New(3, -2) {
		t.Errorf(`NewFromFloatWithExponent(0.025, 2) should be 0.03, d = %v`, d)
	}
	if d := NewFromFloatWithExponent(-0.015, 2); d != New(-1, -2) {
		t.Errorf(`NewFromFloat(0.015, 2) should be -0.01, d = %v`, d)
	}
}

func TestNewOneOrOnOrYesFromString(t *testing.T) {
	ones := [...]string{"1", "01", "001", "1.0", "1.00", "1.000", "1.0000e0", "0.001e3", "on", "On", "oN", "yes", "Yes", "yEs", "yeS", "yES", "YeS", "YEs", "YES"}
	for _, s := range ones {
		if d, err := NewFromString(s); err != nil {
			t.Errorf(`NewFromString("%s") returns err = %s`, s, err)
		} else {
			if d != 1 {
				t.Errorf(`d should be exact 1, d = %v, d == 1 = %t`, d, d == 1)
			}

			if !d.IsPositive() {
				t.Errorf(`d should be positive, d = %v, d.IsPositive() = %t`, d, d.IsPositive())
			}

			if d.IsNegative() {
				t.Errorf(`d should not be negative, d = %v, d.IsNegative() = %t`, d, d.IsNegative())
			}

			if d, err = NewFromString("~" + s); err != nil {
				t.Errorf(`NewFromString("%s") returns err = %s`, s, err)
			}

			if !d.Equal(1) {
				t.Errorf(`d = NewFromString("~%s") should be near 1, d = %v, d.Equal(1) = %t`, s, d, d.Equal(1))
			}

			if !d.IsPositive() {
				t.Errorf(`d should be positive, d = %v, d.IsPositive() = %t`, d, d.IsPositive())
			}

			if d.IsNegative() {
				t.Errorf(`d should not be negative, d = %v, d.IsNegative() = %t`, d, d.IsNegative())
			}
		}
	}
}

func TestNewInfiniteFromString(t *testing.T) {
	infs := [...]string{"inf", "inF", "iNf", "iNF", "Inf", "InF", "INf", "INF", "+inf", "+inF", "+iNf", "+iNF", "+Inf", "+InF", "+INf", "+INF", "1E1000"}
	for _, s := range infs {
		if d, err := NewFromString(s); err != nil {
			t.Errorf(`NewFromString("%s") returns err = %s`, s, err)
		} else {
			if d != PositiveInfinity {
				t.Errorf(`d should be exact +Inf, d = %v, d == +Inf = %t`, d, d == PositiveInfinity)
			}

			if !d.IsPositive() {
				t.Errorf(`d should be positive, d = %v, d.IsPositive() = %t`, d, d.IsPositive())
			}

			if d.IsNegative() {
				t.Errorf(`d should not be negative, d = %v, d.IsNegative() = %t`, d, d.IsNegative())
			}
		}
	}
	minfs := [...]string{"-inf", "-inF", "-iNf", "-iNF", "-Inf", "-InF", "-INf", "-INF", "-1.234E+500"}
	for _, s := range minfs {
		if d, err := NewFromString(s); err != nil {
			t.Errorf(`NewFromString("%s") returns err = %s`, s, err)
		} else {
			if d != NegativeInfinity {
				t.Errorf(`d should be exact -Inf, d = %v, d == -Inf = %t`, d, d == NegativeInfinity)
			}

			if d.IsPositive() {
				t.Errorf(`d should not be positive, d = %v, d.IsPositive() = %t`, d, d.IsPositive())
			}

			if !d.IsNegative() {
				t.Errorf(`d should be negative, d = %v, d.IsNegative() = %t`, d, d.IsNegative())
			}
		}
	}

	errs := [...]string{"0.a", ".123e--19", "azerty", "-mCF", "-+23"}
	for _, s := range errs {
		if _, err := NewFromString(s); err == nil {
			t.Errorf(`NewFromString("%s") returns no error`, s)
		}
	}
}

func TestIsEqual(t *testing.T) {
	d, err := NewFromString("0.001")
	if err != nil {
		t.Errorf(`NewFromString("0.001") has result = %v and error = %v`, d, err)
	}

	d = d.Mul(1000)

	if d != 1 {
		t.Errorf(`d*1000 = %v and should be equal to 1, d == 1 is %t`, d, d == 1)
	}

	if New(1, -3).Mul(1000) != 1 {
		t.Errorf(`New(1, -3) * 1000 = %v and should be equal to 1, d == 1 is %t`, New(1, -3).Mul(1000), New(1, -3).Mul(1000) == 1)
	}

	d = Null
	if !d.Equal(0) {
		t.Errorf(`Null should be == to 0, d == 0 is %t`, d.Equal(0))
	}
	if !d.GreatherThanOrEqual(0) {
		t.Errorf(`Null should be >= to 0, d == 0 is %t`, d.GreatherThanOrEqual(0))
	}

	d = Zero
	if !d.Equal(0) {
		t.Errorf(`Zero should be equal to 0, d == 0 is %t`, d.Equal(0))
	}
	if !d.GreatherThanOrEqual(0) {
		t.Errorf(`Zero should be >= to 0, d == 0 is %t`, d.GreatherThanOrEqual(0))
	}
}

func TestRoundZeros(t *testing.T) {
	zeros := [...]string{"0", "~0", "-~0", "+~0", "0.4999999999", "~0.49", "0.3567433445234", "1e-10", "-0.333", "-0.5", "~-0.5"}
	for _, s := range zeros {
		d := RequireFromString(s)

		if d.Round(0) != Zero {
			t.Errorf(`d = %v rounded to 0 decimals should be Zero, d.Round(0) = %v`, d, d.Round(0))
		}
	}

	zeros_ceil := [...]string{"0", "~0", "-~0", "+~0", "-0.3567433445234", "-1e-10", "-0.333", "-0.5", "~-0.9999"}
	for _, s := range zeros_ceil {
		d := RequireFromString(s)

		if d.RoundCeil(0) != Zero {
			t.Errorf(`d = %v rounded ceil to 0 decimals should be Zero, d.RoundCeil(0) = %v`, d, d.RoundCeil(0))
		}
	}

	zeros_floor := [...]string{"0", "~0", "-~0", "+~0", "0.3567433445234", "1e-10", "0.333", "0.5", "~0.9999"}
	for _, s := range zeros_floor {
		d := RequireFromString(s)

		if d.RoundFloor(0) != Zero {
			t.Errorf(`d = %v rounded floor to 0 decimals should be Zero, d.RoundFloor(0) = %v`, d, d.RoundFloor(0))
		}
	}
}

func TestRoundSmalls(t *testing.T) {
	smalls := [...]string{"0.001", "~.001", "0.0014", "~0.00123", ".00116457344", "0.0005", "~0.0009"}
	for _, s := range smalls {
		d := RequireFromString(s)

		if d.Round(3) != New(1, -3) {
			t.Errorf(`d = %v rounded to 3 decimals should be 0.001, d.Round(3) = %v`, d, d.Round(3))
		}
	}

	ones := [...]string{"1", "1.000000000001", "1.0000000000000000000000000000000000000000000000000001", "~1", "0.5", "~.5", "0.99999", "1.4999999999999"}
	for _, s := range ones {
		d := RequireFromString(s)

		if d.Round(0) != 1 {
			t.Errorf(`d = %v rounded to 0 decimals should be 1, d.Round(0) = %v`, d, d.Round(0))
		}
	}

	ones_ceil := [...]string{"1", "1.00000000000000000000", "~1", "0.5", "~.5", "0.99999"}
	for _, s := range ones_ceil {
		d := RequireFromString(s)

		if d.RoundCeil(0) != 1 {
			t.Errorf(`d = %v rounded ceil to 0 decimals should be 1, d.RoundCeil(0) = %v`, d, d.RoundCeil(0))
		}
	}

	ones_floor := [...]string{"1", "1.000000000000001", "~1", "1.5", "~1.5", "1.99999"}
	for _, s := range ones_floor {
		d := RequireFromString(s)

		if d.RoundFloor(0) != 1 {
			t.Errorf(`d = %v rounded floor to 0 decimals should be 1, d.RoundFloor(0) = %v`, d, d.RoundFloor(0))
		}
	}
}

func TestRoundTens(t *testing.T) {
	tens := [...]string{"10", "10.4999", "9.5", "~10"}
	for _, s := range tens {
		d := RequireFromString(s)

		if d.Round(-1) != 10 {
			t.Errorf(`d rounded to -1 decimals should be 10, d = %v, d.Round(-1) = %v`, d, d.Round(-1))
		}
	}

	tens_ceil := [...]string{"10", "9.00000001", "9.9999", "9.5", "~10"}
	for _, s := range tens_ceil {
		d := RequireFromString(s)

		if d.RoundCeil(-1) != 10 {
			t.Errorf(`d rounded to -1 decimals should be 10, d = %v, d.RoundCeil(-1) = %v`, d, d.RoundCeil(-1))
		}
	}

	five_dot_five := [...]string{"5.5", "5.45", "5.54", "~5.50"}
	for _, s := range five_dot_five {
		d := RequireFromString(s)

		if d.Round(1) != New(55, -1) {
			t.Errorf(`d rounded to 1 decimals should be 5.5, d = %v, d.Round(-1) = %v`, d, d.Round(1))
		}
	}
}

func TestRoundCeil(t *testing.T) {
	// RoundCeil
	if d := New(545, 0).RoundCeil(-2); d != 600 {
		t.Errorf(`545 rounded ceil to -2 decimals should be 600 and not %v`, d)
	}
	if d := New(500, 0).RoundCeil(-2); d != 500 {
		t.Errorf(`500 rounded ceil to -2 decimals should be 500 and not %v`, d)
	}
	if d := New(11001, -4).RoundCeil(2); d != New(111, -2) {
		t.Errorf(`1.1001 rounded ceil to 2 decimals should be 1.11 and not %v`, d)
	}
	if d := New(-14, -1).RoundCeil(1); d != New(-14, -1) {
		t.Errorf(`-1.4 rounded ceil to 1 decimal should be -1.4 and not %v`, d)
	}
	if d := New(-1454, -3).RoundCeil(1); d != New(-14, -1) {
		t.Errorf(`-1.454 rounded ceil to 1 decimal should be -1.4 and not %v`, d)
	}
	if d := New(1, -3).RoundCeil(1); d != New(1, -1) {
		t.Errorf(`0.001 rounded ceil to 1 decimal should be 0.1 and not %v`, d)
	}
	if d := New(-1, -3).RoundCeil(1); d != Zero {
		t.Errorf(`-0.001 rounded ceil to 1 decimal should be 0 and not %v`, d)
	}

	// RoundCeil on magic Decimals
	if d := NaN.RoundCeil(1); !d.IsNaN() {
		t.Errorf(`NaN rounded ceil to 1 decimal should be NaN and not %v`, d)
	}
	if d := PositiveInfinity.RoundCeil(1); d != PositiveInfinity {
		t.Errorf(`+Inf rounded ceil to 1 decimal should be +Inf and not %v`, d)
	}
	if d := NegativeInfinity.RoundCeil(1); d != NegativeInfinity {
		t.Errorf(`-Inf rounded ceil to 1 decimal should be -Inf and not %v`, d)
	}
	if d := NearZero.RoundCeil(1); d != Zero {
		t.Errorf(`~0 rounded ceil to 1 decimal should be exactly 0 and not %v`, d)
	}
	if d := NearPositiveZero.RoundCeil(1); d != Zero {
		t.Errorf(`~+0 rounded ceil to 1 decimal should be exactly 0 and not %v`, d)
	}
	if d := NearNegativeZero.RoundCeil(1); d != Zero {
		t.Errorf(`~-0 rounded ceil to 1 decimal should be exactly 0 and not %v`, d)
	}
}

func TestRoundFloor(t *testing.T) {
	// RoundFloor
	if d := New(545, 0).RoundFloor(-2); d != 500 {
		t.Errorf(`545 rounded floor to -2 decimals should be 600 and not %v`, d)
	}
	if d := New(500, 0).RoundFloor(-2); d != 500 {
		t.Errorf(`500 rounded floor to -2 decimals should be 500 and not %v`, d)
	}
	if d := New(11001, -4).RoundFloor(2); d != New(110, -2) {
		t.Errorf(`1.1001 rounded floor to 2 decimals should be 1.11 and not %v`, d)
	}
	if d := New(-14, -1).RoundFloor(1); d != New(-14, -1) {
		t.Errorf(`-1.4 rounded floor to 1 decimal should be -1.4 and not %v`, d)
	}
	if d := New(-1454, -3).RoundFloor(1); d != New(-15, -1) {
		t.Errorf(`-1.454 rounded floor to 1 decimal should be -1.5 and not %v`, d)
	}
	if d := New(1, -3).RoundFloor(1); d != Zero {
		t.Errorf(`0.001 rounded floor to 1 decimal should be 0 and not %v`, d)
	}
	if d := New(-1, -3).RoundFloor(1); d != New(-1, -1) {
		t.Errorf(`-0.001 rounded floor to 1 decimal should be -0.1 and not %v`, d)
	}

	// RoundFloor on magic Decimals
	if d := NaN.RoundFloor(1); !d.IsNaN() {
		t.Errorf(`NaN rounded ceil to 1 decimal should be NaN and not %v`, d)
	}
	if d := PositiveInfinity.RoundFloor(1); d != PositiveInfinity {
		t.Errorf(`+Inf rounded ceil to 1 decimal should be +Inf and not %v`, d)
	}
	if d := NegativeInfinity.RoundFloor(1); d != NegativeInfinity {
		t.Errorf(`-Inf rounded ceil to 1 decimal should be -Inf and not %v`, d)
	}
	if d := NearZero.RoundFloor(1); d != Zero {
		t.Errorf(`~0 rounded ceil to 1 decimal should be exactly 0 and not %v`, d)
	}
	if d := NearPositiveZero.RoundFloor(1); d != Zero {
		t.Errorf(`~+0 rounded ceil to 1 decimal should be exactly 0 and not %v`, d)
	}
	if d := NearNegativeZero.RoundFloor(1); d != Zero {
		t.Errorf(`~-0 rounded ceil to 1 decimal should be exactly 0 and not %v`, d)
	}
}

func TestRoundBank(t *testing.T) {
	if d := New(545, 0).RoundBank(-1); d != 540 {
		t.Errorf(`545 rounded bank to -1 decimals should be 540 and not %v`, d)
	}
	if d := New(546, 0).RoundBank(-1); d != 550 {
		t.Errorf(`546 rounded bank to -1 decimals should be 550 and not %v`, d)
	}
	if d := New(555, 0).RoundBank(-1); d != 560 {
		t.Errorf(`555 rounded bank to -1 decimals should be 560 and not %v`, d)
	}

	if d := NewFromFloat(5.45).RoundBank(1); d != New(54, -1) {
		t.Errorf(`5.45 rounded bank to 1 decimals should be 5.4 and not %v`, d)
	}
}

func TestAdd(t *testing.T) {
	d1, err := NewFromString("123.456")
	if err != nil {
		t.Errorf(`NewFromString("123.456") has result = %v and error = %v`, d1, err)
	}

	d2, err := NewFromString("0.544")
	if err != nil {
		t.Errorf(`NewFromString("0.544") has result = %v and error = %v`, d2, err)
	}

	d := d1.Add(d2)

	if d != 124 {
		t.Errorf(`d1+d2 = %v and should be equal to 124, d == 124 is %t`, d, d == 124)
	}

	d = d.Sub(d1)

	if d != d2 {
		t.Errorf(`d-d1 = %v and should be equal to d2, d == d2 is %t`, d, d == d2)
	}

	d = New(1, 18) // d is now too big to be a real integer
	d1 = d.Add(1)  // d1 is now approximate of d
	d2 = d1.Sub(1) // d2 is still approximate of d

	if d == d2 || d == d1 {
		t.Errorf(`addition should have different result, d = 0x%016x, d1 = 0x%016x, d2 = 0x%016x`, uint64(d), uint64(d1), uint64(d2))
	}
	if !d.Equal(d2) {
		t.Errorf(`addition should have been approximative, d = 0x%016x, d1 = 0x%016x, d2 = 0x%016x`, uint64(d), uint64(d1), uint64(d2))
	}

	d = New(1, 30)  // d is now too big to be a real integer and mantissa could not be 1 as exposant is too high
	d1 = d.Add(100) // d1 is now approximate of d
	d2 = d1.Sub(1)  // d2 is still approximate of d

	if d == d1 || d == d2 {
		t.Errorf("addition should have different result:\n   d  = %v\n d1 = %v\n d2 = %v", d, d1, d2)
	}
	if !d.Equal(d2) {
		t.Errorf("addition should have been approximative:\n   d = %v\n d1 = %v\n d2 = %v", d, d1, d2)
	}

}

func TestAddMagicZeros(t *testing.T) {
	d := New(123456, -3)

	if d.String() != "123.456" {
		t.Errorf(`New(123456, -3) has result = %v and should be equal to 123.456`, d)
	}

	d0 := d.Add(Null)
	if d0.String() != "123.456" {
		t.Errorf(`123.456 + Null has result = %v`, d0)
	}

	d1 := d0.Add(Zero)
	if d1.String() != "123.456" {
		t.Errorf(`123.456 + 0 has result = %v`, d1)
	}

	d2 := d1.Add(NearZero)
	if d2.String() != "~123.456" {
		t.Errorf(`123.456 + ~0 has result = %v`, d2)
	}

	d3 := d1.Add(NearPositiveZero)
	if d3.String() != "~123.456" || d3 != d2 {
		t.Errorf(`123.456 + ~+0 has result = %v`, d3)
	}

	d4 := d1.Add(NearNegativeZero)
	if d4.String() != "~123.456" || d4 != d2 {
		t.Errorf(`123.456 + ~-0 has result = %v`, d4)
	}
}

func TestAddMagicNans(t *testing.T) {
	d1 := New(123456, -3)
	d5 := d1.Add(PositiveInfinity)
	if d5 != PositiveInfinity {
		t.Errorf(`123.456 + +Inf has result = %v`, d5)
	}

	d6 := d1.Add(NegativeInfinity)
	if d6 != NegativeInfinity {
		t.Errorf(`123.456 + -Inf has result = %v`, d6)
	}

	d7 := d1.Add(NaN)
	if d7 != NaN {
		t.Errorf(`123.456 + NaN has result = %v`, d7)
	}

	if d5.Add(PositiveInfinity) != d5 {
		t.Errorf(`+Inf + +Inf has result = %v`, d5.Add(PositiveInfinity))
	}

	if d5.Add(NegativeInfinity) != NaN {
		t.Errorf(`+Inf + -Inf has result = %v`, d5.Add(NegativeInfinity))
	}

	if d6.Add(NegativeInfinity) != d6 {
		t.Errorf(`-Inf + -Inf has result = %v`, d6.Add(NegativeInfinity))
	}

	if d6.Add(PositiveInfinity) != NaN {
		t.Errorf(`-Inf + +Inf has result = %v`, d6.Add(PositiveInfinity))
	}
}

func TestMulMagic(t *testing.T) {
	d1 := New(123456, -3)

	d2 := d1.Mul(NearZero)
	if d2 != NearZero {
		t.Errorf(`123.456 * ~0 has result = %v`, d2)
	}
	if d2.Mul(-1) != NearZero {
		t.Errorf(`123.456 * ~0 * -~0 has result = %v`, d2.Mul(-1))
	}
	if !d2.Mul(NegativeInfinity).IsNaN() {
		t.Errorf("~0 * Inf should be NaN, result = %v", d2.Mul(NegativeInfinity))
	}

	d3 := d1.Mul(NearPositiveZero)
	if d3 != NearPositiveZero || d3.String() != "+~0" {
		t.Errorf(`123.456 * ~+0 has result = %v`, d3)
	}

	d4 := d1.Mul(NearNegativeZero)
	if d4 != NearNegativeZero || d4.String() != "-~0" {
		t.Errorf(`123.456 * ~-0 has result = %v`, d4)
	}

	d5 := d1.Mul(PositiveInfinity)
	if d5 != PositiveInfinity {
		t.Errorf(`123.456 * +Inf has result = %v`, d5)
	}

	d6 := d1.Mul(NegativeInfinity)
	if d6 != NegativeInfinity {
		t.Errorf(`123.456 + -Inf has result = %v`, d6)
	}

	d7 := d1.Mul(NaN)
	if !d7.IsNaN() {
		t.Errorf(`123.456 * NaN has result = %v`, d7)
	}

	if d5.Mul(PositiveInfinity) != d5 {
		t.Errorf(`+Inf * +Inf has result = %v`, d5.Mul(PositiveInfinity))
	}

	if d5.Mul(NegativeInfinity) != d6 {
		t.Errorf(`+Inf * -Inf has result = %v`, d5.Mul(NegativeInfinity))
	}

	if d6.Mul(NegativeInfinity) != d5 {
		t.Errorf(`-Inf * -Inf has result = %v`, d6.Mul(NegativeInfinity))
	}

	if d6.Mul(PositiveInfinity) != d6 {
		t.Errorf(`-Inf * +Inf has result = %v`, d6.Mul(PositiveInfinity))
	}
}

func TestMul(t *testing.T) {
	d1 := New(1230, -3)
	d2 := New(999, 2)
	d := d1.Mul(d2)

	if d != 122877 {
		t.Errorf(`d1*d2 = %v and should be equal to 122877, d == 122877 is %t`, d, d == 122877)
	}

	d = d.Div(d1)

	if d != d2 {
		t.Errorf(`d/d1 = %v and should be equal to d2, d == d2 is %t`, d, d == d2)
	}

	d = New(123456789012345678, 0) // adding one more digit will cause precision loss
	if d.String() != "123456789012345678" {
		t.Errorf("big number not correctly seen, d = %v", d)
	}
	d1 = d.Add(6543210987654321)
	if d1.String() != "129999999999999999" {
		t.Errorf("big number not correctly seen, d1 = %v", d1)
	}
	d1 = d1.Add(1)
	if d1.String() != "130000000000000000" {
		t.Errorf("big number not correctly seen, d1 = %v", d1)
	}
	d1 = d1.Add(1)
	if d1.String() != "130000000000000001" {
		t.Errorf("big number not correctly seen, d1 = %v", d1)
	}
	d2 = d1.Mul(111111111)
	if d2.String() != "~14444444430000000000000000" {
		t.Errorf("big number not correctly seen, d2 = %v", d2)
	}
	d2 = d2.Div(10000000000000000).Round(0)
	if d2 != 1444444443 {
		t.Errorf("big number not correctly seen, d2 = %v", d2)
	}
	d2 = d2.Mul(98765432)
	if d2 != 142661179412894376 {
		t.Errorf("big number not correctly seen, d2 = %v", d2)
	}
}

func TestQuoRem(t *testing.T) {
	d1 := NewFromInt(4)
	d2 := NewFromInt(3)
	q, r := d1.QuoRem(d2, 3)
	if q != New(1333, -3) && r != New(1, -3) {
		t.Errorf("4.QuoRem(3, 3) should be equal to 1.333, 0.001 but quo = %v and rem = %v", q, r)
	}

	d1 = New(4111, -3)
	d2 = 3
	q, r = d1.QuoRem(d2, 3)
	if q != New(1370, -3) && r != New(1, -3) {
		t.Errorf("(4,111).QuoRem(3, 3) should be equal to 1.370, 0.001 but quo = %v and rem = %v", q, r)
	}

	d1 = New(4111, -2)
	d2 = 3
	q, r = d1.QuoRem(d2, 3)
	if q != New(13703, -3) && r != New(1, -3) {
		t.Errorf("(41,11).QuoRem(3, 3) should be equal to 13.703, 0.001 but quo = %v and rem = %v", q, r)
	}

	d1 = New(4235, -2)
	d2 = New(55, -1)
	q, r = d1.QuoRem(d2, 1)
	if q != New(77, -1) && r != Zero {
		t.Errorf("(42.35).QuoRem(5.5, 1) should be equal to 7.7, 0 but quo = %v and rem = %v", q, r)
	}
	q, r = d1.QuoRem(d2, 0)
	if q != 7 && r != New(385, -2) {
		t.Errorf("(42.35).QuoRem(5.5, 0) should be equal to 7, 3.85 but quo = %v and rem = %v", q, r)
	}
	log.Printf("%v = %v * %v, remainder = %v", d1, d2, q, r)

	d1 = New(14569568235, -3)
	d2 = New(5607988, -5)
	q, r = d1.QuoRem(d2, 0)
	/* if q != 7 && r != New(385, -2) {
		t.Errorf("(42.35).QuoRem(5.5, 0) should be equal to 7, 3.85 but quo = %v and rem = %v", q, r)
	} */
	log.Printf("%v = %v * %v, remainder = %v", d1, d2, q, r)
}

func TestBigNumber(t *testing.T) {
	d1 := New(144115188075855871, 2)
	d2 := d1.Mul(d1)

	if d2 != PositiveInfinity {
		t.Errorf(`d1*d1 should be equal to +Inf, value is %v`, d2)
	}
}

func TestDiv1_3(t *testing.T) {
	d1 := New(1, 0)
	d2 := NewFromInt(2)
	d3 := NewFromInt32(3)
	d := d1.Div(d3)

	log.Printf(`1/3 = %v`, d)

	if d.IsExact() {
		t.Errorf(`1/3 = %v and should not be exact`, d)
	}

	d = d.Add(d2.Div(d3))

	log.Printf(`1/3 + 2/3 = %v`, d)

	if d.IsExact() {
		t.Errorf(`1/3 + 2/3 = %v and should not be exact`, d)
	}

	d = d.Round(0).Div(New(5, -1))
	if !d.IsExact() || d != 2 {
		t.Errorf(`(1/3 + 2/3)/0.5 = %v and should not be exact`, d)
	}

	if z := d.Div(PositiveInfinity); z != NearPositiveZero {
		t.Errorf(`2/+Inf = %v and should be near positive zero`, z)
	}
	if z := d.Div(NegativeInfinity); z != NearNegativeZero {
		t.Errorf(`2/-Inf = %v and should be near negative zero`, z)
	}
	if z := d.Div(NearPositiveZero); z != PositiveInfinity {
		t.Errorf(`2/~+0 = %v and should be positive infinity`, z)
	}
	if z := d.Div(NearNegativeZero); z != NegativeInfinity {
		t.Errorf(`2/~-0 = %v and should be negative infinity`, z)
	}
	if z := d.Sub(2).Div(NearNegativeZero); !z.IsNaN() {
		t.Errorf(`0/~+0 = %v and should be nan`, z)
	}
}

func TestCumulativeAddMul(t *testing.T) {
	s, err := NewFromString("0.01")
	if err != nil {
		t.Errorf(`NewFromString("0.01") has result = %v and error = %v`, s, err)
	}

	var d Decimal = 0

	for j := 0; j < 100000; j++ {
		d = d.Add(s)
	}

	log.Printf(`Cumulative 100000 times d = d.Add(%v) = %v`, s, d)

	if d != 1000 {
		t.Errorf(`Cumulative 100000 times d = d.Add(%v) = %v and should be equal to 1000 as int64, d == 1000 is %t (d hex = 0x%016x)`, s, d, d == 1000, int64(d))
	}

	var sf float64 = 0.01
	var f float64 = 0

	for j := 0; j < 100000; j++ {
		f += sf
	}

	log.Printf(`CumulativeAdd on float64 100000 times of %v is %v`, sf, f)

	ds := New(1000001, -6)
	d = 1
	for j := 0; j < 100000; j++ {
		d = d.Mul(ds)
	}
	log.Printf(`Cumulative 100000 times d = d.Mul(%v) = %v`, ds, d)

	f = 1.000001
	sf = 1
	for j := 0; j < 100000; j++ {
		sf = sf * f
	}
	log.Printf(`Cumulative 100000 times sf = sf * %v = %v`, f, sf)
}

func TestSumAvg(t *testing.T) {
	list := []Decimal{1, RequireFromString("1e30"), 1, RequireFromString("-1e30")}
	d := Sum(list[0], list[1:]...)

	if !d.Equal(2) {
		t.Errorf(`.Sum(...) = %v and should be equal to approximately 2, d == ~2 is %t`, d, d.Equal(2))
	}

	// check naive sum
	sum := Zero
	for _, item := range list {
		sum = sum.Add(item)
	}

	log.Printf(`Naive sum of %v = %v, .Sum() = %v`, list, sum, d)
	avg := Avg(list[0], list[1:]...)

	if !avg.Equal(New(5, -1)) {
		t.Errorf(`.Avg(...) = %v and should be equal to approximately 0.5, avg == ~0.5 is %t`, avg, avg.Equal(New(5, -1)))
	}

	min := Min(list[0], list[1:]...)

	if !min.Equal(RequireFromString("-1e30")) {
		t.Errorf(`.Min(...) = %v and should be equal to -1e30`, min)
	}

	max := Max(list[0], list[1:]...)

	if !max.Equal(RequireFromString("1e30")) {
		t.Errorf(`.Max(...) = %v and should be equal to 1e30`, max)
	}
}

func TestIntConversion(t *testing.T) {
	d := New(45712, -2)

	if i, err := d.IntPartErr(); err != nil {
		t.Errorf(`.IntPartErr(...) returned error = %s`, err)
	} else if i != d.Int64() {
		t.Errorf(`.IntPartErr(...) returned different integer %v != %v`, i, d.Int64())
	}
}

func TestSign(t *testing.T) {
	var d Decimal

	if d.Sign() != 0 {
		t.Errorf(`Null.Sign() = %v and should be equal to 0`, d.Sign())
	}

	if Zero.Sign() != 0 {
		t.Errorf(`Zero.Sign() = %v and should be equal to 0`, Zero.Sign())
	}

	if NearZero.Sign() != 0 {
		t.Errorf(`NearZero.Sign() = %v and should be equal to 0`, NearZero.Sign())
	}

	if NearPositiveZero.Sign() != 1 {
		t.Errorf(`NearPositiveZero.Sign() = %v and should be equal to 1`, NearPositiveZero.Sign())
	}

	if NearNegativeZero.Sign() != -1 {
		t.Errorf(`NearNegativeZero.Sign() = %v and should be equal to -1`, NearNegativeZero.Sign())
	}

	if PositiveInfinity.Sign() != 1 {
		t.Errorf(`PositiveInfinity.Sign() = %v and should be equal to 1`, NegativeInfinity.Sign())
	}

	if NegativeInfinity.Sign() != -1 {
		t.Errorf(`NegativeInfinity.Sign() = %v and should be equal to -1`, NegativeInfinity.Sign())
	}

	d = 123
	if d.Sign() != 1 {
		t.Errorf(`123.Sign() = %v and should be equal to 1`, d.Sign())
	}

	d = -d
	if d.Sign() != -1 {
		t.Errorf(`-123.Sign() = %v and should be equal to -1`, d.Sign())
	}

	d = (-d).Div(7)
	if d.Sign() != 1 {
		t.Errorf(`(123/7).Sign() = %v and should be equal to 1`, d.Sign())
	}

	d = -d.Div(7)
	if d.Sign() != -1 {
		t.Errorf(`(-123/7).Sign() = %v and should be equal to -1`, d.Sign())
	}

	log.Printf("-123/7/7 = %v", d)

	if d.IsExact() {
		t.Errorf(`(-123/7/7).IsExact() should be false but is %v`, d.IsExact())
	}

	if !d.Mul(7).Mul(-7).Round(12).Equal(123) {
		t.Errorf(`(-123/7/7)*7*(-7) is %v`, d.Mul(7).Mul(-7))
	}
}

func TestTranscendantalFunctions(t *testing.T) {
	sqrt2 := New(2, 0).Sqrt()
	_sqrt2 := New(141421356237309514, -17) // FIXME: since exponent can only be between -16 and +15, mantissa will be truncated
	log.Printf("float64 sqrt of 2 = %v, its square = %v, (2).Sqrt() = %v, ((2).Sqrt())² = %v, _sqrt2 = %v, _sqrt2² = %v", math.Sqrt(2), math.Sqrt(2)*math.Sqrt(2), sqrt2, sqrt2.Mul(sqrt2), _sqrt2, _sqrt2.Mul(_sqrt2))

	if !sqrt2.Mul(sqrt2).Round(15).Equal(2) {
		t.Errorf(`((2).Sqrt())² should be 2, but is %v`, sqrt2.Mul(sqrt2).Round(15))
	}

	pi4 := NewFromFloat(math.Pi / 4)
	sinpi4 := pi4.Sin()
	cospi4 := pi4.Cos()
	tanpi4 := pi4.Tan()
	if !sinpi4.Round(15).Equal(sqrt2.Div(2).Round(15)) {
		t.Errorf(`(pi/4).Sin() should be (2).Sqrt()/2, but is %v`, sinpi4)
	}
	if !cospi4.Round(15).Equal(sqrt2.Div(2).Round(15)) {
		t.Errorf(`(pi/4).Cos() should be (2).Sqrt()/2, but is %v`, cospi4)
	}
	if !tanpi4.Equal(1) {
		t.Errorf(`(pi/4).Tan() should be near 1, but is %v`, tanpi4)
	}
	log.Printf("pi/4 = %v, sin(pi/4) = %v (decimal sin(pi/4) = %v), cos(pi/4) = %v (decimal cos(pi/4) = %v)", pi4, math.Sin(math.Pi/4), sinpi4, math.Cos(math.Pi/4), cospi4)
	log.Printf("tan(pi/4) = %v, decimal tan(pi/4) = %v, decimal sin(pi/4)/cos(pi/4) = %v", math.Tan(math.Pi/4), tanpi4, sinpi4.Div(cospi4))

	pi2 := NewFromFloat(math.Pi / 2)
	log.Printf("tan(pi/2) = %v, decimal tan(pi/2) = %v, decimal sin(pi/2)/cos(pi/2) = %v", math.Tan(math.Pi/2), pi2.Tan(), pi2.Sin().Div(pi2.Cos()))
}

func TestUnmarshalBinary(t *testing.T) {
	var d Decimal = 99

	if err := d.UnmarshalBinary([]byte{0x00}); err != nil {
		t.Errorf(`UnmarshalBinary(0x00) should be ok, error = %v`, err)
	} else if d != Null {
		t.Errorf(`UnmarshalBinary(0x00) should be null decimal, d = %v`, d)
	}

	if err := d.UnmarshalBinary([]byte{0x80}); err != nil {
		t.Errorf(`UnmarshalBinary(0x80) should be ok, error = %v`, err)
	} else if d != Zero {
		t.Errorf(`UnmarshalBinary(0x80) should be zero decimal, d = %v`, d)
	}

	if err := d.UnmarshalBinary([]byte{0xc0}); err != nil {
		t.Errorf(`UnmarshalBinary(0xc0) should be ok, error = %v`, err)
	} else if d != NearZero {
		t.Errorf(`UnmarshalBinary(0xc0) should be near zero decimal, d = %v`, d)
	}

	if err := d.UnmarshalBinary([]byte{0x60}); err != nil {
		t.Errorf(`UnmarshalBinary(0xe0) should be ok, error = %v`, err)
	} else if d != NearPositiveZero {
		t.Errorf(`UnmarshalBinary(0x60) should be near positive zero decimal, d = %v`, d)
	}

	if err := d.UnmarshalBinary([]byte{0xe0}); err != nil {
		t.Errorf(`UnmarshalBinary(0xe0) should be ok, error = %v`, err)
	} else if d != NearNegativeZero {
		t.Errorf(`UnmarshalBinary(0xe0) should be near negative zero decimal, d = %v`, d)
	}

	if err := d.UnmarshalBinary([]byte{0x01, 0x64}); err != nil {
		t.Errorf(`UnmarshalBinary(0x01, 0x64) should be ok, error = %v`, err)
	} else if d != 100 {
		t.Errorf(`UnmarshalBinary(0x01, 0x64) should be 100, d = %v`, d)
	}

	if err := d.UnmarshalBinary([]byte{0xbd, 0x65}); err != nil {
		t.Errorf(`UnmarshalBinary(0x3d, 0x65) should be ok, error = %v`, err)
	} else if d != New(-101, -2) {
		t.Errorf(`UnmarshalBinary(0x3d, 0x65) should be -1.01, d = %v`, d)
	}
}

func TestMarshalBinaryZero(t *testing.T) {
	var d Decimal

	if b, err := d.MarshalBinary(); err != nil {
		t.Errorf(`Null.MarshalBinary() should be ok, error = %v`, err)
	} else if len(b) != 1 || b[0] != 0x00 {
		t.Errorf(`Null.MarshalBinary() should be { 0x00 }, buff = %v`, b)
	}

	d = Zero
	if b, err := d.MarshalBinary(); err != nil {
		t.Errorf(`Zero.MarshalBinary() should be ok, error = %v`, err)
	} else if len(b) != 1 || b[0] != 0x80 {
		t.Errorf(`Zero.MarshalBinary() should be { 0x80 }, buff = %v`, b)
	}
}
func TestMarshalBinaryNearZero(t *testing.T) {
	d := NearZero

	if b, err := d.MarshalBinary(); err != nil {
		t.Errorf(`NearZero.MarshalBinary() should be ok, error = %v`, err)
	} else if len(b) != 1 || b[0] != 0xc0 {
		t.Errorf(`NearZero.MarshalBinary() should be { 0xc²0 }, buff = %v`, b)
	}

	d = NearPositiveZero
	if b, err := d.MarshalBinary(); err != nil {
		t.Errorf(`NearPositiveZero.MarshalBinary() should be ok, error = %v`, err)
	} else if len(b) != 1 || b[0] != 0x60 {
		t.Errorf(`NearPositiveZero.MarshalBinary() should be { 0x60 }, buff = %v`, b)
	}

	d = NearNegativeZero
	if b, err := d.MarshalBinary(); err != nil {
		t.Errorf(`NearNegativeZero.MarshalBinary() should be ok, error = %v`, err)
	} else if len(b) != 1 || b[0] != 0xe0 {
		t.Errorf(`NearNegativeZero.MarshalBinary() should be { 0xe0 }, buff = %v`, b)
	}

}

func TestMarshalBinary(t *testing.T) {
	d := NewFromInt(100)

	if b, err := d.MarshalBinary(); err != nil {
		t.Errorf(`100.MarshalBinary() should be ok, error = %v`, err)
	} else if len(b) != 2 || b[0] != 0x01 && b[1] != 0x64 {
		t.Errorf(`100.MarshalBinary() should be { 0x01, 0x64 }, buff = %v`, b)
	}

	d = -320
	if b, err := d.MarshalBinary(); err != nil {
		t.Errorf(`(-320).MarshalBinary() should be ok, error = %v`, err)
	} else if len(b) != 3 || b[0] != 0x81 && b[1] != 0xc0 && b[2] != 0x02 {
		t.Errorf(`(-320).MarshalBinary() should be { 0x81, 0xc0, 0x02 }, buff = %v`, b)
	}

	d = New(101, -2)
	if b, err := d.MarshalBinary(); err != nil {
		t.Errorf(`(1.01).MarshalBinary() should be ok, error = %v`, err)
	} else if len(b) != 2 || b[0] != 0x3d && b[1] != 0x65 {
		t.Errorf(`(1.01).MarshalBinary() should be { 0x3d, 0x65 }, buff = %v`, b)
	}
}

func BenchmarkIsExactlyZero(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		d := Decimal(i % 257)

		if d.IsExactlyZero() {
			count++
		}
	}
}

func BenchmarkTestExactlyZero(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		d := Decimal(i % 257)

		if d == Null || d == Zero {
			count++
		}
	}
}

func BenchmarkIsZero(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		d := Decimal(i % 257)

		if d.IsZero() {
			count++
		}
	}
}

func BenchmarkDecimalNewFromFloat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewFromFloat(100020003000400050e-17)
	}
}

func BenchmarkDecimalNewFromString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewFromString("100020003000400050e-17")
	}
}

func BenchmarkIntNewFromString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		strconv.ParseInt("100020003000400050", 0, 64)
	}
}

func BenchmarkFloatNewFromString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		strconv.ParseFloat("100020003000400050e-17", 64)
	}
}

func BenchmarkDecimalString(b *testing.B) {
	d, _ := NewFromString("100020003000400050e-17")

	for i := 0; i < b.N; i++ {
		_ = d.String()
	}
}

func BenchmarkIntString(b *testing.B) {
	var f int64 = 100020003000400050

	for i := 0; i < b.N; i++ {
		strconv.FormatInt(f, 10)
	}
}

func BenchmarkFloatString(b *testing.B) {
	f := 100020003000400050e-17

	for i := 0; i < b.N; i++ {
		strconv.FormatFloat(f, 'E', -1, 64)
	}
}

func BenchmarkDecimalAdd(b *testing.B) {
	s, _ := NewFromString("0.000001")
	var d Decimal

	for i := 0; i < b.N; i++ {
		d = d.Add(s)
	}
}

func BenchmarkFloat64Add(b *testing.B) {
	var sf float64 = 0.000001
	var f float64 = 0

	for i := 0; i < b.N; i++ {
		f = f + sf
	}
}

func BenchmarkDecimalMul(b *testing.B) {
	s, _ := NewFromString("1.00123456789")
	d := New(123456789, 0)

	for i := 0; i < b.N; i++ {
		_ = d.Mul(s)
	}
}

func BenchmarkFloat64Mul(b *testing.B) {
	var sf float64 = 1.00123456789
	var f float64 = 123456789

	for i := 0; i < b.N; i++ {
		_ = f * sf
	}
}

func BenchmarkDecimalDiv(b *testing.B) {
	s, _ := NewFromString("1.00123456789")
	d := New(123456789, 0)

	for i := 0; i < b.N; i++ {
		_ = d.Div(s)
	}
}

func BenchmarkFloat64Div(b *testing.B) {
	var sf float64 = 1.00123456789
	var f float64 = 123456789

	for i := 0; i < b.N; i++ {
		_ = f / sf
	}
}

func BenchmarkDecimalQuoRem(b *testing.B) {
	d1 := New(14569568235, -3)
	d2 := New(5607988, -5)

	for i := 0; i < b.N; i++ {
		_, _ = d1.QuoRem(d2, 0)
	}
}

func BenchmarkDecimalRound(b *testing.B) {
	s, _ := NewFromString("-1.454")

	for i := 0; i < b.N; i++ {
		s.Round(1)
	}
}

func BenchmarkDecimalRoundCeil(b *testing.B) {
	s, _ := NewFromString("-1.454")

	for i := 0; i < b.N; i++ {
		s.RoundCeil(1)
	}
}

func BenchmarkPublicDecimalAdd(b *testing.B) {
	d1 := New(551, -2)
	d2 := New(6019, -3)

	for i := 0; i < b.N; i++ {
		_ = d1.Add(d2)
	}
}

func BenchmarkPublicDecimalMul(b *testing.B) {
	d1 := New(212, -2)
	d2 := New(31, 1)

	for i := 0; i < b.N; i++ {
		_ = d1.Mul(d2)
	}
}

func BenchmarkPublicDecimalInexactQuo(b *testing.B) {
	d1 := New(212, -2)
	d2 := New(31, 1)

	for i := 0; i < b.N; i++ {
		_ = d1.Div(d2)
	}
}

func BenchmarkPublicDecimalExactQuo(b *testing.B) {
	d1 := New(3255, -2)
	d2 := New(31, 1)

	for i := 0; i < b.N; i++ {
		_ = d1.Div(d2)
	}
}

func BenchmarkPublicDecimalPow60(b *testing.B) {
	d1 := New(11, -1)
	d2 := New(60, 0)

	for i := 0; i < b.N; i++ {
		_ = d1.Pow(d2)
	}
}

func BenchmarkPublicDecimalPow600(b *testing.B) {
	d1 := New(101, -2)
	d2 := New(600, 0)

	for i := 0; i < b.N; i++ {
		_ = d1.Pow(d2)
	}
}

