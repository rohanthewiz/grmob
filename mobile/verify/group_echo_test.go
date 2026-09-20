package verify

import (
	"strings"
	"testing"
)

// A labelled *group* keeps its content readable — and must not read the part
// of it that is already in the group's own announcement.
//
// This is the other half of TestComposeHearsANamedControlByItsNameAlone. A
// named control's label replaces its whole content, so everything under it
// goes quiet (LocalGrMobNamedControl). A group's label does not replace
// anything: a comps.Stepper's number and a comps.Drawer panel's rows are the
// content the name introduces. But Compose emits a merging node's label as a
// fake child and reads a stated value as stateDescription, so content that
// *repeats* either is spoken twice. Heard on a Galaxy Z Fold6 (Samsung
// TalkBack 16.2): "2, Guests, 2" for a Stepper, "Notebook, Notebook" for a
// Drawer panel.
//
//	RenderNode(node)
//	  spokenByLabel(node) non-empty ─► LocalGrMobGroupSaid provides {label, value}
//	                                     └─ GrMobText under it whose content
//	                                        folds to one of them:
//	                                          clearAndSetSemantics { }
//
// Neither widget can drop its Text instead: the web scopes aria-value* to
// progressbar and so hears a Stepper's number only from the Text between the
// buttons, and a Drawer's title is a visible heading. Compose is the one
// target that reads a merged label this way, which is why the repair is here
// and not in comps.
func TestComposeDoesNotRepeatWhatALabelledGroupAlreadySaid(t *testing.T) {
	render := codeOf(t, kotlinRenderer, "fun RenderNode(")
	for _, pin := range []struct{ expr, why string }{
		{"val said = spokenByLabel(node)",
			"what this node's own label and value put into the announcement"},
		{"val says = said.isNotEmpty() && said != LocalGrMobGroupSaid.current",
			"a labelled node replaces the scope; an unlabelled one inherits it"},
		{"if (says) add(LocalGrMobGroupSaid provides said)",
			"provided with the other subtree locals"},
	} {
		if !strings.Contains(render, pin.expr) {
			t.Errorf("%s: RenderNode has lost %q — %s", kotlinRenderer, pin.expr, pin.why)
		}
	}

	// The scope must be *replaced* at each labelled node, never latched on the
	// way the boolean locals are. Compose's own merge stops at a child that
	// merges its descendants, so a nested labelled node's Texts answer to its
	// label alone; a latched union would silence a Text that echoes some
	// outer group it was never merged into.
	if strings.Contains(render, "LocalGrMobGroupSaid provides true") ||
		strings.Contains(render, "LocalGrMobGroupSaid.current + said") {
		t.Errorf("%s: LocalGrMobGroupSaid must be provided as this node's own "+
			"set, not latched or unioned — Compose's merge stops at the "+
			"nearest merging node", kotlinRenderer)
	}

	spoken := valuesOf(t, kotlinRenderer, "internal fun spokenByLabel(")
	for _, pin := range []struct{ expr, why string }{
		{`if (s.accessibilityHidden || s.accessibilityLabel.isEmpty()) return emptySet()`,
			"no label means no merge, so nothing of this node's is read with its children's"},
		{`mutableSetOf(s.accessibilityLabel.trim().lowercase())`,
			"the label, folded — a reader speaks \"Notebook\" and \"notebook \" alike"},
		{`val value = s.accessibilityValue.text`,
			"and the stated value, which becomes stateDescription"},
	} {
		if !strings.Contains(spoken, pin.expr) {
			t.Errorf("%s: spokenByLabel has lost %q — %s", kotlinRenderer, pin.expr, pin.why)
		}
	}

	text := codeOf(t, kotlinRenderer, "private fun GrMobText(")
	for _, pin := range []struct{ expr, why string }{
		{"val echo = node.style?.accessibilityLabel.isNullOrEmpty() &&",
			"a labelled Text opened the scope itself and must not silence itself"},
		{"content.trim().lowercase() in LocalGrMobGroupSaid.current",
			"folded the same way spokenByLabel folds"},
		{".then(if (quiet || echo) Modifier.clearAndSetSemantics { } else Modifier)",
			"after boxModifier, so it clears the text semantics Text adds inside the chain"},
	} {
		if !strings.Contains(text, pin.expr) {
			t.Errorf("%s: GrMobText has lost %q — %s", kotlinRenderer, pin.expr, pin.why)
		}
	}
}
