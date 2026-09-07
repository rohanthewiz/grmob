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
