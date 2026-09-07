package core

import (
	"sort"
	"testing"
)

// The placement audit. A core.StackAlign outside a stack is inert on all four
// targets and says so nowhere, which is the whole reason this arm exists — the
// prop compiles, merges, crosses the bridge and is read by nobody.
//
// The helpers (auditWith, requireKind, requireNoKind) are the accessibility
// audit's, because it is one walk and one collector.

// The plain case: a placement on a Column's child. Nothing places it, and no
// target reports it.
func TestAPlacementOutsideAStackIsReported(t *testing.T) {
	found := auditWith(t, &Node{Type: "Column", Style: &Style{}, Children: []*Node{
		{Type: "Text", Style: &Style{StackAlign: StackAlignTopEnd}},
	}})
	requireKind(t, found, ConcernInertPlacement, "root/0", "top-end", "a Column")
}

// The case the arm must not fire on, and the reason the check is worth having
// at all: a layer of a real stack.
func TestAPlacementInsideAStackIsNotReported(t *testing.T) {
	found := auditWith(t, &Node{Type: "ZStack", Style: &Style{}, Children: []*Node{
		{Type: "Box", Style: &Style{}},
		{Type: "Text", Style: &Style{StackAlign: StackAlignTop}},
	}})
	requireNoKind(t, found, ConcernInertPlacement)
}

// A Fragment is transparent to a placement, so a core.For inside a ZStack
// overlays what it generated rather than stacking it — htmlout forwards the
// imposed declaration through one for exactly this reason.
//
// This is the case a naive "look at the tree parent" check gets wrong, and it
// gets it wrong on the framework's own idiom for generating layers: every
// generated layer in every stack would be reported, which is the failure mode
// that makes an audit worth turning off.
func TestAPlacementUnderAGroupingNodeReadsThroughIt(t *testing.T) {
	inStack := auditWith(t, &Node{Type: "ZStack", Style: &Style{}, Children: []*Node{
		{Type: "Fragment", Children: []*Node{
			{Type: "Theme", Children: []*Node{
				{Type: "Text", Style: &Style{StackAlign: StackAlignBottom}},
			}},
		}},
	}})
	requireNoKind(t, inStack, ConcernInertPlacement)

	// And the same nesting under a container that does not place: transparency
	// forwards whatever is above it, including "nothing useful".
	inRow := auditWith(t, &Node{Type: "Row", Style: &Style{}, Children: []*Node{
		{Type: "Fragment", Children: []*Node{
			{Type: "Text", Style: &Style{StackAlign: StackAlignBottom}},
		}},
	}})
	requireKind(t, inRow, ConcernInertPlacement, "root/0/0", "a Row")
}

// A stack's grandchild is not a layer. This is the shape that reads correctly
// and is wrong — the node is inside a ZStack, so a reader skimming for "is
// there a stack above this" would pass it, and the Row between them is what
// actually decides the placement.
func TestAPlacementOnAStacksGrandchildIsReported(t *testing.T) {
	found := auditWith(t, &Node{Type: "ZStack", Style: &Style{}, Children: []*Node{
		{Type: "Row", Style: &Style{}, Children: []*Node{
			{Type: "Text", Style: &Style{StackAlign: StackAlignStart}},
		}},
	}})
	requireKind(t, found, ConcernInertPlacement, "root/0/0", "a Row")
}

// The root has no container above it, which is the same finding with a
// different sentence. Worth its own case because "" is the placer value the
// walk starts with, and a check written as `placer != "ZStack"` and a check
// written as `!placesChildren(placer)` differ nowhere else.
func TestAPlacementOnTheRootIsReported(t *testing.T) {
	found := auditWith(t, &Node{Type: "Box", Style: &Style{StackAlign: StackAlignTop}})
	requireKind(t, found, ConcernInertPlacement, "root", "the root of the tree")
}

// The centre is the zero value: every node in every tree carries it. An arm
// that reported it would report the whole tree, which is the same reason
// StackAlignments() excludes it and neither native has an arm for it.
func TestTheCentreIsNotAPlacementToReport(t *testing.T) {
	found := auditWith(t, &Node{Type: "Column", Style: &Style{}, Children: []*Node{
		{Type: "Text", Style: &Style{StackAlign: StackAlignCenter}},
		{Type: "Text", Style: &Style{}},
		{Type: "Text"}, // no Style at all, which is what a hand-built node has
	}})
	requireNoKind(t, found, ConcernInertPlacement)
}

// Every stated placement is reportable, not just the one a case above happens
// to use. The eight are a two-axis vocabulary and the check reads the field
// rather than switching on it — so this is cheap, and it is what would fail if
// the guard were ever narrowed to a subset (the "top-*" row, say, on the theory
// that those are the ones people write).
func TestEveryStatedPlacementIsReportedOutsideAStack(t *testing.T) {
	for _, align := range StackAlignments() {
		found := auditWith(t, &Node{Type: "Row", Style: &Style{}, Children: []*Node{
			{Type: "Text", Style: &Style{StackAlign: align}},
		}})
		requireKind(t, found, ConcernInertPlacement, string(align))
	}
}

// The two sets the walk turns on, pinned.
//
// Both are one or two entries today and both are consulted by an audit that
// silently passes when they are wrong: an emptied placingContainers reports
// every layer in every stack, and an emptied groupingContainers reports every
// generated one. Neither failure names the table it came from, so the table is
// asserted directly.
func TestThePlacementCensusNamesTheContainersItMeansTo(t *testing.T) {
	for _, c := range []struct {
		what string
		got  []string
		want []string
		why  string
	}{
		{
			"PlacingContainers", PlacingContainers(), []string{"ZStack"},
			"core.ZStack is the framework's only overlay — core.Box was one on the " +
				"natives and deliberately stopped being one. A second entry means a " +
				"second container reads StackAlign, which is a change to htmlout's " +
				"imposed channel and to both native stack renderers as well",
		},
		{
			"GroupingContainers", GroupingContainers(), []string{"Fragment", "Theme"},
			"these are the two node types with no box of their own (core.For's wrapper " +
				"and core.WithTheme's). A type added here becomes transparent to a " +
				"placement, and one removed makes every placement under it a finding",
		},
	} {
		got := append([]string(nil), c.got...)
		sort.Strings(got)
		if len(got) != len(c.want) {
			t.Errorf("%s() = %v, want %v — %s", c.what, got, c.want, c.why)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s() = %v, want %v — %s", c.what, got, c.want, c.why)
				break
			}
		}
	}
}

// The finding must name what to do about it, because "inert" is not actionable
// on its own: the author wrote a placement and meant something by it, and the
// two ways out are moving the node into a stack or saying the same thing with
// the container's own props.
//
// Asserted rather than left to the prose, since a message is the entire product
// of a debug-only check and nothing else in the repository reads this one.
func TestThePlacementFindingSaysWhatToDoInstead(t *testing.T) {
	found := auditWith(t, &Node{Type: "Column", Style: &Style{}, Children: []*Node{
		{Type: "Text", Style: &Style{StackAlign: StackAlignTopStart}},
	}})
	requireKind(t, found, ConcernInertPlacement,
		"move the node into a ZStack", "AlignItems", "JustifyContent")
}
