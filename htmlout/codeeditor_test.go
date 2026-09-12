package htmlout

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/highlight"
)

// core.CodeEditor exports as the <pre> a TextGrid does, plus a gutter, and
// with nothing to type into: a static document has no event loop, so what
// survives is the buffer, coloured, and the callback IDs a loader can wire.
func TestCodeEditorExportsAGutterAndItsRows(t *testing.T) {
	src := "func main() {\n\tprintln(1)\n}"
	n := core.CodeEditor(src, func(string) {}, highlight.Go().Rows(src, highlight.Darcula),
		core.LineNumbers(),
		core.OnSelectionChange(func(int, int) {}),
		core.FontSize(13),
	).Render(core.NewContext())
	out := ExportHTML(n)

	for _, want := range []string{
		// The chassis first, the theme's field base and the author's style
		// after it, and the gutter inset last of all — which is the one
		// chassis declaration that outranks the author; see renderNode.
		`<pre style="margin:0; line-height:1.2; white-space:normal; overflow:auto; position:relative;`,
		`font-size:13px`,
		`padding-left:3ch" data-onchange=`,
		// The gutter: chrome, out of flow, unselectable, and hidden from
		// readers. Its numbers are one text node, newline-separated.
		`position:absolute; left:0; top:0; width:3ch`,
		`data-grmob-chrome="gutter"`,
		`aria-hidden="true"`,
		`>1
2
3</div>`,
		// The rows are the grid's own rows, drawn by renderGridRow.
		`<div style="min-height:1.2em; white-space:nowrap"`,
		`<span style="white-space:pre; color:` + highlight.Darcula.Keyword + `">func</span>`,
		// The IDs travel even though nothing can fire them here.
		`data-onchange=`,
		`data-onselectionchange=`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("export lacks %q:\n%s", want, out)
		}
	}
	if got := strings.Count(out, `min-height:1.2em; white-space:nowrap`); got != 3 {
		t.Errorf("exported %d rows, want one per line:\n%s", got, out)
	}
}

// No gutter, no inset, no chrome — an editor that did not ask for line numbers
// is a <pre> of rows and nothing else.
func TestCodeEditorWithoutLineNumbersHasNoGutter(t *testing.T) {
	n := core.CodeEditor("x", func(string) {}, []core.GridRow{{{Text: "x"}}}).Render(core.NewContext())
	out := ExportHTML(n)

	if strings.Contains(out, "data-grmob-chrome") {
		t.Errorf("a gutterless editor still exported chrome:\n%s", out)
	}
	if strings.Contains(out, "padding-left") {
		t.Errorf("a gutterless editor still exported the gutter inset:\n%s", out)
	}
}

// The gutter is exactly as wide as its widest number plus a column of room,
// in `ch` units — so it fits at any font size with nothing measured. The WASM
// runtime computes the same string.
func TestGutterWidthFollowsTheDigitCount(t *testing.T) {
	for lines, want := range map[int]string{1: "3ch", 9: "3ch", 10: "4ch", 100: "5ch"} {
		if got := gutterWidth(lines); got != want {
			t.Errorf("gutterWidth(%d) = %q, want %q", lines, got, want)
		}
	}
}

// A row's white space is content and the exporter's own indentation is not —
// the same three-level split a TextGrid makes, which is what lets the document
// be pretty-printed without the code gaining blank lines.
func TestCodeEditorPreservesIndentation(t *testing.T) {
	n := core.CodeEditor("\tif x {", func(string) {}, []core.GridRow{{{Text: "\tif x {"}}}).
		Render(core.NewContext())
	out := ExportHTML(n)
	// Literal, not a character reference: the reference is only needed for a
	// run that is *nothing but* white space, which the pretty-printer would
	// otherwise discard as its own indentation. A run with a glyph in it goes
	// through element's escaping path and keeps its leading tab. Same rule
	// renderGridRow applies to a TextGrid.
	if !strings.Contains(out, "<span style=\"white-space:pre\">\tif x {</span>") {
		t.Errorf("the leading tab did not survive:\n%s", out)
	}
}
