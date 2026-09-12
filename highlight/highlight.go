// Package highlight turns source text into core.GridRow values: one styled
// row per line, ready for a core.TextGrid or a core.CodeEditor.
//
// It was lifted out of the tutorial, where a private go/scanner highlighter
// had been colouring the lesson snippets. Two callers now want the same thing
// and neither is the tutorial — components.CodeEditor colours an editable
// buffer, and any screen that shows a config file or a payload wants the same
// rows — so the lexers live here, behind one interface, and the tutorial is a
// caller like the rest.
//
// # The shape, and why it is rows rather than spans
//
// A row is a line and a line is what both consumers address. core.TextGrid
// gives each row its own node, so a changed line is one patch; core.CodeEditor
// applies a row's decoration only when the row's text still equals the host's
// own line (the stale-line rule in core/codeeditor.go). Both depend on the
// rows being *line-for-line* with the input, which is this package's one hard
// contract:
//
//	len(Rows(src, scheme)) == strings.Count(src, "\n") + 1
//
// It holds for every lexer here, including the fallbacks, and
// TestEveryLexerIsLineForLine pins it.
//
// # The classification is lexical, and per byte
//
// Every lexer in the package works the same way: it produces one class byte
// per *source byte*, and the shared rowsOf/runsOf pair slices that array at
// the newlines. The reason is that a block comment or a raw string crosses
// line boundaries, and a per-byte array makes that split a slice expression
// with no case analysis at all — the part of a multi-line comment on line 3 is
// classes[start:end] for line 3. Slicing at a newline can never split a rune,
// because a newline is a single ASCII byte and every byte of a multi-byte rune
// belongs to the same token.
//
// A byte per class rather than a colour per byte because the array is as long
// as the source; colours would make it eight times the size for no gain, and
// the class→colour step is where a Scheme gets applied.
package highlight

import "github.com/rohanthewiz/grmob/core"

// Highlighter turns source into one styled row per line.
//
// Rows must be line-for-line with the input — see the package doc. A
// highlighter that cannot make sense of its input returns plain rows rather
// than half-coloured ones: colours that have lost their place are worse than
// no colours, because a run of code tinted as a string actively misleads.
type Highlighter interface {
	Rows(src string, scheme Scheme) []core.GridRow
}

// Scheme is the palette a highlighter paints with. Every field is a CSS
// colour ("#rrggbb"); an empty field means "inherit", which on the wire is an
// empty GridRun.Fg and in every renderer is the grid's own text colour.
//
// Ink and Bg are not used by Rows at all — a run in the default ink carries no
// colour, which is what keeps a full screen of plain code from putting the
// same hex on the wire a thousand times. They are here because the *surface*
// is part of a scheme: components.CodeEditor paints Bg behind the buffer and
// sets Ink as the grid's TextColor, so a caller who names a scheme gets one
// coherent picture rather than Darcula's token colours over the app's own
// background.
type Scheme struct {
	Keyword    string // keywords, and anything that reads as part of the language
	String     string // string, rune and character literals
	Number     string // numeric literals
	Comment    string // line comments
	DocComment string // block comments, which several schemes colour differently
	Func       string // an identifier being declared or called; a JSON object key
	Ink        string // the default foreground of the surface
	Bg         string // the surface itself
}

// Darcula is JetBrains' scheme of the same name, in its own hex values rather
// than an approximation, so a reader who lives in GoLand or IntelliJ sees a
// snippet in the colours their editor already uses for the same tokens.
//
// This is the scheme the tutorial has always used, and the byte-for-byte
// identity of its output is pinned from the tutorial side
// (TestHighlightClassifiesGoTokens and friends) as well as from here.
var Darcula = Scheme{
	Keyword:    "#CC7832",
	String:     "#6A8759",
	Number:     "#6897BB",
	Comment:    "#808080",
	DocComment: "#629755", // Darcula greens block comments where it greys line ones
	Func:       "#FFC66D",
	Ink:        "#A9B7C6", // Darcula's default foreground
	Bg:         "#2B2B2B", // Darcula's editor background
}

// Light is the same six roles chosen for a light surface. It is not a
// mechanical inversion of Darcula — inverting a hue gives a colour, not a
// legible one — but the GitHub-light family, which is the scheme most readers
// have seen on a light background.
//
// It exists because a code surface on a light theme has to read as code
// without becoming a dark rectangle in the middle of a light screen, and no
// palette *role* means "the colour of a keyword": the theme has Ink, Surface
// and the accents, and none of them is about syntax. So a scheme is a scheme,
// and components.CodeEditor picks between these two by the brightness of the
// theme's own Surface rather than by inventing token colours from roles.
var Light = Scheme{
	Keyword:    "#CF222E",
	String:     "#0A3069",
	Number:     "#0550AE",
	Comment:    "#6E7781",
	DocComment: "#6E7781",
	Func:       "#8250DF",
	Ink:        "#24292F",
	Bg:         "#F6F8FA",
}

// The token classes every lexer in this package classifies into. A lexer
// emits one of these per source byte; classColors turns them into a Scheme's
// colours and the GridRun attribute bits that go with them.
//
// classPlain is zero on purpose: it is the class of every byte no token claims
// — the whitespace between tokens, and every operator and delimiter, which
// each scheme here leaves in the default ink. So the per-byte array needs no
// initializing pass, and a byte a lexer never mentions is already right.
const (
	classPlain byte = iota
	classKeyword
	classString
	classNumber
	classLineCmt
	classDocCmt
	classFunc
)

// classStyle is one class resolved against a scheme: the colour to paint and
// the attribute bits that ride with it.
type classStyle struct {
	fg   string
	attr int
}

// classColors resolves a class against a scheme. classPlain has no entry and
// the zero classStyle is exactly right for it: an empty Fg means "inherit the
// grid's own TextColor", which is what default ink is.
//
// Italic on both comment classes, as Darcula and GitHub-light both draw them.
// Every renderer has a spelling for it (font-style on the two DOM targets,
// FontStyle.Italic in Compose, .italic() in SwiftUI), so this is not a
// decoration that survives on some targets and not others.
func classColors(s Scheme, class byte) classStyle {
	switch class {
	case classKeyword:
		return classStyle{s.Keyword, 0}
	case classString:
		return classStyle{s.String, 0}
	case classNumber:
		return classStyle{s.Number, 0}
	case classLineCmt:
		return classStyle{s.Comment, core.GridItalic}
	case classDocCmt:
		return classStyle{s.DocComment, core.GridItalic}
	case classFunc:
		return classStyle{s.Func, 0}
	}
	return classStyle{}
}

// rowsOf slices a source string and its class array into one GridRow per
// line. The two must be the same length; every lexer here allocates the array
// with make([]byte, len(src)), so they are.
//
// A line with no bytes yields a nil row, which core.TextGrid normalizes to an
// empty GridRow and every renderer draws as one line's worth of height. So a
// blank line between two statements keeps its blank line without a filler
// character.
//
// The loop runs to i == len(src) inclusive, which is what makes the row count
// strings.Count(src, "\n")+1 rather than the number of newlines: the text
// after the last newline is a line, and so is the empty text after a trailing
// one.
func rowsOf(src string, classes []byte, scheme Scheme) []core.GridRow {
	rows := make([]core.GridRow, 0, 8)
	start := 0
	for i := 0; i <= len(src); i++ {
		if i < len(src) && src[i] != '\n' {
			continue
		}
		rows = append(rows, runsOf(src[start:i], classes[start:i], scheme))
		start = i + 1
	}
	return rows
}

// runsOf groups a line's bytes into maximal runs of one class.
//
// Maximal matters: `core.Text(` is three runs and not eleven, and a full
// screen of code is a few hundred runs rather than a few thousand — which is
// the difference between one patch-sized payload and one that is not, on the
// same wire the reconciler uses for everything else.
func runsOf(line string, classes []byte, scheme Scheme) core.GridRow {
	var row core.GridRow
	for i := 0; i < len(line); {
		j := i
		for j < len(line) && classes[j] == classes[i] {
			j++
		}
		c := classColors(scheme, classes[i])
		row = append(row, core.GridRun{Text: line[i:j], Fg: c.fg, Attr: c.attr})
		i = j
	}
	return row
}

// plainHighlighter is the no-colour lexer: one run per line, in the default
// ink. It is also every other lexer's fallback, which is why it is a type
// rather than a function — a lexer that gives up returns plainRows and the
// caller cannot tell the difference from the outside.
type plainHighlighter struct{}

func (plainHighlighter) Rows(src string, scheme Scheme) []core.GridRow {
	return plainRows(src)
}

// Plain returns a Highlighter that colours nothing: one run per line, in the
// grid's own ink. It is what ForLanguage answers for a language nothing here
// lexes, and it is a useful value in its own right — a CodeEditor over a text
// file still wants the monospace grid, the gutter and the tab handling.
func Plain() Highlighter { return plainHighlighter{} }

// plainRows is the uncoloured rows of src: the shared slicing over an
// all-classPlain array.
//
// Allocating the zero array rather than special-casing the split keeps one
// implementation of the line-for-line contract in the package. The array is
// len(src) bytes and is discarded immediately; that is cheaper than a second
// row-splitting loop that could disagree with the first.
func plainRows(src string) []core.GridRow {
	return rowsOf(src, make([]byte, len(src)), Scheme{})
}

// ForLanguage returns the Highlighter for a language name, or Plain for one
// this package does not lex.
//
// The names are the ones a file extension or a Markdown fence would give:
// "go", "json". Matching is exact and lowercase — a caller with a filename
// resolves it to a language first, because a path is not a language ("go.mod"
// is not Go) and this package has no business guessing.
//
// An unknown name is deliberately not an error. The caller is a widget with a
// Language field, and a screen whose editor mis-spells its language should
// show uncoloured code, not fail to render.
func ForLanguage(name string) Highlighter {
	switch name {
	case "go":
		return Go()
	case "json":
		return JSON()
	}
	return Plain()
}
