package reconcile

import (
	"reflect"
	"strconv"

	"github.com/rohanthewiz/grmob/core"
)

// Patch represents a minimal change set between two Node trees.
//
// TargetID is a slash-delimited positional path from the root (e.g. "root/0/2").
// Because paths are positional rather than identity-based, patch order matters:
// renderers must apply patches in the exact order emitted. In particular,
// sibling removals are emitted highest-index-first so that applying one removal
// never shifts the index of a later removal target in the same batch.
type Patch struct {
	Type     string // "add", "remove", "replace", "update-props", "update-style", "add-child", "remove-child"
	TargetID string // positional node path, e.g. "root/1/0"
	Changes  any    // *core.Node for add/replace/add-child, Props map or *Style for updates
}

// Diff compares two Node trees and returns the list of patches that transforms
// the old tree into the new one.
//
// The algorithm is a single top-down pass:
//
//	old == new == nil      -> nothing
//	one side nil           -> add / remove
//	type changed           -> replace whole subtree (cheaper and safer than
//	                          trying to morph one widget kind into another)
//	otherwise              -> shallow props/style updates, then recurse into
//	                          children pairwise by index
//
// Children are matched by index, not identity. When both children at an index
// carry keys and the keys differ, we emit a replace for that slot: positional
// TargetIDs cannot express "this node moved from index 3 to index 1" safely,
// because the first applied move would invalidate the paths of every patch
// after it. True move patches require identity-based node IDs and are planned
// alongside that change (see ai_docs/plans/grmob-mobile-feasibility-analysis.md).
// Until then, a keyed mismatch rebuilds the slot — visually correct, though the
// replaced subtree loses transient native state (focus, scroll offset).
func Diff(old, new *core.Node, path string) []Patch {
	// Same pointer: the subtree is unchanged by definition, since nodes are
	// frozen after render (see core.Node's immutability contract). This is the
	// fast path core.Cached buys — a cached view returns the identical *Node
	// every pass, so its whole subtree costs one comparison here instead of a
	// deep props/style/children walk. Also covers old == new == nil.
	if old == new {
		return nil
	}
	// One side absent (both-absent was handled by the pointer guard above).
	// These nil checks also protect the recursive calls below from
	// dereferencing nil when a slot is empty on one side.
	if old == nil {
		return []Patch{{
			Type:     "add",
			TargetID: path,
			Changes:  new,
		}}
	}
	if new == nil {
		return []Patch{{
			Type:     "remove",
			TargetID: path,
		}}
	}
	if old.Type != new.Type {
		return []Patch{{
			Type:     "replace",
			TargetID: path,
			Changes:  new,
		}}
	}

	var patches []Patch

	if propsChanged(old.Props, new.Props) {
		patches = append(patches, Patch{
			Type:     "update-props",
			TargetID: path,
			Changes:  new.Props,
		})
	}
	if styleChanged(old.Style, new.Style) {
		patches = append(patches, Patch{
			Type:     "update-style",
			TargetID: path,
			Changes:  new.Style,
		})
	}

	// Recurse into the children both trees have, pairwise by index.
	minLen := min(len(old.Children), len(new.Children))
	for i := range minLen {
		childPath := path + "/" + strconv.Itoa(i)
		oldChild := old.Children[i]
		newChild := new.Children[i]

		if oldChild != nil && newChild != nil &&
			oldChild.Key != "" && newChild.Key != "" &&
			oldChild.Key != newChild.Key {
			// Keyed slot whose occupant changed: rebuild rather than diff.
			// Diffing across different keys would leak state between logically
			// distinct items (the classic un-keyed-list bug this guards against).
			patches = append(patches, Patch{
				Type:     "replace",
				TargetID: childPath,
				Changes:  newChild,
			})
		} else {
			patches = append(patches, Diff(oldChild, newChild, childPath)...)
		}
	}

	// New tree has extra children: append them in order. add-child targets the
	// parent, so these are index-shift safe by construction.
	for i := minLen; i < len(new.Children); i++ {
		patches = append(patches, Patch{
			Type:     "add-child",
			TargetID: path,
			Changes:  new.Children[i],
		})
	}

	// Old tree has extra children: remove them highest-index-first. A renderer
	// that resolves paths against its live tree would otherwise remove index N
	// and then find index N+1 pointing at the wrong (shifted) sibling.
	for i := len(old.Children) - 1; i >= minLen; i-- {
		patches = append(patches, Patch{
			Type:     "remove-child",
			TargetID: path + "/" + strconv.Itoa(i),
		})
	}

	return patches
}

// propsChanged reports whether two prop maps differ.
//
// reflect.DeepEqual is used per value instead of `!=` for two reasons:
// props may hold uncomparable values (slices, maps, structs containing them),
// and comparing those with == panics at runtime; and DeepEqual gives correct
// content comparison for those same values. The explicit missing-key check
// distinguishes "key absent in b" from "key present with a nil/zero value",
// which the previous `b[k] != v` lookup conflated.
func propsChanged(a, b map[string]any) bool {
	if len(a) != len(b) {
		return true
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok {
			return true
		}
		if !reflect.DeepEqual(av, bv) {
			return true
		}
	}
	return false
}

// styleChanged reports whether two styles differ by value.
//
// DeepEqual follows pointers, so two distinct allocations with equal contents
// compare equal. That property is load-bearing here: every render rebuilds the
// tree and re-allocates every Style, so the previous pointer comparison
// (`a != b`) flagged every styled node as changed on every render, turning the
// "minimal" diff into a near-full tree broadcast. It also correctly handles
// nested pointers inside Style (HoverStyle, FocusStyle, PseudoStates).
func styleChanged(a, b *core.Style) bool {
	return !reflect.DeepEqual(a, b)
}
