package core

import "testing"

// The inset props form a three-level width lattice — Padding(all) over the
// two axis props over the four sides, and the same for Margin — and the whole
// of how they compose is one rule: a narrower prop can override or clear a
// wider one, a wider prop cannot preserve a narrower one, so the wider brush
// goes first.
//
// # Why this is a test and not just the two assertions that were here
//
// Each family already had one ordering test (TestSideAndAxisPropsOrderLike-
// EveryOtherStyleProp and its margin twin), and each checked one pair on one
// axis. That is enough to catch a side prop that forgot to settle, which is
// what those tests were written for, and it is not enough to state the rule:
// three widths give three ordered pairs per axis, and only one of the six was
// covered on either family.
//
// The gap this closes is the Next-list item it was written for. "Padding has
// no way to say clear-this-side ahead of an axis prop" is true, and the
// answer is that there must not be one — a prop whose effect survives the
// props written after it would be the one StyleProp in the package that is
// not last-one-wins, and it would be invisible at the call site of whatever
// it defeats. What the item was actually missing is that the pair *is*
// expressible, in the other order, and that nothing said so. This says so.
//
// Read the table as: apply wider, then narrower, and get the mixed result;
// apply narrower, then wider, and get the wider one's uniform result. Every
// row is both directions.
func TestTheWiderBrushGoesFirst(t *testing.T) {
	// One starting shape for every row: four explicit sides, no shorthand,
	// which is the shape a theme container arrives with.
	start := EdgeInsets{Top: 12, Bottom: 12, Left: 16, Right: 16}

	cases := []struct {
		name string
		// wider and narrower are the two props under test, and get is how
		// the family's field is read back off a Style.
		wider, narrower StyleProp
		get             func(Style) EdgeInsets
		// widerFirst is what "state the wider brush first" produces: the
		// mixed result the caller wanted. narrowerFirst is what the other
		// order produces: the wider prop's own uniform answer.
		widerFirst, narrowerFirst EdgeInsets
	}{
		// --- Padding: axis over side, both axes ---
		{
			name:  "padding horizontal then left",
			wider: PaddingHorizontal(16), narrower: PaddingLeft(0),
			get:        func(s Style) EdgeInsets { return s.Padding },
			widerFirst: EdgeInsets{Top: 12, Bottom: 12, Left: 0, Right: 16},
			// The axis prop writes both sides and its own shorthand, so the
			// zero that ran first is gone without trace.
			narrowerFirst: EdgeInsets{Top: 12, Bottom: 12, Left: 16, Right: 16, Horizontal: 16},
		},
		{
			name:  "padding vertical then top",
			wider: PaddingVertical(8), narrower: PaddingTop(0),
			get:           func(s Style) EdgeInsets { return s.Padding },
			widerFirst:    EdgeInsets{Top: 0, Bottom: 8, Left: 16, Right: 16},
			narrowerFirst: EdgeInsets{Top: 8, Bottom: 8, Left: 16, Right: 16, Vertical: 8},
		},

		// --- Padding: all over axis, and all over side ---
		{
			name:  "padding all then horizontal",
			wider: Padding(6), narrower: PaddingHorizontal(20),
			get:        func(s Style) EdgeInsets { return s.Padding },
			widerFirst: EdgeInsets{Top: 6, Bottom: 6, Left: 20, Right: 20, Horizontal: 20},
			// Padding(all) clears the shorthands outright rather than
			// settling them, which is what makes it the widest of the three.
			narrowerFirst: EdgeInsets{Top: 6, Bottom: 6, Left: 6, Right: 6},
		},
		{
			name:  "padding all then left",
			wider: Padding(6), narrower: PaddingLeft(0),
			get:           func(s Style) EdgeInsets { return s.Padding },
			widerFirst:    EdgeInsets{Top: 6, Bottom: 6, Left: 0, Right: 6},
			narrowerFirst: EdgeInsets{Top: 6, Bottom: 6, Left: 6, Right: 6},
		},

		// --- Margin: the identical lattice on the other family ---
		{
			name:  "margin horizontal then left",
			wider: MarginHorizontal(16), narrower: MarginLeft(0),
			get:           func(s Style) EdgeInsets { return s.Margin },
			widerFirst:    EdgeInsets{Top: 12, Bottom: 12, Left: 0, Right: 16},
			narrowerFirst: EdgeInsets{Top: 12, Bottom: 12, Left: 16, Right: 16, Horizontal: 16},
		},
		{
			name:  "margin vertical then top",
			wider: MarginVertical(8), narrower: MarginTop(0),
			get:           func(s Style) EdgeInsets { return s.Margin },
			widerFirst:    EdgeInsets{Top: 0, Bottom: 8, Left: 16, Right: 16},
			narrowerFirst: EdgeInsets{Top: 8, Bottom: 8, Left: 16, Right: 16, Vertical: 8},
		},
		{
			name:  "margin all then horizontal",
			wider: Margin(6), narrower: MarginHorizontal(20),
			get:           func(s Style) EdgeInsets { return s.Margin },
			widerFirst:    EdgeInsets{Top: 6, Bottom: 6, Left: 20, Right: 20, Horizontal: 20},
			narrowerFirst: EdgeInsets{Top: 6, Bottom: 6, Left: 6, Right: 6},
		},
		{
			name:  "margin all then left",
			wider: Margin(6), narrower: MarginLeft(0),
			get:           func(s Style) EdgeInsets { return s.Margin },
			widerFirst:    EdgeInsets{Top: 6, Bottom: 6, Left: 0, Right: 6},
			narrowerFirst: EdgeInsets{Top: 6, Bottom: 6, Left: 6, Right: 6},
		},
	}

	for _, c := range cases {
		// Both families start from the same shape, applied to whichever
		// field the row's getter reads.
		fresh := func() Style { return Style{Padding: start, Margin: start} }

		s := fresh()
		c.wider.Apply(&s)
		c.narrower.Apply(&s)
		if got := c.get(s); got != c.widerFirst {
			t.Errorf("%s (wider first) = %+v, want %+v — the narrower prop must be able "+
				"to override or clear the wider one", c.name, got, c.widerFirst)
		}

		s = fresh()
		c.narrower.Apply(&s)
		c.wider.Apply(&s)
		if got := c.get(s); got != c.narrowerFirst {
			t.Errorf("%s (narrower first) = %+v, want %+v — a wider prop writes every "+
				"side it covers and must not preserve what ran before it", c.name, got,
				c.narrowerFirst)
		}
	}
}

// The two orders must actually differ, on every row, or the test above is
// asserting a tautology.
//
// This is the assertion that carries the rule's content. If some future edit
// made an axis prop merge rather than overwrite — "keep a side that was
// already explicit" is a plausible-looking change, and it is exactly what the
// Next-list item asked for — every row's two orders would converge and the
// table above would still pass, because both expectations would simply be
// rewritten to the same value by whoever made the change.
func TestTheTwoOrdersAreDistinguishable(t *testing.T) {
	start := EdgeInsets{Top: 12, Bottom: 12, Left: 16, Right: 16}

	pairs := []struct {
		name            string
		wider, narrower StyleProp
		get             func(Style) EdgeInsets
	}{
		{"padding axis/side", PaddingHorizontal(16), PaddingLeft(0),
			func(s Style) EdgeInsets { return s.Padding }},
		{"padding all/axis", Padding(6), PaddingHorizontal(20),
			func(s Style) EdgeInsets { return s.Padding }},
		{"padding all/side", Padding(6), PaddingLeft(0),
			func(s Style) EdgeInsets { return s.Padding }},
		{"margin axis/side", MarginHorizontal(16), MarginLeft(0),
			func(s Style) EdgeInsets { return s.Margin }},
		{"margin all/axis", Margin(6), MarginHorizontal(20),
			func(s Style) EdgeInsets { return s.Margin }},
		{"margin all/side", Margin(6), MarginLeft(0),
			func(s Style) EdgeInsets { return s.Margin }},
	}

	for _, p := range pairs {
		a := Style{Padding: start, Margin: start}
		p.wider.Apply(&a)
		p.narrower.Apply(&a)

		b := Style{Padding: start, Margin: start}
		p.narrower.Apply(&b)
		p.wider.Apply(&b)

		if p.get(a) == p.get(b) {
			t.Errorf("%s: both orders give %+v — prop order has stopped mattering for "+
				"this pair, which means either the narrower prop no longer settles its "+
				"axis or the wider one no longer overwrites", p.name, p.get(a))
		}
	}
}
