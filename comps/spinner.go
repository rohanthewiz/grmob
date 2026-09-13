package comps

import (
	"fmt"

	"github.com/rohanthewiz/grmob/core"
)

// Spinner is the "something is happening, shape unknown" indicator: a ring
// with a dot orbiting it.
//
//	comps.Spinner{Hidden: !loading.Get()}
//	comps.Spinner{Size: comps.SpinnerLarge, Label: "Uploading"}
//
// It completes the loading trio. Skeleton is for content whose layout is known
// and whose data is not; ProgressBar is for work that knows how far it has
// got; Spinner is for everything else.
//
// # The platform turns it
//
// The ring carries core.Spin(spinPeriodMs), so each renderer's own frame clock
// drives the rotation: CSS keyframes on the web, a graphics layer on Compose, a
// TimelineView on SwiftUI. Go declares the spin once, in the style of the first
// pass, and sends nothing afterwards.
//
// It used to be stepped from Go instead — a state slot holding the angle and
// hooks.UseIntervalWhile adding 30 degrees every 80ms — because no renderer
// could draw a loop on its own. That cost a render pass of the whole app per
// step (twelve a second) while visible, drew twelve discrete positions rather
// than a turn, and gave the widget two hook slots with the ordering rule that
// comes with them. core.Spin removed all three.
//
// # No hooks, so it is conditional-safe
//
// Spinner takes no hook slot, so core.If(loading, comps.Spinner{}) is as
// correct as Hidden. Hidden remains the switch to prefer where the spinner has
// a fixed place in a layout: a hidden spinner is Display none, which keeps the
// tree the same shape both ways, so showing it is a style patch rather than an
// inserted subtree. Either way a hidden or absent spinner draws no frames on
// any target (see core.Spin, "What it costs").
//
//	┌ Box  role=status  label="Loading" ┐
//	│  ┌ Column  ring, Spin(1000) ────┐ │
//	│  │            ●                 │ │
//	│  │                              │ │
//	│  └──────────────────────────────┘ │
//	└───────────────────────────────────┘
//
// The dot is what makes the rotation visible. A ring of uniform colour turned
// about its centre draws the same pixels at every angle, and core has no
// per-side border colour to draw a gap in the ring with.
//
// # Accessibility
//
// The outer Box is RoleStatus with Label (default "Loading"), a polite live
// region announced when it appears. The ring beneath it is decoration and is
// hidden from assistive technology. A hidden spinner is display:none and so is
// not announced at all.
//
// No target slows or stops the spin for a reduce-motion setting, because core
// has no signal for it yet; see core.Spin, "Reduced motion".
//
// # Theme roles read
//
//	Ring       Colors.BorderColor()
//	Dot        Colors.Primary
//	Diameter   Spacing.MD / LG / XL for Small / Medium / Large
type Spinner struct {
	// Hidden removes the spinner from display, which also stops its frames.
	// Prefer it to leaving the widget out where the spinner has a fixed place
	// in the layout; see the type doc.
	Hidden bool

	// Size picks the diameter. The zero value is SpinnerMedium.
	Size SpinnerSize

	// Label is the status announcement. Empty uses "Loading".
	Label string

	// Style is applied to the outer Box after the widget's own props.
	Style []core.StyleProp
}

// SpinnerSize selects a Spinner's diameter from the theme's spacing scale.
type SpinnerSize string

const (
	// SpinnerMedium is Spacing.LG across; the zero value.
	SpinnerMedium SpinnerSize = ""
	// SpinnerSmall is Spacing.MD across, for inline use beside text.
	SpinnerSmall SpinnerSize = "small"
	// SpinnerLarge is Spacing.XL across, for a spinner that is the screen's
	// whole content.
	SpinnerLarge SpinnerSize = "large"
)

// spinPeriodMs is one revolution a second, the rate the stepped spinner had
// (30 degrees every 80ms is 360 degrees in 960ms), so the change of mechanism
// did not also change how busy the widget looks. Every size shares it: a larger
// ring at the same period moves its dot faster, which is how a platform
// spinner behaves too.
const spinPeriodMs = 1000

// diameter returns the ring's size in points for the theme.
func (s Spinner) diameter(t *core.Theme) float64 {
	switch s.Size {
	case SpinnerSmall:
		return float64(t.Spacing.MD)
	case SpinnerLarge:
		return float64(t.Spacing.XL)
	default:
		return float64(t.Spacing.LG)
	}
}

// Render builds the status box and the spinning ring.
func (s Spinner) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	d := s.diameter(t)
	dot := d / 4
	// Point sizes cross the wire as CSS-style lengths, the form Skeleton and
	// Avatar already pass to core.Width and core.Height.
	px := func(v float64) string { return fmt.Sprintf("%gpx", v) }

	ring := core.Column(
		core.Width(px(d)),
		core.Height(px(d)),
		core.Padding(0),
		core.Gap(0),
		core.BorderRadius(d/2),
		core.BorderWidth(2),
		core.BorderColor(t.Colors.BorderColor()),
		core.AlignItemsProp(core.AlignItemsCenter),
		// On the ring, not the status Box: the spin turns the box it is
		// declared on, and the outer Box also holds a caller's Style, which
		// could give it a size or a padding the dot would then orbit.
		core.Spin(spinPeriodMs),
		core.AccessibilityHidden(),
		core.Box(
			core.Width(px(dot)),
			core.Height(px(dot)),
			core.BorderRadius(dot/2),
			core.BackgroundColor(t.Colors.Primary),
		),
	)

	items := make([]core.PropsAndChildren, 0, len(s.Style)+5)
	items = append(items,
		core.AccessibilityRole(core.RoleStatus),
		core.AccessibilityLabel(orDefault(s.Label, "Loading")),
	)
	if s.Hidden {
		items = append(items, core.Display(core.DisplayNone))
	}
	for _, sp := range s.Style {
		items = append(items, sp)
	}
	items = append(items, ring)
	return core.Box(items...).Render(ctx)
}
