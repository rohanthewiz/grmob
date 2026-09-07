// Checks for GrMobStackSolver — the overlay arithmetic behind the iOS
// renderer's ZStack (GrMobStack.swift).
//
// The solver was split out of the SwiftUI Layout precisely so it could be
// checked here: a Layout can only be exercised by mounting it in a view
// hierarchy, which needs a simulator, while both rules an overlay has — what
// the container sizes to, and where a placed layer lands — are functions from
// numbers to numbers.
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
import CoreGraphics

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

    return problems
}
