# Package richtext

```go
import "github.com/rohanthewiz/grmob/richtext"
```

Package richtext is the document model behind core.RichTextEditor: a formatted document as Go values, owned by Go on every target.

## Why the value is a document and not a string

A rich-text editor's value cannot be a plain string — the formatting has nowhere to live — and it must not be a platform's own markup. An NSAttributedString, a Compose AnnotatedString and a contenteditable's innerHTML are three different things with three different vocabularies, and an app that stored whichever one the user happened to type on would have a database it could not read from the other two targets.

So one model, in Go, and each host maps to and from it. What is persisted is this package's JSON (bytdb takes it as-is); what crosses the wire is the same JSON; what the hosts hold is their own native representation, built from it and serialized back on every edit.

	Go            wire              host
	────────────────────────────────────────────────────────────
	Doc  ──JSON──▶ doc prop ──▶ NSAttributedString / Spannable /
	                            contenteditable DOM
	Doc ◀──JSON── onChange ◀──  the same, serialized back

## The shape, and what is deliberately not in it

Seven block kinds and six marks. No tables, no images, no nesting of lists, no colours, no fonts. Each of those is a real feature with a real driver behind it, and none of them has one yet; adding one later is a new BlockKind or a new Run field plus four host mappings, which is the same cost it would be today.

The one structural simplification worth naming is that a list is a \*run of blocks\*, not a container: three bullets are three Bullet blocks in a row, exactly as three paragraphs are three Paragraph blocks. HTML wants a \<ul> around them and HTML() gathers consecutive bullets into one, but the model has no list node — which is what keeps the document a flat sequence that a host can map to a flat sequence of paragraphs with paragraph styles, which is what all three native text engines actually have.

## Markdown is import/export, not the wire

Markdown() and FromMarkdown() exist so a document can leave this system and come back, and so that tests read as documents rather than as JSON. They are deliberately not the value: Markdown cannot represent a selection-preserving edit, and making it the wire would put a Markdown parser in four hosts.

## Index

- [`type Block`](#type-block)
    - [`func (Block) Text`](#func-block-text)
- [`type BlockKind`](#type-blockkind)
    - [`func BlockKinds`](#func-blockkinds)
    - [`func (BlockKind) IsList`](#func-blockkind-islist)
- [`type Doc`](#type-doc)
    - [`func FromMarkdown`](#func-frommarkdown)
    - [`func ParseJSON`](#func-parsejson)
    - [`func (Doc) HTML`](#func-doc-html)
    - [`func (Doc) IsEmpty`](#func-doc-isempty)
    - [`func (Doc) JSON`](#func-doc-json)
    - [`func (Doc) Markdown`](#func-doc-markdown)
    - [`func (Doc) MarshalJSON`](#func-doc-marshaljson)
    - [`func (Doc) PlainText`](#func-doc-plaintext)
    - [`func (*Doc) UnmarshalJSON`](#func-doc-unmarshaljson)
- [`type Run`](#type-run)

## Types

### type Block

```go
type Block struct {
	Kind BlockKind
	Runs []Run
}
```

Block is one paragraph-level unit: its kind, and the styled runs that make up its text.

A block with no runs is a blank line, which is a thing a writer types deliberately and which therefore survives every round trip here.

<small>[richtext/doc.go:65](https://github.com/rohanthewiz/grmob/blob/master/richtext/doc.go#L65)</small>

#### func (Block) Text

```go
func (b Block) Text() string
```

Text is one block's text, with the marks dropped.

<small>[richtext/doc.go:168](https://github.com/rohanthewiz/grmob/blob/master/richtext/doc.go#L168)</small>

### type BlockKind

```go
type BlockKind string
```

BlockKind is what a block \*is\*: a paragraph, a heading, a list item, a quote, a code block.

A string rather than an int, and the string is the wire value. That costs a few bytes per block and buys two things worth more than the bytes: a document in a database is readable by a person, and a new kind cannot be silently reinterpreted by an older reader the way an integer can — an unknown kind reads as an unknown string and normalizes to Paragraph, where an unknown 8 would read as whatever kind 8 meant when that reader was written.

The values are also the vocabulary the \`block:\` editor command uses, so a toolbar and a document name a heading with the same string.

<small>[richtext/doc.go:111](https://github.com/rohanthewiz/grmob/blob/master/richtext/doc.go#L111)</small>

```go
const (
	Paragraph BlockKind = "p"
	Heading1  BlockKind = "h1"
	Heading2  BlockKind = "h2"
	Heading3  BlockKind = "h3"
	Bullet    BlockKind = "bullet"
	Numbered  BlockKind = "numbered"
	Quote     BlockKind = "quote"
	// BlockCode is a code block: preformatted, monospace, and never
	// inline-parsed. Its runs carry the source verbatim, newlines included, so
	// one block is one fenced region rather than one line.
	BlockCode BlockKind = "code"
)
```

#### func BlockKinds

```go
func BlockKinds() []BlockKind
```

BlockKinds returns every kind this package defines, in toolbar order.

Exported because the vocabulary has four other consumers — the widget's toolbar, and the \`block:\` command in each of the three live hosts — and a list restated in each of them is a list that drifts. core.SelectMenuSections is exported for the same reason one property over.

<small>[richtext/doc.go:139](https://github.com/rohanthewiz/grmob/blob/master/richtext/doc.go#L139)</small>

#### func (BlockKind) IsList

```go
func (k BlockKind) IsList() bool
```

IsList reports whether a kind is one of the two list items, which is the one distinction three of the four outputs have to make: HTML gathers a run of them into a \<ul> or an \<ol>, Markdown prefixes each, and the hosts draw a bullet or a number in front of the line.

<small>[richtext/doc.go:165](https://github.com/rohanthewiz/grmob/blob/master/richtext/doc.go#L165)</small>

### type Doc

```go
type Doc struct {
	Blocks []Block
}
```

Doc is a whole document: blocks in order, top to bottom.

The zero Doc is a valid empty document, and an empty document is what an editor showing its placeholder holds. Nothing here distinguishes "empty" from "not yet loaded"; an app that needs to can keep that outside the document, where it belongs.

<small>[richtext/doc.go:56](https://github.com/rohanthewiz/grmob/blob/master/richtext/doc.go#L56)</small>

#### func FromMarkdown

```go
func FromMarkdown(src string) (Doc, error)
```

FromMarkdown reads the subset back.

The error is always nil today and the signature keeps it anyway, because the alternative is worse than an unused return: this is the \*import\* door, the thing most likely to grow a real refusal (a size limit, a fence that never closes), and a signature change later would break every caller. A parse that cannot make sense of a line makes it a paragraph, which is what every forgiving Markdown reader does.

<small>[richtext/markdown.go:216](https://github.com/rohanthewiz/grmob/blob/master/richtext/markdown.go#L216)</small>

#### func ParseJSON

```go
func ParseJSON(data string) (Doc, error)
```

ParseJSON reads a document from the wire form.

A forwarder rather than a call to json.Unmarshal at each site, because the three places that do it (the node's onChange, a database read, a test) all want the same thing: a Doc and an error, with no pointer dance.

<small>[richtext/json.go:197](https://github.com/rohanthewiz/grmob/blob/master/richtext/json.go#L197)</small>

#### func (Doc) HTML

```go
func (d Doc) HTML() string
```

HTML renders the document as a fragment: block elements at the top level, with no wrapper around them.

No wrapper, so the caller decides what box this goes in — which for the one in-tree consumer is a core node's own div, and for anything else is whatever it already has.

<small>[richtext/html.go:39](https://github.com/rohanthewiz/grmob/blob/master/richtext/html.go#L39)</small>

#### func (Doc) IsEmpty

```go
func (d Doc) IsEmpty() bool
```

IsEmpty reports whether the document has no text in it at all.

Not \`len(d.Blocks) == 0\`: an editor that has been focused and left holds one empty paragraph on every host, and a caller asking "is there anything here" means the text. This is what a placeholder is shown on and what a "nothing to save" check should ask.

<small>[richtext/doc.go:198](https://github.com/rohanthewiz/grmob/blob/master/richtext/doc.go#L198)</small>

#### func (Doc) JSON

```go
func (d Doc) JSON() string
```

JSON is Marshal without the error, for the call sites that cannot fail and should not have to say so.

MarshalJSON here can only fail if encoding/json cannot encode a struct of strings and ints, which it can. The node builder needs a string and a render pass has nowhere to put an error, so this is where that is stated once instead of at every call site with a dropped \`\_\`.

<small>[richtext/json.go:180](https://github.com/rohanthewiz/grmob/blob/master/richtext/json.go#L180)</small>

#### func (Doc) Markdown

```go
func (d Doc) Markdown() string
```

Markdown renders the document as CommonMark (plus one \`\<u>\`; see above).

Blocks are separated by a blank line, which is what makes two consecutive paragraphs two paragraphs on the way back in. List items are the exception: consecutive bullets are written on consecutive lines, because a blank line between them makes a "loose" list in CommonMark and, more to the point, makes them read as separate lists.

<small>[richtext/markdown.go:55](https://github.com/rohanthewiz/grmob/blob/master/richtext/markdown.go#L55)</small>

#### func (Doc) MarshalJSON

```go
func (d Doc) MarshalJSON() ([]byte, error)
```

MarshalJSON writes the wire form. See the file doc for the shape.

<small>[richtext/json.go:98](https://github.com/rohanthewiz/grmob/blob/master/richtext/json.go#L98)</small>

#### func (Doc) PlainText

```go
func (d Doc) PlainText() string
```

PlainText is the whole document with every mark and every block marker dropped — the string a search index wants, and the one a plain-text export produces.

Blocks are joined with a single newline rather than a blank line between them, because a Block is a \*line\* of the document: an empty block is already how a writer says "blank line here", and doubling the separator would turn one deliberate blank into two.

<small>[richtext/doc.go:184](https://github.com/rohanthewiz/grmob/blob/master/richtext/doc.go#L184)</small>

#### func (*Doc) UnmarshalJSON

```go
func (d *Doc) UnmarshalJSON(data []byte) error
```

UnmarshalJSON reads the wire form, normalizing as it goes.

Everything that arrives here came from outside Go — a host's serializer, a database row written by an older version of this package — so nothing is trusted to be in the vocabulary. An unknown block kind becomes a paragraph (see normalizeKind); a mark is read as truthy rather than as a particular type (markFlag); a run with no text is dropped, because an empty run is invisible on every target and costs a span on all four.

What is deliberately \*not\* dropped is an empty block: that is a blank line, and a blank line is something a writer typed.

<small>[richtext/json.go:142](https://github.com/rohanthewiz/grmob/blob/master/richtext/json.go#L142)</small>

### type Run

```go
type Run struct {
	Text      string
	Bold      bool
	Italic    bool
	Underline bool
	Strike    bool
	// Code is inline code — a monospace span inside a sentence, not a code
	// block. BlockCode is the block-level one.
	Code bool
	Link string
}
```

Run is a span of one block's text in one combination of marks.

The marks are independent booleans rather than a bitmask because they are app data as much as wire data — \`run.Bold\` reads better than \`run.Marks&MarkBold\` at every call site — and because the JSON encoding drops the false ones anyway, so the wire cost is the same.

Link is a URL and "" is "not a link". It is a string rather than a bool plus a separate href for the obvious reason: the two can never disagree.

<small>[richtext/doc.go:79](https://github.com/rohanthewiz/grmob/blob/master/richtext/doc.go#L79)</small>

