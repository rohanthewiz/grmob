# Package render

```go
import "github.com/rohanthewiz/grmob/render"
```

Package render is the loop that turns an app function into patches: it owns the retained tree, serialises every pass, and decides when a pass happens.

A [Manager](#type-manager) is constructed once per app with a root view function, and from then on it is the only thing that calls that function.

	m := render.New(core.NewContext(), App)
	m.SetListener(hostListener)     // optional: the Go -> host push channel
	first := m.RenderInitial()      // the whole tree, as JSON
	patches := m.Dispatch("tick", fn)

## Two ways in, one at a time

A pass starts from one of two directions, and the Manager's mutex is what keeps them from interleaving:

	host event ──▶ Dispatch* ──┐
	                           ├──▶ render ──▶ Diff ──▶ patches
	State.Set  ──▶ push pump ──┘      (mutex held for the whole pass)

Both paths mutate state that cannot tolerate interleaving — the context's hook cursor and its callback registry are both reset at the start of a pass — so an event handler can never run in the middle of one. The practical consequence for an app is a guarantee worth relying on: a handler always observes a settled tree, and the writes it makes are rendered together by the pass that follows.

## Pushed updates are coalesced

State written outside a host event (a timer, a network response, any goroutine) reaches the screen through [PatchListener](#type-patchlistener). Nudges are coalesced through a one-slot buffer, so a burst of writes produces one pass over the settled state rather than one pass per write, and the last write is never lost. An empty patch set is never pushed.

PatchListener.ApplyPatches is called from a background goroutine. A native implementation must hop to its own UI thread before touching views.

## Closing

[Manager.Close](#func-manager-close) stops the pump and closes the context tree, which stops the background resources hooks registered on it. It is the one shutdown entry point, which is what lets a host that replaces a running app — mobile.Register, the WASM runtime re-mounting, a hot reload — do so without leaking a ticker that renders into a dead tree.

## Index

- [`type Manager`](#type-manager)
    - [`func New`](#func-new)
    - [`func (*Manager) Close`](#func-manager-close)
    - [`func (*Manager) Dispatch`](#func-manager-dispatch)
    - [`func (*Manager) DispatchBoolCallback`](#func-manager-dispatchboolcallback)
    - [`func (*Manager) DispatchCallback`](#func-manager-dispatchcallback)
    - [`func (*Manager) DispatchIntCallback`](#func-manager-dispatchintcallback)
    - [`func (*Manager) DispatchTextCallback`](#func-manager-dispatchtextcallback)
    - [`func (*Manager) DispatchTextEdit`](#func-manager-dispatchtextedit)
    - [`func (*Manager) RenderAgain`](#func-manager-renderagain)
    - [`func (*Manager) RenderAndGetPatches`](#func-manager-renderandgetpatches)
    - [`func (*Manager) RenderIfDirty`](#func-manager-renderifdirty)
    - [`func (*Manager) RenderInitial`](#func-manager-renderinitial)
    - [`func (*Manager) Resync`](#func-manager-resync)
    - [`func (*Manager) SetListener`](#func-manager-setlistener)
- [`type PatchListener`](#type-patchlistener)

## Types

### type Manager

```go
type Manager struct {
	// contains filtered or unexported fields
}
```

<small>[render/manager.go:37](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L37)</small>

#### func New

```go
func New(ctx *core.Context, rootView func(*core.Context) core.View) *Manager
```

<small>[render/manager.go:88](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L88)</small>

#### func (*Manager) Close

```go
func (m *Manager) Close()
```

Close stops the push pump and closes the app's context tree, which stops the background resources hooks registered on it (interval tickers, pending timeouts). The Manager is the app-lifetime owner, so its Close is the one shutdown entry point: hosts that replace an app (mobile.Register, the WASM runtime re-mounting) close the old Manager and thereby cannot leak tickers rendering into a dead tree. Normally an app-lifetime singleton, so this mainly matters for tests and hot-reload hosts.

<small>[render/manager.go:112](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L112)</small>

#### func (*Manager) Dispatch

```go
func (m *Manager) Dispatch(label string, fn func()) string
```

Dispatch runs fn under the render mutex — the same serialization the callback dispatches above get — then renders and returns the resulting patches. It is the event path for host events (mobile.ReportHostEvent): traffic that reaches Go from the shell without a callback ID, such as the audio player's status ticks. label names the work in the panic log.

Before the initial mount there is no tree to diff against, so fn runs and the state it wrote is simply part of the RenderInitial that follows; the "\[]" tells the host nothing needs applying. That case is rare (a status tick cannot precede the Load that caused it, and Load needs a rendered button) but a host that sent it would otherwise receive a diff against nothing, which no renderer can apply.

<small>[render/manager.go:632](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L632)</small>

#### func (*Manager) DispatchBoolCallback

```go
func (m *Manager) DispatchBoolCallback(id string, value bool) string
```

DispatchBoolCallback is DispatchCallback for bool-carrying events.

<small>[render/manager.go:605](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L605)</small>

#### func (*Manager) DispatchCallback

```go
func (m *Manager) DispatchCallback(id string) string
```

DispatchCallback runs the void handler registered under id (a button tap, say), then renders and returns the resulting patches — the whole sequence under the render mutex, so the handler cannot interleave with a pump render pass and its state writes are diffed in the same lock hold. This is the event path native bridges should use; the Dispatch\*/Trigger split exists because dispatching a handler without an immediate render (the async shape) is only wanted by hosts that poll or rely purely on the push channel.

Handlers may call State.Set freely: the resulting RequestRender nudge is a non-blocking buffered-channel send, so nothing in the handler path re-enters this mutex. The pump then wakes on that nudge, finds the write already drawn by this dispatch's pass (core's dirtyFlag generation), and renders nothing (N-138, plan P1).

With a listener attached the patches are handed to it before the lock is released and the return value is "\[]"; without one they are returned. Every Dispatch\* below does the same. See deliverLocked for the ordering hole the return value left open.

<small>[render/manager.go:578](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L578)</small>

#### func (*Manager) DispatchIntCallback

```go
func (m *Manager) DispatchIntCallback(id string, value int) string
```

DispatchIntCallback is DispatchCallback for int-carrying events.

<small>[render/manager.go:613](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L613)</small>

#### func (*Manager) DispatchTextCallback

```go
func (m *Manager) DispatchTextCallback(id string, value string) string
```

DispatchTextCallback is DispatchCallback for string-carrying events.

<small>[render/manager.go:586](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L586)</small>

#### func (*Manager) DispatchTextEdit

```go
func (m *Manager) DispatchTextEdit(id string, value string, seq, epoch int) string
```

DispatchTextEdit is DispatchTextCallback for a native text field's keystroke, carrying the host's sequence number and adopted rewrite epoch. An edit typed against text Go has since rewritten is acknowledged without reaching the handler; see core/text\_edit.go for the protocol.

<small>[render/manager.go:597](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L597)</small>

#### func (*Manager) RenderAgain

```go
func (r *Manager) RenderAgain() string
```

RenderAgain ReRender Used after an event (input/click/state change) to get diff

<small>[render/manager.go:396](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L396)</small>

#### func (*Manager) RenderAndGetPatches

```go
func (r *Manager) RenderAndGetPatches() string
```

RenderAndGetPatches renders one pass and returns whichever payload the host needs: the full tree when nothing is mounted yet, the diff against the mounted tree afterwards. It is the "mount or update, I don't want to know which" entry point for a host driving passes by hand; render.Manager's own callers use RenderInitial and RenderAgain, which say which they mean.

It delegates rather than re-implementing the two passes, which is the whole of the fix here. Its own copy had drifted badly: it reset the root cursor with \`r.context.Cursor = 0\` instead of \`Reset()\`, so every child scope kept its cursor from the previous pass and its slots grew by one per render; and it never called PurgeUnusedCallbacks or ClearDirty, so handlers for vanished nodes stayed dispatchable and a polling host re-rendered forever.

<small>[render/manager.go:772](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L772)</small>

#### func (*Manager) RenderIfDirty

```go
func (r *Manager) RenderIfDirty() string
```

RenderIfDirty is RenderAgain for a host that asks after every event: it renders only when some write has not been drawn yet, and otherwise answers "\[]" without building anything.

##### Why the web hosts need it

A browser page calls GrMobWASM.ReceiveEvent and then RenderAgain for every DOM event. With the push listener attached, the handler's writes reach the page through the pump, which on js/wasm runs inside ReceiveEvent, so the RenderAgain that followed built the whole app again to return "\[]"; 1.2 to 6.0 ms a tap in the plan's measurements (N-138, P1). Asked under mu, the same as the pump's check, so the answer cannot go stale before the pass.

RenderAgain itself keeps rendering unconditionally: a test or a host driving passes by hand may want a pass whatever the flag says (state read from outside any State, for instance), and that is what it has always promised.

<small>[render/manager.go:419](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L419)</small>

#### func (*Manager) RenderInitial

```go
func (r *Manager) RenderInitial() string
```

<small>[render/manager.go:283](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L283)</small>

#### func (*Manager) Resync

```go
func (m *Manager) Resync() string
```

Resync sends the host the whole tree Go believes it holds, as one replace of the root, so a host whose tree has drifted from Go's can start again from Go's (N-133, the plan's R4).

##### Why a host needs it

Patches are positional and incremental: each is correct only against the exact tree the previous one produced. A host that misses one, or applies one wrongly, stays wrong, and every later patch compounds it. Every host already noticed the symptom (a patch whose path resolves to nothing) and logged it; the TreeStores said "the next full replace re-synchronizes", but nothing ever asked for one. A host now asks, through mobile.Resync or GrMobWASM.Resync, when a path dangles, a payload is not an array, or a patch throws.

##### What it sends

currentTree, which is the tree the last committed pass left the host with (an unencodable or panicking pass is not committed; see renderAgainLocked). Nothing is re-rendered: the question is what the host should be holding, and a render would answer a different one. The callback IDs in that tree are the ones still registered, for the same reason. It uses no new wire shape: every host already applies a replace of "root", because a navigation emits one.

It goes out like a dispatch (deliverLocked): through the listener inside the lock when one is attached, so it cannot be overtaken by a pass and cannot overtake one, and returned otherwise. Before the mount there is nothing to send, and the answer is "\[]".

<small>[render/manager.go:672](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L672)</small>

#### func (*Manager) SetListener

```go
func (m *Manager) SetListener(l PatchListener)
```

SetListener attaches (or replaces) the native push target. Any state change that happened before attachment is flushed immediately so the listener never starts out behind.

<small>[render/manager.go:122](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L122)</small>

### type PatchListener

```go
type PatchListener interface {
	ApplyPatches(patches string)
}
```

PatchListener is the Go→native push channel: the native shell implements it and registers via Manager.SetListener, and Go calls ApplyPatches whenever state changes outside a native event — timer ticks, network responses, any goroutine calling State.Set. Without it the bridge is strictly request/response and async updates never reach the screen (WASM worked around this by polling IsDirty).

The single string-parameter method is deliberate: gomobile bind maps this interface onto Java/Kotlin and Objective-C/Swift directly, so an Android Activity or iOS view controller can implement it without JNI glue.

ApplyPatches is invoked from a background goroutine (the pump), or from the thread that called a Dispatch\* method, inside that call: with a listener attached, a dispatch delivers its pass here rather than returning it (see deliverLocked). Either way the render lock is held. Native implementations must hop to their UI thread before touching views (runOnUiThread / DispatchQueue.main) and return, without waiting on the hop; the payload is the same patch-array JSON RenderAgain returns, and an empty patch set is never pushed.

<small>[render/manager.go:33](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L33)</small>

