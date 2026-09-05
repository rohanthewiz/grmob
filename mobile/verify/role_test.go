package verify

import (
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// core.AccessibilityRole, held against both native renderers.
//
// This prop is the package's typical subject with one twist. The typical part:
// a Go style field crosses the bridge as JSON, both web targets honour it for
// free (core.Role's values *are* ARIA's spellings, so the DOM mapping is the
// value verbatim), and the only thing between "implemented" and "silently
// inert on device" is whether each native bothers to read the key.
//
// The twist is that most of the vocabulary maps to nothing on these two
// platforms, on purpose — SwiftUI has no landmarks, Compose has no tabular
// semantics — so "does nothing" is a legitimate answer for a role here and is
// indistinguishable, at the call site and on the device, from "nobody taught
// this renderer about it". That is exactly why both dispatches spell out the
// roles they drop, and why these checks are coverage checks rather than
// behavior checks: what is pinned is that every role has been *considered*,
// arm by arm, not that any particular one does something.

// The style field itself. An unread JSON key is not a type error in either
// language, so a renderer that skipped this line would compile, run, and
// render every role as nothing at all.
func TestBothNativeParsersReadTheRoleField(t *testing.T) {
	for _, pin := range []struct {
		file, key string
	}{
		{swiftStyle, `str("AccessibilityRole")`},
		{kotlinStyle, `optString("AccessibilityRole")`},
	} {
		if src := readNative(t, pin.file); !strings.Contains(src, pin.key) {
			t.Errorf("%s: never parses %s — core.AccessibilityRole reaches both web targets "+
				"and does nothing on this platform", pin.file, pin.key)
		}
	}
}

// Reading the key and doing nothing with it is the same outcome as not reading
// it, so each renderer is also pinned to the platform primitive its mapping
// has to end up in — the same pair collections_test.go checks for the tier B
// props. SwiftUI states a role as traits on the view; Compose states it as
// properties inside a semantics lambda.
func TestBothNativesApplyTheRoleThroughTheirSemanticsPrimitive(t *testing.T) {
	for _, pin := range []struct {
		file, decl, primitive string
	}{
		{swiftStyle, "func grMobRole(", "accessibilityAddTraits("},
		{kotlinStyle, "fun SemanticsPropertyReceiver.grMobRole(", "heading()"},
	} {
		src := declSource(t, pin.file, pin.decl)
		if !strings.Contains(src, pin.primitive) {
			t.Errorf("%s: %s never reaches %s — the role is parsed and then dropped",
				pin.file, pin.decl, pin.primitive)
		}
	}
	// The Compose half needs one more link than the Swift half: grMobRole is
	// an extension on the receiver rather than a modifier, so it does nothing
	// unless boxModifier actually calls it from inside its semantics lambda.
	// SwiftUI's is a modifier in the grMobBox chain, which the chain check
	// below covers.
	if src := readNative(t, kotlinStyle); !strings.Contains(src, "grMobRole(kind)") {
		t.Errorf("%s: boxModifier never calls grMobRole — the mapping exists and nothing invokes it",
			kotlinStyle)
	}
	if src := readNative(t, swiftStyle); !strings.Contains(src, ".grMobRole(s)") {
		t.Errorf("%s: grMobBox's chain never applies grMobRole — the mapping exists and nothing "+
			"invokes it", swiftStyle)
	}
}

// The two dispatches, held against core.Roles(). See requireRoleCoverage for
// what each direction catches.
func TestSwiftTraitsCoverEveryRole(t *testing.T) {
	syntax := swiftSwitch.with(
		swiftStyle,
		"func grMobTraitsFor(",
		"switch role {",
	)
	requireRoleCoverage(t, "GrMobStyle.swift", "grMobTraitsFor", syntax.labels(t))
}

func TestKotlinRoleCoversEveryRole(t *testing.T) {
	syntax := kotlinWhen.with(
		kotlinStyle,
		"fun SemanticsPropertyReceiver.grMobRole(",
		"when (kind) {",
	)
	requireRoleCoverage(t, "GrMobStyle.kt", "grMobRole", syntax.labels(t))
}

// requireRoleCoverage checks a renderer's arms against core.Roles() in both
// directions.
//
// Both directions matter and they fail for different reasons:
//
//   - A role with no arm falls into the catch-all, where it is inert. That is
//     the same rendering nine of the fourteen roles get on purpose, which is
//     precisely why the omission has to be caught here: on this platform there
//     is no visible difference between "deliberately does nothing" and
//     "nobody has heard of it", and the difference is the whole of what the
//     next person needs to know.
//   - An arm with no role is dead code that reads as support. Either a typo
//     that has never matched anything, or a role deleted from core and left
//     standing here.
//
// RoleNone is not in core.Roles() and must not appear as an arm: it is the
// field's zero value, so every node in the tree carries it, and an arm for it
// would be a renderer implementing "unset".
func requireRoleCoverage(t *testing.T, file, fn string, arms []string) {
	t.Helper()

	missing := map[string]bool{}
	for _, role := range core.Roles() {
		missing[string(role)] = true
	}

	for _, label := range arms {
		if !missing[label] {
			// Not a role, or already claimed by an earlier arm — in the second
			// case this arm is unreachable, which deserves the same complaint.
			t.Errorf("%s: %s has an arm for %q, which is not a core.Role (or is a second arm for "+
				"one, and therefore unreachable)", file, fn, label)
			continue
		}
		delete(missing, label)
	}

	for _, role := range sortedRoles(missing) {
		t.Errorf("%s: %s has no arm for core.Role %q — it falls into the catch-all, where it is "+
			"indistinguishable from the roles this platform drops on purpose", file, fn, role)
	}
}

// Deterministic failure order, for the reason sortedNames exists in
// switchlabels_test.go: a run that reports several missing roles should report
// them in the same order twice.
func sortedRoles(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for role := range m {
		out = append(out, role)
	}
	sort.Strings(out)
	return out
}
