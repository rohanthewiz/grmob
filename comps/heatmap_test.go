package comps

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// subpaths counts the closed rectangles in a shape node's path — one per
// cell — from the wire form, where each core.PathMove starts a subpath.
// Walking it by opcode keeps a coordinate of 0 from being read as a MoveTo.
func subpaths(shape *core.Node) int {
	d, _ := shape.Props["d"].([]float64)
	n := 0
	for i := 0; i < len(d); {
		switch d[i] {
		case core.PathMove:
			n++
			i += 3
		case core.PathLine:
			i += 3
		case core.PathCubic:
			i += 7
		case core.PathClose:
			i++
		default:
			return -1
		}
	}
	return n
}

var orderGrid = Heatmap{
	Subject:      "Orders by hour",
	RowLabels:    []string{"Mon", "Tue", "Wed"},
	ColumnLabels: []string{"9", "12", "15", "18"},
	Values: [][]float64{
		{2, 8, 5, 1},
		{3, 9, 7, 2},
		{1, 4, math.NaN(), 0},
	},
}

// One shape per step plus one for no data, every cell drawn once, the top
// value in the top step and the NaN cell in Surface.
func TestHeatmapPaintsOneShapePerStep(t *testing.T) {
	_, n := renderDebug(t, orderGrid)

	if n.Style.AccessibilityRole != core.RoleImg {
		t.Errorf("role = %q, want img", n.Style.AccessibilityRole)
	}
	want := "Orders by hour: 3 rows by 4 columns; low 0 at Wed, 18; high 9 at Tue, 12; 1 cell with no data."
	if got := n.Style.AccessibilityLabel; got != want {
		t.Errorf("summary = %q\nwant      %q", got, want)
	}

	canvas := canvasOf(n)
	scale := core.DefaultTheme.Colors.SequentialColors()
	if len(canvas.Children) != len(scale)+1 {
		t.Fatalf("shapes = %d, want one per step and one for no data", len(canvas.Children))
	}
	if fill := canvas.Children[0].Props["fill"]; fill != core.DefaultTheme.Colors.Surface {
		t.Errorf("no-data fill = %v, want Surface", fill)
	}
	if got := subpaths(canvas.Children[0]); got != 1 {
		t.Errorf("no-data cells = %d, want the one NaN", got)
	}
	cells := 0
	for i, s := range canvas.Children[1:] {
		if k := subpaths(s); k > 0 {
			cells += k
			if s.Props["fill"] != scale[i] {
				t.Errorf("step %d fill = %v, want %s", i, s.Props["fill"], scale[i])
			}
		}
	}
	if cells != 11 {
		t.Errorf("valued cells = %d, want 11", cells)
	}
	// Five equal steps of 1.8 over 0–9: the top step, from 7.2, holds 8 and
	// 9; the bottom, below 1.8, holds 0 and both 1s.
	if got := subpaths(canvas.Children[len(scale)]); got != 2 {
		t.Errorf("top-step cells = %d, want the 8 and the 9", got)
	}
	if got := subpaths(canvas.Children[1]); got != 3 {
		t.Errorf("bottom-step cells = %d, want 0, 1 and 1", got)
	}
	if canvas.Style.Height != "60px" {
		t.Errorf("canvas height = %s, want three 20px rows", canvas.Style.Height)
	}
}

func TestHeatmapStepEdges(t *testing.T) {
	for _, c := range []struct {
		v    float64
		want int
	}{{0, 0}, {1.9, 0}, {2, 1}, {9.99, 4}, {10, 4}, {-1, 0}, {11, 4}} {
		if got := heatStep(c.v, 0, 10, 5); got != c.want {
			t.Errorf("heatStep(%v) = %d, want %d", c.v, got, c.want)
		}
	}
	if got := heatStep(3, 3, 3, 5); got != 4 {
		t.Errorf("a flat range = step %d, want the top", got)
	}
}

// An empty label gives its slot to the label before it: "Mar" over three
// weeks owns three slots and starts at the first.
func TestHeatmapSpanLabels(t *testing.T) {
	row := spanLabels(core.DefaultTheme, []string{"", "Mar", "", "", "Apr"}, 5).Render(core.NewContext())
	var weights []float64
	var texts []string
	for _, c := range row.Children {
		weights = append(weights, c.Style.FlexGrow)
		texts = append(texts, c.Children[0].Props["content"].(string))
	}
	if strings.Join(texts, "|") != "|Mar|Apr" {
		t.Fatalf("labels = %q", texts)
	}
	if weights[0] != 1 || weights[1] != 3 || weights[2] != 1 {
		t.Errorf("weights = %v, want the filler 1, Mar 3, Apr 1", weights)
	}
	if row.Children[1].Children[0].Style.Align != core.AlignStart {
		t.Error("a spanning label should start at its first slot")
	}
}

// 17 weeks ending on a Wednesday: the days after it are not drawn, empty days
// are no data, and the summary counts days and names the busiest.
func TestCalendarHeatmapLaysDaysIntoWeeks(t *testing.T) {
	end := time.Date(2026, time.March, 11, 18, 0, 0, 0, time.UTC) // a Wednesday
	days := []DayValue{
		{Day: time.Date(2026, time.March, 3, 9, 0, 0, 0, time.UTC), Value: 2},
		{Day: time.Date(2026, time.March, 3, 20, 0, 0, 0, time.UTC), Value: 2}, // same day, summed
		{Day: time.Date(2026, time.March, 10, 8, 0, 0, 0, time.UTC), Value: 1},
		{Day: time.Date(2026, time.March, 12, 8, 0, 0, 0, time.UTC), Value: 9},  // after End: dropped
		{Day: time.Date(2025, time.January, 1, 8, 0, 0, 0, time.UTC), Value: 9}, // before the grid
	}
	_, n := renderDebug(t, CalendarHeatmap{Subject: "Workouts", Days: days, End: end})

	want := "Workouts: 5 over 17 weeks, on 2 days; most on Tue 3 Mar 2026, 4."
	if got := n.Style.AccessibilityLabel; got != want {
		t.Errorf("summary = %q\nwant      %q", got, want)
	}
	canvas := canvasOf(n)
	drawn := 0
	for _, s := range canvas.Children {
		drawn += max(0, subpaths(s))
	}
	// 17 weeks of 7, less Thursday to Saturday of the last week.
	if drawn != 17*7-3 {
		t.Errorf("cells drawn = %d, want %d", drawn, 17*7-3)
	}
	if got := subpaths(canvas.Children[0]); got != drawn-2 {
		t.Errorf("no-data days = %d, want every drawn day but the two active", got)
	}
	if canvas.Style.Height != "98px" {
		t.Errorf("canvas height = %s, want seven 14px rows", canvas.Style.Height)
	}
	for _, s := range []string{"Mon", "Wed", "Fri", "Dec", "Jan", "Feb", "Mar"} {
		if findText(n, s) == nil {
			t.Errorf("label %q missing", s)
		}
	}
	if findText(n, "Nov") != nil {
		t.Error("the first column (from 16 Nov) should not be named when December starts two columns later")
	}
}

// WeeksFor makes square cells: a 360px phone less 32px of screen padding at
// the 14px default is (328−30)/14 → 21 weeks; a wide window caps at a year and
// a tiny one floors at one week. A taller cell fits fewer.
func TestCalendarHeatmapWeeksForSquaresTheCells(t *testing.T) {
	for _, tc := range []struct {
		cell, width float64
		want        int
	}{
		{0, 328, 21},
		{0, 2000, 53},
		{0, 10, 1},
		{20, 328, 14},
	} {
		if got := (CalendarHeatmap{CellHeight: tc.cell}).WeeksFor(tc.width); got != tc.want {
			t.Errorf("CellHeight %v, width %v: WeeksFor = %d, want %d", tc.cell, tc.width, got, tc.want)
		}
	}
}

// Two days tied for the most: the summary names the earlier, whichever order
// the entries arrive in, since the grid is walked oldest first.
func TestCalendarHeatmapTieNamesTheEarliestDay(t *testing.T) {
	end := time.Date(2026, time.March, 11, 18, 0, 0, 0, time.UTC)
	early := DayValue{Day: time.Date(2026, time.February, 17, 9, 0, 0, 0, time.UTC), Value: 3}
	late := DayValue{Day: time.Date(2026, time.March, 9, 9, 0, 0, 0, time.UTC), Value: 3}
	for _, days := range [][]DayValue{{early, late}, {late, early}} {
		_, n := renderDebug(t, CalendarHeatmap{Subject: "Runs", Days: days, End: end})
		want := "Runs: 6 over 17 weeks, on 2 days; most on Tue 17 Feb 2026, 3."
		if got := n.Style.AccessibilityLabel; got != want {
			t.Errorf("summary = %q\nwant      %q", got, want)
		}
	}
}
