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
// neither package imports the other. Required follows the same shape — the
// form knows which fields reject an empty value, the widget only draws the
// mark:
//
//	Required: form.Required("email"),
type FormField struct {
	Label string

	// Required draws the conventional marker after the label. It is
	// annotation only: this widget validates nothing, and setting it does not
	// make an empty field fail — forms.Required is what does that.
	//
	// Which is why it is worth feeding it from forms.Form.Required rather
	// than from a literal true. A hand-set flag is a second, independent
	// claim about the same field, and the two drift the first time a rule is
	// added or dropped: a marked field that submits empty, or an unmarked one
	// the user cannot get past. The form derives its answer from the rules
	// themselves, so the mark cannot outlive the rule that justified it.
	//
	// Ignored when Label is empty — there is nothing to mark. A control whose
	// label lives elsewhere (the checkbox row in examples/signup, whose title
	// belongs to the ListRow) has to carry its own.
	Required bool

	// Hint is the quiet guidance line under the input. Error replaces it
	// when non-empty — a field shows one line of feedback, and an error
	// outranks guidance.
	Hint  string
	Error string
	Input core.View
	// Style is applied to the wrapping column after the field's own layout.
	Style []core.StyleProp
}

// requiredMarker is the glyph drawn after a required field's label. An
// asterisk is the convention every platform's forms already use, and a form
// wanting different copy ("(required)") can put it in the Label — a knob here
// would be a theme decision made one widget at a time.
const requiredMarker = "*"

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
		label := core.Text(f.Label,
			core.UseStyle(t.Typography.Caption),
			core.TextColor(t.Colors.TextPrimary),
			core.FontWeight(core.Bold),
		)
		if f.Required {
			// The marker is a sibling node rather than " *" glued onto the
			// label string, for two reasons that both need it to be its own
			// element: it is inked in the error color while the label stays
			// primary, and it carries an accessibility label so a screen
			// reader announces "required" instead of spelling out a star.
			//
			// The Row is built only on this branch, so an ordinary field
			// renders exactly the tree it did before this option existed —
			// one Text, no wrapper for a renderer to lay out.
			label = core.Row(
				// Row's theme padding is screen-level (8/16) and would inset
				// the label away from the input it belongs to; the finest
				// spacing step is the gap a space would have been.
				core.Padding(0),
				core.Gap(float64(t.Spacing.XS)),
				core.AlignItemsProp(core.AlignItemsCenter),
				label,
				core.Text(requiredMarker,
					core.UseStyle(t.Typography.Caption),
					core.TextColor(t.Colors.Error),
					core.FontWeight(core.Bold),
					core.AccessibilityLabel("required"),
				),
			)
		}
		items = append(items, label)
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
