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

## Index

- [Variables](#variables) — `Darcula`, `Light`
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

The names are the ones a file extension or a Markdown fence would give: "go", "json". Matching is exact and lowercase — a caller with a filename resolves it to a language first, because a path is not a language ("go.mod" is not Go) and this package has no business guessing.

An unknown name is deliberately not an error. The caller is a widget with a Language field, and a screen whose editor mis-spells its language should show uncoloured code, not fail to render.

<small>[highlight/highlight.go:250](https://github.com/rohanthewiz/grmob/blob/master/highlight/highlight.go#L250)</small>

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

