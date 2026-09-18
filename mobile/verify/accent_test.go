package verify

import (
	"strings"
	"testing"
)

// core.Style.AccentColor, held against both native renderers.
//
// The field is the one colour a platform-drawn control takes from Go: a
// Switch's on track, a Slider's filled track, a Checkbox's box. Both web
// targets write it as CSS accent-color. Each native has to carry it to its
// own tint slot, and each link fails silently: an unread JSON key compiles,
// and so does a parsed colour nothing hands to the control. The symptom is a
// control in the platform's colour (Material purple, iOS green) in an app
// whose theme says otherwise, which is what the device pass found.

// The style field itself, on both.
func TestBothNativeParsersReadTheAccentField(t *testing.T) {
	for _, pin := range []struct{ file, key string }{
		{swiftStyle, `str("AccentColor")`},
		{kotlinStyle, `optString("AccentColor")`},
	} {
		if src := valuesIn(t, pin.file); !strings.Contains(src, pin.key) {
			t.Errorf("%s: never parses %s — core.AccentColor reaches both web targets "+
				"and does nothing on this platform", pin.file, pin.key)
		}
	}
}

// And each control that takes the accent hands it to the platform's slot.
// SwiftUI has one modifier for all three; Compose has a colours object per
// control, and a control left out keeps Material's purple.
func TestBothNativesTintTheirControlsWithTheAccent(t *testing.T) {
	swift := codeIn(t, swiftRenderer)
	// Checkbox and Switch are both Toggles on iOS, and Slider is the third.
	if n := strings.Count(swift, ".tint(node.style?.accentColor)"); n != 3 {
		t.Errorf("%s: .tint(node.style?.accentColor) appears %d times, want 3 "+
			"(GrMobCheckbox, GrMobSwitch, GrMobSlider)", swiftRenderer, n)
	}

	kotlin := codeIn(t, kotlinRenderer)
	for _, pin := range []struct{ expr, why string }{
		{"CheckboxDefaults.colors(checkedColor = it)", "the Checkbox's checked box"},
		{"SwitchDefaults.colors(checkedTrackColor = it", "the Switch's on track"},
		{"SliderDefaults.colors(", "the Slider's thumb and tracks"},
		{"activeTrackColor = it", "the Slider's filled track"},
	} {
		if !strings.Contains(kotlin, pin.expr) {
			t.Errorf("%s: %q not found — %s would stay in Material's colour", kotlinRenderer, pin.expr, pin.why)
		}
	}
}
