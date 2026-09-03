package hooks

import (
	"sync"

	"github.com/rohanthewiz/grmob/core"
)

// lifecycleRecord is the per-hook-slot memory of UseLifecycle: whether this
// slot's subscription is live. Same shape and same reasons as audioRecord —
// identity by slot position, nothing package-level to reset, a mutex for the
// one write that happens off the render goroutine (the close path).
type lifecycleRecord struct {
	mu         sync.Mutex
	subscribed bool
}

// UseLifecycle returns whether the app is on screen and re-renders the app
// on every transition. It is how a screen reacts to being foregrounded —
// the reconnect-on-resume case that put the event on the roadmap:
//
//	state := hooks.UseLifecycle(ctx)
//	hooks.UseEffect(ctx, func() { if state == core.LifecycleActive { conn.Resume() } }, state)
//
// A component that only needs to *act* on the transition, not re-render
// for it, can subscribe with core.OnLifecycle from wherever it owns the
// connection instead; this hook is for the tree.
//
// The subscription is taken on the hook's first render and released when
// the context tree is closed (ctx.Close, normally via
// render.Manager.Close), not when the component leaves the view tree —
// hooks have no unmount signal today, the same limit UseInterval and
// UseAudio carry. The cost of a stale subscription is one redundant
// RequestRender per transition, coalesced by the manager into a pass that
// diffs to nothing.
//
// The state itself is not copied into a hook slot: core keeps one record
// for the process (core.CurrentLifecycle), and reading it at render time is
// what makes every subscriber see the same transition. The slot only
// remembers that this component already subscribed.
func UseLifecycle(ctx *core.Context) core.LifecycleState {
	slot := core.NewState(ctx, &lifecycleRecord{})
	rec := slot.Get()

	rec.mu.Lock()
	already := rec.subscribed
	rec.subscribed = true
	rec.mu.Unlock()

	if !already {
		cancel := core.OnLifecycle(func(core.LifecycleState) {
			// RequestRender rather than MarkDirty: the transition arrives
			// from the host with no native event in flight, exactly the
			// async source the push channel exists for.
			ctx.RequestRender()
		})
		ctx.OnClose(func() {
			cancel()
			// After a Close the tree stays renderable and a re-mount must
			// be able to subscribe again, so the slot forgets that it did.
			rec.mu.Lock()
			rec.subscribed = false
			rec.mu.Unlock()
		})
	}
	return core.CurrentLifecycle()
}
