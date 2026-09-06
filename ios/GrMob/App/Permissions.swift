import AVFoundation
import CoreLocation
import Foundation
import Photos

/// The iOS half of Go's permission package.
///
///     permission.Check   ──▶ "permission" {command:"check",   kind:…} ──▶ here
///     permission.Request ──▶ "permission" {command:"request", kind:…} ──▶ here
///     permission.Current ◀── "permission" host event ◀── here
///
/// # Four frameworks, one question
///
/// There is no single authorization API on this platform. Each capability
/// lives behind the framework that owns it, each has its own enum, and three
/// of the four answer asynchronously in their own way:
///
///     camera       AVCaptureDevice.authorizationStatus(for: .video)
///     microphone   AVCaptureDevice.authorizationStatus(for: .audio)
///     location     CLLocationManager.authorizationStatus  (delegate callback)
///     storage      PHPhotoLibrary.authorizationStatus(for: .readWrite)
///
/// So the mapping table Go's Status exists to avoid on the *web* is
/// unavoidable here, and it is written once, in `map(_:)` below, rather than
/// four times.
///
/// # Why location needs an object and the other three do not
///
/// The three AV/Photos APIs are static: ask for a status, get one back; pass a
/// completion handler, get called. `CLLocationManager` is a stateful object
/// with a delegate, and it reports an authorization change through
/// `locationManagerDidChangeAuthorization` rather than through the completion
/// handler `requestWhenInUseAuthorization` does not have. It must also outlive
/// the request — a manager released while its prompt is on screen never
/// reports anything — which is why this is a shared singleton holding one.
///
/// It is deliberately *not* HeadingSensor's manager, even though both wrap
/// CLLocationManager. That one is a compass: it reports headings and asks for
/// nothing, on the explicit reasoning in its own comment that an unexpected
/// permission dialog is worse than a bearing a few degrees off. Sharing the
/// object would tie the two together and make one of those decisions the
/// other's.
///
/// # When in use, not always
///
/// `requestWhenInUseAuthorization` is the only one this asks for. "Always" is
/// a second prompt Apple shows on its own schedule after the app has
/// demonstrably used location in the foreground, and requesting it up front is
/// how an app gets refused. Go's single `permission.Location` constant says
/// the narrower thing on every platform for exactly this reason.
///
/// # Info.plist
///
/// Every one of these needs its usage-description key present or the app
/// *crashes* when the prompt would appear — a launch-time contract no Go code
/// can satisfy. See ios/GrMob/Info.plist; the keys are
/// NSCameraUsageDescription, NSMicrophoneUsageDescription,
/// NSLocationWhenInUseUsageDescription and NSPhotoLibraryUsageDescription.
final class Permissions: NSObject, CLLocationManagerDelegate {
    static let shared = Permissions()

    /// The host→app channel, set by `SystemEvents.attach`.
    var report: ((String, String) -> Void)?

    private let location = CLLocationManager()

    /// Whether a location *request* is outstanding. The delegate callback
    /// fires for every authorization change including the one that happens at
    /// first launch, so without this a screen that merely checked would be
    /// answered twice — once by the check and once by the delegate — and the
    /// second answer would arrive with no question behind it.
    private var locationRequested = false

    private override init() {
        super.init()
        location.delegate = self
    }

    /// Dispatches one "permission" system event. Unknown kinds and commands
    /// are dropped, matching every host's contract for unknown events.
    func handle(_ data: [String: Any]) {
        guard let kind = data["kind"] as? String else { return }
        switch data["command"] as? String {
        case "check": check(kind)
        case "request": request(kind)
        default: break
        }
    }

    // MARK: - Check

    private func check(_ kind: String) {
        switch kind {
        case "camera": send(kind, map(AVCaptureDevice.authorizationStatus(for: .video)))
        case "microphone": send(kind, map(AVCaptureDevice.authorizationStatus(for: .audio)))
        case "storage": send(kind, map(PHPhotoLibrary.authorizationStatus(for: .readWrite)))
        case "location": send(kind, map(location.authorizationStatus))
        default: break
        }
    }

    // MARK: - Request

    private func request(_ kind: String) {
        switch kind {
        case "camera":
            AVCaptureDevice.requestAccess(for: .video) { [weak self] ok in
                self?.send(kind, ok ? "granted" : "denied")
            }
        case "microphone":
            AVCaptureDevice.requestAccess(for: .audio) { [weak self] ok in
                self?.send(kind, ok ? "granted" : "denied")
            }
        case "storage":
            PHPhotoLibrary.requestAuthorization(for: .readWrite) { [weak self] status in
                self?.send(kind, self?.map(status) ?? "denied")
            }
        case "location":
            // Already decided: iOS shows nothing for a second request, so
            // answering from the current status is the truthful thing to do
            // and is what stops a screen waiting on a prompt that will not
            // appear. Only .notDetermined actually prompts.
            guard location.authorizationStatus == .notDetermined else {
                send(kind, map(location.authorizationStatus))
                return
            }
            locationRequested = true
            location.requestWhenInUseAuthorization()
        default:
            break
        }
    }

    // MARK: - CLLocationManagerDelegate

    func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        guard locationRequested else { return }
        // .notDetermined here means the prompt is still up — the delegate
        // fires once when the manager is created, before the user has
        // answered — so it is not an answer and the request stays open.
        guard manager.authorizationStatus != .notDetermined else { return }
        locationRequested = false
        send("location", map(manager.authorizationStatus))
    }

    // MARK: - The mapping

    /// AVFoundation and Photos share a shape; CoreLocation does not. Three
    /// overloads rather than one stringly-typed function, so the compiler
    /// checks that every case of every enum is named — each of these is
    /// `@frozen`-adjacent in practice but not guaranteed, hence the
    /// `@unknown default`, which reports Unavailable rather than guessing.
    private func map(_ status: AVAuthorizationStatus) -> String {
        switch status {
        case .authorized: return "granted"
        case .notDetermined: return "prompt"
        case .denied: return "denied"
        // A parental control or an MDM profile. Not the same as denied: the
        // user cannot fix it in Settings, which is the distinction Go's
        // Unavailable draws.
        case .restricted: return "unavailable"
        @unknown default: return "unavailable"
        }
    }

    private func map(_ status: PHAuthorizationStatus) -> String {
        switch status {
        case .authorized: return "granted"
        // The user picked specific photos. The app can read those, so this is
        // granted from Go's point of view — the alternative would hide a
        // feature that works. Which photos is the picker's business, not this
        // channel's.
        case .limited: return "granted"
        case .notDetermined: return "prompt"
        case .denied: return "denied"
        case .restricted: return "unavailable"
        @unknown default: return "unavailable"
        }
    }

    private func map(_ status: CLAuthorizationStatus) -> String {
        switch status {
        case .authorizedAlways, .authorizedWhenInUse: return "granted"
        case .notDetermined: return "prompt"
        case .denied: return "denied"
        case .restricted: return "unavailable"
        @unknown default: return "unavailable"
        }
    }

    private func send(_ kind: String, _ status: String) {
        guard let report else {
            NSLog("GrMob: permission answer with no host-event channel attached")
            return
        }
        guard let data = try? JSONSerialization.data(withJSONObject: ["kind": kind, "status": status]),
              let json = String(data: data, encoding: .utf8)
        else { return }
        report("permission", json)
    }
}
