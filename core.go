package decimal

import (
	// "log"
	"bytes"
	"math"
	"math/bits"
	"strconv"
)

const (
	// sign and loss bit are the same of any decimal types
	sign uint64 = 0x8000000000000000
	loss        = 0x4000000000000000
)

// array of power of ten suitable to be hold in uint64
var ten_pow = [...]uint64{
	1, 10, 100, 1000, 10000, 100000, 1000000, 10000000, 100000000, 1000000000,
	10000000000, 100000000000, 1000000000000, 10000000000000, 100000000000000,
	1000000000000000, 10000000000000000, 100000000000000000, 1000000000000000000, 10000000000000000000,
}

// normalize a VME tuple according to maximal value of mantissa, minimal and maximal value of exposant
// normalize always try to hold an integerer value into mantissa so that exposant is 0 if possible
// if it is not possible to hold as an integer, normalize it so that mantissa is not divisible by 10 unless exposant is out of range
// this make possible to use a decimal as a hash key and use == or != operators (it is still not possible to use < or > operator)
// example :
//
//	  d, _ := decimal.NewFromString("0.001").Mul(1000)
//	or
//	  d := decimal.New(1, -3).Mul(1000)
//
//	  fmt.Printf("d = %v, d == 1 is %t\n", d1, d1 == 1)
//
// the following magic case apply when m == 0 for (v,m,e) :
//
//   - loss is set :
//     ~0   = (sign | loss, 0, 0)
//     +~0  = (loss, 0, min_e)
//     -~0  = (sign | loss, 0, min_e)
//     NaN  = (loss | sign, 0, any other than 0 or min_e or max_e)
//     +Inf = (loss, 0, max_e)
//     -Inf = (sign | loss, 0, max_e)
//
//   - loss is not set :
//     null = (0, 0, 0)
//     0    = (sign, 0, any) or (0, 0, not 0)
func vmeNormalize(v, m uint64, e int64, max_m uint64, min_e, max_e int64) (uint64, uint64, int64) {
	if m == 0 {
		return veNormalizeMagic(v, e, min_e, max_e)
	} else {
		// check if an decimal can be represented as a compatible int64 value
		// it is possible ony if loss bit is not set and exposant is in acceptable range
		if v&loss == 0 {
			if e == 0 {
				if m <= max_m {
					return v, m, 0
				}
			} else if e > 0 {
				if e < int64(len(ten_pow)) {
					h, l := bits.Mul64(m, ten_pow[e])

					if h == 0 && l <= max_m {
						return v, l, 0
					}
				}
			} else if m&1 == 0 {
				if e > -int64(len(ten_pow)) {
					q, r := bits.Div64(0, m, ten_pow[-e])

					if r == 0 && q <= max_m {
						return v, q, 0
					}
				}
			}
		}

		// normalize m while it is divisible by 10 or greather than max_m
		for m > max_m || e <= max_e && m > 9 && m&1 == 0 {
			q, r := bits.Div64(0, m, 10)
			if r != 0 {
				if m <= max_m {
					break
				}

				v |= loss

				// round to the nearest, but using round bank approach to minimize errors
				if r > 5 || r == 5 && q&1 == 1 {
					q++
				}
			}

			m = q
			e++
		}

		return vmeNormalizeExposant(v, m, e, max_m, min_e, max_e)
	}
}

func vmeNormalizeExposant(v, m uint64, e int64, max_m uint64, min_e, max_e int64) (uint64, uint64, int64) {
	// normalize too small exposant by updating mantissa and adding if necessary a precision loss
	if e < min_e {
		if min_e-e < int64(len(ten_pow)) {
			var r uint64

			m, r = bits.Div64(0, m, ten_pow[min_e-e])
			if r != 0 {
				v |= loss

				// round to the nearest
				if (r << 1) >= ten_pow[min_e-e] {
					m++
				}
			}
		} else {
			v |= loss

			m = 0
		}

		// e is now min_e
		e = min_e
	}

	// normalize too big exposant
	if e > max_e {
		if e-max_e < int64(len(ten_pow)) {
			h, l := bits.Mul64(m, ten_pow[e-max_e])

			if h == 0 || l < max_m {
				m = l
			} else {
				v |= loss

				// infinity has a special value of mantissa of 0 while e is equal to max_e
				m = 0
			}
		} else {
			v |= loss

			// infinity has a special value of mantissa of 0 while e is equal to max_e
			m = 0
		}

		// e is now max_e
		e = max_e
	}

	return v, m, e
}

func veNormalizeMagic(v uint64, e int64, min_e, max_e int64) (uint64, uint64, int64) {
	if v&loss == 0 {
		return v, 0, 0
	} else {
		if e < min_e {
			e = min_e
		} else if e > max_e {
			e = max_e
			//log.Printf("  -> vme is normalized as (0x%016x,%v,%v)\n", v, m, e)
		}
	}

	return v, 0, e
}

func vmhmeReduce(v, mh, m uint64, e int64) (uint64, uint64, int64) {
	if mh > 0 {
		for i, p := range ten_pow {
			if mh < p {
				q, r := bits.Div64(mh, m, p)
				if r != 0 {
					v |= loss

					// round to nearest
					if (r << 1) >= p {
						q++
					}
				}
				mh, m = 0, q
				e += int64(i)
				break
			}
		}
	}

	// check for this rare case where h is too big, another division by 10 is enough (ie (2^63-1)^2/(10^19) / 10 < 2^63-1)
	// note this case is not reached using Decimal as only 57 bits are used for mantissa
	if mh > 0 {
		qh, rh := bits.Div64(0, mh, 10)
		qm, rm := bits.Div64(rh, m, 10)
		if rm != 0 {
			v |= loss

			// round to nearest
			if rm >= 5 {
				qm++
			}
		}

		i := len(ten_pow) - 1
		p := ten_pow[i]

		q, r := bits.Div64(qh, qm, p)
		if r != 0 {
			v |= loss
		}
		mh, m = 0, q

		e += 1 + int64(i)
	}

	return v, m, e
}

// extract a VME tuple from bytes which need to be normalized
func vmeMagicFromBytes(b []byte, v, m uint64, e int64, parsed_sign bool) (uint64, uint64, int64, error) {
	// strongly reduce cyclomatic complexity at the expense of memory allocation in this case
	switch string(bytes.ToLower(b)) {
	case "on", "yes":
		return v, 1, 0, nil

	case "no", "off":
		if v&loss == 0 || !parsed_sign {
			return v | sign, 0, 0, nil
		} else {
			return v, 0, math.MinInt64, nil
		}

	case "nan":
		return loss, 0, 1, nil

	case "nil", "null":
		return 0, 0, 0, nil

	case "inf":
		return v | loss, 0, math.MaxInt64, nil
	}

	return 0, 0, 0, ErrSyntax
}

// extract a VME tuple from bytes which need to be normalized
func vmeFromBytes(b []byte) (v, m uint64, e int64, err error) {
	// take care of utf8 encoding with TrimSpace which is no more needed in the following code or a syntax error is raised
	b = bytes.TrimSpace(b)

	i := 0
	j := len(b) - 1

	if i < j && (b[i] == '"' && b[j] == '"' || b[i] == '\'' && b[j] == '\'') {
		i++
		j--
	}

	if i > j {
		return 0, 0, 0, nil
	}

	// allow ~ to be first byte
	if b[i] == '~' {
		v |= loss

		i++
		if i > j {
			return 0, 0, 0, ErrSyntax
		}
	}

	parsed_sign := false

	if b[i] == '+' {
		parsed_sign = true

		i++
		if i > j {
			return 0, 0, 0, ErrSyntax
		}
	} else if b[i] == '-' {
		v |= sign

		parsed_sign = true

		i++
		if i > j {
			return 0, 0, 0, ErrSyntax
		}
	}

	// allow ~ to be after sign as well
	if b[i] == '~' {
		v |= loss

		i++
		if i > j {
			return 0, 0, 0, ErrSyntax
		}
	}

	doti := -1

	for i <= j {
		if b[i] >= '0' && b[i] <= '9' {
			h, l := bits.Mul64(m, 10)

			// if uint64 is big enough to hold this number
			if h == 0 {
				m = l + uint64(b[i]-'0')

				if doti >= 0 && e <= 0 {
					e--
				} else if doti < 0 && e > 0 {
					e++
				}
			} else {
				if e >= 0 && b[i] != '0' {
					v |= loss
				}
				if doti < 0 {
					e++
				}
			}

			i++

			continue
		} else if b[i] == '.' {
			if doti < 0 { // only one dot is allowed or a syntax error is raised
				doti = i
			} else {
				return 0, 0, 0, ErrSyntax
			}

			i++

			continue
		} else if (b[i] | 0x20) == 'e' { // a little more compact and faster and equivalent to b[i] == 'e' || b[i] == 'E'
			if _e, err := strconv.ParseInt(string(b[i+1:j+1]), 10, 64); err == nil {
				e += _e

				break
			} else {
				return 0, 0, 0, err
			}
		} else {
			return vmeMagicFromBytes(b[i:j+1], v, m, e, parsed_sign)
		}
	}

	//log.Printf("vmeFromBytes(%s) = (v=0x%016x,m=%d,e=%d)", b, v, m, e)
	// FIXME: NaN does not occurs here, so fix v and e to avoid NaN report
	if m == 0 {
		if v&loss != 0 {
			if parsed_sign {
				e = math.MinInt64
			} else {
				v |= sign
				e = 0
			}
		} else {
			v = sign
			e = 0
		}
	}

	return
}

// bytes appends decimal representation of a VME tuple to b
// ext is a boolean value to allow extended output (~ if loss), Inf for Infinite and NaN for not-a-number
// str is a boolean value to add double quote before and after output
func vmeBytes(b []byte, v, m uint64, e int64, ext, str bool) []byte {
	if str {
		b = append(b, '"')
	}

	if m > 0 {
		var r uint64
		var i int
		var output bool

		if ext && (v&loss) != 0 {
			b = append(b, '~')
		}
		if v&sign != 0 {
			b = append(b, '-')
		}
		for i = len(ten_pow) - 1; i >= 0; i-- {
			if int64(i)+e+1 == 0 {
				if !output {
					b = append(b, '0')
				}

				b = append(b, '.')

				output = true
			}

			m, r = bits.Div64(0, m, ten_pow[i])

			if output || m > 0 || int64(i)+e <= 0 {
				b = append(b, byte(m)+'0')

				output = true
			}

			m = r
		}

		for e += int64(i); e >= 0; e-- {
			b = append(b, '0')
		}
	} else {
		if v&loss != 0 {
			b = veMagicBytes(b, v, e, ext)
		} else {
			b = append(b, '0')
		}
	}

	if str {
		b = append(b, '"')
	}

	//log.Printf(`bytes(0x%016x, %d, %d) = "%s"`, v, m, e, b)
	return b
}

// veMagicBytes appends decimal representation of a VME magic tuple to b
// ext is a boolean value to allow extended output (~ if loss), Inf for Infinite and NaN for not-a-number
func veMagicBytes(b []byte, v uint64, e int64, ext bool) []byte {
	if ext {
		if e == math.MaxInt64 {
			if v&sign != 0 {
				b = append(b, '-')
			} else {
				b = append(b, '+')
			}
			b = append(b, 'I', 'n', 'f')
		} else if e == 0 {
			b = append(b, '~', '0')
		} else if e > math.MinInt64 {
			b = append(b, 'N', 'a', 'N')
		} else {
			if v&sign != 0 {
				b = append(b, '-')
			} else {
				b = append(b, '+')
			}
			b = append(b, '~', '0')
		}
	} else {
		if e == 0 || e == math.MinInt64 {
			b = append(b, '0')
		} else {
			b = append(b, 'n', 'u', 'l', 'l')
		}
	}

	return b
}

func vmeAddMagic1(v1 uint64, e1 int64, v2, m2 uint64, e2 int64) (v, m uint64, e int64) {
	// m1 is already 0 and loss bit is set so check if d1 is ~0, ~+0, ~-0, NaN, -Inf or +Inf
	if e1 == 0 || e1 == math.MinInt64 { // if d1 == ~0 or d1 == -~0 or d1 == +~0
		if m2 == 0 && v2&loss != 0 {
			if v2 == sign|loss && e2 == 0 { // if d2 == ~0
				return v2, m2, e2
			} else if e2 == math.MinInt64 { // if d2 == +~0 or d2 == -~0
				if (v1^v2)&sign == 0 {
					return v2, m2, e2
				} else {
					return sign | loss, 0, 0 // ~0
				}
			} else {
				return v2, m2, e2 // NaN or +Inf or -Inf
			}
		} else {
			if v2 == 0 && m2 == 0 && e2 == 0 {
				// FIXME: return d1 is d2 is null
				return v1, 0, e1
			} else {
				//log.Printf("AddMagic() returns 0x%016x, %v, %v", v2 | loss, m2, e2)
				return v2 | loss, m2, e2
			}
		}
	} else if e1 == math.MaxInt64 { // if d1 == -Inf or +Inf
		if m2 == 0 && v2&loss != 0 { // if d2 is a magic decimal
			if e2 == math.MaxInt64 { // if d2 == -Inf or +Inf
				if sign&(v1^v2) == 0 { // if sign are the same, +Inf + +Inf = +Inf or -Inf + -Inf = -Inf
					return v2, m2, e2
				} else { // if sign are differnt +Inf + -Inf == NaN
					return loss, 0, 1 // NaN must have v = loss, m = 0 and e is not 0, math.MaxInt64 or math.MinInt64
				}
			} else if e2 == math.MinInt64 { // if d2 == ~0
				return v1, 0, e1
			} else { // if d2 is a NaN decimal
				return v2, m2, e2
			}
		} else { // d2 is a normal decimal, return d1 which is -Inf or +Inf
			return v1, 0, e1
		}
	} else { // d1 is a NaN decimal
		return v1, 0, e1
	}
}

func vmeAdd(v1, m1 uint64, e1 int64, v2, m2 uint64, e2 int64) (v, m uint64, e int64) {
	// v1, m1, e1 and v2, m2, e2 are respectively representation of decimal d1 and d2

	// swap d1 and d2 vme so e1 <= e2
	if e1 > e2 {
		v1, m1, e1, v2, m2, e2 = v2, m2, e2, v1, m1, e1
	}

	// handle magic of d1
	if m1 == 0 {
		if v1&loss != 0 {
			//log.Printf("<< addMagic(0x%016x,%d, 0x%016x,%d,%d)", v1,e1, v2,m2,e2)
			return vmeAddMagic1(v1, e1, v2, m2, e2)
		} else { // d1 == 0 (because loss is not set)
			//log.Printf(">> add(0x%016x,%d,%d, 0x%016x,%d,%d) = (0x%016x,%d,%d)", v1,m1,e1, v2,m2,e2, v2,m2,e2)
			return v2, m2, e2
		}
	}

	// handle magic of d2
	if m2 == 0 {
		if v2&loss != 0 {
			//log.Printf("<< addMagic(0x%016x,%d, 0x%016x,%d,%d)", v2,e2, v1,m1,e1)
			return vmeAddMagic1(v2, e2, v1, m1, e1)
		} else { // d2 == 0 (because loss is not set)
			//log.Printf(">> add(0x%016x,%d,%d, 0x%016x,%d,%d) = (0x%016x,%d,%d)", v1,m1,e1, v2,m2,e2, v1,m1,e1)
			return v1, m1, e1
		}
	}

	e = e1

	if e1 < e2 {
		if e2-e1 < int64(len(ten_pow)) {
			h2, l2 := bits.Mul64(m2, ten_pow[e2-e1])

			if h2 != 0 {
				//log.Printf("too big add(0x%016x,%d,%d, 0x%016x,%d,%d) uses h2,l2 = %d,%d from Mul64(%d, %d (using 10^%d))", v1,m1,e1, v2,m2,e2, h2,l2, m2, ten_pow[e2-e1], e2-e1)
				//log.Printf("m2 = %d is now h2 = %d, l2 = %d is increased by %d digits", m2, h2, l2, e2-e1)

				// reduce precision so that h2, l2 is divided by p=10 ^ i appropriate so that h2 < p
				// do the same with m1 as well, so that e is updated accordingly

				// FIXME: speed the following code to avoid trying the 20 items of ten_pow : for i, p := range ten_pow {
				i := 1
				j := len(ten_pow) - 1

				// step 1 : reduce up to 10 items to examine in ten_pow
				k := (i + j) >> 1
				if h2 < ten_pow[k] {
					j = k
				} else {
					i = k
				}

				// step 2 : reduce up to 5 items to examine in ten_pow
				k = (i + j) >> 1
				if h2 < ten_pow[k] {
					j = k
				} else {
					i = k
				}

				// step 3 : reduce up to 3 items to examine in ten_pow
				k = (i + j) >> 1
				if h2 < ten_pow[k] {
					j = k
				} else {
					i = k
				}

				for k = i; k <= j; k++ {
					p := ten_pow[k]

					// FIXME: see FIXME above : for i, p := range ten_pow {
					if h2 < p {
						q2, r2 := bits.Div64(h2, l2, p)
						q1, r1 := bits.Div64(0, m1, p)
						if r2 != 0 || r1 != 0 {
							v |= loss
						}
						//log.Printf("reducing precision by %d digits (%d), m1 = %d is now %d, m2 = h2/l2 (%d/%d) is now %d", k, p, m1, q1, h2, l2, q2)
						m2 = q2
						m1 = q1
						e += int64(k)
						break
					}
				}
			} else {
				m2 = l2
			}
		} else {
			// out of range because d1 is too small compared to d2
			return v2 | loss, m2, e2
		}
	}

	// check if d1 and d2 have the same sign
	if sign&(v1^v2) == 0 {
		v |= v1 & sign
		m = m1 + m2
	} else {
		// d1 and d2 have different sign the resulting sign is the greatest mantissa
		if m1 < m2 {
			v |= v2 & sign
			m = m2 - m1
		} else {
			v |= v1 & sign
			m = m1 - m2
		}
	}

	// merge loss bit of both source
	v |= loss & (v1 | v2)

	// handle special case for zero result
	if m == 0 {
		v |= sign
		e = 0
	}

	//log.Printf(">> add(0x%016x,%d,%d, 0x%016x,%d,%d) = (0x%016x,%d,%d)", v1,m1,e1, v2,m2,e2, v,m,e)
	return
}

func vmeMulMagic1(v1, m1 uint64, e1 int64, v2, m2 uint64, e2 int64) (v, m uint64, e int64) {
	if e1 == 0 { // d1 is ~0
		// so check if d2 is NaN or infinity
		if m2 == 0 {
			if v2&loss != 0 {
				if e2 != 0 && e2 != math.MinInt64 {
					// d2 is NaN, +Inf or -Inf
					return loss, 0, 1 // return NaN
				}

				// d2 is ~0 or ~+0 or -~0
			} else {
				return sign, 0, 0 // return 0
			}
		}

		return sign | loss, 0, 0 // return ~0
	} else if e1 == math.MinInt64 { // d1 is +~0 or -~0
		// so check if d2 is NaN or infinity
		if m2 == 0 {
			if v2&loss != 0 {
				if e2 != 0 && e2 != math.MinInt64 {
					// d2 is NaN, +Inf or -Inf
					return loss, 0, 1 // return NaN
				}

				// if d2 is ~0 then result is ~0 as well as no sign exists
				if e2 == 0 {
					return sign | loss, 0, 0 // return ~0
				}
			} else {
				return sign, 0, 0 // return 0
			}
		}

		return (v1^v2)&sign | loss, 0, math.MinInt64 // return ~+0 or ~-0
	} else if e1 == math.MaxInt64 { // d1 is +Inf or -Inf
		// so check if d2 is NaN or too close to zero
		if m2 == 0 {
			if v2&loss != 0 {
				if e2 != 0 && e2 != math.MaxInt64 {
					// d2 is too close to 0 or NaN
					return loss, 0, 1 // return NaN
				}
			} else {
				return loss, 0, 1 // return NaN
			}
		}

		return (v1^v2)&sign | loss, 0, math.MaxInt64 // return +Inf or -Inf
	}

	// d1 is NaN and whatever multiplied to NaN is still NaN
	return loss, 0, 1 // NaN
}

func vmeMul(v1, m1 uint64, e1 int64, v2, m2 uint64, e2 int64) (v, m uint64, e int64) {
	// handle magic of d1
	if m1 == 0 {
		if v1&loss != 0 {
			return vmeMulMagic1(v1, m1, e1, v2, m2, e2)
		} else {
			return sign, 0, 0 // return Zero vme
		}
	}

	// handle magic of d2
	if m2 == 0 {
		if v2&loss != 0 {
			return vmeMulMagic1(v2, m2, e2, v1, m1, e1)
		} else {
			return sign, 0, 0 // return Zero vme
		}
	}

	// d1 nor d2 are zero
	v = (v1|v2)&loss | (v1^v2)&sign

	e = e1 + e2

	// make sure no overflow occurs
	if e < e1 && e2 > 0 {
		return v | loss, 0, math.MaxInt64 // return +Inf or -Inf
	} else if e > e1 && e2 < 0 {
		return v | loss, 0, math.MinInt64 // return ~+0 or ~-0
	}

	mh, m := bits.Mul64(m1, m2)

	// reduce precision if h > 0
	return vmhmeReduce(v, mh, m, e)
}

// d2 is already magic number (loss is set, mantissa is already 0)
func vmeDivRemMagic2(v1, m1 uint64, e1 int64, v2, m2 uint64, e2 int64) (v, m uint64, e int64, r uint64, re int64) {
	if e2 == 0 { // d2 is ~0
		return loss, 0, 1, 0, 0 // return NaN and remainder 0
	} else if e2 == math.MinInt64 { // d2 is +~0 or -~0
		if m1 == 0 {
			if v1&loss != 0 { // d1 is also magic
				if e1 == 0 || e1 == math.MinInt64 { // d1 is ~0, +~0 or -~0 result is also NaN
					return loss, 0, 1, 0, 0 // return NaN and remainder 0
				} else if e1 == math.MaxInt64 { // d1 is +Inf or -Inf result if +Inf or -Inf
					return loss | (v1^v2)&sign, 0, math.MaxInt64, 0, 0 // return +Inf or -Inf and remainder 0
				}
			} else { // d1 is 0 or null
				return loss, 0, 1, 0, 0 // return NaN and remainder 0
			}
		} else { // d1 is an ordinary decimal not near 0
			return loss | (v1^v2)&sign, 0, math.MaxInt64, 0, 0 // return +Inf or -Inf and remainder 0
		}
	} else if e2 == math.MaxInt64 { // d2 is +Inf or -Inf
		if m1 == 0 {
			if v1&loss != 0 { // d1 is also magic
				if e1 == 0 { // d1 is ~0
					return sign | loss, 0, 0, 0, 0 // return ~0 and remainder 0
				} else if e1 == math.MinInt64 { // d1 is ~0, +~0 or -~0 result is also +~0 or -~0
					return loss | (v1&v2)&sign, 0, math.MinInt64, 0, 0 // return +~0 or -~0 and remainder 0
				} else if e1 == math.MaxInt64 { // d1 is +Inf or -Inf result if +Inf or -Inf
					return loss, 0, 1, 0, 0 // return NaN and remainder 0
				}
			} else { // d1 is 0 or null
				return sign | loss, 0, 0, 0, 0 // return ~0 and remainder 0
			}
		} else { // d1 is an ordinary decimal not near 0
			return loss | (v1^v2)&sign, 0, math.MinInt64, 0, 0 // return ~+0 or ~-0 and remainder 0
		}
	}

	return loss, 0, 1, 0, 0 // return NaN and remainder 0
}

func vmeDivRem(v1, m1 uint64, e1 int64, v2, m2 uint64, e2 int64, precision int32) (v, m uint64, e int64, r uint64, re int64) {
	// handle magic of d2
	if m2 == 0 {
		if v2&loss != 0 {
			return vmeDivRemMagic2(v1, m1, e1, v2, m2, e2)
		} else {
			return loss, 0, 1, 0, 0 // return NaN and remainder 0
		}
	}

	// handle magic of d1
	if m1 == 0 {
		if v1&loss != 0 {
			return loss | (v1^v2)&sign, 0, e1, 0, 0 // FIXME: a lot of magic here as d1 can be ~0, +~0, -~0, +Inf, -Inf or NaN but d2 is not magic nor 0
		} else {
			return sign, 0, 0, 0, 0 // return 0 and remainder 0
		}
	}

	v = (v1|v2)&loss | (v1^v2)&sign
	e = e1 - e2 // - int64(precision)

	re = -int64(precision)
	ten_pow_i := e + int64(precision)
	if ten_pow_i < 0 {
		// FIXME: fix re as well
		re += ten_pow_i
		ten_pow_i = 0
	}
	if int(ten_pow_i) >= len(ten_pow) {
		ten_pow_i = int64(len(ten_pow) - 1)
	}
	e -= ten_pow_i
	h1, l1 := bits.Mul64(m1, ten_pow[ten_pow_i])

	// avoid panic if m2 <= h1
	if m2 <= h1 {
		for i, p := range ten_pow {
			if h1 < p {
				q, r := bits.Div64(h1, l1, p)
				if r != 0 {
					v |= loss
				}
				h1, l1 = 0, q
				e += int64(i)
				break
			}
		}
	}

	m, r = bits.Div64(h1, l1, m2)

	// FIXME: fix m and r when ten_pow_i was strictly negative
	if re < -int64(precision) {
		ten_pow_i = -int64(precision) - re
		xq, xr := bits.Div64(0, m, ten_pow[ten_pow_i])
		m = xq * ten_pow[ten_pow_i]
		r += xr * m2
	}
	re += e2

	return
}

func vmeRound(v, m uint64, e int64, places int32) (uint64, uint64, int64) {
	// no rouding nan or infinity but only 0 or near 0
	if m == 0 {
		if e == 0 || e == math.MinInt64 {
			return sign, 0, 0 // Zero
		} else {
			return v, m, e
		}
	} else {
		// clear loss bit
		v &= ^uint64(loss)

		if i := e + int64(places); i < 0 {
			if -i < int64(len(ten_pow)) {
				p := ten_pow[int(-i)]

				if (m << 1) < p {
					return sign, 0, 0 // Zero
				} else {
					q, r := bits.Div64(0, m, p)

					m = q
					if (r<<1) > p || (r<<1) == p && v&sign == 0 {
						m++
					}

					e = -int64(places)
				}
			} else {
				return sign, 0, 0 // Zero
			}
		}

		return v, m, e
	}
}

func vmeRoundBank(v, m uint64, e int64, places int32) (uint64, uint64, int64) {
	// no rouding nan or infinity but only 0 or near 0
	if m == 0 {
		if e == 0 || e == math.MinInt64 {
			return sign, 0, 0 // Zero
		} else {
			return v, m, e
		}
	} else {
		// clear loss bit
		v &= ^uint64(loss)

		if i := e + int64(places); i < 0 {
			if -i < int64(len(ten_pow)) {
				p := ten_pow[int(-i)]

				if (m << 1) < p {
					return sign, 0, 0 // Zero
				} else {
					q, r := bits.Div64(0, m, p)

					m = q
					if (r<<1) > p || (r<<1) == p && m&1 == 1 {
						m++
					}

					e = -int64(places)
				}
			} else {
				return sign, 0, 0 // Zero
			}
		}

		return v, m, e
	}
}

func vmeRoundCeil(v, m uint64, e int64, places int32) (uint64, uint64, int64) {
	// no rouding nan or infinity but only 0 or near 0
	if m == 0 {
		if e == 0 || e == math.MinInt64 {
			return sign, 0, 0 // Zero
		} else {
			return v, m, e
		}
	} else {
		// clear loss bit
		v &= ^uint64(loss)

		if i := e + int64(places); i < 0 {
			if -i < int64(len(ten_pow)) {
				p := ten_pow[int(-i)]

				if (m << 1) < p {
					if v&sign == 0 {
						return 0, 1, -int64(places) // first decimal above Zero
					} else {
						return sign, 0, 0 // Zero
					}
				} else {
					q, r := bits.Div64(0, m, p)

					m = q
					if r > 0 {
						if v&sign == 0 {
							m++
						}
					}

					e = -int64(places)
				}
			} else {
				return sign, 0, 0 // Zero
			}
		}

		return v, m, e
	}
}

func vmeRoundFloor(v, m uint64, e int64, places int32) (uint64, uint64, int64) {
	// no rouding nan or infinity but only 0 or near 0
	if m == 0 {
		if e == 0 || e == math.MinInt64 {
			return sign, 0, 0 // Zero
		} else {
			return v, m, e
		}
	} else {
		// clear loss bit
		v &= ^uint64(loss)

		if i := e + int64(places); i < 0 {
			if -i < int64(len(ten_pow)) {
				p := ten_pow[int(-i)]

				if (m << 1) < p {
					if v&sign != 0 {
						return sign, 1, -int64(places) // first decimal below Zero
					} else {
						return sign, 0, 0 // Zero
					}
				} else {
					q, r := bits.Div64(0, m, p)

					m = q
					if r > 0 {
						if v&sign != 0 {
							m++
						}
					}

					e = -int64(places)

					if m == 0 && e == 0 {
						v = sign // Zero
					}
				}
			} else {
				return sign, 0, 0 // Zero
			}
		}

		return v, m, e
	}
}

func newFromFloat(v, m2 uint64, e2 int64) Decimal {
	var m uint64
	var e int64

	z := bits.TrailingZeros64(m2)
	if z == 64 {
		if v != 0 {
			return NearNegativeZero
		} else {
			return Zero
		}
	} else {
		// normalize float as a integer mantissa instead of a fraction mantissa
		m2 = m2 >> z
		e2 += int64(z) - 63

		if fixFloatMantissa(&m2) {
			v |= loss
		}

		// normalize mantissa if negative exponent
		for e2 < 0 {
			hi, lo := bits.Mul64(m2, ten_pow[len(ten_pow)-1])
			e -= int64(len(ten_pow) - 1)
			if (lo & sign) != 0 {
				hi++
			}
			m2 = hi
			e2 += 64
		}
		// normalize mantissa if too big exponent
		for e2 >= 64 {
			q, r := bits.Div64(m2, 0, ten_pow[len(ten_pow)-1])
			e += int64(len(ten_pow) - 1)
			if r >= (ten_pow[len(ten_pow)-1] >> 1) {
				q++
			}
			m2 = q
			e2 -= 64
		}
		if e2 > 0 {
			hi := m2 >> (64 - e2)
			lo := m2 << e2
			i := len(ten_pow) - 1
			for i >= 0 && ten_pow[i] > hi {
				i--
			}
			q, r := bits.Div64(hi, lo, ten_pow[i+1])
			e += int64(i + 1)
			if r > 0 && r >= (ten_pow[i+1]>>1) {
				q++
			}
			if r != 0 {
				v |= loss
			}
			m = q
		} else {
			m = m2
		}
	}

	return vmeAsDecimal(v, m, e)
}

func fixFloatMantissa(m *uint64) bool {
	// some magic to round mantissa, try to fix small errors (only from float64)
	if *m&0xfffffffc == 0 {
		if *m&0xffffffff != 0 {
			*m &= 0xffffffff00000000

			return true
		}
	}
	if *m|0x3 == 0xffffffff {
		*m = (*m | 0x3) + 1

		return true
	}

	return false
}
