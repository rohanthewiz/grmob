package htmlout

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The census: every core.TextAlignment must have a row.
//
// This is the same test ObjectFits gets, and it exists for the same reason —
// the failure is invisible on both sides. A missing row exports no declaration
// and, in the runtime, clears the property, so the text falls back to the
// document's alignment while both natives honor the value the author asked
// for. Nothing errors; the page is just wrong on two targets out of four.
//
// It is worth restating why this census can exist at all when the tag and
// <input> type tables get none: Alignment is a named type with declared
// constants and core enumerates them. Node types are still bare string
// literals at their construction sites, so those two tables have no list to be
// held to.
func TestTextAlignsCoverEveryTextAlignment(t *testing.T) {
	table := TextAligns()

	for _, align := range core.TextAlignments() {
		value, ok := table[string(align)]
		switch {
		case !ok:
			t.Errorf("%s has no text-align value; a Text using it would export nothing", align)
		case value == "":
			t.Errorf("%s maps to the empty value, which is how the table spells 'unrecognized'", align)
		}
		delete(table, string(align))
	}
	// The other direction: a row core cannot produce. Harmless in the output,
	// but the wasm conformance test would go on requiring the runtime to carry
	// the same dead row — and a reader would take it for a supported value.
	for align, value := range table {
		t.Errorf("table maps %q to %q, but core.TextAlignments() does not list it", align, value)
	}
}

// The two Alignments that are deliberately absent, asserted as absent rather
// than left to the census's silence.
//
// Style.Align has a second role — the cross-axis fallback the native
// containers read when AlignItems is unset — and these two values exist only
// for it. Without this test, someone "completing" the table by adding
// text-align:stretch (not a keyword) or text-align:baseline (not a keyword)
// would produce declarations the browser drops, and the census above would be
// happy: it only checks that every TextAlignment is present, and neither of
// these is one.
func TestTextAlignsOmitTheCrossAxisOnlyAlignments(t *testing.T) {
	for _, align := range []core.Alignment{core.AlignStretch, core.AlignBaseline} {
		if got := TextAlignFor(string(align)); got != "" {
			t.Errorf("TextAlignFor(%q) = %q; %s names a cross-axis placement, not a text "+
				"alignment, and CSS text-align has no such keyword", align, got, align)
		}
	}
}

// The declaration prefix is the one part of the mapping htmlout does not share
// with the runtime, so it is the one part the conformance test cannot see.
func TestTextAlignDeclPrefixesTheProperty(t *testing.T) {
	for align, want := range map[string]string{
		"start":   "text-align:left",
		"center":  "text-align:center",
		"end":     "text-align:right",
		"justify": "text-align:justify",
	} {
		if got := textAlignDecl(align); got != want {
			t.Errorf("textAlignDecl(%q) = %q, want %q", align, got, want)
		}
	}
}

// "" has to stay "" rather than become "text-align:", which is malformed — and
// one malformed declaration can invalidate the ones beside it in the same
// style attribute, so this would cost more than the alignment.
func TestTextAlignDeclDropsAnUnknownAlignment(t *testing.T) {
	for _, align := range []string{"", "stretch", "baseline", "middle", "left"} {
		if got := textAlignDecl(align); got != "" {
			t.Errorf("textAlignDecl(%q) = %q, want no declaration at all", align, got)
		}
	}
}

// End-to-end through the exporter, for the value that was silently dropped
// before this table existed. The unit tests above prove the table; this proves
// the table is the one styleValue reaches.
func TestJustifiedTextExports(t *testing.T) {
	node := &core.Node{
		Type:  "Text",
		Props: map[string]any{"content": "hello"},
		Style: &core.Style{Align: core.AlignJustify},
	}
	if html := ExportHTML(node); !strings.Contains(html, "text-align:justify") {
		t.Errorf("Align(AlignJustify) exported no text-align:justify:\n%s", html)
	}
}

// A copy, for the reason ObjectFits hands out one: both callers delete from
// what they are given as they match rows.
func TestTextAlignsReturnsACopy(t *testing.T) {
	first := TextAligns()
	delete(first, "center")
	first["start"] = "right"

	if got := TextAlignFor("center"); got != "center" {
		t.Errorf("deleting from the returned map changed TextAlignFor(center): %q", got)
	}
	if got := TextAlignFor("start"); got != "left" {
		t.Errorf("writing to the returned map changed TextAlignFor(start): %q", got)
	}
}
