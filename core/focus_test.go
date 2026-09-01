package core

import (
	"strings"
	"testing"
)

// The focus surface: OnFocus/OnBlur are ordinary void behavior props, and the
// leaf builders (the input family) accept behavior props at all, which is
// what makes them reachable on the nodes that actually receive focus.

// leafBuilders is every builder that goes through leafNode, each reduced to
// the same shape so the contract tests below can run over all of them. The
// map is the point: a new input builder that forgets leafNode fails here.
var leafBuilders = map[string]func(...PropsAndChildren) View{
	"Input": func(p ...PropsAndChildren) View {
		return Input("v", "ph", func(string) {}, p...)
	},
	"InputWithSubmit": func(p ...PropsAndChildren) View {
		return InputWithSubmit("v", "ph", func(string) {}, func() {}, p...)
	},
	"InputPassword": func(p ...PropsAndChildren) View {
		return InputPassword("v", "ph", func(string) {}, p...)
	},
	"NumericInput": func(p ...PropsAndChildren) View {
		return NumericInput(1, func(int) {}, p...)
	},
	"TextArea": func(p ...PropsAndChildren) View {
		return TextArea("v", func(string) {}, 3, p...)
	},
	"Checkbox": func(p ...PropsAndChildren) View {
		return Checkbox(true, func(bool) {}, p...)
	},
}

func TestLeafBuildersCarryFocusProps(t *testing.T) {
	for name, build := range leafBuilders {
		t.Run(name, func(t *testing.T) {
			ctx := NewContext()
			ctx.BeginRenderPass()

			var focused, blurred bool
			n := build(
				OnFocus(func() { focused = true }),
				OnBlur(func() { blurred = true }),
			).Render(ctx)

			focusID, ok := n.Props["onFocus"].(string)
			if !ok || focusID == "" {
				t.Fatalf("%s node has no onFocus prop: %#v", name, n.Props)
			}
			blurID, ok := n.Props["onBlur"].(string)
			if !ok || blurID == "" {
				t.Fatalf("%s node has no onBlur prop: %#v", name, n.Props)
			}
			if focusID == blurID {
				t.Fatalf("%s: the two edges share callback ID %q", name, focusID)
			}

			// Both must dispatch on the void channel — the renderers send
			// them through runtime.click, and a text dispatch would find no
			// handler at all.
			ctx.TriggerCallback(focusID)
			if !focused || blurred {
				t.Fatalf("%s onFocus dispatch: focused=%v blurred=%v", name, focused, blurred)
			}
			ctx.TriggerCallback(blurID)
			if !blurred {
				t.Fatalf("%s onBlur dispatch did not run its handler", name)
			}
		})
	}
}

func TestLeafBuildersKeepTheirOwnProps(t *testing.T) {
	// The widening must not have cost the builders their intrinsic props: a
	// behavior prop is added to the map, never in place of it.
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := Input("hello", "type here", func(string) {}, OnBlur(func() {})).Render(ctx)

	if n.Type != "Input" {
		t.Fatalf("node type = %q, want Input", n.Type)
	}
	if n.Props["value"] != "hello" {
		t.Errorf("value = %v, want hello", n.Props["value"])
	}
	if n.Props["placeholder"] != "type here" {
		t.Errorf("placeholder = %v, want type here", n.Props["placeholder"])
	}
	if id, ok := n.Props["onChange"].(string); !ok || !strings.HasPrefix(id, "txt_cb_") {
		t.Errorf("onChange = %v, want a text callback ID", n.Props["onChange"])
	}
	if len(n.Children) != 0 {
		t.Errorf("leaf node has %d children", len(n.Children))
	}
}

func TestLeafStylePropsStillApply(t *testing.T) {
	// Style props keep working, and keep applying in argument order, with a
	// behavior prop interleaved between them.
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := Input("v", "ph", func(string) {},
		Padding(8),
		OnFocus(func() {}),
		Padding(16),
	).Render(ctx)

	if n.Style.Padding.Top != 16 {
		t.Errorf("Padding.Top = %v, want 16 (last style prop wins)", n.Style.Padding.Top)
	}
}

func TestLeafBuilderCallbacksPrecedeBehaviorProps(t *testing.T) {
	// leafNode's ordering contract, and the half of it that matters: the
	// builder's own callbacks are registered first, so interleaving style and
	// behavior props cannot move them. Both spellings below must produce the
	// same IDs.
	ids := func(items ...PropsAndChildren) (string, string) {
		ctx := NewContext()
		ctx.BeginRenderPass()
		n := InputWithSubmit("v", "ph", func(string) {}, func() {}, items...).Render(ctx)
		return n.Props["onSubmit"].(string), n.Props["onBlur"].(string)
	}

	submitA, blurA := ids(Padding(8), OnBlur(func() {}), Margin(4))
	submitB, blurB := ids(OnBlur(func() {}), Padding(8), Margin(4))

	if submitA != "cb_0" {
		t.Errorf("onSubmit = %q, want cb_0 (the builder registers before the props)", submitA)
	}
	if submitA != submitB || blurA != blurB {
		t.Errorf("argument order changed the IDs: (%s,%s) vs (%s,%s)",
			submitA, blurA, submitB, blurB)
	}
}

func TestLeafBehaviorPropsRegisterInArgumentOrder(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	// Input registers one text callback (onChange) and no void ones, so the
	// two behavior props below take cb_0 and cb_1 in the order written.
	n := Input("v", "ph", func(string) {},
		OnBlur(func() {}),
		OnFocus(func() {}),
	).Render(ctx)

	if n.Props["onBlur"] != "cb_0" || n.Props["onFocus"] != "cb_1" {
		t.Fatalf("onBlur=%v onFocus=%v, want cb_0 and cb_1",
			n.Props["onBlur"], n.Props["onFocus"])
	}
}

func TestLeafNodeSkipsNilProp(t *testing.T) {
	// The MaybeProp contract, on a leaf: a dropped prop leaves no trace — and
	// "no trace" includes the concern log. Run in debug mode precisely so a
	// nil falling through to the unknown-item arm is caught; without that the
	// missing prop alone would look identical either way.
	SetDebugMode(true)
	defer SetDebugMode(false)
	ClearConcerns()

	ctx := NewContext()
	ctx.BeginRenderPass()

	n := Input("v", "ph", func(string) {},
		MaybeProp(false, OnBlur(func() {})),
	).Render(ctx)

	if _, ok := n.Props["onBlur"]; ok {
		t.Fatalf("a false MaybeProp still wrote onBlur: %#v", n.Props)
	}
	if id, ok := n.Props["onChange"].(string); !ok || id != "txt_cb_0" {
		t.Errorf("onChange = %v; the nil must not have disturbed registration", n.Props["onChange"])
	}
	if cs := Concerns(); len(cs) != 0 {
		t.Errorf("a dropped prop raised a concern:\n%s", DumpConcerns())
	}
}

func TestLeafNodeReportsAChildView(t *testing.T) {
	// A View is the one item containerNode accepts and leafNode cannot. It
	// must be named rather than silently dropped.
	SetDebugMode(true)
	defer SetDebugMode(false)
	ClearConcerns()

	ctx := NewContext()
	ctx.BeginRenderPass()
	Input("v", "ph", func(string) {}, Text("nowhere to go")).Render(ctx)

	found := false
	for _, c := range Concerns() {
		if c.Kind == ConcernUnknownItem && strings.Contains(c.Detail, "Input") {
			found = true
		}
	}
	if !found {
		t.Fatalf("a View passed to a leaf raised no unknown-item concern: %+v", Concerns())
	}
}

func TestFocusPropsDoNotClobberExistingProps(t *testing.T) {
	// On is a read-modify-write of n.Props. A leaf arrives with its map
	// already populated, so a bug that replaced the map instead of writing
	// into it would drop value/onChange — invisibly, since the node would
	// still render.
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := TextArea("body", func(string) {}, 4, OnFocus(func() {}), OnBlur(func() {})).Render(ctx)

	for _, key := range []string{"value", "rows", "onChange", "onFocus", "onBlur"} {
		if _, ok := n.Props[key]; !ok {
			t.Errorf("prop %q missing after the focus props were applied: %#v", key, n.Props)
		}
	}
}

func TestFocusPropsAreOptionalAndIndependent(t *testing.T) {
	// A node may carry one edge without the other — the reason these are two
	// props rather than one bool-carrying handler.
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := Input("v", "ph", func(string) {}, OnBlur(func() {})).Render(ctx)
	if _, ok := n.Props["onFocus"]; ok {
		t.Errorf("onFocus present without OnFocus: %#v", n.Props)
	}
	if _, ok := n.Props["onBlur"]; !ok {
		t.Errorf("onBlur missing: %#v", n.Props)
	}

	plain := Input("v", "ph", func(string) {}).Render(ctx)
	if _, ok := plain.Props["onFocus"]; ok {
		t.Errorf("onFocus on a field that asked for neither edge: %#v", plain.Props)
	}
	if _, ok := plain.Props["onBlur"]; ok {
		t.Errorf("onBlur on a field that asked for neither edge: %#v", plain.Props)
	}
}

func TestContainersCarryFocusProps(t *testing.T) {
	// The props are uniform BehaviorProps, so a container accepts them too.
	// The native renderers ignore them there (documented on OnFocus), but the
	// HTML export does not, and nothing in Go should special-case node types.
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := Box(OnFocus(func() {}), OnBlur(func() {})).Render(ctx)
	if n.Props["onFocus"] == nil || n.Props["onBlur"] == nil {
		t.Fatalf("container dropped the focus props: %#v", n.Props)
	}
}
