package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func TestChipStripWrapsAndSpaces(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	theme := core.DefaultTheme

	n := ChipStrip{Chips: []Chip{
		{Label: "All"}, {Label: "Sermons"}, {Label: "Articles"},
	}}.Render(ctx)

	// Wrapping, not scrolling: core.Scroll is vertical only today, so a long
	// strip takes a second line rather than advertising a scroll it cannot do.
	if n.Style.FlexWrap != "wrap" {
		t.Errorf("flexWrap = %q, want %q", n.Style.FlexWrap, "wrap")
	}
	if n.Style.Gap != float64(theme.Spacing.SM) {
		t.Errorf("gap = %v, want the theme SM step %v", n.Style.Gap, theme.Spacing.SM)
	}
	// The theme's Row inset is assigned over: a strip of chips is a run of
	// content inside a screen that is already padded, not a row with margins.
	if (n.Style.Padding != core.EdgeInsets{}) {
		t.Errorf("padding = %+v, want zero", n.Style.Padding)
	}
	if len(n.Children) != 3 {
		t.Fatalf("children = %d, want 3", len(n.Children))
	}
	for i, want := range []string{"All", "Sermons", "Articles"} {
		if n.Children[i].Props["label"] != want {
			t.Errorf("child %d = %v, want %q — order must be preserved", i, n.Children[i].Props["label"], want)
		}
	}
}

// Taking []Chip rather than a parallel vocabulary is what keeps a chip in a
// strip configured exactly like a chip anywhere else — Selected and OnTap
// included.
func TestChipStripChipsKeepTheirOwnConfiguration(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	tapped := ""
	n := ChipStrip{Chips: []Chip{
		{Label: "All", Selected: true, OnTap: func() { tapped = "All" }},
		{Label: "Sermons", OnTap: func() { tapped = "Sermons" }},
	}}.Render(ctx)

	theme := core.DefaultTheme
	if n.Children[0].Style.Background != theme.Colors.Surface {
		t.Error("the selected chip should carry Chip's selected look")
	}
	if n.Children[1].Style.Background != theme.Components.Button.Background {
		t.Error("the unselected chip should carry the theme Button base")
	}
	ctx.TriggerCallback(n.Children[1].Props["onClick"].(string))
	if tapped != "Sermons" {
		t.Errorf("tapped = %q, want Sermons", tapped)
	}
}

func TestChipStripCustomGap(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ChipStrip{Chips: []Chip{{Label: "x"}}, Gap: 20}.Render(ctx)

	if n.Style.Gap != 20 {
		t.Errorf("gap = %v, want 20", n.Style.Gap)
	}
}

func TestChipStripChildrenSlotWinsOverChips(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ChipStrip{
		Chips: []Chip{{Label: "ignored"}},
		Children: []core.View{
			Chip{Label: "Matthew 5"},
			// A nil entry stands for a conditional item and must cost no node.
			nil,
			Badge{Text: "3 more"},
		},
	}.Render(ctx)

	if findButton(n, "ignored") != nil {
		t.Error("Children should replace Chips, not sit beside them")
	}
	if len(n.Children) != 2 {
		t.Fatalf("children = %d, want 2 — the nil entry should be skipped", len(n.Children))
	}
	if findButton(n, "Matthew 5") == nil || findText(n, "3 more") == nil {
		t.Error("both non-nil children should be rendered")
	}
}

// An empty Children slice is still a decision — "render nothing" — and must
// not fall back to Chips.
func TestChipStripEmptyChildrenSliceStillWins(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ChipStrip{Chips: []Chip{{Label: "x"}}, Children: []core.View{}}.Render(ctx)

	if len(n.Children) != 0 {
		t.Errorf("children = %d, want 0", len(n.Children))
	}
}

func TestChipStripStyleOverridesTheDefaults(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ChipStrip{Chips: []Chip{{Label: "x"}}, Style: []core.StyleProp{
		core.Padding(12), core.FlexWrap(false),
	}}.Render(ctx)

	if n.Style.FlexWrap != "nowrap" {
		t.Errorf("flexWrap = %q, want the caller's override %q", n.Style.FlexWrap, "nowrap")
	}
	if n.Style.Padding.Top != 12 {
		t.Errorf("padding = %+v, want the caller's override", n.Style.Padding)
	}
}
