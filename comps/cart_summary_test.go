package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/cart"
	"github.com/rohanthewiz/grmob/core"
)

// summaryQuote is the type doc's example: $58.00 of goods, 10% off, free
// shipping over $50, 8.25% tax.
func summaryQuote() cart.Quote {
	cfg := cart.Config{Rules: []cart.Rule{
		cart.PercentOff("SPRING10", "Spring sale", cart.Percent(10)),
		cart.FlatShipping("Shipping", 599, 5000),
		cart.Tax("Tax", cart.Percent(8.25)),
	}}
	c := cfg.Reduce(cart.Cart{}, cart.Add{Item: cart.Item{ID: "k", Label: "Kettle", UnitAmount: 4500}})
	c = cfg.Reduce(c, cart.Add{Item: cart.Item{ID: "f", Label: "Filters", UnitAmount: 650, Quantity: 2}})
	c = cfg.Reduce(c, cart.ApplyCode{Code: "spring10"})
	return cfg.Price(c)
}

func TestCartSummaryDrawsTheQuote(t *testing.T) {
	_, n := renderDebug(t, CartSummary{Quote: summaryQuote()})
	for _, name := range []string{
		"Subtotal, 3 items, $58.00",
		"Spring sale, SPRING10, -$5.80",
		"Shipping, Free",
		"Tax, $4.31",
		"Total, $56.51",
	} {
		if findNamed(n, name) == nil {
			t.Errorf("no row named %q", name)
		}
	}
	if n.Type != "Card" {
		t.Errorf("root is %s, want a Card like StripeCheckout", n.Type)
	}
}

func TestCartSummaryLabelsAndSlots(t *testing.T) {
	_, n := renderDebug(t, CartSummary{
		Quote:         summaryQuote(),
		Title:         "Résumé",
		SubtotalLabel: "Sous-total",
		TotalLabel:    "Total TTC",
		FreeLabel:     "Gratuit",
		CountLabel:    func(n int) string { return "articles" },
		Footer:        core.Text("footer"),
	})
	for _, name := range []string{"Sous-total, articles, $58.00", "Shipping, Gratuit", "Total TTC, $56.51"} {
		if findNamed(n, name) == nil {
			t.Errorf("no row named %q", name)
		}
	}
	title := findText(n, "Résumé")
	if title == nil || title.Style.AccessibilityRole != core.RoleHeading {
		t.Error("Title is a heading")
	}
	if findText(n, "footer") == nil {
		t.Error("Footer is drawn")
	}

	_, n = renderDebug(t, CartSummary{Quote: summaryQuote(), HideCount: true})
	if findNamed(n, "Subtotal, $58.00") == nil {
		t.Error("HideCount drops the count")
	}
}

func TestCartSummaryZeroValue(t *testing.T) {
	_, n := renderDebug(t, CartSummary{})
	if findNamed(n, "Total, $0.00") == nil {
		t.Fatal("the zero summary is a $0 USD total")
	}
}

func TestStripeCheckoutFromCartAgreesWithTheQuote(t *testing.T) {
	q := summaryQuote()
	co := StripeCheckoutFromCart(q)
	if co.Total() != q.Total {
		t.Fatalf("StripeCheckout total %d, quote %d", co.Total(), q.Total)
	}
	if co.Currency != "USD" || len(co.Items) != 2 || co.Items[1].Quantity != 2 || co.Disabled {
		t.Fatalf("got %+v", co)
	}
	if !StripeCheckoutFromCart(cart.Quote{}).Disabled {
		t.Error("an empty cart's checkout is disabled")
	}
}
