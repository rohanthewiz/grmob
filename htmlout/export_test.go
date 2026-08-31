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
