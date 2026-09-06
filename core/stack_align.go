package core

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
//	iOS       .frame(maxWidth: .infinity, maxHeight: .infinity, alignment:)
//	          around the layer — SwiftUI has no per-child ZStack alignment
//	Android   Modifier.align(Alignment.*) in the Box's scope
//
// The three constructs each have a nine-value 2D placement vocabulary and they
// agree value for value, which is what makes this portable where a flexbox
// property would not have been.
//
// # The one divergence, and how to avoid it
//
// SwiftUI's spelling is the odd one: a frame that fills the stack, with the
// layer placed inside it. A filling frame is *greedy*, so on iOS a stack that
// states no size of its own grows to whatever its parent offers as soon as one
// layer is aligned, where a Compose Box and a CSS grid track both stay the
// size of their largest child.
//
// core.ZStack's own doc already asks a stack to state its dimensions ("pinning
// the box is what keeps a smaller overlay from deciding the size"), and a
// stack that does is identical on all four targets. An unsized stack with an
// aligned layer is the case to avoid — and it is close to meaningless anyway,
// since "top-start" of a box with no size is wherever the largest layer
// happens to end.
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
func StackAlign(value StackAlignment) StyleProp {
	return styleFunc(func(s *Style) {
		s.StackAlign = value
	})
}
