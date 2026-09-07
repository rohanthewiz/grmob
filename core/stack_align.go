package core

import "sort"

// StackAlignment is where one layer of a ZStack sits inside the stack.
//
// Every layer of a ZStack is centred, which is the alignment contract the node
// type documents and the one arrangement a SwiftUI ZStack, a Compose Box and a
// single-cell CSS grid all agree on without argument. This is the opt-out, per
// layer:
//
//	   top-start        top       top-end
//	       start      (center)      end
//	bottom-start     bottom    bottom-end
//
// The centre of that grid is the zero value and has no spelling of its own —
// StackAlignCenter is "" — so a layer that says nothing is placed exactly as
// every layer was before this type existed.
//
// # Why a two-axis value rather than Style.AlignSelf
//
// AlignSelf is CSS's flexbox align-self: one axis, and *which* axis depends on
// the container's flex direction. A layer needs both axes named at once, and a
// stack has no main axis for the other one to be the cross of. Reusing the
// field would also have given it two meanings dispatched on the parent's node
// type — a stack's child means one thing by it and a Row's child another —
// which is the shape core.Box and core.ZStack were split apart to stop.
//
// It is also why AlignSelf could not simply be taught to the natives: it is
// honoured by the two DOM targets only, and a portable-looking prop that works
// on the web is precisely what core.ZStack exists instead of.
//
// # What each target does with it
//
//	web       justify-self / align-self on the grid item, imposed by the
//	          stack (see htmlout's imposed) rather than written by the layer,
//	          because align-self means something else on a flex child
//	iOS       a coordinate handed to GrMobStackLayout, a custom SwiftUI
//	          Layout — SwiftUI has no per-child ZStack alignment
//	Android   Modifier.align(Alignment.*) in the Box's scope
//
// The three constructs each have a nine-value 2D placement vocabulary and they
// agree value for value, which is what makes this portable where a flexbox
// property would not have been.
//
// # The divergence this used to carry, and what closed it
//
// SwiftUI's spelling was the odd one: a frame that fills the stack, with the
// layer placed inside it. That is SwiftUI's own idiom for a job it has no
// direct spelling for, and a filling frame is *greedy* — so on iOS a stack
// that stated no size of its own grew to whatever its parent offered as soon
// as one layer was aligned, where a Compose Box and a CSS grid track both stay
// the size of their largest child.
//
// It was documented in four places and avoided by pinning the stack's box,
// which core.ZStack asks for anyway. What it was not was *pinned*: ios/verify
// type-checks and replays a transcript, and neither of those measures a size.
//
// The frame is gone. The iOS renderer places a layer by coordinate through a
// custom SwiftUI Layout (GrMobStackLayout), so nothing is wrapped and nothing
// is greedy, and the container reports the largest child on each axis like the
// other three. The arithmetic lives in GrMobStack.swift as pure CoreGraphics —
// split out for the reason GrMobFlexSolver was — and ios/verify measures both
// halves of it: what the stack sizes to, and where each of the nine anchors
// puts a layer.
//
// Pinning a stack's dimensions is still good advice, and for the reason it
// always had: "top-start" of a box with no size is wherever the largest layer
// happens to end. It is no longer the difference between two renderings.
type StackAlignment string

const (
	// StackAlignCenter is the zero value: centred on both axes, which is
	// core.ZStack's contract and what every layer did before this type. It has
	// no spelling so that an unset Style.StackAlign *is* it — the same reason
	// SelectedUnset and VariantDefault are the empty string.
	StackAlignCenter StackAlignment = ""

	StackAlignTopStart StackAlignment = "top-start"
	StackAlignTop      StackAlignment = "top"
	StackAlignTopEnd   StackAlignment = "top-end"

	// StackAlignStart and StackAlignEnd are the middle row: the named edge
	// horizontally, centred vertically. Spelled without a "center-" prefix to
	// match StackAlignTop and StackAlignBottom, which are the same shape one
	// axis over.
	StackAlignStart StackAlignment = "start"
	StackAlignEnd   StackAlignment = "end"

	StackAlignBottomStart StackAlignment = "bottom-start"
	StackAlignBottom      StackAlignment = "bottom"
	StackAlignBottomEnd   StackAlignment = "bottom-end"
)

// StackAlignments returns the eight stated placements, in declaration order.
//
// StackAlignCenter is excluded for the reason SelectedStates() excludes
// SelectedUnset: it is the field's zero value, every node in every tree
// carries it, and no renderer has — or should have — an arm for "the default".
// A coverage check that demanded one would be asking each renderer to
// implement doing nothing.
//
// Pinned to the const block above by stack_align_enum_test.go, and consumed by
// mobile/verify's native coverage checks and by htmlout's placement table, so
// a census that quietly stopped listing a value would quietly stop requiring
// an arm for it on three renderers at once.
func StackAlignments() []StackAlignment {
	return []StackAlignment{
		StackAlignTopStart,
		StackAlignTop,
		StackAlignTopEnd,
		StackAlignStart,
		StackAlignEnd,
		StackAlignBottomStart,
		StackAlignBottom,
		StackAlignBottomEnd,
	}
}

// StackAlign places this node inside the core.ZStack it is a layer of.
//
//	core.ZStack(
//	    core.Width("160px"), core.Height("160px"),
//	    rose,
//	    core.Text("N", core.StackAlign(core.StackAlignTop)),
//	)
//
// It says nothing anywhere else. A stack is the only container that places its
// children in two dimensions at once, so this prop on a child of a Column, a
// Row or a Box is inert on every target — deliberately, and not merely as an
// accident of the implementations: on the web the value never reaches the
// element at all (the stack imposes the declaration, see htmlout's imposed),
// because align-self *does* mean something to a flex child and a layer prop
// that silently re-placed a row's children would be worse than one that did
// nothing.
//
// Inert is the right behaviour and *silent* is not, so the tree walk says so:
// with debug mode on, core.AuditTree reports a placement no container will read
// as ConcernInertPlacement, naming the node path and the container that was
// going to place it. That is the only diagnostic any target produces, and the
// argument for putting it there rather than in a renderer is in
// placement_audit.go.
func StackAlign(value StackAlignment) StyleProp {
	return styleFunc(func(s *Style) {
		s.StackAlign = value
	})
}

// The two node-type facts a placement depends on, stated here because this is
// the property they exist for and because core owns the node types.
//
// They were htmlout's alone until the audit needed them. That is the wrong way
// round for a fact about core.ZStack — an exporter is a consumer of the node
// vocabulary, not its author — and it left core unable to say anything about a
// placement it defines. htmlout's own tables carry the *rendering* consequence
// of each set (a grid cell, a wrapper that would swallow a flex gap) and stay
// where they are; TestHtmloutAgreesWithCoreOnWhoPlacesAndWhoGroups pins them to
// these, so there is one census with two consumers rather than two lists that
// happen to agree.

// placingContainers are the node types that read a child's StackAlign. A
// layer's placement is the *container's* decision to honour it — see
// StackAlign's doc and htmlout's imposed channel — so this is the set of
// containers that make the decision at all.
//
// One entry, and the reason it is a set rather than a comparison against the
// string "ZStack" is that a second overlay is a change to this line rather
// than to the audit, the exporter and two renderers.
var placingContainers = map[string]bool{
	"ZStack": true,
}

// groupingContainers are the node types that have no box of their own and
// hand their children straight to the container above them: core.For's
// Fragment and core.WithTheme's Theme.
//
// They matter to a placement because they are transparent to one. A core.For
// inside a ZStack overlays what it generated — htmlout forwards the imposed
// declaration through a Fragment rather than absorbing it, and both natives
// render these as a bare pass-through — so the container that places a node is
// the nearest ancestor that is not one of these, not simply its tree parent.
var groupingContainers = map[string]bool{
	"Fragment": true,
	"Theme":    true,
}

// PlacingContainers returns the node types that place their children, sorted.
//
// Sorted for the reason htmlout's OverlayTypes is: a test looping over a map
// reports in a different order every run, and a census that names the offender
// wants one order.
func PlacingContainers() []string {
	return sortedNodeTypes(placingContainers)
}

// GroupingContainers returns the transparent node types, sorted. See
// groupingContainers.
func GroupingContainers() []string {
	return sortedNodeTypes(groupingContainers)
}

// placesChildren reports whether a container of this type reads its children's
// StackAlign.
func placesChildren(nodeType string) bool { return placingContainers[nodeType] }

// groupsChildren reports whether this node type is transparent to a placement:
// it has no box, so the container above it is the one that places its children.
func groupsChildren(nodeType string) bool { return groupingContainers[nodeType] }

func sortedNodeTypes(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
