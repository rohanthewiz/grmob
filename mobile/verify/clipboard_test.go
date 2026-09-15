package verify

import (
	"os"
	"strings"
	"testing"
)

// The clipboard is spelled in four places that never compile together:
// core/clipboard.go, the Kotlin object, the Swift class and the browser
// runtime. Unlike lifecycle, a misspelling here does not merely drop an
// event — a read whose reply comes back under the wrong name or without its
// id leaves Go holding the caller's callback forever, so a Paste button
// silently stops working. This holds each shell to the event name, both
// commands and the three reply keys core decodes.
//
// Kotlin and Swift write the reply keys as quoted strings; the browser
// runtime quotes them too (a shorthand object literal would hide them from
// this scan), so one spelling check covers all three.
func TestClipboardEventSpellingsAgree(t *testing.T) {
	want := []string{`"clipboard"`, `"write"`, `"read"`, `"id"`, `"text"`, `"ok"`}
	for _, shell := range []struct {
		name  string
		files []string
	}{
		{"android", []string{
			nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "Clipboard.kt"),
			nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "SystemEvents.kt"),
		}},
		{"ios", []string{
			nativeFile("ios", "GrMob", "App", "Clipboard.swift"),
			nativeFile("ios", "GrMob", "App", "SystemEvents.swift"),
		}},
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
			if !strings.Contains(src.String(), lit) {
				t.Errorf("%s shell never spells %s", shell.name, lit)
			}
		}
	}
}

// Each shell's system-event dispatcher must route "clipboard" to its
// clipboard object. The spelling test above would pass with the literals in
// Clipboard.kt alone and the dispatcher arm missing, which is exactly the
// half a new shell author forgets.
func TestEveryShellDispatchesTheClipboardEvent(t *testing.T) {
	for _, c := range []struct{ file, arm string }{
		{nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "SystemEvents.kt"),
			`"clipboard" -> Clipboard.handle(data)`},
		{nativeFile("ios", "GrMob", "App", "SystemEvents.swift"),
			`case "clipboard": Clipboard.shared.handle(object)`},
		{nativeFile("wasm", "grmob-runtime.js"),
			`GrMob.clipboard.handle(JSON.parse(payloadJSON))`},
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
