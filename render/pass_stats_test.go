package render_test

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/grmobtest"
	"github.com/rohanthewiz/grmob/render"
)

// withDebugMode turns debug mode on for one test and restores the previous
// setting, and starts from an empty pass record.
func withDebugMode(t *testing.T) {
	t.Helper()
	was := core.IsDebugMode()
	core.SetDebugMode(true)
	core.ClearPassStats()
	t.Cleanup(func() {
		core.SetDebugMode(was)
		core.ClearPassStats()
	})
}

type nopListener struct{}

func (nopListener) ApplyPatches(string) {}

// Each phase is charged to the pass that ran it, and to the right phase.
//
// Inside a fake-clock bubble time moves only when something sleeps, so a
// handler that sleeps 5 ms and a render that sleeps 3 ms give exactly those
// durations and every other phase exactly zero: the attribution is checked
// to the nanosecond, not within a tolerance.
func TestPassStatsChargeEachPhaseToItsPass(t *testing.T) {
	withDebugMode(t)
	grmobtest.WithFakeClock(t, func(t *testing.T) {
		var slowRender atomic.Bool
		app := grmobtest.New(t, func(ctx *core.Context) core.View {
			n := core.NewState(ctx, 0)
			if slowRender.Load() {
				time.Sleep(3 * time.Millisecond)
			}
			return core.Column(
				core.Text(fmt.Sprintf("n=%d", n.Get())),
				core.Button("+", func() {
					time.Sleep(5 * time.Millisecond)
					n.Set(n.Get() + 1)
				}),
			)
		})

		wire := app.Manager().RenderInitial()
		slowRender.Store(true)
		// Tap reads a fresh tree to find the button (a RenderInitial, so a
		// mount-shaped pass, now slow), then dispatches.
		patches := app.Tap("+")

		stats := core.PassStats()
		if len(stats) != 3 {
			t.Fatalf("recorded %d passes, want 3 (the mount, Tap's read, the dispatch): %+v", len(stats), stats)
		}
		mount, slowMount, tap := stats[0], stats[1], stats[2]

		if mount.Kind != "mount" || mount.Bytes != len(wire) || mount.Patches != 0 ||
			mount.Handler != 0 || mount.Diff != 0 || mount.Build != 0 {
			t.Errorf("the first mount = %+v, want a mount of %d bytes with no time spent", mount, len(wire))
		}
		if slowMount.Build != 3*time.Millisecond || slowMount.Encode != 0 {
			t.Errorf("a render sleeping 3ms gave Build %v, Encode %v", slowMount.Build, slowMount.Encode)
		}
		if tap.Kind != "dispatch" || !strings.HasPrefix(tap.Trigger, "cb_") {
			t.Errorf("the tap's pass = %+v, want a dispatch triggered by its callback ID", tap)
		}
		if tap.Handler != 5*time.Millisecond || tap.Build != 3*time.Millisecond ||
			tap.Diff != 0 || tap.Encode != 0 {
			t.Errorf("the tap's phases = handler %v build %v diff %v encode %v, want 5ms, 3ms, 0, 0",
				tap.Handler, tap.Build, tap.Diff, tap.Encode)
		}
		if tap.Total() != 8*time.Millisecond {
			t.Errorf("Total = %v, want the phases' sum, 8ms", tap.Total())
		}
		if tap.Bytes != len(patches) || tap.Patches != 1 {
			t.Errorf("the tap sent %d patches in %d bytes; recorded %d in %d",
				1, len(patches), tap.Patches, tap.Bytes)
		}
	})
}

// A pass no handler started (the pump pushing a state change from a
// goroutine) is a "render", and a dispatch's handler time does not leak into
// the pass after it.
func TestAPushIsARenderPassWithNoHandler(t *testing.T) {
	withDebugMode(t)
	grmobtest.WithFakeClock(t, func(t *testing.T) {
		var bump func()
		app := grmobtest.New(t, func(ctx *core.Context) core.View {
			n := core.NewState(ctx, 0)
			bump = func() { n.Set(n.Get() + 1) }
			return core.Text(fmt.Sprintf("n=%d", n.Get()))
		})
		app.Tree()
		app.Manager().SetListener(nopListener{})
		time.Sleep(time.Millisecond) // the attach's own flush, if any
		core.ClearPassStats()

		go bump()
		time.Sleep(10 * time.Millisecond) // the pump renders and pushes

		stats := core.PassStats()
		if len(stats) != 1 || stats[0].Kind != "render" || stats[0].Handler != 0 || stats[0].Trigger != "" {
			t.Fatalf("a pushed state change recorded %+v, want one render pass with no handler", stats)
		}
	})
}

// With debug mode off nothing is timed or recorded.
func TestPassStatsAreOffWithDebugMode(t *testing.T) {
	was := core.IsDebugMode()
	core.SetDebugMode(false)
	core.ClearPassStats()
	t.Cleanup(func() { core.SetDebugMode(was) })

	mgr := render.New(core.NewContext(), func(ctx *core.Context) core.View {
		return core.Button("x", func() {})
	})
	defer mgr.Close()
	mgr.RenderInitial()
	mgr.RenderAgain()
	if got := core.PassStats(); len(got) != 0 {
		t.Fatalf("debug mode off recorded %d passes", len(got))
	}
}

// How many times one tap renders the root, with a host listening: once. It
// used to be twice: the dispatch rendered the pass the tap needed, and the
// handler's State.Set also left a token for the pump, which woke and
// rendered again to find nothing ("[]"). Plan item P1 (N-138) gave the dirty
// flag a generation, so the pump now sees the write was already drawn and
// skips. The pin started at two per tap and was changed to one by that fix.
//
// In a fake-clock bubble the 10 ms between taps lets the pump run to
// completion each time, so the count is exact rather than a race between
// the pump and the next tap. The passes are counted twice over, by the root
// function and by the debug-mode pass record, and the two must agree.
func TestATapRendersTheRootOnce(t *testing.T) {
	withDebugMode(t)
	grmobtest.WithFakeClock(t, func(t *testing.T) {
		var roots atomic.Int32
		app := grmobtest.New(t, func(ctx *core.Context) core.View {
			roots.Add(1)
			n := core.NewState(ctx, 0)
			return core.Column(
				core.Text(fmt.Sprintf("n=%d", n.Get())),
				core.Button("+", func() { n.Set(n.Get() + 1) }),
			)
		})
		app.Tree()
		app.Manager().SetListener(nopListener{})
		time.Sleep(10 * time.Millisecond)

		const taps = 20
		// Read every tap's ID from a tree, as a host does, before counting:
		// the reads are render passes too.
		ids := make([]string, taps)
		for i := range ids {
			ids[i] = app.Find(grmobtest.Button("+")).Props["onClick"].(string)
		}
		roots.Store(0)
		core.ClearPassStats()
		for _, id := range ids {
			app.Manager().DispatchCallback(id)
			time.Sleep(10 * time.Millisecond)
		}

		const perTap = 1 // was 2 before P1
		if got := roots.Load(); got != taps*perTap {
			t.Fatalf("%d taps rendered the root %d times, want %d (%d per tap)", taps, got, taps*perTap, perTap)
		}
		var dispatches, renders int
		for _, s := range core.PassStats() {
			switch s.Kind {
			case "dispatch":
				dispatches++
			case "render":
				renders++
				if s.Patches != 0 {
					t.Errorf("the pump's pass after a tap sent %d patches; it should find nothing left", s.Patches)
				}
			}
		}
		if dispatches != taps || renders != taps*(perTap-1) {
			t.Fatalf("the pass record has %d dispatches and %d renders, want %d and %d",
				dispatches, renders, taps, taps*(perTap-1))
		}
	})
}

// payloadLog keeps every pushed payload, for the tests that ask what
// the pump sent rather than how often it rendered.
type payloadLog struct {
	mu   sync.Mutex
	sent []string
}

func (l *payloadLog) ApplyPatches(p string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sent = append(l.sent, p)
}

func (l *payloadLog) all() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.sent, "\n")
}

// The other half of P1: skipping the pump's pass must not skip a write the
// dispatch's pass never saw. Here the tap's own pass writes a second state
// while it builds (the shape of an effect's Set, or a timer firing on another
// goroutine mid-pass). That write is newer than the generation the pass
// started from, so it stays dirty, and the pump must render it and push it.
// Clearing the flag outright at the end of the pass would have swallowed it:
// the screen would say "echo=0" until some unrelated event.
func TestAWriteDuringATapsPassStillGetsItsOwnPass(t *testing.T) {
	grmobtest.WithFakeClock(t, func(t *testing.T) {
		var roots atomic.Int32
		app := grmobtest.New(t, func(ctx *core.Context) core.View {
			roots.Add(1)
			n := core.NewState(ctx, 0)
			echo := core.NewState(ctx, 0)
			if n.Get() != echo.Get() {
				echo.Set(n.Get()) // a write during the build
			}
			return core.Column(
				core.Text(fmt.Sprintf("echo=%d", echo.Get())),
				core.Button("+", func() { n.Set(n.Get() + 1) }),
			)
		})
		app.Tree()
		l := &payloadLog{}
		app.Manager().SetListener(l)
		time.Sleep(10 * time.Millisecond)

		id := app.Find(grmobtest.Button("+")).Props["onClick"].(string)
		roots.Store(0)
		app.Manager().DispatchCallback(id)
		time.Sleep(10 * time.Millisecond)

		if got := roots.Load(); got != 2 {
			t.Fatalf("the tap rendered the root %d times, want 2: its own pass and the pump's for the write made during it", got)
		}
		if !strings.Contains(l.all(), "echo=1") {
			t.Fatalf("the write made during the tap's pass never reached the host; pushed:\n%s", l.all())
		}
	})
}
