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

// OnTouch fires the moment a finger (or pen, or mouse button) goes down on
// the node, before the press has resolved into a tap or a long press. Use it
// for immediate feedback — a sound, a highlight — not for the action itself:
// a press that slides off the node still fired this.
//
// Web only today. The DOM runtime maps it to pointerdown; neither native
// renderer reads the prop, so a node carrying it on iOS or Android simply
// never hears from it. (It used to be worse: the DOM's event-name fallback
// derived "touch", which is not a DOM event either, so the prop attached a
// listener nothing could ever fire on any target.)
func OnTouch(handler func()) BehaviorProp {
	return On("Touch", handler)
}

// OnLongPress fires after 500ms of held press without the finger lifting —
// the default on all three targets (UILongPressGestureRecognizer,
// Android's ViewConfiguration, and the DOM runtime's own timer).
//
// A node may carry both OnClick and OnLongPress, and one gesture produces
// exactly one handler call: Compose's combinedClickable splits them natively,
// while SwiftUI and the DOM each suppress the tap that follows a fired long
// press. Which handler runs is decided by how long the press was held, never
// by both running.
//
// Wired on all three targets, containers and leaves alike, including Button —
// which needs its own wiring on both natives, since a Button draws its own
// control rather than going through the generic gesture path.
func OnLongPress(handler func()) BehaviorProp {
	return On("LongPress", handler)
}

// OnBack claims the platform's system back — Android's back button and back
// gesture — for as long as the node carrying it is on screen. While any such
// node is, a back press runs the innermost one's handler instead of the
// platform default; when none is, back does what the platform would have done
// anyway, which on Android is to leave the app.
//
// core.Navigator attaches one to the route it shows whenever core.CanPop is
// true, so a pushed screen pops on back with no app code. comps.AppBar
// attaches its back arrow's action, and comps.Drawer its OnDismiss while open.
//
// # A prop, not a host event
//
// The other things a shell reports without a callback — lifecycle, a deep
// link — arrive as host events. Back cannot, because Android has to know
// before the press whether the app will take it: OnBackPressedDispatcher
// consults its callbacks' enabled flags synchronously on the UI thread, and
// falls through to finishing the Activity if none is enabled. A host event
// reaches Go after that decision, so a shell built on one would need a second,
// Go→host "enabled" signal kept in step with the app's state. A prop already
// is that signal: its presence in the tree is the enabled flag, and the tree
// diff keeps the shell's copy current with no channel of its own.
//
// # Innermost wins
//
// Compose's BackHandler gives priority to the handler registered most
// recently. Handlers composed in one pass register parent before child, so
// the innermost wins; one composed later — a drawer that has just opened —
// outranks every handler already on screen.
//
//	Navigator route root  onBack = Pop            outermost, runs last
//	  AppBar row          onBack = AppBar.OnBack
//	  Drawer panel layer  onBack = OnDismiss      while Open; runs first
//
// The one ordering this gets wrong is a parent that gains the prop after its
// descendants already hold one: it registers last and outranks them. Keep
// OnBack on nodes whose lifetimes nest the way their handlers should, which
// the three above do.
//
// # Hosts
//
//	Android  RenderNode wraps the node in androidx's BackHandler. A node
//	         hidden with Display none is not composed, so its handler is
//	         inactive while hidden. A Modal needs none: the Dialog window
//	         reports back through the Modal's own onDismiss.
//	iOS      nothing. There is no system back; the edge swipe belongs to a
//	         UINavigationController, which the SwiftUI renderer does not use.
//	Web      nothing. The browser's back button moves the page's history,
//	         which the page owns (examples/tutorial/deeplink.go's "route"
//	         host event is that arrangement). The runtime skips the prop
//	         rather than attach a listener for a "back" DOM event that does
//	         not exist, and htmlout does not export it.
func OnBack(handler func()) BehaviorProp {
	return On("Back", handler)
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
