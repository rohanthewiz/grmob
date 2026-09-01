package main

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/htmlout"
)

// core.Alignment -> CSS align-items, the fifth table the runtime restates in
// JavaScript, and the second (after text-align) that was added to close a gap
// rather than to pin a copy that already existed: Style.Align's cross-axis
// fallback was read by both natives since they existed and by neither DOM
// target. Go holds the authority (htmlout/crossaxis.go), the runtime holds a
// copy because it is the side assigning the property, and this keeps them
// equal under a plain `go test ./...`. See jstable_test.go for the parse.
//
// What a mismatch costs is a Column whose children center in the exported
// HTML and start-pack in the live app, or the reverse, with no error anywhere.
func TestRuntimeCrossAxisAlignsMatchGo(t *testing.T) {
	table := parseRuntimeTable(t, runtimeSource(t), "crossAxisAlignFor", "")

	want := htmlout.CrossAxisAligns()
	for align, jsValue := range table {
		if goValue := want[align]; goValue != jsValue {
			t.Errorf("%s: runtime says %q, htmlout says %q", align, jsValue, goValue)
		}
		delete(want, align)
	}
	for align, goValue := range want {
		t.Errorf("%s: htmlout says %q, runtime has no entry", align, goValue)
	}
}

// The value table's twin: which node types consult the fallback at all. The
// natives read it for Column, Card and List and decline it everywhere else —
// most pointedly on Row — and each DOM renderer carries that gate as a table
// of its own, so the gate can drift exactly the way a value table can: one
// DOM target reading the fallback for a type the other ignores.
func TestRuntimeAlignFallbackAxesMatchGo(t *testing.T) {
	table := parseRuntimeTable(t, runtimeSource(t), "alignFallbackAxisFor", "")

	want := htmlout.AlignFallbackAxes()
	for typ, jsValue := range table {
		if goValue := want[typ]; goValue != jsValue {
			t.Errorf("%s: runtime says %q, htmlout says %q", typ, jsValue, goValue)
		}
		delete(want, typ)
	}
	for typ, goValue := range want {
		t.Errorf("%s: htmlout says %q, runtime has no entry", typ, goValue)
	}
}

// The tables being right is not enough on their own: styleFromGrMob has to
// consult them, with AlignItems taking precedence and both gates applied —
// the same reason TestRuntimeStyleAppliesTextAlign exists. Substrings of the
// actual expressions rather than function names alone, so a comment cannot
// satisfy the pin (the property mobile/verify's stretch pin established).
func TestRuntimeStyleAppliesTheAlignFallback(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []string{
		`dir.startsWith("column") && alignFallbackAxisFor(nodeType)`,
		`alignItems = crossAxisAlignFor(style.Align || "")`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("grmob-runtime.js: styleFromGrMob does not contain %q — the cross-axis tables "+
				"are pinned to htmlout but the fallback is not being read, so Align places children "+
				"in the exported HTML and not in the live app", want)
		}
	}
}
