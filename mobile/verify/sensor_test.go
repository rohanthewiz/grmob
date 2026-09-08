package verify

import (
	"strings"
	"testing"
)

// The "sensor" system event on both native shells.
//
// core.StartHeading sends one event and then waits. If a shell has no arm for
// it the event is dropped — which is the documented contract for an *unknown*
// event and exactly the wrong outcome for a known one: nothing errors, nothing
// logs, and the app waits forever for a reading whose request never reached a
// magnetometer. Go cannot notice, because from Go's side a sent event and a
// heard event look identical.

var (
	swiftSystemEvents = nativeFile("ios", "GrMob", "App", "SystemEvents.swift")
	swiftHeading      = nativeFile("ios", "GrMob", "App", "HeadingSensor.swift")

	kotlinSystemEvents = nativeFile("android", "app", "src", "main", "java", "com",
		"grmob", "app", "SystemEvents.kt")
	kotlinHeading = nativeFile("android", "app", "src", "main", "java", "com",
		"grmob", "app", "HeadingSensor.kt")
)

// Both shells must dispatch "sensor" and must attach the sensor's return
// channel — a dispatch with no report path starts a magnetometer whose
// readings go nowhere.
func TestBothShellsDispatchTheSensorEvent(t *testing.T) {
	for _, pin := range []struct{ file, dispatch, attach string }{
		{
			file:     swiftSystemEvents,
			dispatch: `case "sensor": HeadingSensor.shared.handle(object)`,
			attach:   "HeadingSensor.shared.report =",
		},
		{
			file:     kotlinSystemEvents,
			dispatch: `"sensor" -> HeadingSensor.handle(data)`,
			attach:   "HeadingSensor.attach(appContext, runtime::hostEvent)",
		},
	} {
		src := valuesIn(t, pin.file)
		if !strings.Contains(src, pin.dispatch) {
			t.Errorf("%s: no arm for the \"sensor\" event — core.StartHeading is "+
				"dropped here and the app waits on a reading nobody asked for",
				pin.file)
		}
		if !strings.Contains(src, pin.attach) {
			t.Errorf("%s: the heading sensor's report channel is never attached, so "+
				"readings have nowhere to go", pin.file)
		}
	}
}

// Both hosts must answer "this device has no compass" rather than going quiet.
//
// Availability is the one piece of the contract a screen cannot work around:
// "no compass" and "no reading yet" call for different UI, and a host that
// stays silent leaves Go unable to tell them apart. Both hosts have a real
// path that hits it — an iPad with no magnetometer, the iOS simulator, an
// emulator without sensor emulation — so both must send the event.
func TestBothHostsReportAnAbsentCompass(t *testing.T) {
	for _, pin := range []struct{ file, guard, report string }{
		{
			file:   swiftHeading,
			guard:  "CLLocationManager.headingAvailable()",
			report: `send(["available": false, "error": "no compass on this device"])`,
		},
		{
			file:   kotlinHeading,
			guard:  "TYPE_ROTATION_VECTOR",
			report: `put("available", false)`,
		},
	} {
		src := valuesIn(t, pin.file)
		if !strings.Contains(src, pin.guard) {
			t.Errorf("%s: no availability guard (%s)", pin.file, pin.guard)
		}
		if !strings.Contains(src, pin.report) {
			t.Errorf("%s: an unavailable compass is not reported to Go — the app "+
				"cannot tell \"no compass\" from \"not yet\"", pin.file)
		}
	}
}

// Both hosts must throttle. The sensors deliver far faster than a dial needs,
// every event that crosses the bridge costs a full Go render pass, and the
// throttle has to be on the host side of the bridge to save anything.
func TestBothHostsThrottleTheirSensorStream(t *testing.T) {
	for _, pin := range []struct{ file, marker string }{
		{swiftHeading, "minInterval"},
		{kotlinHeading, "MIN_INTERVAL_MS"},
	} {
		if src := codeIn(t, pin.file); !strings.Contains(src, pin.marker) {
			t.Errorf("%s: no throttle (%s) — a 60Hz sensor drives 60 render passes "+
				"a second", pin.file, pin.marker)
		}
	}
}

// Neither host may fold the bearing onto the circle or convert it to some
// other unit: core.ReceiveHeading is the single place that normalises, which
// is what makes "Magnetic is in [0, 360)" an invariant rather than a hope.
//
// The Android host is the one that could plausibly do it locally — its sensor
// answers radians over (-pi, pi] and it already converts to degrees for its
// own smoothing filter — so what is pinned is that the conversion it does is
// the unit one and the fold is left to Go.
func TestAndroidConvertsUnitsButLeavesTheFoldToGo(t *testing.T) {
	src := valuesIn(t, kotlinHeading)
	if !strings.Contains(src, "Math.toDegrees(orientation[0].toDouble())") {
		t.Errorf("%s: the azimuth is not converted from radians — Go would read "+
			"a bearing of at most 6 degrees", kotlinHeading)
	}
	// The smoothing accumulator is bounded with a %, which is not a fold onto
	// [0, 360) — it keeps the float's precision and deliberately preserves the
	// sign, so the value handed over can still be negative.
	if !strings.Contains(src, `put("magnetic", next.toDouble())`) {
		t.Errorf("%s: the smoothed bearing is not what gets sent", kotlinHeading)
	}
}
