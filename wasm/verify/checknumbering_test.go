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

	seenExempt := map[string]bool{}
	for _, f := range files {
		if _, exempt := citationExempt[f.path]; exempt {
			seenExempt[f.path] = true
			continue
		}
		act, classified := citationSenses[f.sense]
		if !classified {
			continue // already reported above
		}
		for _, cite := range f.cites {
			if cite < 1 || cite > checks {
				// Which of the two the table says this sense gets. Today every
				// row fails; the branch is here because the decision is a field
				// now rather than the shape of this loop.
				say := t.Errorf
				if !act.Fails {
					say = t.Logf
				}
				say("%s (%s) cites check %d, and browser.mjs has %d. Either the "+
					"citation was not moved when the sequence was renumbered, or it "+
					"names a check that no longer exists.\n\n"+
					"%s",
					f.path, f.sense, cite, checks, act.Act)
			}
		}
	}

	// The exemptions, in the two ways one can outlive its reason.
	for path, why := range citationExempt {
		if _, reached := considered[path]; !reached {
			t.Errorf("%s is exempted from the citation check (%s) and the enumeration "+
				"(%s) does not reach it at all.\n\n"+
				"citationSkipDirs classifies PREFIXES and is asked of git; this table "+
				"classifies FILES and was asked of nothing. A renamed or deleted path "+
				"leaves a row here that can never be satisfied, and the failure it "+
				"produces is the one below — \"this file no longer cites a check\" — "+
				"which sends the reader to look at an exemption when what happened was "+
				"a rename.", path, why, from)
			continue
		}
		if !seenExempt[path] {
			t.Errorf("%s is exempted from the citation check (%s) and no longer cites a "+
				"check at all. An exemption outlives its reason silently: it goes on "+
				"telling the next reader that this file numbers things of its own, and "+
				"the day it starts citing browser.mjs instead nothing looks.", path, why)
		}
	}
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
	// True in all three rows. See citationSenseVerdict for the guard that keeps
	// a table of falses from being a walk that cannot fail.
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
