// Package htmlout exports a rendered core.Node tree as a standalone HTML
// document. It is the demo/inspection path (the example apps print its output);
// the WASM runtime does not consume it, so readability is favored over
// compactness.
package htmlout

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rohanthewiz/element"
	"github.com/rohanthewiz/grmob/core"
)

// ExportHTML renders the node tree into a complete HTML document.
//
// Output is built on the element library rather than hand-assembled strings so
// that escaping is handled once, in one place: element quote-escapes every
// attribute value (a raw double quote is the attribute-breakout character), and
// text content goes through TE(), which entity-escapes it. User-originated
// strings — Text content, input values, labels, image srcs — therefore cannot
// re-enter the document as live markup.
func ExportHTML(node *core.Node) string {
	b := element.NewBuilder()
	// b.Html writes the <!DOCTYPE html> declaration itself.
	b.Html("lang", "en").R(
		b.Body().R(
			renderNode(b, node),
		),
	)
	// Pretty re-indents the compact single-pass output for human readers.
	// Escaped content is inert entities by this point, so re-parsing is safe.
	return b.Pretty()
}

// renderNode writes one node (and, for containers, its subtree) into the
// builder. The return value exists only so calls can sit inline as R()
// arguments, which is how element establishes evaluation order; it is ignored.
func renderNode(b *element.Builder, node *core.Node) (x any) {
	if node == nil {
		return
	}

	// Spacer is pure layout: a fixed-height gap, no children, no other props.
	if node.Type == "Spacer" {
		if size, ok := node.Props["size"].(int); ok {
			b.Div("style", fmt.Sprintf("height:%dpx", size)).R()
			return
		}
	}

	// Shared attributes: inline style first, then the callback-ID data
	// attributes the WASM runtime dispatches on (their names are a contract
	// with runtime/main.go's event bridge).
	attrs := make([]string, 0, 8)
	if sv := styleValue(node.Style, node.Type); sv != "" {
		attrs = append(attrs, "style", sv)
	}
	// A slice, not a map: map iteration order would make attribute order (and
	// therefore the exported document) nondeterministic across runs.
	for _, cb := range [...]struct{ prop, attr string }{
		{"onClick", "data-onclick"},
		{"onChange", "data-onchange"},
		{"onToggle", "data-ontoggle"},
	} {
		if id, ok := node.Props[cb.prop].(string); ok {
			attrs = append(attrs, cb.attr, id)
		}
	}

	switch node.Type {
	case "Input":
		b.Input(withLead(attrs, "type", "text",
			"value", getStr(node.Props["value"]),
			"placeholder", getStr(node.Props["placeholder"]))...).R()
	case "InputPassword":
		b.Input(withLead(attrs, "type", "password",
			"value", getStr(node.Props["value"]),
			"placeholder", getStr(node.Props["placeholder"]))...).R()
	case "NumericInput":
		b.Input(withLead(attrs, "type", "number",
			"value", getStr(node.Props["value"]))...).R()
	case "TextArea":
		rows := 3
		if r, ok := node.Props["rows"].(int); ok {
			rows = r
		}
		// TE keeps a value containing "</textarea>" from closing the element.
		b.TextArea(withLead(attrs, "rows", strconv.Itoa(rows))...).TE(getStr(node.Props["value"]))
	case "Checkbox":
		lead := []string{"type", "checkbox"}
		if v, ok := node.Props["checked"].(bool); ok && v {
			// element emits key="value" pairs only; checked="checked" is the
			// spec-blessed spelling of the bare boolean attribute.
			lead = append(lead, "checked", "checked")
		}
		b.Input(withLead(attrs, lead...)...).R()
	case "Image":
		if src, ok := node.Props["src"].(string); ok {
			b.Img(withLead(attrs, "src", src)...).R()
			return
		}
		// No src: fall through to the default container rendering, matching
		// how unknown/underspecified nodes degrade to a plain div.
		renderContainer(b, node, attrs)
	case "Text":
		b.Span(attrs...).TE(getStr(node.Props["content"]))
	case "Button":
		b.Button(attrs...).TE(getStr(node.Props["label"]))
	case "CameraView":
		b.Div(attrs...).T("[Camera View]") // placeholder text authored here, not user data
	default:
		renderContainer(b, node, attrs)
	}
	return
}

// renderContainer renders a generic container tag with the node's children.
// element writes the opening tag when Ele() is called and the closing tag when
// R() runs, so the children rendered in between land inside the element.
func renderContainer(b *element.Builder, node *core.Node, attrs []string) {
	e := b.Ele(tagForType(node.Type), attrs...)
	for _, child := range node.Children {
		renderNode(b, child)
	}
	e.R()
}

// withLead prepends type-specific attribute pairs (type, value, src, ...) ahead
// of the shared style/data attributes, preserving the attribute order the
// previous string-based exporter emitted.
func withLead(attrs []string, lead ...string) []string {
	return append(lead, attrs...)
}

func getStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func tagForType(t string) string {
	switch t {
	case "Text":
		return "span"
	case "Column", "Row", "Card", "Scroll", "Fragment", "Theme", "SafeArea":
		return "div"
	case "Button":
		return "button"
	default:
		return "div"
	}
}

// styleValue serializes the subset of Style the HTML exporter understands into
// a CSS declaration list ("" when nothing is set). The caller places it in a
// style attribute; element handles the attribute-value escaping.
//
// nodeType is needed for Gap: CSS gap only has meaning on a flex/grid
// container, and the main axis it spaces along is the node's own stacking
// direction (Row lays out horizontally, every other container vertically) —
// information that lives in the node type, not in Style.
func styleValue(s *core.Style, nodeType string) string {
	if s == nil {
		return ""
	}
	styles := []string{}
	if s.TextColor != "" {
		styles = append(styles, fmt.Sprintf("color:%s", s.TextColor))
	}
	if s.Background != "" {
		styles = append(styles, fmt.Sprintf("background:%s", s.Background))
	}
	if s.FontSize != 0 {
		styles = append(styles, fmt.Sprintf("font-size:%gpx", s.FontSize))
	}
	if s.Align != "" {
		switch s.Align {
		case core.AlignCenter:
			styles = append(styles, "text-align:center")
		case core.AlignStart:
			styles = append(styles, "text-align:left")
		case core.AlignEnd:
			styles = append(styles, "text-align:right")
		}
	}
	// Flex container properties, emitted before Display so an explicit Display
	// (set by the author) lands after and wins the browser's
	// last-declaration-wins parse.
	//
	// The native renderers implement these directly — a Compose Row/Column or
	// a SwiftUI HStack/VStack is inherently a stack, so Gap becomes
	// Arrangement.spacedBy / stack spacing and JustifyContent becomes the
	// arrangement. HTML has no such default: a plain <div> is block flow and
	// ignores gap, justify-content and align-items entirely, so the container
	// must be made flex for any of them to do anything.
	//
	// Only nodes that actually set one of these become flex containers;
	// everything else keeps the block-flow output this exporter has always
	// produced. The main axis defaults to the node's own stacking direction
	// (Row horizontal, every other container vertical) and an explicit
	// FlexDirection overrides it.
	if s.Gap != 0 || s.JustifyContent != "" || s.AlignItems != "" || s.FlexDirection != "" {
		dir := "column"
		if nodeType == "Row" {
			dir = "row"
		}
		if s.FlexDirection != "" {
			dir = string(s.FlexDirection)
		}
		styles = append(styles, "display:flex", fmt.Sprintf("flex-direction:%s", dir))
		if s.Gap != 0 {
			styles = append(styles, fmt.Sprintf("gap:%gpx", s.Gap))
		}
		if s.JustifyContent != "" {
			styles = append(styles, fmt.Sprintf("justify-content:%s", s.JustifyContent))
		}
		if s.AlignItems != "" {
			styles = append(styles, fmt.Sprintf("align-items:%s", s.AlignItems))
		}
	}
	if s.Display != "" {
		styles = append(styles, fmt.Sprintf("display:%s", s.Display))
	}
	if s.Padding != (core.EdgeInsets{}) {
		styles = append(styles, fmt.Sprintf("padding:%dpx %dpx %dpx %dpx", s.Padding.Top, s.Padding.Right, s.Padding.Bottom, s.Padding.Left))
	}
	if s.Margin != (core.EdgeInsets{}) {
		styles = append(styles, fmt.Sprintf("margin:%dpx %dpx %dpx %dpx", s.Margin.Top, s.Margin.Right, s.Margin.Bottom, s.Margin.Left))
	}
	// Flex *item* properties, as opposed to the container properties above:
	// they describe how this node behaves inside its parent's flex layout, so
	// they need no display:flex of their own.
	if s.FlexGrow != 0 {
		styles = append(styles, fmt.Sprintf("flex-grow:%g", s.FlexGrow))
	}
	if s.BorderRadius != 0 {
		styles = append(styles, fmt.Sprintf("border-radius:%gpx", s.BorderRadius))
	}
	if s.Transition != "" {
		// core.Transition's canonical "<ms>ms <easing>" is valid CSS as-is;
		// "all" scopes it to every animatable property, matching the native
		// renderers' behavior.
		styles = append(styles, fmt.Sprintf("transition:all %s", s.Transition))
	}
	return strings.Join(styles, "; ")
}
