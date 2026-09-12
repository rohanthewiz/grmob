package htmlout

import (
	"github.com/rohanthewiz/element"
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/richtext"
)

// core.RichTextEditor, on the HTML side: the document, and nothing else.
//
// A static export has no event loop, so there is no caret, no toolbar and no
// way to type. That is not a degradation to apologize for — a read-only rich
// text editor *is* the display half of the widget (a comment, a note, a
// description), and this is what it looks like when the page cannot run. The
// callback IDs still travel as data attributes from the shared loop in
// renderNode, so a loader with an event loop can wire the live editor out of
// the static document, the same upgrade path an exported MapView offers.
//
// # Why the markup is written raw
//
// richtext.Doc.HTML() escapes every string that reaches its output — the text,
// and a link's href — because a document is a value and the one way that could
// stop being true is a run of text re-entering the page as a tag. What comes
// back is therefore markup this package generated from escaped content, which
// is exactly the shape element's T() is for. Passing it through TE() instead
// would escape the tags this function exists to emit and print `<p>` on the
// screen.
//
// The escaping lives in richtext rather than here for the reason it lives in
// one place at all: three of the four targets build the same document out of
// the same runs, and a rule about user content that is stated per target is a
// rule that is missing on one of them.
func renderRichTextEditor(b *element.Builder, node *core.Node, attrs []string) {
	doc, err := richtext.ParseJSON(getStr(node.Props["doc"]))
	if err != nil {
		// A node whose doc prop is not a document exports as an empty box
		// rather than panicking — the same degradation an Image with no src
		// gets. The prop is written by core.RichTextEditor, so this is reachable
		// only from a hand-built node.
		b.Div(attrs...).R()
		return
	}
	if doc.IsEmpty() {
		// The placeholder, which on a live target is the platform's own prompt
		// and here is the only thing there is to draw. Through TE, unlike the
		// document below: a placeholder is a plain string an author wrote, not
		// markup this package generated.
		e := b.Div(attrs...)
		if prompt := getStr(node.Props["placeholder"]); prompt != "" {
			b.Span("style", richTextPlaceholderStyle).TE(prompt)
		}
		e.R()
		return
	}
	b.Div(attrs...).T(doc.HTML())
}

// richTextPlaceholderStyle is the empty editor's prompt: the node's own ink,
// faded. The same treatment every renderer gives a placeholder, and faded
// rather than recoloured because the one thing a placeholder must not look like
// is content.
const richTextPlaceholderStyle = "opacity:0.45"

// richTextChassis is the fixed rules of a rich-text editor's box.
//
// Only two, and both are about the *document* rather than the control. A
// browser gives <h1>, <p>, <ul> and <blockquote> generous default margins that
// were chosen for a whole page, and inside a note-sized box they read as gaps
// rather than as structure — so the box states a line height the blocks sit on
// and lets its own padding do the insetting. The element margins themselves are
// left alone: an author who wants them tightened has a Style, and a rule here
// would have to reach inside the markup, which is the one thing this exporter
// does not do.
//
// `white-space: normal` is deliberate and is the opposite of what a code
// surface wants: prose wraps, and a rich-text document is prose.
const richTextChassis = "line-height:1.5; white-space:normal"
