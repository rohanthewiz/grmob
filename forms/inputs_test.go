package forms_test

import (
	"testing"

	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/forms"
)

// The bound builders exist to make one class of bug unwritable: a control
// that reads one field and writes another. These tests therefore do not
// inspect the builders' arguments — they drive the callback the way a native
// event does and check that the value comes back out of the field the control
// was rendering.

func drive(t *testing.T, ctx *core.Context, v core.View) *core.Node {
	t.Helper()
	n := v.Render(ctx)
	if n == nil {
		t.Fatal("builder rendered nothing")
	}
	return n
}

func TestBoundBuildersReadAndWriteTheSameField(t *testing.T) {
	cases := []struct {
		name string
		// build renders the control for field "target"; a second field
		// "decoy" is always present so a builder that crossed its wires has
		// somewhere to land.
		build func(*forms.Form) core.View
		// fire dispatches the control's own event kind.
		fire func(ctx *core.Context, n *core.Node)
		want string
	}{
		{
			name:  "Input",
			build: func(f *forms.Form) core.View { return f.Input("target", "placeholder") },
			fire: func(ctx *core.Context, n *core.Node) {
				ctx.TriggerTextCallback(n.Props["onChange"].(string), "typed")
			},
			want: "typed",
		},
		{
			name:  "Password",
			build: func(f *forms.Form) core.View { return f.Password("target", "••••") },
			fire: func(ctx *core.Context, n *core.Node) {
				ctx.TriggerTextCallback(n.Props["onChange"].(string), "hunter22")
			},
			want: "hunter22",
		},
		{
			name:  "TextArea",
			build: func(f *forms.Form) core.View { return f.TextArea("target", 4) },
			fire: func(ctx *core.Context, n *core.Node) {
				ctx.TriggerTextCallback(n.Props["onChange"].(string), "a longer note")
			},
			want: "a longer note",
		},
		{
			name:  "Checkbox",
			build: func(f *forms.Form) core.View { return f.Checkbox("target") },
			fire: func(ctx *core.Context, n *core.Node) {
				ctx.TriggerBoolCallback(n.Props["onToggle"].(string), true)
			},
			// Bools ride the value map as text, in the spelling Values.Bool
			// and the Accepted rule both read.
			want: "true",
		},
		{
			name: "InputWithSubmit",
			build: func(f *forms.Form) core.View {
				return f.InputWithSubmit("target", "placeholder", nil)
			},
			fire: func(ctx *core.Context, n *core.Node) {
				ctx.TriggerTextCallback(n.Props["onChange"].(string), "committed")
			},
			want: "committed",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := core.NewContext()
			spec := specOf(
				forms.Field{Name: "target"},
				forms.Field{Name: "decoy", Initial: "untouched"},
			)
			form := render(ctx, spec)
			n := drive(t, ctx, c.build(form))

			c.fire(ctx, n)

			if got := form.Value("target"); got != c.want {
				t.Errorf("target = %q, want %q", got, c.want)
			}
			if got := form.Value("decoy"); got != "untouched" {
				t.Errorf("decoy = %q; the control wrote a field it does not render", got)
			}
		})
	}
}

func TestBoundBuildersRenderTheFieldsCurrentValue(t *testing.T) {
	ctx := core.NewContext()
	spec := specOf(
		forms.Field{Name: "email", Initial: "you@example.com"},
		forms.Field{Name: "bio", Initial: "hello"},
		forms.Field{Name: "terms", Initial: "true"},
	)
	form := render(ctx, spec)

	if got := drive(t, ctx, form.Input("email", "p")).Props["value"]; got != "you@example.com" {
		t.Errorf("Input value = %v", got)
	}
	if got := drive(t, ctx, form.TextArea("bio", 3)).Props["value"]; got != "hello" {
		t.Errorf("TextArea value = %v", got)
	}
	// Initial: "true" is how a box that starts ticked is declared.
	if got := drive(t, ctx, form.Checkbox("terms")).Props["checked"]; got != true {
		t.Errorf("Checkbox checked = %v, want true", got)
	}
	// rows and placeholder still reach the core builder untouched.
	if got := drive(t, ctx, form.TextArea("bio", 3)).Props["rows"]; got != 3 {
		t.Errorf("TextArea rows = %v", got)
	}
	if got := drive(t, ctx, form.Input("email", "you@example.com")).Props["placeholder"]; got != "you@example.com" {
		t.Errorf("Input placeholder = %v", got)
	}
}

// Style props are forwarded rather than swallowed: a bound control is the
// core control, not a restricted version of it.
func TestBoundBuildersForwardStyleProps(t *testing.T) {
	ctx := core.NewContext()
	form := render(ctx, specOf(forms.Field{Name: "a"}))

	n := drive(t, ctx, form.Input("a", "p", core.FlexGrow(1)))
	if n.Style == nil || n.Style.FlexGrow != 1 {
		t.Errorf("style props did not reach the control: %+v", n.Style)
	}
}

// InputWithSubmit's return key runs a full submit — validation included — not
// just the handler.
func TestInputWithSubmitValidatesBeforeCallingTheHandler(t *testing.T) {
	ctx := core.NewContext()
	spec := forms.Spec{Fields: []forms.Field{
		{Name: "code", Rules: []forms.Rule{forms.Required("Required")}},
	}}
	form := render(ctx, spec)

	ran := false
	n := drive(t, ctx, form.InputWithSubmit("code", "Promo code", func(forms.Values) { ran = true }))

	ctx.TriggerCallback(n.Props["onSubmit"].(string))
	if ran {
		t.Error("the handler ran with the field empty")
	}
	form = render(ctx, spec)
	if got := form.Error("code"); got != "Required" {
		t.Errorf("error after the return key = %q, want it revealed", got)
	}

	ctx.TriggerTextCallback(n.Props["onChange"].(string), "SPRING")
	n = drive(t, ctx, render(ctx, spec).InputWithSubmit("code", "Promo code", func(forms.Values) { ran = true }))
	ctx.TriggerCallback(n.Props["onSubmit"].(string))
	if !ran {
		t.Error("the handler did not run once the field was valid")
	}
}

// The end-to-end shape the package exists for: the form produces exactly the
// string components.FormField's Error slot has always rendered, and nothing
// in either package had to learn about the other.
func TestFormFieldRendersTheFormsError(t *testing.T) {
	ctx := core.NewContext()
	spec := forms.Spec{Fields: []forms.Field{
		{Name: "email", Rules: []forms.Rule{forms.Required("Required"), forms.Email("Not a valid address")}},
	}}

	field := func(f *forms.Form) *core.Node {
		return components.FormField{
			Label: "Email",
			Hint:  "We never share it",
			Error: f.Error("email"),
			Input: f.Input("email", "you@example.com"),
		}.Render(ctx)
	}

	// Before a submit the default policy shows the hint, not a complaint —
	// even though the field is already invalid.
	form := render(ctx, spec)
	n := field(form)
	if findText(n, "We never share it") == nil {
		t.Error("hint should show while nothing is revealed")
	}
	if findText(n, "Required") != nil {
		t.Error("an untouched, unsubmitted field must not be complaining")
	}

	// A failed submit turns the explanation on, and it replaces the hint.
	form.Submit(nil)
	form = render(ctx, spec)
	n = field(form)
	if findText(n, "Required") == nil {
		t.Error("the revealed error should render")
	}
	if findText(n, "We never share it") != nil {
		t.Error("an error outranks the hint")
	}

	// Typing something invalid moves to the next rule's message live.
	form.SetValue("email", "nope")
	form = render(ctx, spec)
	if findText(field(form), "Not a valid address") == nil {
		t.Error("the second rule's message should render once the first passes")
	}

	// And fixing it puts the hint back.
	form.SetValue("email", "you@example.com")
	form = render(ctx, spec)
	if findText(field(form), "We never share it") == nil {
		t.Error("the hint should return once the field is valid")
	}
}

// findText locates a Text node by its content, the same predicate walk the
// components tests use — structure is found by what it says, not by child
// index, so a decorative wrapper does not break the test.
func findText(n *core.Node, s string) *core.Node {
	if n == nil {
		return nil
	}
	if n.Type == "Text" && n.Props["content"] == s {
		return n
	}
	for _, c := range n.Children {
		if found := findText(c, s); found != nil {
			return found
		}
	}
	return nil
}
