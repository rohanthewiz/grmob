package htmlout

import (
	"strconv"
	"strings"

	"github.com/rohanthewiz/element"
	"github.com/rohanthewiz/grmob/core"
)

// core.CodeEditor, on the HTML side.
//
// # What a static export of an editor is
//
// It is the buffer, coloured, with its gutter — and no caret, no selection and
// no way to type, because a snapshot has no event loop. That is not a
// degradation to apologize for: an exported CodeEditor is exactly the code
// block the tutorial used to hand-roll, which is what a document wants from
// one. The callback IDs still travel as data attributes (the shared loop in
// renderNode writes data-onchange and data-onselectionchange), so a loader that
// does have an event loop — grmob-runtime.js, or anything reading that table —
// can wire the live editor out of the static document, the same upgrade path
// an exported MapView offers.
//
// # The element is a <pre>, and the rows are its direct children
//
// The same <pre> a core.TextGrid exports as, for the same reason: it is the one
// element whose default styling already says "fixed pitch, no wrapping", and
// the editor's rows *are* a TextGrid's rows — core.CodeEditor builds them with
// the same gridRowNode, so renderGridRow draws them here with no second row
// writer.
//
// The gutter sits inside that <pre> as chrome, ahead of the rows, and the rows
// stay the <pre>'s own direct children rather than gaining a wrapper. That
// shape is a contract with the WASM runtime rather than a preference: patches
// are addressed positionally, and the runtime resolves an added row's DOM slot
// as "the node index, shifted past the leading chrome". A wrapper here would
// make the two DOM targets structurally different documents for one node type,
// which is the thing wasm/verify's replay exists to prevent.
//
// # The gutter is chrome, not content
//
// It carries no data-node-path, is marked data-grmob-chrome like a TabView's
// bar, and is drawn with `user-select: none` so that selecting the buffer and
// copying it does not come away with the line numbers interleaved. Numbers
// inside the rows would be selectable, copyable and — in the live runtime —
// editable, and a buffer whose first columns are not the user's is not the
// buffer.
func renderCodeEditor(b *element.Builder, node *core.Node, attrs []string, path string) {
	lines := len(node.Children)
	gutter := node.Props["lineNumbers"] == true && lines > 0

	e := b.Ele(TagFor(node.Type), attrs...)
	if gutter {
		b.Div("style", codeGutterStyle(lines), "data-grmob-chrome", "gutter", "aria-hidden", "true").
			T(lineNumberText(lines))
	}
	for i, child := range node.Children {
		renderNode(b, child, imposed{}, childPath(path, i))
	}
	e.R()
}

// codeEditorChassis is the fixed rules of an editor box, and is the grid's own
// chassis plus the two things an overlay needs.
//
// position:relative makes the box the containing block for the gutter (and,
// in the live runtime, for the transparent <textarea> that sits over the
// rows). overflow:auto lets a buffer wider or taller than its frame scroll in
// both directions rather than spilling, which is what `no wrapping` costs and
// is the whole reason a code editor scrolls sideways where prose does not.
//
// The white-space split is the grid's, unchanged and for the grid's reasons:
// `normal` here because the only white space *between* rows is the exporter's
// own indentation, `nowrap` on each row, `pre` on each run. See
// textGridChassis for the long version.
const codeEditorChassis = "margin:0; line-height:1.2; white-space:normal; " +
	"overflow:auto; position:relative"

// codeEditorPadding is the inset the rows need to clear the gutter, and is
// written as padding on the box rather than as a margin on each row so that a
// row is the same element in an editor as it is in a grid.
//
// An absolutely positioned child is placed against the *padding* box, so the
// gutter at left:0 lands in the space this padding opens up and the rows begin
// where the padding ends. One declaration, no per-row arithmetic.
func codeEditorPadding(node *core.Node) string {
	if node.Props["lineNumbers"] != true || len(node.Children) == 0 {
		return ""
	}
	return "padding-left:" + gutterWidth(len(node.Children))
}

// gutterWidth is wide enough for the largest line number plus a column of
// breathing room, in `ch` units — the width of a "0" in the current font,
// which in a fixed-pitch box is the width of every glyph. So the gutter is
// exactly as wide as it needs to be at any font size, with no measurement.
func gutterWidth(lines int) string {
	digits := len(strconv.Itoa(lines))
	return strconv.Itoa(digits+2) + "ch"
}

// codeGutterStyle is the line-number column: out of flow at the left edge of
// the padding the box opened for it, right-aligned so the digits line up on
// their units column, unselectable, and dimmed so the buffer is what the eye
// lands on.
//
// pointer-events:none so a drag that starts over the numbers still selects
// text in the buffer behind them rather than doing nothing.
func codeGutterStyle(lines int) string {
	return "position:absolute; left:0; top:0; width:" + gutterWidth(lines) +
		"; padding-right:1ch; box-sizing:border-box; text-align:right; " +
		"white-space:pre; opacity:0.45; user-select:none; pointer-events:none"
}

// lineNumberText is the gutter's whole content: the numbers, one per line,
// separated by newlines. One text node rather than one element per line,
// because `white-space: pre` makes the newlines significant and the gutter's
// line height is the box's — so number N sits beside row N by construction
// rather than by two element lists being kept the same length.
func lineNumberText(lines int) string {
	var b strings.Builder
	for i := 1; i <= lines; i++ {
		if i > 1 {
			b.WriteByte('\n')
		}
		b.WriteString(strconv.Itoa(i))
	}
	return b.String()
}
