import SwiftUI

/// Node-tree → SwiftUI mapping.
///
/// The design deliberately leans on SwiftUI's own diffing for everything the
/// Go reconciler doesn't do: view identity across updates (ForEach ids are
/// the node instances), state retention in unchanged siblings, animation
/// plumbing, and accessibility semantics (via native controls). The Go side
/// only has to keep the data tree correct; nothing here caches views or
/// paths.
///
/// Identity: a child's ForEach id is its explicit key when set, otherwise the
/// node's object identity. In-place mutations (`update-props`/`update-style`)
/// keep the instance, so identity — and any local view state — survives;
/// `replace` swaps in a fresh instance, so identity changes and SwiftUI
/// resets the subtree, which is exactly the Go reconciler's intent.

// The runtime rides the environment (the analog of Renderer.kt's
// CompositionLocal) so leaf views can dispatch events without prop-drilling.
private struct GrMobRuntimeKey: EnvironmentKey {
    static let defaultValue: GrMobRuntime? = nil
}

extension EnvironmentValues {
    var grMobRuntime: GrMobRuntime? {
        get { self[GrMobRuntimeKey.self] }
        set { self[GrMobRuntimeKey.self] = newValue }
    }
}

struct GrMobRoot: View {
    let runtime: GrMobRuntime

    var body: some View {
        if let root = runtime.store.root {
            // Pin the tree to the window's top-leading corner: SwiftUI centers
            // a smaller-than-window view by default, but GrMob's layout
            // model (like the Android renderer's root Column) flows content
            // from the top and lets the tree decide its own extent.
            RenderNode(node: root)
                .environment(\.grMobRuntime, runtime)
                // The style layer's gesture modifier dispatches through this
                // plain closure instead of the runtime type; see
                // GrMobDispatchKey for why the indirection exists.
                .environment(\.grMobDispatch) { [runtime] id in runtime.click(id) }
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        }
    }
}

/// `grow` carries the parent-scope sizing (main-axis FlexGrow) that only the
/// parent can determine, since it knows the stack axis; see GrMobGrow.
struct RenderNode: View {
    let node: GrMobNode
    var grow: GrMobGrow = .none

    var body: some View {
        let style = node.style
        if style?.display == "none" {
            // Not rendered at all; "hidden" keeps space via opacity(0) in grMobBox.
        } else {
            switch node.type {
            case "Text": GrMobText(node: node, grow: grow)
            case "Button": GrMobButton(node: node, grow: grow)

            case "Input": GrMobTextField(node: node, grow: grow)
            case "InputPassword": GrMobTextField(node: node, grow: grow, password: true)
            case "NumericInput": GrMobTextField(node: node, grow: grow, numeric: true)
            case "TextArea": GrMobTextField(node: node, grow: grow, multiline: true)
            case "Checkbox": GrMobCheckbox(node: node, grow: grow)
            case "Slider": GrMobSlider(node: node, grow: grow)
            case "TextGrid": GrMobTextGrid(node: node, grow: grow)
            // A row reached on its own (never from core.TextGrid, which draws
            // its rows itself) still renders as a line of runs.
            case "GridRow": GrMobGridRow(node: node, base: nil)

            case "Row": GrMobRow(node: node, grow: grow)
            case "Column", "Card": GrMobColumn(node: node, grow: grow) // Card = Column whose Go theme style carries the card look
            case "List": GrMobList(node: node, grow: grow)
            case "Box": ZStack(alignment: .topLeading) { PlainChildren(node: node) }
                .grMobBox(node.style, grow: grow,
                            onTap: node.stringProp("onClick"),
                            onLongPress: node.stringProp("onLongPress"))
            case "Spacer": Color.clear.frame(width: CGFloat(node.intProp("size")), height: CGFloat(node.intProp("size")))
            case "Scroll": GrMobScroll(node: node, grow: grow)
            case "SafeArea":
                // SwiftUI already lays out inside the safe area by default, so
                // this is a grouping box — the node exists so Go apps can be
                // explicit about it and so an ignoresSafeArea escape hatch has
                // a home later. One thing does reach past the inset: the
                // node's background, painted edge to edge so a screen that
                // colours its safe area (components.Screen forwards its
                // background here) has no light strip under the status bar.
                // The box's own background is painted by grMobBox as well,
                // inside the inset; the two are the same colour, so the
                // second coat is invisible and keeps the box's clip/border
                // layering exactly as every other node has it.
                ZStack(alignment: .topLeading) { PlainChildren(node: node) }
                    .grMobBox(node.style, grow: grow)
                    .background((node.style?.background ?? Color.clear).ignoresSafeArea())

            case "TabView": GrMobTabView(node: node, grow: grow)
            case "Modal": GrMobModal(node: node)
            case "Image":
                // The AccessibilityLabel style prop travels through grMobBox;
                // the legacy "alt" prop fills in only when no style label is set
                // (an unconditional outer accessibilityLabel would override the
                // box's label with an empty string).
                AsyncImage(url: URL(string: node.stringProp("src"))) { image in
                    grMobScaled(image, mode: node.stringProp("contentMode"))
                } placeholder: {
                    ProgressView()
                }
                .grMobBox(node.style, grow: grow,
                            onTap: node.stringProp("onClick"),
                            onLongPress: node.stringProp("onLongPress"))
                .grMobAltLabel(
                    (node.style?.accessibilityLabel ?? "").isEmpty
                        ? node.stringProp("alt") : "")

            // Camera capture needs an AVFoundation integration pass of its
            // own; until then render the styled surface and any overlay so
            // layouts hold up.
            case "CameraView": ZStack(alignment: .topLeading) { PlainChildren(node: node) }.grMobBox(node.style, grow: grow)

            // Fragment and Theme are grouping nodes with no visual box of
            // their own: Group flattens the children into whatever stack
            // scope we're currently in.
            case "Fragment", "Theme": Group { PlainChildren(node: node) }

            // Unknown node type (newer Go core than this runtime): render the
            // children so the subtree isn't a dead end.
            default: VStack(alignment: .leading, spacing: 0) { PlainChildren(node: node) }.grMobBox(node.style, grow: grow)
            }
        }
    }
}

/// core.ContentMode -> SwiftUI image scaling. An absent or unknown mode is
/// fit, which is both core.Image's documented default and what this renderer
/// drew before the prop existed, so existing trees are unchanged.
///
/// `.clipped()` on the two overflowing modes is not cosmetic: CSS object-fit
/// and Compose's ContentScale.Crop both crop to the box, and an uncropped
/// SwiftUI image would paint over its siblings instead.
///
/// Every mode core declares is listed explicitly, including "fit" — whose
/// body the `default` arm below repeats verbatim. The repetition is deliberate
/// and must not be folded away: `default` swallows an unrecognized mode
/// silently, so a fifth ContentMode added to core would draw here as fit while
/// both DOM targets fell back to the browser default, and nothing anywhere
/// would fail. Listing the modes makes the coverage readable from outside, and
/// mobile/verify/contentmode_test.go reads it — it compares these case labels
/// with core.ContentModes() under a plain `go test ./...`. Keep one
/// `case "…":` per line and keep the `default` arm; that is the shape the
/// parse requires.
@ViewBuilder
private func grMobScaled(_ image: Image, mode: String) -> some View {
    switch mode {
    case "fit":
        image.resizable().scaledToFit()
    case "fill":
        image.resizable().scaledToFill().clipped()
    case "stretch":
        // Resizable with no aspectRatio: the image takes the frame exactly,
        // distorting. Nothing to clip — it never exceeds the box.
        image.resizable()
    case "center":
        // Deliberately NOT resizable, so the bitmap keeps its intrinsic pixel
        // size and is centered by the frame it sits in.
        image.clipped()
    default:
        // Absent (core.imageNode omits the prop entirely) or a mode this build
        // of the runtime predates. Same drawing as "fit" above.
        image.resizable().scaledToFit()
    }
}

extension GrMobNode {
    /// ForEach identity: explicit key when present, object identity otherwise
    /// (see the header comment on why object identity is the right default).
    var viewID: AnyHashable {
        key.isEmpty ? AnyHashable(ObjectIdentifier(self)) : AnyHashable(key)
    }
}

/// Children of a non-flex container (no grow, no justify-content emulation).
private struct PlainChildren: View {
    let node: GrMobNode

    var body: some View {
        ForEach(node.children, id: \.viewID) { child in
            RenderNode(node: child)
        }
    }
}

// ---------------------------------------------------------------------------
// Flex containers
// ---------------------------------------------------------------------------

private struct GrMobRow: View {
    let node: GrMobNode
    let grow: GrMobGrow

    var body: some View {
        let s = node.style
        // core.FlexWrap(true) asks for CSS flex-wrap: children that do not fit
        // continue on the next line instead of being shrunk onto one. The
        // flex stack cannot do that — it is a single-line algorithm that
        // shrinks proportionally, so a row of chips wider than the screen
        // squeezed every chip's label until it broke mid-word. The wrap
        // layout keeps each child at its ideal size and breaks lines instead;
        // Gap serves as both the in-line and the between-line spacing, as it
        // does in CSS and in the Android FlowRow.
        if s?.flexWrap == "wrap" {
            GrMobWrapLayout(spacing: s?.gap ?? 0) {
                FlexChildren(node: node, axis: .horizontal)
            }
            .grMobBox(s, grow: grow,
                        onTap: node.stringProp("onClick"),
                        onLongPress: node.stringProp("onLongPress"),
                        axis: .horizontal)
        } else {
            GrMobFlexStack(axis: .horizontal, style: s) {
                FlexChildren(node: node, axis: .horizontal)
            }
            .grMobBox(s, grow: grow,
                        onTap: node.stringProp("onClick"),
                        onLongPress: node.stringProp("onLongPress"),
                        axis: .horizontal)
        }
    }
}

/// The Layout behind a wrapping Row. Line breaking is GrMobWrapSolver's
/// (GrMobFlex.swift); this only measures, asks where the breaks fall, and
/// places. Children are measured with an unspecified proposal so each reports
/// its ideal size — a chip is as wide as its label, never as wide as the line.
///
/// Width: when the offer is definite and more than one line results, the
/// layout takes the whole offer, since its lines are laid out against it.
/// When everything fits on one line it hugs, the way a non-wrapping Row does,
/// so switching FlexWrap on does not move a row that never needed to wrap.
private struct GrMobWrapLayout: Layout {
    let spacing: CGFloat

    private var solver: GrMobWrapSolver { GrMobWrapSolver(spacing: spacing) }

    private func ideal(_ subviews: Subviews) -> [CGSize] {
        subviews.map { $0.sizeThatFits(.unspecified) }
    }

    private func lineHeight(_ line: [Int], _ sizes: [CGSize]) -> CGFloat {
        line.map { sizes[$0].height }.max() ?? 0
    }

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        guard !subviews.isEmpty else { return .zero }
        let sizes = ideal(subviews)
        let widths = sizes.map(\.width)
        // An infinite proposal is SwiftUI probing for a maximum, not an offer
        // to wrap against; treat it like no offer at all.
        let available = GrMobFlexSolver.definite(proposal.width)
        let lines = solver.lines(widths: widths, available: available)
        let natural = solver.natural(widths: widths)
        let width: CGFloat
        if let available, lines.count > 1 {
            width = available
        } else if let available {
            width = min(natural, available)
        } else {
            width = natural
        }
        // Two statements rather than one expression: the closure-plus-operator
        // chain is exactly the shape the Swift type checker times out on.
        let contentHeight: CGFloat = lines.map { lineHeight($0, sizes) }.reduce(0, +)
        let lineGaps: CGFloat = spacing * CGFloat(max(lines.count - 1, 0))
        let height = contentHeight + lineGaps
        return CGSize(width: width, height: height)
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        guard !subviews.isEmpty else { return }
        let sizes = ideal(subviews)
        // Broken against `bounds`, not `proposal`: bounds is the width
        // actually being drawn into, which the parent may have changed.
        let lines = solver.lines(widths: sizes.map(\.width), available: bounds.width)
        var y = bounds.minY
        for line in lines {
            var x = bounds.minX
            for i in line {
                subviews[i].place(at: CGPoint(x: x, y: y), anchor: .topLeading,
                                  proposal: ProposedViewSize(sizes[i]))
                x += sizes[i].width + spacing
            }
            y += lineHeight(line, sizes) + spacing
        }
    }
}

private struct GrMobColumn: View {
    let node: GrMobNode
    let grow: GrMobGrow

    var body: some View {
        let s = node.style
        GrMobFlexStack(axis: .vertical, style: s) {
            FlexChildren(node: node, axis: .vertical)
        }
        .grMobBox(s, grow: grow,
                    onTap: node.stringProp("onClick"),
                    onLongPress: node.stringProp("onLongPress"),
                    axis: .vertical)
    }
}

/// The children of a flex container, each tagged with the flex weight its
/// parent's layout needs.
///
/// Two channels carry the same number, and both are required. The layout
/// value is what GrMobFlexStack reads to divide leftover space; the
/// GrMobGrow flags are what make the child actually accept the size it is
/// then proposed (see GrMobGrow). Cross-axis stretch rides the second
/// channel only — it is a property of the container, so the layout already
/// knows it.
private struct FlexChildren: View {
    let node: GrMobNode
    let axis: Axis

    var body: some View {
        // The same cross-axis read GrMobFlexStack does, and it has to be the
        // same one: the layout decides where a stretched child is placed and
        // this decides whether the child accepts the size it is proposed, so
        // the two disagreeing means the placement promises a fill nothing
        // applies. This read used to be `alignItems` alone, so a Column
        // written `Align(AlignStretch)` without AlignItems laid out as
        // stretched and rendered unstretched. (Compose's isColumnStretch has
        // carried the fallback for both spellings for a while; this is iOS
        // catching up.)
        //
        // Vertical only, exactly as in GrMobFlexStack: Align is a
        // text-alignment concept and has never been read for a Row's vertical
        // cross axis, and honoring it there now would move existing rows.
        let cross = axis == .vertical ? crossAxisValue(node.style)
                                      : (node.style?.alignItems ?? "")
        let stretch = axis == .vertical ? columnStretches(cross) : cross == "stretch"
        ForEach(node.children, id: \.viewID) { child in
            let weight = child.style?.flexGrow ?? 0
            // A child that keeps its own width opts out of the stretch on
            // both channels: the layout must not propose it the full cross
            // extent (GrMobFlexHugs), and it must not accept one (no
            // fillWidth). Vertical only, since that is the only axis with
            // a stretch default — see hugsContent.
            let hugs = axis == .vertical && hugsContent(child.style)
            RenderNode(node: child, grow: fill(weight: weight, stretch: stretch && !hugs))
                .layoutValue(key: GrMobFlexWeight.self, value: weight)
                .layoutValue(key: GrMobFlexHugs.self, value: hugs)
        }
    }

    private func fill(weight: CGFloat, stretch: Bool) -> GrMobGrow {
        var g = GrMobGrow()
        if weight > 0 {
            if axis == .horizontal { g.fillWidth = true } else { g.fillHeight = true }
        }
        if stretch {
            if axis == .horizontal { g.fillHeight = true } else { g.fillWidth = true }
        }
        return g
    }
}

/// Per-child flex weight, handed from FlexChildren to GrMobFlexLayout.
///
/// A LayoutValueKey rather than a stored array on the layout because a Layout
/// receives its children as opaque proxies: `subviews[i]` cannot be traced
/// back to the GrMobNode it came from, and a parallel array would silently
/// mis-align the moment SwiftUI flattened a Group or dropped an empty view.
private struct GrMobFlexWeight: LayoutValueKey {
    static let defaultValue: CGFloat = 0
}

/// Whether a child of a stretched Column keeps its own width (see
/// hugsContent). Carried the same way as the weight, for the same reason:
/// the layout sees subviews, not nodes, and this is a per-child decision it
/// has to make when it proposes the cross size.
private struct GrMobFlexHugs: LayoutValueKey {
    static let defaultValue = false
}

/// The flex containers' layout: a SwiftUI `Layout` running the CSS algorithm.
///
/// SwiftUI's own stacks cannot express three things GrMob's Go DSL declares,
/// which is what this replaces HStack/VStack for:
///
///  1. **Proportional FlexGrow.** A stack has no Compose-style weight. The
///     previous approach — an infinity frame on every grower — makes SwiftUI
///     split leftover space *equally*, so `FlexGrow(3)` beside `FlexGrow(1)`
///     rendered 50/50 instead of 75/25.
///  2. **`AlignItems: "stretch"`.** A stack's alignment only *places* a
///     child; stretch has to *size* it, which only the container's layout can
///     propose.
///  3. **Exact `justify-content`.** The Spacer emulation could not tell
///     space-around from space-evenly (CSS gives space-around half-width
///     gaps at the two edges), and every Spacer was an extra view in the
///     tree that the app never declared.
///
/// The arithmetic lives in GrMobFlexSolver (GrMobFlex.swift), which is pure
/// and therefore testable off-device; what stays here is the part that needs
/// SwiftUI — measuring subviews and placing them.
struct GrMobFlexStack<Content: View>: View {
    let axis: Axis
    let style: GrMobStyle?
    @ViewBuilder let content: Content

    var body: some View {
        GrMobFlexLayout(
            axis: axis,
            spacing: style?.gap ?? 0,
            justify: style?.justifyContent ?? "",
            // AlignItems governs cross-axis placement; crossAxisValue folds
            // in the DSL's simpler Align ("center"/"end") as the fallback,
            // but the fallback applies only where it ever has — a Column's
            // horizontal cross axis. Align is a text-alignment concept and
            // was never read for a Row's vertical one; honoring it there now
            // would move existing rows.
            crossAlign: axis == .vertical ? crossAxisValue(style)
                                          : (style?.alignItems ?? "")
        ) {
            content
        }
    }
}

private struct GrMobFlexLayout: Layout {
    let axis: Axis
    let spacing: CGFloat
    let justify: String
    let crossAlign: String

    private var solver: GrMobFlexSolver {
        GrMobFlexSolver(spacing: spacing, justify: justify)
    }

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        guard !subviews.isEmpty else { return .zero }
        // The cross extent on offer is handed to every child as a bound (see
        // baseMains); nil when the parent is asking for an ideal size.
        let crossBound = GrMobFlexSolver.definite(crossOf(proposal))
        let bases = baseMains(subviews, crossBound: crossBound)
        let weights = subviews.map { $0[GrMobFlexWeight.self] }
        let offered = mainOf(proposal)
        let main = solver.containerMain(offered: offered, bases: bases, weights: weights)
        let resolved = solver.resolve(main: main, bases: bases, weights: weights)

        // Cross size is re-measured at each child's *final* main size: a Text
        // that had to shrink wraps to more lines, and asking it before the
        // main axis was settled would under-report its height.
        let cross = zip(subviews, resolved.mains)
            .map { crossOf($0.sizeThatFits(proposed(main: $1, cross: crossBound))) }
            .max() ?? 0
        return size(main: main, cross: cross)
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        guard !subviews.isEmpty else { return }
        // Resolved against `bounds`, not `proposal`: the parent is free to
        // hand over a different size than the one sizeThatFits asked for, and
        // bounds is the size that is actually being drawn into.
        let containerCross = crossOf(bounds.size)
        let bases = baseMains(subviews, crossBound: containerCross)
        let weights = subviews.map { $0[GrMobFlexWeight.self] }
        let resolved = solver.resolve(main: mainOf(bounds.size), bases: bases, weights: weights)
        // The same read FlexChildren makes, and it has to be the same one:
        // an unset value stretches on the vertical axis (the CSS default the
        // DOM targets have always drawn) and packs on the horizontal one.
        let stretch = axis == .vertical ? columnStretches(crossAlign) : crossAlign == "stretch"

        var offset = resolved.leading
        for (i, subview) in subviews.enumerated() {
            let childMain = resolved.mains[i]
            // Every child is proposed the container's cross extent, as a
            // bound (see baseMains). Stretch and non-stretch differ in what
            // the child does with it, not in what it is told: a stretched
            // child carries a flexible frame from FlexChildren and accepts
            // the whole extent, an unstretched one — or one that hugs its
            // content by its own style — takes what it needs.
            let childProposal = proposed(main: childMain, cross: containerCross)
            let childCross = stretch && !subview[GrMobFlexHugs.self]
                ? containerCross
                : crossOf(subview.sizeThatFits(childProposal))

            let mainPos = mainOf(bounds.origin) + offset
            let crossPos = crossOf(bounds.origin) + GrMobFlexSolver.crossOffset(
                align: crossAlign, child: childCross, extent: containerCross)
            subview.place(
                at: axis == .horizontal ? CGPoint(x: mainPos, y: crossPos)
                                        : CGPoint(x: crossPos, y: mainPos),
                anchor: .topLeading,
                proposal: childProposal
            )
            offset += childMain + spacing + resolved.gap
        }
    }

    /// Each child's content size along the main axis (CSS `flex-basis: auto`).
    ///
    /// Unspecified on the main axis so the child reports its ideal size
    /// rather than accepting whatever the container was offered — but bounded
    /// on the cross axis by what the container itself was offered. That bound
    /// is the difference between a Column and a stack of single-line labels:
    /// a Text proposed no width reports the width of its longest line and
    /// never wraps, so a Column of paragraphs measured with a fully
    /// unspecified proposal came out wider than the screen, and the overflow
    /// was centred by the root frame — every screen edge sat a few points
    /// off the window. This is also what Compose does: a Column measures each
    /// child with its own incoming maxWidth as the child's maxWidth, and the
    /// child is free to be narrower. A Row with FlexGrow children fills that
    /// bound (GrMobFlexSolver.containerMain), a paragraph wraps to it, and a
    /// short label ignores it.
    ///
    /// ```
    ///   Column proposed (W, H)
    ///     child base:   sizeThatFits(width: W,  height: nil)   <- wraps at W
    ///     child final:  place(width: W, height: resolved)      <- same bound
    /// ```
    private func baseMains(_ subviews: Subviews, crossBound: CGFloat?) -> [CGFloat] {
        subviews.map { mainOf($0.sizeThatFits(proposed(main: nil, cross: crossBound))) }
    }

    // -- axis-agnostic helpers ---------------------------------------------

    private func mainOf(_ s: CGSize) -> CGFloat { axis == .horizontal ? s.width : s.height }
    private func crossOf(_ s: CGSize) -> CGFloat { axis == .horizontal ? s.height : s.width }
    private func mainOf(_ p: CGPoint) -> CGFloat { axis == .horizontal ? p.x : p.y }
    private func crossOf(_ p: CGPoint) -> CGFloat { axis == .horizontal ? p.y : p.x }
    private func mainOf(_ p: ProposedViewSize) -> CGFloat? { axis == .horizontal ? p.width : p.height }
    private func crossOf(_ p: ProposedViewSize) -> CGFloat? { axis == .horizontal ? p.height : p.width }

    private func size(main: CGFloat, cross: CGFloat) -> CGSize {
        axis == .horizontal ? CGSize(width: main, height: cross)
                            : CGSize(width: cross, height: main)
    }

    /// A proposal built per axis; nil on either axis means "unspecified".
    private func proposed(main: CGFloat?, cross: CGFloat?) -> ProposedViewSize {
        axis == .horizontal ? ProposedViewSize(width: main, height: cross)
                            : ProposedViewSize(width: cross, height: main)
    }
}

/// core.Scroll: a vertically scrolling column.
///
/// A ScrollView proposes its content no height at all, so a FlexGrow child
/// inside one has nothing to grow into and wraps its content — which is why
/// components.Screen{Fill, Scroll}, the ordinary form-shaped screen, stopped
/// short of the bottom on iOS: the column's background ended where its last
/// field did. Compose had the same collapse (worse: a weight under an
/// unbounded constraint resolves to zero) and fixed it by measuring the
/// viewport first; this is the SwiftUI spelling of the same fix. The scroll
/// view's own frame is read through a preference — the frame *is* the
/// viewport, and reading it off a background GeometryReader leaves the
/// ScrollView's sizing untouched, where wrapping the whole thing in a
/// GeometryReader would make it greedy on both axes — and a grow child is
/// given that height as a floor (GrMobGrow.minHeight): it fills the screen
/// when the content is short, so its background covers the viewport, and
/// grows past it, scrolling, when the content is tall. That is what the DOM
/// gives `flex-grow` under `overflow: auto` too.
///
/// Cross-axis stretch applies as in any vertical container: the Scroll is a
/// flex column on the web, so its children fill its width unless they hug.
private struct GrMobScroll: View {
    let node: GrMobNode
    let grow: GrMobGrow
    @State private var viewport: CGFloat = 0

    var body: some View {
        let stretch = columnStretches(crossAxisValue(node.style))
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                ForEach(node.children, id: \.viewID) { child in
                    let grows = (child.style?.flexGrow ?? 0) > 0
                    RenderNode(node: child, grow: GrMobGrow(
                        fillWidth: stretch && !hugsContent(child.style),
                        minHeight: grows ? viewport : 0))
                }
            }
        }
        .background(GeometryReader { geo in
            Color.clear.preference(key: GrMobViewportHeight.self, value: geo.size.height)
        })
        .onPreferenceChange(GrMobViewportHeight.self) { viewport = $0 }
        .grMobKeyboardAware(node.boolProp("keyboardAware"))
        .grMobBox(node.style, grow: grow)
    }
}

/// The scroll viewport's height, carried up from the GeometryReader in
/// GrMobScroll's background. One value per scroll view, so reduce keeps the
/// latest.
private struct GrMobViewportHeight: PreferenceKey {
    static let defaultValue: CGFloat = 0
    static func reduce(value: inout CGFloat, nextValue: () -> CGFloat) {
        value = nextValue()
    }
}

/// The virtualized sibling of GrMobColumn: LazyVStack materializes only the
/// rows near the viewport as the ScrollView scrolls, so Go can hand over a
/// thousand-row feed as plain data. (SwiftUI's List is deliberately not used:
/// it brings UITableView chrome — separators, insets, selection styling —
/// that GrMob's unopinionated box model doesn't ask for.)
///
/// Go's For helper wraps generated rows in a Fragment node; those wrappers
/// are flattened so each row is an individually lazy item rather than one
/// giant Fragment item. Row identity is viewID (explicit key, else object
/// identity), same as every other children loop.
private struct GrMobList: View {
    let node: GrMobNode
    let grow: GrMobGrow

    var body: some View {
        let s = node.style
        let rows = flattenFragments(node.children)
        ScrollView {
            LazyVStack(alignment: crossAlignmentH(s), spacing: CGFloat(s?.gap ?? 0)) {
                // Cross-axis stretch, the same contract GrMobFlexStack
                // implements for Row/Column. A lazy stack cannot be replaced
                // by a custom Layout — laziness is the whole point of using
                // it — so stretch is expressed the only way it can be here:
                // a flexible frame on each row. There is no main-axis
                // counterpart because a scrolling axis has no leftover space
                // for FlexGrow to divide.
                //
                // Read through crossAxisValue — the same read the stack's
                // alignment makes — so the Style.Align fallback reaches this
                // binding too. It used to test alignItems alone while
                // crossAlignmentH read the fallback, and the two answers
                // disagreed exactly when it mattered: Align: stretch with
                // AlignItems unset took crossAlignmentH's "stretch" arm,
                // whose comment promises this frame does the filling, and
                // then no frame was applied. See crossAxisValue for the pin
                // that keeps the two reads together.
                let stretch = columnStretches(crossAxisValue(s))
                ForEach(rows, id: \.viewID) { row in
                    RenderNode(node: row, grow: stretch && !hugsContent(row.style) ? .horizontal : .none)
                }
            }
            // A Transition declared on the List itself animates row
            // *placement*: keyed rows slide/fade on reorder, insertion, and
            // removal instead of teleporting — the Android renderer's
            // animateItemPlacement analog. Scoped by the row-identity array
            // so only structural changes trigger it; a row's own property
            // changes animate under its own Transition via grMobBox.
            .animation(s?.swiftUIAnimation, value: rows.map(\.viewID))
        }
        .grMobKeyboardAware(node.boolProp("keyboardAware"))
        .grMobBox(s, grow: grow,
                    onTap: node.stringProp("onClick"),
                    onLongPress: node.stringProp("onLongPress"),
                    axis: .vertical)
    }
}

/// Inlines Fragment/Theme grouping nodes so their children become list rows.
private func flattenFragments(_ children: [GrMobNode]) -> [GrMobNode] {
    guard children.contains(where: { $0.type == "Fragment" || $0.type == "Theme" }) else {
        return children
    }
    var out: [GrMobNode] = []
    out.reserveCapacity(children.count)
    for child in children {
        if child.type == "Fragment" || child.type == "Theme" {
            out.append(contentsOf: flattenFragments(child.children))
        } else {
            out.append(child)
        }
    }
    return out
}

/// The effective cross-axis value of a style: AlignItems, else the DSL's
/// simpler Align as the fallback when AlignItems is unset.
///
/// One function because the fallback is read in three places — GrMobFlexStack's
/// crossAlign (on the vertical axis only; the asymmetry is explained there),
/// the lazy list's placement dispatch (crossAlignmentH), and the lazy list's
/// stretch binding — and two of those reads have already come apart once.
/// GrMobList's stretch binding tested alignItems alone while crossAlignmentH
/// read the fallback, so Align: stretch with AlignItems unset landed on the
/// "stretch" arm below — whose comment says the flexible frame does the
/// filling — while the binding that applies that frame saw no stretch at all.
/// Rows placed leading and filled nothing, on both natives, while a Column
/// with the identical style stretched. An equality test has no arms, so the
/// coverage checks in mobile/verify could not see it;
/// TestListStretchFillReadsTheAlignFallback now pins the binding to this
/// helper and this helper to the `align` read instead.
private func crossAxisValue(_ s: GrMobStyle?) -> String {
    let items = s?.alignItems ?? ""
    return items.isEmpty ? (s?.align ?? "") : items
}

/// Whether a vertical container's effective cross-axis value stretches its
/// children. An unset value does: that is the CSS default (`align-items:
/// stretch`) and therefore what the two DOM targets have always drawn — an
/// Input in a Column runs the full width of the screen in the browser, and
/// on a phone it used to hug its placeholder because the flex stack packed
/// unaligned children at the leading edge. Reading "unset" as stretch brings
/// iOS to the picture the web already shows; an explicit "flex-start" still
/// packs. Rows keep packing: the same CSS rule applies to them in principle,
/// but Compose cannot stretch a Row without an intrinsic-height measurement
/// that has real costs inside a List, and the natives move together.
private func columnStretches(_ cross: String) -> Bool {
    cross.isEmpty || cross == "stretch"
}

/// Whether a child of a stretched Column keeps its own width instead.
///
/// Two things exempt a child, and both come from the DOM targets, which are
/// the reference for what the default should look like:
///
///  - An explicit Width. CSS never stretches an item with a definite cross
///    size; here the flexible frame grMobGrow adds would sit outside the
///    fixed one grMobDimension adds and win, so the child has to skip the
///    stretch rather than override it.
///  - An inline Display. The bundled themes give Button (and Badge) an
///    inline display as their way of saying "hug your content", and
///    grmob-runtime.js turns that into `width: fit-content` for exactly this
///    case — a flex column would otherwise spread every button across the
///    screen. components.Button's FullWidth is the documented way to ask
///    for the stretch back (it sets both Width and a block display).
///
/// Text is not exempt: a stretched Text is the same picture as a hugging
/// one (its alignment is a text property), and stretching it is what lets
/// Align(AlignCenter) on a Column child centre its lines.
private func hugsContent(_ s: GrMobStyle?) -> Bool {
    guard let s else { return false }
    return !s.width.isEmpty || s.display == "inline" || s.display == "inline-block"
}

/// AlignItems governs cross-axis placement; the DSL's simpler Align
/// ("center"/"end") acts as a fallback when AlignItems is unset (the
/// crossAxisValue read above).
///
/// Only GrMobList still needs this. Row and Column place their children
/// through GrMobFlexLayout, which computes the offset itself — it has to,
/// since it also has to handle the "stretch" value that no SwiftUI alignment
/// can express. A LazyVStack cannot be replaced by a custom Layout without
/// giving up laziness, so it keeps a native alignment. (The vertical
/// counterpart went with the HStack it served.)
/// Held to core.AlignItemsValues() by
/// TestSwiftCrossAlignmentCoversEveryAlignItems in mobile/verify. One arm per
/// line, string literals first, `default:` last; the arms that duplicate
/// `default:`'s body are deliberate and must not be folded into it.
///
/// The "end" label alongside "flex-end" is not an AlignItems at all — it is
/// core.AlignEnd arriving through crossAxisValue's fallback, which is why
/// this dispatch answers for two vocabularies at once.
private func crossAlignmentH(_ s: GrMobStyle?) -> HorizontalAlignment {
    let v = crossAxisValue(s)
    switch v {
    case "flex-start", "start": return .leading
    case "center": return .center
    case "flex-end", "end": return .trailing
    // Not placement. A stretched row is given the whole cross extent by the
    // flexible frame GrMobList puts on it (see the `stretch` binding in its
    // body), so by the time the stack's own alignment is consulted there is
    // nothing left to align — every alignment would look identical. Listed so
    // that "handled elsewhere" is distinguishable from "not handled".
    case "stretch": return .leading
    default: return .leading
    }
}

// ---------------------------------------------------------------------------
// Leaf components
// ---------------------------------------------------------------------------

private struct GrMobText: View {
    let node: GrMobNode
    let grow: GrMobGrow

    var body: some View {
        Text(node.stringProp("content"))
            .grMobTextStyle(node.style)
            .grMobBox(node.style, grow: grow,
                        onTap: node.stringProp("onClick"),
                        onLongPress: node.stringProp("onLongPress"))
    }
}

extension View {
    /// Text styling shared by Text and the input fields.
    func grMobTextStyle(_ s: GrMobStyle?, defaultSize: CGFloat = 17) -> some View {
        let size = (s?.fontSize ?? 0) > 0 ? s!.fontSize : defaultSize
        return self
            .font(.system(size: size, weight: grMobFontWeight(s?.fontWeight ?? 0)))
            .foregroundStyle(s?.textColor ?? .primary)
            // SwiftUI has line *spacing*, not line height; the difference
            // between the requested height and the font size approximates it.
            .lineSpacing((s?.lineHeight ?? 0) > 0 ? max(CGFloat(s!.lineHeight) - size, 0) : 0)
            .multilineTextAlignment(grMobTextAlignment(s?.align ?? ""))
    }
}

/// Go's Weight constants are the CSS numeric scale (200/400/700...); map the
/// hundreds onto SwiftUI's named weights.
private func grMobFontWeight(_ w: Int) -> Font.Weight {
    switch w {
    case ..<1: .regular
    case ..<200: .ultraLight
    case ..<300: .thin
    case ..<400: .light
    case ..<500: .regular
    case ..<600: .medium
    case ..<700: .semibold
    case ..<800: .bold
    case ..<900: .heavy
    default: .black
    }
}

/// core.TextAlignments -> SwiftUI's TextAlignment.
///
/// Every value listed explicitly, including the two that `default` would have
/// produced anyway. That redundancy is the point and must not be folded away:
/// SwiftUI has no "unset" alignment to fall through to, so an unlisted value
/// is not left alone, it is silently rendered as leading — which is how
/// justified text came to render on Compose and nowhere else. Held to
/// core.TextAlignments() by TestSwiftTextAlignmentCoversEveryTextAlignment in
/// mobile/verify, which parses these arms; keep one arm per line with its
/// string literals first and `default:` last.
///
/// AlignStretch and AlignBaseline are absent by design. They are Alignments
/// that name a cross-axis placement rather than a text alignment, they reach
/// Style.Align through its other role, and core.TextAlignments() leaves them
/// out for exactly that reason — so they fall to `default:` here, which is the
/// same nothing htmlout and the WASM runtime do with them.
private func grMobTextAlignment(_ align: String) -> TextAlignment {
    switch align {
    case "start": .leading
    case "center": .center
    case "end": .trailing
    // SwiftUI's TextAlignment has three members and no justified setting;
    // Text cannot justify at all. This arm exists to say that out loud, not
    // to do something `default:` would not have done — htmlout and the WASM
    // runtime emit text-align:justify and Compose sets TextAlign.Justify, so
    // this is the one target that cannot honor the value, and the difference
    // deserves to be visible here rather than inferred from an absence.
    case "justify": .leading
    default: .leading
    }
}

private struct GrMobButton: View {
    let node: GrMobNode
    let grow: GrMobGrow
    @Environment(\.grMobRuntime) private var runtime

    /// Set by the long-press gesture so the tap that follows the release is
    /// swallowed rather than firing onClick as well.
    ///
    /// A flag rather than gesture arbitration because the two are not in
    /// conflict from SwiftUI's point of view: a `simultaneousGesture` is, by
    /// name, allowed to run alongside the Button's own tap, so a press held
    /// past the threshold and then released would fire both handlers. One
    /// gesture must produce one handler call — combinedClickable's
    /// onClick/onLongClick split gives Android that for free, and the DOM
    /// runtime does the same thing with a `longPressFired` dataset flag.
    @State private var longPressFired = false

    var body: some View {
        let s = node.style
        let onClick = node.stringProp("onClick")
        let onLongPress = node.stringProp("onLongPress")
        // Style properties the Go theme owns are fed into the button's own
        // label/background rather than grMobBox: the control draws its own
        // container, so background/radius/padding belong inside the pressable
        // area (and inside the press feedback), with only margin/size outside.
        Button {
            if longPressFired {
                longPressFired = false
                return
            }
            if !onClick.isEmpty { runtime?.click(onClick) }
        } label: {
            Text(node.stringProp("label"))
                .font(.system(size: (s?.fontSize ?? 0) > 0 ? s!.fontSize : 17,
                              weight: grMobFontWeight(s?.fontWeight ?? 0)))
                .foregroundStyle(s?.textColor ?? .white)
                .padding(paddingOrDefault(s))
                .frame(maxWidth: grow == .horizontal ? .infinity : nil)
        }
        .buttonStyle(GrMobButtonStyle(
            background: s?.background ?? .accentColor,
            radius: (s?.borderRadius ?? 0) > 0 ? s!.borderRadius : 8
        ))
        // core.OnLongPress on a Button. Every other node type gets this from
        // grMobBox's onLongPress argument, but a Button draws its own control
        // and hands grMobBox only margin and size, so the gesture has to be
        // attached here — which is why the prop was documented as wired on
        // both natives while GrMobButton read nothing but onClick.
        //
        // `including:` is how a gesture is conditionally absent in SwiftUI:
        // `.subviews` scopes it away from this view, leaving a button with no
        // onLongPress exactly as it was. 0.5s matches
        // UILongPressGestureRecognizer's default, Android's
        // ViewConfiguration, and the DOM runtime's LONG_PRESS_MS.
        .simultaneousGesture(
            LongPressGesture(minimumDuration: 0.5).onEnded { _ in
                longPressFired = true
                runtime?.click(onLongPress)
            },
            including: onLongPress.isEmpty ? .subviews : .all
        )
        .grMobBox(marginAndSizeOnly(s), grow: grow)
    }

    private func paddingOrDefault(_ s: GrMobStyle?) -> EdgeInsets {
        let p = s?.padding ?? .zero
        if p == .zero { return EdgeInsets(top: 10, leading: 16, bottom: 10, trailing: 16) }
        return p.insets
    }
}

private struct GrMobButtonStyle: ButtonStyle {
    let background: Color
    let radius: CGFloat

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .background(background)
            .clipShape(RoundedRectangle(cornerRadius: radius))
            // The platform has no ripple; dimming on press is the SwiftUI idiom.
            .opacity(configuration.isPressed ? 0.65 : 1)
    }
}

/// Margin + explicit dimensions only — for controls that draw their own box.
/// Mirrors the Android renderer's marginAndSize: reuse grMobBox's ordering
/// by handing it a style stripped of the box-drawing fields.
private func marginAndSizeOnly(_ s: GrMobStyle?) -> GrMobStyle? {
    guard var t = s else { return nil }
    t.background = nil
    t.borderColor = nil
    t.borderWidth = 0
    t.borderRadius = 0
    t.shadow = 0
    t.padding = .zero
    return t
}

private struct GrMobCheckbox: View {
    let node: GrMobNode
    let grow: GrMobGrow
    @Environment(\.grMobRuntime) private var runtime

    var body: some View {
        let cb = node.stringProp("onToggle")
        // iOS has no checkbox control; Toggle (a switch) is the platform
        // idiom for the same bool. Controlled like everything else: the value
        // always comes from Go, the change goes up as a bool event.
        Toggle(isOn: Binding(
            get: { node.boolProp("checked") },
            set: { if !cb.isEmpty { runtime?.toggled(cb, $0) } }
        )) { EmptyView() }
            .labelsHidden()
            .grMobBox(marginAndSizeOnly(node.style), grow: grow)
    }
}

/// A core.Slider: SwiftUI's Slider over Go's [min, max]. Controlled with the
/// same compromise the text fields make — Go's value is shown except while
/// the thumb is being dragged, when the finger's value is, so a status tick
/// arriving mid-drag (a seek bar fed by the audio player) cannot snap the
/// thumb back. onChange goes up on every move; onChangeEnd once, on release,
/// with the final value (core.OnSliderChangeEnd). Both are text callbacks
/// carrying the number; Go parses Swift's String(Double).
private struct GrMobSlider: View {
    let node: GrMobNode
    let grow: GrMobGrow
    @Environment(\.grMobRuntime) private var runtime
    @State private var local: Double = 0
    @State private var editing = false

    private var bounds: ClosedRange<Double> {
        let lower = node.doubleProp("min")
        let upper = node.doubleProp("max")
        return lower...(upper > lower ? upper : lower + 1)
    }

    var body: some View {
        let range = bounds
        let upstream = min(max(node.doubleProp("value"), range.lowerBound), range.upperBound)
        let step = node.doubleProp("step")
        let onChange = node.stringProp("onChange")
        let onChangeEnd = node.stringProp("onChangeEnd")
        let value = Binding<Double>(
            get: { editing ? local : upstream },
            set: { v in
                local = v
                if !onChange.isEmpty { runtime?.textChanged(onChange, String(v)) }
            })
        let edited: (Bool) -> Void = { began in
            if began {
                local = upstream
                editing = true
            } else {
                editing = false
                if !onChangeEnd.isEmpty { runtime?.textChanged(onChangeEnd, String(local)) }
            }
        }
        Group {
            if step > 0 {
                Slider(value: value, in: range, step: step, onEditingChanged: edited)
            } else {
                Slider(value: value, in: range, onEditingChanged: edited)
            }
        }
        .grMobBox(marginAndSizeOnly(node.style), grow: grow)
    }
}

/// A core.TextGrid: a vertical stack of monospace rows, each an
/// AttributedString built from the row's runs. The grid's own style (size,
/// colour) is the base every run inherits; a run's fg/bg and attribute bits
/// override it per run. Rows never wrap — a terminal row is exactly as wide
/// as its cells — so a grid wider than the screen scrolls sideways instead.
///
/// Rows are read straight off the children rather than through RenderNode.
/// The Go reconciler pairs them by index and an update-props on one row
/// mutates that GrMobNode's props alone; with @Observable tracking, only the
/// Text that read those props re-evaluates. That per-row invalidation is the
/// whole reason the grid is a container of rows rather than one prop.
private struct GrMobTextGrid: View {
    let node: GrMobNode
    let grow: GrMobGrow

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            VStack(alignment: .leading, spacing: 0) {
                ForEach(node.children, id: \.viewID) { row in
                    GrMobGridRow(node: row, base: node.style)
                }
            }
        }
        .grMobBox(node.style, grow: grow,
                  onTap: node.stringProp("onClick"),
                  onLongPress: node.stringProp("onLongPress"))
    }
}

/// One row of a core.TextGrid. The runs prop is an array of dictionaries
/// with the short keys core.GridRun's json tags declare: t (text), fg, bg
/// (CSS hex colours, absent to inherit) and a (the Grid* attribute bits:
/// 1 bold, 2 dim, 4 italic, 8 underline, 16 strike). Dim has no direct
/// spelling here either; it fades the run's colour, or the grid's text
/// colour when the run has none, and leaves a run with neither alone.
private struct GrMobGridRow: View {
    let node: GrMobNode
    let base: GrMobStyle?

    var body: some View {
        let size = (base?.fontSize ?? 0) > 0 ? base!.fontSize : 13
        let baseFont = Font.system(size: size, design: .monospaced)
        var text = AttributedString()
        let runs = node.props["runs"] as? [[String: Any]] ?? []
        for run in runs {
            var piece = AttributedString(run["t"] as? String ?? "")
            let a = (run["a"] as? NSNumber)?.intValue ?? 0
            var font = baseFont
            if a & 1 != 0 { font = font.bold() }
            if a & 4 != 0 { font = font.italic() }
            piece.font = font
            var fg = GrMobStyle.parseColor(run["fg"] as? String) ?? base?.textColor
            if a & 2 != 0, let c = fg { fg = c.opacity(0.6) }
            if let fg { piece.foregroundColor = fg }
            if let bg = GrMobStyle.parseColor(run["bg"] as? String) { piece.backgroundColor = bg }
            if a & 8 != 0 { piece.underlineStyle = .single }
            if a & 16 != 0 { piece.strikethroughStyle = .single }
            text.append(piece)
        }
        // An empty row still takes one line, so the rows below it stay on
        // the cell grid: a space in the base font has the line's height and
        // draws nothing.
        if runs.isEmpty {
            var blank = AttributedString(" ")
            blank.font = baseFont
            text.append(blank)
        }
        return Text(text)
            .lineLimit(1)
            .fixedSize(horizontal: true, vertical: false)
    }
}

/// The controlled-input compromise: Go owns the value, but the keyboard needs
/// its keystrokes echoed instantly, and the Go round trip is asynchronous. So
/// the field is locally-owned *while focused* (every edit is sent upstream
/// but late echoes never snap the cursor back), and Go-owned when not focused
/// (an async upstream change — validation rewrites, state restores — lands
/// the moment the user isn't mid-typing).
///
/// The one upstream change that must land mid-focus is a deliberate rewrite —
/// Go clearing the draft after a submit, a validator normalizing the text.
/// Echoes and rewrites are told apart by bookkeeping, not heuristics: every
/// value this field sends upstream is queued, and an upstream change that
/// matches a queued entry is an echo of our own edit (drop the queue through
/// that point — Go may coalesce renders, skipping intermediate values), while
/// one that matches nothing we sent can only be Go speaking for itself, so it
/// wins even while focused. Moving the cursor then is correct: the text under
/// it was replaced.
private struct GrMobTextField: View {
    let node: GrMobNode
    let grow: GrMobGrow
    var password = false
    var numeric = false
    var multiline = false

    @Environment(\.grMobRuntime) private var runtime
    @FocusState private var focused: Bool
    @State private var text = ""
    @State private var pendingEchoes: [String] = []

    var body: some View {
        let upstream = node.stringProp("value")
        let onChange = node.stringProp("onChange")
        let onSubmit = node.stringProp("onSubmit")
        // The keyboard's action key, decided in Go from core.UseFocusOrder:
        // "next" on every field of a declared order but the last. Go also
        // wired the onSubmit above to advance the focus, so this prop only
        // chooses the label — the action itself is an ordinary submit.
        let imeAction = node.stringProp("imeAction")
        let onFocus = node.stringProp("onFocus")
        let onBlur = node.stringProp("onBlur")
        // The imperative half: core.Focus / core.DismissKeyboard reach the
        // screen as these two props. See applyFocusCommand and core/focus.go.
        let focusEpoch = node.intProp("focusEpoch")
        let focusAction = node.stringProp("focusAction")
        let prompt = Text(node.stringProp("placeholder"))

        // While focused the local buffer is authoritative; otherwise render
        // straight from Go. The buffer is seeded from upstream at the moment
        // focus arrives, so editing always starts from the Go value.
        let value = Binding<String>(
            get: { focused ? text : upstream },
            set: { v in
                text = v
                if !onChange.isEmpty {
                    pendingEchoes.append(v)
                    runtime?.textChanged(onChange, v)
                }
            }
        )

        field(value: value, prompt: prompt)
            .focused($focused)
            // The return key dispatches onSubmit as a plain void event — the
            // same channel as a Button tap — and advertises what it will do:
            // "next" for a field with somewhere to go (core.UseFocusOrder),
            // "done" for one that acts on return, and the plain return key
            // for a field that does neither.
            //
            // Next is tested first because it is the more specific claim: Go
            // only stamps it on a field whose onSubmit it wired itself, so the
            // label and the action can never disagree.
            //
            // Deliberately not a chained @FocusState enum walked by this
            // renderer: that would make the order SwiftUI's idea of it, which
            // is derived from layout and differs from Compose's. The order is
            // declared in Go and stays there — the platform only reports that
            // the key was pressed.
            .submitLabel(imeAction == "next" ? .next : (onSubmit.isEmpty ? .return : .done))
            .onSubmit { if !onSubmit.isEmpty { runtime?.click(onSubmit) } }
            // The focus edges. This is also where the local buffer is seeded,
            // and the seeding goes first: a dispatch into Go can land a render
            // before this closure returns, and the buffer must already agree
            // with upstream when it does.
            //
            // Both edges ride the void channel, like onSubmit above.
            //
            // No blur is dispatched at mount: onChange(of:) fires on a
            // *change*, not on the initial value, so the field's starting
            // unfocused state is never reported as having lost focus. The
            // Compose side has to arrange that explicitly — see the seenFocus
            // flag in GrMobTextField — because collectIsFocusedAsState emits
            // its initial false.
            .onChange(of: focused) { _, isFocused in
                if isFocused {
                    text = node.stringProp("value")
                    pendingEchoes.removeAll()
                    if !onFocus.isEmpty { runtime?.click(onFocus) }
                } else if !onBlur.isEmpty {
                    runtime?.click(onBlur)
                }
            }
            // Go's focus *commands*, the other direction from the edges
            // above. Keyed on the epoch alone, never on the action: the
            // action is what to do, the epoch is when — and a second
            // core.Focus on the already-focused field has to re-fire, which
            // only a changed value can express.
            //
            // Two modifiers where Compose needs one. onChange(of:) does not
            // fire for the initial value — the same asymmetry that lets this
            // renderer skip Compose's seenFocus flag above — so a field that
            // mounts while it is already the target would never hear its
            // command without the onAppear. That case is the one worth
            // supporting: "push a screen and put the cursor in its search
            // box" issues the command in the handler that navigates, one pass
            // before the field it names exists.
            .onAppear { applyFocusCommand(epoch: focusEpoch, action: focusAction) }
            .onChange(of: focusEpoch) { _, epoch in
                applyFocusCommand(epoch: epoch, action: focusAction)
            }
            .onChange(of: upstream) { _, newValue in
                guard focused else { return }
                if let echo = pendingEchoes.firstIndex(of: newValue) {
                    pendingEchoes.removeSubrange(...echo)
                } else {
                    text = newValue
                    pendingEchoes.removeAll()
                }
            }
            .textFieldStyle(.plain)
            .grMobTextStyle(node.style)
            .grMobBox(node.style, grow: grow)
    }

    /// Runs one focus command from Go.
    ///
    /// Epoch 0 means no command has ever been issued (Go stamps nothing at
    /// all), so a screen that never touches focus passes through here on
    /// every appear and does nothing.
    ///
    /// "blur" is guarded on this field actually holding focus: @FocusState is
    /// per-field, so a dismiss reaches every field on screen and exactly one
    /// of them is the one to release. Only the target acts on "focus"; every
    /// other field is told "" and does nothing, because setting focus over
    /// there already takes it from here — having both sides act would make
    /// the outcome depend on the order SwiftUI happens to run two closures in.
    private func applyFocusCommand(epoch: Int, action: String) {
        guard epoch != 0 else { return }
        switch action {
        case "focus": focused = true
        case "blur": if focused { focused = false }
        default: break
        }
    }

    @ViewBuilder
    private func field(value: Binding<String>, prompt: Text) -> some View {
        if password {
            SecureField(text: value, prompt: prompt) { EmptyView() }
        } else if multiline {
            let rows = node.intProp("rows")
            TextField(text: value, prompt: prompt, axis: .vertical) { EmptyView() }
                .lineLimit(rows > 0 ? rows : 3, reservesSpace: true)
        } else {
            TextField(text: value, prompt: prompt) { EmptyView() }
                .grMobKeyboard(numeric: numeric)
        }
    }
}

extension View {
    /// Go's core.KeyboardAware, applied to the two scrolling node types.
    ///
    /// Only half of what the prop names has to be done here: SwiftUI already
    /// treats the keyboard as its own safe-area region and insets a ScrollView
    /// for it, so the shrink that Compose needs Modifier.imePadding() for is
    /// the platform default and applies whether the flag is set or not. What
    /// the flag buys on iOS is the dismissal — dragging the region puts the
    /// keyboard away, which is the behavior every native scrolling form has
    /// and which SwiftUI does not turn on by itself.
    ///
    /// `.interactively` rather than `.immediately`: the keyboard follows the
    /// drag and comes back if the finger returns, so a scroll that was only
    /// meant to peek at the field above does not cost the user their keyboard.
    @ViewBuilder fileprivate func grMobKeyboardAware(_ on: Bool) -> some View {
        if on {
            scrollDismissesKeyboard(.interactively)
        } else {
            self
        }
    }

    /// The Modal sheet's background, when the app has painted its own
    /// surface (see GrMobModal); unchanged otherwise.
    @ViewBuilder fileprivate func grMobPresentationSurface(_ color: Color?) -> some View {
        if let color {
            presentationBackground(color)
        } else {
            self
        }
    }

    @ViewBuilder fileprivate func grMobKeyboard(numeric: Bool) -> some View {
        #if os(iOS)
        keyboardType(numeric ? .decimalPad : .default)
        #else
        self
        #endif
    }
}

// ---------------------------------------------------------------------------
// Composite components
// ---------------------------------------------------------------------------

/// A top tab bar + the selected child. SwiftUI's native TabView is a bottom
/// bar whose selection wants to be locally owned; GrMob's selection is Go
/// state (a controlled int prop), so a hand-rolled top bar — matching the
/// Android renderer's material TabRow — is both simpler and correct.
private struct GrMobTabView: View {
    let node: GrMobNode
    let grow: GrMobGrow
    @Environment(\.grMobRuntime) private var runtime

    var body: some View {
        let selected = node.intProp("selectedIndex")
        let onTabChange = node.stringProp("onTabChange")
        let tabs = node.props["tabs"] as? [[String: Any]] ?? []

        VStack(spacing: 0) {
            HStack(spacing: 0) {
                ForEach(tabs.indices, id: \.self) { i in
                    Button {
                        if !onTabChange.isEmpty { runtime?.intChanged(onTabChange, i) }
                    } label: {
                        VStack(spacing: 8) {
                            Text(tabs[i]["label"] as? String ?? "")
                                .font(.subheadline.weight(i == selected ? .semibold : .regular))
                                .foregroundStyle(i == selected ? Color.accentColor : Color.secondary)
                            Rectangle()
                                .fill(i == selected ? Color.accentColor : Color.clear)
                                .frame(height: 2)
                        }
                        .padding(.top, 12)
                        .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .frame(maxWidth: .infinity)
                }
            }
            // Go renders every tab's content as a child; selection is
            // presentation state, so only the selected child gets a view.
            // .id(selected) gives each tab its own view identity, so per-tab
            // state (input text, scroll) is dropped on switch — matching the
            // replace semantics the Go reconciler would apply anyway.
            if node.children.indices.contains(selected) {
                RenderNode(node: node.children[selected])
                    .id(selected)
            }
        }
        .grMobBox(node.style, grow: grow)
    }
}

/// Modal rides a sheet — the iOS idiom for what Android renders as a centered
/// Dialog. Visibility is a controlled bool: presenting and dismissing both go
/// through Go (the binding's set only reports the dismiss gesture upstream;
/// the sheet actually closes when Go flips `visible` and the patch lands).
private struct GrMobModal: View {
    let node: GrMobNode
    @Environment(\.grMobRuntime) private var runtime

    var body: some View {
        let onDismiss = node.stringProp("onDismiss")
        Color.clear
            .frame(width: 0, height: 0)
            .sheet(isPresented: Binding(
                get: { node.boolProp("visible") },
                set: { shown in
                    if !shown, !onDismiss.isEmpty { runtime?.click(onDismiss) }
                }
            )) {
                // The sheet's own surface is the system's — light in light
                // mode — and the DOM targets draw a Modal as the backdrop
                // and nothing else, so an app that gave its dialog column a
                // dark background has already drawn the card and the
                // system surface showed around it as a frame. When a direct
                // child paints a background, the sheet takes that colour as
                // its own so the app's surface is the dialog's edge on every
                // target; unstyled content keeps the system surface.
                let surface = node.children.compactMap { $0.style?.background }.first
                VStack(alignment: .leading, spacing: 0) {
                    PlainChildren(node: node)
                }
                .padding(16)
                .presentationDetents([.medium, .large])
                .grMobPresentationSurface(surface)
            }
    }
}
