// The canvas geometry, checked as behaviour.
//
// A SwiftUI Canvas's drawing closure cannot be run outside a host app, so the
// two decisions it depends on were moved out of it into
// GrMobCanvasGeometry.swift: how a viewBox maps onto the measured box, and how
// core's path opcodes decode into drawing calls. This file runs both against
// internal/canvasfixture, whose answers Go computes with core.CanvasMapping,
// and android/verify runs the Kotlin transliteration against the same table.
// What stays out of reach is the last step — handing each call to a Path and
// the paint to the GraphicsContext.
import Foundation

struct CanvasCallCase: Decodable {
    let op: String
    let args: [Double]
}

struct CanvasCase: Decodable {
    let name: String
    let vw: Double
    let vh: Double
    let boxW: Double
    let boxH: Double
    let stretch: Bool
    let ops: [Double]
    let sx: Double
    let sy: Double
    let ox: Double
    let oy: Double
    let calls: [CanvasCallCase]
}

/// Every difference between Swift's geometry and Go's, one line each. Within
/// 1e-9: both sides multiply in the same order, so anything wider than
/// rounding noise is a real disagreement.
func checkCanvas(_ cases: [CanvasCase]) -> [String] {
    // A generated table can arrive empty, and an empty one passes everything.
    if cases.count < 6 { return ["only \(cases.count) canvas cases were generated"] }
    func close(_ a: Double, _ b: Double) -> Bool { abs(a - b) <= 1e-9 }
    var problems: [String] = []
    for c in cases {
        let vp = GrMobCanvasViewport(vw: c.vw, vh: c.vh, width: c.boxW, height: c.boxH, stretch: c.stretch)
        guard close(vp.scaleX, c.sx), close(vp.scaleY, c.sy),
              close(vp.offsetX, c.ox), close(vp.offsetY, c.oy)
        else {
            problems.append("\(c.name): viewport \(vp), want (\(c.sx), \(c.sy), \(c.ox), \(c.oy))")
            continue
        }
        var got: [CanvasCallCase] = []
        grMobDecodeCanvasPath(c.ops.map { NSNumber(value: $0) }, vp) { call in
            switch call {
            case let .move(x, y): got.append(CanvasCallCase(op: "M", args: [x, y]))
            case let .line(x, y): got.append(CanvasCallCase(op: "L", args: [x, y]))
            case let .cubic(x1, y1, x2, y2, x, y):
                got.append(CanvasCallCase(op: "C", args: [x1, y1, x2, y2, x, y]))
            case .close: got.append(CanvasCallCase(op: "Z", args: []))
            }
        }
        guard got.count == c.calls.count else {
            problems.append("\(c.name): \(got.count) calls, want \(c.calls.count)")
            continue
        }
        for (i, (g, w)) in zip(got, c.calls).enumerated()
        where g.op != w.op || g.args.count != w.args.count || zip(g.args, w.args).contains(where: { !close($0, $1) }) {
            problems.append("\(c.name): call \(i) is \(g.op)\(g.args), want \(w.op)\(w.args)")
        }
    }
    return problems
}
