package verify

import (
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/htmlout"
)

// Every node type the framework can emit has an arm in both native renderers.
//
// # The gap this closes
//
// A new node type costs four renderers, and three of the four already say so
// when one is missed. htmlout has a tag table that is documented as a census —
// "a type that is merely ordinary should appear here, so that adding a node
// type to core and forgetting the renderers shows up as a gap in a list rather
// than as silence" — and the WASM runtime's copy of that table is held to it by
// TestRuntimeTagsMatchGo. The two natives had nothing. Their dispatch ends in a
// catch-all (`else ->` and `default:`), which is what makes a forgotten arm
// silent *by construction*: a Switch with no arm draws as a Column of its
// children, which for a childless control is an empty box. Nothing crashes,
// nothing logs, and the control is simply not there on the phone while working
// on both web targets.
//
// That is the failure this is written against, and it is not hypothetical —
// core.Switch was added with its four renderer arms, and the only thing that
// would have caught three of them was this test not existing yet.
//
// # What the list is, and why it is htmlout's
//
// The authority is htmlout.Tags() plus htmlout.TransparentTypes(), which
// together are every node type either DOM renderer knows. core does not export
// a census of its own node types — a type is a string literal in whichever
// builder emits it — and inventing a second list here would be exactly the
// untracked copy htmlout/tag.go exists to remove. So the tag table is the
// census, for the natives too, and a node type that reaches neither DOM
// renderer is outside every target's knowledge rather than just these two.
//
// The transparent types are in because both natives do handle them (`Fragment`,
// `Theme` -> render the children into the parent's scope), and their arm is
// load-bearing in a way the catch-all would not be: the fallback wraps children
// in a Column, which for a Fragment inside a Row rotates the axis of whatever
// core.For generated.
//
// # Why a parse and not a compile
//
// See switchlabels_test.go. `else ->` and `default:` make a string dispatch
// exhaustive by construction in both languages, so "you forgot a node type" is
// not a type error in either and never will be — the only thing that can notice
// is something holding the arms up against Go's list.
func TestBothNativeRenderersDispatchEveryNodeType(t *testing.T) {
	want := nodeTypeCensus()

	for _, r := range []struct {
		name   string
		syntax dispatchSyntax
	}{
		{"Renderer.swift", swiftSwitch.with(
			swiftRenderer,
			"struct RenderNode: View {",
			"switch node.type {",
		)},
		{"Renderer.kt", kotlinWhen.with(
			kotlinRenderer,
			"private fun RenderNodeContent(",
			"when (node.type) {",
		)},
	} {
		t.Run(r.name, func(t *testing.T) {
			got := map[string]bool{}
			for _, label := range r.syntax.labels(t) {
				got[label] = true
			}

			var missing []string
			for _, nodeType := range want {
				if !got[nodeType] {
					missing = append(missing, nodeType)
				}
			}
			if len(missing) > 0 {
				t.Errorf("%s has no arm for %s — the dispatch ends in a catch-all, so each of "+
					"these draws as a generic container on the phone while drawing correctly on "+
					"both web targets. That is the silence this test exists for.",
					r.name, strings.Join(missing, ", "))
			}

			// The other direction. An arm for a node type nothing emits is not
			// a bug in the renderer and is not failed here — a renderer may
			// legitimately know a type before the tag table does, which is the
			// order a new node type is usually built in. It is reported,
			// because the other thing it can be is a node type that was renamed
			// in core and left behind here, where it draws nothing and costs
			// nobody an error.
			var extra []string
			for label := range got {
				if !slices.Contains(want, label) {
					extra = append(extra, label)
				}
			}
			if len(extra) > 0 {
				sort.Strings(extra)
				t.Logf("%s dispatches on %s, which no DOM renderer knows — either a node type "+
					"on its way in, or one renamed in core and left behind here",
					r.name, strings.Join(extra, ", "))
			}
		})
	}
}

// nodeTypeCensus is every node type either DOM renderer knows, sorted so a
// failure reports in a stable order. See the test above for why this is
// htmlout's list and not one of this package's own.
func nodeTypeCensus() []string {
	tags := htmlout.Tags()
	out := make([]string, 0, len(tags)+2)
	for nodeType := range tags {
		out = append(out, nodeType)
	}
	out = append(out, htmlout.TransparentTypes()...)
	sort.Strings(out)
	return out
}
