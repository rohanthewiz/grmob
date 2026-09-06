package verify

import (
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// core.Style.AccessibilitySelected, held against both native renderers.
//
// The field is the reverse of the shape role_test.go covers. A role is one
// value that most of these two platforms cannot say; a selected state is one
// value that *both* can say, which is the rarer case here and makes the
// failure mode a quiet one — a widget marking its chips announces correctly on
// the two web targets and silently loses the state on device, with nothing in
// any log and no visual difference at all.
//
// The two platforms need no switch on the role, unlike the web exporters,
// because each has one spelling where ARIA has two attributes: Compose's
// `selected` property and SwiftUI's `.isSelected` trait both cover
// aria-selected and aria-pressed. So what is pinned per platform is the chain
// — parsed, mapped, and reaching the platform's own semantics primitive —
// plus the one asymmetry between them.

// The style field itself, on both. An unread JSON key is not a type error in
// either language, so a renderer that dropped this line would compile, run,
// and announce every chip in the app as though nothing were chosen.
func TestBothNativeParsersReadTheSelectedField(t *testing.T) {
	for _, pin := range []struct {
		file, key string
	}{
		{swiftStyle, `str("AccessibilitySelected")`},
		{kotlinStyle, `optString("AccessibilitySelected")`},
	} {
		if src := readNative(t, pin.file); !strings.Contains(src, pin.key) {
			t.Errorf("%s: never parses %s — core.AccessibilitySelected reaches both web "+
				"targets and does nothing on this platform", pin.file, pin.key)
		}
	}
}

// Reading the key and doing nothing with it is the same outcome as not reading
// it, so each renderer is also pinned to the platform primitive the mapping
// has to end up in, and to the call that invokes the mapping. Both links break
// silently: a mapping nothing calls compiles, and a call into a mapping that
// sets nothing compiles too.
func TestBothNativesApplyTheSelectedStateThroughTheirSemanticsPrimitive(t *testing.T) {
	swift := readNative(t, swiftStyle)
	for _, pin := range []struct{ expr, why string }{
		{"private func grMobSelectedTrait(", "the mapping from core.SelectedState onto a trait"},
		{".isSelected", "the SwiftUI trait itself — the one thing VoiceOver can be told here"},
		{"grMobSelectedTrait(s.accessibilitySelected)",
			"the call. grMobRole unions the state into the role's traits, so the mapping " +
				"reaches the view through the modifier chain that is already there"},
	} {
		if !strings.Contains(swift, pin.expr) {
			t.Errorf("%s: %q not found — %s", swiftStyle, pin.expr, pin.why)
		}
	}

	kotlin := readNative(t, kotlinStyle)
	for _, pin := range []struct{ expr, why string }{
		{"fun SemanticsPropertyReceiver.grMobSelected(", "the mapping onto Compose semantics"},
		{"selected = true", "the on arm, which is the property TalkBack reads"},
		{"selected = false",
			"the off arm. Compose is the one native that can state it, which is half of " +
				"why core.SelectedState has three values"},
		{"grMobSelected(selectedState)",
			"the call from inside boxModifier's semantics lambda. grMobSelected is an " +
				"extension on the receiver rather than a modifier, so it does nothing " +
				"unless something invokes it there"},
	} {
		if !strings.Contains(kotlin, pin.expr) {
			t.Errorf("%s: %q not found — %s", kotlinStyle, pin.expr, pin.why)
		}
	}
}

// The semantics lambda is entered only when there is something to say, and the
// state has to be one of the things that opens it.
//
// This is the failure the field is most likely to have shipped with, and it is
// invisible from the mapping: grMobSelected can be correct, called, and never
// reached, because a chip that carries a state and no label, hint, role or
// disabled flag would take the else branch and get no semantics at all.
func TestKotlinOpensItsSemanticsLambdaForASelectedStateAlone(t *testing.T) {
	src := declSource(t, kotlinStyle, "fun GrMobStyle?.boxModifier(")
	if !strings.Contains(src, "selectedState.isNotEmpty()") {
		t.Errorf("%s: boxModifier's semantics guard does not include the selected state — "+
			"a node whose only semantics are a selection would fall to the else branch "+
			"and announce nothing", kotlinStyle)
	}
}

// The one place the two platforms genuinely differ, written down rather than
// left to be rediscovered: SwiftUI has no trait for "not selected", so
// SelectedOff and SelectedUnset produce the same view there.
//
// Pinned as a documentation check for the reason heading_level_test.go pins
// Compose's missing heading level: a limitation that is merely absent from the
// code reads exactly like an oversight, and the next person to look has to
// re-derive it from Apple's docs. If SwiftUI ever grows the trait, this fails
// and hands over the paragraph to rewrite.
func TestSwiftWritesDownTheUnselectedGap(t *testing.T) {
	// readNative rather than declSource: the note is a doc comment, and
	// declSource cuts from the declaration line down. Same reason
	// TestKotlinWritesDownTheHeadingLevelGap reads the whole file.
	src := readNative(t, swiftStyle)
	for _, phrase := range []string{"no word for the *off* state", "not selectable"} {
		if !strings.Contains(src, phrase) {
			t.Errorf("%s: grMobSelectedTrait no longer explains that SwiftUI cannot state "+
				"the unselected case (looking for %q). Either the platform grew the trait, "+
				"in which case map it, or the note was lost", swiftStyle, phrase)
		}
	}
}

// Compose's dispatch, held against core.SelectedStates() the way role_test.go
// holds the role dispatches against core.Roles().
//
// Only Compose gets a coverage check, and the asymmetry is the point rather
// than an omission. A `when` over the two spellings is a dispatch with a
// catch-all, so a state with no arm is silently inert exactly as a role with
// no arm is. SwiftUI's mapping is a single ternary — it has one trait and no
// arms to be missing — so there is nothing there for a census to be held
// against; what that platform can get wrong is the *value*, which
// core/selected_enum_test.go pins on the Go side for all four renderers.
func TestKotlinSelectedCoversEveryState(t *testing.T) {
	syntax := kotlinWhen.with(
		kotlinStyle,
		"fun SemanticsPropertyReceiver.grMobSelected(",
		"when (state) {",
	)
	requireSelectedCoverage(t, "GrMobStyle.kt", "grMobSelected", syntax.labels(t))
}

// requireSelectedCoverage checks a renderer's arms against
// core.SelectedStates() in both directions, for the reasons requireRoleCoverage
// gives at length: a state with no arm falls into the catch-all where it is
// inert, and an arm with no state is dead code that reads as support.
//
// SelectedUnset is not in core.SelectedStates() and must not appear as an arm.
// It is the field's zero value, so every node in the tree carries it, and an
// arm for it would be a renderer implementing "unstated".
func requireSelectedCoverage(t *testing.T, file, fn string, arms []string) {
	t.Helper()

	missing := map[string]bool{}
	for _, state := range core.SelectedStates() {
		missing[string(state)] = true
	}

	for _, label := range arms {
		if !missing[label] {
			t.Errorf("%s: %s has an arm for %q, which is not a core.SelectedState (or is a "+
				"second arm for one, and therefore unreachable)", file, fn, label)
			continue
		}
		delete(missing, label)
	}

	states := make([]string, 0, len(missing))
	for state := range missing {
		states = append(states, state)
	}
	sort.Strings(states)
	for _, state := range states {
		t.Errorf("%s: %s has no arm for core.SelectedState %q — it falls into the catch-all, "+
			"where the state is dropped and the control announces as though it had none",
			file, fn, state)
	}
}
