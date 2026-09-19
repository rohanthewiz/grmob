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
    /// The nearest GrMobScroll's ScrollViewReader proxy, for core.ScrollIntoView
    /// (see GrMobBringIntoView). nil outside every Scroll.
    var grMobScrollProxy: ScrollViewProxy? {
        get { self[GrMobIntoViewProxyKey.self] }
        set { self[GrMobIntoViewProxyKey.self] = newValue }
    }
}

private struct GrMobIntoViewProxyKey: EnvironmentKey {
    static let defaultValue: ScrollViewProxy? = nil
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
                // Root identity, the same rule ForEach applies to children
                // (viewID). The root sits outside any ForEach, so without this
                // SwiftUI keys it structurally and a whole-tree "replace" keeps
                // the old tree's view state — a ScrollView's offset among it.
                // core.Navigator keys each route's root by its stack frame, so
                // navigation resets it; an unkeyed root uses object identity,
                // which also changes only on a replace.
                .id(root.viewID)
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
        // Unconditional, so a node the command stamps keeps its view identity
        // (a conditional modifier would be two view types, and switching
        // between them rebuilds the subtree, a text field's focus with it).
        // An unstamped node pays one Int comparison. See GrMobBringIntoView.
        content.modifier(GrMobBringIntoView(epoch: node.intProp("scrollEpoch"), id: node.viewID))
    }

    @ViewBuilder private var content: some View {
        let style = node.style
        if style?.display == "none" {
            // Not rendered at all; "hidden" keeps space via opacity(0) in grMobBox.
        } else {
            switch node.type {
            case "Text": GrMobText(node: node, grow: grow)
            // core.Paragraph: runs of one flow of text; see GrMobParagraph.
            case "Paragraph": GrMobParagraph(node: node, grow: grow)
            case "Button": GrMobButton(node: node, grow: grow)

            case "Input": GrMobTextField(node: node, grow: grow)
            case "InputPassword": GrMobTextField(node: node, grow: grow, password: true)
            case "NumericInput": GrMobTextField(node: node, grow: grow, numeric: true)
            case "TextArea": GrMobTextField(node: node, grow: grow, multiline: true)
            case "Select": GrMobSelect(node: node, grow: grow)
            case "Checkbox": GrMobCheckbox(node: node, grow: grow)
            case "Switch": GrMobSwitch(node: node, grow: grow)
            case "Slider": GrMobSlider(node: node, grow: grow)
            case "TextGrid": GrMobTextGrid(node: node, grow: grow)
            // The programmer's editor. Its rows are a grid's rows and it draws
            // them itself, over a UITextView that owns the buffer — see
            // GrMobCodeEditor.swift, and core/codeeditor.go for why neither
            // SwiftUI nor a Go-side overlay can express this.
            case "CodeEditor": GrMobCodeEditor(node: node, grow: grow)
            // The prose editor. Its value is a richtext.Doc, mapped to and from
            // an NSAttributedString — see GrMobRichText.swift, and
            // core/richtext.go for why the document is Go's on every target.
            case "RichTextEditor": GrMobRichTextEditor(node: node, grow: grow)
            // A row reached on its own (never from core.TextGrid, which draws
            // its rows itself) still renders as a line of runs.
            case "GridRow": GrMobGridRow(node: node, base: nil)

            case "Row": GrMobRow(node: node, grow: grow)
            // Card = Column whose Go theme style carries the card look. Box =
            // Column with no theme base at all: core.Box is documented as one
            // of the flex-style containers ("Row, Column, Card, Box, List
            // share one argument contract"), differing from Column only in
            // carrying no theme style, and both DOM targets stack its
            // children — the WASM runtime lists it in STACK_CONTAINERS. This
            // was a ZStack: two children drew on top of each other on device
            // and side by side down the page in the browser. GrMobColumn
            // attaches the tap and long-press gestures itself.
            case "Column", "Card", "Box": GrMobColumn(node: node, grow: grow)
            // The one container in the vocabulary that *is* an overlay, which
            // is what the arm above stopped being. A SwiftUI ZStack is the
            // construct core.ZStack was named for; mobile/verify's
            // TestNativeZStackOverlaysItsChildren pins the arm to it.
            case "ZStack": GrMobZStack(node: node, grow: grow)
            case "List": GrMobList(node: node, grow: grow)
            case "Spacer": GrMobSpacer(node: node, grow: grow)
            case "Scroll": GrMobScroll(node: node, grow: grow)
            case "SafeArea":
                // SwiftUI already lays out inside the safe area by default, so
                // this is a grouping box — the node exists so Go apps can be
                // explicit about it and so an ignoresSafeArea escape hatch has
                // a home later. One thing does reach past the inset: the
                // node's background, painted edge to edge so a screen that
                // colours its safe area (comps.Screen forwards its
                // background here) has no light strip under the status bar.
                // The box's own background is painted by grMobBox as well,
                // inside the inset; the two are the same colour, so the
                // second coat is invisible and keeps the box's clip/border
                // layering exactly as every other node has it.
                //
                // A vertical stack, not a ZStack, for the reason the Box arm
                // above gives: a SafeArea is a flex column on both DOM
                // targets (the WASM runtime lists it in STACK_CONTAINERS), so
                // two children stacked in a browser and drew on top of each
                // other here. The single-child case moved too, and in the
                // same direction: a ZStack sizes to its children, so a
                // screen's content column hugged its widest child instead of
                // filling the width `align-items: stretch` gives it on the
                // web. GrMobColumn's flex stack supplies that, and honors a
                // FlexGrow on the content column while it is there.
                //
                // The background still rides outside GrMobColumn's own box,
                // where ignoresSafeArea can carry it under the bars.
                GrMobColumn(node: node, grow: grow)
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

            // The live map (core.MapView). Its Marker children are data rather
            // than views — GrMobMapView reads them off node.children itself and
            // turns them into MKAnnotations, the way GrMobTextGrid reads its
            // rows — so they are deliberately NOT rendered here. See
            // GrMobMapView.swift.
            case "MapView": GrMobMapView(node: node, grow: grow)

            // A Marker reached on its own, outside a map: nothing. It is data
            // for the node above it, and drawing a box for one would put an
            // empty rectangle in a layout. The arm exists so the dispatch says
            // so rather than falling through to the container default, which
            // would.
            case "Marker": EmptyView()

            // A vector drawing (core.Canvas). Its CanvasShape children are data
            // read inside the Canvas closure, like a map's markers — see
            // GrMobCanvas below and GrMobCanvasGeometry.swift.
            case "Canvas": GrMobCanvas(node: node, grow: grow)
            // A shape reached on its own, outside a canvas: nothing, for the
            // reason a lone Marker is nothing.
            case "CanvasShape": EmptyView()

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

/// core.ScrollIntoView (core/scroll_to.go), on the node carrying `epoch`: the
/// nearest GrMobScroll scrolls to it, once, the first time this app meets an
/// epoch that high.
///
/// # Finding the node
///
/// ScrollViewProxy.scrollTo looks a view up by identity, and every GrMob child
/// is built inside `ForEach(…, id: \.viewID)`, so the node's viewID already is
/// that identity. No `.id()` is added, which would give every node a second,
/// explicit identity for the sake of the one a command might name. A node
/// that is not a ForEach child (the root) cannot be found, and the command
/// scrolls nothing, as core says of a name no node carries.
///
/// # Once
///
/// The node keeps its stamp, so the mark is the runtime's, app-wide
/// (GrMobRuntime.scrollEpochApplied), for the reason core gives: a node built
/// again later must not pull its scroll view back to it.
///
/// `scrollTo` with no anchor is SwiftUI's "the least scrolling that shows it",
/// the meaning core gives the command on every host. Deferred one turn of the
/// main loop, because a command can name a node built in this very update,
/// which the scroll view has not laid out yet.
private struct GrMobBringIntoView: ViewModifier {
    let epoch: Int
    let id: AnyHashable
    @Environment(\.grMobScrollProxy) private var proxy
    @Environment(\.grMobRuntime) private var runtime

    func body(content: Content) -> some View {
        content.onChange(of: epoch, initial: true) { _, epoch in
            guard epoch > 0, let runtime, let proxy, epoch > runtime.scrollEpochApplied else { return }
            runtime.scrollEpochApplied = epoch
            DispatchQueue.main.async {
                withAnimation { proxy.scrollTo(id) }
            }
        }
    }
}

extension GrMobNode {
    /// ForEach identity: explicit key when present, object identity otherwise
    /// (see the header comment on why object identity is the right default).
    var viewID: AnyHashable {
        key.isEmpty ? AnyHashable(ObjectIdentifier(self)) : AnyHashable(key)
    }

    /// viewID as a String, for GrMobList's scrollPosition(id:), which matches
    /// ids by the binding's type. Bound as AnyHashable, it reported SwiftUI's
    /// own UniqueIDs from inside the rows (measured on the iOS 26.5
    /// simulator), which never equal a key; bound as String, only this id can
    /// match. The key when there is one; otherwise object identity, spelled
    /// so it cannot collide with a key (keys never start with "#object:").
    var rowKey: String {
        key.isEmpty ? "#object:\(ObjectIdentifier(self).hashValue)" : key
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
        let s = node.containerStyle
        // core.FlexWrap(true) asks for CSS flex-wrap: children that do not fit
        // continue on the next line instead of being shrunk onto one. The
        // flex stack cannot do that — it is a single-line algorithm that
        // shrinks proportionally, so a row of chips wider than the screen
        // squeezed every chip's label until it broke mid-word. The wrap
        // layout keeps each child at its ideal size and breaks lines instead;
        // the gap spaces both axes — chips along a line by horizontalGap
        // (ColumnGap, else Gap) and the lines apart by verticalGap (RowGap,
        // else Gap), as CSS gap does on a wrapping flex container and as the
        // Android FlowRow does with its two arrangements.
        if s?.flexWrap == "wrap" {
            GrMobWrapLayout(spacing: s?.horizontalGap ?? 0,
                            lineSpacing: s?.verticalGap ?? 0,
                            crossAlign: s?.alignItems ?? "") {
                // The wrap layout has no solver to resolve a percentage cap,
                // so each child keeps resolving its own.
                FlexChildren(node: node, axis: .horizontal, resolvesPercentCaps: false)
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
    /// Between two children on the same line. This is the only spacing the
    /// solver sees, because it is the only one line breaking depends on.
    let spacing: CGFloat
    /// Between two lines. Never reaches the solver — it changes the height
    /// the lines occupy, not where they break.
    let lineSpacing: CGFloat
    /// The Row's AlignItems: where a child shorter than its line sits in it,
    /// as CSS align-items places items within a wrapped flex line. Top used
    /// to be the only answer, on the grounds that wrapped chip rows are one
    /// height. comps.Breadcrumb is not: its ancestor crumbs are buttons with
    /// a touch-target height and its current page is a bare Text, which sat
    /// 10pt above them (the simulator, lesson 4.27). The same gap Compose's
    /// FlowRow had; see GrMobRow in Renderer.kt.
    var crossAlign: String = ""

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
        let lineGaps: CGFloat = lineSpacing * CGFloat(max(lines.count - 1, 0))
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
            let height = lineHeight(line, sizes)
            for i in line {
                // The non-wrapping row's own cross-axis rule, applied per line.
                // "stretch" places at the top here, as it always has: nothing
                // proposes a wrapped child its line's height.
                let dy = GrMobFlexSolver.crossOffset(align: crossAlign, child: sizes[i].height, extent: height)
                subviews[i].place(at: CGPoint(x: x, y: y + dy), anchor: .topLeading,
                                  proposal: ProposedViewSize(sizes[i]))
                x += sizes[i].width + spacing
            }
            y += height + lineSpacing
        }
    }
}

private struct GrMobColumn: View {
    let node: GrMobNode
    let grow: GrMobGrow

    var body: some View {
        let s = node.containerStyle
        GrMobFlexStack(axis: .vertical, style: s) {
            FlexChildren(node: node, axis: .vertical)
        }
        .grMobBox(s, grow: grow,
                    onTap: node.stringProp("onClick"),
                    onLongPress: node.stringProp("onLongPress"),
                    axis: .vertical)
    }
}

/// core.ZStack: every child drawn in the same box, in tree order, so the last
/// one written is on top.
///
/// `.center` is stated rather than left to the default even though SwiftUI's
/// ZStack already centres. The alignment is a cross-target contract — a
/// Compose Box defaults to TopStart and has to be told, and the DOM targets
/// centre a single grid cell — so the value is written on all three, where a
/// reader can compare them. An inherited default is invisible from the other
/// two renderers.
///
/// Not FlexChildren: an overlay divides no leftover space along an axis, so
/// there is no flex weight to hand a layer and no cross-axis stretch to apply.
/// A layer that wants the stack's full extent states its own dimensions, which
/// is the contract core.ZStack documents.
///
/// # The per-layer opt-out, and why it stopped being a frame
///
/// core.Style.StackAlign lets one layer sit in a corner instead. SwiftUI is
/// the target with no direct spelling for it — a ZStack's `alignment:` is the
/// stack's, not the layer's, and there is no `.align()` for a child the way
/// Compose's BoxScope has one — so the layer used to be wrapped in a frame
/// that filled the stack and placed inside it. That is SwiftUI's own idiom for
/// the job, and it carried this target's one divergence with it: **a filling
/// frame is greedy**. An *unsized* stack grew to whatever its parent proposed
/// as soon as one layer was aligned, where a Compose Box and a CSS grid track
/// both stay the size of their largest child.
///
/// The frame is gone. GrMobStackLayout below places each layer by coordinate,
/// so nothing is wrapped and nothing is greedy, and the container reports the
/// largest child on each axis — which is core.ZStack's stated contract and what
/// the other three targets already did. An unsized stack with an aligned layer
/// now renders the same on all four.
///
/// The Layout replaces the ZStack for *every* stack rather than only for ones
/// with a placed layer, and that is deliberate: two paths that have to agree
/// about sizing is a worse trade than one path. What keeps the change honest is
/// that the arithmetic is separately testable — GrMobStackSolver is pure and
/// `ios/verify` measures it, which a `ZStack { }` never could be.
private struct GrMobZStack: View {
    let node: GrMobNode
    let grow: GrMobGrow

    var body: some View {
        let s = node.containerStyle
        GrMobStackLayout {
            ForEach(node.children, id: \.viewID) { child in
                RenderNode(node: child)
                    .layoutValue(key: GrMobStackPlacement.self,
                                 value: grMobStackAnchor(child.style?.stackAlign ?? ""))
            }
        }
        .grMobBox(s, grow: grow,
                    onTap: node.stringProp("onClick"),
                    onLongPress: node.stringProp("onLongPress"))
    }
}

/// One layer's placement, handed from GrMobZStack to GrMobStackLayout.
///
/// A LayoutValueKey rather than an array on the layout, for the reason
/// GrMobFlexWeight gives: a Layout receives its children as opaque proxies,
/// `subviews[i]` cannot be traced back to the GrMobNode it came from, and a
/// parallel array would silently mis-align the moment SwiftUI flattened a
/// Group or dropped an empty view. A stack is exactly where that would show —
/// a core.For inside one generates its layers.
///
/// nil is the centre, which is core.StackAlignCenter and every layer's default.
/// Carried as an Optional rather than defaulting to `.center` here so that
/// "said nothing" and "asked for the centre" stay one state all the way down;
/// grMobStackAnchor returns nil for both.
private struct GrMobStackPlacement: LayoutValueKey {
    static let defaultValue: GrMobStackAnchor? = nil
}

/// One SwiftUI subview, as GrMobStack.swift's arithmetic sees it.
///
/// The whole of the adapter, and the whole of what SwiftUI contributes to the
/// two questions the solver asks: `sizeThatFits` behind `size(proposing:)`,
/// and the LayoutValueKey behind `anchor`. Everything either one is used *for*
/// is decided next door, where it can be run.
///
/// The size *vocabulary* moved next door too, and later: `proposedViewSize`
/// and `GrMobProposal.init(_:)` are in GrMobStackBridge.swift, which
/// `ios/verify` compiles and runs. They were three field-copying expressions
/// written out here, which is code a reader checks by eye and a compiler
/// agrees with whichever way round it is written — see that file.
///
/// A struct wrapping the proxy rather than an extension on `LayoutSubview`
/// itself: the conformance would then be visible to anything in the module
/// that happens to hold one, and this is a private detail of one layout.
private struct GrMobStackSubview: GrMobStackLayer {
    let subview: LayoutSubview

    func size(proposing proposal: GrMobProposal) -> CGSize {
        subview.sizeThatFits(proposal.proposedViewSize)
    }

    var anchor: GrMobStackAnchor? { subview[GrMobStackPlacement.self] }
}

/// The overlay's layout: measure every layer, size to the largest, place each
/// one at its own anchor.
///
/// Both methods are two lines because both are the same shape — convert the
/// subviews, ask GrMobStackSolver, do the one thing only SwiftUI can do. Which
/// proposal each layer is measured with, whether the container may be clamped
/// to it, and what a layer is offered at placement are all decided in
/// GrMobStack.swift, and `ios/verify` runs them there against a recording
/// fake. The conversion between the two size vocabularies is run there too,
/// out of GrMobStackBridge.swift. What is left here is `subviews.map`, one
/// `sizeThatFits` and one `place()`.
///
/// What still cannot be checked off-device is the assumption underneath: that
/// a real `LayoutSubview` answers `sizeThatFits` the way the fake does. That
/// is SwiftUI's behaviour rather than this framework's, and a simulator is the
/// only thing that can say.
private struct GrMobStackLayout: Layout {
    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        GrMobStackSolver.containerSize(
            layers: subviews.map(GrMobStackSubview.init),
            proposing: GrMobProposal(proposal))
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        // `bounds`, not `proposal`: the parent is free to hand over a size it
        // never asked sizeThatFits about. The plan carries the offer each
        // layer was measured with so the same one is proposed at placement —
        // see GrMobStackSolver.placements.
        let plan = GrMobStackSolver.placements(
            layers: subviews.map(GrMobStackSubview.init), in: bounds)
        for (subview, placement) in zip(subviews, plan) {
            subview.place(
                at: placement.origin,
                anchor: .topLeading,
                proposal: placement.proposal.proposedViewSize)
        }
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
    /// Whether the layout these children are handed to is GrMobFlexLayout,
    /// which resolves a Row child's percentage MaxWidth itself (see
    /// GrMobFlexSolver.percentCaps). False for GrMobWrapLayout.
    var resolvesPercentCaps: Bool = true

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
            // A percentage floor along this stack's axis, resolved by the
            // layout against its own extent (GrMobFlexSolver.percentFloors).
            let floor = GrMobMinSize.fraction(axis == .horizontal ? (child.style?.minWidth ?? "")
                                                                  : (child.style?.minHeight ?? "")) ?? 0
            // A percentage cap along a Row, resolved the same way; a Column's
            // main axis is height, and MaxWidth is not read there.
            let cap = axis == .horizontal && resolvesPercentCaps
                ? GrMobMinSize.fraction(child.style?.maxWidth ?? "") ?? 0 : 0
            RenderNode(node: child, grow: fill(weight: weight, floored: floor > 0, stretch: stretch && !hugs))
                // Tells the child's GrMobMaxWidthModifier to stand down; set
                // only where the layout below will apply the cap instead.
                .environment(\.grMobPercentCapResolved, cap > 0)
                .layoutValue(key: GrMobFlexWeight.self, value: weight)
                .layoutValue(key: GrMobFlexPercentFloor.self, value: floor)
                .layoutValue(key: GrMobFlexPercentCap.self, value: cap)
                .layoutValue(key: GrMobFlexCapMargin.self, value: grMobHorizontalMargin(child.style))
                // The reading, not the raw field: core.FlexShrink(0) arrives as
                // core.ShrinkNone and an absent declaration as 0, and
                // shrinkFactor is the one place that knows which is which.
                .layoutValue(key: GrMobFlexShrink.self,
                             value: child.style?.shrinkFactor ?? 1)
                .layoutValue(key: GrMobFlexHugs.self, value: hugs)
                // A zero flex-basis, carried as the child's padding along
                // this axis (its whole CSS base size), or -1 for the default
                // content-sized basis. See GrMobFlexZeroBasis.
                .layoutValue(key: GrMobFlexZeroBasis.self, value: zeroBasisPadding(child.style))
                // The CSS `min-width: auto` floor, measured off the node
                // because no view on this host will report it.
                //
                // A Column's `min-height: auto` floor cannot be measured off
                // the node — a text's min-content HEIGHT is a function of the
                // width it wraps at, which a tree walk does not know — but the
                // layout already measures exactly that number: it is the
                // child's base size. So a Column hands over a verdict instead
                // of a number. `.infinity` means "floor at your base", which
                // minMains clamps down to the base; 0 means no floor. See
                // GrMobMinContent.floorsHeightAtContent for which children
                // get which, and GrMobMinContent for every case either axis
                // deliberately floors at zero.
                .layoutValue(key: GrMobFlexMin.self,
                             value: axis == .horizontal
                                 ? GrMobMinContent.width(of: child)
                                 : (GrMobMinContent.floorsHeightAtContent(child) ? .infinity : 0))
        }
    }

    /// The GrMobFlexZeroBasis value for a child: its padding along this
    /// stack's axis when it declares a zero flex-basis, -1 otherwise. A
    /// function rather than inline, where the conditional arithmetic took the
    /// type-checker past its time limit.
    private func zeroBasisPadding(_ style: GrMobStyle?) -> CGFloat {
        guard let style, style.zeroBasis else { return -1 }
        let edges = axis == .horizontal ? style.padding.left + style.padding.right
                                        : style.padding.top + style.padding.bottom
        return CGFloat(edges)
    }

    /// `floored` is a percentage floor on the main axis: like a grower, such a
    /// child may be given a slot longer than its content, and a main-axis
    /// fill is what makes it take the slot rather than draw its content
    /// inside it.
    private func fill(weight: CGFloat, floored: Bool, stretch: Bool) -> GrMobGrow {
        var g = GrMobGrow()
        if weight > 0 || floored {
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

/// A child's percentage floor along the container's main axis, as a fraction
/// (0.4 for "40%"), 0 for none. Carried the same way as the weight; see
/// GrMobFlexSolver.percentFloors for why the container, not the child,
/// resolves it.
private struct GrMobFlexPercentFloor: LayoutValueKey {
    static let defaultValue: CGFloat = 0
}

/// A Row child's percentage MaxWidth as a fraction (0.8 for "80%"), 0 for
/// none, and the horizontal margin its cap stands outside of. See
/// GrMobFlexSolver.percentCaps for why the container resolves it.
private struct GrMobFlexPercentCap: LayoutValueKey {
    static let defaultValue: CGFloat = 0
}

private struct GrMobFlexCapMargin: LayoutValueKey {
    static let defaultValue: CGFloat = 0
}

/// Per-child flex-shrink factor, carried the same way and for the same reason.
///
/// The default is 1, not 0: this is the one flex property whose CSS initial
/// value is not zero, and a subview SwiftUI hands the layout without one of
/// these — there should be none, but a default is a default — must shrink like
/// every child always did rather than refuse to.
private struct GrMobFlexShrink: LayoutValueKey {
    static let defaultValue: CGFloat = 1
}

/// Whether a child of a stretched Column keeps its own width (see
/// hugsContent). Carried the same way as the weight, for the same reason:
/// the layout sees subviews, not nodes, and this is a per-child decision it
/// has to make when it proposes the cross size.
private struct GrMobFlexHugs: LayoutValueKey {
    static let defaultValue = false
}

/// This child's automatic minimum size along the container's main axis — CSS
/// `min-width: auto`, in points, carried the same way as the weight.
///
/// A number rather than a flag, because the layout cannot obtain it: SwiftUI's
/// documented way to ask a subview for its minimum is a zero proposal, and a
/// `Text` answers a zero proposal with zero (see GrMobMinContent for the
/// measurement that established it). So the value is computed from the NODE,
/// on the one side of the wall that still knows what the string is, and handed
/// across with the weight and the shrink factor.
///
/// The default is 0 — no floor, and the behaviour every Row here had before
/// this existed. That is the opposite choice from GrMobFlexShrink's CSS-initial
/// default, and deliberately: an unset shrink factor has one right answer,
/// while an unset floor would be a guess at a measurement, and a guessed floor
/// that is too high overflows a line a browser would have fitted.
///
/// Along a Column's main axis the value is a verdict rather than a
/// measurement: `.infinity` for a child whose automatic minimum height is its
/// content height, 0 for a child with no floor. The content height is the base
/// size the layout measures anyway, and minMains clamps every value to the
/// base, so infinity lands on exactly that number with no second measurement.
///
/// ```
///   axis        value sent              floor the solver sees
///   ----        ----------              ---------------------
///   horizontal  min-content width       min(value, base)
///   vertical    .infinity | 0           base | 0
/// ```
private struct GrMobFlexMin: LayoutValueKey {
    static let defaultValue: CGFloat = 0
}

/// A child's core.FlexBasis("0"), as its padding along the main axis, or -1
/// for the default basis (its content size).
///
/// # Why a zero basis has to be read
///
/// This host modelled every item as `flex-basis: auto`, and for most rows that
/// is the same picture. It is not for the rows core's widgets divide by
/// weight — a chart's label slots, its bar-value segments, a StatTile's
/// columns — which write FlexGrow(w) with FlexBasis("0") precisely so the
/// line is shared in proportion to the weights whatever each box holds:
///
///                      base          size = base + free · w/Σw
///   basis auto (was)   text width    off by (text − mean text) · …
///   basis 0    (CSS)   padding       exactly w/Σw of the line, less padding
///
/// With auto bases a label's box grew by its own text's width, so a bar's
/// value written just past its tip sat a few points inside the bar or clear
/// of it, by an amount that changed with the digits. Chrome and Compose
/// (whose weight() ignores content) divide exactly.
///
/// CSS still clamps the item by its automatic minimum (`min-width: auto`), so
/// a zero-basis item with unbreakable content keeps that content's width.
/// The clamp comes AFTER the free space is shared, not in the base:
/// GrMobFlexLayout.minMains passes the GrMobFlexMin floor to the solver
/// unclamped for such a child, and GrMobFlexSolver.grow raises and freezes
/// only a child whose share falls short of it. (A first version folded the
/// floor into the base, which is the content width again; lesson 4.9's
/// calendar showed it, with "10" and "31" in wider columns than "8".) A Column's verdict (.infinity, "floor at your
/// content") cannot be clamped against without a measurement, so a
/// zero-basis Column child with that verdict keeps its measured base.
private struct GrMobFlexZeroBasis: LayoutValueKey {
    static let defaultValue: CGFloat = -1
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
            // The spacing along this stack's own axis: a vertical stack is
            // spaced by RowGap (the gap *between rows*), a horizontal one by
            // ColumnGap, each falling back to the isotropic Gap. See
            // GrMobStyle.verticalGap.
            spacing: axis == .vertical ? (style?.verticalGap ?? 0)
                                       : (style?.horizontalGap ?? 0),
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
        let offered = mainOf(proposal)
        let floors = percentFloors(subviews, extent: offered)
        let caps = percentCaps(subviews, extent: offered)
        let bases = GrMobFlexSolver.capped(
            baseMains(subviews, crossBound: crossBound, floors: floors,
                      definite: GrMobFlexSolver.definite(offered) != nil),
            by: caps)
        let weights = subviews.map { $0[GrMobFlexWeight.self] }
        let main = solver.containerMain(offered: offered, bases: bases, weights: weights,
                                        percentCapped: caps.contains { $0 != nil })
        // The container's own size is unchanged by the floor, and that is the
        // CSS shape: a flex container that cannot fit its children OVERFLOWS
        // them — it does not report itself bigger and take the room from its
        // parent. Nesting still composes, because GrMobMinContent sums a
        // nested Row's children itself, so the outer Row is told what the
        // inner one cannot give up before it decides anything.
        let resolved = solver.resolve(
            main: main, bases: bases, weights: weights,
            shrinks: subviews.map { $0[GrMobFlexShrink.self] },
            mins: minMains(subviews, bases: bases, floors: floors))

        // Cross size is re-measured at each child's *final* main size: a Text
        // that had to shrink wraps to more lines, and asking it before the
        // main axis was settled would under-report its height.
        let cross = zip(subviews, GrMobFlexSolver.capped(resolved.mains, by: caps))
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
        // The percentage floors against the same extent sizeThatFits used, the
        // offer, so a hugging container is not re-floored against its own
        // hugged size; the bounds stand in only when no offer was made.
        let extent = GrMobFlexSolver.definite(mainOf(proposal)) ?? mainOf(bounds.size)
        let floors = percentFloors(subviews, extent: extent)
        // Caps against the same extent as the floors, for the same reason.
        let caps = percentCaps(subviews, extent: extent)
        let bases = GrMobFlexSolver.capped(
            baseMains(subviews, crossBound: containerCross, floors: floors, definite: true),
            by: caps)
        let weights = subviews.map { $0[GrMobFlexWeight.self] }
        let resolved = solver.resolve(
            main: mainOf(bounds.size), bases: bases, weights: weights,
            shrinks: subviews.map { $0[GrMobFlexShrink.self] },
            mins: minMains(subviews, bases: bases, floors: floors))
        // The same read FlexChildren makes, and it has to be the same one:
        // an unset value stretches on the vertical axis (the CSS default the
        // DOM targets have always drawn) and packs on the horizontal one.
        let stretch = axis == .vertical ? columnStretches(crossAlign) : crossAlign == "stretch"

        let mains = GrMobFlexSolver.capped(resolved.mains, by: caps)
        var offset = resolved.leading
        for (i, subview) in subviews.enumerated() {
            let childMain = mains[i]
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
    ///
    /// A child's percentage floor (see percentFloors) raises its base.
    ///
    /// A zero flex-basis child (GrMobFlexZeroBasis) starts from its padding
    /// instead of its content — but only when the container's main extent is
    /// `definite`. Its automatic minimum and percentage floor are NOT folded
    /// into the base: minMains hands them to the solver as a minimum, which
    /// applies them after the free space is shared (GrMobFlexSolver.grow).
    /// Folding them in made the base the content width again, and the day
    /// columns of a calendar came out wider for "10" than for "8". Asked for an ideal size, a
    /// row of zero-basis boxes would otherwise report the sum of their
    /// paddings and be laid out at nearly nothing; CSS sizes such a container
    /// from its items' content contributions, which is the measured base.
    /// placeSubviews always has definite bounds, and sharing those bounds out
    /// by weight from zero bases fills them exactly as the measured ones did.
    private func baseMains(_ subviews: Subviews, crossBound: CGFloat?, floors: [CGFloat],
                           definite: Bool) -> [CGFloat] {
        subviews.enumerated().map { i, subview in
            let padding = subview[GrMobFlexZeroBasis.self]
            let automatic = subview[GrMobFlexMin.self]
            if definite, padding >= 0, automatic.isFinite {
                return padding
            }
            return max(mainOf(subview.sizeThatFits(proposed(main: nil, cross: crossBound))), floors[i])
        }
    }

    /// Each child's percentage floor along the main axis in points, against
    /// `extent`; see GrMobFlexSolver.percentFloors.
    private func percentFloors(_ subviews: Subviews, extent: CGFloat?) -> [CGFloat] {
        GrMobFlexSolver.percentFloors(fractions: subviews.map { $0[GrMobFlexPercentFloor.self] },
                                      extent: extent)
    }

    /// Each child's percentage cap along the main axis in points, against
    /// `extent`; see GrMobFlexSolver.percentCaps.
    private func percentCaps(_ subviews: Subviews, extent: CGFloat?) -> [CGFloat?] {
        GrMobFlexSolver.percentCaps(fractions: subviews.map { $0[GrMobFlexPercentCap.self] },
                                    margins: subviews.map { $0[GrMobFlexCapMargin.self] },
                                    extent: extent)
    }

    /// Each child's automatic minimum size along the main axis — the floor
    /// the shrink arm may not push it below (CSS `min-width: auto`).
    ///
    /// Read off the layout value its parent computed rather than measured
    /// here: the first attempt at this probed each subview with a zero main
    /// proposal, which is SwiftUI's documented way of asking for a minimum,
    /// and a `Text` answered 0.0 — it accepts any width and wraps to fit, so
    /// there is no minimum in the view layer to read. GrMobMinContent computes
    /// it from the node instead.
    ///
    /// The clamp to `base` is belt and braces for a Row — a floor above the
    /// ideal size is not a shape GrMobMinContent.width produces — and the
    /// whole mechanism for a Column, whose children send `.infinity` to mean
    /// "floor at the base" (see GrMobFlexMin). The solver clamps again
    /// regardless.
    ///
    /// A percentage floor outranks the automatic one: CSS's min-width is the
    /// declared minimum, and it is at most the base, which it already raised.
    ///
    /// A zero-basis child is the exception to the clamp: its base is its
    /// padding, below its content, and the content floor is exactly what the
    /// solver must still honour once the free space is shared. So its min is
    /// passed whole (when finite; a Column's `.infinity` verdict keeps its
    /// measured base in baseMains and takes the clamp like any other child).
    /// When the extent is not definite baseMains measured the base, which is
    /// at or above the floor, and the clamp is a no-op either way.
    private func minMains(_ subviews: Subviews, bases: [CGFloat], floors: [CGFloat]) -> [CGFloat] {
        subviews.enumerated().map { i, subview in
            let automatic = subview[GrMobFlexMin.self]
            let zeroBasis = subview[GrMobFlexZeroBasis.self] >= 0 && automatic.isFinite
            return max(zeroBasis ? automatic : min(automatic, bases[i]), floors[i])
        }
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
/// comps.Screen{Fill, Scroll}, the ordinary form-shaped screen, stopped
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
/// core.Spacer: a fixed void that does not give way.
///
/// # Why the node has a box at all
///
/// It used to be one expression — `Color.clear.frame(width:height:)` — with no
/// `.grMobBox` on it, which made a Spacer the one node type on this target
/// whose own Style was dropped whole. Both DOM renderers had always applied it
/// (htmlout's `spacerChassis`, the WASM runtime's `applySpacerChassis`), so a
/// hand-assembled Spacer carrying a `Background` was coloured in a browser and
/// invisible on a phone, and the same divergence swallowed its accessibility
/// props, its callback IDs and its margin.
///
/// `core.Spacer(n)` builds a node with a size prop and nothing else, so on
/// every tree core produces this renders exactly what the old expression did.
/// A node assembled by hand is the case that changed, and it changed toward
/// what the other three targets already did.
///
/// # The chassis goes underneath the author's style, per axis
///
/// The size prop is the node *type's* fixed look, and everywhere else in this
/// framework a type's look yields to the author's declarations rather than
/// overriding them — `modalChassis` in htmlout says so in as many words, and
/// `applySpacerChassis` records which of its three properties the author
/// claimed for exactly this reason.
///
/// So `spacerExtent` returns nil on an axis the Style already pins, and nil is
/// SwiftUI's "do not constrain this one". That is not merely tidier than
/// letting `grMobBox`'s own frame sit outside a fixed one: `Color.clear` is a
/// *flexible* view, and leaving the claimed axis unconstrained is what keeps
/// it flexible there, so the background `grMobBox` paints inside its frame
/// fills the whole of the box the author asked for. A fixed 10×10 clear inside
/// a 200-wide frame would instead paint a 10-point square in a 200-point hole.
///
/// ```
///   Style says nothing            Style says Width(200)
///
///   +--------+                    +--------------------------+
///   | 10x10  |  frame(10, 10)     |        200 x 10          |  frame(nil, 10)
///   +--------+                    +--------------------------+  + grMobBox's
///                                                                 width frame
/// ```
///
/// # The children, and why they are an overlay rather than the content
///
/// A hand-assembled Spacer's children used to be dropped here: this was a bare
/// `Color.clear`, a leaf, where both DOM renderers emit a Spacer's children
/// like any other element's. `core.Spacer(n)` builds none, so every tree core
/// produces is unaffected; a node assembled by hand is the case that changed,
/// and it changed toward what the two DOM targets already did — the same
/// direction the Style, the accessibility props, the callback IDs and the
/// margin all moved in above.
///
/// They stack in a **vertical flex column**, not a ZStack, for the reason the
/// `"Column", "Card", "Box"` arm gives: an overlay construct draws two
/// children on top of one another while both DOM targets stack them down the
/// page. Spacer is in htmlout's `stackAxes` on "column" so the axis is one
/// stated fact both DOM renderers and both natives read, rather than a choice
/// invented per target. A Spacer with children is a Box with a fixed size.
///
/// `Color.clear` stays, as the thing that is *sized*, with the stack laid over
/// it — and that is load-bearing rather than incidental. `grMobBox` paints the
/// background **inside** its own dimension frame (see its modifier order), so
/// on an axis the Style claims, the fill covers only as much as the content
/// asks for. `Color.clear` is flexible and asks for all of it; an empty flex
/// stack is zero-sized and would ask for none, which would repaint the
/// `Width(200)` case above as a fill of nothing. Overlaying leaves the whole
/// of that argument untouched: the geometry of a childless Spacer is
/// byte-for-byte what it was, and the children are drawn in the box it
/// already had.
///
/// `.topLeading`, because a fixed box's children start at its leading edge in
/// normal flow on both DOM targets and at TopStart in Compose. It only shows
/// when the stack does not fill the box — a stack proposed the base's size
/// takes it, so this is the degenerate case rather than the usual one.
///
/// The residual, named because it is the one thing that does not agree: a
/// child taller than the void overflows here and on both DOM targets, and is
/// measured into the size by Compose's `Modifier.size`. That is not a Spacer
/// property — it is what every fixed-size container in this framework does on
/// that target — so it is stated and not special-cased.
private struct GrMobSpacer: View {
    let node: GrMobNode
    let grow: GrMobGrow

    var body: some View {
        let s = node.style
        let size = node.intProp("size")
        Color.clear
            .frame(width: spacerExtent(size, stated: s?.width ?? ""),
                   height: spacerExtent(size, stated: s?.height ?? ""))
            .overlay(alignment: .topLeading) {
                GrMobFlexStack(axis: .vertical, style: s) {
                    FlexChildren(node: node, axis: .vertical)
                }
            }
            .grMobBox(s, grow: grow,
                        onTap: node.stringProp("onClick"),
                        onLongPress: node.stringProp("onLongPress"),
                        axis: .vertical)
    }
}

/// One axis of the Spacer chassis, or nil where the Style already claims it.
///
/// "Claims it" is the same test `grMobDimension` applies: everything but the
/// empty string and "auto" produces a frame there, including a value it cannot
/// parse. That last case looks like a hole and is the agreeing answer — a
/// declaration the CSS parser rejects also counts as authored on both DOM
/// targets, because `applySpacerChassis` asks whether the style pass wrote the
/// property rather than whether the browser kept it.
private func spacerExtent(_ size: Int, stated: String) -> CGFloat? {
    guard stated.isEmpty || stated == "auto" else { return nil }
    return CGFloat(size)
}

/// Cross-axis stretch applies as in any vertical container: the Scroll is a
/// flex column on the web, so its children fill its width unless they hug.
private struct GrMobScroll: View {
    let node: GrMobNode
    let grow: GrMobGrow
    @State private var viewport: CGFloat = 0

    var body: some View {
        // core.Horizontal() turns the region on its side. It arrives as
        // Style.FlexDirection because that is the property both DOM targets
        // already implement it with (see core/layout.go's Horizontal), so the
        // SwiftUI half is just the sideways spelling of the same two views.
        if node.style?.flexDirection == "row" {
            horizontal
        } else {
            vertical
        }
    }

    /// The sideways strip: chips, tabs, a card carousel.
    ///
    /// No viewport preference and no minHeight, unlike the vertical body
    /// below: that measurement exists to give a FlexGrow child a floor on the
    /// axis the scroll made unbounded, and here the unbounded axis is the
    /// horizontal one — a strip's height is whatever its parent proposes, and
    /// growing sideways inside a region that already scrolls sideways has no
    /// meaning. Compose's GrMobScroll skips its BoxWithConstraints on this
    /// branch for the same reason.
    ///
    /// Cross-axis alignment is .top rather than the vertical body's .leading
    /// stretch: a Row's cross axis is vertical, and neither native stretches a
    /// row's children unless AlignItems says "stretch" outright — the
    /// Style.Align fallback is a vertical-container rule (see crossAxisValue).
    ///
    /// # A strip with a grower
    ///
    /// A ScrollView proposes its content an unbounded width, so a FlexGrow
    /// child has no free space to take: lesson 4.8's footer drew its count
    /// 16pt after "Start over" on the simulator, where the web and Compose
    /// (GrMobGrowStrip) push it to the strip's far edge. CSS divides the free
    /// space whenever the content is narrower than the viewport.
    ///
    /// So a strip with a grower is a flex row (GrMobFlexStack, which divides
    /// free space and reads AlignItems) proposed at least the viewport width:
    ///
    /// ```
    ///   viewport   the ScrollView's width, read in its background
    ///   ideal      the row's width with nothing to divide (bases + gaps)
    ///   proposed   max(ideal, viewport)    free space only when it is short
    /// ```
    ///
    /// A strip with no grower keeps the plain HStack, as Compose keeps its
    /// plain Row: nothing to divide, and no row that already draws moves.
    @ViewBuilder private var horizontal: some View {
        let grows = node.children.contains { ($0.style?.flexGrow ?? 0) > 0 }
        // The reader is what core.ScrollIntoView scrolls with; its proxy
        // reaches the content through the environment (GrMobBringIntoView).
        ScrollViewReader { proxy in
            ScrollView(.horizontal, showsIndicators: false) {
                Group {
                    if grows {
                        GrMobStripContentLayout(viewport: viewport) {
                            GrMobFlexStack(axis: .horizontal, style: node.style) {
                                FlexChildren(node: node, axis: .horizontal)
                            }
                        }
                    } else {
                        HStack(alignment: .top, spacing: node.style?.horizontalGap ?? 0) {
                            ForEach(node.children, id: \.viewID) { child in
                                RenderNode(node: child, grow: .none)
                            }
                        }
                    }
                }
                .environment(\.grMobScrollProxy, proxy)
            }
        }
        // The viewport width, for the grower branch. `viewport` holds a
        // height on the vertical body and a width here; a Scroll is one or
        // the other for its whole life, since the axis is its node type's.
        //
        // onGeometryChange rather than the GeometryReader preference the
        // vertical body uses: on the simulator that preference reached this
        // modifier as 0 on every change, so the row was proposed only its
        // ideal width and the spacer took nothing. A preference also travels
        // up to every ancestor, where a strip inside a vertical Scroll would
        // hand that Scroll its width as the page's viewport height.
        .onGeometryChange(for: CGFloat.self) { proxy in
            proxy.size.width
        } action: { width in
            viewport = grows ? width : 0
        }
        .grMobKeyboardAware(node.boolProp("keyboardAware"))
        .grMobBox(node.style, grow: grow)
    }

    private var vertical: some View {
        let stretch = columnStretches(crossAxisValue(node.style))
        // The reader is what core.ScrollIntoView scrolls with; see horizontal.
        return ScrollViewReader { proxy in
            ScrollView {
                // spacing, not a hard 0: a Scroll is a flex column on both web
                // targets (the WASM runtime lists it in STACK_CONTAINERS and
                // htmlout emits gap for it), so core.Gap on a Scroll spaced its
                // children in the browser and was silently dropped here.
                VStack(alignment: .leading, spacing: node.style?.verticalGap ?? 0) {
                    ForEach(node.children, id: \.viewID) { child in
                        let grows = (child.style?.flexGrow ?? 0) > 0
                        RenderNode(node: child, grow: GrMobGrow(
                            fillWidth: stretch && !hugsContent(child.style),
                            minHeight: grows ? viewport : 0))
                    }
                }
                .environment(\.grMobScrollProxy, proxy)
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

/// The content of a horizontal Scroll that has a FlexGrow child: the row is
/// proposed `max(ideal, viewport)` wide, so its growers divide the viewport's
/// free space when the row is shorter, and it scrolls at its ideal width when
/// it is longer. See GrMobScroll.horizontal.
///
/// A Layout rather than `.frame(minWidth:)`: a flexible frame passes the
/// ScrollView's unbounded proposal straight to its child and only pads the
/// result, so the row would still be measured with no width to divide.
private struct GrMobStripContentLayout: Layout {
    let viewport: CGFloat

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        guard let row = subviews.first else { return .zero }
        let width = stripWidth(row, height: proposal.height)
        let size = row.sizeThatFits(ProposedViewSize(width: width, height: proposal.height))
        return CGSize(width: max(size.width, width), height: size.height)
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        subviews.first?.place(at: bounds.origin, anchor: .topLeading,
                              proposal: ProposedViewSize(width: bounds.width, height: bounds.height))
    }

    private func stripWidth(_ row: LayoutSubview, height: CGFloat?) -> CGFloat {
        let ideal = row.sizeThatFits(ProposedViewSize(width: nil, height: height)).width
        return max(ideal, viewport)
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
    @Environment(\.grMobDispatch) private var dispatch
    /// Row placement animates under the List's Transition, so it snaps under
    /// Reduce Motion like every other Transition (see grMobTransition).
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    /// The row at the top edge of a StartAtEnd list, kept by SwiftUI through
    /// scrollPosition(id:) below. See the note there.
    @State private var topRow: String?
    /// Whether the last row is materialized, which in a lazy stack means the
    /// reader is at or near the end. Read by stickToEnd.
    @State private var nearEnd = false

    var body: some View {
        let s = node.style
        let rows = flattenFragments(node.children)
        let startAtEnd = node.boolProp("startAtEnd")
        let startReached = node.stringProp("onStartReached")
        // Whether the top-edge row is tracked at all: a thread, or any List
        // reporting its top edge (see topEdgeReached).
        let tracksTop = startAtEnd || !startReached.isEmpty
        // Read here rather than inside rowView so the equality stays in the
        // declaration mobile/verify's TestListStretchFillReadsTheAlignFallback
        // anchors on — the pin exists because this read and crossAlignmentH's
        // once came apart, and a helper it cannot see would hide the next
        // drift exactly as effectively.
        let stretch = columnStretches(crossAxisValue(s))
        // A List with no Height inside a scrolled page needs no special arm
        // here, unlike Compose's GrMobList (LazyColumn throws under an infinite
        // height). The page's ScrollView proposes this one no height, and a
        // ScrollView answers that with its content's height. Measured on a
        // simulator: the tutorial's 4.3 outline came out 228pt with all six rows
        // inside the frame, 4.6's grouped list 248pt including its Load more
        // footer, and a drag that starts on the list scrolls the page rather
        // than bouncing the list. Whether the LazyVStack stays lazy under that
        // proposal has not been measured; a long feed should carry a Height
        // anyway so that OnEndReached has a viewport to act on.
        ScrollViewReader { proxy in
        ScrollView {
            // Two stacks rather than one taking an empty pinnedViews, because
            // a Section is not free of consequence: it changes what the lazy
            // stack's items *are*, and every List that predates
            // core.StickyHeader should compose exactly as it did. Sections
            // exist here only when a child actually asked to be pinned, which
            // is what listSections returning nil says.
            //
            // A Transition declared on the List itself animates row
            // *placement*: keyed rows slide/fade on reorder, insertion, and
            // removal instead of teleporting — the Android renderer's
            // animateItemPlacement analog. Scoped by the row-identity array
            // so only structural changes trigger it; a row's own property
            // changes animate under its own Transition via grMobBox. Written
            // on both branches rather than on a wrapper, so neither stack
            // gains a view between itself and the scroll.
            if let sections = listSections(rows) {
                LazyVStack(alignment: crossAlignmentH(s),
                           spacing: s?.verticalGap ?? 0,
                           pinnedViews: [.sectionHeaders]) {
                    ForEach(sections) { group in
                        Section {
                            ForEach(group.rows, id: \.viewID) { child in
                                rowView(child, stretch: stretch, last: rows.last)
                            }
                        } header: {
                            // An EmptyView header is what the leading run
                            // gets: the rows above the first band, which
                            // belong to no group and must not be pinned.
                            if let header = group.header {
                                rowView(header, stretch: stretch, last: rows.last)
                            }
                        }
                    }
                }
                .scrollTargetLayout()
                .animation(reduceMotion ? nil : s?.swiftUIAnimation, value: rows.map(\.viewID))
            } else {
                LazyVStack(alignment: crossAlignmentH(s), spacing: s?.verticalGap ?? 0) {
                    ForEach(rows, id: \.viewID) { child in
                        rowView(child, stretch: stretch, last: rows.last)
                    }
                }
                .scrollTargetLayout()
                .animation(reduceMotion ? nil : s?.swiftUIAnimation, value: rows.map(\.viewID))
            }
        }
        // core.StartAtEnd and core.OnStartReached. Three parts here, and
        // stickToEnd below.
        //
        // The anchor opens the content at its bottom. nil is the default
        // anchor, so the modifier is unconditional and every other List
        // composes as it did.
        //
        // The anchor does not keep the reader's place when an older page is
        // prepended: measured on the iOS 26.5 simulator, 4.33's thread stayed
        // at its top after each page landed, the new first row appeared, and
        // the thread loaded all three pages from one arrival at the top.
        // scrollPosition(id:) is SwiftUI's own answer: bound to the id of the
        // row at the top edge (the stacks are its scrollTargetLayout), it
        // keeps that row in position when rows are inserted before it, so
        // the older page lands above the reader. Bound only on a list that
        // tracks its top; any other gets a constant nil, which tracks nothing.
        //
        // The same binding is the top edge (topEdgeReached), and a change to
        // the last row is stickToEnd's cue.
        .defaultScrollAnchor(startAtEnd ? .bottom : nil)
        .scrollPosition(id: tracksTop ? $topRow : .constant(nil), anchor: .top)
        .onChange(of: topRow) { _, top in
            topEdgeReached(top, rows: rows, callback: startReached)
        }
        .onChange(of: rows.last?.rowKey) { old, new in
            stickToEnd(startAtEnd, old: old, new: new, proxy: proxy)
        }
        }
        .grMobKeyboardAware(node.boolProp("keyboardAware"))
        .grMobBox(s, grow: grow,
                    onTap: node.stringProp("onClick"),
                    onLongPress: node.stringProp("onLongPress"),
                    axis: .vertical)
    }

    /// core.OnStartReached: the row at the top edge is one of the first
    /// START_REACHED_SLACK rows.
    ///
    /// Read from the scrollPosition binding, which SwiftUI keeps on the row
    /// at the top edge, and not from the first row's .onAppear, the way the
    /// end edge is read. A LazyVStack materializes rows ahead of the viewport,
    /// and on the iOS 26.5 simulator that was enough for the first row of 4.33's
    /// thread to "appear" as the reader neared the top and again as soon as
    /// each older page landed: one drag loaded two pages, the next the third.
    /// The binding says which row the reader is actually at.
    private func topEdgeReached(_ top: String?, rows: [GrMobNode], callback: String) {
        guard !callback.isEmpty, let top else { return }
        let slack = 2
        if rows.prefix(slack).contains(where: { $0.rowKey == top }) {
            dispatch?(callback)
        }
    }

    /// core.StartAtEnd's second half: a new last row, arriving while the
    /// reader was at the end, is scrolled into view at the bottom. A reader
    /// who has scrolled back is left where they are, and a prepend (which
    /// leaves the last row as it was) moves nothing.
    ///
    /// Needed because the scrollPosition binding holds the row at the top
    /// edge through every change, appends included: measured on the iOS 26.5
    /// simulator, a message sent at the end of 4.33's thread landed below the
    /// box, out of view, with the anchor at .bottom all the while.
    private func stickToEnd(_ startAtEnd: Bool, old: String?, new: String?, proxy: ScrollViewProxy) {
        guard startAtEnd, nearEnd, let old, let new, old != new else { return }
        withAnimation { proxy.scrollTo(new, anchor: .bottom) }
    }

    /// One row, plus the end-reached trip wire on the last of them.
    ///
    /// Cross-axis stretch, the same contract GrMobFlexStack implements for
    /// Row/Column. A lazy stack cannot be replaced by a custom Layout —
    /// laziness is the whole point of using it — so stretch is expressed the
    /// only way it can be here: a flexible frame on each row. There is no
    /// main-axis counterpart because a scrolling axis has no leftover space
    /// for FlexGrow to divide.
    ///
    /// Read through crossAxisValue — the same read the stack's alignment
    /// makes — so the Style.Align fallback reaches this binding too. It used
    /// to test alignItems alone while crossAlignmentH read the fallback, and
    /// the two answers disagreed exactly when it mattered: Align: stretch
    /// with AlignItems unset took crossAlignmentH's "stretch" arm, whose
    /// comment promises this frame does the filling, and then no frame was
    /// applied. See crossAxisValue for the pin that keeps the two reads
    /// together.
    ///
    /// core.OnEndReached rides the last row's .onAppear. A LazyVStack
    /// materializes a row shortly before it scrolls into view, so "appeared"
    /// is already a little ahead of "seen" — which is the same head start
    /// Compose buys with its three-row slack and the runtime with its
    /// rootMargin. The identity test is against the row the *whole list* ends
    /// with, not the section's, so a pinned-header list arms the wire once
    /// rather than once per group.
    ///
    /// .onAppear fires again every time the row is recycled back into view,
    /// which is expected and deliberately not guarded here: core.OnEndReached
    /// debounces on the Go side by remembering the row count at the last
    /// fire, which is where all four renderers' different notions of "again"
    /// are reconciled. An empty list has no last row and so never reports the
    /// edge — the first page is the app's to ask for.
    private func rowView(_ child: GrMobNode, stretch: Bool, last: GrMobNode?) -> some View {
        let endReached = node.stringProp("onEndReached")
        return RenderNode(node: child,
                          grow: stretch && !hugsContent(child.style) ? .horizontal : .none)
            .onAppear {
                if child === last { nearEnd = true }
                guard !endReached.isEmpty, child === last else { return }
                dispatch?(endReached)
            }
            .onDisappear {
                if child === last { nearEnd = false }
            }
            // The row's id as scrollPosition(id:) reads it: a String, the type
            // topRow is bound as (see GrMobNode.rowKey). It names the same row
            // the ForEach's viewID does, so a row's identity never changes
            // under it.
            .id(child.rowKey)
    }
}

/// A run of List rows, optionally introduced by a pinned band.
///
/// Identified by the header's view identity when there is one and by the
/// first row's otherwise, so a ForEach over these groups is as stable as a
/// ForEach over the rows was: both are derived from the same keys the
/// reconciler assigned.
private struct GrMobListGroup: Identifiable {
    let id: AnyHashable
    let header: GrMobNode?
    let rows: [GrMobNode]
}

/// Splits a List's children into sticky-header sections.
///
/// A child carrying core.StickyHeader (Style.Position == "sticky") opens a
/// new section and becomes its header; everything after it, up to the next
/// such child, is that section's content. Rows before the first header — a
/// search box above the bands, a list with no headers at all — form one
/// leading section with no header, which SwiftUI renders as a plain run.
///
///     rows:     [search] [Jan] a b [Feb] c
///     sections: (nil: search) (Jan: a b) (Feb: c)
///
/// nil when no child asked for a band — an ordinary list, which the caller
/// renders as the flat lazy stack it has always been. The distinction is the
/// whole reason this returns an optional rather than a one-group array: an
/// unsectioned list must not acquire a Section it never asked for.
private func listSections(_ rows: [GrMobNode]) -> [GrMobListGroup]? {
    guard rows.contains(where: { $0.style?.position == "sticky" }) else { return nil }

    var groups: [GrMobListGroup] = []
    for row in rows {
        if row.style?.position == "sticky" {
            groups.append(GrMobListGroup(id: row.viewID, header: row, rows: []))
            continue
        }
        guard let open = groups.last else {
            // No section open yet: these are the rows above the first band.
            groups.append(GrMobListGroup(id: row.viewID, header: nil, rows: [row]))
            continue
        }
        groups[groups.count - 1] = GrMobListGroup(
            id: open.id, header: open.header, rows: open.rows + [row])
    }
    return groups
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
///  - A fixed Width, in points. CSS never stretches an item with a definite
///    cross size; here the flexible frame grMobGrow adds would sit outside
///    the fixed one grMobDimension adds and win, so the child has to skip
///    the stretch rather than override it. A percentage is not fixed: it is
///    relative to the parent's extent, so the child must be *proposed* that
///    extent (Width("100%") is how comps.Button spells FullWidth, and
///    a button proposed only its label's width had nothing to fill).
///  - An inline Display. The bundled themes give Button (and Badge) an
///    inline display as their way of saying "hug your content", and
///    grmob-runtime.js turns that into `width: fit-content` for exactly this
///    case — a flex column would otherwise spread every button across the
///    screen. comps.Button's FullWidth is the documented way to ask
///    for the stretch back (it sets both Width and a block display).
///
/// Text is not exempt: a stretched Text is the same picture as a hugging
/// one (its alignment is a text property), and stretching it is what lets
/// Align(AlignCenter) on a Column child centre its lines.
private func hugsContent(_ s: GrMobStyle?) -> Bool {
    guard let s else { return false }
    let fixedWidth = !s.width.isEmpty && !s.width.hasSuffix("%")
    return fixedWidth || s.display == "inline" || s.display == "inline-block"
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
        // core.MaxLines: nil is SwiftUI's "no limit". Tail truncation is
        // Text's default already; it is stated so the pairing with the other
        // targets' trailing ellipsis is visible here.
        let cap = node.style?.maxLines ?? 0
        Text(node.stringProp("content"))
            .grMobTextStyle(node.style)
            .lineLimit(cap > 0 ? cap : nil)
            .truncationMode(.tail)
            .grMobBox(node.style, grow: grow,
                        onTap: node.stringProp("onClick"),
                        onLongPress: node.stringProp("onLongPress"))
    }
}

/// core.Paragraph: one Text of an AttributedString, a run per part.
///
/// The runs prop is a list of maps with core/paragraph.go's keys: t, and b, i,
/// u, s, c (bold, italic, underline, strike, code) present as 1, fg a colour,
/// cb a void callback ID. The paragraph's own style is the base, applied by
/// grMobTextStyle exactly as a Text's is; bold, italic and code are
/// presentation intents, so they take the base font and change only its
/// weight, slant or pitch.
///
/// # Links
///
/// A run with a callback is a link: its `link` attribute is a `grmob-run:`
/// URL naming the callback, and the OpenURLAction below catches that scheme
/// and dispatches the callback instead of opening anything. That is the one
/// way SwiftUI's Text lets a range of it be tapped, and it gives VoiceOver the
/// run as a link, in the rotor with the paragraph's others, for free.
///
/// Each link run carries its own colour as `foregroundColor`, which Go has
/// already resolved (the theme's Primary unless the run named one), and that
/// is what draws it: on the iOS 26.5 simulator, 4.29's sentence draws its
/// guide link in Primary and its "report a problem" link in Error, side by
/// side (TutorialDevicePassUITests.testParagraphLinksKeepTheirOwnColours).
/// SwiftUI's default for a link is the tint, which is why the tint is still
/// set to the first link's colour: it is the fallback where a run's own
/// colour does not reach a link (only iOS 26.5 has been measured, and the
/// floor is 17), and there it is the closest single answer. An earlier note
/// here called one colour per paragraph a limit of this platform; the
/// measurement says it is not.
private struct GrMobParagraph: View {
    let node: GrMobNode
    let grow: GrMobGrow
    @Environment(\.grMobRuntime) private var runtime

    private static let scheme = "grmob-run"

    var body: some View {
        let runs = node.props["runs"] as? [[String: Any]] ?? []
        let cap = node.style?.maxLines ?? 0
        Text(Self.attributed(runs))
            .grMobTextStyle(node.style)
            .lineLimit(cap > 0 ? cap : nil)
            .truncationMode(.tail)
            .tint(Self.linkTint(runs))
            .environment(\.openURL, OpenURLAction { url in
                guard url.scheme == Self.scheme else { return .systemAction }
                let id = String(url.absoluteString.dropFirst(Self.scheme.count + 1))
                    .removingPercentEncoding ?? ""
                if !id.isEmpty { runtime?.click(id) }
                return .handled
            })
            .grMobBox(node.style, grow: grow,
                        onTap: node.stringProp("onClick"),
                        onLongPress: node.stringProp("onLongPress"))
    }

    private static func on(_ run: [String: Any], _ key: String) -> Bool {
        if let n = run[key] as? NSNumber { return n.intValue != 0 }
        return (run[key] as? Bool) ?? false
    }

    static func attributed(_ runs: [[String: Any]]) -> AttributedString {
        var out = AttributedString()
        for run in runs {
            let text = run["t"] as? String ?? ""
            guard !text.isEmpty else { continue }
            var part = AttributedString(text)
            var intent: InlinePresentationIntent = []
            if on(run, "b") { intent.insert(.stronglyEmphasized) }
            if on(run, "i") { intent.insert(.emphasized) }
            if on(run, "c") { intent.insert(.code) }
            if !intent.isEmpty { part.inlinePresentationIntent = intent }
            if on(run, "u") { part.underlineStyle = .single }
            if on(run, "s") { part.strikethroughStyle = .single }
            if let fg = GrMobStyle.parseColor(run["fg"] as? String) { part.foregroundColor = fg }
            if let cb = run["cb"] as? String, !cb.isEmpty,
               let encoded = cb.addingPercentEncoding(withAllowedCharacters: .alphanumerics),
               let url = URL(string: "\(scheme):\(encoded)") {
                part.link = url
            }
            out += part
        }
        return out
    }

    private static func linkTint(_ runs: [[String: Any]]) -> Color? {
        for run in runs where (run["cb"] as? String).map({ !$0.isEmpty }) ?? false {
            return GrMobStyle.parseColor(run["fg"] as? String)
        }
        return nil
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
///
/// The ladder itself is grMobFontWeightPair, in GrMobMinContent.swift, which
/// answers in two vocabularies at once — this one and the number CoreText
/// wants — so that the face the min-content floor is measured at is the face
/// that gets drawn. See that function for why it lives on that side.
private func grMobFontWeight(_ w: Int) -> Font.Weight { grMobFontWeightPair(w).0 }

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
    /// For grMobShape: core's corners are physical, SwiftUI's leading/trailing.
    @Environment(\.layoutDirection) private var layoutDirection

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
        // The Button's action, named so the extra keyboard chords behind it
        // (grMobKeyShortcut) run exactly what a tap runs.
        let press = {
            if longPressFired {
                longPressFired = false
                return
            }
            if !onClick.isEmpty { runtime?.click(onClick) }
        }
        Button(action: press) {
            Text(node.stringProp("label"))
                .font(.system(size: (s?.fontSize ?? 0) > 0 ? s!.fontSize : 17,
                              weight: grMobFontWeight(s?.fontWeight ?? 0)))
                .foregroundStyle(s?.textColor ?? .white)
                .padding(paddingOrDefault(s))
                .frame(maxWidth: grow == .horizontal ? .infinity : nil)
        }
        .buttonStyle(GrMobButtonStyle(
            background: s?.background ?? .accentColor,
            // The Button's own default of 8, and corners when it named them.
            shape: grMobShape(s, defaultRadius: 8, direction: layoutDirection) ?? UnevenRoundedRectangle(),
            // The border travels with the other container fields rather than
            // through grMobBox, which this view is handed a stripped style for
            // (marginAndSizeOnly). Without it core.BorderColor/BorderWidth were
            // the one pair a Button silently dropped, so an outlined button had
            // its rule on the web and none on device.
            borderColor: s?.borderColor,
            borderWidth: s?.borderWidth ?? 0
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
        // core.AccessibilityKeyShortcuts: a page-global chord (Control, Alt
        // or Meta held) presses this button from a hardware keyboard, which
        // also lists it in iPadOS's shortcut overlay. On the Button itself,
        // because keyboardShortcut triggers the primary action of the view it
        // modifies, and a disabled button's shortcut is disabled with it.
        .grMobKeyShortcut(s?.accessibilityKeyShortcuts ?? "", press: press)
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
    let shape: UnevenRoundedRectangle
    /// nil / 0 mean "no border", which is grMobBorder's identity case and the
    /// state every button was in before this pair was carried.
    let borderColor: Color?
    let borderWidth: CGFloat

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .background(background)
            .clipShape(shape)
            // After the clip and on the same shape, so the stroke lands exactly
            // on the edge the fill was cut to. strokeBorder insets it inward
            // rather than straddling the edge, which is the placement
            // Modifier.border gives it on Compose and the one grMobBox already
            // uses for every other node.
            .grMobBorder(shape, color: borderColor, width: borderWidth)
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
    t.corners = nil
    t.shadow = 0
    t.padding = .zero
    return t
}

/// A core.Select: the chosen option's label, with the list hung off it as a
/// menu.
///
/// # Why a Menu and not a Picker
///
/// SwiftUI has a picker control — `Picker(...).pickerStyle(.menu)` — and it
/// draws a frame, a chevron and an inset of its own that no Go style can
/// remove. That would make the picker the one control in the vocabulary whose
/// edge came from the platform here and from the theme everywhere else, which
/// is exactly the divergence htmlout's borderResetTypes was extended to
/// prevent: the <select> row in that set rests on this arm drawing nothing but
/// what the style asks for.
///
/// So the box is grMobBox's, like every other node's, and the Menu supplies
/// only the behavior. The label takes the style's font and ink the way
/// GrMobButton's does, for the same reason — the theme's Input base is what
/// core.Select reads, and a menu label rendered in the system default would
/// not match the text fields beside it.
///
/// Controlled, like every other input: the label shown is whichever option
/// matches Go's value, and a choice goes up as the option's *value*. An
/// unmatched value falls back to showing the raw string rather than an empty
/// box — the same degradation a <select> makes, and the honest one for a state
/// the app has put the widget in.
private struct GrMobSelect: View {
    let node: GrMobNode
    let grow: GrMobGrow
    @Environment(\.grMobRuntime) private var runtime

    var body: some View {
        let s = node.style
        let options = node.props["options"] as? [[String: Any]] ?? []
        let value = node.stringProp("value")
        let cb = node.stringProp("onChange")
        let chosen = options.first { ($0["value"] as? String) == value }

        Menu {
            // The options, split into the runs core.SelectOption.Group
            // describes. The split itself is grMobMenuSections in
            // GrMobSelectMenu.swift — UI-free, so ios/verify runs it against
            // cases generated from core.SelectMenuSections rather than reading
            // this file as text. Sectioned ahead of the ForEach because
            // SwiftUI's Section is a container and a run has to be handed to
            // it whole.
            ForEach(grMobMenuSections(options), id: \.first) { section in
                if section.heading.isEmpty {
                    grMobMenuItems(section.items, cb, runtime)
                } else {
                    Section(section.heading) {
                        grMobMenuItems(section.items, cb, runtime)
                    }
                    // The one place a modifier lands on the Section rather
                    // than on its buttons, and the only case where taking the
                    // whole run down is what was asked for:
                    // core.SelectOption.GroupDisabled. The buttons are already
                    // refused — core propagates a disabled run onto its items,
                    // because the two targets with no section construct have
                    // nowhere else to read it — so what this line is for is
                    // the header, which would otherwise stay as legible as the
                    // ones above it. An ungrouped run has no header and takes
                    // no branch here.
                    .disabled(section.isDisabled)
                }
            }
        } label: {
            Text(chosen?["label"] as? String ?? value)
                .font(.system(size: (s?.fontSize ?? 0) > 0 ? s!.fontSize : 17,
                              weight: grMobFontWeight(s?.fontWeight ?? 0)))
                .foregroundStyle(s?.textColor ?? .primary)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
        .grMobBox(s, grow: grow)
    }
}

/// The buttons for one run of options.
///
/// A disabled option is still drawn and still announced — that is what
/// disabling one buys over leaving it out — and `.disabled` is what stops the
/// tap. It goes on the Button, which is the only granularity
/// core.SelectOption.Disabled has: disabling a Section takes its whole run
/// with it, and that is a different declaration (GroupDisabled) made one level
/// up, where the Section is.
///
/// The choice goes up as the option's *value*. core.Select registers a
/// func(string), so the dispatch is the text channel; the index is the one
/// identity that changes when the list is reordered, and a label is written to
/// be read.
@ViewBuilder
private func grMobMenuItems(_ items: [GrMobMenuItem], _ cb: String,
                            _ runtime: GrMobRuntime?) -> some View {
    // Keyed on the option's index, which is what GrMobMenuItem carries it for:
    // a [String: Any] is not Hashable and two options may share a label.
    ForEach(items, id: \.index) { item in
        Button(item.label) {
            if !cb.isEmpty {
                runtime?.textChanged(cb, item.value)
            }
        }
        .disabled(item.isDisabled)
    }
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
            // The theme's accent on the on track (core.AccentColor). A nil
            // tint is the system colour, so this is one modifier and not a
            // branch.
            .tint(node.style?.accentColor)
            .grMobBox(marginAndSizeOnly(node.style), grow: grow)
    }
}

/// A core.Switch: SwiftUI's Toggle, which is what a switch is on this
/// platform. Same wire shape as the Checkbox above — the state arrives as
/// `checked` and the change goes up through the bool channel — because it is
/// the same bool; see core.Switch for why the two are separate node types and
/// why the prop keeps the DOM's spelling rather than Go's `on`.
///
/// It draws identically to GrMobCheckbox, and that is the honest answer rather
/// than a gap. iOS has no checkbox control, so a Checkbox already borrows this
/// one; here the borrowing runs the other way and the control is the thing it
/// was made for. The two node types still earn their separation on the targets
/// that draw them apart (Material's Checkbox and Switch, and the `switch`
/// attribute on the web) and in what the app is saying: a Checkbox on iOS that
/// sat beside a Submit button was always drawing a switch, and writing
/// core.Switch for a setting that takes effect on the tap says so.
private struct GrMobSwitch: View {
    let node: GrMobNode
    let grow: GrMobGrow
    @Environment(\.grMobRuntime) private var runtime

    var body: some View {
        let cb = node.stringProp("onToggle")
        Toggle(isOn: Binding(
            get: { node.boolProp("checked") },
            set: { if !cb.isEmpty { runtime?.toggled(cb, $0) } }
        )) { EmptyView() }
            .labelsHidden()
            // The theme's accent on the on track (core.AccentColor). A nil
            // tint is the system colour, so this is one modifier and not a
            // branch.
            .tint(node.style?.accentColor)
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
        // The node's name and hint on the Slider itself, and not through
        // grMobBox, for the reason GrMobTextField.boxStyle gives for the text
        // fields: grMobBox names a node with `.accessibilityElement(children:
        // .combine)`, which around a native control makes a new element.
        // Around a Slider that element kept the slider type and lost the
        // slider: XCUITest's adjust(toNormalizedSliderPosition:) failed on
        // comps.AudioPlayer's "Position" bar with "Unable to get expected
        // attributes for slider" (its scrubber positions read 0,0). On the
        // Slider, the name is the control's own and it stays adjustable.
        // The value (core.ValueRange.Text) still comes from grMobBox, whose
        // accessibilityValue lands on the Slider now that nothing wraps it.
        .grMobControlName(node.style)
        // The theme's accent on the filled track (core.AccentColor).
        .tint(node.style?.accentColor)
        .grMobBox(unnamed(marginAndSizeOnly(node.style)), grow: grow)
    }
}

/// A style with its accessible name and hint removed, for a native control
/// that states them on itself (grMobControlName). The same stripping
/// GrMobTextField.boxStyle does for the text fields, kept here because the
/// ios/verify harness compiles this file without GrMobTextInput.swift.
private func unnamed(_ s: GrMobStyle?) -> GrMobStyle? {
    guard var t = s else { return nil }
    t.accessibilityLabel = ""
    t.accessibilityHint = ""
    return t
}

extension View {
    /// A native control's stated name and hint, applied to the control. Each
    /// only when stated: `.accessibilityLabel("")` would blank the name the
    /// control reads for itself.
    @ViewBuilder fileprivate func grMobControlName(_ s: GrMobStyle?) -> some View {
        let label = s?.accessibilityLabel ?? ""
        let hint = s?.accessibilityHint ?? ""
        if label.isEmpty && hint.isEmpty {
            self
        } else if hint.isEmpty {
            accessibilityLabel(label)
        } else if label.isEmpty {
            accessibilityHint(hint)
        } else {
            accessibilityLabel(label).accessibilityHint(hint)
        }
    }
}

/// The miter limit every target draws with. SVG's `stroke-miterlimit` and
/// Compose's `Stroke.DefaultMiter` are both 4; SwiftUI's `StrokeStyle` defaults
/// to 10, which would draw a sharp corner's spike up to 2.5× longer here than on
/// the web and Android. Pinning it also bounds GrMobCanvas's outset.
let grMobCanvasMiterLimit: CGFloat = 4

/// A core.Canvas: a SwiftUI Canvas that draws each CanvasShape child as a Path.
///
///     core.Canvas(100, 50, shapes, core.CanvasStretch)
///       ──▶ Canvas { ctx, size in for each child: fill(path); stroke(path) }
///
/// # The shapes are data
///
/// The children are never rendered as views; their props are read inside the
/// Canvas closure, the way GrMobTextGrid reads its rows. With @Observable
/// tracking an update-props on one shape invalidates this view, and the
/// closure redraws the whole drawing — which for a Canvas is the unit anyway.
///
/// # Transformed points, untransformed strokes
///
/// Points go through GrMobCanvasViewport before they reach the Path, and the
/// stroke is drawn in points untransformed. That is core.Canvas's rule —
/// strokes are layout units, never scaled — spelled the way SVG's
/// vector-effect="non-scaling-stroke" spells it on the web. A context
/// scaleBy would have scaled the line width with the geometry.
///
/// # Sizing
///
/// core's defaults go inside the box modifier, so an author's Width and
/// Height (applied by grMobBox, outside) win: fill the proposed width when no
/// Width was given, and take the viewBox's aspect ratio when no Height was.
/// SwiftUI's Canvas clips to its frame, unlike the web's overflow:visible, so
/// the drawing is laid out over an outset and translated back (see body).
private struct GrMobCanvas: View {
    let node: GrMobNode
    let grow: GrMobGrow

    var body: some View {
        let vw = node.doubleProp("vw") > 0 ? node.doubleProp("vw") : 100
        let vh = node.doubleProp("vh") > 0 ? node.doubleProp("vh") : 100
        let stretch = node.stringProp("scale") == "stretch"
        let fillWidth = (node.style?.width ?? "").isEmpty
        let keepRatio = (node.style?.height ?? "").isEmpty
        let shapes = node.children.map(\.props)

        // A SwiftUI Canvas clips to its frame, but the other three targets do
        // not: htmlout's <svg> is overflow:visible and Compose's drawBehind has
        // no clip. So a stroke centred on the box's edge (a chart's baseline,
        // a round-capped dot on the last point at x = vw) drew half of itself
        // on the web and Android and was cut in half here.
        //
        //   ┌ outset ─────────────────┐   the Canvas is laid out `outset`
        //   │ ┌ layout box ─────────┐ │   larger on every side by a negative
        //   │ │ viewBox maps here   │ │   padding, which reports the original
        //   │ └─────────────────────┘ │   size to the parent; the drawing is
        //   └─────────────────────────┘   translated back by `outset`
        //
        // The outset is the furthest any stroke can reach past its path, taken
        // over every shape without measuring paths:
        //
        //   round / square cap    w/2 (w/2·√2 on a diagonal square cap)  ≤ w
        //   round / bevel join    w/2                                     ≤ w
        //   miter join            (w/2)/sin(θ/2) for a corner of angle θ,
        //                         cut to a bevel once that passes
        //                         miterLimit·w/2 — so at most 2w at limit 4
        //
        // A miter is the default join, so a stroked shape without a round or
        // bevel join gets 2w and every other stroked shape w. The limit is
        // pinned to 4 below, which is what bounds the miter case at all.
        // Layout, hit-testing and the viewport arithmetic all still see the
        // unpadded box.
        let outset = CGFloat(shapes.reduce(0.0) { acc, props in
            guard props["stroke"] != nil || props["strokeGradient"] != nil else { return acc }
            let w = (props["strokeWidth"] as? NSNumber)?.doubleValue ?? 1
            let join = props["join"] as? String
            let reach = (join == "round" || join == "bevel") ? w : w * Double(grMobCanvasMiterLimit) / 2
            return max(acc, reach)
        })

        let drawing = Canvas { ctx, outer in
            let size = CGSize(width: max(0, outer.width - 2 * outset),
                              height: max(0, outer.height - 2 * outset))
            ctx.translateBy(x: outset, y: outset)
            let vp = GrMobCanvasViewport(vw: vw, vh: vh, width: Double(size.width),
                                         height: Double(size.height), stretch: stretch)
            for props in shapes {
                guard let ops = props["d"] as? [Any] else { continue }
                var path = Path()
                grMobDecodeCanvasPath(ops, vp) { call in
                    switch call {
                    case let .move(x, y): path.move(to: CGPoint(x: x, y: y))
                    case let .line(x, y): path.addLine(to: CGPoint(x: x, y: y))
                    case let .cubic(x1, y1, x2, y2, x, y):
                        path.addCurve(to: CGPoint(x: x, y: y),
                                      control1: CGPoint(x: x1, y: y1),
                                      control2: CGPoint(x: x2, y: y2))
                    case .close: path.closeSubpath()
                    }
                }
                // Fill first, then the stroke over it: SVG's paint order.
                // Go writes "fill" or the gradient keys, never both.
                // core.FillEvenOdd; nonzero is FillStyle's default.
                let fillStyle = FillStyle(eoFill: props["fillRule"] as? String == "evenodd")
                if let gradient = grMobCanvasGradient(props) {
                    grMobFillGradient(in: ctx, path: path, gradient: gradient, viewport: vp, style: fillStyle)
                } else if let fill = GrMobStyle.parseColor(props["fill"] as? String) {
                    ctx.fill(path, with: .color(fill), style: fillStyle)
                }
                // Go writes "stroke" or the strokeGradient keys, never both.
                let strokeGradient = grMobCanvasGradient(props, prefix: "stroke")
                let strokeColor = GrMobStyle.parseColor(props["stroke"] as? String)
                guard strokeGradient != nil || strokeColor != nil else { continue }
                let width = (props["strokeWidth"] as? NSNumber)?.doubleValue ?? 1
                var dash = ((props["dash"] as? [Any]) ?? []).compactMap { ($0 as? NSNumber)?.doubleValue }
                // SVG repeats an odd dash list to make it even; do the same.
                if dash.count % 2 == 1 { dash += dash }
                let cap: CGLineCap = switch props["cap"] as? String {
                case "round": .round
                case "square": .square
                default: .butt
                }
                let join: CGLineJoin = switch props["join"] as? String {
                case "round": .round
                case "bevel": .bevel
                default: .miter
                }
                let strokeStyle = StrokeStyle(lineWidth: width, lineCap: cap, lineJoin: join,
                                              miterLimit: grMobCanvasMiterLimit,
                                              dash: dash.map { CGFloat($0) })
                if let gradient = strokeGradient {
                    // A gradient stroke is the stroke's outline, taken in box
                    // points so its width is untransformed, then filled with
                    // the gradient in viewBox space like a gradient fill.
                    // Stroking inside grMobFillGradient's transformed context
                    // instead would scale the width with the viewport, and
                    // unequally on the two axes under stretch. The outline is
                    // filled nonzero: a stroke's self-overlaps (a loop, a
                    // round join) must all paint, as ctx.stroke paints them.
                    grMobFillGradient(in: ctx, path: path.strokedPath(strokeStyle), gradient: gradient,
                                      viewport: vp, style: FillStyle())
                } else if let stroke = strokeColor {
                    ctx.stroke(path, with: .color(stroke), style: strokeStyle)
                }
            }
        }
        .padding(-outset)
        // The outset area must not take taps meant for a neighbour.
        .allowsHitTesting(false)

        // Conditional rather than aspectRatio(nil, ...): a nil ratio is not
        // "no ratio" but "the child's ideal size's ratio", and a Canvas's ideal
        // size is an arbitrary placeholder that would squash a canvas whose
        // author set a Height.
        Group {
            if keepRatio {
                drawing.aspectRatio(CGFloat(vw / vh), contentMode: .fit)
            } else {
                drawing
            }
        }
        .frame(maxWidth: fillWidth ? .infinity : nil)
        .grMobBox(node.style, grow: grow,
                  onTap: node.stringProp("onClick"),
                  onLongPress: node.stringProp("onLongPress"))
    }
}

/// A core.Gradient fill, decoded from a shape's gradient keys: the geometry in
/// viewBox units (four numbers for linear, three for radial) and the stops.
struct GrMobCanvasGradientSpec {
    let radial: Bool
    let at: [Double]
    let gradient: Gradient
}

/// Reads a shape's gradient keys, or nil when it has none or they are
/// malformed (which paints nothing, as the web targets do). `prefix` picks the
/// paint: "" for the fill's keys (gradient, gradientAt, ...), "stroke" for the
/// stroke's (strokeGradient, strokeGradientAt, ...); see core.GradientKey.
func grMobCanvasGradient(_ props: [String: Any], prefix: String = "") -> GrMobCanvasGradientSpec? {
    func key(_ name: String) -> String {
        prefix.isEmpty ? name : prefix + name.prefix(1).uppercased() + name.dropFirst()
    }
    guard let kind = props[key("gradient")] as? String,
          let rawAt = props[key("gradientAt")] as? [Any],
          let rawStops = props[key("gradientStops")] as? [Any],
          let rawColors = props[key("gradientColors")] as? [Any],
          !rawStops.isEmpty, rawStops.count == rawColors.count else { return nil }
    let at = rawAt.compactMap { ($0 as? NSNumber)?.doubleValue }
    guard at.count == rawAt.count else { return nil }
    var stops: [Gradient.Stop] = []
    for (o, c) in zip(rawStops, rawColors) {
        guard let offset = (o as? NSNumber)?.doubleValue,
              let color = GrMobStyle.parseColor(c as? String) else { return nil }
        stops.append(Gradient.Stop(color: color, location: CGFloat(offset)))
    }
    switch kind {
    case "linear" where at.count == 4:
        return GrMobCanvasGradientSpec(radial: false, at: at, gradient: Gradient(stops: stops))
    case "radial" where at.count == 3 && at[2] > 0:
        return GrMobCanvasGradientSpec(radial: true, at: at, gradient: Gradient(stops: stops))
    default:
        return nil
    }
}

/// Fills `path` (already mapped into box points) with a gradient whose
/// geometry is in viewBox units.
///
/// The gradient is drawn in a copy of the context that carries the viewport's
/// transform, with the path mapped back into viewBox units for it, so the
/// gradient's whole space is scaled the way SVG scales userSpaceOnUse under a
/// viewBox. Mapping only its end points would keep a stretched radial circular
/// and a stretched diagonal's bands perpendicular on screen; see core.Gradient
/// and canvasGradientBrush in GrMobCanvas.kt, which does the same with a
/// shader matrix.
///
///   box point = viewBox point · scale + offset
///   ctx'      = ctx ∘ translate(offset) ∘ scale(scale)
///
/// GraphicsContext is a value type, so the copy's transform does not leak into
/// the shapes drawn after this one. A zero scale (a box with no area) has no
/// inverse and nothing to paint, so it is skipped.
func grMobFillGradient(in ctx: GraphicsContext, path: Path, gradient: GrMobCanvasGradientSpec,
                       viewport vp: GrMobCanvasViewport, style: FillStyle) {
    guard vp.scaleX != 0, vp.scaleY != 0 else { return }
    let toBox = CGAffineTransform(translationX: vp.offsetX, y: vp.offsetY)
        .scaledBy(x: vp.scaleX, y: vp.scaleY)
    var local = ctx
    local.concatenate(toBox)
    let at = gradient.at.map { CGFloat($0) }
    let shading: GraphicsContext.Shading = gradient.radial
        ? .radialGradient(gradient.gradient, center: CGPoint(x: at[0], y: at[1]),
                          startRadius: 0, endRadius: at[2])
        : .linearGradient(gradient.gradient, startPoint: CGPoint(x: at[0], y: at[1]),
                          endPoint: CGPoint(x: at[2], y: at[3]))
    local.fill(path.applying(toBox.inverted()), with: shading, style: style)
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

// core.Input, InputPassword, NumericInput and TextArea are GrMobTextField,
// in GrMobTextInput.swift: a UIKit field, because SwiftUI's TextField kept a
// second copy of the text and lost keys to it (see that file).

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
///
/// The sheet is also what makes this target need no dialog role. VoiceOver
/// announces a presented sheet as a modal in its own right and traps focus
/// inside it, so the semantics the two DOM renderers now write by hand
/// (role="dialog" plus aria-modal — see htmlout's modalSemantics) come free
/// here. That is why core.Role has no RoleDialog for an author to set.
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
                // Content taller than the sheet scrolls; content that fits does
                // not move.
                //
                // The sheet opens at the medium detent, about half the screen,
                // and gives its content exactly that height. Without a scroll
                // view the content was squeezed into it: GrMobFlexLayout floors
                // a Row's children at their min-content width but a Column's
                // at zero, so a DatePicker's six weeks were shrunk until each
                // day cell's marker dots drew over its number, and on a
                // landscape phone nothing could reach the rest. The other
                // targets now scroll the same overflow (GrMobModal in
                // Renderer.kt; the overlay's overflow-y on the web).
                //
                //     content height    before              now
                //     --------------    ------              ---
                //     fits the detent   top of the sheet    top of the sheet
                //     taller            squeezed to fit     natural size, scrolls
                //
                // Content that fits was already at the top of the sheet (the
                // placement comps.ActionSheet's type doc records), and a
                // ScrollView puts it there too, so nothing short moves. The
                // maxWidth frame keeps content narrower than the sheet centred
                // across it, where the sheet used to centre it. bounce is
                // basedOnSize so a short dialog does not rubber-band as though
                // it scrolled.
                //
                // Always a ScrollView rather than one only when the content is
                // too tall (ViewThatFits): switching containers changes the
                // content's structural identity, which resets @State below it,
                // and the switch would fire exactly when a keyboard rises and
                // shortens the sheet — dropping a text field's focus mid-type.
                //
                // comps.ActionSheet's grow filler is unaffected: PlainChildren
                // reads no grow, so it was zero-height here before and is now.
                ScrollView {
                    VStack(alignment: .leading, spacing: 0) {
                        PlainChildren(node: node)
                    }
                    .padding(16)
                    .frame(maxWidth: .infinity)
                }
                .scrollBounceBehavior(.basedOnSize)
                .presentationDetents([.medium, .large])
                .grMobPresentationSurface(surface)
            }
    }
}
