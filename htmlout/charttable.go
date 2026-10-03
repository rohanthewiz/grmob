package htmlout

import (
	"strconv"

	"github.com/rohanthewiz/element"
	"github.com/rohanthewiz/grmob/core"
)

// A chart's data table: the web half of core.AccessibilityChart (N-010).
//
// A chart node carrying a "chartData" prop is written as
//
//	<div role="figure" aria-label="…summary…">
//	  <table data-grmob-chrome="charttable" style="…visually hidden…">
//	    <caption>Visits</caption>
//	    <thead><tr><th scope="col"></th><th scope="col">Visits</th></tr></thead>
//	    <tbody><tr><th scope="row">Jan</th><td>12</td></tr> …</tbody>
//	  </table>
//	  …the chart's own children, every one aria-hidden…
//	</div>
//
// # Why figure, not img
//
// The chart asks for RoleImg everywhere, and on the natives that is still the
// answer. On the web an img's children are presentational: Chrome and Safari
// prune them from the accessibility tree, so a table inside role="img" is a
// table nobody can reach. role="figure" keeps the label (a figure is named by
// aria-label as an img is) and leaves its content in the tree. The swap is
// made only where the data is, so a RoleImg with no table is untouched.
//
// # Why first, and why chrome
//
// The table is not a node: it carries no data-node-path and no patch names
// it. The runtime marks it data-grmob-chrome and puts it first, which is the
// one place its node-index arithmetic already skips (chromeOffset); htmlout
// writes it in the same place so the two targets produce the same document.
// First is also the order a reader wants: the summary (the figure's name),
// then the numbers.
//
// # Visually hidden
//
// The established pattern, minus the negative margin some versions carry: a
// 1px box clipped to nothing, absolutely positioned at its static position so
// it takes no room in the chart's flex column and does not stick out of it.
// It is the reason this is web only — what a native reader does with a view
// that size is unmeasured; see core.ChartData.

// ChartTableStyle is the table's inline style. The runtime restates it as
// CHART_TABLE_STYLE; wasm/verify/chart_test.mjs holds them together.
const ChartTableStyle = "position:absolute;width:1px;height:1px;padding:0;border:0;" +
	"overflow:hidden;clip:rect(0 0 0 0);clip-path:inset(50%);white-space:nowrap"

// ChartDataOf returns the chart data a node carries, if any.
func ChartDataOf(props map[string]any) (core.ChartData, bool) {
	d, ok := props["chartData"].(core.ChartData)
	return d, ok
}

// ChartTableRows lays d out as a table: a caption, one header row, and the
// body rows, each body row's first cell being its row header.
//
// A categorical chart is a row per category and a column per series, which
// is how the chart is read: across a row, one category's value in every
// series. A numeric one (ScatterChart) is a row per point, headed by its
// series, with its x and y. A missing value is an empty cell. A point's own
// Text (the chart's formatting) is preferred to the bare number.
//
// The runtime restates this as chartTableRows; gen.go's chartCases computes
// Go's answer for real charts and chart_test.mjs compares.
func ChartTableRows(d core.ChartData) (caption string, head []string, body [][]string) {
	caption = d.Title
	if len(d.X.Categories) > 0 {
		head = append(head, d.X.Title)
		for _, s := range d.Series {
			head = append(head, s.Name)
		}
		for _, c := range d.X.Categories {
			row := []string{c}
			for _, s := range d.Series {
				row = append(row, categoryCell(s, c))
			}
			body = append(body, row)
		}
		return caption, head, body
	}
	head = []string{"", d.X.Title, d.Y.Title}
	for _, s := range d.Series {
		for _, p := range s.Points {
			x := p.XText
			if x == "" {
				x = plainNumber(p.XValue)
			}
			body = append(body, []string{s.Name, x, pointText(p)})
		}
	}
	return caption, head, body
}

// categoryCell is series s's value at category c, or "" when it has none.
// The first point naming c wins; a chart does not repeat a category.
func categoryCell(s core.ChartDataSeries, c string) string {
	for _, p := range s.Points {
		if p.X == c {
			return pointText(p)
		}
	}
	return ""
}

func pointText(p core.ChartDataPoint) string {
	if p.Text != "" {
		return p.Text
	}
	return plainNumber(p.Y)
}

// plainNumber writes v with as few digits as round-trip it. The runtime's
// String(v) agrees for every value a chart plots; the two part only at
// exponent notation (1e21 and up, 1e-7 and below), where comps always
// supplies Text anyway.
func plainNumber(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// figureRole returns attrs with role="img" changed to role="figure"; see
// "Why figure, not img". attrs is element's flat name/value list.
func figureRole(attrs []string) []string {
	out := append([]string(nil), attrs...)
	for i := 0; i+1 < len(out); i += 2 {
		if out[i] == "role" && out[i+1] == string(core.RoleImg) {
			out[i+1] = "figure"
		}
	}
	return out
}

// renderChartTable writes d's table at the builder's position.
func renderChartTable(b *element.Builder, d core.ChartData) {
	caption, head, body := ChartTableRows(d)
	t := b.Ele("table", "data-grmob-chrome", "charttable", "style", ChartTableStyle)
	if caption != "" {
		b.Ele("caption").TE(caption)
	}
	thead := b.Ele("thead")
	tr := b.Ele("tr")
	for _, h := range head {
		b.Ele("th", "scope", "col").TE(h)
	}
	tr.R()
	thead.R()
	tbody := b.Ele("tbody")
	for _, row := range body {
		tr := b.Ele("tr")
		for i, cell := range row {
			if i == 0 {
				b.Ele("th", "scope", "row").TE(cell)
				continue
			}
			b.Ele("td").TE(cell)
		}
		tr.R()
	}
	tbody.R()
	t.R()
}
