package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/cart"
	"github.com/rohanthewiz/grmob/core"
)

// findNamed finds the first node whose spoken name is name.
func findNamed(n *core.Node, name string) *core.Node {
	return findFirst(n, func(x *core.Node) bool {
		return x.Style != nil && x.Style.AccessibilityLabel == name
	})
}

// tapNamed fires the onClick of the node named name, failing if there is none.
func tapNamed(t *testing.T, ctx *core.Context, n *core.Node, name string) {
	t.Helper()
	b := findNamed(n, name)
	if b == nil {
		t.Fatalf("no node named %q", name)
	}
	ctx.TriggerCallback(b.Props["onClick"].(string))
}

var kettleLine = cart.Item{ID: "kettle", Label: "Pour-over kettle", Detail: "Matte black", UnitAmount: 4500, Quantity: 2, MaxQuantity: 3}

func TestCartLineNamesTheItemAsOnePhrase(t *testing.T) {
	_, n := renderDebug(t, CartLine{Item: kettleLine})
	if findNamed(n, "Pour-over kettle, Matte black, 2 × $45.00, $90.00") == nil {
		t.Fatal("the item's text is one labelled phrase")
	}
	// Zero value: read-only, no controls to be inert.
	if findFirst(n, func(x *core.Node) bool { return x.Type == "Button" }) != nil {
		t.Error("a CartLine without callbacks draws no buttons")
	}
}

func TestCartLineStepperIsBoundedByTheItemsCapAndNamedForIt(t *testing.T) {
	var got []int
	line := CartLine{Item: kettleLine, OnQuantity: func(n int) { got = append(got, n) }}
	ctx, n := renderDebug(t, line)

	group := findNamed(n, "Quantity, Pour-over kettle")
	if group == nil || group.Style.AccessibilityRole != core.RoleGroup {
		t.Fatal("the Stepper is named after the item")
	}
	tapNamed(t, ctx, n, "Increase")
	tapNamed(t, ctx, n, "Decrease")
	if len(got) != 2 || got[0] != 3 || got[1] != 1 {
		t.Fatalf("OnQuantity got %v, want [3 1]", got)
	}

	// At the cap the "+" is disabled; at one the "−" is.
	_, n = renderDebug(t, CartLine{Item: cart.Item{ID: "x", Label: "X", Quantity: 3, MaxQuantity: 3}, OnQuantity: func(int) {}})
	if b := findNamed(n, "Increase"); b == nil || !b.Style.Disabled {
		t.Error("+ is disabled at the cap")
	}
	_, n = renderDebug(t, CartLine{Item: cart.Item{ID: "x", Label: "X", Quantity: 1}, OnQuantity: func(int) {}})
	if b := findNamed(n, "Decrease"); b == nil || !b.Style.Disabled {
		t.Error("− stops at one: removal is the Remove button's")
	}
}

func TestCartLineCappedAtOneHasNoStepper(t *testing.T) {
	_, n := renderDebug(t, CartLine{Item: cart.Item{ID: "x", Label: "Gift card", Quantity: 1, MaxQuantity: 1}, OnQuantity: func(int) {}})
	if findNamed(n, "Increase") != nil {
		t.Fatal("a line capped at one has nothing to step")
	}
}

func TestCartLineRemoveIsNamedForTheItemAndSwipeIsOptIn(t *testing.T) {
	removed := 0
	line := CartLine{Item: kettleLine, OnRemove: func() { removed++ }}
	ctx, n := renderDebug(t, line)
	tapNamed(t, ctx, n, "Remove Pour-over kettle")
	if removed != 1 {
		t.Fatalf("removed %d", removed)
	}
	if _, ok := n.Props["onSwipe"]; ok {
		t.Error("swipe to remove is off by default")
	}

	line.SwipeToRemove = true
	ctx, n = renderDebug(t, line)
	id, ok := n.Props["onSwipe"].(string)
	if !ok {
		t.Fatal("SwipeToRemove puts a swipe on the row")
	}
	ctx.TriggerTextCallback(id, string(core.SwipeTrailing))
	if removed != 1 {
		t.Error("a trailing swipe does not remove")
	}
	ctx.TriggerTextCallback(id, string(core.SwipeLeading))
	if removed != 2 {
		t.Error("a leading swipe removes")
	}
}

func TestCartLineDisabledDisablesItsControls(t *testing.T) {
	_, n := renderDebug(t, CartLine{Item: kettleLine, OnQuantity: func(int) {}, OnRemove: func() {}, Disabled: true, SwipeToRemove: true})
	for _, name := range []string{"Increase", "Decrease", "Remove Pour-over kettle"} {
		if b := findNamed(n, name); b == nil || !b.Style.Disabled {
			t.Errorf("%q is disabled", name)
		}
	}
	if _, ok := n.Props["onSwipe"]; ok {
		t.Error("a disabled line takes no swipe")
	}
}

func TestCartLineThumbnailAndLeadingSlot(t *testing.T) {
	it := kettleLine
	it.ImageURL = "https://example.com/k.png"
	_, n := renderDebug(t, CartLine{Item: it})
	img := findFirst(n, func(x *core.Node) bool { return x.Type == "Image" })
	if img == nil || !img.Style.AccessibilityHidden {
		t.Fatal("the thumbnail is drawn and hidden from readers")
	}
	_, n = renderDebug(t, CartLine{Item: it, HideImage: true})
	if findFirst(n, func(x *core.Node) bool { return x.Type == "Image" }) != nil {
		t.Error("HideImage leaves it out")
	}
	_, n = renderDebug(t, CartLine{Item: it, Leading: core.Text("☕")})
	if findText(n, "☕") == nil || findFirst(n, func(x *core.Node) bool { return x.Type == "Image" }) != nil {
		t.Error("the Leading slot wins over the image")
	}
}

func TestCartLineFormatAndCurrency(t *testing.T) {
	_, n := renderDebug(t, CartLine{Item: cart.Item{ID: "x", Label: "Tea", UnitAmount: 450, Quantity: 1}, Currency: "jpy"})
	if findText(n, "¥450") == nil {
		t.Error("yen has no minor unit")
	}
	_, n = renderDebug(t, CartLine{
		Item:   cart.Item{ID: "x", Label: "Tea", UnitAmount: 450, Quantity: 1},
		Format: func(a int64, c string) string { return c + "!" },
	})
	if findText(n, "USD!") == nil {
		t.Error("Format is used, with the defaulted, upper-cased currency")
	}
}

func TestCartLineCallerStyleWins(t *testing.T) {
	_, n := renderDebug(t, CartLine{Item: kettleLine, Style: []core.StyleProp{core.Gap(31)}})
	if n.Style.Gap != 31 {
		t.Fatalf("gap %v, want the caller's 31", n.Style.Gap)
	}
}
