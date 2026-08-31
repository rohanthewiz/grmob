package forms_test

import (
	"regexp"
	"testing"

	"github.com/rohanthewiz/grmob/forms"
)

// Each rule is a pure function, so these are ordinary table tests: value in,
// message out, "" meaning "nothing wrong".

func TestRulesAcceptAndReject(t *testing.T) {
	digits := regexp.MustCompile(`^[0-9]{5}$`)

	cases := []struct {
		name  string
		rule  forms.Rule
		value string
		want  string // "" = accepted; otherwise the exact message
	}{
		// Required is the only rule with an opinion about emptiness, and
		// whitespace does not count as content.
		{"required empty", forms.Required("Need it"), "", "Need it"},
		{"required spaces", forms.Required("Need it"), "   ", "Need it"},
		{"required tab/newline", forms.Required("Need it"), "\t\n", "Need it"},
		{"required ok", forms.Required("Need it"), "x", ""},

		// Runes, not bytes: "héllo" is 6 bytes and 5 characters.
		{"minlen short", forms.MinLen(5, "Too short"), "abcd", "Too short"},
		{"minlen exact", forms.MinLen(5, "Too short"), "héllo", ""},
		{"maxlen over", forms.MaxLen(3, "Too long"), "abcd", "Too long"},
		{"maxlen exact", forms.MaxLen(3, "Too long"), "héo", ""},

		{"email plain", forms.Email("Bad"), "you@example.com", ""},
		{"email padded", forms.Email("Bad"), "  you@example.com  ", ""},
		{"email no domain dot", forms.Email("Bad"), "you@example", "Bad"},
		{"email no at", forms.Email("Bad"), "you example.com", "Bad"},
		{"email space inside", forms.Email("Bad"), "yo u@example.com", "Bad"},

		{"pattern ok", forms.Pattern(digits, "Five digits"), "12345", ""},
		{"pattern no", forms.Pattern(digits, "Five digits"), "1234", "Five digits"},
		// A nil regexp rejects rather than panics or silently accepts: a
		// forgotten var is a bug the form should surface, not swallow.
		{"pattern nil", forms.Pattern(nil, "Five digits"), "12345", "Five digits"},

		{"integer ok", forms.Integer("NaN"), " 42 ", ""},
		{"integer negative", forms.Integer("NaN"), "-7", ""},
		{"integer text", forms.Integer("NaN"), "12x", "NaN"},
		{"integer decimal", forms.Integer("NaN"), "1.5", "NaN"},

		{"range low", forms.Range(1, 10, "1-10"), "0", "1-10"},
		{"range lo edge", forms.Range(1, 10, "1-10"), "1", ""},
		{"range hi edge", forms.Range(1, 10, "1-10"), "10", ""},
		{"range high", forms.Range(1, 10, "1-10"), "11", "1-10"},
		// Range subsumes Integer: unparseable text fails it too.
		{"range text", forms.Range(1, 10, "1-10"), "five", "1-10"},

		{"oneof ok", forms.OneOf("Nope", "card", "bank"), "bank", ""},
		{"oneof no", forms.OneOf("Nope", "card", "bank"), "cash", "Nope"},
		{"oneof empty set", forms.OneOf("Nope"), "card", "Nope"},

		// Accepted is not empty-skipped: an unticked box is "false", and the
		// rule has to speak about exactly that case.
		{"accepted true", forms.Accepted("Tick it"), "true", ""},
		{"accepted false", forms.Accepted("Tick it"), "false", "Tick it"},
		{"accepted empty", forms.Accepted("Tick it"), "", "Tick it"},
		{"accepted garbage", forms.Accepted("Tick it"), "yes please", "Tick it"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.rule(c.value); got != c.want {
				t.Errorf("rule(%q) = %q, want %q", c.value, got, c.want)
			}
		})
	}
}

// The rule that makes an optional field possible: everything but Required and
// Accepted stays silent about "". Without it, a field the user has not
// reached yet would already be complaining, and a required field would have
// two opinions about one empty string where FormField can show only one.
func TestNonRequiredRulesSkipEmptyValues(t *testing.T) {
	rules := map[string]forms.Rule{
		"MinLen":  forms.MinLen(8, "Too short"),
		"MaxLen":  forms.MaxLen(3, "Too long"),
		"Email":   forms.Email("Bad"),
		"Pattern": forms.Pattern(regexp.MustCompile(`^[0-9]+$`), "Digits"),
		"Integer": forms.Integer("NaN"),
		"Range":   forms.Range(1, 10, "1-10"),
		"OneOf":   forms.OneOf("Nope", "card"),
	}
	for name, rule := range rules {
		if got := rule(""); got != "" {
			t.Errorf("%s(%q) = %q, want silence on an empty value", name, "", got)
		}
	}
}

// An empty message argument falls back to the rule's own English default, so
// a prototype does not have to invent nine strings — but every default must
// actually say something, or the field would silently pass.
func TestEmptyMessageFallsBackToADefault(t *testing.T) {
	failures := map[string]struct {
		rule  forms.Rule
		value string
	}{
		"Required": {forms.Required(""), ""},
		"MinLen":   {forms.MinLen(8, ""), "abc"},
		"MaxLen":   {forms.MaxLen(2, ""), "abc"},
		"Email":    {forms.Email(""), "nope"},
		"Pattern":  {forms.Pattern(regexp.MustCompile(`^z$`), ""), "a"},
		"Integer":  {forms.Integer(""), "x"},
		"Range":    {forms.Range(1, 3, ""), "9"},
		"OneOf":    {forms.OneOf("", "card"), "cash"},
		"Accepted": {forms.Accepted(""), "false"},
	}
	for name, c := range failures {
		if got := c.rule(c.value); got == "" {
			t.Errorf("%s with no message accepted %q; a failing rule must always say something", name, c.value)
		}
	}
}

// The number in the message is the number the caller passed. A default that
// hard-coded a bound would be wrong for every other bound.
func TestDefaultMessagesQuoteTheirBounds(t *testing.T) {
	if got := forms.MinLen(8, "")("a"); got != "Must be at least 8 characters" {
		t.Errorf("MinLen default = %q", got)
	}
	if got := forms.MaxLen(2, "")("abc"); got != "Must be at most 2 characters" {
		t.Errorf("MaxLen default = %q", got)
	}
	if got := forms.Range(3, 9, "")("1"); got != "Must be between 3 and 9" {
		t.Errorf("Range default = %q", got)
	}
}
