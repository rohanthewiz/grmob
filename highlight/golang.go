// Go syntax highlighting.
//
// The lexer is go/scanner from the standard library, not a set of regexps.
// That is the whole design decision here and it buys three things a
// hand-rolled tokenizer would each have to earn separately: the input is real
// Go, so the only tokenizer guaranteed to agree with the reader's own editor
// is the compiler's; string literals, rune literals and block comments stop
// being special cases (a `//` inside a string is a string, and the scanner
// knows that without being told); and the token set cannot drift as the
// language grows.
//
// It is lexical only. There is no type information here and no parse tree, so
// this file colours what a token *is*, never what it means — see classifyGo
// for the one place that looks past a single token, and for what is
// deliberately left uncoloured because a lexer cannot know it.
package highlight

import (
	"go/scanner"
	"go/token"
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// Go returns the Go highlighter.
//
// A function rather than an exported variable so the returned value is always
// a fresh, immutable, stateless thing: a Highlighter is held by a widget
// across renders and may be called from whatever goroutine a render pass runs
// on, and a package-level value invites someone to give it a field.
func Go() Highlighter { return goHighlighter{} }

type goHighlighter struct{}

// Rows lexes src and returns one styled row per line.
//
// # The fallback
//
// Source the scanner cannot read cleanly is returned unhighlighted rather than
// half-coloured. A lexer that has lost its place produces colours that are
// worse than none, so the failure is total and silent. This matters more now
// than it did in the tutorial: a CodeEditor's buffer is *mid-edit* most of the
// time, so an unterminated string is the normal state of a line someone is
// typing, and the honest answer for that keystroke is plain ink until the
// quote is closed.
func (goHighlighter) Rows(src string, scheme Scheme) []core.GridRow {
	classes, ok := scanGo(src)
	if !ok {
		return plainRows(src)
	}
	return rowsOf(src, classes, scheme)
}

// predeclared is Go's universe block — the identifiers that are always in
// scope and are not keywords. Darcula gives them the keyword colour, which is
// also the honest reading: `string` and `len` are as much part of the language
// as `func` is.
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
	off  int // byte offset into the source
	tok  token.Token
	text string // exactly src[off : off+len(text)], checked in scanGoTokens
}

// scanGo lexes the source and returns one class byte per source byte, or
// ok=false if the scan cannot be trusted.
func scanGo(src string) (classes []byte, ok bool) {
	toks, ok := scanGoTokens(src)
	if !ok {
		return nil, false
	}
	classes = make([]byte, len(src))
	for i, t := range toks {
		var next token.Token
		if i+1 < len(toks) {
			next = toks[i+1].tok
		}
		class := classifyGo(t, next)
		if class == classPlain {
			continue // already zero
		}
		for j := t.off; j < t.off+len(t.text); j++ {
			classes[j] = class
		}
	}
	return classes, true
}

// scanGoTokens runs go/scanner over the source.
//
// Two things make this different from scanning a whole file. First, the input
// is often a *fragment* — a bare expression, a few statements with no package
// clause — but that only matters to a parser; the scanner is happy to tokenize
// any text, which is precisely why the lexical layer is the right one to build
// on.
//
// Second, the scanner's positions have to be converted back into byte offsets
// into the string we were handed, and that conversion is the one place this
// could silently go wrong. So every token is checked against the source it
// claims to cover before it is trusted (see the identity check below); a
// single mismatch abandons the whole input rather than colouring from an
// offset that has drifted.
func scanGoTokens(src string) ([]tokenSpan, bool) {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))

	// Any lexical error at all disqualifies the input. The scanner recovers
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

// classifyGo decides a token's colour. next is the token that follows it,
// which is needed for exactly one rule and is token.ILLEGAL's zero value at
// the end of the input.
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
func classifyGo(t tokenSpan, next token.Token) byte {
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
		// as something the source defined.
		if predeclared[t.text] {
			return classKeyword
		}
		// The one lookahead rule: an identifier immediately followed by an
		// open paren is being called or declared. It is a purely lexical
		// stand-in for "function name", and it is right for both halves of
		// what most code is made of — `func Profile(` at a declaration and
		// `core.Text(` at a call — because a declaration and a call have the
		// same shape at this level. A conversion like `MyType(x)` is the
		// known false positive; it reads as a call to most people anyway.
		if next == token.LPAREN {
			return classFunc
		}
	}
	return classPlain
}
