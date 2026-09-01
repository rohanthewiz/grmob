package verify

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The two native ContentMode mappings, held to core.ContentModes().
//
// core.ContentMode is restated four times, once per renderer, and until now
// only two of those copies were checked. The DOM pair could be checked by
// comparing tables, because htmlout and the WASM runtime map a mode onto the
// same CSS keyword and so hold literally the same table (htmlout/objectfit.go
// is the authority; wasm/verify/objectfit_test.go is the pin). The natives map
// onto vocabularies with no CSS in them at all:
//
//	mode     | CSS object-fit | SwiftUI                       | Compose ContentScale
//	---------+----------------+-------------------------------+---------------------
//	fit      | contain        | resizable + scaledToFit       | Fit
//	fill     | cover          | resizable + scaledToFill + clip | Crop
//	stretch  | fill           | resizable, no aspect ratio    | FillBounds
//	center   | none           | not resizable                 | None
//
// Three columns of values with nothing in common, so no table comparison can
// reach them. What all four share is the *key set*, and coverage of that key
// set is what these tests check.
//
// Coverage turns out to be the half that bites. Both natives fold the
// unrecognized case into fit, because SwiftUI and Compose have no "unset" to
// fall back to the way CSS does. So a fifth ContentMode added to core would
// today draw as fit on iOS and Android while both DOM targets fell back to the
// browser's default — four renderers, two behaviors, no error anywhere. The
// failure mode is not a crash but a design that is subtly wrong on half the
// platforms it ships to, which is exactly the sort of thing that ships.
//
// See switchlabels_test.go for why this is a parse and not a compile.

func TestSwiftScalingCoversEveryContentMode(t *testing.T) {
	syntax := swiftSwitch.with(
		swiftRenderer,
		"func grMobScaled(",
		"switch mode {",
	)
	requireContentModeCoverage(t, "Renderer.swift", "grMobScaled", syntax.labels(t))
}

func TestKotlinContentScaleCoversEveryContentMode(t *testing.T) {
	syntax := kotlinWhen.with(
		kotlinRenderer,
		"fun contentScaleFor(",
		"when (mode) {",
	)
	requireContentModeCoverage(t, "Renderer.kt", "contentScaleFor", syntax.labels(t))
}

// requireContentModeCoverage checks a renderer's arms against core's list in
// both directions.
//
// Both directions matter, and they fail for different reasons:
//
//   - A mode with no arm is the gap this whole exercise is about: it falls
//     through to the catch-all and draws as fit, silently disagreeing with the
//     DOM targets.
//   - An arm with no mode is dead code that reads as deliberate support. It is
//     either a typo ("centre") that has quietly never matched anything, or a
//     mode that was removed from core and left behind here — and the next
//     person to read the renderer would reasonably conclude the framework
//     supports it.
//
// The comparison is written out here rather than shared with the DOM table
// checks because it is not the same comparison: those compare two maps of
// values, this compares a key set against a list, and folding them together
// would mean a helper whose signature explains less than the ten lines it
// replaced.
func requireContentModeCoverage(t *testing.T, file, fn string, arms []string) {
	t.Helper()

	missing := map[string]bool{}
	for _, mode := range core.ContentModes() {
		missing[string(mode)] = true
	}

	for _, label := range arms {
		if !missing[label] {
			// Not in the list, or already claimed by an earlier arm — in the
			// second case the later arm is unreachable, which is worth the
			// same complaint.
			t.Errorf("%s: %s has an arm for %q, which is not a core.ContentMode (or is a second arm "+
				"for one, and therefore unreachable)", file, fn, label)
			continue
		}
		delete(missing, label)
	}

	for mode := range missing {
		t.Errorf("%s: %s has no arm for core.ContentMode %q — it falls through to the catch-all and "+
			"draws as fit, while htmlout and the WASM runtime fall back to the browser default",
			file, fn, mode)
	}
}
