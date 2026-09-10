package main

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

// The machine every wall-clock number in this package's prose was taken on.
//
// # Why this file exists, and what it is a sibling of
//
// wasm/verify/timings_test.go says the general case: a wall clock is a reading
// of a machine, the load on it and the toolchain's scheduler, so a number in a
// comment with no machine attached is a number a reader holding a different
// one cannot reason about. That record covers wasm/verify. This one covers
// this package, which had timings in it and no record at all.
//
// It is a second copy rather than a shared helper because the two are separate
// programs — wasm/verify is `package main` under wasm/, this is `package main`
// under internal/ — and a `internal/timingrecord` imported by both would be a
// package existing so that two structs could be one. The DUPLICATION is the
// cheaper of the two, and what has to stay in step is a shape, not a value:
// the numbers are different numbers about different machines-at-different-
// moments and were never meant to match.
//
// # Re-taking it
//
//	go test -count=1 ./internal/themehistory                      wholePackage
//	go test -count=1 -run TestRetiringAHealthyGit ./internal/…    batchRetire
//	go build -o th ./internal/themehistory && ./th                wholeRun
//
// Anything re-taken here is re-taken WITH this record: a run on another
// machine that updates a timing and leaves the machine alone has put the same
// unattributed number back, one commit later.
var themehistoryTimingsTakenOn = struct {
	// For a reader. Nothing checks this.
	machine string
	// For the arm. runtime answers all four.
	goos, goarch, goVersion string
	cores                   int
	// The whole package's tests, `go test -count=1 ./internal/themehistory`.
	//
	// The previous saved session recorded 1.87s for the same command with
	// three fewer arms in it. Nothing was made faster in between, and the
	// difference is not explained here — it is a 60% gap between two readings
	// of the same code on the same machine, which is what a wall clock with no
	// record beside it looks like from the other side, and the reason this
	// file exists.
	wholePackage string
	// The command itself over this repository's history — the walk the batch
	// reader exists to make affordable, and the number the batch's whole cost
	// argument is measured against.
	//
	// Taken from a COMPILED binary and not `go run`, which folds a build into
	// the reading and is therefore a measurement of the toolchain's cache as
	// much as of this program.
	wholeRun string
	// One healthy retire: close stdin, drain stdout, Wait. See
	// TestRetiringAHealthyGitLeavesBeforeTheDeadline, which takes this, and
	// batchRetireGrace, which is a bound over it.
	//
	// This is the only figure in either record that was written down as an
	// ARGUMENT before it was a measurement. batchRetireGrace's comment said
	// the healthy case was "microseconds to low milliseconds, four orders of
	// magnitude under" five seconds. That was reasoning about a pipe drain and
	// a process exit, and it was never timed.
	batchRetire string
}{
	machine:      "Apple M3 (Mac15,13), macOS 26.2",
	goos:         "darwin",
	goarch:       "arm64",
	goVersion:    "go1.26.1",
	cores:        8,
	wholePackage: "1.15–1.27s over seven runs",
	wholeRun:     "1.52–1.70s over seven runs, built binary",
	batchRetire:  "0.18–0.30ms over four sets of seven",
}

// This run says whether it is standing on the machine the timings came from.
//
// The same reporting arm wasm/verify/timings_test.go carries, and for the same
// reasons: asserting a machine would fail on every computer that is not this
// one, and what is worth having is the SENTENCE in front of the person holding
// a number in a comment against a number on their screen.
//
// The one thing here that IS an assertion is that the record is filled in at
// all. A zeroed field is a record somebody added a timing to without saying
// where it came from.
func TestTheTimingsInThisPackageSayWhichMachineTheyCameFrom(t *testing.T) {
	rec := themehistoryTimingsTakenOn
	if rec.machine == "" || rec.goos == "" || rec.goarch == "" ||
		rec.goVersion == "" || rec.cores == 0 || rec.wholePackage == "" ||
		rec.wholeRun == "" || rec.batchRetire == "" {
		t.Fatalf("themehistoryTimingsTakenOn has an empty field (%+v).\n\n"+
			"Every wall-clock number in this package's prose is attributed to "+
			"this record, and a record with a hole in it attributes them to "+
			"nothing.", rec)
	}

	var differs []string
	if got := runtime.GOOS; got != rec.goos {
		differs = append(differs, fmt.Sprintf("GOOS %s against %s", got, rec.goos))
	}
	if got := runtime.GOARCH; got != rec.goarch {
		differs = append(differs, fmt.Sprintf("GOARCH %s against %s", got,
			rec.goarch))
	}
	// Compared as a prefix: a patch release is a different toolchain and worth
	// naming, and `devel` builds carry a suffix no equality test would match.
	if got := runtime.Version(); !strings.HasPrefix(got, rec.goVersion) {
		differs = append(differs, fmt.Sprintf("%s against %s", got, rec.goVersion))
	}
	if got := runtime.NumCPU(); got != rec.cores {
		differs = append(differs, fmt.Sprintf("%d cores against %d", got,
			rec.cores))
	}

	if len(differs) == 0 {
		t.Logf("this run is on the machine the timings in this package were "+
			"taken on: %s, %s, %s/%s, %d cores. The package's tests were %s "+
			"there, the command itself %s, and one healthy retire %s.",
			rec.machine, rec.goVersion, rec.goos, rec.goarch, rec.cores,
			rec.wholePackage, rec.wholeRun, rec.batchRetire)
		return
	}
	t.Logf("the wall-clock numbers in this package's comments were taken on %s "+
		"(%s, %s/%s, %d cores), and this run is not on it: %s.\n\n"+
		"The package's tests were %s there, the command itself %s, and one "+
		"healthy retire %s. A reading some percent off one of these is a "+
		"difference between two computers before it is anything else — which "+
		"is what this record is for, and why none of these numbers is an "+
		"assertion.",
		rec.machine, rec.goVersion, rec.goos, rec.goarch, rec.cores,
		strings.Join(differs, ", "), rec.wholePackage, rec.wholeRun,
		rec.batchRetire)
}

// How many healthy retires the measurement below takes.
//
// Seven for the reason wasm/verify's ranges are over seven: one reading of a
// wall clock is a reading of whatever else the machine was doing, and the
// spread is the part worth recording. Seven git processes is a few hundred
// milliseconds, which is affordable on a test that runs on every green run.
const healthyRetireSamples = 7

// A healthy git leaves on its own, well before the deadline — measured rather
// than argued.
//
// # What was here before this, which was an argument
//
// batchRetireGrace is five seconds, and its comment justifies the number by
// saying the healthy case is "microseconds to low milliseconds, four orders of
// magnitude under this". That is a sound piece of reasoning about a pipe drain
// from a local process plus a process exit, and it had never been timed. A
// bound whose justification is entirely a priori is a bound nobody can tell has
// stopped being true — if a healthy retire were somehow costing two seconds,
// every number in that comment would still read exactly as it does.
//
// TestRetiringAWedgedProcessDoesNotWaitForever covers the other side: that the
// deadline exists and that the Kill behind it makes the wait finish. It says
// nothing about the case that actually happens on every run.
//
// # What is asserted, and what is only recorded
//
// The wall clock is NOT asserted, for the reason both timings records give: an
// assertion over a wall clock fails on a busy laptop and on every CI runner,
// and a test that fails for reasons nobody can act on is a test people learn
// to ignore.
//
// What IS asserted is the OUTCOME, which is not a timing at all: the process
// exited 0, on its own, having noticed the EOF on its stdin. A retire that hit
// the deadline kills the process, and a killed process does not exit 0 — so
// this arm fires exactly when the claim under batchRetireGrace ("no run that
// is working can reach it") has stopped being true, and never because a
// machine was busy. That is the falsifiable half, and it is the half worth
// having.
//
// The number goes into themehistoryTimingsTakenOn.batchRetire, where it is a
// reading of a machine like every other number in these two files.
func TestRetiringAHealthyGitLeavesBeforeTheDeadline(t *testing.T) {
	var took []time.Duration
	for i := 0; i < healthyRetireSamples; i++ {
		b, err := newBatchReader(".")
		if err != nil {
			t.Skipf("cannot start `git cat-file --batch` here: %v", err)
		}
		// One real request first, so what is being retired is a reader in the
		// state retire is actually called on rather than a process that has
		// never been spoken to. `HEAD:go.mod` is resolved from the repository
		// root whatever directory the batch was started in, which is what
		// makes it a name this test can name from inside internal/.
		if _, err := b.read("HEAD", "go.mod"); err != nil {
			b.retire()
			t.Skipf("`git cat-file --batch` cannot resolve HEAD:go.mod here "+
				"(a shallow or bare checkout, or a fresh repository with no "+
				"commit): %v", err)
		}

		start := time.Now()
		b.retire()
		took = append(took, time.Since(start))

		// The assertion. Kill leaves a signalled process, never a 0 exit — so
		// this is the deadline having fired, stated as a fact about the child
		// rather than as a comparison between two clocks.
		state := b.cmd.ProcessState
		if state == nil {
			t.Fatalf("retire returned and the child has not been waited on. "+
				"Kill signals a process; only Wait reaps it. (sample %d)", i+1)
		}
		if state.ExitCode() != 0 {
			t.Fatalf("a healthy `git cat-file --batch` did not leave on its "+
				"own: it ended as %v after %v, against a %v grace.\n\n"+
				"retire closes stdin, drains stdout and Waits, and git exits "+
				"when it notices the EOF. A non-zero end here means the "+
				"deadline fired and the Kill behind it did the shutdown — "+
				"which is the case batchRetireGrace's comment says no working "+
				"run can reach. Either that has stopped being true, or "+
				"something is holding the drain open.\n\n"+
				"This is not a timing assertion and cannot fail because a "+
				"machine was busy: a slow retire is still a 0 exit right up "+
				"to the deadline.", state, took[i], batchRetireGrace)
		}
	}

	lo, hi := took[0], took[0]
	for _, d := range took {
		if d < lo {
			lo = d
		}
		if d > hi {
			hi = d
		}
	}
	// The ratio, because the ratio is the claim. batchRetireGrace's comment
	// says the healthy case is four orders of magnitude under the grace, and
	// that sentence is the whole justification for the number being five
	// seconds rather than something somebody would have to re-tune. Printed
	// rather than asserted for the usual reason, and printed as a MULTIPLE so
	// that a reader on another machine can check the claim without knowing
	// what this one was doing.
	t.Logf("%d healthy retire(s): %v–%v, against a %v grace — the slowest is "+
		"1/%.0f of it. Recorded as themehistoryTimingsTakenOn.batchRetire; "+
		"see batchRetireGrace for what that ratio is holding up.",
		len(took), lo.Round(time.Microsecond), hi.Round(time.Microsecond),
		batchRetireGrace, float64(batchRetireGrace)/float64(hi))
}
