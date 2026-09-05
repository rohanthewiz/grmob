package components

import (
	"fmt"

	"github.com/rohanthewiz/grmob/core"
)

// Skeleton is the grey placeholder that holds a screen's shape while its
// content loads: one bar, or a stack of them standing in for a paragraph.
//
//	components.Skeleton{}                            // one line
//	components.Skeleton{Lines: 3}                    // a paragraph, last line short
//	components.Skeleton{Width: "44px", Height: 44, Radius: 999}   // an avatar
//
// # Skeleton or EmptyState
//
// They answer different questions. A skeleton says "content is coming and it
// will look roughly like this", which is worth saying when the layout is
// known and stable — a feed of rows, a profile header. EmptyState says "there
// is nothing here yet, and here is why", which is what a screen with no
// predictable shape, or a wait long enough to need explaining, wants instead.
// A list of three placeholder rows reads better than "Loading…"; a whole
// screen of grey bars reads worse.
//
// # No shimmer, and why that is not a shortcut
//
// The moving highlight every design system puts on a skeleton is a repeating
// keyframe animation. core.Transition is not one — it animates a property
// from one declared value to another, driven natively, and there is no state
// change here to drive. The alternative is looping it from Go with
// hooks.UseInterval, which would push a render pass and a patch across the
// bridge for every frame of a decoration, on every placeholder on screen. That
// is the one thing the framework's "declare in Go, animate natively" model
// exists to avoid, so the bars are static until a repeating animation is a
// core primitive.
//
// # The color is the Border role, not Surface
//
// Surface is the palette's obvious "muted fill", and it is the wrong answer
// for the same reason Separator gives: it is the fill a *panel* uses, so a
// Surface bar inside a card disappears. Border is the neutral that is visible
// against both Background and Surface, which is where placeholders sit. It is
// nominally a stroke role and this is a fill — the palette carries no third
// neutral, and being visible beats being nominally correct.
type Skeleton struct {
	// Lines is how many bars to stack. Zero means one.
	Lines int

	// Height is a bar's height in points. Zero takes the theme's body font
	// size, so a text placeholder is about as tall as the text it stands in
	// for and scales with the theme.
	Height float64

	// Width is every bar's width, as a CSS-ish length ("100%", "180px").
	// Empty is "100%".
	Width string

	// LastLineWidth shortens the final bar, which is what makes a stack read
	// as a paragraph rather than as a table. Empty is "60%". It applies only
	// when Lines is 2 or more: on a single bar the last line is the only
	// line, and silently rendering it at 60% would make the simplest call
	// surprising.
	LastLineWidth string

	// Gap is the space between bars. Zero takes the theme's SM step.
	Gap float64

	// Radius is the bar's corner radius. Zero is 4 — enough to read as a
	// placeholder rather than as a rule. Set it to half the height (or just
	// to 999, which clamps) for a pill, or with a square Width/Height for the
	// circle an avatar placeholder wants.
	Radius float64

	// Color overrides the bar fill. Empty takes the theme's Border role; see
	// the type comment for why that and not Surface.
	Color string

	// AccessibilityLabel names the whole block for assistive technology.
	// Empty is "Loading".
	//
	// The individual bars are always hidden — a reader walking six unlabeled
	// boxes is worse than silence — and the label goes on the container
	// instead. Be aware of the limit: a labelled container with no role is
	// announced by the natives but not reliably by the DOM targets, so a
	// screen where the wait genuinely needs announcing should say so in text
	// (EmptyState{Title: "Loading sermons…"}) rather than rely on this. For
	// no announcement at all, pass core.AccessibilityHidden() in Style.
	AccessibilityLabel string

	// Style is applied to the container after the widget's own defaults.
	// Per-bar styling is not exposed: a skeleton whose bars differ is a
	// composition of Skeletons, not one Skeleton with more knobs.
	Style []core.StyleProp
}

func (s Skeleton) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	lines := s.Lines
	if lines < 1 {
		lines = 1
	}
	height := s.Height
	if height == 0 {
		height = t.Typography.Body.FontSize
	}
	width := s.Width
	if width == "" {
		width = "100%"
	}
	lastWidth := s.LastLineWidth
	if lastWidth == "" {
		lastWidth = "60%"
	}
	gap := s.Gap
	if gap == 0 {
		gap = float64(t.Spacing.SM)
	}
	radius := s.Radius
	if radius == 0 {
		radius = 4
	}
	color := s.Color
	if color == "" {
		color = t.Colors.BorderColor()
	}
	label := s.AccessibilityLabel
	if label == "" {
		label = "Loading"
	}

	items := make([]core.PropsAndChildren, 0, lines+len(s.Style)+2)
	items = append(items,
		core.Gap(gap),
		core.AccessibilityLabel(label),
	)
	for _, sp := range s.Style {
		items = append(items, sp)
	}

	for i := 0; i < lines; i++ {
		w := width
		if lines > 1 && i == lines-1 {
			w = lastWidth
		}
		items = append(items, core.Box(
			core.Width(w),
			core.Height(fmt.Sprintf("%gpx", height)),
			core.BackgroundColor(color),
			core.BorderRadius(radius),
			core.AccessibilityHidden(),
		))
	}

	// Box, not Column: the container is pure structure and must contribute no
	// padding of its own, or a one-line skeleton would be inset from the row
	// it is standing in for. Same reason Separator and AppBar reach for it.
	return core.Box(items...).Render(ctx)
}
