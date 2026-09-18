package rebasefixture

import "testing"

// Every case's answer, written out. Cases computes Want from Rebase, so this
// is the check on Rebase itself: the harnesses only prove the hosts agree
// with it.
func TestTheCasesSayWhatTheyMean(t *testing.T) {
	want := map[string]struct {
		text  string
		caret int
	}{
		"typed at the end, draft cleared (TagInput)":     {"gam", 3},
		"typed at the start":                             {"xAB", 1},
		"nothing typed since":                            {"HELLO", 5},
		"a PIN cell Go filled under a typed key":         {"14", 2},
		"typed at the end, rewrite appended":             {"ab!c", 4},
		"mid-text under UPPERCASE":                       {"HELLOAbWORLD", 7},
		"mid-text, a run of keys under UPPERCASE":        {"HELLOAbcdWORLD", 9},
		"mid-text before Go's change":                    {"onex TWO", 4},
		"mid-text after Go's change":                     {"ONE twox", 8},
		"a deletion replayed":                            {"Hell", 4},
		"a replacement replayed":                         {"Cat mat", 5},
		"typing inside text Go replaced with other text": {"", 0},
		"typing inside a rewrite of other length":        {"bye", 3},
		"typing beside a space Go collapsed":             {"a xb", 3},
		"an emoji typed after an emoji":                  {"😀!😁", 5},
		"an emoji replaced by one sharing its high half": {"X😁", 3},
	}
	cs := Cases()
	if len(cs) != len(want) {
		t.Fatalf("%d cases, %d answers written out", len(cs), len(want))
	}
	for _, c := range cs {
		w, ok := want[c.Name]
		if !ok {
			t.Errorf("no answer written out for %q", c.Name)
			continue
		}
		if c.Want != w.text || c.WantCaret != w.caret {
			t.Errorf("%s: got %q caret %d, want %q caret %d", c.Name, c.Want, c.WantCaret, w.text, w.caret)
		}
	}
}

// A replayed span is whole characters: nothing Rebase returns for text that
// was valid UTF-16 may hold a lone surrogate.
func TestNoSurrogateIsSplit(t *testing.T) {
	texts := []string{"", "a", "😀", "😁", "a😀", "😀a", "😀😁", "x😀y", "𝄞", "é"}
	for _, b := range texts {
		for _, l := range texts {
			for _, r := range texts {
				out := Rebase(b, l, r)
				if string([]rune(out)) != out {
					t.Fatalf("Rebase(%q, %q, %q) = %q is not valid UTF-8", b, l, r, out)
				}
				for _, rr := range out {
					if rr == 0xFFFD {
						t.Fatalf("Rebase(%q, %q, %q) = %q holds a replacement character", b, l, r, out)
					}
				}
				if c := Caret(b, l, r, len(units(l))); c < 0 || c > len(units(out)) {
					t.Fatalf("Caret(%q, %q, %q) = %d is outside %q", b, l, r, c, out)
				}
			}
		}
	}
}
