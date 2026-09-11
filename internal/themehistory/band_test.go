package main

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Comparing a reading this run took against the band the record carries for
// it, which is the one check in this repository that was being done by eye.
//
// # What went wrong, which is why this exists
//
// `verifyTimingsTakenOn.wholeFile` recorded a floor of 2.88s. The session
// that set it read the package at 2.866s in its own closing figures and
// called that in band, and the session after it read 2.840–2.955s — nine
// runs of twelve under a floor written the day before. Nothing caught it,
// because comparing a reported range against a recorded one is arithmetic a
// person does while reading two numbers on two screens, and it is the only
// step in this repository's verification that has no machine doing it.
//
// This record had the same fault at the same moment and nobody had looked:
// `wholeRun` said 1.56–1.67s and six readings came back 1.538–1.569s.
//
// # Why it prints and cannot assert
//
// The argument is the one this file's header makes and it has not changed: a
// wall clock is a fact about one computer, so asserting one fails on every
// computer that is not it. A band is worse than a bare reading in that
// respect, not better — it is two numbers about one computer instead of one.
//
// What is new is only that the COMPARISON is now made by something that does
// not get it wrong. A reader under `-v` sees the word "under" where they used
// to see two ranges and have to subtract.
//
// The limit is worth stating because it is the same one the reporting arm
// states about itself: a Logf is visible under `-v` and on a run where
// something else failed, and a quiet green run shows none of it. That is
// enough for the occasion this is for, which is a person taking figures at
// the end of a session — they run `-v` to read them — and it is not enough
// to catch a floor going stale between one session and the next. Closing
// that would need the band to be an arm, which it cannot be.
//
// # And why it is not compared off this machine
//
// A reading taken on a different computer is not evidence about the band, so
// the verdict says the machine differs and stops. Otherwise every CI run
// would report a band failure that means nothing, which is the noise that
// argues for deleting the whole record.

// The range a record field opens with.
//
// Every field in both records is written the same way and always has been:
// the reading first, as a range, then what it is a reading OF — "1.56–1.67s
// over seven runs, in process, 2955 objects fetched". So the band is a
// prefix, and reading it back costs one anchored pattern rather than a
// second copy of each number in a form a machine likes better.
//
// That is the property this rests on and it is worth naming, because it is
// also the thing that would break it: a field written "over seven runs,
// 1.56–1.67s" parses as no band at all. It comes back not-ok rather than
// wrong, and the verdict says so — see recordedBand's caller.
//
// The en dash is the separator this repository writes ranges with, in both
// records and in every sentence quoting one. A hyphen is not accepted, for
// the reason the figures rule one package over gives: a form that admits two
// spellings is a form nobody can grep.
var recordedBandForm = regexp.MustCompile(
	`^(\d+(?:\.\d+)?)–(\d+(?:\.\d+)?)(µs|ms|s)\b`)

// recordedBand reads that prefix back as two durations.
//
// Both ends carry the unit of the second one, because that is how the record
// writes them: "0.18–0.29ms" is two readings in milliseconds and not one in
// microseconds. There is no field in either record that mixes units within a
// range, and a range that did would be unreadable to a person before it was
// unreadable here.
func recordedBand(field string) (lo, hi, step time.Duration, ok bool) {
	m := recordedBandForm.FindStringSubmatch(field)
	if m == nil {
		return 0, 0, 0, false
	}
	unit := map[string]time.Duration{
		"µs": time.Microsecond,
		"ms": time.Millisecond,
		"s":  time.Second,
	}[m[3]]
	parse := func(s string) time.Duration {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0
		}
		return time.Duration(f * float64(unit))
	}
	lo, hi = parse(m[1]), parse(m[2])
	// And the PRECISION the ends are written at, which is the third thing a
	// band says and the one nothing was reading.
	//
	// An end is a reading rounded outward to the record's own two decimals —
	// that is the one departure from "an end is a reading" the record allows
	// — so "does this reading reach the floor" has an exact answer at that
	// precision and no answer at all without it. A reading of 2.5512s is the
	// floor of a band written 2.55–2.78s; it is not the floor of one written
	// 2.551–2.780s.
	//
	// Taken from the FINER of the two ends. A band spelled "0.19–0.245s" is a
	// hundredth at one end and a thousandth at the other, and the step has to
	// be fine enough not to call a reading an end it is not.
	digits := 0
	for _, end := range []string{m[1], m[2]} {
		if dot := strings.IndexByte(end, '.'); dot >= 0 {
			if d := len(end) - dot - 1; d > digits {
				digits = d
			}
		}
	}
	step = unit
	for i := 0; i < digits && step > 1; i++ {
		step /= 10
	}
	// A range written backwards is a typing error in the record rather than a
	// reading of anything, and it would otherwise make every run "outside the
	// band" with no clue as to why.
	if lo <= 0 || hi <= 0 || hi < lo {
		return 0, 0, 0, false
	}
	return lo, hi, step, true
}

// bandPlacement is where in a band a reading fell, and which of the band's two
// ends — if either — the reading is evidence FOR.
//
// # What this is for, which is an end nobody took
//
// A band's ends are readings, not choices. That rule is written out at
// wasm/verify/timings_test.go and it cost five re-takings to arrive at, and it
// has a consequence nothing was acting on: an end that no reading has reached
// is an end standing on whatever the session that wrote it had in front of it,
// and there is no way to tell one of those from an end twenty runs have landed
// on. Both are two decimals in a struct literal.
//
// What can be said cheaply, on every run that asks for a verdict, is whether
// THIS reading reaches an end. That is the evidence accumulating in the only
// place it honestly can — beside the reading, in the line a person reads when
// they take figures at the end of a session — rather than in a mark somebody
// has to remember to keep.
//
// # Why it does not suggest moving anything
//
// Because the rule says not to, and the rule is the expensive half of this
// record's history. A reading in the middle of a band says nothing about
// either end; a band nothing has reached the ends of is not a band to narrow,
// and walkParse was narrowed on exactly that evidence and falsified five runs
// later. So the sentence names what the reading is evidence for and stops,
// and the one instruction it carries is the one that was got wrong.
//
// # The width, which is here because prose kept copying it
//
// Three sentences in these two records stated a band's width as a number —
// `3% wide`, `300ms wide`, `130ms wide against 130ms` — and all three were
// stale, two of them describing bands that had since been widened and one
// making a comparison that had since reversed. Every one of them was
// derivable from two numbers in the same file. So the width is printed here,
// beside the reading, and the prose says what it is FOR instead of what it
// was.
func bandPlacement(lo, hi, got, step time.Duration) string {
	width := hi - lo
	reaches := func(end time.Duration) bool {
		if step <= 0 {
			return got == end
		}
		return got.Round(step) == end
	}
	if width <= 0 {
		return fmt.Sprintf("a band with one value in it, %v wide", width)
	}
	switch {
	case reaches(lo):
		return fmt.Sprintf("at the floor of a band %v wide, at the precision "+
			"the band is written to (%v) — so this reading is one the floor "+
			"stands on", width, step)
	case reaches(hi):
		return fmt.Sprintf("at the ceiling of a band %v wide, at the "+
			"precision the band is written to (%v) — so this reading is one "+
			"the ceiling stands on", width, step)
	}
	// A ratio of two durations is a count rather than a duration, which is
	// why it is converted before it is printed: %d over a time.Duration
	// prints its nanoseconds.
	return fmt.Sprintf("%d%% up a band %v wide, so it reaches neither end. An "+
		"end no reading has reached is evidence of nothing, and not a reason "+
		"to move one", int(100*(got-lo)/width), width)
}

// recordMachineDiffers is every way this computer is not the one the record
// was taken on, in the words the reporting arm prints.
//
// Extracted from that arm rather than copied, because a second copy of the
// machine comparison is exactly the shape this package's censuses exist to
// find. Both callers want the same four answers: the arm prints them, and
// the band verdict uses only whether the list is empty.
func recordMachineDiffers() []string {
	rec := themehistoryTimingsTakenOn
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
	// is about is the machine underneath.
	if cores := runtime.NumCPU(); cores != rec.cores {
		differs = append(differs, fmt.Sprintf("%d cores against %d", cores,
			rec.cores))
	}
	return differs
}

// againstBand is the sentence that says where a reading this run took fell
// in the range the record carries for it.
//
// Returned rather than logged, so that it lands at the end of the line that
// already reports the reading. A reader comparing a number to a band wants
// both in one sentence; a second Logf a screen away is the arrangement this
// is replacing.
//
// The name is the field's, spelled as a reader would go and edit it, because
// what the out-of-band case asks for is an edit to that field.

// The verdict is asked for, and the reason is `go test ./...`.
//
// `go test` runs package binaries in PARALLEL. The bands here were taken
// with one package running alone, which is the command their method lines
// name, and a package sharing an eight-core laptop with fifteen other test
// binaries is not that command. Measured: this package reads 2.5-2.7s alone
// and 2.98-3.10s inside `go test ./...`, and internal/themehistory's
// wholeRun reads 1.4-1.6s alone and 1.70s there. Both report OVER.
//
// That path runs in every session's verification. A verdict that is wrong
// every time somebody runs the whole tree is not a weaker verdict — it is
// the noise this file's own header says argues for deleting the record, and
// it would teach a reader to skip the line in the one case it is right.
//
// There is no way for a test binary to know what else `go test` is running.
// So it is asked for instead, spelled out the way GRMOB_PER_OBJECT_FETCH is
// and for the same reason: a typo should be a setting that names itself
// rather than one that silently did nothing. The reporting arm, which
// prints on every `-v` run, says how to ask.
const (
	bandVerdictEnv   = "GRMOB_BAND_VERDICT"
	bandVerdictAsked = "required"
)

// bandVerdictWanted is whether this run asked for a band verdict. See above.
func bandVerdictWanted() bool {
	return os.Getenv(bandVerdictEnv) == bandVerdictAsked
}

func againstBand(fieldName, field string, got time.Duration) string {
	if !bandVerdictWanted() {
		return ""
	}
	return againstBandGiven(fieldName, field, recordMachineDiffers(), got)
}

// againstBandGiven is that sentence, as a function of nothing but its
// arguments.
//
// # Why the gate and the machine check are the caller's
//
// Two callers, and they are asking different questions. The three arms in this
// package take a reading of THIS run, so the gate is "was a verdict asked for"
// and the machine is this computer. The arm that places the figure `go test`
// itself printed takes a reading from OUTSIDE the process: its gate is the
// presence of that figure, and the machine that matters is the one the figure
// came from, which is this one only because the person ran both commands in a
// row.
//
// Pushing both out leaves a function of four arguments that returns a
// sentence, which is the form that can be ASSERTED — see
// TestWhereAReadingFellInItsBandIsReadOffTheBandsOwnPrecision, which is the
// only kind of test anything in this file can carry. The gate and the machine
// read the environment and the runtime, and a test over either is a test of
// the computer it runs on.
//
// # And why it is a two-copy shape
//
// wasm/verify places a figure the same way and neither package can import the
// other's tests. Two packages spelling "UNDER the band, by 14ms" differently
// is two records a reader cannot hold against each other, which is the whole
// point of their being a pair — so this is in twoCopyFunctionShapes beside
// recordedBand and bandPlacement.
func againstBandGiven(fieldName, field string, differs []string,
	got time.Duration) string {

	lo, hi, step, ok := recordedBand(field)
	if !ok {
		return fmt.Sprintf("\n\nNo band was read out of %s. Its value has to "+
			"OPEN with the range, the way every field in both records is "+
			"written — see recordedBandForm. Until it does, the reading "+
			"above is not being compared with anything.", fieldName)
	}
	if len(differs) > 0 {
		return fmt.Sprintf("\n\nNot compared against %s (%v–%v): this run is "+
			"not on the machine that record was taken on — %s. A reading "+
			"from a different computer is not evidence about the band.",
			fieldName, lo, hi, strings.Join(differs, ", "))
	}
	// Rounded against the BAND and not against a fixed unit. batchRetire's
	// range is 180µs–290µs, and a difference rounded to the millisecond
	// printed there as "by 0s" — a verdict that says the reading is outside
	// a range and then says by nothing, which is worse than not printing it.
	// A hundredth of the band's own width is fine enough to be true at every
	// scale either record holds and coarse enough not to print seven digits.
	// Rounded against the BAND and not against a fixed unit: batchRetire's
	// range is 180µs–290µs, and a difference rounded to the millisecond
	// printed there as "by 0s".
	//
	// That was the first half of the fix and it was not enough. A reading
	// one step under the floor still rounds to zero, and "outside the band
	// by nothing" is the same useless sentence arrived at from the other
	// side. So a difference that rounds away is reported at microsecond
	// resolution instead — whatever it is, it is not nothing, because the
	// arm that prints it only runs when the reading is outside.
	round := func(d time.Duration) time.Duration {
		step := (hi - lo) / 100
		if step <= 0 {
			step = time.Microsecond
		}
		if r := d.Round(step); r != 0 {
			return r
		}
		return d.Round(time.Microsecond)
	}
	// Compared at the precision the band is WRITTEN to, and not at the clock's.
	//
	// # The false UNDER this fixes, which this machinery produced twice in one
	// # hour
	//
	// An end is a reading rounded outward to the record's two decimals — that
	// is the one departure from "an end is a reading" the record allows, and it
	// means a floor of `2.76s` stands for readings down to 2.755s. A raw
	// comparison calls 2.755s UNDER by 5ms, which sends a reader to re-take a
	// band that is right. Both records did exactly that, within an hour of the
	// arm that places these figures existing: wasm/verify read 2.755s against a
	// 2.76s floor and internal/themehistory 2.816s against 2.82s, and both
	// reported UNDER.
	//
	// bandPlacement already judged "does this reading reach an end" this way.
	// The in-band decision did not, so the two halves of one sentence
	// disagreed: a reading could be reported outside a band and, by the
	// placement rule, be AT its floor.
	//
	// The distance printed is still the true one, lo-got rather than a rounded
	// difference. What the rounding decides is WHICH branch, not what to say
	// once the branch is chosen.
	switch rounded := got.Round(step); {
	case rounded < lo:
		return fmt.Sprintf("\n\nUNDER the band %s records (%v–%v), by %v. On "+
			"the machine that record names, so it is not another computer. "+
			"Either this got faster and the floor is stale, or the floor was "+
			"set from too few runs — which has happened to both records in "+
			"this repository and is why this line is printed at all. Re-take "+
			"it: widen the range to hold both takings rather than replacing "+
			"it, unless something is known to have changed the code.",
			fieldName, lo, hi, round(lo-got))
	case rounded > hi:
		return fmt.Sprintf("\n\nOVER the band %s records (%v–%v), by %v. On "+
			"the machine that record names, so it is not another computer — "+
			"but it may well be a busy one, and one reading over a ceiling "+
			"is not a regression. Take it several times. If it holds, "+
			"something here costs more than it did and the record is the "+
			"place that says so.",
			fieldName, lo, hi, round(got-hi))
	default:
		// The placement is the half of this line that is about the BAND
		// rather than about the reading: a reading inside a band is a pass,
		// and which part of the band it is in is the only thing it tells
		// anybody about the two ends. See bandPlacement.
		return fmt.Sprintf("\n\nIn the band %s records (%v–%v), on the "+
			"machine it names — %s.", fieldName, lo, hi,
			bandPlacement(lo, hi, got, step))
	}
}

// Where a reading fell in its band, asserted — which nothing else about a band
// in this repository can be.
//
// # Why this one IS a test when the rest of the file is not
//
// Everything else here is a reading of a machine and cannot be asserted: that
// argument is at the top of this file and it has not changed. bandPlacement is
// not. It takes four durations and returns a sentence, and given the four the
// answer is the same on every computer — so the one piece of this machinery
// that can be held to being right is held to it.
//
// What that covers is the half that was actually got wrong twice in this
// record's history by hand: which end a reading is evidence for. A person
// comparing 2.5512s against a floor of 2.55s decides it "is" the floor, and a
// person comparing it against 2.551s decides it is not, and both of those are
// arithmetic about a written precision rather than judgements. The rest of the
// line — the reading itself — is a wall clock and stays a report.
//
// # And it covers both copies
//
// bandPlacement is in twoCopyFunctionShapes, so wasm/verify's copy is held to
// being this same declaration by the shared repository parse. A second test in
// that package would assert the same arithmetic about a function a census
// already says is identical; this is the cheaper arrangement and it is the one
// the other two-copy shapes use.
func TestWhereAReadingFellInItsBandIsReadOffTheBandsOwnPrecision(t *testing.T) {
	const (
		ms  = time.Millisecond
		sec = time.Second
	)
	// Named rather than ranged over inline so that the count in the log line
	// below is the list's own length. A hand-written "7 placements" is a copy
	// of a fact about the list, which is the defect class this repository
	// spends most of its censuses on.
	cases := []struct {
		why             string
		lo, hi, got     time.Duration
		step            time.Duration
		wants, wantsNot []string
	}{
		{
			why: "a reading exactly at the floor",
			lo:  2550 * ms, hi: 2780 * ms, got: 2550 * ms, step: 10 * ms,
			wants: []string{"at the floor", "230ms wide", "the floor stands on"},
		},
		{
			// The case the precision exists for: 1.2ms above a floor written
			// to the hundredth is the floor, because the hundredth is what
			// the record claims to know.
			why: "a reading inside half a step of the floor",
			lo:  2550 * ms, hi: 2780 * ms, got: 2551200 * time.Microsecond,
			step:  10 * ms,
			wants: []string{"at the floor"},
		},
		{
			// The same reading against the same numbers written one decimal
			// finer. Nothing about the machine changed; what changed is what
			// the record says it knows.
			why: "the same reading against a band written to the thousandth",
			lo:  2550 * ms, hi: 2780 * ms, got: 2551200 * time.Microsecond,
			step:     ms,
			wants:    []string{"reaches neither end"},
			wantsNot: []string{"at the floor"},
		},
		{
			why: "a reading exactly at the ceiling",
			lo:  2550 * ms, hi: 2780 * ms, got: 2780 * ms, step: 10 * ms,
			wants: []string{"at the ceiling", "the ceiling stands on"},
		},
		{
			why: "a reading a quarter of the way up",
			lo:  2000 * ms, hi: 2400 * ms, got: 2100 * ms, step: 10 * ms,
			wants: []string{"25% up a band 400ms wide", "reaches neither end",
				"not a reason to move one"},
		},
		{
			// A microsecond band, which is batchRetire's shape: the point is
			// that the same arithmetic reads at a scale three orders down,
			// because the step comes off the record rather than from a unit
			// chosen here.
			why: "a microsecond band, at its floor",
			lo:  180 * time.Microsecond, hi: 290 * time.Microsecond,
			got: 184 * time.Microsecond, step: 10 * time.Microsecond,
			wants: []string{"at the floor", "110µs wide"},
		},
		{
			// Not reachable from a record today — recordedBand rejects hi<lo
			// and both records write two ends — and cheap to be right about
			// rather than to divide by.
			why: "a band with no width",
			lo:  sec, hi: sec, got: sec, step: 10 * ms,
			wants:    []string{"one value in it"},
			wantsNot: []string{"%"},
		},
	}
	for _, c := range cases {
		got := bandPlacement(c.lo, c.hi, c.got, c.step)
		for _, want := range c.wants {
			if !strings.Contains(got, want) {
				t.Errorf("%s: the placement of %v in %v–%v at a step of %v "+
					"does not say %q.\n\nIt says: %s\n\n"+
					"This sentence is what a reader taking figures is told "+
					"about the band's ENDS, which are the half of a band "+
					"nothing else in this repository checks — an end is a "+
					"reading, and a reading that reaches one is the only "+
					"evidence there is that it is.",
					c.why, c.got, c.lo, c.hi, c.step, want, got)
			}
		}
		for _, not := range c.wantsNot {
			if strings.Contains(got, not) {
				t.Errorf("%s: the placement of %v in %v–%v at a step of %v "+
					"says %q and should not.\n\nIt says: %s",
					c.why, c.got, c.lo, c.hi, c.step, not, got)
			}
		}
	}
	t.Logf("%d placement(s) asserted, including both ends at two precisions "+
		"and a microsecond band. bandPlacement is in twoCopyFunctionShapes, "+
		"so wasm/verify's copy is held to being this same declaration.",
		len(cases))
}

// The figure `go test` printed, for the arm that places it in its band.
//
// Spelled out and the same name in both packages, the way GRMOB_BAND_VERDICT
// is: a person who has the recipe for one record has it for the other.
const packageReadingEnv = "GRMOB_PACKAGE_READING"

// The figure `go test` prints, handed back to the code that knows the band.
//
// # Why this one had no verdict, and why that cost something
//
// themehistoryTimingsTakenOn.wholePackage is what `go test -count=1 ./internal/themehistory` reports, and that number is
// produced by the `go` command rather than by the test binary. No test can see
// it. It is also the number a person is most likely to be holding this record
// up against, because it is the one they get by running the tests — and it has
// drifted twice, both times found by somebody running the command a dozen
// times and comparing two ranges by eye.
//
// That comparison was made by eye once more in the run that added this, and it
// nearly went wrong in the direction the rule exists for: the package read
// 2.778s against a floor of 2.78s, which LOOKS like a reading under the floor
// and is the floor — an end is a reading rounded outward to the record's two
// decimals, so at the precision the band is written to those are the same
// number. A person without that rule in front of them re-takes a floor that
// was reached, which is the false verdict every paragraph in this record
// warns about.
//
// # Why the reading comes from the environment and not from a nested run
//
// The obvious alternative is an opt-in arm that runs `go test` itself and
// reads the `ok` line. Measured before it was written: three nested runs of
// wasm/verify read 2.751–2.902s where nine plain runs read 2.778–2.891s, and
// one of the three was under the recorded floor. A figure taken by a `go test`
// running underneath another process is a figure from a different invocation,
// and "the method is part of the reading" is the thing this record repeats
// most often. A nested run would have added a new measurement in order to
// check an old one.
//
// So the reading is the one `go test` already printed, for the run the person
// actually made, and the only new thing is the arithmetic — which is
// againstBandGiven, the same sentence the other record's arms print.
//
// # What it cannot check
//
// That the figure came from a run of this package ALONE. `go test ./...` runs
// package binaries in parallel and reports a figure a band's width above this
// one; nothing here can tell one from the other, which is the same limit the
// verdict lever has and is stated for the same reason. The command below is
// the one to take it from.
func TestTheFigureGoTestPrintedForThisPackageIsPlacedInItsBand(t *testing.T) {
	raw := os.Getenv(packageReadingEnv)
	if raw == "" {
		t.Skipf("%s is not set. This places the figure `go test` prints for "+
			"this package — the one reading in themehistoryTimingsTakenOn that no test can see — "+
			"in the band the record carries for it:\n\n"+
			"    go test -count=1 ./internal/themehistory\n"+
			"    %s=<that figure> go test -count=1 -v -run "+
			"TestTheFigureGoTestPrinted ./internal/themehistory\n\n"+
			"Two commands because the first one's own wall clock is what is "+
			"being placed, and only the `go` command can see it.",
			packageReadingEnv, packageReadingEnv)
	}
	// A duration, spelled the way `go test` spells it: `2.891s`. Parsed rather
	// than scanned for digits, so that a figure copied with its unit attached
	// is the form that works and a bare number is refused rather than read as
	// nanoseconds.
	got, err := time.ParseDuration(raw)
	if err != nil || got <= 0 {
		t.Fatalf("%s=%q is not a duration this can read (%v).\n\n"+
			"It takes the figure `go test` prints, with its unit: `2.891s`. "+
			"Refused rather than guessed at, for the reason the verdict "+
			"lever is spelled out rather than defaulted — a value nothing "+
			"understood should say so instead of placing a reading nobody "+
			"took.", packageReadingEnv, raw, err)
	}
	// TrimLeft because againstBandGiven's sentence is built to land at the end
	// of a line that already reports a reading, and here it IS the line.
	t.Log(strings.TrimLeft(againstBandGiven("themehistoryTimingsTakenOn.wholePackage",
		themehistoryTimingsTakenOn.wholePackage, recordMachineDiffers(), got), "\n"))
}

// The sentence built around a placement, asserted — the four ways it can come
// out and the one input that silences it.
//
// # Why this is separate from the placement test
//
// They answer different questions. bandPlacement is arithmetic about a band;
// againstBandGiven is the DECISION about what to say, which includes two cases
// that are not about the band at all — a field whose value cannot be read as a
// band, and a reading from a different computer. Those two are the ones worth
// holding: both return early, both are silent about the placement, and a
// regression in either would read as a passing verdict rather than as a
// failure.
//
// The machine list is a parameter here, which is the reason this can be a test
// at all: recordMachineDiffers reads the runtime, and a test over it is a test
// of the computer it runs on. See againstBandGiven's header.
func TestTheSentenceAroundAPlacementSaysWhichOfTheFourCasesItIs(t *testing.T) {
	const field = "1.40–1.67s over twenty-three runs"
	cases := []struct {
		why             string
		field           string
		differs         []string
		got             time.Duration
		wants, wantsNot []string
	}{
		{
			why:   "a reading inside the band",
			field: field, got: 1500 * time.Millisecond,
			wants: []string{"In the band", "1.4s–1.67s", "37% up a band 270ms wide"},
		},
		{
			// Half a step under the floor is AT the floor: the band is written
			// to the hundredth, so 1.3951s and 1.40s are the same number at
			// the precision the record claims. This case is the false UNDER
			// the rounding fixed, and it fired on both records within an hour
			// of the arm that places these figures existing.
			why:   "a reading within half a step of the floor",
			field: field, got: 1395100 * time.Microsecond,
			wants:    []string{"In the band", "at the floor"},
			wantsNot: []string{"UNDER"},
		},
		{
			why:   "a reading under the floor",
			field: field, got: 1300 * time.Millisecond,
			// 99.9ms and not 100ms: the difference is rounded to a
			// hundredth of the band's own width, which is 2.7ms here. That
			// rule is there so a microsecond band does not print "by 0s", and
			// the cost of it is a tenth of a millisecond of honesty on a
			// figure like this one.
			wants:    []string{"UNDER the band", "by 99.9ms", "widen the range"},
			wantsNot: []string{"up a band"},
		},
		{
			why:   "a reading over the ceiling",
			field: field, got: 1800 * time.Millisecond,
			wants: []string{"OVER the band", "by 129.6ms",
				"Take it several times"},
			wantsNot: []string{"up a band"},
		},
		{
			// The field written the other way round. It comes back as a
			// finding about the FIELD rather than about the reading, which is
			// the distinction that matters: nothing is wrong with the run.
			why:   "a field whose value does not open with a band",
			field: "over twenty-three runs, 1.40–1.67s", got: 1500 * time.Millisecond,
			wants:    []string{"No band was read out of", "has to OPEN"},
			wantsNot: []string{"In the band", "UNDER", "OVER"},
		},
		{
			// A reading from another computer is not evidence about the band,
			// so the sentence says which machine and stops. This is the case
			// that would otherwise report a band failure on every CI runner —
			// the noise that argues for deleting the record.
			why:   "a reading from a different machine",
			field: field, differs: []string{"4 cores against 8"},
			got:      1300 * time.Millisecond,
			wants:    []string{"Not compared against", "4 cores against 8"},
			wantsNot: []string{"UNDER the band", "up a band"},
		},
	}
	for _, c := range cases {
		got := againstBandGiven("theField", c.field, c.differs, c.got)
		for _, want := range c.wants {
			if !strings.Contains(got, want) {
				t.Errorf("%s: the sentence for %v against %q does not say "+
					"%q.\n\nIt says: %s\n\n"+
					"This is the line a person reads when they take figures, "+
					"and the two records print the same one — see "+
					"twoCopyFunctionShapes. A case that stops saying which "+
					"of the four it is reads as a verdict that passed.",
					c.why, c.got, c.field, want, got)
			}
		}
		for _, not := range c.wantsNot {
			if strings.Contains(got, not) {
				t.Errorf("%s: the sentence for %v against %q says %q and "+
					"should not.\n\nIt says: %s",
					c.why, c.got, c.field, not, got)
			}
		}
	}
	t.Logf("%d sentence(s) asserted: in band, half a step under the floor "+
		"(which is in band, at the floor), under, over, a field that is not a "+
		"band, and a reading from another machine.", len(cases))
}
