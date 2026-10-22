package decimal

import (
	"fmt"
	"reflect"
	"testing"
)

func TestFormat(t *testing.T) {
	cases := []struct {
		format string
		d      Decimal
		want   string
	}{
		{"%v", New(-314, -2), "-3.14"},
		{"%s", New(-314, -2), "-3.14"},
		{"%v", Null, "0"},
		{"%8v|", New(314, -2), "    3.14|"},
		{"%-8s|", New(314, -2), "3.14    |"},
		{"%q", New(314, -2), `"3.14"`},
		{"%x", New(5, 0), "35"},
		{"%.2f", New(314159, -5), "3.14"},
		{"%.2f", New(5, -3), "0.01"},
		{"%.2f", New(-6, -3), "-0.01"},
		{"%.2f", New(5, 0), "5.00"},
		{"%.2f", Null, "0.00"},
		{"%.2f", Zero, "0.00"},
		{"%.0f", New(25, -1), "3"},
		{"%.2f", New(1, 20), "100000000000000000000.00"},
		{"%f", New(-314, -2), "-3.14"},
		{"%F", New(314, -2), "3.14"},
		{"%8.2f|", New(314, -2), "    3.14|"},
		{"%-8.2f|", New(314, -2), "3.14    |"},
		{"%08.2f", New(-314, -2), "-0003.14"},
		{"%+.2f", New(314, -2), "+3.14"},
		{"% .2f", New(314, -2), " 3.14"},
		{"%+.2f", New(-314, -2), "-3.14"},
		{"%.2f", NaN, "NaN"},
		{"%06.2f", NaN, "   NaN"},
		{"%.2f", PositiveInfinity, "+Inf"},
		{"%d", New(42, 0), "42"},
		{"%5d", New(-42, 0), "  -42"},
		{"%d", Null, "0"},
		{"%d", Zero, "0"},
		{"%d", New(-314, -2), "%!d(decimal.Decimal=-3.14)"},
		{"%e", New(314, -2), "%!e(decimal.Decimal=3.14)"},
		{"%#v", New(-314, -2), "decimal.New(-314, -2)"},
		{"%#v", New(5, 0), "decimal.New(5, 0)"},
		{"%#v", Null, "decimal.Null"},
		{"%#v", Zero, "decimal.Zero"},
		{"%#v", NaN, "decimal.NaN"},
		{"%#v", NegativeInfinity, "decimal.NegativeInfinity"},
		{"%#v", PositiveInfinity, "decimal.PositiveInfinity"},
		{"%#v", NearZero, "decimal.NearZero"},
		{"%#v", NearPositiveZero, "decimal.NearPositiveZero"},
		{"%#v", NearNegativeZero, "decimal.NearNegativeZero"},
		{"%.3s", New(314159, -5), "3.1"},
		{"%8.3v|", New(314159, -5), "     3.1|"},
		{"%f", Null, "0"},
	}
	for _, c := range cases {
		if got := fmt.Sprintf(c.format, c.d); got != c.want {
			t.Errorf("Sprintf(%q, %v) = %q, want %q", c.format, c.d, got, c.want)
		}
	}

	// %#v must recreate an inexact value bit for bit
	third := New(1, 0).Div(New(3, 0))
	if got, want := fmt.Sprintf("%#v", third), fmt.Sprintf("decimal.Decimal(%d)", int64(third)); got != want {
		t.Errorf("Sprintf(%%#v, 1/3) = %q, want %q", got, want)
	}

	// Println goes through Format with %v
	if got := fmt.Sprintln(New(15, -1)); got != "1.5\n" {
		t.Errorf("Sprintln(1.5) = %q, want %q", got, "1.5\n")
	}
}

func TestNormalize(t *testing.T) {
	a, _ := NewFromString("1.50")
	b := New(15, -1)
	if a.Normalize() != b.Normalize() || !reflect.DeepEqual(a.Normalize(), b) {
		t.Errorf(`NewFromString("1.50").Normalize() should equal New(15, -1), got %v`, a.Normalize())
	}
	if New(0, -3).Normalize() != Zero {
		t.Errorf(`New(0, -3).Normalize() should be Zero, got %#v`, New(0, -3).Normalize())
	}
	for _, d := range []Decimal{Null, NaN, PositiveInfinity, NearPositiveZero, New(1, 0).Div(New(3, 0))} {
		if d.Normalize() != d {
			t.Errorf("%#v.Normalize() should be unchanged, got %#v", d, d.Normalize())
		}
	}
}
