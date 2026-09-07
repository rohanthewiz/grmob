package core

import "testing"

// Every role core declares is on one side of the tappable-container question,
// and the side it is on has a reason attached.
//
// # The hole this closes
//
// TappableContainerRoles() is two values, and two values written out is
// exactly the shape that goes stale without anything failing. The WASM
// runtime's toolbar keyboard reads them — a container is one of a toolbar's
// controls when it carries one of these roles *and* an OnTap — and
// wasm/verify pins the runtime's copy against the Go one, so the two spellings
// cannot drift.
//
// What neither of those checks can see is a role that should have been added.
// A future RoleCheckbox would be a tappable container by exactly the argument
// the Content roles block makes for RoleButton — a Box the renderers draw as
// scenery, which a reader announces as text until the role says otherwise —
// and adding it would compile, ship, and leave a toolbar of checkbox rows with
// one tab stop and nothing to arrow to. Nothing in this repository would have
// said a word.
//
// So the list is held to the *vocabulary* rather than to itself: every value
// in Roles() is either a tappable container or has an entry below saying why
// it is not. Adding a role to core without deciding fails here.
//
// # Why the reasons are prose and that is not a cop-out
//
// A reason string cannot be checked, and this file does not pretend otherwise
// — what is checked is that *an answer exists*, which is the whole of what a
// census can buy. The value is in the moment of writing one: the four kinds
// below came out of asking the question about twenty-five roles at once, and
// three of them (the members, the valued role, the fallback) are distinctions
// nobody had had to state before.
var notTappable = map[Role]string{
	// Structure. A table, a row, a cell, a list, a listitem, a rowgroup:
	// these say how content is arranged. Putting one on a container makes it
	// readable, not operable, and a tap handler on a row is a shortcut rather
	// than the row's own semantics — components.ListRow's Selectable rows
	// carry RoleOption when they are members of something and nothing when
	// they are not.
	RoleTable:        "structure: arrangement, not operation",
	RoleRowGroup:     "structure: arrangement, not operation",
	RoleRow:          "structure: arrangement, not operation",
	RoleColumnHeader: "structure: arrangement, not operation",
	RoleCell:         "structure: arrangement, not operation",
	RoleList:         "structure: content, and ARIA gives it no keyboard at all",
	RoleListItem:     "structure: arrangement, not operation",

	// The composite containers. Operable, and their tab stop is the widget's
	// own — see KeyboardComposites(). A toolbar taking one as a *control*
	// would put a second keyboard on a widget that already has one, which is
	// the same collision ConcernNestedComposite reports.
	RoleListBox: "a composite container: it owns a keyboard, it is not a control",
	RoleTabList: "a composite container: it owns a keyboard, it is not a control",
	RoleToolbar: "a composite container: it owns a keyboard, it is not a control",

	// The members of those containers. Interactive, and deliberately not here:
	// a member's tab stop belongs to its container's roving tabindex, so a
	// walk that took one as a toolbar control would hand the same element two
	// owners writing tabindex onto it.
	RoleOption: "a composite member: its tab stop belongs to its container",
	RoleTab:    "a composite member: its tab stop belongs to its container",

	// Regions. A screen's landmarks, which a reader jumps between rather than
	// operates. (RoleToolbar is a landmark too and is above, under the louder
	// of its two answers.)
	RoleBanner:     "a landmark: a region to jump to, not a control",
	RoleNavigation: "a landmark: a region to jump to, not a control",
	RoleSearch:     "a landmark: a region to jump to, not a control",
	RoleTabPanel:   "a region: it holds whatever a screen holds",

	// Live regions. Content that announces itself when it changes; nothing
	// about them is operable.
	RoleStatus: "a live region: it announces, it is not pressed",
	RoleAlert:  "a live region: it announces, it is not pressed",
	RoleLog:    "a live region: it announces, it is not pressed",

	// Content. What a node is when it is not a region and not a control.
	RoleHeading: "content: an outline entry",
	RoleImg:     "content: a picture standing in for its parts",
	RoleGroup:   "content: the fallback that makes a name legal on a container",

	// The one valued role. A progressbar is read, not operated — ARIA's own
	// division from `slider`, which is the operable member of that family and
	// which core has no role for because core.Slider is a node type that
	// exports as <input type="range">.
	RoleProgressBar: "read, not operated: ARIA's own progressbar/slider division",
}

func TestEveryRoleAnswersTheTappableContainerQuestion(t *testing.T) {
	tappable := map[Role]bool{}
	for _, r := range TappableContainerRoles() {
		tappable[r] = true
		if _, also := notTappable[r]; also {
			t.Errorf("core.Role %q is both a tappable container and excluded as one", r)
		}
	}

	for _, r := range Roles() {
		if tappable[r] {
			continue
		}
		if _, ok := notTappable[r]; !ok {
			t.Errorf("core.Role %q is neither in TappableContainerRoles() nor "+
				"excluded from it: decide. If putting it on a Box with an OnTap "+
				"should make that Box one of a toolbar's controls, add it to the "+
				"list — otherwise add a line here saying why not. Left undecided it "+
				"ships as a role a toolbar steps over, and the toolbar looks fine",
				r)
		}
	}

	// And in the other direction: an entry naming a role the vocabulary no
	// longer has is an argument about nothing, and would hide the next real
	// gap behind a count that already looks complete.
	declared := map[Role]bool{}
	for _, r := range Roles() {
		declared[r] = true
	}
	for r := range notTappable {
		if !declared[r] {
			t.Errorf("notTappable names %q, which core.Roles() does not declare", r)
		}
	}
	for _, r := range TappableContainerRoles() {
		if !declared[r] {
			t.Errorf("TappableContainerRoles() names %q, which core.Roles() does "+
				"not declare", r)
		}
	}
}
