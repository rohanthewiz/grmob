package core

// CodeEditor is an editable monospace buffer with syntax colour, a line-number
// gutter and the keyboard behaviour a programmer's editor has.
//
//	core.CodeEditor(state.Src, onChange, highlight.Go().Rows(state.Src, highlight.Darcula),
//	    core.LineNumbers(),
//	    core.TabSize(4),
//	    core.OnSelectionChange(func(start, end int) { ... }),
//	    core.Height("240px"),
//	)
//
// components.CodeEditor is the widget over it — it runs the highlighter, wires
// a toolbar and picks a scheme from the theme — and is what application code
// should reach for. This is the primitive it is built on.
//
// # Why a node type rather than a composition
//
// The obvious pure-Go construction is a transparent core.TextArea in a ZStack
// over a core.TextGrid: Go colours the grid, the user types into the invisible
// field above it, and the two line up. They do not line up, and cannot:
// core.Style has no font-family, so the TextArea is drawn in the platform's
// proportional UI face while the grid is monospace, and no amount of styling
// from outside can pitch-match a SwiftUI TextEditor or a Compose
// BasicTextField to a separate text view. The overlay has to be built by
// something that owns *both* elements, which is the renderer. That is the
// whole argument for this being a node type, and it is the same argument
// TextGrid makes one step earlier.
//
// # The three rules every host implements
//
//  1. Echo guard, unchanged from TextArea. `value` from Go is applied only
//     when it is not an echo of the host's own last onChange. The buffer is
//     the host's while focused and Go's otherwise — see GrMobTextField's
//     pendingEchoes in either native renderer for the bookkeeping, which this
//     node reuses verbatim rather than restating.
//
//  2. Decoration is advisory and per line. The rows are GridRow children,
//     exactly as core.TextGrid builds them, and a host applies row N's styling
//     only if that row's concatenated text equals the host's current line N. A
//     line that disagrees — Go is a keystroke behind, which it is for a few
//     milliseconds after every keypress — is drawn in plain ink until the next
//     patch. Never the other way round: decoration never rewrites the buffer,
//     so a lexer that is wrong can make the screen ugly and can never make it
//     lose text.
//
//  3. Commands are epoch-stamped props. See core/editor.go.
//
// # The behaviour the contract tests pin
//
// Monospace, no wrapping, horizontal scroll. Tab inserts an indent rather than
// moving focus. Enter copies the previous line's leading white space.
// Autocorrect, autocapitalization and smart quotes are off — every one of them
// corrupts source. A readOnly buffer is still selectable and still shows a
// caret, because a code block the user cannot copy out of is a screenshot.
//
// # Focus commands
//
// core.Focus and core.DismissKeyboard reach an editor: CodeEditor is in
// focusableLeafTypes, so every command stamps it like any other text control
// and a background tap puts its keyboard away.
//
// The one thing worth knowing is where the command lands. This node is a
// *box* — a scroll container holding a gutter and a buffer — where an Input is
// the control itself, so each renderer resolves the stamp to the buffer rather
// than applying it where it arrived. The gutter is chrome and never takes the
// caret. htmlout applies it nowhere at all: its editor is a read-only snapshot
// with no editable element to autofocus.
//
// # Known gaps in v1
//
// Nothing that needs a caret is exercised by any harness here — the IME
// composing region, a hardware Tab on iPad, and paste from another app are
// arguments rather than tests.
//
// # value and rows are two facts about one buffer, and they can disagree
//
// value is the text; rows are a *decoration of that text*, computed in Go from
// that same text. They always agree at the moment Go builds them, and are
// allowed to disagree with the host mid-keystroke, which is what rule 2 is
// about. A caller that computes rows from something other than value has not
// broken anything — the rows simply never match and the buffer is drawn plain.
func CodeEditor(value string, onChange func(string), rows []GridRow, props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		// Every option is seeded with its default rather than left absent, and
		// the behavior props below overwrite. This is not tidiness: an
		// update-props patch carries the *whole new props map* and every host
		// iterates the keys it contains, so a key that stops being written
		// between two passes is invisible on the far side — an editor that
		// lost its LineNumbers() would keep its gutter forever. Seeding means
		// every pass writes every key, so "off" is a value rather than an
		// absence.
		n := leafNode(ctx, "CodeEditor", ctx.Theme().Components.TextArea, map[string]any{
			"value":    value,
			"onChange": ctx.registerTextCallback(onChange),
			// The gutter is off by default: a two-line snippet with line
			// numbers reads as a listing rather than as code.
			"lineNumbers": false,
			"readOnly":    false,
			// Four spaces, which is what the Tab key inserts and what one
			// EditIndent is worth. Zero means a literal tab character — see
			// TabSize.
			"tabSize": 4,
			// The line-comment prefix EditCommentLine toggles. "//" covers Go,
			// C, JavaScript, Swift, Kotlin and Rust, which is most of what a
			// code editor in an app is showing; a language whose comments are
			// spelled differently sets its own, and one with no line comments
			// at all sets "" and gets a command that does nothing.
			"commentPrefix": "//",
		}, props)
		// The rows are children for core.TextGrid's reason, which applies
		// harder here: the reconciler pairs children by index and compares
		// props by value, so re-highlighting a thousand-line buffer after one
		// keystroke emits update-props for the lines that actually changed
		// colour and nothing else. As one prop, every keystroke would put the
		// whole decoration on the wire.
		//
		// Built by the same helper TextGrid uses, so a row is the same node on
		// the wire in both and every renderer's row reader is one function.
		n.Children = make([]*Node, len(rows))
		for i, row := range rows {
			n.Children[i] = gridRowNode(row)
		}
		return n
	})
}

// LineNumbers turns on the gutter.
//
// The gutter is drawn by the host rather than being part of the buffer, which
// is the only arrangement that works: numbers inside the text would be
// selectable, copyable and editable, and a buffer whose first four columns are
// not the user's is not the buffer.
func LineNumbers() BehaviorProp {
	return behaviorFunc(func(ctx *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["lineNumbers"] = true
	})
}

// TabSize sets how many spaces one indent is worth — what the Tab key inserts,
// and what EditIndent adds and EditOutdent removes.
//
// Zero means a literal tab character instead of spaces, which is what Go source
// wants. A negative size is clamped to zero rather than refused: the honest
// reading of "minus two spaces" is "no spaces", and a render pass is not a
// place to panic over an argument.
func TabSize(spaces int) BehaviorProp {
	if spaces < 0 {
		spaces = 0
	}
	return behaviorFunc(func(ctx *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["tabSize"] = spaces
	})
}

// CommentPrefix sets the line-comment marker EditCommentLine toggles — "//"
// for Go, "#" for shell and Python, "--" for SQL.
//
// An empty prefix makes EditCommentLine a no-op, which is the right answer for
// a language that has no line comments (JSON) rather than inserting a marker
// that would make the document invalid.
func CommentPrefix(prefix string) BehaviorProp {
	return behaviorFunc(func(ctx *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["commentPrefix"] = prefix
	})
}
