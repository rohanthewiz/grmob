package components

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

var sermonColumns = []Column[sermon]{
	{Title: "Title", Text: func(s sermon) string { return s.Title }, Weight: 2,
		Less: func(a, b sermon) bool { return a.Title < b.Title }},
	{Title: "Month", Text: func(s sermon) string { return s.Month }, Narrow: true},
	{Title: "ID", Cell: func(s sermon) core.View { return Badge{Text: itoa(s.ID)} },
		Align: core.JustifyEnd, Sortable: true},
}

// tableParts splits a rendered DataTable into its header row and body list.
func tableParts(t *testing.T, n *core.Node) (header, body *core.Node) {
	t.Helper()
	if n.Type != "Column" || len(n.Children) < 2 {
		t.Fatalf("DataTable should be a Column of header + body, got %q with %d children", n.Type, len(n.Children))
	}
	header, body = n.Children[0], n.Children[1]
	if header.Type != "Row" || body.Type != "List" {
		t.Fatalf("want Row header and List body, got %q and %q", header.Type, body.Type)
	}
	return header, body
}

func TestDataTableHeaderCellsAndBodyRows(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := DataTable[sermon]{Columns: sermonColumns, Rows: sermons, Key: sermonKey}.Render(ctx)
	header, body := tableParts(t, n)

	if len(header.Children) != 3 {
		t.Fatalf("header should have one cell per column, got %d", len(header.Children))
	}
	// Weight becomes FlexGrow; a weightless column hugs its content; Align
	// becomes flex justification on the cell.
	if header.Children[0].Style.FlexGrow != 2 || header.Children[1].Style.FlexGrow != 0 {
		t.Errorf("cell FlexGrow = %v, %v; want 2, 0", header.Children[0].Style.FlexGrow, header.Children[1].Style.FlexGrow)
	}
	if header.Children[2].Style.JustifyContent != core.JustifyEnd {
		t.Errorf("Align should set the cell's justification, got %q", header.Children[2].Style.JustifyContent)
	}

	if got := strings.Join(childKeys(body), " "); got != "sermon:1 sermon:2 sermon:3 sermon:4 sermon:5" {
		t.Errorf("body keys = %q", got)
	}
	row := body.Children[0]
	if len(row.Children) != 3 || findText(row, "Grace") == nil || findText(row, "1") == nil {
		t.Error("a body row should carry one cell per column with Text and Cell content")
	}
}

func TestDataTableSortIsControlledAndToggles(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	var asked Sort
	n := DataTable[sermon]{
		Columns: sermonColumns, Rows: sermons, Key: sermonKey,
		Sort:   &Sort{Column: 0},
		OnSort: func(s Sort) { asked = s },
	}.Render(ctx)
	header, body := tableParts(t, n)

	// Client-side sort on the column with a comparator, ascending.
	if got := strings.Join(childKeys(body), " "); got != "sermon:1 sermon:4 sermon:5 sermon:2 sermon:3" {
		t.Errorf("rows should be sorted by title ascending, got %q", got)
	}
	if findText(header, "Title ▲") == nil {
		t.Error("active column shows the ascending glyph")
	}

	// Month has neither Less nor Sortable: no handler. Title and ID do.
	if _, ok := header.Children[1].Props["onClick"]; ok {
		t.Error("an unsortable column must not register a handler")
	}
	ctx.TriggerCallback(header.Children[0].Props["onClick"].(string))
	if asked != (Sort{Column: 0, Desc: true}) {
		t.Errorf("tapping the active column toggles direction, got %+v", asked)
	}
	ctx.TriggerCallback(header.Children[2].Props["onClick"].(string))
	if asked != (Sort{Column: 2}) {
		t.Errorf("tapping another column starts ascending, got %+v", asked)
	}

	// Descending, and the caller's slice is untouched by the sort.
	ctx.BeginRenderPass()
	n = DataTable[sermon]{Columns: sermonColumns, Rows: sermons, Key: sermonKey,
		Sort: &Sort{Column: 0, Desc: true}, OnSort: func(Sort) {}}.Render(ctx)
	_, body = tableParts(t, n)
	if got := strings.Join(childKeys(body), " "); got != "sermon:3 sermon:2 sermon:5 sermon:4 sermon:1" {
		t.Errorf("descending order = %q", got)
	}
	if sermons[0].ID != 1 || sermons[4].ID != 5 {
		t.Error("sorting must not reorder the caller's Rows")
	}
}

func TestDataTableSortableWithoutComparatorReportsOnly(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := DataTable[sermon]{Columns: sermonColumns, Rows: sermons, Key: sermonKey,
		Sort: &Sort{Column: 2, Desc: true}, OnSort: func(Sort) {}}.Render(ctx)
	header, body := tableParts(t, n)
	if findText(header, "ID ▼") == nil {
		t.Error("a server-sorted column still shows its glyph")
	}
	if got := strings.Join(childKeys(body), " "); got != "sermon:1 sermon:2 sermon:3 sermon:4 sermon:5" {
		t.Errorf("without Less the rows keep the caller's order, got %q", got)
	}
}

func TestDataTableCompactDropsNarrowColumnsButKeepsIndices(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	var asked Sort
	n := DataTable[sermon]{Columns: sermonColumns, Rows: sermons[:1], Key: sermonKey,
		Compact: true, Sort: &Sort{Column: 2}, OnSort: func(s Sort) { asked = s }}.Render(ctx)
	header, body := tableParts(t, n)
	if len(header.Children) != 2 || findText(header, "Month") != nil {
		t.Fatalf("compact header should drop the Narrow column, got %d cells", len(header.Children))
	}
	if len(body.Children[0].Children) != 2 {
		t.Error("body rows must drop the same column as the header")
	}
	// ID is the caller's column 2 even though it is the second visible cell.
	if findText(header, "ID ▲") == nil {
		t.Error("Sort.Column addresses the caller's column list, not the visible one")
	}
	ctx.TriggerCallback(header.Children[1].Props["onClick"].(string))
	if asked.Column != 2 {
		t.Errorf("OnSort reports the caller's column index, got %d", asked.Column)
	}
}

func TestDataTableClientSidePaging(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := DataTable[sermon]{Columns: sermonColumns, Rows: sermons, Key: sermonKey,
		Pagination: &Pagination{Page: 1, PageSize: 2, OnChange: func(int) {}}}.Render(ctx)
	_, body := tableParts(t, n)
	if got := strings.Join(childKeys(body), " "); got != "sermon:3 sermon:4" {
		t.Errorf("page 1 of size 2 = %q", got)
	}
	if findText(n, "Page 2 of 3") == nil {
		t.Error("the footer should derive the page count from the rows")
	}

	// A stale page past the end lands on the last page.
	ctx.BeginRenderPass()
	n = DataTable[sermon]{Columns: sermonColumns, Rows: sermons, Key: sermonKey,
		Pagination: &Pagination{Page: 9, PageSize: 2}}.Render(ctx)
	_, body = tableParts(t, n)
	if got := strings.Join(childKeys(body), " "); got != "sermon:5" {
		t.Errorf("clamped last page = %q", got)
	}
	if findText(n, "Page 3 of 3") == nil {
		t.Error("clamped footer should read Page 3 of 3")
	}
}

func TestDataTableServerSidePagingLeavesRowsAlone(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := DataTable[sermon]{Columns: sermonColumns, Rows: sermons[:2], Key: sermonKey,
		Pagination: &Pagination{Page: 3, PageSize: 2, PageCount: 10}}.Render(ctx)
	_, body := tableParts(t, n)
	if len(body.Children) != 2 || findText(n, "Page 4 of 10") == nil {
		t.Error("with PageCount set the caller owns paging: rows and label pass through")
	}
}

// The partial-sort concern: sorting here, over rows the caller has said the
// server pages, is the one shape of the mistake the table can detect.
//
// The three negative cases matter as much as the positive one — a concern
// that fires on a correct table is worse than no concern at all, since the
// list it lands in is meant to be read as "these are bugs".
func TestDataTableReportsSortingOverAServerPagedWindow(t *testing.T) {
	core.SetDebugMode(true)
	defer core.SetDebugMode(false)

	// PageCount declared by the caller: the server picked these rows, and
	// the Title column carries a Less, so the table sorts a window.
	core.ClearConcerns()
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	DataTable[sermon]{
		Columns: sermonColumns, Rows: sermons, Key: sermonKey,
		Sort:       &Sort{Column: 0},
		Pagination: &Pagination{Page: 0, PageCount: 9, OnChange: func(int) {}},
	}.Render(ctx)
	if !hasConcern(ConcernPartialSort) {
		t.Error("sorting client-side under a server-declared PageCount should be reported")
	}

	// Every arrangement that is actually correct, each differing from the
	// case above in one thing only.
	fine := []struct {
		name  string
		table DataTable[sermon]
	}{
		{"client-side paging: the table holds every row and slices them itself",
			DataTable[sermon]{Columns: sermonColumns, Rows: sermons, Key: sermonKey,
				Sort:       &Sort{Column: 0},
				Pagination: &Pagination{Page: 0, PageSize: 2, OnChange: func(int) {}}}},
		{"server paging with Sortable, not Less: the tap goes to the query",
			DataTable[sermon]{Columns: sermonColumns, Rows: sermons, Key: sermonKey,
				Sort:       &Sort{Column: 2}, // the ID column: Sortable, no Less
				Pagination: &Pagination{Page: 0, PageCount: 9, OnChange: func(int) {}}}},
		{"server paging with no active sort at all",
			DataTable[sermon]{Columns: sermonColumns, Rows: sermons, Key: sermonKey,
				Pagination: &Pagination{Page: 0, PageCount: 9, OnChange: func(int) {}}}},
	}
	for _, c := range fine {
		core.ClearConcerns()
		ctx.BeginRenderPass()
		c.table.Render(ctx)
		if hasConcern(ConcernPartialSort) {
			t.Errorf("false positive — %s", c.name)
		}
	}

	// Off in a release build: the check is behind IsDebugMode, so nothing is
	// recorded and nothing is built to record.
	core.SetDebugMode(false)
	core.ClearConcerns()
	ctx.BeginRenderPass()
	DataTable[sermon]{
		Columns: sermonColumns, Rows: sermons, Key: sermonKey,
		Sort:       &Sort{Column: 0},
		Pagination: &Pagination{Page: 0, PageCount: 9, OnChange: func(int) {}},
	}.Render(ctx)
	if hasConcern(ConcernPartialSort) {
		t.Error("the check should cost nothing and record nothing with debug off")
	}
}

// hasConcern reports whether the collector holds a concern of this kind.
func hasConcern(kind string) bool {
	for _, c := range core.Concerns() {
		if c.Kind == kind {
			return true
		}
	}
	return false
}

// The trailing group's count is suppressed here for the same reason as in
// GroupedList: a table fed by an append-style pager holds only the rows
// loaded so far, so its last run is still open. Pinned separately because
// DataTable reaches appendRows down a different path (its own cell renderer
// and row wrapper), and a dropped argument there would go unnoticed.
func TestDataTableHidesTheTrailingCountWhenMoreMayFollow(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	table := DataTable[sermon]{
		Columns: sermonColumns, Rows: sermons, Key: sermonKey, GroupBy: byMonth,
		HideTrailingCount: true,
	}
	_, body := tableParts(t, table.Render(ctx))

	// Body children: header, 2 rows, header, 1 row, header, 2 rows.
	if findText(body.Children[0], "2") == nil {
		t.Error("a closed group header lost its count badge")
	}
	trailing := body.Children[5]
	if findText(trailing, "Month 2025-11") == nil {
		t.Fatalf("child 5 is not the trailing header: %+v", trailing)
	}
	if findText(trailing, "2") != nil {
		t.Error("the open trailing group published a count the next page can invalidate")
	}

	ctx.EndRenderPass()
	ctx.BeginRenderPass()
	table.HideTrailingCount = false
	_, body = tableParts(t, table.Render(ctx))
	if findText(body.Children[5], "2") == nil {
		t.Error("with nothing more to load the trailing group should carry its count")
	}
}

func TestDataTableRowTapSelectionGroupingAndStates(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	var tapped sermon
	n := DataTable[sermon]{
		Columns: sermonColumns, Rows: sermons, Key: sermonKey,
		GroupBy:  byMonth,
		OnRowTap: func(s sermon) { tapped = s },
		Selected: func(s sermon) bool { return s.ID == 3 },
		Dividers: true,
	}.Render(ctx)
	_, body := tableParts(t, n)
	got := strings.Join(childKeys(body), " ")
	want := "group:2026-01 sermon:1 sep:sermon:1 sermon:2 group:2025-12 sermon:3 group:2025-11 sermon:4 sep:sermon:4 sermon:5"
	if got != want {
		t.Errorf("grouped body keys\n got %q\nwant %q", got, want)
	}
	row3 := findFirst(body, func(n *core.Node) bool { return n.Key == "sermon:3" })
	if row3.Style.Background != core.DefaultTheme.Colors.Surface {
		t.Errorf("selected row should take the Surface tint, got %q", row3.Style.Background)
	}
	ctx.TriggerCallback(row3.Props["onClick"].(string))
	if tapped.ID != 3 {
		t.Errorf("OnRowTap should receive the row, got %+v", tapped)
	}

	ctx.BeginRenderPass()
	n = DataTable[sermon]{Columns: sermonColumns, Rows: sermons, Key: sermonKey,
		Loading: core.Text("busy")}.Render(ctx)
	_, body = tableParts(t, n)
	if len(body.Children) != 1 || findText(body, "busy") == nil {
		t.Error("Loading replaces the body rows and keeps the header")
	}

	ctx.BeginRenderPass()
	n = DataTable[sermon]{Columns: sermonColumns, Key: sermonKey, Empty: core.Text("none")}.Render(ctx)
	_, body = tableParts(t, n)
	if findText(body, "none") == nil {
		t.Error("Empty renders in the body when there are no rows")
	}
	if _, ok := body.Children[0].Props["onClick"]; ok {
		t.Error("a presentational table registers no row handlers")
	}
}

func TestDataTableExportsHTML(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := DataTable[sermon]{Columns: sermonColumns, Rows: sermons, Key: sermonKey,
		Pagination: &Pagination{PageSize: 3}}.Render(ctx)
	html := htmlout.ExportHTML(n)
	for _, want := range []string{"Title", "Grace", "Vine", "Page 1 of 2"} {
		if !strings.Contains(html, want) {
			t.Errorf("exported HTML should contain %q", want)
		}
	}
	if strings.Contains(html, "Light") {
		t.Error("rows beyond the page must not be exported")
	}
}

// StickyHeaders pins the *group* bands, as in GroupedList. The column header
// is deliberately not among them: it is a sibling of the body list rather
// than a row inside it — which is what keeps it from scrolling away in the
// first place — and both natives implement pinning inside their lazy
// container only.
func TestDataTableStickyHeadersPinTheGroupBandsOnly(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := DataTable[sermon]{
		Rows:          sermons,
		Key:           sermonKey,
		Columns:       []Column[sermon]{{Title: "Title", Text: func(s sermon) string { return s.Title }}},
		GroupBy:       byMonth,
		StickyHeaders: true,
	}.Render(ctx)

	body := findFirst(n, func(c *core.Node) bool { return c.Type == "List" })
	if body == nil {
		t.Fatal("no body List rendered")
	}
	bands := 0
	for _, child := range body.Children {
		if !strings.HasPrefix(child.Key, "group:") {
			continue
		}
		bands++
		if child.Style == nil || child.Style.Position != core.PositionSticky {
			t.Errorf("band %q is not sticky: %+v", child.Key, child.Style)
		}
	}
	if bands == 0 {
		t.Fatal("no group bands rendered")
	}
	// The column header sits outside the list, so nothing there is pinned.
	for _, child := range n.Children {
		if child.Type == "List" {
			continue
		}
		if child.Style != nil && child.Style.Position == core.PositionSticky {
			t.Error("the column header was pinned — no native renderer can honor that, " +
				"so it would be a web-only behavior sold as a cross-platform one")
		}
	}
}
