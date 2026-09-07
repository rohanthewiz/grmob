// Checks for GrMobStackSolver — the overlay arithmetic behind the iOS
// renderer's ZStack (GrMobStack.swift).
//
// The solver was split out of the SwiftUI Layout precisely so it could be
// checked here: a Layout can only be exercised by mounting it in a view
// hierarchy, which needs a simulator, while the rules an overlay follows are
// decisions the solver can make on its own. The first half of this file is the
// arithmetic — what the container sizes to, and where a placed layer lands.
// The second half is the three decisions the Layout used to keep to itself,
// reached through GrMobStackLayer; see checkStackLayoutRules.
//
// # What this is the check for
//
// Until this file existed, the iOS stack was the one place in the framework
// where a *documented* cross-target divergence had nothing pinning it. The
// placement was a `.frame(maxWidth: .infinity, maxHeight: .infinity,
// alignment:)` around the layer, which is greedy — an unsized stack with an
// aligned layer grew to its parent's proposal here and stayed the size of its
// largest child on the other three targets. The divergence was recorded in
// four doc comments and asserted nowhere, because `ios/verify` type-checks and
// replays and neither of those measures anything.
//
// Expected values are derived from what a Compose Box and a single-cell CSS
// grid do, so a test failing means this renderer disagrees with the Android
// renderer and the browser, not merely that the code changed.
//
// SwiftUI is imported for the third section alone — the adapter between
// ProposedViewSize and GrMobProposal (GrMobStackBridge.swift). Importing it
// here does not weaken what GrMobStack.swift claims about itself: that file
// still imports CoreGraphics and nothing else, which is what lets it be linked
// into a plain command-line binary. What SwiftUI contributes to this harness is
// one struct with two optional fields, never a view.
import CoreGraphics
import SwiftUI

private func nearPoint(_ a: CGPoint, _ b: CGPoint) -> Bool {
    abs(a.x - b.x) < 0.0001 && abs(a.y - b.y) < 0.0001
}

private func nearSize(_ a: CGSize, _ b: CGSize) -> Bool {
    abs(a.width - b.width) < 0.0001 && abs(a.height - b.height) < 0.0001
}

private func check(
    _ name: String, _ got: CGSize, _ want: CGSize, into problems: inout [String]
) {
    if !nearSize(got, want) { problems.append("\(name): got \(got), want \(want)") }
}

private func check(
    _ name: String, _ got: CGPoint, _ want: CGPoint, into problems: inout [String]
) {
    if !nearPoint(got, want) { problems.append("\(name): got \(got), want \(want)") }
}

func checkStackSolver() -> [String] {
    var problems: [String] = []

    // --- What the stack sizes to ------------------------------------------
    //
    // The bug this file exists for, stated as a size. Two layers, the larger
    // 160x160; the stack is 160x160 whatever else is in it and whatever the
    // parent offered. The old implementation had no answer of its own here —
    // the frame around a placed layer took the proposal — so this is the
    // assertion that would have caught the divergence.
    check("largest child on both axes",
          GrMobStackSolver.containerSize(children: [
              CGSize(width: 160, height: 160),
              CGSize(width: 20, height: 14),
          ]),
          CGSize(width: 160, height: 160), into: &problems)

    // Per axis, independently: the widest child and the tallest need not be
    // the same one. A Compose Box and a CSS grid track both do this, and a
    // solver that returned "the size of the biggest child" — by area, say —
    // would pass the case above and fail this one.
    check("the axes are independent",
          GrMobStackSolver.containerSize(children: [
              CGSize(width: 200, height: 10),
              CGSize(width: 10, height: 120),
          ]),
          CGSize(width: 200, height: 120), into: &problems)

    // Nothing in it occupies nothing. Not the proposal: an empty overlay that
    // filled its parent would be a transparent box swallowing taps.
    check("an empty stack is zero",
          GrMobStackSolver.containerSize(children: []),
          .zero, into: &problems)

    // --- Where a layer lands ----------------------------------------------
    //
    // The nine placements against a 200x120 box with a 40x20 layer: the slack
    // is 160x100, and each anchor takes its own fraction of it.
    let bounds = CGRect(x: 0, y: 0, width: 200, height: 120)
    let child = CGSize(width: 40, height: 20)

    for (align, want) in [
        ("top-start", CGPoint(x: 0, y: 0)),
        ("top", CGPoint(x: 80, y: 0)),
        ("top-end", CGPoint(x: 160, y: 0)),
        ("start", CGPoint(x: 0, y: 50)),
        ("end", CGPoint(x: 160, y: 50)),
        ("bottom-start", CGPoint(x: 0, y: 100)),
        ("bottom", CGPoint(x: 80, y: 100)),
        ("bottom-end", CGPoint(x: 160, y: 100)),
    ] {
        guard let anchor = grMobStackAnchor(align) else {
            problems.append("grMobStackAnchor(\"\(align)\") returned nil — every stated "
                + "placement in core.StackAlignments() needs an arm")
            continue
        }
        check("place \(align)",
              GrMobStackSolver.origin(child: child, in: bounds, anchor: anchor),
              want, into: &problems)
    }

    // The centre is the empty string and has no arm, which is core's own rule
    // (StackAlignments() excludes StackAlignCenter because it is the zero
    // value every node carries). The caller substitutes .center, so the
    // arithmetic still has to place it — dead centre of the slack.
    if grMobStackAnchor("") != nil {
        problems.append("grMobStackAnchor(\"\") has an arm; the centre is the catch-all's")
    }
    check("the centre is the middle of the slack",
          GrMobStackSolver.origin(child: child, in: bounds, anchor: .center),
          CGPoint(x: 80, y: 50), into: &problems)

    // An unknown value is a Go binary newer than this app, and the contract's
    // own default is where such a layer belongs.
    if grMobStackAnchor("diagonal") != nil {
        problems.append("grMobStackAnchor accepted an unknown placement")
    }

    // --- Two properties the arithmetic must keep --------------------------
    //
    // The bounds' origin is added, not assumed zero. A stack is placed inside
    // its parent, and a solver that returned coordinates relative to the stack
    // would put every layer of every nested overlay in the top-left of the
    // screen — which looks right in the one case that matters least.
    let offset = CGRect(x: 30, y: 45, width: 200, height: 120)
    check("placement is relative to the bounds' origin",
          GrMobStackSolver.origin(child: child, in: offset,
                                  anchor: grMobStackAnchor("top-start")!),
          CGPoint(x: 30, y: 45), into: &problems)

    // A layer bigger than the stack overhangs about its anchor rather than
    // being clamped: asked for the trailing edge, its trailing edge is there.
    // Negative slack, and the same expression.
    check("an oversized layer overhangs its anchor",
          GrMobStackSolver.origin(child: CGSize(width: 260, height: 20), in: bounds,
                                  anchor: grMobStackAnchor("end")!),
          CGPoint(x: -60, y: 50), into: &problems)

    problems += checkStackLayoutRules()
    problems += checkStackProposalBridge()
    return problems
}

// --- The three decisions the Layout used to keep to itself -----------------
//
// Everything above is arithmetic: numbers in, numbers out, and it was always
// checkable. What follows is the part that lived in Renderer.swift's `Layout`
// conformance until GrMobStackLayer existed, where a type-check was the only
// thing looking at it. None of the three is arithmetic and all three are
// load-bearing:
//
//	the incoming proposal reaches each layer   or a greedy background stops
//	                                           covering anything
//	the container is not clamped to it         or the divergence this whole
//	                                           file exists for comes back
//	placement re-proposes bounds.size          or a layer is sized against an
//	                                           offer the stack did not get
//
// What is being checked is what the *framework* decides, not what SwiftUI
// does with it: a real LayoutSubview answering sizeThatFits differently from
// RecordingLayer is a question only a simulator can settle, and it is a
// question about SwiftUI rather than about this code.

/// A layer that reports a fixed size and remembers every offer it was made.
///
/// A class, so the recording survives being handed to the solver by value —
/// and because a stack of identical struct layers could not be told apart
/// afterwards, which is exactly what the order check below needs.
private final class RecordingLayer: GrMobStackLayer {
    let reported: CGSize
    let anchor: GrMobStackAnchor?
    private(set) var offers: [GrMobProposal] = []

    init(_ reported: CGSize, anchor: GrMobStackAnchor? = nil) {
        self.reported = reported
        self.anchor = anchor
    }

    func size(proposing proposal: GrMobProposal) -> CGSize {
        offers.append(proposal)
        return reported
    }
}

/// A layer that takes whatever it is offered — a background stating
/// `maxWidth: .infinity`, which is the one shape the proposal decision is
/// about. An unspecified dimension falls back to its intrinsic size, exactly
/// as such a view does.
private final class GreedyLayer: GrMobStackLayer {
    let intrinsic: CGSize
    let anchor: GrMobStackAnchor?

    init(_ intrinsic: CGSize, anchor: GrMobStackAnchor? = nil) {
        self.intrinsic = intrinsic
        self.anchor = anchor
    }

    func size(proposing proposal: GrMobProposal) -> CGSize {
        CGSize(width: proposal.width ?? intrinsic.width,
               height: proposal.height ?? intrinsic.height)
    }
}

func checkStackLayoutRules() -> [String] {
    var problems: [String] = []

    // --- Sizing: every layer is measured with the offer the stack got ------
    let offered = GrMobProposal(width: 100, height: 50)
    let small = RecordingLayer(CGSize(width: 40, height: 20))
    let wide = RecordingLayer(CGSize(width: 300, height: 10))
    let sized = GrMobStackSolver.containerSize(layers: [small, wide], proposing: offered)

    for (i, layer) in [small, wide].enumerated() {
        if layer.offers != [offered] {
            problems.append("sizing: layer \(i) was offered \(layer.offers), want "
                + "exactly one \(offered) — a layout that measures with .unspecified "
                + "makes every greedy background report its intrinsic size and stop "
                + "covering the stack")
        }
    }

    // And the result is the content, not the offer. This is the divergence
    // GrMobStack.swift was written for, stated as the clamp that must not
    // happen: 300 wide under a 100-wide proposal.
    check("sizing is not clamped to the proposal", sized,
          CGSize(width: 300, height: 20), into: &problems)

    // An unspecified offer travels through unchanged rather than being
    // substituted for a number somewhere in the middle.
    let unspecified = GrMobProposal(width: nil, height: nil)
    let asked = RecordingLayer(CGSize(width: 12, height: 8))
    _ = GrMobStackSolver.containerSize(layers: [asked], proposing: unspecified)
    if asked.offers != [unspecified] {
        problems.append("sizing: an unspecified proposal reached the layer as "
            + "\(asked.offers) — nil is SwiftUI's \"you decide\" and inventing a "
            + "number for it would size every layer against a box nobody offered")
    }

    // A greedy layer still fills, which is the property the pass-through is
    // for: the stack around it is as big as the offer.
    check("a greedy layer still makes the stack fill",
          GrMobStackSolver.containerSize(
              layers: [GreedyLayer(CGSize(width: 10, height: 10))], proposing: offered),
          CGSize(width: 100, height: 50), into: &problems)

    // --- Placement: the bounds are the offer -------------------------------
    let bounds = CGRect(x: 30, y: 45, width: 200, height: 120)
    let box = GrMobProposal(bounds.size)
    let corner = RecordingLayer(CGSize(width: 40, height: 20),
                                anchor: grMobStackAnchor("bottom-end"))
    let middle = RecordingLayer(CGSize(width: 40, height: 20))
    let plan = GrMobStackSolver.placements(layers: [corner, middle], in: bounds)

    if plan.count != 2 {
        problems.append("placement: \(plan.count) placements for 2 layers")
        return problems
    }

    for (i, layer) in [corner, middle].enumerated() {
        if layer.offers != [box] {
            problems.append("placement: layer \(i) was measured with \(layer.offers), "
                + "want exactly one \(box) — the parent may hand over a size it never "
                + "asked sizeThatFits about, so the offer that decides a layer's size "
                + "has to be the box it is actually being drawn into")
        }
        if plan[i].proposal != box {
            problems.append("placement: layer \(i) is placed with proposal "
                + "\(plan[i].proposal), want \(box) — a layer measured at one size and "
                + "placed with another is free to draw at the second")
        }
    }

    // In subview order, and each at its own anchor. Two layers of identical
    // size with different anchors, so a plan that reversed them or applied one
    // anchor to both is caught.
    check("placement keeps subview order (anchored layer)", plan[0].origin,
          CGPoint(x: 30 + 160, y: 45 + 100), into: &problems)
    check("placement keeps subview order (unanchored layer)", plan[1].origin,
          CGPoint(x: 30 + 80, y: 45 + 50), into: &problems)

    // A greedy layer placed in the box fills it and therefore lands at the
    // origin whatever anchor it carries — zero slack, so the anchor cannot
    // move it. The case that would fail if placement re-proposed the *sizing*
    // proposal instead of the bounds.
    let greedyPlan = GrMobStackSolver.placements(
        layers: [GreedyLayer(CGSize(width: 10, height: 10),
                             anchor: grMobStackAnchor("bottom-end"))], in: bounds)
    check("a greedy layer fills the bounds and cannot be anchored away",
          greedyPlan[0].origin, CGPoint(x: 30, y: 45), into: &problems)

    // No layers is no placements, and no crash. An overlay with nothing in it
    // is a real tree — core.For over an empty slice.
    let empty: [RecordingLayer] = []
    if !GrMobStackSolver.placements(layers: empty, in: bounds).isEmpty {
        problems.append("placement: an empty stack produced placements")
    }

    return problems
}

// --- The adapter, which used to be unexecuted ------------------------------
//
// `GrMobProposal.init(_ proposal: ProposedViewSize)` and `proposedViewSize`
// are the whole of the conversion between SwiftUI's size vocabulary and the
// solver's. They lived inside Renderer.swift as three field-copying
// expressions, which put them on the one side of this target that nothing
// runs — a swapped axis or a dimension dropped to nil compiles, draws, and is
// invisible to `TestTheSwiftStackLayoutDelegatesToTheSolver`, whose subject is
// a Layout that computes a size *itself*.
//
// They are checkable here for the reason GrMobStackLayer was written: the
// obstacle was never SwiftUI, it was `LayoutSubview` in particular.
// `ProposedViewSize` is an ordinary public struct with a public initializer,
// so both directions can be handed real values and read back.
//
// Both stakes are visible in the table below, and they pull opposite ways:
//
//	an unspecified offer arriving as a number   every layer is sized against
//	                                            a box nobody proposed
//	a stated offer arriving as nil              every greedy background reports
//	                                            its intrinsic size and stops
//	                                            covering the stack

private func check(
    _ name: String, _ got: GrMobProposal, _ want: GrMobProposal, into problems: inout [String]
) {
    if got != want { problems.append("\(name): got \(got), want \(want)") }
}

private func check(
    _ name: String, _ got: ProposedViewSize, _ want: ProposedViewSize,
    into problems: inout [String]
) {
    if got != want {
        problems.append("\(name): got (\(String(describing: got.width)), "
            + "\(String(describing: got.height))), want (\(String(describing: want.width)), "
            + "\(String(describing: want.height)))")
    }
}

func checkStackProposalBridge() -> [String] {
    var problems: [String] = []

    // Asymmetric numbers on purpose: 100x50 catches a swap, 100x100 would not.
    check("a stated offer reaches the solver",
          GrMobProposal(ProposedViewSize(width: 100, height: 50)),
          GrMobProposal(width: 100, height: 50), into: &problems)
    check("a stated offer goes back to SwiftUI",
          GrMobProposal(width: 100, height: 50).proposedViewSize,
          ProposedViewSize(width: 100, height: 50), into: &problems)

    // nil is a value in both vocabularies and means the same thing in both:
    // "you decide". `.unspecified` is SwiftUI's own spelling of the pair.
    check("an unspecified offer reaches the solver as nil",
          GrMobProposal(ProposedViewSize.unspecified),
          GrMobProposal(width: nil, height: nil), into: &problems)
    check("an unspecified offer goes back as .unspecified",
          GrMobProposal(width: nil, height: nil).proposedViewSize,
          ProposedViewSize.unspecified, into: &problems)

    // One axis stated and one not, both ways round. This is the case a swap
    // survives the two symmetric checks above: a flex child measured on its
    // cross axis alone is offered exactly this shape.
    check("a half-stated offer keeps its axis (width)",
          GrMobProposal(ProposedViewSize(width: 100, height: nil)),
          GrMobProposal(width: 100, height: nil), into: &problems)
    check("a half-stated offer keeps its axis (height)",
          GrMobProposal(ProposedViewSize(width: nil, height: 50)),
          GrMobProposal(width: nil, height: 50), into: &problems)
    check("a half-stated offer goes back on its own axis",
          GrMobProposal(width: nil, height: 50).proposedViewSize,
          ProposedViewSize(width: nil, height: 50), into: &problems)

    // Zero is a real offer and not an absence — a layer in a collapsed
    // container is proposed it — so a conversion treating it as "unspecified"
    // would hand a greedy background its intrinsic size in the one case where
    // the stack has no room for it.
    check("a zero offer is stated, not unspecified",
          GrMobProposal(ProposedViewSize(width: 0, height: 0)),
          GrMobProposal(width: 0, height: 0), into: &problems)

    // The round trip, over the four nil combinations at once: whatever the
    // solver was handed is what SwiftUI gets back when a layer is placed with
    // it, which is the property placeSubviews rests on.
    for offer in [ProposedViewSize(width: 200, height: 120),
                  ProposedViewSize(width: 200, height: nil),
                  ProposedViewSize(width: nil, height: 120),
                  ProposedViewSize.unspecified] {
        check("round trip", GrMobProposal(offer).proposedViewSize, offer, into: &problems)
    }

    // And the named initializer the placement rule uses: bounds.size is fully
    // stated by construction, so `GrMobProposal(bounds.size).proposedViewSize`
    // is what every layer is actually offered at placement.
    check("bounds become a fully stated offer",
          GrMobProposal(CGSize(width: 200, height: 120)).proposedViewSize,
          ProposedViewSize(width: 200, height: 120), into: &problems)

    return problems
}
