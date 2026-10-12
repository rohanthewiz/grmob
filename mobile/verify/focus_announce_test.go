package verify

import (
	"strings"
	"testing"
)

var kotlinFocusAnnouncer = nativeFile("android", "app", "src", "main", "java", "com", "grmob",
	"runtime", "GrMobFocusAnnouncer.kt")

// TalkBack stays on the control keyboard focus moves to on Compose (N-058).
//
// Compose 1.7.6 sends TYPE_VIEW_FOCUSED, the event TalkBack follows, only for
// a node it already had in its previous 100ms batch. Its batches hold only
// nodes on screen, so a Tab onto a control below the fold was never
// announced. On the emulator, 4.18's ☰ and ‹ Prev were silent, and on a long
// page (4.34, 4.16) most Tabs were. TalkBack then read prose no Tab visits,
// and Enter pressed a control other than the one just spoken. A long jump
// also lost its scroll: TalkBack moved its own focus as its node left the
// screen, which cut Compose's bring-into-view short and left the focused
// control off screen (4.9, 745px short).
//
// GrMobFocusAnnouncer watches each window's events, holds the newly focused
// node as a target, and finishes what Compose left undone:
//
//	target off screen, scroll settled ─► ScrollBy on the nearest scroller
//	TalkBack's focus on target        ─► done (nothing extra sent)
//	settled, TalkBack elsewhere       ─► TYPE_VIEW_FOCUSED, one retry
//
// After: Tab on 4.18 speaks Copy → Open navigation → ‹ Prev → Next ›, every
// Tab on 4.34 and 4.9 speaks its control, and a Modal opened from the
// keyboard speaks the control Compose focused in it.
func TestComposeKeepsTalkBackOnKeyboardFocus(t *testing.T) {
	for _, site := range []struct{ anchor, why string }{
		{"fun GrMobRoot(", "the Activity's window"},
		{"private fun GrMobModal(", "a Modal's own Dialog window, which is a separate Compose view"},
	} {
		if !strings.Contains(codeOf(t, kotlinRenderer, site.anchor), "AnnounceFocusMovesToScreenReader()") {
			t.Errorf("%s: %s no longer runs AnnounceFocusMovesToScreenReader — TalkBack loses "+
				"keyboard focus in %s", kotlinRenderer, strings.TrimSuffix(site.anchor, "("), site.why)
		}
	}

	code := codeIn(t, kotlinFocusAnnouncer)
	for _, pin := range []struct{ code, why string }{
		{"!view.isInTouchMode && accessibility?.isTouchExplorationEnabled == true",
			"it acts for a screen reader driven from a keyboard, and must not move TalkBack's focus under touch"},
		{"provider.findFocus(AccessibilityNodeInfo.FOCUS_ACCESSIBILITY) == info",
			"a move TalkBack already followed must not be announced again"},
		{"AccessibilityEvent.obtain(AccessibilityEvent.TYPE_VIEW_FOCUSED)",
			"TYPE_VIEW_FOCUSED is the event TalkBack follows input focus by"},
		{"event.setSource(view, virtualId)",
			"the event must name the focused node, or TalkBack has nothing to move to"},
		{"getOrNull(SemanticsActions.ScrollBy)",
			"an off-screen target is scrolled on through its scroller's own action, as ACTION_SHOW_ON_SCREEN does"},
	} {
		if !strings.Contains(code, pin.code) {
			t.Errorf("%s: lacks %q — %s", kotlinFocusAnnouncer, pin.code, pin.why)
		}
	}
}
