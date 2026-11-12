package core

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// PassStat is what one render pass cost, as the render loop measured it.
// Recorded only under debug mode (SetDebugMode); read with PassStats.
//
// The phases are the pass's own, in the order they run:
//
//	host event ─▶ Handler ─▶ Build ─▶ Diff ─▶ Encode ─▶ payload to the host
//	              (dispatch)  the view    reconcile  json
//	                          tree        .Diff
//
// A mount has no handler and no diff (its payload is the whole tree), and a
// push (a state change from a timer or a goroutine, rendered by the pump) has
// no handler. A pass that was abandoned (a panicking render, an unencodable
// value) is not recorded: those are logged, and reported as concerns, where
// they happen.
type PassStat struct {
	// At is when the pass began: the handler's start for a dispatch, the
	// render's otherwise.
	At time.Time

	// Kind is "mount" (RenderInitial), "dispatch" (a host event and its
	// handler), or "render" (a pass with no handler: the pump's push, or a
	// host's explicit RenderAgain).
	Kind string

	// Trigger is the dispatch's callback ID or Dispatch label; empty for the
	// other kinds.
	Trigger string

	Handler time.Duration
	Build   time.Duration
	Diff    time.Duration
	Encode  time.Duration

	// Patches is the number of patches the pass sent; 0 for a mount, whose
	// payload is a tree, and 0 for a pass that changed nothing ("[]").
	Patches int

	// Bytes is the payload's size: what crossed to the host.
	Bytes int
}

// Total is the pass's whole cost in Go: handler, build, diff and encode.
func (s PassStat) Total() time.Duration {
	return s.Handler + s.Build + s.Diff + s.Encode
}

// String is the one-line form SetPassLogging writes:
//
//	grmob: pass dispatch cb_3: 1.21ms (handler 12µs, build 1.1ms, diff 97µs, encode 20µs), 3 patches, 412 B
func (s PassStat) String() string {
	what := s.Kind
	if s.Trigger != "" {
		what += " " + s.Trigger
	}
	return fmt.Sprintf("grmob: pass %s: %v (handler %v, build %v, diff %v, encode %v), %d patches, %d B",
		what, s.Total(), s.Handler, s.Build, s.Diff, s.Encode, s.Patches, s.Bytes)
}

// passStatsCap bounds the record. Debug mode is often left on for a whole
// development session, and a ticker re-rendering ten times a second would
// otherwise grow the record without end; the most recent passes are the ones
// a reader asks about.
const passStatsCap = 512

// passStats is a ring of the most recent passes. The mutex, not the render
// lock, guards it: PassStats is read from a test or a debug overlay on its
// own goroutine, and two apps in one process (two Managers) both record.
var passStats struct {
	mu    sync.Mutex
	ring  [passStatsCap]PassStat
	next  int // where the next stat goes
	count int // how many of ring are filled, up to passStatsCap
}

var passLogging atomic.Bool

// RecordPassStat adds one pass to the record, and logs it when pass logging
// is on. render.Manager calls it; a host that drives render passes itself
// may too. It does nothing when debug mode is off, so the callers' timing is
// the only cost they need to skip.
func RecordPassStat(s PassStat) {
	if !IsDebugMode() {
		return
	}
	passStats.mu.Lock()
	passStats.ring[passStats.next] = s
	passStats.next = (passStats.next + 1) % passStatsCap
	if passStats.count < passStatsCap {
		passStats.count++
	}
	passStats.mu.Unlock()

	if passLogging.Load() {
		log.Print(s.String())
	}
}

// PassStats returns the recorded passes, oldest first: the most recent 512
// while debug mode has been on.
func PassStats() []PassStat {
	passStats.mu.Lock()
	defer passStats.mu.Unlock()
	out := make([]PassStat, 0, passStats.count)
	start := (passStats.next - passStats.count + passStatsCap) % passStatsCap
	for i := 0; i < passStats.count; i++ {
		out = append(out, passStats.ring[(start+i)%passStatsCap])
	}
	return out
}

// ClearPassStats empties the record, as ClearConcerns empties the concern
// collector; a test measuring its own passes calls it first.
func ClearPassStats() {
	passStats.mu.Lock()
	defer passStats.mu.Unlock()
	passStats.next, passStats.count = 0, 0
	passStats.ring = [passStatsCap]PassStat{}
}

// SetPassLogging turns on a log line per recorded pass (PassStat.String), for
// watching a screen's cost while using it. It needs debug mode too, since
// that is what records the passes; it is a switch of its own because a line
// per pass is too much to have on whenever debug mode is.
func SetPassLogging(on bool) {
	passLogging.Store(on)
}
