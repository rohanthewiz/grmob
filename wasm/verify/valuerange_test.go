package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/internal/valuefixture"
)

// valuerange.mjs is internal/valuefixture as browser.mjs mounts it, and this is
// what keeps it from being a transcription.
//
// # Why the browser needs a copy at all
//
// The same reason palette.mjs exists: browser.mjs mounts JSON trees through the
// real runtime in a real Chrome, and it cannot call into Go while it does. The
// twenty-one cases and Go's answer for each have to arrive as data.
//
// # What the browser adds that the JVM pass cannot
//
// internal/valuefixture had one consumer, android/verify, which runs
// GrMobProgress.kt against core.ValueRange.Progress on a JVM. That settles
// Compose and says nothing about the web, and the web is not a target that
// merely *transliterates* this rule — it is the target that does not implement
// it at all. Both DOM exporters hand aria-valuenow/-min/-max to the browser
// verbatim on the argument that a browser applies ARIA's rules itself, and
// until this pass nothing in the repository had ever watched one do it:
//
//	the implicit 0..100        a bare aria-valuenow announcing as a percentage
//	                           is the premise components.ProgressBar rests on
//	one bound stated           "3" with only a max of 5 is step 3 of 5, and the
//	                           other end is ARIA's, not zero
//	indeterminate by omission  a progressbar with no aria-valuenow is a bar
//	                           that is running, not a bar at the start
//	clamping                   a live counter that overshoots must not announce
//	                           a number outside its own range
//
// Every one of those is a claim about somebody else's software, which is
// exactly the kind this repository otherwise cannot make.
//
// # And it found a divergence
//
// The cases where a stated number is not a number are the ones the three
// implementations disagree about, and the disagreement is not subtle: Chrome
// reads aria-valuenow="half" as 0 and pins the bar at the start of its range,
// where core.Progress and Compose both read it as absent and announce an
// indeterminate bar. A bound that does not parse is worse — Chrome reads
// aria-valuemax="lots" as 0, which inverts the range and then clamps the
// position down into it, so a bar at 45% announces as complete.
//
// So the table carries `parses`, derived from core.ValueRange.Unparsed, and
// browser.mjs uses it in both directions: a case whose numbers all parse must
// agree with Go, and a case with an unparseable field must *not*. The second
// half is a pinned divergence rather than a wish — if a browser ever starts
// applying ARIA's defaults there it will fail, which is the moment somebody
// should hear about it. core.AuditTree reports the same shape in Go, as
// ConcernUnusableValueRange.

// One row of the table, in both languages.
type valueRow struct {
	name                string
	now, min, max, text string  // the wire strings, as core.ValueRange holds them
	reading             string  // core.Progress's reading
	rnow, rmin, rmax    float64 // the numbers it resolved them to
	parses              bool    // core.ValueRange.Unparsed found nothing
}

// String renders a row the way valuerange.mjs spells it, so a drifted table can
// be pasted back rather than retyped — which is how a transcription gets made
// in the first place.
func (r valueRow) String() string {
	return fmt.Sprintf(
		"{\n        name: %q,\n"+
			"        wire: { Now: %q, Min: %q, Max: %q, Text: %q },\n"+
			"        reading: %q, now: %s, min: %s, max: %s, parses: %t,\n    }",
		r.name, r.now, r.min, r.max, r.text,
		r.reading, num(r.rnow), num(r.rmin), num(r.rmax), r.parses)
}

// num formats a resolved number the way JavaScript's own literal does: no
// exponent for anything in this table, and no trailing zeros.
func num(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

var (
	mjsValueRow = regexp.MustCompile(
		`\{\s*name:\s*"([^"]*)",\s*wire:\s*\{([^}]*)\},\s*` +
			`reading:\s*"([^"]*)",\s*now:\s*(-?[0-9.]+),\s*min:\s*(-?[0-9.]+),\s*` +
			`max:\s*(-?[0-9.]+),\s*parses:\s*(true|false),?\s*\}`)
	mjsWireField = regexp.MustCompile(`(\w+):\s*"([^"]*)"`)
)

// wantValueRows is the table as internal/valuefixture and core state it, in the
// fixture's own order — which is the order browser.mjs mounts the bars in and
// the order a reader of the .mjs file sees.
func wantValueRows() []valueRow {
	cases := valuefixture.Cases()
	out := make([]valueRow, 0, len(cases))
	for _, c := range cases {
		p := valuefixture.Want(c)
		out = append(out, valueRow{
			name: c.Name,
			now:  c.Range.Now, min: c.Range.Min, max: c.Range.Max, text: c.Range.Text,
			reading: string(p.Reading),
			rnow:    p.Now, rmin: p.Min, rmax: p.Max,
			parses: len(c.Range.Unparsed()) == 0,
		})
	}
	return out
}

func valueRangeMJS(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(".", "valuerange.mjs"))
	if err != nil {
		t.Fatalf("reading valuerange.mjs: %v", err)
	}
	return string(src)
}

// The browser's copy of the fixture is Go's.
//
// Both directions fail, and they fail differently — the same two failures
// palette_test.go names. A missing row is a case the browser pass has never
// mounted, which means one of ARIA's rules is settled on the JVM and nowhere
// else; a surplus row is a bar being checked against an answer no Go authority
// produced, which passes whatever it measures.
func TestTheBrowsersValueRangeTableIsGos(t *testing.T) {
	src := valueRangeMJS(t)

	var got []valueRow
	for _, m := range mjsValueRow.FindAllStringSubmatch(src, -1) {
		row := valueRow{name: m[1], reading: m[3], parses: m[7] == "true"}
		for _, f := range mjsWireField.FindAllStringSubmatch(m[2], -1) {
			switch f[1] {
			case "Now":
				row.now = f[2]
			case "Min":
				row.min = f[2]
			case "Max":
				row.max = f[2]
			case "Text":
				row.text = f[2]
			default:
				t.Errorf("valuerange.mjs: row %q states a wire field %q, which is not "+
					"one of core.ValueRange's four", row.name, f[1])
			}
		}
		for i, dst := range []*float64{&row.rnow, &row.rmin, &row.rmax} {
			f, err := strconv.ParseFloat(m[4+i], 64)
			if err != nil {
				t.Errorf("valuerange.mjs: row %q has an unreadable number %q: %v",
					row.name, m[4+i], err)
			}
			*dst = f
		}
		got = append(got, row)
	}
	want := wantValueRows()

	if len(got) != len(want) {
		t.Errorf("valuerange.mjs has %d rows, internal/valuefixture has %d",
			len(got), len(want))
	}
	for i := 0; i < len(got) && i < len(want); i++ {
		if got[i] != want[i] {
			t.Errorf("valuerange.mjs row %d is\n    %s\nGo says\n    %s",
				i, got[i], want[i])
		}
	}
	for _, row := range want[min(len(got), len(want)):] {
		t.Errorf("valuerange.mjs has no row for %q — browser.mjs mounts every row in "+
			"that file and nothing else, so this case has only ever been checked on a "+
			"JVM", row.name)
	}
	for _, row := range got[min(len(got), len(want)):] {
		t.Errorf("valuerange.mjs carries a row internal/valuefixture does not have:\n"+
			"    %s\nThe browser is holding a bar to an answer no Go authority "+
			"produced", row)
	}

	if t.Failed() {
		var b strings.Builder
		for _, row := range want {
			b.WriteString("    " + row.String() + ",\n")
		}
		t.Logf("valuerange.mjs's VALUE_RANGES should be:\n%s", b.String())
	}
}

// The table has to reach both sides of the divergence, or browser.mjs is only
// asserting one of the two things it says it asserts.
//
// This is internal/valuefixture's own TestTheTableReachesEveryReading one level
// out: that test makes sure the JVM comparison can see a Kotlin branch that got
// an arm wrong, and this one makes sure the browser comparison can see a
// browser that changed its mind. A table of only well-formed ranges would agree
// with a check that had deleted its disagreement half.
func TestTheValueTableReachesBothSidesOfTheBrowserDivergence(t *testing.T) {
	var parses, doesNot int
	for _, row := range wantValueRows() {
		if row.parses {
			parses++
		} else {
			doesNot++
		}
	}
	if parses == 0 {
		t.Error("no case in internal/valuefixture has three usable numbers — the " +
			"browser pass would assert nothing about ARIA's defaults, its clamping or " +
			"its indeterminate spelling")
	}
	if doesNot == 0 {
		t.Error("no case in internal/valuefixture states a number that is not one — " +
			"the browser pass would never exercise the half that pins Chrome's " +
			"disagreement with core.Progress, and a browser that started agreeing " +
			"would go unnoticed")
	}
}

// Every reading core has is one the browser pass knows how to check.
//
// axAgreesWithGo decides what to assert about a bar by switching on the row's
// reading, and a reading with no arm is a bar nothing is asserted about — a
// silent pass rather than an error, since the row still mounts and still gets
// looked up. A fifth member of core.ProgressReading would arrive exactly that
// way: internal/valuefixture would grow a case for it (its own
// TestTheTableReachesEveryReading requires one), the pin above would carry it
// into the table, and the browser would walk straight past it.
//
// The switch lives in valuerange.mjs beside the table rather than in
// browser.mjs, because half of the verdict it feeds cannot fail on a machine
// with a shipping browser and had to be reachable from a unit test — see the
// note there and valuerange_test.mjs. This reads the file it moved to; a
// substring search of browser.mjs would now find the readings nowhere and
// report four failures, or worse, find them in a comment and report none.
//
// Held in Go because Go is where the vocabulary is. A source check because the
// alternative is teaching a Node harness to enumerate a Go type.
func TestEveryReadingIsOneTheBrowserChecks(t *testing.T) {
	src := valueRangeMJS(t)
	for _, reading := range []core.ProgressReading{
		core.ProgressUnstated, core.ProgressIndeterminate,
		core.ProgressDeterminate, core.ProgressEmptyRange,
	} {
		if !strings.Contains(src, fmt.Sprintf("%q", string(reading))) {
			t.Errorf("valuerange.mjs's axAgreesWithGo has no arm for the %q reading — "+
				"a row carrying it mounts, is found, and has nothing asserted about it",
				reading)
		}
	}
}
