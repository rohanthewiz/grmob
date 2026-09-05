package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func TestStatTileOrderAndRoles(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	theme := core.DefaultTheme

	n := StatTile{Label: "Attendance", Value: "412", Delta: "+18 vs last week"}.Render(ctx)

	if len(n.Children) != 3 {
		t.Fatalf("children = %d, want label, value, delta", len(n.Children))
	}
	// Label above the figure: in a row of tiles that keeps the labels on one
	// line and the figures on another, which survives labels of different
	// lengths.
	if n.Children[0].Props["content"] != "Attendance" {
		t.Errorf("first child = %v, want the label", n.Children[0].Props["content"])
	}
	if n.Children[1].Props["content"] != "412" {
		t.Errorf("second child = %v, want the value", n.Children[1].Props["content"])
	}

	label := n.Children[0]
	if label.Style.FontSize != theme.Typography.Caption.FontSize {
		t.Errorf("label size = %v, want the Caption role", label.Style.FontSize)
	}
	value := n.Children[1]
	if value.Style.FontSize != theme.Typography.Title.FontSize {
		t.Errorf("value size = %v, want the Title role", value.Style.FontSize)
	}
	// Ink included: the figure is content and takes the theme's own heading
	// color, not a brand accent laid on top of it.
	if value.Style.TextColor != theme.Typography.Title.TextColor {
		t.Errorf("value ink = %q, want the Title role's own %q", value.Style.TextColor, theme.Typography.Title.TextColor)
	}
}

// The one place in this package where VariantDefault is not the theme's
// Primary. A delta is a measurement, and whether up is good is the caller's
// domain — so the default says nothing.
func TestStatTileDeltaDefaultsToNeutralInkNotPrimary(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	theme := core.DefaultTheme

	n := StatTile{Label: "Spend", Value: "MZN 12,400", Delta: "+8%"}.Render(ctx)

	delta := findText(n, "+8%")
	if delta == nil {
		t.Fatal("no delta")
	}
	if delta.Style.TextColor != theme.Colors.TextSecondary {
		t.Errorf("delta ink = %q, want the secondary ink %q", delta.Style.TextColor, theme.Colors.TextSecondary)
	}
	if delta.Style.TextColor == theme.Colors.Primary {
		t.Error("the zero variant must not resolve to Primary here — a movement is not a brand accent")
	}
}

func TestStatTileDeltaVariantColors(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	theme := core.DefaultTheme

	for _, v := range []Variant{VariantSuccess, VariantWarning, VariantError} {
		n := StatTile{Value: "1", Delta: "up", DeltaVariant: v}.Render(ctx)
		delta := findText(n, "up")
		if delta == nil {
			t.Fatalf("variant %q: no delta", v)
		}
		if delta.Style.TextColor != v.Color(theme) {
			t.Errorf("variant %q ink = %q, want the role color %q", v, delta.Style.TextColor, v.Color(theme))
		}
	}
}

// Fill sets FlexGrow *and* a zero FlexBasis. The natives divide the whole axis
// by weight already; CSS flex-grow divides only the leftover space, so without
// the basis a tile with a longer value comes out wider on the web alone.
func TestStatTileFillConvergesTheTargets(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := StatTile{Label: "Attendance", Value: "412", Fill: true}.Render(ctx)
	if n.Style.FlexGrow != 1 {
		t.Errorf("FlexGrow = %v, want 1", n.Style.FlexGrow)
	}
	if n.Style.FlexBasis != "0" {
		t.Errorf("FlexBasis = %q, want %q — without it CSS shares only the slack", n.Style.FlexBasis, "0")
	}

	if plain := (StatTile{Value: "412"}).Render(ctx); plain.Style.FlexGrow != 0 || plain.Style.FlexBasis != "" {
		t.Error("a tile without Fill should hug its content and carry neither prop")
	}
}

// "Tile" names the content, not a card: the widget paints and insets nothing,
// which is what lets three tiles share one card or take one each.
func TestStatTileHasNoFrame(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := StatTile{Label: "Giving", Value: "MZN 42,750"}.Render(ctx)

	if n.Type != "Box" {
		t.Errorf("container = %q, want Box (no theme base)", n.Type)
	}
	if n.Style.Background != "" {
		t.Errorf("background = %q, want none", n.Style.Background)
	}
	if (n.Style.Padding != core.EdgeInsets{}) {
		t.Errorf("padding = %+v, want zero", n.Style.Padding)
	}
	if n.Style.BorderWidth != 0 || n.Style.BorderRadius != 0 {
		t.Error("a tile should draw no border and round no corners")
	}
}

func TestStatTileEmptyFieldsCostNoNode(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	if n := (StatTile{Value: "412"}).Render(ctx); len(n.Children) != 1 {
		t.Errorf("children = %d, want just the value", len(n.Children))
	}
	if n := (StatTile{}).Render(ctx); len(n.Children) != 0 {
		t.Errorf("an empty tile rendered %d children, want none", len(n.Children))
	}
}

func TestStatTileTapAndAccessibility(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	// Presentational: no callback in the props at all, so nothing survives the
	// render pass's callback sweep for nothing.
	plain := StatTile{Label: "Attendance", Value: "412"}.Render(ctx)
	if _, ok := plain.Props["onClick"]; ok {
		t.Error("a presentational tile should register no handler")
	}

	opened := false
	n := StatTile{
		Label: "Attendance", Value: "412",
		OnTap:              func() { opened = true },
		AccessibilityLabel: "Attendance, 412, up 18 from last week",
		AccessibilityHint:  "Opens the attendance report",
	}.Render(ctx)

	id, ok := n.Props["onClick"].(string)
	if !ok {
		t.Fatal("OnTap should make the whole tile a target")
	}
	ctx.TriggerCallback(id)
	if !opened {
		t.Error("tapping should invoke OnTap")
	}
	if n.Style.AccessibilityLabel != "Attendance, 412, up 18 from last week" {
		t.Errorf("label = %q, want the caller's", n.Style.AccessibilityLabel)
	}
	if n.Style.AccessibilityHint != "Opens the attendance report" {
		t.Errorf("hint = %q, want the caller's", n.Style.AccessibilityHint)
	}
	// Nothing is synthesized: the three lines carry meaning the widget cannot
	// combine correctly on its own.
	if plain.Style.AccessibilityLabel != "" {
		t.Errorf("unlabeled tile label = %q, want none synthesized", plain.Style.AccessibilityLabel)
	}
}

func TestStatTileStyleOverridesTheDefaults(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := StatTile{Label: "x", Value: "1", Style: []core.StyleProp{
		core.AlignItemsProp(core.AlignItemsCenter), core.Gap(0),
	}}.Render(ctx)

	if n.Style.AlignItems != core.AlignItemsCenter {
		t.Errorf("alignItems = %q, want the caller's centering recipe", n.Style.AlignItems)
	}
	if n.Style.Gap != 0 {
		t.Errorf("gap = %v, want the caller's override", n.Style.Gap)
	}
}
