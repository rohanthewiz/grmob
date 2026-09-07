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

// A shut trailing group withholds the edge sensor, and opening it gives it
// back.
//
// # What the failure looks like without this
//
// An append pager extends the last run and a shut run emits nothing, so a page
// that lands in a collapsed trailing group is invisible. core.OnEndReached
// then refuses to fire again — its guard is the List's child count, which a
// hidden page does not move — so the *first* auto-load spends a page and every
// one after it is swallowed. The feed reads as exhausted while the pager's
// offset has quietly moved on.
//
// So the prop is withheld rather than left to starve, and the Footer stays as
// the deliberate way to ask.
func TestGroupedListWithholdsTheEdgeWhileTheLastRunIsShut(t *testing.T) {
	rows := []string{"jan-1", "jan-2", "feb-1"}
	byMonth := func(s string) Group {
		k := s[:3]
		return Group{Key: k, Label: k}
	}

	render := func(shut map[string]bool) *core.Node {
		ctx := core.NewContext()
		ctx.BeginRenderPass()
		return GroupedList[string]{
			Items:        rows,
			Key:          func(s string) string { return s },
			Row:          func(s string) core.View { return core.Text(s) },
			GroupBy:      byMonth,
			OnEndReached: func() {},
			Collapse: Collapse{
				IsCollapsed: func(g Group) bool { return shut[g.Key] },
				OnToggle:    func(Group) {},
			},
		}.Render(ctx)
	}

	if _, ok := render(map[string]bool{"feb": true}).Props["onEndReached"]; ok {
		t.Error("the last group is shut and the list still advertises its edge — a " +
			"page fetched now lands in a hidden run, and the guard then refuses " +
			"every fire after it")
	}
	if _, ok := render(map[string]bool{}).Props["onEndReached"]; !ok {
		t.Error("nothing is shut and the edge is gone — opening the run has to give " +
			"the auto-load back, or a reader who collapses a group once loses " +
			"infinite scrolling for the session")
	}
	// A shut group *above* the last one changes nothing: the pager was never
	// going to extend it, so its rows being hidden is not the pager's problem.
	if _, ok := render(map[string]bool{"jan": true}).Props["onEndReached"]; !ok {
		t.Error("a shut group above the trailing one withheld the edge — the run a " +
			"page lands in is open, so there is nothing to withhold")
	}
}

// AutoLoadWithheld answers the question the tree cannot be asked.
//
// The withholding above is right and it is silent: a feed that stopped fetching
// because the last run is shut and a feed that has genuinely run out produce
// the same tree and the same experience. This method is the only way a caller
// can tell them apart, and the property that makes it worth having is that it
// agrees with the tree — a method that said "withheld" while the prop was still
// on the List would send a screen into showing two ways to load the next page.
func TestAutoLoadWithheldAgreesWithTheRenderedList(t *testing.T) {
	rows := []string{"jan-1", "jan-2", "feb-1"}
	byMonth := func(s string) Group { return Group{Key: s[:3], Label: s[:3]} }

	base := func(shut map[string]bool) GroupedList[string] {
		return GroupedList[string]{
			Items:   rows,
			Key:     func(s string) string { return s },
			Row:     func(s string) core.View { return core.Text(s) },
			GroupBy: byMonth,
			Collapse: Collapse{
				IsCollapsed: func(g Group) bool { return shut[g.Key] },
				OnToggle:    func(Group) {},
			},
		}
	}

	for _, c := range []struct {
		name string
		shut map[string]bool
		want bool
	}{
		{"the trailing run is shut", map[string]bool{"feb": true}, true},
		{"nothing is shut", map[string]bool{}, false},
		{"a run above the trailing one is shut", map[string]bool{"jan": true}, false},
		{"every run is shut", map[string]bool{"jan": true, "feb": true}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			list := base(c.shut)
			list.OnEndReached = func() {}
			if got := list.AutoLoadWithheld(); got != c.want {
				t.Errorf("AutoLoadWithheld() = %v, want %v", got, c.want)
			}
			// And the tree agrees, which is the whole claim: the answer is
			// worth nothing if it is not the same answer Render acted on.
			ctx := core.NewContext()
			ctx.BeginRenderPass()
			_, advertised := list.Render(ctx).Props["onEndReached"]
			if advertised == c.want {
				t.Errorf("AutoLoadWithheld() says %v and the List %s advertise its "+
					"edge — a caller acting on this would show a second way to load a "+
					"page the scroll is already fetching, or hide the only one there is",
					c.want, map[bool]string{true: "does", false: "does not"}[advertised])
			}
		})
	}

	// A manual pager withholds nothing, because there is nothing to withhold.
	// The question is "is the automatic path off right now", and on a list
	// that never had one the answer is not "yes" — Collapse.hides is what a
	// caller asks about the run itself.
	shutList := base(map[string]bool{"feb": true})
	if shutList.AutoLoadWithheld() {
		t.Error("a list with no OnEndReached reports its edge as withheld — there is " +
			"no sensor to withhold, and a footer shown on the strength of this would " +
			"appear on every manual pager with a collapsed last group")
	}
}

// The three shapes that have nothing to withhold, each of which used to be the
// only shape this list had.
//
// A flat feed has no runs, an empty one has no rows, and a zero Collapse hides
// nothing — so all three keep the prop, and the feature costs a list that has
// never heard of collapsing exactly nothing.
func TestGroupedListKeepsTheEdgeWhenThereIsNothingToHide(t *testing.T) {
	for _, c := range []struct {
		name string
		list GroupedList[string]
	}{
		{"flat", GroupedList[string]{Items: []string{"a", "b"}}},
		{"empty", GroupedList[string]{
			GroupBy: func(s string) Group { return Group{Key: s, Label: s} },
			Collapse: Collapse{
				IsCollapsed: func(Group) bool { return true },
				OnToggle:    func(Group) {},
			},
		}},
		{"no collapse", GroupedList[string]{
			Items:   []string{"a", "b"},
			GroupBy: func(s string) Group { return Group{Key: s, Label: s} },
		}},
		// A predicate with no handler hides nothing — Collapse.hides requires
		// both halves, because a run withheld behind a band nobody can press
		// is a feed that silently loses rows. The edge must follow the rows:
		// they are all on screen, so the pager's next page has somewhere to
		// land.
		{"predicate with no handler", GroupedList[string]{
			Items:    []string{"a", "b"},
			GroupBy:  func(s string) Group { return Group{Key: s, Label: s} },
			Collapse: Collapse{IsCollapsed: func(Group) bool { return true }},
		}},
	} {
		ctx := core.NewContext()
		ctx.BeginRenderPass()
		list := c.list
		list.Row = func(s string) core.View { return core.Text(s) }
		list.OnEndReached = func() {}
		if _, ok := list.Render(ctx).Props["onEndReached"]; !ok {
			t.Errorf("%s: the edge was withheld from a list with nothing shut", c.name)
		}
	}
}

// trailingRun agrees with groupRuns about the last run.
//
// The widget asks its Collapse predicate about the group trailingRun returns,
// and the band the reader presses is the one groupRuns produced. If the two
// disagreed about the Label or the Count, a feed could withhold its edge for a
// group whose band says it is open — and the two derivations are deliberately
// different, because one is a full partition and the other is a walk back from
// the end.
func TestTrailingRunAgreesWithTheFullPartition(t *testing.T) {
	byKey := func(s string) Group {
		// Label deliberately varies within a key, which is the case the
		// first-item rule is about.
		return Group{Key: s[:1], Label: s}
	}
	for _, items := range [][]string{
		{"a1"},
		{"a1", "a2", "b1"},
		{"a1", "b1", "b2", "b3"},
		{"a1", "b1", "a2"}, // an unsorted input: the last run is one item
	} {
		runs := groupRuns(items, byKey)
		want := runs[len(runs)-1].Group
		got, ok := trailingRun(items, byKey)
		if !ok {
			t.Errorf("%v: trailingRun found nothing, groupRuns found %+v", items, want)
			continue
		}
		if got != want {
			t.Errorf("%v: trailingRun = %+v, groupRuns last = %+v", items, got, want)
		}
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

	// Level 2: a band titles a section *of* the screen whose name AppBar's
	// title carries at level 1. Without the tier the two announce as peers and
	// a reader navigating by heading cannot tell a screen from a month inside
	// it, which is the flat outline the level field was added to fix.
	if label.Style.AccessibilityHeadingLevel != 2 {
		t.Errorf("band label level = %d, want 2 (AppBar's title is 1)",
			label.Style.AccessibilityHeadingLevel)
	}
}

// --- The two facts a band is handed ------------------------------------------

// bandGroups renders a list whose Header override records every Group it is
// handed, in band order. That is the only vantage point from which
// Group.Trailing and Group.AutoLoadWithheld are observable at all: they are
// stamped inside appendRows and consumed by whatever builds the band, so a
// test that looked at the rendered tree would be looking at what the override
// decided rather than at what it was told.
func bandGroups(t *testing.T, list GroupedList[string]) []Group {
	t.Helper()
	var seen []Group
	list.Row = func(s string) core.View { return core.Text(s) }
	list.Header = func(g Group) core.View {
		seen = append(seen, g)
		return core.Text(g.Label)
	}
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	list.Render(ctx)
	return seen
}

// threeRuns is two January rows and one February row: three items, two runs,
// and the trailing run is the short one — so a check that confused "the last
// group" with "the biggest" or with "the first" fails.
var threeRuns = []string{"jan-1", "jan-2", "feb-1"}

func byPrefix(s string) Group { return Group{Key: s[:3], Label: s[:3]} }

// Exactly one band is told it is the trailing one, and it is the last.
//
// This is the fact HideTrailingCount rests on, and until Group carried it a
// Header override could not implement the same rule: the field's own
// documentation says an override "owns the decision itself", and the decision
// is *do not publish an open run's count* — which needs to know which run is
// open. Deriving that in the override meant re-walking Items with the same
// GroupBy the widget had just walked.
func TestABandIsToldWhetherItIsTheTrailingRun(t *testing.T) {
	got := bandGroups(t, GroupedList[string]{Items: threeRuns, GroupBy: byPrefix})
	if len(got) != 2 {
		t.Fatalf("got %d bands, want 2 — the fixture this test reasons about has changed", len(got))
	}
	if got[0].Trailing {
		t.Errorf("the %q band says it is trailing — a pager cannot extend a run that "+
			"another run already closed, and an override eliding this band's count "+
			"would be hiding a number that is final", got[0].Key)
	}
	if !got[1].Trailing {
		t.Errorf("the %q band does not say it is trailing — it is the run an append "+
			"pager extends, so its Count is the one that changes under the reader",
			got[1].Key)
	}

	// A single run is trailing: there is no "and also some earlier ones"
	// requirement, and a one-group feed is exactly the shape whose count is
	// most obviously still open.
	one := bandGroups(t, GroupedList[string]{Items: []string{"jan-1"}, GroupBy: byPrefix})
	if len(one) != 1 || !one[0].Trailing {
		t.Errorf("a list with one run reported %+v — the only band there is is the "+
			"trailing one", one)
	}
}

// The default band's own trailing rule is the same fact, so the two cannot
// drift.
//
// HideTrailingCount used to be `ri == len(runs)-1` at the band and
// trailingRun's own walk at the sensor: two derivations of one question, in a
// function where they are forty lines apart. They are one statement now, and
// this is the assertion that the default band still reads it — a band that
// went back to counting its own index would pass every test above and this
// one would keep it honest only if the two answers can be made to disagree,
// which is why the check is on the *rendered* band rather than on the Group.
func TestTheDefaultBandsTrailingRuleReadsTheStampedFact(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := GroupedList[string]{
		Items:             threeRuns,
		GroupBy:           byPrefix,
		Row:               func(s string) core.View { return core.Text(s) },
		HideTrailingCount: true,
	}.Render(ctx)

	// Two bands: the first keeps its badge, the second does not.
	bands := []*core.Node{}
	for _, c := range n.Children {
		if strings.HasPrefix(c.Key, "group:") {
			bands = append(bands, c)
		}
	}
	if len(bands) != 2 {
		t.Fatalf("got %d bands, want 2", len(bands))
	}
	if findText(bands[0], "2") == nil {
		t.Error("the closed run's band lost its count — only the trailing run's number " +
			"is still open, and hiding a closed one throws away a fact the reader can use")
	}
	if findText(bands[1], "1") != nil {
		t.Error("the trailing run's band published a count while more rows may follow — " +
			"HideTrailingCount is no longer reading Group.Trailing")
	}
}

// The withholding reaches the band that caused it, and no other.
//
// GroupedList.AutoLoadWithheld answers the caller, who owns the Footer. A
// Header override is a different reader in a different place, and the tree it
// builds is the only thing on screen that could say "collapsed — auto-load is
// off here". Before this it had to close over Items, GroupBy and Collapse and
// recompute an answer the widget had just computed.
func TestTheBandThatWithheldTheEdgeIsToldSo(t *testing.T) {
	list := func(shut map[string]bool, sensor bool) GroupedList[string] {
		g := GroupedList[string]{
			Items:   threeRuns,
			GroupBy: byPrefix,
			Collapse: Collapse{
				IsCollapsed: func(g Group) bool { return shut[g.Key] },
				OnToggle:    func(Group) {},
			},
		}
		if sensor {
			g.OnEndReached = func() {}
		}
		return g
	}

	for _, c := range []struct {
		name string
		shut map[string]bool
		want []bool // per band, in order
	}{
		{"the trailing run is shut", map[string]bool{"feb": true}, []bool{false, true}},
		{"nothing is shut", map[string]bool{}, []bool{false, false}},
		// The one that matters most: a shut run above the trailing one hides
		// its rows and withholds nothing, because the pager was never going to
		// extend it. A band that announced a pause here would be wrong on a
		// feed that is still fetching normally.
		{"a run above the trailing one is shut", map[string]bool{"jan": true}, []bool{false, false}},
		{"every run is shut", map[string]bool{"jan": true, "feb": true}, []bool{false, true}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := bandGroups(t, list(c.shut, true))
			if len(got) != len(c.want) {
				t.Fatalf("got %d bands, want %d", len(got), len(c.want))
			}
			for i, want := range c.want {
				if got[i].AutoLoadWithheld != want {
					t.Errorf("band %q: AutoLoadWithheld = %v, want %v",
						got[i].Key, got[i].AutoLoadWithheld, want)
				}
			}
		})
	}

	// No sensor, no withholding — the same answer the method gives, for the
	// same reason. A band on a manual pager that announced a pause would
	// appear on every list with a collapsed last group and no auto-load at all.
	for _, g := range bandGroups(t, list(map[string]bool{"feb": true}, false)) {
		if g.AutoLoadWithheld {
			t.Errorf("band %q reports the edge as withheld on a list with no "+
				"OnEndReached — there was no sensor to withhold", g.Key)
		}
	}
}

// What the bands are told and what the list does are one answer.
//
// This is the property that makes the field worth carrying at all, and it is
// the same one TestAutoLoadWithheldAgreesWithTheRenderedList asserts for the
// method: a Group saying "auto-load is off" above a List that kept its sensor
// would put two ways to load one page on the screen, and the opposite pair
// would hide the only one there is.
//
// It is a separate assertion from the method's because the two travel by
// different routes — the method is called on the value, the field is stamped
// inside the row loop — and a render that computed the answer twice could
// disagree with itself between them.
func TestTheBandAndTheListAgreeAboutTheWithheldEdge(t *testing.T) {
	for _, shut := range []map[string]bool{
		{"feb": true},
		{},
		{"jan": true},
		{"jan": true, "feb": true},
	} {
		list := GroupedList[string]{
			Items:        threeRuns,
			GroupBy:      byPrefix,
			OnEndReached: func() {},
			Collapse: Collapse{
				IsCollapsed: func(g Group) bool { return shut[g.Key] },
				OnToggle:    func(Group) {},
			},
		}

		var announced bool
		for _, g := range bandGroups(t, list) {
			if g.AutoLoadWithheld {
				announced = true
			}
		}

		ctx := core.NewContext()
		ctx.BeginRenderPass()
		l := list
		l.Row = func(s string) core.View { return core.Text(s) }
		_, hasSensor := l.Render(ctx).Props["onEndReached"]

		if announced == hasSensor {
			t.Errorf("shut=%v: a band %s the edge is withheld and the List %s its "+
				"sensor", shut,
				map[bool]string{true: "says", false: "does not say"}[announced],
				map[bool]string{true: "kept", false: "dropped"}[hasSensor])
		}
		if announced != list.AutoLoadWithheld() {
			t.Errorf("shut=%v: the bands say %v and the method says %v", shut,
				announced, list.AutoLoadWithheld())
		}
	}
}

// Every reader of a Group sees the same Group.
//
// The fields are stamped once, before anything is asked about the run — the
// Collapse predicate, the Header override, the default band, OnToggle. That
// ordering is not cosmetic: a predicate handed a bare Group and an override
// handed a stamped one is two shapes of the same value in one render, and the
// next person to key a collapse map on something other than Key would find
// out the hard way.
//
// trailingRun is the one deliberate exception and it is asserted here too: it
// stamps Trailing (it is, by construction, the trailing run) and never
// AutoLoadWithheld, which is the answer that walk is being run to compute.
func TestEveryReaderOfAGroupSeesTheSameGroup(t *testing.T) {
	var predicate, toggled []Group
	list := GroupedList[string]{
		Items:        threeRuns,
		GroupBy:      byPrefix,
		Row:          func(s string) core.View { return core.Text(s) },
		OnEndReached: func() {},
		Collapse: Collapse{
			IsCollapsed: func(g Group) bool {
				predicate = append(predicate, g)
				return g.Key == "feb"
			},
			OnToggle: func(g Group) { toggled = append(toggled, g) },
		},
	}
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	list.Render(ctx)

	// The bands, for comparison.
	bands := bandGroups(t, list)
	byKey := map[string]Group{}
	for _, g := range bands {
		byKey[g.Key] = g
	}

	// Every predicate call from inside the row loop matches the band for the
	// same run. The pre-render probe (trailingRun, from Render's own
	// AutoLoadWithheld call) is the exception: it carries Trailing and not the
	// answer it is computing.
	for _, g := range predicate {
		want := byKey[g.Key]
		if g == want {
			continue
		}
		probe := want
		probe.AutoLoadWithheld = false
		if g == probe && g.Trailing {
			continue // the trailingRun probe, as documented
		}
		t.Errorf("the Collapse predicate was asked about %+v; the band for that run "+
			"is %+v", g, want)
	}

	// Pressing a band calls OnToggle with the same value the band was built
	// from. A handler keying off Trailing — "collapse every run but the last",
	// which is the shape a feed's "collapse all" control takes — must see the
	// run the reader actually pressed.
	//
	// The default band, because a Header override is handed no OnToggle: the
	// override draws its own control (CollapseBand) and calls the caller's
	// function itself.
	ctx2 := core.NewContext()
	ctx2.BeginRenderPass()
	toggled = nil
	tree := list.Render(ctx2)
	var pressed int
	for _, band := range tree.Children {
		if !strings.HasPrefix(band.Key, "group:") {
			continue
		}
		button := findFirst(band, func(n *core.Node) bool {
			return n.Style != nil && n.Style.AccessibilityRole == core.RoleButton
		})
		if button == nil {
			t.Fatalf("band %s built no control — Collapse is active, so it should be a "+
				"disclosure", band.Key)
		}
		id, ok := button.Props["onClick"].(string)
		if !ok {
			t.Fatalf("band %s has a Button with no onClick", band.Key)
		}
		ctx2.TriggerCallback(id)
		pressed++
	}
	if pressed != len(bands) {
		t.Fatalf("pressed %d bands, want %d", pressed, len(bands))
	}
	if len(toggled) != len(bands) {
		t.Fatalf("OnToggle fired %d times for %d presses", len(toggled), len(bands))
	}
	for i, g := range toggled {
		if g != bands[i] {
			t.Errorf("pressing band %d called OnToggle with %+v; the band was built "+
				"from %+v", i, g, bands[i])
		}
	}
}

// A DataTable's bands carry the position and never the pause.
//
// A table has no OnEndReached, so there is no sensor for a shut run to
// withhold; the census in rows_spec_test.go records the field as
// GroupedList's for exactly that reason. What a table's bands *do* get is
// Trailing, because "which run is still open" is a fact about grouping rather
// than about paging, and a table's Header override wants it for the same
// count rule.
func TestADataTableBandIsToldItsPositionAndNeverThatAPageWasWithheld(t *testing.T) {
	var seen []Group
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	DataTable[string]{
		Columns: []Column[string]{{Title: "Row", Cell: func(s string) core.View { return core.Text(s) }}},
		Rows:    threeRuns,
		GroupBy: byPrefix,
		Header: func(g Group) core.View {
			seen = append(seen, g)
			return core.Text(g.Label)
		},
	}.Render(ctx)

	if len(seen) != 2 {
		t.Fatalf("got %d bands, want 2", len(seen))
	}
	if seen[0].Trailing || !seen[1].Trailing {
		t.Errorf("a table's bands report Trailing %v/%v, want false/true — the "+
			"grouping walk is shared, so the position is too",
			seen[0].Trailing, seen[1].Trailing)
	}
	for _, g := range seen {
		if g.AutoLoadWithheld {
			t.Errorf("the %q band of a table says an auto-load was withheld — a table "+
				"has no edge sensor, so there is nothing that could have been", g.Key)
		}
	}
}
