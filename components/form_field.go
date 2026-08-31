package components

import "github.com/rohanthewiz/grmob/core"

// FormField wraps any input in the label / input / hint-or-error frame — the
// pattern element's form_field made its most-used component, transplanted:
// the wrapper owns the text furniture so every form in an app annotates its
// inputs the same way, and the Input slot keeps it agnostic to what is being
// wrapped (Input, TextArea, NumericInput, a custom picker...).
//
// The widget renders feedback; it does not produce any. Error is a string the
// caller supplies, and the thing that supplies it — including the decision
// about *when* a message should be visible at all — is package forms:
//
//	components.FormField{
//	    Label: "Email",
//	    Hint:  "We never share it",
//	    Error: form.Error("email"),
//	    Input: form.Input("email", "you@example.com"),
//	}
//
// That split is why the dependency runs one way and only in the caller: forms
// produces the strings and the bound controls, this widget frames them, and
// neither package imports the other.
type FormField struct {
	Label string
	// Hint is the quiet guidance line under the input. Error replaces it
	// when non-empty — a field shows one line of feedback, and an error
	// outranks guidance.
	Hint  string
	Error string
	Input core.View
	// Style is applied to the wrapping column after the field's own layout.
	Style []core.StyleProp
}

func (f FormField) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	// A tight stack: the theme Column's screen-level padding would push each
	// field into its own island, so it is zeroed (Padding(0) assigns, not
	// merges) and rows are separated by the finest spacing step instead —
	// the label/hint must read as attached to their input.
	items := make([]core.PropsAndChildren, 0, len(f.Style)+5)
	items = append(items, core.Padding(0), core.Gap(float64(t.Spacing.XS)))
	for _, sp := range f.Style {
		items = append(items, sp)
	}

	if f.Label != "" {
		// Caption scale but primary ink and bold: label-sized without
		// reading as de-emphasized the way a true caption does.
		items = append(items, core.Text(f.Label,
			core.UseStyle(t.Typography.Caption),
			core.TextColor(t.Colors.TextPrimary),
			core.FontWeight(core.Bold),
		))
	}
	if f.Input != nil {
		items = append(items, f.Input)
	}
	switch {
	case f.Error != "":
		items = append(items, core.Text(f.Error,
			core.UseStyle(t.Typography.Caption),
			core.TextColor(t.Colors.Error),
		))
	case f.Hint != "":
		items = append(items, core.Text(f.Hint,
			core.UseStyle(t.Typography.Caption),
		))
	}

	return core.Column(items...).Render(ctx)
}
