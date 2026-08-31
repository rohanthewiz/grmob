package forms

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Rule reports what is wrong with a field's value, or "" when nothing is.
//
// A rule is a plain function, not an interface: every rule below is a closure
// over its own parameters, and an app's own checks ("this username is
// reserved", "this date is in the past") are written inline with no type to
// implement.
//
//	{Name: "handle", Rules: []forms.Rule{
//	    forms.Required(""),
//	    func(v string) string {
//	        if reserved[strings.ToLower(v)] {
//	            return "That handle is taken by the system"
//	        }
//	        return ""
//	    },
//	}}
//
// A rule must be pure. It is evaluated on every read of the form's errors —
// several times per render pass — so it must not have side effects, must not
// mutate what it is given, and must not perform I/O. (It runs outside the
// form's lock, so it *may* safely read the form it belongs to, but a rule
// that needs another field's value belongs in Spec.Validate instead.)
type Rule func(value string) string

// msgOr lets every rule take its message as an argument while still being
// usable with "": the caller owns the app's voice and its localization, and
// this package has no business shipping copy — but a prototype should not
// have to invent nine strings before it can see a form work.
func msgOr(msg, fallback string) string {
	if msg != "" {
		return msg
	}
	return fallback
}

// optional lifts a check into a rule that says nothing about an empty value.
//
// Every rule below except Required and Accepted is wrapped in this, because
// emptiness is Required's subject and nobody else's. Without it an optional
// field carrying MinLen(8) would report "Must be at least 8 characters"
// before the user has typed anything — and a *required* field carrying both
// would have two opinions about the same empty string, of which FormField can
// only show one.
//
// The value is tested raw rather than trimmed: a field of nothing but spaces
// is not empty to MaxLen or Pattern, and Required is the rule that decides
// whitespace does not count as content.
func optional(check func(string) string) Rule {
	return func(v string) string {
		if v == "" {
			return ""
		}
		return check(v)
	}
}

// Required rejects a value that is empty or nothing but whitespace. It is the
// only rule that speaks about emptiness; see optional.
func Required(msg string) Rule {
	return func(v string) string {
		if strings.TrimSpace(v) == "" {
			return msgOr(msg, "Required")
		}
		return ""
	}
}

// MinLen requires at least n characters — runes, not bytes, so "héllo" is
// five and a name in a non-Latin script is not silently held to a longer
// standard than an ASCII one.
func MinLen(n int, msg string) Rule {
	return optional(func(v string) string {
		if utf8.RuneCountInString(v) < n {
			return msgOr(msg, fmt.Sprintf("Must be at least %d characters", n))
		}
		return ""
	})
}

// MaxLen allows at most n characters (runes; see MinLen).
//
// A max is a display-time complaint, not a keystroke filter: the field still
// accepts the extra characters and the user sees why they are too many. A
// hard cap belongs on the native control, which grmob does not expose — and
// silently dropping keystrokes is the worse behavior anyway.
func MaxLen(n int, msg string) Rule {
	return optional(func(v string) string {
		if utf8.RuneCountInString(v) > n {
			return msgOr(msg, fmt.Sprintf("Must be at most %d characters", n))
		}
		return ""
	})
}

// emailShape is deliberately not an RFC 5322 grammar. The addresses that
// grammar admits and this pattern rejects (quoted local parts, comments,
// bracketed IP domains) are not what a user typos, and the ones it rejects
// that this admits are caught by the only check that actually settles the
// question: sending mail to it. What a client-side rule is for is catching
// "you@example" and "you example.com" before a round trip.
var emailShape = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// Email checks that a value has the shape of an address. See emailShape for
// what this rule does and does not claim.
func Email(msg string) Rule {
	return optional(func(v string) string {
		if !emailShape.MatchString(strings.TrimSpace(v)) {
			return msgOr(msg, "Not a valid email address")
		}
		return ""
	})
}

// Pattern requires the value to match re.
//
// It takes a compiled *regexp.Regexp rather than an expression string on
// purpose. A Spec is rebuilt on every render pass, so a rule that compiled
// its own pattern would run regexp.Compile on every pass of every form —
// and, with MustCompile, would turn a typo in a pattern into a panic on a
// render goroutine rather than at startup. A package-level var compiles once:
//
//	var postcode = regexp.MustCompile(`^[0-9]{5}$`)
//	...
//	forms.Pattern(postcode, "Five digits")
func Pattern(re *regexp.Regexp, msg string) Rule {
	return optional(func(v string) string {
		if re == nil || !re.MatchString(v) {
			return msgOr(msg, "Not in the expected format")
		}
		return ""
	})
}

// Integer requires a whole number.
//
// Pair it with core.Input, not core.NumericInput: NumericInput's change
// callback parses with strconv.Atoi and *drops the event* when the text does
// not parse, so the unparseable value never reaches the form and this rule
// can never fire. A validated numeric field is a text field with a rule.
func Integer(msg string) Rule {
	return optional(func(v string) string {
		if _, err := strconv.Atoi(strings.TrimSpace(v)); err != nil {
			return msgOr(msg, "Must be a whole number")
		}
		return ""
	})
}

// Range requires a whole number between lo and hi inclusive. A value that is
// not a number at all fails this rule too, so Range alone is enough for a
// bounded numeric field; adding Integer before it only changes which of the
// two messages an unparseable value gets.
func Range(lo, hi int, msg string) Rule {
	return optional(func(v string) string {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil || n < lo || n > hi {
			return msgOr(msg, fmt.Sprintf("Must be between %d and %d", lo, hi))
		}
		return ""
	})
}

// OneOf restricts a value to a fixed set — a picker or segmented control
// whose selection is carried as text.
//
// The message comes first here, against the convention of every other rule in
// this file, because Go requires the variadic parameter to be last. The
// alternative was a []string parameter, which would read
// forms.OneOf([]string{"card", "bank"}, "") at every call site.
func OneOf(msg string, allowed ...string) Rule {
	return optional(func(v string) string {
		for _, a := range allowed {
			if v == a {
				return ""
			}
		}
		return msgOr(msg, "Not an allowed value")
	})
}

// Accepted requires a checkbox to be ticked — the terms-of-service field.
//
// Not wrapped in optional: an unticked box is "false", not "", so there is no
// empty case to skip, and treating it as one would make the rule silent
// exactly when it matters.
func Accepted(msg string) Rule {
	return func(v string) string {
		if !isTrue(v) {
			return msgOr(msg, "Must be accepted")
		}
		return ""
	}
}
