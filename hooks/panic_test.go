package hooks_test

import (
	"io"
	"log"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

// These pin plan item R7 (N-135): a callback a hook runs on its own goroutine
// is recovered rather than ending the process. Without the guard each test
// here does not fail, it crashes the test binary, since an unrecovered panic
// on any goroutine is fatal. That is the same outcome a phone or a browser
// tab had.

// quietPanics silences the recovered panics' log lines (each carries a stack)
// and turns debug mode on so each recovery is also a concern a test can wait
// for. Both are process-wide, and no hooks test runs in parallel.
func quietPanics(t *testing.T) {
	t.Helper()
	prev := log.Writer()
	log.SetOutput(io.Discard)
	core.ClearConcerns()
	core.SetDebugMode(true)
	t.Cleanup(func() {
		core.SetDebugMode(false)
		core.ClearConcerns()
		log.SetOutput(prev)
	})
}

// awaitConcern polls until a handler-panic concern whose detail starts with
// prefix has been recorded. Polling, because the panic happens on a goroutine
// the test does not own.
func awaitConcern(t *testing.T, prefix string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, c := range core.Concerns() {
			if c.Kind == core.ConcernHandlerPanic && strings.HasPrefix(c.Detail, prefix) {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("no handler-panic concern starting %q; have %v", prefix, core.Concerns())
}

// The plan's pin: a panicking tick costs that tick only, and the next still
// fires.
func TestAPanickingIntervalTickDoesNotStopTheNextTick(t *testing.T) {
	quietPanics(t)
	ctx := core.NewContext()
	defer ctx.Close()

	var n atomic.Int32
	ticks := make(chan int32, 64)
	renderPass(ctx, func(ctx *core.Context) {
		hooks.UseInterval(ctx, func() {
			k := n.Add(1)
			if k == 1 {
				panic("first tick fails")
			}
			ticks <- k
		}, 10*time.Millisecond)
	})

	if got := awaitSignal(t, ticks, "a tick after the panicking one"); got != 2 {
		t.Fatalf("the tick after the panic was #%d, want #2", got)
	}
	awaitConcern(t, "UseInterval tick panicked: first tick fails")
}

// The tick that panicked still requests its render: it may have written
// state before it stopped, and the screen should show what exists.
func TestAPanickingIntervalTickStillRequestsARender(t *testing.T) {
	quietPanics(t)
	ctx := core.NewContext()
	defer ctx.Close()

	renders := make(chan struct{}, 64)
	ctx.OnStateChange(func() {
		select {
		case renders <- struct{}{}:
		default:
		}
	})
	renderPass(ctx, func(ctx *core.Context) {
		hooks.UseInterval(ctx, func() { panic("every tick fails") }, 10*time.Millisecond)
	})
	awaitSignal(t, renders, "a render requested by a panicking tick")
}

// An effect that panics is recovered, and the hook still runs the effect
// again when its deps change.
func TestAPanickingEffectIsRecoveredAndTheNextRuns(t *testing.T) {
	quietPanics(t)
	ctx := core.NewContext()
	runs := make(chan int, 8)

	render := func(dep int) {
		renderPass(ctx, func(ctx *core.Context) {
			hooks.UseEffect(ctx, func() {
				if dep == 1 {
					panic("mount effect fails")
				}
				runs <- dep
			}, dep)
		})
	}

	render(1)
	awaitConcern(t, "UseEffect panicked: mount effect fails")
	render(2)
	if got := awaitSignal(t, runs, "the effect after a deps change"); got != 2 {
		t.Fatalf("effect ran with dep %d, want 2", got)
	}
}

func TestAPanickingTimeoutIsRecovered(t *testing.T) {
	quietPanics(t)
	ctx := core.NewContext()
	defer ctx.Close()

	renderPass(ctx, func(ctx *core.Context) {
		hooks.UseTimeout(ctx, func() { panic("timeout fails") }, 5*time.Millisecond)
	})
	awaitConcern(t, "UseTimeout panicked: timeout fails")
}

// A panicking UseTimeoutWhile fire leaves the hook able to arm again on the
// next activation.
func TestAPanickingTimeoutWhileArmsAgainOnTheNextActivation(t *testing.T) {
	quietPanics(t)
	ctx := core.NewContext()
	defer ctx.Close()

	var n atomic.Int32
	fired := make(chan int32, 8)
	pass := func(active bool) {
		renderPass(ctx, func(ctx *core.Context) {
			hooks.UseTimeoutWhile(ctx, active, func() {
				k := n.Add(1)
				if k == 1 {
					panic("first fire fails")
				}
				fired <- k
			}, 5*time.Millisecond)
		})
	}

	pass(true)
	awaitConcern(t, "UseTimeoutWhile panicked: first fire fails")
	pass(false)
	pass(true)
	if got := awaitSignal(t, fired, "the fire of the second activation"); got != 2 {
		t.Fatalf("second activation fired #%d, want #2", got)
	}
}

// A panicking debounced call is recovered, and the Debouncer takes the next
// Call as usual.
func TestAPanickingDebouncedCallLeavesTheDebouncerWorking(t *testing.T) {
	quietPanics(t)
	ctx := core.NewContext()
	defer ctx.Close()

	var d *hooks.Debouncer
	renderPass(ctx, func(ctx *core.Context) {
		d = hooks.UseDebounce(ctx, 5*time.Millisecond)
	})

	d.Call(func() { panic("search fails") })
	awaitConcern(t, "Debouncer call panicked: search fails")

	fired := make(chan struct{}, 1)
	d.Call(func() { fired <- struct{}{} })
	awaitSignal(t, fired, "the debounced call after the panicking one")
}
