import UIKit

/// The iOS half of core.Share (core/share.go).
///
///     core.Share ──▶ "share" {id, text, url}
///                ◀── host event "share" {id, ok}
///
/// A UIActivityViewController, the system share sheet, presented over
/// whatever the key window is showing. ok is whether it was presented: the
/// sheet's completion reports a pick or a dismissal, and a dismissal is not
/// something a caller should fall back on (see "What ok means" in share.go),
/// so the answer goes back as soon as the sheet is up rather than when it
/// closes. ok=false means there was no window to present from, or nothing to
/// share.
///
/// Every request is answered, because Go holds the caller's done until its id
/// comes back.
///
/// In App/ rather than Runtime/ because UIActivityViewController is UIKit,
/// and Runtime/ is type-checked against macOS by ios/verify.
@MainActor
final class Share {
    static let shared = Share()

    /// The host-event reporter, set by `SystemEvents.attach`.
    var report: ((String, String) -> Void)?

    func handle(_ data: [String: Any]) {
        guard let id = data["id"] as? String, !id.isEmpty else { return }
        let text = data["text"] as? String ?? ""
        let link = (data["url"] as? String).flatMap { $0.isEmpty ? nil : URL(string: $0) }

        // The text and the URL go as separate items rather than one string:
        // a URL item is what gives Messages its link preview and what lets
        // "Copy" and "Add to Reading List" act on the link itself.
        var items: [Any] = []
        if !text.isEmpty { items.append(text) }
        if let link { items.append(link) }
        guard !items.isEmpty, let presenter = topViewController() else {
            send(id: id, ok: false)
            return
        }

        let sheet = UIActivityViewController(activityItems: items, applicationActivities: nil)
        // On an iPad the sheet is a popover and UIKit raises an exception
        // without an anchor. core.Share does not say which control asked, so
        // it is anchored at the centre of the presenting view with no arrow,
        // which is how a sheet with no source reads on iPadOS.
        if let popover = sheet.popoverPresentationController {
            popover.sourceView = presenter.view
            popover.sourceRect = CGRect(x: presenter.view.bounds.midX,
                                        y: presenter.view.bounds.midY,
                                        width: 0, height: 0)
            popover.permittedArrowDirections = []
        }
        presenter.present(sheet, animated: true) { [weak self] in
            self?.send(id: id, ok: true)
        }
    }

    /// The view controller the person is looking at: the key window's root,
    /// followed down through anything it is presenting (a GrMob Modal drawn
    /// as a sheet is one), because a controller that is already presenting
    /// cannot present a second time.
    private func topViewController() -> UIViewController? {
        let window = UIApplication.shared.connectedScenes
            .compactMap { $0 as? UIWindowScene }
            .flatMap(\.windows)
            .first { $0.isKeyWindow }
        var top = window?.rootViewController
        while let next = top?.presentedViewController, !next.isBeingDismissed {
            top = next
        }
        return top
    }

    private func send(id: String, ok: Bool) {
        let object: [String: Any] = ["id": id, "ok": ok]
        guard let data = try? JSONSerialization.data(withJSONObject: object),
              let payload = String(data: data, encoding: .utf8)
        else { return }
        report?("share", payload)
    }
}
