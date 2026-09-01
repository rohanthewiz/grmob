package verify

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Shared machinery for reading a native renderer's dispatch on a prop value —
// Swift's `switch mode { case "…": }` and Kotlin's `when (mode) { "…" -> }`.
//
// Both languages write the same construct: a list of string-literal arms
// followed by a catch-all. The two syntaxes differ only in punctuation, so the
// difference is a dispatchSyntax value and everything downstream — locating
// the function, refusing to read past the catch-all, validating the label
// lists, failing loudly on anything unexpected — is written once.
//
// These parse rather than compile, and that is the point rather than a
// compromise. A native compiler cannot answer the question being asked here:
// `default` and `else` make a string switch exhaustive by construction, so
// "you forgot a mode" is not a type error in either language and never will
// be. The only thing that can notice is something holding the arms up against
// Go's list, which means reading them out of the source. Doing it in Go keeps
// the check inside `go test ./...`, where it runs for anyone with a Go
// toolchain and no Xcode, no Android SDK and no memory of a run.sh.
//
// The cost is a constraint on how those functions may be written: one arm per
// line, the string literals first on the line, and the catch-all last. That
// constraint is stated in a comment beside each of them, and every violation
// of it below is a named fatal rather than a short result — a rewrite is meant
// to land as a failure that says what happened, not as an empty comparison
// that agrees with everything.
type dispatchSyntax struct {
	// file is the path to read, relative to this package.
	file string
	// fn is the text that anchors the search to the right function. Anchoring
	// on the function rather than on a bare `switch` is what stops an
	// unrelated dispatch elsewhere in a 600-line renderer from being read as
	// this one.
	fn string
	// open is the switch header, including the name of the value being
	// switched on. Requiring the name is what proves the arms found below are
	// keyed by the prop this test is about and not by something else.
	open string
	// arm matches one labeled arm and captures its label list. Both are
	// written to require the line to begin with a string literal, which is
	// what keeps a comment or a pattern-matching arm from being read as a
	// label.
	arm *regexp.Regexp
	// fallback matches the catch-all arm. Arms are only collected from *above*
	// it: anything below is unreachable, and a test that counted it would
	// report coverage the renderer does not have.
	fallback *regexp.Regexp
	// fallbackDesc names the catch-all in failure messages, in the language's
	// own spelling.
	fallbackDesc string
}

var (
	swiftSwitch = dispatchSyntax{
		arm:          regexp.MustCompile(`(?m)^[ \t]*case[ \t]+("[^\n]*?)[ \t]*:[ \t]*$`),
		fallback:     regexp.MustCompile(`(?m)^[ \t]*default[ \t]*:`),
		fallbackDesc: "`default:`",
	}
	kotlinWhen = dispatchSyntax{
		arm:          regexp.MustCompile(`(?m)^[ \t]*("[^\n]*?)[ \t]*->`),
		fallback:     regexp.MustCompile(`(?m)^[ \t]*else[ \t]*->`),
		fallbackDesc: "`else ->`",
	}
)

// with fills in the per-check half of a syntax: which file, which function,
// which switch header. The punctuation halves above are language facts and are
// shared; these three are what a particular check is about.
func (d dispatchSyntax) with(file, fn, open string) dispatchSyntax {
	d.file, d.fn, d.open = file, fn, open
	return d
}

// labels returns the string labels of the described dispatch, in source order,
// having first proved it found the right one.
//
// Failure at every step rather than an empty slice, for the reason the replay
// harness grew a control mutant: a check that reads nothing must not be able to
// read as a pass. Renaming the function, switching on a different value, or
// deleting the catch-all all land here as fatals that name what changed.
func (d dispatchSyntax) labels(t *testing.T) []string {
	t.Helper()

	raw, err := os.ReadFile(d.file)
	if err != nil {
		t.Fatalf("reading %s: %v", d.file, err)
	}
	src := string(raw)

	fn := strings.Index(src, d.fn)
	if fn < 0 {
		t.Fatalf("%s: no %s found — if it was renamed or restructured, update this test", d.file, d.fn)
	}
	body := src[fn:]

	open := strings.Index(body, d.open)
	if open < 0 {
		t.Fatalf("%s: %s does not dispatch on %q — this test cannot tell which arms are the ones it "+
			"is about", d.file, d.fn, d.open)
	}
	// Bounded to the dispatch's own braces before anything is read out of it.
	// Both renderers are ~600 lines with several string switches in them — one
	// on text alignment, one on justify-content, both with a "center" arm — so
	// a scan that simply ran forward from here would happily collect arms out
	// of the next switch down, and would find *its* catch-all when this one had
	// been deleted. That is not a hypothetical: it is what the first version of
	// this test did, and the mutation that deleted grMobScaled's `default` arm
	// went uncaught because of it.
	body = matchingBrace(t, d, body[open+len(d.open)-1:])

	// Cutting at the catch-all before collecting arms does two jobs: it proves
	// the catch-all still exists (the contract that an absent or unrecognized
	// value has a defined rendering), and it keeps unreachable arms below it
	// from counting as coverage.
	stop := d.fallback.FindStringIndex(body)
	if stop == nil {
		t.Fatalf("%s: %s has no %s arm — every native renderer is required to define what an absent "+
			"or unrecognized value does", d.file, d.fn, d.fallbackDesc)
	}

	var labels []string
	for _, m := range d.arm.FindAllStringSubmatch(body[:stop[0]], -1) {
		labels = append(labels, d.parseLabelList(t, m[1])...)
	}
	if len(labels) == 0 {
		t.Fatalf("%s: %s's arms parsed as empty, which is not a pass — the dispatch was found but no "+
			"labels were read out of it", d.file, d.fn)
	}
	return labels
}

// parseLabelList reads the string literals out of one arm's label list,
// covering the multi-label forms both languages allow (`case "a", "b":`,
// `"a", "b" ->`).
//
// The list is validated rather than merely scraped: after the literals and
// their separators are removed, anything left means the arm says something
// this parse does not understand, and the honest response is to stop. Scraping
// alone would silently truncate such an arm to whichever labels it happened to
// recognize, and a truncated arm reads as missing coverage — a confusing
// failure — or, worse, as coverage that is not there.
func (d dispatchSyntax) parseLabelList(t *testing.T, list string) []string {
	t.Helper()

	var out []string
	rest := stringLiteral.ReplaceAllStringFunc(list, func(lit string) string {
		out = append(out, strings.Trim(lit, `"`))
		return ""
	})
	if leftover := strings.Trim(rest, " \t,"); leftover != "" {
		t.Fatalf("%s: %s has an arm this test cannot read: %q (unhandled: %q) — arms are required to "+
			"be string literals so their coverage can be checked", d.file, d.fn, list, leftover)
	}
	return out
}

var stringLiteral = regexp.MustCompile(`"[^"]*"`)

// matchingBrace returns the contents of the block that src opens with, from
// just after the opening brace to just before the brace that closes it.
//
// Brace counting has to skip comments and string literals or it counts the
// wrong braces — a `{` inside an arm's explanatory comment would end the block
// early, and both renderers do contain string literals with braces in them
// elsewhere. Swift and Kotlin spell all three constructs the same way, so one
// scanner serves both. Neither language's extras matter here: Swift's `\(…)`
// interpolation nests parentheses rather than braces, and Kotlin's `${…}` is
// balanced, so it counts a `{` and its `}` and comes out even.
func matchingBrace(t *testing.T, d dispatchSyntax, src string) string {
	t.Helper()

	depth := 0
	for i := 0; i < len(src); i++ {
		switch {
		case strings.HasPrefix(src[i:], "//"):
			nl := strings.IndexByte(src[i:], '\n')
			if nl < 0 {
				i = len(src)
				continue
			}
			i += nl
		case strings.HasPrefix(src[i:], "/*"):
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				t.Fatalf("%s: unterminated block comment inside %s", d.file, d.fn)
			}
			i += 2 + end + 1
		case src[i] == '"':
			// Escapes are honored so that a literal ending in \" does not read
			// as still open, which would swallow the rest of the file.
			j := i + 1
			for j < len(src) && src[j] != '"' {
				if src[j] == '\\' {
					j++
				}
				j++
			}
			i = j
		case src[i] == '{':
			depth++
		case src[i] == '}':
			depth--
			if depth == 0 {
				return src[1:i]
			}
		}
	}
	t.Fatalf("%s: %s's dispatch block is unterminated — the opening brace of %q has no match",
		d.file, d.fn, d.open)
	return ""
}
