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
	dirty  bool
	parent *Context

	// The three pointers below are the app-instance state that every context
	// derived from one NewContext root shares: derived contexts (children,
	// scopes, WithTheme/WithConfig copies) copy the pointers, so there is
	// exactly one of each per app. They used to be package-level globals,
	// which made two apps in one process (or two managers in one test binary)
	// share render notifications, callback IDs, and a navigation stack.
	renderManager *RenderManager
	registry      *callbackRegistry
	nav           *navigatorState
	cleanup       *cleanupRegistry

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

func (ctx *Context) MarkDirty() {
	ctx.lock.Lock()
	defer ctx.lock.Unlock()
	ctx.dirty = true
}

func (ctx *Context) IsDirty() bool {
	ctx.lock.Lock()
	defer ctx.lock.Unlock()
	return ctx.dirty
}

func (ctx *Context) ClearDirty() {
	ctx.lock.Lock()
	defer ctx.lock.Unlock()
	ctx.dirty = false
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
