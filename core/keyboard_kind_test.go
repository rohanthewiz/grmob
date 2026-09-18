package core

import "testing"

// core.Keyboard writes its kind on the field, and the text keyboard writes
// nothing, so a tree that never asks is unchanged.
func TestKeyboardPropIsWrittenOnlyWhenAsked(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()
	n := Input("", "", func(string) {}, Keyboard(KeyboardDigits)).Render(ctx)
	if n.Props["keyboard"] != "digits" {
		t.Errorf("keyboard = %v, want digits", n.Props["keyboard"])
	}
	plain := Input("", "", func(string) {}, Keyboard(KeyboardText)).Render(ctx)
	if _, ok := plain.Props["keyboard"]; ok {
		t.Error("the text keyboard should write no prop")
	}
}
