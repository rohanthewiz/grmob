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

// renderCanvas writes the <svg>, its gradients, and its shapes.
//
// # Gradients live in a leading <defs>, not inside their <path>
//
// SVG 2 allows a paint server as a child of the shape it paints, which would
// have kept each shape one self-contained element. Chrome does not resolve
// one there (a url(#id) fill pointing into a <path> paints nothing; checked
// headless), so the gradients go where every browser finds them: a <defs> at
// the head of the <svg>.
//
//	<svg viewBox="0 0 100 50">
//	  <defs data-grmob-chrome="gradients">          ← chrome: no node path
//	    <linearGradient id="grmob-root-0-fill-0" gradientUnits="userSpaceOnUse" …>
//	      <stop offset="0" stop-color="#2A78D666"/> …
//	  </defs>
//	  <path d="…" fill="url(#grmob-root-0-fill-0)"/>  ← still the canvas's
//	  <path d="…" stroke="…"/>                           i-th *node* child
//	</svg>
//
// It is marked data-grmob-chrome like a TabView's bar, which is how the live
// runtime's add-child patches skip it (chromeOffset), and it is written only
// when some shape has a gradient, so a flat canvas exports as it always did.
func renderCanvas(b *element.Builder, node *core.Node, attrs []string, path string) {
	lead := []string{
		"viewBox", "0 0 " + formatNumber(node.Props["vw"]) + " " + formatNumber(node.Props["vh"]),
		"preserveAspectRatio", preserveAspectRatio(getStr(node.Props["scale"])),
	}
	e := b.Ele("svg", withLead(attrs, lead...)...)
	var defs element.Element
	open := false
	for i, c := range node.Children {
		tag, gattrs, stops := CanvasGradient(c.Props, CanvasGradientID(path, i))
		if tag == "" {
			continue
		}
		if !open {
			defs, open = b.Ele("defs", "data-grmob-chrome", "gradients"), true
		}
		g := b.Ele(tag, gattrs...)
		for _, st := range stops {
			b.Ele("stop", st...).R()
		}
		g.R()
	}
	if open {
		defs.R()
	}
	for i, c := range node.Children {
		renderNode(b, c, imposed{}, childPath(path, i))
	}
	e.R()
}

// renderCanvasShape writes one <path>. The attribute set is CanvasShapeAttrs,
// which the WASM runtime restates as canvasShapeAttrs in grmob-runtime.js.
// path is the shape's own node path; its parent's is everything before the
// last slash, which is what the gradient id is scoped by.
func renderCanvasShape(b *element.Builder, node *core.Node, attrs []string, path string) {
	canvas, i := path, 0
	if slash := strings.LastIndexByte(path, '/'); slash >= 0 {
		canvas = path[:slash]
		i, _ = strconv.Atoi(path[slash+1:])
	}
	b.Ele("path", withLead(attrs, CanvasShapeAttrs(node.Props, CanvasGradientID(canvas, i))...)...).R()
}

// CanvasGradientID is the document id of the gradient shape i of the canvas
// at canvasPath fills with: the canvas's tab-style scope plus "-fill-i", so
// "root/0" shape 2 is "grmob-root-0-fill-2". Scoped by node path because a
// path is unique in the document, which an id must be, and because it is the
// one name both web targets can derive without talking to each other. The
// runtime restates it as canvasGradientId.
func CanvasGradientID(canvasPath string, i int) string {
	return tabScope(canvasPath) + "-fill-" + strconv.Itoa(i)
}

// CanvasGradient is the paint-server element for a shape's gradient props
// (see core.Gradient's wire keys): the tag, its attributes, and one attribute
// list per <stop>. tag is "" when the shape has no gradient, or when the keys
// are malformed (mismatched stop and colour counts, wrong geometry length),
// in which case the shape paints no fill rather than a wrong one.
//
// gradientUnits="userSpaceOnUse" puts the geometry in viewBox units, which is
// the contract core.Gradient states; SVG's default, objectBoundingBox, would
// read (0, 0)–(1, 1) as the shape's own bounds.
func CanvasGradient(props map[string]any, id string) (tag string, attrs []string, stops [][]string) {
	kind := getStr(props["gradient"])
	at := floats(props["gradientAt"])
	offsets := floats(props["gradientStops"])
	colors := strs(props["gradientColors"])
	if len(offsets) == 0 || len(offsets) != len(colors) {
		return "", nil, nil
	}
	num := func(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }
	switch {
	case kind == "linear" && len(at) == 4:
		tag = "linearGradient"
		attrs = []string{"id", id, "gradientUnits", "userSpaceOnUse",
			"x1", num(at[0]), "y1", num(at[1]), "x2", num(at[2]), "y2", num(at[3])}
	case kind == "radial" && len(at) == 3:
		tag = "radialGradient"
		attrs = []string{"id", id, "gradientUnits", "userSpaceOnUse",
			"cx", num(at[0]), "cy", num(at[1]), "r", num(at[2])}
	default:
		return "", nil, nil
	}
	for i, o := range offsets {
		stops = append(stops, []string{"offset", num(o), "stop-color", colors[i]})
	}
	return tag, attrs, stops
}

// strs reads a []string prop, accepting the []any a JSON-decoded node carries.
func strs(v any) []string {
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		out := make([]string, 0, len(s))
		for _, e := range s {
			str, ok := e.(string)
			if !ok {
				return nil
			}
			out = append(out, str)
		}
		return out
	}
	return nil
}

// CanvasShapeAttrs is the SVG attribute list for one CanvasShape's props, as
// name/value pairs in a fixed order. Exported so wasm/verify can hold the
// runtime's copy to it.
//
// fill="none" is written for a shape with no fill because SVG's default fill
// is black — the one place the two vocabularies disagree about "unset".
//
// gradientID is the id CanvasGradient's element carries for this shape (see
// CanvasGradientID); a shape with a well-formed gradient fills with a
// reference to it. A malformed one falls to fill="none", as CanvasGradient
// writes no element for it and a reference to nothing would paint black in
// some engines rather than nothing.
func CanvasShapeAttrs(props map[string]any, gradientID string) []string {
	out := []string{"d", PathData(floats(props["d"]))}
	fill := getStr(props["fill"])
	if tag, _, _ := CanvasGradient(props, gradientID); tag != "" {
		fill = "url(#" + gradientID + ")"
	}
	if fill != "" {
		out = append(out, "fill", fill)
		if rule := getStr(props["fillRule"]); rule == "evenodd" {
			out = append(out, "fill-rule", rule)
		}
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
