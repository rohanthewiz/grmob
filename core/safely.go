package core

import (
	"fmt"
	"log"
)

// Safely runs fn on the calling goroutine and recovers a panic escaping it.
// It logs the panic with its stack and, under debug mode, also reports it as
// a ConcernHandlerPanic naming where. It reports whether fn returned normally.
//
// # Why the hooks need it
//
// Some of the framework's work runs on goroutines of its own, off the render
// path: a UseEffect body, a UseInterval tick, a UseTimeout or
// UseTimeoutWhile fire, a Debouncer call, an alarm's OnRing. Go gives a panic
// on such a goroutine no catcher. An unrecovered panic on *any* goroutine
// ends the whole process, so on a phone the app vanishes, and in the browser
// the wasm program exits and the page stops answering. The render driver's
// guards cannot help. Guard wraps a pass and guardHandler wraps a dispatched
// handler, and neither is on these goroutines' stacks.
//
//	goroutine             what runs there          guarded by
//	────────────────────  ───────────────────────  ──────────────────────
//	host's event call     Dispatch* → handler      render's guardHandler
//	render pass           Render, ErrorBoundary    Guard / ErrorBoundary
//	hook's own goroutine  effect, tick, fire       Safely (here)
//
// # What recovery means
//
// The same partial thing it means for a recovered event handler: fn's work
// is abandoned wherever it stopped, and whatever state it had already written
// stays written. The caller carries on. An interval's next tick still fires,
// and a Debouncer accepts the next Call. Each hook still requests the render
// it would have requested after a normal return, so the screen shows the
// state that actually exists.
//
// It logs, where ErrorBoundary deliberately does not, for render's
// logRenderPanic reason: nothing in the app owns this panic, and recovering
// it silently would turn a loud crash into a timer that quietly does
// nothing.
//
// # For app code
//
// It is exported so an app can give its own goroutines the same treatment:
//
//	go core.Safely("load profile", func() {
//	    p, err := api.Profile(id)
//	    ...
//	})
//
// A nil fn is reported as a panic like any other, since calling it is one.
func Safely(where string, fn func()) (ok bool) {
	rerr := Guard(fn)
	if rerr == nil {
		return true
	}
	// Same shape as render's logRenderPanic ("recovered panic during …"),
	// so one grep finds every recovered panic whichever guard caught it.
	log.Printf("grmob: recovered panic during %s: %v\n%s", where, rerr.Value, rerr.Stack)
	if IsDebugMode() {
		ReportConcern(ConcernHandlerPanic, fmt.Sprintf("%s panicked: %v", where, rerr.Value))
	}
	return false
}
