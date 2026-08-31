package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// middleOf returns the row's growing centre column — the Column child whose
// FlexGrow is 1. Located by predicate rather than index so these tests keep
// passing if the row ever gains a decorative wrapper slot.
func middleOf(n *core.Node) *core.Node {
	for _, c := range n.Children {
		if c.Type == "Column" && c.Style != nil && c.Style.FlexGrow == 1 {
			return c
		}
	}
	return nil
}

func TestListRowStructureAndSlotOrder(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := ListRow{
		Leading:  core.Checkbox(false, func(bool) {}),
		Title:    "Buy milk",
		Subtitle: "Due today",
		Trailing: core.Text("✕"),
	}.Render(ctx)

	if n.Type != "Row" {
		t.Fatalf("root type = %q, want Row (the theme's Row base supplies the padding)", n.Type)
	}
	if len(n.Children) != 3 {
		t.Fatalf("want leading, middle, trailing; got %d children", len(n.Children))
	}
	if n.Children[0].Type != "Checkbox" {
		t.Errorf("leading slot must render first, got %q", n.Children[0].Type)
	}
	if n.Children[2].Props["content"] != "✕" {
		t.Errorf("trailing slot must render last, got %v", n.Children[2].Props)
	}

	mid := middleOf(n)
	if mid == nil {
		t.Fatal("the middle column must carry FlexGrow(1) — that is what pins Trailing to the edge")
	}
	if mid != n.Children[1] {
		t.Error("the growing middle must sit between leading and trailing")
	}
	if findText(mid, "Buy milk") == nil || findText(mid, "Due today") == nil {
		t.Error("Title and Subtitle both belong in the middle column")
	}
	if mid.Style.Padding != (core.EdgeInsets{}) {
		t.Errorf("middle column must zero the theme Column padding, got %+v", mid.Style.Padding)
	}
}

// The pinning mechanism must be FlexGrow on the middle, never JustifyBetween
// on the row: the two disagree whenever a slot is missing, and reproducing
// that disagreement is what this widget exists to prevent.
func TestListRowPinsWithFlexGrowNotJustify(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ListRow{Title: "Balance", Trailing: core.Text("$12.00")}.Render(ctx)

	if n.Style.JustifyContent != "" {
		t.Errorf("row must not set JustifyContent, got %q", n.Style.JustifyContent)
	}
	if middleOf(n) == nil {
		t.Error("middle column missing its FlexGrow")
	}
	if n.Style.AlignItems != core.AlignItemsCenter {
		t.Errorf("row should centre its slots vertically, got %q", n.Style.AlignItems)
	}
}

// The middle is structure, not content: it is emitted even when empty so the
// trailing slot pins identically in every configuration.
func TestListRowKeepsMiddleWhenEmpty(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ListRow{Leading: core.Text("icon"), Trailing: core.Text("›")}.Render(ctx)

	mid := middleOf(n)
	if mid == nil {
		t.Fatal("the growing middle must survive an empty Title/Subtitle/Content")
	}
	if len(mid.Children) != 0 {
		t.Errorf("an empty middle must render no placeholder text, got %d children", len(mid.Children))
	}
}

func TestListRowContentOverridesTitleAndSubtitle(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ListRow{
		Title:    "ignored",
		Subtitle: "also ignored",
		Content:  core.Text("custom middle"),
	}.Render(ctx)

	if findText(n, "custom middle") == nil {
		t.Error("Content slot should render")
	}
	if findText(n, "ignored") != nil || findText(n, "also ignored") != nil {
		t.Error("Title/Subtitle must be suppressed when Content is set — the slot is the escape hatch")
	}
}

func TestListRowTapAndLongPress(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	var tapped, held bool
	n := ListRow{
		Title:       "Article 1",
		OnTap:       func() { tapped = true },
		OnLongPress: func() { held = true },
	}.Render(ctx)

	tapID, ok := n.Props["onClick"].(string)
	if !ok {
		t.Fatal("OnTap should register an onClick callback on the row itself")
	}
	holdID, ok := n.Props["onLongPress"].(string)
	if !ok {
		t.Fatal("OnLongPress should register an onLongPress callback")
	}
	ctx.TriggerCallback(tapID)
	ctx.TriggerCallback(holdID)
	if !tapped || !held {
		t.Errorf("callbacks did not fire: tapped=%v held=%v", tapped, held)
	}
}

// A presentational row must stay free of handler props, so a long list does
// not register a callback per row for nothing.
func TestListRowWithoutHandlersRegistersNothing(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ListRow{Title: "read only"}.Render(ctx)

	if _, ok := n.Props["onClick"]; ok {
		t.Error("no OnTap should mean no onClick prop")
	}
	if _, ok := n.Props["onLongPress"]; ok {
		t.Error("no OnLongPress should mean no onLongPress prop")
	}
}

func TestListRowSelectedThemeDefault(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ListRow{Title: "Article 1", Selected: true}.Render(ctx)

	if n.Style.Background != core.DefaultTheme.Colors.Surface {
		t.Errorf("selected background = %q, want theme Surface %q", n.Style.Background, core.DefaultTheme.Colors.Surface)
	}
}

func TestListRowSelectedStyleOverridesCallerStyle(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ListRow{
		Title:         "Article 1",
		Selected:      true,
		Style:         []core.StyleProp{core.BackgroundColor("#FFFFFF")},
		SelectedStyle: []core.StyleProp{core.BackgroundColor("#E8F0FE")},
	}.Render(ctx)

	if n.Style.Background != "#E8F0FE" {
		t.Errorf("selection must win over the base Style, got %q", n.Style.Background)
	}
}

func TestListRowStyleOverridesWidgetDefaults(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ListRow{
		Title: "Article 1",
		Style: []core.StyleProp{core.Gap(2), core.Padding(0)},
	}.Render(ctx)

	if n.Style.Gap != 2 {
		t.Errorf("caller Gap should beat the widget default, got %v", n.Style.Gap)
	}
	if n.Style.Padding != (core.EdgeInsets{}) {
		t.Errorf("caller Padding should beat the theme Row base, got %+v", n.Style.Padding)
	}
}

func TestListRowAccessibilityAnnouncesSelection(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	base := ListRow{
		Title:              "Article 1",
		AccessibilityLabel: "Article 1",
		AccessibilityHint:  "Selects the article; long-press to star it",
	}
	unselected := base.Render(ctx)
	base.Selected = true
	selected := base.Render(ctx)

	if got := unselected.Style.AccessibilityLabel; got != "Article 1" {
		t.Errorf("unselected label = %q", got)
	}
	if got := selected.Style.AccessibilityLabel; got != "Article 1, selected" {
		t.Errorf("selected label = %q, want the state appended", got)
	}
	if got := selected.Style.AccessibilityHint; got != "Selects the article; long-press to star it" {
		t.Errorf("hint = %q", got)
	}
}

// No label is synthesized from Title: labelling the container would override
// how the row's own children are announced.
func TestListRowSynthesizesNoLabel(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ListRow{Title: "Buy milk", Selected: true}.Render(ctx)

	if got := n.Style.AccessibilityLabel; got != "" {
		t.Errorf("row label = %q, want empty when the caller named nothing", got)
	}
}
