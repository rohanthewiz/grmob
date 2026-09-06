package main

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/htmlout"
)

// The set of node types whose user-agent border has to be written back to
// nothing.
//
// Go is the authority (borderResetTypes in htmlout/tag.go) and the runtime
// restates it, so the two are compared here rather than remembered — the same
// treatment the tag, input-type and generic-tag tables get. A drift is silent
// in the worst way: the exported document and the live app draw the same node
// with and without a rule around it, and nothing errors on either side.
//
// Parsed with a regexp rather than through parseRuntimeTable, which reads
// key/value object literals; this one is a flat array of strings.
func TestRuntimeBorderResetTypesMatchGo(t *testing.T) {
	src := runtimeSource(t)

	m := regexp.MustCompile(`const BORDER_RESET_TYPES = new Set\(\[([^\]]*)\]\);`).FindStringSubmatch(src)
	if m == nil {
		t.Fatalf("grmob-runtime.js: no `const BORDER_RESET_TYPES = new Set([...])` found — if it " +
			"was renamed or spread over several lines, update this test rather than deleting it")
	}
	var got []string
	for _, q := range regexp.MustCompile(`"([^"]*)"`).FindAllStringSubmatch(m[1], -1) {
		got = append(got, q[1])
	}
	sort.Strings(got)

	want := htmlout.BorderResetTypes()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BORDER_RESET_TYPES is %v; htmlout.BorderResetTypes() is %v — a node type the "+
			"browser's own border survives on one web target and not the other", got, want)
	}
}

// Every member of the set has to be a node type the tag table knows, and the
// three <input> types that are *not* in it have to stay out.
//
// The set was keyed by tag while <button> was its only member and is keyed by
// node type now that text fields have joined, so the spellings changed from
// "button" to "Button" — a difference no compiler notices, since both sides
// are map keys of type string. A stale lowercase entry would simply never
// match, and the symptom would be the divergence this whole mechanism exists
// to close, showing up as a missing rule rather than as a failure.
func TestBorderResetTypesAreNodeTypes(t *testing.T) {
	tags := htmlout.Tags()
	for _, nodeType := range htmlout.BorderResetTypes() {
		if _, ok := tags[nodeType]; !ok {
			t.Errorf("borderResetTypes has %q, which is not a node type in htmlout.Tags() — "+
				"the set is keyed by core node type, not by HTML tag", nodeType)
		}
	}
	// The controls the user agent draws in their entirety. Resetting their
	// border would be asking the browser not to draw the box that *is* the
	// checkbox; a range track has no border to begin with. See
	// borderResetTypes for why keying by tag would have swept both in.
	for _, nodeType := range []string{"Checkbox", "Slider"} {
		if htmlout.ResetsUABorder(nodeType) {
			t.Errorf("%s is in borderResetTypes — the user agent draws that control itself, "+
				"and the Go style is not meant to own its frame", nodeType)
		}
	}
}

// The set is only half the contract; the other half is that the runtime
// actually consults it, and consults it in the *negative* arm.
//
// Putting the reset in the positive arm, or writing it as a separate guarded
// assignment, would both look right and break styleFromGrMob's totality rule:
// every property it manages is assigned on every call, so that a field back at
// its zero value clears the declaration an earlier patch left behind. A
// guarded `if (BORDER_RESET_TYPES.has(...)) out.border = "none"` would leave a
// button that loses its border showing the last one it had.
func TestRuntimeResetsTheUserAgentBorder(t *testing.T) {
	src := runtimeSource(t)
	const expr = `: (BORDER_RESET_TYPES.has(nodeType) ? "none" : "");`
	if !strings.Contains(src, expr) {
		t.Errorf("grmob-runtime.js: styleFromGrMob is missing %q — without it a <button> with no "+
			"border in its style keeps the browser's, which is the divergence "+
			"components.Button's EmphasisGhost exposed", expr)
	}
}
