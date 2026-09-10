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
// spreads over 1.81–1.92s across seven runs on one idle machine, which is 6%
// wide by itself — so a reader holding a 2.0s run against a 1.88s written down
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
//	go test -count=1 ./wasm/verify                          wholeFile
//	go test -count=1 -run TestHowWideTheNarrowerFold ./wasm/verify   foldWalk
//	go test -bench . -run '^$' ./wasm/verify                the per-walk ones
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
	foldWalk string
}{
	machine:   "Apple M3 (Mac15,13), macOS 26.2",
	goos:      "darwin",
	goarch:    "arm64",
	goVersion: "go1.26.1",
	cores:     8,
	wholeFile: "1.81–1.92s over seven runs",
	foldWalk:  "0.40–0.52s over seven runs, node v22.12.0",
}

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
	t.Logf("the wall-clock numbers in this package's comments were taken on %s "+
		"(%s, %s/%s, %d cores), and this run is not on it: %s.\n\n"+
		"The whole package was %s there and the fold walk %s, `go test "+
		"-count=1 ./wasm/verify`. A reading of these tests that is some "+
		"percent off one of their numbers is a difference between two "+
		"computers before it is anything else — which is what this record is "+
		"for, and why none of these numbers is an assertion.",
		rec.machine, rec.goVersion, rec.goos, rec.goarch, rec.cores,
		strings.Join(differs, ", "), rec.wholeFile, rec.foldWalk)
}
