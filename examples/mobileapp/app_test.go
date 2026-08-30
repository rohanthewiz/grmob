package mobileapp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/mobile"
)

// The package init has already run mobile.Register by the time tests execute,
// so these tests exercise exactly the call sequence the Kotlin shell makes.

type node struct {
	Type     string
	Props    map[string]any
	Children []*node
}

// findProp depth-first searches the tree for the first node of the given type
// and returns the named prop (callback IDs, values).
func findProp(n *node, nodeType, prop string) (string, bool) {
	if n == nil {
		return "", false
	}
	if n.Type == nodeType {
		if v, ok := n.Props[prop].(string); ok {
			return v, true
		}
	}
	for _, c := range n.Children {
		if v, ok := findProp(c, nodeType, prop); ok {
			return v, true
		}
	}
	return "", false
}

func TestDemoAppRendersAndDispatchesEvents(t *testing.T) {
	initial := mobile.RenderInitial()

	var tree node
	if err := json.Unmarshal([]byte(initial), &tree); err != nil {
		t.Fatalf("initial tree is not valid JSON: %v", err)
	}
	if !strings.Contains(initial, "Count: 0") {
		t.Fatalf("initial tree missing counter text:\n%s", initial)
	}

	// Void event: tapping the Increment button must patch the counter text.
	onClick, ok := findProp(&tree, "Button", "onClick")
	if !ok {
		t.Fatal("no Button with onClick in initial tree")
	}
	patches := mobile.TriggerCallback(onClick)
	if !strings.Contains(patches, "Count: 1") {
		t.Errorf("click patches don't update the counter:\n%s", patches)
	}

	// Int event: switching tabs must re-render with the new selectedIndex.
	onTabChange, ok := findProp(&tree, "TabView", "onTabChange")
	if !ok {
		t.Fatal("no TabView with onTabChange in initial tree")
	}
	patches = mobile.TriggerIntCallback(onTabChange, 1)
	if !strings.Contains(patches, `"selectedIndex":1`) {
		t.Errorf("tab-change patches don't carry the new selection:\n%s", patches)
	}

	// Text event: typing into the Input must patch the greeting.
	// Re-read the tree first — the callback IDs may have shifted with the
	// tab switch above, and the shell always dispatches IDs from its
	// current tree, never a stale one.
	var current node
	if err := json.Unmarshal([]byte(mobile.RenderInitial()), &current); err != nil {
		t.Fatalf("re-rendered tree is not valid JSON: %v", err)
	}
	onChange, ok := findProp(&current, "Input", "onChange")
	if !ok {
		t.Fatal("no Input with onChange in tree")
	}
	patches = mobile.TriggerTextCallback(onChange, "Ada")
	if !strings.Contains(patches, "Hello, Ada!") {
		t.Errorf("input patches don't update the greeting:\n%s", patches)
	}
}

func TestFeedTabListGestures(t *testing.T) {
	// The gap-5 surface end to end at the bridge level: the Feed tab holds a
	// List whose rows carry OnClick/OnLongPress behavior props — dispatching
	// them must land in Go state and patch the status line back out.
	mobile.TriggerIntCallback(mustFindT(t, "TabView", "onTabChange"), 2)

	patches := mobile.TriggerCallback(mustFindT(t, "Row", "onClick"))
	if !strings.Contains(patches, "Selected: Article 1") {
		t.Errorf("row tap patches don't update the selection status:\n%s", patches)
	}

	patches = mobile.TriggerCallback(mustFindT(t, "Row", "onLongPress"))
	if !strings.Contains(patches, "Starred: Article 1") {
		t.Errorf("row long-press patches don't update the starred status:\n%s", patches)
	}
	if !strings.Contains(patches, "Article 1 ★") {
		t.Errorf("row long-press patches don't restyle the row title:\n%s", patches)
	}

	// Leave the app back on the Counter tab so test order doesn't matter to
	// any later bridge-level test.
	mobile.TriggerIntCallback(mustFindT(t, "TabView", "onTabChange"), 0)
}

// mustFindT re-reads the current tree and returns the named prop of the first
// matching node — the same "dispatch only IDs from the live tree" discipline
// the shells follow.
func mustFindT(t *testing.T, nodeType, prop string) string {
	t.Helper()
	var tree node
	if err := json.Unmarshal([]byte(mobile.RenderInitial()), &tree); err != nil {
		t.Fatalf("tree is not valid JSON: %v", err)
	}
	v, ok := findProp(&tree, nodeType, prop)
	if !ok {
		t.Fatalf("no %s with %s in current tree", nodeType, prop)
	}
	return v
}
