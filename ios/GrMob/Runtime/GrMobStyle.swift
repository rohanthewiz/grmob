import SwiftUI

/// Swift mirror of Go's core.Style, decoded from the tree/patch JSON.
///
/// Field names in the JSON are the Go struct's exported names verbatim
/// ("FontSize", "TextColor", ...) because core.Style carries no json tags.
/// Only the subset the Go DSL can actually produce today is mapped; the
/// web-oriented fields (Position, ZIndex, Animation, pseudo states) have no
/// SwiftUI analog at this layer and are intentionally ignored rather than
/// half-implemented. Transition IS mapped: Go declares it, SwiftUI drives
/// the frames (see swiftUIAnimation and grMobTransition below).
///
/// This is a value type swapped wholesale by `update-style` patches — the
/// exact analog of the Kotlin data class held in a `mutableStateOf`.
struct GrMobStyle: Equatable {
    struct Edges: Equatable {
        var top = 0, right = 0, bottom = 0, left = 0
        static let zero = Edges()
        var insets: EdgeInsets {
            EdgeInsets(top: CGFloat(top), leading: CGFloat(left),
                       bottom: CGFloat(bottom), trailing: CGFloat(right))
        }
    }

    var fontSize: CGFloat = 0
    var fontWeight: Int = 0
    var textColor: Color?
    var background: Color?
    var padding: Edges = .zero
    var margin: Edges = .zero
    var borderRadius: CGFloat = 0
    var shadow: CGFloat = 0
    var align: String = ""
    var display: String = ""
    var width: String = ""
    var height: String = ""
    var borderColor: Color?
    var borderWidth: CGFloat = 0
    var gap: CGFloat = 0
    /// core.RowGap / core.ColumnGap: the per-axis spacings. CSS `gap` IS
    /// `row-gap` plus `column-gap`, so these are not extra properties beside
    /// Gap but the two halves of it, and an axis value set explicitly wins
    /// over the isotropic one. Read through verticalGap/horizontalGap below
    /// rather than directly — a container knows its own axis and should ask
    /// for that axis's spacing, not pick between three fields itself.
    var rowGap: CGFloat = 0
    var columnGap: CGFloat = 0
    var justifyContent: String = ""
    var alignItems: String = ""
    var flexGrow: CGFloat = 0
    /// core.FlexWrap: "wrap" or "nowrap" (empty when unset). Read by GrMobRow only.
    var flexWrap: String = ""
    /// core.FlexDirection: "row" or "column" (empty when unset). Read by
    /// GrMobScroll only — every other container here has its axis fixed by
    /// construction (an HStack is an HStack), so the field would tell them
    /// nothing they do not already know. A Scroll is the one type with both
    /// spellings, and core.Horizontal() is how Go asks for the sideways one.
    var flexDirection: String = ""
    /// core.Style.Position, carried for its "sticky" value alone. The other
    /// three (relative/absolute/fixed) still have no analog at this layer;
    /// sticky does — a pinned Section header inside a LazyVStack is exactly
    /// what CSS sticky positioning means in a scrolling list. Read by
    /// GrMobList only; core.StickyHeader is the Go spelling.
    var position: String = ""
    var lineHeight: Int = 0
    var accessibilityLabel: String = ""
    var accessibilityHint: String = ""
    var accessibilityHidden: Bool = false
    /// Go's core.Role, verbatim; mapped to traits by grMobTraitsFor below.
    var accessibilityRole: String = ""
    /// Platform disabled state; see Go's core.Style.Disabled.
    var disabled: Bool = false
    var transition: String = ""

    /// The spacing between items stacked along one axis, resolving the CSS
    /// shorthand the way a browser does: the axis longhand when it is set,
    /// the isotropic Gap otherwise.
    ///
    ///     Column / List / Scroll  ── stack vertically ──▶ verticalGap   (RowGap)
    ///     Row                     ── stack horizontally ▶ horizontalGap (ColumnGap)
    ///     wrapping Row            ── both: items along horizontalGap,
    ///                                wrapped lines apart by verticalGap
    ///
    /// Named for the axis they space along rather than for the CSS property
    /// they come from, because `row-gap` spaces items *vertically* (it is the
    /// gap between rows) and reading the field name as the direction is the
    /// mistake this pair exists to make impossible.
    var verticalGap: CGFloat { rowGap != 0 ? rowGap : gap }

    var horizontalGap: CGFloat { columnGap != 0 ? columnGap : gap }

    /// The Go Transition declaration as a SwiftUI Animation, or nil when the
    /// node doesn't animate. Canonical Go form is "<ms>ms <easing>"
    /// (core.Transition); the CSS longhand ("all 0.3s ease") is tolerated —
    /// unknown tokens are skipped. The easing keywords map onto the CSS
    /// spec's cubic-bezier control points so one Go declaration animates
    /// with the same curve on every platform.
    var swiftUIAnimation: Animation? {
        var durationMs = 0
        var easing = "ease"
        for token in transition.split(separator: " ") {
            if token.hasSuffix("ms"), let v = Int(token.dropLast(2)) {
                durationMs = v
            } else if token.hasSuffix("s"), let v = Double(token.dropLast(1)) {
                durationMs = Int(v * 1000)
            } else if ["linear", "ease", "ease-in", "ease-out", "ease-in-out"].contains(String(token)) {
                easing = String(token)
            }
        }
        guard durationMs > 0 else { return nil }
        let d = Double(durationMs) / 1000
        switch easing {
        case "linear": return .linear(duration: d)
        case "ease-in": return .timingCurve(0.42, 0, 1, 1, duration: d)
        case "ease-out": return .timingCurve(0, 0, 0.58, 1, duration: d)
        case "ease-in-out": return .timingCurve(0.42, 0, 0.58, 1, duration: d)
        default: return .timingCurve(0.25, 0.1, 0.25, 1, duration: d) // "ease"
        }
    }

    static func parse(_ obj: [String: Any]?) -> GrMobStyle? {
        guard let obj else { return nil }
        func num(_ key: String) -> CGFloat { CGFloat((obj[key] as? NSNumber)?.doubleValue ?? 0) }
        func int(_ key: String) -> Int { (obj[key] as? NSNumber)?.intValue ?? 0 }
        func str(_ key: String) -> String { obj[key] as? String ?? "" }

        var s = GrMobStyle()
        s.fontSize = num("FontSize")
        s.fontWeight = int("FontWeight")
        s.textColor = parseColor(str("TextColor"))
        s.background = parseColor(str("Background"))
        s.padding = parseEdges(obj["Padding"] as? [String: Any])
        s.margin = parseEdges(obj["Margin"] as? [String: Any])
        s.borderRadius = num("BorderRadius")
        s.shadow = num("Shadow")
        s.align = str("Align")
        s.display = str("Display")
        s.width = str("Width")
        s.height = str("Height")
        s.borderColor = parseColor(str("BorderColor"))
        s.borderWidth = num("BorderWidth")
        s.gap = num("Gap")
        s.rowGap = num("RowGap")
        s.columnGap = num("ColumnGap")
        s.justifyContent = str("JustifyContent")
        s.alignItems = str("AlignItems")
        s.flexGrow = num("FlexGrow")
        s.flexWrap = str("FlexWrap")
        s.flexDirection = str("FlexDirection")
        s.position = str("Position")
        s.lineHeight = int("LineHeight")
        s.accessibilityLabel = str("AccessibilityLabel")
        s.accessibilityHint = str("AccessibilityHint")
        s.accessibilityHidden = obj["AccessibilityHidden"] as? Bool ?? false
        s.accessibilityRole = str("AccessibilityRole")
        s.disabled = obj["Disabled"] as? Bool ?? false
        s.transition = str("Transition")
        return s
    }

    /// Go's EdgeInsets carries per-side values plus Horizontal/Vertical
    /// shorthands; the shorthand fills any side not set explicitly, which
    /// matches how the DSL's PaddingHorizontal-style helpers are used.
    private static func parseEdges(_ obj: [String: Any]?) -> Edges {
        guard let obj else { return .zero }
        func int(_ key: String) -> Int { (obj[key] as? NSNumber)?.intValue ?? 0 }
        let h = int("Horizontal"), v = int("Vertical")
        func side(_ name: String, _ shorthand: Int) -> Int {
            let explicit = int(name)
            return explicit != 0 ? explicit : shorthand
        }
        return Edges(top: side("Top", v), right: side("Right", h),
                     bottom: side("Bottom", v), left: side("Left", h))
    }

    /// Accepts CSS-style #RGB, #RRGGBB, and #RRGGBBAA (Go emits the latter two).
    static func parseColor(_ hex: String?) -> Color? {
        guard let hex, hex.hasPrefix("#") else { return nil }
        let s = String(hex.dropFirst())
        guard let v = UInt64(s, radix: 16) else { return nil }
        let r, g, b, a: Double
        switch s.count {
        case 3:
            r = Double((v >> 8) & 0xF) * 17
            g = Double((v >> 4) & 0xF) * 17
            b = Double(v & 0xF) * 17
            a = 255
        case 6:
            r = Double((v >> 16) & 0xFF)
            g = Double((v >> 8) & 0xFF)
            b = Double(v & 0xFF)
            a = 255
        case 8: // CSS byte order: RRGGBBAA — alpha last
            r = Double((v >> 24) & 0xFF)
            g = Double((v >> 16) & 0xFF)
            b = Double((v >> 8) & 0xFF)
            a = Double(v & 0xFF)
        default:
            return nil
        }
        return Color(.sRGB, red: r / 255, green: g / 255, blue: b / 255, opacity: a / 255)
    }
}

/// Per-axis "fill the space you were given" flags, computed by a flex parent
/// for each child.
///
/// Two separate axes rather than one enum because the two reasons a child
/// fills are independent and can both apply at once: **main**-axis fill comes
/// from FlexGrow (the child was allotted a slice of the leftover space by
/// GrMobFlexStack), and **cross**-axis fill comes from the container's
/// `AlignItems: "stretch"`. A grown, stretched child in a Row fills both.
///
/// The layout arithmetic itself lives in GrMobFlexStack — it proposes each
/// child an exact size. This modifier is the other half of that handshake:
/// a proposal is only an offer, and a Text or a styled Box would take its
/// ideal size and leave the rest of the region empty (with its background
/// unpainted) without a flexible frame telling it to accept.
struct GrMobGrow: Equatable {
    var fillWidth = false
    var fillHeight = false
    /// A floor rather than a fill: the child is at least this tall and may
    /// be taller. Zero means no floor. It exists for one parent, Scroll,
    /// whose main axis is unbounded — there is no leftover space for
    /// FlexGrow to claim, so a grow child inside it is given the viewport
    /// height as a minimum instead (see GrMobScroll), the same thing the
    /// DOM does for `flex-grow` under `overflow: auto` and Compose does
    /// with heightIn(min = viewport).
    var minHeight: CGFloat = 0

    static let none = GrMobGrow()
    static let horizontal = GrMobGrow(fillWidth: true)
    static let vertical = GrMobGrow(fillHeight: true)

    /// Adds `other`'s axes to this one — how a container combines a child's
    /// main-axis growth with its own cross-axis stretch.
    func union(_ other: GrMobGrow) -> GrMobGrow {
        GrMobGrow(fillWidth: fillWidth || other.fillWidth,
                  fillHeight: fillHeight || other.fillHeight,
                  minHeight: max(minHeight, other.minHeight))
    }
}

/// Event dispatch for gesture-bearing boxes, injected as a plain closure so
/// this file stays free of GrMobRuntime — the verify harness compiles the
/// style/node/store layer without Renderer.swift or the runtime, and an
/// EnvironmentKey referencing the runtime type would drag them in.
/// GrMobRoot fills it in from the live runtime.
private struct GrMobDispatchKey: EnvironmentKey {
    static let defaultValue: (@MainActor (String) -> Void)? = nil
}

extension EnvironmentValues {
    var grMobDispatch: (@MainActor (String) -> Void)? {
        get { self[GrMobDispatchKey.self] }
        set { self[GrMobDispatchKey.self] = newValue }
    }
}

/// Tap/long-press wiring for nodes that don't draw their own control (Button
/// and the inputs handle their own interaction). Inserted into grMobBox
/// after the background layer, so the touch target is the visible box —
/// padding included, margin excluded — matching the Android renderer.
/// The accessibility actions mirror the gestures so a VoiceOver user can
/// activate a row (and reach its long-press action by name) without touch.
private struct GrMobGestures: ViewModifier {
    let onTap: String
    let onLongPress: String
    let disabled: Bool
    @Environment(\.grMobDispatch) private var dispatch

    func body(content: Content) -> some View {
        // A disabled node keeps its callback IDs — Go still has the handlers
        // registered (see core.Style.Disabled) — and simply stops recognizing
        // gestures, which also removes the accessibility actions below so
        // VoiceOver stops offering an activation that would do nothing.
        // `.disabled(true)` in grMobBox handles the controls; this handles
        // the plain boxes and rows, which draw no control of their own.
        if disabled || (onTap.isEmpty && onLongPress.isEmpty) {
            content
        } else {
            content
                // Transparent regions of the box must still hit-test.
                .contentShape(Rectangle())
                .grMobOnTap(onTap, dispatch)
                .grMobOnLongPress(onLongPress, dispatch)
        }
    }
}

extension View {
    @ViewBuilder fileprivate func grMobOnTap(
        _ id: String, _ dispatch: (@MainActor (String) -> Void)?
    ) -> some View {
        if id.isEmpty {
            self
        } else {
            onTapGesture { dispatch?(id) }
                .accessibilityAddTraits(.isButton)
                .accessibilityAction { dispatch?(id) }
        }
    }

    @ViewBuilder fileprivate func grMobOnLongPress(
        _ id: String, _ dispatch: (@MainActor (String) -> Void)?
    ) -> some View {
        if id.isEmpty {
            self
        } else {
            onLongPressGesture { dispatch?(id) }
                .accessibilityAction(named: Text("Long press")) { dispatch?(id) }
        }
    }
}

extension View {
    /// Applies this node's box styling in CSS box-model order. CSS lists the
    /// layers outermost-first (margin → size → shadow → clip → background →
    /// border → padding) while SwiftUI modifier chains read innermost-first,
    /// so the chain below is that list reversed — and the order is
    /// load-bearing: background before clipShape would leave square corners
    /// painted, padding after background would paint outside the box, etc.
    ///
    /// `onTap`/`onLongPress` are the node's gesture callback IDs (empty when
    /// absent); see GrMobGestures for where they sit in the layer order.
    ///
    /// `axis` is set by the flex containers (Row: horizontal, Column and
    /// List: vertical) and nil for everything else; it only decides how the
    /// node's alignment styles map onto its fill frames (grMobFrameAlignment).
    func grMobBox(
        _ s: GrMobStyle?, grow: GrMobGrow = .none,
        onTap: String = "", onLongPress: String = "", axis: Axis? = nil
    ) -> some View {
        let shape = RoundedCornerShapeIfAny(radius: s?.borderRadius ?? 0)
        let alignment = grMobFrameAlignment(s, axis: axis)
        return self
            .padding((s?.padding ?? .zero).insets)
            .background(s?.background ?? .clear)
            .modifier(GrMobGestures(onTap: onTap, onLongPress: onLongPress,
                                    disabled: s?.disabled ?? false))
            .grMobClip(shape)
            .grMobBorder(shape, color: s?.borderColor, width: s?.borderWidth ?? 0)
            .grMobShadow(s?.shadow ?? 0)
            .grMobDimension(s?.width ?? "", axis: .horizontal, alignment: alignment)
            .grMobDimension(s?.height ?? "", axis: .vertical, alignment: alignment)
            .padding((s?.margin ?? .zero).insets)
            .grMobGrow(grow, alignment: alignment)
            // "hidden" keeps the node's space but not its pixels ("none" is
            // handled earlier by not rendering the node at all — see RenderNode).
            .opacity(s?.display == "hidden" ? 0 : 1)
            // The platform disabled state. SwiftUI's `.disabled` propagates
            // down the subtree, which is deliberate and is what the Android
            // renderer's LocalGrMobDisabled and the web target's
            // pointer-events:none reproduce: one Go flag on a container means
            // the same thing on all three. It also carries the accessibility
            // half — VoiceOver announces the control as dimmed — which no
            // amount of styling can fake. Applied unconditionally because
            // `.disabled(false)` is the identity case, and a @ViewBuilder
            // branch here would add another _ConditionalContent layer to this
            // chain (see grMobTransition for what that costs).
            .disabled(s?.disabled ?? false)
            .grMobAccessibility(s)
            .grMobRole(s)
            .grMobTransition(s)
    }

    /// The property-change half of Transition support: when the style
    /// declares one, any style change on this node (an update-style patch —
    /// background, text color, padding, opacity, explicit size) animates
    /// with the declared curve instead of snapping. Outermost in the chain
    /// so every animatable modifier below it is covered; scoped by
    /// `value: s` so unrelated tree changes never trigger it. A `replace`
    /// patch swaps the node instance, so replaced nodes snap — matching the
    /// Go reconciler's intent (replace = a different thing, not a changed
    /// one).
    ///
    /// Deliberately NOT a @ViewBuilder conditional: `.animation(nil, ...)`
    /// is already the no-animation case, and one more _ConditionalContent
    /// layer on top of grMobBox's opaque-type tower crashes the Swift
    /// compiler ("non-terminating conformance substitution" in
    /// substOpaqueTypesWithUnderlyingTypes).
    fileprivate func grMobTransition(_ s: GrMobStyle?) -> some View {
        animation(s?.swiftUIAnimation, value: s)
    }

    /// Accessibility semantics from the Go style. Hidden wins and prunes the
    /// whole subtree. A label on a container collapses its children into one
    /// accessibility element (the feed-row pattern: one swipe stop per row,
    /// announced by the label) — leaves are single elements already, so the
    /// combine is a no-op for them.
    @ViewBuilder fileprivate func grMobAccessibility(_ s: GrMobStyle?) -> some View {
        if s?.accessibilityHidden == true {
            accessibilityHidden(true)
        } else if let s, !s.accessibilityLabel.isEmpty {
            accessibilityElement(children: .combine)
                .accessibilityLabel(s.accessibilityLabel)
                .grMobA11yHint(s.accessibilityHint)
        } else if let s, !s.accessibilityHint.isEmpty {
            grMobA11yHint(s.accessibilityHint)
        } else {
            self
        }
    }

    @ViewBuilder fileprivate func grMobA11yHint(_ hint: String) -> some View {
        if hint.isEmpty { self } else { accessibilityHint(hint) }
    }

    /// The role half of the semantics, as traits.
    ///
    /// A separate step in the chain rather than another branch inside
    /// grMobAccessibility, and applied unconditionally: an empty
    /// AccessibilityTraits is the identity case, so this costs nothing on the
    /// several hundred nodes of a tree that have no role, and one more
    /// _ConditionalContent layer on grMobBox's opaque-type tower is a real
    /// cost (see grMobTransition, which is written the way it is for exactly
    /// that reason).
    ///
    /// Hidden still wins: a pruned subtree has no element for a trait to
    /// describe, which is the same exclusive choice grMobAccessibility and
    /// Compose's clearAndSetSemantics make.
    fileprivate func grMobRole(_ s: GrMobStyle?) -> some View {
        guard let s, !s.accessibilityHidden else { return accessibilityAddTraits([]) }
        return accessibilityAddTraits(grMobTraitsFor(s.accessibilityRole))
    }

    /// Conditional label for the Image "alt" fallback (internal because the
    /// Image case in Renderer.swift decides whether the fallback applies).
    @ViewBuilder func grMobAltLabel(_ label: String) -> some View {
        if label.isEmpty { self } else { accessibilityLabel(label) }
    }

    @ViewBuilder fileprivate func grMobClip(_ shape: RoundedRectangle?) -> some View {
        // Clipping is strictly conditional: a radius-0 clipShape would still
        // cut off child overflow (e.g. shadows), which un-clipped boxes allow.
        if let shape { clipShape(shape) } else { self }
    }

    @ViewBuilder fileprivate func grMobBorder(_ shape: RoundedRectangle?, color: Color?, width: CGFloat) -> some View {
        if let color, width > 0 {
            // strokeBorder insets the stroke fully inside the shape — the
            // Compose Modifier.border behavior — where a plain stroke would
            // straddle the edge and get half clipped away.
            overlay((shape ?? RoundedRectangle(cornerRadius: 0)).strokeBorder(color, lineWidth: width))
        } else {
            self
        }
    }

    @ViewBuilder fileprivate func grMobShadow(_ radius: CGFloat) -> some View {
        if radius > 0 {
            // compositingGroup flattens the subtree first so the shadow wraps
            // the box as a whole; without it SwiftUI shadows every opaque
            // pixel individually (each text glyph gets its own halo).
            compositingGroup().shadow(radius: radius / 2, y: radius / 3)
        } else {
            self
        }
    }

    /// Maps a Go dimension string onto a frame. Supported forms: "120px" or a
    /// bare number (points), "100%" (fill the parent), other percentages
    /// (fraction of the nearest container — an approximation of
    /// fraction-of-parent, which SwiftUI cannot express without a
    /// GeometryReader), and ""/"auto" (intrinsic size, no frame).
    @ViewBuilder fileprivate func grMobDimension(
        _ value: String, axis: Axis, alignment: Alignment = .topLeading
    ) -> some View {
        if value.isEmpty || value == "auto" {
            self
        } else if value == "100%" {
            switch axis {
            case .horizontal: frame(maxWidth: .infinity, alignment: alignment)
            case .vertical: frame(maxHeight: .infinity, alignment: alignment)
            }
        } else if value.hasSuffix("%"), let pct = Double(value.dropLast()) {
            containerRelativeFrame(axis == .horizontal ? .horizontal : .vertical) { length, _ in
                length * min(max(pct / 100, 0), 1)
            }
        } else if let number = Double(value.hasSuffix("px") ? String(value.dropLast(2)) : value) {
            switch axis {
            case .horizontal: frame(width: CGFloat(number))
            case .vertical: frame(height: CGFloat(number))
            }
        } else {
            self
        }
    }

    /// Kept strictly conditional: the no-fill case must add no frame at all,
    /// or every leaf in the tree would gain a layout container that changes
    /// how it reports its own ideal size.
    ///
    /// `alignment` is where the content sits when the frame is bigger than
    /// it, which for a fill frame is the usual case; see grMobFrameAlignment
    /// for why that is not left to SwiftUI's default.
    @ViewBuilder func grMobGrow(_ grow: GrMobGrow, alignment: Alignment = .topLeading) -> some View {
        if grow == .none {
            self
        } else {
            frame(maxWidth: grow.fillWidth ? .infinity : nil,
                  minHeight: grow.minHeight > 0 ? grow.minHeight : nil,
                  maxHeight: grow.fillHeight ? .infinity : nil,
                  alignment: alignment)
        }
    }
}

/// Where a node's content sits inside a frame larger than the content — the
/// flexible frames grMobGrow and grMobDimension("100%") add.
///
/// SwiftUI's `frame` centres by default, which is the wrong default for a
/// box model: a FlexGrow(1) title box in a header row put its text in the
/// middle of the row, and the FlexGrow(1) content box of a screen floated
/// short content to the vertical middle of the window. Compose's equivalents
/// (fillMaxWidth on a Box, weight on a Column child) keep content at the
/// top-start, so top-leading is the default here.
///
/// The exceptions follow the node's own alignment styles, because the flex
/// layouts hug their children (GrMobFlexSolver.containerMain and the cross
/// size in GrMobFlexLayout): a `Column(AlignItems(center), Width("100%"))`
/// is a content-wide layout inside a screen-wide frame, and its children are
/// centred within the layout, so the frame has to centre the layout or the
/// AlignItems is invisible. Compose has the same two steps (fillMaxWidth on
/// the Column, horizontalAlignment for the children) and they agree by
/// construction; here the frame is told what the layout was told.
///
///   - Column/List (vertical): horizontal from AlignItems, with Align as the
///     fallback exactly as the layout reads it (see crossAxisValue in
///     Renderer.swift); vertical is top.
///   - Row (horizontal): vertical from AlignItems, the Row's cross axis;
///     horizontal from Align, which is the leading edge unless the app set
///     one — a hugging Row is packed to the start, as flex-start is.
///   - Everything else: horizontal from Align, the DSL's text/content
///     alignment, which keeps `Text(.., Width("100%"), Align(AlignCenter))`
///     centred as it is on every other target; vertical is top.
///
/// "stretch" is not a placement and maps to leading/top: a stretched node
/// already fills the extent there is to place it in.
func grMobFrameAlignment(_ s: GrMobStyle?, axis: Axis? = nil) -> Alignment {
    let align = s?.align ?? ""
    let items = s?.alignItems ?? ""
    let horizontal: String
    let vertical: String
    switch axis {
    case .vertical:
        horizontal = items.isEmpty ? align : items
        vertical = ""
    case .horizontal:
        horizontal = align
        vertical = items
    case nil:
        horizontal = align
        vertical = ""
    }
    let h: HorizontalAlignment
    switch horizontal {
    case "center": h = .center
    case "end", "flex-end": h = .trailing
    default: h = .leading
    }
    let v: VerticalAlignment
    switch vertical {
    case "center": v = .center
    case "end", "flex-end": v = .bottom
    default: v = .top
    }
    return Alignment(horizontal: h, vertical: v)
}

private func RoundedCornerShapeIfAny(radius: CGFloat) -> RoundedRectangle? {
    radius > 0 ? RoundedRectangle(cornerRadius: radius) : nil
}

/// Maps one core.Role onto SwiftUI accessibility traits.
///
/// Four of the fifteen roles land on a trait; the other eleven are spelled out
/// anyway. SwiftUI's AccessibilityTraits is a small set about *controls* —
/// button, link, image, search field, header — and has no landmarks at all
/// (VoiceOver's rotor navigates by heading, not by banner) and no tabular
/// vocabulary, so nine of those eleven have nothing here to be mapped onto;
/// the remaining two are the live regions, which SwiftUI states imperatively
/// rather than as a property of the view (see the arm below).
/// Listing them is what keeps that a decision rather than an oversight: a
/// `default:` that swallowed them would look exactly like a role nobody had
/// taught this renderer about, which is the failure grMobScaled's ContentMode
/// arms already exist to prevent.
///
/// A column header is announced as a header, which is the nearest true thing
/// this platform can say about it — VoiceOver has one notion of heading.
///
/// One arm per line, string literals first, `default:` last: mobile/verify's
/// TestSwiftTraitsCoverEveryRole reads these arms out of the source and holds
/// them against core.Roles().
private func grMobTraitsFor(_ role: String) -> AccessibilityTraits {
    switch role {
    case "heading", "columnheader": .isHeader
    case "button": .isButton
    case "search": .isSearchField
    // No SwiftUI trait names these.
    case "table", "rowgroup", "row", "cell": []
    case "list", "listitem": []
    case "banner", "navigation", "toolbar": []
    // Nor these: SwiftUI announces a change through
    // AccessibilityNotification, which is an imperative call at the moment of
    // the change and not a property of the view that changed.
    case "status", "alert": []
    default: []
    }
}
