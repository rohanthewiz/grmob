import Foundation
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
                // The inbound half of core.OpenURL: a URL the OS hands this
                // app, forwarded to Go as the "deeplink" host event and parsed
                // there. See core/deeplink.go on why the shell does not parse
                // it, and Info.plist's CFBundleURLTypes for the scheme.
                //
                // One callback for both cases, which is the difference from
                // Android: SwiftUI delivers the launch URL and a URL arriving
                // at a running app through the same modifier, so there is no
                // equivalent of onNewIntent to write. It is on the root view
                // rather than the Scene because a Scene-level .onOpenURL fires
                // before the first body evaluation on a cold launch, and the
                // app's subscriber lives in a hook slot that has not run yet.
                //
                // One thing a simulator run makes clear and a reader would
                // otherwise be surprised by: iOS puts a confirmation in front
                // of a custom-scheme link from an unknown source ("Open in
                // GrMobApp?"), so this is not the silent one-command hop
                // Android's `am start -d` is. That gate is the platform's and
                // is the reason a scheme is a convenience rather than a trust
                // boundary — an app shipping to users wants a verified
                // Universal Link, which does not prompt. Verified reaching Go
                // through the prompt: tapping Open lands on the lesson the URL
                // names.
                .onOpenURL { url in
                    // Serialised rather than interpolated into a literal: a
                    // URL can carry a quote or a backslash in its query, and
                    // "{\"url\":\"\(url)\"}" would hand Go malformed JSON
                    // that ReceiveHostEvent drops with a log nobody reads.
                    guard let data = try? JSONSerialization.data(
                            withJSONObject: ["url": url.absoluteString]),
                          let json = String(data: data, encoding: .utf8)
                    else { return }
                    runtime.hostEvent("deeplink", json)
                }
        }
        .onChange(of: scenePhase) { _, phase in
            AppLifecycle.report(phase, to: runtime)
            // Coming back from Settings is the only signal iOS gives for the
            // device-wide Location Services switch being turned back on — it
            // sends no authorization callback for that one. See
            // LocationSensor.retryIfArmed for the measurement behind that
            // sentence and for why this costs nothing when nothing is armed.
            //
            // The compass is refused by that same switch and has even less to
            // go on: it asks for no authorization, so there is no callback it
            // could have used instead. Both are no-ops unless a sensor was
            // actually running and actually killed.
            if phase == .active {
                LocationSensor.shared.retryIfArmed()
                HeadingSensor.shared.retryIfArmed()
            }
        }
    }
}
