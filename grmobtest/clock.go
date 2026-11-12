package grmobtest

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"
)

// SettleLimit is how much real time WithFakeClock gives a bubble before it
// decides the bubble will never settle. A bubble that works settles in
// milliseconds of real time whatever its fake clock reads, so this only has
// to outlast a slow, loaded CI machine running under -race.
var SettleLimit = 30 * time.Second

// WithFakeClock runs fn in a testing/synctest bubble, where time is fake:
// time.Sleep, timers, tickers and every hook built on them (UseInterval,
// UseTimeout, UseDebounce, UseNow, AwaitText's polling) wait on a clock that
// jumps ahead whenever every goroutine in the bubble is waiting. A test of a
// two-second timeout costs no real time, and cannot flake on a slow machine.
//
//	grmobtest.WithFakeClock(t, func(t *testing.T) {
//		app := grmobtest.New(t, App)  // inside: the app's timers are fake too
//		app.Tap("Start")
//		app.AwaitText("Done")         // 2 s of fake time, ~0 s of real time
//	})
//
// Make the Driver inside fn, as above: goroutines started outside the bubble
// (a Driver made before it, and its timers) keep real time.
//
// # Why not synctest.Test directly
//
// It is synctest.Test plus a watchdog. In a bubble, a wait that never ends
// is a hot loop rather than a stall, because the fake clock jumps to the
// next timer the moment everything is waiting. The case that matters is a
// test that drains a channel until it goes quiet while a ticker nothing
// stopped keeps feeding it: a regression in a hook's Close path does exactly
// that to the hook's own test (UseInterval's, for one). Under bare
// synctest.Test that spins until go test's timeout, ten minutes by default;
// here it is a panic after SettleLimit of real time that names the test and
// ends the binary with every goroutine's stack, the runaway ticker's among
// them. (A goroutine still blocked when fn returns is synctest's own
// "deadlock" panic, at once; the watchdog is for the bubble that never
// gets that far.)
func WithFakeClock(t *testing.T, fn func(t *testing.T)) {
	t.Helper()
	name := t.Name()
	// Created outside the bubble, so it runs on the real clock.
	watchdog := time.AfterFunc(SettleLimit, func() {
		panic(fmt.Sprintf("grmobtest.WithFakeClock: %s's bubble did not settle in %v of real time. "+
			"Something in it waits without end on the fake clock; the usual cause is a wait for "+
			"quiet that a ticker or timer nothing stopped keeps interrupting (a Context that was "+
			"never closed, or a hook whose Close no longer stops it). The stacks below include it.",
			name, SettleLimit))
	})
	defer watchdog.Stop()
	synctest.Test(t, fn)
}
