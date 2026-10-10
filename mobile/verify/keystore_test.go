package verify

import (
	"os"
	"strings"
	"testing"
)

// The keystore is spelled in four places that never compile together:
// keystore/keystore.go, the Kotlin object, the Swift class and the browser
// runtime. As with the clipboard, a misspelling does not merely drop an event:
// a reply that comes back under the wrong name or without its id leaves Go
// holding the caller's callback forever, so an app waiting on its saved token
// at launch never gets past its splash screen. This holds each shell to the
// event name, the three commands, the request keys it reads, the reply keys
// core decodes, and the reserved "unavailable" reason.
//
// The browser refuses everything and so reads neither "key" nor "value" and
// writes neither "found" nor "value"; it is held to the subset it does speak.
// Kotlin and Swift write every key as a quoted string, and the browser
// runtime quotes its reply keys too (a shorthand object literal would hide
// them from this scan).
func TestKeystoreEventSpellingsAgree(t *testing.T) {
	native := []string{
		`"keystore"`, `"save"`, `"get"`, `"delete"`,
		`"id"`, `"command"`, `"key"`, `"value"`,
		`"ok"`, `"found"`, `"error"`,
	}
	web := []string{`"keystore"`, `"id"`, `"ok"`, `"error"`, `"unavailable"`}
	for _, shell := range []struct {
		name  string
		want  []string
		files []string
	}{
		{"android", native, []string{
			nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "Keystore.kt"),
			nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "SystemEvents.kt"),
		}},
		{"ios", native, []string{
			nativeFile("ios", "GrMob", "App", "Keystore.swift"),
			nativeFile("ios", "GrMob", "App", "SystemEvents.swift"),
		}},
		{"web", web, []string{nativeFile("wasm", "grmob-runtime.js")}},
	} {
		var src strings.Builder
		for _, f := range shell.files {
			raw, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("reading %s: %v", f, err)
			}
			src.Write(raw)
		}
		for _, lit := range shell.want {
			if !strings.Contains(src.String(), lit) {
				t.Errorf("%s shell never spells %s", shell.name, lit)
			}
		}
	}
}

// Each shell's system-event dispatcher must route "keystore" to its keystore
// object, and each native must attach that object's reporter. The spelling
// test above would pass with the literals in Keystore.kt alone and either of
// these missing — and a host that handles requests but never reports is the
// same hung callback as one that never handles them.
func TestEveryShellDispatchesTheKeystoreEvent(t *testing.T) {
	for _, c := range []struct {
		file  string
		lines []string
	}{
		{nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "SystemEvents.kt"), []string{
			`"keystore" -> Keystore.handle(data)`,
			`Keystore.attach(appContext, runtime::hostEvent)`,
		}},
		{nativeFile("ios", "GrMob", "App", "SystemEvents.swift"), []string{
			`case "keystore": Keystore.shared.handle(object)`,
			`Keystore.shared.report = { name, payload in runtime.hostEvent(name, payload) }`,
		}},
		{nativeFile("wasm", "grmob-runtime.js"), []string{
			`GrMob.keystore.handle(JSON.parse(payloadJSON))`,
		}},
	} {
		raw, err := os.ReadFile(c.file)
		if err != nil {
			t.Fatalf("reading %s: %v", c.file, err)
		}
		for _, line := range c.lines {
			if !strings.Contains(string(raw), line) {
				t.Errorf("%s is missing %q", c.file, line)
			}
		}
	}
}

// The two natives' storage policy, pinned where a refactor would quietly
// change it. iOS items are AfterFirstUnlockThisDeviceOnly (readable by a
// background launch, never restored onto other hardware), and Android's key
// is created inside AndroidKeyStore rather than in process memory. Each is a
// one-identifier change that compiles, runs and passes every functional
// check while undoing the reason the package exists.
func TestKeystoreStoragePolicyIsPinned(t *testing.T) {
	for _, c := range []struct{ file, want, why string }{
		{nativeFile("ios", "GrMob", "App", "Keystore.swift"),
			"kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly",
			"items must stay on this device and be readable after first unlock"},
		{nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "Keystore.kt"),
			`PROVIDER = "AndroidKeyStore"`,
			"the sealing key must live in the Android Keystore"},
	} {
		raw, err := os.ReadFile(c.file)
		if err != nil {
			t.Fatalf("reading %s: %v", c.file, err)
		}
		if !strings.Contains(string(raw), c.want) {
			t.Errorf("%s no longer says %s: %s", c.file, c.want, c.why)
		}
	}
}
