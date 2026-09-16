import Foundation
import UserNotifications

/// The iOS half of core's local notifications (core/notifications.go).
///
///     core.PostNotification   ──▶ "notification" {command: "post", id, title, body, at?}
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
/// # Scheduled posts ("at")
///
/// A post carrying `at` (Unix milliseconds, sent only when it is in the
/// future) gets a UNCalendarNotificationTrigger instead of a nil trigger, and
/// the notification center holds the request: it fires with the app
/// suspended or terminated, which is the point. The trigger matches the local
/// wall-clock date down to the second rather than counting an interval, so a
/// clock or time-zone change between scheduling and firing moves it the way
/// an alarm clock would move. iOS keeps at most 64 pending requests per app
/// and drops the rest without an error; hooks.UseAlarms caps itself at 60.
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
            // JSONSerialization hands a number back as NSNumber whatever its
            // Go type was.
            let at = (data["at"] as? NSNumber).map { Date(timeIntervalSince1970: $0.doubleValue / 1000) }
            post(id: id, title: data["title"] as? String ?? "", body: data["body"] as? String ?? "", at: at)
        case "cancel":
            let center = UNUserNotificationCenter.current()
            center.removePendingNotificationRequests(withIdentifiers: [id])
            center.removeDeliveredNotifications(withIdentifiers: [id])
        default:
            break
        }
    }

    private func post(id: String, title: String, body: String, at: Date?) {
        let content = UNMutableNotificationContent()
        content.title = title
        content.body = body
        content.sound = .default
        // A nil trigger delivers now. A date that has passed by the time it
        // arrives here is posted now too: a calendar trigger for a moment
        // already gone would never fire.
        var trigger: UNNotificationTrigger?
        if let at, at > Date() {
            let parts = Calendar.current.dateComponents(
                [.year, .month, .day, .hour, .minute, .second], from: at)
            trigger = UNCalendarNotificationTrigger(dateMatching: parts, repeats: false)
        }
        // A banner already delivered under this id is taken down, so a
        // re-post that schedules replaces what is showing as it would on the
        // other hosts; adding under the same identifier replaces a pending one.
        if trigger != nil {
            UNUserNotificationCenter.current().removeDeliveredNotifications(withIdentifiers: [id])
        }
        let request = UNNotificationRequest(identifier: id, content: content, trigger: trigger)
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
