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
            // The prompt. running is set first so the authorization callback
            // below knows a start is outstanding; nothing is reported yet,
            // because the answer is the user's and has not arrived.
            running = true
            manager.requestWhenInUseAuthorization()
            return
        case .restricted, .denied:
            send(["available": false, "error": "location permission denied"])
            return
        default:
            break
        }
        running = true
        lastSentAt = 0
        manager.startUpdatingLocation()
    }

    private func stop() {
        guard running else { return }
        running = false
        manager.stopUpdatingLocation()
    }

    // MARK: - CLLocationManagerDelegate

    func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        // Only interesting while a start is outstanding: an authorization that
        // changes while nothing asked for a fix is Permissions' business, not
        // this object's.
        guard running else { return }
        switch manager.authorizationStatus {
        case .authorizedWhenInUse, .authorizedAlways:
            lastSentAt = 0
            manager.startUpdatingLocation()
        case .restricted, .denied:
            running = false
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
