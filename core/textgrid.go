package core

// TextGrid is a monospace grid of styled text: a terminal pane, a log tail, a
// hex dump. Rows are given in order; each row is a run of styled spans that
// the renderer lays out in a fixed-pitch font with no wrapping.
//
//	core.TextGrid(rows, core.FontSize(12), core.Background("#000"))
//
// # Why a node type and not a Column of Text
//
// Nothing else in core can draw this. Style has no font family, so a Text
// cannot ask for a fixed pitch, and a row of Text nodes has no way to keep
// its glyphs on a cell grid across styled runs. Emulating a grid from Row and
// Text would also put every run in the node tree as its own element, which
// for an 80×24 pane at terminal diff rate is a patch stream measured in
// thousands of nodes per second.
//
// # Rows are children, so a changed row is one patch
//
// The grid renders as a container node with one GridRow child per row, and
// each row's runs are one prop on that child. The reconciler pairs children
// by index and compares props by value, so a pass that changes three rows of
// twenty-four emits three update-props patches and nothing else; an unchanged
// row costs a DeepEqual on its runs and no traffic. A renderer therefore
// repaints a row, never the grid. No caching is needed to get this; it falls
// out of the node shape.
//
// Each platform draws its own: Compose an AnnotatedString in a monospace
// Text per row, SwiftUI an AttributedString with the monospaced design, the
// browser and htmlout a <pre> of <div> rows holding <span> runs.
//
// The Style applies to the grid as a whole (FontSize, TextColor and
// Background are the ones that matter; a run's own colours override the
// grid's). Behavior props apply to the grid too, so a tap on any cell is a
// tap on the grid.
func TextGrid(rows []GridRow, props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		n := leafNode(ctx, "TextGrid", Style{}, map[string]any{"rows": len(rows)}, props)
		n.Children = make([]*Node, len(rows))
		for i, row := range rows {
			n.Children[i] = gridRowNode(row)
		}
		return n
	})
}

// GridRow is one row of a TextGrid: its runs, in order, left to right.
type GridRow []GridRun

// GridRun is a span of one row drawn in one style. Text is the glyphs; Fg
// and Bg are CSS colours ("#rrggbb"), each "" to inherit the grid's; Attr
// is a bitmask of the Grid* attributes.
//
// The json tags are the wire shape the renderers read. They are short
// because a full pane is a few thousand runs a second at diff rate, and the
// key names are the part of a run that is not content.
type GridRun struct {
	Text string `json:"t"`
	Fg   string `json:"fg,omitempty"`
	Bg   string `json:"bg,omitempty"`
	Attr int    `json:"a,omitempty"`
}

// GridRun attribute bits. A renderer without a native spelling for one may
// drop it (there is no dim on the web's font-weight scale, say, so the DOM
// targets fake it with opacity), but must never fail the row.
const (
	GridBold      = 1 << iota // heavier weight
	GridDim                   // reduced intensity
	GridItalic                // slanted
	GridUnderline             // a line below
	GridStrike                // a line through
)

// gridRowNode is one row as the tree carries it. Rows are never built by
// app code directly and take no props of their own; everything about the
// row is in its runs.
func gridRowNode(row GridRow) *Node {
	if row == nil {
		// A nil row and an empty row must compare equal across passes, or a
		// row that goes from "no runs" to "no runs" would patch every time.
		row = GridRow{}
	}
	return &Node{
		Type:  "GridRow",
		Props: map[string]any{"runs": row},
	}
}
