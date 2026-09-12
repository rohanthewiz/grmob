package verify

import (
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// Every role core declares is on one side of the tappable-container question,
// the side it is on has a reason attached, and the *kind* of reason is checked
// against the specification wherever the specification has an opinion.
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
// core's "Content roles" block makes for RoleButton — a Box the renderers draw
// as scenery, which a reader announces as text until the role says otherwise —
// and adding it would compile, ship, and leave a toolbar of checkbox rows with
// one tab stop and nothing to arrow to. Nothing in this repository would have
// said a word.
//
// So the list is held to the *vocabulary* rather than to itself: every value
// in core.Roles() is either a tappable container or has an entry below saying
// why it is not. Adding a role to core without deciding fails here.
//
// # Why it lives here now
//
// It was written in package core, next to the vocabulary it censuses, and the
// reasons were prose — every one of them, with a comment saying so and calling
// that the whole of what a census can buy. That was true while core was the
// only thing in the room. This package is the room where a claim about ARIA
// has an authority to be wrong against (see doc.go), and four of the kinds the
// census sorts roles into turn out to be *derivable* from the fixture and from
// core's own composite tables. So the table moved to where it can be checked,
// and core keeps the half that needs no fixture — that its two exported role
// lists agree with each other (core/role_control_test.go).
//
// This is the mechanism `refusal.Nesting` uses one file over: declare the
// value, derive it independently, and let the two disagree in a test rather
// than in somebody's memory.

// roleKind is why a role is not a tappable container. Four of the seven are
// derived below; three are prose, and the split is the point of the type.
type roleKind string

const (
	// Derived from ARIA's Required Owned Elements: a role that owns specific
	// children, or that such a role owns.
	kindStructure roleKind = "structure"
	// Derived from core.KeyboardComposites().
	kindComposite roleKind = "composite container"
	// Derived from core.CompositeMemberRole() over those containers.
	kindMember roleKind = "composite member"
	// Derived from the fixture's attribute list: the roles ARIA gives a value
	// range to.
	kindValued roleKind = "valued"

	// The three with no authority in the fixture. ARIA's landmark and live
	// region groupings are prose in the specification's own text rather than
	// rows in a role definition, and `aria/gen` reads role definitions — so
	// nothing generated from the specification can contradict these three, and
	// saying so is better than letting them sit beside the derived four
	// looking equally checked. This is exactly the split refusal.Nesting and
	// refusal.Shape make.
	kindLandmark roleKind = "landmark"
	kindLive     roleKind = "live region"
	// And the fallback, which is not undeclarable but underivable: "content"
	// is what a role is when it is none of the above, so a derivation for it
	// would be the negation of the other six and would agree with anything.
	kindContent roleKind = "content"
)

// derivedKinds are the four the fixture and core can answer for. A set rather
// than a predicate on roleKind so that adding a kind is a decision about which
// half it lands in, made here, rather than a default.
var derivedKinds = map[roleKind]bool{
	kindStructure: true,
	kindComposite: true,
	kindMember:    true,
	kindValued:    true,
}

type nonControl struct {
	// Kind is why, in the vocabulary above.
	Kind roleKind
	// Why is the argument, in the form it is worth reading in a failure. It
	// carries what the kind cannot: which of the two answers a role with two
	// was filed under, and what goes wrong if the decision is reversed.
	Why string
}

var notTappable = map[core.Role]nonControl{
	// Structure. A table, a row, a cell, a list, a listitem, a rowgroup:
	// these say how content is arranged. Putting one on a container makes it
	// readable, not operable, and a tap handler on a row is a shortcut rather
	// than the row's own semantics — comps.ListRow's Selectable rows
	// carry RoleOption when they are members of something and nothing when
	// they are not.
	core.RoleTable:        {kindStructure, "arrangement, not operation"},
	core.RoleRowGroup:     {kindStructure, "arrangement, not operation"},
	core.RoleRow:          {kindStructure, "arrangement, not operation"},
	core.RoleColumnHeader: {kindStructure, "arrangement, not operation"},
	core.RoleCell:         {kindStructure, "arrangement, not operation"},
	core.RoleList:         {kindStructure, "content, and ARIA gives it no keyboard at all"},
	core.RoleListItem:     {kindStructure, "arrangement, not operation"},

	// The composite containers. Operable, and their tab stop is the widget's
	// own — see core.KeyboardComposites(). A toolbar taking one as a *control*
	// would put a second keyboard on a widget that already has one, which is
	// the same collision ConcernNestedComposite reports.
	core.RoleListBox: {kindComposite, "it owns a keyboard, it is not a control"},
	core.RoleTabList: {kindComposite, "it owns a keyboard, it is not a control"},
	core.RoleToolbar: {kindComposite, "it owns a keyboard, it is not a control"},

	// The members of those containers. Interactive, and deliberately not
	// tappable containers: a member's tab stop belongs to its container's
	// roving tabindex, so a walk that took one as a toolbar control would hand
	// the same element two owners writing tabindex onto it.
	core.RoleOption: {kindMember, "its tab stop belongs to its container"},
	core.RoleTab:    {kindMember, "its tab stop belongs to its container"},

	// Regions. A screen's landmarks, which a reader jumps between rather than
	// operates. (RoleToolbar is a landmark too and is above, under the louder
	// of its two answers — which is the kind of thing Why is for.)
	core.RoleBanner:     {kindLandmark, "a region to jump to, not a control"},
	core.RoleNavigation: {kindLandmark, "a region to jump to, not a control"},
	core.RoleSearch:     {kindLandmark, "a region to jump to, not a control"},
	core.RoleTabPanel:   {kindLandmark, "a region: it holds whatever a screen holds"},

	// Live regions. Content that announces itself when it changes; nothing
	// about them is operable.
	core.RoleStatus: {kindLive, "it announces, it is not pressed"},
	core.RoleAlert:  {kindLive, "it announces, it is not pressed"},
	core.RoleLog:    {kindLive, "it announces, it is not pressed"},

	// Content. What a node is when it is not a region and not a control.
	core.RoleHeading: {kindContent, "an outline entry"},
	core.RoleImg:     {kindContent, "a picture standing in for its parts"},
	core.RoleGroup:   {kindContent, "the fallback that makes a name legal on a container"},

	// The one valued role. A progressbar is read, not operated — ARIA's own
	// division from `slider`, which is the operable member of that family and
	// which core has no role for because core.Slider is a node type that
	// exports as <input type="range">.
	core.RoleProgressBar: {kindValued, "ARIA's own progressbar/slider division"},
}

// The census: every role decided, in one direction and the other.
func TestEveryRoleAnswersTheTappableContainerQuestion(t *testing.T) {
	tappable := map[core.Role]bool{}
	for _, r := range core.TappableContainerRoles() {
		tappable[r] = true
		if _, also := notTappable[r]; also {
			t.Errorf("core.Role %q is both a tappable container and excluded as one", r)
		}
	}

	declared := map[core.Role]bool{}
	for _, r := range core.Roles() {
		declared[r] = true
		if tappable[r] {
			continue
		}
		entry, ok := notTappable[r]
		if !ok {
			t.Errorf("core.Role %q is neither in core.TappableContainerRoles() nor "+
				"excluded from it: decide. If putting it on a Box with an OnTap "+
				"should make that Box one of a toolbar's controls, add it to the "+
				"list — otherwise add a line here saying why not. Left undecided it "+
				"ships as a role a toolbar steps over, and the toolbar looks fine", r)
			continue
		}
		if entry.Why == "" {
			t.Errorf("notTappable[%q] has a kind and no argument — the kind says "+
				"which bucket, and Why is what a reader needs when the bucket is "+
				"the surprising one", r)
		}
	}

	// And in the other direction: an entry naming a role the vocabulary no
	// longer has is an argument about nothing, and would hide the next real
	// gap behind a count that already looks complete.
	for r := range notTappable {
		if !declared[r] {
			t.Errorf("notTappable names %q, which core.Roles() does not declare", r)
		}
	}
}

// The four kinds the specification and core can answer for, derived and then
// compared with what the table says.
//
// # Why derive at all
//
// The table's reasons used to be prose top to bottom, with a comment arguing
// that a reason string cannot be checked and that what a census buys is the
// moment of writing one. Half of that is right and half was a habit. "This
// role is one of ARIA's structural containers" is not an opinion — it is the
// Required Owned Elements row, which this package already has in machine form
// and already uses to hold the refusals table's Nesting field honest.
//
// What the derivation catches is a role filed under the wrong kind, which is
// exactly the failure a prose reason cannot have: a sentence saying
// "arrangement, not operation" beside a role ARIA owns nothing with reads
// perfectly and is wrong, and the next person to reach for it copies it.
//
// # The four derivations
//
//	structure    ARIA requires the role to own specific children, or a role
//	             that is not a composite requires to own it. The second half is
//	             what puts `cell`, `columnheader` and `listitem` in — they own
//	             nothing themselves and are only ever somebody's contents.
//	composite    core.KeyboardComposites(), which is core's own statement and
//	             is separately pinned to the runtime's two tables.
//	member       core.CompositeMemberRole() over those containers.
//	valued       the fixture's attribute list carries aria-valuenow.
//
// The composite exclusion in the structure rule is load-bearing and is not a
// special case: `listbox` owns `option` and `tablist` owns `tab`, so without
// it every composite and every member would derive as structure and the three
// kinds would collapse into one. The question the rule asks is whether a role
// is part of ARIA's *arrangement* vocabulary, and a widget's members are not —
// they have a keyboard, which is the whole distinction the census turns on.
//
// `group` is the case that shows the rule is doing work rather than agreeing
// by construction: it is in `listbox`'s Required Owned Elements, so a rule
// that did not exclude composite owners would file it as structure. ARIA uses
// it as a pure wrapper (it requires nothing itself), the table files it as the
// content fallback, and the derivation agrees — for the stated reason and not
// by luck.
func TestTheFourDerivableKindsAgreeWithTheSpecification(t *testing.T) {
	spec := loadSpec(t)

	composites := map[core.Role]bool{}
	members := map[core.Role]bool{}
	for _, c := range core.KeyboardComposites() {
		composites[c] = true
		if m, _ := core.CompositeMemberRole(c); m != "" {
			members[m] = true
		}
	}

	// A role ARIA requires to own children, or one that such a role requires
	// to own — over core's own vocabulary, and with the composites and their
	// members held out.
	declared := map[core.Role]bool{}
	for _, r := range core.Roles() {
		declared[r] = true
	}
	structural := map[core.Role]bool{}
	for name, def := range spec.Roles {
		role := core.Role(name)
		// A role core does not carry cannot say anything about how core
		// arranges content. Without this, `menu`, `menubar` and `tree` — three
		// patterns aria/verify/refusals_test.go records as refused, precisely
		// because core carries none of their member roles — would file `group`
		// as structure through their own Required Owned Elements rows. It is
		// in theirs for the reason the refusals table already states: ARIA
		// uses `group` as a pure wrapper in every pattern that allows it.
		if !declared[role] || composites[role] || members[role] {
			continue
		}
		if len(def.RequiredOwned) == 0 {
			continue
		}
		structural[role] = true
		for _, owned := range def.RequiredOwned {
			o := core.Role(owned)
			if declared[o] && !composites[o] && !members[o] {
				structural[o] = true
			}
		}
	}

	for role, entry := range notTappable {
		def, ok := spec.Roles[string(role)]
		if !ok {
			// aria_test.go is what reports a core.Role ARIA does not have; a
			// second complaint here would be noise, but the derivations below
			// would all read an empty definition and agree with anything.
			continue
		}
		want := roleKind("")
		switch {
		case composites[role]:
			want = kindComposite
		case members[role]:
			want = kindMember
		case structural[role]:
			want = kindStructure
		case hasAttribute(def, "aria-valuenow"):
			want = kindValued
		}
		if want == "" {
			// Nothing derivable. The entry must then be one of the three kinds
			// that say so — an undeclarable kind is a claim that no authority
			// exists, and it is only honest while none does.
			if derivedKinds[entry.Kind] {
				t.Errorf("notTappable[%q].Kind is %q, and nothing in the fixture or "+
					"in core's composite tables puts it there: ARIA owns nothing "+
					"with this role, no composite claims it, and it carries no "+
					"value range. Either the kind is wrong or the fixture is stale",
					role, entry.Kind)
			}
			continue
		}
		if entry.Kind != want {
			t.Errorf("notTappable[%q].Kind is %q and the specification says %q — "+
				"a reason of the wrong kind reads perfectly and is what the next "+
				"person copies", role, entry.Kind, want)
		}
	}

	// The derivation must reach every derived kind, or a rule that answered ""
	// for everything would pass the loop above by never disagreeing.
	seen := map[roleKind]bool{}
	for _, entry := range notTappable {
		seen[entry.Kind] = true
	}
	for kind := range derivedKinds {
		if !seen[kind] {
			t.Errorf("no role in the table is filed as %q, so that derivation is "+
				"never exercised and could answer anything", kind)
		}
	}
}

// A tappable container is none of the four derivable kinds.
//
// The other direction of the same rule, and the one that would catch the
// mistake the census exists for pointing the wrong way: a role added to
// core.TappableContainerRoles() that ARIA actually owns children with, or that
// a composite claims as a member, is a container being offered to a toolbar's
// walk as one of its controls while something else already owns its tab stop.
func TestNoTappableContainerIsSomethingElsesMember(t *testing.T) {
	spec := loadSpec(t)
	for _, r := range core.TappableContainerRoles() {
		for _, c := range core.KeyboardComposites() {
			if m, _ := core.CompositeMemberRole(c); m == r {
				t.Errorf("core.TappableContainerRoles() names %q, and it is the "+
					"member role of %q — a toolbar walking onto one would be the "+
					"second widget writing a tabindex on it", r, c)
			}
			if c == r {
				t.Errorf("core.TappableContainerRoles() names %q, which owns a "+
					"keyboard of its own", r)
			}
		}
		if owned := spec.Roles[string(r)].RequiredOwned; len(owned) > 0 {
			t.Errorf("core.TappableContainerRoles() names %q, and ARIA requires it "+
				"to own %s — a container with required children is an arrangement, "+
				"and a tap handler on one is a shortcut rather than its semantics",
				r, strings.Join(owned, ", "))
		}
	}
}

// hasAttribute reports whether the fixture gives a role the named ARIA
// attribute. Sorted lookup is not worth it for lists this short; what matters
// is that the question is asked of the generated fixture rather than of a
// second list written here.
func hasAttribute(def roleSpec, attr string) bool {
	i := sort.SearchStrings(def.Attributes, attr)
	if i < len(def.Attributes) && def.Attributes[i] == attr {
		return true
	}
	// The fixture's attribute lists are not required to be sorted, so the
	// binary search above is an optimisation and this is the answer.
	for _, a := range def.Attributes {
		if a == attr {
			return true
		}
	}
	return false
}
