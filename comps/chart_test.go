package comps

import (
	"math"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// Heckbert's ticks: whole 1/2/5 steps, the data inside them, and the budget
// kept.
func TestNiceScale(t *testing.T) {
	cases := []struct {
		lo, hi   float64
		maxTicks int
		zero     bool
		want     valueScale
	}{
		{12, 45, 5, false, valueScale{10, 50, 10}},
		// 0..50 by 10 is six ticks, over a budget of five, so the step
		// moves up to the next nice number.
		{12, 45, 5, true, valueScale{0, 60, 20}},
		{0, 95, 6, false, valueScale{0, 100, 20}},
		{-2000, 12000, 6, true, valueScale{-5000, 15000, 5000}},
		{0.1, 0.9, 6, false, valueScale{0, 1, 0.2}},
		{0.1, 0.9, 5, false, valueScale{0, 1, 0.5}},
		{7, 7, 5, false, valueScale{6, 8, 0.5}},
		{0, 0, 6, true, valueScale{0, 1, 0.2}},
	}
	for _, c := range cases {
		got := niceScale(c.lo, c.hi, c.maxTicks, c.zero)
		ticks := got.ticks()
		if len(ticks) > c.maxTicks {
			t.Errorf("niceScale(%v, %v, %d): %d ticks %v, over budget", c.lo, c.hi, c.maxTicks, len(ticks), ticks)
		}
		if got.lo > c.lo || got.hi < c.hi {
			t.Errorf("niceScale(%v, %v): [%v, %v] does not contain the data", c.lo, c.hi, got.lo, got.hi)
		}
		if c.zero && (got.lo > 0 || got.hi < 0) {
			t.Errorf("niceScale(%v, %v, zero): [%v, %v] omits zero", c.lo, c.hi, got.lo, got.hi)
		}
		// Every step is 1, 2 or 5 × 10ⁿ.
		f := got.step / math.Pow(10, math.Floor(math.Log10(got.step)))
		if r := math.Round(f*1e6) / 1e6; r != 1 && r != 2 && r != 5 {
			t.Errorf("niceScale(%v, %v): step %v is not nice", c.lo, c.hi, got.step)
		}
		if c.want.step != 0 && got != c.want {
			t.Errorf("niceScale(%v, %v, %d, %v) = %+v, want %+v", c.lo, c.hi, c.maxTicks, c.zero, got, c.want)
		}
	}
}

func TestFormatTick(t *testing.T) {
	cases := []struct {
		v, step, extent float64
		want            string
	}{
		{20, 10, 40, "20"},
		{0.4, 0.2, 1, "0.4"},
		{1, 0.5, 1, "1.0"},
		{-0.0000001, 0.2, 1, "0.0"},
		{0, 0.5, 1, "0"},
		{15000, 5000, 15000, "15k"},
		{2500, 500, 2500, "2500"},
		{-5000, 5000, 15000, "-5k"},
		{1500000, 500000, 1500000, "1.5M"},
		{500000, 500000, 1500000, "0.5M"},
	}
	for _, c := range cases {
		if got := formatTick(c.v, c.step, c.extent); got != c.want {
			t.Errorf("formatTick(%v, %v, %v) = %q, want %q", c.v, c.step, c.extent, got, c.want)
		}
	}
}

// Duplicated roles collapse (DefaultTheme's Secondary and Success are the same
// green), and the cycle continues with translucent tints, not new hues.
func TestChartPaletteDedupesRolesAndTints(t *testing.T) {
	th := core.DefaultTheme
	p := chartPalette(th)
	seen := map[string]bool{}
	for _, c := range p {
		if seen[strings.ToLower(c)] {
			t.Errorf("palette repeats %s: %v", c, p)
		}
		seen[strings.ToLower(c)] = true
	}
	if p[0] != th.Colors.Primary {
		t.Errorf("first series colour = %s, want Primary %s", p[0], th.Colors.Primary)
	}
	half := len(p) / 2
	for i := 0; i < half; i++ {
		if p[half+i] != withAlpha(p[i], "99") {
			t.Errorf("palette[%d] = %s, want a tint of %s", half+i, p[half+i], p[i])
		}
	}
}

func TestWithAlpha(t *testing.T) {
	for in, want := range map[string]string{
		"#0040DD":   "#0040DD33",
		"#abc":      "#aabbcc33",
		"#3C3C4399": "#3C3C4399",
		"red":       "red",
	} {
		if got := withAlpha(in, "33"); got != want {
			t.Errorf("withAlpha(%q) = %q, want %q", in, got, want)
		}
	}
}

// columnWeights reads the FlexGrow of each box in an x label row, with its
// text.
func columnWeights(row *core.Node) (weights []float64, texts []string) {
	for _, box := range row.Children {
		weights = append(weights, box.Style.FlexGrow)
		text := ""
		if len(box.Children) > 0 {
			text, _ = box.Children[0].Props["content"].(string)
		}
		texts = append(texts, text)
	}
	return weights, texts
}

// Labels under a line chart sit on their points: interior boxes are centred on
// the point, the ends hug the edges, and the weights tile the whole axis.
func TestPointLabelsTileTheAxis(t *testing.T) {
	th := core.DefaultTheme
	labels := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	row := renderView(t, pointLabels(th, labels, 12))
	weights, texts := columnWeights(row)
	// 11 intervals, stride ceil(11/4) = 3: Jan, Apr, Jul, Oct, then a filler.
	wantW := []float64{1.5, 3, 3, 3, 0.5}
	wantT := []string{"Jan", "Apr", "Jul", "Oct", ""}
	if len(weights) != len(wantW) {
		t.Fatalf("weights %v texts %v, want %v %v", weights, texts, wantW, wantT)
	}
	sum, pos := 0.0, 0.0
	for i := range wantW {
		if weights[i] != wantW[i] || texts[i] != wantT[i] {
			t.Errorf("box %d = %v %q, want %v %q", i, weights[i], texts[i], wantW[i], wantT[i])
		}
		// A centred box's middle must be its point's index.
		if i > 0 && i < 4 {
			if mid := pos + weights[i]/2; mid != float64(i*3) {
				t.Errorf("box %d centred at %v, want point %d", i, mid, i*3)
			}
		}
		pos += weights[i]
		sum += weights[i]
	}
	if sum != 11 {
		t.Errorf("weights sum to %v, want 11 (the point span)", sum)
	}

	// When the stride lands on the last point it is labelled, end-aligned.
	row = renderView(t, pointLabels(th, labels[:9], 9))
	weights, texts = columnWeights(row)
	if texts[len(texts)-1] != "Sep" || row.Children[len(row.Children)-1].Children[0].Style.Align != core.AlignEnd {
		t.Errorf("last label = %q (weights %v); want Sep, end-aligned", texts[len(texts)-1], weights)
	}
}

// The y label column's arithmetic: N+1 boxes of one label line, spaced so box i
// starts at i·H/N.
func TestYAxisSpacingPutsLabelsOnGridlines(t *testing.T) {
	th := core.DefaultTheme
	s := valueScale{0, 40, 10}
	col := renderView(t, yAxis(th, s, 160, nil))
	if len(col.Children) != 5 {
		t.Fatalf("%d labels, want 5", len(col.Children))
	}
	gap := col.Style.Gap
	for i := range col.Children {
		top := float64(i) * (chartLabelLine + gap)
		if want := float64(i) * 160 / 4; math.Abs(top-want) > 1e-9 {
			t.Errorf("label %d top at %v, want %v", i, top, want)
		}
	}
	if findText(col, "40") == nil || findText(col, "0") == nil {
		t.Error("tick labels 40 and 0 missing")
	}
	if first := col.Children[0].Children[0].Props["content"]; first != "40" {
		t.Errorf("top label = %v, want the highest tick", first)
	}
}

func canvasOf(n *core.Node) *core.Node {
	return findFirst(n, func(n *core.Node) bool { return n.Type == "Canvas" })
}

func TestLineChartIsOneElementWithASummary(t *testing.T) {
	n := renderView(t, LineChart{
		Subject: "Visits",
		Labels:  []string{"Mon", "Tue", "Wed"},
		Series:  []ChartSeries{{Name: "Visits", Values: []float64{12, 45, 30}}},
	})
	if n.Style.AccessibilityRole != core.RoleImg {
		t.Errorf("role = %q, want img", n.Style.AccessibilityRole)
	}
	want := "Visits: 3 points, from 12 in Mon to 30 in Wed, low 12 in Mon, high 45 in Tue."
	if n.Style.AccessibilityLabel != want {
		t.Errorf("label = %q\n want %q", n.Style.AccessibilityLabel, want)
	}
	c := canvasOf(n)
	if c == nil || c.Props["scale"] != string(core.CanvasStretch) {
		t.Fatalf("canvas missing or not stretched: %+v", c)
	}
	if !c.Style.AccessibilityHidden {
		t.Error("the canvas inside a summarised chart should be hidden")
	}
	// One series, no legend.
	if findText(n, "Visits") != nil {
		t.Error("a single series should draw no legend")
	}
}

// A NaN splits the line into two subpaths, and a value alone between gaps is a
// dot rather than a vanishing zero-length line.
func TestLinePathsBreakAtGaps(t *testing.T) {
	s := valueScale{0, 10, 5}
	nan := math.NaN()
	line, area, dots := linePaths([]float64{1, 2, nan, 5, nan, 7, 8}, 7, s, s.y(0))
	moves := func(p *core.Path) int {
		n := renderView(t, core.Canvas(100, 100, []core.Shape{{Path: p, Stroke: "#000"}}))
		d := n.Children[0].Props["d"].([]float64)
		count := 0
		for i := 0; i < len(d); {
			switch d[i] {
			case core.PathMove:
				count++
				i += 3
			case core.PathLine:
				i += 3
			case core.PathCubic:
				i += 7
			default:
				i++
			}
		}
		return count
	}
	if got := moves(line); got != 2 {
		t.Errorf("line has %d subpaths, want 2", got)
	}
	if got := moves(area); got != 2 {
		t.Errorf("area has %d subpaths, want 2", got)
	}
	if got := moves(dots); got != 1 {
		t.Errorf("dots have %d subpaths, want 1 (the lone 5)", got)
	}
}

func TestLineChartLegendForSeveralSeries(t *testing.T) {
	n := renderView(t, LineChart{Series: []ChartSeries{
		{Name: "2025", Values: []float64{1, 2}},
		{Name: "2026", Values: []float64{2, 3}, Color: "#123456"},
	}})
	if findText(n, "2025") == nil || findText(n, "2026") == nil {
		t.Error("legend names missing")
	}
	sw := findFirst(n, func(n *core.Node) bool { return n.Style != nil && n.Style.Background == "#123456" })
	if sw == nil {
		t.Error("a series' own Color should key its legend swatch")
	}
}

func TestAreaChartIsZeroBased(t *testing.T) {
	area := renderView(t, AreaChart{Series: []ChartSeries{{Values: []float64{50, 60}}}})
	if findText(area, "0") == nil {
		t.Error("an area chart's axis should reach zero")
	}
	line := renderView(t, LineChart{Series: []ChartSeries{{Values: []float64{50, 60}}}})
	if findText(line, "0") != nil {
		t.Error("a line chart's axis should not be pulled to zero by default")
	}
}

func TestBarChartSummaryAndBars(t *testing.T) {
	n := renderView(t, BarChart{
		Subject: "Steps",
		Labels:  []string{"Mon", "Tue"},
		Series:  []ChartSeries{{Values: []float64{8200, -300}}},
	})
	if got, want := n.Style.AccessibilityLabel, "Steps: Mon 8200, Tue -300."; got != want {
		t.Errorf("label = %q, want %q", got, want)
	}
	c := canvasOf(n)
	// Gridlines, one bar path, the zero line.
	var bars *core.Node
	for _, s := range c.Children {
		if s.Props["fill"] != nil {
			bars = s
		}
	}
	if bars == nil {
		t.Fatal("no bar shape")
	}
	d := bars.Props["d"].([]float64)
	// Two rectangles: move + 3 lines + close each = 3+9+1 = 13 numbers.
	if len(d) != 26 {
		t.Errorf("bar path has %d numbers, want two closed rectangles (26)", len(d))
	}
	// The negative bar hangs below the zero line: its top is at the zero line.
	scale := niceScale(-300, 8200, chartMaxTicks(chartHeight), true)
	zero := math.Round(scale.y(0)*100) / 100
	if d[13+2] != zero {
		t.Errorf("negative bar top at %v, want the zero line %v", d[15], zero)
	}
	// Categories label equal boxes.
	if findText(n, "Mon") == nil || findText(n, "Tue") == nil {
		t.Error("category labels missing")
	}
}

func TestPercentagesSumTo100(t *testing.T) {
	cases := [][]float64{
		{1, 1, 1},
		{1200, 450, 200, 150},
		{1, 0, 2},
		{0.2, 0.2, 0.2, 0.2, 0.2, 0.2, 0.2},
	}
	for _, vals := range cases {
		total := 0.0
		for _, v := range vals {
			total += v
		}
		got := percentages(vals, total)
		sum := 0
		for i, p := range got {
			sum += p
			if vals[i] == 0 && p != 0 {
				t.Errorf("percentages(%v): a zero share got %d%%", vals, p)
			}
		}
		if sum != 100 {
			t.Errorf("percentages(%v) = %v, sums to %d", vals, got, sum)
		}
	}
	if got := percentages([]float64{1, 1, 1}, 3); got[0] != 34 || got[1] != 33 || got[2] != 33 {
		t.Errorf("ties should go to the earlier slice: %v", got)
	}
}

func TestDonutChartSlotsAndSummary(t *testing.T) {
	n := renderView(t, DonutChart{
		Subject:     "Budget",
		Slices:      []ChartSlice{{Label: "Rent", Value: 3}, {Label: "None", Value: 0}, {Label: "Food", Value: 1}},
		CenterValue: "4",
		CenterLabel: "total",
	})
	want := "Budget: Rent 3, 75%; Food 1, 25%. 4 total."
	if n.Style.AccessibilityLabel != want {
		t.Errorf("label = %q, want %q", n.Style.AccessibilityLabel, want)
	}
	c := canvasOf(n)
	// Track plus one slot per slice, the zero slice kept as an empty shape.
	if len(c.Children) != 4 {
		t.Fatalf("%d shapes, want 4", len(c.Children))
	}
	if _, filled := c.Children[2].Props["fill"]; filled {
		t.Error("a zero slice should draw nothing")
	}
	if c.Props["scale"] != string(core.CanvasFit) {
		t.Error("a donut must stay round: CanvasFit")
	}
	for _, s := range []string{"75%", "0%", "25%", "4", "total"} {
		if findText(n, s) == nil {
			t.Errorf("text %q missing", s)
		}
	}
}

func TestPieChartHasNoHoleOrCentre(t *testing.T) {
	n := renderView(t, PieChart{Slices: []ChartSlice{{Label: "A", Value: 1}}, CenterValue: "ignored"})
	if findText(n, "ignored") != nil {
		t.Error("a pie draws no centre text")
	}
	c := canvasOf(n)
	d := c.Children[1].Props["d"].([]float64)
	// A wedge starts at the centre.
	if d[0] != core.PathMove || d[1] != 50 || d[2] != 50 {
		t.Errorf("pie slice starts at (%v, %v), want the centre", d[1], d[2])
	}
}

func TestGauge(t *testing.T) {
	n := renderView(t, Gauge{Value: 72, Label: "Battery", ValueText: "72%"})
	if got := n.Style.AccessibilityLabel; got != "Battery, 72%" {
		t.Errorf("label = %q", got)
	}
	if got := (Gauge{Value: 3, Min: 0, Max: 4}).label(); got != "3 of 4" {
		t.Errorf("unlabelled gauge = %q, want \"3 of 4\"", got)
	}
	if f := (Gauge{Value: 150}).fraction(); f != 1 {
		t.Errorf("an over-range value should clamp to full, got %v", f)
	}
	// An empty gauge keeps the fill's slot.
	empty := canvasOf(renderView(t, Gauge{Value: 0}))
	if len(empty.Children) != 2 {
		t.Errorf("%d shapes, want track and (empty) fill", len(empty.Children))
	}
	if d := empty.Children[1].Props["d"].([]float64); len(d) != 0 {
		t.Errorf("empty gauge fill path = %v, want none", d)
	}
	// The track's round caps stay inside the box: the radius leaves half a
	// stroke. Size 100 and Thickness 10 mean one viewBox unit per px.
	track := canvasOf(renderView(t, Gauge{Value: 50, Size: 100, Thickness: 10})).Children[0]
	d := track.Props["d"].([]float64)
	x, y := d[1]-50, d[2]-50
	if r := math.Hypot(x, y); math.Abs(r-45) > 0.01 {
		t.Errorf("track radius %v, want 45", r)
	}
}

func TestSparklineStaysInsideItsBox(t *testing.T) {
	n := renderView(t, Sparkline{Values: []float64{3, 9, 1, 7}, ShowLast: true, Height: 30, StrokeWidth: 2})
	if n.Type != "Canvas" || n.Style.AccessibilityRole != core.RoleImg {
		t.Fatalf("sparkline should be a labelled canvas, got %s %q", n.Type, n.Style.AccessibilityRole)
	}
	if !strings.Contains(n.Style.AccessibilityLabel, "from 3 to 7, low 1, high 9") {
		t.Errorf("label = %q", n.Style.AccessibilityLabel)
	}
	// The dot is 3 strokes (6 px) wide, so its centre must be at least 3 px
	// (10 viewBox units of a 30 px box) from each edge.
	line := n.Children[1].Props["d"].([]float64)
	for i := 0; i < len(line); i += 3 {
		if y := line[i+2]; y < 10-1e-6 || y > 90+1e-6 {
			t.Errorf("point %d at y=%v, inside the 3 px margin", i/3, y)
		}
	}
	last := n.Children[3].Props["d"].([]float64)
	if len(last) != 6 || last[1] != 100 {
		t.Errorf("last-value dot = %v, want a zero-length stroke at x=100", last)
	}
}

// bezierAt evaluates one cubic segment at u.
func bezierAt(p0, c1, c2, p1, u float64) float64 {
	v := 1 - u
	return v*v*v*p0 + 3*v*v*u*c1 + 3*v*u*u*c2 + u*u*u*p1
}

// A smoothed line is a monotone cubic: every segment stays between its two
// end values (no invented peak), a flat run stays flat, and it still passes
// through every point.
func TestSmoothLineIsMonotone(t *testing.T) {
	xs := []float64{0, 20, 40, 60, 80, 100}
	ys := []float64{90, 10, 12, 12, 80, 20}
	p := core.NewPath().MoveTo(xs[0], ys[0])
	traceRun(p, xs, ys, true, false)
	n := renderView(t, core.Canvas(100, 100, []core.Shape{{Path: p, Stroke: "#000"}}))
	d := n.Children[0].Props["d"].([]float64)

	seg := 0
	for i := 3; i < len(d); i += 7 {
		if d[i] != core.PathCubic {
			t.Fatalf("op %v at %d, want a cubic", d[i], i)
		}
		y0, y1 := ys[seg], ys[seg+1]
		c1, c2, end := d[i+2], d[i+4], d[i+6]
		if math.Abs(end-y1) > 0.01 || math.Abs(d[i+5]-xs[seg+1]) > 0.01 {
			t.Errorf("segment %d ends at (%v, %v), want point (%v, %v)", seg, d[i+5], end, xs[seg+1], y1)
		}
		lo, hi := math.Min(y0, y1), math.Max(y0, y1)
		for u := 0.0; u <= 1; u += 0.05 {
			if y := bezierAt(y0, c1, c2, y1, u); y < lo-0.01 || y > hi+0.01 {
				t.Errorf("segment %d leaves [%v, %v] at u=%.2f: %v", seg, lo, hi, u, y)
				break
			}
		}
		seg++
	}
	if seg != len(xs)-1 {
		t.Errorf("%d segments, want %d", seg, len(xs)-1)
	}

	// Reversed, it is the same curve: the segment from point 1 back to 0 has
	// the forward segment's controls swapped.
	r := core.NewPath().MoveTo(xs[len(xs)-1], ys[len(ys)-1])
	traceRun(r, xs, ys, true, true)
	rd := renderView(t, core.Canvas(100, 100, []core.Shape{{Path: r, Stroke: "#000"}})).Children[0].Props["d"].([]float64)
	last := len(rd) - 7 // the final reversed segment runs from point 1 to point 0
	if math.Abs(rd[last+1]-d[3+3]) > 0.01 || math.Abs(rd[last+2]-d[3+4]) > 0.01 ||
		math.Abs(rd[last+3]-d[3+1]) > 0.01 || math.Abs(rd[last+4]-d[3+2]) > 0.01 {
		t.Errorf("reversed segment controls %v, want forward %v swapped", rd[last+1:last+5], d[4:8])
	}
}

// Stacked series draw running totals, the axis spans the totals, and the
// summary still reads each series' own values.
func TestStackedAreaDrawsTotals(t *testing.T) {
	nan := math.NaN()
	series := []ChartSeries{
		{Name: "A", Values: []float64{10, 20, nan}},
		{Name: "B", Values: []float64{5, 5, 5}},
	}
	got := stackSeries(series, 3)
	if want := []float64{15, 25, 5}; !slicesEqual(got[1].Values, want) {
		t.Errorf("totals = %v, want %v", got[1].Values, want)
	}
	if want := []float64{10, 20, 0}; !slicesEqual(got[0].Values, want) {
		t.Errorf("first band = %v, want a missing value as 0: %v", got[0].Values, want)
	}
	n := renderView(t, AreaChart{Subject: "Mix", Stacked: true, Series: series})
	if findText(n, "25") == nil && findText(n, "30") == nil {
		t.Error("the axis should reach the highest total (25)")
	}
	if !strings.Contains(n.Style.AccessibilityLabel, "B, 3 points, from 5") {
		t.Errorf("summary should read B's own values: %q", n.Style.AccessibilityLabel)
	}
}

func slicesEqual(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// A stacked bar puts positives above zero end to end and negatives below it,
// each series one segment of the category's single bar.
func TestStackedBarsRunEndToEnd(t *testing.T) {
	c := BarChart{Stacked: true, Series: []ChartSeries{
		{Values: []float64{10, -4}},
		{Values: []float64{30, 6}},
	}}
	lo, hi := stackRange(c.Series, 2)
	if lo != -4 || hi != 40 {
		t.Errorf("stackRange = %v..%v, want -4..40", lo, hi)
	}
	s := valueScale{-10, 40, 10}
	r := c.barRects(2, 0.5, s)
	// Category 0: series 0 from 0 to 10, series 1 from 10 to 40, same band.
	a, b := r[0][0], r[1][0]
	if a.a0 != b.a0 || a.a1 != b.a1 {
		t.Errorf("stacked segments in different bands: %+v %+v", a, b)
	}
	if math.Abs(a.b1-s.y(0)) > 1e-9 || math.Abs(a.b0-s.y(10)) > 1e-9 || math.Abs(b.b1-s.y(10)) > 1e-9 || math.Abs(b.b0-s.y(40)) > 1e-9 {
		t.Errorf("segments %+v %+v do not run 0→10→40", a, b)
	}
	// Category 1: -4 hangs below zero, 6 stands on zero, not on -4.
	neg, pos := r[0][1], r[1][1]
	if math.Abs(neg.b0-s.y(0)) > 1e-9 || math.Abs(pos.b1-s.y(0)) > 1e-9 {
		t.Errorf("category 1: negative %+v and positive %+v should both start at zero", neg, pos)
	}
}

// Horizontal bars run along x from the zero line, one band per category from
// the top, with names in a column and every x tick labelled.
func TestHorizontalBarsLieAlongX(t *testing.T) {
	c := BarChart{Horizontal: true, Labels: []string{"Rent", "Food"},
		Series: []ChartSeries{{Values: []float64{1200, 450}}}}
	s := niceScale(0, 1200, chartMaxXLabels, true)
	r := c.barRects(2, 0.7, s)
	if want := 1200 / s.hi * chartView; r[0][0].b0 != 0 || math.Abs(r[0][0].b1-want) > 1e-9 {
		t.Errorf("the 1200 bar should run from x=0 to %v: %+v (scale %+v)", want, r[0][0], s)
	}
	if r[0][0].a0 >= r[0][1].a0 {
		t.Error("the first category should be the top band")
	}
	n := renderView(t, c)
	if findText(n, "Rent") == nil || findText(n, "Food") == nil {
		t.Error("names missing")
	}
	for _, tick := range s.ticks() {
		if findText(n, formatTick(tick, s.step, 1200)) == nil {
			t.Errorf("tick %v unlabelled", tick)
		}
	}
	name := findText(n, "Rent")
	if name.Style.MaxLines != 1 {
		t.Error("a category name should be one line, cut when long")
	}
}

// ShowValues cells centre on their bars: the running weight at a cell's middle
// is its bar's middle, in slot units.
func TestBarValueCellsCentreOnBars(t *testing.T) {
	c := BarChart{Series: []ChartSeries{{Values: []float64{1, 2}}, {Values: []float64{3, math.NaN()}}}}
	texts, weights := c.barValueCells(2, 0.6, formatValue)
	// Per category: pad, two bars, pad.
	if len(texts) != 8 {
		t.Fatalf("texts %v", texts)
	}
	if texts[1] != "1" || texts[2] != "3" || texts[5] != "2" || texts[6] != "" {
		t.Errorf("texts = %q", texts)
	}
	s := valueScale{0, 3, 1}
	rects := c.barRects(2, 0.6, s)
	pos := 0.0
	for k, w := range weights {
		mid := (pos + w/2) * chartView / 2 // two slots across the viewBox
		pos += w
		var r barRect
		switch k {
		case 1:
			r = rects[0][0]
		case 2:
			r = rects[1][0]
		case 5:
			r = rects[0][1]
		default:
			continue
		}
		if barMid := (r.a0 + r.a1) / 2; math.Abs(mid-barMid) > 1e-9 {
			t.Errorf("cell %d centred at %v, bar at %v", k, mid, barMid)
		}
	}
	if math.Abs(pos-2) > 1e-9 {
		t.Errorf("weights sum to %v, want 2 slots", pos)
	}

	stacked := BarChart{Stacked: true, Series: c.Series}
	texts, _ = stacked.barValueCells(2, 0.6, formatValue)
	if len(texts) != 6 || texts[1] != "4" || texts[4] != "2" {
		t.Errorf("stacked totals = %q", texts)
	}
}

// A scatter draws every finite point as a dot in one path, labels its x ticks,
// and summarises extents rather than points.
func TestScatterChart(t *testing.T) {
	n := renderView(t, ScatterChart{
		Subject: "Fit",
		Series: []ScatterSeries{{Name: "Runs", Points: []ChartPoint{
			{1, 10}, {2, 30}, {math.NaN(), 5}, {4, 20},
		}}},
	})
	want := "Fit: Runs, 3 points, x from 1 to 4, y from 10 to 30."
	if n.Style.AccessibilityLabel != want {
		t.Errorf("label = %q\n want %q", n.Style.AccessibilityLabel, want)
	}
	c := canvasOf(n)
	var dots *core.Node
	for _, s := range c.Children {
		if s.Props["cap"] == string(core.CapRound) {
			dots = s
		}
	}
	if dots == nil {
		t.Fatal("no dot shape")
	}
	if d := dots.Props["d"].([]float64); len(d) != 3*6 {
		t.Errorf("dot path has %d numbers, want three zero-length strokes (18)", len(d))
	}
	if findText(n, "1") == nil || findText(n, "4") == nil {
		t.Error("x ticks 1 and 4 should be labelled")
	}
}

// An x label is one line in a slot with no minimum width, so a long label is
// cut rather than widening its slot.
func TestXLabelsAreCappedInTheirSlots(t *testing.T) {
	row := renderView(t, bandLabels(core.DefaultTheme, []string{"A very long category name", "B"}, 2))
	for _, slot := range row.Children {
		if slot.Style.MinWidth != "0px" {
			t.Errorf("slot MinWidth = %q, want 0px", slot.Style.MinWidth)
		}
		if slot.Children[0].Style.MaxLines != 1 {
			t.Error("an x label should be capped at one line")
		}
	}
}
