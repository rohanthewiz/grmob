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
	Type string
	// Everything but Type is omitted from JSON when it is at its zero value,
	// for the reason core.Style's fields are — see the note above that struct.
	// Type is not, because a node without one is not a node, and a renderer
	// reading an absent Type would fall through its dispatch to whatever its
	// default arm is rather than say what is wrong.
	Key      string         `json:",omitzero"`
	Props    map[string]any `json:",omitzero"`
	Style    *Style         `json:",omitzero"`
	Children []*Node        `json:",omitzero"`
}
