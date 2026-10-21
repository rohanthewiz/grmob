package comps

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/cart"
	"github.com/rohanthewiz/grmob/core"
)

// promoHarness renders a PromoCodeField over a real cart.Store, the wiring
// the type doc shows, through the rowHarness so the hooks survive passes.
type promoHarness struct {
	*rowHarness
	store *cart.Store
}

func newPromoHarness(t *testing.T) *promoHarness {
	t.Helper()
	h := &promoHarness{store: cart.NewStore(cart.Config{Rules: []cart.Rule{
		cart.PercentOff("SPRING10", "Spring sale", cart.Percent(10)),
	}})}
	h.rowHarness = newRowHarness(t, func() core.View {
		return PromoCodeField{
			OnApply:  h.store.ApplyCode,
			Applied:  h.store.Cart().Codes,
			OnRemove: h.store.RemoveCode,
		}
	})
	return h
}

func (h *promoHarness) input() *core.Node {
	h.t.Helper()
	in := findFirst(h.node, func(n *core.Node) bool { return n.Type == "Input" })
	if in == nil {
		h.t.Fatal("no input")
	}
	return in
}

func (h *promoHarness) typeText(s string) {
	h.t.Helper()
	h.ctx.TriggerTextCallback(h.input().Props["onChange"].(string), s)
	h.render()
}

func (h *promoHarness) alert() string {
	a := findFirst(h.node, func(n *core.Node) bool {
		return n.Style != nil && n.Style.AccessibilityRole == core.RoleAlert
	})
	if a == nil {
		return ""
	}
	return a.Props["content"].(string)
}

func TestPromoCodeFieldAppliesRefusesAndRemoves(t *testing.T) {
	h := newPromoHarness(t)
	if b := findButton(h.node, "Apply"); b == nil || !b.Style.Disabled {
		t.Fatal("Apply is disabled with nothing typed")
	}

	// A refused code: the reason is announced and the draft kept.
	h.typeText("nope")
	h.ctx.TriggerCallback(findButton(h.node, "Apply").Props["onClick"].(string))
	h.render()
	if h.alert() != cart.ErrUnknownCode.Error() || h.input().Props["value"] != "nope" {
		t.Fatalf("alert %q draft %q", h.alert(), h.input().Props["value"])
	}
	// Typing again clears the stale refusal.
	h.typeText("spring10")
	if h.alert() != "" {
		t.Error("typing clears the refusal")
	}

	// Return applies too; the draft clears and the code becomes a pill.
	h.ctx.TriggerCallback(h.input().Props["onSubmit"].(string))
	h.render()
	if !slices.Equal(h.store.Cart().Codes, []string{"SPRING10"}) || h.input().Props["value"] != "" {
		t.Fatalf("codes %v draft %q", h.store.Cart().Codes, h.input().Props["value"])
	}
	if findText(h.node, "SPRING10") == nil {
		t.Fatal("the applied code is drawn")
	}

	tapNamed(t, h.ctx, h.node, "Remove SPRING10")
	h.render()
	if len(h.store.Cart().Codes) != 0 || findText(h.node, "SPRING10") != nil {
		t.Fatal("✕ removes the code")
	}
}

func TestPromoCodeFieldReadOnlyPillsAndDisabled(t *testing.T) {
	_, n := renderDebug(t, PromoCodeField{OnApply: func(string) error { return nil }, Applied: []string{"A"}})
	if findText(n, "A") == nil || findNamed(n, "Remove A") != nil {
		t.Error("without OnRemove the pill has no ✕")
	}
	_, n = renderDebug(t, PromoCodeField{OnApply: func(string) error { return nil }, Disabled: true,
		Applied: []string{"A"}, OnRemove: func(string) {}})
	in := findFirst(n, func(x *core.Node) bool { return x.Type == "Input" })
	if !in.Style.Disabled || !findNamed(n, "Remove A").Style.Disabled {
		t.Error("Disabled disables the input and the ✕")
	}
}

func TestPromoCodeFieldCustomErrorAndLabels(t *testing.T) {
	h := newRowHarness(t, func() core.View {
		return PromoCodeField{
			OnApply:    func(string) error { return errors.New("Minimum order $50") },
			Label:      "Code promo",
			ApplyLabel: "Appliquer",
		}
	})
	in := findFirst(h.node, func(n *core.Node) bool { return n.Type == "Input" })
	if in.Style.AccessibilityLabel != "Code promo" || in.Props["placeholder"] != "Code promo" {
		t.Errorf("label %q placeholder %v", in.Style.AccessibilityLabel, in.Props["placeholder"])
	}
	h.ctx.TriggerTextCallback(in.Props["onChange"].(string), "X")
	h.render()
	h.ctx.TriggerCallback(findButton(h.node, "Appliquer").Props["onClick"].(string))
	h.render()
	if findText(h.node, "Minimum order $50") == nil {
		t.Error("OnApply's error text is drawn")
	}
}

func TestPromoCodeFieldWithoutOnApplyIsReported(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	PromoCodeField{}.Render(ctx)
	if !strings.Contains(core.DumpConcerns(), ConcernPromoCodeFieldInert) {
		t.Fatal("an inert PromoCodeField is reported")
	}
}
