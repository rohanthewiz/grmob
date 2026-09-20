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

// The one control the head-of-chain modifier cannot reach.
//
// Compose resolves a focus target's properties by walking up the modifier
// chain from the target, and the walk stops at the first FocusTarget it meets
// (visitSelfAndAncestors takes untilType = Nodes.FocusTarget). A scroll
// container delegates a focus target, so any field composed behind one inside
// a single GrMob node is cut off from the `extra` RenderNode prepended:
//
//	Row(extra: canFocus = false)        ← where Inert lands
//	 └ verticalScroll     ── FocusTarget
//	    └ horizontalScroll ── FocusTarget   ← the walk from the field ends here
//	       └ BasicTextField ── FocusTarget  ← never sees canFocus = false
//
// The CodeEditor is the only bundled control shaped that way, and only an
// *editable* one showed it: a read-only buffer's own gate answers no to a Tab
// search regardless. So the editor reads the local itself, and its refusal is
// the conjunction of the two — whichever says no, wins.
func TestComposeCodeEditorHonoursAnInertAncestor(t *testing.T) {
	editor := valuesIn(t, kotlinCodeEditor)
	for _, pin := range []struct{ expr, why string }{
		{"val inert = LocalGrMobInert.current",
			"read in the editor's own composition, which is where the local is in scope"},
		{".focusProperties { canFocus = !inert && (!readOnly || gate.open) }",
			"applied on the field itself, the only chain the walk from the field sees"},
	} {
		if !strings.Contains(editor, pin.expr) {
			t.Errorf("%s: the editor has lost %q — %s; an editable CodeEditor in a "+
				"shut Drawer panel is a Tab stop again", kotlinCodeEditor, pin.expr, pin.why)
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
