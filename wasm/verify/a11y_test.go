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
		{`const own = hidden ? "" : ownRole(nodeType);`,
			`the self-roling types' role, looked up from what the node IS. ownRole is the ` +
				`runtime's copy of htmlout.ownRoles — a dialog for a Modal, a switch for a ` +
				`core.Switch — and TestRuntimeOwnRolesMatchGo holds the two tables equal`},
		{`? (style.AccessibilityRole || own)`,
			`that role, defaulted after the author's own core.Role so a hand-built node ` +
				`that states one still wins`},
		{`setOrRemove(el, "aria-level", hidden ? "" : ariaLevel(style))`,
			"the level — a heading's tier or a nested item's depth, whichever the role calls " +
				"for — and aria-hidden winning over it as it does over the role"},
		{`setOrRemove(el, "aria-selected", selected[0])`,
			"the selection half of the state, for a tab, a row or a column header"},
		{`setOrRemove(el, "aria-pressed", selected[1])`,
			"the toggle half, for a button. Both are written on every call rather than only " +
				"the one the role asks for: a role can change between passes, and writing " +
				"one would leave the other standing"},
		{`setOrRemove(el, "id", hidden ? "" : (style.AccessibilityID || ""))`,
			"the element identity an aria-controls somewhere else on the page points at. " +
				"core.Style.AccessibilityID is written verbatim, and a page that carries one " +
				"is left unwired by the TabView so the two writers never share the slot"},
		{`setOrRemove(el, "aria-controls", hidden ? "" : (style.AccessibilityControls || ""))`,
			"the pointing. It is the vocabulary's one reference to another element, which is " +
				"the whole reason it is a prop rather than a value like the hint"},
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
// core.Role carries four, and aria-pressed for button alone. The near miss is
// the one a reader of the switch will wonder about — a `cell` is not a
// gridcell — and so is the pair the switch now splits: `option` is here and
// `listitem`, which reads like its synonym, is deliberately not. Their absence
// is as load-bearing as the arms that are there.
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
		{`case "option":`, "the option arm — the one that lets a selectable row in a " +
			"listbox announce its state, which is what components.ListRow.Selectable buys"},
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

// The expanded state's guards, which htmlout's ariaExpanded applies as well.
//
// The reason this is a test of its own rather than three more lines in the
// selection's is the whole point of the field: aria-expanded's role list is
// ARIA's own and it is *not* aria-selected's. It is defined for application,
// button, checkbox, combobox, gridcell, link, listbox, menuitem, row,
// rowheader, tab and treeitem, and inherits into columnheader — of which
// core.Role carries six. Relative to the selection that drops `option` and
// adds `link` and `listbox`, so a switch shared between the two would be wrong
// at four roles and wrong silently.
//
// The absent arms are held below for the same reason the present ones are.
// `option` reading like it belongs is the trap here, exactly as `listitem`
// looking like `option` is the trap one function up.
func TestRuntimeGuardsTheExpandedStateTheSameWay(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []struct{ expr, why string }{
		{`function ariaExpanded(style, nodeType) {`,
			"the function htmlout's ariaExpanded mirrors"},
		{`const value = style.AccessibilityExpanded || "";`,
			"reading the field at all — an unread key is not an error in JavaScript, so a " +
				"runtime that dropped this line would render every accordion header as a " +
				"button with nothing behind it"},
		{`if (!value) return "";`,
			"the zero value writing nothing, which is what every node in every existing " +
				"tree carries"},
		{`case "link":`, "the link arm, one of the two roles this list has and the " +
			"selection's does not"},
		{`case "listbox":`, "the listbox arm — the popup half of a combobox, and the other " +
			"role the two lists disagree about"},
		{`return nodeType === "Button" ? value : "";`,
			"the node type standing in for an unstated role. ARIA's disclosure pattern is a " +
				"button, so without this the attribute would be defined for exactly the node " +
				"type that could not carry it"},
		{`default:
                return "";`,
			"the catch-all. A role ARIA does not scope this to writes nothing — and there " +
				"is no RoleGroup-shaped rescue here, because `group` is not on the list"},
	} {
		if !strings.Contains(src, want.expr) {
			t.Errorf("grmob-runtime.js: ariaExpanded is missing %q — %s", want.expr, want.why)
		}
	}

	// The attribute is written on every call, including when the guard returns
	// "". This is the totality rule, and it is the line a "tidier" guarded
	// write would remove: a disclosure that stops being one would keep the
	// attribute it had, announcing a section that is gone as open.
	if !strings.Contains(src, `setOrRemove(el, "aria-expanded", hidden ? "" : ariaExpanded(style, nodeType));`) {
		t.Error("grmob-runtime.js: applyAccessibility does not write aria-expanded " +
			"unconditionally — an update-style patch carries the whole new Style, so a " +
			"guarded write leaves a stale state standing")
	}
}

// A Modal core built carries no Style at all, and applyAccessibility rides on
// the applyStyle path — so whether a modal announces as a dialog comes down to
// whether that path runs for a node with nothing to style.
//
// It does, for every node: createElement passes an empty object where there is
// no Style rather than skipping the call. The Modal branch used to restate the
// semantics itself precisely because the call was conditional, and that second
// copy is gone now that it cannot be reached differently from the first.
//
// This is the check the .mjs suite makes behaviorally ("a Modal announces as a
// dialog with no Style at all", and styleless_test.mjs from the other side);
// here it is the one line that makes it true. stack_test.go pins the same line
// for the other thing that rests on it — a styleless container's flex axis.
func TestRuntimeGivesAStylelessModalItsSemantics(t *testing.T) {
	src := runtimeSource(t)
	if !strings.Contains(src, `applyStyle(el, node.Style || {}, node.Type);`) {
		t.Error(`grmob-runtime.js: createElement no longer styles a node that carries no ` +
			`Style — core.ModalNode has no Style, so nothing would apply the dialog semantics`)
	}
}

// The role a named container is given when it has none of its own, restated in
// the runtime as ariaRole and in htmlout as ariaRole. Both must agree, and the
// consequence of drift is a silence rather than an error: the target that
// stopped supplying the role keeps writing an aria-label that ARIA prohibits
// on `generic` and every browser drops.
//
// The three parts pinned here are the three the rule is made of — the
// author-wins short circuit, the name that earns the role, and the tag test
// that keeps it off a <button>, an <img> or an <input>, which can carry a name
// unaided and would *lose* the role their tag implies. See core.RoleGroup.
func TestRuntimeSuppliesTheGroupRole(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []struct{ expr, why string }{
		{"function ariaRole(el, style) {",
			"the rule itself, which htmlout states as ariaRole in export.go"},
		{"if (authored) return authored;",
			"an author who said what the node is keeps their word; the fallback only ever " +
				"fills an empty slot"},
		{"if (!style.AccessibilityLabel) return \"\";",
			"the fallback exists to rescue a name. A roleless, nameless container is a div, " +
				"which is what it should be"},
		{`return GENERIC_TAGS.has(el.tagName.toLowerCase()) ? "group" : "";`,
			"only the tags whose implicit role is `generic` may be given one — writing a role " +
				"onto a <button> or an <img> replaces the role the browser already gives it"},
		{"? (style.AccessibilityRole || own)\n            : ariaRole(el, style))",
			"a self-roling node answers from ownRole before the fallback is consulted, which " +
				"is why ariaRole needs no equivalent of htmlout's CarriesOwnRole guard. It " +
				"matters more for a Switch than it ever did for a Modal: an <input> is not a " +
				"generic tag, so the fallback would decline to write any role at all and the " +
				"control would announce itself as the checkbox it is made of"},
	} {
		if !strings.Contains(src, want.expr) {
			t.Errorf("grmob-runtime.js: %q not found — %s. htmlout supplies the role, so the "+
				"two web targets no longer describe the same screen", want.expr, want.why)
		}
	}
}

// The TabView wiring's two shared slots, which the group role and the IDREF
// pair both reach into.
//
// role: the wiring writes "tabpanel" and applyAccessibility writes the author's
// role or the supplied group. canBeTabPanel accepts a group because a tabpanel
// says everything a group says and one thing more — and it *must*, since a
// named page is given a group whether the author asked or not, and rejecting it
// would silently stop every page with an AccessibilityLabel from being wired.
//
// id: the wiring writes panelID and applyAccessibility writes the author's
// AccessibilityID. Here the author wins outright, because something else on the
// page is pointing at their string; comparing against the wiring's own value is
// what tells an authored id from one this runtime wrote a sync ago.
func TestRuntimeSharesTheRoleAndIDSlotsWithTheTabWiring(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []struct{ expr, why string }{
		{`(role === null || role === "group" || (role === "tabpanel" && mine))`,
			"the group exemption, which htmlout states in tabPanelBox, and the marker " +
				"that stands in for the value test the wiring used to make. `mine` is " +
				"data-grmob-panel: until core.RoleTabPanel existed, an element carrying " +
				`"tabpanel" could only have got it from wireTabPanel, which made the ` +
				"absence of that constant load-bearing and kept a hand-built strip from " +
				"ever naming its own panels"},
		{`const mine = page.dataset.grmobPanel !== undefined;`,
			"where the marker is read. htmlout writes it too, so the two web targets " +
				"emit one document even though only this one syncs twice"},
		{`if (style.AccessibilityRole) {
            delete el.dataset.grmobPanel;
        }`,
			"an authored role voiding the wiring's claim on the slot. It is the one case " +
				"the marker alone cannot decide — an author who writes core.RoleTabPanel " +
				"puts the wiring's own value in the slot — and applyAccessibility is the " +
				"only place this target has the Style in hand, which is where htmlout's " +
				"tabPanelBox reads it from"},
		{`(id === null || id === panelId(scope, i))`,
			"a page carrying its own AccessibilityID is left unwired rather than having the " +
				"id taken from under an aria-controls that points at it"},
		{`setOrRemove(page, "role", named ? "group" : "");`,
			"unwiring restores what applyAccessibility would have left — a named page keeps " +
				"the group that makes its name audible instead of falling back into silence"},
		{`delete page.dataset.grmobPanel;`,
			"the marker going with the role it vouched for. Left standing, it would make " +
				"the next sync treat an author's own element as this wiring's"},
	} {
		if !strings.Contains(src, want.expr) {
			t.Errorf("grmob-runtime.js: %q not found — %s", want.expr, want.why)
		}
	}
}

// The value family's guards, which htmlout's ariaValue applies as well.
//
// This is the fourth state mapping in applyAccessibility and the one with the
// narrowest role list: aria-valuenow and its bounds are defined for six roles
// and core.Role carries one of them, progressbar. The near miss worth pinning
// is core.Slider, which is a node *type* rather than a role — it exports as
// <input type="range"> and states its own value, so an ARIA range on top would
// be a second claim about one fact.
//
// The other pin here is that all four attributes are written on every call. The
// role can change between passes, and a bar that stops being a progressbar must
// not keep a range — the same totality the selection pair is held to, one
// attribute wider.
func TestRuntimeGuardsTheValueTheSameWay(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []struct{ expr, why string }{
		{`function ariaValue(style) {`,
			"the function htmlout's ariaValue mirrors"},
		{`case "progressbar":`,
			"the one role ARIA's value family and core.Role have in common"},
		{`const value = hidden ? EMPTY_VALUE : ariaValue(style);`,
			"aria-hidden winning over the range, as it does over every other attribute here"},
		{`setOrRemove(el, "aria-valuenow", value.Now);`, "the position"},
		{`setOrRemove(el, "aria-valuemin", value.Min);`, "the lower bound"},
		{`setOrRemove(el, "aria-valuemax", value.Max);`, "the upper bound"},
		{`setOrRemove(el, "aria-valuetext", value.Text);`,
			"the spoken form, which a reader announces instead of the number"},
		{`default:
                return EMPTY_VALUE;`,
			"the catch-all. A role with no arm must write no value rather than falling " +
				"through to one"},
	} {
		if !strings.Contains(src, want.expr) {
			t.Errorf("grmob-runtime.js: ariaValue is missing %q — %s. htmlout writes it, "+
				"so the two web targets no longer describe the same bar", want.expr, want.why)
		}
	}
}
