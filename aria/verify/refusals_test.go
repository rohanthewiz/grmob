package verify

import (
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The composite patterns core.Role deliberately does not carry, and what each
// refusal is actually blocked on.
//
// # Why a table
//
// "core has no vocabulary for menu, tree and grid" has been a sentence in this
// repository for three sessions, and it has been in four places: core/role.go's
// header, RoleListBox's doc, htmlout/orientation.go's account of the roles it
// leaves out, and the comment above the runtime's COMPOSITE_MEMBERS. The four
// agreed because somebody read them all, which is the same non-mechanism the
// ARIA fixture next door exists to replace — and it was not entirely true. The
// blockers are not one blocker, and two of them are already gone.
//
// # The two halves of a refusal
//
// Every one of these patterns needs two things this framework does not have,
// and they fail independently:
//
//	the member role      core.Role has to be able to say what the container's
//	                     members ARE. A `tree` of divs is a tree with no
//	                     treeitems, which role.go's structural rule calls worse
//	                     than no role at all: role="tree" claims the things
//	                     inside it are tree items, and a reader believes it.
//	the walk             the runtime has to be able to find them and move a tab
//	                     stop between them. That machinery is one-dimensional
//	                     and flat, so a grid's two axes and a tree's recursion
//	                     are shapes it has no form of.
//
// Members below is the first half and Shape is the second, and the test holds
// each row to the first: a member role that appears in core.Roles() means the
// vocabulary half of that refusal is over, and the row has to be re-argued
// rather than left standing.
//
// # The worked example is toolbar, which is no longer here
//
// `toolbar` sat in exactly this position for two releases — announced with an
// axis, given no keyboard, with a comment saying ARIA does not name a toolbar's
// members. Both halves turned out to be false in an interesting way: the
// missing member role was not missing, it was *never going to exist*, because a
// toolbar's members are whatever controls are in it. Once that was the reading,
// the walk was twenty lines (focusableMembers) and the pattern was one Set.
//
// So a row here is a claim that can end, and the way it ends may be that the
// question was wrong. See wasm/grmob-runtime.js, COMPOSITE_FOCUSABLE.
type refusal struct {
	// Role is the ARIA container role core does not carry.
	Role string
	// Members are the roles the pattern's arrow keys move between, as ARIA
	// names them. The vocabulary core would have to grow first.
	Members []string
	// Blocked is which half is still the *first* thing in the way:
	// "vocabulary" when core carries none of the member roles yet, "walk" when
	// it carries them all and only the runtime is missing.
	//
	// Declared rather than derived, and then checked against the derivation.
	// That is the whole mechanism: the value is computable from core.Roles(),
	// so writing it down turns "somebody will notice when this changes" into a
	// failing test that names the row.
	Blocked string
	// Nesting is where ARIA puts the pattern's members relative to its
	// container, in the one place the specification says so mechanically —
	// the Required Owned Elements row:
	//
	//	"direct"    every member role is one the container itself owns.
	//	            listbox owns option, tablist owns tab, and the runtime's
	//	            walk is a descent looking for that one role.
	//	"through"   a member is reachable only via an intermediate owned role
	//	            that owns things itself. `grid` owns `row`, and a
	//	            `gridcell` is a row's, so the members are laid out on two
	//	            axes rather than in a run.
	//
	// Declared and then derived, which is the mechanism Blocked uses and the
	// reason this field exists at all: it is the half of Shape below that has
	// an authority. "A grid's arrows move by row and by column" was a sentence
	// nothing could contradict; "gridcell is not in grid's requiredOwned and is
	// reachable only through row" is the same claim, checked against the
	// fixture.
	//
	// `group` is owned by half these containers and is not a level: ARIA uses
	// it as a pure wrapper (it requires no owned elements of its own), so a
	// walk descends through one and finds the same members. That falls out of
	// the derivation rather than needing an exception — the question asked is
	// whether the *member* is directly owned, and a wrapper beside it in the
	// list does not change the answer.
	Nesting string
	// Shape is the rest of what the runtime's member walk would need beyond
	// what it does for a listbox — the part with no authority — or "" when a
	// one-dimensional walk over a flat member list is the whole of it.
	//
	// Prose, deliberately and with the cost stated: a submenu with its own
	// Escape, and a tree's expand-in-place arrows, are facts about ARIA's
	// *authoring practices* rather than about its role definitions, and the
	// practices are not machine-readable in any form this repository fetches.
	// So nothing here can contradict these sentences, and the honest thing is
	// to say which field is checked and which is not rather than to leave them
	// looking alike. Nesting above is what was extractable; this is what was
	// left.
	//
	// Stated for every row, including the ones blocked on vocabulary first,
	// because it is the blocker that outlives the other: adding a member role
	// is one constant, and adding a second dimension to the walk is not.
	Shape string
	// Why is the argument, in the form it is worth reading in a failure.
	Why string
}

var refusals = []refusal{
	{
		Role:    "menu",
		Members: []string{"menuitem", "menuitemcheckbox", "menuitemradio"},
		Blocked: "vocabulary",
		Nesting: "direct",
		Shape:   "submenus, which are menus inside menus with their own tab stop and their own Escape",
		Why: "no widget here is a menu. core.Select's picker is a native <select> " +
			"on the web and a platform picker on both natives, so the menu is drawn " +
			"by something that is not this framework — which is also why " +
			"SelectOption.GroupDisabled is spelled as an optgroup rather than as a " +
			"menu section",
	},
	{
		Role:    "menubar",
		Members: []string{"menuitem", "menuitemcheckbox", "menuitemradio"},
		Blocked: "vocabulary",
		Nesting: "direct",
		Shape:   "a menubar opens menus, so it needs everything menu needs and a second axis besides",
		Why:     "the same absence as menu, one level up",
	},
	{
		Role:    "tree",
		Members: []string{"treeitem"},
		Blocked: "vocabulary",
		Nesting: "direct",
		Shape: "recursion, and an expansion state per node — a tree's arrows " +
			"collapse and expand as well as move",
		Why: "components.disclosure is the heading-around-button shape a twisty " +
			"needs and Collapse is the caller-owned expansion state a node would " +
			"want, so the pieces exist and nothing assembles them. What ends this " +
			"is a widget, not a constant: `treeitem` is also the only role that " +
			"carries aria-level and aria-selected together, which is the pairing " +
			"components.ListRow.Selectable has to refuse today",
	},
	{
		Role:    "treegrid",
		Members: []string{"row"},
		Blocked: "walk",
		Nesting: "direct",
		Shape:   "a grid's two axes and a tree's recursion at once",
		Why: "the one refused pattern whose vocabulary is already complete, which " +
			"is not something anyone would have guessed from the prose. ARIA's " +
			"treegrid moves between `row`s, core carries RoleRow, and a row takes " +
			"aria-level and aria-expanded — which core spells as " +
			"AccessibilityNestingLevel and AccessibilityExpanded, both already " +
			"scoped to that role. So nothing is missing but the walk, and the walk " +
			"is the hardest one on this list",
	},
	{
		Role:    "grid",
		Members: []string{"gridcell"},
		Blocked: "vocabulary",
		Nesting: "through",
		Shape: "two dimensions. A grid's arrows move by row and by column, and " +
			"Home/End mean the ends of a row rather than of a list",
		Why: "components.Calendar is the widget that would be one — forty-two " +
			"tappable day cells laid out in six rows of seven — and it is built as " +
			"a run of role=button toggle cells instead, with the argument written " +
			"at its cell builder. The vocabulary is closer than it looks: core " +
			"already carries row, cell and columnheader, so a grid needs one " +
			"container role and one member role rather than five. What it does not " +
			"have is the walk",
	},
	{
		Role:    "radiogroup",
		Members: []string{"radio"},
		Blocked: "vocabulary",
		Nesting: "direct",
		Shape:   "",
		Why: "the one pattern here whose walk this machinery could already do — " +
			"a flat run of members, one selected, arrows between them, which is a " +
			"listbox with a different word. It is absent because nothing builds a " +
			"radio group: components.SegmentedControl is the one-of-N control this " +
			"framework has, and it is drawn as a joined strip rather than as a " +
			"column of radios. Adding it means core.RoleRadio and " +
			"core.RoleRadioGroup and a widget that writes them",
	},
}

// Every refusal is a refusal of something real, and is still in force.
func TestTheRefusedPatternsAreRolesARIAActuallyHas(t *testing.T) {
	spec := loadSpec(t)
	carried := map[string]bool{}
	for _, r := range core.Roles() {
		carried[string(r)] = true
	}

	for _, r := range refusals {
		if _, ok := spec.Roles[r.Role]; !ok {
			t.Errorf("refusals names %q, which ARIA does not define — a refusal has to "+
				"be a refusal of something", r.Role)
			continue
		}
		if carried[r.Role] {
			t.Errorf("core.Roles() carries %q and refusals still says it does not. "+
				"Either the row is stale or the constant was added without the "+
				"pattern behind it, which is the failure core/role.go's header calls "+
				"out: a role with no arm falls into a catch-all and is silently inert",
				r.Role)
		}
	}
}

// Which half of each refusal is still the first thing in the way, derived from
// core.Roles() and compared to what the row claims.
//
// This is the assertion that makes the table a mechanism rather than a comment.
// The value is computable, so a row that states it is a row that fails when the
// computation changes — and the changes that matter are exactly the quiet ones:
//
//	a member role arrives    somebody adds core.RoleTreeItem for its aria-level
//	                         and aria-selected pairing, which is a reason with
//	                         nothing to do with trees. The `tree` row flips from
//	                         vocabulary to walk and this fails, which is the
//	                         only moment anyone is going to re-read the argument.
//	a member role leaves     a role removed from core.Roles() silently restores
//	                         a blocker somebody had already worked past.
//
// The failure message carries the row's own Shape, because that is what is left
// of the refusal once the vocabulary half is gone and it is the thing the reader
// now has to decide about.
func TestEachRefusalKnowsWhichHalfIsStillInTheWay(t *testing.T) {
	spec := loadSpec(t)
	carried := map[string]bool{}
	for _, r := range core.Roles() {
		carried[string(r)] = true
	}

	for _, r := range refusals {
		if r.Blocked != "vocabulary" && r.Blocked != "walk" {
			t.Errorf("refusals[%q].Blocked = %q, want \"vocabulary\" or \"walk\"",
				r.Role, r.Blocked)
			continue
		}
		if len(r.Members) == 0 {
			t.Errorf("refusals[%q] names no member roles — a composite pattern moves "+
				"between something, and a row that does not say what cannot be "+
				"checked against anything", r.Role)
			continue
		}

		var absent []string
		for _, m := range r.Members {
			if _, ok := spec.Roles[m]; !ok {
				t.Errorf("refusals[%q] names member role %q, which the fixture has no "+
					"entry for — add it to aria/spec.NearMisses and regenerate "+
					"(go run ./aria/gen)", r.Role, m)
				continue
			}
			if !carried[m] {
				absent = append(absent, m)
			}
		}

		want := "walk"
		if len(absent) > 0 {
			want = "vocabulary"
		}
		if r.Blocked == want {
			continue
		}
		if want == "walk" {
			t.Errorf("refusals[%q] says it is blocked on vocabulary, and core.Roles() "+
				"now carries every member role it needs (%s).\n\nWhat is left of the "+
				"refusal is: %s.\n\nRe-argue the row from that alone, or close it and "+
				"give %s a keyboard.",
				r.Role, strings.Join(r.Members, ", "), shapeOrNothing(r), r.Role)
			continue
		}
		t.Errorf("refusals[%q] says it is blocked on the walk, and core.Roles() does "+
			"not carry %s.\n\nA blocker came back: either a role was removed from the "+
			"vocabulary, or the row named the wrong members.",
			r.Role, strings.Join(absent, ", "))
	}
}

// Where ARIA puts each pattern's members, derived from the fixture and
// compared to what the row claims.
//
// # What this closes
//
// Blocked was derived and checked; Shape was not, and sat in the same struct
// looking like it was. "Two dimensions", "a submenu with its own Escape" — a
// reader has no way to tell which of the two fields is load-bearing, and the
// one that is not is exactly the shape this whole directory exists to stop
// producing: an entry no guard argues with is prose again with braces around
// it.
//
// There is no authority for most of Shape and there is one for a piece of it.
// ARIA's Required Owned Elements row says, mechanically, whether a pattern's
// members are the container's own or somebody else's, and that is the whole of
// the difference between a walk this runtime already does and the one it does
// not have:
//
//	listbox  owns option                    a descent looking for one role
//	tablist  owns tab                       the same descent
//	grid     owns row, and a row owns        a walk over two axes, which is
//	         gridcell                        the thing that is missing
//
// So the two-dimensionality claim is now derived from the same fixture every
// other check here rests on, and what is left in Shape is labelled as what it
// is.
//
// # Why the reachability half is asserted too
//
// A row claiming "through" has to be *reachable* through something, or the
// claim is that the member belongs to a pattern it has nothing to do with. The
// closure is one hop deep on purpose: two hops would let any role reach any
// other through `row` and `group` and answer yes for everything, which is a
// check that cannot fail.
func TestEachRefusalsMembersSitWhereItSaysTheyDo(t *testing.T) {
	spec := loadSpec(t)

	for _, r := range refusals {
		if r.Nesting != "direct" && r.Nesting != "through" {
			t.Errorf("refusals[%q].Nesting = %q, want \"direct\" or \"through\"",
				r.Role, r.Nesting)
			continue
		}
		owned := map[string]bool{}
		for _, o := range spec.Roles[r.Role].RequiredOwned {
			owned[o] = true
		}
		if len(owned) == 0 {
			t.Errorf("the fixture gives %q no required owned elements at all, so "+
				"nothing here can be derived. A composite whose members ARIA does "+
				"not name is a real thing — `toolbar` is one — and it is not a "+
				"refusal, it is a pattern with a different membership rule",
				r.Role)
			continue
		}

		for _, m := range r.Members {
			if owned[m] {
				if r.Nesting == "through" {
					t.Errorf("refusals[%q] says its members sit through an "+
						"intermediate role, and ARIA has %q owning %q directly. A "+
						"direct member is the flat run this runtime's walk already "+
						"does, so that part of the refusal is over.",
						r.Role, r.Role, m)
				}
				continue
			}
			if r.Nesting == "direct" {
				t.Errorf("refusals[%q] says %q is one of its own owned roles and "+
					"ARIA does not: the pattern's members are somebody else's, "+
					"which is a second axis rather than a longer list, and the row's "+
					"remaining blocker is bigger than it says.", r.Role, m)
				continue
			}
			// "through" — and it must actually go through something.
			reachable := false
			for owner := range owned {
				for _, deep := range spec.Roles[owner].RequiredOwned {
					if deep == m {
						reachable = true
					}
				}
			}
			if !reachable {
				t.Errorf("refusals[%q] says %q is reached through an intermediate "+
					"owned role, and none of %s owns it. Either the member is wrong "+
					"or the pattern is not this container's at all.",
					r.Role, m, strings.Join(spec.Roles[r.Role].RequiredOwned, ", "))
				continue
			}
		}
	}
}

// shapeOrNothing is the row's remaining blocker, in the words a failure needs.
func shapeOrNothing(r refusal) string {
	if r.Shape == "" {
		return "nothing — the one-dimensional walk this machinery already does would serve"
	}
	return r.Shape
}

// The refused patterns and the near misses are the same list, read two ways.
//
// aria/spec.NearMisses is what the *fixture* carries beyond core's vocabulary,
// and this file is what the *argument* is for six of those entries. A pattern
// argued about here and absent from the fixture would be an argument with no
// facts under it — every check above would report "ARIA does not define it",
// which is true of the fixture and false of ARIA, and is the least useful way
// to be told that a scope needs widening.
func TestEveryRefusedPatternIsInTheFixturesScope(t *testing.T) {
	spec := loadSpec(t)
	var missing []string
	for _, r := range refusals {
		if _, ok := spec.Roles[r.Role]; !ok {
			missing = append(missing, r.Role)
		}
		for _, m := range r.Members {
			if _, ok := spec.Roles[m]; !ok {
				missing = append(missing, m)
			}
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("refusals argues about %s, which the fixture has no entry for — add "+
			"them to aria/spec.NearMisses and regenerate (go run ./aria/gen)",
			strings.Join(dedup(missing), ", "))
	}
}

func dedup(in []string) []string {
	out := in[:0:0]
	for i, s := range in {
		if i == 0 || s != in[i-1] {
			out = append(out, s)
		}
	}
	return out
}
