package shotclaims

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The three questions the manifest can answer about itself.
//
// The fourth — whether the strings are really on the screen — is not here
// and cannot be: it needs the app, so it lives in the app's own package
// and this file only checks that something claims to ask it. See the
// package header.
//
// # Why one test and three subtests
//
// The same reason wasm/verify/sharedparse_test.go gives: each of these
// ends in an arm that stops on a reading that came back empty, and a stop
// ends the goroutine. Three questions in one function is three questions
// any one of which can silence the other two — and the silence here is
// the failure mode the whole package exists to end.
func TestEveryScreenshotIsClaimedAndEveryClaimIsShown(t *testing.T) {
	root := filepath.Join("..", "..")

	t.Run("the manifest against the image directory", func(t *testing.T) {
		checkFilesAndClaimsAgree(t, root)
	})
	t.Run("each row's own shape", func(t *testing.T) {
		checkRowsAreWellFormed(t, root)
	})
	t.Run("the README's captions against the claims", func(t *testing.T) {
		checkCaptionsQuoteTheScreen(t, root)
	})
	t.Run("the harness that takes them", func(t *testing.T) {
		checkEveryClaimIsTakeable(t, root)
	})
}

// imageDir is where the shots live, relative to the repository root. One
// constant because four checks name it and a reader chasing a failure
// wants one answer to "where".
const imageDir = "docs/images"

// scriptDir is where the actions that produce them live. One action
// script per image, named after it: docs/images/todo.png is taken by
// wasm/shots/scripts/todo.js.
const scriptDir = "wasm/shots/scripts"

// checkFilesAndClaimsAgree holds the manifest and the directory to being
// the same set, in both directions.
//
// Both, because the two failures are different and neither implies the
// other. An unclaimed file is a picture nothing holds — which is the
// state this package was written to end, arriving again by somebody
// adding a shot. A claim with no file is a row describing nothing, and it
// is the more confusing of the two: its per-app test goes on passing, so
// the repository reports that a screenshot which does not exist is
// accurate.
func checkFilesAndClaimsAgree(t *testing.T, root string) {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(root, imageDir))
	if err != nil {
		t.Fatalf("reading %s: %v", imageDir, err)
	}
	onDisk := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		onDisk[e.Name()] = true
	}
	// The reaching arm. An empty directory would report every claim as a
	// row describing nothing — a thousand findings in the shape of one
	// wrong reading — and the repository has seven shots in it.
	if len(onDisk) == 0 {
		t.Fatalf("%s holds no files, and this manifest has %d claim(s). "+
			"The reading is wrong rather than the repository being bare.",
			imageDir, len(Claims))
	}

	claimed := map[string]bool{}
	for _, c := range Claims {
		if claimed[c.File] {
			t.Errorf("%s is claimed twice. A file has one state and one set "+
				"of strings; two rows for it means one of them is describing "+
				"a picture that was replaced.", c.File)
		}
		claimed[c.File] = true
		if !onDisk[c.File] {
			t.Errorf("the manifest claims %s/%s and no such file is there.\n\n"+
				"Its per-app test goes on passing while this is true, so the "+
				"repository is reporting that a screenshot it does not have "+
				"is accurate. Either the shot was renamed — say so here — or "+
				"it is gone and this row should go with it.", imageDir, c.File)
		}
	}
	for _, name := range sortedKeys(onDisk) {
		if claimed[name] {
			continue
		}
		t.Errorf("%s/%s is in the tree and nothing claims it.\n\n"+
			"A screenshot with no claim is a fact about a screen written "+
			"down on the day it was true, with nothing that reads it — which "+
			"is what this package exists to stop. Add a row saying which app "+
			"it is of, how it was driven there, and what is legible in it.",
			imageDir, name)
	}
}

// checkRowsAreWellFormed holds each row to being one of the two shapes
// this manifest has: a shot of an app, or a composite of other shots.
//
// The split is not decoration. A composite shows nothing its parts do not,
// so giving it Shows of its own would be asserting the same strings twice
// with one copy free to drift — the exact failure the package is about,
// reintroduced by the manifest itself.
func checkRowsAreWellFormed(t *testing.T, root string) {
	t.Helper()

	byFile := map[string]bool{}
	for _, c := range Claims {
		byFile[c.File] = true
	}

	for _, c := range Claims {
		if c.Composite() {
			if c.Package != "" || len(c.Shows) != 0 || c.Test != "" {
				t.Errorf("%s is a composite and also carries an app, strings "+
					"or a test. A composite is a picture of pictures: what it "+
					"shows is held by the claims it is made of, and a second "+
					"copy here is a claim nothing re-derives.", c.File)
			}
			for _, part := range c.MadeOf {
				if !byFile[part] {
					t.Errorf("%s is made of %s and no claim names that file.",
						c.File, part)
				}
			}
			continue
		}

		// A shot of an app. Every field is load-bearing: the directory is
		// where a person goes, the state is how they get back to the
		// picture, the strings are what is checked, and the test is what
		// checks them.
		if c.Package == "" || c.State == "" || len(c.Shows) == 0 || c.Test == "" {
			t.Errorf("%s is missing one of the four things a claim needs: "+
				"package %q, state %q, %d string(s), test %q.", c.File,
				c.Package, c.State, len(c.Shows), c.Test)
			continue
		}
		if _, err := os.Stat(filepath.Join(root, c.Package)); err != nil {
			t.Errorf("%s says it is a shot of %s and that directory is not "+
				"there: %v", c.File, c.Package, err)
		}
		shown := map[string]bool{}
		for _, s := range c.Shows {
			if shown[s] {
				t.Errorf("%s lists %q twice.", c.File, s)
			}
			shown[s] = true
		}
		for _, q := range c.Quoted {
			if !shown[q] {
				t.Errorf("%s says its caption quotes %q and that is not one "+
					"of the strings the shot is held to.\n\n"+
					"Quoted is the join between the README's prose and a "+
					"rendered tree, and it only joins them where the same "+
					"string is on both sides. A quotation that is not in "+
					"Shows is held against the caption and against nothing "+
					"else, which is the state the caption was already in.",
					c.File, q)
			}
		}
	}
}

// imgTag reads one HTML <img> out of the markdown. The README frames its
// shots in raw HTML rather than in markdown image syntax because it
// centres and sizes them, so this is the only form that has to be read.
var imgTag = regexp.MustCompile(`<img\s+[^>]*>`)

// attr pulls one double-quoted attribute out of a tag.
func attr(tag, name string) string {
	m := regexp.MustCompile(name + `="([^"]*)"`).FindStringSubmatch(tag)
	if m == nil {
		return ""
	}
	return m[1]
}

// checkCaptionsQuoteTheScreen holds the README's alt text to the claims.
//
// # What this is actually for
//
// Alt text is the caption a reader who cannot see the picture gets, and
// this repository's alt text states screen contents: "showing Count: 3",
// "the error 'The two passwords differ'", "0 of 55 lessons opened". Those
// are the same kind of sentence as the six lesson counts that were wrong
// in three directions at once — a fact about a screen, transcribed.
//
// So the quotations are held from both ends. The app's own test says the
// string is really rendered; this says the caption really says it. A
// wording change then fails on the side that changed rather than
// silently making the other side's copy stale.
//
// # Why both directions of the reference are checked
//
// An image the README does not show is one nobody is served by keeping
// current, and a README image with no claim is the original problem
// walking back in through the document rather than through the directory.
func checkCaptionsQuoteTheScreen(t *testing.T, root string) {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("reading README.md: %v", err)
	}
	byFile := map[string]Claim{}
	for _, c := range Claims {
		byFile[c.File] = c
	}

	shown := map[string]bool{}
	for _, tag := range imgTag.FindAllString(string(raw), -1) {
		src := attr(tag, "src")
		if !strings.HasPrefix(src, imageDir+"/") {
			// An image from somewhere else entirely is not this
			// manifest's business; nothing in the README has one today,
			// and a badge or an external asset arriving should not fail
			// a check about screenshots.
			continue
		}
		name := strings.TrimPrefix(src, imageDir+"/")
		claim, ok := byFile[name]
		if !ok {
			t.Errorf("README.md shows %s and nothing claims it.\n\n"+
				"Every screenshot in the README is a statement about a "+
				"screen, and one with no claim is a statement nothing "+
				"re-derives. Add a row to Claims.", src)
			continue
		}
		shown[name] = true

		alt := attr(tag, "alt")
		if alt == "" {
			t.Errorf("README.md shows %s with no alt text.", src)
			continue
		}
		for _, q := range claim.Quoted {
			if !strings.Contains(alt, q) {
				t.Errorf("README.md's caption for %s does not quote %q.\n\n"+
					"caption: %s\n\n"+
					"That string is what the claim and the caption have in "+
					"common — it is asserted against a rendered tree by %s "+
					"and stated here in prose. If the screen's wording "+
					"changed, both ends move together; if the caption was "+
					"simply reworded, drop the entry from Quoted rather than "+
					"leaving a join that joins nothing.",
					src, q, alt, claim.Test)
			}
		}
	}
	for _, c := range Claims {
		if !shown[c.File] {
			t.Errorf("%s/%s is claimed and the README does not show it.\n\n"+
				"The cost of a shot is keeping it true, and the reason to pay "+
				"it is that somebody reads it. A picture nothing displays is "+
				"neither.", imageDir, c.File)
		}
	}
}

// sortedKeys is the set in a stable order, so that findings from two runs
// can be diffed against each other. Map order cannot be.
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// shotHeader is the one line of JSON every action script opens with,
// saying which app it drives and how big the frame is. Parsed here rather
// than merely looked for, because the half worth checking is the app name:
// a script that drives todoapp under a claim that says the picture is of
// signup is a harness and a manifest that disagree about what the file
// is a picture of, and neither of them would notice.
var shotHeader = regexp.MustCompile(`(?m)^\s*//\s*grmob-shot:\s*(\{.*\})\s*$`)

// checkEveryClaimIsTakeable holds the manifest against the harness that
// produces the images.
//
// # Why this is worth a check of its own
//
// The first version of this harness was deleted rather than committed, and
// re-taking a screenshot then meant rebuilding an import-swap script, a
// host page and a CDP driver out of a session document. What replaced it
// is wasm/shots, and what stops it decaying the same way is that a claim
// with no way to re-take it is exactly as unpinned as the picture was: the
// test in the app's package would still say the strings are on the screen,
// and nobody could produce the image that shows them.
//
// Both directions again. A claim with no script cannot be re-taken; a
// script with no claim takes a picture nothing holds, which is where this
// whole mechanism came in.
func checkEveryClaimIsTakeable(t *testing.T, root string) {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(root, scriptDir))
	if err != nil {
		t.Fatalf("reading %s: %v", scriptDir, err)
	}
	scripts := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".js") {
			continue
		}
		scripts[strings.TrimSuffix(e.Name(), ".js")] = true
	}
	if len(scripts) == 0 {
		t.Fatalf("%s holds no action scripts, and this manifest has %d "+
			"claim(s). The reading is wrong rather than the harness being "+
			"empty.", scriptDir, len(Claims))
	}

	claimed := map[string]bool{}
	for _, c := range Claims {
		base := strings.TrimSuffix(c.File, filepath.Ext(c.File))
		claimed[base] = true
		if !scripts[base] {
			t.Errorf("nothing takes %s/%s: %s/%s.js is not there.\n\n"+
				"A picture that cannot be re-taken is a picture that will "+
				"be wrong and stay wrong, because the cost of correcting it "+
				"is reconstructing a harness. Add the script, or drop the "+
				"claim and the image with it.", imageDir, c.File, scriptDir, base)
			continue
		}
		checkScriptDrivesTheClaimedApp(t, root, base, c)
	}
	for _, name := range sortedKeys(scripts) {
		if !claimed[name] {
			t.Errorf("%s/%s.js takes a picture nothing claims.\n\n"+
				"Either it writes %s/%s.png and that image needs a row here, "+
				"or it is left over from a shot that has gone.",
				scriptDir, name, imageDir, name)
		}
	}
}

// checkScriptDrivesTheClaimedApp holds one script's header to its claim.
func checkScriptDrivesTheClaimedApp(t *testing.T, root, base string, c Claim) {
	t.Helper()

	path := filepath.Join(scriptDir, base+".js")
	raw, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Errorf("reading %s: %v", path, err)
		return
	}
	m := shotHeader.FindSubmatch(raw)
	if m == nil {
		t.Errorf("%s has no `// grmob-shot: {…}` header, so nothing in it "+
			"says which app it drives or how big the frame is. The driver "+
			"refuses such a script; this says so at test time instead.", path)
		return
	}
	var head struct {
		App  string `json:"app"`
		Page string `json:"page"`
	}
	if err := json.Unmarshal(m[1], &head); err != nil {
		t.Errorf("%s: the grmob-shot header is not JSON: %v", path, err)
		return
	}
	if c.Composite() {
		if head.Page == "" {
			t.Errorf("%s claims to be a composite and %s mounts an app "+
				"(%q) rather than naming a page of its own.", c.File, path, head.App)
		}
		return
	}
	// The claim says "examples/todoapp" and the header says "todoapp":
	// the same word with the directory in front of it, which is the whole
	// point of choosing the app by its directory name in the host.
	if want := strings.TrimPrefix(c.Package, "examples/"); head.App != want {
		t.Errorf("%s says docs/images/%s is a shot of %s, and %s drives "+
			"%q.\n\n"+
			"One of the two is a picture of a different app from the one it "+
			"is held to, and the test that asserts its strings would go on "+
			"passing either way — it renders the app the CLAIM names.",
			"the manifest", c.File, c.Package, path, head.App)
	}
}
