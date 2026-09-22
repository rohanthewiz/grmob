// Package canvasfixture is the one table of core.Canvas drawings the native
// renderers' geometry is held to.
//
// A Canvas reaches the web as SVG, and a browser applies the viewBox mapping
// and reads the path letters itself. The natives have no SVG: each maps the
// viewBox onto its measured box and replays core's path opcodes into its own
// path API, which is two pieces of arithmetic written twice more, in Kotlin
// (GrMobCanvasGeometry.kt) and Swift (GrMobCanvasGeometry.swift).
//
//	android/verify  runs canvasViewport + decodeCanvasPath on a JVM
//	ios/verify      runs the Swift equivalents
//
// As in valuefixture, cases carry only inputs and Want computes the answer —
// the mapping from core.CanvasMapping, the decoding from Decode below, which
// states the opcode contract core.PathMove and its siblings document.
//
// It lives under internal/ because it is fixture data for this repository's
// own harnesses, not part of the framework's API.
package canvasfixture

import "github.com/rohanthewiz/grmob/core"

// Case is one drawing: a viewBox, the box it is drawn into, a scale, and one
// path's wire opcodes.
type Case struct {
	Name    string
	VW, VH  float64
	BoxW    float64
	BoxH    float64
	Stretch bool
	Ops     []float64
	// Mirror is a CanvasMirrorsRTL canvas laid out right-to-left: the
	// mapping goes through core.MirrorCanvasMapping.
	Mirror bool
}

// Call is one drawing call a renderer makes, in box pixels: Op is "M", "L", "C"
// or "Z", and Args its operands.
type Call struct {
	Op   string
	Args []float64
}

// Want is what a renderer must produce for a case: the mapping and the calls.
type Want struct {
	SX, SY, OX, OY float64
	Calls          []Call
}

// Cases is the table. Each names the decision it witnesses.
func Cases() []Case {
	circle := core.Circle(50, 25, 20)
	return []Case{
		// Fit with slack on the vertical axis: the offset is the half that is
		// easy to get backwards.
		{"fit, wide viewBox in a square box", 100, 50, 300, 300, false, wire(core.Line(0, 0, 100, 50)), false},
		// And on the horizontal axis.
		{"fit, tall viewBox in a wide box", 10, 40, 200, 100, false, wire(core.Rect(0, 0, 10, 40)), false},
		// Stretch: two scales, no offset.
		{"stretch", 100, 50, 300, 300, true, wire(core.Polyline(0, 50, 50, 0, 100, 25)), false},
		// Every opcode, cubics included, through a non-trivial mapping.
		{"a circle, every opcode", 100, 50, 320, 120, false, wire(circle), false},
		// A hand-assembled node's zero viewBox reads as 100, not a division by zero.
		{"zero viewBox reads as 100", 0, 0, 50, 50, false, []float64{0, 100, 100}, false},
		// Truncation and an unknown opcode end the path; neither fails it.
		{"truncated operation", 100, 100, 100, 100, true, []float64{0, 1, 2, 1, 3}, false},
		{"unknown opcode", 100, 100, 100, 100, true, []float64{0, 1, 2, 9, 1, 3, 4}, false},
		// A mirrored stretch: the chart case. The first point lands at the
		// right edge and the last at the left.
		{"mirrored stretch", 100, 50, 300, 120, true, wire(core.Polyline(0, 50, 50, 0, 100, 25)), true},
		// A mirrored fit with horizontal slack: the offset is reflected too, so
		// the drawing stays centred rather than sliding to one side.
		{"mirrored fit, tall viewBox in a wide box", 10, 40, 200, 100, false, wire(core.Rect(0, 0, 4, 40)), true},
		// Every opcode, cubics included, reflected.
		{"a circle, mirrored", 100, 50, 320, 120, false, wire(core.Circle(30, 25, 20)), true},
	}
}

// wire is a path's opcodes as a Canvas sends them.
func wire(p *core.Path) []float64 {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := core.Canvas(100, 100, []core.Shape{{Path: p}}).Render(ctx)
	return n.Children[0].Props["d"].([]float64)
}

// WantFor computes the answer for c.
func WantFor(c Case) Want {
	scale := core.CanvasFit
	if c.Stretch {
		scale = core.CanvasStretch
	}
	sx, sy, ox, oy := core.CanvasMapping(c.VW, c.VH, c.BoxW, c.BoxH, scale)
	if c.Mirror {
		sx, ox = core.MirrorCanvasMapping(sx, ox, c.BoxW)
	}
	return Want{SX: sx, SY: sy, OX: ox, OY: oy, Calls: Decode(c.Ops, sx, sy, ox, oy)}
}

// Decode replays opcodes into calls under a mapping, ending at the first
// truncated or unknown operation.
func Decode(ops []float64, sx, sy, ox, oy float64) []Call {
	var out []Call
	for i := 0; i < len(ops); {
		var op string
		var n int
		switch ops[i] {
		case core.PathMove:
			op, n = "M", 2
		case core.PathLine:
			op, n = "L", 2
		case core.PathCubic:
			op, n = "C", 6
		case core.PathClose:
			op, n = "Z", 0
		default:
			return out
		}
		if n > 0 && i+n >= len(ops) {
			return out
		}
		args := make([]float64, n)
		for j := range n {
			if j%2 == 0 {
				args[j] = ops[i+1+j]*sx + ox
			} else {
				args[j] = ops[i+1+j]*sy + oy
			}
		}
		out = append(out, Call{op, args})
		i += 1 + n
	}
	return out
}
