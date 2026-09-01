package core

import "testing"

// The gap-5 surface: containers carry gesture behavior props on the node they
// actually return (an earlier Column applied them to a throwaway map), List
// is a first-class virtualized container, and accessibility rides Style.

func TestContainersCarryBehaviorProps(t *testing.T) {
	builders := map[string]func(...PropsAndChildren) View{
		"Row":    Row,
		"Column": Column,
		"Card":   Card,
		"Box":    Box,
		"List":   List,
		// Scroll joined this contract when it stopped taking a bare ...View:
		// the renderers had always read a Scroll node's style and props.
		"Scroll": Scroll,
	}
	for name, build := range builders {
		t.Run(name, func(t *testing.T) {
			ctx := NewContext()
			ctx.BeginRenderPass()

			var clicked, longPressed bool
			n := build(
				OnClick(func() { clicked = true }),
				OnLongPress(func() { longPressed = true }),
				Text("child"),
			).Render(ctx)

			clickID, ok := n.Props["onClick"].(string)
			if !ok || clickID == "" {
				t.Fatalf("%s node has no onClick prop: %#v", name, n.Props)
			}
			pressID, ok := n.Props["onLongPress"].(string)
			if !ok || pressID == "" {
				t.Fatalf("%s node has no onLongPress prop: %#v", name, n.Props)
			}

			// The registered IDs must dispatch the right handlers.
			ctx.TriggerCallback(clickID)
			if !clicked || longPressed {
				t.Fatalf("onClick dispatch: clicked=%v longPressed=%v", clicked, longPressed)
			}
			ctx.TriggerCallback(pressID)
			if !longPressed {
				t.Fatal("onLongPress dispatch did not run its handler")
			}

			if len(n.Children) != 1 || n.Children[0].Type != "Text" {
				t.Fatalf("%s children mangled by behavior props: %+v", name, n.Children)
			}
		})
	}
}

func TestContainerCallbacksRegisterBeforeChildren(t *testing.T) {
	// The ordering contract on containerNode: a container's behavior props
	// take callback IDs before any of its children register theirs, even
	// when the behavior prop is passed after the children in the arguments.
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := Column(
		Button("first child", func() {}),
		OnClick(func() {}),
	).Render(ctx)

	if got := n.Props["onClick"]; got != "cb_0" {
		t.Fatalf("container onClick = %v, want cb_0 (registered before children)", got)
	}
	if got := n.Children[0].Props["onClick"]; got != "cb_1" {
		t.Fatalf("child button onClick = %v, want cb_1", got)
	}
}

func TestListRendersKeyedChildren(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	items := []string{"a", "b", "c"}
	n := List(
		For(items, func(item string, i int) View {
			return Keyed(item, Text(item))
		}),
	).Render(ctx)

	if n.Type != "List" {
		t.Fatalf("node type = %q, want List", n.Type)
	}
	// For wraps its children in a Fragment node; the renderer flattens it.
	if len(n.Children) != 1 || n.Children[0].Type != "Fragment" {
		t.Fatalf("List children = %+v, want one Fragment", n.Children)
	}
	rows := n.Children[0].Children
	if len(rows) != 3 {
		t.Fatalf("row count = %d, want 3", len(rows))
	}
	for i, want := range items {
		if rows[i].Key != want {
			t.Errorf("row %d key = %q, want %q", i, rows[i].Key, want)
		}
	}
}

func TestAccessibilityStyleProps(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := Box(
		AccessibilityLabel("Profile photo"),
		AccessibilityHint("Opens the profile"),
	).Render(ctx)
	if n.Style.AccessibilityLabel != "Profile photo" {
		t.Errorf("AccessibilityLabel = %q", n.Style.AccessibilityLabel)
	}
	if n.Style.AccessibilityHint != "Opens the profile" {
		t.Errorf("AccessibilityHint = %q", n.Style.AccessibilityHint)
	}
	if n.Style.AccessibilityHidden {
		t.Error("AccessibilityHidden set without the prop")
	}

	deco := Text("···", AccessibilityHidden()).Render(ctx)
	if !deco.Style.AccessibilityHidden {
		t.Error("AccessibilityHidden() did not set the style field")
	}
}
