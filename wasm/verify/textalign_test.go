package main

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/htmlout"
)

// core.Alignment -> CSS text-align, the fourth of the tables the runtime
// restates in JavaScript. Same arrangement as the tag, <input> type and
// object-fit tables: Go holds the authority (htmlout/textalign.go), the
// runtime holds a copy because it is the side assigning the property, and this
// keeps them equal under a plain `go test ./...`. See jstable_test.go for the
// parse.
//
// This one is different from the other three in what it was for. Those pinned
// two copies that already existed and already agreed. Here there was no second
// copy: the runtime never read style.Align in any form, so every alignment on
// the web target was dropped while htmlout emitted a declaration for three of
// the six values and both natives set one. The table and this test arrived
// together, and the test's job from here on is the ordinary one — to keep the
// two from drifting apart again.
//
// What a mismatch costs, in either direction, is text that reads centered in
// the exported HTML and left-aligned in the live app, or the reverse, with no
// error anywhere.
func TestRuntimeTextAlignsMatchGo(t *testing.T) {
	table := parseRuntimeTable(t, runtimeSource(t), "textAlignFor", "")

	want := htmlout.TextAligns()
	for align, jsValue := range table {
		if goValue := want[align]; goValue != jsValue {
			t.Errorf("%s: runtime says %q, htmlout says %q", align, jsValue, goValue)
		}
		delete(want, align)
	}
	// Anything left is an alignment Go emits and the runtime does not, which is
	// the direction that silently falls back to the document's alignment.
	for align, goValue := range want {
		t.Errorf("%s: htmlout says %q, runtime has no entry", align, goValue)
	}
}

// The table being right is not enough on its own: styleFromGrMob has to
// actually call it, which is the half that was missing for the entire life of
// this runtime. A table nothing reads would pass the conformance test above
// and change nothing on screen.
//
// Checked by reading the source rather than by running the runtime, for the
// reason the whole of this file parses: a Go test that needed Node would stop
// running for anyone who has only Go. run.sh exercises the behavior itself.
func TestRuntimeStyleAppliesTextAlign(t *testing.T) {
	src := runtimeSource(t)
	const call = "out.textAlign = textAlignFor(style.Align)"
	if !strings.Contains(src, call) {
		t.Errorf("grmob-runtime.js: styleFromGrMob does not contain %q — the text-align table is "+
			"pinned to htmlout but nothing reads it, so Style.Align renders as nothing on the web "+
			"target while htmlout and both natives honor it", call)
	}
}
