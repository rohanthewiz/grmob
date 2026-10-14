package highlight

import (
	"strings"
	"testing"
)

// Every language ForLanguage answers to joins the census in
// highlight_test.go, so each inherits the line-for-line, byte-preserving,
// maximal-run and scheme properties over the whole corpus.
func init() {
	for _, name := range Languages() {
		lexers[name] = ForLanguage(name)
	}
}

// classOf returns the colour the lexer gave the first occurrence of tok in
// src, by walking the runs. A token split across runs of different colours
// reports "split", which every case below would read as a failure.
func classOf(t *testing.T, lang, src, tok string) string {
	t.Helper()
	at := strings.Index(src, tok)
	if at < 0 {
		t.Fatalf("%s: %q is not in the source", lang, tok)
	}
	off := 0
	colour := ""
	seen := false
	for _, row := range ForLanguage(lang).Rows(src, Darcula) {
		for _, run := range row {
			start, end := off, off+len(run.Text)
			if end > at && start < at+len(tok) {
				if seen && run.Fg != colour {
					return "split"
				}
				colour, seen = run.Fg, true
			}
			off = end
		}
		off++ // the newline between rows
	}
	return colour
}

// role names a Darcula colour, so a failure says "keyword" rather than a hex.
func role(fg string) string {
	switch fg {
	case Darcula.Keyword:
		return "keyword"
	case Darcula.String:
		return "string"
	case Darcula.Number:
		return "number"
	case Darcula.Comment:
		return "comment"
	case Darcula.DocComment:
		return "block comment"
	case Darcula.Func:
		return "func"
	case "":
		return "plain"
	}
	return fg
}

// One case per role per language family: a token, and the role it must get.
// The sources are small but real, so a case also says what the language's
// delimiters are in practice.
func TestTableLanguagesColourTheirTokens(t *testing.T) {
	cases := []struct {
		lang, src, tok, want string
	}{
		{"javascript", "const x = fetch('/a') // go\n", "const", "keyword"},
		{"javascript", "const x = fetch('/a') // go\n", "fetch", "func"},
		{"javascript", "const x = fetch('/a') // go\n", "'/a'", "string"},
		{"javascript", "const x = fetch('/a') // go\n", "// go", "comment"},
		{"javascript", "let s = `one\ntwo`\n", "`one\ntwo`", "string"},
		{"typescript", "interface P { n: number }", "interface", "keyword"},
		{"python", "def greet(x):\n    return 0x1F  # hex\n", "def", "keyword"},
		{"python", "def greet(x):\n    return 0x1F  # hex\n", "greet", "func"},
		{"python", "def greet(x):\n    return 0x1F  # hex\n", "0x1F", "number"},
		{"python", "def greet(x):\n    return 0x1F  # hex\n", "# hex", "comment"},
		{"python", "s = \"\"\"doc\nmore\"\"\"\n", "\"\"\"doc\nmore\"\"\"", "string"},
		{"bash", "if [ -n \"$X\" ]; then echo hi; fi # done\n", "then", "keyword"},
		{"bash", "if [ -n \"$X\" ]; then echo hi; fi # done\n", "\"$X\"", "string"},
		{"bash", "if [ -n \"$X\" ]; then echo hi; fi # done\n", "# done", "comment"},
		{"bash", "n=${#arr}\n", "#arr", "plain"},
		{"sql", "SELECT name FROM users WHERE id = 42; -- one\n", "SELECT", "keyword"},
		{"sql", "select name from users where id = 42; -- one\n", "from", "keyword"},
		{"sql", "SELECT name FROM users WHERE id = 42; -- one\n", "42", "number"},
		{"sql", "SELECT name FROM users WHERE id = 42; -- one\n", "-- one", "comment"},
		{"sql", "SELECT 'it''s'", "'it''s'", "string"},
		{"yaml", "name: app # the name\nlist:\n  - true\n", "name", "func"},
		{"yaml", "name: app # the name\nlist:\n  - true\n", "true", "keyword"},
		{"yaml", "name: app # the name\nlist:\n  - true\n", "# the name", "comment"},
		{"yaml", "url: http://x/#frag\n", "#frag", "plain"},
		{"toml", "[server]\nport = 8080\n", "port", "func"},
		{"toml", "[server]\nport = 8080\n", "8080", "number"},
		{"rust", "fn main() { let s = \"x\"; } /* b */", "fn", "keyword"},
		{"rust", "fn main() { let s = \"x\"; } /* b */", "main", "func"},
		{"rust", "fn main() { let s = \"x\"; } /* b */", "/* b */", "block comment"},
		{"java", "public static void main(String[] a) {}", "public", "keyword"},
		{"kotlin", "fun greet(n: String) = \"hi $n\"", "fun", "keyword"},
		{"swift", "func greet() -> String { return \"hi\" }", "func", "keyword"},
		{"c", "#include <stdio.h>\nint main(void) { return 0; }", "#include", "keyword"},
		{"cpp", "template <typename T> class Box {};", "template", "keyword"},
		{"csharp", "public class A { string s = \"x\"; }", "class", "keyword"},
		{"ruby", "def hi\n  puts \"hi\" # say\nend\n", "def", "keyword"},
		{"ruby", "def hi\n  puts \"hi\" # say\nend\n", "# say", "comment"},
		{"php", "<?php function f($a) { return $a; }", "function", "keyword"},
		{"css", ".a { color: #fff; margin: 10px; } /* c */", "color", "func"},
		{"css", ".a { color: #fff; margin: 10px; } /* c */", "10px", "number"},
		{"css", "@media screen { }", "@media", "keyword"},
		{"lua", "local x = 1 --[[ block\nmore ]] -- line\n", "local", "keyword"},
		{"lua", "local x = 1 --[[ block\nmore ]] -- line\n", "--[[ block\nmore ]]", "block comment"},
		{"lua", "local x = 1 --[[ block\nmore ]] -- line\n", "-- line", "comment"},
		{"dockerfile", "FROM golang:1.26 AS build\nRUN go build\n", "FROM", "keyword"},
		{"dockerfile", "FROM golang:1.26 AS build\nRUN go build\n", "RUN", "keyword"},
		{"html", "<a href=\"/x\">link</a><!-- note -->", "a", "keyword"},
		{"html", "<a href=\"/x\">link</a><!-- note -->", "href", "func"},
		{"html", "<a href=\"/x\">link</a><!-- note -->", "\"/x\"", "string"},
		{"html", "<a href=\"/x\">link</a><!-- note -->", "link", "plain"},
		{"html", "<a href=\"/x\">link</a><!-- note -->", "<!-- note -->", "block comment"},
		{"xml", "<?xml version=\"1.0\"?>\n<root/>", "xml", "keyword"},
		{"xml", "<?xml version=\"1.0\"?>\n<root/>", "version", "func"},
		{"html", "<p>a < 3</p>", "< 3", "plain"},
	}
	for _, c := range cases {
		if got := role(classOf(t, c.lang, c.src, c.tok)); got != c.want {
			t.Errorf("%s: %q in %q is %s, want %s", c.lang, c.tok, c.src, got, c.want)
		}
	}
}

// The unterminated rule: an opener with no closer is plain, and everything
// after it is lexed as if it were not there. The cases are the ones a notes
// app meets every day: an apostrophe in a shell word, a string mid-typing,
// a block comment not yet closed.
func TestUnterminatedOpenersStayPlainAndDoNotDrift(t *testing.T) {
	cases := []struct {
		lang, src, tok, want string
	}{
		// The apostrophe opens nothing, and the keyword after it is still
		// a keyword.
		{"bash", "echo don't\nif true; then :; fi\n", "'t", "plain"},
		{"bash", "echo don't\nif true; then :; fi\n", "then", "keyword"},
		// A string still being typed: the rest of its line is plain, and the
		// next line is lexed normally.
		{"javascript", "let s = \"open\nlet n = 1\n", "\"open", "plain"},
		{"javascript", "let s = \"open\nlet n = 1\n", "1", "number"},
		// A block comment never closed is not a comment, so the code after
		// its opener keeps its colours.
		{"c", "/* todo\nint x = 3;\n", "int", "keyword"},
		{"html", "<!-- open\n<b>x</b>", "b", "keyword"},
	}
	for _, c := range cases {
		if got := role(classOf(t, c.lang, c.src, c.tok)); got != c.want {
			t.Errorf("%s: %q in %q is %s, want %s", c.lang, c.tok, c.src, got, c.want)
		}
	}
}

// Languages is sorted, includes the hand-written lexers, and names nothing
// ForLanguage would answer with Plain.
func TestLanguagesAreAllLexed(t *testing.T) {
	names := Languages()
	for i := 1; i < len(names); i++ {
		if names[i-1] >= names[i] {
			t.Errorf("Languages() is not sorted and unique at %q, %q", names[i-1], names[i])
		}
	}
	for _, want := range []string{"go", "json", "html", "python", "bash"} {
		found := false
		for _, n := range names {
			found = found || n == want
		}
		if !found {
			t.Errorf("Languages() is missing %q", want)
		}
	}
	for _, n := range names {
		if _, plain := ForLanguage(n).(plainHighlighter); plain {
			t.Errorf("Languages() lists %q but ForLanguage answers Plain for it", n)
		}
	}
}

// LineComment answers for every lexed language and for nothing else, and a
// few markers are pinned by name.
func TestLineComment(t *testing.T) {
	for _, n := range Languages() {
		if _, ok := LineComment(n); !ok {
			t.Errorf("LineComment(%q) is not ok, but %q is lexed", n, n)
		}
	}
	for lang, want := range map[string]string{
		"go": "//", "python": "#", "bash": "#", "sql": "--", "lua": "--",
		"javascript": "//", "json": "", "css": "", "html": "",
	} {
		if got, _ := LineComment(lang); got != want {
			t.Errorf("LineComment(%q) = %q, want %q", lang, got, want)
		}
	}
	if _, ok := LineComment("cobol"); ok {
		t.Error(`LineComment("cobol") is ok; an unknown language must say so`)
	}
}
