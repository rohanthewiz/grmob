package core

import (
	"encoding/json"
	"testing"
)

func renderParagraph(ctx *Context, runs []Span, props ...PropsAndChildren) *Node {
	ctx.BeginRenderPass()
	return Paragraph(runs, props...).Render(ctx)
}

// The runs travel as the wire's short keys, only the marks a run has, and a
// run with no text is dropped.
func TestParagraphRunsOnTheWire(t *testing.T) {
	ctx := NewContext().WithTheme(DefaultTheme)
	n := renderParagraph(ctx, []Span{
		{Text: "plain "},
		{Text: ""},
		{Text: "all", Bold: true, Italic: true, Underline: true, Strike: true, Code: true, Color: "#123456"},
	}, FontSize(15))
	if n.Type != "Paragraph" || n.Style.FontSize != 15 {
		t.Fatalf("got %s with font size %v", n.Type, n.Style.FontSize)
	}
	b, _ := json.Marshal(n.Props["runs"])
	want := `[{"t":"plain "},{"b":1,"c":1,"fg":"#123456","i":1,"s":1,"t":"all","u":1}]`
	if string(b) != want {
		t.Errorf("runs = %s, want %s", b, want)
	}
}

// A run with OnTap is a link: it carries a void callback that runs the
// handler, and the theme's Primary unless it names its own colour.
func TestParagraphLinkRuns(t *testing.T) {
	ctx := NewContext().WithTheme(DefaultTheme)
	tapped := 0
	n := renderParagraph(ctx, []Span{
		{Text: "terms", OnTap: func() { tapped++ }},
		{Text: "red", Color: "#ff0000", OnTap: func() {}},
	})
	runs := n.Props["runs"].([]map[string]any)
	if runs[0]["fg"] != DefaultTheme.Colors.Primary {
		t.Errorf("an uncoloured link should take Primary, got %v", runs[0]["fg"])
	}
	if runs[1]["fg"] != "#ff0000" {
		t.Errorf("a link's own colour should win, got %v", runs[1]["fg"])
	}
	id, _ := runs[0]["cb"].(string)
	if id == "" {
		t.Fatal("a link run carries no callback")
	}
	ctx.TriggerCallback(id)
	if tapped != 1 {
		t.Fatalf("the link's callback ran the handler %d times", tapped)
	}
}
