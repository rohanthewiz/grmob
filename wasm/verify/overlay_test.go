package main

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/htmlout"
)

// The set of node types whose children are drawn on top of one another.
//
// Go is the authority (overlayTypes in htmlout/stack.go) and the runtime
// restates it, so the two are compared here rather than remembered — the same
// treatment the tag, input-type, generic-tag, stack-axis and border-reset
// tables get. Drift in either direction is silent and total: the layers of one
// overlay stack down the page on one web target and lie on top of each other
// on the other, and nothing errors on either side.
//
// Parsed with a regexp rather than through parseRuntimeTable, which reads
// key/value object literals; this one is a flat array of strings, as
// BORDER_RESET_TYPES is.
func TestRuntimeOverlayTypesMatchGo(t *testing.T) {
	src := runtimeSource(t)

	m := regexp.MustCompile(`const OVERLAY_TYPES = new Set\(\[([^\]]*)\]\);`).FindStringSubmatch(src)
	if m == nil {
		t.Fatalf("grmob-runtime.js: no `const OVERLAY_TYPES = new Set([...])` found — if it was " +
			"renamed or spread over several lines, update this test rather than deleting it")
	}
	var got []string
	for _, q := range regexp.MustCompile(`"([^"]*)"`).FindAllStringSubmatch(m[1], -1) {
		got = append(got, q[1])
	}
	sort.Strings(got)

	want := htmlout.OverlayTypes()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("OVERLAY_TYPES is %v; htmlout.OverlayTypes() is %v — a node type overlays its "+
			"children on one web target and stacks them on the other", got, want)
	}
}

// The cell itself, which is the other half of the same contract and lives in a
// second constant because the two renderers write it in different places: an
// inline declaration list here, a CSSOM property there.
//
// Compared as the *value* rather than the whole declaration, since Go writes
// "grid-area:1/1" into a style attribute and the runtime assigns "1/1" to
// element.style.gridArea. A mismatch would put the layers of one stack in
// different cells on the two targets, which is a grid of layers on one and a
// pile on the other.
func TestRuntimeOverlayChildAreaMatchesGo(t *testing.T) {
	src := runtimeSource(t)

	m := regexp.MustCompile(`const OVERLAY_CHILD_AREA = "([^"]*)";`).FindStringSubmatch(src)
	if m == nil {
		t.Fatal("grmob-runtime.js: no `const OVERLAY_CHILD_AREA = \"...\"` found")
	}
	want := strings.TrimPrefix(htmlout.OverlayChildDecl, "grid-area:")
	if want == htmlout.OverlayChildDecl {
		t.Fatalf("htmlout.OverlayChildDecl is %q, which no longer names grid-area; this "+
			"comparison cannot strip a property it cannot find", htmlout.OverlayChildDecl)
	}
	if m[1] != want {
		t.Errorf("OVERLAY_CHILD_AREA is %q; htmlout places layers in %q", m[1], want)
	}
}

// The runtime must consult the set in styleFromGrMob's *first* branch, ahead
// of the flex test.
//
// Both facts matter and neither is visible from the table comparison above.
// Placing the overlay arm after the flex test would let a stack carrying a Gap
// or an AlignItems be promoted to a flex container and stop overlaying
// altogether — the loss of the whole node type, caused by a prop that is
// merely inert on an overlay. And an overlay branch that did not also clear
// flexDirection would break the function's totality rule, leaving the
// direction the box last had.
func TestRuntimeResolvesTheOverlayBeforeTheFlexTest(t *testing.T) {
	src := runtimeSource(t)

	overlay := strings.Index(src, `const overlay = OVERLAY_TYPES.has(nodeType);`)
	if overlay < 0 {
		t.Fatal("grmob-runtime.js: styleFromGrMob does not read OVERLAY_TYPES at all — a ZStack " +
			"renders as an ordinary block-flow div in the live app")
	}
	flex := strings.Index(src, `} else if (style.Gap || style.RowGap`)
	if flex < 0 {
		t.Fatal("grmob-runtime.js: the flex promotion test is no longer the arm after the " +
			"overlay one; update this test rather than deleting it")
	}
	if overlay > flex {
		t.Error("grmob-runtime.js: the overlay is resolved after the flex test, so a ZStack " +
			"carrying a Gap becomes a flex container and stops overlaying")
	}
	// The centring, which is the alignment contract core.ZStack documents, and
	// the two totality clears beside it.
	for _, expr := range []string{
		`out.alignItems = overlay ? "center" : (alignItems || "");`,
		`out.justifyItems = overlay ? "center" : "";`,
	} {
		if !strings.Contains(src, expr) {
			t.Errorf("grmob-runtime.js: styleFromGrMob is missing %q", expr)
		}
	}
}

// The layers are stamped by a pass over the container's children, and the pass
// has to run on the patch path too.
//
// The initial render is the easy half. An "add" patch lands a brand new layer
// under a ZStack that was not itself touched, so a runtime that stamped only
// at creation would place every layer of a stack except the ones that appeared
// later — and a layer with no cell is auto-placed into its own implicit grid
// row, i.e. below the stack instead of on it.
func TestRuntimeStampsOverlayLayersOnBothPaths(t *testing.T) {
	src := runtimeSource(t)
	for _, expr := range []string{
		"child.style.gridArea = OVERLAY_CHILD_AREA;", // the pass itself
		"syncOverlay(el);",                           // the initial render
		"syncTouchedOverlays(touched);",              // the patch path
	} {
		if !strings.Contains(src, expr) {
			t.Errorf("grmob-runtime.js: missing %q — an overlay's layers are not all placed "+
				"in its cell", expr)
		}
	}
}
