package tutorial

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// The two-pane layout (split.go). The mode is delivered the way wasm/main.go
// delivers every host event — core.ReceiveHostEvent with a decoded payload —
// so these tests stand where the page stands.

// setLayout sends what the page sends at boot, on the toggle and on a resize
// across its breakpoint.
func setLayout(mode string) {
	core.ReceiveHostEvent(layoutEvent, map[string]any{"mode": mode})
}

// newSplitApp is newApp in the split layout. It renders once before asking,
// because the app takes its subscription during its first render — which is
// also the page's order: it sends the mode after the mount.
func newSplitApp(t *testing.T) *render.Manager {
	t.Helper()
	mgr := newApp(t)
	tree(t, mgr)
	setLayout("split")
	return mgr
}

// byID finds the node the page's stylesheet would find by this id.
func byID(root *node, id string) *node {
	return findNode(root, func(n *node) bool {
		return n.Style != nil && n.Style.AccessibilityID == id
	})
}

// panes returns the split layout's two halves, failing when the tree is not
// in it.
func panes(t *testing.T, root *node) (guide, phone *node) {
	t.Helper()
	guide, phone = byID(root, guideID), byID(root, phoneScreenID)
	if guide == nil || phone == nil {
		t.Fatalf("expected the split layout, got guide=%v phone=%v", guide != nil, phone != nil)
	}
	return guide, phone
}

// isDemoPanel reports a demo panel's root, by the key demoPanel gives it.
func isDemoPanel(n *node) bool { return strings.HasPrefix(n.Key, demoKeyPrefix) }

// Without a layout event nothing changes: the phone layout is the default on
// every host, and it is the Navigator's tree untouched.
func TestPhoneLayoutIsTheDefault(t *testing.T) {
	mgr := newApp(t)
	if byID(tree(t, mgr), splitID) != nil {
		t.Fatal("the contents must not be split before the page asks")
	}
	openLesson(t, mgr, "Hello, GrMob")
	if byID(tree(t, mgr), splitID) != nil {
		t.Fatal("a lesson must not be split before the page asks")
	}
	assertNoConcerns(t)
}

// Every lesson splits cleanly: its guide keeps the lesson's own words and
// controls and loses every demo panel, each to the phone, with one pointer
// left behind per panel; and no Modal is left over the reading pane.
func TestEveryLessonSplitsIntoGuideAndPhone(t *testing.T) {
	mgr := newSplitApp(t)

	openLesson(t, mgr, flatLessons[0].Title)
	for i, e := range flatLessons {
		guide, phone := panes(t, tree(t, mgr))

		if !hasText(guide, e.ID+"  "+e.Title) {
			t.Fatalf("%s: the guide should carry the lesson's heading", e.ID)
		}
		if findNodes(guide, isDemoPanel) != nil {
			t.Fatalf("%s: a demo panel was left in the guide", e.ID)
		}
		if findNode(guide, func(n *node) bool { return n.Type == "Modal" }) != nil {
			t.Fatalf("%s: a Modal was left in the guide", e.ID)
		}
		demos := findNodes(phone, isDemoPanel)
		pointers := findNodes(guide, func(n *node) bool {
			return n.Type == "Text" && n.Props["content"] == "on the phone →"
		})
		if len(demos) == 0 {
			t.Fatalf("%s: the phone shows no demo", e.ID)
		}
		if len(pointers) != len(demos) {
			t.Fatalf("%s: %d demos on the phone but %d pointers in the guide", e.ID, len(demos), len(pointers))
		}

		if i < len(flatLessons)-1 {
			tap(t, mgr, "Next ›") // the guide's own nav, which must still work
		}
	}
	tap(t, mgr, "Finish ✓")
	assertNoConcerns(t)
}

// The contents open in the guide with the phone idle beside them, and a row
// tapped there opens the lesson split.
func TestContentsSplitWithASplash(t *testing.T) {
	mgr := newSplitApp(t)

	guide, phone := panes(t, tree(t, mgr))
	if !hasText(guide, "GrMob Interactive Tutorial") {
		t.Fatal("the contents should be in the guide")
	}
	if !hasTextContaining(phone, "Pick a lesson") {
		t.Fatal("the phone should show the splash on the contents")
	}

	openLesson(t, mgr, "Hello, GrMob")
	_, phone = panes(t, tree(t, mgr))
	if !hasText(phone, "1.1  Hello, GrMob") {
		t.Fatal("the phone's header should name the open lesson")
	}
	assertNoConcerns(t)
}

// The demos on the phone are the live demos: a control there drives the
// lesson's state, and the result lands on the phone.
func TestSplitDemoIsLive(t *testing.T) {
	mgr := newSplitApp(t)
	openLesson(t, mgr, "Hello, GrMob")

	_, phone := panes(t, tree(t, mgr))
	if !hasText(phone, "1.2k") {
		t.Fatal("the stats block should render on the phone while its toggle is on")
	}
	toggleCheckbox(t, mgr, 1, false) // 0: Header, 1: Stats, 2: Bio
	_, phone = panes(t, tree(t, mgr))
	if hasText(phone, "1.2k") {
		t.Fatal("unchecking Stats() on the phone should remove the stats subtree")
	}
	assertNoConcerns(t)
}

// Flipping the layout mid-lesson rearranges nodes and nothing else, so the
// demo's state survives both directions — the whole reason the split works on
// the rendered tree rather than by rendering the body differently.
func TestLayoutSwitchKeepsDemoState(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Hello, GrMob")

	toggleCheckbox(t, mgr, 2, true) // Bio on, in the phone layout
	setLayout("split")
	_, phone := panes(t, tree(t, mgr))
	if !hasTextContaining(phone, "without leaving Go") {
		t.Fatal("the bio turned on before the switch should still be on after it")
	}

	toggleCheckbox(t, mgr, 2, false) // and off, in the split layout
	setLayout("phone")
	root := tree(t, mgr)
	if byID(root, splitID) != nil {
		t.Fatal(`"phone" should turn the split off`)
	}
	if hasTextContaining(root, "without leaving Go") {
		t.Fatal("the bio turned off in the split should still be off in the phone layout")
	}
	assertNoConcerns(t)
}

// A screen a demo pushes (chapter 6) is not a lesson: it runs on the phone
// whole, and the guide says which lesson the reader is in until it pops.
func TestPushedDemoScreenRunsOnThePhone(t *testing.T) {
	mgr := newSplitApp(t)
	openLesson(t, mgr, "Screens are a stack")

	tap(t, mgr, "Push the detail screen ›")
	guide, phone := panes(t, tree(t, mgr))
	if !hasText(phone, "The detail screen") {
		t.Fatal("the pushed screen should be on the phone")
	}
	if hasText(guide, "The detail screen") {
		t.Fatal("the pushed screen must not be in the guide")
	}
	if !hasTextContaining(guide, "Screens are a stack") {
		t.Fatal("the guide should name the lesson underneath")
	}

	tap(t, mgr, "‹ Pop back to the lesson")
	guide, _ = panes(t, tree(t, mgr))
	if findNode(guide, func(n *node) bool { return n.Type == "Button" && n.Props["label"] == "Next ›" }) == nil {
		t.Fatal("popping back should bring the lesson's guide back")
	}
	assertNoConcerns(t)
}

// liftDemos never writes the tree it is given: a rendered node is frozen, and
// may be a core.Cached node shared with other passes.
func TestLiftDemosCopiesOnWrite(t *testing.T) {
	panel := &core.Node{Type: "Column", Key: demoKeyPrefix + "hint"}
	modal := &core.Node{Type: "Modal"}
	untouched := &core.Node{Type: "Text"}
	body := &core.Node{Type: "Column", Children: []*core.Node{untouched, panel, modal}}
	root := &core.Node{Type: "Screen", Children: []*core.Node{body}}

	var hints []string
	guide, demos, overlays := liftDemos(root, func(hint string) *core.Node {
		hints = append(hints, hint)
		return &core.Node{Type: "Pointer"}
	})

	if len(root.Children) != 1 || len(body.Children) != 3 || body.Children[1] != panel {
		t.Fatal("liftDemos wrote into its input")
	}
	if len(demos) != 1 || demos[0] != panel || len(overlays) != 1 || overlays[0] != modal {
		t.Fatalf("expected the panel and the modal lifted, got %d demos, %d overlays", len(demos), len(overlays))
	}
	if len(hints) != 1 || hints[0] != "hint" {
		t.Fatalf("the pointer should be asked for the panel's hint, got %v", hints)
	}
	got := guide.Children[0].Children
	if len(got) != 2 || got[0] != untouched || got[1].Type != "Pointer" {
		t.Fatal("the guide should keep the untouched child and hold a pointer where the panel was")
	}

	// A tree with nothing to lift comes back as the same pointer.
	if same, _, _ := liftDemos(untouched, nil); same != untouched {
		t.Fatal("a tree with no demo in it should be returned as is")
	}
}
