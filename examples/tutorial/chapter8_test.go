package tutorial

import (
	"fmt"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// Chapter 8's demos are the one place in the tutorial that deliberately
// provokes panics and concerns, inside a suite whose every test ends with
// assertNoConcerns. The tests here therefore run a stricter version of that
// discipline: after arming a provoker they assert the expected concern WAS
// filed (a positive assertion — the debug machinery checking itself), and
// they end by disarming, clearing the collector, driving one benign pass,
// and asserting it stayed empty.

// assertOnlyConcernKind requires at least one concern of the given kind and
// no concern of any other kind — a provoked failure must not smuggle in an
// unrelated finding (a cursor drift, say) under the cover of the expected
// one.
func assertOnlyConcernKind(t *testing.T, kind string) {
	t.Helper()
	list := core.Concerns()
	if len(list) == 0 {
		t.Fatalf("expected a %q concern, collector is empty", kind)
	}
	for _, c := range list {
		if c.Kind != kind {
			t.Fatalf("expected only %q concerns, found %q: %s", kind, c.Kind, c.Detail)
		}
	}
}

// --- 8.1 Error boundaries ---------------------------------------------------

func TestErrorBoundaryFallsBackAndHeals(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Error boundaries")

	// Healthy: the protected panel renders, no fallback in sight.
	root := tree(t, mgr)
	if !hasTextContaining(root, "Standup moved") {
		t.Fatal("the inbox panel should render while the data is intact")
	}
	if hasText(root, "Inbox is unavailable") {
		t.Fatal("no fallback should render while the child is healthy")
	}

	// Arm the planted bug: the panel's messages[0] read now panics, and the
	// boundary must swap in the fallback carrying the real panic message.
	toggleCheckbox(t, mgr, 0, true)
	root = tree(t, mgr)
	if !hasText(root, "Inbox is unavailable") {
		t.Fatal("the boundary should render the fallback when the child panics")
	}
	if !hasTextContaining(root, "index out of range") {
		t.Fatal("the fallback should show the actual panic via err.Error()")
	}
	if hasTextContaining(root, "Standup moved") {
		t.Fatal("the failed panel's own content must not survive into the tree")
	}
	// The catch is silent on screen but not to debug mode.
	assertOnlyConcernKind(t, core.ConcernRenderPanic)

	// Disarm: the boundary does not latch, so the very next pass heals.
	toggleCheckbox(t, mgr, 0, false)
	root = tree(t, mgr)
	if !hasTextContaining(root, "Standup moved") {
		t.Fatal("the panel should be back the pass after the state settles")
	}
	if hasText(root, "Inbox is unavailable") {
		t.Fatal("the fallback should be gone once the child renders clean")
	}

	core.ClearConcerns()
	_ = tree(t, mgr) // one benign pass: nothing may re-file
	assertNoConcerns(t)
}

// --- 8.2 Handler panics -----------------------------------------------------

func TestHandlerPanicSkewsStateAndIsRecovered(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "When a handler panics")

	// The handler completes: both counters move together.
	tap(t, mgr, "Advance both counters")
	root := tree(t, mgr)
	if !hasText(root, "A: 1") || !hasText(root, "B: 1") {
		t.Fatal("a completing handler should advance both counters")
	}
	if !hasTextContaining(root, "in step") {
		t.Fatal("the status line should report the counters in step")
	}

	// Arm the tripwire and tap again: the write before the panic sticks,
	// the write after it never runs — and the process survives, which is
	// the guard's whole job (an unguarded dispatch would kill the test
	// binary right here).
	toggleCheckbox(t, mgr, 0, true)
	tap(t, mgr, "Advance both counters")
	root = tree(t, mgr)
	if !hasText(root, "A: 2") || !hasText(root, "B: 1") {
		t.Fatal("the panic between the writes should leave the state half-applied")
	}
	if !hasTextContaining(root, "Skewed by 1") {
		t.Fatal("the status line should report the skew")
	}
	assertOnlyConcernKind(t, core.ConcernHandlerPanic)

	// Repair is app policy — the demo's repair button re-syncs.
	toggleCheckbox(t, mgr, 0, false)
	tap(t, mgr, "Repair (set B = A)")
	root = tree(t, mgr)
	if !hasText(root, "B: 2") || !hasTextContaining(root, "in step") {
		t.Fatal("repair should bring B back level with A")
	}

	core.ClearConcerns()
	_ = tree(t, mgr)
	assertNoConcerns(t)
}

// --- 8.3 Debug mode ---------------------------------------------------------

// The demo's checkbox order, used by both tests below: index 0 is the live
// SetDebugMode switch, index 1 renders the duplicate-key list.

func TestDebugLessonRecordsAndClearsConcerns(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Debug mode")

	if !hasTextContaining(tree(t, mgr), "None — the collector is empty") {
		t.Fatal("the inspector should start empty")
	}

	// Render the bad list: the duplicate-key check fires during the same
	// pass, and the inspector — rendered below the list in the tree — shows
	// the finding immediately.
	toggleCheckbox(t, mgr, 1, true)
	root := tree(t, mgr)
	if !hasTextContaining(root, "Row A — Keyed") {
		t.Fatal("the provoked list should actually render")
	}
	if !hasTextContaining(root, "[duplicate-key]") {
		t.Fatal("the inspector should list the duplicate-key concern")
	}

	// Stop provoking: the list goes, the recorded finding stays.
	toggleCheckbox(t, mgr, 1, false)
	root = tree(t, mgr)
	if hasTextContaining(root, "Row A — Keyed") {
		t.Fatal("the bad list should be gone once unchecked")
	}
	if !hasTextContaining(root, "[duplicate-key]") {
		t.Fatal("clearing is explicit — the concern should still be listed")
	}

	// The demo's own clear button empties the collector; with the provoker
	// off, nothing re-files on the following passes.
	tap(t, mgr, "Clear concerns")
	if !hasTextContaining(tree(t, mgr), "None — the collector is empty") {
		t.Fatal("the inspector should be empty after Clear concerns")
	}
	assertNoConcerns(t)
}

func TestDebugSwitchOffRecordsNothing(t *testing.T) {
	mgr := newApp(t)
	// The demo drives the real process-wide switch; make sure a failure
	// between here and the end cannot leave the rest of the suite unaudited.
	t.Cleanup(func() { core.SetDebugMode(true) })
	openLesson(t, mgr, "Debug mode")

	// Flip debug off through the demo's own switch, then render the bad
	// list: the zero-overhead claim, proven — the same duplicate keys
	// record nothing at all.
	toggleCheckbox(t, mgr, 0, false)
	toggleCheckbox(t, mgr, 1, true)
	root := tree(t, mgr)
	if !hasTextContaining(root, "Row A — Keyed") {
		t.Fatal("the bad list should render regardless of the debug switch")
	}
	if !hasTextContaining(root, "None — the collector is empty") {
		t.Fatal("with debug off, the duplicate keys must not be recorded")
	}
	if got := core.Concerns(); len(got) != 0 {
		t.Fatalf("with debug off the collector must stay empty, got %d", len(got))
	}

	// Disarm first, then re-enable the checks; the passes that follow are
	// audited again and must stay clean.
	toggleCheckbox(t, mgr, 1, false)
	toggleCheckbox(t, mgr, 0, true)
	_ = tree(t, mgr)
	assertNoConcerns(t)
}

// --- 8.4 Cached -------------------------------------------------------------

func TestCachedLessonRendersUnderDebugBypass(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Cached: freeze the static")

	root := tree(t, mgr)
	if !hasTextContaining(root, "Live — rendered at") {
		t.Fatal("the live stamp card should render")
	}
	if !hasTextContaining(root, "Cached — rendered at") {
		t.Fatal("the cached stamp card should render (debug mode bypasses the cache)")
	}
	// The lesson reads core.IsDebugMode() live and must say which mode the
	// reader is actually in — under the test suite, that is the bypass.
	if !hasTextContaining(root, "Cached is bypassed") {
		t.Fatal("under debug mode the demo should announce the cache bypass")
	}

	tap(t, mgr, "Force another pass")
	if !hasTextContaining(tree(t, mgr), "passes forced: 1") {
		t.Fatal("the pass counter should advance")
	}

	// The meaningful half of the audit: the cached probe is contract-clean
	// (no hooks, no callbacks, no theme reads), so even with the debug
	// bypass measuring it every pass, no cached-* concern may appear.
	assertNoConcerns(t)
}

// --- 8.5 Finale -------------------------------------------------------------

func TestFinaleReportsCurriculumAndToasts(t *testing.T) {
	// Recorder in place of a host, as in chapter 6's toast test; the
	// handler is process-wide, so restore it on cleanup.
	var toasts []string
	core.SetSystemEventHandler(func(name string, data map[string]any) {
		if name == "toast" {
			toasts = append(toasts, data["message"].(string))
		}
	})
	t.Cleanup(func() { core.SetSystemEventHandler(nil) })

	mgr := newApp(t)
	openLesson(t, mgr, "The whole model")

	// The stats are computed from the live curriculum, so they can never
	// drift from what the reader actually walked.
	want := fmt.Sprintf("%d lessons across %d chapters", len(flatLessons), len(Chapters))
	if !hasTextContaining(tree(t, mgr), want) {
		t.Fatalf("the finale should report %q", want)
	}

	tap(t, mgr, "Take a bow 🎉")
	if len(toasts) != 1 || toasts[0] != "Tutorial complete — now go build something" {
		t.Fatalf("expected the completion toast, got %v", toasts)
	}
	assertNoConcerns(t)
}
