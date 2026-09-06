package core

import (
	"math"
	"sync"
)

// Heading is the device's compass bearing: which way the top of the phone is
// pointing. It is a *sensor*, and a sensor is a third kind of thing beside the
// two channels core already had.
//
//	a node        is reconciled — it has a place in the tree and a diff
//	a service     is commanded — AudioPlay(), OpenURL(), one-way and stateless
//	a sensor      is subscribed — it costs battery while on, reports until off
//
// The middle one, audio, is the closest existing relative and this file is
// modeled on it: two one-way channels with a record in between.
//
//	StartHeading/StopHeading ──"sensor" system event──▶ host magnetometer
//	CurrentHeading ◀── heading record ◀──"heading" host event── host
//
// What makes a sensor different from the player is the first line of the table:
// a magnetometer left running flattens a battery, so the interesting design
// question is not what the payload looks like but *who is allowed to turn it
// off*. See "Reference counting" below.
//
// Each host maps the same two commands onto its own sensor:
//
//	Android   SensorManager TYPE_ROTATION_VECTOR → getOrientation azimuth
//	iOS       CLLocationManager.startUpdatingHeading
//	Browser   deviceorientationabsolute, or webkitCompassHeading on Safari
//	Headless  nothing (no system-event handler registered)
//
// # Reference counting
//
// StartHeading and StopHeading are balanced, not idempotent: the "sensor"
// system event leaves only on the first Start and the last Stop, and the calls
// in between move a counter. Two screens can each ask for the heading and each
// give it back without either one's Stop blinding the other, which is exactly
// what hooks.UseHeading needs to be usable by more than one component at a
// time.
//
// The alternative — a plain on/off flag — was rejected for the failure it
// makes easy and silent: a tab bar showing a compass badge and a compass
// screen behind it both start the sensor, the screen is popped, its Stop turns
// the sensor off, and the badge quietly stops updating forever with nothing in
// any log. Refcounting is four lines and that bug cannot be written.
//
// An unbalanced Stop (more Stops than Starts) is clamped at zero rather than
// allowed to go negative, so a double-cleanup cannot leave the counter in a
// state where the next Start fails to actually start anything.
//
// # Availability
//
// A desktop browser has no magnetometer, and a phone whose sensor is broken or
// whose user refused the browser's motion prompt has one it cannot use. Both
// arrive as `available: false` in the host event, which is a different fact
// from "we have not heard anything yet" — the first says stop waiting and draw
// something else, the second says the reading is still in flight. Available
// therefore starts false and only a host event moves it, and Received is what
// distinguishes "no" from "not yet".

// Heading is one compass reading. Degrees increase clockwise, so 0 is north,
// 90 is east, 180 south, 270 west — the convention all three platform APIs and
// every paper compass share.
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

// Cardinal returns the 16-point compass abbreviation for the magnetic bearing
// — "N", "NNE", "NE", ... — which is what a compact readout shows beside (or
// instead of) the number.
//
// Sixteen points rather than eight or thirty-two: eight is coarse enough that
// a bearing can sit 22 degrees from the label naming it, and thirty-two needs
// four-letter names ("NbE") that no reader outside sailing recognises.
func (h Heading) Cardinal() string { return Cardinal(h.Magnetic) }

// compassPoints is the 16-point rose, indexed clockwise from north.
var compassPoints = [16]string{
	"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE",
	"S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW",
}

// Cardinal names any bearing in degrees, normalising it first so a caller can
// pass an unwrapped or negative angle.
//
// Each of the sixteen sectors is 22.5 degrees wide and *centred* on its point,
// which is the half worth stating: north is 348.75 through 11.25, not 0
// through 22.5, so a bearing one degree west of north reads "N" rather than
// "NNW". The +11.25 before the divide is what shifts the sector boundaries
// off the points and onto the gaps between them.
func Cardinal(deg float64) string {
	d := NormalizeDegrees(deg + 11.25)
	return compassPoints[int(d/22.5)%16]
}

// NormalizeDegrees folds any angle into [0, 360). Negative angles and angles
// past a full turn both come back on the circle, so -90 is 270 and 730 is 10.
//
// math.Mod alone is not enough: it keeps the sign of its first argument, so
// -90 comes back as -90 rather than 270.
func NormalizeDegrees(deg float64) float64 {
	if math.IsNaN(deg) || math.IsInf(deg, 0) {
		return 0
	}
	d := math.Mod(deg, 360)
	if d < 0 {
		d += 360
	}
	return d
}

// AngleDelta is the signed shortest turn from a to b, in (-180, 180]:
// positive clockwise, negative counter-clockwise.
//
// This is the arithmetic that makes a compass behave at the seam. Plain
// subtraction says the step from 359 degrees to 1 degree is -358, which is
// wrong by every measure that matters: it is a two-degree nudge, not most of a
// lap, and code that treats the difference as a magnitude (a change threshold,
// a smoothing filter, an animation) gets a spurious lurch once per rotation.
func AngleDelta(a, b float64) float64 {
	d := math.Mod(b-a, 360)
	if d > 180 {
		d -= 360
	}
	if d <= -180 {
		d += 360
	}
	return d
}

// headingNotifyEpsilon is how far the bearing must move before subscribers are
// told, in degrees.
//
// The hosts throttle to roughly 15 Hz, and a full render pass fifteen times a
// second to move a needle by a tenth of a degree is a real cost for a change
// no eye can see. The record itself is updated on every event — CurrentHeading
// is always the newest reading — and only the *notification* is filtered, so
// a screen re-rendering for any other reason still shows the exact value.
//
// Half a degree is under a pixel of arc at the edge of a 100pt dial, and well
// inside the accuracy of any phone magnetometer. Non-numeric changes
// (availability, an error, HasTrue appearing) always notify regardless: those
// are state transitions a screen must react to, not drift.
const headingNotifyEpsilon = 0.5

// sensorKindHeading is the "kind" field of the "sensor" system event. It is a
// field rather than a second event name because location and motion are the
// next two sensors on the roadmap and they want the same start/stop pair; one
// event with a kind keeps the host's dispatch a single switch.
const sensorKindHeading = "heading"

// hostEventHeading is the name hosts report readings under. See
// receiveHeading for the payload contract.
const hostEventHeading = "heading"

var (
	headingMu       sync.RWMutex
	headingCurrent  Heading
	headingRefs     int
	headingSubs     = map[int]func(Heading){}
	headingNext     int
	headingNotified Heading // the last value subscribers were told about
)

// StartHeading asks the host to begin reporting compass headings, and is
// balanced by StopHeading — see "Reference counting" in the package comment.
//
// On a browser this is also the permission moment: Safari's
// DeviceOrientationEvent.requestPermission must be called from inside a user
// gesture, and the host runtime makes that call here rather than exposing a
// separate permission API. A start that happens outside a gesture is refused
// by the browser and comes back as an "available: false" event carrying the
// reason, which is why Heading.Error exists.
func StartHeading() {
	headingMu.Lock()
	headingRefs++
	first := headingRefs == 1
	if first {
		headingCurrent.Active = true
	}
	next := headingCurrent
	headingMu.Unlock()

	if first {
		SendSystemEvent("sensor", map[string]any{
			"command": "start",
			"kind":    sensorKindHeading,
		})
		notifyHeading(next, true)
	}
}

// StopHeading releases one Start. The sensor is turned off when the last
// holder lets go; extra Stops are ignored rather than driving the count
// negative.
func StopHeading() {
	headingMu.Lock()
	if headingRefs == 0 {
		headingMu.Unlock()
		return
	}
	headingRefs--
	last := headingRefs == 0
	if last {
		headingCurrent.Active = false
	}
	next := headingCurrent
	headingMu.Unlock()

	if last {
		SendSystemEvent("sensor", map[string]any{
			"command": "stop",
			"kind":    sensorKindHeading,
		})
		notifyHeading(next, true)
	}
}

// HeadingActive reports whether the sensor is running. Mostly useful to
// tests and to a debug overlay; a screen wants Heading.Active, which travels
// with the reading it belongs to.
func HeadingActive() bool {
	headingMu.RLock()
	defer headingMu.RUnlock()
	return headingRefs > 0
}

// CurrentHeading returns the last reading, exactly as it arrived — the
// notification filter described on headingNotifyEpsilon does not apply here.
// Safe from any goroutine.
func CurrentHeading() Heading {
	headingMu.RLock()
	defer headingMu.RUnlock()
	return headingCurrent
}

// OnHeading subscribes fn to heading changes. The returned function cancels
// the subscription.
//
// Subscribing does not start the sensor: the two are separate on purpose,
// because a screen that wants to *display* a heading and a screen that wants
// the sensor *on* are not always the same screen (a background service, a
// second view of one reading). hooks.UseHeading does both, which is what most
// callers want.
//
// fn runs on whichever goroutine delivered the reading — a host bridge call —
// and must not block; the usual body is a RequestRender.
func OnHeading(fn func(Heading)) (cancel func()) {
	headingMu.Lock()
	id := headingNext
	headingNext++
	headingSubs[id] = fn
	headingMu.Unlock()
	return func() {
		headingMu.Lock()
		defer headingMu.Unlock()
		delete(headingSubs, id)
	}
}

// ReceiveHeading is the typed entry point for a host that builds the reading
// in Go (a test, an embedder). The JSON hosts arrive through
// ReceiveHostEvent("heading", ...) instead, which decodes into this.
//
// Bearings are normalised here rather than trusted: a host that reports 360.0
// at north, or a negative azimuth (which Android's getOrientation returns for
// half the circle — it answers in radians over -pi..pi), would otherwise leak
// an out-of-range angle into every consumer's arithmetic. This is the one
// place all four hosts funnel through, so it is the one place the invariant
// "Magnetic is in [0, 360)" can actually be established.
//
// Active is core's bookkeeping and is overwritten from the reference count, not
// taken from the caller: a host does not know how many screens asked.
func ReceiveHeading(h Heading) {
	h.Magnetic = NormalizeDegrees(h.Magnetic)
	if h.HasTrue {
		h.True = NormalizeDegrees(h.True)
	} else {
		h.True = 0
	}
	if h.Accuracy < 0 {
		h.Accuracy = -1
	}
	if h.Available {
		h.Error = ""
	}
	h.Received = true

	headingMu.Lock()
	h.Active = headingRefs > 0
	headingCurrent = h
	headingMu.Unlock()

	notifyHeading(h, false)
}

// receiveHeading decodes the "heading" host event. The payload keys are the
// contract every host writes:
//
//	magnetic   number   degrees clockwise from magnetic north
//	true       number   degrees from true north; omitted when unknown
//	accuracy   number   error margin in degrees; omitted when unknown
//	available  bool     omitted means true — see below
//	error      string   why the sensor could not start
//	ts         number   host timestamp in ms; read by nobody in Go today
//
// `available` defaults to *true* when absent, which is the one asymmetry worth
// justifying. Every other missing field means "I do not know", but a host that
// is sending a reading at all has demonstrably got a sensor, and requiring
// three hosts to remember a boolean on every one of fifteen events a second is
// a contract that will be got wrong. The hosts that cannot produce headings
// send exactly one event, with available:false and a reason.
//
// `ts` is accepted and ignored. Hosts already have it (all three sensor APIs
// hand one over), the smoothing that would use it lives on the host side of
// the throttle, and a field that is dropped here is cheaper to start reading
// later than one the hosts never learned to send.
func receiveHeading(data map[string]any) {
	h := Heading{Accuracy: -1, Available: true}
	h.Magnetic, _ = numberProp(data, "magnetic")
	if t, ok := numberProp(data, "true"); ok {
		h.True, h.HasTrue = t, true
	}
	if a, ok := numberProp(data, "accuracy"); ok {
		h.Accuracy = a
	}
	if av, ok := data["available"].(bool); ok {
		h.Available = av
	}
	h.Error, _ = data["error"].(string)
	ReceiveHeading(h)
}

// notifyHeading tells subscribers, unless the only thing that changed is a
// bearing that barely moved. force skips the filter, for the start/stop
// transitions where the bearing does not change at all but Active does.
//
// Subscribers run outside the lock so one may read CurrentHeading, subscribe
// or cancel from inside its own handler.
func notifyHeading(h Heading, force bool) {
	headingMu.Lock()
	prev := headingNotified
	// Everything except the two bearings is a state transition rather than
	// drift, so any change to one always gets through.
	stateChanged := prev.Available != h.Available ||
		prev.Received != h.Received ||
		prev.Active != h.Active ||
		prev.HasTrue != h.HasTrue ||
		prev.Error != h.Error ||
		prev.Accuracy != h.Accuracy
	moved := math.Abs(AngleDelta(prev.Magnetic, h.Magnetic)) >= headingNotifyEpsilon ||
		(h.HasTrue && math.Abs(AngleDelta(prev.True, h.True)) >= headingNotifyEpsilon)
	if !force && !stateChanged && !moved {
		headingMu.Unlock()
		return
	}
	headingNotified = h
	fns := make([]func(Heading), 0, len(headingSubs))
	for _, fn := range headingSubs {
		fns = append(fns, fn)
	}
	headingMu.Unlock()

	for _, fn := range fns {
		fn(h)
	}
}

// resetHeadingForTest returns the record, the reference count and the
// subscriptions to their initial state, so one test's sensor cannot leak into
// the next.
func resetHeadingForTest() {
	headingMu.Lock()
	defer headingMu.Unlock()
	headingCurrent = Heading{}
	headingNotified = Heading{}
	headingRefs = 0
	headingSubs = map[int]func(Heading){}
}
