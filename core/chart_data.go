package core

import "math"

// ChartData is a chart's numbers, carried on the chart's node for the hosts
// that have somewhere better to put them than a sentence (N-010).
//
// A chart is announced as one RoleImg element with a summary sentence (see
// comps' chart.go, "One element, one sentence"). The sentence is the right
// first thing to hear and the wrong last one: "12 points, from 12 in Jan to 45
// in Dec" says nothing about March. The data is what a sighted reader reads
// off the axes, and each platform has its own way to offer it.
//
// # Why per host, and not a screen-reader-only node
//
// The obvious primitive is a node every reader can find and nobody can see,
// with the table built out of ordinary nodes inside it. That is a web idea:
// the visually hidden pattern (1px, clipped) is defined by what browsers and
// their readers do with it, and what VoiceOver and TalkBack do with a
// zero-size or transparent view is unmeasured — it cannot be measured without
// Accessibility Inspector or a device in hand, and a primitive whose meaning
// on two of four targets is a guess is not one to put in core. So the data
// travels as data, and each host spends it in its own idiom:
//
//	Web       a visually hidden <table> inside the chart's element, which is
//	          written role="figure" rather than role="img": an img's children
//	          are presentational in ARIA, so a table inside one is pruned from
//	          the tree. The runtime keeps the table as trailing chrome
//	          (data-grmob-chrome="charttable"), so node paths are unmoved.
//	htmlout   the same table and the same role, written into the page.
//	iOS       AXChartDescriptor, the platform's own chart data API (Audio
//	          Graphs and VoiceOver's "Chart details"), built from the same
//	          data on the chart's accessibility element.
//	Android   nothing. Compose has no chart semantics and TalkBack no
//	          equivalent, so the summary sentence is the whole of it there.
//
// # Shape
//
// Series of points, as AXChartDescriptor has it, because a table can be built
// from that and the reverse loses the axes. A categorical x axis names its
// categories in order (X.Categories) and each point names its category
// (Point.X); a numeric one states a range (X.Min, X.Max) and each point its
// value (Point.XValue). A missing value is a point left out, never NaN:
// encoding/json refuses NaN, and "no value for March" is an absent row cell
// on the web and an absent data point on iOS, which is what it means.
type ChartData struct {
	// Title is the chart's subject ("Visits"), the table's caption and the
	// descriptor's title. Empty when the chart was given none.
	Title string `json:"title,omitempty"`

	// X and Y describe the two axes. A chart without a value axis in the
	// cartesian sense (a donut, a funnel) still states Y's range, which is
	// what the descriptor's numeric axis needs, and leaves X categorical.
	X ChartAxis `json:"x"`
	Y ChartAxis `json:"y"`

	// Series are the runs of points, one per legend entry.
	Series []ChartDataSeries `json:"series"`
}

// ChartAxis is one axis of a ChartData: categorical when Categories is set,
// numeric (Min to Max) otherwise.
type ChartAxis struct {
	Title      string   `json:"title,omitempty"`
	Categories []string `json:"categories,omitempty"`
	Min        float64  `json:"min"`
	Max        float64  `json:"max"`
}

// ChartDataSeries is one named run of points.
type ChartDataSeries struct {
	Name string `json:"name"`
	// Continuous says the points are samples of one line (a LineChart),
	// rather than separate marks (bars, slices, scatter points). It is the
	// descriptor's isContinuous, which decides whether Audio Graphs plays a
	// sweep or a run of tones.
	Continuous bool             `json:"continuous,omitempty"`
	Points     []ChartDataPoint `json:"points"`
}

// ChartDataPoint is one value. X names its category on a categorical axis;
// XValue is its position on a numeric one, and XText that position as the
// chart formats it. Text is Y as the chart itself formats it ("$1.2k",
// "45%"), so the table reads what the axis labels say; an empty Text or
// XText means the host formats the number plainly.
type ChartDataPoint struct {
	X      string  `json:"x,omitempty"`
	XValue float64 `json:"xv,omitempty"`
	XText  string  `json:"xt,omitempty"`
	Y      float64 `json:"y"`
	Text   string  `json:"text,omitempty"`
}

// AccessibilityChart attaches d to the node as its "chartData" prop: what
// each host does with it is ChartData's "# Why per host". Put it on the node
// that carries the chart's RoleImg and summary label, which is the element
// the table lives in on the web and the element the descriptor describes on
// iOS.
//
// A copy is stored, with NaN and infinite values dropped (a point with no
// finite Y is a missing value, and the wire cannot carry the non-finite
// ones). The copy also means a caller reusing its slices for the next pass
// cannot rewrite a node that has already been rendered — see "Nodes are
// immutable once rendered".
func AccessibilityChart(d ChartData) BehaviorProp {
	clean := d.finite()
	return behaviorFunc(func(_ *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["chartData"] = clean
	})
}

// finite returns a deep copy of d with every non-finite number removed: a
// point whose Y or XValue is NaN or ±Inf is dropped, and a non-finite axis
// bound becomes 0.
func (d ChartData) finite() ChartData {
	bound := func(v float64) float64 {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0
		}
		return v
	}
	axis := func(a ChartAxis) ChartAxis {
		a.Categories = append([]string(nil), a.Categories...)
		a.Min, a.Max = bound(a.Min), bound(a.Max)
		return a
	}
	out := ChartData{Title: d.Title, X: axis(d.X), Y: axis(d.Y)}
	out.Series = make([]ChartDataSeries, 0, len(d.Series))
	for _, s := range d.Series {
		cs := ChartDataSeries{Name: s.Name, Continuous: s.Continuous,
			Points: make([]ChartDataPoint, 0, len(s.Points))}
		for _, p := range s.Points {
			if math.IsNaN(p.Y) || math.IsInf(p.Y, 0) || math.IsNaN(p.XValue) || math.IsInf(p.XValue, 0) {
				continue
			}
			cs.Points = append(cs.Points, p)
		}
		out.Series = append(out.Series, cs)
	}
	return out
}
