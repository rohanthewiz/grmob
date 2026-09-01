package main

import (
	"testing"

	"github.com/rohanthewiz/grmob/htmlout"
)

// core.ContentMode -> CSS object-fit, the third of the tables the runtime
// restates in JavaScript. Same arrangement as the tag and <input> type tables:
// Go holds the authority (htmlout/objectfit.go), the runtime holds a copy
// because it is the side assigning the property, and this keeps them equal
// under a plain `go test ./...`. See jstable_test.go for the parse.
//
// Unlike those two, the runtime's copy needed no reshaping to be readable by
// the shared parser — it was already a flat literal with an "" fallback, which
// is where that shape came from in the first place.
//
// What a mismatch costs, in either direction, is an image that scales one way
// in the exported HTML and another in the live app: a "cover" that crops
// against a "contain" that letterboxes, with no error anywhere. The mode a
// caller asked for is honored; it is simply honored differently.
func TestRuntimeObjectFitsMatchGo(t *testing.T) {
	table := parseRuntimeTable(t, runtimeSource(t), "objectFitFor", "")

	want := htmlout.ObjectFits()
	for mode, jsFit := range table {
		if goFit := want[mode]; goFit != jsFit {
			t.Errorf("%s: runtime says %q, htmlout says %q", mode, jsFit, goFit)
		}
		delete(want, mode)
	}
	// Anything left is a mode Go scales and the runtime does not, which is the
	// direction that silently falls back to the browser's default.
	for mode, goFit := range want {
		t.Errorf("%s: htmlout says %q, runtime has no entry", mode, goFit)
	}
}
