package components

import "github.com/rohanthewiz/grmob/core"

// InputRow is the composer: a single-line text field that fills the row, and
// an optional trailing button that commits it.
//
//	Row (Gap)
//	  ├─ Input   ← FlexGrow(1), value / placeholder / onChange / onSubmit
//	  └─ Button  ← only when Button.Label is set; OnTap defaults to OnSubmit
//
// Two call sites spelled this out by hand — chat's message composer and
// todoapp's entry row — and they were near-identical down to the Gap(8) and
// the FlexGrow(1). What they also shared was the wiring, which is the part
// that is easy to get subtly wrong: one commit action reached by three paths
// (the keyboard's return/done key, the trailing button, and — through
// onChange — the Go state that both of them read).
//
// # The three paths, and why OnSubmit drives two of them
//
// A composer is not "a field and a button that happen to sit together": the
// button *is* the field's submit, rendered as a tap target for the case where
// the keyboard's return key is not obvious or not reachable. Both hand-written
// sites therefore named the same helper twice, once as core.InputWithSubmit's
// onSubmit and once as the button's OnTap. Here OnSubmit is the commit action
// and the button inherits it, so the two cannot drift apart:
//
//	components.InputRow{
//	    Value:       draft.Get(),
//	    Placeholder: "What needs doing?",
//	    OnChange:    func(v string) { draft.Set(v) },
//	    OnSubmit:    addTodo,
//	    Button:      components.Button{Label: "Add"},
//	}
//
// Setting Button.OnTap explicitly still wins — a "Send anyway" that skips a
// validation the keyboard path performs is a real shape — but it has to be
// said out loud.
//
// # Gap defaults to the theme's step, unlike Screen's
//
// Screen's Gap treats zero as "don't set one", because the spacing between a
// screen's sections is the app's decision and a theme's Column base may
// already carry one. The gap here is the opposite kind of thing: it is the
// widget's own internal layout — the field and the button must not touch —
// so InputRow owns it the way FormField owns the spacing between its label
// and its input. Zero therefore means "the theme's SM step" (8pt in both
// bundled themes, which is exactly what both hand-written sites had picked).
//
// A caller that genuinely wants no gap says so through Style, which is applied
// after: Style: []core.StyleProp{core.Gap(0)}.
//
// # The trailing button is optional
//
// A zero Button renders nothing at all — no node, not an empty one — so a
// search field or a filter box that commits on the return key alone is the
// same widget with one less field set. Absence is keyed on Label because a
// button with no visible label is not a button; a glyph button spells its
// glyph ("✕", "→") in Label and its meaning in AccessibilityLabel.
//
// # The input is owned, not slotted
//
// Unlike FormField, which takes whatever input it is given, InputRow builds
// its own — the wiring above *is* the widget, and a slot would hand it back to
// the caller. The consequence is that the field itself takes no per-call
// styling; a composer that needs to restyle its input has outgrown this and
// should go back to core.Row + core.InputWithSubmit.
type InputRow struct {
	// Value is the field's text. The field is fully controlled: it displays
	// exactly this, and OnChange is the only way it changes.
	Value string

	// Placeholder is the empty-state text inside the field. It is the field's
	// only label — InputRow has no caption slot; wrap it in a FormField when
	// one is needed.
	Placeholder string

	// OnChange fires on every keystroke and is what keeps Value in step with
	// what the user typed. Without it the field is read-only in practice: the
	// next render paints Value back over the keystrokes.
	OnChange func(string)

	// OnSubmit is the commit action: the keyboard's return key (iOS) or IME
	// done action (Android), and the trailing button's tap.
	//
	// When it is nil the field is built without a submit path at all rather
	// than with one wired to a no-op, so neither platform advertises a submit
	// affordance the row would ignore.
	OnSubmit func()

	// Button is the trailing commit button. Its zero value renders nothing;
	// a Button with a Label renders with OnTap defaulted to OnSubmit. Every
	// other Button field — Variant, Emphasis, Disabled, Style, the
	// accessibility pair — works as it does anywhere else.
	Button Button

	// Gap is the horizontal spacing between the field and the button, in
	// points. Zero means the theme's SM step, not zero spacing; see above.
	Gap float64

	// Style is applied to the row after Gap, so a caller can override it —
	// or add the background and padding that make the row read as a docked
	// bar, which the widget itself has no opinion about.
	Style []core.StyleProp
}

func (r InputRow) Render(ctx *core.Context) *core.Node {
	gap := r.Gap
	if gap == 0 {
		gap = float64(ctx.Theme().Spacing.SM)
	}

	// Exact capacity: the gap prop, the caller's overrides, the field, and at
	// most one button.
	items := make([]core.PropsAndChildren, 0, len(r.Style)+3)
	items = append(items, core.Gap(gap))
	// Caller Style last among the props: containerNode applies them in
	// argument order, so anything here wins over the gap above.
	for _, sp := range r.Style {
		items = append(items, sp)
	}

	// FlexGrow(1) is what makes this a composer rather than two things side
	// by side: the field takes the row's leftover width and the button keeps
	// its intrinsic size.
	// A nil OnChange is substituted with a no-op before it reaches core:
	// the callback registry invokes handlers unguarded, so an InputRow left
	// without OnChange panicked on the first keystroke instead of behaving
	// as the read-only field the doc above describes.
	onChange := r.OnChange
	if onChange == nil {
		onChange = func(string) {}
	}

	if r.OnSubmit != nil {
		items = append(items, core.InputWithSubmit(
			r.Value, r.Placeholder, onChange, r.OnSubmit, core.FlexGrow(1)))
	} else {
		items = append(items, core.Input(
			r.Value, r.Placeholder, onChange, core.FlexGrow(1)))
	}

	if r.Button.Label != "" {
		btn := r.Button
		if btn.OnTap == nil {
			btn.OnTap = r.OnSubmit
		}
		items = append(items, btn)
	}

	return core.Row(items...).Render(ctx)
}
