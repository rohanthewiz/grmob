//go:build !race

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// The one figure in this repository that nothing was measuring, measured.
//
// # Why this package was the one that drifted
//
// `verifyTimingsTakenOn.wholeFile` is what `go test -count=1 ./wasm/verify`
// costs. Two sessions running, it was found to be recording a floor its own
// taking was under — 2.88s written while the session writing it read 2.866s,
// and 2.840s the evening after. Both times a person found it by running the
// command a dozen times and comparing two numbers by eye.
//
// internal/themehistory does not have that problem any more, because three of
// its arms take a reading in process and now print where the reading fell —
// see againstBand there. This package could not do the same for its headline
// figure for one reason: **nothing inside the package measures it.** The
// number is what `go test` itself reports, and no test can see it.
//
// TestMain can. It is the only thing in a test binary that runs on both
// sides of every test in the package, so a clock around `m.Run()` measures
// the tests themselves.
//
// # What it measures is NOT wholeFile, and that was checked rather than
// # assumed
//
// The first version of this file compared the reading against
// `verifyTimingsTakenOn.wholeFile` and reported UNDER by 170ms on a run
// `go test` called 2.958s. The two are different quantities: `go test`'s
// figure includes the build check, the process starting and package
// initialisation, and a clock inside the process sees none of it.
//
// So the record grew a second field. `wholeFileInProcess` is what this
// measures and is what this is compared against — the same pair, and for
// the same reason, as `wholePackage` and `wholeRun` in
// internal/themehistory.
//
// # And it is NOT more visible than the other verdicts, which was the
// # other thing this file claimed before it was run
//
// The claim was that writing to stderr from TestMain would put the verdict
// in front of a plain `go test`, where a `t.Logf` only reaches `-v`. It
// does not: `go test` prints a passing package's output only under `-v`,
// exactly like a Logf, and the line below was seen on a plain run once —
// on a run where the package was FAILING, which is the other occasion
// band_test.go already names.
//
// What this earns is therefore smaller than it was written to be, and still
// worth having: it is the only in-process reading this package has, where
// before there was none. The visibility limit is the one stated in
// band_test.go and is unchanged by the spelling.
//
// # The four runs this must not speak about
//
// The band is one number about one command, and four ordinary invocations
// are not that command:
//
//	-race        a different figure entirely, roughly twice this one, and
//	             recorded separately. Handled by the build tag at the top of
//	             this file rather than by a flag: under -race this file is
//	             not compiled, there is no TestMain, and nothing is printed
//	-short       skips work the recorded figure includes
//	-run X       a filtered run is a fraction of the package
//	-count=N     N runs reported as one
//	-bench       adds time the figure is not about
//
// Everything but the first is read off the flag package after `m.Run()` has
// returned, which is after `flag.Parse`, so the values are the ones in
// effect. A guard that got this wrong would not fail — it would print a
// verdict about a number that is not the recorded one, which is the sort of
// wrong this whole file exists to stop.
//
// And off the machine the record names, it says nothing at all. The
// reporting arm already says the machine differs; a band verdict from
// another computer is noise that argues for deleting the record.

// The band reader and its pattern are NOT here, and were until this package's
// record census grew a question about them.
//
// This file is `//go:build !race` — see the four runs above — and a
// declaration in it does not exist under `-race`. That is right for the
// TestMain clock and wrong for the band READER, which is ordinary parsing with
// no clock in it: `checkEveryBandFieldHasATakingCommand` asks whether a record
// field's value opens with a band, on every run including the race one, and a
// reader behind this tag made that census a build failure in exactly the suite
// that is most likely to be run last.
//
// So recordedBand and recordedBandForm are in timings_test.go, beside
// bandVerdictEnv, which was moved out of this file one iteration earlier for
// the same reason. The tag bounds what must not RUN outside one command; it is
// not where a package keeps the things that read its record.

// wholeFileRunIsTheRecordedOne says whether this invocation is the command
// the band is a reading of, and names what it is instead when it is not.
//
// Returned as a reason rather than a bool so that the caller can stay silent
// and a reader debugging why nothing printed can find the list in one place.
func wholeFileRunIsTheRecordedOne() (why string, ok bool) {
	// Every one of these is read off the flag package, `-short` included.
	// testing.Short() would be the obvious spelling and it is the wrong one:
	// this repository holds `-short` to being ONE lever — see
	// shortlever_test.go, which caught this — and a read outside a test
	// function is a second lever on something else. Nothing here skips
	// anything. What this asks is which command ran, and the flag is that
	// question; the lever is a different question with the same name.
	//
	// Looked up rather than declared, for the ordinary reason: these are the
	// testing package's own flags and re-declaring one would be a second
	// definition of a flag that already exists.
	if f := flag.Lookup("test.short"); f != nil && f.Value.String() == "true" {
		return "-short", false
	}
	if f := flag.Lookup("test.run"); f != nil && f.Value.String() != "" {
		return "a filtered run (-run " + f.Value.String() + ")", false
	}
	if f := flag.Lookup("test.bench"); f != nil && f.Value.String() != "" {
		return "a benchmark run", false
	}
	// The default is "1" and the recorded figure is one run. Anything else is
	// N runs reported as one number.
	if f := flag.Lookup("test.count"); f != nil && f.Value.String() != "1" {
		return "-count=" + f.Value.String(), false
	}
	if differs := recordMachineDiffers(); len(differs) > 0 {
		return "not the machine the record names — " +
			strings.Join(differs, ", "), false
	}
	return "", true
}

// wholeFileVerdict is the line this package prints about its own wall clock,
// or the empty string when there is nothing it can honestly say.

func wholeFileVerdict(took time.Duration) string {
	if !bandVerdictWanted() {
		return ""
	}
	if _, ok := wholeFileRunIsTheRecordedOne(); !ok {
		return ""
	}
	// The whole of the comparison is againstBandGiven, which is the sentence
	// the other record's three arms print and is held identical across the two
	// packages by checkTwoCopyDecls.
	//
	// # Why this stopped spelling it out, which is a defect it had
	//
	// It used to decide UNDER, OVER and in-band itself, with its own wording
	// and its own rounding, because the sentence it wanted read "this package
	// took X, UNDER the Y–Z that …records" rather than as a fragment appended
	// to a line. That is a third implementation of one decision, and two of
	// the three were held in step by a census while this one was not.
	//
	// It drifted, exactly there. The session that found both records
	// reporting a false UNDER — a reading of 2.755s against a floor of 2.76s,
	// which at the precision the band is written to IS the floor — fixed the
	// two copies the census holds and left this one comparing raw clocks. The
	// same false verdict was still reachable here two iterations later, on the
	// line that prints on every asked run of this package.
	//
	// So the decision is made in one place and this supplies the half that is
	// local: which reading, and the sentence it belongs to. The machine
	// comparison is asked for again rather than assumed empty — the guard
	// above has already returned if it is not, and a verdict that quietly
	// depends on an earlier return is one nobody can move.
	return fmt.Sprintf("this package took %v.%s", took.Round(time.Millisecond),
		againstBandGiven("verifyTimingsTakenOn.wholeFileInProcess",
			verifyTimingsTakenOn.wholeFileInProcess, recordMachineDiffers(),
			took))
}

// TestMain times the package and reports the reading against the record.
//
// The clock is the only thing added. `m.Run()`'s result is returned
// unchanged through os.Exit, because a TestMain that swallows a failure is a
// green run that should have been red — and this one exists to make a
// number visible, which is not worth any risk to what the package reports.
func TestMain(m *testing.M) {
	started := time.Now()
	code := m.Run()
	if line := wholeFileVerdict(time.Since(started)); line != "" {
		// stderr, and after m.Run: `go test` shows it on a plain green run,
		// which is the whole difference between this and a Logf.
		fmt.Fprintln(os.Stderr, "\n"+line)
	}
	os.Exit(code)
}
