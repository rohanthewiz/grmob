# Native Android & iOS

The native targets run your Go app inside a thin platform shell: Go renders
and diffs; Kotlin/Swift apply patches to real platform views. The connection
is the `mobile` package — a gomobile-bindable bridge that narrows the
framework surface to strings, bools, and one single-method interface
(gomobile cannot bind function parameters, generics, or maps).

## The contract

Your app package registers itself in an `init`:

```go
func init() { mobile.Register(core.NewContext(), App) }

// gomobile only links a bound package that exports at least one bindable
// symbol; App (function-typed) is not bindable, so export something trivial:
func AppName() string { return "My App" }
```

The shell then drives the bridge:

```mermaid
sequenceDiagram
    participant S as Shell (Kotlin / Swift)
    participant B as mobile bridge
    participant M as render.Manager

    S->>B: SetDataDir(filesDir)  — before first render
    S->>B: SetListener(l)        — push target for async updates
    S->>B: RenderInitial()
    B->>M: full render
    M-->>S: full tree JSON — mount it
    loop User interaction
        S->>B: TriggerCallback(id) / TriggerTextCallback(id, v) / ...
        B->>M: handler + re-render under the render mutex
        M-->>S: patch JSON (synchronous return) — apply in order
    end
    Note over M,S: Timers / goroutines: State.Set →<br/>listener.ApplyPatches(patches) on a background goroutine —<br/>hop to the UI thread before touching views
```

**Delivery guarantee:** each render pass produces its diff exactly once, on
exactly one of the two paths (synchronous return or push). Apply everything
you receive from either path, in arrival order, and the native tree stays
consistent. Patch semantics — positional paths, ordering rules — are in
[Reconciliation](../concepts/reconciliation.md#patches).

| Bridge function | Purpose |
|---|---|
| `Register(ctx, root)` | Install the app (from Go `init`). Re-registering closes the previous manager |
| `SetDataDir(path)` | Writable sandbox dir for Go-side persistence (Application Support on iOS, `filesDir` on Android). Call before `RenderInitial` |
| `SetListener(l)` | Attach the async push target (`ApplyPatches(string)`) |
| `RenderInitial()` | Full tree JSON for the first mount |
| `TriggerCallback(id)` / `TriggerTextCallback` / `TriggerBoolCallback` / `TriggerIntCallback` | Event dispatch; returns the resulting patches |
| `RenderAgain()` | Escape hatch for shells that drive rendering themselves |
| `SetSystemEventListener(l)` | Sink for app→host system events (`toast`, `open_url`, `audio`); `OnSystemEvent(name, payloadJSON)` |
| `ReportHostEvent(name, payloadJSON)` | Host→app events that answer no callback (`audio_status`, `lifecycle`); returns the resulting patches like `Trigger*` |

## Building — Android

Prerequisites: Android SDK + NDK, and gomobile:

```bash
go install golang.org/x/mobile/cmd/gomobile@latest golang.org/x/mobile/cmd/gobind@latest
gomobile init
```

Then:

```bash
android/build.sh ./examples/todoapp   # any package whose init calls mobile.Register
```

The script binds `./mobile` plus your app package into
`android/app/libs/grmob.aar` (it defaults `ANDROID_HOME` /
`ANDROID_NDK_HOME` to the standard macOS locations if unset). Open
`android/` in Android Studio and run the `app` module — the Kotlin shell in
`android/app/src` implements the renderer against the bridge contract above.

From the command line, use the committed Gradle wrapper (it downloads its own
pinned Gradle, so no system install is required):

```bash
cd android && ./gradlew assembleDebug
adb install -r app/build/outputs/apk/debug/app-debug.apk
```

## Building — iOS

Requires **full Xcode** (not just Command Line Tools) — gomobile drives
`xcodebuild`:

```bash
sudo xcode-select -s /Applications/Xcode.app
ios/build.sh ./examples/todoapp
```

This produces the xcframework consumed by the Xcode project under `ios/`
(`project.yml` / SwiftUI shell in `ios/GrMob`).

## Shipping a different app

Both build scripts take the app package as their first argument. Any Go
package whose `init` calls `mobile.Register` (and exports one bindable
symbol) drops into the same shells — that is the whole integration contract,
and it's why the examples are structured as packages, not mains.

An app in its own module (one made by `grmob new`) does not run these
scripts, which bind from this repository's root. `grmob android` and
`grmob ios` do the same work from the app's root: they check the target's
prerequisites (the same table `grmob doctor` prints), copy this repository's
shell into the app's `android/` or `ios/` once — setting the application ID
and launcher name from the app's `grmob.json` — build gomobile and gobind
from the app's module graph, bind `mobile` plus `./app`, and finish with
Gradle `assembleDebug` or `xcodegen` and a simulator `xcodebuild`. The copied
shell belongs to the app from then on; `-refresh` copies a newer one over it.

The copy is made the app's own in three more places than its name:

- **Deep links.** The shells claim the `grmob://` scheme for the demo; a copy
  claims the application ID in lower case (`com.example.hello://`), so two
  GrMob apps on one device do not fight over a link.
- **Permission prompts (iOS).** The four `NS*UsageDescription` strings are
  replaced with a neutral one naming the app ("Hello uses the camera when you
  allow it."). The keys
  stay, because iOS terminates an app that requests a permission whose key is
  missing. Rewrite them to say *why* before shipping — App Review reads them.
- **UI tests (iOS).** `GrMobUITests` drives the demo by its labels, so it is not
  copied and its target and scheme test action are removed from `project.yml`.

gomobile and gobind are recorded in the app's `go.mod` as `tool` directives,
the way this repository records them, so `go mod tidy` keeps
`golang.org/x/mobile` rather than dropping it until the next native build adds
it back.

## Text grids

`core.TextGrid` renders one monospace `Text` per row: Compose builds an
`AnnotatedString` of `SpanStyle` runs in `FontFamily.Monospace`, SwiftUI an
`AttributedString` in the `.monospaced` design. Rows never wrap; a grid wider
than the screen scrolls horizontally. Both natives draw a dim run by fading
its colour, so a dim run with no colour of its own (and no grid `TextColor`)
renders at full weight.

## Audio

`core.AudioLoad`, `AudioPlay`, `AudioPause`, `AudioToggle`, `AudioSeek`,
`AudioSkip`, `AudioSetRate` and `AudioStop` drive one player per process,
behind the platform's own media session — so background playback, the
lock-screen card, headset buttons and CarPlay/Android Auto come from the OS
rather than from the app. A screen reads the player with `hooks.UseAudio`,
which re-renders it on every status tick:

```go
status := hooks.UseAudio(ctx)
core.Slider(status.Position, 0, status.Duration, nil,
    core.OnSliderChangeEnd(core.AudioSeek))
core.Button(map[bool]string{true: "Pause", false: "Play"}[status.State == core.AudioPlaying],
    core.AudioToggle)
```

Two channels carry it:

```
app ──SendSystemEvent("audio", {command: load|play|pause|seek|skip|rate|stop})──▶ shell
app ◀──ReportHostEvent("audio_status", {url, state, position, duration, rate, error})── shell
```

The second is new with this feature and generic: `core.ReceiveHostEvent`
fans any named host→app event out to `core.OnHostEvent` subscribers after
core's own consumers, and `mobile.ReportHostEvent` is its one bridge entry
point (`GrMobWASM.HostEvent` in the browser). Shells dispatch it on the same
serial executor as `Trigger*` calls and apply the returned patches the same
way. The app lifecycle rides the same channel; see [Lifecycle](#lifecycle).

| Shell | Player | Session | Background |
|---|---|---|---|
| Android | Media3 ExoPlayer in `GrMobAudioService` (a `MediaSessionService`), driven through a `MediaController` from `AudioPlayer.kt` | Media3's own notification | `FOREGROUND_SERVICE_MEDIA_PLAYBACK` + `WAKE_LOCK`; the service is declared in the manifest |
| iOS | `AVPlayer` in `AudioPlayer.swift` | `MPNowPlayingInfoCenter` + `MPRemoteCommandCenter` | `UIBackgroundModes: [audio]` (project.yml) and a `.playback` audio session |
| Browser | `HTMLAudioElement` in `grmob-runtime.js` | the Media Session API | the tab |

Status arrives twice a second while playing. `AudioLoad` and `AudioStop`
update `core.CurrentAudioStatus` optimistically (loading / idle) so the very
next render already shows the right track; everything else is the shell's
word. `examples/mobileapp`'s Audio tab exercises the whole surface.

## Lifecycle

`core.CurrentLifecycle()` reports whether the app is on screen —
`LifecycleActive`, `LifecycleInactive` or `LifecycleBackground` —
`core.OnLifecycle` subscribes to transitions, and `hooks.UseLifecycle`
re-renders a component on each one. The case that put it on the roadmap is
a client reconnecting on resume: a phone that spent an hour in the
background comes back with a dead socket, and without this the app learns
that from its first failed write rather than from the foregrounding itself.

```go
// Wherever the connection lives, outside the tree:
core.OnLifecycle(func(s core.LifecycleState) {
    if s == core.LifecycleActive { conn.Resume() }
})
```

The vocabulary is SwiftUI's `ScenePhase`, the most finely divided of the
three hosts; the others map onto it:

| Shell | Source | active | inactive | background |
|---|---|---|---|---|
| iOS | `scenePhase`, read at the `App` (`AppLifecycle.swift`) | `.active` | `.inactive` | `.background` |
| Android | `ProcessLifecycleOwner` (`AppLifecycle.kt`) | `ON_RESUME` | `ON_PAUSE` | `ON_STOP` |
| Browser | Page Visibility API (`grmob-runtime.js`) | visible | — | hidden |

Android deliberately observes the *process*, not the Activity: an Activity
is torn down and rebuilt on every rotation, and an app subscribed to
reconnect on resume would redial every time the phone turned. The process
owner delays `ON_PAUSE`/`ON_STOP` by a short grace period and cancels them
when another Activity of the app takes over, so they mean "the user left".

It travels as the `"lifecycle"` host event with one key, `state`; core
consumes it into the record before app subscribers to the raw event run,
the same ordering `audio_status` has, and drops a repeat of the current
state, so subscribers hear transitions only. The initial state is active —
an app that has just started is on screen — and a shell that disagrees says
so with its first report. `mobile/verify` holds the three shells' spellings
of the event and its states to core's.

## Permissions

`permission.Check(p)` and `permission.Request(p)` travel as the `"permission"`
system event with two keys — `command` and `kind` — and each host answers over
the `"permission"` host event with `kind` and `status`. Go validates both keys
on arrival and drops anything it cannot read rather than guessing: every wrong
guess has a cost, since `granted` opens a device the OS did not authorise,
`denied` hides a feature that works, and `unavailable` sends a user to a
settings page with no switch on it.

The vocabulary is the browser's, so the browser needs no mapping table and
these two hosts each map their own richer enum onto it.

| Go | iOS | Android |
|---|---|---|
| `Granted` | `.authorized` / `.authorizedWhenInUse` / `.authorizedAlways`, and Photos' `.limited` | `PERMISSION_GRANTED` |
| `Prompt` | `.notDetermined` | `PERMISSION_DENIED`, no rationale, never asked by this install |
| `Denied` | `.denied` | `PERMISSION_DENIED` with a rationale, or after this install has asked |
| `Unavailable` | `.restricted` — a parental control or an MDM profile | the permission is missing from the manifest |

Photos' `.limited` maps to `Granted` because the app really can read the
photos the user picked; which ones is the picker's business, not this
channel's.

**Android has to reconstruct three states out of two.** `checkSelfPermission`
answers GRANTED or DENIED and nothing else, and
`shouldShowRequestPermissionRationale` is false both for "never asked" and for
"don't ask again" — so the shell tracks whether this install has asked, and
breaks the tie towards `Denied`. Reporting `Prompt` for a permanently refused
permission would leave a screen offering a button that does nothing.

That flag is in `SharedPreferences`, in the shell's own file, written on the
request path only. It used to be in memory, on the argument that a framework
shell should not write to disk unasked, and the cost was paid on every cold
start: a permanently refused permission read as `Prompt` until the next request
proved otherwise, so the first thing the user saw was a button offering to ask
and pressing it did nothing. What settled it is that the fact being remembered
is the *platform's* — "has this install ever asked for X" is bookkeeping
Android keeps and will not answer — and a quirk of one platform belongs in the
one file that knows about it rather than in every app built on top.

**A `denied` request still reaches the launcher.** Persisting the flag
introduces a failure the in-memory version could not have: Android 11+ revokes
permissions for apps the user has not opened in months and resets
don't-ask-again with them, so the flag on disk says "asked" while a request
would in fact show the dialog again. A request path that short-circuited on
`Denied` would lock the user out of a permission the system had just handed
back. So it short-circuits on `Granted` and `Unavailable` only — safe because
`registerForActivityResult` always delivers a result, so a permanent refusal
comes straight back as DENIED with nothing on screen.

**Nothing announces a permission changing in Settings**, on either phone. The
signal both platforms do send is the app returning to the foreground, which is
what `permission.WatchForeground` — and `hooks.UsePermissionLive` above it —
turns into a re-check. It is reference-counted by permission rather than by
screen, so five screens watching the camera produce one `check` per resume
between them and an app with no live watcher takes no lifecycle subscription at
all.

**Two build-time halves no Go code can supply.** An iOS prompt whose
`NS*UsageDescription` key is missing from `Info.plist` *terminates the app* at
the moment it would appear; an Android permission missing from
`AndroidManifest.xml` is auto-denied with nothing on screen. Both shells ship
the entries for all four permissions, `mobile/verify/permission_test.go` holds
them there, and the Android shell reports an undeclared permission as
`Unavailable` rather than `Denied` — it is a build the user cannot influence.

**Location asks for the narrower option on both.** iOS requests
when-in-use, not always: "always" is a second prompt Apple shows on its own
schedule after the app has demonstrably used location in the foreground, and
requesting it up front is how an app gets refused. Android requests
`ACCESS_COARSE_LOCATION` alone, so no precise/approximate chooser appears for a
promise the Go API did not make. One `permission.Location` constant means the
narrower thing everywhere, which is why there is no second constant for the
wider one.

**iOS needs an object where the others need a function.** `CLLocationManager`
reports an authorization change through its delegate rather than through a
completion handler, and a manager released while its prompt is up reports
nothing at all — so `Permissions.swift` is a singleton holding one. It is
deliberately not `HeadingSensor`'s manager: the compass asks for nothing on
purpose (an unexpected permission dialog is worse than a bearing a few degrees
off a map), and sharing the object would make one of those decisions the
other's.

**Android needs the Activity.** `registerForActivityResult` is an Activity API
whose registration must happen in `onCreate`, so `Permissions.attach` is called
from `MainActivity` rather than from `SystemEvents` — which keeps only an
application context, deliberately, so an Activity handed to it is not leaked.

## Maps

`core.MapView` is the platform's own map, wrapped so a declarative tree can hold
an imperative view: `MKMapView` behind a `UIViewRepresentable` on iOS
(`GrMobMapView.swift`), osmdroid behind an `AndroidView` on Android
(`GrMobMapView.kt`). Pins are `core.Marker` child nodes, which each renderer
reads off `node.children` itself rather than rendering as views — the same move
`core.TextGrid` makes with its rows.

**Neither needs a key.** MapKit is free on iOS. On Android the choice is
osmdroid over OpenStreetMap tiles rather than Google Maps Compose, for two
reasons that point the same way: a Maps key is a deployment secret every app
adopting this framework would have to obtain before a map drew anything, and
Play Services is absent on a real share of devices. Google Maps is a better map
where both are present, and it is a second provider `GrMobMapView.kt` could grow
rather than a reason to start there.

**The tiles are somebody else's bandwidth.** osmdroid's MAPNIK source and
Leaflet's OSM layer both hit the OpenStreetMap project's own servers, which have
a usage policy: identify your app, do not bulk download, expect to be blocked
above a modest volume. The Android host sets the user agent from the package name
(osmdroid's default is refused with a 418), puts its tile cache in the app's own
cache directory so no storage permission is involved, and is the place to point
at a paid provider for a real user base. MapKit has no such concern — Apple
serves its own tiles.

**Zoom is not a span, and iOS is where they meet.** Every engine here speaks the
slippy-tile zoom level, and osmdroid and Leaflet speak it natively. MapKit
speaks `MKCoordinateSpan`, in degrees, so `GrMobMapView.swift` converts through
Web Mercator's own definition — 256 × 2^zoom pixels around the equator, so a
view *w* points wide shows `360 × w / (256 × 2^zoom)` degrees of longitude. The
width is read off the live view through a `GeometryReader`, so zoom 14 shows the
same ground on an iPhone as in a browser. The latitude half is derived from the
aspect ratio and the cosine of the latitude and is an approximation; the
longitude half is exact, and the zoom level is defined by longitude.

**A gesture is reported once, after it ends.** A pan generates a region per
frame and each one crossing the bridge would be a full Go render pass, so every
host throttles on its own side: a 120 ms quiet window on iOS and on the web, and
osmdroid's own `DelayedMapListener` at the same interval on Android. The window
also coalesces a pinch, which ends as a pan *and* a zoom a few milliseconds
apart.

**The echo guard is in all three.** Each host remembers the region it last
applied and compares, which is what keeps Go's re-renders from snapping the map
out from under a finger — and the same comparison is what stops the host's own
recentring from arriving back in Go as a user gesture. See
[Views](../concepts/views.md#leaves) for the contract.

## The location sensor

`core.StartLocation`/`StopLocation` ride the same `"sensor"` system event the
compass does, with `kind: "location"` — one event name, two objects, and each
object drops the kind that is not its own. `LocationSensor.swift` wraps
`CLLocationManager`; `LocationSensor.kt` wraps the platform `LocationManager`
(not the fused provider, for the same Play Services reason the map gives).

**The two shells differ on who prompts, and that is the platform's difference.**
`CLLocationManager` can request authorization from anywhere, and there is no fix
at all without it — `startUpdatingLocation` on an undetermined status reports
nothing, forever, with no error — so the iOS sensor asks. Android cannot: only
an Activity can show the dialog, and that Activity is `Permissions.kt`'s. So the
Android sensor reports `available: false` with a reason instead. Both ends look
the same to Go, which is why `core.StartLocation` tells an app that wants to
control the moment to check and ask first.

**Android asks both providers.** `NETWORK_PROVIDER` answers in seconds with a
coarse fix; `GPS_PROVIDER` can take a minute outdoors and never answers indoors.
Requesting one means either a slow first fix or no fix in a building, so the host
requests both and Go takes whichever arrives — `Location.Accuracy` is what tells
them apart, which is what that field is documented for.

**Neither host normalises the coordinates.** `core.ReceiveLocation` is the single
place that clamps the latitude and wraps the longitude, which is what makes those
invariants properties rather than hopes — the same arrangement
`core.ReceiveHeading` has for the bearing.

## Persistence on device

Go code cannot discover the writable sandbox path itself — it is an OS-level
fact only the shell knows. The shell passes it via `SetDataDir` before the
first render; Go code reads it with `mobile.DataDir()` and opens its store
lazily on first render (the bound package's `init` runs before `SetDataDir`,
so don't open stores in `init`). With no data dir set — web preview, tests —
`DataDir()` returns `""` and persistence-aware apps run in-memory. See the
[tutorial's persistence chapter](../tutorial-todo.md) for the bytdb store
pattern.

## How the layout model reaches each platform

Go declares CSS-flavored layout; neither native toolkit speaks it natively.
Four of the mappings are worth knowing because they are where the platforms
needed real work rather than a lookup table.

### `FlexGrow` — proportional weights

Compose has `Modifier.weight`, which is proportional by construction, so a Row
child with `FlexGrow(3)` beside one with `FlexGrow(1)` has always split the
leftover space 75/25 there.

SwiftUI has no weight. The renderer used to give every grower a
`.frame(maxWidth: .infinity)`, which makes a stack split leftover space
**equally** — so the same declaration rendered 50/50 on iOS and 75/25 on
Android. `GrMobFlexStack`, a hand-written SwiftUI `Layout`, now runs the CSS
algorithm directly (`flex-basis: auto`, proportional grow, proportional
shrink) and replaces `HStack`/`VStack` for `Row`/`Column`. Two things came
along with it: `justify-content` is exact rather than emulated with hidden
`Spacer`s (`space-around` and `space-evenly` are different values again), and
the stack keeps SwiftUI's hug-unless-something-claims-the-space behavior, so
layouts that predate it render unchanged.

### The min-content floor, and where each target's comes from

CSS gives every flex item `min-width: auto`, so an overflowing row **overflows**
rather than grinding its children down to nothing. The three targets arrive at
that from three different directions, and only one of them had to be built:

| target | where the floor comes from |
|---|---|
| HTML / WASM | the browser's own `min-width: auto`. Nothing to do. |
| Android | Compose's `Row` has no proportional shrink *at all* — an unweighted child is offered whatever the ones before it left, and measures itself within that — so there is no shrink arm for a floor to bound. The same fact is why the fixture in `internal/pinfixture` diverges from CSS on the *siblings*. |
| iOS | built. `GrMobFlexSolver` really does share the deficit out, so without a floor it is the one renderer that can assign a child less than its content needs — and did: the tutorial's lesson numbers rendered as `4 / . / 1 / 2` down the side of each row on the first simulator run of that app. |

The iOS floor is `GrMobMinContent`, and the shape of it is worth knowing because
it is not where you would look for it. The floor is **computed from the node
tree, not measured from the view.** SwiftUI documents `ProposedViewSize.zero`
as the way to ask a subview for its minimum; a `Text` answers it with `0.0`,
because a Text accepts any width it is offered and wraps to fit. So there is no
minimum in the view layer to read, and the measurement is taken from the string
and the style instead — the widest unbreakable run, with CoreText, which is why
`ios/verify` can check it without a simulator.

`GrMobFlexSolver.resolve` then runs CSS's own resolution loop rather than one
division, because a child that stops at its floor is no longer absorbing its
share of the deficit and the rest have to take what it refused.

Every unknown in that computation resolves **downwards** — a floor that is too
low leaves a child exactly as crushable as it was before, while one that is too
high overflows a line a browser would have fitted. So a child with a declared
width floors at 0 (CSS's automatic minimum is the *smaller* of the declared size
and the content size, and this host cannot see the content behind the frame a
declaration becomes), and every leaf that is not text floors at 0 as well.
`GrMobMinContent`'s own doc carries the full table.

### `AlignItems: "stretch"`

A stretched child is *sized* to the container's cross axis, not *placed*
along it, so neither toolkit's alignment enum can express it — both have
start/center/end and nothing else.

- **iOS** — `GrMobFlexStack` proposes the full cross extent to each child.
- **Android** — the container hands each child a `fillMaxWidth()` (Column) or
  `fillMaxHeight()` (Row); a stretched Row is additionally pinned to
  `IntrinsicSize.Max` so its children stretch to the tallest sibling rather
  than to the whole screen. Intrinsic measurement is why a `List` inside a
  stretched `Row` is unsupported — a lazy list cannot report an intrinsic
  height. Give that Row an explicit `Height` instead.

Both renderers apply it to lazy lists too, where it is the only flex property
that means anything (a scrolling axis has no leftover space to divide).

**Stretch is the default on the vertical containers.** A `Column`, `List` or
`Scroll` with no `AlignItems` and no `Align` stretches its children, on both
natives, because that is the CSS default (`align-items: stretch`) and
therefore what the two DOM targets have always drawn: an `Input` in a Column
runs the full width in the browser, and on a phone it used to hug its
placeholder. Two kinds of child keep their own width, exactly as on the web:
one with an explicit `Width`, and one whose `Display` is inline — which is
how the bundled themes make `Button` and `Badge` hug their content (the web
runtime turns that into `width: fit-content`); `components.Button{FullWidth:
true}` asks for the stretch back. An explicit `AlignItems(AlignFlexStart)`
still packs. Rows keep their top-aligned default: the intrinsic-height
measurement above has real costs inside a List, so nothing turns it on
unasked.

### A `Spacer`'s own `Style`

`core.Spacer(n)` is a node whose size arrives as a **prop** rather than as a
`Style` declaration, and that is what made it the one node type on both natives
whose `Style` went missing entirely. Each arm was a single expression built from
the prop — `Color.clear.frame(width:height:)` and `Spacer(Modifier.size(n.dp))`
— with no call to the renderer's own box helper anywhere in it, so a
hand-assembled Spacer carrying a `Background`, a `Margin`, an
`AccessibilityLabel` or an `OnTap` got none of them. Both DOM targets had always
applied it, so the same node was coloured in a browser and invisible on a phone.

Both arms now build the node's box, and the size prop goes **underneath** the
author's declarations — the rule every chassis in this framework follows, stated
at `modalChassis` in `htmlout` and at `applySpacerChassis` in the WASM runtime.
The two languages get there in opposite ways, which is worth knowing before
moving either line:

| | how the author wins |
|---|---|
| SwiftUI | later in a chain is further *out*, and an outer frame wins — so the chassis frame is written before `.grMobBox`. On an axis the `Style` claims it is not written at all (`nil`), which keeps `Color.clear` flexible there so the background fills the frame `grMobBox` puts around it |
| Compose | constraints flow outside-in and an inner `size()` coerces itself into what it was handed, so `boxModifier` first and `.size()` after is the whole of it — per axis, and the background lands at the measured size |

#### And the children, which were the last thing dropped

A hand-assembled Spacer's **children** used to go the same way its `Style` did.
A Compose `Spacer` and a SwiftUI `Color.clear` are leaves, where both DOM
renderers emit a Spacer's children like any other element's — so a subtree
rendered in a browser and vanished on a phone. It was left open on the grounds
that `core.Spacer(n)` builds no children, which is true and is not the same as
unreachable: `htmlout` exports any `*core.Node` it is handed and the WASM
runtime renders any tree the wire carries.

The DOM side is also the side that cannot move. The runtime addresses patches by
the `data-node-path` attributes it writes while walking `node.Children`, so an
element it declines to emit takes every patch beneath it with it — the same
reason `Fragment` and `Theme` are boxed there.

So the natives emit them, as a **column**:

| | how |
|---|---|
| Compose | the arm hands the node to `GrMobColumn` with the size as its `outer` modifier, which produces the identical chain (`boxModifier(...).then(size)`) the arm used to spell out |
| SwiftUI | `Color.clear` stays as the thing that is *sized* and the flex stack is laid **over** it. `grMobBox` paints its background inside its own dimension frame, so an axis the `Style` claims is filled by whatever the content asks for — `Color.clear` is flexible and asks for all of it, an empty stack would ask for none |

The axis is not each renderer's choice. `Spacer` is in `htmlout`'s `stackAxes`
on `column`, which is the one place all four targets read it from; without that
row two `Text` children would run together on one line in a browser and down the
page on device, the exact divergence the table exists to prevent. **A Spacer with
children is a `Box` with a fixed size**, and every tree `core` produces is
unaffected — a childless flex box of fixed size lays out as a childless block
box of the same size.

`mobile/verify/spacer_test.go` pins that the box is built at all and that the
chassis sits under the author; `mobile/verify/stacking_test.go` pins that the
children are stacked rather than dropped, and that the axis is the one
`stackAxes` names.

### `ContentMode` on `Image`

```go
core.ImageWithMode(url, core.ContentModeFill, core.Width("64px"), core.Height("64px"))
```

`core.Image` keeps its old signature and its old rendering; the mode is a
required argument on the `WithMode` builder rather than an option buried in a
style list.

| `core.ContentMode` | SwiftUI | Compose | CSS |
|---|---|---|---|
| `Fit` (default) | `scaledToFit()` | `ContentScale.Fit` | `object-fit: contain` |
| `Fill` | `scaledToFill().clipped()` | `ContentScale.Crop` | `object-fit: cover` |
| `Stretch` | `resizable()` | `ContentScale.FillBounds` | `object-fit: fill` |
| `Center` | intrinsic size, clipped | `ContentScale.None` | `object-fit: none` |

`Fill` and `Center` crop on every target — an unclipped image would paint over
its siblings.

Note the three value columns have nothing in common: SwiftUI modifier chains,
Compose `ContentScale` cases, CSS keywords. Only the first column — the mode
names — is shared by all four renderers, so the DOM pair can be checked by
comparing tables while the natives can only be checked for **coverage**: does
every mode core declares have an arm of its own?

Coverage is the half that matters here, because both natives fold the
unrecognized case into `Fit` (neither SwiftUI nor Compose has CSS's "unset" to
fall back to). Without a check, adding a fifth `ContentMode` would draw as
`Fit` on iOS and Android while both DOM targets fell back to the browser
default — four renderers, two behaviors, no error anywhere.

`mobile/verify/contentmode_test.go` closes that. It reads the `switch mode` in
`Renderer.swift` and the `when (mode)` in `Renderer.kt` as text and holds their
arms up against `core.ContentModes()`. Reading rather than compiling is not a
shortcut: `default` and `else` make a string switch exhaustive by construction,
so "you forgot a mode" is not a type error in either language and never will
be — only something comparing the arms with Go's list can notice. Doing that in
Go is what puts the check inside a plain `go test ./...`, where it runs without
Xcode, without the Android SDK, and without anyone remembering a `run.sh`.

The price is a shape both functions must keep, stated in a comment beside each:
one arm per line, string literals first on the line, the catch-all last, and
every mode listed explicitly — including the one the catch-all would have
handled anyway. Every violation fails as a named error saying what changed,
rather than as an empty comparison that agrees with everything.

### `Alignment`, `JustifyContent` and `AlignItems`

The same coverage argument, applied to the three alignment types — and this
time it found live bugs rather than a hypothetical one.

These are unlike `ContentMode` in one way that matters. `htmlout` and the WASM
runtime emit `justify-content` and `align-items` **verbatim**: core's spellings
*are* the CSS ones, so the DOM pair cannot be wrong about a value it never
interprets. All of the drift risk is native, where eleven `switch`/`when`
dispatches across three files each turn a string into a SwiftUI or Compose
value, and each ends in a catch-all that renders *something*. A value with no
arm does not error — it packs to the start.

| core | iOS | Android |
|---|---|---|
| `JustifyContent` | `GrMobFlexSolver.leading` + `.gap` (arithmetic) | `horizontalArrangement` / `verticalArrangement` |
| `AlignItems` | `GrMobFlexSolver.crossOffset`, `crossAlignmentH` | Row / Column / List cross alignment |
| `Alignment` (as text) | `grMobTextAlignment` | `textStyle`'s `textAlign` |

`mobile/verify/alignment_test.go` holds each of them to `core.JustifyContents()`,
`core.AlignItemsValues()` or `core.TextAlignments()`. Two details are worth
knowing:

- **The iOS solver answers `justify-content` with two dispatches**, one for the
  offset before the run of children and one for the gap between them, and each
  returns 0 for the half the other owns. They are checked *separately*: a union
  of their arms would pass if each half answered for values it has no business
  answering for, and the arrangement that makes the solver readable is that each
  states its own complete opinion.
- **Several cross-axis dispatches serve two vocabularies at once.** `Style.Align`
  doubles as the fallback a container reads when `AlignItems` is unset, so a
  Column's switch legitimately carries `"start"`/`"end"` arms that are
  `core.Alignment` values, not `AlignItems` ones. Those are *permitted, not
  required* — `GrMobRow` deliberately declines the fallback, because `Align` is
  a text-alignment concept and has never been read for a Row's vertical axis.
  All four targets now draw that line in the same place: the DOM pair reads
  the same fallback through `htmlout/crossaxis.go`'s tables, gated to the
  same three container types and declining `Row` the same way (see
  [WebAssembly](wasm.md#the-cross-axis-fallback)).

#### What this found

`Align` was the worst-behaved prop in the framework: one value, four behaviors.
`core.Align(core.AlignJustify)` justified the text on Compose, rendered leading
on SwiftUI, exported no declaration at all from `htmlout`, and did nothing
whatsoever on the web — the WASM runtime **never read `Style.Align` in any
form**. All four are now held to `core.TextAlignments()`, and the web target
reads the prop at all for the first time (see
[WebAssembly](wasm.md#text-alignment)).

Coverage means a target has *said* what it does with a value, not that it can
honor it. SwiftUI genuinely cannot justify text — `TextAlignment` has three
members — so `grMobTextAlignment` carries an explicit `"justify"` arm that
falls back to leading and names the limit. An explicit arm is an answer;
silence is not.

`Align(AlignStretch)` on a Column was a second divergence, and one no coverage
test can reach: it stretched on iOS and did nothing on Android, because Compose
tested `alignItems` alone where SwiftUI consulted the `Align` fallback. That is
an equality test, not a dispatch — there are no arms to hold up against a list —
so it was found by reading and fixed by hand (`isColumnStretch`). The switch
checks reach the switches, and that is all they reach.

`Align(AlignStretch)` on a `List` was the same story again, this time with the
natives in perfect agreement — on the wrong answer. Each List's placement
dispatch reads the `Align` fallback, and its `"stretch"` arm defers to the fill
modifier in the item loop; each item loop tested `alignItems` alone. The value
took the arm's word for a fill that never happened: rows placed at the start
edge and stretched nowhere, while a Column with the identical style stretched.
Both loops now read the fallback-aware helper (`crossAxisValue` on iOS,
`isColumnStretch` on Android — a List's cross axis is horizontal like a
Column's), and `TestListStretchFillReadsTheAlignFallback` pins the loop to the
helper and the helper to the fallback — the one stretch equality a test now
reaches.

#### Known divergence, left alone

Compose's five distributing `Arrangement`s take no spacing argument, so a `Row`
with both a `Gap` and a `JustifyContent` loses the gap on Android. CSS treats
gap as a minimum that `justify-content` adds to, and the iOS solver does the
same (it carries `spacing` separately from `justify`).
`Arrangement.spacedBy(gap, alignment)` would fix the three packing values;
nothing expresses gap-plus-distribution for the `space-*` three without a custom
`Arrangement`. Not attempted, because it is a rendering change on the one target
this repo cannot build.

### `AccessibilityRole`

`core.AccessibilityRole(role)` becomes traits on iOS and semantics properties
on Android, and on both it is a partial mapping by design: `heading` and
`columnheader` become `.isHeader` / `heading()`, `button` becomes `.isButton`
/ `Role.Button`, `link` and `search` become `.isLink` and `.isSearchField`
(no Compose analog for either — its `Role` has no Link), and `status` /
`alert` become Compose live regions (no SwiftUI analog). The remaining twelve
— the tabular set, both collection pairs, the landmarks and `group` — have no
vocabulary on either platform and do nothing there while working on both web
targets.

`group` is worth telling apart from the other eleven, because it is empty for
the opposite reason. Those are silent because the platform has no way to say
the thing; `group` is silent because neither platform *needs* it. It exists
because ARIA prohibits an accessible name on the `generic` role a `<div>`
carries, so a labelled container was announced by nothing on the web — while a
`contentDescription` and an `accessibilityLabel` are honoured on any node here.
Both web exporters supply it automatically to a named, roleless node (see
[Styling & Theming](../concepts/styling-and-theming.md#rolegroup-and-the-one-role-you-get-without-asking)),
which means the role reaches both natives too and correctly does nothing.

Both renderers *name* every role anyway, in a `switch`/`when` with explicit
empty arms, so a role that does nothing is on record rather than lost in a
catch-all — and `mobile/verify/role_test.go` holds both dispatches against
`core.Roles()` under a plain `go test ./...`. See
[Styling & Theming](../concepts/styling-and-theming.md#accessibilityrole).

### `AccessibilityHeadingLevel`

The tier that goes with `RoleHeading` splits the two platforms rather than
mapping partially to both. iOS has `.accessibilityHeading(.h1 … .h6)`, so the
level reaches VoiceOver's heading rotor; Compose's `heading()` takes no
argument and its semantics package has no level property, so the Android
renderer deliberately never parses the JSON key.

That silence is written down beside the role dispatch in `GrMobStyle.kt`, for
the same reason the empty role arms are: a field a renderer simply ignored
looks exactly like one nobody had heard of.
`mobile/verify/heading_level_test.go` pins both halves — that Swift carries the
level through all three links, and that Kotlin's note has not outlived the
limitation.

### `AccessibilityNestingLevel`

The other two roles `aria-level` serves — `listitem` and `row` — map to nothing
on either platform. SwiftUI has no nesting-depth property at all, and Compose's
nearest one, `collectionItemInfo`, states an item's index and span within *one*
collection rather than its depth within nested ones; filling it from this field
would tell TalkBack something the app never said. So unlike the heading tier,
which splits the two platforms, this field is inert on both and lives on the
web alone.

Neither renderer parses the key, and both say why — `GrMobStyle.kt` beside the
role dispatch, `GrMobStyle.swift` beside `grMobHeadingLevel`, which is where a
reader who has just seen the heading third mapped will ask about the other two.
`mobile/verify/nesting_level_test.go` pins the notes and their absence of a
parse together.

### `AccessibilityID` and `AccessibilityControls`

The two IDREF props — an element identity and the `aria-controls` that points
at it — are the third shape in this section: inert on both platforms, like the
nesting level, and for a reason that is about the *reader* rather than about
the API. VoiceOver and TalkBack both move through a screen by swiping to the
next element; neither follows a relationship from a tab to the region it shows,
which is what a browser's reader uses `aria-controls` for. There is nothing in
either semantics vocabulary for the pair to become.

Neither renderer parses either key, and both say why — beside `grMobRole` in
each file, which is where a reader looking for the mapping arrives.
`mobile/verify/idref_test.go` pins the notes and the absence of a parse
together, as the two level tests do.

`core.RoleTabPanel` joins them for the same reason and gets an empty arm in both
dispatches: what makes a tab panel announce as one is being *pointed at*, and
neither reader follows a reference. A `core.TabView` still announces correctly
on both phones, because each hands the whole strip to the platform's own tab
container.

It also pins one thing they do not, because this field has a near miss the
levels never had: `accessibilityIdentifier` on iOS and `testTag` on Compose.
Both look like the obvious mapping for `AccessibilityID` and both are *test*
selectors rather than accessibility properties — XCUITest and Espresso read
them, VoiceOver and TalkBack do not. Mapping onto them would quietly turn every
hand-built tab into a test handle and still announce nothing, so each note names
the property it is turning down and the test checks the naming is still there.

### `AccessibilityExpanded`

The disclosure state is the one accessibility field where the *two natives*
disagree, which is the reverse of every shape above.

**Compose maps it, and to an action rather than a property.** There is no
expanded property in Compose semantics; there are `expand()` and `collapse()`,
which become `AccessibilityNodeInfo`'s `ACTION_EXPAND` and `ACTION_COLLAPSE`
and which TalkBack offers as "double-tap to expand". So a closed disclosure is
given the expand action and an open one the collapse action — the inverse of
the state, which is this mapping's one silent bug, since both arms compile and
both toggle the section correctly when activated.

An action has to *do* something, and the only thing that can open the section
is the callback the node's tap already runs. That callback lives on the node
rather than on the style, which is why this mapping is the one that does not
live in `GrMobStyle.boxModifier`'s semantics lambda beside `grMobRole` and
`grMobSelected` — it is `grMobDisclosure` in `Renderer.kt`'s `gestureModifier`,
the one place holding both halves. The consequence is that **a node with an
expanded state and no `OnClick` gets nothing**, which is the honest outcome: an
expand action TalkBack can invoke and Compose cannot perform is worse than a
disclosure that is merely quiet.

**SwiftUI has nothing.** `AccessibilityTraits` has no expanded member, and
SwiftUI's own `DisclosureGroup` announces its state by writing an accessibility
**value** — a localized string SwiftUI supplies from its own bundle. That is
the near miss, and it is a sharper one than `accessibilityIdentifier` was
above, because `accessibilityValue` is a real accessibility channel that the
platform genuinely uses for this. What makes it wrong for a framework is that
this renderer would have to supply the literal "expanded" or "collapsed" in
English, for every app in every locale, in a slot the app may want for a value
of its own. It is the same move `components.Chip`'s `", selected"` name suffix
was deleted for.

So the key crosses the bridge and `GrMobStyle.swift` does not parse it, with a
note beside `grMobTraitsFor` naming `accessibilityValue` and saying why.
`mobile/verify/expanded_test.go` pins both sides: Compose's parse, its two arms
against `core.ExpandedStates()`, and the state/action pairing as one string so
a transposition fails; and iOS's absence of a parse plus both halves of the
note.

### `AccessibilityValue`

The value of a valued control splits the two platforms too, and along a
different seam from the disclosure above: here they disagree about *which half*
of the field they can say.

**Compose takes the numbers.** `progressBarRangeInfo` is one of the better
mappings in this framework — TalkBack turns a position and its bounds into a
percentage it localizes itself, so a bar reports "45 percent" in the user's own
language with no string ever crossing the bridge. That is exactly what
`components.ProgressBar` could not do while its value lived in the accessible
name. A range with bounds and no position becomes
`ProgressBarRangeInfo.Indeterminate`, which is ARIA's indeterminate bar said in
Compose's words; a range that states nothing numeric leaves the property alone,
because a `Text` on an ordinary node must not turn it into a progress bar.

**SwiftUI takes only the words.** There is no numeric accessibility value on the
platform — `accessibilityValue` takes a `Text` and there is no equivalent of
`ProgressBarRangeInfo` — so the three numbers arrive, are visible in
`GrMobStyle`, and reach no view modifier. Formatting them into a string here is
the tempting fix and is the same move the disclosure note above turns down: a
renderer that spells a number out loud is choosing a language for every app that
uses it.

`ValueRange.Text` is the half both platforms take, through `stateDescription`
and `accessibilityValue`, and the difference that makes it safe is whose words
they are — the app's own, on the channel `AccessibilityLabel` and
`AccessibilityHint` already ride. It is, incidentally, the value slot the
`AccessibilityExpanded` note says the framework has no room for: an app that
wants VoiceOver to hear "expanded" can now say so in its own language, which is
a different thing from this renderer deciding to.

Neither renderer consults the role, for the reason neither consults it for a
selection: both honour these properties on any node, and ARIA is the strict one.
`mobile/verify/value_test.go` pins Compose like a mapping — parsed, dispatched,
reaching both primitives, invoked from the semantics lambda — and SwiftUI in
both directions: the words reaching a modifier, and the numbers reaching
nothing.

The one detail worth knowing about the wire: the numbers cross as **strings**,
because 0 is a bar at the start of an upload and is also the zero value of a Go
float, and `core.Style` merges on "non-zero wins". Kotlin keeps the distinction
after parsing — nullable `Float`s rather than a `0f` default — or it would
reintroduce the same bug one layer down and could not tell an indeterminate bar
from one that has not started.

#### The reading has an authority now

The three arms above — determinate, indeterminate, nothing claimed — used to
exist only in Kotlin, in a branch inside `grMobValue`, which is an extension on
`SemanticsPropertyReceiver`: reaching it means constructing a Compose semantics
scope, which means a device. What checked it instead was `mobile/verify`
searching `GrMobStyle.kt` for the word `Indeterminate`, which is green for a
branch that reaches the word on the wrong condition.

The rules are ARIA's, not Compose's — the implicit `0..100`, the missing
position that means "running with no idea how far" — and both were already
written in prose on `core.ValueRange`'s own fields. `ValueRange.Progress` makes
that prose executable. It returns one of four readings, because the branch was
really four: the two above, plus an *empty range* (`Max` at or below `Min`),
which Compose cannot hold at all — `ProgressBarRangeInfo` throws on one — and
which must therefore be told apart from "nothing was claimed" before the
property is assigned. Both silent readings are named arms in the `when` rather
than an `else`, so the second one is visible rather than an absence.

The web exporters do not call it, and that is the honest reason it lives in
`core` rather than beside them: they hand the three attributes to a browser,
which applies the same rules itself. Its consumers are the transliteration and
the harness that compares them.

    core/value.go                    ValueRange.Progress — the authority
    internal/valuefixture            the shared table of wire ranges
    GrMobProgress.kt                 the Kotlin reading, importing nothing
    android/verify                   runs one against the other on a JVM

The parse is inside what is compared, not beside it: `grMobProgressNumber`
takes the wire string, so "half", `""` and `NaN` are all checked to be *no
position* rather than assumed to be. `GrMobProgress.kt` is the second file to
earn `TestNativeMenuDecompositionIsUIFree`'s rule — an import there would end
the JVM pass and send the check back to searching source text.

### The field with a widget spending it

The field now has a widget spending it — `components.ListRow`'s `NestingLevel`,
for an outline flattened into one list — and that changes nothing here, which
is worth saying plainly rather than leaving to be inferred. Such a row goes out
with `role="listitem"` and a depth; on device the role reaches the `when`/
`switch` and lands on an empty arm, and the depth is not parsed at all. A
flattened outline therefore announces as a flat list of items on both phones
and as a nested one on both web targets. That is the same honest partial the
nine unmapped roles have: each platform says the truest thing it can, and the
half it cannot say is written down instead of faked.

### A `core.Button`'s border

A Button draws its own container on both platforms, so each renderer hands it a
style with the box-drawing fields stripped (`marginAndSize` on Android,
`marginAndSizeOnly` on iOS) and feeds them back through the platform control's
own slots — Compose's `Button(colors:, shape:, contentPadding:)`, SwiftUI's
`GrMobButtonStyle`. The border was the one field stripped and never fed back,
so `core.BorderColor`/`BorderWidth` were silently dropped on Buttons alone and
`components.Button`'s outlined emphasis had no rule on device.

Both now carry it: a `BorderStroke` into material3's `border` slot (and into
the `Surface` the long-press path rebuilds the button out of), and a
`strokeBorder` overlay on the same rounded rectangle the fill is clipped to.
The guard is the same on every target — a width **and** a color — so half a
border is no border everywhere. `mobile/verify/button_border_test.go` pins both
renderers.

### A text field's frame

Nothing to do here either, and for the opposite reason to the Button above: both
renderers already honored the style. Compose composes a `BasicTextField` through
the ordinary `boxModifier`, and SwiftUI a `.plain`-styled `TextField` through
`grMobBox`, so each drew exactly what `core.Style` asked for — and until both
bundled themes gave `Components.Input` and `Components.TextArea` a border, what
they asked for was nothing.

That is why the missing frame was a theme bug rather than a renderer one. The
phones were right and looked wrong; the web was wrong and looked right, because
a browser draws its own border on an `<input>`. Both themes now state a control
boundary at WCAG 1.4.11's 3:1, all four targets draw it from the same field, and
`borderResetTypes` could finally take the browser's away.

### A picker's frame, and why it is not a platform picker

`core.Select` is drawn on both natives as an ordinary styled box with a menu
hung off it — a SwiftUI `Menu`, a Compose `Box` anchored to a `DropdownMenu` —
and **not** from either platform's own picker control. That is a deliberate
cost: `.pickerStyle(.menu)` and Material's `ExposedDropdownMenuBox` are the
idiomatic controls and each draws a frame, a container colour and an indicator
of its own that no Go style can remove.

Taking them would have made the picker the one control in the vocabulary whose
edge came from the platform on two targets and from the theme on the other two
— the exact divergence `borderResetTypes` was extended to prevent, since the
web's `<select>` reset rests on the frame being the theme's everywhere. So each
native draws the style's own box (`grMobBox` / `boxModifier`, like every other
node) and the platform supplies only the behaviour.
`mobile/verify/select_test.go` pins both halves: that the arm draws through the
style's box, and that neither forbidden construct appears in it.

`SelectOption.Group` and `SelectOption.Disabled` are drawn by each platform's
own means, and this is where the two menus stop looking alike:

- **SwiftUI** has a `Section`, so a run of options sharing a heading is handed
  to one. A `Section` is a container, so a run has to be handed over whole,
  which is why the rows are drawn by `grMobMenuItems` rather than inline. The
  disabling is `.disabled` on the **Button** — on the Section it would take the
  whole run with it.
- **Compose**'s `DropdownMenu` has no section construct, so a heading is an
  ordinary `DropdownMenuItem` with `enabled = false` and an empty `onClick`,
  written ahead of its run. That is what makes it a label rather than a choice.

Both split by *runs*: consecutive options sharing a heading, in the order
written. See `core.SelectOption.Group` for why a gather would be the wrong
shape.

`SelectOption.GroupDisabled` disables a whole run, and this is the one case
where taking the Section down with it is the request rather than the accident:
SwiftUI puts `.disabled` on the **Section**, so the header greys along with its
rows. Compose needs nothing — its heading item is already `enabled = false` —
and both targets get the refusal itself the same way, because
`core.SelectMenuSections` marks every item of a disabled run before either
renderer sees it. That propagation is not a convenience: Compose has no section
construct to disable, and on the web a run with no heading has no `<optgroup>`
to carry the attribute.

#### The menu is invisible, so the decision was moved out of it

A SwiftUI `Menu`'s content is a closure of views and a Compose `DropdownMenu`'s
is a composable lambda. Neither can be read back, so nothing short of a
simulator or a device could open a picker and check its sections — which left
three facts about the open menu (the runs, the headings, the refused rows)
resting on `mobile/verify` finding the right substrings in a 900-line renderer.

The decision is out of the closure now. `GrMobSelectMenu.swift` and
`GrMobSelectMenu.kt` turn the flat option list into sections and rows, each
renderer's menu is a loop over the result, and both files import no UI at all
— which is what makes them runnable off a device. `core.SelectMenuSections` is
the authority both transliterate (and the one `htmlout` calls directly), and
both are now *run* against cases generated from it:

    ios/verify/run.sh       compiles GrMobSelectMenu.swift into its harness
    android/verify/run.sh   compiles GrMobSelectMenu.kt and runs it on a JVM

Neither harness needs a device, a simulator or gradle. `android/verify` is the
newer of the two and is described below.

What no harness reaches on **either** platform is the last step — the line
handing a row's value to `textChanged` and its disabled flag to the construct
that refuses the tap. That is what `mobile/verify/select_test.go` is for now,
and it is a much smaller surface than the one it used to cover.

#### `android/verify`: Kotlin without gradle

`GrMobSelectMenu.kt` imports nothing — no Compose, no Android — precisely so a
plain JVM could execute it. `android/verify/run.sh` is what finally does.

The obvious shape for it, an `android/app/src/test` source set driven by
`./gradlew test`, is the one it deliberately avoids. That needs JUnit resolved
through gradle, and this repository's Android build runs `--offline` against
whatever the local cache holds; a check that only runs on a machine which has
already downloaded the right test dependencies is a check nobody runs. It would
also drag the whole AGP pipeline in to execute a function that touches no
Android API.

So the script compiles the runtime files it can run, plus two of its own:

    android/verify/gen.go      the case tables, as Kotlin source
    android/verify/Harness.kt  the comparisons, and main()

Two decisions go through it now, both for the same reason — a Kotlin file that
imports nothing can be executed off a device. `GrMobSelectMenu.kt` is how a
flat option list becomes a picker menu; `GrMobProgress.kt` is what a
`core.ValueRange`'s three numbers amount to.

The table crosses as **Kotlin source** rather than as JSON, which is the one
place this harness differs from `ios/verify`. Swift decodes a JSON transcript
with `Decodable` from its standard library; Kotlin's standard library has no
JSON parser and neither does the JDK, so a JSON fixture here would mean either
a dependency — the thing the harness exists to avoid — or a hand-written parser
standing between the fixture and the code under test.

The compiler is `kotlinc` if one is on `PATH`, and otherwise the compiler jars
in the gradle cache the Android build already populates. If neither is present
the pass **skips** rather than fails, the same stance `ios/verify` takes toward
a missing iPhoneOS SDK.

#### And the Kotlin that *does* import Compose

Everything above is about the two files that import nothing. The other nine
runtime files — the renderer, the style solver, the map, both editors — import
Compose and the Android SDK, and for a long while nothing in this repository
compiled any of them: they were held to their contracts **textually**, by
`mobile/verify` searching their source for the calls they are supposed to make.
Those tests are real checks of the rules and no check at all of whether the file
resolves.

`android/verify/sources.sh` closes that. It type-checks the whole package:

    android/gradlew :app:printVerifyClasspath    the resolution + the platform jars
    the Kotlin compiler from the gradle cache    with the Compose compiler plugin
    com.grmob.runtime                            always
    com.grmob.app                                when app/libs/grmob.aar exists

Three choices in it are worth knowing about.

**It is not `:app:compileDebugKotlin`.** That task needs `app/libs/grmob.aar`,
which is gitignored and produced by `android/build.sh` — gomobile, the NDK and a
Go toolchain. `com.grmob.runtime` imports nothing from that `.aar`, deliberately,
so it can be compiled on a checkout that has never run gomobile. That is the
difference between a check that runs and a check that needs half an hour of
setup first, and it is why the app layer is a second stage rather than the whole
thing.

**The classpath comes from gradle, not from a glob over its cache.** A cache
holds whatever versions anything ever resolved, and this one holds two of every
Compose artifact: 1.6.8, which the BOM pins, and 1.10.0. Newest-wins picks
1.10.0 and the pass then fails on `Modifier.animateItemPlacement`, an API that
1.6.8 has and 1.10.0 removed — a classpath assembled by guesswork failing on the
library rather than on the source, which is the one thing a compile check must
never do. Same lesson the fixed-size census records about its sources jar,
arriving from the other direction.

**The Compose compiler plugin is required, not preferred.** Without it
`@Composable` is an ordinary annotation and `fun Bad() { Text("x") }` compiles
clean; with it, that is two errors. A compile without the plugin is a weaker
check wearing the same OK, so its absence is a named SKIP. It also has to be the
compiler's own version — which is why this pass always uses the cache's matched
pair and never a `kotlinc` on `PATH`, the opposite of what `run.sh` prefers.

The first run of it found that `GrMobCodeEditor.kt` called
`GrMobNode.isDisabled()`, which `Renderer.kt` declared `private` — file-private,
in Kotlin, for a top-level declaration. The Android app had not compiled since
the editors landed and nothing here could have said so.

Whether the menu is **open** is the renderer's own state and nothing else's.
There is no prop for it and no patch describes it; the *selection* stays
controlled like every other input's value. A picker that closed on every
unrelated re-render would be unusable, which is what putting the flag in the
tree would cause.

A picker takes no keyboard focus stamp either (`focusableLeafTypes` in
`core/focus.go`). It looks like a field and reads a field's theme base, so the
omission reads as an oversight — but the set is about the *keyboard*, and a
menu is not a keyboard target on either phone. The web is the outlier: a
browser focuses a `<select>` happily, and `htmlout` exports the `autofocus`
for it, because focus there is a document concept rather than a keyboard one.

### An overlay on device

`core.ZStack` is a SwiftUI `ZStack` and a Compose `Box` — the two constructs
the node type was named for — and both are told to centre their content
explicitly. On SwiftUI that restates a default; on Compose it overrides one,
since a `Box` places its children at the top-start corner. Stating it in both
is what keeps the alignment comparable from the other renderer, and
`mobile/verify`'s `TestNativeZStackOverlaysItsChildren` reads both.

The layers are rendered through neither renderer's *flex* children loop: an
overlay divides no leftover space along an axis, so there is no `FlexGrow`
weight to hand a layer and no cross-axis stretch to apply. A layer that wants
the stack's full extent states its own dimensions.

`core.StackAlign` is the per-layer opt-out from that centring, and it is the
one place these two constructs stop being interchangeable.

- **Compose** spells it directly: `Modifier.align(Alignment.*)` exists inside
  `BoxScope` for exactly this, it places without resizing, and a placed child
  still contributes its size to the `Box`. The children are looped over inside
  `GrMobZStack` rather than handed to `RenderChildren` only because the
  modifier has to be built per child, in that scope.
- **SwiftUI** has no equivalent — a `ZStack`'s `alignment:` is the stack's, not
  the layer's — so the whole stack is a custom `Layout` (`GrMobStackLayout`)
  that measures every layer, sizes to the largest, and places each one at its
  own anchor. The layer's anchor rides to the layout on a `LayoutValueKey`,
  because a `Layout` sees opaque subview proxies and cannot get back to the
  `GrMobNode` a child came from.

This used to be `.frame(maxWidth: .infinity, maxHeight: .infinity, alignment:)`
around the placed layer, which is SwiftUI's own idiom for the job and carried
this target's one divergence: **a filling frame is greedy**, so an *unsized*
stack with an aligned layer grew to its parent's proposal on iOS where a Compose
`Box` and a CSS grid track both stay the size of their largest child. It was
documented in four places, avoidable by pinning the stack's box, and pinned
nowhere — `ios/verify` type-checks and replays, and neither measures a size.

The `Layout` closes it, and the arithmetic behind it is a pure CoreGraphics file
(`GrMobStack.swift`) for the reason `GrMobFlexSolver` is one: a `Layout` needs a
view hierarchy to exercise and a function from numbers to numbers does not. So
`ios/verify` now *measures* the overlay — the container sizing to the largest
child on each axis independently, all nine anchors against a known box, the
bounds' origin being added, an oversized layer overhanging rather than being
clamped — where before it could only check that the file compiled.

#### Where the line between the two files is

It started at the arithmetic, because that part was obviously testable, and
that left three decisions on the far side of it with a type-check as their only
reader: measuring children with the **incoming** proposal rather than an
unspecified one, refusing to **clamp** the container to that proposal, and
**re-proposing `bounds.size`** at placement. None of the three is arithmetic
and all three are load-bearing — the first decides whether a greedy background
still covers anything, the second *is* the divergence above — so the line moved
to take them in.

What made that possible is a two-method protocol. A `LayoutSubview` is an
opaque proxy with no public initializer, so a test can never construct one and
every rule written in terms of `Subviews` needs a running app; but the only two
things the layout asks a subview are "how big are you if I offer you this" and
"where did you ask to sit". `GrMobStackLayer` is those two, `GrMobProposal` is
SwiftUI's `ProposedViewSize` with the framework taken out, and
`GrMobStackSolver.containerSize(layers:proposing:)` and `.placements(layers:in:)`
are the decisions, run in `ios/verify` against a fake that records every offer
it was made.

The *vocabulary* moved out too, and later. Converting between `ProposedViewSize`
and `GrMobProposal` was three field-copying expressions inside `Renderer.swift`
— "three lines with no decision in them", which was true and still left them on
the half of this target that nothing runs. A swapped axis or a dimension dropped
to `nil` compiles, draws, and is invisible to the source-text pin that catches a
`Layout` computing a size itself. So both directions live in
`GrMobStackBridge.swift` and `ios/verify` runs them: the obstacle was never
SwiftUI, it was `LayoutSubview` in particular, and a `ProposedViewSize` is an
ordinary public struct a check can construct. It is a separate file from
`GrMobStack.swift` because it needs `import SwiftUI`, and that file's claim to
import CoreGraphics and nothing else is what lets it be linked into a plain
command-line binary.

What is left in `Renderer.swift` is `subviews.map`, one `sizeThatFits` and one
`place()`.

What still needs a simulator is the assumption underneath: that a real
`LayoutSubview` answers `sizeThatFits` the way the fake does. That is SwiftUI's
behaviour rather than this framework's.

Both mappings return "no placement" for the centre rather than the platform's
own centre constant, so "said nothing" and "asked for the centre" stay one state
all the way down. `mobile/verify`'s `TestNativeStackAlignmentsCoverEveryPlacement`
holds the arms to `core.StackAlignments()`,
`TestNativeStackAlignmentsLeaveTheCentreToTheCatchAll` holds them to *not*
carrying an arm for it, and `TestNativeZStackOverlaysItsChildren` now also fails
if a filling frame ever comes back.

This is the counterpart to the `Box` fix: `Box` and `SafeArea` used to be built
from these same two constructs and were moved onto the column implementations,
because both DOM targets stack them. `ZStack` is the node type that gives the
behaviour a name.

### `core.Modal` and the dialog role

Nothing to do here, and that is the point. iOS presents a Modal as a sheet and
Android as a Compose `Dialog`; both announce themselves as modal and confine
exploration to their own content, so the `role="dialog"` / `aria-modal` pair
the two DOM renderers write by hand comes free on device. That asymmetry is why
`core.Role` has no `RoleDialog` for an author to set — see
[Styling & Theming](../concepts/styling-and-theming.md#roles-a-node-type-carries-for-itself).

### `Disabled`

`core.Disabled(bool)` becomes the platform's own inert state — `enabled =
false` on Android, `.disabled(true)` on iOS — and propagates to the subtree on
both, so one declaration can freeze a whole section. Full contract, including
why it deliberately changes no colors and why it must not also be announced
through the accessibility label, in
[Styling & Theming](../concepts/styling-and-theming.md#disabled).

## Testing without a device

`render.Manager` and the bridge are plain Go — the exact call sequence a
shell makes runs in a test (`examples/todoapp/app_test.go`,
`mobile/bridge_test.go`). Most development happens at that level; the
simulator/emulator is for final verification.

`ios/verify/run.sh` goes one step further without needing Xcode or a
simulator: Go generates a bridge transcript from the demo app, Swift replays
it through the real `GrMobNode`/`GrMobStyle`/`TreeStore` files and deep-
compares the resulting tree against Go's final render, and the view layer
(`Renderer.swift`, `GrMobFlexStack` included) is then type-checked. Only Go
and the Xcode Command Line Tools are required.

### Asking the solver whether a change was free

`ios/verify` carries `GrMobFlexSolver` and `GrMobStackSolver`, split out of the
SwiftUI `Layout`s precisely so they can be exercised without mounting anything.
That makes them the one thing in the repository that can answer a question the
web answers with a screenshot: **did this layout change move any pixels?**

`components.GroupHeader`'s band is the case it was first asked about. The band's
padding used to be on the `Row` and is now on the growing control inside it, so
that a press lands on the whole band rather than a strip in the middle of it —
and the argument that the move is *free* is that padding on a stretched child
fills exactly the space the same padding on its parent held. That was verified
on the web by the pixels it did not move, and everywhere else it was an
assumption.

`internal/bandfixture` reads the band's real numbers off a rendered
`GroupHeader` — the insets, the gap, the flex-grow, the alignment — and derives
the arrangement they came from by a stated rule; the pair rides in the same
transcript the picker cases do, and `ios/verify/band.swift` solves both at four
container widths. The content sizes are synthetic, because what is under test is
the arithmetic of an arrangement and not the width of a string.

The answer is **yes, with two recorded exceptions**, and finding them is what
the check was worth:

- **Under overflow the two diverge.** This solver shrinks each child in
  proportion to its base size, and a base includes that child's own padding — so
  the control's 32 points are inside the proportion in one arrangement and
  outside it in the other. At a 120pt offer a 100pt label gets 63.14pt with the
  insets on the control and 64.52pt with them on the `Row`. It needs a second
  child to show: with the count hidden the control is alone on the line and is
  clamped to the container either way. CSS distributes shrink over the *inner*
  flex base size rather than the outer one — and a browser **has** now been
  asked: `wasm/verify/browser.mjs` mounts the same fixture in a real Chrome and
  measures 64.52pt in *both* arrangements, at every offer. So this is a genuine
  cross-target divergence — one renderer's shrink proportion counts a child's
  padding and the other's does not — rather than an artefact of either
  implementation, and each target asserts its own answer.
- **A badge taller than the control would change the band's height.** With
  children centred, a `Row`'s height is its tallest child plus its own vertical
  padding, so moving that padding onto one child stops it being added to the
  other. It cannot happen to the real band — the control carries more vertical
  padding than the badge and both wrap the same caption type — so it is a case
  with a made-up badge, asserted in the *other* direction so the agreement of
  the real ones is not holding for a reason nobody stated. The browser agrees
  about this one: it is CSS's own rule, and the SwiftUI `Layout` was checking a
  transliteration of it.

`mobile/verify` needs even less — just Go. It holds the checks that have to
hold in *both* native renderers at once, which is why they live under
`mobile/` (the bridge surface both shells are written against) rather than in
either platform's own harness. Its checks read native source as text, so they
run under `go test ./...` alongside everything else: `ContentMode` coverage was
the first, and the three alignment types are now held the same way — twelve
dispatches across `Renderer.swift`, `GrMobFlex.swift` and `Renderer.kt`, plus
one array literal.

The price is a shape those dispatches must keep, stated in a comment beside
each: one arm per line, string literals first on the line, the catch-all last,
and every value listed explicitly — including the ones the catch-all would have
handled anyway. Those redundant arms are the point and must not be tidied away;
a value that falls through is indistinguishable from one nobody considered.

### A fixed-size box, on four targets

`core.Spacer` became "a `Box` with a fixed size" so that all four targets would
lay a Spacer's children out the same way, and the note closing that work
recorded a difference nobody had measured: Compose constrains a child to the
declared size where the DOM was believed to let it spill. It is not a `Spacer`
property — it is what every fixed-size container on that target does — and
nothing anywhere had asked whether the four targets agree about overflow for
**any** fixed-size box.

| | main axis | cross axis | how |
|---|---|---|---|
| WASM runtime | squeezed | spills | measured |
| htmlout | squeezed | spills | inherited |
| SwiftUI | **measured** | spills | main axis measured, cross axis derived |
| Compose | squeezed | **squeezed** | derived, from the pinned version's source |

The DOM row was measured, in a real Chrome, and it is not the blanket "spills"
the note assumed — see [the WASM
harness](wasm.md#a-fixed-size-box-on-four-targets) for that half.

The SwiftUI row is half measured. Its cross axis is a SwiftUI fact and stays a
reading of the call site; its main axis is not SwiftUI's at all — the squeeze
comes from `GrMobFlexSolver`, which is this repository's own arithmetic and
which `ios/verify` executes. So `checkFixedSizeContainer` in
`ios/verify/flex.swift` runs the census's box through the solver, on both axes'
worth of container, and gets the browser's answer. It uses browser.mjs's
numbers, and `wasm/verify`'s `TestTheFixedSizeCensusUsesOneSetOfNumbers` holds
the two harnesses to one fixture: two passes agreeing about different boxes is a
weaker statement than the row makes, and neither pass could tell.

The rest is derived from the platform call each renderer makes, and
`mobile/verify/fixedsize_test.go` is what holds the renderers to those calls:

- **SwiftUI.** `.frame(width:)` / `.frame(height:)` *proposes* a size to its
  content and reports the fixed size to its parent; content that insists on
  being larger keeps its size and is drawn overflowing, and nothing clips it
  (`Renderer.swift` spends `.clipped()` on exactly two image content modes).
  The main-axis squeeze is not the frame's doing at all — it is
  `GrMobFlexSolver`, this repository's own CSS flex arithmetic, which is why
  that column matches the DOM's.
- **Compose.** `Modifier.width(n)` / `Modifier.height(n)` set the child's
  **minimum and maximum alike**. A maximum is what the other three do not
  impose, and it is the whole of the divergence.

The two things a check has to refuse are the near-misses, not the far ones:
`Modifier.requiredSize` ignores the incoming constraints entirely and
`Modifier.sizeIn` sets a range, and either would leave the mapping compiling
with this target's row silently wrong. On the Swift side it is `.clipped()`,
which would compile, look tidier, and make one target hide an overflow the other
three show.

**Which `foundation-layout` the Compose half is read from.** It used to be
whichever one a gradle cache happened to hold — 1.10.0 on the machine where the
paragraph was written, while `android/app/build.gradle` pins the Compose BOM at
`2024.06.00`, which resolves `foundation-layout` to 1.6.8. A reading of the
wrong version is the mistake `gobindVersion` exists to prevent one file over,
and it is worse here: the claim is prose about a third party's arithmetic, and a
reader cannot tell a paragraph that was checked from one that was true two
releases ago.

Two pieces close that, and neither costs a network call at test time:

- **There is only one version.** `android/app/build.gradle` declares a
  `composeLayoutSources` configuration for the sources jar and names *no*
  version for it: the Compose BOM sits on that configuration and resolves it,
  exactly as it resolves the artifacts the app compiles against. So the source
  read cannot be a different release from the source built against — not
  because something compares two numbers, but because there is one number.
  `TestTheComposeSourcesTakeTheirVersionFromTheBOM` holds that shape, since the
  one way to lose it is silent: a version written back into the coordinate
  resolves perfectly and is then checked against nothing.
- **The claims are read from the source.**
  `./gradlew :app:fetchComposeLayoutSources` puts the sources jar in the cache
  once, and `TestTheComposeCensusClaimsAreWhatTheSourceSays` then reads
  `Size.kt` and `RowColumnMeasurementHelper.kt` out of it at the version the BOM
  gives. Six readings, not two: that `Modifier.width` is
  `SizeElement(minWidth = width, maxWidth = width, enforceIncoming = true)`;
  that the zero-weight measure branch offers a child
  `(mainAxisMax - fixedSpace).coerceAtLeast(0)` against a cleared minimum; that
  the spacing after a child is clamped to what is left; that the trailing
  spacing comes back off after the loop; that `mainAxisLayoutSize` is raised to
  the Row's minimum and *never* lowered to its maximum; and that `SizeNode`
  reports `layout(placeable.width, placeable.height)` unclamped. The last four
  are what `internal/pinfixture`'s transcription of that loop rests on, and
  until they were listed here only the first `mainAxisMax - fixedSpace` was read
  out of the jar by anything. Machines that have never fetched skip that half,
  with the command in the skip message — which is the honest state for a check
  whose subject has to be downloaded.

The call site pins stay either way: they are about *this* repository's code,
which no reading of androidx can answer for.

### `core.FlexShrink(0)` on a target with no proportional shrink

`core.FlexShrink` was a web-only prop for two releases, and then half of it
stopped being one: `GrMobFlexSolver` implements CSS's scaled-base rule, so every
factor means on iOS what it means in a browser. Compose was the target left out,
and the reason was real — a Compose `Row` has no proportional shrink *at all*.
Its measure policy walks the unweighted children in order and offers each one
`mainAxisMax - fixedSpace`, the main-axis space its predecessors did not take.
There is no factor anywhere in that arithmetic for a fractional value to scale.

That argument covers the fractional values and not the one that matters most.
Zero is not a proportion, it is a refusal, and a refusal is expressible:

```kotlin
private fun Modifier.pinMainAxis(horizontal: Boolean): Modifier = layout { measurable, constraints ->
    val unbounded = if (horizontal) constraints.copy(minWidth = 0, maxWidth = Constraints.Infinity)
                    else            constraints.copy(minHeight = 0, maxHeight = Constraints.Infinity)
    val placeable = measurable.measure(unbounded)
    layout(placeable.width, placeable.height) { placeable.place(0, 0) }
}
```

The child is measured against its own content rather than against what is left,
and — the half that is easy to get wrong — the size reported back to the `Row`
is the measured one, not a value clamped to the incoming constraints. So the
row's running total passes its own maximum and the overflow is visible, which is
what `flex-shrink: 0` does everywhere else. Clamping on the way out compiles,
looks better behaved, and produces a third thing that matches no target: a child
drawn spilling out of a box its parent still believes it fits inside.
`mobile/verify` refuses that spelling by name.

```
   Row(maxWidth = 120)   [  pinned child, 200 wide  ][ next child ]
                         └──── reports 200 ────┘      └ offered 0 ┘
```

What still diverges is the siblings, and that sentence used to be the end of the
matter. `internal/pinfixture` turns it into numbers: one overflowing `Row`, three
children, the pin moved through all three positions, the control with no pin at
all, and two of those arrangements repeated with a gap. `GrMobFlexSolver` solves
each one (`ios/verify/pin.swift`), a real Chrome lays the same six `Row`s out
(`wasm/verify`'s check 12), and the Compose column is a **transcription** of
`foundation-layout`'s zero-weight measure loop — not androidx's code, and
labelled as such wherever it appears.

| | CSS | Compose |
|---|---|---|
| no pin | 24, 80, 16 | 60, 60, 0 |
| pin first `[P,A,B]` | **200**, 0, 0 | **200**, 0, 0 |
| pin middle `[A,P,B]` | 0, **200**, 0 | 60, **200**, 0 |
| pin last `[A,B,P]` | 0, 0, **200** | 60, 40, **200** |
| pin middle, 8px gap `[A,P,B]` | 0, **200**, 0 | 60, **200**, 0 |
| pin last, 16px gap `[A,B,P]` | 0, 0, **200** | 60, 40, **200** |

Three things are readable there and none of them was before. The pinned child is
200 in every row and on both targets, so the declaration means one thing
everywhere — including on the target that had no way to express it — and order
does not matter to it: `remaining` is ignored whether the pin is first or last.
The control row is what makes that a statement about `core.FlexShrink(0)` rather
than about a `Row` with a big child in it: same container, same children, one
factor apart, and that child is 80 instead of 200. And the divergence is the
*siblings*: CSS shares the deficit among the items that can shrink, in proportion
to their bases, so a flex line's sizes do not depend on the order; a Compose `Row`
gives each child what the ones before it left, so its answer does.

The pin-first row is the one where the two agree, and it is asserted as an
agreement for the reason the band census asserts an unbadged band — an "it
diverges" with no case that does not is a claim about whatever happened. The
agreement is a coincidence of two rules rather than a shared one: CSS clamps the
shrinkable children to zero because the deficit exceeds their bases, and Compose
offers them nothing because the pinned child had already taken more than the
`Row` had.

One consequence falls out of the same two lines and no census row had stated it.
`mainAxisLayoutSize` is `max(content, mainAxisMin)` and is never coerced *down*
to `mainAxisMax`, and `Size.kt`'s own node reports `layout(placeable.width, …)`
unclamped — so a fixed-width `Row` whose children overflow is measured wider than
it was told to be (200, 260, 300 above) rather than clipping. That is what makes
the pin an overflow on this target rather than a clip, which is what
`overflow: visible` does on the other three. The spacing collapses with it:
`spaceAfterLastNoWeight` is `min(spacing, what is left)`, so a `Row` that has
spent its main axis inserts no gap after the child that spent it.

That last sentence is what the two gapped rows are for, and it is the reason
each one's columns of extents are its ungapped twin's twice over. A gap is used
space in a flex line like any other, so adding 8px — or 16px — to a `Row` whose
shrinkable children were already clamped to zero moves no child on either
target. What changes is underneath:

| | CSS | Compose |
|---|---|---|
| pin middle, 8px gap | 8, 8 | 8, 0 |
| pin last, 16px gap | 16, 16 | 16, 4 |

The first row is the two ENDS of `min(spacing, what is left)`. The `Row` charges
its spacing after the lead child, has nothing left after the pin, and charges
none; a flex line charges both regardless, and is 8px wider for it.

The second row is the middle, and it is there because a `min` has three answers
and the first row reaches two. With `[A,B,P]` and a 16px gap the `Row` charges 16
after the lead child, is left with 4 when it reaches the second, and charges 4 —
neither the spacing nor zero. Every other gap in the fixture is one or the other,
which is also exactly what "charge the gap unless the `Row` has overflowed" would
produce, so without this row the transcribed rule and that simpler wrong one are
indistinguishable everywhere the fixture looks.

Until those rows existed every case in `internal/pinfixture` carried `gap: 0`, so
the sentence three documents repeat had never been put in front of a browser or a
solver — `MeasureCompose` implemented the line and nothing compared it with
anything.

The CSS column is a browser's, and for a while it was not. `GrMobFlexSolver` is
this repository's own flex arithmetic rather than a browser's, and the band
census one section down records a place the two part company — under overflow,
when a child has padding, the solver shrinks in proportion to a base that
includes that padding and CSS does not. The pin fixture's children have none, so
the two rules coincide; that sentence was reasoning, made by whoever wrote the
fixture and asked of nobody. `wasm/verify`'s check 12 mounts the six `Row`s and
measures them, and a browser produces the CSS column above exactly. It recomputes
nothing — every claim it makes is one the fixture states, held against pixels:
the pinned child keeps its base, the extents match the Compose column precisely
where `mainsAgreeWithCSS` says they do, and a child's width does not depend on
where it sits. Those three pin the three pinned rows to their exact numbers; the
control row's proportional split is still the solver's alone, because a flex line
transcribed into JavaScript would only ask whether two transcriptions agree.

What none of this is, is a measurement of Compose. The transcription's weakest
link is that somebody read androidx's loop and wrote it out; every other link is
checked, and `internal/pinfixture`'s header names them.

Only the zero is read. `GrMobStyle.kt`'s `shrinkFactor` maps the sentinel the
way every other runtime does (`wasm/verify`'s `shrink_test.go` pins all four
spellings of `core.ShrinkNone` to the same number), and `shrinkPinned` is what
the two children loops consult. A weighted child is not given the modifier, and
that is not an omission: `Modifier.weight` already fixes that child's main axis
as both a minimum and a maximum, and CSS never applies grow and shrink at once
either — one divides positive free space, the other negative.

### Why the band's census has three rows

The same limit decides where `internal/bandfixture` stops. The band's two inset
arrangements are checked against a real browser and against `GrMobFlexSolver`,
and both are executable for the same reason: the arithmetic is ours, or the
browser is a browser. Compose's `Row` is neither — a `FlexGrow` child is handed
`Modifier.weight` and androidx's own measure policy does the distributing — so
"do the two arrangements agree on Compose" is a question about androidx's code.
`android/verify` runs Kotlin on a plain JVM, which is what made it look like the
place to ask, and the thing it can run is Kotlin that *imports nothing*;
a Compose measure policy measures `Measurable`s into `Placeable`s through
`compose-ui`, which needs the Android runtime — and `Placeable` has `internal`
abstract members, so the interfaces cannot even be implemented from another
module, runtime or no runtime.

That is why `internal/pinfixture` transcribes the zero-weight branch rather than
running it, and why it transcribes only that branch: the weighted half is what
this section is about, and modelling it would be putting the band's fourth answer
in a Go file and calling it measured.

What the derivation says is that Compose agrees with the web, for a third reason
again: its `Row` has no proportional shrink at all, an unweighted child is
measured with what is left and a weighted one gets `(available − fixed) / total
weight`, and the band's badge is unweighted — so the whole deficit lands on the
growing control in both arrangements, the badge keeps its width where both other
renderers shrink it, and the two arrangements come out equal because the
control's padding is inside its weighted extent either way.

`TestTheComposeRowDelegatesItsDistributionToCompose` holds the *premise* rather
than the answer: the census stops at three rows because Android delegates, and a
renderer that stopped delegating would put the fourth answer back within reach
and make it one of ours to be wrong about.

### The gates, and why they are functions

Every harness in this repository skips rather than fails when the machine is
missing an optional toolchain — `ios/verify` without an iPhoneOS SDK,
`android/verify` without a Kotlin compiler — and both of those were inline shell
conditions whose arms could only be reached by *owning a machine with the
fault*. On a Mac with Xcode the SKIP branch never ran; in a bare container the
OK branch never did. A gate's failure mode is a swapped or misordered stance,
and that is exactly the mutation that leaves a pass green.

So each is a function of values now — `ios/verify/gate.sh`,
`android/verify/gate.sh` — printing a verdict, with a `gate_test.sh` beside it
that hands over every combination, run by `run.sh` before the pass itself. Same
move `wasm/verify/startup.mjs` made for the browser pass and `localCopyGate`
before it.

**Extracting the Android one is what found the order was wrong.** The script
asked for a `kotlinc` first and took that path, and only afterwards checked for
a `java`. But `kotlinc` is a JVM application: on a machine with a Kotlin
compiler and no JDK the pass did not skip, it ran `kotlinc`, which failed, and
`set -e` turned an absent optional toolchain into a red pass. Java is decided
first for both paths now, and the order is asserted directly rather than being a
property of how the arms happen to be written.

The iOS gate gained a distinction on the way past. `[ -n "$sdk" ] && [ -d "$sdk" ]`
is two different machines with two different remedies — Xcode is not installed,
or `xcrun` names an SDK that is not there, which is what a moved or half-removed
Xcode leaves behind — and the combined test told them apart for nobody.

### The bridge stand-in

Three files in `ios/GrMob/App` — the `@main` entry point among them — begin
`import GrMob`, the module a `gomobile bind` produces, so they could not be
type-checked without full Xcode and a gomobile toolchain. `ios/verify`
compiles a hand-written stand-in module of that name instead
(`ios/verify/gomobile_stub.swift`), and the app layer type-checks against it.

A hand-written stand-in for generated code is a copy, and a drifted copy is
worse than no check at all — the shell would keep type-checking green against
a bridge Go no longer has. So `mobile/verify/gomobilestub_test.go` holds it to
package `mobile`, in two directions and at two levels:

- **The names.** Every bindable exported function and every exported interface
  has a declaration, under gobind's naming (`Mobile` + the Go name, and
  `…Protocol` for an interface, since Swift suffixes the protocol to break its
  collision with the class of the same name). Nothing in the stub claims a Go
  symbol that is not there.
- **The signatures.** Each declaration is character-for-character what gobind
  would emit — parameter types and order, argument labels, nullability and the
  return — for the package functions and the protocols' methods alike. Adding
  a bindable function without adding it here fails `go test ./...`, and so does
  renaming one of its parameters.

- **The header comment.** The declarations were pinned character-for-character
  and the comment above them was a hand-written description of the rules doing
  the pinning — the naming, the nullability asymmetry, the result arms. It agreed with the checker because it was written from it, which is the
  same copy-that-drifts this whole stand-in exists to refuse. So the
  load-bearing half of it is now written as rows in a delimited block, and the
  test reads them out of the comment and holds each to the thing it describes:
  the version to `go.mod`, the prefix and suffix to the names the checker
  builds, each type row to `swiftType`, and each result row to the whole
  declaration the checker produces for a signature of that shape. The prose around the rows is not checked
  and is not meant to be — a paragraph explaining *why* nullability is
  asymmetric cannot be wrong in the way `string String? String` can.

The type mapping is three rows and a protocol rule (`gobindSwiftTypes`), which
is all this bridge's narrow surface can need — a Go type outside it already
stops the bind. It is split by position because gobind's nullability is not
symmetric: a `string` parameter arrives as `String?` and a `string` result as
`String`, and a stub that took `String` everywhere would accept shell code the
real framework rejects.

**The facts come from gobind, not from a header somebody once produced.**
`golang.org/x/mobile` is in this module — held there by `go.mod`'s `tool` block,
which pins the gobind the stub is written against — so the mapping is read off
`bind/genobjc.go`: `objcParamType` for a parameter, `objcType` for every other
position, `funcSummary` for the shape of a result clause. `gobindVersion` pins
the version those readings were made at, and a bump fails until somebody looks.

That provenance closed a refusal for **a returned bound interface**.
`objcParamType` special-cases exactly one Go type, `String`, and falls through
to `objcType` for everything else — so the asymmetry the two columns exist for
is `string` and nothing else, and a returned protocol is `_Nullable` exactly as
a parameter is.

#### The result arms, and the one thing the module cache could not settle

A **multi-result signature** used to be refused, and the refusal said: run
`gomobile bind` once and write the row from what it produced, because writing it
on a guess is the one thing that must not happen. That instruction was right and
it was the whole problem — the next bridge function of that shape was blocked on
somebody having a Mac.

Somebody ran it. A package with one function and one interface method of every
result shape was bound with `gomobile bind -target=ios`, and the module it
produced was read back through `swiftc`. Every row below is that, not a reading
of the generator and not reasoning about Clang:

| Go results | package `func` | interface method |
|---|---|---|
| `()` | — | — |
| `T` | `-> T` | `-> T` |
| `error` | `(_ error: NSErrorPointer) -> Bool` | `throws` |
| `(string, error)` | `(_ error: NSErrorPointer) -> String` | `(error: NSErrorPointer) -> String` |
| `(Iface, error)` | `(_ error: NSErrorPointer) -> MobileXProtocol?` | `throws -> MobileXProtocol` |
| `([]byte, error)` | `(_ error: NSErrorPointer) -> Data?` | `throws -> Data` |
| `(int, error)` | `(_ ret0: UnsafeMutablePointer<Int>?, _ error: NSErrorPointer) -> Bool` | `(ret0_: UnsafeMutablePointer<Int>?) throws` |
| `(bool, error)` | `(_ ret0: UnsafeMutablePointer<ObjCBool>?, _ error: NSErrorPointer) -> Bool` | `(ret0_: UnsafeMutablePointer<ObjCBool>?) throws` |

**No package function ever throws**, and the prediction that said so was
correct. gobind emits every package-level func as a plain C function
(`genFuncH`: `FOUNDATION_EXPORT … s.asFunc(g)`), and Clang's error convention —
the rewrite of a trailing `NSError**` into a Swift `throws` — is the
Objective-C *method* convention. A C function gets none of it without an
explicit `swift_error` attribute, and gobind emits none:

```
func F() (string, error)
  ->  FOUNDATION_EXPORT NSString* _Nonnull MobileF(NSError* _Nullable* _Nullable error);
  ->  public func MobileF(_ error: NSErrorPointer) -> String
```

**A bound interface method does throw — except in one case no amount of reading
would have produced.** The convention needs a return it can use to signal
failure: `BOOL`, or a nullable object. A `(string, error)` method returns
`NSString* _Nonnull`, which is neither, so it keeps its explicit error parameter
and does not throw. A `(Iface, error)` method returns a nullable object, throws,
*and loses the optional* — `nil` is the error signal, so it can no longer also
be a value.

That last row is about the **annotation** and not about the Go type, and the
`[]byte` row is what made the difference matter. `[]byte` was refused for two
releases: its C spelling was legible (`NSData* _Nullable` in both positions, on
one line of gobind's own golden) and the gap named in the refusal was the
two-result split — "a nullable first result stays the return where a scalar
moves into an out-pointer, and no golden exercises `([]byte, error)`". The split
is `isNullableType`, one line of `bind/types.go`, and it is nullable because
`nil` is assignable to a slice. So the value stays the return, and what remained
were two questions for a compiler rather than for a bind: what `NSData*` is
called in Swift, and what the convention does to a *method* that returns one.
`ios/verify/importer.h` declares both — a C function and a protocol method — and
`importer.swift` states the answers as types. The refusal that remains is `uint8`
and its alias, and it is about the C rather than about the Swift: gobind spells a
bare `byte` that nothing it emits declares, so there is no C to hand the importer
at all.

The reading behind that file used to run only where Swift does. `mobile/verify`
holds its two type tables to what `importer.swift` *says*, and `swiftc` is what
holds that file to the truth — so on any machine that is not a Mac the pairing
was verified, the reading behind it was not, and a green `go test ./...` could
not tell you which. It runs the typecheck itself now, skips with a named reason
where it cannot, and `GRMOB_IMPORTER=required` turns that skip into a failure for
a machine that is supposed to have the toolchain. `ios/verify/run.sh` calls the
same test rather than spelling the command a second time.

**Two shapes are still refused, and they are gobind's refusals rather than this
table's.** It stops with `too many result values` for three or more, and
`second result value must be of type error` for a two-result signature whose
second is something else, and builds nothing either way. So neither is a mapping
the table is missing; the fix is to change the Go signature.

The bind also turned up a hole. `error` was in neither the bindable-type set nor
the spelling table, on the reasonable-looking grounds that nothing in `mobile`
returns one — but gobind binds it, so a bridge function that grew an error would
have been judged unbindable, gone undeclared in the stub, and taken
`GomobileBridge.swift`'s type-check with it, silently. The two tables now answer
for each other.

The signature half was left out for a while, on the argument that a wrong
signature fails the Swift type-check the moment the shell calls it. That
assumed every declaration has a call site. Three do not — `MobileDataDir`,
`MobileRenderAgain` and `MobileReportHostEvent` are all reachable from a shell
that never touches them — and for those the type-check proved nothing.
