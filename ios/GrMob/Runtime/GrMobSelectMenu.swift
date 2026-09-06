import Foundation

/// A picker's option list, decomposed into the menu a person sees.
///
/// Split out of GrMobSelect (Renderer.swift) for the reason GrMobFlex.swift is
/// split out of GrMobFlexLayout, and it is the same reason twice: a SwiftUI
/// `Menu`'s content is a closure of views, and there is no way to ask a built
/// view what is in it — no test can open a picker, read back its Sections, or
/// find out whether the third row refuses a tap. Only a simulator and a human
/// can. Everything that is actually easy to get wrong, though — which options
/// share a heading, where a run ends, which value a row dispatches, which row
/// is refused — is a function from a list of dictionaries to a list of
/// structs. This file imports Foundation and nothing else, so `ios/verify` can
/// run it on a plain macOS host and compare its answers with Go's.
///
/// The rule it implements is core.SelectOption.Group's, and the authority for
/// it is `core.SelectMenuSections` (core/select_menu.go), which htmlout calls
/// directly and which ios/verify generates this harness's cases from. The
/// comments here say what the code does; that file says why.

/// One choosable row of the menu: an option resolved out of the flat wire map.
///
/// `index` is the option's position in the original list, and it is what
/// identifies a row to `ForEach`. Neither of the alternatives works: a
/// `[String: Any]` is not `Hashable`, and two options are allowed to share a
/// label (core.Select's Value is the identity; the Label is written to be
/// read).
struct GrMobMenuItem: Equatable {
    let index: Int
    let value: String
    let label: String
    let isDisabled: Bool
}

/// One run of consecutive options sharing a heading.
///
/// An empty `heading` is the run of options that stand on their own at the top
/// level of the list. That is a real section rather than an absence of one, so
/// the drawing code is a single loop over sections with the wrapper as its
/// only branch.
struct GrMobMenuSection: Equatable {
    let heading: String
    let items: [GrMobMenuItem]

    /// What identifies the section to `ForEach`. The heading cannot:
    /// core.SelectOption.Group allows the same heading either side of a
    /// different one, and that is two sections. The starting index can,
    /// because a section is a contiguous run.
    var first: Int { items.first?.index ?? 0 }
}

/// Splits a picker's flattened options into runs by their "group".
///
/// Runs, not a gather: consecutive options sharing a heading are one section,
/// in the order they were written. A run is closed by the *next* option naming
/// a different heading, so the last run of a list has nothing following it and
/// is flushed after the loop — the one piece of bookkeeping this rule needs,
/// and the piece each renderer that reimplemented it had to remember.
func grMobMenuSections(_ options: [[String: Any]]) -> [GrMobMenuSection] {
    var sections: [GrMobMenuSection] = []
    // The run being filled, and the heading it was opened with. `building` is
    // separate from the heading because the empty heading is a legitimate one
    // and would otherwise be indistinguishable from "nothing open yet".
    var heading = ""
    var items: [GrMobMenuItem] = []
    var building = false

    for (i, option) in options.enumerated() {
        let group = option["group"] as? String ?? ""
        if !building || group != heading {
            if building {
                sections.append(GrMobMenuSection(heading: heading, items: items))
            }
            heading = group
            items = []
            building = true
        }
        let value = option["value"] as? String ?? ""
        // The label defaulted to the value at core.Select's flattening seam,
        // so this fallback is for a hand-assembled node that never went
        // through it — a menu row with no text is worse than one showing its
        // value.
        let label = option["label"] as? String ?? ""
        items.append(GrMobMenuItem(
            index: i,
            value: value,
            label: label.isEmpty ? value : label,
            // The wire carries the string "true", core.SelectedState's
            // spelling one property over. Anything else is not disabled.
            isDisabled: (option["disabled"] as? String ?? "") == "true"
        ))
    }
    if building {
        sections.append(GrMobMenuSection(heading: heading, items: items))
    }
    return sections
}
