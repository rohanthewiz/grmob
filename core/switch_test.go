package core

import (
	"reflect"
	"testing"
)

// The node a Switch builds, and the two facts about its prop map that the four
// renderers are written against: the state is spelled `checked` (the DOM's
// word, not Go's `on`) and the change rides the bool callback channel.
func TestSwitchRendersCheckedAndTogglesThroughTheBoolChannel(t *testing.T) {
	ctx := NewContext()
	var seen []bool
	n := Switch(true, func(on bool) { seen = append(seen, on) }).Render(ctx)

	if n.Type != "Switch" {
		t.Fatalf("type = %q, want Switch — the renderers dispatch on it, and a Checkbox is "+
			"a different control", n.Type)
	}
	if n.Props["checked"] != true {
		t.Errorf("props = %v, want checked:true. The wire key is the DOM's spelling so both "+
			"web renderers' existing `checked` handling — create AND update-props — works "+
			"untouched; see core.Switch", n.Props)
	}
	if _, has := n.Props["on"]; has {
		t.Errorf("props carry an `on` key as well as `checked`: %v", n.Props)
	}

	id, ok := n.Props["onToggle"].(string)
	if !ok {
		t.Fatalf("onToggle = %v, want a callback ID", n.Props["onToggle"])
	}
	ctx.TriggerBoolCallback(id, false)
	ctx.TriggerBoolCallback(id, true)
	if len(seen) != 2 || seen[0] != false || seen[1] != true {
		t.Errorf("handler saw %v, want [false true]", seen)
	}
}

// A Switch reads Components.CheckBox rather than a field of its own — see
// core.Switch for why a Components.Switch would be a palette entry no palette
// could spend. Pinned because the consequence of it quietly becoming a zero
// Style is a control with no geometry on the two web targets, which looks like
// a theme bug rather than a missing base.
func TestSwitchReadsTheCheckBoxThemeBase(t *testing.T) {
	ctx := NewContext()
	n := Switch(false, nil).Render(ctx)
	if n.Style == nil {
		t.Fatal("Style is nil; a Switch takes the theme's CheckBox base")
	}
	// reflect.DeepEqual rather than ==: Style carries a map field, so the type
	// is not comparable.
	want := ctx.Theme().Components.CheckBox
	if !reflect.DeepEqual(*n.Style, want) {
		t.Errorf("Style = %+v, want the theme's Components.CheckBox %+v", *n.Style, want)
	}
}

// The two boolean controls differ by node type and by nothing else on the wire.
// That is the whole contract the renderers rest on: a Switch is dispatched to a
// different platform control and handed the same prop map, so a renderer that
// grows a Switch arm needs no second reading of the state.
//
// It is also what makes the exchange a *replace* rather than an update —
// reconcile emits one for a changed type — which is the argument core.Switch
// makes for a node type over a flag on Checkbox.
func TestSwitchAndCheckboxDifferOnlyInType(t *testing.T) {
	sw := Switch(true, func(bool) {}).Render(NewContext())
	cb := Checkbox(true, func(bool) {}).Render(NewContext())

	if sw.Type == cb.Type {
		t.Fatalf("both render as %q", sw.Type)
	}
	if len(sw.Props) != len(cb.Props) {
		t.Fatalf("prop maps differ in size: switch %v, checkbox %v", sw.Props, cb.Props)
	}
	for k, v := range cb.Props {
		if sw.Props[k] != v {
			t.Errorf("prop %q: switch has %v, checkbox has %v", k, sw.Props[k], v)
		}
	}
}

// A Switch is not stamped with the focus command, for the reason a Checkbox is
// not: neither native renderer gives one keyboard focus, so the stamp would
// emit a patch per focus command that nothing on the far side reads. The web
// focuses the <input> without being asked.
func TestSwitchIsNotAFocusableLeaf(t *testing.T) {
	if focusableLeafTypes["Switch"] {
		t.Error("Switch is in focusableLeafTypes; neither native renderer focuses a switch")
	}
}
