/// core.Focus / core.DismissKeyboard, for the two editors that host a UIKit
/// text view inside SwiftUI.
///
/// # Why this is not the modifier Renderer.swift uses
///
/// Every other focusable node in this runtime is a SwiftUI control, so its
/// focus command goes through `@FocusState`: the renderer sets a Bool and
/// SwiftUI moves the responder. Both editors are `UIViewRepresentable`s over a
/// `UITextView`, and a `.focused($flag)` on a representable does nothing —
/// SwiftUI has no idea the hosted view is a text control, and the responder
/// chain is UIKit's. So the command is applied to the responder directly.
///
/// # Why the epoch is remembered per editor rather than read per pass
///
/// A focus command is a *moment*, and `updateUIView` runs on every pass for
/// every reason — a style change, a keystroke coming back from Go, a parent
/// re-laying out. Without the memory, one `core.Focus` would re-take the
/// responder on every later pass for as long as the stamp stayed on the node,
/// which it does forever (see core/focus.go: nothing consumes the epoch).
///
/// It differs from `runCommand`'s adopt-on-first-sight in exactly one way, and
/// deliberately: this one *does* fire the first time it sees a non-zero epoch.
/// That is the case worth supporting — "push a screen and put the cursor in its
/// editor" issues the command in the handler that navigates, one pass before
/// the editor it names exists — and it is the same asymmetry the SwiftUI field
/// spells as an `.onAppear` beside its `.onChange`.
///
/// # Why it hops to the next runloop turn
///
/// `becomeFirstResponder()` on a view that is not yet in a window returns false
/// and does nothing, and on the first pass `updateUIView` runs before SwiftUI
/// has placed the representable. One async hop is the UIKit equivalent of the
/// web runtime's `requestAnimationFrame` deferral, and it is harmless on every
/// later pass, where the view is already live.
///
/// A command that still misses — the editor is inside a screen that went away
/// between the stamp and the hop — misses quietly. That is the honest outcome
/// and the one core/focus.go already describes for a ref whose node is not in
/// the tree.
#if canImport(UIKit)

import UIKit

/// One editor's focus-command memory. A struct rather than two loose fields so
/// that a coordinator gains the behaviour by holding one value, and so the
/// "have I applied this epoch" rule cannot be half-copied into the second
/// editor.
struct GrMobEditorFocus {
    /// The last epoch applied. Zero is safe as the initial value because zero
    /// is core's own sentinel for "no command has ever been issued" — the two
    /// props always travel together, and a renderer must never read a 0 epoch
    /// as an instruction.
    private var applied = 0

    /// Applies one stamp to `view`, at most once per epoch.
    ///
    /// `action` is core's three-valued vocabulary. "" means some *other* node
    /// is being focused and this one should do nothing: the platform takes the
    /// responder away on its own, and having both sides act would make the
    /// outcome depend on the order two effects ran in.
    ///
    /// "blur" is guarded on this view actually being first responder, because a
    /// dismiss reaches every focusable leaf on screen and exactly one of them
    /// holds the keyboard.
    mutating func apply(epoch: Int, action: String, to view: UITextView?) {
        guard epoch != 0, epoch != applied else { return }
        applied = epoch
        guard action == "focus" || action == "blur" else { return }
        DispatchQueue.main.async {
            guard let view else { return }
            if action == "focus" {
                view.becomeFirstResponder()
            } else if view.isFirstResponder {
                view.resignFirstResponder()
            }
        }
    }
}

#endif
