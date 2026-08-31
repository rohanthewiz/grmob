package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func TestTabsWrapsTabView(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	var changedTo int
	n := Tabs{
		Items:    []core.TabItem{core.Tab("Home", "🏠"), core.Tab("Search", "🔍")},
		Selected: 1,
		OnChange: func(i int) { changedTo = i },
		Content:  []core.View{core.Text("home page"), core.Text("search page")},
	}.Render(ctx)

	if n.Type != "TabView" {
		t.Fatalf("Tabs must delegate to the core TabView node contract, got %q", n.Type)
	}
	if n.Props["selectedIndex"] != 1 {
		t.Errorf("selectedIndex = %v, want 1", n.Props["selectedIndex"])
	}
	tabs, ok := n.Props["tabs"].([]map[string]string)
	if !ok || len(tabs) != 2 || tabs[0]["label"] != "Home" || tabs[1]["icon"] != "🔍" {
		t.Errorf("tabs serialization wrong: %v", n.Props["tabs"])
	}
	if len(n.Children) != 2 || findText(n, "search page") == nil {
		t.Errorf("all content pages must be children per the TabView contract, got %d", len(n.Children))
	}

	id, ok := n.Props["onTabChange"].(string)
	if !ok {
		t.Fatal("OnChange should register an int callback")
	}
	ctx.TriggerIntCallback(id, 0)
	if changedTo != 0 {
		t.Error("tab-change events should reach OnChange")
	}
}

func TestTabsWithoutHandlerOmitsCallback(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := Tabs{
		Items:   []core.TabItem{core.Tab("Only", "")},
		Content: []core.View{core.Text("page")},
	}.Render(ctx)
	if _, present := n.Props["onTabChange"]; present {
		t.Error("a static tab strip must not register a callback (keeps the node diff-stable)")
	}
}
