package comps

import (
	"math"

	"github.com/rohanthewiz/grmob/core"
)

// Candle is one period of a CandlestickChart: where the price opened and
// closed, and the highest and lowest it reached in between.
type Candle struct {
	Open, High, Low, Close float64
}

// CandlestickChart draws a price per period as a candle: a thin wick from the
// period's low to its high, and a body from its open to its close, coloured
// by which way the period went.
//
//	comps.CandlestickChart{
//	    Subject: "ACME",
//	    Labels:  []string{"Mon", "Tue", "Wed"},
//	    Candles: []comps.Candle{{Open: 102, High: 108, Low: 101, Close: 107}, ...},
//	}
//
// # Slots
//
// The width is BarChart's: one equal slot per period, so the label row under
// the plot is bandLabels unchanged and each label sits under its candle on
// every target without Go knowing the drawn width.
//
//	│  slot 0  │  slot 1  │  slot 2  │
//	│    │     │          │    │     │   wick: a 1 px line, low to high,
//	│   ┌┴┐    │    │     │   ┌┴┐    │         at the slot's centre
//	│   │ │    │   ┌┴┐    │   │ │    │   body: BodyFill of the slot wide,
//	│   └┬┘    │   └┬┘    │   └┬┘    │         open to close
//	│   Mon    │   Tue    │   Wed    │
//
// # The axis does not start at zero
//
// A bar's length is its value, so BarChart's axis must include zero. A candle
// states a range, not a length, and a share trading between 101 and 108 drawn
// on an axis from 0 would be a row of identical slivers. The axis is
// niceScale over the lows and highs, as LineChart's is by default.
//
// # Which colour is up
//
// A period that closed at or above its open takes UpColor, the theme's
// Success, and one that closed below takes DownColor, its Error. That is the
// Western convention. Markets in China, Japan and Korea print it the other
// way round (red rises), so both are fields, and a chart for those readers
// swaps them. ShowLegend keys the two colours for a page whose readers may be
// used to either.
//
// Colour is the only drawn difference between the two, so the spoken summary
// counts the periods that rose and fell.
//
// # A doji
//
// A period that closed where it opened has a body of no height, which would
// leave a bare wick that looks like missing data. Its body is drawn one px
// tall. The plot is Height px tall and CanvasStretch maps the viewBox's y
// linearly onto it, so one px is exactly chartView/Height units: no
// measurement needed.
//
// # One shape per kind, not per candle
//
// Four shapes carry every candle: the rising wicks, the falling wicks, the
// rising bodies, the falling bodies, each as one path of subpaths. Thirty
// candles cost four nodes and not sixty, and the slots stay put while the
// data changes, so a new period patches props and moves nothing.
//
// # Theme roles read
//
//	Rising     Colors.Success
//	Falling    Colors.Error
//	Gridlines  Colors.BorderColor()
//	Labels     Colors.TextSecondary
type CandlestickChart struct {
	// Candles are the periods, oldest first. A candle holding a NaN or an
	// infinity is not drawn and keeps its slot, as a gap in the row.
	Candles []Candle

	// Labels name the periods. Up to five are drawn; all feed the summary.
	Labels []string

	// Subject says what is priced. It leads the spoken summary and is not
	// drawn.
	Subject string

	// Height is the plot's height in px, excluding labels; 0 means 160.
	Height float64

	// UpColor and DownColor ink a period that rose (close at or above open)
	// and one that fell; empty means the theme's Success and Error. See
	// "Which colour is up".
	UpColor   string
	DownColor string

	// BodyFill is the fraction of each slot the body takes; 0 means 0.6.
	BodyFill float64

	// ShowLegend draws a key for the two colours under the plot, reading
	// UpLabel and DownLabel ("Up" and "Down" when empty).
	ShowLegend bool
	UpLabel    string
	DownLabel  string

	// Format writes a tick label and a spoken price.
	Format func(float64) string

	// Style is applied last, to the outer column.
	Style []core.StyleProp

	// AccessibilityLabel replaces the generated summary.
	AccessibilityLabel string
}

// candleSummaryLimit is the most periods whose open and close are all read
// out. Past it the summary gives the first open, the last close and the
// range, which is what a reader asks of a month of candles anyway.
const candleSummaryLimit = 5

func (c CandlestickChart) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	h := c.Height
	if h <= 0 {
		h = chartHeight
	}
	fill := c.BodyFill
	if fill <= 0 || fill > 1 {
		fill = 0.6
	}
	up := orDefault(c.UpColor, t.Colors.Success)
	down := orDefault(c.DownColor, t.Colors.Error)

	n := max(len(c.Candles), len(c.Labels))

	lo, hi := math.Inf(1), math.Inf(-1)
	for _, k := range c.Candles {
		if !k.valid() {
			continue
		}
		l, u := k.extent()
		lo, hi = math.Min(lo, l), math.Max(hi, u)
	}
	// An empty chart falls through niceScale's own guard (lo > hi) to 0..1.
	scale := niceScale(lo, hi, chartMaxTicks(h), false)

	// One px in viewBox units along y; see "A doji".
	onePx := chartView / h

	wicks := [2]*core.Path{core.NewPath(), core.NewPath()}
	bodies := [2]*core.Path{core.NewPath(), core.NewPath()}
	if n > 0 {
		slot := chartView / float64(n)
		for i, k := range c.Candles {
			if !k.valid() {
				continue
			}
			kind := 0
			if k.Close < k.Open {
				kind = 1
			}
			cx := (float64(i) + 0.5) * slot
			// The wick spans everything the candle states, so a High below
			// the body (a caller's bad row) still draws a wick that covers
			// the body rather than one that stops inside it.
			l, u := k.extent()
			wicks[kind].MoveTo(cx, scale.y(u)).LineTo(cx, scale.y(l))

			top, bottom := scale.y(math.Max(k.Open, k.Close)), scale.y(math.Min(k.Open, k.Close))
			if bottom-top < onePx {
				mid := (top + bottom) / 2
				top, bottom = mid-onePx/2, mid+onePx/2
			}
			x0, x1 := cx-slot*fill/2, cx+slot*fill/2
			bodies[kind].MoveTo(x0, top).LineTo(x1, top).LineTo(x1, bottom).LineTo(x0, bottom).Close()
		}
	}

	shapes := gridShapes(t, scale)
	// Wicks under bodies, so a body's fill covers the wick through it and the
	// candle reads as one mark.
	shapes = append(shapes,
		core.Shape{Path: wicks[0], Stroke: up, StrokeWidth: 1},
		core.Shape{Path: wicks[1], Stroke: down, StrokeWidth: 1},
		core.Shape{Path: bodies[0], Fill: up},
		core.Shape{Path: bodies[1], Fill: down},
	)
	canvas := core.Canvas(chartView, chartView, shapes, core.CanvasStretch, core.CanvasMirrorsRTL, core.Height(px(h)))

	var legendView core.View
	if c.ShowLegend {
		legendView = legend(t,
			[]string{orDefault(c.UpLabel, "Up"), orDefault(c.DownLabel, "Down")},
			[]string{up, down})
	}
	return cartesianFrame(ctx, scale, h, c.Format, canvas, bandLabels(t, c.Labels, n),
		legendView, c.label(), c.Style)
}

// valid reports whether all four prices are finite.
func (k Candle) valid() bool {
	for _, v := range [4]float64{k.Open, k.High, k.Low, k.Close} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

// extent is the lowest and highest of the four prices. For a well-formed
// candle that is Low and High; taking all four keeps a malformed one inside
// the axis.
func (k Candle) extent() (lo, hi float64) {
	return math.Min(math.Min(k.Open, k.Close), math.Min(k.Low, k.High)),
		math.Max(math.Max(k.Open, k.Close), math.Max(k.Low, k.High))
}

// label reads each period's open and close when there are at most
// candleSummaryLimit of them, and otherwise the first open and last close.
// Either way it ends with the range and the count that rose and fell, which
// is what the colours say.
func (c CandlestickChart) label() string {
	if c.AccessibilityLabel != "" {
		return c.AccessibilityLabel
	}
	format := c.Format
	if format == nil {
		format = formatValue
	}
	name := func(i int) string {
		if i < len(c.Labels) && c.Labels[i] != "" {
			return c.Labels[i]
		}
		return "period " + formatValue(float64(i+1))
	}

	first, last, low, high := -1, -1, -1, -1
	rose, fell, count := 0, 0, 0
	var each []string
	for i, k := range c.Candles {
		if !k.valid() {
			continue
		}
		count++
		if first < 0 {
			first = i
		}
		last = i
		l, u := k.extent()
		if low < 0 {
			low, high = i, i
		}
		if pl, _ := c.Candles[low].extent(); l < pl {
			low = i
		}
		if _, ph := c.Candles[high].extent(); u > ph {
			high = i
		}
		if k.Close < k.Open {
			fell++
		} else {
			rose++
		}
		each = append(each, name(i)+" open "+format(k.Open)+", close "+format(k.Close))
	}
	if count == 0 {
		return summaryPrefix(c.Subject) + "no data."
	}

	var parts []string
	if count <= candleSummaryLimit {
		parts = append(parts, each...)
	} else {
		parts = append(parts, formatValue(float64(count))+" periods, from open "+
			format(c.Candles[first].Open)+" in "+name(first)+" to close "+
			format(c.Candles[last].Close)+" in "+name(last))
	}
	l, _ := c.Candles[low].extent()
	_, u := c.Candles[high].extent()
	parts = append(parts,
		"low "+format(l)+" in "+name(low)+", high "+format(u)+" in "+name(high),
		formatValue(float64(rose))+" up, "+formatValue(float64(fell))+" down")
	return summaryPrefix(c.Subject) + joinSentences(parts)
}
