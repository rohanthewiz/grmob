package htmlout

import "github.com/rohanthewiz/grmob/core"

// objectFits is the one authoritative statement of core.ContentMode -> the CSS
// object-fit table, the third and last of the mappings the DOM renderers each
// used to keep their own copy of:
//
//	htmlout (this package)      queries it through ObjectFitFor
//	wasm/grmob-runtime.js       restates it in JavaScript (objectFitFor)
//
// The runtime's copy is pinned to this one by TestRuntimeObjectFitsMatchGo in
// wasm/verify, the same way the tag and <input> type tables are.
//
// # The value, not the declaration
//
// This table stores "contain", where the other two store the whole answer.
// That is deliberate, and it is what made the two copies incomparable before:
// htmlout is assembling a semicolon-joined declaration list and needs
// "object-fit:contain", while the runtime assigns el.style.objectFit and needs
// "contain". They agree on the mapping and differ on what they wrap it in, so
// the shared table holds the part they share and objectFitDecl (export.go) adds
// the property name for the one caller that wants a declaration.
//
// # Two ways of saying nothing
//
// An absent or unrecognized mode yields "". The two renderers then express
// that differently and mean the same thing: htmlout emits no declaration at
// all, and the runtime assigns "" to the property, which *clears* it — which
// matters on the patch path, where an Image whose contentMode prop is removed
// has to go back to the browser's default rather than keep the last mode it
// was given.
//
// Note that "" is not the same as ContentModeFit here. core.Image omits the
// prop entirely rather than writing "fit" (see imageNode), so the mode-less
// case has to export exactly as it did before ContentMode existed; the natives
// are the ones that fold the missing prop into Fit, because SwiftUI and
// Compose have no "unset" to fall back to.
var objectFits = map[core.ContentMode]string{
	core.ContentModeFit:     "contain",
	core.ContentModeFill:    "cover",
	core.ContentModeStretch: "fill",
	core.ContentModeCenter:  "none",
}

// ObjectFitFor returns the CSS object-fit value a content mode maps to, or ""
// for a mode that is absent or unrecognized.
//
// It takes a string rather than a core.ContentMode because every caller is
// reading it back out of a node's Props, where it arrives as one — the same
// reason InputTypeFor and TagFor take strings.
func ObjectFitFor(mode string) string {
	return objectFits[core.ContentMode(mode)]
}

// ObjectFits returns a copy of the whole table, keyed by the mode's string
// form, for the callers that must enumerate it rather than query it: the WASM
// runtime conformance test, which compares table against table, and the census
// test that holds this table to core.ContentModes().
//
// A copy, not the map itself, for the reason InputTypes and Tags return one.
// Keyed by string rather than by core.ContentMode because both callers are
// comparing against something that has already lost the Go type — a parsed
// JavaScript literal, and a set built from strings.
func ObjectFits() map[string]string {
	out := make(map[string]string, len(objectFits))
	for k, v := range objectFits {
		out[string(k)] = v
	}
	return out
}
