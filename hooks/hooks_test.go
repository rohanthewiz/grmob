package hooks_test

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

// awaitSignal waits for one value on ch or fails the test. The generous
// timeout only bounds a hang; passing runs take a few milliseconds.
func awaitSignal[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
		panic("unreachable")
	}
}

// assertQuiet asserts nothing arrives on ch for a settle window. The window
// must be long enough that a wrongly-live producer would almost surely emit
// into it (producers in these tests run at 5–20ms periods).
func assertQuiet[T any](t *testing.T, ch <-chan T, window time.Duration, what string) {
	t.Helper()
	select {
	case <-ch:
		t.Fatalf("unexpected %s", what)
	case <-time.After(window):
	}
}

// renderPass simulates what render.Manager does around a render: reset the
// hook cursor, then run the component body. The hooks under test only need
// the cursor discipline, not a full manager.
func renderPass(ctx *core.Context, body func(ctx *core.Context)) {
	ctx.Reset()
	body(ctx)
}

func TestUseEffectRunsOnMountAndOnDepsChangeOnly(t *testing.T) {
	// The old implementation indexed effects with a global counter that never
	// reset, so every render minted a fresh index and re-ran every effect —
	// deps were dead code. The slot-backed version must run on mount, skip
	// when deps are unchanged, and re-run when they change.
	ctx := core.NewContext()
	runs := make(chan int, 8)

	render := func(dep int) {
		renderPass(ctx, func(ctx *core.Context) {
			hooks.UseEffect(ctx, func() { runs <- dep }, dep)
		})
	}

	render(1)
	if got := awaitSignal(t, runs, "mount effect"); got != 1 {
		t.Fatalf("mount effect saw dep %d, want 1", got)
	}

	render(1)
	assertQuiet(t, runs, 100*time.Millisecond, "effect re-run with unchanged deps")

	render(2)
	if got := awaitSignal(t, runs, "effect after deps change"); got != 2 {
		t.Fatalf("changed-deps effect saw dep %d, want 2", got)
	}
}

func TestUseEffectWithNoDepsRunsOnce(t *testing.T) {
	ctx := core.NewContext()
	runs := make(chan struct{}, 8)

	body := func(ctx *core.Context) {
		hooks.UseEffect(ctx, func() { runs <- struct{}{} })
	}

	renderPass(ctx, body)
	awaitSignal(t, runs, "mount effect")
	renderPass(ctx, body)
	renderPass(ctx, body)
	assertQuiet(t, runs, 100*time.Millisecond, "re-run of a no-deps effect")
}

func TestUseEffectDoesNotAliasCallerDeps(t *testing.T) {
	// A caller spreading a slice it owns: the record must hold a copy, or the
	// caller's mutation would rewrite the stored deps too, DeepEqual would
	// report "unchanged", and the effect would never run again.
	ctx := core.NewContext()
	deps := []any{1}
	runs := make(chan int, 8)

	render := func() {
		renderPass(ctx, func(ctx *core.Context) {
			hooks.UseEffect(ctx, func() { runs <- deps[0].(int) }, deps...)
		})
	}

	render()
	awaitSignal(t, runs, "mount effect")

	deps[0] = 2
	render()
	if got := awaitSignal(t, runs, "effect after the caller mutated its deps slice"); got != 2 {
		t.Fatalf("re-run effect saw dep %d, want 2", got)
	}
}

func TestUseEffectSlotsAreIndependent(t *testing.T) {
	// Two effects in one component and the same component shape in a second
	// app: four distinct slots, each with its own deps memory. Under the old
	// global index none of them could be told apart.
	type appHarness struct {
		ctx  *core.Context
		runs chan string
	}
	newApp := func() *appHarness {
		return &appHarness{ctx: core.NewContext(), runs: make(chan string, 8)}
	}
	render := func(a *appHarness, name string, depA, depB int) {
		renderPass(a.ctx, func(ctx *core.Context) {
			hooks.UseEffect(ctx, func() { a.runs <- name + "-first" }, depA)
			hooks.UseEffect(ctx, func() { a.runs <- name + "-second" }, depB)
		})
	}

	one, two := newApp(), newApp()
	render(one, "one", 1, 1)
	render(two, "two", 1, 1)
	// Both apps mount both effects (order within an app is not guaranteed —
	// effects run on their own goroutines — so collect and count).
	seen := map[string]int{}
	for range 2 {
		seen[awaitSignal(t, one.runs, "mount effects (app one)")]++
		seen[awaitSignal(t, two.runs, "mount effects (app two)")]++
	}
	for _, name := range []string{"one-first", "one-second", "two-first", "two-second"} {
		if seen[name] != 1 {
			t.Fatalf("mount runs = %v, want each effect exactly once", seen)
		}
	}

	// Change only app one's second dep: exactly one effect re-runs, in the
	// right app.
	render(one, "one", 1, 2)
	render(two, "two", 1, 1)
	if got := awaitSignal(t, one.runs, "app one's changed effect"); got != "one-second" {
		t.Fatalf("re-run effect = %q, want one-second", got)
	}
	assertQuiet(t, one.runs, 100*time.Millisecond, "extra effect run in app one")
	assertQuiet(t, two.runs, 100*time.Millisecond, "effect run in untouched app two")
}

func TestUseIntervalTicksAndStopsOnContextClose(t *testing.T) {
	ctx := core.NewContext()
	ticks := make(chan struct{}, 64)

	renderPass(ctx, func(ctx *core.Context) {
		hooks.UseInterval(ctx, func() { ticks <- struct{}{} }, 10*time.Millisecond)
	})

	awaitSignal(t, ticks, "first interval tick")
	ctx.Close()

	// A tick already in flight when Close lands may still arrive; drain until
	// the channel goes quiet, then require it to stay quiet.
	for {
		select {
		case <-ticks:
			continue
		case <-time.After(50 * time.Millisecond):
		}
		break
	}
	assertQuiet(t, ticks, 100*time.Millisecond, "tick after Close")
}

func TestUseIntervalSameCursorInTwoAppsBothTick(t *testing.T) {
	// Regression for the global store keyed by cursor alone: two apps whose
	// intervals landed on the same cursor shared the key "interval-0", so the
	// second app's ticker was never created.
	tickApp := func(t *testing.T) (*core.Context, chan struct{}) {
		t.Helper()
		ctx := core.NewContext()
		ticks := make(chan struct{}, 64)
		renderPass(ctx, func(ctx *core.Context) {
			hooks.UseInterval(ctx, func() { ticks <- struct{}{} }, 10*time.Millisecond)
		})
		return ctx, ticks
	}

	ctxA, ticksA := tickApp(t)
	defer ctxA.Close()
	ctxB, ticksB := tickApp(t)
	defer ctxB.Close()

	awaitSignal(t, ticksA, "tick from app A")
	awaitSignal(t, ticksB, "tick from app B")
}

func TestUseIntervalSameCursorInTwoComponentsBothTick(t *testing.T) {
	// The same collision existed within one app: two components each calling
	// UseInterval at their own cursor 0 mapped to one global key. Scoped child
	// contexts give each component its own slot array.
	ctx := core.NewContext()
	defer ctx.Close()
	ticksA := make(chan struct{}, 64)
	ticksB := make(chan struct{}, 64)

	renderPass(ctx, func(ctx *core.Context) {
		hooks.UseInterval(ctx.Scope("a"), func() { ticksA <- struct{}{} }, 10*time.Millisecond)
		hooks.UseInterval(ctx.Scope("b"), func() { ticksB <- struct{}{} }, 10*time.Millisecond)
	})

	awaitSignal(t, ticksA, "tick from component A")
	awaitSignal(t, ticksB, "tick from component B")
}

func TestUseIntervalTicksRunTheLatestClosure(t *testing.T) {
	// Re-renders refresh the callback, so ticks observe current state
	// captures instead of the mount render's stale closure.
	ctx := core.NewContext()
	defer ctx.Close()
	ticks := make(chan int, 64)

	render := func(generation int) {
		renderPass(ctx, func(ctx *core.Context) {
			hooks.UseInterval(ctx, func() { ticks <- generation }, 10*time.Millisecond)
		})
	}

	render(1)
	awaitSignal(t, ticks, "tick from mount closure")
	render(2)

	// Ticks from generation 1 may still drain through; generation 2 must
	// appear once they do.
	deadline := time.After(2 * time.Second)
	for {
		select {
		case g := <-ticks:
			if g == 2 {
				return
			}
		case <-deadline:
			t.Fatal("interval never ran the refreshed closure")
		}
	}
}

func TestUseTimeoutFiresOnceAndDoesNotRearm(t *testing.T) {
	ctx := core.NewContext()
	defer ctx.Close()
	fires := make(chan struct{}, 8)

	body := func(ctx *core.Context) {
		hooks.UseTimeout(ctx, func() { fires <- struct{}{} }, 20*time.Millisecond)
	}

	renderPass(ctx, body)
	awaitSignal(t, fires, "timeout fire")

	// The old store deleted a fired timeout's key, so the next render armed
	// it again — firing repeatedly on render cadence. A fired slot must stay
	// fired.
	renderPass(ctx, body)
	renderPass(ctx, body)
	assertQuiet(t, fires, 100*time.Millisecond, "re-fire after later renders")
}

func TestUseTimeoutCancelledByClose(t *testing.T) {
	ctx := core.NewContext()
	fires := make(chan struct{}, 8)

	renderPass(ctx, func(ctx *core.Context) {
		hooks.UseTimeout(ctx, func() { fires <- struct{}{} }, 50*time.Millisecond)
	})
	ctx.Close()

	assertQuiet(t, fires, 200*time.Millisecond, "fire from a cancelled timeout")
}
