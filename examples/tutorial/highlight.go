// Go syntax highlighting for the tutorial's code blocks.
//
// The lexer is go/scanner from the standard library, not a set of regexps.
// That is the whole design decision here and it buys three things a
// hand-rolled tokenizer would each have to earn separately: the snippets are
// real Go, so the only tokenizer guaranteed to agree with the reader's own
// editor is the compiler's; string literals, rune literals and block comments
// stop being special cases (a `//` inside a string is a string, and the
// scanner knows that without being told); and the token set cannot drift as
// the language grows.
//
// It is lexical only. There is no type information here and no parse tree, so
// this file colours what a token *is*, never what it means — see classify for
// the one place that looks past a single token, and for what is deliberately
// left uncoloured because a lexer cannot know it.
package tutorial

import (
	"go/scanner"
	"go/token"
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// The Darcula palette, from JetBrains' scheme of the same name. These are the
// scheme's own hex values rather than an approximation, so a reader who lives
// in GoLand or IntelliJ sees the tutorial's snippets in the colours their
// editor already uses for the same tokens.
//
// Held apart from the codeBg/codeInk pair in widgets.go only by role: those
// two are the surface and its default ink (the grid's Style), and these are
// the per-run overrides. Together they are the tutorial's one hard-coded
// palette — everything else in the chrome reads theme roles, because an
// editor-dark code surface has to read as code under a light theme and a dark
// one alike, and the palette has no role that means that.
const (
	darculaKeyword = "#CC7832" // keywords, and the predeclared identifiers below
	darculaString  = "#6A8759" // string and rune literals
	darculaNumber  = "#6897BB" // integer, float and imaginary literals
	darculaLineCmt = "#808080" // //-comments
	darculaDocCmt  = "#629755" // /* */-comments, which Darcula greens
	darculaFunc    = "#FFC66D" // an identifier being declared or called
)

// The token classes this file distinguishes. A byte per class rather than a
// string colour, because the classification is stored one byte per *source
// byte* (see highlightGo) and colours would make that array eight times the
// size of the snippet for no gain.
//
// classPlain is zero on purpose: it is the class of every byte no token
// claims — the whitespace between tokens, and every operator and delimiter,
// which Darcula leaves in the default ink. So the array needs no initializing
// pass, and a byte the scanner never mentions is already right.
const (
	classPlain byte = iota
	classKeyword
	classString
	classNumber
	classLineCmt
	classDocCmt
	classFunc
)

// classColors maps a class onto its Darcula colour and the GridRun attribute
// bits it carries. classPlain has no row: an empty Fg means "inherit the
// grid's own TextColor", which is exactly what default ink is, and writing
// codeInk into every plain run would put the same colour on the wire
// thousands of times per screen.
var classColors = map[byte]struct {
	fg   string
	attr int
}{
	classKeyword: {darculaKeyword, 0},
	classString:  {darculaString, 0},
	classNumber:  {darculaNumber, 0},
	// Italic on both comment classes, as Darcula draws them. Every renderer
	// has a spelling for it (font-style on the two DOM targets, FontStyle
	// .Italic in Compose, .italic() in SwiftUI), so this is not a decoration
	// that survives on some targets and not others.
	classLineCmt: {darculaLineCmt, core.GridItalic},
	classDocCmt:  {darculaDocCmt, core.GridItalic},
	classFunc:    {darculaFunc, 0},
}

// predeclared is Go's universe block — the identifiers that are always in
// scope and are not keywords. Darcula gives them the keyword colour, which is
// also the honest reading: `string` and `len` are as much part of the language
// as `func` is, and the tutorial's snippets are full of both.
//
// Spelled out rather than derived, because there is no exported list of them
// in the standard library (go/types has the set, but as *types.Basic and
// builtin objects in a Universe scope, which is a type-checker dependency for
// a lexical highlighter). Kept in the spec's own order so a future addition to
// the universe block is a one-line append in the obvious place.
var predeclared = map[string]bool{
	// Types.
	"any": true, "bool": true, "byte": true, "comparable": true,
	"complex64": true, "complex128": true, "error": true, "float32": true,
	"float64": true, "int": true, "int8": true, "int16": true, "int32": true,
	"int64": true, "rune": true, "string": true, "uint": true, "uint8": true,
	"uint16": true, "uint32": true, "uint64": true, "uintptr": true,
	// Constants, and the zero value.
	"true": true, "false": true, "iota": true, "nil": true,
	// Functions.
	"append": true, "cap": true, "clear": true, "close": true,
	"complex": true, "copy": true, "delete": true, "imag": true, "len": true,
	"make": true, "max": true, "min": true, "new": true, "panic": true,
	"print": true, "println": true, "real": true, "recover": true,
}

// tokenSpan is one scanned token with the source it covers. The scanner hands
// back a position and a literal; this pairs them so the classification pass
// can look at a token's neighbour, which is the one thing a single Scan call
// cannot show.
type tokenSpan struct {
	off  int // byte offset into the snippet
	tok  token.Token
	text string // exactly src[off : off+len(text)], checked in scanGo
}

// highlightGo turns a Go snippet into TextGrid rows, one row per line, each a
// sequence of coloured runs.
//
// # Why rows of runs and not a Column of Text
//
// A code line is several colours, so the unit that carries a colour has to be
// smaller than a line. core.TextGrid is the node type built for exactly that
// shape — a row is a list of styled runs — and it brings two things this
// tutorial had been working around by hand:
//
//   - It is monospace on every target by construction (a <pre> on the two DOM
//     renderers, FontFamily.Monospace in Compose, .monospaced in SwiftUI).
//     Style has no font-family prop, so nothing else in core can ask for a
//     fixed pitch, and the old code block said so in its own comment.
//   - Its rows do not wrap and its chassis scrolls sideways, which is what
//     the old block spelled out per-line with WhiteSpace("nowrap") and an
//     Overflow on the column.
//
// It also retires the NBSP substitution the old block needed: indentation
// survives because `white-space: pre` and a Compose/SwiftUI Text preserve
// leading spaces, where a <span> in ordinary flow collapses them.
//
// # The fallback
//
// A snippet the scanner cannot read cleanly is returned unhighlighted rather
// than half-coloured. A lexer that has lost its place produces colours that
// are worse than none — a run of code tinted as a string is actively
// misleading in a document whose job is to teach the syntax — so the failure
// is total and silent, and the reader gets the plain block they used to have.
// TestEveryTutorialSnippetHighlights holds every snippet in the tutorial to
// the clean path, so the fallback is proven to be unreached in-tree rather
// than merely believed to be.
func highlightGo(code string) []core.GridRow {
	src := strings.Trim(code, "\n")
	classes, ok := scanGo(src)
	if !ok {
		classes = make([]byte, len(src)) // all classPlain
	}
	return rowsOf(src, classes)
}

// scanGo lexes the snippet and returns one class byte per source byte, or
// ok=false if the scan cannot be trusted.
//
// # Why a byte-per-byte array and not a list of spans
//
// The output is rows, and a row is a line, so the classification has to be
// sliced at every newline — and a block comment or a raw string crosses those
// boundaries. Storing the class per byte makes that split a slice expression
// with no case analysis at all: the portion of a multi-line comment on line 3
// is just classes[start:end] for line 3. Slicing at a newline can never split
// a rune, because a newline is a single ASCII byte and every byte of a
// multi-byte rune belongs to the same token.
func scanGo(src string) (classes []byte, ok bool) {
	toks, ok := scanTokens(src)
	if !ok {
		return nil, false
	}
	classes = make([]byte, len(src))
	for i, t := range toks {
		var next token.Token
		if i+1 < len(toks) {
			next = toks[i+1].tok
		}
		class := classify(t, next)
		if class == classPlain {
			continue // already zero
		}
		for j := t.off; j < t.off+len(t.text); j++ {
			classes[j] = class
		}
	}
	return classes, true
}

// scanTokens runs go/scanner over the snippet.
//
// Two things make this different from scanning a whole file. First, a snippet
// is a *fragment* — several of the tutorial's are a bare expression or a few
// statements with no package clause — but that only matters to a parser; the
// scanner is happy to tokenize any text, which is precisely why the lexical
// layer is the right one to build on.
//
// Second, the scanner's positions have to be converted back into byte offsets
// into the string we were handed, and that conversion is the one place this
// could silently go wrong. So every token is checked against the source it
// claims to cover before it is trusted (see the identity check below); a
// single mismatch abandons the whole snippet rather than colouring from an
// offset that has drifted.
func scanTokens(src string) ([]tokenSpan, bool) {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))

	// Any lexical error at all disqualifies the snippet. The scanner recovers
	// and keeps going after most of them, but "recovered" is not the same as
	// "understood": an unterminated string makes every token after it suspect,
	// and this file's only job is to be right about which bytes are what.
	clean := true
	var s scanner.Scanner
	s.Init(file, []byte(src), func(token.Position, string) { clean = false }, scanner.ScanComments)

	var toks []tokenSpan
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		// Automatic semicolon insertion: the scanner reports a SEMICOLON at
		// the end of a line that ends a statement, with "\n" as its literal
		// because there is no semicolon in the source to point at. It covers
		// no bytes, so recording it would paint a class onto the newline (or
		// past the end of the line, at EOF).
		if tok == token.SEMICOLON && lit != ";" {
			continue
		}
		// Literals and comments carry their text in lit; operators and
		// delimiters carry it in the token's own spelling.
		text := lit
		if text == "" {
			text = tok.String()
		}
		off := file.Offset(pos)
		// The identity check. If a token does not cover exactly the source it
		// says it does, the offsets are not what this code assumes and every
		// class after it would land on the wrong bytes.
		if off < 0 || off+len(text) > len(src) || src[off:off+len(text)] != text {
			return nil, false
		}
		toks = append(toks, tokenSpan{off: off, tok: tok, text: text})
	}
	return toks, clean
}

// classify decides a token's colour. next is the token that follows it, which
// is needed for exactly one rule and is token.ILLEGAL's zero value at the end
// of the snippet.
//
// # What is deliberately not coloured
//
// A lexer knows what a token is and nothing about what it refers to, and this
// function does not pretend otherwise:
//
//   - A package qualifier stays default ink. In `core.Text(...)` only `Text`
//     is coloured; `core` is an identifier that happens to be a package here
//     and could be a variable in the next snippet, and there is no lexical
//     difference between the two.
//   - Struct field names in composite literals stay default ink, though
//     Darcula would purple them. `Scroll: true` and a `case x:` label and a
//     map key are the same three tokens to a scanner, so the rule would have
//     to guess, and a wrong guess here would tint ordinary values.
//   - Type names stay default ink for the same reason, apart from the
//     predeclared ones, which are known without any scope analysis.
func classify(t tokenSpan, next token.Token) byte {
	switch {
	case t.tok == token.COMMENT:
		if strings.HasPrefix(t.text, "//") {
			return classLineCmt
		}
		return classDocCmt
	case t.tok == token.STRING || t.tok == token.CHAR:
		return classString
	case t.tok == token.INT || t.tok == token.FLOAT || t.tok == token.IMAG:
		return classNumber
	case t.tok.IsKeyword():
		return classKeyword
	case t.tok == token.IDENT:
		// Predeclared first: `len`, `make` and `string` are followed by `(`
		// as often as not, and they read as part of the language rather than
		// as something the snippet defined.
		if predeclared[t.text] {
			return classKeyword
		}
		// The one lookahead rule: an identifier immediately followed by an
		// open paren is being called or declared. It is a purely lexical
		// stand-in for "function name", and it is right for both halves of
		// what the tutorial's snippets are made of — `func Profile(` at a
		// declaration and `core.Text(` at a call — because a declaration and
		// a call have the same shape at this level. A conversion like
		// `MyType(x)` is the known false positive; it is rare in these
		// snippets and reads as a call to most people anyway.
		if next == token.LPAREN {
			return classFunc
		}
	}
	return classPlain
}

// rowsOf slices the snippet and its class array into one GridRow per line.
//
// A line with no bytes yields a nil row, which core.TextGrid normalizes to an
// empty GridRow, and every renderer draws that as one line's worth of height
// (htmlout's gridRowChassis sets a min-height, the Compose and SwiftUI rows
// append a blank). So a blank line between two statements keeps its blank
// line without the single-space filler the old code block needed.
func rowsOf(src string, classes []byte) []core.GridRow {
	var rows []core.GridRow
	start := 0
	for i := 0; i <= len(src); i++ {
		if i < len(src) && src[i] != '\n' {
			continue
		}
		rows = append(rows, runsOf(src[start:i], classes[start:i]))
		start = i + 1
	}
	return rows
}

// runsOf groups a line's bytes into maximal runs of one class. Maximal
// matters: `core.Text(` is three runs and not eleven, and a full lesson screen
// is a few hundred runs rather than a few thousand — which is the difference
// between one patch-sized payload and one that is not, on the same wire the
// reconciler uses for everything else.
func runsOf(line string, classes []byte) core.GridRow {
	var row core.GridRow
	for i := 0; i < len(line); {
		j := i
		for j < len(line) && classes[j] == classes[i] {
			j++
		}
		c := classColors[classes[i]] // the zero value is classPlain's: no colour, no attrs
		row = append(row, core.GridRun{Text: line[i:j], Fg: c.fg, Attr: c.attr})
		i = j
	}
	return row
}
