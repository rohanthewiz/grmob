# Package highlight

```go
import "github.com/rohanthewiz/grmob/highlight"
```

Go syntax highlighting.

The lexer is go/scanner from the standard library, not a set of regexps. That is the whole design decision here and it buys three things a hand-rolled tokenizer would each have to earn separately: the input is real Go, so the only tokenizer guaranteed to agree with the reader's own editor is the compiler's; string literals, rune literals and block comments stop being special cases (a \`//\` inside a string is a string, and the scanner knows that without being told); and the token set cannot drift as the language grows.

It is lexical only. There is no type information here and no parse tree, so this file colours what a token \*is\*, never what it means — see classifyGo for the one place that looks past a single token, and for what is deliberately left uncoloured because a lexer cannot know it.

Package highlight turns source text into core.GridRow values: one styled row per line, ready for a core.TextGrid or a core.CodeEditor.

It was lifted out of the tutorial, where a private go/scanner highlighter had been colouring the lesson snippets. Two callers now want the same thing and neither is the tutorial — comps.CodeEditor colours an editable buffer, and any screen that shows a config file or a payload wants the same rows — so the lexers live here, behind one interface, and the tutorial is a caller like the rest.

## The shape, and why it is rows rather than spans

A row is a line and a line is what both consumers address. core.TextGrid gives each row its own node, so a changed line is one patch; core.CodeEditor applies a row's decoration only when the row's text still equals the host's own line (the stale-line rule in core/codeeditor.go). Both depend on the rows being \*line-for-line\* with the input, which is this package's one hard contract:

	len(Rows(src, scheme)) == strings.Count(src, "\n") + 1

It holds for every lexer here, including the fallbacks, and TestEveryLexerIsLineForLine pins it.

## The classification is lexical, and per byte

Every lexer in the package works the same way: it produces one class byte per \*source byte\*, and the shared rowsOf/runsOf pair slices that array at the newlines. The reason is that a block comment or a raw string crosses line boundaries, and a per-byte array makes that split a slice expression with no case analysis at all — the part of a multi-line comment on line 3 is classes\[start:end] for line 3. Slicing at a newline can never split a rune, because a newline is a single ASCII byte and every byte of a multi-byte rune belongs to the same token.

A byte per class rather than a colour per byte because the array is as long as the source; colours would make it eight times the size for no gain, and the class→colour step is where a Scheme gets applied.

JSON syntax highlighting.

A hand lexer rather than encoding/json, and the reason is the same one that makes go/scanner right for Go and a \*parser\* wrong for both: the input is very often invalid. A CodeEditor's buffer is mid-edit most of the time — half a key typed, a closing brace not there yet — and encoding/json answers "invalid" for all of it, which would mean a document that loses every colour the moment someone puts the cursor in it.

So this lexer never fails. It classifies what it can see and treats anything it does not recognize as plain ink, which degrades exactly the way an editor should: the bytes you have finished typing are coloured, the ones you are in the middle of are not yet.

HTML and XML highlighting.

Markup is not a token stream the table in table.go can describe: what a word means depends on whether it is inside a tag, and the text between tags is prose. So it gets its own small lexer, with the same class bytes and the same unterminated rule as the table (see table.go):

	<!-- … -->            block comment (only once closed)
	<name  </name  <?xml  the name: Keyword
	attr=                 an attribute name inside a tag: Func
	"…"  '…'              an attribute value inside a tag: String
	everything else       plain ink, including the text between tags and
	                      the contents of <script> and <style>

The roles mirror JSON's: a tag is the language's vocabulary and an attribute name is a key.

Table-driven highlighting for the languages that have no lexer of their own in this package.

Go has go/scanner and JSON has a grammar small enough to lex exactly. The rest — JavaScript, Python, shell, SQL, YAML and the C family among them — are coloured by one lexer driven by a per-language table (langSpec): its keywords, its comment and string delimiters, and a few flags for the shapes that differ (a \`#\` that only starts a comment after whitespace, a YAML key before its colon, SQL's case-insensitive keywords). HTML and XML are tags rather than tokens and get a small lexer of their own (markup.go).

## Why a table and not a lexer per language

The colour roles a Scheme has are few: keyword, string, number, comment, block comment, and the identifier being called or the key being set. At that resolution most languages differ only in their vocabulary and their delimiters, so a table states each language in a few lines that read like its spec, and every language shares one scanning loop that the property tests in highlight\_test.go hold to the package's contracts (line-for-line, every byte preserved, maximal runs). Twenty hand-written lexers would be twenty places for the run arithmetic to go wrong.

The price is precision: a table cannot know that \`'a\` is a Rust lifetime or that \`$((…))\` is shell arithmetic. Where a construct cannot be described in the table it is left uncoloured rather than guessed at, which is the package's standing rule (see Highlighter).

## Unterminated constructs

Go's lexer gives up on the whole input when go/scanner reports any error. There is no reference scanner here to defer to, so the rule is local instead: a string or block comment is coloured only once its closing delimiter is found. An opener with no closer is plain ink, and scanning resumes on the byte after it. Colours therefore never run on past the point where the lexer lost its place, which is the failure the Go fallback exists to prevent, and a fence holding a half-written line (or an apostrophe in a shell word) keeps its colours everywhere else.

## Index

- [Variables](#variables) — `Darcula`, `Light`
- [`func Languages`](#func-languages)
- [`func LineComment`](#func-linecomment)
- [`type Highlighter`](#type-highlighter)
    - [`func ForLanguage`](#func-forlanguage)
    - [`func Go`](#func-go)
    - [`func JSON`](#func-json)
    - [`func Plain`](#func-plain)
- [`type Scheme`](#type-scheme)

## Variables

Darcula is JetBrains' scheme of the same name, in its own hex values rather than an approximation, so a reader who lives in GoLand or IntelliJ sees a snippet in the colours their editor already uses for the same tokens.

This is the scheme the tutorial has always used, and the byte-for-byte identity of its output is pinned from the tutorial side (TestHighlightClassifiesGoTokens and friends) as well as from here.

```go
var Darcula = Scheme{
	Keyword:    "#CC7832",
	String:     "#6A8759",
	Number:     "#6897BB",
	Comment:    "#808080",
	DocComment: "#629755",
	Func:       "#FFC66D",
	Ink:        "#A9B7C6",
	Bg:         "#2B2B2B",
}
```

<small>[highlight/highlight.go:82](https://github.com/rohanthewiz/grmob/blob/master/highlight/highlight.go#L82)</small>

Light is the same six roles chosen for a light surface. It is not a mechanical inversion of Darcula — inverting a hue gives a colour, not a legible one — but the GitHub-light family, which is the scheme most readers have seen on a light background.

It exists because a code surface on a light theme has to read as code without becoming a dark rectangle in the middle of a light screen, and no palette \*role\* means "the colour of a keyword": the theme has Ink, Surface and the accents, and none of them is about syntax. So a scheme is a scheme, and comps.CodeEditor picks between these two by the brightness of the theme's own Surface rather than by inventing token colours from roles.

```go
var Light = Scheme{
	Keyword:    "#CF222E",
	String:     "#0A3069",
	Number:     "#0550AE",
	Comment:    "#6E7781",
	DocComment: "#6E7781",
	Func:       "#8250DF",
	Ink:        "#24292F",
	Bg:         "#F6F8FA",
}
```

<small>[highlight/highlight.go:104](https://github.com/rohanthewiz/grmob/blob/master/highlight/highlight.go#L104)</small>

## Functions

### func Languages

```go
func Languages() []string
```

Languages lists every language name ForLanguage lexes, sorted. A caller mapping its own names (a Markdown fence's "js", a file's ".py") onto these can check its table against this list rather than against the docs.

<small>[highlight/table.go:457](https://github.com/rohanthewiz/grmob/blob/master/highlight/table.go#L457)</small>

### func LineComment

```go
func LineComment(lang string) (prefix string, ok bool)
```

LineComment is the line-comment marker of a language ForLanguage lexes: "#" for Python and shell, "--" for SQL and Lua, "//" for the C family and JavaScript. ok is true for every language this package knows, and prefix is "" for one with no line comment (JSON, CSS, HTML, XML), so an editor's comment command can tell "no marker" apart from "unknown language". comps.CodeEditor uses it to pick the marker its comment command toggles.

<small>[highlight/table.go:472](https://github.com/rohanthewiz/grmob/blob/master/highlight/table.go#L472)</small>

## Types

### type Highlighter

```go
type Highlighter interface {
	Rows(src string, scheme Scheme) []core.GridRow
}
```

Highlighter turns source into one styled row per line.

Rows must be line-for-line with the input — see the package doc. A highlighter that cannot make sense of its input returns plain rows rather than half-coloured ones: colours that have lost their place are worse than no colours, because a run of code tinted as a string actively misleads.

<small>[highlight/highlight.go:49](https://github.com/rohanthewiz/grmob/blob/master/highlight/highlight.go#L49)</small>

#### func ForLanguage

```go
func ForLanguage(name string) Highlighter
```

ForLanguage returns the Highlighter for a language name, or Plain for one this package does not lex.

The names are each language's own lowercase name: "go", "json", "javascript", "python", "bash" and the rest Languages lists. Matching is exact — a caller with a filename or a Markdown fence's short form ("js", "py") resolves it to a language first, because a path is not a language ("go.mod" is not Go) and this package has no business guessing.

Go and JSON have lexers of their own (golang.go, json.go), HTML and XML share one (markup.go), and every other language is a row in the table in table.go.

An unknown name is deliberately not an error. The caller is a widget with a Language field, and a screen whose editor mis-spells its language should show uncoloured code, not fail to render.

<small>[highlight/highlight.go:255](https://github.com/rohanthewiz/grmob/blob/master/highlight/highlight.go#L255)</small>

#### func Go

```go
func Go() Highlighter
```

Go returns the Go highlighter.

A function rather than an exported variable so the returned value is always a fresh, immutable, stateless thing: a Highlighter is held by a widget across renders and may be called from whatever goroutine a render pass runs on, and a package-level value invites someone to give it a field.

<small>[highlight/golang.go:32](https://github.com/rohanthewiz/grmob/blob/master/highlight/golang.go#L32)</small>

#### func JSON

```go
func JSON() Highlighter
```

JSON returns the JSON highlighter. See Go() for why this is a function.

<small>[highlight/json.go:19](https://github.com/rohanthewiz/grmob/blob/master/highlight/json.go#L19)</small>

#### func Plain

```go
func Plain() Highlighter
```

Plain returns a Highlighter that colours nothing: one run per line, in the grid's own ink. It is what ForLanguage answers for a language nothing here lexes, and it is a useful value in its own right — a CodeEditor over a text file still wants the monospace grid, the gutter and the tab handling.

<small>[highlight/highlight.go:226](https://github.com/rohanthewiz/grmob/blob/master/highlight/highlight.go#L226)</small>

### type Scheme

```go
type Scheme struct {
	Keyword    string // keywords, and anything that reads as part of the language
	String     string // string, rune and character literals
	Number     string // numeric literals
	Comment    string // line comments
	DocComment string // block comments, which several schemes colour differently
	Func       string // an identifier being declared or called; a JSON object key
	Ink        string // the default foreground of the surface
	Bg         string // the surface itself
}
```

Scheme is the palette a highlighter paints with. Every field is a CSS colour ("#rrggbb"); an empty field means "inherit", which on the wire is an empty GridRun.Fg and in every renderer is the grid's own text colour.

Ink and Bg are not used by Rows at all — a run in the default ink carries no colour, which is what keeps a full screen of plain code from putting the same hex on the wire a thousand times. They are here because the \*surface\* is part of a scheme: comps.CodeEditor paints Bg behind the buffer and sets Ink as the grid's TextColor, so a caller who names a scheme gets one coherent picture rather than Darcula's token colours over the app's own background.

<small>[highlight/highlight.go:64](https://github.com/rohanthewiz/grmob/blob/master/highlight/highlight.go#L64)</small>

