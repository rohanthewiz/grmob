package core

import "testing"

// themeMargin is the shape a caller arrives with once anything has set a
// margin at all: four explicit sides, no shorthand. Margin(all) writes this.
var themeMargin = EdgeInsets{Top: 12, Bottom: 12, Left: 16, Right: 16}

// Each side prop writes its own side and leaves the other three exactly as
// they were.
//
// This matters more for margin than it did for padding. The alternative — a
// whole EdgeInsets through UseStyle — replaces all four, and a margin's other
// three sides are usually zero, so the replacement looks like it did nothing
// while clearing every gap something else had asked for.
func TestEachMarginSidePropTouchesOnlyItsOwnSide(t *testing.T) {
	cases := []struct {
		name string
		prop StyleProp
		want EdgeInsets
	}{
		{"top", MarginTop(4), EdgeInsets{Top: 4, Bottom: 12, Left: 16, Right: 16}},
		{"bottom", MarginBottom(4), EdgeInsets{Top: 12, Bottom: 4, Left: 16, Right: 16}},
		{"left", MarginLeft(32), EdgeInsets{Top: 12, Bottom: 12, Left: 32, Right: 16}},
		{"right", MarginRight(32), EdgeInsets{Top: 12, Bottom: 12, Left: 16, Right: 32}},
	}
	for _, c := range cases {
		s := Style{Margin: themeMargin}
		c.prop.Apply(&s)
		if s.Margin != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, s.Margin, c.want)
		}
	}
}

// A zero must clear, over explicit sides and over a raw shorthand alike. The
// second half is what settleHorizontal/settleVertical exist for; see
// core/padding_sides.go for the derivation.
func TestAMarginSidePropCanClearWhatCameBeforeIt(t *testing.T) {
	cases := []struct {
		name  string
		start EdgeInsets
		prop  StyleProp
		want  EdgeInsets
	}{
		{"top over sides", themeMargin, MarginTop(0),
			EdgeInsets{Bottom: 12, Left: 16, Right: 16}},
		{"left over sides", themeMargin, MarginLeft(0),
			EdgeInsets{Top: 12, Bottom: 12, Right: 16}},

		{"left over Horizontal", EdgeInsets{Horizontal: 16}, MarginLeft(0),
			EdgeInsets{Right: 16}},
		{"right over Horizontal", EdgeInsets{Horizontal: 16}, MarginRight(0),
			EdgeInsets{Left: 16}},
		{"top over Vertical", EdgeInsets{Vertical: 8}, MarginTop(0),
			EdgeInsets{Bottom: 8}},
		{"bottom over Vertical", EdgeInsets{Vertical: 8}, MarginBottom(0),
			EdgeInsets{Top: 8}},

		// Over the axis prop, which writes both the sides and the shorthand:
		// the settle has to clear the latter or the zero side falls back to it.
		{"left over MarginHorizontal", EdgeInsets{Horizontal: 16, Left: 16, Right: 16},
			MarginLeft(0), EdgeInsets{Right: 16}},
		{"top over MarginVertical", EdgeInsets{Vertical: 8, Top: 8, Bottom: 8},
			MarginTop(0), EdgeInsets{Bottom: 8}},
	}
	for _, c := range cases {
		s := Style{Margin: c.start}
		c.prop.Apply(&s)
		if s.Margin != c.want {
			t.Errorf("%s: %+v -> %+v, want %+v", c.name, c.start, s.Margin, c.want)
		}
	}
}

// Once a side prop has touched an axis, that axis is described per-side and
// nowhere else — otherwise a renderer resolves the *other* side off a value
// the props have already superseded.
//
// The second table is the one that catches a dropped "only if the side is
// unset" guard: when the explicit side is the one the prop then writes, the
// write puts the right number back and hides the damage, so both orientations
// are here.
func TestAMarginSidePropLeavesItsAxisStatedPerSide(t *testing.T) {
	for _, c := range []struct {
		name  string
		start EdgeInsets
		prop  StyleProp
		axis  func(EdgeInsets) int
	}{
		{"horizontal", EdgeInsets{Horizontal: 16}, MarginLeft(24), func(e EdgeInsets) int { return e.Horizontal }},
		{"horizontal", EdgeInsets{Horizontal: 16}, MarginRight(24), func(e EdgeInsets) int { return e.Horizontal }},
		{"vertical", EdgeInsets{Vertical: 8}, MarginTop(2), func(e EdgeInsets) int { return e.Vertical }},
		{"vertical", EdgeInsets{Vertical: 8}, MarginBottom(2), func(e EdgeInsets) int { return e.Vertical }},
	} {
		s := Style{Margin: c.start}
		c.prop.Apply(&s)
		if got := c.axis(s.Margin); got != 0 {
			t.Errorf("%s shorthand survived a margin side prop: %+v", c.name, s.Margin)
		}
	}

	for _, c := range []struct {
		name  string
		start EdgeInsets
		prop  StyleProp
		want  EdgeInsets
	}{
		{"left keeps an explicit right", EdgeInsets{Horizontal: 8, Right: 20},
			MarginLeft(4), EdgeInsets{Left: 4, Right: 20}},
		{"right keeps an explicit left", EdgeInsets{Horizontal: 8, Left: 20},
			MarginRight(4), EdgeInsets{Left: 20, Right: 4}},
		{"top keeps an explicit bottom", EdgeInsets{Vertical: 6, Bottom: 2},
			MarginTop(1), EdgeInsets{Top: 1, Bottom: 2}},
		{"bottom keeps an explicit top", EdgeInsets{Vertical: 6, Top: 2},
			MarginBottom(1), EdgeInsets{Top: 2, Bottom: 1}},
	} {
		s := Style{Margin: c.start}
		c.prop.Apply(&s)
		if s.Margin != c.want {
			t.Errorf("%s: %+v -> %+v, want %+v", c.name, c.start, s.Margin, c.want)
		}
	}

	// The untouched axis is left alone.
	s := Style{Margin: EdgeInsets{Horizontal: 16, Vertical: 8}}
	MarginLeft(24).Apply(&s)
	if s.Margin.Vertical != 8 {
		t.Errorf("MarginLeft settled the vertical axis too: %+v", s.Margin)
	}
}

// Last one wins, in both directions, exactly as the padding family orders.
func TestMarginSideAndAxisPropsOrderLikeEveryOtherStyleProp(t *testing.T) {
	s := Style{Margin: themeMargin}
	MarginHorizontal(8).Apply(&s)
	MarginLeft(40).Apply(&s)
	if s.Margin.Left != 40 || s.Margin.Right != 8 {
		t.Errorf("side after axis: %+v, want Left 40 / Right 8", s.Margin)
	}

	s = Style{Margin: themeMargin}
	MarginLeft(40).Apply(&s)
	MarginHorizontal(8).Apply(&s)
	if s.Margin.Left != 8 || s.Margin.Right != 8 {
		t.Errorf("axis after side: %+v, want 8 on both", s.Margin)
	}

	s = Style{Margin: themeMargin}
	MarginVertical(2).Apply(&s)
	MarginTop(9).Apply(&s)
	if s.Margin.Top != 9 || s.Margin.Bottom != 2 {
		t.Errorf("top after vertical: %+v, want Top 9 / Bottom 2", s.Margin)
	}

	// Margin(all) is the widest and clears the shorthands outright.
	s = Style{Margin: EdgeInsets{Horizontal: 16}}
	MarginLeft(0).Apply(&s)
	Margin(6).Apply(&s)
	if s.Margin != (EdgeInsets{Top: 6, Right: 6, Bottom: 6, Left: 6}) {
		t.Errorf("Margin(6) after a settled side: %+v", s.Margin)
	}
}

// The four sides compose into a full set without any of them undoing another.
func TestTheFourMarginSidePropsCompose(t *testing.T) {
	s := Style{}
	MarginTop(1).Apply(&s)
	MarginRight(2).Apply(&s)
	MarginBottom(3).Apply(&s)
	MarginLeft(4).Apply(&s)
	if s.Margin != (EdgeInsets{Top: 1, Right: 2, Bottom: 3, Left: 4}) {
		t.Errorf("four margin side props = %+v, want 1/2/3/4", s.Margin)
	}
}

// The margin props must not touch Padding, and the padding props must not
// touch Margin.
//
// Stated because this is the one way the two families can go wrong that no
// value check above would notice. Every prop here is a copy of its padding
// twin with one identifier changed, and the identifier is the *argument to
// the settle helper* — settleHorizontal(&s.Padding) inside MarginLeft
// compiles, type-checks, and produces a prop that clears a container's left
// padding while setting its left margin. Both directions, because the same
// slip is available in either file.
func TestTheMarginAndPaddingFamiliesDoNotReachIntoEachOther(t *testing.T) {
	both := EdgeInsets{Horizontal: 16, Vertical: 8}

	for _, c := range []struct {
		name string
		prop StyleProp
	}{
		{"MarginTop", MarginTop(3)},
		{"MarginBottom", MarginBottom(3)},
		{"MarginLeft", MarginLeft(3)},
		{"MarginRight", MarginRight(3)},
		{"MarginHorizontal", MarginHorizontal(3)},
		{"MarginVertical", MarginVertical(3)},
		{"Margin", Margin(3)},
	} {
		s := Style{Padding: both, Margin: both}
		c.prop.Apply(&s)
		if s.Padding != both {
			t.Errorf("%s changed Padding: %+v -> %+v", c.name, both, s.Padding)
		}
		if s.Margin == both {
			t.Errorf("%s left Margin untouched at %+v — the prop did nothing", c.name, s.Margin)
		}
	}

	for _, c := range []struct {
		name string
		prop StyleProp
	}{
		{"PaddingTop", PaddingTop(3)},
		{"PaddingBottom", PaddingBottom(3)},
		{"PaddingLeft", PaddingLeft(3)},
		{"PaddingRight", PaddingRight(3)},
		{"PaddingHorizontal", PaddingHorizontal(3)},
		{"PaddingVertical", PaddingVertical(3)},
		{"Padding", Padding(3)},
	} {
		s := Style{Padding: both, Margin: both}
		c.prop.Apply(&s)
		if s.Margin != both {
			t.Errorf("%s changed Margin: %+v -> %+v", c.name, both, s.Margin)
		}
		if s.Padding == both {
			t.Errorf("%s left Padding untouched at %+v — the prop did nothing", c.name, s.Padding)
		}
	}
}

// The two shapes that were live workarounds, written as the props that
// replace them. Both used to go through UseStyle with a whole EdgeInsets,
// which on this field silently clears the three sides it does not mention.
func TestTheMarginWorkaroundShapesAreOnePropEach(t *testing.T) {
	// components/separator.go: an inset rule.
	s := Style{}
	MarginHorizontal(16).Apply(&s)
	if s.Margin.Left != 16 || s.Margin.Right != 16 || s.Margin.Top != 0 || s.Margin.Bottom != 0 {
		t.Errorf("MarginHorizontal(16) = %+v, want the horizontal pair alone", s.Margin)
	}

	// examples/chat: the gap under a message bubble.
	s = Style{}
	MarginBottom(8).Apply(&s)
	if s.Margin != (EdgeInsets{Bottom: 8}) {
		t.Errorf("MarginBottom(8) = %+v, want Bottom 8 alone", s.Margin)
	}
}
