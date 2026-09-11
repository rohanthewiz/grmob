package main

import (
	"testing"

	"github.com/rohanthewiz/grmob/htmlout"
)

// The node type -> its own ARIA role table, the third the runtime restates in
// JavaScript. Same arrangement as the tag and <input> type tables next door:
// Go holds the authority (ownRoles in htmlout/tag.go), the runtime holds a copy
// because it is the side writing the attribute onto a live element, and this
// test keeps them equal under a plain `go test ./...`. See jstable_test.go for
// the parse.
//
// What the table holds are the roles this framework writes from what a node
// *is* rather than from a core.Role: `dialog` for a Modal, whose
// core.ModalNode has no Style for a role to ride on, and `switch` for a
// core.Switch, which the DOM has no element for. Neither is a value a caller
// can spell, which is exactly why the two copies need holding together — there
// is no Role constant whose absence would be noticed.
//
// The two directions of a mismatch fail differently, and both are silences:
//
//	runtime has a row Go lacks   → the live app announces a control the static
//	                               export does not, so the two web targets
//	                               describe different screens
//	Go has a row the runtime lacks → the runtime writes no role, and the
//	                               control announces itself as whatever its tag
//	                               implies: a Switch read out as a checkbox, a
//	                               Modal as an unnamed group
func TestRuntimeOwnRolesMatchGo(t *testing.T) {
	table := parseRuntimeTable(t, runtimeSource(t), "ownRole", "")

	for nodeType, jsRole := range table {
		if goRole := htmlout.OwnRoleFor(nodeType); goRole != jsRole {
			if goRole == "" {
				t.Errorf("%s: runtime writes role=%q, htmlout writes none", nodeType, jsRole)
				continue
			}
			t.Errorf("%s: runtime says role=%q, htmlout says role=%q", nodeType, jsRole, goRole)
		}
	}

	// The other direction. There is no exported list of the Go table's keys —
	// OwnRoleFor answers one at a time — so the node types it could answer for
	// are the ones the tag table knows, which is every type either DOM
	// renderer can be handed.
	for nodeType := range htmlout.Tags() {
		goRole := htmlout.OwnRoleFor(nodeType)
		if goRole == "" {
			continue
		}
		if _, ok := table[nodeType]; !ok {
			t.Errorf("%s: htmlout writes role=%q, runtime has no row — the control would "+
				"announce itself as whatever its tag implies", nodeType, goRole)
		}
	}

	// Both halves above are censuses over tables, which makes them exact and
	// leaves one thing unasserted: that the framework's two self-roling types
	// are in fact these two. Named here, so a row deleted from BOTH tables at
	// once — the one edit the comparison cannot see — fails rather than
	// agreeing with itself.
	for _, nodeType := range []string{"Modal", "Switch"} {
		if !htmlout.CarriesOwnRole(nodeType) {
			t.Errorf("htmlout.CarriesOwnRole(%s) = false; the TabView wiring would write "+
				"role=\"tabpanel\" on top of the role this type already has", nodeType)
		}
		if _, ok := table[nodeType]; !ok {
			t.Errorf("grmob-runtime.js: ownRole has no %s row", nodeType)
		}
	}
}
