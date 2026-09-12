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
    - [`func (*Manager) RenderAgain`](#func-manager-renderagain)
    - [`func (*Manager) RenderAndGetPatches`](#func-manager-renderandgetpatches)
    - [`func (*Manager) RenderInitial`](#func-manager-renderinitial)
    - [`func (*Manager) SetListener`](#func-manager-setlistener)
- [`type PatchListener`](#type-patchlistener)

## Types

### type Manager

```go
type Manager struct {
	// contains filtered or unexported fields
}
```

<small>[render/manager.go:32](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L32)</small>

#### func New

```go
func New(ctx *core.Context, rootView func(*core.Context) core.View) *Manager
```

<small>[render/manager.go:69](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L69)</small>

#### func (*Manager) Close

```go
func (m *Manager) Close()
```

Close stops the push pump and closes the app's context tree, which stops the background resources hooks registered on it (interval tickers, pending timeouts). The Manager is the app-lifetime owner, so its Close is the one shutdown entry point: hosts that replace an app (mobile.Register, the WASM runtime re-mounting) close the old Manager and thereby cannot leak tickers rendering into a dead tree. Normally an app-lifetime singleton, so this mainly matters for tests and hot-reload hosts.

<small>[render/manager.go:93](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L93)</small>

#### func (*Manager) Dispatch

```go
func (m *Manager) Dispatch(label string, fn func()) string
```

Dispatch runs fn under the render mutex — the same serialization the callback dispatches above get — then renders and returns the resulting patches. It is the event path for host events (mobile.ReportHostEvent): traffic that reaches Go from the shell without a callback ID, such as the audio player's status ticks. label names the work in the panic log.

Before the initial mount there is no tree to diff against, so fn runs and the state it wrote is simply part of the RenderInitial that follows; the "\[]" tells the host nothing needs applying. That case is rare (a status tick cannot precede the Load that caused it, and Load needs a rendered button) but a host that sent it would otherwise receive a diff against nothing, which no renderer can apply.

<small>[render/manager.go:388](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L388)</small>

#### func (*Manager) DispatchBoolCallback

```go
func (m *Manager) DispatchBoolCallback(id string, value bool) string
```

DispatchBoolCallback is DispatchCallback for bool-carrying events.

<small>[render/manager.go:361](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L361)</small>

#### func (*Manager) DispatchCallback

```go
func (m *Manager) DispatchCallback(id string) string
```

DispatchCallback runs the void handler registered under id (a button tap, say), then renders and returns the resulting patches — the whole sequence under the render mutex, so the handler cannot interleave with a pump render pass and its state writes are diffed in the same lock hold. This is the event path native bridges should use; the Dispatch\*/Trigger split exists because dispatching a handler without an immediate render (the async shape) is only wanted by hosts that poll or rely purely on the push channel.

Handlers may call State.Set freely: the resulting RequestRender nudge is asynchronous (a buffered-channel send plus a goroutine hop), so nothing in the handler path re-enters this mutex. The pump's follow-up pass then finds an empty diff and pushes nothing.

<small>[render/manager.go:345](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L345)</small>

#### func (*Manager) DispatchIntCallback

```go
func (m *Manager) DispatchIntCallback(id string, value int) string
```

DispatchIntCallback is DispatchCallback for int-carrying events.

<small>[render/manager.go:369](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L369)</small>

#### func (*Manager) DispatchTextCallback

```go
func (m *Manager) DispatchTextCallback(id string, value string) string
```

DispatchTextCallback is DispatchCallback for string-carrying events.

<small>[render/manager.go:353](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L353)</small>

#### func (*Manager) RenderAgain

```go
func (r *Manager) RenderAgain() string
```

RenderAgain ReRender Used after an event (input/click/state change) to get diff

<small>[render/manager.go:265](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L265)</small>

#### func (*Manager) RenderAndGetPatches

```go
func (r *Manager) RenderAndGetPatches() string
```

RenderAndGetPatches renders one pass and returns whichever payload the host needs: the full tree when nothing is mounted yet, the diff against the mounted tree afterwards. It is the "mount or update, I don't want to know which" entry point for a host driving passes by hand; render.Manager's own callers use RenderInitial and RenderAgain, which say which they mean.

It delegates rather than re-implementing the two passes, which is the whole of the fix here. Its own copy had drifted badly: it reset the root cursor with \`r.context.Cursor = 0\` instead of \`Reset()\`, so every child scope kept its cursor from the previous pass and its slots grew by one per render; and it never called PurgeUnusedCallbacks or ClearDirty, so handlers for vanished nodes stayed dispatchable and a polling host re-rendered forever.

<small>[render/manager.go:444](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L444)</small>

#### func (*Manager) RenderInitial

```go
func (r *Manager) RenderInitial() string
```

<small>[render/manager.go:202](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L202)</small>

#### func (*Manager) SetListener

```go
func (m *Manager) SetListener(l PatchListener)
```

SetListener attaches (or replaces) the native push target. Any state change that happened before attachment is flushed immediately so the listener never starts out behind.

<small>[render/manager.go:103](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L103)</small>

### type PatchListener

```go
type PatchListener interface {
	ApplyPatches(patches string)
}
```

PatchListener is the Go→native push channel: the native shell implements it and registers via Manager.SetListener, and Go calls ApplyPatches whenever state changes outside a native event — timer ticks, network responses, any goroutine calling State.Set. Without it the bridge is strictly request/response and async updates never reach the screen (WASM worked around this by polling IsDirty).

The single string-parameter method is deliberate: gomobile bind maps this interface onto Java/Kotlin and Objective-C/Swift directly, so an Android Activity or iOS view controller can implement it without JNI glue.

ApplyPatches is invoked from a background goroutine. Native implementations must hop to their UI thread before touching views (runOnUiThread / DispatchQueue.main); the payload is the same patch-array JSON RenderAgain returns, and an empty patch set is never pushed.

<small>[render/manager.go:28](https://github.com/rohanthewiz/grmob/blob/master/render/manager.go#L28)</small>

