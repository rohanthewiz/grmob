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
		if src := valuesIn(t, pin.file); !strings.Contains(src, pin.key) {
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
	swift := codeIn(t, swiftStyle)
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

	kotlin := codeIn(t, kotlinStyle)
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

// core.Style.AccessibilityCurrent has no property on either native and is
// folded into the selected state on both (see core.CurrentKind). Pinned as the
// selected chain is: the parse, the fold, the call that reaches the platform
// primitive, and on Compose the lambda opening for a current node alone. Each
// link breaks silently — an unread key and an uncalled mapping both compile.
func TestBothNativesFoldTheCurrentItemIntoSelected(t *testing.T) {
	for _, pin := range []struct{ file, key string }{
		{swiftStyle, `str("AccessibilityCurrent")`},
		{kotlinStyle, `optString("AccessibilityCurrent")`},
	} {
		if src := valuesIn(t, pin.file); !strings.Contains(src, pin.key) {
			t.Errorf("%s: never parses %s — BottomBar's current cell and StepIndicator's "+
				"current step would announce nothing on this platform", pin.file, pin.key)
		}
	}
	swift := codeIn(t, swiftStyle)
	for _, expr := range []string{
		"private func grMobCurrentTrait(",
		"grMobCurrentTrait(s.accessibilityCurrent, selected: s.accessibilitySelected)",
	} {
		if !strings.Contains(swift, expr) {
			t.Errorf("%s: %q not found — the fold of a current item into .isSelected", swiftStyle, expr)
		}
	}
	kotlin := codeIn(t, kotlinStyle)
	for _, expr := range []string{
		"fun SemanticsPropertyReceiver.grMobCurrent(",
		"grMobCurrent(currentKind, selectedState)",
		"currentKind.isNotEmpty()",
	} {
		if !strings.Contains(kotlin, expr) {
			t.Errorf("%s: %q not found — the fold of a current item into selected, or the "+
				"semantics lambda opening for one", kotlinStyle, expr)
		}
	}

	// The fold expressions themselves, read with their literals intact: each
	// now names "date" as the kind it leaves out, and codeIn blanks string
	// literals. See TestBothNativesSpeakTheCurrentDateIntoTheName for why.
	for _, pin := range []struct{ file, expr string }{
		{swiftStyle, `!kind.isEmpty && kind != "date" && selected.isEmpty ? .isSelected : []`},
		{kotlinStyle, `if (kind.isNotEmpty() && kind != "date" && state.isEmpty()) selected = true`},
	} {
		if !strings.Contains(valuesIn(t, pin.file), pin.expr) {
			t.Errorf("%s: %q not found — the fold of a current item into the selected "+
				"state, with the current date left out of it", pin.file, pin.expr)
		}
	}
}

// core.CurrentDate is the one kind neither native folds into selected: a
// calendar's today cell announced as selected would name the wrong day as the
// chosen one, and a calendar has a chosen day of its own. Both speak it into
// the name instead, as the ", today" comps.Calendar used to write in Go for
// every target (see core.CurrentKind, "CurrentDate is the exception to the
// fold").
//
// Three links on each platform, and every one of them fails silently: the fold
// could fold it again (TestBothNativesFoldTheCurrentItemIntoSelected holds the
// exclusion), the helper could stop appending, or the name could stop being
// routed through the helper — in which case today on a phone would simply be a
// day, with nothing anywhere saying so. Read with literals intact, because the
// suffix and the kind are both string literals.
func TestBothNativesSpeakTheCurrentDateIntoTheName(t *testing.T) {
	for _, pin := range []struct{ file, expr, why string }{
		{swiftStyle, `private func grMobCurrentLabel(_ label: String, kind: String) -> String {`,
			"the helper that speaks the current date into the label"},
		{swiftStyle, `kind == "date" ? label + ", " + grMobTodayWord() : label`,
			"the suffix itself"},
		{swiftStyle, `formatter.localizedString(from: DateComponents(day: 0))`,
			"the suffix's word, from the platform in the user's language"},
		{swiftStyle, `.accessibilityLabel(grMobCurrentLabel(s.accessibilityLabel, kind: s.accessibilityCurrent))`,
			"grMobAccessibility routing the label through the helper"},
		{kotlinStyle, `fun grMobCurrentLabel(label: String, kind: String): String =`,
			"the helper that speaks the current date into the contentDescription"},
		{kotlinStyle, `if (kind == "date" && label.isNotEmpty()) "$label, ${grMobTodayWord()}" else label`,
			"the suffix itself, and no bare suffix on a node with no name"},
		{kotlinStyle, `.format(RelativeDateTimeFormatter.Direction.THIS, RelativeDateTimeFormatter.AbsoluteUnit.DAY)`,
			"the suffix's word, from ICU in the device's language"},
		{kotlinStyle, `listOf(grMobCurrentLabel(accessibilityLabel, currentKind), accessibilityHint)`,
			"boxModifier routing the description through the helper"},
	} {
		if !strings.Contains(valuesIn(t, pin.file), pin.expr) {
			t.Errorf("%s: %q not found — %s. comps.Calendar's today cell would be "+
				"announced as an ordinary day on this platform", pin.file, pin.expr, pin.why)
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
	src := codeOf(t, kotlinStyle, "fun GrMobStyle?.boxModifier(")
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
	// proseOf: the subject is grMobSelectedTrait's own note, which is a doc
	// comment above it — so this wants the declaration's region with the prose
	// kept, which is the one cut this package did not used to have. It read the
	// whole file, and a phrase found anywhere in a thousand lines of renderer
	// answered a question about one function.
	src := proseOf(t, swiftStyle, "func grMobSelectedTrait(")
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
