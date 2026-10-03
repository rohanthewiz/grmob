package comps

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func TestFormatMoney(t *testing.T) {
	cases := []struct {
		amount   int64
		currency string
		want     string
	}{
		{4500, "usd", "$45.00"},
		{123456789, "USD", "$1,234,567.89"},
		{5, "usd", "$0.05"},
		{0, "usd", "$0.00"},
		{-500, "eur", "-€5.00"},
		{4500, "jpy", "¥4,500"}, // zero-decimal: no minor unit
		{1500, "kwd", "KWD 1.500"},
		{9900, "sek", "SEK 99.00"},
		{100000, "gbp", "£1,000.00"},
	}
	for _, c := range cases {
		if got := FormatMoney(c.amount, c.currency); got != c.want {
			t.Errorf("FormatMoney(%d, %q) = %q, want %q", c.amount, c.currency, got, c.want)
		}
	}
}

var cart = []CheckoutItem{
	{Label: "Pour-over kettle", Detail: "Matte black", UnitAmount: 4500},
	{Label: "Filters (100)", Quantity: 2, UnitAmount: 650},
}

// The summary: a heading, one named row per line, the total, and a Pay
// button that says the total.
func TestStripeCheckoutSummary(t *testing.T) {
	s := StripeCheckout{
		Items:       cart,
		Adjustments: []CheckoutAdjustment{{Label: "Shipping", Amount: 500}, {Label: "Discount", Amount: -1000}},
		CheckoutURL: "https://buy.stripe.com/test_123",
	}
	if s.Total() != 4500+1300+500-1000 {
		t.Errorf("Total = %d", s.Total())
	}
	_, n := renderDebug(t, s)

	if h := findText(n, "Order summary"); h == nil || h.Style.AccessibilityRole != core.RoleHeading {
		t.Error("the default title should be a heading")
	}
	for _, phrase := range []string{
		"Pour-over kettle, Matte black, $45.00",
		"Filters (100), 2 × $6.50, $13.00", // quantity drawn as the detail
		"Shipping, $5.00",
		"Discount, -$10.00",
		"Total, $53.00",
	} {
		if findFirst(n, func(x *core.Node) bool { return x.Style.AccessibilityLabel == phrase }) == nil {
			t.Errorf("no row named %q", phrase)
		}
	}
	btns := buttonsOf(n)
	if len(btns) != 1 || btns[0].Props["label"] != "Pay $53.00" {
		t.Fatalf("buttons = %v", btns)
	}
	if btns[0].Style.AccessibilityHint != "Opens Stripe Checkout in your browser" {
		t.Errorf("hint = %q", btns[0].Style.AccessibilityHint)
	}
	if findText(n, "Payments are processed securely by Stripe.") == nil {
		t.Error("the default note is missing")
	}
}

// OnPay wins over CheckoutURL and is called once per tap.
func TestStripeCheckoutOnPayWins(t *testing.T) {
	paid := 0
	ctx, n := renderDebug(t, StripeCheckout{Items: cart, CheckoutURL: "https://buy.stripe.com/x", OnPay: func() { paid++ }})
	btn := buttonsOf(n)[0]
	if btn.Style.AccessibilityHint != "" {
		t.Error("OnPay's destination is the app's; the browser hint should not be claimed")
	}
	ctx.TriggerCallback(btn.Props["onClick"].(string))
	if paid != 1 {
		t.Errorf("OnPay called %d times", paid)
	}
}

// Pending relabels and disables the button; a tap that races the patch does
// nothing.
func TestStripeCheckoutPending(t *testing.T) {
	paid := 0
	ctx, n := renderDebug(t, StripeCheckout{Items: cart, OnPay: func() { paid++ }, Pending: true})
	btn := buttonsOf(n)[0]
	if btn.Props["label"] != "Redirecting to Stripe…" || !btn.Style.Disabled {
		t.Errorf("label %q disabled %v", btn.Props["label"], btn.Style.Disabled)
	}
	ctx.TriggerCallback(btn.Props["onClick"].(string))
	if paid != 0 {
		t.Error("a pending checkout must not pay twice")
	}
}

// Error is an alert above the button; HideNote drops the note; the labels
// and title are replaceable.
func TestStripeCheckoutOptions(t *testing.T) {
	_, n := renderDebug(t, StripeCheckout{
		Title: "Your cart", Items: cart, OnPay: func() {}, Error: "Your card was declined.",
		PayLabel: "Check out", HideNote: true, Currency: "jpy",
	})
	if msg := findText(n, "Your card was declined."); msg == nil || msg.Style.AccessibilityRole != core.RoleAlert {
		t.Error("the error should be an alert")
	}
	if findText(n, "Your cart") == nil {
		t.Error("Title should replace the heading")
	}
	if buttonsOf(n)[0].Props["label"] != "Check out" {
		t.Error("PayLabel should replace the button label")
	}
	if findFirst(n, func(x *core.Node) bool {
		return x.Type == "Text" && strings.Contains(x.Props["content"].(string), "Stripe")
	}) != nil {
		t.Error("HideNote should leave the note out")
	}
	if findFirst(n, func(x *core.Node) bool { return x.Style.AccessibilityLabel == "Total, ¥5,800" }) == nil {
		t.Error("yen amounts should have no decimals")
	}
}

// Nowhere to send the reader is a concern; so is a non-https URL. Disabled
// silences the first: a disabled button is meant to do nothing.
func TestStripeCheckoutConcerns(t *testing.T) {
	cases := []struct {
		s    StripeCheckout
		want string
	}{
		{StripeCheckout{Items: cart}, ConcernStripeCheckoutInert},
		{StripeCheckout{Items: cart, CheckoutURL: "http://buy.stripe.com/x"}, ConcernStripeCheckoutInsecureURL},
		{StripeCheckout{Items: cart, Disabled: true}, ""},
	}
	for _, c := range cases {
		core.SetDebugMode(true)
		core.ClearConcerns()
		ctx := core.NewContext()
		ctx.BeginRenderPass()
		c.s.Render(ctx)
		ctx.EndRenderPass()
		dump := core.DumpConcerns()
		if c.want == "" && dump != "" || c.want != "" && !strings.Contains(dump, c.want) {
			t.Errorf("%+v: concerns %q, want %q", c.s, dump, c.want)
		}
		core.SetDebugMode(false)
		core.ClearConcerns()
	}
}

func TestStripeCheckoutCallerStyleWins(t *testing.T) {
	_, n := renderDebug(t, StripeCheckout{Items: cart, OnPay: func() {}, Style: []core.StyleProp{core.Gap(29)}})
	if n.Style.Gap != 29 {
		t.Errorf("gap = %v, want the caller's 29", n.Style.Gap)
	}
}

func TestStripeCheckoutEveryTheme(t *testing.T) {
	for name, th := range core.BundledThemes() {
		core.SetDebugMode(true)
		core.ClearConcerns()
		ctx := core.NewContext().WithTheme(th)
		ctx.BeginRenderPass()
		n := StripeCheckout{Items: cart, OnPay: func() {}, Error: "x"}.Render(ctx)
		ctx.EndRenderPass()
		core.AuditTree(n)
		if dump := core.DumpConcerns(); dump != "" {
			t.Errorf("%s: %s", name, dump)
		}
		core.SetDebugMode(false)
		core.ClearConcerns()
	}
}
