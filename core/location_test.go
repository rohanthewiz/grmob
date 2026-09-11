package core

import (
	"math"
	"testing"
)

// recordLocationEvents installs a recording system-event handler and resets the
// location record, so one test's sensor never leaks into the next (both are
// package-level). The heading twin is recordSensorEvents next door; this one
// filters to the location kind, because both sensors share the event name.
func recordLocationEvents(t *testing.T) *[]map[string]any {
	t.Helper()
	resetLocationForTest()
	var events []map[string]any
	SetSystemEventHandler(func(name string, data map[string]any) {
		if name == "sensor" && data["kind"] == sensorKindLocation {
			events = append(events, data)
		}
	})
	t.Cleanup(func() {
		SetSystemEventHandler(nil)
		resetLocationForTest()
	})
	return &events
}

// Longitude wraps and latitude clamps, which are two different facts about a
// sphere. A single "normalise" doing one of them twice is the bug this pair of
// rules exists to prevent.
func TestWrapLongitudeGoesRoundRatherThanStopping(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{0, 0}, {179.9, 179.9}, {-179.9, -179.9},
		{180, 180},  // the antimeridian itself is a longitude
		{-180, 180}, // and its other spelling folds onto the same meridian
		{190, -170}, // past it, round to the western side
		{-190, 170}, // and the other way
		{540, 180},  // a lap and a half
		{-540, 180}, // the same, westward
		{361, 1},    // just past a lap
	}
	for _, c := range cases {
		if got := WrapLongitude(c.in); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("WrapLongitude(%g) = %g, want %g", c.in, got, c.want)
		}
	}
	// A host dividing by zero upstream must not put a NaN into every
	// consumer's arithmetic — the same floor NormalizeDegrees has.
	if got := WrapLongitude(math.NaN()); got != 0 {
		t.Errorf("WrapLongitude(NaN) = %g, want 0", got)
	}
	if got := WrapLongitude(math.Inf(-1)); got != 0 {
		t.Errorf("WrapLongitude(-Inf) = %g, want 0", got)
	}
}

// The distance function, and the one case that decides whether it is haversine
// or the flat approximation: a step across the antimeridian.
func TestDistanceMetersIsAGreatCircleAndSurvivesTheAntimeridian(t *testing.T) {
	// A degree of latitude is about 111.2km anywhere, which is the easy
	// sanity check.
	if d := DistanceMeters(0, 0, 1, 0); math.Abs(d-111195) > 500 {
		t.Errorf("one degree of latitude = %.0fm, want ~111195m", d)
	}
	// A degree of longitude shrinks with latitude — the fact a flat
	// approximation has to be told and a haversine knows.
	equator := DistanceMeters(0, 0, 0, 1)
	sixty := DistanceMeters(60, 0, 60, 1)
	if math.Abs(sixty-equator/2) > 500 {
		t.Errorf("a degree of longitude at 60°N = %.0fm, want about half the equator's %.0fm",
			sixty, equator)
	}
	// The seam. 179.95°E to 179.95°W is a tenth of a degree apart, not 359.9.
	if d := DistanceMeters(0, 179.95, 0, -179.95); d > 12000 {
		t.Errorf("across the antimeridian = %.0fm, want about 11km — a flat "+
			"approximation reports most of the planet here", d)
	}
	// Antipodal points are half the circumference and must not come back NaN:
	// floating point can push the haversine term above 1, and an Asin of that
	// poisons every later comparison in the notification filter.
	d := DistanceMeters(0, 0, 0, 180)
	if math.IsNaN(d) {
		t.Fatal("antipodal distance is NaN")
	}
	if math.Abs(d-20015000) > 50000 {
		t.Errorf("antipodal distance = %.0fm, want about 20015km", d)
	}
	if got := DistanceMeters(10, 20, 10, 20); got != 0 {
		t.Errorf("a point from itself = %g, want 0", got)
	}
}

// Start/stop is refcounted, not toggled — the same contract the compass has and
// for a sharper reason: GPS is the most expensive sensor on the device, so a
// second screen's Stop must not blind the first.
func TestLocationStartStopIsRefcounted(t *testing.T) {
	events := recordLocationEvents(t)

	StartLocation()
	StartLocation()
	if len(*events) != 1 {
		t.Fatalf("two Starts sent %d events, want 1", len(*events))
	}
	if (*events)[0]["command"] != "start" || (*events)[0]["kind"] != "location" {
		t.Errorf("first event = %v", (*events)[0])
	}
	if !LocationActive() {
		t.Error("the sensor should be active")
	}

	StopLocation()
	if len(*events) != 1 {
		t.Fatalf("the first Stop of two sent an event: %v", *events)
	}
	if !LocationActive() {
		t.Error("one holder let go; the other still wants the sensor")
	}

	StopLocation()
	if len(*events) != 2 || (*events)[1]["command"] != "stop" {
		t.Fatalf("the last Stop did not stop the sensor: %v", *events)
	}
	if LocationActive() {
		t.Error("nobody holds a reference; the sensor should be off")
	}

	// An unbalanced Stop is clamped rather than driving the count negative: a
	// double cleanup must not leave the next Start unable to start anything.
	StopLocation()
	StopLocation()
	if len(*events) != 2 {
		t.Errorf("extra Stops sent events: %v", *events)
	}
	StartLocation()
	if len(*events) != 3 || (*events)[2]["command"] != "start" {
		t.Errorf("the Start after the extra Stops did not reach the host: %v", *events)
	}
}

// The decode path's contract, which is the one place all four hosts funnel
// through and therefore the only place the coordinate invariants can be
// established.
func TestReceiveLocationNormalisesAndFillsTheRecord(t *testing.T) {
	recordLocationEvents(t)

	ReceiveHostEvent("location", map[string]any{
		"lat": 91.0, "lng": 190.0,
		"accuracy": 12.5,
		"altitude": 84.0,
		"ts":       1234.0,
	})
	got := CurrentLocation()
	if got.Lat != 90 {
		t.Errorf("lat = %g, want clamped to 90", got.Lat)
	}
	if got.Lng != -170 {
		t.Errorf("lng = %g, want wrapped to -170", got.Lng)
	}
	if got.Accuracy != 12.5 {
		t.Errorf("accuracy = %g", got.Accuracy)
	}
	if !got.HasAltitude || got.Altitude != 84 {
		t.Errorf("altitude = %g (has=%v)", got.Altitude, got.HasAltitude)
	}
	if !got.Received {
		t.Error("Received should be true once any event has arrived")
	}
	// available is absent, which means available — see receiveLocation.
	if !got.Available {
		t.Error("an absent `available` must read as available")
	}
	if got.Active {
		t.Error("nobody started the sensor; Active is core's own bookkeeping")
	}
}

// A fix with no altitude key leaves the field at zero AND says so, because 0 is
// most of the world's coastline.
func TestAnAbsentAltitudeIsNotZeroMetres(t *testing.T) {
	recordLocationEvents(t)
	ReceiveHostEvent("location", map[string]any{"lat": 1.0, "lng": 2.0})
	if got := CurrentLocation(); got.HasAltitude {
		t.Error("HasAltitude is true with no altitude in the payload")
	}
	ReceiveHostEvent("location", map[string]any{"lat": 1.0, "lng": 2.0, "altitude": 0.0})
	if got := CurrentLocation(); !got.HasAltitude || got.Altitude != 0 {
		t.Error("an explicit altitude of 0 must read as a real sea-level fix")
	}
}

// "No location here" and "no fix yet" are different facts and different
// screens: the second is a spinner, and on cold GPS a long one.
func TestNoLocationIsDistinctFromNoFixYet(t *testing.T) {
	recordLocationEvents(t)

	before := CurrentLocation()
	if before.Received || before.Available {
		t.Fatalf("a fresh record should be neither received nor available: %+v", before)
	}

	ReceiveHostEvent("location", map[string]any{
		"available": false,
		"error":     "location permission denied",
	})
	got := CurrentLocation()
	if !got.Received {
		t.Error("the refusal is an event and must set Received")
	}
	if got.Available {
		t.Error("Available should be false")
	}
	if got.Error != "location permission denied" {
		t.Errorf("error = %q", got.Error)
	}

	// And an available fix clears the error, so a recovered sensor does not
	// leave a stale reason on screen.
	ReceiveHostEvent("location", map[string]any{"lat": 1.0, "lng": 2.0, "error": "ignored"})
	if got := CurrentLocation(); got.Error != "" {
		t.Errorf("error = %q on an available fix, want cleared", got.Error)
	}
}

// Sub-metre jitter updates the record and does not wake subscribers. A
// stationary phone still jitters as satellites come and go, and a render pass
// per jitter for the life of a screen is what the filter exists to stop.
func TestSubMetreJitterUpdatesTheRecordWithoutWakingSubscribers(t *testing.T) {
	recordLocationEvents(t)
	var calls int
	cancel := OnLocation(func(Location) { calls++ })
	defer cancel()

	ReceiveHostEvent("location", map[string]any{"lat": 38.7223, "lng": -9.1393, "accuracy": 8.0})
	if calls != 1 {
		t.Fatalf("the first fix notified %d times, want 1", calls)
	}

	// About 30cm north, and the same accuracy.
	ReceiveHostEvent("location", map[string]any{"lat": 38.72230270, "lng": -9.1393, "accuracy": 8.0})
	if calls != 1 {
		t.Errorf("a 30cm jitter woke subscribers (%d calls)", calls)
	}
	// The record still moved: the filter is on the notification, not the fix.
	if got := CurrentLocation().Lat; got != 38.72230270 {
		t.Errorf("record lat = %v, want the newest reading", got)
	}

	// Ten metres north is news.
	ReceiveHostEvent("location", map[string]any{"lat": 38.72239, "lng": -9.1393, "accuracy": 8.0})
	if calls != 2 {
		t.Errorf("a ten-metre step did not notify (%d calls)", calls)
	}
}

// Accuracy is a filtered number here and a state transition in the compass, and
// the difference is in what the hosts report: a GPS radius drifts continuously,
// so treating every change as news would leave the distance filter doing
// nothing.
func TestAccuracyDriftIsFilteredButARealChangeNotifies(t *testing.T) {
	recordLocationEvents(t)
	var calls int
	cancel := OnLocation(func(Location) { calls++ })
	defer cancel()

	ReceiveHostEvent("location", map[string]any{"lat": 0.0, "lng": 0.0, "accuracy": 8.0})
	ReceiveHostEvent("location", map[string]any{"lat": 0.0, "lng": 0.0, "accuracy": 8.3})
	if calls != 1 {
		t.Errorf("a 0.3m accuracy drift notified (%d calls)", calls)
	}
	// A fix that degrades from street to city is a different screen: a map
	// centred confidently on a 2km circle is a lie.
	ReceiveHostEvent("location", map[string]any{"lat": 0.0, "lng": 0.0, "accuracy": 2000.0})
	if calls != 2 {
		t.Errorf("a collapse in accuracy did not notify (%d calls)", calls)
	}
}

// The start/stop transitions notify even though nothing moved: Active changed,
// and a screen drawing "acquiring…" has to hear it.
func TestLocationStartAndStopNotifySubscribers(t *testing.T) {
	recordLocationEvents(t)
	var seen []bool
	cancel := OnLocation(func(l Location) { seen = append(seen, l.Active) })
	defer cancel()

	StartLocation()
	StopLocation()
	if len(seen) != 2 || !seen[0] || seen[1] {
		t.Errorf("Active seen as %v, want [true false]", seen)
	}
}

// A fix that arrives while the sensor is running carries Active, which is
// core's count and not the host's word.
func TestAFixWhileRunningCarriesActive(t *testing.T) {
	recordLocationEvents(t)
	StartLocation()
	defer StopLocation()
	ReceiveHostEvent("location", map[string]any{"lat": 1.0, "lng": 2.0})
	if !CurrentLocation().Active {
		t.Error("a fix delivered while the sensor runs should carry Active")
	}
}

func TestOnLocationCancelStopsDelivery(t *testing.T) {
	recordLocationEvents(t)
	var calls int
	cancel := OnLocation(func(Location) { calls++ })
	ReceiveHostEvent("location", map[string]any{"lat": 1.0, "lng": 1.0})
	cancel()
	ReceiveHostEvent("location", map[string]any{"lat": 40.0, "lng": 40.0})
	if calls != 1 {
		t.Errorf("a cancelled subscription received %d events, want 1", calls)
	}
}

// The two sensors are independent: one event name, two kinds, two records, two
// reference counts. A Start on one must not start the other, which is the
// failure a shared counter would produce.
func TestTheTwoSensorsDoNotShareAReferenceCount(t *testing.T) {
	resetHeadingForTest()
	recordLocationEvents(t)
	t.Cleanup(resetHeadingForTest)

	StartLocation()
	defer StopLocation()
	if HeadingActive() {
		t.Error("starting the GPS started the magnetometer")
	}
	ReceiveHostEvent("heading", map[string]any{"magnetic": 90.0})
	if CurrentLocation().Lat != 0 || CurrentLocation().Received {
		t.Error("a heading event reached the location record")
	}
}

// The window between a re-armed sensor and its first fix.
//
// Both natives keep a refused start armed, because the grant arrives after the
// screen that asks for it. When it does, the GPS begins working and Go's last
// event is still the refusal — for tens of seconds on cold GPS. This is the
// host withdrawing that refusal, and the assertion that matters is the last
// one: a screen that has never heard of LocationAcquiring stops printing the
// reason, because the reason is gone rather than because the screen learned a
// new field.
func TestAReArmedSensorStopsReportingWhyTheLastStartFailed(t *testing.T) {
	recordLocationEvents(t)
	StartLocation()
	t.Cleanup(StopLocation)

	ReceiveHostEvent("location", map[string]any{
		"available": false,
		"error":     "location permission not granted",
	})
	if got := CurrentLocation(); got.Available || got.Error == "" {
		t.Fatalf("setup: expected a refusal on the record, got %+v", got)
	}

	ReceiveHostEvent("location", map[string]any{"acquiring": true})

	got := CurrentLocation()
	if got.Error != "" {
		t.Errorf("Error = %q after a re-arm, want cleared", got.Error)
	}
	if !got.Available {
		t.Error("Available should be true while acquiring: the sensor took the start")
	}
	if got.Received {
		t.Error("Received should be false: the sensor has reported nothing about this run")
	}
	if !got.Active {
		t.Error("Active should still be true: the reference count did not move")
	}
	if got.Lat != 0 || got.Lng != 0 || got.Accuracy != -1 || got.HasAltitude {
		t.Errorf("an acquiring report carries no reading, got %+v", got)
	}
}

// An acquiring report is a state and not a reading, so nothing else in the
// payload is read. A host that sent coordinates with it would be claiming a
// fix it has just said it does not have.
func TestAnAcquiringReportIgnoresEveryOtherKey(t *testing.T) {
	recordLocationEvents(t)
	StartLocation()
	t.Cleanup(StopLocation)

	ReceiveHostEvent("location", map[string]any{
		"acquiring": true,
		"lat":       51.5, "lng": -0.12,
		"accuracy": 5.0, "altitude": 30.0,
		"error": "ignored",
	})
	got := CurrentLocation()
	if got.Lat != 0 || got.Lng != 0 || got.Accuracy != -1 || got.HasAltitude || got.Error != "" {
		t.Errorf("acquiring should carry nothing, got %+v", got)
	}
}

// A re-arm with nothing running is a host bug, and acting on it would move the
// record for a sensor nobody asked for.
func TestAcquiringWithNoConsumerChangesNothing(t *testing.T) {
	recordLocationEvents(t)
	ReceiveHostEvent("location", map[string]any{
		"available": false,
		"error":     "location permission not granted",
	})
	before := CurrentLocation()
	ReceiveHostEvent("location", map[string]any{"acquiring": true})
	if got := CurrentLocation(); got != before {
		t.Errorf("record moved with no consumer: %+v -> %+v", before, got)
	}
}

// A new run does not inherit the last one's refusal, which is the other half
// of the same defect: two screens apart, or one screen after a trip to
// Settings, the reason the last start failed is history.
func TestANewRunDoesNotInheritARefusal(t *testing.T) {
	recordLocationEvents(t)
	StartLocation()
	ReceiveHostEvent("location", map[string]any{
		"available": false,
		"error":     "location permission not granted",
	})
	StopLocation()

	StartLocation()
	t.Cleanup(StopLocation)
	got := CurrentLocation()
	if got.Error != "" || !got.Available || got.Received {
		t.Errorf("a new run should start from nothing known, got %+v", got)
	}
}

// But a fix survives one, because a place does not stop being true. A screen
// that opens seconds after another closed should draw the last position it
// has rather than a spinner.
func TestANewRunKeepsAFix(t *testing.T) {
	recordLocationEvents(t)
	StartLocation()
	ReceiveHostEvent("location", map[string]any{"lat": 51.5, "lng": -0.12})
	StopLocation()

	StartLocation()
	t.Cleanup(StopLocation)
	got := CurrentLocation()
	if got.Lat != 51.5 || got.Lng != -0.12 || !got.Received || !got.Available {
		t.Errorf("a fix should survive a new run, got %+v", got)
	}
}

// The re-arm is a state change, so subscribers hear about it: a screen sitting
// on a refusal has to repaint when the refusal is withdrawn, and the distance
// filter would have swallowed it (nothing moved).
func TestAReArmNotifiesSubscribers(t *testing.T) {
	recordLocationEvents(t)
	StartLocation()
	t.Cleanup(StopLocation)
	ReceiveHostEvent("location", map[string]any{
		"available": false, "error": "location permission not granted",
	})

	var calls int
	cancel := OnLocation(func(Location) { calls++ })
	defer cancel()
	ReceiveHostEvent("location", map[string]any{"acquiring": true})
	if calls != 1 {
		t.Errorf("subscriber calls = %d, want 1", calls)
	}
}
