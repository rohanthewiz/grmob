import Foundation
import Observation

/// One node of the live UI tree, mirroring Go's core.Node.
///
/// This is a *data* tree, not a view tree: SwiftUI views read from it and the
/// Observation framework does the rest. `@Observable` gives per-property
/// access tracking — the SwiftUI analog of Compose snapshot state — so a
/// patch write invalidates exactly the views that read the mutated property:
///
///   patch            mutates            re-evaluates
///   ─────            ───────            ────────────
///   update-props  →  node.props      →  only views reading this node's props
///   update-style  →  node.style      →  only this node's box/text styling
///   add/remove/   →  parent.children →  only the parent's children loop
///   replace                             (siblings keep their view identity —
///                                       ForEach ids are the node instances)
///
/// `type` and `key` are immutable by design: the Go reconciler never mutates
/// a node across those axes — it emits `replace`, which swaps in a fresh
/// GrMobNode instance here. Because instances are the ForEach identity,
/// that swap is precisely what resets SwiftUI view state for the replaced
/// subtree while structural inserts/removes leave sibling state untouched.
@Observable
final class GrMobNode {
    let type: String
    let key: String
    var props: [String: Any]
    var style: GrMobStyle?
    var children: [GrMobNode]

    init(type: String, key: String, props: [String: Any], style: GrMobStyle?, children: [GrMobNode]) {
        self.type = type
        self.key = key
        self.props = props
        self.style = style
        self.children = children
    }

    /// The style a container renders with: `style`, plus `labelOnly` when
    /// every child is AccessibilityHidden (or there are none), which is what
    /// tells grMobAccessibility that a combine would have nothing to merge.
    ///
    /// Computed on read rather than stored at parse time, because children
    /// arrive and leave by patch: a stored flag would describe the tree as it
    /// was first sent. Reading `children` and each child's `style` here is
    /// also what registers the container's body with Observation for both,
    /// so a child that stops being hidden re-renders the parent. Only labelled
    /// containers pay for the walk, and it is one level deep.
    var containerStyle: GrMobStyle? {
        guard var s = style, !s.accessibilityLabel.isEmpty, !s.accessibilityHidden else { return style }
        s.labelOnly = children.allSatisfy { $0.style?.accessibilityHidden == true }
        s.holdsControls = holdsControls
        return s
    }

    // MARK: Members a labelled container keeps (N-078)
    //
    // A labelled container used to become one VoiceOver element whatever it
    // held (`.combine`). That is the feed-row pattern, and right for a row
    // around one Switch: the row *is* the switch. It is wrong for a named
    // container of several stops, because one element has one activation and
    // one sentence. Seen in XCUITest's tree on the iOS 26.5 simulator:
    // RangeSlider's "Price" was one slider valued "10%, 40%" with two
    // unlabelled sliders under it, and EditableGrid's "Budget" was one static
    // text with every text cell gone. SwiftUI's own Buttons happened to
    // survive the combine; a tappable Row (a grid cell, a swatch) and a Slider
    // did not. And a Poll's results, three named rows, were one stop where the
    // widget promises one per option.
    //
    // So a labelled container holding two or more members (isMember: a
    // control, a pressable node, a named container) keeps them (`.contain`:
    // the label names the container, each member stays an element), and one
    // holding at most one still combines. This is what Compose does without
    // being asked: every clickable node and every labelled node is its own
    // merge boundary there, so TalkBack reached each of these members all
    // along.
    //
    // Decided by content and not by role, because the roles cannot tell the
    // cases apart: Stepper, Drawer's panel, MessageBubble and RangeSlider
    // are all RoleGroup.
    //
    //	labelled container         children              VoiceOver sees
    //	───────────────────────    ──────────────────    ──────────────────────
    //	every child hidden         (nothing to merge)    one element  .ignore
    //	two or more members        Slider, Slider        a container  .contain
    //	  anywhere below           cell, cell, …         of elements
    //	otherwise                  Text, Switch          one element  .combine
    //
    // ios/verify's childmode.swift runs this over the bundled widgets and
    // holds each labelled container to the shape Go lists for it.

    /// The node types that are controls in their own right: each draws a
    /// native control VoiceOver operates directly.
    static let controlTypes: Set<String> = [
        "Button", "Input", "InputPassword", "NumericInput", "TextArea", "Select",
        "Checkbox", "Switch", "Slider", "CodeEditor", "RichTextEditor",
    ]

    /// Whether this node is one member of a labelled container: a stop of
    /// its own for VoiceOver, which a merge into the container would lose.
    ///
    ///	a control type            it draws a native control
    ///	a pressable node          a tap or long press (a Row or Box made
    ///	                          pressable, which GrMobGestures gives the
    ///	                          button trait), when it has something to
    ///	                          announce: a label, or a child not hidden
    ///	a labelled container      it named itself, so it is one stop (or a
    ///	                          named container of its own); Compose makes
    ///	                          every labelled node a merge boundary too
    ///
    /// Hidden nodes are not members; VoiceOver never reaches them. PINInput's
    /// tap-catching row is the case for "something to announce": pressable,
    /// unnamed, every child hidden, so SwiftUI makes no element of it.
    var isMember: Bool {
        guard let s = style else {
            return GrMobNode.controlTypes.contains(type) || isPressable && !children.isEmpty
        }
        if s.accessibilityHidden { return false }
        if GrMobNode.controlTypes.contains(type) { return true }
        let named = !s.accessibilityLabel.isEmpty
        if isPressable {
            return named || children.contains { $0.style?.accessibilityHidden != true }
        }
        return named && GrMobNode.labelledContainerTypes.contains(type)
    }

    /// A tap or a long press, which is what makes a container a control.
    var isPressable: Bool {
        !stringProp("onClick").isEmpty || !stringProp("onLongPress").isEmpty
    }

    /// The node types whose views apply a label through containerStyle, and
    /// so with a child behaviour: GrMobRow, GrMobColumn (Column, Card, Box)
    /// and GrMobZStack. A labelled leaf is one element whatever holds it.
    static let labelledContainerTypes: Set<String> = ["Row", "Column", "Card", "Box", "ZStack"]

    /// Whether two or more members (isMember) sit below this node.
    ///
    /// Pre-order, and it stops at the second member, which bounds what the
    /// container's body reads: every node visited registers with Observation
    /// (its props, style and children), so a container is re-evaluated when
    /// one of them changes. A grid finds its second cell in its first row and
    /// reads nothing past it. A container with fewer than two members reads
    /// its whole subtree; those are small (a bubble, a labelled row) or have
    /// their members near the top.
    ///
    /// A member's own subtree is not entered: a pressable Row's texts are its
    /// label, and a named container answers for its own members. A hidden
    /// subtree is not entered either.
    var holdsControls: Bool {
        var found = 0
        func walk(_ node: GrMobNode) -> Bool {
            for child in node.children {
                if child.style?.accessibilityHidden == true { continue }
                if child.isMember {
                    found += 1
                    if found >= 2 { return true }
                    continue
                }
                if walk(child) { return true }
            }
            return false
        }
        return walk(self)
    }

    // Typed prop accessors; Go serializes props with lowercase keys.
    func stringProp(_ name: String) -> String { props[name] as? String ?? "" }
    func boolProp(_ name: String) -> Bool { props[name] as? Bool ?? false }
    func intProp(_ name: String) -> Int { (props[name] as? NSNumber)?.intValue ?? 0 }
    func doubleProp(_ name: String) -> Double { (props[name] as? NSNumber)?.doubleValue ?? 0 }

    /// Decodes a Go core.Node JSON object (keys are the Go field names,
    /// as produced by JSONSerialization).
    static func parse(_ obj: [String: Any]) -> GrMobNode {
        var children: [GrMobNode] = []
        if let childArray = obj["Children"] as? [Any] {
            children.reserveCapacity(childArray.count)
            for slot in childArray {
                // Go child slots can hold JSON null (nil *Node); skip them —
                // the reconciler's Diff treats nil slots as absent too.
                guard let child = slot as? [String: Any] else { continue }
                children.append(parse(child))
            }
        }
        return GrMobNode(
            type: obj["Type"] as? String ?? "",
            key: obj["Key"] as? String ?? "",
            props: obj["Props"] as? [String: Any] ?? [:],
            style: GrMobStyle.parse(obj["Style"] as? [String: Any]),
            children: children
        )
    }
}

/// A labelled container's child behaviour, as a value the ios/verify harness
/// can compare without SwiftUI. grMobChildBehavior maps it onto SwiftUI's
/// AccessibilityChildBehavior one to one.
enum GrMobChildMode: String {
    case ignore, contain, combine
}

/// The precedence, in one place: nothing to merge first (a container whose
/// children are all hidden has no members either, and `.ignore` is the one
/// that still makes its element), then members to keep, then the combine.
/// Only meaningful for a labelled container's style (GrMobNode.containerStyle).
func grMobChildMode(_ s: GrMobStyle) -> GrMobChildMode {
    if s.labelOnly { return .ignore }
    if s.holdsControls { return .contain }
    return .combine
}
