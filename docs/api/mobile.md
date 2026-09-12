# Package mobile

```go
import "github.com/rohanthewiz/grmob/mobile"
```

Package mobile is the gomobile-bindable bridge surface for native shells.

The rest of the framework's API is not bind-safe (gomobile cannot bind function parameters, generics, or map-typed exports), so this package narrows everything the native side needs to strings, bools, and a single-method interface. An app author writes their root view in Go, wires it in an init step, and binds this package plus their own:

	// app's Go code
	func init() { mobile.Register(core.NewContext(), App) }

	// Kotlin (via the gomobile-generated classes)
	Mobile.setListener { patches -> runOnUiThread { renderer.applyPatches(patches) } }
	val tree = Mobile.renderInitial()
	...
	val patches = Mobile.triggerCallback(id) // event path, synchronous

Event delivery contract: patches reach the native side on two paths — the synchronous return value of the Trigger\* functions (the event path) and PatchListener pushes (the async path: timers, goroutines calling State.Set). Each render pass produces its diff exactly once and delivers it on exactly one of the two paths, so the native renderer applies everything it receives from either path, in arrival order, and stays consistent.

## Index

- [`func DataDir`](#func-datadir)
- [`func Register`](#func-register)
- [`func RenderAgain`](#func-renderagain)
- [`func RenderInitial`](#func-renderinitial)
- [`func ReportHostEvent`](#func-reporthostevent)
- [`func SetDataDir`](#func-setdatadir)
- [`func SetListener`](#func-setlistener)
- [`func SetSystemEventListener`](#func-setsystemeventlistener)
- [`func TriggerBoolCallback`](#func-triggerboolcallback)
- [`func TriggerCallback`](#func-triggercallback)
- [`func TriggerIntCallback`](#func-triggerintcallback)
- [`func TriggerTextCallback`](#func-triggertextcallback)
- [`type PatchListener`](#type-patchlistener)
- [`type SystemEventListener`](#type-systemeventlistener)

## Functions

### func DataDir

```go
func DataDir() string
```

DataDir returns the writable directory the shell registered, or "" if none.

<small>[mobile/bridge.go:59](https://github.com/rohanthewiz/grmob/blob/master/mobile/bridge.go#L59)</small>

### func Register

```go
func Register(ctx *core.Context, root func(*core.Context) core.View)
```

Register installs the app's root view and context. It must be called from Go (typically the bound app package's init) before the native shell invokes any other function here. Calling it again replaces the app — the previous manager's push pump is shut down so it cannot keep rendering the old tree.

<small>[mobile/bridge.go:67](https://github.com/rohanthewiz/grmob/blob/master/mobile/bridge.go#L67)</small>

### func RenderAgain

```go
func RenderAgain() string
```

RenderAgain re-renders and returns the diff against the last rendered tree. Native shells normally don't call this directly — the Trigger\* functions already fold it into the event path — but it is the escape hatch for shells that drive rendering themselves.

<small>[mobile/bridge.go:83](https://github.com/rohanthewiz/grmob/blob/master/mobile/bridge.go#L83)</small>

### func RenderInitial

```go
func RenderInitial() string
```

RenderInitial returns the full initial tree as JSON for the first mount.

<small>[mobile/bridge.go:75](https://github.com/rohanthewiz/grmob/blob/master/mobile/bridge.go#L75)</small>

### func ReportHostEvent

```go
func ReportHostEvent(name string, payload string) string
```

ReportHostEvent is the host→app half of the bridge for traffic that is not an answer to a registered callback: the audio player's status ticks today, and whatever the shells report next (see core/host\_events.go for the channel and why it is generic).

It mirrors the Trigger\* functions exactly. name is the event kind ("audio\_status"); payload is its data as a JSON object, crossing as text for the reason system events do — a Go map is not a bindable type. The return value is the patches of the render pass that follows, delivered on the event path, so a shell dispatches it on the same serial executor as its Trigger\* calls and applies the result the same way. Nothing else is needed for a status tick to reach the screen.

A malformed payload is dropped here with a log line and "\[]" returned — the shell built the JSON, so the bug is on that side and a Go-side panic would only obscure it. An empty payload is a valid empty object.

<small>[mobile/hostevents.go:24](https://github.com/rohanthewiz/grmob/blob/master/mobile/hostevents.go#L24)</small>

### func SetDataDir

```go
func SetDataDir(path string)
```

SetDataDir records the app's writable directory for Go-side persistence (see examples/todoapp's bytdb store). The shell must call it before RenderInitial so the first render can already read persisted state; the bound app package's init runs earlier still, which is why apps open their stores lazily on first render rather than in init. Left unset, DataDir returns "" and persistence-aware apps run in-memory — the right behavior for the web preview targets and for tests that don't care about storage.

<small>[mobile/bridge.go:54](https://github.com/rohanthewiz/grmob/blob/master/mobile/bridge.go#L54)</small>

### func SetListener

```go
func SetListener(l PatchListener)
```

SetListener attaches the native push target for async updates.

<small>[mobile/bridge.go:88](https://github.com/rohanthewiz/grmob/blob/master/mobile/bridge.go#L88)</small>

### func SetSystemEventListener

```go
func SetSystemEventListener(l SystemEventListener)
```

SetSystemEventListener installs the shell's sink. Passing nil detaches it, after which system events are dropped silently — the correct behavior for a headless run, and the state a shell that never calls this stays in.

It exists because core.SetSystemEventHandler takes a func, which gobind cannot bind, so before this there was no way for a native shell to receive a toast at all: the events were emitted into a nil handler and vanished. (The WASM host had its own path and did not; see wasm/main.go.)

Registering replaces any previous listener rather than fanning out. A process has one screen, so a second sink would mean one gesture producing two toasts.

<small>[mobile/sysevents.go:47](https://github.com/rohanthewiz/grmob/blob/master/mobile/sysevents.go#L47)</small>

### func TriggerBoolCallback

```go
func TriggerBoolCallback(id string, value bool) string
```

TriggerBoolCallback dispatches a bool-carrying event (e.g. checkbox toggle).

<small>[mobile/bridge.go:106](https://github.com/rohanthewiz/grmob/blob/master/mobile/bridge.go#L106)</small>

### func TriggerCallback

```go
func TriggerCallback(id string) string
```

TriggerCallback dispatches a void event (e.g. a button tap) by callback ID and returns the resulting patches. Dispatch goes through the manager so the handler and its follow-up render run under the render mutex — an event can never interleave with a push-pump render pass.

<small>[mobile/bridge.go:96](https://github.com/rohanthewiz/grmob/blob/master/mobile/bridge.go#L96)</small>

### func TriggerIntCallback

```go
func TriggerIntCallback(id string, value int) string
```

TriggerIntCallback dispatches an int-carrying event (e.g. tab selection).

<small>[mobile/bridge.go:111](https://github.com/rohanthewiz/grmob/blob/master/mobile/bridge.go#L111)</small>

### func TriggerTextCallback

```go
func TriggerTextCallback(id string, value string) string
```

TriggerTextCallback dispatches a string-carrying event (e.g. input change).

<small>[mobile/bridge.go:101](https://github.com/rohanthewiz/grmob/blob/master/mobile/bridge.go#L101)</small>

## Types

### type PatchListener

```go
type PatchListener interface {
	ApplyPatches(patches string)
}
```

PatchListener is re-declared here (rather than aliased to render.PatchListener) so this package binds standalone; see that type for the threading contract — ApplyPatches arrives on a background goroutine and implementations must hop to their UI thread.

<small>[mobile/bridge.go:35](https://github.com/rohanthewiz/grmob/blob/master/mobile/bridge.go#L35)</small>

### type SystemEventListener

```go
type SystemEventListener interface {
	// OnSystemEvent delivers one event. name is the event kind ("toast",
	// "open_url"); payload is its data as a JSON object.
	OnSystemEvent(name string, payload string)
}
```

SystemEventListener is the native shell's sink for app→host system events: the transient, non-tree things a platform draws or performs itself — core.ShowToast and core.OpenURL today.

It mirrors PatchListener exactly, and for the same reason: gomobile cannot bind a function parameter, but it can bind a single-method interface, so an interface is how a callback crosses the FFI at all. Both halves of the payload are strings for the same reason — core's handler signature is (string, map\[string]any), and a Go map is not a bindable type — so the data crosses as JSON text, the same envelope discipline patches and events already use.

#### Threading

OnSystemEvent arrives synchronously on whichever goroutine called ShowToast or OpenURL. That may be the render goroutine (a tap handler), a timer, or any goroutine the app spawned — never, reliably, the platform's UI thread. Implementations must therefore hop to their own main thread before touching UI, exactly as PatchListener implementations do.

<small>[mobile/sysevents.go:29](https://github.com/rohanthewiz/grmob/blob/master/mobile/sysevents.go#L29)</small>

