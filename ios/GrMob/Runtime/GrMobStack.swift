import CoreGraphics

/// Overlay placement for core.ZStack, as pure arithmetic.
///
/// Split out of GrMobStackLayout (Renderer.swift) for the reason
/// GrMobFlexSolver was split out of the flex layout: a SwiftUI `Layout` can
/// only be exercised by mounting it in a view hierarchy, which needs a
/// simulator, while both rules that actually matter here — what the stack
/// sizes to, and where a placed layer lands inside it — are functions from
/// numbers to numbers. This file imports CoreGraphics and nothing else, so
/// `ios/verify` can measure it on a plain macOS host.
///
/// # Why this exists at all
///
/// It is the fix for the one divergence core.StackAlign used to document.
///
/// SwiftUI has no per-child ZStack alignment — a ZStack's `alignment:` is the
/// stack's, not the layer's, and there is no `.align()` for a child the way
/// Compose's BoxScope has one — so a placed layer used to be wrapped in
/// `.frame(maxWidth: .infinity, maxHeight: .infinity, alignment:)`. That is
/// SwiftUI's own idiom and it works, with one cost: **a filling frame is
/// greedy**. An unsized stack with one aligned layer therefore grew to
/// whatever its parent proposed on iOS, where a Compose Box and a CSS grid
/// track both stay the size of their largest child. The stack's own doc
/// resolved it by asking every stack to pin its dimensions, which is good
/// advice and was not the same thing as agreeing.
///
/// A custom `Layout` places without filling, so nothing is wrapped and nothing
/// is greedy. The container reports the largest child on every axis — the
/// contract core.ZStack documents for all four targets — and each layer is put
/// where its anchor says inside those bounds.
///
/// ```
///   bounds 200 x 120, child 40 x 20, anchor (1.0, 0.0) = top-end
///
///   +-------------------------------+
///   |                        [ 40 ] |   x = 0 + (200 - 40) * 1.0 = 160
///   |                               |   y = 0 + (120 - 20) * 0.0 =   0
///   |                               |
///   +-------------------------------+
/// ```
public struct GrMobStackAnchor: Equatable {
    /// Unit fractions along each axis: 0 is leading/top, 1 is trailing/bottom.
    /// The centre is (0.5, 0.5), which is what an unplaced layer is given.
    public let x: CGFloat
    public let y: CGFloat

    public init(x: CGFloat, y: CGFloat) {
        self.x = x
        self.y = y
    }

    /// core.StackAlignCenter: the stack's contract and every layer's default.
    public static let center = GrMobStackAnchor(x: 0.5, y: 0.5)
}

/// Go's core.StackAlignment as a pair of unit fractions, or nil for the centre.
///
/// # Why fractions rather than a SwiftUI Alignment
///
/// This replaced `grMobStackAlignment`, which returned SwiftUI's `Alignment`
/// because the old implementation handed the value to a `.frame(alignment:)`.
/// A `Layout` places by coordinate instead, and `Alignment` cannot be converted
/// to one — it is an opaque pair of alignment guides, resolvable only by
/// SwiftUI while it lays a view out. Fractions are the same nine placements in
/// the form the arithmetic can use, and moving the vocabulary here is what lets
/// `ios/verify` check it: the SwiftUI type could only ever be compared against
/// itself.
///
/// The three constructs each have a nine-value 2D placement vocabulary and they
/// agree value for value — SwiftUI's `Alignment`, Compose's `Alignment`, CSS's
/// justify-self/align-self — which is what makes core.StackAlignment portable
/// where a flexbox property would not have been.
///
/// # The centre has no arm, deliberately
///
/// core.StackAlignCenter is the empty string: the field's zero value, carried
/// by every node in every tree. An arm for it would be an arm for "the
/// default", and mobile/verify's own check requires there not to be one —
/// nil here and `GrMobStackAnchor.center` at the call site keep the vocabulary
/// to the eight *stated* placements that core.StackAlignments() returns.
///
/// mobile/verify's coverage check holds these arms to that list, so a ninth
/// placement in Go fails there until it has a row here.
public func grMobStackAnchor(_ align: String) -> GrMobStackAnchor? {
    switch align {
    case "top-start": return GrMobStackAnchor(x: 0, y: 0)
    case "top": return GrMobStackAnchor(x: 0.5, y: 0)
    case "top-end": return GrMobStackAnchor(x: 1, y: 0)
    case "start": return GrMobStackAnchor(x: 0, y: 0.5)
    case "end": return GrMobStackAnchor(x: 1, y: 0.5)
    case "bottom-start": return GrMobStackAnchor(x: 0, y: 1)
    case "bottom": return GrMobStackAnchor(x: 0.5, y: 1)
    case "bottom-end": return GrMobStackAnchor(x: 1, y: 1)
    default: return nil
    }
}

/// The two numeric rules an overlay follows.
public enum GrMobStackSolver {
    /// What the stack sizes to: the largest child on each axis, independently.
    ///
    /// Per axis rather than per child — the widest child and the tallest child
    /// need not be the same one, and a stack of a wide short banner over a
    /// narrow tall column is as wide as the first and as tall as the second.
    /// That is what a Compose Box and a single-cell CSS grid track both do.
    ///
    /// Not clamped to the proposal. A SwiftUI ZStack reports the union of its
    /// children whether or not that fits what it was offered, and this replaced
    /// a ZStack: clamping would be a *second* behaviour change riding along
    /// with the one this file exists for, and it would silently resize any
    /// stack whose content is larger than its parent's offer.
    ///
    /// Empty is zero, which is what a container with nothing in it should
    /// occupy. `.zero` rather than the proposal for the same reason.
    public static func containerSize(children: [CGSize]) -> CGSize {
        guard !children.isEmpty else { return .zero }
        return CGSize(
            width: children.map(\.width).max() ?? 0,
            height: children.map(\.height).max() ?? 0)
    }

    /// Where a child's top-leading corner goes.
    ///
    /// The anchor is a fraction of the *slack* — the space the child does not
    /// occupy — so 0 pins to the leading edge, 1 to the trailing edge and 0.5
    /// centres. One expression covers all nine placements and the default,
    /// which is why the centre needs no arm anywhere.
    ///
    /// A child larger than the stack produces negative slack, and the same
    /// arithmetic then overhangs symmetrically about the anchor. That is the
    /// right answer rather than an edge case: a layer wider than the stack
    /// asked to sit at the trailing edge should have its trailing edge there.
    public static func origin(child: CGSize, in bounds: CGRect,
                              anchor: GrMobStackAnchor) -> CGPoint {
        CGPoint(
            x: bounds.minX + (bounds.width - child.width) * anchor.x,
            y: bounds.minY + (bounds.height - child.height) * anchor.y)
    }
}
