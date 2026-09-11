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
//
// Two sensors share the event name now, which widens the failure rather than
// changing it: each shell hands the event to both objects and each object
// answers for its own kind, so a shell that forgot one call has a sensor whose
// start is dropped while the other works — which is the same silence, in half
// the app.

var (
	swiftSystemEvents = nativeFile("ios", "GrMob", "App", "SystemEvents.swift")
	swiftHeading      = nativeFile("ios", "GrMob", "App", "HeadingSensor.swift")
	swiftLocation     = nativeFile("ios", "GrMob", "App", "LocationSensor.swift")

	kotlinSystemEvents = nativeFile("android", "app", "src", "main", "java", "com",
		"grmob", "app", "SystemEvents.kt")
	kotlinHeading = nativeFile("android", "app", "src", "main", "java", "com",
		"grmob", "app", "HeadingSensor.kt")
	kotlinLocation = nativeFile("android", "app", "src", "main", "java", "com",
		"grmob", "app", "LocationSensor.kt")
)

// Both shells must dispatch "sensor" and must attach the sensor's return
// channel — a dispatch with no report path starts a magnetometer whose
// readings go nowhere.
func TestBothShellsDispatchTheSensorEvent(t *testing.T) {
	for _, pin := range []struct{ file, dispatch, attach string }{
		{
			file:     swiftSystemEvents,
			dispatch: `case "sensor":`,
			attach:   "HeadingSensor.shared.report =",
		},
		{
			file:     kotlinSystemEvents,
			dispatch: `"sensor" -> {`,
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

// The second sensor reaches both shells, and each shell reports its fixes back.
//
// Held separately from the heading half above rather than folded into it,
// because the two sensors are genuinely independent — one event name, two
// objects, two reference counts in Go — and the failure of wiring one and not
// the other is the one this checks for. A shell that forwards "sensor" to the
// compass alone looks completely healthy to anybody watching a compass.
func TestBothShellsDispatchTheLocationSensor(t *testing.T) {
	for _, pin := range []struct{ file, dispatch, attach string }{
		{
			file:     swiftSystemEvents,
			dispatch: "LocationSensor.shared.handle(object)",
			attach:   "LocationSensor.shared.report =",
		},
		{
			file:     kotlinSystemEvents,
			dispatch: "LocationSensor.handle(data)",
			attach:   "LocationSensor.attach(appContext, runtime::hostEvent)",
		},
	} {
		src := valuesIn(t, pin.file)
		if !strings.Contains(src, pin.dispatch) {
			t.Errorf("%s: the \"sensor\" event never reaches the location sensor — "+
				"core.StartLocation is dropped here and hooks.UseLocation waits forever",
				pin.file)
		}
		if !strings.Contains(src, pin.attach) {
			t.Errorf("%s: the location sensor's report channel is never attached, so "+
				"fixes have nowhere to go", pin.file)
		}
	}
}

// Each sensor object answers for its own kind and drops the rest. That guard is
// what makes one event name with two recipients work at all: without it, a
// "sensor" event for the compass would also start the GPS, which is the most
// expensive accidental subscription in this framework.
func TestEachSensorAnswersOnlyForItsOwnKind(t *testing.T) {
	for _, pin := range []struct{ file, guard string }{
		{swiftHeading, `(data["kind"] as? String) == "heading"`},
		{swiftLocation, `(data["kind"] as? String) == "location"`},
		{kotlinHeading, `data.optString("kind") != "heading"`},
		{kotlinLocation, `data.optString("kind") != "location"`},
	} {
		if src := valuesIn(t, pin.file); !strings.Contains(src, pin.guard) {
			t.Errorf("%s: no kind guard (%s) — this object acts on the other sensor's "+
				"start and stop", pin.file, pin.guard)
		}
	}
}

// Both hosts must answer "no location here" rather than going quiet, for the
// reason both must answer it about the compass: "no fix yet" and "this device
// will never tell you" call for different screens, and on cold GPS the first
// one lasts long enough that a silence is indistinguishable from a failure.
//
// The two hosts reach that state by different routes, which is the platform's
// difference rather than this framework's. iOS can prompt from the sensor, so
// its refusal path is an authorization that came back denied; Android cannot —
// only an Activity can show the dialog — so its refusal path is a permission
// that was never granted. Both end in the same event.
func TestBothHostsReportAnUnavailableLocation(t *testing.T) {
	for _, pin := range []struct{ file, guard, report string }{
		{
			file:   swiftLocation,
			guard:  "case .restricted, .denied:",
			report: `send(["available": false, "error": "location permission denied"])`,
		},
		{
			file:   kotlinLocation,
			guard:  "if (!hasPermission(ctx))",
			report: `.put("available", false)`,
		},
	} {
		src := valuesIn(t, pin.file)
		if !strings.Contains(src, pin.guard) {
			t.Errorf("%s: no permission guard (%s)", pin.file, pin.guard)
		}
		if !strings.Contains(src, pin.report) {
			t.Errorf("%s: an unavailable location is not reported to Go — the app cannot "+
				"tell \"no permission\" from \"not yet\"", pin.file)
		}
	}
}

// A location start the permission refused must stay armed, on both hosts,
// because on both hosts the refusal arrives before the grant does.
//
// # The bug this pins
//
// An emulator run granted the permission after the screen had mounted, and the
// sensor stayed dead for the life of the context tree:
//
//	launch with permission revoked     permission: denied    Available: false
//	grant it through the app           permission: granted   Available: false
//	12s later, with a fix being fed    permission: granted   Available: false
//
// The permission answer reached Go and nothing reached the sensor. Both hosts'
// start() returned without recording that a start was outstanding, and
// core.StartLocation emits its "sensor" event only on the 0→1 transition of its
// reference count — while hooks.UseLocation guards its own slot with
// locationRecord.started — so the command that would re-arm the sensor is never
// sent twice. The only recovery was a remount, which is the navigation route
// hooks.UseLocation recommends for an unrelated reason.
//
// It is the likelier order rather than an edge case: the tap that asks for the
// permission is on the screen that wants the fix, and permission.Request must
// come from a gesture.
//
// # The two hosts wire it differently, and the platform is why
//
// CoreLocation reports authorization changes to its delegate, including ones
// the user made in Settings while the app was backgrounded, so iOS re-arms from
// locationManagerDidChangeAuthorization and needs no seam at all. Android's
// LocationManager has no such callback, so Permissions.kt — which has the
// answer in hand either way — calls into the sensor. Hence a third pin on that
// file: the flag is worth nothing without something to clear it.
//
// Both must also disarm on stop. A screen that unmounts while still waiting
// must not leave the sensor armed, or an answer arriving later starts a GPS
// with no consumer.
func TestBothHostsStayArmedForALateLocationGrant(t *testing.T) {
	for _, pin := range []struct{ file, arm, retry, disarm string }{
		{
			file: swiftLocation,
			arm:  "awaitingAuthorization = true",
			// The guard that used to read `guard running else { return }`, which
			// is what dropped the grant.
			retry:  "guard running || awaitingAuthorization else { return }",
			disarm: "awaitingAuthorization = false",
		},
		{
			file:   kotlinLocation,
			arm:    "awaitingPermission = true",
			retry:  "fun permissionAnswer(kind: String, status: String)",
			disarm: "awaitingPermission = false",
		},
	} {
		src := valuesIn(t, pin.file)
		if !strings.Contains(src, pin.arm) {
			t.Errorf("%s: a start the permission refused is abandoned rather than armed "+
				"(%s) — the grant arrives a tap later and no second start command is "+
				"ever sent, so the sensor is dead for the life of the context tree",
				pin.file, pin.arm)
		}
		if !strings.Contains(src, pin.retry) {
			t.Errorf("%s: nothing retries the armed start (%s) — the flag is set and "+
				"never read, which is the same dead sensor with bookkeeping",
				pin.file, pin.retry)
		}
		if !strings.Contains(src, pin.disarm) {
			t.Errorf("%s: the armed start is never cleared (%s) — a screen that unmounts "+
				"while waiting leaves the sensor armed, and a grant arriving later for "+
				"some other reason starts a GPS with no consumer", pin.file, pin.disarm)
		}
	}

	// Android's half of the wiring, which iOS does not need: the answer has to
	// be handed to the sensor, because LocationManager will not announce it.
	kotlinPermissions := nativeFile("android", "app", "src", "main", "java", "com",
		"grmob", "app", "Permissions.kt")
	const handoff = "LocationSensor.permissionAnswer(kind, status)"
	if src := valuesIn(t, kotlinPermissions); !strings.Contains(src, handoff) {
		t.Errorf("%s: the permission answer never reaches the location sensor (%s) — "+
			"on Android that is the only signal a refused start can recover from",
			kotlinPermissions, handoff)
	}
}

// Both hosts must throttle their fix stream, and here the throttle is about
// power as much as about render passes: a provider asked for fixes ten times a
// second keeps the radio awake.
func TestBothHostsThrottleTheirLocationStream(t *testing.T) {
	for _, pin := range []struct{ file, marker string }{
		{swiftLocation, "minInterval"},
		{kotlinLocation, "MIN_INTERVAL_MS"},
	} {
		if src := codeIn(t, pin.file); !strings.Contains(src, pin.marker) {
			t.Errorf("%s: no throttle (%s) — a 1Hz provider is 1 render pass a second "+
				"and a radio that never sleeps", pin.file, pin.marker)
		}
	}
}

// Neither host may normalise the coordinates: core.ReceiveLocation is the single
// place that clamps the latitude and wraps the longitude, which is what makes
// those invariants properties rather than hopes. What is pinned is that each host
// sends the platform's own numbers through.
func TestNeitherHostNormalisesItsCoordinates(t *testing.T) {
	for _, pin := range []struct{ file, sent string }{
		{swiftLocation, `"lat": fix.coordinate.latitude`},
		{kotlinLocation, `.put("lat", location.latitude)`},
	} {
		if src := valuesIn(t, pin.file); !strings.Contains(src, pin.sent) {
			t.Errorf("%s: the platform's own latitude is not what gets sent (%s)",
				pin.file, pin.sent)
		}
	}
}
