package htmlout

import (
	"maps"
	"sort"
)

// stackAxes is the one authoritative statement of which node types are stacks
// — containers that lay their children out along an axis whether or not the
// author's Style says so — and which axis each one stacks along.
//
//	htmlout (this package)      queries it through StackAxisFor
//	wasm/grmob-runtime.js       restates it in JavaScript (stackAxisFor)
//
// Pinned to the runtime's copy by TestRuntimeStackAxesMatchGo in wasm/verify,
// the same arrangement as the tag, <input> type, object-fit, text-align and
// cross-axis tables.
//
// # What the table is for
//
// On both natives a container is a stack by construction: a Compose
// Row/Column and a SwiftUI HStack/VStack have no other mode, and Box,
// SafeArea, Card, Scroll and List all route through one of them. HTML has no
// such default — a <div> is block flow, which runs inline children (Text
// exports as a <span>) together on one line and drops gap, justify-content
// and align-items entirely. So the DOM targets have to opt into the stack or
// diverge from the other two.
//
// The WASM runtime opted in long ago; this exporter did not, and the gap was
// visible in its output: examples/layout's BodySection is a core.Row that
// carries only padding, and it exported as `<div style="padding:…">` — a
// block-flow box whose two children ran down the page in the browser and
// across it on every other target. Nothing errored, because "block flow" is a
// perfectly good layout, just not the one the node asked for.
//
// # Why the value is the axis and not just membership
//
// The runtime's copy used to be a bare Set with a `Type === "Row" ? "row" :
// "column"` ternary beside it, and this package had that same ternary written
// out a third time in styleValue. Membership and direction are one fact —
// a Row is a stack *because* it stacks horizontally — so the table states it
// once and both renderers read the axis from here.
//
// # TabView, and the part of its divergence this row does not close
//
// TabView is a column stack on both natives: Renderer.kt builds a Compose
// Column holding a TabRow and the selected page, Renderer.swift a SwiftUI
// VStack holding a hand-rolled tab bar and the same page. It was the last
// type that stacked there and ran in block flow here, and it was held out of
// this table's first cut on the grounds that the two DOM targets at least
// agreed with each other about it. Agreeing on the wrong layout is still the
// wrong layout, so it is in now, on the axis both natives use.
// mobile/verify's TestNativeTabViewIsAColumnStack holds that claim against
// the two renderers rather than leaving it as prose here.
//
// The row closes the axis and nothing else, and the rest of the gap is worth
// naming because this table is where a reader will come looking for it:
// neither DOM target draws a tab bar or reads selectedIndex at all. Both
// render every page of the TabView as a visible child, where the natives draw
// the bar and the one selected page. That is a missing feature rather than a
// layout default — the runtime would have to turn the tabs prop into real
// elements and wire their clicks back through onTabChange, and it would have
// to hide the unselected pages with display:none rather than by dropping
// them, since patches are addressed positionally and its DOM has to stay
// isomorphic to the node tree. Until that pass, this row at least has the
// pages stack down the page as they do on device instead of running together
// on one line, and makes the container's own gap, justify-content and
// align-items work.
//
// TabView is deliberately *not* in alignFallbackAxes next door. That gate is
// the types whose native stack reads Style.Align for cross-axis placement,
// and neither TabView composite does — Compose's Column is given no
// horizontalAlignment and SwiftUI's VStack no alignment argument. Scroll is
// the standing precedent for a type that stacks here and reads no fallback:
// the two tables answer different questions and are not required to agree.
//
// # The types that are deliberately absent
//
//   - Modal carries its own fixed-overlay chassis on both DOM targets, and
//     that chassis already sets display and flex-direction (and toggles
//     display through the visible prop, which a default here would fight).
//   - Spacer is a sized void, not a container: it has no children to stack.
//   - Fragment and Theme are the known divergence: they stack in the runtime,
//     which must box them to keep its positional patch addressing valid, and
//     this exporter emits their children with no box at all (transparentTypes
//     in tag.go has the full reason). The conformance test narrows the
//     exemption to those two types rather than skipping the comparison.
var stackAxes = map[string]string{
	"Row":      "row",
	"Column":   "column",
	"Card":     "column",
	"Box":      "column",
	"Scroll":   "column",
	"SafeArea": "column",
	"List":     "column",
	"TabView":  "column",
}

// StackAxisFor returns the flex axis a node type stacks its children along,
// or "" for a type that is not a stack container.
//
// The "" answer is load-bearing in two ways at the call sites: it is what
// keeps a Text or a Button that happens to set a container prop from being
// given a stacking default it never asked for, and it is what a caller reads
// as "this node is only a flex container if its Style makes it one".
func StackAxisFor(nodeType string) string {
	return stackAxes[nodeType]
}

// StackAxes returns a copy of the whole table, for the WASM runtime
// conformance test, which compares table against table and so cannot go
// through StackAxisFor one key at a time.
//
// A copy, not the map itself, for the reason Tags returns one: a
// package-level map is reachable and writable by any importer, and the test
// deletes from what it is given as it matches rows.
func StackAxes() map[string]string {
	out := make(map[string]string, len(stackAxes))
	maps.Copy(out, stackAxes)
	return out
}

// StackTypes returns the stack container types, sorted so a test looping over
// them reports in a stable order — the same service AlignFallbackTypes and
// TransparentTypes provide, and for the same reason: a hand-written list in a
// test is the untracked second copy these tables exist to remove.
func StackTypes() []string {
	out := make([]string, 0, len(stackAxes))
	for t := range stackAxes {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
