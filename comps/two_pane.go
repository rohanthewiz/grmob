package comps

import (
	"fmt"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

// TwoPane lays two views out side by side when the window has room for both,
// one after the other (or just one) when it does not, and — on a foldable —
// on either side of the hinge rather than across it.
//
//	comps.TwoPane{
//	    First:   inboxList,
//	    Second:  messageDetail,
//	    Compact: comps.TwoPaneFirst, // a phone shows the list; a tap navigates
//	}
//
// # The decision, in order
//
// The first rule that applies wins:
//
//  1. a separating fold, vertical     Row:    First | hinge | Second
//  2. a separating fold, horizontal   Column: First / hinge / Second
//  3. width class ≥ SplitAt           Row:    First | Gap | Second, by Ratio
//  4. otherwise (compact)             Compact decides: both stacked, or one
//
// A fold outranks width because it is a physical fact about the glass: an
// unfolded book-style device is medium or expanded *and* has a hinge down the
// middle, and a ratio split that happened to put a button on the crease is
// the bug foldable support exists to prevent. A horizontal fold is what the
// tabletop posture looks like — the natural layout there is content above
// the hinge and controls below, which is rule 2 with First above.
//
// Only a *separating* fold is laid out around. A flat, continuous panel (a
// Galaxy Z Fold opened flat) reports a fold the platform says content may
// cross, and falls through to rule 3 so an app is not split in half on a
// screen with nothing in the middle.
//
// # Lining up with the hinge
//
// Fold bounds are in window coordinates (see core/window.go) and a TwoPane
// lays out in its own, so the two agree only when the TwoPane starts where
// the window does. Origin is the pane's top-left corner in window
// coordinates, for the layouts where it does not: a TwoPane under an 64dp
// app bar in tabletop posture passes Origin{Y: statusBar + 64}. A TwoPane
// that fills the window along the split axis — the usual arrangement for a
// vertical hinge, where nothing sits to its left — needs none.
//
// First is sized to end exactly at the hinge (a fixed width or height) and
// Second grows to fill what remains, which means the split is exact on the
// First side and assumes the TwoPane reaches the window's far edge on the
// Second. A hinge the pane does not actually contain (First would be zero or
// negative, or the hinge sits past the window) falls through to the ratio
// rules rather than drawing a pane with no room.
//
// An occluding hinge (a dual-screen seam with real width) becomes a spacer of
// that width, so nothing is drawn under it. A non-occluding one is a line and
// adds nothing; each pane's own padding keeps its content off the crease.
//
// # One hook
//
// TwoPane calls hooks.UseWindow, so it re-renders itself when the device
// folds, unfolds or bends, with nothing for the caller to wire. That makes
// the rule Accordion and Snackbar document apply here too: render a TwoPane
// in a stable position on every pass rather than conditionally.
//
// # Filling
//
// The container grows (FlexGrow 1) and stretches its panes along the cross
// axis. A side-by-side split has to fill the height it is given to look like
// two panes rather than two cards, and a top/bottom split has to fill the
// height for the hinge arithmetic to mean anything, so filling is the default
// and Style can undo it.
type TwoPane struct {
	// First and Second are the two panes: list and detail, content and
	// controls, left page and right page. First is leading (left in a
	// left-to-right layout) or top.
	First, Second core.View

	// Ratio is the fraction of the width First takes in a ratio split (rule
	// 3). Zero means 0.5; a value outside (0, 1) is clamped to it. A list
	// beside a detail usually wants about 0.4.
	Ratio float64

	// Gap is the space between the panes in a ratio split and in a stacked
	// compact layout. Zero means the theme's MD spacing. It is not applied
	// around a hinge — see "Lining up with the hinge".
	Gap float64

	// SplitAt is the smallest width class that splits side by side without
	// a fold. Zero means core.SizeMedium, which is where an unfolded
	// book-style foldable and a portrait tablet land; core.SizeExpanded keeps
	// medium windows single-pane for content that needs width on both sides.
	SplitAt core.SizeClass

	// Compact is what a compact window with no separating fold shows. Zero is
	// TwoPaneStack.
	Compact TwoPaneCompact

	// Origin is the TwoPane's top-left in window coordinates, for aligning to
	// a hinge when the pane does not start at the window's corner. Only X and
	// Y are read.
	Origin core.WindowRect

	// IgnoreHorizontalFold skips rule 2, for a TwoPane inside a scrolling
	// column. Scrolling moves the pane up and down past a horizontal hinge,
	// so no fixed Origin.Y can say where the hinge falls in it, and a First
	// sized to a stale offset is worse than a ratio split. A vertical hinge
	// is unaffected: vertical scrolling does not move anything sideways.
	IgnoreHorizontalFold bool

	// Style is applied to the container after the widget's own props.
	Style []core.StyleProp
}

// TwoPaneCompact chooses what a compact window shows.
type TwoPaneCompact int

const (
	// TwoPaneStack shows both panes, First above Second. Right for content
	// and controls that both belong on screen.
	TwoPaneStack TwoPaneCompact = iota
	// TwoPaneFirst shows only First — the list in list–detail, where the
	// caller navigates to the detail on a phone.
	TwoPaneFirst
	// TwoPaneSecond shows only Second — the detail, once something is
	// selected.
	TwoPaneSecond
)

// twoPaneArrangement is the decision Render acts on, split out so the rules
// can be tested against a Window without rendering.
type twoPaneArrangement int

const (
	arrangeHingeRow twoPaneArrangement = iota
	arrangeHingeColumn
	arrangeRatioRow
	arrangeStack
	arrangeFirstOnly
	arrangeSecondOnly
)

// twoPaneLayout is a resolved arrangement plus the lengths it needs.
type twoPaneLayout struct {
	arrange twoPaneArrangement
	// lead is First's fixed width (hinge row) or height (hinge column).
	lead float64
	// seam is the occluded hinge width or height, 0 when it hides nothing.
	seam float64
}

// resolve applies the rules in the type comment to one window.
func (p TwoPane) resolve(win core.Window) twoPaneLayout {
	if f, ok := win.SeparatingFold(); ok {
		switch f.Orientation {
		case core.FoldVertical:
			lead := f.Bounds.X - p.Origin.X
			// The hinge must lie inside the window, past the pane's start:
			// otherwise First would have no width, or Second none.
			if lead > 0 && f.Bounds.X+f.Bounds.Width < win.Width {
				return twoPaneLayout{arrange: arrangeHingeRow, lead: lead, seam: occluded(f, f.Bounds.Width)}
			}
		case core.FoldHorizontal:
			if p.IgnoreHorizontalFold {
				break
			}
			lead := f.Bounds.Y - p.Origin.Y
			if lead > 0 && f.Bounds.Y+f.Bounds.Height < win.Height {
				return twoPaneLayout{arrange: arrangeHingeColumn, lead: lead, seam: occluded(f, f.Bounds.Height)}
			}
		}
	}

	if sizeRank(win.WidthClass()) >= sizeRank(p.splitAt()) {
		return twoPaneLayout{arrange: arrangeRatioRow}
	}
	switch p.Compact {
	case TwoPaneFirst:
		return twoPaneLayout{arrange: arrangeFirstOnly}
	case TwoPaneSecond:
		return twoPaneLayout{arrange: arrangeSecondOnly}
	}
	return twoPaneLayout{arrange: arrangeStack}
}

// occluded is the seam to leave empty: the hinge's thickness when it hides
// pixels, nothing when it is a line on a continuous panel.
func occluded(f core.Fold, thickness float64) float64 {
	if f.Occluding && thickness > 0 {
		return thickness
	}
	return 0
}

func (p TwoPane) splitAt() core.SizeClass {
	if p.SplitAt == "" {
		return core.SizeMedium
	}
	return p.SplitAt
}

// sizeRank orders size classes so SplitAt can be compared with ≥. An unknown
// class ranks above expanded, which means "never split on width alone" — the
// conservative reading of a value this version does not know.
func sizeRank(c core.SizeClass) int {
	switch c {
	case core.SizeCompact:
		return 0
	case core.SizeMedium:
		return 1
	case core.SizeExpanded:
		return 2
	}
	return 3
}

func (p TwoPane) ratio() float64 {
	if p.Ratio <= 0 || p.Ratio >= 1 {
		return 0.5
	}
	return p.Ratio
}

// Render reads the window, resolves the arrangement and builds it.
func (p TwoPane) Render(ctx *core.Context) *core.Node {
	// Unconditional and first: the hook slot must exist on every pass.
	win := hooks.UseWindow(ctx)
	t := ctx.Theme()
	gap := p.Gap
	if gap == 0 {
		gap = float64(t.Spacing.MD)
	}

	lay := p.resolve(win)

	items := make([]core.PropsAndChildren, 0, len(p.Style)+8)
	items = append(items,
		core.FlexGrow(1),
		core.AlignItemsProp(core.AlignItemsStretch),
		// Padding 0: a TwoPane is a layout, not a surface. A theme's
		// container padding would otherwise shift First off the hinge by
		// exactly that much.
		core.Padding(0),
	)

	// grow is the "take what remains" recipe the charts and StatTile use:
	// a zero basis so the grow factors alone divide the room.
	grow := func(v core.View, factor float64) core.View {
		return core.Column(core.FlexGrow(factor), core.FlexBasis("0"), core.Padding(0), v)
	}

	// horizontal picks the container's node type. A Row, not a Column with
	// core.Horizontal(): that prop is the scrolling strip's, and the natives
	// choose a stack's axis from the node type, so a Column told to be
	// horizontal is a row in a browser and a column on a phone.
	horizontal := false
	var children []core.PropsAndChildren
	switch lay.arrange {
	case arrangeHingeRow, arrangeHingeColumn:
		row := lay.arrange == arrangeHingeRow
		horizontal = row
		lead := core.Width(paneLength(lay.lead))
		if !row {
			lead = core.Height(paneLength(lay.lead))
		}
		// FlexShrink 0: First's length *is* the hinge position, and a
		// Second with wide content must not squeeze it off the crease.
		children = append(children, core.Column(lead, core.FlexShrink(0), core.Padding(0), p.First))
		if lay.seam > 0 {
			seam := core.Width(paneLength(lay.seam))
			if !row {
				seam = core.Height(paneLength(lay.seam))
			}
			children = append(children, core.Box(seam, core.FlexShrink(0), core.Padding(0), core.AccessibilityHidden()))
		}
		children = append(children, grow(p.Second, 1))

	case arrangeRatioRow:
		r := p.ratio()
		horizontal = true
		items = append(items, core.Gap(gap))
		children = append(children, grow(p.First, r), grow(p.Second, 1-r))

	case arrangeStack:
		items = append(items, core.Gap(gap))
		children = append(children, p.First, p.Second)

	case arrangeFirstOnly:
		children = append(children, grow(p.First, 1))

	case arrangeSecondOnly:
		children = append(children, grow(p.Second, 1))
	}

	for _, sp := range p.Style {
		items = append(items, sp)
	}
	items = append(items, children...)
	if horizontal {
		return core.Row(items...).Render(ctx)
	}
	return core.Column(items...).Render(ctx)
}

// paneLength formats a layout length as the dimension string Width and Height
// take. Unlike clock.go's px it rounds to a hundredth: Android's dp conversion
// yields values like 420.57142857, and the extra digits only make patches
// noisier.
func paneLength(v float64) string {
	return fmt.Sprintf("%.2fpx", v)
}
