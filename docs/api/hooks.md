# Package hooks

```go
import "github.com/rohanthewiz/grmob/hooks"
```

Package hooks is the layer of conveniences built on core's state slots: effects, timers, memoisation, a reducer, and read-only views of the things the host reports asynchronously (audio status, location, lifecycle, permission decisions).

Everything here is built from [core.NewState](core.md#func-newstate) and [core.Context.OnClose](core.md#func-context-onclose) and could be written by an app instead. What the package adds is the part that is easy to get wrong — deciding when to re-run, and cleaning up when the context closes.

## Slots, and therefore order

Every hook here occupies one or more of the calling context's slots, and slots are positional (see [core.NewState](core.md#func-newstate)). So the rules of hooks apply to this package exactly as they do to core's: call them unconditionally, in the same order, on every pass. A hook behind an \`if\` does not merely skip itself; it shifts every hook after it onto the wrong slot.

## Dependencies

	UseEffect(ctx, fn)             every pass
	UseEffect(ctx, fn, a, b)       when a or b changes from the last pass
	UseEffect(ctx, fn)             with no deps and a Close registration:
	                               the once-and-teardown shape

Comparison is by value equality on the dependency list. A dependency that is a function value or a freshly built slice compares unequal every pass, which turns a conditional effect into an unconditional one — silently, since the effect still does the right thing, just far more often than intended.

## Timers hold the latest closure

[UseInterval](#func-useinterval) and [UseTimeout](#func-usetimeout) re-bind their function on every pass, so the callback that eventually fires is the one from the most recent render and sees current state rather than the state of the pass that started the timer. Both stop themselves when the context closes, which is what makes a ticker inside a screen safe to leave to the navigator.

## Index

- [`func UseAudio`](#func-useaudio)
- [`func UseEffect`](#func-useeffect)
- [`func UseHeading`](#func-useheading)
- [`func UseInterval`](#func-useinterval)
- [`func UseLifecycle`](#func-uselifecycle)
- [`func UseLocation`](#func-uselocation)
- [`func UseLocationWhen`](#func-uselocationwhen)
- [`func UseMemo`](#func-usememo)
- [`func UsePermission`](#func-usepermission)
- [`func UsePermissionLive`](#func-usepermissionlive)
- [`func UseReducer`](#func-usereducer)
- [`func UseTimeout`](#func-usetimeout)
- [`type Debouncer`](#type-debouncer)
    - [`func UseDebounce`](#func-usedebounce)
    - [`func (*Debouncer) Call`](#func-debouncer-call)
    - [`func (*Debouncer) Cancel`](#func-debouncer-cancel)
    - [`func (*Debouncer) Pending`](#func-debouncer-pending)

## Functions

### func UseAudio

```go
func UseAudio(ctx *core.Context) core.AudioStatus
```

UseAudio returns the player's current status and re-renders the app on every change to it — a position tick, a pause, an error. It is how a screen with transport controls stays live:

	status := hooks.UseAudio(ctx)
	label := "Play"
	if status.State == core.AudioPlaying { label = "Pause" }
	core.Button(label, core.AudioToggle)

The subscription is taken on the hook's first render and released when the context tree is closed (ctx.Close, normally via render.Manager.Close), not when the component leaves the view tree — hooks have no unmount signal today, the same limit UseInterval carries. The cost of a stale subscription is one redundant RequestRender per status tick, coalesced by the manager into a pass that diffs to nothing; it is not a leak of anything the user can see.

The status itself is not copied into a hook slot: core keeps one record for the process (core.CurrentAudioStatus), and reading it at render time is what makes every subscriber see the same tick. The slot only remembers that this component already subscribed, so re-renders do not stack subscriptions.

<small>[hooks/audio.go:42](https://github.com/rohanthewiz/grmob/blob/master/hooks/audio.go#L42)</small>

### func UseEffect

```go
func UseEffect(ctx *core.Context, effect func(), deps ...any)
```

UseEffect runs effect when the hook first mounts and again whenever deps change between renders (compared with reflect.DeepEqual). With no deps it runs exactly once for the lifetime of the slot.

The effect runs on its own goroutine so a slow effect cannot stall the render pass; anything it changes via State.Set reaches the screen through the normal RequestRender → push-channel path.

<small>[hooks/effect.go:37](https://github.com/rohanthewiz/grmob/blob/master/hooks/effect.go#L37)</small>

### func UseHeading

```go
func UseHeading(ctx *core.Context) core.Heading
```

UseHeading turns the compass on for as long as this component is mounted, and re-renders on every meaningful change to the reading:

	h := hooks.UseHeading(ctx)
	if !h.Available && h.Received {
	    return comps.EmptyState{Hint: "This device has no compass"}
	}
	return comps.Compass{Heading: h.Magnetic, ShowDegrees: true}

#### What it owns

Two things, taken together on the first render and released together on close: a core.OnHeading subscription, and one reference on the sensor (core.StartHeading). The reference is the half that matters — the sensor costs battery while it runs, so a screen that shows a compass must be the thing that turns it off, and "while this component is mounted" is the only scope that reliably describes when nobody is looking at it any more.

Because core refcounts Start/Stop, two components may each call UseHeading and each release independently; the magnetometer stops when the second one lets go, not the first.

#### The mount/unmount limit, and why it is worse here

The subscription is taken on the hook's first render and released when the context tree is closed (ctx.Close, normally via render.Manager.Close), not when the component leaves the view tree — hooks have no unmount signal today, the same limit UseAudio and UseInterval carry.

For those two the cost is a redundant render that diffs to nothing. Here it is a sensor left running, which is a battery cost a user can measure, so it is worth saying plainly what the workaround is: put a compass on its own navigation route. A route's frame has its own cleanup registry (see core/cleanup.go) and popping it closes that registry, which releases the reference for real.

#### One redundant render at mount

The subscription is taken \*before\* core.StartHeading, so the hook hears the sensor coming on and asks for a render it does not strictly need: the pass that mounted it already read the post-Start value. That extra pass diffs to nothing, and the ordering it buys is worth more than it costs.

The alternative — start, then subscribe — drops any reading that lands in the gap, and the reading most likely to land there is the one that matters most: a host with no compass answers \`available: false\` immediately and then never speaks again. Missing that single event leaves a screen waiting on a reading that is never coming, with nothing in any log. Missing a heading tick costs 66 milliseconds of staleness that the next tick repairs.

#### Why the reading is not stored in the slot

core keeps one record for the process (core.CurrentHeading) and reading it at render time is what makes every subscriber see the same tick. The slot only remembers that this component already started the sensor, so re-renders do not stack references — which, with refcounting, would be a leak that never reaches zero.

<small>[hooks/heading.go:78](https://github.com/rohanthewiz/grmob/blob/master/hooks/heading.go#L78)</small>

### func UseInterval

```go
func UseInterval(ctx *core.Context, fn func(), interval time.Duration)
```

UseInterval invokes fn every interval for as long as the app lives. The ticker starts on the hook's first render; later renders only refresh the callback closure, so ticks always run the latest one (with the current render's state captures) rather than the closure from the mount render. The interval duration itself is fixed by the first render — a changed duration on a re-render is ignored, matching the original behavior.

The ticker is owned by the context tree: it stops when the tree is closed (ctx.Close, normally reached via render.Manager.Close), not when the component leaves the view tree — hooks have no unmount signal today.

<small>[hooks/interval.go:45](https://github.com/rohanthewiz/grmob/blob/master/hooks/interval.go#L45)</small>

### func UseLifecycle

```go
func UseLifecycle(ctx *core.Context) core.LifecycleState
```

UseLifecycle returns whether the app is on screen and re-renders the app on every transition. It is how a screen reacts to being foregrounded — the reconnect-on-resume case that put the event on the roadmap:

	state := hooks.UseLifecycle(ctx)
	hooks.UseEffect(ctx, func() { if state == core.LifecycleActive { conn.Resume() } }, state)

A component that only needs to \*act\* on the transition, not re-render for it, can subscribe with core.OnLifecycle from wherever it owns the connection instead; this hook is for the tree.

The subscription is taken on the hook's first render and released when the context tree is closed (ctx.Close, normally via render.Manager.Close), not when the component leaves the view tree — hooks have no unmount signal today, the same limit UseInterval and UseAudio carry. The cost of a stale subscription is one redundant RequestRender per transition, coalesced by the manager into a pass that diffs to nothing.

The state itself is not copied into a hook slot: core keeps one record for the process (core.CurrentLifecycle), and reading it at render time is what makes every subscriber see the same transition. The slot only remembers that this component already subscribed.

<small>[hooks/lifecycle.go:41](https://github.com/rohanthewiz/grmob/blob/master/hooks/lifecycle.go#L41)</small>

### func UseLocation

```go
func UseLocation(ctx *core.Context) core.Location
```

UseLocation turns the device's positioning on for as long as this component is mounted, and re-renders when the fix meaningfully changes:

	loc := hooks.UseLocation(ctx)
	switch {
	case !loc.Received:  return comps.Skeleton{}          // acquiring
	case !loc.Available: return comps.EmptyState{Hint: loc.Error}
	default:             return comps.StaticMap{Lat: loc.Lat, Lng: loc.Lng}
	}

It is UseHeading with a different sensor, and every argument in that hook's doc applies here — the refcounted Start/Stop, the subscribe-before-start ordering, why the reading is not kept in the slot. What follows is only what differs, and the two differences are the two expensive things about location.

#### Ask for the permission first

Every platform gates location, and the dialog is the expected path rather than an exception. This hook does not ask: it starts the sensor, the host asks the OS if it must, and a refusal comes back as Available: false with a reason. Which means that on an \*undecided\* permission, mounting a component that calls this hook is what puts the system dialog on screen — a side effect of a screen appearing, which is the surest way to spend an app's one chance at a yes.

So a screen that can be reached cold draws the permission's state beside the fix, and asks from a gesture:

	status := hooks.UsePermissionLive(ctx, permission.Location)
	loc := UseLocation(ctx)          // unconditionally — see below

	switch status {
	case permission.Granted:     return readout(loc)
	case permission.Prompt:      return askButton()      // permission.Request
	case permission.Denied:      return settingsHint()
	case permission.Unavailable: return nil
	default:                     return comps.Skeleton{}
	}

#### Both hooks run on every pass, and the branch is about drawing

This example used to read \`case permission.Granted: return mapScreen(ctx)\` with the hook inside mapScreen, which is a hook inside a conditional. Hook slots are handed out by call position (core.NewState), so an arm turning on or off moves every slot after it and the component starts reading somebody else's state. core/debug.go's cursor audit names that failure, and examples/tutorial's TestMain turns the audit on — which is how the shape got corrected: lesson 4.12 could not be written the way this doc described it.

The consequence is that mounting the screen starts the sensor. What gates that is the \*route\*: this hook's reference is released when the route frame is popped, so a screen the user navigated to is a screen the user asked for. A feature that must not ask until a tap goes behind a button that navigates.

A screen that cannot be its own route — a settings pane with a map preview beside the permission toggle — wants UseLocationWhen instead: the same one slot on every pass, with the sensor following a flag the caller owns.

The hook is deliberately not the thing that enforces a permission check. A hook that refused to start until a permission was granted would be a second authorization policy living in the wrong package, and it would be wrong for the app that genuinely wants the OS to ask at that moment — a "find my nearest branch" button, where the dialog \*is\* the response to the tap.

#### A refusal is not the end of it

The grant usually arrives after this hook has already run, because the tap that asks for it is on this screen. Both native hosts keep a refused start open and re-arm it when the permission answer changes, so no remount is needed — see LocationSensor.kt's awaitingPermission. The re-armed start says \`acquiring: true\` before any fix exists (core.LocationAcquiring), so the record goes back to Active-with-nothing-received rather than carrying the old reason through the whole acquisition window: the example above draws its Skeleton arm, which is what it always meant to.

#### Leaving the screen has to actually release it

The mount/unmount limit UseHeading describes is the same here and the cost is higher: a hook's subscription is released when the context tree is closed rather than when the component leaves the view tree, so a GPS left running is a battery cost a user can measure and attribute to this app.

The workaround is the same and the reason to take it seriously is not: put a location-reading screen on its own navigation route. A route's frame has its own cleanup registry (core/cleanup.go), and popping it releases the reference for real.

<small>[hooks/location.go:162](https://github.com/rohanthewiz/grmob/blob/master/hooks/location.go#L162)</small>

### func UseLocationWhen

```go
func UseLocationWhen(ctx *core.Context, want bool) core.Location
```

UseLocationWhen is UseLocation with a switch: the sensor runs while \`want\` is true and is released when it goes false, and the hook occupies ONE slot either way.

	status := hooks.UsePermissionLive(ctx, permission.Location)
	loc := hooks.UseLocationWhen(ctx, status == permission.Granted)

#### What it is for, which is not saving a Start call

UseLocation's own doc has a paragraph about mounting a screen being what puts the system dialog on screen, and an instruction: put such a screen behind a navigation route, because the route frame is what releases the reference. That is honest and it costs a screen — a settings pane that shows a map \*preview\* beside a permission toggle cannot be its own route, and today it either prompts on mount or does not use the hook.

The alternative a caller reaches for is the one thing the hook rules forbid:

	if granted {                       // WRONG
	    loc = hooks.UseLocation(ctx)
	}

Hook slots are handed out by call position (core.NewState), so an arm turning on or off moves every slot after it and the component starts reading somebody else's state — core/debug.go's cursor audit reports exactly this. The flag is the same intent with the call site held still: one NewState, every pass, whatever the answer.

#### It is still not an authorization policy

The flag is the caller's, and it does not have to be a permission — "the map tab is the visible one", "the user asked to be followed", "the app is in the foreground" are all it. This hook has no opinion about what makes \`want\` true, which is the same refusal UseLocation makes for the same reason: a hook that read a permission itself would be a second authorization policy living in the wrong package, and wrong for the app that wants the OS to ask at that moment.

#### Flipping it is cheap, and stopping is real

A false flag releases this slot's reference through the same path a Close takes, so the GPS goes off when the last holder lets go — which is the release UseLocation's own doc says a mounted component cannot do without a route change. A true flag re-subscribes and starts again; core.StartLocation is reference-counted, so a sibling still holding it keeps the sensor up and this slot merely rejoins.

What does not come back is anything the sensor said while the flag was false: core keeps the last fix, so the record a re-enabled hook reads is the last position the device reported, which may be old. Location.Received and Location.Active are what say whether it is being refreshed.

<small>[hooks/location.go:217](https://github.com/rohanthewiz/grmob/blob/master/hooks/location.go#L217)</small>

### func UseMemo

```go
func UseMemo[T any](ctx *core.Context, compute func() T, deps ...any) T
```

UseMemo returns the result of compute, recomputing it only when deps change between renders (compared with reflect.DeepEqual). With no deps it computes exactly once for the lifetime of the slot — the same mount-once rule UseEffect follows for a depless effect.

It exists for work that is expensive relative to a render pass — parsing, sorting or filtering a large slice, building a derived index — since a render function is re-run in full on every pass:

	visible := hooks.UseMemo(ctx, func() []Todo {
	    return filterAndSort(todos.Get(), filter.Get())
	}, todos.Get(), filter.Get())

Two things it deliberately is not:

  - It is not a correctness tool. compute must be pure, and callers must treat the returned value as read-only — the same value is handed back on every cache hit, so mutating it corrupts later renders.
  - There is no UseCallback counterpart. Memoizing a closure only pays off in a framework that skips subtrees on unchanged prop identity; here the reconciler diffs the rendered tree instead, so a stable closure buys nothing.

compute runs inline on the render goroutine (not on its own goroutine like UseEffect) because its result is needed to build this pass's view.

<small>[hooks/memo.go:51](https://github.com/rohanthewiz/grmob/blob/master/hooks/memo.go#L51)</small>

### func UsePermission

```go
func UsePermission(ctx *core.Context, p permission.Permission) permission.Status
```

UsePermission reports what the platform currently says about p, checking it once on mount and re-rendering whenever the answer changes:

	switch hooks.UsePermission(ctx, permission.Location) {
	case permission.Granted:
	    return mapView(ctx)
	case permission.Prompt:
	    return comps.Button{Label: "Use my location",
	        OnClick: func() { permission.Request(permission.Location) }}
	case permission.Denied:
	    return comps.EmptyState{Hint: "Location is off — turn it on in Settings"}
	default: // Unknown while the check is in flight, Unavailable on a device
	         // that cannot do it at all
	    return comps.Skeleton{}
	}

#### It checks and does not ask

The opening call is permission.Check, never permission.Request, and the difference is the whole reason the two are separate functions. A Check prompts nothing, so it is safe in a render pass; a Request run from one would put the OS dialog on screen as a side effect of drawing, at a moment the user did nothing to cause. That is the surest way to a permanent refusal, and a permanent refusal is much harder to undo than a permission never asked for. Requesting stays the caller's, from a gesture.

#### What the branches may not contain

Hooks. The switch above returns different subtrees per status, and a hook called inside one of those arms is a hook inside a conditional: slots are handed out by call position (core.NewState), so the arm turning on or off moves every slot after it and the component reads state that belongs to something else. core/debug.go's cursor audit reports it by name.

\`mapView(ctx)\` above is therefore only safe if mapView draws and does not hook. A branch that needs a sensor calls the sensor's hook \*beside\* this one, unconditionally, and uses the status to decide what to draw with it — see hooks.UseLocation, whose own doc used to get this wrong, and lesson 4.12 in examples/tutorial, which is the pair written out.

#### Unknown is a state the caller has to draw

The check is asynchronous, so the first pass returns permission.Unknown and a later one returns the platform's answer. That is not a wart to hide: on a slow shell the gap is visible, and a screen that treats Unknown as Denied flashes an error state before the permission it already has arrives. Draw a placeholder for it, as the example does.

A headless run — a Go test, a static export, any host that registered no system-event sink — resolves to Unavailable immediately rather than sitting on Unknown, so an app under test takes the "cannot do this" branch instead of the spinner. See permission.send for why that is the honest answer.

#### Coming back from Settings

A user can grant or revoke a permission in the system settings and return to the app, and no platform tells the app that happened. This hook does not re-check on foreground: it reports what the record says and asks once, on mount. UsePermissionLive is this hook plus the re-check, and its doc carries the argument for why the re-check is owned by the permission rather than by the screen — which is the objection that used to be written here, and the reason it no longer stands.

#### The mount/unmount limit

The subscription is taken on the hook's first render and released when the context tree is closed (ctx.Close, normally via render.Manager.Close), not when the component leaves the view tree — hooks have no unmount signal today, the same limit UseAudio, UseInterval and UseHeading carry. The cost here is the mild one: a subscription that outlives its screen asks for a render that diffs to nothing. Nothing is left running, because a permission query is a question and not a sensor.

<small>[hooks/permission.go:94](https://github.com/rohanthewiz/grmob/blob/master/hooks/permission.go#L94)</small>

### func UsePermissionLive

```go
func UsePermissionLive(ctx *core.Context, p permission.Permission) permission.Status
```

UsePermissionLive is UsePermission that also re-checks p every time the app returns to the foreground.

	switch hooks.UsePermissionLive(ctx, permission.Camera) {
	case permission.Granted:
	    return scanner(ctx)
	case permission.Denied:
	    return comps.EmptyState{
	        Hint:   "Camera is off",
	        Action: comps.Button{Label: "Open Settings", OnTap: openSettings},
	    }
	...
	}

That Denied branch is the whole reason this exists. It sends the user to the system settings, they flip the switch, and they come back — and with UsePermission the screen still says the camera is off, because nothing on any of the three platforms announces a permission change and the hook's one check already happened on mount. The screen is wrong until something else re-renders it, and the button that fixed the problem is the thing still telling the user it is broken.

#### What it costs, and why it is not per-screen

The re-check is permission.WatchForeground, which reference-counts by permission rather than by caller: five screens watching the camera produce one check per resume between them, and an app with no live watcher takes no lifecycle subscription at all. See that function's file for the argument — it is the one that used to keep this hook from existing.

A resume that changed nothing costs one system event out and one host event back, and stops there: permission.set notifies only on a change, so no subscriber runs and no render is requested. Only the resume that actually changed something reaches the screen.

#### The same rule about branches applies

\`scanner(ctx)\` above must not call hooks, for the reason UsePermission's "What the branches may not contain" gives: a hook in a status arm shifts every slot after it. A sensor's hook goes beside this call, not inside a branch of it.

#### Use this one by default

UsePermission is the narrower tool and stays for the screens that want it — a debug readout, a settings row rendered inside a sheet that cannot be left without unmounting it — but a screen that draws a Denied state and offers a way out of it wants this. The extra cost over UsePermission is one lifecycle subscription per process and one round trip per resume per kind.

The watch is released when the context tree is closed, the same mount/unmount limit UsePermission carries and for the same reason: hooks have no unmount signal. A watch that outlives its screen costs a check per resume that diffs to nothing.

<small>[hooks/permission.go:192](https://github.com/rohanthewiz/grmob/blob/master/hooks/permission.go#L192)</small>

### func UseReducer

```go
func UseReducer[S any, A any](ctx *core.Context, reducer func(S, A) S, initial S) (S, func(A))
```

UseReducer holds state that evolves through named actions instead of raw writes. It returns the current state for this render and a dispatch function; dispatch applies reducer to the live state and requests a render, exactly as State.Set does.

	type action int
	const (increment action = iota; reset)

	count, dispatch := hooks.UseReducer(ctx, func(s int, a action) int {
	    switch a {
	    case increment:
	        return s + 1
	    case reset:
	        return 0
	    }
	    return s
	}, 0)

	core.Button("+1", func() { dispatch(increment) })

Rules the implementation depends on:

  - reducer must be pure and must return a \*new\* state value rather than mutating the one it is given — earlier renders still hold the old value, and the reconciler diffs against the tree they produced.
  - reducer must not dispatch. It runs while the record's mutex is held, so a re-entrant dispatch deadlocks. Chain actions from an event handler or a hooks.UseEffect instead.
  - initial is evaluated on every render but only the first render's value is kept, the same as core.NewState.

dispatch is safe to call from any goroutine — timers, network handlers, a native event thread — and is stable enough to hand to child views for the life of the app: every render returns a fresh closure, but all of them write through the same record.

<small>[hooks/reducer.go:65](https://github.com/rohanthewiz/grmob/blob/master/hooks/reducer.go#L65)</small>

### func UseTimeout

```go
func UseTimeout(ctx *core.Context, fn func(), delay time.Duration)
```

UseTimeout invokes fn once, delay after the hook's first render. Renders while the timer is pending only refresh the closure (the fire runs the latest one); renders after it has fired do nothing. That last part is a deliberate change from the store-keyed version, which forgot a fired timeout and therefore re-armed it on whichever render happened to come next — a repeat schedule driven by unrelated render timing.

A pending timer is cancelled when the context tree is closed, so a Manager shutdown cannot leak a late fn call into a dead app.

<small>[hooks/interval.go:127](https://github.com/rohanthewiz/grmob/blob/master/hooks/interval.go#L127)</small>

## Types

### type Debouncer

```go
type Debouncer struct {
	// contains filtered or unexported fields
}
```

Debouncer collapses a burst of calls into one: each Call cancels the previous pending call and re-arms the delay, so only the last one in a run actually fires. It is what a search box, an autosave, or a recompute-on-resize wants — an expensive reaction driven by an input that changes far faster than the reaction is worth running.

#### Why this is not UseTimeout

UseTimeout arms exactly once per mount and then stays fired: a render after it has fired does nothing, deliberately, because the store-keyed version it replaced used to re-arm itself on whichever unrelated render came next. That is the right contract for "do X once, shortly after this appears", and it is the wrong one here — a debounce is defined by re-arming. So the two hooks are separate rather than one hook with a flag; they differ in the one thing the type is about.

	                Call    Call  Call            (a burst of keystrokes)
	                  │       │     │
	timer  ───────────x───────x─────┬──── delay ──▶ fn()
	               (stopped)      (armed)

#### Threading

Call may arrive from any goroutine — it normally arrives from a native event callback — and the timer fires on its own. mu guards the whole record; fn itself runs outside the lock so a slow callback cannot block the next Call, and so an fn that re-enters Call (a debounced action that schedules another) does not deadlock.

<small>[hooks/debounce.go:38](https://github.com/rohanthewiz/grmob/blob/master/hooks/debounce.go#L38)</small>

#### func UseDebounce

```go
func UseDebounce(ctx *core.Context, delay time.Duration) *Debouncer
```

UseDebounce returns the Debouncer stored in this component's hook slot, refreshing its delay from the current render.

Like every hook it must be called unconditionally, in a stable order, on every pass: it consumes a cursor slot through core.NewState (see UseInterval for why a hook reserves its slot that way rather than by bumping the cursor).

	d := hooks.UseDebounce(ctx, 300*time.Millisecond)
	...
	comps.SearchField{
	    Value: query.Get(),
	    OnChange: func(s string) {
	        query.Set(s)                       // the field is controlled: now
	        d.Call(func() { runSearch(s) })    // the query is not: in 300ms
	    },
	    OnSubmit: func() { d.Cancel(); runSearch(query.Get()) },
	}

Unlike UseInterval, whose period is fixed by the first render, the delay is re-read every pass: it is held on the record rather than baked into a running ticker, so a delay driven by state (a "search as I type" preference) takes effect on the next Call.

A pending call is dropped when the context tree is closed, so a Manager shutdown cannot land a late callback in a dead app.

<small>[hooks/debounce.go:79](https://github.com/rohanthewiz/grmob/blob/master/hooks/debounce.go#L79)</small>

#### func (*Debouncer) Call

```go
func (d *Debouncer) Call(fn func())
```

Call schedules fn to run once the delay has passed with no further Call, replacing whatever was pending.

With a delay of zero or less fn runs synchronously, right here. That is the honest reading of "debounce by nothing", and it means a caller can turn the debounce off from state without a second code path. No render is requested in that case: the call is indistinguishable from the caller having invoked fn itself, and it is already inside whatever event handling led here. The delayed path does request one, because a timer firing with no native event in flight has no other way to reach the screen (the same reason UseTimeout and UseInterval do it).

A nil fn is a no-op and does \*not\* cancel a pending call — cancelling is Cancel's job, and reading "schedule nothing" as "unschedule everything" would make a guard like \`d.Call(handlerFor(mode))\` silently destructive when the handler happens to be nil.

<small>[hooks/debounce.go:120](https://github.com/rohanthewiz/grmob/blob/master/hooks/debounce.go#L120)</small>

#### func (*Debouncer) Cancel

```go
func (d *Debouncer) Cancel()
```

Cancel drops a pending call. Use it when something has superseded the debounced work outright — a submit that runs the search now, a screen leaving, a field cleared.

It makes no promise about a call already in flight: time.Timer.Stop reports false once the timer has fired, and by then fn may already be running on the timer goroutine. Cancel is therefore "no \*further\* call", not "undo". Work that must not run twice needs its own guard, as it would with any timer.

<small>[hooks/debounce.go:153](https://github.com/rohanthewiz/grmob/blob/master/hooks/debounce.go#L153)</small>

#### func (*Debouncer) Pending

```go
func (d *Debouncer) Pending() bool
```

Pending reports whether a call is scheduled and has not fired yet. It is for the UI that wants to say so — a "searching…" hint that appears the moment typing stops rather than when the request goes out.

It is a snapshot, not a lock: the timer can fire in the instant after it returns true.

<small>[hooks/debounce.go:168](https://github.com/rohanthewiz/grmob/blob/master/hooks/debounce.go#L168)</small>

