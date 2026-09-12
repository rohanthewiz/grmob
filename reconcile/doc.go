// Package reconcile is the diff: it compares two [core.Node] trees and returns
// the [Patch] list that turns the old one into the new one.
//
// It is the whole of grmob's cross-platform contract. Every host — the Compose
// renderer, the SwiftUI renderer, the JavaScript runtime, the two web exporters
// — implements the same handful of patch types against its own widget library,
// and nothing else about a host is grmob's concern.
//
// # Patches are positional, so order is load-bearing
//
// A patch addresses its target by position ("root/1/0"), not by identity. That
// makes a patch list an ordered program rather than a set: applying one patch
// can change the path of a later one. [Diff] emits in an order that is safe to
// apply front to back — sibling removals highest index first, so that removing
// one never shifts the index of a removal still to come — and a renderer that
// reorders or parallelises the list will corrupt the tree.
//
// # What it does not do yet
//
// There are no move patches. A keyed child whose key changed is replaced rather
// than moved, because a positional path cannot express "this moved from 3 to 1"
// without invalidating every path after it. The result is visually correct and
// costs the replaced subtree its transient native state — focus, scroll offset.
// Identity-based node IDs are the prerequisite for fixing it.
package reconcile
