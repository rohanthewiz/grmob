import UIKit

/// The iOS half of core's clipboard (core/clipboard.go).
///
///     core.WriteClipboard ──▶ "clipboard" {command: "write", text}
///     core.ReadClipboard  ──▶ "clipboard" {command: "read", id}
///                         ◀── host event "clipboard" {id, text, ok}
///
/// Every read is answered, including on failure, because Go holds the
/// caller's callback until the id comes back and has no timeout to fall back
/// on (see the lifetime note in clipboard.go).
///
/// In App/ rather than Runtime/ because UIPasteboard is UIKit, and Runtime/
/// is type-checked against macOS by ios/verify.
@MainActor
final class Clipboard {
    static let shared = Clipboard()

    /// The host-event reporter, set by `SystemEvents.attach`.
    var report: ((String, String) -> Void)?

    func handle(_ data: [String: Any]) {
        switch data["command"] as? String {
        case "write": write(data["text"] as? String ?? "")
        case "read": read(data["id"] as? String ?? "")
        default: break
        }
    }

    private func write(_ text: String) {
        UIPasteboard.general.string = text
    }

    private func read(_ id: String) {
        guard !id.isEmpty else { return }
        // Reading `string` is what triggers iOS 16's "Allow Paste" prompt when
        // the read is not attributable to a system paste control. A declined
        // prompt returns nil, as does a clipboard holding no text, and the
        // two cannot be told apart from here — so both answer ok=true with
        // "", and the ok=false case is left to the platforms that can report
        // a refusal (the browser). `hasStrings` is deliberately not checked
        // first: it avoids the prompt but also answers false for a pasteboard
        // the user has not yet allowed, which would turn every first Paste
        // into a silent no-op.
        let text = UIPasteboard.general.string ?? ""
        send(id: id, text: text, ok: true)
    }

    private func send(id: String, text: String, ok: Bool) {
        let object: [String: Any] = ["id": id, "text": text, "ok": ok]
        guard let data = try? JSONSerialization.data(withJSONObject: object),
              let payload = String(data: data, encoding: .utf8)
        else { return }
        report?("clipboard", payload)
    }
}
