package core

import (
	"math"
	"testing"
)

// recordSensorEvents installs a recording system-event handler and resets the
// heading record, so one test's sensor never leaks into the next (both are
// package-level).
func recordSensorEvents(t *testing.T) *[]map[string]any {
	t.Helper()
	resetHeadingForTest()
	var events []map[string]any
	SetSystemEventHandler(func(name string, data map[string]any) {
		if name == "sensor" {
			events = append(events, data)
		}
	})
	t.Cleanup(func() {
		SetSystemEventHandler(nil)
		resetHeadingForTest()
	})
	return &events
}

func TestNormalizeDegreesFoldsOntoTheCircle(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{0, 0}, {90, 90}, {359.9, 359.9},
		{360, 0},    // a full turn is north, not a 361st degree
		{-90, 270},  // math.Mod alone keeps the sign; this is the case it gets wrong
		{730, 10},   // two laps and change
		{-450, 270}, // more than a lap the other way
	}
	for _, c := range cases {
		if got := NormalizeDegrees(c.in); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("NormalizeDegrees(%g) = %g, want %g", c.in, got, c.want)
		}
	}
	// A host that divides by zero somewhere upstream must not put a NaN into
	// every consumer's arithmetic.
	if got := NormalizeDegrees(math.NaN()); got != 0 {
		t.Errorf("NormalizeDegrees(NaN) = %g, want 0", got)
	}
	if got := NormalizeDegrees(math.Inf(1)); got != 0 {
		t.Errorf("NormalizeDegrees(+Inf) = %g, want 0", got)
	}
}

// The seam at north is the single most important piece of arithmetic in this
// file: every smoothing filter, change threshold and animation is built on it.
func TestAngleDeltaTakesTheShortWayRound(t *testing.T) {
	cases := []struct{ a, b, want float64 }{
		{0, 10, 10},
		{10, 0, -10},
		{359, 1, 2},    // the seam, forwards — NOT -358
		{1, 359, -2},   // the seam, backwards
		{0, 180, 180},  // exactly opposite resolves clockwise
		{0, 181, -179}, // just past opposite goes the other way
		{350, 370, 20}, // an unwrapped input is still a 20-degree turn
	}
	for _, c := range cases {
		if got := AngleDelta(c.a, c.b); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("AngleDelta(%g, %g) = %g, want %g", c.a, c.b, got, c.want)
		}
	}
}

func TestCardinalCentresEachSectorOnItsPoint(t *testing.T) {
	cases := []struct {
		deg  float64
		want string
	}{
		{0, "N"},
		{11, "N"},   // inside north's half-sector
		{349, "N"},  // the other side of it: north wraps across 0
		{12, "NNE"}, // just past the boundary
		{45, "NE"},
		{90, "E"},
		{180, "S"},
		{270, "W"},
		{-90, "W"},  // normalised first
		{412, "NE"}, // an unwrapped bearing still names a point
	}
	for _, c := range cases {
		if got := Cardinal(c.deg); got != c.want {
			t.Errorf("Cardinal(%g) = %q, want %q", c.deg, got, c.want)
		}
	}
	// Heading.Cardinal is the same answer taken from the magnetic bearing.
	if got := (Heading{Magnetic: 225}).Cardinal(); got != "SW" {
		t.Errorf("Heading{225}.Cardinal() = %q, want SW", got)
	}
}

// The reference count is the reason this file exists rather than a bool: two
// screens must be able to hold the sensor independently.
func TestStartStopIsRefcountedNotToggled(t *testing.T) {
	events := recordSensorEvents(t)

	StartHeading()
	StartHeading()
	if len(*events) != 1 {
		t.Fatalf("two Starts sent %d events, want 1", len(*events))
	}
	if (*events)[0]["command"] != "start" || (*events)[0]["kind"] != "heading" {
		t.Errorf("start payload = %v", (*events)[0])
	}

	// The first release must NOT stop the sensor — this is the bug the
	// refcount exists to make unwritable.
	StopHeading()
	if len(*events) != 1 {
		t.Fatalf("the first Stop of two sent an event: %v", *events)
	}
	if !HeadingActive() {
		t.Error("sensor reported inactive while one holder remains")
	}

	StopHeading()
	if len(*events) != 2 || (*events)[1]["command"] != "stop" {
		t.Fatalf("events = %v, want a trailing stop", *events)
	}
	if HeadingActive() {
		t.Error("sensor still active after the last Stop")
	}
}

func TestUnbalancedStopIsClampedAtZero(t *testing.T) {
	events := recordSensorEvents(t)

	// A double cleanup must not drive the count negative — if it did, the
	// next Start would only bring it back to zero and would never actually
	// start the sensor.
	StopHeading()
	StopHeading()
	if len(*events) != 0 {
		t.Fatalf("Stop with nothing running sent %v", *events)
	}

	StartHeading()
	if len(*events) != 1 || (*events)[0]["command"] != "start" {
		t.Fatalf("events = %v, want a start after the spurious stops", *events)
	}
	if !HeadingActive() {
		t.Error("sensor inactive after Start following unbalanced Stops")
	}
}

func TestReceiveHeadingNormalisesAndFillsTheRecord(t *testing.T) {
	recordSensorEvents(t)
	StartHeading()

	// Android's getOrientation answers negative for half the circle; this is
	// the one place all four hosts funnel through, so it is where the
	// [0, 360) invariant gets established.
	ReceiveHostEvent("heading", map[string]any{
		"magnetic": -45.0, "true": 372.0, "accuracy": 3.0, "ts": 1234.0,
	})

	h := CurrentHeading()
	if h.Magnetic != 315 {
		t.Errorf("Magnetic = %g, want 315", h.Magnetic)
	}
	if !h.HasTrue || h.True != 12 {
		t.Errorf("True = %g (has=%v), want 12 true", h.True, h.HasTrue)
	}
	if h.Accuracy != 3 {
		t.Errorf("Accuracy = %g, want 3", h.Accuracy)
	}
	if !h.Available || !h.Received || !h.Active {
		t.Errorf("flags: available=%v received=%v active=%v, want all true",
			h.Available, h.Received, h.Active)
	}
}

// "available" defaults to true when absent — the asymmetry documented on
// receiveHeading, so three hosts need not repeat a boolean fifteen times a
// second.
func TestAbsentAvailableMeansAvailable(t *testing.T) {
	resetHeadingForTest()
	t.Cleanup(resetHeadingForTest)

	ReceiveHostEvent("heading", map[string]any{"magnetic": 10.0})
	if h := CurrentHeading(); !h.Available {
		t.Error("a reading with no available key was treated as unavailable")
	}
}

// "no compass" and "no reading yet" are different answers and a UI draws
// different things for them.
func TestUnavailableIsDistinctFromNotYetReceived(t *testing.T) {
	resetHeadingForTest()
	t.Cleanup(resetHeadingForTest)

	before := CurrentHeading()
	if before.Received || before.Available {
		t.Fatalf("initial heading = %+v, want neither received nor available", before)
	}

	ReceiveHostEvent("heading", map[string]any{
		"available": false, "error": "no compass on this device",
	})
	h := CurrentHeading()
	if h.Available {
		t.Error("Available stayed true after an unavailable report")
	}
	if !h.Received {
		t.Error("Received stayed false after the host answered")
	}
	if h.Error != "no compass on this device" {
		t.Errorf("Error = %q", h.Error)
	}

	// A later good reading clears the error rather than leaving it standing.
	ReceiveHostEvent("heading", map[string]any{"magnetic": 90.0})
	if h := CurrentHeading(); h.Error != "" || !h.Available {
		t.Errorf("recovery left %+v", h)
	}
}

// The notification filter: the record always holds the newest reading, but
// subscribers are spared sub-degree drift at 15 Hz.
func TestSubDegreeDriftUpdatesTheRecordWithoutWakingSubscribers(t *testing.T) {
	resetHeadingForTest()
	t.Cleanup(resetHeadingForTest)

	var seen []float64
	cancel := OnHeading(func(h Heading) { seen = append(seen, h.Magnetic) })
	defer cancel()

	ReceiveHostEvent("heading", map[string]any{"magnetic": 100.0})
	ReceiveHostEvent("heading", map[string]any{"magnetic": 100.1}) // drift
	ReceiveHostEvent("heading", map[string]any{"magnetic": 100.2}) // drift
	if len(seen) != 1 {
		t.Fatalf("notified %v, want only the first reading", seen)
	}
	if got := CurrentHeading().Magnetic; got != 100.2 {
		t.Errorf("CurrentHeading = %g, want the newest 100.2 regardless", got)
	}

	// Accumulated drift does eventually cross the threshold: the filter
	// compares against the last *notified* value, not the last received one,
	// so a slow turn is not swallowed forever.
	ReceiveHostEvent("heading", map[string]any{"magnetic": 100.6})
	if len(seen) != 2 {
		t.Fatalf("notified %v, want the accumulated turn to get through", seen)
	}
}

// The threshold must not swallow a state change. Availability, errors and the
// active flag are transitions a screen has to react to at any bearing.
func TestStateChangesNotifyEvenWithoutMovement(t *testing.T) {
	events := recordSensorEvents(t)
	_ = events

	var seen []Heading
	cancel := OnHeading(func(h Heading) { seen = append(seen, h) })
	defer cancel()

	ReceiveHostEvent("heading", map[string]any{"magnetic": 50.0})
	// Same bearing, now unavailable: zero movement, but everything changed.
	ReceiveHostEvent("heading", map[string]any{
		"magnetic": 50.0, "available": false, "error": "sensor lost",
	})
	if len(seen) != 2 {
		t.Fatalf("notified %d times, want 2 — the second is a state change at the same bearing", len(seen))
	}
	if seen[1].Available || seen[1].Error != "sensor lost" {
		t.Errorf("second notification = %+v", seen[1])
	}
}

// The seam again, this time through the filter: a two-degree step across
// north must wake subscribers, and would not if the filter subtracted plainly.
func TestMovementAcrossNorthIsMeasuredTheShortWay(t *testing.T) {
	resetHeadingForTest()
	t.Cleanup(resetHeadingForTest)

	var seen int
	cancel := OnHeading(func(Heading) { seen++ })
	defer cancel()

	ReceiveHostEvent("heading", map[string]any{"magnetic": 359.0})
	ReceiveHostEvent("heading", map[string]any{"magnetic": 359.2}) // drift, filtered
	if seen != 1 {
		t.Fatalf("notified %d times after a 0.2 degree drift, want 1", seen)
	}
	ReceiveHostEvent("heading", map[string]any{"magnetic": 1.0}) // 2 degrees across the seam
	if seen != 2 {
		t.Errorf("notified %d times, want the cross-north step to get through", seen)
	}
}

// Start and Stop change Heading.Active with no reading in flight, so they must
// bypass the movement filter or a screen never learns the sensor came on.
func TestStartAndStopNotifySubscribers(t *testing.T) {
	recordSensorEvents(t)

	var seen []bool
	cancel := OnHeading(func(h Heading) { seen = append(seen, h.Active) })
	defer cancel()

	StartHeading()
	StopHeading()
	if len(seen) != 2 || !seen[0] || seen[1] {
		t.Errorf("Active notifications = %v, want [true false]", seen)
	}
}

func TestOnHeadingCancelStopsDelivery(t *testing.T) {
	resetHeadingForTest()
	t.Cleanup(resetHeadingForTest)

	var seen int
	cancel := OnHeading(func(Heading) { seen++ })
	ReceiveHostEvent("heading", map[string]any{"magnetic": 10.0})
	cancel()
	ReceiveHostEvent("heading", map[string]any{"magnetic": 200.0})
	if seen != 1 {
		t.Errorf("delivered %d times, want 1 before the cancel", seen)
	}
}
