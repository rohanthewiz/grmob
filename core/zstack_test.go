package core

import "testing"

// core.ZStack is an ordinary container built from containerNode: it takes the
// same mixed argument list every other container does, so a style prop, a
// behavior prop and a child can be written in any order.
//
// Worth its own test because the node type is the *only* thing that makes an
// overlay an overlay — every renderer keys on it, and none of them can key on
// anything else. A ZStack that arrived as some other type, or that dropped its
// children into a nested box, would render as a vertical stack on all four
// targets and fail nowhere.
func TestZStackIsAContainerOfLayers(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	clicked := false
	n := ZStack(
		Width("160px"),
		Box(),
		OnClick(func() { clicked = true }),
		Text("over"),
		Height("160px"),
	).Render(ctx)

	if n.Type != "ZStack" {
		t.Fatalf("node type = %q, want ZStack — the type is the only thing any renderer keys on", n.Type)
	}
	if n.Style.Width != "160px" || n.Style.Height != "160px" {
		t.Errorf("size = %q x %q, want 160px square; a style prop after a child was dropped",
			n.Style.Width, n.Style.Height)
	}
	if len(n.Children) != 2 {
		t.Fatalf("layer count = %d, want 2", len(n.Children))
	}
	// Tree order is paint order on every target, so the children must arrive
	// in the order they were written and not, say, grouped after the props.
	if n.Children[0].Type != "Box" || n.Children[1].Type != "Text" {
		t.Errorf("layers are %q then %q, want Box then Text", n.Children[0].Type, n.Children[1].Type)
	}
	// A behavior prop lands on the node that is returned, which is the whole
	// reason the containers share containerNode.
	id, ok := n.Props["onClick"].(string)
	if !ok {
		t.Fatal("OnClick did not reach the stack")
	}
	ctx.TriggerCallback(id)
	if !clicked {
		t.Error("the stack's own click handler was registered but does not fire")
	}
}

// No theme base, like Box and Scroll and unlike Column. A theme Column's
// screen inset applied to an overlay would offset every layer by the same
// padding, which changes nothing about their relationship and everything about
// where the stack sits.
func TestZStackCarriesNoThemeBase(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	// Padding is the field to read: it is what the theme's Column base
	// actually carries, so it is the one an accidental base would show up in.
	// Style holds a map and so cannot be compared whole.
	inset := ctx.Theme().Components.Column.Padding
	if inset == (EdgeInsets{}) {
		t.Skip("the theme's Column carries no padding; there is nothing to have inherited")
	}

	n := ZStack().Render(ctx)
	if n.Style == nil {
		t.Fatal("no style at all")
	}
	if n.Style.Padding != (EdgeInsets{}) {
		t.Errorf("a bare ZStack arrived inset by %+v; it must carry no theme base", n.Style.Padding)
	}
	// And the control: a Column in the same context does pick the inset up,
	// so the assertion above is about ZStack and not about an empty theme.
	if c := Column().Render(ctx); c.Style.Padding != inset {
		t.Errorf("the control Column is inset by %+v, want the theme's %+v", c.Style.Padding, inset)
	}
}
