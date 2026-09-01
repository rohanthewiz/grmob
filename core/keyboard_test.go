package core

import (
	"reflect"
	"testing"
)

func TestKeyboardAwareMarksTheNode(t *testing.T) {
	// Every container, not only the scrolling ones: the Android renderer
	// applies the inset at the single funnel every node passes through, so a
	// fixed column can lift for the keyboard the same way a scroll region can
	// shorten for it. Screen relies on exactly that when it has no Scroll.
	builders := map[string]func(...PropsAndChildren) View{
		"Scroll": Scroll,
		"List":   List,
		"Column": Column,
		"Box":    Box,
	}
	for name, build := range builders {
		t.Run(name, func(t *testing.T) {
			ctx := NewContext()
			ctx.BeginRenderPass()
			n := build(KeyboardAware(), Text("child")).Render(ctx)

			if n.Props["keyboardAware"] != true {
				t.Errorf("%s props = %#v, want keyboardAware:true", name, n.Props)
			}
			if len(n.Children) != 1 || n.Children[0].Type != "Text" {
				t.Errorf("%s children mangled by the prop: %+v", name, n.Children)
			}
		})
	}
}

func TestKeyboardAwareLeavesOtherPropsAlone(t *testing.T) {
	// The prop writes into whatever props map the node already has rather
	// than installing a fresh one, which is the failure mode of every
	// map-writing BehaviorProp: a handler registered before it must survive.
	ctx := NewContext()
	ctx.BeginRenderPass()

	var tapped bool
	n := Scroll(OnClick(func() { tapped = true }), KeyboardAware(), Text("child")).Render(ctx)

	id, ok := n.Props["onClick"].(string)
	if !ok || id == "" {
		t.Fatalf("onClick was lost: %#v", n.Props)
	}
	ctx.TriggerCallback(id)
	if !tapped || n.Props["keyboardAware"] != true {
		t.Errorf("tapped=%v props=%#v", tapped, n.Props)
	}
}

func TestScrollWithoutTheFlagCarriesNoProp(t *testing.T) {
	// The default has to stay exactly what it was: no prop, so nothing on
	// either native changes for a region that did not ask.
	ctx := NewContext()
	ctx.BeginRenderPass()
	n := Scroll(Text("child")).Render(ctx)

	if _, present := n.Props["keyboardAware"]; present {
		t.Errorf("plain Scroll should carry no keyboardAware prop, got %#v", n.Props)
	}
}

func TestScrollHasNoThemeBase(t *testing.T) {
	// Scroll is unopinionated like Box: a theme base here would put the
	// Column's screen padding on every scrolling region, insetting content
	// that has already been inset by the screen it sits in.
	ctx := NewContext().WithTheme(DefaultTheme)
	ctx.BeginRenderPass()
	n := Scroll(Text("child")).Render(ctx)

	if n.Style == nil || !reflect.DeepEqual(*n.Style, Style{}) {
		t.Errorf("Scroll style = %+v, want the zero Style", n.Style)
	}
}
