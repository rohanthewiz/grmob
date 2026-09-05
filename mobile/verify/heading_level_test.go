package verify

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// core.Style.AccessibilityHeadingLevel, held against both native renderers.
//
// The field is the asymmetric case this package exists for. SwiftUI has
// accessibilityHeading, whose AccessibilityHeadingLevel is the same 1-6 idea,
// so the tier survives all the way to VoiceOver's rotor. Compose's heading()
// takes no argument and the semantics package has no level property at all, so
// there is nothing on Android for the field to become.
//
// "Nothing to become" and "nobody read the key" render identically on device,
// which is the whole reason role_test.go spells out the roles each native
// drops. The two checks below are that rule applied to a field instead of to a
// vocabulary: one pins that Swift actually carries the level through, the
// other pins that Kotlin's silence is written down as a limitation rather than
// left as an omission.

// The Swift half: parsed, mapped, and reaching the SwiftUI modifier. All three
// links are checked because breaking any one of them leaves a renderer that
// compiles and silently flattens every heading on the platform.
func TestSwiftCarriesTheHeadingLevelThrough(t *testing.T) {
	src := readNative(t, swiftStyle)
	for _, pin := range []struct{ expr, why string }{
		{`int("AccessibilityHeadingLevel")`,
			"the JSON key. An unread key is not a type error in Swift — the renderer would " +
				"compile and drop every tier"},
		{"private func grMobHeadingLevel(", "the mapping from Go's 1-6 onto AccessibilityHeadingLevel"},
		{"accessibilityHeading(grMobHeadingLevel(s))",
			"the modifier that puts the level on the view. Without this the mapping exists and " +
				"nothing invokes it"},
	} {
		if !strings.Contains(src, pin.expr) {
			t.Errorf("%s: %q not found — %s", swiftStyle, pin.expr, pin.why)
		}
	}

	// The role guard, which is what keeps a tier off a node that is not a
	// heading. It is ARIA's scoping and both web exporters apply it too; a
	// Swift mapping that skipped it would put a heading level on a table cell
	// whose Style happened to carry one.
	body := declSource(t, swiftStyle, "private func grMobHeadingLevel(")
	if !strings.Contains(body, `s.accessibilityRole == "`+string(core.RoleHeading)+`"`) {
		t.Errorf("%s: grMobHeadingLevel does not gate on the heading role — a level would "+
			"reach nodes that are not headings, which the two web targets refuse to emit",
			swiftStyle)
	}
	// Dropped, not clamped: `default: return .unspecified` is the arm that
	// makes a 0 and a 7 mean the same honest thing. See core.Style's field doc.
	if !strings.Contains(body, "default: return .unspecified") {
		t.Errorf("%s: grMobHeadingLevel has no catch-all returning .unspecified — an "+
			"out-of-range level must be dropped, not rounded into a tier nobody asked for",
			swiftStyle)
	}
}

// The Kotlin half, which is a documentation check because there is nothing
// else to check: Compose cannot express a heading level, so the renderer
// deliberately never parses the key.
//
// Pinned so the note cannot outlive the limitation. If Compose ever grows a
// level property, this test fails, and the person adding the mapping is
// handed the paragraph that has to be rewritten.
func TestKotlinWritesDownTheHeadingLevelGap(t *testing.T) {
	src := readNative(t, kotlinStyle)
	if !strings.Contains(src, "AccessibilityHeadingLevel is not read here, and cannot be") {
		t.Errorf("%s: the heading-level gap is not documented beside the role dispatch — a "+
			"field this file simply ignored is indistinguishable from one nobody had heard of",
			kotlinStyle)
	}
	// The other half of the same claim: it really does not read the key. An
	// arm that parsed it and threw it away would be the silent case the note
	// says this is not.
	if strings.Contains(src, `optString("AccessibilityHeadingLevel")`) {
		t.Errorf("%s: parses AccessibilityHeadingLevel while documenting that it cannot use "+
			"it — one of the two is now wrong", kotlinStyle)
	}
}
