package htmlout

import (
	"maps"
	"sort"

	"github.com/rohanthewiz/grmob/core"
)

// crossAxisAligns is the one authoritative statement of core.Alignment -> the
// CSS align-items table: the fifth of the mappings the two DOM renderers would
// otherwise each keep their own copy of, and the table that serves
// Style.Align's *second* role.
//
//	htmlout (this package)      queries it through CrossAxisAlignFor
//	wasm/grmob-runtime.js       restates it in JavaScript (crossAxisAlignFor)
//
// The runtime's copy is pinned to this one by TestRuntimeCrossAxisAlignsMatchGo
// in wasm/verify, the same way the tag, <input> type, object-fit and
// text-align tables are.
//
// # The second role
//
// Style.Align is the text alignment of a Text node — that role is
// textalign.go's table — and it is also the value a vertical-stacking
// container falls back to for cross-axis placement when AlignItems is unset.
// Both natives have read that fallback for as long as they have existed
// (crossAxisValue in Renderer.swift, the `alignItems.ifEmpty { align }` reads
// in Renderer.kt); the DOM pair read it for placement not at all. So
// core.Align(core.AlignCenter) on a Column centered the children on iOS and
// Android and only the *text* on the web, and core.Align(core.AlignStretch)
// filled rows on device while the web honored it only by accident, wherever
// block flow or the flex default happened to coincide. Same shape as every
// gap this family of tables closed: one value, different behaviors, no error
// anywhere.
//
// # Why the values are the AlignItems spellings
//
// Each row maps an Alignment onto the string the corresponding AlignItems
// constant already is — AlignStart onto "flex-start", not onto CSS's newer
// "start" — because both DOM renderers emit AlignItems verbatim, and the
// fallback means "behave as if that AlignItems had been set". One semantic,
// one CSS spelling, whichever prop stated it. That identity is what the
// census test holds the table to: its values must be exactly
// core.AlignItemsValues(), which keeps this table from quietly inventing a
// cross-axis vocabulary of its own.
//
// # The two rows that are deliberately missing
//
// AlignJustify names no cross-axis placement anywhere — no native dispatch
// has an arm for it and CSS align-items has no such keyword — and
// AlignBaseline, though CSS *could* honor it, falls through every native
// dispatch to start-packing. A row here would baseline-align on exactly two
// targets out of four, which is the disease this table exists to cure. Both
// map to "", which the callers read as "no fallback": the container is left
// exactly as it would be with Align unset.
var crossAxisAligns = map[core.Alignment]string{
	core.AlignStart:   string(core.AlignItemsStart),
	core.AlignCenter:  string(core.AlignItemsCenter),
	core.AlignEnd:     string(core.AlignItemsEnd),
	core.AlignStretch: string(core.AlignItemsStretch),
}

// CrossAxisAlignFor returns the CSS align-items value an alignment falls back
// to when AlignItems is unset, or "" for one that is absent, unrecognized, or
// a text-only Alignment.
//
// It takes a string rather than a core.Alignment for the reason TextAlignFor
// does: the callers that need it most are reading the value back out of a
// place that has already lost the Go type.
func CrossAxisAlignFor(align string) string {
	return crossAxisAligns[core.Alignment(align)]
}

// CrossAxisAligns returns a copy of the whole table, keyed by the alignment's
// string form, for the two callers that must enumerate it rather than query
// it: the WASM runtime conformance test and the census test.
//
// A copy, not the map itself, for the reason TextAligns hands out one — both
// callers delete from what they are given as they match rows.
func CrossAxisAligns() map[string]string {
	out := make(map[string]string, len(crossAxisAligns))
	for k, v := range crossAxisAligns {
		out[string(k)] = v
	}
	return out
}

// alignFallbackAxes says which node types read the fallback at all, by naming
// the flex axis each one stacks along. The set is exactly the containers the
// natives read it for — Column, Card (a Column whose Go theme style carries
// the card look, on every renderer) and List — and the value is *why* each is
// in the set: stacking along the column axis is what makes the cross axis
// horizontal, and the fallback has only ever applied to a horizontal cross
// axis. Row is absent on purpose, everywhere: Align is a text-alignment
// concept and has never been read for a Row's vertical axis, on any target
// (GrMobFlexStack consults it only when its axis is vertical, GrMobRow's
// dispatch reads alignItems alone, and the two DOM renderers now decline it
// through this table). Honoring it there would move existing rows.
//
// A type table rather than a "not Row" test because most node types are not
// containers at all. styleValue serializes every node through one function,
// and without the gate a Text or Button carrying Align — the prop's ordinary
// text role — would be turned into a flex container by its own alignment.
//
// The runtime restates this in alignFallbackAxisFor, pinned by
// TestRuntimeAlignFallbackAxesMatchGo — same arrangement as the value table
// above, because a gate that drifted would be the same silent disagreement:
// a type reading the fallback on one DOM target and not the other.
var alignFallbackAxes = map[string]string{
	"Column": "column",
	"Card":   "column",
	"List":   "column",
}

// AlignFallbackAxisFor returns the flex axis a node type stacks along, for
// the types that read the Align cross-axis fallback, and "" for every type
// that does not.
func AlignFallbackAxisFor(nodeType string) string {
	return alignFallbackAxes[nodeType]
}

// AlignFallbackAxes returns a copy of the gate table, for the WASM
// conformance test — table against table, like CrossAxisAligns.
func AlignFallbackAxes() map[string]string {
	out := make(map[string]string, len(alignFallbackAxes))
	maps.Copy(out, alignFallbackAxes)
	return out
}

// AlignFallbackTypes returns the node types that read the fallback, sorted so
// a test looping over them reports in a stable order. The behavioral tests
// range over this rather than a hand-written list, for the reason
// TransparentTypes exists: a list in a test is the untracked second copy this
// file removes.
func AlignFallbackTypes() []string {
	out := make([]string, 0, len(alignFallbackAxes))
	for t := range alignFallbackAxes {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
