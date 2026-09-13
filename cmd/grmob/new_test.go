package main

import (
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
}
