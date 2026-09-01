package core

import (
	"testing"
)

// Focus traversal: core.UseFocusOrder, the derived IME action, and
// core.FocusNext / core.FocusPrevious. See core/focus_order.go.

// orderFixture renders a three-field form whose refs are declared as one
// traversal order, plus one unnamed field and one named field left out of the
// order — the two shapes that must stay untouched by traversal.
type orderFixture struct {
	ctx *Context
	// The three ordered refs and the deliberately unordered one, re-read from
	// their hook slots on every pass.
	a, b, c, loose *FocusRef
	// declare controls whether the pass calls UseFocusOrder at all, so a test
	// can render the same tree with and without an order.
	declare bool
	// members, when non-nil, replaces the default (a, b, c) membership.
	members func(f *orderFixture) []*FocusRef
	// submitOn names a field built with InputWithSubmit instead of Input.
	submitOn string
}

func newOrderFixture() *orderFixture {
	return &orderFixture{ctx: NewContext(), declare: true}
}

// pass renders one frame and returns the children by index:
// 0 = a, 1 = b, 2 = c, 3 = an unnamed Input, 4 = the loose named Input.
func (f *orderFixture) pass(t *testing.T) []*Node {
	t.Helper()
	f.ctx.BeginRenderPass()
	f.ctx.Reset()

	view := ComponentFunc(func(ctx *Context) *Node {
		f.a = UseFocusRef(ctx)
		f.b = UseFocusRef(ctx)
		f.c = UseFocusRef(ctx)
		f.loose = UseFocusRef(ctx)
		if f.declare {
			members := []*FocusRef{f.a, f.b, f.c}
			if f.members != nil {
				members = f.members(f)
			}
			UseFocusOrder(ctx, members...)
		}
		field := func(name string, ref *FocusRef) View {
			if f.submitOn == name {
				return InputWithSubmit(name, "", func(string) {}, func() {}, FocusTarget(ref))
			}
			return Input(name, "", func(string) {}, FocusTarget(ref))
		}
		return containerNode(ctx, "Column", Style{}, []PropsAndChildren{
			field("a", f.a),
			field("b", f.b),
			field("c", f.c),
			Input("plain", "", func(string) {}),
			Input("loose", "", func(string) {}, FocusTarget(f.loose)),
		})
	})
	return view.Render(f.ctx).Children
}

// ime reads the keyboard action a node advertises. ok is false when the node
// carries no imeAction key at all — a distinct state from an empty action,
// and the one an app that never declares an order must stay in forever.
func ime(n *Node) (action string, ok bool) {
	a, has := n.Props["imeAction"].(string)
	return a, has
}

func TestNoOrderStampsNoImeAction(t *testing.T) {
	// The traversal equivalent of the epoch-0 rule: an app that never calls
	// UseFocusOrder renders exactly the tree it rendered before this feature,
	// FocusTarget and all.
	f := newOrderFixture()
	f.declare = false
	for i, n := range f.pass(t) {
		if action, ok := ime(n); ok {
			t.Errorf("child %d (%s) advertises %q with no order declared",
				i, n.Type, action)
		}
		if _, hasSubmit := n.Props["onSubmit"]; hasSubmit {
			t.Errorf("child %d wired a submit with no order declared", i)
		}
	}
}

func TestOrderAdvertisesNextOnEveryFieldButTheLast(t *testing.T) {
	f := newOrderFixture()
	kids := f.pass(t)

	for i, want := range []string{"next", "next", ""} {
		action, ok := ime(kids[i])
		if !ok {
			t.Fatalf("ordered field %d carries no imeAction: %#v", i, kids[i].Props)
		}
		if action != want {
			t.Errorf("ordered field %d advertises %q, want %q", i, action, want)
		}
	}
	// The last field of an order still carries the key, with an empty value.
	// It has to: an update-props patch carries the whole new props map, so a
	// key that appears only sometimes cannot be seen to change on the far
	// side — and a field can gain a successor when a conditional field below
	// it renders.
	if _, ok := ime(kids[2]); !ok {
		t.Error("the last field of an order dropped its imeAction key entirely")
	}
}

func TestUnnamedFieldStaysOutOfTheOrder(t *testing.T) {
	// A field with no FocusTarget has no identity, so it cannot be a member
	// of anything. It must not advertise an action even while an order exists
	// elsewhere on the screen.
	f := newOrderFixture()
	kids := f.pass(t)
	if action, ok := ime(kids[3]); ok {
		t.Errorf("an unnamed field advertises %q", action)
	}
}

func TestNamedFieldOutsideTheOrderAdvertisesNothing(t *testing.T) {
	// Named but not listed: it carries the key (an order exists in this app)
	// with an empty value, because it has nowhere to advance to.
	f := newOrderFixture()
	kids := f.pass(t)
	action, ok := ime(kids[4])
	if !ok {
		t.Fatalf("a named field carries no imeAction once an order exists: %#v", kids[4].Props)
	}
	if action != "" {
		t.Errorf("a field outside the order advertises %q, want %q", action, "")
	}
	if _, hasSubmit := kids[4].Props["onSubmit"]; hasSubmit {
		t.Error("a field outside the order was wired a submit")
	}
}

func TestNextActionWiresASubmitThatAdvances(t *testing.T) {
	// The whole traversal mechanism in one test: the Next key is an ordinary
	// onSubmit dispatch, and dispatching it issues a focus command naming the
	// following field.
	f := newOrderFixture()
	kids := f.pass(t)

	id, ok := kids[0].Props["onSubmit"].(string)
	if !ok {
		t.Fatalf("the first field of an order wired no submit: %#v", kids[0].Props)
	}
	f.ctx.TriggerCallback(id)

	// The command has been issued; the stamp lands on the next pass.
	next := f.pass(t)
	if _, action, _ := stamp(next[1]); action != "focus" {
		t.Errorf("field b was told %q after a Next on field a, want focus", action)
	}
	if _, action, _ := stamp(next[0]); action != "" {
		t.Errorf("field a was told %q after handing focus on, want the empty action", action)
	}
}

func TestExplicitSubmitWinsOverTheNextAction(t *testing.T) {
	// InputWithSubmit says in so many words what the return key does.
	// Traversal must not silently replace it — nor advertise Next while
	// running the app's own handler, which would be a keyboard that lies.
	f := newOrderFixture()
	f.submitOn = "a"
	kids := f.pass(t)

	if action, _ := ime(kids[0]); action != "" {
		t.Errorf("a field with its own submit advertises %q, want %q", action, "")
	}
	// And the app's handler is still the one wired: the traversal callback
	// would have been registered later in the pass, so a higher ID.
	id := kids[0].Props["onSubmit"].(string)
	if id != "cb_0" {
		t.Errorf("onSubmit is %q, want the builder's own cb_0 — traversal overwrote it", id)
	}
}

func TestFocusNextAndPreviousWalkTheOrder(t *testing.T) {
	f := newOrderFixture()
	f.pass(t)

	FocusNext(f.a)
	if got := f.ctx.focus.target; got != f.b {
		t.Errorf("FocusNext(a) targeted %p, want b (%p)", got, f.b)
	}
	FocusNext(f.b)
	if got := f.ctx.focus.target; got != f.c {
		t.Errorf("FocusNext(b) targeted %p, want c (%p)", got, f.c)
	}
	FocusPrevious(f.c)
	if got := f.ctx.focus.target; got != f.b {
		t.Errorf("FocusPrevious(c) targeted %p, want b (%p)", got, f.b)
	}
}

func TestTraversalStopsAtBothEnds(t *testing.T) {
	// No wrap-around, and no command consumed: the last field of a form
	// submits, it does not send the user back to the top, and an epoch spent
	// on a command with no target would re-stamp every field for nothing.
	f := newOrderFixture()
	f.pass(t)

	before := f.ctx.focus.epoch
	FocusNext(f.c)
	FocusPrevious(f.a)
	if f.ctx.focus.epoch != before {
		t.Errorf("walking off the ends issued %d command(s), want none",
			f.ctx.focus.epoch-before)
	}
}

func TestTraversalIgnoresRefsInNoOrder(t *testing.T) {
	f := newOrderFixture()
	f.pass(t)

	before := f.ctx.focus.epoch
	FocusNext(f.loose)
	FocusPrevious(f.loose)
	if f.ctx.focus.epoch != before {
		t.Errorf("a ref in no order moved focus %d time(s)", f.ctx.focus.epoch-before)
	}
}

func TestTraversalIsNilSafe(t *testing.T) {
	// Matching Focus, DismissKeyboard and FocusTarget: a nil ref is a no-op,
	// so `core.FocusNext(maybeRef)` degrades instead of crashing a handler.
	FocusNext(nil)
	FocusPrevious(nil)
	UseFocusOrder(nil)

	// A nil inside the list is skipped rather than taking a position, so the
	// two fields either side of it become neighbours.
	f := newOrderFixture()
	f.members = func(f *orderFixture) []*FocusRef { return []*FocusRef{f.a, nil, f.c} }
	f.pass(t)
	FocusNext(f.a)
	if got := f.ctx.focus.target; got != f.c {
		t.Errorf("a nil in the order was given a position: FocusNext(a) targeted %p, want c", got)
	}
}

func TestOrderIsRebuiltEveryPass(t *testing.T) {
	// A conditionally rendered field joins and leaves the order as it appears
	// and disappears, which is the reason the order is recomputed rather than
	// held in a hook slot.
	f := newOrderFixture()
	f.pass(t)
	if action, _ := ime(f.pass(t)[1]); action != "next" {
		t.Fatal("field b does not advertise Next with c in the order")
	}

	f.members = func(f *orderFixture) []*FocusRef { return []*FocusRef{f.a, f.b} }
	kids := f.pass(t)
	if action, _ := ime(kids[1]); action != "" {
		t.Errorf("field b still advertises %q after c left the order", action)
	}
	if action, ok := ime(kids[2]); !ok || action != "" {
		t.Errorf("field c advertises (%q, present=%v) after leaving the order, want (\"\", true)", action, ok)
	}
	// And the walk agrees with the stamp.
	before := f.ctx.focus.epoch
	FocusNext(f.b)
	if f.ctx.focus.epoch != before {
		t.Error("FocusNext(b) still advanced after c left the order")
	}
}

func TestOrderDoesNotSpanApps(t *testing.T) {
	// focusState is per-app for the same reason the callback registry is. A
	// ref from another app must not join this one's order, or two managers in
	// one test binary would move each other's cursors.
	other := NewContext()
	other.BeginRenderPass()
	other.Reset()
	foreign := UseFocusRef(other)

	f := newOrderFixture()
	f.members = func(f *orderFixture) []*FocusRef { return []*FocusRef{f.a, foreign, f.c} }
	f.pass(t)

	FocusNext(f.a)
	if got := f.ctx.focus.target; got != f.c {
		t.Errorf("a foreign ref took a position in the order: FocusNext(a) targeted %p, want c", got)
	}
	if foreign.order != nil {
		t.Error("a foreign ref was given order membership")
	}
}

func TestOrderCopiesTheCallersSlice(t *testing.T) {
	// A call site that spreads its own slice must not be able to reorder the
	// form afterwards by writing to it.
	f := newOrderFixture()
	var mine []*FocusRef
	f.members = func(f *orderFixture) []*FocusRef {
		mine = []*FocusRef{f.a, f.b, f.c}
		return mine
	}
	f.pass(t)

	mine[1] = f.loose
	FocusNext(f.a)
	if got := f.ctx.focus.target; got != f.b {
		t.Errorf("mutating the caller's slice changed the order: FocusNext(a) targeted %p, want b", got)
	}
}

func TestOrderMembershipIsIdempotentAcrossPasses(t *testing.T) {
	// UseFocusOrder runs on every pass and must land on the same answer each
	// time — the refs are re-read from their hook slots, so a pass that
	// shifted an index would break traversal after the first keystroke.
	f := newOrderFixture()
	for i := range 3 {
		f.pass(t)
		if f.a.index != 0 || f.b.index != 1 || f.c.index != 2 {
			t.Fatalf("pass %d: indices are %d/%d/%d, want 0/1/2",
				i, f.a.index, f.b.index, f.c.index)
		}
	}
}

func TestRepeatedRefKeepsItsLastPosition(t *testing.T) {
	// A typo that lists a ref twice is not worth a panic. Last wins, which is
	// the only rule that leaves every other field's neighbours intact.
	f := newOrderFixture()
	f.members = func(f *orderFixture) []*FocusRef { return []*FocusRef{f.a, f.b, f.a, f.c} }
	f.pass(t)

	FocusNext(f.a)
	if got := f.ctx.focus.target; got != f.c {
		t.Errorf("FocusNext(a) targeted %p, want c — the later position should win", got)
	}
}

func TestFocusOrderSurvivesDerivedContexts(t *testing.T) {
	// The order lives on focusState, which every derived context shares by
	// pointer. A field rendered inside a child context or a theme override
	// must still traverse.
	root := NewContext()
	root.BeginRenderPass()
	root.Reset()

	child := root.NewChildContext()
	if child.focus != root.focus {
		t.Fatal("a child context got its own focus state")
	}
	a := UseFocusRef(root)
	b := UseFocusRef(root)
	UseFocusOrder(child, a, b)

	FocusNext(a)
	if root.focus.target != b {
		t.Error("an order declared on a child context did not reach the app")
	}
}
