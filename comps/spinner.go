package comps

import (
	"fmt"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
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
// # It holds hooks, so render it unconditionally
//
// There is no looping animation any renderer draws on its own: core.Transition
// animates one change and stops, Style.Rotate is applied without interpolation
// on both natives, and Style.Animation is honoured only by the two web targets.
// So the spin is stepped from Go: one core.NewState for the angle and one
// hooks.UseIntervalWhile advancing it by stepDegrees every stepEvery.
//
// That makes this the third widget in the package with hook obligations,
// after Accordion and DatePicker, and the rules are theirs: render a Spinner in
// a stable position on every pass. To stop showing it, set Hidden rather than
// leaving it out of the tree; leaving it out moves the two hook slots and every
// hook after them.
//
// # What it costs, and why Hidden is the switch
//
// Each step is a state change, so a visible spinner costs a render pass per
// step (about twelve a second). Hidden both hides the node and pauses the
// interval; UseIntervalWhile drops a paused tick before it can request a
// render, so a hidden spinner costs nothing but a goroutine wake. That is the
// reason the hook exists: with plain UseInterval, a spinner that had ever been
// mounted would re-render the whole app on every tick for the life of the
// process. A looping transition prop on all four renderers would remove the
// stepping altogether and is recorded as a follow-up.
//
//	┌ Box  role=status  label="Loading" ┐
//	│  ┌ Column  ring, Rotate(angle) ┐  │
//	│  │            ●                │  │
//	│  │                             │  │
//	│  └─────────────────────────────┘  │
//	└───────────────────────────────────┘
//
// The dot is what makes the rotation visible. A ring of uniform colour turned
// about its centre draws the same pixels at every angle, and core has no
// per-side border colour to draw a gap in the ring with.
//
// # Accessibility
//
// The outer Box is RoleStatus with Label (default "Loading"), a polite live
// region announced when it appears. The ring beneath it changes style every
// step and is hidden from assistive technology, so the steps are never read.
// A hidden spinner is display:none and so is not announced at all.
//
// # Theme roles read
//
//	Ring       Colors.BorderColor()
//	Dot        Colors.Primary
//	Diameter   Spacing.MD / LG / XL for Small / Medium / Large
type Spinner struct {
	// Hidden removes the spinner from display and pauses its ticks. Use it
	// instead of conditionally rendering the widget; see the type doc.
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

// The step is 30 degrees every 80ms: one revolution a second in twelve visible
// positions. Fewer, larger steps would look like ticking; more, smaller ones
// would cost more render passes for motion the eye does not resolve better.
const (
	stepDegrees = 30
	stepEvery   = 80 * time.Millisecond
)

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

// Render builds the status box and the rotating ring.
func (s Spinner) Render(ctx *core.Context) *core.Node {
	// Both hooks run first and on every pass, Hidden or not: their slot
	// positions must not depend on a field the caller flips.
	angle := core.NewState(ctx, 0)
	hooks.UseIntervalWhile(ctx, !s.Hidden, func() {
		// Wrapped to [0, 360): with no Transition on the ring the two
		// spellings draw identically, and a bounded value keeps the wire
		// number short however long the spinner runs.
		angle.Set((angle.Get() + stepDegrees) % 360)
	}, stepEvery)

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
		core.Rotate(float64(angle.Get())),
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
