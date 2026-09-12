package components

import (
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/richtext"
)

// RichTextEditor is a formatted-text editor: bold, italics, headings, lists,
// quotes, links — with a toolbar whose buttons show what is active under the
// caret.
//
//	bar := components.UseRichToolbar(ctx)
//
//	components.RichTextEditor{
//	    Doc:         note.Get(),
//	    OnChange:    note.Set,
//	    Placeholder: "Write something…",
//	    Toolbar:     bar,
//	    MinHeight:   "160px",
//	}
//
//	┌ Column ──────────────────────────────────────────────────────┐
//	│ ┌ Row, wrapping (the toolbar) ─────────────────────────────┐ │
//	│ │ [B] [I] [U] [S] [</>] [H1] [H2] [•] [1.] [""] [🔗]       │ │
//	│ └──────────────────────────────────────────────────────────┘ │
//	│ ┌ core.RichTextEditor ─────────────────────────────────────┐ │
//	│ │  The document, edited in place by the platform's own     │ │
//	│ │  text engine.                                            │ │
//	│ └──────────────────────────────────────────────────────────┘ │
//	└──────────────────────────────────────────────────────────────┘
//
// # A read-only editor with no toolbar is the display half
//
// A comment, a note, a description — anything that shows formatted text the
// reader did not write. There is no second "RichTextView" node, because there
// does not need to be one: the renderer's own text engine draws the document
// either way, and `ReadOnly` is the difference between reading it and writing
// it. That is also why this widget takes no hook of its own; see below.
//
// # The toolbar is the caller's, and that is what keeps the display half free
//
// A toolbar needs three things that must survive a render pass: a
// core.EditorRef to send commands to, the last reported selection (so the bold
// button can look pressed), and whether the link prompt is open. All three are
// hooks, and a widget that called them would make *every* RichTextEditor a
// hook-slot consumer — something that must be rendered unconditionally on every
// pass, like Accordion and DatePicker.
//
// That is a fine obligation for an editor with a toolbar and a bad one for a
// note being *displayed*, which is the thing rendered inside an `if`, inside a
// loop, inside a list of comments. So UseRichToolbar is the hook, the caller
// makes it, and an editor with no toolbar touches nothing.
// components.CodeEditor makes the same split for the same reason.
type RichTextEditor struct {
	// Doc is the document. The editor is controlled: it renders what it is
	// given and reports edits through OnChange.
	Doc richtext.Doc

	// OnChange receives every edit as a whole document. A nil OnChange makes
	// the editor read-only in practice; set ReadOnly instead when that is what
	// you mean, which also tells the platform.
	OnChange func(richtext.Doc)

	// Placeholder is the prompt an empty editor shows.
	Placeholder string

	// ReadOnly shows the document with a caret and a selection and refuses
	// edits. Deliberately not Disabled: a note being displayed is content the
	// reader is meant to select and copy.
	ReadOnly bool

	// Toolbar, when set, adds the command row above the editor. Build it with
	// components.UseRichToolbar(ctx) in the calling component.
	Toolbar *RichToolbar

	// MinHeight gives an empty editor something to be. Without it a document
	// with one line in it is one line tall, which reads as a text field rather
	// than as a place to write.
	MinHeight string

	// Height fixes the editor's height instead, so a long document scrolls
	// inside it rather than growing the screen.
	Height string

	// Style is applied to the editor after the widget's own frame.
	Style []core.StyleProp
}

// RichToolItem is one button on the toolbar: what it says, and what it sends.
//
// Command is a core Edit* constant or one of the two builders (core.EditBlock,
// core.EditLink) — except for RichToolLink, which is this package's own
// sentinel for "open the link prompt", because a URL has to be typed before
// there is a command to send.
type RichToolItem struct {
	Label   string
	Command string
	// AccessibilityLabel names the button for a screen reader. A toolbar is
	// glyphs — "B", "H1", "🔗" — and a glyph is not a name.
	AccessibilityLabel string
}

// RichToolLink is the sentinel Command that opens the link prompt. Not a core
// command: core.EditLink needs a URL, and the prompt is where one comes from.
const RichToolLink = "components:link"

// RichToolbarDefault is the toolbar most notes want: the five marks, three
// block kinds, the two lists, a quote, and the link prompt.
//
// A var rather than a func so a caller can take a slice of it, append to it, or
// reorder it — which is the whole reason the toolbar is a list of items rather
// than a bool. It is package state, so treat it as read-only; UseRichToolbar
// copies it.
var RichToolbarDefault = []RichToolItem{
	{Label: "B", Command: core.EditBold, AccessibilityLabel: "Bold"},
	{Label: "I", Command: core.EditItalic, AccessibilityLabel: "Italic"},
	{Label: "U", Command: core.EditUnderline, AccessibilityLabel: "Underline"},
	{Label: "S", Command: core.EditStrike, AccessibilityLabel: "Strikethrough"},
	{Label: "</>", Command: core.EditCode, AccessibilityLabel: "Inline code"},
	{Label: "H1", Command: core.EditBlock(richtext.Heading1), AccessibilityLabel: "Heading 1"},
	{Label: "H2", Command: core.EditBlock(richtext.Heading2), AccessibilityLabel: "Heading 2"},
	{Label: "¶", Command: core.EditBlock(richtext.Paragraph), AccessibilityLabel: "Paragraph"},
	{Label: "•", Command: core.EditBlock(richtext.Bullet), AccessibilityLabel: "Bulleted list"},
	{Label: "1.", Command: core.EditBlock(richtext.Numbered), AccessibilityLabel: "Numbered list"},
	{Label: "❝", Command: core.EditBlock(richtext.Quote), AccessibilityLabel: "Quote"},
	{Label: "🔗", Command: RichToolLink, AccessibilityLabel: "Add link"},
}

// RichToolbar is everything a toolbar needs that has to survive a render pass.
//
// Built by UseRichToolbar, which is a hook: the ref must be the same pointer
// every pass or the buttons would command an editor nobody is listening to,
// and the selection has to be remembered between the report arriving and the
// next render drawing the buttons from it.
type RichToolbar struct {
	// Items is what the toolbar offers, in order. Defaults to a copy of
	// RichToolbarDefault; assign to it to offer something else.
	Items []RichToolItem

	ref      *core.EditorRef
	selected core.State[core.RichSelection]
	linkOpen core.State[bool]
	linkURL  core.State[string]
}

// UseRichToolbar builds the state a RichTextEditor's toolbar needs.
//
// Four hook slots, in a fixed order, so — like any hook user — it must be called
// unconditionally on every pass of the component that owns it.
//
//	func NoteScreen(ctx *core.Context) core.View {
//	    note := core.NewState(ctx, richtext.Doc{})
//	    bar  := components.UseRichToolbar(ctx)
//	    return components.RichTextEditor{Doc: note.Get(), OnChange: note.Set, Toolbar: bar}
//	}
func UseRichToolbar(ctx *core.Context) *RichToolbar {
	// The order is the contract: four slots, always the same four, always in
	// this sequence.
	ref := core.UseEditorRef(ctx)
	selected := core.NewState(ctx, core.RichSelection{})
	linkOpen := core.NewState(ctx, false)
	linkURL := core.NewState(ctx, "")

	// A copy, so a caller who appends to their own Items never mutates the
	// package's default list out from under every other screen.
	items := make([]RichToolItem, len(RichToolbarDefault))
	copy(items, RichToolbarDefault)

	return &RichToolbar{
		Items: items, ref: ref,
		selected: selected, linkOpen: linkOpen, linkURL: linkURL,
	}
}

// Selection is the last selection the editor reported, which is what the
// toolbar draws its pressed state from — and is worth reading directly for a
// status line, or to decide whether a "Link" action makes sense.
func (b *RichToolbar) Selection() core.RichSelection { return b.selected.Get() }

func (r RichTextEditor) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	editor := core.RichTextEditor(r.Doc, r.OnChange, r.editorProps(t)...)
	if r.Toolbar == nil {
		return editor.Render(ctx)
	}
	return core.Column(
		core.Gap(float64(t.Spacing.XS)),
		r.toolbar(ctx, t),
		editor,
		// The link prompt, a sibling of the editor rather than a child of the
		// toolbar: an overlay nested inside a bordered row inherits that row's
		// clip on any target that honours overflow. DatePicker's sheet is
		// placed the same way for the same reason.
		r.linkPrompt(ctx, t),
	).Render(ctx)
}

func (r RichTextEditor) editorProps(t *core.Theme) []core.PropsAndChildren {
	items := make([]core.PropsAndChildren, 0, len(r.Style)+8)
	items = append(items,
		core.BackgroundColor(t.Colors.Surface),
		core.BorderRadius(t.Components.Input.BorderRadius),
		core.Padding(t.Spacing.SM),
	)
	if r.Placeholder != "" {
		items = append(items, core.Placeholder(r.Placeholder))
	}
	if r.ReadOnly {
		items = append(items, core.ReadOnly())
	}
	if r.MinHeight != "" {
		items = append(items, core.MinHeight(r.MinHeight))
	}
	if r.Height != "" {
		items = append(items, core.Height(r.Height))
	}
	if bar := r.Toolbar; bar != nil {
		items = append(items,
			core.EditorTarget(bar.ref),
			// The toolbar's pressed state is this report and nothing else: Go
			// owns the document but not the caret, so which button should look
			// pressed is a question only the host can answer.
			core.OnRichSelectionChange(bar.selected.Set),
		)
	}
	for _, sp := range r.Style {
		items = append(items, sp)
	}
	return items
}

// toolbar is the command row: a wrapping strip of small toggle buttons, in the
// ChipStrip idiom.
//
// Wrapping matters. Twelve buttons do not fit across a phone, and a row that
// overflows would put the link button off the edge of the screen with no way to
// reach it — so the row wraps onto as many lines as it needs, exactly as
// ChipStrip does with a filter bar.
//
// # Why the strip carries no core.RoleToolbar
//
// It would be the right role and this package may not declare it. RoleToolbar
// is one of core's keyboard composites, and a widget that declares a composite
// container role makes a *nested* composite reachable by ordinary composition —
// two of these on a screen, or one inside a caller's own listbox — which is a
// finding core.AuditTree exists to report and which no screen should be able to
// produce by accident. TestNoWidgetDeclaresACompositeContainerRole in this
// package is the rule, with the argument written out.
//
// A caller who wants the landmark declares it on their own box, which is what
// makes the pairing deliberate:
//
//	core.Box(core.AccessibilityRole(core.RoleToolbar), editor)
func (r RichTextEditor) toolbar(ctx *core.Context, t *core.Theme) core.View {
	bar := r.Toolbar
	selection := bar.selected.Get()

	items := make([]core.PropsAndChildren, 0, len(bar.Items)+4)
	items = append(items,
		core.FlexWrap(true),
		core.Gap(float64(t.Spacing.XS)),
		core.AlignItemsProp(core.AlignItemsCenter),
	)
	for _, item := range bar.Items {
		items = append(items, r.toolButton(item, selection, t))
	}
	return core.Row(items...)
}

// toolButton is one command. Ghost when inactive, filled when the caret is
// inside what it applies — which is the same two-state treatment Chip gives a
// filter, and for the same reason: the button is not a command so much as a
// *state* the writer is toggling.
func (r RichTextEditor) toolButton(item RichToolItem, sel core.RichSelection, t *core.Theme) core.View {
	bar := r.Toolbar
	active := richToolActive(item.Command, sel)

	emphasis := EmphasisGhost
	if active {
		emphasis = EmphasisOutlined
	}
	return Button{
		Label:              item.Label,
		Emphasis:           emphasis,
		Disabled:           r.ReadOnly,
		AccessibilityLabel: item.AccessibilityLabel,
		OnTap: func() {
			if item.Command == RichToolLink {
				// The prompt is pre-filled with the link already under the
				// caret, so "edit this link" is the same button as "add one".
				bar.linkURL.Set(sel.Link)
				bar.linkOpen.Set(true)
				return
			}
			core.RunEditorCommand(bar.ref, item.Command)
		},
		Style: []core.StyleProp{
			core.PaddingHorizontal(t.Spacing.SM),
			core.PaddingVertical(t.Spacing.XS),
		},
	}
}

// richToolActive decides whether a command's button should read as pressed.
//
// The three cases are the three shapes a command has, and they are told apart
// by their prefix rather than by a table — which is what lets a caller add
// `core.EditBlock(richtext.Heading3)` to their own Items and have it light up
// without touching this function.
func richToolActive(command string, sel core.RichSelection) bool {
	switch command {
	case core.EditBold:
		return sel.Bold
	case core.EditItalic:
		return sel.Italic
	case core.EditUnderline:
		return sel.Underline
	case core.EditStrike:
		return sel.Strike
	case core.EditCode:
		return sel.Code
	case RichToolLink:
		return sel.Link != ""
	}
	if kind, ok := richToolBlockKind(command); ok {
		// A block command is active when the caret is already in a block of
		// that kind. The empty Block is "the host had nothing to say", which is
		// not the same as a paragraph — so a document with no blocks yet lights
		// nothing up rather than lighting up the paragraph button.
		return sel.Block != "" && sel.Block == kind
	}
	return false
}

func richToolBlockKind(command string) (richtext.BlockKind, bool) {
	const prefix = "block:"
	if len(command) <= len(prefix) || command[:len(prefix)] != prefix {
		return "", false
	}
	return richtext.BlockKind(command[len(prefix):]), true
}

// linkPrompt is the modal the link button opens: one field and two actions.
//
// A modal rather than an inline field because the toolbar wraps — an input
// appearing inside it would reflow the whole strip and move every other button
// out from under the finger that just tapped one.
//
// "Remove" is offered whenever the caret is already in a link, so the prompt is
// the one place a link is managed rather than two.
func (r RichTextEditor) linkPrompt(ctx *core.Context, t *core.Theme) core.View {
	bar := r.Toolbar
	close := func() { bar.linkOpen.Set(false) }
	linked := bar.selected.Get().Link != ""

	actions := []core.PropsAndChildren{
		core.Gap(float64(t.Spacing.SM)),
		core.Justify(core.JustifyEnd),
	}
	if linked {
		actions = append(actions, Button{
			Label:    "Remove",
			Emphasis: EmphasisGhost,
			Variant:  VariantError,
			OnTap: func() {
				close()
				core.RunEditorCommand(bar.ref, core.EditUnlink)
			},
		})
	}
	actions = append(actions,
		Button{Label: "Cancel", Emphasis: EmphasisGhost, OnTap: close},
		Button{
			Label: "Link",
			OnTap: func() {
				url := bar.linkURL.Get()
				close()
				// An empty URL is core.EditLink's own unlink case, so "Link"
				// with nothing typed removes rather than linking to nowhere.
				core.RunEditorCommand(bar.ref, core.EditLink(url))
			},
		},
	)

	return core.Modal(
		core.Visible(bar.linkOpen.Get()),
		core.OnDismiss(close),
		core.ModalContent(
			core.Column(
				core.BackgroundColor(t.Colors.Surface),
				core.BorderRadius(t.Components.Card.BorderRadius),
				core.Padding(t.Spacing.MD),
				core.Gap(float64(t.Spacing.SM)),
				core.MinWidth(richLinkPromptMinWidth),
				core.Text("Link", core.FontSize(t.Typography.Subtitle.FontSize),
					core.FontWeight(core.Bold)),
				core.Input(bar.linkURL.Get(), "https://", bar.linkURL.Set,
					core.AccessibilityLabel("Link address"),
				),
				core.Row(actions...),
			),
		),
	)
}

// richLinkPromptMinWidth keeps the sheet from shrinking to the width of its
// buttons: a modal sizes to its content, and a URL field with nothing in it has
// no content to size to. Same floor, and the same reason, as
// datePickerMinWidth.
const richLinkPromptMinWidth = "280px"
