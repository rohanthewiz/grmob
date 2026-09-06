package htmlout

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// selectNode builds a picker node in the wire shape core.Select produces.
func selectNode(value string, style *core.Style, opts ...map[string]string) *core.Node {
	return &core.Node{
		Type:  "Select",
		Style: style,
		Props: map[string]any{"value": value, "options": opts},
	}
}

func opt(value, label string) map[string]string {
	return map[string]string{"value": value, "label": label}
}

// The options come from the props, and the chosen one is marked on the option
// rather than on the element — <select> has no value attribute, so a value
// written there would be inert and the picker would open on its first entry.
func TestSelectWritesItsOptionsAndMarksTheChosenOne(t *testing.T) {
	out := ExportHTML(selectNode("pt", nil,
		opt("us", "United States"),
		opt("pt", "Portugal"),
	))

	if !strings.Contains(out, "<select") {
		t.Fatalf("not exported as a <select>:\n%s", out)
	}
	if !strings.Contains(out, `<option value="pt" selected="selected">Portugal</option>`) {
		t.Errorf("the chosen option is not marked:\n%s", out)
	}
	if strings.Contains(out, `<option value="us" selected="selected"`) {
		t.Errorf("an unchosen option is marked:\n%s", out)
	}
	if strings.Contains(out, `value="pt"`) && strings.Contains(out, `<select value=`) {
		t.Errorf("a value attribute was written on the element, where it is inert:\n%s", out)
	}
}

// A value matching nothing marks nothing, which is what a live <select> does
// with an out-of-list value: it shows the first option. Degrading the same way
// on both web targets matters more than inventing a better answer on one.
func TestSelectWithAnUnmatchedValueMarksNothing(t *testing.T) {
	out := ExportHTML(selectNode("zz", nil, opt("us", "United States"), opt("pt", "Portugal")))
	if strings.Contains(out, "selected") {
		t.Errorf("an unmatched value marked an option:\n%s", out)
	}
}

// Option labels and values are as user-originated as any other content here —
// a country list read from a server is the normal case — so both halves go out
// escaped. The value travels the attribute path (a double quote is the
// breakout character) and the label the text path.
func TestSelectOptionsAreEscaped(t *testing.T) {
	out := ExportHTML(selectNode("", nil,
		opt(`" onfocus="steal()`, `<script>alert(1)</script>`),
	))
	if strings.Contains(out, `" onfocus="`) {
		t.Errorf("an option value broke out of its attribute:\n%s", out)
	}
	if strings.Contains(out, "<script>") {
		t.Errorf("an option label rendered as live markup:\n%s", out)
	}
}

// The picker is a form control, which is what makes HTML's own disabled
// attribute legal on it — the pointer-events fallback a disabled container
// gets would leave the control focusable and silently unannounced.
func TestADisabledSelectIsDisabledTheHTMLWay(t *testing.T) {
	out := ExportHTML(selectNode("", &core.Style{Disabled: true}, opt("a", "A")))
	if !strings.Contains(out, `disabled="disabled"`) {
		t.Errorf("no disabled attribute:\n%s", out)
	}
	if strings.Contains(out, "pointer-events:none") {
		t.Errorf("the container fallback was used on a form control:\n%s", out)
	}
}

// The border decision this node type was added to settle: the Go style owns
// the frame, so a picker with no border in its style is talked out of the
// browser's, exactly as a text field is.
//
// Both directions, because a reset that swallowed a real border would be the
// opposite bug and just as silent.
func TestSelectFrameIsTheStylesAndNotTheBrowsers(t *testing.T) {
	bare := ExportHTML(selectNode("", nil, opt("a", "A")))
	if !strings.Contains(bare, "border:none") {
		t.Errorf("a picker with no border in its style keeps the browser's:\n%s", bare)
	}

	framed := ExportHTML(selectNode("", &core.Style{BorderWidth: 1, BorderColor: "#8E8E93"}, opt("a", "A")))
	if !strings.Contains(framed, "border:1px solid #8E8E93") {
		t.Errorf("the style's own frame was swallowed:\n%s", framed)
	}
	if strings.Contains(framed, "border:none") {
		t.Errorf("the reset fired on a picker that asked for a frame:\n%s", framed)
	}
}

// A themed picker draws the field frame the theme states, which is the whole
// premise of the reset above: there is something to reset *to*. This is the
// end-to-end version — a real core.Select rendered against a real theme —
// where the tests above build nodes by hand.
func TestAThemedPickerWearsTheThemesFieldFrame(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	out := ExportHTML(core.Select("pt", []core.SelectOption{
		{Value: "pt", Label: "Portugal"},
	}, func(string) {}).Render(ctx))

	base := core.DefaultTheme.Components.Input
	if !strings.Contains(out, "border:1px solid "+base.BorderColor) {
		t.Errorf("a themed picker does not wear the theme's field frame %q:\n%s", base.BorderColor, out)
	}
	if strings.Contains(out, "border:none") {
		t.Errorf("the reset fired on a themed picker:\n%s", out)
	}
	// And the change callback is recorded, like every other control's.
	if !strings.Contains(out, "data-onchange=") {
		t.Errorf("no change callback ID was exported:\n%s", out)
	}
}
