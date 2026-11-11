import Foundation
import Observation
import os

/// Holds the live node tree and applies Go's reconciler patches to it.
///
/// Patches are resolved against the *current* tree at apply time by walking
/// the positional path, so there is no stale-path cache to drift out of sync
/// after structural changes. The ordering guarantees the Go side documents
/// (patches applied in emitted order; sibling removals arrive
/// highest-index-first) are exactly what makes this walk safe within a batch.
///
/// Threading: @MainActor — every mutation happens on the main thread.
/// GrMobRuntime funnels both delivery paths (synchronous event returns and
/// async pushes) through DispatchQueue.main, which also preserves the
/// bridge's arrival-order contract.
@MainActor
@Observable
final class TreeStore {
    private(set) var root: GrMobNode?

    private let log = Logger(subsystem: "com.grmob", category: "TreeStore")

    /// Called when this tree is found to have drifted from Go's: a patch
    /// whose path resolves to nothing, or a payload that is not a patch
    /// list. GrMobRuntime answers it with mobile.Resync, whose replace of
    /// "root" puts Go's tree back (N-133). Patches are positional and
    /// incremental, so a tree that is wrong once stays wrong; the log line
    /// that used to say "the next full replace re-synchronizes" was true,
    /// but nothing ever asked for one.
    @ObservationIgnored var onDrift: (@MainActor () -> Void)?

    /// Once per drift, not once per symptom: the stale patches already
    /// queued behind the one that dangled will dangle too, and each asking
    /// again would only fetch the same tree again. Cleared by a replace of
    /// "root", which is the answer (or a navigation, which resets it too).
    @ObservationIgnored private var resyncPending = false

    private func drifted() {
        if resyncPending { return }
        resyncPending = true
        onDrift?()
    }

    /// Mounts the initial full tree (the RenderInitial payload).
    func mount(_ json: String) {
        guard json.drop(while: \.isWhitespace).hasPrefix("{"),
              let data = json.data(using: .utf8),
              let obj = (try? JSONSerialization.jsonObject(with: data)) as? [String: Any]
        else {
            // Not a tree — most likely a Go-side error report. Surface it
            // instead of crashing on the parse and burying the real failure.
            log.error("initial payload is not a tree: \(json, privacy: .public)")
            return
        }
        root = GrMobNode.parse(obj)
    }

    /// Applies one patch batch (the RenderAgain / push payload).
    func applyPatches(_ json: String) {
        guard let data = json.data(using: .utf8),
              let patches = (try? JSONSerialization.jsonObject(with: data)) as? [Any]
        else {
            log.error("patch payload is not an array: \(json, privacy: .public)")
            drifted()
            return
        }
        for case let p as [String: Any] in patches {
            apply(type: p["Type"] as? String ?? "",
                  path: p["TargetID"] as? String ?? "",
                  changes: p["Changes"])
        }
    }

    private func apply(type: String, path: String, changes: Any?) {
        switch type {
        case "update-props":
            // A miss used to be silent here (optional chaining), and an
            // update is the commonest patch, so most drift went unseen.
            guard let node = resolve(path) else { return warn(type, path) }
            node.props = changes as? [String: Any] ?? [:]

        case "update-style":
            guard let node = resolve(path) else { return warn(type, path) }
            node.style = GrMobStyle.parse(changes as? [String: Any])

        case "replace":
            guard let obj = changes as? [String: Any] else { return }
            let node = GrMobNode.parse(obj)
            guard let (parent, index) = parentOf(path) else {
                // Path is "root" itself: swap the whole tree. That is a
                // resync's answer, so whatever had drifted is gone now.
                if path == Self.rootPath {
                    root = node
                    resyncPending = false
                } else {
                    warn(type, path)
                }
                return
            }
            if parent.children.indices.contains(index) {
                parent.children[index] = node
            } else {
                warn(type, path)
            }

        // "add" targets the slot the node should occupy; "add-child" targets
        // the parent and always appends. Both reduce to an insert clamped to
        // the current child count.
        case "add":
            guard let obj = changes as? [String: Any] else { return }
            guard let (parent, index) = parentOf(path) else { return warn(type, path) }
            parent.children.insert(GrMobNode.parse(obj), at: min(max(index, 0), parent.children.count))

        case "add-child":
            guard let obj = changes as? [String: Any] else { return }
            guard let parent = resolve(path) else { return warn(type, path) }
            parent.children.append(GrMobNode.parse(obj))

        case "remove", "remove-child":
            guard let (parent, index) = parentOf(path) else { return warn(type, path) }
            if parent.children.indices.contains(index) {
                parent.children.remove(at: index)
            } else {
                warn(type, path)
            }

        default:
            warn(type, path)
        }
    }

    /// Walks a positional path ("root/0/2") to its node, or nil if it dangles.
    private func resolve(_ path: String) -> GrMobNode? {
        guard var node = root else { return nil }
        if path == Self.rootPath { return node }
        for seg in path.dropFirst(Self.rootPath.count + 1).split(separator: "/") {
            guard let idx = Int(seg), node.children.indices.contains(idx) else { return nil }
            node = node.children[idx]
        }
        return node
    }

    /// Resolves a path to (parent node, child index); nil for "root" or a dangling path.
    private func parentOf(_ path: String) -> (GrMobNode, Int)? {
        guard let cut = path.lastIndex(of: "/"),
              let index = Int(path[path.index(after: cut)...]),
              let parent = resolve(String(path[..<cut]))
        else { return nil }
        return (parent, index)
    }

    private func warn(_ type: String, _ path: String) {
        // A dangling patch means the Go and Swift trees disagree — log loudly
        // rather than crash, and ask Go for its tree (drifted).
        log.warning("patch \(type, privacy: .public) could not resolve \(path, privacy: .public)")
        drifted()
    }

    private static let rootPath = "root"
}
