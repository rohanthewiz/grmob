// Package permission asks the platform for the capabilities an app cannot
// simply take: the camera, the microphone, the user's location, the shared
// media store.
//
// # What kind of thing this is
//
// core draws three lines between the things that cross the app/host boundary,
// and a permission is a fourth that borrows from two of them:
//
//	a node        is reconciled — it has a place in the tree and a diff
//	a service     is commanded — AudioPlay(), OpenURL(), one-way and stateless
//	a sensor      is subscribed — it costs battery while on, reports until off
//	a permission  is *asked* — one question, one answer that outlives the ask
//
// The shape is core.Heading's, minus the reference counting: two one-way
// channels with a record in between, because there is no request/reply
// primitive on this bridge and inventing one for this would be a second way
// to do what the sensor already does.
//
//	Request/Check ──"permission" system event──▶ host authorization API
//	Current ◀── status record ◀──"permission" host event── host
//
// That choice is what keeps a permission out of the view tree. A screen reads
// Current(p) and re-renders when On fires, exactly as a compass screen reads
// CurrentHeading — so there is no callback to leak, no request id to
// correlate, and a prompt the user leaves standing for a minute cannot strand
// anything. hooks.UsePermission does both halves and is what most callers
// want.
//
// # Why this is not folded into the capability that needs it
//
// core.StartHeading makes the browser's motion prompt itself, and its doc says
// plainly that a separate permission API is one more thing to forget. That is
// the right rule for a capability whose permission moment *is* its start —
// a sensor, a camera preview mounting — and this package is not a second way
// to do it. Two things it cannot serve:
//
//	the rationale     a screen that wants to say "we will need your location,
//	                  and here is why" before the OS dialog appears. Android
//	                  has a whole API about this moment; asking mid-render is
//	                  how an app gets a permanent refusal.
//	the read-back     a settings screen showing "Location: denied — open
//	                  Settings". Check never prompts, so it is safe on mount,
//	                  and it is the only way to draw that row at all.
//
// There is a consumer for the second today, one file over from the compass:
// ios/GrMob/App/HeadingSensor.swift reports Heading.HasTrue only when location
// authorization has been granted, and its comment says an app that wants true
// north "asks for location authorization by its own route" — a route that,
// until this package had functions in it, did not exist.
//
// # The vocabulary is the browser's
//
// Status's spellings are the W3C Permissions API's own ("granted", "denied",
// "prompt"), for the reason core.Role's are ARIA's: the set has to be *some*
// published vocabulary, three of the four hosts would each need a mapping
// table whichever one was picked, and choosing the one a host already speaks
// means that host needs no table at all. iOS and Android each map their own
// richer enum onto it — see the per-host notes on Status's constants.
//
// # History
//
// This package existed for months as four Permission constants, three Status
// constants and two commented-out functions in Portuguese, left over from the
// govinci rebrand; they called a core.InvokeNative that is not in the
// codebase. Nothing imported it. The vocabulary below is that file's, with
// Pending replaced (see Prompt) and functions that exist.
package permission

import (
	"sync"

	"github.com/rohanthewiz/grmob/core"
)

// Permission is one capability the platform guards.
//
// The set is deliberately the four the original file named rather than every
// permission the three platforms have. A value here has to mean the same thing
// on all of them or the type is lying, and each of these four does; the
// per-constant notes say what each host actually asks for.
type Permission string

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

// Permissions returns every declared Permission, in declaration order.
//
// Pinned to the const block above by permission_enum_test.go and consumed by
// the host coverage checks, on the same footing as core.Roles(): a host that
// has no arm for a permission drops it silently, which on this bridge is
// indistinguishable from a user who has not answered yet.
func Permissions() []Permission {
	return []Permission{Camera, Location, Storage, Microphone}
}

// Status is what the platform says about one Permission.
type Status string

// PermissionStatus is the name this type had when the package was a
// vocabulary with no functions.
//
// Deprecated: use Status. permission.PermissionStatus stutters, and the alias
// costs one line where a rename would break an import that may exist outside
// this repository.
type PermissionStatus = Status

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
	// remember whether it has already asked. Both arrive here as Denied
	// rather than being guessed at — see the host notes in
	// docs/platforms/native.md.
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

// Statuses returns every status a host can report, in declaration order.
//
// Unknown is excluded for the reason core.RoleNone is excluded from Roles():
// it is the field's zero value rather than one of the answers, no host has an
// arm for it, and a coverage check that demanded one would be asking each
// platform to implement "we have not asked yet".
func Statuses() []Status {
	return []Status{Granted, Denied, Prompt, Unavailable}
}

// The two channel names and the two commands, which are the whole wire
// contract. Both hosts and both web runtimes switch on these strings.
const (
	systemEventName = "permission"
	hostEventName   = "permission"

	commandRequest = "request"
	commandCheck   = "check"
)

var (
	mu       sync.RWMutex
	statuses = map[Permission]Status{}
	subs     = map[int]func(Permission, Status){}
	nextSub  int
)

// init subscribes to the host's answers.
//
// Process-wide and at package init, matching core.OnHostEvent's own note on
// why its subscriptions are not scoped to a context tree: the thing on the far
// side is one physical device with one camera and one location, so there is no
// tree to hang the record on. Registering here rather than on first use means
// an answer that arrives before any screen has asked — a shell that reports
// what it already knows at startup — is still recorded.
func init() {
	core.OnHostEvent(hostEventName, receive)
}

// Request asks the platform to grant p, showing its dialog if there is one to
// show.
//
// It returns immediately and the answer arrives later, on the host's own
// goroutine, through the record and On. There is no callback parameter and no
// request id: see the package comment for why the answer is state rather than
// a reply.
//
// Call it from a user gesture. Every platform here either requires that (a
// browser will refuse a permission request made outside one) or punishes the
// alternative (a dialog a user did not expect is the one they refuse), and a
// permission refused once is much harder to get than one never asked for.
//
// Requesting something already Granted is harmless and re-reports the same
// status; requesting something Denied usually shows nothing at all, which is
// what Denied's doc is about.
func Request(p Permission) { send(commandRequest, p) }

// Check asks what the platform currently says about p, without prompting.
//
// Safe on mount and safe to repeat, which is what makes it the right call on
// a lifecycle change: a user can grant or revoke a permission in the system
// settings and come back, and nothing tells an app that happened. Re-checking
// when core.CurrentLifecycle returns to "active" is how a screen notices —
// hooks.UsePermission does not do it for you, because the hook cannot know
// whether the screen is still the one on top.
func Check(p Permission) { send(commandCheck, p) }

// send emits one command, or records Unavailable when nothing is listening.
//
// The short-circuit is the part worth explaining. core.SendSystemEvent drops
// events silently when no host has registered a handler — the right behaviour
// for a headless run, and the state a static export and every Go test are in.
// Without this, a screen in that state would sit on Unknown forever, drawing
// the spinner Unknown is for. "There is no platform to ask" is exactly what
// Unavailable means, so it is reported rather than waited for.
//
// A shell that *has* a handler but no arm for this event name is a case Go
// cannot see, and it stays on Unknown. That is the same silence every system
// event has on a shell that predates it, and it is why the host coverage
// checks in mobile/verify exist.
func send(command string, p Permission) {
	if !core.HasSystemEventHandler() {
		set(p, Unavailable)
		return
	}
	core.SendSystemEvent(systemEventName, map[string]any{
		"command": command,
		"kind":    string(p),
	})
}

// Current returns the last status recorded for p, or Unknown if none.
// Safe from any goroutine.
func Current(p Permission) Status {
	mu.RLock()
	defer mu.RUnlock()
	return statuses[p]
}

// Granted reports whether p is usable right now. Sugar for the comparison
// every call site would otherwise write, and named for the answer rather than
// the question so the condition reads as one.
//
// Note which way the unknown cases fall: only Granted is true, so a status
// that has not come back yet is treated as not-yet-usable rather than
// optimistically allowed. That is the safe direction — the alternative
// reaches for a camera the OS has not opened.
func IsGranted(p Permission) bool { return Current(p) == Granted }

// On subscribes fn to status changes. The returned function cancels the
// subscription.
//
// fn runs on whichever goroutine delivered the host event — a bridge call on
// the natives, the JS callback's goroutine in the browser — and must not
// block; the usual body is a core.RequestRender. Subscribing does not ask
// anything: On observes, Check and Request ask, and hooks.UsePermission does
// both because that is what a screen wants.
//
// Only *changes* notify. A host that answers a Check with the status already
// on record — which is every repeat check on an unchanged permission, and a
// lifecycle-driven re-check is mostly those — reaches here and stops, so the
// screen does not re-render for news that is not news.
func On(fn func(Permission, Status)) (cancel func()) {
	mu.Lock()
	id := nextSub
	nextSub++
	subs[id] = fn
	mu.Unlock()
	return func() {
		mu.Lock()
		defer mu.Unlock()
		delete(subs, id)
	}
}

// receive decodes the "permission" host event. The payload keys are the
// contract every host writes:
//
//	kind     string   the Permission this answers about
//	status   string   one of Statuses()
//
// Both are required and neither is guessed at. A missing or unrecognised kind
// has no record to write to; a missing or unrecognised status would otherwise
// have to become one of the four, and every choice is a lie — Granted opens a
// camera the OS did not, Denied hides a feature that works, Unavailable sends
// a user to a settings screen with nothing on it. So a malformed event is
// dropped and the permission stays where it was, which leaves the screen
// showing what it was showing.
//
// Unknown is refused as an incoming status for the same reason it is not in
// Statuses(): it means "we have not asked", and a host reporting it would be
// answering a question with the absence of an answer.
func receive(data map[string]any) {
	kind, _ := data["kind"].(string)
	raw, _ := data["status"].(string)
	if kind == "" || raw == "" {
		return
	}

	p := Permission(kind)
	if !valid(p) {
		return
	}
	status := Status(raw)
	if !reportable(status) {
		return
	}
	set(p, status)
}

// Receive is the typed entry point for a host that builds the answer in Go —
// a test, an embedder, a shell driving the tree directly. The JSON hosts
// arrive through core.ReceiveHostEvent("permission", ...), which decodes into
// this.
//
// It validates exactly as the JSON path does rather than trusting a typed
// caller: the two constants are strings, so a typed caller can produce the
// same nonsense a malformed payload can.
func Receive(p Permission, status Status) {
	if !valid(p) || !reportable(status) {
		return
	}
	set(p, status)
}

// set records a status and notifies, unless nothing changed. Subscribers run
// outside the lock so one may read Current, subscribe or cancel from inside
// its own handler.
func set(p Permission, status Status) {
	mu.Lock()
	if statuses[p] == status {
		mu.Unlock()
		return
	}
	statuses[p] = status
	fns := make([]func(Permission, Status), 0, len(subs))
	for _, fn := range subs {
		fns = append(fns, fn)
	}
	mu.Unlock()

	for _, fn := range fns {
		fn(p, status)
	}
}

func valid(p Permission) bool {
	for _, known := range Permissions() {
		if known == p {
			return true
		}
	}
	return false
}

// reportable rejects Unknown along with anything unspelled, which is the
// difference between this and a plain membership test — see receive.
func reportable(s Status) bool {
	for _, known := range Statuses() {
		if known == s {
			return true
		}
	}
	return false
}

// resetForTest returns the record and the subscriptions to their initial
// state, so one test's answers cannot leak into the next.
func resetForTest() {
	mu.Lock()
	defer mu.Unlock()
	statuses = map[Permission]Status{}
	subs = map[int]func(Permission, Status){}
}
