package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// Windows the rules are exercised against, each named for the device shape it
// stands in for. Sizes are the Android emulator's foldable profiles in dp.
var (
	phoneWindow = core.Window{Width: 411, Height: 891, Received: true}

	// A 7.6" book-style foldable opened flat: one continuous panel, so the
	// fold is present and not separating.
	unfoldedFlat = core.Window{Width: 841, Height: 673, Received: true, HasFold: true,
		Fold: core.Fold{State: core.FoldFlat, Orientation: core.FoldVertical,
			Bounds: core.WindowRect{X: 420.5, Height: 673}}}

	// The same device half-opened like a book.
	bookPosture = core.Window{Width: 841, Height: 673, Received: true, HasFold: true,
		Fold: core.Fold{State: core.FoldHalfOpened, Orientation: core.FoldVertical, Separating: true,
			Bounds: core.WindowRect{X: 420.5, Height: 673}}}

	// Rotated and half-opened on a table.
	tabletop = core.Window{Width: 673, Height: 841, Received: true, HasFold: true,
		Fold: core.Fold{State: core.FoldHalfOpened, Orientation: core.FoldHorizontal, Separating: true,
			Bounds: core.WindowRect{Y: 420.5, Width: 673}}}

	// A dual-screen device: a flat seam that separates and hides 28dp.
	dualScreen = core.Window{Width: 748, Height: 720, Received: true, HasFold: true,
		Fold: core.Fold{State: core.FoldFlat, Orientation: core.FoldVertical, Separating: true, Occluding: true,
			Bounds: core.WindowRect{X: 360, Width: 28, Height: 720}}}
)

func TestTwoPaneResolvesEachDeviceShape(t *testing.T) {
	for _, c := range []struct {
		name string
		pane TwoPane
		win  core.Window
		want twoPaneLayout
	}{
		{"phone stacks by default", TwoPane{}, phoneWindow, twoPaneLayout{arrange: arrangeStack}},
		{"phone shows first", TwoPane{Compact: TwoPaneFirst}, phoneWindow, twoPaneLayout{arrange: arrangeFirstOnly}},
		{"phone shows second", TwoPane{Compact: TwoPaneSecond}, phoneWindow, twoPaneLayout{arrange: arrangeSecondOnly}},
		{"unknown window is a phone", TwoPane{Compact: TwoPaneFirst}, core.Window{}, twoPaneLayout{arrange: arrangeFirstOnly}},

		// Flat and continuous: nothing to avoid, so the width decides.
		{"unfolded flat splits by ratio", TwoPane{}, unfoldedFlat, twoPaneLayout{arrange: arrangeRatioRow}},
		{"expanded threshold keeps medium single", TwoPane{SplitAt: core.SizeExpanded, Compact: TwoPaneFirst},
			core.Window{Width: 700, Height: 900}, twoPaneLayout{arrange: arrangeFirstOnly}},

		{"book splits at the hinge", TwoPane{}, bookPosture, twoPaneLayout{arrange: arrangeHingeRow, lead: 420.5}},
		// The fold outranks SplitAt: a hinge is on the glass whatever the
		// width threshold says.
		{"book ignores SplitAt", TwoPane{SplitAt: core.SizeExpanded}, bookPosture, twoPaneLayout{arrange: arrangeHingeRow, lead: 420.5}},
		{"tabletop splits above and below", TwoPane{}, tabletop, twoPaneLayout{arrange: arrangeHingeColumn, lead: 420.5}},
		{"origin shifts the lead", TwoPane{Origin: core.WindowRect{Y: 120.5}}, tabletop, twoPaneLayout{arrange: arrangeHingeColumn, lead: 300}},
		// Inside a scroll the horizontal hinge is not used; 673dp is medium.
		{"scrolling pane ignores tabletop", TwoPane{IgnoreHorizontalFold: true}, tabletop, twoPaneLayout{arrange: arrangeRatioRow}},
		{"occluding seam leaves a gap", TwoPane{}, dualScreen, twoPaneLayout{arrange: arrangeHingeRow, lead: 360, seam: 28}},

		// A hinge the pane does not contain falls through to the width rules
		// rather than drawing a First with no room.
		// (673dp wide is medium, so the width rule splits it.)
		{"hinge behind the origin", TwoPane{Origin: core.WindowRect{Y: 500}}, tabletop, twoPaneLayout{arrange: arrangeRatioRow}},
		{"hinge past the window", TwoPane{}, core.Window{Width: 841, Height: 673, HasFold: true,
			Fold: core.Fold{State: core.FoldHalfOpened, Orientation: core.FoldVertical, Separating: true,
				Bounds: core.WindowRect{X: 900}}}, twoPaneLayout{arrange: arrangeRatioRow}},
	} {
		if got := c.pane.resolve(c.win); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}

// What core.SafeInsets is for, end to end: the status bar's height is the
// Origin a pane below it needs, and reading it off the window is the whole
// of the arithmetic a caller has to do.
//
//	window top ──────────────  y = 0     ← the fold's bounds are measured here
//	  status bar   24dp             = win.Insets.Top
//	pane top   ──────────────  y = 24    ← Origin
//	  ⋮
//	hinge      ──────────────  y = 420.5 ← Fold.Bounds.Y, still in window space
//
// lead = 420.5 − 24: First gets 396.5dp and the crease falls exactly on the
// boundary between the panes rather than 24dp into Second.
func TestTwoPaneTakesItsOriginFromTheWindowInsets(t *testing.T) {
	win := tabletop
	win.Insets = core.SafeInsets{Top: 24, Bottom: 48}

	pane := TwoPane{Origin: core.WindowRect{Y: win.Insets.Top}}
	want := twoPaneLayout{arrange: arrangeHingeColumn, lead: tabletop.Fold.Bounds.Y - 24}
	if got := pane.resolve(win); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}

	// And a pane that forgets its origin is off by exactly the bar, which is
	// the bug the record exists to let a caller avoid.
	if got := (TwoPane{}).resolve(win); got.lead != tabletop.Fold.Bounds.Y {
		t.Errorf("no Origin gave lead %v, want the raw hinge position %v",
			got.lead, tabletop.Fold.Bounds.Y)
	}
}

func TestTwoPaneRatioClamps(t *testing.T) {
	for in, want := range map[float64]float64{0: 0.5, -1: 0.5, 1: 0.5, 2: 0.5, 0.4: 0.4} {
		if got := (TwoPane{Ratio: in}).ratio(); got != want {
			t.Errorf("Ratio %v: got %v, want %v", in, got, want)
		}
	}
}

// The rendered book layout: a Row whose First is pinned to the hinge and whose
// Second takes the remainder.
func TestTwoPaneRendersAHingeRow(t *testing.T) {
	core.ReceiveWindow(dualScreen)
	ctx, n := renderDebug(t, TwoPane{First: core.Text("list"), Second: core.Text("detail")})
	defer ctx.Close()

	// A Row node, not a Column with a row direction: the natives read the
	// axis from the node type.
	if n.Type != "Row" {
		t.Fatalf("container = %q, want Row", n.Type)
	}
	if len(n.Children) != 3 {
		t.Fatalf("want First, seam, Second; got %d children", len(n.Children))
	}
	first, seam, second := n.Children[0], n.Children[1], n.Children[2]
	if shrink, set := first.Style.ShrinkFactor(); first.Style.Width != "360.00px" || !set || shrink != 0 || findText(first, "list") == nil {
		t.Errorf("first: width %q shrink %v (set %v)", first.Style.Width, shrink, set)
	}
	if seam.Style.Width != "28.00px" || !seam.Style.AccessibilityHidden {
		t.Errorf("seam: width %q hidden %v", seam.Style.Width, seam.Style.AccessibilityHidden)
	}
	if second.Style.FlexGrow != 1 || findText(second, "detail") == nil {
		t.Errorf("second: grow %v", second.Style.FlexGrow)
	}
}

func TestTwoPaneRendersTabletopAsAColumn(t *testing.T) {
	core.ReceiveWindow(tabletop)
	ctx, n := renderDebug(t, TwoPane{First: core.Text("video"), Second: core.Text("controls")})
	defer ctx.Close()

	if n.Type != "Column" || len(n.Children) != 2 {
		t.Fatalf("container %q, %d children; want a two-child Column", n.Type, len(n.Children))
	}
	if h := n.Children[0].Style.Height; h != "420.50px" {
		t.Errorf("first height = %q, want the hinge's Y", h)
	}
}

// Compact with one pane chosen renders only that pane.
func TestTwoPaneCompactRendersOnePane(t *testing.T) {
	core.ReceiveWindow(phoneWindow)
	ctx, n := renderDebug(t, TwoPane{First: core.Text("list"), Second: core.Text("detail"), Compact: TwoPaneSecond})
	defer ctx.Close()
	if findText(n, "list") != nil || findText(n, "detail") == nil {
		t.Error("compact TwoPaneSecond should render the detail alone")
	}
}

// A ratio split is a Row too, with the grow factors carrying the ratio.
func TestTwoPaneRendersARatioRow(t *testing.T) {
	core.ReceiveWindow(unfoldedFlat)
	ctx, n := renderDebug(t, TwoPane{First: core.Text("list"), Second: core.Text("detail"), Ratio: 0.4})
	defer ctx.Close()
	if n.Type != "Row" || len(n.Children) != 2 {
		t.Fatalf("container %q with %d children, want a two-child Row", n.Type, len(n.Children))
	}
	if g := n.Children[0].Style.FlexGrow; g != 0.4 {
		t.Errorf("first grow = %v, want 0.4", g)
	}
}
