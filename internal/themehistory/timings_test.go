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
//	GRMOB_PER_OBJECT_FETCH=required go test -count=1 -v \
//	    -run TestOneProcessPerObject ./internal/…                 perObjectRun
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
	// Everything on top of that IS accounted for, and all of it is one arm:
	// TestTheWholeWalkGoesRoundOneBatchProcess runs the program over the whole
	// history (the 1.49–1.61s in wholeRun below) and enumerates the objects it
	// expects to be fetched first, which is one `ls-tree` per commit and
	// another ~0.86s. So this figure is roughly the old one plus that arm; the
	// paragraph above is about a gap with no such explanation, and the two are
	// worth keeping apart.
	//
	// What that arm costs, since it is the one thing here anybody would want
	// to switch off — and `-short` is the switch:
	//
	//	              default        -short
	//	plain         3.77–3.93s     1.31–1.40s
	//	-race         7.36–7.60s     2.53–2.55s
	//
	// Three runs each for the three that are not this field. The arm is around
	// 2.5s of a plain run and around 4.8s of a -race one, which is the price
	// of the only test here that runs the real program over the real history.
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
	// The shape this program REPLACED, re-created on demand: one
	// `git cat-file -p` per object, over the same objects the batch fetches,
	// with both routes timed in the same run.
	//
	// # Why this field exists at all
	//
	// blob's comment said the per-file shape cost thirty-one seconds, and
	// that was the last number in this package nothing could re-take — the
	// code which cost it was deleted in the session that measured it. A
	// previous session got closer by break-testing blob into retiring its
	// reader per fetch and recording 31.8s in a sentence, which is the same
	// shape one step along: a reading believed because a past session says it
	// took it.
	//
	// TestOneProcessPerObjectIsSlowerThanOneProcessForAllOfThem re-creates the
	// shape instead of describing it, and is off unless
	// GRMOB_PER_OBJECT_FETCH=required — thirty seconds and three thousand
	// processes is not a green-run cost. What it buys is that the RATIO in
	// blob's cost argument is now two measurements taken in one run on
	// whatever machine is asking, rather than a figure from a lost tree.
	//
	// The batched half here is the FETCHES ALONE and is much smaller than
	// wholeRun above, which also pays 88 `ls-tree` processes and every
	// revision's parse. The two are not readings of the same thing and the
	// ratio below is the one that belongs beside blob's argument.
	perObjectRun string
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
	wholePackage: "3.77–3.93s over seven runs",
	wholeRun:     "1.49–1.61s over seven runs, in process, 2906 objects fetched",
	perObjectRun: "30.14–30.42s over three runs, 2906 objects, one process " +
		"each, against 399–401ms for the same fetches batched — 75–76×",
	batchRetire: "0.18–0.30ms over four sets of seven",
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
		rec.wholeRun == "" || rec.perObjectRun == "" || rec.batchRetire == "" {
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
			"there, the command itself %s, one healthy retire %s, and the "+
			"pre-batch shape %s.",
			rec.machine, rec.goVersion, rec.goos, rec.goarch, rec.cores,
			rec.wholePackage, rec.wholeRun, rec.batchRetire, rec.perObjectRun)
		return
	}
	t.Logf("the wall-clock numbers in this package's comments were taken on %s "+
		"(%s, %s/%s, %d cores), and this run is not on it: %s.\n\n"+
		"The package's tests were %s there, the command itself %s, one "+
		"healthy retire %s, and the pre-batch shape %s. A reading some "+
		"percent off one of these is a "+
		"difference between two computers before it is anything else — which "+
		"is what this record is for, and why none of these numbers is an "+
		"assertion.",
		rec.machine, rec.goVersion, rec.goos, rec.goarch, rec.cores,
		strings.Join(differs, ", "), rec.wholePackage, rec.wholeRun,
		rec.batchRetire, rec.perObjectRun)
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

// The history this repository has to have for the reading below to mean
// anything. Below this the walk is over somebody else's checkout — a shallow
// clone, a source tarball, a fresh repository — and the test says so and
// leaves rather than failing on a machine that is not doing anything wrong.
//
// Which is a consequence worth stating rather than leaving in the mechanism:
// the number in themehistoryTimingsTakenOn.wholeRun can only ever be RE-TAKEN
// on a full checkout. On a shallow one this arm asserts nothing and takes no
// clock, and that is the correct behaviour — a walk over eleven commits is not
// the walk the figure is about.
const wholeWalkCommitsFloor = 50

// batchesStartedSince is how many `git cat-file --batch` processes have started
// since it was called.
//
// # Why a closure and not a reset
//
// batchesStarted counts for the lifetime of the test binary and nothing sets
// it back. That is deliberate: what it counts is a reader being RETIRED AND
// REPLACED, so a counter that anything could zero would lose the one event it
// exists to notice, and a reset in one test would silently subtract from any
// delta another test had open.
//
// So the discipline is a delta, and this is the discipline written down once
// instead of in each reader of the counter. There is one reader today; a
// second one wanting a process count takes another of these rather than
// finding a way to zero the global.
func batchesStartedSince() func() int64 {
	before := batchesStarted.Load()
	return func() int64 { return batchesStarted.Load() - before }
}

// The whole walk goes round ONE `git cat-file --batch` and fetches exactly the
// objects the walk itself says it needs. What that costs is recorded rather
// than asserted.
//
// # The claim this is under
//
// blob's comment is the cost argument for this entire program: reading each
// file with its own `git cat-file -p` cost a process per object and around
// thirty seconds for a table of sixteen rows, and `--batch` replaced that with
// one process the whole run talks to. The thirty seconds is re-taken on demand
// by TestOneProcessPerObjectIsSlowerThanOneProcessForAllOfThem, which is the
// same objects fetched both ways. Every number in that paragraph was taken
// once and typed in, and the structural half of it — one process — was
// asserted nowhere. TestTheBatchedFetchReadsWhatCatFileDoes holds the batch to
// reading the same BYTES `-p` does, and
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
//	it served the run     the surviving reader's fetch count is exactly the
//	                       number of objects themeSourcesAt names across the
//	                       history, and its `dead` is nil
//	the table came out    the run's stdout is the table, headline included.
//	                       A walk that read four thousand objects and printed
//	                       nothing has not done what the timing is a timing of
//
// # Why the fetch count is derived and no longer a floor
//
// It was `reader.reads >= 1000`, chosen against a run that fetched 2906. What
// that guarded was real — "one process served eleven objects" would pass the
// process count while saying nothing — but a third of one reading, in a file
// whose whole subject is numbers taken once, is the shape this repository
// keeps writing arms against. If core/ halved, the floor would silently stop
// discriminating; if the walk broke in a way that still fetched 1200, it
// passed.
//
// So the expectation comes out of the repository on every run: one object per
// path themeSourcesAt returns, per commit that touched core/. That turns a
// floor into an EQUALITY, which fails in both directions — a revision the walk
// skipped, and an object it fetched twice — and it scales with whatever
// history the checkout has.
//
// The enumeration is the walk's own function rather than a second copy of the
// suffix and directory rules; see themeSourcesAt for why that matters. It
// costs one `git ls-tree` per commit, which is the price of the number not
// being a constant.
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
// Around two seconds — the walk itself, plus the enumeration above it — which
// roughly doubles this package's test time, and is paid again under -race on
// every `go test ./...` anybody runs, to hold a claim that changes about once
// a year. That is the price of the one arm that runs the actual program over
// the actual history: everything else here is over a scratch repository or a
// hand-built stream, which is right for the shapes they check and is why none
// of them could have caught a walk that quietly started four thousand
// processes.
//
// The lever for anybody who does not want to pay it is `-short`, which skips
// this and nothing else in the package. Not skipped by default, because a
// claim nobody checks on a green run is the state this arm was written to end.
func TestTheWholeWalkGoesRoundOneBatchProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("-short: the whole-walk arm runs the real program over the " +
			"whole history and costs a couple of seconds.\n\n" +
			"Skipped means the process count, the fetch count and the table " +
			"are not asserted on this run, and no wall clock is taken for " +
			"themehistoryTimingsTakenOn.wholeRun.")
	}
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
	shas := strings.Fields(log)
	commits := len(shas)
	if commits < wholeWalkCommitsFloor {
		t.Skipf("only %d commit(s) touch %s/ in this checkout, and this "+
			"reading needs at least %d.\n\n"+
			"A shallow clone, a source tarball or a fresh repository has a "+
			"history this walk cannot be measured over — the counts below "+
			"would be about that checkout rather than about the program.",
			commits, themePkg, wholeWalkCommitsFloor)
	}

	// What the walk will fetch, asked of the walk. One `ls-tree` per commit,
	// and the same enumeration leavesAt does — so the number below is a
	// property of this checkout's history rather than a constant tuned against
	// one run of it.
	enumStart := time.Now()
	wantReads := 0
	for _, sha := range shas {
		direct, _, err := themeSourcesAt(sha)
		if err != nil {
			t.Fatalf("enumerating %s/ at %s: %v", themePkg, sha[:8], err)
		}
		wantReads += len(direct)
	}
	enumTook := time.Since(enumStart)
	// The enumeration reaching the history. Every revision in this repository
	// holds several sources directly in core/, so fewer objects than commits
	// means this loop and not the walk is the thing that has stopped working —
	// and an expectation of nought would make the equality below pass for a
	// walk that fetched nothing at all.
	if wantReads < commits {
		t.Fatalf("the enumeration names %d object(s) across %d commit(s), and "+
			"a revision that touched %s/ holds at least one source in it.\n\n"+
			"The fetch count below is held to this number, so an enumeration "+
			"that has stopped reading the history would make that assertion "+
			"pass over nothing. themeSourcesAt is the walk's own filter; if it "+
			"now declines everything, the table this run prints is empty too.",
			wantReads, commits, themePkg)
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
	// The delta, not a reset: see batchesStartedSince.
	since := batchesStartedSince()

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

	started := since()
	blobsMu.Lock()
	reader := blobs
	blobsMu.Unlock()

	if started != 1 {
		t.Errorf("the walk started %d `git cat-file --batch` process(es) and "+
			"the whole cost argument for this program is that it starts "+
			"ONE.\n\n"+
			"Reading each file with its own `git cat-file -p` cost one process "+
			"per object and around thirty seconds — see "+
			"TestOneProcessPerObjectIsSlowerThanOneProcessForAllOfThem, which "+
			"re-takes that; `--batch` replaced it with "+
			"one process the run talks to for as long as the pipe is open. "+
			"More than one here means a reader was retired and replaced "+
			"mid-run — see blob, which does that on any error carrying "+
			"errDesync. That is correct behaviour and it is a different "+
			"program from the one blob's comment describes, so it is worth "+
			"knowing it happened.\n\n"+
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
	if reader.reads != wantReads {
		// Both directions, because they are different findings. Short: a
		// revision whose sources were not fetched, or a cache between the
		// walk and the reader. Long: an object fetched twice, which is the
		// batch being asked for work the walk does not need.
		by := fmt.Sprintf("%d over", reader.reads-wantReads)
		if reader.reads < wantReads {
			by = fmt.Sprintf("short by %d", wantReads-reader.reads)
		}
		t.Errorf("the surviving batch reader served %d fetch(es) and the walk "+
			"names %d object(s) across %d commit(s) — %s.\n\n"+
			"The claim being measured is that ONE process serves the whole "+
			"walk, and the fetch count is what makes that mean something: a "+
			"single process that answered eleven requests would satisfy the "+
			"count above while saying nothing about it. The expectation is "+
			"themeSourcesAt summed over the history rather than a constant, "+
			"so a difference here is the walk and this enumeration disagreeing "+
			"about which objects the table is built from — not a number that "+
			"has gone stale.\n\n"+
			"Fewer fetches than objects: a revision the walk did not read, or "+
			"a fetch answered from somewhere other than this reader. More: an "+
			"object fetched twice.",
			reader.reads, wantReads, commits, by)
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

	// The by-product. Printed as the numbers together, because the wall clock
	// on its own is the thing this record exists to stop anybody writing down.
	t.Logf("the whole walk: %v over %d commit(s), %d object(s) fetched through "+
		"%d `git cat-file --batch` process(es) — the fetch count is exactly "+
		"what themeSourcesAt names, enumerated here in %v. Recorded as "+
		"themehistoryTimingsTakenOn.wholeRun.\n\n"+
		"Not asserted — see this file's header. The assertions above are the "+
		"process count, the fetch count and the table; the clock is a reading "+
		"of this machine.",
		took.Round(time.Millisecond), commits, reader.reads, started,
		enumTook.Round(time.Millisecond))
}

// The switch that runs the per-object re-creation below, and the one value it
// takes.
//
// Spelled out rather than "any non-empty value is truthy", the way
// GRMOB_COMPOSE_SOURCES is in mobile/verify: a typo should be a failure that
// names itself instead of a setting that silently did nothing.
const (
	perObjectEnv      = "GRMOB_PER_OBJECT_FETCH"
	perObjectRequired = "required"
)

// One process per object, re-created on purpose, against the batch over the
// same objects.
//
// # The number this is here for, and what was wrong with it
//
// blob's comment says the per-file shape cost thirty-one seconds. That figure
// was taken once, in the session that deleted the code which cost it, and it
// has been carried in a comment ever since — the last number in this package
// that nothing could re-take. The previous session came closer: a break-test
// broke blob into retiring its reader on every fetch, measured 31.8s, and
// recorded THAT in a sentence saying a session had done it. Which is the same
// shape one step along: a reading nothing can re-take, believed because a past
// session says it took it.
//
// So the shape is re-created here rather than described. Not by breaking blob
// — a test that mutates the program to measure it is measuring the mutation —
// but by fetching the same objects the way the old code did, with one
// `git cat-file -p` per object, next to the batch doing the same work.
//
// # Why it is off by default
//
// It is thirty seconds and three thousand processes, and what it settles
// changes about once in the life of the program. `GRMOB_PER_OBJECT_FETCH=required`
// runs it:
//
//	GRMOB_PER_OBJECT_FETCH=required go test -count=1 -v \
//	    -run TestOneProcessPerObject ./internal/themehistory
//
// # What is asserted, since a wall clock never is here
//
//	the same bytes      every object comes back byte-identical through both
//	                    routes. TestTheBatchedFetchReadsWhatCatFileDoes makes
//	                    that comparison at HEAD; this one makes it at every
//	                    revision the table is built from, which is the only
//	                    thing in this file that has ever checked the batch
//	                    against `-p` over the whole history
//	one process, many    the batch serves all of them through a single
//	                    process — batchesStarted moves by one while the
//	                    per-object route starts one per object by construction
//	the direction        per-object is slower. Not by how much: the MULTIPLE
//	                    is a reading of this machine's fork cost against its
//	                    pipe cost, and an arm over it would be a wall-clock
//	                    assertion wearing a ratio's clothes
//
// The ratio is what blob's comment now rests on, and it is two numbers taken
// in the same run, over the same objects, on whatever machine is running —
// which is the thing thirty-one seconds in a comment could never be.
func TestOneProcessPerObjectIsSlowerThanOneProcessForAllOfThem(t *testing.T) {
	switch got := os.Getenv(perObjectEnv); got {
	case perObjectRequired:
	case "":
		t.Skipf("%s is not set. This re-creates the pre-batch shape — one "+
			"`git cat-file -p` per object over the whole history, which is "+
			"thousands of processes and around half a minute — so it is off "+
			"unless asked for:\n\n    %s=%s go test -count=1 -v -run "+
			"TestOneProcessPerObject ./internal/themehistory\n\n"+
			"It is what themehistoryTimingsTakenOn.perObjectRun and the ratio "+
			"in blob's comment are re-taken with.",
			perObjectEnv, perObjectEnv, perObjectRequired)
	default:
		t.Fatalf("%s=%q, and the only value it takes is %q.\n\n"+
			"Refused rather than read as \"not required\", because a typo in a "+
			"switch that turns a measurement ON is a measurement nobody "+
			"notices was not taken.", perObjectEnv, got, perObjectRequired)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on this machine, so there is nothing to fetch from.")
	}
	root, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		t.Skipf("this checkout is not a git repository: %v", err)
	}
	t.Chdir(strings.TrimSpace(root))

	log, err := git("log", "--format=%H", "--", themePkg)
	if err != nil {
		t.Skipf("cannot read the history of %s/: %v", themePkg, err)
	}
	shas := strings.Fields(log)
	if len(shas) < wholeWalkCommitsFloor {
		t.Skipf("only %d commit(s) touch %s/ here, and this reading needs at "+
			"least %d — see wholeWalkCommitsFloor.", len(shas), themePkg,
			wholeWalkCommitsFloor)
	}

	// Every object the walk fetches, named the way the walk names them. The
	// pairs are collected first so that both routes below are handed exactly
	// the same work in exactly the same order — the enumeration's own
	// `ls-tree` per commit is paid once and belongs to neither reading.
	type object struct{ rev, path string }
	var objects []object
	for _, sha := range shas {
		direct, _, err := themeSourcesAt(sha)
		if err != nil {
			t.Fatalf("enumerating %s/ at %s: %v", themePkg, sha[:8], err)
		}
		for _, p := range direct {
			objects = append(objects, object{rev: sha, path: p})
		}
	}
	if len(objects) < len(shas) {
		t.Fatalf("the enumeration names %d object(s) across %d commit(s); see "+
			"the same check in TestTheWholeWalkGoesRoundOneBatchProcess.",
			len(objects), len(shas))
	}

	// The batch, first, with a reader of its own. Retired afterwards so the
	// per-object pass below cannot be answered out of it and so nothing this
	// test started outlives it.
	blobsMu.Lock()
	if blobs != nil {
		blobs.retire()
		blobs = nil
	}
	blobsMu.Unlock()
	since := batchesStartedSince()
	batched := make([]string, len(objects))
	batchStart := time.Now()
	for i, o := range objects {
		src, err := blob(o.rev, o.path)
		if err != nil {
			t.Fatalf("batched fetch of %s:%s: %v", o.rev[:8], o.path, err)
		}
		batched[i] = src
	}
	batchTook := time.Since(batchStart)
	started := since()
	blobsMu.Lock()
	if blobs != nil {
		blobs.retire()
		blobs = nil
	}
	blobsMu.Unlock()

	if started != 1 {
		t.Errorf("%d objects went through %d `git cat-file --batch` "+
			"process(es) and the point of the batch is that they go through "+
			"one. The comparison below is then not the comparison this test "+
			"describes.", len(objects), started)
	}

	// And the shape that was here before it: a process per object. This is
	// what leavesAt did before blob existed, spelled the same way — one
	// `git cat-file -p <rev>:<path>` at a time.
	perObject := make([]string, len(objects))
	perStart := time.Now()
	for i, o := range objects {
		src, err := git("cat-file", "-p", o.rev+":"+o.path)
		if err != nil {
			t.Fatalf("per-object fetch of %s:%s: %v", o.rev[:8], o.path, err)
		}
		perObject[i] = src
	}
	perTook := time.Since(perStart)

	// The falsifiable half, and the one that is not about a clock at all: the
	// two routes are the same reading. Reported once with a count rather than
	// per object — three thousand identical failures is not a finding, it is
	// a wall.
	differed := 0
	firstDiff := ""
	for i, o := range objects {
		if batched[i] == perObject[i] {
			continue
		}
		differed++
		if firstDiff == "" {
			firstDiff = fmt.Sprintf("%s:%s — %d byte(s) batched against %d "+
				"through `-p`", o.rev[:8], o.path, len(batched[i]),
				len(perObject[i]))
		}
	}
	if differed > 0 {
		t.Errorf("%d of %d object(s) read differently through the two routes; "+
			"the first is %s.\n\n"+
			"`--batch` and `-p` both write a blob's contents raw, with no "+
			"filters and no line-ending conversion, which is what makes the "+
			"batch a change in how the text is fetched and not in what it "+
			"says. TestTheBatchedFetchReadsWhatCatFileDoes holds that at "+
			"HEAD; this is the same claim at every revision the table is "+
			"built from, which is where a size read one byte short would show "+
			"up as every following file in the run coming back wrong.",
			differed, len(objects), firstDiff)
	}

	// The direction, which is the whole cost argument. Not the multiple — see
	// the header.
	if perTook <= batchTook {
		t.Errorf("one process per object took %v and the batch took %v over "+
			"the same %d object(s), so the shape this program was rewritten "+
			"to avoid is no longer the slower one.\n\n"+
			"blob's comment is a cost argument: fork, exec, opening the "+
			"repository and tearing the process down, once per file, against "+
			"one process and a pipe round trip per file. If that has stopped "+
			"being true on this machine, the argument is what needs "+
			"rewriting, not this test.", perTook, batchTook, len(objects))
	}

	t.Logf("%d object(s) over %d commit(s), fetched both ways:\n"+
		"    one `cat-file --batch`   %v\n"+
		"    one `cat-file -p` each   %v  (%d processes)\n"+
		"    ratio                    %.1f×\n\n"+
		"Recorded as themehistoryTimingsTakenOn.perObjectRun, and it is the "+
		"after-figure's other half in blob's cost argument. Not asserted "+
		"beyond the direction: both numbers are readings of this machine's "+
		"fork cost against its pipe cost.",
		len(objects), len(shas), batchTook.Round(time.Millisecond),
		perTook.Round(time.Millisecond), len(objects),
		float64(perTook)/float64(batchTook))
}
