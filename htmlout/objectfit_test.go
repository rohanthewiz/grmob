package htmlout

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The other two tables are keyed by node type, and node types are string
// literals at two dozen construction sites in core — there is no list to check
// coverage against, so a type nobody taught the renderers about just renders
// as a <div>. ContentMode is different: it is a named type with four declared
// constants and core.ContentModes enumerates them, so this table's coverage
// can be a test instead of a hope.
//
// It is worth having because the failure is invisible on both sides. A mode
// missing from the table exports no declaration and, in the runtime, clears
// the property — so a new ContentMode would render as the browser default on
// both DOM targets while the natives honored it, and nothing would say so.
func TestObjectFitsCoversEveryContentMode(t *testing.T) {
	table := ObjectFits()

	for _, mode := range core.ContentModes() {
		fit, ok := table[string(mode)]
		switch {
		case !ok:
			t.Errorf("%s has no object-fit value; an Image using it would export nothing", mode)
		case fit == "":
			t.Errorf("%s maps to the empty value, which is how the table spells 'unrecognized'", mode)
		}
		delete(table, string(mode))
	}
	// The other direction: a value core can no longer produce. Harmless in the
	// output, but the wasm conformance test would go on requiring the runtime
	// to carry the same dead row.
	for mode, fit := range table {
		t.Errorf("table maps %q to %q, but no core.ContentMode has that value", mode, fit)
	}
}

// The declaration prefix is the one part of the mapping htmlout does not
// share with the runtime, so it is the one part the conformance test cannot
// see. Checked here, once per mode, against values written out rather than
// read from the table — TestImageContentModeBecomesObjectFit in export_test.go
// is the same assertion made end-to-end through ExportHTML.
func TestObjectFitDeclPrefixesTheProperty(t *testing.T) {
	for mode, want := range map[string]string{
		"fit":     "object-fit:contain",
		"fill":    "object-fit:cover",
		"stretch": "object-fit:fill",
		"center":  "object-fit:none",
	} {
		if got := objectFitDecl(mode); got != want {
			t.Errorf("objectFitDecl(%q) = %q, want %q", mode, got, want)
		}
	}
}

// "" has to stay "" rather than become "object-fit:", which is a malformed
// declaration — and a malformed declaration can invalidate the ones beside it
// in the same style attribute.
func TestObjectFitDeclDropsAnUnknownMode(t *testing.T) {
	for _, mode := range []string{"", "sideways", "contain"} {
		if got := objectFitDecl(mode); got != "" {
			t.Errorf("objectFitDecl(%q) = %q, want no declaration at all", mode, got)
		}
	}
}

// A copy, for the reason InputTypes and Tags hand out one: the conformance
// test deletes from what it is given as it matches rows, and so does the
// census above.
func TestObjectFitsReturnsACopy(t *testing.T) {
	first := ObjectFits()
	delete(first, "fit")
	first["fill"] = "contain"

	if got := ObjectFitFor("fit"); got != "contain" {
		t.Errorf("deleting from the returned map changed ObjectFitFor(fit): %q", got)
	}
	if got := ObjectFitFor("fill"); got != "cover" {
		t.Errorf("writing to the returned map changed ObjectFitFor(fill): %q", got)
	}
}
