package comps

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

// renderBanded renders the sermon fixture with the given collapse state and
// returns the list node. Three runs: 2026-01 (2 rows), 2025-12 (1), 2025-11 (2).
func renderBanded(t *testing.T, c Collapse) *core.Node {
	t.Helper()
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	return GroupedList[sermon]{
		Items: sermons, Key: sermonKey, Row: sermonRow, GroupBy: byMonth,
		Collapse: c,
	}.Render(ctx)
}

// shutSet is the ordinary caller shape: a set of shut group keys.
func shutSet(keys ...string) map[string]bool {
	out := map[string]bool{}
	for _, k := range keys {
		out[k] = true
	}
	return out
}

// A shut run emits no rows, and the bands stay.
//
// No rows *at all*, rather than hidden ones. A collapsed group that still
// emitted its children would keep them in the tree for the reconciler to walk
// and diff on every pass, and — on the two web targets — in the document,
// where a display:none subtree is still reachable by a reader in some modes.
func TestACollapsedRunEmitsNoRows(t *testing.T) {
	shut := shutSet("2026-01", "2025-11")
	n := renderBanded(t, Collapse{
		IsCollapsed: func(g Group) bool { return shut[g.Key] },
		OnToggle:    func(Group) {},
	})

	var bands, rows []string
	for _, c := range n.Children {
		switch {
		case strings.HasPrefix(c.Key, "group:"):
			bands = append(bands, c.Key)
		case strings.HasPrefix(c.Key, "sermon:"):
			rows = append(rows, c.Key)
		}
	}

	wantBands := []string{"group:2026-01", "group:2025-12", "group:2025-11"}
	if strings.Join(bands, ",") != strings.Join(wantBands, ",") {
		t.Errorf("bands = %v, want all three still present %v", bands, wantBands)
	}
	// Only the open middle run's single row.
	if strings.Join(rows, ",") != "sermon:3" {
		t.Errorf("rows = %v, want only the open run's sermon:3", rows)
	}
}

// The band keys survive a toggle, so the reconciler patches the rows away
// rather than replacing the band.
//
// This is what makes a collapse cheap and a re-open smooth: a band that lost
// its identity across the toggle would be torn down and rebuilt, taking any
// sticky positioning and any focus inside it with it.
func TestABandKeepsItsKeyAcrossACollapse(t *testing.T) {
	open := renderBanded(t, Collapse{OnToggle: func(Group) {}})
	shut := renderBanded(t, Collapse{
		IsCollapsed: func(Group) bool { return true },
		OnToggle:    func(Group) {},
	})

	keysOf := func(n *core.Node) []string {
		var out []string
		for _, c := range n.Children {
			if strings.HasPrefix(c.Key, "group:") {
				out = append(out, c.Key)
			}
		}
		return out
	}
	if a, b := keysOf(open), keysOf(shut); strings.Join(a, ",") != strings.Join(b, ",") {
		t.Errorf("band keys changed across a collapse: %v -> %v", a, b)
	}
}

// Pressing a band calls OnToggle with that band's own Group.
//
// The Group and not just its key, because a caller's handler may want the
// label or the count — and because the closure capturing the wrong run is the
// classic loop-variable bug, which one band would never reveal.
func TestPressingABandReportsItsOwnGroup(t *testing.T) {
	var got []string
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := GroupedList[sermon]{
		Items: sermons, Key: sermonKey, Row: sermonRow, GroupBy: byMonth,
		Collapse: Collapse{OnToggle: func(g Group) { got = append(got, g.Key+"/"+g.Label) }},
	}.Render(ctx)

	for _, c := range n.Children {
		if !strings.HasPrefix(c.Key, "group:") {
			continue
		}
		button := findFirst(c, func(x *core.Node) bool {
			return x.Style != nil && x.Style.AccessibilityRole == core.RoleButton
		})
		if button == nil {
			t.Fatalf("band %q has no button", c.Key)
		}
		id, ok := button.Props["onClick"].(string)
		if !ok {
			t.Fatalf("band %q carries no handler", c.Key)
		}
		ctx.TriggerCallback(id)
	}

	want := "2026-01/Month 2026-01,2025-12/Month 2025-12,2025-11/Month 2025-11"
	if strings.Join(got, ",") != want {
		t.Errorf("toggles reported %v, want %v", got, want)
	}
}

// The zero Collapse is the list exactly as it was: plain bands, every row.
//
// The regression this guards is a widget that gained a knob and started
// building the new shape for everybody. The band is a heading on its *label*
// when it is not a disclosure — the count badge is beside it, and a heading
// spanning the row would be named "January 2026 3".
func TestTheZeroCollapseLeavesTheBandAsItWas(t *testing.T) {
	n := renderBanded(t, Collapse{})

	if b := findFirst(n, func(c *core.Node) bool {
		return c.Style != nil && c.Style.AccessibilityRole == core.RoleButton
	}); b != nil {
		t.Error("a list with no Collapse built a control in its bands")
	}
	rows := 0
	for _, c := range n.Children {
		if strings.HasPrefix(c.Key, "sermon:") {
			rows++
		}
	}
	if rows != len(sermons) {
		t.Errorf("%d rows, want all %d", rows, len(sermons))
	}
	// The heading is on the label Text, not on the band Row.
	label := findText(n, "Month 2026-01")
	if label == nil || label.Style.AccessibilityRole != core.RoleHeading {
		t.Error("the plain band's heading is not on its label")
	}
}

// IsCollapsed without OnToggle hides nothing.
//
// The two are one fact and Collapse.active keys on the handler, deliberately:
// a predicate with no handler would hide rows behind a band with no control,
// which is a feed that silently loses items. A handler with no predicate is
// the coherent half — a disclosure that is always open — and stays allowed.
func TestAPredicateWithNoHandlerHidesNothing(t *testing.T) {
	n := renderBanded(t, Collapse{IsCollapsed: func(Group) bool { return true }})

	rows := 0
	for _, c := range n.Children {
		if strings.HasPrefix(c.Key, "sermon:") {
			rows++
		}
	}
	if rows != len(sermons) {
		t.Errorf("%d rows, want all %d — a predicate with no handler must not hide a run",
			rows, len(sermons))
	}
}

// A Header override keeps its own band and still gets the row hiding.
//
// This is the division the field documents and the one place Collapse differs
// from StickyHeaders and HeadingLevel, which an override swallows entirely. A
// view the caller built is the caller's — but *which rows are emitted* is not
// inside that view, and a list that ignored the collapse under an override
// would give a caller a control that does nothing.
func TestAHeaderOverrideStillHidesTheRun(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := GroupedList[sermon]{
		Items: sermons, Key: sermonKey, Row: sermonRow, GroupBy: byMonth,
		Header: func(g Group) core.View { return core.Text("mine: " + g.Label) },
		Collapse: Collapse{
			IsCollapsed: func(g Group) bool { return g.Key == "2026-01" },
			OnToggle:    func(Group) {},
		},
	}.Render(ctx)

	var rows []string
	for _, c := range n.Children {
		if strings.HasPrefix(c.Key, "sermon:") {
			rows = append(rows, c.Key)
		}
	}
	if strings.Join(rows, ",") != "sermon:3,sermon:4,sermon:5" {
		t.Errorf("rows = %v, want the 2026-01 run hidden and the rest kept", rows)
	}
	// And the override's own band is untouched — no button was added around
	// or inside it.
	if findText(n, "mine: Month 2026-01") == nil {
		t.Error("the override's band did not survive")
	}
	if b := findFirst(n, func(c *core.Node) bool {
		return c.Style != nil && c.Style.AccessibilityRole == core.RoleButton
	}); b != nil {
		t.Error("a control was built for a band the caller owns")
	}
}

// Dividers are counted within the run that is actually emitted.
//
// A shut run contributes neither rows nor separators, and the open runs are
// unaffected — the last-row check is per run, so hiding one cannot leave a
// trailing hairline under another.
func TestACollapsedRunTakesItsDividersWithIt(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := GroupedList[sermon]{
		Items: sermons, Key: sermonKey, Row: sermonRow, GroupBy: byMonth,
		Dividers: true,
		Collapse: Collapse{
			IsCollapsed: func(g Group) bool { return g.Key == "2026-01" },
			OnToggle:    func(Group) {},
		},
	}.Render(ctx)

	seps := 0
	for _, c := range n.Children {
		if strings.HasPrefix(c.Key, "sep:") {
			seps++
		}
	}
	// Runs of 2 (shut), 1, 2 -> 0 + 0 + 1.
	if seps != 1 {
		t.Errorf("%d separators, want 1", seps)
	}
}

// A collapsible band still publishes its count, and the badge is outside the
// control.
//
// Inside a button a reader does not descend into the children, so the count
// would stop being announced — it is content, not chrome. The assertion is
// structural rather than visual: the badge must not be found under the
// button.
func TestACollapsibleBandKeepsItsCountOutsideTheControl(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := GroupHeader{
		Group:    Group{Key: "2026-01", Label: "January 2026", Count: 3},
		Expanded: true,
		OnToggle: func() {},
	}.Render(ctx)

	if findText(n, "3") == nil {
		t.Fatal("the band lost its count badge")
	}
	button := findFirst(n, func(c *core.Node) bool {
		return c.Style != nil && c.Style.AccessibilityRole == core.RoleButton
	})
	if button == nil {
		t.Fatal("no button")
	}
	if findText(button, "3") != nil {
		t.Error("the count badge is inside the button, where a reader will not announce it")
	}
}

// The band's heading wrapper is what grows, so the badge still pins to the
// trailing edge.
//
// The flex line belongs to the band Row, so only its own direct child can
// take the free space. Plain, that is the Box around the label; as a
// disclosure it is the heading — and a FlexGrow left on the inner button
// would distribute nothing and let the badge float in beside the words.
func TestTheCollapsibleBandGrowsAtTheHeading(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := GroupHeader{
		Group:    Group{Key: "2026-01", Label: "January 2026", Count: 3},
		Expanded: true,
		OnToggle: func() {},
	}.Render(ctx)

	var heading *core.Node
	for _, c := range n.Children {
		if c.Style != nil && c.Style.AccessibilityRole == core.RoleHeading {
			heading = c
		}
	}
	if heading == nil {
		t.Fatal("the heading is not a direct child of the band")
	}
	if heading.Style.FlexGrow != 1 {
		t.Errorf("heading FlexGrow = %v, want 1", heading.Style.FlexGrow)
	}
}

// The whole arrangement, at the level a browser reads it.
//
// Every assertion above is about the node tree, which is where the widget's
// decisions are — but the state is only worth stating if it survives the
// export, and aria-expanded is dropped by the exporter on any role ARIA does
// not scope it to. A band that built the state onto the wrong node would pass
// every structural check here and reach a reader as an ordinary button.
func TestACollapsibleBandExportsTheDisclosure(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := GroupHeader{
		Group:        Group{Key: "2026-01", Label: "January 2026", Count: 3},
		HeadingLevel: 2,
		Expanded:     false,
		OnToggle:     func() {},
	}.Render(ctx)

	html := htmlout.ExportHTML(n)
	for _, want := range []string{
		`role="heading"`,
		`aria-level="2"`,
		`role="button"`,
		`aria-expanded="false"`,
		`aria-label="January 2026"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("export is missing %s:\n%s", want, html)
		}
	}
	// The button inside the heading, not beside it. Read off the export
	// because that is the one place the nesting is a fact about a document
	// rather than about a Go slice.
	h := strings.Index(html, `role="heading"`)
	b := strings.Index(html, `role="button"`)
	if h < 0 || b < 0 || b < h {
		t.Errorf("the button is not inside the heading:\n%s", html)
	}
}

// The list's collapse state reaches the band's announcement.
//
// The gap this closes is a wiring one, and it is the kind nothing else here
// could see: every other test in this file asks whether the *rows* are
// emitted, and the band's aria-expanded is derived from the same predicate on
// a separate line. Inverting that line — gh.Expanded = collapsed(group) —
// leaves the rows correct and tells every reader the opposite of what the
// screen shows, which is worse than either half being wrong on its own.
//
// Both runs in one pass, so the assertion is about the mapping rather than
// about a default: a band that hard-coded either state would fail one of them.
func TestTheBandAnnouncesWhetherItsOwnRunIsShowing(t *testing.T) {
	n := renderBanded(t, Collapse{
		IsCollapsed: func(g Group) bool { return g.Key == "2026-01" },
		OnToggle:    func(Group) {},
	})

	want := map[string]core.ExpandedState{
		"group:2026-01": core.ExpandedClosed, // shut
		"group:2025-12": core.ExpandedOpen,
		"group:2025-11": core.ExpandedOpen,
	}
	seen := 0
	for _, c := range n.Children {
		w, ok := want[c.Key]
		if !ok {
			continue
		}
		seen++
		button := findFirst(c, func(x *core.Node) bool {
			return x.Style != nil && x.Style.AccessibilityRole == core.RoleButton
		})
		if button == nil {
			t.Errorf("band %q has no control", c.Key)
			continue
		}
		if got := button.Style.AccessibilityExpanded; got != w {
			t.Errorf("band %q announces %q, want %q — the run is %s",
				c.Key, got, w, map[bool]string{true: "hidden", false: "showing"}[w == core.ExpandedClosed])
		}
	}
	if seen != len(want) {
		t.Errorf("found %d of %d bands", seen, len(want))
	}
}
