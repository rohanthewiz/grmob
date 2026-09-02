package core

import "testing"

func TestTextGridRendersOneRowChildPerRow(t *testing.T) {
	ctx := NewContext()
	rows := []GridRow{
		{{Text: "$ ls"}},
		{{Text: "a.go", Fg: "#00ff00"}, {Text: "  b.go", Attr: GridBold | GridUnderline}},
		nil,
	}
	n := TextGrid(rows, FontSize(12)).Render(ctx)
	if n.Type != "TextGrid" {
		t.Fatalf("type = %q", n.Type)
	}
	if n.Style == nil || n.Style.FontSize != 12 {
		t.Errorf("style props did not reach the grid: %+v", n.Style)
	}
	if n.Props["rows"] != 3 || len(n.Children) != 3 {
		t.Fatalf("rows = %v, children = %d", n.Props["rows"], len(n.Children))
	}
	for i, c := range n.Children {
		if c.Type != "GridRow" {
			t.Errorf("child %d type = %q", i, c.Type)
		}
	}
	runs, ok := n.Children[1].Props["runs"].(GridRow)
	if !ok || len(runs) != 2 || runs[1].Attr != GridBold|GridUnderline {
		t.Errorf("row 1 runs = %#v", n.Children[1].Props["runs"])
	}
	// A nil row is carried as an empty one, so it compares equal to an empty
	// row on the next pass rather than patching forever.
	if r, ok := n.Children[2].Props["runs"].(GridRow); !ok || r == nil || len(r) != 0 {
		t.Errorf("nil row carried as %#v, want an empty GridRow", n.Children[2].Props["runs"])
	}
}

func TestGridAttrBitsAreDistinct(t *testing.T) {
	seen := map[int]bool{}
	for _, a := range []int{GridBold, GridDim, GridItalic, GridUnderline, GridStrike} {
		if a == 0 || a&(a-1) != 0 || seen[a] {
			t.Errorf("attr %d is not a distinct single bit", a)
		}
		seen[a] = true
	}
}
