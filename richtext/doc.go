// Package richtext is the document model behind core.RichTextEditor: a
// formatted document as Go values, owned by Go on every target.
//
// # Why the value is a document and not a string
//
// A rich-text editor's value cannot be a plain string — the formatting has
// nowhere to live — and it must not be a platform's own markup. An
// NSAttributedString, a Compose AnnotatedString and a contenteditable's
// innerHTML are three different things with three different vocabularies, and
// an app that stored whichever one the user happened to type on would have a
// database it could not read from the other two targets.
//
// So one model, in Go, and each host maps to and from it. What is persisted is
// this package's JSON (bytdb takes it as-is); what crosses the wire is the same
// JSON; what the hosts hold is their own native representation, built from it
// and serialized back on every edit.
//
//	Go            wire              host
//	────────────────────────────────────────────────────────────
//	Doc  ──JSON──▶ doc prop ──▶ NSAttributedString / Spannable /
//	                            contenteditable DOM
//	Doc ◀──JSON── onChange ◀──  the same, serialized back
//
// # The shape, and what is deliberately not in it
//
// Seven block kinds and six marks. No tables, no images, no nesting of lists,
// no colours, no fonts. Each of those is a real feature with a real driver
// behind it, and none of them has one yet; adding one later is a new BlockKind
// or a new Run field plus four host mappings, which is the same cost it would
// be today.
//
// The one structural simplification worth naming is that a list is a *run of
// blocks*, not a container: three bullets are three Bullet blocks in a row,
// exactly as three paragraphs are three Paragraph blocks. HTML wants a <ul>
// around them and HTML() gathers consecutive bullets into one, but the model
// has no list node — which is what keeps the document a flat sequence that a
// host can map to a flat sequence of paragraphs with paragraph styles, which is
// what all three native text engines actually have.
//
// # Markdown is import/export, not the wire
//
// Markdown() and FromMarkdown() exist so a document can leave this system and
// come back, and so that tests read as documents rather than as JSON. They are
// deliberately not the value: Markdown cannot represent a selection-preserving
// edit, and making it the wire would put a Markdown parser in four hosts.
package richtext

import "strings"

// Doc is a whole document: blocks in order, top to bottom.
//
// The zero Doc is a valid empty document, and an empty document is what an
// editor showing its placeholder holds. Nothing here distinguishes "empty" from
// "not yet loaded"; an app that needs to can keep that outside the document,
// where it belongs.
type Doc struct {
	Blocks []Block
}

// Block is one paragraph-level unit: its kind, and the styled runs that make up
// its text.
//
// A block with no runs is a blank line, which is a thing a writer types
// deliberately and which therefore survives every round trip here.
type Block struct {
	Kind BlockKind
	Runs []Run
}

// Run is a span of one block's text in one combination of marks.
//
// The marks are independent booleans rather than a bitmask because they are app
// data as much as wire data — `run.Bold` reads better than `run.Marks&MarkBold`
// at every call site — and because the JSON encoding drops the false ones
// anyway, so the wire cost is the same.
//
// Link is a URL and "" is "not a link". It is a string rather than a bool plus a
// separate href for the obvious reason: the two can never disagree.
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

// marks returns r's formatting without its text, which is what the inline
// parsers and emitters pass around: a run being built inherits the marks of
// whatever it is nested inside and supplies its own text.
func (r Run) marks() Run {
	r.Text = ""
	return r
}

// BlockKind is what a block *is*: a paragraph, a heading, a list item, a quote,
// a code block.
//
// A string rather than an int, and the string is the wire value. That costs a
// few bytes per block and buys two things worth more than the bytes: a document
// in a database is readable by a person, and a new kind cannot be silently
// reinterpreted by an older reader the way an integer can — an unknown kind
// reads as an unknown string and normalizes to Paragraph, where an unknown 8
// would read as whatever kind 8 meant when that reader was written.
//
// The values are also the vocabulary the `block:` editor command uses, so a
// toolbar and a document name a heading with the same string.
type BlockKind string

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

// blockKinds is the census, in the order a toolbar would offer them. Used by
// normalizeKind and by the tests that hold the four hosts to the same list.
var blockKinds = []BlockKind{
	Paragraph, Heading1, Heading2, Heading3, Bullet, Numbered, Quote, BlockCode,
}

// BlockKinds returns every kind this package defines, in toolbar order.
//
// Exported because the vocabulary has four other consumers — the widget's
// toolbar, and the `block:` command in each of the three live hosts — and a
// list restated in each of them is a list that drifts. core.SelectMenuSections
// is exported for the same reason one property over.
func BlockKinds() []BlockKind {
	out := make([]BlockKind, len(blockKinds))
	copy(out, blockKinds)
	return out
}

// normalizeKind maps anything that is not a known kind onto Paragraph.
//
// Called on the way in from JSON, which is the boundary this package does not
// control: the document may have been written by a newer version of this
// package, or by a host that got a `block:` command it should not have had. A
// paragraph is the honest degradation — the *text* is never at risk, only its
// presentation — and it is what every reader already knows how to draw.
func normalizeKind(k BlockKind) BlockKind {
	for _, known := range blockKinds {
		if k == known {
			return k
		}
	}
	return Paragraph
}

// IsList reports whether a kind is one of the two list items, which is the one
// distinction three of the four outputs have to make: HTML gathers a run of
// them into a <ul> or an <ol>, Markdown prefixes each, and the hosts draw a
// bullet or a number in front of the line.
func (k BlockKind) IsList() bool { return k == Bullet || k == Numbered }

// Text is one block's text, with the marks dropped.
func (b Block) Text() string {
	var out strings.Builder
	for _, run := range b.Runs {
		out.WriteString(run.Text)
	}
	return out.String()
}

// PlainText is the whole document with every mark and every block marker
// dropped — the string a search index wants, and the one a plain-text export
// produces.
//
// Blocks are joined with a single newline rather than a blank line between
// them, because a Block is a *line* of the document: an empty block is already
// how a writer says "blank line here", and doubling the separator would turn
// one deliberate blank into two.
func (d Doc) PlainText() string {
	lines := make([]string, len(d.Blocks))
	for i, block := range d.Blocks {
		lines[i] = block.Text()
	}
	return strings.Join(lines, "\n")
}

// IsEmpty reports whether the document has no text in it at all.
//
// Not `len(d.Blocks) == 0`: an editor that has been focused and left holds one
// empty paragraph on every host, and a caller asking "is there anything here"
// means the text. This is what a placeholder is shown on and what a "nothing to
// save" check should ask.
func (d Doc) IsEmpty() bool {
	for _, block := range d.Blocks {
		for _, run := range block.Runs {
			if run.Text != "" {
				return false
			}
		}
	}
	return true
}
