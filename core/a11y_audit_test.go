package core

import (
	"strings"
	"testing"
)

// The accessibility audit. Every finding here is a failure with no symptom on
// any of the four targets, which is the whole reason the pass exists — the
// exporters cannot catch these because none of them is a fact about one
// element.

func auditWith(t *testing.T, root *Node) []Concern {
	t.Helper()
	SetDebugMode(true)
	ClearConcerns()
	t.Cleanup(func() {
		SetDebugMode(false)
		ClearConcerns()
	})
	AuditTree(root)
	return Concerns()
}

func kinds(found []Concern) map[string]int {
	out := map[string]int{}
	for _, c := range found {
		out[c.Kind] += c.Count
	}
	return out
}

func requireKind(t *testing.T, found []Concern, kind string, substrings ...string) {
	t.Helper()
	for _, c := range found {
		if c.Kind != kind {
			continue
		}
		ok := true
		for _, want := range substrings {
			if !strings.Contains(c.Detail, want) {
				ok = false
			}
		}
		if ok {
			return
		}
	}
	t.Errorf("no %s concern mentioning %v; got %+v", kind, substrings, found)
}

func requireNoKind(t *testing.T, found []Concern, kind string) {
	t.Helper()
	if n := kinds(found)[kind]; n != 0 {
		t.Errorf("unexpected %s concern: %+v", kind, found)
	}
}

// Two elements, one id. Both are written verbatim — nothing rewrites the string
// — so the document is invalid and every reference to that id resolves to
// whichever the browser parsed first, which is a tab strip switching the wrong
// region with nothing anywhere reporting it.
func TestADuplicateAccessibilityIDIsReported(t *testing.T) {
	found := auditWith(t, &Node{Type: "Column", Children: []*Node{
		{Type: "Box", Style: &Style{AccessibilityID: "panel"}},
		{Type: "Box", Children: []*Node{
			{Type: "Box", Style: &Style{AccessibilityID: "panel"}},
		}},
	}})

	// Both locations named. A duplicate is exactly the bug where knowing one
	// of the two is no help.
	requireKind(t, found, ConcernDuplicateAccessibilityID, `"panel"`, "root/0", "root/1/0")
}

func TestDistinctIDsAreNotReported(t *testing.T) {
	found := auditWith(t, &Node{Type: "Column", Children: []*Node{
		{Type: "Box", Style: &Style{AccessibilityID: "a"}},
		{Type: "Box", Style: &Style{AccessibilityID: "b"}},
	}})
	requireNoKind(t, found, ConcernDuplicateAccessibilityID)
}

// A reference to an id nothing claims. Both exporters write it anyway and say
// so in their own comments — an export is one document with no index, a patch
// is one element with none at all — which is correct from where they sit and
// leaves the failure with nowhere to be caught but here.
func TestADanglingControlsReferenceIsReported(t *testing.T) {
	found := auditWith(t, &Node{Type: "Row", Children: []*Node{
		{Type: "Box", Style: &Style{
			AccessibilityRole:     RoleTab,
			AccessibilityControls: "app-panel",
		}},
		{Type: "Box", Style: &Style{AccessibilityID: "other-panel"}},
	}})

	requireKind(t, found, ConcernDanglingReference, `"app-panel"`, "root/0")
}

// The resolution happens after the whole walk, not during it, because the
// ordinary shape points forwards *and* backwards: a bottom tab bar sits under
// the region it switches, and a top one sits over it.
func TestAReferenceResolvesInEitherDirection(t *testing.T) {
	for _, tc := range []struct {
		name  string
		nodes []*Node
	}{
		{"the tab comes first", []*Node{
			{Type: "Box", Style: &Style{AccessibilityControls: "p"}},
			{Type: "Box", Style: &Style{AccessibilityID: "p"}},
		}},
		{"the panel comes first", []*Node{
			{Type: "Box", Style: &Style{AccessibilityID: "p"}},
			{Type: "Box", Style: &Style{AccessibilityControls: "p"}},
		}},
	} {
		found := auditWith(t, &Node{Type: "Column", Children: tc.nodes})
		if n := kinds(found)[ConcernDanglingReference]; n != 0 {
			t.Errorf("%s: %+v", tc.name, found)
		}
	}
}

// An id is a single HTML token. "app panel" is written verbatim into an invalid
// document, and the element then sits there looking correct while nothing —
// not an aria-controls, not getElementById, not a `#id` selector — can find it.
func TestAnIDWithWhitespaceIsReported(t *testing.T) {
	for _, id := range []string{"app panel", "app\tpanel", "app\npanel", " panel", "panel "} {
		found := auditWith(t, &Node{Type: "Box", Style: &Style{AccessibilityID: id}})
		requireKind(t, found, ConcernInvalidAccessibilityID, "whitespace")
	}

	// The wider Unicode spaces are not ASCII whitespace and are legal in an
	// id, so reaching for unicode.IsSpace here would report a working document
	// as broken. Spelled as an escape rather than typed, because a
	// non-breaking space in a source literal is invisible next to the ASCII
	// one three lines up.
	found := auditWith(t, &Node{Type: "Box", Style: &Style{AccessibilityID: "app\u00a0panel"}})
	requireNoKind(t, found, ConcernInvalidAccessibilityID)
}

// The reserved prefix, which the field's doc has always stated and nothing
// enforced. It is the harder collision of the two to find: it breaks a
// core.TabView wiring the app never wrote, so nothing in their own code
// mentions the id that stopped working.
func TestAReservedIDPrefixIsReported(t *testing.T) {
	found := auditWith(t, &Node{Type: "Box", Style: &Style{
		AccessibilityID: "grmob-root-panel-0",
	}})
	requireKind(t, found, ConcernInvalidAccessibilityID, "reserved", "grmob-")

	ok := auditWith(t, &Node{Type: "Box", Style: &Style{AccessibilityID: "my-grmob-panel"}})
	requireNoKind(t, ok, ConcernInvalidAccessibilityID)
}

// A disclosure Compose cannot act on. The web writes aria-expanded regardless,
// so this is the one finding here that is a *divergence* rather than a
// universal silence — and it is a deliberate one, since an expand action
// nothing performs is worse than none. What was missing is anything saying so
// at the call site.
func TestAnInertDisclosureIsReported(t *testing.T) {
	found := auditWith(t, &Node{Type: "Box", Style: &Style{
		AccessibilityRole:     RoleButton,
		AccessibilityExpanded: ExpandedOpen,
	}})
	requireKind(t, found, ConcernInertDisclosure, "root", "OnClick")
}

// The guard is "neither handler", because that is what Renderer.kt's
// gestureModifier tests: it returns early when a node carries neither, so a
// long-press alone still reaches the disclosure wiring. Checking only for a
// click would report a node that works.
func TestADisclosureWithAHandlerIsNotReported(t *testing.T) {
	for _, prop := range []string{"onClick", "onLongPress"} {
		found := auditWith(t, &Node{
			Type:  "Box",
			Props: map[string]any{prop: "cb_1"},
			Style: &Style{AccessibilityExpanded: ExpandedClosed},
		})
		if n := kinds(found)[ConcernInertDisclosure]; n != 0 {
			t.Errorf("%s: %+v", prop, found)
		}
	}
}

// A keyboard contract on a widget with no keyboard.
//
// This is the only finding here whose subject is a *data* attribute rather
// than an ARIA one, and the reason is that ARIA has no attribute for it:
// aria-* says what a widget is, and selection-follows-focus says what its
// keyboard does. So the WASM runtime writes it into its own namespace, on any
// node that asks — deliberately, since consulting the composite tables at the
// point attributes are written would put those tables in two places — and
// nothing anywhere reads it back off a role that has no arrows.
//
// A `list` is the near miss and is why this is worth reporting at all: it is a
// collection, it looks like the kind of thing arrow keys cross, and ARIA gives
// it no keyboard because it is content rather than a control. The author who
// set the flag on one wanted RoleListBox.
func TestAFollowsFocusFlagOnARoleWithNoKeyboardIsReported(t *testing.T) {
	for _, role := range []Role{RoleList, RoleGroup, RoleOption, RoleTab} {
		found := auditWith(t, &Node{Type: "Box", Style: &Style{
			AccessibilityRole:                  role,
			AccessibilitySelectionFollowsFocus: true,
		}})
		requireKind(t, found, ConcernInertFollowsFocus, "root", string(role))
	}

	// No role at all is the commonest way to get here, and the one worth
	// reporting most: the strip looks right, announces as a plain box, and has
	// no arrow keys — the flag is the only evidence anyone meant a composite.
	found := auditWith(t, &Node{Type: "Row", Style: &Style{
		AccessibilitySelectionFollowsFocus: true,
	}})
	requireKind(t, found, ConcernInertFollowsFocus, "carries no AccessibilityRole")
}

// And every role that does have a keyboard is silent — derived from
// core.KeyboardComposites() rather than listed, so a fourth pattern does not
// arrive already being reported as inert.
//
// The member roles are checked in the other direction above: the flag is a
// statement about a container's contract with the keyboard, and a per-member
// spelling would let a strip disagree with itself (see the field's own doc).
func TestAFollowsFocusFlagOnACompositeIsSilent(t *testing.T) {
	for _, role := range KeyboardComposites() {
		found := auditWith(t, &Node{Type: "Row", Style: &Style{
			AccessibilityRole:                  role,
			AccessibilitySelectionFollowsFocus: true,
		}})
		requireNoKind(t, found, ConcernInertFollowsFocus)
	}
}

// One keyboard pattern inside another.
//
// The divergence is real and deliberate, and until now it was stated in three
// comments and a documentation section — all of them in the WASM target, where
// an author writing Go does not read. The runtime's own member walks stop at a
// nested composite, so both containers keep a roving tabindex and the pair is
// two tab stops; ARIA describes one, by making the inner widget's current
// member the outer widget's member, which would need two widgets writing
// tabindex onto one element and an owner rule for when they disagree.
//
// Every ordered pair is checked rather than the one shape that prompted it (a
// tablist in a toolbar), because the two member walks reach the same outcome
// by different routes: a toolbar's walk stops at any composite because nothing
// names its members, and a listbox's stops at a nested listbox because the two
// would pool their options. The finding is about the outcome.
func TestOneCompositeInsideAnotherIsReported(t *testing.T) {
	for _, outer := range KeyboardComposites() {
		for _, inner := range KeyboardComposites() {
			found := auditWith(t, &Node{
				Type:  "Row",
				Style: &Style{AccessibilityRole: outer},
				Children: []*Node{
					{Type: "Box", Style: &Style{AccessibilityRole: inner}},
				},
			})
			requireKind(t, found, ConcernNestedComposite,
				string(inner), string(outer), "two tab stops")
		}
	}
}

// The nesting is found however deep it is buried, which is the half a
// direct-child check would miss: a strip inside a scroller inside a Box is the
// same two tab stops, and the runtime's walks are subtree walks for exactly
// that reason.
func TestANestedCompositeIsFoundThroughOrdinaryContainers(t *testing.T) {
	// The intermediates carry Styles of their own, which is the ordinary case
	// and the one that exercises the line that matters: the walk only reaches
	// this check for a node with a Style at all, so a version that forgot the
	// ancestor at every non-composite would still find a strip buried under
	// bare Boxes and would lose one under a padded scroller. Nearly every real
	// container has a Style.
	found := auditWith(t, &Node{
		Type:  "Row",
		Style: &Style{AccessibilityRole: RoleToolbar},
		Children: []*Node{
			{Type: "Box", Style: &Style{Padding: EdgeInsets{Left: 8}}, Children: []*Node{
				{Type: "Scroll", Style: &Style{Overflow: "auto"}, Children: []*Node{
					{Type: "Row", Style: &Style{AccessibilityRole: RoleTabList}},
				}},
			}},
		},
	})
	requireKind(t, found, ConcernNestedComposite, "root/0/0/0", "tablist", "toolbar")
}

// Two composites that are siblings say nothing, which is the ordinary shape
// and the one worth asserting: a tab strip over the region it switches is a
// tablist beside a tabpanel, and that panel holds whatever a screen holds —
// including another widget with a keyboard. The two are linked by
// aria-controls rather than by containment, so a walk that reported on
// *proximity* rather than on nesting would fire on every tabbed screen and be
// worth nothing.
func TestCompositesThatAreNotNestedAreSilent(t *testing.T) {
	found := auditWith(t, &Node{Type: "Column", Children: []*Node{
		{Type: "Row", Style: &Style{AccessibilityRole: RoleTabList}},
		{Type: "Box", Style: &Style{AccessibilityRole: RoleTabPanel}, Children: []*Node{
			{Type: "Column", Style: &Style{AccessibilityRole: RoleListBox}},
		}},
	}})
	requireNoKind(t, found, ConcernNestedComposite)
}

// A number that is not a number, on the one role that carries a range.
//
// This is the finding whose subject is not a relationship between two elements
// but a disagreement between four platforms about one string: a browser reads
// aria-valuenow="45%" as 0 and pins the bar at the start, Compose reads it as
// absent and announces an indeterminate bar, SwiftUI has no numeric value
// property at all, and the fill on screen goes on drawing the caller's own
// float — so the bar looks right everywhere and announces three different
// wrong things.
func TestAValueRangeThatIsNotNumbersIsReported(t *testing.T) {
	found := auditWith(t, &Node{Type: "Row", Style: &Style{
		AccessibilityRole:  RoleProgressBar,
		AccessibilityLabel: "Upload",
		// The formatting bug this actually is in the wild: a percent sign on
		// a value ARIA wants bare.
		AccessibilityValue: ValueRange{Now: "45%", Min: "0", Max: "100"},
	}})
	requireKind(t, found, ConcernUnusableValueRange, "field Now is", `Now = "45%"`)
}

// Both bounds at once, so the report is a sentence rather than three of them.
func TestSeveralUnparseableFieldsAreOneFinding(t *testing.T) {
	found := auditWith(t, &Node{Type: "Row", Style: &Style{
		AccessibilityRole:  RoleProgressBar,
		AccessibilityValue: ValueRange{Now: "45", Min: "none", Max: "lots"},
	}})
	requireKind(t, found, ConcernUnusableValueRange,
		"fields Min and Max are", `Min = "none"`, `Max = "lots"`)
	if n := kinds(found)[ConcernUnusableValueRange]; n != 1 {
		t.Errorf("one node produced %d findings, want 1 — an author fixes the range, "+
			"not the fields one at a time", n)
	}
}

// A range with no position in it.
//
// Separated from the parse failures because it is not a string problem: every
// one of these three parses, and "5" is a perfectly good number to be nowhere
// between 9 and 1. It is the arm Compose cannot express at all —
// ProgressBarRangeInfo throws on an empty range — so Renderer.kt drops the
// property, and the web keeps the numbers and clamps.
func TestAnEmptyValueRangeIsReported(t *testing.T) {
	for _, c := range []struct {
		name  string
		value ValueRange
	}{
		{"inverted", ValueRange{Now: "5", Min: "9", Max: "1"}},
		{"a single point", ValueRange{Now: "5", Min: "5", Max: "5"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			found := auditWith(t, &Node{Type: "Row", Style: &Style{
				AccessibilityRole:  RoleProgressBar,
				AccessibilityValue: c.value,
			}})
			requireKind(t, found, ConcernUnusableValueRange, "range is empty")
		})
	}
}

// The ranges that are fine, and the two that look like mistakes and are not.
//
// The second pair is the whole reason this check asks core.ValueRange rather
// than testing strings: an unstated position *is* ARIA's indeterminate bar, and
// an unstated bound *is* ARIA's 0..100 — both are the vocabulary working, and a
// check that reported them would fire on components.ProgressBar's own output
// and on every indeterminate spinner in every app.
func TestAUsableValueRangeIsSilent(t *testing.T) {
	for _, c := range []struct {
		name  string
		value ValueRange
	}{
		{"a bare percentage", ValueOf(45, 0, 100)},
		{"a bare position under ARIA's defaults", ValueRange{Now: "45"}},
		{"a step out of five", ValueOf(3, 1, 5)},
		{"a position outside its range, which every target clamps", ValueOf(150, 0, 100)},
		{"a fractional position", ValueRange{Now: "45.5", Min: "0", Max: "100"}},
		{"a negative range", ValueOf(-5, -10, 0)},
		{"ARIA's indeterminate bar", ValueRange{Min: "0", Max: "100"}},
		{"words with no numbers, which are not a range at all",
			ValueRange{Text: "almost done"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			found := auditWith(t, &Node{Type: "Row", Style: &Style{
				AccessibilityRole:  RoleProgressBar,
				AccessibilityValue: c.value,
			}})
			requireNoKind(t, found, ConcernUnusableValueRange)
		})
	}
}

// The role is not part of the question, deliberately.
//
// A range on a node with no role is dropped by both web exporters by design and
// announced by neither phone as a number — that guard is tested at each writer,
// and the audit's header says why such cases are not findings. What is *not*
// covered anywhere is the same broken string on a node whose role will be added
// next week, so the check reads the value rather than the pair. It is the
// author's own numbers either way, and they are unusable either way.
func TestAnUnusableRangeIsReportedWhateverTheRole(t *testing.T) {
	found := auditWith(t, &Node{Type: "Box", Style: &Style{
		AccessibilityValue: ValueRange{Now: "half"},
	}})
	requireKind(t, found, ConcernUnusableValueRange, "field Now is")
}

// The zero value says nothing and is what every node in every tree carries, so
// it cannot be a finding.
func TestAnOrdinaryTreeIsSilent(t *testing.T) {
	found := auditWith(t, &Node{Type: "Column", Children: []*Node{
		{Type: "Text", Props: map[string]any{"content": "Sermons"}},
		{Type: "Box", Style: &Style{AccessibilityRole: RoleHeading, AccessibilityHeadingLevel: 1}},
		{Type: "Button", Props: map[string]any{"onClick": "cb_0"}},
	}})
	if len(found) != 0 {
		t.Errorf("an ordinary tree reported %+v", found)
	}
}

// Off is off — as far as anything can observe.
//
// Worth being exact about what this proves, because it is less than it looks.
// upsertConcern carries its own IsDebugMode guard as a backstop, so deleting
// the one in AuditTree leaves the *behaviour* identical and this test passes
// either way. What that guard buys is cost: without it, every production app
// pays a full tree walk per frame to build detail strings that are then thrown
// away, which is the same bargain the cursor audit and the duplicate-key check
// strike and the reason all three are written the same way.
//
// No behavioural test can distinguish the two, and one that claimed to would be
// asserting something it had not measured. This asserts the part that is
// observable and says so.
//
// The part that is not observable is now measured, next door:
// TestTheAccessibilityAuditCostsNothingWithDebugModeOff in debug_cost_test.go
// counts the allocations the guard saves, which is the only form the claim can
// take and the only one that can fail.
func TestTheAuditIsSilentWithDebugModeOff(t *testing.T) {
	SetDebugMode(false)
	ClearConcerns()
	AuditTree(&Node{Type: "Column", Children: []*Node{
		{Type: "Box", Style: &Style{AccessibilityID: "dup"}},
		{Type: "Box", Style: &Style{AccessibilityID: "dup"}},
	}})
	if got := Concerns(); len(got) != 0 {
		t.Errorf("the audit ran with debug mode off: %+v", got)
	}
}

// A nil tree is the state a host is in before its first render, not an error.
func TestAuditingNothingIsNotAFinding(t *testing.T) {
	found := auditWith(t, nil)
	if len(found) != 0 {
		t.Errorf("auditing nil reported %+v", found)
	}
}

// The paths are sorted before the detail string is built, and the reason is the
// deduplication: the collector keys on (kind, detail), so an unsorted list from
// a map iteration would make the same duplicate id read as a new finding every
// few passes — and the count that is meant to say "wrong for 200 frames" would
// sit at 1.
func TestARepeatedFindingCountsRatherThanMultiplying(t *testing.T) {
	tree := &Node{Type: "Column", Children: []*Node{
		{Type: "Box", Style: &Style{AccessibilityID: "dup"}},
		{Type: "Box", Style: &Style{AccessibilityID: "dup"}},
		{Type: "Box", Style: &Style{AccessibilityID: "dup"}},
	}}
	SetDebugMode(true)
	ClearConcerns()
	defer func() {
		SetDebugMode(false)
		ClearConcerns()
	}()
	for i := 0; i < 25; i++ {
		AuditTree(tree)
	}
	found := Concerns()
	if len(found) != 1 {
		t.Fatalf("25 identical passes produced %d findings, want 1: %+v", len(found), found)
	}
	if found[0].Count != 25 {
		t.Errorf("count = %d, want 25", found[0].Count)
	}
}
