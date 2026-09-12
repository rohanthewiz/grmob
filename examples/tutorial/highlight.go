// The tutorial's Go syntax highlighting, which is now the highlight package's.
//
// Everything that used to be in this file — the go/scanner pass, the
// class-per-source-byte array, the Darcula palette, the row and run splitting
// — moved to github.com/rohanthewiz/grmob/highlight when a second consumer
// appeared (comps.CodeEditor colours an editable buffer with the same
// lexer). What is left here is the tutorial's own two decisions: that snippets
// are Go, and that they are drawn in Darcula.
//
// The file stays rather than being folded into widgets.go so that
// highlight_test.go still has the thing it tests next to it: those tests read
// the tutorial's own corpus out of its sources and hold every shipped snippet
// to the highlighter, which is a statement about the tutorial and not about
// the package. The package's own tests are in highlight/.
package tutorial

import (
	"strings"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/highlight"
)

// The Darcula palette, named locally so the assertions in highlight_test.go
// still read as "a keyword is the keyword colour" rather than as a column of
// hexes. They are the scheme's own fields, not copies: a change to the palette
// reaches the tutorial without a second edit, and the tests below go on
// pinning that the tutorial's rows are what they always were.
//
// The surface and its default ink are the same scheme's Bg and Ink, named by
// the code block in widgets.go; these six are the per-run overrides. One
// scheme, read in two places, rather than two palettes that agree.
var (
	darculaKeyword = highlight.Darcula.Keyword
	darculaString  = highlight.Darcula.String
	darculaNumber  = highlight.Darcula.Number
	darculaLineCmt = highlight.Darcula.Comment
	darculaDocCmt  = highlight.Darcula.DocComment
	darculaFunc    = highlight.Darcula.Func
)

// goHighlighter is the lexer every snippet goes through. One value for the
// package rather than a highlight.Go() call per code block: the highlighter is
// stateless and immutable, and a code block is rendered on every pass of every
// lesson screen.
var goHighlighter = highlight.Go()

// highlightGo turns a Go snippet into TextGrid rows, one row per line, each a
// sequence of coloured runs.
//
// The Trim is the tutorial's, not the package's, and the split matters: a
// snippet is written as a raw string literal that starts and ends with a
// newline (that is what makes the literal readable in the source), and those
// two newlines are formatting rather than content. The highlight package must
// not trim — its rows are line-for-line with its input by contract, because
// core.CodeEditor pairs row N with the host's line N — so the trimming belongs
// at the one call site that knows the newlines are decorative.
func highlightGo(code string) []core.GridRow {
	return goHighlighter.Rows(strings.Trim(code, "\n"), highlight.Darcula)
}
