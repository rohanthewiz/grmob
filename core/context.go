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
	index := ctx.Cursor
	ctx.Cursor++

	// Same locking rationale as NewState: the append can reallocate the slots
	// backing array while a concurrent State.Set writes through it.
	ctx.lock.Lock()
	defer ctx.lock.Unlock()
	if index >= len(ctx.slots) {
		ctx.slots = append(ctx.slots, ctx.NewChildContext())
	}
	return ctx.slots[index].(*Context)
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

func (ctx *Context) WithConfig(cfg *AppConfig) *Context {
	return &Context{
		slots:         ctx.slots,
		Cursor:        ctx.Cursor,
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
		slots:         ctx.slots,
		Cursor:        ctx.Cursor,
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
	index := ctx.Cursor
	ctx.Cursor++

	ctx.lock.Lock()
	if index >= len(ctx.slots) {
		// First render at this cursor position: seed the slot. The append is
		// under the lock because it can reallocate the backing array, which
		// must not race a concurrent Set writing through the old one.
		ctx.slots = append(ctx.slots, initial)
	}
	ctx.lock.Unlock()

	return State[T]{
		get: func() T {
			ctx.lock.Lock()
			defer ctx.lock.Unlock()
			return ctx.slots[index].(T)
		},
		set: func(val T) {
			ctx.lock.Lock()
			ctx.slots[index] = val
			// Unlock before notifying: RequestRender -> MarkDirty takes the
			// same (non-reentrant) lock, and holding it across the notify
			// would also serialize slot access behind render scheduling.
			ctx.lock.Unlock()
			// RequestRender rather than a bare TriggerRender: it also marks
			// the tree dirty, so polling runtimes (WASM IsDirty) and the push
			// channel observe the same signal.
			ctx.RequestRender()
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

func (ctx *Context) Scope(key string) *Context {
	if child, ok := ctx.scopes[key]; ok {
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
