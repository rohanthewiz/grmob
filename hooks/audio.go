package hooks

import (
	"sync"

	"github.com/rohanthewiz/grmob/core"
)

// audioRecord is the per-hook-slot memory of UseAudio: whether this slot's
// subscription is live. It lives in the context's hook-slot array (via
// core.NewState) for the same reasons intervalRecord does — identity by
// slot position, no cross-talk between components at the same cursor, and
// nothing package-level to reset. The mutex covers the one write that
// happens off the render goroutine: the close path clearing subscribed.
type audioRecord struct {
	mu         sync.Mutex
	subscribed bool
}

// UseAudio returns the player's current status and re-renders the app on
// every change to it — a position tick, a pause, an error. It is how a
// screen with transport controls stays live:
//
//	status := hooks.UseAudio(ctx)
//	label := "Play"
//	if status.State == core.AudioPlaying { label = "Pause" }
//	core.Button(label, core.AudioToggle)
//
// The subscription is taken on the hook's first render and released when
// the context tree is closed (ctx.Close, normally via render.Manager.Close),
// not when the component leaves the view tree — hooks have no unmount
// signal today, the same limit UseInterval carries. The cost of a stale
// subscription is one redundant RequestRender per status tick, coalesced
// by the manager into a pass that diffs to nothing; it is not a leak of
// anything the user can see.
//
// The status itself is not copied into a hook slot: core keeps one record
// for the process (core.CurrentAudioStatus), and reading it at render time
// is what makes every subscriber see the same tick. The slot only remembers
// that this component already subscribed, so re-renders do not stack
// subscriptions.
func UseAudio(ctx *core.Context) core.AudioStatus {
	slot := core.NewState(ctx, &audioRecord{})
	rec := slot.Get()

	rec.mu.Lock()
	already := rec.subscribed
	rec.subscribed = true
	rec.mu.Unlock()

	if !already {
		cancel := core.OnAudioStatus(func(core.AudioStatus) {
			// RequestRender rather than MarkDirty so a tick reaches the
			// screen through the push channel with no native event in
			// flight — a status change is exactly that kind of async
			// source, like a timer.
			ctx.RequestRender()
		})
		ctx.OnClose(func() {
			cancel()
			// Same drain semantics as UseInterval: after a Close the tree
			// stays renderable and a re-mount must be able to subscribe
			// again, so the slot forgets that it did.
			rec.mu.Lock()
			rec.subscribed = false
			rec.mu.Unlock()
		})
	}
	return core.CurrentAudioStatus()
}
