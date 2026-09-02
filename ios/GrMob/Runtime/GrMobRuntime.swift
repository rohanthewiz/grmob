import Foundation

/// The surface the Go side exposes, as seen from Swift.
///
/// This mirrors the gomobile-generated `Mobile*` functions one-to-one (see
/// mobile/bridge.go); it exists as a protocol so the runtime compiles and
/// tests without the generated xcframework linked. The app target provides
/// the real implementation (GomobileBridge).
///
/// Sendable is part of the contract, not a formality: Trigger* calls arrive
/// on the runtime's serial events queue while setListener is called from the
/// main thread, so an implementation must be safe to touch from both. The
/// gomobile functions are — cgo calls into a Go runtime that serializes
/// render passes behind its own mutex.
protocol GrMobBridge: Sendable {
    func renderInitial() -> String
    func triggerCallback(_ id: String) -> String
    func triggerTextCallback(_ id: String, _ value: String) -> String
    func triggerBoolCallback(_ id: String, _ value: Bool) -> String
    func triggerIntCallback(_ id: String, _ value: Int) -> String

    /// Registers the Go→native push target; called from a Go goroutine.
    func setListener(_ listener: @escaping (String) -> Void)

    /// Registers the sink for app→host system events: the transient chrome
    /// and OS hand-offs that are deliberately not part of the view tree —
    /// `core.ShowToast` and `core.OpenURL` today. See mobile/sysevents.go;
    /// `name` is the event kind and `payload` its data as a JSON object.
    ///
    /// Called from whichever Go goroutine emitted the event, so an
    /// implementation must hop to the main actor before touching UIKit — the
    /// same contract `setListener` carries.
    func setSystemEventListener(_ listener: @escaping (String, String) -> Void)

    /// Reports a host→app event that answers no registered callback — the
    /// audio player's status ticks today (see mobile/hostevents.go) — and
    /// returns the patches of the render it caused, exactly like the
    /// Trigger* calls. `name` is the event kind, `payload` its data as a
    /// JSON object.
    func reportHostEvent(_ name: String, _ payload: String) -> String
}

/// Wires the bridge to a TreeStore and owns the threading model.
///
/// Patches reach Swift on two paths — the synchronous return value of a
/// Trigger* call, and asynchronous pushes from Go goroutines (timers,
/// network, State.Set off-thread). The Go side guarantees each render's diff
/// is delivered on exactly one path, in order; our side of the contract is to
/// apply payloads in arrival order on one thread. Both paths therefore funnel
/// into `DispatchQueue.main.async { store.applyPatches(...) }` — the main
/// queue executes blocks in FIFO order, which *is* the ordering guarantee.
/// (Not `Task { @MainActor in ... }`: unstructured tasks carry no ordering
/// promise between one another, which would break the contract.)
///
///   UI event ─▶ events queue ─▶ MobileTrigger*() ─┐  (sync return)
///                                                 ├─▶ main.async ─▶ TreeStore ─▶ view update
///   Go goroutine (timer/State.Set) ── listener ───┘  (async push)
///
/// Trigger* calls run on a dedicated serial queue, not the main thread: a
/// bridge call spans a full Go render pass and may briefly block on the
/// render mutex, and the serial queue keeps events themselves ordered.
@MainActor
final class GrMobRuntime {
    let store = TreeStore()

    private let bridge: GrMobBridge
    private let events = DispatchQueue(label: "grmob-events")

    init(bridge: GrMobBridge) {
        self.bridge = bridge
    }

    /// Mounts the initial tree and opens the push channel. Call once, before
    /// the first body evaluation reads the store.
    func start() {
        store.mount(bridge.renderInitial())
        // Listener attaches after the initial mount so a pre-mount push can
        // never race tree construction; Go re-flushes pending changes on
        // attach, so nothing that happened in between is lost.
        bridge.setListener { [store] patches in
            DispatchQueue.main.async {
                MainActor.assumeIsolated { store.applyPatches(patches) }
            }
        }
    }

    func click(_ callbackID: String) {
        dispatch { $0.triggerCallback(callbackID) }
    }

    func textChanged(_ callbackID: String, _ value: String) {
        dispatch { $0.triggerTextCallback(callbackID, value) }
    }

    func toggled(_ callbackID: String, _ value: Bool) {
        dispatch { $0.triggerBoolCallback(callbackID, value) }
    }

    func intChanged(_ callbackID: String, _ value: Int) {
        dispatch { $0.triggerIntCallback(callbackID, value) }
    }

    /// Delivers a host event (a player status tick, say) to Go on the same
    /// serial queue as UI events, so it can never interleave with one, and
    /// applies the patches it produced the same way.
    func hostEvent(_ name: String, _ payload: String) {
        dispatch { $0.reportHostEvent(name, payload) }
    }

    private func dispatch(_ call: @escaping (GrMobBridge) -> String) {
        events.async { [bridge, store] in
            let patches = call(bridge)
            DispatchQueue.main.async {
                MainActor.assumeIsolated { store.applyPatches(patches) }
            }
        }
    }
}
