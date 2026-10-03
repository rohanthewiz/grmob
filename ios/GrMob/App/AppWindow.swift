import SwiftUI

/// The iOS half of grmob's window event: tells Go how big the app window is,
/// through the "window" host event (core/window.go).
///
///     GeometryReader size          ──▶ { width, height }   (points)
///     GeometryReader safeAreaInsets ──▶ { insets: {…} }
///     Window scene's interface style ──▶ { scheme: "light" | "dark" }
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
/// # The colour scheme
///
/// core.Window.ColorScheme is the system's light or dark mode. It used to be
/// the reader's `colorScheme` environment value, which was the system's while
/// nothing in the app set a scheme. Since N-085 the shell overrides the
/// window's interface style to match the Go tree's page (GrMobSurface.swift),
/// and the environment reports that override, so the system's scheme is read
/// from the window scene instead (GrMobSystemScheme), which the override does
/// not reach. The environment value is kept as the fallback for the moment
/// before the scene has been read; no override has been applied by then.
/// It is watched like the insets, because a switch in Control Centre changes
/// it with no resize.
///
/// Go dedupes a size it already has, so SwiftUI re-offering the same size (it
/// does around scene connection) costs a bridge call and nothing more. The
/// insets ride in the same report and are deduped with it.
enum AppWindow {
    /// Main-actor because `GrMobRuntime.hostEvent` is, and because the
    /// callers — onAppear and onChange on the reader — already run there.
    @MainActor
    static func report(_ size: CGSize, insets: EdgeInsets, scheme: ColorScheme, to runtime: GrMobRuntime) {
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
            // Two words, spelled as core.ColorSchemeLight and Dark are. A
            // scheme SwiftUI adds later reads as light rather than as a word
            // core would drop.
            "scheme": scheme == .dark ? "dark" : "light",
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
    /// The window's scheme: the system's only until GrMobWindowStyle first
    /// overrides it. See "The colour scheme".
    @Environment(\.colorScheme) private var windowScheme
    /// Read through Observation, so the scene's style changing re-evaluates
    /// this body and the onChange below sees it.
    private let system = GrMobSystemScheme.shared

    private var scheme: ColorScheme { system.scheme ?? windowScheme }

    var body: some View {
        GeometryReader { geo in
            Color.clear
                // onAppear for the first measurement, onChange for every
                // resize after it; onChange does not fire for the initial
                // value.
                .onAppear { AppWindow.report(geo.size, insets: geo.safeAreaInsets, scheme: scheme, to: runtime) }
                .onChange(of: geo.size) { _, size in
                    AppWindow.report(size, insets: geo.safeAreaInsets, scheme: scheme, to: runtime)
                }
                // The bars can move without the window resizing: a rotation
                // that puts the sensor housing on the other edge, a keyboard
                // that changes the bottom inset. onChange(of: size) alone
                // would miss those, so the insets are watched too.
                .onChange(of: geo.safeAreaInsets) { _, insets in
                    AppWindow.report(geo.size, insets: insets, scheme: scheme, to: runtime)
                }
                // Dark mode switched with no resize. See "The colour scheme".
                .onChange(of: scheme) { _, now in
                    AppWindow.report(geo.size, insets: geo.safeAreaInsets, scheme: now, to: runtime)
                }
        }
        .ignoresSafeArea()
    }
}
