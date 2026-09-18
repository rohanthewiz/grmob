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

// newSplitApp is newApp in the split layout, switched after the first render:
// the path a live tree takes on the toggle and on a resize. The page's boot
// sends the mode before the first render instead, which
// TestAModeSentBeforeTheFirstRenderIsTheFirstFrame covers.
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

// The page sends the mode before RenderInitial, so a wide window's first frame
// is the split rather than one frame of the phone layout and then a patch.
// And a mode is the boot value of the app it was sent to, never of the next
// one: once that app closes, the following app boots in the phone layout.
func TestAModeSentBeforeTheFirstRenderIsTheFirstFrame(t *testing.T) {
	setLayout("split")
	mgr := newApp(t)
	if byID(tree(t, mgr), splitID) == nil {
		t.Fatal("a mode sent before the first render should be the first frame's layout")
	}
	mgr.Close()

	next := newApp(t)
	if byID(tree(t, next), splitID) != nil {
		t.Fatal("the closed app's mode leaked into the next app's boot")
	}
	// A live tree's mode is its own state: sent now, it switches this tree
	// and is not recorded as anyone's boot value.
	setLayout("split")
	if byID(tree(t, next), splitID) == nil {
		t.Fatal("a mode sent to a live tree should switch it")
	}
	if bootSplit() {
		t.Fatal("a mode sent while a tree listens must not become a boot value")
	}
}

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
	// The covered lesson itself, not a note about it: its Next button is
	// there, switched off, and nothing in the guide carries a callback ID,
	// which would name one of the pushed screen's handlers by now.
	next := findNode(guide, func(n *node) bool { return n.Type == "Button" && n.Props["label"] == "Next ›" })
	if next == nil || next.Style == nil || !next.Style.Disabled {
		t.Fatal("the covered lesson's controls should be in the guide, disabled")
	}
	if cb := findNode(guide, func(n *node) bool {
		for k, v := range n.Props {
			if id, ok := v.(string); ok && id != "" && strings.HasPrefix(k, "on") {
				return true
			}
		}
		return false
	}); cb != nil {
		t.Fatalf("a %s in the covered guide still carries a callback", cb.Type)
	}
	// The lesson keeps slot 0 of the pane and its key, and the bar comes
	// after it: the reconciler matches by position, so this is what keeps the
	// guide's scroll position across the push and the pop.
	pane := byID(tree(t, mgr), guideID)
	if len(pane.Children) != 2 || !strings.HasSuffix(pane.Children[0].Key, "/"+lessonRootKey) ||
		byID(pane.Children[1], guideNoteID) == nil {
		t.Fatalf("expected [lesson, note] in the guide pane, got %d children", len(pane.Children))
	}

	tap(t, mgr, "‹ Pop back to the lesson")
	guide, _ = panes(t, tree(t, mgr))
	live := findNode(guide, func(n *node) bool { return n.Type == "Button" && n.Props["label"] == "Next ›" })
	if live == nil || live.Props["onClick"] == nil {
		t.Fatal("popping back should bring the live lesson's guide back")
	}
	assertNoConcerns(t)
}

// inert copies: the tree it is given is frozen, like liftDemos'.
func TestInertCopiesAndDisables(t *testing.T) {
	button := &core.Node{Type: "Button", Props: map[string]any{"label": "Go", "onClick": "cb_3", "focusEpoch": 2}}
	text := &core.Node{Type: "Text", Props: map[string]any{"text": "hi"}}
	root := &core.Node{Type: "Column", Key: "k", Children: []*core.Node{text, button}}

	out := inert(root)
	if button.Props["onClick"] != "cb_3" || button.Style != nil || button.Props["focusEpoch"] != 2 {
		t.Fatal("inert wrote into its input")
	}
	if out.Key != "k" || len(out.Children) != 2 || out.Children[0].Props["text"] != "hi" {
		t.Fatal("inert must keep keys, order and every non-callback prop")
	}
	b := out.Children[1]
	if _, ok := b.Props["onClick"]; ok || b.Style == nil || !b.Style.Disabled {
		t.Fatalf("the copy's button should have no callback and be disabled: %+v", b.Props)
	}
	if _, ok := b.Props["focusEpoch"]; ok {
		t.Fatal("a focus command would fire again on a host that rebuilds the copy")
	}
	if out.Children[0].Style != nil {
		t.Fatal("a node with no callback should not be marked disabled")
	}

	// A Paragraph's link runs carry their callbacks a level down.
	runs := []map[string]any{{"t": "see "}, {"t": "terms", "cb": "cb_9", "fg": "#00f"}}
	para := inert(&core.Node{Type: "Paragraph", Props: map[string]any{"runs": runs}})
	got := para.Props["runs"].([]map[string]any)
	if _, ok := got[1]["cb"]; ok || got[1]["fg"] != "#00f" || runs[1]["cb"] != "cb_9" {
		t.Fatalf("the copy's link run should lose only its callback, and the input keep it: %v / %v", got, runs)
	}
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

// A guide pointer brings its own panel into view on the phone: tapping the
// second pointer of a two-demo lesson stamps the second panel, and only it,
// with a scroll command (core.ScrollIntoView). Which element the host then
// scrolls is the host's; the stamp is what Go promises.
func TestAPointerScrollsThePhoneToItsPanel(t *testing.T) {
	mgr := newSplitApp(t)
	openLesson(t, mgr, "Controlled inputs")

	guide, _ := panes(t, tree(t, mgr))
	pointers := findNodes(guide, func(n *node) bool {
		return n.Style != nil && strings.HasPrefix(n.Style.AccessibilityLabel, "Show on the phone: ")
	})
	if len(pointers) != 2 {
		t.Fatalf("lesson 2.3 should leave two pointers in the guide, found %d", len(pointers))
	}
	if pointers[1].Style.AccessibilityRole != string(core.RoleButton) {
		t.Fatal("a pointer is a button to accessibility")
	}
	id, _ := pointers[1].Props["onClick"].(string)
	if id == "" {
		t.Fatal("a pointer should be tappable")
	}
	mgr.DispatchCallback(id)

	_, phone := panes(t, tree(t, mgr))
	panels := findNodes(phone, isDemoPanel)
	if len(panels) != 2 {
		t.Fatalf("expected both panels on the phone, found %d", len(panels))
	}
	if _, ok := panels[0].Props["scrollEpoch"]; ok {
		t.Fatal("the first panel was stamped for the second pointer")
	}
	// A number off the wire: the test tree is decoded JSON.
	if e, _ := panels[1].Props["scrollEpoch"].(float64); e < 1 {
		t.Fatalf("the second panel should carry the scroll command, got %#v", panels[1].Props)
	}
	assertNoConcerns(t)
}
