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
	Key      string
	Props    map[string]any
	Style    *nodeStyle
	Children []*node
}

// nodeStyle decodes the handful of core.Style fields the tests assert on —
// the wire tree is a straight json.Marshal of core.Node, so the keys are the
// Go field names. Chapter 7's tests read these to watch prop order, merging,
// theme swaps and transition declarations land on actual nodes.
type nodeStyle struct {
	Background   string
	TextColor    string
	FontSize     float64
	FontWeight   int
	BorderRadius float64
	Transition   string

	// The three layout fields chapter 4's endless-feed lesson asserts on.
	// core.Horizontal and core.StickyHeader are StyleProps rather than props,
	// which is the whole point of how they are built — so the only place a
	// test can see them is here, in the style the renderers receive.
	FlexDirection string
	Overflow      string
	Position      string

	// Chapter 4's calendar lesson asserts on these three. A day cell is a
	// tappable Box rather than a Button — it stacks a numeral over a mark —
	// so its spoken name is the only thing that identifies it, its ring is a
	// border rather than a fill, and whether it is inert is a style flag
	// rather than a missing handler.
	AccessibilityLabel string
	BorderWidth        float64
	Disabled           bool

	// Chapter 4's compass lesson asserts on this one. core.Rotate is a style
	// field and the rose is an unlabelled Box, so the angle is the only thing
	// in the tree that says which way the widget is pointing.
	Rotate float64

	// Chapter 4's small-controls lesson asserts on this one. A hidden
	// comps.Spinner stays in the tree (it owns hook slots) and is hidden by
	// Display, so this is the only field that says whether it is showing.
	Display core.DisplayMode

	// Chapter 4's tab-strip arrangement asserts on these two. They are the
	// whole subject of that demo — the same widget announcing itself
	// differently — and neither is visible anywhere else in the tree: a tab
	// and a filter chip draw identically, so the style is the only place the
	// difference exists.
	AccessibilityRole     string
	AccessibilitySelected string

	// The disclosure state, for 4.4's accordion lesson. It is the one thing on
	// that screen with no visual counterpart at all: the chevron flips and the
	// section opens whether or not this is stated, which is how the widget
	// shipped without it for as long as it did.
	AccessibilityExpanded string

	// The heading tier, for the outline assertion in this file. It is the one
	// property of a lesson screen that exists nowhere else in the tree: three
	// headings on one screen draw at three different sizes, but "which is a
	// section of which" is carried by this number alone.
	AccessibilityHeadingLevel int

	// The collection depth, for chapter 4's flattened-outline demo. Same
	// argument one role over: the rows are indented in pixels and siblings in
	// the tree, so this number is the only place their nesting exists.
	AccessibilityNestingLevel int

	// The field frame, for chapter 5's picker lesson. The claim there is that
	// a picker and a text field wear the same edge, and the edge is a style
	// field on both — there is nothing in the props to compare.
	BorderColor string

	// The two IDREF props, for 4.5's hand-assembled tab strip. They are the
	// only place the relationship between a tab and the region it shows
	// exists at all: the strip and the pages are siblings in the tree, drawn
	// identically whether or not either end has been stated.
	AccessibilityID       string
	AccessibilityControls string

	// The inset set, for 7.2's box-model half. The lesson there is entirely
	// about which of the four sides a layer leaves behind, so the sides have
	// to be readable individually — a rendered box does not say in its props
	// whether its top padding survived. Decoded as the wire shape, six fields
	// and not four, because the shorthand pair is exactly what the per-side
	// props dissolve and the test wants to see it gone.
	Padding struct {
		Top, Right, Bottom, Left int
		Horizontal, Vertical     int
	}
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

// findNodes is findNode's plural: every node under n matching the predicate,
// in tree order.
//
// It exists for the assertions that are about a *set* of nodes rather than
// one — chapter 4's tab strip, where what is being checked is that every tab
// answers and exactly one says yes. Written as a separate walk rather than by
// giving findNode a limit, so the singular stays the cheap early-exit it is
// for the dozen callers that want the first match.
func findNodes(n *node, pred func(*node) bool) []*node {
	var out []*node
	var walk func(*node)
	walk = func(n *node) {
		if n == nil {
			return
		}
		if pred(n) {
			out = append(out, n)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return out
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
//
// It reads the tutorial's two ways of putting words on a screen: a Text
// node's content, and one row of a code block. Code blocks became
// core.TextGrids when they gained syntax highlighting, so the text of a line
// is no longer one string in one node — it is the row's runs concatenated,
// and where the run boundaries fall depends on how Go tokenizes the line.
// Several lessons print a live literal into their code block and assert on it
// (chapter 4's button demo names the variant it built), and every one of
// those assertions would otherwise have gone quietly vacuous: the substring
// would simply stop being found, which is a passing negative check and a
// failing positive one.
func hasTextContaining(n *node, sub string) bool {
	return findNode(n, func(n *node) bool {
		if s, ok := n.Props["content"].(string); ok && n.Type == "Text" {
			return strings.Contains(s, sub)
		}
		return n.Type == "GridRow" && strings.Contains(gridRowText(n), sub)
	}) != nil
}

// gridRowText reassembles one core.TextGrid row from its runs.
//
// The runs arrive as they crossed the wire — core.GridRun's json tags, so the
// glyphs are under "t" — because these tests read the marshalled tree a native
// shell would receive rather than the Go nodes. A row whose props are some
// other shape yields "", which is the same answer the renderers give it.
//
// Only within one row: a match spanning a line break is not a match, exactly
// as it would not be for a Text node's content.
func gridRowText(n *node) string {
	runs, ok := n.Props["runs"].([]any)
	if !ok {
		return ""
	}
	var b strings.Builder
	for _, raw := range runs {
		run, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if t, ok := run["t"].(string); ok {
			b.WriteString(t)
		}
	}
	return b.String()
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
//
// # It opens the chapter first when it has to
//
// The chapter cards are collapsed by default, so a lesson's row is not on the
// screen until its card is. Pressing the band and then the row is what a reader
// does, and doing it here rather than in forty tests is what keeps every lesson
// test about its own lesson.
//
// The band is pressed only when the row is already absent, because pressing an
// open chapter's band shuts it — so an unconditional press would hide the row
// this function exists to find, on every test whose chapter happened to be open
// already (chapter 1 always is, and so is whichever one a previous step
// visited).
func openLesson(t *testing.T, mgr *render.Manager, title string) {
	t.Helper()
	row := func() *node {
		return findNode(tree(t, mgr), func(n *node) bool {
			_, clickable := n.Props["onClick"].(string)
			return clickable && n.Type != "Button" && hasText(n, title)
		})
	}
	n := row()
	if n == nil {
		expandChapterFor(t, mgr, title)
		n = row()
	}
	if n == nil {
		t.Fatalf("no tappable contents row titled %q", title)
	}
	mgr.DispatchCallback(n.Props["onClick"].(string))
}

// expandChapterFor presses the disclosure on the chapter card holding the
// lesson with this title.
//
// The chapter is looked up in flatLessons rather than found by walking the
// tree: the index is the curriculum's own answer to "which chapter is this
// lesson in", and deriving it from the screen would make the helper agree with
// whatever the screen currently draws — including a screen that had put the
// lesson in the wrong card.
//
// The band itself is matched on the exact words it draws (chapterBandText), and
// those words are only ever inside the disclosure's button, so there is nothing
// else in the tree for this to land on.
func expandChapterFor(t *testing.T, mgr *render.Manager, title string) {
	t.Helper()
	chapter := -1
	for _, e := range flatLessons {
		if e.Title == title {
			chapter = e.ChapterNum - 1
			break
		}
	}
	if chapter < 0 {
		t.Fatalf("no lesson titled %q in the curriculum", title)
	}
	expandChapter(t, mgr, chapter)
}

// toggleCheckbox flips the idx-th checkbox in tree order. The demos'
// checkboxes carry no distinguishing props, so position is the only address —
// the same walk signup uses for its second password field.
func toggleCheckbox(t *testing.T, mgr *render.Manager, idx int, on bool) {
	t.Helper()
	toggleBool(t, mgr, "Checkbox", idx, on)
}

// toggleBool is the same walk over either boolean control. It takes the node
// type because that is the only thing separating the two — both carry
// `checked` and both report through `onToggle` (see core.Switch) — so a test
// that asked for "the first bool control" would flip whichever one the demo
// happened to write first.
func toggleBool(t *testing.T, mgr *render.Manager, nodeType string, idx int, on bool) {
	t.Helper()
	var found []*node
	var walk func(n *node)
	walk = func(n *node) {
		if n == nil {
			return
		}
		if n.Type == nodeType {
			found = append(found, n)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(tree(t, mgr))
	if idx >= len(found) {
		t.Fatalf("wanted %s %d, tree has %d", nodeType, idx, len(found))
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

// The contents screen lists every chapter, and every lesson of the chapters
// that are open.
//
// It used to list all 49 lessons at once and this test used to say so. The
// cards collapse now, so the claim splits in two: the *curriculum* is still
// wholly reachable — that is the part a reader would notice going missing —
// and what is on the screen at any moment is the chapters plus one chapter's
// rows. Both halves are checked, because a screen that had stopped drawing the
// rows of an OPEN chapter would pass a check that only counted headers.
func TestHomeListsEveryChapterAndTheOpenChapterSLessons(t *testing.T) {
	mgr := newApp(t)
	root := tree(t, mgr)

	if !hasText(root, "GrMob Interactive Tutorial") {
		t.Fatal("home is missing its title")
	}

	// Every chapter has a band, open or shut.
	for ci := range Chapters {
		if !hasText(root, chapterBandText(ci)) {
			t.Errorf("home is missing chapter %d's disclosure", ci+1)
		}
	}

	// Chapter 1 is the one seeded open: its rows are on the screen and no
	// other chapter's are. The second half is what makes this a test of the
	// collapse rather than of the curriculum — without it, a card that had
	// stopped collapsing would pass.
	for _, e := range flatLessons {
		open := e.ChapterNum == 1
		if got := hasText(root, e.Title); got != open {
			t.Errorf("lesson %s (%s) on screen = %v, want %v",
				e.ID, e.Title, got, open)
		}
		if got := hasText(root, e.ID); got != open {
			t.Errorf("the %s ordinal on screen = %v, want %v", e.ID, got, open)
		}
	}

	// And every lesson is reachable by opening its chapter, which is the claim
	// the old whole-screen check was really making.
	for ci := range Chapters {
		if ci > 0 {
			expandChapter(t, mgr, ci)
		}
		now := tree(t, mgr)
		for _, e := range flatLessons {
			if e.ChapterNum-1 != ci {
				continue
			}
			if !hasText(now, e.Title) {
				t.Errorf("lesson %s (%s) is unreachable with chapter %d open",
					e.ID, e.Title, ci+1)
			}
			if !hasText(now, e.ID) {
				t.Errorf("the %s ordinal is unreachable with chapter %d open",
					e.ID, ci+1)
			}
		}
	}

	want := fmt.Sprintf("0 of %d lessons opened", len(flatLessons))
	if !hasText(tree(t, mgr), want) {
		t.Fatalf("home is missing the progress caption %q", want)
	}
	assertNoConcerns(t)
}

// A chapter card's disclosure, pressed once. Toggles: the caller has to know
// which way it will go, which is why openLesson asks whether the row is
// already showing before it comes here.
func expandChapter(t *testing.T, mgr *render.Manager, chapter int) {
	t.Helper()
	words := chapterBandText(chapter)
	n := findNode(tree(t, mgr), func(n *node) bool {
		_, clickable := n.Props["onClick"].(string)
		return clickable && hasText(n, words)
	})
	if n == nil {
		t.Fatalf("no chapter disclosure showing %q", words)
	}
	mgr.DispatchCallback(n.Props["onClick"].(string))
}

// The contents screen is inset once, by its List.
//
// This was a real 16-point shift, found by measuring ink columns in simulator
// screenshots rather than by eye, and it survived a code review because
// neither inset is written in any source file: comps.Screen's column and
// core.List are both built on the theme's Components.Column, so moving this
// page from a scrolled Column to a List silently added a second copy of the
// same padding. comps.Screen now drops its own when its whole content is
// a scrolling page; what that rule is worth is exactly this screen, so the
// assertion lives here as well as in the widget's own tests.
//
//	SafeArea
//	  └─ Column   padding 0        ← a safe-area frame, nothing else
//	       └─ List padding 12/16   ← the page, inset once
func TestHomeIsInsetOnce(t *testing.T) {
	mgr := newApp(t)
	root := tree(t, mgr)

	list := findNode(root, func(n *node) bool { return n.Type == "List" })
	if list == nil || list.Style == nil {
		t.Fatal("the contents screen is not a List any more; the inset rule below is about that List")
	}
	base := core.DefaultTheme.Components.Column.Padding
	if list.Style.Padding.Left != base.Left || list.Style.Padding.Top != base.Top {
		t.Errorf("the page lost its own inset: %+v, want the theme's %+v", list.Style.Padding, base)
	}

	// The scaffold's column is the List's parent, so it is the one node
	// between the safe area and the page that could inset it a second time.
	col := findNode(root, func(n *node) bool {
		if n.Type != "Column" {
			return false
		}
		return len(n.Children) == 1 && n.Children[0] == list
	})
	if col == nil {
		t.Fatalf("no column holds the List; the tree shape changed")
	}
	if col.Style != nil && (col.Style.Padding.Left != 0 || col.Style.Padding.Top != 0) {
		t.Errorf("the contents screen is inset twice: the scaffold's column adds %+v on top of the List's",
			col.Style.Padding)
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

// --- What the contents screen costs to send ------------------------------

// The initial tree of the contents screen, in bytes of JSON.
//
// This is not a performance test and it does not assert a budget; it prints a
// number that is otherwise invisible and fails only if the screen's cost
// changes by an order of magnitude. What it records is the half of an emulator
// measurement that no device was needed to explain — and then the fix that
// halved it, which no device was needed to find either.
//
// # The measurement that made this number interesting
//
// Four arms on one emulator, means of five cold launches, from
// android/device/launch.sh, which carries the table:
//
//	home = title + progress card only, same binary     2529 ms
//	home = the whole contents as a core.List           5045 ms
//	home = the whole contents as a scrolled Column     6062 ms
//
// The lazy container wins 1017ms — Compose composes the rows on screen and not
// the other 45 — which is the win iOS saw and the reason this screen is a
// core.List. But the List arm was still 2516ms above the same binary with the
// cards taken off the screen, and laziness cannot touch any of it: the whole
// tree crosses the bridge whether or not Compose composes it.
//
// # Where that 2516ms actually went
//
// Attributed by GrMobRuntime's own stage clocks (see Startup.kt), five cold
// launches, the same emulator:
//
//	bridge  Go's render + marshal + the gomobile crossing     17 ms
//	parse   org.json turning 423,472 bytes into JSONObjects  1666 ms
//	build   GrMobNode.parse walking those into the tree       427 ms
//
// So it was never the bridge and never Go. It was the parse, and the parse was
// large because the payload was: 92.4% of those bytes were core.Style, written
// out field by field for 336 nodes that had 1,168 non-zero style fields
// between them — three and a half each, out of fifty-eight.
//
// core.Style's fields are `,omitzero` now, which is the whole fix. The size
// below is the after; the before was 423,472.
//
//	                    bytes      parse    build    cold launch
//	every field       423,472    1666 ms   427 ms      4850 ms
//	zero omitted       53,408     249 ms   185 ms      3530 ms
//
// # The third row that is not in the table
//
// The tags had stopped one level too high: a *present* Padding still wrote all
// six of core.EdgeInsets' untagged ints, and the axis pair was zero in all 77
// insets on this screen. Tagging them took the payload to 51,242 — 2,166
// bytes, 4.0% — and the same five-launch A/B on the same emulator an hour
// after the one above says that bought no time it can resolve:
//
//	                 bytes    parse            build            launch
//	untagged        53,408    201.3 ±16.8 ms   178.4 ±51.9 ms   3464 ±411 ms
//	tagged          51,242    197.4 ±31.1 ms   180.5 ± 8.7 ms   3233 ±186 ms
//
// Parse and build together move 379.8 ms → 377.9 ms, which is 1.9 ms against
// run-to-run spreads of 17 to 52. A 4% byte cut predicts about 8 ms if the
// parse is linear in length, and 8 ms is where this instrument stops seeing.
// The 231 ms in the launch column is not the tags either — the untagged arm's
// five runs fall 4000, 3764, 3394, 3106, 3058, which is a machine warming up.
//
// So it is in the tree for the bytes and for the rule, not for a reading. The
// bytes are certain and free; the milliseconds are below the floor. See the
// note on core.EdgeInsets, and TestEveryWireFieldOmitsZero for what now stops
// the same omission recurring a level further down.
//
// # Why the number is still worth printing
//
// Because the next screen can undo it. Nothing in the type system stops a
// widget from putting a kilobyte in Props, and the parse is still the largest
// single stage of an Android launch — it is simply now proportional to
// something small. The bound is a factor of ten in each direction because the
// number is a fact about 54 lessons of prose, which is edited: a new chapter
// should not fail a test, and a screen that suddenly sends four megabytes
// should.
func TestHomeTreeSize(t *testing.T) {
	mgr := newApp(t)
	size := len(mgr.RenderInitial())
	t.Logf("the contents screen is %d bytes of JSON on the wire", size)

	// The current size, not the launch-measured one. The A/B above was taken
	// at 53,408 and the EdgeInsets pass took it to here; the baseline this
	// guards drift from should be what the screen costs today, and the table
	// is where the measured arms keep their own numbers.
	//
	// # The collapse took another 67% off, and none of it was a wire change
	//
	// The chapter cards collapse now (one open, the rest shut — see the
	// tutorial's `expanded` field), so the rows of seven chapters are simply
	// not in the tree:
	//
	//	all eight cards open      53,156    what this screen used to send
	//	one card open             17,366    what it sends today
	//	every card shut           12,889    the floor: eight bands and their
	//	                                    summaries, plus the title and the
	//	                                    progress card
	//
	// Two things are worth taking from those numbers rather than from the
	// percentage. A row costs about 800 bytes, so the saving is a linear
	// function of how many rows are on screen and would be the same on any
	// screen that stops drawing rows nobody asked for. And the floor is
	// 12,889 — a quarter of the original — which is what a disclosure per
	// chapter costs: the heading-around-button-around-chevron shape is four
	// nodes with styles, and eight of them are not free. Collapsing is worth
	// it here because 49 rows is far more than eight bands; it would not be
	// worth it for a screen with three rows per section.
	//
	// This is also the comparison the windowing proposal wanted: windowing
	// core.List over the bridge was sized at 66-76% of the payload and needs a
	// bootstrap guess plus placeholder children. The collapse is 67% with no
	// protocol change at all.
	//
	// Re-profiled after it, windowing is worth 42-50% of a payload a quarter
	// the size — about 7-9KB where it was 34-39KB — because the bytes a window
	// would have declined to send are mostly the bytes the collapse already
	// stopped sending. It is declined on that number in
	// ai_docs/plans/non_goals.md, with the re-taken table in
	// TestWhatWindowingWouldSave. Not retired as an idea: a single open chapter
	// of forty lessons would still send forty rows, and that is the screen that
	// would argue for it.
	const recorded = 17366
	if size < recorded/10 || size > recorded*10 {
		t.Errorf("the contents screen is %d bytes of JSON, an order of magnitude "+
			"from the %d it costs today — a screen whose 53,408-byte arm is what "+
			"android/device/launch.sh attributed 1320ms of the Android launch to. "+
			"Not a budget failure — a prompt to re-measure and rewrite that "+
			"table.", size, recorded)
	}
	assertNoConcerns(t)
}

// TestWhatWindowingWouldSave prints how much of the contents screen lies in
// each child of its core.List, cumulatively, so the value of sending only the
// children near the viewport can be read off rather than re-derived.
//
// It asserts nothing about the numbers. Windowing is an open proposal, not a
// budget, and what this exists to stop is the proposal being re-priced from
// memory: the estimate it replaces ("worth ~450ms") was a whole-stage figure
// that nobody had split by what a window could actually remove.
//
// # The profile, re-taken after the collapse
//
//	 n  child                bytes    cumulative   sent    % of screen
//	 1  title                  459           459     681      3.9%
//	 2  progress card          696         1,155   1,377      7.9%
//	 3  chapter-0 card       5,872         7,027   7,249     41.7%
//	 4  chapter-1 band       1,436         8,463   8,685     50.0%
//	 5  chapter-2 band       1,427         9,890  10,112     58.2%
//	 …
//	10  chapter-7 band       1,429        17,144  17,366    100.0%
//
// "sent" is the whole payload for that window: everything outside the List
// (the scaffold, the List's own node) plus the children in it.
//
// # The three things this changes about the proposal
//
// **The granularity is the chapter, not the lesson.** The List has ten
// children — a title, a progress card, and eight chapter Cards — not the 49
// rows Home's comment counts, because the rows are nested inside the cards.
// So a window cannot be "the rows that fit"; the smallest one it can express
// is "the cards that fit".
//
// On a 1080x2400 emulator that fold falls inside the fourth child: title,
// progress, the whole of chapter-0's card (its five lesson rows), and the top
// of chapter-1's. The collapse did not move it — children 1 to 3 occupy
// exactly the pixels they did before, because chapter-0 is the open card and
// is unchanged — so the honest window is still n=4 and one card of overscan
// is n=5.
//
// # And what it changed is the prize
//
// Those two windows are 50.0% and 58.2% of the payload now, where the same
// two were 24.1% and 34.0% before. Windowing is worth **42-50% of the bytes**
// rather than 66-76% — and the percentage is the wrong number to read, because
// what an org.json parse costs is a function of bytes:
//
//	                          payload    window n=4-5 removes
//	before the collapse        51,242    33,800 - 38,900 bytes
//	today                      17,366     7,254 -  8,681 bytes
//
// A quarter of what it was. The seven shut chapters are bands of ~1,430 bytes
// where they were cards of ~5,300, so the bytes a window would have declined
// to send are the bytes the collapse already stopped sending — the two
// optimisations were mostly aimed at the same 45 lesson rows, and one of them
// got there first.
//
// Priced against the same emulator's attribution — ~377ms of parse-and-build
// for 51,242 bytes, so ~128ms for today's 17,366 if the parse is linear in
// length, which is the assumption the 8ms noise floor was derived under — a
// window saves about **54-64ms**. Still seven times the floor, so that
// instrument would still see it. But it is 1.7% of a 3,200ms launch where it
// used to be 7.7%, and the cost is unchanged: a visible-range event, a
// bootstrap guess, and placeholder children in four renderers.
//
// That is the re-decision, and it is why this test now asserts instead of only
// printing. The table above went stale in silence — it was taken at 51,242
// bytes and was still being quoted as the price after the screen became a
// third of that — which is the exact failure this repository writes numbers
// down to avoid. A printed number nothing checks is a number that has already
// moved.
//
// **The protocol change it was priced with already exists.** The item assumed
// a new bridge surface for the host to report a visible range. It does not
// need one: core.OnHostEvent / mobile.ReportHostEvent is a generic host→app
// channel that already carries a name and a JSON payload, already returns the
// following pass's patches on the event path, and is already serialized with
// render passes by the manager (see core/host_events.go). A visible range is
// one more event name.
//
// **What it actually costs is two things nobody had named.** First, the
// bootstrap: a cold launch has no visible range, because the host cannot lay
// out what it has not received, so the first render has to guess a window and
// be corrected. Second, and worse, the scroll extent — a LazyColumn sent
// three children believes there are three, so the scrollbar is wrong and the
// scroll stops short until more arrive. The honest fix is placeholder
// children (right key, estimated height, ~40 bytes) rather than absent ones,
// which keeps the extent right and still drops ~76% of the payload. That is a
// design the measurement supports and the original framing did not describe.
//
// # Why this is not built
//
// Because no screen in this repository prices it any more. See
// ai_docs/plans/non_goals.md, where the proposal now lives with this table.
//
// The short of it: 54-64ms on the one emulator anybody has measured, and that
// emulator is the machine most favourable to the argument — its org.json
// spends ~200ms on 51KB where iOS's parser spends 6ms on the same tree, so
// every byte is worth more there than anywhere else this runtime ships. A
// physical phone's ART would make it smaller, not larger, and the device pass
// that would say by how much is in ai_docs/plans/need_hardware.md.
//
// What is *not* in the way is the measurement. The same emulator's five-launch
// noise floor is about 8ms, established by an A/B of a 4% payload change that
// it could not see (TestHomeTreeSize above, and android/device/README.md), so
// 54-64ms is still seven times what that instrument can resolve. The reason to
// decline is not that the effect is unmeasurable. It is that the effect is now
// 1.7% of a launch and the cost is a protocol change plus placeholder children
// in four renderers.
//
// What would bring it back is a long list — a forty-lesson chapter open, or
// any app built on this framework with a genuinely long core.List. Windowing
// was always a framework answer rather than a tutorial one; what changed is
// that the tutorial stopped being the screen that argued for it.
func TestWhatWindowingWouldSave(t *testing.T) {
	mgr := newApp(t)
	tree := mgr.RenderInitial()

	// Decoded through json.RawMessage rather than the `node` type above,
	// because the interesting quantity is the exact byte count each child
	// occupies on the wire — and a decode-then-re-encode round trip through
	// map[string]any would renormalize key order and report a different
	// number than the one that crossed the bridge.
	var root wireNode
	if err := json.Unmarshal([]byte(tree), &root); err != nil {
		t.Fatalf("initial tree is not valid JSON: %v", err)
	}
	list, ok := findList(&root)
	if !ok {
		t.Fatal("the contents screen has no core.List — Home built one when this " +
			"profile was taken, and the whole proposal this prices is about that " +
			"container. Either the screen changed shape or the table below now " +
			"describes something that is not there.")
	}

	total := len(tree)
	var inChildren int
	for _, c := range list.Children {
		inChildren += len(c)
	}
	// Everything the host receives no matter how narrow the window: the
	// scaffold above the List and the List's own node, less its children.
	outside := total - inChildren

	t.Logf("the contents screen is %d bytes; %d of them (%.1f%%) are the "+
		"%d children of its core.List",
		total, inChildren, 100*float64(inChildren)/float64(total), len(list.Children))
	t.Logf("%-3s %-14s %-8s %9s %11s %9s %9s",
		"n", "key", "type", "bytes", "cumulative", "sent", "% sent")

	var cum int
	// What a window of four children would send, which is the one row of the
	// table the decision is actually taken on. Captured while the table is
	// printed rather than recomputed after it, so the number asserted below and
	// the number a reader sees are the same arithmetic.
	var windowFour int
	for i, raw := range list.Children {
		var c wireNode
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatalf("List child %d is not valid JSON: %v", i, err)
		}
		cum += len(raw)
		sent := outside + cum
		if i+1 == windowChildren {
			windowFour = sent
		}
		t.Logf("%-3d %-14s %-8s %9d %11d %9d %8.1f%%",
			i+1, c.Key, c.Type, len(raw), cum, sent,
			100*float64(sent)/float64(total))
	}

	// And the assertion the printing did without for too long.
	//
	// This test asserted nothing, so when the chapter collapse took the screen
	// from 51,242 bytes to 17,366 the table above did not change and did not
	// complain: it went on describing a screen that no longer existed, and the
	// proposal went on being priced at 66-76% when the true figure had become
	// 42-50% of a quarter as many bytes. A printed number nothing checks is a
	// number that has already moved.
	//
	// What is checked is the SHARE, not the byte count, and deliberately: the
	// share is what the decision turns on, and it is the quantity that stays
	// still while lessons are added. A band rather than an equality, because
	// this is not a budget — one more chapter moves it a point or two and that
	// is not a failure. The band is wide enough to ignore ordinary drift and
	// narrow enough to catch the kind of change that happened here, which moved
	// it twenty-six points.
	if windowFour == 0 {
		t.Fatalf("the List has %d children, fewer than the %d a window was "+
			"priced at. The table above describes a screen with eight chapter "+
			"cards and this one has something else.",
			len(list.Children), windowChildren)
	}
	share := 100 * float64(windowFour) / float64(total)
	if share < windowShareLow || share > windowShareHigh {
		t.Errorf("a window of %d children would send %.1f%% of the payload, "+
			"outside the %.0f-%.0f%% the table above was written at.\n\n"+
			"Not a budget failure. It is a prompt to re-take the profile and "+
			"rewrite that table and the decision under it, the way the chapter "+
			"collapse should have and did not: the proposal's price is this "+
			"number, and the last time it moved without anybody noticing it "+
			"went on being quoted at the old one for a session.",
			windowChildren, share, windowShareLow, windowShareHigh)
	}
	assertNoConcerns(t)
}

// The window the profile is read at, and the share of the payload it sent when
// the table above was written.
//
// Four is where a 1080x2400 emulator's fold falls: the title, the progress
// card, the whole of the open chapter's card, and the top of the next
// chapter's band. See the comment above TestWhatWindowingWouldSave.
const (
	windowChildren  = 4
	windowShareLow  = 40.0
	windowShareHigh = 60.0
)

// wireNode reads the three fields the byte profile needs off a node without
// decoding the rest of it. Children stay raw so their exact serialized length
// is available; see TestWhatWindowingWouldSave for why that matters.
type wireNode struct {
	Type     string
	Key      string
	Children []json.RawMessage
}

// findList returns the first core.List in document order, which on the
// contents screen is the page itself. Depth-first rather than by a fixed path
// so that wrapping the screen in another container does not silently turn
// this into a test of nothing.
func findList(n *wireNode) (*wireNode, bool) {
	if n.Type == "List" {
		return n, true
	}
	for _, raw := range n.Children {
		var c wireNode
		if err := json.Unmarshal(raw, &c); err != nil {
			continue
		}
		if found, ok := findList(&c); ok {
			return found, true
		}
	}
	return nil, false
}

// The chapter a reader came out of is open when they land back on the contents.
//
// This is the whole reason the expansion is session state rather than eight
// widgets' own: a comps.Accordion would have collapsed itself again on the
// way back, and nothing on the contents screen could have reached in to say
// otherwise. Both doors into a lesson are checked, because there are two and
// the rule lives in neither of them — see markVisited.
func TestTheChapterAReaderCameOutOfIsOpen(t *testing.T) {
	// Door one: the app's own controls, crossing a chapter boundary.
	//
	// The route is the last lesson of chapter 1 — which is open already, so
	// nothing about this arm depends on openLesson having expanded anything —
	// and then Next, which lands in chapter 2. Chapter 2 is shut at that
	// moment, and what is being checked is that coming back finds it open.
	mgr := newApp(t)
	last := flatLessons[0]
	for _, e := range flatLessons {
		if e.ChapterNum == 1 {
			last = e
		}
	}
	openLesson(t, mgr, last.Title)
	tap(t, mgr, "Next ›")
	tap(t, mgr, "‹ Contents")
	next := flatLessons[last.Index+1]
	if next.ChapterNum != 2 {
		t.Fatalf("the lesson after %s is %s, not the start of chapter 2 —"+
			" this test's route through the curriculum no longer crosses a"+
			" chapter boundary", last.ID, next.ID)
	}
	if !hasText(tree(t, mgr), next.Title) {
		t.Errorf("chapter 2 is shut after coming back from %s", next.ID)
	}

	// Door two: a deep link, which never passes through the row or through
	// `open`. This is the path that was actually broken when the rule lived on
	// one door — grmob://lesson/2.3 opens a lesson the reader never pressed a
	// row for, and ‹ Contents afterwards landed them on a shut card.
	mgr2 := newApp(t)
	tree(t, mgr2) // the first render is what subscribes the app
	route("2.3")
	tap(t, mgr2, "‹ Contents")
	linked, _ := resolveRoute("2.3")
	if !hasText(tree(t, mgr2), linked.Title) {
		t.Errorf("chapter 2 is shut after a deep link to 2.3 and back")
	}
}

// A chapter's band says how far the reader has gotten in it, which is the only
// progress a shut card can show.
func TestAChapterBandCountsItsLessons(t *testing.T) {
	mgr := newApp(t)
	n := len(Chapters[0].Lessons)

	root := tree(t, mgr)
	if want := fmt.Sprintf("%d lessons", n); !hasText(root, want) {
		t.Errorf("chapter 1's band is missing %q", want)
	}

	// Open one, come back: the count becomes a ratio. "1 of 6" rather than
	// "6 lessons" is the difference between a card that says how big it is and
	// one that says where the reader is in it.
	openLesson(t, mgr, Chapters[0].Lessons[0].Title)
	tap(t, mgr, "‹ Contents")
	if want := fmt.Sprintf("1 of %d", n); !hasText(tree(t, mgr), want) {
		t.Errorf("chapter 1's band is missing %q after opening one lesson", want)
	}
}
