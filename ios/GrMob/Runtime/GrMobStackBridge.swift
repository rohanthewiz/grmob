import SwiftUI

/// The conversion between SwiftUI's size vocabulary and GrMobStack.swift's.
///
/// # Why it is a file of its own
///
/// `GrMobProposal` is `ProposedViewSize` with the framework taken out, and
/// GrMobStack.swift restates it for one reason: so the overlay's decisions can
/// be *run* on a plain macOS host rather than only type-checked. That reason
/// only holds while that file imports CoreGraphics and nothing else — a
/// `import SwiftUI` at the top of it would quietly make the claim in its own
/// doc comment false. So the two-way conversion lives here instead, next to
/// the type it converts and outside the file whose purity is the point.
///
/// # Why it is not in Renderer.swift, where it used to be
///
/// It was three expressions inside `GrMobStackLayout` and
/// `GrMobStackSubview`, and Renderer.swift is the half of this target that
/// nothing runs: `ios/verify` type-checks it and a simulator is the only thing
/// that mounts it. Every *decision* moved into GrMobStack.swift for that
/// reason, and the adapter stayed behind as "three lines with no decision in
/// them" — which was true, and still left the three lines unexecuted. A
/// swapped axis or a dimension dropped to nil compiles, and
/// `TestTheSwiftStackLayoutDelegatesToTheSolver` cannot see it: that check
/// catches a Layout that computes a size itself, which is a different mistake.
///
/// Moved here, both directions run under `ios/verify` against values it can
/// construct — `ProposedViewSize` is an ordinary public struct, unlike
/// `LayoutSubview`, which is the opaque proxy that forced the protocol in the
/// first place. What is left in Renderer.swift is `subviews.map`, one
/// `sizeThatFits` and one `place()`.
///
/// # nil is a value, not a missing one
///
/// Both types spell "you decide" as a nil dimension, so the conversion is a
/// pair of field copies and the whole risk is in the pairing. That is exactly
/// the kind of code a reader checks by looking at it and a compiler agrees
/// with either way, which is why the round trip is asserted rather than
/// argued: an unspecified offer that arrived as a number would size every
/// layer against a box nobody proposed, and a stated one that arrived as nil
/// would make every greedy background report its intrinsic size and stop
/// covering the stack.
extension GrMobProposal {
    /// A SwiftUI offer, as the solver sees one.
    init(_ proposal: ProposedViewSize) {
        self.init(width: proposal.width, height: proposal.height)
    }

    /// The same offer, back in SwiftUI's vocabulary — what a layer is measured
    /// with and what it is placed with.
    var proposedViewSize: ProposedViewSize {
        ProposedViewSize(width: width, height: height)
    }
}
