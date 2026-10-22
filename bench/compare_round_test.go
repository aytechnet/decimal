package bench

import (
	"fmt"
	"testing"

	ay "github.com/aytechnet/decimal"
	ss "github.com/shopspring/decimal"
)

// TestRoundParity checks rounding parity with shopspring/decimal: every rounding entry point must give the same value on a
// grid of 4-decimal values in [-3, 3], negative halves included (a credit note must round like its
// invoice). Run from this directory: go test -run RoundParity
func TestRoundParity(t *testing.T) {
	type op struct {
		name string
		a    func(ay.Decimal, int32) string
		s    func(ss.Decimal, int32) string
		num  bool
	}
	ops := []op{
		{"Round", func(d ay.Decimal, p int32) string { return d.Round(p).String() }, func(d ss.Decimal, p int32) string { return d.Round(p).String() }, true},
		{"StringFixed", func(d ay.Decimal, p int32) string { return d.StringFixed(p) }, func(d ss.Decimal, p int32) string { return d.StringFixed(p) }, false},
		{"RoundBank", func(d ay.Decimal, p int32) string { return d.RoundBank(p).String() }, func(d ss.Decimal, p int32) string { return d.RoundBank(p).String() }, true},
		{"StringFixedBank", func(d ay.Decimal, p int32) string { return d.StringFixedBank(p) }, func(d ss.Decimal, p int32) string { return d.StringFixedBank(p) }, false},
		{"RoundUp", func(d ay.Decimal, p int32) string { return d.RoundUp(p).String() }, func(d ss.Decimal, p int32) string { return d.RoundUp(p).String() }, true},
		{"RoundDown", func(d ay.Decimal, p int32) string { return d.RoundDown(p).String() }, func(d ss.Decimal, p int32) string { return d.RoundDown(p).String() }, true},
		{"RoundCeil", func(d ay.Decimal, p int32) string { return d.RoundCeil(p).String() }, func(d ss.Decimal, p int32) string { return d.RoundCeil(p).String() }, true},
		{"RoundFloor", func(d ay.Decimal, p int32) string { return d.RoundFloor(p).String() }, func(d ss.Decimal, p int32) string { return d.RoundFloor(p).String() }, true},
		// shopspring v1.5+ also truncates the integer part for a negative precision; aytechnet does not (see places below)
		{"Truncate", func(d ay.Decimal, p int32) string { return d.Truncate(p).String() }, func(d ss.Decimal, p int32) string { return d.Truncate(p).String() }, true},
		{"DivRound3", func(d ay.Decimal, p int32) string { return d.DivRound(ay.New(3, 0), p).String() }, func(d ss.Decimal, p int32) string { return d.DivRound(ss.New(3, 0), p).String() }, true},
		{"DivRound8", func(d ay.Decimal, p int32) string { return d.DivRound(ay.New(8, 0), p).String() }, func(d ss.Decimal, p int32) string { return d.DivRound(ss.New(8, 0), p).String() }, true},
		{"RoundCash", func(d ay.Decimal, p int32) string { return d.RoundCash(uint8(p)).String() }, func(d ss.Decimal, p int32) string { return d.RoundCash(uint8(p)).String() }, true},
	}
	for _, o := range ops {
		places := []int32{-1, 0, 1, 2, 3}
		switch o.name {
		case "RoundCash":
			places = []int32{5, 10, 25, 50, 100}
		case "Truncate":
			places = []int32{0, 1, 2, 3}
		}
		bad, ex := 0, ""
		for n := int64(-30000); n <= 30000; n += 1 {
			for _, p := range places {
				da, ds := ay.New(n, -4), ss.New(n, -4)
				ga, gs := o.a(da, p), o.s(ds, p)
				eq := ga == gs
				if !eq && o.num {
					x, err := ss.NewFromString(ga)
					eq = err == nil && x.Equal(ss.RequireFromString(gs))
				}
				if !eq {
					bad++
					if len(ex) < 300 {
						ex += fmt.Sprintf(" %s@%d: aytechnet %s, shopspring %s;", ds, p, ga, gs)
					}
				}
			}
		}
		if bad > 0 {
			t.Errorf("%s: %d mismatches with shopspring:%s", o.name, bad, ex)
		}
	}
}
