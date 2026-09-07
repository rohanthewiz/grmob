package verify

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The Kotlin file holding the numeric reading. Split out of GrMobStyle.kt so
// that it imports nothing and android/verify can run it on a JVM — see
// TestTheKotlinValueReadingIsUIFree.
var kotlinProgress = nativeFile("android", "app", "src", "main", "java", "com", "grmob",
	"runtime", "GrMobProgress.kt")

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
		// stand over `?: 0f`, which is exactly the fix somebody reaches for to
		// make the type non-null — and which reintroduces the bug the nullable
		// field exists to prevent.
		//
		// The parse goes through grMobProgressNumber rather than calling
		// toFloatOrNull here, so that one rule decides what counts as a number
		// and android/verify can run it: the parse is half of what the reading
		// depends on, and a harness given pre-parsed floats would be checking
		// the half that cannot go wrong.
		{"fun num(name: String): Float? = grMobProgressNumber(obj.optString(name))\n",
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

	// The third link, and the one this file used to *be*: the numeric reading
	// itself. It is grMobProgressOf's now, in a file android/verify runs
	// against core.ValueRange.Progress, so what is left to check here is that
	// grMobValue still asks it — a branch reinlined into this lambda would be
	// correct-looking, unrunnable, and checked by nothing.
	if !strings.Contains(body, "grMobProgressOf(range.now, range.min, range.max)") {
		t.Errorf("%s: grMobValue does not delegate to grMobProgressOf — the three-way "+
			"reading is only checkable while it lives in a file that imports nothing "+
			"(GrMobProgress.kt); inlined here it can only be exercised on a device",
			kotlinStyle)
	}
	// Both silent readings are named rather than reached by a catch-all. They
	// are different facts — nothing was claimed, versus a claim Compose cannot
	// hold — and an `else` arm would make the second one an absence again,
	// which is the state this whole item started in.
	for _, reading := range []string{"GRMOB_PROGRESS_UNSTATED", "GRMOB_PROGRESS_EMPTY_RANGE"} {
		if !strings.Contains(body, reading) {
			t.Errorf("%s: grMobValue does not name %s — the two readings that assign "+
				"nothing say different things and both belong in the when", kotlinStyle, reading)
		}
	}
}

// The numeric reading imports nothing, which is what makes android/verify able
// to run it.
//
// Same rule TestNativeMenuDecompositionIsUIFree states for GrMobSelectMenu.kt,
// and the second file to earn it. A single Compose import here would end the
// JVM pass, and the fallback would be exactly the substring search over
// GrMobStyle.kt that this file used to rely on: a check that the word
// `Indeterminate` appears somewhere, which is green for a branch that reaches
// it on the wrong condition.
func TestTheKotlinValueReadingIsUIFree(t *testing.T) {
	for _, line := range strings.Split(readNative(t, kotlinProgress), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "import ") {
			t.Errorf("%s: %q — the reading has to stay runnable off a device, which "+
				"means importing nothing at all", kotlinProgress, strings.TrimSpace(line))
		}
	}
}

// The four readings are spelled the same on both sides.
//
// android/verify compares the values, which is the real check; this is what
// makes that comparison possible to write, because the harness compares
// strings rather than mapping between two vocabularies. A rename on either
// side would otherwise turn every case into a difference that looks like a
// logic bug.
func TestTheReadingNamesAgreeWithCore(t *testing.T) {
	src := readNative(t, kotlinProgress)
	for _, reading := range []core.ProgressReading{
		core.ProgressUnstated,
		core.ProgressIndeterminate,
		core.ProgressDeterminate,
		core.ProgressEmptyRange,
	} {
		if want := `"` + string(reading) + `"`; !strings.Contains(src, want) {
			t.Errorf("%s: no constant spelled %s — core.ProgressReading is a string type "+
				"so the two sides can compare values, and a Kotlin literal that drifts "+
				"turns every android/verify case into a difference", kotlinProgress, want)
		}
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
