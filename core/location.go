package core

import (
	"math"
	"sync"
)

// Location is where the device is. It is the second sensor, and it is built on
// exactly the arrangement heading.go established — two one-way channels with a
// record in between, the start/stop pair reference counted:
//
//	StartLocation/StopLocation ──"sensor" system event──▶ host GPS
//	CurrentLocation ◀── location record ◀──"location" host event── host
//
// Each host maps the same two commands onto its own API:
//
//	Android   FusedLocationProvider, or LocationManager where Play Services
//	          is absent
//	iOS       CLLocationManager.startUpdatingLocation
//	Browser   navigator.geolocation.watchPosition
//	Headless  nothing (no system-event handler registered)
//
// # What this is not
//
// It is not the blue dot on a map. core.MapView's ShowUserLocation asks the
// *host's* map widget to draw the user's position, which every map SDK does
// from its own location plumbing and without telling Go where anybody is. The
// two features overlap in what they need from the OS (the same permission) and
// in nothing else: a screen can show the dot without Go ever holding a fix,
// and can hold a fix with no map on screen at all. Keeping them separate is
// what lets "centre the map on me" be an app decision rather than a map prop.
//
// # Permission, and why there is none of it here
//
// Location is the one sensor every platform gates. core.StartLocation asks the
// host to start, the host asks the OS, and a refusal comes back as
// `available: false` with a reason — the same shape a missing magnetometer
// arrives in, for the same reason: a sensor call is not the place to put a
// modal dialog.
//
// What *is* different from the compass is that the dialog is expected rather
// than exceptional, so a screen draws the permission's state beside the fix:
//
//	status := hooks.UsePermissionLive(ctx, permission.Location)
//	loc := hooks.UseLocation(ctx)          // unconditionally; it is a hook
//
//	switch status {
//	case permission.Granted: return readout(loc)
//	case permission.Prompt:  return askButton()  // Request from a tap
//	...
//	}
//
// Both hooks on every pass, and the branch on what is *drawn*. This example
// used to put UseLocation inside the Granted arm, which is a hook inside a
// conditional: hook slots are handed out by call position (core.NewState), so
// the arm turning on or off moves every slot after it and the component reads
// somebody else's state. core/debug.go's cursor audit reports it by name.
//
// permission.Request is the prompt and it must come from a gesture — see the
// permission package.
//
// # Which leaves when the dialog appears, and the answer is the route
//
// Calling the hook unconditionally means mounting the screen starts the
// sensor, and on iOS that is what shows the dialog. The gate is the
// *navigation*: a hook's reference is released when the route's frame is
// popped, so a screen the user navigated to is a screen the user asked for. A
// feature that must not ask until a tap belongs behind a button that navigates
// to it — the same mechanism, spelled with one more screen.
//
// A refusal is recoverable either way. The hosts keep a refused start open and
// re-arm it when the permission answer changes, so the grant does not have to
// arrive before the screen does; see LocationSensor.kt's awaitingPermission and
// LocationSensor.swift's awaitingAuthorization. What is still stale in that
// window is Error — it holds the refusal until the first fix lands.
//
// examples/tutorial lesson 4.12 runs all of this.
//
// # Accuracy is the field a location screen actually reads
//
// A fix with a 2000 metre radius is a fix of the city, and a map centred on it
// at building zoom is a lie told confidently. Every platform reports the
// radius; Location.Accuracy carries it in metres, and a screen that draws a
// position without drawing its uncertainty is the commonest mistake in this
// area.

// Location is one position fix.
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
	Available bool

	// Received is true once any location event has arrived. A first fix can
	// take tens of seconds on cold GPS, so this is the flag that separates
	// "still acquiring" — which is a spinner, and a long one — from "this
	// device will never tell you", which is a different screen.
	Received bool

	// Active is true while the sensor is running, which is core's own
	// reference count rather than the host's word: it flips on the Start call
	// rather than a round trip later.
	Active bool

	// Error is the host's message when it could not start or keep the sensor:
	// a permission refused, location services switched off system-wide, a
	// browser with no geolocation. Set alongside Available: false.
	Error string
}

// Speed and course are not here, and the omission is deliberate. Both are
// reported by all three platforms and both are about *travel* rather than
// position — they are meaningful only while moving, they are noisy to the point
// of uselessness while stationary, and the first consumer of either will be a
// navigation feature that also wants a smoothed track and a distance
// remaining. Adding them now would be three hosts filling in two fields that
// no screen reads, and the first real consumer would find the shape wrong.

// locationNotifyEpsilonMeters is how far the fix must move before subscribers
// are told.
//
// The record itself is updated on every event — CurrentLocation is always the
// newest fix — and only the *notification* is filtered, exactly as
// headingNotifyEpsilon does for the compass.
//
// One metre, which is about a pixel and a half at the deepest map zoom, and
// well inside any phone's accuracy. The reason a filter is needed at all is
// that a watchPosition on a moving device fires every second or so and a
// stationary one still jitters by a couple of metres as satellites come and
// go: a render pass per jitter, to move a marker by nothing, for the whole
// time a screen is open.
const locationNotifyEpsilonMeters = 1.0

// locationAccuracyEpsilonMeters is the same idea for the accuracy radius, and
// it is a filtered number here where Heading.Accuracy is a state transition
// there.
//
// The difference is in what the hosts report. A magnetometer's accuracy is a
// bucket that changes a handful of times in a session, so any change to it is
// news. A GPS accuracy is a float that drifts continuously — 8.0, 8.3, 7.9 —
// so treating every change as news would notify on every event and leave the
// distance filter above doing nothing at all.
const locationAccuracyEpsilonMeters = 1.0

// sensorKindLocation is the "kind" field of the "sensor" system event, beside
// sensorKindHeading. One event with a kind keeps each host's dispatch a single
// switch — which is the whole reason the compass spelled it that way before
// there was a second sensor to justify it.
const sensorKindLocation = "location"

// hostEventLocation is the name hosts report fixes under. See receiveLocation
// for the payload contract.
const hostEventLocation = "location"

var (
	locationMu       sync.RWMutex
	locationCurrent  Location
	locationRefs     int
	locationSubs     = map[int]func(Location){}
	locationNext     int
	locationNotified Location // the last value subscribers were told about
)

// StartLocation asks the host to begin reporting position fixes, and is
// balanced by StopLocation — the reference counting heading.go describes, and
// for the same reason twice over: GPS is the most expensive sensor on the
// device, and two screens that each want the user's position must each be able
// to let go without blinding the other.
//
// A host that needs an OS permission asks for one here if it has to, and a
// refusal arrives as an `available: false` event with a reason. An app that
// wants to control when that dialog appears asks first — see the type comment.
func StartLocation() {
	locationMu.Lock()
	locationRefs++
	first := locationRefs == 1
	if first {
		locationCurrent.Active = true
	}
	next := locationCurrent
	locationMu.Unlock()

	if first {
		SendSystemEvent("sensor", map[string]any{
			"command": "start",
			"kind":    sensorKindLocation,
		})
		notifyLocation(next, true)
	}
}

// StopLocation releases one Start. The sensor is turned off when the last
// holder lets go; extra Stops are ignored rather than driving the count
// negative.
func StopLocation() {
	locationMu.Lock()
	if locationRefs == 0 {
		locationMu.Unlock()
		return
	}
	locationRefs--
	last := locationRefs == 0
	if last {
		locationCurrent.Active = false
	}
	next := locationCurrent
	locationMu.Unlock()

	if last {
		SendSystemEvent("sensor", map[string]any{
			"command": "stop",
			"kind":    sensorKindLocation,
		})
		notifyLocation(next, true)
	}
}

// LocationActive reports whether the sensor is running. Mostly useful to tests
// and to a debug overlay; a screen wants Location.Active, which travels with
// the fix it belongs to.
func LocationActive() bool {
	locationMu.RLock()
	defer locationMu.RUnlock()
	return locationRefs > 0
}

// CurrentLocation returns the last fix, exactly as it arrived — the
// notification filter does not apply here. Safe from any goroutine.
func CurrentLocation() Location {
	locationMu.RLock()
	defer locationMu.RUnlock()
	return locationCurrent
}

// OnLocation subscribes fn to location changes. The returned function cancels
// the subscription.
//
// Subscribing does not start the sensor, for the reason OnHeading does not:
// wanting to *see* a position and wanting the GPS *on* are not always the same
// screen. hooks.UseLocation does both.
//
// fn runs on whichever goroutine delivered the fix — a host bridge call — and
// must not block; the usual body is a RequestRender.
func OnLocation(fn func(Location)) (cancel func()) {
	locationMu.Lock()
	id := locationNext
	locationNext++
	locationSubs[id] = fn
	locationMu.Unlock()
	return func() {
		locationMu.Lock()
		defer locationMu.Unlock()
		delete(locationSubs, id)
	}
}

// ReceiveLocation is the typed entry point for a host that builds the fix in Go
// (a test, an embedder). The JSON hosts arrive through
// ReceiveHostEvent("location", ...), which decodes into this.
//
// Coordinates are normalised here rather than trusted, which is this
// function's reason for existing beyond plumbing: it is the one place all four
// hosts funnel through, so it is the only place the invariant can be
// established. Latitude is clamped to ±90 and longitude wrapped into
// (-180, 180] — two rules, because they are two different facts about the
// sphere. A latitude past the pole is not a place; a longitude past the
// antimeridian is the same meridian spelled the long way round, and clamping
// it would move the device to the far side of the Pacific.
//
// Active is core's bookkeeping and is overwritten from the reference count
// rather than taken from the caller: a host does not know how many screens
// asked.
func ReceiveLocation(l Location) {
	l.Lat = clampLatitude(l.Lat)
	l.Lng = WrapLongitude(l.Lng)
	if !l.HasAltitude {
		l.Altitude = 0
	}
	if l.Accuracy < 0 {
		l.Accuracy = -1
	}
	if l.Available {
		l.Error = ""
	}
	l.Received = true

	locationMu.Lock()
	l.Active = locationRefs > 0
	locationCurrent = l
	locationMu.Unlock()

	notifyLocation(l, false)
}

// receiveLocation decodes the "location" host event. The payload keys are the
// contract every host writes:
//
//	lat        number   degrees, WGS-84
//	lng        number   degrees, WGS-84
//	accuracy   number   horizontal radius in metres; omitted when unknown
//	altitude   number   metres above the ellipsoid; omitted when unknown
//	available  bool     omitted means true — see below
//	error      string   why the sensor could not start or keep running
//	ts         number   host timestamp in ms; read by nobody in Go today
//
// `available` defaults to *true* when absent, the same asymmetry receiveHeading
// justifies: a host sending a fix has demonstrably got a sensor, and requiring
// three hosts to remember a boolean on every event is a contract that will be
// got wrong. A host that cannot produce fixes sends one event with
// available:false and a reason.
//
// A payload with no `lat`/`lng` at all decodes to 0,0 — which is a real place,
// and is why a host must never send a positionless fix as an ordinary event.
// The refusal shape is available:false with an error, where the coordinates are
// not read.
func receiveLocation(data map[string]any) {
	l := Location{Accuracy: -1, Available: true}
	l.Lat, _ = numberProp(data, "lat")
	l.Lng, _ = numberProp(data, "lng")
	if a, ok := numberProp(data, "accuracy"); ok {
		l.Accuracy = a
	}
	if alt, ok := numberProp(data, "altitude"); ok {
		l.Altitude, l.HasAltitude = alt, true
	}
	if av, ok := data["available"].(bool); ok {
		l.Available = av
	}
	l.Error, _ = data["error"].(string)
	ReceiveLocation(l)
}

// notifyLocation tells subscribers, unless the only thing that changed is a
// position that barely moved. force skips the filter, for the start/stop
// transitions where the position does not change at all but Active does.
//
// Subscribers run outside the lock so one may read CurrentLocation, subscribe
// or cancel from inside its own handler.
func notifyLocation(l Location, force bool) {
	locationMu.Lock()
	prev := locationNotified
	// The transitions a screen has to react to, as opposed to drift. Accuracy
	// is *not* in here — see locationAccuracyEpsilonMeters — but the appearance
	// or disappearance of an altitude is, because that is a field becoming
	// readable rather than a number changing.
	stateChanged := prev.Available != l.Available ||
		prev.Received != l.Received ||
		prev.Active != l.Active ||
		prev.HasAltitude != l.HasAltitude ||
		prev.Error != l.Error
	moved := DistanceMeters(prev.Lat, prev.Lng, l.Lat, l.Lng) >= locationNotifyEpsilonMeters ||
		math.Abs(prev.Accuracy-l.Accuracy) >= locationAccuracyEpsilonMeters
	if !force && !stateChanged && !moved {
		locationMu.Unlock()
		return
	}
	locationNotified = l
	fns := make([]func(Location), 0, len(locationSubs))
	for _, fn := range locationSubs {
		fns = append(fns, fn)
	}
	locationMu.Unlock()

	for _, fn := range fns {
		fn(l)
	}
}

// clampLatitude holds a latitude to the range that exists. Unexported because
// it is the trivial half of the pair and nothing outside core has asked; the
// longitude half is exported because wrapping is the part a caller gets wrong.
func clampLatitude(lat float64) float64 {
	if math.IsNaN(lat) {
		return 0
	}
	if lat > 90 {
		return 90
	}
	if lat < -90 {
		return -90
	}
	return lat
}

// WrapLongitude folds a longitude into (-180, 180] by going round rather than
// by stopping at the edge: 190° east is 170° west, the same meridian, and a
// clamp would move the point to the antimeridian instead.
//
// Exported because every consumer of a coordinate needs it and getting it
// wrong is silent — a map centred 20 degrees from where it was asked to be
// still looks like a map. components.StaticMap does the same arithmetic for the
// same reason.
//
// math.Mod keeps the sign of its first argument, so a negative input stays
// west, and the two adjustments are what carry a value past ±180 round to the
// other side.
func WrapLongitude(lng float64) float64 {
	if math.IsNaN(lng) || math.IsInf(lng, 0) {
		return 0
	}
	lng = math.Mod(lng, 360)
	if lng > 180 {
		lng -= 360
	}
	if lng <= -180 {
		lng += 360
	}
	return lng
}

// earthRadiusMeters is the mean radius, which is what a haversine over a
// sphere can use. The ellipsoid's equatorial and polar radii differ by 21km —
// about a third of a percent — so a distance from this function is accurate to
// roughly that, which is metres in a kilometre and is far inside any GPS fix's
// own uncertainty.
const earthRadiusMeters = 6371000.0

// DistanceMeters is the great-circle distance between two coordinates, by the
// haversine formula.
//
// Haversine rather than the flat approximation (scale the longitude by
// cos(lat), then Pythagoras) because the flat one is wrong in exactly the case
// a map application cares about: it degrades with latitude, and it breaks
// completely across the antimeridian, where two points a kilometre apart are
// 360 degrees of longitude apart on paper. The notification filter in this file
// is a consumer — a device crossing 180° must not be told it has travelled
// 40,000km — and so is any "within n metres of here" an app writes.
//
// Haversine rather than Vincenty, which is the next step up: Vincenty solves on
// the ellipsoid and is accurate to millimetres, at the cost of an iterative
// solver that fails to converge for antipodal points. A third of a percent is
// already well inside a GPS fix.
func DistanceMeters(lat1, lng1, lat2, lng2 float64) float64 {
	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	dφ := φ2 - φ1
	// The longitude delta goes through the wrap, which is what makes the
	// antimeridian ordinary: 179.9°E to 179.9°W is 0.2 degrees apart, not
	// 359.8.
	dλ := WrapLongitude(lng2-lng1) * math.Pi / 180

	h := math.Sin(dφ/2)*math.Sin(dφ/2) +
		math.Cos(φ1)*math.Cos(φ2)*math.Sin(dλ/2)*math.Sin(dλ/2)
	// Clamped before the arcsine: floating point can push h a hair above 1 for
	// antipodal points, and math.Asin of that is NaN, which would poison a
	// distance filter into never firing again.
	if h > 1 {
		h = 1
	}
	return 2 * earthRadiusMeters * math.Asin(math.Sqrt(h))
}

// resetLocationForTest returns the record, the reference count and the
// subscriptions to their initial state, so one test's sensor cannot leak into
// the next.
func resetLocationForTest() {
	locationMu.Lock()
	defer locationMu.Unlock()
	locationCurrent = Location{}
	locationNotified = Location{}
	locationRefs = 0
	locationSubs = map[int]func(Location){}
}
