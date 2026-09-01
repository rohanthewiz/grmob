package core

// Button takes the same mixed argument list the inputs do — style props and
// behavior props in any order — rather than the `...StyleProp` it took
// originally:
//
//	core.Button("Delete", onDelete,
//	    core.BackgroundColor(theme.Danger),
//	    core.OnLongPress(confirmDestructive),
//	)
//
// It was the last leaf that could not carry a behavior prop, which meant
// OnLongPress — a gesture both native renderers already wire and a button is
// the most natural home for — was unreachable on the one node type that
// exists to be pressed. Widening it through leafNode closes that and puts
// every leaf on one argument contract.
//
// The widening is source-compatible for the same reason the inputs' was: a
// StyleProp is a PropsAndChildren, so every existing core.Button(label, fn,
// core.Padding(8)) call compiles untouched. The shape it does break is
// forwarding — a []StyleProp cannot be spread into a ...PropsAndChildren — so
// a wrapper that collected style props into a slice has to widen its own
// slice to []core.PropsAndChildren. components.Button and components.Chip are
// the two in this tree that did.
//
// See leafNode for the ordering and nil contracts, and for why a View passed
// here is a debug-mode concern rather than a silent no-op.
//
// Button is deliberately absent from focusableLeafTypes: a phone does not
// give a button keyboard focus, so it carries no focus-command stamp and a
// core.Focus aimed at one would do nothing. FocusTarget still applies if an
// app wants the stamp anyway — see core/focus.go.
func Button(label string, onClick func(), props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return leafNode(ctx, "Button", ctx.Theme().Components.Button, map[string]any{
			"label": label,
			// Registered unconditionally, nil handler included: the renderers
			// key off the prop's presence, and components.Button relies on
			// core.Button accepting whatever it is handed rather than
			// second-guessing a nil. Changing that here would silently drop
			// the "on" prop and change what a disabled button diffs to.
			"onClick": ctx.registerCallback(onClick),
		}, props)
	})
}

// ButtonWithEvent is Button with the event name chosen by the caller, for the
// gestures that have no dedicated builder. It is largely superseded by the
// widening above — core.Button(label, fn, core.On("LongPress", g)) says the
// same thing and keeps the click — but it stays because it is the only way to
// build a button whose *only* wiring is a non-click event.
func ButtonWithEvent(label string, event string, handler func(), props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return leafNode(ctx, "Button", ctx.Theme().Components.Button, map[string]any{
			"label":      label,
			"on" + event: ctx.registerCallback(handler),
		}, props)
	})
}
