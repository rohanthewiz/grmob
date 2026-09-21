package comps

import (
	"slices"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// padLabels is the keys' drawn text, in tree order.
func padLabels(n *core.Node) []string {
	var out []string
	for _, b := range buttonsOf(n) {
		out = append(out, b.Props["label"].(string))
	}
	return out
}

// Twelve cells in the telephone layout: eleven buttons and a hidden blank when
// there is no Extra, twelve buttons when there is.
func TestNumberPadLayoutAndAccessibility(t *testing.T) {
	_, n := renderDebug(t, NumberPad{OnKey: func(string) {}, OnBackspace: func() {}})

	if n.Style.AccessibilityRole != core.RoleGroup || n.Style.AccessibilityLabel != "Number pad" {
		t.Errorf("role %q label %q", n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
	// The corner with no Extra is a key that is not painted: see blank().
	want := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", " ", "0", "⌫"}
	if got := padLabels(n); !slices.Equal(got, want) {
		t.Errorf("keys = %v, want %v", got, want)
	}
	if len(n.Children) != 4 {
		t.Fatalf("rows = %d, want 4", len(n.Children))
	}
	// The blank keeps the zero under the eight, and is scenery: a key's box,
	// unpainted, inert and unannounced.
	blank := n.Children[3].Children[0]
	if blank.Type != "Button" || !blank.Style.AccessibilityHidden || !blank.Style.Disabled {
		t.Errorf("the empty corner should be a hidden, disabled key: %s hidden=%v disabled=%v",
			blank.Type, blank.Style.AccessibilityHidden, blank.Style.Disabled)
	}
	if alpha, declared := blank.Style.OpacityFactor(); !declared || alpha != 0 {
		t.Errorf("the empty corner should be drawn at opacity 0, got %v (declared %v)", alpha, declared)
	}
	// Every cell is an equal share and a full-size touch target.
	for _, row := range n.Children {
		if len(row.Children) != 3 {
			t.Fatalf("a row has %d cells, want 3", len(row.Children))
		}
		for _, c := range row.Children {
			if c.Style.FlexGrow != 1 || c.Style.FlexBasis != "0" || c.Style.MinHeight != "56px" {
				t.Errorf("cell %v: grow %v basis %q min-height %q", c.Props["label"], c.Style.FlexGrow, c.Style.FlexBasis, c.Style.MinHeight)
			}
		}
	}
	keys := buttonsOf(n)
	if got := keys[len(keys)-1].Style.AccessibilityLabel; got != "Delete" {
		t.Errorf("backspace is named %q, want Delete", got)
	}
}

func TestNumberPadExtraKey(t *testing.T) {
	var got []string
	ctx, n := renderDebug(t, NumberPad{
		OnKey: func(k string) { got = append(got, k) }, OnBackspace: func() {},
		Extra: ".", ExtraLabel: "Decimal point",
	})
	keys := buttonsOf(n)
	if len(keys) != 12 {
		t.Fatalf("keys = %d, want 12", len(keys))
	}
	extra := keys[9]
	if extra.Props["label"] != "." || extra.Style.AccessibilityLabel != "Decimal point" {
		t.Errorf("extra key label %v name %q", extra.Props["label"], extra.Style.AccessibilityLabel)
	}
	ctx.TriggerCallback(extra.Props["onClick"].(string))
	if !slices.Equal(got, []string{"."}) {
		t.Errorf("reported %v, want [.]", got)
	}
}

// A key reports its text exactly once, with the light tick before it; the
// backspace reports through its own callback and never through OnKey.
func TestNumberPadKeysReportOnceWithAHaptic(t *testing.T) {
	seen := recordSystemEvents(t)
	var keys []string
	deletes := 0
	ctx, n := renderDebug(t, NumberPad{
		OnKey:       func(k string) { keys = append(keys, k) },
		OnBackspace: func() { deletes++ },
	})
	b := buttonsOf(n)
	ctx.TriggerCallback(b[4].Props["onClick"].(string))  // "5"
	ctx.TriggerCallback(b[11].Props["onClick"].(string)) // ⌫ (index 9 is the unpainted corner)
	if !slices.Equal(keys, []string{"5"}) || deletes != 1 {
		t.Errorf("keys %v deletes %d, want [5] and 1", keys, deletes)
	}
	if !slices.Equal(*seen, []string{"haptic:light", "haptic:light"}) {
		t.Errorf("system events = %v, want one light haptic per key", *seen)
	}
}

// Disabled: every key is drawn disabled, and a tap that races the disabling
// patch reports nothing and ticks nothing.
func TestNumberPadDisabledIsSilent(t *testing.T) {
	seen := recordSystemEvents(t)
	called := false
	ctx, n := renderDebug(t, NumberPad{
		OnKey: func(string) { called = true }, OnBackspace: func() { called = true },
		Disabled: true,
	})
	for _, b := range buttonsOf(n) {
		if !b.Style.Disabled {
			t.Errorf("key %v is not disabled", b.Props["label"])
		}
		ctx.TriggerCallback(b.Props["onClick"].(string))
	}
	if called || len(*seen) != 0 {
		t.Errorf("a disabled pad reported (called=%v, events=%v)", called, *seen)
	}
}

// No OnBackspace disables that key alone, and raises nothing: an append-only
// pad is legitimate.
func TestNumberPadWithoutBackspace(t *testing.T) {
	_, n := renderDebug(t, NumberPad{OnKey: func(string) {}})
	keys := buttonsOf(n)
	if !keys[11].Style.Disabled {
		t.Error("⌫ with no OnBackspace should be disabled")
	}
	if keys[0].Style.Disabled {
		t.Error("the digits should stay live")
	}
}

func TestNumberPadInertConcern(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	NumberPad{}.Render(ctx)
	ctx.EndRenderPass()
	if dump := core.DumpConcerns(); !strings.Contains(dump, ConcernNumberPadInert) {
		t.Errorf("want %s, got:\n%s", ConcernNumberPadInert, dump)
	}

	// Disabled says why nothing happens, so it is not a bug to report.
	core.ClearConcerns()
	ctx.BeginRenderPass()
	NumberPad{Disabled: true}.Render(ctx)
	ctx.EndRenderPass()
	if dump := core.DumpConcerns(); dump != "" {
		t.Errorf("a disabled pad raised:\n%s", dump)
	}
}
