package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// glyphTexts returns the Text nodes that draw the stars, in order.
func glyphTexts(n *core.Node, full, empty string) []*core.Node {
	var out []*core.Node
	var walk func(*core.Node)
	walk = func(n *core.Node) {
		if n.Type == "Text" && (n.Props["content"] == full || n.Props["content"] == empty) {
			out = append(out, n)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return out
}

func TestRatingDrawsRoundedGlyphsAndStatesTheScore(t *testing.T) {
	_, n := renderDebug(t, Rating{Value: 3.4, ReadOnly: true})

	g := glyphTexts(n, "★", "☆")
	if len(g) != 5 {
		t.Fatalf("want 5 glyphs, got %d", len(g))
	}
	for i, want := range []string{"★", "★", "★", "☆", "☆"} {
		if g[i].Props["content"] != want {
			t.Errorf("glyph %d = %v, want %s", i, g[i].Props["content"], want)
		}
	}
	if g[0].Style.TextColor != core.DefaultTheme.Colors.WarningOnLightColor() {
		t.Errorf("filled colour = %q", g[0].Style.TextColor)
	}
	if n.Style.AccessibilityRole != core.RoleGroup || n.Style.AccessibilityLabel != "Rating" {
		t.Errorf("group a11y = %q / %q", n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
	if v := n.Style.AccessibilityValue; v.Text != "3 of 5" || v.Now != "3" || v.Max != "5" {
		t.Errorf("value = %+v", v)
	}
}

func TestRatingReadOnlyRegistersNothingAndHidesGlyphs(t *testing.T) {
	_, n := renderDebug(t, Rating{Value: 4, OnChange: func(int) {}, ReadOnly: true})
	if findFirst(n, func(n *core.Node) bool { _, ok := n.Props["onClick"]; return ok }) != nil {
		t.Error("a read-only rating must register no callbacks")
	}
	for _, g := range glyphTexts(n, "★", "☆") {
		if !g.Style.AccessibilityHidden {
			t.Error("read-only glyphs are decoration; the group value is the announcement")
		}
	}
}

func TestRatingInteractiveGlyphsAreNamedButtons(t *testing.T) {
	var got []int
	ctx, n := renderDebug(t, Rating{Value: 2, Max: 4, OnChange: func(v int) { got = append(got, v) }})

	var btns []*core.Node
	for _, c := range n.Children {
		if c.Style != nil && c.Style.AccessibilityRole == core.RoleButton {
			btns = append(btns, c)
		}
	}
	if len(btns) != 4 {
		t.Fatalf("want 4 glyph buttons, got %d", len(btns))
	}
	if btns[2].Style.AccessibilityLabel != "3 of 4" {
		t.Errorf("third glyph label = %q", btns[2].Style.AccessibilityLabel)
	}
	ctx.TriggerCallback(btns[3].Props["onClick"].(string))
	ctx.TriggerCallback(btns[1].Props["onClick"].(string)) // the current value: no-op
	if len(got) != 1 || got[0] != 4 {
		t.Errorf("changes = %v, want [4]", got)
	}
}

func TestRatingNilOnChangeIsDisplayOnlyAndGlyphsOverride(t *testing.T) {
	_, n := renderDebug(t, Rating{Value: 9, Max: 3, Glyph: "●", EmptyGlyph: "○", Label: "Spice"})
	g := glyphTexts(n, "●", "○")
	if len(g) != 3 || g[2].Props["content"] != "●" {
		t.Errorf("a value above Max clamps to all filled, got %d glyphs", len(g))
	}
	if n.Style.AccessibilityLabel != "Spice" || n.Style.AccessibilityValue.Text != "3 of 3" {
		t.Errorf("label/value = %q / %q", n.Style.AccessibilityLabel, n.Style.AccessibilityValue.Text)
	}
	if findFirst(n, func(n *core.Node) bool { _, ok := n.Props["onClick"]; return ok }) != nil {
		t.Error("no OnChange means no handlers")
	}
}
