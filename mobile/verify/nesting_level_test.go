package verify

import (
	"strings"
	"testing"
)

// core.Style.AccessibilityNestingLevel, held against both native renderers.
//
// This is the field with no native half at all, which makes it the strongest
// case for the kind of check this package does. The heading tier next door is
// asymmetric — SwiftUI carries it, Compose cannot — so half of it is provable
// by looking for a mapping. This one is inert on both platforms: SwiftUI has
// no nesting-depth property of any kind, and Compose's nearest thing,
// collectionItemInfo, states an item's index and span within *one* collection
// rather than its depth within nested ones, which is a different claim. Filling
// it from this field would tell TalkBack something the app never said.
//
// So there is nothing to pin except the two things that keep silence honest:
// the note that says the gap is deliberate, and the absence of a parse that
// would contradict it. That pairing is the same one
// TestKotlinWritesDownTheHeadingLevelGap makes, and it exists because "nothing
// to become" and "nobody read the key" render identically on device.

// The two documentation halves. Each native says why it drops the field, in
// the file where the next person would look for the mapping.
func TestBothNativesWriteDownTheNestingLevelGap(t *testing.T) {
	for _, pin := range []struct{ file, note, why string }{
		{kotlinStyle, "AccessibilityNestingLevel is not read here either",
			"beside the role dispatch and the heading-level note, which is where a reader " +
				"looking for the mapping arrives"},
		{swiftStyle, "The other two roles aria-level serves have no mapping here",
			"beside grMobHeadingLevel, since SwiftUI does map the heading third and the " +
				"obvious question on reading it is what happened to the other two"},
	} {
		if !strings.Contains(readNative(t, pin.file), pin.note) {
			t.Errorf("%s: %q not found — the note belongs %s, and a field a renderer simply "+
				"ignored is indistinguishable from one nobody had heard of",
				pin.file, pin.note, pin.why)
		}
	}
}

// The other half of the same claim: neither renderer parses the key. An arm
// that read it and threw it away would be exactly the silent case the notes
// say this is not — and if either platform ever grows a depth property, this
// test is what hands the person adding the mapping the paragraphs to rewrite.
func TestNeitherNativeParsesTheNestingLevel(t *testing.T) {
	for _, file := range []string{kotlinStyle, swiftStyle} {
		if strings.Contains(readNative(t, file), "AccessibilityNestingLevel\"") {
			t.Errorf("%s: parses AccessibilityNestingLevel — if the platform grew a "+
				"nesting-depth property, the note that says it cannot has to go with it",
				file)
		}
	}
}
