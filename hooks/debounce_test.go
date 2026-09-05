package hooks

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// The delays here are deliberately coarse — a debounce of 30ms observed after
// 200ms — because these tests assert *which* calls happen, not when. A tight
// margin would turn a scheduler hiccup on a loaded CI box into a failure
// about timer semantics, which is not what is being pinned.
const (
	debounceDelay = 30 * time.Millisecond
	debounceWait  = 200 * time.Millisecond
)

func TestDebounceCollapsesABurst(t *testing.T) {
	ctx := core.NewContext()
	d := UseDebounce(ctx, debounceDelay)

	var calls int32
	var last atomic.Int32
	for i := 1; i <= 5; i++ {
		n := int32(i)
		d.Call(func() {
			atomic.AddInt32(&calls, 1)
			last.Store(n)
		})
	}

	time.Sleep(debounceWait)
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("a burst of 5 calls fired %d times, want 1", got)
	}
	// The surviving call must be the *last* one, not the first: a debounced
	// search that ran the query from the first keystroke would be worse than
	// no debounce at all.
	if got := last.Load(); got != 5 {
		t.Errorf("the call that fired was #%d, want the last one (#5)", got)
	}
}

func TestDebounceFiresAgainAfterAQuietPeriod(t *testing.T) {
	ctx := core.NewContext()
	d := UseDebounce(ctx, debounceDelay)

	var calls int32
	fn := func() { atomic.AddInt32(&calls, 1) }

	d.Call(fn)
	time.Sleep(debounceWait)
	d.Call(fn)
	time.Sleep(debounceWait)

	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("two separated calls fired %d times, want 2 — the debouncer must re-arm", got)
	}
}

func TestDebounceCancelDropsThePendingCall(t *testing.T) {
	ctx := core.NewContext()
	d := UseDebounce(ctx, debounceDelay)

	var calls int32
	d.Call(func() { atomic.AddInt32(&calls, 1) })
	if !d.Pending() {
		t.Error("Pending should be true between Call and the fire")
	}
	d.Cancel()
	if d.Pending() {
		t.Error("Pending should be false after Cancel")
	}

	time.Sleep(debounceWait)
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Errorf("a cancelled call fired %d times, want 0", got)
	}
}

// A zero delay runs synchronously, which is what lets a caller turn the
// debounce off from state without growing a second code path.
func TestDebounceZeroDelayRunsInline(t *testing.T) {
	ctx := core.NewContext()
	d := UseDebounce(ctx, 0)

	ran := false
	d.Call(func() { ran = true })
	if !ran {
		t.Error("a zero delay should run fn before Call returns")
	}
	if d.Pending() {
		t.Error("a zero delay leaves nothing pending")
	}
}

// A nil fn is inert and specifically does *not* cancel — reading "schedule
// nothing" as "unschedule everything" would make a guarded Call silently
// destructive.
func TestDebounceNilCallLeavesThePendingCallAlone(t *testing.T) {
	ctx := core.NewContext()
	d := UseDebounce(ctx, debounceDelay)

	var calls int32
	d.Call(func() { atomic.AddInt32(&calls, 1) })
	d.Call(nil)

	time.Sleep(debounceWait)
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("calls after a nil Call = %d, want the pending one to survive (1)", got)
	}
}

// The delay is held on the record rather than baked into a running timer, so
// a re-render with a new duration takes effect on the next Call. This is the
// one place the hook deliberately differs from UseInterval, whose period is
// fixed by the first render.
func TestDebounceDelayIsRefreshedByLaterRenders(t *testing.T) {
	ctx := core.NewContext()

	first := UseDebounce(ctx, time.Hour)

	// Reset, not BeginRenderPass: rewinding the hook cursor is what makes the
	// next call land on the same slot, and it is what a render pass does.
	// BeginRenderPass only cycles the callback registry, so using it here
	// would hand out a second, fresh debouncer and the test would pass for
	// the wrong reason.
	ctx.Reset()
	d := UseDebounce(ctx, debounceDelay)
	if d != first {
		t.Fatal("the second pass should re-bind the same slot, not allocate a new one")
	}

	var calls int32
	d.Call(func() { atomic.AddInt32(&calls, 1) })
	time.Sleep(debounceWait)
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("calls = %d, want 1 — the second render's delay should be the one in force", got)
	}
}

// Two hooks in one component must not share a record. This is the collision
// the slot-keyed rewrite of UseInterval fixed, and it is worth pinning for
// every hook that reserves a slot.
func TestDebounceHooksGetSeparateSlots(t *testing.T) {
	ctx := core.NewContext()
	first := UseDebounce(ctx, debounceDelay)
	second := UseDebounce(ctx, debounceDelay)

	if first == second {
		t.Fatal("two UseDebounce calls in one pass returned the same record")
	}

	var firstCalls, secondCalls int32
	first.Call(func() { atomic.AddInt32(&firstCalls, 1) })
	second.Call(func() { atomic.AddInt32(&secondCalls, 1) })

	time.Sleep(debounceWait)
	if a, b := atomic.LoadInt32(&firstCalls), atomic.LoadInt32(&secondCalls); a != 1 || b != 1 {
		t.Errorf("calls = (%d, %d), want (1, 1) — one debouncer must not cancel the other", a, b)
	}
}

// Closing the context tree drops a pending call, so a Manager shutdown cannot
// land a late callback in a dead app.
func TestDebounceClosingTheContextCancels(t *testing.T) {
	ctx := core.NewContext()
	d := UseDebounce(ctx, debounceDelay)

	var calls int32
	d.Call(func() { atomic.AddInt32(&calls, 1) })
	ctx.Close()

	time.Sleep(debounceWait)
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Errorf("a call pending at Close fired %d times, want 0", got)
	}
}

// Close is a drain, not a terminal state: the tree stays renderable, so a
// re-mounted hook has to register a fresh cleanup rather than take a
// permanent early return. Same contract UseInterval's OnClose documents.
func TestDebounceSurvivesCloseAndRemount(t *testing.T) {
	ctx := core.NewContext()
	first := UseDebounce(ctx, debounceDelay)
	ctx.Close()

	// See the note in TestDebounceDelayIsRefreshedByLaterRenders: Reset is
	// what re-binds the slot, and re-binding it is the whole point here — the
	// record survives the Close, so the remounted hook is the same one.
	ctx.Reset()
	d := UseDebounce(ctx, debounceDelay)
	if d != first {
		t.Fatal("the remount should re-bind the same record, not allocate a new one")
	}

	var calls int32
	d.Call(func() { atomic.AddInt32(&calls, 1) })
	time.Sleep(debounceWait)
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("calls after remount = %d, want 1", got)
	}

	// And the fresh cleanup must be wired, or the second lifetime leaks.
	d.Call(func() { atomic.AddInt32(&calls, 1) })
	ctx.Close()
	time.Sleep(debounceWait)
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("calls after the second Close = %d, want 1 — the remount did not re-register cleanup", got)
	}
}

// Call arrives from native event threads, and the timer fires on its own; the
// race detector is what this test is for.
func TestDebounceConcurrentCallsAreSafe(t *testing.T) {
	ctx := core.NewContext()
	d := UseDebounce(ctx, time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d.Call(func() {})
			d.Pending()
		}()
	}
	wg.Wait()
	time.Sleep(debounceWait)
	d.Cancel()
}
