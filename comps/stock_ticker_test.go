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

// The gliding form: a clipping box, two copies each gliding their own width,
// the first one the tape a reader hears and the second hidden, and the same
// quotes in both so the seam is invisible.
func TestStockTickerGlideShape(t *testing.T) {
	_, n := renderDebug(t, StockTicker{Quotes: tickerQuotes(), Glide: 12000})

	// An HBox renders as a Row node with no theme base.
	if n.Type != "Row" || n.Style.Overflow != "hidden" {
		t.Fatalf("root = %s overflow %q, want a clipping HBox (a Row node)", n.Type, n.Style.Overflow)
	}
	if n.Style.AccessibilityRole != "" || n.Style.AccessibilityLabel != "" {
		t.Errorf("the clipping box carries a role %q / label %q; the first copy is the tape",
			n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
	if len(n.Children) != 2 {
		t.Fatalf("copies = %d, want 2", len(n.Children))
	}
	first, second := n.Children[0], n.Children[1]
	for i, c := range n.Children {
		if c.Type != "Row" {
			t.Errorf("copy %d = %s, want an HBox (a Row node)", i, c.Type)
		}
		if c.Style.Glide != 12000 {
			t.Errorf("copy %d glide = %d, want 12000", i, c.Style.Glide)
		}
		if c.Style.FlexShrink != core.ShrinkNone {
			t.Errorf("copy %d shrinks (%v); it must keep its natural width", i, c.Style.FlexShrink)
		}
		if c.Style.Padding.Left == 0 {
			t.Errorf("copy %d has no leading padding, so the seam between copies has no gap", i)
		}
		if len(c.Children) != 3 {
			t.Errorf("copy %d holds %d quotes, want 3", i, len(c.Children))
		}
	}

	// The first copy is the list; the second is pixels only.
	if first.Style.AccessibilityRole != core.RoleList || first.Style.AccessibilityLabel != "Stock prices" {
		t.Errorf("first copy role %q label %q, want the labelled list",
			first.Style.AccessibilityRole, first.Style.AccessibilityLabel)
	}
	if !second.Style.AccessibilityHidden {
		t.Error("the second copy is not hidden from readers, so the tape is read twice")
	}
	if got := len(quotesOf(first)); got != 3 {
		t.Errorf("first copy has %d spoken quotes, want 3", got)
	}
	for i, q := range second.Children {
		if q.Style.AccessibilityRole != "" || q.Style.AccessibilityLabel != "" {
			t.Errorf("second copy's quote %d has role %q label %q inside a hidden node",
				i, q.Style.AccessibilityRole, q.Style.AccessibilityLabel)
		}
	}

	// Same keys in both copies (keys are per parent), so each copy is the
	// same reorder under a step.
	for i := range first.Children {
		if first.Children[i].Key != second.Children[i].Key {
			t.Errorf("copy keys differ at %d: %q vs %q", i, first.Children[i].Key, second.Children[i].Key)
		}
	}
	if first.Children[0].Key != "quote:AAPL" {
		t.Errorf("first key = %q, want quote:AAPL", first.Children[0].Key)
	}
}

// Both copies answer OnTap — the reader sees the second as much as the first —
// and the tape becomes a group, as it does when stepping.
func TestStockTickerGlideTapsBothCopies(t *testing.T) {
	var got []string
	_, n := renderDebug(t, StockTicker{Quotes: tickerQuotes(), Glide: 9000,
		OnTap: func(q StockQuote) { got = append(got, q.Symbol) }})

	first, second := n.Children[0], n.Children[1]
	if first.Style.AccessibilityRole != core.RoleGroup {
		t.Errorf("first copy role = %q, want group when the quotes are buttons", first.Style.AccessibilityRole)
	}
	for i, copy := range []*core.Node{first, second} {
		for j, q := range copy.Children {
			if q.Props["onClick"] == nil {
				t.Errorf("copy %d quote %d registered no tap", i, j)
			}
		}
	}
	if role := second.Children[0].Style.AccessibilityRole; role != "" {
		t.Errorf("a hidden copy's quote has role %q", role)
	}
}

// Start still picks the quote drawn first, in both copies; the caller's Style
// lands on the clipping box; and Glide zero is the stepping tape.
func TestStockTickerGlideStartStyleAndZero(t *testing.T) {
	_, n := renderDebug(t, StockTicker{Quotes: tickerQuotes(), Glide: 9000, Start: 1,
		Style: []core.StyleProp{core.BackgroundColor("#123456")}})
	for i, c := range n.Children {
		if c.Children[0].Key != "quote:MSFT" {
			t.Errorf("copy %d starts on %q, want quote:MSFT", i, c.Children[0].Key)
		}
	}
	if n.Style.Background != "#123456" {
		t.Errorf("caller's Style did not reach the clipping box: %q", n.Style.Background)
	}

	_, still := renderDebug(t, StockTicker{Quotes: tickerQuotes()})
	if still.Type != "Scroll" || still.Style.Glide != 0 {
		t.Errorf("Glide zero should be the stepping Scroll, got %s glide %d", still.Type, still.Style.Glide)
	}
}
