package comps

import (
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/highlight"
)

// CodeEditor is a programmer's editor: a monospace buffer with syntax colour,
// an optional line-number gutter, and an optional toolbar of the editing
// commands a code surface needs.
//
//	comps.CodeEditor{
//	    Value:       src.Get(),
//	    OnChange:    src.Set,
//	    Language:    "go",
//	    LineNumbers: true,
//	    Height:      "240px",
//	}
//
//	┌ Column ──────────────────────────────────────────────────────┐
//	│ ┌ Row (the toolbar, only when Toolbar names a ref) ────────┐ │
//	│ │ [ ⇥ ]  [ ⇤ ]  [ // ]                                     │ │
//	│ └──────────────────────────────────────────────────────────┘ │
//	│ ┌ core.CodeEditor ─────────────────────────────────────────┐ │
//	│ │  1  func main() {                                        │ │
//	│ │  2      println("hi")                                    │ │
//	│ │  3  }                                                    │ │
//	│ └──────────────────────────────────────────────────────────┘ │
//	└──────────────────────────────────────────────────────────────┘
//
// It is the widget over core.CodeEditor and is what application code should
// reach for: it runs the highlighter, picks a colour scheme that suits the
// theme, and builds the toolbar. The node underneath is the primitive, and its
// doc carries the three rules every host implements.
//
// # It holds no state and calls no hook
//
// Value is the caller's, exactly as SearchField's is, and every keystroke
// arrives through OnChange.
//
// The hook question is worth spelling out, because the obvious design fails it.
// A toolbar needs a core.EditorRef, a ref must be stable across passes, and the
// way to get one is core.UseEditorRef — a hook. Calling it in here would make
// every CodeEditor a hook-slot consumer, and therefore something that must be
// rendered unconditionally on every pass, like Accordion and DatePicker. That
// is a fine obligation for a date field and a bad one for a *code block*: a
// read-only editor with no toolbar is the thing a document renders inside an
// `if`, inside a loop, inside a lesson body. So the ref is the caller's, and it
// is the Toolbar field itself — a toolbar is exactly "a ref plus some buttons",
// and naming the ref is how you ask for one:
//
//	ref := core.UseEditorRef(ctx)
//	comps.CodeEditor{Value: v, OnChange: set, Toolbar: ref}
//
// An editor with no toolbar touches no hook and can be rendered anywhere.
//
// # The highlighter runs every pass, deliberately
//
// No memoization. go/scanner over a thousand lines is well under a millisecond
// — the tutorial has re-lexed every snippet on every render pass since it had
// snippets — and the alternative is hooks.UseMemo, which is the hook obligation
// this widget has just been designed out of. If a buffer ever grows past the
// point where that is true, the answer is for the *caller* to memoize and pass
// a Highlighter that caches, not for this widget to start consuming slots.
type CodeEditor struct {
	// Value is the buffer's text. The editor is controlled: it renders what it
	// is given and reports edits through OnChange.
	Value string

	// OnChange receives every edit. A nil OnChange makes the editor read-only
	// in practice — it will render Value and drop keystrokes — so set ReadOnly
	// instead when that is what you mean, which also tells the platform.
	OnChange func(string)

	// Language picks the lexer by name: "go", "json", or "" for no colour. An
	// unknown name is uncoloured rather than an error, so a screen whose editor
	// mis-spells its language still renders. It also picks the comment marker
	// the comment command toggles; see CommentPrefix.
	Language string

	// Highlighter overrides Language with a lexer of your own — anything
	// satisfying highlight.Highlighter, including one that caches.
	Highlighter highlight.Highlighter

	// Scheme is the palette. The zero value picks highlight.Darcula or
	// highlight.Light from the theme's own background, so an editor in a light
	// app is not a dark rectangle in the middle of the screen and one in a dark
	// app is not a white page.
	Scheme highlight.Scheme

	// LineNumbers turns on the gutter. Off by default: a three-line snippet
	// with line numbers reads as a listing rather than as code.
	LineNumbers bool

	// ReadOnly shows a caret and allows selection while refusing edits. This is
	// the display half of the widget — a code block in a document, a payload in
	// a log viewer — and it is deliberately not Disabled: the reader is meant
	// to select the code and copy it.
	ReadOnly bool

	// TabSize is how many spaces one indent is worth. Zero is core's default of
	// four; use TabSize(0) on the node directly for a literal tab.
	TabSize int

	// CommentPrefix overrides the line-comment marker the toolbar's comment
	// button toggles. Empty takes it from Language.
	CommentPrefix string

	// Height fixes the editor's height, so a long buffer scrolls inside it
	// rather than growing the screen. Empty lets it size to its content.
	Height string

	// Toolbar, when set, adds a row of editing commands above the editor and is
	// the ref they are sent to. Build it with core.UseEditorRef(ctx) in the
	// calling component — see the type's doc for why the ref is yours and not
	// this widget's.
	Toolbar *core.EditorRef

	// OnSelectionChange reports the caret as byte offsets into Value. Useful
	// for a status line ("line 12, column 4") and for a toolbar that has to
	// know whether anything is selected.
	OnSelectionChange func(start, end int)

	// Style is applied to the editor after the widget's own surface, so the
	// fill, the ink, the radius and the padding are all overridable.
	Style []core.StyleProp
}

func (c CodeEditor) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	scheme := c.scheme(t)

	// The rows are the *decoration of Value*, computed from the same string the
	// node is given on the same pass, which is what makes the hosts' stale-line
	// rule a statement about the round trip rather than about this widget.
	rows := c.highlighter().Rows(c.Value, scheme)

	editor := core.CodeEditor(c.Value, c.onChange(), rows, c.editorProps(t, scheme)...)
	if c.Toolbar == nil {
		return editor.Render(ctx)
	}
	return core.Column(
		core.Gap(float64(t.Spacing.XS)),
		c.toolbar(t),
		editor,
	).Render(ctx)
}

// editorProps assembles the node's argument list: the surface the scheme
// describes, the options, and the caller's style last so any of it can be
// overridden.
func (c CodeEditor) editorProps(t *core.Theme, scheme highlight.Scheme) []core.PropsAndChildren {
	items := make([]core.PropsAndChildren, 0, len(c.Style)+12)
	items = append(items,
		// The surface and its default ink come from the scheme rather than from
		// theme roles, and that is the same call the tutorial's code block has
		// always made: an editor-dark (or editor-light) surface reads as "this
		// is code" whatever the app around it looks like, and no palette role
		// means "the background of a code listing".
		core.BackgroundColor(scheme.Bg),
		core.TextColor(scheme.Ink),
		// A code pitch, not a form-field pitch. The theme's TextArea base — the
		// node's own base style — is 17pt, which is right for prose in a field
		// and puts about half as much line on a phone as code needs.
		core.FontSize(13),
		core.Padding(t.Spacing.SM),
		core.BorderRadius(t.Components.Input.BorderRadius),
		// The theme's field frame would draw a box around a surface that is
		// already its own box, and in a colour chosen for a form control rather
		// than for this one.
		core.BorderWidth(0),
	)
	if c.LineNumbers {
		items = append(items, core.LineNumbers())
	}
	if c.ReadOnly {
		items = append(items, core.ReadOnly())
	}
	if c.TabSize > 0 {
		items = append(items, core.TabSize(c.TabSize))
	}
	if prefix, ok := c.commentPrefix(); ok {
		items = append(items, core.CommentPrefix(prefix))
	}
	if c.Height != "" {
		items = append(items, core.Height(c.Height))
	}
	if c.Toolbar != nil {
		items = append(items, core.EditorTarget(c.Toolbar))
	}
	if c.OnSelectionChange != nil {
		items = append(items, core.OnSelectionChange(c.OnSelectionChange))
	}
	for _, sp := range c.Style {
		items = append(items, sp)
	}
	return items
}

// toolbar is the command row: three ghost buttons, in the order a keyboard
// would reach them.
//
// Ghost rather than filled, because the buttons sit above a code surface whose
// colours are a scheme's and not the theme's — a filled Primary button there is
// the loudest thing on the screen, next to the quietest.
//
// The comment button is dropped rather than disabled when the language has no
// line comment (JSON): a permanently inert control is a question the reader has
// to answer every time they see it, and the command would do nothing anyway.
func (c CodeEditor) toolbar(t *core.Theme) core.View {
	glyph := func(label, hint string, command string) core.View {
		return Button{
			Label:    label,
			Emphasis: EmphasisGhost,
			// Captured by value: the ref is the widget's field and the command
			// is a constant, so the closure outlives this call safely.
			OnTap:              func() { core.RunEditorCommand(c.Toolbar, command) },
			AccessibilityLabel: hint,
			Style: []core.StyleProp{
				core.PaddingHorizontal(t.Spacing.SM),
				core.PaddingVertical(t.Spacing.XS),
			},
		}
	}

	items := []core.PropsAndChildren{
		core.Gap(float64(t.Spacing.XS)),
		core.AlignItemsProp(core.AlignItemsCenter),
		glyph("⇥", "Indent", core.EditIndent),
		glyph("⇤", "Outdent", core.EditOutdent),
	}
	if prefix, ok := c.commentPrefix(); !ok || prefix != "" {
		label := prefix
		if label == "" {
			label = "//" // core's own default, which is what the node will use
		}
		items = append(items, glyph(label, "Toggle comment", core.EditCommentLine))
	}
	return core.Row(items...)
}

// highlighter resolves the two ways a lexer can be named. An explicit
// Highlighter wins, because it is the more specific statement.
func (c CodeEditor) highlighter() highlight.Highlighter {
	if c.Highlighter != nil {
		return c.Highlighter
	}
	return highlight.ForLanguage(c.Language)
}

// scheme resolves the palette, deriving one from the theme when the caller
// named none.
//
// The test is the theme's *background*, not its Surface: the question is
// whether the app reads light or dark, and Background is the page the editor
// will be seen against. The 0.5 threshold is the midpoint of WCAG's relative
// luminance, which is the same measure everything else in this package judges a
// backdrop by.
//
// An unparseable background falls to Darcula, which is also what a theme with
// no background at all should get: a code surface has to be *some* deliberate
// colour, and the dark one is the one every editor defaults to.
func (c CodeEditor) scheme(t *core.Theme) highlight.Scheme {
	if c.Scheme != (highlight.Scheme{}) {
		return c.Scheme
	}
	if lum, ok := relativeLuminance(t.Colors.Background); ok && lum > 0.5 {
		return highlight.Light
	}
	return highlight.Darcula
}

// commentPrefix answers what the comment command should toggle, and whether
// this widget has an opinion at all.
//
// ok=false means "leave the node's own default alone", which matters because
// the empty string is a real answer — a language with no line comment — and not
// the same as having nothing to say. core.CodeEditor defaults the prop to "//",
// which is right for most of what a code editor in an app shows, so an editor
// with no Language named keeps it.
func (c CodeEditor) commentPrefix() (string, bool) {
	if c.CommentPrefix != "" {
		return c.CommentPrefix, true
	}
	switch c.Language {
	case "go":
		return "//", true
	case "json":
		// JSON has no comment. An empty prefix makes the command a no-op rather
		// than one that inserts a marker making the document invalid.
		return "", true
	}
	return "", false
}

// onChange is the handler the node is given, never nil: core.CodeEditor
// registers whatever it is handed and the dispatcher invokes it unguarded, so a
// nil here would panic on the first keystroke rather than ignoring it. Same
// guard SearchField and Button apply.
func (c CodeEditor) onChange() func(string) {
	if c.OnChange != nil {
		return c.OnChange
	}
	return func(string) {}
}
