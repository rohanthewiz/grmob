package htmlout

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// chartNode is a chart's root as comps builds it: a RoleImg column with a
// summary label, here carrying d. Built from core rather than comps, which
// this package cannot import.
func chartNode(d *core.ChartData) *core.Node {
	props := []core.PropsAndChildren{
		core.AccessibilityRole(core.RoleImg),
		core.AccessibilityLabel("Visits: 2 points."),
		core.Text("Jan", core.AccessibilityHidden()),
	}
	if d != nil {
		props = append(props, core.AccessibilityChart(*d))
	}
	return core.Column(props...).Render(core.NewContext())
}

var visits = core.ChartData{
	Title: "Visits",
	X:     core.ChartAxis{Categories: []string{"Jan", "Feb", "Mar"}},
	Y:     core.ChartAxis{Min: 0, Max: 50},
	Series: []core.ChartDataSeries{
		{Name: "Web", Points: []core.ChartDataPoint{{X: "Jan", Y: 12, Text: "12k"}, {X: "Mar", Y: 45}}},
		{Name: "App", Points: []core.ChartDataPoint{{X: "Jan", Y: 3}, {X: "Feb", Y: 4.5}, {X: "Mar", Y: 6}}},
	},
}

// A categorical chart is a row per category and a column per series; a
// missing value is an empty cell, and a point's own Text wins over the number.
func TestChartTableRowsCategorical(t *testing.T) {
	caption, head, body := ChartTableRows(visits)
	if caption != "Visits" {
		t.Errorf("caption = %q", caption)
	}
	if want := []string{"", "Web", "App"}; !reflect.DeepEqual(head, want) {
		t.Errorf("head = %q, want %q", head, want)
	}
	want := [][]string{{"Jan", "12k", "3"}, {"Feb", "", "4.5"}, {"Mar", "45", "6"}}
	if !reflect.DeepEqual(body, want) {
		t.Errorf("body = %q, want %q", body, want)
	}
}

// A numeric x axis is a row per point, headed by its series.
func TestChartTableRowsNumeric(t *testing.T) {
	d := core.ChartData{
		X: core.ChartAxis{Title: "x", Min: 0, Max: 10},
		Y: core.ChartAxis{Title: "y", Min: 0, Max: 10},
		Series: []core.ChartDataSeries{{Name: "A", Points: []core.ChartDataPoint{
			{XValue: 1.5, XText: "1.5 kg", Y: 2}, {XValue: 3, Y: 4, Text: "4 cm"},
		}}},
	}
	_, head, body := ChartTableRows(d)
	if want := []string{"", "x", "y"}; !reflect.DeepEqual(head, want) {
		t.Errorf("head = %q, want %q", head, want)
	}
	want := [][]string{{"A", "1.5 kg", "2"}, {"A", "3", "4 cm"}}
	if !reflect.DeepEqual(body, want) {
		t.Errorf("body = %q, want %q", body, want)
	}
}

// The export: a chart with data is a figure whose first child is the hidden
// table; one without is the img it always was, with no table.
func TestAChartExportsAFigureWithItsTableFirst(t *testing.T) {
	// The export is indented; markup is compared with the space between
	// tags taken out.
	out := regexp.MustCompile(`>\s+<`).ReplaceAllString(ExportHTML(chartNode(&visits)), "><")
	if !strings.Contains(out, `role="figure"`) || strings.Contains(out, `role="img"`) {
		t.Fatalf("a chart with data should be a figure, not an img:\n%s", out)
	}
	table := strings.Index(out, `<table data-grmob-chrome="charttable"`)
	label := strings.Index(out, `aria-label="Visits: 2 points."`)
	child := strings.Index(out, ">Jan</span>")
	if table < 0 || label < 0 || child < 0 || !(label < table && table < child) {
		t.Fatalf("want the figure's label, then its table, then its own children:\n%s", out)
	}
	for _, want := range []string{
		`style="` + ChartTableStyle + `"`,
		"<caption>Visits</caption>",
		`<th scope="col">Web</th>`,
		`<th scope="row">Feb</th><td></td><td>4.5</td>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("export lacks %s:\n%s", want, out)
		}
	}

	plain := ExportHTML(chartNode(nil))
	if !strings.Contains(plain, `role="img"`) || strings.Contains(plain, "<table") {
		t.Errorf("a RoleImg with no data must be left alone:\n%s", plain)
	}
}

// Names are caller data and are escaped.
func TestChartTableEscapesNames(t *testing.T) {
	d := core.ChartData{Title: "<b>", X: core.ChartAxis{Categories: []string{"a&b"}},
		Series: []core.ChartDataSeries{{Name: `"q"`, Points: []core.ChartDataPoint{{X: "a&b", Y: 1}}}}}
	out := ExportHTML(chartNode(&d))
	if strings.Contains(out, "<caption><b>") || !strings.Contains(out, "a&amp;b") {
		t.Errorf("chart names reached the page unescaped:\n%s", out)
	}
}
