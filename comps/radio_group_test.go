package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func shippingGroup(value string, onChange func(string)) RadioGroup {
	return RadioGroup{
		Label: "Shipping",
		Options: []RadioOption{
			{Value: "std", Label: "Standard", Subtitle: "3–5 days"},
			{Value: "exp", Label: "Express", Subtitle: "Next day"},
			{Value: "pick", Label: "Pick up", Disabled: true},
		},
		Value:    value,
		OnChange: onChange,
	}
}

// optionRows returns the option rows in order.
func optionRows(n *core.Node) []*core.Node {
	var out []*core.Node
	for _, c := range n.Children {
		if c.Style != nil && c.Style.AccessibilityRole == core.RoleOption {
			out = append(out, c)
		}
	}
	return out
}

func TestRadioGroupIsALabelledListboxOfOptions(t *testing.T) {
	_, n := renderDebug(t, shippingGroup("exp", func(string) {}))

	if n.Type != "Column" || n.Style.AccessibilityRole != core.RoleListBox || n.Style.AccessibilityLabel != "Shipping" {
		t.Fatalf("root = %q role %q label %q", n.Type, n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
	rows := optionRows(n)
	if len(rows) != 3 {
		t.Fatalf("want 3 option rows, got %d", len(rows))
	}
	for i, want := range []core.SelectedState{core.SelectedWhen(false), core.SelectedWhen(true), core.SelectedWhen(false)} {
		if rows[i].Style.AccessibilitySelected != want {
			t.Errorf("row %d selected = %q, want %q: every option states its selection",
				i, rows[i].Style.AccessibilitySelected, want)
		}
	}
	if rows[0].Style.AccessibilityLabel != "Standard" || rows[0].Style.AccessibilityHint != "3–5 days" {
		t.Errorf("row name/hint = %q / %q", rows[0].Style.AccessibilityLabel, rows[0].Style.AccessibilityHint)
	}
	if rows[1].Style.AccessibilityLabel != "Express" {
		t.Errorf("a Selectable row must not append \", selected\", got %q", rows[1].Style.AccessibilityLabel)
	}
	if rows[1].Style.Background == core.DefaultTheme.Colors.Surface {
		t.Error("the ring is the selection cue; the selected row takes no tint")
	}
}

func TestRadioGroupRingFillsOnlyTheSelectedOption(t *testing.T) {
	_, n := renderDebug(t, shippingGroup("exp", func(string) {}))
	rows := optionRows(n)

	ringOf := func(row *core.Node) *core.Node { return row.Children[0] }
	std, exp, pick := ringOf(rows[0]), ringOf(rows[1]), ringOf(rows[2])

	if !std.Style.AccessibilityHidden {
		t.Error("the ring is drawn, not a control: hidden from assistive technology")
	}
	if len(std.Children) != 0 || len(exp.Children) != 1 {
		t.Errorf("dots: std %d, exp %d; want 0 and 1", len(std.Children), len(exp.Children))
	}
	th := core.DefaultTheme
	if exp.Style.BorderColor != th.Colors.PrimaryOnLightColor() {
		t.Errorf("selected ring = %q", exp.Style.BorderColor)
	}
	if pick.Style.BorderColor != th.Colors.TextSecondary {
		t.Errorf("disabled ring = %q", pick.Style.BorderColor)
	}
	if std.Style.Width != "20px" || std.Style.BorderRadius != 10 {
		t.Errorf("ring = %s radius %v", std.Style.Width, std.Style.BorderRadius)
	}
}

func TestRadioGroupReportsOnlyRealChanges(t *testing.T) {
	var got []string
	ctx, n := renderDebug(t, shippingGroup("std", func(v string) { got = append(got, v) }))
	rows := optionRows(n)

	ctx.TriggerCallback(rows[0].Props["onClick"].(string)) // already selected
	ctx.TriggerCallback(rows[1].Props["onClick"].(string)) // a change
	ctx.TriggerCallback(rows[2].Props["onClick"].(string)) // disabled
	if len(got) != 1 || got[0] != "exp" {
		t.Errorf("changes = %v, want [exp]", got)
	}
	if !rows[2].Style.Disabled {
		t.Error("a disabled option carries the platform disabled flag")
	}
}

func TestRadioGroupWithoutOnChangeRegistersNothing(t *testing.T) {
	_, n := renderDebug(t, shippingGroup("std", nil))
	if findFirst(n, func(n *core.Node) bool { _, ok := n.Props["onClick"]; return ok }) != nil {
		t.Error("a display-only group must register no callbacks")
	}
}

func TestRadioGroupDisabledDisablesEveryOption(t *testing.T) {
	g := shippingGroup("std", func(string) { t.Error("a disabled group must not change") })
	g.Disabled = true
	ctx, n := renderDebug(t, g)
	for i, row := range optionRows(n) {
		if !row.Style.Disabled {
			t.Errorf("row %d not disabled", i)
		}
		ctx.TriggerCallback(row.Props["onClick"].(string))
	}
}
