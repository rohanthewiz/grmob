package com.grmob.runtime

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberUpdatedState
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
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.drawscope.clipPath
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.TextMeasurer
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.drawText
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.rememberTextMeasurer
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.unit.dp

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
 * No clip of the canvas's own: a stroke centred on the viewBox edge (a chart's
 * baseline at y = vh) draws half outside the box, as it does with the web's
 * overflow:visible. A shape's own core.Shape.Clip is a DrawScope.clipPath
 * around that shape's draws alone.
 *
 * # Text
 *
 * A CanvasText child is measured with the composition's TextMeasurer (which
 * caches, so a redraw of unchanged text does not lay it out again) and drawn
 * at its anchor less the aligned share of its size. Its Size is in layout
 * units, so it is converted dp → sp with the font scale divided back out:
 * a drawing's labels must not grow with the system font size and overflow
 * the drawing, as they do not on the web or iOS.
 *
 * # Taps
 *
 * A canvas with core.Shape.OnClick handlers carries onShapeTap, and a tap on
 * it reports "x,y,w,h" in dp — the point in the box, the box's size — for Go
 * to hit-test (see "Tapping a shape" on core.Canvas). The detector sits
 * inside the node's clickable, so it sees the press first and consumes it:
 * the canvas's own onClick is not also run (Go runs it on a miss), while its
 * TalkBack and keyboard activation, which never pass through here, still
 * reach it. A long press is taken here too, for the same reason, and sent to
 * onLongPress as the clickable would have.
 */
@Composable
internal fun GrMobCanvas(node: GrMobNode, modifier: Modifier) {
    val vw = node.doubleProp("vw").takeIf { it > 0 } ?: 100.0
    val vh = node.doubleProp("vh").takeIf { it > 0 } ?: 100.0
    val stretch = node.stringProp("scale") == "stretch"
    // core.CanvasMirrorsRTL: opt-in, because a clock face or a QR code must
    // never mirror. See core.CanvasMirror.
    val mirror = node.boolProp("mirror")
    val measurer = rememberTextMeasurer()

    var sized = modifier
    if (node.style?.width.isNullOrEmpty()) sized = sized.fillMaxWidth()
    if (node.style?.height.isNullOrEmpty()) sized = sized.aspectRatio((vw / vh).toFloat())

    val onShapeTap = node.stringProp("onShapeTap")
    if (onShapeTap.isNotEmpty()) {
        val runtime = LocalGrMobRuntime.current
        // The tap needs the direction outside a DrawScope; the drawing reads
        // its own below. Both are this canvas's layout direction.
        val reflected = mirror && LocalLayoutDirection.current == LayoutDirection.Rtl
        // Read at fire time, so a pass that renumbers the callbacks does not
        // restart the detector (which would drop a press in progress).
        val tapId by rememberUpdatedState(onShapeTap)
        val longPressId by rememberUpdatedState(node.stringProp("onLongPress"))
        sized = sized.pointerInput(reflected) {
            detectTapGestures(
                onLongPress = { if (longPressId.isNotEmpty()) runtime.click(longPressId) },
                onTap = { at ->
                    val w = size.width.toFloat()
                    val h = size.height.toFloat()
                    if (w > 0f && h > 0f) {
                        // Reflected back to the drawing's own x, as the
                        // viewport below reflects the drawing.
                        val x = if (reflected) w - at.x else at.x
                        runtime.textChanged(tapId, "${x / density},${at.y / density},${w / density},${h / density}")
                    }
                },
            )
        }
    }

    Canvas(sized) {
        // The DrawScope's own layoutDirection, not a CompositionLocal read
        // outside it: it is the direction this canvas was laid out in, and
        // reading it here keeps a direction change a redraw, not a
        // recomposition. Mirrored in the mapping (CanvasViewport.mirrored)
        // rather than by a scale(-1, 1) on the scope, so android/verify holds
        // it to Go's table and the gradient shaders, built from vp, follow.
        val reflected = mirror && layoutDirection == LayoutDirection.Rtl
        val base = canvasViewport(vw, vh, size.width.toDouble(), size.height.toDouble(), stretch)
        val vp = if (reflected) base.mirrored(size.width.toDouble()) else base
        for (shape in node.children) {
            val props = shape.props
            // core.Shape.Clip, in viewBox units like the path, so mapped
            // through the same viewport. An empty one encloses nothing and
            // hides the shape, as an empty <clipPath> does on the web.
            val clip = (props["clip"] as? List<*>)?.let { canvasPath(it, vp) }
            val draw: DrawScope.() -> Unit = when (shape.type) {
                "CanvasText" -> { { drawCanvasText(props, vp, reflected, measurer) } }
                else -> { { drawCanvasShape(props, vp) } }
            }
            if (clip != null) clipPath(clip) { draw() } else draw()
        }
    }
}

/** core's path opcodes, mapped through vp, as a Compose Path. */
private fun canvasPath(ops: List<*>, vp: CanvasViewport): Path {
    val path = Path()
    decodeCanvasPath(ops, vp, object : CanvasPathSink {
        override fun moveTo(x: Double, y: Double) = path.moveTo(x.toFloat(), y.toFloat())
        override fun lineTo(x: Double, y: Double) = path.lineTo(x.toFloat(), y.toFloat())
        override fun cubicTo(x1: Double, y1: Double, x2: Double, y2: Double, x: Double, y: Double) =
            path.cubicTo(x1.toFloat(), y1.toFloat(), x2.toFloat(), y2.toFloat(), x.toFloat(), y.toFloat())
        override fun close() = path.close()
    })
    return path
}

/**
 * One core.CanvasText: measured as a single line at its layout size, and drawn
 * with the anchor share of its width and height taken off the mapped anchor.
 *
 *   align   start 0 · middle ½ · end 1 of the width, left of the anchor;
 *           reversed (1 − share) in a reflected canvas, where start is the
 *           text's right-hand end (core.CanvasText's "Alignment")
 *   valign  top 0 · middle ½ · bottom 1 of the line's height, above it
 *
 * The line's height is the font's ascent to descent, the same edges the web's
 * text-before-edge and text-after-edge name. No fill paints nothing.
 */
private fun DrawScope.drawCanvasText(
    props: Map<String, Any?>, vp: CanvasViewport, reflected: Boolean,
    measurer: TextMeasurer,
) {
    val color = GrMobStyle.parseColor(props["fill"] as? String) ?: return
    val text = props["text"] as? String ?: return
    val at = (props["at"] as? List<*>)?.mapNotNull { (it as? Number)?.toDouble() }?.takeIf { it.size == 2 } ?: listOf(0.0, 0.0)
    val sizeDp = (props["size"] as? Number)?.toFloat()?.takeIf { it > 0f } ?: 12f
    val layout = measurer.measure(
        AnnotatedString(text),
        style = TextStyle(
            color = color,
            fontSize = sizeDp.dp.toSp(),
            fontWeight = if (props["bold"] == true) FontWeight.Bold else FontWeight.Normal,
        ),
        overflow = TextOverflow.Visible,
        softWrap = false,
        maxLines = 1,
    )
    var share = when (props["align"]) {
        "middle" -> 0.5f
        "end" -> 1f
        else -> 0f
    }
    if (reflected) share = 1f - share
    val drop = when (props["valign"]) {
        "top" -> 0f
        "bottom" -> 1f
        else -> 0.5f
    }
    val x = (at[0] * vp.scaleX + vp.offsetX).toFloat()
    val y = (at[1] * vp.scaleY + vp.offsetY).toFloat()
    drawText(layout, topLeft = Offset(x - share * layout.size.width, y - drop * layout.size.height))
}

/** One CanvasShape: its fill, then its stroke. */
private fun DrawScope.drawCanvasShape(props: Map<String, Any?>, vp: CanvasViewport) {
    val ops = props["d"] as? List<*> ?: return
    val path = canvasPath(ops, vp)

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

    // Go writes "stroke" or the strokeGradient keys, never both. The
    // brush's shader carries the viewport matrix, so it colours the
    // stroke in viewBox space while Stroke's width stays in dp: the
    // same split SVG makes under vector-effect="non-scaling-stroke".
    val strokeBrush = canvasGradientBrush(props, vp, prefix = "stroke")
    val strokeColor = GrMobStyle.parseColor(props["stroke"] as? String)
    if (strokeBrush == null && strokeColor == null) return
    val widthDp = (props["strokeWidth"] as? Number)?.toFloat() ?: 1f
    val dash = (props["dash"] as? List<*>)
        ?.mapNotNull { (it as? Number)?.toFloat()?.times(density) }
        ?.takeIf { it.isNotEmpty() }
    val strokeStyle = Stroke(
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
    )
    if (strokeBrush != null) {
        drawPath(path, strokeBrush, style = strokeStyle)
    } else if (strokeColor != null) {
        drawPath(path, strokeColor, style = strokeStyle)
    }
}

/**
 * A core.Gradient paint as a Compose brush, or null when the shape has none
 * (or its keys are malformed, which paints nothing, as the web targets do).
 *
 * `prefix` picks the paint: "" reads the fill's keys (gradient, gradientAt,
 * ...), "stroke" the stroke's (strokeGradient, strokeGradientAt, ...); see
 * core.GradientKey.
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
internal fun canvasGradientBrush(props: Map<String, Any?>, vp: CanvasViewport, prefix: String = ""): Brush? {
    fun key(name: String) = if (prefix.isEmpty()) name else prefix + name.replaceFirstChar { it.uppercaseChar() }
    val kind = props[key("gradient")] as? String ?: return null
    val at = (props[key("gradientAt")] as? List<*>)?.map { (it as? Number)?.toFloat() ?: return null } ?: return null
    val offsets = (props[key("gradientStops")] as? List<*>)?.map { (it as? Number)?.toFloat() ?: return null } ?: return null
    val colors = (props[key("gradientColors")] as? List<*>)?.map { GrMobStyle.parseColor(it as? String) ?: return null } ?: return null
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
