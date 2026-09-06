package components

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

// middleOf returns the row's growing centre column — the Column child whose
// FlexGrow is 1. Located by predicate rather than index so these tests keep
// passing if the row ever gains a decorative wrapper slot.
func middleOf(n *core.Node) *core.Node {
	for _, c := range n.Children {
		if c.Type == "Column" && c.Style != nil && c.Style.FlexGrow == 1 {
			return c
		}
	}
	return nil
}

func TestListRowStructureAndSlotOrder(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := ListRow{
		Leading:  core.Checkbox(false, func(bool) {}),
		Title:    "Buy milk",
		Subtitle: "Due today",
		Trailing: core.Text("✕"),
	}.Render(ctx)

	if n.Type != "Row" {
		t.Fatalf("root type = %q, want Row (the theme's Row base supplies the padding)", n.Type)
	}
	if len(n.Children) != 3 {
		t.Fatalf("want leading, middle, trailing; got %d children", len(n.Children))
	}
	if n.Children[0].Type != "Checkbox" {
		t.Errorf("leading slot must render first, got %q", n.Children[0].Type)
	}
	if n.Children[2].Props["content"] != "✕" {
		t.Errorf("trailing slot must render last, got %v", n.Children[2].Props)
	}

	mid := middleOf(n)
	if mid == nil {
		t.Fatal("the middle column must carry FlexGrow(1) — that is what pins Trailing to the edge")
	}
	if mid != n.Children[1] {
		t.Error("the growing middle must sit between leading and trailing")
	}
	if findText(mid, "Buy milk") == nil || findText(mid, "Due today") == nil {
		t.Error("Title and Subtitle both belong in the middle column")
	}
	if mid.Style.Padding != (core.EdgeInsets{}) {
		t.Errorf("middle column must zero the theme Column padding, got %+v", mid.Style.Padding)
	}
}

// The pinning mechanism must be FlexGrow on the middle, never JustifyBetween
// on the row: the two disagree whenever a slot is missing, and reproducing
// that disagreement is what this widget exists to prevent.
func TestListRowPinsWithFlexGrowNotJustify(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ListRow{Title: "Balance", Trailing: core.Text("$12.00")}.Render(ctx)

	if n.Style.JustifyContent != "" {
		t.Errorf("row must not set JustifyContent, got %q", n.Style.JustifyContent)
	}
	if middleOf(n) == nil {
		t.Error("middle column missing its FlexGrow")
	}
	if n.Style.AlignItems != core.AlignItemsCenter {
		t.Errorf("row should centre its slots vertically, got %q", n.Style.AlignItems)
	}
}

// The middle is structure, not content: it is emitted even when empty so the
// trailing slot pins identically in every configuration.
func TestListRowKeepsMiddleWhenEmpty(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ListRow{Leading: core.Text("icon"), Trailing: core.Text("›")}.Render(ctx)

	mid := middleOf(n)
	if mid == nil {
		t.Fatal("the growing middle must survive an empty Title/Subtitle/Content")
	}
	if len(mid.Children) != 0 {
		t.Errorf("an empty middle must render no placeholder text, got %d children", len(mid.Children))
	}
}

func TestListRowContentOverridesTitleAndSubtitle(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ListRow{
		Title:    "ignored",
		Subtitle: "also ignored",
		Content:  core.Text("custom middle"),
	}.Render(ctx)

	if findText(n, "custom middle") == nil {
		t.Error("Content slot should render")
	}
	if findText(n, "ignored") != nil || findText(n, "also ignored") != nil {
		t.Error("Title/Subtitle must be suppressed when Content is set — the slot is the escape hatch")
	}
}

func TestListRowTapAndLongPress(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	var tapped, held bool
	n := ListRow{
		Title:       "Article 1",
		OnTap:       func() { tapped = true },
		OnLongPress: func() { held = true },
	}.Render(ctx)

	tapID, ok := n.Props["onClick"].(string)
	if !ok {
		t.Fatal("OnTap should register an onClick callback on the row itself")
	}
	holdID, ok := n.Props["onLongPress"].(string)
	if !ok {
		t.Fatal("OnLongPress should register an onLongPress callback")
	}
	ctx.TriggerCallback(tapID)
	ctx.TriggerCallback(holdID)
	if !tapped || !held {
		t.Errorf("callbacks did not fire: tapped=%v held=%v", tapped, held)
	}
}

// A presentational row must stay free of handler props, so a long list does
// not register a callback per row for nothing.
func TestListRowWithoutHandlersRegistersNothing(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ListRow{Title: "read only"}.Render(ctx)

	if _, ok := n.Props["onClick"]; ok {
		t.Error("no OnTap should mean no onClick prop")
	}
	if _, ok := n.Props["onLongPress"]; ok {
		t.Error("no OnLongPress should mean no onLongPress prop")
	}
}

func TestListRowSelectedThemeDefault(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ListRow{Title: "Article 1", Selected: true}.Render(ctx)

	if n.Style.Background != core.DefaultTheme.Colors.Surface {
		t.Errorf("selected background = %q, want theme Surface %q", n.Style.Background, core.DefaultTheme.Colors.Surface)
	}
}

func TestListRowSelectedStyleOverridesCallerStyle(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ListRow{
		Title:         "Article 1",
		Selected:      true,
		Style:         []core.StyleProp{core.BackgroundColor("#FFFFFF")},
		SelectedStyle: []core.StyleProp{core.BackgroundColor("#E8F0FE")},
	}.Render(ctx)

	if n.Style.Background != "#E8F0FE" {
		t.Errorf("selection must win over the base Style, got %q", n.Style.Background)
	}
}

func TestListRowStyleOverridesWidgetDefaults(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ListRow{
		Title: "Article 1",
		Style: []core.StyleProp{core.Gap(2), core.Padding(0)},
	}.Render(ctx)

	if n.Style.Gap != 2 {
		t.Errorf("caller Gap should beat the widget default, got %v", n.Style.Gap)
	}
	if n.Style.Padding != (core.EdgeInsets{}) {
		t.Errorf("caller Padding should beat the theme Row base, got %+v", n.Style.Padding)
	}
}

func TestListRowAccessibilityAnnouncesSelection(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	base := ListRow{
		Title:              "Article 1",
		AccessibilityLabel: "Article 1",
		AccessibilityHint:  "Selects the article; long-press to star it",
	}
	unselected := base.Render(ctx)
	base.Selected = true
	selected := base.Render(ctx)

	if got := unselected.Style.AccessibilityLabel; got != "Article 1" {
		t.Errorf("unselected label = %q", got)
	}
	if got := selected.Style.AccessibilityLabel; got != "Article 1, selected" {
		t.Errorf("selected label = %q, want the state appended", got)
	}
	if got := selected.Style.AccessibilityHint; got != "Selects the article; long-press to star it" {
		t.Errorf("hint = %q", got)
	}
}

// No label is synthesized from Title: labelling the container would override
// how the row's own children are announced.
func TestListRowSynthesizesNoLabel(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := ListRow{Title: "Buy milk", Selected: true}.Render(ctx)

	if got := n.Style.AccessibilityLabel; got != "" {
		t.Errorf("row label = %q, want empty when the caller named nothing", got)
	}
}

// --- Depth ------------------------------------------------------------------

// The first consumer core.Style.AccessibilityNestingLevel has ever had.
//
// The field has been in core since the heading tier's sibling landed, exported
// by both web targets and exercised by nothing but their own unit tests — the
// note that came with it said to watch for the first nested list downstream.
// This is it: a flattened outline, where the rows are siblings in the markup
// because a list is a flat run of children, and the depth a reader needs has
// nowhere else to live.
func TestListRowStatesItsDepth(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	for _, level := range []int{1, 2, 7} {
		n := ListRow{Title: "Matthew", NestingLevel: level}.Render(ctx)
		if n.Style.AccessibilityRole != core.RoleListItem {
			t.Errorf("level %d: role = %q, want %q — a depth with no role is dropped by "+
				"every target that reads it", level, n.Style.AccessibilityRole, core.RoleListItem)
		}
		if n.Style.AccessibilityNestingLevel != level {
			t.Errorf("level %d: depth = %d", level, n.Style.AccessibilityNestingLevel)
		}
	}
}

// Zero leaves the row exactly the unroled Box it has always been.
//
// This is the ownership rule, not caution. A `listitem` with no `list` around
// it names a structure that is not there, and a row cannot see its container,
// so the opt-in is what keeps every existing list in every app from quietly
// growing orphan roles.
func TestListRowWithNoDepthClaimsNothing(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := ListRow{Title: "Matthew", AccessibilityLabel: "Matthew"}.Render(ctx)
	if n.Style.AccessibilityRole != "" {
		t.Errorf("role = %q on a row that asked for no depth", n.Style.AccessibilityRole)
	}
	if n.Style.AccessibilityNestingLevel != 0 {
		t.Errorf("depth = %d on a row that asked for none", n.Style.AccessibilityNestingLevel)
	}
}

// A depth does not bring the selection state in with it, and the pairing is
// the point: the two fields get opposite answers, and the reason is a property
// of the roles rather than of the widget.
//
// A depth's role is `listitem`, one of the three ARIA defines aria-level for,
// and ARIA defines no selection state for it. So a nested row that is also
// chosen still spells the state into its accessible name — the fallback the
// widget has always had, kept because the alternative here is silence. A row
// that wants the state stated properly asks for Selectable and gives up the
// depth; TestSelectableTakesTheRowsOneRoleFromTheDepth is that case.
func TestDepthDoesNotBringTheSelectionInWithIt(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := ListRow{
		Title:              "Matthew",
		AccessibilityLabel: "Matthew",
		Selected:           true,
		NestingLevel:       2,
	}.Render(ctx)

	if n.Style.AccessibilityNestingLevel != 2 {
		t.Errorf("depth = %d, want 2", n.Style.AccessibilityNestingLevel)
	}
	if n.Style.AccessibilitySelected != "" {
		t.Errorf("selected = %q — ARIA defines no selection state for listitem, so a row "+
			"that took one would be announcing into a slot readers drop",
			n.Style.AccessibilitySelected)
	}
	if n.Style.AccessibilityLabel != "Matthew, selected" {
		t.Errorf("name = %q, want the suffix a row still has to spell",
			n.Style.AccessibilityLabel)
	}
}

// --- Selectable -------------------------------------------------------------

// The row's first real state, four sessions after the widget started asking
// for one.
//
// Both values are stated, not just the chosen one. A listbox in which only the
// selected option answers announces the rest as plain rows — the same failure
// core.SelectedOff was added for one widget over, where a tab strip's four
// quiet tabs read as furniture beside one real tab.
func TestSelectableRowStatesBothSidesOfTheChoice(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	for _, tc := range []struct {
		on   bool
		want core.SelectedState
	}{
		{true, core.SelectedOn},
		{false, core.SelectedOff},
	} {
		n := ListRow{Title: "Weekly", Selectable: true, Selected: tc.on}.Render(ctx)

		if n.Style.AccessibilityRole != core.RoleOption {
			t.Errorf("selected=%v: role = %q, want %q — a state with no role is dropped "+
				"by both web targets exactly as a name on a generic element is",
				tc.on, n.Style.AccessibilityRole, core.RoleOption)
		}
		if n.Style.AccessibilitySelected != tc.want {
			t.Errorf("selected=%v: state = %q, want %q", tc.on,
				n.Style.AccessibilitySelected, tc.want)
		}
	}
}

// The suffix is a fallback and stops the moment the state has a real home.
//
// Appending it to a Selectable row would announce the selection twice — once
// inside the row's name and once as the control state — and would put a
// changing word in a name that is meant to be stable, which is the whole
// complaint core.SelectedState was written to answer.
func TestSelectableRowDropsTheNameSuffix(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	row := ListRow{Title: "Weekly", AccessibilityLabel: "Weekly",
		Selected: true, Selectable: true}
	n := row.Render(ctx)

	if got := n.Style.AccessibilityLabel; got != "Weekly" {
		t.Errorf("name = %q, want the caller's own label — the state is stated "+
			"separately now and the name must stop moving", got)
	}

	// The same row without the opt-in still spells it, because there is still
	// nowhere else for it to go.
	row.Selectable = false
	if got := row.Render(ctx).Style.AccessibilityLabel; got != "Weekly, selected" {
		t.Errorf("unroled row name = %q, want the suffix it has always had", got)
	}
}

// Selectable takes the row's one role, and the depth is what loses.
//
// A node has one role and the two candidates are exclusive: `option` carries
// aria-selected and not aria-level, `listitem` the reverse. A row asking for
// both is describing a container that is a list and a listbox at once, which
// does not exist — so the state wins, because it is what the row is being
// tapped to change, and a depth inside a container that has not claimed to be
// a list is decoration.
//
// The level field is not merely ignored downstream; it is never set, so no
// exporter has to know about a combination core does not produce.
func TestSelectableTakesTheRowsOneRoleFromTheDepth(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := ListRow{
		Title:        "Matthew",
		Selectable:   true,
		Selected:     true,
		NestingLevel: 3,
	}.Render(ctx)

	if n.Style.AccessibilityRole != core.RoleOption {
		t.Errorf("role = %q, want %q", n.Style.AccessibilityRole, core.RoleOption)
	}
	if n.Style.AccessibilityNestingLevel != 0 {
		t.Errorf("depth = %d, want it dropped: ARIA defines aria-level for listitem "+
			"and not for option, so a level here would describe nothing",
			n.Style.AccessibilityNestingLevel)
	}
}

// A row that asks for nothing claims nothing — the ownership rule, checked
// from the selection's side.
//
// An `option` with no `listbox` around it is the orphan-role failure one row
// down from an orphan `listitem`, and a row cannot see its container. So the
// state is opt-in too: every list in every existing app keeps rendering the
// unroled Boxes it always did.
func TestAnUnselectableRowStatesNoSelection(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	n := ListRow{Title: "Weekly", Selected: true}.Render(ctx)

	if n.Style.AccessibilityRole != "" {
		t.Errorf("role = %q on a row that did not ask to be an option",
			n.Style.AccessibilityRole)
	}
	if n.Style.AccessibilitySelected != "" {
		t.Errorf("state = %q on an unroled Box — both web targets drop it, so it would "+
			"be announced on the natives alone", n.Style.AccessibilitySelected)
	}
}

// The look is the same either way: Selectable is an accessibility opt-in, not
// a second selected style. A caller adding it to an existing row must not find
// the row repainting itself.
func TestSelectableDoesNotChangeTheSelectedLook(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	plain := ListRow{Title: "Weekly", Selected: true}.Render(ctx)
	option := ListRow{Title: "Weekly", Selected: true, Selectable: true}.Render(ctx)

	if plain.Style.Background != option.Style.Background {
		t.Errorf("background differs: %q vs %q", plain.Style.Background, option.Style.Background)
	}
}

// An unroled row's name reaches a browser at all.
//
// This is the oldest of the two silences the widget documents and the one that
// was never the widget's to fix: a ListRow with an AccessibilityLabel and
// neither Selectable nor NestingLevel is a plain Box, ARIA prohibits an
// accessible name on the `generic` role a <div> carries, and both web targets
// dropped it while both natives read it out. core.RoleGroup — supplied by the
// exporters rather than set here — is what closed it.
//
// Asserted through the export rather than on the node, because the node is
// exactly what it always was. The fix is downstream of this widget, and this
// test's job is to prove the widget's rows are on the right side of it.
func TestAnUnroledRowsNameIsAnnouncedOnTheWeb(t *testing.T) {
	ctx := core.NewContext()
	n := ListRow{Title: "Ana", AccessibilityLabel: "Ana, 3 unread"}.Render(ctx)

	// The widget itself states no role: that is the premise, not an oversight.
	if n.Style != nil && n.Style.AccessibilityRole != core.RoleNone {
		t.Fatalf("an unroled row should still state no role, got %q", n.Style.AccessibilityRole)
	}

	html := htmlout.ExportHTML(n)
	if !strings.Contains(html, `aria-label="Ana, 3 unread"`) {
		t.Fatalf("the name did not reach the export:\n%s", html)
	}
	if !strings.Contains(html, `role="group"`) {
		t.Errorf("no role to carry the name, so no browser announces it:\n%s", html)
	}
}

// The two roles the widget *does* state keep the slot. A row that is one
// choice in a listbox, or one item at a depth, has already said what it is,
// and the fallback only ever fills an empty slot.
func TestARoledRowKeepsItsOwnRole(t *testing.T) {
	ctx := core.NewContext()
	for _, tc := range []struct {
		name string
		row  ListRow
		want string
	}{
		{"selectable", ListRow{Title: "Weekly", Selectable: true, Selected: true,
			AccessibilityLabel: "Weekly"}, `role="option"`},
		{"nested", ListRow{Title: "Matthew", NestingLevel: 2,
			AccessibilityLabel: "Matthew"}, `role="listitem"`},
	} {
		html := htmlout.ExportHTML(tc.row.Render(ctx))
		if !strings.Contains(html, tc.want) {
			t.Errorf("%s: missing %s:\n%s", tc.name, tc.want, html)
		}
		if strings.Contains(html, `role="group"`) {
			t.Errorf("%s: the fallback overwrote the row's own role:\n%s", tc.name, html)
		}
	}
}
