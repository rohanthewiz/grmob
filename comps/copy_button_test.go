package comps

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// recordSystemEvents installs a system-event recorder for the test and
// removes it afterwards. CopyButton's whole effect is three system events
// (clipboard, haptic, toast), so the recorder is the only place a headless
// test can see a copy happen.
func recordSystemEvents(t *testing.T) *[]string {
	t.Helper()
	var seen []string
	core.SetSystemEventHandler(func(name string, data map[string]any) {
		switch name {
		case "clipboard":
			seen = append(seen, "clipboard:"+data["command"].(string)+":"+data["text"].(string))
		case "haptic":
			seen = append(seen, "haptic:"+data["kind"].(string))
		case "toast":
			seen = append(seen, "toast:"+data["message"].(string))
		default:
			seen = append(seen, name)
		}
	})
	t.Cleanup(func() { core.SetSystemEventHandler(nil) })
	return &seen
}

func copyButtonNode(t *testing.T, c CopyButton) (*core.Context, *core.Node) {
	t.Helper()
	ctx, n := renderDebug(t, c)
	b := buttonsOf(n)
	if len(b) != 1 {
		t.Fatalf("buttons = %d, want one", len(b))
	}
	return ctx, b[0]
}

// The defaults: caption "Copy", named by its caption, with a hint saying
// where the text goes.
func TestCopyButtonDefaults(t *testing.T) {
	_, b := copyButtonNode(t, CopyButton{Text: "abc"})
	if b.Props["label"] != "Copy" {
		t.Errorf("caption = %v, want Copy", b.Props["label"])
	}
	if b.Style.AccessibilityLabel != "Copy" {
		t.Errorf("name = %q, want the caption", b.Style.AccessibilityLabel)
	}
	if b.Style.AccessibilityHint != "Copies to the clipboard" {
		t.Errorf("hint = %q", b.Style.AccessibilityHint)
	}
	if b.Style.Disabled {
		t.Error("a button with text to copy is enabled")
	}
}

// A tap writes the text, ticks, and toasts, in that order and once each.
func TestCopyButtonTapCopiesAndConfirms(t *testing.T) {
	seen := recordSystemEvents(t)
	ctx, b := copyButtonNode(t, CopyButton{
		Text:               "cats://pair?code=42",
		Label:              "Copy link",
		CopiedMessage:      "Link copied",
		AccessibilityLabel: "Copy invite link",
	})
	if b.Style.AccessibilityLabel != "Copy invite link" {
		t.Errorf("name = %q, want the caller's", b.Style.AccessibilityLabel)
	}

	ctx.TriggerCallback(b.Props["onClick"].(string))

	want := []string{
		"clipboard:write:cats://pair?code=42",
		"haptic:light",
		"toast:Link copied",
	}
	if len(*seen) != len(want) {
		t.Fatalf("events = %v, want %v", *seen, want)
	}
	for i := range want {
		if (*seen)[i] != want[i] {
			t.Errorf("event %d = %q, want %q", i, (*seen)[i], want[i])
		}
	}
}

// Empty Text is inert: disabled, and a tap that arrives anyway (the race
// Button's Disabled doc describes) must not clear the clipboard.
func TestCopyButtonWithNothingToCopyIsInert(t *testing.T) {
	seen := recordSystemEvents(t)
	ctx, b := copyButtonNode(t, CopyButton{})
	if !b.Style.Disabled {
		t.Error("empty Text should disable the button")
	}
	ctx.TriggerCallback(b.Props["onClick"].(string))
	if len(*seen) != 0 {
		t.Errorf("events = %v, want none: an empty write clears the clipboard", *seen)
	}
}

// Disabled wins over a non-empty Text.
func TestCopyButtonDisabled(t *testing.T) {
	_, b := copyButtonNode(t, CopyButton{Text: "abc", Disabled: true})
	if !b.Style.Disabled {
		t.Error("Disabled should disable the button")
	}
}

// Stateless: rendered in one pass and dropped in the next, it must not move
// the slot of a state allocated after it. A hook inside the widget would
// hand that state the widget's slot on the second pass (and the debug hook
// check would report the changed count).
func TestCopyButtonTakesNoHookSlot(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })

	ctx := core.NewContext()
	var sentinel string
	pass := func(show bool) {
		ctx.BeginRenderPass()
		ctx.Reset()
		if show {
			CopyButton{Text: "abc"}.Render(ctx)
		}
		slot := core.NewState(ctx, "kept")
		sentinel = slot.Get()
		ctx.EndRenderPass()
	}

	pass(true)
	pass(false)
	pass(true)

	if sentinel != "kept" {
		t.Errorf("the state after the widget reads %q, want its own value", sentinel)
	}
	if dump := core.DumpConcerns(); dump != "" {
		t.Errorf("a conditional CopyButton raised concerns:\n%s", dump)
	}
}
