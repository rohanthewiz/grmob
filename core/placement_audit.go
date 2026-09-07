package core

import "fmt"

// The placement audit: the one finding in AuditTree's walk that is not about
// accessibility.
//
// # Why it rides along in that walk
//
// It is the same shape as the other four and it fails the same way. A
// core.StackAlign on a node whose container is not an overlay compiles, merges
// into the Style like any other prop, crosses the bridge as a JSON key, and
// does nothing on all four targets with no diagnostic anywhere — which is
// exactly the line a11y_audit.go draws for its own four ("would a reader be
// told something false, with nothing anywhere saying so", one property over:
// would an author be told nothing).
//
// It is also a fact about a *relationship*, which is the reason that walk
// exists rather than a guard in the exporters. No renderer can report it, and
// two of them are structurally unable to: htmlout never writes the value onto
// the element at all — the stack imposes the declaration and a non-stack
// imposes nothing (see htmlout's imposed) — so by the time a Row's child is
// being written there is nothing left to notice. The natives are the same
// story from the other end: grMobStackAlignment is consulted by the two stack
// renderers and by nobody else, so a placement on a Column's child is a string
// no code path ever reads.
//
// The walk already carries the node, the path and — once it is threaded — the
// container above it, so this costs one map lookup per node in debug mode and
// nothing at all outside it.
//
// # Why the *placing* container and not the tree parent
//
// A Fragment and a Theme have no box, so a placement inside one is honoured by
// whatever container is above *them*: htmlout forwards the imposed declaration
// through a Fragment rather than absorbing it, precisely so that a core.For
// inside a ZStack overlays what it generated instead of stacking it. Checking
// the tree parent would report every generated layer in the framework's own
// idiom for generating layers.
//
// See groupingContainers and placingContainers in stack_align.go, which are
// the two sets this walks by and which htmlout's own tables are pinned to.

// ConcernInertPlacement: a node states core.Style.StackAlign and the container
// that would place it is not an overlay.
//
// Deliberately inert rather than accidentally so — a layer prop that re-placed
// a Row's children would be worse than one that did nothing, and align-self
// already means something else to a flex child — which is what makes this a
// concern and not a bug to fix in a renderer. The author has written a prop
// that cannot work where they put it, and the four targets agree silently.
const ConcernInertPlacement = "inert-stack-placement"

// reportInertPlacement flags a placement no container will read.
//
// placer is the node type of the nearest ancestor that is not a grouping
// container, or "" for the root — a root node has nothing above it to place it,
// which is the same finding with a different sentence.
//
// The centre needs no check and must not get one: StackAlignCenter is the
// field's zero value, so every node in every tree carries it, and reporting it
// would mean reporting the whole tree.
func reportInertPlacement(n *Node, path, placer string) {
	if n.Style == nil || n.Style.StackAlign == StackAlignCenter {
		return
	}
	if placesChildren(placer) {
		return
	}

	where := fmt.Sprintf("a %s", placer)
	if placer == "" {
		where = "nothing — it is the root of the tree"
	}
	upsertConcern(ConcernInertPlacement, fmt.Sprintf(
		"%s states StackAlign %q and the container that would place it is %s: only an "+
			"overlay (%v) places its children in two dimensions, so the web exporter "+
			"imposes no placement here and neither native stack renderer ever reads the "+
			"value. The prop is dropped on all four targets with no diagnostic. Either "+
			"move the node into a ZStack, or say what was meant with the container's own "+
			"props (AlignItems, JustifyContent, Align)",
		path, n.Style.StackAlign, where, PlacingContainers()))
}

// placerFor returns the container type that places this node's children.
//
// A grouping container has no box, so it hands its children to whatever places
// *it*; anything else places its own children. This is the one rule that keeps
// a core.For inside a ZStack from reporting every layer it generated.
func placerFor(nodeType, inherited string) string {
	if groupsChildren(nodeType) {
		return inherited
	}
	return nodeType
}
