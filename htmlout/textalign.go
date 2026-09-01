package htmlout

import "github.com/rohanthewiz/grmob/core"

// textAligns is the one authoritative statement of core.Alignment -> the CSS
// text-align table, the fourth of the mappings the two DOM renderers would
// otherwise each keep their own copy of:
//
//	htmlout (this package)      queries it through textAlignDecl
//	wasm/grmob-runtime.js       restates it in JavaScript (textAlignFor)
//
// The runtime's copy is pinned to this one by TestRuntimeTextAlignsMatchGo in
// wasm/verify, the same way the tag, <input> type and object-fit tables are,
// and this table's coverage is held to core.TextAlignments() by
// TestTextAlignsCoverEveryTextAlignment.
//
// # What this table changed
//
// It was written because Align was the worst-behaved prop in the framework:
// one value, four behaviors. Before it existed, htmlout carried this mapping
// as an inline switch in styleValue with three arms and no justify case, so
// core.Align(core.AlignJustify) exported no declaration at all; the WASM
// runtime did not read Style.Align in *any* form, so every Align on the web
// target was silently dropped; Renderer.swift fell through to .leading; and
// Renderer.kt was the single target that actually justified the text. Nothing
// could notice, because there was no list for anything to be checked against.
//
// # Logical keywords, not physical ones
//
// Start maps to CSS's direction-aware "start" and End to "end", not to the
// physical "left"/"right" this exporter originally emitted. The physical
// spellings were a divergence: both natives use the direction-aware one
// (SwiftUI .leading/.trailing, Compose TextAlign .Start/.End), so an RTL
// locale would left-align on the web and trailing-align on iOS and Android
// from the same core.AlignStart. In an LTR document the two spellings render
// identically, which is why the divergence survived as long as it did — and
// why no test can see this choice: the pin below compares this table with the
// runtime's copy, and they agree with each other under either spelling. The
// direction-aware keywords are what keep all four renderers answering
// core.AlignStart the same way, so a reversion to "left"/"right" here must be
// treated as reopening that divergence, not as a cosmetic swap.
//
// # Two ways of saying nothing
//
// An absent or unrecognized alignment yields "". The two renderers express
// that differently and mean the same thing: htmlout emits no declaration at
// all (textAlignDecl below), and the runtime assigns "" to the property, which
// clears it.
//
// "Unrecognized" covers more than a typo here. AlignStretch and AlignBaseline
// are declared Alignments that name no text alignment — they exist for
// Style.Align's other role, as the cross-axis fallback the vertical-stacking
// containers read when AlignItems is unset (crossaxis.go, the table for that
// role) — so they are deliberately absent from this
// table and deliberately absent from core.TextAlignments(). A Text node given
// one falls back to the document's alignment, which is the same nothing the
// natives do with it.
var textAligns = map[core.Alignment]string{
	core.AlignStart:   "start",
	core.AlignCenter:  "center",
	core.AlignEnd:     "end",
	core.AlignJustify: "justify",
}

// TextAlignFor returns the CSS text-align value an alignment maps to, or ""
// for one that is absent, unrecognized, or a cross-axis-only Alignment.
//
// It takes a string rather than a core.Alignment for the reason ObjectFitFor,
// InputTypeFor and TagFor do: the callers that need it most are reading the
// value back out of a place that has already lost the Go type.
func TextAlignFor(align string) string {
	return textAligns[core.Alignment(align)]
}

// TextAligns returns a copy of the whole table, keyed by the alignment's
// string form, for the two callers that must enumerate it rather than query
// it: the WASM runtime conformance test, which compares table against table,
// and the census test that holds this table to core.TextAlignments().
//
// A copy, not the map itself, for the reason ObjectFits hands out one — both
// callers delete from what they are given as they match rows.
func TextAligns() map[string]string {
	out := make(map[string]string, len(textAligns))
	for k, v := range textAligns {
		out[string(k)] = v
	}
	return out
}

// textAlignDecl builds the whole declaration for styleValue's semicolon-joined
// list, and is the part of the mapping the runtime does not share: it assigns
// el.style.textAlign and needs the bare value.
//
// "" has to stay "" rather than become "text-align:", which is a malformed
// declaration — and a malformed declaration can invalidate the ones beside it
// in the same style attribute.
func textAlignDecl(align string) string {
	if value := TextAlignFor(align); value != "" {
		return "text-align:" + value
	}
	return ""
}
