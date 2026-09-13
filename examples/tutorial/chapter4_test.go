package tutorial

import (
	"slices"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// Chapter 4 demo-liveness tests. The widget lessons are all synchronous —
// every change rides a dispatched callback — so no waiting primitive is
// needed; what is new is structural addressing for clickable *rows*: ListRow
// and the Accordion header register OnClick on a container, not a Button, so
// the tap helper's label lookup can't reach them.

// tapRow dispatches the click of the outermost clickable non-Button node
// whose subtree contains sub — a ListRow, an accordion header. It is
// openLesson's finder generalized to substring matching, because row titles
// here sit next to Sprintf-assembled captions the exact matcher would miss.
func tapRow(t *testing.T, mgr *render.Manager, sub string) {
	t.Helper()
	n := findNode(tree(t, mgr), func(n *node) bool {
		_, clickable := n.Props["onClick"].(string)
		return clickable && n.Type != "Button" && hasTextContaining(n, sub)
	})
	if n == nil {
		t.Fatalf("no tappable row containing %q", sub)
	}
	mgr.DispatchCallback(n.Props["onClick"].(string))
}

// --- 4.1 Buttons ----------------------------------------------------------

func TestButtonDemoAxesDisableDispatch(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Buttons: two axes")

	// The preview button is live: taps land and are counted.
	tap(t, mgr, "Save changes")
	tap(t, mgr, "Save changes")
	cur := tree(t, mgr)
	if !hasTextContaining(cur, "taps landed: 2") {
		t.Fatal("both taps on the preview button should land")
	}
	// With both axes at zero the printed literal must not name them — the
	// zero-value-contributes-nothing claim, checked against the demo's own
	// code block (the static intro block names Error/Outlined, never these).
	if hasTextContaining(cur, "comps.VariantWarning") ||
		hasTextContaining(cur, "comps.EmphasisGhost") {
		t.Fatal("the printed literal should omit zero-value axes")
	}

	// Moving the knobs rewrites the printed literal.
	tap(t, mgr, "Warning")
	tap(t, mgr, "Ghost")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "comps.VariantWarning") {
		t.Fatal("selecting the Warning variant should print Variant in the literal")
	}
	if !hasTextContaining(cur, "comps.EmphasisGhost") {
		t.Fatal("selecting Ghost emphasis should print Emphasis in the literal")
	}

	// Disabling keeps the handler registered but swaps it for a no-op, so a
	// dispatched tap (the racing-tap window) must not count.
	toggleCheckbox(t, mgr, 0, true) // 0: Disabled, 1: Full width
	if !hasTextContaining(tree(t, mgr), "Disabled: true") {
		t.Fatal("the printed literal should show Disabled: true")
	}
	tap(t, mgr, "Save changes")
	if !hasTextContaining(tree(t, mgr), "taps landed: 2") {
		t.Fatal("a tap dispatched to a disabled button must not land")
	}
	assertNoConcerns(t)
}

// --- 4.2 Badges, chips & segments -----------------------------------------

func TestPillsDemoBadgeAndChipSelection(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Badges, chips & segments")

	// The badge renders the status the segmented control selects. Badge text
	// is a Text node; the segments are Buttons — hasText only sees the badge.
	cur := tree(t, mgr)
	if !hasText(cur, "Draft") {
		t.Fatal("the badge should open on Draft")
	}
	tap(t, mgr, "Live")
	cur = tree(t, mgr)
	if !hasText(cur, "Live") {
		t.Fatal("selecting Live should re-render the badge with it")
	}
	if hasText(cur, "Draft") {
		t.Fatal("the old badge text should be gone after the selection moves")
	}

	// Chips are a controlled multi-select over one map: taps toggle keys, and
	// the summary caption derives from the same map in topic order.
	if !hasTextContaining(cur, "nothing picked") {
		t.Fatal("the chip group should open with nothing picked")
	}
	tap(t, mgr, "generics")
	tap(t, mgr, "testing")
	if !hasTextContaining(tree(t, mgr), "picked: generics · testing") {
		t.Fatal("both picked topics should be summarized, in topic order")
	}
	tap(t, mgr, "generics")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "picked: testing") || hasTextContaining(cur, "generics ·") {
		t.Fatal("re-tapping a picked chip should remove it from the map")
	}
	assertNoConcerns(t)
}

// The tab-strip arrangement on the same panel: the same SegmentedControl, two
// roles, and a state that comes out as the other ARIA attribute.
//
// The announcement is the whole lesson, so the announcement is what is
// checked, not just that the caption moved. Both halves matter and they fail
// separately — a strip that lost its roles still switches panes, and a strip
// that kept them but stopped stating the unselected tabs still announces the
// live one.
func TestPillsDemoTabStripArrangementAnnouncesEveryTab(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Badges, chips & segments")

	if !hasTextContaining(tree(t, mgr), "showing: Sermons") {
		t.Fatal("the tab strip should open on its first pane")
	}
	tap(t, mgr, "Articles")
	if !hasTextContaining(tree(t, mgr), "showing: Articles") {
		t.Fatal("tapping a tab should move the pane")
	}

	// A Chip renders as a Button carrying its caption in the label prop, not
	// as a Text child, so the tabs are found by that prop rather than by text.
	on, off := 0, 0
	for _, n := range findNodes(tree(t, mgr), func(n *node) bool {
		return n.Type == "Button" && n.Style != nil && n.Style.AccessibilityRole == "tab"
	}) {
		switch n.Style.AccessibilitySelected {
		case "true":
			on++
		case "false":
			off++
		default:
			t.Fatalf("a tab says nothing about its state (%q); an unselected tab has to "+
				"say so, not go quiet", n.Style.AccessibilitySelected)
		}
	}
	if on != 1 || off != 2 {
		t.Fatalf("tabs announced: %d selected, %d unselected — want 1 and 2", on, off)
	}
	assertNoConcerns(t)
}

// --- 4.3 ListRow & Avatar --------------------------------------------------

func TestListRowDemoControlledSelection(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "ListRow & Avatar")

	cur := tree(t, mgr)
	if !hasTextContaining(cur, "No row selected") {
		t.Fatal("the roster should open with no selection")
	}
	// The Leading Avatar derives initials from the name (first word, last
	// word) — "June Gopher" must show as a JG disc.
	if !hasText(cur, "JG") {
		t.Fatal("June Gopher's avatar should render derived JG initials")
	}

	tapRow(t, mgr, "June Gopher")
	if !hasTextContaining(tree(t, mgr), "Selected: June Gopher") {
		t.Fatal("tapping a row should select it")
	}
	// Selection is single: choosing another row moves it, no unselect needed.
	tapRow(t, mgr, "Rex Burrows")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "Selected: Rex Burrows") {
		t.Fatal("tapping another row should move the selection")
	}
	if hasTextContaining(cur, "Selected: June Gopher") {
		t.Fatal("the previous selection should be gone")
	}
	// Re-tapping the selected row clears — the toggle in the row's OnTap.
	tapRow(t, mgr, "Rex Burrows")
	if !hasTextContaining(tree(t, mgr), "No row selected") {
		t.Fatal("re-tapping the selected row should clear the selection")
	}
	assertNoConcerns(t)
}

// The roster is a listbox, and every row in it answers.
//
// The pairing is the subject, as it is for the outline below: an `option` is
// owned by a `listbox`, so the two halves are set in different places by
// different people — the caller roles the container, the widget states the
// row — and either alone is a role naming a structure that is not there.
//
// Both values are asserted, not just the chosen row's. A listbox in which only
// the selection answers announces the rest as plain rows, which is the exact
// failure core.SelectedOff exists to prevent one widget over.
func TestListRowDemoAnnouncesItsSelectionAsAState(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "ListRow & Avatar")

	tapRow(t, mgr, "June Gopher")
	cur := tree(t, mgr)

	box := findNode(cur, func(n *node) bool {
		return n.Style != nil && n.Style.AccessibilityRole == "listbox"
	})
	if box == nil {
		t.Fatal("no role=listbox in the tree — every option under the roster is an orphan, " +
			"and an orphan option is the structure-that-is-not-there failure")
	}

	options := findNodes(box, func(n *node) bool {
		return n.Style != nil && n.Style.AccessibilityRole == "option"
	})
	if len(options) != len(teamMembers) {
		t.Fatalf("%d options under the listbox, want %d", len(options), len(teamMembers))
	}

	on, off := 0, 0
	for i, n := range options {
		switch n.Style.AccessibilitySelected {
		case "true":
			on++
		case "false":
			off++
		default:
			t.Errorf("option %d states %q — an option that says nothing is announced as one "+
				"that cannot be chosen", i, n.Style.AccessibilitySelected)
		}
		// The name must have stopped moving: the state lives in the state now.
		if strings.Contains(n.Style.AccessibilityLabel, ", selected") {
			t.Errorf("option %d still spells the state into its name (%q) — that is the "+
				"fallback for a row with no role to carry it", i, n.Style.AccessibilityLabel)
		}
	}
	if on != 1 {
		t.Errorf("%d options are selected, want exactly 1", on)
	}
	if off != len(teamMembers)-1 {
		t.Errorf("%d options say they are unselected, want %d — the quiet ones read as "+
			"furniture beside the chosen row", off, len(teamMembers)-1)
	}

	assertNoConcerns(t)
}

// A flattened tree announces its depths, and the list around it owns them.
//
// core.Style.AccessibilityNestingLevel had no consumer anywhere in the
// framework until ListRow.NestingLevel: it was exported by both web targets
// and exercised only by their own unit tests. This is the first nested list,
// and it is the shape the field exists for — six rows that are siblings in the
// markup, because a list is a flat run of children, carrying a nesting that
// has nowhere else to live.
//
// The pairing is what the test is really for. A listitem is owned by a list,
// so the two halves are set in different places by different people — the
// widget states the depth, the caller states the container — and either alone
// is a role naming a structure that is not there.
func TestOutlineDemoStatesItsDepths(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "ListRow & Avatar")
	cur := tree(t, mgr)

	list := findNode(cur, func(n *node) bool {
		return n.Style != nil && n.Style.AccessibilityRole == "list"
	})
	if list == nil {
		t.Fatal("no role=list in the tree — every listitem under the demo is an orphan")
	}

	items := findNodes(list, func(n *node) bool {
		return n.Style != nil && n.Style.AccessibilityRole == "listitem"
	})
	if len(items) != len(canonOutline) {
		t.Fatalf("%d listitems under the list, want %d", len(items), len(canonOutline))
	}
	for i, n := range items {
		if got := n.Style.AccessibilityNestingLevel; got != canonOutline[i].depth {
			t.Errorf("row %d (%s): depth = %d, want %d",
				i, canonOutline[i].title, got, canonOutline[i].depth)
		}
	}

	// Three distinct depths, or the demo is an indent with a constant beside
	// it and would keep passing if every row said 1.
	seen := map[int]bool{}
	for _, n := range items {
		seen[n.Style.AccessibilityNestingLevel] = true
	}
	if len(seen) < 3 {
		t.Errorf("the outline spans %d depths, want at least 3 — a flat list states nothing "+
			"aria-level was added for", len(seen))
	}

	// The pixels half. indentBy is one core.PaddingLeft now, where it used to
	// be a whole EdgeInsets through UseStyle — which replaced all four sides,
	// so the helper also owned the top, bottom and right, in numbers copied
	// out of the theme that a theme edit would never reach.
	//
	// The baseline is the same lesson's other ListRow demo: the listbox rows
	// above, which are the same widget with no inset prop on them. Comparing
	// against those and not against literals is the whole point — a test that
	// spelled the theme's numbers would be the third copy of the thing the
	// conversion removed, and it would keep passing if indentBy went back to
	// hardcoding them.
	plain := findNodes(cur, func(n *node) bool {
		return n.Style != nil && n.Style.AccessibilityRole == "option"
	})
	if len(plain) == 0 {
		t.Fatal("no un-indented ListRow on the lesson to measure the indent against")
	}
	base := plain[0].Style.Padding
	if base.Left == 0 || base.Top == 0 {
		t.Fatalf("the baseline row carries no padding to compare against: %+v", base)
	}

	for i, n := range items {
		p := n.Style.Padding
		if want := 16 * canonOutline[i].depth; p.Left != want {
			t.Errorf("row %d (%s): left inset = %d, want %d",
				i, canonOutline[i].title, p.Left, want)
		}
		if p.Top != base.Top || p.Bottom != base.Bottom || p.Right != base.Right {
			t.Errorf("row %d (%s): the indent changed sides it does not own — %+v, "+
				"want top/bottom/right from the widget's own base %+v",
				i, canonOutline[i].title, p, base)
		}
	}

	assertNoConcerns(t)
}

// --- 4.4 Accordion ----------------------------------------------------------

func TestAccordionDemoTogglesContent(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Accordion: the stateful widget")

	// InitiallyExpanded seeds only the first accordion open; the others'
	// answers must not be in the tree at all while collapsed.
	cur := tree(t, mgr)
	if !hasTextContaining(cur, "one and only hook user") {
		t.Fatal("the first accordion should open expanded")
	}
	if hasTextContaining(cur, "reads its neighbor's bool") {
		t.Fatal("a collapsed accordion's content should not render")
	}

	tapRow(t, mgr, "Why not wrap one in core.If?")
	if !hasTextContaining(tree(t, mgr), "reads its neighbor's bool") {
		t.Fatal("tapping a collapsed header should expand its content")
	}
	tapRow(t, mgr, "Where does the open state live?")
	if hasTextContaining(tree(t, mgr), "one and only hook user") {
		t.Fatal("tapping an expanded header should collapse its content")
	}
	assertNoConcerns(t)
}

// --- 4.5 Tabs ---------------------------------------------------------------

func TestTabsDemoSwitchesPagesAndKeepsState(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Tabs & the wire contract")

	// Unlike TabView (where every page is a child), the hand-rolled Match
	// puts only the selected page in the tree.
	cur := tree(t, mgr)
	if !hasTextContaining(cur, "Pages are plain views") {
		t.Fatal("the demo should open on the Info page")
	}
	if hasTextContaining(cur, "gopher quota used") {
		t.Fatal("unselected pages should not be in the tree")
	}

	tap(t, mgr, "Stats")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "gopher quota used") {
		t.Fatal("selecting Stats should switch the page")
	}
	if hasTextContaining(cur, "Pages are plain views") {
		t.Fatal("the Info page should leave the tree when deselected")
	}

	// The Settings checkbox writes a slot declared above the page switch, so
	// its value must survive the page being unmounted and remounted.
	tap(t, mgr, "Settings")
	toggleCheckbox(t, mgr, 0, true)
	if !hasTextContaining(tree(t, mgr), "email notifications: ON") {
		t.Fatal("toggling the Settings checkbox should flip its caption")
	}
	tap(t, mgr, "Info")
	tap(t, mgr, "Settings")
	if !hasTextContaining(tree(t, mgr), "email notifications: ON") {
		t.Fatal("page state declared above the switch should survive switching away and back")
	}
	assertNoConcerns(t)
}

// The half of 4.5 that has no visible effect and is the reason the lesson
// spends a paragraph on it: a hand-assembled strip states the relationship
// core.TabView writes from the node type. The strip and the region it switches
// are siblings drawn identically whether or not either end has been stated, so
// the style is the only place this exists.
func TestTabsDemoWiresTheStripToItsPanel(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Tabs & the wire contract")
	cur := tree(t, mgr)

	// The region, named for whichever page is showing.
	panel := findNode(cur, func(n *node) bool {
		return n.Style != nil && n.Style.AccessibilityID == tabDemoPanelID
	})
	if panel == nil {
		t.Fatalf("no element carries %q, so every segment's aria-controls dangles",
			tabDemoPanelID)
	}
	if panel.Style.AccessibilityLabel != "Info" {
		t.Errorf("panel name = %q, want the showing page's", panel.Style.AccessibilityLabel)
	}

	// The strip claims its children are tabs...
	if findNode(cur, roleIs("tablist")) == nil {
		t.Error("the SegmentedControl row is not a tablist, so its chips announce as a " +
			"row of toggle buttons")
	}
	// ...and every one of them points at the one region. One id for three
	// tabs is right rather than lazy: there is a single region whose contents
	// change, so a per-tab id would name two regions that do not exist.
	tabs := findNodes(cur, roleIs("tab"))
	if len(tabs) != len(tabPageLabels) {
		t.Fatalf("found %d tabs, want %d", len(tabs), len(tabPageLabels))
	}
	for i, tab := range tabs {
		if got := tab.Style.AccessibilityControls; got != tabDemoPanelID {
			t.Errorf("tab %d controls %q, want %q", i, got, tabDemoPanelID)
		}
	}

	// The name follows the selection, so a reader arriving through the
	// relationship is told where it landed.
	tap(t, mgr, "Stats")
	panel = findNode(tree(t, mgr), func(n *node) bool {
		return n.Style != nil && n.Style.AccessibilityID == tabDemoPanelID
	})
	if panel == nil || panel.Style.AccessibilityLabel != "Stats" {
		t.Error("the panel's name did not follow the tab")
	}
	assertNoConcerns(t)
}

// --- 4.6 Collections --------------------------------------------------------

// The banded demo: a Header override placing a comps.CollapseBand.
//
// The lesson is the first thing in the repository that uses CollapseBand at
// all — its only readers were its own tests, which is a slightly bigger hole
// than an unused widget usually is, because its second field (ControlStyle)
// exists to answer a question that only arises when somebody assembles a real
// custom band. So this test is about the assembly rather than about the
// control: the widget's own tests already say what a CollapseBand builds.
//
// Three claims, and each is a different half of the division GroupedList draws
// between the widget and a Header override:
//
//	the run hides                   the widget's half, which reaches past the
//	                                override — the rows are withheld by
//	                                Collapse.hides, not by anything here
//	the control toggles it          the override's half, wired to the *same*
//	                                Collapse, so the chevron and the run
//	                                cannot disagree
//	the count sits outside it       the caller's own chrome, in a row the
//	                                caller laid out
func TestTheBandedDemoCollapsesUnderItsOwnHeader(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Collections: GroupedList & DataTable")

	// Every month starts shut, so a title that appears in no other demo on
	// the page is the address for "the banded list's rows".
	//
	// "A Lamp on a Stand" is in January 2026, which the paged demo above
	// reaches only after two Load more taps and the table only on its last
	// page — so before either is touched, its presence is this list's alone.
	cur := tree(t, mgr)
	if hasText(cur, "A Lamp on a Stand") {
		t.Fatal("the banded demo should open with every month shut")
	}

	band := findBand(cur, "January 2026")
	if band == nil {
		t.Fatal("no button for the January band — a shut run must still have a " +
			"control announcing it, or the rows are gone with no way back")
	}
	if got := band.Style.AccessibilityExpanded; got != "false" {
		t.Fatalf("the shut band announces AccessibilityExpanded %q, want "+
			"\"false\" — the state has to be restated on every pass, so the two "+
			"silent readings (unset, and set to the other value) must not be "+
			"reachable here", got)
	}

	// The badge is the caller's own and lives outside the control, because a
	// reader does not descend into a button: a count put inside it would stop
	// being announced and would join the button's name if it were.
	if hasText(band, "2") {
		t.Fatal("the group's count is inside the button, where it is not announced")
	}
	if !hasText(cur, "2") {
		t.Fatal("the January badge is not in the tree at all")
	}

	// The override's control and the widget's row hiding answer to one state.
	mgr.DispatchCallback(band.Props["onClick"].(string))
	cur = tree(t, mgr)
	if !hasText(cur, "A Lamp on a Stand") {
		t.Fatal("tapping the band did not bring its run back")
	}
	reopened := findBand(cur, "January 2026")
	if reopened == nil {
		t.Fatal("the band lost its control on the way to being open")
	}
	if got := reopened.Style.AccessibilityExpanded; got != "true" {
		t.Fatalf("the open band announces AccessibilityExpanded %q, want \"true\"", got)
	}

	// And back, which is the assertion that a toggle is a toggle rather than a
	// one-way reveal — the demo clones the set on every press.
	mgr.DispatchCallback(reopened.Props["onClick"].(string))
	if hasText(tree(t, mgr), "A Lamp on a Stand") {
		t.Fatal("tapping the open band did not shut its run again")
	}

	// The insets are on the control, which is the whole subject of
	// CollapseBand.ControlStyle: padding on the row around the button is dead
	// space, because the button fills the box it was handed.
	//
	// The trailing inset and the vertical one, and deliberately not the
	// leading 16. A disclosure's control is a core.Row, which arrives carrying
	// the theme's own padding recipe — 16 horizontal, the same number the
	// default band's bandInsets uses — so a leading-edge assertion of 16 is
	// satisfied whether or not the demo asked for anything, which is a check
	// that cannot fail. That is not hypothetical: it is what this assertion
	// was, and a mutation deleting the demo's PaddingLeft passed it. The two
	// numbers below are the demo's own and are not the theme's.
	if got := band.Style.Padding.Right; got != 8 {
		t.Errorf("the band's control is padded %d on the trailing edge, want 8 — "+
			"ControlStyle is not reaching the button, so a caller's chrome would "+
			"land on the row instead, where a press does nothing", got)
	}
	if got := band.Style.Padding.Vertical; got != 10 {
		t.Errorf("the band's control is padded %d vertically, want 10 — same", got)
	}
	assertNoConcerns(t)
}

// The disclosure control of the band whose heading reads label.
//
// A node carrying core.RoleButton rather than a Button node: comps.
// disclosure builds the control as a styled Box, because ARIA's pattern nests
// the button *inside* the heading and the two have to be separate nodes for
// the tier and the name to land on the right one. Matching on the role is
// also what keeps this test honest if the control ever becomes a real Button
// — the claim is "a control announcing itself as one", not "this node type".
func findBand(root *node, label string) *node {
	return findNode(root, func(n *node) bool {
		return n.Style != nil && n.Style.AccessibilityRole == "button" &&
			hasText(n, label)
	})
}

func TestCollectionsDemoSortsPagesAndLoadsMore(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Collections: GroupedList & DataTable")

	cur := tree(t, mgr)
	// Client-side paging: nine rows, four per page, count derived.
	if !hasText(cur, "Page 1 of 3") {
		t.Fatal("the table should open on page 1 of 3")
	}
	if !hasText(cur, "The Narrow Gate") || hasText(cur, "Treasures in Heaven") {
		t.Fatal("page 1 shows the first four rows only")
	}
	tap(t, mgr, "Older ›")
	cur = tree(t, mgr)
	if !hasText(cur, "Page 2 of 3") || !hasText(cur, "Treasures in Heaven") {
		t.Fatal("Next should move to page 2")
	}

	// Sorting is controlled: the header tap lands in the lesson's state and
	// the glyph reflects it; a second tap flips direction.
	tapRow(t, mgr, "Title")
	if !hasText(tree(t, mgr), "Title ▲") {
		t.Fatal("tapping the Title header should sort ascending")
	}
	tapRow(t, mgr, "Title ▲")
	cur = tree(t, mgr)
	if !hasText(cur, "Title ▼") {
		t.Fatal("tapping the active header should flip to descending")
	}
	// Descending by title, page 2 (rows 5-8 of Wise, Two, Treasures,
	// Salt, Narrow, Lamp, Golden, Blessed, Ask) starts at "The Narrow Gate".
	if !hasText(cur, "The Narrow Gate") || hasText(cur, "Wise and Foolish Builders") {
		t.Fatal("the page should show the sorted window")
	}
	tap(t, mgr, "Clear sort")
	if hasTextContaining(tree(t, mgr), "Title ▼") {
		t.Fatal("clearing the sort should drop the glyph")
	}

	// Compact drops the Narrow Speaker column from header and body alike.
	if !hasText(cur, "Speaker") {
		t.Fatal("the Speaker column should be visible before Compact")
	}
	tap(t, mgr, "Compact")
	if hasText(tree(t, mgr), "Speaker") {
		t.Fatal("Compact should hide the Narrow Speaker column")
	}

	// The grouped list starts with one page and grows by Load more until the
	// archive is complete, at which point the tail is gone.
	cur = tree(t, mgr)
	if !hasText(cur, "March 2026") || hasText(cur, "The Two Houses") {
		t.Fatal("the grouped list should open with the first three rows under March")
	}
	tap(t, mgr, "Load more")
	cur = tree(t, mgr)
	if !hasText(cur, "February 2026") || !hasText(cur, "The Two Houses") {
		t.Fatal("Load more should reveal the next three rows and their month header")
	}
	tap(t, mgr, "Load more")
	cur = tree(t, mgr)
	if !hasText(cur, "December 2025") {
		t.Fatal("the whole archive should be loaded")
	}
	if findNode(cur, func(n *node) bool { return n.Type == "Button" && n.Props["label"] == "Load more" }) != nil {
		t.Fatal("a complete list has no Load more tail")
	}
	assertNoConcerns(t)
}

// --- 4.7 Screen furniture -------------------------------------------------

// countNodes is findNode's counting twin — the furniture demo asserts on how
// many placeholder bars a Skeleton drew, which is a number rather than a
// presence.
func countNodes(n *node, pred func(*node) bool) int {
	if n == nil {
		return 0
	}
	count := 0
	if pred(n) {
		count++
	}
	for _, c := range n.Children {
		count += countNodes(c, pred)
	}
	return count
}

// skeletonBars counts the demo's placeholder bars. A Skeleton bar is the only
// thing on this screen that is a filled Box with the widget's 4pt radius, and
// the wire tree carries both fields — which is why the count is addressed this
// way rather than by the accessibility label the widget also sets (nodeStyle
// decodes no accessibility fields).
func skeletonBars(n *node) int {
	return countNodes(n, func(n *node) bool {
		return n.Type == "Box" && n.Style != nil &&
			n.Style.BorderRadius == 4 && n.Style.Background != ""
	})
}

func TestScreenFurnitureDemoSearchesFiltersAndSwitchesStates(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Screen furniture: bars, banners & placeholders")

	cur := tree(t, mgr)
	if !hasTextContaining(cur, "9 of 9 sermons") {
		t.Fatal("the bar's subtitle should open with the whole archive")
	}
	// The lesson is a pushed screen, so CanPop is true and the automatic back
	// control is real — that is the demo's point, and it is also why the demo
	// sets OnBack.
	back := findNode(cur, func(n *node) bool { return n.Type == "Button" && n.Props["label"] == "‹" })
	if back == nil {
		t.Fatal("a pushed lesson should give the demo AppBar its automatic back control")
	}
	mgr.DispatchCallback(back.Props["onClick"].(string))
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "OnBack ran instead of core.Pop") {
		t.Fatal("OnBack should have run")
	}
	// And it must have *replaced* Pop, not preceded it: the lesson is still on
	// screen.
	if !hasTextContaining(cur, "9 of 9 sermons") {
		t.Fatal("tapping back popped the lesson — OnBack should replace core.Pop")
	}

	// The field is controlled, so the keystroke lands immediately whatever the
	// debounce is doing. (Whether the *list* has moved yet is a race by
	// construction, so the assertions below go through the paths that are
	// synchronous: OnSubmit, which cancels and applies, and awaitText.)
	typeInto(t, mgr, "salt")
	if findNode(tree(t, mgr), func(n *node) bool {
		return n.Type == "Input" && n.Props["value"] == "salt"
	}) == nil {
		t.Fatal("the field should show the keystroke immediately")
	}
	awaitText(t, mgr, "1 of 9 sermons")
	cur = tree(t, mgr)
	if !hasText(cur, "Salt and Light") || hasText(cur, "The Narrow Gate") {
		t.Fatal("the debounced query should have narrowed the list")
	}

	// Enter means now: Cancel drops the pending call, then the caller acts.
	typeInto(t, mgr, "meek")
	submit := findNode(tree(t, mgr), func(n *node) bool { return n.Type == "Input" })
	mgr.DispatchCallback(submit.Props["onSubmit"].(string))
	cur = tree(t, mgr)
	if !hasText(cur, "Blessed Are the Meek") || hasText(cur, "Salt and Light") {
		t.Fatal("OnSubmit should apply the query synchronously")
	}

	// Nothing matches: the empty state, and its action as the way out of the
	// state that produced it.
	typeInto(t, mgr, "zzz")
	mgr.DispatchCallback(findNode(tree(t, mgr), func(n *node) bool { return n.Type == "Input" }).Props["onSubmit"].(string))
	cur = tree(t, mgr)
	if !hasText(cur, "Nothing matches that") {
		t.Fatal("a query with no matches should show the EmptyState")
	}
	tap(t, mgr, "Clear filters")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "9 of 9 sermons") || !hasText(cur, "The Narrow Gate") {
		t.Fatal("Clear filters should restore the whole archive")
	}

	// The chip strip is a second, independent filter, and the StatTile reads
	// the same sentinel the "All" chip does.
	tap(t, mgr, "R. Okafor")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "3 of 9 sermons") {
		t.Fatal("the speaker chip should narrow the list")
	}
	if !hasText(cur, "R. Okafor") || hasText(cur, "The Narrow Gate") {
		t.Fatal("only that speaker's sermons should remain")
	}
	tap(t, mgr, "All")
	if !hasTextContaining(tree(t, mgr), "9 of 9 sermons") {
		t.Fatal("the All chip should clear the speaker filter")
	}

	assertNoConcerns(t)
}

func TestScreenFurnitureDemoBannerAndSkeleton(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Screen furniture: bars, banners & placeholders")

	if n := skeletonBars(tree(t, mgr)); n != 0 {
		t.Fatalf("a loaded list should draw no placeholder bars, got %d", n)
	}

	// The bar's own refresh action drives the loading state, so the demo shows
	// an AppBar action doing something rather than a decorative glyph.
	tap(t, mgr, "↻")
	cur := tree(t, mgr)
	if n := skeletonBars(cur); n != 3 {
		t.Fatalf("the loading state should draw 3 placeholder bars, got %d", n)
	}
	if hasText(cur, "The Narrow Gate") {
		t.Fatal("the skeleton should stand in for the rows, not sit beside them")
	}
	tap(t, mgr, "↻")
	if !hasText(tree(t, mgr), "The Narrow Gate") {
		t.Fatal("the rows should come back")
	}

	// The banner is conditional, so it costs no node when there is nothing to
	// say — the idiom core.If exists for.
	if hasTextContaining(tree(t, mgr), "Showing a saved copy") {
		t.Fatal("no banner before anything has gone wrong")
	}
	tap(t, mgr, "Simulate offline")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "Showing a saved copy") {
		t.Fatal("the offline simulation should raise the banner")
	}
	// Warning's glyph, and the action that clears the condition.
	if !hasText(cur, "⚠") {
		t.Fatal("a warning banner should carry the warning glyph")
	}
	tap(t, mgr, "Reconnect")
	if hasTextContaining(tree(t, mgr), "Showing a saved copy") {
		t.Fatal("the action should have cleared the banner")
	}

	assertNoConcerns(t)
}

// --- 4.8 Endless feeds -----------------------------------------------------

// The three props of the lesson, seen through the wire tree the renderers
// actually receive. None of them can be driven here the way a tap can —
// scrolling is the trigger for two of them and there is no viewport in a test
// — so what is checked is that each one reaches the tree in the shape the
// four renderers were written to read, and that the callback behind the edge
// does what the demo says when it is fired.
func TestEndlessFeedDemoCarriesTheThreeCollectionProps(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Endless feeds: sticky bands & sideways strips")

	cur := tree(t, mgr)

	// B1: the chip strip is a Scroll on the row axis, not a wrapping Row.
	strip := findNode(cur, func(n *node) bool {
		return n.Type == "Scroll" && n.Style != nil && n.Style.FlexDirection == "row"
	})
	if strip == nil {
		t.Fatal("ChipStrip{Scrollable: true} should render a Scroll on the row axis")
	}
	if strip.Style.Overflow != "auto" {
		t.Fatalf("the strip's overflow is %q — without it the browser clips the row "+
			"instead of panning it", strip.Style.Overflow)
	}

	// B3: every group band carries the sticky marker, and the rows do not.
	list := findNode(cur, func(n *node) bool { return n.Type == "List" })
	if list == nil {
		t.Fatal("the demo should render a List")
	}
	bands := 0
	for _, child := range list.Children {
		sticky := child.Style != nil && child.Style.Position == "sticky"
		if strings.HasPrefix(child.Key, "group:") {
			bands++
			if !sticky {
				t.Errorf("band %q is not pinned", child.Key)
			}
			continue
		}
		if sticky {
			t.Errorf("row %q was pinned along with the bands", child.Key)
		}
	}
	if bands == 0 {
		t.Fatal("no group bands rendered — StickyHeaders has nothing to pin without GroupBy")
	}

	// B2: the edge is advertised on the List itself.
	if _, ok := list.Props["onEndReached"].(string); !ok {
		t.Fatalf("the List carries no onEndReached prop: %#v", list.Props)
	}

	assertNoConcerns(t)
}

// The debounce, seen from the one angle a test without a viewport can reach.
//
// Each report here is followed by a real render, so consecutive reports are
// genuinely different bottoms and each one legitimately loads a page — that
// is the feature working, not the guard failing. What the guard is for is the
// case where a report changes nothing: once the archive is exhausted, a page
// adds no rows, and every further report of that same bottom must be refused.
// A feed that re-asked forever at the end of itself is exactly the bug
// core.OnEndReached's row-count ledger exists to prevent.
func TestEndlessFeedStopsAskingOnceTheArchiveIsExhausted(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Endless feeds: sticky bands & sideways strips")

	edge := func() string {
		list := findNode(tree(t, mgr), func(n *node) bool { return n.Type == "List" })
		if list == nil {
			t.Fatal("the demo should render a List")
		}
		id, _ := list.Props["onEndReached"].(string)
		if id == "" {
			t.Fatal("the List carries no onEndReached prop")
		}
		return id
	}

	if !hasTextContaining(tree(t, mgr), "2 of 9 rows, 0 fetches") {
		t.Fatal("the demo should open with one page loaded and nothing fetched")
	}

	// One report, one page: the ordinary case.
	mgr.DispatchCallback(edge())
	if !hasTextContaining(tree(t, mgr), "4 of 9 rows, 1 fetches") {
		t.Fatal("reaching the end should have loaded the next page")
	}

	// Read to the bottom of the fixture. Generous enough to exhaust it
	// whatever the page size is, which is the point: the loop is not what is
	// under test.
	for range 10 {
		mgr.DispatchCallback(edge())
	}
	settled := feedCaption(t, tree(t, mgr))
	if !strings.HasPrefix(settled, "9 of 9 rows") {
		t.Fatalf("the archive should be exhausted by now, caption reads %q", settled)
	}

	// Now the reports that must go nowhere: the bottom is the same bottom,
	// because the last page added no rows.
	mgr.DispatchCallback(edge())
	mgr.DispatchCallback(edge())
	if got := feedCaption(t, tree(t, mgr)); got != settled {
		t.Fatalf("the feed kept fetching at the end of itself: %q became %q", settled, got)
	}

	assertNoConcerns(t)
}

// feedCaption returns the demo's "N of M rows, K fetches" line, which is the
// only place the fetch count is observable from outside.
func feedCaption(t *testing.T, n *node) string {
	t.Helper()
	found := findNode(n, func(n *node) bool {
		content, ok := n.Props["content"].(string)
		return n.Type == "Text" && ok && strings.Contains(content, "fetches")
	})
	if found == nil {
		t.Fatal("the demo should show a row/fetch caption")
	}
	return found.Props["content"].(string)
}

// --- 4.9 Calendars ---------------------------------------------------------

// The demo's whole claim is that one selection drives two widgets: tap a day
// in the grid and the field under it reports the same date, because neither
// widget owns it.
func TestCalendarDemoSharesOneSelectionWithTheField(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Calendars: a month grid and a date field")

	cur := tree(t, mgr)
	if !hasTextContaining(cur, "March 2026") {
		t.Fatal("the demo should open on the month its pinned today falls in")
	}
	if !hasTextContaining(cur, "Tap a dotted day") {
		t.Fatal("with nothing selected the caption should be the prompt")
	}

	// March 15 2026 carries "Salt and Light" in the 4.6 archive, so it is one
	// of the dotted days. A day cell is a tappable Box holding a numeral, not
	// a Button holding a caption, so the tap is addressed by the cell's spoken
	// name — which is also the only unambiguous handle on it, since the
	// numeral "15" appears in the field's sheet as well as in the grid.
	cell := findNode(cur, func(n *node) bool {
		_, clickable := n.Props["onClick"].(string)
		return clickable && n.Style != nil && n.Style.AccessibilityLabel == "Sunday, March 15, 2026"
	})
	if cell == nil {
		t.Fatal("no day cell named \"Sunday, March 15, 2026\" — the grid should name its cells for screen readers")
	}
	mgr.DispatchCallback(cell.Props["onClick"].(string))

	cur = tree(t, mgr)
	if !hasTextContaining(cur, "Salt and Light") {
		t.Fatal("selecting a dotted day should surface what is on it")
	}
	// The same state reaches the DatePicker's trigger, in its own layout.
	if !hasTextContaining(cur, "Mar 15, 2026") {
		t.Fatal("the date field should summarize the selection the grid just made")
	}

	assertNoConcerns(t)
}

// The two things the counted Marked and the deselectable grid buy this demo,
// neither of which a caption can show: the cluster under a day is as many dots
// as the day has things on it, and tapping the chosen day again empties the
// selection the field below shares.
func TestCalendarDemoCountsItsDotsAndClearsOnASecondTap(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Calendars: a month grid and a date field")
	cur := tree(t, mgr)

	// dots returns the mark cluster of the named cell: the cell's second
	// child, after the numeral. findNode is depth-first and the standalone
	// grid precedes the DatePicker's, whose sheet carries cells with the very
	// same spoken names.
	dots := func(name string) []*node {
		t.Helper()
		cell := findNode(cur, func(n *node) bool {
			return n.Style != nil && strings.HasPrefix(n.Style.AccessibilityLabel, name)
		})
		if cell == nil {
			t.Fatalf("no day cell named %q", name)
		}
		if len(cell.Children) < 2 {
			t.Fatalf("cell %q has %d children, want the numeral and the cluster", name, len(cell.Children))
		}
		return cell.Children[1].Children
	}
	inked := func(name string) int {
		n := 0
		for _, d := range dots(name) {
			if d.Style != nil && d.Style.Background != "#00000000" {
				n++
			}
		}
		return n
	}

	// March 1 carries "Ask, Seek, Knock" plus two other things, March 15 a
	// sermon plus one, March 22 a sermon alone, March 8 nothing at all.
	for _, tc := range []struct {
		name string
		want int
	}{
		{"Sunday, March 1, 2026", 3},
		{"Sunday, March 15, 2026", 2},
		{"Sunday, March 22, 2026", 1},
		{"Sunday, March 8, 2026", 0},
	} {
		if got := inked(tc.name); got != tc.want {
			t.Errorf("%s: %d dots inked, want %d", tc.name, got, tc.want)
		}
	}
	// The empty day still carries its placeholder, which is what keeps the
	// numerals on one baseline across the grid.
	if n := len(dots("Sunday, March 8, 2026")); n != 1 {
		t.Errorf("an unmarked day holds %d dot boxes, want the one transparent placeholder", n)
	}

	// tap fires the first cell whose spoken name starts with name.
	tap := func(name string) {
		t.Helper()
		cell := findNode(cur, func(n *node) bool {
			if n.Style == nil || !strings.HasPrefix(n.Style.AccessibilityLabel, name) {
				return false
			}
			_, clickable := n.Props["onClick"].(string)
			return clickable
		})
		if cell == nil {
			t.Fatalf("no tappable day cell named %q", name)
		}
		mgr.DispatchCallback(cell.Props["onClick"].(string))
		cur = tree(t, mgr)
	}

	tap("Sunday, March 15, 2026")
	if !hasTextContaining(cur, "Salt and Light") {
		t.Fatal("the first tap should select the day and surface what is on it")
	}
	if !hasTextContaining(cur, "and 1 more that day") {
		t.Error("the caption should name what the second dot stands for; the grid can count and cannot say")
	}

	// The same cell again — now announced as selected, which is why the tap
	// helper matches on a prefix.
	tap("Sunday, March 15, 2026")
	if hasTextContaining(cur, "Salt and Light") {
		t.Fatal("a second tap on the chosen day should clear the selection, not keep it")
	}
	if !hasTextContaining(cur, "Tap a dotted day") {
		t.Error("with the selection cleared the caption should be the prompt again")
	}

	assertNoConcerns(t)
}

// Today is a field, and the range bounds the arrows. Both are checked through
// the tree because both are invisible to a caption.
func TestCalendarDemoRingsItsPinnedTodayAndStopsAtTheRange(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Calendars: a month grid and a date field")
	cur := tree(t, mgr)

	today := findNode(cur, func(n *node) bool {
		return n.Style != nil && n.Style.AccessibilityLabel == "Wednesday, March 11, 2026, today"
	})
	if today == nil {
		t.Fatal("the pinned today should be announced as today")
	}
	if today.Style.BorderWidth != 1 {
		t.Errorf("today is drawn without its ring: border width %v", today.Style.BorderWidth)
	}

	// March 2026 is the demo's Max month, so the forward arrow has nowhere to
	// go and says so rather than paging into an empty April.
	fwd := findNode(cur, func(n *node) bool {
		return n.Type == "Button" && n.Style != nil && n.Style.AccessibilityLabel == "Next month"
	})
	if fwd == nil {
		t.Fatal("the header should carry a next-month arrow")
	}
	if !fwd.Style.Disabled {
		t.Error("the forward arrow should be dead at the top of the range")
	}
	back := findNode(cur, func(n *node) bool {
		return n.Type == "Button" && n.Style != nil && n.Style.AccessibilityLabel == "Previous month"
	})
	if back == nil || back.Style.Disabled {
		t.Error("February is inside the range, so the back arrow should be live")
	}

	assertNoConcerns(t)
}

// The picker's own two states: the sheet opens on the trigger and closes on
// the tap that picks, with no Done button in between.
func TestCalendarDemoDatePickerOpensAndPicksInOneTap(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Calendars: a month grid and a date field")

	modal := func() *node {
		return findNode(tree(t, mgr), func(n *node) bool { return n.Type == "Modal" })
	}
	if m := modal(); m == nil || m.Props["visible"] != false {
		t.Fatal("the picker's sheet should start closed")
	}

	tapRow(t, mgr, "Choose a date")
	if modal().Props["visible"] != true {
		t.Fatal("tapping the trigger should open the sheet")
	}

	// The sheet's grid follows the same selection, so it opens on March too.
	// Searched inside the Modal, not the page: the lesson renders two
	// calendars over one selection, and the inline one carries a cell with
	// this very name.
	cell := findNode(modal(), func(n *node) bool {
		_, clickable := n.Props["onClick"].(string)
		return clickable && n.Style != nil && n.Style.AccessibilityLabel == "Sunday, March 22, 2026"
	})
	if cell == nil {
		t.Fatal("the sheet should open on the month the field is showing")
	}
	mgr.DispatchCallback(cell.Props["onClick"].(string))

	cur := tree(t, mgr)
	if findNode(cur, func(n *node) bool { return n.Type == "Modal" }).Props["visible"] != false {
		t.Error("picking a day should close the sheet — there is nothing left to confirm")
	}
	if !hasTextContaining(cur, "The Narrow Gate") {
		t.Error("the picker's selection is the demo's one selection and should reach the caption")
	}

	assertNoConcerns(t)
}

// --- 4.10 the compass -------------------------------------------------------

// The rose turns against the chosen bearing, and the caption names the
// rotation it applied. A sign error here is invisible in prose and obvious in
// the tree.
func TestCompassDemoTurnsTheRoseAgainstTheChosenBearing(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Sensors: the compass")

	cur := tree(t, mgr)
	// The lesson opens on 45 degrees.
	if !hasTextContaining(cur, "45° NE") {
		t.Fatal("the demo should open showing its 45 degree bearing")
	}
	rose := findNode(cur, func(n *node) bool {
		return n.Style != nil && n.Style.Rotate == -45
	})
	if rose == nil {
		t.Fatal("no node rotated by -45: the rose should turn against the heading")
	}

	// 359 is the chip that matters. A compass that looks right in the middle
	// of the circle and wrong beside north has a normalisation bug, and the
	// readout is where it shows.
	//
	// A Chip renders as a Button carrying its caption in the "label" prop
	// rather than as a Text child, so it is found by the prop and not by
	// hasTextContaining — the same distinction the Load-more and arrow
	// lookups above make.
	chip := findNode(cur, func(n *node) bool {
		return n.Type == "Button" && n.Props["label"] == "359°"
	})
	if chip == nil {
		t.Fatal("no 359° chip — the seam is the bearing worth being able to pick")
	}
	mgr.DispatchCallback(chip.Props["onClick"].(string))

	cur = tree(t, mgr)
	if !hasTextContaining(cur, "359° N") {
		t.Fatal("a bearing one degree short of north should read as N, not NNW")
	}
	if findNode(cur, func(n *node) bool { return n.Style != nil && n.Style.Rotate == -359 }) == nil {
		t.Fatal("the rose should carry the unfolded -359, not a normalised 1")
	}

	assertNoConcerns(t)
}

// The live half, in the state a test always finds it in: mounted, sensor
// started, no host event yet. "Waiting" and "no compass" are different
// sentences and this is the one that must be showing.
func TestCompassDemoDistinguishesWaitingFromAbsent(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Sensors: the compass")

	cur := tree(t, mgr)
	if !hasTextContaining(cur, "Waiting for the first reading") {
		t.Fatal("with no reading delivered the live panel should say it is waiting")
	}
	if hasTextContaining(cur, "No compass here") {
		t.Fatal("a sensor that has not answered yet was reported as absent")
	}
	if !hasTextContaining(cur, "Received=false") {
		t.Fatal("the live panel should show the flags it is explaining")
	}

	assertNoConcerns(t)
}

// The permission half of the same lesson, in the state a test is always in:
// no host attached, so nothing can grant anything.
//
// Unavailable rather than Unknown is the assertion that matters. The check is
// asynchronous, so the obvious implementation leaves a headless run sitting on
// the zero value forever — which is exactly the state the screen draws
// "Checking…" for. permission.send short-circuits instead, because "there is
// no platform to ask" is what Unavailable means, and a Go test is one of the
// three places (with a static export and an unwired embedder) that is really
// in it.
//
// The lesson must also offer no button here: an "ask" that cannot ask is the
// dead control the four-value Status exists to prevent.
func TestCompassDemoReportsLocationAsUnavailableWithNoHost(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Sensors: the compass")

	cur := tree(t, mgr)
	if !hasTextContaining(cur, "permission.Location — unavailable") {
		t.Fatal("a headless run should resolve the permission immediately, not sit on the " +
			"zero value drawing its placeholder")
	}
	if hasTextContaining(cur, "Checking…") {
		t.Fatal("the check never resolved; a permission-gated screen would spin forever " +
			"under test")
	}
	if findNode(cur, func(n *node) bool {
		return n.Type == "Button" && n.Props["label"] == "Use my location"
	}) != nil {
		t.Fatal("an ask button on a platform that cannot grant anything — the button is " +
			"only for Prompt, which is the one status where asking does something")
	}

	assertNoConcerns(t)
}

// The accordion lesson's three FAQ headers each announce as a disclosure, and
// the state follows the taps.
//
// Through the live app rather than the widget, because what is being checked
// is that the lesson *demonstrates* what its prose now teaches: three
// accordions on one screen, one open and two shut, is the arrangement that
// makes ExpandedClosed visible as a value rather than as an absence.
//
// It is also the assertion with no visible effect. The chevrons already flip,
// the sections already open, and every one of those keeps working if the state
// is deleted — which is exactly how the widget shipped for as long as it did.
func TestTheAccordionDemoAnnouncesEachHeaderAsADisclosure(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Accordion: the stateful widget")

	states := func() []string {
		var out []string
		for _, n := range findNodes(tree(t, mgr), roleIs("button")) {
			if n.Style.AccessibilityExpanded != "" {
				out = append(out, string(n.Style.AccessibilityExpanded))
			}
		}
		return out
	}

	// The seeded arrangement: accordionFAQ opens the first entry and leaves
	// the other two shut. Every header answers — the two shut ones are what a
	// two-valued field would have left silent.
	got := states()
	// The literals are ARIA's own spellings, which is what crosses the wire —
	// core.ExpandedOpen and core.ExpandedClosed are these two strings, and
	// this side of the bridge sees only the strings.
	want := []string{"true", "false", "false"}
	if len(got) != len(want) {
		t.Fatalf("found %d disclosure headers, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("header %d states %q, want %q", i, got[i], want[i])
		}
	}

	// And it moves. Tapping the second question opens it, which is the half a
	// hard-coded state would still pass the check above with.
	//
	// Not tap(): an accordion header is a Row with an OnClick, not a Button
	// node — the tap target is the whole row, which is what lets the chevron
	// and the title share it. So the header is found by its button *role*, the
	// thing the widget now states, and the tree is walked the way openLesson
	// walks it.
	header := findNodes(tree(t, mgr), roleIs("button"))[1]
	mgr.DispatchCallback(header.Props["onClick"].(string))
	if got := states(); got[1] != "true" {
		t.Errorf("after the tap the second header states %q, want \"true\" — the state is "+
			"not following the widget's own bool", got[1])
	}
}

// --- The heading outline, end to end ----------------------------------------

// A lesson screen carries a 1-2-3 outline and nobody wrote the 3.
//
// The accordion lesson is the one screen in the app where all three tiers are
// live at once, which is what makes it the place to assert them: the lesson's
// own name is level 1 (lessonHeader, a call site — a lesson screen has no
// AppBar to claim it), "Key points" is level 2 (keyPoints, also a call site),
// and each FAQ question is level 3 straight out of comps.Accordion's
// default. Before that default existed the third tier had no consumer anywhere
// in the framework and levels 3 through 6 were plumbing nothing reached.
//
// Through the live app rather than against the widget, because the widget test
// can only see its own node. What is being checked here is a *relationship* —
// that the three tiers nest — and that only exists once three widgets from
// three different files are on one screen.
func TestALessonScreenHasAThreeTierOutline(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Accordion: the stateful widget")
	cur := tree(t, mgr)

	// Two shapes of heading, because the third tier changed shape when
	// comps.Accordion's header became a button. Levels 1 and 2 ride the
	// words, which is where every heading in the components package lives;
	// level 3 rides the Box wrapped around an accordion's header row, named by
	// an explicit AccessibilityLabel, because the row itself has to be the
	// button that carries aria-expanded. Both are matched here so the
	// *relationship* — the thing this test exists for — is still asserted
	// across the change rather than around it.
	levelOf := func(content string) int {
		n := findNode(cur, func(n *node) bool {
			if n.Style == nil || n.Style.AccessibilityRole != "heading" {
				return false
			}
			return (n.Type == "Text" && n.Props["content"] == content) ||
				n.Style.AccessibilityLabel == content
		})
		if n == nil {
			t.Fatalf("no heading reading %q on the lesson screen", content)
		}
		return n.Style.AccessibilityHeadingLevel
	}

	if got := levelOf("4.4  Accordion: the stateful widget"); got != 1 {
		t.Errorf("the lesson's own name is level %d, want 1 — there is no AppBar above it", got)
	}
	if got := levelOf("Key points"); got != 2 {
		t.Errorf("the recap is level %d, want 2 — a section of the lesson", got)
	}
	if got := levelOf("Where does the open state live?"); got != 3 {
		t.Errorf("an accordion question is level %d, want 3 — a disclosure inside a section", got)
	}

	// And every heading on the screen states a tier. A role with no level
	// announces a heading a reader cannot place, which is the state the
	// outline exists to end; one unplaced heading among three placed ones is
	// worse than none at all, because the reader trusts the other two.
	for _, n := range findNodes(cur, func(n *node) bool {
		return n.Style != nil && n.Style.AccessibilityRole == "heading"
	}) {
		level := n.Style.AccessibilityHeadingLevel
		if level < 1 || level > 6 {
			// Named by content where there is any, and by the accessibility
			// label otherwise — an accordion's heading is a wrapper Box with
			// no text of its own.
			name, _ := n.Props["content"].(string)
			if name == "" {
				name = n.Style.AccessibilityLabel
			}
			t.Errorf("heading %q carries level %d, which every target drops", name, level)
		}
	}

	assertNoConcerns(t)
}

// roleIs matches a node by its core.AccessibilityRole. The wire carries the
// role as the ARIA spelling, which is the string every renderer reads.
func roleIs(role string) func(*node) bool {
	return func(n *node) bool {
		return n.Style != nil && n.Style.AccessibilityRole == role
	}
}

// --- 4.11 Maps ------------------------------------------------------------

// The map lesson's demo is a URL builder with a picture attached, so what the
// test drives is the three knobs and what it reads is the URL the widget
// actually requested — which the lesson prints, for exactly this reason.
//
// The two awkward coordinates are the point of the last assertions: a widget
// that looks right over Lisbon and wrong at 179°E has a wrapping bug, and the
// clamp and the wrap are two different rules that a single "normalise" would
// have collapsed.
func TestStaticMapDemoBuildsTheURLItPrints(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Maps: a picture and a hand-off")

	cur := tree(t, mgr)
	if !hasTextContaining(cur, "Requested: https://staticmap.openstreetmap.de") {
		t.Fatal("the demo should print the URL it requested through the default provider")
	}
	// Lisbon, the marker on, street zoom: the seeded state.
	if !hasTextContaining(cur, "center=38.7223%2C-9.1393") {
		t.Error("the seeded place should be centred in the URL")
	}
	if !hasTextContaining(cur, "markers=") {
		t.Error("the marker switch starts on, so the URL should carry a marker")
	}

	// The switch is a setting: one toggle, no confirmation, and the URL moves.
	toggleBool(t, mgr, "Switch", 0, false)
	if hasTextContaining(tree(t, mgr), "markers=") {
		t.Error("turning the marker off should drop it from the requested URL")
	}

	// A zoom chip, which is the other half of what a provider is handed.
	tap(t, mgr, "Building")
	if !hasTextContaining(tree(t, mgr), "zoom=18") {
		t.Error("the Building chip should request zoom 18")
	}

	// The antimeridian: 178.4419°E is an ordinary longitude and must survive.
	tap(t, mgr, "Suva")
	if !hasTextContaining(tree(t, mgr), "center=-18.1416%2C178.4419") {
		t.Error("a longitude just short of 180 must pass through untouched")
	}

	// And past the pole, where the latitude is held at Mercator's limit.
	tap(t, mgr, "Past the pole")
	if !hasTextContaining(tree(t, mgr), "center=85.0511%2C20") {
		t.Error("a latitude past Mercator's limit should be clamped to it")
	}
	assertNoConcerns(t)
}

// --- 4.12 Live maps -------------------------------------------------------

// mapNodeProp reads one prop off the single MapView in the current tree. The
// map's three callbacks are what the demo is made of, and they are addressed by
// name rather than by position because they all ride the same text channel.
func mapNodeProp(t *testing.T, mgr *render.Manager, prop string) string {
	t.Helper()
	n := findNode(tree(t, mgr), func(n *node) bool { return n.Type == "MapView" })
	if n == nil {
		t.Fatal("no MapView in the current tree")
	}
	id, ok := n.Props[prop].(string)
	if !ok {
		t.Fatalf("MapView has no %s callback: %v", prop, n.Props)
	}
	return id
}

// markerIDs lists the ids of the map's Marker children, in tree order — which is
// also the order the keys were assigned, so this is what a test asserting "the
// pin was added and the others were left alone" reads.
func markerIDs(t *testing.T, mgr *render.Manager) []string {
	t.Helper()
	var out []string
	for _, n := range findNodes(tree(t, mgr), func(n *node) bool { return n.Type == "Marker" }) {
		id, _ := n.Props["id"].(string)
		out = append(out, id)
	}
	return out
}

// The live-map lesson drives all three map callbacks, and the assertion that
// matters most is the *negative* one: a reported region must not move the region
// the demo is asking for. That separation is the lesson, and a demo that echoed
// it would show nothing.
func TestLiveMapDemoAddsPinsAndKeepsTheTwoRegionsApart(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Live maps: markers and the echo guard")

	if got := markerIDs(t, mgr); len(got) != 3 || got[0] != "rossio" {
		t.Fatalf("seed markers = %v, want the three landmarks", got)
	}
	if !hasTextContaining(tree(t, mgr), "Nothing reported yet") {
		t.Fatal("the reported line should start empty")
	}

	// A map tap drops a pin and selects it, which is core.OnMapTap's whole
	// purpose — a "choose a place" screen.
	mgr.DispatchTextCallback(mapNodeProp(t, mgr, "onMapTap"), "38.72,-9.13")
	cur := tree(t, mgr)
	if got := markerIDs(t, mgr); len(got) != 4 || got[3] != "pin-1" {
		t.Fatalf("markers after a map tap = %v, want a fourth keyed pin-1", got)
	}
	if !hasTextContaining(cur, "pin-1 selected") {
		t.Error("a dropped pin should be the selected one")
	}

	// A marker tap reports the id, not a coordinate: the id is what an app has
	// an index of.
	mgr.DispatchTextCallback(mapNodeProp(t, mgr, "onMarkerTap"), "belem")
	if !hasTextContaining(tree(t, mgr), "belem selected") {
		t.Error("a marker tap should select by id")
	}

	// A reported region moves the readout and nothing else. The demo renders
	// from its own region slot, so this is the visible form of "Go merely
	// re-rendering is not an instruction".
	mgr.DispatchTextCallback(mapNodeProp(t, mgr, "onRegionChange"), "40.1234,-8.5,16.5")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "Reported: 40.1234, -8.5000 at zoom 16.50") {
		t.Error("the reported region did not reach the readout")
	}
	if !hasTextContaining(cur, "Asking for: 38.7139, -9.1394 at zoom 13.00") {
		t.Error("a reported pan moved the region the app is asking for — the two " +
			"slots are the lesson")
	}

	// And a button is an instruction, which is the other half.
	tap(t, mgr, "Show Belém")
	if !hasTextContaining(tree(t, mgr), "Asking for: 38.6970, -9.2065 at zoom 15.00") {
		t.Error("the button did not move the region the app is asking for")
	}

	tap(t, mgr, "Reset pins")
	if got := markerIDs(t, mgr); len(got) != 3 {
		t.Errorf("markers after a reset = %v, want the three seeds", got)
	}
	assertNoConcerns(t)
}

// The live map's position readout: the pair of hooks that used to have no
// consumer anywhere in the repository.
//
// # Why this test is behavioural and not a grep
//
// Before this lesson panel existed, `hooks.UseLocation` appeared twice in
// examples/tutorial and both were inside strings — one prose(...) and one
// codeBlock(...). So the hook, core.OnLocation's subscription path and both
// native sensors had no consumer any run could exercise, and a source search
// for the name said otherwise. This drives the rendered tree instead: the
// readout can only say what it says if the hook actually ran.
//
// # What a headless run is, and why that is the interesting case
//
// A Go test registers no system-event sink, so permission.Check resolves to
// Unavailable immediately and core.StartLocation reaches nothing — which is
// the state a static export and a browser with no geolocation are also in.
// The lesson has to say so rather than spin, and the ordering that makes that
// true is the permission being consulted before Received: with Received first
// this screen would print "waiting for the first fix" forever on every host
// that can never have one.
//
// The hooks being called unconditionally is checked by assertNoConcerns, which
// is this package's TestMain audit — a UseLocation inside the permission's
// Granted arm fails there, which is how the shape of the panel was decided and
// why hooks.UseLocation's own doc no longer recommends the other one.
func TestLiveMapDemoRunsTheLocationHookAndSaysWhenItCannot(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Live maps: markers and the echo guard")
	cur := tree(t, mgr)

	if !hasTextContaining(cur, "No position, and none on the way") {
		t.Error("the fix readout does not report a headless run as hopeless — a host " +
			"that can never be granted the permission must not be shown a spinner")
	}
	if hasTextContaining(cur, "Waiting for the first fix") {
		t.Error("the readout is waiting for a fix on a host with no sensor at all: " +
			"Received is only a question about time once a permission exists for the " +
			"fix to arrive under")
	}
	if !hasTextContaining(cur, "This platform cannot grant it at all") {
		t.Error("the permission line does not draw the Unavailable state, so a reader " +
			"in a browser preview is told nothing about why there is no position")
	}
	if hasTextContaining(cur, "Checking…") {
		t.Error("the permission check never resolved; the panel would spin forever " +
			"under test")
	}

	// The ask button belongs to Prompt alone — asking is pointless on a
	// platform that cannot grant, and the compass lesson pins the same rule.
	if findNode(cur, func(n *node) bool {
		return n.Type == "Button" && n.Props["label"] == "Use my location"
	}) != nil {
		t.Error("an ask button on a platform that cannot grant anything")
	}

	// "Centre the map on me" is disabled rather than absent, because a tap
	// with no fix would set the region to 0,0 — a real place in the Gulf of
	// Guinea, and a map that looks like it worked.
	centre := findNode(cur, func(n *node) bool {
		return n.Type == "Button" && n.Props["label"] == "Centre the map on me"
	})
	if centre == nil {
		t.Fatal("the centre-on-me button is missing, so the codeBlock above it shows " +
			"a line the panel does not run")
	}
	if centre.Style == nil || !centre.Style.Disabled {
		t.Error("the centre-on-me button is live with no fix in hand — tapping it " +
			"would centre the map on 0,0 and look like a working feature")
	}

	// And the fix reaches the readout when one arrives, which is the path
	// core.OnLocation -> ctx.RequestRender -> this text exists for. Fed
	// through core.ReceiveLocation, the typed entry every host funnels into.
	//
	// core's location record is process-wide, so this leaks into whatever runs
	// next unless it is put back. The cleanup can only put back so much —
	// Received is a one-way latch by design, since "this device has answered
	// once" is exactly what separates "not yet" from "never" — but an
	// unavailable fix restores the sentence the panel shows, which is what a
	// later test would read.
	t.Cleanup(func() {
		core.ReceiveLocation(core.Location{Error: "no location host in a test"})
	})
	core.ReceiveLocation(core.Location{
		Lat: 38.7223, Lng: -9.1393, Accuracy: 12, Available: true,
	})
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "38.72230, -9.13930 — accurate to about 12 m") {
		t.Error("a fix delivered to core never reached the lesson's readout, so the " +
			"hook's subscription is not wired")
	}
	if hasTextContaining(cur, "No position, and none on the way") {
		t.Error("the readout still says there is no position while holding one")
	}

	assertNoConcerns(t)
}

// The second gate: the lesson's GPS switch, which is hooks.UseLocationWhen's
// first screen consumer.
//
// # Why a test and not just a demo
//
// The hook it exercises is the one a session added with nothing but its own
// unit tests behind it, which is the shape the live-map readout above exists to
// avoid: a hook whose documented use appears only inside strings. This drives
// the rendered tree, so the switch can only read the way it reads if the flag
// actually reached the hook.
//
// The default is true, deliberately — flipping it is what a reader does, and
// the lesson's behaviour on mount has to be what it was before the switch
// existed, or the emulator run that verified this panel would be verifying
// something else.
func TestTheLiveMapLessonGatesItsSensorWithAFlag(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Live maps: markers and the echo guard")
	cur := tree(t, mgr)

	toggle := findNode(cur, func(n *node) bool { return n.Type == "Switch" })
	if toggle == nil {
		t.Fatal("the GPS switch is missing, so the prose beside it describes a control " +
			"the panel does not have")
	}
	if on, _ := toggle.Props["checked"].(bool); !on {
		t.Fatal("the GPS switch starts off — mounting the lesson must behave exactly as " +
			"it did when the hook was called unconditionally")
	}
	if hasTextContaining(cur, "The GPS is off") {
		t.Error("the readout says the GPS is off while the switch says it is on")
	}

	// By position, which is the only address a Switch has here — the same walk
	// the checkbox demos use. It is the lesson's only one.
	toggleBool(t, mgr, "Switch", 0, false)
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "The GPS is off") {
		t.Error("flicking the switch off never reached the readout, so the flag is not " +
			"the thing the hook is reading")
	}
	// The arm has to outrank a fix, because core keeps the last one: a
	// coordinate printed beside a switch that says the GPS is off is the panel
	// contradicting itself.
	if hasTextContaining(cur, "accurate to about") {
		t.Error("a coordinate is printed while the GPS is switched off")
	}

	assertNoConcerns(t)
}

// --- 4.15 Small controls ------------------------------------------------------

// tapLabelled dispatches the click of the node whose accessibility label is
// name: the glyph buttons and bar cells carry their names as labels rather
// than as Button props.
func tapLabelled(t *testing.T, mgr *render.Manager, name string) {
	t.Helper()
	n := findNode(tree(t, mgr), func(n *node) bool {
		_, clickable := n.Props["onClick"].(string)
		return clickable && n.Style != nil && n.Style.AccessibilityLabel == name
	})
	if n == nil {
		t.Fatalf("no tappable node labelled %q", name)
	}
	mgr.DispatchCallback(n.Props["onClick"].(string))
}

func TestSmallControlsDemoDrivesAllFourWidgets(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Small controls: Stepper, Rating, Spinner & BottomBar")

	// Stepper: the + clamps at Max 8 and stops reporting.
	for range 10 {
		tapLabelled(t, mgr, "Increase")
	}
	if !hasText(tree(t, mgr), "Booking for 8") {
		t.Fatal("ten taps of + should clamp the guests at 8")
	}

	// Rating: tap the fourth star.
	if !hasText(tree(t, mgr), "Not rated yet") {
		t.Fatal("the rating starts empty")
	}
	tapLabelled(t, mgr, "4 of 5")
	if !hasText(tree(t, mgr), "You rated it 4 of 5") {
		t.Fatal("tapping the fourth star should rate 4")
	}

	// Spinner: rendered hidden until loading starts, then shown, then hidden.
	status := func() *node {
		return findNode(tree(t, mgr), func(n *node) bool {
			return n.Style != nil && n.Style.AccessibilityLabel == "Loading the demo"
		})
	}
	if s := status(); s == nil || s.Style.Display != core.DisplayNone {
		t.Fatal("the spinner is in the tree from the start, hidden")
	}
	tap(t, mgr, "Simulate loading")
	if s := status(); s.Style.Display == core.DisplayNone {
		t.Fatal("loading should show the spinner")
	}
	tap(t, mgr, "Stop loading")
	if s := status(); s.Style.Display != core.DisplayNone {
		t.Fatal("stopping should hide the spinner again")
	}

	// BottomBar: the current cell's name carries the selection.
	tapLabelled(t, mgr, "Search")
	cur := tree(t, mgr)
	if !hasText(cur, "Showing: Search") {
		t.Fatal("tapping Search should switch the tab")
	}
	if findNode(cur, func(n *node) bool {
		return n.Style != nil && n.Style.AccessibilityLabel == "Search, selected"
	}) == nil {
		t.Fatal("the current cell should announce itself as selected")
	}
	assertNoConcerns(t)
}

// --- 4.16 Choices and progress ------------------------------------------------

func TestChoicesLessonRadioGroupPicksByRow(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Radio groups, steps & timelines")

	// The radio group is the flow's Shipping step.
	tap(t, mgr, "Next")
	if !hasText(tree(t, mgr), "On Shipping · shipping: Standard") {
		t.Fatal("the group starts on Standard")
	}
	tapLabelled(t, mgr, "Express")
	if !hasText(tree(t, mgr), "On Shipping · shipping: Express") {
		t.Fatal("tapping the Express row should choose it")
	}
	tapLabelled(t, mgr, "Pick up in store")
	if !hasText(tree(t, mgr), "On Shipping · shipping: Express") {
		t.Fatal("a disabled option must not be chosen")
	}

	express := findNode(tree(t, mgr), func(n *node) bool {
		return n.Style != nil && n.Style.AccessibilityLabel == "Express"
	})
	if express.Style.AccessibilityRole != string(core.RoleRadio) {
		t.Fatalf("row role = %q, want radio", express.Style.AccessibilityRole)
	}
	assertNoConcerns(t)
}

func TestChoicesLessonStepIndicatorGoesBackOnlyToDoneSteps(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Radio groups, steps & timelines")

	strip := func() *node {
		return findNode(tree(t, mgr), func(n *node) bool {
			return n.Type == "Scroll" && n.Style != nil && n.Style.AccessibilityRole == string(core.RoleNavigation)
		})
	}
	if s := strip(); s == nil || s.Style.AccessibilityLabel != "Checkout, step 1 of 4: Account" {
		t.Fatal("the strip starts on Account and is named by its position")
	}

	tap(t, mgr, "Next")
	tap(t, mgr, "Next")
	if strip().Style.AccessibilityLabel != "Checkout, step 3 of 4: Payment" {
		t.Fatal("two Nexts should reach Payment")
	}

	// An upcoming step has no handler; a done one takes the flow back.
	if findNode(tree(t, mgr), func(n *node) bool {
		_, clickable := n.Props["onClick"].(string)
		return clickable && n.Style != nil && n.Style.AccessibilityLabel == "Step 4: Review"
	}) != nil {
		t.Fatal("an upcoming step must not be tappable")
	}
	tapLabelled(t, mgr, "Step 1: Account, done")
	if strip().Style.AccessibilityLabel != "Checkout, step 1 of 4: Account" {
		t.Fatal("tapping a done step should go back to it")
	}
	assertNoConcerns(t)
}

func TestChoicesLessonTimelineGrowsWithTheOrder(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Radio groups, steps & timelines")

	events := func() int {
		list := findNode(tree(t, mgr), func(n *node) bool {
			return n.Style != nil && n.Style.AccessibilityRole == string(core.RoleList) &&
				n.Style.AccessibilityLabel == "Order history"
		})
		if list == nil {
			t.Fatal("no Order history list")
		}
		return len(list.Children)
	}

	if events() != 2 {
		t.Fatalf("the order starts with 2 events, got %d", events())
	}
	tap(t, mgr, "Advance the order")
	tap(t, mgr, "Advance the order")
	if events() != 4 || !hasText(tree(t, mgr), "Delivered") {
		t.Fatal("two advances should reach Delivered")
	}
	// Disabled at the end: the handler stays registered but the flow stops.
	tap(t, mgr, "Advance the order")
	if events() != 4 {
		t.Fatal("the order cannot advance past Delivered")
	}
	assertNoConcerns(t)
}

// --- 4.17 Menus ---------------------------------------------------------------

// openSheets returns every visible Modal. Each menu in 4.17 carries its own,
// closed ones included, so "the" sheet is the one that is showing.
func openSheets(root *node) []*node {
	return findNodes(root, func(n *node) bool {
		return n.Type == "Modal" && n.Props["visible"] == true
	})
}

// tapInSheet taps the button with this label inside the one open sheet. The
// labels repeat across the rows' closed menus, so a search of the whole tree
// could land on a button nobody can see.
func tapInSheet(t *testing.T, mgr *render.Manager, label string) {
	t.Helper()
	open := openSheets(tree(t, mgr))
	if len(open) != 1 {
		t.Fatalf("%d sheets open, want exactly 1", len(open))
	}
	n := findNode(open[0], func(n *node) bool {
		return n.Type == "Button" && n.Props["label"] == label
	})
	if n == nil {
		t.Fatalf("no Button labeled %q in the open sheet", label)
	}
	mgr.DispatchCallback(n.Props["onClick"].(string))
}

// menuNoteTitles returns the note titles in row order, skipping the Modals:
// each row's sheet repeats the note's title as its heading.
func menuNoteTitles(root *node) []string {
	var out []string
	var walk func(*node)
	walk = func(n *node) {
		if n.Type == "Modal" {
			return
		}
		if n.Type == "Text" {
			switch n.Props["content"] {
			case "Groceries", "Trip ideas", "Books to read":
				out = append(out, n.Props["content"].(string))
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	return out
}

func TestMenusLessonRowMenusPinDeleteAndRestore(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Menus")

	want := func(why string, titles ...string) {
		t.Helper()
		if got := menuNoteTitles(tree(t, mgr)); !slices.Equal(got, titles) {
			t.Fatalf("%s: rows = %v, want %v", why, got, titles)
		}
	}
	if n := len(openSheets(tree(t, mgr))); n != 0 {
		t.Fatalf("%d sheets open at start, want none", n)
	}
	want("newest first", "Trip ideas", "Groceries", "Books to read")

	tapLabelled(t, mgr, "Actions for Books to read")
	tapInSheet(t, mgr, "Pin to top")
	if n := len(openSheets(tree(t, mgr))); n != 0 {
		t.Fatal("picking an item must close the menu")
	}
	want("a pinned note leads", "Books to read", "Trip ideas", "Groceries")

	// Cancel closes without acting.
	tapLabelled(t, mgr, "Actions for Trip ideas")
	tapInSheet(t, mgr, "Cancel")
	if n := len(openSheets(tree(t, mgr))); n != 0 {
		t.Fatal("Cancel must close the menu")
	}

	// Deleting a row changes the row count under one open-state; the next
	// row's menu still opens and is still the only one showing.
	tapLabelled(t, mgr, "Actions for Trip ideas")
	tapInSheet(t, mgr, "Delete")
	want("Trip ideas deleted", "Books to read", "Groceries")
	if !hasText(tree(t, mgr), "✓ deleted Trip ideas") {
		t.Fatal("Delete should report itself")
	}
	tapLabelled(t, mgr, "Actions for Groceries")
	if n := len(openSheets(tree(t, mgr))); n != 1 {
		t.Fatalf("%d sheets open after a delete, want the one just opened", n)
	}
	tapInSheet(t, mgr, "Cancel")

	tap(t, mgr, "Restore deleted notes")
	want("restored", "Books to read", "Trip ideas", "Groceries")
	assertNoConcerns(t)
}

func TestMenusLessonSortPickerChecksTheCurrentOrder(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Menus")

	tapLabelled(t, mgr, "Sort: Newest")
	sheet := openSheets(tree(t, mgr))
	if len(sheet) != 1 {
		t.Fatalf("%d sheets open, want the sort menu", len(sheet))
	}
	if findNode(sheet[0], func(n *node) bool {
		return n.Type == "Button" && n.Props["label"] == "✓ Newest" &&
			n.Style != nil && n.Style.AccessibilityLabel == "Newest, selected"
	}) == nil {
		t.Fatal("the current order is checked and named as selected")
	}

	tapInSheet(t, mgr, "Title")
	if got := menuNoteTitles(tree(t, mgr)); !slices.Equal(got, []string{"Books to read", "Groceries", "Trip ideas"}) {
		t.Fatalf("sorted by title: rows = %v", got)
	}
	if !hasText(tree(t, mgr), "✓ sorted by title") {
		t.Fatal("the pick should report itself")
	}

	tapLabelled(t, mgr, "Sort: Title")
	tapInSheet(t, mgr, "✓ Title")
	if n := len(openSheets(tree(t, mgr))); n != 0 {
		t.Fatal("picking the checked item still closes the menu")
	}
	assertNoConcerns(t)
}
