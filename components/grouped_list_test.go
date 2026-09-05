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

// The counts on an append-paged list: every closed group carries one, the
// open one at the tail does not.
//
// This is the case a server-paged feed hits on every render — the trailing
// run is only as long as the pages loaded so far — so it is worth pinning
// that the badge disappears from exactly one header and that the rest are
// untouched.
func TestGroupedListHidesOnlyTheTrailingCountWhenMoreMayFollow(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	list := GroupedList[sermon]{
		Items: sermons, Key: sermonKey, Row: sermonRow, GroupBy: byMonth,
		HideTrailingCount: true,
	}
	n := list.Render(ctx)

	// Headers are children 0, 3 and 5 (see the key sequence above): counts 2,
	// 1 and 2. The last group is open, so its badge is gone and its label
	// stands alone.
	headers := []struct {
		child int
		count string
	}{{0, "2"}, {3, "1"}}
	for _, h := range headers {
		if findText(n.Children[h.child], h.count) == nil {
			t.Errorf("closed group header %d lost its count badge %q", h.child, h.count)
		}
	}
	trailing := n.Children[5]
	if findText(trailing, "Month 2025-11") == nil {
		t.Fatalf("child 5 is not the trailing header: %+v", trailing)
	}
	if findText(trailing, "2") != nil {
		t.Error("the open trailing group published a count the next page can invalidate")
	}

	// The flag off is the complete-list case: every group counts, including
	// the last.
	ctx.EndRenderPass()
	ctx.BeginRenderPass()
	list.HideTrailingCount = false
	if findText(list.Render(ctx).Children[5], "2") == nil {
		t.Error("with nothing more to load the trailing group should carry its count")
	}
}

// A Header override is handed the Group untouched, trailing or not: the
// widget cannot reach inside a view the caller built, so the caller owns the
// decision. Pinned because the alternative — zeroing Count for the trailing
// run — would silently make an override print "0".
func TestGroupedListTrailingCountLeavesAHeaderOverrideAlone(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := GroupedList[sermon]{
		Items: sermons, Key: sermonKey, Row: sermonRow, GroupBy: byMonth,
		HideTrailingCount: true,
		Header: func(g Group) core.View {
			return core.Text(g.Label + " has " + itoa(g.Count))
		},
	}.Render(ctx)
	if findText(n, "Month 2025-11 has 2") == nil {
		t.Error("an override should still see the trailing group's real Count")
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

// --- roadmap tier B, seen from the widget ------------------------------------

// StickyHeaders puts core.StickyHeader on each default band. The marker has
// to land on the band node itself — the child the List actually sees — since
// a wrapper would put a plain Box there with the sticky row hidden inside it,
// where no renderer looks.
func TestGroupedListStickyHeadersPinTheBands(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := GroupedList[string]{
		Items:         []string{"a", "b", "c"},
		Key:           func(s string) string { return s },
		Row:           func(s string) core.View { return core.Text(s) },
		GroupBy:       func(s string) Group { return Group{Key: "g" + s, Label: s} },
		StickyHeaders: true,
	}.Render(ctx)

	bands := 0
	for _, child := range n.Children {
		if !strings.HasPrefix(child.Key, "group:") {
			continue
		}
		bands++
		if child.Style == nil || child.Style.Position != core.PositionSticky {
			t.Errorf("band %q is not sticky: %+v", child.Key, child.Style)
		}
	}
	if bands != 3 {
		t.Fatalf("found %d bands, want 3", bands)
	}
	// Rows are not pinned — only the bands.
	for _, child := range n.Children {
		if strings.HasPrefix(child.Key, "group:") {
			continue
		}
		if child.Style != nil && child.Style.Position == core.PositionSticky {
			t.Errorf("row %q was pinned along with the bands", child.Key)
		}
	}
}

// Off by default, so every list that predates the option renders exactly as
// it did.
func TestGroupedListBandsAreNotStickyByDefault(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := GroupedList[string]{
		Items:   []string{"a"},
		Row:     func(s string) core.View { return core.Text(s) },
		GroupBy: func(s string) Group { return Group{Key: "g", Label: "G"} },
	}.Render(ctx)

	band := findFirst(n, func(c *core.Node) bool { return strings.HasPrefix(c.Key, "group:") })
	if band == nil {
		t.Fatal("no band rendered")
	}
	if band.Style != nil && band.Style.Position != "" {
		t.Errorf("band is positioned without StickyHeaders: %q", band.Style.Position)
	}
}

// A Header override is handed the Group and builds its own view, which the
// widget cannot reach into — the same division HideTrailingCount draws. Such
// a header pins itself by putting core.StickyHeader() in its own Style, and
// the flag must not silently half-apply.
func TestGroupedListStickyHeadersLeavesAnOverrideAlone(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := GroupedList[string]{
		Items:         []string{"a"},
		Row:           func(s string) core.View { return core.Text(s) },
		GroupBy:       func(s string) Group { return Group{Key: "g", Label: "G"} },
		Header:        func(g Group) core.View { return core.Row(core.Text(g.Label)) },
		StickyHeaders: true,
	}.Render(ctx)

	band := findFirst(n, func(c *core.Node) bool { return strings.HasPrefix(c.Key, "group:") })
	if band == nil {
		t.Fatal("no band rendered")
	}
	if band.Style != nil && band.Style.Position == core.PositionSticky {
		t.Error("the widget pinned a caller's own header view — an override owns its own Style")
	}
}

// OnEndReached puts core.OnEndReached on the List, which is what turns the
// footer's "Load more" button into an infinite feed.
func TestGroupedListOnEndReachedReachesTheList(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	var loaded int
	n := GroupedList[string]{
		Items:        []string{"a", "b"},
		Row:          func(s string) core.View { return core.Text(s) },
		OnEndReached: func() { loaded++ },
	}.Render(ctx)

	id, _ := n.Props["onEndReached"].(string)
	if id == "" {
		t.Fatalf("the List carries no onEndReached prop: %#v", n.Props)
	}
	ctx.TriggerCallback(id)
	if loaded != 1 {
		t.Fatalf("handler ran %d times, want 1", loaded)
	}
}

// Nil leaves the list exactly as it was: a manual pager driven by its footer,
// with no edge reported and no prop on the node.
func TestGroupedListWithoutOnEndReachedCarriesNoProp(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := GroupedList[string]{
		Items: []string{"a"},
		Row:   func(s string) core.View { return core.Text(s) },
	}.Render(ctx)

	if _, ok := n.Props["onEndReached"]; ok {
		t.Error("a list with no OnEndReached still advertises the edge")
	}
}

// A band titles a run of rows, so its label is a heading — the thing a reader
// navigating a long banded feed by heading moves between once the screen's own
// title has scrolled away.
//
// The role is on the label and not on the band, so the heading's name is
// "Month 2026-01" and not "Month 2026-01, 2". The count is still there, as the
// separate thing it is.
func TestGroupHeaderLabelIsAHeading(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := GroupedList[sermon]{
		Items: sermons, Key: sermonKey, Row: sermonRow, GroupBy: byMonth,
	}.Render(ctx)

	label := findText(n, "Month 2026-01")
	if label == nil {
		t.Fatal("no band label in the rendered list")
	}
	if label.Style.AccessibilityRole != core.RoleHeading {
		t.Errorf("band label = %q, want heading", label.Style.AccessibilityRole)
	}
	band := n.Children[0]
	if band.Style.AccessibilityRole != core.RoleNone {
		t.Errorf("the band itself = %q; the role belongs on the label, so the count "+
			"badge stays out of the heading's name", band.Style.AccessibilityRole)
	}
}
