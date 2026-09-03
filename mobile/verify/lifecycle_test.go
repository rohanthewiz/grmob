package verify

import (
	"os"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The lifecycle host event is spelled in four places that never compile
// together: core's consumer, the Kotlin observer, the Swift scene-phase
// mapping and the browser's visibility listener. A shell that reports
// "Lifecycle" or "foreground" would be dropped by core with a log line and
// nothing else — no compile error, no failing replay, just an app that
// never reconnects on resume. This holds every shell's spellings to core's
// exported states, and the event name to the one literal core reads.
//
// The web runtime cannot report "inactive" (the Page Visibility API has two
// states), so that spelling is required of the natives only.
func TestLifecycleEventSpellingsAgree(t *testing.T) {
	const event = `"lifecycle"`
	states := map[core.LifecycleState]bool{
		core.LifecycleActive:     true,
		core.LifecycleInactive:   true,
		core.LifecycleBackground: true,
	}
	for _, shell := range []struct {
		file     string
		inactive bool
	}{
		{nativeFile("android", "app", "src", "main", "java", "com", "grmob", "app", "AppLifecycle.kt"), true},
		{nativeFile("ios", "GrMob", "App", "AppLifecycle.swift"), true},
		{nativeFile("wasm", "grmob-runtime.js"), false},
	} {
		raw, err := os.ReadFile(shell.file)
		if err != nil {
			t.Fatalf("reading %s: %v", shell.file, err)
		}
		src := string(raw)
		if !strings.Contains(src, event) {
			t.Errorf("%s does not report the %s host event", shell.file, event)
		}
		for state, _ := range states {
			if state == core.LifecycleInactive && !shell.inactive {
				continue
			}
			if !strings.Contains(src, `"`+string(state)+`"`) {
				t.Errorf("%s never reports the %q state", shell.file, state)
			}
		}
	}
}
