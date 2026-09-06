package verify

import (
	"regexp"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/permission"
)

// The "permission" system event on both native shells, and the two things
// only these platforms can get wrong.
//
// The subject is this package's usual one: a Go call that reaches both web
// targets for free and does nothing at all on a phone unless the shell has an
// arm for it. It is worse here than for a sensor. permission.Check sends one
// event and then *waits* — the answer is a host event that only the shell can
// produce — so a missing arm leaves a screen on permission.Unknown forever,
// which is the state its doc says to draw a spinner for. Nothing errors and
// nothing logs, and from Go's side a sent event and a heard one look
// identical.
//
// Source text rather than behaviour, for the reason every check in this
// package reads source: neither native runs under `go test ./...`.

var (
	swiftPermissions = nativeFile("ios", "GrMob", "App", "Permissions.swift")
	// The xcodegen spec, not the Info.plist it produces: the plist is
	// generated and gitignored (ios/.gitignore), so on a fresh clone it does
	// not exist and a check reading it would fail for the wrong reason. This
	// file is the tracked source of truth, and it is the one a fix has to
	// touch anyway — editing the generated plist alone is undone by the next
	// `xcodegen generate`.
	iosProjectSpec = nativeFile("ios", "project.yml")

	kotlinPermissions = nativeFile("android", "app", "src", "main", "java", "com",
		"grmob", "app", "Permissions.kt")
	kotlinMainActivity = nativeFile("android", "app", "src", "main", "java", "com",
		"grmob", "app", "MainActivity.kt")
	androidManifest = nativeFile("android", "app", "src", "main", "AndroidManifest.xml")
)

// Both shells must dispatch "permission" and must attach the return channel.
//
// The two are separate failures with the same symptom. A shell that does not
// dispatch never hears the question; a shell that dispatches without attaching
// hears it, answers it, and drops the answer on the floor — and Go, which is
// waiting either way, cannot tell them apart.
func TestBothShellsDispatchThePermissionEvent(t *testing.T) {
	for _, pin := range []struct{ file, dispatch, attach, attachIn string }{
		{
			file:     swiftSystemEvents,
			dispatch: `case "permission": Permissions.shared.handle(object)`,
			attach:   "Permissions.shared.report =",
			attachIn: swiftSystemEvents,
		},
		{
			file:     kotlinSystemEvents,
			dispatch: `"permission" -> Permissions.handle(data)`,
			// Android attaches from the Activity rather than from
			// SystemEvents, because registerForActivityResult is an Activity
			// API that must run in onCreate. That is the one structural
			// difference between the two shells here, so it is pinned where it
			// actually lives.
			attach:   "Permissions.attach(this, runtime::hostEvent)",
			attachIn: kotlinMainActivity,
		},
	} {
		if src := readNative(t, pin.file); !strings.Contains(src, pin.dispatch) {
			t.Errorf("%s: no arm for the \"permission\" event — permission.Check is dropped "+
				"here and every screen gated on one sits on Unknown forever", pin.file)
		}
		if src := readNative(t, pin.attachIn); !strings.Contains(src, pin.attach) {
			t.Errorf("%s: the permission host's report channel is never attached, so the "+
				"platform's answer has nowhere to go", pin.attachIn)
		}
	}
}

// Both hosts must answer both commands.
//
// They are genuinely different operations and a host that collapsed them would
// be wrong in whichever direction it collapsed: a check that prompts puts the
// OS dialog up as a side effect of a screen mounting, which is how an app
// earns a permanent refusal, and a request that only checks leaves a button
// that does nothing.
func TestBothHostsAnswerCheckAndRequest(t *testing.T) {
	for _, pin := range []struct{ file, check, request string }{
		{swiftPermissions, `case "check": check(kind)`, `case "request": request(kind)`},
		{kotlinPermissions, `"check" -> send(kind, status(kind))`, `"request" -> request(kind)`},
	} {
		src := readNative(t, pin.file)
		if !strings.Contains(src, pin.check) {
			t.Errorf("%s: no arm for the check command — a screen can never read a "+
				"permission without prompting for it", pin.file)
		}
		if !strings.Contains(src, pin.request) {
			t.Errorf("%s: no arm for the request command — nothing here can ever "+
				"put the platform's dialog on screen", pin.file)
		}
	}
}

// Every permission.Permission must have an arm in both hosts.
//
// Same coverage argument as core.Roles() one file over, and a sharper failure:
// an unknown kind is dropped on purpose (that is the contract), so a
// permission nobody taught a shell about is indistinguishable — on the device
// and at the call site — from one the user has not answered yet.
func TestBothHostsCoverEveryPermission(t *testing.T) {
	for _, pin := range []struct{ file, arm string }{
		// Both hosts key their tables on the Go constant's own string, so the
		// arm is the quoted value in each language's spelling.
		{swiftPermissions, `case %q`},
		{kotlinPermissions, `%q to `},
	} {
		src := readNative(t, pin.file)
		for _, p := range permission.Permissions() {
			if !strings.Contains(src, sprintfArm(pin.arm, string(p))) {
				t.Errorf("%s: no arm for permission.Permission %q — the kind is dropped, "+
					"and a dropped kind reads exactly like an unanswered one",
					pin.file, p)
			}
		}
	}
}

// Every status a host reports must be one permission.Statuses() names.
//
// The wire carries strings, so a typo is not a compile error in either
// language: it crosses the bridge, fails permission.reportable, and is
// dropped in Go with the record left where it was. The screen keeps showing
// whatever it was showing, which is the quietest failure in the whole channel.
//
// Only the literals passed to each host's send are read, so prose and
// framework enum names cannot produce a false pass.
func TestBothHostsReportOnlyDeclaredStatuses(t *testing.T) {
	known := map[string]bool{}
	for _, s := range permission.Statuses() {
		known[string(s)] = true
	}

	for _, pin := range []struct {
		file  string
		calls []string
	}{
		{swiftPermissions, []string{`send(kind, "`, `send("location", `, `return "`}},
		{kotlinPermissions, []string{`send(kind, "`, `return "`}},
	} {
		src := readNative(t, pin.file)
		for _, literal := range quotedAfter(src, pin.calls) {
			// The kind arguments are quoted too and are not statuses; a
			// permission name is never a status name, so membership in either
			// declared set is the test.
			if known[literal] || isPermissionName(literal) {
				continue
			}
			t.Errorf("%s: reports the status %q, which permission.Statuses() does not "+
				"declare — Go drops it and the record never moves", pin.file, literal)
		}
	}
}

// iOS crashes rather than prompts when a usage-description key is missing, so
// the Info.plist entry is part of the implementation and not documentation.
//
// It is also the one half of this feature that no amount of Swift can supply
// and no test on the device would reach before the crash: the failure happens
// at the exact moment the feature is first exercised, which is the moment
// after everything looked fine.
func TestTheIOSShellDeclaresAUsageDescriptionPerPermission(t *testing.T) {
	src := readNative(t, iosProjectSpec)
	for _, key := range []string{
		"NSCameraUsageDescription",
		"NSMicrophoneUsageDescription",
		"NSLocationWhenInUseUsageDescription",
		"NSPhotoLibraryUsageDescription",
	} {
		// A key with a non-empty value, matched as a whole property rather
		// than as a substring: an empty description is as fatal as a missing
		// one, and a plain Contains would also be satisfied by the key's name
		// appearing in a comment or inside a longer misspelling.
		declared := regexp.MustCompile(`(?m)^\s*` + key + `:\s*\S`)
		if !declared.MatchString(src) {
			t.Errorf("%s: no %s with a description — iOS terminates the app when the "+
				"prompt this key describes would appear", iosProjectSpec, key)
		}
	}
}

// An Android permission missing from the manifest is never granted and never
// prompts: requestPermissions returns DENIED immediately with nothing on
// screen. The shell's own Permissions.declared reports that case as
// "unavailable", which is honest — and useless if the entries are simply
// absent, because then every permission is unavailable.
func TestTheAndroidShellDeclaresEveryRuntimePermission(t *testing.T) {
	src := readNative(t, androidManifest)
	for _, name := range []string{
		// Matched as the attribute rather than as a bare substring, so a
		// permission merely *named in a comment* — which several are, right
		// above the declarations — cannot satisfy the check.
		"android.permission.CAMERA",
		"android.permission.RECORD_AUDIO",
		"android.permission.ACCESS_COARSE_LOCATION",
		// The storage pair: Android 13 split the old read permission and stops
		// granting it, so both spellings have to be there for the app to work
		// on either side of that line.
		"android.permission.READ_MEDIA_IMAGES",
		"android.permission.READ_EXTERNAL_STORAGE",
	} {
		if !strings.Contains(src, `android:name="`+name+`"`) {
			t.Errorf("%s: does not declare %s — the request is auto-denied with no "+
				"dialog, on every device", androidManifest, name)
		}
	}
}

// sprintfArm is fmt.Sprintf with one verb, kept local so this file's intent
// (a per-language arm spelling) reads at the call site.
func sprintfArm(pattern, value string) string {
	return strings.Replace(pattern, "%q", `"`+value+`"`, 1)
}

func isPermissionName(s string) bool {
	for _, p := range permission.Permissions() {
		if string(p) == s {
			return true
		}
	}
	return false
}

// quotedAfter collects the string literal that follows each occurrence of each
// prefix. Comments are stripped first so a status named in prose — and both
// files name several — cannot be mistaken for one that is reported.
func quotedAfter(src string, prefixes []string) []string {
	src = stripLineComments(src)

	var out []string
	for _, prefix := range prefixes {
		rest := src
		for {
			at := strings.Index(rest, prefix)
			if at < 0 {
				break
			}
			rest = rest[at+len(prefix):]
			// The prefix may or may not include the opening quote.
			tail := rest
			if !strings.HasSuffix(prefix, `"`) {
				q := strings.Index(tail, `"`)
				// Only a literal on the same line: `send(kind, map(...))`
				// passes an expression, not a string, and must not swallow the
				// next line's quote.
				if q < 0 || strings.Contains(tail[:q], "\n") {
					continue
				}
				tail = tail[q+1:]
			}
			end := strings.Index(tail, `"`)
			if end < 0 {
				break
			}
			out = append(out, tail[:end])
		}
	}
	return out
}

// The Android asked-before flag has to reach disk, and the request path has to
// stay open to a "denied" it may have produced.
//
// The two halves are one fact and each is useless without the other, which is
// why they are pinned together:
//
//	persisted    without it, a permanently refused permission reads as
//	             "prompt" on every cold start, so the first thing the user
//	             sees is a button offering to ask and pressing it does nothing
//	asked in the request path only, never in status()
//	launched     without it, Android 11+'s auto-reset — which revokes an
//	             unused app's permissions and clears don't-ask-again with them
//	             — leaves the stale flag saying "denied" and a short-circuiting
//	             request path locking the user out of a permission the system
//	             has just handed back
//
// Source text, because neither is reachable from Go and the Android build's
// only check is that the file compiles. The reasoning is in Permissions.kt's
// class doc under "The flag is on disk" and "Auto-reset".
func TestTheAndroidAskedFlagSurvivesARestart(t *testing.T) {
	src := readNative(t, kotlinPermissions)

	for _, pin := range []struct{ needle, why string }{
		{"getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)",
			"there is no preferences store at all, so the asked-before flag has " +
				"nowhere to live past the process that learned it"},
		{"getStringSet(ASKED_KEY, null)",
			"the store is opened and never read, which is the same cold start as " +
				"having no store: a permanent refusal reports as \"prompt\" and the " +
				"button offering to ask does nothing"},
		{"putStringSet(ASKED_KEY, asked.toSet())",
			"the asked-before flag is never written, so nothing survives the " +
				"process it was learned in"},
	} {
		if !strings.Contains(src, pin.needle) {
			t.Errorf("%s: %s", kotlinPermissions, pin.why)
		}
	}

	// The short-circuit, matched as the whole condition rather than as the
	// word "granted": the failure being guarded against is one arm too many in
	// this test, and a substring check would pass on the version that has it.
	openToDenied := regexp.MustCompile(
		`if \(current == "granted" \|\| current == "unavailable"\) \{`)
	if !openToDenied.MatchString(src) {
		t.Errorf("%s: the request path does not short-circuit on exactly granted and "+
			"unavailable — a \"denied\" that came from the persisted flag has to reach "+
			"the launcher, because auto-reset can make that flag a lie and the "+
			"launcher is the only thing that can say so", kotlinPermissions)
	}

	// The write happens on the request path and nowhere else. A check runs
	// inside a render pass (hooks.UsePermission calls it on mount, and
	// UsePermissionLive again on every resume), and a render pass that touches
	// the disk is a render pass that can jank.
	if strings.Contains(src, `"check" -> `) {
		checkArm := src[strings.Index(src, `"check" -> `):]
		if end := strings.Index(checkArm, "\n"); end > 0 {
			checkArm = checkArm[:end]
		}
		if strings.Contains(checkArm, "rememberAsked") {
			t.Errorf("%s: the check command records the asked-before flag — a check "+
				"is what a render pass runs, and it must not write to disk",
				kotlinPermissions)
		}
	}
}
