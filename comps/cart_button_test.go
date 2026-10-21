package comps

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func TestCartButtonIsOneNamedButtonWithACount(t *testing.T) {
	taps := 0
	ctx, n := renderDebug(t, CartButton{Count: 3, OnTap: func() { taps++ }})
	if n.Style.AccessibilityRole != core.RoleButton || n.Style.AccessibilityLabel != "Cart, 3 items" {
		t.Fatalf("role %q label %q", n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
	if findText(n, "3") == nil {
		t.Error("the badge draws the count")
	}
	ctx.TriggerCallback(n.Props["onClick"].(string))
	if taps != 1 {
		t.Error("a tap opens the cart")
	}
}

func TestCartButtonEmptyAndCappedCounts(t *testing.T) {
	_, n := renderDebug(t, CartButton{OnTap: func() {}})
	if findText(n, "0") != nil || n.Style.AccessibilityLabel != "Cart, 0 items" {
		t.Error("an empty cart draws no badge, and still says its count")
	}
	_, n = renderDebug(t, CartButton{Count: 1, OnTap: func() {}})
	if n.Style.AccessibilityLabel != "Cart, 1 item" {
		t.Errorf("singular: %q", n.Style.AccessibilityLabel)
	}
	_, n = renderDebug(t, CartButton{Count: 120, OnTap: func() {}})
	if findText(n, "99+") == nil || n.Style.AccessibilityLabel != "Cart, 120 items" {
		t.Error("the badge caps at 99+; the name says the real count")
	}
	_, n = renderDebug(t, CartButton{Count: 12, MaxCount: 9, Label: "Panier",
		CountLabel: func(n int) string { return "12 articles" }, OnTap: func() {}})
	if findText(n, "9+") == nil || n.Style.AccessibilityLabel != "Panier, 12 articles" {
		t.Error("MaxCount, Label and CountLabel are configurable")
	}
}

func TestCartButtonWithoutOnTapIsReported(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := CartButton{Count: 1}.Render(ctx)
	if !strings.Contains(core.DumpConcerns(), ConcernCartButtonInert) {
		t.Fatal("an inert CartButton is reported")
	}
	// Still a registered no-op, so a native tap cannot hit a nil handler.
	ctx.TriggerCallback(n.Props["onClick"].(string))
}
