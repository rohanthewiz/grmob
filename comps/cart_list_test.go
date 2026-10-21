package comps

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/cart"
	"github.com/rohanthewiz/grmob/core"
)

// The list wired to a real cart.Store the way the docs show: method values
// as callbacks, the store's cart re-read each pass.
func TestCartListDrivesAStore(t *testing.T) {
	store := cart.NewStore(cart.Config{MaxQuantity: 5})
	store.Add(cart.Item{ID: "kettle", Label: "Kettle", UnitAmount: 4500})
	store.Add(cart.Item{ID: "filters", Label: "Filters", UnitAmount: 650, Quantity: 2})

	h := newRowHarness(t, func() core.View {
		q := store.Quote()
		return CartList{Items: q.Cart.Items, Currency: q.Currency,
			OnQuantity: store.SetQuantity, OnRemove: store.Remove}
	})

	// The second line's "+" (the steppers are named per item).
	plus := findFirst(findNamed(h.node, "Quantity, Filters"), func(x *core.Node) bool {
		return x.Style != nil && x.Style.AccessibilityLabel == "Increase"
	})
	h.ctx.TriggerCallback(plus.Props["onClick"].(string))
	h.render()
	if it, _ := store.Cart().Find("filters"); it.Quantity != 3 {
		t.Fatalf("filters qty %d, want 3", it.Quantity)
	}

	tapNamed(t, h.ctx, h.node, "Remove Kettle")
	h.render()
	if store.Cart().Count() != 3 || findNamed(h.node, "Remove Kettle") != nil {
		t.Fatalf("kettle removed from store and list: %+v", store.Cart())
	}
}

func TestCartListKeysLinesAndSeparatorsByID(t *testing.T) {
	items := []cart.Item{{ID: "a", Label: "A", Quantity: 1}, {ID: "b", Label: "B", Quantity: 1}}
	_, n := renderDebug(t, CartList{Items: items})
	var keys []string
	for _, c := range n.Children {
		keys = append(keys, c.Key)
	}
	if got := strings.Join(keys, " "); got != "line:a sep:b line:b" {
		t.Fatalf("keys %q", got)
	}
}

func TestCartListEmptyStateAndSlot(t *testing.T) {
	_, n := renderDebug(t, CartList{})
	if findText(n, "Your cart is empty") == nil {
		t.Fatal("zero value draws the empty state")
	}
	_, n = renderDebug(t, CartList{EmptyTitle: "Nothing here", EmptyHint: "Browse the shop"})
	if findText(n, "Nothing here") == nil || findText(n, "Browse the shop") == nil {
		t.Error("empty state lines are configurable")
	}
	_, n = renderDebug(t, CartList{Empty: core.Text("custom")})
	if findText(n, "custom") == nil {
		t.Error("the Empty slot replaces it")
	}
}

func TestCartListReadOnlyAndLineSlot(t *testing.T) {
	items := []cart.Item{{ID: "a", Label: "A", Quantity: 2}}
	_, n := renderDebug(t, CartList{Items: items, ReadOnly: true,
		OnQuantity: func(string, int) {}, OnRemove: func(string) {}})
	if findFirst(n, func(x *core.Node) bool { return x.Type == "Button" }) != nil {
		t.Error("ReadOnly draws no controls")
	}
	_, n = renderDebug(t, CartList{Items: items, Line: func(it cart.Item) core.View {
		return core.Text("row " + it.ID)
	}})
	if findText(n, "row a") == nil || n.Children[0].Key != "line:a" {
		t.Error("Line replaces the CartLine and keeps the key")
	}
}

func TestCartListReportsDuplicateIDs(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	CartList{Items: []cart.Item{{ID: "a", Label: "A"}, {ID: "a", Label: "A again"}}}.Render(ctx)
	if !strings.Contains(core.DumpConcerns(), ConcernCartListDuplicateID) {
		t.Fatal("duplicate IDs are reported")
	}
}
