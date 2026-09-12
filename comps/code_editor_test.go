package comps

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/highlight"
)

// editorNode digs the core.CodeEditor out of whatever the widget rendered — it
// is the node itself when there is no toolbar and the column's second child
// when there is.
func editorNode(t *testing.T, n *core.Node) *core.Node {
	t.Helper()
	if n.Type == "CodeEditor" {
		return n
	}
	for _, c := range n.Children {
		if c.Type == "CodeEditor" {
			return c
		}
	}
	t.Fatalf("no CodeEditor in the rendered tree: %s", n.Type)
	return nil
}

func rowText(row core.GridRow) string {
	var b strings.Builder
	for _, run := range row {
		b.WriteString(run.Text)
	}
	return b.String()
}

// The plain case: no toolbar, so no column, no hook and no wrapper — the widget
// renders the node itself. That last part is what lets a read-only code block be
// rendered inside a conditional, which is the whole reason the ref is the
// caller's.
func TestCodeEditorWithoutAToolbarIsTheNodeItself(t *testing.T) {
	ctx := core.NewContext()
	n := CodeEditor{Value: "func f() {}", Language: "go"}.Render(ctx)

	if n.Type != "CodeEditor" {
		t.Fatalf("type = %q, want the node with no wrapper around it", n.Type)
	}
	if len(n.Children) != 1 {
		t.Errorf("one line of source produced %d rows", len(n.Children))
	}
}

func TestCodeEditorRunsTheLanguageLexer(t *testing.T) {
	ctx := core.NewContext()
	n := CodeEditor{Value: "func f() {}", Language: "go", Scheme: highlight.Darcula}.Render(ctx)

	runs, ok := n.Children[0].Props["runs"].(core.GridRow)
	if !ok || len(runs) == 0 {
		t.Fatalf("row 0 carries %#v", n.Children[0].Props["runs"])
	}
	if runs[0].Text != "func" || runs[0].Fg != highlight.Darcula.Keyword {
		t.Errorf("the keyword was not coloured: %#v", runs[0])
	}
	if rowText(runs) != "func f() {}" {
		t.Errorf("the rows do not reassemble the source: %q", rowText(runs))
	}
}

// An unknown language is uncoloured, not an error. A screen whose editor
// mis-spells its language has to render.
func TestCodeEditorWithAnUnknownLanguageIsPlain(t *testing.T) {
	ctx := core.NewContext()
	n := CodeEditor{Value: "func f() {}", Language: "gopher"}.Render(ctx)
	runs, _ := n.Children[0].Props["runs"].(core.GridRow)
	if len(runs) != 1 || runs[0].Fg != "" {
		t.Errorf("an unknown language produced decoration: %#v", runs)
	}
}

// An explicit Highlighter is the more specific statement and wins over Language.
func TestCodeEditorHighlighterOverridesLanguage(t *testing.T) {
	ctx := core.NewContext()
	n := CodeEditor{Value: `{"a":1}`, Language: "go", Highlighter: highlight.JSON()}.Render(ctx)
	runs, _ := n.Children[0].Props["runs"].(core.GridRow)
	if len(runs) < 2 {
		t.Fatalf("the JSON lexer did not run: %#v", runs)
	}
}

// The scheme is derived from the theme's own background when the caller names
// none, so an editor in a light app is not a dark rectangle and one in a dark
// app is not a white page.
func TestCodeEditorPicksASchemeFromTheTheme(t *testing.T) {
	for name, want := range map[string]highlight.Scheme{
		"light": highlight.Light,
		"dark":  highlight.Darcula,
	} {
		// A copy of the bundled theme with one role changed, which is the
		// documented way to brand one — and the only part of it this widget
		// consults.
		theme := *core.DefaultTheme
		theme.Colors.Background = map[string]string{"light": "#FFFFFF", "dark": "#101014"}[name]
		ctx := core.NewContext().WithTheme(&theme)

		n := CodeEditor{Value: "x"}.Render(ctx)
		if n.Style == nil || n.Style.Background != want.Bg {
			t.Errorf("%s theme: surface = %q, want %q", name, n.Style.Background, want.Bg)
		}
		if n.Style.TextColor != want.Ink {
			t.Errorf("%s theme: ink = %q, want %q", name, n.Style.TextColor, want.Ink)
		}
	}
}

// A named scheme wins outright, whatever the theme is.
func TestCodeEditorSchemeOverridesTheTheme(t *testing.T) {
	theme := *core.DefaultTheme
	theme.Colors.Background = "#FFFFFF"
	ctx := core.NewContext().WithTheme(&theme)

	n := CodeEditor{Value: "x", Scheme: highlight.Darcula}.Render(ctx)
	if n.Style.Background != highlight.Darcula.Bg {
		t.Errorf("surface = %q, want Darcula's", n.Style.Background)
	}
}

func TestCodeEditorPassesItsOptionsThrough(t *testing.T) {
	ctx := core.NewContext()
	n := CodeEditor{
		Value:       "x",
		Language:    "go",
		LineNumbers: true,
		ReadOnly:    true,
		TabSize:     2,
		Height:      "240px",
	}.Render(ctx)

	for key, want := range map[string]any{
		"lineNumbers":   true,
		"readOnly":      true,
		"tabSize":       2,
		"commentPrefix": "//",
	} {
		if got := n.Props[key]; got != want {
			t.Errorf("%q = %#v, want %#v", key, got, want)
		}
	}
	if n.Style == nil || n.Style.Height != "240px" {
		t.Errorf("Height did not reach the node: %+v", n.Style)
	}
	// The theme's field frame would draw a box around a surface that is already
	// its own box, in a colour chosen for a form control.
	if n.Style.BorderWidth != 0 {
		t.Errorf("the theme's field frame survived: %v", n.Style.BorderWidth)
	}
}

// JSON has no line comment, so the widget says so explicitly rather than
// leaving the node's "//" default to make the document invalid.
func TestCodeEditorTakesTheCommentMarkerFromTheLanguage(t *testing.T) {
	ctx := core.NewContext()
	for language, want := range map[string]string{"go": "//", "json": ""} {
		n := CodeEditor{Value: "x", Language: language}.Render(ctx)
		if got := n.Props["commentPrefix"]; got != want {
			t.Errorf("%s: commentPrefix = %#v, want %q", language, got, want)
		}
	}
	// An editor that names no language keeps the node's own default, which is
	// right for most of what a code editor in an app shows.
	n := CodeEditor{Value: "x"}.Render(ctx)
	if got := n.Props["commentPrefix"]; got != "//" {
		t.Errorf("no language: commentPrefix = %#v, want the node's default", got)
	}
	// And an explicit prefix wins over both.
	n = CodeEditor{Value: "x", Language: "go", CommentPrefix: "#"}.Render(ctx)
	if got := n.Props["commentPrefix"]; got != "#" {
		t.Errorf("explicit prefix = %#v", got)
	}
}

// The toolbar: a column, a row of buttons above the editor, and the ref on the
// editor so the commands have somewhere to land.
func TestCodeEditorToolbarCommandsReachTheEditor(t *testing.T) {
	ctx := core.NewContext()
	var ref *core.EditorRef
	view := core.ComponentFunc(func(c *core.Context) *core.Node {
		ref = core.UseEditorRef(c)
		return CodeEditor{Value: "a\nb", Language: "go", Toolbar: ref}.Render(c)
	})

	ctx.Reset()
	n := view.Render(ctx)
	if n.Type != "Column" || len(n.Children) != 2 {
		t.Fatalf("a toolbar editor rendered as %q with %d children", n.Type, len(n.Children))
	}
	bar := n.Children[0]
	if bar.Type != "Row" || len(bar.Children) != 3 {
		t.Fatalf("toolbar = %q with %d buttons, want a Row of three", bar.Type, len(bar.Children))
	}

	editor := editorNode(t, n)
	if _, stamped := editor.Props["editorEpoch"]; stamped {
		t.Error("an editor whose toolbar has not been touched carries a command stamp")
	}

	// Tap "indent" and confirm the stamp reaches the editor on the next pass.
	id, _ := bar.Children[0].Props["onClick"].(string)
	if id == "" {
		t.Fatal("the indent button registered no handler")
	}
	ctx.TriggerCallback(id)

	ctx.Reset()
	n = view.Render(ctx)
	editor = editorNode(t, n)
	if editor.Props["editorCommand"] != core.EditIndent || editor.Props["editorEpoch"] != 1 {
		t.Errorf("after the tap: epoch %#v, command %#v",
			editor.Props["editorEpoch"], editor.Props["editorCommand"])
	}
}

// A language with no line comment loses the button rather than showing an inert
// one: a permanently disabled control is a question the reader has to answer
// every time they see it.
func TestCodeEditorToolbarDropsTheCommentButtonForJSON(t *testing.T) {
	ctx := core.NewContext()
	var ref *core.EditorRef
	view := core.ComponentFunc(func(c *core.Context) *core.Node {
		ref = core.UseEditorRef(c)
		return CodeEditor{Value: "{}", Language: "json", Toolbar: ref}.Render(c)
	})
	ctx.Reset()
	n := view.Render(ctx)
	if got := len(n.Children[0].Children); got != 2 {
		t.Errorf("JSON toolbar has %d buttons, want indent and outdent only", got)
	}
}

// A nil OnChange must not reach the node as a nil callback: core.CodeEditor
// registers whatever it is given and the dispatcher invokes it unguarded.
func TestCodeEditorWithNoHandlerDoesNotPanicOnAKeystroke(t *testing.T) {
	ctx := core.NewContext()
	n := CodeEditor{Value: "x"}.Render(ctx)
	id, _ := n.Props["onChange"].(string)
	if id == "" {
		t.Fatal("no onChange was registered")
	}
	ctx.TriggerTextCallback(id, "xy") // must not panic
}

// The selection handler is wired through and parsed in core, so the widget's
// caller sees two ints.
func TestCodeEditorReportsSelection(t *testing.T) {
	ctx := core.NewContext()
	var got [2]int
	n := CodeEditor{
		Value:             "abc",
		OnSelectionChange: func(start, end int) { got = [2]int{start, end} },
	}.Render(ctx)

	id, _ := n.Props["onSelectionChange"].(string)
	if id == "" {
		t.Fatal("no onSelectionChange was registered")
	}
	ctx.TriggerTextCallback(id, "1:2")
	if got != [2]int{1, 2} {
		t.Errorf("got %v", got)
	}
}
