package htmlout

import (
	"fmt"
	"github.com/rohanthewiz/grmob/core"
	"strings"
)

func ExportHTML(node *core.Node) string {
	var builder strings.Builder
	builder.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<body>\n")
	renderNode(&builder, node, 1)
	builder.WriteString("</body>\n</html>")
	return builder.String()
}

func renderNode(b *strings.Builder, node *core.Node, indent int) {
	pad := strings.Repeat("  ", indent)

	tag := tagForType(node.Type)
	if tag == "" {
		return
	}

	// Special case for Spacer
	if node.Type == "Spacer" {
		if size, ok := node.Props["size"].(int); ok {
			b.WriteString(fmt.Sprintf("%s<div style=\"height:%dpx\"></div>\n", pad, size))
			return
		}
	}

	attrs := styleAttr(node.Style)

	// Add dynamic attributes
	if id, ok := node.Props["onClick"].(string); ok {
		attrs += fmt.Sprintf(" data-onclick=\"%s\"", id)
	}
	if id, ok := node.Props["onChange"].(string); ok {
		attrs += fmt.Sprintf(" data-onchange=\"%s\"", id)
	}
	if id, ok := node.Props["onToggle"].(string); ok {
		attrs += fmt.Sprintf(" data-ontoggle=\"%s\"", id)
	}

	// Open tag
	switch node.Type {
	case "Input":
		val := getStr(node.Props["value"])
		ph := getStr(node.Props["placeholder"])
		b.WriteString(fmt.Sprintf("%s<input type=\"text\" value=\"%s\" placeholder=\"%s\"%s />\n", pad, val, ph, attrs))
		return
	case "InputPassword":
		val := getStr(node.Props["value"])
		ph := getStr(node.Props["placeholder"])
		b.WriteString(fmt.Sprintf("%s<input type=\"password\" value=\"%s\" placeholder=\"%s\"%s />\n", pad, val, ph, attrs))
		return
	case "NumericInput":
		val := getStr(node.Props["value"])
		b.WriteString(fmt.Sprintf("%s<input type=\"number\" value=\"%s\"%s />\n", pad, val, attrs))
		return
	case "TextArea":
		val := getStr(node.Props["value"])
		rows := 3
		if r, ok := node.Props["rows"].(int); ok {
			rows = r
		}
		b.WriteString(fmt.Sprintf("%s<textarea rows=\"%d\"%s>%s</textarea>\n", pad, rows, attrs, val))
		return
	case "Checkbox":
		checked := ""
		if v, ok := node.Props["checked"].(bool); ok && v {
			checked = " checked"
		}
		b.WriteString(fmt.Sprintf("%s<input type=\"checkbox\"%s%s />\n", pad, checked, attrs))
		return
	case "Image":
		if src, ok := node.Props["src"].(string); ok {
			b.WriteString(fmt.Sprintf("%s<img src=\"%s\"%s />\n", pad, src, attrs))
			return
		}
	case "Text":
		b.WriteString(fmt.Sprintf("%s<span%s>", pad, attrs))
		if content, ok := node.Props["content"].(string); ok {
			b.WriteString(content)
		}
		b.WriteString("</span>\n")
		return
	case "Button":
		b.WriteString(fmt.Sprintf("%s<button%s>", pad, attrs))
		if label, ok := node.Props["label"].(string); ok {
			b.WriteString(label)
		}
		b.WriteString("</button>\n")
		return
	case "CameraView":
		b.WriteString(fmt.Sprintf("%s<div%s>[Camera View]</div>\n", pad, attrs))
		return
	}

	// Default open tag
	b.WriteString(fmt.Sprintf("%s<%s%s>\n", pad, tag, attrs))

	// Children
	for _, child := range node.Children {
		renderNode(b, child, indent+1)
	}

	// Close tag
	b.WriteString(fmt.Sprintf("%s</%s>\n", pad, tag))
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

func styleAttr(s *core.Style) string {
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
	if s.Display != "" {
		styles = append(styles, fmt.Sprintf("display:%s", s.Display))
	}
	if s.Padding != (core.EdgeInsets{}) {
		styles = append(styles, fmt.Sprintf("padding:%dpx %dpx %dpx %dpx", s.Padding.Top, s.Padding.Right, s.Padding.Bottom, s.Padding.Left))
	}
	if s.Margin != (core.EdgeInsets{}) {
		styles = append(styles, fmt.Sprintf("margin:%dpx %dpx %dpx %dpx", s.Margin.Top, s.Margin.Right, s.Margin.Bottom, s.Margin.Left))
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
	if len(styles) == 0 {
		return ""
	}
	return fmt.Sprintf(" style=\"%s\"", strings.Join(styles, "; "))
}
