package core

import (
	"strings"
	"testing"
)

// MaybeProp's whole value is what it does *not* leave behind, so the tests
// are mostly assertions about absence: no child, no slot, no style.

// The headline difference from core.If, and the reason MaybeProp exists at
// all. Both express "this child only when the flag holds", but If(false, ...)
// returns an empty Fragment, which renders as a real node and takes a slot in
// the container's flex layout.
func TestMaybePropLeavesNoNodeWhereIfLeavesAnEmptyFragment(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	withIf := Column(
		Text("kept"),
		If(false, Text("dropped")),
	).Render(ctx)
	if len(withIf.Children) != 2 {
		t.Fatalf("baseline changed: If(false) no longer emits a child (%d children) — "+
			"if that is now true, MaybeProp's rationale needs revisiting",
			len(withIf.Children))
	}

	withMaybe := Column(
		Text("kept"),
		MaybeProp(false, Text("dropped")),
	).Render(ctx)
	if len(withMaybe.Children) != 1 {
		t.Fatalf("MaybeProp(false) left %d children, want 1: %+v",
			len(withMaybe.Children), withMaybe.Children)
	}
	if got := withMaybe.Children[0].Props["content"]; got != "kept" {
		t.Fatalf("surviving child is %v, want the unconditional one", got)
	}
}

// The true path has to be a plain pass-through: the item lands in the same
// position it would have occupied if written inline, since sibling order is
// what the renderers lay out and what keyed reconciliation diffs.
func TestMaybePropTrueKeepsTheItemInPlace(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()

	n := Column(
		Text("first"),
		MaybeProp(true, Text("middle")),
		Text("last"),
	).Render(ctx)

	var got []string
	for _, c := range n.Children {
		got = append(got, c.Props["content"].(string))
	}
	if strings.Join(got, ",") != "first,middle,last" {
		t.Fatalf("child order is %v, want [first middle last]", got)
	}
}

// The second half of the rationale: If is View -> View, so a *style* prop had
// no conditional expression form before this. A false MaybeProp must leave
// the style untouched rather than applying a zero value over it.
func TestMaybePropCoversStyleAndBehaviorProps(t *testing.T) {
	t.Run("style applied when true", func(t *testing.T) {
		ctx := NewContext()
		ctx.BeginRenderPass()
		n := Box(MaybeProp(true, BackgroundColor("#123456"))).Render(ctx)
		if n.Style.Background != "#123456" {
			t.Fatalf("background is %q, want it applied", n.Style.Background)
		}
	})

	t.Run("style untouched when false", func(t *testing.T) {
		ctx := NewContext()
		ctx.BeginRenderPass()
		n := Box(
			BackgroundColor("#ABCDEF"),
			MaybeProp(false, BackgroundColor("#123456")),
		).Render(ctx)
		if n.Style.Background != "#ABCDEF" {
			t.Fatalf("background is %q — the false branch overwrote the earlier prop",
				n.Style.Background)
		}
	})

	t.Run("behavior prop attached only when true", func(t *testing.T) {
		ctx := NewContext()
		ctx.BeginRenderPass()

		on := Box(MaybeProp(true, OnClick(func() {}))).Render(ctx)
		if id, ok := on.Props["onClick"].(string); !ok || id == "" {
			t.Fatalf("onClick missing from %#v", on.Props)
		}

		off := Box(MaybeProp(false, OnClick(func() {}))).Render(ctx)
		if _, ok := off.Props["onClick"]; ok {
			t.Fatalf("onClick present on the false branch: %#v", off.Props)
		}
	})
}

// Every container builder shares containerNode, so all five honor the nil
// contract or none do. Asserted across the set because a future container
// that hand-rolls its own item loop would pass a Column-only test.
func TestMaybePropWorksInEveryContainer(t *testing.T) {
	builders := map[string]func(...PropsAndChildren) View{
		"Row":    Row,
		"Column": Column,
		"Card":   Card,
		"Box":    Box,
		"List":   List,
	}
	for name, build := range builders {
		t.Run(name, func(t *testing.T) {
			ctx := NewContext()
			ctx.BeginRenderPass()
			n := build(
				MaybeProp(false, Text("dropped")),
				MaybeProp(true, Text("kept")),
			).Render(ctx)
			if len(n.Children) != 1 {
				t.Fatalf("%s has %d children, want 1", name, len(n.Children))
			}
			if got := n.Children[0].Props["content"]; got != "kept" {
				t.Fatalf("%s kept the wrong child: %v", name, got)
			}
		})
	}
}

// MaybeProp's nil must not be mistaken for the footgun it sits next to: an
// argument the container cannot classify. Debug mode reports the second and
// stays silent about the first.
func TestUnknownContainerItemIsReportedButNilIsNot(t *testing.T) {
	SetDebugMode(true)
	defer SetDebugMode(false)

	t.Run("nil is silent", func(t *testing.T) {
		ClearConcerns()
		ctx := NewContext()
		ctx.BeginRenderPass()
		Column(MaybeProp(false, Text("dropped"))).Render(ctx)
		if got := Concerns(); len(got) != 0 {
			t.Fatalf("MaybeProp's false path raised concerns: %+v", got)
		}
	})

	t.Run("unclassifiable argument is reported", func(t *testing.T) {
		ClearConcerns()
		ctx := NewContext()
		ctx.BeginRenderPass()
		// The canonical mistake: a bare Style where UseStyle(style) was meant.
		// It compiles, because PropsAndChildren is any, and does nothing.
		Row(Style{Background: "#FF0000"}).Render(ctx)

		got := Concerns()
		if len(got) != 1 || got[0].Kind != ConcernUnknownItem {
			t.Fatalf("want one %s concern, got %+v", ConcernUnknownItem, got)
		}
		if !strings.Contains(got[0].Detail, "core.Style") || !strings.Contains(got[0].Detail, "Row") {
			t.Fatalf("concern detail should name the container and the type: %q", got[0].Detail)
		}
	})
}
