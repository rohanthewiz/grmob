import CoreLocation
import Foundation

/// The iOS half of grmob's compass (core/heading.go).
///
///     core.StartHeading   ──▶ "sensor" {kind:"heading", command:"start"} ──▶ here
///     core.CurrentHeading ◀── "heading" host event ◀── here
///
/// # Why CoreLocation and not CoreMotion
///
/// The compass lives in `CLLocationManager`, not in the motion framework, and
/// it is the one place on iOS that does the magnetometer fusion and the
/// declination lookup for you. `CMMotionManager` can hand over a raw magnetic
/// field vector, but turning that into a bearing means reimplementing both,
/// and the declination half needs a world magnetic model this app has no
/// business shipping.
///
/// # Magnetic is free, true is not
///
/// `startUpdatingHeading` needs no authorization at all — the magnetic bearing
/// is not private information. `trueHeading` is: geographic north requires
/// knowing where on the planet you are, so CoreLocation reports it only when
/// location authorization has been granted, and otherwise sets it negative.
///
/// So this host asks for nothing and reports `magnetic` always, adding `true`
/// only when the value is valid. An app that wants true north asks for
/// location authorization by its own route; nothing here prompts, because a
/// permission dialog the user did not expect is a worse failure than a
/// bearing that is a few degrees off a map.
final class HeadingSensor: NSObject, CLLocationManagerDelegate {
    static let shared = HeadingSensor()

    /// The host→app channel, set by `SystemEvents.attach`.
    var report: ((String, String) -> Void)?

    private let manager = CLLocationManager()
    private var running = false
    private var lastSentAt: TimeInterval = 0

    /// Throttle floor, matching the other two hosts: CoreLocation delivers
    /// heading updates far faster than a dial needs, and every event that
    /// reaches Go costs a full render pass.
    private let minInterval: TimeInterval = 0.066

    private override init() {
        super.init()
        manager.delegate = self
        // Degrees of change before CoreLocation bothers to tell us. This is
        // the platform's own filter and is cheaper than ours: below it, no
        // event is generated at all. Half a degree matches the notification
        // threshold Go applies on the other side (headingNotifyEpsilon), so
        // the two do not fight.
        manager.headingFilter = 0.5
        // The device's own top edge is the direction the compass reports,
        // which is what a Compass widget drawn upright on the screen means.
        manager.headingOrientation = .portrait
    }

    /// Dispatches one "sensor" system event. Unknown kinds and commands are dropped.
    func handle(_ data: [String: Any]) {
        guard (data["kind"] as? String) == "heading" else { return }
        switch data["command"] as? String {
        case "start": start()
        case "stop": stop()
        default: break
        }
    }

    private func start() {
        guard !running else { return }
        guard CLLocationManager.headingAvailable() else {
            // An iPad without a magnetometer, or the simulator, which reports
            // false here. One event, and Go stops waiting for a reading that
            // is never coming.
            send(["available": false, "error": "no compass on this device"])
            return
        }
        running = true
        lastSentAt = 0
        manager.startUpdatingHeading()
    }

    private func stop() {
        guard running else { return }
        running = false
        manager.stopUpdatingHeading()
    }

    // MARK: - CLLocationManagerDelegate

    func locationManager(_ manager: CLLocationManager, didUpdateHeading heading: CLHeading) {
        guard running else { return }

        // A negative magneticHeading means the reading is invalid — the
        // magnetometer is being interfered with, or has not settled. Dropped
        // rather than forwarded: Go would fold it into a plausible-looking
        // bearing, and a compass pointing confidently in the wrong direction
        // is worse than one that has not updated.
        guard heading.magneticHeading >= 0 else { return }

        let now = Date.timeIntervalSinceReferenceDate
        guard now - lastSentAt >= minInterval else { return }
        lastSentAt = now

        var payload: [String: Any] = [
            "magnetic": heading.magneticHeading,
            "ts": Int(now * 1000),
        ]
        // Both of these are negative-when-unknown in CoreLocation's API, and
        // both are simply omitted rather than forwarded as a negative: Go's
        // contract is that a missing key means "not known", which is exactly
        // what a negative means here.
        if heading.trueHeading >= 0 {
            payload["true"] = heading.trueHeading
        }
        if heading.headingAccuracy >= 0 {
            payload["accuracy"] = heading.headingAccuracy
        }
        send(payload)
    }

    func locationManagerShouldDisplayHeadingCalibration(_ manager: CLLocationManager) -> Bool {
        // Let the system put up its own figure-eight calibration sheet when it
        // decides the magnetometer needs it. Returning false here is the
        // choice an app makes when a modal would interrupt something more
        // important; a compass on screen is the one case where the interruption
        // is the point, and the sheet dismisses itself once calibrated.
        return true
    }

    func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
        // CLError.headingFailure is transient — strong local interference —
        // and CoreLocation recovers on its own, so it is not reported as
        // unavailable. Anything else means the manager has stopped, and Go
        // needs to know it is not getting readings.
        if (error as? CLError)?.code == .headingFailure { return }
        send(["available": false, "error": error.localizedDescription])
    }

    private func send(_ payload: [String: Any]) {
        guard let report else {
            NSLog("GrMob: heading reading with no host-event channel attached")
            return
        }
        guard let data = try? JSONSerialization.data(withJSONObject: payload),
              let json = String(data: data, encoding: .utf8)
        else { return }
        report("heading", json)
    }
}
