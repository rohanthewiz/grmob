package htmlout

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/richtext"
)

var exportDoc = richtext.Doc{Blocks: []richtext.Block{
	{Kind: richtext.Heading2, Runs: []richtext.Run{{Text: "Notes"}}},
	{Kind: richtext.Paragraph, Runs: []richtext.Run{
		{Text: "A "},
		{Text: "bold", Bold: true},
		{Text: " word and a "},
		{Text: "link", Link: "https://example.com"},
		{Text: "."},
	}},
	{Kind: richtext.Bullet, Runs: []richtext.Run{{Text: "one"}}},
	{Kind: richtext.Bullet, Runs: []richtext.Run{{Text: "two"}}},
}}

// A static export of an editor is the document: no caret, no toolbar, nothing
// to type into. The callback IDs travel anyway, so a loader with an event loop
// can wire the live editor out of it.
func TestRichTextEditorExportsTheDocument(t *testing.T) {
	n := core.RichTextEditor(exportDoc, func(richtext.Doc) {},
		core.OnRichSelectionChange(func(core.RichSelection) {}),
	).Render(core.NewContext())
	out := ExportHTML(n)

	for _, want := range []string{
		`line-height:1.5; white-space:normal`,
		"<h2>Notes</h2>",
		`<p>A <strong>bold</strong> word and a <a href="https://example.com">link</a>.</p>`,
		// The exporter pretty-prints, so the list's own elements are checked
		// one at a time rather than as the one string richtext.HTML returns.
		"<ul>",
		"<li>one</li>",
		"<li>two</li>",
		"</ul>",
		"data-onchange=",
		"data-onselectionchange=",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("export lacks %q:\n%s", want, out)
		}
	}
}

// An empty document draws its placeholder, which is the only thing a static
// export of an empty editor has to show.
func TestRichTextEditorExportsThePlaceholderWhenEmpty(t *testing.T) {
	n := core.RichTextEditor(richtext.Doc{}, nil,
		core.Placeholder("Write something…"),
	).Render(core.NewContext())
	out := ExportHTML(n)

	if !strings.Contains(out, `<span style="opacity:0.45">Write something…</span>`) {
		t.Errorf("no placeholder:\n%s", out)
	}
}

// A document is a value, not markup. The escaping lives in richtext — three
// targets build the same document from the same runs — and this is the export's
// half of the check: what reaches the page is the escaped form.
func TestRichTextEditorExportEscapesContent(t *testing.T) {
	doc := richtext.Doc{Blocks: []richtext.Block{{Runs: []richtext.Run{
		{Text: `<script>alert(1)</script>`},
	}}}}
	out := ExportHTML(core.RichTextEditor(doc, nil).Render(core.NewContext()))
	if strings.Contains(out, "<script>") {
		t.Errorf("a run of text re-entered the page as a tag:\n%s", out)
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Errorf("the text did not survive as text:\n%s", out)
	}
}

// A hand-built node whose doc prop is not a document exports as an empty box
// rather than panicking — the same degradation an Image with no src gets.
func TestRichTextEditorWithAForeignDocIsEmpty(t *testing.T) {
	n := &core.Node{Type: "RichTextEditor", Props: map[string]any{"doc": "nope"}}
	out := ExportHTML(n)
	if strings.Contains(out, "nope") {
		t.Errorf("the unparseable prop reached the page:\n%s", out)
	}
}
