package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func TestFABIsARaisedDiscButton(t *testing.T) {
	tapped := 0
	ctx, n := renderDebug(t, FAB{Icon: "+", AccessibilityLabel: "Add", OnTap: func() { tapped++ }})
	defer ctx.Close()

	if n.Type != "Button" {
		t.Fatalf("root = %q, want a Button: the FAB is comps.Button in a circle", n.Type)
	}
	if n.Style.Width != "56px" || n.Style.Height != "56px" || n.Style.BorderRadius != 28 {
		t.Errorf("disc = %s×%s radius %v, want 56px×56px radius 28", n.Style.Width, n.Style.Height, n.Style.BorderRadius)
	}
	if n.Style.Shadow != fabElevation || fabElevation <= 2 {
		t.Errorf("shadow = %v, want %v and deeper than a Card's 2: the disc must float", n.Style.Shadow, fabElevation)
	}
	if n.Style.Padding != (core.EdgeInsets{}) {
		t.Errorf("padding = %+v, want none so the glyph centres in a fixed-width disc", n.Style.Padding)
	}
	if n.Style.AccessibilityLabel != "Add" {
		t.Errorf("label = %q, want the caller's name: a glyph is not one", n.Style.AccessibilityLabel)
	}
	if n.Props["label"] != "+" {
		t.Errorf("visible label = %v, want the glyph alone", n.Props["label"])
	}
	id, ok := n.Props["onClick"].(string)
	if !ok {
		t.Fatal("the FAB registers a tap")
	}
	ctx.TriggerCallback(id)
	if tapped != 1 {
		t.Errorf("tapped %d times, want exactly one", tapped)
	}
}

func TestFABExtendedIsAPillNamedByItsLabel(t *testing.T) {
	ctx, n := renderDebug(t, FAB{Icon: "✎", Label: "Compose", OnTap: func() {}})
	defer ctx.Close()

	if n.Style.Width != "" {
		t.Errorf("width = %s, want none: the pill takes its width from the word", n.Style.Width)
	}
	if n.Style.Height != "56px" || n.Style.BorderRadius != 28 {
		t.Errorf("pill = height %s radius %v, want the disc's height so a label does not move the button", n.Style.Height, n.Style.BorderRadius)
	}
	md := core.DefaultTheme.Spacing.MD
	if p := n.Style.Padding; p.Left != md || p.Right != md || p.Top != 0 || p.Bottom != 0 {
		t.Errorf("padding = %+v, want %d on the sides only", p, md)
	}
	if n.Props["label"] != "✎  Compose" {
		t.Errorf("visible label = %v, want glyph then word", n.Props["label"])
	}
	if n.Style.AccessibilityLabel != "" {
		t.Error("the extended form is named by its label like any button; no override was asked for")
	}
	ctx2, btn := renderDebug(t, Button{Label: "Compose"})
	defer ctx2.Close()
	if n.Style.FontSize != btn.Style.FontSize {
		t.Errorf("font size = %v, want a Button's %v: the glyph ratio is for the disc alone", n.Style.FontSize, btn.Style.FontSize)
	}
}

func TestFABSmallAndGlyphScale(t *testing.T) {
	ctx, n := renderDebug(t, FAB{Icon: "+", Size: FABSmall, AccessibilityLabel: "Add"})
	defer ctx.Close()
	if n.Style.Width != "40px" || n.Style.Height != "40px" || n.Style.BorderRadius != 20 {
		t.Errorf("small = %s×%s radius %v, want 40px", n.Style.Width, n.Style.Height, n.Style.BorderRadius)
	}
	if want := 40 * fabGlyphRatio; n.Style.FontSize != want {
		t.Errorf("glyph = %v, want %v: the glyph scales with the disc", n.Style.FontSize, want)
	}
}

func TestFABDisabledAndStylePassThroughButton(t *testing.T) {
	tapped := false
	ctx, n := renderDebug(t, FAB{
		Icon: "+", AccessibilityLabel: "Add", Disabled: true,
		OnTap: func() { tapped = true },
		Style: []core.StyleProp{core.Shadow(0)},
	})
	defer ctx.Close()
	if !n.Style.Disabled {
		t.Error("Disabled reaches the button")
	}
	if n.Style.Shadow != 0 {
		t.Error("the caller's Style lands after the widget's own, so the shadow can be removed")
	}
	if n.Style.Background != core.DefaultTheme.Colors.Surface {
		t.Error("a disabled FAB takes Button's muted treatment")
	}
	ctx.TriggerCallback(n.Props["onClick"].(string))
	if tapped {
		t.Error("a disabled FAB drops the tap")
	}
}

func TestFABZeroVariantIsTheThemesButtonPairing(t *testing.T) {
	ctx, fab := renderDebug(t, FAB{Icon: "+", AccessibilityLabel: "Add"})
	defer ctx.Close()
	ctx2, btn := renderDebug(t, Button{Label: "+"})
	defer ctx2.Close()
	if fab.Style.Background != btn.Style.Background || fab.Style.TextColor != btn.Style.TextColor {
		t.Errorf("FAB fill/ink = %s/%s, Button = %s/%s: the two must agree about primary",
			fab.Style.Background, fab.Style.TextColor, btn.Style.Background, btn.Style.TextColor)
	}
}
