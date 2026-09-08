package verify

import (
	"strings"
	"testing"
)

// core.Style.AccessibilityID and core.Style.AccessibilityControls, held
// against both native renderers.
//
// The pair is the vocabulary's only reference: an element identity, and the
// aria-controls that points at it. What asked for them was a tab strip built
// by hand — core.TabView mints its own ids and writes the whole wiring from
// the node type, but a strip assembled out of chips could say role="tab" and
// role="tablist" and then had no way to say which region each tab shows.
//
// That is a browser question end to end. Neither native has a relationship of
// this kind in its semantics vocabulary, and neither reader needs one:
// VoiceOver and TalkBack both move through a screen by swiping to the next
// element rather than by following a reference. So this is the shape
// nesting_level_test.go covers — a field inert on both platforms — and it is
// pinned the same way, because "nothing to become" and "nobody read the key"
// render identically on device.
//
// It has one wrinkle that field does not, and it is why the notes matter more
// here than the absence does. Both platforms *do* have a property that looks
// like AccessibilityID: accessibilityIdentifier on iOS, testTag on Compose.
// Neither is an accessibility property — both are UI-test selectors, read by
// XCUITest and Espresso and not exposed to either screen reader — so a future
// reader who maps this field onto one of them will have made the framework
// quietly rewrite every app's test handles and will still have announced
// nothing. The notes are what stand between that reader and the mistake.

// The two documentation halves. Each native says why it drops the pair, in the
// file and beside the function where the next person would look for it.
func TestBothNativesWriteDownTheIDRefGap(t *testing.T) {
	for _, pin := range []struct{ file, note, why string }{
		{swiftStyle, "AccessibilityID and AccessibilityControls are not read here",
			"above grMobTraitsFor, which is where a reader looking for the mapping arrives"},
		{kotlinStyle, "AccessibilityID and AccessibilityControls are not read here either",
			"above grMobRole, beside the two level notes it repeats the shape of"},
	} {
		if !strings.Contains(proseIn(t, pin.file), pin.note) {
			t.Errorf("%s: %q not found — the note belongs %s, and a field a renderer simply "+
				"ignored is indistinguishable from one nobody had heard of",
				pin.file, pin.note, pin.why)
		}
	}
}

// Each note also names the test selector it is turning down, because that is
// the specific wrong mapping, not a general one. A note that merely said "no
// equivalent" would leave the next person to rediscover accessibilityIdentifier
// and think they had found the equivalent.
func TestBothNativesNameTheTestSelectorTheyAreNotUsing(t *testing.T) {
	// Backticked, which is how each note spells the property it is turning
	// down. A bare "testTag" would also match testTagsAsResourceId in the
	// sentence after it, so the note could lose the naming and still pass.
	for _, pin := range []struct{ file, property string }{
		{swiftStyle, "`accessibilityIdentifier`"},
		{kotlinStyle, "`testTag`"},
	} {
		if !strings.Contains(proseIn(t, pin.file), pin.property) {
			t.Errorf("%s: the IDREF note does not name %s — that property is the near miss "+
				"this field must not be mapped onto, and naming it is the whole warning",
				pin.file, pin.property)
		}
	}
}

// The other half of the same claim: neither renderer parses either key. An arm
// that read one and threw it away would be exactly the silent case the notes
// say this is not.
func TestNeitherNativeParsesTheIDRefPair(t *testing.T) {
	for _, file := range []string{kotlinStyle, swiftStyle} {
		src := codeIn(t, file)
		for _, key := range []string{`AccessibilityID"`, `AccessibilityControls"`} {
			if strings.Contains(src, key) {
				t.Errorf("%s: parses %s — if the platform grew a way to state that one "+
					"element controls another, the note that says it cannot has to go "+
					"with it", file, key)
			}
		}
	}
}
