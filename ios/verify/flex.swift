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

    // --- Grow with minimums (zero flex-basis) -----------------------------
    //
    // A zero-basis child's base is its padding and its min is its content,
    // so a min can exceed a base here and nowhere else. Seven day cells of a
    // calendar week, bases 0, weight 1, numerals 8 to 18 wide, in 350: the
    // share (50 each) clears every floor, so the floors change nothing and
    // the columns are equal. Folding the floors into the bases gave
    // [50-ish ± 5] instead, which is the bug lesson 4.9 showed.
    check("zero bases share the line by weight whatever their minimums",
          plain.resolve(main: 350, bases: [0, 0, 0, 0, 0, 0, 0], weights: [1, 1, 1, 1, 1, 1, 1],
                        mins: [8, 8, 18, 18, 18, 18, 18]).mains,
          [50, 50, 50, 50, 50, 50, 50], into: &problems)

    // A share below a floor: 100 by weight 1:1 gives 50 each, the first
    // child's content needs 70, so it is frozen there and the other takes
    // the remaining 30. CSS 9.7's min violation, the grow half.
    check("a grower short of its minimum is raised and the rest share again",
          plain.resolve(main: 100, bases: [0, 0], weights: [1, 1],
                        mins: [70, 0]).mains,
          [70, 30], into: &problems)

    // A padded zero-basis child (base 10) shares on top of its padding:
    // 110 free gives [65, 55], clear of a floor of 60, so nothing moves.
    check("a padded zero-basis child grows from its padding",
          plain.resolve(main: 120, bases: [10, 0], weights: [1, 1],
                        mins: [60, 0]).mains,
          [65, 55], into: &problems)

    // With a floor of 80 the same share falls short, and the raise lands on
    // the floor exactly, not on base plus share: frozen at 80, the other
    // absorbs everything left, 120 - 80 = 40.
    check("a raised child keeps its floor, not its base plus a share",
          plain.resolve(main: 120, bases: [10, 0], weights: [1, 1],
                        mins: [80, 0]).mains,
          [80, 40], into: &problems)

    // Minima that fill the line decide grow-or-shrink: 90 + 60 > 120 is a
    // shrink, from the hypothetical sizes, whose floors are those same
    // sizes, so nothing gives and the line overflows as CSS does.
    check("zero-basis minima that overflow overflow",
          plain.resolve(main: 120, bases: [0, 0], weights: [1, 1],
                        mins: [90, 60]).mains,
          [90, 60], into: &problems)

    // A zero-basis child that does not grow is its content: hypothetical
    // size, and the leftover becomes position.
    check("a zero-basis child with no weight is its minimum",
          plain.resolve(main: 200, bases: [0, 40], weights: [0, 0],
                        mins: [30, 40]).mains,
          [30, 40], into: &problems)

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

    // --- the min-content floor --------------------------------------------
    //
    // CSS floors every flex item at its automatic minimum size, so an
    // overflowing line overflows rather than grinding its children away. The
    // bug this closes: the tutorial's lesson number, a bare Text beside a
    // grow column, was compressed to one glyph and wrapped down the side of
    // the row as 4 / . / 1 / 2.
    //
    // 300 of content into 200 with the first child floored at 90: the plain
    // rule would give it 66.67, so it clamps, keeps 90, and the deficit it
    // did not absorb lands on the only other child — 200 - 90 = 110.
    check("a floored child keeps its minimum and the rest absorb the difference",
          plain.resolve(main: 200, bases: [100, 200], weights: [0, 0],
                        mins: [90, 0]).mains,
          [90, 110], into: &problems)

    // A floor the plain rule already clears changes nothing, which is what
    // makes the floor safe to apply everywhere: 66.67 is above 50.
    check("a floor below the shrunk size is not reached",
          plain.resolve(main: 200, bases: [100, 200], weights: [0, 0],
                        mins: [50, 0]).mains,
          plain.resolve(main: 200, bases: [100, 200], weights: [0, 0]).mains,
          into: &problems)

    // The pass above is what makes this a loop rather than one division, and
    // this is the case that shows the loop converging on the exact answer
    // rather than merely on a bigger one. Two equal children, 30 of deficit:
    // the plain rule gives them 85 each, the first clamps at its floor of 90,
    // and the 5 it refused is owed by the second — which can pay it, landing
    // at 80. The line then adds up to exactly the 170 it was given. A
    // single-pass implementation gives [90, 85] and overflows by 5 that
    // nothing accounts for.
    check("the deficit a floored child refuses lands on the rest",
          plain.resolve(main: 170, bases: [100, 100], weights: [0, 0],
                        mins: [90, 0]).mains,
          [90, 80], into: &problems)

    // Floors that cannot all be met overflow, exactly as CSS does — the line
    // is 200 wide and the two minima are 240 together. Nothing is crushed to
    // make the arithmetic come out.
    check("floors that do not fit overflow",
          plain.resolve(main: 200, bases: [100, 200], weights: [0, 0],
                        mins: [100, 140]).mains,
          [100, 140], into: &problems)

    // A pin outranks a floor: FlexShrink(0) is "no smaller than the ideal
    // size", which is never below "no smaller than the content".
    check("a pinned child is floored at its base, not at its min",
          plain.resolve(main: 200, bases: [100, 200], weights: [0, 0],
                        shrinks: [0, 1], mins: [10, 0]).mains,
          [100, 100], into: &problems)

    // Passing no floors has to be exactly what the solver did before they
    // existed, or every caller that cannot measure a view has quietly
    // changed — internal/pinfixture's CSS column is the one that matters, and
    // it is a column a real browser has been watched producing.
    check("all-zero mins are the same as none",
          plain.resolve(main: 200, bases: [100, 200], weights: [0, 0],
                        mins: [0, 0]).mains,
          plain.resolve(main: 200, bases: [100, 200], weights: [0, 0]).mains,
          into: &problems)

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

// Checks for GrMobMaxWidth — core.MaxWidth's arithmetic on iOS.
//
// The Layout that applies the cap needs a view hierarchy; which strings are a
// cap, what a percentage resolves against, and how a cap meets a rigid Width
// do not. Expected values are CSS's: `max-width` in px, in % of the containing
// block (none when that block is indefinite), `none` as the initial value, and
// min(width, max-width) for a declared width.
func checkMaxWidth() -> [String] {
    var problems: [String] = []
    func limit(_ name: String, _ value: String, _ available: CGFloat?, _ want: CGFloat?) {
        let got = GrMobMaxWidth.limit(value, available: available)
        if got != want { problems.append("maxWidth \(name): got \(String(describing: got)), want \(String(describing: want))") }
    }
    limit("px", "320px", 400, 320)
    limit("bare number is points", "320", nil, 320)
    limit("px binds with no parent width", "320px", nil, 320)
    limit("percent of the offer", "50%", 400, 200)
    limit("percent with no offer is none", "50%", nil, nil)
    limit("percent of an infinite probe is none", "50%", .infinity, nil)
    limit("percent above 100 is kept", "150%", 200, 300)
    limit("empty", "", 400, nil)
    limit("none", "none", 400, nil)
    limit("auto", "auto", 400, nil)
    limit("negative is invalid, not zero", "-5px", 400, nil)
    limit("unknown unit", "20vw", 400, nil)
    limit("junk percent", "abc%", 400, nil)
    limit("zero is a cap", "0px", 400, 0)

    if GrMobMaxWidth.fixedLimit("50%") != nil {
        problems.append("maxWidth fixedLimit: a percentage has no length before the Layout resolves it")
    }
    if GrMobMaxWidth.fixedLimit("320px") != 320 {
        problems.append("maxWidth fixedLimit: points should resolve without a parent")
    }
    check("maxWidth clamps a wider width", GrMobMaxWidth.clamp(600, to: 520), 520, into: &problems)
    check("maxWidth leaves a narrower width", GrMobMaxWidth.clamp(400, to: 520), 400, into: &problems)
    check("maxWidth with no cap", GrMobMaxWidth.clamp(600, to: nil), 600, into: &problems)
    return problems
}

// Checks for GrMobMinSize — core.MinWidth and core.MinHeight's arithmetic on
// iOS, which GrMobMinimumLayout resolves per proposal. Expected values are
// CSS's for a floor: px, % of the containing block (no floor when that block
// is indefinite), and no floor for zero, a negative or anything unreadable.
func checkMinSize() -> [String] {
    var problems: [String] = []
    func floor(_ name: String, _ value: String, _ available: CGFloat?, _ want: CGFloat?) {
        let got = GrMobMinSize.floor(value, available: available)
        if got != want { problems.append("minSize \(name): got \(String(describing: got)), want \(String(describing: want))") }
    }
    floor("px", "280px", 400, 280)
    floor("bare number is points", "280", nil, 280)
    floor("percent of the offer", "50%", 400, 200)
    floor("percent with no offer is none", "50%", nil, nil)
    floor("percent of an infinite probe is none", "50%", .infinity, nil)
    floor("zero is no floor", "0px", 400, nil)
    floor("zero percent is no floor", "0%", 400, nil)
    floor("negative is invalid", "-5px", 400, nil)
    floor("auto", "auto", 400, nil)
    floor("empty", "", 400, nil)
    floor("junk percent", "abc%", 400, nil)

    // The fraction GrMobFlexLayout resolves on a child's main axis, and what
    // it resolves to. See GrMobFlexSolver.percentFloors.
    func fraction(_ name: String, _ value: String, _ want: CGFloat?) {
        let got = GrMobMinSize.fraction(value)
        if got != want { problems.append("minSize fraction \(name): got \(String(describing: got)), want \(String(describing: want))") }
    }
    fraction("percent", "40%", 0.4)
    fraction("points are not a fraction", "280px", nil)
    fraction("zero percent is none", "0%", nil)
    fraction("junk", "abc%", nil)

    let noOffer = GrMobFlexSolver.percentFloors(fractions: [0.4, 0], extent: nil)
    if noOffer != [0, 0] { problems.append("percentFloors with no extent: got \(noOffer), want [0, 0]") }
    let probe = GrMobFlexSolver.percentFloors(fractions: [0.4], extent: .infinity)
    if probe != [0] { problems.append("percentFloors against an infinite probe: got \(probe), want [0]") }

    // Lesson 1.4's row on a 402pt phone: 294pt of content, A's label box
    // 31pt, B 49pt, C 67pt, 8pt gaps, A floored at 40%. The floor raises A's
    // base to 117.6 and nothing else moves: the run (117.6 + 49 + 67 + 16 =
    // 249.6) still fits, so there is nothing to shrink and justify-start packs
    // it at the leading edge.
    let row = GrMobFlexSolver(spacing: 8, justify: "")
    let floors = GrMobFlexSolver.percentFloors(fractions: [0.4, 0, 0], extent: 294)
    let bases = zip([31, 49, 67] as [CGFloat], floors).map { max($0, $1) }
    let laid = row.resolve(main: 294, bases: bases, weights: [0, 0, 0],
                           mins: zip([31, 49, 67] as [CGFloat], floors).map { max($0, $1) })
    if abs(laid.mains[0] - 117.6) > 0.001 || laid.mains[1] != 49 || laid.mains[2] != 67 {
        problems.append("percent floor in a row: got \(laid.mains), want [117.6, 49, 67]")
    }
    // Squeezed to 200pt, B and C shrink and A holds its floor, because the
    // floor is also its minimum.
    let tight = GrMobFlexSolver.percentFloors(fractions: [0.4, 0, 0], extent: 200)
    let tightBases = zip([31, 49, 67] as [CGFloat], tight).map { max($0, $1) }
    let squeezed = row.resolve(main: 200, bases: tightBases, weights: [0, 0, 0],
                               mins: [tight[0], 0, 0])
    if abs(squeezed.mains[0] - 80) > 0.001 || squeezed.mains[1] >= 49 || squeezed.mains[2] >= 67 {
        problems.append("percent floor under shrink: got \(squeezed.mains), want A held at 80 and B, C shrunk")
    }
    return problems
}
