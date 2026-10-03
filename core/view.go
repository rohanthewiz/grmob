package core

type View interface {
	Render(ctx *Context) *Node
}

type ComponentFunc func(ctx *Context) *Node

func (f ComponentFunc) Render(ctx *Context) *Node {
	return f(ctx)
}

// Keyed gives child's root node an identity, key, which outlives its position
// among its siblings.
//
// The key does two jobs. The reconciler replaces a keyed slot whose key
// changed rather than patching one node into another. And the subtree's
// callback IDs are named by the key (Context.keyScope): a sibling that grows
// or shrinks its handler count no longer renumbers this subtree's handlers,
// and this subtree no longer renumbers anyone else's.
func Keyed(key string, child View) View {
	return ComponentFunc(func(ctx *Context) *Node {
		n := ctx.keyScope(key, func() *Node { return child.Render(ctx) })
		n.Key = key
		return n
	})
}
