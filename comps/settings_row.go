package comps

import "github.com/rohanthewiz/grmob/core"

// SwitchRow is the settings-screen row: a title, an optional subtitle, a
// switch on the trailing edge, and the whole row tappable rather than only the
// switch.
//
//	comps.SwitchRow{Title: "Notifications", Subtitle: "Push and email",
//	    On: notify.Get(), OnToggle: notify.Set}
//
// It is ListRow with the trailing slot fixed to core.Switch and OnTap wired to
// the same setter. CheckboxRow, below, is the same row with a core.Checkbox:
// the switch for a setting that takes effect on the tap, the checkbox for a
// value a form collects (see core.Switch for why the two are different
// controls).
//
// # One tap, one state change
//
// The row and the control are both tappable, and the targets disagree about
// what a tap on the control does:
//
//	target    tap on the control                      handlers that fire
//	───────   ─────────────────────────────────────   ──────────────────────
//	Compose   Switch's toggleable consumes the press   control only
//	SwiftUI   Toggle's gesture wins over the row's     control only
//	web       native toggle, then the click BUBBLES    row (click), then
//	          to the row <div>, then `change` fires    control (change)
//
// On the web one tap therefore reaches Go twice. Go cannot tell which element
// the click landed on, so the row cannot skip its handler for taps "on the
// control". Two other ways out were considered and rejected:
//
//   - Render the control Disabled and let the row be the only handler. That is
//     what the web runtime would need (a disabled control gets
//     pointer-events:none, so the click falls through), but Disabled is the
//     platform's disabled state, not a hit-test flag: Material, SwiftUI and
//     the browser all draw the control greyed, and every screen reader says
//     "dimmed". A settings screen of disabled-looking switches is broken.
//   - Give the control a no-op handler. On both natives the control consumes
//     its own press, so tapping the switch itself would do nothing.
//
// What makes the double dispatch harmless is that both handlers *set* a value
// rather than flip one, and both go through one guard:
//
//	set(v) = if v != On { OnToggle(v) }
//
// The row calls set(!On) with the On it was rendered with. The control calls
// set(v) with the value the platform reports. On the web the row's click is
// dispatched first, Go re-renders, and the registry now holds closures over
// the new On (callback IDs are positional per pass and a pass overwrites each
// handler; see core's callbackRegistry). The `change` that follows reports the
// same v, which now equals On, so the guard drops it. If the two events ever
// arrived within one pass, both would compute the same target value and a
// setter is idempotent. Either way OnToggle sees exactly one change per tap.
//
// OnToggle is therefore documented as a setter: it receives the new value, and
// a caller should apply it rather than invert its own state.
//
// # Accessibility
//
// The control carries AccessibilityLabel(Title) and, when present,
// AccessibilityHint(Subtitle), so a reader landing on it hears "Notifications,
// switch, on" rather than an unnamed switch; a platform control has no label
// of its own (core.Switch leaves it to the caller). The row takes no role:
// ListRow's rule is that a container is not relabelled, and the switch or
// checkbox is the control a reader is looking for.
//
// # Theme roles read
//
// Everything ListRow reads (Spacing.SM gap, Typography.Body and Caption for the
// text), plus Components.CheckBox through the control.
type SwitchRow struct {
	// Title is the setting's name. It is also the control's accessible label.
	Title string

	// Subtitle is the secondary line under the title, and the control's
	// accessibility hint.
	Subtitle string

	// Leading is an optional icon or avatar before the text, as in ListRow.
	Leading core.View

	// On is the caller's current value.
	On bool

	// OnToggle receives the new value. It is a setter — apply the value; do
	// not invert your own state — because on the web the row and the switch
	// can both report one tap (see the type doc).
	OnToggle func(on bool)

	// Disabled disables both the switch and the row: no taps reach OnToggle,
	// and the control is announced as disabled.
	Disabled bool

	// Style is passed to the underlying ListRow and so beats its defaults.
	Style []core.StyleProp
}

// Render builds the ListRow described in the type doc.
func (r SwitchRow) Render(ctx *core.Context) *core.Node {
	set := toggleSetter(r.On, r.OnToggle)
	control := core.Switch(r.On, set, controlProps(r.Title, r.Subtitle, r.Disabled)...)
	return toggleRow(r.Title, r.Subtitle, r.Leading, control, r.On, set, r.Disabled, r.Style).Render(ctx)
}

// CheckboxRow is SwitchRow with a core.Checkbox on the trailing edge: the
// "I agree to the terms" row, or a filter a form will apply later. Everything
// in SwitchRow's doc applies unchanged, including OnToggle being a setter.
//
//	comps.CheckboxRow{Title: "I agree to the terms", Checked: ok.Get(), OnToggle: ok.Set}
//
// Inside a comps.FormField the field owns the label, so leave Title empty and
// put the sentence in the field's Label, or keep Title and give the field no
// label; the row renders the same either way.
type CheckboxRow struct {
	// Title is the choice's text. It is also the checkbox's accessible label.
	Title string

	// Subtitle is the secondary line under the title, and the checkbox's
	// accessibility hint.
	Subtitle string

	// Leading is an optional icon or avatar before the text.
	Leading core.View

	// Checked is the caller's current value.
	Checked bool

	// OnToggle receives the new value. It is a setter; see SwitchRow.OnToggle.
	OnToggle func(checked bool)

	// Disabled disables both the checkbox and the row.
	Disabled bool

	// Style is passed to the underlying ListRow and so beats its defaults.
	Style []core.StyleProp
}

// Render builds the ListRow described in SwitchRow's doc.
func (r CheckboxRow) Render(ctx *core.Context) *core.Node {
	set := toggleSetter(r.Checked, r.OnToggle)
	control := core.Checkbox(r.Checked, set, controlProps(r.Title, r.Subtitle, r.Disabled)...)
	return toggleRow(r.Title, r.Subtitle, r.Leading, control, r.Checked, set, r.Disabled, r.Style).Render(ctx)
}

// toggleSetter wraps the caller's setter in the one guard both the row and the
// control dispatch through: a report of the value already rendered is dropped.
// That guard is what turns the web's two dispatches per tap into one call (see
// SwitchRow's doc). A nil setter yields a no-op rather than nil because the
// core controls register their handler unconditionally, and a nil handler in
// the registry panics when a native event arrives for it.
func toggleSetter(current bool, onToggle func(bool)) func(bool) {
	return func(v bool) {
		if onToggle == nil || v == current {
			return
		}
		onToggle(v)
	}
}

// controlProps names the trailing control after the row's text, and disables
// it with the row. Disabled(false) is never written: an absent flag is already
// enabled, and writing it would add a style field to every row for nothing.
func controlProps(title, subtitle string, disabled bool) []core.PropsAndChildren {
	props := make([]core.PropsAndChildren, 0, 3)
	if title != "" {
		props = append(props, core.AccessibilityLabel(title))
	}
	if subtitle != "" {
		props = append(props, core.AccessibilityHint(subtitle))
	}
	if disabled {
		props = append(props, core.Disabled(true))
	}
	return props
}

// toggleRow is the ListRow both widgets share. The row's tap sets the inverse
// of the rendered value, through the same guarded setter the control uses.
//
// A disabled row still registers its OnTap and carries Disabled(true) rather
// than dropping the handler: core.Style.Disabled's contract is that the
// callback stays registered so a native tap racing the disabling patch finds a
// handler, and the renderers refuse the dispatch (Compose drops the clickable,
// the web runtime sets pointer-events:none).
func toggleRow(title, subtitle string, leading, control core.View, current bool,
	set func(bool), disabled bool, style []core.StyleProp) ListRow {

	rowStyle := make([]core.StyleProp, 0, len(style)+1)
	if disabled {
		rowStyle = append(rowStyle, core.Disabled(true))
	}
	rowStyle = append(rowStyle, style...)

	return ListRow{
		Leading:  leading,
		Title:    title,
		Subtitle: subtitle,
		Trailing: control,
		OnTap:    func() { set(!current) },
		Style:    rowStyle,
	}
}
