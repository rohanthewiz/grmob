# Package core — Device services

```go
import "github.com/rohanthewiz/grmob/core"
```

Audio, camera, clipboard, haptics, local notifications, compass heading, location, maps and the app lifecycle.

One of 11 topic pages of [package core](core.md), which has the package overview and an index of every topic. This page documents the declarations in `core/audio.go`, `core/camera.go`, `core/clipboard.go`, `core/haptics.go`, `core/notifications.go`, `core/heading.go`, `core/location.go`, `core/mapview.go`, `core/lifecycle.go`.

## Index

- [Constants](#constants) — `DefaultMapZoom`
- [`func AngleDelta`](#func-angledelta)
- [`func AudioLoad`](#func-audioload)
- [`func AudioPause`](#func-audiopause)
- [`func AudioPlay`](#func-audioplay)
- [`func AudioSeek`](#func-audioseek)
- [`func AudioSetRate`](#func-audiosetrate)
- [`func AudioSkip`](#func-audioskip)
- [`func AudioStop`](#func-audiostop)
- [`func AudioToggle`](#func-audiotoggle)
- [`func CameraView`](#func-cameraview)
- [`func CancelNotification`](#func-cancelnotification)
- [`func Cardinal`](#func-cardinal)
- [`func DistanceMeters`](#func-distancemeters)
- [`func FormatLatLng`](#func-formatlatlng)
- [`func FormatRegion`](#func-formatregion)
- [`func Haptic`](#func-haptic)
- [`func HeadingActive`](#func-headingactive)
- [`func LocationAcquiring`](#func-locationacquiring)
- [`func LocationActive`](#func-locationactive)
- [`func MapView`](#func-mapview)
- [`func Marker`](#func-marker)
- [`func NormalizeDegrees`](#func-normalizedegrees)
- [`func OnAudioStatus`](#func-onaudiostatus)
- [`func OnHeading`](#func-onheading)
- [`func OnLifecycle`](#func-onlifecycle)
- [`func OnLocation`](#func-onlocation)
- [`func OnMapTap`](#func-onmaptap)
- [`func OnMarkerTap`](#func-onmarkertap)
- [`func OnNotificationTap`](#func-onnotificationtap)
- [`func OnRegionChange`](#func-onregionchange)
- [`func ParseLatLng`](#func-parselatlng)
- [`func PostNotification`](#func-postnotification)
- [`func ReadClipboard`](#func-readclipboard)
- [`func ReceiveAudioStatus`](#func-receiveaudiostatus)
- [`func ReceiveHeading`](#func-receiveheading)
- [`func ReceiveLifecycle`](#func-receivelifecycle)
- [`func ReceiveLocation`](#func-receivelocation)
- [`func ShowUserLocation`](#func-showuserlocation)
- [`func StartHeading`](#func-startheading)
- [`func StartLocation`](#func-startlocation)
- [`func StopHeading`](#func-stopheading)
- [`func StopLocation`](#func-stoplocation)
- [`func SweepNotifications`](#func-sweepnotifications)
- [`func WrapLongitude`](#func-wraplongitude)
- [`func WriteClipboard`](#func-writeclipboard)
- [`type AudioOpt`](#type-audioopt)
    - [`func AudioAutoplay`](#func-audioautoplay)
    - [`func AudioStartAt`](#func-audiostartat)
    - [`func AudioWithRate`](#func-audiowithrate)
- [`type AudioState`](#type-audiostate)
- [`type AudioStatus`](#type-audiostatus)
    - [`func CurrentAudioStatus`](#func-currentaudiostatus)
    - [`func (AudioStatus) Loaded`](#func-audiostatus-loaded)
    - [`func (AudioStatus) Progress`](#func-audiostatus-progress)
- [`type AudioTrack`](#type-audiotrack)
- [`type CameraNode`](#type-cameranode)
- [`type CameraProp`](#type-cameraprop)
    - [`func OnCapture`](#func-oncapture)
    - [`func OnError`](#func-onerror)
    - [`func SetFacing`](#func-setfacing)
    - [`func WithFlash`](#func-withflash)
    - [`func WithOverlay`](#func-withoverlay)
    - [`func WithStyle`](#func-withstyle)
- [`type HapticKind`](#type-haptickind)
    - [`func HapticKinds`](#func-haptickinds)
- [`type Heading`](#type-heading)
    - [`func CurrentHeading`](#func-currentheading)
    - [`func (Heading) Cardinal`](#func-heading-cardinal)
- [`type LifecycleState`](#type-lifecyclestate)
    - [`func CurrentLifecycle`](#func-currentlifecycle)
- [`type LocalNotification`](#type-localnotification)
- [`type Location`](#type-location)
    - [`func CurrentLocation`](#func-currentlocation)
- [`type Region`](#type-region)
    - [`func ParseRegion`](#func-parseregion)

## Constants

DefaultMapZoom is the scale a Region with no Zoom is drawn at: a neighbourhood, which is close enough to read street names and wide enough to hold more than one marker.

14 rather than comps.DefaultMapZoom's 15, and the difference is the difference between the two widgets. A static map answers "where is this one place"; a live map is usually showing a set, and one level out is about four times the area.

```go
const DefaultMapZoom = 14.0
```

<small>[core/mapview.go:182](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L182)</small>

## Functions

### func AngleDelta

```go
func AngleDelta(a, b float64) float64
```

AngleDelta is the signed shortest turn from a to b, in (-180, 180]: positive clockwise, negative counter-clockwise.

This is the arithmetic that makes a compass behave at the seam. Plain subtraction says the step from 359 degrees to 1 degree is -358, which is wrong by every measure that matters: it is a two-degree nudge, not most of a lap, and code that treats the difference as a magnitude (a change threshold, a smoothing filter, an animation) gets a spurious lurch once per rotation.

<small>[core/heading.go:163](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L163)</small>

### func AudioLoad

```go
func AudioLoad(track AudioTrack, opts ...AudioOpt)
```

AudioLoad replaces whatever is loaded with track and, by default, starts playing it. An empty URL is dropped here rather than sent, since every host would have to reject it separately and none could report that it had (the same rule OpenURL applies).

The status record is updated optimistically — see the package comment — and subscribers are notified before the command leaves, so a screen that re-renders on the notification already sees the new track.

<small>[core/audio.go:180](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L180)</small>

### func AudioPause

```go
func AudioPause()
```

AudioPause pauses the loaded track, keeping its position.

<small>[core/audio.go:219](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L219)</small>

### func AudioPlay

```go
func AudioPlay()
```

AudioPlay resumes the loaded track. After AudioEnded it starts over from the beginning. With nothing loaded it does nothing.

<small>[core/audio.go:211](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L211)</small>

### func AudioSeek

```go
func AudioSeek(seconds float64)
```

AudioSeek moves playback to the given second. The host clamps it to \[0, duration].

<small>[core/audio.go:239](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L239)</small>

### func AudioSetRate

```go
func AudioSetRate(rate float64)
```

AudioSetRate changes the playback speed; 1 is normal, 1.5 is the podcast listener's favorite. Non-positive rates are ignored — 0 would be a pause spelled confusingly, and the hosts reject it anyway.

<small>[core/audio.go:263](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L263)</small>

### func AudioSkip

```go
func AudioSkip(delta float64)
```

AudioSkip moves playback by delta seconds relative to where it actually is — negative to go back. The host does the arithmetic, not core: the position core knows is up to one status tick old, and a "+30s" computed from a stale number lands somewhere subtly wrong.

<small>[core/audio.go:253](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L253)</small>

### func AudioStop

```go
func AudioStop()
```

AudioStop unloads the track and releases the media session: the lock-screen controls disappear and the status returns to idle. Pause is what a user usually wants; Stop is for "sign out", "this content is no longer available", and the like.

<small>[core/audio.go:274](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L274)</small>

### func AudioToggle

```go
func AudioToggle()
```

AudioToggle plays when paused and pauses when playing — the one-button transport control. Anything else (loading, ended, error) is treated as "please play", which is what a user tapping the button means.

<small>[core/audio.go:229](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L229)</small>

### func CameraView

```go
func CameraView(props ...CameraProp) View
```

<small>[core/camera.go:17](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L17)</small>

### func CancelNotification

```go
func CancelNotification(id string)
```

CancelNotification takes down the notification posted under id, whether it is still on screen or already in the notification list. Cancelling one that is not there is harmless on every host.

<small>[core/notifications.go:162](https://github.com/rohanthewiz/grmob/blob/master/core/notifications.go#L162)</small>

### func Cardinal

```go
func Cardinal(deg float64) string
```

Cardinal names any bearing in degrees, normalising it first so a caller can pass an unwrapped or negative angle.

Each of the sixteen sectors is 22.5 degrees wide and \*centred\* on its point, which is the half worth stating: north is 348.75 through 11.25, not 0 through 22.5, so a bearing one degree west of north reads "N" rather than "NNW". The +11.25 before the divide is what shifts the sector boundaries off the points and onto the gaps between them.

<small>[core/heading.go:134](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L134)</small>

### func DistanceMeters

```go
func DistanceMeters(lat1, lng1, lat2, lng2 float64) float64
```

DistanceMeters is the great-circle distance between two coordinates, by the haversine formula.

Haversine rather than the flat approximation (scale the longitude by cos(lat), then Pythagoras) because the flat one is wrong in exactly the case a map application cares about: it degrades with latitude, and it breaks completely across the antimeridian, where two points a kilometre apart are 360 degrees of longitude apart on paper. The notification filter in this file is a consumer — a device crossing 180° must not be told it has travelled 40,000km — and so is any "within n metres of here" an app writes.

Haversine rather than Vincenty, which is the next step up: Vincenty solves on the ellipsoid and is accurate to millimetres, at the cost of an iterative solver that fails to converge for antipodal points. A third of a percent is already well inside a GPS fix.

<small>[core/location.go:544](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L544)</small>

### func FormatLatLng

```go
func FormatLatLng(lat, lng float64) string
```

FormatLatLng writes the wire form of a point, "lat,lng". See FormatRegion.

<small>[core/mapview.go:411](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L411)</small>

### func FormatRegion

```go
func FormatRegion(r Region) string
```

FormatRegion writes the wire form of a region, "lat,lng,zoom".

Nothing in Go sends one today — the hosts are the writers, in Swift, Kotlin and JavaScript — and this exists so that the format has a Go statement for the harnesses to check those three against, and so a Go-side host (a test, an embedder driving the tree directly) has the same spelling available rather than inventing one.

'f' with -1 precision: the shortest form that round-trips, so 38.7223 stays "38.7223" and a whole degree stays "38". Deliberately not 'g', which switches to exponent form for small numbers — "1e-05" is a valid float in Go and is not what a hand-written host parser expects to find in a comma-separated coordinate.

<small>[core/mapview.go:406](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L406)</small>

### func Haptic

```go
func Haptic(kind HapticKind)
```

Haptic asks the host to play one haptic effect.

A kind outside HapticKinds is dropped here rather than sent: every shell would have to ignore it separately, and an older shell receiving a newer kind already ignores it, so the check in Go only catches a typo'd HapticKind("sucess") at the one place that can log nothing useful either way — the call simply does nothing, same as on a device with no motor.

<small>[core/haptics.go:89](https://github.com/rohanthewiz/grmob/blob/master/core/haptics.go#L89)</small>

### func HeadingActive

```go
func HeadingActive() bool
```

HeadingActive reports whether the sensor is running. Mostly useful to tests and to a debug overlay; a screen wants Heading.Active, which travels with the reading it belongs to.

<small>[core/heading.go:265](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L265)</small>

### func LocationAcquiring

```go
func LocationAcquiring()
```

LocationAcquiring is the host saying the sensor has just started — or started again — and has nothing to report yet.

#### The window it exists for

A refused start stays armed on both natives, because the grant usually arrives after the screen that wants it: the tap that asks is on that screen. When it does arrive the host re-arms and the GPS begins working, but the last thing Go was told is still the refusal, and a cold first fix is tens of seconds away. A screen written to the documented shape — \`case !loc.Available: EmptyState{Hint: loc.Error}\` — therefore prints "location permission not granted" for the whole of that window, about a sensor that is running.

The refusal is the host's statement about a run, so only the host can withdraw it, which is why this is an event a host sends rather than something core could infer: nothing in Go knows that a re-arm happened.

#### Why it is not a field on Location

Because the state it produces is one the record could already express and never reached: \*\*Active and not Received\*\* is a sensor that is running and has said nothing, which is exactly "acquiring". What was missing was a way to get BACK to it after a refusal, not a way to describe it. So this resets the report — the coordinates, the accuracy, the altitude, Received, Error — leaving Available true, because a sensor that accepted the start is one that can try and nobody yet knows better.

The reset is what makes a screen that has never heard of this improve without being touched: with Error cleared and Available true, the arm that used to print the refusal no longer matches, and the arm that draws a spinner does.

Ignored when nothing is running. A host reporting a re-arm with no consumer is a host bug, and acting on it would mean a record moving for a sensor nobody asked for.

<small>[core/location.go:380](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L380)</small>

### func LocationActive

```go
func LocationActive() bool
```

LocationActive reports whether the sensor is running. Mostly useful to tests and to a debug overlay; a screen wants Location.Active, which travels with the fix it belongs to.

<small>[core/location.go:270](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L270)</small>

### func MapView

```go
func MapView(region Region, props ...PropsAndChildren) View
```

MapView is a live map: the platform's own map widget, panned and zoomed by the user, with markers as child nodes.

	core.MapView(core.Region{Lat: 38.7223, Lng: -9.1393, Zoom: 14},
	    core.Width("100%"), core.Height("280px"),
	    core.ShowUserLocation(),
	    core.OnMarkerTap(func(id string) { open(id) }),
	    core.Marker("hall", 38.7223, -9.1393, "The hall"),
	    core.Marker("annex", 38.7251, -9.1402, "The annex"),
	)

Each host draws its platform's map:

	iOS       MapKit, which is free and needs no key
	Android   osmdroid over OpenStreetMap tiles — no key, and no dependency on
	          Play Services being present on the device
	Browser   Leaflet over OpenStreetMap tiles, loaded by the host page
	htmlout   a placeholder box, as CameraView is: a static snapshot has no
	          engine to run and no tiles to fetch

#### When to use comps.StaticMap instead

Almost always, if the question is "where is this". A static map is an image and a hand-off to the platform's maps app: no engine, no tile budget, no key, nothing to keep in step, and the directions the user actually wanted come from the app that has their home address in it.

This node is for the cases that need the map to be \*part of\* the screen: a set of markers to compare, a region the user explores, a position they pick by tapping. Those are interactions, and an image cannot have them.

#### Markers are children, not a prop

The same decision core.TextGrid makes about its rows, for the same reason. A marker set sent as one prop means every marker is re-read whenever any of them moves: the reconciler sees one changed value and the host rebuilds the annotation layer, which on every platform is a visible flicker and on two of them loses the selected callout.

As children they are ordinary nodes. The reconciler pairs them by key, emits an update-props patch for the one marker that moved, an add for the one that appeared and a remove for the one that left — and each host's annotation bookkeeping is the patch handling it already has.

Marker keys itself from its id, so a caller writing core.Marker in a loop gets stable identity without having to remember core.Keyed. See Marker.

#### The map is controlled, with the echo guard a drag needs

Region is Go's statement of where the map should be, and it is applied to the host widget \*only when it changes\*. It is not re-asserted on every patch.

That sounds like a detail and it is the whole usability of the node. A map is the one widget whose value the user changes continuously by touching it: if every render re-centred the host map on Go's Region, then an app that does not echo OnRegionChange back into its own state would snap the map back under the user's finger on the next unrelated re-render — and an app that does echo it would fight its own round trip, because the echo arrives a frame late and moves the map again.

So each host remembers the Region it last applied and compares: Go moving the map is an instruction, and Go merely re-rendering is not. It is the same compromise the text fields and the Slider make — the value shown is Go's except where the finger is the authority — and it has to be implemented the same way in all three live hosts, which mobile/verify and wasm/verify check.

An app that wants the map pinned to its own state does nothing special: it echoes OnRegionChange into state, and every Region it renders is one it chose. An app that wants "show me this place, then let the user wander" renders a constant Region, which is applied once.

#### Two memories, and the one thing that cannot be said

Each host keeps the Region \*Go\* last asked for and, separately, where the \*map\* last came to rest. They are the same value until somebody touches the map, and each direction reads the one that answers its own question — Go changing its mind is an instruction; the map already being there is not.

Folding those into one slot is the bug this contract exists to prevent, and it shipped in all three hosts: a pan wrote the user's Region into the slot the apply path reads, so Go's \*unchanged\* Region read as a change and the next patch to reach the map — a pin dropped, a marker moved, any unrelated re-render — snapped the map back. It survived a unit test because a fake map can be panned without firing the event a real one always fires, and it was found by opening a browser.

The consequence a caller can see is this: re-rendering the \*same\* Region is never a re-centre. An app that pans away and then wants the opening view back cannot get it by handing the same numbers over again, because from here that is indistinguishable from the unrelated re-render above. The remedy is the one this node already recommends — echo OnRegionChange into state, so the Region an app renders tracks where the map is and a "back to the start" button is a genuine change. There is deliberately no imperative recentre command; adding one is a host feature in three languages, and the echo costs one line.

#### Tiles are somebody else's bandwidth

Two of the three hosts draw OpenStreetMap tiles from the project's own servers, which have a usage policy: identify your app, do not bulk download, and expect to be blocked if you send a million tile requests a day. An app shipping this to a real user base should point its host at a tile provider it pays for. That decision lives in each host rather than in this node — it is a URL template in osmdroid's configuration and in Leaflet's layer — because it is a deployment fact rather than a property of the view.

<small>[core/mapview.go:114](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L114)</small>

### func Marker

```go
func Marker(id string, lat, lng float64, title string, props ...PropsAndChildren) View
```

Marker is one pin on a MapView: a child node, keyed by its id.

	core.Marker("hall", 38.7223, -9.1393, "The hall")

#### It keys itself

The key is "marker:" + id, written here rather than left to the caller, which is a departure from how every other keyed child in this framework works — core.List's rows are the caller's to key, and forgetting is a documented mistake with a debug-mode concern behind it.

The difference is that a marker already carries its identity. The id is required, it is what OnMarkerTap reports, and there is no sensible second answer to "which marker is this" — so a caller writing core.Keyed around one would be restating the id, and a caller forgetting to would get the failure keys exist to prevent (a marker layer rebuilt on every change) in the one place it is most expensive.

An empty id is allowed and keys nothing, which is the honest answer for the single unnamed marker a "you are here" view draws: there is nothing to tell it apart from, and nothing will ever report a tap on it by name.

#### A marker is data, not a box

It carries no style and draws no element of its own. Every host reads its props and creates a native annotation; the node exists so the reconciler can address it. Both DOM renderers give it a hidden element, because a patch path is positional and a node with no element would put every later patch in the wrong place — the same reason htmlout and the WASM runtime disagree about Fragment (see htmlout's transparentTypes).

Title is the callout the platform shows when a marker is tapped, and may be empty. It is not an accessibility label: a map's annotations are announced by each platform's own map accessibility, which this framework does not reach into.

<small>[core/mapview.go:337](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L337)</small>

### func NormalizeDegrees

```go
func NormalizeDegrees(deg float64) float64
```

NormalizeDegrees folds any angle into \[0, 360). Negative angles and angles past a full turn both come back on the circle, so -90 is 270 and 730 is 10.

math.Mod alone is not enough: it keeps the sign of its first argument, so -90 comes back as -90 rather than 270.

<small>[core/heading.go:144](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L144)</small>

### func OnAudioStatus

```go
func OnAudioStatus(fn func(AudioStatus)) (cancel func())
```

OnAudioStatus subscribes fn to every status change. The returned function cancels the subscription. fn runs on whichever goroutine delivered the change — a host bridge call, or the app's own AudioLoad — and must not block; the usual body is a State write or a RequestRender.

Most screens want hooks.UseAudio instead, which subscribes once per component and re-renders on each change.

<small>[core/audio.go:293](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L293)</small>

### func OnHeading

```go
func OnHeading(fn func(Heading)) (cancel func())
```

OnHeading subscribes fn to heading changes. The returned function cancels the subscription.

Subscribing does not start the sensor: the two are separate on purpose, because a screen that wants to \*display\* a heading and a screen that wants the sensor \*on\* are not always the same screen (a background service, a second view of one reading). hooks.UseHeading does both, which is what most callers want.

fn runs on whichever goroutine delivered the reading — a host bridge call — and must not block; the usual body is a RequestRender.

<small>[core/heading.go:291](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L291)</small>

### func OnLifecycle

```go
func OnLifecycle(fn func(LifecycleState)) (cancel func())
```

OnLifecycle subscribes fn to lifecycle transitions. The returned function cancels the subscription; calling it more than once is harmless.

Like OnAudioStatus and OnHostEvent, the subscription is process-wide — there is one app and one screen, so there is no context tree to scope it to. A component that subscribes during render must guard against subscribing again on the next pass; hooks.UseLifecycle does that.

fn runs on whichever goroutine delivered the event (see the threading note in host\_events.go) and must not block. Writing State and calling RequestRender are fine from there.

<small>[core/lifecycle.go:92](https://github.com/rohanthewiz/grmob/blob/master/core/lifecycle.go#L92)</small>

### func OnLocation

```go
func OnLocation(fn func(Location)) (cancel func())
```

OnLocation subscribes fn to location changes. The returned function cancels the subscription.

Subscribing does not start the sensor, for the reason OnHeading does not: wanting to \*see\* a position and wanting the GPS \*on\* are not always the same screen. hooks.UseLocation does both.

fn runs on whichever goroutine delivered the fix — a host bridge call — and must not block; the usual body is a RequestRender.

<small>[core/location.go:293](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L293)</small>

### func OnMapTap

```go
func OnMapTap(fn func(lat, lng float64)) BehaviorProp
```

OnMapTap reports where the user tapped on the map, in degrees — the prop a "choose a place" screen is built on.

It does not fire for a tap that hit a marker: that is OnMarkerTap's event, and a host that sent both would make every marker tap also drop a pin. Each host suppresses it the way its own map does — an annotation's hit test runs first on all three.

Like OnRegionChange it crosses as text, "lat,lng", and an unparseable payload is dropped rather than delivered as 0,0.

<small>[core/mapview.go:286](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L286)</small>

### func OnMarkerTap

```go
func OnMarkerTap(fn func(id string)) BehaviorProp
```

OnMarkerTap reports the id of the marker the user tapped — the id given to core.Marker, unchanged.

An id rather than the marker's coordinates, because the id is what the app has an index of. Two markers can share a position (a building with two tenants, a rounded coordinate) and no app wants to identify a row by comparing floats.

<small>[core/mapview.go:264](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L264)</small>

### func OnNotificationTap

```go
func OnNotificationTap(fn func(id string)) (cancel func())
```

OnNotificationTap subscribes fn to taps on the app's notifications; fn receives the ID the tapped notification was posted under. The returned function cancels the subscription.

Like OnDeepLink, a typed wrapper over OnHostEvent and nothing more: core keeps no record of taps, because a tap is an instruction ("show me this") rather than a state anyone reads later. fn runs on the goroutine that delivered the host event and must not block. An empty or absent id is dropped — a subscriber cannot route a tap it cannot identify.

<small>[core/notifications.go:181](https://github.com/rohanthewiz/grmob/blob/master/core/notifications.go#L181)</small>

### func OnRegionChange

```go
func OnRegionChange(fn func(Region)) BehaviorProp
```

OnRegionChange reports where the user moved the map to, after they stop moving it.

"After" is the contract and the hosts enforce it, because a pan is a stream: a finger dragging across a map generates a region per frame, and each one that crossed the bridge would be a full Go render pass. Every host therefore throttles — it reports on the gesture's end, and at a bounded rate during a sustained one — which is the same arrangement the sensors have, for the same reason, and is checked in the same places.

The region arrives through the text callback channel as "lat,lng,zoom", which is how Slider's float crosses and why no new bridge channel was added for this node. A payload that does not parse is dropped rather than delivered as zeros: 0,0 is a real place, and an app that centred on it because of a formatting bug in one host would be looking at the Gulf of Guinea with no error anywhere.

<small>[core/mapview.go:241](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L241)</small>

### func ParseLatLng

```go
func ParseLatLng(s string) (lat, lng float64, ok bool)
```

ParseLatLng reads a host's "lat,lng" payload, for OnMapTap. Same contract as ParseRegion: false rather than zeros.

<small>[core/mapview.go:380](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L380)</small>

### func PostNotification

```go
func PostNotification(n LocalNotification)
```

PostNotification asks the host to show n — now, or at n.At — replacing any notification already showing or scheduled under the same ID. Dropped without an ID or without any text; see the file comment for why the ID is required and for permissions.

<small>[core/notifications.go:135](https://github.com/rohanthewiz/grmob/blob/master/core/notifications.go#L135)</small>

### func ReadClipboard

```go
func ReadClipboard(fn func(text string, ok bool))
```

ReadClipboard asks the host for the clipboard's text and calls fn with the answer, exactly once. See the file comment for what ok means.

With no host registered (a headless test, htmlout) fn runs immediately with ("", false) on the caller's goroutine: there is no platform to ask, and a caller waiting on a reply that can never come would be a hang, not a degradation.

fn is registered before the event is sent, because a native host may answer synchronously inside SendSystemEvent's call and the reply must find its callback already waiting.

<small>[core/clipboard.go:97](https://github.com/rohanthewiz/grmob/blob/master/core/clipboard.go#L97)</small>

### func ReceiveAudioStatus

```go
func ReceiveAudioStatus(s AudioStatus)
```

ReceiveAudioStatus is the typed entry point for a host that builds the status in Go (a test, an embedder). The JSON hosts arrive through ReceiveHostEvent("audio\_status", ...) instead, which decodes into this.

Track metadata is not something a host reports — it only ever echoes the URL — so the incoming Track's URL is matched against the loaded track: the same URL keeps the title, artist and artwork the app supplied; a different one (a host playing something the app did not load, which should not happen but must not corrupt the record) keeps only the URL.

<small>[core/audio.go:315](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L315)</small>

### func ReceiveHeading

```go
func ReceiveHeading(h Heading)
```

ReceiveHeading is the typed entry point for a host that builds the reading in Go (a test, an embedder). The JSON hosts arrive through ReceiveHostEvent("heading", ...) instead, which decodes into this.

Bearings are normalised here rather than trusted: a host that reports 360.0 at north, or a negative azimuth (which Android's getOrientation returns for half the circle — it answers in radians over -pi..pi), would otherwise leak an out-of-range angle into every consumer's arithmetic. This is the one place all four hosts funnel through, so it is the one place the invariant "Magnetic is in \[0, 360)" can actually be established.

Active is core's bookkeeping and is overwritten from the reference count, not taken from the caller: a host does not know how many screens asked.

<small>[core/heading.go:317](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L317)</small>

### func ReceiveLifecycle

```go
func ReceiveLifecycle(s LifecycleState)
```

ReceiveLifecycle is the typed entry point for a host that reports in Go (a test, an embedder). The JSON hosts arrive through ReceiveHostEvent("lifecycle", ...), which decodes into this.

A state that is not one of the three is dropped rather than stored: a newer shell reporting a fourth state to an older app must not leave CurrentLifecycle answering something no switch in that app has an arm for. A repeat of the current state is absorbed silently (see the file comment). Subscribers are notified outside the lock, so one may read CurrentLifecycle, subscribe or cancel from inside its handler.

<small>[core/lifecycle.go:115](https://github.com/rohanthewiz/grmob/blob/master/core/lifecycle.go#L115)</small>

### func ReceiveLocation

```go
func ReceiveLocation(l Location)
```

ReceiveLocation is the typed entry point for a host that builds the fix in Go (a test, an embedder). The JSON hosts arrive through ReceiveHostEvent("location", ...), which decodes into this.

Coordinates are normalised here rather than trusted, which is this function's reason for existing beyond plumbing: it is the one place all four hosts funnel through, so it is the only place the invariant can be established. Latitude is clamped to ±90 and longitude wrapped into (-180, 180] — two rules, because they are two different facts about the sphere. A latitude past the pole is not a place; a longitude past the antimeridian is the same meridian spelled the long way round, and clamping it would move the device to the far side of the Pacific.

Active is core's bookkeeping and is overwritten from the reference count rather than taken from the caller: a host does not know how many screens asked.

<small>[core/location.go:322](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L322)</small>

### func ShowUserLocation

```go
func ShowUserLocation() BehaviorProp
```

ShowUserLocation asks the host's map to draw the user's position — the blue dot, with the platform's own styling and its own accuracy halo.

It is the host's location, not core.Location's. Every map SDK has this built in, reads the OS permission itself, and tells Go nothing about where anybody is. A screen that needs the coordinates wants hooks.UseLocation, which is a separate feature that happens to need the same permission — see core.Location's "What this is not".

The dot needs that permission. On a platform where it has not been granted, every one of these hosts draws the map and no dot, with nothing reported: a map is still a map. An app that wants to explain the absence has to check the permission itself, which is what permission.Location is for.

<small>[core/mapview.go:216](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L216)</small>

### func StartHeading

```go
func StartHeading()
```

StartHeading asks the host to begin reporting compass headings, and is balanced by StopHeading — see "Reference counting" in the package comment.

On a browser this is also the permission moment: Safari's DeviceOrientationEvent.requestPermission must be called from inside a user gesture, and the host runtime makes that call here rather than exposing a separate permission API. A start that happens outside a gesture is refused by the browser and comes back as an "available: false" event carrying the reason, which is why Heading.Error exists.

<small>[core/heading.go:217](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L217)</small>

### func StartLocation

```go
func StartLocation()
```

StartLocation asks the host to begin reporting position fixes, and is balanced by StopLocation — the reference counting heading.go describes, and for the same reason twice over: GPS is the most expensive sensor on the device, and two screens that each want the user's position must each be able to let go without blinding the other.

A host that needs an OS permission asks for one here if it has to, and a refusal arrives as an \`available: false\` event with a reason. An app that wants to control when that dialog appears asks first — see the type comment.

<small>[core/location.go:213](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L213)</small>

### func StopHeading

```go
func StopHeading()
```

StopHeading releases one Start. The sensor is turned off when the last holder lets go; extra Stops are ignored rather than driving the count negative.

<small>[core/heading.go:239](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L239)</small>

### func StopLocation

```go
func StopLocation()
```

StopLocation releases one Start. The sensor is turned off when the last holder lets go; extra Stops are ignored rather than driving the count negative.

<small>[core/location.go:244](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L244)</small>

### func SweepNotifications

```go
func SweepNotifications(prefix string, fn func(fired []string))
```

SweepNotifications cancels every notification still scheduled under an ID beginning with prefix, and calls fn once with the IDs among the scheduled ones whose time had arrived (see "Sweeping by prefix"). Banners already shown stay. fn may be nil.

An empty prefix is refused (fn runs with no ids and nothing is sent): sweeping everything would take down notifications this caller never posted. With no host registered fn runs at once on the caller's goroutine, as ReadClipboard's does; otherwise on the goroutine that delivers the reply.

<small>[core/notifications.go:246](https://github.com/rohanthewiz/grmob/blob/master/core/notifications.go#L246)</small>

### func WrapLongitude

```go
func WrapLongitude(lng float64) float64
```

WrapLongitude folds a longitude into (-180, 180] by going round rather than by stopping at the edge: 190° east is 170° west, the same meridian, and a clamp would move the point to the antimeridian instead.

Exported because every consumer of a coordinate needs it and getting it wrong is silent — a map centred 20 degrees from where it was asked to be still looks like a map. comps.StaticMap does the same arithmetic for the same reason.

math.Mod keeps the sign of its first argument, so a negative input stays west, and the two adjustments are what carry a value past ±180 round to the other side.

<small>[core/location.go:508](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L508)</small>

### func WriteClipboard

```go
func WriteClipboard(text string)
```

WriteClipboard puts text on the system clipboard. Fire-and-forget, like OpenURL: callable from any goroutine, silent when no host is registered.

An empty string is sent rather than dropped (unlike OpenURL's empty url): clearing the clipboard — after copying a one-time code, say — is a legitimate thing to ask for.

<small>[core/clipboard.go:79](https://github.com/rohanthewiz/grmob/blob/master/core/clipboard.go#L79)</small>

## Types

### type AudioOpt

```go
type AudioOpt interface {
	Apply(*audioLoadConfig)
}
```

AudioOpt configures AudioLoad.

<small>[core/audio.go:130](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L130)</small>

#### func AudioAutoplay

```go
func AudioAutoplay(on bool) AudioOpt
```

AudioAutoplay controls whether AudioLoad starts playing as soon as the host can. The default is true: a tap on "play" that only buffered would need a second tap, and the browser only allows autoplay from inside a user gesture anyway — which a tap handler is.

<small>[core/audio.go:148](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L148)</small>

#### func AudioStartAt

```go
func AudioStartAt(seconds float64) AudioOpt
```

AudioStartAt begins playback at the given second instead of at 0 — the "resume where you left off" option. The host clamps it to the track.

<small>[core/audio.go:154](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L154)</small>

#### func AudioWithRate

```go
func AudioWithRate(rate float64) AudioOpt
```

AudioWithRate sets the initial playback speed; 1 is normal. Non-positive values are ignored.

<small>[core/audio.go:164](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L164)</small>

### type AudioState

```go
type AudioState string
```

AudioState is the player's phase, as the host last reported it (or as core set optimistically; see the package comment).

<small>[core/audio.go:55](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L55)</small>

```go
const (
	AudioIdle    AudioState = "idle"    // nothing loaded
	AudioLoading AudioState = "loading" // buffering, or waiting for the host to answer a Load
	AudioPlaying AudioState = "playing"
	AudioPaused  AudioState = "paused"
	AudioEnded   AudioState = "ended" // played to the end; Play or Seek starts it again
	AudioError   AudioState = "error" // see AudioStatus.Error
)
```

### type AudioStatus

```go
type AudioStatus struct {
	Track    AudioTrack
	State    AudioState
	Position float64
	Duration float64
	Rate     float64
	Error    string // set when State is AudioError
}
```

AudioStatus is everything the app can know about playback. Position and Duration are seconds; Duration is 0 until the host has learned it (a streamed file reports it once the headers arrive). Rate is the playback speed, 1 being normal.

<small>[core/audio.go:82](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L82)</small>

#### func CurrentAudioStatus

```go
func CurrentAudioStatus() AudioStatus
```

CurrentAudioStatus returns the last known status. Safe from any goroutine.

<small>[core/audio.go:280](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L280)</small>

#### func (AudioStatus) Loaded

```go
func (s AudioStatus) Loaded() bool
```

Loaded reports whether a track is loaded, in any state but idle.

<small>[core/audio.go:92](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L92)</small>

#### func (AudioStatus) Progress

```go
func (s AudioStatus) Progress() float64
```

Progress is Position as a fraction of Duration, 0 while Duration is unknown — what a seek slider wants.

<small>[core/audio.go:96](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L96)</small>

### type AudioTrack

```go
type AudioTrack struct {
	URL        string
	Title      string
	Artist     string // the speaker, the band, the podcast host
	Album      string // the series, the show
	ArtworkURL string
}
```

AudioTrack names what to play and how the platform should describe it on the lock screen. Only URL is required; the rest is metadata the media session shows (and, on a phone, is the difference between a notification reading "Unknown" and one reading the sermon's title).

<small>[core/audio.go:70](https://github.com/rohanthewiz/grmob/blob/master/core/audio.go#L70)</small>

### type CameraNode

```go
type CameraNode struct {
	OnCapture func(string)
	OnError   func(string)
	Active    bool
	Flash     bool
	Facing    string
	Overlay   View
	Style     Style
}
```

<small>[core/camera.go:7](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L7)</small>

### type CameraProp

```go
type CameraProp interface {
	Apply(*CameraNode)
}
```

<small>[core/camera.go:3](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L3)</small>

#### func OnCapture

```go
func OnCapture(fn func(string)) CameraProp
```

<small>[core/camera.go:78](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L78)</small>

#### func OnError

```go
func OnError(fn func(string)) CameraProp
```

<small>[core/camera.go:72](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L72)</small>

#### func SetFacing

```go
func SetFacing(facing string) CameraProp
```

<small>[core/camera.go:66](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L66)</small>

#### func WithFlash

```go
func WithFlash(enabled bool) CameraProp
```

<small>[core/camera.go:60](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L60)</small>

#### func WithOverlay

```go
func WithOverlay(view View) CameraProp
```

<small>[core/camera.go:84](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L84)</small>

#### func WithStyle

```go
func WithStyle(style Style) CameraProp
```

<small>[core/camera.go:90](https://github.com/rohanthewiz/grmob/blob/master/core/camera.go#L90)</small>

### type HapticKind

```go
type HapticKind string
```

HapticKind names one haptic effect. See the table above for what each one maps to on every host.

<small>[core/haptics.go:52](https://github.com/rohanthewiz/grmob/blob/master/core/haptics.go#L52)</small>

```go
const (
	HapticSelection HapticKind = "selection"
	HapticLight     HapticKind = "light"
	HapticMedium    HapticKind = "medium"
	HapticHeavy     HapticKind = "heavy"
	HapticSuccess   HapticKind = "success"
	HapticWarning   HapticKind = "warning"
	HapticError     HapticKind = "error"
)
```

#### func HapticKinds

```go
func HapticKinds() []HapticKind
```

HapticKinds lists every kind, in the order the table above documents them.

It exists for the shell coverage test in mobile/verify rather than for apps: iterating core's own list is what makes adding an eighth kind fail that test until all three shells spell it, instead of the new kind silently doing nothing on whichever shell was forgotten.

<small>[core/haptics.go:75](https://github.com/rohanthewiz/grmob/blob/master/core/haptics.go#L75)</small>

### type Heading

```go
type Heading struct {
	// Magnetic is the bearing relative to magnetic north, in [0, 360).
	Magnetic float64

	// True is the bearing relative to *geographic* north, in [0, 360). The two
	// differ by the local magnetic declination, which is a fraction of a degree
	// in some places and more than 15 degrees in others, so a map application
	// wants this one and a "which way am I facing" readout does not care.
	//
	// It requires the host to know where it is: iOS reports trueHeading only
	// with location authorization, and neither Android's rotation vector nor
	// the browser's orientation events carry it at all. HasTrue says whether
	// the number is real; True is 0 when it is not, and 0 is also a perfectly
	// good northward bearing, which is why the bool exists rather than a
	// sentinel.
	True    float64
	HasTrue bool

	// Accuracy is the reading's error margin in degrees, or -1 when the host
	// does not say. Android reports a bucketed sensor accuracy, iOS a
	// headingAccuracy in degrees, and the browser nothing at all; a large value
	// is the cue to show the platform's figure-eight calibration prompt.
	Accuracy float64

	// Available reports whether this device can produce headings. False before
	// the first event and false forever on a desktop browser; see Received.
	Available bool

	// Received is true once any heading event has arrived, which is what
	// separates "this device has no compass" from "the first reading has not
	// landed yet". A spinner is right for the second and wrong for the first.
	Received bool

	// Active is true while the sensor is running — that is, while the
	// reference count is above zero. It is core's own bookkeeping rather than
	// the host's word, so it flips on the Start call rather than a round trip
	// later.
	Active bool

	// Error is the host's message when it could not start the sensor: a
	// browser motion permission refused, a magnetometer that failed to open.
	// Set alongside Available: false.
	Error string
}
```

Heading is one compass reading. Degrees increase clockwise, so 0 is north, 90 is east, 180 south, 270 west — the convention all three platform APIs and every paper compass share.

<small>[core/heading.go:66](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L66)</small>

#### func CurrentHeading

```go
func CurrentHeading() Heading
```

CurrentHeading returns the last reading, exactly as it arrived — the notification filter described on headingNotifyEpsilon does not apply here. Safe from any goroutine.

<small>[core/heading.go:274](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L274)</small>

#### func (Heading) Cardinal

```go
func (h Heading) Cardinal() string
```

Cardinal returns the 16-point compass abbreviation for the magnetic bearing — "N", "NNE", "NE", ... — which is what a compact readout shows beside (or instead of) the number.

Sixteen points rather than eight or thirty-two: eight is coarse enough that a bearing can sit 22 degrees from the label naming it, and thirty-two needs four-letter names ("NbE") that no reader outside sailing recognises.

<small>[core/heading.go:118](https://github.com/rohanthewiz/grmob/blob/master/core/heading.go#L118)</small>

### type LifecycleState

```go
type LifecycleState string
```

LifecycleState is where the app sits in the platform's foreground / background lifecycle. See the package comment above for the three values.

<small>[core/lifecycle.go:53](https://github.com/rohanthewiz/grmob/blob/master/core/lifecycle.go#L53)</small>

```go
const (
	LifecycleActive     LifecycleState = "active"
	LifecycleInactive   LifecycleState = "inactive"
	LifecycleBackground LifecycleState = "background"
)
```

#### func CurrentLifecycle

```go
func CurrentLifecycle() LifecycleState
```

CurrentLifecycle reports the last state the host announced; active until it has announced anything.

<small>[core/lifecycle.go:75](https://github.com/rohanthewiz/grmob/blob/master/core/lifecycle.go#L75)</small>

### type LocalNotification

```go
type LocalNotification struct {
	// ID identifies the notification for replacement, cancellation and taps.
	// Posting a second notification with the same ID replaces the first,
	// scheduled or shown.
	ID    string
	Title string
	Body  string

	// At schedules the notification for a moment instead of posting it now.
	// The zero value, or any time not after the moment of posting, posts
	// immediately. See "Scheduled notifications" above for what each host
	// promises.
	At time.Time
}
```

LocalNotification is one banner to post. ID is required; Title and Body may each be empty but not both.

<small>[core/notifications.go:103](https://github.com/rohanthewiz/grmob/blob/master/core/notifications.go#L103)</small>

### type Location

```go
type Location struct {
	// Lat and Lng are degrees, WGS-84 — the datum every platform API and every
	// tile provider here uses, so no conversion happens anywhere in this
	// framework.
	Lat, Lng float64

	// Accuracy is the horizontal radius of the fix in metres: the device
	// believes it is somewhere inside this circle. -1 when the host does not
	// say, which in practice no host does.
	//
	// It is not an error bar to be ignored. 5 metres is a GPS fix outdoors,
	// 50 is a fix through a roof, 2000 is a guess from the cell tower or the
	// IP address, and the last one is what a desktop browser reports while
	// looking exactly like the first to any code that reads only Lat and Lng.
	Accuracy float64

	// Altitude is metres above the WGS-84 ellipsoid, and HasAltitude says
	// whether the number is real. 0 is a perfectly good altitude — it is most
	// of the world's coastline — which is why the bool exists rather than a
	// sentinel, exactly as Heading.HasTrue does one file over.
	//
	// Vertical accuracy is deliberately not carried. It is reported by two of
	// the three hosts, it is a different and much larger number than the
	// horizontal one, and nothing has asked for it; a field no screen reads is
	// three hosts remembering to fill it in for nothing.
	Altitude    float64
	HasAltitude bool

	// Available reports whether this device can produce a fix. False before
	// the first event, and false after a refusal or a hardware failure; see
	// Received for the difference between "no" and "not yet".
	//
	// It is the best answer anyone has rather than a guarantee: while the
	// sensor is running and has not reported yet — Active with Received false,
	// which is what LocationAcquiring puts the record back into — it is true,
	// because a sensor that accepted the start is one that can try.
	Available bool

	// Received is true once any location event has arrived. A first fix can
	// take tens of seconds on cold GPS, so this is the flag that separates
	// "still acquiring" — which is a spinner, and a long one — from "this
	// device will never tell you", which is a different screen.
	//
	// Reset to false when a run begins, which is the half of it that took a
	// second emulator run to find: a refused start that is later granted
	// re-arms on both natives, and without the reset the record still carried
	// the refusal all the way to the first fix. Active && !Received is the
	// acquiring state, and LocationAcquiring is how a host gets back to it.
	Received bool

	// Active is true while the sensor is running, which is core's own
	// reference count rather than the host's word: it flips on the Start call
	// rather than a round trip later.
	Active bool

	// Error is the host's message when it could not start or keep the sensor:
	// a permission refused, location services switched off system-wide, a
	// browser with no geolocation. Set alongside Available: false, and cleared
	// whenever a run begins — the reason a previous attempt failed is not a
	// statement about the one now running.
	Error string
}
```

Location is one position fix.

<small>[core/location.go:88](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L88)</small>

#### func CurrentLocation

```go
func CurrentLocation() Location
```

CurrentLocation returns the last fix, exactly as it arrived — the notification filter does not apply here. Safe from any goroutine.

<small>[core/location.go:278](https://github.com/rohanthewiz/grmob/blob/master/core/location.go#L278)</small>

### type Region

```go
type Region struct {
	// Lat and Lng are the centre of the view, in degrees, WGS-84.
	Lat, Lng float64

	// Zoom is the slippy-tile zoom level every one of these engines speaks: 0
	// is the whole world, 19 is a building. Fractional values are allowed and
	// meaningful — a pinch lands between levels, and the hosts report what the
	// user actually reached rather than rounding it.
	//
	// A zero Zoom means DefaultMapZoom rather than "the whole world", which is
	// the one legitimate value this type spends on a default. It is the same
	// trade comps.StaticMap.Zoom makes and for the same reason: a map with
	// no zoom stated is a map somebody forgot to scale, and the world is never
	// what they meant.
	Zoom float64
}
```

Region is a place and a scale: where a map is looking and how closely.

One struct rather than three floats at every call site, and the same struct in both directions — MapView takes one and OnRegionChange hands one back, so an app that echoes the user's pan into its own state is storing the type it renders from. Two types here (a "MapRegion" in and a "RegionChange" out) would differ in nothing and convert at every seam.

<small>[core/mapview.go:157](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L157)</small>

#### func ParseRegion

```go
func ParseRegion(s string) (Region, bool)
```

ParseRegion reads a host's "lat,lng,zoom" payload. The bool is false for anything that does not parse, and a caller must not substitute zeros: see OnRegionChange.

Exported because all three hosts format this string and a test in each harness has to read one back. Keeping the parse in one place is also what makes the wire format a single fact rather than three.

<small>[core/mapview.go:359](https://github.com/rohanthewiz/grmob/blob/master/core/mapview.go#L359)</small>

