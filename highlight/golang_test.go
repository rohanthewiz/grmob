package highlight

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// runsOfRow renders a row as "text|fg|attr" triples, which is the form the
// run-boundary assertions below read best in: a wrong boundary shows up as a
// different split, and a wrong class as a different colour, in one diff.
func runsOfRow(row core.GridRow) []string {
	out := make([]string, len(row))
	for i, r := range row {
		// A run in the default ink is written as its bare text, so an
		// expectation below reads as the line it describes with the coloured
		// pieces called out — rather than as a column of trailing pipes.
		out[i] = r.Text
		if r.Fg != "" {
			out[i] += "|" + r.Fg
		}
		if r.Attr != 0 {
			out[i] += "|italic"
		}
	}
	return out
}

func assertRuns(t *testing.T, row core.GridRow, want ...string) {
	t.Helper()
	got := runsOfRow(row)
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("runs:\n got %q\nwant %q", got, want)
	}
}

// The whole Go class table on one line each, with the run boundaries the
// maximal-run rule produces. This is the test that would catch a class being
// silently reassigned or a boundary shifting by a byte.
func TestGoClassifiesEachTokenKind(t *testing.T) {
	d := Darcula
	rows := Go().Rows(strings.Join([]string{
		`// line comment`,
		`/* block */`,
		`func main() {`,
		`	x := 42 + 1.5`,
		`	s := "hi" + ` + "`raw`" + ` + 'r'`,
		`	var b []byte = make([]byte, len(s))`,
		`}`,
	}, "\n"), d)

	assertRuns(t, rows[0], "// line comment|"+d.Comment+"|italic")
	assertRuns(t, rows[1], "/* block */|"+d.DocComment+"|italic")
	// `main` is followed by `(`, so the lookahead rule calls it a func name.
	assertRuns(t, rows[2], "func|"+d.Keyword, " ", "main|"+d.Func, "() {")
	assertRuns(t, rows[3], "\tx := ", "42|"+d.Number, " + ", "1.5|"+d.Number)
	assertRuns(t, rows[4], "\ts := ", `"hi"|`+d.String, " + ", "`raw`|"+d.String, " + ", "'r'|"+d.String)
	// byte, make and len are all predeclared: they take the keyword colour,
	// and `make(` does not become a func name because predeclared is tested
	// first. `b` and `s` stay plain — a lexer cannot know what they are.
	assertRuns(t, rows[5], "\t", "var|"+d.Keyword, " b []", "byte|"+d.Keyword, " = ",
		"make|"+d.Keyword, "([]", "byte|"+d.Keyword, ", ", "len|"+d.Keyword, "(s))")
	assertRuns(t, rows[6], "}")
}

// A `//` inside a string is a string. This is the single best argument for
// go/scanner over a regexp pass, so it gets its own test.
func TestGoCommentMarkerInsideAStringIsNotAComment(t *testing.T) {
	rows := Go().Rows(`u := "https://example.com" // the real comment`, Darcula)
	assertRuns(t, rows[0], "u := ", `"https://example.com"|`+Darcula.String, " ",
		"// the real comment|"+Darcula.Comment+"|italic")
}

// A block comment is coloured on every line it covers, which is the property
// the per-byte class array exists for: the comment is one token and three
// rows, and slicing the array at the newlines splits it without the lexer
// having to know about lines at all.
func TestGoBlockCommentIsColouredOnEveryLineItCovers(t *testing.T) {
	rows := Go().Rows("/* one\n   two */\nx := 1", Darcula)
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	for i := range 2 {
		if len(rows[i]) != 1 || rows[i][0].Fg != Darcula.DocComment {
			t.Errorf("row %d is not all comment: %#v", i, rows[i])
		}
	}
	assertRuns(t, rows[2], "x := ", "1|"+Darcula.Number)
}

// Source the scanner cannot read cleanly comes back plain rather than
// half-coloured. This is the normal state of a line someone is typing in a
// CodeEditor, so "plain" has to mean plain and not "coloured up to the
// mistake".
func TestGoUnlexableSourceIsReturnedPlain(t *testing.T) {
	const src = "x := \"unterminated\ny := 1"
	rows := Go().Rows(src, Darcula)
	if got := rowsText(rows); got != src {
		t.Fatalf("round trip: got %q, want %q", got, src)
	}
	for i, row := range rows {
		for _, run := range row {
			if run.Fg != "" || run.Attr != 0 {
				t.Errorf("row %d carries decoration after a failed scan: %#v", i, run)
			}
		}
	}
}

// Indentation survives. A grid run's spaces are the only white space in a grid
// that means anything (htmlout's textGridChassis says so from the other end),
// and the lexer must therefore hand the leading tabs and spaces through as
// part of a run rather than trimming them.
func TestGoPreservesIndentation(t *testing.T) {
	rows := Go().Rows("core.Row(\n    core.Gap(8),\n)", Darcula)
	if got := rows[1][0].Text; !strings.HasPrefix(got, "    ") {
		t.Errorf("row 1 starts with %q, want four leading spaces", got)
	}
}
