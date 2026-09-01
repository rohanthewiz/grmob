package render_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/render"
)

// core.Focus and core.DismissKeyboard have no bridge call to travel on: they
// reach the renderers as props, which means the only proof they work is that
// a command actually turns into patches on the wire. These tests drive the
// whole path — command, render pass, diff, JSON — the way a native shell sees
// it.

type patch struct {
	Type     string
	TargetID string
	Changes  map[string]any
}

func decode(t *testing.T, s string) []patch {
	t.Helper()
	var ps []patch
	if err := json.Unmarshal([]byte(s), &ps); err != nil {
		t.Fatalf("decoding patches %q: %v", s, err)
	}
	return ps
}

// focusApp is two text fields and a button: root/0 is the field a FocusTarget
// names, root/1 an unnamed one, root/2 the button.
//
// The button's handler issues whatever command the test installed, so a
// command travels the real event path — dispatch, handler, render, diff —
// rather than being poked in from the side. ref is written on every pass so a
// test can name the field it wants focused; the same pointer comes back each
// time, which is the property UseFocusRef exists to provide.
func focusApp(command *func(*core.Context), ref **core.FocusRef) func(*core.Context) core.View {
	return func(ctx *core.Context) core.View {
		*ref = core.UseFocusRef(ctx)
		return core.Column(
			core.Input("a", "", func(string) {}, core.FocusTarget(*ref)),
			core.Input("b", "", func(string) {}),
			core.Button("act", func() {
				if *command != nil {
					(*command)(ctx)
				}
			}),
		)
	}
}

func TestFocusCommandReachesTheWire(t *testing.T) {
	var command func(*core.Context)
	var ref *core.FocusRef
	mgr := render.New(core.NewContext(), focusApp(&command, &ref))
	defer mgr.Close()

	initial := mgr.RenderInitial()
	if strings.Contains(initial, "focusEpoch") {
		t.Fatal("the initial tree carries a focus stamp before any command")
	}

	// The button's handler issues the command; DispatchCallback runs it and
	// renders in the same lock hold, so the patches come straight back.
	command = func(ctx *core.Context) { core.Focus(ref) }
	out := mgr.DispatchCallback(clickIDOf(t, initial))

	byTarget := map[string]map[string]any{}
	for _, p := range decode(t, out) {
		if p.Type == "update-props" {
			byTarget[p.TargetID] = p.Changes
		}
	}

	// Both inputs are patched — the target with the instruction, its sibling
	// with the same epoch and nothing to do.
	target, ok := byTarget["root/0"]
	if !ok {
		t.Fatalf("no update-props for the target field; got %s", out)
	}
	if target["focusAction"] != "focus" {
		t.Errorf("target focusAction = %v, want focus", target["focusAction"])
	}
	sibling, ok := byTarget["root/1"]
	if !ok {
		t.Fatalf("no update-props for the sibling field; got %s", out)
	}
	if sibling["focusAction"] != "" {
		t.Errorf("sibling focusAction = %v, want empty", sibling["focusAction"])
	}
	// The button is not focusable and must not have been touched.
	if _, ok := byTarget["root/2"]; ok {
		t.Errorf("the Button was patched by a focus command: %s", out)
	}
}

func TestDismissReachesTheWire(t *testing.T) {
	var command func(*core.Context)
	var ref *core.FocusRef
	mgr := render.New(core.NewContext(), focusApp(&command, &ref))
	defer mgr.Close()

	initial := mgr.RenderInitial()
	command = func(ctx *core.Context) { core.DismissKeyboard(ctx) }
	out := mgr.DispatchCallback(clickIDOf(t, initial))

	blurred := 0
	for _, p := range decode(t, out) {
		if p.Type == "update-props" && p.Changes["focusAction"] == "blur" {
			blurred++
		}
	}
	// Both fields, because the one the user tapped into is the one Go was
	// never told about.
	if blurred != 2 {
		t.Fatalf("%d fields told to blur, want 2: %s", blurred, out)
	}
}

func TestAnUnrelatedRenderEmitsNoFocusPatch(t *testing.T) {
	// The stamp must be inert between commands. If a re-render for any other
	// reason re-emitted it, every renderer would re-run the last command —
	// stealing focus back on every keystroke elsewhere on the screen.
	var command func(*core.Context)
	var ref *core.FocusRef
	mgr := render.New(core.NewContext(), focusApp(&command, &ref))
	defer mgr.Close()

	initial := mgr.RenderInitial()
	id := clickIDOf(t, initial)

	command = func(ctx *core.Context) { core.Focus(ref) }
	mgr.DispatchCallback(id)

	// A second tap that issues no command at all.
	command = nil
	out := mgr.DispatchCallback(id)

	if strings.Contains(out, "focusEpoch") || strings.Contains(out, "focusAction") {
		t.Fatalf("a command-free render re-emitted the focus stamp: %s", out)
	}
}

// clickIDOf digs the button's callback ID out of the initial tree JSON. The
// button is the third child of the root column.
func clickIDOf(t *testing.T, tree string) string {
	t.Helper()
	var n struct {
		Children []struct {
			Props map[string]any
		}
	}
	if err := json.Unmarshal([]byte(tree), &n); err != nil {
		t.Fatalf("decoding tree: %v", err)
	}
	id, ok := n.Children[2].Props["onClick"].(string)
	if !ok {
		t.Fatalf("no onClick on the button: %#v", n.Children[2].Props)
	}
	return id
}
