# Package permission

```go
import "github.com/rohanthewiz/grmob/permission"
```

Package permission asks the platform for the capabilities an app cannot simply take: the camera, the microphone, the user's location, the shared media store.

## What kind of thing this is

core draws three lines between the things that cross the app/host boundary, and a permission is a fourth that borrows from two of them:

	a node        is reconciled — it has a place in the tree and a diff
	a service     is commanded — AudioPlay(), OpenURL(), one-way and stateless
	a sensor      is subscribed — it costs battery while on, reports until off
	a permission  is *asked* — one question, one answer that outlives the ask

The shape is core.Heading's, minus the reference counting: two one-way channels with a record in between, because there is no request/reply primitive on this bridge and inventing one for this would be a second way to do what the sensor already does.

	Request/Check ──"permission" system event──▶ host authorization API
	Current ◀── status record ◀──"permission" host event── host

That choice is what keeps a permission out of the view tree. A screen reads Current(p) and re-renders when On fires, exactly as a compass screen reads CurrentHeading — so there is no callback to leak, no request id to correlate, and a prompt the user leaves standing for a minute cannot strand anything. hooks.UsePermission does both halves and is what most callers want.

## Why this is not folded into the capability that needs it

core.StartHeading makes the browser's motion prompt itself, and its doc says plainly that a separate permission API is one more thing to forget. That is the right rule for a capability whose permission moment \*is\* its start — a sensor, a camera preview mounting — and this package is not a second way to do it. Two things it cannot serve:

	the rationale     a screen that wants to say "we will need your location,
	                  and here is why" before the OS dialog appears. Android
	                  has a whole API about this moment; asking mid-render is
	                  how an app gets a permanent refusal.
	the read-back     a settings screen showing "Location: denied — open
	                  Settings". Check never prompts, so it is safe on mount,
	                  and it is the only way to draw that row at all.
	the return        and the moment after that row's button worked: the user
	                  granted it in Settings and came back, which no platform
	                  announces. foreground.go is that half; the capability
	                  that makes its own prompt has nowhere to hang it.

There is a consumer for the second today, one file over from the compass: ios/GrMob/App/HeadingSensor.swift reports Heading.HasTrue only when location authorization has been granted, and its comment says an app that wants true north "asks for location authorization by its own route" — a route that, until this package had functions in it, did not exist.

## The vocabulary is the browser's

Status's spellings are the W3C Permissions API's own ("granted", "denied", "prompt"), for the reason core.Role's are ARIA's: the set has to be \*some\* published vocabulary, three of the four hosts would each need a mapping table whichever one was picked, and choosing the one a host already speaks means that host needs no table at all. iOS and Android each map their own richer enum onto it — see the per-host notes on Status's constants.

## History

This package existed for months as four Permission constants, three Status constants and two commented-out functions in Portuguese, left over from the govinci rebrand; they called a core.InvokeNative that is not in the codebase. Nothing imported it. The vocabulary below is that file's, with Pending replaced (see Prompt) and functions that exist.

## Index

- [`func Check`](#func-check)
- [`func IsGranted`](#func-isgranted)
- [`func On`](#func-on)
- [`func Receive`](#func-receive)
- [`func Request`](#func-request)
- [`func WatchForeground`](#func-watchforeground)
- [`func WatchedForeground`](#func-watchedforeground)
- [`type Permission`](#type-permission)
    - [`func Permissions`](#func-permissions)
- [`type PermissionStatus`](#type-permissionstatus)
- [`type Status`](#type-status)
    - [`func Current`](#func-current)
    - [`func Statuses`](#func-statuses)

## Functions

### func Check

```go
func Check(p Permission)
```

Check asks what the platform currently says about p, without prompting.

Safe on mount and safe to repeat, which is what makes it the right call on a lifecycle change: a user can grant or revoke a permission in the system settings and come back, and nothing tells an app that happened. Re-checking when core.CurrentLifecycle returns to "active" is how a screen notices, and WatchForeground is that arrangement written once — one check per permission per resume however many screens are watching. hooks.UsePermissionLive is this and the mount check together, and is what most callers want.

<small>[permission/permission.go:267](https://github.com/rohanthewiz/grmob/blob/master/permission/permission.go#L267)</small>

### func IsGranted

```go
func IsGranted(p Permission) bool
```

Granted reports whether p is usable right now. Sugar for the comparison every call site would otherwise write, and named for the answer rather than the question so the condition reads as one.

Note which way the unknown cases fall: only Granted is true, so a status that has not come back yet is treated as not-yet-usable rather than optimistically allowed. That is the safe direction — the alternative reaches for a camera the OS has not opened.

<small>[permission/permission.go:309](https://github.com/rohanthewiz/grmob/blob/master/permission/permission.go#L309)</small>

### func On

```go
func On(fn func(Permission, Status)) (cancel func())
```

On subscribes fn to status changes. The returned function cancels the subscription.

fn runs on whichever goroutine delivered the host event — a bridge call on the natives, the JS callback's goroutine in the browser — and must not block; the usual body is a core.RequestRender. Subscribing does not ask anything: On observes, Check and Request ask, and hooks.UsePermission does both because that is what a screen wants.

Only \*changes\* notify. A host that answers a Check with the status already on record — which is every repeat check on an unchanged permission, and a lifecycle-driven re-check is mostly those — reaches here and stops, so the screen does not re-render for news that is not news.

<small>[permission/permission.go:324](https://github.com/rohanthewiz/grmob/blob/master/permission/permission.go#L324)</small>

### func Receive

```go
func Receive(p Permission, status Status)
```

Receive is the typed entry point for a host that builds the answer in Go — a test, an embedder, a shell driving the tree directly. The JSON hosts arrive through core.ReceiveHostEvent("permission", ...), which decodes into this.

It validates exactly as the JSON path does rather than trusting a typed caller: the two constants are strings, so a typed caller can produce the same nonsense a malformed payload can.

<small>[permission/permission.go:380](https://github.com/rohanthewiz/grmob/blob/master/permission/permission.go#L380)</small>

### func Request

```go
func Request(p Permission)
```

Request asks the platform to grant p, showing its dialog if there is one to show.

It returns immediately and the answer arrives later, on the host's own goroutine, through the record and On. There is no callback parameter and no request id: see the package comment for why the answer is state rather than a reply.

Call it from a user gesture. Every platform here either requires that (a browser will refuse a permission request made outside one) or punishes the alternative (a dialog a user did not expect is the one they refuse), and a permission refused once is much harder to get than one never asked for.

Requesting something already Granted is harmless and re-reports the same status; requesting something Denied usually shows nothing at all, which is what Denied's doc is about.

<small>[permission/permission.go:256](https://github.com/rohanthewiz/grmob/blob/master/permission/permission.go#L256)</small>

### func WatchForeground

```go
func WatchForeground(p Permission) (cancel func())
```

WatchForeground asks for p to be re-checked every time the app returns to the foreground, and returns the function that stops asking.

Balanced rather than idempotent, exactly as core.StartHeading/StopHeading are and for the same failure: two screens both watching the camera, one of them closing, and a plain on/off flag turning the re-check off under the other one with nothing in any log. Calling the returned cancel more than once is harmless — the second call finds the watch already released and does nothing, so a component that both defers a cancel and calls it on an error path cannot drive the count negative.

It does not check anything itself. A watcher registering says what should happen on the \*next\* resume; the opening check is the caller's, which for almost every caller means hooks.UsePermission doing it on mount. hooks.UsePermissionLive is the two together.

Safe from any goroutine.

<small>[permission/foreground.go:101](https://github.com/rohanthewiz/grmob/blob/master/permission/foreground.go#L101)</small>

### func WatchedForeground

```go
func WatchedForeground(p Permission) int
```

WatchedForeground reports how many watchers p has. Bookkeeping for tests and a debug overlay, the same job core.HeadingActive does for the sensor; nothing in an app should need it.

<small>[permission/foreground.go:156](https://github.com/rohanthewiz/grmob/blob/master/permission/foreground.go#L156)</small>

## Types

### type Permission

```go
type Permission string
```

Permission is one capability the platform guards.

The set is deliberately the four the original file named rather than every permission the three platforms have. A value here has to mean the same thing on all of them or the type is lying, and each of these four does; the per-constant notes say what each host actually asks for.

<small>[permission/permission.go:86](https://github.com/rohanthewiz/grmob/blob/master/permission/permission.go#L86)</small>

```go
const (
	// Camera is the still/video capture device.
	//
	//	iOS       AVCaptureDevice.requestAccess(for: .video)
	//	Android   Manifest.permission.CAMERA
	//	Browser   the "camera" descriptor
	Camera Permission = "camera"

	// Location is the device's position. Coarse or fine is the *host's*
	// choice and not a second constant, because the two platforms that draw
	// the distinction draw it differently — iOS has when-in-use and always,
	// Android has coarse and fine — and a value that means different things
	// on two targets is worse than one that means the least common thing on
	// all of them. Each host asks for the narrower of its options.
	//
	//	iOS       CLLocationManager.requestWhenInUseAuthorization()
	//	Android   Manifest.permission.ACCESS_COARSE_LOCATION
	//	Browser   the "geolocation" descriptor
	Location Permission = "location"

	// Storage is the shared media store — the user's photos and videos, not
	// the app's own sandbox, which needs no permission anywhere.
	//
	//	iOS       PHPhotoLibrary.requestAuthorization(for: .readWrite)
	//	Android   READ_MEDIA_IMAGES on 13+, READ_EXTERNAL_STORAGE below it
	//	Browser   Unavailable. A page reaches a file through a file input or
	//	          the file-system access API, both of which are a user gesture
	//	          rather than a permission, so there is nothing to ask.
	Storage Permission = "storage"

	// Microphone is audio capture.
	//
	//	iOS       AVAudioApplication.requestRecordPermission
	//	Android   Manifest.permission.RECORD_AUDIO
	//	Browser   the "microphone" descriptor
	Microphone Permission = "microphone"
)
```

#### func Permissions

```go
func Permissions() []Permission
```

Permissions returns every declared Permission, in declaration order.

Pinned to the const block above by permission\_enum\_test.go and consumed by the host coverage checks, on the same footing as core.Roles(): a host that has no arm for a permission drops it silently, which on this bridge is indistinguishable from a user who has not answered yet.

<small>[permission/permission.go:132](https://github.com/rohanthewiz/grmob/blob/master/permission/permission.go#L132)</small>

### type PermissionStatus

```go
type PermissionStatus = Status
```

PermissionStatus is the name this type had when the package was a vocabulary with no functions.

Deprecated: use Status. permission.PermissionStatus stutters, and the alias costs one line where a rename would break an import that may exist outside this repository.

<small>[permission/permission.go:145](https://github.com/rohanthewiz/grmob/blob/master/permission/permission.go#L145)</small>

### type Status

```go
type Status string
```

Status is what the platform says about one Permission.

<small>[permission/permission.go:137](https://github.com/rohanthewiz/grmob/blob/master/permission/permission.go#L137)</small>

```go
const (
	// Unknown is the zero value: nobody has asked yet, or the answer has not
	// come back. It is not a platform state — no host ever reports it — and
	// it exists for the reason core.Heading.Received does: "we have not heard"
	// and "the platform says no" are different facts, and a screen draws
	// different things for them. A spinner is right for one and wrong for the
	// other.
	Unknown Status = ""

	// Granted — the app may use the capability now.
	//
	//	iOS       .authorized / .authorizedWhenInUse / .authorizedAlways
	//	Android   PackageManager.PERMISSION_GRANTED
	//	Browser   "granted"
	Granted Status = "granted"

	// Denied — the user said no. Calling Request again is not an error and is
	// usually not useful: iOS never re-prompts for a permission it has been
	// refused once, and a browser that has recorded a denial answers the next
	// request from its own store without showing anything. The move that
	// works from here is to send the user to the platform's settings, which
	// is why a screen wants to be able to tell this apart from Prompt.
	//
	// Android is the one host where the state is genuinely coarser than the
	// word: a refusal that will still re-prompt and one that will not are the
	// same PERMISSION_DENIED, and telling them apart needs the app to
	// remember whether it has already asked. The shell remembers it, in its
	// own preferences and across restarts, because the platform will not — so
	// both arrive here as Denied rather than being guessed at, and a cold
	// start no longer reports a permanent refusal as Prompt. See the host
	// notes in docs/platforms/native.md.
	Denied Status = "denied"

	// Prompt — undecided. Request will show the platform's dialog, which is
	// the one state where asking is worth doing and the one moment a
	// rationale belongs in front of.
	//
	//	iOS       .notDetermined
	//	Android   PERMISSION_DENIED with no prior request recorded
	//	Browser   "prompt"
	Prompt Status = "prompt"

	// Unavailable — this device or platform cannot grant it at all, so asking
	// will never do anything. An iPad with no camera, an iOS restriction set
	// by a parental control, an Android permission the manifest never
	// declared, a browser that has no descriptor for it, and — the case every
	// host shares — an app running with no host attached at all.
	//
	// It is distinct from Denied because the remedy is: a denial is fixed in
	// the platform's settings, and this is not fixable. A screen that offers
	// "Open Settings" for both sends the user somewhere with no switch on it.
	Unavailable Status = "unavailable"
)
```

#### func Current

```go
func Current(p Permission) Status
```

Current returns the last status recorded for p, or Unknown if none. Safe from any goroutine.

<small>[permission/permission.go:295](https://github.com/rohanthewiz/grmob/blob/master/permission/permission.go#L295)</small>

#### func Statuses

```go
func Statuses() []Status
```

Statuses returns every status a host can report, in declaration order.

Unknown is excluded for the reason core.RoleNone is excluded from Roles(): it is the field's zero value rather than one of the answers, no host has an arm for it, and a coverage check that demanded one would be asking each platform to implement "we have not asked yet".

<small>[permission/permission.go:207](https://github.com/rohanthewiz/grmob/blob/master/permission/permission.go#L207)</small>

