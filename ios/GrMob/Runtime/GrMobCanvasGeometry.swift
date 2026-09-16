import Foundation

// The arithmetic half of core.Canvas on iOS: how a viewBox maps onto the
// canvas's measured box, and how core's flat path opcodes decode into drawing
// calls. Everything that can be wrong without a pixel being drawn lives here.
//
// Foundation only (for NSNumber, which is what JSONSerialization hands back for
// every number), so ios/verify links it into its harness and runs it against
// cases Go generates (internal/canvasfixture). GrMobCanvas in Renderer.swift is
// the SwiftUI half, which only forwards these calls to a Path. The Kotlin
// transliteration is GrMobCanvasGeometry.kt, held to the same table.

/// A viewBox-to-box mapping: a point (x, y) in viewBox units lands at
/// (x · scaleX + offsetX, y · scaleY + offsetY) in points.
struct GrMobCanvasViewport: Equatable {
    let scaleX: Double
    let scaleY: Double
    let offsetX: Double
    let offsetY: Double

    /// The mapping for a vw × vh viewBox drawn into a width × height box —
    /// core.CanvasMapping, restated.
    ///
    ///   stretch  each axis on its own: SVG preserveAspectRatio="none"
    ///   fit      one factor, the smaller, with the slack split evenly on the
    ///            axis that has it: SVG "xMidYMid meet"
    ///
    /// A non-positive viewBox side reads as 100, which is what core.Canvas
    /// writes for one, so a hand-assembled node cannot divide by zero.
    init(vw: Double, vh: Double, width: Double, height: Double, stretch: Bool) {
        let w = vw > 0 ? vw : 100
        let h = vh > 0 ? vh : 100
        let sx = width / w
        let sy = height / h
        if stretch {
            (scaleX, scaleY, offsetX, offsetY) = (sx, sy, 0, 0)
        } else {
            let k = min(sx, sy)
            (scaleX, scaleY, offsetX, offsetY) = (k, k, (width - w * k) / 2, (height - h * k) / 2)
        }
    }
}

/// One drawing call in box points, as the decoder emits it.
enum GrMobCanvasCall: Equatable {
    case move(Double, Double)
    case line(Double, Double)
    case cubic(Double, Double, Double, Double, Double, Double)
    case close
}

/// Replays core's path opcodes (0 move, 1 line, 2 cubic, 3 close — core.PathMove
/// and its siblings) through the mapping, calling emit once per operation. A
/// truncated or unknown operation, or an operand that is not a number, ends the
/// path there, as htmlout's PathData does: one bad shape must not fail the
/// drawing.
func grMobDecodeCanvasPath(_ ops: [Any], _ vp: GrMobCanvasViewport, _ emit: (GrMobCanvasCall) -> Void) {
    func num(_ v: Any) -> Double? {
        if let n = v as? NSNumber { return n.doubleValue }
        return v as? Double
    }
    var i = 0
    while i < ops.count {
        guard let code = num(ops[i]) else { return }
        let operands: Int
        switch code {
        case 0, 1: operands = 2
        case 2: operands = 6
        case 3: operands = 0
        default: return
        }
        if operands > 0 && i + operands >= ops.count { return }
        var a = [Double]()
        a.reserveCapacity(operands)
        for j in 0..<operands {
            guard let v = num(ops[i + 1 + j]) else { return }
            // Even operands are x, odd are y.
            a.append(j % 2 == 0 ? v * vp.scaleX + vp.offsetX : v * vp.scaleY + vp.offsetY)
        }
        switch code {
        case 0: emit(.move(a[0], a[1]))
        case 1: emit(.line(a[0], a[1]))
        case 2: emit(.cubic(a[0], a[1], a[2], a[3], a[4], a[5]))
        default: emit(.close)
        }
        i += 1 + operands
    }
}
