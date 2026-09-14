import GameController

/// The F-key half of core.AccessibilityKeyShortcuts: presses from a hardware
/// keyboard, read through the GameController framework and handed to
/// GrMobRuntime.pressFunctionKey.
///
/// # Why GameController, after the routes that do not work
///
/// SwiftUI cannot deliver an F-key: a KeyEquivalent built from AppKit's
/// function-key character compiles and never fires (see "F-keys are not
/// SwiftUI's" on grMobKeyChord). The UIKit routes were weighed, and one was
/// measured, on the iOS 26.5 simulator:
///
/// ```
///   route                           why not
///   ─────                           ───────
///   a UIKeyCommand                  found along the responder chain from the
///                                   first responder; with nothing focused
///                                   there is none, and the objects at the end
///                                   of the chain (window, scene, application,
///                                   app delegate) are SwiftUI's
///   a UIApplication subclass        named by NSPrincipalClass, which a
///   overriding sendEvent            SwiftUI App ignores: the running
///                                   application's class was measured as
///                                   SwiftUIApplication, and the subclass's
///                                   init never ran
///   GCKeyboard.keyChangedHandler    every key of every attached keyboard,
///                                   app-wide, with no responder involved
/// ```
///
/// That last one is the position MainActivity.dispatchKeyEvent holds on
/// Android and the window keydown listener holds on the web, which is what a
/// page-global F-key needs.
///
/// # What it does not do
///
/// It cannot take the key: GameController observes a press, and UIKit still
/// delivers it to whatever responder would have had it. An F-key has no action
/// of its own on iOS, so nothing is doubled in practice, where the web runtime
/// calls preventDefault. The handler reports a key's changes of state, not its
/// auto-repeat, so a held key is one press, as on the other two targets.
@MainActor
enum GrMobFunctionKeys {
    /// Listens on the keyboard attached now and on any that attaches later.
    /// `GCKeyboard.coalesced` is every attached keyboard as one, so one
    /// handler covers two keyboards at once, and the connect notification
    /// installs it when the first keyboard arrives after launch (a keyboard
    /// paired to an iPad, or the simulator's, connected once the app is up).
    static func attach(to runtime: GrMobRuntime) {
        if let keyboard = GCKeyboard.coalesced { listen(keyboard, runtime) }
        NotificationCenter.default.addObserver(forName: .GCKeyboardDidConnect, object: nil, queue: .main) {
            [weak runtime] note in
            guard let keyboard = note.object as? GCKeyboard else { return }
            MainActor.assumeIsolated {
                guard let runtime else { return }
                listen(keyboard, runtime)
            }
        }
    }

    /// F1 to F12, in order, so an index plus one is the number
    /// GrMobFunctionKeyChord carries.
    private static let functionRow: [GCKeyCode] = [
        .F1, .F2, .F3, .F4, .F5, .F6, .F7, .F8, .F9, .F10, .F11, .F12,
    ]

    private static func listen(_ keyboard: GCKeyboard, _ runtime: GrMobRuntime) {
        // The handler runs on the keyboard's handlerQueue, which is the main
        // queue unless something changes it; the hop keeps the tree read on
        // the main actor either way. Modifiers are read inside the handler,
        // while the press is current, as either key of each pair held.
        let row = functionRow
        keyboard.keyboardInput?.keyChangedHandler = { [weak runtime] input, _, keyCode, pressed in
            guard pressed, let index = row.firstIndex(of: keyCode) else { return }
            func held(_ left: GCKeyCode, _ right: GCKeyCode) -> Bool {
                (input.button(forKeyCode: left)?.isPressed ?? false)
                    || (input.button(forKeyCode: right)?.isPressed ?? false)
            }
            let chord = GrMobFunctionKeyChord(
                number: index + 1,
                control: held(.leftControl, .rightControl),
                alt: held(.leftAlt, .rightAlt),
                meta: held(.leftGUI, .rightGUI),
                shift: held(.leftShift, .rightShift))
            DispatchQueue.main.async {
                MainActor.assumeIsolated { _ = runtime?.pressFunctionKey(chord) }
            }
        }
    }
}
