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

            case "Row": GrMobRow(node: node, grow: grow)
            case "Column", "Card": GrMobColumn(node: node, grow: grow) // Card = Column whose Go theme style carries the card look
            case "List": GrMobList(node: node, grow: grow)
            case "Box": ZStack(alignment: .topLeading) { PlainChildren(node: node) }
                .grMobBox(node.style, grow: grow,
                            onTap: node.stringProp("onClick"),
                            onLongPress: node.stringProp("onLongPress"))
            case "Spacer": Color.clear.frame(width: CGFloat(node.intProp("size")), height: CGFloat(node.intProp("size")))
            case "Scroll":
                ScrollView {
                    VStack(alignment: .leading, spacing: 0) { PlainChildren(node: node) }
                }.grMobBox(node.style, grow: grow)
            case "SafeArea":
                // SwiftUI already lays out inside the safe area by default, so
                // this is just a grouping box — the node exists so Go apps can
                // be explicit about it and so an ignoresSafeArea escape hatch
                // has a home later.
                ZStack(alignment: .topLeading) { PlainChildren(node: node) }.grMobBox(node.style, grow: grow)

            case "TabView": GrMobTabView(node: node, grow: grow)
            case "Modal": GrMobModal(node: node)
            case "Image":
                // The AccessibilityLabel style prop travels through grMobBox;
                // the legacy "alt" prop fills in only when no style label is set
                // (an unconditional outer accessibilityLabel would override the
                // box's label with an empty string).
                AsyncImage(url: URL(string: node.stringProp("src"))) { image in
                    image.resizable().scaledToFit()
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
        HStack(alignment: crossAlignmentV(s), spacing: CGFloat(s?.gap ?? 0)) {
            FlexChildren(node: node, axis: .horizontal)
        }
        .grMobBox(s, grow: grow,
                    onTap: node.stringProp("onClick"),
                    onLongPress: node.stringProp("onLongPress"))
    }
}

private struct GrMobColumn: View {
    let node: GrMobNode
    let grow: GrMobGrow

    var body: some View {
        let s = node.style
        VStack(alignment: crossAlignmentH(s), spacing: CGFloat(s?.gap ?? 0)) {
            FlexChildren(node: node, axis: .vertical)
        }
        .grMobBox(s, grow: grow,
                    onTap: node.stringProp("onClick"),
                    onLongPress: node.stringProp("onLongPress"))
    }
}

/// Main-axis distribution. SwiftUI stacks have no justify-content, so the
/// CSS values are emulated with flexible Spacers — faithful because CSS
/// justify-content likewise only matters when the container has free space
/// along the main axis (here: when the stack was given a size or a grow
/// frame; a hugging stack has no free space and the spacers collapse).
/// space-around is approximated as space-evenly (equal spacers at edges and
/// between) — exact half-size edge gaps would need a custom Layout.
private struct FlexChildren: View {
    let node: GrMobNode
    let axis: Axis

    var body: some View {
        let justify = node.style?.justifyContent ?? ""
        let edges = ["center", "flex-end", "space-around", "space-evenly"].contains(justify)
        let between = ["space-between", "space-around", "space-evenly"].contains(justify)
        let n = node.children.count

        if edges, justify != "flex-end" { Spacer(minLength: 0) }
        if justify == "flex-end" { Spacer(minLength: 0) }
        ForEach(Array(node.children.enumerated()), id: \.element.viewID) { i, child in
            if between, i > 0 { Spacer(minLength: 0) }
            RenderNode(node: child, grow: growFor(child))
        }
        if edges, justify != "flex-end", n > 0 { Spacer(minLength: 0) }
    }

    /// FlexGrow maps onto an infinity frame along this stack's main axis —
    /// the parent computes it and hands it down (see GrMobGrow).
    private func growFor(_ child: GrMobNode) -> GrMobGrow {
        guard (child.style?.flexGrow ?? 0) > 0 else { return .none }
        return axis == .horizontal ? .horizontal : .vertical
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
                ForEach(rows, id: \.viewID) { row in
                    RenderNode(node: row)
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
        .grMobBox(s, grow: grow,
                    onTap: node.stringProp("onClick"),
                    onLongPress: node.stringProp("onLongPress"))
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

/// AlignItems governs cross-axis placement; the DSL's simpler Align
/// ("center"/"end") acts as a fallback when AlignItems is unset.
private func crossAlignmentH(_ s: GrMobStyle?) -> HorizontalAlignment {
    var v = s?.alignItems ?? ""
    if v.isEmpty { v = s?.align ?? "" }
    switch v {
    case "center": return .center
    case "flex-end", "end": return .trailing
    default: return .leading
    }
}

private func crossAlignmentV(_ s: GrMobStyle?) -> VerticalAlignment {
    switch s?.alignItems ?? "" {
    case "center": return .center
    case "flex-end": return .bottom
    default: return .top
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

private func grMobTextAlignment(_ align: String) -> TextAlignment {
    switch align {
    case "center": .center
    case "end": .trailing
    default: .leading
    }
}

private struct GrMobButton: View {
    let node: GrMobNode
    let grow: GrMobGrow
    @Environment(\.grMobRuntime) private var runtime

    var body: some View {
        let s = node.style
        let onClick = node.stringProp("onClick")
        // Style properties the Go theme owns are fed into the button's own
        // label/background rather than grMobBox: the control draws its own
        // container, so background/radius/padding belong inside the pressable
        // area (and inside the press feedback), with only margin/size outside.
        Button {
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
            // same channel as a Button tap — and advertises itself as "done"
            // so the keyboard reflects that the field acts on return.
            .submitLabel(onSubmit.isEmpty ? .return : .done)
            .onSubmit { if !onSubmit.isEmpty { runtime?.click(onSubmit) } }
            .onChange(of: focused) { _, isFocused in
                if isFocused {
                    text = node.stringProp("value")
                    pendingEchoes.removeAll()
                }
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
                VStack(alignment: .leading, spacing: 0) {
                    PlainChildren(node: node)
                }
                .padding(16)
                .presentationDetents([.medium, .large])
            }
    }
}
