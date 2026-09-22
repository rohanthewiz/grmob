package core

import (
	"math"
	"reflect"
	"testing"
)

// cubicAt evaluates a cubic Bézier at t.
func cubicAt(t, p0, p1, p2, p3 float64) float64 {
	u := 1 - t
	return u*u*u*p0 + 3*u*u*t*p1 + 3*u*t*t*p2 + t*t*t*p3
}

// segments walks a path's ops and calls fn for each cubic with its start point.
func eachCubic(t *testing.T, ops []float64, fn func(x0, y0 float64, c []float64)) {
	t.Helper()
	var x, y float64
	for i := 0; i < len(ops); {
		switch ops[i] {
		case PathMove, PathLine:
			x, y = ops[i+1], ops[i+2]
			i += 3
		case PathCubic:
			fn(x, y, ops[i+1:i+7])
			x, y = ops[i+5], ops[i+6]
			i += 7
		case PathClose:
			i++
		default:
			t.Fatalf("unknown opcode %v at %d", ops[i], i)
		}
	}
}

// Every point sampled along an arc lies on its circle, for sweeps in both
// directions and of sizes that split into one, two and four segments.
func TestArcStaysOnTheCircle(t *testing.T) {
	const cx, cy, r = 50, 40, 30
	for _, c := range []struct{ start, sweep float64 }{
		{0, 90}, {30, 135}, {-90, 360}, {200, -250}, {10, 45},
	} {
		p := NewPath().Arc(cx, cy, r, c.start, c.sweep)
		n := 0
		eachCubic(t, p.ops, func(x0, y0 float64, k []float64) {
			n++
			for s := 0.0; s <= 1; s += 0.125 {
				x := cubicAt(s, x0, k[0], k[2], k[4])
				y := cubicAt(s, y0, k[1], k[3], k[5])
				if dev := math.Abs(math.Hypot(x-cx, y-cy)-r) / r; dev > 3e-4 {
					t.Errorf("arc %v: point at t=%g is %.4f%% off the radius", c, s, dev*100)
				}
			}
		})
		if want := int(math.Ceil(math.Abs(c.sweep) / 90)); n != want {
			t.Errorf("arc %v: %d segments, want %d", c, n, want)
		}
		// The arc ends where the sweep says, which is the half of correctness
		// the radius check cannot see (a wrong-direction arc is still round).
		end := (c.start + c.sweep) * math.Pi / 180
		gx, gy := p.cx, p.cy
		if math.Abs(gx-(cx+r*math.Cos(end))) > 1e-9 || math.Abs(gy-(cy+r*math.Sin(end))) > 1e-9 {
			t.Errorf("arc %v ends at (%g, %g)", c, gx, gy)
		}
	}
}

// Clockwise on screen: a 90° sweep from three o'clock ends at six o'clock,
// which in a y-down space is +y.
func TestArcSweepIsClockwiseOnScreen(t *testing.T) {
	p := NewPath().Arc(0, 0, 10, 0, 90)
	if math.Abs(p.cx) > 1e-9 || math.Abs(p.cy-10) > 1e-9 {
		t.Fatalf("0→90° ended at (%g, %g), want (0, 10)", p.cx, p.cy)
	}
}

// The raised cubic traces the quadratic exactly.
func TestQuadToIsTheSameCurve(t *testing.T) {
	p := NewPath().MoveTo(0, 0).QuadTo(50, 100, 100, 0)
	eachCubic(t, p.ops, func(x0, y0 float64, k []float64) {
		for s := 0.0; s <= 1; s += 0.1 {
			u := 1 - s
			qx := u*u*0 + 2*u*s*50 + s*s*100
			qy := u*u*0 + 2*u*s*100 + s*s*0
			if math.Abs(cubicAt(s, x0, k[0], k[2], k[4])-qx) > 1e-9 ||
				math.Abs(cubicAt(s, y0, k[1], k[3], k[5])-qy) > 1e-9 {
				t.Fatalf("cubic leaves the quadratic at t=%g", s)
			}
		}
	})
}

func TestPathShapes(t *testing.T) {
	if got, want := Rect(1, 2, 3, 4).ops, []float64{0, 1, 2, 1, 4, 2, 1, 4, 6, 1, 1, 6, 3}; !reflect.DeepEqual(got, want) {
		t.Errorf("Rect ops = %v", got)
	}
	if got, want := Polyline(0, 0, 10, 5, 20, 0, 99).ops, []float64{0, 0, 0, 1, 10, 5, 1, 20, 0}; !reflect.DeepEqual(got, want) {
		t.Errorf("Polyline ops = %v", got)
	}
	// A donut sector is one closed subpath: a move, two arcs joined by a
	// line, and a close.
	moves, lines, closes := 0, 0, 0
	for i, ops := 0, Sector(0, 0, 5, 10, 0, 90).ops; i < len(ops); {
		switch ops[i] {
		case PathMove:
			moves++
			i += 3
		case PathLine:
			lines++
			i += 3
		case PathCubic:
			i += 7
		case PathClose:
			closes++
			i++
		}
	}
	if moves != 1 || lines != 1 || closes != 1 {
		t.Errorf("Sector: %d moves, %d lines, %d closes", moves, lines, closes)
	}
}

func TestCoordinatePlaces(t *testing.T) {
	for extent, want := range map[float64]int{1: 4, 100: 2, 360: 2, 1000: 1, 50000: 0} {
		if got := coordinatePlaces(extent); got != want {
			t.Errorf("coordinatePlaces(%g) = %d, want %d", extent, got, want)
		}
	}
}

func renderCanvas(v View) *Node {
	ctx := NewContext()
	ctx.BeginRenderPass()
	return v.Render(ctx)
}

func TestCanvasNode(t *testing.T) {
	n := renderCanvas(Canvas(100, 50, []Shape{
		{Path: Line(0, 0, 33.333333, 10), Stroke: "#000", Cap: CapRound, Dash: []float64{4, 2}},
		{Fill: "#f00"},
	}, CanvasStretch))

	if n.Type != "Canvas" || n.Props["vw"] != 100.0 || n.Props["vh"] != 50.0 || n.Props["scale"] != "stretch" {
		t.Fatalf("canvas props = %v", n.Props)
	}
	if !n.Style.AccessibilityHidden {
		t.Error("an unlabelled canvas is not hidden")
	}
	if len(n.Children) != 2 {
		t.Fatalf("%d children, want 2", len(n.Children))
	}
	line := n.Children[0].Props
	if !reflect.DeepEqual(line["d"], []float64{0, 0, 0, 1, 33.33, 10}) {
		t.Errorf("d = %v, want coordinates rounded to 2 places", line["d"])
	}
	if line["strokeWidth"] != 1.0 || line["cap"] != "round" || line["join"] != nil {
		t.Errorf("stroke props = %v", line)
	}
	empty := n.Children[1].Props
	if d, ok := empty["d"].([]float64); !ok || d == nil || len(d) != 0 {
		t.Errorf("pathless shape d = %#v, want an empty non-nil slice", empty["d"])
	}
	if _, ok := empty["strokeWidth"]; ok {
		t.Error("an unstroked shape carries a stroke width")
	}
}

func TestLabelledCanvasIsAnImage(t *testing.T) {
	n := renderCanvas(Canvas(10, 10, nil, AccessibilityLabel("Sales rising")))
	if n.Style.AccessibilityHidden || n.Style.AccessibilityRole != RoleImg {
		t.Errorf("labelled canvas: hidden=%v role=%q", n.Style.AccessibilityHidden, n.Style.AccessibilityRole)
	}
	if n.Props["scale"] != "fit" {
		t.Errorf("default scale = %v", n.Props["scale"])
	}
}

func TestCanvasMapping(t *testing.T) {
	// A 100×50 drawing in a 300×300 box: fit scales by 3 and centres vertically.
	sx, sy, ox, oy := CanvasMapping(100, 50, 300, 300, CanvasFit)
	if sx != 3 || sy != 3 || ox != 0 || oy != 75 {
		t.Errorf("fit = %g %g %g %g", sx, sy, ox, oy)
	}
	sx, sy, ox, oy = CanvasMapping(100, 50, 300, 300, CanvasStretch)
	if sx != 3 || sy != 6 || ox != 0 || oy != 0 {
		t.Errorf("stretch = %g %g %g %g", sx, sy, ox, oy)
	}
}

// FillEvenOdd crosses the wire only with a fill; the nonzero default and a
// rule on an unfilled shape write nothing.
func TestFillRuleIsWrittenOnlyWhenItPaints(t *testing.T) {
	ring := NewPath().Arc(5, 5, 4, 0, 360).Close().Arc(5, 5, 2, 0, 360).Close()
	n := renderCanvas(Canvas(10, 10, []Shape{
		{Path: ring, Fill: "#000", FillRule: FillEvenOdd},
		{Path: ring, Fill: "#000", FillRule: FillNonZero},
		{Path: ring, Stroke: "#000", FillRule: FillEvenOdd},
	}))
	if got := n.Children[0].Props["fillRule"]; got != "evenodd" {
		t.Errorf("even-odd fill: fillRule = %v", got)
	}
	for i, c := range n.Children[1:] {
		if _, ok := c.Props["fillRule"]; ok {
			t.Errorf("shape %d carries fillRule %v; only a filled even-odd shape should", i+1, c.Props["fillRule"])
		}
	}
}

// A gradient that paints replaces "fill" with the four gradient keys; stops
// are clamped and made non-decreasing; the degenerate cases collapse to a
// flat fill in the last colour; and the fill rule rides along with either.
func TestGradientWire(t *testing.T) {
	box := Rect(0, 0, 100, 50)
	n := renderCanvas(Canvas(100, 50, []Shape{
		// 0: linear, with an out-of-range and an out-of-order stop.
		{Path: box, Fill: "#ff0000", FillRule: FillEvenOdd, FillGradient: LinearGradientFill(0, 0, 0, 50.123456,
			Stop(-1, "#000000"), Stop(0.7, "#111111"), Stop(0.3, "#222222"), Stop(2, "#333333"))},
		// 1: radial.
		{Path: box, FillGradient: RadialGradientFill(50, 25, 20, Stop(0, "#ffffff"), Stop(1, "#ffffff00"))},
		// 2: one stop is a flat fill.
		{Path: box, FillGradient: LinearGradientFill(0, 0, 10, 0, Stop(0.5, "#abcdef"))},
		// 3: a zero-length line is a flat fill in the last colour.
		{Path: box, FillGradient: LinearGradientFill(5, 5, 5, 5, Stop(0, "#000000"), Stop(1, "#123456"))},
		// 4: a non-positive radius, likewise.
		{Path: box, FillGradient: RadialGradientFill(5, 5, 0, Stop(0, "#000000"), Stop(1, "#654321"))},
		// 5: no stops paints nothing, so the flat Fill stands.
		{Path: box, Fill: "#00ff00", FillGradient: LinearGradientFill(0, 0, 1, 1)},
	}))
	p := n.Children[0].Props
	if _, ok := p["fill"]; ok {
		t.Errorf("a painting gradient also wrote fill %v; the two keys are exclusive", p["fill"])
	}
	if p["gradient"] != "linear" || p["fillRule"] != "evenodd" {
		t.Errorf("linear: gradient %v, fillRule %v", p["gradient"], p["fillRule"])
	}
	if got, want := p["gradientAt"], []float64{0, 0, 0, 50.12}; !reflect.DeepEqual(got, want) {
		t.Errorf("gradientAt = %v, want %v (rounded to the canvas's places)", got, want)
	}
	if got, want := p["gradientStops"], []float64{0, 0.7, 0.7, 1}; !reflect.DeepEqual(got, want) {
		t.Errorf("gradientStops = %v, want %v", got, want)
	}
	if got, want := p["gradientColors"], []string{"#000000", "#111111", "#222222", "#333333"}; !reflect.DeepEqual(got, want) {
		t.Errorf("gradientColors = %v, want %v", got, want)
	}
	r := n.Children[1].Props
	if r["gradient"] != "radial" || !reflect.DeepEqual(r["gradientAt"], []float64{50, 25, 20}) {
		t.Errorf("radial: %v %v", r["gradient"], r["gradientAt"])
	}
	for i, want := range map[int]string{2: "#abcdef", 3: "#123456", 4: "#654321", 5: "#00ff00"} {
		c := n.Children[i].Props
		if c["fill"] != want {
			t.Errorf("shape %d: fill = %v, want %s", i, c["fill"], want)
		}
		if _, ok := c["gradient"]; ok {
			t.Errorf("shape %d: a degenerate gradient reached the wire", i)
		}
	}
}

// A stroke gradient writes the same four keys under the "stroke" prefix, in
// place of "stroke", and brings the stroke's width and style with it; a
// degenerate one is the flat stroke it reduces to; one that paints nothing
// leaves the flat Stroke standing; and a fill and a stroke gradient on one
// shape do not share keys.
func TestStrokeGradientWire(t *testing.T) {
	box := Rect(0, 0, 100, 50)
	n := renderCanvas(Canvas(100, 50, []Shape{
		// 0: both gradients, and a cap.
		{Path: box,
			FillGradient:   LinearGradientFill(0, 0, 0, 50, Stop(0, "#000000"), Stop(1, "#ffffff")),
			StrokeGradient: RadialGradientFill(50, 25, 30, Stop(0, "#2a78d6"), Stop(1, "#eb6834")),
			Stroke:         "#ff0000", StrokeWidth: 3, Cap: CapRound},
		// 1: a zero-length line is a flat stroke in the last colour, with the
		// default width.
		{Path: box, StrokeGradient: LinearGradientFill(5, 5, 5, 5, Stop(0, "#000000"), Stop(1, "#123456"))},
		// 2: no stops, so the flat Stroke stands.
		{Path: box, Stroke: "#00ff00", StrokeGradient: LinearGradientFill(0, 0, 1, 1)},
		// 3: no stops and no Stroke is no stroke at all.
		{Path: box, StrokeWidth: 4, StrokeGradient: LinearGradientFill(0, 0, 1, 1)},
	}))
	p := n.Children[0].Props
	if _, ok := p["stroke"]; ok {
		t.Errorf("a painting stroke gradient also wrote stroke %v", p["stroke"])
	}
	if p["gradient"] != "linear" || !reflect.DeepEqual(p["gradientAt"], []float64{0, 0, 0, 50}) {
		t.Errorf("the fill gradient's keys moved: %v %v", p["gradient"], p["gradientAt"])
	}
	if p["strokeGradient"] != "radial" || !reflect.DeepEqual(p["strokeGradientAt"], []float64{50, 25, 30}) {
		t.Errorf("stroke gradient: %v %v", p["strokeGradient"], p["strokeGradientAt"])
	}
	if !reflect.DeepEqual(p["strokeGradientStops"], []float64{0, 1}) ||
		!reflect.DeepEqual(p["strokeGradientColors"], []string{"#2a78d6", "#eb6834"}) {
		t.Errorf("stroke stops %v colours %v", p["strokeGradientStops"], p["strokeGradientColors"])
	}
	if p["strokeWidth"] != 3.0 || p["cap"] != "round" {
		t.Errorf("a gradient stroke lost its style: width %v cap %v", p["strokeWidth"], p["cap"])
	}
	for i, want := range map[int]string{1: "#123456", 2: "#00ff00"} {
		c := n.Children[i].Props
		if c["stroke"] != want || c["strokeWidth"] != 1.0 {
			t.Errorf("shape %d: stroke %v width %v, want %s at 1", i, c["stroke"], c["strokeWidth"], want)
		}
		if _, ok := c["strokeGradient"]; ok {
			t.Errorf("shape %d: a degenerate stroke gradient reached the wire", i)
		}
	}
	if c := n.Children[3].Props; len(c) != 1 {
		t.Errorf("shape 3 paints nothing, so only d belongs on the wire: %v", c)
	}
	if got := GradientKey("stroke", "gradientColors"); got != "strokeGradientColors" {
		t.Errorf("GradientKey = %q", got)
	}
}

// CanvasMirrorsRTL is opt-in and writes to the wire only when set, so every
// Canvas that does not ask for it sends exactly the patch it sent before the
// prop existed. It means nothing to any other node.
func TestCanvasMirrorIsOptInAndCanvasOnly(t *testing.T) {
	render := func(v View) *Node {
		ctx := NewContext()
		ctx.BeginRenderPass()
		return v.Render(ctx)
	}
	shapes := []Shape{{Path: Line(0, 0, 10, 10), Stroke: "#000"}}
	if _, ok := render(Canvas(10, 10, shapes)).Props["mirror"]; ok {
		t.Error("a Canvas that did not ask to mirror carries a mirror prop")
	}
	if got := render(Canvas(10, 10, shapes, CanvasMirrorsRTL)).Props["mirror"]; got != true {
		t.Errorf("CanvasMirrorsRTL wrote mirror = %v, want true", got)
	}
	if _, ok := render(Canvas(10, 10, shapes, CanvasMirrorsRTL, CanvasMirror(false))).Props["mirror"]; ok {
		t.Error("a later CanvasMirror(false) should take the prop back off")
	}
	if _, ok := render(Column(CanvasMirrorsRTL)).Props["mirror"]; ok {
		t.Error("CanvasMirrorsRTL wrote a prop on a Column")
	}
}

// MirrorCanvasMapping reflects about the box's vertical centre line: the
// viewBox's left edge lands on the box's right edge and the other way round,
// under both scales, and y is untouched by construction (it is not an input).
func TestMirrorCanvasMappingReflectsTheBox(t *testing.T) {
	for _, scale := range []CanvasScale{CanvasStretch, CanvasFit} {
		sx, _, ox, _ := CanvasMapping(10, 40, 200, 100, scale)
		msx, mox := MirrorCanvasMapping(sx, ox, 200)
		for _, x := range []float64{0, 3, 10} {
			plain := x*sx + ox
			mirrored := x*msx + mox
			if math.Abs(mirrored-(200-plain)) > 1e-9 {
				t.Errorf("%s: x=%v lands at %v mirrored, want %v", scale, x, mirrored, 200-plain)
			}
		}
	}
}
