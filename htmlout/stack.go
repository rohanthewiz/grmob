package htmlout

import (
	"maps"
	"sort"

	"github.com/rohanthewiz/grmob/core"
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

// overlayTypes are the node types whose children are drawn *on top of each
// other* rather than along an axis — the z-stack, as opposed to the six
// tables' worth of flex stacks above.
//
//	htmlout (this package)      queries it through IsOverlay
//	wasm/grmob-runtime.js       restates it as OVERLAY_TYPES
//
// Pinned to the runtime's copy by TestRuntimeOverlayTypesMatchGo in
// wasm/verify, the same arrangement stackAxes and the tag table get.
//
// # Why a table for one node type
//
// Because the runtime cannot read this one. The membership question is asked
// on both DOM targets and answered in two languages, which is the whole
// criterion these tables are built on — borderResetTypes held a single member
// for the same reason, and grew to five. A `nodeType == "ZStack"` here and
// another in JavaScript would be the untracked second copy.
//
// # Why a single-cell grid and not absolute positioning
//
// The obvious HTML overlay is `position: relative` on the parent and
// `position: absolute` on each child, and it costs the layout the one thing an
// overlay still needs: an absolutely positioned child is out of flow, so it
// contributes nothing to its parent's size and the stack collapses to nothing
// unless the author states dimensions. Both natives size an overlay to its
// largest child (a SwiftUI ZStack and a Compose Box each do), so the DOM
// targets would have disagreed with them on every unsized stack.
//
// Placing every child in row 1, column 1 of a grid keeps them in flow: the
// track sizes to the widest and tallest child, the rest are drawn in the same
// cell, and the parent ends up the size both phones give it. The centring
// comes from the same declaration block — see overlayChassis.
var overlayTypes = map[string]bool{
	"ZStack": true,
}

// IsOverlay reports whether a node type draws its children on top of one
// another. See overlayTypes.
func IsOverlay(nodeType string) bool {
	return overlayTypes[nodeType]
}

// OverlayTypes returns those node types, sorted so a test looping over them
// reports in a stable order — the service StackTypes and BorderResetTypes
// provide, for the reason they give.
func OverlayTypes() []string {
	out := make([]string, 0, len(overlayTypes))
	for t := range overlayTypes {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// The two halves of the overlay's CSS, named here rather than written into
// styleValue so the WASM runtime has something to be compared against and so
// the pair reads as one decision.
//
// OverlayChassis goes on the stack itself. `align-items` and `justify-items`
// are the grid spellings of "where does an item sit inside its cell", and
// centre on both axes is the alignment contract core.ZStack documents — the
// one arrangement a SwiftUI ZStack, a Compose Box and a grid cell all agree on.
// They are the *items* properties, not the *content* ones: `justify-content`
// would place the single track inside the container, which on an auto-sized
// container is a no-op, and that difference is easy to write and impossible to
// see.
//
// OverlayChildDecl goes on every child, imposed by the parent (see imposed in
// export.go) because a child has no idea it is a layer. `grid-area: 1/1` is
// the whole of it: row 1, column 1, on all of them. The layer's placement
// inside that cell is appended after it, from stackPlacements below — the one
// part of the imposed declaration that differs from sibling to sibling.
const (
	OverlayChassis   = "display:grid; align-items:center; justify-items:center"
	OverlayChildDecl = "grid-area:1/1"
)

// stackPlacements is the one authoritative statement of core.StackAlignment ->
// the CSS grid-item placement pair, and the sixth of the mappings the two DOM
// renderers would otherwise each keep their own copy of.
//
//	htmlout (this package)      queries it through StackPlacementFor
//	wasm/grmob-runtime.js       restates it as STACK_PLACEMENTS
//
// The runtime's copy is pinned to this one by TestRuntimeStackPlacementsMatchGo
// in wasm/verify, the same way the tag, <input> type, object-fit, text-align
// and cross-axis tables are.
//
// # Why justify-self/align-self and not the chassis
//
// OverlayChassis states the stack's *default* placement with the grid `-items`
// properties; a layer overrides it with the matching `-self` property on its
// own box, which is the ordinary CSS relationship between the two. Nothing
// here touches the chassis, so a stack with one aligned layer still centres
// the other seven.
//
// justify-self is the inline axis (horizontal in every writing mode this
// framework targets) and align-self the block axis, which is why "top" writes
// the align half and "start" the justify half. The values are the grid
// spellings — `start`/`center`/`end` — not the flexbox `flex-start`/`flex-end`
// that crossAxisAligns emits: those are what a *flex* container understands,
// and an overlay is a grid.
//
// # The centre IS in the table, unlike every other zero value here
//
// core.StackAlignments() omits StackAlignCenter, because it is the field's
// zero value and no *dispatch* should have an arm for "do the default". This
// table is not a dispatch: it is a total statement of what an overlay writes
// on each of its layers, and the centre row is what makes it total.
//
// Two things need it. The runtime restates every property it manages on every
// pass so an update-style patch clears what the new Style dropped — a layer
// that loses its placement has to go back to the middle, and a row of "" is
// how the pass says so. And `align-self` is a property a *layer* can already
// set for itself, through Style.AlignSelf, which is CSS's flexbox spelling and
// applies to a grid item too: a stack that wrote nothing on its unplaced
// layers would let that flex prop move one of them, on the two DOM targets
// only, in flat contradiction of the alignment contract core.ZStack documents.
// Imposing the centre closes that, on both of them, for the price of two
// declarations per layer.
var stackPlacements = map[core.StackAlignment]string{
	core.StackAlignCenter:      "justify-self:center; align-self:center",
	core.StackAlignTopStart:    "justify-self:start; align-self:start",
	core.StackAlignTop:         "justify-self:center; align-self:start",
	core.StackAlignTopEnd:      "justify-self:end; align-self:start",
	core.StackAlignStart:       "justify-self:start; align-self:center",
	core.StackAlignEnd:         "justify-self:end; align-self:center",
	core.StackAlignBottomStart: "justify-self:start; align-self:end",
	core.StackAlignBottom:      "justify-self:center; align-self:end",
	core.StackAlignBottomEnd:   "justify-self:end; align-self:end",
}

// StackPlacementFor returns the CSS declarations that place a layer.
//
// Total over core.StackAlignment's declared values, centre included. A value
// the table does not name — which can only be a hand-written string, since the
// type is closed by its const block — falls back to the centre rather than to
// nothing, so an unrecognised placement renders as the contract's default
// instead of leaving a layer wherever a stray flex prop put it.
func StackPlacementFor(align core.StackAlignment) string {
	if decl, ok := stackPlacements[align]; ok {
		return decl
	}
	return stackPlacements[core.StackAlignCenter]
}

// StackPlacements returns a copy of the whole table, keyed by the placement's
// string form — the service CrossAxisAligns provides, for the reason it gives:
// the conformance test in wasm/verify compares table against table, and a
// caller that could mutate the original would be comparing the runtime against
// whatever the last test left behind.
func StackPlacements() map[string]string {
	out := make(map[string]string, len(stackPlacements))
	for k, v := range stackPlacements {
		out[string(k)] = v
	}
	return out
}
