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
/// its content size) and the default `flex-shrink: 1`:
///
/// ```
/// base_i    = subview ideal size along the axis
/// natural   = sum(base_i) + spacing * (n - 1)
/// free      = container - natural
/// free > 0  -> size_i = base_i + free * weight_i / sum(weight)   [grow]
/// free < 0  -> size_i = base_i + free * base_i   / sum(base)     [shrink]
/// leftover  -> distributed by justify-content as offsets, not sizes
/// ```
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

    func resolve(main: CGFloat, bases: [CGFloat], weights: [CGFloat]) -> Resolved {
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
            // Overflow: shrink in proportion to base size, which is what
            // `flex-shrink: 1` (the CSS default, and the only value the Go
            // DSL's renderers honor) computes. A run of zero-width bases
            // cannot shrink, so the guard also avoids dividing by zero.
            let totalBase = bases.reduce(0, +)
            if totalBase > 0 {
                for i in 0..<n {
                    mains[i] = max(0, bases[i] + free * bases[i] / totalBase)
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
