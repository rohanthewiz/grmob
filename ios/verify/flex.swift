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

    // A shrink factor of 0 keeps a child at its base and moves the whole
    // deficit onto its neighbours, which is CSS's scaled-base rule with the
    // factors no longer all equal to 1.
    //
    // This arm was unreachable until core.ShrinkNone: `core.FlexShrink(0)`
    // wrote a zero that Style.Merge, htmlout and the WASM runtime all read as
    // "unset", so "do not shrink" was a declaration nobody could write and the
    // solver had no reason to take a factor at all. 300 of content into 200,
    // with the first child pinned: the second absorbs all 100.
    check("a zero shrink factor keeps its base and the rest absorb the deficit",
          plain.resolve(main: 200, bases: [100, 200], weights: [0, 0],
                        shrinks: [0, 1]).mains,
          [100, 100], into: &problems)

    // And the default is still 1: passing the factors explicitly must produce
    // exactly what passing nothing does, or every call site that does not care
    // has quietly changed meaning.
    check("all-ones shrinks are the same as none",
          plain.resolve(main: 200, bases: [100, 200], weights: [0, 0],
                        shrinks: [1, 1]).mains,
          plain.resolve(main: 200, bases: [100, 200], weights: [0, 0]).mains,
          into: &problems)

    // Every child pinned: nothing shrinks and the container overflows, which
    // is the instruction rather than a failure to follow one. The guard that
    // makes this work is the same one that stops a division by zero when every
    // base is zero, so it is worth having a case that reaches it deliberately.
    check("every child pinned means nothing shrinks",
          plain.resolve(main: 100, bases: [100, 200], weights: [0, 0],
                        shrinks: [0, 0]).mains,
          [100, 200], into: &problems)

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

// --------------------------------------------------------------------------
// The fixed-size census, SwiftUI's main-axis row
// --------------------------------------------------------------------------
//
// What a container of a declared size does with a child bigger than it, for
// the one target whose arithmetic this repository owns.
//
// # Why this is here and not derived
//
// docs/platforms/native.md carries a four-target census of that question, and
// its rows were not all established the same way:
//
//	WASM runtime   measured    browser.mjs check 11 mounts the trees and reads
//	                           the rects
//	htmlout        inherited   it emits the same declarations for the same tree
//	SwiftUI        DERIVED     read off .frame(width:)/.frame(height:) and
//	                           GrMobFlexSolver, by a person
//	Compose        derived     read off Modifier.width/height
//
// The SwiftUI row was the odd one out: half of it is a SwiftUI fact (a .frame
// PROPOSES a size and does not enforce one, so the cross axis spills and
// nothing clips it — mobile/verify pins that call site), and the other half is
// not SwiftUI's at all. The main-axis squeeze is GrMobFlexSolver's, which is
// this repository's own CSS flex arithmetic, and this harness executes it. So
// that half can be measured, and reasoning about code we run is the one thing
// in the census that had no excuse.
//
// # The numbers
//
// browser.mjs's, deliberately: the census's claim is that the targets AGREE
// about the main axis, and two harnesses agreeing on different numbers is a
// weaker statement than two harnesses agreeing on the same ones. They are
// pinned to each other by wasm/verify's TestTheFixedSizeCensusUsesOneSetOfNumbers,
// so a fixture edited on one side fails rather than quietly measuring
// something else.
private let voidW: CGFloat = 120
private let voidH: CGFloat = 40
private let oversizeW: CGFloat = 200
private let oversizeH: CGFloat = 80

func checkFixedSizeContainer() -> [String] {
    var problems: [String] = []
    let plain = GrMobFlexSolver(spacing: 0, justify: "")

    // Both containers, because "the main axis is the one that squeezes" is the
    // claim and one container cannot tell it from "width is the one that
    // squeezes". A core.Row stacks along the width and a core.Box (a Column
    // with no theme style) along the height, so the two put the same fixture
    // on opposite axes — which is exactly the pair browser.mjs mounts.
    for (name, voidMain, childMain) in [
        ("a fixed-size core.Row (main axis: width)", voidW, oversizeW),
        ("a fixed-size core.Box (main axis: height)", voidH, oversizeH),
    ] {
        // The declared size reaches the solver as a definite proposal, and a
        // definite offer smaller than the content wins — the container does
        // not grow to fit its child. This is the step that makes the next one
        // an overflow rather than an ordinary layout.
        check("\(name): the container takes its declared extent",
              plain.containerMain(offered: voidMain, bases: [childMain], weights: [0]),
              voidMain, into: &problems)

        // The squeeze itself, and the whole point of the exercise: the child
        // is a flex item with the default factor of 1, the deficit is the
        // whole overflow, and it is the only item — so it gives up all of it
        // and lands exactly on the container's extent. Same answer as the
        // browser's, arrived at by running the code rather than by reading it.
        check("\(name): the child is squeezed to the container's extent",
              plain.resolve(main: voidMain, bases: [childMain], weights: [0]).mains,
              [voidMain], into: &problems)

        // And the declaration that stops it. core.FlexShrink(0) is a factor of
        // zero (core.ShrinkNone), the deficit has nowhere else to go, and the
        // child keeps its base while the container overflows — which is the
        // instruction, not a failure to follow one. The pair is what makes
        // this a statement about the declaration: same container, same child,
        // one factor apart.
        check("\(name): a pinned child keeps its own size and overflows",
              plain.resolve(main: voidMain, bases: [childMain], weights: [0],
                            shrinks: [0]).mains,
              [childMain], into: &problems)
    }

    return problems
}

// Checks for GrMobWrapSolver — the line breaking behind a Row with
// core.FlexWrap(true). Expected values follow CSS flex-line collection with
// flex-shrink: 0, which is also what the Android FlowRow and the browser's
// flex-wrap produce.
func checkWrapSolver() -> [String] {
    var problems: [String] = []
    let solver = GrMobWrapSolver(spacing: 8)

    func check(_ name: String, _ got: [[Int]], _ want: [[Int]]) {
        if got != want { problems.append("\(name): got \(got), want \(want)") }
    }

    // Three 100pt chips with 8pt gaps need 316pt on one line.
    let three: [CGFloat] = [100, 100, 100]
    check("fits on one line", solver.lines(widths: three, available: 316), [[0, 1, 2]])
    check("one pt short breaks the last", solver.lines(widths: three, available: 315), [[0, 1], [2]])
    check("two per line", solver.lines(widths: three, available: 250), [[0, 1], [2]])
    check("one per line", solver.lines(widths: three, available: 100), [[0], [1], [2]])
    // A child wider than the line is neither shrunk nor dropped: it takes a
    // line of its own and the next child starts fresh.
    check("oversized child overflows its own line",
          solver.lines(widths: [300, 50, 50], available: 250), [[0], [1, 2]])
    // No definite width means nothing to wrap against.
    check("no offer -> single line", solver.lines(widths: three, available: nil), [[0, 1, 2]])
    check("infinite offer -> single line", solver.lines(widths: three, available: .infinity), [[0, 1, 2]])
    check("empty", solver.lines(widths: [], available: 100), [])

    if solver.natural(widths: three) != 316 {
        problems.append("natural: got \(solver.natural(widths: three)), want 316")
    }
    return problems
}
