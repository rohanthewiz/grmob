// Package grmobtest drives a GrMob app in a plain go test: render it, find a
// control on screen, tap or type into it the way a host does, and read what
// is on screen afterwards.
//
//	func TestCounter(t *testing.T) {
//		app := grmobtest.New(t, App)
//		app.Tap("+")
//		app.Tap("+")
//		if !app.HasText("Count: 2") {
//			t.Fatal("two taps did not reach 2")
//		}
//		app.NoConcerns()
//	}
//
// # What it drives
//
// A Driver wraps a render.Manager, the same engine every target runs, so a
// test needs no browser, simulator or device. Every read renders the current
// tree and decodes the JSON a host would receive, and every tap or keystroke
// dispatches the callback ID found in that tree. That is the discipline every
// host follows: callback IDs are assigned per pass, so an ID is always read
// from the tree it is about to act on, never kept from an earlier one.
//
// The decoded tree is a *core.Node, the type the wire format is marshalled
// from, so Style's fields read with their Go names and types. Props hold
// JSON's types after the round trip: numbers are float64, a callback is its
// ID string, and a nested value is a map[string]any or a []any.
//
// # Process-wide state
//
// Some of grmob's state outlives any one app: the device records (window,
// lifecycle, location, heading, audio), the pending host requests (clipboard,
// share, keystore, notification sweeps), the permission record and the debug
// concern collector. A test that leaves a location fix behind would hand it
// to the next test's first render, which is how a suite passes in file order
// and fails under go test -shuffle.
//
// New therefore calls Reset when it creates the only live Driver in the
// process, and the Driver's cleanup calls it again when the last one closes,
// so each test starts from a fresh process's state and leaves one behind.
// Two Drivers alive in one test share the state, as two screens in one app
// would. Reset is exported for tests that do not use a Driver.
//
// One consequence: a report a host makes at boot (a window size, a fold, a
// colour scheme) is sent after New and before the first read, not before
// New, which would reset it away. That is also when a host sends it, because
// New renders nothing; the first read is the first render.
//
//	app := grmobtest.New(t, App)
//	core.ReceiveWindow(core.Window{Width: 841, Height: 673})
//	app.Tree() // the first render reads the window just reported
//
// Tests that use a Driver must not call t.Parallel: the state above is one
// per process, so two tests running at once would read each other's.
//
// # Debug mode
//
// NoConcerns reads the debug audit (core.SetDebugMode), which records only
// while debug mode is on. Turn it on once for the package, in TestMain,
// before any app renders:
//
//	func TestMain(m *testing.M) {
//		core.SetDebugMode(true)
//		m.Run()
//	}
//
// NoConcerns fails the test when debug mode is off, rather than passing with
// nothing audited.
package grmobtest

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/internal/testreset"
	"github.com/rohanthewiz/grmob/render"
)

// AwaitTimeout is how long AwaitText waits before failing the test.
//
// Generous because it is a ceiling, not a cost: AwaitText returns as soon as
// the text is on screen, so a passing test waits only as long as the app
// takes. Under testing/synctest the wait is on the bubble's fake clock and
// costs no real time at all.
var AwaitTimeout = 5 * time.Second

// awaitPoll is how often AwaitText re-renders while it waits. Each poll is a
// whole render pass (audited under debug mode), so it is not made smaller
// than a timer test needs.
const awaitPoll = 25 * time.Millisecond

// Driver is one running app under test. Make one with New; it is closed by
// the test's cleanup.
type Driver struct {
	t   testing.TB
	mgr *render.Manager
	ctx *core.Context
}

// New renders root in a fresh core.Context and returns a Driver for it. The
// app is closed when the test ends.
func New(t testing.TB, root func(*core.Context) core.View) *Driver {
	t.Helper()
	return NewWithContext(t, core.NewContext(), root)
}

// NewWithContext is New with a Context the caller prepared, for an app that
// needs one: a theme (core.NewContext().WithTheme(theme)), or a test that
// keeps the Context to drive navigation or deep links itself.
func NewWithContext(t testing.TB, ctx *core.Context, root func(*core.Context) core.View) *Driver {
	t.Helper()
	acquire()
	mgr := render.New(ctx, root)
	// Close before release: closing the Context unsubscribes the app's hooks
	// from the device records, so the reset that release may run finds no
	// subscriber of this app's still attached.
	t.Cleanup(func() {
		mgr.Close()
		release()
	})
	return &Driver{t: t, mgr: mgr, ctx: ctx}
}

// Manager returns the render.Manager underneath, for a dispatch the Driver
// has no method for (DispatchIntCallback, DispatchTextEdit, Dispatch).
func (d *Driver) Manager() *render.Manager { return d.mgr }

// Context returns the app's Context.
func (d *Driver) Context() *core.Context { return d.ctx }

// Tree renders the app and returns the tree a host would receive.
func (d *Driver) Tree() *core.Node {
	d.t.Helper()
	return Parse(d.t, d.mgr.RenderInitial())
}

// Find returns the first node in the current tree, in depth-first order,
// that pred accepts, or nil.
func (d *Driver) Find(pred func(*core.Node) bool) *core.Node {
	d.t.Helper()
	return Find(d.Tree(), pred)
}

// FindAll returns every node in the current tree that pred accepts, in
// depth-first order.
func (d *Driver) FindAll(pred func(*core.Node) bool) []*core.Node {
	d.t.Helper()
	return FindAll(d.Tree(), pred)
}

// HasText reports whether a Text node in the current tree says exactly s.
func (d *Driver) HasText(s string) bool {
	d.t.Helper()
	return HasText(d.Tree(), s)
}

// Shows reports whether s appears anywhere in the words the current tree
// puts on screen. See the package-level Shows.
func (d *Driver) Shows(s string) bool {
	d.t.Helper()
	return Shows(d.Tree(), s)
}

// Tap presses the Button labelled label and returns the patches the pass
// produced. The test fails if no such Button is on screen.
func (d *Driver) Tap(label string) string {
	d.t.Helper()
	return d.TapWhere(Button(label), "a Button labelled "+quote(label))
}

// TapWhere presses the first node pred accepts that has an onClick, and
// returns the patches. what describes the node for the failure message
// ("the row titled Inbox").
//
// Only nodes with an onClick are considered, so a predicate that also matches
// the node's ancestors (a row is inside a list that shows the same words)
// lands on the one a finger would.
func (d *Driver) TapWhere(pred func(*core.Node) bool, what string) string {
	d.t.Helper()
	n := Find(d.Tree(), func(n *core.Node) bool {
		_, ok := n.Props["onClick"].(string)
		return ok && pred(n)
	})
	if n == nil {
		d.t.Fatalf("grmobtest: no tappable node on screen is %s", what)
	}
	return d.mgr.DispatchCallback(n.Props["onClick"].(string))
}

// Type delivers text to the field showing placeholder, as one change from the
// platform's text watcher (a paste, or the whole value after a keystroke),
// and returns the patches. The test fails if no field with that placeholder
// and an onChange is on screen.
//
// It sets the field's whole value; it does not append to what is there,
// because that is what a host sends.
func (d *Driver) Type(placeholder, text string) string {
	d.t.Helper()
	n := Find(d.Tree(), func(n *core.Node) bool {
		_, ok := n.Props["onChange"].(string)
		return ok && n.Props["placeholder"] == placeholder
	})
	if n == nil {
		d.t.Fatalf("grmobtest: no field with placeholder %s on screen", quote(placeholder))
	}
	return d.mgr.DispatchTextCallback(n.Props["onChange"].(string), text)
}

// Toggle sets the Checkbox or Switch named name to on, and returns the
// patches. A control's name is its accessible name
// (core.AccessibilityLabel), which is also what a screen reader says and what
// comps' rows give their control; a bare checkbox with no name cannot be told
// apart from another one, so Toggle refuses to guess and fails the test.
func (d *Driver) Toggle(name string, on bool) string {
	d.t.Helper()
	n := Find(d.Tree(), func(n *core.Node) bool {
		return (n.Type == "Checkbox" || n.Type == "Switch") && accessibleName(n) == name
	})
	if n == nil {
		d.t.Fatalf("grmobtest: no Checkbox or Switch named %s on screen", quote(name))
	}
	id, ok := n.Props["onToggle"].(string)
	if !ok {
		d.t.Fatalf("grmobtest: the %s named %s has no onToggle", n.Type, quote(name))
	}
	return d.mgr.DispatchBoolCallback(id, on)
}

// AwaitText re-renders until s is on screen (Shows), failing the test after
// AwaitTimeout. It is for text a timer, an effect or a goroutine puts there.
//
// It polls the tree rather than sleeping a fixed interval, so a test is as
// fast as the app and every poll is an audited render.
func (d *Driver) AwaitText(s string) {
	d.t.Helper()
	deadline := time.Now().Add(AwaitTimeout)
	for !d.Shows(s) {
		if time.Now().After(deadline) {
			d.t.Fatalf("grmobtest: %s was not on screen after %v", quote(s), AwaitTimeout)
		}
		time.Sleep(awaitPoll)
	}
}

// NoConcerns fails the test if the debug audit recorded anything (a hook
// called in a different order, a duplicate key, a handler that panicked),
// with the full dump. It fails too when debug mode is off, since the audit
// then records nothing and the check would pass vacuously.
func (d *Driver) NoConcerns() {
	d.t.Helper()
	NoConcerns(d.t)
}

// NoConcerns is Driver.NoConcerns for a test without a Driver.
func NoConcerns(t testing.TB) {
	t.Helper()
	if !core.IsDebugMode() {
		t.Fatal("grmobtest: NoConcerns needs debug mode, which is off, so nothing was audited. " +
			"Call core.SetDebugMode(true) in TestMain before any app renders.")
	}
	if cs := core.Concerns(); len(cs) != 0 {
		t.Fatalf("grmobtest: debug concerns raised:\n%s", core.DumpConcerns())
	}
}

// Parse decodes a tree as a host receives it (render.Manager.RenderInitial,
// mobile.RenderInitial), failing the test if it is not one.
func Parse(t testing.TB, wire string) *core.Node {
	t.Helper()
	var root core.Node
	if err := json.Unmarshal([]byte(wire), &root); err != nil {
		t.Fatalf("grmobtest: the tree is not valid JSON: %v", err)
	}
	return &root
}

// Reset returns grmob's process-wide state to a fresh process's: the device
// records and their subscriptions, every pending host request, the
// permission record, and the debug concern collector. The debug-mode switch
// itself is left alone, since TestMain sets it once for the package.
//
// New and the last Driver's cleanup call it, so a test using a Driver need
// not. Call it directly between tests that drive an app some other way (the
// mobile bridge, a bare render.Manager), and never while an app is running:
// it drops the running app's subscriptions to the records it clears.
func Reset() {
	testreset.All()
	core.ClearConcerns()
}

// live counts the Drivers not yet closed, so Reset runs only on the edges:
// when the first one opens and when the last one closes. A reset while a
// Driver is running would cut that app's hooks off from the records they
// subscribe to.
var (
	liveMu sync.Mutex
	live   int
)

func acquire() {
	liveMu.Lock()
	defer liveMu.Unlock()
	if live == 0 {
		Reset()
	}
	live++
}

func release() {
	liveMu.Lock()
	defer liveMu.Unlock()
	live--
	if live == 0 {
		Reset()
	}
}
