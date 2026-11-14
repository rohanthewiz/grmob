import CryptoKit
import Foundation
#if canImport(AlarmKit)
import AlarmKit
import SwiftUI
#endif

/// The AlarmKit half of core's alarm notifications ("Alarms" in
/// core/notifications.go): a post that carries `alarm` becomes a system alarm
/// on iOS 26+, with the system's own alarm screen — over the lock screen,
/// through Silent mode and Focus — instead of a banner that chimes once.
///
///     Notifications.post(alarm) ──usable()?──▶ yes: Alarms.schedule  (AlarmKit)
///                                         └──▶ no:  the banner, as before
///
/// # When an alarm is used, and when the banner is
///
/// All three must hold, and every miss falls back to the banner rather than
/// failing, so Go never asks what the device can do:
///
///   - iOS 26 or later — AlarmKit does not exist before it.
///   - NSAlarmKitUsageDescription in Info.plist. `grmob ios` keeps it only for
///     an app with "alarms": true in grmob.json; its absence is how a build
///     says it did not opt in. Without it AlarmKit refuses to authorize.
///   - The user authorized it (permission.Alarms). This never prompts: a
///     prompt that appears because the app went to the background is the one
///     that gets refused (core/notifications.go, "Permission is the app's to
///     ask for"). Not yet asked is a banner.
///
/// # The Go id is a UUID here
///
/// AlarmKit keys alarms by UUID, so the Go id is hashed into one (SHA-256,
/// the first 16 bytes, stamped as a version-8 UUID). The same id always gives
/// the same UUID, so cancel and re-post find the alarm with no table to keep
/// and none to lose when the process dies — the same reasoning as Android's
/// "the id is the tag".
///
/// # Snooze
///
/// With `snooze` set the alert gets a Snooze button whose behaviour is
/// `.countdown`: the system silences the alarm and rings it again after the
/// alarm's postAlert countdown, by itself, under the same UUID. Go is not told
/// (see core/notifications.go). No countdown *presentation* is supplied —
/// that is a Live Activity, which needs a widget extension this shell does
/// not have — so a snoozed alarm waits without a Live Activity and simply
/// rings again.
///
/// # Ordering
///
/// `schedule` is async and `cancel` is not. A cancel that ran while an earlier
/// schedule for the same id was still in flight would find nothing and then
/// lose to it, leaving an alarm Go thinks it took down — exactly what a quick
/// background-and-back does to 60 alarms at once. So every AlarmKit call goes
/// through one serial chain, in the order the commands arrived.
enum Alarms {
    /// The previous operation in the chain; see "Ordering". Main-thread only,
    /// like every command that reaches Notifications.
    private static var chain: Task<Void, Never>?

    /// Runs `op` after every operation queued before it.
    private static func enqueue(_ op: @escaping () async -> Void) {
        let previous = chain
        chain = Task { @MainActor in
            await previous?.value
            await op()
        }
    }

    /// Whether this build opted in: the usage key is present and non-empty.
    /// Also Permissions.swift's "unavailable" test.
    static var declared: Bool {
        let text = Bundle.main.object(forInfoDictionaryKey: "NSAlarmKitUsageDescription") as? String
        return !(text ?? "").isEmpty
    }

    /// Whether a post with `alarm` should become an AlarmKit alarm now; see
    /// "When an alarm is used".
    static var usable: Bool {
        #if canImport(AlarmKit)
        if #available(iOS 26.0, *), declared {
            return AlarmManager.shared.authorizationState == .authorized
        }
        #endif
        return false
    }

    /// The UUID an id maps to; see "The Go id is a UUID here".
    static func uuid(for id: String) -> UUID {
        var bytes = Array(SHA256.hash(data: Data(id.utf8)).prefix(16))
        bytes[6] = (bytes[6] & 0x0F) | 0x80 // version 8: custom
        bytes[8] = (bytes[8] & 0x3F) | 0x80 // RFC 4122 variant
        return UUID(uuid: (bytes[0], bytes[1], bytes[2], bytes[3], bytes[4], bytes[5], bytes[6], bytes[7],
                           bytes[8], bytes[9], bytes[10], bytes[11], bytes[12], bytes[13], bytes[14], bytes[15]))
    }

    /// Schedules (or replaces) the alarm for `id` at `at`. The title is what
    /// the alarm screen shows; AlarmKit has no body line, so an empty title
    /// borrows the body. `snooze` in seconds, 0 for no Snooze button.
    static func schedule(id: String, title: String, body: String, at: Date, snooze: TimeInterval) {
        #if canImport(AlarmKit)
        guard #available(iOS 26.0, *) else { return }
        let uuid = uuid(for: id)
        let text = title.isEmpty ? body : title
        enqueue {
            let manager = AlarmManager.shared
            // Replacing: a second schedule under one UUID is not documented to
            // replace the first, so the old one is taken out explicitly. It
            // throws when there is none, which is the common case.
            try? manager.cancel(id: uuid)
            let alert = Self.alert(title: text, snooze: snooze > 0)
            let attributes = AlarmAttributes<GrMobAlarmMetadata>(
                presentation: AlarmPresentation(alert: alert), metadata: nil, tintColor: .accentColor)
            let configuration = AlarmManager.AlarmConfiguration<GrMobAlarmMetadata>(
                countdownDuration: snooze > 0 ? Alarm.CountdownDuration(preAlert: nil, postAlert: snooze) : nil,
                // A fixed instant rather than a relative wall-clock time: Go
                // already resolved the wall clock (package alarm), and posts
                // each occurrence as its own id.
                schedule: .fixed(max(at, Date().addingTimeInterval(1))),
                attributes: attributes)
            do {
                _ = try await manager.schedule(id: uuid, configuration: configuration)
            } catch {
                NSLog("GrMob: alarm \(id) not scheduled: \(error)")
            }
        }
        #endif
    }

    #if canImport(AlarmKit)
    /// The alert's presentation: the title, and Snooze when asked for.
    ///
    /// iOS 26.0 requires the Stop button to be described by the app; 26.1
    /// draws its own and deprecated the parameter, adding an initializer
    /// without it that 26.0 does not have. So the button is passed on 26.0
    /// only — the deprecation warning on that branch is the price of running
    /// there at all.
    @available(iOS 26.0, *)
    private static func alert(title: String, snooze: Bool) -> AlarmPresentation.Alert {
        let resource = LocalizedStringResource(stringLiteral: title)
        let snoozeButton = snooze ? AlarmButton(text: "Snooze", textColor: .white, systemImageName: "zzz") : nil
        let behavior: AlarmPresentation.Alert.SecondaryButtonBehavior? = snooze ? .countdown : nil
        if #available(iOS 26.1, *) {
            return AlarmPresentation.Alert(
                title: resource, secondaryButton: snoozeButton, secondaryButtonBehavior: behavior)
        }
        return AlarmPresentation.Alert(
            title: resource,
            stopButton: AlarmButton(text: "Stop", textColor: .white, systemImageName: "stop.circle"),
            secondaryButton: snoozeButton, secondaryButtonBehavior: behavior)
    }
    #endif

    /// Takes the alarm for `id` down, scheduled or ringing. Harmless when
    /// there is none, and a no-op below iOS 26.
    static func cancel(id: String) {
        #if canImport(AlarmKit)
        guard #available(iOS 26.0, *) else { return }
        let uuid = uuid(for: id)
        enqueue { try? AlarmManager.shared.cancel(id: uuid) }
        #endif
    }

    /// The sweep's half: cancels what is still to come among `ids` and leaves
    /// any that is ringing right now, as a sweep leaves a shown banner (see
    /// core.SweepNotifications). A snoozed alarm is still to come.
    static func sweep(ids: [String]) {
        #if canImport(AlarmKit)
        guard #available(iOS 26.0, *), !ids.isEmpty else { return }
        let uuids = Set(ids.map(uuid(for:)))
        enqueue {
            let manager = AlarmManager.shared
            let ringing = Set(((try? manager.alarms) ?? []).filter { $0.state == .alerting }.map(\.id))
            for uuid in uuids where !ringing.contains(uuid) {
                try? manager.cancel(id: uuid)
            }
        }
        #endif
    }

    // MARK: - permission.Alarms

    /// The Go status for permission.Alarms; see permission.go.
    static func status() -> String {
        #if canImport(AlarmKit)
        if #available(iOS 26.0, *), declared {
            return map(AlarmManager.shared.authorizationState)
        }
        #endif
        return "unavailable"
    }

    /// Asks when undecided (a system dialog) and answers with the status
    /// either way, through `answer` on the main thread.
    static func request(_ answer: @escaping (String) -> Void) {
        #if canImport(AlarmKit)
        if #available(iOS 26.0, *), declared {
            let manager = AlarmManager.shared
            guard manager.authorizationState == .notDetermined else {
                answer(map(manager.authorizationState))
                return
            }
            Task { @MainActor in
                // A thrown request (no usage key would be one) reads back as
                // whatever the state now is, which is still "not determined"
                // only if nothing changed.
                _ = try? await manager.requestAuthorization()
                answer(map(manager.authorizationState))
            }
            return
        }
        #endif
        answer("unavailable")
    }

    #if canImport(AlarmKit)
    @available(iOS 26.0, *)
    private static func map(_ state: AlarmManager.AuthorizationState) -> String {
        switch state {
        case .authorized: return "granted"
        case .denied: return "denied"
        case .notDetermined: return "prompt"
        @unknown default: return "unavailable"
        }
    }
    #endif
}

#if canImport(AlarmKit)
/// AlarmKit's attributes are generic over app metadata, which this shell has
/// none of: Go keeps everything it needs in the notification id.
@available(iOS 26.0, *)
struct GrMobAlarmMetadata: AlarmMetadata {}
#endif
