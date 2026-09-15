import Foundation
import UserNotifications

/// The iOS half of core's local notifications (core/notifications.go).
///
///     core.PostNotification   ──▶ "notification" {command: "post", id, title, body}
///     core.CancelNotification ──▶ "notification" {command: "cancel", id}
///     core.OnNotificationTap  ◀── "notification_tap" {id}   (the delegate, below)
///
/// # Why this object is the notification center's delegate
///
/// Two things only a delegate can do, and both are part of the contract:
///
///   - Foreground presentation. With no delegate, iOS delivers a notification
///     posted while the app is on screen silently — nothing is drawn — which
///     would make a post from a running app look broken on exactly the
///     platform where the app is most often running when it posts.
///   - Taps. `didReceive` is the only place a tap is reported, including the
///     tap that cold-launched the app.
///
/// The delegate must be set before the app finishes launching or a
/// cold-launch tap is lost, which is why `attach()` runs from
/// SystemEvents.attach inside GrMobApp.init — the earliest point there is.
///
/// # The Go id is the request identifier
///
/// UNNotificationRequest is keyed by a string the app chooses, so the Go id is
/// used directly: re-adding under the same identifier replaces the banner, and
/// cancel removes exactly it, pending and delivered alike.
///
/// Authorization is the permission package's (Permissions.swift). A post the
/// user has not allowed is dropped by the system without an error, which is
/// the contract core documents.
final class Notifications: NSObject, UNUserNotificationCenterDelegate {
    static let shared = Notifications()

    /// The host-event reporter, set by `SystemEvents.attach`.
    var report: ((String, String) -> Void)?

    private override init() { super.init() }

    /// Becomes the notification center's delegate. See the class comment for
    /// why this has to happen during launch.
    func attach() {
        UNUserNotificationCenter.current().delegate = self
    }

    func handle(_ data: [String: Any]) {
        guard let id = data["id"] as? String, !id.isEmpty else { return }
        switch data["command"] as? String {
        case "post":
            post(id: id, title: data["title"] as? String ?? "", body: data["body"] as? String ?? "")
        case "cancel":
            let center = UNUserNotificationCenter.current()
            center.removePendingNotificationRequests(withIdentifiers: [id])
            center.removeDeliveredNotifications(withIdentifiers: [id])
        default:
            break
        }
    }

    private func post(id: String, title: String, body: String) {
        let content = UNMutableNotificationContent()
        content.title = title
        content.body = body
        content.sound = .default
        // A nil trigger delivers now. Scheduling is a different feature with
        // its own vocabulary (dates, repeats) that core does not have.
        let request = UNNotificationRequest(identifier: id, content: content, trigger: nil)
        UNUserNotificationCenter.current().add(request) { error in
            if let error { NSLog("GrMob: notification \(id) not posted: \(error)") }
        }
    }

    // MARK: - UNUserNotificationCenterDelegate

    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification,
        withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
    ) {
        // Draw it while the app is on screen, as a banner and in the list, with
        // its sound — the same as when the app is in the background.
        completionHandler([.banner, .list, .sound])
    }

    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse,
        withCompletionHandler completionHandler: @escaping () -> Void
    ) {
        defer { completionHandler() }
        // Only the default action — a tap on the banner. A dismissal arrives
        // here too when a category asks for it, and is not a tap.
        guard response.actionIdentifier == UNNotificationDefaultActionIdentifier else { return }
        let id = response.notification.request.identifier
        guard let data = try? JSONSerialization.data(withJSONObject: ["id": id]),
              let payload = String(data: data, encoding: .utf8)
        else { return }
        // The report closure calls the runtime, which is main-actor work; the
        // delegate may be called on a background queue.
        DispatchQueue.main.async { [report] in
            report?("notification_tap", payload)
        }
    }
}
