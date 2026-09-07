package components

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// CollapseBand, the disclosure a Header override places in a layout of its own.
//
// The gap it closes is a division rather than an omission: GroupedList.Collapse
// reaches past a Header override for the row emission and stops at it for the
// control, so an override's run hides and its band is the caller's to build.
// That left three things to rebuild by hand — a button, an aria-expanded stated
// on every pass, and a heading wrapper whose nesting order is argued at length
// in an unexported type — and the shape most likely to come out of that is one
// of the two that Accordion tried and discarded, both of which look correct in
// an export.
//
// So the load-bearing test here is the first one: what CollapseBand builds is
// what the default band builds, compared against it rather than against this
// file's own idea of the shape.

func renderBand(t *testing.T, v core.View) *core.Node {
	t.Helper()
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	return v.Render(ctx)
}

// The same shape as the default band's, in both states.
//
// Compared against GroupHeader rather than pinned, on exactly the argument
// TestBothDisclosuresBuildTheSameShape makes for Accordion and GroupHeader: a
// third construction of this arrangement checked only against its own
// expectations would drift the first time either of the other two was touched.
func TestACollapseBandIsTheSameDisclosureTheDefaultBandBuilds(t *testing.T) {
	group := Group{Key: "2026-01", Label: "January 2026", Count: 3}

	for _, open := range []bool{false, true} {
		shut := Collapse{
			IsCollapsed: func(Group) bool { return !open },
			OnToggle:    func(Group) {},
		}

		band := readDisclosure(t, renderBand(t, CollapseBand{
			Collapse:     shut,
			Group:        group,
			HeadingLevel: 2,
		}))
		def := readDisclosure(t, renderBand(t, GroupHeader{
			Group:        group,
			HeadingLevel: 2,
			Expanded:     open,
			OnToggle:     func() {},
		}))

		if band != def {
			t.Errorf("open=%v: CollapseBand built %+v, the default band builds %+v",
				open, band, def)
		}
	}
}

// The expansion is derived from the caller's own predicate, not stated twice.
//
// This is the field an override author would otherwise have to compute, and the
// direction is the one that is easy to invert: the widget's vocabulary is "which
// groups are shut" and ARIA's is "is this open", which is why GroupedList's own
// band derives it in one place and says so.
func TestACollapseBandReadsTheExpansionFromTheCallersPredicate(t *testing.T) {
	group := Group{Key: "2026-01", Label: "January 2026"}
	for _, tc := range []struct {
		collapsed bool
		want      core.ExpandedState
	}{
		{false, core.ExpandedOpen},
		{true, core.ExpandedClosed},
	} {
		shut := Collapse{
			IsCollapsed: func(Group) bool { return tc.collapsed },
			OnToggle:    func(Group) {},
		}
		got := readDisclosure(t, renderBand(t, CollapseBand{Collapse: shut, Group: group}))
		if got.expanded != tc.want {
			t.Errorf("IsCollapsed=%v produced aria-expanded %v, want %v — a control "+
				"pointing the other way from the run it hides",
				tc.collapsed, got.expanded, tc.want)
		}
	}
}

// The button toggles the group it is about.
//
// The whole reason the helper takes the caller's own Collapse rather than a
// bare func: an override that wired its control to a second Collapse would have
// a chevron obeying one state and a run obeying another.
func TestACollapseBandTogglesItsOwnGroup(t *testing.T) {
	var toggled []string
	shut := Collapse{
		IsCollapsed: func(Group) bool { return false },
		OnToggle:    func(g Group) { toggled = append(toggled, g.Key) },
	}
	// Rendered inline rather than through renderBand: firing the handler needs
	// the same Context the pass registered the callback ID on.
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := CollapseBand{
		Collapse: shut,
		Group:    Group{Key: "2026-02", Label: "February 2026"},
	}.Render(ctx)

	button := findFirst(n, func(c *core.Node) bool {
		return c.Style != nil && c.Style.AccessibilityRole == core.RoleButton
	})
	if button == nil {
		t.Fatal("no button")
	}
	id, ok := button.Props["onClick"].(string)
	if !ok {
		t.Fatalf("the button carries no onClick: %#v", button.Props["onClick"])
	}
	ctx.TriggerCallback(id)

	if len(toggled) != 1 || toggled[0] != "2026-02" {
		t.Errorf("pressing the band reported %v, want [2026-02]", toggled)
	}
}

// An inactive Collapse builds a heading and no control.
//
// Not a button with a dead handler. An expansion stated with nothing to toggle
// it is announced on both web targets, is silently nothing on Android, and is
// what core.AuditTree reports as ConcernInertDisclosure — so building one here
// would be building the thing the audit exists to find.
func TestAnInactiveCollapseBandIsAPlainHeading(t *testing.T) {
	group := Group{Key: "2026-01", Label: "January 2026"}
	for _, tc := range []struct {
		name string
		c    Collapse
	}{
		{"the zero value", Collapse{}},
		{"a predicate with no handler", Collapse{IsCollapsed: func(Group) bool { return true }}},
	} {
		n := renderBand(t, CollapseBand{Collapse: tc.c, Group: group, HeadingLevel: 3})

		if b := findFirst(n, func(c *core.Node) bool {
			return c.Style != nil && c.Style.AccessibilityRole == core.RoleButton
		}); b != nil {
			t.Errorf("%s built a control with nothing behind it", tc.name)
		}
		if e := findFirst(n, func(c *core.Node) bool {
			return c.Style != nil && c.Style.AccessibilityExpanded != core.ExpandedUnset
		}); e != nil {
			t.Errorf("%s stated an expansion nothing can change", tc.name)
		}

		heading := findFirst(n, func(c *core.Node) bool {
			return c.Style != nil && c.Style.AccessibilityRole == core.RoleHeading
		})
		if heading == nil {
			t.Fatalf("%s built no heading — the band still titles its run", tc.name)
		}
		if heading.Style.AccessibilityHeadingLevel != 3 {
			t.Errorf("%s put the heading at level %d, want 3",
				tc.name, heading.Style.AccessibilityHeadingLevel)
		}
		if findText(n, "January 2026") == nil {
			t.Errorf("%s dropped the label", tc.name)
		}
	}
}

// The default tier is the one GroupHeader takes, and it is taken from the same
// place — a band is a section of the screen whose name an AppBar carries at
// level 1.
func TestACollapseBandsDefaultTierMatchesTheDefaultBands(t *testing.T) {
	group := Group{Key: "2026-01", Label: "January 2026"}
	shut := Collapse{IsCollapsed: func(Group) bool { return false }, OnToggle: func(Group) {}}

	band := readDisclosure(t, renderBand(t, CollapseBand{Collapse: shut, Group: group}))
	def := readDisclosure(t, renderBand(t, GroupHeader{
		Group: group, Expanded: true, OnToggle: func() {},
	}))
	if band.headingLevel != def.headingLevel {
		t.Errorf("CollapseBand defaults to level %d and the default band to %d",
			band.headingLevel, def.headingLevel)
	}
}

// Content replaces the words inside the button and not the name outside it.
//
// A button's children are presentational — a reader does not descend into them
// — so whatever a caller puts here is decoration, and the announced name has to
// keep coming from the group. An implementation that named the control after
// its content would announce an icon-only band as nothing at all.
func TestACollapseBandsContentDoesNotBecomeItsName(t *testing.T) {
	group := Group{Key: "2026-01", Label: "January 2026"}
	shut := Collapse{IsCollapsed: func(Group) bool { return false }, OnToggle: func(Group) {}}

	n := renderBand(t, CollapseBand{
		Collapse: shut,
		Group:    group,
		Content:  []core.View{core.Text("★ Jan")},
	})
	got := readDisclosure(t, n)

	if got.buttonLabel != "January 2026" || got.headingLabel != "January 2026" {
		t.Errorf("the control is named %q inside a heading named %q, want the group's "+
			"label for both — a name assembled from content is the announcement "+
			"components.Chip's \", selected\" suffix was deleted for",
			got.buttonLabel, got.headingLabel)
	}
	if findText(n, "★ Jan") == nil {
		t.Error("the caller's content is not in the tree")
	}
	if findText(n, "January 2026") != nil {
		t.Error("the default label was drawn as well as the caller's content")
	}
}

// It is the control and nothing else: no Surface, no padding, no count badge.
//
// That absence is the whole reason it exists beside GroupHeader — a caller who
// wanted the band's chrome would use the band.
func TestACollapseBandCarriesNoneOfTheBandsChrome(t *testing.T) {
	group := Group{Key: "2026-01", Label: "January 2026", Count: 3}
	shut := Collapse{IsCollapsed: func(Group) bool { return false }, OnToggle: func(Group) {}}

	n := renderBand(t, CollapseBand{Collapse: shut, Group: group})
	if findText(n, "3") != nil {
		t.Error("the count badge came along; a caller placing this in a row of their " +
			"own decides where the count goes, and one inside the button would stop " +
			"being announced at all")
	}

	def := renderBand(t, GroupHeader{Group: group, Expanded: true, OnToggle: func() {}})
	if findText(def, "3") == nil {
		t.Error("the default band lost its badge, so the assertion above is vacuous")
	}
}

// --- The band's insets are the band's tap target ---------------------------

// bandNodes pulls the three nodes a band's geometry is spread over: the Row
// the list sees, the control a finger lands on, and the badge if there is one.
//
// The control is found by role rather than by path, because the two branches
// nest differently — a disclosure is a heading wrapping a button, a plain band
// is one Box — and which node holds the padding is the fact under test rather
// than a step on the way to it.
func bandNodes(t *testing.T, n *core.Node) (row, control *core.Node) {
	t.Helper()
	control = findFirst(n, func(c *core.Node) bool {
		return c.Style != nil && c.Style.AccessibilityRole == core.RoleButton
	})
	if control == nil {
		t.Fatalf("the band built no control: %+v", n)
	}
	return n, control
}

// The insets a reader sees are on the node a press reaches.
//
// # What this is asserting
//
// A band is a Row holding a control and a count. The chrome — 16px before the
// chevron, 4px above and below — used to be padding on the Row, which meant
// the button occupied the middle of the band and a press anywhere in the
// insets did nothing. Nothing about that was visible: the band looked right,
// announced right and exported right, and the only symptom was a header that
// needed aiming at.
//
// It is now padding on the control, which is the same pixels on a different
// node (see bandInsets for the picture). This test is what stops it drifting
// back, and the failure it is written against is a well-meant one: moving an
// inset "up to the band, where the rest of the styling is" reads as tidying.
func TestTheBandsInsetsAreOnItsControl(t *testing.T) {
	t.Run("with a count", func(t *testing.T) {
		row, control := bandNodes(t, renderBand(t, GroupHeader{
			Group:    Group{Key: "a", Label: "A", Count: 3},
			OnToggle: func() {},
		}))

		// The leading inset and both vertical ones are the control's; the
		// trailing one is an override for the badge that follows, which is
		// also what settles the horizontal shorthand into two sides here.
		want := core.EdgeInsets{Top: 4, Bottom: 4, Left: 16, Right: 8, Vertical: 4}
		if got := control.Style.Padding; got != want {
			t.Errorf("the control is padded %+v, want %+v — the band's insets are not "+
				"on it, so the 16px before the chevron is a place a press does nothing",
				got, want)
		}
		// And the Row holds the badge's own trailing inset and nothing else,
		// because that is the one inset past the control's right edge.
		if got := row.Style.Padding; got.Left != 0 || got.Top != 0 || got.Bottom != 0 ||
			got.Horizontal != 0 || got.Vertical != 0 {
			t.Errorf("the band Row still carries insets %+v — every one of them is "+
				"dead space, since the control fills the box it is given and no more",
				got)
		}
		if got, want := row.Style.Padding.Right, 16; got != want {
			t.Errorf("the band Row's trailing padding is %d, want %d — the badge's own "+
				"breathing room is the one inset that cannot move onto the control",
				got, want)
		}
	})

	t.Run("without a count", func(t *testing.T) {
		row, control := bandNodes(t, renderBand(t, GroupHeader{
			Group:     Group{Key: "a", Label: "A", Count: 3},
			HideCount: true,
			OnToggle:  func() {},
		}))
		if got := row.Style.Padding; got != (core.EdgeInsets{}) {
			t.Errorf("the band Row carries %+v with nothing after the control — every "+
				"inset on this band is reachable, so none of them belongs here", got)
		}
		// The whole horizontal step, both ends, since nothing follows — and
		// the shorthand still live beside its sides, which is what makes this
		// the case caller_style_insets_test.go reads for the settle.
		want := core.EdgeInsets{Top: 4, Bottom: 4, Left: 16, Right: 16,
			Horizontal: 16, Vertical: 4}
		if got := control.Style.Padding; got != want {
			t.Errorf("the control is padded %+v, want %+v — with no badge the trailing "+
				"inset is the control's too", got, want)
		}
	})
}

// The two branches are the same band, geometrically.
//
// A caller who adds OnToggle to a GroupHeader is asking for a control, not a
// relayout. The insets moved onto the control in the disclosure branch, and
// the plain branch had to move them onto the Box that stands in for one, or
// the day a feed became collapsible every band in it would have shifted 16px.
func TestABandIsTheSameSizeWithAndWithoutItsControl(t *testing.T) {
	group := Group{Key: "a", Label: "A", Count: 3}

	plain := renderBand(t, GroupHeader{Group: group})
	live := renderBand(t, GroupHeader{Group: group, OnToggle: func() {}})

	if plain.Style.Padding != live.Style.Padding {
		t.Errorf("the band Row is padded %+v without a control and %+v with one",
			plain.Style.Padding, live.Style.Padding)
	}

	// The plain branch's stand-in is the Box carrying the growth, which is
	// the same job the heading wrapper does in the other branch.
	inner := findFirst(plain, func(c *core.Node) bool {
		return c.Style != nil && c.Style.FlexGrow == 1
	})
	if inner == nil {
		t.Fatalf("the plain band has no growing child: %+v", plain)
	}
	_, control := bandNodes(t, live)
	if inner.Style.Padding != control.Style.Padding {
		t.Errorf("the plain band's label box is padded %+v and the control %+v — "+
			"adding a handler to a band moves it", inner.Style.Padding,
			control.Style.Padding)
	}
}

// A caller's ControlStyle reaches the control in both branches.
//
// The field exists because the insets are no longer where GroupHeader.Style
// lands, and a knob that worked on one branch only would be the relayout the
// test above rules out, arriving through the caller instead.
func TestControlStyleReachesBothBranches(t *testing.T) {
	group := Group{Key: "a", Label: "A", Count: 3}

	for _, c := range []struct {
		name string
		band GroupHeader
	}{
		{"plain", GroupHeader{Group: group,
			ControlStyle: []core.StyleProp{core.PaddingLeft(0)}}},
		{"disclosure", GroupHeader{Group: group, OnToggle: func() {},
			ControlStyle: []core.StyleProp{core.PaddingLeft(0)}}},
	} {
		n := renderBand(t, c.band)
		inner := findFirst(n, func(x *core.Node) bool {
			return x.Style != nil && x.Style.FlexGrow == 1
		})
		if c.name == "disclosure" {
			// The wrapper grows; the padding is on the button inside it.
			_, inner = bandNodes(t, n)
		}
		if inner == nil {
			t.Fatalf("%s: no node to read", c.name)
		}
		if got := inner.Style.Padding; got.Left != 0 || got.Horizontal != 0 {
			t.Errorf("%s: ControlStyle's PaddingLeft(0) left %+v — the caller's prop "+
				"is not the last one applied, or the shorthand was not settled",
				c.name, got)
		}
	}
}

// CollapseBand takes the same knob, which is the question it was built to
// answer one level out: a caller placing the control in their own row still
// has chrome to put somewhere, and putting it on the row is the shape this
// whole change was about.
func TestACollapseBandTakesTheChromeOnItsControl(t *testing.T) {
	group := Group{Key: "a", Label: "A", Count: 3}
	shut := Collapse{OnToggle: func(Group) {}}

	live := renderBand(t, CollapseBand{Collapse: shut, Group: group,
		ControlStyle: []core.StyleProp{core.PaddingHorizontal(16)}})
	_, control := bandNodes(t, live)
	if got, want := control.Style.Padding.Horizontal, 16; got != want {
		t.Errorf("CollapseBand's control is padded %d horizontally, want %d",
			got, want)
	}

	// And the inactive band, whose stand-in has to be the same size or a
	// caller's row changes shape when the handler arrives.
	plain := renderBand(t, CollapseBand{Group: group,
		ControlStyle: []core.StyleProp{core.PaddingHorizontal(16)}})
	if got, want := plain.Style.Padding.Horizontal, 16; got != want {
		t.Errorf("an inactive CollapseBand is padded %d horizontally, want %d — the "+
			"band changes size on the day it gets a handler", got, want)
	}
}
