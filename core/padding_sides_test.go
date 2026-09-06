package core

import "testing"

// themePadding is the shape a container arrives with: four explicit sides,
// no shorthand. The theme's Column and Row both look like this.
var themePadding = EdgeInsets{Top: 12, Bottom: 12, Left: 16, Right: 16}

// Each side prop writes its own side and leaves the other three exactly as
// they were — which is the whole point, since the alternative (a whole
// EdgeInsets through UseStyle) replaces all four and forces the caller to
// copy the theme's numbers into their screen.
func TestEachSidePropTouchesOnlyItsOwnSide(t *testing.T) {
	cases := []struct {
		name string
		prop StyleProp
		want EdgeInsets
	}{
		{"top", PaddingTop(4), EdgeInsets{Top: 4, Bottom: 12, Left: 16, Right: 16}},
		{"bottom", PaddingBottom(4), EdgeInsets{Top: 12, Bottom: 4, Left: 16, Right: 16}},
		{"left", PaddingLeft(32), EdgeInsets{Top: 12, Bottom: 12, Left: 32, Right: 16}},
		{"right", PaddingRight(32), EdgeInsets{Top: 12, Bottom: 12, Left: 16, Right: 32}},
	}
	for _, c := range cases {
		s := Style{Padding: themePadding}
		c.prop.Apply(&s)
		if s.Padding != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, s.Padding, c.want)
		}
	}
}

// A zero must clear. Without the axis settle this is the case that silently
// fails: the renderers read a zero side as "unset" and fall back to the
// axis shorthand, so PaddingLeft(0) over a Horizontal 16 used to render 16px
// on every target while looking like it had applied.
//
// Both starting shapes matter. Over explicit sides the assignment alone is
// enough; over a shorthand it is not, and the second half of each pair is
// what the settle exists for.
func TestASidePropCanClearWhatCameBeforeIt(t *testing.T) {
	cases := []struct {
		name  string
		start EdgeInsets
		prop  StyleProp
		want  EdgeInsets
	}{
		// Over explicit sides: the opposite side survives untouched.
		{"top over sides", themePadding, PaddingTop(0),
			EdgeInsets{Bottom: 12, Left: 16, Right: 16}},
		{"left over sides", themePadding, PaddingLeft(0),
			EdgeInsets{Top: 12, Bottom: 12, Right: 16}},

		// Over a raw shorthand: the shorthand is dissolved into the opposite
		// side and cleared, so the zero is the only thing describing this one.
		{"left over Horizontal", EdgeInsets{Horizontal: 16}, PaddingLeft(0),
			EdgeInsets{Right: 16}},
		{"right over Horizontal", EdgeInsets{Horizontal: 16}, PaddingRight(0),
			EdgeInsets{Left: 16}},
		{"top over Vertical", EdgeInsets{Vertical: 8}, PaddingTop(0),
			EdgeInsets{Bottom: 8}},
		{"bottom over Vertical", EdgeInsets{Vertical: 8}, PaddingBottom(0),
			EdgeInsets{Top: 8}},

		// Over the shorthand prop, which writes both the sides and the
		// shorthand field: the settle has to clear the latter or the zero
		// side falls back to it.
		{"left over PaddingHorizontal", EdgeInsets{Horizontal: 16, Left: 16, Right: 16},
			PaddingLeft(0), EdgeInsets{Right: 16}},
	}
	for _, c := range cases {
		s := Style{Padding: c.start}
		c.prop.Apply(&s)
		if s.Padding != c.want {
			t.Errorf("%s: %+v -> %+v, want %+v", c.name, c.start, s.Padding, c.want)
		}
	}
}

// Settling must not leave the shorthand field set, or a later renderer read
// resolves the *other* side off a value the props have already superseded.
// Stated separately from the value checks above because it is the invariant
// rather than an outcome: once a side prop has touched an axis, that axis is
// described per-side and nowhere else.
func TestASidePropLeavesItsAxisStatedPerSide(t *testing.T) {
	for _, c := range []struct {
		name  string
		start EdgeInsets
		prop  StyleProp
		axis  func(EdgeInsets) int
	}{
		{"horizontal", EdgeInsets{Horizontal: 16}, PaddingLeft(24), func(e EdgeInsets) int { return e.Horizontal }},
		{"horizontal", EdgeInsets{Horizontal: 16}, PaddingRight(24), func(e EdgeInsets) int { return e.Horizontal }},
		{"vertical", EdgeInsets{Vertical: 8}, PaddingTop(2), func(e EdgeInsets) int { return e.Vertical }},
		{"vertical", EdgeInsets{Vertical: 8}, PaddingBottom(2), func(e EdgeInsets) int { return e.Vertical }},
	} {
		s := Style{Padding: c.start}
		c.prop.Apply(&s)
		if got := c.axis(s.Padding); got != 0 {
			t.Errorf("%s shorthand survived a side prop: %+v", c.name, s.Padding)
		}
	}
	// Settling fills only the side that was actually taking the shorthand.
	// An explicit opposite side has already superseded it and must survive —
	// without this guard, PaddingLeft would quietly reset Right to the
	// shorthand on its way past, and the prop's own write would hide it on
	// the side under test.
	for _, c := range []struct {
		name  string
		start EdgeInsets
		prop  StyleProp
		want  EdgeInsets
	}{
		{"left keeps an explicit right", EdgeInsets{Horizontal: 8, Right: 20},
			PaddingLeft(4), EdgeInsets{Left: 4, Right: 20}},
		{"right keeps an explicit left", EdgeInsets{Horizontal: 8, Left: 20},
			PaddingRight(4), EdgeInsets{Left: 20, Right: 4}},
		{"top keeps an explicit bottom", EdgeInsets{Vertical: 6, Bottom: 2},
			PaddingTop(1), EdgeInsets{Top: 1, Bottom: 2}},
		{"bottom keeps an explicit top", EdgeInsets{Vertical: 6, Top: 2},
			PaddingBottom(1), EdgeInsets{Top: 2, Bottom: 1}},
	} {
		s := Style{Padding: c.start}
		c.prop.Apply(&s)
		if s.Padding != c.want {
			t.Errorf("%s: %+v -> %+v, want %+v", c.name, c.start, s.Padding, c.want)
		}
	}

	// And the untouched axis is left alone: a horizontal prop must not
	// dissolve Vertical, which nothing has superseded.
	s := Style{Padding: EdgeInsets{Horizontal: 16, Vertical: 8}}
	PaddingLeft(24).Apply(&s)
	if s.Padding.Vertical != 8 {
		t.Errorf("PaddingLeft settled the vertical axis too: %+v", s.Padding)
	}
}

// Last one wins, in both directions: a side after an axis narrows it, and an
// axis after a side overwrites it. This is the ordering every other StyleProp
// has, and it is the reason PaddingHorizontal writes the explicit sides as
// well as the shorthand.
func TestSideAndAxisPropsOrderLikeEveryOtherStyleProp(t *testing.T) {
	s := Style{Padding: themePadding}
	PaddingHorizontal(8).Apply(&s)
	PaddingLeft(40).Apply(&s)
	if s.Padding.Left != 40 || s.Padding.Right != 8 {
		t.Errorf("side after axis: %+v, want Left 40 / Right 8", s.Padding)
	}

	s = Style{Padding: themePadding}
	PaddingLeft(40).Apply(&s)
	PaddingHorizontal(8).Apply(&s)
	if s.Padding.Left != 8 || s.Padding.Right != 8 {
		t.Errorf("axis after side: %+v, want 8 on both", s.Padding)
	}

	// Padding(all) is the widest and clears the shorthands outright, so it
	// resets whatever a side prop had settled.
	s = Style{Padding: EdgeInsets{Horizontal: 16}}
	PaddingLeft(0).Apply(&s)
	Padding(6).Apply(&s)
	if s.Padding != (EdgeInsets{Top: 6, Right: 6, Bottom: 6, Left: 6}) {
		t.Errorf("Padding(6) after a settled side: %+v", s.Padding)
	}
}

// The four props compose into a full inset set without any of them undoing
// another — the shape a caller reaches for when they want three of the four
// sides different.
func TestTheFourSidePropsCompose(t *testing.T) {
	s := Style{}
	PaddingTop(1).Apply(&s)
	PaddingRight(2).Apply(&s)
	PaddingBottom(3).Apply(&s)
	PaddingLeft(4).Apply(&s)
	if s.Padding != (EdgeInsets{Top: 1, Right: 2, Bottom: 3, Left: 4}) {
		t.Errorf("four side props = %+v, want 1/2/3/4", s.Padding)
	}
}
