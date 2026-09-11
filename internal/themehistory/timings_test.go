package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/internal/themeleaves"
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
// this paragraph: wasm/verify/copies_test.go finds every
// `…TimingsTakenOn` in the repository, holds each to the five machine fields
// and to having this reporting test in its package, and FAILS at the third —
// where the trade stops being one package's cost against one duplicate and
// becomes a shape kept in step by whoever remembers to. The reasoning above is
// what that check quotes back when it fires.
//
// # What a re-taking also moves, and the counts that move on their own
//
// Two different problems, and this record had both.
//
// The first is that attribution by reference is a pointer that moves. Every
// figure here says it was taken on the machine this record names, which is the
// discipline the record exists for and is not sufficient: when a field is
// re-taken, every sentence elsewhere that quotes its value silently starts
// claiming to be from a taking it is not from. main.go carried
// `30.14–30.42s against 399–401ms`, attributed to this record in those words,
// while the record said 30.40–30.52s against 398–405ms. main_test.go carried
// the same pair and a wholeRun of `1.49–1.61s` against a recorded 1.52–1.60s.
// Three copies of two numbers, two of them wrong, all three attributed.
//
// The fix is not to re-take the copies. It is for them to stop being copies:
// main.go states the ORDERS OF MAGNITUDE its argument rests on and names the
// field for the readings, and main_test.go names the field instead of
// restating it. A ratio and an order of magnitude do not drift; a reading does.
//
// The second is sharper and belongs to this package in particular. The object
// count and the tree count — 2906 and 88 as they were written — are not
// readings of a machine at all. They are readings of THIS REPOSITORY'S OWN
// HISTORY, and it grows. They were 2955 and
// 89 by the time this paragraph was written, in the same session that found
// them, because the commits that record a count are commits.
//
// So the counts stay in this record, where they say what the timings are a
// reading OVER, and they are gone from the sentences that used to restate
// them — fourteen of those, across both packages that quote this record. Those say `one ls-tree per commit` and `every object in the
// history` now, which is what they were always about — and the arm prints the
// live number on every run, which is the only place a count like this can be
// right.
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
	// history (wholeRun below is what it costs) and enumerates the objects it
	// expects to be fetched first, which is one `ls-tree` per commit. So this
	// figure is roughly the old one plus that arm; the paragraph above is
	// about a gap with no such explanation, and the two are worth keeping
	// apart.
	//
	// That enumeration was 0.90s of it and is now 0.22s: it runs in a bounded
	// pool, which is the one concurrent thing in this package and is explained
	// at enumWorkers. The whole figure moved 3.77–3.93s → 3.01–3.11s for that
	// change and nothing else.
	//
	// # Which makes this figure a reading of a CORE COUNT, and it says so
	//
	// enumWorkers is min(NumCPU, 8), so the arm's largest term scales with the
	// machine in a way nothing else recorded here does. Measured by fixing the
	// worker count and re-running, three runs apiece on the eight cores named
	// above:
	//
	//	workers    the enumeration    the whole package
	//	1          0.90–0.92s         3.69–3.82s
	//	2          0.51s              3.30–3.31s
	//	4          0.29–0.30s         3.07–3.16s
	//	8          0.21–0.23s         3.05–3.14s
	//
	// The one-core row is the figure this package had BEFORE the pool, which
	// is what says the pool is the only thing that moved. So `2.92–3.22s`
	// below is not a number about this code: it is a number about this code on
	// eight cores, and a single-core CI runner pays about 0.7s more for the
	// same green run.
	//
	// # What the arm costs, since it is the one thing here anybody would want
	// # to switch off — and `-short` is the switch
	//
	//	              default        -short
	//	plain         2.92–3.22s     1.18–1.31s
	//	-race         6.46–6.70s     2.39–2.44s
	//
	// Seven runs for the plain default, three for the other three, and every
	// row here holds more than one afternoon's readings. The -race row is six
	// rather than three: it was 6.52–6.64s, and a re-take after the
	// concurrency census grew came in at 6.46–6.58s. Nothing in that arm
	// changed and the two ranges overlap across most of their width, so the
	// row is widened to hold both rather than replaced — which is what a range
	// is for, and the alternative is a record that reports the last afternoon.
	// The two `-short` figures and the plain default were widened the same way
	// by a later re-take, by a hundredth or two at one end apiece.
	//
	// The plain default was widened again, by a tenth this time, and the
	// widening is worth more than the number: fourteen readings came in at
	// 3.06–3.19s, and fourteen more taken by putting the session's changes
	// back and forth — the code as it was, then as it is, alternating — came
	// in at 3.08–3.22s for the OLD code and 3.04–3.19s for the new. So the
	// afternoon is a tenth dearer than the last one and the change is not why;
	// this row is a reading of a machine on a day, which is the whole reason
	// it is a record rather than an assertion.
	//
	// # And then the other three, because half a table is worse than none
	//
	// Widening the plain default and leaving the rest made this a table where
	// one row was an afternoon old and the three under it were not, with
	// nothing on any of them saying which — a reader comparing the -race cost
	// against the plain one would have been comparing two different days and
	// had no way to know. That is the same fault the whole record exists to
	// end, arriving inside it.
	//
	// So all four were re-taken, three runs each, run one after another with
	// nothing else on the machine — which is not how the first re-take of the
	// plain row was taken, and is why that one needed fourteen readings and
	// an A/B to be worth anything. Every row moved by a hundredth or two in
	// the same direction as the plain one, and every row is widened to hold
	// both takings:
	//
	//	plain -short    1.205–1.304s    was 1.20–1.27s
	//	-race           6.549–6.697s    was 6.46–6.64s
	//	-race -short    2.406–2.422s    was 2.41–2.44s
	//
	// What the table says about the ARM is unchanged, which is the thing it is
	// for: about 1.8s of a plain run and about 4.2s of a -race one, the same
	// as before the machine got a tenth slower.
	//
	// # And a third taking, which went back the other way
	//
	// An hour later, with nothing in this package changed but a comment, every
	// row that moved moved DOWN — and by about what the taking before had
	// moved it up:
	//
	//	plain           2.920–3.057s    the taking before: 3.06–3.19s
	//	plain -short    1.181–1.189s    1.205–1.304s
	//	-race           6.466–6.609s    6.549–6.697s
	//	-race -short    2.391–2.436s    2.406–2.422s
	//
	// So the afternoon that read a tenth dear was not a machine that had got
	// slower; it was one end of this machine's own spread across a session,
	// and an hour of it is worth as much as every code change this package has
	// seen. Three takings in, the honest width of the plain row is three
	// tenths on a three-second figure — which is the number a reader holding a
	// disagreeing reading actually needs, and is bigger than any one taking
	// would have told them.
	//
	// # Sixteen readings and not three, because three was chasing
	//
	// The plain row was set from three readings, contradicted by the next run,
	// set again, and contradicted again. That is a floor being chased rather
	// than measured: three readings find a range that the fourth leaves, and
	// the answer is not to take three more. So this row is sixteen —
	// 2.920–3.057s — and the floor comes from the set rather than from
	// whichever reading was last.
	//
	// The other three rows are three readings apiece and are therefore
	// NARROWER THAN THE TRUTH, by about what the plain row gained: somewhere
	// around a twentieth at each end. That is written here rather than fixed,
	// because fixing it is thirty more runs to learn something this row has
	// already said — and a reader who lands outside one of those rows by a
	// few hundredths has been told, here, that the row is not wide enough to
	// judge them.
	//
	// Every row is widened rather than replaced, again, and the whole table is
	// taken together, again, for the reason the section above gives.
	//
	// The arm is around 1.8s of a plain run and around 4.2s of a -race one,
	// which is the price of the only test here that runs the real program over
	// the real history.
	//
	// That saving is the half of the table that moves with the machine, and it
	// moves the OTHER way: a short run skips the enumeration entirely, so what
	// `-short` is worth is the whole 0.90s on one core against 0.22s on eight.
	// The lever's justification is therefore strongest on the machines least
	// likely to have been measured, which is worth knowing before anybody
	// reads the two seconds above as the number to beat.
	//
	// It is also the repository's only `-short` lever, which is a decision
	// rather than a coincidence: wasm/verify/shortlever_test.go holds the set
	// of them to one and says what a short run stops asserting.
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
	// wholeRun above, which also pays one `ls-tree` process per commit and
	// every revision's parse. The two are not readings of the same thing and the
	// ratio below is the one that belongs beside blob's argument.
	//
	// # The other terms, which are here so that nobody has to subtract
	//
	// wholeRun and the batched half of this field were the one number pair in
	// either record that invited a subtraction, and a reader who did it got a
	// figure nobody had measured. So the missing terms are taken here too:
	//
	//	the fetches      409–412ms, the batched half above
	//	the trees        0.87–0.91s, one `ls-tree` per commit run SERIALLY,
	//	                 which is what the walk itself pays. Not the pooled
	//	                 0.22s the whole-walk arm reports, a different quantity
	//	the parse        0.125–0.145s, themeleaves.Of over the same sources at
	//	                 the same revisions — the same call the walk makes, over
	//	                 the bytes the batch just returned
	//	                 ─────
	//	together         1.41–1.47s, against the wholeRun above
	//
	// What is left is the diff between consecutive revisions and the printing,
	// and it is still a REMAINDER rather than a reading: roughly 0.1–0.2s,
	// which is the same size as the disagreement between three separate
	// readings of this machine, so it is not worth a clock of its own until it
	// is bigger than the noise around it. The parse was worth one because it
	// was the largest unmeasured term and because this arm already holds every
	// source in memory — which is what made it measurable here and nowhere
	// else.
	//
	// Declined rather than deferred. See ai_docs/plans/non_goals.md, which
	// carries the argument in full and the condition under which it stops
	// holding — so that the fourth clock is something a reader finds decided
	// rather than missing.
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
	machine:   "Apple M3 (Mac15,13), macOS 26.2",
	goos:      "darwin",
	goarch:    "arm64",
	goVersion: "go1.26.1",
	cores:     8,
	wholePackage: "2.92–3.22s over fifty-seven runs in three sessions, and " +
		"3.69–3.82s on a single core",
	wholeRun: "1.56–1.67s over seven runs, in process, 2955 objects fetched, " +
		"the expectation enumerated alongside in 0.22s over 8 workers",
	perObjectRun: "30.53–30.84s over three runs, 2955 objects, one process " +
		"each, against 409–412ms for the same fetches batched — 74.7–75.1×; " +
		"the 89 `ls-tree` the walk pays serially, 0.87–0.91s; themeleaves.Of " +
		"over the same sources, 0.125–0.145s",
	batchRetire: "0.18–0.29ms over four sets of seven",
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
	// NumCPU and not GOMAXPROCS, the way the other record puts it: what this
	// is about is the machine underneath. It is also the one field here that
	// changes a recorded number by a term a reader can name, so the messages
	// below say which term.
	cores := runtime.NumCPU()
	if cores != rec.cores {
		differs = append(differs, fmt.Sprintf("%d cores against %d", cores,
			rec.cores))
	}
	// Which term the core count moves, said out loud whenever the two differ:
	// a reader told "8 cores against 4" and nothing else has to go and find
	// out what that is worth. The standing half is coresAttribution, which
	// wasm/verify/copies_test.go holds every record to carrying; the
	// worker counts are this run's and belong here.
	pooled := ""
	if cores != rec.cores {
		pooled = fmt.Sprintf("\n\nThe whole-walk arm enumerates its "+
			"expectation over min(NumCPU, 8) workers — %d here against %d "+
			"where the record was taken. %s",
			min(cores, 8), min(rec.cores, 8), coresAttribution)
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
		"assertion.%s",
		rec.machine, rec.goVersion, rec.goos, rec.goarch, rec.cores,
		strings.Join(differs, ", "), rec.wholePackage, rec.wholeRun,
		rec.batchRetire, rec.perObjectRun, pooled)
}

// Which of this package's figures move with the core count, and by how much.
//
// # Why every record carries one of these
//
// `cores` is the one machine field that changes a recorded number by a term a
// reader can name, and the two records in this repository used to say
// different amounts about it. This one carried the table and wasm/verify's
// said nothing — which a reader holding a run that disagrees reads as "nobody
// measured that" rather than as "that is not where the difference is". Both
// now state it, and wasm/verify/copies_test.go holds every record to
// having one, in the same pass that holds them to the five machine fields.
//
// What is worth comparing between the two is the SHAPE. This package's term is
// one git process per commit, so the improvement runs all the way to eight
// workers;
// wasm/verify's is one Go program's own goroutines, and its figure is flat
// from two cores upwards. A reader on a four-core machine should expect a
// different fraction of each.
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
const coresAttribution = "The enumeration is the largest single term in the " +
	"package figure and it is pooled at `min(runtime.NumCPU(), 8)`: 0.22s " +
	"at eight workers, 0.51s at two, 0.90s at one, with the package total " +
	"moving 3.05s to 3.75s across the same range. See `enumWorkers` and the " +
	"table in `wholePackage`'s comment. A difference of that size between this run and " +
	"the number above is accounted for before anything else is — and it runs " +
	"all the way to eight, which is the opposite of wasm/verify's figure, " +
	"where the core-scaled term is flat from two cores upwards."

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

// How many `git ls-tree` processes the enumeration below runs at once.
//
// # Why this is the one concurrent thing in this package
//
// The whole-walk arm holds the batch reader's fetch count to an EQUALITY
// against what themeSourcesAt names across the history, and that expectation
// is one `git ls-tree` per commit — 88 processes over this repository, 0.90s
// serial (0.87–0.92s measured, which is what the 0.83s this line used to quote
// was an estimate of), which was 35% of the arm's wall clock and the largest
// single thing `-short` skips. Almost none of that is git doing anything: it is fork, exec,
// the repository being opened and the process being torn down, which is the
// same cost blob's comment is an argument about, once per commit instead of
// once per object.
//
// The listings are independent of each other and of everything else here —
// themeSourcesAt calls treePaths calls git(), which is an exec.Command with no
// shared state behind it — so they go out in a bounded pool. 0.90s becomes
// 0.22s on the eight cores themehistoryTimingsTakenOn names, which puts the
// expectation at a seventh of the walk it is checking rather than a third.
//
// # Which makes this arm's cost a property of the machine
//
// Worth saying plainly, because it is the only figure in either record that
// does this. The saving is real on eight cores and roughly nothing on one, and
// everything in between is measured rather than extrapolated —
// themehistoryTimingsTakenOn.wholePackage carries the table, 0.90s at one
// worker through 0.22s at eight, with the whole package moving 3.75s to 3.05s
// across the same range.
//
// So the one-core figure is the number this package had BEFORE the pool. A
// reader on a single-core runner has not lost the change; they never had it,
// and the `-short` lever is worth correspondingly more to them.
//
// Bounded rather than one goroutine per commit, because the bound is what
// makes this a fixed number of git processes at a time on any machine and any
// history: a repository with a thousand commits touching core/ would otherwise
// fork a thousand. Capped at eight as well as by the core count, since past
// that the machine is scheduling git processes rather than running them.
//
// # And what this does NOT make parallel
//
// blob's mutex comment says this program is single-threaded, and the
// os.Stdout redirect in the whole-walk arm is safe only because nothing in
// this package is parallel. Both still hold: this pool runs BEFORE the walk,
// touches no package state — not blobs, not batchesStarted, not os.Stdout —
// and is joined before anything is measured. What is concurrent here is a git
// process per commit, which is a fact about the machine rather than about this
// program.
//
// That paragraph used to be the whole of it, which is the state this package
// keeps writing arms against: an argument nobody re-checks, holding up two
// claims made somewhere else. TestTheGoroutinesInThisPackageAreTheOnesDecided-
// On is the arm. It finds every `go` statement here, holds each to a row
// saying what joins it, and follows the package's own call graph out of each
// one to make sure none of them reaches blobs, os.Stdout, an fmt.Print or a
// t.Fatal. A later edit that moved a fetch inside this loop to save a second
// fails there, on every run, rather than on the runs where two workers
// happened to overlap.
var enumWorkers = min(runtime.NumCPU(), 8)

// themeObject is one fetch the walk will make: a path, at a revision.
type themeObject struct{ rev, path string }

// themeSourcesAcross is every object the walk names over these commits, in
// commit order, and what enumerating them cost.
//
// # Why both arms take their expectation from here
//
// This is the walk's own filter — themeSourcesAt, the function leavesAt calls
// — asked across a history instead of at one revision. Neither arm spells the
// `.go` rule or the directory rule again: a test carrying its own copy of them
// would be asserting one copy against another, and both can be wrong together.
// That argument is written down in themeSourcesAt's header, which is where it
// belongs; this is the one place either arm reaches it from.
//
// # Why `workers` is a parameter, when one of the two callers always passes 1
//
// The two arms want different things out of the same enumeration:
//
//	the whole-walk arm    enumWorkers. It pays this on every green run to have
//	                      an expectation rather than a constant, so it wants it
//	                      cheap and does not care what it cost
//	the per-object arm    1. The SERIAL figure is the ls-tree half of what a
//	                      walk spends — the term wholeRun's decomposition needs
//	                      MEASURED rather than subtracted, and a parallel
//	                      reading of it would not be that number
//
// Results are written into a slice indexed by commit and read back in order,
// so eight workers name the same objects in the same order as one. Nothing
// here touches package state; see enumWorkers.
func themeSourcesAcross(t *testing.T, shas []string, workers int) ([]themeObject, time.Duration) {
	t.Helper()
	if workers < 1 {
		workers = 1
	}
	// Indexed by commit rather than appended to, which is what keeps the order
	// independent of how many workers there were and makes the writes
	// disjoint: two goroutines never touch one element.
	per := make([][]string, len(shas))
	errs := make([]error, len(shas))

	start := time.Now()
	work := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range work {
				per[i], _, errs[i] = themeSourcesAt(shas[i])
			}
		}()
	}
	for i := range shas {
		work <- i
	}
	close(work)
	wg.Wait()
	took := time.Since(start)

	// Errors are reported after the join and in commit order, so a failure
	// names the same revision whatever order the workers finished in.
	var objects []themeObject
	for i, sha := range shas {
		if errs[i] != nil {
			t.Fatalf("enumerating %s/ at %s: %v", themePkg, sha[:8], errs[i])
		}
		for _, p := range per[i] {
			objects = append(objects, themeObject{rev: sha, path: p})
		}
	}

	// The enumeration reaching the history. Every revision in this repository
	// holds several sources directly in core/, so fewer objects than commits
	// means this function and not the walk is the thing that has stopped
	// working — and an expectation of nought would make the equality in the
	// whole-walk arm pass for a walk that fetched nothing at all.
	if len(objects) < len(shas) {
		t.Fatalf("the enumeration names %d object(s) across %d commit(s), and "+
			"a revision that touched %s/ holds at least one source in it.\n\n"+
			"Both arms here take their expectation from this function, so an "+
			"enumeration that has stopped reading the history would make those "+
			"assertions pass over nothing. themeSourcesAt is the walk's own "+
			"filter; if it now declines everything, the table a run prints is "+
			"empty too.", len(objects), len(shas), themePkg)
	}
	return objects, took
}

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
// # Whether that price buys anything, which was an open question
//
// The equality costs 88 `git ls-tree` processes to assert a number a floor
// scaled off HEAD would have got within a few of for one process. What settles
// it is whether an OFF-BY-ONE in the fetch path is a failure mode anybody
// expects — and it is, because this program has one written into it:
//
//	a path with a newline   blob sends it to a `cat-file -p` of its own,
//	                        because the batch protocol is line-terminated and
//	                        asking anyway DESYNCHRONISES the stream rather than
//	                        failing. That fetch never reaches batchReader.read,
//	                        so `reads` comes back short by exactly one per such
//	                        path per revision — with one process started, one
//	                        live reader, and a correct table printed
//	a desync mid-run        the reader is retired and replaced, so `reads`
//	                        starts again from nought on the new one. Caught by
//	                        the process count above as well as by this
//
// The first is the case only this equality sees: every other assertion in this
// arm passes through it unchanged, and a floor of a thousand would never
// notice a run that quietly stopped batching a file. Nothing in this
// repository has such a path today, which is the point — the arm is what says
// so on each run, rather than a sentence saying nobody has added one.
//
// So the trade is made, and the price is paid down rather than accepted: the
// enumeration runs in a bounded pool (see enumWorkers), which takes it from
// 0.90s to 0.22s and from 35% of this arm to about a seventh of it.
//
// # And the price it was made at, which is not the price everybody pays
//
// That paragraph settled it against 0.22s, on eight cores, and 0.22s is the
// BEST case of a term that is 0.90s at one worker — the figure this package
// had before the pool existed. On a single-core runner the equality is a third
// of this arm again, and the decision to keep it was never taken at that
// number: it was taken at the one the machine it was written on happened to
// produce.
//
// Taken at 0.90s, deliberately, it is the same decision:
//
//	what it buys      a fetch-path off-by-one, which is a failure this program
//	                  has a mechanism for — the newline path above — and which
//	                  every other assertion in this arm passes through
//	                  unchanged. Not a hypothetical class: a shape that is one
//	                  file rename away, and invisible when it happens
//	what it costs     0.22s to 0.90s, on the one test in this repository that
//	                  runs the real program over the real history, which is
//	                  already the expensive arm by construction
//	the alternative   a floor scaled off HEAD, which is what this replaced. It
//	                  costs one `ls-tree` and cannot see the thing the
//	                  equality is for
//	the lever         `-short`, which skips the whole arm. It is worth the
//	                  whole 0.90s where the pool is worth nothing, which is
//	                  the machine most likely to mind — see enumWorkers
//
// What stops that argument going stale a second time is that the share is no
// longer quoted from a comment: the log line below reports the enumeration as
// a fraction of this arm on the machine that just ran it, so a reader on one
// core is told a third rather than a seventh.
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
// 1.8s on the eight cores themehistoryTimingsTakenOn names — the walk itself,
// plus the enumeration above it — which roughly doubles this package's test
// time, and is paid again under -race on every `go test ./...` anybody runs,
// to hold a claim that changes about once a year.
//
// That number is a reading of a CORE COUNT and not just of a machine: the
// enumeration is pooled at min(NumCPU, 8), so the same arm is about 2.5s where
// there is one core. See enumWorkers, and the table in wholePackage's comment
// — the figure quoted here is the eight-core end of a range four times wide at
// its own term. That is the price of the one arm that runs the actual program over
// the actual history: everything else here is over a scratch repository or a
// hand-built stream, which is right for the shapes they check and is why none
// of them could have caught a walk that quietly started four thousand
// processes.
//
// The lever for anybody who does not want to pay it is `-short`, which skips
// this and nothing else in the repository. Not skipped by default, because a
// claim nobody checks on a green run is the state this arm was written to end.
//
// That lever is the only one here, which is a fact about the repository rather
// than about this file and is held to by an arm of its own:
// wasm/verify/shortlever_test.go finds every testing.Short() there is, holds
// each to skipping rather than shrinking, and holds the SET of them to the
// convention this repository has decided on — one, with a row saying what a
// short run stops asserting. A second lever is where `-short` stops meaning
// one named thing, and that check is what says so.
func TestTheWholeWalkGoesRoundOneBatchProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("-short: the whole-walk arm runs the real program over the " +
			"whole history and costs a couple of seconds.\n\n" +
			"Skipped means the process count, the fetch count and the table " +
			"are not asserted on this run, and no wall clock is taken for " +
			"themehistoryTimingsTakenOn.wholeRun.\n\n" +
			"This is the repository's only `-short` lever, and it is one on " +
			"purpose — see wasm/verify/shortlever_test.go, which is where " +
			"what a short run does and does not assert is written down.")
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
	// one run of it. Run in a pool; see enumWorkers for what that is worth and
	// what it deliberately leaves serial.
	objects, enumTook := themeSourcesAcross(t, shas, enumWorkers)
	wantReads := len(objects)

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
		// revision whose sources were not fetched, a fetch that went round the
		// batch, or a cache between the walk and the reader. Long: an object
		// fetched twice, which is the batch being asked for work the walk does
		// not need.
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
			"so a difference here is the walk and that enumeration disagreeing "+
			"about which objects the table is built from — not a number that "+
			"has gone stale.\n\n"+
			"Short by exactly the number of paths with a NEWLINE in them is "+
			"the one difference this program produces on purpose: blob sends "+
			"such a path to a `cat-file -p` of its own, which never touches "+
			"this counter. See blob, and TestAPathWithANewlineInItGoesRound"+
			"TheBatch. Short by anything else: a revision the walk did not "+
			"read, or a fetch answered from somewhere other than this reader. "+
			"More: an object fetched twice.",
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
	//
	// The enumeration's own figure is printed with its worker count attached,
	// and that is not decoration: the walk pays the SAME `ls-tree`
	// processes serially inside itself, so a pooled reading of them is not the
	// walk's ls-tree cost and must not be subtracted from the total as though
	// it were. The serial figure is taken by the per-object arm and the
	// subtraction is done there — see themehistoryTimingsTakenOn.perObjectRun,
	// which is where the measured terms of this number live.
	// What the equality cost, as a share of this arm on THIS machine. The
	// header argues that price at 0.22s and again at 0.90s, which are the two
	// ends of a term that moves with the core count — and a share quoted from
	// a comment is a share taken on somebody else's computer. This one is
	// taken here, so a reader on one core is told a third and a reader on
	// eight is told a seventh, without either of them having to work out
	// which end of the range they are standing at.
	armTook := took + enumTook
	share := 0.0
	if armTook > 0 {
		share = 100 * float64(enumTook) / float64(armTook)
	}
	t.Logf("the whole walk: %v over %d commit(s), %d object(s) fetched through "+
		"%d `git cat-file --batch` process(es), against the %d object(s) "+
		"themeSourcesAt names, enumerated here in %v over %d worker(s) — "+
		"%.0f%% of this arm's %v. Recorded as "+
		"themehistoryTimingsTakenOn.wholeRun.\n\n"+
		"Not asserted — see this file's header. The assertions above are the "+
		"process count, the fetch count and the table; the clock is a reading "+
		"of this machine. What the walk spent it ON is not read off this "+
		"line: the enumeration above it is pooled and the walk's is not, so "+
		"the split is in perObjectRun rather than in a subtraction here.\n\n"+
		"The share IS worth reading off it. That is what the equality costs "+
		"to have, on the machine in front of you, and the decision to keep it "+
		"was argued at both ends of the range that number moves through — "+
		"about a seventh where the pool has eight workers and about a third "+
		"where it has one. `-short` is the lever, and it is worth most "+
		"exactly where this percentage is largest.",
		took.Round(time.Millisecond), commits, reader.reads, started,
		wantReads, enumTook.Round(time.Millisecond), enumWorkers,
		share, armTook.Round(time.Millisecond))
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

// The least the per-object shape has to cost, as a multiple of the batch, for
// blob's argument to be an argument.
//
// # What was here before, which could not fail
//
// `perTook <= batchTook`: thirty seconds against four hundred milliseconds
// over 2906 objects, on a comparison no machine reverses. That is an assertion
// whose failure is unreachable, so the content was the two numbers in the log
// beside it and the arm was decoration — which is the same shape this package
// keeps writing arms against, one level up.
//
// # Why a multiple is not a wall clock in a ratio's clothes
//
// That was the objection to a floor, and it is the wrong reading of what these
// two numbers are. They are taken in ONE RUN over the SAME objects in the same
// order, so everything about this machine that scales both — a slower disk, a
// busy core, a cold page cache, a `-race` binary — divides out of the
// quotient. What is left is exactly the quantity blob's comment is made of:
// the cost of forking a process against the cost of a pipe round trip. A
// number that survives the machine changing is not a reading of the machine.
//
// The individual figures stay unasserted for the usual reason and are still
// only logged; it is their RATIO that is a claim about the program.
//
// # Where five comes from
//
// 75–76× on the machine themehistoryTimingsTakenOn names. Five is deliberately
// nowhere near that: a machine whose forks are fifteen times cheaper relative
// to its pipes than this one's still passes, which is the room a floor over a
// timing has to leave if it is not to fail on somebody else's computer for
// reasons they cannot act on.
//
// What it does fail is the trade having actually gone — a platform where
// spawning a process costs about what a pipe round trip does — and that is a
// finding about blob's cost argument, which is the only thing this test exists
// to hold up.
const perObjectSlowdownFloor = 5

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
//	the multiple        per-object is at least perObjectSlowdownFloor times
//	                    slower. The direction on its own could not fail; the
//	                    ratio is the cost argument's own quantity and divides
//	                    the machine out of both readings — see that constant
//	the parse reached   themeleaves.Of named at least one leaf across the
//	                    history. Not a clock: it is what says the clock beside
//	                    it is a reading of the parse rather than of a loop
//	                    that parsed nothing
//	the terms           the fetches, the trees, the parse and the per-object
//	                    route are logged together, so wholeRun's split is in
//	                    one place rather than a subtraction across two records
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
	// `ls-tree` per commit is paid once and belongs to neither fetch reading.
	//
	// SERIALLY, which is the one place in this package that asks for that. The
	// whole-walk arm pools it because it only wants the answer; this one wants
	// the COST, because an `ls-tree` per commit one after another is what the
	// walk itself spends on trees, and it is the term that turns wholeRun from
	// a number inviting a subtraction into three measured parts. See
	// themeSourcesAcross, and the log at the end of this test.
	objects, enumTook := themeSourcesAcross(t, shas, 1)

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

	// The third term of wholeRun, which used to be a remainder.
	//
	// # Why this is measurable here and nowhere else
	//
	// The walk's cost is fetches, plus trees, plus what it does with the text.
	// The first two are taken above; the third was left as "wholeRun minus the
	// other two", named honestly as a subtraction nobody had done rather than
	// printed as a reading. What was missing was somewhere to take it: timing
	// themeleaves.Of means having every source at every revision in memory at
	// once, which is 2906 blobs, and this is the one arm that already has
	// them.
	//
	// So it is the SAME call the walk makes — themeleaves.Of over one
	// revision's direct sources, once per revision, in commit order — over the
	// bytes the batch just returned. What is left out of it is the diff
	// between consecutive revisions and the printing, which is what the
	// remainder now is: a much smaller thing, and still named as a remainder.
	//
	// The sources are collected per revision first and timed after, so the map
	// building is not counted as parse. objects is in commit order (see
	// themeSourcesAcross), so a revision's sources are contiguous.
	type revSources struct {
		sha     string
		sources map[string]string
	}
	var perRev []revSources
	for i, o := range objects {
		if i == 0 || o.rev != objects[i-1].rev {
			perRev = append(perRev, revSources{sha: o.rev,
				sources: map[string]string{}})
		}
		perRev[len(perRev)-1].sources[o.path] = batched[i]
	}
	parseStart := time.Now()
	leaves := 0
	for _, r := range perRev {
		exp := themeleaves.Of(r.sources, themeType)
		leaves += len(exp.Names)
	}
	parseTook := time.Since(parseStart)

	// The parse reaching the sources. themeleaves.Of over a revision holding
	// core.Theme names its fields, and every revision in this history holds
	// one — so nought leaves across the whole run is a parse that read
	// nothing, and the clock above would then be a reading of an empty loop
	// rather than of the walk's third term.
	if leaves == 0 {
		t.Errorf("themeleaves.Of named no leaves at all across %d revision(s) "+
			"in %v, and the walk's own table is built out of exactly these "+
			"calls.\n\n"+
			"A run that parses %d source(s) and finds no field of %s has not "+
			"done the work this clock is a clock of, so the figure below is "+
			"not the walk's parse cost — it is the cost of failing to parse.",
			len(perRev), parseTook.Round(time.Millisecond), len(objects),
			themeType)
	}

	// The cost argument, as the multiple it is actually made of. See
	// perObjectSlowdownFloor for why this is a ratio and not the direction it
	// used to be.
	if batchTook <= 0 {
		t.Fatalf("the batched pass over %d object(s) measured %v, which is not "+
			"a duration anything can be divided by. Nothing below is a "+
			"reading if this is one.", len(objects), batchTook)
	}
	ratio := float64(perTook) / float64(batchTook)
	if ratio < perObjectSlowdownFloor {
		t.Errorf("one process per object took %v and the batch took %v over "+
			"the same %d object(s) — %.1f×, against a floor of %d×.\n\n"+
			"blob's comment is a cost argument: fork, exec, opening the "+
			"repository and tearing the process down, once per file, against "+
			"one process and a pipe round trip per file. This is that "+
			"argument's own quantity — both readings are of the same objects "+
			"in the same run, so everything about this machine that scales "+
			"both of them divides out and what is left is its fork cost "+
			"against its pipe cost.\n\n"+
			"The floor is a long way under what this shape has ever measured "+
			"(%d× against 75× where the record was taken), so a failure here "+
			"is not a busy laptop: it is the trade blob describes having "+
			"changed on this machine, and the argument in that comment is "+
			"then what needs rewriting rather than this test.",
			perTook.Round(time.Millisecond), batchTook.Round(time.Millisecond),
			len(objects), ratio, perObjectSlowdownFloor, perObjectSlowdownFloor)
	}

	// The terms, in one place, because the alternative is a reader subtracting
	// across two records. wholeRun is the walk; the fetch, tree and parse
	// lines here are the parts of it this arm can measure, and what is left
	// over is named as a remainder rather than printed as though somebody had
	// timed it.
	t.Logf("%d object(s) over %d commit(s), fetched both ways:\n"+
		"    one `cat-file --batch`   %v\n"+
		"    one `cat-file -p` each   %v  (%d processes)\n"+
		"    ratio                    %.1f×  (floor %d×)\n"+
		"    the trees, serially      %v  (%d `ls-tree` processes)\n"+
		"    themeleaves.Of           %v  (%d revision(s), %d leaf name(s))\n"+
		"    ─────\n"+
		"    the three, together      %v against a wholeRun of %s\n\n"+
		"Recorded as themehistoryTimingsTakenOn.perObjectRun. The first line, "+
		"the trees and the parse are the three terms of wholeRun this arm can "+
		"put a clock on — the fetches, the `ls-tree` per commit the walk pays "+
		"inside itself, and the same themeleaves.Of call over the same "+
		"sources. What is left is the diff between consecutive revisions and "+
		"the printing, and it is still a REMAINDER rather than a reading: "+
		"nothing here has timed it, and the terms above are separate readings "+
		"of this machine rather than one decomposition taken in one run — so "+
		"the three need not, and do not, land exactly under the total in "+
		"either direction. That gap IS the remainder plus the noise between "+
		"readings, and it is smaller than either on its own.\n\n"+
		"Only the ratio is asserted; see perObjectSlowdownFloor.",
		len(objects), len(shas), batchTook.Round(time.Millisecond),
		perTook.Round(time.Millisecond), len(objects), ratio,
		perObjectSlowdownFloor, enumTook.Round(time.Millisecond), len(shas),
		parseTook.Round(time.Millisecond), len(perRev), leaves,
		(batchTook + enumTook + parseTook).Round(time.Millisecond),
		themehistoryTimingsTakenOn.wholeRun)
}
