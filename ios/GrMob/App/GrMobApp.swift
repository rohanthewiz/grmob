import SwiftUI

/// The whole shell: build a runtime over the gomobile bridge, mount the Go
/// tree, hand the store to GrMobRoot. Everything the app *does* lives in Go
/// (the bound package registered via mobile.Register — see examples/mobileapp).
@main
struct GrMobApp: App {
    private let runtime: GrMobRuntime
    // Foreground/background, reported to Go as the "lifecycle" host event.
    // Read here, at the App, so it is the aggregate over every scene; see
    // AppLifecycle.swift.
    @Environment(\.scenePhase) private var scenePhase

    init() {
        // App.init runs on the main thread (SwiftUI's App protocol is
        // MainActor-isolated), satisfying GrMobRuntime.start's contract.
        // start() before the first body evaluation so the initial tree is
        // there on the very first frame — no empty-flash-then-mount.
        let bridge = GomobileBridge()
        let runtime = GrMobRuntime(bridge: bridge)
        // System events (toasts, external URLs, audio) are wired before
        // start() so an event emitted during the very first render pass has
        // a sink; without a listener Go drops them silently. The runtime is
        // built first because the audio player reports back through it, but
        // nothing renders until start(). See SystemEvents.swift.
        SystemEvents.attach(bridge, runtime: runtime)
        runtime.start()
        self.runtime = runtime
    }

    var body: some Scene {
        WindowGroup {
            GrMobRoot(runtime: runtime)
        }
        .onChange(of: scenePhase) { _, phase in
            AppLifecycle.report(phase, to: runtime)
        }
    }
}
