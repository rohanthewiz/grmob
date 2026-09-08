// core.FlexShrink(0) on an overflowing Row, on this target and on Compose, over
// one fixture.
//
// # What this settles
//
// core.ShrinkNone exists so that "do not shrink" is a declaration an author can
// write and all four targets honour. Three of them honour it by having CSS's
// flex-shrink; the fourth has no proportional shrink at all, and Renderer.kt
// honours it with Modifier.pinMainAxis — measure the child unbounded, report
// what was measured. Everything holding THAT up was source-level: mobile/verify
// reads the call sites and the two lines inside the modifier, and
// compileDebugKotlin proves it parses. Nothing produced the arithmetic.
//
// So this is the pair the census was missing. GrMobFlexSolver is this
// repository's CSS flex arithmetic — the same code the SwiftUI Layout runs on,
// split out precisely so it can be executed without a simulator — and it solves
// the Row here. internal/pinfixture carries the Compose answer for the same Row,
// as a transcription of foundation-layout's zero-weight measure loop; what makes
// that worth comparing against rather than believing is written down there,
// along with the one link in its chain that no test checks.
//
// # The three things asserted, and why each needs its own row
//
//	the declaration agrees   the pinned child keeps its own base on BOTH
//	                         targets, in all three positions. That is what
//	                         core.FlexShrink(0) says, and it is the whole point
//	                         of the pin existing on a target with no shrink
//	                         factors.
//
//	the pin is load-bearing  the control row has the same three children with
//	                         nothing pinned, and that child comes back smaller
//	                         on both targets. One declaration apart.
//
//	the siblings diverge     CSS shares the deficit over every child that can
//	                         shrink, so a flex line's sizes do not depend on the
//	                         order; Compose gives each child what the ones
//	                         before it left, so its answer does. Three orders,
//	                         three different Compose answers, one CSS answer.
//
// The row where the two AGREE is asserted as an agreement, for the reason
// band.swift states an unbadged band: an "it diverges" with no case that does
// not is a claim about whatever happened. internal/pinfixture states which is
// which (mainsAgreeWithCSS) and derives it from the case rather than setting it.
//
// # What this is not
//
// A measurement of Compose. Nothing in this repository can run androidx's
// measure policy — see internal/pinfixture for why not — so the Compose column
// is a transcription and is labelled one everywhere it appears. What is measured
// here is this target, and what is compared is one fixture rather than two
// harnesses that happen to agree about different Rows.
import CoreGraphics
import Foundation

/// Tolerance for the solver's CGFloat arithmetic. The fixture's numbers are
/// integers and the solver divides by a total, so an exact comparison would be
/// asking floating point a question it does not answer; anything larger than
/// this would let a whole point of layout through.
private let pinEpsilon: CGFloat = 0.0001

func checkPinnedRow(_ cases: [PinCase]) -> [String] {
    var problems: [String] = []
    if cases.isEmpty {
        return ["the transcript carries no pinned-Row cases — internal/pinfixture "
                + "produced nothing, so this pass has no subject"]
    }

    // The claim each row is a rearrangement of: three children, one Row, and
    // one flag moving. Counted across the whole fixture rather than per row,
    // because it is the SET of rows that says what it says.
    var agreed = 0, diverged = 0
    // The same count for the spacing, which is a separate divergence: the
    // fixture's last case has the same extents on both targets and different
    // gaps.
    var gapsAgreed = 0, gapsDiverged = 0
    // The same child's extent with and without the pin, keyed by name. Filled
    // as the rows are solved and compared after all of them, because the pair
    // it is about spans two cases.
    var pinnedExtent: [String: CGFloat] = [:]
    var unpinnedExtent: [String: CGFloat] = [:]

    for c in cases {
        let bases = c.children.map { $0.base }
        // Every child is unweighted. The renderer never pins a weighted one —
        // Modifier.weight fixes that child's main axis as both minimum and
        // maximum, and CSS never applies grow and shrink at once either — so a
        // weight anywhere here would mean the fixture had grown a case this
        // comparison is not about.
        let weights = [CGFloat](repeating: 0, count: bases.count)
        // core.FlexShrink(0) reaches this solver as a factor of 0 and everything
        // else takes CSS's default of 1. That mapping is the SwiftUI half of the
        // same declaration Modifier.pinMainAxis is the Compose half of.
        let shrinks = c.children.map { $0.pinned ? CGFloat(0) : CGFloat(1) }
        let natural = bases.reduce(0, +) + c.gap * CGFloat(max(bases.count - 1, 0))

        // A Row with room to spare squeezes nobody, so a pinned child in it is
        // indistinguishable from an unpinned one and every comparison below
        // passes by never reaching the arithmetic. gen.go refuses to write such
        // a transcript; this refuses to read one, because a fixture arriving
        // over a wire is not the fixture that was written.
        if natural <= c.offer + pinEpsilon {
            problems.append("\(c.what): the children want \(natural)pt and the Row "
                + "offers \(c.offer)pt, so nothing overflows and this case cannot tell "
                + "a pinned child from an unpinned one")
            continue
        }
        if c.compose.mains.count != bases.count || c.compose.offered.count != bases.count {
            problems.append("\(c.what): the transcript carries \(bases.count) children "
                + "and \(c.compose.mains.count) Compose extents, so the two columns are "
                + "not about the same Row")
            continue
        }

        if c.css.count != bases.count {
            problems.append("\(c.what): the fixture states \(c.css.count) CSS extents "
                + "for \(bases.count) children, so the column the census prints is about "
                + "a different Row")
            continue
        }

        let rowHasAPin = c.children.contains { $0.pinned }
        let solver = GrMobFlexSolver(spacing: c.gap, justify: "")
        let css = solver.resolve(main: c.offer, bases: bases, weights: weights,
                                 shrinks: shrinks).mains

        // --- the census's CSS column ----------------------------------------
        //
        // The whole column, not just the pinned child. internal/pinfixture
        // STATES these numbers and computes none of them — a flex line
        // transcribed into Go would be a third spelling after this solver and a
        // browser — so the claim only means anything while something is held to
        // it, and this is one of the two things that are.
        //
        // The control row is why it matters. Three of the four rows are pinned
        // down by other assertions below (the pin keeps its base, the extents
        // match Compose where the fixture says they do, a child's width does
        // not depend on where it sits). The no-pin row's 24/80/16 is determined
        // by nothing else at all: it is the scaled-base rule producing three
        // particular numbers, and until the fixture carried them a reader who
        // trusted that row was trusting a comment.
        for i in 0..<bases.count where abs(css[i] - c.css[i]) > pinEpsilon {
            let got = (0..<bases.count).map { "\(c.children[$0].name) \(css[$0])" }
                .joined(separator: ", ")
            problems.append("\(c.what): GrMobFlexSolver lays the children out at "
                + "[\(got)] and internal/pinfixture states the CSS column as "
                + "\(c.css).\n"
                + "      That column is the one the census prints, and nothing computes "
                + "it: the fixture states it and this solver and a browser are what hold "
                + "it. The control row is the sharpest case — its 24/80/16 is the "
                + "scaled-base rule's answer and no other assertion here determines it")
            break
        }

        // --- the spacing ----------------------------------------------------
        //
        // A CSS flex line's gap is used space: it is charged between every
        // adjacent pair whatever the children did, and this solver takes it out
        // of the free space in `natural` before any shrinking happens. A
        // Compose Row clamps each gap to what was left, so an overflowing Row
        // inserts none after the child that spent the axis.
        //
        // That sentence is repeated in three documents and, until the fixture
        // grew a case with a gap in it, was measured by nothing. Both arms are
        // asserted, because a fixture where every row collapsed would asserted
        // the collapse and never reach the agreement.
        var gapsSame = true
        for i in 0..<max(bases.count - 1, 0)
        where abs(c.compose.gaps[i] - c.gap) > pinEpsilon {
            gapsSame = false
        }
        if gapsSame != c.gapsAgreeWithCSS {
            problems.append("\(c.what): the Row's own spacing is \(c.gap)pt and the "
                + "Compose column inserts \(c.compose.gaps), which \(gapsSame ? "is" : "is not") "
                + "what a flex line inserts — and the fixture says gapsAgreeWithCSS="
                + "\(c.gapsAgreeWithCSS).\n"
                + "      This solver charges `spacing` between every adjacent pair before "
                + "it distributes anything, so CSS's answer is the gap repeated. "
                + "spaceAfterLastNoWeight is min(spacing, what is left), so Compose's is "
                + "not, once the Row has overflowed. A fixture that had stopped carrying "
                + "both kinds of row would assert one of them twice")
        }
        if gapsSame { gapsAgreed += 1 } else { gapsDiverged += 1 }

        // --- the declaration ------------------------------------------------
        //
        // The pinned child keeps its base, on both targets. This is the sentence
        // core.ShrinkNone's doc comment makes about all four targets and the one
        // the fourth had no way to express before Modifier.pinMainAxis.
        for (i, child) in c.children.enumerated() where child.pinned {
            if abs(css[i] - child.base) > pinEpsilon {
                problems.append("\(c.what): the pinned child is \(css[i])pt and its base "
                    + "is \(child.base)pt. core.FlexShrink(0) is a factor of zero, so this "
                    + "solver's scaled-base rule gives it none of the deficit — a pinned "
                    + "child that shrank here means the factor is not reaching the solver, "
                    + "which is the state this target was in before core.ShrinkNone")
            }
            if abs(c.compose.mains[i] - child.base) > pinEpsilon {
                problems.append("\(c.what): the Compose column has the pinned child at "
                    + "\(c.compose.mains[i])pt and its base is \(child.base)pt. "
                    + "Modifier.pinMainAxis measures with Constraints.Infinity and reports "
                    + "what it measured; a smaller number means the transcription in "
                    + "internal/pinfixture no longer mirrors it")
            }
            // The pin is only saying anything where the Row offered less than
            // the child wanted. Where it offered more, the child would have kept
            // its size unpinned and the row says nothing about the declaration —
            // which is true of exactly one arrangement, the one where the pin
            // comes first and is handed the whole container.
            let pinIsFirst = c.children.first?.pinned == true
            if !pinIsFirst && c.compose.offered[i] >= child.base {
                problems.append("\(c.what): the Row offered the pinned child "
                    + "\(c.compose.offered[i])pt for a base of \(child.base)pt, so the pin "
                    + "refused nothing. Only the arrangement where the pin comes FIRST may "
                    + "be in that position; anywhere else it means the children before it "
                    + "stopped taking any room and this row has become a duplicate of that "
                    + "one")
            }
        }

        // --- the pin against no pin -----------------------------------------
        //
        // The control row holds the same three children with nothing pinned, so
        // "the pin changed something" is a statement about the PAIR rather than
        // about either row. Recorded here by child name and compared once the
        // whole fixture has been solved, because the two rows are two passes
        // through this loop.
        for (i, child) in c.children.enumerated() {
            if child.pinned {
                pinnedExtent[child.name] = css[i]
            } else if !rowHasAPin {
                unpinnedExtent[child.name] = css[i]
            }
        }

        // --- where the two targets part company -----------------------------
        var same = true
        for i in 0..<bases.count where abs(css[i] - c.compose.mains[i]) > pinEpsilon {
            same = false
        }
        if same != c.mainsAgreeWithCSS {
            let pairs = (0..<bases.count).map {
                "\(c.children[$0].name) \(css[$0])/\(c.compose.mains[$0])"
            }.joined(separator: ", ")
            if c.mainsAgreeWithCSS {
                problems.append("\(c.what): the fixture says the two targets land on the "
                    + "same extents here and they do not (CSS/Compose: \(pairs)).\n"
                    + "      One row in the fixture is supposed to agree, and it is the "
                    + "bound on everything the others claim: an 'it diverges' with no case "
                    + "that does not is a claim about whatever happened. See "
                    + "internal/pinfixture for which row that is and why the agreement is "
                    + "two different rules arriving at one answer rather than a shared one")
            } else {
                problems.append("\(c.what): the two targets land on the same extents "
                    + "(\(pairs)), and this row records a divergence.\n"
                    + "      CSS shares a deficit over every child that can shrink, so a "
                    + "flex line's sizes do not depend on the order; a Compose Row gives "
                    + "each child what the ones before it left, so its answer does. If "
                    + "that has stopped being true, the good news is here and the note in "
                    + "internal/pinfixture is now wrong")
            }
        }
        if same { agreed += 1 } else { diverged += 1 }

        // --- the container itself -------------------------------------------
        //
        // The other half of what a pin does, and the half no census row states.
        // A definite offer smaller than the content wins here: the container
        // takes its declared extent and the child spills out of it, which is
        // `overflow: visible`. foundation-layout's mainAxisLayoutSize is
        // max(content, mainAxisMin) with NO upper bound, and Size.kt's own node
        // reports the measured size unclamped — so the Compose Row is measured
        // wider than it was told to be.
        let container = solver.containerMain(offered: c.offer, bases: bases, weights: weights)
        if abs(container - c.offer) > pinEpsilon {
            problems.append("\(c.what): the container is \(container)pt for a declared "
                + "\(c.offer)pt. A definite offer smaller than the content is supposed to "
                + "win — that is what makes the next line an overflow rather than an "
                + "ordinary layout")
        }
        if c.compose.rowMain < c.offer - pinEpsilon {
            problems.append("\(c.what): the Compose Row reports \(c.compose.rowMain)pt for "
                + "a declared \(c.offer)pt, which is smaller than its own minimum")
        }
    }

    // And the pair itself: one declaration apart, and the child's own size is
    // what the declaration is about.
    for (name, pinned) in pinnedExtent.sorted(by: { $0.key < $1.key }) {
        guard let unpinned = unpinnedExtent[name] else {
            problems.append("the fixture pins \(name) and never solves a row with it "
                + "unpinned, so nothing here says the declaration changed anything. The "
                + "control row is what makes these a statement about core.FlexShrink(0) "
                + "rather than about a Row with a big child in it")
            continue
        }
        if pinned <= unpinned + pinEpsilon {
            problems.append("\(name) is \(unpinned)pt unpinned and \(pinned)pt pinned. "
                + "core.FlexShrink(0) is the difference between taking a share of the "
                + "deficit and taking none of it; equal numbers mean the factor is not "
                + "reaching the solver")
        }
    }

    // Both kinds of row must be present, or the fixture has quietly become a
    // one-sided claim. This is the count band.swift keeps for the same reason.
    if agreed == 0 {
        problems.append("no case in the fixture has the two targets agreeing. The "
            + "divergence recorded here is meant to be bounded by a row that does not "
            + "diverge; without one it is a claim about whatever the fixture happens to "
            + "hold")
    }
    if diverged == 0 {
        problems.append("no case in the fixture has the two targets disagreeing, so "
            + "nothing here is measuring the no-proportional-shrink divergence the "
            + "census records")
    }
    if gapsAgreed == 0 {
        problems.append("no case in the fixture inserts the spacing a flex line "
            + "inserts. Four of the five rows carry no gap at all, which agrees "
            + "vacuously and is what bounds the one that does not")
    }
    if gapsDiverged == 0 {
        problems.append("no case in the fixture has the spacing collapsing, so the one "
            + "line of the measure policy nothing used to measure — "
            + "spaceAfterLastNoWeight is min(spacing, what is left) — is unmeasured "
            + "again. internal/pinfixture is supposed to carry a row with a gap in it")
    }
    return problems
}
