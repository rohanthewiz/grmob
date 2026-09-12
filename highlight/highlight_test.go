package highlight

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// rowsText reassembles a highlighted source from its rows and runs.
//
// Most of what follows leans on it, because the invariant that matters most
// about a highlighter is that it is a *colouring*: it may not add, drop or
// reorder a single byte of what the author wrote. A highlighter that fails
// this is not producing wrong colours, it is producing wrong text.
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

// lexers is the census this package's shared contracts are checked against.
// A new lexer is added here and inherits every property test below; that is
// the point of the list existing rather than each test naming Go() and JSON().
var lexers = map[string]Highlighter{
	"go":    Go(),
	"json":  JSON(),
	"plain": Plain(),
}

// The sources every lexer is run over. They are deliberately not all valid in
// every language — a Go lexer fed JSON and a JSON lexer fed Go both have to
// obey the line-for-line contract, because that is exactly the situation a
// mis-set Language field produces on a real screen.
var corpus = map[string]string{
	"empty":            "",
	"one line":         "x",
	"no trailing nl":   "a\nb",
	"trailing nl":      "a\nb\n",
	"blank lines":      "a\n\n\nb",
	"only newlines":    "\n\n",
	"leading blank":    "\nx",
	"go":               "package main\n\nfunc main() {\n\tprintln(\"hi\")\n}\n",
	"json":             "{\n  \"a\": [1, 2],\n  \"b\": null\n}\n",
	"unterminated str": "x := \"open\nnext line",
	"block comment":    "/* one\n   two */\nx := 1",
	"multibyte":        "s := \"héllo → wörld\"\nt := '→'",
	"crlf":             "a\r\nb\r\n",
	"tabs and spaces":  "\t  \tif x {\n\t\treturn\n\t}",
}

// The line-for-line contract, which both consumers depend on: core.TextGrid
// pairs rows by index, and core.CodeEditor's stale-line rule compares row N
// with the host's line N. A lexer that returned a different number of rows
// than the source has lines would silently decorate the wrong lines.
func TestEveryLexerIsLineForLine(t *testing.T) {
	for lexName, lex := range lexers {
		for srcName, src := range corpus {
			want := strings.Count(src, "\n") + 1
			if got := len(lex.Rows(src, Darcula)); got != want {
				t.Errorf("%s.Rows(%q): %d rows, want %d", lexName, srcName, got, want)
			}
		}
	}
}

// The other half of the same contract, and the stronger claim: the rows are
// not merely the right *count*, they are the right text.
func TestEveryLexerPreservesEverySourceByte(t *testing.T) {
	for lexName, lex := range lexers {
		for srcName, src := range corpus {
			if got := rowsText(lex.Rows(src, Darcula)); got != src {
				t.Errorf("%s.Rows(%s) round trip:\n got %q\nwant %q", lexName, srcName, got, src)
			}
		}
	}
}

// Runs must be maximal: adjacent runs of one class are one run. This is the
// wire-size property — `core.Text(` is three runs and not eleven — and it is
// checked as a *local* rule (no two neighbours agree on both colour and
// attributes) rather than by counting, so it holds for every lexer and every
// input rather than for one hand-picked line.
func TestRunsAreMaximal(t *testing.T) {
	for lexName, lex := range lexers {
		for srcName, src := range corpus {
			for i, row := range lex.Rows(src, Darcula) {
				for j := 1; j < len(row); j++ {
					prev, cur := row[j-1], row[j]
					if prev.Fg == cur.Fg && prev.Bg == cur.Bg && prev.Attr == cur.Attr {
						t.Errorf("%s.Rows(%s) row %d: runs %d and %d share a style and were not merged (%q | %q)",
							lexName, srcName, i, j-1, j, prev.Text, cur.Text)
					}
				}
			}
		}
	}
}

// No run may be empty. An empty run is invisible on every target and costs a
// span (or a SpanStyle, or an AttributedString append) on all four, so it is
// pure waste that only a bug in the run-boundary arithmetic produces.
func TestNoRunIsEmpty(t *testing.T) {
	for lexName, lex := range lexers {
		for srcName, src := range corpus {
			for i, row := range lex.Rows(src, Darcula) {
				for j, run := range row {
					if run.Text == "" {
						t.Errorf("%s.Rows(%s) row %d run %d is empty", lexName, srcName, i, j)
					}
				}
			}
		}
	}
}

// A scheme is applied, not baked in. The same source through the same lexer
// under two schemes has to differ only in the colours — same rows, same run
// boundaries, same text — because that is what makes Light a drop-in for
// Darcula on a light theme.
func TestSchemeChangesColoursAndNothingElse(t *testing.T) {
	const src = "// a comment\nfunc main() { x := \"s\" + 1 }"
	for lexName, lex := range lexers {
		dark, light := lex.Rows(src, Darcula), lex.Rows(src, Light)
		if len(dark) != len(light) {
			t.Fatalf("%s: %d rows under Darcula, %d under Light", lexName, len(dark), len(light))
		}
		for i := range dark {
			if len(dark[i]) != len(light[i]) {
				t.Fatalf("%s row %d: %d runs under Darcula, %d under Light", lexName, i, len(dark[i]), len(light[i]))
			}
			for j := range dark[i] {
				if dark[i][j].Text != light[i][j].Text {
					t.Errorf("%s row %d run %d: text differs by scheme (%q vs %q)",
						lexName, i, j, dark[i][j].Text, light[i][j].Text)
				}
				if dark[i][j].Attr != light[i][j].Attr {
					t.Errorf("%s row %d run %d: attrs differ by scheme (%d vs %d)",
						lexName, i, j, dark[i][j].Attr, light[i][j].Attr)
				}
			}
		}
	}
}

// The zero Scheme is a legal scheme: every run inherits, nothing is coloured.
// Plain() is defined in terms of it, and a caller who forgets to name a scheme
// gets uncoloured code rather than empty rows or a panic.
func TestZeroSchemeColoursNothing(t *testing.T) {
	for lexName, lex := range lexers {
		for _, row := range lex.Rows("func f() { return \"s\" }", Scheme{}) {
			for _, run := range row {
				if run.Fg != "" || run.Bg != "" {
					t.Errorf("%s under the zero scheme produced a coloured run %#v", lexName, run)
				}
			}
		}
	}
}

// ForLanguage's whole table, including the answer for a name nothing lexes.
// The unknown case is the one worth pinning: a screen whose editor mis-spells
// its language must show uncoloured code, not fail to render.
func TestForLanguage(t *testing.T) {
	const goSrc = "func f() {}"
	if got := ForLanguage("go").Rows(goSrc, Darcula)[0][0]; got.Fg != Darcula.Keyword {
		t.Errorf(`ForLanguage("go") did not colour a keyword: %#v`, got)
	}
	if got := ForLanguage("json").Rows(`{"k":1}`, Darcula)[0]; len(got) < 2 {
		t.Errorf(`ForLanguage("json") produced %d runs for an object, want several`, len(got))
	}
	for _, name := range []string{"", "text", "GO", "golang", "markdown", "js"} {
		rows := ForLanguage(name).Rows(goSrc, Darcula)
		if len(rows) != 1 || len(rows[0]) != 1 || rows[0][0].Fg != "" {
			t.Errorf("ForLanguage(%q) coloured something; an unknown language must be Plain: %#v", name, rows)
		}
	}
}

// Plain is one run per line and no colour — the property the CodeEditor leans
// on for a text file, and the shape every other lexer falls back to.
func TestPlainIsOneUncolouredRunPerLine(t *testing.T) {
	rows := Plain().Rows("one\ntwo\n\nfour", Darcula)
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4", len(rows))
	}
	for i, row := range rows {
		if i == 2 { // the blank line
			if len(row) != 0 {
				t.Errorf("blank line produced %d runs, want none", len(row))
			}
			continue
		}
		if len(row) != 1 {
			t.Errorf("row %d produced %d runs, want 1", i, len(row))
		}
		if row[0].Fg != "" || row[0].Attr != 0 {
			t.Errorf("row %d is decorated: %#v", i, row[0])
		}
	}
}
