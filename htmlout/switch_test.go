package htmlout

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// A Switch exports as the element HTML actually has for one: an
// <input type="checkbox"> carrying the `switch` attribute and role="switch".
//
// All three parts are load-bearing and each covers a different browser:
//
//	type="checkbox"   what makes the <input> a control and not a text box
//	switch            WHATWG's own attribute. Safari draws a track and a thumb
//	                  from it; engines that do not support it draw the box,
//	                  which is the same bool in the same state
//	role="switch"     what a screen reader announces, in every browser,
//	                  whether or not the control is drawn as a switch
func TestSwitchExportsTheSwitchElement(t *testing.T) {
	n := &core.Node{Type: "Switch", Props: map[string]any{"checked": true}}
	out := ExportHTML(n)
	for _, want := range []string{`type="checkbox"`, `switch="switch"`, `role="switch"`, `checked="checked"`} {
		if !strings.Contains(out, want) {
			t.Errorf("switch export lacks %s:\n%s", want, out)
		}
	}
}

// The off state drops `checked` and keeps everything that says what the control
// is. The attribute is the *default* state in HTML rather than the live one, so
// writing checked="false" would be a checked box — which is why the exporter
// omits it instead.
func TestSwitchOffOmitsOnlyTheState(t *testing.T) {
	n := &core.Node{Type: "Switch", Props: map[string]any{"checked": false}}
	out := ExportHTML(n)
	if strings.Contains(out, "checked") {
		t.Errorf("an off switch exported a checked attribute:\n%s", out)
	}
	for _, want := range []string{`switch="switch"`, `role="switch"`} {
		if !strings.Contains(out, want) {
			t.Errorf("off switch lacks %s:\n%s", want, out)
		}
	}
}

// The role comes from the node type, so a Switch with no Style at all still
// announces itself. That is the case a Style-only role would miss, and it is
// not hypothetical: a hand-assembled node carries no Style, and nothing in the
// export path would have supplied one.
func TestAStylelessSwitchStillAnnouncesItself(t *testing.T) {
	if out := ExportHTML(&core.Node{Type: "Switch"}); !strings.Contains(out, `role="switch"`) {
		t.Errorf("a styleless Switch exported no role:\n%s", out)
	}
}

// An author's own role wins, which is Modal's rule rather than a new one: a
// caller who wrote a Style saying what this node is means it. The guard that
// matters is the *other* direction — exactly one role attribute reaches the
// element, because a browser settles a duplicated attribute by source order.
func TestAnAuthoredRoleReplacesTheSwitchRole(t *testing.T) {
	n := &core.Node{
		Type:  "Switch",
		Props: map[string]any{"checked": false},
		Style: &core.Style{AccessibilityRole: core.RoleButton},
	}
	out := ExportHTML(n)
	if strings.Contains(out, `role="switch"`) {
		t.Errorf("the author's role did not win:\n%s", out)
	}
	if got := strings.Count(out, "role="); got != 1 {
		t.Errorf("%d role attributes on one element, want 1:\n%s", got, out)
	}
}

// Hidden wins over the type's role, as it does over a Modal's: an element
// pruned from the accessibility tree has no role left to describe.
func TestAHiddenSwitchExportsNoRole(t *testing.T) {
	n := &core.Node{Type: "Switch", Style: &core.Style{AccessibilityHidden: true}}
	out := ExportHTML(n)
	if strings.Contains(out, "role=") {
		t.Errorf("aria-hidden should win alone:\n%s", out)
	}
}

// The two boolean controls are told apart by one attribute and not by the type
// attribute, which they share. Asserted because the shared value is what makes
// the <input> table stop one step short of deciding the control — see
// inputTypes — and a Checkbox that started exporting `switch` would be drawn as
// a switch in Safari with nothing else changed.
func TestOnlyTheSwitchCarriesTheSwitchAttribute(t *testing.T) {
	if got := InputTypeFor("Switch"); got != InputTypeFor("Checkbox") {
		t.Errorf("InputTypeFor(Switch) = %q, InputTypeFor(Checkbox) = %q — HTML has one "+
			"element for both", got, InputTypeFor("Checkbox"))
	}
	cb := ExportHTML(&core.Node{Type: "Checkbox", Props: map[string]any{"checked": true}})
	if strings.Contains(cb, "switch") {
		t.Errorf("a Checkbox exported a switch attribute or role:\n%s", cb)
	}
}

// A Switch accepts the disabled attribute, like every other control that
// exports as a real form element. A switch an app has turned off is the
// commonest thing a settings screen does.
func TestADisabledSwitchGetsTheAttribute(t *testing.T) {
	n := &core.Node{Type: "Switch", Style: &core.Style{Disabled: true}}
	if out := ExportHTML(n); !strings.Contains(out, "disabled") {
		t.Errorf("disabled switch lacks the attribute:\n%s", out)
	}
}
