package verify

import (
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// core.Style.AccessibilityExpanded, held against both native renderers.
//
// This field splits the two platforms further apart than any other in the
// accessibility vocabulary, and the split runs the opposite way to the usual
// one. Elsewhere the two natives agree and the web is the strict target —
// both phones honour a label, a role hint and a selected state on any node,
// while ARIA scopes its attributes and the exporters drop what does not fit.
// Here:
//
//	web        aria-expanded, scoped to six roles plus the Button node type
//	Compose    expand()/collapse() semantics *actions*, wired to the node's
//	           own click callback
//	SwiftUI    nothing at all
//
// So the two platforms are pinned for opposite things. Compose is pinned like
// a mapping — parsed, dispatched, reaching a platform primitive, covering
// every state — and SwiftUI like a *gap*, the shape idref_test.go and
// nesting_level_test.go cover, because a field a renderer silently ignores is
// indistinguishable on device from one nobody had heard of.

// --- Compose: the field is read at all -------------------------------------

// An unread JSON key is not an error in Kotlin either, so a renderer that
// dropped this line would compile, run, and offer TalkBack no way to open any
// accordion in the app.
func TestKotlinParsesTheExpandedField(t *testing.T) {
	src := valuesIn(t, kotlinStyle)
	for _, pin := range []struct{ expr, why string }{
		{`optString("AccessibilityExpanded")`, "the parse"},
		{"val accessibilityExpanded: String,", "the field it parses into"},
	} {
		if !strings.Contains(src, pin.expr) {
			t.Errorf("%s: %q not found — %s", kotlinStyle, pin.expr, pin.why)
		}
	}
}

// --- Compose: the mapping, and where it had to live -------------------------

// The mapping reaches Compose's own disclosure primitives, and it is invoked.
//
// Both links break silently and separately: a mapping nothing calls compiles,
// and a call into a mapping that sets nothing compiles too. That is the same
// pair selected_test.go pins — but the *location* is what is unusual here and
// is pinned alongside, because it is the thing a later tidy-up would undo.
//
// Every other accessibility field is a semantics **property** and is spent in
// GrMobStyle.boxModifier's semantics lambda beside grMobRole and grMobSelected.
// Compose has no expanded property; it has `expand` and `collapse`, which are
// *actions*, and an action has to have something to perform. The only thing
// that can open this disclosure is the callback the node's own tap already
// runs — which lives on the node, not on the style — so the mapping is in
// Renderer.kt's gesture layer. Moving it next to its siblings would put it
// somewhere the click ID cannot be reached.
func TestKotlinMapsTheDisclosureOntoItsSemanticsActions(t *testing.T) {
	src := valuesIn(t, kotlinRenderer)
	for _, pin := range []struct{ expr, why string }{
		{"private fun Modifier.grMobDisclosure(", "the mapping from core.ExpandedState"},
		{"import androidx.compose.ui.semantics.expand", "the expand action's import"},
		{"import androidx.compose.ui.semantics.collapse", "the collapse action's import"},
		{"expand { toggle(); true }",
			"the expand arm — the action TalkBack offers on a *closed* disclosure, wired to " +
				"the same callback a tap runs"},
		{"collapse { toggle(); true }", "the collapse arm, offered on an open one"},
		{".grMobDisclosure(node.style?.accessibilityExpanded ?: \"\")",
			"the call, from gestureModifier, which is the one place that holds both the " +
				"style's state and the node's click ID"},
	} {
		if !strings.Contains(src, pin.expr) {
			t.Errorf("%s: %q not found — %s", kotlinRenderer, pin.expr, pin.why)
		}
	}
}

// Which action goes with which state, asserted as the pairing rather than as
// two independent arms.
//
// The transposition is this mapping's one silent bug and it is invisible on
// every screen: both arms compile, both wire the same callback, and both
// toggle the section correctly when activated. Only the announcement is wrong
// — a shut accordion inviting the reader to collapse it. Two `strings.Contains`
// checks for `expand` and `collapse` would pass on the swapped version, so the
// state literal and its action are matched here as one string.
func TestKotlinOffersTheActionTheStateCallsFor(t *testing.T) {
	src := valuesOf(t, kotlinRenderer, "private fun Modifier.grMobDisclosure(")
	for _, pin := range []struct{ expr, why string }{
		{`"false" -> this.semantics { expand`,
			"a closed disclosure offers *expand*. Offering collapse here announces a shut " +
				"section as one the reader can close"},
		{`"true" -> this.semantics { collapse`, "and an open one offers collapse"},
	} {
		if !strings.Contains(src, pin.expr) {
			t.Errorf("%s: grMobDisclosure is missing %q — %s", kotlinRenderer, pin.expr, pin.why)
		}
	}
}

// Compose's dispatch, held against core.ExpandedStates() the way
// selected_test.go holds grMobSelected against core.SelectedStates().
//
// Only Compose gets a coverage check, and for the reason stated there: a
// `when` over the spellings is a dispatch with a catch-all, so a state with no
// arm is silently inert. SwiftUI has no dispatch here at all — it has no arms
// to be missing, because it has nothing to map onto.
func TestKotlinDisclosureCoversEveryState(t *testing.T) {
	syntax := kotlinWhen.with(
		kotlinRenderer,
		"private fun Modifier.grMobDisclosure(",
		"when (state) {",
	)
	requireExpandedCoverage(t, "Renderer.kt", "grMobDisclosure", syntax.labels(t))
}

// requireExpandedCoverage checks a renderer's arms against
// core.ExpandedStates() in both directions, for the reasons
// requireSelectedCoverage gives: a state with no arm falls into the catch-all
// where it is inert, and an arm with no state is dead code that reads as
// support.
//
// ExpandedUnset is not in core.ExpandedStates() and must not appear as an arm.
// It is the field's zero value, so every node in the tree carries it, and an
// arm for it would be a renderer implementing "not a disclosure".
func requireExpandedCoverage(t *testing.T, file, fn string, arms []string) {
	t.Helper()

	missing := map[string]bool{}
	for _, state := range core.ExpandedStates() {
		missing[string(state)] = true
	}

	for _, label := range arms {
		if !missing[label] {
			t.Errorf("%s: %s has an arm for %q, which is not a core.ExpandedState (or is a "+
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
		t.Errorf("%s: %s has no arm for core.ExpandedState %q — it falls into the catch-all, "+
			"where the state is dropped and the header announces as a plain button with "+
			"nothing behind it", file, fn, state)
	}
}

// --- SwiftUI: the gap, and the near miss ------------------------------------

// The key crosses the bridge and is deliberately not parsed. Not parsed rather
// than parsed and dropped, on the rule nesting_level_test.go states: a field
// read into a property nothing spends looks like support to the next reader.
func TestSwiftDoesNotParseTheExpandedField(t *testing.T) {
	src := codeIn(t, swiftStyle)
	for _, absent := range []string{
		`str("AccessibilityExpanded")`,
		"accessibilityExpanded",
	} {
		if strings.Contains(src, absent) {
			t.Errorf("%s: parses %q. SwiftUI has no expanded trait, so either the platform "+
				"grew one — in which case map it and rewrite the note — or this is a field "+
				"read and thrown away, which reads as support and is not",
				swiftStyle, absent)
		}
	}
}

// The note, in the file and beside the function where the next person looks.
//
// It sits with the IDREF note above grMobTraitsFor for the same reason that one
// does: a reader hunting for "where does AccessibilityExpanded become a trait"
// arrives at the trait table, and what has to be there is the sentence saying
// it does not.
func TestSwiftWritesDownTheExpandedGap(t *testing.T) {
	src := proseIn(t, swiftStyle)
	if !strings.Contains(src, "AccessibilityExpanded is not read here") {
		t.Errorf("%s: the note explaining why the disclosure state is dropped is gone. "+
			"A dropped field with no note is indistinguishable from one nobody implemented",
			swiftStyle)
	}
}

// The note names the property it is turning down, and says the shape of the
// wrong mapping rather than merely that there is none.
//
// This is idref_test.go's wrinkle in a second form. There the near miss was
// `accessibilityIdentifier`, which is a test selector wearing an accessibility
// name; here it is `accessibilityValue`, which is a real accessibility channel
// that SwiftUI's own DisclosureGroup uses — so the mapping is not merely
// tempting, it is what the platform does. What makes it wrong for a *framework*
// is that SwiftUI supplies the string from its own localized bundle and this
// renderer would have to supply an English literal, for every app in every
// locale, in a slot the app may want for a real value.
//
// A note that said only "no equivalent" would leave the next person to find
// accessibilityValue and think they had found the equivalent.
func TestSwiftNamesTheValueChannelItIsNotUsing(t *testing.T) {
	src := proseIn(t, swiftStyle)
	// Backticked, the way the other native notes spell a property they turn
	// down: a bare "accessibilityValue" also matches prose about values
	// elsewhere in the file, which is a pin weak enough to survive the note
	// being deleted.
	if !strings.Contains(src, "`accessibilityValue`") {
		t.Errorf("%s: the note no longer names accessibilityValue as the near miss. That is "+
			"the specific wrong mapping — SwiftUI's own DisclosureGroup announces through it "+
			"— and naming it is what stands between the next reader and an English literal "+
			"shipped to every locale", swiftStyle)
	}
	if !strings.Contains(src, "AccessibilityTraits` has no expanded member") {
		t.Errorf("%s: the note no longer says *why* there is no trait mapping. If SwiftUI "+
			"grows the member, this fails and hands over the paragraph to rewrite", swiftStyle)
	}
}
