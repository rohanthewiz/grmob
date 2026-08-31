package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func TestCardTitleAndSlots(t *testing.T) {
	ctx := core.NewContext()
	n := Card{
		Title:  "Account",
		Body:   core.Text("body content"),
		Footer: core.Text("footer content"),
	}.Render(ctx)

	if n.Type != "Card" {
		t.Fatalf("root type = %q, want Card (theme base must come from core.Card)", n.Type)
	}
	title := findText(n, "Account")
	if title == nil {
		t.Fatal("Title should render as a Text child")
	}
	if title.Style == nil || title.Style.FontWeight != core.Bold {
		t.Errorf("title should be bold, got style %+v", title.Style)
	}
	if findText(n, "body content") == nil || findText(n, "footer content") == nil {
		t.Error("Body and Footer slots should both render")
	}
	// Regions must appear in header, body, footer order.
	if len(n.Children) != 3 {
		t.Fatalf("want 3 children (title, body, footer), got %d", len(n.Children))
	}
	if n.Children[0].Props["content"] != "Account" || n.Children[2].Props["content"] != "footer content" {
		t.Errorf("region order wrong: %v, %v", n.Children[0].Props, n.Children[2].Props)
	}
}

func TestCardHeaderSlotOverridesTitle(t *testing.T) {
	ctx := core.NewContext()
	n := Card{
		Title:  "ignored",
		Header: core.Text("custom header"),
		Body:   core.Text("body"),
	}.Render(ctx)

	if findText(n, "custom header") == nil {
		t.Error("Header slot should render")
	}
	if findText(n, "ignored") != nil {
		t.Error("Title must be suppressed when Header is set — the slot is the escape hatch")
	}
}

func TestCardOmitsEmptyRegions(t *testing.T) {
	ctx := core.NewContext()
	n := Card{Body: core.Text("only body")}.Render(ctx)
	if len(n.Children) != 1 {
		t.Errorf("unset regions must not render placeholders, got %d children", len(n.Children))
	}
}
