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

// Halves: 3.6 rounds to 3.5 — three full stars, one half, one empty, all
// drawn as canvases — announced "3.5 of 5".
func TestRatingHalvesDrawsAHalfStar(t *testing.T) {
	_, n := renderDebug(t, Rating{Value: 3.6, Halves: true, ReadOnly: true})

	if v := n.Style.AccessibilityValue; v.Text != "3.5 of 5" || v.Now != "3.5" {
		t.Errorf("value = %+v, want 3.5 of 5", v)
	}
	if len(n.Children) != 5 {
		t.Fatalf("glyphs = %d, want 5", len(n.Children))
	}
	on := core.DefaultTheme.Colors.WarningOnLightColor()
	for i, want := range []string{"full", "full", "full", "half", "empty"} {
		c := n.Children[i]
		if c.Type != "Canvas" || !c.Style.AccessibilityHidden {
			t.Fatalf("glyph %d = %q, want a hidden Canvas", i, c.Type)
		}
		shapes := c.Children
		var got string
		switch {
		case len(shapes) == 2 && shapes[0].Props["fill"] == on:
			got = "half"
		case len(shapes) == 1 && shapes[0].Props["fill"] == on:
			got = "full"
		case len(shapes) == 1 && shapes[0].Props["fill"] == nil:
			got = "empty"
		default:
			got = "unrecognized"
		}
		if got != want {
			t.Errorf("glyph %d is %s, want %s", i, got, want)
		}
	}
}

// Rounding to halves at the edges: 4.74 is 4.5, 4.75 is 5, and a value past
// Max clamps.
func TestRatingHalvesRounding(t *testing.T) {
	for v, want := range map[float64]string{4.74: "4.5 of 5", 4.75: "5 of 5", 0.2: "0 of 5", 0.3: "0.5 of 5", 9: "5 of 5"} {
		_, n := renderDebug(t, Rating{Value: v, Halves: true, ReadOnly: true})
		if got := n.Style.AccessibilityValue.Text; got != want {
			t.Errorf("Value %v announced %q, want %q", v, got, want)
		}
	}
}

// An interactive Halves rating keeps its whole-star buttons, each holding a
// drawn star, and still reports whole numbers.
func TestRatingHalvesInteractive(t *testing.T) {
	var got []int
	ctx, n := renderDebug(t, Rating{Value: 2.5, Halves: true, OnChange: func(v int) { got = append(got, v) }})
	btn := n.Children[3]
	if btn.Style.AccessibilityRole != core.RoleButton || btn.Children[0].Type != "Canvas" {
		t.Fatalf("glyph 4 = role %q holding %q, want a button around a Canvas", btn.Style.AccessibilityRole, btn.Children[0].Type)
	}
	ctx.TriggerCallback(btn.Props["onClick"].(string))
	if len(got) != 1 || got[0] != 4 {
		t.Errorf("changes = %v, want [4]", got)
	}
}

// Without Halves nothing changed: text glyphs, whole rounding.
func TestRatingWithoutHalvesIsUnchanged(t *testing.T) {
	_, n := renderDebug(t, Rating{Value: 3.6, ReadOnly: true})
	if n.Children[0].Type != "Text" || n.Style.AccessibilityValue.Text != "4 of 5" {
		t.Errorf("glyph %q value %q, want text glyphs and 4 of 5", n.Children[0].Type, n.Style.AccessibilityValue.Text)
	}
}
