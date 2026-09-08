package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
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
//	                            sequence actually has. "Anywhere" is literal: the
//	                            tree is walked, and the few files that number
//	                            things of their own are named with the reason
//	                            (see citationExempt)
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

	// Every citation, everywhere in the repository. The number means nothing
	// on its own; what makes it an address is that it resolves.
	checkCitationsResolve(t, len(header))
}

// The repository, walked, and every `check N` in it held to the sequence.
//
// # What this replaced
//
// A hand-written list of nine files. It was a stated limit rather than a claim
// of totality, and the limit was real in both the ways a list is: two files that
// cited the sequence were missing from it and were added when somebody noticed,
// and docs/platforms/native.md — the census the whole numbering exists to serve
// — cited check 12 twice and was never in it at all.
//
// A walk has no such gap. A file that starts citing a check is covered the day
// it is written rather than the day somebody remembers this list.
//
// # The inversion, and why it is the whole of the improvement
//
// The list used to say which files OPT IN. It says now which files opt OUT, and
// each entry carries the reason it is not about browser.mjs. That is a much
// smaller thing to keep true: an omission from an opt-in list is a file nobody
// checks, and an omission from an opt-out list is a failure that names the file.
//
// # And the exemptions are held too
//
// Every entry below must still have a citation in it. An exemption that stopped
// being needed is the same kind of dead weight the old list's missing entries
// were, one direction over — it reads as "this file is known to cite checks of
// its own" long after it stopped doing so, and the next reader trusts it.
func checkCitationsResolve(t *testing.T, checks int) {
	t.Helper()

	root := filepath.Join("..", "..")

	// The enumeration git succeeded at and produced nothing from, which is not
	// the same fault as having no git and must not be told as if it were.
	//
	// A `git ls-files` that exits 0 with an empty listing describes a
	// repository containing no files, and this one contains this test. So it is
	// a fault in the question rather than in the answer — the wrong directory,
	// a repository that is not this one — and falling back to the walk would
	// hand the reader a passing check and the other story.
	//
	// Asked separately from the enumeration below, and so git runs twice. That
	// is deliberate rather than an oversight: citingFiles reports which
	// enumeration answered and not why, because for every other purpose the two
	// are one answer, and folding this state into its return would make the
	// common path carry a case only this line cares about.
	if listed, err := repositoryFiles(root); err == nil && len(listed) == 0 {
		t.Fatalf("`git ls-files` succeeded in %s and listed no files at all.\n\n"+
			"That is not a machine without git — this check falls back to a filesystem "+
			"walk for that, and says so. It is git answering about a repository with "+
			"nothing in it, which this one is not: the enumeration is being run "+
			"somewhere other than the working tree, and every citation in the "+
			"repository is invisible to it.", root)
	}

	files, from, err := citingFiles(root)
	if err != nil {
		t.Fatalf("enumerating the repository for citations (%s): %v", from, err)
	}
	// An enumeration that found nothing must not read as a pass. browser.mjs
	// alone carries a dozen of them, so anything near zero means it is not
	// reaching the tree.
	if len(files) < 5 {
		t.Fatalf("%s found citations in %d files. browser.mjs, its own gen.go and "+
			"the census in docs/ all carry several apiece, so a number this small means "+
			"the enumeration is looking somewhere else or skipping everything.",
			from, len(files))
	}
	// Which set was actually searched. The two cover different things — git
	// leaves out anything it is ignoring, the walk leaves out five prefixes —
	// so a reader looking at a failure below needs to know which of them
	// produced the list, and a reader looking at a PASS needs to know the check
	// was not quietly running in its weaker form.
	t.Logf("citations enumerated by %s: %d files", from, len(files))

	seenExempt := map[string]bool{}
	for _, f := range files {
		if _, exempt := citationExempt[f.path]; exempt {
			seenExempt[f.path] = true
			continue
		}
		for _, cite := range f.cites {
			if cite < 1 || cite > checks {
				t.Errorf("%s cites check %d, and browser.mjs has %d. Either the citation "+
					"was not moved when the sequence was renumbered, or it names a check "+
					"that no longer exists.", f.path, cite, checks)
			}
		}
	}

	for path, why := range citationExempt {
		if !seenExempt[path] {
			t.Errorf("%s is exempted from the citation check (%s) and no longer cites a "+
				"check at all. An exemption outlives its reason silently: it goes on "+
				"telling the next reader that this file numbers things of its own, and "+
				"the day it starts citing browser.mjs instead nothing looks.", path, why)
		}
	}
}

// The files whose `check N` is not an address into browser.mjs, and why.
//
// Keyed by repository-relative path with forward slashes. Every entry is
// asserted to still carry a citation — see checkCitationsResolve.
var citationExempt = map[string]string{
	"core/debug.go": "its \"Check 1\", \"Check 2\" and \"Check 3\" are that file's own " +
		"numbered sections — a development-mode audit that has nothing to do with a " +
		"browser pass, and predates it",
	"wasm/verify/checknumbering_test.go": "this file's own prose quotes citations as " +
		"examples, the stale `check 10` that prompted the walk among them. A test " +
		"that read its own explanation as an address would fail on the sentence " +
		"describing the failure it exists to catch",
}

// The directories a citation walk does not enter, and why.
//
// Kept as path prefixes rather than as base names so that a directory named
// `build` somewhere else in the tree is not skipped by accident.
//
// Four of the five are reached only by the fallback enumeration. git excludes
// the build directories itself, because the two platform .gitignore files
// already name them — so on any machine with git these entries are a second
// statement of something already stated, kept for the walk that runs when there
// is no git to ask. `docs/site` is load-bearing in both paths, and `ai_docs` is
// a decision no mechanism could make.
var citationSkipDirs = []string{
	".git",
	// Saved sessions and plans are a RECORD of what was true when they were
	// written. A renumbering does not make last week's session doc wrong, and
	// rewriting one to keep a test green would be falsifying the record.
	"ai_docs",
	// Build output: generated sources, jars, and a wasm binary, none of it
	// written by anybody here.
	"android/build",
	"android/.gradle",
	"android/app/build",
	"ios/build",
	"docs/site",
}

// citing is one file and the check numbers it names.
type citing struct {
	path  string
	cites []int
}

// repositoryFiles asks git which files are this working tree's.
//
// # Why git rather than the disk
//
// The enumeration below used to be a filesystem walk minus five directory
// prefixes, and that list was the whole of what stood between the citation
// check and a `node_modules` somebody adds: a vendored dependency carrying the
// words "check 3" in a changelog is a failure naming a file nobody here wrote,
// and the fix would have been a sixth prefix, and then a seventh.
//
// `git ls-files --cached --others --exclude-standard` is every tracked file
// plus every untracked one git would offer to add. That is exactly the set a
// person here writes, and it keeps the property the walk was adopted for —
// `--others` covers a file written five minutes ago, so a new document is
// checked before it is committed.
//
// What remains outside it is a dependency vendored by COMMITTING it, which is
// this repository's file by every mechanical test there is. That failure is
// loud and names the file, and saying so is better than an enumeration
// pretending to cover it.
//
// # The three answers, which are three different things
//
// A non-nil error is "there is no git here, or it would not answer" — the
// caller falls back to the walk and says so. A nil error with an empty listing
// is a different fault and not a smaller one: git SUCCEEDED and reported a
// repository with no files in it, which is not a state this tree can be in, and
// falling back would tell the reader the first story about the second
// situation. So the two are returned distinguishably and the caller separates
// them.
func repositoryFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "-C", root, "ls-files", "--cached", "--others",
		"--exclude-standard", "-z")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	// NUL-separated, which is what -z buys: a path with a newline or a quote in
	// it is a path git would otherwise escape and this would have to unescape.
	var files []string
	for _, p := range strings.Split(string(out), "\x00") {
		if p != "" {
			files = append(files, p)
		}
	}
	sort.Strings(files)
	return files, nil
}

// walkedFiles is the enumeration for a machine with no git: every regular file
// under root, minus the skip prefixes.
//
// Kept rather than deleted, and it is not dead weight — a source tarball, a
// container image built by copying the tree in, and `go test` run from an
// export all reach it. What it cannot do is the thing git does for free, which
// is why the caller announces which enumeration it used.
func walkedFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			for _, skip := range citationSkipDirs {
				if rel == skip {
					return fs.SkipDir
				}
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	return out, err
}

// citingFiles returns every text file of the repository carrying a citation,
// with the repository-relative slash path a failure names.
//
// `from` says which enumeration answered, so the caller can tell the reader
// what the check was able to see. It is not decoration: the two enumerations
// cover different sets, and a failure that lists a vendored file means
// something different depending on which one produced it.
//
// Binary files are skipped by looking for a NUL byte rather than by extension:
// an extension list is a second thing to keep current, and the one property
// that actually matters here — "a regexp over this is meaningless" — is exactly
// what a NUL is evidence of.
func citingFiles(root string) (found []citing, from string, err error) {
	files, gitErr := repositoryFiles(root)
	from = "git ls-files"
	if gitErr != nil {
		from = fmt.Sprintf("a filesystem walk (git could not answer: %v)", gitErr)
		if files, err = walkedFiles(root); err != nil {
			return nil, from, err
		}
	}

	for _, rel := range files {
		// The skip prefixes apply to both enumerations. Under git they are
		// mostly redundant — see citationSkipDirs — and `ai_docs` is not: it is
		// tracked, and being tracked is precisely why it has to be named here.
		skipped := false
		for _, skip := range citationSkipDirs {
			if rel == skip || strings.HasPrefix(rel, skip+"/") {
				skipped = true
				break
			}
		}
		if skipped {
			continue
		}
		p := filepath.Join(root, filepath.FromSlash(rel))
		info, statErr := os.Stat(p)
		if statErr != nil {
			// git lists the index, and an index entry can name a file that is
			// not on disk right now — a deleted-but-unstaged path, a sparse
			// checkout. Not this check's business either way.
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		// A bound rather than a measurement: nothing in this repository that a
		// person writes citations into is anywhere near it, and it keeps a
		// stray large artifact out of memory.
		if info.Size() > 4<<20 {
			continue
		}
		raw, readErr := os.ReadFile(p)
		if readErr != nil {
			return nil, from, readErr
		}
		if bytes.IndexByte(raw, 0) >= 0 {
			continue
		}
		if cites := citations(string(raw)); len(cites) > 0 {
			found = append(found, citing{path: rel, cites: cites})
		}
	}
	return found, from, nil
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
