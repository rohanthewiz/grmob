package permission

import (
	"sync"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// resume drives one full foreground transition.
//
// Two calls rather than one because core.OnLifecycle notifies on *changes*:
// the record starts at active, so an app that never left it has nothing to
// come back from. Going away first is what makes the return a transition —
// which is exactly the sequence a real resume produces.
func resume() {
	core.ReceiveLifecycle(core.LifecycleBackground)
	core.ReceiveLifecycle(core.LifecycleActive)
}

// kinds pulls the "kind" out of every permission command the recorder saw, in
// order, so a test can say which permissions were asked about without caring
// how the payload is spelled.
func kinds(evs []event) []string {
	out := make([]string, 0, len(evs))
	for _, e := range evs {
		if e.name != systemEventName {
			continue
		}
		if k, ok := e.data["kind"].(string); ok {
			out = append(out, k)
		}
	}
	return out
}

func TestWatchForegroundChecksOnTheWayBackIn(t *testing.T) {
	r := withHost(t)
	defer WatchForeground(Camera)()

	resume()

	if got := kinds(r.all()); len(got) != 1 || got[0] != string(Camera) {
		t.Fatalf("commands after a resume = %v, want one check of %q", got, Camera)
	}
	if cmd, _ := r.all()[0].data["command"].(string); cmd != commandCheck {
		t.Errorf("command = %q, want %q — a resume must never prompt: the user did "+
			"nothing to ask for a dialog, and a dialog they did not expect is the "+
			"one they refuse", cmd, commandCheck)
	}
}

// The count is per permission, not per watcher — the whole reason this is not
// a lifecycle subscription inside hooks.UsePermission.
func TestFiveWatchersOfOneKindAreOneCheck(t *testing.T) {
	r := withHost(t)
	for i := 0; i < 5; i++ {
		defer WatchForeground(Camera)()
	}

	resume()

	if got := kinds(r.all()); len(got) != 1 {
		t.Fatalf("commands = %v, want exactly one — five screens asking the same "+
			"question of one device is still one question", got)
	}
}

func TestEachWatchedKindIsCheckedOnce(t *testing.T) {
	r := withHost(t)
	defer WatchForeground(Camera)()
	defer WatchForeground(Location)()
	defer WatchForeground(Camera)()

	resume()

	got := kinds(r.all())
	want := []string{string(Camera), string(Location)}
	if len(got) != len(want) {
		t.Fatalf("commands = %v, want %v — one check per watched kind, "+
			"whatever the watcher count", got, want)
	}
}

// The events leave in the order Permissions() lists them, so a shell's log and
// a test's expectations read the same way twice running.
//
// Asserted over repeated resumes rather than one, because Go randomizes map
// iteration per range and a three-key map lands in declaration order often
// enough that a single pass would let the map version through half the time.
// Six identical batches is the difference between "it happened to come out
// right" and "it comes out right".
func TestTheResumeOrderIsDeclarationOrder(t *testing.T) {
	r := withHost(t)
	defer WatchForeground(Microphone)()
	defer WatchForeground(Camera)()
	defer WatchForeground(Storage)()

	const rounds = 6
	for i := 0; i < rounds; i++ {
		resume()
	}

	got := kinds(r.all())
	one := []string{string(Camera), string(Storage), string(Microphone)}
	if len(got) != rounds*len(one) {
		t.Fatalf("commands = %v, want %d rounds of %v", got, rounds, one)
	}
	for i, k := range got {
		if want := one[i%len(one)]; k != want {
			t.Fatalf("commands = %v\n  round %d position %d is %q, want %q — "+
				"the resume loop is iterating the map rather than Permissions()",
				got, i/len(one), i%len(one), k, want)
		}
	}
}

// The failure a plain on/off flag makes easy: two screens watching, one of
// them closing, and the re-check quietly off under the other.
func TestOneCancelDoesNotStopTheOtherWatcher(t *testing.T) {
	r := withHost(t)
	first := WatchForeground(Camera)
	defer WatchForeground(Camera)()

	first()
	resume()

	if got := kinds(r.all()); len(got) != 1 {
		t.Fatalf("commands = %v, want one — the second watcher is still holding "+
			"the watch open", got)
	}
}

func TestTheLastCancelStopsTheChecks(t *testing.T) {
	r := withHost(t)
	cancel := WatchForeground(Camera)
	cancel()

	resume()

	if got := kinds(r.all()); len(got) != 0 {
		t.Fatalf("commands = %v, want none once nothing is watched", got)
	}
	if n := WatchedForeground(Camera); n != 0 {
		t.Errorf("watchers = %d, want 0", n)
	}
}

// A second cancel is a no-op rather than a decrement. A component that both
// defers its cancel and calls it on an error path must not be able to release
// a watch it does not hold.
func TestASecondCancelIsHarmless(t *testing.T) {
	r := withHost(t)
	cancel := WatchForeground(Camera)
	other := WatchForeground(Camera)
	defer other()

	cancel()
	cancel()
	cancel()

	resume()

	if got := kinds(r.all()); len(got) != 1 {
		t.Fatalf("commands = %v, want one — the surviving watcher's watch was "+
			"released by somebody else's repeated cancel", got)
	}
}

func TestOnlyTheReturnToActiveChecks(t *testing.T) {
	r := withHost(t)
	defer WatchForeground(Camera)()

	// The system permission dialog itself puts an iOS app into inactive, and
	// a check fired there would race the answer the request path is about to
	// deliver.
	core.ReceiveLifecycle(core.LifecycleInactive)
	core.ReceiveLifecycle(core.LifecycleBackground)

	if got := kinds(r.all()); len(got) != 0 {
		t.Fatalf("commands = %v, want none — only the way back in re-checks", got)
	}

	core.ReceiveLifecycle(core.LifecycleActive)
	if got := kinds(r.all()); len(got) != 1 {
		t.Fatalf("commands = %v, want one on the return to active", got)
	}
}

// An app that never asks for a live permission must be exactly as it was: no
// subscription, so no work on any transition.
func TestNoWatchersMeansNoLifecycleSubscription(t *testing.T) {
	withHost(t)

	fgMu.Lock()
	subscribed := fgCancel != nil
	fgMu.Unlock()
	if subscribed {
		t.Fatal("a lifecycle subscription exists with nothing watched")
	}

	cancel := WatchForeground(Camera)
	fgMu.Lock()
	subscribed = fgCancel != nil
	fgMu.Unlock()
	if !subscribed {
		t.Fatal("no lifecycle subscription after the first watch")
	}

	cancel()
	fgMu.Lock()
	subscribed = fgCancel != nil
	fgMu.Unlock()
	if subscribed {
		t.Error("the lifecycle subscription outlived the last watcher")
	}
}

// An unspelled permission has no host arm, so counting it would put an event
// on the wire that no shell answers. The cancel is still a real function, so
// a caller's defer needs no nil check.
func TestAnUnknownPermissionIsNotWatched(t *testing.T) {
	r := withHost(t)
	cancel := WatchForeground(Permission("bluetooth"))
	defer cancel()

	resume()

	if got := kinds(r.all()); len(got) != 0 {
		t.Fatalf("commands = %v, want none for an unspelled permission", got)
	}
	if n := WatchedForeground(Permission("bluetooth")); n != 0 {
		t.Errorf("watchers = %d, want 0", n)
	}
}

// A resume that changed nothing must reach the record and stop: set notifies
// only on a change, which is what keeps a watched screen from re-rendering
// every time the user switches apps and back.
func TestAResumeThatChangesNothingNotifiesNobody(t *testing.T) {
	withHost(t)
	defer WatchForeground(Camera)()

	Receive(Camera, Granted)

	var notified int
	defer On(func(Permission, Status) { notified++ })()

	resume()
	// The host answers the check with what it already said.
	Receive(Camera, Granted)

	if notified != 0 {
		t.Errorf("subscribers ran %d times for news that is not news", notified)
	}
	if got := Current(Camera); got != Granted {
		t.Errorf("status = %q, want %q", got, Granted)
	}
}

// The whole point, end to end: the user grants it in Settings and comes back.
func TestAGrantMadeInSettingsReachesTheScreen(t *testing.T) {
	r := withHost(t)
	defer WatchForeground(Camera)()

	Receive(Camera, Denied)

	answers := make(chan Status, 4)
	defer On(func(p Permission, s Status) {
		if p == Camera {
			answers <- s
		}
	})()

	resume()
	if got := kinds(r.all()); len(got) != 1 {
		t.Fatalf("commands = %v, want one check on resume", got)
	}
	// The host's answer to that check, now that the switch has been flipped.
	Receive(Camera, Granted)

	select {
	case got := <-answers:
		if got != Granted {
			t.Errorf("subscriber saw %q, want %q", got, Granted)
		}
	default:
		t.Fatal("nothing was notified — the screen is still drawing the denied state")
	}
}

// Registering and cancelling from several goroutines at once must not lose
// the subscription or leave one unreferenced. Run under -race, which is where
// a missing lock shows up.
func TestConcurrentWatchesSettleAtZero(t *testing.T) {
	withHost(t)

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cancel := WatchForeground(Camera)
			resume()
			cancel()
		}()
	}
	wg.Wait()

	if n := WatchedForeground(Camera); n != 0 {
		t.Errorf("watchers = %d, want 0", n)
	}
	fgMu.Lock()
	subscribed := fgCancel != nil
	fgMu.Unlock()
	if subscribed {
		t.Error("a lifecycle subscription survived every cancel")
	}
}
