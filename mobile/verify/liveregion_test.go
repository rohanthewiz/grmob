package verify

import (
	"strings"
	"testing"
)

// A live-region Text is heard when its text changes on Compose (N-108).
//
// core.RoleStatus, RoleAlert and RoleLog map to Compose's liveRegion. TalkBack
// announced a live region whose content description changed, and said nothing
// when only a Text's text did. On the emulator a RoleStatus Text counting
// every 5s was silent for 32s, while the same Text with its words as its
// label announced every change. comps.Wizard's "Step N of M" line is such a
// Text, and on Android it is the only announcement of a step change (Compose
// cannot move TalkBack's focus to the new title). GrMobText now gives a
// live-region Text with no label of its own its content as its description.
// With that, the Wizard's line was heard at every step ("Step 3 of 3",
// "Step 1 of 3", "Step 2 of 3, optional").
func TestComposeLiveRegionTextAnnouncesItsChanges(t *testing.T) {
	code := codeIn(t, kotlinRenderer)
	for _, pin := range []struct{ code, why string }{
		{"node.style?.accessibilityRole in grMobLiveRoles &&",
			"a Text's live role must be read before its semantics are built"},
		{".then(if (live) Modifier.semantics { contentDescription = content } else Modifier)",
			"a live-region Text must carry its words as its description, or TalkBack does not hear them change"},
	} {
		if !strings.Contains(code, pin.code) {
			t.Errorf("%s: lacks %q — %s", kotlinRenderer, pin.code, pin.why)
		}
	}
	values := valuesIn(t, kotlinRenderer)
	if !strings.Contains(values, `private val grMobLiveRoles = setOf("status", "alert", "log")`) {
		t.Errorf("%s: grMobLiveRoles no longer names the three live roles roleSemantics maps", kotlinRenderer)
	}
}
