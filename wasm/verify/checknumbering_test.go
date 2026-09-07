package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// browser.mjs's checks are one numbered sequence, and the number is an address.
//
// # Why this needed a mechanism
//
// The file opens with a numbered list of what it settles and main() marks each
// check with the same number, and the two had drifted into being different
// numberings of the same eleven things: the header folded the toolbar walk into
// check 1 and stopped at ten, the body had two checks numbered 6, and both
// counted differently from the tally in the first sentence. A dozen comments in
// four languages cite these numbers — browser.mjs itself, wasm/verify's gen.go
// and fixedsize_test.go, mobile/verify, ios/verify/flex.swift,
// internal/bandfixture — so "check 8" resolved to two different checks
// depending on which list the reader had in front of them.
//
// Nothing could have caught that. The numbers are prose, the checks are a
// browser pass that skips without a Chrome, and a duplicate 6 changes no
// behaviour at all. What follows is the smallest thing that makes the sequence
// a fact rather than a convention.
//
// # What is held
//
//	the header lists 1..N       contiguous, no duplicates, in order
//	the body marks 1..N         the same, in the order main() runs them
//	the two agree               entry N's text begins with marker N's text, so
//	                            renaming a check in one place and not the other
//	                            is a failure rather than two names for one thing
//	the tallies agree           the opening sentence counts the checks by kind
//	                            and a later line counts them outright; both must
//	                            come to N
//	every citation resolves     `check K` anywhere in the repository is a K the
//	                            sequence actually has
//
// It lives in Go rather than in the pass itself for the reason switchlabels
// does one directory over: a comment is not executed, so the only thing that
// can hold two of them together is something that reads the file — and doing it
// here keeps it inside `go test ./...`, where it runs for anyone with a Go
// toolchain and no Chrome.
const browserChecks = "browser.mjs"

// A header entry: `//   7. a real widget draws the palette. Check 5 paints…`,
// at column zero, with its number right-aligned in a three-space field.
var headerEntry = regexp.MustCompile(`^// {1,3}(\d+)\. (.*)$`)

// Its continuation lines, indented to sit under the text.
var headerCont = regexp.MustCompile(`^// {6}(\S.*)$`)

// A body marker: the same shape, indented inside main(), and always followed by
// a rule. The rule is what tells a marker from an ordinary numbered list in
// some other comment — there is no other line in the file that a numbered
// comment and a row of dashes both describe.
var bodyMarker = regexp.MustCompile(`^\s+// (\d+)\. (.*)$`)
var bodyCont = regexp.MustCompile(`^\s+// {4,5}(\S.*)$`)
var bodyRule = regexp.MustCompile(`^\s+// -{10,}$`)

// numbered is one entry or marker: its number and its text with the
// continuations joined on.
type numbered struct {
	n    int
	text string
	line int
}

func TestTheBrowserChecksAreOneNumberedSequence(t *testing.T) {
	lines := strings.Split(readFixtureFile(t, browserChecks), "\n")

	header := headerList(lines)
	body := bodyList(lines)

	// Both are sequences before either is compared with the other: a failure
	// here is about one list, and saying which one is most of the fix.
	if !isSequence(t, "the header's list", header) {
		return
	}
	if !isSequence(t, "main()'s markers", body) {
		return
	}
	if len(header) != len(body) {
		t.Fatalf("%s: the header lists %d checks and main() marks %d. The number is an "+
			"address — a dozen comments across four languages cite these — so a list "+
			"that is shorter than the pass means a citation resolves to the wrong "+
			"check or to nothing.", browserChecks, len(header), len(body))
	}
	if len(header) == 0 {
		t.Fatalf("%s: no checks were found at all. Either the file was restructured or "+
			"the two patterns above no longer match it, and a parse that finds nothing "+
			"must not read as a pass.", browserChecks)
	}

	// And they are the same checks. A prefix rather than equality: the marker
	// is the check's name and the header entry is the name followed by why it
	// is here, which is the division the file already had.
	for i, h := range header {
		b := body[i]
		if !strings.HasPrefix(fold(h.text), fold(b.text)) {
			t.Errorf("%s: check %d is %q in the header (line %d) and %q in main() "+
				"(line %d).\n\n"+
				"The header entry is supposed to open with the marker's own words and "+
				"then say why the check exists. Two names for one check is how the "+
				"numbering came apart the first time: a reader looking up a citation "+
				"finds a different check under the same number.",
				browserChecks, h.n, clip(h.text), h.line, b.text, b.line)
		}
	}

	// The tallies. Both are sentences somebody has to remember to update, which
	// is exactly why they are the two that had already gone stale. They are read
	// out of the header as prose rather than as source: each wraps across
	// several comment lines, and a regexp over the raw file would be matching
	// the line breaks rather than the sentence.
	checkTallies(t, headerProse(lines), len(header))

	// Every citation, everywhere. The number means nothing on its own; what
	// makes it an address is that it resolves.
	for _, file := range []string{
		browserChecks,
		"gen.go",
		"fixedsize_test.go",
		"../../mobile/verify/fixedsize_test.go",
		"../../ios/verify/flex.swift",
		"../../internal/bandfixture/bandfixture.go",
	} {
		for _, cite := range citations(readFixtureFile(t, file)) {
			if cite < 1 || cite > len(header) {
				t.Errorf("%s cites check %d, and browser.mjs has %d. Either the citation "+
					"was not moved when the sequence was renumbered, or it names a check "+
					"that no longer exists.", file, cite, len(header))
			}
		}
	}
}

// headerProse is the opening comment with its markers stripped and its lines
// joined, so a sentence that wraps is one string.
//
// Bounded at the first import for the reason headerList is: everything above it
// is that one comment.
func headerProse(lines []string) string {
	var b strings.Builder
	for _, line := range lines {
		if strings.HasPrefix(line, "import ") {
			break
		}
		if !strings.HasPrefix(line, "//") {
			continue
		}
		b.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "//"), " "))
		b.WriteByte(' ')
	}
	return fold(b.String())
}

// headerList reads the numbered list in the opening comment.
//
// Bounded at the first import, because everything above it is that one comment
// and nothing below it is: a column-zero `//  N.` further down the file would
// otherwise be read as a twelfth entry.
func headerList(lines []string) []numbered {
	var out []numbered
	for i, line := range lines {
		if strings.HasPrefix(line, "import ") {
			break
		}
		m := headerEntry.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		text := m[2]
		for j := i + 1; j < len(lines); j++ {
			c := headerCont.FindStringSubmatch(lines[j])
			if c == nil {
				break
			}
			text += " " + c[1]
		}
		out = append(out, numbered{n: n, text: text, line: i + 1})
	}
	return out
}

// bodyList reads main()'s markers, in source order.
//
// A marker is a numbered comment followed by a rule, once its continuations
// have been taken. The rule is the whole discriminator and it is worth being
// explicit about why that is enough: this file has other numbered prose, and
// none of it is underlined.
func bodyList(lines []string) []numbered {
	var out []numbered
	for i, line := range lines {
		m := bodyMarker.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		text := m[2]
		j := i + 1
		for ; j < len(lines); j++ {
			c := bodyCont.FindStringSubmatch(lines[j])
			if c == nil {
				break
			}
			text += " " + c[1]
		}
		if j >= len(lines) || !bodyRule.MatchString(lines[j]) {
			continue // numbered prose, not a check marker
		}
		out = append(out, numbered{n: n, text: text, line: i + 1})
	}
	return out
}

// isSequence reports whether a list is 1..len in order, naming the first place
// it is not.
func isSequence(t *testing.T, what string, list []numbered) bool {
	t.Helper()
	for i, e := range list {
		if e.n != i+1 {
			t.Errorf("%s: the %s entry is numbered %d (line %d, %q). The sequence has to "+
				"be 1..%d in order, or a citation is ambiguous — which is what two "+
				"checks numbered 6 did to it.",
				browserChecks, ordinal(i+1), e.n, e.line, e.text, len(list))
			return false
		}
	}
	return true
}

// The two sentences that count the checks in prose.
var tallyByKind = regexp.MustCompile(
	`(\w+) about the keyboard, (\w+) about paint, (\w+) about layout, and (\w+) about`)
var tallyOutright = regexp.MustCompile(`(\w+) claims sit exactly in that blind spot`)

// checkTallies holds both to the sequence's length.
//
// Spelled-out numbers, which is how the file writes them, so the words are
// mapped rather than parsed. Only the counts the file actually uses are
// listed: a tally that grew past them fails at the lookup, which is the right
// answer — somebody has to write the new word anyway.
func checkTallies(t *testing.T, src string, want int) {
	t.Helper()
	words := map[string]int{
		"one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6,
		"seven": 7, "eight": 8, "nine": 9, "ten": 10, "eleven": 11, "twelve": 12,
	}

	if m := tallyByKind.FindStringSubmatch(src); m == nil {
		t.Errorf("%s: the opening sentence no longer counts the checks by kind where "+
			"this looks for it (%s). It is the first thing a reader is told about the "+
			"file's size, and it had already gone stale once.", browserChecks, tallyByKind)
	} else {
		sum := 0
		for _, w := range m[1:] {
			n, ok := words[strings.ToLower(w)]
			if !ok {
				t.Errorf("%s: the opening sentence counts %q of something, which is not a "+
					"number word this knows. Add it here or write the count in figures.",
					browserChecks, w)
				return
			}
			sum += n
		}
		if sum != want {
			t.Errorf("%s: the opening sentence counts %d checks by kind (%s) and there "+
				"are %d. Every check belongs to exactly one of those four kinds, so the "+
				"tally is the sequence's length written a second way.",
				browserChecks, sum, strings.Join(m[1:], "+"), want)
		}
	}

	m := tallyOutright.FindStringSubmatch(src)
	if m == nil {
		t.Errorf("%s: nothing says how many claims sit in dom.mjs's blind spot where "+
			"this looks for it (%s)", browserChecks, tallyOutright)
		return
	}
	if got, ok := words[strings.ToLower(m[1])]; !ok || got != want {
		t.Errorf("%s: %q claims are said to sit in dom.mjs's blind spot, and the file "+
			"checks %d of them", browserChecks, m[1], want)
	}
}

// citations finds every `check N` / `Check N` in a file.
var citation = regexp.MustCompile(`[Cc]heck (\d+)`)

func citations(src string) []int {
	var out []int
	for _, m := range citation.FindAllStringSubmatch(src, -1) {
		n, err := strconv.Atoi(m[1])
		if err == nil {
			out = append(out, n)
		}
	}
	return out
}

// clip shortens a header entry to the part that is being compared. The entries
// run to a paragraph apiece, and a failure that prints one whole is a failure
// nobody reads to the end of.
func clip(s string) string {
	const width = 80
	if len(s) <= width {
		return s
	}
	return s[:width] + "…"
}

// fold normalises a title for comparison: one space between words, no case.
// The two copies wrap at different widths, so the line breaks are not part of
// what is being compared.
func fold(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

// ordinal names a position in a failure, because "the 7th entry is numbered 6"
// is the sentence that says what happened and "list[6].n == 6" is not.
func ordinal(n int) string {
	suffix := "th"
	switch {
	case n%100 >= 11 && n%100 <= 13:
	case n%10 == 1:
		suffix = "st"
	case n%10 == 2:
		suffix = "nd"
	case n%10 == 3:
		suffix = "rd"
	}
	return fmt.Sprintf("%d%s", n, suffix)
}
