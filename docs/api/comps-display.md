# Package comps — Data display & maps

```go
import "github.com/rohanthewiz/grmob/comps"
```

Avatars and avatar stacks, stat tiles, the compass, clocks, countdowns and alarms, an audio player, message bubbles, expandable text, QR codes, map panels and static maps.

One of 7 topic pages of [package comps](comps.md), which has the package overview and an index of every topic. This page documents the declarations in `comps/avatar.go`, `comps/avatar_stack.go`, `comps/stat_tile.go`, `comps/compass.go`, `comps/clock.go`, `comps/timers.go`, `comps/alarm.go`, `comps/audio_player.go`, `comps/message_bubble.go`, `comps/expandable_text.go`, `comps/qr_code.go`, `comps/map_panel.go`, `comps/static_map.go`.

## Index

- [Constants](#constants) — `ConcernAudioPlayerNoTrack`, `ConcernCountdownUntilUnset`, `ConcernNoMapProvider`, `ConcernQRDataTooLong`, `ConcernStopwatchSinceUnset`, `DefaultMapHeight`, `DefaultMapPanelHeight`, `DefaultMapScale`, `DefaultMapWidth`, `DefaultMapZoom`, `FitPadding`, `MaxFitZoom`, and 6 more
- [`func FitRegion`](#func-fitregion)
- [`func GoogleMapsHandoff`](#func-googlemapshandoff)
- [`func OSMStaticMap`](#func-osmstaticmap)
- [`func OpenStreetMapHandoff`](#func-openstreetmaphandoff)
- [`func PlaceCount`](#func-placecount)
- [`type AlarmRinging`](#type-alarmringing)
    - [`func (AlarmRinging) Render`](#func-alarmringing-render)
- [`type AlarmRow`](#type-alarmrow)
    - [`func (AlarmRow) Render`](#func-alarmrow-render)
- [`type AnalogClock`](#type-analogclock)
    - [`func (AnalogClock) Render`](#func-analogclock-render)
- [`type AudioPlayer`](#type-audioplayer)
    - [`func (AudioPlayer) Render`](#func-audioplayer-render)
- [`type Avatar`](#type-avatar)
    - [`func (Avatar) Render`](#func-avatar-render)
- [`type AvatarStack`](#type-avatarstack)
    - [`func (AvatarStack) Render`](#func-avatarstack-render)
- [`type Compass`](#type-compass)
    - [`func (Compass) Render`](#func-compass-render)
- [`type Countdown`](#type-countdown)
    - [`func (Countdown) Render`](#func-countdown-render)
- [`type DigitalClock`](#type-digitalclock)
    - [`func (DigitalClock) Render`](#func-digitalclock-render)
- [`type ECLevel`](#type-eclevel)
- [`type ExpandableText`](#type-expandabletext)
    - [`func (ExpandableText) Render`](#func-expandabletext-render)
- [`type MapHandoff`](#type-maphandoff)
- [`type MapPanel`](#type-mappanel)
    - [`func (MapPanel) Render`](#func-mappanel-render)
- [`type MapPin`](#type-mappin)
- [`type MessageBubble`](#type-messagebubble)
    - [`func (MessageBubble) Render`](#func-messagebubble-render)
- [`type QRCode`](#type-qrcode)
    - [`func (QRCode) Render`](#func-qrcode-render)
- [`type StatTile`](#type-stattile)
    - [`func (StatTile) Render`](#func-stattile-render)
- [`type StaticMap`](#type-staticmap)
    - [`func (StaticMap) Area`](#func-staticmap-area)
    - [`func (StaticMap) Render`](#func-staticmap-render)
- [`type StaticMapArea`](#type-staticmaparea)
- [`type StaticMapProvider`](#type-staticmapprovider)
    - [`func GoogleStaticMap`](#func-googlestaticmap)
- [`type Stopwatch`](#type-stopwatch)
    - [`func (Stopwatch) Render`](#func-stopwatch-render)

## Constants

```go
const (
	// DefaultMapPanelHeight is a map tall enough to have a shape and short
	// enough to leave a caption and a row or two of context on a phone screen.
	DefaultMapPanelHeight = "280px"

	// MinFitSpread is the tightest view FitRegion will open at, in degrees of
	// latitude — about 550 metres, which is a few blocks.
	//
	// It exists because a single pin has a spread of zero and the logarithm of
	// a division by zero is not a zoom level. A floor rather than a branch:
	// one rule, no arm that runs only for one point, and the floor is the view
	// a single pin wants anyway.
	MinFitSpread = 0.005

	// FitPadding widens the fitted span so the outermost pins are not against
	// the edge of the frame. A fifth, which is about one pin's height at the
	// sizes a panel is drawn at.
	FitPadding = 1.2

	// MinFitZoom and MaxFitZoom bound the answer. 19 is as deep as the
	// standard tile pyramid goes; 1 is a view half the globe wide, which is
	// the widest this returns.
	//
	// Not 0, even though 0 is the world and would fit anything: core.Region
	// reads a zero Zoom as "unstated" and substitutes core.DefaultMapZoom, so
	// returning 0 here would hand back a neighbourhood view of a set spanning
	// continents — the exact opposite of what was asked for. See
	// core.Region.Zoom, which explains why zero cannot mean the world.
	MinFitZoom = 1.0
	MaxFitZoom = 19.0
)
```

<small>[comps/map_panel.go:116](https://github.com/rohanthewiz/grmob/blob/master/comps/map_panel.go#L116)</small>

The defaults and the one clamp, named so a caller can reason about them without reading the render function.

```go
const (
	// DefaultMapZoom is street level: a block or two across the image, which
	// is the scale at which "where is this" is legible. 15 on the slippy
	// scale, the same number both providers mean by it.
	//
	// One level deeper than core.MapView's default, and the gap is the
	// difference between the two widgets: a static map answers "where is this
	// one place", a live map is usually showing a set, and one level out is
	// about four times the area.
	DefaultMapZoom = 15

	// DefaultMapWidth and DefaultMapHeight are a 16:9 panel the width of a
	// phone card. Landscape because a map of one point has nothing to say
	// vertically that it does not say horizontally, and a tall map wastes the
	// screen a caption wants.
	DefaultMapWidth  = 320
	DefaultMapHeight = 180

	// MaxMapDimension is the smaller of the two bundled providers' limits.
	// Google's free static API refuses above 640 per side; the OSM service's
	// ceiling is the same order. See StaticMap.Width for why this clamps
	// rather than passing the request through to be refused.
	MaxMapDimension = 640

	// DefaultMapScale is one image pixel per logical pixel, which is what a
	// caller who says nothing about the device gets — and what every caller
	// got before Scale existed. See "Device pixel ratio" on StaticMap for why
	// it is not read off the screen.
	DefaultMapScale = 1

	// MaxMapScale is 3, the deepest ratio shipping phones use. It clamps the
	// *request*, and is not a promise about the answer: Google's API tops out
	// at MaxGoogleMapScale and a provider with no scale of its own serves 1x
	// whatever it is handed.
	MaxMapScale = 3

	// MaxGoogleMapScale is what the Maps Static API accepts — 1 or 2, and
	// nothing else. A 3x device therefore gets the 2x image, which is the
	// sharpest thing that API will serve and still four times the pixels of
	// the 1x it used to get.
	//
	// Google applies its size limit *before* scaling, which is why Width and
	// Height are not also divided down to make room: a scale=2 request for a
	// 640px box is accepted and served as 1280px.
	MaxGoogleMapScale = 2

	// MercatorLatLimit is where Web Mercator stops. Every tile provider here
	// projects with it, so this is the edge of the addressable world rather
	// than a choice — see StaticMap.Lat.
	MercatorLatLimit = 85.0511
)
```

<small>[comps/static_map.go:235](https://github.com/rohanthewiz/grmob/blob/master/comps/static_map.go#L235)</small>

ConcernAudioPlayerNoTrack is raised, in debug builds only, when an AudioPlayer has no Track.URL. Play would load nothing, so the widget disables it — and a player whose only button is dimmed looks exactly like one waiting for its stream to buffer.

```go
const ConcernAudioPlayerNoTrack = "audio-player-no-track"
```

<small>[comps/audio_player.go:15](https://github.com/rohanthewiz/grmob/blob/master/comps/audio_player.go#L15)</small>

ConcernCountdownUntilUnset is raised, in debug builds only, when Until is the zero time.Time. The countdown is then permanently expired: it draws 0:00 and fires OnDone on its first pass, which on screen is exactly what a timer that has just finished looks like. So a Countdown rendered before its deadline was assigned — a struct built from a half-filled record, a field spelled differently in the caller — would otherwise announce itself as a completed timer and nobody would go looking.

```go
const ConcernCountdownUntilUnset = "countdown-until-unset"
```

<small>[comps/timers.go:20](https://github.com/rohanthewiz/grmob/blob/master/comps/timers.go#L20)</small>

ConcernNoMapProvider: a StaticMap rendered with no Provider, which draws an empty frame. It is a development-time finding rather than a panic because the failure is survivable — a screen missing its map is still a screen — and because the fix is configuration, which is exactly the class of mistake that is invisible in a running app and obvious in a concern list.

```go
const ConcernNoMapProvider = "no-map-provider"
```

<small>[comps/static_map.go:318](https://github.com/rohanthewiz/grmob/blob/master/comps/static_map.go#L318)</small>

ConcernQRDataTooLong is raised, in debug builds only, when Data is longer than any QR Code can hold at the requested level. The widget then draws nothing: there is no half of a QR Code that is worth showing, and a symbol that encodes a truncated URL is worse than a blank space because it scans.

```go
const ConcernQRDataTooLong = "qr-data-too-long"
```

<small>[comps/qr_code.go:54](https://github.com/rohanthewiz/grmob/blob/master/comps/qr_code.go#L54)</small>

ConcernStopwatchSinceUnset is raised, in debug builds only, when Running is true and Since is the zero time.Time. The elapsed time is then measured from year 1, which reads as a seventeen-million-hour stopwatch — visibly wrong, but only if somebody is looking at the digits rather than at a screenshot, and silently wrong in the accessible label.

```go
const ConcernStopwatchSinceUnset = "stopwatch-since-unset"
```

<small>[comps/timers.go:27](https://github.com/rohanthewiz/grmob/blob/master/comps/timers.go#L27)</small>

## Functions

### func FitRegion

```go
func FitRegion(pins []MapPin) (core.Region, bool)
```

FitRegion is the view that contains every pin: a centre and the zoom level at which the whole set is on screen.

Exported separately from the widget because it is the reusable half. A caller driving core.MapView directly — holding the region in state, echoing OnRegionChange — still needs this answer for its opening view, and the alternative is every such caller deriving the same logarithm.

	zoom ≈ log2(360 / spread in degrees)

#### The projection correction

The longitude spread is scaled by the cosine of the centre latitude before the two spans are compared. A degree of longitude narrows towards the poles and a degree of latitude does not, so a fit computed from raw degrees is too tight in Reykjavík and about right in Quito. Whichever span needs the wider view decides.

Very near a pole the cosine approaches zero and the correction stops being meaningful, so it is floored — a map that far north is showing one place anyway.

#### Two sets it does not fit, and says so here rather than pretending

A set spanning more than half the globe is clamped to MinFitZoom rather than contained: that is a view 180 degrees wide, and the alternative would be a zoom core.Region cannot express (see MinFitZoom). Pins outside it are off screen, and a map of the whole planet would not have shown them usefully anyway.

A set straddling the antimeridian used to be measured the long way round: Tokyo and Honolulu are 62 degrees apart going east and were read as 298 going the other way, so the fit opened on the Atlantic with both pins off the edges. That is fixed, and the fix is longitudeSpan — the smallest arc of longitude containing every point, found as the complement of the widest gap between adjacent points.

The thing that makes it a fix rather than a trade is that the answer does not change for any set that does not straddle. For those, the widest gap IS the one that wraps from the easternmost point back round to the westernmost, so its complement runs from min to max and the centre is the same place the average used to give, arrived at by the general rule.

The same \*place\*, not the same float. The old path added two longitudes and halved; this one adds half a width to an endpoint, which is one subtraction fewer and cancels less. A real set of pins in Austin moved from -97.75800000000001 to -97.758 — fourteen significant figures of agreement, a few nanometres of ground, and a re-recorded snapshot. Worth saying plainly because "nothing moves" is what this paragraph wanted to claim and is not quite what is true.

The centre it returns is the middle of that arc, which is the contract a \*fit\* wants. A circular mean — the direction of the summed unit vectors — is the other candidate and is the wrong one here: it is pulled by clusters, so nineteen pins in Tokyo and one in Honolulu would centre on Tokyo and leave the twentieth off screen, which is precisely what a fit must not do.

#### An empty set has no answer

FitRegion of nothing returns the zero Region and false. There is no sensible centre for no points, and the tempting answer — 0,0 — is a real place in the Gulf of Guinea that a map will happily draw. Same rule as comps.StaticMap's "Zero is a place": a caller with nothing to show must not render the map at all.

<small>[comps/map_panel.go:212](https://github.com/rohanthewiz/grmob/blob/master/comps/map_panel.go#L212)</small>

### func GoogleMapsHandoff

```go
func GoogleMapsHandoff(lat, lng float64, label string) string
```

GoogleMapsHandoff is the default tap target: Google's documented cross-platform maps URL.

It is the one URL all three platforms resolve, and on both phones it reaches the installed maps app rather than a browser tab — which is the whole point of handing off at all.

The label is deliberately not in it, and the reason is worth stating because the URL has a slot that looks like it wants one. \`query\` is a \*search\*: given a name it runs a geocode, which for "St Mary's" lands on whichever St Mary's the search liked, possibly in another country. Given a coordinate pair it lands on the coordinate. The app knows exactly where this place is, so it says so, and the pin is unnamed rather than wrong.

A caller who wants the name \*and\* the point has to pick a platform to say it to — \`geo:lat,lng?q=lat,lng(Label)\` on Android, \`[https://maps.apple.com/?ll=lat,lng&q=Label](https://maps.apple.com/?ll=lat,lng&q=Label)\` on iOS — which is what the label parameter on this signature is for.

<small>[comps/static_map.go:422](https://github.com/rohanthewiz/grmob/blob/master/comps/static_map.go#L422)</small>

### func OSMStaticMap

```go
func OSMStaticMap(a StaticMapArea) string
```

OSMStaticMap renders through staticmap.openstreetmap.de, the OpenStreetMap community's static-image service.

Deprecated: that service no longer exists. The OpenStreetMap wiki's StaticMapLite page says "This service has been discontinued", and the host stopped resolving — NXDOMAIN, checked 2026-09-11. Every URL this function builds is now a fetch that fails and a frame that stays empty.

It is kept rather than deleted for two reasons. A caller that names it still compiles, which is the difference between a deprecation and a breakage; and a provider that returns a URL to a dead host is a far easier thing to diagnose than a symbol that has gone away, because the URL is printable and the failure is one nslookup from being understood.

It was this package's default, on the strength of being keyless. See "The image is a network fetch, and the provider is required" on StaticMap for what replaced it, which is nothing, and why.

Scale is ignored. The service had no scale parameter while it was up.

The marker style name ("ol-marker") is the service's own vocabulary rather than anything this package defines, which is the general shape of a provider function — it translates a StaticMapArea into one service's dialect and nothing more.

<small>[comps/static_map.go:357](https://github.com/rohanthewiz/grmob/blob/master/comps/static_map.go#L357)</small>

### func OpenStreetMapHandoff

```go
func OpenStreetMapHandoff(lat, lng float64, label string) string
```

OpenStreetMapHandoff opens the point on openstreetmap.org, for an app that would rather not send its users to Google.

It opens a browser on every platform, including the two with a maps app installed — OSM has no app with a URL scheme to claim the link. That is the trade, and it is the reason this is not the default: directions are what a person taps a map for, and a browser is a worse place to get them.

<small>[comps/static_map.go:436](https://github.com/rohanthewiz/grmob/blob/master/comps/static_map.go#L436)</small>

### func PlaceCount

```go
func PlaceCount(n int) string
```

PlaceCount is the caption a MapPanel usually wants: how many places are on the map, which is not the same number as how many things the caller has.

A recurring event is many entries and one pin; an item with no coordinates is not on the map at all. Saying "3 places" rather than "3 events" is the difference between a caption and a wrong count, and it is a mistake worth one exported function to not make twice.

<small>[comps/map_panel.go:383](https://github.com/rohanthewiz/grmob/blob/master/comps/map_panel.go#L383)</small>

## Types

### type AlarmRinging

```go
type AlarmRinging struct {
	Alarm alarm.Alarm

	// Hour24 writes the time as 06:30.
	Hour24 bool

	OnSnooze  func()
	OnDismiss func()

	// Style is applied last, to the panel.
	Style []core.StyleProp
}
```

AlarmRinging is the screen an alarm puts up while it rings: the time it was set for, its label, and two buttons.

	ringer := hooks.UseAlarms(ctx, alarms.Get(), opts)
	if a, ok := ringer.Ringing(); ok {
	    return comps.AlarmRinging{Alarm: a, OnSnooze: ringer.Snooze, OnDismiss: ringer.Dismiss}
	}

	          ┌───────────────────────┐
	          │        6:30 AM        │   the alarm's time, display size
	          │        Wake up        │   label, if any
	          │                       │
	          │ [       Snooze      ] │   filled: the easy target half-awake
	          │ [      Dismiss      ] │   outlined
	          └───────────────────────┘

#### Why Snooze is the big button

Every alarm clock makes snooze the easy target and dismiss the deliberate one, because the costly mistake is dismissing by accident — a snooze pressed by mistake costs nine minutes, a dismiss pressed by mistake costs the morning. Filled against outlined is that distinction in this library's vocabulary. OnSnooze nil drops the button, for an alarm with no snooze.

#### Accessibility

The panel is an alert (core.RoleAlert), so a screen reader announces it when it appears rather than waiting for the user to find it, and its label is a sentence: "Alarm, 6:30 AM, Wake up". The time and label inside are hidden to avoid reading that twice; the buttons are ordinary buttons.

<small>[comps/alarm.go:76](https://github.com/rohanthewiz/grmob/blob/master/comps/alarm.go#L76)</small>

#### func (AlarmRinging) Render

```go
func (r AlarmRinging) Render(ctx *core.Context) *core.Node
```

<small>[comps/alarm.go:89](https://github.com/rohanthewiz/grmob/blob/master/comps/alarm.go#L89)</small>

### type AlarmRow

```go
type AlarmRow struct {
	Alarm alarm.Alarm

	// Hour24 writes the time as 06:30 rather than 6:30 AM.
	Hour24 bool

	// OnToggle receives the new Enabled value.
	OnToggle func(on bool)

	// Style is passed through to the row.
	Style []core.StyleProp
}
```

AlarmRow is one alarm in a list: its time as the title, its label and repeat days under it, and a switch that turns it on and off.

	core.For(alarms.Get(), func(a alarm.Alarm, i int) core.View {
	    return core.Keyed(a.ID, comps.AlarmRow{Alarm: a, OnToggle: func(on bool) { setEnabled(a.ID, on) }})
	})

It is a SwitchRow, so everything in that type's doc applies — the whole row toggles, and OnToggle is a setter that receives the new value. The subtitle is "Wake up · Weekdays", or just the days for an unlabelled alarm, which is what a reader hears as the switch's hint after hearing the time as its name.

<small>[comps/alarm.go:19](https://github.com/rohanthewiz/grmob/blob/master/comps/alarm.go#L19)</small>

#### func (AlarmRow) Render

```go
func (r AlarmRow) Render(ctx *core.Context) *core.Node
```

<small>[comps/alarm.go:32](https://github.com/rohanthewiz/grmob/blob/master/comps/alarm.go#L32)</small>

### type AnalogClock

```go
type AnalogClock struct {
	// Time is the instant to draw, in the location it should be read in.
	Time time.Time

	// Size is the dial's diameter in px; 0 means 200.
	Size float64

	// ShowSeconds draws the second hand.
	ShowSeconds bool

	// Numerals draws 1–12 inside the ticks.
	Numerals bool

	// MinuteTicks adds a fine tick for every minute between the hour ticks.
	// Off by default: it is 48 more layers, and at small sizes the ticks run
	// together into a ring.
	MinuteTicks bool

	// Smooth animates each step of the hands (a short ease-out) instead of
	// jumping like a quartz movement. See "Angles" above for the midnight
	// caveat that comes with it.
	Smooth bool

	// Face fills the dial and Ink draws the ticks, numerals and the hour and
	// minute hands; empty uses the theme's Surface and TextPrimary.
	// SecondColor draws the second hand and hub, and defaults to Primary so
	// the fastest-moving part is the one the eye can find.
	Face        string
	Ink         string
	SecondColor string

	// Style is applied last, to the stack.
	Style []core.StyleProp

	// AccessibilityLabel overrides the spoken time.
	AccessibilityLabel string
}
```

AnalogClock draws a time as a clock face with hands.

	now := hooks.UseNow(ctx, time.Second)
	comps.AnalogClock{Time: now, ShowSeconds: true, Numerals: true}

#### How a hand pivots at the centre

core.Rotate turns a node about its own centre and deliberately has no transform-origin; its doc says to wrap the thing in a box whose centre is the pivot. So every hand, tick and numeral is its own layer of a ZStack, and each layer is exactly the size of the dial. The layer is what turns, and its centre is the dial's centre:

	┌───────────────┐   one layer, size × size, Rotate(angle)
	│               │
	│   (spacer)    │   height = size/2 − length
	│       ┃       │
	│       ┃       │   the hand: length + tail tall, horizontally centred
	│       ● ──────┼── the layer's centre, which the hand's lower end passes
	│       ┃       │   tail (second hand only)
	│               │
	└───────────────┘

Everything outside the stick is transparent, so a stack of such layers draws as a face with hands. The cost is one node per layer, which on every target is a view with no content, and it uses only primitives that already agree on all four (see Style.Rotate's table) — no shape primitive needed.

#### Angles, and why they do not wrap

All three hands are derived from the seconds elapsed in the local day:

	hour   = s / 120     0 … 720°   (two turns a day)
	minute = s / 10      0 … 8640°
	second = s × 6       0 … 518400°

Unwrapped because Smooth animates each change with a Transition, and a transition from 354° to 0° sweeps the long way back. Rotate passes its value through unnormalised for exactly this reason. The largest value is well within Compose's Float precision (about 0.03° at that magnitude).

The day does roll over. At midnight every hand's angle falls to zero, which under a Transition would spin the second hand backwards 1,440 turns; so the pass whose time is in the first second of the day omits the Transition and the hands jump. A caller that skips that exact second (an app suspended over midnight) sees one backwards sweep, and only with Smooth.

#### Accessibility

One element that says the time, as DigitalClock does; the face is hidden.

<small>[comps/clock.go:216](https://github.com/rohanthewiz/grmob/blob/master/comps/clock.go#L216)</small>

#### func (AnalogClock) Render

```go
func (c AnalogClock) Render(ctx *core.Context) *core.Node
```

<small>[comps/clock.go:271](https://github.com/rohanthewiz/grmob/blob/master/comps/clock.go#L271)</small>

### type AudioPlayer

```go
type AudioPlayer struct {
	// Track is what this player plays. URL is required, and is how the
	// widget recognizes its own track in the shared status.
	Track core.AudioTrack

	// SkipSeconds is the back / forward step; 0 means 15, and a negative
	// value leaves the skip buttons out.
	SkipSeconds float64

	// Rates, when set, adds a speed button that cycles through them in order
	// ("Speed 1.25×"). A rate the player is at that is not in the list steps
	// to the first.
	Rates []float64

	// ShowStop adds a Stop button, which unloads the track.
	ShowStop bool

	// Style is applied to the outer column after its defaults.
	Style []core.StyleProp
}
```

AudioPlayer is the transport for one track on the app's one player: the title, a seek bar with the elapsed and total time under it, and back / play-pause / forward, with an optional speed button and Stop.

	comps.AudioPlayer{
	    Track: core.AudioTrack{URL: sermon.URL, Title: sermon.Title, Artist: sermon.Speaker},
	    Rates: []float64{1, 1.25, 1.5, 2},
	}

	┌ Column  role=group  name=Title ──────────────────────┐
	│ Sunday, 14 March                                     │  Typography.Body, bold
	│ Pastor Ade                                           │  Artist, or the state
	│ ●━━━━━━━━━━━━━━━━○──────────────────────────────     │  Slider, seeks on release
	│ 12:04                                        41:30   │  elapsed · total
	│            [ −15s ]  [ Pause ]  [ +15s ]             │
	│               [ Speed 1.25× ]  [ Stop ]              │  when Rates / ShowStop
	└──────────────────────────────────────────────────────┘

#### One player, many widgets

core's audio is a singleton (see core/audio.go): one stream, one media session, one lock screen. So an AudioPlayer does not own a player, it is a view of the one there is, and it asks one question of the status: is the loaded track mine (same URL)? If it is, the controls drive it. If it is not — nothing loaded, or another screen's track — this widget shows its own track idle, Play loads it (replacing whatever was playing, as a phone does), and the other controls are disabled because they would act on somebody else's stream. A list of sermons can therefore put an AudioPlayer on every detail screen without any of them fighting.

#### The scrub reading is the widget's

While the thumb is down, the elapsed time follows the finger, and the seek happens once, on release (core.OnSliderChangeEnd). The thumb itself needs nothing — the native renderers show the finger's value during a drag — but the time label does, so the widget holds "dragging to t" in a hook.

That is the opposite of SliderRow, which gives its draft to the caller, and the difference is what the reading is for. SliderRow's live reading is decoration on a value the app owns, and holding it would charge every SliderRow a whole-tree render per drag tick. Here the value being drafted is the host's playback position, which no app holds, and the reading is the point of scrubbing: you are looking for 12:04. The cost is also already paid — a playing track re-renders the tree on every status tick.

#### Accessibility

The column is a RoleGroup named by the track's title, so a reader entering it hears what is playing. The seek bar is named "Position" with its value spoken as "12:04 of 41:30" rather than as seconds. The skip buttons show "−15s" and are named "Back 15 seconds" / "Forward 15 seconds".

#### Theme roles read

	Title        Typography.Body, bold
	Second line  Typography.Caption over TextSecondary
	Times        Typography.Caption over TextSecondary
	Controls     comps.Button: Play filled, the rest outlined
	Gaps         Spacing.SM

<small>[comps/audio_player.go:76](https://github.com/rohanthewiz/grmob/blob/master/comps/audio_player.go#L76)</small>

#### func (AudioPlayer) Render

```go
func (p AudioPlayer) Render(ctx *core.Context) *core.Node
```

Render draws the transport. It takes two hook slots — the audio status subscription and the scrub reading — so render it unconditionally.

<small>[comps/audio_player.go:99](https://github.com/rohanthewiz/grmob/blob/master/comps/audio_player.go#L99)</small>

### type Avatar

```go
type Avatar struct {
	// Src is the image URL. Empty falls back to the initials disc.
	Src string

	// Name is the person's full name: the source of both the derived initials
	// and the accessibility label.
	Name string

	// Initials overrides what the fallback disc shows. Set it when the derived
	// pair is wrong — a mononym, a handle, a name whose ordering the
	// first-word/last-word rule gets backwards.
	Initials string

	// Size is the diameter in px; 0 means 40.
	Size float64

	// Background is the fallback disc's fill; empty uses the theme's Primary.
	// TextColor is the initials' ink; empty uses the theme's Background, which
	// is the palette's designated on-Primary color.
	Background string
	TextColor  string

	// Style is applied last, over both branches, so it can restyle the image
	// and the disc identically (a ring, a shadow, a square crop).
	Style []core.StyleProp

	// AccessibilityLabel names the avatar, overriding Name. See the type
	// comment for what happens when both are empty.
	AccessibilityLabel string
}
```

Avatar is the circular portrait that fronts a person in a list row, a header, or a comment: a remote image when there is one, and initials on a colored disc when there is not.

	comps.Avatar{Src: user.PhotoURL, Name: user.Name}   // image, labelled
	comps.Avatar{Name: "Ada Lovelace"}                  // "AL" on a disc

#### The circle

Both branches are the same square with BorderRadius = Size/2. An oversized radius would be simpler (Badge uses 999 to get a stadium at any height), but a circle needs the radius to track the diameter exactly: a fixed 999 on a square still yields a circle, yet any caller Style that changes Size would silently keep the old geometry. Deriving it means Size stays the single knob.

#### A note on non-square images

The iOS renderer scales images with .scaledToFit, so a portrait that is not square letterboxes inside the circle rather than filling it (Compose's AsyncImage defaults to Fit as well). That is a renderer-level choice shared with every other Image in the framework, not something Avatar can set from Go today — the fix is a ContentMode prop on Image, which would want its own pass across both renderers.

#### Accessibility

Unlike ListRow, Avatar \*does\* synthesize a label, because it can: an avatar has exactly one meaning, the person it depicts, and Name is that meaning. The rule is:

	AccessibilityLabel set  -> used verbatim
	Name set                -> used as the label
	neither                 -> the node is hidden from assistive tech

The last case is the important one. An avatar with no name is decoration sitting next to text that already names the person, and an unlabeled image in that position is announced as an unhelpful "image" — or, worse, as its URL. Hiding it is the correct default rather than a fallback.

<small>[comps/avatar.go:50](https://github.com/rohanthewiz/grmob/blob/master/comps/avatar.go#L50)</small>

#### func (Avatar) Render

```go
func (a Avatar) Render(ctx *core.Context) *core.Node
```

<small>[comps/avatar.go:81](https://github.com/rohanthewiz/grmob/blob/master/comps/avatar.go#L81)</small>

### type AvatarStack

```go
type AvatarStack struct {
	// Avatars are the faces, drawn leading to trailing. Each one's Size is
	// replaced by the stack's, so a stack is one size throughout.
	Avatars []Avatar

	// Max is the most discs drawn, counting the "+N" disc. Zero or less draws
	// every face.
	Max int

	// Size is each face's diameter in px; 0 means 32, a row's worth.
	Size float64

	// Overlap is how much of each face the next one covers, as a fraction of
	// Size; 0 means 0.2. Kept below 0.5 by the reader's eye rather than by
	// the widget — past half, a face is more hidden than shown. The default is
	// set by initials rather than photos: a photo survives losing a third of
	// itself, but at 0.3 and at 0.25 the next disc and its ring cut into the
	// second letter of a two-letter pair (seen in a static export's serif,
	// the widest face any target draws). 0.2 is also what the web's common
	// avatar groups ship.
	Overlap float64

	// RingWidth is the gap drawn around each face in px; 0 means 2. Negative
	// draws no ring.
	RingWidth float64

	// RingColor is the ring's fill; empty takes the theme's Background, which
	// is right when the stack sits on the screen's own background. On a Card
	// or a Surface panel, pass that panel's fill.
	RingColor string

	// Label replaces the synthesized accessible name. See "Accessibility".
	Label string

	// Style is applied to the stack after its defaults.
	Style []core.StyleProp
}
```

AvatarStack is the overlapping row of faces that says who is in a thread, who is going, who has seen it — with a "+N" disc when there are more than fit.

	comps.AvatarStack{Avatars: attendees, Max: 4}

	  ╭──╮╭──╮╭──╮╭──╮
	 │AL││GH││KJ││+3│      Max 4 of 6: three faces and the surplus
	  ╰──╯╰──╯╰──╯╰──╯

#### A ZStack, because a Row cannot overlap

Overlap in a Row needs a negative margin or a negative gap, and neither is portable: CSS honours both, Compose's spacedBy and padding reject a negative value, and SwiftUI's stack spacing takes one but lays the row out wider than it draws. So the stack is a core.ZStack whose every layer is placed core.StackAlignStart — the leading edge, centred vertically — and pushed right by a MarginLeft that grows by one step per face. A margin on a stack layer is the same machinery Screen.Floating spends to hold a FAB off the corner, so nothing here is new to any renderer.

	stack  Width  = ring + step·(n-1),  Height = ring
	layer  2i     the ring disc,  MarginLeft(step·i)
	layer  2i+1   the avatar,     MarginLeft(step·i + RingWidth)

The stack's box is pinned, as core.ZStack asks: "start" of a box with no size is wherever the largest layer happens to end.

#### The ring is a layer, not a border

Each face sits on a disc of the theme's Background a little larger than itself, which is what cuts the face under it and keeps two photos from blurring into one. It is drawn as its own layer rather than as a border on the Avatar. When this was written a border was sized differently across targets: the natives painted it inside the box and the static export, then content-box, painted it outside, so the overlap arithmetic was off by 4px on the web. htmlout now writes a border-box rule for sized, bordered nodes, so the four agree; the layer stays because a plain Box with a width, a height and a fill needs no target to agree about anything.

#### Later faces sit on top, and the surplus last of all

Render order is paint order, so each face overlaps the one before it and the "+N" disc, drawn last, is never covered — it carries text, and a count half under a photo is a count nobody can read.

#### Max counts the surplus disc

Max is the most discs drawn, the "+N" among them, so a Max of 4 is a stack four discs wide whatever the list's length — the width a layout reserves. Six avatars under Max 4 draw three faces and "+3". Max ≤ 0 draws every face.

#### Accessibility

The stack is RoleImg with one name: "Ada Lovelace, Grace Hopper and 3 others". A facepile is a picture of who is here, and the reader wants the sentence once rather than six images announced in turn — the case RoleImg exists for (see core/role.go), and every face inside is hidden behind it. The name counts the faces that were not drawn as well as the unnamed ones, because both are people the picture stands for. Label replaces it, for a caller that wants "6 attendees" or a language other than English.

#### Theme roles read

	Faces       Avatar's own roles (Primary disc, Background initials)
	Ring        Colors.Background
	Surplus     Colors.Surface disc, TextSecondary count

<small>[comps/avatar_stack.go:79](https://github.com/rohanthewiz/grmob/blob/master/comps/avatar_stack.go#L79)</small>

#### func (AvatarStack) Render

```go
func (s AvatarStack) Render(ctx *core.Context) *core.Node
```

Render builds ZStack(ring, face, ring, face, …, ring, +N).

<small>[comps/avatar_stack.go:118](https://github.com/rohanthewiz/grmob/blob/master/comps/avatar_stack.go#L118)</small>

### type Compass

```go
type Compass struct {
	// Heading is the bearing to draw, in degrees clockwise from north. Any
	// value works: it is normalised for the readout and the spoken label, and
	// applied as-is to the rotation (see core.Style.Rotate on winding).
	Heading float64

	// Size is the rose's diameter in px; 0 means 160.
	Size float64

	// ShowDegrees adds the numeric readout under the rose — "312° NW". Off by
	// default: a compass beside a map is a picture, and the number is noise
	// until something asks for it.
	ShowDegrees bool

	// Background is the rose's fill and Color its lettering and border; empty
	// uses the theme's Surface and TextSecondary. NorthColor inks the N and
	// the index mark, and defaults to the theme's Primary — north is the one
	// letter a reader looks for, and the accent is what makes it findable
	// while the rose is turning.
	Background string
	Color      string
	NorthColor string

	// Style is applied last, to the outer column, so a caller can space the
	// widget or give the whole thing a margin without reaching inside it.
	Style []core.StyleProp

	// AccessibilityLabel overrides the spoken bearing. See the type comment.
	AccessibilityLabel string
}
```

Compass draws a bearing as a compass rose: a round card lettered N/E/S/W that turns so its N points the way north actually is, under a fixed index mark at the top showing where the device is pointing.

	h := hooks.UseHeading(ctx)
	comps.Compass{Heading: h.Magnetic, ShowDegrees: true}

	      ┌──▼───┐        index mark — fixed, over the rose's rim
	      │   N  │        the rose turns by -Heading, so N stays
	      │ W   E│        pointing at north while the device turns
	      │   S  │
	      └──────┘
	      312° NW         optional readout

#### Which half turns

Two conventions exist and they are not interchangeable. A magnetic compass has a fixed card and a needle that swings to north. A navigation compass — what every phone ships — turns the whole card and keeps a fixed index at twelve o'clock, so the bearing is read where the index crosses the rose. This is the second one, because the second one answers the question a phone user is asking ("which way am I facing", read off the top) rather than the question a hiker with a paper map is asking ("where is north").

The rose therefore rotates by \*minus\* the heading. Turning the device clockwise increases the heading, and the rose must turn counter-clockwise by the same amount to keep pointing at the same piece of the world.

#### Where the index mark sits, and where it used to

On the rose's rim, at twelve o'clock, drawn over it — which is where a compass index belongs and where this one could not go for as long as core had no z-axis container. Box stacks its children vertically on all four targets (settled deliberately — see the "Box overlay divergence" note in core.Box), and CSS absolute positioning is a declared web-only prop neither native renderer reads, so anything drawn \*over\* the rose would have been a web-only widget wearing a portable name. The mark was parked in the row above instead, costing a glyph of height and reading as a separate thing pointing at the dial rather than as part of it.

core.ZStack is that container, and this is its first consumer. The dial is a stack of two layers — the rose, then the mark, which asks for the top with core.StackAlign because a ZStack centres every layer that says nothing.

The mark used to be wrapped in a full-height column justifying its child to the start, which is the escape core.ZStack documented while it had no per-child alignment. This widget being its only consumer is what kept the prop out; StackAlign is the second half of that argument arriving.

The rose's inset went from half a letter to a whole one to make room. The mark's glyph is three quarters of a letter tall, so a ring that deep is what keeps N clear of it at heading zero — the one bearing where the two deliberately coincide, since an index pointing at N is exactly what facing north looks like.

#### A float, not a core.Heading

The field is a bearing in degrees rather than the sensor's own struct, so the widget also draws a bearing that has nothing to do with the compass — the direction of a route leg, a wind reading, the way a photograph was taken. hooks.UseHeading hands over the sensor's; anything else can hand over its own, and neither has to know about the other.

#### Accessibility

The rose is four letters whose positions carry the meaning, which is exactly the content a screen reader cannot convey: read aloud in tree order it is "N W E S", turning or not. So the whole widget announces once, as a spoken bearing ("Heading 312 degrees, northwest"), and every part inside it is hidden. AccessibilityLabel overrides the sentence for a caller whose bearing means something more specific than a heading.

The container takes core.RoleImg, and it is still the right value now that the two web exporters supply core.RoleGroup to any named container that has none (see core.RoleGroup). Both make the label legal — ARIA forbids an accessible name on a generic element, which is what a plain core layout node exports as, so the label alone was dropped by screen readers on both web targets while both natives read it out. What only \`img\` says is the part this widget depends on: a picture standing in for one fact, whose parts a reader should not read. \`group\` names its children and leaves them readable, which for a rose is "Heading 312 degrees, northwest" followed by "N W E S".

<small>[comps/compass.go:90](https://github.com/rohanthewiz/grmob/blob/master/comps/compass.go#L90)</small>

#### func (Compass) Render

```go
func (c Compass) Render(ctx *core.Context) *core.Node
```

<small>[comps/compass.go:121](https://github.com/rohanthewiz/grmob/blob/master/comps/compass.go#L121)</small>

### type Countdown

```go
type Countdown struct {
	// Until is the deadline. The widget draws the time from now to here,
	// clamped at zero.
	Until time.Time

	// OnDone is called once when the deadline passes, from the hook's effect
	// and so on its own goroutine. Nil is a display-only countdown, which
	// also stops ticking the moment it is hidden.
	OnDone func()

	// Format writes the digits. It receives the remaining time already
	// rounded up to a whole second — the same number the default writes — so
	// a custom format cannot disagree with the widget about which second it
	// is showing. Nil uses the phone-timer format: M:SS under an hour,
	// H:MM:SS at or over one.
	Format func(time.Duration) string

	// Size is the digits' font size in px; 0 means 40, as in DigitalClock.
	Size float64

	// Color inks the digits; empty uses the theme's TextPrimary.
	Color string

	// Hidden removes the widget from display, which also stops its tick
	// unless OnDone is still owed. Prefer it to leaving the widget out of the
	// tree; see "It holds hooks".
	Hidden bool

	// AccessibilityLabel overrides the spoken remaining time.
	AccessibilityLabel string

	// Style is applied last, to the digits.
	Style []core.StyleProp
}
```

Countdown draws the time left until a deadline, one tick a second, and reports once when it runs out.

	comps.Countdown{Until: expiresAt, OnDone: func() { code.Set("") }}

	  2:59      the digits, in DigitalClock's face

It is the half of the clock family that watches a \*duration\* rather than an instant: an alarm row that wants to say how long until it rings, a one-time-code field that wants to say how long the code is good for, a rest timer between sets. Stopwatch, below, is the same widget counting the other way.

#### It holds hooks, so it is not conditional-safe

Unlike DigitalClock — which takes a time.Time and holds nothing — a countdown has to know what "now" is, so it owns a tick. That makes it a hook caller, with the rule Accordion and Snackbar document: render it in a stable position on every pass and drive Hidden, rather than wrapping it in a core.If. A hidden countdown is Display none, so it costs no pixels.

#### What the tick is, and what it is not

The tick is hooks.UseIntervalWhile with an empty callback: the widget reads the clock itself in Render, so all a tick has to do is bring the render back. It runs only while there is a reason for it:

	state                       ticking
	────────────────────────    ───────
	counting, visible           yes
	counting, Hidden            only if OnDone is set
	finished                    no
	Hidden and no OnDone        no

The middle row is the one worth stating. Hiding a countdown removes the first reason to tick (nothing to draw) but not the second (somebody is waiting to be told it ran out), so a hidden countdown that owes an OnDone keeps counting. A hidden one that owes nothing stops dead.

#### Why not hooks.UseNow

UseNow is the clock hook and aligns its ticks to the wall clock, so a DigitalClock changes its seconds digit when the phone's status bar does. A countdown has no such phase to share: its own boundaries fall at Until minus a whole number of seconds, which is a phase nothing else on the screen is on. There being nothing to align to, the cheaper hook wins — and UseNow cannot be paused, which the table above needs.

The visible consequence is that the deadline is noticed on the first tick at or after it, so the digits reach 0:00 up to a second late. The reading is rounded \*up\* to compensate, which makes the lag conservative rather than arbitrary: a Countdown never tells you that you have less time left than you do.

#### OnDone comes from the effect, not from the render

A render pass is not a place to run a handler — it may run more than once for one state, it runs while the tree is being built, and a handler that set state from inside it would re-enter the renderer. So OnDone is a hooks.UseEffect keyed on whether the deadline has passed, which gives it exactly the semantics the name implies:

	remaining  5s ──── 4s ──── … ──── 1s ──── 0 ──── 0 ──── 0
	deps       false   false         false   true   true   true
	OnDone      ·       ·             ·      fire    ·      ·

Once per crossing, on the tick that crosses, off the render goroutine. Two consequences follow from "per crossing" rather than "per widget":

  - A Countdown whose Until is already in the past when it first renders fires immediately. That is the correct reading of a deadline restored from disk while the app was closed, and it is why the zero Until is a concern rather than a quiet no-op.
  - Moving Until forward re-arms it. A restart is Until: time.Now().Add(d) and nothing else; the widget needs no reset call.

OnDone is a display-grade signal and not a scheduler. It only fires while the widget is rendered and the app is running, and it is late by up to one tick. Something that must happen at a time whether or not anyone is looking belongs in the alarm package and hooks.UseAlarms.

#### Accessibility

The digits are one element with RoleImg and a spoken label — "4 minutes 12 seconds remaining", "Time is up" — for the reason DigitalClock gives: read as text, "4:12" is punctuation, and RoleImg is what makes a label survive on the web. The label is not a live region, so a screen reader is not told the new number every second; a caller who wants the announcement puts the Countdown beside its own core.RoleStatus text.

#### Theme roles read

	Digits   Colors.TextPrimary, unless Color says otherwise

<small>[comps/timers.go:122](https://github.com/rohanthewiz/grmob/blob/master/comps/timers.go#L122)</small>

#### func (Countdown) Render

```go
func (c Countdown) Render(ctx *core.Context) *core.Node
```

Render reads the clock once, arms the tick and the effect, and draws the digits.

<small>[comps/timers.go:159](https://github.com/rohanthewiz/grmob/blob/master/comps/timers.go#L159)</small>

### type DigitalClock

```go
type DigitalClock struct {
	// Time is the instant to draw, in the location it should be read in.
	Time time.Time

	// Hour24 draws 22:42 instead of 10:42 PM.
	Hour24 bool

	// ShowSeconds adds :07 to the digits. Off by default: a clock that only
	// changes once a minute can be driven by hooks.UseNow(ctx, time.Minute).
	ShowSeconds bool

	// ShowDate adds a line with the weekday, month and day.
	ShowDate bool

	// Size is the digits' font size in px; 0 means 40. The AM/PM marker and
	// the date line scale from it.
	Size float64

	// Color inks the digits; empty uses the theme's TextPrimary. The date line
	// always uses TextSecondary, so it reads as subordinate.
	Color string

	// Style is applied last, to the outer column.
	Style []core.StyleProp

	// AccessibilityLabel overrides the spoken time.
	AccessibilityLabel string
}
```

DigitalClock draws a time as digits, with an optional date line under them.

	now := hooks.UseNow(ctx, time.Second)
	comps.DigitalClock{Time: now, ShowSeconds: true, ShowDate: true}

	    10:42:07 PM         digits, with the AM/PM marker set smaller
	 Wednesday, September 16    optional date

#### A time, not a ticker

The widget draws whatever Time it is handed and holds no hooks, so it can be rendered conditionally, tested at a fixed instant, and fed a time that is not "now here" (a world clock is Time: now.In(tokyo)). hooks.UseNow is the ticker; its doc explains why it aligns to the wall clock rather than to mount.

#### Accessibility

Read in tree order the parts are "10:42:07", "PM", "Wednesday, September 16" — three stops for one fact. So the whole widget is one element that announces a sentence, with its parts hidden, the same shape Compass uses and for the same reason: RoleImg is what makes the label survive on the web (see Compass's Accessibility section).

#### Known limit

core.Style has no font family, so the digits are the platform's proportional ones and the line's width can change by a pixel or two as the digits do. Centring (the default alignment here) keeps that from reading as a jitter at either edge.

<small>[comps/clock.go:41](https://github.com/rohanthewiz/grmob/blob/master/comps/clock.go#L41)</small>

#### func (DigitalClock) Render

```go
func (c DigitalClock) Render(ctx *core.Context) *core.Node
```

<small>[comps/clock.go:70](https://github.com/rohanthewiz/grmob/blob/master/comps/clock.go#L70)</small>

### type ECLevel

```go
type ECLevel string
```

ECLevel is a QR Code's error-correction level: how much of the symbol is redundancy, and so how much of it may be covered, smudged or reflected off and still read.

A string enum with an empty zero value, the package's idiom (see Variant), so that adding the field to an existing QRCode changes nothing. The values are the specification's own one-letter names, which is what every other QR tool a developer will compare against prints.

<small>[comps/qr_code.go:16](https://github.com/rohanthewiz/grmob/blob/master/comps/qr_code.go#L16)</small>

```go
const (
	// ECDefault is the zero value and means ECMedium.
	ECDefault ECLevel = ""

	// ECLow recovers about 7% of the symbol, ECMedium about 15%, ECQuartile
	// about 25% and ECHigh about 30%.
	ECLow      ECLevel = "L"
	ECMedium   ECLevel = "M"
	ECQuartile ECLevel = "Q"
	ECHigh     ECLevel = "H"
)
```

### type ExpandableText

```go
type ExpandableText struct {
	// Text is the full text.
	Text string

	// Lines is the collapsed cap; 0 means 3.
	Lines int

	// ToggleAfter is the length in runes past which the text is capped and
	// the toggle shown; 0 means Lines × 40, negative always shows it. See
	// "When the toggle shows".
	ToggleAfter int

	// MoreLabel and LessLabel caption the toggle; empty gives "Read more"
	// and "Read less". MoreLabel is also its accessible name in both states.
	MoreLabel, LessLabel string

	// Style is applied to the text after its defaults.
	Style []core.StyleProp
}
```

ExpandableText is body text capped at a few lines, with a "Read more" that opens it in place and a "Read less" that closes it again: a product description, a review, the summary of an episode.

	comps.ExpandableText{Text: episode.Summary, Lines: 3}

	┌ Column ────────────────────────────────────────────┐
	│ The third episode follows the team to the coast,   │  core.MaxLines(3)
	│ where the survey that was meant to take a week     │  while collapsed
	│ turns into a month of weather, tides and a …       │
	│ [ Read more ]                                      │  ghost, aria-expanded
	└────────────────────────────────────────────────────┘

#### When the toggle shows: a threshold, because nothing measures

The honest rule is "show Read more when the cap actually cut something", and no host can say whether it did: that is a rendered height, the layout measurement Tooltip is blocked on. A toggle that is always there shows "Read more" under a two-line paragraph, which opens onto nothing.

So the rule is a length the caller can tune. The toggle shows when the text has more than ToggleAfter characters (runes) — by default Lines × 40, about what fits a phone's line of body text. A caller who knows better states it: a negative ToggleAfter always shows the toggle, a very large one never does. Where the estimate is wrong the failure is mild in both directions: a toggle that reveals a few more words, or a paragraph a line longer than the cap that is simply shown in full — collapsed with no toggle means no cap, since a cap with no way to lift it would hide the end of the text for good.

#### The open state is the widget's

Whether the text is open is held here, in a hook — no application wants it, PasswordField's test — so render an ExpandableText unconditionally, in a stable position.

#### Accessibility

The text node carries the whole string on every target; the cap is visual only (a screen reader reads a clamped Text in full on the web, and VoiceOver and TalkBack read a lineLimit / maxLines Text's full content). The toggle is a button whose name stays "Read more" and whose expanded state is stated with core.AccessibilityExpanded — aria-expanded — so the name does not flip between two different-sounding buttons. See PasswordField's "Accessibility" for the same rule.

#### Theme roles read

	Text     Typography.Body
	Toggle   a ghost Button (Primary's on-light tone)
	Gap      Spacing.XS

<small>[comps/expandable_text.go:60](https://github.com/rohanthewiz/grmob/blob/master/comps/expandable_text.go#L60)</small>

#### func (ExpandableText) Render

```go
func (e ExpandableText) Render(ctx *core.Context) *core.Node
```

Render draws the text and, when it is long enough, the toggle. It takes one hook, the open state.

<small>[comps/expandable_text.go:82](https://github.com/rohanthewiz/grmob/blob/master/comps/expandable_text.go#L82)</small>

### type MapHandoff

```go
type MapHandoff func(lat, lng float64, label string) string
```

MapHandoff turns a point and its name into a URL for core.OpenURL. See StaticMap.Handoff.

<small>[comps/static_map.go:331](https://github.com/rohanthewiz/grmob/blob/master/comps/static_map.go#L331)</small>

### type MapPanel

```go
type MapPanel struct {
	// Pins are the points to show. An empty set renders Empty rather than a
	// map, because the alternative is a map of 0,0 — see FitRegion, which
	// refuses to answer for a set with nothing in it.
	Pins []MapPin

	// OnPinTap is called with the id of the pin the user tapped. Pins with an
	// empty id are drawn and never reported, which is core.Marker's own rule.
	OnPinTap func(id string)

	// OnMapTap is called with the coordinates of a tap that did not hit a pin.
	// The suppression is each host's; see core.OnMapTap.
	OnMapTap func(lat, lng float64)

	// Height sizes the map. "" means DefaultMapPanelHeight. Width is always
	// 100%: a map narrower than its column is a map with a gutter beside it,
	// and a caller who wants one wraps this in a box.
	Height string

	// Caption is a line under the map — a count, a legend, a note about what
	// is not on it. Empty draws nothing, including no padding.
	Caption string

	// Empty is what to render for a set with no pins in it. The zero value is
	// a generic EmptyState; a caller who knows what the reader was looking for
	// should say so, because "nothing to show" is the least useful sentence a
	// screen can end on.
	Empty EmptyState

	// Zoom overrides the fitted zoom level. 0 means "fit the pins", which is
	// what this widget is for; a value is for a caller who wants a fixed scale
	// and only the centring.
	Zoom float64

	// Style is applied last, to the outer column, so a caller can give the
	// panel a margin or a border without reaching inside it.
	Style []core.StyleProp
}
```

MapPanel is "where are these": a live map over a set of points, opened at a view that contains all of them.

	comps.MapPanel{
	    Pins: []comps.MapPin{
	        {ID: "hall", Lat: 38.7223, Lng: -9.1393, Title: "The hall"},
	        {ID: "annex", Lat: 38.7251, Lng: -9.1402, Title: "The annex"},
	    },
	    OnPinTap: func(id string) { open(id) },
	    Height:   "320px",
	}

#### What this adds to core.MapView, which is the whole of its case

One thing, and it is the thing every consumer of a set of points needs and would otherwise write itself: the \*opening region\*. core.MapView takes a Region and applies it only when it changes, which is correct and leaves the caller holding a question — what region shows all of my points? — whose answer is a bounding box, a projection correction and a logarithm. See FitRegion, which is that arithmetic and is exported for a caller who wants it without this widget.

Everything else here is arrangement a caller could write in ten lines and which is worth having in one place anyway: the pins as keyed children, the empty state for a set with nothing in it, and an optional caption.

#### What it deliberately does not add

No region state, no recentre control, no echo of OnRegionChange. The region is computed once from the pins and handed over, and where the reader takes the map from there is the reader's. An app that wants to follow the map reaches for core.MapView directly and holds the Region itself — which is the arrangement core.MapView's "The map is controlled, with the echo guard a drag needs" describes, and it is a different widget's job.

The pins changing DOES move the map: a new set is a new fitted region, which core.MapView applies because it changed. That is the right behaviour for the case this widget is for — the set is the subject — and it is the reason a caller whose pins update every few seconds wants core.MapView instead.

#### Tiles are somebody else's bandwidth

Inherited from core.MapView, and unchanged by this widget: two of the three hosts draw OpenStreetMap tiles from the project's own servers, under a usage policy. See that node's doc. Unlike StaticMap this needs no key from anyone, because each host's own engine draws the map — which is the odd asymmetry that the richer widget is the free one.

<small>[comps/map_panel.go:58](https://github.com/rohanthewiz/grmob/blob/master/comps/map_panel.go#L58)</small>

#### func (MapPanel) Render

```go
func (m MapPanel) Render(ctx *core.Context) *core.Node
```

<small>[comps/map_panel.go:313](https://github.com/rohanthewiz/grmob/blob/master/comps/map_panel.go#L313)</small>

### type MapPin

```go
type MapPin struct {
	// ID is what OnPinTap reports, and what the reconciler keys the marker by.
	// Empty is allowed — the single unnamed "you are here" pin — and is never
	// reported. See core.Marker.
	ID string
	// Lat and Lng are the point, in degrees.
	Lat, Lng float64
	// Title is the callout the platform shows when the pin is tapped. It is
	// not an accessibility label; a map's annotations are announced by each
	// platform's own map accessibility.
	Title string
}
```

MapPin is one point in a MapPanel: the data a core.Marker needs, as a value a caller can hold in a slice and sort.

A struct rather than four arguments for the reason StaticMapArea is one: a caller builds these in a loop from their own data, and a field added here is a field existing code ignores.

<small>[comps/map_panel.go:103](https://github.com/rohanthewiz/grmob/blob/master/comps/map_panel.go#L103)</small>

### type MessageBubble

```go
type MessageBubble struct {
	// Text is the message.
	Text string

	// Sender is drawn above the text of someone else's message. Leave it
	// empty to hide it — on the second of two consecutive messages from the
	// same person, or in a one-to-one chat. It is never drawn on Mine.
	Sender string

	// Mine puts the bubble on the trailing side in the Primary fill.
	Mine bool

	// Time is drawn small under the text, at the trailing edge ("10:42").
	// The widget does not format times; pass what the screen should show.
	Time string

	// MineLabel stands in for the sender in the spoken name of the reader's
	// own message; empty gives "You".
	MineLabel string

	// Style is applied to the outer row — the placement, where a margin
	// between messages belongs — after its defaults.
	Style []core.StyleProp
}
```

MessageBubble is one message in a conversation: a rounded box of text held to one side of the row — the trailing side for the reader's own, the leading side for everyone else's — with an optional sender line above the text and a time under it.

	comps.MessageBubble{Text: "Já viste a nova versão?", Sender: "Ana", Time: "10:42"}
	comps.MessageBubble{Text: "Ainda não", Mine: true, Time: "10:43"}

	┌ Row  justify=start ───────────────────────────────────┐
	│ ┌ Column  Surface + hairline ─────┐                   │  theirs
	│ │ Ana                             │  bold caption     │
	│ │ Já viste a nova versão?         │                   │
	│ │                          10:42  │  caption, end     │
	│ └─────────────────────────────────┘                   │
	└───────────────────────────────────────────────────────┘
	┌ Row  justify=end ─────────────────────────────────────┐
	│                   ┌ Column  Primary ────────────────┐ │  mine
	│                   │ Ainda não                 10:43 │ │
	│                   └─────────────────────────────────┘ │
	└───────────────────────────────────────────────────────┘

#### The two sides' colours

Mine is the theme's Primary with the ink chosen by contrast against it (Variant.Ink), the fill every chat app gives the reader's own words. Theirs has no palette role to take: there is no muted container tone (Banner's doc has the long version of that gap), and inventing one is a theme decision a chat bubble should not force. So theirs is Surface with a Border hairline — Banner's answer to the same gap — which separates it from the page in every bundled theme without a colour anybody has to choose.

#### What it is not

  - \*\*Not a thread.\*\* A conversation opens at its newest message, which is a scroll offset, and no host reports or accepts one (Carousel's wall). A caller lays bubbles out in its own Column or core.List, as examples/chat does, and the spacing between them is the caller's too.
  - \*\*No tail.\*\* The little point on a bubble's corner is one sharp corner on a rounded box, and core has one radius, not four — DateRangePicker's notch again.

#### Accessibility

The bubble is one stop named "Ana, Já viste a nova versão?, 10:42" — who, what, when — with its parts hidden, so a reader moving through a transcript hears each message whole rather than as three fragments. The reader's own messages are named with MineLabel ("You") in the sender's place, since no sender is drawn on them. Put the bubbles under a core.RoleLog container (examples/chat does) so new ones are announced.

#### Theme roles read

	Mine       Colors.Primary fill, ink by contrast (Variant.Ink)
	Theirs     Colors.Surface fill, ColorPalette.BorderColor hairline, TextPrimary ink
	Sender     Typography.Caption, bold
	Text       Typography.Body
	Time       Typography.Caption; TextSecondary on theirs, the ink on mine

<small>[comps/message_bubble.go:62](https://github.com/rohanthewiz/grmob/blob/master/comps/message_bubble.go#L62)</small>

#### func (MessageBubble) Render

```go
func (m MessageBubble) Render(ctx *core.Context) *core.Node
```

Render builds Row(justify, Column(sender?, text, time?)). It takes no hook slot.

<small>[comps/message_bubble.go:89](https://github.com/rohanthewiz/grmob/blob/master/comps/message_bubble.go#L89)</small>

### type QRCode

```go
type QRCode struct {
	// Data is encoded in byte mode, so any string is legal — the only input
	// that cannot be drawn is one too long for a version-40 symbol (2953
	// bytes at ECLow, 1273 at ECHigh).
	Data string

	// Size is the box's side in px, quiet zone included; 0 means 160.
	Size float64

	// Level is the error-correction level; the zero value is ECMedium.
	Level ECLevel

	// Quiet is the light margin in modules; 0 means the specification's four.
	// A negative Quiet draws none, for a caller who is supplying the margin
	// from the surrounding layout instead.
	Quiet int

	// Label names the code to assistive tech; empty says "QR code".
	Label string

	// Style is applied last, to the canvas.
	Style []core.StyleProp
}
```

QRCode draws its Data as a QR Code: a core.Canvas of module rectangles, encoded in Go, with no image file, no network round trip and no dependency outside this module.

	comps.QRCode{Data: "cats://pair?t=9f2c1a", Label: "Pairing code"}

#### What is drawn

Two shapes and no more: one light rectangle covering the whole box, and one path holding every dark module. The modules go in a single path rather than a Shape each for two reasons. A version-10 symbol has some three thousand modules, and three thousand child nodes is a reconciler's worst case for a drawing that is either identical between passes or wholly different. And a single path is filled once, so adjacent modules have no seam between them — separate shapes would be antialiased against each other and leave hairlines that a decoder's binarizer can read as light.

Within a row, consecutive dark modules are merged into one rectangle, which costs one comparison per module and typically halves the path.

	██ ██████ ██      one row, four runs
	└┘ └────┘ └┘      four rectangles, not eight

#### Size, and where the quiet zone comes from

Size is the side of the whole box in px, the quiet zone included. The symbol itself is therefore Size × n/(n+2·Quiet) across, where n is the version's module count — a 160px box at the default quiet zone of 4 gives a 29-module version-3 symbol about 125px of picture and 17px of margin.

Putting the margin inside the box rather than outside it is what makes the widget's footprint predictable: a caller lays out a 160px square and gets one, whatever the data does to the version.

#### Colour is not themed, and that is deliberate

A QR Code is read by a camera, not by a person, and every decoder's binarizer assumes dark modules on a light field. Painting one in a dark theme's colours — light ink on a dark surface — produces a symbol that many readers simply will not see, and the ones that do invert are the exception.

So the widget draws dark-on-light always. It uses the theme's own ink and surface when those \*are\* dark-on-light with room to spare, so that a light theme's code sits in the page rather than on a hard white patch; otherwise it falls back to black on white, which in a dark theme means a white square — the same thing every banking and payment app shows, for the same reason.

There is no Foreground or Background field. Every colour a caller could pass is either the pair already chosen or a worse one, and an unscannable QR code fails silently: it looks exactly like a working one.

#### Cost

Encoding runs on every render pass: choosing a version, laying out the blocks, and then scoring all eight data masks to pick one. For a link-sized payload that is around 0.2 ms on a current phone-class core, which is under a hundredth of a frame and not worth caching for a screen that shows a code and waits.

A code that sits in a tree re-rendering every frame — an animation, or a list being scrolled — is a different matter, and the lever is core.Cached, which QRCode is a fit for: it holds no hooks and registers no callbacks.

	core.Cached(comps.QRCode{Data: link, Label: "Pairing code"})

#### Accessibility

One image element, named by Label ("QR code" when empty). The data is never spoken: a reader announcing a 300-character URL one character at a time helps nobody, and a person who needs the link needs it as a link. Give the code a Label that says what scanning it will do, and put the underlying action on screen as well where you can.

<small>[comps/qr_code.go:138](https://github.com/rohanthewiz/grmob/blob/master/comps/qr_code.go#L138)</small>

#### func (QRCode) Render

```go
func (q QRCode) Render(ctx *core.Context) *core.Node
```

<small>[comps/qr_code.go:162](https://github.com/rohanthewiz/grmob/blob/master/comps/qr_code.go#L162)</small>

### type StatTile

```go
type StatTile struct {
	// Label is what the figure measures. Value is the figure itself,
	// pre-formatted — the widget does no number formatting, because currency,
	// grouping and locale are the caller's, not a layout widget's.
	Label string
	Value string

	// Delta is the movement line under the value: "+18 vs last week", "−3%",
	// "unchanged". Empty renders nothing.
	Delta string

	// DeltaVariant colors the delta with a semantic role. The zero value is
	// the theme's secondary ink — neutral, saying nothing about whether the
	// movement is good. See the type comment for why this one field departs
	// from the package's usual reading of VariantDefault.
	//
	// The same contrast caveat applies here as to an outlined or ghost
	// Button, and for the same reason: the role color is laid on a backdrop
	// the widget cannot see, so the theme's own numbers are what you get.
	// Against each theme's Background, Success is 2.22:1 under DefaultTheme
	// and Warning 2.20:1 — well under the 4.5:1 body-text floor, i.e. a delta
	// that is decoration rather than text. Under MaterialTheme the same two
	// are 5.13:1 and 3.08:1. So on the default palette a colored delta needs
	// to be reinforcement for wording that already says it ("+18, best week
	// this term"), never the only place the direction appears; a caller who
	// needs the color itself to be legible wants a Badge, which owns its fill
	// and picks its ink by contrast.
	DeltaVariant Variant

	// Fill makes the tile take an equal share of a row rather than hugging
	// its content, which is what a row of two or three tiles wants.
	//
	// It sets FlexGrow *and* a zero FlexBasis, which looks like belt and
	// braces and is actually what makes the four targets agree. Compose's
	// Modifier.weight and SwiftUI's equivalent divide the whole axis by
	// weight, so equal grows are already equal widths there; CSS flex-grow
	// divides only the *leftover* space, so on the web a tile with a longer
	// value would come out wider. Zeroing the basis is what makes CSS
	// distribute the whole axis too. The natives ignore FlexBasis, so the one
	// prop that is inert on two targets is precisely the one that converges
	// the other two.
	Fill bool

	// OnTap makes the whole tile a target — a KPI that opens the report
	// behind it. Wired only when non-nil, so a presentational tile registers
	// no callback.
	OnTap func()

	// AccessibilityLabel names the tile as one thing, which is usually what
	// you want: a reader moving through three tiles should hear "Attendance,
	// 412, up 18 from last week", not six fragments. Nothing is synthesized
	// when it is empty — the three lines are then announced separately, which
	// is correct but wordier.
	AccessibilityLabel string
	AccessibilityHint  string

	// Style is applied after the widget's own defaults.
	Style []core.StyleProp
}
```

StatTile is one figure with its name and, optionally, its movement — the unit a dashboard, a profile header, or an account summary is made of.

	core.Row(core.Gap(12),
	    comps.StatTile{Label: "Attendance", Value: "412", Fill: true,
	        Delta: "+18 vs last week", DeltaVariant: comps.VariantSuccess},
	    comps.StatTile{Label: "Giving", Value: "MZN 42,750", Fill: true},
	)

#### It has no frame, deliberately

"Tile" names the content, not a card. The widget renders a text stack and paints nothing: no background, no border, no padding of its own. That is what lets the two common arrangements both be composition rather than configuration — three tiles inside one core.Card, or three tiles each in their own — instead of a Framed bool that is wrong half the time. The fintech example's balance block was already exactly this shape inside a card; it is the card that was doing the framing, and it still is.

#### Order: label, then figure

The label sits above the value. In a row of tiles that keeps the labels on one line at the top and the figures on another below them, which survives labels of different lengths; the other order ("412" over "Attendance") makes the figures ragged the moment one label wraps to two lines. For a centered arrangement, pass core.AlignItemsProp(core.AlignItemsCenter) in Style — each line then shrinks to its content and centers.

#### The delta's zero value is neutral, not Primary

Everywhere else in this package VariantDefault resolves to the theme's Primary — that is what Badge and Button do, and what makes their zero value a no-op. Here it resolves to the secondary text ink instead, and the reason is that a delta is a measurement rather than a status. Whether a number going up is good is entirely the caller's domain: attendance up is success, expenses up is not, and latency up is an incident. A widget that guessed — by coloring on the sign, or by defaulting to the brand color as if any movement were noteworthy — would be confidently wrong on half the tiles a real screen carries. So the default says nothing, and a caller who knows what the movement means says it with DeltaVariant.

<small>[comps/stat_tile.go:45](https://github.com/rohanthewiz/grmob/blob/master/comps/stat_tile.go#L45)</small>

#### func (StatTile) Render

```go
func (s StatTile) Render(ctx *core.Context) *core.Node
```

<small>[comps/stat_tile.go:105](https://github.com/rohanthewiz/grmob/blob/master/comps/stat_tile.go#L105)</small>

### type StaticMap

```go
type StaticMap struct {
	// Lat and Lng are the point to show, in degrees.
	//
	// Latitude is clamped to the Web Mercator limit (±85.0511°) rather than to
	// ±90: every tile provider here projects with Mercator, where the poles
	// are at infinity, and a request past the limit comes back as an error
	// image rather than as a view of the Arctic. Longitude wraps instead of
	// clamping, because longitude is a circle — 190° is 170°W, and clamping it
	// to 180 would move the point rather than name it.
	Lat, Lng float64

	// Zoom is the slippy-tile zoom level both providers share: 0 is the whole
	// world, 19 is a building. 0 means DefaultMapZoom (street level), which is
	// the scale "where is this" is asking at.
	//
	// Zero cannot mean "the whole world" here, and that is the one place this
	// widget spends a legitimate value on a default. A world map centred on a
	// church is a picture of the Atlantic; a caller who genuinely wants zoom 0
	// has a provider of their own to ask.
	Zoom int

	// Width and Height are the image's size in px. 0 means
	// DefaultMapWidth/DefaultMapHeight, a card-width landscape panel.
	//
	// Both are clamped to MaxMapDimension, which is the smaller of what the
	// two bundled providers will serve. A request past it is refused by the
	// server, and a refusal arrives as a broken image with no error anywhere —
	// so the clamp is what turns "nothing rendered and nobody knows why" into
	// "a slightly smaller map than you asked for".
	Width, Height int

	// Scale is the device pixel ratio to ask the provider for: 1, 2 or 3.
	// 0 means DefaultMapScale, which is 1x.
	//
	// It changes the image, never the box: a Scale of 2 leaves the widget
	// exactly Width by Height logical pixels on screen and asks for four
	// times as many actual pixels to fill them. Clamped to MaxMapScale and
	// then spent, or ignored, by the provider — see "Device pixel ratio" on
	// the type for both halves of that.
	Scale int

	// Marker draws a pin at the point. Off by default: a map whose centre *is*
	// the subject does not always need one, and a single pin in the middle of
	// a small image can hide the very corner a reader is looking at.
	Marker bool

	// Label is what the place is called — "Lisbon Baptist Church", "The
	// rehearsal hall". It is the spoken name of the image, and it is handed to
	// the Handoff func, which may or may not have anywhere to put it: neither
	// built-in hand-off does (see GoogleMapsHandoff), and a platform-specific
	// one a caller writes can.
	//
	// Empty falls back to the coordinates, which is honest and reads badly —
	// "Map at 38.7223, -9.1393". Worth setting for that reason alone.
	Label string

	// Provider builds the image URL. Required: nil renders no image at all
	// and reports ConcernNoMapProvider in debug mode. See "The image is a
	// network fetch, and the provider is required" on the type for why this
	// stopped having a default.
	Provider StaticMapProvider

	// Handoff builds the URL a tap opens. nil means GoogleMapsHandoff.
	//
	// Setting it to a func that returns "" is how a caller says *no hand-off*:
	// core.OpenURL drops an empty URL, and this widget reads the same answer
	// one step earlier and renders a picture with no link role and no
	// callback, rather than a control that does nothing when tapped.
	Handoff MapHandoff

	// OnTap replaces the hand-off entirely — a caller who wants to push their
	// own map screen, open a sheet of directions, or log the tap first.
	//
	// It takes precedence over Handoff, and it does not compose with it: a
	// widget that both called back and opened an external app would leave the
	// caller no way to express either one alone. An OnTap that wants the
	// hand-off too can ask for it: core.OpenURL(comps.GoogleMapsHandoff(
	// lat, lng, label)).
	OnTap func()

	// Style is applied last, to the outer box, so a caller can give the map a
	// margin, a radius or a border without reaching inside it.
	Style []core.StyleProp

	// AccessibilityLabel overrides the spoken name. AccessibilityHint
	// overrides the "opens in your maps app" description a tappable map
	// carries; it is ignored when there is nothing to tap.
	AccessibilityLabel string
	AccessibilityHint  string
}
```

StaticMap is "where is this": a map image of one point, which hands off to the platform's own maps app when it is tapped.

	comps.StaticMap{
	    Lat: 38.7223, Lng: -9.1393,
	    Label:  "Lisbon Baptist Church",
	    Marker: true,
	}

	┌─────────────────────────┐
	│                         │   a core.Image of a rendered map,
	│           📍            │   fetched from a tile provider
	│                         │
	└─────────────────────────┘
	        tap ──▶ core.OpenURL ──▶ Google Maps / Apple Maps / a browser

#### Why an image and a hand-off rather than a map view

Because this answers the question an app actually asks. "Where is the church", "where is this event" wants a picture that says \*there\*, and then the user wants directions — which is a thing the platform's maps app does better than any embedded view could, with the user's own saved home address, their transport preferences and their downloaded tiles.

It is also the version that exists today on all four targets with no renderer work at all: a core.Image the reconciler already knows how to patch and a core.OpenURL the three hosts already hand to the system. A live panning, zooming map is a node type (MapKit, osmdroid, Leaflet — three host implementations and a marker protocol), which is a much larger feature and is the thing to build when an app needs to \*interact\* with a map rather than point at one.

#### The image is a network fetch, and the provider is required

Nothing here draws a map. The widget builds a URL and hands it to core.Image, so what arrives is whatever the provider serves — which makes the provider a decision an app has to own:

	GoogleStaticMap(key)  a paid API with an availability promise, and the
	                      only bundled provider whose host resolves.
	a func of your own    StaticMapProvider is one function; a provider this
	                      package has never heard of is five lines.
	OSMStaticMap          deprecated, and dead. See that function.

There is no default, and there used to be. OSMStaticMap was it, on the argument that a widget nobody can render without first buying something is a widget nobody evaluates — which was the right argument for as long as there was a keyless service to point at. There is not one now: every static-map service the OpenStreetMap wiki still lists takes a key, and the two keyless entries on that page are a web form and an HTML-embed generator, neither of which is a URL an image node can fetch.

So a StaticMap with no Provider renders its frame and no image, exactly as GoogleStaticMap("") does, and records ConcernNoMapProvider in debug mode. A build that has not chosen should look unfinished and say so, rather than draw a map of a host it cannot reach.

#### Why this cannot just draw the tiles itself

The obvious escape is to stop asking for a rendered image and compose one out of the raster tiles this repository already uses — tile.openstreetmap.org is keyless, alive, and is what core.MapView draws through on all three hosts. It is not available here. A tile grid centred on an arbitrary point needs its tiles placed at pixel offsets inside a clipped box, which is absolute positioning, and absolute positioning is the one Style.Position value with no Compose analog (see GrMobStyle.kt, which says so). A widget that laid out correctly on web and iOS and drifted on Android is worse than one that asks for a key.

#### Device pixel ratio

Width and Height are \*logical\* pixels: they set the size of the box on the screen. Scale is how many device pixels the provider should put inside each of them. Two numbers because they answer two questions — how big is this, and how sharp is it — and until Scale existed one number answered both: a 320px request stretched across a 2x phone's 640 real pixels, which arrives visibly soft and could not be said otherwise from the call site.

	Width: 320, Scale: 2    a 320-logical-px box holding a 640px image

Nothing reads the ratio off the device, because nothing in this framework knows it: there is no host event that reports screen metrics, on any of the three renderers. A caller who has the number states it; a caller who says nothing is choosing 1x, which is what every caller got before the field existed.

Whether the number can be spent is the provider's business. Google's API has a scale parameter that doubles the raster without touching the cartography; a provider with no such parameter ignores the field, which is why this is a StaticMapArea value rather than a multiply applied to Width before the provider sees it. The multiply would be wrong twice over: asking a map service for twice the pixels at the same zoom returns twice as much \*map\*, and asking at zoom+1 returns the same ground drawn for a deeper zoom, whose labels then land at half the physical size they were drawn for.

#### The hand-off is one URL for three platforms

Each platform has a scheme of its own — \`geo:\` on Android, \`maps://\` on iOS — and nothing in this framework knows which platform it is running on (deliberately: see core.OpenURL, which promises only the part that is portable). So the default Handoff is an https URL that all three resolve, and that on both phones reaches the installed maps app rather than a browser tab.

An app that does know its platform can say so in one line — \`Handoff: func(lat, lng float64, label string) string { return "geo:..." }\` — and an app that would rather not leave for Google has OpenStreetMapHandoff, which opens a browser everywhere.

#### Accessibility

A map image read by a screen reader is a rectangle with nothing in it: the meaning is in the arrangement, which is exactly the case core.RoleImg exists for (see its doc, and comps.Compass, which made the same argument about a compass rose). So the widget announces once and hides the image inside it.

When it is tappable the role is core.RoleLink rather than RoleButton, and the distinction is the one core.RoleLink's doc draws: a button does something here and a link goes somewhere else. Tapping this leaves the app entirely, which is as far as "somewhere else" goes, and a reader deciding whether to follow it deserves to know that before they do.

#### Zero is a place

Lat 0, Lng 0 is the Gulf of Guinea, and this widget draws it. There is no "unset" coordinate to detect — a float64 pair has no third state — so a caller whose location has not loaded yet must not render the widget at all, exactly as they would not render an EmptyState's action with no handler. comps.Skeleton is the placeholder for that gap.

<small>[comps/static_map.go:142](https://github.com/rohanthewiz/grmob/blob/master/comps/static_map.go#L142)</small>

#### func (StaticMap) Area

```go
func (m StaticMap) Area() StaticMapArea
```

Area resolves the caller's fields into the view a provider is handed: the defaults applied, the size clamped, the latitude clamped and the longitude wrapped. Everything downstream reads this rather than the struct, so there is one statement of what a zero means.

Exported because the answer is worth asking for from outside. A caller who wants to know what URL this widget will request — to log it, to pre-warm a cache, to show it in a tutorial — can ask the provider about this rather than re-deriving the defaults, which is the one way to get a second answer that disagrees.

<small>[comps/static_map.go:478](https://github.com/rohanthewiz/grmob/blob/master/comps/static_map.go#L478)</small>

#### func (StaticMap) Render

```go
func (m StaticMap) Render(ctx *core.Context) *core.Node
```

<small>[comps/static_map.go:583](https://github.com/rohanthewiz/grmob/blob/master/comps/static_map.go#L583)</small>

### type StaticMapArea

```go
type StaticMapArea struct {
	Lat, Lng      float64
	Zoom          int
	Width, Height int

	// Scale is the device pixel ratio asked for, already defaulted and
	// clamped. Width and Height stay logical: a provider that honours Scale
	// returns Width*Scale actual pixels for the same map, and a provider that
	// cannot honour it returns Width by Height and is not wrong to. See
	// "Device pixel ratio" on StaticMap.
	Scale int

	Marker bool
}
```

StaticMapArea is the view a provider is asked to render: the point, the scale, the pixel size, and whether to pin it.

A struct rather than five arguments because a provider is a function a caller writes, and a signature that grows is a signature that breaks every one of them. A field added here is a field an existing provider ignores.

The values arrive already defaulted and already clamped — a provider never sees a zero Zoom, a 4000px Width or a latitude off the end of Mercator — so every provider is spared the same four lines and none of them can disagree about what a zero means.

<small>[comps/static_map.go:298](https://github.com/rohanthewiz/grmob/blob/master/comps/static_map.go#L298)</small>

### type StaticMapProvider

```go
type StaticMapProvider func(StaticMapArea) string
```

StaticMapProvider turns a view into an image URL. See StaticMap.Provider.

<small>[comps/static_map.go:321](https://github.com/rohanthewiz/grmob/blob/master/comps/static_map.go#L321)</small>

#### func GoogleStaticMap

```go
func GoogleStaticMap(key string) StaticMapProvider
```

GoogleStaticMap renders through the Google Static Maps API with the given key, which the caller has obtained and is billed for.

A constructor rather than a bare provider because the key is the caller's: it is configuration, it differs per build, and a provider that read it out of a package variable would be a second place for a deployment to be wrong.

An empty key yields a provider that returns "", which renders the widget as a box with no image in it rather than as a map of Google's "this request is not authorized" error tile. A misconfigured build should look unfinished, not broken.

<small>[comps/static_map.go:380](https://github.com/rohanthewiz/grmob/blob/master/comps/static_map.go#L380)</small>

### type Stopwatch

```go
type Stopwatch struct {
	// Since is when the current run started. Read only while Running.
	Since time.Time

	// Elapsed is time banked from earlier runs, and the whole reading while
	// paused. Zero for a stopwatch that has never been paused.
	Elapsed time.Duration

	// Running says whether the current run is counting.
	Running bool

	// Format writes the digits. It receives the elapsed time already
	// truncated to a whole second. Nil uses the phone-timer format: M:SS
	// under an hour, H:MM:SS at or over one.
	Format func(time.Duration) string

	// Size is the digits' font size in px; 0 means 40, as in DigitalClock.
	Size float64

	// Color inks the digits; empty uses the theme's TextPrimary.
	Color string

	// Hidden removes the widget from display and stops its tick.
	Hidden bool

	// AccessibilityLabel overrides the spoken elapsed time.
	AccessibilityLabel string

	// Style is applied last, to the digits.
	Style []core.StyleProp
}
```

Stopwatch draws time elapsed, one tick a second, and can be paused and resumed without losing what it has counted.

	comps.Stopwatch{Since: startedAt.Get(), Elapsed: banked.Get(), Running: running.Get()}

It is Countdown counting the other way, and it is the simpler of the two: there is no deadline, so there is nothing to report and no OnDone.

#### The caller holds the two numbers, and why there are two

A stopwatch that knew only when it started could not be paused: the instant the finger lifts is not recorded anywhere, so on the next render the widget would either keep counting or forget everything. So the state is the pair every stopwatch keeps — the time banked from earlier runs, and the start of the current one — and the reading is their sum:

	Elapsed + (Running ? now − Since : 0)

which makes the four moves assignments the caller can write inline:

	start    Since = time.Now();  Elapsed = 0;                     Running = true
	pause    Elapsed += time.Since(Since);                         Running = false
	resume   Since = time.Now();                                   Running = true
	reset    Elapsed = 0;                                          Running = false

The pair lives with the caller rather than in the widget for the reason SliderRow's draft does: state held here would be state the app cannot save, restore or show anywhere else, and a stopwatch is exactly the thing an app wants to keep running across a screen change.

#### No hundredths

Real stopwatches show hundredths and this one shows seconds, because a core.State change requests a render of the whole tree: a centisecond stopwatch would charge the app a hundred render passes a second to animate two digits. A lap timer that genuinely needs them wants a renderer-side clock, which is what core.Spin is for animation and what no core primitive offers for text.

The reading is rounded \*down\*, the opposite of Countdown's rounding and for the same reason: a stopwatch never claims more elapsed time than has actually passed, so 0:00 covers the first second exactly as a phone's does.

#### Ticking

hooks.UseIntervalWhile again, active while Running and not Hidden. There is no exception for a hidden one, because unlike Countdown it owes nobody a callback — a hidden stopwatch has nothing to do but keep its arithmetic, which is the caller's two fields and needs no ticks at all. It therefore holds a hook and is not conditional-safe; see Countdown.

#### Theme roles read

	Digits   Colors.TextPrimary, unless Color says otherwise

<small>[comps/timers.go:278](https://github.com/rohanthewiz/grmob/blob/master/comps/timers.go#L278)</small>

#### func (Stopwatch) Render

```go
func (s Stopwatch) Render(ctx *core.Context) *core.Node
```

Render arms the tick and draws the digits.

<small>[comps/timers.go:311](https://github.com/rohanthewiz/grmob/blob/master/comps/timers.go#L311)</small>

