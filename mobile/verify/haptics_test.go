package verify

import (
	"os"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// core.Haptic's kinds are spelled in four places that never compile
// together: core/haptics.go, the Kotlin object, the Swift enum and the
// browser runtime's pattern table. A kind a shell does not spell is silence
// on that shell and nothing else — no log, no failure — so this walks
// core.HapticKinds and requires every shell to spell every one, plus the
// event name and each dispatcher's arm.
func TestHapticEventSpellingsAgree(t *testing.T) {
	for _, shell := range []struct {
		name  string
		files []string
	}{
		{"android", []string{nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "Haptics.kt")}},
		{"ios", []string{nativeFile("ios", "GrMob", "App", "Haptics.swift")}},
		{"web", []string{nativeFile("wasm", "grmob-runtime.js")}},
	} {
		var src strings.Builder
		for _, f := range shell.files {
			raw, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("reading %s: %v", f, err)
			}
			src.Write(raw)
		}
		for _, kind := range core.HapticKinds() {
			if !strings.Contains(src.String(), `"`+string(kind)+`"`) {
				t.Errorf("%s shell never spells the %q haptic kind", shell.name, kind)
			}
		}
	}

	for _, c := range []struct{ file, arm string }{
		{nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "SystemEvents.kt"),
			`"haptic" -> Haptics.handle(data)`},
		{nativeFile("ios", "GrMob", "App", "SystemEvents.swift"),
			`case "haptic": Haptics.handle(object)`},
		{nativeFile("wasm", "grmob-runtime.js"),
			`GrMob.haptics.handle(JSON.parse(payloadJSON))`},
	} {
		raw, err := os.ReadFile(c.file)
		if err != nil {
			t.Fatalf("reading %s: %v", c.file, err)
		}
		if !strings.Contains(string(raw), c.arm) {
			t.Errorf("%s has no dispatch arm %q", c.file, c.arm)
		}
	}
}

// Vibrator.vibrate throws SecurityException without VIBRATE, so a shell
// that forgets the manifest line does not degrade to silence like every
// other haptics failure — it crashes the first time an app buzzes. VIBRATE
// is a normal permission (granted at install, never prompted), which is why
// it is not a permission.Permission and TestTheAndroidShellDeclaresEvery
// RuntimePermission does not cover it.
func TestTheAndroidShellDeclaresVibrate(t *testing.T) {
	path := nativeFile("android", "app", "src", "main", "AndroidManifest.xml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if !strings.Contains(string(raw), `android:name="android.permission.VIBRATE"`) {
		t.Errorf("%s does not declare android.permission.VIBRATE; core.Haptic would crash", path)
	}
}
