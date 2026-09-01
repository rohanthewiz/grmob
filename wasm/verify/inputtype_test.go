package main

import (
	"testing"

	"github.com/rohanthewiz/grmob/htmlout"
)

// The node type -> <input> type table exists twice by necessity: once in Go
// (htmlout.InputTypeFor, the authority) and once in JavaScript, because the
// WASM runtime sets the attribute in the browser and cannot call into Go to
// ask. This test reads the runtime's copy out of its source and compares it
// against the Go one. See jstable_test.go for the parse and why it is a parse.

func TestRuntimeInputTypesMatchGo(t *testing.T) {
	table := parseRuntimeTable(t, runtimeSource(t), "inputTypeFor", "")

	want := htmlout.InputTypes()
	for nodeType, jsType := range table {
		if goType := want[nodeType]; goType != jsType {
			t.Errorf("%s: runtime says %q, htmlout says %q", nodeType, jsType, goType)
		}
		delete(want, nodeType)
	}
	// Anything left is a type Go maps onto an <input> and the runtime does
	// not, which is the direction that silently draws the wrong control.
	for nodeType, goType := range want {
		t.Errorf("%s: htmlout says %q, runtime has no entry", nodeType, goType)
	}
}

// The unknown-type answer is the other half of the contract and lives in the
// fallback rather than the table. The runtime's half is checked by the parse
// above, which requires the literal to be followed by `[type] || "";` — this
// is Go's half, so that the two ends of the same rule are asserted rather than
// one being assumed from the other.
//
// "" is what leaves a <textarea> and a <span> with no type attribute at all:
// createElement only sets one when the lookup is truthy, and htmlout only
// emits one where it writes the attribute by name.
func TestInputTypeForUnlistedTypeIsEmpty(t *testing.T) {
	if got := htmlout.InputTypeFor("TextArea"); got != "" {
		t.Fatalf("htmlout.InputTypeFor(TextArea) = %q, want the empty default", got)
	}
}
