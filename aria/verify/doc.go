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
// # What this is not
//
// It is not generated. The ARIA specification publishes its role definitions in
// a machine-readable form, and transcribing them by hand is a weaker thing than
// reading that file at build time — a transcription can be wrong in exactly the
// way the prose it replaces could be wrong.
//
// What it buys anyway, and the reason it is worth having in this form, is that
// the fact is now stated *once*. A wrong entry here is one wrong entry, and it
// fails a test the moment a guard disagrees with it; a wrong sentence in a doc
// comment was one of thirty restatements, and disagreed with nothing.
// Regenerating this file from the spec's own JSON is a strictly better later
// step and needs no change to anything that reads it.
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
