package verify

import (
	"os"
	"strings"
	"testing"
)

// core.Share is spelled in four places that never compile together:
// core/share.go, the Kotlin object, the Swift class and the browser runtime.
// A reply under the wrong name or without its id leaves Go holding the
// caller's done forever, so a share button's fallback never runs. This holds
// each shell to the event name, the request keys it reads and the reply keys
// core decodes, as TestClipboardEventSpellingsAgree does for the clipboard.
func TestShareEventSpellingsAgree(t *testing.T) {
	want := []string{`"share"`, `"id"`, `"text"`, `"url"`, `"ok"`}
	for _, shell := range []struct {
		name  string
		files []string
	}{
		{"android", []string{
			nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "Share.kt"),
			nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "SystemEvents.kt"),
		}},
		{"ios", []string{
			nativeFile("ios", "GrMob", "App", "Share.swift"),
			nativeFile("ios", "GrMob", "App", "SystemEvents.swift"),
		}},
		// The runtime reads text and url as data.text and data.url, which
		// this scan cannot see, so its request keys are pinned by
		// share_test.mjs instead; the reply keys are quoted.
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
		for _, lit := range want {
			if shell.name == "web" && (lit == `"text"` || lit == `"url"`) {
				continue
			}
			if !strings.Contains(src.String(), lit) {
				t.Errorf("%s shell never spells %s", shell.name, lit)
			}
		}
	}
}

// Each shell's system-event dispatcher must route "share" to its share
// object, and each native must attach the reporter that answers. The
// spelling test above would pass with the literals in Share.kt alone and the
// dispatcher arm missing.
func TestEveryShellDispatchesTheShareEvent(t *testing.T) {
	for _, c := range []struct{ file, arm string }{
		{nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "SystemEvents.kt"),
			`"share" -> Share.handle(data)`},
		{nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "SystemEvents.kt"),
			`Share.attach(appContext, runtime::hostEvent)`},
		{nativeFile("ios", "GrMob", "App", "SystemEvents.swift"),
			`case "share": Share.shared.handle(object)`},
		{nativeFile("ios", "GrMob", "App", "SystemEvents.swift"),
			`Share.shared.report = { name, payload in runtime.hostEvent(name, payload) }`},
		{nativeFile("wasm", "grmob-runtime.js"),
			`GrMob.share.handle(JSON.parse(payloadJSON))`},
	} {
		raw, err := os.ReadFile(c.file)
		if err != nil {
			t.Fatalf("reading %s: %v", c.file, err)
		}
		if !strings.Contains(string(raw), c.arm) {
			t.Errorf("%s has no line %q", c.file, c.arm)
		}
	}
}
