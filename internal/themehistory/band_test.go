package main

import (
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"
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
func againstBand(fieldName, field string, got time.Duration) string {
	lo, hi, ok := recordedBand(field)
	if !ok {
		return fmt.Sprintf("\n\nNo band was read out of %s. Its value has to "+
			"OPEN with the range, the way every field in both records is "+
			"written — see recordedBandForm. Until it does, the reading "+
			"above is not being compared with anything.", fieldName)
	}
	if differs := recordMachineDiffers(); len(differs) > 0 {
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
	step := (hi - lo) / 100
	if step <= 0 {
		step = time.Nanosecond
	}
	switch {
	case got < lo:
		return fmt.Sprintf("\n\nUNDER the band %s records (%v–%v), by %v. On "+
			"the machine that record names, so it is not another computer. "+
			"Either this got faster and the floor is stale, or the floor was "+
			"set from too few runs — which has happened to both records in "+
			"this repository and is why this line is printed at all. Re-take "+
			"it: widen the range to hold both takings rather than replacing "+
			"it, unless something is known to have changed the code.",
			fieldName, lo, hi, (lo - got).Round(step))
	case got > hi:
		return fmt.Sprintf("\n\nOVER the band %s records (%v–%v), by %v. On "+
			"the machine that record names, so it is not another computer — "+
			"but it may well be a busy one, and one reading over a ceiling "+
			"is not a regression. Take it several times. If it holds, "+
			"something here costs more than it did and the record is the "+
			"place that says so.",
			fieldName, lo, hi, (got - hi).Round(step))
	default:
		return fmt.Sprintf("\n\nIn the band %s records (%v–%v), on the "+
			"machine it names.", fieldName, lo, hi)
	}
}
