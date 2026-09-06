package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

// The two web targets deliberately do different things with a listbox and a
// tablist, and this file holds the line between them.
//
//	wasm/grmob-runtime.js   the whole ARIA keyboard pattern: a roving
//	                        tabindex, arrow movement, Home/End, activation
//	htmlout                 nothing at all
//
// Everywhere else in this repository a difference between the two DOM targets
// is a bug being closed — the Modal chassis, the tab bar, the stack axes are
// all "the two web targets must agree" pins. This one is the exception, and it
// needs saying out loud or the next reader closes it:
//
// A roving tabindex without the key handler that moves it is strictly worse
// than no pattern. It takes every member but one out of the page's tab order
// and supplies nothing that reaches the rest, so a static export would go from
// "three tab stops" to "one tab stop and two unreachable rows". htmlout is not
// a runtime (docs/platforms/exporters.md, and its own TabView bar is inert
// chrome), so the attribute has nothing to move it and must not be written.
//
// tabindex is behaviour here, not semantics. Every *semantic* half of the two
// patterns — the roles, aria-selected, the aria-controls wiring — is already
// written by both targets and pinned as such elsewhere.

// The composite table's two rows, as core spells them. A rename in core has to
// reach the runtime, and the failure if it does not is the quiet kind: the
// role goes on being written, the runtime goes on not recognising it, and the
// widget silently loses its keyboard again.
var compositeRoles = []struct{ container, member core.Role }{
	{core.RoleListBox, core.RoleOption},
	{core.RoleTabList, core.RoleTab},
}

// COMPOSITE_MEMBERS is a const object literal rather than a lookup function,
// so it is read directly instead of through parseRuntimeTable — which requires
// the `[param] || fallback` tail that proves it found the right braces. The
// equivalent proof here is the assignment being matched whole, name included.
func TestRuntimeCompositeRolesMatchCore(t *testing.T) {
	src := runtimeSource(t)

	decl := regexp.MustCompile(`const COMPOSITE_MEMBERS = \{([^}]*)\};`)
	m := decl.FindStringSubmatch(src)
	if m == nil {
		t.Fatal("grmob-runtime.js: no COMPOSITE_MEMBERS object literal — if it was " +
			"renamed or given a computed shape, update this test rather than deleting it")
	}

	table := map[string]string{}
	for _, pair := range jsPair.FindAllStringSubmatch(m[1], -1) {
		table[pair[1]] = pair[2]
	}
	if len(table) != len(compositeRoles) {
		t.Fatalf("COMPOSITE_MEMBERS has %d rows, want %d: %v",
			len(table), len(compositeRoles), table)
	}
	for _, want := range compositeRoles {
		got, ok := table[string(want.container)]
		if !ok {
			t.Errorf("COMPOSITE_MEMBERS has no row for core.Role %q — a container "+
				"carrying it gets no keyboard, and nothing says so", want.container)
			continue
		}
		if got != string(want.member) {
			t.Errorf("COMPOSITE_MEMBERS[%q] = %q, want %q — the runtime is looking "+
				"for members that no exporter writes", want.container, got, want.member)
		}
	}

	// The default-orientation table has to name the same containers, or a
	// composite whose axis nothing set falls through to `undefined` and takes
	// the horizontal arrows whatever it is.
	orient := regexp.MustCompile(`const COMPOSITE_DEFAULT_VERTICAL = \{([^}]*)\};`)
	om := orient.FindStringSubmatch(src)
	if om == nil {
		t.Fatal("grmob-runtime.js: no COMPOSITE_DEFAULT_VERTICAL object literal")
	}
	for _, want := range compositeRoles {
		if !strings.Contains(om[1], string(want.container)+":") {
			t.Errorf("COMPOSITE_DEFAULT_VERTICAL has no row for %q — a container of "+
				"that role whose axis nothing set gets the horizontal arrows by "+
				"accident rather than by decision", want.container)
		}
	}
}

// htmlout writes the semantics and stops there. See the file comment for why
// this is the one two-web-target difference that is deliberate.
func TestTheStaticExportWritesNoRovingTabindex(t *testing.T) {
	for _, pair := range compositeRoles {
		container := &core.Node{
			Type:  "Column",
			Style: &core.Style{AccessibilityRole: pair.container},
			Children: []*core.Node{
				{Type: "Box", Style: &core.Style{
					AccessibilityRole:     pair.member,
					AccessibilitySelected: core.SelectedOn,
				}},
				{Type: "Box", Style: &core.Style{
					AccessibilityRole:     pair.member,
					AccessibilitySelected: core.SelectedOff,
				}},
			},
		}
		out := htmlout.ExportHTML(container)

		if strings.Contains(out, "tabindex") {
			t.Errorf("htmlout wrote a tabindex for a %s:\n%s\n\n"+
				"A roving tabindex with nothing to move it takes every member but "+
				"one out of the tab order and reaches none of them. The static "+
				"export has no key handler, so it must write no tab stops.",
				pair.container, out)
		}
		// The semantic half is still there, and is the reason the runtime can
		// do the rest without a single new prop.
		for _, want := range []string{
			`role="` + string(pair.container) + `"`,
			`role="` + string(pair.member) + `"`,
			`aria-selected="true"`,
			`aria-selected="false"`,
		} {
			if !strings.Contains(out, want) {
				t.Errorf("htmlout dropped %s from a %s:\n%s", want, pair.container, out)
			}
		}
	}
}
