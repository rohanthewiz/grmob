package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func tickerQuotes() []StockQuote {
	return []StockQuote{
		{Symbol: "AAPL", Name: "Apple", Price: 189.12, Change: 1.23},
		{Symbol: "MSFT", Name: "Microsoft", Price: 411.65, Change: -2.10},
		{Symbol: "TSLA", Price: 242.00},
	}
}

// quotesOf returns the tape's entries in drawn order.
func quotesOf(n *core.Node) []*core.Node {
	var out []*core.Node
	for _, c := range n.Children {
		if r := c.Style.AccessibilityRole; r == core.RoleListItem || r == core.RoleButton {
			out = append(out, c)
		}
	}
	return out
}

func TestStockTickerStructureAndAccessibility(t *testing.T) {
	_, n := renderDebug(t, StockTicker{Quotes: tickerQuotes()})

	if n.Type != "Scroll" {
		t.Fatalf("root = %s, want a sideways Scroll", n.Type)
	}
	if n.Style.AccessibilityRole != core.RoleList {
		t.Errorf("root role = %q, want list", n.Style.AccessibilityRole)
	}
	if n.Style.AccessibilityLabel != "Stock prices" {
		t.Errorf("root label = %q, want the default", n.Style.AccessibilityLabel)
	}

	qs := quotesOf(n)
	if len(qs) != 3 {
		t.Fatalf("quotes = %d, want 3", len(qs))
	}
	want := []string{
		"Apple, AAPL, 189.12, up 1.23, 0.65 percent",
		"Microsoft, MSFT, 411.65, down 2.10, 0.51 percent",
		"TSLA, 242.00, unchanged",
	}
	for i, q := range qs {
		if q.Style.AccessibilityLabel != want[i] {
			t.Errorf("quote %d spoken = %q, want %q", i, q.Style.AccessibilityLabel, want[i])
		}
		if q.Props["onClick"] != nil {
			t.Errorf("quote %d registered a tap with no OnTap", i)
		}
	}

	// Drawn figures are magnitudes behind a glyph, and hidden from readers
	// so the glyph is never announced.
	for _, s := range []string{"▲ 1.23 (0.65%)", "▼ 2.10 (0.51%)", "0.00 (0.00%)"} {
		txt := findText(n, s)
		if txt == nil {
			t.Errorf("no movement drawn as %q", s)
			continue
		}
		if !txt.Style.AccessibilityHidden {
			t.Errorf("%q is not hidden from readers", s)
		}
	}
}

func TestStockTickerDirectionInk(t *testing.T) {
	for name, th := range core.BundledThemes() {
		ctx := core.NewContext().WithTheme(th)
		ctx.BeginRenderPass()

		for _, redRises := range []bool{false, true} {
			n := StockTicker{Quotes: tickerQuotes(), RedRises: redRises}.Render(ctx)
			rise, fall := VariantSuccess.AsInk(th), VariantError.AsInk(th)
			if redRises {
				rise, fall = fall, rise
			}
			if got := findText(n, "▲ 1.23 (0.65%)").Style.TextColor; got != rise {
				t.Errorf("%s redRises=%v: rise ink = %q, want %q", name, redRises, got, rise)
			}
			if got := findText(n, "▼ 2.10 (0.51%)").Style.TextColor; got != fall {
				t.Errorf("%s redRises=%v: fall ink = %q, want %q", name, redRises, got, fall)
			}
			if got := findText(n, "0.00 (0.00%)").Style.TextColor; got != th.Colors.TextSecondary {
				t.Errorf("%s redRises=%v: flat ink = %q, want the secondary ink", name, redRises, got)
			}
		}
	}
}

func TestStockTickerStartWrapsAndKeepsKeys(t *testing.T) {
	symbols := func(n *core.Node) []string {
		var out []string
		for _, q := range quotesOf(n) {
			out = append(out, q.Children[0].Props["content"].(string))
		}
		return out
	}
	cases := []struct {
		start int
		want  []string
	}{
		{0, []string{"AAPL", "MSFT", "TSLA"}},
		{1, []string{"MSFT", "TSLA", "AAPL"}},
		{4, []string{"MSFT", "TSLA", "AAPL"}},
		{-1, []string{"TSLA", "AAPL", "MSFT"}},
	}
	for _, c := range cases {
		_, n := renderDebug(t, StockTicker{Quotes: tickerQuotes(), Start: c.start})
		got := symbols(n)
		for i := range c.want {
			if got[i] != c.want[i] {
				t.Errorf("Start %d: order = %v, want %v", c.start, got, c.want)
				break
			}
		}
	}
}

func TestStockTickerDuplicateSymbolsAreDistinct(t *testing.T) {
	// renderDebug runs the whole-tree audit, which reports duplicate ids.
	_, n := renderDebug(t, StockTicker{Quotes: []StockQuote{
		{Symbol: "BRK", Price: 1, Change: 0.1},
		{Symbol: "BRK", Price: 2, Change: 0.2},
	}, OnTap: func(StockQuote) {}})
	qs := quotesOf(n)
	if len(qs) != 2 {
		t.Fatalf("quotes = %d, want 2", len(qs))
	}
	if qs[0].Props["onClick"] == qs[1].Props["onClick"] {
		t.Error("two quotes with one symbol share a callback id")
	}
}

func TestStockTickerOnTapReportsTheQuote(t *testing.T) {
	var got StockQuote
	ctx, n := renderDebug(t, StockTicker{Quotes: tickerQuotes(), OnTap: func(q StockQuote) { got = q }})

	if n.Style.AccessibilityRole != core.RoleGroup {
		t.Errorf("root role = %q, want group when the quotes are buttons", n.Style.AccessibilityRole)
	}
	qs := quotesOf(n)
	if qs[1].Style.AccessibilityRole != core.RoleButton {
		t.Fatalf("quote role = %q, want button", qs[1].Style.AccessibilityRole)
	}
	ctx.TriggerCallback(qs[1].Props["onClick"].(string))
	if got.Symbol != "MSFT" {
		t.Errorf("OnTap got %q, want MSFT", got.Symbol)
	}
}

func TestStockTickerFormatPercentAndLabels(t *testing.T) {
	_, n := renderDebug(t, StockTicker{
		Quotes:      []StockQuote{{Symbol: "SAP", Price: 110, Change: -10}},
		Format:      func(v float64) string { return "€" + formatTwo(v) },
		HidePercent: true,
		Label:       "Kurse",
		DownLabel:   "runter",
	})
	if n.Style.AccessibilityLabel != "Kurse" {
		t.Errorf("label = %q", n.Style.AccessibilityLabel)
	}
	// The sign is the widget's: Format sees the magnitude.
	if findText(n, "▼ €10.00") == nil {
		t.Error("no movement drawn as ▼ €10.00")
	}
	if q := quotesOf(n)[0]; q.Style.AccessibilityLabel != "SAP, €110.00, runter €10.00" {
		t.Errorf("spoken = %q", q.Style.AccessibilityLabel)
	}
}

func TestStockTickerPercentNeedsAPositiveBase(t *testing.T) {
	for _, c := range []struct{ price, change float64 }{{5, 5}, {1, 3}} {
		if _, ok := percentChange(c.price, c.change); ok {
			t.Errorf("price %v change %v: percent reported over a non-positive base", c.price, c.change)
		}
	}
	if p, ok := percentChange(110, 10); !ok || p != 10 {
		t.Errorf("percentChange(110, 10) = %v, %v; want 10, true", p, ok)
	}
}

func TestStockTickerZeroValueAndStyle(t *testing.T) {
	_, n := renderDebug(t, StockTicker{})
	if len(n.Children) != 0 {
		t.Errorf("an empty tape drew %d children", len(n.Children))
	}

	_, n = renderDebug(t, StockTicker{Quotes: tickerQuotes(), Style: []core.StyleProp{core.Gap(3)}})
	if n.Style.Gap != 3 {
		t.Errorf("gap = %v, want the caller's 3", n.Style.Gap)
	}
}

func formatTwo(v float64) string {
	return StockTicker{}.format(v)
}
