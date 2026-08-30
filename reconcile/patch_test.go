package reconcile

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// node is a test helper that builds a Node the same way core's view builders
// do: a fresh allocation per call, so tests exercise the value-vs-pointer
// comparison semantics the reconciler must get right.
func node(typ string, props map[string]any, style *core.Style, children ...*core.Node) *core.Node {
	return &core.Node{
		Type:     typ,
		Props:    props,
		Style:    style,
		Children: children,
	}
}

func keyed(n *core.Node, key string) *core.Node {
	n.Key = key
	return n
}

// requirePatchTypes asserts the exact sequence of patch types, since renderers
// apply patches in emitted order and that order is part of the contract.
func requirePatchTypes(t *testing.T, patches []Patch, want ...string) {
	t.Helper()
	if len(patches) != len(want) {
		t.Fatalf("got %d patches %+v, want types %v", len(patches), patches, want)
	}
	for i, w := range want {
		if patches[i].Type != w {
			t.Errorf("patch[%d].Type = %q, want %q (all: %+v)", i, patches[i].Type, w, patches)
		}
	}
}

func TestDiffBothNil(t *testing.T) {
	// Regression: the previous implementation dereferenced old.Type and panicked.
	if patches := Diff(nil, nil, "root"); len(patches) != 0 {
		t.Fatalf("Diff(nil, nil) = %+v, want no patches", patches)
	}
}

func TestDiffAddWhenOldNil(t *testing.T) {
	n := node("Text", map[string]any{"content": "hi"}, nil)
	patches := Diff(nil, n, "root")
	requirePatchTypes(t, patches, "add")
	if patches[0].TargetID != "root" || patches[0].Changes != n {
		t.Errorf("add patch = %+v, want target root carrying the new node", patches[0])
	}
}

func TestDiffRemoveWhenNewNil(t *testing.T) {
	patches := Diff(node("Text", nil, nil), nil, "root")
	requirePatchTypes(t, patches, "remove")
	if patches[0].TargetID != "root" {
		t.Errorf("remove target = %q, want root", patches[0].TargetID)
	}
}

func TestDiffTypeChangeReplacesSubtree(t *testing.T) {
	old := node("Text", nil, nil)
	new := node("Button", nil, nil)
	patches := Diff(old, new, "root")
	requirePatchTypes(t, patches, "replace")
	if patches[0].Changes != new {
		t.Errorf("replace should carry the whole new node, got %+v", patches[0].Changes)
	}
}

func TestDiffIdenticalTreesEmitNothing(t *testing.T) {
	// The critical regression test for the style pointer-compare bug: both
	// trees are built from scratch (distinct Style allocations, equal values),
	// exactly as consecutive renders do. A correct reconciler must see no change.
	build := func() *core.Node {
		return node("Column", nil, &core.Style{Gap: 8},
			node("Text", map[string]any{"content": "hello"}, &core.Style{FontSize: 16, TextColor: "#333"}),
			node("Button", map[string]any{"label": "Go", "onClick": "cb_1"}, &core.Style{BorderRadius: 6}),
		)
	}
	if patches := Diff(build(), build(), "root"); len(patches) != 0 {
		t.Fatalf("identical trees produced patches: %+v", patches)
	}
}

func TestDiffStyleValueChange(t *testing.T) {
	old := node("Text", nil, &core.Style{FontSize: 16})
	new := node("Text", nil, &core.Style{FontSize: 20})
	patches := Diff(old, new, "root")
	requirePatchTypes(t, patches, "update-style")
	if patches[0].Changes != new.Style {
		t.Errorf("update-style should carry the new style, got %+v", patches[0].Changes)
	}
}

func TestDiffStyleNilTransitions(t *testing.T) {
	styled := func() *core.Node { return node("Text", nil, &core.Style{FontSize: 16}) }
	bare := func() *core.Node { return node("Text", nil, nil) }

	requirePatchTypes(t, Diff(bare(), styled(), "root"), "update-style")
	requirePatchTypes(t, Diff(styled(), bare(), "root"), "update-style")
	if patches := Diff(bare(), bare(), "root"); len(patches) != 0 {
		t.Fatalf("nil->nil style produced patches: %+v", patches)
	}
}

func TestDiffNestedStyleChange(t *testing.T) {
	// DeepEqual must follow nested style pointers (HoverStyle etc.).
	old := node("Button", nil, &core.Style{HoverStyle: &core.Style{Background: "#eee"}})
	new := node("Button", nil, &core.Style{HoverStyle: &core.Style{Background: "#ddd"}})
	requirePatchTypes(t, Diff(old, new, "root"), "update-style")

	same1 := node("Button", nil, &core.Style{HoverStyle: &core.Style{Background: "#eee"}})
	same2 := node("Button", nil, &core.Style{HoverStyle: &core.Style{Background: "#eee"}})
	if patches := Diff(same1, same2, "root"); len(patches) != 0 {
		t.Fatalf("equal nested styles produced patches: %+v", patches)
	}
}

func TestDiffPropsValueChange(t *testing.T) {
	old := node("Text", map[string]any{"content": "a"}, nil)
	new := node("Text", map[string]any{"content": "b"}, nil)
	patches := Diff(old, new, "root")
	requirePatchTypes(t, patches, "update-props")
}

func TestDiffPropsKeySetChangeSameLength(t *testing.T) {
	// Same map length, different key sets: the length check alone cannot catch
	// this; the per-key membership check must.
	old := node("Input", map[string]any{"value": "x"}, nil)
	new := node("Input", map[string]any{"placeholder": "x"}, nil)
	requirePatchTypes(t, Diff(old, new, "root"), "update-props")
}

func TestDiffPropsUncomparableValuesDoNotPanic(t *testing.T) {
	// Regression: `b[k] != v` panics at runtime when a prop holds a slice or
	// map. DeepEqual must both survive and correctly compare such values.
	oldEq := node("List", map[string]any{"items": []string{"a", "b"}}, nil)
	newEq := node("List", map[string]any{"items": []string{"a", "b"}}, nil)
	if patches := Diff(oldEq, newEq, "root"); len(patches) != 0 {
		t.Fatalf("equal slice props produced patches: %+v", patches)
	}

	oldNe := node("List", map[string]any{"items": []string{"a", "b"}}, nil)
	newNe := node("List", map[string]any{"items": []string{"a", "c"}}, nil)
	requirePatchTypes(t, Diff(oldNe, newNe, "root"), "update-props")
}

func TestDiffChildrenAdded(t *testing.T) {
	old := node("Column", nil, nil,
		node("Text", map[string]any{"content": "a"}, nil),
	)
	new := node("Column", nil, nil,
		node("Text", map[string]any{"content": "a"}, nil),
		node("Text", map[string]any{"content": "b"}, nil),
		node("Text", map[string]any{"content": "c"}, nil),
	)
	patches := Diff(old, new, "root")
	requirePatchTypes(t, patches, "add-child", "add-child")
	for _, p := range patches {
		if p.TargetID != "root" {
			t.Errorf("add-child targets the parent; got target %q", p.TargetID)
		}
	}
	// Appended in tree order so the renderer reproduces the new sibling order.
	if patches[0].Changes.(*core.Node).Props["content"] != "b" ||
		patches[1].Changes.(*core.Node).Props["content"] != "c" {
		t.Errorf("add-child patches out of order: %+v", patches)
	}
}

func TestDiffChildrenRemovedHighestIndexFirst(t *testing.T) {
	old := node("Column", nil, nil,
		node("Text", nil, nil),
		node("Text", nil, nil),
		node("Text", nil, nil),
	)
	new := node("Column", nil, nil,
		node("Text", nil, nil),
	)
	patches := Diff(old, new, "root")
	requirePatchTypes(t, patches, "remove-child", "remove-child")
	// Descending order: removing root/2 first leaves root/1 still meaning the
	// same child when an index-resolving renderer applies the next patch.
	if patches[0].TargetID != "root/2" || patches[1].TargetID != "root/1" {
		t.Errorf("removals must be emitted highest-index-first, got %q then %q",
			patches[0].TargetID, patches[1].TargetID)
	}
}

func TestDiffKeyedMismatchReplacesSlot(t *testing.T) {
	old := node("Column", nil, nil,
		keyed(node("Text", map[string]any{"content": "a"}, nil), "a"),
		keyed(node("Text", map[string]any{"content": "b"}, nil), "b"),
	)
	new := node("Column", nil, nil,
		keyed(node("Text", map[string]any{"content": "b"}, nil), "b"),
		keyed(node("Text", map[string]any{"content": "a"}, nil), "a"),
	)
	patches := Diff(old, new, "root")
	requirePatchTypes(t, patches, "replace", "replace")
	if patches[0].TargetID != "root/0" || patches[1].TargetID != "root/1" {
		t.Errorf("unexpected replace targets: %+v", patches)
	}
}

func TestDiffKeyedSameKeyDiffsInPlace(t *testing.T) {
	// Matching keys must update in place (state-preserving), not replace.
	old := node("Column", nil, nil,
		keyed(node("Text", map[string]any{"content": "old"}, nil), "row-1"),
	)
	new := node("Column", nil, nil,
		keyed(node("Text", map[string]any{"content": "new"}, nil), "row-1"),
	)
	patches := Diff(old, new, "root")
	requirePatchTypes(t, patches, "update-props")
	if patches[0].TargetID != "root/0" {
		t.Errorf("update should target the child path, got %q", patches[0].TargetID)
	}
}

func TestDiffNestedChildPathAddressing(t *testing.T) {
	old := node("Column", nil, nil,
		node("Row", nil, nil,
			node("Text", map[string]any{"content": "x"}, nil),
			node("Text", map[string]any{"content": "y"}, nil),
		),
	)
	new := node("Column", nil, nil,
		node("Row", nil, nil,
			node("Text", map[string]any{"content": "x"}, nil),
			node("Text", map[string]any{"content": "z"}, nil),
		),
	)
	patches := Diff(old, new, "root")
	requirePatchTypes(t, patches, "update-props")
	if patches[0].TargetID != "root/0/1" {
		t.Errorf("nested target = %q, want root/0/1", patches[0].TargetID)
	}
}

func TestDiffNilChildSlots(t *testing.T) {
	// Defensive: a View that returns nil produces a nil child pointer. The
	// pairwise walk must tolerate nil on either or both sides.
	old := node("Column", nil, nil, nil, node("Text", nil, nil))
	new := node("Column", nil, nil, nil, nil)
	patches := Diff(old, new, "root")
	requirePatchTypes(t, patches, "remove")
	if patches[0].TargetID != "root/1" {
		t.Errorf("remove target = %q, want root/1", patches[0].TargetID)
	}
}

func TestDiffMixedChangeOrdering(t *testing.T) {
	// A node with prop, style, and child-count changes at once: updates come
	// before structural child patches, and each patch stands alone.
	old := node("Column", map[string]any{"scrollable": true}, &core.Style{Gap: 4},
		node("Text", nil, nil),
		node("Text", nil, nil),
	)
	new := node("Column", map[string]any{"scrollable": false}, &core.Style{Gap: 8},
		node("Text", nil, nil),
	)
	patches := Diff(old, new, "root")
	requirePatchTypes(t, patches, "update-props", "update-style", "remove-child")
	if patches[2].TargetID != "root/1" {
		t.Errorf("remove-child target = %q, want root/1", patches[2].TargetID)
	}
}
