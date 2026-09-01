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

func TextArea(value string, onChange func(string), rows int, props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return leafNode(ctx, "TextArea", ctx.Theme().Components.TextArea, map[string]any{
			"value":    value,
			"rows":     rows,
			"onChange": ctx.registerTextCallback(onChange),
		}, props)
	})
}
