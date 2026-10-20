package comps

import (
	"math"
	"strconv"

	"github.com/rohanthewiz/grmob/core"
)

// StockQuote is one entry on a StockTicker: a symbol, its last price and its
// movement since the previous close.
type StockQuote struct {
	// Symbol is the drawn ticker symbol ("AAPL"). It is also the quote's
	// identity on the tape: two quotes with the same Symbol are told apart by
	// their order, so keep symbols unique where the tape can be reordered.
	Symbol string

	// Name is spoken before the symbol and never drawn ("Apple"). Symbols are
	// read letter by letter, or worse, as a word ("MSFT", "SNAP"), so a name
	// is what makes the tape listenable. Empty speaks the symbol alone.
	Name string

	// Price is the last traded price.
	Price float64

	// Change is the absolute movement since the previous close, signed:
	// negative is down. The percentage is derived from it (see percentChange)
	// rather than passed alongside, so the two drawn figures cannot disagree.
	Change float64
}

// StockTicker is a one-line tape of stock quotes — symbol, price and movement
// — that pans sideways: the strip along the top of a markets screen, a
// watchlist header, a portfolio's holdings at a glance.
//
//	comps.StockTicker{Quotes: []comps.StockQuote{
//	    {Symbol: "AAPL", Name: "Apple", Price: 189.12, Change: 1.23},
//	    {Symbol: "MSFT", Name: "Microsoft", Price: 411.65, Change: -2.10},
//	    {Symbol: "TSLA", Name: "Tesla", Price: 242.00},
//	}}
//
//	┌ Scroll (Horizontal; RoleList, "Stock prices") ─────────────────────────────┐
//	│ ┌ HBox (listitem) ─────────────┐   ┌ HBox (listitem) ──────────────┐       │
//	│ │ AAPL  189.12  ▲ 1.23 (0.65%) │   │ MSFT  411.65  ▼ 2.10 (0.51%)  │  ...→ │
//	│ └──────────────────────────────┘   └───────────────────────────────┘       │
//	└────────────────────────────────────────────────────────────────────────────┘
//
// # Controlled, no hooks
//
// Prices are the caller's, and so is the motion. The widget holds no state
// and starts no timer, so it may be rendered conditionally (inside core.If).
//
// # Stepping the tape, not scrolling it
//
// A broadcast-style ticker glides continuously, and core has no primitive for
// that: Spin is the only looping animation, and its doc explains why a general
// looping translate was left out until a second consumer needed one. A glide
// driven from Go would be a patch per frame over the bridge. Scrolling with
// core.ScrollIntoView is no substitute either: it is one app-wide command, so
// a ticker issuing it every few seconds would take the command away from the
// app's own "jump to" links, and would pull the page back to the tape whenever
// the reader had scrolled it off screen.
//
// So the tape pans under the reader's finger, and a caller who wants it to
// move steps it with Start: the quote drawn first, wrapping round the end.
// Each quote is keyed by its symbol, so a step is a reorder of existing nodes
// rather than a re-patch of every figure.
//
//	off := core.NewState(ctx, 0)
//	hooks.UseIntervalWhile(ctx, !paused.Get(), func() { off.Set(off.Get() + 1) }, 3*time.Second)
//	comps.StockTicker{Quotes: qs, Start: off.Get()}
//
// Content that moves by itself for more than five seconds needs a way to stop
// it (WCAG 2.2.2, Pause, Stop, Hide). A caller that steps the tape owns that
// switch too, which is the other reason the stepping lives with the caller.
//
// # Direction is never colour alone
//
// Each movement is drawn with a glyph (▲ up, ▼ down) and spoken with a word
// ("up", "down", "unchanged"); the colour reinforces the glyph and never
// replaces it. Red-green is the most common colour-vision deficiency, and the
// pair this widget defaults to is exactly that pair.
//
// Unlike StatTile, which refuses to guess whether up is good, the colours here
// are on by default: on a price tape the convention is the meaning, and every
// market uses one. Which one differs, though. Most of the world draws a rise
// green; mainland China, Taiwan, Japan and Korea draw it red. RedRises swaps
// the pair, and the zero value is the more common reading.
//
// # Numbers
//
// Format writes the price and the size of the change; nil writes two decimal
// places. The sign is the widget's, not Format's: Format is handed the
// magnitude and the widget adds the glyph, so a currency formatter that puts
// its own minus in a different place cannot produce "▼ -$2.10". The
// percentage is always two places, because it is a ratio and not money.
//
// # Accessibility
//
// Each quote is one stop for a screen reader, read as a sentence: "Apple,
// AAPL, 189.12, up 1.23, 0.65 percent". Its drawn parts are hidden, so the
// glyph is never read out as "black up-pointing triangle". The tape itself
// is a labelled list (Label, "Stock prices" by default). With OnTap set each
// quote becomes a button instead of a list item, and the tape a labelled
// group, because a list item and a button are two roles and a node has one.
//
// # Theme roles read
//
//	Symbol       Typography.Body, bold
//	Price        Typography.Body
//	Movement     Typography.Caption; Success / Error ink tones, or
//	             Colors.TextSecondary when unchanged
//	Gaps         Spacing.LG between quotes, Spacing.XS within one
type StockTicker struct {
	// Quotes are the tape's contents, in order.
	Quotes []StockQuote

	// Start is the index of the quote drawn first; the ones before it follow
	// the last. Out-of-range values wrap, negative ones included, so a caller
	// can step it with a bare counter. See the type comment.
	Start int

	// Format writes the price and the magnitude of the change. Nil writes two
	// decimal places.
	Format func(float64) string

	// HidePercent drops the percentage after the change, for a tape too
	// narrow for it. The spoken sentence drops it too.
	HidePercent bool

	// RedRises draws a rise red and a fall green, the convention in mainland
	// China, Taiwan, Japan and Korea. The glyphs and spoken words are
	// unchanged. The zero value draws rises green.
	RedRises bool

	// OnTap is called with the quote that was tapped — open the symbol's
	// detail screen. Nil leaves the quotes inert, and registers nothing.
	OnTap func(StockQuote)

	// Label names the tape for screen readers. Empty is "Stock prices".
	Label string

	// UpLabel, DownLabel and UnchangedLabel are the spoken directions, for
	// localisation. Empty is "up", "down" and "unchanged". PercentLabel is
	// spoken after the percentage; empty is "percent".
	UpLabel        string
	DownLabel      string
	UnchangedLabel string
	PercentLabel   string

	// Style is applied to the scrolling strip after the widget's own props.
	Style []core.StyleProp
}

// tickerDirection is a quote's movement reduced to the three cases that
// change how it is drawn and spoken.
type tickerDirection int

const (
	tickerFlat tickerDirection = iota
	tickerUp
	tickerDown
)

func (s StockTicker) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	items := make([]core.PropsAndChildren, 0, len(s.Style)+len(s.Quotes)+5)
	items = append(items,
		// A Scroll turned sideways rather than a Row with an overflow: the
		// natives pan only inside their scroll composites (see ChipStrip's
		// Scrollable for the full reasoning).
		core.Horizontal(),
		core.Gap(float64(t.Spacing.LG)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.AccessibilityLabel(orDefault(s.Label, "Stock prices")),
	)
	if s.OnTap == nil {
		items = append(items, core.AccessibilityRole(core.RoleList))
	} else {
		// A group, not a list: the children are buttons, and a list whose
		// items are not listitems is announced as an empty list.
		items = append(items, core.AccessibilityRole(core.RoleGroup))
	}
	items = append(items, asProps(s.Style)...)

	n := len(s.Quotes)
	if n > 0 {
		// Go's % keeps the dividend's sign, so -1 % 5 is -1; adding n and
		// taking the remainder again folds every integer into [0, n).
		first := ((s.Start % n) + n) % n

		// Occurrence counts make duplicate symbols distinct keys. Counted in
		// the caller's order, not the drawn order, so the same quote keeps the
		// same key however far the tape has been stepped.
		keys := make([]string, n)
		seen := make(map[string]int, n)
		for i, q := range s.Quotes {
			k := "quote:" + q.Symbol
			if c := seen[q.Symbol]; c > 0 {
				k += "#" + strconv.Itoa(c)
			}
			seen[q.Symbol]++
			keys[i] = k
		}

		for j := range n {
			i := (first + j) % n
			items = append(items, core.Keyed(keys[i], s.quote(t, s.Quotes[i])))
		}
	}
	return core.Scroll(items...).Render(ctx)
}

// quote builds one entry on the tape.
func (s StockTicker) quote(t *core.Theme, q StockQuote) core.View {
	dir := tickerFlat
	switch {
	case q.Change > 0:
		dir = tickerUp
	case q.Change < 0:
		dir = tickerDown
	}

	price := s.format(q.Price)
	size := s.format(math.Abs(q.Change))
	pct, hasPct := percentChange(q.Price, q.Change)
	pctText := ""
	if hasPct && !s.HidePercent {
		pctText = strconv.FormatFloat(math.Abs(pct), 'f', 2, 64) + "%"
	}

	// Drawn: "▲ 1.23 (0.65%)". The glyph carries the sign, so the figures
	// are magnitudes; "▼ −2.10" would say "down" twice.
	move := size
	switch dir {
	case tickerUp:
		move = "▲ " + size
	case tickerDown:
		move = "▼ " + size
	}
	if pctText != "" {
		move += " (" + pctText + ")"
	}

	// Spoken: "Apple, AAPL, 189.12, up 1.23, 0.65 percent".
	spoken := q.Symbol
	if q.Name != "" {
		spoken = q.Name + ", " + q.Symbol
	}
	spoken += ", " + price + ", "
	switch dir {
	case tickerUp:
		spoken += orDefault(s.UpLabel, "up") + " " + size
	case tickerDown:
		spoken += orDefault(s.DownLabel, "down") + " " + size
	default:
		spoken += orDefault(s.UnchangedLabel, "unchanged")
	}
	if pctText != "" && dir != tickerFlat {
		spoken += ", " + strconv.FormatFloat(math.Abs(pct), 'f', 2, 64) + " " + orDefault(s.PercentLabel, "percent")
	}

	props := []core.PropsAndChildren{
		// HBox, not Row: the Row base carries the theme's screen inset, and a
		// quote is a run of words inside a strip, not a band of its own.
		core.Gap(float64(t.Spacing.XS)),
		core.AlignItemsProp(core.AlignItemsCenter),
		// A quote never wraps or squeezes: the strip pans, so each entry
		// keeps its natural width rather than sharing the visible one.
		core.FlexShrink(0),
		core.AccessibilityLabel(spoken),
	}
	if s.OnTap != nil {
		onTap := s.OnTap
		props = append(props,
			core.OnClick(func() { onTap(q) }),
			core.AccessibilityRole(core.RoleButton),
		)
	} else {
		props = append(props,
			core.AccessibilityRole(core.RoleListItem),
			core.AccessibilityNestingLevel(1),
		)
	}
	props = append(props,
		core.Text(q.Symbol,
			core.UseStyle(t.Typography.Body),
			core.FontWeight(core.Bold),
			core.AccessibilityHidden(),
		),
		core.Text(price,
			core.UseStyle(t.Typography.Body),
			core.AccessibilityHidden(),
		),
		core.Text(move,
			core.UseStyle(t.Typography.Caption),
			core.TextColor(s.directionInk(t, dir)),
			core.AccessibilityHidden(),
		),
	)
	return core.HBox(props...)
}

// format writes a price or a change magnitude with the caller's Format, or
// two decimal places.
func (s StockTicker) format(v float64) string {
	if s.Format != nil {
		return s.Format(v)
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// directionInk is the movement's colour. The ink tone rather than the fill
// colour of each role, for StatTile's reason: the text sits on a backdrop the
// widget cannot see, so it needs the tone made to be read on the theme's page.
//
//	           rise      fall      flat
//	default    Success   Error     TextSecondary
//	RedRises   Error     Success   TextSecondary
func (s StockTicker) directionInk(t *core.Theme, dir tickerDirection) string {
	rise, fall := VariantSuccess, VariantError
	if s.RedRises {
		rise, fall = fall, rise
	}
	switch dir {
	case tickerUp:
		return rise.AsInk(t)
	case tickerDown:
		return fall.AsInk(t)
	default:
		return t.Colors.TextSecondary
	}
}

// percentChange is the movement as a percentage of the previous close, which
// is the price before the change:
//
//	previous = price − change
//	percent  = change / previous × 100
//
// It reports false when there is no meaningful base: a previous close of zero
// or below (a quote just listed, or bad data), or a non-finite input. The
// percentage is then left off rather than drawn as "Inf%".
func percentChange(price, change float64) (float64, bool) {
	prev := price - change
	if prev <= 0 || math.IsNaN(prev) || math.IsInf(prev, 0) {
		return 0, false
	}
	pct := change / prev * 100
	if math.IsNaN(pct) || math.IsInf(pct, 0) {
		return 0, false
	}
	return pct, true
}
