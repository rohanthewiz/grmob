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

// The margin side props settle the same two axes with the same two helpers,
// and the margin CSS comes out of the same EdgeCSS. Held here for the two
// facts core's own tests cannot see.
//
// The first is that the transformation is resolution-preserving on this
// field too. That follows from settleHorizontal/settleVertical being shared
// verbatim and is not really in doubt — but "shared verbatim" is a fact
// about today's source, and this is the assertion that would survive someone
// giving margin its own copy of the helpers.
//
// The second is the payoff, and it is the one that could not be stated in
// core at all: a zero margin side really does render a zero. That is what
// the shorthand's "non-zero means set" rule made impossible before, and it
// is why MarginLeft(0) is a prop a caller can rely on rather than one that
// looks applied and resolves back to the theme's number.
func TestSettlingAMarginAxisPreservesEveryResolvedSide(t *testing.T) {
	cases := []struct {
		name string
		in   core.EdgeInsets
		prop core.StyleProp
	}{
		{"left over a horizontal shorthand", core.EdgeInsets{Horizontal: 16}, core.MarginLeft(16)},
		{"right over a horizontal shorthand", core.EdgeInsets{Horizontal: 16}, core.MarginRight(16)},
		{"top over a vertical shorthand", core.EdgeInsets{Vertical: 6}, core.MarginTop(6)},
		{"bottom over a vertical shorthand", core.EdgeInsets{Vertical: 6}, core.MarginBottom(6)},
		{"left with both shorthands", core.EdgeInsets{Horizontal: 8, Vertical: 6}, core.MarginLeft(8)},
		{"opposite side explicit", core.EdgeInsets{Horizontal: 8, Right: 20}, core.MarginLeft(8)},
		{"per-side only", core.EdgeInsets{Top: 1, Right: 2, Bottom: 3, Left: 4}, core.MarginLeft(4)},
	}
	for _, c := range cases {
		before := EdgeCSS(c.in)
		s := core.Style{Margin: c.in}
		c.prop.Apply(&s)
		if after := EdgeCSS(s.Margin); after != before {
			t.Errorf("%s: %+v resolved %q, settled to %+v resolving %q",
				c.name, c.in, before, s.Margin, after)
		}
	}
}

// A zero margin side resolves to zero, and an axis prop resolves to the pair
// it names — the two workaround shapes, at the level the renderers read.
func TestTheMarginPropsResolveAsWritten(t *testing.T) {
	for _, c := range []struct {
		name  string
		start core.EdgeInsets
		props []core.StyleProp
		want  string
	}{
		{"a zero side clears its shorthand", core.EdgeInsets{Horizontal: 16},
			[]core.StyleProp{core.MarginLeft(0)}, "0px 16px 0px 0px"},
		{"the inset rule", core.EdgeInsets{},
			[]core.StyleProp{core.MarginHorizontal(16)}, "0px 16px 0px 16px"},
		{"the bubble gap", core.EdgeInsets{},
			[]core.StyleProp{core.MarginBottom(8)}, "0px 0px 8px 0px"},
	} {
		s := core.Style{Margin: c.start}
		for _, p := range c.props {
			p.Apply(&s)
		}
		if got := EdgeCSS(s.Margin); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}
