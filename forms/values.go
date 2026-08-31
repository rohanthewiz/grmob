package forms

import (
	"strconv"
	"strings"
)

// Values is a form's field set: name to raw text.
//
// Strings all the way down, deliberately. Every event the framework carries
// from a native control is a string on the wire — core.NumericInput already
// stores an int and ships it through registerTextCallback, parsing on the way
// back in — so a form that kept typed values would be converting twice and
// would need a heterogeneous map to hold them. Keeping the raw text is also
// what makes validation possible at all: "12x" has to survive long enough for
// Integer to complain about it, and a map[string]int has nowhere to put it.
//
// Values is a plain map type, so ordinary indexing is the way to read a
// field: v["email"]. The methods below are for the values that are not text.
type Values map[string]string

// Trimmed reads a field with leading and trailing space removed. Required
// trims before deciding a field is empty, so a value that passed validation
// may still be padded; a submit handler that stores what it was given
// generally wants this rather than the raw text.
func (v Values) Trimmed(name string) string {
	return strings.TrimSpace(v[name])
}

// Bool reads a checkbox field. Anything strconv.ParseBool does not accept as
// true — including an absent field and any free text — reads as false.
//
// No ok result, unlike Int: a checkbox has two states and "not checked" is a
// complete answer for every value that is not "true". The form writes these
// values itself (OnToggle formats with strconv.FormatBool), so the lossy case
// cannot arise from user input.
func (v Values) Bool(name string) bool {
	return isTrue(v[name])
}

// isTrue is the single definition of "checked" in this package: OnToggle
// writes with strconv.FormatBool, Values.Bool reads, and the Accepted rule
// judges — all three have to agree, and a rule that spelled the test itself
// would be free to drift from the writer.
func isTrue(s string) bool {
	b, err := strconv.ParseBool(strings.TrimSpace(s))
	return err == nil && b
}

// Int reads a numeric field, reporting whether it parsed. The ok result is
// real here — the field is free text a user typed, and "" and "twelve" are
// both reachable — so a caller either pairs the field with an Integer rule
// and can ignore ok inside a submit handler, or checks it.
func (v Values) Int(name string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(v[name]))
	return n, err == nil
}

// Float reads a decimal field, reporting whether it parsed. See Int.
func (v Values) Float(name string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(v[name]), 64)
	return f, err == nil
}

// Clone returns an independent copy. The form hands a clone to every rule
// evaluation and to every submit handler, so neither can reach back into the
// live map — a handler that spawns a goroutine (the usual shape for a network
// submit) would otherwise be reading a map the user is still typing into.
func (v Values) Clone() Values {
	out := make(Values, len(v))
	for k, val := range v {
		out[k] = val
	}
	return out
}
