package verify

import (
	"regexp"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// A labelled control is heard by its name alone on Compose, as it is on the
// web (aria-label) and in SwiftUI (a label after .combine).
//
// Compose did not do that by itself: a merging node's label becomes a fake
// child, and every Text under it is spoken after it. On a Galaxy Z Fold6 a
// comps.Rating star read "1 of 5, White star, Button" and a comps.Calendar day
// "…, March 2, 2026, 2, Button". The renderer now opens LocalGrMobNamedControl
// below a named control and GrMobText drops its semantics inside it.
//
//	RenderNode(node)
//	  namesItsContent(node)? ── yes ─► LocalGrMobNamedControl provides true
//	                                     └─ GrMobText under it:
//	                                          clearAndSetSemantics { }
//	                                          (not the named node itself)
func TestComposeHearsANamedControlByItsNameAlone(t *testing.T) {
	render := codeOf(t, kotlinRenderer, "fun RenderNode(")
	for _, pin := range []struct{ expr, why string }{
		{"val named = !LocalGrMobNamedControl.current && namesItsContent(node)",
			"the scope opens once, at the outermost named control"},
		{"if (named) add(LocalGrMobNamedControl provides true)",
			"and is provided with the other subtree locals"},
	} {
		if !strings.Contains(render, pin.expr) {
			t.Errorf("%s: RenderNode has lost %q — %s", kotlinRenderer, pin.expr, pin.why)
		}
	}

	text := codeOf(t, kotlinRenderer, "private fun GrMobText(")
	for _, pin := range []struct{ expr, why string }{
		{"val quiet = LocalGrMobNamedControl.current && !namesItsContent(node)",
			"a Text that is itself the named control keeps its label"},
		{".then(if (quiet) Modifier.clearAndSetSemantics { } else Modifier)",
			"after boxModifier, so it clears the text semantics Text adds inside the chain"},
	} {
		if !strings.Contains(text, pin.expr) {
			t.Errorf("%s: GrMobText has lost %q — %s", kotlinRenderer, pin.expr, pin.why)
		}
	}

	// A label alone must not open the scope: a labelled group (comps.Stepper,
	// comps.RadioGroup) keeps its content readable, as on the web.
	names := valuesOf(t, kotlinRenderer, "internal fun namesItsContent(")
	for _, pin := range []string{
		`if (node.stringProp("onClick").isNotEmpty()) return true`,
		"return s.accessibilityRole in NAME_IS_CONTENT_ROLES",
	} {
		if !strings.Contains(names, pin) {
			t.Errorf("%s: namesItsContent has lost %q", kotlinRenderer, pin)
		}
	}

	// Every role in the set is one core can emit, spelled as core spells it:
	// a misspelt entry would silently never match.
	set := valuesOf(t, kotlinRenderer, "private val NAME_IS_CONTENT_ROLES")
	known := map[string]bool{}
	for _, r := range []core.Role{
		core.RoleButton, core.RoleImg, core.RoleTab, core.RoleRadio, core.RoleOption,
		core.RoleProgressBar, core.RoleLink, core.RoleGridCell,
	} {
		known[string(r)] = true
	}
	found := regexp.MustCompile(`"([a-z]+)"`).FindAllStringSubmatch(set, -1)
	if len(found) == 0 {
		t.Fatalf("%s: NAME_IS_CONTENT_ROLES lists no roles", kotlinRenderer)
	}
	for _, m := range found {
		if !known[m[1]] {
			t.Errorf("%s: NAME_IS_CONTENT_ROLES has %q, which is not a core.Role this "+
				"test knows as a control — add it here if core grew one", kotlinRenderer, m[1])
		}
		if m[1] == string(core.RoleGroup) {
			t.Errorf("%s: NAME_IS_CONTENT_ROLES names group — a labelled group's "+
				"content must stay readable", kotlinRenderer)
		}
	}
}
