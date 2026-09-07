package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/aria/spec"
)

// What this command does that aria/spec does not, checked.
//
// # Why it needed a test at all
//
// Everything interesting about the ARIA fixture is in aria/spec: the parse, the
// scoping, the rendering, the version guard. All four are covered there, and
// `go test ./...` reaches them through aria/verify's conformance test. What it
// never reached was this file — the walk up to the module root, the two paths,
// the error a missing download produces, and the write itself. A command whose
// only coverage is "somebody ran it once and the diff looked right" is a
// command that can be broken by an edit to any of those four and stay silent
// until the next time a human regenerates.
//
// The failures are all plumbing failures and they are all quiet ones, which is
// the same argument the floors in spec.Parse rest on:
//
//	the wrong path written   a fixture regenerated somewhere nothing reads,
//	                         and a stale aria.json that every guard still
//	                         passes against because it did not change
//	the wrong path read      LocalPath and FixturePath are both under aria/,
//	                         both are JSON-or-HTML, and reading the fixture
//	                         as a spec fails with spec.Parse's floor rather
//	                         than with anything naming the mistake
//	a partial run            an error after the write leaves a fixture from a
//	                         document the command then refused
//	the missing download     the one error a human actually meets, and the
//	                         only thing standing between them and "no such
//	                         file or directory" is the sentence it is wrapped
//	                         in
//
// # The document these tests generate
//
// A synthetic specification rather than the real 1.4MB download, for the reason
// aria/spec's own tests use one: the download is deliberately not committed, so
// a test that needed it would not run. It has to satisfy three things at once —
// spec.Parse's two floors, the version guard, and spec.Scope's requirement that
// every role in the fixture's scope have a section — and syntheticSpec builds
// exactly that and nothing else.
//
// The expected output is never written out here. Each test compares what run
// wrote against what spec.Scope and Fixture.Render say about the same document,
// which is the arrangement internal/valuefixture makes for the same reason: a
// transcription of what the generator was believed to emit would agree with a
// generator that had stopped calling Scope.

// syntheticSpec is a document in ReSpec's shape holding every role the fixture
// scopes, plus enough filler to clear spec.Parse's floors.
//
// The floors are the reason for the filler: Parse refuses a document with fewer
// than 80 role sections or fewer than 8 implicit-orientation sentences, because
// both are what a silent markup change looks like. Neither number is a fact
// about this test, so both are read from the failure they would cause rather
// than guessed at — the counts below are comfortably past them.
//
// A handful of the scoped roles carry real cells so the rendered fixture has
// content to be wrong about. An all-empty document would produce a fixture in
// which every entry is `{"attributes": []}`, and a generator that had dropped
// Scope's attribute filter entirely would still match it.
func syntheticSpec(t *testing.T) string {
	t.Helper()

	// The cells a few named roles get. Chosen as the ones aria/verify actually
	// argues about: the value family on progressbar, a selection on tab, the
	// level on heading, and an orientation with a required-owned pair on
	// tablist.
	cells := map[string]string{
		"progressbar": `<tr><th class="role-required-properties-head" scope="row">Required States and Properties:</th>
		    <td class="role-required-properties"><ul>
		    <li><a href="#aria-valuenow" class="property-reference"><code>aria-valuenow</code></a></li>
		    <li><a href="#aria-valuemin" class="property-reference"><code>aria-valuemin</code></a></li>
		    <li><a href="#aria-valuemax" class="property-reference"><code>aria-valuemax</code></a></li>
		    <li><a href="#aria-valuetext" class="property-reference"><code>aria-valuetext</code></a></li>
		    </ul></td></tr>`,
		"tab": `<tr><th class="role-properties-head" scope="row">Supported States and Properties:</th>
		    <td class="role-properties"><a href="#aria-selected" class="state-reference"><code>aria-selected</code></a></td></tr>`,
		"heading": `<tr><th class="role-required-properties-head" scope="row">Required States and Properties:</th>
		    <td class="role-required-properties"><a href="#aria-level" class="property-reference"><code>aria-level</code></a></td></tr>`,
		"tablist": `<tr><th class="role-mustcontain-head" scope="row">Required Owned Elements:</th>
		    <td class="role-mustcontain"><a href="#tab" class="role-reference"><code>tab</code></a></td></tr>`,
	}

	// The roles that carry the implicit-orientation sentence. Nine, matching
	// what ARIA 1.2 publishes, so the floor is cleared by the same margin the
	// real document clears it by.
	oriented := map[string]string{
		"tablist": "horizontal", "toolbar": "horizontal", "separator": "horizontal",
		"listbox": "vertical", "menu": "vertical", "menubar": "horizontal",
		"scrollbar": "vertical", "slider": "horizontal", "tree": "vertical",
	}

	section := func(name string) string {
		var b strings.Builder
		fmt.Fprintf(&b, "<section class=\"role notoc\" id=%q>\n", name)
		fmt.Fprintf(&b, `  <div class="role-description"><p>The %s role.</p>`, name)
		if axis, ok := oriented[name]; ok {
			// Worded exactly as the published document words it — this is the
			// one fact in the whole parse that is prose rather than a table
			// cell, and a test whose sentence differed from the real one would
			// be exercising a regex nothing else uses.
			fmt.Fprintf(&b, `<p>Elements with the role <code>%s</code> have an implicit
    <a href="#aria-orientation" class="property-reference"><code>aria-orientation</code></a>
    value of <code>%s</code>.</p>`, name, axis)
		}
		b.WriteString("</div>\n")
		b.WriteString(`  <table class="role-features"><tbody>` + "\n")
		if extra, ok := cells[name]; ok {
			fmt.Fprintf(&b, "    %s\n", extra)
		}
		// Every section states Name From, because a section without one is a
		// synonym stub as far as parseDoc is concerned.
		b.WriteString(`    <tr><th class="role-namefrom-head" scope="row">Name From:</th>
        <td class="role-namefrom">author</td></tr>` + "\n")
		b.WriteString("  </tbody></table>\n</section>\n")
		return b.String()
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<h1 id="title" class="title">Accessible Rich Internet `+
		`Applications (WAI-ARIA) %s</h1>`+"\n", spec.Version)
	scoped := spec.ScopedRoles()
	for _, name := range scoped {
		b.WriteString(section(name))
	}
	// Filler up to 90 sections, named so they cannot collide with a real role.
	for i := len(scoped); i < 90; i++ {
		b.WriteString(section(fmt.Sprintf("filler%c%c", 'a'+i/26, 'a'+i%26)))
	}
	return b.String()
}

// wantFixture is what the authority says about the same document: the bytes
// spec.Scope and Fixture.Render produce. Never a transcription.
func wantFixture(t *testing.T, html string) []byte {
	t.Helper()
	doc, err := spec.Parse(html)
	if err != nil {
		t.Fatalf("the synthetic document does not parse: %v", err)
	}
	fx, err := spec.Scope(doc)
	if err != nil {
		t.Fatalf("the synthetic document does not scope: %v", err)
	}
	return fx.Render()
}

// fakeRoot lays out a module the way this command expects to find one: a
// go.mod at the top, the two directories the paths name, and the given
// document at LocalPath (or no document at all when html is "").
//
// It returns the root. The caller chdir's into it, or into a subdirectory of
// it, which is the half of run that repoRoot exists for.
func fakeRoot(t *testing.T, html string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"),
		[]byte("module example.com/fake\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{spec.LocalPath, spec.FixturePath} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if html != "" {
		if err := os.WriteFile(filepath.Join(root, spec.LocalPath), []byte(html), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// The command reads LocalPath, writes FixturePath, and finds both from
// wherever it was run.
//
// The working directory is a *subdirectory* on purpose. `go run ./aria/gen`
// leaves the caller wherever they were — which is the repository root for
// almost everyone and is not for anyone who runs it from inside aria/ — and
// repoRoot is the whole of what makes those two the same run. A version of it
// that used the working directory directly passes from the root and writes
// aria/verify/testdata/aria.json into a subdirectory from anywhere else, which
// is a fixture regenerated where nothing reads it.
func TestGenWritesTheFixtureFromWhereverItIsRun(t *testing.T) {
	html := syntheticSpec(t)
	want := wantFixture(t, html)

	for _, from := range []string{".", "aria", "aria/spec/testdata"} {
		t.Run(from, func(t *testing.T) {
			root := fakeRoot(t, html)
			// A fixture already on disk, and not the one the run should
			// produce. The write has to replace it: a command that created
			// the file and skipped an existing one would pass every check
			// below that only looked for valid JSON.
			stale := filepath.Join(root, spec.FixturePath)
			if err := os.WriteFile(stale, []byte("{\"stale\": true}\n"), 0o644); err != nil {
				t.Fatal(err)
			}

			t.Chdir(filepath.Join(root, from))
			var out bytes.Buffer
			if err := run(&out); err != nil {
				t.Fatalf("run from %s: %v", from, err)
			}

			got, err := os.ReadFile(stale)
			if err != nil {
				t.Fatalf("no fixture at %s after a successful run: %v", spec.FixturePath, err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("the fixture written from %s is not what spec.Scope and "+
					"Fixture.Render say about the same document.\n got %d bytes\nwant %d bytes",
					from, len(got), len(want))
			}

			// The summary is the only thing a person running this sees, and
			// both of its numbers are claims: the role count is the fixture's
			// scope and the prohibition count is a fact about the whole
			// document rather than about the scope.
			doc, err := spec.Parse(html)
			if err != nil {
				t.Fatal(err)
			}
			fx, err := spec.Scope(doc)
			if err != nil {
				t.Fatal(err)
			}
			line := fmt.Sprintf("OK: %d roles, %d name-prohibited -> %s\n",
				len(fx.Order), len(fx.NameProhibited), spec.FixturePath)
			if out.String() != line {
				t.Errorf("summary line is %q, want %q", out.String(), line)
			}
		})
	}
}

// The spec is read from LocalPath and the fixture is written to FixturePath,
// and neither is the other.
//
// Worth its own assertion because the two paths are siblings under aria/ and a
// swap is the kind of edit that looks right. Reading the fixture as a spec
// fails with spec.Parse's version guard, which names ARIA rather than naming
// the swap; writing the spec's regenerated content over the download fails with
// nothing at all, because the download is not committed and nobody would
// notice until the next fetch.
func TestGenLeavesTheDownloadAlone(t *testing.T) {
	html := syntheticSpec(t)
	root := fakeRoot(t, html)
	t.Chdir(root)

	if err := run(&bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	after, err := os.ReadFile(filepath.Join(root, spec.LocalPath))
	if err != nil {
		t.Fatalf("the download is gone after a run: %v", err)
	}
	if string(after) != html {
		t.Error("the run rewrote the specification it was asked to read — the two " +
			"paths are siblings and a swap between them is silent, since the " +
			"download is not committed and nothing checks it")
	}
}

// The error a human actually meets.
//
// This is the first run on a fresh checkout, every time, because the download
// is deliberately not committed. What os.ReadFile alone would say is "open
// aria/spec/testdata/wai-aria-1.2.html: no such file or directory", which is
// true and tells somebody who has never read aria/fetch.sh's header nothing at
// all — and the file's absence is not a mistake, it is the state the repository
// ships in.
func TestAMissingDownloadNamesTheFetch(t *testing.T) {
	root := fakeRoot(t, "")
	t.Chdir(root)

	err := run(&bytes.Buffer{})
	if err == nil {
		t.Fatal("run succeeded with no specification on disk")
	}
	for _, want := range []string{spec.LocalPath, "aria/fetch.sh", "not committed"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not mention %q: %v", want, err)
		}
	}
	if _, statErr := os.Stat(filepath.Join(root, spec.FixturePath)); statErr == nil {
		t.Error("a failed run wrote a fixture anyway")
	}
}

// A refusal from spec.Parse reaches the caller intact and writes nothing.
//
// The stale-edition case is the one that matters, and the two halves are
// separate claims. The message is spec.Version's, checked there; what is
// checked here is that this command does not swallow it and — the half only
// this file can be wrong about — that the fixture on disk is untouched. A
// generator that wrote before it validated would leave a checkout holding four
// rewritten facts *and* an error message telling the reader to re-fetch, which
// is the worst of both.
func TestARefusedDocumentLeavesTheFixtureUntouched(t *testing.T) {
	stale := strings.Replace(syntheticSpec(t),
		"(WAI-ARIA) "+spec.Version, "(WAI-ARIA) 1.1", 1)
	root := fakeRoot(t, stale)
	committed := []byte("{\"the\": \"committed fixture\"}\n")
	if err := os.WriteFile(filepath.Join(root, spec.FixturePath), committed, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	err := run(&bytes.Buffer{})
	if err == nil {
		t.Fatal("run regenerated the fixture from a WAI-ARIA 1.1 document")
	}
	if !strings.Contains(err.Error(), "1.1") {
		t.Errorf("the refusal reached the caller without naming the edition: %v", err)
	}
	got, readErr := os.ReadFile(filepath.Join(root, spec.FixturePath))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, committed) {
		t.Error("the fixture was rewritten by a run that then failed — the diff a " +
			"reader is left holding is not from the document they were told to fetch")
	}
}

// Outside a module there is no root to resolve the two paths against, and the
// only honest thing to do is say so.
//
// repoRoot walking to / and stopping is the arm that says it: without the
// terminating check it is an infinite loop, and with a wrong one it returns
// "/" and the command tries to write /aria/verify/testdata/aria.json.
func TestOutsideAModuleTheCommandSaysWhy(t *testing.T) {
	t.Chdir(t.TempDir())

	err := run(&bytes.Buffer{})
	if err == nil {
		t.Fatal("run found a module root above a temporary directory")
	}
	if !strings.Contains(err.Error(), "go.mod") {
		t.Errorf("the error does not name what it was looking for: %v", err)
	}
}
