package core

import "testing"

// The two exported role lists, held against each other.
//
// # Where the rest of this went
//
// This file used to carry the whole tappable-container census: a table naming
// every role that is *not* one and why, checked for totality against Roles().
// The table has moved to aria/verify, and the move is the point rather than a
// tidy-up. Its reasons were prose top to bottom, with a comment here arguing
// that a reason string cannot be checked and that what a census buys is the
// moment of writing one. Four of the seven kinds it sorted roles into turned
// out to be derivable — from ARIA's Required Owned Elements row, from
// KeyboardComposites(), from CompositeMemberRole() and from the attribute list
// that gives a role a value range — and deriving them needs the generated
// specification fixture, which lives in aria/verify and cannot be imported
// backwards into core.
//
// So the census is there, where a claim about ARIA has something to be wrong
// against, and what stays here is the half that has nothing to do with ARIA:
// the consistency of core's own two exported lists. A role in
// TappableContainerRoles() that Roles() does not declare is a value the WASM
// runtime is pinned to and the vocabulary has never heard of, and that is a
// core question with a core answer.
func TestEveryTappableContainerRoleIsDeclared(t *testing.T) {
	declared := map[Role]bool{}
	for _, r := range Roles() {
		declared[r] = true
	}
	for _, r := range TappableContainerRoles() {
		if !declared[r] {
			t.Errorf("TappableContainerRoles() names %q, which Roles() does not "+
				"declare: the WASM runtime's toolbar keyboard is pinned to this "+
				"list, so a role only it knows about is a membership rule for a "+
				"role no exporter writes", r)
		}
	}
	if len(TappableContainerRoles()) == 0 {
		t.Error("TappableContainerRoles() is empty — a toolbar's members are the " +
			"natively focusable tags plus the containers carrying one of these, " +
			"so an empty list leaves a ChipStrip with one tab stop and nothing " +
			"to arrow to")
	}
}

// CompositeMemberRole's second return is exactly KeyboardComposites().
//
// # The contract that used to be a sentence
//
// CompositeMemberRole returns "" for two different reasons: a toolbar has a
// keyboard and no member role for ARIA to name, and a RoleHeading has neither.
// Its doc used to say that callers separate the two by asking
// KeyboardComposites() first — which is a contract a doc comment can state and
// cannot enforce, and the one caller inside core got it right by never being
// handed a non-composite rather than by asking.
//
// The two are separated at the source now, and this is what keeps the second
// return honest. It is a claim about two functions that must not drift, and
// each direction fails differently:
//
//	a composite reporting false   CompositeWalkAt takes the "no walk at all"
//	                              arm for a container that has one, so the
//	                              audit stops reporting a nested pair it should
//	                              be reporting
//
//	a non-composite reporting     the trap comes back one function over: a
//	true                          caller filtering on the flag admits a role
//	                              with no keyboard, and every question it then
//	                              asks about that role's walk is answered
//	                              confidently and meaninglessly
func TestTheCompositeFlagIsExactlyTheKeyboardCompositeList(t *testing.T) {
	keyboard := map[Role]bool{}
	for _, r := range KeyboardComposites() {
		keyboard[r] = true
		if _, composite := CompositeMemberRole(r); !composite {
			t.Errorf("KeyboardComposites() names %q and CompositeMemberRole reports "+
				"it is not one. CompositeWalkStopsAt would put it in the arm meant "+
				"for a role with no walk at all", r)
		}
	}

	// Over every declared role, not just the composites: the failure worth
	// catching is a role that is not on the list and says it is, and only a
	// total sweep can see one.
	for _, r := range Roles() {
		member, composite := CompositeMemberRole(r)
		if composite && !keyboard[r] {
			t.Errorf("CompositeMemberRole(%q) reports a keyboard composite and "+
				"KeyboardComposites() does not list it. That list is what the WASM "+
				"runtime's roving tabindex is pinned to, so this role has a walk in "+
				"core's answers and none on any target", r)
		}
		if !composite && member != "" {
			t.Errorf("CompositeMemberRole(%q) names %q as its member role and reports "+
				"it is not a composite — a member role belongs to a container with a "+
				"keyboard, and nothing would ever consult this one", r, member)
		}
	}

	// And the empty-but-composite case exists, or the second return has no
	// subject and this whole test passes by the two answers never differing.
	empty := 0
	for _, r := range KeyboardComposites() {
		if member, _ := CompositeMemberRole(r); member == "" {
			empty++
		}
	}
	if empty == 0 {
		t.Errorf("every keyboard composite names a member role, so the two reasons " +
			"for an empty answer have collapsed into one and the second return is " +
			"checking nothing. RoleToolbar is the case: ARIA defines no `toolbaritem`, " +
			"which is why the runtime supplies a membership rule of its own")
	}
}

// A role with no keyboard gets no answer about a walk it does not have.
//
// CompositeWalkStopsAt returns true for one, and true is the safe answer rather
// than the true one — the true one is that the question does not apply. What
// this holds is that the arm exists and is reached for the reason it says: the
// two used to be a single `members == ""` test, so a RoleHeading was answered
// with a toolbar's reasoning.
func TestCompositeWalkStopsAtSeparatesNoWalkFromNoMemberRole(t *testing.T) {
	if _, composite := CompositeMemberRole(RoleHeading); composite {
		t.Fatalf("RoleHeading reports as a keyboard composite, so this test's " +
			"subject is gone")
	}
	if !CompositeWalkStopsAt(RoleHeading, RoleTabList) {
		t.Errorf("CompositeWalkStopsAt(RoleHeading, RoleTabList) says the walk " +
			"descends. A RoleHeading has no keyboard and therefore no walk")
	}
	// The toolbar arm, which is the other empty answer and a different fact:
	// it has a walk, and it stops because nothing says whose a control inside
	// a nested composite is.
	if member, composite := CompositeMemberRole(RoleToolbar); member != "" || !composite {
		t.Fatalf("RoleToolbar is (%q, %v), want (\"\", true)", member, composite)
	}
	if !CompositeWalkStopsAt(RoleToolbar, RoleTabList) {
		t.Errorf("CompositeWalkStopsAt(RoleToolbar, RoleTabList) says the walk " +
			"descends. A toolbar names no member role, so nothing says whose a " +
			"button buried in the strip is")
	}
	// And a pair that really does descend, so the two arms above are not
	// passing because everything stops.
	if CompositeWalkStopsAt(RoleListBox, RoleTabList) {
		t.Errorf("CompositeWalkStopsAt(RoleListBox, RoleTabList) says the walk " +
			"stops. An option below a tablist is still the listbox's option — the " +
			"roles say whose it is — so this is the descending case, and if nothing " +
			"descends the two arms above prove nothing")
	}
}

// The three answers are three values, and each is reached for its own reason.
//
// # What this replaces
//
// CompositeWalkStopsAt's two stopping arms used to be indistinguishable from
// outside: a non-composite outer and a toolbar both came back `true`. The test
// above this one could reach both arms and could not tell them apart, so
// swapping their bodies — the mutation that turns "there is no walk" into
// "the walk stops" — changed nothing any check could see. That is the whole
// case for CompositeWalkAt, and this is the check that mutation now fails.
//
// Every pair below is (outer, inner) and every one names why it lands where it
// does, because the values are cheap to assert and worthless asserted without
// the reason.
func TestCompositeWalkAtAnswersWithThreeDistinguishableValues(t *testing.T) {
	for _, c := range []struct {
		outer, inner Role
		want         CompositeWalk
		why          string
	}{
		{RoleHeading, RoleTabList, CompositeWalkNotApplicable,
			"a RoleHeading has no keyboard, so it has no member walk and neither " +
				"answer is true of it. This is the value that did not exist: the bool " +
				"spelling answered `true` here, which is a confident statement about " +
				"a rotation that does not exist"},
		{RoleToolbar, RoleTabList, CompositeWalkStops,
			"a toolbar has a walk and ARIA names no member role for it, so nothing " +
				"says whose a button buried in the strip is and the walk stops. The " +
				"same `true` as the row above, for an entirely different reason"},
		{RoleListBox, RoleListBox, CompositeWalkStops,
			"two listboxes share a member role, so descending would pool their " +
				"options and let one widget's arrows walk out into the other's rows"},
		{RoleListBox, RoleTabList, CompositeWalkDescends,
			"an option below a tablist is still the listbox's option — the roles say " +
				"whose it is — so the walk carries on through"},
		{RoleTabList, RoleToolbar, CompositeWalkDescends,
			"a tab inside a toolbar is still the tablist's tab, and a toolbar names " +
				"no member role to collide with the tablist's"},
	} {
		if got := CompositeWalkAt(c.outer, c.inner); got != c.want {
			t.Errorf("CompositeWalkAt(%q, %q) = %v, want %v — %s",
				c.outer, c.inner, got, c.want, c.why)
		}
	}

	// The three values must also be three: an enum whose members compared
	// equal would satisfy every row above and separate nothing.
	if CompositeWalkNotApplicable == CompositeWalkStops ||
		CompositeWalkStops == CompositeWalkDescends ||
		CompositeWalkNotApplicable == CompositeWalkDescends {
		t.Error("two CompositeWalk values are equal, so the answers this type exists " +
			"to separate are back to being one answer")
	}
	// And they print differently, because the audit's finding is built out of
	// the value and a reader who sees two of them spelled the same is back
	// where they started.
	seen := map[string]CompositeWalk{}
	for _, w := range []CompositeWalk{
		CompositeWalkNotApplicable, CompositeWalkStops, CompositeWalkDescends,
	} {
		if prior, dup := seen[w.String()]; dup {
			t.Errorf("CompositeWalk(%d) and CompositeWalk(%d) both print as %q",
				int(prior), int(w), w.String())
		}
		seen[w.String()] = w
	}
}

// CompositeWalkStopsAt is exactly "not descends", over every pair of declared
// roles.
//
// The bool is what the runtime's walks and the audit's own guard ask for, and
// it is kept — but it is now a reading of CompositeWalkAt rather than a second
// implementation, and this is what stops the two drifting into disagreement.
// A total sweep rather than the composites alone: the pairs where they could
// disagree are the ones where the outer role is not a composite at all, which
// is precisely what a check over KeyboardComposites() would never mount.
func TestTheBoolIsTheEnumWithTheThirdValueFoldedIn(t *testing.T) {
	pairs := 0
	for _, outer := range Roles() {
		for _, inner := range Roles() {
			pairs++
			want := CompositeWalkAt(outer, inner) != CompositeWalkDescends
			if got := CompositeWalkStopsAt(outer, inner); got != want {
				t.Errorf("CompositeWalkStopsAt(%q, %q) = %v and CompositeWalkAt says "+
					"%v. The bool is documented as the enum with the not-applicable "+
					"value folded into the stop, and a second implementation of the "+
					"rule is how the audit and the runtime's walks stop agreeing",
					outer, inner, got, CompositeWalkAt(outer, inner))
			}
		}
	}
	if pairs == 0 {
		t.Error("Roles() is empty, so the sweep above compared nothing")
	}
}
