// Checks for GrMobFlexSolver — the CSS flex arithmetic behind the iOS
// renderer's Row/Column layout (GrMobFlex.swift).
//
// The solver was split out of the SwiftUI Layout precisely so it could be
// checked here: a Layout can only be exercised by mounting it in a view
// hierarchy, which needs a simulator, while every rule that is easy to get
// wrong is a function from numbers to numbers.
//
// Expected values are derived from the CSS spec rather than from the
// implementation, so a test failing means the renderer disagrees with the
// Android renderer and the browser, not merely that the code changed.
import CoreGraphics

private func near(_ a: CGFloat, _ b: CGFloat) -> Bool { abs(a - b) < 0.0001 }

private func check(
    _ name: String, _ got: [CGFloat], _ want: [CGFloat], into problems: inout [String]
) {
    guard got.count == want.count, zip(got, want).allSatisfy(near) else {
        problems.append("\(name): got \(got), want \(want)")
        return
    }
}

private func check(
    _ name: String, _ got: CGFloat, _ want: CGFloat, into problems: inout [String]
) {
    if !near(got, want) { problems.append("\(name): got \(got), want \(want)") }
}

func checkFlexSolver() -> [String] {
    var problems: [String] = []
    let plain = GrMobFlexSolver(spacing: 0, justify: "")

    // --- FlexGrow is proportional, not equal ------------------------------
    //
    // The bug this whole layout exists to fix. Two growers, weights 3 and 1,
    // in a 400pt container with no content: CSS gives 300/100. The infinity
    // frames this replaced gave 200/200.
    check("grow 3:1",
          plain.resolve(main: 400, bases: [0, 0], weights: [3, 1]).mains,
          [300, 100], into: &problems)

    // Growth is added *on top of* each child's base size — flex-grow divides
    // the leftover, it does not set the size. 100 leftover split 3:1 onto
    // bases of 50 and 150.
    check("grow adds to the base",
          plain.resolve(main: 300, bases: [50, 150], weights: [3, 1]).mains,
          [125, 175], into: &problems)

    // A single grower absorbs everything left over; a zero-weight sibling
    // keeps its base. This is the common case (a label and a spacer-ish
    // field) and the one the old approach happened to get right.
    check("one grower absorbs the slack",
          plain.resolve(main: 200, bases: [40, 0], weights: [0, 1]).mains,
          [40, 160], into: &problems)

    // Spacing is consumed before growth: 200 - 40 - 10 = 150.
    check("gap comes out of the free space",
          GrMobFlexSolver(spacing: 10, justify: "")
              .resolve(main: 200, bases: [40, 0], weights: [0, 1]).mains,
          [40, 150], into: &problems)

    // --- Shrink -----------------------------------------------------------
    //
    // Overflow shrinks in proportion to base size (flex-shrink: 1, the CSS
    // default): 300 of content into 200 means each child keeps two thirds.
    check("shrink is proportional to base",
          plain.resolve(main: 200, bases: [100, 200], weights: [0, 0]).mains,
          [200.0 / 3, 400.0 / 3], into: &problems)

    // Shrink applies even to growers — free space is negative, so there is
    // nothing to grow into.
    check("growers shrink too",
          plain.resolve(main: 100, bases: [100, 100], weights: [1, 1]).mains,
          [50, 50], into: &problems)

    // No child may be assigned a negative size, whatever the overflow. The
    // spacing is what makes this reachable: shrinking content alone can only
    // reach zero (each child gives up its own share of a deficit that is at
    // most the content itself), but the gaps are not shrinkable, so a
    // container narrower than the gaps alone drives the formula past zero.
    let crushed = GrMobFlexSolver(spacing: 20, justify: "")
        .resolve(main: 0, bases: [100, 100], weights: [0, 0]).mains
    if crushed.contains(where: { $0 < 0 }) {
        problems.append("shrink produced a negative size: \(crushed)")
    }

    // --- justify-content --------------------------------------------------
    //
    // Distribution is position, not size: the children keep their bases and
    // the leftover becomes a leading offset and/or an inter-item gap. One
    // container (300) holding three 50pt items leaves 150 in every case.
    let bases3: [CGFloat] = [50, 50, 50]
    let zero3: [CGFloat] = [0, 0, 0]

    for (justify, wantLeading, wantGap) in [
        ("", CGFloat(0), CGFloat(0)),          // flex-start
        ("center", 75, 0),
        ("flex-end", 150, 0),
        ("space-between", 0, 75),              // 150 over the 2 inner gaps
        ("space-around", 25, 50),              // half-size edge gaps: 25 / 50 / 50 / 25
        ("space-evenly", 37.5, 37.5),          // 4 equal gaps
    ] {
        let r = GrMobFlexSolver(spacing: 0, justify: justify)
            .resolve(main: 300, bases: bases3, weights: zero3)
        check("justify \(justify.isEmpty ? "flex-start" : justify) leading",
              r.leading, wantLeading, into: &problems)
        check("justify \(justify.isEmpty ? "flex-start" : justify) gap",
              r.gap, wantGap, into: &problems)
        check("justify \(justify.isEmpty ? "flex-start" : justify) sizes",
              r.mains, bases3, into: &problems)
    }

    // space-between with one child has no inner gap to divide; the item stays
    // at the leading edge rather than dividing by zero.
    let single = GrMobFlexSolver(spacing: 0, justify: "space-between")
        .resolve(main: 300, bases: [50], weights: [0])
    check("space-between, one child: leading", single.leading, 0, into: &problems)
    check("space-between, one child: gap", single.gap, 0, into: &problems)

    // A grower consumes the free space, so there is none left to distribute —
    // justify-content becomes a no-op, exactly as in CSS.
    let growAndJustify = GrMobFlexSolver(spacing: 0, justify: "center")
        .resolve(main: 300, bases: [50], weights: [1])
    check("a grower leaves nothing to justify", growAndJustify.leading, 0, into: &problems)
    check("a grower still grows under justify", growAndJustify.mains, [300], into: &problems)

    // --- hug vs. fill -----------------------------------------------------
    //
    // The rule that keeps pre-existing layouts unchanged: claim the offered
    // extent only when something would use it.
    check("hug: no grower, no justify",
          plain.containerMain(offered: 500, bases: [50, 50], weights: [0, 0]),
          100, into: &problems)
    check("fill: a grower claims the space",
          plain.containerMain(offered: 500, bases: [50, 50], weights: [0, 1]),
          500, into: &problems)
    check("fill: justify-content claims the space",
          GrMobFlexSolver(spacing: 0, justify: "space-between")
              .containerMain(offered: 500, bases: [50, 50], weights: [0, 0]),
          500, into: &problems)
    // Overflow is not a choice: a definite offer smaller than the content
    // always wins, so the shrink branch above can run.
    check("a too-small offer always wins",
          plain.containerMain(offered: 60, bases: [50, 50], weights: [0, 0]),
          60, into: &problems)
    // An unspecified or infinite proposal is SwiftUI probing, not an offer.
    check("unspecified proposal hugs",
          plain.containerMain(offered: nil, bases: [50, 50], weights: [0, 1]),
          100, into: &problems)
    check("infinite proposal hugs",
          plain.containerMain(offered: .infinity, bases: [50, 50], weights: [0, 1]),
          100, into: &problems)

    // --- cross-axis placement --------------------------------------------
    check("cross start", GrMobFlexSolver.crossOffset(align: "", child: 20, extent: 100),
          0, into: &problems)
    check("cross center", GrMobFlexSolver.crossOffset(align: "center", child: 20, extent: 100),
          40, into: &problems)
    check("cross end", GrMobFlexSolver.crossOffset(align: "flex-end", child: 20, extent: 100),
          80, into: &problems)
    // A child larger than the container is not pushed off the leading edge.
    check("cross center, oversized child",
          GrMobFlexSolver.crossOffset(align: "center", child: 200, extent: 100),
          0, into: &problems)

    // --- degenerate input -------------------------------------------------
    if !plain.resolve(main: 100, bases: [], weights: []).mains.isEmpty {
        problems.append("empty child list produced sizes")
    }
    // Zero-width bases cannot shrink; the proportional formula would divide
    // by their zero total.
    let zeroBase = plain.resolve(main: -10, bases: [0, 0], weights: [0, 0]).mains
    if zeroBase != [0, 0] {
        problems.append("zero bases under overflow: got \(zeroBase)")
    }

    return problems
}
