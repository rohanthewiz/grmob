package comps

import (
	"fmt"
	"strconv"

	"github.com/rohanthewiz/grmob/core"
)

// StepIndicator is the "step 2 of 4" header of a multi-screen flow: numbered
// circles joined by short rules, done steps ticked, the current step filled.
//
//	comps.StepIndicator{
//	    Steps:   []string{"Account", "Address", "Payment", "Review"},
//	    Current: step.Get(),
//	    OnTap:   step.Set, // done steps only
//	}
//
//	┌ Scroll Horizontal  role=navigation  "Step 2 of 4: Address" ───────────┐
//	│ (✓) Account ── (2) Address ── (3) Payment ── (4) Review               │
//	│  done, button    current        upcoming       upcoming              │
//	└───────────────────────────────────────────────────────────────────────┘
//
// # The overflow decision: scroll, not collapse
//
// Five or six labelled steps do not fit a phone's width. The plan offered two
// answers: collapse to "2 / 4" past a threshold, or let the strip scroll
// sideways. Collapsing needs a width to compare against, which Go does not
// have, so any threshold would be a guess that is wrong on some screen. The
// strip is a core.Scroll with core.Horizontal, which every target already
// scrolls, and a flow short enough to fit simply does not move.
//
// The Scroll is the strip, as it is in ChipStrip, and nothing wraps it. Its
// first version sat inside a Row because the web host pages sized every
// Scroll as a screen's vertical viewport, which collapsed a sideways strip to
// 0px inside a column. That rule now leaves horizontal Scrolls their content
// height (see the Scroll rules in wasm/index.html), so the wrapper went away.
//
// # Which steps can be tapped
//
// Only done steps, and only when OnTap is set. Going back to fix an address is
// what a flow allows; jumping ahead past a step that has not been validated is
// not, and a widget that made upcoming steps tappable would push every caller
// to write that guard. The current step is not tappable either: tapping it
// would do nothing.
//
// # Accessibility
//
// The strip is RoleNavigation when OnTap is set, because done steps then are
// destinations, and RoleGroup when it is not, because a picture of progress
// is not navigation. Either way its name states the position: "Step 2 of 4:
// Address", prefixed by Label when one is given. Each step is named
// ("Step 1: Account, done", "Step 2: Address, current", "Step 3: Payment") and
// a tappable one is RoleButton. The circle, its glyph and the rules are drawn
// for sighted users and hidden from assistive technology, so a step is read
// once.
//
// The ", done" and ", current" suffixes are English, the fallback ListRow and
// BottomBar already take; there is no aria-current in core's vocabulary.
//
// # Theme roles read
//
//	Done circle        Colors.SuccessColor(), ink chosen by Variant.Ink
//	Current circle     Colors.Primary, ink chosen by Variant.Ink
//	Upcoming circle    Colors.ControlBorder ring (BorderColor() if unset),
//	                   Colors.TextSecondary number
//	Rule               Colors.SuccessColor() after a done step, else
//	                   Colors.BorderColor()
//	Labels             Typography.Body; the current step bold, upcoming steps
//	                   in Colors.TextSecondary
//	Glyphs             Typography.Caption, bold
//	Gap                Spacing.XS
type StepIndicator struct {
	// Steps are the step names, in order.
	Steps []string

	// Current is the zero-based index of the step in progress. It is clamped
	// into the Steps range, so a flow's "finished" index draws every step but
	// the last as done.
	Current int

	// OnTap receives the index of a tapped done step. Nil makes the strip a
	// display of progress with no targets.
	OnTap func(int)

	// Label prefixes the strip's accessible name, for a screen with more than
	// one flow ("Checkout, step 2 of 4: Address").
	Label string

	// Style is applied to the strip after the widget's own props.
	Style []core.StyleProp
}

// stepCircle is the diameter of a step's circle, in points.
const stepCircle = 24

// stepState is where one step sits relative to Current.
type stepState int

const (
	stepDone stepState = iota
	stepCurrent
	stepUpcoming
)

// Render builds the horizontal Scroll of steps and rules.
func (s StepIndicator) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	n := len(s.Steps)
	// Clamped once so the drawn states, the strip's name and the tap guard
	// agree on one index.
	cur := 0
	if n > 0 {
		cur = min(max(s.Current, 0), n-1)
	}

	role := core.RoleGroup
	if s.OnTap != nil {
		role = core.RoleNavigation
	}

	items := make([]core.PropsAndChildren, 0, len(s.Style)+2*n+6)
	items = append(items,
		core.Horizontal(),
		core.Gap(float64(t.Spacing.XS)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.AccessibilityRole(role),
	)
	if n > 0 {
		name := fmt.Sprintf("Step %d of %d: %s", cur+1, n, s.Steps[cur])
		if s.Label != "" {
			name = fmt.Sprintf("%s, step %d of %d: %s", s.Label, cur+1, n, s.Steps[cur])
		}
		items = append(items, core.AccessibilityLabel(name))
	}
	for _, sp := range s.Style {
		items = append(items, sp)
	}

	for i, step := range s.Steps {
		state := stepUpcoming
		switch {
		case i < cur:
			state = stepDone
		case i == cur:
			state = stepCurrent
		}
		if i > 0 {
			items = append(items, s.rule(t, i-1 < cur))
		}
		items = append(items, s.step(t, i, step, state))
	}
	return core.Scroll(items...).Render(ctx)
}

// step builds one circle-and-label cell. The cell, not the circle, is the
// target and carries the name, so the whole step is one tab stop.
func (s StepIndicator) step(t *core.Theme, i int, label string, state stepState) core.View {
	name := fmt.Sprintf("Step %d: %s", i+1, label)
	labelColor, weight := t.Colors.TextPrimary, core.Normal
	switch state {
	case stepDone:
		name += ", done"
	case stepCurrent:
		name += ", current"
		weight = core.Bold
	case stepUpcoming:
		labelColor = t.Colors.TextSecondary
	}

	cell := []core.PropsAndChildren{
		core.Gap(float64(t.Spacing.XS)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Padding(0),
		// A step never gives up width to its neighbours: the strip scrolls
		// instead, and a squeezed label would wrap mid-word.
		core.FlexShrink(0),
		core.AccessibilityLabel(name),
	}
	if state == stepDone && s.OnTap != nil {
		idx := i // captured per step: the handler outlives this call
		cell = append(cell,
			core.AccessibilityRole(core.RoleButton),
			core.OnClick(func() { s.OnTap(idx) }),
		)
	}
	cell = append(cell,
		s.circle(t, i, state),
		core.Text(label,
			core.UseStyle(t.Typography.Body),
			core.TextColor(labelColor),
			core.FontWeight(weight),
			core.AccessibilityHidden(),
		),
	)
	return core.Row(cell...)
}

// circle draws the numbered or ticked disc for one step.
func (s StepIndicator) circle(t *core.Theme, i int, state stepState) core.View {
	glyph := strconv.Itoa(i + 1)
	props := []core.PropsAndChildren{
		core.Width(fmt.Sprintf("%dpx", stepCircle)),
		core.Height(fmt.Sprintf("%dpx", stepCircle)),
		core.FlexShrink(0),
		core.Padding(0),
		core.Gap(0),
		core.BorderRadius(stepCircle / 2),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Justify(core.JustifyCenter),
		core.AccessibilityHidden(),
	}

	var ink string
	switch state {
	case stepDone:
		glyph = "✓"
		fill := t.Colors.SuccessColor()
		ink = VariantSuccess.Ink(t, fill)
		props = append(props, core.BackgroundColor(fill))
	case stepCurrent:
		fill := t.Colors.Primary
		ink = VariantDefault.Ink(t, fill)
		props = append(props, core.BackgroundColor(fill))
	default:
		stroke := t.Colors.ControlBorder
		if stroke == "" {
			stroke = t.Colors.BorderColor()
		}
		ink = t.Colors.TextSecondary
		props = append(props, core.BorderWidth(2), core.BorderColor(stroke))
	}

	props = append(props, core.Text(glyph,
		core.UseStyle(t.Typography.Caption),
		core.TextColor(ink),
		core.FontWeight(core.Bold),
	))
	return core.Column(props...)
}

// rule draws the short line between two steps, green once the step before it
// is done so the finished part of the flow reads as one run.
func (s StepIndicator) rule(t *core.Theme, done bool) core.View {
	color := t.Colors.BorderColor()
	if done {
		color = t.Colors.SuccessColor()
	}
	return core.Box(
		core.Width(fmt.Sprintf("%dpx", t.Spacing.MD)),
		core.Height("2px"),
		core.FlexShrink(0),
		core.BackgroundColor(color),
		core.AccessibilityHidden(),
	)
}
