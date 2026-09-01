package core

import (
	"fmt"
	"runtime/debug"
)

// RenderError is a panic that escaped a component's Render and was caught by
// an ErrorBoundary (or by the render driver's top-level guard).
//
// It carries the raw panic value rather than just a message because the panic
// may well be a real error worth inspecting: a *net.OpError, a wrapped
// sentinel, a custom type the app wants to switch on. Unwrap exposes it to
// errors.Is/errors.As when it is one, so
//
//	if errors.Is(err, sql.ErrNoRows) { ... }
//
// works inside a fallback even though the value travelled through panic().
type RenderError struct {
	// Value is exactly what was passed to panic(): usually an error or a
	// string, but it can be any type.
	Value any

	// Stack is debug.Stack() captured inside the deferred recover, so it
	// still contains the frames being unwound — the component that actually
	// panicked is in here, which is the whole point of keeping it. Nil only
	// if a RenderError was constructed by hand.
	Stack []byte
}

// Error renders the panic value as a message. The "panic during render"
// prefix is deliberate: a RenderError frequently ends up in a log line next
// to ordinary application errors, and without it a bare "index out of range"
// gives no hint that it came from a render pass.
func (e *RenderError) Error() string {
	if err, ok := e.Value.(error); ok {
		return "grmob: panic during render: " + err.Error()
	}
	return fmt.Sprintf("grmob: panic during render: %v", e.Value)
}

// Unwrap exposes the panicked value to errors.Is/As when it is an error, and
// returns nil otherwise (a panic("boom") wraps nothing).
func (e *RenderError) Unwrap() error {
	if err, ok := e.Value.(error); ok {
		return err
	}
	return nil
}

// Guard runs fn and converts a panic escaping it into a *RenderError,
// returning nil when fn completes normally.
//
// Exported because the render driver needs the same guard around a whole pass
// that ErrorBoundary needs around a subtree, and render is a separate package.
// It deliberately takes a func() rather than a View: the driver must also
// cover the root-view *construction* call, not only Render.
//
// Guard restores nothing — it is the bare recover. Callers that intend to
// keep rendering after the failure are responsible for repairing whatever the
// half-finished work left behind; see renderRecovered for what that means
// inside a boundary.
func Guard(fn func()) (rerr *RenderError) {
	defer func() {
		if r := recover(); r != nil {
			// debug.Stack() here runs while the panicking frames are still on
			// the goroutine's stack (deferred calls execute before those
			// frames are popped), so the trace names the guilty component.
			rerr = &RenderError{Value: r, Stack: debug.Stack()}
		}
	}()
	fn()
	return nil
}

// ErrorBoundary renders child, and if child's render panics, renders
// fallback(err) in its place instead of letting the panic reach the render
// driver — where, on a native host, it would take the whole app down.
//
// Pass a nil fallback to get DefaultErrorFallback.
//
//	core.ErrorBoundary(
//	    ProfilePanel(user),
//	    func(err error) core.View {
//	        log.Printf("profile panel failed: %v", err)
//	        return core.Text("Profile unavailable")
//	    },
//	)
//
// # The fallback is also the notification hook
//
// ErrorBoundary logs nothing itself. fallback is called on every pass in
// which the child fails, receives the full *RenderError (stack included), and
// is the intended place to log, report, or degrade. Note "every pass": a
// component that panics deterministically panics again next frame, so a
// fallback that logs unconditionally will log at frame rate. Rate-limit, or
// log from a boundary placed high enough that failures are rare.
//
// # It does not latch
//
// React's error boundaries stay in the fallback until explicitly reset,
// because there the failed subtree's instances are unrecoverable. Nothing of
// the sort is true here: the tree is rebuilt from scratch every pass, so a
// child that panicked on a stale slice index simply renders normally on the
// next pass once the state settles. Latching would turn a one-frame glitch
// into a permanent dead panel, so the boundary retries every pass and heals
// on its own. The cost is the repeated-panic case above, which is the right
// trade — a stuck fallback is worse than a noisy one.
//
// # What it repairs, and why it needs its own contexts
//
// A panic partway through a render leaves two pieces of per-pass bookkeeping
// half-advanced, and both are positional, so leaving them where they fell
// would corrupt *unrelated* components rendered later in the same pass:
//
//	hook slots      parent ctx.Cursor sits between the child's hooks, so
//	                every later sibling reads the wrong slots — sibling
//	                state visibly swaps
//	callback IDs    the registry counters sit past the handlers the child
//	                managed to register, so every later sibling's IDs shift
//	                and taps land on the wrong handler
//
// The hook half is solved structurally rather than by rollback: the boundary
// takes two child contexts (one for child, one for the fallback) and renders
// into those. A panic can then only strand a cursor inside the child's own
// context, and the boundary consumes exactly two parent slots whether the
// child succeeds, fails early, or fails late.
//
//	parent ctx slots:   [ ... | childCtx | fallbackCtx | ... ]
//	                             ^ panic strands the cursor in here only
//
// The callback half is a genuine rollback: renderRecovered snapshots the
// registry counters before the child renders and rewinds them after a panic,
// so the boundary's ID footprint equals the fallback's footprint and does not
// depend on how far the failed render got.
//
// # Consequence: the child gets its own hook namespace
//
// Because child renders into a child context, its hook slots and its
// ctx.Scope table are its own rather than the parent's. State is keyed by
// position within a context, so this is transparent for the child itself —
// but a component that reaches for ctx.Scope("x") expecting to share a scope
// with something *outside* the boundary will get a different scope. Shared
// app state (navigation, callbacks, theme, config) lives on pointers copied
// into every derived context and is unaffected.
func ErrorBoundary(child View, fallback func(err error) View) View {
	return ComponentFunc(func(ctx *Context) *Node {
		// Both contexts are claimed unconditionally, before anything can
		// fail, so the boundary's parent-slot footprint is fixed at two.
		childCtx := UseChildContext(ctx)
		fallbackCtx := UseChildContext(ctx)

		node, rerr := renderRecovered(childCtx, child)
		if rerr == nil {
			return node
		}

		if IsDebugMode() {
			upsertConcern(ConcernRenderPanic, fmt.Sprintf(
				"ErrorBoundary caught a panic from %T: %v", child, rerr.Value))
		}

		if fallback == nil {
			fallback = DefaultErrorFallback
		}
		// The fallback is built AND rendered inside the guard: constructing it
		// runs app code too (fallback(err) may format the error, look
		// something up, dereference a nil), so a crash there must not escape
		// the boundary that exists to prevent crashes.
		fbNode, fbErr := renderRecovered(fallbackCtx, ComponentFunc(func(c *Context) *Node {
			return fallback(rerr).Render(c)
		}))
		if fbErr == nil {
			return fbNode
		}

		if IsDebugMode() {
			upsertConcern(ConcernRenderPanic, fmt.Sprintf(
				"ErrorBoundary's fallback itself panicked: %v (original: %v)", fbErr.Value, rerr.Value))
		}
		// Last resort. Built by hand rather than through Text() so that this
		// path cannot itself run a builder: no hooks, no callbacks, no theme
		// lookup, nothing that could panic a third time.
		return &Node{
			Type:  "Text",
			Props: map[string]any{"content": "Something went wrong."},
			Style: &Style{},
		}
	})
}

// SafeRender is ErrorBoundary with the built-in fallback — the one-liner for
// wrapping a subtree you merely want to survive, with no opinion about what
// replaces it.
func SafeRender(child View) View {
	return ErrorBoundary(child, nil)
}

// DefaultErrorFallback is the fallback ErrorBoundary uses when given nil: a
// bordered card in the theme's Error role.
//
// The detail line is gated on debug mode on purpose. A panic message is
// developer-facing text — "runtime error: index out of range [7] with length
// 3" tells a user nothing and quietly leaks internals into a screenshot — so
// a release build shows only the generic line, while a debug build shows the
// message that identifies the bug. The full *RenderError, stack included, is
// always available to a custom fallback either way.
func DefaultErrorFallback(err error) View {
	return ComponentFunc(func(ctx *Context) *Node {
		theme := ctx.Theme()
		items := []PropsAndChildren{
			Gap(4),
			Text("Something went wrong",
				FontWeight(Bold), TextColor(theme.Colors.Error)),
		}
		if IsDebugMode() && err != nil {
			items = append(items, Text(err.Error(),
				FontSize(12), TextColor(theme.Colors.TextSecondary)))
		}
		return Card(
			BackgroundColor(theme.Colors.Surface),
			BorderColor(theme.Colors.Error),
			BorderWidth(1),
			BorderRadius(8),
			Padding(12),
			Column(items...),
		).Render(ctx)
	})
}

// renderRecovered renders view into ctx under a panic guard, and on failure
// undoes the per-pass bookkeeping the aborted render advanced.
//
// Two repairs, both explained at length on ErrorBoundary:
//
//   - Callback counters are rewound to their pre-render values, and the
//     liveness marks for the IDs in between are dropped. Dropping the marks
//     matters as much as rewinding the counters: purge keeps every ID marked
//     used this pass, so without this the handlers of a subtree that is not
//     on screen would survive the pass and stay dispatchable. The map entries
//     themselves are left for purge to collect — that is its job, and
//     deleting them here would duplicate it.
//
//   - ctx.Cursor is zeroed rather than left mid-way. Nothing positional
//     depends on it (the next pass calls Reset), but the debug cursor-drift
//     audit reads it, and a truncated cursor looks exactly like a
//     conditional-hook bug. Zero is the audit's documented "this context
//     rendered nothing this pass" value, which is precisely what happened, so
//     the panic is reported once as a panic instead of twice as a panic plus
//     a phantom drift.
func renderRecovered(ctx *Context, view View) (node *Node, rerr *RenderError) {
	snap := ctx.registry.snapshotCounters()

	rerr = Guard(func() { node = view.Render(ctx) })
	if rerr == nil {
		return node, nil
	}

	ctx.registry.rollbackCounters(snap)
	// hookOwner: a themed copy's own Cursor is never used, so zeroing it
	// would leave the real cursor stranded mid-pass.
	ctx.hookOwner().Cursor = 0
	return nil, rerr
}
