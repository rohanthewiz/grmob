package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
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

	files, considered, from, err := citingFiles(root)
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

	// And that the two things citingFiles returns are two questions.
	//
	// citationExemptInputs takes `considered` and `files` and reads one lookup
	// out of each, and a fixture holds it to the four ways those two can
	// answer. What produces them is still ONE loop in citingFiles that appends
	// to `found` only inside the branch that has just written `considered` —
	// correct, and the only thing making them different questions rather than
	// two names for one set. A loop that stopped recording the paths it opened
	// and nothing else, or a `considered` narrowed to the citing paths as a
	// tidy-up, would leave every row of that fixture passing while the walk
	// below could no longer reach the case the fixture exists to separate: an
	// exemption whose file is still there and has stopped citing anything.
	//
	// So the property is asked here, of the real tree. Both halves: every
	// citing path was opened (or `cited` is true where `reached` is false, a
	// pair citationExemptVerdict has no arm for), and something was opened
	// that cites nothing (or the two maps are the same set and the separating
	// case is unreachable from this repository).
	for _, f := range files {
		if _, opened := considered[f.path]; !opened {
			t.Errorf("%s carries citations and is not among the %d paths the "+
				"enumeration recorded opening.\n\n"+
				"citationExemptInputs reads `reached` out of that table and `cited` out "+
				"of this list, and the ORDER of citationExemptVerdict's two arms rests "+
				"on a citing path always having been reached — a file that cites "+
				"without having been opened is a pair that function has no arm for and "+
				"would diagnose as a rename.", f.path, len(considered))
			break
		}
	}
	// The separating case itself, rather than the two sizes it shows up in.
	//
	// This was `len(considered) > len(files)`, which says a silent file exists
	// SOMEWHERE and says it by arithmetic: it is a statement about the
	// separating case only while every citing path is also a considered one,
	// which is the loop above — and that loop reports and carries on, so a run
	// where it fired could go on to read the inequality as evidence about
	// nesting that had just been shown not to hold.
	//
	// So the set is taken: the paths the enumeration opened and found no
	// citation in. Same property, measured on the population it is about, and
	// a failure can name what it looked at instead of leaving a reader to
	// subtract two numbers.
	citing := map[string]bool{}
	for _, f := range files {
		citing[f.path] = true
	}
	silent := []string{}
	for path := range considered {
		if !citing[path] {
			silent = append(silent, path)
		}
	}
	sort.Strings(silent)
	if len(silent) == 0 {
		t.Errorf("the enumeration opened %d files and every one of them carries a "+
			"citation, so nothing it looked inside is silent.\n\n"+
			"citingFiles returns the two as separate answers because they are separate "+
			"questions — citationExempt has to know a path still EXISTS and "+
			"citationSenses has to know which senses were produced — and the case that "+
			"tells them apart is a path the walk opened and found no citation in. With "+
			"the two sets equal that case is unreachable here: TestCitationExempt"+
			"InputsComeFromTheTwoEnumerations would go on passing over a fixture, and "+
			"an exemption that had simply gone quiet would report as a rename against "+
			"a file still sitting on disk.", len(considered))
	}

	// And whether any of that set is a file an exemption could be about.
	//
	// # A separating case drawn from the wrong population
	//
	// The example named above was silent[0], the alphabetically first path the
	// walk opened and found no citation in — which in this repository is
	// .github/workflows/ci.yml, a file nothing would ever cite. It satisfies
	// "reached and not citing" and it is evidence for nothing:
	// citationExemptVerdict's silent arm is about a file somebody wrote an
	// EXEMPTION for, and an exemption is a claim that a file's own `check N`
	// is not an address into browser.mjs. A silent set that was nothing but
	// workflow files and licences would read exactly like this one and would
	// support none of what the sentence above claims — the walk would have
	// proved it can open a YAML file.
	//
	// The population an exemption is drawn from is the files this check is
	// actually about, and that population is measured rather than listed: the
	// extensions the citing files have, plus the extensions citationExempt's
	// own rows have. Both are in hand here, and the second matters on its own —
	// an exemption's file has stopped citing by hypothesis, so its kind can be
	// absent from `files` precisely in the case this is about.
	//
	// Being wrong towards a wider population would put a .md back in the
	// example and cost the reader a weaker illustration; being wrong towards a
	// narrower one would report a separating case as missing while one was
	// sitting there. So the kinds are a union and not an intersection.
	kinds := map[string]bool{}
	for _, f := range files {
		kinds[filepath.Ext(f.path)] = true
	}
	// The tighter of the two, kept apart because it is what the EXAMPLE is
	// drawn from. An exemption is a claim about a file somebody wrote, and the
	// two rows on file are both Go source; a silent README satisfies the
	// property and a silent .go file is what the sentence is describing. The
	// assertion stays over the union, so a repository that stopped exempting
	// Go fails nothing here — it is the illustration that narrows, not the
	// claim.
	//
	// # Why a kind stands in for a file, and what it cannot see
	//
	// An exemption is about CONTENT: this file numbers things of its own, so a
	// `check N` in it is not an address into browser.mjs. Nothing here can ask
	// that of a path, and the closest thing that can be asked cheaply is the
	// extension — which stands in for it on the evidence of the rows
	// themselves. A kind with an exemption written for it is a kind that has
	// been shown to carry the numbered prose the arm is about; a kind with
	// none has not, and a silent file of that kind is silent for reasons the
	// arm has nothing to say about.
	//
	// The proxy is coarse in exactly one direction and it is worth naming:
	// core/debug.go is hand-written and a generated zz_gen.go would share its
	// kind, so the population takes in files no human numbered anything in.
	// That costs the ILLUSTRATION and not the claim — the assertion below is
	// over the union, which is wider still — and the report says which rows
	// lend the example its kind, so a population that widened is a population
	// the next reader can see widening rather than a number that moved.
	//
	// And only a row the enumeration actually reached lends one. A path this
	// walk never opens is a rename or a delete that citationExemptVerdict is
	// about to report on its own line, and until it does its extension would
	// go on tightening this population — the example would be drawn from a
	// kind whose whole evidence is a row about a file that is not there.
	//
	// # Every row of the kind, not the first of them
	//
	// This used to keep the alphabetically first row per extension, chosen so
	// that two exemptions sharing a kind named the same lender on every run.
	// Stability is right and picking one is what was wrong with it: both rows
	// here are Go source and their arguments are nothing alike — one is a
	// development-mode audit with numbered sections of its own, the other is a
	// test file that quotes citations as examples. A reader sent to whichever
	// sorts first is being shown an argument that may have nothing to do with
	// the file in the example, and the table's `why` — the half that tells them
	// apart — was in hand at this point and read by nothing.
	//
	// So the kind carries all of its reached rows, sorted, and the recital
	// prints each with the reason written beside it. Sorted for the same
	// stability the single lender was chosen for.
	//
	// # And an argument printed whole has no bound
	//
	// Every `why` in full is the right direction and it grows with the number
	// of rows sharing a kind: two of them already put six hundred characters
	// into a PASSING run's log, and a third makes it a paragraph. A line nobody
	// reads is a line that says nothing, which is where the count-shaped
	// version of this started.
	//
	// The reader's question is which of these arguments fits the file in the
	// example, and this file cannot answer it — an exemption is a claim about
	// CONTENT and all that is in hand here is a path. What it can do is put the
	// likeliest first and bound the rest: rows in the example's own directory
	// lead, alphabetical within that, and the arguments are clipped at
	// citationWhyBudget with the count of what was left out and the table named
	// to read it in. Proximity is a guess and it is stated as one — the line
	// says the order it is in — where picking one row and printing it alone
	// stated nothing at all.
	exemptKinds := map[string][]string{}
	for path := range citationExempt {
		kinds[filepath.Ext(path)] = true
		if _, reached := considered[path]; !reached {
			continue
		}
		ext := filepath.Ext(path)
		exemptKinds[ext] = append(exemptKinds[ext], path)
	}
	for ext := range exemptKinds {
		sort.Strings(exemptKinds[ext])
	}
	couldBeExempt, likeAnExemption := []string{}, []string{}
	for _, path := range silent {
		if kinds[filepath.Ext(path)] {
			couldBeExempt = append(couldBeExempt, path)
		}
		if len(exemptKinds[filepath.Ext(path)]) > 0 {
			likeAnExemption = append(likeAnExemption, path)
		}
	}
	// The tightest example there is, and the wider one when the tight
	// population is empty. Both are silent paths of a kind this check is
	// about; the first is one of a kind an exemption has actually been written
	// for.
	example := couldBeExempt
	if len(likeAnExemption) > 0 {
		example = likeAnExemption
	}
	switch {
	case len(silent) == 0:
		// Already reported above, and reporting it twice would send a reader
		// looking for two faults.
	case len(couldBeExempt) == 0:
		t.Errorf("%d of the %d files the enumeration opened cite nothing, and not one "+
			"of them is a kind anything in this repository cites or is exempted for "+
			"(the kinds are %v; the silent set starts %s).\n\n"+
			"The separating case citationExemptInputs needs is a path the walk opened "+
			"and found no citation in, and it needs it as evidence about EXEMPTIONS: "+
			"citationExemptVerdict's silent arm fires for a file somebody claimed "+
			"numbers things of its own and that has stopped citing. A silent set made "+
			"entirely of files nothing would ever cite satisfies the arithmetic and "+
			"supports none of that — the walk has shown it can open a file, which was "+
			"never in doubt.", len(silent), len(considered), kindList(kinds), silent[0])
	default:
		// Named, because a passing run's evidence for "the two enumerations are
		// two questions" is this list existing and nothing else says so — and
		// named from the population the claim is about rather than from the top
		// of an alphabetical list.
		//
		// With the rows that lend the example its kind and the reason each of
		// them is exempted, which is the half a count cannot carry: the tight
		// population is as good as the arguments its extension is in it for,
		// and an exemption written for some other kind widens it without moving
		// any number here. Printing the reasons puts the thing a reader would
		// have to open the table for into the line a passing run prints.
		lenders := citationLendersNear(exemptKinds[filepath.Ext(example[0])],
			example[0])
		how := "a kind this check is about, with no exemption written for it"
		if len(lenders) > 0 {
			shown := lenders
			if len(shown) > citationLendersShown {
				shown = shown[:citationLendersShown]
			}
			said := make([]string, 0, len(shown))
			for _, lender := range shown {
				said = append(said, fmt.Sprintf("%s, because %s", lender,
					citationClipWhy(citationExempt[lender])))
			}
			rows := fmt.Sprintf("%d rows are", len(lenders))
			if len(lenders) == 1 {
				rows = "one row is"
			}
			rest := ""
			if len(lenders) > len(shown) {
				rest = fmt.Sprintf("; and %d more of this kind, in citationExempt",
					len(lenders)-len(shown))
			}
			// With the reading that says whether "nearest first" chose
			// anything. See citationLenderSpread: on a kind whose rows are all
			// in one directory the phrase is true and empty, and a reader who
			// takes it for a match has been told something this file cannot
			// know.
			n, near, dirs, spread := citationLenderSpread(
				lenders, example[0], exemptKinds)
			how = fmt.Sprintf("the kind %s exempted under, nearest this file first "+
				"(%s): %s%s", rows,
				citationLenderWhy(n, near, dirs, spread, len(exemptKinds)),
				strings.Join(said, "; and "), rest)
		}
		t.Logf("%d of the %d files opened cite nothing, %d of them a kind this check "+
			"is about and %d a kind citationExempt itself names (e.g. %s), which "+
			"is the pair citationExemptVerdict's silent arm is about.\n\nThat is %s",
			len(silent), len(considered), len(couldBeExempt), len(likeAnExemption),
			example[0], how)
	}

	// Every sense the enumeration produced has to be one somebody classified.
	//
	// The senses used to reach a failure message and no assertion: repoFile
	// carried them so a reader could be told who has to act, and nothing in the
	// check turned on which one it was. That is not the same as their being
	// equal — it is the question never having been asked, and a third sense
	// arriving (the walk's, on a machine with no git) would have slipped
	// through the same way. See citationSenses.
	present := map[string]bool{}
	for path, sense := range considered {
		if _, classified := citationSenses[sense]; !classified {
			t.Errorf("%s came back as %q and citationSenses has no row for it.\n\n"+
				"A sense is how the enumeration knows this file is the repository's, "+
				"and citationSenses is where what that costs the reader is decided. An "+
				"unclassified one is a file being held to a rule nobody chose for it.",
				path, sense)
			break
		}
		present[sense] = true
	}
	// And whether what the table decides about those senses leaves this run able
	// to fail at all. See citationSenseVerdict.
	if v := citationSenseVerdict(present); v != "" {
		t.Error(v)
	}

	for _, f := range files {
		if _, exempt := citationExempt[f.path]; exempt {
			continue
		}
		act, classified := citationSenses[f.sense]
		if !classified {
			continue // already reported above
		}
		for _, cite := range f.cites {
			// The decision is citationVerdict's, so that both of its arms are
			// held by a fixture rather than one of them being reachable only by
			// editing citationSenses. Choosing which reporter carries the answer
			// is citationReport's, for the same reason one level up: the three
			// arms of that choice are not reachable from any repository this
			// walk can be pointed at either.
			say, fails := citationVerdict(f.path, f.sense, cite, checks, act)
			citationReport(t, say, fails)
		}
	}

	// The exemptions, in the two ways one can outlive its reason.
	//
	// The decision is citationExemptVerdict's for the reason citationVerdict and
	// citationReport are functions: it has three arms and this walk reaches one
	// of them. Every exemption in the table is reached and still cites a check,
	// so both failing arms are unreachable from any repository this test can be
	// pointed at, and the sentence each one carries was prose nobody could ask a
	// question of.
	//
	// What FEEDS that decision is citationExemptInputs', for the same reason
	// one level further in: the two booleans come from two different
	// enumerations, and the wrong one on either side flips the diagnosis while
	// leaving the verdict function perfectly correct.
	for path, why := range citationExempt {
		reached, cited := citationExemptInputs(path, considered, files)
		if v := citationExemptVerdict(path, why, from, reached, cited); v != "" {
			t.Error(v)
		}
	}
}

// The file kinds a failure names, sorted so two runs read the same.
//
// A map's iteration order is deliberate noise in Go, and a message that listed
// extensions straight out of one would differ between two runs over an
// unchanged tree — which is the shape of thing that makes a reader wonder what
// moved.
func kindList(kinds map[string]bool) []string {
	out := make([]string, 0, len(kinds))
	for ext := range kinds {
		out = append(out, ext)
	}
	sort.Strings(out)
	return out
}

// What the recital of a kind's exemptions is allowed to cost a passing run.
//
// The arguments are the half that tells two rows of one kind apart — see where
// exemptKinds is built — and they are prose somebody wrote to be read once, in
// the table, by a reader who had gone looking. Printed whole on every green run
// they are unbounded: two of them already come to about six hundred characters
// and nothing stops a third.
//
// Two rows with their arguments clipped is the compromise. It is enough to see
// that the arguments DIFFER, which is what the single-lender version could not
// show at all, and the count and the table name are what a reader follows when
// neither of the two shown is the one they need.
const (
	citationLendersShown = 2
	citationWhyBudget    = 120
)

// citationLendersNear is a kind's exempt rows with the ones nearest the example
// first.
//
// An exemption is a claim about a file's CONTENT and the example is a silent
// file: which of these arguments applies to it is not a question this file can
// answer, and the previous version's answer — whichever sorted first — was a
// worse guess than no guess, because nothing in the line said it was one.
//
// Directory is the one thing in hand that correlates with content at all: a
// silent .go file in wasm/verify sits beside a .go exemption written about
// wasm/verify, and the two are far likelier to be the same kind of file than
// either is to something in core. Alphabetical within each group, so two runs
// over an unchanged tree print the same line — the stability the single lender
// was picked for, kept.
//
// The input is already sorted, so the two groups come out sorted; the copy is
// because the caller's slice is the map's own.
func citationLendersNear(lenders []string, example string) []string {
	near, far := []string{}, []string{}
	dir := filepath.Dir(example)
	for _, lender := range lenders {
		if filepath.Dir(lender) == dir {
			near = append(near, lender)
			continue
		}
		far = append(far, lender)
	}
	return append(near, far...)
}

// citationLenderSpread is what the proximity sort actually had to decide, for
// one kind and across all of them.
//
// # A sort with no population under it
//
// citationLendersNear puts same-directory rows first, and on this repository
// the example is a core/ file and both core/ exemptions lead — which looks like
// the heuristic working and is equally consistent with there being two rows and
// one directory between them. Nothing counted, so "nearest this file first" was
// a claim about a sort rather than about a match, and a reader had no way to
// tell an ordering that fired from one that could not have.
//
// Three numbers say which:
//
// \trows      the kind's exemptions. Under two the sort has nothing to order.
// \tnear      how many of them are in the example's own directory. Zero means
// \t          every row is "far" and the order is alphabetical under another
// \t          name; all of them means the same from the other side.
// \tdirs      how many directories the kind's rows are spread over. One is the
// \t          case where proximity cannot separate anything at all.
//
// # And the same question one level up
//
// `kinds` is how many kinds in the whole table have rows in more than one
// directory, which is the population the heuristic can EVER be about. A
// repository where that is zero is one where this sort is dead code wearing a
// sentence, and the line should say so rather than print "nearest this file
// first" over a list that was going to come out in that order regardless.
//
// Taken over the whole exempt table rather than over the kinds that reached an
// example, because the question is about the sort and not about this run's
// silent set: a kind with lenders in three directories is evidence the ordering
// has work to do even on a run where no file of that kind went silent.
func citationLenderSpread(lenders []string, example string,
	byKind map[string][]string) (rows, near, dirs, kinds int) {
	rows = len(lenders)
	here := filepath.Dir(example)
	seen := map[string]bool{}
	for _, lender := range lenders {
		dir := filepath.Dir(lender)
		if dir == here {
			near++
		}
		if !seen[dir] {
			seen[dir] = true
			dirs++
		}
	}
	for _, of := range byKind {
		spread := map[string]bool{}
		for _, lender := range of {
			spread[filepath.Dir(lender)] = true
		}
		if len(spread) > 1 {
			kinds++
		}
	}
	return rows, near, dirs, kinds
}

// citationLenderWhy is that spread as the clause that qualifies the order.
//
// Said in the direction the numbers are actually in, because the three states
// send a reader to different conclusions: an order the sort decided, an order
// it could not have decided, and an order there was nothing to decide about.
// One sentence covering all three is the count-shaped answer this whole line
// was rebuilt out of.
func citationLenderWhy(rows, near, dirs, kinds, allKinds int) string {
	over := fmt.Sprintf("%d of the %d kind%s citationExempt names %s rows in more "+
		"than one directory", kinds, allKinds,
		map[bool]string{true: "", false: "s"}[allKinds == 1],
		map[bool]string{true: "has", false: "have"}[kinds == 1])
	switch {
	case rows < 2:
		return fmt.Sprintf("one row, so there is no order here to have chosen — %s",
			over)
	case dirs == 1:
		return fmt.Sprintf("all %d of them in one directory, so \"nearest first\" is "+
			"alphabetical order under another name here and decided nothing — %s",
			rows, over)
	case near == 0:
		return fmt.Sprintf("none of the %d in this file's own directory, across %d of "+
			"them, so every row is equally far and the order below is alphabetical — "+
			"%s", rows, dirs, over)
	default:
		return fmt.Sprintf("%d of the %d in this file's own directory and the rest "+
			"across %d more, so the order below is one the sort chose — %s",
			near, rows, dirs-1, over)
	}
}

// The proximity sort, asked of orderings this repository does not have.
//
// # A guess with one example under it
//
// citationLendersNear puts same-directory rows first and citationLenderSpread
// now says whether that decided anything. On this repository it decides
// nothing: the example is a file in neither lender's directory, so both rows
// are "far" and the order is alphabetical. That is a real reading and it is
// also the whole of what any run here exercises — the near group is never
// non-empty, the sort's own arm has never run, and the census reports honestly
// on a function nothing has asked a question of.
//
// So the orderings are built. Four shapes, each of which sends a reader
// somewhere different, and each with what the spread has to say about it:
//
// \tone row              nothing to order
// \tone directory        every row near or every row far; either way the
// \t                     output is the input and "nearest first" is a name for
// \t                     alphabetical
// \tnone near            this repository's own shape, where the sort is a
// \t                     no-op it cannot report as one without counting
// \tsome near            the only shape where the order below is one the sort
// \t                     chose, which is the arm no run had reached
//
// # And the two properties underneath all of them
//
// The output has to be a permutation of the input, because a row silently
// dropped is an exemption a reader is never shown and a row duplicated is one
// they are shown twice. And it has to be stable within each group, because the
// stability is what the single-lender version was picked for and is what keeps
// two runs over an unchanged tree printing one line.
func TestTheLenderOrderIsAskedOfOrderingsThisRepositoryDoesNotHave(t *testing.T) {
	for _, c := range []struct {
		what    string
		lenders []string
		example string
		// The order citationLendersNear has to produce.
		want []string
		// And what citationLenderSpread has to say about that ordering, over a
		// table of one kind — the numbers, and the clause a reader gets.
		rows, near, dirs int
		says             string
	}{
		{
			what:    "one row, so there is no order to have chosen",
			lenders: []string{"core/debug.go"},
			example: "wasm/verify/x.go",
			want:    []string{"core/debug.go"},
			rows:    1, near: 0, dirs: 1,
			says: "one row, so there is no order here to have chosen",
		},
		{
			// Every row in the example's own directory. "Nearest first" is
			// true of all of them, which is the same as being true of none.
			what:    "one directory, and it is this file's",
			lenders: []string{"core/a.go", "core/b.go"},
			example: "core/x.go",
			want:    []string{"core/a.go", "core/b.go"},
			rows:    2, near: 2, dirs: 1,
			says: "all 2 of them in one directory",
		},
		{
			// This repository's shape today, and the reading that says the
			// sentence above the list is describing a sort that did nothing.
			what:    "two directories, neither of them this file's",
			lenders: []string{"core/debug.go", "wasm/verify/check.go"},
			example: "render/x.go",
			want:    []string{"core/debug.go", "wasm/verify/check.go"},
			rows:    2, near: 0, dirs: 2,
			says: "none of the 2 in this file's own directory",
		},
		{
			// The arm the whole heuristic exists for, which no run in this
			// repository has reached: a far row sorted BEHIND a near one that
			// comes after it alphabetically.
			what:    "a near row that alphabetises last",
			lenders: []string{"aaa/first.go", "zzz/near.go", "zzz/other.go"},
			example: "zzz/x.go",
			want:    []string{"zzz/near.go", "zzz/other.go", "aaa/first.go"},
			rows:    3, near: 2, dirs: 2,
			says: "2 of the 3 in this file's own directory and the rest across 1 more",
		},
	} {
		got := citationLendersNear(c.lenders, c.example)
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: citationLendersNear(%v, %q) is %v and the row wants %v.\n\n"+
				"An exemption is a claim about a file's CONTENT and all this file has "+
				"is a path, so directory is the one thing in hand that correlates "+
				"with content at all — and the rows a reader is shown are the first "+
				"two of this list. An order that is not this one puts a different "+
				"pair of arguments in front of them.",
				c.what, c.lenders, c.example, got, c.want)
		}
		// A permutation, asked apart from the order: a sort that dropped a row
		// would satisfy any `want` short enough to match it.
		bag, back := append([]string(nil), c.lenders...), append([]string(nil), got...)
		sort.Strings(bag)
		sort.Strings(back)
		if !slices.Equal(bag, back) {
			t.Errorf("%s: citationLendersNear returned %v over an input of %v.\n\n"+
				"The count beside this list — \"and N more of this kind\" — is taken "+
				"off the returned slice, so a row dropped here is an exemption a "+
				"reader is never shown and is not told about either. Two groups "+
				"appended is a permutation by construction; this is the assertion "+
				"that it stayed one.", c.what, got, c.lenders)
		}
		rows, near, dirs, kinds := citationLenderSpread(
			c.lenders, c.example, map[string][]string{".go": c.lenders})
		if rows != c.rows || near != c.near || dirs != c.dirs {
			t.Errorf("%s: citationLenderSpread reports %d rows, %d near, %d "+
				"directories; the row is %d, %d, %d.\n\n"+
				"Those three are what separate an order the sort chose from one it "+
				"could not have chosen, and the clause a passing run prints is built "+
				"out of them. A number that does not describe the ordering leaves "+
				"\"nearest this file first\" qualified by a sentence about some other "+
				"list.", c.what, rows, near, dirs, c.rows, c.near, c.dirs)
		}
		why := citationLenderWhy(rows, near, dirs, kinds, 1)
		if !strings.Contains(why, c.says) {
			t.Errorf("%s: citationLenderWhy does not say %q.\n\nIt said: %q\n\n"+
				"The three states send a reader to different conclusions — an order "+
				"the sort decided, an order it could not have decided, and an order "+
				"there was nothing to decide about — and one sentence covering all "+
				"three is the count-shaped answer this line was rebuilt out of.",
				c.what, c.says, why)
		}
	}
	// And the kind census one level up, which is the population the heuristic
	// can ever be about.
	//
	// Asked over a table rather than over one kind's rows: a repository where
	// no kind has lenders in more than one directory is one where this sort is
	// dead code wearing a sentence, and the line should be able to say so.
	for _, c := range []struct {
		what  string
		table map[string][]string
		kinds int
	}{
		{what: "every kind in one directory", kinds: 0, table: map[string][]string{
			".go":  {"core/a.go", "core/b.go"},
			".mjs": {"wasm/verify/a.mjs"},
		}},
		{what: "one kind spread, one not", kinds: 1, table: map[string][]string{
			".go":  {"core/a.go", "wasm/verify/b.go"},
			".mjs": {"wasm/verify/a.mjs", "wasm/verify/b.mjs"},
		}},
		{what: "both spread", kinds: 2, table: map[string][]string{
			".go":  {"core/a.go", "wasm/verify/b.go"},
			".mjs": {"core/a.mjs", "wasm/verify/b.mjs"},
		}},
	} {
		_, _, _, kinds := citationLenderSpread(
			c.table[".go"], "render/x.go", c.table)
		if kinds != c.kinds {
			t.Errorf("%s: citationLenderSpread reports %d kinds with rows in more "+
				"than one directory and the row is %d (%v).\n\n"+
				"That count is the population under the whole heuristic. At zero the "+
				"sort cannot separate anything anywhere in the table, and the line "+
				"printing \"nearest this file first\" over a list that was going to "+
				"come out in that order regardless is a claim about a sort presented "+
				"as a claim about a match.", c.what, kinds, c.kinds, c.table)
		}
	}
}

// citationClipWhy is one exemption's argument, bounded.
//
// Cut at a word rather than mid-token, and marked, so a reader can see that
// what they are looking at is the head of a sentence and not the whole of a
// thin one. The full text is in citationExempt and the line says so.
func citationClipWhy(why string) string {
	if len(why) <= citationWhyBudget {
		return why
	}
	cut := why[:citationWhyBudget]
	if i := strings.LastIndexByte(cut, ' '); i > citationWhyBudget/2 {
		cut = cut[:i]
	}
	return cut + "…"
}

// The two enumerations citationExemptVerdict's answer is decided by.
//
// # The line the whole pair rests on
//
// citationExemptVerdict tells a renamed exemption from a silent one, and a
// fixture holds it to both arms. What feeds it was two expressions in the walk
// and nothing looked at either: `reached` came from `considered`, every path
// the enumeration opened, and `cited` came from a set built while walking
// `files`, the paths that turned out to HAVE citations. Two lists, two
// different questions, and the verdict function cannot tell them apart — hand
// it the wrong pair and it reports the wrong failure with complete confidence.
//
// The interesting swap is `reached`. Taken from `files` instead — which is the
// natural mistake, because that is the list the loop above walks and the only
// one in scope where the exemption is first noticed — an exemption whose file
// is still sitting there and has simply stopped citing anything reads as NOT
// REACHED, and the message sends the reader off to look for a rename that
// never happened. That is the exact confusion the ORDER of the two arms inside
// citationExemptVerdict exists to prevent, arriving one level up where the
// ordering cannot see it.
//
// So the pair is a function of the two enumerations, and the fixture below
// hands it the case that separates them: a path the walk opened and found no
// citation in.
func citationExemptInputs(path string, considered map[string]string,
	files []citing) (reached, cited bool) {

	_, reached = considered[path]
	for _, f := range files {
		if f.path == path {
			cited = true
			break
		}
	}
	return reached, cited
}

// Whether one row of citationExempt still means what it says, and if not, which
// of the two ways it stopped.
//
// # The two failures are one sentence apart and send the reader to two places
//
// An exemption is a claim about a file: this one numbers things of its own, so
// a `check N` in it is not a citation of browser.mjs's sequence. That claim can
// go stale in two ways, and telling them apart is the whole value of reporting
// either.
//
// NOT REACHED means the enumeration never saw the path — a rename, a delete, or
// a prefix that citationSkipDirs now excludes. The row can never be satisfied,
// and the file it was written about may be sitting somewhere else citing checks
// with nobody looking.
//
// REACHED AND SILENT means the file is still there and has stopped citing
// anything. The exemption is now dead weight: it goes on telling the next
// reader that this file numbers things of its own, and the day it starts citing
// browser.mjs's sequence instead, this table waves it through.
//
// Reported in that order, because "not reached" subsumes "did not cite": a path
// the walk never opened has no citations by construction, and reporting the
// second would send the reader to look at an exemption when what happened was a
// rename.
//
// Returns "" when the row is doing its job, which is every row today.
func citationExemptVerdict(path, why, from string, reached, cited bool) string {
	switch {
	case !reached:
		return fmt.Sprintf("%s is exempted from the citation check (%s) and the "+
			"enumeration (%s) does not reach it at all.\n\n"+
			"citationSkipDirs classifies PREFIXES and is asked of git; this table "+
			"classifies FILES and was asked of nothing. A renamed or deleted path "+
			"leaves a row here that can never be satisfied, and the failure it "+
			"produces is the other one — \"this file no longer cites a check\" — "+
			"which sends the reader to look at an exemption when what happened was "+
			"a rename.", path, why, from)
	case !cited:
		return fmt.Sprintf("%s is exempted from the citation check (%s) and no longer "+
			"cites a check at all. An exemption outlives its reason silently: it goes "+
			"on telling the next reader that this file numbers things of its own, and "+
			"the day it starts citing browser.mjs instead nothing looks.", path, why)
	}
	return ""
}

// What each sense of "this repository's file" costs whoever reads a failure.
//
// # The decision nobody had made
//
// repoFile carries the sense so a failure can say who has to act, and until
// this table that was the whole of it: the phrase reached a message and no
// assertion. A citation in a committed file and one in a file still on
// somebody's desk were the same failure at the same severity — which may well
// be right, and was not a decision anybody made. It was the enumeration
// happening to tell the reader something and nothing downstream caring.
//
// It is a decision now, and the decision is that all three fail. The check
// exists so that every `check N` in the repository is an address into
// browser.mjs, and a bad address is a bad address whether or not it has been
// committed yet — an untracked one is a citation about to become a committed
// one, and the moment to fix it is now. What differs is who acts and where, and
// that is what the row says.
//
// The value of writing it down is the third sense. senseOnDisk comes from the
// walk, which runs on a machine with no git and cannot say whether the file is
// the repository's at all; it reads as the weakest of the three and is held to
// the same rule, which is a choice rather than an oversight. And the next sense
// added to the enumeration fails here until somebody makes the same choice for
// it, rather than inheriting one.
// citationSense is what one sense decides, and what it tells the reader.
//
// # The decision, and where it used to live
//
// The three sentences were the whole of this table, and the check read them
// only to paste one into a failure it was going to produce anyway. So "all
// three senses fail" was not a row anybody could point at: it was the absence
// of a branch. The sameness was the argument's conclusion and also its
// mechanism, and a row that changed its mind — an untracked citation as a
// warning, say, while somebody is mid-edit — had nowhere to say so.
//
// Fails is that branch, made into data. It reads the same today, because the
// decision is the same and it is argued above: a bad address is a bad address
// whether or not it has been committed. What has changed is that the decision
// is now written down where it can be changed, and that the check consults it
// instead of not having one.
type citationSense struct {
	// Fails is whether a citation that names no check is an error in a file
	// reached this way, rather than something a reader is merely told about.
	//
	// True in all three rows, so only one arm of the branch it decides has ever
	// executed against a real repository. Both are held as values by
	// TestCitationVerdictsAreDecidedByValues; see citationSenseVerdict for the
	// guard that keeps a table of falses from being a walk that cannot fail.
	Fails bool
	// Act is what whoever reads the failure has to do about it, which is the
	// part that really does differ between the three.
	Act string
}

var citationSenses = map[string]citationSense{
	senseTracked: {true, "The parenthesis is which sense of \"this repository's file\" the " +
		"enumeration used. This one is tracked: the citation is in the history, so " +
		"fixing it is a commit."},
	senseUntracked: {true, "The parenthesis is which sense of \"this repository's file\" the " +
		"enumeration used. This one is not committed yet: the citation is an edit in " +
		"the working tree of whoever is running this, and fixing it is a save. It " +
		"fails here rather than waiting for the commit, because a citation that is " +
		"wrong now is wrong when it lands."},
	senseOnDisk: {true, "The parenthesis is which sense of \"this repository's file\" the " +
		"enumeration used. This one came from the filesystem walk, which runs where " +
		"git could not answer and cannot say whether this file is the repository's " +
		"at all — a source tarball, a container image, an export. It is held to the " +
		"same rule as the other two: a `check N` that names nothing is wrong wherever " +
		"the file came from."},
}

// Whether the senses this run actually met can produce a failure at all.
//
// The table is allowed to decide that a sense is reported and not failed — that
// is the point of Fails being a field. What it must not be allowed to decide is
// that every sense in front of it today is one of those, because then the walk
// runs, finds every bad citation in the repository and reports a pass.
//
// Asked of the senses the ENUMERATION produced rather than of the whole table,
// which is the sharper question: on a machine with no git every file comes back
// as senseOnDisk, so a single row flipped to false would empty this check on
// exactly the machines that have the weakest enumeration to begin with, and the
// table would still look like two thirds of a check.
//
// Returns "" while at least one sense present can fail.
func citationSenseVerdict(present map[string]bool) string {
	can := []string{}
	for sense := range present {
		if citationSenses[sense].Fails {
			can = append(can, sense)
		}
	}
	if len(can) > 0 {
		return ""
	}
	got := make([]string, 0, len(present))
	for sense := range present {
		got = append(got, sense)
	}
	sort.Strings(got)
	return fmt.Sprintf("every file this enumeration reached came back as one of %v, "+
		"and citationSenses marks none of those as failing.\n\n"+
		"That is a walk that reads every `check N` in the repository, finds the ones "+
		"naming checks that do not exist, and reports a pass. A sense is allowed to "+
		"be reported rather than failed — that is what the Fails field is for — but "+
		"not every sense a run can see, or this check is only telling somebody "+
		"something it has already decided not to act on.", got)
}

// One citation, judged: the sentence to report about it, and whether reporting
// it is a failure. Returns "" while the number is an address browser.mjs has.
//
// # Why this is a function and not four lines in the loop
//
// It was four lines in the loop, and the loop is inside a check that walks the
// repository: the failing arm runs whenever a `check N` is stale, and the other
// arm runs when citationSenses says a sense is reported rather than failed —
// which no row says and no enumeration can bring about. It was reachable by
// editing a row, which is how it was tested, and an arm whose only fixture is
// somebody's uncommitted edit is an arm nobody has checked.
//
// As a function of values both arms are two rows of a table. Same move as
// fold.mjs's foldVerdict, gate.sh's jvm_harness_verdict, browser.mjs's
// startupVerdict and mobile/verify's composeSourcesVerdict — the decision
// stated where it can be handed its inputs, and the test that holds it not
// having to arrange a repository to get there.
//
// checks is how many the sequence has, so a citation is in range when it is
// between 1 and that. Zero and negatives are out of range like anything else:
// `check 0` is not an address either, and a walk that produced one is reporting
// a number it read rather than one it invented.
func citationVerdict(path, sense string, cite, checks int, act citationSense) (string, bool) {
	if cite >= 1 && cite <= checks {
		return "", false
	}
	return fmt.Sprintf("%s (%s) cites check %d, and browser.mjs has %d. Either the "+
		"citation was not moved when the sequence was renumbered, or it names a check "+
		"that no longer exists.\n\n"+
		"%s", path, sense, cite, checks, act.Act), act.Fails
}

// citationReporter is the half of the loop above that no pure function can be:
// which of the testing package's two reporters a verdict is handed to.
//
// Two methods, because two is what the choice makes. testing.TB has these and a
// great deal else, and a fake of the whole of it would be a page of methods
// nothing calls; *testing.T satisfies this as it stands, which the declaration
// below asserts at compile time.
type citationReporter interface {
	Error(args ...any)
	Log(args ...any)
}

var _ citationReporter = (*testing.T)(nil)

// citationReport hands one verdict to the reporter it asks for.
//
// # Why this is a function and not three lines in the loop
//
// The same argument citationVerdict carries, one level up, and the last note on
// this file said so: `switch { case say == "": ; case fails: Error; default:
// Log }` was three lines inside a walk of the repository, and no fixture
// reached two of the three. Every sense in citationSenses fails, so the Log arm
// had never run; and a `default` that called Error instead would have passed
// every test in this file, because no shipped row gets there.
//
// The silent arm is the one worth stating rather than reading off the switch.
// An empty sentence reaches NEITHER reporter, whatever fails says: a citation
// in range has nothing wrong with it, an Error there would fail every run, and
// a Log there would print one line for every citation in the repository. The
// ordering of the arms is what decides that, and until this function had a test
// the ordering was a fact about three lines nobody could ask a question of.
func citationReport(rep citationReporter, say string, fails bool) {
	switch {
	case say == "":
	case fails:
		rep.Error(say)
	default:
		rep.Log(say)
	}
}

// citationRecorder is a citationReporter that remembers rather than reports.
type citationRecorder struct {
	errors []string
	logs   []string
}

func (r *citationRecorder) Error(args ...any) {
	r.errors = append(r.errors, fmt.Sprint(args...))
}

func (r *citationRecorder) Log(args ...any) {
	r.logs = append(r.logs, fmt.Sprint(args...))
}

// All four inputs citationReport can be handed, and what each does.
//
// Two of the four are states this repository cannot produce — a sentence with
// fails false needs a sense citationSenses does not have — and the pair with an
// empty sentence is the one the switch's ORDER decides rather than its
// conditions. Asked here, where the inputs are arguments.
func TestCitationReportChoosesItsReporterFromTheVerdict(t *testing.T) {
	const say = "a stale citation, said once"
	for _, tc := range []struct {
		what   string
		say    string
		fails  bool
		errors []string
		logs   []string
	}{
		{what: "a citation in range says nothing and reaches neither reporter"},
		{
			what:  "an empty sentence stays silent even where the sense fails, which is the arm the switch's order decides",
			fails: true,
		},
		{
			what: "a sentence from a failing sense is an error", say: say,
			fails: true, errors: []string{say},
		},
		{
			what: "a sentence from a sense that is merely reported is a log", say: say,
			logs: []string{say},
		},
	} {
		t.Run(tc.what, func(t *testing.T) {
			var rec citationRecorder
			citationReport(&rec, tc.say, tc.fails)
			if !slices.Equal(rec.errors, tc.errors) {
				t.Errorf("citationReport(%q, fails=%v) reported %q as errors, want %q.\n\n"+
					"%s", tc.say, tc.fails, rec.errors, tc.errors, tc.what)
			}
			if !slices.Equal(rec.logs, tc.logs) {
				t.Errorf("citationReport(%q, fails=%v) reported %q as logs, want %q.\n\n"+
					"%s", tc.say, tc.fails, rec.logs, tc.logs, tc.what)
			}
		})
	}
}

// Where citationExemptVerdict's two inputs come from.
//
// # The half of the pair a fixture over the verdict cannot reach
//
// The test below hands citationExemptVerdict all four (reached, cited) pairs
// and holds each to its sentence. That is the decision. It says nothing about
// whether the walk computes the pair correctly — and the pair is two lookups
// in two different maps, which is the one line a correct verdict function
// still rests on.
//
// The case that separates the two enumerations is the third one here: a path
// the walk opened and found no citations in. It is in `considered` and NOT in
// `files`, so a `reached` derived from the wrong list turns "still there,
// stopped citing" into "renamed or deleted" — a confident report about a file
// that is exactly where the table says it is.
//
// The fourth row is unreachable from a real enumeration, because citingFiles
// only records a citation for a path it considered. It is here because the
// function is defined over both maps independently and an implementation that
// derived one from the other would pass the other three.
func TestCitationExemptInputsComeFromTheTwoEnumerations(t *testing.T) {
	const path = "wasm/verify/gen.go"
	for _, tc := range []struct {
		what       string
		considered map[string]string
		files      []citing
		reached    bool
		cited      bool
	}{
		{
			what:       "opened and citing",
			considered: map[string]string{path: senseTracked},
			files:      []citing{{path: path, sense: senseTracked, cites: []int{1}}},
			reached:    true, cited: true,
		},
		{
			what:       "never opened and not citing is a rename",
			considered: map[string]string{"wasm/verify/browser.mjs": senseTracked},
			files: []citing{
				{path: "wasm/verify/browser.mjs", sense: senseTracked, cites: []int{1}},
			},
		},
		{
			what:       "opened and not citing is the case the two lists disagree about",
			considered: map[string]string{path: senseTracked},
			files: []citing{
				{path: "wasm/verify/browser.mjs", sense: senseTracked, cites: []int{1}},
			},
			reached: true,
		},
		{
			what:       "citing without having been considered, which no enumeration produces",
			considered: map[string]string{},
			files:      []citing{{path: path, sense: senseTracked, cites: []int{1}}},
			cited:      true,
		},
	} {
		t.Run(tc.what, func(t *testing.T) {
			reached, cited := citationExemptInputs(path, tc.considered, tc.files)
			if reached != tc.reached || cited != tc.cited {
				t.Errorf("citationExemptInputs(%q) is (reached=%v, cited=%v), want "+
					"(%v, %v).\n\n"+
					"`reached` is a question about the ENUMERATION — did the walk open "+
					"this path — and `cited` is a question about what was found in it. "+
					"Reading either off the other list is what makes a file that is "+
					"still there and has stopped citing arrive as a rename, which is "+
					"the failure citationExemptVerdict's arm order exists to prevent "+
					"and cannot prevent from up here.\n\n%s",
					path, reached, cited, tc.reached, tc.cited, tc.what)
			}
		})
	}
}

// All four inputs citationExemptVerdict can be handed, and which of them says
// something.
//
// Three arms, and the walk above reaches one: every row of citationExempt names
// a file the enumeration finds and that still cites a check, so the two failing
// arms are unreachable from any repository this test can be pointed at. That is
// the same position citationVerdict and citationReport were in, and the same
// answer — the decision is a function and the inputs are arguments.
//
// The fourth input is the one the ORDER decides rather than the conditions: a
// path that was not reached and did not cite is both failures at once, and has
// to report the rename, because a walk that never opened a file cannot have
// seen a citation in it.
func TestCitationExemptVerdictTellsARenameFromASilentFile(t *testing.T) {
	const path, why, from = "wasm/verify/gen.go", "it numbers its own probes", "git"
	for _, tc := range []struct {
		what             string
		reached, cited   bool
		wantSay          bool
		wantSubstring    string
		wantNotSubstring string
	}{
		{
			what:    "a row doing its job says nothing",
			reached: true, cited: true,
		},
		{
			what:    "a file the enumeration never reached is a rename",
			cited:   true,
			wantSay: true, wantSubstring: "does not reach it at all",
		},
		{
			what:    "a file still there and no longer citing is dead weight",
			reached: true,
			wantSay: true, wantSubstring: "no longer cites a check at all",
		},
		{
			what:    "neither reached nor citing reports the rename, which is the arm the order decides",
			wantSay: true, wantSubstring: "does not reach it at all",
			wantNotSubstring: "no longer cites a check at all",
		},
	} {
		t.Run(tc.what, func(t *testing.T) {
			got := citationExemptVerdict(path, why, from, tc.reached, tc.cited)
			if (got != "") != tc.wantSay {
				t.Errorf("citationExemptVerdict(reached=%v, cited=%v) returned %q, and "+
					"this input is supposed to say %s.\n\n%s",
					tc.reached, tc.cited, got,
					map[bool]string{true: "something", false: "nothing"}[tc.wantSay],
					tc.what)
				return
			}
			if tc.wantSubstring != "" && !strings.Contains(got, tc.wantSubstring) {
				t.Errorf("citationExemptVerdict(reached=%v, cited=%v) said %q, which "+
					"does not carry %q.\n\nThe two failures send the reader to two "+
					"different places — a path that moved, and a file that stopped "+
					"citing — so which sentence comes back is the whole of what the "+
					"report is for.\n\n%s",
					tc.reached, tc.cited, got, tc.wantSubstring, tc.what)
			}
			if tc.wantNotSubstring != "" && strings.Contains(got, tc.wantNotSubstring) {
				t.Errorf("citationExemptVerdict(reached=%v, cited=%v) said %q, which "+
					"carries %q and should not.\n\n%s",
					tc.reached, tc.cited, got, tc.wantNotSubstring, tc.what)
			}
		})
	}
}

// Both verdicts, over the inputs the repository cannot produce.
//
// citationSenses says every sense fails, and the argument for that is written
// above the table. The consequence is that two decisions in this file have an
// arm no run reaches: citationVerdict's reported-not-failed arm, and
// citationSenseVerdict's refusal. Both are functions of their arguments, so
// both arms are rows here — the same reason fold_test.mjs exists for a guard
// nine bands have never tripped.
//
// What this does NOT do is assert that the shipped table has a false in it. It
// asserts what the code would do if it did, which is the half that stops being
// true silently.
func TestCitationVerdictsAreDecidedByValues(t *testing.T) {
	fails := citationSense{Fails: true, Act: "commit the fix"}
	reports := citationSense{Fails: false, Act: "somebody is mid-edit"}

	for _, c := range []struct {
		what      string
		cite      int
		checks    int
		act       citationSense
		wantSay   bool
		wantFails bool
	}{
		{"the first check", 1, 12, fails, false, false},
		{"the last check", 12, 12, fails, false, false},
		{"one past the last", 13, 12, fails, true, true},
		{"check 0, which is not an address", 0, 12, fails, true, true},
		{"a negative, which a renumbering cannot produce and a parser can",
			-1, 12, fails, true, true},
		// The arm citationSenses has never taken. A sense the table reports
		// rather than fails still produces the sentence — the reader is told —
		// and the test does not fail on it.
		{"out of range under a sense that only reports", 13, 12, reports, true, false},
		{"in range under a sense that only reports", 3, 12, reports, false, false},
	} {
		say, got := citationVerdict("some/file.go", "tracked", c.cite, c.checks, c.act)
		if (say != "") != c.wantSay {
			t.Errorf("%s: citationVerdict said %q, wanted a sentence: %v", c.what, say, c.wantSay)
			continue
		}
		if got != c.wantFails {
			t.Errorf("%s: citationVerdict reports fails=%v, want %v.\n\n"+
				"Whether a stale citation is an error is citationSenses' decision and "+
				"this function's to carry. An arm that stopped being carried would show "+
				"up nowhere else: every row of the shipped table fails, so the "+
				"reported-not-failed path is never taken by a real run.",
				c.what, got, c.wantFails)
		}
		if say != "" && !strings.Contains(say, c.act.Act) {
			t.Errorf("%s: the sentence does not carry the sense's Act. That sentence is "+
				"the whole of what the reader is told to do about it", c.what)
		}
	}

	// And the guard over the senses a run actually met.
	for _, c := range []struct {
		what    string
		present map[string]bool
		refuse  bool
	}{
		{"the three senses this repository produces",
			map[string]bool{senseTracked: true, senseUntracked: true, senseOnDisk: true}, false},
		{"only the walk's sense, which is every machine without git",
			map[string]bool{senseOnDisk: true}, false},
		{"a sense with no row at all, which classifies as not failing",
			map[string]bool{"invented": true}, true},
		// The state the guard exists for: everything the enumeration met is a
		// sense somebody marked as reported-only, so the walk reads every bad
		// citation in the repository and returns a pass.
		{"nothing present, which only a fixture can arrange",
			map[string]bool{}, true},
	} {
		if got := citationSenseVerdict(c.present) != ""; got != c.refuse {
			t.Errorf("%s: citationSenseVerdict refuses=%v, want %v", c.what, got, c.refuse)
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

// Every skip prefix's relationship to git, asked of git.
//
// citationSkipDirs says of each entry whether git already leaves it out and
// how. That is the kind of claim that is true the day it is written and stays
// on the page afterwards: a .gitignore is a file somebody edits, and an entry
// that moved from "a second statement of something already stated" to "the only
// thing keeping generated sources out of this check" reads exactly the same.
//
// The failure this is against is silent in both directions. A redundant entry
// that has become load-bearing is a check resting on a line documented as
// decorative — the next person deleting it as dead weight would be deleting the
// guard. A load-bearing entry that has become redundant is dead weight that
// reads as a guard, which is the same thing citationExempt's own assertion
// exists to catch one level up.
//
// git is the authority on both halves and answers cheaply: `check-ignore` for
// the rules, and the listing itself for what those rules produce. On a machine
// with no git there is nothing to ask and the check says so rather than
// passing — it is a claim ABOUT git, and the fallback walk it does not describe.
func TestTheCitationSkipsGitAlreadyMakes(t *testing.T) {
	root := filepath.Join("..", "..")

	files, err := repositoryFiles(root)
	if err != nil {
		t.Skipf("git could not enumerate %s (%v), and every claim here is about what "+
			"git's own rules do. The citation walk falls back to the filesystem on this "+
			"machine and says so; this check has nothing to ask.", root, err)
	}
	if len(files) == 0 {
		t.Fatalf("`git ls-files` succeeded in %s and listed no files, so \"git lists "+
			"nothing under this prefix\" is true of every prefix below and the whole "+
			"table would pass on a repository that is not this one", root)
	}

	// Which prefixes git actually offers something under. A prefix git lists
	// files for is one the enumeration would read if this table did not stop
	// it.
	listed := map[string]repoFile{}
	for _, f := range files {
		for _, skip := range citationSkipDirs {
			if f.path == skip.prefix || strings.HasPrefix(f.path, skip.prefix+"/") {
				if _, seen := listed[skip.prefix]; !seen {
					listed[skip.prefix] = f
				}
			}
		}
	}

	for _, skip := range citationSkipDirs {
		// `check-ignore` exits 0 when a path matches an exclude rule and 1 when
		// it does not, and it answers for a path that does not exist — which is
		// the case that matters, since a build directory is absent on a clean
		// checkout and its entry has to be checkable there too.
		ignored := exec.Command("git", "-C", root, "check-ignore", "-q",
			skip.prefix).Run() == nil

		switch skip.kind {
		case gitNeverLists:
			if got, any := listed[skip.prefix]; any {
				t.Errorf("citationSkipDirs calls %s gitNeverLists (%s) and `git ls-files` "+
					"lists %s (%s) under it. git does not enumerate its own store, so "+
					"this prefix is now naming something else — and whatever that is, it "+
					"is being skipped by an entry whose reason does not describe it.",
					skip.prefix, skip.why, got.path, got.sense)
			}
		case gitIgnores:
			if !ignored {
				t.Errorf("citationSkipDirs calls %s gitIgnores (%s) and `git check-ignore` "+
					"says no rule excludes it.\n\n"+
					"That entry is documented as a second statement of something the "+
					"platform .gitignore files already say, kept for the fallback walk. "+
					"It is not: git would offer these files, so the prefix is the only "+
					"thing keeping generated sources out of the citation check on every "+
					"machine. Either restore the ignore rule or move this entry to "+
					"gitWouldList, where a reader will not delete it as decoration.",
					skip.prefix, skip.why)
			}
			if got, any := listed[skip.prefix]; any {
				t.Errorf("citationSkipDirs calls %s gitIgnores and git lists %s (%s) "+
					"under it anyway. An ignored path git still lists is a file that was "+
					"committed before the rule arrived; an unignored one is the rule "+
					"having gone. Either way the entry is classified by an effect it no "+
					"longer has.", skip.prefix, got.path, got.sense)
			}
		case gitWouldList:
			if ignored {
				t.Errorf("citationSkipDirs calls %s gitWouldList (%s) and `git check-ignore` "+
					"says a rule already excludes it. The entry is documented as "+
					"load-bearing under git and is not — it is now dead weight that "+
					"reads as a guard, which is exactly what an outlived citationExempt "+
					"row is one level up.", skip.prefix, skip.why)
			}
		}
	}
}

// The directories a citation walk does not enter, and why — and, for each one,
// how it relates to git's own enumeration.
//
// Kept as path prefixes rather than as base names so that a directory named
// `build` somewhere else in the tree is not skipped by accident.
//
// # Why each entry carries a kind
//
// This list used to say, in prose, that four of its entries were redundant
// under git and two were load-bearing. Both halves were assumptions. git
// excludes a build directory because a .gitignore names it, and a .gitignore is
// a file somebody edits: the day `app/build/` came out of android/.gitignore,
// that entry would go from a second statement of something already stated to
// the only thing keeping generated sources out of the citation check — the
// enumeration still right, for a reason that had moved, and nothing anywhere
// saying so.
//
// So the relationship is declared per entry and asked of git by
// TestTheCitationSkipsGitAlreadyMakes. The kinds are the three that exist:
//
//	gitNeverLists   git's own directory. No rule excludes it; `ls-files` does
//	                not enumerate it, and only the disk walk can reach it.
//	gitIgnores      a .gitignore names it, so git leaves it out. These entries
//	                exist for the walk, which has no exclude rules to read.
//	gitWouldList    git does NOT ignore it, so `--cached` or `--others` offers
//	                the files under it and this prefix is the only thing that
//	                keeps them out of the check. Load-bearing in both paths.
var citationSkipDirs = []skipDir{
	{".git", gitNeverLists, "git's own store"},
	// Saved sessions and plans are a RECORD of what was true when they were
	// written. A renumbering does not make last week's session doc wrong, and
	// rewriting one to keep a test green would be falsifying the record.
	//
	// Tracked, and being tracked is exactly why the prefix is needed: no
	// mechanism could make this decision, which is the other half of what
	// gitWouldList means.
	{"ai_docs", gitWouldList, "saved sessions are a record, not a claim about today's numbering"},
	// Build output: generated sources, jars, and a wasm binary, none of it
	// written by anybody here. Named by the two platform .gitignore files, so
	// under git these four are a second statement of something already stated
	// and are kept for the walk.
	{"android/build", gitIgnores, "Gradle build output"},
	{"android/.gradle", gitIgnores, "Gradle's own state"},
	{"android/app/build", gitIgnores, "the app module's build output"},
	{"ios/build", gitIgnores, "Xcode build state"},
	// mkdocs output. Nothing ignores it — it does not exist until somebody
	// runs a build, and when it does git offers every generated page under it
	// as an untracked file — so this prefix is what keeps a rendered copy of
	// the census from being read as a second citation of every check it
	// quotes.
	{"docs/site", gitWouldList, "mkdocs renders the whole of docs/ into it"},
}

// How a skip prefix relates to git's own enumeration. See citationSkipDirs.
type skipKind int

const (
	gitNeverLists skipKind = iota
	gitIgnores
	gitWouldList
)

// skipDir is one prefix the citation enumeration does not descend into.
type skipDir struct {
	prefix string
	kind   skipKind
	why    string
}

// citing is one file and the check numbers it names, with the sense in which
// the enumeration says the file is this repository's. See repoFile.
type citing struct {
	path  string
	sense string
	cites []int
}

// repoFile is one path and the sense in which git calls it this repository's.
//
// The two senses are not decoration. `--cached` is a file the history has, and
// `--others --exclude-standard` is a file somebody wrote and has not added yet;
// a citation failure naming the first is a commit that has to be fixed, and one
// naming the second is an edit still on the desk of whoever is running the test.
// The enumeration knows which, and until this type it threw the answer away and
// handed every failure the same sentence.
type repoFile struct {
	path string
	// sense is the phrase a failure carries. The walk has a third one,
	// because a filesystem cannot answer this question at all.
	sense string
}

const (
	senseTracked   = "tracked"
	senseUntracked = "written here and not committed yet"
	senseOnDisk    = "on disk; the walk cannot say whether git knows it"
)

// repositoryFiles asks git which files are this working tree's, in the two
// senses git has of the question.
//
// # Why git rather than the disk
//
// The enumeration below used to be a filesystem walk minus five directory
// prefixes, and that list was the whole of what stood between the citation
// check and a `node_modules` somebody adds: a vendored dependency carrying the
// words "check 3" in a changelog is a failure naming a file nobody here wrote,
// and the fix would have been a sixth prefix, and then a seventh.
//
// `git ls-files --cached` plus `git ls-files --others --exclude-standard` is
// every tracked file plus every untracked one git would offer to add. That is
// exactly the set a person here writes, and it keeps the property the walk was
// adopted for — `--others` covers a file written five minutes ago, so a new
// document is checked before it is committed.
//
// What remains outside it is a dependency vendored by COMMITTING it, which is
// this repository's file by every mechanical test there is. That failure is
// loud and names the file, and saying so is better than an enumeration
// pretending to cover it.
//
// # Two invocations rather than one
//
// `--cached --others` in one command returns one list, and which flag produced
// a given path is not recoverable from it. Asking twice is what makes the sense
// a fact the caller can print; `--others` already excludes anything tracked, so
// the two lists are disjoint and no path needs reconciling.
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
func repositoryFiles(root string) ([]repoFile, error) {
	var files []repoFile
	for _, listing := range []struct {
		args  []string
		sense string
	}{
		{[]string{"ls-files", "--cached", "-z"}, senseTracked},
		{[]string{"ls-files", "--others", "--exclude-standard", "-z"}, senseUntracked},
	} {
		out, err := exec.Command("git", append([]string{"-C", root}, listing.args...)...).Output()
		if err != nil {
			return nil, err
		}
		// NUL-separated, which is what -z buys: a path with a newline or a
		// quote in it is a path git would otherwise escape and this would have
		// to unescape.
		for _, p := range strings.Split(string(out), "\x00") {
			if p != "" {
				files = append(files, repoFile{path: p, sense: listing.sense})
			}
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	return files, nil
}

// walkedFiles is the enumeration for a machine with no git: every regular file
// under root, minus the skip prefixes.
//
// Kept rather than deleted, and it is not dead weight — a source tarball, a
// container image built by copying the tree in, and `go test` run from an
// export all reach it. What it cannot do is the thing git does for free, which
// is why the caller announces which enumeration it used — and why every file it
// produces carries senseOnDisk rather than one of git's two answers.
func walkedFiles(root string) ([]repoFile, error) {
	var out []repoFile
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
				if rel == skip.prefix {
					return fs.SkipDir
				}
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		out = append(out, repoFile{path: rel, sense: senseOnDisk})
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
func citingFiles(root string) (found []citing, considered map[string]string,
	from string, err error) {

	files, gitErr := repositoryFiles(root)
	from = "git ls-files"
	if gitErr != nil {
		from = fmt.Sprintf("a filesystem walk (git could not answer: %v)", gitErr)
		if files, err = walkedFiles(root); err != nil {
			return nil, nil, from, err
		}
	}

	// Every path this enumeration actually looked inside, with the sense it
	// came back under. Returned alongside the citations because the two tables
	// above ask different questions of it: citationExempt names paths and has
	// to know they still exist (a renamed file leaves a row nothing can ever
	// satisfy), and citationSenses classifies senses and has to know which ones
	// were produced.
	considered = map[string]string{}
	for _, f := range files {
		rel := f.path
		// The skip prefixes apply to both enumerations, and which of them each
		// prefix is actually doing work in is a fact about git's exclude rules
		// rather than a standing assumption — see citationSkipDirs and
		// TestTheCitationSkipsGitAlreadyMakes.
		skipped := false
		for _, skip := range citationSkipDirs {
			if rel == skip.prefix || strings.HasPrefix(rel, skip.prefix+"/") {
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
			return nil, nil, from, readErr
		}
		if bytes.IndexByte(raw, 0) >= 0 {
			continue
		}
		considered[rel] = f.sense
		if cites := citations(string(raw)); len(cites) > 0 {
			found = append(found, citing{path: rel, sense: f.sense, cites: cites})
		}
	}
	return found, considered, from, nil
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
