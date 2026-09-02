package htmlout

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// core.TextGrid exports as a <pre> of row <div>s holding <span> runs. The
// grid's chassis pins the line height so an empty row keeps its line; a run
// carries only the declarations it set, so a plain run is a bare span.
func TestTextGridExportsRowsOfRuns(t *testing.T) {
	n := core.TextGrid([]core.GridRow{
		{{Text: "$ ls"}},
		{{Text: "a.go", Fg: "#00ff00", Bg: "#000000"}, {Text: " b.go", Attr: core.GridBold | core.GridUnderline | core.GridStrike}},
		{},
		{{Text: "<script>", Attr: core.GridDim | core.GridItalic}},
	}, core.FontSize(12)).Render(core.NewContext())
	out := ExportHTML(n)

	for _, want := range []string{
		`<pre style="margin:0; line-height:1.2; white-space:pre; overflow-x:auto; font-size:12px">`,
		`<span>$ ls</span>`,
		`<span style="color:#00ff00; background:#000000">a.go</span>`,
		`<span style="font-weight:700; text-decoration:underline line-through"> b.go</span>`,
		`<span style="opacity:0.6; font-style:italic">&lt;script&gt;</span>`,
		`min-height:1.2em`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("export lacks %q:\n%s", want, out)
		}
	}
	// Four rows, one of them empty, all present.
	if got := strings.Count(out, `<div style="min-height:1.2em"`); got != 4 {
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
