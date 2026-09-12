package bandfixture

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The fixture is the band the framework actually builds.
//
// Cases reads its numbers off a rendered comps.GroupHeader, which is what
// keeps it from being a copy — but "reads them off the widget" is only worth
// anything if the numbers are the theme's recipe rather than whatever happened
// to come back. This is that half: the band's chrome is
// Spacing.MD horizontally, Spacing.XS vertically and Spacing.SM before the
// badge, and a band that quietly started spending literals would pass every
// Swift check downstream while having stopped being the theme's band.
func TestTheArrangementIsTheThemesOwnSpacing(t *testing.T) {
	sp := core.DefaultTheme.Spacing
	withBadge := Cases()[0].Now

	for _, c := range []struct {
		what string
		got  float64
		want float64
	}{
		{"the control's leading inset", withBadge.Control.Left, float64(sp.MD)},
		{"the control's trailing inset (the gap before the badge)",
			withBadge.Control.Right, float64(sp.SM)},
		{"the control's top inset", withBadge.Control.Top, float64(sp.XS)},
		{"the control's bottom inset", withBadge.Control.Bottom, float64(sp.XS)},
		{"the Row's trailing inset (the badge's own)", withBadge.Row.Right, float64(sp.MD)},
	} {
		if c.got != c.want {
			t.Errorf("%s is %v, and the theme's own step is %v — the band has stopped "+
				"spending the spacing scale, so the geometry ios/verify checks is no "+
				"longer the theme's", c.what, c.got, c.want)
		}
	}

	// The three insets that must be *gone* from the Row, because that is the
	// move: an inset left behind on the Row would be counted twice.
	for _, c := range []struct {
		what string
		got  float64
	}{
		{"the Row's leading inset", withBadge.Row.Left},
		{"the Row's top inset", withBadge.Row.Top},
		{"the Row's bottom inset", withBadge.Row.Bottom},
		{"the Row's gap", withBadge.Gap},
	} {
		if c.got != 0 {
			t.Errorf("%s is %v and should be zero — the insets moved onto the control, "+
				"so anything still on the Row is chrome the band pays for twice",
				c.what, c.got)
		}
	}
}

// Rewind is the move read backwards, and nothing is lost either way.
//
// The two arrangements have to enclose the same content in the same space, and
// the checkable form of that is the total: the same horizontal chrome (insets
// plus the gap) and the same vertical chrome, whichever node is carrying it. A
// Rewind that dropped an inset would produce a "before" that is narrower than
// the band ever was, and every comparison in ios/verify would then be between
// two arrangements that were never the same band.
func TestRewindMovesTheChromeWithoutLosingAny(t *testing.T) {
	for _, c := range Cases() {
		for _, axis := range []struct {
			what        string
			now, before float64
		}{
			{"horizontal chrome",
				c.Now.Row.Left + c.Now.Row.Right + c.Now.Gap +
					c.Now.Control.Left + c.Now.Control.Right,
				c.Before.Row.Left + c.Before.Row.Right + c.Before.Gap +
					c.Before.Control.Left + c.Before.Control.Right},
			{"vertical chrome",
				c.Now.Row.Top + c.Now.Row.Bottom + c.Now.Control.Top + c.Now.Control.Bottom,
				c.Before.Row.Top + c.Before.Row.Bottom +
					c.Before.Control.Top + c.Before.Control.Bottom},
		} {
			if axis.now != axis.before {
				t.Errorf("%s: %s is %v now and %v before the move — the two "+
					"arrangements are not the same band, so comparing them settles "+
					"nothing", c.What, axis.what, axis.now, axis.before)
			}
		}
		if c.Before.Control != (Insets{}) {
			t.Errorf("%s: the rewound arrangement leaves %+v on the control — before "+
				"the move the control had none, which is the whole difference between "+
				"the two", c.What, c.Before.Control)
		}
		if c.Now.Grow != c.Before.Grow {
			t.Errorf("%s: the growing child's weight changed with the arrangement "+
				"(%v -> %v); the move was about padding and nothing else",
				c.What, c.Now.Grow, c.Before.Grow)
		}
	}
}

// The real band's control is the taller child, which is why the cross-axis
// divergence is a made-up case rather than a defect.
//
// ios/verify asserts that the two arrangements produce the same band height,
// and that assertion has a precondition: with align-items centre, a Row's
// height is its tallest child plus its own vertical padding, so moving that
// padding onto one child stops it being added to the other. The heights agree
// exactly while the padded control is the taller child.
//
// Half of that is a text measurement and is nobody's here to make. The other
// half is these insets, and it is the half that could change under an edit: the
// control carries more vertical padding than the badge, so the control wins
// unless the badge's *type* is taller — and both are the theme's Caption.
func TestTheControlIsPaddedMoreThanTheBadge(t *testing.T) {
	c := Cases()[0]
	control := c.Now.Control.Top + c.Now.Control.Bottom
	badge := c.BadgeInsets.Top + c.BadgeInsets.Bottom
	if control <= badge {
		t.Errorf("the control's vertical insets total %v and the badge's %v — the band "+
			"height only survives the move while the padded control is the taller "+
			"child, and with the badge padded at least as much that now depends "+
			"entirely on which of two caption-sized runs of text measures taller",
			control, badge)
	}
}

// Every case gives the solver something to distribute, and something to fail on.
//
// A case whose offers were all wider than its content would exercise the grow
// arm alone; one with no negative offer would never ask what the two
// arrangements hug to, which is the one question answered by a sum rather than
// a distribution. Both are easy to lose in a fixture edit and neither would
// fail anything downstream — the Swift check would simply agree about less.
func TestEveryCaseOffersBothAWideAndANarrowContainer(t *testing.T) {
	for _, c := range Cases() {
		natural := c.Label.W + c.Now.Control.Left + c.Now.Control.Right +
			c.Badge.W + c.Now.Row.Left + c.Now.Row.Right
		var wider, narrower, indefinite bool
		for _, o := range c.Offers {
			switch {
			case o < 0:
				indefinite = true
			case o > natural:
				wider = true
			case o < natural:
				narrower = true
			}
		}
		if !wider || !narrower || !indefinite {
			t.Errorf("%s: offers %v against a natural width of %v — a case wants one "+
				"offer with slack to grow into, one too narrow so the shrink arm runs, "+
				"and an indefinite one so the hug is compared too",
				c.What, c.Offers, natural)
		}
	}
}

// SharesADeficit is a fact about the case, not a flag somebody set.
//
// It decides which way band.swift asserts the overflow arm, so a case with it
// wrong would assert the opposite of the truth and pass by agreeing with
// itself. What it actually means is "there is more than one child on the line",
// which is exactly whether the band has a badge — a lone growing control is
// clamped to the container in either arrangement, so there is no deficit to
// divide and no divergence to see.
func TestSharesADeficitIsWhetherTheBandHasABadge(t *testing.T) {
	for _, c := range Cases() {
		if want := c.Badge.W > 0; c.SharesADeficit != want {
			t.Errorf("%s: SharesADeficit is %v and the band %s a badge — the flag is "+
				"whether a second child is on the line to divide an overflow with, and "+
				"ios/verify asserts the opposite arm on the strength of it",
				c.What, c.SharesADeficit,
				map[bool]string{true: "has", false: "does not have"}[want])
		}
	}
}

// The band centres its children, which is what the cross-axis comparison rests
// on.
//
// ios/verify's height model is "the tallest child, plus the Row's own vertical
// padding" — the centred case. Under `stretch` every child would be the Row's
// height instead, the padding would end up inside them, and the comparison
// would be modelling a layout the framework does not build. band.swift refuses
// a case that is not centred; this is the half that says which one that is, in
// the package that reads it off the widget.
func TestTheBandCentresItsChildren(t *testing.T) {
	for _, c := range Cases() {
		for _, a := range []Arrangement{c.Now, c.Before} {
			if a.Align != string(core.AlignItemsCenter) {
				t.Errorf("%s: the band aligns its children %q, and the geometry "+
					"ios/verify compares is only that geometry when they are centred",
					c.What, a.Align)
			}
		}
	}
}
