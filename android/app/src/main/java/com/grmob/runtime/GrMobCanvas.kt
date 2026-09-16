package com.grmob.runtime

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.drawscope.Stroke

/**
 * The Compose half of core.Canvas: a foundation Canvas that draws each
 * CanvasShape child as a Path.
 *
 *     core.Canvas(100, 50, shapes, core.CanvasStretch)
 *       ──▶ Canvas(modifier) { for each child: drawPath(fill); drawPath(stroke) }
 *
 * # The shapes are data, read in the draw phase
 *
 * The CanvasShape children are never composed; this reads their props inside
 * the DrawScope, the way GrMobTextGrid reads its rows and GrMobMapView its
 * markers. GrMobNode.props is snapshot state, and a snapshot read during
 * drawing invalidates only the draw — so an update-props patch that moves one
 * hand of a clock redraws the canvas without recomposing or re-measuring it.
 *
 * # Transformed points, untransformed strokes
 *
 * Every point goes through canvasViewport's mapping before it reaches the
 * Path, and the stroke width is converted from dp only. That is how
 * core.Canvas's rule — strokes are layout units, never scaled — comes out the
 * same here as SVG's vector-effect="non-scaling-stroke": a DrawScope scale()
 * would have scaled the stroke with the geometry, and unequally on the two
 * axes under stretch.
 *
 * # Sizing
 *
 * `modifier` is the node's box chain (boxModifier), which applies an author's
 * Width and Height. The canvas adds core's defaults after it: fill the width
 * when no Width was given, and take the viewBox's aspect ratio when no Height
 * was — the same rule htmlout's canvasChassis states in CSS. Compose resolves
 * constraints outside-in, so a stated Width has already fixed the incoming
 * constraints by the time fillMaxWidth is reached and fillMaxWidth fills
 * exactly that.
 *
 * No clip: a stroke centred on the viewBox edge (a chart's baseline at y = vh)
 * draws half outside the box, as it does with the web's overflow:visible.
 */
@Composable
internal fun GrMobCanvas(node: GrMobNode, modifier: Modifier) {
    val vw = node.doubleProp("vw").takeIf { it > 0 } ?: 100.0
    val vh = node.doubleProp("vh").takeIf { it > 0 } ?: 100.0
    val stretch = node.stringProp("scale") == "stretch"

    var sized = modifier
    if (node.style?.width.isNullOrEmpty()) sized = sized.fillMaxWidth()
    if (node.style?.height.isNullOrEmpty()) sized = sized.aspectRatio((vw / vh).toFloat())

    Canvas(sized) {
        val vp = canvasViewport(vw, vh, size.width.toDouble(), size.height.toDouble(), stretch)
        for (shape in node.children) {
            val props = shape.props
            val ops = props["d"] as? List<*> ?: continue
            val path = Path()
            decodeCanvasPath(ops, vp, object : CanvasPathSink {
                override fun moveTo(x: Double, y: Double) = path.moveTo(x.toFloat(), y.toFloat())
                override fun lineTo(x: Double, y: Double) = path.lineTo(x.toFloat(), y.toFloat())
                override fun cubicTo(x1: Double, y1: Double, x2: Double, y2: Double, x: Double, y: Double) =
                    path.cubicTo(x1.toFloat(), y1.toFloat(), x2.toFloat(), y2.toFloat(), x.toFloat(), y.toFloat())
                override fun close() = path.close()
            })

            // Fill first, then the stroke over it: SVG's paint order, and
            // core.Shape's documented one.
            GrMobStyle.parseColor(props["fill"] as? String)?.let { drawPath(path, it) }

            val stroke = GrMobStyle.parseColor(props["stroke"] as? String) ?: continue
            val widthDp = (props["strokeWidth"] as? Number)?.toFloat() ?: 1f
            val dash = (props["dash"] as? List<*>)
                ?.mapNotNull { (it as? Number)?.toFloat()?.times(density) }
                ?.takeIf { it.isNotEmpty() }
            drawPath(
                path, stroke,
                style = Stroke(
                    width = widthDp * density,
                    cap = when (props["cap"]) {
                        "round" -> StrokeCap.Round
                        "square" -> StrokeCap.Square
                        else -> StrokeCap.Butt
                    },
                    join = when (props["join"]) {
                        "round" -> StrokeJoin.Round
                        "bevel" -> StrokeJoin.Bevel
                        else -> StrokeJoin.Miter
                    },
                    // Android's dash effect wants an even count; SVG repeats an
                    // odd list to make one, so do the same.
                    pathEffect = dash?.let {
                        PathEffect.dashPathEffect((if (it.size % 2 == 1) it + it else it).toFloatArray())
                    },
                ),
            )
        }
    }
}
