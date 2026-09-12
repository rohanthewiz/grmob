package core

import (
	"encoding/json"
	"strings"

	"github.com/rohanthewiz/grmob/richtext"
)

// RichTextEditor is an editable formatted document: bold, italics, headings,
// lists, quotes, links.
//
//	core.RichTextEditor(state.Doc, func(d richtext.Doc) { state.Doc = d },
//	    core.Placeholder("Write something…"),
//	    core.EditorTarget(ref),
//	    core.OnRichSelectionChange(func(sel core.RichSelection) { bar.Set(sel) }),
//	)
//
// components.RichTextEditor is the widget over it — it builds the toolbar and
// wires the link prompt — and is what application code should reach for.
//
// # The value is a document, and the document is Go's
//
// richtext.Doc crosses the wire as JSON and each host maps it to and from its
// own text representation. That is the whole architectural decision here and
// the package doc for richtext carries the argument: an NSAttributedString, a
// Spannable and a contenteditable's innerHTML are three vocabularies, and an
// app that stored whichever one the user happened to type on would have a
// database its other two targets could not read.
//
// # The rules it shares with CodeEditor, and the one it does not
//
// The echo guard is unchanged — the doc JSON is compared exactly as a
// TextArea's value is, so Go's echo of the host's own last onChange never
// resets the caret. Commands are epoch-stamped props, same mechanism, same
// adopt-on-first-sight rule (see core/editor.go).
//
// The stale-line rule is *not* here and does not need to be. A CodeEditor has
// two facts about one buffer — the text and a decoration of it computed
// separately — which can disagree for a frame. Here the doc *is* the styled
// buffer: there is nothing to compare it against, because the formatting and
// the characters arrive together.
//
// # Known gaps in v1
//
// core.Focus and core.DismissKeyboard reach an editor — the type is in
// focusableLeafTypes, exactly as CodeEditor is. On both phones the control is
// a classic text view hosted inside the declarative framework (a UITextView,
// an EditText), so neither renderer can hand the command to its platform's own
// focus system and each drives the responder directly; see core/focus.go.
//
// Collaborative editing, images, tables and per-run fonts are non-goals; each
// is a driver away and none changes the design above.
func RichTextEditor(doc richtext.Doc, onChange func(richtext.Doc), props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		return leafNode(ctx, "RichTextEditor", ctx.Theme().Components.TextArea, map[string]any{
			// The document as JSON, which is both the wire form and what an app
			// persists. Marshalling here rather than in each host is what keeps
			// the three host serializers reading one shape.
			"doc": doc.JSON(),
			"onChange": ctx.registerTextCallback(func(payload string) {
				if onChange == nil {
					return
				}
				next, err := richtext.ParseJSON(payload)
				if err != nil {
					// A host sending something this package cannot read is a bug
					// in that host, and the right response is to drop the edit
					// rather than to hand app code an empty document — which
					// would be echoed straight back and would delete the note.
					return
				}
				onChange(next)
			}),
			// Seeded for the reason core.CodeEditor seeds its options: an
			// update-props patch carries the whole new props map and every host
			// iterates the keys it contains, so a key that stops being written
			// is invisible on the far side. "Off" has to be a value.
			"readOnly":    false,
			"placeholder": "",
		}, props)
	})
}

// The commands a core.RichTextEditor understands. Each toggles on the current
// selection, or sets the typing attributes when the selection is empty — which
// is what makes "press bold, then type" work.
const (
	EditBold      = "bold"
	EditItalic    = "italic"
	EditUnderline = "underline"
	EditStrike    = "strike"
	// EditCode is the inline mark — a monospace span inside a sentence. The
	// block-level one is EditBlock(richtext.BlockCode).
	EditCode = "code"
	// EditUnlink removes the link from the selection, leaving its text.
	EditUnlink = "unlink"
	// EditUndo and EditRedo are the editing history. Each host uses its own
	// (UIKit's UndoManager, EditText's), except the web, where the runtime keeps
	// a stack of Docs: a browser's native history does not survive the
	// programmatic attribute edits the other commands make.
	EditUndo = "undo"
	EditRedo = "redo"
)

// EditLink is the command that makes the selection a link to url.
//
// A function rather than a constant because the command carries an argument,
// and the argument rides in the string — the command channel is one prop and
// widening it to a map would change the shape all four hosts read for the sake
// of one command. "link:" is the prefix; everything after the first colon is
// the URL, so a URL containing colons (every one of them does) is intact.
//
// An empty url is EditUnlink's job and is refused here rather than sent as a
// link to nowhere.
func EditLink(url string) string {
	if url == "" {
		return EditUnlink
	}
	return "link:" + url
}

// EditBlock is the command that makes every block the selection touches the
// given kind.
//
// The kind's wire value *is* richtext.BlockKind's, which is why that type is a
// string: a toolbar naming a heading and a document holding one use the same
// token, so there is no second table to keep in step.
func EditBlock(kind richtext.BlockKind) string {
	return "block:" + string(kind)
}

// RichSelection is what a rich-text editor reports about its caret: where it
// is, and what formatting is active there.
//
// The marks matter more than the offsets, and that is the reason this is a
// struct rather than the two ints a CodeEditor reports. A toolbar has to show
// its bold button as *on* when the caret is inside bold text, and nothing in Go
// can work that out — the document is Go's, but where the caret is inside it is
// the host's.
//
// Start and End are byte offsets into the document's plain text
// (richtext.Doc.PlainText), which is the one coordinate system all four hosts
// can produce and which is stable across the marks. They are there for a status
// line and for "is anything selected"; a command never needs them, because
// every command acts on the host's own selection.
type RichSelection struct {
	Start, End int

	Bold      bool
	Italic    bool
	Underline bool
	Strike    bool
	Code      bool

	// Link is the URL under the caret, or "" when there is none. A toolbar uses
	// it to decide between offering "Link" and offering "Unlink", and to
	// pre-fill the prompt when editing one.
	Link string

	// Block is the kind of the block the caret is in. An empty value means the
	// host had nothing to say, which happens on a document with no blocks yet.
	Block richtext.BlockKind
}

// HasSelection reports whether anything is actually selected, as opposed to a
// bare caret. The distinction is what a "Link" button needs: linking an empty
// selection has nothing to attach to.
func (s RichSelection) HasSelection() bool { return s.End > s.Start }

// wireSelection is the shape the hosts send, which is the plan's:
//
//	{"s":12,"e":18,"marks":["bold","italic"],"link":"https://x","block":"h2"}
//
// The marks travel as a list of names rather than as five booleans because the
// list is what the hosts can build cheaply: each of them asks its text engine
// "which attributes are set here" and gets back a set, and appending the names
// it recognizes is one loop with no per-mark plumbing.
type wireSelection struct {
	Start int                `json:"s"`
	End   int                `json:"e"`
	Marks []string           `json:"marks"`
	Link  string             `json:"link"`
	Block richtext.BlockKind `json:"block"`
}

// parseRichSelection reads one selection report.
//
// Total: anything it cannot read is refused with ok=false rather than delivered
// as a zeroed selection, which a toolbar would draw as "nothing is bold" — a
// lie that looks exactly like the truth.
func parseRichSelection(payload string) (RichSelection, bool) {
	var in wireSelection
	if err := json.Unmarshal([]byte(payload), &in); err != nil {
		return RichSelection{}, false
	}
	sel := RichSelection{
		Start: in.Start,
		End:   in.End,
		Link:  in.Link,
		Block: richtext.BlockKind(in.Block),
	}
	if sel.Start > sel.End {
		// A drag upward is reported end-first by some hosts and is an ordinary
		// selection, so it is normalized rather than refused — the same rule
		// parseSelection applies to a CodeEditor's pair.
		sel.Start, sel.End = sel.End, sel.Start
	}
	for _, mark := range in.Marks {
		switch strings.ToLower(mark) {
		case EditBold:
			sel.Bold = true
		case EditItalic:
			sel.Italic = true
		case EditUnderline:
			sel.Underline = true
		case EditStrike:
			sel.Strike = true
		case EditCode:
			sel.Code = true
		}
		// A mark this version does not know is ignored rather than refused: a
		// newer host is allowed to report more than an older Go can name.
	}
	return sel, true
}

// OnRichSelectionChange reports a rich-text editor's caret and the formatting
// active at it.
//
// A separate builder from OnSelectionChange rather than an overload, because
// the two carry different things and the difference is the point: a code
// editor's selection is two offsets, and a rich editor's is the state a toolbar
// has to draw. They share the prop name on the wire ("onSelectionChange") and
// the text channel; what differs is the payload each node type sends and the
// parse core does before app code sees it.
func OnRichSelectionChange(handler func(RichSelection)) BehaviorProp {
	if handler == nil {
		return nil
	}
	return behaviorFunc(func(ctx *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["onSelectionChange"] = ctx.registerTextCallback(func(payload string) {
			if sel, ok := parseRichSelection(payload); ok {
				handler(sel)
			}
		})
	})
}
