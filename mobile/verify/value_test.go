package verify

import (
	"strings"
	"testing"
)

// core.Style.AccessibilityValue, held against both native renderers.
//
// The fourth accessibility state field, and the one where the two platforms
// disagree about *which half* they can say — the same shape the tab pair has in
// core.Role, one layer down:
//
//	web ×2     aria-valuenow / -valuemin / -valuemax / -valuetext, scoped to
//	           the one range role core.Role carries
//	Compose    the numbers, through progressBarRangeInfo, which TalkBack turns
//	           into a percentage it localizes itself; and the words, through
//	           stateDescription
//	SwiftUI    the words alone, through accessibilityValue. There is no numeric
//	           accessibility value on this platform at all.
//
// So Compose is pinned like a mapping — parsed, dispatched, reaching a platform
// primitive, invoked from the semantics lambda — and SwiftUI is pinned in both
// directions: the words reaching a modifier, and the numbers reaching nothing.
// That second half is the shape idref_test.go and nesting_level_test.go cover,
// and it matters for the same reason: a renderer that quietly turned "45" into
// the English string "45 percent" would look, on device, exactly like one that
// had a real mapping — and would be inventing words for every app in every
// locale, which is the move GrMobStyle.swift already turns down for
// AccessibilityExpanded.

// --- Both: the field is read at all ----------------------------------------

func TestBothNativeParsersReadTheValueField(t *testing.T) {
	for _, pin := range []struct{ file, expr, why string }{
		{kotlinStyle, `optJSONObject("AccessibilityValue")`, "the parse"},
		{kotlinStyle, "val accessibilityValue: ValueRange,", "the field it parses into"},
		{swiftStyle, `obj["AccessibilityValue"] as? [String: Any]`, "the parse"},
		{swiftStyle, "var accessibilityValue: ValueRange", "the field it parses into"},
	} {
		if src := readNative(t, pin.file); !strings.Contains(src, pin.expr) {
			t.Errorf("%s: %q not found — %s. core.ValueRange reaches both web targets, "+
				"and a key no renderer parses is indistinguishable from one nobody "+
				"had heard of", pin.file, pin.expr, pin.why)
		}
	}
}

// The numbers cross as strings and are parsed on the far side, and that is not
// an encoding detail: 0 is a bar at the start of an upload and is also the zero
// value of a Go float, and core.Style merges on "non-zero wins" — so a stated 0
// would be dropped by every merge in the chain. See core.ValueRange.
//
// The Kotlin side has to keep that distinction after parsing, or it reintroduces
// the bug one layer down: a Float with a 0f default cannot tell an indeterminate
// bar from one at the start.
func TestKotlinKeepsAnUnstatedNumberUnstated(t *testing.T) {
	src := readNative(t, kotlinStyle)
	for _, pin := range []struct{ expr, why string }{
		{"val now: Float?,", "a nullable position, because 0 is a real one"},
		// The whole line, terminator included. A prefix match would still
		// stand over `toFloatOrNull() ?: 0f`, which is exactly the fix
		// somebody reaches for to make the type non-null — and which
		// reintroduces the bug the nullable field exists to prevent.
		{"fun num(name: String): Float? = obj.optString(name).toFloatOrNull()\n",
			"parsing to null rather than to a 0 default — a value that failed to " +
				"parse must not read as a bar at the start"},
		{"ProgressBarRangeInfo.Indeterminate",
			"ARIA's indeterminate bar, which is a range with bounds and no position " +
				"and is a real state rather than a missing one"},
	} {
		if !strings.Contains(src, pin.expr) {
			t.Errorf("%s: %q not found — %s", kotlinStyle, pin.expr, pin.why)
		}
	}
}

// --- Compose: the mapping, and that something calls it ---------------------

// Both links break silently and separately: a mapping nothing calls compiles,
// and a call into a mapping that sets nothing compiles too. Same pair
// selected_test.go and role_test.go pin.
func TestKotlinAppliesTheValueThroughItsSemanticsPrimitives(t *testing.T) {
	body := declSource(t, kotlinStyle, "fun SemanticsPropertyReceiver.grMobValue(")
	for _, pin := range []struct{ expr, why string }{
		{"progressBarRangeInfo = ProgressBarRangeInfo(",
			"the numeric half — the one mapping in this file that gets a localized " +
				"percentage out of TalkBack without a string crossing the bridge"},
		{"stateDescription = range.text",
			"the words, which TalkBack honours on any node at all"},
	} {
		if !strings.Contains(body, pin.expr) {
			t.Errorf("%s: grMobValue never reaches %s — %s", kotlinStyle, pin.expr, pin.why)
		}
	}
	if src := readNative(t, kotlinStyle); !strings.Contains(src, "grMobValue(valueRange)") {
		t.Errorf("%s: boxModifier never calls grMobValue — the mapping exists and "+
			"nothing invokes it", kotlinStyle)
	}
}

// --- SwiftUI: the words reach a modifier -----------------------------------

func TestSwiftAppliesTheValueTextThroughAccessibilityValue(t *testing.T) {
	body := declSource(t, swiftStyle, "fileprivate func grMobValueText(")
	if !strings.Contains(body, "accessibilityValue(Text(") {
		t.Errorf("%s: grMobValueText never reaches accessibilityValue — the words are "+
			"parsed and then dropped", swiftStyle)
	}
	if src := readNative(t, swiftStyle); !strings.Contains(src, ".grMobValueText(s)") {
		t.Errorf("%s: grMobBox's chain never applies grMobValueText — the mapping "+
			"exists and nothing invokes it", swiftStyle)
	}
}

// --- SwiftUI: the numbers reach nothing, on purpose ------------------------

// The gap half. SwiftUI has no numeric accessibility value — accessibilityValue
// takes a Text and there is no equivalent of ProgressBarRangeInfo — so the three
// numbers arrive, are visible in the struct, and go no further.
//
// What this catches is the tempting fix: formatting them into a string here.
// That would put the framework in the business of inventing English for every
// app in every locale, and it would overwrite a value slot that belongs to the
// app. The note is pinned alongside the absence so it cannot quietly outlive
// the limitation it explains.
func TestSwiftDoesNotInventWordsForTheNumbers(t *testing.T) {
	// The *body* of the one function that touches the range, not the file:
	// this file's own prose names the numbers repeatedly, and a pin that
	// matched the explanation rather than the code is the failure mode two
	// sessions of flush-pin misses have already produced.
	body := declSource(t, swiftStyle, "fileprivate func grMobValueText(")
	for _, member := range []string{".now", ".min", ".max"} {
		if strings.Contains(body, "accessibilityValue"+member) ||
			strings.Contains(body, "\\(s?.accessibilityValue"+member) {
			t.Errorf("%s: grMobValueText reads %s — a renderer that spells a number "+
				"out loud is choosing a language for every app that uses it, and it "+
				"is overwriting a value slot that belongs to the app. "+
				"components.ProgressBar.ValueText is where an app supplies its own "+
				"words", swiftStyle, member)
		}
	}
	src := readNative(t, swiftStyle)
	for _, pin := range []struct{ expr, why string }{
		{"SwiftUI has no numeric",
			"the note saying which property this platform is missing rather than " +
				"which one it is turning down"},
		{"mobile/verify/value_test.go", "the pointer back at this file"},
	} {
		if !strings.Contains(src, pin.expr) {
			t.Errorf("%s: %q not found — %s", swiftStyle, pin.expr, pin.why)
		}
	}
}
