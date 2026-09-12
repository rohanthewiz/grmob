# Package reconcile

```go
import "github.com/rohanthewiz/grmob/reconcile"
```

Package reconcile is the diff: it compares two [core.Node](core.md#type-node) trees and returns the [Patch](#type-patch) list that turns the old one into the new one.

It is the whole of grmob's cross-platform contract. Every host — the Compose renderer, the SwiftUI renderer, the JavaScript runtime, the two web exporters — implements the same handful of patch types against its own widget library, and nothing else about a host is grmob's concern.

## Patches are positional, so order is load-bearing

A patch addresses its target by position ("root/1/0"), not by identity. That makes a patch list an ordered program rather than a set: applying one patch can change the path of a later one. [Diff](#func-diff) emits in an order that is safe to apply front to back — sibling removals highest index first, so that removing one never shifts the index of a removal still to come — and a renderer that reorders or parallelises the list will corrupt the tree.

## What it does not do yet

There are no move patches. A keyed child whose key changed is replaced rather than moved, because a positional path cannot express "this moved from 3 to 1" without invalidating every path after it. The result is visually correct and costs the replaced subtree its transient native state — focus, scroll offset. Identity-based node IDs are the prerequisite for fixing it.

## Index

- [`type Patch`](#type-patch)
    - [`func Diff`](#func-diff)

## Types

### type Patch

```go
type Patch struct {
	Type     string // "add", "remove", "replace", "update-props", "update-style", "add-child", "remove-child"
	TargetID string // positional node path, e.g. "root/1/0"
	Changes  any    // *core.Node for add/replace/add-child, Props map or *Style for updates
}
```

Patch represents a minimal change set between two Node trees.

TargetID is a slash-delimited positional path from the root (e.g. "root/0/2"). Because paths are positional rather than identity-based, patch order matters: renderers must apply patches in the exact order emitted. In particular, sibling removals are emitted highest-index-first so that applying one removal never shifts the index of a later removal target in the same batch.

<small>[reconcile/patch.go:17](https://github.com/rohanthewiz/grmob/blob/master/reconcile/patch.go#L17)</small>

#### func Diff

```go
func Diff(old, new *core.Node, path string) []Patch
```

Diff compares two Node trees and returns the list of patches that transforms the old tree into the new one.

The algorithm is a single top-down pass:

	old == new == nil      -> nothing
	one side nil           -> add / remove
	type changed           -> replace whole subtree (cheaper and safer than
	                          trying to morph one widget kind into another)
	otherwise              -> shallow props/style updates, then recurse into
	                          children pairwise by index

Children are matched by index, not identity. When both children at an index carry keys and the keys differ, we emit a replace for that slot: positional TargetIDs cannot express "this node moved from index 3 to index 1" safely, because the first applied move would invalidate the paths of every patch after it. True move patches require identity-based node IDs and are planned alongside that change (see ai\_docs/plans/grmob-mobile-feasibility-analysis.md). Until then, a keyed mismatch rebuilds the slot — visually correct, though the replaced subtree loses transient native state (focus, scroll offset).

<small>[reconcile/patch.go:43](https://github.com/rohanthewiz/grmob/blob/master/reconcile/patch.go#L43)</small>

