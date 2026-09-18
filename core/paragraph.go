package core

// Paragraph: one flow of text made of runs, each with its own marks, and any
// of them tappable. The inline half of text that core.Text cannot express.
//
// # What a Text node cannot do
//
// A Text is one string in one style. Three things kept arriving that are all
// the same thing underneath, a style that changes *inside* a line:
//
//	a word in bold in the middle of a sentence
//	"Read the terms" as a link inside the sentence that mentions them
//	strikethrough and underline, which core.Style never had
//
// A Row of Texts cannot fake it: a Row wraps between its children and not
// inside them, so a sentence built from three Texts breaks at the seams and
// never mid-run. What wraps as one sentence has to be one node on every host,
// and every host has exactly that node: an element of spans on the web, an
// AnnotatedString on Compose, an AttributedString on SwiftUI.
//
//	core.Paragraph([]core.Span{
//	    {Text: "By continuing you accept the "},
//	    {Text: "terms", Underline: true, OnTap: showTerms},
//	    {Text: ". Nothing is ", Italic: true},
//	    {Text: "shared", Bold: true},
//	    {Text: "."},
//	}, core.FontSize(15))
//
// # One node, its runs a prop
//
// The runs travel in the "runs" prop, a list of objects with the richtext
// wire's own keys (richtext.Run marshals the same letters), plus two of its
// own:
//
//	t   text          b  bold      i  italic     u  underline
//	s   strike        c  code (monospace)        fg colour
//	cb  a void callback ID: the run is a link, and tapping it runs OnTap
//
// A prop rather than a child node per run because a run is not a box. It has
// no padding, no border and no size, and a child node would invite every host
// to draw one; GridRow's runs are the precedent. The whole paragraph is one
// node to the reconciler, so a change to any run rewrites the prop, which
// every host redraws in full. Paragraphs are short, and that is the right
// trade against a patch protocol for text inside a node.
//
// The Paragraph's own Style is the text's base: FontSize, FontWeight,
// TextColor, Align, LineHeight, MaxLines, and the box fields. A run's marks
// layer over it; a run's colour replaces TextColor for that run.
//
// # Links
//
// A run with OnTap is a link: drawn in the theme's Primary unless the run
// names its own Color (resolved in Go, so every host draws the same link),
// in the run's own marks otherwise (Underline is the caller's to ask for),
// announced as a link by every screen reader, and reachable by keyboard on
// the web. The tap
// reaches Go on the same void channel a Button's does. The runs' callbacks are
// registered in run order, so their IDs are as stable across passes as any
// other positional callback.
//
// # Accessibility
//
// A paragraph reads as the sentence it is: one text, with its links inside
// it as links, which is what VoiceOver, TalkBack and a browser each do with
// their own inline-link text. It carries no role of its own; AccessibilityLabel
// replaces the whole reading, as it does on a Text.

// Span is one run of a Paragraph.
type Span struct {
	Text string

	Bold      bool
	Italic    bool
	Underline bool
	Strike    bool
	// Code is a monospace run inside the sentence.
	Code bool

	// Color replaces the paragraph's TextColor for this run: "#rrggbb" or
	// any value a Style colour takes. Empty inherits.
	Color string

	// OnTap makes the run a link. Nil is plain text.
	OnTap func()
}

// Paragraph renders runs as one wrapping flow of text; see the file doc.
// props are the paragraph's own: style props for its base text and box, and
// behaviour props (an AccessibilityLabel, a key).
//
// Empty runs are dropped: a run with no text draws nothing on any host, and
// a link with no text would be a tap target nobody can see.
func Paragraph(runs []Span, props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		wire := make([]map[string]any, 0, len(runs))
		for _, r := range runs {
			if r.Text == "" {
				continue
			}
			wire = append(wire, spanWire(ctx, r))
		}
		// A bare base, as core.Text has: body text takes its look from the
		// props (a Typography role through UseStyle, or the style props), not
		// from a component default.
		return leafNode(ctx, "Paragraph", Style{}, map[string]any{
			"runs": wire,
		}, props)
	})
}

// spanWire is one run on the wire. Only the marks a run has are written, the
// wire's omitzero convention, so a plain sentence costs its text and nothing
// else.
func spanWire(ctx *Context, r Span) map[string]any {
	m := map[string]any{"t": r.Text}
	if r.Bold {
		m["b"] = 1
	}
	if r.Italic {
		m["i"] = 1
	}
	if r.Underline {
		m["u"] = 1
	}
	if r.Strike {
		m["s"] = 1
	}
	if r.Code {
		m["c"] = 1
	}
	color := r.Color
	if r.OnTap != nil {
		m["cb"] = ctx.registerCallback(r.OnTap)
		// A link's colour is resolved here, in Go, rather than left to each
		// host's idea of a link: a browser draws an unstyled span, Compose's
		// BasicText an unstyled annotation, and SwiftUI its tint. One colour
		// from the theme is the same link on all four.
		if color == "" {
			color = ctx.Theme().Colors.Primary
		}
	}
	if color != "" {
		m["fg"] = color
	}
	return m
}
