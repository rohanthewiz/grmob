package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// A caller's side prop reaches past a widget's own inset defaults.
//
// # What this is for
//
// core/padding_sides.go states the composition rule for the inset props: the
// wider brush goes first, because a side prop settles its axis and can clear
// it while an axis prop overwrites both sides and cannot preserve one. Every
// (axis, side) pair is therefore expressible, in exactly one order.
//
// That rule is only useful to a caller if the caller's props are the *last*
// ones applied. Every widget in this package documents that its Style field
// is applied after its own defaults — "Style is applied last, so every
// default above is overridable" is the phrasing Separator uses — and until
// now that was a promise made twenty-odd times in prose and checked nowhere.
// A widget that appended one inset prop after its Style loop would take the
// last word away from the caller, and the ordering rule would stop being
// reachable for that widget with nothing failing.
//
// # Why these five
//
// The census is over widgets whose *root node* carries a non-zero inset
// default, since those are the only ones where a caller's side prop has
// anything to reach past. They divide into the two shapes that matter:
//
//	explicit sides       Card (padding and margin), ListRow
//	                     the caller's assignment is the whole of it
//
//	a live shorthand     GroupHeader, SearchField, Separator
//	                     the widget's default arrives as EdgeInsets.Horizontal
//	                     or .Vertical, which every renderer resolves into any
//	                     side left at zero — so a side prop that only assigned
//	                     would be undone by the resolution and render the
//	                     shorthand's number. These are the cases settleHorizontal
//	                     and settleVertical exist for, and the axis assertion
//	                     below is what proves the settle happened out here and
//	                     not just in core's own unit tests.
//
// Both families are represented, because the props were duplicated onto
// Margin and a settle called on the wrong field compiles (see
// TestTheMarginAndPaddingFamiliesDoNotReachIntoEachOther).
func TestACallerStylePropOutranksAWidgetsOwnInsets(t *testing.T) {
	// The four sides, named, so a case can say which one it clears and which
	// one has to survive untouched.
	top := func(e core.EdgeInsets) int { return e.Top }
	bottom := func(e core.EdgeInsets) int { return e.Bottom }
	left := func(e core.EdgeInsets) int { return e.Left }
	right := func(e core.EdgeInsets) int { return e.Right }
	horizontal := func(e core.EdgeInsets) int { return e.Horizontal }
	vertical := func(e core.EdgeInsets) int { return e.Vertical }

	padding := func(n *core.Node) core.EdgeInsets { return n.Style.Padding }
	margin := func(n *core.Node) core.EdgeInsets { return n.Style.Margin }

	cases := []struct {
		name string
		// The same widget twice: without a caller Style, and with one whose
		// single prop clears one side.
		bare, styled core.View
		read         func(*core.Node) core.EdgeInsets
		// cleared is the side the caller zeroed; kept is a side on the same
		// axis that must come through unchanged; axis is the shorthand field
		// for that axis, which must end at zero or the renderers resolve the
		// cleared side straight back to it.
		clearedName, keptName string
		cleared, kept, axis   func(core.EdgeInsets) int
	}{
		{
			// The band's own PaddingHorizontal(Spacing.MD), which arrives as
			// a live shorthand. A caller indenting a nested band's label
			// wants the left gap gone and the right one kept.
			name: "GroupHeader padding, horizontal shorthand",
			bare: GroupHeader{Group: Group{Key: "a", Label: "A", Count: 1}},
			styled: GroupHeader{Group: Group{Key: "a", Label: "A", Count: 1},
				Style: []core.StyleProp{core.PaddingLeft(0)}},
			read: padding, clearedName: "Left", keptName: "Right",
			cleared: left, kept: right, axis: horizontal,
		},
		{
			// The same widget's other axis, from PaddingVertical(Spacing.XS).
			name: "GroupHeader padding, vertical shorthand",
			bare: GroupHeader{Group: Group{Key: "a", Label: "A", Count: 1}},
			styled: GroupHeader{Group: Group{Key: "a", Label: "A", Count: 1},
				Style: []core.StyleProp{core.PaddingTop(0)}},
			read: padding, clearedName: "Top", keptName: "Bottom",
			cleared: top, kept: bottom, axis: vertical,
		},
		{
			// SearchField's PaddingVertical(Spacing.XS) over the theme's Row
			// base, so the shorthand and the explicit sides are both live.
			name:   "SearchField padding, vertical shorthand over theme sides",
			bare:   SearchField{Value: "x"},
			styled: SearchField{Value: "x", Style: []core.StyleProp{core.PaddingBottom(0)}},
			read:   padding, clearedName: "Bottom", keptName: "Top",
			cleared: bottom, kept: top, axis: vertical,
		},
		{
			// The margin family, and the widget whose whole Inset field is
			// one MarginHorizontal. A rule inset on both ends that a caller
			// wants flush to the left edge only.
			name:   "Separator margin, horizontal shorthand",
			bare:   Separator{Inset: 16},
			styled: Separator{Inset: 16, Style: []core.StyleProp{core.MarginLeft(0)}},
			read:   margin, clearedName: "Left", keptName: "Right",
			cleared: left, kept: right, axis: horizontal,
		},
		{
			// Both families on one widget, from the theme's Card base, and
			// the explicit-sides shape rather than a shorthand. Card's margin
			// is the trap margin_sides.go was written about: a caller who
			// reached for UseStyle here would clear the other three sides
			// without seeing it.
			name:   "Card margin, explicit sides",
			bare:   Card{},
			styled: Card{Style: []core.StyleProp{core.MarginTop(0)}},
			read:   margin, clearedName: "Top", keptName: "Bottom",
			cleared: top, kept: bottom, axis: vertical,
		},
		{
			name:   "ListRow padding, explicit sides from the theme Row base",
			bare:   ListRow{Title: "t"},
			styled: ListRow{Title: "t", Style: []core.StyleProp{core.PaddingLeft(0)}},
			read:   padding, clearedName: "Left", keptName: "Right",
			cleared: left, kept: right, axis: horizontal,
		},
	}

	render := func(v core.View) *core.Node {
		ctx := core.NewContext()
		ctx.BeginRenderPass()
		return v.Render(ctx)
	}

	for _, c := range cases {
		bare := c.read(render(c.bare))
		styled := c.read(render(c.styled))

		// The guard. An entry whose widget has stopped setting this side
		// proves nothing — clearing a zero is a no-op that passes — and a
		// theme edit or a widget rewrite is exactly how that happens.
		if c.cleared(bare) == 0 {
			t.Errorf("%s: the widget's own %s is already 0 (%+v) — this case no longer "+
				"has a default to reach past, so pick another side or another widget",
				c.name, c.clearedName, bare)
			continue
		}
		if c.kept(bare) == 0 {
			t.Errorf("%s: the widget's own %s is 0 (%+v) — this case cannot show that "+
				"the prop touched only its own side", c.name, c.keptName, bare)
			continue
		}

		if got := c.cleared(styled); got != 0 {
			t.Errorf("%s: caller's side prop left %s at %d (%+v), want 0 — the widget "+
				"is applying an inset default after its caller's Style",
				c.name, c.clearedName, got, styled)
		}
		if got, want := c.kept(styled), c.kept(bare); got != want {
			t.Errorf("%s: %s moved from %d to %d — the side prop reached its own axis's "+
				"other side", c.name, c.keptName, want, got)
		}
		// The settle, observed through a real widget. A zero side sitting
		// beside a non-zero shorthand is resolved back to the shorthand by
		// htmlout.EdgeCSS, edgeToCSS, and both parseEdges — so this is the
		// difference between the prop working and the prop appearing to work.
		if got := c.axis(styled); got != 0 {
			t.Errorf("%s: the axis shorthand is still %d beside a cleared %s (%+v) — "+
				"every renderer resolves the zero side back to it", c.name, got,
				c.clearedName, styled)
		}
	}
}
