// JSON syntax highlighting.
//
// A hand lexer rather than encoding/json, and the reason is the same one that
// makes go/scanner right for Go and a *parser* wrong for both: the input is
// very often invalid. A CodeEditor's buffer is mid-edit most of the time —
// half a key typed, a closing brace not there yet — and encoding/json answers
// "invalid" for all of it, which would mean a document that loses every colour
// the moment someone puts the cursor in it.
//
// So this lexer never fails. It classifies what it can see and treats anything
// it does not recognize as plain ink, which degrades exactly the way an editor
// should: the bytes you have finished typing are coloured, the ones you are in
// the middle of are not yet.
package highlight

import "github.com/rohanthewiz/grmob/core"

// JSON returns the JSON highlighter. See Go() for why this is a function.
func JSON() Highlighter { return jsonHighlighter{} }

type jsonHighlighter struct{}

func (jsonHighlighter) Rows(src string, scheme Scheme) []core.GridRow {
	return rowsOf(src, scanJSON(src), scheme)
}

// scanJSON classifies every byte of src.
//
// # The four things JSON has that are worth a colour
//
//	a key        the string on the left of a colon        -> classFunc
//	a string     every other string literal               -> classString
//	a number     -1.5e3 and friends                       -> classNumber
//	a literal    true, false, null                        -> classKeyword
//
// Everything else — the braces, the brackets, the commas, the colons and the
// white space between them — is structure, and structure stays in the default
// ink. That is the same call the Go lexer makes about operators and
// delimiters, for the same reason: colouring punctuation adds noise to the
// thing the colours are supposed to pick out.
//
// # Keys are told from strings by looking ahead, not by tracking depth
//
// A string is a key exactly when the next non-space byte after it is a colon.
// That is a purely local test and it is right for valid JSON, which is what
// matters: no bracket stack to keep, nothing to get wrong when the document is
// unbalanced mid-edit, and the failure mode on invalid input is that a string
// is coloured as a string. The alternative (remember whether we are inside an
// object and whether we are before or after a colon) needs a stack that a
// half-typed document will regularly leave in the wrong state.
//
// classFunc is the class a key takes because Scheme has no Key role and a key
// is a *name*, which is the role Func names. Darcula and GitHub-light both
// give names a colour distinct from strings, which is the distinction a reader
// of JSON actually wants.
func scanJSON(src string) []byte {
	classes := make([]byte, len(src))
	for i := 0; i < len(src); {
		switch c := src[i]; {
		case c == '"':
			end := scanJSONString(src, i)
			class := byte(classString)
			if jsonNextIsColon(src, end) {
				class = classFunc
			}
			fill(classes, i, end, class)
			i = end
		case c == '-' || c == '+' || (c >= '0' && c <= '9'):
			end := scanJSONNumber(src, i)
			fill(classes, i, end, classNumber)
			i = end
		case c >= 'a' && c <= 'z':
			// A bare word. Only the three JSON literals are coloured; any
			// other word is something the document should not contain, and
			// leaving it plain is how the reader gets told so.
			end := i
			for end < len(src) && src[end] >= 'a' && src[end] <= 'z' {
				end++
			}
			switch src[i:end] {
			case "true", "false", "null":
				fill(classes, i, end, classKeyword)
			}
			i = end
		default:
			i++
		}
	}
	return classes
}

// scanJSONString returns the offset one past the closing quote of the string
// starting at i (which must be the opening quote).
//
// An unterminated string runs to the end of the *source*, not to the end of
// the line, which is deliberate even though JSON forbids a literal newline
// inside a string: the byte after the opening quote of a string someone is
// still typing is the rest of the document, and colouring it as a string is
// what every editor does while the quote is open. The Go lexer reaches the
// same outcome by a different road — it abandons the whole scan — because
// there the rest of the file really does become unreadable, whereas here a
// following line is still lexed from scratch on the next keystroke.
func scanJSONString(src string, i int) int {
	for j := i + 1; j < len(src); j++ {
		switch src[j] {
		case '\\':
			// Skip the escaped byte, whatever it is. \uXXXX needs no special
			// handling: its four hex digits are ordinary string bytes, and the
			// only byte that could end the string early is the quote, which
			// this skip has already consumed if it was escaped.
			j++
		case '"':
			return j + 1
		}
	}
	return len(src)
}

// scanJSONNumber returns the offset one past the last byte of the number
// starting at i.
//
// Permissive on purpose: it accepts the bytes a number is made of rather than
// checking the grammar, so "1.2.3" and "1e" are one number-coloured run each.
// Validating here would buy nothing — this is a colouring, and a malformed
// number is the writer's problem to see, not the highlighter's to hide by
// refusing to colour it.
func scanJSONNumber(src string, i int) int {
	j := i
	for j < len(src) {
		c := src[j]
		if (c >= '0' && c <= '9') || c == '.' || c == '-' || c == '+' || c == 'e' || c == 'E' {
			j++
			continue
		}
		break
	}
	return j
}

// jsonNextIsColon reports whether the next non-space byte at or after i is a
// colon. White space here is JSON's own set plus the carriage return, so a
// CRLF document answers the same as an LF one.
func jsonNextIsColon(src string, i int) bool {
	for ; i < len(src); i++ {
		switch src[i] {
		case ' ', '\t', '\n', '\r':
			continue
		case ':':
			return true
		}
		return false
	}
	return false
}

// fill writes class over classes[from:to]. The bounds come from the scanners
// above, which never return an offset past len(src), so no clamping is needed
// — but the half-open convention is worth naming, because the per-byte array
// is the one place in this package where an off-by-one paints a colour onto a
// byte that belongs to the next token.
func fill(classes []byte, from, to int, class byte) {
	for i := from; i < to && i < len(classes); i++ {
		classes[i] = class
	}
}
