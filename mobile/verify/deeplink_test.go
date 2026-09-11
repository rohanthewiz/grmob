package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The inbound deep link, which is spelled in five places that never compile
// together: core's helper, two manifests that are data rather than code, and
// two shells.
//
// # Why it needs a census at all
//
// The failure is silent in both directions and in a way no test on either
// side of the bridge can see. A shell that reports the URL under a name core
// does not know gets a log line and nothing else. A shell whose manifest
// forgets the scheme is worse: nothing arrives, because the OS never offers
// the app the link in the first place — `adb shell am start -d` answers
// "Error: Activity not started, unable to resolve Intent", and a simulator
// simply does nothing. Neither is a crash and neither reaches Go.
//
// The manifest half is the part that cannot be checked any other way. It is
// XML and plist, read by the OS at install time, and no Kotlin or Swift
// compiler has an opinion about whether the entry is there.
//
// # What is pinned, host by host
//
// core   the event name, as a literal, because the string is the contract
// both   the URL forwarded verbatim and unparsed — a shell that split the path
//
//	would be a shell that only serves the app it was written beside
//
// kt     the manifest's VIEW filter and scheme, plus singleTop: without the
//
//	launch mode a link to a running app starts a second Activity and
//	onNewIntent never fires, so a warm link is silently a cold one
//
// swift  CFBundleURLTypes in the generated Info.plist's source, plus
//
//	.onOpenURL, which covers the cold and warm cases in one callback
func TestBothShellsForwardAnInboundURL(t *testing.T) {
	const event = `"deeplink"`
	if src := valuesIn(t, filepath.Join("..", "..", "core", "deeplink.go")); !strings.Contains(src, event) {
		t.Errorf("core does not name the deeplink host event (%s), so there is no "+
			"contract for the shells to write against", event)
	}

	for _, pin := range []struct{ file, forward, verbatim string }{
		{
			file:    nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "MainActivity.kt"),
			forward: `runtime?.hostEvent("deeplink"`,
			// The Uri straight to a string. A shell that reached for
			// intent.data?.path or lastPathSegment would be parsing.
			verbatim: "intent.data?.toString()",
		},
		{
			file:     nativeFile("ios", "GrMob", "App", "GrMobApp.swift"),
			forward:  `runtime.hostEvent("deeplink", json)`,
			verbatim: "url.absoluteString",
		},
	} {
		src := valuesIn(t, pin.file)
		if !strings.Contains(src, pin.forward) {
			t.Errorf("%s: an inbound URL is never reported to Go (%s) — the OS hands "+
				"this shell the link and the shell drops it", pin.file, pin.forward)
		}
		if !strings.Contains(src, pin.verbatim) {
			t.Errorf("%s: the URL is not forwarded whole (%s) — a shell that parses it "+
				"is a shell that knows one app's vocabulary, which is the thing "+
				"core/deeplink.go keeps out of here", pin.file, pin.verbatim)
		}
	}
}

// The two manifests, which are the half that decides whether a link is offered
// to the app at all.
func TestBothShellsDeclareTheURLScheme(t *testing.T) {
	for _, pin := range []struct {
		file  string
		needs []string
		why   string
	}{
		{
			file: filepath.Join("..", "..", "android", "app", "src", "main", "AndroidManifest.xml"),
			needs: []string{
				"android.intent.action.VIEW",
				"android.intent.category.BROWSABLE",
				`android:scheme="grmob"`,
				`android:launchMode="singleTop"`,
			},
			why: "without the filter the system will not resolve a grmob:// intent to " +
				"this activity; without BROWSABLE a link in another app cannot open " +
				"it; without singleTop a link arriving at a running app starts a " +
				"second Activity and onNewIntent never fires",
		},
		{
			file: filepath.Join("..", "..", "ios", "project.yml"),
			needs: []string{
				"CFBundleURLTypes",
				"CFBundleURLSchemes: [grmob]",
			},
			why: "project.yml is the source of truth for the generated Info.plist " +
				"(GrMobApp.xcodeproj is untracked), so this is where the scheme has " +
				"to be declared; without it .onOpenURL never fires",
		},
	} {
		raw, err := os.ReadFile(pin.file)
		if err != nil {
			t.Errorf("cannot read %s: %v", pin.file, err)
			continue
		}
		src := string(raw)
		for _, need := range pin.needs {
			if !strings.Contains(src, need) {
				t.Errorf("%s: missing %q — %s", pin.file, need, pin.why)
			}
		}
	}
}
