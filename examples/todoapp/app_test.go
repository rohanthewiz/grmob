package todoapp

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/mobile"
)

// TestMain gives the whole suite a data directory before the first render,
// mirroring the shells (which call SetDataDir before RenderInitial). That
// routes every mutation in TestTodoLifecycle through the bytdb write-through
// path too, instead of leaving persistence to its dedicated test alone.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "todoapp-test-")
	if err != nil {
		panic(err)
	}
	mobile.SetDataDir(dir)
	code := m.Run()
	closeStore() // release the file lock before removing the directory
	os.RemoveAll(dir)
	os.Exit(code)
}

// The package init has already run mobile.Register by the time tests execute,
// so these tests exercise exactly the call sequence the native shells make:
// read the current tree, dispatch a callback ID found in it, assert on the
// patches (or the next full tree). The bridge is a process-wide singleton, so
// the whole user journey lives in one ordered test rather than independent
// tests that would fight over shared state.

type node struct {
	Type     string
	Props    map[string]any
	Children []*node
}

// findNode depth-first searches for the first node satisfying pred. Predicates
// match on type plus identifying props (a Button's label) because callback IDs
// are per-pass sequence numbers — position in the tree, not identity, is the
// only stable way to locate a widget.
func findNode(n *node, pred func(*node) bool) *node {
	if n == nil {
		return nil
	}
	if pred(n) {
		return n
	}
	for _, c := range n.Children {
		if found := findNode(c, pred); found != nil {
			return found
		}
	}
	return nil
}

// currentTree re-renders and parses the full tree. Every dispatch below reads
// its callback ID from a fresh tree — the "dispatch only IDs from the live
// tree" discipline the shells follow, since any state change can renumber IDs.
func currentTree(t *testing.T) *node {
	t.Helper()
	var tree node
	if err := json.Unmarshal([]byte(mobile.RenderInitial()), &tree); err != nil {
		t.Fatalf("tree is not valid JSON: %v", err)
	}
	return &tree
}

// mustCallback locates a widget in the live tree and returns the named
// callback ID, failing the test if the widget or the prop is missing.
func mustCallback(t *testing.T, pred func(*node) bool, prop, desc string) string {
	t.Helper()
	n := findNode(currentTree(t), pred)
	if n == nil {
		t.Fatalf("no node in current tree matching: %s", desc)
	}
	id, ok := n.Props[prop].(string)
	if !ok {
		t.Fatalf("node %s has no %q callback prop: %#v", desc, prop, n.Props)
	}
	return id
}

func byType(nodeType string) func(*node) bool {
	return func(n *node) bool { return n.Type == nodeType }
}

func buttonLabeled(label string) func(*node) bool {
	return func(n *node) bool {
		return n.Type == "Button" && n.Props["label"] == label
	}
}

// addTodo drives the two-step entry flow: type into the controlled Input,
// then commit with the Add button.
func addTodo(t *testing.T, title string) {
	t.Helper()
	mobile.TriggerTextCallback(mustCallback(t, byType("Input"), "onChange", "Input"), title)
	mobile.TriggerCallback(mustCallback(t, buttonLabeled("Add"), "onClick", `Button "Add"`))
}

func TestTodoLifecycle(t *testing.T) {
	initial := mobile.RenderInitial()
	if !strings.Contains(initial, "No tasks yet") {
		t.Fatalf("initial tree missing empty state:\n%s", initial)
	}
	if !strings.Contains(initial, "0 items left") {
		t.Fatalf("initial tree missing zero count:\n%s", initial)
	}

	// Typing must patch the controlled Input's value back out — that
	// round-trip is what keeps the native field and Go state in sync.
	patches := mobile.TriggerTextCallback(
		mustCallback(t, byType("Input"), "onChange", "Input"), "Buy milk")
	if !strings.Contains(patches, "Buy milk") {
		t.Errorf("typing patches don't echo the draft value:\n%s", patches)
	}

	// Committing the draft must create the row, clear the input, and bump
	// the remaining count.
	patches = mobile.TriggerCallback(
		mustCallback(t, buttonLabeled("Add"), "onClick", `Button "Add"`))
	if !strings.Contains(patches, "Buy milk") {
		t.Errorf("add patches don't contain the new row:\n%s", patches)
	}
	if !strings.Contains(patches, "1 item left") {
		t.Errorf("add patches don't update the count:\n%s", patches)
	}

	// A blank draft must be a no-op: same tree before and after the tap.
	before := mobile.RenderInitial()
	mobile.TriggerCallback(mustCallback(t, buttonLabeled("Add"), "onClick", `Button "Add"`))
	if after := mobile.RenderInitial(); after != before {
		t.Errorf("adding a blank draft changed the tree:\nbefore: %s\nafter: %s", before, after)
	}

	addTodo(t, "Walk dog")
	if tree := mobile.RenderInitial(); !strings.Contains(tree, "2 items left") {
		t.Fatalf("second add didn't land:\n%s", tree)
	}

	// Bool event: checking the first row's checkbox marks "Buy milk" done,
	// which drops the remaining count and reveals the bulk-clear button.
	patches = mobile.TriggerBoolCallback(
		mustCallback(t, byType("Checkbox"), "onToggle", "Checkbox"), true)
	if !strings.Contains(patches, "1 item left") {
		t.Errorf("toggle patches don't update the count:\n%s", patches)
	}
	if tree := mobile.RenderInitial(); !strings.Contains(tree, "Clear completed") {
		t.Errorf("completing a task didn't reveal Clear completed:\n%s", tree)
	}

	// Filters are pure view state: Done shows only the completed todo,
	// Active only the open one, and the data itself is untouched.
	mobile.TriggerCallback(mustCallback(t, buttonLabeled("Done"), "onClick", `Button "Done"`))
	tree := mobile.RenderInitial()
	if !strings.Contains(tree, "Buy milk") || strings.Contains(tree, "Walk dog") {
		t.Errorf("Done filter shows the wrong rows:\n%s", tree)
	}

	mobile.TriggerCallback(mustCallback(t, buttonLabeled("Active"), "onClick", `Button "Active"`))
	tree = mobile.RenderInitial()
	if !strings.Contains(tree, "Walk dog") || strings.Contains(tree, "Buy milk") {
		t.Errorf("Active filter shows the wrong rows:\n%s", tree)
	}

	// Deleting under the Active filter must remove "Walk dog" everywhere,
	// not just from the filtered view — the visible slice is derived, so the
	// row's closure has to address the todo by ID, not by position.
	mobile.TriggerCallback(mustCallback(t, buttonLabeled("✕"), "onClick", `Button "✕"`))
	if tree = mobile.RenderInitial(); !strings.Contains(tree, "No active tasks") {
		t.Errorf("delete under Active filter didn't empty the view:\n%s", tree)
	}

	mobile.TriggerCallback(mustCallback(t, buttonLabeled("All"), "onClick", `Button "All"`))
	tree = mobile.RenderInitial()
	if strings.Contains(tree, "Walk dog") {
		t.Errorf("deleted todo still present in All view:\n%s", tree)
	}
	if !strings.Contains(tree, "Buy milk") {
		t.Errorf("delete removed the wrong todo:\n%s", tree)
	}

	// Bulk clear removes the remaining (done) todo and, with nothing done
	// anymore, must take its own button with it.
	mobile.TriggerCallback(mustCallback(t, buttonLabeled("Clear completed"), "onClick", `Button "Clear completed"`))
	tree = mobile.RenderInitial()
	if strings.Contains(tree, "Buy milk") {
		t.Errorf("Clear completed left a done todo behind:\n%s", tree)
	}
	if strings.Contains(tree, "Clear completed") {
		t.Errorf("Clear completed button lingers with nothing to clear:\n%s", tree)
	}
	if !strings.Contains(tree, "No tasks yet") {
		t.Errorf("emptied list doesn't show the empty state:\n%s", tree)
	}

	// Submit path: the input's onSubmit is a void callback carrying the same
	// commit as the Add button, so typing then dispatching it must create the
	// row and clear the draft without the button being involved.
	mobile.TriggerTextCallback(mustCallback(t, byType("Input"), "onChange", "Input"), "Via enter")
	patches = mobile.TriggerCallback(mustCallback(t, byType("Input"), "onSubmit", "Input onSubmit"))
	if !strings.Contains(patches, "Via enter") {
		t.Errorf("submit patches don't contain the new row:\n%s", patches)
	}
	if !strings.Contains(patches, `"value":""`) {
		t.Errorf("submit patches don't clear the draft:\n%s", patches)
	}
	if !strings.Contains(patches, "1 item left") {
		t.Errorf("submit patches don't update the count:\n%s", patches)
	}
}

// relaunch simulates the app process being killed and started again: close
// the store (drop the file lock and the in-memory snapshot) and register a
// fresh context, so the next render hydrates from disk exactly as a cold
// start would. The data directory carries over — that's the disk.
func relaunch(t *testing.T) {
	t.Helper()
	closeStore()
	mobile.Register(core.NewContext(), App)
}

func TestTodoPersistence(t *testing.T) {
	// A fresh directory isolates this test from whatever TestTodoLifecycle
	// left in the TestMain-wide store; openStore notices the changed path
	// and reopens there. The fresh Register drops the previous test's
	// in-memory state for the same reason.
	mobile.SetDataDir(t.TempDir())
	mobile.Register(core.NewContext(), App)

	if tree := mobile.RenderInitial(); !strings.Contains(tree, "No tasks yet") {
		t.Fatalf("fresh store isn't empty:\n%s", tree)
	}

	addTodo(t, "First task")
	addTodo(t, "Second task")
	mobile.TriggerBoolCallback(
		mustCallback(t, byType("Checkbox"), "onToggle", "Checkbox"), true)

	// Relaunch #1: both rows, the done flag, and the derived count must all
	// come back from disk, not from the (discarded) context state.
	relaunch(t)
	tree := mobile.RenderInitial()
	for _, want := range []string{"First task", "Second task", "1 item left", "Clear completed"} {
		if !strings.Contains(tree, want) {
			t.Errorf("relaunched tree missing %q:\n%s", want, tree)
		}
	}

	// The ID sequence must resume past persisted rows. If it restarted at 1,
	// the next add would collide with "First task", and deleting the first
	// row (the first ✕ in the tree, list is id-ordered) would take the new
	// row with it.
	addTodo(t, "Third task")
	mobile.TriggerCallback(mustCallback(t, buttonLabeled("✕"), "onClick", `Button "✕"`))
	tree = mobile.RenderInitial()
	if strings.Contains(tree, "First task") {
		t.Errorf("deleting the first row didn't remove First task:\n%s", tree)
	}
	if !strings.Contains(tree, "Third task") {
		t.Errorf("post-relaunch add shares an ID with a persisted row (deleted with it):\n%s", tree)
	}

	// Relaunch #2: the delete and the new row survive. (The done row went
	// with the delete, so nothing is completed at this point.)
	relaunch(t)
	tree = mobile.RenderInitial()
	if strings.Contains(tree, "First task") {
		t.Errorf("deleted row came back after relaunch:\n%s", tree)
	}
	if !strings.Contains(tree, "Third task") {
		t.Errorf("row added after relaunch wasn't persisted:\n%s", tree)
	}

	// Bulk clear persists too: complete "Second task" (now the first row,
	// hence the first checkbox), clear, and the clear must hold across a
	// relaunch while the untouched row stays.
	mobile.TriggerBoolCallback(
		mustCallback(t, byType("Checkbox"), "onToggle", "Checkbox"), true)
	mobile.TriggerCallback(mustCallback(t, buttonLabeled("Clear completed"), "onClick", `Button "Clear completed"`))
	relaunch(t)
	tree = mobile.RenderInitial()
	if strings.Contains(tree, "Second task") {
		t.Errorf("cleared row came back after relaunch:\n%s", tree)
	}
	if !strings.Contains(tree, "Third task") {
		t.Errorf("clear-completed removed an active row from disk:\n%s", tree)
	}
}
