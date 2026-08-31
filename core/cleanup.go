package core

import "sync"

// cleanupRegistry collects stop functions for background resources started on
// behalf of a context tree — interval tickers, pending timeouts, anything a
// hook spins up that outlives the render pass that created it. One registry
// exists per NewContext root and is shared by pointer with every derived
// context (children, scopes, WithTheme/WithConfig copies), the same pattern
// as renderManager and the callback registry: resources registered anywhere
// in the tree are stopped together, and two apps in one process can never
// stop each other's.
//
// Registries nest. A sub-registry covers a slice of the context tree whose
// lifetime is shorter than the app's — today that means one navigation stack
// frame (see navigation.go). Closing a parent closes its children, so the
// app-wide Close still stops everything; closing and detaching a child stops
// only that frame's resources and unlinks it, which is what makes a popped
// route's ticker actually stop instead of firing into a screen that no longer
// exists.
//
//	root registry (app)
//	 ├─ frame 0 registry   ← Pop/Reset closes + detaches one of these
//	 └─ frame 1 registry
//
// Close has drain semantics rather than terminal semantics: it runs and
// forgets the functions registered so far, but the registry stays usable.
// That is deliberate — a host that re-mounts an app over the same context
// (the WASM runtime re-invoking RenderInitial, a hot-reload harness) closes
// the old manager and renders again, and the re-render's hooks must be able
// to register fresh resources for the next Close to stop.
type cleanupRegistry struct {
	mu  sync.Mutex
	fns []func()

	// parent is nil for the app root. Only detach reads it, and it is written
	// once at construction, so it needs no synchronization of its own.
	parent   *cleanupRegistry
	children []*cleanupRegistry
}

func newCleanupRegistry() *cleanupRegistry {
	return &cleanupRegistry{}
}

// sub creates a nested registry whose resources can be stopped independently
// of the rest of the app, and links it into this one so an app-wide Close
// still reaches it.
func (r *cleanupRegistry) sub() *cleanupRegistry {
	child := &cleanupRegistry{parent: r}
	r.mu.Lock()
	r.children = append(r.children, child)
	r.mu.Unlock()
	return child
}

// close drains this registry and every registry nested under it.
//
// Children run first, innermost outward: a nested registry's resources were
// started by components living inside this one's subtree, so they are the
// resources that should stop first. The registered functions run outside the
// registry lock — a cleanup that itself registers or closes (however
// unlikely) must not deadlock, mirroring the dispatch-outside-the-lock rule
// the callback registry follows. Nothing here holds two locks at once: the
// snapshot is taken and released before any child is touched.
func (r *cleanupRegistry) close() {
	r.mu.Lock()
	fns := r.fns
	children := r.children
	r.fns = nil
	r.children = nil
	r.mu.Unlock()

	for _, child := range children {
		child.close()
	}
	for _, fn := range fns {
		fn()
	}
}

// detach unlinks r from its parent so a closed sub-registry is not retained
// (and re-closed) by the next app-wide Close. Without it, an app that pushes
// and pops screens all day would grow one dead registry per pop.
func (r *cleanupRegistry) detach() {
	p := r.parent
	if p == nil {
		return
	}
	p.mu.Lock()
	for i, child := range p.children {
		if child == r {
			p.children = append(p.children[:i], p.children[i+1:]...)
			break
		}
	}
	p.mu.Unlock()
	r.parent = nil
}

// OnClose registers fn to run when this context tree is closed. Hooks use it
// to hand ownership of their background resources to whoever drives the app's
// lifecycle (normally render.Manager, whose Close closes its context).
//
// "This context tree" is the registry the context carries, which for most
// contexts is the app-wide one. A context inside a navigation stack frame
// carries that frame's registry instead, so its resources also stop when the
// frame leaves the stack — earlier than the app's own shutdown.
func (ctx *Context) OnClose(fn func()) {
	r := ctx.cleanup
	r.mu.Lock()
	r.fns = append(r.fns, fn)
	r.mu.Unlock()
}

// Close stops every background resource registered on this context tree since
// the last Close (see the drain semantics on cleanupRegistry). The tree itself
// remains renderable afterwards; a subsequent render pass simply re-registers
// whatever resources it still needs.
func (ctx *Context) Close() {
	ctx.cleanup.close()
}
