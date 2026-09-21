package comps

import "testing"

const (
	phoneMask   = "(###) ###-####"
	cardMask    = "#### #### #### ####"
	intlMask    = "+1 (###) ###-####"
	plateMask   = "AAA-###"
	serialMask  = "**-**"
	expiryMask  = "##/##"
	noSlotsMask = "--"
)

func TestApplyMask(t *testing.T) {
	cases := []struct{ name, mask, raw, want string }{
		{"nothing typed draws nothing, not the leading literal", phoneMask, "", ""},
		{"the leading literal arrives with the first digit", phoneMask, "5", "(5"},
		{"a group's closing literals are written late", phoneMask, "555", "(555"},
		{"and arrive with the next digit", phoneMask, "5556", "(555) 6"},
		{"a full value", phoneMask, "5551234567", "(555) 123-4567"},
		{"raw past the last slot is cut", phoneMask, "55512345678", "(555) 123-4567"},
		{"a character the slot refuses is dropped, not the rest", phoneMask, "55a51", "(555) 1"},
		{"letters then digits", plateMask, "abc123", "abc-123"},
		{"a digit where a letter belongs is dropped", plateMask, "1ab", "ab"},
		{"either", serialMask, "a1b2", "a1-b2"},
		{"a literal a slot could hold", intlMask, "555", "+1 (555"},
		{"a mask with no slots formats to nothing", noSlotsMask, "12", ""},
		{"no mask at all", "", "12", ""},
		{"non-ASCII digits and letters are digits and letters", "A#", "é٣", "é٣"},
	}
	for _, c := range cases {
		if got := applyMask(c.mask, c.raw); got != c.want {
			t.Errorf("%s: applyMask(%q, %q) = %q, want %q", c.name, c.mask, c.raw, got, c.want)
		}
	}
}

func TestUnmask(t *testing.T) {
	cases := []struct{ name, mask, text, want string }{
		{"a formatted value", phoneMask, "(555) 123-4567", "5551234567"},
		{"a key typed at the end", phoneMask, "(5556", "5556"},
		{"a key typed into an empty field", phoneMask, "5", "5"},
		{"a paste that never saw the mask", phoneMask, "555.123.4567", "5551234567"},
		{"a paste with its own punctuation", phoneMask, "555-123-4567", "5551234567"},
		{"a key typed mid-text", phoneMask, "(5955) 123", "5955123"},
		{"a digit deleted mid-text", phoneMask, "(55) 123-4567", "551234567"},
		{"a literal deleted: nothing was lost", phoneMask, "(555 123", "555123"},
		{"a refused key", phoneMask, "(55x", "55"},
		{"text past the mask is ignored", expiryMask, "12/3456", "1234"},
		// The case the walk exists for: stripping non-digits would read the
		// literal 1 as data and grow the value on every keystroke.
		{"a literal a slot could hold is the literal", intlMask, "+1 (555", "555"},
		{"and again after a round trip", intlMask, "+1 (555) 1", "5551"},
		{"typed bare, a leading 1 is read as that literal", intlMask, "1555", "555"},
		{"card groups", cardMask, "4111 1111 2", "411111112"},
	}
	for _, c := range cases {
		if got := unmask(c.mask, c.text); got != c.want {
			t.Errorf("%s: unmask(%q, %q) = %q, want %q", c.name, c.mask, c.text, got, c.want)
		}
	}
}

// The property the widget depends on: what Go renders, read back, is what Go
// held. Without it a field would drift on every echo.
func TestMaskRoundTrips(t *testing.T) {
	for _, c := range []struct{ mask, raw string }{
		{phoneMask, ""}, {phoneMask, "5"}, {phoneMask, "555"}, {phoneMask, "5551234567"},
		{cardMask, "41111111"}, {intlMask, "5551"}, {plateMask, "abc12"}, {serialMask, "a1b"},
		{expiryMask, "123"},
	} {
		shown := applyMask(c.mask, c.raw)
		if got := unmask(c.mask, shown); got != c.raw {
			t.Errorf("mask %q: %q draws %q, which reads back %q", c.mask, c.raw, shown, got)
		}
	}
}

// A backspace must always change the raw value while there is one: the wall
// that eager literals build ("(555) " ⌫ "(555)" → "(555) ").
func TestMaskBackspaceAlwaysBites(t *testing.T) {
	raw := "5551234567"
	for len(raw) > 0 {
		shown := []rune(applyMask(phoneMask, raw))
		after := unmask(phoneMask, string(shown[:len(shown)-1]))
		if len(after) != len(raw)-1 {
			t.Fatalf("backspace on %q left raw %q (was %q)", string(shown), after, raw)
		}
		raw = after
	}
}

func TestMaskCapacity(t *testing.T) {
	for mask, want := range map[string]int{phoneMask: 10, cardMask: 16, plateMask: 6, noSlotsMask: 0, "": 0} {
		if got := maskCapacity(mask); got != want {
			t.Errorf("maskCapacity(%q) = %d, want %d", mask, got, want)
		}
	}
}
