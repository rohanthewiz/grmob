package comps

import "github.com/rohanthewiz/grmob/core"

// ConcernPasswordFieldInert is raised, in debug builds only, when a
// PasswordField has no OnChange. Every keystroke goes nowhere, so the field
// either snaps back to empty or, on a target that keeps typed text until a
// patch says otherwise, shows dots for a password the application never
// received — and a sign-in that fails with a filled-in field is the worst
// kind of broken, because it looks like a wrong password. The bar
// TimePicker's and PINInput's inert cases are reported against.
const ConcernPasswordFieldInert = "password-field-inert"

// PasswordField is a password input with a reveal toggle: the dots, and a
// "Show" button trailing them that swaps the dots for the text and back.
//
//	comps.FormField{
//	    Label: "Password",
//	    Error: form.Error("password"),
//	    Input: comps.PasswordField{
//	        Value:    pw.Get(),
//	        OnChange: pw.Set,
//	        Label:    "Password",
//	    },
//	}
//
//	┌ Row ──────────────────────────────────────────────┐
//	│ ┌ InputPassword FlexGrow(1) ───────────┐  [Show]  │
//	│ │ ••••••••••                           │   ghost  │
//	│ └──────────────────────────────────────┘          │
//	└───────────────────────────────────────────────────┘
//	      revealed: core.Input, and the button says Hide
//
// # It is the input, not the field
//
// The plan called this a FormField whose input swaps. It is built as the
// input instead, and goes in a FormField's slot like every other input,
// because FormField already owns the label, the hint, the error and the
// required mark — DatePicker's rule, stated there: a control that grew its
// own label would be a second way to write a form, worded and spaced slightly
// differently from every other field on the screen. It also keeps forms'
// Error and Required wiring exactly as it is for a plain Input.
//
// # The swap
//
// Revealed, the field is a core.Input; hidden, a core.InputPassword. The two
// are different node types, so the reconciler replaces the element rather
// than patching an attribute. That is harmless here, and the reason is where
// focus is: the reader just pressed the toggle, so focus is on the button and
// not in the field being replaced. The text survives because it was never the
// element's — it is Value, the caller's, drawn into whichever element is up.
//
// # The one piece of state is the widget's
//
// Whether the password is showing is held here, in a hook, and not handed to
// the caller. It is the test TimePicker's doc states — hold state in the
// widget only when no application wants it — and no application wants it: a
// form submits the value, never whether it was visible while being typed.
// Owning it means inheriting the hook rules, as DatePicker does: render a
// PasswordField unconditionally, in a stable position, every pass.
//
// It also resets on its own. The state lives as long as the screen's hooks
// do, and a Navigator frame that is popped starts fresh when it is pushed
// again, so a sign-in screen that is left and returned to opens hidden — the
// behaviour a shoulder-surfer's victim wants, without anyone writing it.
//
// # Accessibility
//
// The input is named by Label, since the FormField's visible label is a
// separate Text that no target associates with it. The toggle's visible
// caption changes (Show, Hide) but its accessible name does not: it is "Show
// password" with core.AccessibilitySelected stating whether it is pressed —
// aria-pressed on the web, the selected trait on the natives. That is ARIA's
// rule for a toggle button, and the reason for it: a name that flips between
// "Show" and "Hide" as it is pressed reads as two different buttons, and a
// reader cannot tell which state the field is in from a name that describes
// the action rather than the state.
//
// # Theme roles read
//
//	Input      Components.Input, through core.Input / core.InputPassword
//	Toggle     Colors.Primary's on-light tone, through a ghost Button
//	Gap        Spacing.XS between the input and the toggle
type PasswordField struct {
	// Value is the password, owned by the caller.
	Value string

	// OnChange receives every keystroke. Nil reports
	// ConcernPasswordFieldInert.
	OnChange func(string)

	// Placeholder is drawn in the empty field.
	Placeholder string

	// Label names the input for assistive technology ("Password"). It is not
	// drawn: the FormField around the field is the visible label.
	Label string

	// ShowLabel and HideLabel caption the toggle; empty gives "Show" and
	// "Hide".
	ShowLabel, HideLabel string

	// RevealLabel is the toggle's accessible name, the same in both states;
	// empty gives "Show password". See "Accessibility".
	RevealLabel string

	// Disabled disables the input and the toggle.
	Disabled bool

	// Style is applied to the row after its defaults.
	Style []core.StyleProp
}

// Render draws the input and its toggle. It takes one hook, the reveal state.
func (p PasswordField) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	// Before any branch, so the slot is taken on every pass whatever the
	// field is showing — the hook rule DatePicker's doc spells out.
	revealed := core.NewState(ctx, false)

	if core.IsDebugMode() && p.OnChange == nil {
		core.ReportConcern(ConcernPasswordFieldInert,
			"PasswordField has no OnChange, so every keystroke is dropped and the password never reaches the application")
	}

	onChange := p.OnChange
	if onChange == nil {
		// The input still needs a callback to register; one that drops the
		// text is what "inert" means, and the concern above has said so.
		onChange = func(string) {}
	}

	input := []core.PropsAndChildren{core.FlexGrow(1)}
	if p.Label != "" {
		input = append(input, core.AccessibilityLabel(p.Label))
	}
	if p.Disabled {
		input = append(input, core.Disabled(true))
	}

	var field core.View
	if revealed.Get() {
		field = core.Input(p.Value, p.Placeholder, onChange, input...)
	} else {
		field = core.InputPassword(p.Value, p.Placeholder, onChange, input...)
	}

	caption := orDefault(p.ShowLabel, "Show")
	if revealed.Get() {
		caption = orDefault(p.HideLabel, "Hide")
	}

	items := make([]core.PropsAndChildren, 0, len(p.Style)+5)
	items = append(items,
		// Inside a FormField, which already supplies the inset; the theme
		// Row base's padding would indent this input past its neighbours.
		core.Padding(0),
		core.Gap(float64(t.Spacing.XS)),
		core.AlignItemsProp(core.AlignItemsCenter),
	)
	items = append(items, asProps(p.Style)...)
	items = append(items,
		field,
		Button{
			Label:              caption,
			OnTap:              func() { revealed.Set(!revealed.Get()) },
			Emphasis:           EmphasisGhost,
			Disabled:           p.Disabled,
			AccessibilityLabel: orDefault(p.RevealLabel, "Show password"),
			Style: []core.StyleProp{
				// SelectedWhen, not "only when on": a toggle that says
				// nothing while off is announced as a plain button, and
				// "not pressed" is what tells the reader it is a toggle.
				core.AccessibilitySelected(core.SelectedWhen(revealed.Get())),
				// The caption changes width between Show and Hide; a toggle
				// that shoved the input a few pixels on every press would be
				// the field jumping under the reader's thumb.
				core.FlexShrink(0),
			},
		},
	)
	return core.Row(items...).Render(ctx)
}
