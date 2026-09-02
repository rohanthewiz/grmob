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
