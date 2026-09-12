package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func threeTabs(taps *[]string, selected int) BottomBar {
	tap := func(s string) func() { return func() { *taps = append(*taps, s) } }
	return BottomBar{
		Items: []BarItem{
			{Icon: "🏠", Label: "Home", OnTap: tap("home")},
			{Icon: "🔍", Label: "Search", OnTap: tap("search")},
			{Label: "Me", AccessibilityLabel: "Profile", OnTap: tap("me")},
		},
		Selected: selected,
	}
}

func TestBottomBarIsANavigationOfEqualCells(t *testing.T) {
	var taps []string
	ctx, n := renderDebug(t, threeTabs(&taps, 1))

	if n.Type != "Row" || n.Style.AccessibilityRole != core.RoleNavigation {
		t.Fatalf("root = %q role %q, want a navigation Row", n.Type, n.Style.AccessibilityRole)
	}
	if len(n.Children) != 3 {
		t.Fatalf("cells = %d", len(n.Children))
	}
	for i, c := range n.Children {
		if c.Style.FlexGrow != 1 || c.Style.AccessibilityRole != core.RoleButton {
			t.Errorf("cell %d: grow=%v role=%q, want equal-width buttons", i, c.Style.FlexGrow, c.Style.AccessibilityRole)
		}
	}
	if got := n.Children[1].Style.AccessibilityLabel; got != "Search, selected" {
		t.Errorf("current label = %q", got)
	}
	if got := n.Children[0].Style.AccessibilityLabel; got != "Home" {
		t.Errorf("other label = %q", got)
	}
	if got := n.Children[2].Style.AccessibilityLabel; got != "Profile" {
		t.Errorf("AccessibilityLabel override = %q", got)
	}
	if icon := findText(n.Children[0], "🏠"); icon == nil || !icon.Style.AccessibilityHidden {
		t.Error("the icon is decoration and hidden from assistive technology")
	}
	cur, other := findText(n.Children[1], "Search"), findText(n.Children[0], "Home")
	if cur.Style.TextColor != core.DefaultTheme.Colors.PrimaryOnLightColor() || cur.Style.FontWeight != core.Bold {
		t.Errorf("current item colour/weight = %q/%v", cur.Style.TextColor, cur.Style.FontWeight)
	}
	if other.Style.TextColor != core.DefaultTheme.Colors.TextSecondary {
		t.Errorf("other item colour = %q", other.Style.TextColor)
	}

	ctx.TriggerCallback(n.Children[2].Props["onClick"].(string))
	if len(taps) != 1 || taps[0] != "me" {
		t.Errorf("taps = %v", taps)
	}
}

func TestBottomBarWithNoSelectionIsAToolbar(t *testing.T) {
	var taps []string
	_, n := renderDebug(t, threeTabs(&taps, -1))
	if n.Style.AccessibilityRole != core.RoleToolbar {
		t.Errorf("role = %q, want toolbar", n.Style.AccessibilityRole)
	}
	for _, c := range n.Children {
		if c.Style.AccessibilityLabel == "Search, selected" || c.Style.AccessibilityLabel == "Home, selected" {
			t.Error("an action strip has no current item")
		}
	}
}

func TestBottomBarStyleOverrides(t *testing.T) {
	_, n := renderDebug(t, BottomBar{
		Items: []BarItem{{Label: "A"}},
		Style: []core.StyleProp{core.BackgroundColor("#101010")},
	})
	if n.Style.Background != "#101010" {
		t.Errorf("background = %q", n.Style.Background)
	}
	if _, ok := n.Children[0].Props["onClick"]; ok {
		t.Error("an item with no OnTap registers no callback")
	}
}
