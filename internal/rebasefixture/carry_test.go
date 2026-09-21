package rebasefixture

import "testing"

// Every carry case's answer, written out, as TestTheCasesSayWhatTheyMean does
// for Rebase: CarryCases fills Want from Carry, so this is the check on Carry.
func TestTheCarryCasesSayWhatTheyMean(t *testing.T) {
	want := map[string]int{
		"a mask's leading literal arrives with the first key": 2,
		"a mask's group break arrives with the fourth key":    7,
		"a mask drops a key it refuses":                       3,
		// "(59|55) 666" → "(59|5) 566-6": the two texts share the prefix
		// "(59", so the caret is at the differing span's start and stays,
		// which is just after the key that was typed.
		"a key typed mid-mask reflows the text after it":   3,
		"UPPERCASE mid-text":                               6,
		"UPPERCASE at the end":                             5,
		"a committed TagInput draft":                       0,
		"the caret before a change stays":                  2,
		"the caret inside text replaced at another length": 4,
		"an emoji before the caret":                        3,
		"text inserted before an emoji and the caret":      3,
	}
	cs := CarryCases()
	if len(cs) != len(want) {
		t.Fatalf("%d cases, %d answers written out", len(cs), len(want))
	}
	for _, c := range cs {
		w, ok := want[c.Name]
		if !ok {
			t.Errorf("no answer written out for %q", c.Name)
			continue
		}
		if c.Want != w {
			t.Errorf("%s: Carry(%q, %q, %d) = %d, want %d", c.Name, c.Before, c.After, c.Caret, c.Want, w)
		}
	}
}
