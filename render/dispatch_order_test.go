package render_test

import (
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// recordingListener is a host whose push path appends to the same arrival log
// the dispatch return path does, as GrMobRuntime.kt and GrMobRuntime.swift
// both do (one FIFO, fed by both paths). It never blocks: the delivery runs
// inside the render lock, and a listener that waited on the test would wait
// on a test that is waiting on the lock.
type recordingListener struct{ log *arrivalLog }

func (r recordingListener) ApplyPatches(patches string) { r.log.record(patches) }

// dispatchOrderApp is a counter whose state the test can also write from
// outside, the shape of a timer or network callback.
func dispatchOrderApp(st *core.State[int], once *sync.Once) func(*core.Context) core.View {
	return func(ctx *core.Context) core.View {
		n := core.NewState(ctx, 0)
		once.Do(func() { *st = n })
		return core.Column(
			core.Text(fmt.Sprintf("n: %d", n.Get())),
			core.Button("bump", func() { n.Set(n.Get() + 1) }),
		)
	}
}

// TestADispatchsPatchesArriveBeforeALaterPush is the converse of
// TestPushDeliveryPrecedesLaterDispatchPatches (push_order_test.go), and the
// pin for N-132.
//
// That test holds a push open and shows a dispatch cannot overtake it. This is
// the other direction. A dispatch used to return its patches as a value, and
// the render lock was released when it returned, before the host had queued
// them. A pump pass started in that gap rendered the next state and pushed it,
// so the host's FIFO received pass N+1 ahead of pass N:
//
//	event thread                      pump
//	lock; tap; render n:1; unlock
//	  (host still holding "n: 1")     lock; render n:2; push "n: 2"; unlock
//	queue "n: 1"   <-- applied second, against a tree already at 2
//
// Paths are positional, so applying the stale pass writes to the wrong node or
// leaves an old prop standing, silently. A probe measured 1 to 4 inversions per
// 20,000 taps; the JNI string copy widens the gap on a device.
//
// The gap is forced here: the "host" holds the dispatch's return value until
// the push has been delivered, and only then queues it. With the fix the
// dispatch hands its patches to the listener inside the lock and returns "[]",
// so there is nothing left to queue late.
func TestADispatchsPatchesArriveBeforeALaterPush(t *testing.T) {
	var (
		st   core.State[int]
		once sync.Once
	)
	m := render.New(core.NewContext(), dispatchOrderApp(&st, &once))
	defer m.Close()

	tree := decodeTree(t, m.RenderInitial())
	onClick := tree.Children[1].Props["onClick"].(string)

	log := &arrivalLog{}
	m.SetListener(recordingListener{log: log})

	// The host's event thread: dispatch, then hold the return value until
	// told to queue it, which is the JNI copy taking its time.
	returned := make(chan string)
	queue := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		out := m.DispatchCallback(onClick) // n: 0 -> 1
		returned <- out
		<-queue
		log.record(out)
	}()

	var out string
	select {
	case out = <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("the dispatch never returned")
	}

	// A goroutine's write lands in the gap and is pushed at once.
	st.Set(2)
	for waited := time.Duration(0); !slices.Contains(log.snapshot(), 2); waited += 5 * time.Millisecond {
		if waited > 2*time.Second {
			t.Fatalf("the pump never pushed the write made in the gap; arrivals %v", log.snapshot())
		}
		time.Sleep(5 * time.Millisecond)
	}

	close(queue)
	<-done

	got := log.snapshot()
	if !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("arrivals %v, want [1 2]: the tap's pass reached the host after the pass "+
			"that followed it, and would be applied against a tree it was not diffed from", got)
	}
	if out != "[]" {
		t.Errorf("with a listener attached the dispatch returned %s, want []: its patches "+
			"belong on the listener, the one ordered door", out)
	}
}

// The same ordering under load, the shape the probe measured: taps on one
// thread, each handler also writing from a goroutine of its own, and every
// payload from either path landing in one log. Whatever interleaving the
// scheduler picks, the counter the host sees must never go backwards.
func TestDispatchesInterleavedWithGoroutineWritesArriveInOrder(t *testing.T) {
	var (
		st   core.State[int]
		once sync.Once
		// The writes are numbered and made under one mutex, so the state
		// itself only ever rises. A read-modify-write from two goroutines
		// could set an older number after a newer one, and the test would
		// then be measuring the app, not the delivery.
		writeMu sync.Mutex
		next    int
	)
	write := func() {
		writeMu.Lock()
		defer writeMu.Unlock()
		next++
		st.Set(next)
	}
	app := func(ctx *core.Context) core.View {
		n := core.NewState(ctx, 0)
		once.Do(func() { st = n })
		return core.Column(
			core.Text(fmt.Sprintf("n: %d", n.Get())),
			core.Button("bump", func() {
				// Once in the handler, once from a goroutine: the second
				// write races the dispatch's own return, which is the window.
				write()
				go write()
			}),
		)
	}
	m := render.New(core.NewContext(), app)
	defer m.Close()

	tree := decodeTree(t, m.RenderInitial())
	onClick := tree.Children[1].Props["onClick"].(string)

	log := &arrivalLog{}
	m.SetListener(recordingListener{log: log})

	const taps = 2000
	for range taps {
		log.record(m.DispatchCallback(onClick))
	}
	// Let the last goroutine writes be pushed.
	time.Sleep(50 * time.Millisecond)

	got := log.snapshot()
	for i := 1; i < len(got); i++ {
		if got[i] < got[i-1] {
			t.Fatalf("arrival %d is n: %d after n: %d; the host would apply an older pass "+
				"over a newer one (%d arrivals in all)", i, got[i], got[i-1], len(got))
		}
	}
}
