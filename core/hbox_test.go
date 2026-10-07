package core

import "testing"

// HBox is a Row with no theme base. The node type is the half of that
// contract the renderers read: every target lays a "Row" out as a row, so an
// HBox that emitted anything else would need a new arm in each of them and
// would render as a column wherever one was missing.
func TestHBoxIsARowNode(t *testing.T) {
	ctx := NewContext().WithTheme(DefaultTheme)
	ctx.BeginRenderPass()

	clicked := false
	n := HBox(
		Gap(8),
		Text("a"),
		OnClick(func() { clicked = true }),
		Text("b"),
	).Render(ctx)

	if n.Type != "Row" {
		t.Fatalf("HBox node type = %q, want %q", n.Type, "Row")
	}
	if len(n.Children) != 2 {
		t.Fatalf("HBox has %d children, want 2", len(n.Children))
	}
	if n.Style == nil || n.Style.Gap != 8 {
		t.Errorf("a caller's Gap(8) did not reach the node: %+v", n.Style)
	}
	// The behavior prop goes through the same containerNode path a Row's
	// does, so the handler is registered and fires.
	if id, ok := n.Props["onClick"].(string); !ok {
		t.Errorf("onClick was not registered: props %#v", n.Props)
	} else {
		ctx.TriggerCallback(id)
		if !clicked {
			t.Error("the HBox's click handler was registered but does not fire")
		}
	}
}

// The other half: the theme's Row base must not arrive. Padding is the field
// to read, because it is the only one the bundled themes put on a Row. Style
// holds a map and so cannot be compared whole.
func TestHBoxCarriesNoThemeBase(t *testing.T) {
	ctx := NewContext().WithTheme(DefaultTheme)
	ctx.BeginRenderPass()

	inset := ctx.Theme().Components.Row.Padding
	if inset == (EdgeInsets{}) {
		t.Skip("the theme's Row carries no padding; there is nothing to have inherited")
	}

	n := HBox(Text("child")).Render(ctx)
	if n.Style == nil {
		t.Fatal("no style at all")
	}
	if n.Style.Padding != (EdgeInsets{}) {
		t.Errorf("a bare HBox arrived inset by %+v; it must carry no theme base", n.Style.Padding)
	}
	// The control: a Row in the same context does pick the inset up, so the
	// assertion above is about HBox and not about an empty theme.
	if r := Row().Render(ctx); r.Style.Padding != inset {
		t.Errorf("the control Row is inset by %+v, want the theme's %+v", r.Style.Padding, inset)
	}
	// And a caller's own padding still lands, so the opt-out is only of the
	// base, not of padding.
	if p := HBox(Padding(4)).Render(ctx); p.Style.Padding.Top != 4 || p.Style.Padding.Left != 4 {
		t.Errorf("HBox(Padding(4)) has padding %+v, want 4 on every side", p.Style.Padding)
	}
}
