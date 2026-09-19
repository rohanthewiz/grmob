package verify

import (
	"strings"
	"testing"
)

// core.Style.Inert on Compose, and core.Focus on a Button: the two halves of
// comps.Drawer's keyboard story on Android. Both were checked on a Galaxy Z
// Fold6 with a USB keyboard.
//
//	Drawer shut                          Drawer open
//	  screen   ☰ …   (focusable)           screen  Inert → canFocus = false
//	  panel    Inert → canFocus = false     panel   ✕ ← core.Focus(CloseRef)
//	                                        Back → OnDismiss → core.Focus(☰)
//
// Before either, Tab went through the shut panel's ✕ and rows unseen and
// unheard, and opening or closing the drawer left focus behind.
func TestComposeReadsInertAsNoKeyboardFocus(t *testing.T) {
	if src := valuesIn(t, kotlinStyle); !strings.Contains(src, `inert = obj.optBoolean("Inert", false)`) {
		t.Errorf("%s: the Inert field is no longer parsed — a shut Drawer's panel takes "+
			"Tab stops again", kotlinStyle)
	}
	render := codeOf(t, kotlinRenderer, "fun RenderNode(")
	for _, pin := range []struct{ expr, why string }{
		{"val inertHere = LocalGrMobInert.current || node.style?.inert == true",
			"the node itself and everything under it"},
		{"if (inertHere) mods = Modifier.focusProperties { canFocus = false }.then(mods)",
			"at the head of each node's chain, since focusProperties covers only " +
				"the focus targets after it, never a whole subtree"},
		{"if (inert) add(LocalGrMobInert provides true)",
			"opened once, at the outermost inert node"},
	} {
		if !strings.Contains(render, pin.expr) {
			t.Errorf("%s: RenderNode has lost %q — %s", kotlinRenderer, pin.expr, pin.why)
		}
	}
}

func TestComposeButtonTakesAFocusCommand(t *testing.T) {
	button := valuesOf(t, kotlinRenderer, "private fun GrMobButton(")
	for _, pin := range []struct{ expr, why string }{
		{`val focusEpoch = node.intProp("focusEpoch")`, "the command's when"},
		{`val focusAction = node.stringProp("focusAction")`, "the command's what"},
		{"LaunchedEffect(focusEpoch) {", "re-fired by a new epoch, as the field's is"},
		{"runCatching { focusRequester.requestFocus() }", "the focus itself"},
		// The Drawer's ✕ becomes focusable (its panel stops being Inert) in the
		// same pass as the ☰'s handler focuses it; a request made before the
		// focus tree takes that in is refused.
		{"withFrameNanos { }", "one frame before the request"},
		{"val focusable = extra.focusRequester(focusRequester)",
			"on `extra`, so the long-press button is covered as well as material3's"},
		{"GrMobButtonControl(node, focusable)", "and the control is handed that chain"},
	} {
		if !strings.Contains(button, pin.expr) {
			t.Errorf("%s: GrMobButton has lost %q — %s", kotlinRenderer, pin.expr, pin.why)
		}
	}
}
