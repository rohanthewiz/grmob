import Observation
import SwiftUI
import UIKit

// The shell's surface and the status bar, both taken from the Go tree rather
// than from the system's dark mode (N-085).
//
// The window is the Go app's page. An app that does not follow
// core.Window.ColorScheme stays light on a dark iPhone, and two things here
// followed the system all the same:
//
//   - The surface. SwiftUI's window background is systemBackground, black in
//     dark mode, so a light app drew its theme's dark ink on black (seen on
//     the simulator, iOS 26.5: lesson 1.1's title black on black, its prose
//     dark grey on black). The screens paint no background of their own;
//     the host's surface is meant to be the theme's page.
//   - The status bar, and every piece of native chrome (a TextField's caret
//     and placeholder, a Menu, a sheet's grabber), which take the window's
//     interface style, i.e. the system's.
//
// Both now come from the tree, by the rule the Android shell uses
// (GrMobSurface.kt):
//
//   the bars' colour = the innermost painted SafeArea's Background
//                      (the SafeArea arm paints it under the bars)
//                      ?? the root node's Background
//                      ?? shellPage (core.DefaultTheme's Background)
//   the surface      = the root node's Background ?? shellPage,
//                      painted behind the tree, under the bars
//   the window style = .dark over a dark bars' colour, .light otherwise
//
// # One style where Android has two
//
// Android sets the bar icons alone (WindowInsetsController) and leaves the
// window theme light. iOS has no such split under the SwiftUI App lifecycle:
// the status bar's .default style follows the root view controller's
// interface style, and that controller is SwiftUI's own hosting controller,
// which an App cannot subclass to answer preferredStatusBarStyle. So the
// window's override carries both, and native chrome follows the colour
// under the bars. That is the page in the shapes the framework builds:
// comps.Screen forwards its Background to its SafeArea, so a painted SafeArea
// is a screen's page rather than a header strip.
//
// # The record still reads the system
//
// core.Window.ColorScheme is the system's scheme, not the shell's, and an
// app that follows it depends on that. A window override is visible to
// everything inside the window — SwiftUI's colorScheme environment included,
// which is what AppWindowReader used to read — and `preferredColorScheme`
// would be the same override spelled in SwiftUI. Trait overrides flow down
// the hierarchy, though, not up: the window scene above the window still
// carries the system's style. GrMobSystemScheme reads it there, and watches
// it, so the report is unchanged by anything this file does to the window.

/// The page colour behind a tree whose root states none: core.DefaultTheme's
/// Background (#FFFFFF), and the colour of every light bundled theme's page.
let grMobShellPage = Color.white

/// A painted SafeArea's Background, carried up to GrMobRoot as the colour
/// under the bars.
///
/// Innermost wins, which a plain `.preference` would not give: a preference
/// set on a view replaces whatever its subtree reported, so an outer
/// SafeArea would hide an inner one. The SafeArea arm uses
/// `transformPreference` and fills the value only when nothing below it did.
/// Between siblings the later one wins, which is reduce's order.
struct GrMobBarSurfaceKey: PreferenceKey {
    static var defaultValue: Color? { nil }
    static func reduce(value: inout Color?, nextValue: () -> Color?) {
        value = nextValue() ?? value
    }
}

extension View {
    /// The SafeArea arm's claim on the bars; see GrMobBarSurfaceKey.
    func grMobClaimsBars(_ background: Color?) -> some View {
        transformPreference(GrMobBarSurfaceKey.self) { value in
            if value == nil { value = background }
        }
    }
}

/// Whether a colour reads as dark: WCAG relative luminance at or below 0.5,
/// the threshold Compose's `luminance()` is compared with on Android, so the
/// two shells call the same colours dark.
func grMobIsDark(_ color: Color) -> Bool {
    var r: CGFloat = 0, g: CGFloat = 0, b: CGFloat = 0, a: CGFloat = 0
    guard UIColor(color).getRed(&r, green: &g, blue: &b, alpha: &a) else { return false }
    // sRGB to linear light, then the Rec. 709 weights.
    func linear(_ c: CGFloat) -> CGFloat {
        c <= 0.04045 ? c / 12.92 : pow((c + 0.055) / 1.055, 2.4)
    }
    let luminance = 0.2126 * linear(r) + 0.7152 * linear(g) + 0.0722 * linear(b)
    return luminance <= 0.5
}

/// The system's own light or dark mode, read from the window scene so that
/// the window override GrMobWindowStyle applies cannot hide it (see "The
/// record still reads the system" above).
///
/// nil until the first GrMobWindowStyle view reaches a window. AppWindowReader
/// falls back to its colorScheme environment until then, which is still the
/// system's at that point: the override is applied by the same call that
/// fills this in, and after it.
@Observable
@MainActor
final class GrMobSystemScheme {
    static let shared = GrMobSystemScheme()
    private(set) var scheme: ColorScheme?

    func read(_ scene: UIWindowScene) {
        let now: ColorScheme = scene.traitCollection.userInterfaceStyle == .dark ? .dark : .light
        if scheme != now { scheme = now }
    }
}

/// Applies the window's interface style from the Go tree, and reads the
/// system's scheme from the scene on the way. A zero-size UIKit view in
/// GrMobRoot's background: SwiftUI gives no handle on the hosting window, and
/// a view inside it is the shortest way to reach one.
struct GrMobWindowStyle: UIViewRepresentable {
    let dark: Bool

    func makeUIView(context: Context) -> Probe {
        let view = Probe()
        view.isUserInteractionEnabled = false
        view.isAccessibilityElement = false
        return view
    }

    func updateUIView(_ view: Probe, context: Context) {
        view.dark = dark
    }

    final class Probe: UIView {
        var dark = false {
            didSet { if dark != oldValue { apply() } }
        }
        private weak var watchedScene: UIWindowScene?
        private var registration: UITraitChangeRegistration?

        override func didMoveToWindow() {
            super.didMoveToWindow()
            watch(window?.windowScene)
            apply()
        }

        /// Reads the scene's style now and on every change. The scene is
        /// re-watched only when it differs, since didMoveToWindow also fires
        /// on the way out (window nil), and a view can be re-parented.
        private func watch(_ scene: UIWindowScene?) {
            guard let scene, scene !== watchedScene else { return }
            if let old = watchedScene, let registration {
                old.unregisterForTraitChanges(registration)
            }
            watchedScene = scene
            // Read before the first apply(): the scene does not see the
            // window's override either way, but the order keeps
            // GrMobSystemScheme's comment true without relying on that.
            GrMobSystemScheme.shared.read(scene)
            registration = scene.registerForTraitChanges([UITraitUserInterfaceStyle.self]) {
                (scene: UIWindowScene, _: UITraitCollection) in
                GrMobSystemScheme.shared.read(scene)
            }
        }

        /// The override itself. Only the style this tree asks for is ever
        /// set; `.unspecified` (follow the system) is never written back,
        /// because following the system is the behaviour being replaced.
        private func apply() {
            guard let window else { return }
            let style: UIUserInterfaceStyle = dark ? .dark : .light
            if window.overrideUserInterfaceStyle != style {
                window.overrideUserInterfaceStyle = style
            }
        }
    }
}
