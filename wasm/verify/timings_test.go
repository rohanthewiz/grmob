package main

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

// The machine every wall-clock number in this package's prose was taken on.
//
// # Why a number here needs a machine attached and the others do not
//
// Everything else recorded in this package is re-derived on every run.
// affordedMeasuredOn keeps eighty names and four counts and the census over
// them is re-walked; foldMeasuredOn keeps the fold's counts and names the
// Unicode build they are a reading of, so a count that moved is a count about
// the DATA. Either way a number that moved is a finding.
//
// The timings cannot work that way. `685ms`, `112ms`, `2530.0us`, `5.5s`,
// `128ms`: every one of them is a number somebody took once and typed in, and
// a number in a note is a number that has already moved.
//
// It cannot become an arm either, and not for want of trying. A wall clock is
// a reading of the machine, the load on it, the Go version's scheduler and
// whatever else was compiling at the time; an assertion over one would fail on
// a busy laptop and on every CI runner, and a test that fails for reasons
// nobody can act on is a test people learn to ignore. That is a worse outcome
// than an unattributed number.
//
// What a timing CAN carry is where it came from. `go test ./wasm/verify`
// spreads over 2.49–2.59s across seven runs on one idle machine, roughly 4%
// wide by itself — so a reader holding a 2.7s run against a 2.54s written down
// somewhere has nothing to reason with: the difference is inside one machine's
// own spread, or it is a regression, or it is a different computer, and the
// number alone distinguishes none of them. The spread is why the recorded
// figures are RANGES and the machine is why there is a record at all; the arm
// below says which computer this run is standing on when it is not this one.
//
// # Why this is its own file
//
// It began inside themenearmiss_test.go, named for that file's `afforded*`
// family, describing "every wall-clock number in this file". That was already
// untrue when it was written: inkglyph_test.go states the astral fold walk's
// cost twice, and it is as much a reading of this machine as any of
// themenearmiss's. A record cited by two files that lives inside one of them
// reads as that file's property, and the next person adding a timing to a
// third does not find it.
//
// # Why the fields are the ones a program can read
//
// A model name is what a person wants and what nothing can check, so the
// record carries both: `machine` for the reader, and four fields the runtime
// answers for itself. A mismatch in those is a machine difference stated as a
// fact rather than guessed at from a number that looks wrong.
//
// # Re-taking it
//
//	go test -count=1 ./wasm/verify                                  wholeFile
//	go test -count=1 -run TestHowWideTheNarrowerFold ./wasm/verify   foldWalk
//
// There is no third line, and there was: this table used to offer `go test
// -bench . -run '^$' ./wasm/verify` for "the per-walk ones" — the 96.7us,
// 7.4us and 2530.0us in themenearmiss_test.go's account of what affordedEachSet
// replaced. That command does nothing. The benchmarks it names were written to
// take those three numbers and removed once they had been taken, so the
// re-taking instruction outlived the thing it instructed. Those three are in
// the same position as the 128ms below and for the same reason: they are a
// record of a change that was already made, not a property anything asserts.
//
// Anything re-taken here is re-taken WITH this record: a run on another machine
// that updates a timing and leaves the machine alone has put the same
// unattributed number back, one commit later.
var verifyTimingsTakenOn = struct {
	// For a reader. Nothing checks this.
	machine string
	// For the arm. runtime answers all four.
	goos, goarch, goVersion string
	cores                   int
	// The whole package, `go test -count=1 ./wasm/verify`. Not the sum of the
	// numbers in the prose — those are pieces of it, taken at different times
	// — and the one number a reader is most likely to be holding this record
	// up against, because it is the one they get by running the tests.
	//
	// It moved 2.49–2.59s → 2.70–2.80s when two repository-wide censuses were
	// added: TestTheDottedVersionParsersAreTheOnesTheReasonCovers and
	// TestTheShortLeversAreTheOnesThisRepositoryHasDecidedOn, each of which
	// parses every Go file in the tree and reports 0.18s of doing it.
	//
	// Two arms at 0.18s against a total that moved 0.21s do not add up, and
	// the gap is not explained here for the same reason the 60% one in the
	// other record is not: these are two readings taken in two sessions, and a
	// difference of that size is what a wall clock is worth. Each of these
	// walks is 0.18s whether it runs alone or beside the others, which was
	// measured rather than assumed.
	//
	// # How many of them there are, which this comment used to get wrong
	//
	// It said three, naming the copies census — then called
	// TestEveryTimingsRecordIsTheSameShape, then
	// TestTheShapesThisRepositoryKeepsTwoCopiesOfAreInStep, now
	// TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn — as
	// the last.
	// There were four: TestEveryGitListingAsksForNulSeparatedPaths has parsed
	// the whole tree since before any of the other three existed and was
	// simply not counted. And a walk in this package is not always a test —
	// checkCitationsResolve is a helper, and it walks once per caller.
	//
	// So the count stopped being kept here. TestTheRepositoryWideWalksIn-
	// ThisPackageAreTheOnesDecidedOn reads it out of the source: SEVEN walks a
	// run, four of them parsing every Go file, each with a row saying what it
	// asks the repository and how deep it goes. About a second of the figure
	// below, and the number that decides when one shared parse becomes worth
	// its lifetime rules is that arm's repositoryParseBudget rather than a
	// sentence here.
	//
	// That arm costs 0.01s: it reads this ONE DIRECTORY and parses only the
	// files whose bytes name an enumeration — eight of thirty-seven where
	// this record was taken, and a ratio that moves with every file added
	// here, which is why the arm prints both numbers on every run rather than
	// leaving this sentence to be the record of them. A census of repository
	// walks that was itself a repository walk would have been the eighth
	// walk, and the figure below did not move for it — 2.70–2.80s became
	// 2.73–2.84s, which is one machine's spread.
	//
	// # And the readings the copies census grew, which is the same story again
	//
	// TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn took on two
	// more questions — the import-resolving helpers held identical across both
	// packages, and every core-count read held to being named in its package's
	// cores note — and both are read off declarations the walk had already
	// built. Measured by taking them out and putting them back: 0.18s to
	// 0.19s, which is the fourth parse walk costing a hundredth more than the
	// three beside it.
	//
	// The figure below moved further than that, 2.73–2.84s to 2.78–2.92s over
	// fourteen runs, and the 0.01s does not account for it. What the rest is
	// cannot be said from here — it is the width of one machine's own spread,
	// which is the reason this record holds ranges and the reason none of
	// these numbers is an assertion.
	//
	// # A hundredth further, and what was under it
	//
	// The top end went to 2.93s on one reading of seven the next time this
	// was taken, and the row is widened rather than left to be contradicted by
	// a run. Two things changed in between and they pull opposite ways: the
	// cores-note check grew a second direction, which reads the two
	// record-carrying directories at 0.011s (see repositoryWalkRow.besides),
	// and the walk-counting pass stopped recomputing what each function binds
	// once per walk name, which took six traversals out of seven.
	//
	// Neither is a hundredth of a second on its own and the pair of them
	// certainly is not. The same afternoon put internal/themehistory a tenth
	// above ITS recorded range with no change to that package at all, which
	// was measured properly there — alternating the old code and the new —
	// and came out the same for both. So this is the machine, said once in
	// each record rather than argued about twice.
	//
	// # And how wide this range actually is, which is wider than it says
	//
	// It is twenty-one readings across two sessions, which is more than most
	// figures here and is still not enough to have found its ends. The other
	// record learned that the expensive way in the same afternoon: its plain
	// row was set from three readings, contradicted by the next run, set
	// again, and contradicted again — a floor being CHASED rather than
	// measured — and settled only when it was taken from sixteen at once. The
	// argument is written out at internal/themehistory/timings_test.go's
	// wholePackage and it is not about that package.
	//
	// It applies to every figure in this record and to every inline number in
	// this package's prose, all of which were set from three or seven
	// readings. None of them is wrong. All of them are NARROWER THAN THE
	// TRUTH by roughly what that row gained when it was taken properly —
	// somewhere around a twentieth at each end — and a reader landing just
	// outside one of them has been told, here, that the range is not tight
	// enough to judge them by.
	//
	// Not fixed, because fixing it is a few hundred runs to learn what one
	// row has already said, and because the direction it is wrong in is the
	// harmless one: a range too narrow reports a difference that is not there,
	// which sends somebody to look and find nothing. A range too wide would
	// hide one.
	wholeFile string
	// TestHowWideTheNarrowerFoldIsAndWhatHoldsTheGap end to end, which is what
	// inkglyph_test.go's `128ms` sits inside.
	//
	// The 128ms itself is NOT re-taken here and the difference matters: it is
	// the astral walk alone, measured once from inside the test, and isolating
	// it again would mean either instrumenting the test to print a number for
	// a comment or running a second copy of the walk outside it. Both are
	// worse than an honest note. What is re-derivable without either is the
	// enclosing test's wall clock, which brackets it — so that is what is
	// recorded, and the 128ms is a sub-figure of it rather than a number this
	// record vouches for.
	//
	// # And whether it earns its place at all, which is a question this record
	// # cannot answer and inkglyph_test.go can
	//
	// A bracketed number from an unknown day is still a number from an unknown
	// day, and no amount of attribution makes it re-derivable. The question is
	// therefore not "how do we re-take it" but "what is it holding up", and
	// the answer turned out to be nothing that needs it: the astral walk's
	// AFFORDABILITY is demonstrated by the test running it on every green run,
	// and the walk still going all the way up is asserted by foldMeasuredOn's
	// six astral counts, which a narrowing would fail. Both are re-derived; the
	// wall clock is not, and does not have to be.
	//
	// So the figure is kept where it was taken, marked as the reading a past
	// decision was made on, and the sentence it used to carry now stands on
	// the arms instead. That is the resolution rather than a re-measurement:
	// the number was never the load-bearing part, and pricing it a second time
	// would have been work to keep something honest that could instead stop
	// being asked to hold anything.
	//
	// This field, foldWalk, is the one that IS re-takeable, and the command
	// for it is in the table above.
	foldWalk string
}{
	machine:   "Apple M3 (Mac15,13), macOS 26.2",
	goos:      "darwin",
	goarch:    "arm64",
	goVersion: "go1.26.1",
	cores:     8,
	wholeFile: "2.78–2.93s over twenty-one runs in two sessions",
	foldWalk:  "0.40–0.52s over seven runs, node v22.12.0",
}

// Which of this package's figures move with the core count, and which do not.
//
// # Why every record carries one of these
//
// `cores` is the one machine field that changes a recorded number by a term a
// reader can name, and the two records in this repository were saying
// different amounts about it: internal/themehistory's carries a measured table
// — 0.90s at one worker down to 0.22s at eight — and this one said nothing,
// which a reader holding a 3.1s run against the 2.78s below reads as "nobody
// measured that" rather than as "that is not where the difference is".
//
// Silence is the wrong answer either way round, so both records now state it
// and wasm/verify/copies_test.go holds every record to having one — and holds
// this one to naming every term in this package that actually scales.
//
// # Why the note names declarations and not only a family
//
// It is held to being COMPLETE, not merely to being there — copies_test.go's
// checkCoresNoteNamesEveryScaledTerm finds every declaration in this package
// that reads runtime.NumCPU or runtime.GOMAXPROCS and requires this sentence
// to name it. A note that says which terms scale is a claim about an inventory,
// and the inventory is the half a parse can settle: the numbers below are a
// wall clock and cannot be asserted, but a pool arriving in a third place is
// somebody adding code, and this fails on it.
//
// So `affordedKLeafBandWalk` and `affordedTwoStepBands` are written out. They
// are the whole of the `afforded*` family's concurrency — the reporting arm's
// own NumCPU is the comparison against the record rather than a term, and is
// skipped there by name.
//
// # What this package's answer actually is, which is not "nothing"
//
// The expensive half of wholeFile does not move. The four repository-wide
// walks hand every Go file in the tree to go/parser one after another — 389
// tracked Go files where this record was taken, 0.18s each, single-threaded,
// the same number on any machine — and foldWalk is a node process this package waits on
// rather than shares a core with.
//
// What does move is the afforded* band family in themenearmiss_test.go —
// affordedKLeafBandWalk and affordedTwoStepBands, which are the two that take
// `workers := runtime.GOMAXPROCS(0)` and divide their combinations across it.
// Measured by fixing GOMAXPROCS and re-running, three runs apiece on the eight
// cores named above:
//
//	GOMAXPROCS    the whole package
//	1             3.08–3.27s
//	2             2.78–2.88s
//	4             2.74–2.86s
//	8             2.75–2.85s
//
// Nine runs a row rather than three: the table has been taken three times —
// when it was first measured, when the copies census grew its two extra
// readings, and again when the row above it was widened to 2.78–2.93s. Each
// row holds every taking's range for the reason the other record's -race row
// does.
//
// The third one was the whole table and not the row that had moved, which is
// the point of it. Widening `wholeFile` and leaving these four would have made
// a record where the figure at the top was from one afternoon and the table
// explaining it was from another, with nothing on either saying which — a
// reader comparing a one-core run against the number above would have been
// comparing two days. That is the fault this record exists to end, and there
// is no version of it that is acceptable inside the record itself.
//
// Only the four-core row moved, by seven hundredths at the top end. The other
// three came back inside the ranges they already had, which is the useful
// half of re-taking a table nothing has changed. The shape did not move —
// one core is a tenth dearer and two is where it stops improving —
// which is the part the sentence below is about.
//
// Which is a different SHAPE from the other record's, and that is the part
// worth having: this one is flat from two cores upwards, so a core count that
// differs from eight is worth about a tenth of the figure and only between one
// and two. internal/themehistory's keeps improving all the way to eight,
// because its term is 88 git processes rather than one Go program's own
// goroutines. A reader on a four-core machine should expect this package's
// number and not that one.
// # How this has to be written, which is a constraint from outside
//
// Literals joined by `+`, and nothing else. wasm/verify/copies_test.go reads
// this note without running the package — stringLiteralValue evaluates a
// string constant of exactly that shape — and holds what it names to being
// terms this package still has, in both directions. A note assembled by a
// function, or built out of other constants, comes back empty and is reported
// rather than silently exempt, which is a finding about this declaration
// arriving in another package's test.
//
// The terms themselves go in BACKQUOTES. That is what says a word is meant as
// the name of a declaration rather than as English, and it is the only thing
// either direction reads — a term named in prose is neither claimed nor
// checked.
const coresAttribution = "The four repository-wide walks in this package are " +
	"single-threaded — go/parser over 389 tracked Go files " +
	"where this record was taken, 0.18s each — and " +
	"`foldWalk` is a node process. Those do not move with the core count. " +
	"Two declarations do, and they are the whole of it: " +
	"`affordedKLeafBandWalk` and `affordedTwoStepBands` in " +
	"themenearmiss_test.go each take `workers := runtime.GOMAXPROCS(0)` and " +
	"divide the afforded* band family's combinations across it. The package is 3.08–3.27s at one core, " +
	"2.78–2.88s at two, and flat from there to eight. So a differing core " +
	"count is worth about a tenth of the figure above, and only between one " +
	"core and two — which is the opposite shape from " +
	"internal/themehistory's, where the term is 88 git processes and the " +
	"improvement runs all the way to eight."

// This run says whether it is standing on the machine the timings came from.
//
// # Why this reports and does not assert
//
// A timing cannot be an arm — see verifyTimingsTakenOn — and neither can the
// machine, for a reason that is not the same: asserting the machine would fail
// on every computer that is not this one, which is every computer except this
// one. What is worth having is the SENTENCE, in front of the person comparing
// a number in a comment with a number on their screen, saying that the two
// were taken on different things before they go looking for a regression.
//
// So it is a Logf, and the file is honest about what that buys: it is visible
// under -v and on any run where something else in this package has failed,
// which are the two occasions anybody reads this output. A green quiet run
// does not need it.
//
// The one thing here that IS an assertion is that the record is filled in at
// all. A zeroed field is a record somebody added a timing to without saying
// where it came from, which is the state this whole thing exists to end, and
// it is a fact about the source rather than about the machine.
func TestTheTimingsInThisPackageSayWhichMachineTheyCameFrom(t *testing.T) {
	rec := verifyTimingsTakenOn
	if rec.machine == "" || rec.goos == "" || rec.goarch == "" ||
		rec.goVersion == "" || rec.cores == 0 || rec.wholeFile == "" ||
		rec.foldWalk == "" {
		t.Fatalf("verifyTimingsTakenOn has an empty field (%+v).\n\n"+
			"Every wall-clock number in this package's prose is attributed to "+
			"this record, and a record with a hole in it attributes them to "+
			"nothing. That is the state this record exists to end: a reader "+
			"a few percent off one of these numbers cannot tell a regression "+
			"from a different computer.", rec)
	}

	var differs []string
	if got := runtime.GOOS; got != rec.goos {
		differs = append(differs, fmt.Sprintf("GOOS %s against %s", got, rec.goos))
	}
	if got := runtime.GOARCH; got != rec.goarch {
		differs = append(differs, fmt.Sprintf("GOARCH %s against %s", got,
			rec.goarch))
	}
	// The toolchain, because the numbers are of code this compiles and the
	// scheduler that runs it. Compared as a prefix: a patch release is a
	// different toolchain and worth naming, and `devel` builds carry a suffix
	// no equality test would ever match.
	if got := runtime.Version(); !strings.HasPrefix(got, rec.goVersion) {
		differs = append(differs, fmt.Sprintf("%s against %s", got, rec.goVersion))
	}
	// NumCPU and not GOMAXPROCS: the band walks split their family across
	// GOMAXPROCS, which a caller can set, and what the record is about is the
	// machine underneath it.
	if got := runtime.NumCPU(); got != rec.cores {
		differs = append(differs, fmt.Sprintf("%d cores against %d", got,
			rec.cores))
	}

	if len(differs) == 0 {
		t.Logf("this run is on the machine the timings in this package were "+
			"taken on: %s, %s, %s/%s, %d cores. The whole package was %s "+
			"there, and the fold walk %s.",
			rec.machine, rec.goVersion, rec.goos, rec.goarch, rec.cores,
			rec.wholeFile, rec.foldWalk)
		return
	}
	// The core count, said out loud whenever it differs. A reader told "8
	// cores against 4" and nothing else has to go and find out which term
	// that moves, and the honest answer here is "a tenth of it, and only
	// below two" — which is worth as much as a table would be, and is the
	// half that used to be missing. See coresAttribution.
	cores := ""
	if runtime.NumCPU() != rec.cores {
		cores = "\n\n" + coresAttribution
	}
	t.Logf("the wall-clock numbers in this package's comments were taken on %s "+
		"(%s, %s/%s, %d cores), and this run is not on it: %s.\n\n"+
		"The whole package was %s there and the fold walk %s, `go test "+
		"-count=1 ./wasm/verify`. A reading of these tests that is some "+
		"percent off one of their numbers is a difference between two "+
		"computers before it is anything else — which is what this record is "+
		"for, and why none of these numbers is an assertion.%s",
		rec.machine, rec.goVersion, rec.goos, rec.goarch, rec.cores,
		strings.Join(differs, ", "), rec.wholeFile, rec.foldWalk, cores)
}
