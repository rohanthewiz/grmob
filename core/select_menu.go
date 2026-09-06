package core

// The decomposition of a picker's option list into the menu a person sees.
//
// core.Select flattens []SelectOption into []map[string]string — one flat
// shape all four renderers read (see Select's doc for why). Turning that flat
// list back into sections is then a decision each renderer had to make for
// itself, and all four made it four times: htmlout opened and closed an
// <optgroup>, the WASM runtime opened and closed one against the live DOM,
// Renderer.swift built SwiftUI Sections, Renderer.kt wrote a heading item
// ahead of each run.
//
// Four copies of one rule is three too many, and the rule has an edge that
// each copy has to get right on its own: a run is closed by the *next* option
// naming a different heading, so the last run of a list has nothing following
// it and needs an explicit flush. htmlout's copy of that flush went untested
// for a release — the fixture's last option was ungrouped, so the flush was
// never reached and a `strings.Count` of the closing tags agreed with the bug.
// That is the shape of hole a shared decomposition removes rather than
// documents.
//
// This is the authority. htmlout consumes it directly; the two native
// renderers each carry a transliteration (GrMobSelectMenu.swift,
// GrMobSelectMenu.kt) because neither can call into Go while drawing, and
// ios/verify runs the Swift one against cases generated from this file so the
// transliteration is checked by behaviour rather than by reading the source.

// SelectMenuItem is one choosable row of a picker's menu: an option, resolved
// out of the flat wire map into the three things every renderer asks it.
//
// Index is the option's position in the original list. It is carried because
// a menu row needs an identity that survives two options sharing a label —
// which core.Select explicitly allows, since the Value is the identity and the
// Label is written to be read — and because a renderer that keys its rows on
// the map itself cannot: a map is not hashable in Swift and not comparable in
// Go.
type SelectMenuItem struct {
	Index    int
	Value    string
	Label    string
	Disabled bool
}

// SelectMenuSection is one run of consecutive options sharing a heading.
//
// Heading is empty for the options that stand on their own at the top level,
// and such a run is a real section rather than an absence of one: every
// renderer needs somewhere to put those options, and giving them a section
// with no heading means the drawing code is one loop over sections rather than
// a loop with a special case in it.
type SelectMenuSection struct {
	Heading string
	Items   []SelectMenuItem
}

// First is the index of the section's first option, which is what identifies
// the section to a renderer that needs a key.
//
// The heading cannot be that key: core.SelectOption.Group allows the same
// heading either side of a different one, and that is two sections. The index
// can, because a section is a contiguous run and no two runs start in the same
// place. A section with no items cannot occur — SelectMenuSections opens one
// only when it has an option to put in it — so the read is total.
func (s SelectMenuSection) First() int {
	if len(s.Items) == 0 {
		return 0
	}
	return s.Items[0].Index
}

// SelectMenuSections splits a picker's flattened options into the runs
// core.SelectOption.Group describes.
//
// Runs, not a gather: consecutive options sharing a heading are one section,
// in the order they were written, and the same heading either side of a
// different one is two sections. The field's own doc carries the argument —
// the list's order is the caller's, and no renderer could undo a reordering.
//
// The loop closes a run when the next option names a different heading, and
// flushes the last one after the loop, which is the only bookkeeping the rule
// needs. An empty list gives an empty slice rather than nil, so a renderer can
// range over the result without a guard.
func SelectMenuSections(options []map[string]string) []SelectMenuSection {
	sections := make([]SelectMenuSection, 0, len(options))
	// The section currently being filled, and the heading it was opened with.
	// Two variables rather than reading the heading back off `open`, because
	// the empty heading is a legitimate one and would be indistinguishable
	// from "nothing open yet".
	var open SelectMenuSection
	building := false
	for i, o := range options {
		heading := o["group"]
		if !building || heading != open.Heading {
			if building {
				sections = append(sections, open)
			}
			open = SelectMenuSection{Heading: heading}
			building = true
		}
		// The label already defaulted to the value at core.Select's flattening
		// seam, so no fallback belongs here — but a hand-assembled node can
		// carry a map that never went through it, and a menu row with no text
		// is worse than one showing its value. Same degradation renderSelect
		// gives an options prop of the wrong type entirely.
		label := o["label"]
		if label == "" {
			label = o["value"]
		}
		open.Items = append(open.Items, SelectMenuItem{
			Index: i,
			Value: o["value"],
			Label: label,
			// The wire carries the string "true", core.SelectedState's
			// spelling — it is what the DOM writes, so the web half needs no
			// translation. Anything else is not disabled, which is the same
			// reading all four renderers already made.
			Disabled: o["disabled"] == "true",
		})
	}
	// The last run has no following option to close it. This line is the one
	// the four copies each had to remember.
	if building {
		sections = append(sections, open)
	}
	return sections
}
