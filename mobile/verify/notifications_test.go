package verify

import (
	"os"
	"strings"
	"testing"
)

// Local notifications are spelled in four places that never compile together:
// core/notifications.go, the Kotlin object (plus MainActivity, which reports
// the tap), the Swift delegate and the browser runtime. A misspelled command
// is a post that never draws; a misspelled tap event is a banner that opens
// the app at its front door with the id thrown away. Neither logs anything a
// person would see, so this holds every shell to the three event names, the
// three commands and the payload keys — the sweep's included, whose reply a
// misspelling would leave a caller waiting on forever.
func TestNotificationEventSpellingsAgree(t *testing.T) {
	want := []string{`"notification"`, `"notification_tap"`, `"post"`, `"cancel"`,
		`"id"`, `"title"`, `"body"`, `"at"`,
		`"sweep"`, `"notification_swept"`, `"prefix"`, `"request"`, `"fired"`}
	for _, shell := range []struct {
		name  string
		files []string
	}{
		{"android", []string{
			nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "Notifications.kt"),
			kotlinSystemEvents,
			kotlinMainActivity,
		}},
		{"ios", []string{
			nativeFile("ios", "GrMob", "App", "Notifications.swift"),
			swiftSystemEvents,
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

// Each dispatcher routes "notification", and each shell wires the tap back:
// Android from MainActivity (the only component a tapped PendingIntent
// reaches), iOS by becoming the notification center's delegate during launch
// (a delegate set any later misses the tap that cold-launched the app, and
// with no delegate a post from a running app is never drawn at all).
func TestEveryShellDispatchesNotificationsAndReportsTaps(t *testing.T) {
	for _, c := range []struct{ file, want, why string }{
		{kotlinSystemEvents, `"notification" -> Notifications.handle(data)`,
			"the dispatch arm; without it every post is dropped"},
		{kotlinMainActivity, `Notifications.tapId(`,
			"the tap report; without it a tapped banner opens the app and says nothing"},
		{swiftSystemEvents, `case "notification": Notifications.shared.handle(object)`,
			"the dispatch arm; without it every post is dropped"},
		{swiftSystemEvents, `Notifications.shared.attach()`,
			"the delegate, set during launch from GrMobApp.init"},
		{nativeFile("ios", "GrMob", "App", "Notifications.swift"),
			`UNUserNotificationCenter.current().delegate = self`,
			"becoming the delegate is what attach() is for"},
		{nativeFile("wasm", "grmob-runtime.js"), `GrMob.notifications.handle(JSON.parse(payloadJSON))`,
			"the browser's dispatch arm"},
	} {
		raw, err := os.ReadFile(c.file)
		if err != nil {
			t.Fatalf("reading %s: %v", c.file, err)
		}
		if !strings.Contains(string(raw), c.want) {
			t.Errorf("%s: %q not found — %s", c.file, c.want, c.why)
		}
	}
}

// A sweep cancels what is still to come and leaves banners already shown
// (core.SweepNotifications). Each shell's sweep declaration is read with its
// literals and without its comments, so a comment explaining the old call
// does not trip the check and a call put back does.
func TestSweepsLeaveShownBanners(t *testing.T) {
	for _, c := range []struct {
		file, anchor string
		must, never  []string
	}{
		{nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "Notifications.kt"),
			"private fun sweep(",
			[]string{"alarms?.cancel(alarmIntent(context, id"},
			[]string{"manager.cancel(", "activeNotifications"}},
		{nativeFile("ios", "GrMob", "App", "Notifications.swift"),
			"private func sweep(",
			[]string{"removePendingNotificationRequests"},
			[]string{"removeDeliveredNotifications", "getDeliveredNotifications"}},
		{nativeFile("wasm", "grmob-runtime.js"),
			"function sweep(",
			[]string{"clearTimer(id)"},
			[]string{".close()"}},
	} {
		src := valuesOf(t, c.file, c.anchor)
		// The runtime's functions are nested in one closure, so the cut runs
		// past the sweep into its neighbours (tapped closes a banner, rightly);
		// end it at the next function.
		if next := strings.Index(src[1:], "function "); strings.HasSuffix(c.file, ".js") && next >= 0 {
			src = src[:next+1]
		}
		for _, want := range c.must {
			if !strings.Contains(src, want) {
				t.Errorf("%s: the sweep no longer cancels pending posts (%q)", c.file, want)
			}
		}
		for _, bad := range c.never {
			if strings.Contains(src, bad) {
				t.Errorf("%s: the sweep takes down shown banners again (%q)", c.file, bad)
			}
		}
	}
}

// Android's exact-alarm re-check answers from the application context and is
// gated on the host-event channel, not on an Activity being attached: the
// grant is a property of the app, and the answer only needs a Go to hear it.
func TestExactAlarmRecheckNeedsNoActivity(t *testing.T) {
	file := nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "Permissions.kt")
	src := valuesOf(t, file, "fun recheckExactAlarms(")
	if strings.Contains(src, "activity") {
		t.Errorf("%s: recheckExactAlarms reads the Activity again", file)
	}
	for _, want := range []string{"appContext", "report == null", "exactAlarmStatus(context)"} {
		if !strings.Contains(src, want) {
			t.Errorf("%s: recheckExactAlarms is missing %q", file, want)
		}
	}
}
