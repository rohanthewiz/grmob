package core

import "math"

// Canvas draws vector shapes — lines, curves, arcs, filled polygons — in a
// coordinate space of its own, scaled into whatever box the layout gives it.
//
//	p := core.NewPath().MoveTo(0, 80).LineTo(40, 20).LineTo(100, 50)
//
//	core.Canvas(100, 100, []core.Shape{
//	    {Path: core.Circle(50, 50, 48), Fill: t.Colors.Surface},
//	    {Path: p, Stroke: t.Colors.Primary, StrokeWidth: 2, Cap: core.CapRound},
//	}, core.Width("100%"))
//
// It is the one primitive in core that can join two arbitrary points, which is
// what line, area, pie and scatter charts are made of. Everything else in the
// vocabulary is a flex box, and a box can only fake a diagonal by turning.
//
// # The coordinate space is not the size
//
// w × h is a viewBox: the units the shapes are written in, with the origin at
// the top-left and y growing downwards (screen convention, which is what all
// three platforms' drawing APIs use). The node's size on screen comes from its
// Style like any other node's, and the drawing is mapped onto that box by the
// CanvasScale prop:
//
//	CanvasFit      one scale for both axes, centred — a clock face stays round
//	               in a wide box. The default. SVG `xMidYMid meet`.
//	CanvasStretch  each axis scaled on its own — a line chart spans a wide box.
//	               SVG `none`.
//
// A Canvas whose Style gives no Height takes the viewBox's aspect ratio, and
// one that gives no Width fills its parent's width, so the common case — a
// chart as wide as the screen — needs no sizing props at all.
//
// # Strokes are in layout units, never scaled
//
// A StrokeWidth of 2 is 2 px (dp, pt) whatever the scale, on every target.
// Scaled strokes would make a stretched chart's lines fat on one axis and thin
// on the other, and would make the same chart's lines change weight between a
// phone and a tablet. SVG says this with vector-effect: non-scaling-stroke; the
// natives transform the path's points and stroke the result untransformed,
// which is the same thing.
//
// # Four opcodes on the wire
//
// Path offers moves, lines, quadratic and cubic Béziers, and circular arcs, but
// what reaches a renderer is only move, line, cubic and close. Quadratics are
// raised to cubics exactly, and arcs are approximated by cubic segments of at
// most 90° (error under 0.03% of the radius) — in Go, once.
//
// The alternative was teaching three languages four arc conventions: SVG's
// endpoint form with its large-arc and sweep flags, Compose's bounding
// rectangle with degrees, SwiftUI's centre with radians and a clockwise flag
// that is flipped in a y-down space. Each is a place for the three targets to
// disagree by a sign, and a disagreement there draws a plausible wrong
// picture rather than failing. Every renderer already has moveTo, lineTo,
// cubicTo and close with identical meaning, so that is the whole contract.
//
// # One child per shape
//
// A Canvas is a container of CanvasShape nodes, the shape TextGrid has with
// its rows and for the same reason: the reconciler pairs children by index and
// compares props by value, so a pass that moves one hand of a clock sends one
// update-props patch and leaves the face alone. Shapes are drawn in order, so
// a later shape paints over an earlier one.
//
// Shapes take no props of their own and are never built directly by app code.
// Behavior props (OnClick, ...) apply to the canvas as a whole.
//
// # Accessibility
//
// A drawing has no text a reader could find in it. A Canvas without an
// AccessibilityLabel is treated as decoration and hidden; one with a label is
// a single image element that speaks it. A chart should always be given one
// that states what the chart shows, not that it is a chart.
//
// # Not in v1
//
// Text inside the drawing (lay labels out around it as Text nodes), gradients,
// clipping, per-shape hit-testing, and the even-odd fill rule. Fills use the
// nonzero rule, which is every target's default.
func Canvas(w, h float64, shapes []Shape, props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		if w <= 0 {
			w = 100
		}
		if h <= 0 {
			h = 100
		}
		n := leafNode(ctx, "Canvas", Style{}, map[string]any{
			"vw":    w,
			"vh":    h,
			"scale": string(CanvasFit),
		}, props)
		// Decoration unless named; see "Accessibility" above. Written into the
		// style rather than left for each renderer to infer, so the four
		// targets share one rule instead of four readings of it.
		if n.Style.AccessibilityLabel == "" {
			n.Style.AccessibilityHidden = true
		} else if n.Style.AccessibilityRole == RoleNone {
			n.Style.AccessibilityRole = RoleImg
		}

		places := coordinatePlaces(max(w, h))
		n.Children = make([]*Node, 0, len(shapes))
		for _, s := range shapes {
			n.Children = append(n.Children, s.node(places))
		}
		return n
	})
}

// CanvasScale says how a Canvas's viewBox is mapped onto its box. It is passed
// among a Canvas's props like a StyleProp:
//
//	core.Canvas(200, 100, shapes, core.CanvasStretch, core.Height("160px"))
type CanvasScale string

const (
	// CanvasFit scales both axes by the same factor, the largest that fits
	// the whole drawing in the box, and centres it. The default.
	CanvasFit CanvasScale = "fit"

	// CanvasStretch scales each axis independently so the viewBox exactly
	// fills the box. Shapes distort; strokes do not (see Canvas).
	CanvasStretch CanvasScale = "stretch"
)

// Apply makes a CanvasScale a BehaviorProp: it writes a node prop rather than
// a Style field, because it means nothing to any node but a Canvas.
func (c CanvasScale) Apply(_ *Context, n *Node) {
	if n.Type != "Canvas" {
		return
	}
	if c != CanvasStretch {
		c = CanvasFit
	}
	n.Props["scale"] = string(c)
}

// CanvasMapping is how a w × h viewBox lands in a boxW × boxH box under a
// scale: a viewBox point (x, y) is drawn at (x·sx + ox, y·sy + oy).
//
//	CanvasStretch  sx = boxW/w, sy = boxH/h, no offset
//	CanvasFit      sx = sy = the smaller of the two, and the slack on the
//	               other axis split evenly — SVG's "xMidYMid meet"
//
// The web targets never call this; they hand the viewBox to SVG, which
// applies the same rule itself. It is the statement the two native renderers
// restate (canvasViewport in GrMobCanvasGeometry.kt, the Swift equivalent),
// and internal/canvasfixture holds them to it. A non-positive viewBox side
// reads as 100, which is what Canvas writes for one.
func CanvasMapping(w, h, boxW, boxH float64, scale CanvasScale) (sx, sy, ox, oy float64) {
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 100
	}
	sx, sy = boxW/w, boxH/h
	if scale == CanvasStretch {
		return sx, sy, 0, 0
	}
	k := min(sx, sy)
	return k, k, (boxW - w*k) / 2, (boxH - h*k) / 2
}

// LineCap is how an open stroke's ends are drawn.
type LineCap string

const (
	CapButt   LineCap = "butt" // flush with the end point; the default
	CapRound  LineCap = "round"
	CapSquare LineCap = "square"
)

// LineJoin is how a stroke turns a corner.
type LineJoin string

const (
	JoinMiter LineJoin = "miter" // the default
	JoinRound LineJoin = "round"
	JoinBevel LineJoin = "bevel"
)

// Shape is one path drawn once: filled, stroked, or both (fill first, then the
// stroke over it, on every target). A shape with neither colour draws nothing
// and still occupies its child slot, which keeps the slots of the shapes after
// it stable while it comes and goes.
type Shape struct {
	Path *Path

	// Fill and Stroke are CSS colours ("#rrggbb", "#rrggbbaa"); "" for none.
	Fill   string
	Stroke string

	// StrokeWidth is in layout units, not viewBox units; 0 means 1.
	StrokeWidth float64

	Cap  LineCap
	Join LineJoin

	// Dash alternates dash and gap lengths, in layout units like StrokeWidth.
	// Nil for a solid line.
	Dash []float64
}

// node is the wire form of one shape. Only set fields are written, so an
// unchanged shape compares equal across passes and a solid, unfilled line
// costs four keys.
func (s Shape) node(places int) *Node {
	props := map[string]any{}
	var d []float64
	if s.Path != nil {
		d = s.Path.wire(places)
	}
	// Always present, and never nil: a nil and an empty slice would compare
	// unequal across passes and patch a shape that did not change.
	if d == nil {
		d = []float64{}
	}
	props["d"] = d
	if s.Fill != "" {
		props["fill"] = s.Fill
	}
	if s.Stroke != "" {
		props["stroke"] = s.Stroke
		w := s.StrokeWidth
		if w <= 0 {
			w = 1
		}
		props["strokeWidth"] = w
		if s.Cap != "" && s.Cap != CapButt {
			props["cap"] = string(s.Cap)
		}
		if s.Join != "" && s.Join != JoinMiter {
			props["join"] = string(s.Join)
		}
		if len(s.Dash) > 0 {
			props["dash"] = append([]float64(nil), s.Dash...)
		}
	}
	return &Node{Type: "CanvasShape", Props: props}
}

// Wire opcodes. The numbers after each are its operands:
//
//	0 x y                   move to
//	1 x y                   line to
//	2 x1 y1 x2 y2 x y       cubic Bézier to, via two control points
//	3                       close the subpath
//
// Held to the renderers by the canvas tests in mobile/verify and wasm/verify.
const (
	PathMove  = 0
	PathLine  = 1
	PathCubic = 2
	PathClose = 3
)

// Path is a sequence of subpaths built by chained calls. The builder methods
// mutate and return the receiver, so a Path should be finished before it is
// handed to a Shape: the node takes a copy when the Canvas renders, and a
// change after that is invisible (see Node immutability).
type Path struct {
	ops []float64

	// The current point and the start of the current subpath, which LineTo's
	// successors and Close need, and whether there is a current point at all.
	cx, cy float64
	sx, sy float64
	open   bool
}

// NewPath returns an empty path.
func NewPath() *Path { return &Path{} }

// MoveTo starts a new subpath at (x, y).
func (p *Path) MoveTo(x, y float64) *Path {
	p.ops = append(p.ops, PathMove, x, y)
	p.cx, p.cy, p.sx, p.sy, p.open = x, y, x, y, true
	return p
}

// LineTo draws a straight segment to (x, y). With no current point it moves
// there instead, which is what every platform's API does and what makes a
// polyline a single loop body.
func (p *Path) LineTo(x, y float64) *Path {
	if !p.open {
		return p.MoveTo(x, y)
	}
	p.ops = append(p.ops, PathLine, x, y)
	p.cx, p.cy = x, y
	return p
}

// CubicTo draws a cubic Bézier to (x, y) via control points (x1, y1) and
// (x2, y2).
func (p *Path) CubicTo(x1, y1, x2, y2, x, y float64) *Path {
	if !p.open {
		p.MoveTo(p.cx, p.cy)
	}
	p.ops = append(p.ops, PathCubic, x1, y1, x2, y2, x, y)
	p.cx, p.cy = x, y
	return p
}

// QuadTo draws a quadratic Bézier to (x, y) via control point (qx, qy). It is
// sent as the cubic that traces the identical curve: each cubic control point
// sits two thirds of the way from an end point to the quadratic one.
func (p *Path) QuadTo(qx, qy, x, y float64) *Path {
	x0, y0 := p.cx, p.cy
	return p.CubicTo(
		x0+2.0/3.0*(qx-x0), y0+2.0/3.0*(qy-y0),
		x+2.0/3.0*(qx-x), y+2.0/3.0*(qy-y),
		x, y,
	)
}

// Arc draws part of a circle centred on (cx, cy) with radius r, starting at
// startDeg and sweeping sweepDeg. Angles are degrees clockwise from the
// positive x-axis (three o'clock) — clockwise on screen, the same sense as
// core.Rotate — and a negative sweep goes anticlockwise.
//
// If the path has a current point, a straight line joins it to the arc's
// start, so a pie wedge is MoveTo(centre).Arc(...).Close(). Otherwise the arc
// starts a new subpath.
//
// # How it is approximated
//
// The sweep is split into segments of at most 90°, and each becomes the cubic
// whose control points lie on the tangents at its ends, at distance
// k = 4/3 · tan(θ/4) · r. That k makes the curve's midpoint lie exactly on the
// circle; the worst radial error for a 90° segment is about 0.027% of r, below
// a pixel for any radius a phone can show.
//
//	P1 ──k── C1
//	           ╲         C1 = P1 + k · tangent(P1)
//	            C2       C2 = P2 − k · tangent(P2)
//	             │
//	             P2
func (p *Path) Arc(cx, cy, r, startDeg, sweepDeg float64) *Path {
	if r <= 0 || sweepDeg == 0 {
		return p
	}
	// A sweep past a full turn draws the circle once; more would only
	// overdraw it, and an unbounded segment count is a way to stall a render.
	sweepDeg = math.Max(-360, math.Min(360, sweepDeg))

	a0 := startDeg * math.Pi / 180
	x0, y0 := cx+r*math.Cos(a0), cy+r*math.Sin(a0)
	if p.open {
		p.LineTo(x0, y0)
	} else {
		p.MoveTo(x0, y0)
	}

	segments := int(math.Ceil(math.Abs(sweepDeg)/90 - 1e-9))
	step := sweepDeg / float64(segments) * math.Pi / 180
	k := 4.0 / 3.0 * math.Tan(step/4)
	for i := range segments {
		t0 := a0 + float64(i)*step
		t1 := t0 + step
		c0, s0 := math.Cos(t0), math.Sin(t0)
		c1, s1 := math.Cos(t1), math.Sin(t1)
		// The tangent of (cos t, sin t) is (−sin t, cos t); k carries the
		// sweep's sign, so an anticlockwise segment's controls point back
		// along it.
		p.CubicTo(
			cx+r*(c0-k*s0), cy+r*(s0+k*c0),
			cx+r*(c1+k*s1), cy+r*(s1-k*c1),
			cx+r*c1, cy+r*s1,
		)
	}
	return p
}

// Close joins the current point back to the start of the subpath. A later
// LineTo continues from that start, as it does on every platform.
func (p *Path) Close() *Path {
	if !p.open {
		return p
	}
	p.ops = append(p.ops, PathClose)
	p.cx, p.cy = p.sx, p.sy
	return p
}

// wire returns a copy of the ops with coordinates rounded to places decimal
// places. Opcodes are small integers and round to themselves.
func (p *Path) wire(places int) []float64 {
	if len(p.ops) == 0 {
		return nil
	}
	scale := math.Pow(10, float64(places))
	out := make([]float64, len(p.ops))
	for i, v := range p.ops {
		out[i] = math.Round(v*scale) / scale
	}
	return out
}

// coordinatePlaces is how many decimal places a coordinate keeps for a
// viewBox whose larger side is extent: enough for 1/10000 of the drawing,
// which is finer than any screen can show and keeps the JSON short —
// 33.33333333333333 becomes 33.33 in a 100-unit box.
func coordinatePlaces(extent float64) int {
	return max(0, 4-int(math.Floor(math.Log10(extent))))
}

// Line is a single straight segment.
func Line(x1, y1, x2, y2 float64) *Path {
	return NewPath().MoveTo(x1, y1).LineTo(x2, y2)
}

// Polyline joins the points (x0, y0, x1, y1, ...) with straight segments. An
// odd trailing coordinate is ignored.
func Polyline(xy ...float64) *Path {
	p := NewPath()
	for i := 0; i+1 < len(xy); i += 2 {
		p.LineTo(xy[i], xy[i+1])
	}
	return p
}

// Rect is a closed rectangle with its top-left corner at (x, y).
func Rect(x, y, w, h float64) *Path {
	return NewPath().MoveTo(x, y).LineTo(x+w, y).LineTo(x+w, y+h).LineTo(x, y+h).Close()
}

// Circle is a closed circle, drawn clockwise from three o'clock.
func Circle(cx, cy, r float64) *Path {
	return NewPath().Arc(cx, cy, r, 0, 360).Close()
}

// Sector is a closed ring segment between radii inner and outer — a pie
// wedge when inner is 0, a donut segment otherwise. Angles are as for
// Path.Arc.
func Sector(cx, cy, inner, outer, startDeg, sweepDeg float64) *Path {
	p := NewPath()
	if inner <= 0 {
		return p.MoveTo(cx, cy).Arc(cx, cy, outer, startDeg, sweepDeg).Close()
	}
	// Out along the outer edge, back along the inner one: the Arc's joining
	// line is the wedge's far side.
	p.Arc(cx, cy, outer, startDeg, sweepDeg)
	p.Arc(cx, cy, inner, startDeg+sweepDeg, -sweepDeg)
	return p.Close()
}
