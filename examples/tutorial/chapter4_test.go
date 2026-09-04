package tutorial

import (
	"testing"

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
	if hasTextContaining(cur, "components.VariantWarning") ||
		hasTextContaining(cur, "components.EmphasisGhost") {
		t.Fatal("the printed literal should omit zero-value axes")
	}

	// Moving the knobs rewrites the printed literal.
	tap(t, mgr, "Warning")
	tap(t, mgr, "Ghost")
	cur = tree(t, mgr)
	if !hasTextContaining(cur, "components.VariantWarning") {
		t.Fatal("selecting the Warning variant should print Variant in the literal")
	}
	if !hasTextContaining(cur, "components.EmphasisGhost") {
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

// --- 4.6 Collections --------------------------------------------------------

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
