package main

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/htmlout"
)

// Which node types stack their children whether or not a Style asks — the
// sixth table the runtime restates in JavaScript, and the third added to
// close a gap rather than to pin a copy that already existed. Go holds the
// authority (htmlout/stack.go), the runtime holds a copy because it is the
// side calling createElement, and this keeps them equal under a plain
// `go test ./...`. See jstable_test.go for the parse.
//
// The two directions of a mismatch are both silent, and both are layout that
// simply comes out wrong:
//
//	runtime has a row Go lacks   → the exported HTML runs the container's
//	                               children down the page in block flow while
//	                               the live app stacks them across it
//	Go has a row the runtime lacks → the reverse, plus the container's gap,
//	                               justify-content and align-items go inert in
//	                               the live app
func TestRuntimeStackAxesMatchGo(t *testing.T) {
	table := parseRuntimeTable(t, runtimeSource(t), "stackAxisFor", "")

	// The transparent grouping nodes are exempt, exactly as they are in the
	// tag comparison next door and for the same underlying reason: htmlout
	// emits no element for them, so it has nothing to stack, while the
	// runtime must box them to keep its positional patch addressing valid.
	// A box that were not a stack would swallow its parent's layout, so the
	// exemption is narrowed to the axis the runtime is known to use rather
	// than left as a hole a real drift could hide in.
	for _, nodeType := range htmlout.TransparentTypes() {
		axis, ok := table[nodeType]
		switch {
		case !ok:
			t.Errorf("%s: the runtime boxes it in a real div (see tagForType) and has no stack row; "+
				"that box is block flow and swallows the layout meant for its children", nodeType)
		case axis != "column":
			t.Errorf("%s: runtime stacks it along %q; the known divergence is a column", nodeType, axis)
		}
		delete(table, nodeType)
	}

	want := htmlout.StackAxes()
	for nodeType, jsAxis := range table {
		if goAxis := want[nodeType]; goAxis != jsAxis {
			t.Errorf("%s: runtime stacks along %q, htmlout along %q", nodeType, jsAxis, goAxis)
		}
		delete(want, nodeType)
	}
	for nodeType, goAxis := range want {
		t.Errorf("%s: htmlout stacks along %q, runtime has no row (it stays block flow)", nodeType, goAxis)
	}
}

// The table being right is not enough: something has to consult it, on both
// the path that builds an element and the path that patches one.
//
// On this target those are one function. styleFromGrMob decides the axis and
// the promotion to a flex container, and it is *total* — an update-style patch
// assigns every property it manages, so a display it did not write is a
// display it erased — which is why the read has to be in there and not only at
// build time. createElement then reaches it for every node by passing an empty
// object where there is no Style, so the default is planted and restated by
// one call site rather than two that could disagree. That third substring is
// the one holding the build path up: a runtime that went back to styling only
// the nodes that carry a Style would leave a bare Column in block flow, and
// no patch would ever arrive to correct it. htmlout, which has no patch path
// to be total for, still reads the table in two places of its own.
//
// Substrings of the actual expressions rather than the function name alone,
// so a comment mentioning stackAxisFor cannot satisfy the pin — the property
// mobile/verify's dispatch pins established.
func TestRuntimeAppliesTheStackDefault(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []string{
		// styleFromGrMob: the table decides the axis and the promotion.
		`stackAxisFor(nodeType) || "column"`,
		`alignItems || style.FlexDirection || stackAxisFor(nodeType)`,
		// createElement: every node reaches that function, Style or no Style.
		// a11y_test.go pins the same line for the other thing riding on it.
		`applyStyle(el, node.Style || {}, node.Type);`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("grmob-runtime.js: %q not found — the stack table is pinned to htmlout but "+
				"the runtime does not read it where it must, so a container stacks on one "+
				"web target and runs in block flow on the other", want)
		}
	}
}
