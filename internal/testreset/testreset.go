// Package testreset collects the functions that put grmob's process-wide
// state back the way a fresh process has it, so grmobtest.Reset can run them
// all from outside the packages that own the state.
//
// # Why a registry
//
// The state is spread over several packages (core's device records and
// pending host requests, keystore's pending calls, permission's statuses and
// foreground watches), and each package's reset is unexported, because it is
// not something an app should call. grmobtest cannot reach an unexported
// function, and exporting eleven ResetForTest functions would put a test
// seam on the API pages of three product packages.
//
// So each owning package registers its resets here from an init function,
// and grmobtest runs whatever is registered. That has a second property worth
// having: a package an app does not link registers nothing, so grmobtest
// never imports keystore or permission (and their host channels) into a test
// binary that did not already have them.
//
//	core ─────────┐
//	keystore ─────┼── Register(name, fn) at init ──▶ registry ◀── All() ── grmobtest.Reset
//	permission ───┘
//
// The package is internal, so only grmob's own packages can register, and the
// only caller of All outside tests is grmobtest.
package testreset

import (
	"sort"
	"sync"
)

var (
	mu     sync.Mutex
	resets []entry
)

type entry struct {
	name string
	fn   func()
}

// Register adds fn under name ("core.resetWindowForTest": the owning
// package's name and the function's, which is what the census test in this
// package matches against the source). Registering a name twice panics: two
// init functions claiming one reset is a wiring mistake, and running it twice
// would hide which one was meant.
func Register(name string, fn func()) {
	mu.Lock()
	defer mu.Unlock()
	for _, e := range resets {
		if e.name == name {
			panic("testreset: " + name + " registered twice")
		}
	}
	resets = append(resets, entry{name: name, fn: fn})
}

// All runs every registered reset, in registration order.
//
// The list is copied out under the lock and run outside it: a reset takes its
// own package's lock, and permission's reaches into core to cancel a
// lifecycle subscription, so holding this lock across them would order it
// before locks it has no business being ordered against.
func All() {
	mu.Lock()
	run := make([]func(), len(resets))
	for i, e := range resets {
		run[i] = e.fn
	}
	mu.Unlock()

	for _, fn := range run {
		fn()
	}
}

// Names returns the registered names, sorted, for the census test.
func Names() []string {
	mu.Lock()
	defer mu.Unlock()
	out := make([]string, len(resets))
	for i, e := range resets {
		out[i] = e.name
	}
	sort.Strings(out)
	return out
}
