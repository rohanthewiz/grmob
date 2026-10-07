// Package docsnippets holds no code. Its test compiles and runs the Go
// fences that the component documentation presents as whole, working code:
// the Tally widget in the README, the Tally / Spoiler / test-harness blocks on
// docs/concepts/components.md, and the same three in
// ai_docs/SKILL-component.md, which says its widget "compiles as written
// against the current API".
//
// Those blocks are not quotations of a source file. The existing excerpt
// tests (examples/counter, examples/todoapp) trace a fence back to the
// package it quotes, and that check cannot apply here because the docs are
// the only copy. So the test does what a reader does: it pastes each block
// into a package, builds it, and runs the tests that ship with it. See
// docsnippets_test.go for which fences are selected and how they are
// assembled.
package docsnippets
