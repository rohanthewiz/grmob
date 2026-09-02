package core

import (
	"sync"
)

type Context struct {
	slots  []any
	Cursor int
	theme  *Theme
	config *AppConfig
	idGen  int
	lock   sync.Mutex
	parent *Context

	// The pointers below are the app-instance state that every context
	// derived from one NewContext root shares: derived contexts (children,
	// scopes, WithTheme/WithConfig copies) copy the pointers, so there is
	// exactly one of each per app. They used to be package-level globals,
	// which made two apps in one process (or two managers in one test binary)
	// share render notifications, callback IDs, and a navigation stack.
	//
	// cleanup is the one exception to "exactly one per app": a navigation
	// stack frame carries a sub-registry so its resources can be stopped when
	// the frame is popped (see cleanupRegistry). Every sub-registry is still
	// reachable from the app's root registry, so an app-wide Close reaches
	// everything.
	renderManager *RenderManager
	registry      *callbackRegistry
	nav           *navigatorState
	cleanup       *cleanupRegistry
	dirty         *dirtyFlag
	focus         *focusState

	// hooks is set only on the lightweight copies produced by WithTheme and
	// WithConfig, and points at the context that actually owns the hook
	// slots (never at another copy — hookOwner collapses chains as they are
	// built, so this is always at most one hop).
	//
	// Those copies exist to override theme/config for a subtree, not to open
	// a new hook scope: the children rendered through them are the *same*
	// component positions in the *same* parent scope. Copying `slots` and
	// `Cursor` by value made that untrue in the worst way — the copy read
	// and appended at the parent's cursor while the parent's own cursor
	// never advanced, so children on either side of a WithTheme aliased each
	// other's state:
	//
	//	Column(WithTheme(t, A{NewState(ctx, 0)}), B{NewState(ctx, "s")})
	//	  pass 1: A takes slot 0 on the copy; parent cursor still 0,
	//	          so B also takes slot 0 -> both share one slot
	//	  pass 2: A reads slot 0 as int, B reads the same slot as string
	//	          -> "interface conversion: interface {} is string, not int"
	//
	// Sharing the owner instead means slot indices are handed out by one
	// cursor under one lock, exactly as if WithTheme were not there.
	hooks *Context

	children       []*Context
	childrenCursor int
	scopes         map[string]*Context

	// Debug-mode bookkeeping for the cursor-drift check (see auditCursor):
	// the ending cursor of the last pass in which this context rendered
	// anything, and whether such a pass has happened yet. Only read/written
	// inside EndRenderPass, which runs under the render driver's pass
	// serialization, so no lock is needed.
	debugLastCursor int
	debugPassSeen   bool
}

// dirtyFlag is the app-wide "this tree has unrendered changes" bit, shared by
// pointer with every derived context for the same reason as the callback
// registry and the navigation stack.
//
// It used to be a plain bool on each Context, which quietly made the flag mean
// "this *one* context changed". Polling hosts read it from the root (WASM's
// IsDirty binding), so a state change anywhere below the root — every hook in
// a child context, every UseChildContext subtree, and now every navigation
// frame, which renders into a scope by construction — set a flag nobody ever
// looked at, and the screen simply stopped updating until an unrelated event
// forced a pass. Push-based hosts hid the bug because the render manager
// notification travels a separate, already-shared path.
//
// Its own mutex rather than Context.lock: the two guard unrelated things, and
// keeping them apart means marking the tree dirty never contends with hook
// slot access on a context that happens to be busy rendering.
type dirtyFlag struct {
	mu  sync.Mutex
	set bool
}

func newDirtyFlag() *dirtyFlag { return &dirtyFlag{} }

// MarkDirty records that the tree needs re-rendering, without notifying
// anyone. Callers that want a render to actually happen want RequestRender,
// which does this and nudges the render manager; MarkDirty alone is for paths
// where a pass is already guaranteed to follow.
func (ctx *Context) MarkDirty() {
	ctx.dirty.mu.Lock()
	defer ctx.dirty.mu.Unlock()
	ctx.dirty.set = true
}

// IsDirty reports whether the tree has changes no pass has consumed yet. It
// answers for the whole app, not for the context it is called on.
func (ctx *Context) IsDirty() bool {
	ctx.dirty.mu.Lock()
	defer ctx.dirty.mu.Unlock()
	return ctx.dirty.set
}

func (ctx *Context) ClearDirty() {
	ctx.dirty.mu.Lock()
	defer ctx.dirty.mu.Unlock()
	ctx.dirty.set = false
}

type AppConfig struct {
	Name        string
	Description string
	Version     string
	Locale      string
	Author      string
	Meta        map[string]string
}

func NewContext() *Context {
	return &Context{
		slots:         make([]any, 0),
		Cursor:        0,
		renderManager: NewRenderManager(),
		registry:      newCallbackRegistry(),
		nav:           newNavigatorState(),
		cleanup:       newCleanupRegistry(),
		dirty:         newDirtyFlag(),
		focus:         newFocusState(),
		scopes:        make(map[string]*Context),
	}
}
func (ctx *Context) NewChildContext() *Context {
	return &Context{
		slots:         make([]any, 0),
		Cursor:        0,
		theme:         ctx.theme,
		config:        ctx.config,
		renderManager: ctx.renderManager,
		registry:      ctx.registry,
		nav:           ctx.nav,
		cleanup:       ctx.cleanup,
		dirty:         ctx.dirty,
		focus:         ctx.focus,
		parent:        ctx,
		scopes:        make(map[string]*Context),
	}
}
func UseChildContext(ctx *Context) *Context {
	// Hook slots belong to the owner (see hookOwner); the child itself is
	// still built from ctx so it inherits the theme/config in effect here.
	owner := ctx.hookOwner()
	index := owner.Cursor
	owner.Cursor++

	// Same locking rationale as NewState: the append can reallocate the slots
	// backing array while a concurrent State.Set writes through it.
	owner.lock.Lock()
	defer owner.lock.Unlock()
	if index >= len(owner.slots) {
		owner.slots = append(owner.slots, ctx.NewChildContext())
	}
	return owner.slots[index].(*Context)
}

type State[T any] struct {
	get func() T
	set func(T)
}

func (s *State[T]) Get() T {
	return s.get()
}

func (s *State[T]) Set(val T) {
	s.set(val)
}

func (ctx *Context) Theme() *Theme {
	if ctx.theme != nil {
		return ctx.theme
	}
	return DefaultTheme // fallback
}

func (ctx *Context) Config() *AppConfig {
	if ctx.config == nil {
		return &AppConfig{}
	}
	return ctx.config
}

// hookOwner returns the context whose slots, cursor and lock back this
// context's hooks: itself for a real context, and the original for the
// theme/config copies described on the `hooks` field.
//
// Every slot-touching path (NewState, UseChildContext, Reset) and every
// reader of the cursor goes through this, so a themed copy is indistinguishable
// from its owner as far as hook bookkeeping is concerned.
func (ctx *Context) hookOwner() *Context {
	if ctx.hooks != nil {
		return ctx.hooks
	}
	return ctx
}

func (ctx *Context) WithConfig(cfg *AppConfig) *Context {
	return &Context{
		// Slots, Cursor and lock deliberately stay zero: hookOwner routes
		// every hook operation to the context named by `hooks` instead.
		hooks:         ctx.hookOwner(),
		theme:         ctx.theme,
		config:        cfg,
		renderManager: ctx.renderManager,
		registry:      ctx.registry,
		nav:           ctx.nav,
		cleanup:       ctx.cleanup,
		dirty:         ctx.dirty,
		focus:         ctx.focus,
		// Share the scope table rather than leaving it nil: this is the same
		// context wearing a different config, so a scope reached through it
		// must be the same scope reached through the original. A nil map here
		// also made ctx.Scope panic outright (assignment to a nil map) on
		// every context derived this way — which is the path render.New and
		// the WASM host take.
		scopes: ctx.scopes,
	}
}

func (ctx *Context) WithTheme(theme *Theme) *Context {
	return &Context{
		// See WithConfig: hook state is delegated, never copied.
		hooks:         ctx.hookOwner(),
		theme:         theme,
		config:        ctx.config,
		renderManager: ctx.renderManager,
		registry:      ctx.registry,
		nav:           ctx.nav,
		cleanup:       ctx.cleanup,
		dirty:         ctx.dirty,
		focus:         ctx.focus,
		// See WithConfig: the scope table is shared, not re-created, so
		// ctx.Scope works (and resolves to the same scopes) on a themed copy.
		scopes: ctx.scopes,
	}
}

// NewState allocates (or on re-render, re-binds) the hook slot at the current
// cursor position and returns typed accessors for it.
//
// Slot access is guarded by ctx.lock because reads and writes come from
// different goroutines: renders run on the manager/pump goroutine (or a native
// event thread), while Set may be called from timers, network handlers, or any
// goroutine the app spawns. Render passes themselves are serialized by
// render.Manager, so the lock's job is only to make individual slot accesses
// atomic against concurrent Sets — a Set landing mid-render yields a tree
// mixing old and new values for one pass, which is benign: the Set also
// nudges the pump, so a follow-up pass renders the settled state.
func NewState[T any](ctx *Context, initial T) State[T] {
	// All slot bookkeeping happens on the owner so that a themed/configured
	// copy of a context allocates out of the same numbering as its original
	// (see the `hooks` field). The closures below capture the owner too, so
	// a State outlives the transient copy it was created through.
	owner := ctx.hookOwner()
	index := owner.Cursor
	owner.Cursor++

	owner.lock.Lock()
	if index >= len(owner.slots) {
		// First render at this cursor position: seed the slot. The append is
		// under the lock because it can reallocate the backing array, which
		// must not race a concurrent Set writing through the old one.
		owner.slots = append(owner.slots, initial)
	}
	owner.lock.Unlock()

	return State[T]{
		get: func() T {
			owner.lock.Lock()
			defer owner.lock.Unlock()
			return owner.slots[index].(T)
		},
		set: func(val T) {
			owner.lock.Lock()
			owner.slots[index] = val
			// Unlock before notifying: RequestRender -> MarkDirty takes the
			// same (non-reentrant) lock, and holding it across the notify
			// would also serialize slot access behind render scheduling.
			owner.lock.Unlock()
			// RequestRender rather than a bare TriggerRender: it also marks
			// the tree dirty, so polling runtimes (WASM IsDirty) and the push
			// channel observe the same signal. Sent through the owner so
			// a State created under a themed copy still nudges the app.
			owner.RequestRender()
		},
	}
}

func (ctx *Context) With(opts ...func(*Context)) *Context {
	for _, fn := range opts {
		fn(ctx)
	}
	return ctx
}

func WithThemeOpt(t *Theme) func(*Context) {
	return func(ctx *Context) {
		ctx.theme = t
	}
}

func WithConfigOpt(c *AppConfig) func(*Context) {
	return func(ctx *Context) {
		ctx.config = c
	}
}

func (ctx *Context) Reset() {
	// Reset the context that actually holds the slots: calling Reset on a
	// themed/configured copy must rewind the original, not the copy's empty
	// (and unused) slot list.
	ctx = ctx.hookOwner()
	ctx.Cursor = 0
	for _, child := range ctx.children {
		child.Reset()
	}
	// Snapshot the slots under the lock before scanning for child contexts:
	// iterating the live slice would read each element unsynchronized against
	// a concurrent State.Set overwriting it. Child contexts themselves are
	// only ever appended (never replaced by Set), so recursing outside the
	// lock on the copied values is safe — and necessary, since the children
	// take their own locks during Reset.
	ctx.lock.Lock()
	slots := make([]any, len(ctx.slots))
	copy(slots, ctx.slots)
	ctx.lock.Unlock()
	for _, child := range slots {
		if c, ok := child.(*Context); ok {
			c.Reset()
		}
	}
	for _, scope := range ctx.scopes {
		scope.Reset()
	}
}

// Scope returns a stable child context under key, creating it on first use.
// The child owns its own hook slots, which is what lets a subtree be rendered
// conditionally — a tab that is only drawn when selected, a navigation frame —
// without shifting the positional slots of everything around it.
//
// # Theme and config are re-inherited on every call
//
// A child copies its parent's theme and config when it is built, and a scope
// is then cached for the life of the app. So a theme that changes *after* the
// scope's first render — the common shape for an app whose palette arrives
// from the network — would otherwise never reach anything inside it, while
// everything outside repainted. Nothing in the tree could explain the
// difference, because the scope is invisible at the call site.
//
// Refreshing here is cheap (two pointer assignments) and safe: Scope is called
// during a render pass, which render.Manager serializes, and the theme is only
// ever read during a pass.
//
// It is also the correct semantics. theme and config are *inherited* state, not
// state the scope owns; hook slots are what the scope owns, and those are
// deliberately left alone.
func (ctx *Context) Scope(key string) *Context {
	if child, ok := ctx.scopes[key]; ok {
		child.theme, child.config = ctx.theme, ctx.config
		return child
	}
	child := ctx.NewChildContext()
	ctx.scopes[key] = child
	return child
}

// disposableScope is Scope for a subtree that is expected to be thrown away
// before the app is: the scope it creates carries its own cleanup registry
// (nested under this context's), so dropScope can stop that subtree's
// background resources without touching the rest of the app.
//
// Only the navigation stack uses this today — one disposable scope per stack
// frame. It stays unexported because handing apps a scope they can silently
// leak is worse than making them ask for the one lifetime the framework
// actually manages; ordinary Scope shares the app registry and is right for
// everything that lives as long as the app does.
func (ctx *Context) disposableScope(key string) *Context {
	if child, ok := ctx.scopes[key]; ok {
		// Re-inherit theme and config, for the reason Scope documents: a
		// navigation frame is created once and rendered for as long as it is
		// on the stack, so a later theme change has to reach it.
		child.theme, child.config = ctx.theme, ctx.config
		return child
	}
	child := ctx.NewChildContext()
	child.cleanup = ctx.cleanup.sub()
	ctx.scopes[key] = child
	return child
}

// dropScope forgets a named scope and stops the background resources
// registered under it. The scope's hook slots go with it, so whatever state
// lived there is genuinely gone rather than lying in wait for the next
// component that happens to claim the same key.
//
// Callers must be on the render goroutine. ctx.scopes is a plain map read
// during every pass by Reset, auditCursor and Scope itself, none of which
// lock; deleting from an event handler would race all three. The navigation
// stack therefore records what to drop when the mutation happens and does the
// dropping on the next pass — see navigatorState.retired.
//
// Detaching matters as much as closing: a closed-but-still-linked
// sub-registry would be re-closed by every later app-wide Close and would
// accumulate one dead entry per dropped scope.
func (ctx *Context) dropScope(key string) {
	child, ok := ctx.scopes[key]
	if !ok {
		return
	}
	delete(ctx.scopes, key)
	child.cleanup.close()
	child.cleanup.detach()
}
