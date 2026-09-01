package tutorial

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// Chapter 6 demo-liveness tests. The navigation lessons drive the app's real
// Navigator — the pushed demo screens are genuine frames on the genuine
// stack — so these tests assert on the same things a user sees: which screen
// is on top, what the depth telemetry reads, and which state survived the
// trip. The modal test additionally exercises the one dispatch path no other
// chapter uses (onDismiss, a void callback on a non-Button node), and the
// toast test exercises core's system-event channel end to end.

// modalNode finds the lesson's Modal in the current tree. There is exactly
// one per screen in this chapter, so type alone addresses it.
func modalNode(t *testing.T, root *node) *node {
	t.Helper()
	n := findNode(root, func(n *node) bool { return n.Type == "Modal" })
	if n == nil {
		t.Fatal("no Modal node in the current tree")
	}
	return n
}

// --- 6.1 Screens are a stack ------------------------------------------------

func TestStackPushPopKeepsCoveredState(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Screens are a stack")

	// The lesson sits on top of the contents screen: depth 2.
	if !hasTextContaining(tree(t, mgr), "StackDepth 2 · CanPop true") {
		t.Fatal("the lesson's telemetry should read depth 2")
	}

	// Give the lesson frame a value to recognize after the round trip.
	tap(t, mgr, "+1")
	tap(t, mgr, "+1")
	if !hasTextContaining(tree(t, mgr), "This lesson frame's taps: 2") {
		t.Fatal("the lesson counter should have counted to 2")
	}

	// Push: the detail screen is a new frame with its own slot 0.
	tap(t, mgr, "Push the detail screen ›")
	cur := tree(t, mgr)
	if !hasText(cur, "The detail screen") {
		t.Fatal("the detail screen should be on top after the push")
	}
	if !hasTextContaining(cur, "StackDepth 3") {
		t.Fatal("the push should have deepened the stack to 3")
	}
	if !hasTextContaining(cur, "This frame's taps: 0") {
		t.Fatal("the fresh frame's counter should start at 0")
	}
	tap(t, mgr, "+1")
	if !hasTextContaining(tree(t, mgr), "This frame's taps: 1") {
		t.Fatal("the detail counter should count in its own frame")
	}

	// Pop: the covered lesson frame comes back exactly as it was left.
	tap(t, mgr, "‹ Pop back to the lesson")
	if !hasTextContaining(tree(t, mgr), "This lesson frame's taps: 2") {
		t.Fatal("popping back should find the lesson counter untouched at 2")
	}

	// Re-push: a new frame, a new scope — the detail counter did not survive.
	tap(t, mgr, "Push the detail screen ›")
	if !hasTextContaining(tree(t, mgr), "This frame's taps: 0") {
		t.Fatal("a re-push must mint a fresh frame — the old count must be gone")
	}
	tap(t, mgr, "‹ Pop back to the lesson")
	assertNoConcerns(t)
}

// --- 6.2 Replace ------------------------------------------------------------

func TestReplaceKeepsDepthAndDiscardsTheStep(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Replace: steps with no way back")

	tap(t, mgr, "Start checkout ›")
	cur := tree(t, mgr)
	if !hasText(cur, "Checkout — step 1") || !hasTextContaining(cur, "StackDepth 3") {
		t.Fatal("checkout step 1 should be on top at depth 3")
	}

	// Fill the step, then Replace it away.
	typeField(t, mgr, "A line for the card", "for June")
	if fieldByPlaceholder(t, mgr, "A line for the card").Props["value"] != "for June" {
		t.Fatal("the note field should echo what was typed")
	}
	tap(t, mgr, "Place the order")
	cur = tree(t, mgr)
	if !hasText(cur, "Order confirmed") {
		t.Fatal("Replace should have swapped in the confirmation screen")
	}
	// The defining property: same depth as the step it replaced.
	if !hasTextContaining(cur, "StackDepth 3") {
		t.Fatal("Replace must not deepen the stack")
	}

	// One Pop from the confirmation lands on the lesson — step 1 is not in
	// the history.
	tap(t, mgr, "‹ Back to the lesson")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "Replace: steps with no way back") {
		t.Fatal("popping the confirmation should land on the lesson")
	}
	if hasText(cur, "Checkout — step 1") {
		t.Fatal("the replaced step must not be reachable by walking back")
	}

	// Re-entering runs a fresh frame: the note died with the old one.
	tap(t, mgr, "Start checkout ›")
	if fieldByPlaceholder(t, mgr, "A line for the card").Props["value"] != "" {
		t.Fatal("a re-entered checkout must start with an empty note")
	}
	tap(t, mgr, "‹ Abandon checkout")
	assertNoConcerns(t)
}

// --- 6.3 PopToRoot ----------------------------------------------------------

func TestDrillDownPopToRootLandsOnContents(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Unwinding: PopToRoot vs Reset")

	tap(t, mgr, "Enter the drill-down ›")
	cur := tree(t, mgr)
	if !hasText(cur, "Drill-down — level 1") || !hasTextContaining(cur, "StackDepth 3") {
		t.Fatal("level 1 should sit at depth 3")
	}

	// Each Deeper is a Push of drillRoute(level+1) — parameters by closure.
	tap(t, mgr, "Deeper ›")
	tap(t, mgr, "Deeper ›")
	cur = tree(t, mgr)
	if !hasText(cur, "Drill-down — level 3") || !hasTextContaining(cur, "StackDepth 5") {
		t.Fatal("two Deepers should reach level 3 at depth 5")
	}

	// Pop unwinds one level at a time…
	tap(t, mgr, "‹ Pop one level")
	cur = tree(t, mgr)
	if !hasText(cur, "Drill-down — level 2") || !hasTextContaining(cur, "StackDepth 4") {
		t.Fatal("one Pop should step back to level 2 at depth 4")
	}

	// …and Done unwinds everything: the levels AND the lesson, landing on the
	// tutorial's root frame — the contents screen, progress intact because it
	// lives above the Navigator.
	tap(t, mgr, "Done — back to contents")
	cur = tree(t, mgr)
	if !hasText(cur, "GrMob Interactive Tutorial") {
		t.Fatal("PopToRoot should land on the contents screen")
	}
	if !hasTextContaining(cur, "1 of ") {
		t.Fatal("progress should have survived the unwind")
	}
	assertNoConcerns(t)
}

// --- 6.4 Modal --------------------------------------------------------------

func TestModalIsControlledAndItsContentPersists(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Modal: the overlay that hides")

	// Closed, but present: the content renders every pass regardless of
	// Visible — hiding is a prop, not an unmount.
	cur := tree(t, mgr)
	m := modalNode(t, cur)
	if m.Props["visible"] != false {
		t.Fatal("the modal should start closed")
	}
	if !hasText(m, "Confirm your order") {
		t.Fatal("the dialog content should be in the tree even while hidden")
	}

	// Opening is a state write reflected as a prop flip — the stack is not
	// involved, so the depth telemetry still reads the lesson's own depth.
	tap(t, mgr, "Open the modal")
	cur = tree(t, mgr)
	if modalNode(t, cur).Props["visible"] != true {
		t.Fatal("Open should flip Visible to true")
	}
	if !hasTextContaining(cur, "StackDepth 2") {
		t.Fatal("an open modal must not touch the navigation stack")
	}

	// Tick the box, confirm, and the dialog closes through app state.
	toggleCheckbox(t, mgr, 0, true)
	tap(t, mgr, "Confirm")
	cur = tree(t, mgr)
	if modalNode(t, cur).Props["visible"] != false {
		t.Fatal("Confirm should close the dialog")
	}
	if !hasTextContaining(cur, "Order confirmed, gift receipt included") {
		t.Fatal("the confirmed decision should reflect the ticked box")
	}

	// Reopen: the checkbox is exactly as it was left — modal content has no
	// frame to die with, the opposite of 6.1's detail screen.
	tap(t, mgr, "Open the modal")
	cur = tree(t, mgr)
	cb := findNode(cur, func(n *node) bool { return n.Type == "Checkbox" })
	if cb == nil || cb.Props["checked"] != true {
		t.Fatal("the dialog's state should survive a close and reopen")
	}

	// The backdrop tap arrives as OnDismiss — a void callback the app answers
	// by writing state, exactly like Cancel.
	id, ok := modalNode(t, cur).Props["onDismiss"].(string)
	if !ok {
		t.Fatal("an open modal with OnDismiss should carry its callback id")
	}
	mgr.DispatchCallback(id)
	if modalNode(t, tree(t, mgr)).Props["visible"] != false {
		t.Fatal("the backdrop tap should have closed the dialog")
	}
	assertNoConcerns(t)
}

// --- 6.5 Toast --------------------------------------------------------------

func TestToastEmitsSystemEvents(t *testing.T) {
	// The recorder stands where a host stands. The handler is process-wide by
	// design (see core/sys_events.go), so it is restored on cleanup — leaving
	// it installed would leak this test's slice into every later toast.
	type event struct {
		name string
		data map[string]any
	}
	var events []event
	core.SetSystemEventHandler(func(name string, data map[string]any) {
		events = append(events, event{name, data})
	})
	t.Cleanup(func() { core.SetSystemEventHandler(nil) })

	mgr := newApp(t)
	openLesson(t, mgr, "Toast: fire and forget")

	// Rendering the lesson fires nothing: ShowToast lives in handlers only.
	if len(events) != 0 {
		t.Fatalf("rendering must not emit toasts, got %d", len(events))
	}

	tap(t, mgr, "Show a toast")
	if len(events) != 1 || events[0].name != "toast" {
		t.Fatalf("expected one toast event, got %+v", events)
	}
	if events[0].data["message"] != "Nicely done — that landed" {
		t.Fatalf("wrong message: %v", events[0].data["message"])
	}
	if events[0].data["duration"] != 2000 {
		t.Fatalf("the default duration should be 2000 ms, got %v", events[0].data["duration"])
	}

	tap(t, mgr, "Linger for five seconds")
	if len(events) != 2 || events[1].data["duration"] != 5000 {
		t.Fatalf("Duration(5000) should reach the host, got %+v", events)
	}

	tap(t, mgr, "Show it styled")
	if len(events) != 3 || events[2].data["style"] == nil {
		t.Fatalf("UseToastStyle should ride along in the payload, got %+v", events)
	}

	// The only trace in the tree is the demo's own counter.
	if !hasTextContaining(tree(t, mgr), "Toasts sent this visit: 3") {
		t.Fatal("the lesson's counter should have seen all three")
	}
	assertNoConcerns(t)
}
