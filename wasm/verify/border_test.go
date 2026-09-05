package main

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/htmlout"
)

// The set of tags whose user-agent border has to be written back to nothing.
//
// Go is the authority (borderResetTags in htmlout/tag.go) and the runtime
// restates it, so the two are compared here rather than remembered — the same
// treatment the tag, input-type and generic-tag tables get. A drift is silent
// in the worst way: the exported document and the live app draw the same node
// with and without a rule around it, and nothing errors on either side.
//
// Parsed with a regexp rather than through parseRuntimeTable, which reads
// key/value object literals; this one is a flat array of strings.
func TestRuntimeBorderResetTagsMatchGo(t *testing.T) {
	src := runtimeSource(t)

	m := regexp.MustCompile(`const BORDER_RESET_TAGS = new Set\(\[([^\]]*)\]\);`).FindStringSubmatch(src)
	if m == nil {
		t.Fatalf("grmob-runtime.js: no `const BORDER_RESET_TAGS = new Set([...])` found — if it " +
			"was renamed or spread over several lines, update this test rather than deleting it")
	}
	var got []string
	for _, q := range regexp.MustCompile(`"([^"]*)"`).FindAllStringSubmatch(m[1], -1) {
		got = append(got, q[1])
	}
	sort.Strings(got)

	want := htmlout.BorderResetTags()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BORDER_RESET_TAGS is %v; htmlout.BorderResetTags() is %v — a tag the browser's "+
			"own border survives on one web target and not the other", got, want)
	}
}

// The set is only half the contract; the other half is that the runtime
// actually consults it, and consults it in the *negative* arm.
//
// Putting the reset in the positive arm, or writing it as a separate guarded
// assignment, would both look right and break styleFromGrMob's totality rule:
// every property it manages is assigned on every call, so that a field back at
// its zero value clears the declaration an earlier patch left behind. A
// guarded `if (BORDER_RESET_TAGS.has(...)) out.border = "none"` would leave a
// button that loses its border showing the last one it had.
func TestRuntimeResetsTheUserAgentBorder(t *testing.T) {
	src := runtimeSource(t)
	const expr = `: (BORDER_RESET_TAGS.has(tagForType(nodeType)) ? "none" : "");`
	if !strings.Contains(src, expr) {
		t.Errorf("grmob-runtime.js: styleFromGrMob is missing %q — without it a <button> with no "+
			"border in its style keeps the browser's, which is the divergence "+
			"components.Button's EmphasisGhost exposed", expr)
	}
}
