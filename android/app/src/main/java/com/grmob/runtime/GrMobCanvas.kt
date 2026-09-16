package com.grmob.runtime

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.LinearGradientShader
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.PathEffect
import androidx.compose.ui.graphics.PathFillType
import androidx.compose.ui.graphics.RadialGradientShader
import androidx.compose.ui.graphics.Shader
import androidx.compose.ui.graphics.ShaderBrush
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.TileMode
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

            // core.FillEvenOdd. Set on the path before either paint; a stroke
            // ignores the fill type, so it is safe for both.
            if (props["fillRule"] == "evenodd") path.fillType = PathFillType.EvenOdd

            // Fill first, then the stroke over it: SVG's paint order, and
            // core.Shape's documented one. Go writes "fill" or the gradient
            // keys, never both.
            val gradient = canvasGradientBrush(props, vp)
            if (gradient != null) {
                drawPath(path, gradient)
            } else {
                GrMobStyle.parseColor(props["fill"] as? String)?.let { drawPath(path, it) }
            }

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

/**
 * A core.Gradient fill as a Compose brush, or null when the shape has none
 * (or its keys are malformed, which paints no fill, as the web targets do).
 *
 * # The shader is built in viewBox units and carries the viewport as its matrix
 *
 * core.Gradient's geometry is in viewBox units, mapped onto the box by the
 * same scale as the shapes: SVG's userSpaceOnUse under the canvas's
 * preserveAspectRatio. Mapping only the end points (or the centre and a
 * radius) would be wrong in two cases under CanvasStretch:
 *
 *   radial     one radius cannot say an ellipse; the circle must stretch
 *   diagonal   the bands of a stretched linear gradient stay parallel to the
 *              gradient's *viewBox* normal, which is not the screen normal
 *              once the axes scale unequally
 *
 * So the shader is created exactly as written and given the viewport's
 * scale-then-translate as its local matrix, which transforms the gradient's
 * whole space the way SVG's viewBox transform does. The path itself was
 * already mapped point by point, so shape and paint land in one frame.
 *
 * Compose's Shader is android.graphics.Shader on this platform, which is what
 * makes setLocalMatrix available.
 */
internal fun canvasGradientBrush(props: Map<String, Any?>, vp: CanvasViewport): Brush? {
    val kind = props["gradient"] as? String ?: return null
    val at = (props["gradientAt"] as? List<*>)?.map { (it as? Number)?.toFloat() ?: return null } ?: return null
    val offsets = (props["gradientStops"] as? List<*>)?.map { (it as? Number)?.toFloat() ?: return null } ?: return null
    val colors = (props["gradientColors"] as? List<*>)?.map { GrMobStyle.parseColor(it as? String) ?: return null } ?: return null
    if (offsets.isEmpty() || offsets.size != colors.size) return null
    val matrix = android.graphics.Matrix().apply {
        setScale(vp.scaleX.toFloat(), vp.scaleY.toFloat())
        postTranslate(vp.offsetX.toFloat(), vp.offsetY.toFloat())
    }
    val shader: Shader = when {
        kind == "linear" && at.size == 4 -> LinearGradientShader(
            from = Offset(at[0], at[1]), to = Offset(at[2], at[3]),
            colors = colors, colorStops = offsets, tileMode = TileMode.Clamp,
        )
        kind == "radial" && at.size == 3 && at[2] > 0f -> RadialGradientShader(
            center = Offset(at[0], at[1]), radius = at[2],
            colors = colors, colorStops = offsets, tileMode = TileMode.Clamp,
        )
        else -> return null
    }
    shader.setLocalMatrix(matrix)
    // A fixed shader rather than one sized per draw: its geometry is the
    // viewBox's, not the box's, so the size Compose passes is not an input.
    return object : ShaderBrush() {
        override fun createShader(size: Size): Shader = shader
    }
}
