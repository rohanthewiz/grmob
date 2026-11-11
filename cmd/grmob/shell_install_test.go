package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Plan R9g (N-137 g): the shell used to be copied straight into the app and
// patched afterwards, so an interrupted copy left a shell with grmob's demo
// identity and no record, which the next build adopted as the app's own.
// installShell stages everything beside the destination.

var testShell = shellSpec{dir: "android", skip: []string{"build"}, executable: []string{"gradlew"}}

// shellSource writes a tiny shell: an identity file to patch, a script that
// must be executable, and build output that must be skipped.
func shellSource(t *testing.T) string {
	t.Helper()
	src := t.TempDir()
	for name, body := range map[string]string{
		"app/build.gradle":   "applicationId \"com.grmob.app\"\n",
		"gradlew":            "#!/bin/sh\n",
		"build/intermediate": "not copied\n",
	} {
		path := filepath.Join(src, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return src
}

var identity = []patch{{
	file:  "app/build.gradle",
	apply: replaceOnce(`"com.grmob.app"`, `"com.example.notes"`),
}}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// noStaging asserts no .android-partial-* directory is left in root.
func noStaging(t *testing.T, root string) {
	t.Helper()
	left, _ := filepath.Glob(filepath.Join(root, ".android-partial-*"))
	if len(left) != 0 {
		t.Fatalf("staging left behind: %v", left)
	}
}

func TestInstallShellLandsPatchedAndRecorded(t *testing.T) {
	src, root := shellSource(t), t.TempDir()
	dst := filepath.Join(root, "android")

	if err := installShell(src, dst, "v1.0.0", testShell, identity); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(dst, "app/build.gradle")); !strings.Contains(got, "com.example.notes") {
		t.Errorf("identity not patched: %q", got)
	}
	if got := read(t, filepath.Join(dst, shellRecord)); got != "v1.0.0\n" {
		t.Errorf("record = %q", got)
	}
	if info, err := os.Stat(filepath.Join(dst, "gradlew")); err != nil || info.Mode()&0o100 == 0 {
		t.Errorf("gradlew is not executable: %v %v", info, err)
	}
	if fileExists(filepath.Join(dst, "build")) {
		t.Error("a skipped path was copied")
	}
	noStaging(t, root)
}

// The failure the plan names: a step that fails part-way leaves no android/
// at all, so the next run copies again instead of adopting a half-made shell.
func TestInstallShellThatFailsLeavesNoShell(t *testing.T) {
	src, root := shellSource(t), t.TempDir()
	dst := filepath.Join(root, "android")

	failing := []patch{{file: "app/build.gradle", apply: func(string) (string, error) {
		return "", errors.New("interrupted")
	}}}
	if err := installShell(src, dst, "v1.0.0", testShell, failing); err == nil {
		t.Fatal("a failing patch reported success")
	}
	if fileExists(dst) {
		t.Fatal("a failed install left android/ behind, unpatched and unrecorded")
	}
	noStaging(t, root)
}

// -refresh merges a staged, patched copy over the existing shell: the app's
// own files there survive, and the record moves to the new version. A failed
// refresh leaves the existing shell exactly as it was.
func TestInstallShellRefreshKeepsTheAppsFilesAndFailsCleanly(t *testing.T) {
	src, root := shellSource(t), t.TempDir()
	dst := filepath.Join(root, "android")
	if err := installShell(src, dst, "v1.0.0", testShell, identity); err != nil {
		t.Fatal(err)
	}
	signing := filepath.Join(dst, "keystore.properties")
	if err := os.WriteFile(signing, []byte("storeFile=release.jks\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	failing := []patch{{file: "app/build.gradle", apply: func(string) (string, error) {
		return "", errors.New("shell mismatch")
	}}}
	if err := installShell(src, dst, "v1.1.0", testShell, failing); err == nil {
		t.Fatal("a failing refresh reported success")
	}
	if got := read(t, filepath.Join(dst, shellRecord)); got != "v1.0.0\n" {
		t.Errorf("a failed refresh moved the record to %q", got)
	}
	if got := read(t, filepath.Join(dst, "app/build.gradle")); !strings.Contains(got, "com.example.notes") {
		t.Errorf("a failed refresh unpatched the shell: %q", got)
	}

	if err := installShell(src, dst, "v1.1.0", testShell, identity); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(dst, shellRecord)); got != "v1.1.0\n" {
		t.Errorf("record after refresh = %q", got)
	}
	if !fileExists(signing) {
		t.Error("the refresh dropped the app's own file")
	}
	noStaging(t, root)
}

// A staging directory from a run that was killed outright (no deferred
// cleanup ran) is removed by the next one.
func TestInstallShellClearsAKilledRunsStaging(t *testing.T) {
	src, root := shellSource(t), t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".android-partial-123", "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := installShell(src, filepath.Join(root, "android"), "v1.0.0", testShell, identity); err != nil {
		t.Fatal(err)
	}
	noStaging(t, root)
}
