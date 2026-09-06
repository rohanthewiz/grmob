package verify

import (
	"regexp"
	"strings"
	"testing"
)

// core.Select on the two natives.
//
// The dispatch arm first, for the reason textgrid_test.go gives: both natives
// end their type dispatch in a catch-all that renders the children in a plain
// column, and a picker has no children — so a missing arm draws an empty box
// with no error anywhere.

func TestSwiftTypeDispatchDrawsSelect(t *testing.T) {
	syntax := swiftSwitch.with(swiftRenderer, "struct RenderNode:", "switch node.type {")
	requireArms(t, "Renderer.swift", "RenderNode", syntax.labels(t), "Select")
}

func TestKotlinTypeDispatchDrawsSelect(t *testing.T) {
	syntax := kotlinWhen.with(kotlinRenderer, "fun RenderNodeContent(", "when (node.type) {")
	requireArms(t, "Renderer.kt", "RenderNodeContent", syntax.labels(t), "Select")
}

// Neither renderer may build the picker from a platform picker control.
//
// This is the load-bearing half of a decision made on the *web*: htmlout's
// borderResetTypes writes `border: none` for a <select> whose Go style
// declares no frame, and that reset is only correct because the other three
// targets draw nothing but what the style asks for. SwiftUI's
// .pickerStyle(.menu) and Material's ExposedDropdownMenuBox each draw a frame,
// a container colour and an indicator of their own that no Go style can
// remove — so reaching for either here would silently make the picker the one
// control whose edge comes from the platform on two targets and from the theme
// on the other two.
//
// Checked as an absence on the composite's code rather than on the file,
// because the forbidden constructs are perfectly good elsewhere: nothing else
// in either renderer uses them today, but a future date or time picker very
// well might, and this check has no business reaching into it.
//
// The positive half — that the style's own box is what draws the control — is
// what the grMobBox / boxModifier substring says. Without it the negative half
// alone would pass on a renderer that drew no box at all.
func TestNativeSelectDrawsTheStylesBoxAndNotAPlatformPicker(t *testing.T) {
	for _, pin := range []struct {
		file, marker string
		next         *regexp.Regexp
		// box is the styling primitive the arm must draw through; platform
		// names the control it must not be built from; menu is the behavior
		// construct that supplies the list.
		box, platform, menu string
	}{
		{swiftRenderer, "private struct GrMobSelect", swiftCompositeStart,
			".grMobBox(", "pickerStyle", "Menu {"},
		{kotlinRenderer, "private fun GrMobSelect", kotlinCompositeStart,
			"boxModifier(", "ExposedDropdownMenuBox", "DropdownMenu("},
	} {
		body := dispatchArm(t, pin.file, pin.marker, pin.next)
		if !strings.Contains(body, pin.box) {
			t.Errorf("%s: %s does not draw through %s — the picker's frame, fill and radius "+
				"stop coming from the Go style, and htmlout's border reset for <select> is "+
				"resting on nothing", pin.file, pin.marker, pin.box)
		}
		if strings.Contains(body, pin.platform) {
			t.Errorf("%s: %s is built from %s, which draws a frame of its own that no Go style "+
				"can remove", pin.file, pin.marker, pin.platform)
		}
		if !strings.Contains(body, pin.menu) {
			t.Errorf("%s: %s builds no %s — the option list has nowhere to appear",
				pin.file, pin.marker, pin.menu)
		}
	}
}

// The choice goes up as the option's *value*, through the text channel.
//
// core.Select registers a func(string), so Go put the handler in the text
// callback map. A renderer dispatching the index or the label would reach Go
// with an ID that map has an entry for and a payload that means something
// else — the label case is the dangerous one, since a picker whose labels
// happen to equal its values would work in every test anyone wrote.
func TestNativeSelectDispatchesTheOptionValue(t *testing.T) {
	for _, pin := range []struct {
		file, marker string
		next         *regexp.Regexp
		dispatch     string
	}{
		{swiftRenderer, "private struct GrMobSelect", swiftCompositeStart, "textChanged("},
		{kotlinRenderer, "private fun GrMobSelect", kotlinCompositeStart, "textChanged("},
	} {
		body := dispatchArm(t, pin.file, pin.marker, pin.next)
		at := strings.Index(body, pin.dispatch)
		if at < 0 {
			t.Errorf("%s: %s does not call %s — a choice never reaches Go",
				pin.file, pin.marker, pin.dispatch)
			continue
		}
		// The argument that follows has to read the option's "value" key. Read
		// as text rather than parsed: the two languages spell the subscript
		// differently and there is nothing shared to compare against.
		rest := body[at:]
		if end := strings.Index(rest, "\n"); end > 0 {
			rest = rest[:end]
		}
		if !strings.Contains(rest, `"value"`) {
			t.Errorf("%s: %s dispatches %q, which does not read the option's value key",
				pin.file, pin.marker, strings.TrimSpace(rest))
		}
	}
}
