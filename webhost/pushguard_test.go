package webhost

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

// inline makes a guard that runs its deferred work at once, so a test can read
// the order without waiting on a goroutine.
func inline() *PushGuard {
	return &PushGuard{spawn: func(run func()) { run() }}
}

func TestAPushGuardRunsCallsAtOnceOutsideADelivery(t *testing.T) {
	g := inline()
	if g.Defer("ReceiveEvent cb_1", func() { t.Error("deferred with no delivery in progress") }) {
		t.Fatal("Defer reported a delivery in progress before any had started")
	}
	if g.Delivering() {
		t.Fatal("Delivering before any delivery")
	}
}

// The deadlock's shape, minus the deadlock: a call made from inside push
// (the batch calling back into Go) waits until push has returned, and the
// calls run in the order they arrived.
func TestAPushGuardDefersCallsMadeDuringADeliveryUntilItEnds(t *testing.T) {
	var buf bytes.Buffer
	defer log.SetOutput(log.Writer())
	log.SetOutput(&buf)

	g := inline()
	var order []string
	g.Deliver(func() {
		if !g.Delivering() {
			t.Error("Delivering is false inside push")
		}
		order = append(order, "push begins")
		if !g.Defer("ReceiveEvent cb_4", func() { order = append(order, "cb_4") }) {
			t.Error("Defer ran a call inside a delivery instead of deferring it")
		}
		g.Defer("ReceiveEvent cb_5", func() { order = append(order, "cb_5") })
		order = append(order, "push ends")
	})
	if g.Delivering() {
		t.Error("Delivering is still true after push returned")
	}
	want := []string{"push begins", "push ends", "cb_4", "cb_5"}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Errorf("order = %v, want %v", order, want)
	}
	// Loud: a deferral means the runtime's own queue has a hole.
	if got := buf.String(); !strings.Contains(got, "ReceiveEvent cb_4") || !strings.Contains(got, "dispatchToGo") {
		t.Errorf("the deferral was not logged with its call and the fix to look for:\n%s", got)
	}
}

// A JS throw surfaces through syscall/js as a panic, which the manager's guard
// around ApplyPatches recovers. The guard must come down on the way out, or
// every later call would be deferred forever and nothing would render again.
func TestAPushGuardComesDownWhenPushPanics(t *testing.T) {
	defer log.SetOutput(log.Writer())
	log.SetOutput(&bytes.Buffer{})

	g := inline()
	ran := false
	func() {
		defer func() { _ = recover() }()
		g.Deliver(func() {
			g.Defer("ReceiveEvent cb_1", func() { ran = true })
			panic("TypeError: patches.forEach is not a function")
		})
	}()
	if g.Delivering() {
		t.Fatal("the guard stayed up after push panicked")
	}
	if !ran {
		t.Error("the call deferred before the panic was dropped")
	}
}

// Production starts the deferred work on a goroutine, after the pump has let
// go of the lock; see Deliver.
func TestAPushGuardStartsDeferredWorkOnAGoroutineByDefault(t *testing.T) {
	defer log.SetOutput(log.Writer())
	log.SetOutput(&bytes.Buffer{})

	var g PushGuard
	done := make(chan struct{})
	g.Deliver(func() {
		g.Defer("ReceiveEvent cb_1", func() { close(done) })
	})
	<-done
}
