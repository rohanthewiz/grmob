package hooks

import (
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// The delay lands on the next boundary, and a time already on one waits a full
// interval rather than zero (which would tick twice for one instant).
func TestUntilNextBoundary(t *testing.T) {
	base := time.Date(2026, 9, 16, 10, 42, 7, 0, time.UTC)
	cases := []struct {
		name  string
		t     time.Time
		every time.Duration
		want  time.Duration
	}{
		{"mid second", base.Add(250 * time.Millisecond), time.Second, 750 * time.Millisecond},
		{"on the second", base, time.Second, time.Second},
		{"mid minute", base, time.Minute, 53 * time.Second},
	}
	for _, c := range cases {
		if got := untilNextBoundary(c.t, c.every); got != c.want {
			t.Errorf("%s: untilNextBoundary = %v, want %v", c.name, got, c.want)
		}
	}
}

// A mount returns a real time, a boundary tick requests a render and advances
// what the next pass reads, and Close stops the goroutine.
func TestUseNowTicksOnTheBoundaryAndStopsOnClose(t *testing.T) {
	ctx := core.NewContext()
	renders := make(chan struct{}, 8)
	ctx.OnStateChange(func() { renders <- struct{}{} })

	pass := func() time.Time {
		ctx.Reset()
		return UseNow(ctx, 20*time.Millisecond)
	}

	first := pass()
	if first.IsZero() {
		t.Fatal("mount pass returned the zero time")
	}

	select {
	case <-renders:
	case <-time.After(2 * time.Second):
		t.Fatal("no render requested by a tick")
	}
	second := pass()
	if !second.After(first) {
		t.Fatalf("time did not advance across a tick: %v then %v", first, second)
	}
	// The tick fired at (or just after) a multiple of the interval.
	if off := second.Sub(second.Truncate(20 * time.Millisecond)); off > 10*time.Millisecond {
		t.Errorf("tick landed %v past its boundary", off)
	}

	ctx.Close()
	for {
		select {
		case <-renders:
			continue
		case <-time.After(60 * time.Millisecond):
		}
		break
	}
	select {
	case <-renders:
		t.Fatal("render requested after Close")
	case <-time.After(80 * time.Millisecond):
	}
}
