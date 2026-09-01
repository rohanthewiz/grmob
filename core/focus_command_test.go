package core

import (
	"testing"
)

// The imperative half of focus: core.Focus and core.DismissKeyboard, which
// reach the renderers as the (focusEpoch, focusAction) prop pair rather than
// through any bridge call. See core/focus.go for why.

// focusFixture renders three text fields, the first of them named by a ref,
// plus a Checkbox and a Button — the two leafNode types the platform does not
// give keyboard focus to. pass() runs one render and hands back the five
// nodes, so a test can drive several commands across several passes the way
// the render manager does.
type focusFixture struct {
	ctx *Context
	ref *FocusRef
}

func newFocusFixture(t *testing.T) *focusFixture {
	t.Helper()
	return &focusFixture{ctx: NewContext()}
}

// pass renders one frame and returns the rendered children by index:
// 0 = the named Input, 1 = a second Input, 2 = a TextArea, 3 = Checkbox,
// 4 = Button.
func (f *focusFixture) pass(t *testing.T) []*Node {
	t.Helper()
	f.ctx.BeginRenderPass()
	f.ctx.Reset()

	view := ComponentFunc(func(ctx *Context) *Node {
		// The ref is re-read every pass from the same hook slot, which is the
		// property UseFocusRef exists to provide: identity has to survive a
		// render or FocusTarget and Focus would be comparing two different
		// pointers.
		f.ref = UseFocusRef(ctx)
		return containerNode(ctx, "Column", Style{}, []PropsAndChildren{
			Input("a", "", func(string) {}, FocusTarget(f.ref)),
			Input("b", "", func(string) {}),
			TextArea("c", func(string) {}, 3),
			Checkbox(false, func(bool) {}),
			Button("go", func() {}),
		})
	})
	return view.Render(f.ctx).Children
}

// stamp reads the focus command a node is carrying. ok is false when the node
// carries no stamp at all, which is a distinct state from an empty action.
func stamp(n *Node) (epoch int, action string, ok bool) {
	e, hasEpoch := n.Props["focusEpoch"].(int)
	a, hasAction := n.Props["focusAction"].(string)
	if !hasEpoch && !hasAction {
		return 0, "", false
	}
	// Never one without the other: an update-props patch carries the whole
	// new props map and the renderers iterate the keys it contains, so a key
	// that vanishes between passes is invisible on the far side. A half stamp
	// is a bug even though every individual assertion below would still pass.
	if !hasEpoch || !hasAction {
		panic("half a focus stamp: " + n.Type)
	}
	return e, a, true
}

func TestNoCommandStampsNothing(t *testing.T) {
	// An app that never touches focus must render exactly the tree it always
	// did — no new props, no new patches. This is what the epoch == 0 early
	// return in stampFocus buys, and it is why the whole feature costs
	// existing apps nothing.
	f := newFocusFixture(t)
	for i, n := range f.pass(t) {
		if _, _, ok := stamp(n); ok {
			t.Errorf("child %d (%s) carries a stamp before any command: %#v",
				i, n.Type, n.Props)
		}
	}
}

func TestFocusStampsTargetAndSilencesTheRest(t *testing.T) {
	f := newFocusFixture(t)
	f.pass(t) // establish the hook slot so the ref exists
	Focus(f.ref)
	kids := f.pass(t)

	epoch, action, ok := stamp(kids[0])
	if !ok || epoch != 1 || action != "focus" {
		t.Fatalf("target stamp = (%d, %q, %v), want (1, focus, true)", epoch, action, ok)
	}

	// The other two text fields carry the same epoch with nothing to do.
	// "" rather than a missing key, and rather than "blur": telling them to
	// blur would race the target's focus request, since neither Compose nor
	// SwiftUI orders two effects against each other.
	for _, i := range []int{1, 2} {
		epoch, action, ok := stamp(kids[i])
		if !ok || epoch != 1 || action != "" {
			t.Errorf("child %d (%s) stamp = (%d, %q, %v), want (1, \"\", true)",
				i, kids[i].Type, epoch, action, ok)
		}
	}
}

func TestUnfocusableLeavesAreNeverStamped(t *testing.T) {
	// Checkbox and Button go through leafNode but are absent from
	// focusableLeafTypes: neither platform gives them keyboard focus, so a
	// stamp would be a patch per command that nothing on the far side reads.
	f := newFocusFixture(t)
	f.pass(t)
	Focus(f.ref)
	kids := f.pass(t)

	for _, i := range []int{3, 4} {
		if _, _, ok := stamp(kids[i]); ok {
			t.Errorf("%s carries a focus stamp: %#v", kids[i].Type, kids[i].Props)
		}
	}
}

func TestDismissReachesEveryField(t *testing.T) {
	// A dismiss names no node, and the field the *user* tapped into is one Go
	// was never told about — nothing wires OnFocus unless an app asks. So
	// every focusable leaf has to be told to blur, and each renderer acts
	// only if it actually holds focus.
	f := newFocusFixture(t)
	f.pass(t)
	DismissKeyboard(f.ctx)
	kids := f.pass(t)

	for _, i := range []int{0, 1, 2} {
		epoch, action, ok := stamp(kids[i])
		if !ok || epoch != 1 || action != "blur" {
			t.Errorf("child %d (%s) stamp = (%d, %q, %v), want (1, blur, true)",
				i, kids[i].Type, epoch, action, ok)
		}
	}
	// Including the field a FocusTarget names: being named does not exempt it
	// from a dismiss, and it may well be the one holding focus.
	if _, action, _ := stamp(kids[0]); action != "blur" {
		t.Errorf("the named field escaped the dismiss: action = %q", action)
	}
}

func TestDismissAfterFocusReleasesTheNamedField(t *testing.T) {
	// A dismiss has to clear the standing target, not merely advance the
	// epoch. Leaving the target in place would have the field the app last
	// focused read its own dismiss as "focus" and grab the keyboard straight
	// back — the exact opposite of what was asked for, and invisible to any
	// test that dismisses without having focused first.
	f := newFocusFixture(t)
	f.pass(t)

	Focus(f.ref)
	if _, action, _ := stamp(f.pass(t)[0]); action != "focus" {
		t.Fatalf("setup: target action = %q, want focus", action)
	}

	DismissKeyboard(f.ctx)
	epoch, action, ok := stamp(f.pass(t)[0])
	if !ok || epoch != 2 || action != "blur" {
		t.Fatalf("after dismiss the named field = (%d, %q, %v), want (2, blur, true)",
			epoch, action, ok)
	}
}

func TestRepeatedFocusRefires(t *testing.T) {
	// The reason the stamp carries a counter and not a bool. Two identical
	// prop maps produce no patch and therefore no effect, so focusing the
	// already-focused field — the "try again after a failed submit" case —
	// would silently do nothing if the stamp did not move.
	f := newFocusFixture(t)
	f.pass(t)

	Focus(f.ref)
	first, _, _ := stamp(f.pass(t)[0])
	Focus(f.ref)
	second, action, _ := stamp(f.pass(t)[0])

	if second <= first {
		t.Fatalf("epoch did not advance on the second Focus: %d then %d", first, second)
	}
	if action != "focus" {
		t.Errorf("action = %q after the second Focus, want focus", action)
	}
}

func TestStampIsStableBetweenCommands(t *testing.T) {
	// Nothing consumes the epoch. A re-render for an unrelated reason must
	// leave the stamp exactly as it was, or the reconciler would emit a patch
	// and the renderers would re-run a command the app never issued a second
	// time.
	f := newFocusFixture(t)
	f.pass(t)
	Focus(f.ref)

	e1, a1, _ := stamp(f.pass(t)[0])
	e2, a2, _ := stamp(f.pass(t)[0])
	if e1 != e2 || a1 != a2 {
		t.Fatalf("stamp moved without a command: (%d,%q) then (%d,%q)", e1, a1, e2, a2)
	}
}

func TestCommandsInterleave(t *testing.T) {
	// Focus, dismiss, focus again. The middle command must leave every field
	// on "blur" and the third must put exactly one back on "focus" — the
	// others returning to "" rather than staying on the stale "blur", which
	// would have them fighting the target for focus.
	f := newFocusFixture(t)
	f.pass(t)

	Focus(f.ref)
	DismissKeyboard(f.ctx)
	Focus(f.ref)
	kids := f.pass(t)

	if e, a, _ := stamp(kids[0]); e != 3 || a != "focus" {
		t.Errorf("target stamp = (%d, %q), want (3, focus)", e, a)
	}
	if e, a, _ := stamp(kids[1]); e != 3 || a != "" {
		t.Errorf("sibling stamp = (%d, %q), want (3, \"\") — a stale blur would "+
			"fight the target for focus", e, a)
	}
}

func TestUseFocusRefIsStableAcrossPasses(t *testing.T) {
	// Identity is the whole content of a FocusRef, so a ref that changed
	// pointer between passes would have FocusTarget stamping one identity and
	// Focus comparing against another — the field would simply never focus.
	f := newFocusFixture(t)
	f.pass(t)
	first := f.ref
	f.pass(t)
	if f.ref != first {
		t.Fatalf("UseFocusRef returned a different pointer on the second pass")
	}
}

func TestFocusRequestsARender(t *testing.T) {
	// Both commands go through RequestRender, not a bare MarkDirty: a command
	// issued from a timer or a network callback has no bridge call pending to
	// carry its patches back, so it needs the push channel nudged.
	f := newFocusFixture(t)
	f.pass(t)
	f.ctx.ClearDirty()

	Focus(f.ref)
	if !f.ctx.IsDirty() {
		t.Error("Focus did not mark the tree dirty")
	}

	f.ctx.ClearDirty()
	DismissKeyboard(f.ctx)
	if !f.ctx.IsDirty() {
		t.Error("DismissKeyboard did not mark the tree dirty")
	}
}

func TestNilCommandsAreNoOps(t *testing.T) {
	// Degrading rather than panicking, matching MaybeProp's contract: a
	// conditional ref that turns out to be absent should cost a field its
	// name, not crash a render pass.
	f := newFocusFixture(t)
	f.pass(t)

	Focus(nil)
	DismissKeyboard(nil)

	for i, n := range f.pass(t) {
		if _, _, ok := stamp(n); ok {
			t.Errorf("child %d (%s) was stamped by a nil command: %#v", i, n.Type, n.Props)
		}
	}

	if p := FocusTarget(nil); p != nil {
		t.Error("FocusTarget(nil) must return a nil prop so leafNode skips it")
	}
}

func TestFocusTargetStampsANodeLeafNodeWouldNot(t *testing.T) {
	// FocusTarget writes the complete pair itself rather than relying on
	// leafNode, so applying it to a node type outside focusableLeafTypes
	// still produces a self-consistent stamp instead of half of one.
	ctx := NewContext()
	ctx.BeginRenderPass()
	ref := &FocusRef{ctx: ctx}
	Focus(ref)

	ctx.BeginRenderPass()
	n := Button("go", func() {}, FocusTarget(ref)).Render(ctx)

	epoch, action, ok := stamp(n)
	if !ok || epoch != 1 || action != "focus" {
		t.Fatalf("stamp = (%d, %q, %v), want (1, focus, true)", epoch, action, ok)
	}
}

func TestFocusOnAnUnmountedRefIsHarmless(t *testing.T) {
	// A command for a ref whose node is not in the tree still consumes an
	// epoch — it happened, it simply had no target on screen — and every
	// field on screen must read that as "" rather than as an instruction.
	f := newFocusFixture(t)
	f.pass(t)

	orphan := &FocusRef{ctx: f.ctx}
	Focus(orphan)
	kids := f.pass(t)

	for _, i := range []int{0, 1, 2} {
		if e, a, ok := stamp(kids[i]); !ok || e != 1 || a != "" {
			t.Errorf("child %d stamp = (%d, %q, %v), want (1, \"\", true)", i, e, a, ok)
		}
	}
}

func TestTwoAppsDoNotShareOneKeyboard(t *testing.T) {
	// focusState is per-app instance state carried on Context, shared by
	// pointer with derived contexts exactly like the callback registry — not
	// a package-level global. A global would have two apps in one process
	// (and two managers in one test binary) fighting over one keyboard.
	a := newFocusFixture(t)
	b := newFocusFixture(t)
	a.pass(t)
	b.pass(t)

	Focus(a.ref)

	if _, _, ok := stamp(b.pass(t)[0]); ok {
		t.Fatal("a Focus in one app stamped a field in another")
	}
	if _, _, ok := stamp(a.pass(t)[0]); !ok {
		t.Fatal("the Focus did not reach its own app")
	}
}

func TestDerivedContextsShareFocusState(t *testing.T) {
	// The other half of the same rule: a child context, a scope and a themed
	// copy are all the same app, so a command issued through any of them must
	// be visible through the others.
	ctx := NewContext()
	for name, derived := range map[string]*Context{
		"child": ctx.NewChildContext(),
		"scope": ctx.Scope("s"),
		"theme": ctx.WithTheme(DefaultTheme),
	} {
		if derived.focus != ctx.focus {
			t.Errorf("%s context does not share the app's focus state", name)
		}
	}
}

// --- core.Button's widening -------------------------------------------------

func TestButtonKeepsItsOwnProps(t *testing.T) {
	// Going through leafNode must not have cost Button its intrinsic props,
	// and in particular must not have cost it the unconditional onClick
	// registration components.Button relies on.
	ctx := NewContext()
	ctx.BeginRenderPass()

	clicked := false
	n := Button("Save", func() { clicked = true }, Padding(8)).Render(ctx)

	if n.Type != "Button" {
		t.Fatalf("node type = %q, want Button", n.Type)
	}
	if n.Props["label"] != "Save" {
		t.Errorf("label = %v, want Save", n.Props["label"])
	}
	if n.Style.Padding.Top != 8 {
		t.Errorf("Padding.Top = %v, want 8 — style props must still apply", n.Style.Padding.Top)
	}
	id, ok := n.Props["onClick"].(string)
	if !ok || id == "" {
		t.Fatalf("onClick = %v, want a callback ID", n.Props["onClick"])
	}
	ctx.TriggerCallback(id)
	if !clicked {
		t.Error("the registered onClick did not run")
	}
	if len(n.Children) != 0 {
		t.Errorf("Button rendered %d children", len(n.Children))
	}
}

func TestButtonRegistersANilHandler(t *testing.T) {
	// components.Button substitutes a no-op for a disabled button's handler
	// precisely because core.Button registers whatever it is handed. If this
	// started dropping nil the "onClick" prop would disappear, changing what
	// a disabled button diffs to and leaving a live tap with nowhere to go.
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := Button("Save", nil).Render(ctx)
	if _, ok := n.Props["onClick"].(string); !ok {
		t.Fatalf("a nil handler dropped the onClick prop: %#v", n.Props)
	}
}

func TestButtonTakesBehaviorProps(t *testing.T) {
	// The whole point of the widening: a gesture prop on the node that exists
	// to be pressed. The builder's own onClick still takes the lower ID, so
	// interleaving arguments cannot move it.
	ctx := NewContext()
	ctx.BeginRenderPass()

	longPressed := false
	n := Button("Delete", func() {},
		Padding(4),
		OnLongPress(func() { longPressed = true }),
	).Render(ctx)

	if n.Props["onClick"] != "cb_0" {
		t.Errorf("onClick = %v, want cb_0 (the builder registers first)", n.Props["onClick"])
	}
	id, ok := n.Props["onLongPress"].(string)
	if !ok {
		t.Fatalf("no onLongPress prop: %#v", n.Props)
	}
	ctx.TriggerCallback(id)
	if !longPressed {
		t.Error("the long-press handler did not run")
	}
}

func TestButtonWithEventKeepsItsShape(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := ButtonWithEvent("Hold", "LongPress", func() {}, FontSize(11)).Render(ctx)
	if n.Props["label"] != "Hold" {
		t.Errorf("label = %v", n.Props["label"])
	}
	if _, ok := n.Props["onLongPress"].(string); !ok {
		t.Fatalf("no onLongPress prop: %#v", n.Props)
	}
	// The one thing that distinguishes it from Button: no click is wired.
	if _, ok := n.Props["onClick"]; ok {
		t.Error("ButtonWithEvent registered an onClick it was never given")
	}
	if n.Style.FontSize != 11 {
		t.Errorf("FontSize = %v, want 11", n.Style.FontSize)
	}
}
