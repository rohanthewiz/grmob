package render_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
	"github.com/rohanthewiz/grmob/render"
)

// chanListener adapts the PatchListener push into a channel so tests can wait
// on deliveries with a timeout instead of sleeping.
type chanListener struct{ ch chan string }

func newChanListener() *chanListener {
	// Generous buffer: the pump must never block on a slow listener in tests.
	return &chanListener{ch: make(chan string, 16)}
}

func (c *chanListener) ApplyPatches(patches string) { c.ch <- patches }

// awaitPush waits for a pushed patch payload satisfying pred.
func awaitPush(t *testing.T, l *chanListener, timeout time.Duration, pred func(string) bool) string {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case p := <-l.ch:
			if pred(p) {
				return p
			}
			// Not the payload we're waiting for (e.g. an intermediate
			// coalesced push) — keep draining until the deadline.
		case <-deadline:
			t.Fatalf("timed out after %v waiting for a matching push", timeout)
		}
	}
}

func TestStateChangePushesToListener(t *testing.T) {
	ctx := core.NewContext()
	m := render.New(ctx, counterApp)
	defer m.Close()

	tree := decodeTree(t, m.RenderInitial())
	onClick := tree.Children[1].Props["onClick"].(string)

	l := newChanListener()
	m.SetListener(l)

	// The event is dispatched WITHOUT the native side rendering — exactly the
	// shape of an async state change (hence ctx.TriggerCallback, not
	// m.DispatchCallback, which folds in a render whose diff would consume
	// the change). The push channel alone must carry the update out.
	ctx.TriggerCallback(onClick)

	payload := awaitPush(t, l, 2*time.Second, func(p string) bool {
		return strings.Contains(p, "count: 1")
	})
	var patches []jsonPatch
	if err := json.Unmarshal([]byte(payload), &patches); err != nil {
		t.Fatalf("pushed payload is not patch JSON: %v\n%s", err, payload)
	}
	if len(patches) != 1 || patches[0].Type != "update-props" || patches[0].TargetID != "root/0" {
		t.Errorf("pushed patches = %+v, want single update-props on root/0", patches)
	}
}

func TestRapidStateChangesCoalesce(t *testing.T) {
	ctx := core.NewContext()
	m := render.New(ctx, counterApp)
	defer m.Close()

	tree := decodeTree(t, m.RenderInitial())
	onClick := tree.Children[1].Props["onClick"].(string)

	l := newChanListener()
	m.SetListener(l)

	const clicks = 5
	for range clicks {
		ctx.TriggerCallback(onClick)
	}

	// The final state must arrive; the buffered-by-one request channel means
	// the burst arrives in at most `clicks` pushes (typically far fewer), and
	// nothing after the final value.
	awaitPush(t, l, 2*time.Second, func(p string) bool {
		return strings.Contains(p, fmt.Sprintf("count: %d", clicks))
	})
	select {
	case p := <-l.ch:
		t.Errorf("unexpected push after final state was delivered: %s", p)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestNoListenerLeavesPollingFlowIntact(t *testing.T) {
	// Regression guard for the pump/polling interaction: with no listener the
	// pump must NOT consume the diff, or a polling runtime (WASM today) would
	// call RenderAgain itself and get "[]" while the screen is stale.
	ctx := core.NewContext()
	m := render.New(ctx, counterApp)
	defer m.Close()

	tree := decodeTree(t, m.RenderInitial())
	onClick := tree.Children[1].Props["onClick"].(string)

	ctx.TriggerCallback(onClick)
	// Give the pump a chance to (incorrectly) swallow the change.
	time.Sleep(50 * time.Millisecond)

	patches := decodePatches(t, m.RenderAgain())
	if len(patches) != 1 || patches[0].TargetID != "root/0" {
		t.Fatalf("polling RenderAgain lost the diff to the pump: %+v", patches)
	}
}

func TestListenerAttachedLateReceivesPendingChange(t *testing.T) {
	// A state change that happens before the native side attaches must be
	// flushed on attachment (SetListener re-nudges the pump).
	ctx := core.NewContext()
	m := render.New(ctx, counterApp)
	defer m.Close()

	tree := decodeTree(t, m.RenderInitial())
	onClick := tree.Children[1].Props["onClick"].(string)

	ctx.TriggerCallback(onClick)

	l := newChanListener()
	m.SetListener(l)
	awaitPush(t, l, 2*time.Second, func(p string) bool {
		return strings.Contains(p, "count: 1")
	})
}

func TestIntervalTickPushesWithoutAnyNativeEvent(t *testing.T) {
	// The headline scenario for the push channel: a timer-driven UI updating
	// with no native event in flight to piggyback on.
	tickerApp := func(ctx *core.Context) core.View {
		return core.ComponentFunc(func(ctx *core.Context) *core.Node {
			count := core.NewState(ctx, 0)
			hooks.UseInterval(ctx, func() {
				count.Set(count.Get() + 1)
			}, 20*time.Millisecond)
			return core.Text(fmt.Sprintf("count: %d", count.Get())).Render(ctx)
		})
	}

	// m.Close (deferred below) also stops the interval's ticker: hook
	// resources belong to the context tree now, and closing the manager
	// closes the tree — no global sweep needed for later tests.
	m := render.New(core.NewContext(), tickerApp)
	defer m.Close()

	l := newChanListener()
	m.SetListener(l)
	m.RenderInitial()

	awaitPush(t, l, 2*time.Second, func(p string) bool {
		return strings.Contains(p, "update-props") && strings.Contains(p, "count:")
	})
}

func TestConcurrentDispatchAndTimerPushesDoNotRace(t *testing.T) {
	// The gap-4 marshaling guarantee: native-event dispatch (DispatchCallback)
	// and pump renders driven by timer ticks contend on the same render mutex,
	// so handlers never interleave with a render pass. Several goroutines
	// hammer the event path while an interval mutates state from its own
	// goroutine; -race is the assertion, plus every dispatch must return
	// decodable patch JSON.
	tickerCounterApp := func(ctx *core.Context) core.View {
		return core.ComponentFunc(func(ctx *core.Context) *core.Node {
			clicks := core.NewState(ctx, 0)
			ticks := core.NewState(ctx, 0)
			hooks.UseInterval(ctx, func() {
				ticks.Set(ticks.Get() + 1)
			}, 5*time.Millisecond)
			return core.Column(
				core.Text(fmt.Sprintf("clicks: %d ticks: %d", clicks.Get(), ticks.Get())),
				core.Button("increment", func() {
					clicks.Set(clicks.Get() + 1)
				}),
			).Render(ctx)
		})
	}

	m := render.New(core.NewContext(), tickerCounterApp)
	defer m.Close()

	tree := decodeTree(t, m.RenderInitial())
	onClick := tree.Children[1].Props["onClick"].(string)

	l := newChanListener()
	m.SetListener(l)

	const dispatchers, clicksEach = 3, 20
	var wg sync.WaitGroup
	for range dispatchers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range clicksEach {
				// t.Error (not the Fatal-based decode helper) — Fatal must
				// not be called off the test goroutine.
				var p []jsonPatch
				if out := m.DispatchCallback(onClick); json.Unmarshal([]byte(out), &p) != nil {
					t.Errorf("dispatch returned non-patch JSON: %s", out)
				}
			}
		}()
	}
	wg.Wait()

	// All clicks were dispatched serially under the render mutex, so the
	// final total must eventually surface — on the dispatch return path or
	// via a timer-driven push, either of which renders the settled state.
	want := fmt.Sprintf("clicks: %d", dispatchers*clicksEach)
	awaitPush(t, l, 2*time.Second, func(p string) bool {
		return strings.Contains(p, want)
	})
}

func TestConcurrentStateWritesDoNotRace(t *testing.T) {
	// Regression guard for the slot-access data race: State.Set from app
	// goroutines used to write ctx.slots unsynchronized while the pump's
	// render pass read them. The assertion here is the -race detector itself;
	// the final awaitPush just proves the pipeline survived the storm and the
	// last write still reached the listener.
	var (
		once sync.Once
		st   core.State[int]
	)
	app := func(ctx *core.Context) core.View {
		s := core.NewState(ctx, 0)
		// Capture the accessor exactly once: the closure pair is re-created
		// each render on the render goroutine, and re-assigning the captured
		// variable there would itself race with the writer goroutines below.
		once.Do(func() { st = s })
		return core.Column(
			core.Text(fmt.Sprintf("n: %d", s.Get())),
		)
	}

	m := render.New(core.NewContext(), app)
	defer m.Close()
	m.RenderInitial()

	l := newChanListener()
	m.SetListener(l)

	const writers, writes = 4, 50
	var wg sync.WaitGroup
	for w := range writers {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := range writes {
				st.Set(w*1000 + i)
			}
		}(w)
	}
	wg.Wait()

	// Whichever writer landed last, its value must eventually be pushed.
	final := st.Get()
	awaitPush(t, l, 2*time.Second, func(p string) bool {
		return strings.Contains(p, fmt.Sprintf("n: %d", final))
	})
}

func TestManagerCloseStopsHookIntervals(t *testing.T) {
	// The lifecycle-ownership contract: the Manager owns the app's lifetime,
	// so closing it must also stop background resources hooks registered on
	// the context tree — here an interval ticker. Without this, a replaced
	// app (mobile.Register, WASM re-mount) would keep ticking and rendering
	// into the void forever.
	ticks := make(chan struct{}, 64)
	tickerApp := func(ctx *core.Context) core.View {
		return core.ComponentFunc(func(ctx *core.Context) *core.Node {
			hooks.UseInterval(ctx, func() { ticks <- struct{}{} }, 10*time.Millisecond)
			return core.Text("ticking").Render(ctx)
		})
	}

	m := render.New(core.NewContext(), tickerApp)
	m.RenderInitial()

	select {
	case <-ticks:
	case <-time.After(2 * time.Second):
		t.Fatal("interval never ticked before Close")
	}

	m.Close()

	// A tick in flight at Close time may still land; drain until quiet, then
	// require sustained silence (10x the tick period).
	for {
		select {
		case <-ticks:
			continue
		case <-time.After(50 * time.Millisecond):
		}
		break
	}
	select {
	case <-ticks:
		t.Fatal("interval kept ticking after Manager.Close")
	case <-time.After(100 * time.Millisecond):
	}
}
