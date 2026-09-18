package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/richtext"
)

func sampleDoc() richtext.Doc {
	return richtext.Doc{Blocks: []richtext.Block{
		{Kind: richtext.Heading2, Runs: []richtext.Run{{Text: "A note"}}},
		{Kind: richtext.Paragraph, Runs: []richtext.Run{
			{Text: "Some "}, {Text: "bold", Bold: true}, {Text: " and a "},
			{Text: "link", Link: "https://example.com"}, {Text: "."},
		}},
		{Kind: richtext.Numbered, Runs: []richtext.Run{{Text: "one"}}},
		{Kind: richtext.Numbered, Runs: []richtext.Run{{Text: "two"}}},
		{Kind: richtext.Bullet, Runs: []richtext.Run{{Text: "dot"}}},
		{Kind: richtext.Quote, Runs: []richtext.Run{{Text: "said"}}},
		{Kind: richtext.BlockCode, Runs: []richtext.Run{{Text: "x := 1"}}},
		{Kind: richtext.Paragraph},
	}}
}

// Every block kind lands as a Paragraph in the shape the view's doc gives it,
// consecutive list items are one list and number from 1 per list, and the
// tree passes the accessibility audit.
func TestRichTextViewDrawsEveryBlock(t *testing.T) {
	var opened string
	ctx, root := renderDebug(t, RichTextView{Doc: sampleDoc(), OnLink: func(u string) { opened = u }})
	kids := root.Children
	if len(kids) != 7 {
		t.Fatalf("expected heading, paragraph, two lists, quote, code and empty line, got %d children", len(kids))
	}
	if h := kids[0]; h.Type != "Paragraph" || h.Style.AccessibilityRole != core.RoleHeading ||
		h.Style.AccessibilityHeadingLevel != 2 || h.Style.FontWeight != core.Bold {
		t.Errorf("h2 should be a bold level-2 heading Paragraph: %+v", h.Style)
	}

	para := kids[1]
	runs := para.Props["runs"].([]map[string]any)
	if len(runs) != 5 || runs[1]["b"] != 1 || runs[3]["u"] != 1 {
		t.Fatalf("the paragraph's runs lost their marks: %v", runs)
	}
	id, _ := runs[3]["cb"].(string)
	if id == "" {
		t.Fatal("the link run is not tappable")
	}
	ctx.TriggerCallback(id)
	if opened != "https://example.com" {
		t.Errorf("OnLink got %q", opened)
	}

	numbered, bullets := kids[2], kids[3]
	if numbered.Style.AccessibilityRole != core.RoleList || len(numbered.Children) != 2 {
		t.Fatalf("two numbered items should be one list: %d children", len(numbered.Children))
	}
	if got := numbered.Children[1].Children[0].Props["content"]; got != "2." {
		t.Errorf("the second item's marker = %v", got)
	}
	if len(bullets.Children) != 1 || bullets.Children[0].Children[0].Props["content"] != "•" {
		t.Error("a bullet after a numbered run starts its own list")
	}

	code := kids[5].Children[0].Props["runs"].([]map[string]any)
	if code[0]["c"] != 1 {
		t.Error("a code block's runs should all be code")
	}
	if empty := kids[6].Props["runs"].([]map[string]any); len(empty) != 1 || empty[0]["t"] != " " {
		t.Errorf("an empty block should keep a line: %v", empty)
	}
}
