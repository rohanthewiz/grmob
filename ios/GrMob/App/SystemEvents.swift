import UIKit

/// The iOS half of grmob's system-event channel: the app→host events that are
/// deliberately not part of the view tree, because what they reach is the
/// platform itself rather than anything the reconciler could diff.
///
///     core.ShowToast  ──▶ "toast"     ──▶ a transient overlay label
///     core.OpenURL    ──▶ "open_url"  ──▶ UIApplication.open
///
/// Before this existed the events were emitted into a nil Go handler and
/// vanished on both natives — only the WASM host had a sink — so an app
/// calling ShowToast worked in the browser preview and silently did nothing
/// on a device.
///
/// Unknown event names are dropped, matching the contract every other host
/// applies: a newer app on an older shell degrades to silence rather than to
/// a crash.
enum SystemEvents {
    /// Wires Go's event stream to this process's UI.
    ///
    /// Call before `GrMobRuntime.start()` so an event emitted during the very
    /// first render pass has somewhere to land.
    static func attach(_ bridge: GrMobBridge) {
        bridge.setSystemEventListener { name, payload in
            // The callback runs on the Go goroutine that emitted the event.
            // Everything below is UIKit, which is main-actor only, so every
            // event hops — the same hop GrMobRuntime makes for patches. Main
            // queue dispatches run FIFO, so a toast emitted before a
            // navigation still appears first.
            Task { @MainActor in
                dispatch(name: name, payload: payload)
            }
        }
    }

    @MainActor
    private static func dispatch(name: String, payload: String) {
        guard let data = payload.data(using: .utf8),
              let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any]
        else {
            NSLog("GrMob: malformed payload for system event '\(name)'")
            return
        }
        switch name {
        case "toast": showToast(object)
        case "open_url": openURL(object)
        default: break
        }
    }

    @MainActor
    private static func openURL(_ data: [String: Any]) {
        guard let string = data["url"] as? String, !string.isEmpty,
              let url = URL(string: string)
        else { return }
        // `canOpenURL` is deliberately not consulted first: since iOS 9 it
        // answers only for schemes declared in LSApplicationQueriesSchemes,
        // so it reports false for perfectly openable custom schemes and would
        // suppress links that work. `open` reports the truth in its
        // completion handler, and there is nothing to do with the answer —
        // core.OpenURL is fire-and-forget by contract and a system event has
        // no return channel — so it is logged and dropped.
        UIApplication.shared.open(url, options: [:]) { opened in
            if !opened { NSLog("GrMob: nothing could open \(string)") }
        }
    }

    // MARK: - Toast

    /// The toast currently on screen, if any. A second toast replaces the
    /// first rather than stacking: the payload carries no identity, so there
    /// is no way to lay two of them out relative to each other, and a queue
    /// would make a burst of them outlast the action that caused it.
    @MainActor private static var current: UIView?

    @MainActor
    private static func showToast(_ data: [String: Any]) {
        guard let message = data["message"] as? String, !message.isEmpty else { return }
        // Go sends milliseconds (core.ToastConfig defaults to 2000). Unlike
        // Android's two fixed buckets, an overlay we draw ourselves can honor
        // the number exactly, so it does.
        let seconds = Double(data["duration"] as? Int ?? 2000) / 1000.0

        guard let window = keyWindow else { return }
        current?.removeFromSuperview()

        let label = PaddedLabel()
        label.text = message
        label.numberOfLines = 0
        label.textAlignment = .center
        label.textColor = .white
        label.font = .preferredFont(forTextStyle: .subheadline)
        // The payload's "style" key (core.UseToastStyle) is ignored: honoring
        // a core.Style here would mean a second, partial implementation of
        // GrMobStyle living outside the renderer. A styled toast degrades to
        // the platform look rather than to nothing.
        label.backgroundColor = UIColor.black.withAlphaComponent(0.82)
        label.layer.cornerRadius = 10
        label.layer.masksToBounds = true
        label.alpha = 0
        label.translatesAutoresizingMaskIntoConstraints = false
        // Transparent to touch: a toast is a notification, not a control, and
        // one covering the bottom of the screen must not eat taps meant for
        // whatever is under it.
        label.isUserInteractionEnabled = false

        window.addSubview(label)
        NSLayoutConstraint.activate([
            label.centerXAnchor.constraint(equalTo: window.centerXAnchor),
            label.bottomAnchor.constraint(
                equalTo: window.safeAreaLayoutGuide.bottomAnchor, constant: -32),
            label.leadingAnchor.constraint(
                greaterThanOrEqualTo: window.leadingAnchor, constant: 24),
            label.trailingAnchor.constraint(
                lessThanOrEqualTo: window.trailingAnchor, constant: -24),
        ])
        current = label

        UIView.animate(withDuration: 0.2) { label.alpha = 1 }
        UIView.animate(withDuration: 0.3, delay: seconds, options: []) {
            label.alpha = 0
        } completion: { _ in
            label.removeFromSuperview()
            if current === label { current = nil }
        }
    }

    @MainActor
    private static var keyWindow: UIWindow? {
        UIApplication.shared.connectedScenes
            .compactMap { $0 as? UIWindowScene }
            .flatMap(\.windows)
            .first { $0.isKeyWindow }
    }
}

/// A UILabel with interior padding. UILabel has no content inset, and a toast
/// whose text touches its rounded corners reads as a bug.
private final class PaddedLabel: UILabel {
    private let insets = UIEdgeInsets(top: 10, left: 16, bottom: 10, right: 16)

    override func drawText(in rect: CGRect) {
        super.drawText(in: rect.inset(by: insets))
    }

    override var intrinsicContentSize: CGSize {
        let size = super.intrinsicContentSize
        return CGSize(
            width: size.width + insets.left + insets.right,
            height: size.height + insets.top + insets.bottom)
    }

    // Multi-line labels lay out against a preferredMaxLayoutWidth derived
    // from the bounds, so the insets have to come off that too or the last
    // line runs under the right padding.
    override func textRect(
        forBounds bounds: CGRect, limitedToNumberOfLines numberOfLines: Int
    ) -> CGRect {
        let rect = super.textRect(
            forBounds: bounds.inset(by: insets), limitedToNumberOfLines: numberOfLines)
        return rect.inset(by: UIEdgeInsets(
            top: -insets.top, left: -insets.left,
            bottom: -insets.bottom, right: -insets.right))
    }
}
