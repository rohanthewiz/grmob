package core

import "testing"

// The record starts active, follows the host, and tells subscribers about
// transitions only.
func TestLifecycleRecordsTransitionsAndDedupes(t *testing.T) {
	t.Cleanup(resetLifecycleForTest)
	if got := CurrentLifecycle(); got != LifecycleActive {
		t.Fatalf("initial state = %q, want active", got)
	}

	var seen []LifecycleState
	cancel := OnLifecycle(func(s LifecycleState) { seen = append(seen, s) })
	defer cancel()

	ReceiveLifecycle(LifecycleActive) // a repeat of the current state: nothing
	ReceiveLifecycle(LifecycleBackground)
	ReceiveLifecycle(LifecycleBackground) // the browser's double visibilitychange
	ReceiveLifecycle(LifecycleActive)
	if len(seen) != 2 || seen[0] != LifecycleBackground || seen[1] != LifecycleActive {
		t.Errorf("subscriber saw %v, want [background active]", seen)
	}
	if got := CurrentLifecycle(); got != LifecycleActive {
		t.Errorf("state after the round trip = %q", got)
	}

	cancel()
	cancel() // idempotent
	ReceiveLifecycle(LifecycleInactive)
	if len(seen) != 2 {
		t.Errorf("a cancelled subscriber still heard %v", seen)
	}
}

// A state the app does not know is dropped, not stored: CurrentLifecycle
// must only ever answer one of the three values.
func TestLifecycleIgnoresUnknownStates(t *testing.T) {
	t.Cleanup(resetLifecycleForTest)
	calls := 0
	defer OnLifecycle(func(LifecycleState) { calls++ })()

	ReceiveLifecycle("suspended")
	ReceiveLifecycle("")
	if got := CurrentLifecycle(); got != LifecycleActive || calls != 0 {
		t.Errorf("unknown state was recorded: state=%q calls=%d", got, calls)
	}
}

// The host event path decodes into the record before app subscribers to
// the raw event run, the same ordering audio status has.
func TestLifecycleHostEventFeedsTheRecordFirst(t *testing.T) {
	t.Cleanup(resetLifecycleForTest)
	var seenRaw LifecycleState
	cancel := OnHostEvent(hostEventLifecycle, func(map[string]any) {
		seenRaw = CurrentLifecycle()
	})
	defer cancel()

	ReceiveHostEvent(hostEventLifecycle, map[string]any{"state": "background"})
	if seenRaw != LifecycleBackground {
		t.Errorf("raw subscriber saw %q, want background (core consumer first)", seenRaw)
	}

	// A payload without the key, or with the wrong type, changes nothing.
	ReceiveHostEvent(hostEventLifecycle, map[string]any{"state": 3})
	ReceiveHostEvent(hostEventLifecycle, nil)
	if got := CurrentLifecycle(); got != LifecycleBackground {
		t.Errorf("a malformed payload moved the state to %q", got)
	}
}

// Cancelling from inside a handler is the "once" pattern and must not
// deadlock on the registry lock.
func TestLifecycleSubscriberMayCancelItself(t *testing.T) {
	t.Cleanup(resetLifecycleForTest)
	calls := 0
	var cancel func()
	cancel = OnLifecycle(func(LifecycleState) {
		calls++
		cancel()
	})
	ReceiveLifecycle(LifecycleBackground)
	ReceiveLifecycle(LifecycleActive)
	if calls != 1 {
		t.Errorf("handler ran %d times, want 1", calls)
	}
}
