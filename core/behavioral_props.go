package core

// BehaviorProp attaches event behavior to a node. Apply takes the rendering
// Context (unlike StyleProp) because registering the handler needs the
// context's callback registry — the registry is per-app state on the context
// tree, not a package global.
type BehaviorProp interface {
	Apply(*Context, *Node)
}

type behaviorFunc func(*Context, *Node)

func (f behaviorFunc) Apply(ctx *Context, n *Node) {
	f(ctx, n)
}
func On(event string, handler func()) BehaviorProp {
	return behaviorFunc(func(ctx *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["on"+event] = ctx.registerCallback(handler)
	})
}

func OnClick(handler func()) BehaviorProp {
	return On("Click", handler)
}

func OnTouch(handler func()) BehaviorProp {
	return On("Touch", handler)
}

// OnLongPress fires after the platform's long-press timeout (~500ms) without
// the finger lifting. A node may carry both OnClick and OnLongPress: the
// renderers wire them as one gesture recognizer (combinedClickable on
// Android, tap+longPress on iOS), so a long press never also fires the click.
func OnLongPress(handler func()) BehaviorProp {
	return On("LongPress", handler)
}

// OnFocus fires when the node becomes the input focus — a text field the user
// has tapped into, with the software keyboard on its way up.
//
// OnBlur fires when that focus leaves. The pair is deliberately *not* a single
// bool-carrying handler: the two edges are almost never handled together (a
// form reveals errors on blur and does nothing on focus; a search box does the
// opposite), and one prop per edge lets a node carry only the edge it cares
// about instead of registering a callback to ignore half its calls.
//
// Both ride the void-callback channel, exactly like OnClick — the edge itself
// is the whole payload, and "which node" is already answered by which
// callback ID the platform dispatches.
//
// Focus is a leaf concern in practice: the renderers wire these on the text
// input node types, which are the only things a mobile platform gives focus
// to. The props are attachable to any node because BehaviorProp is uniform,
// but a Row carrying OnFocus will simply never hear from the native
// renderers.
//
// Ordering note: the framework guarantees the edges are dispatched in the
// order they happened, but *not* that a blur on the field being left arrives
// before the focus on the field being entered — that ordering is the
// platform's, and Android and iOS do not agree on it. Handlers must therefore
// be independent: read the field the callback belongs to, never "the field
// that is focused now".
func OnFocus(handler func()) BehaviorProp {
	return On("Focus", handler)
}

// OnBlur fires when the node loses input focus. See OnFocus for the pairing
// and the ordering caveat.
func OnBlur(handler func()) BehaviorProp {
	return On("Blur", handler)
}
