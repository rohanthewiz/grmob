package tutorial

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// TestMain turns on debug mode for the whole package, following the signup
// example's discipline: every render pass driven below is audited for cursor
// drift, duplicate keys, and dropped container items, and each test asserts
// the audit came back empty. The tutorial is the densest hook user in
// examples/ — every lesson demo owns state — so this audit is the test's
// real value: a lesson whose Body calls hooks conditionally fails here, not
// on a device.
func TestMain(m *testing.M) {
	core.SetDebugMode(true)
	m.Run()
}

type node struct {
	Type     string
	Props    map[string]any
	Children []*node
}

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

// hasText reports whether any Text node under n carries exactly this content.
func hasText(n *node, content string) bool {
	return findNode(n, func(n *node) bool {
		return n.Type == "Text" && n.Props["content"] == content
	}) != nil
}

// hasTextContaining matches on substring — for labels assembled with
// Sprintf, where pinning the whole string would make every wording tweak a
// test edit.
func hasTextContaining(n *node, sub string) bool {
	return findNode(n, func(n *node) bool {
		s, ok := n.Props["content"].(string)
		return ok && n.Type == "Text" && strings.Contains(s, sub)
	}) != nil
}

// tree re-renders and parses the current tree. Callback IDs are per-pass
// sequence numbers, so every dispatch reads its ID from a freshly rendered
// tree rather than reusing one — the discipline the native shells follow.
func tree(t *testing.T, mgr *render.Manager) *node {
	t.Helper()
	var root node
	if err := json.Unmarshal([]byte(mgr.RenderInitial()), &root); err != nil {
		t.Fatalf("tree is not valid JSON: %v", err)
	}
	return &root
}

// tap finds the button with this label and dispatches its click, as a native
// tap would.
func tap(t *testing.T, mgr *render.Manager, label string) {
	t.Helper()
	n := findNode(tree(t, mgr), func(n *node) bool {
		return n.Type == "Button" && n.Props["label"] == label
	})
	if n == nil {
		t.Fatalf("no Button labeled %q in the current tree", label)
	}
	mgr.DispatchCallback(n.Props["onClick"].(string))
}

// openLesson taps the contents row whose subtree shows the lesson's title.
// A row is any clickable node (ListRow registers OnClick only when OnTap is
// set) with the title Text beneath it — matching by structure rather than by
// an ID prop, since the tree carries none.
func openLesson(t *testing.T, mgr *render.Manager, title string) {
	t.Helper()
	n := findNode(tree(t, mgr), func(n *node) bool {
		_, clickable := n.Props["onClick"].(string)
		return clickable && n.Type != "Button" && hasText(n, title)
	})
	if n == nil {
		t.Fatalf("no tappable contents row titled %q", title)
	}
	mgr.DispatchCallback(n.Props["onClick"].(string))
}

// toggleCheckbox flips the idx-th checkbox in tree order. The demos'
// checkboxes carry no distinguishing props, so position is the only address —
// the same walk signup uses for its second password field.
func toggleCheckbox(t *testing.T, mgr *render.Manager, idx int, on bool) {
	t.Helper()
	var found []*node
	var walk func(n *node)
	walk = func(n *node) {
		if n == nil {
			return
		}
		if n.Type == "Checkbox" {
			found = append(found, n)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(tree(t, mgr))
	if idx >= len(found) {
		t.Fatalf("wanted checkbox %d, tree has %d", idx, len(found))
	}
	mgr.DispatchBoolCallback(found[idx].Props["onToggle"].(string), on)
}

func assertNoConcerns(t *testing.T) {
	t.Helper()
	if cs := core.Concerns(); len(cs) != 0 {
		t.Fatalf("debug concerns raised:\n%s", core.DumpConcerns())
	}
}

func newApp(t *testing.T) *render.Manager {
	t.Helper()
	core.ClearConcerns() // the collector is process-wide; do not inherit
	mgr := render.New(core.NewContext().WithTheme(core.DefaultTheme), App)
	t.Cleanup(mgr.Close)
	return mgr
}

// --- The contents screen -------------------------------------------------

func TestHomeListsEveryLesson(t *testing.T) {
	mgr := newApp(t)
	root := tree(t, mgr)

	if !hasText(root, "GrMob Interactive Tutorial") {
		t.Fatal("home is missing its title")
	}
	for _, e := range flatLessons {
		if !hasText(root, e.Title) {
			t.Errorf("home is missing lesson %s (%s)", e.ID, e.Title)
		}
		if !hasText(root, e.ID) {
			t.Errorf("home is missing the %s ordinal", e.ID)
		}
	}
	want := fmt.Sprintf("0 of %d lessons opened", len(flatLessons))
	if !hasText(root, want) {
		t.Fatalf("home is missing the progress caption %q", want)
	}
	assertNoConcerns(t)
}

// --- Opening a lesson, and coming back -----------------------------------

func TestOpenLessonMarksProgressAndPopsBack(t *testing.T) {
	mgr := newApp(t)

	openLesson(t, mgr, "Hello, GrMob")
	lesson := tree(t, mgr)
	if !hasTextContaining(lesson, "1.1  Hello, GrMob") {
		t.Fatal("lesson screen did not open on 1.1")
	}
	if !hasText(lesson, "TRY IT") {
		t.Fatal("lesson screen is missing its demo panel")
	}
	// First lesson: no Prev, and the chapter tag names its chapter.
	if findNode(lesson, func(n *node) bool { return n.Props["label"] == "‹ Prev" }) != nil {
		t.Fatal("first lesson should not offer Prev")
	}
	if !hasTextContaining(lesson, "Chapter 1 · Views & Layout") {
		t.Fatal("lesson screen is missing its chapter tag")
	}

	tap(t, mgr, "‹ Contents")
	home := tree(t, mgr)
	want := fmt.Sprintf("1 of %d lessons opened", len(flatLessons))
	if !hasText(home, want) {
		t.Fatalf("after opening one lesson, home should say %q", want)
	}
	if !hasText(home, "opened") {
		t.Fatal("the opened lesson's row should carry an 'opened' badge")
	}
	assertNoConcerns(t)
}

// --- Walking the whole curriculum with Next ------------------------------

func TestNextWalksEveryLessonAndFinishes(t *testing.T) {
	mgr := newApp(t)

	openLesson(t, mgr, flatLessons[0].Title)
	for i, e := range flatLessons {
		cur := tree(t, mgr)
		if !hasTextContaining(cur, e.ID+"  "+e.Title) {
			t.Fatalf("step %d: expected lesson %s (%s) on screen", i, e.ID, e.Title)
		}
		if i < len(flatLessons)-1 {
			tap(t, mgr, "Next ›")
		}
	}

	// The last lesson offers Finish instead of Next, and Finish pops home.
	tap(t, mgr, "Finish ✓")
	home := tree(t, mgr)
	want := fmt.Sprintf("%d of %d lessons opened", len(flatLessons), len(flatLessons))
	if !hasText(home, want) {
		t.Fatalf("after the full walk, home should say %q", want)
	}
	assertNoConcerns(t)
}

func TestPrevStepsBack(t *testing.T) {
	mgr := newApp(t)

	openLesson(t, mgr, flatLessons[0].Title)
	tap(t, mgr, "Next ›")
	if !hasTextContaining(tree(t, mgr), flatLessons[1].Title) {
		t.Fatal("Next did not reach lesson 2")
	}
	tap(t, mgr, "‹ Prev")
	if !hasTextContaining(tree(t, mgr), flatLessons[0].Title) {
		t.Fatal("Prev did not return to lesson 1")
	}
	assertNoConcerns(t)
}

// --- The demos are live --------------------------------------------------

// Lesson 1.1's composition toggles: unchecking Stats() must drop that
// subtree from the tree — the demo's whole claim.
func TestHelloDemoRecomposes(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Hello, GrMob")

	if !hasText(tree(t, mgr), "1.2k") {
		t.Fatal("stats block should render while its toggle is on")
	}
	toggleCheckbox(t, mgr, 1, false) // 0: Header, 1: Stats, 2: Bio
	if hasText(tree(t, mgr), "1.2k") {
		t.Fatal("unchecking Stats() should remove the stats subtree")
	}
	toggleCheckbox(t, mgr, 2, true)
	if !hasTextContaining(tree(t, mgr), "without leaving Go") {
		t.Fatal("checking Bio should add the bio text")
	}
	assertNoConcerns(t)
}

// Lesson 1.3's axis switch swaps the demo container between Row and Column.
// The demo container is identified by its Gap changing with the stepper —
// so this drives both controls and watches the tree respond.
func TestStacksDemoSwitchesAxis(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Rows, Columns & spacing")

	// The A/B/C boxes start out inside a Row (axis 0).
	boxRow := func(root *node, typ string) *node {
		return findNode(root, func(n *node) bool {
			return n.Type == typ && hasText(n, "A") && hasText(n, "B") && hasText(n, "C")
		})
	}
	if boxRow(tree(t, mgr), "Row") == nil {
		t.Fatal("demo boxes should start in a Row")
	}

	// Flip the axis via the segmented control's "Column" chip. A Chip
	// renders as a core.Button whose label is the caption, so the ordinary
	// tap helper reaches it.
	tap(t, mgr, "Column")

	if boxRow(tree(t, mgr), "Column") == nil {
		t.Fatal("after switching the axis, the boxes should sit in a Column")
	}
	assertNoConcerns(t)
}
