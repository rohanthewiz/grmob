# Package core — Editors

```go
import "github.com/rohanthewiz/grmob/core"
```

The code editor and the rich text editor, and the refs and commands that drive them.

One of 11 topic pages of [package core](core.md), which has the package overview and an index of every topic. This page documents the declarations in `core/codeeditor.go`, `core/editor.go`, `core/richtext.go`.

## Index

- [Constants](#constants) — `EditBold`, `EditCode`, `EditCommentLine`, `EditIndent`, `EditItalic`, `EditOutdent`, `EditRedo`, `EditSelectAll`, `EditStrike`, `EditUnderline`, `EditUndo`, `EditUnlink`
- [`func CodeEditor`](#func-codeeditor)
- [`func CommentPrefix`](#func-commentprefix)
- [`func EditBlock`](#func-editblock)
- [`func EditLink`](#func-editlink)
- [`func EditorTarget`](#func-editortarget)
- [`func LineNumbers`](#func-linenumbers)
- [`func OnRichSelectionChange`](#func-onrichselectionchange)
- [`func OnSelectionChange`](#func-onselectionchange)
- [`func Placeholder`](#func-placeholder)
- [`func ReadOnly`](#func-readonly)
- [`func RichTextEditor`](#func-richtexteditor)
- [`func RunEditorCommand`](#func-runeditorcommand)
- [`func TabSize`](#func-tabsize)
- [`type EditorRef`](#type-editorref)
    - [`func UseEditorRef`](#func-useeditorref)
- [`type RichSelection`](#type-richselection)
    - [`func (RichSelection) HasSelection`](#func-richselection-hasselection)

## Constants

The commands a core.CodeEditor understands. Spelled as constants so a toolbar and a renderer cannot disagree about a string literal, and so the census of what v1 supports is one list rather than four.

Each acts on the host's current selection, or on the line the caret is in when the selection is empty — which is what makes "indent" useful without a selection, and is the behavior every code editor has.

```go
const (
	// EditIndent inserts one indent at the start of every line the selection
	// touches. One indent is tabSize spaces, or a tab when tabSize is 0.
	EditIndent = "indent"
	// EditOutdent removes one indent's worth of leading white space from every
	// line the selection touches, and leaves a line that has none alone.
	EditOutdent = "outdent"
	// EditCommentLine toggles the commentPrefix on every line the selection
	// touches: it comments them all when any is uncommented, and uncomments
	// them when every one is already commented. The toggle is decided for the
	// whole run rather than per line, so a partially-commented block becomes
	// fully commented rather than inverting line by line.
	EditCommentLine = "commentLine"
	// EditSelectAll selects the whole buffer.
	EditSelectAll = "selectAll"
)
```

<small>[core/editor.go:191](https://github.com/rohanthewiz/grmob/blob/master/core/editor.go#L191)</small>

The commands a core.RichTextEditor understands. Each toggles on the current selection, or sets the typing attributes when the selection is empty — which is what makes "press bold, then type" work.

```go
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
```

<small>[core/richtext.go:94](https://github.com/rohanthewiz/grmob/blob/master/core/richtext.go#L94)</small>

## Functions

### func CodeEditor

```go
func CodeEditor(value string, onChange func(string), rows []GridRow, props ...PropsAndChildren) View
```

CodeEditor is an editable monospace buffer with syntax colour, a line-number gutter and the keyboard behaviour a programmer's editor has.

	core.CodeEditor(state.Src, onChange, highlight.Go().Rows(state.Src, highlight.Darcula),
	    core.LineNumbers(),
	    core.TabSize(4),
	    core.OnSelectionChange(func(start, end int) { ... }),
	    core.Height("240px"),
	)

comps.CodeEditor is the widget over it — it runs the highlighter, wires a toolbar and picks a scheme from the theme — and is what application code should reach for. This is the primitive it is built on.

#### Why a node type rather than a composition

The obvious pure-Go construction is a transparent core.TextArea in a ZStack over a core.TextGrid: Go colours the grid, the user types into the invisible field above it, and the two line up. They do not line up, and cannot: core.Style has no font-family, so the TextArea is drawn in the platform's proportional UI face while the grid is monospace, and no amount of styling from outside can pitch-match a SwiftUI TextEditor or a Compose BasicTextField to a separate text view. The overlay has to be built by something that owns \*both\* elements, which is the renderer. That is the whole argument for this being a node type, and it is the same argument TextGrid makes one step earlier.

#### The three rules every host implements

 1. Echo guard, unchanged from TextArea. \`value\` from Go is applied only when it is not an echo of the host's own typing. The buffer is the host's while focused and Go's otherwise. On the natives an echo is told from a rewrite by the text-edit stamps (core/text\_edit.go), and edits typed on text Go has since rewritten are replayed onto it; both renderers drive the same TextEditLedger their GrMobTextField does. The web runtime, which dispatches synchronously, keeps the value queue.

 2. Decoration is advisory and per line. The rows are GridRow children, exactly as core.TextGrid builds them, and a host applies row N's styling only if that row's concatenated text equals the host's current line N. A line that disagrees — Go is a keystroke behind, which it is for a few milliseconds after every keypress — is drawn in plain ink until the next patch. Never the other way round: decoration never rewrites the buffer, so a lexer that is wrong can make the screen ugly and can never make it lose text.

 3. Commands are epoch-stamped props. See core/editor.go.

#### The behaviour the contract tests pin

Monospace, no wrapping, horizontal scroll. Tab inserts an indent rather than moving focus. Enter copies the previous line's leading white space. Autocorrect, autocapitalization and smart quotes are off — every one of them corrupts source. A readOnly buffer is still selectable and still shows a caret, because a code block the user cannot copy out of is a screenshot.

#### Focus commands

core.Focus and core.DismissKeyboard reach an editor: CodeEditor is in focusableLeafTypes, so every command stamps it like any other text control and a background tap puts its keyboard away.

The one thing worth knowing is where the command lands. This node is a \*box\* — a scroll container holding a gutter and a buffer — where an Input is the control itself, so each renderer resolves the stamp to the buffer rather than applying it where it arrived. The gutter is chrome and never takes the caret. htmlout applies it nowhere at all: its editor is a read-only snapshot with no editable element to autofocus.

#### Known gaps in v1

Nothing that needs a caret is exercised by any harness here — the IME composing region, a hardware Tab on iPad, and paste from another app are arguments rather than tests.

#### value and rows are two facts about one buffer, and they can disagree

value is the text; rows are a \*decoration of that text\*, computed in Go from that same text. They always agree at the moment Go builds them, and are allowed to disagree with the host mid-keystroke, which is what rule 2 is about. A caller that computes rows from something other than value has not broken anything — the rows simply never match and the buffer is drawn plain.

<small>[core/codeeditor.go:85](https://github.com/rohanthewiz/grmob/blob/master/core/codeeditor.go#L85)</small>

### func CommentPrefix

```go
func CommentPrefix(prefix string) BehaviorProp
```

CommentPrefix sets the line-comment marker EditCommentLine toggles — "//" for Go, "#" for shell and Python, "--" for SQL.

An empty prefix makes EditCommentLine a no-op, which is the right answer for a language that has no line comments (JSON) rather than inserting a marker that would make the document invalid.

<small>[core/codeeditor.go:170](https://github.com/rohanthewiz/grmob/blob/master/core/codeeditor.go#L170)</small>

### func EditBlock

```go
func EditBlock(kind richtext.BlockKind) string
```

EditBlock is the command that makes every block the selection touches the given kind.

The kind's wire value \*is\* richtext.BlockKind's, which is why that type is a string: a toolbar naming a heading and a document holding one use the same token, so there is no second table to keep in step.

<small>[core/richtext.go:135](https://github.com/rohanthewiz/grmob/blob/master/core/richtext.go#L135)</small>

### func EditLink

```go
func EditLink(url string) string
```

EditLink is the command that makes the selection a link to url.

A function rather than a constant because the command carries an argument, and the argument rides in the string — the command channel is one prop and widening it to a map would change the shape all four hosts read for the sake of one command. "link:" is the prefix; everything after the first colon is the URL, so a URL containing colons (every one of them does) is intact.

An empty url is EditUnlink's job and is refused here rather than sent as a link to nowhere.

<small>[core/richtext.go:122](https://github.com/rohanthewiz/grmob/blob/master/core/richtext.go#L122)</small>

### func EditorTarget

```go
func EditorTarget(ref *EditorRef) BehaviorProp
```

EditorTarget marks the editing surface it is applied to as ref's.

A nil ref returns a nil prop rather than panicking — leafNode skips a nil item — so \`core.EditorTarget(maybeRef)\` degrades to an editor no toolbar can command instead of crashing a render pass.

Applying it to anything but a CodeEditor or a RichTextEditor is harmless and pointless: the stamp lands and no renderer reads it.

<small>[core/editor.go:115](https://github.com/rohanthewiz/grmob/blob/master/core/editor.go#L115)</small>

### func LineNumbers

```go
func LineNumbers() BehaviorProp
```

LineNumbers turns on the gutter.

The gutter is drawn by the host rather than being part of the buffer, which is the only arrangement that works: numbers inside the text would be selectable, copyable and editable, and a buffer whose first four columns are not the user's is not the buffer.

<small>[core/codeeditor.go:136](https://github.com/rohanthewiz/grmob/blob/master/core/codeeditor.go#L136)</small>

### func OnRichSelectionChange

```go
func OnRichSelectionChange(handler func(RichSelection)) BehaviorProp
```

OnRichSelectionChange reports a rich-text editor's caret and the formatting active at it.

A separate builder from OnSelectionChange rather than an overload, because the two carry different things and the difference is the point: a code editor's selection is two offsets, and a rich editor's is the state a toolbar has to draw. They share the prop name on the wire ("onSelectionChange") and the text channel; what differs is the payload each node type sends and the parse core does before app code sees it.

<small>[core/richtext.go:243](https://github.com/rohanthewiz/grmob/blob/master/core/richtext.go#L243)</small>

### func OnSelectionChange

```go
func OnSelectionChange(handler func(start, end int)) BehaviorProp
```

OnSelectionChange reports the editing surface's selection as it moves.

The handler receives byte offsets into the UTF-8 value, half-open as every range in Go is: start == end is a caret with nothing selected.

	core.CodeEditor(src, onChange, rows,
	    core.OnSelectionChange(func(start, end int) { status.Set(start, end) }),
	)

It rides the text channel carrying "start:end" rather than needing a fifth bridge channel; see this file's doc. A payload that is not two integers is dropped rather than delivered as a zero range.

Applied to a node that is not an editing surface it is inert, like every other behavior prop on a node type that does not read it.

<small>[core/editor.go:253](https://github.com/rohanthewiz/grmob/blob/master/core/editor.go#L253)</small>

### func Placeholder

```go
func Placeholder(text string) BehaviorProp
```

Placeholder is the prompt an empty editing surface shows.

core.Input and core.TextArea take theirs positionally, because they had one before the mixed argument list existed; the editors take it as a prop, which is the shape every option added since uses.

<small>[core/editor.go:292](https://github.com/rohanthewiz/grmob/blob/master/core/editor.go#L292)</small>

### func ReadOnly

```go
func ReadOnly() BehaviorProp
```

ReadOnly makes an editing surface show a caret and allow selection while refusing every edit.

It is deliberately not core.Disabled. A disabled control is inert and greyed and is skipped by assistive technology's traversal; a read-only code block is \*content\* — the user is meant to read it, select it and copy out of it — and on every platform that is a different state with a different look. The two are not interchangeable and a widget that wants the greyed-out reading can still ask for it.

<small>[core/editor.go:278](https://github.com/rohanthewiz/grmob/blob/master/core/editor.go#L278)</small>

### func RichTextEditor

```go
func RichTextEditor(doc richtext.Doc, onChange func(richtext.Doc), props ...PropsAndChildren) View
```

RichTextEditor is an editable formatted document: bold, italics, headings, lists, quotes, links.

	core.RichTextEditor(state.Doc, func(d richtext.Doc) { state.Doc = d },
	    core.Placeholder("Write something…"),
	    core.EditorTarget(ref),
	    core.OnRichSelectionChange(func(sel core.RichSelection) { bar.Set(sel) }),
	)

comps.RichTextEditor is the widget over it — it builds the toolbar and wires the link prompt — and is what application code should reach for.

#### The value is a document, and the document is Go's

richtext.Doc crosses the wire as JSON and each host maps it to and from its own text representation. That is the whole architectural decision here and the package doc for richtext carries the argument: an NSAttributedString, a Spannable and a contenteditable's innerHTML are three vocabularies, and an app that stored whichever one the user happened to type on would have a database its other two targets could not read.

#### The rules it shares with CodeEditor, and the one it does not

The echo guard is the text fields', so Go's echo of the host's own last onChange never resets the caret. On the natives it runs on the text-edit stamps (core/text\_edit.go), with two differences that both come from the value being a JSON document rather than text. Go compares the host's JSON with its own render as documents, not bytes, because neither host's JSON library spells a document the way encoding/json does. And a rewrite is adopted as it stands: a JSON string has no "typing at either end" to replay onto Go's document, so what is in flight is lost, as it always was here. Commands are epoch-stamped props, same mechanism, same adopt-on-first-sight rule (see core/editor.go).

The stale-line rule is \*not\* here and does not need to be. A CodeEditor has two facts about one buffer — the text and a decoration of it computed separately — which can disagree for a frame. Here the doc \*is\* the styled buffer: there is nothing to compare it against, because the formatting and the characters arrive together.

#### Known gaps in v1

core.Focus and core.DismissKeyboard reach an editor — the type is in focusableLeafTypes, exactly as CodeEditor is. On both phones the control is a classic text view hosted inside the declarative framework (a UITextView, an EditText), so neither renderer can hand the command to its platform's own focus system and each drives the responder directly; see core/focus.go.

Collaborative editing, images, tables and per-run fonts are non-goals; each is a driver away and none changes the design above.

<small>[core/richtext.go:60](https://github.com/rohanthewiz/grmob/blob/master/core/richtext.go#L60)</small>

### func RunEditorCommand

```go
func RunEditorCommand(ref *EditorRef, command string)
```

RunEditorCommand sends one command to ref's editor, to be applied to whatever the host's own selection is at the time it lands.

Called from an event handler, as a toolbar button's whole body:

	core.Button("Bold", func() { core.RunEditorCommand(ref, core.EditBold) })

The command strings each editor understands are its own; see the Edit\* constants below for the census. A command an editor does not know is deliberately a no-op on every host rather than an error — the alternative is a screen that crashes because a toolbar outgrew its editor.

Issuing a command for a ref whose editor is not currently in the tree does nothing visible: no node stamps it, so no host acts. The command still consumes an epoch, which is correct — it happened, it simply had no target on screen.

A nil ref is a no-op rather than a panic, matching EditorTarget.

<small>[core/editor.go:168](https://github.com/rohanthewiz/grmob/blob/master/core/editor.go#L168)</small>

### func TabSize

```go
func TabSize(spaces int) BehaviorProp
```

TabSize sets how many spaces one indent is worth — what the Tab key inserts, and what EditIndent adds and EditOutdent removes.

Zero means a literal tab character instead of spaces, which is what Go source wants. A negative size is clamped to zero rather than refused: the honest reading of "minus two spaces" is "no spaces", and a render pass is not a place to panic over an argument.

<small>[core/codeeditor.go:152](https://github.com/rohanthewiz/grmob/blob/master/core/codeeditor.go#L152)</small>

## Types

### type EditorRef

```go
type EditorRef struct {
	// contains filtered or unexported fields
}
```

EditorRef names one editing surface so that a toolbar can send it commands.

It is the editor half of FocusRef and is used the same way: a hook makes one that is stable across render passes, EditorTarget puts it on the editor, and RunEditorCommand sends to it.

	ref := core.UseEditorRef(ctx)

	core.CodeEditor(src, onChange, rows, core.EditorTarget(ref))
	core.Button("Indent", func() { core.RunEditorCommand(ref, core.EditIndent) })

Unlike FocusRef it carries its own command state rather than pointing at the app's — see the file doc for why one editor's commands are nobody else's.

<small>[core/editor.go:76](https://github.com/rohanthewiz/grmob/blob/master/core/editor.go#L76)</small>

#### func UseEditorRef

```go
func UseEditorRef(ctx *Context) *EditorRef
```

UseEditorRef returns an EditorRef that is stable for the lifetime of this hook slot, which is what makes the ref usable as an identity.

A hook rather than a bare constructor for exactly FocusRef's reason: a ref built inline in a render function is a new pointer every pass, so EditorTarget would stamp one identity and the toolbar's handler would bump another — and the command would silently never reach a node. NewState both pins the pointer and reserves the cursor slot properly.

A widget that calls this consumes a positional hook slot on the caller's context and must therefore be rendered unconditionally, like any other hook user; comps.CodeEditor says so in its own doc.

<small>[core/editor.go:98](https://github.com/rohanthewiz/grmob/blob/master/core/editor.go#L98)</small>

### type RichSelection

```go
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
```

RichSelection is what a rich-text editor reports about its caret: where it is, and what formatting is active there.

The marks matter more than the offsets, and that is the reason this is a struct rather than the two ints a CodeEditor reports. A toolbar has to show its bold button as \*on\* when the caret is inside bold text, and nothing in Go can work that out — the document is Go's, but where the caret is inside it is the host's.

Start and End are byte offsets into the document's plain text (richtext.Doc.PlainText), which is the one coordinate system all four hosts can produce and which is stable across the marks. They are there for a status line and for "is anything selected"; a command never needs them, because every command acts on the host's own selection.

<small>[core/richtext.go:153](https://github.com/rohanthewiz/grmob/blob/master/core/richtext.go#L153)</small>

#### func (RichSelection) HasSelection

```go
func (s RichSelection) HasSelection() bool
```

HasSelection reports whether anything is actually selected, as opposed to a bare caret. The distinction is what a "Link" button needs: linking an empty selection has nothing to attach to.

<small>[core/richtext.go:175](https://github.com/rohanthewiz/grmob/blob/master/core/richtext.go#L175)</small>

