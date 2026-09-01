package main

import (
	"testing"

	"github.com/rohanthewiz/grmob/htmlout"
)

// The node type -> HTML tag table, the second of the two the runtime restates
// in JavaScript. Same arrangement as the <input> type table next door: Go
// holds the authority (htmlout/tag.go), the runtime holds a copy because it is
// the side calling createElement, and this test keeps them equal under a plain
// `go test ./...`. See jstable_test.go for the parse.
//
// The two directions of a mismatch fail differently, and neither is loud on
// its own at runtime:
//
//	runtime has a row Go lacks   → the export and the live app draw different
//	                               elements for the same node
//	Go has a row the runtime lacks → the runtime falls back to <div>, so a
//	                               Button stops being clickable and a TextArea
//	                               stops being editable, with no error anywhere
func TestRuntimeTagsMatchGo(t *testing.T) {
	table := parseRuntimeTable(t, runtimeSource(t), "tagForType", "div")

	// The transparent grouping nodes are exempt from the comparison because
	// the two renderers genuinely disagree there — htmlout emits no element,
	// the runtime must emit one to keep its positional patch addressing valid
	// (transparentTypes in htmlout/tag.go has the full reason). Exempt, but
	// not unchecked: the exemption is narrowed to the exact tag the runtime is
	// known to use, so this cannot become a hole a real drift hides in.
	//
	// If the runtime ever learns to address elementless nodes, these rows go
	// away and this loop fails until it is deleted — which is the right kind
	// of failure to get from a landmark.
	for _, nodeType := range htmlout.TransparentTypes() {
		tag, ok := table[nodeType]
		switch {
		case !ok:
			t.Errorf("%s: htmlout renders it transparently and the runtime has no row for it; "+
				"the runtime needs an explicit box (see transparentTypes)", nodeType)
		case tag != "div":
			t.Errorf("%s: runtime boxes it in <%s>; the known divergence is a <div>", nodeType, tag)
		}
		delete(table, nodeType)
	}

	want := htmlout.Tags()
	for nodeType, jsTag := range table {
		if goTag := want[nodeType]; goTag != jsTag {
			t.Errorf("%s: runtime says <%s>, htmlout says <%s>", nodeType, jsTag, goTag)
		}
		delete(want, nodeType)
	}
	for nodeType, goTag := range want {
		t.Errorf("%s: htmlout says <%s>, runtime has no row (it would fall back to <div>)", nodeType, goTag)
	}
}
