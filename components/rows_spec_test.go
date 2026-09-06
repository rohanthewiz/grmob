package components

import (
	"reflect"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// appendRows' parameters used to be twelve positional arguments shared by two
// widgets, which had two consequences worth pinning now that they are a
// struct.
//
// The first is transposition: hideTrailingCount and sticky sat next to each
// other, both bool, and swapping them compiled. That one was survivable
// because each flag happens to have a test of its own in each widget.
//
// The second is the one nothing guarded, and it is what these tests are for:
// a knob wired at one call site and forgotten at the other. Positional
// arguments make that shape of edit easy — DataTable's call passed its
// twelve on three wrapped lines, and a reader checking them against
// GroupedList's was counting commas. DataTable.HeadingLevel was in fact
// never asserted anywhere before this file.

// wantRowsSpecFields is the census. It exists so that adding a knob to
// rowsSpec fails here rather than silently reaching one widget, and the
// failure message says what else to go and do.
var wantRowsSpecFields = []struct {
	name string
	kind reflect.Kind
}{
	{"Rows", reflect.Slice},
	{"Key", reflect.Func},
	{"Row", reflect.Func},
	{"GroupBy", reflect.Func},
	{"Header", reflect.Func},
	{"HideTrailingCount", reflect.Bool},
	{"StickyHeaders", reflect.Bool},
	{"HeadingLevel", reflect.Int},
	{"Dividers", reflect.Bool},
	{"Wrap", reflect.Func},
}

func TestRowsSpecCensus(t *testing.T) {
	rt := reflect.TypeOf(rowsSpec[sermon]{})
	if rt.NumField() != len(wantRowsSpecFields) {
		t.Fatalf("rowsSpec has %d fields, the census lists %d — a new knob must be "+
			"forwarded by BOTH GroupedList.Render and DataTable.Render, asserted in "+
			"TestBothWidgetsForwardEveryRowsSpecKnob, and added here",
			rt.NumField(), len(wantRowsSpecFields))
	}
	for i, w := range wantRowsSpecFields {
		f := rt.Field(i)
		if f.Name != w.name || f.Type.Kind() != w.kind {
			t.Errorf("field %d = %s %s, want %s (%s)", i, f.Name, f.Type.Kind(), w.name, w.kind)
		}
	}
}

// Every knob at once, through both widgets, so a call site that forwards nine
// of ten fails here. Each assertion below names the field it stands for.
//
// The fixture is deliberately the same rows for both: three runs of sermons
// (2/1/2), so a divider count, a band count and a trailing-count check all
// have known answers.
func TestBothWidgetsForwardEveryRowsSpecKnob(t *testing.T) {
	// bands, seps and rowKeys read one rendered child list.
	inspect := func(t *testing.T, body *core.Node) (bands []*core.Node, seps, rows int) {
		t.Helper()
		for _, c := range body.Children {
			switch {
			case strings.HasPrefix(c.Key, "group:"):
				bands = append(bands, c)
			case strings.HasPrefix(c.Key, "sep:"):
				seps++
			case strings.HasPrefix(c.Key, "sermon:"):
				rows++
			}
		}
		return bands, seps, rows
	}

	check := func(t *testing.T, what string, body *core.Node) {
		t.Helper()
		bands, seps, rows := inspect(t, body)

		// Rows + Key: five rows, keyed by the caller's function rather than
		// positionally.
		if rows != 5 {
			t.Errorf("%s: %d keyed rows, want 5 (Rows or Key not forwarded)", what, rows)
		}
		// GroupBy: three runs of the same month.
		if len(bands) != 3 {
			t.Fatalf("%s: %d bands, want 3 (GroupBy not forwarded)", what, len(bands))
		}
		// Dividers: between rows of a run only — runs of 2, 1, 2 give 1+0+1.
		if seps != 2 {
			t.Errorf("%s: %d separators, want 2 (Dividers not forwarded)", what, seps)
		}
		// StickyHeaders: the pin is on the band node itself.
		for _, b := range bands {
			if b.Style == nil || b.Style.Position != core.PositionSticky {
				t.Errorf("%s: band %q not pinned (StickyHeaders not forwarded)", what, b.Key)
			}
		}
		// HeadingLevel: the band's label carries it. Four, not the default
		// two, so a call site that forwards a zero is distinguishable from
		// one that forwards the field.
		label := findText(bands[0], "Month 2026-01")
		if label == nil {
			t.Fatalf("%s: first band has no label", what)
		}
		if label.Style.AccessibilityHeadingLevel != 4 {
			t.Errorf("%s: band heading level = %d, want 4 (HeadingLevel not forwarded)",
				what, label.Style.AccessibilityHeadingLevel)
		}
		// HideTrailingCount: the closed runs publish a count, the open last
		// one does not.
		if findText(bands[0], "2") == nil {
			t.Errorf("%s: a closed band lost its count badge", what)
		}
		if findText(bands[2], "2") != nil {
			t.Errorf("%s: the open trailing band published a count (HideTrailingCount not forwarded)", what)
		}
	}

	t.Run("GroupedList", func(t *testing.T) {
		ctx := core.NewContext()
		ctx.BeginRenderPass()
		n := GroupedList[sermon]{
			Items: sermons, Key: sermonKey, Row: sermonRow, GroupBy: byMonth,
			HideTrailingCount: true,
			StickyHeaders:     true,
			HeadingLevel:      4,
			Dividers:          true,
		}.Render(ctx)
		check(t, "GroupedList", n)
	})

	t.Run("DataTable", func(t *testing.T) {
		ctx := core.NewContext()
		ctx.BeginRenderPass()
		var tapped sermon
		n := DataTable[sermon]{
			Columns: sermonColumns, Rows: sermons, Key: sermonKey, GroupBy: byMonth,
			HideTrailingCount: true,
			StickyHeaders:     true,
			HeadingLevel:      4,
			Dividers:          true,
			// Wrap is the table's alone: it is how d.wrapRow gets around each
			// cell row, and a dropped Wrap loses every row's tap target.
			OnRowTap: func(s sermon) { tapped = s },
		}.Render(ctx)
		_, body := tableParts(t, n)
		check(t, "DataTable", body)

		row := findFirst(body, func(c *core.Node) bool { return c.Key == "sermon:1" })
		if row == nil {
			t.Fatal("no keyed row to tap")
		}
		id, ok := row.Props["onClick"].(string)
		if !ok {
			t.Fatal("row carries no tap handler (Wrap not forwarded)")
		}
		ctx.TriggerCallback(id)
		if tapped.ID != 1 {
			t.Errorf("tap reached %+v, want sermon 1", tapped)
		}
	})
}

// The two bools are adjacent in the struct as they were in the parameter
// list, so this pins that they remain distinguishable: each flag on alone
// must produce exactly its own effect and not the other's. A literal that
// crossed them — HideTrailingCount: g.StickyHeaders — reads plausibly and
// would otherwise render a list that pins nothing and hides a count nobody
// asked it to hide.
func TestTheTwoBandFlagsAreNotInterchangeable(t *testing.T) {
	render := func(hide, sticky bool) *core.Node {
		ctx := core.NewContext()
		ctx.BeginRenderPass()
		return GroupedList[sermon]{
			Items: sermons, Key: sermonKey, Row: sermonRow, GroupBy: byMonth,
			HideTrailingCount: hide,
			StickyHeaders:     sticky,
		}.Render(ctx)
	}
	bandsOf := func(n *core.Node) []*core.Node {
		var out []*core.Node
		for _, c := range n.Children {
			if strings.HasPrefix(c.Key, "group:") {
				out = append(out, c)
			}
		}
		return out
	}

	hideOnly := bandsOf(render(true, false))
	if findText(hideOnly[2], "2") != nil {
		t.Error("HideTrailingCount alone did not hide the trailing count")
	}
	for _, b := range hideOnly {
		if b.Style != nil && b.Style.Position == core.PositionSticky {
			t.Error("HideTrailingCount alone pinned a band — the flags are crossed")
		}
	}

	stickyOnly := bandsOf(render(false, true))
	if findText(stickyOnly[2], "2") == nil {
		t.Error("StickyHeaders alone hid the trailing count — the flags are crossed")
	}
	for _, b := range stickyOnly {
		if b.Style == nil || b.Style.Position != core.PositionSticky {
			t.Error("StickyHeaders alone did not pin the bands")
		}
	}
}
