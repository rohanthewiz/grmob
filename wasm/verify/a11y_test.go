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
		{`setOrRemove(el, "aria-level", hidden ? "" : headingLevel(style))`,
			"the heading tier, and aria-hidden winning over it as it does over the role"},
	} {
		if !strings.Contains(src, want.expr) {
			t.Errorf("grmob-runtime.js: %q not found — %s. htmlout writes it, so the two web "+
				"targets no longer describe the same screen", want.expr, want.why)
		}
	}
}

// The level's two guards, which htmlout's headingLevel applies as well. Both
// are decisions rather than defensive coding, so both are worth holding:
//
//   - the role guard is ARIA's own scoping. aria-level is defined for heading,
//     listitem and row and for nothing else, which is why a DataTable's column
//     headers take the role and no tier.
//   - the range check drops rather than clamps. Rewriting a 7 into a 6 would
//     put a structure in the document that the app never described.
func TestRuntimeGuardsTheHeadingLevelTheSameWay(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []struct{ expr, why string }{
		{`if (style.AccessibilityRole !== "heading") return "";`,
			"the role guard — a level on anything else describes the depth of something " +
				"that has no depth"},
		{`level >= 1 && level <= 6 ? String(level) : ""`,
			"the range guard, which drops rather than clamps"},
	} {
		if !strings.Contains(src, want.expr) {
			t.Errorf("grmob-runtime.js: headingLevel is missing %q — %s", want.expr, want.why)
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
