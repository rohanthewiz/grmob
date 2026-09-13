# Package core — Navigation & overlays

```go
import "github.com/rohanthewiz/grmob/core"
```

The navigator stack, modals, toasts, deep links and opening URLs.

One of 11 topic pages of [package core](core.md), which has the package overview and an index of every topic. This page documents the declarations in `core/navigation.go`, `core/modal.go`, `core/toast.go`, `core/deeplink.go`, `core/openurl.go`.

## Index

- [`func CanPop`](#func-canpop)
- [`func Modal`](#func-modal)
- [`func Navigator`](#func-navigator)
- [`func OnDeepLink`](#func-ondeeplink)
- [`func OpenURL`](#func-openurl)
- [`func Pop`](#func-pop)
- [`func PopToRoot`](#func-poptoroot)
- [`func Push`](#func-push)
- [`func Render`](#func-render)
- [`func Replace`](#func-replace)
- [`func Reset`](#func-reset)
- [`func ShowToast`](#func-showtoast)
- [`func StackDepth`](#func-stackdepth)
- [`type ModalNode`](#type-modalnode)
- [`type ModalProp`](#type-modalprop)
    - [`func Backdrop`](#func-backdrop)
    - [`func ModalContent`](#func-modalcontent)
    - [`func OnDismiss`](#func-ondismiss)
    - [`func Visible`](#func-visible)
- [`type ToastConfig`](#type-toastconfig)
- [`type ToastOpt`](#type-toastopt)
    - [`func Duration`](#func-duration)
    - [`func UseToastStyle`](#func-usetoaststyle)

## Functions

### func CanPop

```go
func CanPop(ctx *Context) bool
```

CanPop reports whether there is a screen to go back to, which is what a back button or a hardware-back handler needs in order to decide between popping and exiting the app. Pop is a safe no-op when this is false; the point of asking first is to avoid rendering a control that does nothing.

<small>[core/navigation.go:356](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L356)</small>

### func Modal

```go
func Modal(props ...ModalProp) View
```

<small>[core/modal.go:14](https://github.com/rohanthewiz/grmob/blob/master/core/modal.go#L14)</small>

### func Navigator

```go
func Navigator(initial func(*Context) View) View
```

Navigator renders the top of the route stack, seeding the stack with initial the first time it renders. It emits no wrapper node of its own — the tree it returns is the route's tree — so a Navigator can sit anywhere a view can.

Each frame renders into its own scope of the host context, which has three consequences worth knowing:

  - Routes may use hooks freely. NewState in a pushed route claims slot 0 of that frame, not slot 0 of whatever screen is underneath it.
  - A route's state, and any background resource its hooks started, is discarded when its frame leaves the stack (Pop, Replace, Reset).
  - State that must outlive a frame belongs above the Navigator. Routes are closures, so the usual move is to capture the context the Navigator itself renders into and keep the state in a scope of that.

Note that Navigator does not call ctx.Reset(): cursors are restarted once per pass by the render driver, before the root render. A second, partial Reset from inside the tree would rewind the cursor of every context at or below this one mid-pass — harmless when the Navigator is the root view and silently corrupting when it is not, since siblings rendered before it have already consumed slots that the rewind hands out again.

<small>[core/navigation.go:161](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L161)</small>

### func OnDeepLink

```go
func OnDeepLink(fn func(url string)) (cancel func())
```

OnDeepLink subscribes fn to inbound URLs. The returned function cancels the subscription.

	cancel := core.OnDeepLink(func(url string) {
	    if id, ok := strings.CutPrefix(url, "grmob://lesson/"); ok {
	        navigate(id)
	    }
	})

fn runs on whichever goroutine delivered the event — a host bridge call — and must not block; the usual body parses the URL and asks for a render.

An empty or absent "url" field is dropped rather than delivered as "": a subscriber cannot do anything useful with no address, and a shell that reported one has a bug this hides less well than it would pass on.

<small>[core/deeplink.go:76](https://github.com/rohanthewiz/grmob/blob/master/core/deeplink.go#L76)</small>

### func OpenURL

```go
func OpenURL(url string)
```

OpenURL asks the host to open a URL outside the app — the platform's own browser, mail composer, dialer or map, whichever the scheme names.

It is a system event (see sys\_events.go) rather than a node, for the same reason ShowToast is: nothing about it is part of the view tree. There is no element to reconcile, no state to diff, and the thing it ultimately reaches is an OS-level facility the app does not own. So it travels one way, as a named payload handed to whatever host is driving the app, and — like a toast — it is callable from any goroutine and takes no Context.

Each host maps the event onto its platform's own hand-off:

	Android   Intent(ACTION_VIEW, uri), started with FLAG_ACTIVITY_NEW_TASK
	iOS       UIApplication.shared.open(url)
	Browser   window.open(url, "_blank", "noopener,noreferrer")
	Headless  nothing (no handler registered — see SendSystemEvent)

#### Why "outside the app" is the whole contract

The three platforms differ on almost everything about in-app browsing — Custom Tabs versus SFSafariViewController versus an iframe — and agree completely on handing a URL to the system. So this promises only the part that is portable. An app that needs an embedded browser wants a view node, which is a different (and much larger) feature.

#### Failure is silent, and that is deliberate

A malformed URL, a scheme no app on the device claims (a \`tel:\` link on a tablet with no dialer), or a host that registered no handler at all: none of these come back. There is no return channel on a system event, and synthesizing one would mean either blocking the caller on an OS round trip or inventing a callback protocol for a fire-and-forget gesture. Callers that must know whether a link is reachable have to decide that before calling — which in practice means not rendering the affordance at all, exactly as core.Button does with a nil handler.

An empty url is dropped here rather than sent, since every host would have to reject it separately and none could report that it had.

<small>[core/openurl.go:41](https://github.com/rohanthewiz/grmob/blob/master/core/openurl.go#L41)</small>

### func Pop

```go
func Pop(ctx *Context)
```

Pop removes the top route, discarding its state, and reveals the one below. It is a no-op at the root — the stack is never left empty, because Navigator has nothing to render then.

<small>[core/navigation.go:240](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L240)</small>

### func PopToRoot

```go
func PopToRoot(ctx *Context) bool
```

PopToRoot unwinds to the bottom of the stack, discarding the state of every frame above it, and returns whether anything was popped.

It differs from Reset in exactly one way, and it is the way that matters: the root frame is the one already there, state and all. Reset(ctx, root) would look identical on screen and quietly reset the root's scroll position, selected tab and form contents. Reach for PopToRoot to escape a deep drill-down ("Done" out of a five-level settings tree), and for Reset to end a session.

<small>[core/navigation.go:321](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L321)</small>

### func Push

```go
func Push(ctx *Context, route func(*Context) View)
```

Push adds a route on top of the stack. The screen underneath keeps its state and is restored intact by the matching Pop.

Like every mutation here it ends in RequestRender rather than a bare MarkDirty, which these used to do. Marking alone is enough only when a pass is already guaranteed to follow — true for a tap, since the native dispatch path re-renders on the way out, and false for a navigation triggered from anywhere else: an effect goroutine resolving a deep link, a timeout dismissing a splash screen, a websocket pushing the user to a call screen. Those marked the tree dirty and then waited for an unrelated event to notice.

<small>[core/navigation.go:230](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L230)</small>

### func Render

```go
func Render(ctx *Context, view View) *Node
```

Render renders view into ctx after restarting ctx's hook cursors. It is the entry point for a host driving passes by hand; render.Manager does the same two steps itself (with the debug pass boundary around them) and does not call this.

<small>[core/navigation.go:364](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L364)</small>

### func Replace

```go
func Replace(ctx *Context, route func(*Context) View)
```

Replace swaps the top route for another without changing the stack depth, discarding the outgoing route's state. Use it for a step that should not be returned to — the "logged in" screen after a login form, so Back skips the form rather than showing it again.

<small>[core/navigation.go:264](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L264)</small>

### func Reset

```go
func Reset(ctx *Context, route func(*Context) View)
```

Reset discards the entire stack and starts over with route as the only frame. This is the log-out / onboarding-complete operation: every frame's hook state is thrown away and every background resource its hooks started is stopped, so nothing from the previous session survives to be re-displayed.

The new root is a fresh frame even when route is the same function the old root ran, which is the point — resetting to the login screen must not show the previous tenant's half-filled form.

What Reset does not touch is state the app deliberately kept outside the stack: hooks on the context hosting the Navigator, package-level stores, the database. Those outlive navigation by construction, and clearing them is the app's call, not the router's.

<small>[core/navigation.go:300](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L300)</small>

### func ShowToast

```go
func ShowToast(msg string, opts ...ToastOpt)
```

<small>[core/toast.go:14](https://github.com/rohanthewiz/grmob/blob/master/core/toast.go#L14)</small>

### func StackDepth

```go
func StackDepth(ctx *Context) int
```

StackDepth reports how many frames are on the stack.

Before the Navigator's first render it counts only what the app itself pushed — 0 for an app that has not navigated yet, because the initial route is installed lazily by that first render. Afterwards it is at least 1.

<small>[core/navigation.go:346](https://github.com/rohanthewiz/grmob/blob/master/core/navigation.go#L346)</small>

## Types

### type ModalNode

```go
type ModalNode struct {
	Visible   bool
	OnDismiss func()
	Backdrop  string
	Content   []View
}
```

<small>[core/modal.go:7](https://github.com/rohanthewiz/grmob/blob/master/core/modal.go#L7)</small>

### type ModalProp

```go
type ModalProp interface {
	Apply(*ModalNode)
}
```

<small>[core/modal.go:3](https://github.com/rohanthewiz/grmob/blob/master/core/modal.go#L3)</small>

#### func Backdrop

```go
func Backdrop(color string) ModalProp
```

<small>[core/modal.go:60](https://github.com/rohanthewiz/grmob/blob/master/core/modal.go#L60)</small>

#### func ModalContent

```go
func ModalContent(children ...View) ModalProp
```

ModalContent sets the views drawn inside the overlay. It appends rather than replaces, so content may be assembled across several props if a caller finds that clearer; order is render order, top to bottom.

The content renders every pass regardless of Visible — a Modal hides, it does not unmount. Visible is an ordinary prop the host maps to visibility (display on the web), so toggling it is a cheap prop patch, not a subtree add/remove, and any state hooks inside the content survive a close. That makes the trade-off against navigation explicit: a dismissed modal reopens exactly as it was left, where a popped Navigator frame starts fresh. A dialog whose state must NOT survive dismissal should reset it in OnDismiss, where the intent is recorded.

<small>[core/modal.go:78](https://github.com/rohanthewiz/grmob/blob/master/core/modal.go#L78)</small>

#### func OnDismiss

```go
func OnDismiss(fn func()) ModalProp
```

<small>[core/modal.go:54](https://github.com/rohanthewiz/grmob/blob/master/core/modal.go#L54)</small>

#### func Visible

```go
func Visible(v bool) ModalProp
```

<small>[core/modal.go:48](https://github.com/rohanthewiz/grmob/blob/master/core/modal.go#L48)</small>

### type ToastConfig

```go
type ToastConfig struct {
	Duration int // ms
	Style    *Style
}
```

<small>[core/toast.go:7](https://github.com/rohanthewiz/grmob/blob/master/core/toast.go#L7)</small>

### type ToastOpt

```go
type ToastOpt interface {
	Apply(*ToastConfig)
}
```

<small>[core/toast.go:3](https://github.com/rohanthewiz/grmob/blob/master/core/toast.go#L3)</small>

#### func Duration

```go
func Duration(ms int) ToastOpt
```

<small>[core/toast.go:31](https://github.com/rohanthewiz/grmob/blob/master/core/toast.go#L31)</small>

#### func UseToastStyle

```go
func UseToastStyle(s Style) ToastOpt
```

<small>[core/toast.go:37](https://github.com/rohanthewiz/grmob/blob/master/core/toast.go#L37)</small>

