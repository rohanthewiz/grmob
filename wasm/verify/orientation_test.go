package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

// aria-orientation, held across the two web targets.
//
// The attribute is the fix for a divergence that existed *within* one target:
// the runtime read a composite's flex axis to pick its arrow pair and nothing
// wrote the answer down, so a role="tablist" laid out as a Column took Up/Down
// while a reader in browse mode was told — by ARIA's horizontal default for a
// tablist — that it ran the other way. htmlout/orientation.go carries the whole
// argument.
//
// Two things have to hold and they fail differently:
//
//   - the table has to be the same table. A row that exists on one target and
//     not the other is a container that announces its axis in a static export
//     and not in the app, or the reverse.
//   - the runtime's keyboard has to read the attribute rather than the axis.
//     Re-deriving it is exactly the shape the attribute exists to remove: two
//     derivations of one fact, free to drift.
func TestRuntimeOrientationTableMatchesGo(t *testing.T) {
	src := runtimeSource(t)

	// A const object literal rather than a lookup function, so it is read
	// directly instead of through parseRuntimeTable — which requires the
	// `[param] || fallback` tail that proves it found the right braces. The
	// equivalent proof here is the assignment being matched whole, name
	// included. Same shape COMPOSITE_MEMBERS is read in.
	decl := regexp.MustCompile(`const ARIA_ORIENTATIONS = \{([^}]*)\};`)
	m := decl.FindStringSubmatch(src)
	if m == nil {
		t.Fatal("grmob-runtime.js: no ARIA_ORIENTATIONS object literal — if it was " +
			"renamed or given a computed shape, update this test rather than deleting it")
	}

	table := map[string]string{}
	for _, pair := range jsPair.FindAllStringSubmatch(m[1], -1) {
		table[pair[1]] = pair[2]
	}
	want := htmlout.AriaOrientationDefaults()
	for role, orientation := range want {
		got, ok := table[role]
		if !ok {
			t.Errorf("ARIA_ORIENTATIONS has no row for %q — the static export "+
				"announces that role's axis and the runtime does not", role)
			continue
		}
		if got != orientation {
			t.Errorf("ARIA_ORIENTATIONS[%q] = %q, want %q — the two web targets "+
				"disagree about which way a container of that role runs by default",
				role, got, orientation)
		}
		delete(table, role)
	}
	for role := range table {
		t.Errorf("ARIA_ORIENTATIONS has a row for %q that htmlout does not — either "+
			"a role core dropped, or one target announcing an axis the other never will",
			role)
	}
}

// Every role in the table has to be a role, or the attribute is written for a
// value no exporter can produce.
func TestEveryOrientedRoleIsInTheVocabulary(t *testing.T) {
	known := map[core.Role]bool{}
	for _, r := range core.Roles() {
		known[r] = true
	}
	for role := range htmlout.AriaOrientationDefaults() {
		if !known[core.Role(role)] {
			t.Errorf("ariaOrientations names %q, which is not a core.Role — nothing "+
				"can set it, so the row is dead", role)
		}
	}
}

// The runtime's keyboard reads the announcement rather than the axis. This is
// the pin that keeps the two from becoming two facts again: a rewrite that goes
// back to container.style.flexDirection would still pass every behavioural test
// in keynav_test.mjs — until an author set a FlexDirection the accessibility
// pass had not yet applied — and would silently reopen the divergence.
func TestTheKeyboardReadsTheStatedOrientation(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []struct{ expr, why string }{
		{`const stated = container.getAttribute("aria-orientation");`,
			"the read-back. The arrow pair and the announcement have to be one string, " +
				"or a vertical strip can behave one way and say the other again"},
		{`setOrRemove(el, "aria-orientation", hidden ? "" : ariaOrientation(style, nodeType));`,
			"the write, in applyAccessibility — with aria-hidden winning over it as it " +
				"does over every other attribute there"},
		{`const axis = style.FlexDirection || stackAxisFor(nodeType);`,
			"the axis resolution, which is the one styleFromGrMob uses for the CSS " +
				"declaration — an explicit direction over the node type's own"},
	} {
		if !strings.Contains(src, want.expr) {
			t.Errorf("grmob-runtime.js: %q not found — %s", want.expr, want.why)
		}
	}
}

// AriaOrientationFor's own three answers, which are what both targets write.
func TestOrientationFollowsTheLayoutAxis(t *testing.T) {
	for _, tc := range []struct {
		name     string
		role     core.Role
		nodeType string
		dir      core.FlexDirection
		want     string
	}{
		{"a tab strip is a Row", core.RoleTabList, "Row", "", "horizontal"},
		{"a tab strip turned on its side", core.RoleTabList, "Column", "", "vertical"},
		{"an explicit direction beats the node type", core.RoleTabList, "Row", core.FlexColumn, "vertical"},
		{"reverse runs along the same axis", core.RoleTabList, "Row", "column-reverse", "vertical"},
		{"a listbox is usually a Column", core.RoleListBox, "Column", "", "vertical"},
		{"a listbox laid out across", core.RoleListBox, "Row", "", "horizontal"},
		{"a toolbar takes the attribute and no keyboard", core.RoleToolbar, "Column", "", "vertical"},
		{"a role on a node type that is not a stack falls back to ARIA", core.RoleListBox, "Text", "", "vertical"},
		{"and the tablist default is the other way", core.RoleTabList, "Text", "", "horizontal"},
		{"an unoriented role writes nothing", core.RoleList, "Column", "", ""},
		{"and neither does no role at all", core.RoleNone, "Column", "", ""},
	} {
		if got := htmlout.AriaOrientationFor(tc.role, tc.nodeType, tc.dir); got != tc.want {
			t.Errorf("%s: AriaOrientationFor(%q, %q, %q) = %q, want %q",
				tc.name, tc.role, tc.nodeType, tc.dir, got, tc.want)
		}
	}
}
