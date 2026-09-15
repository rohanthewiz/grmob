import UIKit

/// The iOS half of core.Haptic (core/haptics.go): one named effect per
/// "haptic" system event, fire-and-forget.
///
/// UIKit's feedback generators rather than Core Haptics: they are what every
/// system control uses, they respect the user's "System Haptics" setting
/// without the app checking, and they need no engine lifecycle. The cost is
/// that there is no duration to set, which is exactly why core names kinds
/// instead of durations.
///
/// A fresh generator per event rather than a kept, `prepare()`d one: priming
/// keeps the Taptic Engine powered for a few seconds to shave latency off a
/// buzz that follows a gesture, and these arrive from Go with no gesture to
/// anticipate. The first-buzz latency is tens of milliseconds, well under
/// anything a person perceives as late for "something happened".
///
/// In App/ rather than Runtime/ because the generators are UIKit, and
/// Runtime/ is type-checked against macOS by ios/verify.
enum Haptics {
    @MainActor
    static func handle(_ data: [String: Any]) {
        switch data["kind"] as? String {
        case "selection":
            UISelectionFeedbackGenerator().selectionChanged()
        case "light":
            UIImpactFeedbackGenerator(style: .light).impactOccurred()
        case "medium":
            UIImpactFeedbackGenerator(style: .medium).impactOccurred()
        case "heavy":
            UIImpactFeedbackGenerator(style: .heavy).impactOccurred()
        case "success":
            UINotificationFeedbackGenerator().notificationOccurred(.success)
        case "warning":
            UINotificationFeedbackGenerator().notificationOccurred(.warning)
        case "error":
            UINotificationFeedbackGenerator().notificationOccurred(.error)
        default:
            // A kind this shell predates: silence, the system-event contract.
            break
        }
    }
}
