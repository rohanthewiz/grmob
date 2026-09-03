package core

import "sync"

// App lifecycle: the host telling the app whether it is on screen.
//
// A phone app spends most of its life not being looked at, and the platform
// is free to freeze its process while it is not. Anything holding a network
// connection finds out the hard way — the socket is dead by the time the
// user comes back, and the first write after foregrounding is what reports
// it. The lifecycle event lets the app act on the transition itself
// (reconnect on resume, pause a poll in the background) instead of waiting
// to trip over its consequences.
//
// # The three states
//
//	active      on screen and receiving input
//	inactive    on screen but not receiving input: the app switcher, an
//	            incoming call, the system asking for a permission
//	background  not on screen; the process may be suspended at any moment
//
// The vocabulary is SwiftUI's ScenePhase, which is the most finely divided
// of the three hosts. Android maps its process lifecycle onto it
// (ON_RESUME → active, ON_PAUSE → inactive, ON_STOP → background) and the
// browser reports only the two states the Page Visibility API can tell
// apart (visible → active, hidden → background), so an app that wants one
// rule should test for active and treat everything else as "not now".
//
// # How it arrives
//
// It is a host event (see host_events.go) named "lifecycle" with a single
// "state" key, and this file is core's consumer for it — the same shape as
// audio status: the event lands in ReceiveHostEvent, core records it here,
// then app subscribers to the raw event run. Apps do not read the raw event;
// they call CurrentLifecycle, subscribe with OnLifecycle, or use
// hooks.UseLifecycle from inside a component.
//
// # Transitions, not ticks
//
// Subscribers hear changes only. The hosts report every platform callback
// they get, and some of those repeat the current state (Android's process
// observer fires ON_START and then ON_RESUME; a browser tab can fire
// visibilitychange twice for one switch), so the record dedupes before it
// notifies. A subscriber therefore never has to compare against what it
// last saw.
//
// The initial state is active. An app that has just started is on screen,
// and a host that disagrees says so with its first report; nothing waits on
// that report.

// LifecycleState is where the app sits in the platform's foreground /
// background lifecycle. See the package comment above for the three values.
type LifecycleState string

const (
	LifecycleActive     LifecycleState = "active"
	LifecycleInactive   LifecycleState = "inactive"
	LifecycleBackground LifecycleState = "background"
)

// hostEventLifecycle is the host event core consumes into the record below;
// every shell reports it under this name, with the state under "state".
// Held to the shells by mobile/verify's TestLifecycleEventSpellingsAgree.
const hostEventLifecycle = "lifecycle"

var (
	lifecycleMu    sync.RWMutex
	lifecycleState = LifecycleActive
	lifecycleSubs  = map[int]func(LifecycleState){}
	lifecycleNext  int
)

// CurrentLifecycle reports the last state the host announced; active until
// it has announced anything.
func CurrentLifecycle() LifecycleState {
	lifecycleMu.RLock()
	defer lifecycleMu.RUnlock()
	return lifecycleState
}

// OnLifecycle subscribes fn to lifecycle transitions. The returned function
// cancels the subscription; calling it more than once is harmless.
//
// Like OnAudioStatus and OnHostEvent, the subscription is process-wide —
// there is one app and one screen, so there is no context tree to scope
// it to. A component that subscribes during render must guard against
// subscribing again on the next pass; hooks.UseLifecycle does that.
//
// fn runs on whichever goroutine delivered the event (see the threading
// note in host_events.go) and must not block. Writing State and calling
// RequestRender are fine from there.
func OnLifecycle(fn func(LifecycleState)) (cancel func()) {
	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()
	id := lifecycleNext
	lifecycleNext++
	lifecycleSubs[id] = fn
	return func() {
		lifecycleMu.Lock()
		defer lifecycleMu.Unlock()
		delete(lifecycleSubs, id)
	}
}

// ReceiveLifecycle is the typed entry point for a host that reports in Go
// (a test, an embedder). The JSON hosts arrive through
// ReceiveHostEvent("lifecycle", ...), which decodes into this.
//
// A state that is not one of the three is dropped rather than stored: a
// newer shell reporting a fourth state to an older app must not leave
// CurrentLifecycle answering something no switch in that app has an arm
// for. A repeat of the current state is absorbed silently (see the file
// comment). Subscribers are notified outside the lock, so one may read
// CurrentLifecycle, subscribe or cancel from inside its handler.
func ReceiveLifecycle(s LifecycleState) {
	switch s {
	case LifecycleActive, LifecycleInactive, LifecycleBackground:
	default:
		return
	}
	lifecycleMu.Lock()
	if s == lifecycleState {
		lifecycleMu.Unlock()
		return
	}
	lifecycleState = s
	fns := make([]func(LifecycleState), 0, len(lifecycleSubs))
	for _, fn := range lifecycleSubs {
		fns = append(fns, fn)
	}
	lifecycleMu.Unlock()

	for _, fn := range fns {
		fn(s)
	}
}

// receiveLifecycle decodes the "lifecycle" host event. The payload is one
// key, the contract every host writes:
//
//	state  string  "active" | "inactive" | "background"
func receiveLifecycle(data map[string]any) {
	if st, ok := data["state"].(string); ok {
		ReceiveLifecycle(LifecycleState(st))
	}
}

// resetLifecycleForTest returns the record and subscriptions to their
// initial state; tests call it so one test's transitions cannot leak into
// the next.
func resetLifecycleForTest() {
	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()
	lifecycleState = LifecycleActive
	lifecycleSubs = map[int]func(LifecycleState){}
}
