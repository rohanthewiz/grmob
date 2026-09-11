//go:build !race

package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
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

// The range a record field opens with.
//
// A second copy. The first is recordedBand in
// internal/themehistory/band_test.go and this is the same function, for the
// same reason the import-resolving helpers are two copies: these are two
// separate `package main` programs and neither can import the other's tests.
//
// The two ARE held identical, by checkTwoCopyDecls — the same census that
// holds the seven import-resolving helpers, and the reason that census is no
// longer named for them. This function is in twoCopyFunctionShapes and the
// pattern above is in twoCopyStateShapes; change one copy and the shared
// repository parse fails, naming both files.
//
// The alternative considered and declined was for the records to carry their
// bands as DURATIONS and render the prose, which would leave nothing to
// parse. It does not remove the copy — two `package main` programs cannot
// import each other's tests, so a renderer is duplicated exactly as a parser
// is — and it costs more than it saves. See ai_docs/plans/non_goals.md.
var recordedBandForm = regexp.MustCompile(
	`^(\d+(?:\.\d+)?)–(\d+(?:\.\d+)?)(µs|ms|s)\b`)

// recordedBand reads that prefix back as two durations. See the other copy.
func recordedBand(field string) (lo, hi time.Duration, ok bool) {
	m := recordedBandForm.FindStringSubmatch(field)
	if m == nil {
		return 0, 0, false
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
	// A range written backwards is a typing error in the record rather than a
	// reading of anything, and it would otherwise make every run "outside the
	// band" with no clue as to why.
	if lo <= 0 || hi <= 0 || hi < lo {
		return 0, 0, false
	}
	return lo, hi, true
}

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
	rec := verifyTimingsTakenOn
	var differs []string
	if got := runtime.GOOS; got != rec.goos {
		differs = append(differs, fmt.Sprintf("GOOS %s against %s", got, rec.goos))
	}
	if got := runtime.GOARCH; got != rec.goarch {
		differs = append(differs, fmt.Sprintf("GOARCH %s against %s", got,
			rec.goarch))
	}
	// A prefix, for the reason the reporting arm gives: a patch release is a
	// different toolchain and `devel` builds carry a suffix.
	if got := runtime.Version(); !strings.HasPrefix(got, rec.goVersion) {
		differs = append(differs, fmt.Sprintf("%s against %s", got, rec.goVersion))
	}
	if got := runtime.NumCPU(); got != rec.cores {
		differs = append(differs, fmt.Sprintf("%d cores against %d", got, rec.cores))
	}
	if len(differs) > 0 {
		return "not the machine the record names — " +
			strings.Join(differs, ", "), false
	}
	return "", true
}

// wholeFileVerdict is the line this package prints about its own wall clock,
// or the empty string when there is nothing it can honestly say.
func wholeFileVerdict(took time.Duration) string {
	if _, ok := wholeFileRunIsTheRecordedOne(); !ok {
		return ""
	}
	lo, hi, ok := recordedBand(verifyTimingsTakenOn.wholeFileInProcess)
	if !ok {
		return "verifyTimingsTakenOn.wholeFileInProcess does not OPEN with a range, " +
			"so this run was not compared with anything. Every field in " +
			"both records is written reading-first; see recordedBandForm."
	}
	// A hundredth of the band's own width, for the reason the other copy's
	// caller gives: a fixed unit printed "by 0s" against a microsecond band.
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
	switch {
	case took < lo:
		return fmt.Sprintf("this package took %v, UNDER the %v–%v that "+
			"verifyTimingsTakenOn.wholeFileInProcess records, by %v. On the "+
			"machine "+
			"that record names. Either this got faster or the floor is "+
			"stale — it has been stale twice. Re-take it, holding both "+
			"takings unless something is known to have changed.",
			took.Round(time.Millisecond), lo, hi, round(lo-took))
	case took > hi:
		return fmt.Sprintf("this package took %v, OVER the %v–%v that "+
			"verifyTimingsTakenOn.wholeFileInProcess records, by %v. On the "+
			"machine "+
			"that record names — which may also be a busy one, so take it "+
			"several times before believing it.",
			took.Round(time.Millisecond), lo, hi, round(took-hi))
	default:
		return fmt.Sprintf("this package took %v, in the %v–%v that "+
			"verifyTimingsTakenOn.wholeFileInProcess records.",
			took.Round(time.Millisecond), lo, hi)
	}
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
