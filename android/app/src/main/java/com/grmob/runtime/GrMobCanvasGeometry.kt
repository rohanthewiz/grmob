package com.grmob.runtime

/*
 * The arithmetic half of core.Canvas on Android: how a viewBox maps onto the
 * canvas's measured box, and how core's flat path opcodes decode into drawing
 * calls. Everything that can be wrong without a pixel being drawn lives here.
 *
 * This file imports nothing, like GrMobSelectMenu.kt and GrMobProgress.kt, so
 * android/verify can compile it with the bare Kotlin compiler and run it on a
 * JVM against cases Go generates. GrMobCanvas.kt is the Compose half, which
 * only forwards these calls to androidx.compose.ui.graphics.Path.
 */

/**
 * A viewBox-to-box mapping: a point (x, y) in viewBox units lands at
 * (x · scaleX + offsetX, y · scaleY + offsetY) in pixels.
 */
data class CanvasViewport(
    val scaleX: Double,
    val scaleY: Double,
    val offsetX: Double,
    val offsetY: Double,
)

/**
 * The mapping for a vw × vh viewBox drawn into a width × height box.
 *
 *   stretch  each axis on its own: SVG preserveAspectRatio="none"
 *   fit      one factor, the smaller, with the slack split evenly on the axis
 *            that has it: SVG "xMidYMid meet"
 *
 * A non-positive viewBox side is treated as 100, which is what core.Canvas
 * writes for one, so a hand-assembled node cannot divide by zero.
 */
fun canvasViewport(vw: Double, vh: Double, width: Double, height: Double, stretch: Boolean): CanvasViewport {
    val w = if (vw > 0) vw else 100.0
    val h = if (vh > 0) vh else 100.0
    val sx = width / w
    val sy = height / h
    if (stretch) return CanvasViewport(sx, sy, 0.0, 0.0)
    val k = minOf(sx, sy)
    return CanvasViewport(k, k, (width - w * k) / 2, (height - h * k) / 2)
}

/**
 * This mapping reflected about the vertical centre line of a width-wide box:
 * a point that landed at x · scaleX + offsetX lands at width minus that. It is
 * core.MirrorCanvasMapping, restated, and applies to a core.CanvasMirrorsRTL
 * canvas laid out right-to-left. y is untouched, and so is a stroke's width
 * (a reflection has scale magnitude 1).
 *
 * Mirroring the mapping rather than the DrawScope (a scale(-1, 1)) keeps it
 * in this file, where android/verify can hold it to Go's table, and keeps a
 * gradient's shader in step: canvasGradientBrush builds its matrix from the
 * same viewport.
 */
fun CanvasViewport.mirrored(width: Double): CanvasViewport =
    copy(scaleX = -scaleX, offsetX = width - offsetX)

/** The four drawing calls every platform's path API has. */
interface CanvasPathSink {
    fun moveTo(x: Double, y: Double)
    fun lineTo(x: Double, y: Double)
    fun cubicTo(x1: Double, y1: Double, x2: Double, y2: Double, x: Double, y: Double)
    fun close()
}

/**
 * Replays core's path opcodes (0 move, 1 line, 2 cubic, 3 close — core.PathMove
 * and its siblings) into sink, mapped through vp. A truncated or unknown
 * operation ends the path there, as htmlout's PathData does: one bad shape must
 * not fail the drawing.
 *
 * ops is whatever JSON decoding produced, so its elements are Numbers of any
 * width; anything else ends the path too.
 */
fun decodeCanvasPath(ops: List<*>, vp: CanvasViewport, sink: CanvasPathSink) {
    fun x(v: Any?) = (v as Number).toDouble() * vp.scaleX + vp.offsetX
    fun y(v: Any?) = (v as Number).toDouble() * vp.scaleY + vp.offsetY
    var i = 0
    while (i < ops.size) {
        val op = (ops[i] as? Number)?.toInt() ?: return
        val operands = when (op) {
            0, 1 -> 2
            2 -> 6
            3 -> 0
            else -> return
        }
        if (i + operands >= ops.size && operands > 0) return
        for (j in 1..operands) if (ops[i + j] !is Number) return
        when (op) {
            0 -> sink.moveTo(x(ops[i + 1]), y(ops[i + 2]))
            1 -> sink.lineTo(x(ops[i + 1]), y(ops[i + 2]))
            2 -> sink.cubicTo(
                x(ops[i + 1]), y(ops[i + 2]),
                x(ops[i + 3]), y(ops[i + 4]),
                x(ops[i + 5]), y(ops[i + 6]),
            )
            3 -> sink.close()
        }
        i += 1 + operands
    }
}
