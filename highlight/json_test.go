package highlight

import (
	"strings"
	"testing"
)

// The whole JSON class table, with the run boundaries the maximal-run rule
// produces. The interesting line is the first: `"a"` is a key and `"b"` is a
// value, and the only thing that tells them apart is the colon after one of
// them.
func TestJSONClassifiesKeysStringsNumbersAndLiterals(t *testing.T) {
	d := Darcula
	rows := JSON().Rows(strings.Join([]string{
		`{"a": "b",`,
		` "n": -1.5e3,`,
		` "t": true, "f": false, "z": null,`,
		` "list": [1, "two"]}`,
	}, "\n"), d)

	assertRuns(t, rows[0], "{", `"a"|`+d.Func, ": ", `"b"|`+d.String, ",")
	assertRuns(t, rows[1], " ", `"n"|`+d.Func, ": ", "-1.5e3|"+d.Number, ",")
	assertRuns(t, rows[2], " ", `"t"|`+d.Func, ": ", "true|"+d.Keyword, ", ",
		`"f"|`+d.Func, ": ", "false|"+d.Keyword, ", ", `"z"|`+d.Func, ": ", "null|"+d.Keyword, ",")
	assertRuns(t, rows[3], " ", `"list"|`+d.Func, ": [", "1|"+d.Number, ", ", `"two"|`+d.String, "]}")
}

// A key whose colon is on the next line is still a key. The lookahead skips
// white space, newlines included, because a formatter is free to break there
// and the colouring must not depend on where it chose to.
func TestJSONKeyIsFoundAcrossALineBreak(t *testing.T) {
	rows := JSON().Rows("{\n  \"k\"\n  : 1\n}", Darcula)
	if got := rows[1][1].Fg; got != Darcula.Func {
		t.Errorf(`"k" before a colon on the next line coloured %q, want the key colour %q`, got, Darcula.Func)
	}
}

// An escaped quote does not end a string, and a backslash before the closing
// quote does not swallow it. Both are the same skip, and both are the kind of
// thing that shows up as "the rest of the file is green".
func TestJSONEscapesInsideStrings(t *testing.T) {
	rows := JSON().Rows(`{"k": "a\"b\\", "j": 1}`, Darcula)
	assertRuns(t, rows[0], "{", `"k"|`+Darcula.Func, ": ", `"a\"b\\"|`+Darcula.String, ", ",
		`"j"|`+Darcula.Func, ": ", "1|"+Darcula.Number, "}")
}

// The mid-edit case, which is why this is a hand lexer and not encoding/json.
// An unterminated string colours to the end and everything before it keeps the
// colours it had; nothing is dropped and nothing panics.
func TestJSONUnterminatedStringDoesNotLoseTheDocument(t *testing.T) {
	const src = "{\n  \"done\": 1,\n  \"typing\": \"half"
	rows := JSON().Rows(src, Darcula)
	if got := rowsText(rows); got != src {
		t.Fatalf("round trip: got %q, want %q", got, src)
	}
	if got := rows[1][1].Fg; got != Darcula.Func {
		t.Errorf("the finished key above the open string lost its colour: %q", got)
	}
	last := rows[2][len(rows[2])-1]
	if last.Fg != Darcula.String {
		t.Errorf("the open string is not coloured as a string: %#v", last)
	}
}

// A word that is not one of the three literals stays plain. Colouring every
// bare word as a literal would make a typo look valid, which is the opposite
// of what a highlighter is for.
func TestJSONOnlyTheThreeLiteralsAreKeywords(t *testing.T) {
	rows := JSON().Rows(`{"k": nul, "j": truthy}`, Darcula)
	for _, run := range rows[0] {
		if (run.Text == "nul" || run.Text == "truthy") && run.Fg != "" {
			t.Errorf("%q was coloured as a literal: %#v", run.Text, run)
		}
	}
}

// Structure stays in the default ink: braces, brackets, commas and colons are
// the frame, not the content.
func TestJSONStructureIsUncoloured(t *testing.T) {
	for _, run := range JSON().Rows(`{"a":[{},[]]}`, Darcula)[0] {
		if strings.Trim(run.Text, "{}[],:") == "" && run.Fg != "" {
			t.Errorf("punctuation %q was coloured: %#v", run.Text, run)
		}
	}
}
