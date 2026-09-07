package core

import (
	"reflect"
	"testing"
)

// SelectMenuSections is the one statement of how a picker's flat option list
// becomes the menu a person sees, and four renderers rest on it: htmlout calls
// it, and the WASM runtime, Renderer.swift and Renderer.kt each carry a
// transliteration that is checked against it. So the cases here are also the
// specification those three are held to.

// The rule core.SelectOption.Group states: runs, not a gather.
func TestSelectMenuSectionsAreRunsAndNotAGather(t *testing.T) {
	// "Europe" appears twice with an American option between, which the field
	// explicitly allows and which a gather would silently collapse into one
	// section — reordering the caller's list, which is what a person sees and
	// what the keyboard walks.
	got := SelectMenuSections([]map[string]string{
		{"value": "pt", "label": "Portugal", "group": "Europe"},
		{"value": "us", "label": "United States", "group": "Americas"},
		{"value": "fr", "label": "France", "group": "Europe"},
	})

	if len(got) != 3 {
		t.Fatalf("%d sections, want 3 — the same heading either side of a different one "+
			"is two sections, not one:\n%+v", len(got), got)
	}
	for i, want := range []string{"Europe", "Americas", "Europe"} {
		if got[i].Heading != want {
			t.Errorf("section %d heading = %q, want %q", i, got[i].Heading, want)
		}
		if len(got[i].Items) != 1 {
			t.Errorf("section %d has %d items, want 1", i, len(got[i].Items))
		}
	}
	// The order is the caller's, all the way down to the index each option
	// keeps.
	for i, want := range []string{"pt", "us", "fr"} {
		if got[i].Items[0].Value != want || got[i].Items[0].Index != i {
			t.Errorf("section %d holds %+v, want value %q at index %d",
				i, got[i].Items[0], want, i)
		}
	}
}

// A run that ends the list is the case each renderer's own copy had to
// remember, and the one htmlout's copy got wrong in a way its fixture could
// not see: the fixture's last option was ungrouped, so the explicit flush was
// never reached.
func TestSelectMenuSectionsFlushARunThatEndsTheList(t *testing.T) {
	got := SelectMenuSections([]map[string]string{
		{"value": "any", "label": "Pick one"},
		{"value": "pt", "label": "Portugal", "group": "Europe"},
		{"value": "fr", "label": "France", "group": "Europe"},
	})

	if len(got) != 2 {
		t.Fatalf("%d sections, want 2:\n%+v", len(got), got)
	}
	if got[0].Heading != "" || len(got[0].Items) != 1 {
		t.Errorf("the leading ungrouped option is not its own section: %+v", got[0])
	}
	// The whole point: the trailing run holds both of its options rather than
	// one, or none.
	if got[1].Heading != "Europe" || len(got[1].Items) != 2 {
		t.Errorf("the trailing run = %+v, want Europe with 2 options", got[1])
	}
}

// The ungrouped options are a section too, not an absence of one.
func TestSelectMenuSectionsGiveUngroupedOptionsASectionOfTheirOwn(t *testing.T) {
	got := SelectMenuSections([]map[string]string{
		{"value": "a", "label": "A"},
		{"value": "b", "label": "B"},
	})

	// One section, not two and not zero: a renderer draws sections in a single
	// loop, and an empty heading is what says "no wrapper" rather than "no
	// section".
	if len(got) != 1 || got[0].Heading != "" || len(got[0].Items) != 2 {
		t.Fatalf("sections = %+v, want one headingless section of 2", got)
	}
}

// Disabled crosses the wire as the string "true", core.SelectedState's
// spelling, and nothing else counts.
func TestSelectMenuSectionsReadTheDisabledSpelling(t *testing.T) {
	got := SelectMenuSections([]map[string]string{
		{"value": "a", "label": "A", "disabled": "true"},
		{"value": "b", "label": "B", "disabled": "false"},
		{"value": "c", "label": "C"},
	})

	want := []bool{true, false, false}
	for i, w := range want {
		if got[0].Items[i].Disabled != w {
			t.Errorf("option %d disabled = %v, want %v", i, got[0].Items[i].Disabled, w)
		}
	}
}

// First is what a renderer keys a section on, and the heading cannot be: two
// runs may share one.
func TestSelectMenuSectionFirstIsTheStartingIndex(t *testing.T) {
	got := SelectMenuSections([]map[string]string{
		{"value": "pt", "group": "Europe"},
		{"value": "us", "group": "Americas"},
		{"value": "fr", "group": "Europe"},
	})

	seen := map[int]bool{}
	for _, s := range got {
		if seen[s.First()] {
			t.Errorf("two sections start at index %d — the key is not unique", s.First())
		}
		seen[s.First()] = true
	}
	if got[2].First() != 2 {
		t.Errorf("the third section starts at %d, want 2", got[2].First())
	}
	// A section with no items cannot occur, so the guard in First is
	// unreachable through this function — stated here so a future change that
	// makes empty sections possible fails on the claim rather than on a panic.
	for _, s := range got {
		if len(s.Items) == 0 {
			t.Errorf("section %q is empty; SelectMenuSections opens one only for an option", s.Heading)
		}
	}
}

// The label default lives at core.Select's flattening seam, not here — but a
// hand-assembled node can carry a map that never went through it.
func TestSelectMenuSectionsFallBackToTheValueForALabel(t *testing.T) {
	got := SelectMenuSections([]map[string]string{{"value": "42"}})
	if got[0].Items[0].Label != "42" {
		t.Errorf("label = %q, want the value — a menu row with no text is worse than "+
			"one showing its value", got[0].Items[0].Label)
	}
}

// An empty list is an empty slice, not nil, so every renderer can range over
// the result with no guard.
func TestSelectMenuSectionsOfNothingIsRangeable(t *testing.T) {
	got := SelectMenuSections(nil)
	if got == nil {
		t.Fatal("nil options gave a nil slice")
	}
	if !reflect.DeepEqual(got, []SelectMenuSection{}) {
		t.Errorf("sections = %+v, want an empty slice", got)
	}
}

// core.SelectOption.GroupDisabled: the run, not the option.

func TestSelectMenuSectionsTakeADisabledRunFromAnyOptionInIt(t *testing.T) {
	// The declaration is on the run's *last* option, which is the case that
	// separates "read it when the run is closed" from "read it when the run is
	// opened". The second reading passes every list whose first option happens
	// to carry the flag, so a fixture that only ever writes it there would
	// certify a transliteration nobody could rely on.
	got := SelectMenuSections([]map[string]string{
		{"value": "pro", "label": "Pro", "group": "Paid"},
		{"value": "max", "label": "Max", "group": "Paid", "groupDisabled": "true"},
	})

	if len(got) != 1 {
		t.Fatalf("%d sections, want 1: %+v", len(got), got)
	}
	if !got[0].Disabled {
		t.Error("the run is not disabled — a declaration made on the second option " +
			"was read when the run was opened, or not read at all")
	}
	// The propagation is the half two of the four targets depend on entirely:
	// SwiftUI puts .disabled on the Button, Material's dropdown has no section
	// construct, and neither can read a flag that stopped at the section.
	for i, item := range got[0].Items {
		if !item.Disabled {
			t.Errorf("item %d (%q) is choosable inside a disabled run", i, item.Value)
		}
	}
}

func TestADisabledRunDoesNotReachTheNextOne(t *testing.T) {
	// The state is per-run, so opening the next one has to clear it. A
	// transliteration that hoists the flag out of the loop passes every case
	// where the disabled run is last.
	got := SelectMenuSections([]map[string]string{
		{"value": "pro", "label": "Pro", "group": "Paid", "groupDisabled": "true"},
		{"value": "free", "label": "Free", "group": "Free"},
	})

	if len(got) != 2 {
		t.Fatalf("%d sections, want 2: %+v", len(got), got)
	}
	if !got[0].Disabled {
		t.Error("the declared run is not disabled")
	}
	if got[1].Disabled {
		t.Error("the run after a disabled one is disabled too — the flag outlived its run")
	}
	if got[1].Items[0].Disabled {
		t.Error("an option after a disabled run is not choosable")
	}
}

func TestADisabledRunNeverReEnablesAnOption(t *testing.T) {
	// Disabling flows one way. An option that was already refused stays
	// refused, and the run's declaration cannot be undone by an option's own
	// "false" — which is the spelling core.Select never writes but a
	// hand-assembled node can.
	got := SelectMenuSections([]map[string]string{
		{"value": "a", "label": "A", "group": "G", "groupDisabled": "true", "disabled": "true"},
		{"value": "b", "label": "B", "group": "G", "disabled": "false"},
	})

	if len(got) != 1 {
		t.Fatalf("%d sections, want 1: %+v", len(got), got)
	}
	for i, item := range got[0].Items {
		if !item.Disabled {
			t.Errorf("item %d (%q) is choosable inside a disabled run", i, item.Value)
		}
	}
}

func TestAnUngroupedRunCanStillBeDisabled(t *testing.T) {
	// There is no heading to grey and, on the web, no <optgroup> to carry the
	// attribute — so the declaration degrades to exactly "every option in the
	// run is disabled". That is a real outcome rather than a special case, and
	// it is the reason the refusal rides on the items everywhere.
	got := SelectMenuSections([]map[string]string{
		{"value": "a", "label": "A", "groupDisabled": "true"},
		{"value": "b", "label": "B"},
	})

	if len(got) != 1 || got[0].Heading != "" {
		t.Fatalf("want one headingless section, got %+v", got)
	}
	if !got[0].Disabled {
		t.Error("an ungrouped run cannot be disabled")
	}
	for i, item := range got[0].Items {
		if !item.Disabled {
			t.Errorf("item %d (%q) is choosable inside a disabled run", i, item.Value)
		}
	}
}

func TestGroupDisabledIsSpelledLikeEveryOtherWireBool(t *testing.T) {
	// "true" and nothing else, the spelling core.SelectedState uses and the
	// one the DOM writes. Anything else is not a declaration — the same
	// reading the "disabled" key gets one line over.
	for _, spelling := range []string{"", "false", "1", "TRUE", "yes"} {
		got := SelectMenuSections([]map[string]string{
			{"value": "a", "group": "G", "groupDisabled": spelling},
		})
		if got[0].Disabled {
			t.Errorf("groupDisabled=%q disabled the run; only \"true\" should", spelling)
		}
	}
}
