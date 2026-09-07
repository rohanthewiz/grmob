package verify

import (
	"bytes"
	"os"
	"path/filepath"
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
func TestTheFixtureIsWhatTheSpecificationSays(t *testing.T) {
	root := repoRoot(t)

	raw, err := os.ReadFile(filepath.Join(root, spec.LocalPath))
	if err != nil {
		t.Skipf("no local copy of the ARIA specification (%s): run `sh aria/fetch.sh` "+
			"to check the fixture against it", spec.LocalPath)
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
