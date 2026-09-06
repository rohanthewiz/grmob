package core

import (
	"fmt"
	"strconv"
)

// The input builders all take the same mixed argument list the containers do
// — style props and behavior props in any order — rather than the
// `...StyleProp` they took originally. The widening is source-compatible (a
// StyleProp is a PropsAndChildren), and it is what lets a field carry
// OnFocus/OnBlur:
//
//	core.Input(v, "you@example.com", onChange,
//	    core.Padding(8),
//	    core.OnBlur(func() { form.MarkBlurred("email") }),
//	)
//
// See leafNode for the argument contract, for the one call shape the widening
// does break, and for why a View passed to one of these is a debug-mode
// concern rather than a silent no-op.

func Input(value string, placeholder string, onChange func(string), props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return leafNode(ctx, "Input", ctx.Theme().Components.Input, map[string]any{
			"value":       value,
			"placeholder": placeholder,
			"onChange":    ctx.registerTextCallback(onChange),
		}, props)
	})
}

// InputWithSubmit is Input plus a submit action: pressing the keyboard's
// return key (iOS) or IME done action (Android) dispatches onSubmit. The
// submit rides the existing void-callback channel — the renderers read the
// "onSubmit" prop and dispatch it exactly like a Button's onClick — so the
// bridge surface is unchanged. A separate builder rather than a variadic
// change to Input keeps every existing call site compiling untouched.
func InputWithSubmit(value string, placeholder string, onChange func(string), onSubmit func(), props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return leafNode(ctx, "Input", ctx.Theme().Components.Input, map[string]any{
			"value":       value,
			"placeholder": placeholder,
			"onChange":    ctx.registerTextCallback(onChange),
			"onSubmit":    ctx.registerCallback(onSubmit),
		}, props)
	})
}

func Checkbox(checked bool, onToggle func(bool), props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return leafNode(ctx, "Checkbox", ctx.Theme().Components.CheckBox, map[string]any{
			"checked":  checked,
			"onToggle": ctx.registerBoolCallback(onToggle),
		}, props)
	})
}

func InputPassword(value string, placeholder string, onChange func(string), props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return leafNode(ctx, "InputPassword", ctx.Theme().Components.Input, map[string]any{
			"value":       value,
			"placeholder": placeholder,
			"onChange":    ctx.registerTextCallback(onChange),
		}, props)
	})
}

func NumericInput(value int, onChange func(int), props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		id := ctx.registerTextCallback(func(val string) {
			if n, err := strconv.Atoi(val); err == nil {
				onChange(n)
			}
		})

		return leafNode(ctx, "NumericInput", ctx.Theme().Components.Input, map[string]any{
			"value":    fmt.Sprintf("%d", value),
			"onChange": id,
		}, props)
	})
}

// SelectOption is one choice in a Select: the value the app works in, and the
// label a person reads.
//
// Two fields rather than a bare string because the two are different things
// often enough to be worth the type — a country code and a country name, a
// status enum and a sentence — and a widget that took only strings would push
// every caller into keeping a parallel slice. An empty Label means "the value
// is readable enough", which is the common small case (a list of sizes, a list
// of years) and keeps that case a one-word literal.
type SelectOption struct {
	Value string
	Label string
}

// Option builds a SelectOption, mirroring Tab's constructor next door.
func Option(value, label string) SelectOption {
	return SelectOption{Value: value, Label: label}
}

// Select is the picker: one value chosen from a fixed list.
//
//	core.Select(form.Country, []core.SelectOption{
//	    {Value: "us", Label: "United States"},
//	    {Value: "pt", Label: "Portugal"},
//	}, func(v string) { form.Country = v })
//
// Controlled, like every other input here: the value shown is always the one
// Go passed, and a change goes up as an event. onChange carries the option's
// *Value*, never its label or its index — the index is the one identity that
// changes when the list is reordered, and a label is written to be read.
//
// # What each target draws
//
//	web       a <select>, whose options are a prop rather than child nodes
//	iOS       a Menu whose label is the chosen option's text
//	Android   a Box anchored to a DropdownMenu, same shape
//
// The natives are deliberately *not* built from a platform picker control
// (SwiftUI's .pickerStyle(.menu), Material's ExposedDropdownMenuBox). Both of
// those draw a frame of their own, and the whole rule this widget lands under
// is that the Go style owns the frame — see borderResetTypes in htmlout/tag.go,
// which the web half joins for the same reason. A control whose edge came from
// the platform on two targets and from the theme on two others is the
// divergence that rule exists to prevent.
//
// # It reads the theme's Input base
//
// A picker is a field: it sits in a form beside text inputs, and a picker that
// did not match the fields around it would look like a mistake. Reading
// Components.Input is also how it inherits the frame those fields grew — the
// same move components.DatePicker makes for the same reason, and the reason
// this widget needs no palette role of its own.
//
// # Options are a prop, not children
//
// A <select>'s options are elements, but they are not *nodes*: no patch is
// ever addressed to one, they carry no style, and they cannot hold a subtree.
// Sending them as children would put four renderers in the business of
// deciding which child is chrome, which is the complication core.TabView's
// tabs prop already avoids one node type over. The web renderer builds the
// <option> elements from the prop; both natives read the same list.
func Select(value string, options []SelectOption, onChange func(string), props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		// Flattened to []map[string]string here rather than in each renderer,
		// which is what TabView's tabs prop does and for the same reason: the
		// wire is JSON, and a Go struct crossing it as itself would oblige
		// every renderer to know the field names Go happened to capitalise.
		//
		// The label defaults to the value at this seam, once, so no renderer
		// has to carry the fallback and none of them can disagree about it.
		opts := make([]map[string]string, 0, len(options))
		for _, o := range options {
			label := o.Label
			if label == "" {
				label = o.Value
			}
			opts = append(opts, map[string]string{"value": o.Value, "label": label})
		}
		return leafNode(ctx, "Select", ctx.Theme().Components.Input, map[string]any{
			"value":    value,
			"options":  opts,
			"onChange": ctx.registerTextCallback(onChange),
		}, props)
	})
}

func TextArea(value string, onChange func(string), rows int, props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return leafNode(ctx, "TextArea", ctx.Theme().Components.TextArea, map[string]any{
			"value":    value,
			"rows":     rows,
			"onChange": ctx.registerTextCallback(onChange),
		}, props)
	})
}
