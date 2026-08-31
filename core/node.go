package core

// Node is the retained render-tree element the reconciler diffs.
//
// Immutability contract: a Node is frozen once its render pass returns it.
// Builders may assemble a node freely while constructing it (Keyed sets Key,
// containerNode applies behavior props), but after render nothing may write
// to it — the reconciler only reads, and renderers must also only read. The
// contract is what makes sharing safe: Cached returns the same *Node every
// pass and Diff treats pointer equality as proof the subtree is unchanged, so
// a post-render mutation would silently never reach the screen.
type Node struct {
	Type     string
	Key      string
	Props    map[string]any
	Style    *Style
	Children []*Node
}
