package core

// KeyboardAware makes a region yield to the software keyboard instead of
// being covered by it: while the keyboard is up, the node it is applied to
// ends where the keyboard begins.
//
//	core.Scroll(
//	    core.KeyboardAware(),
//	    form,
//	)
//
// # The two shapes it takes
//
// On a *scrolling* node (Scroll, List) it shrinks the viewport. The content
// does not move on its own — but because the viewport now ends above the
// keyboard, the platform's own "scroll the focused field into view" behavior
// (Compose's BasicTextField, SwiftUI's ScrollView) lands the field somewhere
// the user can see, which it cannot do while the viewport still claims the
// rows the keyboard is sitting on. This is the form case.
//
// On any *other* node it lifts that subtree whole. That is the case for a
// screen with something docked at the bottom — a chat composer, a checkout
// bar — which is outside the scrolling region by construction and would
// otherwise be the one thing the keyboard covers. Applied to a whole screen's
// column, it is the classic "the app resizes for the keyboard" behavior,
// asked for explicitly and by one screen at a time.
//
// # What it deliberately does not do
//
// It does not dismiss the keyboard on a tap outside: there is no focus or
// blur event in the framework yet, so nothing here can observe a field being
// left. (Dragging a keyboard-aware scroll region does dismiss it on iOS — see
// below — which is the platform's own gesture, not a handler of ours.)
//
// # What each platform does with it
//
//	Android   Modifier.imePadding() on the node, injected at the one funnel
//	          every node passes through. It needs the window to have stopped
//	          fitting the system windows itself, which the demo activity does
//	          (enableEdgeToEdge plus windowSoftInputMode="adjustResize"); an
//	          app that skips both gets the platform's whole-window resize
//	          instead and this prop then reads a consumed, zero-height inset.
//	iOS       SwiftUI treats the keyboard as its own safe-area region and
//	          insets for it by itself, so the shrink is the platform default
//	          with or without the flag. What the flag adds is
//	          .scrollDismissesKeyboard(.interactively) on the two scrolling
//	          node types: dragging the region puts the keyboard away.
//	HTML/WASM Nothing. A browser has no software keyboard to inset for, and
//	          the exported page scrolls the focused field into view natively.
//
// That asymmetry is why this is a flag and not simply what Scroll always
// does: on iOS the shrink is free, on Android it costs a window-level opt-in
// and a per-region decision about which thing should move, and a Go app
// should be able to name the region without knowing either.
//
// It is also why SafeArea does not carry it. The safe area on Android is
// WindowInsets.safeDrawing, which bundles the IME in with the system bars —
// applied there it would resize every screen whole and, worse, consume the
// inset so that a Scroll asking for it received nothing. The renderer
// subtracts the IME from that set for exactly this reason, leaving the
// keyboard to whichever node asked for it: the same split SwiftUI makes.
func KeyboardAware() BehaviorProp {
	return behaviorFunc(func(ctx *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["keyboardAware"] = true
	})
}
