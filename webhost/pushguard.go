package webhost

import "log"

// PushGuard stops a browser host's bindings from re-entering the render lock
// while the push listener is delivering a batch.
//
// # The deadlock it stops
//
// render.Manager holds its mutex across PatchListener.ApplyPatches, so that a
// dispatch cannot slip a newer pass in front of an older one (renderAndPush
// says why). On the natives ApplyPatches hops to the UI thread and returns. On
// js/wasm there is one thread and nowhere to hop: the listener calls the page's
// GrMobApplyPatches, which applies the batch before it returns, and anything
// the batch does that calls back into Go runs while the pump still holds the
// lock:
//
//	pump: lock ─ ApplyPatches ─ GrMobApplyPatches ─ applyBatch
//	        └ runCodeCommand ─ emitCodeChange ─ GoInvokeCallback
//	            └ ReceiveEvent ─ RenderAgain ─ lock   ✗ held above, same thread
//
// Go then finds every goroutine asleep and aborts with "all goroutines are
// asleep - deadlock!", and the page stays on screen, dead (N-131: CodeEditor
// "Indent" in lesson 4.13, RichTextEditor "Bold" in 4.14).
//
// # Two layers
//
// The runtime's dispatchToGo (wasm/grmob-runtime.js) is the first: it queues a
// call made while a batch is applying and sends it once Go has returned. This
// is the second, for a call that gets past the first anyway (a runtime site
// that bypasses dispatchToGo, a page script of the app's own). Such a call is
// deferred until the delivery is over instead of blocking, and logged, because
// it means the first layer has a hole somebody should close.
//
// # Why a plain bool
//
// js/wasm runs every goroutine on the one JS thread, and the only goroutines
// that touch a guard are the pump (inside Deliver) and the js.Func handlers JS
// calls back into while Deliver is blocked in JS. They never run at the same
// time, so a mutex would guard nothing. This type is not for the natives,
// where those calls come from another thread and blocking on the lock is
// exactly right.
//
// It has no build constraint, unlike host.go, so that `go test` on a
// development machine reaches it.
type PushGuard struct {
	delivering bool
	deferred   []func()

	// spawn starts the deferred work once a delivery ends. A goroutine in
	// production (see Deliver); tests replace it to run the work inline.
	spawn func(func())
}

// Deliver runs push, the call into the page's GrMobApplyPatches, with the guard
// up, then starts whatever was deferred while it was up.
//
// The deferred work runs on a goroutine of its own and not inline after push.
// Deliver is called from inside the pump's lock hold, and a deferred event's
// handler can call back into JS that re-enters Go (a system event the page
// answers on the spot), which would find the lock still held. A goroutine
// started here runs once the pump has released the lock and parked: js/wasm
// schedules it before control returns to JS, so it still runs ahead of the next
// thing the user does. One goroutine for the lot, so the calls keep the order
// they arrived in.
func (g *PushGuard) Deliver(push func()) {
	g.delivering = true
	defer func() {
		g.delivering = false
		if len(g.deferred) == 0 {
			return
		}
		work := g.deferred
		g.deferred = nil
		run := func() {
			for _, fn := range work {
				fn()
			}
		}
		if g.spawn != nil {
			g.spawn(run)
			return
		}
		go run()
	}()
	push()
}

// Delivering reports whether a delivery is in progress, which is when a
// binding must not take the render lock.
func (g *PushGuard) Delivering() bool {
	return g.delivering
}

// Defer queues fn for the end of the current delivery and reports true, or
// reports false and does nothing when no delivery is in progress, in which
// case the caller runs fn itself. what names the call in the log line.
func (g *PushGuard) Defer(what string, fn func()) bool {
	if !g.delivering {
		return false
	}
	log.Printf("grmob: %s re-entered Go while a pushed batch was being applied; "+
		"deferred to the end of the batch instead of deadlocking on the render lock. "+
		"A runtime call site is bypassing dispatchToGo (wasm/grmob-runtime.js).", what)
	g.deferred = append(g.deferred, fn)
	return true
}
