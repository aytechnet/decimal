package decimal

import (
	"fmt"
	"strconv"
)

// Format implements fmt.Formatter, so decimals work with the fmt verbs like shopspring/decimal:
//
//   - %v and %s print String(), and %q, %x and %X quote it or print it in hex, as for any fmt.Stringer.
//   - %f and %F print fixed-point notation. With a precision, as in %.2f, the decimal is rounded like
//     StringFixed. Without one, it is printed like String(). Width and the '-', '+', ' ' and '0' flags
//     work as they do for floats.
//   - %d prints an integer decimal as an int64 (Null and Zero print 0); a fractional or special
//     decimal prints fmt's usual error, such as %!d(decimal.Decimal=3.14).
//   - %#v prints Go syntax that recreates the decimal exactly: decimal.New(-314, -2) for an exact
//     finite value, the constant name for Null, Zero, NaN, ±Inf and the near zeros, and
//     decimal.Decimal(<int64>) for an inexact value (loss bit set).
//
// Other verbs print fmt's usual error.
func (d Decimal) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v', 's', 'q', 'x', 'X':
		if verb == 'v' && f.Flag('#') {
			_, _ = f.Write(d.goSyntax())
		} else {
			formatString(f, verb, d)
		}
	case 'f', 'F':
		d.formatFixed(f)
	case 'd':
		if d.IsInteger() {
			_, _ = fmt.Fprintf(f, fmtDirective(f, verb), d.IntPart())
		} else {
			formatBadVerb(f, verb, d)
		}
	default:
		formatBadVerb(f, verb, d)
	}
}

// formatString formats d.String() as fmt would for the directive that called Format, with verb in place of its own.
func formatString(f fmt.State, verb rune, d Decimal) {
	// Println and plain %v are the common case: no directive to rebuild
	_, hasWidth := f.Width()
	_, hasPrec := f.Precision()
	if (verb == 'v' || verb == 's') && !hasWidth && !hasPrec {
		var buf [32]byte
		if d == Null {
			_, _ = f.Write(append(buf[:0], '0'))
		} else {
			_, _ = f.Write(d.BytesTo(buf[:0]))
		}
		return
	}
	_, _ = fmt.Fprintf(f, fmtDirective(f, verb), d.String())
}

// formatBadVerb prints fmt's error for a verb that does not apply, such as %!d(decimal.Decimal=3.14).
func formatBadVerb(f fmt.State, verb rune, d Decimal) {
	_, _ = fmt.Fprintf(f, "%%!%c(decimal.Decimal=%s)", verb, d.String())
}

// fmtDirective rebuilds the directive that called Format (fmt.FormatString needs Go 1.20).
func fmtDirective(f fmt.State, verb rune) string {
	b := make([]byte, 0, 16)
	b = append(b, '%')
	for _, flag := range "+-# 0" {
		if f.Flag(int(flag)) {
			b = append(b, byte(flag))
		}
	}
	if width, ok := f.Width(); ok {
		b = strconv.AppendInt(b, int64(width), 10)
	}
	if prec, ok := f.Precision(); ok {
		b = strconv.AppendInt(append(b, '.'), int64(prec), 10)
	}
	return string(append(b, string(verb)...))
}

// formatFixed prints d for %f and %F, padded to the width with the '-', '+', ' ' and '0' flags.
func (d Decimal) formatFixed(f fmt.State) {
	var buf [48]byte
	var s []byte
	if prec, ok := f.Precision(); ok {
		s = d.IfNull(Zero).BytesToFixed(buf[:0], int32(prec))
	} else if d == Null {
		s = append(buf[:0], '0')
	} else {
		s = d.BytesTo(buf[:0])
	}

	var sign byte
	switch {
	case s[0] == '-' || s[0] == '+':
		sign, s = s[0], s[1:]
	case f.Flag('+'):
		sign = '+'
	case f.Flag(' '):
		sign = ' '
	}
	pad := 0
	if width, ok := f.Width(); ok {
		pad = width - len(s)
		if sign != 0 {
			pad--
		}
	}

	var p [1]byte
	writePad := func(c byte) {
		p[0] = c
		for ; pad > 0; pad-- {
			_, _ = f.Write(p[:])
		}
	}
	writeSign := func() {
		if sign != 0 {
			p[0] = sign
			_, _ = f.Write(p[:])
		}
	}
	switch {
	case f.Flag('-'):
		writeSign()
		_, _ = f.Write(s)
		writePad(' ')
	case f.Flag('0') && s[0] >= '0' && s[0] <= '9':
		writeSign()
		writePad('0')
		_, _ = f.Write(s)
	default:
		writePad(' ')
		writeSign()
		_, _ = f.Write(s)
	}
}

// goSyntax returns Go syntax that recreates d exactly.
func (d Decimal) goSyntax() []byte {
	switch d {
	case Null:
		return []byte("decimal.Null")
	case Zero:
		return []byte("decimal.Zero")
	case NearZero:
		return []byte("decimal.NearZero")
	case NearPositiveZero:
		return []byte("decimal.NearPositiveZero")
	case NearNegativeZero:
		return []byte("decimal.NearNegativeZero")
	case PositiveInfinity:
		return []byte("decimal.PositiveInfinity")
	case NegativeInfinity:
		return []byte("decimal.NegativeInfinity")
	}
	if d.IsNaN() {
		return []byte("decimal.NaN")
	}

	b := make([]byte, 0, 40)
	if v, m, e := d.vme(); m > 0 && v&loss == 0 {
		b = append(b, "decimal.New("...)
		if v&sign != 0 {
			b = append(b, '-')
		}
		b = strconv.AppendUint(b, m, 10)
		b = append(b, ", "...)
		b = strconv.AppendInt(b, e, 10)
		return append(b, ')')
	}
	b = append(b, "decimal.Decimal("...)
	b = strconv.AppendInt(b, int64(d), 10)
	return append(b, ')')
}
