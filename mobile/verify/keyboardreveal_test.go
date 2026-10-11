package verify

import (
	"strings"
	"testing"
)

// iOS's UIKit field brings itself above the software keyboard (N-102).
//
// SwiftUI's ScrollView scrolls a focused field into view only when it is
// SwiftUI's own TextField. GrMob's field is a UITextField or UITextView in a
// representable, so nothing moved the page for it, and lesson 4.37's last
// grid row opened its editor under the keyboard on the iOS 26.5 simulator.
// GrMobKeyboardReveal does the scrolling. It is a silent no-op if the
// coordinator stops telling it which field is focused, or if it stops
// listening for the keyboard settling. TutorialRoundFourUITests'
// testTheLastGridRowsEditorIsAboveTheKeyboard checks the result on a
// simulator with the soft keyboard on; these hold the wiring under go test.
func TestIOSFieldRevealsItselfAboveTheKeyboard(t *testing.T) {
	file := nativeFile("ios", "GrMob", "Runtime", "GrMobTextInput.swift")
	src := codeIn(t, file)
	for _, pin := range []struct{ code, why string }{
		{"GrMobKeyboardReveal.shared.focused(input)",
			"the coordinator must say which field took focus, or nothing is revealed"},
		{"GrMobKeyboardReveal.shared.blurred(input)",
			"the coordinator must forget a field that lost focus, or a later keyboard move scrolls to it"},
		{"UIResponder.keyboardDidShowNotification",
			"the reveal must wait for the keyboard to settle, when SwiftUI has applied its inset"},
		{"UIResponder.keyboardDidChangeFrameNotification",
			"a keyboard that changes height (the suggestion strip, a different keyboard) must reveal again"},
		{"scroll.contentSize.height > scroll.bounds.height + 1",
			"only a scroller with height to give is moved; EditableGrid's horizontal strip is not"},
	} {
		if !strings.Contains(src, pin.code) {
			t.Errorf("%s: lacks %q — %s", file, pin.code, pin.why)
		}
	}
}
