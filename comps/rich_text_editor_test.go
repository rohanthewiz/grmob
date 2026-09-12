package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/richtext"
)

var note = richtext.Doc{Blocks: []richtext.Block{
	{Kind: richtext.Heading2, Runs: []richtext.Run{{Text: "Notes"}}},
	{Kind: richtext.Paragraph, Runs: []richtext.Run{{Text: "body"}}},
}}

// findNode walks a rendered tree for the first node satisfying pred.
func findNode(n *core.Node, pred func(*core.Node) bool) *core.Node {
	if n == nil {
		return nil
	}
	if pred(n) {
		return n
	}
	for _, child := range n.Children {
		if hit := findNode(child, pred); hit != nil {
			return hit
		}
	}
	return nil
}

func editorIn(t *testing.T, n *core.Node) *core.Node {
	t.Helper()
	hit := findNode(n, func(n *core.Node) bool { return n.Type == "RichTextEditor" })
	if hit == nil {
		t.Fatal("no RichTextEditor in the rendered tree")
	}
	return hit
}

// The display half: no toolbar, so no column, no wrapper and no hook. That last
// part is what lets a read-only note be rendered inside a conditional, which is
// the whole reason the toolbar is the caller's.
func TestRichTextEditorWithoutAToolbarIsTheNodeItself(t *testing.T) {
	ctx := core.NewContext()
	n := RichTextEditor{Doc: note, ReadOnly: true, Placeholder: "Nothing yet"}.Render(ctx)

	if n.Type != "RichTextEditor" {
		t.Fatalf("type = %q, want the node with no wrapper around it", n.Type)
	}
	if n.Props["doc"] != note.JSON() {
		t.Errorf("doc = %#v", n.Props["doc"])
	}
	if n.Props["readOnly"] != true || n.Props["placeholder"] != "Nothing yet" {
		t.Errorf("options did not reach the node: %#v", n.Props)
	}
	if _, ok := n.Props["onSelectionChange"]; ok {
		t.Error("an editor with no toolbar still wired a selection report")
	}
}

func TestRichTextEditorPassesItsSizingThrough(t *testing.T) {
	ctx := core.NewContext()
	n := RichTextEditor{Doc: note, MinHeight: "160px"}.Render(ctx)
	if n.Style == nil || n.Style.MinHeight != "160px" {
		t.Errorf("MinHeight did not reach the node: %+v", n.Style)
	}
}

// The toolbar: a column, a wrapping row of buttons above the editor, the ref on
// the editor, and the selection report wired so the buttons have something to
// draw from.
func TestRichTextEditorToolbarCommandsReachTheEditor(t *testing.T) {
	ctx := core.NewContext()
	var bar *RichToolbar
	view := core.ComponentFunc(func(c *core.Context) *core.Node {
		bar = UseRichToolbar(c)
		return RichTextEditor{Doc: note, Toolbar: bar}.Render(c)
	})

	ctx.Reset()
	n := view.Render(ctx)
	if n.Type != "Column" {
		t.Fatalf("a toolbar editor rendered as %q", n.Type)
	}
	strip := n.Children[0]
	if strip.Type != "Row" || len(strip.Children) != len(RichToolbarDefault) {
		t.Fatalf("toolbar = %q with %d buttons, want a Row of %d",
			strip.Type, len(strip.Children), len(RichToolbarDefault))
	}
	editor := editorIn(t, n)
	if _, ok := editor.Props["onSelectionChange"]; !ok {
		t.Error("the toolbar's editor does not report its selection")
	}

	// Tap "B" and confirm the stamp reaches the editor on the next pass.
	id, _ := strip.Children[0].Props["onClick"].(string)
	if id == "" {
		t.Fatal("the bold button registered no handler")
	}
	ctx.TriggerCallback(id)

	ctx.Reset()
	editor = editorIn(t, view.Render(ctx))
	if editor.Props["editorCommand"] != core.EditBold || editor.Props["editorEpoch"] != 1 {
		t.Errorf("after the tap: epoch %#v, command %#v",
			editor.Props["editorEpoch"], editor.Props["editorCommand"])
	}
}

// The pressed state comes from the last reported selection and nothing else:
// Go owns the document but not the caret, so which button should look pressed
// is a question only the host can answer.
func TestRichTextEditorToolbarDrawsTheReportedSelection(t *testing.T) {
	ctx := core.NewContext()
	var bar *RichToolbar
	view := core.ComponentFunc(func(c *core.Context) *core.Node {
		bar = UseRichToolbar(c)
		return RichTextEditor{Doc: note, Toolbar: bar}.Render(c)
	})

	ctx.Reset()
	n := view.Render(ctx)
	id, _ := editorIn(t, n).Props["onSelectionChange"].(string)
	if id == "" {
		t.Fatal("no selection report was wired")
	}
	ctx.TriggerTextCallback(id, `{"s":0,"e":5,"marks":["bold"],"block":"h2"}`)

	ctx.Reset()
	n = view.Render(ctx)
	if got := bar.Selection(); !got.Bold || got.Block != richtext.Heading2 {
		t.Fatalf("the toolbar did not remember the report: %#v", got)
	}

	// B and H2 read as pressed; I does not. The treatment is Button's two
	// emphases, which is what Chip uses for a filter's two states.
	strip := n.Children[0]
	labels := map[string]*core.Node{}
	for i, item := range RichToolbarDefault {
		labels[item.Label] = strip.Children[i]
	}
	if labels["B"].Style.BorderWidth == 0 {
		t.Error("the bold button is not drawn as active")
	}
	if labels["I"].Style.BorderWidth != 0 {
		t.Error("the italic button is drawn as active and nothing is italic")
	}
	if labels["H2"].Style.BorderWidth == 0 {
		t.Error("the heading button is not drawn as active")
	}
	if labels["¶"].Style.BorderWidth != 0 {
		t.Error("the paragraph button is active inside a heading")
	}
}

// The three shapes a command has, told apart by prefix rather than by a table —
// which is what lets a caller add a block kind of their own to Items and have it
// light up without touching the widget.
func TestRichToolActive(t *testing.T) {
	sel := core.RichSelection{Bold: true, Code: true, Link: "https://x", Block: richtext.Bullet}
	for command, want := range map[string]bool{
		core.EditBold:                       true,
		core.EditItalic:                     false,
		core.EditCode:                       true,
		RichToolLink:                        true,
		core.EditBlock(richtext.Bullet):     true,
		core.EditBlock(richtext.Paragraph):  false,
		core.EditBlock(richtext.BlockCode):  false,
		"something this widget never sends": false,
	} {
		if got := richToolActive(command, sel); got != want {
			t.Errorf("richToolActive(%q) = %v, want %v", command, got, want)
		}
	}
	// An empty Block is "the host had nothing to say", which is not the same as
	// a paragraph — so a document with no blocks yet lights nothing up.
	if richToolActive(core.EditBlock(richtext.Paragraph), core.RichSelection{}) {
		t.Error("an unreported block lit up the paragraph button")
	}
}

// The link prompt: pre-filled from the caret's own link, so "edit this link" is
// the same button as "add one", and offering Remove only when there is one.
func TestRichTextEditorLinkPrompt(t *testing.T) {
	ctx := core.NewContext()
	var bar *RichToolbar
	view := core.ComponentFunc(func(c *core.Context) *core.Node {
		bar = UseRichToolbar(c)
		return RichTextEditor{Doc: note, Toolbar: bar}.Render(c)
	})

	ctx.Reset()
	n := view.Render(ctx)
	modal := findNode(n, func(n *core.Node) bool { return n.Type == "Modal" })
	if modal == nil {
		t.Fatal("no link prompt in the tree")
	}
	if modal.Props["visible"] != false {
		t.Errorf("the prompt opens closed: %#v", modal.Props["visible"])
	}

	// A caret inside an existing link, then the link button.
	id, _ := editorIn(t, n).Props["onSelectionChange"].(string)
	ctx.TriggerTextCallback(id, `{"s":0,"e":4,"marks":[],"link":"https://example.com","block":"p"}`)
	ctx.Reset()
	n = view.Render(ctx)

	linkButton := n.Children[0].Children[len(RichToolbarDefault)-1]
	tap, _ := linkButton.Props["onClick"].(string)
	if tap == "" {
		t.Fatal("the link button registered no handler")
	}
	ctx.TriggerCallback(tap)

	ctx.Reset()
	n = view.Render(ctx)
	modal = findNode(n, func(n *core.Node) bool { return n.Type == "Modal" })
	if modal.Props["visible"] != true {
		t.Fatal("the prompt did not open")
	}
	if bar.linkURL.Get() != "https://example.com" {
		t.Errorf("the prompt was not pre-filled: %q", bar.linkURL.Get())
	}
	field := findNode(modal, func(n *core.Node) bool { return n.Type == "Input" })
	if field == nil || field.Props["value"] != "https://example.com" {
		t.Errorf("the field shows %#v", field)
	}
	// Remove is offered because the caret is in a link; a caret that is not in
	// one gets Cancel and Link alone.
	if findNode(modal, func(n *core.Node) bool {
		return n.Type == "Button" && n.Props["label"] == "Remove"
	}) == nil {
		t.Error("no Remove action inside an existing link")
	}
}

// A caller's Items are theirs: UseRichToolbar hands out a copy, so appending to
// one screen's toolbar cannot change every other screen's.
func TestUseRichToolbarCopiesTheDefaultItems(t *testing.T) {
	ctx := core.NewContext()
	var bar *RichToolbar
	core.ComponentFunc(func(c *core.Context) *core.Node {
		bar = UseRichToolbar(c)
		return core.Box().Render(c)
	}).Render(ctx)

	bar.Items[0] = RichToolItem{Label: "nonsense"}
	if RichToolbarDefault[0].Label != "B" {
		t.Error("UseRichToolbar handed out the package's own slice")
	}
}

// A read-only editor's toolbar is inert rather than absent: the buttons still
// say what the document is, and a disabled control announces itself as one.
func TestRichTextEditorReadOnlyDisablesTheToolbar(t *testing.T) {
	ctx := core.NewContext()
	view := core.ComponentFunc(func(c *core.Context) *core.Node {
		return RichTextEditor{Doc: note, ReadOnly: true, Toolbar: UseRichToolbar(c)}.Render(c)
	})
	ctx.Reset()
	n := view.Render(ctx)
	for i, button := range n.Children[0].Children {
		if button.Style == nil || !button.Style.Disabled {
			t.Errorf("toolbar button %d is live on a read-only editor", i)
		}
	}
}
