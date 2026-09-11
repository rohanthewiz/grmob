import CoreGraphics

/// The CSS flex algorithm for one axis, as pure arithmetic.
///
/// Split out of GrMobFlexLayout (Renderer.swift) deliberately: a SwiftUI
/// `Layout` can only be exercised by mounting it in a view hierarchy, which
/// needs a simulator, while everything that is actually easy to get wrong
/// here — proportional growth, proportional shrink, the five justify-content
/// distributions, hug-vs-fill — is a function from numbers to numbers. This
/// file imports CoreGraphics and nothing else, so `ios/verify` can check it
/// on a plain macOS host.
///
/// The model is CSS Flexbox with `flex-basis: auto` (a child's base size is
/// its content size):
///
/// ```
/// base_i    = subview ideal size along the axis
/// natural   = sum(base_i) + spacing * (n - 1)
/// free      = container - natural
/// free > 0  -> size_i = base_i + free * weight_i / sum(weight)          [grow]
/// free < 0  -> size_i = base_i + free * s_i*base_i / sum(s_j*base_j)  [shrink]
/// leftover  -> distributed by justify-content as offsets, not sizes
/// ```
///
/// `s_i` is the child's flex-shrink factor, and it used to be 1 for every child
/// because 1 was the only value the Go side could express: `core.FlexShrink(0)`
/// wrote a zero that every renderer read as "unset". That is fixed (see
/// core.ShrinkNone), so the shrink arm is now the scaled-base rule CSS actually
/// states, and a child with a factor of 0 keeps its base size while the others
/// absorb the whole deficit.
///
/// # The part of the shrink arm that is missing, found on a simulator
///
/// CSS does not let the rule above run unbounded: every flex item carries
/// `min-width: auto` by default, which floors it at its own min-content size.
/// An overflowing row therefore overflows on the web — it does not grind its
/// children down to nothing. This solver has no such floor, and the difference
/// is visible on the first screen of the tutorial.
///
/// The lesson list is a components.ListRow per lesson: a Row with a
/// FlexGrow(1) centre column and a bare `core.Text("4.12")` beside it. The
/// two-line titles overflow a phone's width, so the deficit is shared out, and
/// with no min-content floor the number is compressed to one glyph and wraps
/// down the side of the row as 4 / . / 1 / 2. The same tree on Android and in
/// the browser prints "4.12".
///
/// Fixing it properly means the solver clamping each child at its min-content
/// size, which it cannot do from `bases` alone — those are ideal sizes.
/// GrMobFlexLayout can get the minima (`subview.sizeThatFits(.zero)`) and pass
/// them in, so the shape of the change is clear; what it is not is small, since
/// every Row on this host lays out through here and ios/verify's fixtures,
/// internal/pinfixture and wasm/verify all encode the current model. It is
/// recorded as its own piece of work rather than done in passing.
///
/// The portable declaration in the meantime is `core.FlexShrink(0)` on a child
/// whose size really is not negotiable — which is what a row number is, on
/// every host, and what examples/tutorial's lessonRow now says.
struct GrMobFlexSolver {
    let spacing: CGFloat
    let justify: String

    /// What every child gets and where the run of them starts.
    struct Resolved: Equatable {
        /// Final main-axis size per child, in child order.
        var mains: [CGFloat]
        /// Offset of the first child from the container's leading edge.
        var leading: CGFloat
        /// Extra space inserted between children, on top of `spacing`.
        var gap: CGFloat
    }

    /// A proposal's extent as an offer the layout can lay out against, or
    /// nil when there is none. SwiftUI proposes nil to ask for an ideal size
    /// and infinity to probe for a maximum; neither is a width to wrap text
    /// at or to fill, so both come back as "no offer". The one rule, shared by
    /// the flex and wrap layouts so they cannot drift on what counts as a
    /// definite offer.
    static func definite(_ offered: CGFloat?) -> CGFloat? {
        guard let offered, offered.isFinite else { return nil }
        return offered
    }

    /// The size the run of children wants with no growing or shrinking.
    func natural(bases: [CGFloat]) -> CGFloat {
        bases.reduce(0, +) + spacing * CGFloat(max(bases.count - 1, 0))
    }

    /// The container's own main extent.
    ///
    /// Hug unless something claims the space — a grower, or a
    /// justify-content that positions against the far edge — and never
    /// exceed a definite offer. That is not a CSS rule (a CSS flex container
    /// is a block box and fills its line); it is what SwiftUI stacks do, and
    /// matching it keeps layouts written before GrMobFlexLayout existed
    /// rendering the way they already did.
    ///
    /// `offered` is nil for an unspecified *or* infinite proposal: an
    /// infinite one is SwiftUI probing for a maximum, not an offer to fill
    /// the universe.
    func containerMain(offered: CGFloat?, bases: [CGFloat], weights: [CGFloat]) -> CGFloat {
        let natural = natural(bases: bases)
        guard let offered, offered.isFinite else { return natural }
        let grows = weights.contains { $0 > 0 }
        if grows || justifyClaimsFreeSpace || offered < natural { return offered }
        return natural
    }

    /// The justify-content values that make the container fill a definite
    /// offer instead of hugging its children: everything the leftover space
    /// can be *spent* on. Only flex-start cannot spend it, since packing to
    /// the leading edge looks identical at either size.
    ///
    /// A fifth copy of core's list, and the one that is not a switch — so it
    /// is pinned by reading the array literal rather than switch arms. See
    /// TestSwiftJustifyClaimsFreeSpaceClassifiesEveryJustifyContent in
    /// mobile/verify, which requires this list plus flex-start to be exactly
    /// core.JustifyContents(): a seventh value added to core has to be
    /// classified here, not merely defaulted. Keep it one flat array of string
    /// literals on a single line.
    var justifyClaimsFreeSpace: Bool {
        ["center", "flex-end", "space-between", "space-around", "space-evenly"].contains(justify)
    }

    /// `shrinks` is one flex-shrink factor per child; nil means the CSS
    /// default of 1 for every one of them.
    ///
    /// Defaulted rather than required because most callers have no per-child
    /// factor to give and "all 1" is what every one of them meant before the
    /// parameter existed — a required argument would have turned a behaviour
    /// that did not change into a diff at every call site.
    func resolve(main: CGFloat, bases: [CGFloat], weights: [CGFloat],
                 shrinks: [CGFloat]? = nil) -> Resolved {
        let n = bases.count
        guard n > 0 else { return Resolved(mains: [], leading: 0, gap: 0) }

        let free = main - natural(bases: bases)
        var mains = bases

        let totalWeight = weights.reduce(0, +)
        if free > 0, totalWeight > 0 {
            for i in 0..<n {
                mains[i] += free * weights[i] / totalWeight
            }
            return Resolved(mains: mains, leading: 0, gap: 0)
        }
        if free < 0 {
            // Overflow: shrink in proportion to the child's base size SCALED BY
            // its shrink factor, which is what CSS states. With every factor at
            // 1 — the default, and what a nil `shrinks` means — the factors
            // cancel and this is the plain proportional-to-base rule it used to
            // be, which is why no existing case moves.
            //
            // A run whose scaled bases are all zero cannot shrink: either every
            // child has a zero factor (they all keep their size and the
            // container overflows, which is the instruction) or every base is
            // zero (there is nothing to take). The guard covers both and also
            // avoids dividing by zero.
            let factors = shrinks ?? Array(repeating: 1, count: n)
            let scaled = (0..<n).map { bases[$0] * (factors.count > $0 ? factors[$0] : 1) }
            let totalScaled = scaled.reduce(0, +)
            if totalScaled > 0 {
                for i in 0..<n {
                    mains[i] = max(0, bases[i] + free * scaled[i] / totalScaled)
                }
            }
            return Resolved(mains: mains, leading: 0, gap: 0)
        }

        // Nothing grew: the leftover becomes position, per justify-content.
        return Resolved(mains: mains,
                        leading: leading(free: free, count: n),
                        gap: gap(free: free, count: n))
    }

    /// The offset of the first child from the leading edge.
    ///
    /// This and `gap` below split one question between them: justify-content
    /// spends the leftover space either *before* the run of children or
    /// *between* them, and most values spend it entirely on one or the other.
    /// So each dispatch lists all six values and each returns 0 for the ones
    /// the other answers for — which is why a value missing from both would be
    /// invisible rather than obvious. It would simply pack to the start.
    ///
    /// Both are held to core.JustifyContents() by
    /// TestSwiftFlexSolverCoversEveryJustifyContent in mobile/verify. One arm
    /// per line, string literals first, `default:` last, and the arms that
    /// duplicate `default:`'s body stay spelled out.
    func leading(free: CGFloat, count n: Int) -> CGFloat {
        guard free > 0, n > 0 else { return 0 }
        switch justify {
        // Nothing before the first child. flex-start packs to the leading edge
        // and spends nothing; space-between spends the whole leftover on the
        // gaps, which is `gap`'s half of the answer.
        case "flex-start", "space-between": return 0
        case "center": return free / 2
        case "flex-end": return free
        // CSS space-around gives each item an equal margin on both sides, so
        // the two edge gaps are half an inner gap. space-evenly makes all
        // n+1 gaps equal. The Spacer emulation this replaced could express
        // neither, and rendered both as space-evenly.
        case "space-around": return free / CGFloat(2 * n)
        case "space-evenly": return free / CGFloat(n + 1)
        default: return 0
        }
    }

    /// Extra space inserted between children, on top of `spacing`. See
    /// `leading` above for why both dispatches list all six values.
    func gap(free: CGFloat, count n: Int) -> CGFloat {
        guard free > 0, n > 0 else { return 0 }
        switch justify {
        // The three that spend the leftover on position rather than on
        // spacing: the children stay packed against each other and only the
        // whole run moves, so the inter-item gap is the container's `spacing`
        // and this adds nothing to it.
        case "flex-start", "center", "flex-end": return 0
        case "space-between": return n > 1 ? free / CGFloat(n - 1) : 0
        case "space-around": return free / CGFloat(n)
        case "space-evenly": return free / CGFloat(n + 1)
        default: return 0
        }
    }

    /// Cross-axis placement of one child within the container's cross extent.
    ///
    /// Held to core.AlignItemsValues() by
    /// TestSwiftCrossOffsetCoversEveryAlignItems in mobile/verify. One arm per
    /// line, string literals first, `default:` last; the arms returning 0 are
    /// not redundant with `default:` in meaning, only in value, and folding
    /// them away would lose the distinction between "handled" and "unknown".
    ///
    /// The "start" and "end" labels are not AlignItems values — they are
    /// core.AlignStart and core.AlignEnd arriving through GrMobFlexStack's
    /// `align` fallback, which a Column consults when AlignItems is unset.
    static func crossOffset(align: String, child: CGFloat, extent: CGFloat) -> CGFloat {
        switch align {
        case "flex-start", "start": return 0
        case "center": return max(0, (extent - child) / 2)
        case "flex-end", "end": return max(0, extent - child)
        // A stretched child was already given the whole cross extent by the
        // caller, so child == extent and every formula above collapses to 0
        // anyway. Listed rather than left to `default:` so the value reads as
        // handled-elsewhere instead of unconsidered.
        case "stretch": return 0
        default: return 0
        }
    }
}

/// Line breaking for a Row with core.FlexWrap(true) — CSS `flex-wrap: wrap`
/// on the horizontal axis, as pure arithmetic for the same reason
/// GrMobFlexSolver is: the SwiftUI Layout that uses it (GrMobWrapLayout in
/// Renderer.swift) can only run in a view hierarchy, while the part that is
/// easy to get wrong — where the breaks fall — is a function from numbers to
/// numbers that ios/verify checks on a plain macOS host.
///
/// The model is CSS flex-line collection with `flex-shrink: 0` on every item:
///
/// ```
/// a child joins the current line if  used + spacing + width <= available
/// otherwise it starts a new line
/// a child wider than `available` gets a line to itself — never shrunk,
///   never dropped, exactly as CSS overflows it
/// no definite width offered -> everything on one line (nothing to wrap
///   against; matches a non-wrapping Row's hug behavior)
/// ```
///
/// Growing is deliberately absent: FlexGrow on a wrapped child would have to
/// distribute each line's leftover, which no caller needs yet and which CSS
/// itself only does per line. Children keep their ideal widths.
struct GrMobWrapSolver {
    let spacing: CGFloat

    /// The width the run of children wants on a single line.
    func natural(widths: [CGFloat]) -> CGFloat {
        widths.reduce(0, +) + spacing * CGFloat(max(widths.count - 1, 0))
    }

    /// Child indices grouped into lines, in order. Every index appears exactly
    /// once; an empty input yields no lines.
    func lines(widths: [CGFloat], available: CGFloat?) -> [[Int]] {
        guard !widths.isEmpty else { return [] }
        guard let available, available.isFinite else { return [Array(widths.indices)] }

        var lines: [[Int]] = []
        var current: [Int] = []
        var used: CGFloat = 0
        for (i, w) in widths.enumerated() {
            let needed = current.isEmpty ? w : used + spacing + w
            // The epsilon absorbs the float noise a chain of dp-to-pt
            // conversions leaves behind, so a line that fits exactly is not
            // broken by a rounding error.
            if !current.isEmpty && needed > available + 0.0001 {
                lines.append(current)
                current = [i]
                used = w
            } else {
                current.append(i)
                used = needed
            }
        }
        lines.append(current)
        return lines
    }
}
