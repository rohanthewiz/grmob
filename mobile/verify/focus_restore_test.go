package verify

import (
	"strings"
	"testing"
)

// Compose 1.7 clears the View's focus when the focused node leaves the
// composition, and every GrMob navigation (Push, Replace, Pop) swaps the
// subtree that held it. Under TalkBack the framework's own recovery, which
// normally puts the next Tab on the first control, never ran: on a Galaxy Z
// Fold6 Tab stayed dead after "Next ›" or a deep link until TalkBack was
// turned off. MainActivity recovers the focus itself; see
// MainActivity.restoreFocusForNavigation.
//
//	key ─► MainActivity.dispatchKeyEvent
//	         ├─ runtime.handleKeyEvent        a page-global chord wins first
//	         ├─ restoreFocusForNavigation     Tab/arrow with no focused View:
//	         │                                restoreDefaultFocus, key spent
//	         └─ super.dispatchKeyEvent        the focused View's own handling
func TestAndroidRecoversFocusLostToANavigation(t *testing.T) {
	activity := nativeFile("android", "app", "src", "main", "java", "com", "grmob",
		"app", "MainActivity.kt")
	dispatch := codeOf(t, activity, "override fun dispatchKeyEvent(")
	chord := strings.Index(dispatch, "runtime?.handleKeyEvent(event)")
	restore := strings.Index(dispatch, "if (restoreFocusForNavigation(event)) return true")
	super := strings.Index(dispatch, "super.dispatchKeyEvent(event)")
	switch {
	case restore < 0:
		t.Errorf("%s: dispatchKeyEvent no longer asks restoreFocusForNavigation — Tab "+
			"is dead under TalkBack after any navigation", activity)
	case chord < 0 || chord > restore:
		t.Errorf("%s: the focus recovery runs before the chord lookup — a chord must "+
			"be answered wherever focus is, including nowhere", activity)
	case super < 0 || super < restore:
		t.Errorf("%s: the focus recovery runs after the window's dispatch — by then "+
			"the Tab has gone unhandled", activity)
	}

	body := codeOf(t, activity, "private fun restoreFocusForNavigation(")
	for _, pin := range []struct{ expr, why string }{
		{"window.decorView.findFocus() != null) return false",
			"only a window with no focused View is touched; a focused one navigates as before"},
		{"return window.decorView.restoreDefaultFocus()",
			"the framework's own recovery, and its answer is whether the key was spent"},
		{"event.action != KeyEvent.ACTION_DOWN) return false",
			"only a press: the release of the Tab that restored focus must not move it again"},
	} {
		if !strings.Contains(body, pin.expr) {
			t.Errorf("%s: restoreFocusForNavigation has lost %q — %s", activity, pin.expr, pin.why)
		}
	}
}
