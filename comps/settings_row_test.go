package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// toggleHarness re-renders a row against a real piece of state after every
// dispatch, the way a Manager does, so the tests exercise the positional
// callback registry rather than a single stale closure.
type toggleHarness struct {
	t       *testing.T
	ctx     *core.Context
	value   bool
	changes int
	view    func(v bool, set func(bool)) core.View
	node    *core.Node
}

func newToggleHarness(t *testing.T, view func(v bool, set func(bool)) core.View) *toggleHarness {
	t.Helper()
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
	h := &toggleHarness{t: t, ctx: core.NewContext(), view: view}
	h.render()
	return h
}

func (h *toggleHarness) render() {
	h.t.Helper()
	h.ctx.BeginRenderPass()
	h.node = h.view(h.value, func(v bool) { h.value = v; h.changes++ }).Render(h.ctx)
	h.ctx.PurgeUnusedCallbacks()
	if dump := core.DumpConcerns(); dump != "" {
		h.t.Errorf("row raised concerns:\n%s", dump)
	}
}

// control returns the trailing Switch or Checkbox node.
func (h *toggleHarness) control() *core.Node {
	return findFirst(h.node, func(n *core.Node) bool { return n.Type == "Switch" || n.Type == "Checkbox" })
}

// tapRow dispatches the row's click and re-renders.
func (h *toggleHarness) tapRow() {
	h.t.Helper()
	id, ok := h.node.Props["onClick"].(string)
	if !ok {
		h.t.Fatal("row has no onClick")
	}
	h.ctx.TriggerCallback(id)
	h.render()
}

// toggleControl dispatches the control's bool callback with v and re-renders.
func (h *toggleHarness) toggleControl(v bool) {
	h.t.Helper()
	id, ok := h.control().Props["onToggle"].(string)
	if !ok {
		h.t.Fatal("control has no onToggle")
	}
	h.ctx.TriggerBoolCallback(id, v)
	h.render()
}

func switchRowView(title, subtitle string) func(bool, func(bool)) core.View {
	return func(v bool, set func(bool)) core.View {
		return SwitchRow{Title: title, Subtitle: subtitle, On: v, OnToggle: set}
	}
}

func checkboxRowView(title string) func(bool, func(bool)) core.View {
	return func(v bool, set func(bool)) core.View {
		return CheckboxRow{Title: title, Checked: v, OnToggle: set}
	}
}

func TestSwitchRowIsAListRowWithATrailingSwitch(t *testing.T) {
	h := newToggleHarness(t, switchRowView("Notifications", "Push and email"))

	n := h.node
	if n.Type != "Row" {
		t.Fatalf("root = %q, want ListRow's Row", n.Type)
	}
	last := n.Children[len(n.Children)-1]
	if last.Type != "Switch" {
		t.Fatalf("trailing slot = %q, want Switch", last.Type)
	}
	if middleOf(n) == nil || findText(middleOf(n), "Notifications") == nil || findText(middleOf(n), "Push and email") == nil {
		t.Error("title and subtitle belong in ListRow's growing middle")
	}
	if last.Props["checked"] != false {
		t.Errorf("checked = %v, want false", last.Props["checked"])
	}
	if last.Style.AccessibilityLabel != "Notifications" || last.Style.AccessibilityHint != "Push and email" {
		t.Errorf("control must be named by the row text, got label=%q hint=%q",
			last.Style.AccessibilityLabel, last.Style.AccessibilityHint)
	}
	if last.Style.Disabled {
		t.Error("an enabled row must not render a disabled control: it would draw greyed")
	}
	if n.Style.AccessibilityRole != "" || n.Style.AccessibilityLabel != "" {
		t.Error("the row container takes no role or label; the control is the thing announced")
	}
}

func TestCheckboxRowHasATrailingCheckbox(t *testing.T) {
	h := newToggleHarness(t, checkboxRowView("I agree to the terms"))
	if c := h.control(); c == nil || c.Type != "Checkbox" {
		t.Fatalf("want a Checkbox control, got %+v", c)
	}
}

// A tap anywhere on the row flips the state once.
func TestSettingsRowTapOnRowFlipsOnce(t *testing.T) {
	for name, view := range map[string]func(bool, func(bool)) core.View{
		"switch":   switchRowView("Wi-Fi", ""),
		"checkbox": checkboxRowView("Remember me"),
	} {
		t.Run(name, func(t *testing.T) {
			h := newToggleHarness(t, view)
			h.tapRow()
			if !h.value || h.changes != 1 {
				t.Fatalf("after one row tap: value=%v changes=%d, want true and 1", h.value, h.changes)
			}
			if h.control().Props["checked"] != true {
				t.Error("the re-render must draw the control in the new state")
			}
			h.tapRow()
			if h.value || h.changes != 2 {
				t.Errorf("after a second row tap: value=%v changes=%d, want false and 2", h.value, h.changes)
			}
		})
	}
}

// A tap on the control itself, which is all the natives dispatch.
func TestSettingsRowTapOnControlFlipsOnce(t *testing.T) {
	h := newToggleHarness(t, switchRowView("Wi-Fi", ""))
	h.toggleControl(true)
	if !h.value || h.changes != 1 {
		t.Errorf("value=%v changes=%d, want true and 1", h.value, h.changes)
	}
}

// The web case: one tap on the switch reaches Go as the row's click (bubbled)
// and then the input's change, with a render in between. Exactly one change.
func TestSettingsRowWebDoubleDispatchFlipsOnce(t *testing.T) {
	for _, start := range []bool{false, true} {
		h := newToggleHarness(t, switchRowView("Wi-Fi", ""))
		if start {
			h.value = true
			h.render()
		}
		h.tapRow()              // click bubbles to the row first
		h.toggleControl(!start) // then `change` reports the platform's new value
		if h.value != !start || h.changes != 1 {
			t.Errorf("start=%v: value=%v changes=%d, want %v and exactly 1",
				start, h.value, h.changes, !start)
		}
	}
}

// If both dispatches ever land inside one pass, they target the same value and
// the setter is idempotent — the state still ends where one tap puts it.
func TestSettingsRowSamePassDoubleDispatchIsIdempotent(t *testing.T) {
	h := newToggleHarness(t, switchRowView("Wi-Fi", ""))
	rowID := h.node.Props["onClick"].(string)
	ctlID := h.control().Props["onToggle"].(string)
	h.ctx.TriggerCallback(rowID)
	h.ctx.TriggerBoolCallback(ctlID, true)
	if !h.value {
		t.Errorf("value = %v, want true", h.value)
	}
}

func TestSettingsRowDisabledMarksRowAndControl(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := SwitchRow{Title: "Sync", On: true, OnToggle: func(bool) {}, Disabled: true}.Render(ctx)

	if !n.Style.Disabled {
		t.Error("a disabled row must carry Disabled so the hosts drop its tap")
	}
	if _, ok := n.Props["onClick"]; !ok {
		t.Error("the handler stays registered while disabled (core.Style.Disabled contract)")
	}
	ctl := findFirst(n, func(n *core.Node) bool { return n.Type == "Switch" })
	if ctl == nil || !ctl.Style.Disabled {
		t.Error("the switch must be disabled with the row")
	}
}

func TestSettingsRowNilOnToggleIsSafe(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := CheckboxRow{Title: "Read only"}.Render(ctx)
	ctx.TriggerCallback(n.Props["onClick"].(string))
	ctl := findFirst(n, func(n *core.Node) bool { return n.Type == "Checkbox" })
	ctx.TriggerBoolCallback(ctl.Props["onToggle"].(string), true)
}

func TestSettingsRowLeadingAndStyle(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := SwitchRow{
		Title:   "Dark mode",
		Leading: core.Text("🌙"),
		Style:   []core.StyleProp{core.Gap(2)},
	}.Render(ctx)
	if n.Children[0].Props["content"] != "🌙" {
		t.Error("Leading must render first")
	}
	if n.Style.Gap != 2 {
		t.Errorf("caller Style must reach the ListRow, gap = %v", n.Style.Gap)
	}
}
