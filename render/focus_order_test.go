package render_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// Focus traversal on the wire. The keyboard's Next key is not a new bridge
// call: it is the ordinary onSubmit dispatch, and the proof it works is that
// dispatching the ID a field advertises produces a focus command aimed at the
// field after it. These tests drive the whole path the way a native shell
// does — initial tree, dispatch, patches, JSON.

// orderApp is a three-field form declared as one traversal order. root/0..2
// are the fields; root/3 is a fourth, unnamed field left out of it.
func orderApp(ctx *core.Context) core.View {
	a := core.UseFocusRef(ctx)
	b := core.UseFocusRef(ctx)
	c := core.UseFocusRef(ctx)
	core.UseFocusOrder(ctx, a, b, c)
	return core.Column(
		core.Input("a", "", func(string) {}, core.FocusTarget(a)),
		core.Input("b", "", func(string) {}, core.FocusTarget(b)),
		core.Input("c", "", func(string) {}, core.FocusTarget(c)),
		core.Input("d", "", func(string) {}),
	)
}

// treeProps decodes the root's children's prop maps out of an initial tree.
func treeProps(t *testing.T, tree string) []map[string]any {
	t.Helper()
	var n struct {
		Children []struct{ Props map[string]any }
	}
	if err := json.Unmarshal([]byte(tree), &n); err != nil {
		t.Fatalf("decoding tree: %v", err)
	}
	out := make([]map[string]any, len(n.Children))
	for i, c := range n.Children {
		out[i] = c.Props
	}
	return out
}

func TestImeActionReachesTheWire(t *testing.T) {
	mgr := render.New(core.NewContext(), orderApp)
	defer mgr.Close()

	props := treeProps(t, mgr.RenderInitial())

	for i, want := range []string{"next", "next", ""} {
		if got := props[i]["imeAction"]; got != want {
			t.Errorf("field %d advertises imeAction %v, want %q", i, got, want)
		}
	}
	// A field with no FocusTarget has no identity and so no place in any
	// order; the key must be absent, not empty.
	if _, ok := props[3]["imeAction"]; ok {
		t.Errorf("the unnamed field carries an imeAction: %#v", props[3])
	}
	// The two fields that advertise Next carry the submit that performs it;
	// the last field carries none, so its return key stays the platform's.
	for i := range 2 {
		if _, ok := props[i]["onSubmit"].(string); !ok {
			t.Errorf("field %d advertises Next with no submit wired: %#v", i, props[i])
		}
	}
	if _, ok := props[2]["onSubmit"]; ok {
		t.Errorf("the last field of the order was wired a submit: %#v", props[2])
	}
}

func TestPressingNextFocusesTheFollowingField(t *testing.T) {
	mgr := render.New(core.NewContext(), orderApp)
	defer mgr.Close()

	props := treeProps(t, mgr.RenderInitial())
	// Exactly what a renderer does when the IME's Next key is pressed: it
	// dispatches the onSubmit ID, knowing nothing about what it means.
	out := mgr.DispatchCallback(props[0]["onSubmit"].(string))

	byTarget := map[string]map[string]any{}
	for _, p := range decode(t, out) {
		if p.Type == "update-props" {
			byTarget[p.TargetID] = p.Changes
		}
	}
	if got := byTarget["root/1"]["focusAction"]; got != "focus" {
		t.Errorf("field b was told %v after Next on field a, want focus: %s", got, out)
	}
	if got := byTarget["root/0"]["focusAction"]; got != "" {
		t.Errorf("field a was told %v after handing focus on, want the empty action", got)
	}
}

func TestTraversalWalksTheWholeFormAndStops(t *testing.T) {
	// Two presses reach the last field; the last field has no Next to press,
	// which is the only thing that stops the walk.
	mgr := render.New(core.NewContext(), orderApp)
	defer mgr.Close()

	mgr.DispatchCallback(treeProps(t, mgr.RenderInitial())[0]["onSubmit"].(string))

	// The second press reads the ID out of a freshly rendered tree rather
	// than reusing the first one's, which is the discipline a native shell
	// follows: callback IDs are per-pass sequence numbers, and update-props
	// hands the renderer the current one.
	out := mgr.DispatchCallback(treeProps(t, mgr.RenderInitial())[1]["onSubmit"].(string))
	byTarget := map[string]map[string]any{}
	for _, p := range decode(t, out) {
		if p.Type == "update-props" {
			byTarget[p.TargetID] = p.Changes
		}
	}
	if got := byTarget["root/2"]["focusAction"]; got != "focus" {
		t.Errorf("field c was told %v after Next on field b, want focus: %s", got, out)
	}
}

func TestNoOrderPutsNoImeActionOnTheWire(t *testing.T) {
	// The guarantee for every app written before traversal existed: a form
	// that names its fields but declares no order renders the same JSON it
	// always did.
	app := func(ctx *core.Context) core.View {
		ref := core.UseFocusRef(ctx)
		return core.Column(
			core.Input("a", "", func(string) {}, core.FocusTarget(ref)),
			core.Input("b", "", func(string) {}),
		)
	}
	mgr := render.New(core.NewContext(), app)
	defer mgr.Close()

	if tree := mgr.RenderInitial(); strings.Contains(tree, "imeAction") {
		t.Fatalf("an order-free app put an imeAction on the wire: %s", tree)
	}
}
