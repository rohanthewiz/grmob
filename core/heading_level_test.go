package core

import "testing"

// AccessibilityHeadingLevel merges on the same "a zero value means unset" rule
// every other Style field follows, and independently of the role.
//
// The independence is the part worth pinning. A theme's Style and a widget's
// own Style are merged in sequence, so the tier and the role can arrive from
// different sources; if the level were only carried when a role came with it,
// a theme that set one would lose it to the widget Style that named the
// heading.
func TestHeadingLevelMerges(t *testing.T) {
	tier := Style{AccessibilityHeadingLevel: 2}
	roled := Style{AccessibilityRole: RoleHeading}

	merged := tier.With(roled)
	if merged.AccessibilityHeadingLevel != 2 || merged.AccessibilityRole != RoleHeading {
		t.Errorf("level and role should survive being merged from two Styles, got %+v",
			struct {
				Role  Role
				Level int
			}{merged.AccessibilityRole, merged.AccessibilityHeadingLevel})
	}

	// Zero is unset, not "level zero": a Style that says nothing about the
	// tier must not erase one the target already has. This is the rule that
	// makes UseStyle safe to apply on top of an already-built Style.
	kept := merged.With(Style{AccessibilityLabel: "March"})
	if kept.AccessibilityHeadingLevel != 2 {
		t.Errorf("a Style with no level cleared one that was set, got %d",
			kept.AccessibilityHeadingLevel)
	}

	// A later Style that does state a tier wins, like every other field.
	if got := merged.With(Style{AccessibilityHeadingLevel: 3}); got.AccessibilityHeadingLevel != 3 {
		t.Errorf("an explicit level should override, got %d", got.AccessibilityHeadingLevel)
	}
}

// The prop constructor writes the field and nothing else — in particular it
// does not imply RoleHeading. The two are separate props because a heading
// with no stated tier is a legitimate thing (every heading written before the
// field existed is one), and because inferring the role here would mean
// AccessibilityHeadingLevel silently restyling a node's semantics.
func TestHeadingLevelPropSetsOnlyTheLevel(t *testing.T) {
	var s Style
	AccessibilityHeadingLevel(4).Apply(&s)
	if s.AccessibilityHeadingLevel != 4 {
		t.Errorf("level = %d, want 4", s.AccessibilityHeadingLevel)
	}
	if s.AccessibilityRole != RoleNone {
		t.Errorf("the level prop invented a role %q; the caller states that separately",
			s.AccessibilityRole)
	}
}
