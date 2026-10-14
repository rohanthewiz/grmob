// HTML and XML highlighting.
//
// Markup is not a token stream the table in table.go can describe: what a
// word means depends on whether it is inside a tag, and the text between tags
// is prose. So it gets its own small lexer, with the same class bytes and the
// same unterminated rule as the table (see table.go):
//
//	<!-- … -->            block comment (only once closed)
//	<name  </name  <?xml  the name: Keyword
//	attr=                 an attribute name inside a tag: Func
//	"…"  '…'              an attribute value inside a tag: String
//	everything else       plain ink, including the text between tags and
//	                      the contents of <script> and <style>
//
// The roles mirror JSON's: a tag is the language's vocabulary and an
// attribute name is a key.
package highlight

import (
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

type markupHighlighter struct{}

func (markupHighlighter) Rows(src string, scheme Scheme) []core.GridRow {
	return rowsOf(src, classifyMarkup(src), scheme)
}

var markupComment = []delim{{open: "<!--", close: "-->", multiline: true}}

// tagValue is an attribute value. Multi-line, because an attribute value may
// span lines, and with no escape: markup escapes with entities, not `\`.
var tagValue = []delim{
	{open: `"`, close: `"`, multiline: true},
	{open: "'", close: "'", multiline: true},
}

func classifyMarkup(src string) []byte {
	cls := make([]byte, len(src))
	for i := 0; i < len(src); {
		if n := matchRegion(src, i, markupComment); n > 0 {
			fill(cls, i, i+n, classDocCmt)
			i += n
			continue
		}
		// An unclosed comment's opener is plain and skipped whole. Read as
		// a `<!` tag instead, it would run to the next `>` and swallow the
		// tag after it.
		if strings.HasPrefix(src[i:], "<!--") {
			i += len("<!--")
			continue
		}
		if src[i] == '<' && i+1 < len(src) && startsTagName(src[i+1]) {
			i = classifyTag(src, i, cls)
			continue
		}
		i++
	}
	return cls
}

// startsTagName reports whether the byte after a `<` begins a tag: a name, a
// closing `/`, or the `!`/`?` of a doctype or processing instruction. A `<`
// followed by anything else (a space, a digit: `a < 3`) is text.
func startsTagName(c byte) bool {
	return isLetter(c) || c == '/' || c == '!' || c == '?'
}

// classifyTag colours one tag starting at the `<` at src[i] and returns the
// offset just past it: past its `>`, or the end of the input when the tag is
// never closed. An unclosed tag colours only what it has seen, so nothing
// past the input's end is claimed.
func classifyTag(src string, i int, cls []byte) int {
	j := i + 1
	for j < len(src) && (src[j] == '/' || src[j] == '!' || src[j] == '?') {
		j++
	}
	nameStart := j
	for j < len(src) && isNameByte(src[j]) {
		j++
	}
	fill(cls, nameStart, j, classKeyword)
	for j < len(src) && src[j] != '>' {
		if n := matchRegion(src, j, tagValue); n > 0 {
			fill(cls, j, j+n, classString)
			j += n
			continue
		}
		if isNameByte(src[j]) {
			k := j
			for k < len(src) && isNameByte(src[k]) {
				k++
			}
			// A name followed by `=` is an attribute being set. A bare
			// attribute (`disabled`) stays plain, as a word with nothing
			// after it is in the table lexer.
			if nextOnLine(src, k) == '=' {
				fill(cls, j, k, classFunc)
			}
			j = k
			continue
		}
		j++
	}
	if j < len(src) {
		j++ // the `>`
	}
	return j
}

// isNameByte is a byte of a tag or attribute name: letters, digits and the
// punctuation XML names allow (`-`, `_`, `:`, `.`), plus any non-ASCII byte.
func isNameByte(c byte) bool {
	return isAlnum(c) || c >= 0x80 || strings.IndexByte("-_:.", c) >= 0
}
