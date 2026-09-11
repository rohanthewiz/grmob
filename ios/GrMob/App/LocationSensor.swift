import CoreLocation
import Foundation

/// The iOS half of grmob's positioning (core/location.go).
///
///     core.StartLocation   ──▶ "sensor" {kind:"location", command:"start"} ──▶ here
///     core.CurrentLocation ◀── "location" host event ◀── here
///
/// Built on HeadingSensor's shape, one file over, and differing from it in the
/// two places location is more expensive than a compass.
///
/// # Three CLLocationManagers, on purpose
///
/// This is the third object in this app wrapping CLLocationManager —
/// HeadingSensor has one and Permissions has one — and they stay separate for
/// the reason Permissions' own comment gives: each one encodes a different
/// decision about when to prompt, and sharing the object would make one of
/// those decisions the others'. HeadingSensor asks for nothing ever.
/// Permissions asks only when the app told it to. This one asks when it has to,
/// because there is no fix at all without authorization, and it says so below.
///
/// # It will prompt, and core says so
///
/// An undetermined authorization is requested here: startUpdatingLocation on a
/// notDetermined status reports nothing, forever, with no error — so a host that
/// refused to ask would leave every app that had not called permission.Request
/// waiting on a fix that cannot arrive.
///
/// core.StartLocation documents that this happens and tells an app that wants
/// to control the moment to check and ask first (hooks.UsePermission plus
/// permission.Request from a gesture). That is the app's call to make, and this
/// is what happens when it does not make one.
///
/// A refusal is an event rather than a silence: `available: false` with a
/// reason, which is the shape Go's Location.Available and Location.Error are
/// written against and the only thing that lets a screen stop waiting.
///
/// # And a refusal is not final
///
/// The authorization usually settles *after* the screen that wants it, whether
/// at the prompt or in Settings a minute later. So a refused start stays armed
/// and `locationManagerDidChangeAuthorization` finishes it — without which the
/// sensor is dead for the life of the context tree, which is what an Android
/// emulator run found in the equivalent host. `awaitingAuthorization` carries
/// the whole argument.
///
/// # Info.plist
///
/// NSLocationWhenInUseUsageDescription must be present or the app *crashes*
/// when the prompt would appear. See ios/GrMob/Info.plist, and Permissions.swift
/// for the other three keys.
final class LocationSensor: NSObject, CLLocationManagerDelegate {
    static let shared = LocationSensor()

    /// The host→app channel, set by `SystemEvents.attach`.
    var report: ((String, String) -> Void)?

    private let manager = CLLocationManager()
    private var running = false
    private var lastSentAt: TimeInterval = 0

    /// A start is outstanding and cannot proceed until the authorization
    /// changes. Set when `start` bails on a refusal or hands the prompt to the
    /// user, cleared by a start that gets through and by `stop`.
    ///
    /// # The bug this exists for
    ///
    /// An Android emulator run found the same shape one file over: the grant
    /// arrives *after* the screen that wants it, because the tap that asks for
    /// it is on that screen, and a sensor that abandoned the start stays dead
    /// for the life of the context tree — `core.StartLocation` emits its
    /// "sensor" event only on the 0→1 transition of its reference count, so the
    /// command never comes twice.
    ///
    /// Two of this file's three refusal paths had it. `notDetermined` was
    /// already handled — it set `running` before prompting precisely so the
    /// authorization callback would find a start outstanding — and that is the
    /// pattern this flag generalises, without the lie that `running` means
    /// updates are flowing.
    ///
    /// The path that did NOT recover is `denied`: `start` reported and gave up,
    /// and the user flipping the switch in Settings and coming back reached
    /// `locationManagerDidChangeAuthorization`, which bailed on `guard running`.
    /// That is the same dead sensor, by the route iOS users are most likely to
    /// take.
    ///
    /// # Why iOS needs no seam to Permissions
    ///
    /// CoreLocation delivers authorization changes to the delegate, including
    /// ones the user made in Settings while the app was backgrounded. So the
    /// signal arrives here on its own. LocationManager on Android has no such
    /// callback, which is why LocationSensor.kt's equivalent flag is re-armed by
    /// Permissions.kt calling into it — the same fix, and the platform is what
    /// decides how it is wired.
    ///
    /// # What it does not fix
    ///
    /// Go's last location event is still the refusal until the first fix lands.
    /// See the note under LocationSensor.kt's `awaitingPermission`; the state
    /// that would say "acquiring after a refusal" is one `core.Location` does
    /// not carry.
    private var awaitingAuthorization = false

    /// Throttle floor, as the compass has: CoreLocation can deliver faster than
    /// any screen needs, and every event that reaches Go costs a full render
    /// pass. Looser than the heading's 0.066 — a position is not a needle, and a
    /// second is a long time at walking pace.
    private let minInterval: TimeInterval = 0.5

    private override init() {
        super.init()
        manager.delegate = self
        // The platform's own filter, which is cheaper than ours: below this
        // distance no event is generated at all. Five metres is inside a good
        // GPS fix's own accuracy, so it filters jitter rather than movement —
        // and Go filters again at one metre (locationNotifyEpsilonMeters) for
        // the hosts that have no such control.
        manager.distanceFilter = 5
        manager.desiredAccuracy = kCLLocationAccuracyBest
    }

    /// Dispatches one "sensor" system event. Unknown kinds and commands are
    /// dropped, exactly as HeadingSensor does — both are handed every sensor
    /// event and each answers for its own kind.
    func handle(_ data: [String: Any]) {
        guard (data["kind"] as? String) == "location" else { return }
        switch data["command"] as? String {
        case "start": start()
        case "stop": stop()
        default: break
        }
    }

    private func start() {
        guard !running else { return }
        guard CLLocationManager.locationServicesEnabled() else {
            // Switched off for the whole device. One event, and Go stops
            // waiting for a fix that is never coming.
            send(["available": false, "error": "location services are off"])
            return
        }
        switch manager.authorizationStatus {
        case .notDetermined:
            // The prompt. The start is recorded as outstanding so the
            // authorization callback below knows to finish it; nothing is
            // reported yet, because the answer is the user's and has not
            // arrived. This used to set `running`, which is the same signal
            // spelled as a claim that updates were flowing.
            awaitingAuthorization = true
            manager.requestWhenInUseAuthorization()
            return
        case .restricted, .denied:
            // Reported *and* left armed, which is the difference between "no"
            // and "not yet": the user can grant this in Settings and come back,
            // and CoreLocation will say so. See `awaitingAuthorization`.
            //
            // `restricted` is a parental-controls or MDM state the user cannot
            // lift from Settings, so arming it will usually earn nothing — but
            // it costs one Bool and the profile can change.
            awaitingAuthorization = true
            send(["available": false, "error": "location permission denied"])
            return
        default:
            break
        }
        running = true
        awaitingAuthorization = false
        lastSentAt = 0
        manager.startUpdatingLocation()
    }

    private func stop() {
        // Cleared even when nothing is running, and that is the point: a screen
        // that unmounts while still waiting for the authorization must not leave
        // the sensor armed, or an answer arriving later would start a GPS with
        // no consumer. core.StopLocation reaches here on the hook's close path
        // whether the sensor ever got going or not.
        let wasArmed = running || awaitingAuthorization
        running = false
        awaitingAuthorization = false
        guard wasArmed else { return }
        manager.stopUpdatingLocation()
    }

    // MARK: - CLLocationManagerDelegate

    func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        // Only interesting while a start is outstanding: an authorization that
        // changes while nothing asked for a fix is Permissions' business, not
        // this object's. `awaitingAuthorization` is the half of "outstanding"
        // that used to be missing — a start refused for want of the permission
        // is exactly the start this callback exists to finish, and a `running`-
        // only guard dropped it.
        guard running || awaitingAuthorization else { return }
        switch manager.authorizationStatus {
        case .authorizedWhenInUse, .authorizedAlways:
            running = true
            awaitingAuthorization = false
            lastSentAt = 0
            manager.startUpdatingLocation()
        case .restricted, .denied:
            // Revoked while running, or refused at the prompt. Either way the
            // start stays armed: this is the one authorization on iOS the user
            // can change from outside the app at any time.
            running = false
            awaitingAuthorization = true
            send(["available": false, "error": "location permission denied"])
        case .notDetermined:
            // The prompt is still on screen. Nothing to say yet.
            break
        @unknown default:
            break
        }
    }

    func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        guard running, let fix = locations.last else { return }
        // A negative horizontalAccuracy means the coordinate is invalid.
        // Dropped rather than forwarded: Go would take it for a position, and a
        // map centred confidently on a bad fix is worse than one that has not
        // updated.
        guard fix.horizontalAccuracy >= 0 else { return }

        let now = Date.timeIntervalSinceReferenceDate
        guard now - lastSentAt >= minInterval else { return }
        lastSentAt = now

        var payload: [String: Any] = [
            "lat": fix.coordinate.latitude,
            "lng": fix.coordinate.longitude,
            "accuracy": fix.horizontalAccuracy,
            "ts": Int(now * 1000),
        ]
        // Omitted rather than sent as a negative, which is this API's spelling
        // of "unknown" and is exactly what Go's missing-key contract means.
        // Altitude is only meaningful when its own accuracy is valid — an
        // altitude with verticalAccuracy < 0 is a number CoreLocation has not
        // stood behind.
        if fix.verticalAccuracy >= 0 {
            payload["altitude"] = fix.altitude
        }
        send(payload)
    }

    func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
        // kCLErrorLocationUnknown is transient — the device cannot get a fix
        // right now and CoreLocation keeps trying — so it is not reported as
        // unavailable. Anything else has stopped the manager, and Go needs to
        // know it is not getting fixes.
        if (error as? CLError)?.code == .locationUnknown { return }
        running = false
        send(["available": false, "error": error.localizedDescription])
    }

    private func send(_ payload: [String: Any]) {
        guard let report else {
            NSLog("GrMob: location fix with no host-event channel attached")
            return
        }
        guard let data = try? JSONSerialization.data(withJSONObject: payload),
              let json = String(data: data, encoding: .utf8)
        else { return }
        report("location", json)
    }
}
