// The picker menu, checked as behavior rather than as source text.
//
// A SwiftUI Menu's content is a closure of views. Nothing can ask a built view
// what is in it, so no test outside a simulator can open a picker, read back
// its Sections, or find out whether the third row refuses a tap. Three facts
// about that menu — that consecutive options sharing a heading form one run,
// that a run ending the list is still closed, and that a disabled option is
// drawn and refused rather than dropped — therefore rested entirely on
// mobile/verify reading Renderer.swift as text and finding the right
// substrings in it.
//
// GrMobSelectMenu.swift is what moves those three facts out of the closure: it
// turns the flat option list into sections and rows with no SwiftUI in sight,
// and Renderer.swift's Menu is one ForEach over the result. This file runs it.
//
// The expectations come from Go — gen.go computes them with
// core.SelectMenuSections, the authority htmlout calls directly, over the case
// table internal/menufixture holds for all three transliterations — so what is
// compared here is the *transliteration*, which is the thing that can drift.
// What is still out of reach is the last step, the one line that hands
// `item.isDisabled` to `.disabled(_:)` and `item.value` to `textChanged`;
// mobile/verify still reads that as text, and that is now the whole of what
// it reads.
import Foundation

/// One case from gen.go: an option list and the menu Go says it becomes.
struct MenuCase: Decodable {
    let name: String
    let options: [[String: String]]
    let want: [WantSection]
}

/// core.SelectMenuSection, over the wire. `first` is carried explicitly so the
/// Swift section's own computed property is compared rather than assumed.
struct WantSection: Decodable {
    let heading: String
    let first: Int
    let disabled: Bool
    let items: [WantItem]
}

/// core.SelectMenuItem, over the wire.
struct WantItem: Decodable {
    let index: Int
    let value: String
    let label: String
    let disabled: Bool
}

/// Runs every case through grMobMenuSections and reports each difference as
/// its own line, naming the case and the position — a failure should say which
/// property broke, not dump two structures.
func checkSelectMenu(_ cases: [MenuCase]) -> [String] {
    var problems: [String] = []

    for c in cases {
        // The renderer receives [[String: Any]] — JSONSerialization's output
        // for a list of flat string maps — so the widening is what makes this
        // the same call Renderer.swift makes.
        let got = grMobMenuSections(c.options.map { $0 as [String: Any] })

        if got.count != c.want.count {
            problems.append("\(c.name): \(got.count) sections, Go says \(c.want.count)")
            continue
        }
        for (i, pair) in zip(got, c.want).enumerated() {
            let (g, w) = pair
            if g.heading != w.heading {
                problems.append("\(c.name) section \(i): heading \"\(g.heading)\", Go says \"\(w.heading)\"")
            }
            // The ForEach key. Two runs may share a heading, so this is the
            // only thing that can tell them apart.
            if g.first != w.first {
                problems.append("\(c.name) section \(i): first \(g.first), Go says \(w.first)")
            }
            // core.SelectOption.GroupDisabled, resolved. The dangerous wrong
            // answer is reading the declaration when the run is *opened*,
            // which passes every case whose first option carries it — hence
            // the fixture case that puts it on the last one.
            if g.isDisabled != w.disabled {
                problems.append("\(c.name) section \(i): disabled \(g.isDisabled), Go says \(w.disabled)")
            }
            if g.items.count != w.items.count {
                problems.append("\(c.name) section \(i): \(g.items.count) options, Go says \(w.items.count)")
                continue
            }
            for (j, itemPair) in zip(g.items, w.items).enumerated() {
                let (gi, wi) = itemPair
                let where_ = "\(c.name) section \(i) option \(j)"
                if gi.index != wi.index {
                    problems.append("\(where_): index \(gi.index), Go says \(wi.index)")
                }
                // The value is what goes up to Go when the row is tapped. The
                // dangerous wrong answer is the label, which would work in
                // every test whose labels happen to equal its values.
                if gi.value != wi.value {
                    problems.append("\(where_): value \"\(gi.value)\", Go says \"\(wi.value)\"")
                }
                if gi.label != wi.label {
                    problems.append("\(where_): label \"\(gi.label)\", Go says \"\(wi.label)\"")
                }
                if gi.isDisabled != wi.disabled {
                    problems.append("\(where_): disabled \(gi.isDisabled), Go says \(wi.disabled)")
                }
            }
        }
    }
    return problems
}
