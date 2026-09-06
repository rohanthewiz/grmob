package htmlout

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// Nodes are built directly rather than through core views: the exporter's
// contract is Node-in/HTML-out, and going through views would drag a Context
// into tests that exercise none of its behavior.

func textNode(content string) *core.Node {
	return &core.Node{Type: "Text", Props: map[string]any{"content": content}}
}

// --- Escaping ---------------------------------------------------------------
//
// The exporter emits user-originated strings (text content, input values,
// labels, srcs) into markup. Anything that arrives from state or a data source
// must not be interpretable as markup on the way out.

func TestTextContentIsEscaped(t *testing.T) {
	out := ExportHTML(textNode(`<script>alert(1)</script>`))
	if strings.Contains(out, "<script>") {
		t.Fatalf("text content rendered as live markup:\n%s", out)
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Fatalf("expected entity-escaped content, got:\n%s", out)
	}
}

func TestButtonLabelIsEscaped(t *testing.T) {
	n := &core.Node{Type: "Button", Props: map[string]any{"label": `<b>bold</b>`}}
	out := ExportHTML(n)
	if strings.Contains(out, "<b>") {
		t.Fatalf("button label rendered as live markup:\n%s", out)
	}
}

func TestTextAreaValueCannotCloseTheElement(t *testing.T) {
	n := &core.Node{Type: "TextArea", Props: map[string]any{"value": `</textarea><script>x()</script>`}}
	out := ExportHTML(n)
	if strings.Contains(out, "<script>") {
		t.Fatalf("textarea value broke out of the element:\n%s", out)
	}
}

// A double quote in an attribute value is the breakout character: unescaped,
// everything after it parses as further attributes on the element.
func TestInputValueCannotInjectAttributes(t *testing.T) {
	n := &core.Node{Type: "Input", Props: map[string]any{
		"value":       `" onfocus="steal()`,
		"placeholder": "name",
	}}
	out := ExportHTML(n)
	if strings.Contains(out, `" onfocus="`) {
		t.Fatalf("input value broke out of its attribute:\n%s", out)
	}
}

func TestPlaceholderCannotInjectAttributes(t *testing.T) {
	n := &core.Node{Type: "Input", Props: map[string]any{
		"value":       "",
		"placeholder": `" onmouseover="pwn()`,
	}}
	out := ExportHTML(n)
	if strings.Contains(out, `" onmouseover="`) {
		t.Fatalf("placeholder broke out of its attribute:\n%s", out)
	}
}

func TestImageSrcCannotInjectAttributes(t *testing.T) {
	n := &core.Node{Type: "Image", Props: map[string]any{"src": `x" onerror="pwn()`}}
	out := ExportHTML(n)
	if strings.Contains(out, `" onerror="`) {
		t.Fatalf("img src broke out of its attribute:\n%s", out)
	}
}

// --- Structure --------------------------------------------------------------

func TestDocumentWrapper(t *testing.T) {
	out := ExportHTML(textNode("hi"))
	if !strings.HasPrefix(out, "<!DOCTYPE html>") {
		t.Fatalf("missing doctype:\n%s", out)
	}
	for _, want := range []string{`<html lang="en">`, "<body>", "</body>", "</html>"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestContainerWrapsChildren(t *testing.T) {
	n := &core.Node{Type: "Column", Children: []*core.Node{textNode("a"), textNode("b")}}
	out := ExportHTML(n)
	if got := strings.Count(out, "<span"); got != 2 {
		t.Fatalf("expected 2 spans, got %d:\n%s", got, out)
	}
	if !strings.Contains(out, "<div") || !strings.Contains(out, "</div>") {
		t.Fatalf("column did not render as a div:\n%s", out)
	}
}

// A Spacer is a square void on both natives (Compose Spacer(Modifier.size),
// SwiftUI Color.clear.frame(width:height:)), so both axes have to be sized
// here. Height alone — what this emitted for the whole life of the exporter —
// made a Spacer inside a Row a zero-width box, invisible on the web and n
// points wide on device.
func TestSpacerSizesBothAxes(t *testing.T) {
	n := &core.Node{Type: "Spacer", Props: map[string]any{"size": 20}}
	out := ExportHTML(n)
	for _, want := range []string{"width:20px", "height:20px", "flex-shrink:0"} {
		if !strings.Contains(out, want) {
			t.Fatalf("spacer missing %q:\n%s", want, out)
		}
	}
}

func TestCheckboxChecked(t *testing.T) {
	n := &core.Node{Type: "Checkbox", Props: map[string]any{"checked": true}}
	out := ExportHTML(n)
	if !strings.Contains(out, `type="checkbox"`) || !strings.Contains(out, "checked") {
		t.Fatalf("checkbox not rendered checked:\n%s", out)
	}
}

// A Slider is a range input whose bounds ride as attributes, in the shortest
// form that round-trips (no "30.000000").
func TestSliderExportsRangeAttributes(t *testing.T) {
	n := &core.Node{Type: "Slider", Props: map[string]any{"value": 12.5, "min": 0.0, "max": 30.0}}
	out := ExportHTML(n)
	for _, want := range []string{`type="range"`, `min="0"`, `max="30"`, `value="12.5"`} {
		if !strings.Contains(out, want) {
			t.Errorf("slider export lacks %s:\n%s", want, out)
		}
	}
	if strings.Contains(out, "step=") {
		t.Errorf("unset step was exported:\n%s", out)
	}
	n.Props["step"] = 0.5
	if out := ExportHTML(n); !strings.Contains(out, `step="0.5"`) {
		t.Errorf("step not exported:\n%s", out)
	}
}

// Callback IDs ride out as data attributes; the WASM runtime dispatches on
// them, so their names are a contract.
func TestCallbackAttrsEmitted(t *testing.T) {
	n := &core.Node{
		Type:     "Button",
		Props:    map[string]any{"label": "Go", "onClick": "cb_0"},
		Children: nil,
	}
	out := ExportHTML(n)
	if !strings.Contains(out, `data-onclick="cb_0"`) {
		t.Fatalf("missing data-onclick attribute:\n%s", out)
	}
}

// The focus edges ride out the same way. They matter more than the others
// here because they are the only events the export can carry that the native
// renderers wire on one node type only — a browser focuses far more than a
// phone does, and the export must not second-guess that.
func TestFocusCallbackAttrsEmitted(t *testing.T) {
	n := &core.Node{
		Type: "Input",
		Props: map[string]any{
			"value":   "",
			"onFocus": "cb_3",
			"onBlur":  "cb_4",
		},
	}
	out := ExportHTML(n)
	if !strings.Contains(out, `data-onfocus="cb_3"`) {
		t.Fatalf("missing data-onfocus attribute:\n%s", out)
	}
	if !strings.Contains(out, `data-onblur="cb_4"`) {
		t.Fatalf("missing data-onblur attribute:\n%s", out)
	}
}

// A focus command exports as autofocus, and only as autofocus. The epoch is a
// runtime coordination number — it says *when*, which a snapshot cannot ask —
// so it never reaches the document; what survives is the standing instruction
// the last command left behind, and HTML already spells that.
func TestFocusCommandExportsAsAutofocus(t *testing.T) {
	n := &core.Node{
		Type: "Input",
		Props: map[string]any{
			"value":       "",
			"focusEpoch":  4,
			"focusAction": "focus",
		},
	}
	out := ExportHTML(n)
	if !strings.Contains(out, `autofocus="autofocus"`) {
		t.Fatalf("missing autofocus attribute:\n%s", out)
	}
	if strings.Contains(out, "focusEpoch") || strings.Contains(out, "focusAction") {
		t.Fatalf("the raw command props leaked into the document:\n%s", out)
	}
}

// The two actions that are not an instruction to a freshly loaded page. A
// blur in particular must not export as anything: there is no focus to
// release before the user has touched the document.
func TestNonFocusCommandsExportNothing(t *testing.T) {
	for _, action := range []string{"blur", ""} {
		n := &core.Node{
			Type: "Input",
			Props: map[string]any{
				"value":       "",
				"focusEpoch":  4,
				"focusAction": action,
			},
		}
		if out := ExportHTML(n); strings.Contains(out, "autofocus") {
			t.Errorf("action %q exported autofocus:\n%s", action, out)
		}
	}
}

// Attribute order is part of the contract: the mapping is a slice rather than
// a map precisely so a re-run produces a byte-identical document.
func TestCallbackAttrOrderIsStable(t *testing.T) {
	n := &core.Node{
		Type: "Input",
		Props: map[string]any{
			"onChange": "txt_cb_0",
			"onFocus":  "cb_1",
			"onBlur":   "cb_2",
		},
	}
	first := ExportHTML(n)
	for i := 0; i < 5; i++ {
		if got := ExportHTML(n); got != first {
			t.Fatalf("export is not deterministic:\n%s\nvs\n%s", first, got)
		}
	}
}

func TestStyleAttrSerialized(t *testing.T) {
	n := &core.Node{
		Type:  "Text",
		Props: map[string]any{"content": "styled"},
		Style: &core.Style{TextColor: "#333333", FontSize: 15},
	}
	out := ExportHTML(n)
	if !strings.Contains(out, "color:#333333") || !strings.Contains(out, "font-size:15px") {
		t.Fatalf("style attribute not serialized:\n%s", out)
	}
}

// FontWeight went unemitted on both DOM targets for the whole life of this
// exporter — core.Bold rendered as regular text on the web while both natives
// honored it — and no snapshot happened to carry the field, which is how the
// gap survived. This test pins the emission (the Weight constants are literal
// CSS font-weight numbers) and the zero-value silence in one place.
func TestFontWeightSerialized(t *testing.T) {
	bold := &core.Node{
		Type:  "Text",
		Props: map[string]any{"content": "styled"},
		Style: &core.Style{FontWeight: core.Bold},
	}
	if out := ExportHTML(bold); !strings.Contains(out, "font-weight:700") {
		t.Fatalf("FontWeight not serialized:\n%s", out)
	}
	unset := &core.Node{
		Type:  "Text",
		Props: map[string]any{"content": "styled"},
		Style: &core.Style{FontSize: 15},
	}
	if out := ExportHTML(unset); strings.Contains(out, "font-weight") {
		t.Fatalf("zero FontWeight must emit nothing:\n%s", out)
	}
}

// Border needs both halves, matching the native renderers: Compose draws its
// Modifier.border only when width > 0 and color != null, so a half-specified
// border must draw nothing here too rather than falling back to CSS's default
// medium/currentColor.
func TestBorderNeedsBothWidthAndColor(t *testing.T) {
	cases := []struct {
		name  string
		style core.Style
		want  string // "" means no border declaration at all
	}{
		{"both", core.Style{BorderWidth: 1, BorderColor: "#FF3B30"}, "border:1px solid #FF3B30"},
		{"fractional", core.Style{BorderWidth: 0.5, BorderColor: "#000"}, "border:0.5px solid #000"},
		{"width only", core.Style{BorderWidth: 2}, ""},
		{"color only", core.Style{BorderColor: "#FF3B30"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := c.style
			out := ExportHTML(&core.Node{Type: "Box", Style: &s})
			if c.want == "" {
				if strings.Contains(out, "border:") {
					t.Fatalf("half-specified border was emitted: %s", out)
				}
				return
			}
			if !strings.Contains(out, c.want) {
				t.Fatalf("want %q in output, got: %s", c.want, out)
			}
		})
	}
}

// --- Grouping nodes ---------------------------------------------------------
//
// Fragment (what core.For wraps its generated children in) and Theme (what
// core.WithTheme wraps a subtree in) carry no Style and have no visual box.
// Both native renderers already inline them into the parent's layout scope,
// so the exporter has to as well or the HTML disagrees with what ships.

func TestFragmentEmitsNoWrapperElement(t *testing.T) {
	n := &core.Node{Type: "Fragment", Children: []*core.Node{
		textNode("first"), textNode("second"),
	}}

	out := ExportHTML(n)
	if strings.Contains(out, "<div>") {
		t.Fatalf("Fragment emitted a wrapper element:\n%s", out)
	}
	if !strings.Contains(out, "first") || !strings.Contains(out, "second") {
		t.Fatalf("Fragment dropped a child:\n%s", out)
	}
}

func TestThemeEmitsNoWrapperElement(t *testing.T) {
	n := &core.Node{Type: "Theme", Props: map[string]any{}, Children: []*core.Node{
		textNode("themed"),
	}}

	out := ExportHTML(n)
	if strings.Contains(out, "<div>") {
		t.Fatalf("Theme emitted a wrapper element:\n%s", out)
	}
	if !strings.Contains(out, "themed") {
		t.Fatalf("Theme dropped its child:\n%s", out)
	}
}

// The bug this fixes, stated as the behavior a user can see. A wrapper around
// the generated children becomes the container's single flex item, so the
// container's gap has nothing to space and the children it was meant to
// separate sit flush against each other. Pinning it on child *count* rather
// than on the absence of a div is what makes it a layout test: it is the
// number of things the flex container can see that was wrong.
func TestFragmentChildrenBecomeSiblingsOfTheFlexContainer(t *testing.T) {
	row := &core.Node{
		Type:  "Row",
		Style: &core.Style{Gap: 8, Display: core.DisplayFlex},
		Children: []*core.Node{
			{Type: "Fragment", Children: []*core.Node{
				{Type: "Button", Props: map[string]any{"label": "All"}},
				{Type: "Button", Props: map[string]any{"label": "Active"}},
				{Type: "Button", Props: map[string]any{"label": "Done"}},
			}},
		},
	}

	out := ExportHTML(row)
	if n := strings.Count(out, "<button"); n != 3 {
		t.Fatalf("want 3 buttons, got %d:\n%s", n, out)
	}
	// The three buttons must be direct children of the gapped row: nothing
	// may sit between the row's opening tag and the first button.
	i := strings.Index(out, "gap:8px")
	if i < 0 {
		t.Fatalf("row lost its gap:\n%s", out)
	}
	between := out[strings.Index(out[i:], ">")+i : strings.Index(out, "<button")]
	if strings.Contains(between, "<") {
		t.Fatalf("an element sits between the gapped row and its first button (%q):\n%s", between, out)
	}
}

// Nested grouping nodes flatten all the way down — a For inside a For, or a
// For inside a WithTheme, must not reintroduce a box at any level.
func TestNestedGroupingNodesFlattenCompletely(t *testing.T) {
	n := &core.Node{Type: "Theme", Children: []*core.Node{
		{Type: "Fragment", Children: []*core.Node{
			{Type: "Fragment", Children: []*core.Node{textNode("deep")}},
		}},
	}}

	out := ExportHTML(n)
	if strings.Contains(out, "<div>") {
		t.Fatalf("nested grouping nodes still emitted a wrapper:\n%s", out)
	}
	if !strings.Contains(out, "deep") {
		t.Fatalf("nested grouping nodes dropped the child:\n%s", out)
	}
}

// --- ContentMode and the disabled state -------------------------------------
//
// Both are renderer-level concepts that the export has to reproduce for the
// HTML target to agree with what ships on device: an Image scales the same
// way, and a disabled control is inert and announced as such.

func TestImageContentModeBecomesObjectFit(t *testing.T) {
	for _, tc := range []struct {
		mode core.ContentMode
		want string
	}{
		{core.ContentModeFit, "object-fit:contain"},
		{core.ContentModeFill, "object-fit:cover"},
		{core.ContentModeStretch, "object-fit:fill"},
		{core.ContentModeCenter, "object-fit:none"},
	} {
		t.Run(string(tc.mode), func(t *testing.T) {
			n := &core.Node{Type: "Image", Props: map[string]any{
				"src": "a.png", "contentMode": string(tc.mode),
			}}
			if out := ExportHTML(n); !strings.Contains(out, tc.want) {
				t.Fatalf("mode %q did not export %q:\n%s", tc.mode, tc.want, out)
			}
		})
	}
}

// The default has to stay absent rather than become object-fit:contain: a
// mode-less Image is the pre-ContentMode output, and every existing snapshot
// pins it.
func TestImageWithoutContentModeEmitsNoObjectFit(t *testing.T) {
	n := &core.Node{Type: "Image", Props: map[string]any{"src": "a.png"}}
	if out := ExportHTML(n); strings.Contains(out, "object-fit") {
		t.Fatalf("a mode-less image emitted object-fit:\n%s", out)
	}
}

// An unrecognized mode must not leak an invalid declaration into the style
// attribute — a malformed declaration can invalidate the ones around it.
func TestUnknownContentModeIsDropped(t *testing.T) {
	n := &core.Node{Type: "Image", Props: map[string]any{
		"src": "a.png", "contentMode": "sideways",
	}}
	if out := ExportHTML(n); strings.Contains(out, "object-fit") {
		t.Fatalf("unknown mode reached the output:\n%s", out)
	}
}

// The object-fit declaration has to join the node's existing style attribute.
// An element carries exactly one; a second would be dropped by the parser,
// silently taking the box styling with it.
func TestContentModeJoinsTheExistingStyleAttribute(t *testing.T) {
	n := &core.Node{
		Type:  "Image",
		Props: map[string]any{"src": "a.png", "contentMode": string(core.ContentModeFill)},
		Style: &core.Style{Width: "80px", Height: "80px"},
	}
	out := ExportHTML(n)
	if strings.Count(out, `style=`) != 1 {
		t.Fatalf("expected exactly one style attribute:\n%s", out)
	}
	for _, want := range []string{"width:80px", "height:80px", "object-fit:cover"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
}

func TestDisabledFormControlsGetTheAttribute(t *testing.T) {
	for _, nodeType := range []string{"Button", "Input", "InputPassword", "NumericInput", "TextArea", "Checkbox", "Slider"} {
		t.Run(nodeType, func(t *testing.T) {
			n := &core.Node{
				Type:  nodeType,
				Props: map[string]any{"label": "Send", "value": ""},
				Style: &core.Style{Disabled: true},
			}
			out := ExportHTML(n)
			if !strings.Contains(out, `disabled="disabled"`) {
				t.Fatalf("%s did not export the disabled attribute:\n%s", nodeType, out)
			}
		})
	}
}

func TestEnabledControlHasNoDisabledAttribute(t *testing.T) {
	n := &core.Node{Type: "Button", Props: map[string]any{"label": "Send"}, Style: &core.Style{}}
	if out := ExportHTML(n); strings.Contains(out, "disabled") {
		t.Fatalf("an enabled button exported a disabled state:\n%s", out)
	}
}

// A container is a div, where `disabled` is not a valid attribute and would be
// ignored. The pair below is what actually reproduces the native behavior: the
// ARIA state for the screen reader, and pointer-events for the taps that
// Compose and SwiftUI stop dispatching.
func TestDisabledContainerGetsAriaAndPointerEvents(t *testing.T) {
	n := &core.Node{
		Type:  "Row",
		Props: map[string]any{"onClick": "cb-1"},
		Style: &core.Style{Disabled: true},
	}
	out := ExportHTML(n)
	if !strings.Contains(out, `aria-disabled="true"`) {
		t.Fatalf("no aria-disabled on a disabled container:\n%s", out)
	}
	if !strings.Contains(out, "pointer-events:none") {
		t.Fatalf("a disabled container still accepts pointer events:\n%s", out)
	}
	if strings.Contains(out, `disabled="disabled"`) {
		t.Fatalf("a div got the form-control disabled attribute:\n%s", out)
	}
	// The callback ID stays: Go keeps the handler registered, and the
	// platform — here the browser — is what refuses to dispatch.
	if !strings.Contains(out, "cb-1") {
		t.Fatalf("the disabled node dropped its callback ID:\n%s", out)
	}
}

// The stretch value needs no special casing — AlignItems is emitted verbatim
// and "stretch" is valid CSS — but the flex block it rides in is conditional,
// so a node that sets *only* AlignItems must still become a flex container.
func TestAlignItemsStretchMakesTheContainerFlex(t *testing.T) {
	n := &core.Node{Type: "Row", Style: &core.Style{AlignItems: core.AlignItemsStretch}}
	out := ExportHTML(n)
	for _, want := range []string{"display:flex", "flex-direction:row", "align-items:stretch"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
}

// --- Display against the flex container --------------------------------------
//
// Style.Display used to be emitted after the flex declarations so that an
// "explicit" Display would win the browser's last-declaration-wins parse. But
// the style merge erases who set what, and DefaultTheme's Card carries
// Display: block — so a themed Card's own theme killed the align-items its
// author asked for, on this exporter alone (the natives read Display only to
// honor "none"; the WASM runtime emits no Display at all). These pin the
// resolution: a flex container stays one, "none" alone still wins.

// The bug as a user saw it. The theme premise is asserted rather than assumed,
// because the test's story — "the Card's OWN theme style fights its
// alignment" — is only true while the theme actually sets block.
func TestThemedCardDisplayBlockDoesNotKillItsAlignItems(t *testing.T) {
	if core.DefaultTheme.Components.Card.Display != core.DisplayBlock {
		t.Fatalf("premise gone: DefaultTheme's Card no longer sets Display: block; " +
			"rewrite or retire this test")
	}
	// The merged style a core.Card(core.AlignItemsProp(...)) produces:
	// containerNode starts from the theme's Card style, the prop lands on top.
	merged := core.DefaultTheme.Components.Card
	merged.AlignItems = core.AlignItemsCenter
	n := &core.Node{Type: "Card", Style: &merged}

	out := ExportHTML(n)
	for _, want := range []string{"display:flex", "align-items:center"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "display:block") {
		t.Fatalf("the theme's Display: block was emitted after the flex container and kills it:\n%s", out)
	}
}

// The same conflict through the Align fallback — the themed Card was the case
// that surfaced it, and the fallback path computes align-items later than the
// explicit prop does, so it gets its own pin.
func TestAlignFallbackSurvivesDisplayBlock(t *testing.T) {
	n := &core.Node{Type: "Card", Style: &core.Style{
		Display: core.DisplayBlock,
		Align:   core.AlignCenter,
	}}
	out := ExportHTML(n)
	for _, want := range []string{"display:flex", "align-items:center"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "display:block") {
		t.Fatalf("Display: block beat the Align fallback's flex container:\n%s", out)
	}
}

// "none" is the one mode that still beats the container: it is the only
// Display value either native reads (both bail out before layout), so hiding
// must win over flex here too. Order matters — the browser honors whichever
// valid declaration comes last — so the test pins the order, not just the
// presence.
func TestDisplayNoneStillHidesAFlexContainer(t *testing.T) {
	n := &core.Node{Type: "Column", Style: &core.Style{
		Display: core.DisplayNone,
		Gap:     8,
	}}
	out := ExportHTML(n)
	flex := strings.Index(out, "display:flex")
	none := strings.Index(out, "display:none")
	if flex < 0 || none < 0 {
		t.Fatalf("want both display:flex and display:none, got:\n%s", out)
	}
	if none < flex {
		t.Fatalf("display:none precedes display:flex, so the node is not hidden:\n%s", out)
	}
}

// Display: inline on a flex container folds into inline-flex — the one CSS
// spelling that keeps both the inline level and the flex layout — rather than
// either declaration overwriting the other.
func TestDisplayInlineBecomesInlineFlexOnAFlexContainer(t *testing.T) {
	n := &core.Node{Type: "Row", Style: &core.Style{
		Display: core.DisplayInline,
		Gap:     8,
	}}
	out := ExportHTML(n)
	if !strings.Contains(out, "display:inline-flex") {
		t.Fatalf("missing display:inline-flex:\n%s", out)
	}
	// Exactly one display declaration: the fold replaces both candidates, and
	// a stray second one would reopen the last-declaration-wins fight.
	if n := strings.Count(out, "display:"); n != 1 {
		t.Fatalf("want exactly 1 display declaration, got %d:\n%s", n, out)
	}
}

// A node the flex block never triggers on keeps the verbatim emission the
// exporter has always produced — the resolution above is strictly about the
// conflict, not a new opinion on Display.
//
// A Text and not a Column, as this used to be: every stack container is a
// flex container now whether or not its Style asks (stackAxes), so a Column
// can no longer be a non-flex node. Which makes the type the case needs one
// outside that table — and Text is the honest choice anyway, since a node
// carrying nothing but Display is not a container in spirit either.
func TestDisplayStaysVerbatimOnNonFlexNodes(t *testing.T) {
	n := &core.Node{Type: "Text", Props: map[string]any{"content": "hi"},
		Style: &core.Style{Display: core.DisplayBlock}}
	out := ExportHTML(n)
	if !strings.Contains(out, "display:block") {
		t.Fatalf("missing display:block:\n%s", out)
	}
	if strings.Contains(out, "display:flex") {
		t.Fatalf("a Display alone must not create a flex container:\n%s", out)
	}
}

// core.UseFocusOrder's keyboard action survives the export where the focus
// *command* does not, and for a reason worth stating: the hint is a standing
// property of the field ("this key means next"), not a moment in time, and a
// static document can carry standing properties perfectly well.
func TestImeActionExportsAsEnterKeyHint(t *testing.T) {
	n := &core.Node{
		Type: "Input",
		Props: map[string]any{
			"value":     "",
			"imeAction": "next",
			"onSubmit":  "cb_7",
		},
	}
	out := ExportHTML(n)
	if !strings.Contains(out, `enterkeyhint="next"`) {
		t.Fatalf("missing enterkeyhint attribute:\n%s", out)
	}
	// The behavior travels as the callback ID, because a soft keyboard's hint
	// only relabels the key — a hardware keyboard ignores it entirely.
	if !strings.Contains(out, `data-onsubmit="cb_7"`) {
		t.Fatalf("missing data-onsubmit attribute:\n%s", out)
	}
	if strings.Contains(out, "imeAction") {
		t.Fatalf("the raw imeAction prop leaked into the document:\n%s", out)
	}
}

// A field that acts on return but does not advance says so as "done",
// mirroring Android's ImeAction.Done and SwiftUI's .done. This is the shape
// InputWithSubmit has always had, and the last field of a focus order.
func TestSubmitWithoutTraversalExportsDone(t *testing.T) {
	n := &core.Node{
		Type: "Input",
		Props: map[string]any{
			"value":     "",
			"imeAction": "",
			"onSubmit":  "cb_2",
		},
	}
	if out := ExportHTML(n); !strings.Contains(out, `enterkeyhint="done"`) {
		t.Fatalf("a submitting field did not export enterkeyhint=done:\n%s", out)
	}
}

// A field with neither exports no hint at all. enterkeyhint has no empty
// value: the attribute's absence is how HTML spells "the browser's default
// return key", and writing enterkeyhint="" would be an invalid document.
func TestNoActionExportsNoEnterKeyHint(t *testing.T) {
	n := &core.Node{
		Type:  "Input",
		Props: map[string]any{"value": "", "imeAction": ""},
	}
	if out := ExportHTML(n); strings.Contains(out, "enterkeyhint") {
		t.Fatalf("a field with no submit action exported a hint:\n%s", out)
	}
}

// InputTypes hands out a copy, which matters because the table it copies is a
// package-level map: an importer that mutated the returned map would
// otherwise be rewriting what every export renders. The conformance test in
// wasm/verify is the caller this contract exists for — it deletes entries as
// it matches them.
func TestInputTypesReturnsACopy(t *testing.T) {
	table := InputTypes()
	if table["Checkbox"] != "checkbox" {
		t.Fatalf("Checkbox = %q, want checkbox", table["Checkbox"])
	}
	delete(table, "Checkbox")
	table["Input"] = "mutated"

	if got := InputTypeFor("Checkbox"); got != "checkbox" {
		t.Errorf("after deleting from the copy, InputTypeFor(Checkbox) = %q", got)
	}
	if got := InputTypeFor("Input"); got != "text" {
		t.Errorf("after writing to the copy, InputTypeFor(Input) = %q", got)
	}
}

// --- Phase 3 parity: Style fields the natives read and the web pair did not ---
//
// Every test below covers a field that was declared in Go, honored by Compose
// and SwiftUI, and silently dropped by this exporter. They are grouped rather
// than scattered because they share one failure mode: the field applies
// cleanly, the export succeeds, and the only symptom is that the web target
// disagrees with the device.

// The shorthand rule, stated once here from the contract rather than read off
// EdgeCSS: an explicit side wins, an unset (zero) side takes its axis's
// shorthand. Both natives resolve it this way (parseEdges in GrMobStyle.swift
// and GrMobStyle.kt); until EdgeCSS existed this exporter read the four
// per-side fields only, so core.PaddingHorizontal(16) rendered as 16px of
// padding on device and as nothing in the browser.
func TestEdgeShorthandFillsUnsetSides(t *testing.T) {
	cases := []struct {
		name string
		in   core.EdgeInsets
		want string
	}{
		{"per-side only", core.EdgeInsets{Top: 1, Right: 2, Bottom: 3, Left: 4}, "1px 2px 3px 4px"},
		{"horizontal only", core.EdgeInsets{Horizontal: 8}, "0px 8px 0px 8px"},
		{"vertical only", core.EdgeInsets{Vertical: 6}, "6px 0px 6px 0px"},
		{"both shorthands", core.EdgeInsets{Horizontal: 8, Vertical: 6}, "6px 8px 6px 8px"},
		// The explicit side wins over the shorthand for its own axis and
		// leaves the opposite side on the shorthand.
		{"explicit beats shorthand", core.EdgeInsets{Horizontal: 8, Left: 20}, "0px 8px 0px 20px"},
		{"empty", core.EdgeInsets{}, "0px 0px 0px 0px"},
	}
	for _, c := range cases {
		if got := EdgeCSS(c.in); got != c.want {
			t.Errorf("%s: EdgeCSS(%+v) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

func TestPaddingAndMarginShorthandReachTheStyleAttribute(t *testing.T) {
	n := &core.Node{
		Type: "Box",
		Style: &core.Style{
			Padding: core.EdgeInsets{Horizontal: 16},
			Margin:  core.EdgeInsets{Vertical: 8},
		},
	}
	out := ExportHTML(n)
	if !strings.Contains(out, "padding:0px 16px 0px 16px") {
		t.Fatalf("PaddingHorizontal not resolved:\n%s", out)
	}
	if !strings.Contains(out, "margin:8px 0px 8px 0px") {
		t.Fatalf("MarginVertical not resolved:\n%s", out)
	}
}

// The elevation-to-box-shadow arithmetic is SwiftUI's mapping restated (blur =
// elevation/2, y offset = elevation/3), so one core.Shadow(6) draws a
// comparable shadow on the three targets that draw one at all.
func TestShadowBecomesBoxShadow(t *testing.T) {
	n := &core.Node{Type: "Card", Style: &core.Style{Shadow: 6}}
	if out := ExportHTML(n); !strings.Contains(out, "box-shadow:0 2px 3px rgba(0,0,0,0.33)") {
		t.Fatalf("Shadow not serialized:\n%s", out)
	}
	// An elevation that does not divide evenly is rounded to the precision a
	// device pixel is meaningful at, not printed at full float width
	// (4/3 = 1.3333333333333333). The WASM runtime rounds identically.
	odd := &core.Node{Type: "Card", Style: &core.Style{Shadow: 4}}
	if out := ExportHTML(odd); !strings.Contains(out, "box-shadow:0 1.33px 2px rgba(0,0,0,0.33)") {
		t.Fatalf("Shadow not rounded:\n%s", out)
	}
	if out := ExportHTML(&core.Node{Type: "Card", Style: &core.Style{}}); strings.Contains(out, "box-shadow") {
		t.Fatalf("zero Shadow must emit nothing:\n%s", out)
	}
}

// px, not CSS's unitless multiplier: the field is an absolute line box height
// on both natives (Compose lineHeight = n.sp, SwiftUI a lineSpacing derived
// from n), and a bare number here would mean "n times the font size" instead.
func TestLineHeightIsAbsolute(t *testing.T) {
	n := &core.Node{Type: "Text", Style: &core.Style{LineHeight: 24}}
	if out := ExportHTML(n); !strings.Contains(out, "line-height:24px") {
		t.Fatalf("LineHeight not serialized as px:\n%s", out)
	}
}

// "hidden" and "visible" are not CSS display keywords. Emitting them there —
// which is what this did — produced a declaration the browser discarded, so
// the mode was stated in Go, written into the document, and had no effect
// anywhere. visibility keeps the node's space and drops its pixels, which is
// what the natives do with the same mode (SwiftUI .opacity(0), Compose alpha 0).
func TestDisplayHiddenBecomesVisibility(t *testing.T) {
	hidden := &core.Node{Type: "Box", Style: &core.Style{Display: core.DisplayHidden}}
	out := ExportHTML(hidden)
	if !strings.Contains(out, "visibility:hidden") {
		t.Fatalf("DisplayHidden must become visibility:hidden:\n%s", out)
	}
	if strings.Contains(out, "display:hidden") {
		t.Fatalf("display:hidden is not valid CSS and must not be emitted:\n%s", out)
	}

	visible := &core.Node{Type: "Box", Style: &core.Style{Display: core.DisplayVisible}}
	out = ExportHTML(visible)
	if !strings.Contains(out, "visibility:visible") {
		t.Fatalf("DisplayVisible must become visibility:visible:\n%s", out)
	}
	if strings.Contains(out, "display:visible") {
		t.Fatalf("display:visible is not valid CSS and must not be emitted:\n%s", out)
	}
}

// The three semantics fields both natives have always read. aria-hidden wins
// alone, matching Compose's clearAndSetSemantics and SwiftUI's
// accessibilityHidden: a name on a pruned subtree is contradictory, not
// additive.
func TestAccessibilityFieldsBecomeAria(t *testing.T) {
	labelled := &core.Node{
		Type:  "Box",
		Style: &core.Style{AccessibilityLabel: "Close", AccessibilityHint: "Dismisses the sheet"},
	}
	out := ExportHTML(labelled)
	if !strings.Contains(out, `aria-label="Close"`) {
		t.Fatalf("AccessibilityLabel not exported:\n%s", out)
	}
	if !strings.Contains(out, `aria-description="Dismisses the sheet"`) {
		t.Fatalf("AccessibilityHint not exported:\n%s", out)
	}

	hidden := &core.Node{
		Type:  "Box",
		Style: &core.Style{AccessibilityHidden: true, AccessibilityLabel: "Close"},
	}
	out = ExportHTML(hidden)
	if !strings.Contains(out, `aria-hidden="true"`) {
		t.Fatalf("AccessibilityHidden not exported:\n%s", out)
	}
	if strings.Contains(out, "aria-label") {
		t.Fatalf("aria-hidden prunes the subtree; a name alongside it is contradictory:\n%s", out)
	}
}

// The fourth semantics field. core.Role's values are ARIA's own spellings, so
// the export is the value verbatim — that is the whole reason the vocabulary
// is spelled the way it is (see core/role.go).
//
// Every declared Role is exercised rather than a sample of them: this is the
// target where a role costs nothing to support, so "supported except for the
// two nobody tried" is a failure mode worth closing by construction.
func TestEveryRoleBecomesTheRoleAttribute(t *testing.T) {
	for _, role := range core.Roles() {
		n := &core.Node{Type: "Box", Style: &core.Style{AccessibilityRole: role}}
		want := `role="` + string(role) + `"`
		if out := ExportHTML(n); !strings.Contains(out, want) {
			t.Errorf("%s not exported as %s:\n%s", role, want, out)
		}
	}

	// The zero value writes nothing at all — an unset role is the state every
	// node in every existing golden is in. Nameless, because a name is the one
	// thing that makes the exporter supply a role of its own; see
	// TestANamedContainerIsGivenTheGroupRole.
	bare := &core.Node{Type: "Box", Style: &core.Style{BorderRadius: 4}}
	if out := ExportHTML(bare); strings.Contains(out, "role=") {
		t.Errorf("RoleNone wrote a role attribute:\n%s", out)
	}
}

// ariaRole's fallback: a container that names itself and says nothing about
// what it is gets `group`, because ARIA prohibits a name on the `generic` role
// a <div> and a <span> carry, and browsers enforce the prohibition by dropping
// the name. Both natives announce such a name already, so this was a silence
// on the two web targets alone. See core.RoleGroup.
func TestANamedContainerIsGivenTheGroupRole(t *testing.T) {
	named := &core.Node{Type: "Box", Style: &core.Style{AccessibilityLabel: "Unread messages"}}
	out := ExportHTML(named)
	if !strings.Contains(out, `role="group"`) {
		t.Errorf("a named Box exported no role, so its aria-label is dropped by every "+
			"browser:\n%s", out)
	}
	if strings.Count(out, "role=") != 1 {
		t.Errorf("expected exactly one role attribute:\n%s", out)
	}

	// A <span> is `generic` too, so core.Text is rescued on the same terms.
	text := &core.Node{Type: "Text", Props: map[string]any{"text": "*"},
		Style: &core.Style{AccessibilityLabel: "required"}}
	if out := ExportHTML(text); !strings.Contains(out, `role="group"`) {
		t.Errorf("a named Text exported no role:\n%s", out)
	}
}

// The three guards. Each one is a case where supplying a role would take
// something away rather than add it.
func TestTheGroupFallbackIsWithheldWhereItWouldCost(t *testing.T) {
	// A tag whose implicit role is not `generic` can already carry a name, and
	// writing a role onto it would replace the browser's.
	for _, nodeType := range []string{"Button", "Image", "Input"} {
		n := &core.Node{Type: nodeType, Style: &core.Style{AccessibilityLabel: "Close"}}
		if out := ExportHTML(n); strings.Contains(out, `role="group"`) {
			t.Errorf("%s was given a group role, replacing the one its tag implies:\n%s",
				nodeType, out)
		}
	}

	// A Modal is a dialog by virtue of being a Modal, and a dialog is
	// nameable. The fallback must not turn one into a group.
	modal := &core.Node{Type: "Modal", Style: &core.Style{AccessibilityLabel: "Confirm"}}
	out := ExportHTML(modal)
	if !strings.Contains(out, `role="dialog"`) || strings.Contains(out, `role="group"`) {
		t.Errorf("a named Modal should stay a dialog:\n%s", out)
	}

	// An author who said what the node is keeps their word.
	authored := &core.Node{Type: "Box", Style: &core.Style{
		AccessibilityRole: core.RoleImg, AccessibilityLabel: "Compass, north"}}
	if out := ExportHTML(authored); strings.Contains(out, `role="group"`) {
		t.Errorf("an authored role was overwritten by the fallback:\n%s", out)
	}

	// aria-hidden still wins alone: there is no element in the accessibility
	// tree for either the name or the role to describe.
	hidden := &core.Node{Type: "Box", Style: &core.Style{
		AccessibilityHidden: true, AccessibilityLabel: "Close"}}
	if out := ExportHTML(hidden); strings.Contains(out, "role=") {
		t.Errorf("a hidden node was given a role:\n%s", out)
	}
}

// core.Style.AccessibilityID and core.Style.AccessibilityControls, the
// vocabulary's only two references. Both are written verbatim, in both
// directions, and nothing checks that the target exists — see
// accessibilityAttrs.
func TestTheIDRefPairExportsVerbatim(t *testing.T) {
	tab := &core.Node{Type: "Box", Style: &core.Style{
		AccessibilityRole:     core.RoleTab,
		AccessibilityID:       "home-tab",
		AccessibilityControls: "app-panel",
	}}
	out := ExportHTML(tab)
	for _, want := range []string{`id="home-tab"`, `aria-controls="app-panel"`, `role="tab"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s:\n%s", want, out)
		}
	}

	// A reference to nothing is inert rather than an error: one export is one
	// tree, and the document it lands in is not this package's to index.
	dangling := &core.Node{Type: "Box", Style: &core.Style{AccessibilityControls: "elsewhere"}}
	if out := ExportHTML(dangling); !strings.Contains(out, `aria-controls="elsewhere"`) {
		t.Errorf("a dangling aria-controls should still be written:\n%s", out)
	}
}

// aria-hidden prunes the subtree, so a role on the same node is contradictory
// for the same reason a name is — the element the role describes is not in the
// tree to describe.
func TestHiddenBeatsRole(t *testing.T) {
	n := &core.Node{
		Type:  "Box",
		Style: &core.Style{AccessibilityHidden: true, AccessibilityRole: core.RoleTable},
	}
	if out := ExportHTML(n); strings.Contains(out, `role="table"`) {
		t.Errorf("aria-hidden should win alone:\n%s", out)
	}
}

// The selected state, and the switch that turns one Go field into whichever
// of ARIA's two attributes the role calls for.
//
// Every role is exercised rather than the three that are expected to write
// something, for the reason TestEveryRoleBecomesTheRoleAttribute walks the
// whole vocabulary: what is being pinned is the *scoping*, and a scoping
// checked only at the values it admits is not checked at all. A role that
// starts writing a state it should not is exactly as wrong as one that stops
// writing a state it should — aria-selected on a listitem is invalid ARIA,
// which a reader drops, so the announcement is silently lost either way. That
// pair is the sharpest case in the vocabulary: `option` admits the attribute
// and `listitem`, the role that looks like its synonym, does not.
func TestSelectedStateBecomesTheAttributeTheRoleCallsFor(t *testing.T) {
	// ARIA's own scoping, restated here rather than read out of the exporter:
	// a test that derived the answer from ariaSelected would agree with any
	// mistake it made.
	want := map[core.Role]string{
		core.RoleOption:       "aria-selected",
		core.RoleTab:          "aria-selected",
		core.RoleRow:          "aria-selected",
		core.RoleColumnHeader: "aria-selected",
		core.RoleButton:       "aria-pressed",
	}

	for _, role := range core.Roles() {
		for _, state := range core.SelectedStates() {
			n := &core.Node{Type: "Box", Style: &core.Style{
				AccessibilityRole:     role,
				AccessibilitySelected: state,
			}}
			out := ExportHTML(n)
			attr, wanted := want[role]

			if wanted {
				pair := attr + `="` + string(state) + `"`
				if !strings.Contains(out, pair) {
					t.Errorf("role %q + %q: want %s\n%s", role, state, pair, out)
				}
				continue
			}
			for _, other := range []string{"aria-selected", "aria-pressed"} {
				if strings.Contains(out, other) {
					t.Errorf("role %q + %q wrote %s — ARIA does not define it for this "+
						"role, so a reader drops it\n%s", role, state, other, out)
				}
			}
		}
	}
}

// The one node type that needs no role, because it already is one.
//
// components.Chip renders as a core.Button and sets no AccessibilityRole — the
// node type carries it, which is the rule core/role.go states for RoleButton
// and RoleDialog. Without this arm the widget that most wants aria-pressed
// would be the single node that could not have it, and nothing would say so:
// the state would merge, cross the bridge, reach both natives, and vanish on
// the web alone.
func TestAButtonNodeCarriesAPressedStateWithoutARole(t *testing.T) {
	n := &core.Node{
		Type:  "Button",
		Props: map[string]any{"label": "Active"},
		Style: &core.Style{AccessibilitySelected: core.SelectedOn},
	}
	if out := ExportHTML(n); !strings.Contains(out, `aria-pressed="true"`) {
		t.Errorf("a core.Button with no role should still take a pressed state:\n%s", out)
	}

	// The node type does not extend to anything else. A Box is a div, and a
	// pressed div is invalid ARIA whatever it does on tap — the case
	// core.RoleButton exists for.
	box := &core.Node{Type: "Box", Style: &core.Style{AccessibilitySelected: core.SelectedOn}}
	if out := ExportHTML(box); strings.Contains(out, "aria-pressed") {
		t.Errorf("a roleless Box is not a button:\n%s", out)
	}
}

// The zero value writes nothing, which is the state every node in every
// existing golden is in.
func TestUnstatedSelectionWritesNoAttribute(t *testing.T) {
	n := &core.Node{Type: "Box", Style: &core.Style{AccessibilityRole: core.RoleTab}}
	out := ExportHTML(n)
	if strings.Contains(out, "aria-selected") || strings.Contains(out, "aria-pressed") {
		t.Errorf("SelectedUnset wrote an attribute:\n%s", out)
	}
	// The role itself is unaffected: a tab that does not say whether it is
	// showing is still a tab.
	if !strings.Contains(out, `role="tab"`) {
		t.Errorf("the role went missing with the state:\n%s", out)
	}
}

// aria-hidden prunes the subtree, so a state on the same node describes an
// element the reader cannot reach — the same exclusion the role and the level
// are already held to.
func TestHiddenBeatsTheSelectedState(t *testing.T) {
	n := &core.Node{Type: "Box", Style: &core.Style{
		AccessibilityHidden:   true,
		AccessibilityRole:     core.RoleTab,
		AccessibilitySelected: core.SelectedOn,
	}}
	if out := ExportHTML(n); strings.Contains(out, "aria-selected") {
		t.Errorf("aria-hidden should win alone:\n%s", out)
	}
}

// aria-expanded is written for the roles ARIA defines it for and no others,
// and the list is not the selection's.
//
// Restated here rather than read out of ariaExpanded, for the reason the
// selection census gives: a test that derived the answer from the exporter
// would agree with any mistake it made. The value of restating is higher here
// than usual, because the two guards look interchangeable and are wrong at
// four roles in the two directions the loop below covers — option takes a
// selection and not a disclosure, link and listbox the reverse.
func TestExpandedStateIsScopedToItsOwnRoles(t *testing.T) {
	want := map[core.Role]bool{
		core.RoleButton:       true,
		core.RoleLink:         true,
		core.RoleListBox:      true,
		core.RoleRow:          true,
		core.RoleColumnHeader: true,
		core.RoleTab:          true,
	}

	for _, role := range core.Roles() {
		for _, state := range core.ExpandedStates() {
			n := &core.Node{Type: "Box", Style: &core.Style{
				AccessibilityRole:     role,
				AccessibilityExpanded: state,
			}}
			out := ExportHTML(n)
			pair := `aria-expanded="` + string(state) + `"`

			if want[role] {
				if !strings.Contains(out, pair) {
					t.Errorf("role %q + %q: want %s\n%s", role, state, pair, out)
				}
				continue
			}
			if strings.Contains(out, "aria-expanded") {
				t.Errorf("role %q + %q wrote aria-expanded — ARIA does not define it for "+
					"this role, so a reader drops it\n%s", role, state, out)
			}
		}
	}

	// The half that a shared guard would get wrong, asserted as a pair so the
	// divergence is stated rather than implied by two absences elsewhere.
	if want[core.RoleOption] {
		t.Error("option takes aria-selected and not aria-expanded: an option is a leaf " +
			"choice, and the thing that expands is the listbox around it")
	}
	if !want[core.RoleLink] || !want[core.RoleListBox] {
		t.Error("link and listbox take aria-expanded and neither selection attribute")
	}
}

// The disclosure pattern is a button, so the node type has to carry it.
//
// core.Button sets no AccessibilityRole and is the element ARIA's own
// disclosure and accordion patterns are built out of — so without this arm the
// attribute would be defined for exactly the node that could not have it.
// Same rule that gives a Chip its aria-pressed and a Modal its dialog role.
func TestAButtonNodeCarriesAnExpandedStateWithoutARole(t *testing.T) {
	n := &core.Node{
		Type:  "Button",
		Props: map[string]any{"label": "What is a hook"},
		Style: &core.Style{AccessibilityExpanded: core.ExpandedClosed},
	}
	if out := ExportHTML(n); !strings.Contains(out, `aria-expanded="false"`) {
		t.Errorf("a core.Button with no role should still take a disclosure state:\n%s", out)
	}

	// And it does not extend past the one type. A named Box is given
	// core.RoleGroup so its name is legal, and `group` is *not* one of the
	// roles above — which is the asymmetry with the name worth pinning: the
	// exporter can supply a role that makes a name announceable and there is
	// no role it could supply that makes a disclosure one.
	box := &core.Node{Type: "Box", Style: &core.Style{
		AccessibilityLabel:    "What is a hook",
		AccessibilityExpanded: core.ExpandedClosed,
	}}
	out := ExportHTML(box)
	if !strings.Contains(out, `role="group"`) {
		t.Errorf("the supplied group went missing:\n%s", out)
	}
	if strings.Contains(out, "aria-expanded") {
		t.Errorf("a group is not a disclosure; the state must be dropped:\n%s", out)
	}
}

// The zero value writes nothing, which is the state every node in every
// existing golden is in — and the role survives it, because a button that does
// not say whether it is open is still a button.
func TestUnstatedDisclosureWritesNoAttribute(t *testing.T) {
	n := &core.Node{Type: "Box", Style: &core.Style{AccessibilityRole: core.RoleButton}}
	out := ExportHTML(n)
	if strings.Contains(out, "aria-expanded") {
		t.Errorf("ExpandedUnset wrote an attribute:\n%s", out)
	}
	if !strings.Contains(out, `role="button"`) {
		t.Errorf("the role went missing with the state:\n%s", out)
	}
}

// aria-hidden prunes the subtree, so a disclosure state on the same node
// describes an element no reader can reach — the exclusion the role, the level
// and the selection are all already held to.
func TestHiddenBeatsTheExpandedState(t *testing.T) {
	n := &core.Node{Type: "Box", Style: &core.Style{
		AccessibilityHidden:   true,
		AccessibilityRole:     core.RoleButton,
		AccessibilityExpanded: core.ExpandedOpen,
	}}
	if out := ExportHTML(n); strings.Contains(out, "aria-expanded") {
		t.Errorf("aria-hidden should win alone:\n%s", out)
	}
}

// The three state attributes coexist on one element without colliding.
//
// A menu button that is both the current tab and showing its submenu is the
// shape core.ExpandedState's doc names as the reason the two states are two
// fields rather than one. Nothing in the framework builds it, which is exactly
// why it is asserted: the two guards are separate switches over the same role,
// and a "simplification" that merged them would still pass every test above.
func TestASelectionAndADisclosureCoexistOnOneNode(t *testing.T) {
	n := &core.Node{Type: "Box", Style: &core.Style{
		AccessibilityRole:     core.RoleTab,
		AccessibilitySelected: core.SelectedOn,
		AccessibilityExpanded: core.ExpandedOpen,
	}}
	out := ExportHTML(n)
	for _, want := range []string{`aria-selected="true"`, `aria-expanded="true"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s — the two states are independent facts:\n%s", want, out)
		}
	}
}

// A label is user-originated (it reaches the DSL from app state like any other
// string), so it goes out through the same escaping every other attribute value
// does. A raw double quote is the attribute-breakout character.
func TestAccessibilityLabelCannotInjectAttributes(t *testing.T) {
	n := &core.Node{
		Type:  "Box",
		Style: &core.Style{AccessibilityLabel: `" onclick="steal()`},
	}
	if out := ExportHTML(n); strings.Contains(out, `onclick="steal()"`) {
		t.Fatalf("accessibility label broke out of its attribute:\n%s", out)
	}
}

// The fields that had a StyleProp constructor in Go and no reader on any of
// the four targets. They are one CSS property each here; the native gap is
// documented rather than faked.
func TestPreviouslyUnreadStyleFieldsAreEmitted(t *testing.T) {
	n := &core.Node{
		Type: "Box",
		Style: &core.Style{
			MinWidth: "10px", MinHeight: "11px",
			MaxWidth: "12px", MaxHeight: "13px",
			Overflow: "hidden", WhiteSpace: "nowrap",
			Position: core.PositionAbsolute,
			Top:      "1px", Right: "2px", Bottom: "3px", Left: "4px",
			ZIndex:   5,
			FlexWrap: "wrap", RowGap: 6, ColumnGap: 7,
			AlignSelf: core.AlignItemsEnd, FlexBasis: "50%", FlexShrink: 2,
			Animation: "bounce 2s infinite",
		},
	}
	out := ExportHTML(n)
	for _, want := range []string{
		"min-width:10px", "min-height:11px", "max-width:12px", "max-height:13px",
		"overflow:hidden", "white-space:nowrap",
		"position:absolute", "top:1px", "right:2px", "bottom:3px", "left:4px",
		"z-index:5",
		"flex-wrap:wrap", "row-gap:6px", "column-gap:7px",
		"align-self:flex-end", "flex-basis:50%", "flex-shrink:2",
		"animation:bounce 2s infinite",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

// FlexWrap does not turn a block-flow box into a flex container on its own —
// it means nothing until the box already is one — so promoting a box for it
// alone would change that box's layout to no purpose.
//
// A Text and not a Box: a Box is a stack container and so is flex before this
// Style is read at all (stackAxes). The runtime's twin of this test uses a
// Text for the same reason.
func TestFlexWrapAloneDoesNotCreateAFlexContainer(t *testing.T) {
	n := &core.Node{Type: "Text", Props: map[string]any{"content": "hi"},
		Style: &core.Style{FlexWrap: "wrap"}}
	if out := ExportHTML(n); strings.Contains(out, "display:flex") {
		t.Fatalf("flex-wrap alone must not promote a box:\n%s", out)
	}
}

// The two axis gaps do promote, because `gap` IS `row-gap` plus `column-gap`:
// a node that sets one of them has asked for the same spacing Gap asks for,
// by another name, and needs the same flex container to get it.
//
// This used to be grouped with FlexWrap above and pinned the other way. What
// changed is not the CSS but the other targets: both natives now read the
// longhands as their stacks' spacing (GrMobStyle's verticalGap/horizontalGap),
// so a Column carrying RowGap alone spaced its children on a phone while this
// exporter emitted an inert `row-gap` into a block-flow div.
func TestAxisGapsCreateAFlexContainer(t *testing.T) {
	for _, tc := range []struct {
		name string
		s    core.Style
		want string
	}{
		{"row-gap", core.Style{RowGap: 4}, "row-gap:4px"},
		{"column-gap", core.Style{ColumnGap: 4}, "column-gap:4px"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			style := tc.s
			out := ExportHTML(&core.Node{Type: "Column", Style: &style})
			if !strings.Contains(out, "display:flex") {
				t.Errorf("%s alone must promote the box, or the declaration is inert:\n%s", tc.name, out)
			}
			if !strings.Contains(out, tc.want) {
				t.Errorf("missing %q:\n%s", tc.want, out)
			}
		})
	}
}

// An axis gap set alongside Gap wins, which the cascade delivers as long as
// the longhands are written after the shorthand. Order is the whole content
// of this check — both declarations are present either way.
func TestAxisGapIsWrittenAfterTheGapShorthand(t *testing.T) {
	n := &core.Node{Type: "Column", Style: &core.Style{Gap: 10, RowGap: 4}}
	out := ExportHTML(n)
	gap, row := strings.Index(out, "gap:10px"), strings.Index(out, "row-gap:4px")
	if gap < 0 || row < 0 {
		t.Fatalf("expected both gap:10px and row-gap:4px:\n%s", out)
	}
	if row < gap {
		t.Errorf("row-gap is declared before gap, so the shorthand overrides it:\n%s", out)
	}
}

// --- Modal ------------------------------------------------------------------

// Every other target honors Visible — Renderer.swift and Renderer.kt bail out
// before laying the node out, the WASM runtime toggles display — and this one
// rendered the content inline regardless, so the export showed a screen with a
// closed dialog's body spliced into the middle of it.
func TestModalHonorsVisible(t *testing.T) {
	closed := &core.Node{
		Type:     "Modal",
		Props:    map[string]any{"visible": false, "backdrop": "#00000088"},
		Children: []*core.Node{textNode("dialog body")},
	}
	out := ExportHTML(closed)
	if !strings.Contains(out, "display:none") {
		t.Fatalf("a closed modal must not be laid out:\n%s", out)
	}
	// The content stays in the document: a Modal hides, it does not unmount
	// (see core.ModalContent), which is what lets its state survive a close.
	if !strings.Contains(out, "dialog body") {
		t.Fatalf("modal content must stay mounted:\n%s", out)
	}

	open := &core.Node{
		Type:     "Modal",
		Props:    map[string]any{"visible": true, "backdrop": "#00000088"},
		Children: []*core.Node{textNode("dialog body")},
	}
	out = ExportHTML(open)
	if !strings.Contains(out, "display:flex") {
		t.Fatalf("an open modal must be laid out:\n%s", out)
	}
	if !strings.Contains(out, "background:#00000088") {
		t.Fatalf("the backdrop is the scrim; it must be emitted:\n%s", out)
	}
	// The chassis: a fixed inset-0 overlay above the app and below the toast
	// layer, the same rules the WASM runtime assigns in createElement.
	for _, want := range []string{"position:fixed", "z-index:1000", "justify-content:center"} {
		if !strings.Contains(out, want) {
			t.Errorf("modal chassis missing %q:\n%s", want, out)
		}
	}
}

// --- Heading level ----------------------------------------------------------

// aria-level, and the two guards that decide whether it is written at all.
//
// The role guard is ARIA's own scoping and the range guard drops rather than
// clamps — see core.Style.AccessibilityHeadingLevel for why an out-of-range
// value is not quietly rewritten into the nearest legal one.
func TestHeadingLevelBecomesAriaLevel(t *testing.T) {
	for level := 1; level <= 6; level++ {
		n := &core.Node{Type: "Box", Style: &core.Style{
			AccessibilityRole:         core.RoleHeading,
			AccessibilityHeadingLevel: level,
		}}
		want := `aria-level="` + strconv.Itoa(level) + `"`
		if out := ExportHTML(n); !strings.Contains(out, want) {
			t.Errorf("level %d not exported as %s:\n%s", level, want, out)
		}
	}

	for _, tc := range []struct {
		name  string
		style core.Style
	}{
		{"unset", core.Style{AccessibilityRole: core.RoleHeading}},
		{"below range", core.Style{AccessibilityRole: core.RoleHeading, AccessibilityHeadingLevel: -1}},
		{"above range", core.Style{AccessibilityRole: core.RoleHeading, AccessibilityHeadingLevel: 7}},
		// aria-level is not defined for columnheader, which is why DataTable's
		// column headers take the role and no tier.
		{"columnheader", core.Style{AccessibilityRole: core.RoleColumnHeader, AccessibilityHeadingLevel: 2}},
		{"no role at all", core.Style{AccessibilityHeadingLevel: 2}},
	} {
		s := tc.style
		if out := ExportHTML(&core.Node{Type: "Box", Style: &s}); strings.Contains(out, "aria-level") {
			t.Errorf("%s: wrote an aria-level it should have dropped:\n%s", tc.name, out)
		}
	}
}

// aria-hidden prunes the subtree, so a level on the same node describes the
// depth of a heading no reader can reach — the same reason a name and a role
// are dropped there.
func TestHiddenBeatsHeadingLevel(t *testing.T) {
	n := &core.Node{Type: "Box", Style: &core.Style{
		AccessibilityHidden:       true,
		AccessibilityRole:         core.RoleHeading,
		AccessibilityHeadingLevel: 2,
	}}
	if out := ExportHTML(n); strings.Contains(out, "aria-level") {
		t.Errorf("aria-hidden should win alone:\n%s", out)
	}
}

// --- Modal semantics --------------------------------------------------------

// A Modal is a dialog by virtue of being a Modal. core.ModalNode carries no
// Style, so there is nothing for a core.Role to ride on — which is exactly why
// the two DOM renderers were the only targets where an overlay announced as an
// unnamed group while both natives got it from their platform dialog.
func TestModalIsADialog(t *testing.T) {
	open := &core.Node{
		Type:     "Modal",
		Props:    map[string]any{"visible": true, "backdrop": "#00000088"},
		Children: []*core.Node{textNode("dialog body")},
	}
	out := ExportHTML(open)
	if !strings.Contains(out, `role="dialog"`) {
		t.Errorf("a Modal must announce as a dialog:\n%s", out)
	}
	if !strings.Contains(out, `aria-modal="true"`) {
		t.Errorf("a Modal must say the rest of the screen is inert behind it:\n%s", out)
	}

	// A closed one keeps both, and needs no special case: modalChassis gives
	// it display:none, which takes the element and its subtree out of the
	// accessibility tree, so there is no reader for the claim to mislead.
	closed := &core.Node{
		Type:  "Modal",
		Props: map[string]any{"visible": false},
	}
	if out := ExportHTML(closed); !strings.Contains(out, "display:none") {
		t.Errorf("a closed modal must not be laid out:\n%s", out)
	}
}

// The author's word beats the framework's, the same precedent modalChassis
// sets for style: its declarations go first so the node's own style outranks
// them. A hand-built Modal that states its own role means it.
func TestAnAuthoredRoleBeatsTheModalDefault(t *testing.T) {
	n := &core.Node{
		Type:  "Modal",
		Props: map[string]any{"visible": true},
		Style: &core.Style{AccessibilityRole: core.RoleAlert},
	}
	out := ExportHTML(n)
	if strings.Contains(out, `role="dialog"`) {
		t.Errorf("the widget's default overrode the author's role:\n%s", out)
	}
	if !strings.Contains(out, `role="alert"`) {
		t.Errorf("the authored role was dropped:\n%s", out)
	}
	// aria-modal is not expressible through core.Role at all, so it is not the
	// author's to replace — it survives whatever role they chose.
	if !strings.Contains(out, `aria-modal="true"`) {
		t.Errorf("aria-modal should not depend on the role:\n%s", out)
	}
}

// Hidden wins over the node type too. An overlay pruned from the accessibility
// tree has no element for role="dialog" to describe, and aria-modal on it would
// claim the document behind it is inert for something a reader cannot reach.
func TestHiddenBeatsTheModalSemantics(t *testing.T) {
	n := &core.Node{
		Type:  "Modal",
		Props: map[string]any{"visible": true},
		Style: &core.Style{AccessibilityHidden: true},
	}
	out := ExportHTML(n)
	if strings.Contains(out, "role=") || strings.Contains(out, "aria-modal") {
		t.Errorf("aria-hidden should win alone:\n%s", out)
	}
}

// CarriesOwnRole is the TabView wiring's view of the same fact the exporter
// acts on, and the two live in different files. Pinned by construction rather
// than by listing "Modal" twice: for every node type the tag table knows, a
// bare node of that type either exports a role attribute or does not, and
// CarriesOwnRole has to agree.
//
// The failure this closes is a duplicate role attribute — a self-roling page
// inside a TabView getting role="tabpanel" written on top of the role its type
// already gave it.
func TestCarriesOwnRoleAgreesWithTheExport(t *testing.T) {
	for nodeType := range tags {
		n := &core.Node{Type: nodeType, Props: map[string]any{"visible": true}}
		exported := strings.Contains(ExportHTML(n), "role=")
		if got := CarriesOwnRole(nodeType); got != exported {
			t.Errorf("%s: CarriesOwnRole = %v but a bare node exports a role = %v — the "+
				"TabView wiring and the exporter disagree about who owns the attribute",
				nodeType, got, exported)
		}
	}
}

// --- The user-agent border reset -------------------------------------------
//
// Three of the four renderers draw a border only when the style asks for one.
// The fourth pair — the DOM targets — used the same guard and got the negative
// case wrong: emitting nothing leaves the user-agent stylesheet in charge, and
// for a <button> that stylesheet draws a 2px outset rule. See borderResetTypes
// in tag.go, and components.Button's EmphasisGhost, which is documented as
// "EmphasisOutlined without the rule" and had one on the web.

func TestButtonWithNoBorderIsTalkedOutOfTheBrowsers(t *testing.T) {
	// EmphasisGhost's shape: a transparent fill and a colored label, with
	// nothing said about a border.
	n := &core.Node{
		Type:  "Button",
		Props: map[string]any{"label": "Skip"},
		Style: &core.Style{Background: "#00000000", TextColor: "#007AFF"},
	}
	if out := ExportHTML(n); !strings.Contains(out, "border:none") {
		t.Errorf("a borderless <button> did not reset the user agent's border:\n%s", out)
	}
}

func TestBorderResetDoesNotSwallowARealBorder(t *testing.T) {
	// EmphasisOutlined's shape. The reset is the *else* of the emission, so a
	// button that states a border must still get exactly that one and no
	// second declaration to argue with it.
	n := &core.Node{
		Type:  "Button",
		Props: map[string]any{"label": "Cancel"},
		Style: &core.Style{BorderWidth: 1, BorderColor: "#007AFF"},
	}
	out := ExportHTML(n)
	if !strings.Contains(out, "border:1px solid #007AFF") {
		t.Errorf("the authored border is missing:\n%s", out)
	}
	if strings.Contains(out, "border:none") {
		t.Errorf("the reset was written alongside a real border:\n%s", out)
	}
}

func TestHalfABorderStillResets(t *testing.T) {
	// Both halves are required on every target — Compose skips its
	// Modifier.border unless width and color are both set, and SwiftUI's
	// grMobBorder guards the same way — so a half-stated border is no border,
	// and no border on a <button> is "none".
	for _, s := range []*core.Style{
		{BorderWidth: 1},
		{BorderColor: "#007AFF"},
	} {
		n := &core.Node{Type: "Button", Props: map[string]any{"label": "x"}, Style: s}
		out := ExportHTML(n)
		if !strings.Contains(out, "border:none") {
			t.Errorf("half a border (%+v) did not reach the reset:\n%s", s, out)
		}
		if strings.Contains(out, "solid") {
			t.Errorf("half a border (%+v) emitted a declaration:\n%s", s, out)
		}
	}
}

func TestOnlyTheNodeTypesTheBrowserDrawsOnAreReset(t *testing.T) {
	// The reset is scoped by node type, not applied to everything with no
	// border. A <div> has no user-agent border, so "none" there would be a
	// declaration that says nothing and that an author's own stylesheet would
	// have to fight. This walks the whole tag table so a new node type joins
	// the set deliberately rather than by being forgotten.
	for nodeType, tag := range tags {
		n := &core.Node{Type: nodeType, Props: map[string]any{"visible": true}}
		reset := strings.Contains(ExportHTML(n), "border:none")
		if want := ResetsUABorder(nodeType); reset != want {
			t.Errorf("%s (<%s>): exported a border reset = %v, ResetsUABorder = %v",
				nodeType, tag, reset, want)
		}
	}
}

// The membership itself, spelled out.
//
// The walk above compares the exporter against ResetsUABorder and so agrees
// with the set whatever the set says — drop a node type from borderResetTypes
// and both sides move together in silence. This is the other assertion: which
// node types the browser actually draws a frame on, written as a list, so that
// removing one is a visible edit to a claim rather than an invisible edit to a
// lookup.
func TestTheBorderResetSetIsExactly(t *testing.T) {
	want := []string{"Button", "Input", "InputPassword", "NumericInput", "Select", "TextArea"}
	got := BorderResetTypes()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("borderResetTypes = %v, want %v — <button>, <input>, <textarea> and <select> "+
			"are the four tags whose user-agent stylesheet draws a frame the Go style is meant "+
			"to own, and Checkbox and Slider share <input> without sharing that", got, want)
	}
}

// The three node types that share <input> with a text field and are pointedly
// not reset.
//
// This is the case that made the set node-type-keyed. A Checkbox and a Slider
// are <input> elements whose user agent draws the *whole control*, so the
// border the browser puts on a checkbox is the box itself rather than chrome
// the Go style is meant to own — and a tag-keyed set would have swept both in
// the moment text fields joined.
func TestTheUserAgentKeepsItsCheckboxAndSlider(t *testing.T) {
	for _, nodeType := range []string{"Checkbox", "Slider"} {
		n := &core.Node{Type: nodeType}
		if out := ExportHTML(n); strings.Contains(out, "border:none") {
			t.Errorf("%s was reset — the browser draws that control itself:\n%s", nodeType, out)
		}
	}
}

// A text field's frame now comes from the theme on all four targets, which is
// what let the two field tags into the set at all: resetting a border nothing
// replaces would have left every web field an unmarked rectangle.
//
// The style here is what core.Input hands the exporter under either bundled
// theme — the reset is the *else* arm, so a stated border must reach the
// document and the reset must not be written beside it.
func TestAThemedFieldKeepsItsOwnFrame(t *testing.T) {
	for _, nodeType := range []string{"Input", "TextArea"} {
		n := &core.Node{
			Type:  nodeType,
			Style: &core.Style{Background: "#FFFFFF", BorderColor: "#8E8E93", BorderWidth: 1},
		}
		out := ExportHTML(n)
		if !strings.Contains(out, "border:1px solid #8E8E93") {
			t.Errorf("%s: the theme's frame is missing:\n%s", nodeType, out)
		}
		if strings.Contains(out, "border:none") {
			t.Errorf("%s: the reset was written alongside a real border:\n%s", nodeType, out)
		}
	}
}

// --- The nesting level, aria-level's other two roles -------------------------
//
// ARIA defines aria-level for heading, listitem and row. core carries the
// heading tier and the collection depth as two fields, because their ranges
// differ (see core.Style.AccessibilityNestingLevel), and ariaLevel is where
// they meet at the single attribute both become.

func TestNestingLevelBecomesAriaLevel(t *testing.T) {
	// No ceiling: ARIA asks only for an integer of 1 or more, so a deep tree
	// is exported as deep as it is. 7 is the value the heading arm drops, and
	// it is here to prove the two arms really do have different ranges rather
	// than one rule applied twice.
	for _, role := range []core.Role{core.RoleListItem, core.RoleRow} {
		for _, level := range []int{1, 2, 7, 42} {
			n := &core.Node{Type: "Box", Style: &core.Style{
				AccessibilityRole:         role,
				AccessibilityNestingLevel: level,
			}}
			want := `aria-level="` + strconv.Itoa(level) + `"`
			if out := ExportHTML(n); !strings.Contains(out, want) {
				t.Errorf("%s depth %d not exported as %s:\n%s", role, level, want, out)
			}
		}
	}

	for _, tc := range []struct {
		name  string
		style core.Style
	}{
		{"unset", core.Style{AccessibilityRole: core.RoleListItem}},
		{"below range", core.Style{AccessibilityRole: core.RoleListItem, AccessibilityNestingLevel: -1}},
		// The roles aria-level is *not* defined for, including the two nearest
		// misses: a list is not a listitem, and a cell is not a row.
		{"list", core.Style{AccessibilityRole: core.RoleList, AccessibilityNestingLevel: 2}},
		{"cell", core.Style{AccessibilityRole: core.RoleCell, AccessibilityNestingLevel: 2}},
		{"columnheader", core.Style{AccessibilityRole: core.RoleColumnHeader, AccessibilityNestingLevel: 2}},
		{"no role at all", core.Style{AccessibilityNestingLevel: 2}},
	} {
		s := tc.style
		if out := ExportHTML(&core.Node{Type: "Box", Style: &s}); strings.Contains(out, "aria-level") {
			t.Errorf("%s: wrote an aria-level it should have dropped:\n%s", tc.name, out)
		}
	}
}

// The two fields cannot contend for the one attribute, because the role picks
// the arm and a node has one role. Set both and state a heading: the tier is
// written and the depth is simply not read — no second attribute, and no
// argument about which wins.
func TestTheRoleDecidesWhichLevelIsRead(t *testing.T) {
	for _, tc := range []struct {
		role core.Role
		want string
	}{
		{core.RoleHeading, `aria-level="2"`},
		{core.RoleListItem, `aria-level="5"`},
		{core.RoleRow, `aria-level="5"`},
	} {
		n := &core.Node{Type: "Box", Style: &core.Style{
			AccessibilityRole:         tc.role,
			AccessibilityHeadingLevel: 2,
			AccessibilityNestingLevel: 5,
		}}
		out := ExportHTML(n)
		if !strings.Contains(out, tc.want) {
			t.Errorf("%s: want %s:\n%s", tc.role, tc.want, out)
		}
		if n := strings.Count(out, "aria-level"); n != 1 {
			t.Errorf("%s: wrote aria-level %d times; one attribute has one slot:\n%s",
				tc.role, n, out)
		}
	}
}

// aria-hidden prunes the subtree, so a depth on the same node describes the
// position of an item no reader can reach — the same reason it beats the
// heading tier, the name and the role.
func TestHiddenBeatsNestingLevel(t *testing.T) {
	n := &core.Node{Type: "Box", Style: &core.Style{
		AccessibilityHidden:       true,
		AccessibilityRole:         core.RoleListItem,
		AccessibilityNestingLevel: 2,
	}}
	if out := ExportHTML(n); strings.Contains(out, "aria-level") {
		t.Errorf("aria-hidden should win alone:\n%s", out)
	}
}

// --- Rotate -----------------------------------------------------------------

func TestRotateExportsATransform(t *testing.T) {
	out := ExportHTML(&core.Node{
		Type:  "Box",
		Props: map[string]any{},
		Style: &core.Style{Rotate: 123.4},
	})
	if !strings.Contains(out, "transform:rotate(123.4deg)") {
		t.Fatalf("expected a rotate transform, got:\n%s", out)
	}
}

// Zero means "not turned", and an identity transform in the declaration would
// still create a containing block for any absolutely-positioned descendant —
// a real behavioral difference, not just noise.
func TestZeroRotateEmitsNoTransform(t *testing.T) {
	out := ExportHTML(&core.Node{
		Type:  "Box",
		Props: map[string]any{},
		Style: &core.Style{Rotate: 0},
	})
	if strings.Contains(out, "transform") {
		t.Fatalf("a zero angle emitted a transform:\n%s", out)
	}
}

// The winding survives the export, matching what the field promises: an
// unwrapped angle is not folded onto the circle on its way to CSS.
func TestNegativeAndUnwrappedRotateSurviveTheExport(t *testing.T) {
	for _, c := range []struct {
		deg  float64
		want string
	}{
		{-90, "transform:rotate(-90deg)"},
		{370, "transform:rotate(370deg)"},
	} {
		out := ExportHTML(&core.Node{
			Type:  "Box",
			Props: map[string]any{},
			Style: &core.Style{Rotate: c.deg},
		})
		if !strings.Contains(out, c.want) {
			t.Errorf("Rotate(%g): want %q in:\n%s", c.deg, c.want, out)
		}
	}
}
