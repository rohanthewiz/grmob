package htmlout

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// core.TextGrid exports as a <pre> of row <div>s holding <span> runs. The
// grid's chassis pins the line height so an empty row keeps its line, and the
// three white-space declarations put the significance where it belongs — see
// textGridChassis. A run carries `white-space:pre` and then only the
// declarations it set, so a plain run is a span with that one rule.
func TestTextGridExportsRowsOfRuns(t *testing.T) {
	n := core.TextGrid([]core.GridRow{
		{{Text: "$ ls"}},
		{{Text: "a.go", Fg: "#00ff00", Bg: "#000000"}, {Text: " b.go", Attr: core.GridBold | core.GridUnderline | core.GridStrike}},
		{},
		{{Text: "<script>", Attr: core.GridDim | core.GridItalic}},
	}, core.FontSize(12)).Render(core.NewContext())
	out := ExportHTML(n)

	for _, want := range []string{
		`<pre style="margin:0; line-height:1.2; white-space:normal; overflow-x:auto; font-size:12px">`,
		`<span style="white-space:pre">$ ls</span>`,
		`<span style="white-space:pre; color:#00ff00; background:#000000">a.go</span>`,
		`<span style="white-space:pre; font-weight:700; text-decoration:underline line-through"> b.go</span>`,
		`<span style="white-space:pre; opacity:0.6; font-style:italic">&lt;script&gt;</span>`,
		`min-height:1.2em; white-space:nowrap`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("export lacks %q:\n%s", want, out)
		}
	}
	// Four rows, one of them empty, all present.
	if got := strings.Count(out, `<div style="min-height:1.2em; white-space:nowrap"`); got != 4 {
		t.Errorf("exported %d rows, want 4:\n%s", got, out)
	}
}

// A GridRow whose runs are not the typed slice exports as an empty row
// rather than panicking on a hand-built node.
func TestGridRowWithForeignRunsIsEmpty(t *testing.T) {
	n := &core.Node{Type: "GridRow", Props: map[string]any{"runs": []string{"nope"}}}
	if out := ExportHTML(n); !strings.Contains(out, "<div") || strings.Contains(out, "nope") {
		t.Errorf("export = %s", out)
	}
}

// The white space inside a run is content, and it survives the pretty-printer
// that lays out the document around it.
//
// This is the case the exporter used to lose outright. Its formatter drops any
// text node that is nothing but white space — it has no way to tell a run's
// spaces from its own indentation — so a run made only of spaces exported as
// an empty span, and an indented, syntax-highlighted line came out with its
// indent and its inter-token gaps deleted. Runs like that are the normal case
// for a highlighted code block (examples/tutorial) and for a terminal pane's
// blank cells, so they are written as character references, which the
// formatter reads as text and the browser reads as spaces.
func TestGridRunWhitespaceSurvivesPrettyPrinting(t *testing.T) {
	n := core.TextGrid([]core.GridRow{{
		{Text: "    ", Bg: "#101010"}, // an indent, and a run with no glyphs at all
		{Text: "func", Fg: "#CC7832"},
		{Text: " "}, // the gap between two coloured tokens
		{Text: "Main", Fg: "#FFC66D"},
	}}).Render(core.NewContext())
	out := ExportHTML(n)

	for _, want := range []string{
		`<span style="white-space:pre; background:#101010">&#32;&#32;&#32;&#32;</span>`,
		`<span style="white-space:pre">&#32;</span>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("export lacks %q — a whitespace-only run was dropped:\n%s", want, out)
		}
	}
	// The document's own formatting must not land inside a row, where it
	// would read as a line break: the row is nowrap and its trailing
	// indentation collapses, but the run's spaces are the ones that count.
	if strings.Contains(out, `<span style="white-space:pre; background:#101010"></span>`) {
		t.Errorf("the indent run exported empty:\n%s", out)
	}
}

// A run with glyphs still goes through the escaping path, so the character
// references above are not a hole in the exporter's escaping guarantee: they
// are reachable only for text that is entirely white space, and only digits
// are emitted inside them.
func TestGridRunWithGlyphsIsStillEscaped(t *testing.T) {
	n := core.TextGrid([]core.GridRow{{
		{Text: `<b>&amp; " done`},
	}}).Render(core.NewContext())
	out := ExportHTML(n)
	if strings.Contains(out, "<b>") {
		t.Errorf("markup in a run reached the document live:\n%s", out)
	}
	if !strings.Contains(out, "&lt;b&gt;&amp;amp; &#34; done") {
		t.Errorf("run text is not escaped as expected:\n%s", out)
	}
}
