# Package core — Views & state

```go
import "github.com/rohanthewiz/grmob/core"
```

View, Node, Context and state slots; conditionals, caching, error boundaries and debug-mode concerns.

One of 11 topic pages of [package core](core.md), which has the package overview and an index of every topic. This page documents the declarations in `core/view.go`, `core/node.go`, `core/text.go`, `core/context.go`, `core/cleanup.go`, `core/cached.go`, `core/conditionals.go`, `core/error_boundary.go`, `core/render_manager.go`, `core/debug.go`.

## Index

- [Constants](#constants) — `ConcernCachedCallbacks`, `ConcernCachedHooks`, `ConcernCursorDrift`, `ConcernDuplicateKey`, `ConcernHandlerPanic`, `ConcernRenderPanic`, `ConcernUnknownItem`
- [`func ClearConcerns`](#func-clearconcerns)
- [`func DumpConcerns`](#func-dumpconcerns)
- [`func IsDebugMode`](#func-isdebugmode)
- [`func MaybeProp`](#func-maybeprop)
- [`func ReportConcern`](#func-reportconcern)
- [`func SetDebugMode`](#func-setdebugmode)
- [`func WithConfigOpt`](#func-withconfigopt)
- [`func WithThemeOpt`](#func-withthemeopt)
- [`type AppConfig`](#type-appconfig)
- [`type ComponentFunc`](#type-componentfunc)
    - [`func (ComponentFunc) Render`](#func-componentfunc-render)
- [`type Concern`](#type-concern)
    - [`func Concerns`](#func-concerns)
- [`type Context`](#type-context)
    - [`func NewContext`](#func-newcontext)
    - [`func UseChildContext`](#func-usechildcontext)
    - [`func (*Context) BeginRenderPass`](#func-context-beginrenderpass)
    - [`func (*Context) ClearDirty`](#func-context-cleardirty)
    - [`func (*Context) Close`](#func-context-close)
    - [`func (*Context) Config`](#func-context-config)
    - [`func (*Context) EndRenderPass`](#func-context-endrenderpass)
    - [`func (*Context) IsDirty`](#func-context-isdirty)
    - [`func (*Context) MarkDirty`](#func-context-markdirty)
    - [`func (*Context) NewChildContext`](#func-context-newchildcontext)
    - [`func (*Context) OnClose`](#func-context-onclose)
    - [`func (*Context) OnStateChange`](#func-context-onstatechange)
    - [`func (*Context) PurgeUnusedCallbacks`](#func-context-purgeunusedcallbacks)
    - [`func (*Context) ReceiveEventPayload`](#func-context-receiveeventpayload)
    - [`func (*Context) RequestRender`](#func-context-requestrender)
    - [`func (*Context) Reset`](#func-context-reset)
    - [`func (*Context) Scope`](#func-context-scope)
    - [`func (*Context) Theme`](#func-context-theme)
    - [`func (*Context) TriggerBoolCallback`](#func-context-triggerboolcallback)
    - [`func (*Context) TriggerCallback`](#func-context-triggercallback)
    - [`func (*Context) TriggerIntCallback`](#func-context-triggerintcallback)
    - [`func (*Context) TriggerTextCallback`](#func-context-triggertextcallback)
    - [`func (*Context) TriggerTextEdit`](#func-context-triggertextedit)
    - [`func (*Context) With`](#func-context-with)
    - [`func (*Context) WithConfig`](#func-context-withconfig)
    - [`func (*Context) WithTheme`](#func-context-withtheme)
- [`type MatchCase`](#type-matchcase)
    - [`func Case`](#func-case)
    - [`func Default`](#func-default)
- [`type Node`](#type-node)
- [`type RenderError`](#type-rendererror)
    - [`func Guard`](#func-guard)
    - [`func (*RenderError) Error`](#func-rendererror-error)
    - [`func (*RenderError) Unwrap`](#func-rendererror-unwrap)
- [`type RenderManager`](#type-rendermanager)
    - [`func NewRenderManager`](#func-newrendermanager)
    - [`func (*RenderManager) TriggerRender`](#func-rendermanager-triggerrender)
- [`type State`](#type-state)
    - [`func NewState`](#func-newstate)
    - [`func (*State) Get`](#func-state-get)
    - [`func (*State) Set`](#func-state-set)
- [`type View`](#type-view)
    - [`func Cached`](#func-cached)
    - [`func DefaultErrorFallback`](#func-defaulterrorfallback)
    - [`func ErrorBoundary`](#func-errorboundary)
    - [`func For`](#func-for)
    - [`func If`](#func-if)
    - [`func IfElse`](#func-ifelse)
    - [`func Keyed`](#func-keyed)
    - [`func Match`](#func-match)
    - [`func MatchBool`](#func-matchbool)
    - [`func SafeRender`](#func-saferender)
    - [`func Text`](#func-text)
- [`type WhenClause`](#type-whenclause)
    - [`func Otherwise`](#func-otherwise)
    - [`func When`](#func-when)

## Constants

Concern kinds. Each names one class of silent bug the debug checks detect.

```go
const (
	// ConcernCursorDrift: a context's hook cursor ended a pass out of step
	// with its slot count or with the previous pass — some NewState /
	// UseChildContext call is conditional or loop-varying, so later slots
	// are (or will be) read by the wrong component.
	ConcernCursorDrift = "cursor-drift"

	// ConcernDuplicateKey: two siblings in one container carry the same
	// non-empty Key, defeating keyed reconciliation for that sibling list.
	ConcernDuplicateKey = "duplicate-key"

	// ConcernCachedHooks: a Cached view consumed hook slots during its
	// render. In production the view renders once and never again, so those
	// slots vanish on later passes and shift every component after it.
	ConcernCachedHooks = "cached-hooks"

	// ConcernCachedCallbacks: a Cached view registered event callbacks. In
	// production its handlers are purged after the first pass it skips, and
	// the un-consumed counter slots shift the callback IDs of everything
	// registered after it.
	ConcernCachedCallbacks = "cached-callbacks"

	// ConcernUnknownItem: a container (Row, Column, Card, Box, List) was
	// handed an argument that is neither a StyleProp, a BehaviorProp nor a
	// View. PropsAndChildren is an alias for any, so the compiler accepts
	// anything and containerNode drops what it cannot classify — the symptom
	// is a style or handler that simply never took effect. An untyped nil is
	// exempt: that is MaybeProp's false path, not a mistake.
	ConcernUnknownItem = "unknown-container-item"

	// ConcernRenderPanic: an ErrorBoundary caught a panic escaping a
	// component's Render and swapped in its fallback. The app kept running —
	// that is the boundary doing its job — which is exactly why this needs
	// reporting: a boundary placed high in the tree can hide a component that
	// has been dead for weeks behind a plausible-looking "unavailable" panel.
	// The detail carries the panic value; the full stack goes to the
	// fallback, not here.
	ConcernRenderPanic = "render-panic"

	// ConcernHandlerPanic: an event handler panicked and the render driver
	// recovered it. Distinct from ConcernRenderPanic because the failure is
	// in a different phase with a different blast radius: a render panic
	// costs a subtree's frame, while a handler panic abandons the handler
	// partway, so the app's state may be half-updated in a way no fallback
	// can describe.
	ConcernHandlerPanic = "handler-panic"
)
```

<small>[core/debug.go:40](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L40)</small>

## Functions

### func ClearConcerns

```go
func ClearConcerns()
```

ClearConcerns drops all recorded concerns. Tests call it between cases; apps can call it after acting on a dump.

<small>[core/debug.go:161](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L161)</small>

### func DumpConcerns

```go
func DumpConcerns() string
```

DumpConcerns renders the recorded concerns as a human-readable block, one line per finding. Empty string when there is nothing to report.

<small>[core/debug.go:169](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L169)</small>

### func IsDebugMode

```go
func IsDebugMode() bool
```

IsDebugMode reports whether debug checks are active.

<small>[core/debug.go:35](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L35)</small>

### func MaybeProp

```go
func MaybeProp(cond bool, prop PropsAndChildren) PropsAndChildren
```

MaybeProp conditionally contributes one item to a container's argument list: Row, Column, Card, Box and List, the variadic ...PropsAndChildren builders. It returns prop when cond holds and an untyped nil otherwise, and containerNode skips a nil item, so a false condition costs the tree nothing at all — no node, no slot, no style.

It exists because core.If cannot do this job, in two separate ways:

 1. If(false, view) returns Fragment(), and an empty Fragment is still a real child node: the reconciler walks and diffs it on every pass, and it occupies a child index, so anything addressing children by position counts it. If earns its place where the alternative is a whole branch of the tree; it is the wrong tool for one optional item in a row of three.

    It does not, however, draw anything. A grouping node with no children renders no box on any of the three targets — that is what both native renderers have always done, and what the HTML exporter now does too. An earlier version of this note claimed the empty Fragment took a flex slot and opened a stray Gap; that was true only of the exporter, which wrapped every grouping node in a div, and it is fixed. The cost is a node, not a gap.

 2. If is typed View -> View. There is no If for a StyleProp or a BehaviorProp, so "apply this padding only when selected" or "attach OnClick only when a handler was supplied" had no expression form at all. MaybeProp takes PropsAndChildren, so it covers all three item kinds with one helper.

Together those replace the accumulate-into-a-slice idiom this codebase kept reaching for:

	items := make([]core.PropsAndChildren, 0, 3)
	items = append(items, core.UseStyle(bubble))
	if !mine {
	    items = append(items, core.Text(from))
	}
	items = append(items, core.Text(body))
	return core.Column(items...)

	// becomes
	return core.Column(
	    core.UseStyle(bubble),
	    core.MaybeProp(!mine, core.Text(from)),
	    core.Text(body),
	)

Two limits, both deliberate:

prop is evaluated eagerly, like any Go argument — the condition does not guard it. That is safe for the prop constructors, which only build values (core.Text returns a closure; nothing renders until the container renders it), but MaybeProp is not a substitute for an if statement around an expression that would panic or do real work on the false path.

The return type is PropsAndChildren (i.e. any), so this is only valid in the container builders' argument lists. Text and Button take typed variadics (...StyleProp), which will not accept it — and must not, since their loops call Apply on every element and would panic on a nil.

<small>[core/conditionals.go:135](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L135)</small>

### func ReportConcern

```go
func ReportConcern(kind, detail string)
```

ReportConcern records a concern from outside this package.

The detection sites for most concern kinds live in core, so they call upsertConcern directly; the panic guards in the render driver do not, and a recovered panic is exactly the sort of silent-by-design event the concern list exists to surface. Callers should gate on IsDebugMode themselves — detail strings usually cost a Sprintf to build, and there is no reason to pay for one in a release build.

Deduplicated on kind+detail like every other concern, so a failure that repeats every frame occupies one entry with a rising count.

<small>[core/debug.go:137](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L137)</small>

### func SetDebugMode

```go
func SetDebugMode(on bool)
```

SetDebugMode turns the debug checks on or off. Zero overhead when off: every check site guards with IsDebugMode before doing any work.

<small>[core/debug.go:30](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L30)</small>

### func WithConfigOpt

```go
func WithConfigOpt(c *AppConfig) func(*Context)
```

<small>[core/context.go:326](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L326)</small>

### func WithThemeOpt

```go
func WithThemeOpt(t *Theme) func(*Context)
```

<small>[core/context.go:320](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L320)</small>

## Types

### type AppConfig

```go
type AppConfig struct {
	Name        string
	Description string
	Version     string
	Locale      string
	Author      string
	Meta        map[string]string
}
```

<small>[core/context.go:121](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L121)</small>

### type ComponentFunc

```go
type ComponentFunc func(ctx *Context) *Node
```

<small>[core/view.go:7](https://github.com/rohanthewiz/grmob/blob/master/core/view.go#L7)</small>

#### func (ComponentFunc) Render

```go
func (f ComponentFunc) Render(ctx *Context) *Node
```

<small>[core/view.go:9](https://github.com/rohanthewiz/grmob/blob/master/core/view.go#L9)</small>

### type Concern

```go
type Concern struct {
	Kind   string
	Detail string
	Count  int
}
```

Concern is one detected issue. Kind is one of the Concern\* constants; Detail is human-readable specifics; Count is how many times this exact (Kind, Detail) pair fired — checks run every pass, so a persistent bug increments its count rather than flooding the collector with duplicates.

<small>[core/debug.go:92](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L92)</small>

#### func Concerns

```go
func Concerns() []Concern
```

Concerns returns a snapshot of all recorded concerns, sorted by kind then detail so test assertions and dumps are deterministic.

<small>[core/debug.go:143](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L143)</small>

### type Context

```go
type Context struct {
	Cursor int
	// contains filtered or unexported fields
}
```

<small>[core/context.go:7](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L7)</small>

#### func NewContext

```go
func NewContext() *Context
```

<small>[core/context.go:130](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L130)</small>

#### func UseChildContext

```go
func UseChildContext(ctx *Context) *Context
```

<small>[core/context.go:161](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L161)</small>

#### func (*Context) BeginRenderPass

```go
func (ctx *Context) BeginRenderPass()
```

BeginRenderPass starts a callback ID pass for this context tree; see callbackRegistry.beginPass for the stability contract.

<small>[core/event.go:332](https://github.com/rohanthewiz/grmob/blob/master/core/event.go#L332)</small>

#### func (*Context) ClearDirty

```go
func (ctx *Context) ClearDirty()
```

<small>[core/context.go:115](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L115)</small>

#### func (*Context) Close

```go
func (ctx *Context) Close()
```

Close stops every background resource registered on this context tree since the last Close (see the drain semantics on cleanupRegistry). The tree itself remains renderable afterwards; a subsequent render pass simply re-registers whatever resources it still needs.

<small>[core/cleanup.go:120](https://github.com/rohanthewiz/grmob/blob/master/core/cleanup.go#L120)</small>

#### func (*Context) Config

```go
func (ctx *Context) Config() *AppConfig
```

<small>[core/context.go:198](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L198)</small>

#### func (*Context) EndRenderPass

```go
func (ctx *Context) EndRenderPass()
```

EndRenderPass closes a render pass for debug purposes: in debug mode it walks the context tree and flags any context whose hook usage this pass is inconsistent. Hosts that drive passes through render.Manager get this for free (the manager calls it after each pass); hand-rolled pass loops should call it after rendering, paired with BeginRenderPass/Reset before.

The pairing with the rest of the pass boundary:

	BeginRenderPass()  — callback ID counters restart
	Reset()            — hook cursors restart
	root.Render(ctx)   — components consume slots, cursor advances
	EndRenderPass()    — cursors audited against slots + previous pass  ← here

A no-op (single atomic load) when debug mode is off.

<small>[core/debug.go:198](https://github.com/rohanthewiz/grmob/blob/master/core/debug.go#L198)</small>

#### func (*Context) IsDirty

```go
func (ctx *Context) IsDirty() bool
```

IsDirty reports whether the tree has changes no pass has consumed yet. It answers for the whole app, not for the context it is called on.

<small>[core/context.go:109](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L109)</small>

#### func (*Context) MarkDirty

```go
func (ctx *Context) MarkDirty()
```

MarkDirty records that the tree needs re-rendering, without notifying anyone. Callers that want a render to actually happen want RequestRender, which does this and nudges the render manager; MarkDirty alone is for paths where a pass is already guaranteed to follow.

<small>[core/context.go:101](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L101)</small>

#### func (*Context) NewChildContext

```go
func (ctx *Context) NewChildContext() *Context
```

<small>[core/context.go:144](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L144)</small>

#### func (*Context) OnClose

```go
func (ctx *Context) OnClose(fn func())
```

OnClose registers fn to run when this context tree is closed. Hooks use it to hand ownership of their background resources to whoever drives the app's lifecycle (normally render.Manager, whose Close closes its context).

"This context tree" is the registry the context carries, which for most contexts is the app-wide one. A context inside a navigation stack frame carries that frame's registry instead, so its resources also stop when the frame leaves the stack — earlier than the app's own shutdown.

<small>[core/cleanup.go:109](https://github.com/rohanthewiz/grmob/blob/master/core/cleanup.go#L109)</small>

#### func (*Context) OnStateChange

```go
func (ctx *Context) OnStateChange(fn func())
```

OnStateChange registers fn to run whenever state anywhere in this context tree is written (State.Set, or anything else calling RequestRender). Only one handler is held: a render driver like render.Manager owns re-rendering for the whole app, so later registrations replace earlier ones rather than fanning out duplicate render passes.

fn is invoked on a fresh goroutine per notification (see TriggerRender), so it must be safe to call concurrently and should be cheap — the intended pattern is a non-blocking nudge into a coalescing channel, not a render.

<small>[core/render_manager.go:52](https://github.com/rohanthewiz/grmob/blob/master/core/render_manager.go#L52)</small>

#### func (*Context) PurgeUnusedCallbacks

```go
func (ctx *Context) PurgeUnusedCallbacks()
```

PurgeUnusedCallbacks drops handlers not re-registered in the current pass; see callbackRegistry.purge.

OnEndReached's debounce ledger is trimmed in the same breath and against the registry's own survivors, so the two can never disagree about which lists are still on screen — a guard outliving its handler would silently suppress the first page fetch of whatever list next inherits the ID.

<small>[core/event.go:343](https://github.com/rohanthewiz/grmob/blob/master/core/event.go#L343)</small>

#### func (*Context) ReceiveEventPayload

```go
func (ctx *Context) ReceiveEventPayload(payload map[string]any)
```

ReceiveEventPayload dispatches a loosely typed event envelope ({"callback": id, "value": ...}) by sniffing the value's type — the shape the WASM host sends. Typed hosts should call the Trigger\* methods directly.

<small>[core/event.go:381](https://github.com/rohanthewiz/grmob/blob/master/core/event.go#L381)</small>

#### func (*Context) RequestRender

```go
func (ctx *Context) RequestRender()
```

RequestRender marks the tree dirty and notifies the registered render driver. This is the one entry point for "state changed, the UI should re-render" — used by State.Set and by async sources such as timers, so changes that happen outside a native event (where no bridge call is pending a response) can still reach the screen via the push channel.

<small>[core/render_manager.go:63](https://github.com/rohanthewiz/grmob/blob/master/core/render_manager.go#L63)</small>

#### func (*Context) Reset

```go
func (ctx *Context) Reset()
```

<small>[core/context.go:332](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L332)</small>

#### func (*Context) Scope

```go
func (ctx *Context) Scope(key string) *Context
```

Scope returns a stable child context under key, creating it on first use. The child owns its own hook slots, which is what lets a subtree be rendered conditionally — a tab that is only drawn when selected, a navigation frame — without shifting the positional slots of everything around it.

##### Theme and config are re-inherited on every call

A child copies its parent's theme and config when it is built, and a scope is then cached for the life of the app. So a theme that changes \*after\* the scope's first render — the common shape for an app whose palette arrives from the network — would otherwise never reach anything inside it, while everything outside repainted. Nothing in the tree could explain the difference, because the scope is invisible at the call site.

Refreshing here is cheap (two pointer assignments) and safe: Scope is called during a render pass, which render.Manager serializes, and the theme is only ever read during a pass.

It is also the correct semantics. theme and config are \*inherited\* state, not state the scope owns; hook slots are what the scope owns, and those are deliberately left alone.

<small>[core/context.go:382](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L382)</small>

#### func (*Context) Theme

```go
func (ctx *Context) Theme() *Theme
```

<small>[core/context.go:191](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L191)</small>

#### func (*Context) TriggerBoolCallback

```go
func (ctx *Context) TriggerBoolCallback(id string, val bool)
```

TriggerBoolCallback dispatches a bool-carrying event (e.g. a toggle).

<small>[core/event.go:365](https://github.com/rohanthewiz/grmob/blob/master/core/event.go#L365)</small>

#### func (*Context) TriggerCallback

```go
func (ctx *Context) TriggerCallback(id string)
```

TriggerCallback dispatches a void event (e.g. a button tap) by callback ID. Unknown IDs are silent no-ops: a late native event racing a purge is expected traffic, not an error.

<small>[core/event.go:351](https://github.com/rohanthewiz/grmob/blob/master/core/event.go#L351)</small>

#### func (*Context) TriggerIntCallback

```go
func (ctx *Context) TriggerIntCallback(id string, val int)
```

TriggerIntCallback dispatches an int-carrying event (e.g. tab selection).

<small>[core/event.go:372](https://github.com/rohanthewiz/grmob/blob/master/core/event.go#L372)</small>

#### func (*Context) TriggerTextCallback

```go
func (ctx *Context) TriggerTextCallback(id string, val string)
```

TriggerTextCallback dispatches a string-carrying event (e.g. input change).

<small>[core/event.go:358](https://github.com/rohanthewiz/grmob/blob/master/core/event.go#L358)</small>

#### func (*Context) TriggerTextEdit

```go
func (ctx *Context) TriggerTextEdit(id, val string, seq, epoch int)
```

TriggerTextEdit dispatches one keystroke's worth of text from a native field: TriggerTextCallback plus the sequence and epoch the host stamped on it. An edit typed before the host had seen Go's latest rewrite of the field is acknowledged and dropped rather than handed to the app. See the file comment.

Unknown IDs are silent no-ops, as for every Trigger\* method.

<small>[core/text_edit.go:281](https://github.com/rohanthewiz/grmob/blob/master/core/text_edit.go#L281)</small>

#### func (*Context) With

```go
func (ctx *Context) With(opts ...func(*Context)) *Context
```

<small>[core/context.go:313](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L313)</small>

#### func (*Context) WithConfig

```go
func (ctx *Context) WithConfig(cfg *AppConfig) *Context
```

<small>[core/context.go:219](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L219)</small>

#### func (*Context) WithTheme

```go
func (ctx *Context) WithTheme(theme *Theme) *Context
```

<small>[core/context.go:243](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L243)</small>

### type MatchCase

```go
type MatchCase[T comparable] struct {
	Value   T
	View    View
	Default bool
}
```

MatchCase Generic Match for comparable values

<small>[core/conditionals.go:53](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L53)</small>

#### func Case

```go
func Case[T comparable](val T, view View) MatchCase[T]
```

<small>[core/conditionals.go:59](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L59)</small>

#### func Default

```go
func Default[T comparable](view View) MatchCase[T]
```

<small>[core/conditionals.go:63](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L63)</small>

### type Node

```go
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
```

Node is the retained render-tree element the reconciler diffs.

Immutability contract: a Node is frozen once its render pass returns it. Builders may assemble a node freely while constructing it (Keyed sets Key, containerNode applies behavior props), but after render nothing may write to it — the reconciler only reads, and renderers must also only read. The contract is what makes sharing safe: Cached returns the same \*Node every pass and Diff treats pointer equality as proof the subtree is unchanged, so a post-render mutation would silently never reach the screen.

<small>[core/node.go:12](https://github.com/rohanthewiz/grmob/blob/master/core/node.go#L12)</small>

### type RenderError

```go
type RenderError struct {
	// Value is exactly what was passed to panic(): usually an error or a
	// string, but it can be any type.
	Value any

	// Stack is debug.Stack() captured inside the deferred recover, so it
	// still contains the frames being unwound — the component that actually
	// panicked is in here, which is the whole point of keeping it. Nil only
	// if a RenderError was constructed by hand.
	Stack []byte
}
```

RenderError is a panic that escaped a component's Render and was caught by an ErrorBoundary (or by the render driver's top-level guard).

It carries the raw panic value rather than just a message because the panic may well be a real error worth inspecting: a \*net.OpError, a wrapped sentinel, a custom type the app wants to switch on. Unwrap exposes it to errors.Is/errors.As when it is one, so

	if errors.Is(err, sql.ErrNoRows) { ... }

works inside a fallback even though the value travelled through panic().

<small>[core/error_boundary.go:19](https://github.com/rohanthewiz/grmob/blob/master/core/error_boundary.go#L19)</small>

#### func Guard

```go
func Guard(fn func()) (rerr *RenderError)
```

Guard runs fn and converts a panic escaping it into a \*RenderError, returning nil when fn completes normally.

Exported because the render driver needs the same guard around a whole pass that ErrorBoundary needs around a subtree, and render is a separate package. It deliberately takes a func() rather than a View: the driver must also cover the root-view \*construction\* call, not only Render.

Guard restores nothing — it is the bare recover. Callers that intend to keep rendering after the failure are responsible for repairing whatever the half-finished work left behind; see renderRecovered for what that means inside a boundary.

<small>[core/error_boundary.go:63](https://github.com/rohanthewiz/grmob/blob/master/core/error_boundary.go#L63)</small>

#### func (*RenderError) Error

```go
func (e *RenderError) Error() string
```

Error renders the panic value as a message. The "panic during render" prefix is deliberate: a RenderError frequently ends up in a log line next to ordinary application errors, and without it a bare "index out of range" gives no hint that it came from a render pass.

<small>[core/error_boundary.go:35](https://github.com/rohanthewiz/grmob/blob/master/core/error_boundary.go#L35)</small>

#### func (*RenderError) Unwrap

```go
func (e *RenderError) Unwrap() error
```

Unwrap exposes the panicked value to errors.Is/As when it is an error, and returns nil otherwise (a panic("boom") wraps nothing).

<small>[core/error_boundary.go:44](https://github.com/rohanthewiz/grmob/blob/master/core/error_boundary.go#L44)</small>

### type RenderManager

```go
type RenderManager struct {
	// contains filtered or unexported fields
}
```

RenderManager is the app's "state changed" notification point: one registered handler (see OnStateChange), invoked whenever anything in the context tree calls RequestRender. One instance per NewContext root, shared by pointer with every derived context.

It is keyed by string rather than holding a bare func because it once carried a second, parallel registration API — RegisterRender, which minted "render\_N" ids, and SubscribeRender, which called it and threw the id away. Nothing ever triggered those ids: State.Set has always notified the hardcoded "default" key, so every SubscribeRender handler was unreachable while the map grew by one entry per call. Both are gone; OnStateChange is the registration side that actually completes the circuit.

<small>[core/render_manager.go:19](https://github.com/rohanthewiz/grmob/blob/master/core/render_manager.go#L19)</small>

#### func NewRenderManager

```go
func NewRenderManager() *RenderManager
```

<small>[core/render_manager.go:24](https://github.com/rohanthewiz/grmob/blob/master/core/render_manager.go#L24)</small>

#### func (*RenderManager) TriggerRender

```go
func (r *RenderManager) TriggerRender(id string)
```

TriggerRender invokes the handler registered under id, if any.

<small>[core/render_manager.go:31](https://github.com/rohanthewiz/grmob/blob/master/core/render_manager.go#L31)</small>

### type State

```go
type State[T any] struct {
	// contains filtered or unexported fields
}
```

<small>[core/context.go:178](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L178)</small>

#### func NewState

```go
func NewState[T any](ctx *Context, initial T) State[T]
```

NewState allocates (or on re-render, re-binds) the hook slot at the current cursor position and returns typed accessors for it.

Slot access is guarded by ctx.lock because reads and writes come from different goroutines: renders run on the manager/pump goroutine (or a native event thread), while Set may be called from timers, network handlers, or any goroutine the app spawns. Render passes themselves are serialized by render.Manager, so the lock's job is only to make individual slot accesses atomic against concurrent Sets — a Set landing mid-render yields a tree mixing old and new values for one pass, which is benign: the Set also nudges the pump, so a follow-up pass renders the settled state.

<small>[core/context.go:273](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L273)</small>

#### func (*State) Get

```go
func (s *State[T]) Get() T
```

<small>[core/context.go:183](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L183)</small>

#### func (*State) Set

```go
func (s *State[T]) Set(val T)
```

<small>[core/context.go:187](https://github.com/rohanthewiz/grmob/blob/master/core/context.go#L187)</small>

### type View

```go
type View interface {
	Render(ctx *Context) *Node
}
```

<small>[core/view.go:3](https://github.com/rohanthewiz/grmob/blob/master/core/view.go#L3)</small>

#### func Cached

```go
func Cached(view View) View
```

Cached returns a View that renders view on first use and replays the same \*Node on every later pass. See the type comment for the constraints on what may be cached.

<small>[core/cached.go:58](https://github.com/rohanthewiz/grmob/blob/master/core/cached.go#L58)</small>

#### func DefaultErrorFallback

```go
func DefaultErrorFallback(err error) View
```

DefaultErrorFallback is the fallback ErrorBoundary uses when given nil: a bordered card in the theme's Error role.

The detail line is gated on debug mode on purpose. A panic message is developer-facing text — "runtime error: index out of range \[7] with length 3" tells a user nothing and quietly leaks internals into a screenshot — so a release build shows only the generic line, while a debug build shows the message that identifies the bug. The full \*RenderError, stack included, is always available to a custom fallback either way.

<small>[core/error_boundary.go:208](https://github.com/rohanthewiz/grmob/blob/master/core/error_boundary.go#L208)</small>

#### func ErrorBoundary

```go
func ErrorBoundary(child View, fallback func(err error) View) View
```

ErrorBoundary renders child, and if child's render panics, renders fallback(err) in its place instead of letting the panic reach the render driver — where, on a native host, it would take the whole app down.

Pass a nil fallback to get DefaultErrorFallback.

	core.ErrorBoundary(
	    ProfilePanel(user),
	    func(err error) core.View {
	        log.Printf("profile panel failed: %v", err)
	        return core.Text("Profile unavailable")
	    },
	)

##### The fallback is also the notification hook

ErrorBoundary logs nothing itself. fallback is called on every pass in which the child fails, receives the full \*RenderError (stack included), and is the intended place to log, report, or degrade. Note "every pass": a component that panics deterministically panics again next frame, so a fallback that logs unconditionally will log at frame rate. Rate-limit, or log from a boundary placed high enough that failures are rare.

##### It does not latch

React's error boundaries stay in the fallback until explicitly reset, because there the failed subtree's instances are unrecoverable. Nothing of the sort is true here: the tree is rebuilt from scratch every pass, so a child that panicked on a stale slice index simply renders normally on the next pass once the state settles. Latching would turn a one-frame glitch into a permanent dead panel, so the boundary retries every pass and heals on its own. The cost is the repeated-panic case above, which is the right trade — a stuck fallback is worse than a noisy one.

##### What it repairs, and why it needs its own contexts

A panic partway through a render leaves two pieces of per-pass bookkeeping half-advanced, and both are positional, so leaving them where they fell would corrupt \*unrelated\* components rendered later in the same pass:

	hook slots      parent ctx.Cursor sits between the child's hooks, so
	                every later sibling reads the wrong slots — sibling
	                state visibly swaps
	callback IDs    the registry counters sit past the handlers the child
	                managed to register, so every later sibling's IDs shift
	                and taps land on the wrong handler

The hook half is solved structurally rather than by rollback: the boundary takes two child contexts (one for child, one for the fallback) and renders into those. A panic can then only strand a cursor inside the child's own context, and the boundary consumes exactly two parent slots whether the child succeeds, fails early, or fails late.

	parent ctx slots:   [ ... | childCtx | fallbackCtx | ... ]
	                             ^ panic strands the cursor in here only

The callback half is a genuine rollback: renderRecovered snapshots the registry counters before the child renders and rewinds them after a panic, so the boundary's ID footprint equals the fallback's footprint and does not depend on how far the failed render got.

##### Consequence: the child gets its own hook namespace

Because child renders into a child context, its hook slots and its ctx.Scope table are its own rather than the parent's. State is keyed by position within a context, so this is transparent for the child itself — but a component that reaches for ctx.Scope("x") expecting to share a scope with something \*outside\* the boundary will get a different scope. Shared app state (navigation, callbacks, theme, config) lives on pointers copied into every derived context and is unaffected.

<small>[core/error_boundary.go:146](https://github.com/rohanthewiz/grmob/blob/master/core/error_boundary.go#L146)</small>

#### func For

```go
func For[T any](items []T, render func(item T, index int) View) View
```

<small>[core/conditionals.go:10](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L10)</small>

#### func If

```go
func If(condition bool, view View) View
```

<small>[core/conditionals.go:3](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L3)</small>

#### func IfElse

```go
func IfElse(condition bool, thenView View, elseView View) View
```

<small>[core/conditionals.go:23](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L23)</small>

#### func Keyed

```go
func Keyed(key string, child View) View
```

<small>[core/view.go:13](https://github.com/rohanthewiz/grmob/blob/master/core/view.go#L13)</small>

#### func Match

```go
func Match[T comparable](input T, cases ...MatchCase[T]) View
```

<small>[core/conditionals.go:67](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L67)</small>

#### func MatchBool

```go
func MatchBool(clauses ...WhenClause) View
```

<small>[core/conditionals.go:43](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L43)</small>

#### func SafeRender

```go
func SafeRender(child View) View
```

SafeRender is ErrorBoundary with the built-in fallback — the one-liner for wrapping a subtree you merely want to survive, with no opinion about what replaces it.

<small>[core/error_boundary.go:195](https://github.com/rohanthewiz/grmob/blob/master/core/error_boundary.go#L195)</small>

#### func Text

```go
func Text(content string, styleProps ...StyleProp) View
```

<small>[core/text.go:3](https://github.com/rohanthewiz/grmob/blob/master/core/text.go#L3)</small>

### type WhenClause

```go
type WhenClause struct {
	Condition bool
	View      View
}
```

<small>[core/conditionals.go:30](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L30)</small>

#### func Otherwise

```go
func Otherwise(view View) WhenClause
```

<small>[core/conditionals.go:39](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L39)</small>

#### func When

```go
func When(cond bool, view View) WhenClause
```

<small>[core/conditionals.go:35](https://github.com/rohanthewiz/grmob/blob/master/core/conditionals.go#L35)</small>

