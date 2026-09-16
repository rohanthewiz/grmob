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
