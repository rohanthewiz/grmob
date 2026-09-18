package comps

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// passwordHarness renders a PasswordField through the rowHarness, which keeps
// one Context across passes so the reveal hook survives a re-render, and
// holds the value OnChange writes back.
type passwordHarness struct {
	*rowHarness
	value   string
	changes []string
}

func newPasswordHarness(t *testing.T) *passwordHarness {
	t.Helper()
	h := &passwordHarness{}
	h.rowHarness = newRowHarness(t, func() core.View {
		return PasswordField{
			Value:    h.value,
			OnChange: func(s string) { h.changes = append(h.changes, s); h.value = s },
			Label:    "Password",
		}
	})
	return h
}

func (h *passwordHarness) input() *core.Node {
	h.t.Helper()
	in := findFirst(h.node, func(n *core.Node) bool { return n.Type == "Input" || n.Type == "InputPassword" })
	if in == nil {
		h.t.Fatal("no input drawn")
	}
	return in
}

func (h *passwordHarness) toggle() *core.Node {
	h.t.Helper()
	b := buttonsOf(h.node)
	if len(b) != 1 {
		h.t.Fatalf("buttons = %d, want the one toggle", len(b))
	}
	return b[0]
}

// Hidden, then shown, then hidden: the element swaps, the caption follows,
// the accessible name does not, and the pressed state is stated both ways.
func TestPasswordFieldRevealTogglesTheInput(t *testing.T) {
	h := newPasswordHarness(t)

	in := h.input()
	if in.Type != "InputPassword" {
		t.Fatalf("input = %q, want the dots first", in.Type)
	}
	if in.Style.AccessibilityLabel != "Password" {
		t.Errorf("input name = %q, want Label", in.Style.AccessibilityLabel)
	}

	for pass, want := range []struct {
		input, caption string
		pressed        core.SelectedState
	}{
		{"InputPassword", "Show", core.SelectedOff},
		{"Input", "Hide", core.SelectedOn},
		{"InputPassword", "Show", core.SelectedOff},
	} {
		tg := h.toggle()
		if got := h.input().Type; got != want.input {
			t.Errorf("pass %d input = %q, want %q", pass, got, want.input)
		}
		if tg.Props["label"] != want.caption {
			t.Errorf("pass %d caption = %v, want %q", pass, tg.Props["label"], want.caption)
		}
		if tg.Style.AccessibilityLabel != "Show password" {
			t.Errorf("pass %d toggle name = %q, want it stable", pass, tg.Style.AccessibilityLabel)
		}
		if tg.Style.AccessibilitySelected != want.pressed {
			t.Errorf("pass %d pressed = %q, want %q", pass, tg.Style.AccessibilitySelected, want.pressed)
		}
		h.ctx.TriggerCallback(tg.Props["onClick"].(string))
		h.render()
	}
}

// Typing reaches the caller once per keystroke, and the text survives the
// swap because it was never the element's.
func TestPasswordFieldReportsTypingAndKeepsItAcrossTheSwap(t *testing.T) {
	h := newPasswordHarness(t)
	h.ctx.TriggerTextCallback(h.input().Props["onChange"].(string), "hunter2")
	h.render()
	if len(h.changes) != 1 || h.changes[0] != "hunter2" {
		t.Fatalf("changes = %v, want exactly one", h.changes)
	}

	h.ctx.TriggerCallback(h.toggle().Props["onClick"].(string))
	h.render()
	if in := h.input(); in.Type != "Input" || in.Props["value"] != "hunter2" {
		t.Errorf("revealed input = %q holding %v, want the same text in the open", in.Type, in.Props["value"])
	}
	if len(h.changes) != 1 {
		t.Errorf("changes = %v; revealing should report nothing", h.changes)
	}
}

func TestPasswordFieldWithoutOnChangeIsAConcern(t *testing.T) {
	newQuietRowHarness(t, func() core.View { return PasswordField{Label: "Password"} })
	if !strings.Contains(core.DumpConcerns(), ConcernPasswordFieldInert) {
		t.Errorf("want %s, got:\n%s", ConcernPasswordFieldInert, core.DumpConcerns())
	}
}
