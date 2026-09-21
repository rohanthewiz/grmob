package comps

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// maskedField renders the widget and returns its one Input node.
func maskedField(t *testing.T, in MaskedInput) (*core.Context, *core.Node) {
	t.Helper()
	ctx, n := renderDebug(t, in)
	if n.Type != "Input" {
		t.Fatalf("a MaskedInput is one %s, want Input", n.Type)
	}
	return ctx, n
}

func TestMaskedInputDrawsTheFormattedValue(t *testing.T) {
	_, n := maskedField(t, MaskedInput{
		Mask: phoneMask, Value: "5551234", Label: "Phone", Hint: "Ten digits",
		Keyboard: core.KeyboardDigits, OnChange: func(string, string) {},
	})
	if got := n.Props["value"]; got != "(555) 123-4" {
		t.Errorf("value = %q, want (555) 123-4", got)
	}
	if got := n.Props["placeholder"]; got != "(___) ___-____" {
		t.Errorf("placeholder = %q, want the mask's shape", got)
	}
	if n.Style.AccessibilityLabel != "Phone" || n.Style.AccessibilityHint != "Ten digits" {
		t.Errorf("label %q hint %q", n.Style.AccessibilityLabel, n.Style.AccessibilityHint)
	}
	if got := n.Props["keyboard"]; got != string(core.KeyboardDigits) {
		t.Errorf("keyboard = %v, want digits", got)
	}
}

// One edit, one report, carrying both forms. The text handed back by a host is
// the formatted text with the new key in it.
func TestMaskedInputReportsRawAndFormatted(t *testing.T) {
	var got [][2]string
	ctx, n := maskedField(t, MaskedInput{
		Mask: phoneMask, Value: "555",
		OnChange: func(raw, formatted string) { got = append(got, [2]string{raw, formatted}) },
	})
	ctx.TriggerTextCallback(n.Props["onChange"].(string), "(5556")
	want := [][2]string{{"5556", "(555) 6"}}
	if !slices.Equal(got, want) {
		t.Errorf("reports = %v, want %v", got, want)
	}
}

// An echo of Go's own text and a key the mask refuses both leave the raw value
// as it was, and neither is news.
func TestMaskedInputIgnoresEchoesAndRefusedKeys(t *testing.T) {
	calls := 0
	ctx, n := maskedField(t, MaskedInput{
		Mask: phoneMask, Value: "555", OnChange: func(string, string) { calls++ },
	})
	id := n.Props["onChange"].(string)
	ctx.TriggerTextCallback(id, "(555")  // the echo
	ctx.TriggerTextCallback(id, "(555x") // a letter in a digit slot
	ctx.TriggerTextCallback(id, "(555)") // a literal typed by hand
	if calls != 0 {
		t.Errorf("OnChange ran %d times for edits that changed nothing", calls)
	}
}

// OnComplete fires on the edit that fills the last slot, and again on a
// correction to a full value; a refused key on a full value is neither.
func TestMaskedInputOnComplete(t *testing.T) {
	var done []string
	in := MaskedInput{
		Mask: expiryMask, Value: "123",
		OnChange:   func(string, string) {},
		OnComplete: func(raw string) { done = append(done, raw) },
	}
	ctx, n := maskedField(t, in)
	ctx.TriggerTextCallback(n.Props["onChange"].(string), "12/34")
	if !slices.Equal(done, []string{"1234"}) {
		t.Fatalf("complete = %v, want [1234]", done)
	}

	in.Value = "1234"
	ctx, n = maskedField(t, in)
	id := n.Props["onChange"].(string)
	ctx.TriggerTextCallback(id, "12/345") // past the last slot: refused
	ctx.TriggerTextCallback(id, "12/35")  // a correction
	if !slices.Equal(done, []string{"1234", "1235"}) {
		t.Errorf("complete = %v, want [1234 1235]", done)
	}
}

// A Value the mask cannot draw is normalised before it is compared, so the
// first real edit is judged against what the field shows.
func TestMaskedInputNormalisesValue(t *testing.T) {
	var got []string
	ctx, n := maskedField(t, MaskedInput{
		Mask: expiryMask, Value: "1x2", OnChange: func(raw, _ string) { got = append(got, raw) },
	})
	if n.Props["value"] != "12" {
		t.Fatalf("value = %q, want 12", n.Props["value"])
	}
	ctx.TriggerTextCallback(n.Props["onChange"].(string), "12") // the echo of what is drawn
	if len(got) != 0 {
		t.Errorf("the echo of a normalised value reported %v", got)
	}
}

func TestMaskedInputDisabled(t *testing.T) {
	calls := 0
	ctx, n := maskedField(t, MaskedInput{
		Mask: phoneMask, Disabled: true, OnChange: func(string, string) { calls++ },
		// A caller's style must not be able to re-enable it.
		Style: []core.StyleProp{core.Disabled(false)},
	})
	if !n.Style.Disabled {
		t.Error("the field should be disabled")
	}
	ctx.TriggerTextCallback(n.Props["onChange"].(string), "5")
	if calls != 0 {
		t.Error("a disabled field reported an edit")
	}
}

func TestMaskedInputConcerns(t *testing.T) {
	for _, c := range []struct {
		name string
		in   MaskedInput
		want string
	}{
		{"no OnChange", MaskedInput{Mask: phoneMask}, ConcernMaskedInputInert},
		{"another library's alphabet", MaskedInput{Mask: "999-999", OnChange: func(string, string) {}}, ConcernMaskedInputNoSlots},
	} {
		t.Run(c.name, func(t *testing.T) {
			core.SetDebugMode(true)
			core.ClearConcerns()
			t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
			ctx := core.NewContext()
			ctx.BeginRenderPass()
			c.in.Render(ctx)
			ctx.EndRenderPass()
			if dump := core.DumpConcerns(); !strings.Contains(dump, c.want) {
				t.Errorf("want %s, got:\n%s", c.want, dump)
			}
		})
	}
}

// A refused key on a native host. The handler changes no state, so the only
// thing that can take the key back out of the field is the text-edit ledger:
// the host said it shows "(555x", the next render says "(555", and that must
// reach the host as a rewrite (a higher editEpoch), not be swallowed as an
// unchanged value. The widget's doc and the web runtime's dispatchFromElement
// both lean on this being true.
func TestMaskedInputRefusedKeyIsRewrittenOnANativeHost(t *testing.T) {
	mgr := render.New(core.NewContext(), func(ctx *core.Context) core.View {
		raw := core.NewState(ctx, "555")
		return MaskedInput{Mask: phoneMask, Value: raw.Get(),
			OnChange: func(r, _ string) { raw.Set(r) }}
	})
	defer mgr.Close()

	var root struct {
		Props map[string]any
	}
	if err := json.Unmarshal([]byte(mgr.RenderInitial()), &root); err != nil {
		t.Fatal(err)
	}
	id := root.Props["onChange"].(string)

	// An accepted key first, which is what makes the host "sequenced".
	accepted := mgr.DispatchTextEdit(id, "(5556", 1, 0)
	if !strings.Contains(accepted, `"value":"(555) 6"`) || !strings.Contains(accepted, `"editEpoch":1`) {
		t.Fatalf("the formatted text should arrive as rewrite 1: %s", accepted)
	}
	refused := mgr.DispatchTextEdit(id, "(555) 6x", 2, 1)
	if !strings.Contains(refused, `"editEpoch":2`) || !strings.Contains(refused, `"value":"(555) 6"`) {
		t.Errorf("a refused key should come back as a rewrite to Go's text: %s", refused)
	}
}
