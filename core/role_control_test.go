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
