// Package verify holds one machine-readable statement of the ARIA facts this
// framework depends on, and the tests that hold every restatement of them to
// it.
//
// # What went wrong without it
//
// Every ARIA claim in this repository was hand-checked prose. There are dozens:
// three role lists in core/style.go, two more in each web exporter, a
// name-prohibition argument in core.RoleGroup, four aria-level scopes, six
// aria-selected roles, an aria-orientation default per composite, and a
// keyboard contract per composite role. They agreed with each other because
// somebody read them all, which is not a mechanism.
//
// The failure it produced was small and instructive: a Next-list entry once
// asserted that `group` supports aria-expanded. It survived three re-sorts and
// was false, and nothing in a test suite of several thousand assertions could
// have said so, because the claim lived in prose and the code that would have
// contradicted it was a switch statement in a different package.
//
// # What this is
//
// testdata/aria.json is the single statement. Each role names the attributes
// ARIA defines or inherits for it, the orientation it defaults to when it has
// one, and the children its role requires. The tests then check that:
//
//   - every core.Role is a role ARIA has
//   - each of the four state guards in the two web exporters accepts exactly
//     the roles the fixture says support that attribute, and refuses the rest
//   - htmlout.AriaOrientationDefaults holds ARIA's own per-role defaults, for
//     exactly the roles ARIA scopes the attribute to
//   - core.RoleGroup is nameable and `generic` is not, which is the entire
//     premise of the fallback both exporters supply
//   - the WASM runtime's composite table names containers whose required
//     children are the member roles it looks for
//
// # Where it comes from
//
// It is generated, from the specification's own machine-readable role
// definitions:
//
//	sh aria/fetch.sh      # the 1.4MB published HTML, not committed
//	go run ./aria/gen     # -> testdata/aria.json
//
// It was hand-transcribed for two sessions, and this doc said at the time that
// transcribing by hand was a weaker thing than reading the specification —
// because a transcription can be wrong in exactly the way the prose it replaces
// could be wrong. It was wrong, in four places, and generating it for the first
// time is what said so:
//
//	list.requiredOwned held `group`      a listbox's allowance, not a list's
//	radiogroup.orientation held a value  ARIA 1.1 had one, 1.2 removed it
//	treegrid.orientation held a value    no version of ARIA states one
//	nameProhibited held `term`, `time`   both take an author name in 1.2
//
// All four were in near-miss rows — the entries that exist so a guard has
// something to argue with, and are therefore the entries least likely to be
// argued with. Nothing failed when they were corrected, which is the finding:
// the fixture's weak rows are exactly the ones no test reaches, so checking them
// by hand was never going to be the fix.
//
// # What still is not generated
//
// Two lists in aria/spec, and the division between them and the facts is the
// thing to hold on to. What ARIA *says* is read from ARIA. What this framework
// *cares about* is chosen here:
//
//	spec.InScopeAttributes   the aria-* attributes some writer in this
//	                         repository can emit. Nine of them. A tenth would
//	                         widen every generated entry and add no guard.
//	spec.NearMisses          the roles core deliberately does not carry, kept
//	                         so `cell is not gridcell` is an assertion rather
//	                         than a remark.
//
// Both are selections rather than claims, so neither can be wrong the way a
// transcribed fact can. The roles in scope come from core.Roles() plus that
// second list, so a role added to the vocabulary is in the fixture the next time
// anyone generates it.
//
// # The offline promise is intact
//
// Nothing on a verification path fetches anything. `go test ./...`, run.sh and
// the three platform harnesses all read the committed fixture. The conformance
// test that holds the fixture to the specification skips when no download is
// present — the stance ios/verify takes toward a missing iPhoneOS SDK — so the
// check exists, costs nothing, and runs for anyone who wants it.
//
// It is also deliberately partial. Only the attributes this framework can write
// are listed — the levels, the two selection spellings, the disclosure, the
// orientation and the value family — because a fixture is only worth what is
// checked against it, and an unchecked entry is prose again with braces around
// it. Roles core does not carry are present for the same reason the exporters'
// comments name them: `gridcell`, `meter`, `slider` and `treeitem` are the near
// misses every guard has to argue with, and having them here turns "cell is not
// gridcell" from a remark into an assertion.
package verify
