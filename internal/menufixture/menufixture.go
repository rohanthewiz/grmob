// Package menufixture is the one table of picker option lists that every
// picker-menu harness is held to.
//
// core.SelectMenuSections (core/select_menu.go) decides how a Select's flat
// option list becomes the menu a person sees. htmlout calls it; the other
// three renderers each carry a transliteration, because none of them can call
// into Go while drawing:
//
//	ios/verify      runs GrMobSelectMenu.swift on a macOS host
//	android/verify  runs GrMobSelectMenu.kt on a JVM
//	wasm/verify     mounts a picker and rebuilds the sections out of the DOM
//
// Each of those needs the same two things: a list of option lists, and the
// answer Go gives for each. Before this package there was one such table, in
// ios/verify/gen.go, and the second harness would have been a second copy of
// it — which is the failure this whole mechanism exists to avoid one level
// down. A fixture written out twice proves that one person made the same
// mistake twice.
//
// The *expected* value is never written here. Cases carry only the input, and
// Wants computes the answer with core.SelectMenuSections, so what each harness
// compares is its target against the authority, not against a hand-copied
// transcription of what the authority was believed to say.
//
// It lives under internal/ because it is not part of the framework's API: it
// is fixture data for this repository's own harnesses, and the harnesses are
// commands in this module.
package menufixture

import "github.com/rohanthewiz/grmob/core"

// Case is one option list and the name a failure reports it by.
//
// Options is the flat wire shape core.Select flattens to — []map[string]string
// — so a harness hands its target exactly what a renderer receives.
type Case struct {
	Name    string              `json:"name"`
	Options []map[string]string `json:"options"`
	Want    []Section           `json:"want"`
}

// Section and Item mirror core.SelectMenuSection / SelectMenuItem with JSON
// names the Swift, Kotlin and JavaScript sides decode without a per-language
// key table.
//
// First is carried explicitly rather than left to be recomputed: it is the key
// a renderer identifies a run by (two runs may share a heading), so the
// target's own First is what should be compared.
type Section struct {
	Heading  string `json:"heading"`
	First    int    `json:"first"`
	Disabled bool   `json:"disabled"`
	Items    []Item `json:"items"`
}

type Item struct {
	Index    int    `json:"index"`
	Value    string `json:"value"`
	Label    string `json:"label"`
	Disabled bool   `json:"disabled"`
}

// lists is the table itself: every case is one property of
// core.SelectOption.Group, .Disabled or .GroupDisabled.
//
// The ones that matter most are the ones a transliteration gets wrong:
//
//   - a heading that reappears after a different one (runs, not a gather)
//   - a run that ends the list, which nothing follows and which therefore
//     needs an explicit flush — the bug htmlout shipped for a release
//   - a run disabled by an option that is not its first, which is only
//     knowable once the run is closed and is therefore the second thing a
//     flush has to do
var lists = []struct {
	name    string
	options []map[string]string
}{
	{"ungrouped", []map[string]string{
		{"value": "s", "label": "Small"},
		{"value": "m", "label": "Medium"},
	}},
	{"one run", []map[string]string{
		{"value": "pt", "label": "Portugal", "group": "Europe"},
		{"value": "fr", "label": "France", "group": "Europe"},
	}},
	{"a heading that comes back is a second run", []map[string]string{
		{"value": "pt", "label": "Portugal", "group": "Europe"},
		{"value": "us", "label": "United States", "group": "Americas"},
		{"value": "fr", "label": "France", "group": "Europe"},
	}},
	{"a run that ends the list", []map[string]string{
		{"value": "any", "label": "Pick one"},
		{"value": "pt", "label": "Portugal", "group": "Europe"},
		{"value": "fr", "label": "France", "group": "Europe"},
	}},
	{"a run that starts the list", []map[string]string{
		{"value": "pt", "label": "Portugal", "group": "Europe"},
		{"value": "any", "label": "Elsewhere"},
	}},
	{"disabled options", []map[string]string{
		{"value": "a", "label": "Aisle", "disabled": "true"},
		{"value": "b", "label": "Aisle"},
		{"value": "c", "label": "Window", "disabled": "false"},
	}},
	{"a disabled option inside a run", []map[string]string{
		{"value": "1", "label": "Row 1", "group": "Exit row", "disabled": "true"},
		{"value": "2", "label": "Row 2", "group": "Main cabin"},
	}},
	// core.SelectOption.GroupDisabled. The declaration is on the *last* option
	// of the run on purpose: a transliteration that reads it when the run is
	// opened, rather than when it is closed, passes every case where the first
	// option happens to carry it.
	{"a run disabled by its last option", []map[string]string{
		{"value": "free", "label": "Free", "group": "Plans"},
		{"value": "pro", "label": "Pro", "group": "Paid", "groupDisabled": "true"},
		{"value": "max", "label": "Max", "group": "Paid", "groupDisabled": "true"},
	}},
	{"a run disabled by one option only", []map[string]string{
		{"value": "pro", "label": "Pro", "group": "Paid"},
		{"value": "max", "label": "Max", "group": "Paid", "groupDisabled": "true"},
	}},
	// The declaration does not leak past the run it was made in — the run
	// after it opens with a fresh state, which a transliteration that forgets
	// to reset gets wrong.
	{"a disabled run followed by a live one", []map[string]string{
		{"value": "pro", "label": "Pro", "group": "Paid", "groupDisabled": "true"},
		{"value": "free", "label": "Free", "group": "Free"},
	}},
	// No heading means no <optgroup> and no header, so the run's state has
	// nowhere to be drawn and degrades to "every option in it is disabled".
	{"a disabled run with no heading", []map[string]string{
		{"value": "a", "label": "A", "groupDisabled": "true"},
		{"value": "b", "label": "B"},
	}},
	// A disabled run whose own options also disagree: an option already
	// disabled stays disabled, and one that was not becomes so. Neither
	// direction can flip back.
	{"a disabled run over a mixed pair", []map[string]string{
		{"value": "a", "label": "A", "group": "G", "groupDisabled": "true", "disabled": "true"},
		{"value": "b", "label": "B", "group": "G", "disabled": "false"},
	}},
	// The label default lives at core.Select's flattening seam; a
	// hand-assembled node can carry a map that never went through it.
	{"a label falling back to the value", []map[string]string{
		{"value": "42"},
	}},
	{"no options at all", []map[string]string{}},
}

// Cases returns the table with each case's expected menu computed by
// core.SelectMenuSections.
//
// Both slices are allocated empty rather than left nil. A nil slice marshals
// to JSON `null`, and Swift's Decodable refuses a null where a non-optional
// array is declared — so the empty-list case would bring a harness down on a
// decode error rather than reporting a menu of no sections.
func Cases() []Case {
	cases := make([]Case, 0, len(lists))
	for _, l := range lists {
		c := Case{Name: l.name, Options: l.options, Want: []Section{}}
		for _, section := range core.SelectMenuSections(l.options) {
			w := Section{
				Heading:  section.Heading,
				First:    section.First(),
				Disabled: section.Disabled,
				Items:    make([]Item, 0, len(section.Items)),
			}
			for _, item := range section.Items {
				w.Items = append(w.Items, Item{
					Index:    item.Index,
					Value:    item.Value,
					Label:    item.Label,
					Disabled: item.Disabled,
				})
			}
			c.Want = append(c.Want, w)
		}
		cases = append(cases, c)
	}
	return cases
}
