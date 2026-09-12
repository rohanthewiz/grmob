package htmlout

import (
	"maps"
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// aria-orientation: which way a composite runs, for the roles ARIA defines the
// attribute on.
//
// # The gap this closes
//
// The WASM runtime reads a composite container's resolved flex axis to decide
// which arrow pair moves within it, so a role="tablist" laid out as a Column
// takes Up/Down — which is correct, and which nothing announced. ARIA's default
// for a tablist is *horizontal*, so a reader in browse mode was told the
// opposite of what the widget did. `listbox` has the same gap in mirror: its
// default is vertical, so a horizontal strip of options behaved one way and
// announced the other.
//
// The attribute is the fix, and it belongs on both web targets rather than on
// the runtime alone, because it is pure semantics — a static export of a
// vertical tab strip describes it just as wrongly as a live one did.
//
// # One fact, not two
//
// The reason the divergence could open at all is that the behaviour and the
// announcement were derived separately: the runtime from the element's
// flex-direction, the exporters from nothing. So the runtime now reads the
// attribute back off the container rather than re-deriving the axis
// (compositeIsVertical in grmob-runtime.js), which makes "which way the arrows
// go" and "which way this says it goes" the same statement by construction.
// A drift between them is no longer a bug that can be written; it would take
// deleting the attribute.
//
// # Why the answer is the layout axis and not a prop
//
// Every container these roles land on is a flex stack whose direction the app
// already stated — by picking Row or Column, or by setting FlexDirection. An
// author who turns a tab strip on its side has said which way its arrows go by
// doing so, and a second prop saying it again is a second claim about one fact,
// with the usual consequence when the two drift.
//
// ARIA's per-role default is the fallback, for the case the axis cannot answer:
// a role placed on a node type that is not a stack (a Text, a Button) and whose
// Style sets no direction. That is the one shape where the framework has
// nothing to read, and ARIA's own answer is better than silence.
//
// # The three roles, and the ones deliberately absent
//
// ARIA scopes aria-orientation to listbox, menu, menubar, radiogroup,
// scrollbar, select, separator, slider, tablist, toolbar, tree and treegrid. Of
// those, core.Role carries exactly three. The rest are absent from the
// vocabulary rather than from this table — see core/role.go — and a role that
// does not exist needs no row here.
//
// `toolbar` is in the table and is not in the runtime's COMPOSITE_MEMBERS,
// which for two releases meant it took the announcement and no keyboard: a
// toolbar's arrow-key pattern needs a notion of which children are its
// controls, and ARIA leaves that open — a toolbar may hold buttons, groups,
// separators and inputs — where the two composite pairs name their members in
// the role itself.
//
// The runtime has a second member rule now (COMPOSITE_FOCUSABLE and
// focusableMembers in grmob-runtime.js), so what this row supplies is read at
// both ends: a toolbar announces its axis here and the arrows follow that same
// axis there. Nothing in this file changed for it, which is the argument for
// the row having been written before the keyboard existed —
// comps.ChipStrip is a Row and horizontal was right either way.
//
// htmlout still writes no tabindex for a toolbar, on the rule that applies to
// all three composites: a roving tabindex with no key handler to move it takes
// every member but one out of the tab order and reaches none of them. See
// wasm/verify/keynav_test.go.
var ariaOrientations = map[string]string{
	string(core.RoleListBox): "vertical",
	string(core.RoleTabList): "horizontal",
	string(core.RoleToolbar): "horizontal",
}

// AriaOrientationDefaults returns a copy of the role → ARIA-default table, for
// the WASM runtime's conformance test — which compares table against table and
// so cannot go through AriaOrientationFor one key at a time.
//
// A copy, not the map itself, for the reason StackAxes and Tags return one: a
// package-level map is reachable and writable by any importer.
func AriaOrientationDefaults() map[string]string {
	out := make(map[string]string, len(ariaOrientations))
	maps.Copy(out, ariaOrientations)
	return out
}

// AriaOrientationFor answers the aria-orientation value for one node, or "" for
// a role the attribute is not defined on.
//
// The axis is resolved exactly as styleValue resolves it for the CSS
// declaration — an explicit FlexDirection over the node type's own stacking
// direction — so the attribute and the layout cannot disagree. The prefix tests
// rather than equality are for "row-reverse" and "column-reverse", which run
// along the same axes as the two they reverse.
//
// grmob-runtime.js restates this as ariaOrientation, and the table it dispatches
// on is pinned to AriaOrientationDefaults by TestRuntimeOrientationTableMatchesGo.
func AriaOrientationFor(role core.Role, nodeType string, dir core.FlexDirection) string {
	def, ok := ariaOrientations[string(role)]
	if !ok {
		return ""
	}
	axis := string(dir)
	if axis == "" {
		axis = StackAxisFor(nodeType)
	}
	switch {
	case strings.HasPrefix(axis, "column"):
		return "vertical"
	case strings.HasPrefix(axis, "row"):
		return "horizontal"
	}
	return def
}
