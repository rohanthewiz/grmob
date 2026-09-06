package htmlout

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// core's per-side padding props dissolve the Horizontal/Vertical shorthand
// into the sides it was standing in for before writing their own side (see
// core/padding_sides.go). That transformation is only safe because it is
// resolution-preserving: it writes the shorthand into sides that were taking
// it anyway, so every side comes out of EdgeCSS with the number it had going
// in.
//
// Measured here rather than in core because core cannot import the exporter,
// and because EdgeCSS is the contract the two natives restate — an edge that
// resolved differently after settling would diverge on all four targets at
// once, and this is the one place a test can see the resolved value.
//
// Each case applies the prop with the value that side *already* resolves to,
// so the write is an identity and the only thing under test is the settle.
func TestSettlingAnAxisPreservesEveryResolvedSide(t *testing.T) {
	cases := []struct {
		name string
		in   core.EdgeInsets
		prop core.StyleProp
	}{
		{"left over a horizontal shorthand", core.EdgeInsets{Horizontal: 16}, core.PaddingLeft(16)},
		{"right over a horizontal shorthand", core.EdgeInsets{Horizontal: 16}, core.PaddingRight(16)},
		{"top over a vertical shorthand", core.EdgeInsets{Vertical: 6}, core.PaddingTop(6)},
		{"bottom over a vertical shorthand", core.EdgeInsets{Vertical: 6}, core.PaddingBottom(6)},
		// Both shorthands present, one axis settled: the other must come
		// through untouched.
		{"left with both shorthands", core.EdgeInsets{Horizontal: 8, Vertical: 6}, core.PaddingLeft(8)},
		// An explicit side already beating its shorthand: settling must not
		// promote the shorthand over it.
		//
		// The *opposite* side is the case that matters, and it is the one a
		// first draft of this test missed: when the explicit side is the one
		// the prop then writes, the write puts the right number back and
		// hides the damage. Both orientations are here for that reason —
		// dropping settle's "only fill a side that is unset" guard passes the
		// first pair and fails the second.
		{"written side explicit", core.EdgeInsets{Horizontal: 8, Left: 20}, core.PaddingLeft(20)},
		{"opposite side explicit", core.EdgeInsets{Horizontal: 8, Right: 20}, core.PaddingLeft(8)},
		{"opposite side explicit, vertical", core.EdgeInsets{Vertical: 6, Bottom: 2}, core.PaddingTop(6)},
		// Nothing to dissolve.
		{"per-side only", core.EdgeInsets{Top: 1, Right: 2, Bottom: 3, Left: 4}, core.PaddingLeft(4)},
	}
	for _, c := range cases {
		before := EdgeCSS(c.in)
		s := core.Style{Padding: c.in}
		c.prop.Apply(&s)
		if after := EdgeCSS(s.Padding); after != before {
			t.Errorf("%s: %+v resolved %q, settled to %+v resolving %q",
				c.name, c.in, before, s.Padding, after)
		}
	}
}

// And the payoff, stated at the resolved level: a zero side prop really does
// render a zero, which is what the shorthand's "non-zero means set" rule made
// impossible before. htmlout/edges.go's doc comment used to name this exact
// pair as the rule's lossy edge; it is lossy only for a hand-built
// EdgeInsets now.
func TestAZeroSidePropResolvesToZero(t *testing.T) {
	s := core.Style{Padding: core.EdgeInsets{Horizontal: 16}}
	core.PaddingLeft(0).Apply(&s)
	if got, want := EdgeCSS(s.Padding), "0px 16px 0px 0px"; got != want {
		t.Errorf("PaddingHorizontal(16) then PaddingLeft(0) = %q, want %q", got, want)
	}
}
