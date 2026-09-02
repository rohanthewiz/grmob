package reconcile

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The property core.TextGrid is built on: a changed row is one update-props
// patch on that row, and nothing else moves. This is what lets a terminal
// pane stream diffs through the reconciler without re-sending the grid, and
// it is a consequence of children pairing by index and props comparing by
// value rather than of anything TextGrid does itself — so it is pinned here,
// beside the rules it depends on.
func TestTextGridChangedRowIsOnePatch(t *testing.T) {
	grid := func(second string) *core.Node {
		return core.TextGrid([]core.GridRow{
			{{Text: "row 0"}},
			{{Text: second, Fg: "#ff0000"}},
			{{Text: "row 2", Attr: core.GridBold}},
		}).Render(core.NewContext())
	}
	patches := Diff(grid("before"), grid("after"), "root")
	if len(patches) != 1 {
		t.Fatalf("got %d patches, want one for the changed row:\n%+v", len(patches), patches)
	}
	p := patches[0]
	if p.Type != "update-props" || p.TargetID != "root/1" {
		t.Errorf("patch = %+v, want update-props at root/1", p)
	}
	props, ok := p.Changes.(map[string]any)
	if !ok {
		t.Fatalf("changes = %T, want the row's props map", p.Changes)
	}
	runs := props["runs"].(core.GridRow)
	if runs[0].Text != "after" {
		t.Errorf("patched runs = %+v", runs)
	}

	// And an unchanged grid is silent, including its empty rows.
	if extra := Diff(grid("same"), grid("same"), "root"); len(extra) != 0 {
		t.Errorf("an unchanged grid emitted %d patches: %+v", len(extra), extra)
	}
}

// Growing the grid by a row adds that row; shrinking removes the tail.
func TestTextGridResizeAddsAndRemovesRows(t *testing.T) {
	rows := func(n int) *core.Node {
		out := make([]core.GridRow, n)
		for i := range out {
			out[i] = core.GridRow{{Text: "x"}}
		}
		return core.TextGrid(out).Render(core.NewContext())
	}
	if p := Diff(rows(2), rows(3), "root"); len(p) != 2 || p[1].Type != "add-child" {
		// The rows count prop changes too, hence two patches.
		t.Errorf("grow: %+v", p)
	}
	if p := Diff(rows(3), rows(2), "root"); len(p) != 2 || p[1].Type != "remove-child" {
		t.Errorf("shrink: %+v", p)
	}
}
