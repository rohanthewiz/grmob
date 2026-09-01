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
// Everything else about the control is unchanged — these forward their props
// to the core builder, and a control with no binding here (a picker, a
// slider, core.NumericInput) is still built by hand out of Value and
// OnChange, which stay exported precisely for that.
//
// They also carry the one binding that is not just a convenience: under
// RevealOnBlur the field's error is gated on an edge only the platform can
// report, so these attach core.OnBlur without being asked. See blurProp.
//
// The props parameter widened from []core.StyleProp to
// []core.PropsAndChildren along with the core builders. It had to: a
// []StyleProp cannot be spread into a ...PropsAndChildren, so a forwarding
// wrapper is the one call shape core's widening does not carry through
// untouched. Callers passing style props are unaffected.

// blurProp is the field's blur binding, or nil when the reveal policy has no
// use for one.
//
// Gating on the policy rather than always attaching is deliberate. A blur
// prop is not free: it registers a callback, adds a prop the reconciler
// diffs, and — the part that actually matters — makes both native renderers
// dispatch an event into Go on every focus change, each of which wakes a
// render pass. Under RevealOnSubmit that pass can change nothing on screen,
// so it is pure round trip. The framework should not manufacture traffic no
// policy reads.
//
// The nil return is load-bearing rather than a sentinel to test for:
// core.leafNode skips a nil item the same way containerNode does (the
// MaybeProp contract), so the unwired case leaves no trace in the tree at
// all — a field under RevealOnSubmit renders exactly the node it always did.
//
// Consequence worth naming: switching Reveal mid-session changes how many
// callbacks each field registers, which shifts the pass-sequential IDs of
// everything rendered after it. That is the same shift any conditional
// subtree causes and it settles on the next pass; a Spec whose Reveal is a
// constant — which is every Spec in practice — never sees it.
func (f *Form) blurProp(name string) core.PropsAndChildren {
	if f.spec.Reveal != RevealOnBlur {
		return nil
	}
	return core.OnBlur(f.OnBlur(name))
}

// withBlur appends the blur binding to the caller's props.
//
// It appends rather than prepends so a caller's own props keep the argument
// order they were written in, and it copies rather than growing the caller's
// slice in place: props arrives as the variadic's backing array, and
// appending into spare capacity would write into whatever the caller passed.
func (f *Form) withBlur(name string, props []core.PropsAndChildren) []core.PropsAndChildren {
	blur := f.blurProp(name)
	if blur == nil {
		return props
	}
	out := make([]core.PropsAndChildren, 0, len(props)+1)
	out = append(out, props...)
	return append(out, blur)
}

// Input is a single-line text field bound to name.
func (f *Form) Input(name, placeholder string, props ...core.PropsAndChildren) core.View {
	return core.Input(f.Value(name), placeholder, f.OnChange(name), f.withBlur(name, props)...)
}

// InputWithSubmit is Input plus the keyboard's return/done key wired to a
// submit of the whole form — the one-field form (a search box, a promo code)
// where the return key is the only commit affordance there is.
func (f *Form) InputWithSubmit(name, placeholder string, handler func(Values), props ...core.PropsAndChildren) core.View {
	return core.InputWithSubmit(
		f.Value(name), placeholder, f.OnChange(name), f.OnSubmit(handler),
		f.withBlur(name, props)...)
}

// Password is a masked single-line field bound to name.
func (f *Form) Password(name, placeholder string, props ...core.PropsAndChildren) core.View {
	return core.InputPassword(f.Value(name), placeholder, f.OnChange(name), f.withBlur(name, props)...)
}

// TextArea is a multi-line field bound to name.
//
// core.TextArea takes no placeholder, so a text area's guidance goes in the
// wrapping FormField's Label or Hint rather than inside the box.
func (f *Form) TextArea(name string, rows int, props ...core.PropsAndChildren) core.View {
	return core.TextArea(f.Value(name), f.OnChange(name), rows, f.withBlur(name, props)...)
}

// Checkbox is a boolean control bound to name, storing "true"/"false" as the
// field's text (see Values.Bool). A box that starts ticked is declared with
// Field{Name: "terms", Initial: "true"}.
//
// A checkbox has no label of its own; the usual pairing is a ListRow, which
// centers the box against its title:
//
//	components.ListRow{Leading: form.Checkbox("terms"), Title: "I accept the terms"}
//
// No blur binding, unlike the text builders: a tick is a commit, not a draft,
// so there is no moment where the user is "still working on" a checkbox and
// nothing for leaving it to signal. Neither native platform gives a checkbox
// keyboard focus anyway. Under RevealOnBlur a required-but-unticked box
// therefore says nothing until the submit reveals it, which is the same
// treatment a field the user never visited gets.
func (f *Form) Checkbox(name string, props ...core.PropsAndChildren) core.View {
	return core.Checkbox(f.Checked(name), f.OnToggle(name), props...)
}
