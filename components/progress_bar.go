package components

import (
	"fmt"
	"math"

	"github.com/rohanthewiz/grmob/core"
)

// ProgressBar is the determinate track-and-fill bar: an upload, a download, a
// quota, a multi-step form's position.
//
//	components.ProgressBar{Value: 0.45, AccessibilityLabel: "Upload"}
//
// # Why the fill is a percentage width and not a pair of flex weights
//
// The obvious construction is two boxes weighted FlexGrow(v) and
// FlexGrow(1-v), letting the flex algorithm split the track. That is exact on
// Android, where FlexGrow maps onto Compose's Modifier.weight — and silently
// wrong on iOS, where it maps onto frame(maxWidth: .infinity). SwiftUI stacks
// have no weight, so two growers split the free space *equally regardless of
// their values*: every bar would render at 50% on iOS, at every value, with
// nothing in the tree to suggest a bug.
//
// A percentage width is proportional on all three targets instead:
//
//	Android   fillMaxWidth(fraction)  — exact
//	HTML      width:<pct>%            — exact
//	iOS       containerRelativeFrame  — proportional, see the caveat below
//
// The iOS caveat is that containerRelativeFrame measures against the nearest
// *container* (the scroll view or root), not the immediate parent. A bar that
// spans its container — the common case, a full-width bar in a screen column
// — is therefore exact; one inset inside a narrow card reads wider than it
// should. That is an over-long fill in an uncommon layout, against a
// permanently-half-full bar everywhere. Proportional weights on iOS need a
// custom SwiftUI Layout, and when that lands this widget can move to flex.
//
// # The fill is always rendered
//
// Even at Value 0, where it is zero pixels wide. Keeping the child count fixed
// means advancing progress is a style patch on one node rather than an insert
// or a remove, so the reconciler emits an update-style op per frame instead of
// restructuring the tree — which is also what lets a Transition on the fill
// animate the bar smoothly.
type ProgressBar struct {
	// Value is the completed fraction, 0 to 1. Values outside that range are
	// clamped rather than rejected: a bar fed a ratio from live counters
	// should pin at full and keep rendering, not draw outside its track.
	// NaN is treated as 0.
	Value float64

	// Thickness is the track height in px; 0 means 6.
	Thickness float64

	// Color is the fill; empty uses the theme's Primary. TrackColor is the
	// groove behind it; empty uses the theme's Surface.
	Color      string
	TrackColor string

	// Style is applied to the track after the defaults, so the bar's width,
	// margins and corner are all overridable.
	Style []core.StyleProp

	// AccessibilityLabel names what is progressing ("Upload"). The percentage
	// is appended by the widget, since no renderer has a progress semantic to
	// carry the value natively — the announcement has to be in the label or
	// it does not exist. When empty the bar is hidden from assistive tech: an
	// unlabeled bar announces a bare number with nothing to attach it to, and
	// a bar beside its own "Uploading, 45%" caption should stay silent.
	AccessibilityLabel string
}

func (p ProgressBar) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	value := p.Value
	if math.IsNaN(value) {
		value = 0
	}
	value = math.Min(math.Max(value, 0), 1)

	thickness := p.Thickness
	if thickness == 0 {
		thickness = 6
	}
	height := fmt.Sprintf("%gpx", thickness)
	// Fully rounded ends: at radius = half the thickness the track's corners
	// are semicircles, the standard pill groove.
	radius := thickness / 2

	fill := p.Color
	if fill == "" {
		fill = t.Colors.Primary
	}
	track := p.TrackColor
	if track == "" {
		track = t.Colors.Surface
	}

	// Rounded to hundredths of a percent before formatting, because %g on a
	// raw product prints binary noise: 0.333*100 is 33.300000000000004 in
	// float64, and that string would travel to every renderer and into the
	// exported CSS. Two decimals is well under a pixel on any real bar
	// (0.01% of a 400px track is 0.04px), and %g then drops the trailing
	// zeros so whole values stay short — "45%", not "45.00%".
	width := fmt.Sprintf("%g%%", math.Round(value*10000)/100)

	items := make([]core.PropsAndChildren, 0, len(p.Style)+5)
	// Row, not Box: the fill has to sit at the leading edge and be laid out
	// along the main axis, which is a Row's job. Padding(0) drops the theme
	// Row's screen padding — a 6px-tall groove cannot carry 8px of inset.
	items = append(items,
		core.Height(height),
		core.Padding(0),
		core.BackgroundColor(track),
		core.BorderRadius(radius),
	)
	if label := p.AccessibilityLabel; label != "" {
		// Rounded for the announcement only; the fill keeps the exact value.
		items = append(items, core.AccessibilityLabel(
			fmt.Sprintf("%s, %d percent", label, int(math.Round(value*100)))))
	} else {
		items = append(items, core.AccessibilityHidden())
	}
	for _, sp := range p.Style {
		items = append(items, sp)
	}

	// The fill carries its own Height because a Compose Box and a SwiftUI
	// ZStack both size to their content, and this one has none: without an
	// explicit height it would be a zero-pixel line inside a correct track.
	items = append(items, core.Box(
		core.Width(width),
		core.Height(height),
		core.BackgroundColor(fill),
		core.BorderRadius(radius),
	))

	return core.Row(items...).Render(ctx)
}
