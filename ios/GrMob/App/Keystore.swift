import Foundation
import Security

/// The iOS half of Go's keystore package (keystore/keystore.go).
///
///     keystore.Save   ──▶ "keystore" {command: "save",   id, key, value}
///     keystore.Get    ──▶ "keystore" {command: "get",    id, key}
///     keystore.Delete ──▶ "keystore" {command: "delete", id, key}
///                     ◀── host event "keystore" {id, ok, found, value, error}
///
/// Every request is answered, failures included, because Go holds the
/// caller's callback until the id comes back and has no timeout to fall back
/// on (the same contract as Clipboard).
///
/// Each key is one Keychain generic-password item: service "grmob.keystore",
/// account = the key, data = the value's UTF-8 bytes. Accessible
/// AfterFirstUnlockThisDeviceOnly — readable by a background launch once the
/// device has been unlocked since boot, never migrated to another device by a
/// backup or iCloud Keychain. See the package comment in keystore.go for why
/// those two halves, and why Android lands on the same answers.
///
/// # Threading
///
/// The object is main-actor, like the other services, because that is where
/// `SystemEvents` delivers and where `report` (GrMobRuntime.hostEvent) has to
/// be called. The Keychain calls themselves are not made there: SecItem*
/// functions block on IPC to securityd, and Apple's guidance is to keep them
/// off the main thread. So each request runs on one serial queue — serial so
/// a Save followed by a Get reads what was saved — and its reply hops back to
/// the main actor to be reported.
///
/// In App/ beside the other services rather than in Runtime/; Security is
/// available on macOS too, so it could have gone either way, and App/ is where
/// SystemEvents looks for its handlers.
@MainActor
final class Keystore {
    static let shared = Keystore()

    /// The host-event reporter, set by `SystemEvents.attach`.
    var report: ((String, String) -> Void)?

    private let queue = DispatchQueue(label: "com.grmob.keystore", qos: .userInitiated)

    func handle(_ data: [String: Any]) {
        guard let id = data["id"] as? String, !id.isEmpty else { return }
        let command = data["command"] as? String ?? ""
        let key = data["key"] as? String ?? ""
        let value = data["value"] as? String ?? ""
        queue.async {
            let reply = KeychainStore.perform(id: id, command: command, key: key, value: value)
            DispatchQueue.main.async {
                MainActor.assumeIsolated { Keystore.shared.send(reply) }
            }
        }
    }

    private func send(_ reply: KeychainStore.Reply) {
        var object: [String: Any] = ["id": reply.id, "ok": reply.ok]
        if reply.ok, let found = reply.found {
            object["found"] = found
            if let value = reply.value { object["value"] = value }
        }
        if let error = reply.error { object["error"] = error }
        guard let data = try? JSONSerialization.data(withJSONObject: object),
              let payload = String(data: data, encoding: .utf8)
        else { return }
        report?("keystore", payload)
    }
}

/// The Keychain work, off the main actor. An enum of static functions because
/// it has no state but the one-time install check, which only the serial
/// queue in `Keystore` ever touches.
enum KeychainStore {
    struct Reply: Sendable {
        let id: String
        let ok: Bool
        var found: Bool? = nil
        var value: String? = nil
        var error: String? = nil
    }

    private static let service = "grmob.keystore"

    /// UserDefaults key recording that this install has already wiped any
    /// items a previous install of the app left behind.
    private static let installedMarker = "grmob.keystore.installed"

    /// Set once the install check has run in this process, so it costs one
    /// UserDefaults read per launch rather than one per request. Only read
    /// and written on `Keystore`'s serial queue.
    nonisolated(unsafe) private static var installChecked = false

    static func perform(id: String, command: String, key: String, value: String) -> Reply {
        // Go refuses an empty key before sending; this guards a host-side
        // caller, not Go. An empty account would otherwise be a real item.
        guard !key.isEmpty else { return Reply(id: id, ok: false, error: "empty key") }
        wipeIfFreshInstall()
        switch command {
        case "save": return save(id: id, key: key, value: value)
        case "get": return get(id: id, key: key)
        case "delete": return delete(id: id, key: key)
        // A newer Go with a command this shell predates: answered, so the
        // caller's callback is not stranded.
        default: return Reply(id: id, ok: false, error: "unknown command '\(command)'")
        }
    }

    /// The query that names one item. Everything else about the item —
    /// its data, its accessibility — is an attribute, not part of its identity.
    private static func itemQuery(_ key: String) -> [String: Any] {
        [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
        ]
    }

    private static func save(id: String, key: String, value: String) -> Reply {
        let attributes: [String: Any] = [
            kSecValueData as String: Data(value.utf8),
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
        ]
        // Update first, add on not-found: the canonical upsert. Delete-then-
        // add would also work but leaves a window where a crash loses the old
        // value without having stored the new one.
        var status = SecItemUpdate(itemQuery(key) as CFDictionary, attributes as CFDictionary)
        if status == errSecItemNotFound {
            let add = itemQuery(key).merging(attributes) { _, new in new }
            status = SecItemAdd(add as CFDictionary, nil)
        }
        guard status == errSecSuccess else { return failure(id, status) }
        return Reply(id: id, ok: true)
    }

    private static func get(id: String, key: String) -> Reply {
        var query = itemQuery(key)
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne
        var out: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &out)
        switch status {
        case errSecSuccess:
            // Every item under this service was written by save() from a Swift
            // String, so non-UTF-8 data means something else wrote it; that is
            // a failure to report, not a value to guess at.
            guard let data = out as? Data, let value = String(data: data, encoding: .utf8) else {
                return Reply(id: id, ok: false, error: "stored item is not UTF-8 text")
            }
            return Reply(id: id, ok: true, found: true, value: value)
        case errSecItemNotFound:
            return Reply(id: id, ok: true, found: false)
        default:
            return failure(id, status)
        }
    }

    private static func delete(id: String, key: String) -> Reply {
        let status = SecItemDelete(itemQuery(key) as CFDictionary)
        // Deleting what is not there succeeds: sign-out clears the token
        // whether or not one was saved.
        guard status == errSecSuccess || status == errSecItemNotFound else { return failure(id, status) }
        return Reply(id: id, ok: true)
    }

    /// Keychain items outlive the app: deleting it leaves them in place, and a
    /// reinstall would find the old token. Android deletes everything with the
    /// app, and an app should not have to know which platform it is on to know
    /// whether uninstall signs the user out — so the first request in a fresh
    /// install deletes this service's items. UserDefaults is the witness
    /// because it is in the app's container and goes with it on uninstall.
    ///
    /// The marker is written after the delete, so a crash between the two
    /// re-runs the (idempotent) wipe on the next launch instead of skipping it.
    private static func wipeIfFreshInstall() {
        guard !installChecked else { return }
        installChecked = true
        let defaults = UserDefaults.standard
        if defaults.bool(forKey: installedMarker) { return }
        let all: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
        ]
        let status = SecItemDelete(all as CFDictionary)
        if status != errSecSuccess && status != errSecItemNotFound {
            // Logged and carried on: a stale item is a privacy wart, not a
            // reason to fail the request that happened to come first.
            NSLog("GrMob: keystore install wipe failed: \(message(status))")
        }
        defaults.set(true, forKey: installedMarker)
    }

    private static func failure(_ id: String, _ status: OSStatus) -> Reply {
        Reply(id: id, ok: false, error: message(status))
    }

    /// The Security framework's own description plus the raw code, which is
    /// what a search for the failure needs (errSecInteractionNotAllowed is
    /// -25308 in every forum post about it).
    private static func message(_ status: OSStatus) -> String {
        let text = SecCopyErrorMessageString(status, nil) as String? ?? "OSStatus"
        return "\(text) (\(status))"
    }
}
