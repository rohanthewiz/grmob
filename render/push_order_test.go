package render_test

import (
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// seqRe pulls the counter out of whatever patch payload carries it. Both the
// push path and the dispatch return path serialize the same Text node, so one
// expression reads both.
var seqRe = regexp.MustCompile(`n: (\d+)`)

func seqIn(payload string) (int, bool) {
	m := seqRe.FindStringSubmatch(payload)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}

// arrivalLog is the test's stand-in for the native side of the delivery
// contract: GrMobRuntime.kt and GrMobRuntime.swift funnel both patch paths
// into one FIFO queue and apply them in arrival order, so what matters is the
// order in which payloads are *handed over*, not which goroutine produced
// them.
type arrivalLog struct {
	mu  sync.Mutex
	seq []int
}

func (a *arrivalLog) record(payload string) {
	if n, ok := seqIn(payload); ok {
		a.mu.Lock()
		a.seq = append(a.seq, n)
		a.mu.Unlock()
	}
}

func (a *arrivalLog) snapshot() []int {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]int, len(a.seq))
	copy(out, a.seq)
	return out
}

// slowListener holds the delivery open long enough for a competing dispatch to
// finish its own render pass, which is precisely the window the ordering bug
// lived in. It announces entry on `entered` so the test can schedule the
// dispatch at the exact moment delivery is in flight.
type slowListener struct {
	log     *arrivalLog
	entered chan struct{}
	hold    time.Duration
	once    sync.Once
}

func (s *slowListener) ApplyPatches(patches string) {
	s.once.Do(func() { close(s.entered) })
	time.Sleep(s.hold)
	s.log.record(patches)
}

// TestPushDeliveryPrecedesLaterDispatchPatches pins the pump's emission/delivery
// atomicity.
//
// The pump used to render under the mutex, release it, and only then call the
// listener. A native event dispatching in that gap ran its own pass and
// enqueued the newer patches first, so the host applied pass N+1 against the
// pre-N tree — and since patch paths are positional ("root/2"), that lands
// changes on the wrong nodes.
//
// Timeline forced here (the listener holds delivery for `hold`):
//
//	async Set(1) ─▶ pump: render "n: 1" ─▶ listener entered ····· hold ····· delivered
//	                                          │
//	                                          └─▶ dispatch: bump to 2, render "n: 2"
//
// Correct behavior is that the dispatch cannot complete its pass until the
// listener returns, so arrivals read 1 then 2.
func TestPushDeliveryPrecedesLaterDispatchPatches(t *testing.T) {
	var (
		once sync.Once
		st   core.State[int]
	)
	app := func(ctx *core.Context) core.View {
		n := core.NewState(ctx, 0)
		// Captured once, on the first render, for the same reason as
		// TestConcurrentStateWritesDoNotRace: re-assigning per pass would
		// itself race the writer goroutine.
		once.Do(func() { st = n })
		return core.Column(
			core.Text(fmt.Sprintf("n: %d", n.Get())),
			core.Button("bump", func() { n.Set(n.Get() + 1) }),
		)
	}

	m := render.New(core.NewContext(), app)
	defer m.Close()

	tree := decodeTree(t, m.RenderInitial())
	onClick := tree.Children[1].Props["onClick"].(string)

	log := &arrivalLog{}
	l := &slowListener{log: log, entered: make(chan struct{}), hold: 150 * time.Millisecond}
	m.SetListener(l)

	// Async state change: no render on this path, so the pump alone carries
	// it out — the shape of a timer or network callback.
	st.Set(1)

	select {
	case <-l.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("the pump never delivered the async state change")
	}

	// Fire the event while delivery of pass 1 is still in flight.
	done := make(chan struct{})
	go func() {
		defer close(done)
		log.record(m.DispatchCallback(onClick))
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("dispatch never returned")
	}

	// Wait for the held-open push to land as well; without the fix it lands
	// *after* the dispatch's newer patches, which is the failure this pins.
	var got []int
	for waited := time.Duration(0); waited < 3*time.Second; waited += 10 * time.Millisecond {
		if got = log.snapshot(); len(got) >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(got) < 2 {
		t.Fatalf("expected two arrivals (one push, one dispatch), got %v", got)
	}
	for i := 1; i < len(got); i++ {
		if got[i] < got[i-1] {
			t.Fatalf("patches arrived out of order: %v — pass %d was applied before pass %d",
				got, got[i], got[i-1])
		}
	}
}

// panicOnceListener throws on its first delivery and records every later one,
// standing in for a JS exception out of GrMobApplyPatches or a Java exception
// surfacing through gomobile.
type panicOnceListener struct {
	mu       sync.Mutex
	panicked bool
	ch       chan string
}

func (p *panicOnceListener) ApplyPatches(patches string) {
	p.mu.Lock()
	first := !p.panicked
	p.panicked = true
	p.mu.Unlock()
	if first {
		panic("listener exploded")
	}
	p.ch <- patches
}

// TestListenerPanicDoesNotKillPump pins the pump's panic guard.
//
// Without it a single throw from the host's patch applier unwound the pump
// goroutine, and since the pump is the only consumer of renderRequests, every
// later State.Set filled the one-slot buffer and was dropped. Taps still
// worked (they render on the caller's thread), so the symptom was "async
// updates just stopped" with no crash to point at.
func TestListenerPanicDoesNotKillPump(t *testing.T) {
	var (
		once sync.Once
		st   core.State[int]
	)
	app := func(ctx *core.Context) core.View {
		n := core.NewState(ctx, 0)
		once.Do(func() { st = n })
		return core.Column(core.Text(fmt.Sprintf("n: %d", n.Get())))
	}

	m := render.New(core.NewContext(), app)
	defer m.Close()
	m.RenderInitial()

	l := &panicOnceListener{ch: make(chan string, 8)}
	m.SetListener(l)

	// First push panics inside the listener.
	st.Set(1)

	// Second push must still arrive, which is only possible if the pump
	// goroutine survived the first.
	deadline := time.After(3 * time.Second)
	for want := 2; ; want++ {
		st.Set(want)
		select {
		case p := <-l.ch:
			if n, ok := seqIn(p); !ok || n == 0 {
				t.Fatalf("second push carried no counter: %s", p)
			}
			return
		case <-time.After(50 * time.Millisecond):
			// Retry: the very first Set may have been coalesced into the
			// pass that panicked.
		case <-deadline:
			t.Fatal("no push arrived after the listener panicked — the pump died")
		}
		if want > 20 {
			t.Fatal("no push arrived after the listener panicked — the pump died")
		}
	}
}
