package htmlout

import (
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

func TestSpacerRendersHeight(t *testing.T) {
	n := &core.Node{Type: "Spacer", Props: map[string]any{"size": 20}}
	out := ExportHTML(n)
	if !strings.Contains(out, "height:20px") {
		t.Fatalf("spacer missing height style:\n%s", out)
	}
}

func TestCheckboxChecked(t *testing.T) {
	n := &core.Node{Type: "Checkbox", Props: map[string]any{"checked": true}}
	out := ExportHTML(n)
	if !strings.Contains(out, `type="checkbox"`) || !strings.Contains(out, "checked") {
		t.Fatalf("checkbox not rendered checked:\n%s", out)
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
	for _, nodeType := range []string{"Button", "Input", "InputPassword", "NumericInput", "TextArea", "Checkbox"} {
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
func TestDisplayStaysVerbatimOnNonFlexNodes(t *testing.T) {
	n := &core.Node{Type: "Column", Style: &core.Style{Display: core.DisplayBlock}}
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
