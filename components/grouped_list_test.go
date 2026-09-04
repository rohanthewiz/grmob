package components

import (
	"errors"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

// sermon is the test fixture: the shape of an archive row.
type sermon struct {
	ID    int
	Title string
	Month string // "2026-01"
}

var sermons = []sermon{
	{1, "Grace", "2026-01"},
	{2, "Truth", "2026-01"},
	{3, "Vine", "2025-12"},
	{4, "Light", "2025-11"},
	{5, "Salt", "2025-11"},
}

func byMonth(s sermon) Group       { return Group{Key: s.Month, Label: "Month " + s.Month} }
func sermonKey(s sermon) string    { return "sermon:" + itoa(s.ID) }
func sermonRow(s sermon) core.View { return core.Text(s.Title) }

// childKeys lists the reconciler keys of a node's children in order, "" for
// unkeyed ones, so a test can assert the header/row sequence in one line.
func childKeys(n *core.Node) []string {
	out := make([]string, 0, len(n.Children))
	for _, c := range n.Children {
		out = append(out, c.Key)
	}
	return out
}

func TestGroupRunsAreRunLengthInDisplayOrder(t *testing.T) {
	items := []sermon{{Month: "a"}, {Month: "a"}, {Month: "b"}, {Month: "a"}}
	runs := groupRuns(items, byMonth)
	if len(runs) != 3 {
		t.Fatalf("want 3 runs (a, b, a), got %d: %+v", len(runs), runs)
	}
	want := []struct {
		key        string
		start, end int
		count      int
	}{{"a", 0, 2, 2}, {"b", 2, 3, 1}, {"a", 3, 4, 1}}
	for i, w := range want {
		r := runs[i]
		if r.Group.Key != w.key || r.Start != w.start || r.End != w.end || r.Group.Count != w.count {
			t.Errorf("run %d = %+v, want key %s [%d,%d) count %d", i, r, w.key, w.start, w.end, w.count)
		}
	}
	if groupRuns[sermon](nil, byMonth) != nil || groupRuns(items, nil) != nil {
		t.Error("no items or no GroupBy should yield no runs")
	}
}

func TestGroupedListEmitsHeadersAndKeyedRows(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := GroupedList[sermon]{
		Items:   sermons,
		Key:     sermonKey,
		Row:     sermonRow,
		GroupBy: byMonth,
		Footer:  LoadMore{HasMore: true, OnLoadMore: func() {}},
	}.Render(ctx)

	if n.Type != "List" {
		t.Fatalf("GroupedList should render a virtualized List, got %q", n.Type)
	}
	got := strings.Join(childKeys(n), " ")
	want := "group:2026-01 sermon:1 sermon:2 group:2025-12 sermon:3 group:2025-11 sermon:4 sermon:5 "
	if got != want {
		t.Errorf("child keys\n got %q\nwant %q", got, want)
	}
	// The default header shows the label and the run's count.
	if findText(n, "Month 2026-01") == nil || findText(n.Children[0], "2") == nil {
		t.Error("first group header should carry its label and count badge")
	}
	// Shed the theme Column padding so rows are flush.
	if n.Style.Padding.Top != 0 || n.Style.Padding.Left != 0 || n.Style.Gap != 0 {
		t.Errorf("list should have zero padding and gap, got %+v gap %v", n.Style.Padding, n.Style.Gap)
	}
}

func TestGroupedListFlatWithoutGroupByAndPositionalKeys(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := GroupedList[sermon]{Items: sermons[:2], Row: sermonRow}.Render(ctx)
	got := strings.Join(childKeys(n), " ")
	if got != "row:0 row:1" {
		t.Errorf("flat list keys = %q, want positional row:0 row:1", got)
	}
}

func TestGroupedListDividersOnlyBetweenRowsOfAGroup(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := GroupedList[sermon]{
		Items: sermons[:3], Key: sermonKey, Row: sermonRow, GroupBy: byMonth, Dividers: true,
	}.Render(ctx)
	got := strings.Join(childKeys(n), " ")
	// A hairline between 1 and 2 (same group); none after 2 (a header
	// follows) and none after 3 (last row).
	want := "group:2026-01 sermon:1 sep:sermon:1 sermon:2 group:2025-12 sermon:3"
	if got != want {
		t.Errorf("keys\n got %q\nwant %q", got, want)
	}
}

func TestGroupedListEmptyStateKeepsFooter(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	err := errors.New("offline")
	n := GroupedList[sermon]{
		Row:    sermonRow,
		Empty:  core.Text("Nothing here"),
		Footer: LoadMore{Err: err, OnLoadMore: func() {}},
	}.Render(ctx)
	if findText(n, "Nothing here") == nil {
		t.Error("Empty view should render when there are no items")
	}
	if findText(n, "offline") == nil {
		t.Error("the footer's error state must stay reachable when the first page failed")
	}
}

func TestGroupedListHeaderOverride(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := GroupedList[sermon]{
		Items: sermons[:1], Key: sermonKey, Row: sermonRow, GroupBy: byMonth,
		Header: func(g Group) core.View { return core.Text("custom " + g.Label) },
	}.Render(ctx)
	if findText(n, "custom Month 2026-01") == nil {
		t.Error("Header override should replace the default GroupHeader")
	}
	if n.Children[0].Key != "group:2026-01" {
		t.Errorf("override keeps the group key, got %q", n.Children[0].Key)
	}
}

func TestGroupedListExportsHTML(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := GroupedList[sermon]{Items: sermons, Key: sermonKey, Row: sermonRow, GroupBy: byMonth}.Render(ctx)
	html := htmlout.ExportHTML(n)
	for _, want := range []string{"Month 2026-01", "Grace", "Salt"} {
		if !strings.Contains(html, want) {
			t.Errorf("exported HTML should contain %q", want)
		}
	}
}
