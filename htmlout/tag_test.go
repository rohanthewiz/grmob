package htmlout

import (
	"regexp"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The tag table is only worth calling an authority if the exporter actually
// obeys it. renderNode does not read it for the node types it handles by name
// — those go through element's typed calls (b.Span, b.Button, b.Img, ...),
// which is more readable at the call site and spells the tag a second time —
// so this test is what ties those spellings back to the table. renderContainer
// does read it, so the div rows are checked here as a matter of course rather
// than by inspection.
//
// Without this, TestRuntimeTagsMatchGo in wasm/verify would be comparing the
// runtime against a table that nothing in Go was required to follow.

// firstBodyTag pulls the name of the first element inside <body> out of an
// exported document. (?s) so the dot spans the newlines Pretty inserts.
var firstBodyTag = regexp.MustCompile(`(?s)<body[^>]*>\s*<([a-z]+)`)

// Props a node type needs before it renders as its table entry at all. Only
// Image has one: renderNode falls through to the container path for an Image
// with no src, degrading to a div the way any underspecified node does, so
// asking for "img" without a src would be asking for a case that does not
// exist.
var tagTestProps = map[string]map[string]any{
	"Image": {"src": "a.png"},
}

func TestExportedTagsMatchTable(t *testing.T) {
	for nodeType, want := range Tags() {
		n := &core.Node{Type: nodeType, Props: tagTestProps[nodeType]}
		out := ExportHTML(n)

		m := firstBodyTag.FindStringSubmatch(out)
		if m == nil {
			t.Errorf("%s: exported no element inside <body>:\n%s", nodeType, out)
			continue
		}
		if m[1] != want {
			t.Errorf("%s: exported as <%s>, table says <%s>", nodeType, m[1], want)
		}
	}
}

// The transparency set is the other half of the table, and the half a bare
// lookup cannot express: these node types render as no element at all. Pinned
// here as a set rather than one Fragment case (TestFragmentEmitsNoWrapperElement
// covers that in depth) so that making some other node type transparent, or
// quietly dropping one of these, has to pass through a failing test.
func TestTransparentTypesEmitNoElementOfTheirOwn(t *testing.T) {
	for _, nodeType := range TransparentTypes() {
		n := &core.Node{Type: nodeType, Children: []*core.Node{
			{Type: "Text", Props: map[string]any{"content": "child"}},
		}}
		out := ExportHTML(n)

		m := firstBodyTag.FindStringSubmatch(out)
		if m == nil {
			t.Errorf("%s: exported nothing at all; the child should have survived:\n%s", nodeType, out)
			continue
		}
		// The child, promoted into the body — not a wrapper with the child
		// inside it.
		if m[1] != "span" {
			t.Errorf("%s: wrapped its children in <%s>", nodeType, m[1])
		}
		if !strings.Contains(out, "child") {
			t.Errorf("%s: dropped its child:\n%s", nodeType, out)
		}
	}
}

// The two tables answer different questions ("which element" and "whether an
// element"), and a node type in both would mean the second answer is being
// ignored: TagFor would hand renderContainer a tag for something renderNode
// returns before ever reaching it, and the wasm conformance test would both
// compare and exempt the same row.
func TestNoTypeIsBothTaggedAndTransparent(t *testing.T) {
	tags := Tags()
	for _, nodeType := range TransparentTypes() {
		if tag, ok := tags[nodeType]; ok {
			t.Errorf("%s is transparent but also has the tag %q", nodeType, tag)
		}
	}
}

// TagFor's default is what an unknown node type gets, and the WASM runtime's
// `|| "div"` fallback is pinned to the same value, so it is stated as a test
// rather than left to the map's zero value — which would be "", an empty tag
// name that createElement rejects outright.
func TestTagForUnknownTypeIsADiv(t *testing.T) {
	if got := TagFor("SomethingNobodyHasWrittenYet"); got != "div" {
		t.Errorf("TagFor(unknown) = %q, want div", got)
	}
}

// Tags hands out a copy, for the same reason InputTypes does: the conformance
// test deletes from what it is given as it matches rows, and would otherwise
// be emptying the exporter's own table as it went.
func TestTagsReturnsACopy(t *testing.T) {
	first := Tags()
	delete(first, "Text")
	first["Row"] = "table"

	if got := TagFor("Text"); got != "span" {
		t.Errorf("deleting from the returned map changed TagFor(Text): %q", got)
	}
	if got := TagFor("Row"); got != "div" {
		t.Errorf("writing to the returned map changed TagFor(Row): %q", got)
	}
}
