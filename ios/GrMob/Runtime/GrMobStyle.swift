import SwiftUI

/// Swift mirror of Go's core.Style, decoded from the tree/patch JSON.
///
/// Field names in the JSON are the Go struct's exported names verbatim
/// ("FontSize", "TextColor", ...) because core.Style's json tags set no names
/// — every tag on it is `,omitzero`, which changes what is *present* and never
/// what a present field is called. Every read below defaults a missing key to
/// the zero value, so an omitted field and a field written as zero decode
/// alike; that equivalence is the contract those tags rely on, and it predates
/// them.
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

    /// Go's core.ValueRange: where a valued control sits inside its range.
    ///
    /// The three numbers are carried as the strings they arrive as, unparsed,
    /// and that is the shape of what this platform can do with them rather
    /// than laziness — see grMobValueText for the whole argument. `text` is
    /// the member with somewhere to go.
    struct ValueRange: Equatable {
        var now = "", min = "", max = "", text = ""
    }

    var fontSize: CGFloat = 0
    var fontWeight: Int = 0
    var textColor: Color?
    var background: Color?
    var padding: Edges = .zero

    /// How far a box's content starts inside its painted edge: the padding,
    /// plus the border's width on every side when a border is drawn.
    ///
    /// CSS's border-box. On the web a 2px border pushes a box's children 2px
    /// in; grMobBorder is an overlay that only paints, so without the extra
    /// inset a child at the top edge was drawn under the stroke. comps.Spinner
    /// showed it: the orbiting dot overlapped the ring's rim instead of
    /// sitting just inside it (and on Compose, whose Modifier.border also
    /// reserves nothing, a round clip cut it in half — the same fix is in
    /// GrMobStyle.kt's boxModifier).
    ///
    /// The same two-part guard grMobBorder uses, so a width with no colour,
    /// which paints nothing, moves nothing either.
    var contentInsets: EdgeInsets {
        let b = borderInset
        return EdgeInsets(top: CGFloat(padding.top) + b, leading: CGFloat(padding.left) + b,
                          bottom: CGFloat(padding.bottom) + b, trailing: CGFloat(padding.right) + b)
    }

    /// The border's width when one is drawn, else zero. See contentInsets.
    var borderInset: CGFloat {
        borderColor != nil && borderWidth > 0 ? borderWidth : 0
    }
    var margin: Edges = .zero
    var borderRadius: CGFloat = 0
    var shadow: CGFloat = 0
    /// core.Rotate: clockwise degrees about the node's own centre. A paint
    /// transform, not a layout one — `.rotationEffect` turns the rendered view
    /// and leaves the frame it reported to its parent alone, matching CSS
    /// `transform` and Compose's `Modifier.rotate`. Carried unnormalised; see
    /// core.Style.Rotate for why the winding is the caller's to choose.
    var rotate: CGFloat = 0
    /// core.Spin: milliseconds per revolution, negative for anticlockwise, 0
    /// for still. Applied by GrMobSpin beside `rotate`.
    var spin: Int = 0
    /// core.Translate, one axis each, as GrMobShift.parse resolves it. Applied
    /// by GrMobTranslate just outside the two rotations.
    var translateX: GrMobShift = .zero
    var translateY: GrMobShift = .zero
    /// core.Overflow. Only "hidden" is read, as a clip to the box (see
    /// RoundedCornerShapeIfAny): it keeps a child translated out of its parent
    /// (a Drawer's shut panel) from drawing over what sits beside the parent.
    var overflow: String = ""
    var align: String = ""
    var display: String = ""
    var width: String = ""
    /// core.MaxWidth: CSS `max-width`, in the forms GrMobMaxWidth.limit reads.
    /// Applied by GrMobMaxWidthLayout at the outside of grMobBox's chain, and
    /// folded into grMobDimension when a rigid Width would otherwise ignore it.
    var maxWidth: String = ""
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

    /// core.Style.FlexShrink, as written — which is NOT the shrink factor.
    ///
    /// Zero means "unset" here, as it does for every other number in a
    /// core.Style, and flex-shrink is the one property whose CSS initial value
    /// is not zero. So the Go side spells a factor of zero as core.ShrinkNone
    /// (-1) and this field carries that verbatim; `shrinkFactor` is the
    /// reading. Storing the raw number rather than the reading keeps this
    /// struct a decode of the JSON and puts the one rule in one place.
    var flexShrink: CGFloat = 0

    /// The shrink factor this style asks for: 1 when nothing was set (the CSS
    /// initial value), 0 for core.ShrinkNone, and the number otherwise.
    ///
    /// The mirror of core.Style.ShrinkFactor, and the only place in this
    /// runtime that knows what -1 means.
    var shrinkFactor: CGFloat {
        if flexShrink == 0 { return 1 }
        if flexShrink == -1 { return 0 }
        return flexShrink
    }

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
    /// Go's core.StackAlignment, verbatim: "top-start", "bottom", ... or ""
    /// for a layer that takes the stack's centre. Read by GrMobZStack alone,
    /// through grMobStackAnchor in GrMobStack.swift — it is a *layer*
    /// property, and a node that is not a layer of an overlay has no use for
    /// it. (The mapping used to live here and return a SwiftUI Alignment; it
    /// moved to the pure file when the stack stopped placing layers with a
    /// frame, so that ios/verify could measure it.)
    var stackAlign: String = ""
    var lineHeight: Int = 0
    var accessibilityLabel: String = ""
    var accessibilityHint: String = ""
    var accessibilityHidden: Bool = false
    /// Go's core.Role, verbatim; mapped to traits by grMobTraitsFor below.
    var accessibilityRole: String = ""
    /// Go's core.Style.AccessibilityHeadingLevel: 1-6, or 0 for a heading that
    /// does not state its tier. Read only when the role is "heading"; see
    /// grMobHeadingLevel below.
    var accessibilityHeadingLevel: Int = 0
    /// Go's core.SelectedState, verbatim: "true", "false", or "" for a node
    /// that makes no claim. Mapped to a trait by grMobSelectedTrait below.
    var accessibilitySelected: String = ""
    /// Go's core.CurrentKind, verbatim: "page", "step", "true", or "" for a
    /// node that is not the current item of a set. Folded into .isSelected by
    /// grMobCurrentTrait below.
    var accessibilityCurrent: String = ""
    /// Go's core.ValueRange, verbatim: where a valued control sits inside its
    /// range. Only `text` is read — see grMobValueText below and the note on
    /// the three numbers this platform cannot say.
    var accessibilityValue: ValueRange = ValueRange()
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
        s.rotate = num("Rotate")
        s.spin = int("Spin")
        s.translateX = GrMobShift.parse(str("TranslateX"))
        s.translateY = GrMobShift.parse(str("TranslateY"))
        s.overflow = str("Overflow")
        s.align = str("Align")
        s.display = str("Display")
        s.width = str("Width")
        s.maxWidth = str("MaxWidth")
        s.height = str("Height")
        s.borderColor = parseColor(str("BorderColor"))
        s.borderWidth = num("BorderWidth")
        s.gap = num("Gap")
        s.rowGap = num("RowGap")
        s.columnGap = num("ColumnGap")
        s.justifyContent = str("JustifyContent")
        s.alignItems = str("AlignItems")
        s.flexGrow = num("FlexGrow")
        s.flexShrink = num("FlexShrink")
        s.flexWrap = str("FlexWrap")
        s.flexDirection = str("FlexDirection")
        s.position = str("Position")
        s.stackAlign = str("StackAlign")
        s.lineHeight = int("LineHeight")
        s.accessibilityLabel = str("AccessibilityLabel")
        s.accessibilityHint = str("AccessibilityHint")
        s.accessibilityHidden = obj["AccessibilityHidden"] as? Bool ?? false
        s.accessibilityRole = str("AccessibilityRole")
        s.accessibilityHeadingLevel = int("AccessibilityHeadingLevel")
        s.accessibilitySelected = str("AccessibilitySelected")
        s.accessibilityCurrent = str("AccessibilityCurrent")
        s.accessibilityValue = parseValueRange(obj["AccessibilityValue"] as? [String: Any])
        s.disabled = obj["Disabled"] as? Bool ?? false
        s.transition = str("Transition")
        return s
    }

    /// Go's core.ValueRange. Absent and all-empty both mean "not a valued
    /// control": the first is a Style that never set one, the second is one
    /// whose range went back to its zero value.
    private static func parseValueRange(_ obj: [String: Any]?) -> ValueRange {
        guard let obj else { return ValueRange() }
        func str(_ key: String) -> String { obj[key] as? String ?? "" }
        return ValueRange(now: str("Now"), min: str("Min"),
                          max: str("Max"), text: str("Text"))
    }

    /// Go's EdgeInsets carries per-side values plus Horizontal/Vertical
    /// shorthands; the shorthand fills any side not set explicitly, which
    /// matches how the DSL's PaddingHorizontal-style helpers are used.
    ///
    /// Every one of the six is `json:",omitzero"` on the Go side, so a key
    /// that is absent here is a field that was zero there. Nothing below needs
    /// to change for that, and the reason is the `explicit != 0` test: this
    /// function has always defined a zero side as "unset, take the axis", so
    /// "absent" and "present and zero" were already the same question. The
    /// `?? 0` is what makes them the same answer — do not replace it with an
    /// optional thinking it recovers a distinction, because Go no longer sends
    /// one.
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

/// core.Spin: the box turns one revolution every `periodMs`, forever.
///
/// # Why TimelineView and not repeatForever
///
/// The textbook spinner — a @State angle flipped to 360 in onAppear under
/// `.linear.repeatForever(autoreverses: false)` — attaches the repeating
/// animation to a transaction, and every other change committed in that
/// transaction's scope rides it too: a patch that moves or resizes the node
/// would start repeating as well. It also restarts from wherever SwiftUI's
/// presentation value happens to be when the view is re-identified. A
/// TimelineView owns no animation at all: each frame the angle is recomputed
/// from the timeline's date, modulo the period, so it cannot leak into other
/// changes and cannot drift or accumulate. The same elapsed-time rule is what
/// the Compose and CSS mappings follow.
///
/// # Always applied, paused when still
///
/// Not a conditional wrapper, for the reason `grMobRotate` gives, and for one
/// more that is specific to this modifier: the TimelineView is a container,
/// and wrapping a node in one only while it spins would change the node's
/// structural identity the moment Spin flips — resetting @State in every view
/// beneath it (a focused text field would lose its text). A paused schedule
/// never ticks, so a still node evaluates the closure once and draws at 0°,
/// the identity rotation.
///
/// Unmeasured: what a TimelineView per node costs across a long tree. The
/// alternative, a conditional, costs correctness instead.
struct GrMobSpin: ViewModifier {
    let periodMs: Int

    func body(content: Content) -> some View {
        TimelineView(.animation(minimumInterval: nil, paused: periodMs == 0)) { timeline in
            content.rotationEffect(.degrees(angle(at: timeline.date)), anchor: .center)
        }
    }

    /// Degrees at `date`: the fraction of the current period elapsed since
    /// the reference date, times 360, negated for an anticlockwise spin.
    /// Measured from a fixed epoch rather than from appearance, so two
    /// spinners on one screen turn in step.
    private func angle(at date: Date) -> Double {
        guard periodMs != 0 else { return 0 }
        let period = Double(abs(periodMs)) / 1000
        let fraction = date.timeIntervalSinceReferenceDate
            .truncatingRemainder(dividingBy: period) / period
        return fraction * 360 * (periodMs < 0 ? -1 : 1)
    }
}

/// The box chain as a named modifier rather than as a chain of generic
/// `some View` extensions applied at each call site.
///
/// # Why this is a struct and not simply the chain
///
/// Every `.grMobXxx` below returns an opaque `some View`, and most of them
/// are `@ViewBuilder`s, so each one roughly doubles the size of the type it
/// produces. Written as a chain returning `some View`, that tower is
/// re-derived for every `Self` it is ever applied to — one instantiation per
/// view type in the renderer.
///
/// A Debug build never notices, because opaque types declared in another
/// file are left abstract when the module is compiled file-by-file. A
/// **Release** build enables whole-module optimisation, which substitutes
/// every opaque type with its underlying type — and the substitution of that
/// tower aborts the Swift compiler outright:
///
///     Abort: function substOpaqueTypesWithUnderlyingTypes
///            at SubstitutionMap.cpp:651
///     Possible non-terminating type substitution detected
///     While silgen emitFunction SIL function "$s8GrMobApp0aB7MapViewV4bodyQrvg"
///
/// (GrMobMapView is merely the first file SILGen reaches; every view in
/// Renderer.swift carries the same tower.)
///
/// A `ViewModifier` struct is not generic over the view it wraps, so the
/// tower is built exactly once — inside this one `body(content:)`, over the
/// single fixed type `_ViewModifier_Content<GrMobBoxModifier>` — and every
/// call site gets the shallow, concrete `ModifiedContent<Self,
/// GrMobBoxModifier>` instead. Nothing about the rendering changes: the
/// modifier order below is the chain verbatim.
struct GrMobBoxModifier: ViewModifier {
    let style: GrMobStyle?
    let grow: GrMobGrow
    let onTap: String
    let onLongPress: String
    let axis: Axis?

    /// The system's Reduce Motion setting. Read here, once per box, rather
    /// than folded into GrMobStyle.swiftUIAnimation, because the style is
    /// parsed from the wire and has no environment; reading it in the view
    /// also means toggling the setting re-evaluates the chain without a patch.
    /// Only Transition consults it: GrMobSpin keeps turning (see core.Spin,
    /// "Reduced motion: it keeps turning").
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    func body(content: Content) -> some View {
        // Bound once so the chain below reads exactly as it did when it was
        // a chain of extensions on `self`.
        let s = style
        let shape = RoundedCornerShapeIfAny(radius: s?.borderRadius ?? 0,
                                            clips: s?.overflow == "hidden")
        let alignment = grMobFrameAlignment(s, axis: axis)
        return content
            // Padding plus the border's width: see GrMobStyle.contentInsets.
            .padding(s?.contentInsets ?? EdgeInsets())
            // The explicit Width and Height sit directly outside the padding
            // and INSIDE the background, clip, border, shadow and gestures —
            // CSS's border-box, and the order Compose's boxModifier already
            // has (dimension, then clip/background/border, then padding).
            //
            // They used to come after the shadow, and that was invisible for
            // any box whose content filled its frame. It was not for a box
            // that hugs smaller content: GrMobFlexLayout reports the size of
            // its children (zero for none), so the fill and the stroke were
            // drawn around that and the fixed frame around them held nothing
            // visible. comps.Spinner was the case that showed it — a 24pt
            // ring drawn as a hollow speck, and its childless 6pt dot drawn
            // not at all.
            //
            // The same move puts the touch target and the shadow on the
            // declared box rather than on its content, which is again what
            // the DOM and Compose do.
            //
            // The cap is passed in as well as applied outside (below): a Width
            // in points or a percentage is a rigid frame, which reports its
            // own size whatever it is proposed, so only folding the cap into
            // the frame itself gives CSS's min(width, max-width).
            .grMobDimension(s?.width ?? "", axis: .horizontal, alignment: alignment,
                            cap: GrMobMaxWidth.fixedLimit(s?.maxWidth ?? ""))
            .grMobDimension(s?.height ?? "", axis: .vertical, alignment: alignment)
            .background(s?.background ?? .clear)
            .modifier(GrMobGestures(onTap: onTap, onLongPress: onLongPress,
                                    disabled: s?.disabled ?? false))
            .grMobClip(shape)
            .grMobBorder(shape, color: s?.borderColor, width: s?.borderWidth ?? 0)
            .grMobShadow(s?.shadow ?? 0)
            // Rotation wraps the whole painted box — padding, the explicit
            // frame, background, gestures, corner clip, border and shadow —
            // and is applied before the margin, which is the CSS rule: a
            // transform turns the border box about its own centre and leaves
            // the space reserved around it axis-aligned. Rotating after the
            // margin would swing an asymmetrically-spaced node about a point
            // that is not its centre.
            //
            // SwiftUI hit-tests through a rotationEffect, so the gesture
            // modifier further in keeps a touch target that turns with the
            // pixels rather than staying square.
            .grMobRotate(s?.rotate ?? 0)
            // core.Spin, just outside the fixed angle and inside the margin,
            // for the same reasons. Rotations about one centre commute, so
            // the order against grMobRotate does not change the pixels.
            // core.Spin and then core.Translate, as one modifier (see
            // GrMobMotion for why one and not two).
            .modifier(GrMobMotion(spinMs: s?.spin ?? 0,
                                  translateX: s?.translateX ?? .zero,
                                  translateY: s?.translateY ?? .zero))
            .padding((s?.margin ?? .zero).insets)
            .grMobGrow(grow, alignment: alignment)
            // core.MaxWidth, outermost of the sizing layers and after
            // grMobGrow on purpose. A stretched or FlexGrow child carries a
            // flexible frame from grMobGrow that accepts whatever it is
            // proposed; capping the proposal OUTSIDE that frame is what makes
            // a stretched child fill min(extent, cap) — CSS's stretch clamped
            // by max-width — rather than fill the extent and draw its capped
            // content somewhere inside it. The margin is inside this layer,
            // so it is added back to the cap: CSS's max-width limits the
            // border box, and the space reserved around it is extra.
            .modifier(GrMobMaxWidthModifier(value: s?.maxWidth ?? "",
                                            margin: grMobHorizontalMargin(s)))
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
            .grMobValueText(s)
            .grMobTransition(s, reduceMotion: reduceMotion)
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
    ///
    /// Returns a concrete `ModifiedContent` rather than `some View`: see
    /// GrMobBoxModifier for the Release-build compiler crash that an opaque
    /// return type here reintroduces.
    func grMobBox(
        _ s: GrMobStyle?, grow: GrMobGrow = .none,
        onTap: String = "", onLongPress: String = "", axis: Axis? = nil
    ) -> ModifiedContent<Self, GrMobBoxModifier> {
        modifier(GrMobBoxModifier(style: s, grow: grow, onTap: onTap,
                                  onLongPress: onLongPress, axis: axis))
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
    ///
    /// Under Reduce Motion the Animation is nil, so every change snaps, which
    /// is core.Transition's "Reduced motion" rule on this target. Still the
    /// same single `.animation` modifier either way, for the reason above.
    fileprivate func grMobTransition(_ s: GrMobStyle?, reduceMotion: Bool) -> some View {
        animation(reduceMotion ? nil : s?.swiftUIAnimation, value: s)
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
                .accessibilityLabel(grMobCurrentLabel(s.accessibilityLabel, kind: s.accessibilityCurrent))
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
    ///
    /// The heading *level* rides along here rather than in a step of its own,
    /// and unconditionally, for the reason stated above and in
    /// grMobTransition: `.unspecified` is AccessibilityHeadingLevel's own
    /// default, so applying it to the several hundred non-heading nodes of a
    /// tree changes nothing, where a @ViewBuilder branch would add another
    /// _ConditionalContent layer to grMobBox's opaque-type tower. Both arms of
    /// the guard below take the same two modifiers so the return types match.
    ///
    /// The selected state rides along for the same reason and by the same
    /// mechanism: it *is* a trait on this platform, so it unions into the
    /// role's set rather than needing a modifier of its own.
    fileprivate func grMobRole(_ s: GrMobStyle?) -> some View {
        guard let s, !s.accessibilityHidden else {
            return accessibilityAddTraits([]).accessibilityHeading(.unspecified)
        }
        let traits = grMobTraitsFor(s.accessibilityRole)
            .union(grMobSelectedTrait(s.accessibilitySelected))
            .union(grMobCurrentTrait(s.accessibilityCurrent, selected: s.accessibilitySelected))
        return accessibilityAddTraits(traits)
            .accessibilityHeading(grMobHeadingLevel(s))
    }

    /// The spoken form of a valued control's position, as the accessibility
    /// value.
    ///
    /// # The one member of core.ValueRange this platform can say
    ///
    /// Go carries four: three numbers and the words. SwiftUI has no numeric
    /// accessibility value — `accessibilityValue` takes a Text and nothing
    /// else, and there is no trait, no ProgressBarRangeInfo, no equivalent of
    /// the range Compose gets — so `now`, `min` and `max` cross the bridge,
    /// are parsed into the struct so a reader of GrMobStyle can see they
    /// arrived, and reach no view modifier. mobile/verify/value_test.go pins
    /// that they do not.
    ///
    /// Turning them into a string here is the move this file has already
    /// turned down twice, for AccessibilityExpanded and for the `, selected`
    /// suffix comps.Chip used to append: a renderer that emits "45
    /// percent" is inventing English for every app in every locale, and the
    /// value slot belongs to the app besides.
    ///
    /// `text` is different in exactly that respect and that is why it is here.
    /// The words are the app's own — the same channel accessibilityLabel and
    /// accessibilityHint already ride — so passing them through invents
    /// nothing. It is also, incidentally, the value slot the
    /// AccessibilityExpanded note below says the framework has no room for: an
    /// app that wants VoiceOver to hear "expanded" can now say so in its own
    /// words, in its own language, which is a different thing from this
    /// renderer deciding to.
    ///
    /// # Unguarded by the role, as the selection is
    ///
    /// VoiceOver announces an accessibility value on any element, so scoping
    /// this to `progressbar` the way the two web exporters must would drop a
    /// value this platform would otherwise have spoken. ARIA is the strict one
    /// here; the framework is not stricter than the platform it is addressing.
    ///
    /// Hidden wins, as it does over the role and the traits: a pruned subtree
    /// has no element for a value to belong to. Applied unconditionally
    /// because an empty string is the identity case — accessibilityValue("")
    /// leaves the announcement alone — which keeps this off grMobBox's
    /// opaque-type tower, the same reason grMobRole is written the way it is.
    fileprivate func grMobValueText(_ s: GrMobStyle?) -> some View {
        let text = (s?.accessibilityHidden ?? true) ? "" : s?.accessibilityValue.text ?? ""
        return accessibilityValue(Text(text))
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

    /// Internal rather than fileprivate: GrMobButtonStyle in Renderer.swift
    /// needs the same stroke. A Button draws its own container, so it is handed
    /// a style stripped of the box-drawing fields (marginAndSizeOnly) and never
    /// reaches grMobBox — which is how core.BorderWidth/BorderColor came to be
    /// dropped on Buttons alone, and why the rule comps.Button's
    /// EmphasisOutlined documents drew on the web and not on device.
    @ViewBuilder func grMobBorder(_ shape: RoundedRectangle?, color: Color?, width: CGFloat) -> some View {
        if let color, width > 0 {
            // strokeBorder insets the stroke fully inside the shape — the
            // Compose Modifier.border behavior — where a plain stroke would
            // straddle the edge and get half clipped away.
            overlay((shape ?? RoundedRectangle(cornerRadius: 0)).strokeBorder(color, lineWidth: width))
        } else {
            self
        }
    }

    /// core.Rotate as a `.rotationEffect`, always applied.
    ///
    /// Not a `@ViewBuilder` branch, for the reason spelled out on `.disabled`
    /// in grMobBox: zero is the identity case, and a conditional would wrap
    /// every unrotated node in a `_ConditionalContent` layer to express it.
    /// `.rotationEffect` is a geometry transform rather than a compositing
    /// group, so the identity costs nothing to apply.
    ///
    /// `.center` is the default anchor and the only origin core.Rotate offers;
    /// it is named here so the agreement with CSS's `transform-origin: 50% 50%`
    /// and Compose's layout-bounds centre is visible rather than inherited.
    fileprivate func grMobRotate(_ degrees: CGFloat) -> some View {
        rotationEffect(.degrees(degrees), anchor: .center)
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
    ///
    /// `cap` is core.MaxWidth in points (GrMobMaxWidth.fixedLimit), horizontal
    /// only, and it clamps the two rigid arms. "100%" is left alone: it is a
    /// flexible frame, it takes what it is proposed, and GrMobMaxWidthLayout
    /// has already narrowed that proposal.
    @ViewBuilder fileprivate func grMobDimension(
        _ value: String, axis: Axis, alignment: Alignment = .topLeading, cap: CGFloat? = nil
    ) -> some View {
        if value.isEmpty || value == "auto" {
            self
        } else if value == "100%" {
            switch axis {
            case .horizontal: frame(maxWidth: .infinity, alignment: alignment)
            case .vertical: frame(maxHeight: .infinity, alignment: alignment)
            }
        } else if value.hasSuffix("%"), let pct = Double(value.dropLast()) {
            containerRelativeFrame(axis == .horizontal ? .horizontal : .vertical,
                                   alignment: alignment) { length, _ in
                GrMobMaxWidth.clamp(length * min(max(pct / 100, 0), 1),
                                    to: axis == .horizontal ? cap : nil)
            }
        } else if let number = Double(value.hasSuffix("px") ? String(value.dropLast(2)) : value) {
            // `alignment` on the fixed and percentage frames as well as the
            // fill one. Left off, SwiftUI centres content smaller than the
            // frame on both axes, where a CSS box and a Compose Column put it
            // at the top-start: a hugging Column (GrMobFlexLayout reports its
            // children's size) of fixed Height drew its first child in the
            // middle. comps.Spinner showed it — the dot that orbits the rim
            // sat at the ring's centre, where turning it moves nothing.
            switch axis {
            case .horizontal: frame(width: GrMobMaxWidth.clamp(CGFloat(number), to: cap),
                                    alignment: alignment)
            case .vertical: frame(height: CGFloat(number), alignment: alignment)
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

/// The box's clip and border shape: rounded when there is a radius, square
/// when the style asks for Overflow("hidden"), and nil otherwise, because
/// grMobClip clips to any shape it is given and an unasked clip would cut off
/// overflow (shadows, a translated child) that CSS's default visible allows.
/// A square shape changes nothing about the border, which strokes a
/// zero-radius rectangle when given nil anyway.
private func RoundedCornerShapeIfAny(radius: CGFloat, clips: Bool = false) -> RoundedRectangle? {
    if radius > 0 { return RoundedRectangle(cornerRadius: radius) }
    return clips ? RoundedRectangle(cornerRadius: 0) : nil
}

/// One axis of core.Translate: an amount in points and a fraction of the
/// view's own extent on that axis. Two numbers because the fraction resolves
/// only against a size, which GrMobTranslate is handed at draw time, and
/// because animating the pair linearly is how CSS interpolates "-100%" to none.
struct GrMobShift: Hashable {
    var points: CGFloat
    var fraction: CGFloat

    static let zero = GrMobShift(points: 0, fraction: 0)

    /// "Npx", a bare number (points) or "N%". Anything else is zero, as it is
    /// on the web targets and Compose.
    static func parse(_ value: String) -> GrMobShift {
        let v = value.trimmingCharacters(in: .whitespaces)
        if v.hasSuffix("%") {
            guard let pct = Double(v.dropLast().trimmingCharacters(in: .whitespaces)),
                  pct.isFinite else { return .zero }
            return GrMobShift(points: 0, fraction: CGFloat(pct / 100))
        }
        let number = v.hasSuffix("px") ? String(v.dropLast(2)) : v
        guard let n = Double(number.trimmingCharacters(in: .whitespaces)),
              n.isFinite else { return .zero }
        return GrMobShift(points: CGFloat(n), fraction: 0)
    }
}

/// core.Spin and core.Translate, applied as a single step of grMobBox's chain.
///
/// # Why one modifier
///
/// Written as two `.modifier(...)` calls, the chain gained one ModifiedContent
/// layer over what it had with Spin alone, and that one layer was enough to
/// abort the Swift compiler building ios/verify's harness ("Possible
/// non-terminating conformance substitution detected" in
/// substOpaqueTypesWithUnderlyingTypes), the same limit GrMobBoxModifier and
/// GrMobMaxWidthModifier exist to stay under. A ViewModifier struct is not
/// generic over its content, so the two effects compose inside this body over
/// one fixed type, and the box chain sees the single layer Spin used to be.
///
/// # Order
///
/// Translate outside Spin: CSS applies the individual `translate` before
/// `rotate` and `transform`, so a turned box slides along the screen's axes
/// rather than its own. Both sit outside grMobRotate and inside the margin.
/// Always applied, like GrMobSpin: zero is the identity transform, and a
/// conditional would add a _ConditionalContent layer to the chain.
struct GrMobMotion: ViewModifier {
    let spinMs: Int
    let translateX: GrMobShift
    let translateY: GrMobShift

    func body(content: Content) -> some View {
        content
            .modifier(GrMobSpin(periodMs: spinMs))
            .modifier(GrMobTranslate(x: translateX, y: translateY))
    }
}

/// core.Translate as a GeometryEffect rather than `.offset`.
///
/// `.offset` takes points, and a percentage has to resolve against the view's
/// own size ("-100%" is a panel's whole width, whatever it is). A
/// GeometryEffect is handed that size in effectValue(size:), with no
/// GeometryReader and no extra layout pass. Its animatableData is the four
/// numbers, so the box chain's `.animation(value:)` (grMobTransition)
/// interpolates a change the way it does any other style change.
///
/// x is not negated for RTL. SwiftUI already mirrors a GeometryEffect's
/// horizontal translation under a right-to-left layoutDirection (measured with
/// ImageRenderer: a 30pt shift moved a leading square right in LTR and left in
/// RTL, the same as `.offset`), which is exactly core.Translate's
/// leading-relative rule. Like `.offset`, the effect moves hit-testing with
/// the pixels and leaves the frame reported to the parent alone.
struct GrMobTranslate: GeometryEffect {
    var x: GrMobShift
    var y: GrMobShift

    var animatableData: AnimatablePair<AnimatablePair<CGFloat, CGFloat>, AnimatablePair<CGFloat, CGFloat>> {
        get {
            AnimatablePair(AnimatablePair(x.points, x.fraction), AnimatablePair(y.points, y.fraction))
        }
        set {
            x = GrMobShift(points: newValue.first.first, fraction: newValue.first.second)
            y = GrMobShift(points: newValue.second.first, fraction: newValue.second.second)
        }
    }

    func effectValue(size: CGSize) -> ProjectionTransform {
        ProjectionTransform(CGAffineTransform(
            translationX: x.points + x.fraction * size.width,
            y: y.points + y.fraction * size.height))
    }
}

/// Maps one core.Role onto SwiftUI accessibility traits.
///
/// Seven of the twenty-five roles land on a trait; the other eighteen are spelled out
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
/// Go's core.Style.AccessibilityHeadingLevel as a SwiftUI heading level.
///
/// SwiftUI is the one native of the two that can express this at all —
/// Compose's `heading()` takes no argument — so the level reaches VoiceOver's
/// heading rotor here and is documented as inert in GrMobStyle.kt.
///
/// Two guards, matching the heading arm of the web exporters line for line
/// (htmlout's ariaLevel and the WASM runtime's). The role guard is ARIA's
/// scoping, which Go's field doc adopts: a level belongs to a heading, and a
/// columnheader takes the header trait without one. The range guard drops
/// rather than clamps — `.unspecified` is what a 0 or a 7 means, and inventing
/// an `.h6` for a 7 would state a structure the app never described.
///
/// # The other two roles aria-level serves have no mapping here
///
/// ARIA defines aria-level for listitem and row as well, which Go carries as
/// core.Style.AccessibilityNestingLevel — how deep an item sits inside a
/// nested collection. SwiftUI has no nesting-depth property of any kind, and
/// Compose's nearest one describes an item's index within a single collection
/// rather than its depth within nested ones, so that field is inert on both
/// natives and lives on the web alone. The key is deliberately not parsed
/// into GrMobStyle, for the reason the heading gap is written down in
/// GrMobStyle.kt: a field silently ignored is indistinguishable from one
/// nobody had heard of. mobile/verify/nesting_level_test.go pins both halves.
private func grMobHeadingLevel(_ s: GrMobStyle) -> AccessibilityHeadingLevel {
    guard s.accessibilityRole == "heading" else { return .unspecified }
    switch s.accessibilityHeadingLevel {
    case 1: return .h1
    case 2: return .h2
    case 3: return .h3
    case 4: return .h4
    case 5: return .h5
    case 6: return .h6
    default: return .unspecified
    }
}

/// # AccessibilityID and AccessibilityControls are not read here
///
/// Go's two IDREF props — an element identity and the aria-controls that
/// points at it — cross the bridge and are deliberately unparsed. There is no
/// relationship of that kind in SwiftUI's semantics vocabulary: VoiceOver moves
/// through a screen by swiping to the next element, not by following a
/// reference from a tab to the region it shows, so there is nothing here for
/// the pair to become.
///
/// The near miss is `accessibilityIdentifier`, and taking it would be wrong
/// rather than approximate. That property is a UI-test selector — XCUITest
/// reads it, VoiceOver never does — so filling it from an ARIA wiring string
/// would silently turn every hand-built tab into a test handle and still
/// announce nothing. mobile/verify/idref_test.go pins both halves.
///
/// # AccessibilitySelectionFollowsFocus is not read here either
///
/// Go's flag says a composite widget should choose the member its arrow keys
/// land on. It crosses the bridge and is deliberately unparsed, and the reason
/// is the flat one: there are no arrow keys. VoiceOver crosses a collection by
/// swipe, and a swipe moves the reader's cursor without moving focus at all —
/// so "selection follows focus" describes a sequence that does not happen on
/// this platform, rather than a behaviour SwiftUI spells differently.
///
/// The near miss is `.accessibilityRespondsToUserInteraction` or reaching for
/// `AccessibilityFocusState`, and both would be wrong in the same way: they are
/// about *whether* an element takes the reader's attention, where this is about
/// what a widget does once the attention has already moved. Acting on the flag
/// would mean firing the app's OnTap on every swipe past a row, which is a
/// selection nobody asked for on the one platform where the user cannot see it
/// coming. mobile/verify/followsfocus_test.go pins both halves.
///
/// # AccessibilityExpanded is not read here either, for a different reason
///
/// Go's core.ExpandedState — whether a disclosure is open — crosses the bridge
/// and is deliberately unparsed. This is not the IDREF case above, where the
/// concept has no meaning on the platform: a disclosure means exactly what it
/// means everywhere, SwiftUI ships a `DisclosureGroup`, and VoiceOver does
/// announce its state. What is missing is a *property to put it in*.
/// `AccessibilityTraits` has no expanded member, and `DisclosureGroup`
/// announces itself by writing an accessibility **value** — a localized string
/// SwiftUI supplies from its own bundle.
///
/// The near miss is therefore `accessibilityValue`, and taking it would mean
/// this renderer emitting the literal "expanded" or "collapsed" in English,
/// for every app, in every locale. That is the same move `comps.Chip`
/// deleted when it stopped appending ", selected" to its accessibility label:
/// a state written into a text channel, announced in the wrong place and in a
/// language nobody chose. The value slot also belongs to the app — a slider
/// or a field that states its own value would have it overwritten.
///
/// So this platform says nothing, and Compose — which has `expand`/`collapse`
/// semantics actions — says what it can. That asymmetry is the reverse of the
/// usual one for this framework, where the natives agree and the web is the
/// strict target. mobile/verify/expanded_test.go pins the note, the absence of
/// a parse, and the property it is turning down.
///
/// # AccessibilityHasPopup is not read here either
///
/// Go's popup kind says a control opens a dialog before it is pressed.
/// `AccessibilityTraits` has no popup member. The near miss is
/// `accessibilityHint`, which belongs to the app and would need this renderer
/// to write the English words "opens a dialog" into it — the move the note
/// above refuses for "expanded". A Modal presents here as a sheet, which
/// VoiceOver announces as it appears, so the warning arrives one step late
/// rather than not at all. The key is deliberately not parsed;
/// mobile/verify/expanded_test.go pins both halves.
private func grMobTraitsFor(_ role: String) -> AccessibilityTraits {
    switch role {
    case "heading", "columnheader": .isHeader
    case "button": .isButton
    // The one platform of the four that distinguishes a link from a button in
    // its own vocabulary, which is the argument for core.RoleLink existing.
    case "link": .isLink
    case "search": .isSearchField
    // A node standing in for a picture — VoiceOver announces the trait and
    // then the accessibilityLabel, instead of reading the parts the label was
    // supplied to replace.
    case "img": .isImage
    // The strip, not the control in it. SwiftUI has the container half of the
    // tab pair and no trait for a single tab; Compose has exactly the
    // opposite, which is why core.Role carries both and neither platform's
    // vocabulary could have supplied them. See core/role.go.
    case "tablist": .isTabBar
    // The other half, which this platform cannot name. A tab announces as a
    // button (it is one) plus, when the app states it, the .isSelected trait
    // grMobSelectedTrait adds — which is the part of "tab 2 of 3" VoiceOver
    // can actually be told here.
    case "tab": []
    // The region a tab shows. No trait, and little lost: what makes a tabpanel
    // announce as one on the web is being pointed at by an aria-controls, and
    // VoiceOver moves by swiping to the next element rather than by following
    // a reference — the same reason the IDREF pair above is unparsed. A
    // core.TabView announces correctly here regardless, through the bar
    // Renderer.swift draws.
    case "tabpanel": []
    // No SwiftUI trait names these.
    case "table", "rowgroup", "row", "cell": []
    case "list", "listitem": []
    // A listbox and one option in it. SwiftUI names neither, and the loss is
    // smaller than the empty arm suggests: what a chosen row most needs said
    // is the *state*, and grMobSelectedTrait below adds .isSelected on any
    // view without asking what contains it. Missing is only the container's
    // word for what the choice is among — which VoiceOver, navigating by
    // swipe rather than by arrow key, does not use the way a browser does.
    case "listbox", "option": []
    // A radio group and one radio in it. SwiftUI has no trait for either —
    // no radio-button trait, no container word for a set of them — so this is
    // the listbox pair's loss again: the checked radio is announced through
    // .isSelected, which grMobSelectedTrait adds on any view, and the word
    // "radio button" is what this platform cannot say.
    case "radiogroup", "radio": []
    // An interactive grid and one cell in it. No trait names a grid, and
    // VoiceOver moves through one by swiping rather than by arrow key, so the
    // container's word is the loss. The cell is a control and .isButton is the
    // control word this platform has — the same trait comps.Calendar's days
    // carried while they were RoleButton, so moving the widget onto the grid
    // pair changed nothing a VoiceOver user hears.
    case "grid": []
    case "gridcell": .isButton
    // A progress bar. No trait names one, and — unlike `tab` above, whose
    // state VoiceOver can at least be told — there is nothing this platform
    // can be told about the position either: `accessibilityValue` takes a
    // string and SwiftUI has no numeric equivalent of Compose's
    // ProgressBarRangeInfo. So the whole announcement here is whatever words
    // the app put in core.ValueRange.Text; see grMobValueText.
    case "progressbar": []
    case "banner", "navigation", "toolbar": []
    // The field that owns a popup list. No trait names one; the field is a
    // TextField, which VoiceOver already announces as editable, and the list
    // under it is reached by swiping. See core.RoleComboBox.
    case "combobox": []
    // Nor these: SwiftUI announces a change through
    // AccessibilityNotification, which is an imperative call at the moment of
    // the change and not a property of the view that changed. "log" is the
    // third of that family and is dropped with them — its distinction from
    // "status" is about the shape of the content (appended and ordered rather
    // than replaced), which this platform has no way to state at all.
    case "status", "alert", "log": []
    // The naming role, and the one empty arm here that is empty because this
    // platform does not *need* it rather than because it cannot say it. On the
    // web a name on a generic element is prohibited and dropped, so `group` is
    // what makes an accessibilityLabel on a plain container audible at all;
    // VoiceOver honours accessibilityLabel on any view, so there is nothing
    // for the role to unlock here. See core/role.go's RoleGroup.
    case "group": []
    default: []
    }
}

/// Go's core.SelectedState as a trait.
///
/// SwiftUI has one word where ARIA has two: .isSelected covers both
/// aria-selected and aria-pressed, so unlike the two web targets this needs no
/// switch on the role — see the mapping table in Go's
/// core.Style.AccessibilitySelected.
///
/// It also has no word for the *off* state. An unselected control simply
/// carries no trait, which is the same view an unstated one produces, so this
/// platform cannot distinguish "off" from "not selectable" and does not try.
/// That is a real loss and it is on SwiftUI's side of the line: the value
/// crosses the bridge and there is nothing here to spend it on. The web
/// targets, where a tablist genuinely needs every tab to answer, write both.
///
/// The role is deliberately not consulted. VoiceOver honours .isSelected on
/// any view, so guarding it the way the web exporters do would drop a state
/// this platform would otherwise have announced — the web is strict because
/// ARIA scopes its attributes, not because the framework does.
private func grMobSelectedTrait(_ state: String) -> AccessibilityTraits {
    state == "true" ? .isSelected : []
}

/// Go's core.CurrentKind as a trait.
///
/// SwiftUI has no current trait. .isSelected is what a UITabBar's current item
/// carries, so VoiceOver announces a current bottom-bar cell here as it
/// announces the platform's. The kind (page, step, true) is not distinguished.
///
/// A stated core.SelectedState wins, including "false": grMobSelectedTrait has
/// already answered for it, and a node that says both is making the more
/// specific claim with that field.
///
/// "date" is not folded. A calendar's today cell carrying .isSelected would be
/// announced as the chosen day, and a calendar has a chosen day of its own; the
/// fact goes into the label instead, through grMobCurrentLabel. See Go's
/// core.CurrentKind, "CurrentDate is the exception to the fold".
private func grMobCurrentTrait(_ kind: String, selected: String) -> AccessibilityTraits {
    !kind.isEmpty && kind != "date" && selected.isEmpty ? .isSelected : []
}

/// The accessibility label with core.CurrentDate spoken into it.
///
/// SwiftUI has no current trait and "today" must not become .isSelected (see
/// grMobCurrentTrait), so the label is the one channel left. The suffix is the
/// one comps.Calendar used to write in Go for every target; it moved here when
/// the web targets gained aria-current="date", which a browser's screen reader
/// announces in the user's own language. It is still English on this
/// platform, as it was before.
///
/// Only called with a non-empty label (grMobAccessibility's branch requires
/// one), so there is no bare ", today" to guard against here.
private func grMobCurrentLabel(_ label: String, kind: String) -> String {
    kind == "date" ? label + ", today" : label
}


/// core.MaxWidth as a proposal cap; see GrMobMaxWidthLayout.
///
/// A ViewModifier, not a `@ViewBuilder` extension on View like its neighbours,
/// and the difference is the compiler's rather than the layout's. Called from
/// grMobBox's chain as a generic extension, `GrMobMaxWidthLayout(...) { self }`
/// crashed swiftc in SILGen (signal 6 in QueryReplacementTypeArray while
/// lowering GrMobBoxModifier.body): a Layout's callAsFunction is itself generic
/// over its content, and nesting that inside the chain's opaque types was one
/// substitution too many. Inside a modifier the content is the one concrete
/// `_ViewModifier_Content<Self>` type — the same reason GrMobBoxModifier
/// exists — and the crash is gone.
///
/// The body stays strictly conditional, like grMobGrow: an uncapped node —
/// nearly every node — gets its content back with no layout container around
/// it, so its ideal size is not reported through one more layer for nothing.
struct GrMobMaxWidthModifier: ViewModifier {
    let value: String
    let margin: CGFloat

    @ViewBuilder func body(content: Content) -> some View {
        if value.isEmpty || value == "none" || value == "auto" {
            content
        } else {
            GrMobMaxWidthLayout(value: value, margin: margin) { content }
        }
    }
}

/// The horizontal margin grMobBox reserves outside a node's border box.
func grMobHorizontalMargin(_ s: GrMobStyle?) -> CGFloat {
    guard let s else { return 0 }
    return CGFloat(s.margin.left + s.margin.right)
}

/// CSS `max-width` for one view: never propose more than the cap, never report
/// more than the cap, and sit at the leading edge of a wider slot.
///
/// A Layout rather than `.frame(maxWidth: cap)`, because a flexible frame with
/// only a maximum is greedy: it reports min(cap, proposal) even around content
/// that wants less, so a hugging label given a 320-point cap would claim 320
/// points of its row. CSS's max-width never grows a box. This proposes the
/// capped width inward and reports what the child actually took.
///
/// ```
///   proposal W     child is proposed              reports               places child
///   ──────────     ─────────────────              ───────               ────────────
///   definite       min(W, cap + margin)           min(child, that)      leading edge
///   nil (ideal)    nil; cap + margin only if      min(child, cap + m)
///                  the ideal overflows the cap
/// ```
///
/// The nil row is what a flex parent's basis measurement asks (baseMains in
/// Renderer.swift proposes nil on the main axis). A Text proposed nil answers
/// with its one-line width; clamping that number without re-proposing would
/// report a capped frame around a line that still draws uncapped, so an ideal
/// wider than the cap is measured again at the cap and wraps there — CSS's
/// hypothetical main size clamped by max-width.
///
/// Placement is leading, not centred: a parent that hands over bounds wider
/// than this reported — a stretched Column child is placed at the whole cross
/// extent — gets a capped box at the start of its slot, which is where CSS
/// puts a stretched item that max-width stopped short. Centring is the
/// author's to ask for with the parent's AlignItems, as on the web.
///
/// Not handled: a FlexGrow child in a Row whose cap binds keeps the share the
/// solver gave it and leaves the rest of that share empty. CSS re-runs the
/// distribution with that item frozen at its max; GrMobFlexSolver has no max
/// input, and Compose has the same gap (see widthModifier in GrMobStyle.kt).
struct GrMobMaxWidthLayout: Layout {
    let value: String
    /// Horizontal margin inside this layer, added back to the cap.
    let margin: CGFloat

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        guard let child = subviews.first else { return .zero }
        let size = child.sizeThatFits(childProposal(proposal, child: child))
        guard let bound = outerLimit(proposal.width) else { return size }
        return CGSize(width: min(size.width, bound), height: size.height)
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        guard let child = subviews.first else { return }
        // Resolved against the bounds actually drawn into, as GrMobFlexLayout
        // does, so a parent that places this wider than it measured still
        // gets a capped child rather than one proposed the whole slot.
        let inner = childProposal(ProposedViewSize(width: bounds.width, height: bounds.height),
                                  child: child)
        child.place(at: bounds.origin, anchor: .topLeading, proposal: inner)
    }

    /// The cap plus the margin it must not eat, against an offered width.
    private func outerLimit(_ offered: CGFloat?) -> CGFloat? {
        GrMobMaxWidth.limit(value, available: offered).map { $0 + margin }
    }

    private func childProposal(_ p: ProposedViewSize, child: LayoutSubview) -> ProposedViewSize {
        guard let offered = p.width else {
            // Ideal-size query: only a points cap can bind (a percentage has
            // nothing to resolve against), and only if the ideal overflows it.
            guard let bound = outerLimit(nil),
                  child.sizeThatFits(p).width > bound else { return p }
            return ProposedViewSize(width: bound, height: p.height)
        }
        // `.infinity` is a proposal too (SwiftUI's max-size probe): a points
        // cap binds it, and a percentage of it resolves to nothing.
        guard let bound = outerLimit(offered.isFinite ? offered : nil) else { return p }
        return ProposedViewSize(width: min(offered, bound), height: p.height)
    }
}
