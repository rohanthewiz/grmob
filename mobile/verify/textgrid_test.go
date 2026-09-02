package verify

import (
	"testing"
)

// core.TextGrid's two node types, held to both native type dispatches.
//
// The type dispatch is the one switch every node has to pass through, and
// both natives end it in a catch-all that renders the children in a plain
// column. So a node type the renderer has not been taught does not fail: a
// TextGrid would draw as a column of nothing (its rows carry no text of their
// own, only a runs prop), with no error anywhere — the same silence the
// ContentMode and alignment checks exist to break. This reads the two
// dispatches and requires an arm for the grid and one for its row.
//
// Membership rather than full coverage, because core has no census of node
// types for the arms to be held against; the tag tables in htmlout and the
// WASM runtime are the DOM side's census, and they are compared to each
// other in wasm/verify. See switchlabels_test.go for the parse.

func requireArms(t *testing.T, file, fn string, arms []string, want ...string) {
	t.Helper()
	seen := map[string]int{}
	for _, label := range arms {
		seen[label]++
	}
	for _, label := range want {
		switch seen[label] {
		case 0:
			t.Errorf("%s: %s has no arm for %q — it falls through to the catch-all and renders as an "+
				"empty column", file, fn, label)
		case 1:
		default:
			t.Errorf("%s: %s has %d arms for %q; all but the first are unreachable", file, fn,
				seen[label], label)
		}
	}
}

func TestSwiftTypeDispatchDrawsTextGrid(t *testing.T) {
	syntax := swiftSwitch.with(swiftRenderer, "struct RenderNode:", "switch node.type {")
	requireArms(t, "Renderer.swift", "RenderNode", syntax.labels(t), "TextGrid", "GridRow")
}

func TestKotlinTypeDispatchDrawsTextGrid(t *testing.T) {
	syntax := kotlinWhen.with(kotlinRenderer, "fun RenderNodeContent(", "when (node.type) {")
	requireArms(t, "Renderer.kt", "RenderNodeContent", syntax.labels(t), "TextGrid", "GridRow")
}
