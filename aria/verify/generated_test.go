package verify

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/aria/spec"
)

// The fixture is what the specification says.
//
// Every other test in this package holds *code* to the fixture. This one holds
// the fixture to ARIA, and it is the only check here whose subject is the data
// rather than a guard — which makes it the answer to the question doc.go used
// to end on: the fixture was hand-transcribed, and a transcription can be wrong
// in exactly the way the prose it replaced could be wrong.
//
// It was wrong, in four places, and none of them could fail a test:
//
//	list.requiredOwned held `group`      a listbox's allowance, not a list's
//	radiogroup.orientation held a value  ARIA 1.1 had one, 1.2 removed it
//	treegrid.orientation held a value    no version of ARIA states one
//	nameProhibited held `term`, `time`   both take an author name in 1.2
//
// All four sat in near-miss rows, which exist so a guard has something to argue
// with and are therefore the rows least likely to be argued with. That is the
// shape of the failure this test closes: not a wrong fact nobody noticed, but a
// wrong fact nothing *could* notice.
//
// # Why it skips rather than fetching
//
// The specification is a 1.4MB download and this repository's verification
// promise is that none of it needs the network — `go test ./...`, run.sh, and
// the three platform harnesses all work from the committed fixture. So the
// download is a developer step (aria/fetch.sh), the fixture is committed, and
// this test runs only when a copy happens to be on disk.
//
// That is the same stance ios/verify takes toward a missing iPhoneOS SDK and
// android/verify toward a missing Kotlin compiler: a check that needs a thing
// the machine may not have says so and passes, rather than either failing
// honest checkouts or quietly not existing.
//
// The cost is real and worth naming: on a machine with no spec copy this is a
// SKIP, so a fixture edited by hand into disagreement with ARIA would reach a
// commit. What stops it going further is that the same edit fails here the
// moment anyone runs the fetch — and that the file now says GENERATED at the
// top, with the command to regenerate it on the next line.
//
// # And a third way to be wrong, which is the download rather than the fixture
//
// The download is not committed and W3C keeps every revision at its own URL
// forever, so what is on a machine is whatever it fetched, whenever it
// fetched. A 1.1 copy is not a broken file — it is a perfectly good
// specification that this repository does not read — and it regenerates a
// fixture that differs from the committed one on four facts. Left to the
// comparison below, that reads as a fixture error. So the edition is checked
// first and reported as what it is.
//
// That decision is localCopyGate, which is a function rather than four lines
// here because two of its three answers had never executed on any machine —
// see its own comment.
func TestTheFixtureIsWhatTheSpecificationSays(t *testing.T) {
	root := repoRoot(t)

	raw, err := os.ReadFile(filepath.Join(root, spec.LocalPath))
	if skip := localCopyGate(raw, err); skip != "" {
		t.Skip(skip)
	}

	doc, err := spec.Parse(string(raw))
	if err != nil {
		t.Fatalf("parsing the specification: %v", err)
	}
	fx, err := spec.Scope(doc)
	if err != nil {
		t.Fatalf("scoping the specification: %v", err)
	}

	want := fx.Render()
	got, err := os.ReadFile(filepath.Join(root, spec.FixturePath))
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	if bytes.Equal(got, want) {
		return
	}

	// Byte comparison, reported by line. The fixture is generated, so any
	// difference at all is a difference — there is no formatting drift to
	// tolerate — and a line-level report is what says *which* fact moved.
	t.Errorf("%s is not what aria/gen would write. Regenerate it:\n"+
		"    go run ./aria/gen\n\nFirst difference:\n%s",
		spec.FixturePath, firstDifference(got, want))
}

// localCopyGate decides what the comparison above does with whatever is on
// disk: run, or skip with a reason. Empty means run.
//
// # Why this is a function and not four lines inline
//
// It was four lines inline, and two of its three branches had never executed.
// The absent-copy branch runs on any machine without a download; the
// wrong-edition branch runs on a machine holding a 1.1 copy, which is to say
// nowhere, ever. Its *subject* was covered — aria/spec's
// TestAStaleDownloadIsRefusedBeforeItIsParsed proves spec.Parse refuses a 1.1
// document and names the download in the refusal — but that is a different
// claim from this one. Parse *fails*; this guard exists precisely so the same
// document produces a SKIP instead, because a checkout is not broken because
// the copy beside it is old. A guard whose whole content is "turn that failure
// into a skip" and which has never run is a guard nothing has checked.
//
// Extracting it is what makes the three outcomes reachable from a test: the
// inputs are bytes and an error, so all three can be handed over directly
// rather than arranged on somebody's filesystem.
//
// # The three outcomes
//
//	no copy on disk      skip. The specification is a 1.4MB download and this
//	                     repository's promise is that verification never needs
//	                     the network, so the fetch is a developer step.
//	a copy of some other  skip, naming both editions. ARIA 1.1 is the same
//	edition              ReSpec output with different cells: it parses cleanly
//	                     to ~94 roles and regenerates a fixture differing on
//	                     exactly the four facts 1.2 changed, so the comparison
//	                     below would name radiogroup's orientation as "the
//	                     first difference" — a true statement about 1.1,
//	                     printed as if somebody had mistyped a fixture nobody
//	                     touched.
//	a copy that says      skip, and say that. An error page, a truncated
//	nothing              download or a proxy's interstitial has no title
//	                     heading, and "is WAI-ARIA \"\"" is not a sentence to
//	                     hand somebody. This branch is the one exercising the
//	                     guard turned up: it was folded in with the edition
//	                     mismatch and reported as one.
func localCopyGate(raw []byte, readErr error) string {
	if readErr != nil {
		return fmt.Sprintf("no local copy of the ARIA specification (%s): run "+
			"`sh aria/fetch.sh` to check the fixture against it", spec.LocalPath)
	}
	switch v := spec.SpecVersion(string(raw)); v {
	case spec.Version:
		return ""
	case "":
		return fmt.Sprintf("the local copy of the specification (%s) does not say "+
			"which edition it is — a truncated download or an error page reads "+
			"this way: re-run `sh aria/fetch.sh`", spec.LocalPath)
	default:
		return fmt.Sprintf("the local copy of the specification is WAI-ARIA %q and "+
			"the fixture is generated from %s: re-run `sh aria/fetch.sh` to check it",
			v, spec.Version)
	}
}

// The title heading the published document opens with, as spec.SpecVersion
// reads it, for an edition of this test's choosing.
//
// Written out here rather than imported because aria/spec keeps its sample
// unexported, and a second copy of a two-line literal is cheaper than an
// export that exists for one caller. What keeps the copy honest is the first
// assertion in the test below: the heading built for the *current* edition
// must be one SpecVersion actually reads. If the regexp moves, that fails and
// says so, rather than every case here quietly falling into the "says nothing"
// branch and agreeing.
func titleHeading(version string) string {
	return `<h1 id="title" class="title">Accessible Rich Internet ` +
		`Applications (WAI-ARIA) ` + version + `</h1>`
}

// All three outcomes, which is two more than the machine running this has ever
// produced.
func TestTheLocalCopyGateAnswersForEveryKindOfCopy(t *testing.T) {
	current := titleHeading(spec.Version)
	if got := spec.SpecVersion(current); got != spec.Version {
		t.Fatalf("the heading this test builds reads as %q, not %q — spec.SpecVersion "+
			"has moved and every case below would take the \"says nothing\" branch "+
			"and agree with anything", got, spec.Version)
	}

	for _, c := range []struct {
		name string
		raw  string
		err  error
		// want are substrings the skip message must carry; an empty slice
		// means the gate must return "" and let the comparison run.
		want []string
	}{
		{
			name: "no copy on disk",
			err:  os.ErrNotExist,
			want: []string{spec.LocalPath, "aria/fetch.sh"},
		},
		{
			// The case the whole guard exists for, and the one no machine has
			// ever run: a perfectly good specification that this repository
			// does not read.
			name: "a copy of the previous edition",
			raw:  titleHeading("1.1") + "<html><body>roles</body></html>",
			want: []string{"1.1", spec.Version, "aria/fetch.sh"},
		},
		{
			name: "a copy with no title heading",
			raw:  "<html><body>502 Bad Gateway</body></html>",
			want: []string{"does not say which edition", "aria/fetch.sh"},
		},
		{
			name: "the edition the fixture came from",
			raw:  current + "<html><body>roles</body></html>",
		},
	} {
		got := localCopyGate([]byte(c.raw), c.err)
		if len(c.want) == 0 {
			if got != "" {
				t.Errorf("%s: the gate skipped with %q — a current copy is the one "+
					"case that must reach the comparison, and a gate that skips on "+
					"it makes this file's only check unreachable", c.name, got)
			}
			continue
		}
		if got == "" {
			t.Errorf("%s: the gate let the comparison run — it would report the "+
				"fixture as wrong about facts it is right about", c.name)
			continue
		}
		for _, want := range c.want {
			if !strings.Contains(got, want) {
				t.Errorf("%s: the skip does not mention %q, which is what the reader "+
					"needs to act on it: %s", c.name, want, got)
			}
		}
	}
}

// firstDifference reports the first line the two files disagree on, with its
// number. Enough to name the fact that moved without printing two 5KB files.
func firstDifference(got, want []byte) string {
	g := bytes.Split(got, []byte("\n"))
	w := bytes.Split(want, []byte("\n"))
	for i := 0; i < len(g) || i < len(w); i++ {
		var gl, wl []byte
		if i < len(g) {
			gl = g[i]
		}
		if i < len(w) {
			wl = w[i]
		}
		if !bytes.Equal(gl, wl) {
			return "  line " + itoa(i+1) + "\n" +
				"    on disk:   " + string(gl) + "\n" +
				"    generated: " + string(wl)
		}
	}
	return "  (the files differ in length only)"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// repoRoot walks up to the directory holding go.mod.
//
// `go test` runs each package in its own directory, so "../.." would work — and
// would break the moment this package moved. The walk costs two stats.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above %s", dir)
		}
		dir = parent
	}
}
