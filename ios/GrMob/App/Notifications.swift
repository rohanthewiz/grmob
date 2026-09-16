import Foundation
import UserNotifications

/// The iOS half of core's local notifications (core/notifications.go).
///
///     core.PostNotification   ──▶ "notification" {command: "post", id, title, body, at?}
///     core.CancelNotification ──▶ "notification" {command: "cancel", id}
///     core.SweepNotifications ──▶ "notification" {command: "sweep", prefix, request}
///                             ◀── "notification_swept" {request, fired: [ids]}
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
/// # The scheduled record, and sweeping
///
/// The notification center lists pending requests but forgets a request once
/// it fires, and its delivered list is whatever the user has not swiped away,
/// so neither can say what fired while the app was not running. A sweep
/// (core.SweepNotifications) needs exactly that, so the shell keeps its own
/// record in UserDefaults, the counterpart of Notifications.kt's store:
///
///     schedule ──▶ record[id] = at (Unix ms)      cancel / post now ──▶ remove
///     sweep(prefix) ──▶ every recorded id under prefix:
///                         remove pending + delivered; at ≤ now ──▶ fired; remove
///
/// The ids to remove come from the record, read synchronously on the main
/// thread where every command arrives, rather than from
/// getPendingNotificationRequests: that answers on a later turn, by which time
/// the app may have scheduled new requests under the same prefix that a sweep
/// taken from its answer would wrongly remove. Delivered banners posted
/// immediately (never recorded) are looked up asynchronously and removed only
/// if they were delivered before the sweep began. Entries more than a week
/// past their time are pruned whenever the record is written.
///
/// Authorization is the permission package's (Permissions.swift). A post the
/// user has not allowed is dropped by the system without an error, which is
/// the contract core documents.
final class Notifications: NSObject, UNUserNotificationCenterDelegate {
    static let shared = Notifications()

    /// The host-event reporter, set by `SystemEvents.attach`.
    var report: ((String, String) -> Void)?

    private override init() { super.init() }

    /// The UserDefaults key of the scheduled record; see "The scheduled record".
    private static let recordKey = "grmob.scheduled-notifications"
    /// How long an entry is kept after its time for a sweep to report it.
    private static let keepMs: Double = 7 * 24 * 60 * 60 * 1000

    private var record: [String: Double] {
        get { UserDefaults.standard.dictionary(forKey: Self.recordKey) as? [String: Double] ?? [:] }
        set { UserDefaults.standard.set(newValue, forKey: Self.recordKey) }
    }

    private static func nowMs() -> Double { Date().timeIntervalSince1970 * 1000 }

    private func remember(_ id: String, at: Date) {
        let now = Self.nowMs()
        var next = record.filter { now - $0.value <= Self.keepMs }
        next[id] = at.timeIntervalSince1970 * 1000
        record = next
    }

    private func forget(_ id: String) {
        var next = record
        if next.removeValue(forKey: id) != nil { record = next }
    }

    /// Becomes the notification center's delegate. See the class comment for
    /// why this has to happen during launch.
    func attach() {
        UNUserNotificationCenter.current().delegate = self
    }

    func handle(_ data: [String: Any]) {
        // The one command addressed by prefix rather than id.
        if data["command"] as? String == "sweep" {
            if let prefix = data["prefix"] as? String, !prefix.isEmpty,
               let request = data["request"] as? String, !request.isEmpty {
                sweep(prefix: prefix, request: request)
            }
            return
        }
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
            forget(id)
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
        if let at, trigger != nil {
            UNUserNotificationCenter.current().removeDeliveredNotifications(withIdentifiers: [id])
            remember(id, at: at)
        } else {
            forget(id)
        }
        let request = UNNotificationRequest(identifier: id, content: content, trigger: trigger)
        UNUserNotificationCenter.current().add(request) { error in
            if let error { NSLog("GrMob: notification \(id) not posted: \(error)") }
        }
    }

    /// Cancels everything under `prefix` and reports the recorded ids whose
    /// time had come. See "The scheduled record, and sweeping".
    private func sweep(prefix: String, request: String) {
        let started = Date()
        let now = Self.nowMs()
        let all = record
        let matched = all.filter { $0.key.hasPrefix(prefix) }
        let fired = matched.filter { $0.value <= now }.map(\.key).sorted()
        let center = UNUserNotificationCenter.current()
        if !matched.isEmpty {
            let ids = Array(matched.keys)
            center.removePendingNotificationRequests(withIdentifiers: ids)
            center.removeDeliveredNotifications(withIdentifiers: ids)
            record = all.filter { !$0.key.hasPrefix(prefix) }
        }
        center.getDeliveredNotifications { delivered in
            let stale = delivered
                .filter { $0.request.identifier.hasPrefix(prefix) && $0.date <= started }
                .map(\.request.identifier)
            if !stale.isEmpty { center.removeDeliveredNotifications(withIdentifiers: stale) }
        }
        guard let data = try? JSONSerialization.data(withJSONObject: ["request": request, "fired": fired]),
              let payload = String(data: data, encoding: .utf8)
        else { return }
        report?("notification_swept", payload)
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
