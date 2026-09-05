package core

import "testing"

// AccessibilityNestingLevel merges on the same "a zero value means unset" rule
// AccessibilityHeadingLevel does, and independently of the role for the same
// reason: a theme's Style and a widget's own Style are merged in sequence, so
// the depth and the role can arrive from different sources.
func TestNestingLevelMerges(t *testing.T) {
	depth := Style{AccessibilityNestingLevel: 2}
	roled := Style{AccessibilityRole: RoleListItem}

	merged := depth.With(roled)
	if merged.AccessibilityNestingLevel != 2 || merged.AccessibilityRole != RoleListItem {
		t.Errorf("depth and role should survive being merged from two Styles, got %+v",
			struct {
				Role  Role
				Level int
			}{merged.AccessibilityRole, merged.AccessibilityNestingLevel})
	}

	kept := merged.With(Style{AccessibilityLabel: "Compline"})
	if kept.AccessibilityNestingLevel != 2 {
		t.Errorf("a Style with no depth cleared one that was set, got %d",
			kept.AccessibilityNestingLevel)
	}

	if got := merged.With(Style{AccessibilityNestingLevel: 3}); got.AccessibilityNestingLevel != 3 {
		t.Errorf("an explicit depth should override, got %d", got.AccessibilityNestingLevel)
	}
}

// The two levels merge independently of each other as well as of the role.
//
// They can never both be *read* — a node has one role, and the exporters
// dispatch on it — but merging is not the layer that knows that. A merge that
// dropped one because the other was set would make the result depend on which
// Style in the chain happened to name the role, which is exactly the ordering
// dependence the independence rule exists to remove.
func TestTheTwoLevelsDoNotDisplaceEachOther(t *testing.T) {
	both := Style{AccessibilityHeadingLevel: 2}.With(Style{AccessibilityNestingLevel: 3})
	if both.AccessibilityHeadingLevel != 2 || both.AccessibilityNestingLevel != 3 {
		t.Errorf("one level displaced the other: heading = %d, nesting = %d; want 2 and 3",
			both.AccessibilityHeadingLevel, both.AccessibilityNestingLevel)
	}

	// And in the other order, since a merge is not symmetric.
	back := Style{AccessibilityNestingLevel: 3}.With(Style{AccessibilityHeadingLevel: 2})
	if back.AccessibilityHeadingLevel != 2 || back.AccessibilityNestingLevel != 3 {
		t.Errorf("merge order changed which levels survived: heading = %d, nesting = %d",
			back.AccessibilityHeadingLevel, back.AccessibilityNestingLevel)
	}
}

// The prop constructor writes its own field and nothing else — no role, and
// not the other level. Same contract AccessibilityHeadingLevel's has: the role
// is a separate statement, and inferring one here would mean a depth prop
// silently restyling a node's semantics.
func TestNestingLevelPropSetsOnlyTheLevel(t *testing.T) {
	var s Style
	AccessibilityNestingLevel(4).Apply(&s)
	if s.AccessibilityNestingLevel != 4 {
		t.Errorf("nesting level = %d, want 4", s.AccessibilityNestingLevel)
	}
	if s.AccessibilityRole != RoleNone {
		t.Errorf("the depth prop invented a role %q; the caller states that separately",
			s.AccessibilityRole)
	}
	if s.AccessibilityHeadingLevel != 0 {
		t.Errorf("the depth prop wrote the heading tier as well (%d) — they are two fields "+
			"with two ranges, and a caller that means one must not get the other",
			s.AccessibilityHeadingLevel)
	}
}
