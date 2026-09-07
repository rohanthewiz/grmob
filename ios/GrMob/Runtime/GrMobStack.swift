import CoreGraphics

/// Overlay placement for core.ZStack, as pure arithmetic.
///
/// Split out of GrMobStackLayout (Renderer.swift) for the reason
/// GrMobFlexSolver was split out of the flex layout: a SwiftUI `Layout` can
/// only be exercised by mounting it in a view hierarchy, which needs a
/// simulator, while the rules that actually matter here are decisions this
/// file can make on its own. It imports CoreGraphics and nothing else, so
/// `ios/verify` can run all of them on a plain macOS host.
///
/// # Where the line is
///
/// It started at the arithmetic — what the stack sizes to, and where a placed
/// layer lands — because those are functions from numbers to numbers and were
/// obviously testable. That left three decisions on the far side of it, in the
/// `Layout` conformance, where a type-check was the only thing looking at
/// them: measuring children with the *incoming* proposal rather than an
/// unspecified one, refusing to clamp the container to that proposal, and
/// re-proposing `bounds.size` at placement. None of the three is arithmetic
/// and all three are load-bearing — the first decides whether a greedy
/// background still covers anything, the second is the divergence this file
/// exists for — so the line moved to take them in.
///
/// What made that possible is GrMobStackLayer: a `LayoutSubview` is an opaque
/// proxy a test can never construct, but the two things the layout asks one
/// are a protocol two lines long. Renderer.swift now holds `subviews.map`, one
/// `sizeThatFits` and the `place()` call, and nothing else — the conversion
/// between `ProposedViewSize` and the proposal type below went to
/// GrMobStackBridge.swift, which `ios/verify` also runs. It is a separate file
/// rather than a second extension here for the reason this one gives about
/// itself: it needs `import SwiftUI`, and the sentence above about importing
/// CoreGraphics and nothing else has to stay true.
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

/// A size offer, as a layout hands one to a child.
///
/// SwiftUI's `ProposedViewSize` with the framework taken out: two optional
/// dimensions, where nil means "you decide" and a number means "this much is
/// available". It exists so the rules below can be *stated* — and executed —
/// in a file that imports nothing but CoreGraphics.
///
/// Restating a framework type is a cost, and it buys the one thing a
/// `Layout` cannot otherwise give: a `LayoutSubview` is an opaque proxy with
/// no public initializer, so a test can never construct one, and every rule
/// written in terms of `Subviews` is a rule that can only be checked by
/// mounting the view in a running app. Written against a protocol instead,
/// the same rules run on a plain macOS host against a fake that records what
/// it was asked. GrMobStackBridge.swift converts in both directions, and
/// `ios/verify` runs that conversion too — a `ProposedViewSize`, unlike a
/// `LayoutSubview`, is an ordinary public struct a check can construct.
public struct GrMobProposal: Equatable {
    /// nil is SwiftUI's `nil` dimension: unspecified, decide for yourself.
    public let width: CGFloat?
    public let height: CGFloat?

    public init(width: CGFloat?, height: CGFloat?) {
        self.width = width
        self.height = height
    }

    /// A fully stated offer — both dimensions fixed. This is what placement
    /// hands every layer, and the reason it is a named initializer is that
    /// "the size actually being drawn into" is the decision, not a conversion.
    public init(_ size: CGSize) {
        self.init(width: size.width, height: size.height)
    }
}

/// One layer of an overlay, as the arithmetic needs to see it.
///
/// Two questions, which are exactly the two a `LayoutSubview` answers for the
/// parts of the job that are decisions rather than drawing: how big are you if
/// I offer you this, and where did you ask to sit.
public protocol GrMobStackLayer {
    /// The size this layer reports for a given offer. Called more than once
    /// per pass, deliberately — see GrMobStackSolver.placements.
    func size(proposing proposal: GrMobProposal) -> CGSize

    /// The layer's own anchor, or nil for the centre. nil rather than
    /// `.center` all the way down, so "said nothing" and "asked for the
    /// centre" stay one state — see grMobStackAnchor.
    var anchor: GrMobStackAnchor? { get }
}

/// Where one layer goes, and what it is offered when it gets there.
///
/// A returned plan rather than a `place()` call inside the solver: placement
/// is the one part of the job that is SwiftUI's, and handing back the two
/// values it needs keeps every decision on this side of the line where it can
/// be read off and compared.
public struct GrMobLayerPlacement: Equatable {
    public let origin: CGPoint
    public let proposal: GrMobProposal

    public init(origin: CGPoint, proposal: GrMobProposal) {
        self.origin = origin
        self.proposal = proposal
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

    // MARK: - The same two rules, over layers rather than numbers
    //
    // Everything above is arithmetic and was always checkable. What follows is
    // the part that used to live in Renderer.swift's `Layout` conformance, and
    // it is there that the judgement calls are: *which* offer each layer is
    // measured with, and whether the container is allowed to shrink to fit.
    // Neither is arithmetic and both are invisible to a type-check, so they
    // are stated here against GrMobStackLayer and Renderer.swift keeps only
    // the conversion.

    /// What the stack reports for an offer: measure every layer with the offer
    /// it was given, then take the largest on each axis.
    ///
    /// Two decisions, and they pull in opposite directions:
    ///
    /// **The incoming proposal is passed through**, not `.unspecified`. That
    /// is what a SwiftUI ZStack does, and it is what keeps a greedy layer
    /// greedy: a background stating `maxWidth: .infinity` reports whatever it
    /// is offered, so a stack containing one still fills — exactly as it did
    /// when a filling frame was doing the placing. Measuring with an
    /// unspecified proposal instead would make every such layer report its
    /// intrinsic size, and backgrounds would quietly stop covering anything.
    ///
    /// **The result is not clamped to the proposal.** A stack whose content is
    /// larger than its parent's offer reports the content, which is what a
    /// SwiftUI ZStack reports and what the other three targets do. Clamping
    /// would have been a second behaviour change riding along with the one
    /// this file exists for.
    public static func containerSize<Layer: GrMobStackLayer>(
        layers: [Layer], proposing proposal: GrMobProposal
    ) -> CGSize {
        containerSize(children: layers.map { $0.size(proposing: proposal) })
    }

    /// Where every layer goes inside bounds the parent has already chosen.
    ///
    /// Each layer is re-measured against `bounds.size` rather than reused from
    /// the sizing pass, and is then offered that same size when it is placed.
    /// The parent is free to hand over a size it never asked about — a stack
    /// inside a frame, or one whose sibling took the slack — and an overlay
    /// offers every layer the whole box, so the offer that decides a layer's
    /// size has to be the one the box actually is. GrMobFlexLayout states the
    /// same rule at its own placeSubviews.
    ///
    /// The proposal travels back in the plan rather than being recomputed by
    /// the caller, so the size a layer was measured at and the size it is
    /// placed with cannot drift apart.
    public static func placements<Layer: GrMobStackLayer>(
        layers: [Layer], in bounds: CGRect
    ) -> [GrMobLayerPlacement] {
        let offer = GrMobProposal(bounds.size)
        return layers.map { layer in
            GrMobLayerPlacement(
                origin: origin(child: layer.size(proposing: offer), in: bounds,
                               anchor: layer.anchor ?? .center),
                proposal: offer)
        }
    }
}
