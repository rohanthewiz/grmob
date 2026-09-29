// The VoiceOver shape of every labelled container in the bundled widgets
// (N-078), computed by the renderer's own code: GrMobNode.containerStyle, which
// is what the three container views read, and grMobChildMode, which
// grMobAccessibility maps one to one onto SwiftUI's
// AccessibilityChildBehavior. What cannot run here is that last mapping and
// VoiceOver itself; XCUITest's tree on the simulator is the check for those.
//
// The cases and their expected shapes are Go's (gen.go's childmode.go), so
// the widgets are the real ones and a change to any of them that moves a
// container from one shape to another fails here by name.
import Foundation

struct ChildModeCase: Decodable {
    let name: String
    let tree: AnyDecodable
    let want: [String: String]
}

/// The raw tree, kept as JSONSerialization's objects: GrMobNode.parse reads
/// those and nothing else.
struct AnyDecodable: Decodable {
    let value: Any
    init(from decoder: Decoder) throws {
        let c = try decoder.singleValueContainer()
        if let o = try? c.decode([String: AnyDecodable].self) {
            value = o.mapValues { $0.value }
        } else if let a = try? c.decode([AnyDecodable].self) {
            value = a.map { $0.value }
        } else if let b = try? c.decode(Bool.self) {
            value = b
        } else if let n = try? c.decode(Double.self) {
            value = NSNumber(value: n)
        } else if let s = try? c.decode(String.self) {
            value = s
        } else {
            value = NSNull()
        }
    }
}

/// The node types whose views read containerStyle (GrMobRow, GrMobColumn for
/// Column, Card and Box, GrMobZStack). A labelled leaf is one element
/// already, and no other container applies the label with a child behaviour.
private let styledContainers: Set<String> = ["Row", "Column", "Card", "Box", "ZStack"]

func checkChildModes(_ cases: [ChildModeCase]) -> [String] {
    var problems: [String] = []
    for c in cases {
        guard let obj = c.tree.value as? [String: Any] else {
            problems.append("\(c.name): the tree is not an object")
            continue
        }
        var got: [String: String] = [:]
        func walk(_ node: GrMobNode) {
            if node.style?.accessibilityHidden == true { return }
            if styledContainers.contains(node.type), let s = node.containerStyle,
               !s.accessibilityLabel.isEmpty {
                let mode = grMobChildMode(s).rawValue
                if let seen = got[s.accessibilityLabel], seen != mode {
                    problems.append("\(c.name): two containers named \"\(s.accessibilityLabel)\" "
                        + "take different shapes (\(seen), \(mode)); name them apart to list them")
                }
                got[s.accessibilityLabel] = mode
            }
            node.children.forEach(walk)
        }
        walk(GrMobNode.parse(obj))
        if got != c.want {
            let keys = Set(got.keys).union(c.want.keys).sorted()
            for k in keys where got[k] != c.want[k] {
                problems.append("\(c.name): \"\(k)\" is \(got[k] ?? "not a labelled container"), "
                    + "want \(c.want[k] ?? "not listed")")
            }
        }
    }
    return problems
}
