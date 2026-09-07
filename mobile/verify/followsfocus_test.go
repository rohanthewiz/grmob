package verify

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

// core.Style.AccessibilitySelectionFollowsFocus, held against the three targets
// that do not act on it.
//
// The field says a composite widget should choose the member its arrow keys
// land on — ARIA's own recommendation for a tab strip over cheap panels, and
// its own warning for a listbox whose selection is expensive. Exactly one
// target implements it, and the other three are silent for two different
// reasons that are worth keeping apart:
//
//	the two natives   there are no arrow keys. VoiceOver and TalkBack cross a
//	                  collection by swipe, and a swipe moves the reader's
//	                  cursor rather than focus, so the sequence the flag
//	                  describes has no occasion on either platform.
//	htmlout           there is no key handler at all. The static exporter
//	                  writes no roving tabindex for the same reason (see
//	                  wasm/verify/keynav_test.go), and a flag saying what the
//	                  arrows should do, in a document where the arrows do
//	                  nothing, is a claim with nothing behind it.
//
// This is the nesting_level_test.go / idref_test.go shape — a field inert on a
// platform — and it is pinned the same way, because "nothing to become" and
// "nobody read the key" render identically on device.

// Neither native parses the key. An arm that read it and threw it away would be
// exactly the silent case the notes say this is not.
func TestNeitherNativeParsesSelectionFollowsFocus(t *testing.T) {
	for _, file := range []string{kotlinStyle, swiftStyle} {
		if src := readNative(t, file); strings.Contains(src, `AccessibilitySelectionFollowsFocus"`) {
			t.Errorf("%s: parses AccessibilitySelectionFollowsFocus — if the platform "+
				"grew a notion of focus moving through a collection under the user's "+
				"control, the note that says it has none has to go with it", file)
		}
	}
}

// Each native says why it drops the field, beside the notes for the other
// fields it drops.
func TestBothNativesWriteDownTheFollowsFocusGap(t *testing.T) {
	for _, pin := range []struct{ file, note string }{
		{swiftStyle, "AccessibilitySelectionFollowsFocus is not read here either"},
		{kotlinStyle, "AccessibilitySelectionFollowsFocus is not read here either"},
	} {
		if !strings.Contains(readNative(t, pin.file), pin.note) {
			t.Errorf("%s: %q not found — a field a renderer simply ignored is "+
				"indistinguishable from one nobody had heard of", pin.file, pin.note)
		}
	}
}

// Each note names the near miss it is turning down, because the general
// statement "no equivalent" is what leaves the next reader to find
// onFocusChanged and think they have found one.
func TestBothNativesNameTheFocusAPITheyAreNotUsing(t *testing.T) {
	for _, pin := range []struct{ file, api string }{
		{swiftStyle, "AccessibilityFocusState"},
		{kotlinStyle, "onFocusChanged"},
	} {
		if !strings.Contains(readNative(t, pin.file), pin.api) {
			t.Errorf("%s: the follows-focus note does not name %s — that is the API this "+
				"field must not be mapped onto, and naming it is the whole warning",
				pin.file, pin.api)
		}
	}
}

// htmlout writes nothing for it, and still writes everything else.
//
// The second half matters as much as the first: an exporter that dropped the
// whole node would also pass "no attribute was written", and the point is that
// the semantics are intact and only the behaviour is absent.
func TestTheStaticExportWritesNoFollowsFocusFlag(t *testing.T) {
	out := htmlout.ExportHTML(&core.Node{
		Type: "Row",
		Style: &core.Style{
			AccessibilityRole:                  core.RoleTabList,
			AccessibilitySelectionFollowsFocus: true,
		},
		Children: []*core.Node{
			{Type: "Box", Style: &core.Style{
				AccessibilityRole:     core.RoleTab,
				AccessibilitySelected: core.SelectedOn,
			}},
		},
	})

	for _, forbidden := range []string{"selection-follows-focus", "SelectionFollowsFocus", "tabindex"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("htmlout wrote %q:\n%s\n\nThe static export has no key handler, so a "+
				"statement about what its arrow keys do is a claim with nothing behind "+
				"it — the same argument that keeps the roving tabindex out.", forbidden, out)
		}
	}
	for _, want := range []string{`role="tablist"`, `role="tab"`, `aria-selected="true"`} {
		if !strings.Contains(out, want) {
			t.Errorf("htmlout dropped %s:\n%s", want, out)
		}
	}
}
