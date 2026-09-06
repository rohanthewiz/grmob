package main

import (
	"strings"
	"testing"
)

// The accessibility attributes are authored twice — htmlout builds a static
// document, grmob-runtime.js sets attributes on live elements — for the same
// reason the Modal chassis and the TabView chrome are: neither web target can
// call into the other. a11y_test.mjs proves the runtime's half *behaves*, but
// it runs only under run.sh, which needs Node and which a human has to
// remember. This is the half `go test ./...` reaches.
//
// What is pinned is the contract rather than the implementation: the attribute
// names, and the two guards that decide whether aria-level is written at all.
// A drift in any of them is silent — the dialog still opens, the heading still
// announces, and the two web targets simply stop saying the same thing.
func TestRuntimeWritesTheSameAccessibilityAttributes(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []struct{ expr, why string }{
		{`setOrRemove(el, "aria-modal", dialog ? "true" : "")`,
			`the modal claim. htmlout's modalSemantics writes it, and it is not expressible ` +
				`through core.Role, so a Modal is the only node that can`},
		{`(dialog ? "dialog" : "")`,
			`the dialog role, defaulted after the author's own core.Role so a hand-built ` +
				`Modal that states one still wins`},
		{`setOrRemove(el, "aria-level", hidden ? "" : ariaLevel(style))`,
			"the level — a heading's tier or a nested item's depth, whichever the role calls " +
				"for — and aria-hidden winning over it as it does over the role"},
		{`setOrRemove(el, "aria-selected", selected[0])`,
			"the selection half of the state, for a tab, a row or a column header"},
		{`setOrRemove(el, "aria-pressed", selected[1])`,
			"the toggle half, for a button. Both are written on every call rather than only " +
				"the one the role asks for: a role can change between passes, and writing " +
				"one would leave the other standing"},
	} {
		if !strings.Contains(src, want.expr) {
			t.Errorf("grmob-runtime.js: %q not found — %s. htmlout writes it, so the two web "+
				"targets no longer describe the same screen", want.expr, want.why)
		}
	}
}

// The level's guards, which htmlout's ariaLevel applies as well. None of them
// is defensive coding, so all of them are worth holding:
//
//   - the role dispatch is ARIA's own scoping. aria-level is defined for
//     heading, listitem and row and for nothing else, which is why a
//     DataTable's column headers take the role and no level. Writing it as a
//     switch is also what makes core's two level fields mutually exclusive:
//     one attribute, one role, one arm.
//   - both range checks drop rather than clamp, and they drop different
//     things. Rewriting a heading's 7 into a 6 would put a structure in the
//     document the app never described; capping a nesting depth at 6 would
//     flatten a tree ARIA considers perfectly well-formed.
func TestRuntimeGuardsTheLevelsTheSameWay(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []struct{ expr, why string }{
		{`switch (style.AccessibilityRole) {`,
			"the role dispatch — a level on any other role describes the depth of something " +
				"that has no depth, and the switch is what keeps the two level fields from " +
				"contending for one attribute"},
		{`case "heading": {`, "the heading arm"},
		{`case "listitem":`, "the listitem arm"},
		{`case "row": {`, "the row arm, which shares the nesting depth with listitem"},
		{`level >= 1 && level <= 6 ? String(level) : ""`,
			"the heading range, which drops rather than clamps"},
		{`level >= 1 ? String(level) : ""`,
			"the nesting range, which has no ceiling — ARIA asks only for an integer of 1 " +
				"or more, and a cap would flatten a deep tree"},
		{`default:
                return "";`,
			"the catch-all. A role with no arm must write no level rather than falling " +
				"through to one"},
	} {
		if !strings.Contains(src, want.expr) {
			t.Errorf("grmob-runtime.js: ariaLevel is missing %q — %s", want.expr, want.why)
		}
	}
}

// The selected state's guards, which htmlout's ariaSelected applies as well.
//
// The role list is ARIA's own scoping and not a shortlist: aria-selected is
// defined for gridcell, option, row, tab, columnheader and rowheader, of which
// core.Role carries three, and aria-pressed for button alone. The two near
// misses are the ones a reader of the switch will wonder about — a `cell` is
// not a gridcell and a `listitem` is not an option — so their absence is as
// load-bearing as the arms that are there.
//
// The Button node type is the one arm that is not ARIA's: a core.Button
// already is a button, which is what lets components.Chip carry a state
// without setting a role. Same rule that gives a Modal its dialog role, and
// the reason this function takes the node type at all.
func TestRuntimeGuardsTheSelectedStateTheSameWay(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []struct{ expr, why string }{
		{`function ariaSelected(style, nodeType) {`,
			"the function htmlout's ariaSelected mirrors"},
		{`if (!value) return ["", ""];`,
			"the zero value writing nothing at all, which is what every node in every " +
				"existing tree carries"},
		{`case "tab":`, "the tab arm"},
		{`case "row":`, "the row arm"},
		{`case "columnheader":`, "the column-header arm"},
		{`case "button":
                return ["", value];`,
			"the button arm, which is the one that becomes aria-pressed"},
		{`return nodeType === "Button" ? ["", value] : ["", ""];`,
			"the node type standing in for an unstated role — without it, a components.Chip " +
				"would be the one node that could not carry the attribute it most wants"},
		{`default:
                return ["", ""];`,
			"the catch-all. A role ARIA does not scope either attribute to must write " +
				"neither, rather than falling through to one"},
	} {
		if !strings.Contains(src, want.expr) {
			t.Errorf("grmob-runtime.js: ariaSelected is missing %q — %s", want.expr, want.why)
		}
	}
}

// A Modal core built carries no Style at all, so the applyStyle path — the one
// applyAccessibility normally rides on — never runs for it. The chassis in
// createElement is what covers that case, and it goes through the same
// function so the two routes cannot drift.
//
// This is the check the .mjs suite makes behaviorally ("a Modal announces as a
// dialog with no Style at all"); here it is the one line that makes it true.
func TestRuntimeGivesAStylelessModalItsSemantics(t *testing.T) {
	src := runtimeSource(t)
	if !strings.Contains(src, `applyAccessibility(el, node.Style || {}, "Modal")`) {
		t.Error(`grmob-runtime.js: createElement's Modal branch no longer applies the dialog ` +
			`semantics — core.ModalNode has no Style, so nothing else would`)
	}
}
