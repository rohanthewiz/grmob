package tutorial

import (
	"testing"

	"github.com/rohanthewiz/grmob/render"
)

// Chapter 2 demo-liveness tests. Same discipline as the chapter-1 pair in
// app_test.go: drive the demo through dispatched callbacks exactly as a
// native shell would, re-reading the tree before every dispatch (callback IDs
// are per-pass), and let TestMain's debug mode audit every pass — the state
// lessons are the ones where a rules-of-hooks slip would actually bite.

// typeInto finds the first Input in tree order and dispatches its onChange
// with text, as the platform text watcher would. The chapter-2 screens hold
// at most one Input, so first-in-tree-order is unambiguous.
func typeInto(t *testing.T, mgr *render.Manager, text string) {
	t.Helper()
	n := findNode(tree(t, mgr), func(n *node) bool { return n.Type == "Input" })
	if n == nil {
		t.Fatal("no Input in the current tree")
	}
	mgr.DispatchTextCallback(n.Props["onChange"].(string), text)
}

// --- 2.1 The counter ------------------------------------------------------

func TestCounterDemoCounts(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "State: the counter")

	if !hasText(tree(t, mgr), "0") {
		t.Fatal("counter should start at 0")
	}
	for range 3 {
		tap(t, mgr, "+1")
	}
	if !hasText(tree(t, mgr), "3") {
		t.Fatal("three +1 taps should show 3")
	}
	tap(t, mgr, "−1")
	if !hasText(tree(t, mgr), "2") {
		t.Fatal("−1 should step the count back to 2")
	}
	tap(t, mgr, "Reset")
	if !hasText(tree(t, mgr), "0") {
		t.Fatal("Reset should return the count to 0")
	}
	assertNoConcerns(t)
}

// --- 2.2 Events & handlers ------------------------------------------------

// The demo card carries both gestures on one node; this drives each through
// its own registered callback and watches the log respond, then Clear empties
// it. Long-press rides the same void-callback channel as click, so the
// ordinary DispatchCallback works for both.
func TestEventsDemoLogsBothGestures(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Events & handlers")

	gestureCard := func() *node {
		n := findNode(tree(t, mgr), func(n *node) bool {
			_, hasLong := n.Props["onLongPress"].(string)
			return hasLong && hasText(n, "Tap or long-press me")
		})
		if n == nil {
			t.Fatal("no gesture card with an onLongPress handler in the tree")
		}
		return n
	}

	if !hasTextContaining(tree(t, mgr), "No events yet") {
		t.Fatal("the log should start empty")
	}
	mgr.DispatchCallback(gestureCard().Props["onClick"].(string))
	if !hasTextContaining(tree(t, mgr), "1 · tap") {
		t.Fatal("a click should log '1 · tap'")
	}
	mgr.DispatchCallback(gestureCard().Props["onLongPress"].(string))
	if !hasTextContaining(tree(t, mgr), "2 · long press") {
		t.Fatal("a long press should log '2 · long press'")
	}

	tap(t, mgr, "Clear log")
	if !hasTextContaining(tree(t, mgr), "No events yet") {
		t.Fatal("Clear log should empty the log again")
	}
	assertNoConcerns(t)
}

// --- 2.3 Controlled inputs ------------------------------------------------

func TestInputDemoEchoesTransformsAndClears(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Controlled inputs")

	if !hasTextContaining(tree(t, mgr), "Nothing typed yet") {
		t.Fatal("the echo should start on its empty-state caption")
	}

	typeInto(t, mgr, "gopher")
	cur := tree(t, mgr)
	if !hasText(cur, "Hello, gopher!") {
		t.Fatal("typing should echo through state into the greeting")
	}
	if !hasTextContaining(cur, "6 characters") {
		t.Fatal("the character count should read the same state")
	}

	// The transform runs on the way in: with UPPERCASE on, the next change
	// callback stores — and therefore displays — the uppercased value.
	toggleCheckbox(t, mgr, 0, true)
	typeInto(t, mgr, "gopher")
	if !hasText(tree(t, mgr), "Hello, GOPHER!") {
		t.Fatal("with the transform on, the stored value should be uppercased")
	}

	// Clear writes state from outside the field; the controlled field follows.
	tap(t, mgr, "Clear")
	if !hasTextContaining(tree(t, mgr), "Nothing typed yet") {
		t.Fatal("Clear should return the demo to its empty state")
	}
	assertNoConcerns(t)
}

// --- 2.4 Conditional rendering --------------------------------------------

func TestConditionalDemoMatchesStatus(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Conditional rendering")

	if !hasTextContaining(tree(t, mgr), "Fetching gophers") {
		t.Fatal("status should start on the Loading branch")
	}

	// The segments are chips, which render as Buttons — the tap helper
	// reaches them by label, as in the chapter-1 axis test.
	tap(t, mgr, "Error")
	cur := tree(t, mgr)
	if !hasText(cur, "Something went wrong") {
		t.Fatal("selecting Error should render the Default branch")
	}
	if hasTextContaining(cur, "Fetching gophers") {
		t.Fatal("Match renders one branch — Loading should be gone")
	}

	tap(t, mgr, "Ready")
	if !hasText(tree(t, mgr), "All systems go") {
		t.Fatal("selecting Ready should render the ready card")
	}

	// The raw-status line is a separate core.If, reading the same state.
	toggleCheckbox(t, mgr, 0, true)
	if !hasTextContaining(tree(t, mgr), "status = 1 (Ready)") {
		t.Fatal("the If branch should reveal the raw status value")
	}
	assertNoConcerns(t)
}

// --- 2.5 Lists & keys -----------------------------------------------------

// hasKey reports whether any node under n carries exactly this key — how the
// test sees what the reconciler sees, since Keyed stamps Node.Key.
func hasKey(n *node, key string) bool {
	return findNode(n, func(n *node) bool { return n.Key == key }) != nil
}

func TestListDemoAddsRemovesAndKeysRows(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Lists & keys")

	cur := tree(t, mgr)
	for _, title := range []string{"Feed the gopher", "Write some Go", "Ship the app"} {
		if !hasText(cur, title) {
			t.Fatalf("seed task %q missing from the list", title)
		}
	}
	for _, key := range []string{"task-1", "task-2", "task-3"} {
		if !hasKey(cur, key) {
			t.Fatalf("row key %q missing — For rows must be Keyed", key)
		}
	}

	tap(t, mgr, "＋ Add to top")
	if !hasKey(tree(t, mgr), "task-4") {
		t.Fatal("adding should insert a row keyed task-4")
	}

	// The first ✕ in tree order belongs to the top row — the one just added.
	tap(t, mgr, "✕")
	cur = tree(t, mgr)
	if hasText(cur, "Task 4") {
		t.Fatal("removing the top row should drop Task 4")
	}
	if !hasText(cur, "Feed the gopher") || !hasKey(cur, "task-1") {
		t.Fatal("removal must not disturb the remaining keyed rows")
	}
	assertNoConcerns(t)
}
