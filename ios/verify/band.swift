// Checks for components.GroupHeader's insets, solved through GrMobFlexSolver —
// the CSS flex arithmetic behind the iOS renderer's Row layout.
//
// # The claim
//
// The band's padding used to be on the Row and is now on the growing control
// inside it (components.bandInsets). The move's whole justification is that a
// press should land on the band rather than on a strip in the middle of it, and
// its whole warrant is that it costs nothing: padding on a stretched child
// fills exactly the space the same padding on its parent held.
//
// That was verified on the web by the pixels it did not move. On this renderer
// it was an assumption. SwiftUI resolves the two arrangements through different
// code — a `.padding()` around the container versus one around a child, with
// GrMobFlexSolver distributing between them — and nothing had ever asked
// whether they come out the same.
//
// So both arrangements are solved here, at four container widths, and compared.
// The numbers are internal/bandfixture's, read off a real rendered band rather
// than transcribed; the content sizes are synthetic, because what is under test
// is the arithmetic of an arrangement and not the width of a string.
//
// # Where the padding sits in the solver's terms
//
// The renderer applies a node's padding inside grMobBox, around the node's own
// content, so a child's measured size *includes* its padding and a container's
// proposal has already had its own padding taken off. In flex terms:
//
//	container inner   = offer - row.left - row.right
//	base of a child   = its content + its own padding
//	used outer size   = base + its share of the free space
//
// which is CSS's own accounting: the free space is the inner extent less the
// sum of the children's outer hypothetical sizes, and growth is added to the
// outer size. So the two arrangements differ only in *which* of the two numbers
// the same 16 points is part of, and that is exactly what is checked below.
import CoreGraphics

/// One arrangement's resolved geometry, in the band Row's own coordinates.
private struct BandLayout: Equatable {
    /// Whether the offer was smaller than the arrangement's natural width, so
    /// the shrink arm ran. See the overflow note in checkBandInsets.
    var overflowed: Bool = false
    /// The band's total width, and its height.
    var width: CGFloat
    var height: CGFloat
    /// The label's content box: where the words start, and how wide they get.
    var labelX: CGFloat
    var labelWidth: CGFloat
    /// The badge's leading edge, or nil when the band has no badge.
    var badgeX: CGFloat?
    /// The control's own box — the thing a finger lands on.
    var controlX: CGFloat
    var controlWidth: CGFloat
}

/// Solves one arrangement at one offer.
///
/// `offer` is nil for an indefinite proposal, which is SwiftUI asking for an
/// ideal size rather than making an offer; the solver hugs there, and the two
/// arrangements have to agree about what they hug to as well.
private func solveBand(_ a: BandArrangement, label: BandSize, badge: BandSize,
                       offer: CGFloat?) -> BandLayout {
    let hasBadge = badge.w > 0
    // A child's base is its content plus its own padding, which is what the
    // renderer measures because grMobBox puts the padding inside the child.
    let controlBase = label.w + a.control.left + a.control.right
    var bases: [CGFloat] = [controlBase]
    var weights: [CGFloat] = [a.grow]
    if hasBadge {
        bases.append(badge.w)
        weights.append(0)
    }

    let solver = GrMobFlexSolver(spacing: a.gap, justify: "")
    let chrome = a.row.left + a.row.right
    // The proposal reaching the Layout has already had the container's own
    // padding removed, so an offer becomes an inner extent and an indefinite
    // proposal stays indefinite.
    let inner = offer.map { $0 - chrome }
    let usedInner = solver.containerMain(offered: inner, bases: bases, weights: weights)
    let resolved = solver.resolve(main: usedInner, bases: bases, weights: weights)

    let controlX = a.row.left + resolved.leading
    var layout = BandLayout(
        overflowed: usedInner < solver.natural(bases: bases) - 0.0001,
        width: usedInner + chrome,
        height: 0,
        labelX: controlX + a.control.left,
        labelWidth: resolved.mains[0] - a.control.left - a.control.right,
        badgeX: nil,
        controlX: controlX,
        controlWidth: resolved.mains[0])
    if hasBadge {
        // The inter-item gap is the arrangement's own spacing *plus* whatever
        // justify-content distributed — resolve() returns the second alone, on
        // the reasoning flex.swift's "gap comes out of the free space" case
        // records: spacing is consumed before growth, so it is already out of
        // the free space and is not part of what gets handed back.
        layout.badgeX = controlX + resolved.mains[0] + a.gap + resolved.gap
    }

    // Cross axis. The band centres its children (AlignItemsCenter), so the
    // Row's content height is its tallest child and its own height adds its own
    // vertical padding — which is the half that moves.
    var tallest = label.h + a.control.top + a.control.bottom
    if hasBadge { tallest = max(tallest, badge.h) }
    layout.height = tallest + a.row.top + a.row.bottom
    return layout
}

// The one place the two arrangements do not agree, and it is not the cross axis.
//
// Under overflow — an offer narrower than the band's own content — this solver
// shrinks each child in proportion to its *base* size, and a child's base
// includes its own padding. So the same 16 points is inside the shrink
// proportion in one arrangement and outside it in the other, and the label ends
// up with slightly different room:
//
//	120pt offered, a 100pt label and a 24pt badge
//	  insets on the control   base 124 of 148 -> 87.14 -> label 63.14
//	  insets on the Row       base 100 of 124 -> 64.52 -> label 64.52
//
// Neither number is wrong as arithmetic; what is recorded here is that they are
// not the same number, which is precisely what the move was believed to cost
// nothing.
//
// It needs a second child to show. With the count hidden the control is the
// only thing on the line, and a lone child is clamped to the container in
// either arrangement — so the unbadged case agrees under overflow and the
// badged one does not, which is why the fixture states which is which
// (SharesADeficit) and both are asserted. An "it diverges" with no case that
// does not would be a claim about whatever happened.
//
// It is a fact about *this* solver, and that is now measured rather than
// suspected. CSS distributes shrink over the inner flex base size rather than
// the outer one, and wasm/verify/browser.mjs mounts the same fixture in a real
// Chrome and reads the rects back: there the two arrangements agree at every
// offer, overflow included, down to the LayoutUnit. So the difference below is
// a genuine cross-target divergence — one renderer's shrink proportion counts a
// child's padding and the other's does not — rather than an artefact of either
// implementation.
//
// Asserted rather than skipped so that a solver which started agreeing is a
// failure somebody reads and not a silent improvement. The browser's half is
// asserted in the other direction for the same reason.
//
// It is also the narrowest possible case: a band this narrow is one whose label
// is already being truncated.
func checkBandInsets(_ cases: [BandCase]) -> [String] {
    var problems: [String] = []
    if cases.isEmpty {
        return ["the transcript carries no band cases — internal/bandfixture " +
                "produced nothing, so this pass has no subject"]
    }

    for c in cases {
        // The height comparison below models a Row that centres its children:
        // its height is its tallest child plus its own vertical padding, which
        // is why moving that padding onto one child stops it being added to the
        // other. Under `stretch` none of that holds, so a case that is not
        // centred is refused rather than answered for.
        for a in [c.now, c.before] where a.align != "center" {
            problems.append("\(c.what): the band aligns its children \(a.align) with " +
                "the insets \(a.what), and the height comparison here is only valid " +
                "for centred children. internal/bandfixture reads this off the widget, " +
                "so the band itself has changed")
        }
        if c.now.align != "center" || c.before.align != "center" { continue }

        var sawSlack = false, sawOverflow = false
        for offer in c.offers {
            let proposal: CGFloat? = offer < 0 ? nil : CGFloat(offer)
            let where_ = "\(c.what) at " +
                (proposal.map { "\($0)pt" } ?? "an indefinite proposal")

            let now = solveBand(c.now, label: c.label, badge: c.badge, offer: proposal)
            let before = solveBand(c.before, label: c.label, badge: c.badge, offer: proposal)

            // The main axis, which is the claim. Every one of these is a
            // separate way the move could have cost something: a band that got
            // wider or narrower, words that start somewhere else, a label with
            // a different amount of room, or a badge that is no longer hard
            // against the trailing edge.
            //
            // The control's own leading edge is deliberately *not* here. It is
            // 0 in one arrangement and the leading inset in the other, and that
            // is the whole point of the move — the target reaches the band's
            // edge instead of starting 16pt into it. It is asserted below, in
            // the direction that says it moved.
            var measures = [
                ("the band's width", now.width, before.width),
                ("the label's leading edge", now.labelX, before.labelX),
                ("the label's width", now.labelWidth, before.labelWidth),
            ]
            if let a = now.badgeX, let b = before.badgeX {
                measures.append(("the badge's leading edge", a, b))
            }

            if now.overflowed != before.overflowed {
                problems.append("\(where_): one arrangement overflows and the other " +
                    "does not, which cannot happen — the two hold the same chrome, so " +
                    "they have the same natural width. internal/bandfixture's " +
                    "Rewind has stopped conserving it")
                continue
            }
            if now.overflowed {
                sawOverflow = true
                // The recorded divergence, and only where there is a second
                // child to divide the deficit with — a lone growing control is
                // clamped to the container in either arrangement and lands on
                // the same number. The rest of the band still has to agree
                // whichever it is, because the deficit itself is the same and
                // only its distribution differs.
                let diverged = abs(now.labelWidth - before.labelWidth) > 0.0001
                if diverged != c.sharesADeficit {
                    if c.sharesADeficit {
                        problems.append("\(where_): the label gets " +
                            "\(now.labelWidth)pt in both arrangements, and with a " +
                            "second child sharing the deficit it should not — this " +
                            "solver shrinks in proportion to a base that includes the " +
                            "control's own padding, so the two arrangements divide the " +
                            "deficit differently. If that has been fixed, this is the " +
                            "good news and the note above it is now wrong")
                    } else {
                        problems.append("\(where_): the label gets " +
                            "\(now.labelWidth)pt with the insets \(c.now.what) and " +
                            "\(before.labelWidth)pt with them \(c.before.what), and " +
                            "there is nothing here to share the deficit with — one " +
                            "growing child is clamped to the container in either " +
                            "arrangement, so this is a difference the shrink " +
                            "proportion cannot account for")
                    }
                }
                if abs(now.width - before.width) > 0.0001 {
                    problems.append("\(where_): the band is \(now.width)pt wide with " +
                        "the insets \(c.now.what) and \(before.width)pt with them " +
                        "\(c.before.what). Overflow divides the deficit differently " +
                        "in the two arrangements, but the deficit itself is the same, " +
                        "so the band's own width must not move")
                }
                continue
            }

            sawSlack = true
            for m in measures {
                if abs(m.1 - m.2) > 0.0001 {
                    problems.append("\(where_): \(m.0) is \(m.1) with the insets " +
                        "\(c.now.what) and \(m.2) with them \(c.before.what). Moving " +
                        "the band's padding onto its control is supposed to be the " +
                        "same pixels one node in, and on this renderer it is not")
                }
            }

            // The control's own box is the one thing that is meant to differ,
            // and it is the reason for the move: after it the target is the
            // band, before it the target was the label alone. Asserted rather
            // than left implied, because an arrangement that agreed here would
            // mean the insets had not actually moved and every comparison above
            // would be between two copies of one band.
            let grew = now.controlWidth - before.controlWidth
            let wantGrew = c.now.control.left + c.now.control.right
            if abs(grew - wantGrew) > 0.0001 {
                problems.append("\(where_): the control is \(grew)pt wider with the " +
                    "insets on it, and the insets it carries total \(wantGrew)pt. The " +
                    "target is supposed to have absorbed exactly the chrome that left " +
                    "the Row")
            }
            if abs(now.controlX - c.now.row.left) > 0.0001 {
                problems.append("\(where_): the control starts at \(now.controlX)pt " +
                    "with the insets on it, and the Row's own leading inset is " +
                    "\(c.now.row.left)pt. The whole point of the move is that a press " +
                    "lands on the band's leading edge rather than 16pt into it")
            }

            // The cross axis, which is where the two arrangements can genuinely
            // disagree even with slack: with align-items centre, a Row's height
            // is its tallest child plus its own vertical padding, so moving that
            // padding onto one child stops it being added to the other. They
            // agree exactly while the padded control is the taller child, which
            // every real band is (internal/bandfixture pins the inset half of
            // that) — and the fixture carries one case where it is not, so the
            // divergence is recorded rather than assumed away.
            let sameHeight = abs(now.height - before.height) <= 0.0001
            if sameHeight != c.sameHeight {
                if c.sameHeight {
                    problems.append("\(where_): the band is \(now.height)pt tall with " +
                        "the insets \(c.now.what) and \(before.height)pt with them " +
                        "\(c.before.what). A band whose control is the tallest child " +
                        "keeps its height across the move; this one did not, so " +
                        "either the alignment is no longer centre or a child is " +
                        "taller than the fixture thinks")
                } else {
                    problems.append("\(where_): the band is \(now.height)pt tall in " +
                        "both arrangements, and this is the case where it should not " +
                        "be — a badge taller than the padded control loses the Row's " +
                        "vertical padding when that padding moves. If the heights now " +
                        "agree, the solver is no longer centring, and a real band's " +
                        "agreement is holding for a different reason than the one " +
                        "recorded")
                }
            }
        }
        // Both arms have to have run. A fixture whose offers all had slack
        // would assert the identity and never reach the divergence, and one
        // with none would assert the divergence and never check the claim the
        // whole file is about.
        if !sawSlack || !sawOverflow {
            problems.append("\(c.what): the offers reached " +
                (sawSlack ? "" : "no case with slack ") +
                (sawOverflow ? "" : "no case that overflows ") +
                "— internal/bandfixture is supposed to carry both, and half this " +
                "check is asserted over nothing")
        }
    }
    return problems
}
