package comps

import (
	"strings"
	"unicode"
)

// The mask language MaskedInput formats with, as two pure functions: one from
// what a field holds to the characters the reader meant (unmask), one from
// those characters to what the field should show (applyMask). Kept apart from
// the widget, as TagInput's splitDraft is, so the rule has a table test that
// needs no render pass.
//
//	mask        "(###) ###-####"
//	field text  "(5556"             what the host sent after the fourth key
//	unmask      "5556"              the raw value: slot characters only
//	applyMask   "(555) 6"           what Go renders back
//
// # The three slot characters
//
//	#   a digit
//	A   a letter
//	*   a letter or a digit
//
// Every other character of a mask is a literal, drawn as written.
//
// # Literals are written late
//
// A literal is written only when a raw character follows it. "555" formats as
// "(555" and not "(555) ", and the ") " arrives with the fourth digit.
//
// The eager form reads better for a moment and cannot be edited: a backspace
// on "(555) " gives "(555)", which unmasks to the same three digits and is
// formatted straight back to "(555) ". The reader would be pressing backspace
// against a wall. Written late, the text never ends in a literal, so the last
// character is always one a backspace can remove. (A mask's leading literal
// is covered by the same rule: it appears with the first raw character and
// leaves with it.)

// maskSlot reports whether a mask character is a slot, and if so which
// characters it accepts.
func maskSlot(m rune) (accepts func(rune) bool, ok bool) {
	switch m {
	case '#':
		return unicode.IsDigit, true
	case 'A':
		return unicode.IsLetter, true
	case '*':
		return func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }, true
	}
	return nil, false
}

// maskCapacity is the number of slots in a mask: the longest raw value it can
// hold.
func maskCapacity(mask string) int {
	n := 0
	for _, m := range mask {
		if _, ok := maskSlot(m); ok {
			n++
		}
	}
	return n
}

// applyMask formats raw through mask. A raw character that does not fit the
// slot it would land in is dropped, and raw beyond the mask's last slot is
// cut, so the result is always a prefix of a fully formatted value.
func applyMask(mask, raw string) string {
	pending := []rune(raw)
	var out strings.Builder
	// Literals met since the last slot, held back until a raw character
	// proves they are needed. See "Literals are written late".
	var held []rune
	for _, m := range mask {
		accepts, slot := maskSlot(m)
		if !slot {
			held = append(held, m)
			continue
		}
		// Skip raw characters this slot refuses. Dropping rather than
		// stopping means a stray letter pasted into a phone number costs
		// that letter and not everything after it.
		for len(pending) > 0 && !accepts(pending[0]) {
			pending = pending[1:]
		}
		if len(pending) == 0 {
			break
		}
		out.WriteString(string(held))
		held = held[:0]
		out.WriteRune(pending[0])
		pending = pending[1:]
	}
	return out.String()
}

// unmask reads the raw value out of a field's text, which may be a formatted
// value, a formatted value with a key typed or deleted anywhere in it, or a
// paste that never saw the mask.
//
// It walks the text and the mask together. For each character of the text:
//
//	the mask is at a literal, and the character is that literal
//	    → it is the literal; both advance
//	the mask is at a literal, and the character is something else
//	    → the literal was not typed (applyMask will write it); the mask
//	      advances and the character is tried against what follows
//	the mask is at a slot that accepts the character
//	    → raw takes it; both advance
//	the mask is at a slot that refuses it
//	    → the character is dropped; the mask stays
//
// The walk is what lets a mask carry a literal its own slots could hold, as
// "+1 (###) ###-####" does: the "1" of the formatted text is matched as the
// literal before any slot sees it, where stripping non-digits would have read
// it as the first digit of the number and grown the value by one "1" on every
// keystroke. The price is stated in MaskedInput's doc: a raw value cannot
// begin with a character equal to such a literal.
//
// Once the mask is exhausted the rest of the text is ignored.
func unmask(mask, text string) string {
	m := []rune(mask)
	mi := 0
	var raw strings.Builder
	for _, c := range text {
		// Pass over literals until this character is placed or matched.
		matchedLiteral := false
		for mi < len(m) {
			if _, slot := maskSlot(m[mi]); slot {
				break
			}
			lit := m[mi]
			mi++
			if lit == c {
				matchedLiteral = true
				break
			}
		}
		if matchedLiteral {
			continue
		}
		if mi >= len(m) {
			break
		}
		if accepts, _ := maskSlot(m[mi]); accepts(c) {
			raw.WriteRune(c)
			mi++
		}
	}
	return raw.String()
}
