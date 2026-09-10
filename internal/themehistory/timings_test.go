package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
// That argument is about TWO, and it is held to two by an arm rather than by
// this paragraph: wasm/verify/timingsrecords_test.go finds every
// `…TimingsTakenOn` in the repository, holds each to the five machine fields
// and to having this reporting test in its package, and FAILS at the third —
// where the trade stops being one package's cost against one duplicate and
// becomes a shape kept in step by whoever remembers to. The reasoning above is
// what that check quotes back when it fires.
//
// # Re-taking it
//
//	go test -count=1 ./internal/themehistory                      wholePackage
//	go test -count=1 -run TestRetiringAHealthyGit ./internal/…    batchRetire
//	go test -count=1 -v -run TestTheWholeWalk ./internal/…        wholeRun
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
	//
	// It has since MORE than doubled, and that one is accounted for:
	// TestTheWholeWalkGoesRoundOneBatchProcess runs the program over the whole
	// history, which is the 1.49–1.61s in wholeRun below. A reading of this
	// package that is roughly its old figure plus that one is the expected
	// shape; the previous paragraph is about a gap that has no such
	// explanation, and the two are worth keeping apart.
	wholePackage string
	// The command itself over this repository's history — the walk the batch
	// reader exists to make affordable, and the number the batch's whole cost
	// argument is measured against.
	//
	// # How this one is taken, which used to be by hand
	//
	// It was `go build -o th ./internal/themehistory && ./th`, timed with
	// `date +%s%N` around a loop. That is a recipe rather than a command, and
	// it made this the only figure in either record a person could not
	// re-derive by running something — the state the 128ms in
	// wasm/verify/inkglyph_test.go is in, and which that file resolved by
	// establishing the number was holding nothing up. This one IS holding
	// something up: it is the after-figure in blob's cost argument, which is
	// the reason this program is shaped the way it is.
	//
	// So it went where the retire figure went — a test that produces the
	// number as a by-product of asserting something falsifiable.
	// TestTheWholeWalkGoesRoundOneBatchProcess runs the walk and asserts the
	// structural half of that argument: one `git cat-file --batch` process,
	// serving thousands of fetches, with the table printed at the end of it.
	// The clock falls out.
	//
	// Taken IN PROCESS, by calling run() rather than by executing a binary.
	// The difference from the built binary is one exec and one dynamic link —
	// 1.48–1.54s over five runs of `./th` against the range below, which is
	// inside the spread of either — and what it buys is that the number is
	// produced by something in the table above. `go run` is the one spelling
	// that is NOT equivalent: it folds a build into the reading and measures
	// the toolchain's cache as much as this program.
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
	wholePackage: "2.84–3.08s over seven runs",
	wholeRun:     "1.49–1.61s over seven runs, in process, 2906 objects fetched",
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

// The floor the whole walk's fetch count has to clear for the reading below to
// be about this repository's history.
//
// Not a target and not a tuned number: the run this was written against fetched
// several thousand objects, and anything in the low hundreds means the walk did
// not reach the history — a checkout with no commits behind it, a `core/` that
// has just been created, a diff that stopped early. The assertion under it is
// about ONE PROCESS, and "one process served eleven objects" is a sentence that
// would pass while saying nothing.
const wholeWalkReadsFloor = 1000

// And the history this repository has to have for the reading to mean
// anything. Below this the walk is over somebody else's checkout — a shallow
// clone, a source tarball, a fresh repository — and the test says so and
// leaves rather than failing on a machine that is not doing anything wrong.
const wholeWalkCommitsFloor = 50

// The whole walk goes round ONE `git cat-file --batch`, and what that costs is
// recorded rather than asserted.
//
// # The claim this is under
//
// blob's comment is the cost argument for this entire program: reading each
// file with its own `git cat-file -p` cost thirty-one seconds and around
// forty-four hundred processes for a table of sixteen rows, and `--batch`
// replaced that with one process the whole run talks to. Every number in that
// paragraph was taken once and typed in, and the structural half of it — one
// process — was asserted nowhere. TestTheBatchedFetchReadsWhatCatFileDoes
// holds the batch to reading the same BYTES `-p` does, and
// TestAKilledBatchIsReplacedRatherThanReadFrom holds a broken one to being
// replaced; neither says anything about how many processes a real run starts,
// which is the thing the argument is made of.
//
// # What is asserted
//
//	one process started    batchesStarted moves by exactly one across the run.
//	                       Two means a desync sent the reader back for another
//	                       — correct behaviour, and a different performance
//	                       story from the one blob's comment tells
//	it served the run     the surviving reader's fetch count clears
//	                       wholeWalkReadsFloor and its `dead` is nil, so the
//	                       one process is the one that did the work rather
//	                       than one that was started and abandoned
//	the table came out    the run's stdout is the table, headline included.
//	                       A walk that read four thousand objects and printed
//	                       nothing has not done what the timing is a timing of
//
// # And what is only recorded
//
// The wall clock, for the reason both timings records give: an assertion over
// one fails on a busy laptop and on every CI runner. It goes into
// themehistoryTimingsTakenOn.wholeRun, which is why this test exists in this
// file — that field was the only figure in either record with no command
// beside it. It had a RECIPE: build the binary, run it, time it by hand. Every
// other number in the two records is produced by something a person can run,
// and this one now is too, as the by-product of an assertion that is worth
// making on its own.
//
// # What it costs, since that is the objection
//
// Around a second and a half, which roughly doubles this package's test time.
// That is the price of the one arm that runs the actual program over the
// actual history: everything else here is over a scratch repository or a
// hand-built stream, which is right for the shapes they check and is why none
// of them could have caught a walk that quietly started four thousand
// processes.
func TestTheWholeWalkGoesRoundOneBatchProcess(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on this machine, so there is no history to walk.")
	}
	root, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		t.Skipf("this checkout is not a git repository: %v", err)
	}
	// run() resolves `core/` and every revision relative to the working
	// directory, so the walk has to stand where the program stands. t.Chdir
	// puts it back afterwards, which matters here more than usual: every other
	// test in this package fetches blobs, and a leftover cwd would send them
	// at a different repository.
	t.Chdir(strings.TrimSpace(root))

	// The history, before anything is measured over it.
	log, err := git("log", "--format=%H", "--", themePkg)
	if err != nil {
		t.Skipf("cannot read the history of %s/: %v", themePkg, err)
	}
	commits := len(strings.Fields(log))
	if commits < wholeWalkCommitsFloor {
		t.Skipf("only %d commit(s) touch %s/ in this checkout, and this "+
			"reading needs at least %d.\n\n"+
			"A shallow clone, a source tarball or a fresh repository has a "+
			"history this walk cannot be measured over — the counts below "+
			"would be about that checkout rather than about the program.",
			commits, themePkg, wholeWalkCommitsFloor)
	}

	// A reader left running by an earlier test would make the counts below
	// that test's as much as this one's, and it would already be past its
	// first fetch — which is the fetch that starts the process this test is
	// counting.
	blobsMu.Lock()
	if blobs != nil {
		blobs.retire()
		blobs = nil
	}
	blobsMu.Unlock()
	before := batchesStarted.Load()

	// The table goes to a file rather than to the test's output: it is ninety
	// lines of the program working correctly, and what this test has to say
	// about it is that it is there. os.Stdout is a variable and fmt.Printf
	// reads it at call time, so redirecting it is the whole capture — and it
	// is safe here only because nothing in this package is parallel; see the
	// note on blobsMu.
	table := filepath.Join(t.TempDir(), "table.txt")
	f, err := os.Create(table)
	if err != nil {
		t.Fatalf("cannot open a file for the run's stdout: %v", err)
	}
	saved := os.Stdout
	os.Stdout = f
	start := time.Now()
	runErr := run()
	took := time.Since(start)
	os.Stdout = saved
	closeErr := f.Close()

	if runErr != nil {
		t.Fatalf("the walk failed after %v: %v", took, runErr)
	}
	if closeErr != nil {
		t.Fatalf("the run's stdout could not be closed: %v", closeErr)
	}

	started := batchesStarted.Load() - before
	blobsMu.Lock()
	reader := blobs
	blobsMu.Unlock()

	if started != 1 {
		t.Errorf("the walk started %d `git cat-file --batch` process(es) and "+
			"the whole cost argument for this program is that it starts "+
			"ONE.\n\n"+
			"Reading each file with its own `git cat-file -p` cost around "+
			"forty-four hundred processes and thirty-one seconds; `--batch` "+
			"replaced that with one process the run talks to for as long as "+
			"the pipe is open. More than one here means a reader was retired "+
			"and replaced mid-run — see blob, which does that on any error "+
			"carrying errDesync. That is correct behaviour and it is a "+
			"different program from the one blob's comment describes, so it "+
			"is worth knowing it happened.\n\n"+
			"This run took %v over %d commit(s).", started, took, commits)
	}
	if reader == nil {
		t.Fatalf("the walk finished and no batch reader survived it. Every " +
			"revision in the table is parsed out of text this reader " +
			"produced, so a run that ends with none has fetched its files " +
			"some other way.")
	}
	if reader.dead != nil {
		t.Errorf("the surviving batch reader is finished: %v.\n\n"+
			"A reader is only marked dead by an error that leaves the stream "+
			"at an unknown offset, and blob retires and replaces one that is "+
			"— so a dead reader at the END of a run is one that failed on the "+
			"last fetch of it.", reader.dead)
	}
	if reader.reads < wholeWalkReadsFloor {
		t.Errorf("the surviving batch reader served %d fetch(es) over %d "+
			"commit(s), and this reading needs at least %d.\n\n"+
			"The claim being measured is that ONE process serves the whole "+
			"walk. A single process that answered %d requests would satisfy "+
			"the count above while saying nothing about it, so the floor is "+
			"what makes the process count mean something. Either the walk is "+
			"not reaching the history or the fetches are going somewhere "+
			"else.", reader.reads, commits, wholeWalkReadsFloor, reader.reads)
	}

	// The table, which is the run having done the thing the clock is a clock
	// of. Only the headline is matched: the counts in it move with the
	// history, and an arm over them would be this file asserting a fact about
	// the repository's commits.
	printed, err := os.ReadFile(table)
	if err != nil {
		t.Fatalf("the run's stdout could not be read back: %v", err)
	}
	if !strings.Contains(string(printed), "leaves moved") {
		t.Errorf("the run wrote %d byte(s) to stdout and none of it is the "+
			"edit-size table.\n\n"+
			"A walk that fetched %d object(s) in %v and printed no table has "+
			"not done the thing this timing is a timing of.\n\nstdout:\n%s",
			len(printed), reader.reads, took, printed)
	}

	// The by-product. Printed as the three numbers together, because the wall
	// clock on its own is the thing this record exists to stop anybody
	// writing down.
	t.Logf("the whole walk: %v over %d commit(s), %d object(s) fetched through "+
		"%d `git cat-file --batch` process(es). Recorded as "+
		"themehistoryTimingsTakenOn.wholeRun.\n\n"+
		"Not asserted — see this file's header. The assertions above are the "+
		"process count, the fetch floor and the table; the clock is a reading "+
		"of this machine.",
		took.Round(time.Millisecond), commits, reader.reads, started)
}
