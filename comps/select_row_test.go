package comps

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// themeOptions is the list every test here picks from: three options, the
// middle one carrying no Label so the value-as-label default is exercised by
// the same fixture that exercises everything else.
func themeOptions() []core.SelectOption {
	return []core.SelectOption{
		core.Option("system", "System"),
		{Value: "light"},
		core.Option("dark", "Dark"),
	}
}

// rowHarness re-renders a state-owning row against one context, the way a
// Manager does, so a tap lands in the registry the next pass reads. SelectRow
// owns its open/shut state, so its tests must respect the pass protocol rather
// than calling Render bare.
type rowHarness struct {
	t    *testing.T
	ctx  *core.Context
	view func() core.View
	node *core.Node
}

func newRowHarness(t *testing.T, view func() core.View) *rowHarness {
	t.Helper()
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
	h := &rowHarness{t: t, ctx: core.NewContext(), view: view}
	h.render()
	return h
}

// newQuietRowHarness is newRowHarness for the one test whose subject is a
// concern: its very first pass raises it, so the constructor cannot assert
// that none was.
func newQuietRowHarness(t *testing.T, view func() core.View) *rowHarness {
	t.Helper()
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
	h := &rowHarness{t: t, ctx: core.NewContext(), view: view}
	h.renderQuietly()
	return h
}

// render drives one pass, audits the finished tree as render.Manager would
// (see renderDebug), and fails on any concern either raised.
func (h *rowHarness) render() {
	h.t.Helper()
	h.ctx.BeginRenderPass()
	h.ctx.Reset()
	h.node = h.view().Render(h.ctx)
	h.ctx.EndRenderPass()
	core.AuditTree(h.node)
	if dump := core.DumpConcerns(); dump != "" {
		h.t.Errorf("row raised concerns:\n%s", dump)
	}
}

// renderQuietly is render without the concern assertion, for the one test that
// is about a concern being raised.
func (h *rowHarness) renderQuietly() {
	h.t.Helper()
	h.ctx.BeginRenderPass()
	h.ctx.Reset()
	h.node = h.view().Render(h.ctx)
	h.ctx.EndRenderPass()
}

// row and sheet name the two halves of SelectRow's tree.
func (h *rowHarness) row() *core.Node {
	h.t.Helper()
	if h.node.Type != "Box" || len(h.node.Children) != 2 {
		h.t.Fatalf("root = %q with %d children, want a Box over the row and the sheet",
			h.node.Type, len(h.node.Children))
	}
	return h.node.Children[0]
}

func (h *rowHarness) sheet() *core.Node {
	h.t.Helper()
	m := h.node.Children[1]
	if m.Type != "Modal" {
		h.t.Fatalf("second child = %q, want the sheet's Modal", m.Type)
	}
	return m
}

func (h *rowHarness) sheetOpen() bool {
	return h.sheet().Props["visible"] == true
}

// tapRow dispatches the row's click and re-renders.
func (h *rowHarness) tapRow() {
	h.t.Helper()
	id, ok := h.row().Props["onClick"].(string)
	if !ok {
		h.t.Fatal("the row carries no onClick")
	}
	h.ctx.TriggerCallback(id)
	h.render()
}

// tapAction dispatches the sheet button whose label contains want.
func (h *rowHarness) tapAction(want string) {
	h.t.Helper()
	for _, b := range buttonsOf(h.node) {
		label, _ := b.Props["label"].(string)
		if !strings.Contains(label, want) {
			continue
		}
		id, ok := b.Props["onClick"].(string)
		if !ok {
			h.t.Fatalf("action %q carries no onClick", label)
		}
		h.ctx.TriggerCallback(id)
		h.render()
		return
	}
	h.t.Fatalf("no sheet action matching %q", want)
}

func TestSelectRowShowsTheChosenLabelAndOpensASheet(t *testing.T) {
	h := newRowHarness(t, func() core.View {
		return SelectRow{Title: "Theme", Options: themeOptions(), Value: "dark", OnChange: func(string) {}}
	})

	if findText(h.node, "Theme") == nil {
		t.Error("the row should show its title")
	}
	if findText(h.node, "Dark") == nil {
		t.Error("the trailing slot should show the chosen option's label")
	}
	if h.sheetOpen() {
		t.Fatal("the sheet starts shut")
	}

	h.tapRow()
	if !h.sheetOpen() {
		t.Error("tapping the row should open the sheet")
	}
}

// The row is the control here, unlike SwitchRow where the switch is: it takes
// a role and says what the tap opens.
func TestSelectRowIsAButtonThatOpensADialog(t *testing.T) {
	h := newRowHarness(t, func() core.View {
		return SelectRow{Title: "Theme", Options: themeOptions(), Value: "dark", OnChange: func(string) {}}
	})

	row := h.row()
	if row.Style.AccessibilityRole != core.RoleButton {
		t.Errorf("row role = %q, want button: a Row is scenery until a role says otherwise",
			row.Style.AccessibilityRole)
	}
	if row.Style.AccessibilityHasPopup != core.PopupDialog {
		t.Errorf("row popup = %q, want dialog", row.Style.AccessibilityHasPopup)
	}
	// The name comes from the row's own text, as ListRow's rule requires.
	if row.Style.AccessibilityLabel != "" {
		t.Errorf("row label = %q, want none: the title and the value already speak",
			row.Style.AccessibilityLabel)
	}
}

func TestSelectRowPicksOnceAndCloses(t *testing.T) {
	value := "dark"
	changes := 0
	h := newRowHarness(t, func() core.View {
		return SelectRow{
			Title: "Theme", Options: themeOptions(), Value: value,
			OnChange: func(v string) { value = v; changes++ },
		}
	})

	h.tapRow()
	h.tapAction("System")

	if changes != 1 {
		t.Errorf("OnChange fired %d times, want exactly one per pick", changes)
	}
	if value != "system" {
		t.Errorf("value = %q, want the picked option's Value", value)
	}
	if h.sheetOpen() {
		t.Error("picking closes the sheet")
	}
	if findText(h.node, "System") == nil {
		t.Error("the trailing slot should follow the new value")
	}
}

// Choosing what is already chosen is a dismissal, not a change: the sheet
// closes and the setter is not called. Same guard SwitchRow's setter applies.
func TestSelectRowPickingTheCurrentValueReportsNothing(t *testing.T) {
	changes := 0
	h := newRowHarness(t, func() core.View {
		return SelectRow{
			Title: "Theme", Options: themeOptions(), Value: "dark",
			OnChange: func(string) { changes++ },
		}
	})

	h.tapRow()
	h.tapAction("Dark")

	if changes != 0 {
		t.Errorf("OnChange fired %d times for a re-pick of the current value, want none", changes)
	}
	if h.sheetOpen() {
		t.Error("the sheet still closes: it was dismissed by choosing")
	}
}

func TestSelectRowChecksTheCurrentOptionAndDefaultsALabelToItsValue(t *testing.T) {
	h := newRowHarness(t, func() core.View {
		return SelectRow{Title: "Theme", Options: themeOptions(), Value: "dark", OnChange: func(string) {}}
	})
	h.tapRow()

	btns := buttonsOf(h.node)
	if len(btns) != 4 {
		t.Fatalf("want three options and a cancel, got %d buttons", len(btns))
	}
	if btns[1].Props["label"] != "light" {
		t.Errorf("option 2 label = %v, want its Value: an empty Label means the value reads well enough",
			btns[1].Props["label"])
	}
	if btns[2].Props["label"] != "✓ Dark" {
		t.Errorf("the chosen action = %v, want the check mark", btns[2].Props["label"])
	}
	if btns[2].Style.AccessibilityCurrent != core.CurrentTrue {
		t.Errorf("the chosen action's current = %q, want true", btns[2].Style.AccessibilityCurrent)
	}
	if btns[3].Props["label"] != "Cancel" {
		t.Errorf("last button = %v, want the default Cancel", btns[3].Props["label"])
	}
}

func TestSelectRowSheetTitleFallsBackToTheRowTitle(t *testing.T) {
	h := newRowHarness(t, func() core.View {
		return SelectRow{Title: "Theme", Options: themeOptions(), Value: "dark", OnChange: func(string) {}}
	})
	card := h.sheet().Children[1]
	if card.Style.AccessibilityLabel != "Theme" {
		t.Errorf("sheet name = %q, want the row's title", card.Style.AccessibilityLabel)
	}

	h2 := newRowHarness(t, func() core.View {
		return SelectRow{
			Title: "Sort", SheetTitle: "Sort by", Options: themeOptions(),
			Value: "dark", OnChange: func(string) {},
		}
	})
	if got := h2.sheet().Children[1].Style.AccessibilityLabel; got != "Sort by" {
		t.Errorf("sheet name = %q, want SheetTitle to win", got)
	}
}

// Disabled is inert on both halves: the row does not open, and the actions
// keep their handlers but refuse them (core.Disabled's contract).
func TestSelectRowDisabledNeitherOpensNorPicks(t *testing.T) {
	changes := 0
	h := newRowHarness(t, func() core.View {
		return SelectRow{
			Title: "Theme", Options: themeOptions(), Value: "dark", Disabled: true,
			OnChange: func(string) { changes++ },
		}
	})

	if !h.row().Style.Disabled {
		t.Error("a disabled row must carry core.Disabled")
	}
	h.tapRow()
	if h.sheetOpen() {
		t.Error("a disabled row does not open its sheet")
	}
	h.tapAction("System")
	if changes != 0 {
		t.Errorf("OnChange fired %d times from a disabled row, want none", changes)
	}
}

func TestSelectRowDisablesAnOptionAndItsRun(t *testing.T) {
	h := newRowHarness(t, func() core.View {
		return SelectRow{
			Title: "Plan",
			Options: []core.SelectOption{
				core.Option("free", "Free"),
				{Value: "pro", Label: "Pro", Disabled: true},
				{Value: "team", Label: "Team", Group: "Paid", GroupDisabled: true},
			},
			Value: "free", OnChange: func(string) {},
		}
	})
	h.tapRow()

	btns := buttonsOf(h.node)
	if btns[1].Style.Disabled != true {
		t.Error("an option's own Disabled must reach its action")
	}
	if btns[2].Style.Disabled != true {
		t.Error("GroupDisabled must reach its action too, read per option")
	}
	// The group is a heading a sheet cannot draw; it must not leak into the
	// label either.
	if label, _ := btns[2].Props["label"].(string); strings.Contains(label, "Paid") {
		t.Errorf("action label = %q, want no heading text in it", label)
	}
}

// A value no option carries is invisible on screen — the row looks unset — so
// debug builds say so.
func TestSelectRowReportsAValueThatIsNotAnOption(t *testing.T) {
	h := newQuietRowHarness(t, func() core.View {
		return SelectRow{
			Title: "Theme", Options: themeOptions(), Value: "sepia",
			Placeholder: "Not set", OnChange: func(string) {},
		}
	})

	dump := core.DumpConcerns()
	if !strings.Contains(dump, ConcernSelectRowValueNotAnOption) {
		t.Errorf("want %s in the concerns, got:\n%s", ConcernSelectRowValueNotAnOption, dump)
	}
	if findText(h.node, "Not set") == nil {
		t.Error("an unmatched value falls back to the placeholder")
	}
	core.ClearConcerns()
}

// An empty value is a legitimate starting state and says nothing.
func TestSelectRowEmptyValueIsQuiet(t *testing.T) {
	h := newRowHarness(t, func() core.View {
		return SelectRow{Title: "Theme", Options: themeOptions(), Placeholder: "Choose", OnChange: func(string) {}}
	})
	if findText(h.node, "Choose") == nil {
		t.Error("an unset row shows its placeholder")
	}
}
