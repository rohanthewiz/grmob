import SwiftUI

/// The iOS half of grmob's window event: tells Go how big the app window is,
/// through the "window" host event (core/window.go).
///
///     GeometryReader size          ──▶ { width, height }   (points)
///     GeometryReader safeAreaInsets ──▶ { insets: {…} }
///
/// No fold is ever reported. No iPhone or iPad has a hinge, so the payload
/// omits the "fold" key, which is exactly what core reads as "no fold". What
/// iOS *does* have is a window that changes size under a running app —
/// Split View, Slide Over and Stage Manager on iPad, rotation everywhere —
/// and that is the same size-class question a foldable asks, answered by the
/// same record, so a layout written for an unfolding phone adapts to an iPad
/// split without a line of platform code.
///
/// # Why a GeometryReader and not UIScreen
///
/// UIScreen.main.bounds is the display, and an iPad app in Split View has a
/// fraction of it. The window is what the tree is laid out in, and the size a
/// full-window view is offered is the window's. The reader sits in the root
/// view's background with the safe area ignored, so it measures the whole
/// window including the area under the status bar and home indicator — the
/// same window coordinates Android and the browser report in.
///
/// # The insets
///
/// Because the reader ignores the safe area, its proxy's `safeAreaInsets`
/// are the window's own — the status bar, the home indicator and, on a
/// notched phone in landscape, the sensor housing. They go to Go as
/// core.SafeInsets so a component can position itself against them; what
/// keeps ordinary content clear of the bars is still the SafeArea node, and
/// SwiftUI's own insetting behind it, not these numbers.
///
/// Go dedupes a size it already has, so SwiftUI re-offering the same size (it
/// does around scene connection) costs a bridge call and nothing more. The
/// insets ride in the same report and are deduped with it.
enum AppWindow {
    /// Main-actor because `GrMobRuntime.hostEvent` is, and because the
    /// callers — onAppear and onChange on the reader — already run there.
    @MainActor
    static func report(_ size: CGSize, insets: EdgeInsets, to runtime: GrMobRuntime) {
        // A zero size is the reader before layout, not a window; Go would
        // drop a zero-height report as meaningless anyway, but sending it
        // would briefly publish a zero-sized window to subscribers.
        guard size.width > 0, size.height > 0 else { return }
        // Serialized rather than interpolated, the same rule AppLifecycle
        // follows. The insets key below is what that bought: a nested object
        // added to the payload without touching how it is written, and
        // decoded by older core as absent rather than as a parse error.
        //
        // Physical edges, not leading/trailing: core.SafeInsets is defined
        // that way because the fold bounds beside it are, and a cutout is
        // where it is whatever the writing direction is. SwiftUI's EdgeInsets
        // is already physical here — the proxy reports the resolved values.
        let fields: [String: Any] = [
            "width": Double(size.width),
            "height": Double(size.height),
            "insets": [
                "top": Double(insets.top),
                "bottom": Double(insets.bottom),
                "left": Double(insets.leading),
                "right": Double(insets.trailing),
            ],
        ]
        guard let data = try? JSONSerialization.data(withJSONObject: fields),
              let payload = String(data: data, encoding: .utf8)
        else { return }
        runtime.hostEvent("window", payload)
    }
}

/// The measuring view: a clear GeometryReader for GrMobRoot's background.
struct AppWindowReader: View {
    let runtime: GrMobRuntime

    var body: some View {
        GeometryReader { geo in
            Color.clear
                // onAppear for the first measurement, onChange for every
                // resize after it; onChange does not fire for the initial
                // value.
                .onAppear { AppWindow.report(geo.size, insets: geo.safeAreaInsets, to: runtime) }
                .onChange(of: geo.size) { _, size in
                    AppWindow.report(size, insets: geo.safeAreaInsets, to: runtime)
                }
                // The bars can move without the window resizing: a rotation
                // that puts the sensor housing on the other edge, a keyboard
                // that changes the bottom inset. onChange(of: size) alone
                // would miss those, so the insets are watched too.
                .onChange(of: geo.safeAreaInsets) { _, insets in
                    AppWindow.report(geo.size, insets: insets, to: runtime)
                }
        }
        .ignoresSafeArea()
    }
}
