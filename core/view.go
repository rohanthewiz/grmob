package core

// View is anything that can be put on screen: Render returns the immutable
// Node tree for one render pass. Every argument a container takes as a child,
// every slot a widget offers, and the root an app registers is a View.
//
// # Vocabulary
//
// GrMob uses four words for the things that implement View, and the docs use
// them in exactly these senses:
//
//	View        the interface: Render(ctx) *Node. Everything below is one.
//	Primitive   a core constructor whose Node type a host draws itself:
//	            Text, Row, Button, TextInput, Canvas, … Adding one touches
//	            every host (Compose, SwiftUI, the wasm runtime, htmlout).
//	Component   a View written in Go entirely out of other Views — no new
//	            node type, no host code. An app's screens are components.
//	Widget      a component built for reuse: a struct with named fields that
//	            keeps the component contract below. Every comps type is one.
//
// # What a component is
//
// A component takes one of three shapes, and all three are Views:
//
//	func row(t Todo) View                       a function returning a View;
//	                                            no hooks, so safe in For / If
//	ComponentFunc(func(ctx) *Node { … })        a render function as a value
//	type Tally struct{ … }; (Tally) Render(…)   a struct: named fields, slots
//
// Composition is plain Go: a component calls other components and returns
// what they render. There is no registration, no lifecycle interface and no
// per-platform code; a component needs none, because it bottoms out in
// primitives that every host already draws.
//
// # The component contract
//
// Rules 1–3 are correctness: break one and the app misbehaves (stale
// patches, drifting state, a panic on a native tap), so every component,
// screens included, keeps them. Rules 4–6 make a component reusable, and
// every widget keeps them too.
//
//  1. Render builds a fresh tree from core's constructors and other Views,
//     and never mutates a Node after returning it.
//  2. Hooks take the caller's positional slots, so a component either uses
//     none (and may be rendered conditionally) or takes all of them before
//     any branch and is rendered every pass. A widget owns state only when
//     that state is purely about its own presentation; anything an app would
//     read, drive or persist arrives as a value plus an OnChange.
//  3. It never registers a nil callback, and calls the caller's callbacks
//     nil-safely.
//  4. Its look comes from ctx.Theme(), never a literal colour or size, and a
//     caller's Style props are applied after its own, so the caller's win.
//  5. It states its role, name and state on its nodes, and reports misuse
//     with ReportConcern under debug mode rather than panicking.
//  6. Every field's zero value means something sensible, because Go has no
//     "unset".
//
// A component that cannot be written under these rules — one that needs a
// node type, a Style field or a role core lacks — is a gap in core's
// primitives, not a component. docs/concepts/components.md walks through
// each rule with a worked example.
type View interface {
	Render(ctx *Context) *Node
}

// ComponentFunc adapts a render function to a View, the way
// http.HandlerFunc adapts a function to a Handler. Most of core's own
// constructors are ComponentFuncs closing over their arguments. A struct
// with a Render method is a View already and needs no adapter.
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
