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

    /// A start was killed by something outside this app and can be resumed
    /// when that something changes back. Cleared by a start that gets through
    /// and by `stop`.
    ///
    /// # Which switch, given that the compass asks for nothing
    ///
    /// The magnetic bearing needs no authorization (see the type comment), so
    /// there is no per-app permission for a user to revoke — but the compass
    /// is still `CLLocationManager`'s, and Location Services switched off
    /// device-wide stops the manager for every one of its jobs. `CLError`
    /// spells that refusal `.denied`, the same code an app-level denial gets,
    /// and here it can only mean the global switch: nothing else was ever
    /// asked for.
    ///
    /// That is also why this file needs no equivalent of LocationSensor's
    /// authorization callback. There is no authorization to change, so
    /// `locationManagerDidChangeAuthorization` has nothing to say about a
    /// compass, and the foreground is not a substitute for a callback here —
    /// it is the only signal there has ever been.
    ///
    /// # Not measured, unlike the location arm
    ///
    /// The equivalent in LocationSensor was verified by driving the Settings
    /// switch on a simulator and reading lesson 4.12's panel. That experiment
    /// cannot be repeated for this file: `CLLocationManager.headingAvailable()`
    /// is false on the simulator, so `start` never gets past its own guard and
    /// there is no session for the switch to kill. What is written here is the
    /// pattern one file over, applied to a manager that is refused by the same
    /// switch for the same reason — and it is stated plainly because the arm
    /// costs one Bool and a dead compass costs the lesson that draws it.
    private var armed = false

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
            //
            // Deliberately NOT armed: a device without a magnetometer will not
            // grow one, and arming it would retry on every foreground for the
            // life of the process to re-send a message Go already has.
            send(["available": false, "error": "no compass on this device"])
            return
        }
        beginUpdates()
    }

    /// Turn the compass on. The one place that does, so `start` and
    /// `retryIfArmed` cannot drift on how.
    private func beginUpdates() {
        running = true
        armed = false
        lastSentAt = 0
        // Torn down before it is built up, for the reason measured one file
        // over (LocationSensor.beginUpdates): a start on a manager that is
        // already in a delivery session is a no-op on top of that session's
        // state, so a manager that has just been refused stays refused and
        // delivers nothing. Harmless on the path where nothing was running —
        // stopping a manager that is not updating is defined to do nothing —
        // and the whole point on the retry path, which is the only path that
        // can reach here with a session the platform has already killed.
        manager.stopUpdatingHeading()
        manager.startUpdatingHeading()
        // No `acquiring` event, and that asymmetry with LocationSensor is the
        // sensor's rather than an omission: a cold GPS fix is tens of seconds
        // away, so Go has to be told the refusal is withdrawn before the first
        // reading can say so, whereas a magnetic bearing arrives in a frame or
        // two and `Available` defaults to true on it (see core.ReceiveHeading,
        // which clears Error on any available reading).
    }

    private func stop() {
        // Cleared even when nothing is running: a screen that unmounts while
        // the compass is refused must not leave it armed, or a later
        // foreground would start a magnetometer with no consumer. The same
        // argument as LocationSensor.stop, and core.StopHeading reaches here
        // on the hook's close path either way.
        let wasOn = running || armed
        running = false
        armed = false
        guard wasOn else { return }
        manager.stopUpdatingHeading()
    }

    /// Retries a start that Location Services being switched off device-wide
    /// killed. Called when the app comes back to the foreground.
    ///
    /// The foreground is the right signal for the same reason it is in
    /// LocationSensor: that switch is several taps inside Settings, so a user
    /// who changes it has necessarily left this app and come back, and there
    /// is no path to it that does not pass through here. It is self-limiting
    /// in the same way too — `armed` is only ever true after a running compass
    /// was refused, so an app that never drew one does nothing on every
    /// foreground, and one that is running does nothing either.
    func retryIfArmed() {
        guard armed, !running else { return }
        // Re-asked rather than assumed. The guard in `start` reports a device
        // with no magnetometer as permanently unavailable, but this arm is
        // only reachable on a device that HAD one and was refused, so the
        // answer here is about the hardware still being there — cheap, and the
        // alternative is starting a manager that cannot deliver.
        guard CLLocationManager.headingAvailable() else { return }
        beginUpdates()
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
        running = false

        // `.denied` is the one failure with a way back, so it is the one that
        // stays armed — the shape LocationSensor's equivalent arrived at after
        // a simulator run found a sensor that never asked again.
        //
        // The message is written here rather than taken from the error, and
        // for the reason measured one file over: `error.localizedDescription`
        // for this code is NSError's untranslated fallback, "The operation
        // couldn't be completed. (kCLErrorDomain error 1.)", and it was
        // reaching the screen verbatim. The string is the one LocationSensor
        // sends for the same switch, so a screen that draws both sensors reads
        // one vocabulary.
        //
        // No authorization is consulted, unlike LocationSensor, because there
        // is none to consult: the compass asks for nothing, so a `.denied` it
        // receives cannot be about this app's permission. See `armed`.
        if (error as? CLError)?.code == .denied {
            armed = true
            send(["available": false, "error": "location services are off"])
            return
        }
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
