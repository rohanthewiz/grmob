package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// heading returns the style of the node carrying the heading for s.
//
// Two shapes, checked in that order, because the package has one exception to
// its own rule.
//
// The rule is that a heading rides the *words*: the Text node reading s, never
// the row around it, because a row also holds a badge or a chevron and a
// heading spanning one is named "▸ What is a hook".
//
// The exception is comps.Accordion, whose header row has to be a button
// so it can carry aria-expanded — and a button's children are presentational,
// so a heading on the words inside it would be pruned. Its tier rides a Box
// wrapped around the row, which is named explicitly (an explicit
// AccessibilityLabel is what overrides content-derived naming, and is the
// whole reason that shape works). So an explicitly-named heading node wins the
// lookup where there is one, and the Text node answers everywhere else.
func heading(t *testing.T, n *core.Node, s string) core.Style {
	t.Helper()
	if named := findFirst(n, func(n *core.Node) bool {
		return n.Style != nil &&
			n.Style.AccessibilityRole == core.RoleHeading &&
			n.Style.AccessibilityLabel == s
	}); named != nil {
		return *named.Style
	}
	node := findText(n, s)
	if node == nil {
		t.Fatalf("no Text node reading %q in the rendered tree", s)
	}
	if node.Style == nil {
		t.Fatalf("%q has no style at all", s)
	}
	return *node.Style
}

// The default outline, read off the widgets rather than off the constants.
//
// This is the census the item behind this file asked for: core has carried
// levels 1..6 since the field existed and nothing in the package went past 2,
// so three of the six were reachable and unexercised. A widget per tier is
// what makes the range mean something — and a table, because the value of an
// outline is the *relationship* between the tiers, which four separate
// assertions would not have stated.
func TestTheDefaultHeadingOutline(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	// Literals, not headingLevelSection and headingLevelSubsection: this is
	// the assertion that the tiers are 1, 2, 2 and 3, and reading them out of
	// the same constants the widgets spend would make it assert only that a
	// constant equals itself.
	for _, tc := range []struct {
		what  string
		text  string
		level int
		node  *core.Node
	}{
		{"AppBar.Title", "Sermons", 1,
			AppBar{Title: "Sermons"}.Render(ctx)},
		{"GroupHeader", "March", 2,
			GroupHeader{Group: Group{Key: "03", Label: "March", Count: 4}}.Render(ctx)},
		{"Card.Title", "Recent", 2,
			Card{Title: "Recent"}.Render(ctx)},
		{"Accordion.Title", "What is a hook?", 3,
			Accordion{Title: "What is a hook?"}.Render(ctx)},
	} {
		s := heading(t, tc.node, tc.text)
		if s.AccessibilityRole != core.RoleHeading {
			t.Errorf("%s: role = %q, want %q — a level with no role is dropped by every "+
				"target that reads it", tc.what, s.AccessibilityRole, core.RoleHeading)
		}
		if s.AccessibilityHeadingLevel != tc.level {
			t.Errorf("%s: level = %d, want %d", tc.what, s.AccessibilityHeadingLevel, tc.level)
		}
	}
}

// The three tiers a widget only guesses at are overridable, and the one that
// is fixed by construction is not.
//
// An AppBar is the screen's own bar, so its title has nothing above it to be a
// section of and takes no field. The other three are *usually* where the table
// says: a grouped list inside a card is a tier deeper than the card, a screen
// of accordions has them at the top, and a bandless feed on a barless screen
// starts its outline at 1.
func TestHeadingLevelIsTheCallersWhenTheyKnowBetter(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	for _, tc := range []struct {
		what  string
		text  string
		level int
		node  *core.Node
	}{
		{"GroupHeader in a barless feed", "March", 1,
			GroupHeader{Group: Group{Label: "March"}, HeadingLevel: 1}.Render(ctx)},
		{"Card inside a section", "Recent", 3,
			Card{Title: "Recent", HeadingLevel: 3}.Render(ctx)},
		{"Accordion four deep", "Why?", 4,
			Accordion{Title: "Why?", HeadingLevel: 4}.Render(ctx)},
		// The far end of the range, which nothing in the framework reaches by
		// itself and which the exporters have always been able to write.
		{"Accordion at the floor of the range", "Why?", 6,
			Accordion{Title: "Why?", HeadingLevel: 6}.Render(ctx)},
	} {
		if got := heading(t, tc.node, tc.text).AccessibilityHeadingLevel; got != tc.level {
			t.Errorf("%s: level = %d, want %d", tc.what, got, tc.level)
		}
	}
}

// A negative level is how a caller asks for a heading with no tier at all.
//
// It needs no case in headingLevel and deliberately has none: core drops an
// out-of-range level rather than clamping it, so the value survives to the
// exporters and is written by none of them — which is the announcement every
// heading in this package made before it had a tier. The role still goes out,
// because "this is a heading" is the half that was never in doubt.
func TestANegativeLevelAsksForAnUntieredHeading(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	s := heading(t, Card{Title: "Recent", HeadingLevel: -1}.Render(ctx), "Recent")
	if s.AccessibilityRole != core.RoleHeading {
		t.Errorf("role = %q, want the heading role — only the tier was declined",
			s.AccessibilityRole)
	}
	if s.AccessibilityHeadingLevel >= 1 {
		t.Errorf("level = %d, want something core will drop", s.AccessibilityHeadingLevel)
	}
}

// The two escape-hatch slots take no heading, because the widget cannot know
// what it was handed.
//
// Card.Header and Accordion.Header replace the line entirely. Stamping a
// heading role onto the caller's view would announce a row of controls, or an
// avatar, as a section title — and the role attribute has one slot per node,
// so it would also silently overwrite whatever the caller put there.
func TestAHeaderSlotIsTheCallersToPlace(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	for _, tc := range []struct {
		what string
		node *core.Node
	}{
		{"Card.Header", Card{
			Title:  "ignored",
			Header: core.Text("Custom"),
		}.Render(ctx)},
		{"Accordion.Header", Accordion{
			Title:  "ignored",
			Header: core.Text("Custom"),
		}.Render(ctx)},
	} {
		s := heading(t, tc.node, "Custom")
		if s.AccessibilityRole != "" {
			t.Errorf("%s: the caller's view was given role %q", tc.what, s.AccessibilityRole)
		}
		if s.AccessibilityHeadingLevel != 0 {
			t.Errorf("%s: the caller's view was given level %d",
				tc.what, s.AccessibilityHeadingLevel)
		}
	}
}

// The band level reaches GroupHeader through both of its hosts.
//
// GroupedList and DataTable share appendRows, which builds the default band;
// the field is one more positional argument on a call with eleven of them, so
// the failure mode is a silently transposed pair rather than a compile error.
func TestBothCollectionsPlaceTheirBands(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	byMonth := func(s string) Group { return Group{Key: "03", Label: "March"} }

	list := GroupedList[string]{
		Items:        []string{"a", "b"},
		Row:          func(s string) core.View { return core.Text(s) },
		GroupBy:      byMonth,
		HeadingLevel: 3,
	}.Render(ctx)
	if got := heading(t, list, "March").AccessibilityHeadingLevel; got != 3 {
		t.Errorf("GroupedList band level = %d, want 3", got)
	}

	table := DataTable[string]{
		Rows:         []string{"a", "b"},
		Columns:      []Column[string]{{Title: "Item", Cell: func(s string) core.View { return core.Text(s) }}},
		GroupBy:      byMonth,
		HeadingLevel: 3,
	}.Render(ctx)
	if got := heading(t, table, "March").AccessibilityHeadingLevel; got != 3 {
		t.Errorf("DataTable band level = %d, want 3", got)
	}

	// And a column header takes the role without a tier, which is not an
	// oversight: aria-level is defined for heading, listitem and row, and
	// pointedly not for columnheader.
	col := findText(table, "Item")
	if col == nil || col.Style == nil {
		t.Fatal("no column header cell in the rendered table")
	}
	if col.Style.AccessibilityHeadingLevel != 0 {
		t.Errorf("column header carries level %d — ARIA defines none for columnheader",
			col.Style.AccessibilityHeadingLevel)
	}
}
