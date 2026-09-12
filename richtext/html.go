package richtext

import (
	"html"
	"strings"
)

// HTML output: the document as markup, for the one target that draws a
// rich-text editor without being able to edit it.
//
// htmlout exports a static document, so a core.RichTextEditor there is its
// content and nothing else — no caret, no toolbar, no way to type. That is not
// a degradation to apologize for: the read-only editor *is* the display half of
// the widget (a comment, a note, a description), and this is what it looks like
// when the page has no event loop.
//
// It is also the obvious thing to hand to anything outside this system that
// wants the document: an email body, a server-rendered page, a preview.
//
// # Escaping
//
// Everything a document contains is user-originated — it is a *text editor* —
// so every string that reaches the output goes through html.EscapeString,
// including a link's href. A document is a value, not markup, and the one way
// that could stop being true is a run of text re-entering the page as a tag.
//
// A link's scheme is deliberately not filtered here. `javascript:` in an href
// is a real concern and it is the *host's*: this package does not know whether
// its output is going into a sanitizing renderer, an email, or a file. An app
// that accepts documents from other people should filter links where it accepts
// them, which is also the only place it knows what its own policy is.

// HTML renders the document as a fragment: block elements at the top level,
// with no wrapper around them.
//
// No wrapper, so the caller decides what box this goes in — which for the one
// in-tree consumer is a core node's own div, and for anything else is whatever
// it already has.
func (d Doc) HTML() string {
	var out strings.Builder
	// The list currently open, so that a run of consecutive Bullet blocks
	// becomes one <ul> rather than three. The model has no list container — see
	// the package doc for why — so the gathering happens here, which is the one
	// output that needs it.
	openList := BlockKind("")

	closeList := func() {
		switch openList {
		case Bullet:
			out.WriteString("</ul>")
		case Numbered:
			out.WriteString("</ol>")
		}
		openList = ""
	}

	for _, block := range d.Blocks {
		if block.Kind.IsList() {
			if openList != block.Kind {
				closeList()
				openList = block.Kind
				if block.Kind == Bullet {
					out.WriteString("<ul>")
				} else {
					out.WriteString("<ol>")
				}
			}
			out.WriteString("<li>" + inlineHTML(block.Runs) + "</li>")
			continue
		}
		closeList()

		switch block.Kind {
		case Heading1:
			out.WriteString("<h1>" + inlineHTML(block.Runs) + "</h1>")
		case Heading2:
			out.WriteString("<h2>" + inlineHTML(block.Runs) + "</h2>")
		case Heading3:
			out.WriteString("<h3>" + inlineHTML(block.Runs) + "</h3>")
		case Quote:
			out.WriteString("<blockquote>" + inlineHTML(block.Runs) + "</blockquote>")
		case BlockCode:
			// The text, escaped and otherwise untouched: a code block's marks
			// are not expressible and its white space is the content. <pre>
			// keeps the newlines, which is the whole reason it is the element.
			out.WriteString("<pre><code>" + html.EscapeString(block.Text()) + "</code></pre>")
		default:
			// An empty paragraph is a blank line the writer typed, and an empty
			// <p> collapses to nothing in every browser. The non-breaking space
			// is what gives it back its height — the same problem, and the same
			// fix, that a GridRow with no runs solves with a min-height.
			body := inlineHTML(block.Runs)
			if body == "" {
				body = "&nbsp;"
			}
			out.WriteString("<p>" + body + "</p>")
		}
	}
	closeList()
	return out.String()
}

// inlineHTML writes one block's runs, nesting the marks in the same order
// Markdown does — link outermost, code innermost — so the two outputs describe
// the same tree.
func inlineHTML(runs []Run) string {
	var out strings.Builder
	for _, run := range runs {
		out.WriteString(run.html())
	}
	return out.String()
}

func (r Run) html() string {
	text := html.EscapeString(r.Text)
	// Code first, and it does not suppress the other marks the way Markdown's
	// backtick does: HTML can express bold code, so nothing is lost here and
	// the JSON's six marks all survive.
	if r.Code {
		text = "<code>" + text + "</code>"
	}
	if r.Underline {
		text = "<u>" + text + "</u>"
	}
	if r.Strike {
		text = "<s>" + text + "</s>"
	}
	if r.Italic {
		// <em> and <strong> rather than <i> and <b>: the semantic pair is what
		// a screen reader announces with emphasis, and the presentational pair
		// is what a browser draws identically and says nothing about.
		text = "<em>" + text + "</em>"
	}
	if r.Bold {
		text = "<strong>" + text + "</strong>"
	}
	if r.Link != "" {
		text = `<a href="` + html.EscapeString(r.Link) + `">` + text + "</a>"
	}
	return text
}
