package core

import (
	"math"
	"strconv"
	"strings"
)

// Per-shape hit-testing for core.Canvas. See "Tapping a shape" on Canvas for
// the rules; this file is how they are computed.
//
// # Everything happens in the box, in layout units
//
// The host reports the tap in the canvas's own box ("x,y,w,h", layout units),
// and each shape's path is mapped *into* that box rather than the tap being
// mapped back into viewBox units. Two reasons:
//
//   - A stroke's width is in layout units (see Canvas), so "is the tap within
//     half a stroke of the path" is only a plain distance in the box. In
//     viewBox units it would be an ellipse under CanvasStretch.
//   - The curve flattening tolerance is then a fraction of a layout unit,
//     the same fineness on a watch-sized canvas as on a tablet-sized one.
//
// Inside-ness of a fill is unchanged by the mapping (a positive scale and an
// offset keep every winding number), so nothing is lost by doing it there too.
//
//	viewBox ops ──CanvasMapping──▶ box polylines ──┬─ winding / parity ─▶ fill hit
//	                                               └─ segment distance ─▶ stroke hit

// canvasHitShape is what the hit-test needs of one tappable shape, read from
// its wire props (canvasHitShapeOf).
type canvasHitShape struct {
	d       []float64 // the path's ops, as drawn
	clip    []float64 // the clip's ops; meaningful only when clipped
	clipped bool
	fill    bool    // a flat fill or a fill gradient is painted
	evenOdd bool    // the fill's rule
	stroke  float64 // the stroke's width in layout units; 0 for no stroke
	onClick func()
}

// canvasHitShapeOf reads a CanvasShape's props. It reads the keys Shape.node
// writes, so the hit-test and the renderers answer "does this shape paint a
// fill" from the same facts.
func canvasHitShapeOf(props map[string]any, onClick func()) canvasHitShape {
	h := canvasHitShape{onClick: onClick}
	h.d, _ = props["d"].([]float64)
	h.clip, h.clipped = props["clip"].([]float64)
	_, flat := props["fill"]
	_, shaded := props["gradient"]
	h.fill = flat || shaded
	h.evenOdd = props["fillRule"] == string(FillEvenOdd)
	_, flatStroke := props["stroke"]
	_, shadedStroke := props[GradientKey("stroke", "gradient")]
	if flatStroke || shadedStroke {
		h.stroke, _ = props["strokeWidth"].(float64)
	}
	return h
}

// parseCanvasTap reads the "x,y,w,h" an onShapeTap carries. Every field must
// be a finite number and the box must have area; anything else is a
// malformed event, which is dropped rather than guessed at.
func parseCanvasTap(s string) (bx, by, bw, bh float64, ok bool) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return 0, 0, 0, 0, false
	}
	var v [4]float64
	for i, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, 0, 0, 0, false
		}
		v[i] = f
	}
	if v[2] <= 0 || v[3] <= 0 {
		return 0, 0, 0, 0, false
	}
	return v[0], v[1], v[2], v[3], true
}

// canvasHit returns the index into shapes of the topmost shape the box point
// (bx, by) hits, or -1. shapes is in paint order, so the search runs
// backwards. vw, vh and scale are the canvas's; bw, bh its box.
func canvasHit(shapes []canvasHitShape, vw, vh float64, scale CanvasScale, bx, by, bw, bh float64) int {
	sx, sy, ox, oy := CanvasMapping(vw, vh, bw, bh, scale)
	toBox := func(x, y float64) (float64, float64) { return x*sx + ox, y*sy + oy }
	for i := len(shapes) - 1; i >= 0; i-- {
		s := shapes[i]
		if s.clipped && !insidePolylines(flattenCanvasPath(s.clip, toBox), bx, by, false) {
			continue
		}
		subs := flattenCanvasPath(s.d, toBox)
		if s.fill && insidePolylines(subs, bx, by, s.evenOdd) {
			return i
		}
		if s.stroke > 0 && nearPolylines(subs, bx, by, s.stroke/2) {
			return i
		}
	}
	return -1
}

// polyline is one flattened subpath in box coordinates: x0, y0, x1, y1, ...
// closed records a Close op, which a stroke draws as a final segment back to
// the start; a fill closes every subpath whether or not it says so.
type polyline struct {
	pts    []float64
	closed bool
}

// hitFlatness is how far, in layout units, a flattened curve may stray from
// the true one. A tenth of a unit is far below a finger and below what any
// screen shows, and costs a handful of segments per curve.
const hitFlatness = 0.1

// flattenCanvasPath replays core's opcodes (the same four every renderer
// decodes) into polylines, mapping each point through toBox. A truncated or
// unknown op ends the path there, as the renderers do.
//
// Each cubic is cut into n equal steps of t, where n comes from the
// standard bound on uniform subdivision: the distance between a cubic and
// its n-segment chord polygon is at most ¾·d/n², d being the larger of the
// two second differences |p0 − 2p1 + p2| and |p1 − 2p2 + p3|. Solving for the
// tolerance gives n = ⌈√(¾·d / hitFlatness)⌉, capped so a degenerate input
// cannot make a tap expensive.
func flattenCanvasPath(ops []float64, toBox func(x, y float64) (float64, float64)) []polyline {
	var out []polyline
	var cur *polyline
	var cx, cy, startX, startY float64
	// begin opens a subpath at (x, y), already in box coordinates.
	begin := func(x, y float64) {
		out = append(out, polyline{pts: []float64{x, y}})
		cur = &out[len(out)-1]
		cx, cy, startX, startY = x, y, x, y
	}
	// ensure gives a segment after a Close somewhere to start: SVG, and every
	// renderer, continue from the closed subpath's start in a new subpath.
	ensure := func() {
		if cur == nil {
			begin(startX, startY)
		}
	}
	for i := 0; i < len(ops); {
		var n int
		switch ops[i] {
		case PathMove, PathLine:
			n = 2
		case PathCubic:
			n = 6
		case PathClose:
			n = 0
		default:
			return out
		}
		if i+n >= len(ops) && n > 0 {
			return out
		}
		switch ops[i] {
		case PathMove:
			x, y := toBox(ops[i+1], ops[i+2])
			begin(x, y)
		case PathLine:
			ensure()
			x, y := toBox(ops[i+1], ops[i+2])
			cur.pts = append(cur.pts, x, y)
			cx, cy = x, y
		case PathCubic:
			ensure()
			x1, y1 := toBox(ops[i+1], ops[i+2])
			x2, y2 := toBox(ops[i+3], ops[i+4])
			x3, y3 := toBox(ops[i+5], ops[i+6])
			d := math.Max(
				math.Hypot(cx-2*x1+x2, cy-2*y1+y2),
				math.Hypot(x1-2*x2+x3, y1-2*y2+y3),
			)
			steps := int(math.Ceil(math.Sqrt(0.75 * d / hitFlatness)))
			steps = max(1, min(steps, 128))
			for k := 1; k <= steps; k++ {
				t := float64(k) / float64(steps)
				u := 1 - t
				a, b, c, e := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
				cur.pts = append(cur.pts,
					a*cx+b*x1+c*x2+e*x3,
					a*cy+b*y1+c*y2+e*y3)
			}
			cx, cy = x3, y3
		case PathClose:
			if cur != nil {
				cur.closed = true
			}
			cur = nil
			cx, cy = startX, startY
		}
		i += 1 + n
	}
	return out
}

// insidePolylines reports whether (px, py) is inside the region the
// subpaths enclose, each one closed implicitly, under the nonzero rule or
// (evenOdd) the even-odd one.
//
// It is the classic crossing count: a ray from the point towards +x, and
// every edge it crosses counted +1 going down the screen and −1 going up.
// The half-open test on y (one end inclusive, the other not) counts a ray
// through a vertex exactly once.
func insidePolylines(subs []polyline, px, py float64, evenOdd bool) bool {
	winding := 0
	for _, s := range subs {
		n := len(s.pts) / 2
		if n < 3 {
			continue // a point or a single segment encloses nothing
		}
		for k := range n {
			x0, y0 := s.pts[2*k], s.pts[2*k+1]
			j := (k + 1) % n
			x1, y1 := s.pts[2*j], s.pts[2*j+1]
			if (y0 <= py) == (y1 <= py) {
				continue // the edge does not straddle the ray
			}
			// Where the edge meets the ray's line; a crossing counts only
			// to the right of the point.
			if x0+(py-y0)*(x1-x0)/(y1-y0) <= px {
				continue
			}
			if y1 > y0 {
				winding++
			} else {
				winding--
			}
		}
	}
	if evenOdd {
		return winding%2 != 0
	}
	return winding != 0
}

// nearPolylines reports whether (px, py) is within r of any drawn segment.
// A closed subpath includes its closing segment; an open one does not. A
// subpath of one point is a dot, hit within r of it, which is what a round
// cap draws for a zero-length stroke.
func nearPolylines(subs []polyline, px, py, r float64) bool {
	r2 := r * r
	for _, s := range subs {
		n := len(s.pts) / 2
		if n == 1 {
			if dx, dy := px-s.pts[0], py-s.pts[1]; dx*dx+dy*dy <= r2 {
				return true
			}
			continue
		}
		last := n - 1
		if s.closed {
			last = n
		}
		for k := range last {
			j := (k + 1) % n
			if segmentDist2(px, py, s.pts[2*k], s.pts[2*k+1], s.pts[2*j], s.pts[2*j+1]) <= r2 {
				return true
			}
		}
	}
	return false
}

// segmentDist2 is the squared distance from (px, py) to the segment from
// (x0, y0) to (x1, y1): the foot of the perpendicular, clamped to the
// segment's ends.
func segmentDist2(px, py, x0, y0, x1, y1 float64) float64 {
	dx, dy := x1-x0, y1-y0
	t := 0.0
	if l2 := dx*dx + dy*dy; l2 > 0 {
		t = math.Max(0, math.Min(1, ((px-x0)*dx+(py-y0)*dy)/l2))
	}
	ex, ey := px-(x0+t*dx), py-(y0+t*dy)
	return ex*ex + ey*ey
}
