// Table-driven highlighting for the languages that have no lexer of their own
// in this package.
//
// Go has go/scanner and JSON has a grammar small enough to lex exactly. The
// rest — JavaScript, Python, shell, SQL, YAML and the C family among them —
// are coloured by one lexer driven by a per-language table (langSpec): its
// keywords, its comment and string delimiters, and a few flags for the shapes
// that differ (a `#` that only starts a comment after whitespace, a YAML key
// before its colon, SQL's case-insensitive keywords). HTML and XML are tags
// rather than tokens and get a small lexer of their own (markup.go).
//
// # Why a table and not a lexer per language
//
// The colour roles a Scheme has are few: keyword, string, number, comment,
// block comment, and the identifier being called or the key being set. At
// that resolution most languages differ only in their vocabulary and their
// delimiters, so a table states each language in a few lines that read like
// its spec, and every language shares one scanning loop that the property
// tests in highlight_test.go hold to the package's contracts (line-for-line,
// every byte preserved, maximal runs). Twenty hand-written lexers would be
// twenty places for the run arithmetic to go wrong.
//
// The price is precision: a table cannot know that `'a` is a Rust lifetime or
// that `$((…))` is shell arithmetic. Where a construct cannot be described in
// the table it is left uncoloured rather than guessed at, which is the
// package's standing rule (see Highlighter).
//
// # Unterminated constructs
//
// Go's lexer gives up on the whole input when go/scanner reports any error.
// There is no reference scanner here to defer to, so the rule is local
// instead: a string or block comment is coloured only once its closing
// delimiter is found. An opener with no closer is plain ink, and scanning
// resumes on the byte after it. Colours therefore never run on past the point
// where the lexer lost its place, which is the failure the Go fallback exists
// to prevent, and a fence holding a half-written line (or an apostrophe in a
// shell word) keeps its colours everywhere else.
package highlight

import (
	"sort"
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// delim is one delimited region: a string or a block comment.
type delim struct {
	open, close string
	// escape is the byte that makes the next byte literal (`\` in most
	// languages), or 0 for a region with no escapes (shell single quotes,
	// SQL strings, comments).
	escape byte
	// multiline lets the region cross a newline. A single-line string that
	// reaches the end of its line unclosed is unterminated.
	multiline bool
}

// langSpec is one language's table. See the file comment.
type langSpec struct {
	keywords map[string]bool
	// foldCase matches keywords case-insensitively (SQL, Dockerfile). The
	// table's keywords are then written in lower case.
	foldCase bool

	lineComments []string
	// hashNeedsSpace makes a `#` line comment start only at the start of a
	// line or after whitespace, as in shell (`${#arr}`, `a#b`), YAML and
	// TOML (`url: http://x/#frag`).
	hashNeedsSpace bool
	blockComments  []delim
	// strs is tried in order, so a longer opener that shares a prefix with a
	// shorter one (`"""` and `"`) comes first.
	strs []delim

	// identExtra are the bytes besides letters, digits and `_` that may
	// appear in a word: `$` for shell and PHP variables, `-` for CSS
	// properties and YAML keys, `#` for C's preprocessor directives.
	identExtra string
	// calls colours a word followed by `(` (after any spaces) as a call or a
	// declaration, the lexical stand-in for "function name" classifyGo uses.
	calls bool
	// keySep colours a word followed by this byte as a key, in the Func role
	// a JSON object key takes: `:` in YAML and CSS, `=` in TOML.
	keySep byte
}

// tableHighlighter adapts a langSpec to the Highlighter interface.
type tableHighlighter struct{ spec *langSpec }

func (h tableHighlighter) Rows(src string, scheme Scheme) []core.GridRow {
	return rowsOf(src, h.spec.classify(src), scheme)
}

// classify produces one class byte per source byte, as every lexer in this
// package does (see the package doc).
//
// The order of the checks is the precedence a reader expects: a comment or a
// string swallows whatever is inside it, so they are tried before numbers and
// words; a block comment before a line comment, because Lua's `--[[` opens a
// block and starts with the line comment's `--`.
func (sp *langSpec) classify(src string) []byte {
	cls := make([]byte, len(src))
	for i := 0; i < len(src); {
		if n := matchRegion(src, i, sp.blockComments); n > 0 {
			fill(cls, i, i+n, classDocCmt)
			i += n
			continue
		}
		if sp.lineCommentAt(src, i) {
			end := strings.IndexByte(src[i:], '\n')
			if end < 0 {
				end = len(src) - i
			}
			fill(cls, i, i+end, classLineCmt)
			i += end
			continue
		}
		if n := matchRegion(src, i, sp.strs); n > 0 {
			fill(cls, i, i+n, classString)
			i += n
			continue
		}
		c := src[i]
		if isDigit(c) || (c == '.' && i+1 < len(src) && isDigit(src[i+1])) {
			// A number runs over every alphanumeric, `.` and `_` that
			// follows, which covers 0x1F, 1e9, 1_000, 3.14f and 10px alike
			// without a grammar per language. A sign is not part of it: `-`
			// is an operator in every language here.
			j := i + 1
			for j < len(src) && (isAlnum(src[j]) || src[j] == '.' || src[j] == '_') {
				j++
			}
			fill(cls, i, j, classNumber)
			i = j
			continue
		}
		if sp.isWordStart(c) {
			j := i + 1
			for j < len(src) && sp.isWordPart(src[j]) {
				j++
			}
			fill(cls, i, j, sp.classifyWord(src, src[i:j], j))
			i = j
			continue
		}
		i++ // an operator, a delimiter, whitespace, or an unterminated opener
	}
	return cls
}

// classifyWord decides a word's colour; end is the offset just past it, for
// the two lookahead rules.
func (sp *langSpec) classifyWord(src, word string, end int) byte {
	key := word
	if sp.foldCase {
		key = strings.ToLower(word)
	}
	if sp.keywords[key] {
		return classKeyword
	}
	// A word made only of identExtra bytes (the `$` of a shell `$(…)`, a
	// YAML list's `-`) is punctuation, not a name being called or set.
	if strings.IndexFunc(word, func(r rune) bool {
		return r == '_' || r >= 0x80 || (r < 0x80 && isAlnum(byte(r)))
	}) < 0 {
		return classPlain
	}
	next := nextOnLine(src, end)
	if sp.keySep != 0 && next == sp.keySep {
		return classFunc
	}
	if sp.calls && next == '(' {
		return classFunc
	}
	return classPlain
}

// lineCommentAt reports whether a line comment starts at src[i].
func (sp *langSpec) lineCommentAt(src string, i int) bool {
	for _, lc := range sp.lineComments {
		if !strings.HasPrefix(src[i:], lc) {
			continue
		}
		if lc == "#" && sp.hashNeedsSpace && i > 0 && !isSpace(src[i-1]) {
			continue
		}
		return true
	}
	return false
}

func (sp *langSpec) isWordStart(c byte) bool {
	return isLetter(c) || c == '_' || c >= 0x80 || strings.IndexByte(sp.identExtra, c) >= 0
}

func (sp *langSpec) isWordPart(c byte) bool { return sp.isWordStart(c) || isDigit(c) }

// matchRegion returns the length of the first region in ds that opens at
// src[i] and closes, or 0 when none does. An opener whose closer never comes
// (the end of the input, or the end of the line for a single-line region)
// does not match, which is the unterminated rule in the file comment.
//
// Bytes >= 0x80 are never an escape or a delimiter here, so a multibyte rune
// inside a region is stepped over a byte at a time without being split
// between two classes.
func matchRegion(src string, i int, ds []delim) int {
	for _, d := range ds {
		if !strings.HasPrefix(src[i:], d.open) {
			continue
		}
		for j := i + len(d.open); j < len(src); j++ {
			c := src[j]
			if d.escape != 0 && c == d.escape {
				j++ // the escaped byte is literal, whatever it is
				continue
			}
			if c == '\n' && !d.multiline {
				break
			}
			if strings.HasPrefix(src[j:], d.close) {
				return j + len(d.close) - i
			}
		}
	}
	return 0
}

// nextOnLine is the first byte at or after i that is not a space or a tab,
// or 0 when the line (or the input) ends first.
func nextOnLine(src string, i int) byte {
	for ; i < len(src); i++ {
		switch src[i] {
		case ' ', '\t':
			continue
		case '\n':
			return 0
		}
		return src[i]
	}
	return 0
}

func isDigit(c byte) bool  { return c >= '0' && c <= '9' }
func isLetter(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
func isAlnum(c byte) bool  { return isLetter(c) || isDigit(c) }
func isSpace(c byte) bool  { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

// words builds a keyword set from a space-separated list, so a language's
// vocabulary reads as the list its spec gives.
func words(list string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(list) {
		m[w] = true
	}
	return m
}

// The delimiters most of the table shares.
var (
	cBlock       = delim{open: "/*", close: "*/", multiline: true}
	dquote       = delim{open: `"`, close: `"`, escape: '\\'}
	squote       = delim{open: "'", close: "'", escape: '\\'}
	tripleDquote = delim{open: `"""`, close: `"""`, escape: '\\', multiline: true}
	backtick     = delim{open: "`", close: "`", escape: '\\', multiline: true}
)

// jsKeywords is JavaScript's reserved words plus the literals; TypeScript
// adds its type vocabulary to the same set.
const jsKeywords = `async await break case catch class const continue debugger default
delete do else export extends false finally for from function get if import in
instanceof let new null of return set static super switch this throw true try
typeof undefined var void while with yield`

const cFamily = `auto break case char const continue default do double else enum extern
float for goto if inline int long register restrict return short signed sizeof
static struct switch typedef union unsigned void volatile while true false NULL
#include #define #undef #if #ifdef #ifndef #elif #else #endif #pragma #error`

// languages is the table, keyed by the canonical name ForLanguage answers to.
var languages = map[string]*langSpec{
	"javascript": {
		keywords: words(jsKeywords), lineComments: []string{"//"},
		blockComments: []delim{cBlock}, strs: []delim{dquote, squote, backtick},
		identExtra: "$", calls: true,
	},
	"typescript": {
		keywords: words(jsKeywords + ` abstract any as boolean declare enum implements
interface keyof namespace never number private protected public readonly
satisfies string symbol type unknown`),
		lineComments: []string{"//"}, blockComments: []delim{cBlock},
		strs: []delim{dquote, squote, backtick}, identExtra: "$", calls: true,
	},
	"python": {
		keywords: words(`and as assert async await break class continue def del elif else
except False finally for from global if import in is lambda None nonlocal not or
pass raise return self True try while with yield match case`),
		lineComments: []string{"#"},
		strs: []delim{
			tripleDquote,
			{open: "'''", close: "'''", escape: '\\', multiline: true},
			dquote, squote,
		},
		calls: true,
	},
	"bash": {
		keywords: words(`if then else elif fi case esac for select while until do done in
function time return exit break continue local export readonly declare unset
echo printf read cd source true false set shift trap eval exec test`),
		lineComments: []string{"#"}, hashNeedsSpace: true,
		// Single quotes take no escapes in the shell: '\' is a backslash.
		// Single-line, though the shell lets them span lines: an apostrophe
		// in an unquoted word (echo don't) would otherwise open a string
		// that closes at the next apostrophe anywhere below it.
		strs:       []delim{dquote, {open: "'", close: "'"}},
		identExtra: "$", calls: true,
	},
	"sql": {
		keywords: words(`add all alter and as asc begin between by case check column commit
constraint create cross database default delete desc distinct drop else end
exists foreign from full group having if in index inner insert into is join
key left like limit not null offset on or order outer primary references
returning right rollback select set table then transaction union unique update
using values view when where with true false integer int text varchar boolean
timestamp date serial bigint real numeric`),
		foldCase: true, lineComments: []string{"--"}, blockComments: []delim{cBlock},
		// SQL doubles a quote to escape it (''), which this reads as one
		// string closing and the next opening: two runs of one class, merged.
		strs:  []delim{{open: "'", close: "'"}, {open: `"`, close: `"`}},
		calls: true,
	},
	"yaml": {
		keywords:     words(`true false null yes no on off`),
		lineComments: []string{"#"}, hashNeedsSpace: true,
		strs:       []delim{dquote, {open: "'", close: "'"}},
		identExtra: "-.", keySep: ':',
	},
	"toml": {
		keywords:     words(`true false`),
		lineComments: []string{"#"}, hashNeedsSpace: true,
		strs: []delim{
			tripleDquote, {open: "'''", close: "'''", multiline: true},
			dquote, {open: "'", close: "'"},
		},
		identExtra: "-.", keySep: '=',
	},
	"rust": {
		keywords: words(`as async await break const continue crate dyn else enum extern false
fn for if impl in let loop match mod move mut pub ref return self Self static
struct super trait true type unsafe use where while Some None Ok Err Option
Result String Vec Box i8 i16 i32 i64 i128 isize u8 u16 u32 u64 u128 usize f32
f64 bool char str`),
		// No single-quote strings: a char literal and a lifetime ('a) share
		// the quote, and only a parser can tell them apart.
		lineComments: []string{"//"}, blockComments: []delim{cBlock},
		strs: []delim{dquote}, calls: true,
	},
	"java": {
		keywords: words(`abstract assert boolean break byte case catch char class const
continue default do double else enum extends final finally float for goto if
implements import instanceof int interface long native new package private
protected public return short static strictfp super switch synchronized this
throw throws transient try void volatile while var record true false null`),
		lineComments: []string{"//"}, blockComments: []delim{cBlock},
		strs: []delim{tripleDquote, dquote, squote}, calls: true,
	},
	"kotlin": {
		keywords: words(`abstract annotation as break by catch class companion const
constructor continue crossinline data do else enum external false final finally
for fun get if import in infix init inline inner interface internal is lateinit
noinline null object open operator out override package private protected public
reified return sealed set super suspend this throw true try typealias val var
vararg when where while`),
		lineComments: []string{"//"}, blockComments: []delim{cBlock},
		strs: []delim{tripleDquote, dquote, squote}, calls: true,
	},
	"swift": {
		keywords: words(`associatedtype async await break case catch class continue default
defer deinit do else enum extension fallthrough false fileprivate for func guard
if import in init inout internal is let nil open operator private protocol
public repeat rethrows return self Self some static struct subscript super
switch throw throws true try typealias var where while any`),
		lineComments: []string{"//"}, blockComments: []delim{cBlock},
		strs: []delim{tripleDquote, dquote}, calls: true,
	},
	"c": {
		keywords: words(cFamily), lineComments: []string{"//"},
		blockComments: []delim{cBlock}, strs: []delim{dquote, squote},
		identExtra: "#", calls: true,
	},
	"cpp": {
		keywords: words(cFamily + ` alignas alignof bool catch class constexpr
const_cast decltype delete dynamic_cast explicit export friend mutable namespace
new noexcept nullptr operator override private protected public
reinterpret_cast static_assert static_cast template this throw try typeid
typename using virtual final concept requires co_await co_return co_yield`),
		lineComments: []string{"//"}, blockComments: []delim{cBlock},
		strs: []delim{dquote, squote}, identExtra: "#", calls: true,
	},
	"csharp": {
		keywords: words(`abstract as async await base bool break byte case catch char checked
class const continue decimal default delegate do double else enum event explicit
extern false finally fixed float for foreach goto if implicit in int interface
internal is lock long namespace new null object operator out override params
private protected public readonly record ref return sbyte sealed short sizeof
stackalloc static string struct switch this throw true try typeof uint ulong
unchecked unsafe ushort using var virtual void volatile while`),
		lineComments: []string{"//"}, blockComments: []delim{cBlock},
		strs: []delim{tripleDquote, dquote, squote}, calls: true,
	},
	"ruby": {
		keywords: words(`alias and begin break case class def defined? do else elsif end
ensure false for if in module next nil not or redo rescue retry return self
super then true undef unless until when while yield require attr_accessor
attr_reader puts`),
		lineComments: []string{"#"}, strs: []delim{dquote, squote},
		identExtra: "?!", calls: true,
	},
	"php": {
		keywords: words(`abstract and array as break callable case catch class clone const
continue declare default do echo else elseif empty enddeclare endfor endforeach
endif endswitch endwhile extends final finally fn for foreach function global
goto if implements include include_once instanceof insteadof interface isset
list match namespace new or print private protected public readonly require
require_once return static switch throw trait try unset use var while xor yield
true false null`),
		lineComments: []string{"//", "#"}, blockComments: []delim{cBlock},
		strs: []delim{dquote, squote}, identExtra: "$", calls: true,
	},
	"css": {
		keywords: words(`@media @import @keyframes @font-face @supports @layer @charset
@page @container !important`),
		blockComments: []delim{cBlock}, strs: []delim{dquote, squote},
		identExtra: "-@!", calls: true, keySep: ':',
	},
	"lua": {
		keywords: words(`and break do else elseif end false for function goto if in local
nil not or repeat return then true until while`),
		lineComments: []string{"--"},
		blockComments: []delim{
			{open: "--[[", close: "]]", multiline: true},
		},
		strs:  []delim{{open: "[[", close: "]]", multiline: true}, dquote, squote},
		calls: true,
	},
	"dockerfile": {
		keywords: words(`from as run cmd label maintainer expose env add copy entrypoint
volume user workdir arg onbuild stopsignal healthcheck shell`),
		foldCase: true, lineComments: []string{"#"}, hashNeedsSpace: true,
		strs: []delim{dquote, squote},
	},
}

// Languages lists every language name ForLanguage lexes, sorted. A caller
// mapping its own names (a Markdown fence's "js", a file's ".py") onto these
// can check its table against this list rather than against the docs.
func Languages() []string {
	names := []string{"go", "json", "html", "xml"}
	for name := range languages {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// LineComment is the line-comment marker of a language ForLanguage lexes:
// "#" for Python and shell, "--" for SQL and Lua, "//" for the C family and
// JavaScript. ok is true for every language this package knows, and prefix is
// "" for one with no line comment (JSON, CSS, HTML, XML), so an editor's
// comment command can tell "no marker" apart from "unknown language".
// comps.CodeEditor uses it to pick the marker its comment command toggles.
func LineComment(lang string) (prefix string, ok bool) {
	switch lang {
	case "go":
		return "//", true
	case "json", "html", "xml":
		return "", true
	}
	spec, known := languages[lang]
	if !known {
		return "", false
	}
	if len(spec.lineComments) == 0 {
		return "", true
	}
	return spec.lineComments[0], true
}
