package comps

import (
	"strconv"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// renderDebug renders v under debug mode and fails on any concern, so every
// widget test also proves the a11y audit and hook checks are quiet.
func renderDebug(t *testing.T, v core.View) (*core.Context, *core.Node) {
	t.Helper()
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := v.Render(ctx)
	ctx.EndRenderPass()
	if dump := core.DumpConcerns(); dump != "" {
		t.Errorf("concerns raised:\n%s", dump)
	}
	return ctx, n
}

func stepperButtons(n *core.Node) (dec, inc *core.Node) {
	b := buttonsOf(n)
	return b[0], b[1]
}

func TestStepperStructureAndAccessibility(t *testing.T) {
	_, n := renderDebug(t, Stepper{Value: 3, Min: 1, Max: 20, Label: "Quantity", OnChange: func(int) {}})

	if n.Type != "Row" || len(n.Children) != 3 {
		t.Fatalf("want Row(−, value, +), got %q with %d children", n.Type, len(n.Children))
	}
	if n.Style.AccessibilityRole != core.RoleGroup || n.Style.AccessibilityLabel != "Quantity" {
		t.Errorf("group a11y = role %q label %q", n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
	if v := n.Style.AccessibilityValue; v.Text != "3" || v.Now != "3" || v.Min != "1" || v.Max != "20" {
		t.Errorf("value = %+v, want 3 in [1,20] with text 3", v)
	}
	if findText(n, "3") == nil {
		t.Error("the value is drawn between the buttons")
	}
	dec, inc := stepperButtons(n)
	if dec.Props["label"] != "−" || inc.Props["label"] != "+" {
		t.Errorf("button order = %v, %v", dec.Props["label"], inc.Props["label"])
	}
	if dec.Style.AccessibilityLabel != "Decrease" || inc.Style.AccessibilityLabel != "Increase" {
		t.Error("the glyph buttons need spoken names")
	}
}

func TestStepperTapsClampAndReportOnlyChanges(t *testing.T) {
	var got []int
	ctx, n := renderDebug(t, Stepper{Value: 19, Min: 1, Max: 20, Step: 5, OnChange: func(v int) { got = append(got, v) }})
	dec, inc := stepperButtons(n)

	ctx.TriggerCallback(inc.Props["onClick"].(string))
	ctx.TriggerCallback(dec.Props["onClick"].(string))
	if len(got) != 2 || got[0] != 20 || got[1] != 14 {
		t.Errorf("changes = %v, want [20 14]: +5 clamps to Max, −5 steps", got)
	}
}

func TestStepperDisablesTheButtonAtEachBound(t *testing.T) {
	_, low := renderDebug(t, Stepper{Value: 1, Min: 1, Max: 3})
	dec, inc := stepperButtons(low)
	if !dec.Style.Disabled || inc.Style.Disabled {
		t.Errorf("at Min: dec disabled=%v inc disabled=%v", dec.Style.Disabled, inc.Style.Disabled)
	}

	fired := false
	ctx, high := renderDebug(t, Stepper{Value: 3, Min: 1, Max: 3, OnChange: func(int) { fired = true }})
	dec, inc = stepperButtons(high)
	if dec.Style.Disabled || !inc.Style.Disabled {
		t.Errorf("at Max: dec disabled=%v inc disabled=%v", dec.Style.Disabled, inc.Style.Disabled)
	}
	ctx.TriggerCallback(inc.Props["onClick"].(string))
	if fired {
		t.Error("a disabled + must not report a change")
	}
}

// A zero Min and Max is no range at all, so nothing is clamped and no numeric
// range is stated; the text still reaches the natives.
func TestStepperUnboundedByDefault(t *testing.T) {
	var got int
	ctx, n := renderDebug(t, Stepper{Value: 0, OnChange: func(v int) { got = v }})
	dec, _ := stepperButtons(n)
	if dec.Style.Disabled {
		t.Error("an unbounded stepper never disables")
	}
	ctx.TriggerCallback(dec.Props["onClick"].(string))
	if got != -1 {
		t.Errorf("got %d, want -1", got)
	}
	if v := n.Style.AccessibilityValue; v.Now != "" || v.Text != "0" {
		t.Errorf("value = %+v, want text only", v)
	}
}

func TestStepperFormatLabelsAndStyle(t *testing.T) {
	_, n := renderDebug(t, Stepper{
		Value:         2,
		Format:        func(v int) string { return strconv.Itoa(v) + " guests" },
		DecreaseLabel: "Menos",
		IncreaseLabel: "Mais",
		Disabled:      true,
		Style:         []core.StyleProp{core.Gap(1)},
	})
	if findText(n, "2 guests") == nil || n.Style.AccessibilityValue.Text != "2 guests" {
		t.Error("Format drives both the drawn and the announced value")
	}
	dec, inc := stepperButtons(n)
	if dec.Style.AccessibilityLabel != "Menos" || inc.Style.AccessibilityLabel != "Mais" {
		t.Error("button names should be overridable")
	}
	if !dec.Style.Disabled || !inc.Style.Disabled {
		t.Error("Disabled disables both buttons")
	}
	if n.Style.Gap != 1 {
		t.Errorf("Style should beat the default gap, got %v", n.Style.Gap)
	}
}
