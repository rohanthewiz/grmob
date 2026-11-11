package core

import (
	"encoding/json"
	"strings"
	"testing"
)

// Plan R9c (N-137 c): a ComponentFunc that returns nil used to keep a child
// slot. It went to the hosts as JSON null, which both natives skip while
// parsing, so every later sibling sat one index below the path Go's diff
// used for it. These pin that a nil takes no slot in any container, and that
// core.Keyed no longer panics on one.

// nothing is a component that draws nothing, the shape the plan names.
func nothing() View {
	return ComponentFunc(func(*Context) *Node { return nil })
}

// texts returns each child's text content, in order.
func texts(t *testing.T, n *Node) []string {
	t.Helper()
	var out []string
	for i, c := range n.Children {
		if c == nil {
			t.Fatalf("%s child %d is nil", n.Type, i)
		}
		s, _ := c.Props["content"].(string)
		out = append(out, s)
	}
	return out
}

func TestANilChildTakesNoSlotInAContainer(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()
	col := Column(Text("a"), nothing(), Text("b"), nothing()).Render(ctx)

	if got := strings.Join(texts(t, col), ","); got != "a,b" {
		t.Fatalf("Column children = %q, want a,b with no slot for either nil", got)
	}
	// And so the wire form has no null in it for a host to skip.
	data, err := json.Marshal(col)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "null") {
		t.Fatalf("the encoded Column holds a null: %s", data)
	}
}

// Every container that renders a sibling list gets the same rule: they all go
// through renderAll, except WithTheme, which has its own loop.
func TestANilChildTakesNoSlotInEverySiblingList(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()
	for name, v := range map[string]View{
		"Fragment":  Fragment(Text("a"), nothing(), Text("b")),
		"For":       For([]int{0, 1, 2}, func(i, _ int) View { return IfElse(i == 1, nothing(), Text(map[int]string{0: "a", 2: "b"}[i])) }),
		"WithTheme": WithTheme(DefaultTheme, Text("a"), nothing(), Text("b")),
	} {
		if got := strings.Join(texts(t, v.Render(ctx)), ","); got != "a,b" {
			t.Errorf("%s children = %q, want a,b", name, got)
		}
	}
}

// core.Keyed set the key on whatever its child returned, so a keyed nil
// panicked. It now passes the nil up, and the container drops it.
func TestKeyedOverANilDrawsNothingAndDoesNotPanic(t *testing.T) {
	ctx := NewContext()
	ctx.BeginRenderPass()
	col := Column(Keyed("x", nothing()), Keyed("y", Text("kept"))).Render(ctx)

	if len(col.Children) != 1 || col.Children[0].Key != "y" {
		t.Fatalf("children = %+v, want only the keyed Text y", col.Children)
	}
}
