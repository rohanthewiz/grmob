package main

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/permission"
)

// The browser half of Go's permission package, pinned from Go.
//
// permission_test.mjs proves this module *behaves* — that a query becomes the
// right descriptor, that a rejected getUserMedia is told apart from an absent
// camera, that a stream is closed again — but it runs only under run.sh, which
// needs Node and which a human has to remember. This is the half
// `go test ./...` reaches, and what it holds is the contract rather than the
// implementation: the two Go vocabularies, and the wire names on both sides.
//
// Every drift here is silent in exactly the same way. A kind the host has no
// entry for is dropped, and the screen sits on permission.Unknown drawing the
// spinner that state is for; a status the host spells wrongly is dropped by
// permission.reportable, and the record never moves.

// Every permission.Permission needs a descriptor-table entry, including the
// one whose entry is deliberately null.
//
// The null entry is the case worth stating: `storage` has no browser
// permission at all, and the difference between "absent from the table" and
// "present and null" is the difference between a screen waiting forever and a
// screen told this platform cannot do it. The host answers the second and
// drops the first, so the entry has to be there.
func TestTheBrowserHostCoversEveryPermission(t *testing.T) {
	src := runtimeSource(t)
	for _, p := range permission.Permissions() {
		if !strings.Contains(src, string(p)+": ") {
			t.Errorf("grmob-runtime.js: DESCRIPTORS has no entry for permission.Permission "+
				"%q — the kind is dropped, and a dropped kind is indistinguishable from "+
				"one the user has not answered", p)
		}
	}
	if !strings.Contains(src, "storage: null") {
		t.Error("grmob-runtime.js: storage must map to null rather than be absent — a null " +
			"entry is answered \"unavailable\", an absent one is silently dropped")
	}
}

// Every status the host can report must be one permission.Statuses() declares.
//
// The three the Permissions API produces are forwarded verbatim rather than
// mapped, which is the whole payoff of Go spelling Status the way the browser
// does — so what is checked is the places the host writes a word itself, and
// that the fourth, which the browser has no word for, is spelled the way Go
// does.
func TestTheBrowserHostReportsOnlyDeclaredStatuses(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []struct{ expr, why string }{
		{`report(kind, "granted")`,
			"a resolved getUserMedia. The query path forwards the browser's own state " +
				"untouched, so this is one of the few places the host writes a word itself"},
		{`? "unavailable" : "denied"`,
			"the split that makes a missing camera a different answer from a refused " +
				"prompt — only one of the two has a fix in the browser's settings"},
		{`report("location", "denied")`,
			"PERMISSION_DENIED, which is the only geolocation error that means a refusal"},
		{`report(kind, "unavailable")`,
			"a page with no media provider at all"},
		{`Promise.resolve("unavailable")`,
			"a descriptor this browser does not have. query() throws for an unknown name " +
				"rather than resolving, and Firefox has no camera descriptor, so this is " +
				"a common path and not an edge case"},
	} {
		if !strings.Contains(src, want.expr) {
			t.Errorf("grmob-runtime.js: %q not found — %s", want.expr, want.why)
		}
	}

	if strings.Contains(src, `report(kind, "prompt")`) {
		t.Error("grmob-runtime.js: the host writes \"prompt\" itself somewhere — that word " +
			"should only ever reach Go as the Permissions API's own answer, forwarded")
	}
}

// The wire, on both sides of it. The names are string literals in four places
// (Go, this runtime, and two native shells) and nothing but a check like this
// connects them.
func TestTheBrowserHostSpeaksThePermissionWire(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []struct{ expr, why string }{
		{`if (name === "permission") {`,
			`the system-event arm. Without it permission.Check is dropped at the page ` +
				`boundary, exactly as it would be on a native shell with no case`},
		{`host.HostEvent("permission", JSON.stringify({ kind, status }))`,
			"the answer's shape — both keys are required, and Go drops a payload missing " +
				"either rather than guessing at it"},
		{`case "check": check(cmd.kind); break;`,
			"the check command, which must never prompt: it is what a screen calls on mount"},
		{`case "request": request(cmd.kind); break;`,
			"the request command, which is the only one that puts a dialog up"},
	} {
		if !strings.Contains(src, want.expr) {
			t.Errorf("grmob-runtime.js: %q not found — %s", want.expr, want.why)
		}
	}
}

// A request must actually prompt, which on this platform means calling the
// feature rather than asking about it.
//
// There is no requestPermission API in a browser: camera, microphone and
// location are obtained by using them, and the prompt is a side effect. A host
// that quietly answered a request with a query would pass every shape check
// above and leave a button that does nothing at all.
func TestABrowserRequestCallsTheFeatureRatherThanQueryingIt(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []struct{ expr, why string }{
		{"md.getUserMedia(constraints)",
			"the camera and microphone prompt"},
		{"stream.getTracks().forEach((t) => t.stop())",
			"closing the device again. A resolved getUserMedia is a live capture device, " +
				"and a page that only wanted an answer must not leave the browser's " +
				"recording indicator lit"},
		{"navigator.geolocation.getCurrentPosition(",
			"the location prompt"},
	} {
		if !strings.Contains(src, want.expr) {
			t.Errorf("grmob-runtime.js: %q not found — %s", want.expr, want.why)
		}
	}
}
