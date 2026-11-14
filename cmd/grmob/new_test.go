package main

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot is this checkout: cmd/grmob is two levels down.
var repoRoot = filepath.Join("..", "..")

func TestOutputPath(t *testing.T) {
	for in, want := range map[string]string{
		"wasm/main.go.tmpl": filepath.Join("wasm", "main.go"),
		"build.sh.tmpl":     "build.sh",
		"dot-gitignore":     ".gitignore",
		"app/app.go.tmpl":   filepath.Join("app", "app.go"),
	} {
		if got := outputPath(in); got != want {
			t.Errorf("outputPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDerivedIdentities(t *testing.T) {
	for base, want := range map[string][2]string{
		"hello":       {"Hello", "hello"},
		"my-todo_app": {"My Todo App", "mytodoapp"},
		"2048":        {"2048", "app"}, // an ID segment may not start with a digit
		"Café":        {"Café", "caf"},
	} {
		if got := displayName(base); got != want[0] {
			t.Errorf("displayName(%q) = %q, want %q", base, got, want[0])
		}
		if got := idSegment(base); got != want[1] {
			t.Errorf("idSegment(%q) = %q, want %q", base, got, want[1])
		}
		if err := validateID("com.example." + idSegment(base)); err != nil {
			t.Errorf("the derived ID for %q does not validate: %v", base, err)
		}
	}
	for _, bad := range []string{"app", "com.example.my-app", "com.1example.app", "com..app", "com.example.my_app"} {
		if validateID(bad) == nil {
			t.Errorf("validateID(%q) accepted an ID one of the platforms rejects", bad)
		}
	}
}

// TestShellPatchesStillApply runs every identity patch against this
// repository's own shells. The native commands copy these exact files into an
// app, so an edit here that moves an anchor (renames the demo's
// applicationId, reformats project.yml) fails this test rather than failing a
// user's first native build.
func TestShellPatchesStillApply(t *testing.T) {
	// An upper-case letter in the ID, so the scheme's lowering is exercised.
	cfg := appConfig{Name: `Tom & "Jerry"`, ID: "com.example.TomJerry"}
	for _, tc := range []struct {
		spec    shellSpec
		patches []patch
		expect  map[string][]string // file → substrings the patched file must contain
		refuse  map[string][]string // file → demo text that must be gone from it
	}{
		{androidShell, androidPatches(cfg), map[string][]string{
			"app/build.gradle": {`applicationId = "com.example.TomJerry"`},
			"app/src/main/AndroidManifest.xml": {
				`android:label="Tom &amp; &quot;Jerry&quot;"`,
				`<data android:scheme="com.example.tomjerry" />`,
			},
		}, map[string][]string{
			"app/src/main/AndroidManifest.xml": {`android:scheme="grmob"`, "grmob://"},
		}},
		{iosShell, iosPatches(cfg), map[string][]string{
			"project.yml": {
				`CFBundleDisplayName: "Tom & \"Jerry\""`,
				"CFBundleURLName: com.example.TomJerry.deeplink",
				"CFBundleURLSchemes: [com.example.tomjerry]",
				`NSCameraUsageDescription: "Tom & \"Jerry\" uses the camera when you allow it."`,
				`NSPhotoLibraryUsageDescription: "Tom & \"Jerry\" uses your photo library when you allow it."`,
				// What removing the UI tests must leave behind: the app target's
				// last setting, then the scheme building it.
				"      OTHER_LDFLAGS: -ObjC\nschemes:\n  GrMobApp:\n    build:\n      targets:\n        GrMobApp: all\n",
			},
		}, map[string][]string{
			"project.yml": {"GrMobUITests", "Demonstrates permission", "[grmob]", "com.grmob.deeplink", "grmob://"},
		}},
	} {
		files := map[string]string{}
		for _, p := range tc.patches {
			if _, ok := files[p.file]; !ok {
				b, err := os.ReadFile(filepath.Join(repoRoot, tc.spec.dir, p.file))
				if err != nil {
					t.Fatal(err)
				}
				files[p.file] = string(b)
			}
			out, err := p.apply(files[p.file])
			if err != nil {
				t.Errorf("%s/%s: %v", tc.spec.dir, p.file, err)
				continue
			}
			files[p.file] = out
		}
		for file, wants := range tc.expect {
			for _, want := range wants {
				if !strings.Contains(files[file], want) {
					t.Errorf("%s/%s after patching does not contain %q", tc.spec.dir, file, want)
				}
			}
		}
		for file, gone := range tc.refuse {
			for _, g := range gone {
				if strings.Contains(files[file], g) {
					t.Errorf("%s/%s after patching still contains %q, which is grmob's demo and not the app's", tc.spec.dir, file, g)
				}
			}
		}
	}
}

// The alarms opt-in (appConfig.Alarms) decides whether the two policy-gated
// declarations survive the copy: without it neither the full-screen-intent
// permission nor the AlarmKit usage key, nor the comments explaining them,
// reach the app; with it both do, and the usage key names the app rather than
// grmob's demo.
func TestShellPatchesFollowTheAlarmsOptIn(t *testing.T) {
	apply := func(t *testing.T, dir string, patches []patch, file string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(repoRoot, dir, file))
		if err != nil {
			t.Fatal(err)
		}
		out := string(b)
		for _, p := range patches {
			if p.file != file {
				continue
			}
			if out, err = p.apply(out); err != nil {
				t.Fatalf("%s/%s: %v", dir, file, err)
			}
		}
		return out
	}
	const manifest = "app/src/main/AndroidManifest.xml"
	for _, alarms := range []bool{false, true} {
		cfg := appConfig{Name: "Wake", ID: "com.example.wake", Alarms: alarms}
		android := apply(t, "android", androidPatches(cfg), manifest)
		ios := apply(t, "ios", iosPatches(cfg), "project.yml")

		has := map[string]bool{
			"the full-screen-intent permission": strings.Contains(android, `android:name="`+fullScreenIntent+`"`),
			"its manifest comment":              strings.Contains(android, "<!-- permission.Alarms"),
			"the AlarmKit usage key":            strings.Contains(ios, alarmKitUsageKey+":"),
			"its project.yml comment":           strings.Contains(ios, "# permission.Alarms"),
		}
		for what, present := range has {
			if present != alarms {
				t.Errorf("alarms=%v: %s present = %v", alarms, what, present)
			}
		}
		if alarms && !strings.Contains(ios, alarmKitUsageKey+`: "Wake rings the alarms you set`) {
			t.Errorf("alarms=true: the AlarmKit usage key does not name the app")
		}
		if strings.Contains(ios, "Demonstrates permission.Alarms") {
			t.Errorf("alarms=%v: the demo's AlarmKit text reached the app", alarms)
		}
		// The neighbours of what was removed are untouched.
		if !strings.Contains(android, "RECEIVE_BOOT_COMPLETED") || !strings.Contains(ios, "NSPhotoLibraryUsageDescription:") {
			t.Errorf("alarms=%v: the removal took a neighbouring declaration with it", alarms)
		}
	}
}

// TestShellSkipsNameRealPaths keeps the skip lists honest in one direction:
// a skip entry naming something the shell no longer has is dead weight at
// best, and at worst a sign the thing it meant to exclude was renamed and is
// now being copied into every app.
func TestShellSkipsNameRealPaths(t *testing.T) {
	// Entries for build state exist only after a build and are expected to
	// be absent from a clean checkout.
	buildState := map[string]bool{
		".gradle": true, ".kotlin": true, ".idea": true, "build": true, "app/build": true,
		"app/libs": true, "local.properties": true, "Frameworks": true,
		"GrMobApp.xcodeproj": true, "DerivedData": true, "xcuserdata": true, "GrMob/Info.plist": true,
	}
	for _, spec := range []shellSpec{androidShell, iosShell} {
		for _, s := range spec.skip {
			if buildState[s] {
				continue
			}
			if !fileExists(filepath.Join(repoRoot, spec.dir, filepath.FromSlash(s))) {
				t.Errorf("%s skip list names %q, which %s/ does not have", spec.dir, s, spec.dir)
			}
		}
	}
}

// TestNewScaffoldsAnAppThatBuildsAndPasses is the end-to-end check of the
// browser path: scaffold against this checkout, let new run the app's own
// build.sh (the js/wasm compile plus the runtime copy), then run the app's
// test and vet its native-side code. It compiles grmob for js/wasm — about a
// second on a warm build cache — and is not behind -short, which this
// repository keeps as one lever (see wasm/verify/shortlever_test.go).
func TestNewScaffoldsAnAppThatBuildsAndPasses(t *testing.T) {
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "hello-app")
	if err := cmdNew([]string{dir, "-replace", abs, "-name", "Hello"}); err != nil {
		t.Fatalf("grmob new: %v", err)
	}

	for _, f := range []string{"grmob.json", ".gitignore", "wasm/main.wasm", "wasm/wasm_exec.js", "wasm/grmob-runtime.js", "wasm/camera.js"} {
		if !fileExists(filepath.Join(dir, f)) {
			t.Errorf("the scaffold has no %s", f)
		}
	}
	// The runtime copy must be byte-identical to the module's: that identity
	// is the point of copying it at build time.
	want, _ := os.ReadFile(filepath.Join(abs, "wasm", "grmob-runtime.js"))
	got, _ := os.ReadFile(filepath.Join(dir, "wasm", "grmob-runtime.js"))
	if string(want) != string(got) {
		t.Error("wasm/grmob-runtime.js in the app differs from the grmob module's")
	}
	cfg, err := readConfig(dir)
	if err != nil || cfg.Name != "Hello" || cfg.ID != "com.example.helloapp" {
		t.Errorf("grmob.json = %+v, %v", cfg, err)
	}

	for _, args := range [][]string{{"test", "./app"}, {"vet", "./app"}} {
		cmd := exec.Command("go", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("go %s in the scaffolded app: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	cmd := exec.Command("go", "vet", "./wasm")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("GOOS=js go vet ./wasm in the scaffolded app: %v\n%s", err, out)
	}

	// A second build must not rewrite the runtime copies: the dev server
	// reads a changed .js in wasm/ as a page edit and would reload the page
	// after every hot swap.
	before, _ := os.Stat(filepath.Join(dir, "wasm", "grmob-runtime.js"))
	build := exec.Command("sh", "build.sh")
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("second build: %v\n%s", err, out)
	}
	after, _ := os.Stat(filepath.Join(dir, "wasm", "grmob-runtime.js"))
	if !before.ModTime().Equal(after.ModTime()) {
		t.Error("a rebuild with no runtime change rewrote wasm/grmob-runtime.js")
	}

	// dev.sh compiles grmob's serve, which imports RWeb, from the app's
	// module. The tool line new writes is what keeps RWeb's go.sum entries
	// through tidy; doctor's check must pass with it and name the missing
	// sum without it — the state of every app made before new wrote it.
	if c := devServerCheck(dir); !c.ok {
		t.Errorf("dev server check on a fresh scaffold: %s", c.detail)
	}
	for _, args := range [][]string{{"mod", "edit", "-droptool=" + grmobModule + "/serve"}, {"mod", "tidy"}} {
		cmd := exec.Command("go", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	if c := devServerCheck(dir); c.ok || !strings.Contains(c.detail, "missing go.sum entry") {
		t.Errorf("dev server check without the tool line = ok %v, %q; want a missing go.sum entry", c.ok, c.detail)
	}
}

// TestWebHostPageSync walks cmdWeb's table against a scaffold of this
// checkout: a page new just wrote is current; an older page is left alone
// without -refresh and replaced with it; a missing one is written.
//
// "Replaced" is held to new's own output byte for byte. syncHostPage renders
// from the module directory and new from the embed; with -replace pointing
// here those are the same file, so any difference is a difference in how the
// two render — which would make every freshly scaffolded app warn.
func TestWebHostPageSync(t *testing.T) {
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "web-app")
	if err := cmdNew([]string{dir, "-replace", abs, "-name", "Web App", "-no-build"}); err != nil {
		t.Fatalf("grmob new: %v", err)
	}
	cfg, err := readConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(dir, filepath.FromSlash(hostPagePath))
	fresh, err := os.ReadFile(page)
	if err != nil {
		t.Fatalf("the scaffold has no %s: %v", hostPagePath, err)
	}

	step := func(what string, refresh bool, want hostPageState, content []byte) {
		t.Helper()
		got, err := syncHostPage(dir, cfg, refresh)
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		if got != want {
			t.Errorf("%s: state %d, want %d", what, got, want)
		}
		on, _ := os.ReadFile(page)
		if string(on) != string(content) {
			t.Errorf("%s: %s is not what it should be now:\n%.200s", what, hostPagePath, on)
		}
	}

	step("a page new just rendered", false, hostPageCurrent, fresh)

	older := []byte("<!DOCTYPE html>\n<!-- the page an older grmob scaffolded -->\n")
	if err := os.WriteFile(page, older, 0o644); err != nil {
		t.Fatal(err)
	}
	step("an older page, no -refresh", false, hostPageDiffers, older)
	step("an older page, -refresh", true, hostPageRefreshed, fresh)

	if err := os.Remove(page); err != nil {
		t.Fatal(err)
	}
	step("no page at all", false, hostPageWritten, fresh)
}

// Plan R9k: a scaffold that failed part-way left a half-made app, and the
// same command then refused the directory as not empty. A failure now removes
// what the run wrote. It is forced here after the templates and grmob.json
// are written, by a module path `go mod init` refuses, so no network is
// involved.
func TestNewThatFailsLeavesTheDirectoryAsItFoundIt(t *testing.T) {
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	const badModule = "not a module path!"

	// A directory new creates goes entirely.
	fresh := filepath.Join(t.TempDir(), "fresh-app")
	err = cmdNew([]string{fresh, "-replace", abs, "-module", badModule, "-no-build"})
	if err == nil {
		t.Fatal("new with an invalid module path reported success")
	}
	if !strings.Contains(err.Error(), "can be run again") {
		t.Errorf("the error does not say the run was undone: %v", err)
	}
	if fileExists(fresh) {
		t.Error("a failed new left the directory it created")
	}

	// A directory that existed with only .git keeps its .git and nothing else,
	// so the next run's emptiness check passes.
	repo := filepath.Join(t.TempDir(), "cloned")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := cmdNew([]string{repo, "-replace", abs, "-module", badModule, "-no-build"}); err == nil {
		t.Fatal("new with an invalid module path reported success")
	}
	entries, _ := os.ReadDir(repo)
	if len(entries) != 1 || entries[0].Name() != ".git" {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("after a failed new the directory holds %v, want only .git", names)
	}
	if _, err := requireEmptyDir(repo); err != nil {
		t.Errorf("the same command could not run again: %v", err)
	}
}

// Plan R9l: the display name went into the host page's <title>, a JS string
// and two Go string literals unescaped, so a name with a quote or an angle
// bracket broke the page or the app's Go code, and could inject markup. Each
// site now escapes for its context, and an ordinary name renders exactly as
// before.
func TestTheDisplayNameIsEscapedWhereItLands(t *testing.T) {
	render := func(name, tmpl string) string {
		t.Helper()
		src, err := templates.ReadFile(tmpl)
		if err != nil {
			t.Fatal(err)
		}
		out, err := renderTemplate(tmpl, src, scaffoldData{Module: "example.com/x", Name: name, ID: "com.example.x"})
		if err != nil {
			t.Fatal(err)
		}
		return string(out)
	}
	const page, app = "templates/wasm/index.html.tmpl", "templates/app/app.go.tmpl"

	// An ordinary name is untouched, which keeps `grmob web`'s comparison
	// quiet for every page an older grmob wrote.
	plain := render("My Notes", page)
	for _, want := range []string{"<title>My Notes</title>", `"My Notes failed to start.`} {
		if !strings.Contains(plain, want) {
			t.Errorf("an ordinary name changed in the page: no %q", want)
		}
	}
	if got := render("My Notes", app); !strings.Contains(got, `return "My Notes" }`) {
		t.Errorf("an ordinary name changed in app.go:\n%s", got)
	}

	hostile := `Tom "Tiny" </title><script>x()</script> \ App`
	got := render(hostile, page)
	if strings.Contains(got, "<script>x()") || strings.Contains(got, `"Tom "Tiny"`) {
		t.Errorf("the name reached the page unescaped:\n%s", got)
	}
	if !strings.Contains(got, "<title>Tom &#34;Tiny&#34; &lt;/title&gt;") {
		t.Errorf("the <title> is not HTML-escaped")
	}

	// app.go must still be Go: parse it.
	if _, err := parser.ParseFile(token.NewFileSet(), "app.go", render(hostile, app), 0); err != nil {
		t.Errorf("app.go does not parse with the name %q: %v", hostile, err)
	}

	if err := validateName("Two\nLines"); err == nil {
		t.Error("a name with a line break was accepted")
	}
	if err := validateName(hostile); err != nil {
		t.Errorf("a name with quotes and brackets was refused: %v", err)
	}
}
