package tutorial

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// rowsText reassembles a highlighted snippet from its rows and runs. Every
// test below leans on this, because the invariant that matters most about a
// highlighter is that it is a *colouring* — it may not add, drop or reorder a
// single byte of what the author wrote.
func rowsText(rows []core.GridRow) string {
	lines := make([]string, len(rows))
	for i, row := range rows {
		var b strings.Builder
		for _, run := range row {
			b.WriteString(run.Text)
		}
		lines[i] = b.String()
	}
	return strings.Join(lines, "\n")
}

// tutorialSnippets reads every literal Go snippet the tutorial ships, by
// parsing this package's own sources and pulling the argument out of each
// codeBlock(`...`) call.
//
// Reading the sources rather than rendering the lessons is what makes this
// total. A lesson's Body only runs when its screen is built, several are
// behind demo state, and a rendered TextGrid has already been through the
// highlighter — so a test driven off the rendered tree would be checking the
// output against itself. The call sites are the corpus.
//
// Calls whose argument is not a literal are skipped on purpose: a few demos
// build their code block with Sprintf from live state (chapter 4 prints the
// button literal it just built), and those strings do not exist until the
// demo runs. They are covered from the other side, by the lesson tests that
// assert on the printed literal through hasTextContaining.
func tutorialSnippets(t *testing.T) map[string]string {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, 0)
	if err != nil {
		t.Fatalf("parsing the tutorial package: %v", err)
	}
	out := map[string]string{}
	for _, pkg := range pkgs {
		for path, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if id, ok := call.Fun.(*ast.Ident); !ok || id.Name != "codeBlock" {
					return true
				}
				if len(call.Args) != 1 {
					return true
				}
				lit, ok := call.Args[0].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				code, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Errorf("%s: codeBlock argument is not an unquotable string: %v", path, err)
					return true
				}
				out[fset.Position(lit.Pos()).String()] = code
				return true
			})
		}
	}
	return out
}

// Every snippet the tutorial ships lexes cleanly, so none of them is served
// through highlightGo's plain fallback.
//
// This is the test that makes the fallback's silence acceptable. Falling back
// is the right behavior for a snippet the scanner cannot read — half-coloured
// code is worse than uncoloured code — but a silent fallback is also a
// perfect hiding place: a snippet with a stray character would simply lose
// its colours, and nothing would say so. Here it is a failure that names the
// file and line.
func TestEveryTutorialSnippetHighlights(t *testing.T) {
	snippets := tutorialSnippets(t)
	// A floor, not a count. The corpus grows with every lesson, so pinning
	// the exact number would make this test an edit on every chapter; a floor
	// catches the failure that actually matters, which is the extraction
	// above quietly matching nothing and passing over an empty map.
	if len(snippets) < 30 {
		t.Fatalf("found %d literal snippets, expected the tutorial's whole corpus — "+
			"the extraction above has probably stopped matching", len(snippets))
	}
	for pos, code := range snippets {
		src := strings.Trim(code, "\n")
		if _, ok := scanGo(src); !ok {
			t.Errorf("%s: snippet does not lex cleanly, so it renders unhighlighted:\n%s", pos, src)
		}
	}
}

// Highlighting is a colouring and nothing more: the glyphs that come out are
// exactly the glyphs that went in, line for line. Checked over the real
// corpus rather than a sample, because the ways this could go wrong — an
// off-by-one in a token span, a dropped auto-inserted semicolon taking a
// newline with it, a multi-line raw string mis-sliced — are all data
// dependent.
func TestHighlightPreservesEverySourceByte(t *testing.T) {
	for pos, code := range tutorialSnippets(t) {
		src := strings.Trim(code, "\n")
		if got := rowsText(highlightGo(code)); got != src {
			t.Errorf("%s: highlighting changed the source:\nwant %q\ngot  %q", pos, src, got)
		}
	}
}

// One row per line, including blank ones. A blank line inside a snippet
// separates two thoughts, and dropping it would reflow the whole block; the
// row for it is empty, which every renderer draws as one line of height.
func TestHighlightKeepsOneRowPerLineIncludingBlanks(t *testing.T) {
	rows := highlightGo("a := 1\n\nb := 2")
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d: %#v", len(rows), rows)
	}
	if len(rows[1]) != 0 {
		t.Errorf("the blank line should be an empty row, got %#v", rows[1])
	}
}

// Leading spaces survive. This is the whole reason the old code block
// substituted non-breaking spaces: indentation is the structure of a Go
// snippet, and a renderer that collapses it turns a nested call into a flat
// one. A grid row preserves its spaces, so the source's own indent crosses
// the wire untouched.
func TestHighlightPreservesIndentation(t *testing.T) {
	rows := highlightGo("core.Row(\n    core.Gap(8),\n)")
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d", len(rows))
	}
	if got := rowsText(rows[1:2]); !strings.HasPrefix(got, "    ") {
		t.Errorf("indent lost on the nested line: %q", got)
	}
}

// runAt is the run covering the first occurrence of sub, or a zero run when
// no single run holds it. Tests below assert on a token's colour, and a
// token that has been split across runs is itself a failure — the runs are
// meant to be maximal.
func runAt(rows []core.GridRow, sub string) core.GridRun {
	for _, row := range rows {
		for _, run := range row {
			if strings.Contains(run.Text, sub) {
				return run
			}
		}
	}
	return core.GridRun{}
}

// The classification, token class by token class, against the Darcula values
// the palette declares. Named constants on both sides: the point is that each
// kind of token gets *its own* colour, not that any particular hex is right,
// and a test spelling the hexes again would only pin the typing.
func TestHighlightClassifiesGoTokens(t *testing.T) {
	rows := highlightGo(`// a comment
func Profile(ctx *core.Context) core.View {
    n := 42
    return core.Text("hi", core.FontSize(20))
}`)

	cases := []struct {
		sub  string
		fg   string
		attr int
		why  string
	}{
		{"// a comment", darculaLineCmt, core.GridItalic, "a line comment"},
		{"func", darculaKeyword, 0, "a keyword"},
		{"return", darculaKeyword, 0, "a keyword"},
		{"Profile", darculaFunc, 0, "the declared function's name"},
		{"Text", darculaFunc, 0, "a called function's name"},
		{`"hi"`, darculaString, 0, "a string literal"},
		{"42", darculaNumber, 0, "a number literal"},
		{"20", darculaNumber, 0, "a number literal"},
		{"core", "", 0, "a package qualifier, which a lexer cannot tell from a variable"},
		{"ctx", "", 0, "an ordinary identifier"},
	}
	for _, c := range cases {
		run := runAt(rows, c.sub)
		if run.Text == "" {
			t.Errorf("%s: %q is not covered by any run", c.why, c.sub)
			continue
		}
		if run.Fg != c.fg {
			t.Errorf("%s: %q is %q, want %q", c.why, c.sub, run.Fg, c.fg)
		}
		if run.Attr != c.attr {
			t.Errorf("%s: %q has attrs %d, want %d", c.why, c.sub, run.Attr, c.attr)
		}
	}
}

// Go's universe block reads as part of the language, so it takes the keyword
// colour — including the ones that are also called like functions, which is
// why predeclared is consulted before the followed-by-a-paren rule.
func TestPredeclaredIdentifiersTakeTheKeywordColor(t *testing.T) {
	rows := highlightGo(`var s string = ""
if len(s) == 0 && s != nil {
    s = make([]byte, 4)
}`)
	for _, sub := range []string{"string", "len", "nil", "make", "byte"} {
		if got := runAt(rows, sub).Fg; got != darculaKeyword {
			t.Errorf("%q is %q, want the keyword colour %q", sub, got, darculaKeyword)
		}
	}
}

// A `//` inside a string literal is part of the string. This is the case that
// a regexp-based highlighter gets wrong and go/scanner cannot: the token is a
// STRING, so the rest of the line is code, not a comment.
func TestCommentMarkerInsideAStringIsNotAComment(t *testing.T) {
	rows := highlightGo(`u := "https://example.com" // the real comment`)
	if got := runAt(rows, `"https://example.com"`).Fg; got != darculaString {
		t.Errorf("the URL is %q, want the string colour %q", got, darculaString)
	}
	if got := runAt(rows, "// the real comment").Fg; got != darculaLineCmt {
		t.Errorf("the trailing comment is %q, want the comment colour %q", got, darculaLineCmt)
	}
}

// A block comment spans lines, and each line gets the comment colour for its
// own portion. The per-source-byte class array is what makes this fall out
// rather than needing a case of its own.
func TestBlockCommentIsColoredOnEveryLineItCovers(t *testing.T) {
	rows := highlightGo("/* one\n   two */\nx := 1")
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d", len(rows))
	}
	for i, want := range []string{darculaDocCmt, darculaDocCmt, ""} {
		if got := rows[i][0].Fg; got != want {
			t.Errorf("row %d starts %q, want %q", i, got, want)
		}
	}
}

// A snippet the scanner cannot read is returned with no colours at all, not
// with the colours it managed to work out before losing its place. Half a
// highlighting is misinformation in a document teaching the syntax.
//
// The unterminated string is the canonical case: everything after the opening
// quote is one unclosed literal as far as the scanner is concerned, so any
// partial colouring would tint real code as string.
func TestAnUnlexableSnippetIsReturnedPlain(t *testing.T) {
	src := "x := \"unterminated\ny := 1"
	rows := highlightGo(src)
	if got := rowsText(rows); got != src {
		t.Fatalf("the fallback changed the source:\nwant %q\ngot  %q", src, got)
	}
	for i, row := range rows {
		for _, run := range row {
			if run.Fg != "" || run.Attr != 0 {
				t.Errorf("row %d run %q carries colour %q/attr %d; the fallback must be total",
					i, run.Text, run.Fg, run.Attr)
			}
		}
	}
}

// Runs are maximal: consecutive bytes of one class are one run, never
// several. This is what keeps a lesson screen a few hundred runs instead of a
// few thousand — the grid's runs cross the same wire the reconciler uses for
// everything else — and it is also what lets runAt above assert on a whole
// token.
func TestRunsAreMaximal(t *testing.T) {
	for pos, code := range tutorialSnippets(t) {
		for i, row := range highlightGo(code) {
			for j := 1; j < len(row); j++ {
				prev, cur := row[j-1], row[j]
				if prev.Fg == cur.Fg && prev.Attr == cur.Attr {
					t.Errorf("%s row %d: runs %d and %d share style %q/%d and should be one run",
						pos, i, j-1, j, cur.Fg, cur.Attr)
				}
			}
		}
	}
}
