package com.grmob.runtime

import androidx.compose.animation.animateContentSize
import androidx.compose.animation.core.CubicBezierEasing
import androidx.compose.animation.core.Easing
import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.SemanticsPropertyReceiver
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.disabled
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.role
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import org.json.JSONObject

/**
 * Kotlin mirror of Go's core.Style, decoded from the tree/patch JSON.
 *
 * Field names in the JSON are the Go struct's exported names verbatim
 * ("FontSize", "TextColor", ...) because core.Style carries no json tags.
 * Only the subset the Go DSL can actually produce today is mapped; the
 * remaining web-oriented fields (ZIndex, Animation, pseudo states) have no
 * Compose analog at this layer and are intentionally ignored rather than
 * half-implemented. Transition IS mapped: Go declares it, Compose drives
 * the frames (see transitionMs/transitionEasing and boxModifier below).
 *
 * Position is mapped for exactly one of its four values. `relative`,
 * `absolute` and `fixed` still have no analog here — Compose has no
 * out-of-flow placement at this layer — but `sticky` does: a pinned
 * LazyColumn header is what CSS sticky positioning means inside a scrolling
 * list, and core.StickyHeader is the Go spelling that produces it. The field
 * is carried verbatim and read by GrMobList alone.
 */
data class GrMobStyle(
    val fontSize: Float,
    val fontWeight: Int,
    val textColor: Color?,
    val background: Color?,
    val padding: Edges,
    val margin: Edges,
    val borderRadius: Float,
    val shadow: Float,
    val align: String,
    val display: String,
    val width: String,
    val height: String,
    val borderColor: Color?,
    val borderWidth: Float,
    val gap: Float,
    /**
     * core.RowGap / core.ColumnGap: the per-axis spacings. CSS `gap` IS
     * `row-gap` plus `column-gap`, so these are not extra properties beside
     * Gap but the two halves of it, and an axis value set explicitly wins
     * over the isotropic one. Read through verticalGap/horizontalGap below
     * rather than directly — a container knows its own axis and should ask
     * for that axis's spacing, not pick between three fields itself.
     */
    val rowGap: Float,
    val columnGap: Float,
    val justifyContent: String,
    val alignItems: String,
    val flexGrow: Float,
    /** core.FlexWrap: "wrap" or "nowrap" (empty when unset). Read by GrMobRow only. */
    val flexWrap: String,
    /**
     * core.FlexDirection: "row" or "column" (empty when unset). Read by
     * GrMobScroll only — every other container here has its axis fixed by
     * construction (a Compose Row is a Row), so the field would say nothing
     * they do not already know. A Scroll is the one type that has both
     * spellings, and core.Horizontal() is how Go asks for the sideways one.
     */
    val flexDirection: String,
    /**
     * core.Style.Position, carried for its "sticky" value alone; see the
     * class comment. Read by GrMobList only.
     */
    val position: String,
    val lineHeight: Int,
    val accessibilityLabel: String,
    val accessibilityHint: String,
    val accessibilityHidden: Boolean,
    /** Go's core.Role, verbatim; mapped by grMobRole below. */
    val accessibilityRole: String,
    /** Platform disabled state; see Go's core.Style.Disabled. */
    val disabled: Boolean,
    /** Parsed Transition duration; 0 means "no transition, snap changes". */
    val transitionMs: Int,
    val transitionEasing: Easing,
) {
    data class Edges(val top: Int, val right: Int, val bottom: Int, val left: Int)

    /** This node's property-change animation spec (callers gate on transitionMs > 0). */
    fun <T> transitionTween() = tween<T>(transitionMs, easing = transitionEasing)

    /**
     * The spacing between items stacked along one axis, resolving the CSS
     * shorthand the way a browser does: the axis longhand when it is set,
     * the isotropic Gap otherwise.
     *
     *   Column / List / Scroll  ── stack vertically ──▶ verticalGap   (RowGap)
     *   Row                     ── stack horizontally ▶ horizontalGap (ColumnGap)
     *   FlowRow (wrapping Row)  ── both: items along horizontalGap,
     *                              wrapped lines apart by verticalGap
     *
     * Named for the axis they space along rather than for the CSS property
     * they come from, because `row-gap` spaces items *vertically* (it is the
     * gap between rows) and reading the field name as the direction is the
     * mistake this pair exists to make impossible.
     */
    val verticalGap: Float get() = if (rowGap != 0f) rowGap else gap

    val horizontalGap: Float get() = if (columnGap != 0f) columnGap else gap

    companion object {
        fun parse(obj: JSONObject?): GrMobStyle? {
            if (obj == null) return null
            return GrMobStyle(
                fontSize = obj.optDouble("FontSize", 0.0).toFloat(),
                fontWeight = obj.optInt("FontWeight", 0),
                textColor = parseColor(obj.optString("TextColor")),
                background = parseColor(obj.optString("Background")),
                padding = parseEdges(obj.optJSONObject("Padding")),
                margin = parseEdges(obj.optJSONObject("Margin")),
                borderRadius = obj.optDouble("BorderRadius", 0.0).toFloat(),
                shadow = obj.optDouble("Shadow", 0.0).toFloat(),
                align = obj.optString("Align"),
                display = obj.optString("Display"),
                width = obj.optString("Width"),
                height = obj.optString("Height"),
                borderColor = parseColor(obj.optString("BorderColor")),
                borderWidth = obj.optDouble("BorderWidth", 0.0).toFloat(),
                gap = obj.optDouble("Gap", 0.0).toFloat(),
                rowGap = obj.optDouble("RowGap", 0.0).toFloat(),
                columnGap = obj.optDouble("ColumnGap", 0.0).toFloat(),
                justifyContent = obj.optString("JustifyContent"),
                alignItems = obj.optString("AlignItems"),
                flexGrow = obj.optDouble("FlexGrow", 0.0).toFloat(),
                flexWrap = obj.optString("FlexWrap"),
                flexDirection = obj.optString("FlexDirection"),
                position = obj.optString("Position"),
                lineHeight = obj.optInt("LineHeight", 0),
                accessibilityLabel = obj.optString("AccessibilityLabel"),
                accessibilityHint = obj.optString("AccessibilityHint"),
                accessibilityHidden = obj.optBoolean("AccessibilityHidden", false),
                accessibilityRole = obj.optString("AccessibilityRole"),
                disabled = obj.optBoolean("Disabled", false),
                transitionMs = parseTransitionMs(obj.optString("Transition")),
                transitionEasing = parseTransitionEasing(obj.optString("Transition")),
            )
        }

        /**
         * Transition parsing. The canonical Go form is "<ms>ms <easing>"
         * (core.Transition); the CSS longhand ("all 0.3s ease") is tolerated
         * for hand-written styles — the property token is simply skipped.
         */
        private fun parseTransitionMs(value: String): Int {
            for (token in value.split(' ')) {
                if (token.endsWith("ms")) {
                    return token.dropLast(2).toIntOrNull() ?: 0
                }
                if (token.endsWith("s")) {
                    val seconds = token.dropLast(1).toFloatOrNull() ?: continue
                    return (seconds * 1000).toInt()
                }
            }
            return 0
        }

        /**
         * CSS easing keyword → Compose curve, using the cubic-bezier control
         * points the CSS spec defines for each keyword, so Go's declaration
         * animates with the same curve on every platform. Default is "ease",
         * matching core.Transition's default.
         */
        private fun parseTransitionEasing(value: String): Easing =
            when (value.split(' ').lastOrNull { it in easingNames }) {
                "linear" -> LinearEasing
                "ease-in" -> CubicBezierEasing(0.42f, 0f, 1f, 1f)
                "ease-out" -> CubicBezierEasing(0f, 0f, 0.58f, 1f)
                "ease-in-out" -> CubicBezierEasing(0.42f, 0f, 0.58f, 1f)
                else -> CubicBezierEasing(0.25f, 0.1f, 0.25f, 1f) // "ease"
            }

        private val easingNames =
            setOf("linear", "ease", "ease-in", "ease-out", "ease-in-out")

        /**
         * Go's EdgeInsets carries per-side values plus Horizontal/Vertical
         * shorthands; the shorthand fills any side not set explicitly, which
         * matches how the DSL's PaddingHorizontal-style helpers are used.
         */
        private fun parseEdges(obj: JSONObject?): Edges {
            if (obj == null) return Edges(0, 0, 0, 0)
            val h = obj.optInt("Horizontal", 0)
            val v = obj.optInt("Vertical", 0)
            fun side(name: String, shorthand: Int): Int {
                val explicit = obj.optInt(name, 0)
                return if (explicit != 0) explicit else shorthand
            }
            return Edges(
                top = side("Top", v),
                right = side("Right", h),
                bottom = side("Bottom", v),
                left = side("Left", h),
            )
        }

        /** Accepts CSS-style #RGB, #RRGGBB, and #RRGGBBAA (Go emits the latter two). */
        fun parseColor(hex: String?): Color? {
            if (hex.isNullOrEmpty() || !hex.startsWith("#")) return null
            val s = hex.substring(1)
            return try {
                when (s.length) {
                    3 -> {
                        val r = s[0].digitToInt(16) * 17
                        val g = s[1].digitToInt(16) * 17
                        val b = s[2].digitToInt(16) * 17
                        Color(r, g, b)
                    }
                    6 -> {
                        val v = s.toLong(16)
                        Color(0xFF000000L or v)
                    }
                    // CSS orders the alpha byte last; android.graphics wants it
                    // first, so recompose the channels rather than parse directly.
                    8 -> {
                        val v = s.toLong(16)
                        val rgb = v ushr 8
                        val a = v and 0xFF
                        Color((a shl 24) or rgb)
                    }
                    else -> null
                }
            } catch (_: NumberFormatException) {
                null
            }
        }
    }
}

/**
 * Builds this style's box modifiers in CSS box-model order, outermost first:
 * margin, size, elevation shadow, corner clip, background, border, then inner
 * padding. The order is load-bearing — e.g. padding before background would
 * paint the background inside the padding, and clip after background would
 * leave square corners painted.
 *
 * `extra` is a scope-dependent modifier the parent computed for this child
 * (today: Row/Column weight from FlexGrow, which only exists as a RowScope/
 * ColumnScope extension and so cannot be built here).
 *
 * `gestures` is the node's tap/long-press modifier (see Renderer.kt's
 * gestureModifier). It is a parameter rather than part of `extra` so it can
 * be inserted at the right box layer: after background/border and before
 * padding, making the whole visible box — padding included, margin excluded —
 * the touch target, with the ripple clipped to the node's shape.
 */
fun GrMobStyle?.boxModifier(extra: Modifier = Modifier, gestures: Modifier = Modifier): Modifier {
    var m: Modifier = extra
    if (this == null) return m.then(gestures)

    // Accessibility semantics come first so they annotate the element as a
    // whole. Hidden wins: clearAndSetSemantics prunes this node and its
    // subtree from the accessibility tree entirely (decorative content).
    // TalkBack has no separate hint slot, so a hint is folded into the
    // content description after the label.
    // Bound before the semantics lambda: inside it, `disabled` would read as
    // the SemanticsPropertyReceiver's own disabled() marker rather than this
    // style's flag.
    val isDisabled = disabled
    val kind = accessibilityRole
    if (accessibilityHidden) {
        m = m.clearAndSetSemantics { }
    } else if (accessibilityLabel.isNotEmpty() || accessibilityHint.isNotEmpty() ||
        isDisabled || kind.isNotEmpty()
    ) {
        val description = listOf(accessibilityLabel, accessibilityHint)
            .filter { it.isNotEmpty() }.joinToString(". ")
        m = m.semantics {
            if (description.isNotEmpty()) contentDescription = description
            // TalkBack announces the Disabled property itself, so a disabled
            // node needs no ", disabled" folded into its description. The
            // material3 controls set this from their own `enabled` parameter;
            // this branch is for everything else — a tappable Box or Row,
            // whose gesture modifier is dropped in Renderer.kt when disabled
            // and which would otherwise still look activatable to TalkBack.
            if (isDisabled) disabled()
            grMobRole(kind)
        }
    }

    if (margin != Edges0) {
        m = m.padding(
            start = margin.left.dp, top = margin.top.dp,
            end = margin.right.dp, bottom = margin.bottom.dp,
        )
    }
    // Size/layout changes animate when the node declares a Transition.
    // Placed before the dimension modifiers so explicit width/height changes
    // (and content-driven ones from padding or children) all animate; color
    // animation is composition state, handled in Renderer.kt's animatedStyle.
    if (transitionMs > 0) {
        m = m.animateContentSize(transitionTween())
    }
    m = m.then(dimensionModifier(width, horizontal = true))
    m = m.then(dimensionModifier(height, horizontal = false))

    val shape = if (borderRadius > 0f) RoundedCornerShape(borderRadius.dp) else null
    if (shadow > 0f) {
        m = m.shadow(elevation = shadow.dp, shape = shape ?: RoundedCornerShape(0.dp))
    }
    if (shape != null) m = m.clip(shape)
    background?.let { m = m.background(it) }
    if (borderWidth > 0f && borderColor != null) {
        m = m.border(borderWidth.dp, borderColor, shape ?: RoundedCornerShape(0.dp))
    }
    m = m.then(gestures)
    if (padding != Edges0) {
        m = m.padding(
            start = padding.left.dp, top = padding.top.dp,
            end = padding.right.dp, bottom = padding.bottom.dp,
        )
    }
    // "hidden" keeps the node's space but not its pixels ("none" is handled
    // earlier by not composing the node at all — see RenderNode).
    if (display == "hidden") m = m.alpha(0f)
    return m
}

private val Edges0 = GrMobStyle.Edges(0, 0, 0, 0)

/**
 * Maps a Go dimension string onto a size modifier. Supported forms: "120px"
 * or a bare number (density-independent pixels), "100%" / other percentages
 * (fraction of the parent), and ""/"auto" (wrap content, i.e. no modifier).
 */
private fun dimensionModifier(value: String, horizontal: Boolean): Modifier {
    if (value.isEmpty() || value == "auto") return Modifier
    if (value.endsWith("%")) {
        val pct = value.dropLast(1).toFloatOrNull() ?: return Modifier
        val fraction = (pct / 100f).coerceIn(0f, 1f)
        return if (horizontal) Modifier.fillMaxWidth(fraction) else Modifier.fillMaxHeight(fraction)
    }
    val number = value.removeSuffix("px").toFloatOrNull() ?: return Modifier
    return if (horizontal) Modifier.width(number.dp) else Modifier.height(number.dp)
}

/**
 * Maps one core.Role onto Compose semantics, inside the semantics lambda that
 * is already open for the label, the hint and the disabled marker.
 *
 * Five of the sixteen roles land on something here; the other eleven are named
 * anyway. Compose has no landmark vocabulary at all — TalkBack navigates by
 * heading, not by banner — and its tabular semantics are collectionInfo, which
 * describes counts and indices this prop does not carry, so a `role="table"`
 * has nothing to be mapped onto that would not be a lie about the shape of the
 * data. Listing them is what keeps that a decision rather than an omission:
 * an `else ->` that swallowed them would look exactly the same as a role
 * nobody had heard of, which is the failure this file's ContentScale mapping
 * already learned about the hard way.
 *
 * The parameter is `kind` and not `role` because `role` inside a
 * SemanticsPropertyReceiver is the semantics property being assigned two lines
 * down; a parameter of that name would shadow it and the assignment would stop
 * compiling.
 *
 * One arm per line, string literals first, `else ->` last: mobile/verify's
 * TestKotlinRoleCoversEveryRole reads these arms out of the source and holds
 * them against core.Roles().
 *
 * # AccessibilityHeadingLevel is not read here, and cannot be
 *
 * Go's core.Style carries a heading tier beside the role — 1 for a screen's
 * name, 2 for a section inside it — which the two web targets emit as
 * aria-level and SwiftUI applies through accessibilityHeading. Compose's
 * `heading()` takes no argument and the semantics package has no level
 * property, so there is nothing here for the field to become and this renderer
 * deliberately does not parse the JSON key.
 *
 * Written down for the same reason the eleven unmapped roles are written down:
 * a field this file simply ignored would be indistinguishable from one nobody
 * had heard of, and the next person to look would have to re-derive that
 * Compose cannot say it. mobile/verify/heading_level_test.go pins this
 * paragraph so the note cannot quietly outlive the limitation.
 */
fun SemanticsPropertyReceiver.grMobRole(kind: String) {
    when (kind) {
        // A column header is a heading over its column; TalkBack has one
        // notion of heading and this is the nearest true thing to say.
        "heading", "columnheader" -> heading()
        "button" -> role = Role.Button
        // The two live regions differ in how rudely they interrupt: polite
        // waits for a pause, assertive cuts in.
        "status" -> liveRegion = LiveRegionMode.Polite
        "alert" -> liveRegion = LiveRegionMode.Assertive
        // No Compose analog. See the note above on why they are spelled out.
        "table", "rowgroup", "row", "cell" -> {}
        "list", "listitem" -> {}
        "banner", "navigation", "search", "toolbar" -> {}
        // Compose's Role has Button, Checkbox, Switch, RadioButton, Tab,
        // Image and DropdownList, and no Link — the one place SwiftUI's
        // vocabulary is the richer of the two.
        "link" -> {}
        else -> {}
    }
}
