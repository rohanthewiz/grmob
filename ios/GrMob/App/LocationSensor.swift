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
/// emulator run found in the equivalent host. `armed` carries the whole
/// argument, and covers Location Services being off device-wide as well.
///
/// Either way, a start that gets through says `acquiring: true` before any fix
/// exists, so that Go stops reporting the reason the previous attempt failed.
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

    /// A start is outstanding and cannot proceed until something outside this
    /// app changes. Set when `start` bails — on a refusal, on the prompt, or on
    /// Location Services being off device-wide — and cleared by a start that
    /// gets through and by `stop`.
    ///
    /// Named for the state rather than for the reason, because there are three
    /// reasons and one of them is not an authorization at all. All three end
    /// the same way: the user goes somewhere else, changes something, and comes
    /// back, and this object has to be still waiting when they do.
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
    /// # And the window it used to leave open
    ///
    /// Go's last location event used to stay the refusal until the first fix
    /// landed — tens of seconds on cold GPS, with a screen printing "location
    /// permission denied" about a sensor that was working. `beginUpdates` now
    /// reports `acquiring: true` the moment updates are actually requested,
    /// which is `core.LocationAcquiring`: it withdraws the refusal and leaves
    /// the record Active with nothing received, which is the spinner state.
    ///
    /// # The arm that was never run, and what running it found
    ///
    /// Location Services switched off device-wide used to be armed here on the
    /// same argument as the other two — that
    /// `locationManagerDidChangeAuthorization` would finish it, being
    /// CoreLocation's only channel for "the world outside changed" — and was
    /// recorded as unverified, because `simctl privacy` drives an app's own
    /// permission and the global toggle is several taps inside Settings.
    ///
    /// Driving those taps refuted it. **iOS delivers no authorization callback
    /// for the global switch**, so the arm led nowhere; worse, the arm was
    /// never even set, because the path the global switch takes is not the one
    /// it was written for. Three separate things had to change, and each is
    /// commented where it lives:
    ///
    ///     didFailWithError    .denied now arms. The global switch kills a
    ///                         RUNNING sensor through the error callback, not
    ///                         through either of the two paths that refuse a
    ///                         start, and that callback cleared `running` and
    ///                         armed nothing — a sensor that never asked again.
    ///     didFailWithError    The reason is read off the authorization, not
    ///                         off locationServicesEnabled(), which returned
    ///                         true with the switch off. The message was also
    ///                         NSError's untranslated fallback.
    ///     retryIfArmed        The recovery, on the foreground, since no
    ///                         callback arrives. See its own doc for the A/B.
    ///
    /// Android needs none of it: `onProviderEnabled` fires for the equivalent
    /// switch, which is the asymmetry LocationSensor.kt's `registered` flag
    /// records from the other side.
    private var armed = false

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
        switch manager.authorizationStatus {
        case .notDetermined:
            // The prompt. The start is recorded as outstanding so the
            // authorization callback below knows to finish it; nothing is
            // reported yet, because the answer is the user's and has not
            // arrived. This used to set `running`, which is the same signal
            // spelled as a claim that updates were flowing.
            armed = true
            manager.requestWhenInUseAuthorization()
            return
        case .restricted, .denied:
            // Reported *and* left armed, which is the difference between "no"
            // and "not yet": the user can grant this in Settings and come back,
            // and CoreLocation will say so. See `armed`.
            //
            // `restricted` is a parental-controls or MDM state the user cannot
            // lift from Settings, so arming it will usually earn nothing — but
            // it costs one Bool and the profile can change.
            armed = true
            send(["available": false, "error": "location permission denied"])
            return
        default:
            break
        }
        beginUpdates()
    }

    /// Turn the sensor on, or say why not — the one place that decides either,
    /// so `start` and the authorization callback cannot drift on it.
    ///
    /// The device-wide switch is checked HERE rather than at the top of
    /// `start`, which is where it used to be, because it is the last thing that
    /// can refuse and the authorization callback needs the same check: an
    /// authorization granted while Location Services is off is an authorization
    /// that still produces no fixes, and starting on it would leave a screen
    /// waiting on a spinner instead of reading a reason.
    private func beginUpdates() {
        guard CLLocationManager.locationServicesEnabled() else {
            // Switched off for the whole device. Reported, so Go stops waiting
            // for a fix that is never coming, and left armed, so the user can
            // go and switch it on.
            running = false
            armed = true
            send(["available": false, "error": "location services are off"])
            return
        }
        running = true
        armed = false
        lastSentAt = 0
        // Torn down before it is built up, and that is not belt-and-braces.
        // A `startUpdatingLocation` on a manager that is already in a delivery
        // session does not restart it — it is a no-op on top of the session's
        // existing state, so a manager that has just been refused stays
        // refused and reports nothing at all. Measured: after Location
        // Services was switched off and on again, this method left the screen
        // on "waiting for the first fix" indefinitely, and the only difference
        // between that and the relaunch that got a fix instantly was a fresh
        // manager.
        //
        // This is the same shape as the fix in LocationSensor.kt, one platform
        // over, where `requestLocationUpdates` on a live registration kept the
        // platform's distance filter and delivered nothing to a device that
        // had not moved. Both hosts now stop before they start.
        manager.stopUpdatingLocation()
        manager.startUpdatingLocation()
        // The sensor is on and has nothing yet. Said out loud because Go's last
        // event may be a refusal this start has just made untrue, and a cold
        // first fix is tens of seconds away — see core.LocationAcquiring.
        send(["acquiring": true])
    }

    /// Retries a start that Location Services being switched off device-wide
    /// killed. Called when the app comes back to the foreground.
    ///
    /// # Why the foreground, and not a callback
    ///
    /// `locationManagerDidChangeAuthorization` is CoreLocation's channel for
    /// "the world outside changed", and it carries this app's authorization
    /// faithfully — deny it in Settings and grant it again and the arm above
    /// finishes the start. It does **not** carry the device-wide Location
    /// Services switch. That was written down as unverified and is now
    /// measured: the same sequence on a simulator, driving the master switch
    /// in Settings and reading lesson 4.12's panel, before and after the three
    /// changes in this file.
    ///
    ///                          before                    after
    ///     services ON      kCLErrorDomain error 1    Waiting for the first fix
    ///     services OFF     No position, none on way  No position, none on way
    ///     services ON      kCLErrorDomain error 1    Waiting for the first fix
    ///     then a relaunch  Waiting for the first fix Waiting for the first fix
    ///
    /// Three things are in that table. The **last row** is what makes the
    /// third one a finding rather than a broken simulator: the platform was
    /// ready the whole time and nothing asked it. The **third row** is the
    /// recovery, and it happens here on the foreground rather than in the
    /// authorization callback, which never fires. And the **first column** is
    /// the second bug — `error.localizedDescription` for a CLError.denied is
    /// NSError's fallback string, and it was going to the screen verbatim.
    ///
    /// The foreground is the right substitute because of where the switch
    /// lives. Location Services is several taps inside Settings, so a user who
    /// changes it has necessarily left this app and come back — there is no
    /// path to that switch that does not pass through here. It is also
    /// self-limiting: `armed` is only true after a start was refused or
    /// killed, so an app that never asked for a fix does nothing on every
    /// foreground, and one that is running does nothing either.
    ///
    /// Android needs none of this: `onProviderEnabled` fires for the
    /// equivalent switch, which is why LocationSensor.kt recovers from a
    /// registered listener alone (see its `registered` flag).
    func retryIfArmed() {
        guard armed, !running else { return }
        // An app-level authorization that is still missing is not this
        // method's case — the callback above handles that one and will fire
        // the moment it changes. Retrying here would only re-send a refusal
        // Go already has.
        switch manager.authorizationStatus {
        case .authorizedWhenInUse, .authorizedAlways:
            beginUpdates()
        default:
            return
        }
    }

    private func stop() {
        // Cleared even when nothing is running, and that is the point: a screen
        // that unmounts while still waiting for the authorization must not leave
        // the sensor armed, or an answer arriving later would start a GPS with
        // no consumer. core.StopLocation reaches here on the hook's close path
        // whether the sensor ever got going or not.
        let wasArmed = running || armed
        running = false
        armed = false
        guard wasArmed else { return }
        manager.stopUpdatingLocation()
    }

    // MARK: - CLLocationManagerDelegate

    func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        // Only interesting while a start is outstanding: an authorization that
        // changes while nothing asked for a fix is Permissions' business, not
        // this object's. `armed` is the half of "outstanding"
        // that used to be missing — a start refused for want of the permission
        // is exactly the start this callback exists to finish, and a `running`-
        // only guard dropped it.
        guard running || armed else { return }
        switch manager.authorizationStatus {
        case .authorizedWhenInUse, .authorizedAlways:
            beginUpdates()
        case .restricted, .denied:
            // Revoked while running, or refused at the prompt. Either way the
            // start stays armed: this is the one authorization on iOS the user
            // can change from outside the app at any time.
            running = false
            armed = true
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

        // `.denied` is the one failure with a way back, so it is the one that
        // stays armed. The user threw a switch — this app's authorization, or
        // Location Services for the whole device — and can throw it back
        // without relaunching.
        //
        // The arm was missing here, and the hole was the shape of the path
        // that was never walked: `armed` was written for the two ways a start
        // is REFUSED (start() and beginUpdates() both set it), and this is the
        // way a start already RUNNING is killed. A simulator run is what found
        // it — Location Services off and on again left the screen reporting
        // the failure for as long as it was watched, while a relaunch got a
        // fix immediately, which is the signature of a sensor that never asked
        // again rather than of a device that cannot answer.
        //
        // The message is read back off the device rather than taken from the
        // error, for two reasons. CoreLocation does not say WHICH switch was
        // thrown — `.denied` covers both — and `error.localizedDescription`
        // for this code is NSError's fallback: "The operation couldn't be
        // completed. (kCLErrorDomain error 1.)", which was reaching the screen
        // verbatim. Both strings below are the ones beginUpdates() and
        // didChangeAuthorization already send when they refuse a start for the
        // same two reasons, so a screen sees one vocabulary whichever path it
        // arrived by.
        if (error as? CLError)?.code == .denied {
            armed = true
            // Which switch was thrown is read off the AUTHORIZATION and not
            // off `CLLocationManager.locationServicesEnabled()`, which was
            // observed returning true on a simulator with Location Services
            // switched off device-wide — so the guard in beginUpdates() above
            // cannot be relied on to name this case, and a message built on it
            // said the wrong one.
            //
            // The authorization is unambiguous by construction: if this app is
            // still authorized and CoreLocation nevertheless refused, the
            // refusal came from outside the app. If the authorization itself
            // is gone, it did not.
            let authorized = manager.authorizationStatus == .authorizedWhenInUse
                || manager.authorizationStatus == .authorizedAlways
            send(["available": false,
                  "error": authorized ? "location services are off"
                                      : "location permission denied"])
            return
        }
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
