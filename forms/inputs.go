package forms

import "github.com/rohanthewiz/grmob/core"

// The builders below are core's input constructors with the value/onChange
// pair already bound to a field name.
//
// They exist for one reason: the unbound spelling names the field three
// times, in three roles, and nothing checks that the three agree.
//
//	// The bug this makes possible — copy a field, change two of the three:
//	Error: form.Error("email"),
//	Input: core.Input(form.Value("email"), "...", form.OnChange("phone")),
//
// A field that reads one name and writes another is a text box that will not
// accept typing, with no error anywhere. Bound, the name is written once:
//
//	Input: form.Input("email", "you@example.com")
//
// Everything else about the control is unchanged — these forward their style
// props to the core builder, and a control with no binding here (a picker, a
// slider, core.NumericInput) is still built by hand out of Value and
// OnChange, which stay exported precisely for that.

// Input is a single-line text field bound to name.
func (f *Form) Input(name, placeholder string, styleProps ...core.StyleProp) core.View {
	return core.Input(f.Value(name), placeholder, f.OnChange(name), styleProps...)
}

// InputWithSubmit is Input plus the keyboard's return/done key wired to a
// submit of the whole form — the one-field form (a search box, a promo code)
// where the return key is the only commit affordance there is.
func (f *Form) InputWithSubmit(name, placeholder string, handler func(Values), styleProps ...core.StyleProp) core.View {
	return core.InputWithSubmit(
		f.Value(name), placeholder, f.OnChange(name), f.OnSubmit(handler), styleProps...)
}

// Password is a masked single-line field bound to name.
func (f *Form) Password(name, placeholder string, styleProps ...core.StyleProp) core.View {
	return core.InputPassword(f.Value(name), placeholder, f.OnChange(name), styleProps...)
}

// TextArea is a multi-line field bound to name.
//
// core.TextArea takes no placeholder, so a text area's guidance goes in the
// wrapping FormField's Label or Hint rather than inside the box.
func (f *Form) TextArea(name string, rows int, styleProps ...core.StyleProp) core.View {
	return core.TextArea(f.Value(name), f.OnChange(name), rows, styleProps...)
}

// Checkbox is a boolean control bound to name, storing "true"/"false" as the
// field's text (see Values.Bool). A box that starts ticked is declared with
// Field{Name: "terms", Initial: "true"}.
//
// A checkbox has no label of its own; the usual pairing is a ListRow, which
// centers the box against its title:
//
//	components.ListRow{Leading: form.Checkbox("terms"), Title: "I accept the terms"}
func (f *Form) Checkbox(name string, styleProps ...core.StyleProp) core.View {
	return core.Checkbox(f.Checked(name), f.OnToggle(name), styleProps...)
}
