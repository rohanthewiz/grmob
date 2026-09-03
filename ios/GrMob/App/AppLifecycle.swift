import SwiftUI

/// The iOS half of grmob's app-lifecycle event: tells Go whether the app is
/// on screen, through the "lifecycle" host event (core/lifecycle.go).
///
///     .active     ──▶ "active"
///     .inactive   ──▶ "inactive"
///     .background ──▶ "background"
///
/// The source is the App's `scenePhase`. Read at the App level it is the
/// aggregate over every scene — active while any scene is, background only
/// when all are — which for a single-window phone app is exactly "is the
/// user looking at this". Core's vocabulary *is* ScenePhase's, so the
/// mapping is a spelling change and nothing more; Android and the browser
/// map their coarser signals onto the same three words.
///
/// SwiftUI reports the phase on the main actor; `GrMobRuntime.hostEvent`
/// hops to the runtime's serial event queue, where a transition can never
/// interleave with a tap that is mid-flight. Go dedupes repeats of the
/// current state, so a phase that re-reports itself (which SwiftUI does on
/// occasion around a scene connecting) costs nothing downstream.
enum AppLifecycle {
    /// Main-actor because `GrMobRuntime.hostEvent` is, and because the
    /// caller — the App's `onChange(of: scenePhase)` closure — already runs
    /// there; the hop to the runtime's event queue happens inside hostEvent.
    @MainActor
    static func report(_ phase: ScenePhase, to runtime: GrMobRuntime) {
        let state: String
        switch phase {
        case .active: state = "active"
        case .inactive: state = "inactive"
        case .background: state = "background"
        // ScenePhase is not frozen; a phase this shell does not know is
        // not one Go could act on either, so it is dropped rather than
        // guessed at.
        @unknown default: return
        }
        // The payload is one key, the contract every host writes; Go reads
        // it in core.receiveLifecycle. Serialized as an object rather than
        // hand-built so the shape survives a second key being added.
        guard let data = try? JSONSerialization.data(withJSONObject: ["state": state]),
              let payload = String(data: data, encoding: .utf8)
        else { return }
        runtime.hostEvent("lifecycle", payload)
    }
}
