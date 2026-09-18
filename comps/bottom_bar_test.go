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
	// The current cell says so as a state, and its name is the label alone.
	if got := n.Children[1].Style.AccessibilityLabel; got != "Search" {
		t.Errorf("current label = %q, want the label with no suffix", got)
	}
	if got := n.Children[1].Style.AccessibilityCurrent; got != core.CurrentPage {
		t.Errorf("current cell AccessibilityCurrent = %q, want page", got)
	}
	for _, i := range []int{0, 2} {
		if got := n.Children[i].Style.AccessibilityCurrent; got != core.CurrentNone {
			t.Errorf("cell %d AccessibilityCurrent = %q, want none", i, got)
		}
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
		if c.Style.AccessibilityCurrent != core.CurrentNone {
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

// A badged item's icon becomes a two-layer stack with the Badge at its
// top-end corner; the count joins the cell's name and is hidden where it is
// drawn. An unbadged item keeps the bare icon Text it always had.
func TestBottomBarBadgeSitsOverTheIcon(t *testing.T) {
	_, n := renderDebug(t, BottomBar{Items: []BarItem{
		{Icon: "🏠", Label: "Home"},
		{Icon: "✉", Label: "Inbox", Badge: "3", BadgeLabel: "3 unread"},
		{Label: "Me", Badge: "!"},
	}})

	if icon := n.Children[0].Children[0]; icon.Type != "Text" || icon.Style.Margin.Left != 0 {
		t.Errorf("unbadged icon = %q with margin %d, want the bare Text", icon.Type, icon.Style.Margin.Left)
	}

	inbox := n.Children[1]
	if got := inbox.Style.AccessibilityLabel; got != "Inbox, 3 unread" {
		t.Errorf("badged name = %q, want the label and the spoken badge", got)
	}
	stack := inbox.Children[0]
	if stack.Type != "ZStack" || len(stack.Children) != 2 {
		t.Fatalf("badged icon = %q with %d layers, want a two-layer ZStack", stack.Type, len(stack.Children))
	}
	glyph, badge := stack.Children[0], stack.Children[1]
	if glyph.Props["content"] != "✉" || glyph.Style.Margin.Left == 0 || glyph.Style.Margin.Left != glyph.Style.Margin.Right {
		t.Errorf("glyph margin = %d/%d, want a symmetric horizontal margin", glyph.Style.Margin.Left, glyph.Style.Margin.Right)
	}
	if glyph.Style.Margin.Top != 0 {
		t.Error("a top margin would drop a badged icon below its neighbours")
	}
	if badge.Props["content"] != "3" || badge.Style.StackAlign != core.StackAlignTopEnd || !badge.Style.AccessibilityHidden {
		t.Errorf("badge = %v placed %q hidden %v", badge.Props["content"], badge.Style.StackAlign, badge.Style.AccessibilityHidden)
	}
	if badge.Style.Background != VariantError.Color(core.DefaultTheme) {
		t.Errorf("badge fill = %q, want the Error role", badge.Style.Background)
	}

	// No icon: the label wears it, and BadgeLabel falls back to Badge.
	me := n.Children[2]
	if got := me.Style.AccessibilityLabel; got != "Me, !" {
		t.Errorf("name = %q", got)
	}
	if me.Children[0].Type != "ZStack" || findText(me.Children[0], "Me") == nil {
		t.Error("with no icon the label should carry the badge")
	}
}
