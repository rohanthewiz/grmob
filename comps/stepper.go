package comps

import (
	"strconv"

	"github.com/rohanthewiz/grmob/core"
)

// Stepper is a number with a − and a + beside it: a quantity in a cart, the
// guests on a booking, the font size in a settings screen.
//
//	comps.Stepper{Value: qty.Get(), Min: 1, Max: 20, OnChange: qty.Set, Label: "Quantity"}
//
// core.NumericInput is the typing form of the same value. A Stepper is the
// tapping form, for small ranges where two taps beat opening a keyboard; the
// examples hand-rolled it three times in chapter 1 of the tutorial alone.
//
//	┌ Row  role=group  label=Label  value="3" ─┐
//	│  [ − ]      3      [ + ]                 │
//	└──────────────────────────────────────────┘
//	   disabled          disabled
//	   at Min            at Max
//
// # Controlled, and clamped in the widget
//
// The widget holds no state. A tap computes Value∓Step, clamps it into the
// bounds and calls OnChange only when the result differs from Value, so a
// caller's handler is a plain setter and never sees a no-op or an
// out-of-range number. At a bound the button that would leave the range is
// rendered Disabled, which is what both natives and the browser announce as
// unavailable; the handler still stays registered, per core.Style.Disabled.
//
// # Bounds are opt-in: they apply when Max > Min
//
// Go has no "unset" int, and Min: 0, Max: 0 is not a range anyone means, so a
// zero-value pair leaves the stepper unbounded. Any range where Max > Min is
// enforced at both ends; a caller wanting only a floor sets a large Max.
//
// # Buttons are outlined, not ghost
//
// The plan sketched ghost buttons to keep the pair quiet. A ghost "−" is a bare
// glyph with no visible edge, which on a phone reads as text rather than as a
// target, so both buttons are EmphasisOutlined — the treatment the tutorial's
// hand-rolled stepper had already settled on.
//
// # Accessibility
//
// There is no spinbutton role in core.Role and none is added for this: the row
// is RoleGroup with Label as its name and the current value as an
// AccessibilityValue. Both natives read the value's Text on any node
// (Compose stateDescription, SwiftUI accessibilityValue). The web scopes
// aria-value* to progressbar and drops it here, and on the web the visible
// number between the buttons is read in document order instead. The buttons
// are named "Decrease"/"Increase" by default, because "−" is announced as
// "minus" or not at all; DecreaseLabel and IncreaseLabel localise them.
//
// # Theme roles read
//
//	Row gap         Spacing.SM
//	Value text      Typography.Body, bold
//	Buttons         comps.Button outlined: Colors.Primary's on-light tone
type Stepper struct {
	// Value is the caller's current number.
	Value int

	// Min and Max bound the value when Max > Min; otherwise it is unbounded.
	Min, Max int

	// Step is the amount one tap adds or removes. Zero or negative means 1.
	Step int

	// OnChange receives the new, clamped value. It is not called when a tap
	// would leave the value unchanged.
	OnChange func(int)

	// Label is the group's accessible name ("Quantity"). It is not drawn:
	// place the stepper in a ListRow's Trailing or a FormField for a visible
	// label, which keeps the stepper usable inside either.
	Label string

	// Format renders the value between the buttons and in the accessibility
	// value. Nil uses strconv.Itoa.
	Format func(int) string

	// Disabled disables both buttons regardless of the bounds.
	Disabled bool

	// DecreaseLabel and IncreaseLabel name the two buttons for screen readers.
	// Empty uses "Decrease" and "Increase".
	DecreaseLabel, IncreaseLabel string

	// Style is applied to the row after the widget's own props.
	Style []core.StyleProp
}

// bounded reports whether Min and Max form a range the stepper enforces.
func (s Stepper) bounded() bool { return s.Max > s.Min }

// step is the per-tap increment, defaulted to 1.
func (s Stepper) step() int {
	if s.Step <= 0 {
		return 1
	}
	return s.Step
}

// clamp folds v into the bounds when there are any.
func (s Stepper) clamp(v int) int {
	if !s.bounded() {
		return v
	}
	return min(max(v, s.Min), s.Max)
}

// text is the value as drawn and announced.
func (s Stepper) text() string {
	if s.Format != nil {
		return s.Format(s.Value)
	}
	return strconv.Itoa(s.Value)
}

// Render builds the group row described in the type doc.
func (s Stepper) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	text := s.text()

	// The value's numbers are stated only for a bounded stepper: a ValueRange
	// needs a Min below its Max to mean anything, and an unbounded one would
	// be reported as unusable by the a11y audit. The Text half is what the
	// natives read either way.
	value := core.ValueRange{Text: text}
	if s.bounded() {
		value = core.ValueOf(float64(s.Value), float64(s.Min), float64(s.Max)).WithText(text)
	}

	items := make([]core.PropsAndChildren, 0, len(s.Style)+9)
	items = append(items,
		core.Gap(float64(t.Spacing.SM)),
		core.AlignItemsProp(core.AlignItemsCenter),
		// The theme Row base pads free-standing rows; a stepper is a control
		// that sits inside a row or a field, which already supplies the inset.
		core.Padding(0),
		core.AccessibilityRole(core.RoleGroup),
		core.AccessibilityValue(value),
	)
	if s.Label != "" {
		items = append(items, core.AccessibilityLabel(s.Label))
	}
	for _, sp := range s.Style {
		items = append(items, sp)
	}

	items = append(items,
		s.button("−", orDefault(s.DecreaseLabel, "Decrease"), -s.step(),
			s.bounded() && s.Value <= s.Min),
		core.Text(text, core.UseStyle(t.Typography.Body), core.FontWeight(core.Bold)),
		s.button("+", orDefault(s.IncreaseLabel, "Increase"), s.step(),
			s.bounded() && s.Value >= s.Max),
	)
	return core.Row(items...).Render(ctx)
}

// button builds one of the pair. atBound disables it when the tap would push
// the value out of range; the clamp in the handler is a second guard for a tap
// that races the disabling patch.
func (s Stepper) button(glyph, name string, delta int, atBound bool) core.View {
	return Button{
		Label:              glyph,
		Emphasis:           EmphasisOutlined,
		Disabled:           s.Disabled || atBound,
		AccessibilityLabel: name,
		OnTap: func() {
			next := s.clamp(s.Value + delta)
			if next != s.Value && s.OnChange != nil {
				s.OnChange(next)
			}
		},
	}
}

// orDefault returns v, or fallback when v is empty.
func orDefault(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
