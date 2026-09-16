package htmlout

import (
	"strconv"
	"strings"

	"github.com/rohanthewiz/element"
	"github.com/rohanthewiz/grmob/core"
)

// core.Canvas, on the HTML side.
//
// # An <svg>, and its shapes are <path> children
//
// SVG is the browser's own vector primitive, and it already means exactly what
// core.Canvas means: a viewBox mapped onto a box, shapes painted in document
// order, fill then stroke. So the export is a transliteration rather than a
// drawing — each CanvasShape node is one <path>, which also keeps the patch
// addressing positional in the live runtime (a shape's element is the canvas's
// i-th child, the way a GridRow is a <pre>'s).
//
//	core.Canvas(100, 50, shapes, core.CanvasStretch)
//	    ↓
//	<svg viewBox="0 0 100 50" preserveAspectRatio="none"
//	     style="display:block; width:100%; aspect-ratio:100 / 50; …author…">
//	  <path d="M0 0L50 25" fill="none" stroke="#000" stroke-width="2"
//	        vector-effect="non-scaling-stroke"/>
//	</svg>
//
// # The chassis, and why it goes first
//
// An <svg> in HTML sizes like a replaced element whose default is 300 × 150,
// which matches nothing core promises. The chassis states core's rule instead —
// fill the width, take the viewBox's aspect ratio — ahead of the author's
// declarations, so a Width or Height the author set wins by coming later. Once
// both are set, CSS ignores aspect-ratio on its own, which is the same "a
// stated Height turns the ratio off" the natives implement by hand.
//
// overflow:visible, because a stroke centred on the viewBox's edge (a chart's
// baseline at y = h) is half outside it, and an <svg> clips at its box by
// default where the natives' canvases do not.

// canvasChassis is the fixed CSS of a Canvas, as a function of its viewBox.
func canvasChassis(props map[string]any) string {
	return "display:block; width:100%; overflow:visible; aspect-ratio:" +
		formatNumber(props["vw"]) + " / " + formatNumber(props["vh"])
}

// preserveAspectRatio is the SVG spelling of a core.CanvasScale. Anything but
// "stretch" is fit, the default core writes and the only other value.
func preserveAspectRatio(scale string) string {
	if scale == string(core.CanvasStretch) {
		return "none"
	}
	return "xMidYMid meet"
}

// renderCanvas writes the <svg> and its shapes.
func renderCanvas(b *element.Builder, node *core.Node, attrs []string, path string) {
	lead := []string{
		"viewBox", "0 0 " + formatNumber(node.Props["vw"]) + " " + formatNumber(node.Props["vh"]),
		"preserveAspectRatio", preserveAspectRatio(getStr(node.Props["scale"])),
	}
	e := b.Ele("svg", withLead(attrs, lead...)...)
	for i, c := range node.Children {
		renderNode(b, c, imposed{}, childPath(path, i))
	}
	e.R()
}

// renderCanvasShape writes one <path>. The attribute set is canvasShapeAttrs,
// which the WASM runtime restates as canvasShapeAttrs in grmob-runtime.js.
func renderCanvasShape(b *element.Builder, node *core.Node, attrs []string) {
	b.Ele("path", withLead(attrs, CanvasShapeAttrs(node.Props)...)...).R()
}

// CanvasShapeAttrs is the SVG attribute list for one CanvasShape's props, as
// name/value pairs in a fixed order. Exported so wasm/verify can hold the
// runtime's copy to it.
//
// fill="none" is written for a shape with no fill because SVG's default fill
// is black — the one place the two vocabularies disagree about "unset".
func CanvasShapeAttrs(props map[string]any) []string {
	out := []string{"d", PathData(floats(props["d"]))}
	if fill := getStr(props["fill"]); fill != "" {
		out = append(out, "fill", fill)
	} else {
		out = append(out, "fill", "none")
	}
	stroke := getStr(props["stroke"])
	if stroke == "" {
		return out
	}
	out = append(out,
		"stroke", stroke,
		"stroke-width", formatNumber(props["strokeWidth"]),
		// Strokes are layout units on every target; see core.Canvas.
		"vector-effect", "non-scaling-stroke",
	)
	if v := getStr(props["cap"]); v != "" {
		out = append(out, "stroke-linecap", v)
	}
	if v := getStr(props["join"]); v != "" {
		out = append(out, "stroke-linejoin", v)
	}
	if dash := floats(props["dash"]); len(dash) > 0 {
		parts := make([]string, len(dash))
		for i, d := range dash {
			parts[i] = strconv.FormatFloat(d, 'g', -1, 64)
		}
		out = append(out, "stroke-dasharray", strings.Join(parts, " "))
	}
	return out
}

// PathData turns core's flat path opcodes into an SVG path string. The
// opcodes are core.PathMove, PathLine, PathCubic and PathClose, and SVG has a
// command letter for each with the same operands in the same order, so this is
// a spelling change and nothing else. A truncated or unknown operation ends the
// path there: a renderer must not fail a drawing over one bad shape.
func PathData(ops []float64) string {
	var sb strings.Builder
	num := func(v float64) {
		sb.WriteString(strconv.FormatFloat(v, 'g', -1, 64))
	}
	for i := 0; i < len(ops); {
		var letter byte
		var operands int
		switch ops[i] {
		case core.PathMove:
			letter, operands = 'M', 2
		case core.PathLine:
			letter, operands = 'L', 2
		case core.PathCubic:
			letter, operands = 'C', 6
		case core.PathClose:
			letter, operands = 'Z', 0
		default:
			return sb.String()
		}
		if i+operands >= len(ops) && operands > 0 {
			return sb.String()
		}
		sb.WriteByte(letter)
		for j := 1; j <= operands; j++ {
			if j > 1 {
				sb.WriteByte(' ')
			}
			num(ops[i+j])
		}
		i += 1 + operands
	}
	return sb.String()
}

// floats reads a []float64 prop, accepting the []any a node decoded from JSON
// carries as well as the typed slice core builds.
func floats(v any) []float64 {
	switch s := v.(type) {
	case []float64:
		return s
	case []any:
		out := make([]float64, 0, len(s))
		for _, e := range s {
			f, ok := e.(float64)
			if !ok {
				return out
			}
			out = append(out, f)
		}
		return out
	}
	return nil
}
