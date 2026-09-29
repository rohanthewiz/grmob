# Package core — Events & focus

```go
import "github.com/rohanthewiz/grmob/core"
```

Event props, host and system events, focus refs and focus order, and scrolling a node into view.

One of 11 topic pages of [package core](core.md), which has the package overview and an index of every topic. This page documents the declarations in `core/event.go`, `core/behavioral_props.go`, `core/host_events.go`, `core/sys_events.go`, `core/focus.go`, `core/focus_order.go`, `core/scroll_to.go`.

## Index

- [`func DismissKeyboard`](#func-dismisskeyboard)
- [`func Focus`](#func-focus)
- [`func FocusNext`](#func-focusnext)
- [`func FocusPrevious`](#func-focusprevious)
- [`func HasSystemEventHandler`](#func-hassystemeventhandler)
- [`func OnHostEvent`](#func-onhostevent)
- [`func ReceiveHostEvent`](#func-receivehostevent)
- [`func ScrollIntoView`](#func-scrollintoview)
- [`func SendSystemEvent`](#func-sendsystemevent)
- [`func SetSystemEventHandler`](#func-setsystemeventhandler)
- [`func UseFocusOrder`](#func-usefocusorder)
- [`type BehaviorProp`](#type-behaviorprop)
    - [`func FocusTarget`](#func-focustarget)
    - [`func On`](#func-on)
    - [`func OnBack`](#func-onback)
    - [`func OnBlur`](#func-onblur)
    - [`func OnClick`](#func-onclick)
    - [`func OnEscape`](#func-onescape)
    - [`func OnFocus`](#func-onfocus)
    - [`func OnLongPress`](#func-onlongpress)
    - [`func OnTouch`](#func-ontouch)
    - [`func PressKeepsFocus`](#func-presskeepsfocus)
    - [`func ScrollTarget`](#func-scrolltarget)
- [`type FocusRef`](#type-focusref)
    - [`func UseFocusRef`](#func-usefocusref)

## Functions

### func DismissKeyboard

```go
func DismissKeyboard(ctx *Context)
```

DismissKeyboard releases the input focus, putting the software keyboard away, as of the next render pass.

This is the other half of the old backlog item OnFocus/OnBlur opened: a tap on the background of a form can now actually close the keyboard.

	core.Box(
	    core.OnClick(func() { core.DismissKeyboard(ctx) }),
	    form,
	)

It takes a Context where Focus does not because a dismiss names no node, and therefore has no ref to carry one. Reaching for a package-level global instead would recreate exactly the bug Context's shared-pointer block documents: two apps in one process sharing one keyboard.

Unlike Focus this reaches every focusable leaf, because the field the user tapped into is one Go was never told about — the framework does not wire OnFocus unless an app asks for it. Each leaf stamps "blur" and each renderer releases focus only if that leaf actually holds it, so exactly one of them does anything.

<small>[core/focus.go:286](https://github.com/rohanthewiz/grmob/blob/master/core/focus.go#L286)</small>

### func Focus

```go
func Focus(ref *FocusRef)
```

Focus puts the input focus — and with it the software keyboard — on ref's node, as of the next render pass.

Called from an event handler, as the imperative counterpart to OnFocus:

	core.Button("Next", func() { core.Focus(password) })

Calling it for a ref whose node is not currently in the tree does nothing visible: no node stamps "focus", so no renderer acts. The command still consumes an epoch, which is correct — it happened, it simply had no target on screen.

A nil ref is a no-op rather than a panic, matching FocusTarget.

<small>[core/focus.go:248](https://github.com/rohanthewiz/grmob/blob/master/core/focus.go#L248)</small>

### func FocusNext

```go
func FocusNext(ref *FocusRef)
```

FocusNext moves the input focus to the field after ref in its order.

This is what the keyboard's Next action runs, and it is callable directly for the cases a keyboard cannot reach — a "Next" button drawn above the keyboard, a barcode scan that fills one field and should land in the next:

	core.Button("Next", func() { core.FocusNext(current) })

It takes the field to move \*from\* rather than reading "the focused field", because Go does not reliably know which field that is: the framework wires OnFocus only where an app asked for it, so the field the user tapped into is one Go was never told about (see focus.go). Naming the source is honest about that, and every caller has it — the keyboard action is stamped on a known field, and an app-drawn toolbar tracks the current field with OnFocus if it wants one.

At the end of the order, or for a ref in no order at all, this does nothing. A nil ref is a no-op, matching Focus and FocusTarget.

<small>[core/focus_order.go:236](https://github.com/rohanthewiz/grmob/blob/master/core/focus_order.go#L236)</small>

### func FocusPrevious

```go
func FocusPrevious(ref *FocusRef)
```

FocusPrevious moves the input focus to the field before ref in its order. See FocusNext for the shape and for why the source field is named.

It has no keyboard action behind it on any platform here: neither the Android IME nor the iOS keyboard offers a "previous" key, and SwiftUI gives no input-accessory toolbar for free. This exists for the toolbar an app draws itself, above a KeyboardAware region, which is where a back-and-forth pair of arrows actually belongs.

<small>[core/focus_order.go:248](https://github.com/rohanthewiz/grmob/blob/master/core/focus_order.go#L248)</small>

### func HasSystemEventHandler

```go
func HasSystemEventHandler() bool
```

HasSystemEventHandler reports whether a host has registered a sink.

It exists for the one caller that has to behave differently when nobody is listening rather than merely have its event dropped: the permission package, where "there is no platform to ask" is a real answer a screen must draw (permission.Unavailable) and not the same thing as "the user has not decided yet". Every other sender is fire-and-forget and correctly does not care — a toast with no screen to draw on is a no-op, not a state.

<small>[core/sys_events.go:42](https://github.com/rohanthewiz/grmob/blob/master/core/sys_events.go#L42)</small>

### func OnHostEvent

```go
func OnHostEvent(name string, fn func(data map[string]any)) (cancel func())
```

OnHostEvent subscribes fn to host events named name. The returned function cancels the subscription; calling it more than once is harmless.

Subscriptions are process-wide, like the system-event handler and for the same reason: the thing on the far side of the channel is one physical device with one audio output, one keystore, one location, so there is no context tree to scope them to. A component that subscribes during render must therefore guard against subscribing again on the next pass — hooks.UseAudio shows the pattern (a hook slot remembers that it did).

<small>[core/host_events.go:60](https://github.com/rohanthewiz/grmob/blob/master/core/host_events.go#L60)</small>

### func ReceiveHostEvent

```go
func ReceiveHostEvent(name string, data map[string]any)
```

ReceiveHostEvent delivers one event from the host. Names core owns are consumed first; then every subscriber for the name runs, outside the registry lock so a subscriber may subscribe or cancel from inside its own handler without deadlocking.

An event nobody consumes is logged rather than dropped silently — unlike an unknown system event, which a host drops because a newer app may legitimately send what an older shell does not understand, an unknown host event means the shell is sending traffic the app never asked for, which is worth a line in the log during development.

<small>[core/host_events.go:93](https://github.com/rohanthewiz/grmob/blob/master/core/host_events.go#L93)</small>

### func ScrollIntoView

```go
func ScrollIntoView(ctx *Context, name string)
```

ScrollIntoView brings the node named name into view, as of the next render pass. Called from an event handler, or from anywhere a State.Set may be: it requests a pass itself.

A name no node carries scrolls nothing, and the command still takes an epoch: it happened, and had no target on screen, as a Focus on an absent ref does.

<small>[core/scroll_to.go:103](https://github.com/rohanthewiz/grmob/blob/master/core/scroll_to.go#L103)</small>

### func SendSystemEvent

```go
func SendSystemEvent(name string, data map[string]any)
```

SendSystemEvent delivers one event to the host, synchronously on the caller's goroutine. The read is under RLock so senders never contend with each other, only with the (rare) handler swap.

<small>[core/sys_events.go:51](https://github.com/rohanthewiz/grmob/blob/master/core/sys_events.go#L51)</small>

### func SetSystemEventHandler

```go
func SetSystemEventHandler(fn func(name string, data map[string]any))
```

SetSystemEventHandler installs the host's sink for system events. Passing nil detaches it, after which SendSystemEvent drops events silently — the right behavior for a headless run, where there is no screen to draw on.

<small>[core/sys_events.go:28](https://github.com/rohanthewiz/grmob/blob/master/core/sys_events.go#L28)</small>

### func UseFocusOrder

```go
func UseFocusOrder(ctx *Context, refs ...*FocusRef)
```

UseFocusOrder declares the order the input focus walks through a set of fields. Call it in the component that renders those fields, above them:

	core.UseFocusOrder(ctx, email, password, confirm)

Two things follow from it. core.FocusNext and core.FocusPrevious can move relative to any ref in the list; and every field but the last advertises the keyboard's "next" action, which advances to the field after it. The last field advertises nothing new, so a form's final field keeps its own submit action — see FocusTarget for the rule when a field has both.

#### Why "above them" is not merely style

Membership is read while a field's props are stamped, so a field rendered before this call runs sees the \*previous\* pass's membership. Hooks belong at the top of a render function anyway, and the fields of a form are rendered by the component that owns their refs, so the natural shape is already the correct one — but a call moved into a child component that renders after the fields would stamp a form that never advances.

#### It reserves no hook slot

Despite the name, this is not a hook: everything it records lives on the refs, which are slot-stable already (see UseFocusRef), so there is nothing for a slot to hold. The Use prefix says where it belongs — inside a render function, on every pass — which is the part a caller has to get right. The consequence of not being a hook is only ever permissive: calling it conditionally is safe, where a real hook would drift the cursor.

A nil ref in the list is skipped rather than panicking, so \`core.UseFocusOrder(ctx, email, maybeRef, confirm)\` degrades to the order without it — the same tolerance MaybeProp and FocusTarget have. A ref belonging to a different app's context is skipped for the same reason focusState is per-app: two apps in one process must not share an order.

Listing a ref twice is allowed and the last position wins; there is no meaningful "field visited twice" and no reason to panic over a typo.

<small>[core/focus_order.go:101](https://github.com/rohanthewiz/grmob/blob/master/core/focus_order.go#L101)</small>

## Types

### type BehaviorProp

```go
type BehaviorProp interface {
	Apply(*Context, *Node)
}
```

BehaviorProp attaches event behavior to a node. Apply takes the rendering Context (unlike StyleProp) because registering the handler needs the context's callback registry — the registry is per-app state on the context tree, not a package global.

<small>[core/behavioral_props.go:7](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L7)</small>

#### func FocusTarget

```go
func FocusTarget(ref *FocusRef) BehaviorProp
```

FocusTarget marks the node it is applied to as ref's node.

It is an ordinary BehaviorProp, so it composes with OnFocus, OnBlur and everything else in the same argument list:

	core.Input(v, "", onChange, core.FocusTarget(email), core.OnBlur(...))

A nil ref returns a nil prop rather than panicking — leafNode and containerNode both skip a nil item (MaybeProp's contract), so \`core.FocusTarget(maybeRef)\` degrades to an unnamed field instead of crashing a render pass.

On a node the platform does not give input focus to, what a Focus does depends on the host:

	Web      the element is made a programmatic target (tabindex -1, out of
	         the Tab order) and focused, so a screen reader moves with it.
	         This is how a heading takes focus after a page-style change
	         (comps.Wizard's TitleRef).
	iOS      a Text moves VoiceOver's focus (AccessibilityFocusState); other
	         node types read nothing.
	Android  nothing: Compose offers no way for a node to take TalkBack's
	         focus, so the stamp lands and no renderer reads it.

core.Text takes style props only, so a Text is named by applying the prop to the node it renders: FocusTarget(ref).Apply(ctx, core.Text(…).Render(ctx)).

<small>[core/focus.go:196](https://github.com/rohanthewiz/grmob/blob/master/core/focus.go#L196)</small>

#### func On

```go
func On(event string, handler func()) BehaviorProp
```

<small>[core/behavioral_props.go:16](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L16)</small>

#### func OnBack

```go
func OnBack(handler func()) BehaviorProp
```

OnBack claims the platform's system back — Android's back button and back gesture — for as long as the node carrying it is on screen. While any such node is, a back press runs the innermost one's handler instead of the platform default; when none is, back does what the platform would have done anyway, which on Android is to leave the app.

core.Navigator attaches one to the route it shows whenever core.CanPop is true, so a pushed screen pops on back with no app code. comps.AppBar attaches its back arrow's action, and comps.Drawer its OnDismiss while open.

##### A prop, not a host event

The other things a shell reports without a callback — lifecycle, a deep link — arrive as host events. Back cannot, because Android has to know before the press whether the app will take it: OnBackPressedDispatcher consults its callbacks' enabled flags synchronously on the UI thread, and falls through to finishing the Activity if none is enabled. A host event reaches Go after that decision, so a shell built on one would need a second, Go→host "enabled" signal kept in step with the app's state. A prop already is that signal: its presence in the tree is the enabled flag, and the tree diff keeps the shell's copy current with no channel of its own.

##### Innermost wins

Compose's BackHandler gives priority to the handler registered most recently. Handlers composed in one pass register parent before child, so the innermost wins; one composed later — a drawer that has just opened — outranks every handler already on screen.

	Navigator route root  onBack = Pop            outermost, runs last
	  AppBar row          onBack = AppBar.OnBack
	  Drawer panel layer  onBack = OnDismiss      while Open; runs first

The one ordering this gets wrong is a parent that gains the prop after its descendants already hold one: it registers last and outranks them. Keep OnBack on nodes whose lifetimes nest the way their handlers should, which the three above do.

##### Its own callback IDs

The handler is an ordinary void callback on the wire, but its ID comes from a sequence of its own ("back\_cb\_N" rather than "cb\_N"). A second quick press is dispatched with the ID from before the first press's patches landed, and a shared sequence would have re-assigned that ID to a tap on the screen the first press revealed. See callbackRegistry.registerBack.

##### Hosts

	Android  RenderNode wraps the node in androidx's BackHandler. A node
	         hidden with Display none is not composed, so its handler is
	         inactive while hidden. A Modal needs none: the Dialog window
	         reports back through the Modal's own onDismiss. The manifest
	         opts in to predictive back, which BackHandler supports.
	iOS      nothing. There is no system back; the edge swipe belongs to a
	         UINavigationController, which the SwiftUI renderer does not use.
	Web      the browser's back button. While any node carrying the prop is
	         on screen, the runtime keeps one history entry of its own above
	         the page's; a back press consumes it and runs the innermost
	         handler (the last claimant in document order, which is the same
	         nesting Compose ranks by), and the entry is pushed again if a
	         claim is still on screen afterwards. An open Modal's onDismiss
	         counts as a claim, matching Android's Dialog. A page that owns
	         its history sets window.GrMobBrowserBack = false. htmlout does
	         not export the prop.

<small>[core/behavioral_props.go:124](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L124)</small>

#### func OnBlur

```go
func OnBlur(handler func()) BehaviorProp
```

OnBlur fires when the node loses input focus. See OnFocus for the pairing and the ordering caveat.

<small>[core/behavioral_props.go:214](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L214)</small>

#### func OnClick

```go
func OnClick(handler func()) BehaviorProp
```

<small>[core/behavioral_props.go:25](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L25)</small>

#### func OnEscape

```go
func OnEscape(handler func()) BehaviorProp
```

OnEscape claims the Escape key for as long as the node is on screen: a press on a hardware keyboard runs the innermost claimant's handler. It is for a layer that closes (comps.Drawer's open panel carries it beside OnBack), and an open core.Modal with an OnDismiss is a claimant without it, so every Dialog, ActionSheet, Menu and Lightbox closes on Escape too.

##### Not OnBack

Back is claimed by more than layers: core.Navigator claims it for a pushed route and comps.AppBar for its back arrow. Escape on a desktop page, or on an iPad keyboard, closes what is open over the screen and does not navigate, which is what every platform's own sheets and popovers do. So Escape reads its own claims, and a layer that wants both carries both.

Its callback IDs come from OnBack's sequence ("back\_cb\_N"), for OnBack's reason: a second quick press is dispatched with the ID from before the first press's patches landed.

##### Hosts

	Web      one window keydown listener; the innermost claimant (onEscape,
	         or an open Modal's onDismiss) in document order wins. A handler
	         under focus that owns the key and calls preventDefault (a
	         combobox clearing its active option) goes first. A modified
	         Escape is left alone. htmlout does not export the prop.
	Android  the Activity walks the tree for a bare KEYCODE_ESCAPE and runs
	         the last claimant in tree order that is on screen, the web's
	         rule (GrMobRuntime.handleKeyEvent). An open Modal is a Dialog
	         window, which takes the keys while it is up, so its content
	         wraps that window's callback and runs onDismiss (DialogEscape):
	         the platform maps Escape to back only with predictive back off,
	         and this shell opts in to it.
	iOS      a node carrying the prop gets an Escape keyboard shortcut on a
	         hidden button while it is on screen. Unverified: XCUITest's
	         Escape reached neither it nor SwiftUI's .cancelAction on the
	         simulator, though the same claim bound to a chord worked (see
	         TutorialKeyShortcutsUITests). A Modal has no claim of its own
	         there yet; whether a real iPad keyboard's Escape closes one
	         through SwiftUI's own dismissal is unchecked.

<small>[core/behavioral_props.go:174](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L174)</small>

#### func OnFocus

```go
func OnFocus(handler func()) BehaviorProp
```

OnFocus fires when the node becomes the input focus — a text field the user has tapped into, with the software keyboard on its way up.

OnBlur fires when that focus leaves. The pair is deliberately \*not\* a single bool-carrying handler: the two edges are almost never handled together (a form reveals errors on blur and does nothing on focus; a search box does the opposite), and one prop per edge lets a node carry only the edge it cares about instead of registering a callback to ignore half its calls.

Both ride the void-callback channel, exactly like OnClick — the edge itself is the whole payload, and "which node" is already answered by which callback ID the platform dispatches.

Focus is a leaf concern in practice: the renderers wire these on the text input node types, which are the only things a mobile platform gives focus to. The props are attachable to any node because BehaviorProp is uniform, but a Row carrying OnFocus will simply never hear from the native renderers.

Ordering note: the framework guarantees the edges are dispatched in the order they happened, but \*not\* that a blur on the field being left arrives before the focus on the field being entered — that ordering is the platform's, and Android and iOS do not agree on it. Handlers must therefore be independent: read the field the callback belongs to, never "the field that is focused now".

<small>[core/behavioral_props.go:208](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L208)</small>

#### func OnLongPress

```go
func OnLongPress(handler func()) BehaviorProp
```

OnLongPress fires after 500ms of held press without the finger lifting — the default on all three targets (UILongPressGestureRecognizer, Android's ViewConfiguration, and the DOM runtime's own timer).

A node may carry both OnClick and OnLongPress, and one gesture produces exactly one handler call: Compose's combinedClickable splits them natively, while SwiftUI and the DOM each suppress the tap that follows a fired long press. Which handler runs is decided by how long the press was held, never by both running.

Wired on all three targets, containers and leaves alike, including Button — which needs its own wiring on both natives, since a Button draws its own control rather than going through the generic gesture path.

<small>[core/behavioral_props.go:56](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L56)</small>

#### func OnTouch

```go
func OnTouch(handler func()) BehaviorProp
```

OnTouch fires the moment a finger (or pen, or mouse button) goes down on the node, before the press has resolved into a tap or a long press. Use it for immediate feedback — a sound, a highlight — not for the action itself: a press that slides off the node still fired this.

Web only today. The DOM runtime maps it to pointerdown; neither native renderer reads the prop, so a node carrying it on iOS or Android simply never hears from it. (It used to be worse: the DOM's event-name fallback derived "touch", which is not a DOM event either, so the prop attached a listener nothing could ever fire on any target.)

<small>[core/behavioral_props.go:39](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L39)</small>

#### func PressKeepsFocus

```go
func PressKeepsFocus() BehaviorProp
```

PressKeepsFocus makes a press on the node leave the input focus where it is: the field being typed into stays focused, and does not blur, while the node's OnClick runs. It is for a control that acts \*on\* a field from beside it: a clear or discard ✕, a toolbar's Bold, a "show password" eye.

	core.Box(core.PressKeepsFocus(), core.OnClick(discard), core.Text("✕"))

##### Why a prop and not a blur payload

A browser moves focus on mousedown and delivers the click on mouseup, so a press on a button beside a field blurs the field before the click arrives. A widget that commits on blur (comps.EditableGrid) then commits under a press meant to discard, and the ✕ leaves the tree under the pointer, so the click lands on nothing. The two ways out were a blur that says where focus went, so the widget could tell "moving to my own ✕" from "leaving", or a press that never moves focus. The second is one flag on the node that knows it is an accessory, needs no payload on OnBlur, and is the idiom the platform itself uses (preventDefault on mousedown is how every rich-text toolbar keeps its selection). The first would put a node path in every blur for the one widget that asks.

##### Hosts

	Web      a mousedown listener that calls preventDefault, which is what
	         stops the browser moving focus. mousedown and not pointerdown:
	         preventDefault on pointerdown does not stop Chrome focusing, and
	         a touch tap reaches the page as a compatibility mousedown after
	         the finger lifts, which the same listener catches. The node can
	         still be reached with Tab and pressed with Enter or Space: this
	         changes what a pointer does, not the keyboard.
	Android  nothing to do. A press on a clickable node does not take the
	         focus from a text field.
	iOS      nothing to do, for the same reason.
	htmlout  not exported; a static page has no fields to keep.

A no-arg flag, like StartAtEnd: a widget that wants the default writes no prop.

<small>[core/behavioral_props.go:255](https://github.com/rohanthewiz/grmob/blob/master/core/behavioral_props.go#L255)</small>

#### func ScrollTarget

```go
func ScrollTarget(name string) BehaviorProp
```

ScrollTarget names the node it is applied to, so ScrollIntoView can bring it into view. An empty name is no target.

<small>[core/scroll_to.go:73](https://github.com/rohanthewiz/grmob/blob/master/core/scroll_to.go#L73)</small>

### type FocusRef

```go
type FocusRef struct {
	// contains filtered or unexported fields
}
```

FocusRef names one focusable node so an app can put the cursor in it later.

It is deliberately opaque and carries no state of its own beyond the context it belongs to: identity is the whole point, and identity is the pointer. That is also why it must be stable across render passes — see UseFocusRef.

<small>[core/focus.go:130](https://github.com/rohanthewiz/grmob/blob/master/core/focus.go#L130)</small>

#### func UseFocusRef

```go
func UseFocusRef(ctx *Context) *FocusRef
```

UseFocusRef returns a FocusRef that is stable for the lifetime of this hook slot, which is what makes the ref usable as an identity:

	email := core.UseFocusRef(ctx)

	core.Input(v, "you@example.com", onChange,
	    core.FocusTarget(email),
	)
	core.Button("Next", func() { core.Focus(email) })

A hook rather than a bare constructor because a ref built inline in a render function is a new pointer every pass: FocusTarget would stamp one identity and the click handler would compare against another, so Focus would silently never match a node. Going through NewState both pins the pointer and reserves the cursor slot properly.

It lives in core rather than hooks because hooks imports core and not the reverse; the focus state it closes over is on Context.

<small>[core/focus.go:160](https://github.com/rohanthewiz/grmob/blob/master/core/focus.go#L160)</small>

