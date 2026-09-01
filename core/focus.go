package core

import "sync"

// Focus commands: the imperative half of the focus system.
//
// core.OnFocus/OnBlur made focus *observable*. This file makes it
// *settable* — an app can put the cursor in a named field (core.Focus) or
// put the keyboard away entirely (core.DismissKeyboard).
//
// # Why this rides the render tree instead of a bridge call
//
// Go has exactly one channel to the native side: the render tree and the
// patches that update it. There is no imperative Go→native call, and adding
// one would mean a new wire format for all three renderers. So a focus
// command is expressed the only way anything reaches a screen here — as
// props on nodes, which the reconciler diffs and the renderers react to:
//
//	Focus(ref) / DismissKeyboard(ctx)
//	        │  epoch++, target = ref (or nil)
//	        ▼
//	next render pass: every focusable leaf stamps
//	        "focusEpoch"  : N       ── the command generation
//	        "focusAction" : "focus" │ "blur" │ ""
//	        ▼
//	reconciler emits update-props for the leaves whose stamp changed
//	        ▼
//	renderer keys on focusEpoch *changing* and runs focusAction once
//
// The epoch is a counter rather than a bool precisely because a command must
// be able to repeat: focusing the same field twice (a "try again" after a
// failed submit) has to re-fire, and two identical prop maps produce no patch
// and therefore no effect. Bumping the counter is what makes the second
// command observable.
//
// Nothing consumes the epoch. Once stamped it stays on the node through every
// later pass, unchanged, so the renderers see one edge per command — the same
// reason the stamp is safe to re-emit on unrelated re-renders.
//
// # Why three actions and not a bool
//
// The obvious encoding is `focused: true` on the target and `false` on
// everyone else, and it is wrong on both platforms. Consider focusing B while
// A holds focus: A would see `focused: false` and clear focus, B would see
// `true` and request it, and the two run in an order neither Compose
// (LaunchedEffect) nor SwiftUI (onChange) guarantees — so B's request can land
// first and A's clear then throws it away.
//
// The fix is to say what each node should *do*, not what it should be:
//
//	"focus"  request focus (target of a Focus command)
//	"blur"   release focus if you hold it (every leaf, on a dismiss)
//	""       do nothing — someone else is being focused; the platform will
//	         take focus away from you on its own, which is exactly right
//
// # The cost, stated plainly
//
// Every focus command re-stamps every focusable leaf in the tree, so a
// dismiss with six fields on screen emits six update-props patches. That is
// accepted deliberately: focus commands are rare and user-initiated (a tap, a
// submit), and the alternative — tracking which node holds focus in Go so
// only one needs stamping — would require the framework to wire OnFocus on
// every field always, manufacturing a Go render pass on every keyboard
// change. Six cheap patches beat continuous traffic.
//
// # The one caveat
//
// A core.Cached subtree returns the identical *Node every pass, and the
// reconciler treats pointer equality as proof nothing changed. A field inside
// a cached subtree therefore never re-stamps and never hears a focus command.
// Cache above the focus system or not at all.

// focusState is the app-instance focus command state, shared by pointer with
// every derived context exactly like the callback registry and the navigation
// stack — see Context's field block. Package-level would make two apps in one
// process (or two managers in one test binary) fight over one keyboard.
type focusState struct {
	mu    sync.Mutex
	epoch int
	// target is the ref that should receive focus. nil with epoch > 0 means
	// the last command was a dismiss, which is what turns every leaf's action
	// into "blur"; nil with epoch == 0 means no command has ever been issued
	// and nothing is stamped at all.
	target *FocusRef
	// ordered records that some UseFocusOrder has run in this app, and stays
	// set for its lifetime. It is the traversal half's equivalent of the
	// epoch-0 sentinel above: until it is set, FocusTarget writes no imeAction
	// key at all, so an app that never declares an order renders unchanged
	// trees. See setOrder in focus_order.go for why it never clears.
	ordered bool
}

func newFocusState() *focusState { return &focusState{} }

// command returns the stamp for a node holding ref. A nil ref is a focusable
// leaf that no app code named — it still participates, because a dismiss has
// to reach whichever field the *user* tapped into, and Go was never told
// which one that was.
//
// issued is a separate return rather than "epoch == 0" so the two meanings
// stay apart. Zero is the sentinel on the *wire* — each renderer checks it,
// because both props always travel together and a 0 must never be read as an
// instruction — but in here the question is whether a command exists at all,
// and answering it with a magic value would let a wrong action ride out under
// a zero epoch that nothing in Go would notice.
func (f *focusState) command(ref *FocusRef) (epoch int, action string, issued bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.epoch == 0 {
		// Nothing has ever been issued: stamp nothing at all, so an app that
		// never touches focus renders byte-identical trees to before.
		return 0, "", false
	}
	switch {
	case f.target == nil:
		return f.epoch, "blur", true
	case f.target == ref:
		return f.epoch, "focus", true
	default:
		return f.epoch, "", true
	}
}

// FocusRef names one focusable node so an app can put the cursor in it later.
//
// It is deliberately opaque and carries no state of its own beyond the
// context it belongs to: identity is the whole point, and identity is the
// pointer. That is also why it must be stable across render passes — see
// UseFocusRef.
type FocusRef struct {
	ctx *Context

	// Traversal membership, written by UseFocusOrder and read by FocusNext,
	// FocusPrevious and FocusTarget — always under focusState.mu, because a
	// command may be issued from a timer goroutine while a render pass is
	// rewriting the order. order is nil for a ref that belongs to no order,
	// which is what makes traversal opt-in. See focus_order.go.
	order *focusOrder
	index int
}

// UseFocusRef returns a FocusRef that is stable for the lifetime of this hook
// slot, which is what makes the ref usable as an identity:
//
//	email := core.UseFocusRef(ctx)
//
//	core.Input(v, "you@example.com", onChange,
//	    core.FocusTarget(email),
//	)
//	core.Button("Next", func() { core.Focus(email) })
//
// A hook rather than a bare constructor because a ref built inline in a
// render function is a new pointer every pass: FocusTarget would stamp one
// identity and the click handler would compare against another, so Focus
// would silently never match a node. Going through NewState both pins the
// pointer and reserves the cursor slot properly.
//
// It lives in core rather than hooks because hooks imports core and not the
// reverse; the focus state it closes over is on Context.
func UseFocusRef(ctx *Context) *FocusRef {
	// NewState keeps only the first value handed to it, so the ref allocated
	// on later passes is discarded and Get returns the original pointer. The
	// slot goes through a variable because State's accessors have pointer
	// receivers and NewState's return value is not addressable — the same
	// two-step hooks.UseMemo makes.
	slot := NewState(ctx, &FocusRef{ctx: ctx})
	return slot.Get()
}

// FocusTarget marks the node it is applied to as ref's node.
//
// It is an ordinary BehaviorProp, so it composes with OnFocus, OnBlur and
// everything else in the same argument list:
//
//	core.Input(v, "", onChange, core.FocusTarget(email), core.OnBlur(...))
//
// A nil ref returns a nil prop rather than panicking — leafNode and
// containerNode both skip a nil item (MaybeProp's contract), so
// `core.FocusTarget(maybeRef)` degrades to an unnamed field instead of
// crashing a render pass.
//
// Applying it to a node the platform never focuses (a Row, a Checkbox) is
// harmless but pointless: the stamp lands and no renderer reads it.
func FocusTarget(ref *FocusRef) BehaviorProp {
	if ref == nil {
		return nil
	}
	return behaviorFunc(func(ctx *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		// Stamped here as well as in leafNode so a FocusTarget on a node type
		// leafNode does not consider focusable still carries a complete,
		// self-consistent pair. Same values either way, so the overwrite is a
		// no-op on the ordinary path.
		stampFocus(ctx, n.Props, ref)
		// The traversal half. It lives here rather than in leafNode because,
		// unlike a focus command, it needs the ref: a field with no name has
		// no place in an order, so there is nothing to stamp on the fields
		// leafNode reaches by type alone.
		stampTraversal(ctx, n, ref)
	})
}

// stampFocus writes the current command onto a props map, or writes nothing
// when no command has ever been issued.
//
// Both keys are always written together, never one alone. That is not tidiness:
// an update-props patch carries the *whole new props map* and the renderers
// iterate the keys it contains, so a key that disappears between passes is
// invisible on the far side. A node that once carried focusAction must
// therefore keep carrying it — with the value "" when it has nothing to do —
// rather than dropping it.
func stampFocus(ctx *Context, props map[string]any, ref *FocusRef) {
	epoch, action, issued := ctx.focus.command(ref)
	if !issued {
		return
	}
	props["focusEpoch"] = epoch
	props["focusAction"] = action
}

// Focus puts the input focus — and with it the software keyboard — on ref's
// node, as of the next render pass.
//
// Called from an event handler, as the imperative counterpart to OnFocus:
//
//	core.Button("Next", func() { core.Focus(password) })
//
// Calling it for a ref whose node is not currently in the tree does nothing
// visible: no node stamps "focus", so no renderer acts. The command still
// consumes an epoch, which is correct — it happened, it simply had no target
// on screen.
//
// A nil ref is a no-op rather than a panic, matching FocusTarget.
func Focus(ref *FocusRef) {
	if ref == nil || ref.ctx == nil {
		return
	}
	ref.ctx.focus.mu.Lock()
	ref.ctx.focus.epoch++
	ref.ctx.focus.target = ref
	ref.ctx.focus.mu.Unlock()

	// RequestRender rather than MarkDirty: a command issued from a timer or a
	// network callback has no bridge call pending to carry the patches back,
	// so it needs the push channel nudged. From an event handler the nudge is
	// redundant but harmless — the coalescing buffer drops it and the pass
	// the dispatch already schedules picks the stamp up.
	ref.ctx.RequestRender()
}

// DismissKeyboard releases the input focus, putting the software keyboard
// away, as of the next render pass.
//
// This is the other half of the old backlog item OnFocus/OnBlur opened: a tap
// on the background of a form can now actually close the keyboard.
//
//	core.Box(
//	    core.OnClick(func() { core.DismissKeyboard(ctx) }),
//	    form,
//	)
//
// It takes a Context where Focus does not because a dismiss names no node,
// and therefore has no ref to carry one. Reaching for a package-level global
// instead would recreate exactly the bug Context's shared-pointer block
// documents: two apps in one process sharing one keyboard.
//
// Unlike Focus this reaches every focusable leaf, because the field the user
// tapped into is one Go was never told about — the framework does not wire
// OnFocus unless an app asks for it. Each leaf stamps "blur" and each
// renderer releases focus only if that leaf actually holds it, so exactly one
// of them does anything.
func DismissKeyboard(ctx *Context) {
	if ctx == nil {
		return
	}
	ctx.focus.mu.Lock()
	ctx.focus.epoch++
	ctx.focus.target = nil
	ctx.focus.mu.Unlock()
	ctx.RequestRender()
}

// focusableLeafTypes are the node types a mobile platform gives keyboard
// focus to, and therefore the only ones leafNode stamps.
//
// Checkbox is deliberately absent even though leafNode builds it: neither
// native renderer gives a checkbox keyboard focus, so stamping it would emit
// a patch per command that nothing on the far side reads. Button is absent
// for the same reason — a phone does not focus a button — despite Button now
// going through leafNode too.
//
// An app that really wants the stamp on one of those can still apply
// FocusTarget explicitly; this set only decides what is stamped by default.
var focusableLeafTypes = map[string]bool{
	"Input":         true,
	"InputPassword": true,
	"NumericInput":  true,
	"TextArea":      true,
}
