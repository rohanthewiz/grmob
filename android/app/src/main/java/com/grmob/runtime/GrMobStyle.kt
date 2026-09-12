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
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.ProgressBarRangeInfo
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.SemanticsPropertyReceiver
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.disabled
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.progressBarRangeInfo
import androidx.compose.ui.semantics.role
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.stateDescription
import androidx.compose.ui.unit.dp
import org.json.JSONObject

/**
 * Kotlin mirror of Go's core.Style, decoded from the tree/patch JSON.
 *
 * Field names in the JSON are the Go struct's exported names verbatim
 * ("FontSize", "TextColor", ...) because core.Style's json tags set no names
 * — every tag on it is `,omitzero`, which changes what is *present* and never
 * what a present field is called. Every read below is an `opt*` with a zero
 * default, so an omitted field and a field written as zero decode alike; that
 * equivalence is the contract those tags rely on, and it predates them.
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
    /**
     * core.Rotate: clockwise degrees about the node's own centre. A paint
     * transform, not a layout one — Modifier.rotate draws the node turned and
     * reports its unrotated bounds, which is what CSS `transform` and
     * SwiftUI's `.rotationEffect` also do, so the three agree without a
     * mapping table. Passed through unnormalised; see core.Style.Rotate for
     * why the winding is the caller's to choose.
     */
    val rotate: Float,
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
    /**
     * core.Style.FlexShrink, as written — which is NOT the shrink factor.
     *
     * Zero means "unset" here, as it does for every other number in a
     * core.Style, and flex-shrink is the one property whose CSS initial value
     * is not zero. So the Go side spells a factor of zero as core.ShrinkNone
     * (-1) and this field carries that verbatim; `shrinkFactor` below is the
     * reading. Storing the raw number rather than the reading keeps this class
     * a decode of the JSON and puts the one rule in one place — the same
     * arrangement GrMobStyle.swift makes, and wasm/verify's shrink_test.go
     * pins all four spellings of the sentinel to core.ShrinkNone.
     */
    val flexShrink: Float,
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
    /**
     * Go's core.StackAlignment, verbatim: "top-start", "bottom", ... or "" for
     * a layer that takes the stack's centre. Read by GrMobZStack alone,
     * through grMobStackAlignment in Renderer.kt — it is a *layer* property,
     * and a node that is not a layer of an overlay has no use for it.
     */
    val stackAlign: String,
    val lineHeight: Int,
    val accessibilityLabel: String,
    val accessibilityHint: String,
    val accessibilityHidden: Boolean,
    /** Go's core.Role, verbatim; mapped by grMobRole below. */
    val accessibilityRole: String,
    /**
     * Go's core.SelectedState, verbatim: "true", "false", or "" for a node
     * that makes no claim. Mapped by grMobSelected below.
     */
    val accessibilitySelected: String,
    /**
     * Go's core.ExpandedState, verbatim: "true", "false", or "" for a node
     * that is not a disclosure. Unlike every other accessibility field here it
     * is *not* spent in boxModifier's semantics block — Compose says this with
     * an action rather than a property, and an action needs the node's click
     * callback to perform. Renderer.kt's gestureModifier is where it lands;
     * see grMobDisclosure there.
     */
    val accessibilityExpanded: String,
    /**
     * Go's core.ValueRange, verbatim: where a valued control sits inside its
     * range, as the four strings ARIA spells them with. Mapped by grMobValue
     * below, which is a separate function from grMobRole for the reason the
     * selection has one — the role says a node *is* a progress bar and this
     * says how far along it is, and Compose needs both in one call.
     *
     * The numbers are strings on the wire because a stated 0 and an unstated
     * one are different facts and Go's Style merges on "non-zero wins"; see
     * core.ValueRange. They are parsed here rather than carried as text
     * because ProgressBarRangeInfo takes floats.
     */
    val accessibilityValue: ValueRange,
    /** Platform disabled state; see Go's core.Style.Disabled. */
    val disabled: Boolean,
    /** Parsed Transition duration; 0 means "no transition, snap changes". */
    val transitionMs: Int,
    val transitionEasing: Easing,
) {
    data class Edges(val top: Int, val right: Int, val bottom: Int, val left: Int)

    /**
     * Go's core.ValueRange: where a valued control sits inside its range.
     *
     * The three numbers are nullable Floats rather than a Float with a
     * sentinel, and the reason is the same one that makes them strings on the
     * wire: 0 is a bar at the start of an upload, so "unstated" cannot be
     * spelled as a number. ARIA says an unstated `now` is an *indeterminate*
     * bar — one that is running with no idea how far — which is a real state
     * this platform can describe and must not be given a false 0 instead.
     *
     * `text` is the one member with a mapping outside the range roles: it
     * becomes stateDescription, which TalkBack announces on any node.
     */
    data class ValueRange(
        val now: Float?,
        val min: Float?,
        val max: Float?,
        val text: String,
    ) {
        /** Whether this range says anything at all; Go's ValueRange.Stated. */
        fun stated(): Boolean = now != null || min != null || max != null || text.isNotEmpty()
    }

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

    /**
     * The shrink factor this style asks for: 1 when nothing was set (the CSS
     * initial value), 0 for core.ShrinkNone, and the number otherwise.
     *
     * The mirror of core.Style.ShrinkFactor and of GrMobStyle.swift's
     * shrinkFactor, and the only place in this runtime that knows what -1
     * means.
     *
     * What this renderer can do with it is narrower than what the other three
     * do, and the narrowing is a property of Compose rather than of this
     * mapping. A Compose Row has no proportional shrink: it measures each
     * unweighted child against whatever main-axis space is left and hands the
     * next one the remainder, so there is no per-item factor to scale and a
     * fractional factor has nothing to mean. Zero does: "measure me against
     * my own content and let the row overflow" is expressible, and
     * shrinkPinned is what the children loops read. See core.ShrinkNone.
     */
    val shrinkFactor: Float get() = when (flexShrink) {
        0f -> 1f
        -1f -> 0f
        else -> flexShrink
    }

    /**
     * Whether this node refuses to shrink — core.FlexShrink(0), and the only
     * shrink declaration a Compose Row can honour.
     *
     * Read rather than `shrinkFactor == 0f` at the call sites so that the one
     * comparison against a float lives here, beside the mapping that produces
     * it.
     */
    val shrinkPinned: Boolean get() = shrinkFactor == 0f

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
                rotate = obj.optDouble("Rotate", 0.0).toFloat(),
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
                flexShrink = obj.optDouble("FlexShrink", 0.0).toFloat(),
                flexWrap = obj.optString("FlexWrap"),
                flexDirection = obj.optString("FlexDirection"),
                position = obj.optString("Position"),
                stackAlign = obj.optString("StackAlign"),
                lineHeight = obj.optInt("LineHeight", 0),
                accessibilityLabel = obj.optString("AccessibilityLabel"),
                accessibilityHint = obj.optString("AccessibilityHint"),
                accessibilityHidden = obj.optBoolean("AccessibilityHidden", false),
                accessibilityRole = obj.optString("AccessibilityRole"),
                accessibilitySelected = obj.optString("AccessibilitySelected"),
                accessibilityExpanded = obj.optString("AccessibilityExpanded"),
                accessibilityValue = parseValueRange(obj.optJSONObject("AccessibilityValue")),
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
         * Go's core.ValueRange. An absent object and an object of empty
         * strings both mean "not a valued control" — the first is a Style that
         * never set one, the second is one whose range went back to its zero
         * value, and Go emits whichever the encoder happens to produce.
         *
         * Each number is parsed independently and independently nullable,
         * because ARIA lets a bar state its position without its bounds (they
         * default to 0 and 100) and lets it state bounds without a position
         * (an indeterminate bar inside a known range). Null rather than a 0f
         * default for the same reason the field is nullable: a value that
         * failed to parse must not read as a bar at the start.
         *
         * The parse is [grMobProgressNumber] rather than a bare
         * `toFloatOrNull` so that one rule decides what counts as a number —
         * it also refuses the non-finite spellings Kotlin's parser accepts,
         * which no range property can hold — and so that the rule is one an
         * off-device harness can run.
         */
        private fun parseValueRange(obj: JSONObject?): ValueRange {
            if (obj == null) return ValueRange(null, null, null, "")
            fun num(name: String): Float? = grMobProgressNumber(obj.optString(name))
            return ValueRange(
                now = num("Now"),
                min = num("Min"),
                max = num("Max"),
                text = obj.optString("Text"),
            )
        }

        /**
         * Go's EdgeInsets carries per-side values plus Horizontal/Vertical
         * shorthands; the shorthand fills any side not set explicitly, which
         * matches how the DSL's PaddingHorizontal-style helpers are used.
         *
         * Every one of the six is `json:",omitzero"` on the Go side, so a key
         * that is absent here is a field that was zero there. Nothing below
         * needs to change for that, and the reason is the `explicit != 0`
         * test: this function has always defined a zero side as "unset, take
         * the axis", so "absent" and "present and zero" were already the same
         * question. The defaults on optInt are what make them the same answer
         * — do not replace them with a has()/optInt pair thinking it recovers
         * a distinction, because Go no longer sends one.
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
    // Bound out here for the same reason `kind` is: inside the lambda,
    // `selected` is the SemanticsPropertyReceiver's own property being
    // assigned rather than this style's field.
    val selectedState = accessibilitySelected
    // And again: inside the lambda `value` would be nothing in particular, but
    // the range has to be read off `this` before the receiver changes.
    val valueRange = accessibilityValue
    if (accessibilityHidden) {
        m = m.clearAndSetSemantics { }
    } else if (accessibilityLabel.isNotEmpty() || accessibilityHint.isNotEmpty() ||
        isDisabled || kind.isNotEmpty() || selectedState.isNotEmpty() ||
        valueRange.stated()
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
            grMobSelected(selectedState)
            grMobValue(valueRange)
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

    // Rotation goes here — after margin and the dimension modifiers, before
    // shadow/clip/background/border — and the position is load-bearing in a
    // way the CSS property it mirrors is not.
    //
    // Modifier.rotate is a graphics layer, and a Compose layer turns only what
    // is drawn *after* it in the chain. Placed below the background it would
    // spin the content inside a square that stayed put; placed here it turns
    // the whole painted box — shadow, corner clip, fill and border together —
    // which is what `transform: rotate()` does to a border box on the web.
    //
    // Margin stays outside it, again matching CSS: the reserved space around
    // the node is not part of the transformed box, and including it would
    // swing an asymmetrically-spaced node about a centre that is not its own.
    //
    // The gesture modifier is added further down and so sits inside the
    // layer, which is what makes the touch target turn with the pixels —
    // Compose transforms pointer coordinates through the layer, so a tap on
    // the visible corner of a turned box lands on the box.
    //
    // Guarded rather than applied unconditionally: unlike the WASM runtime,
    // which reuses a live DOM element and must clear a stale declaration, this
    // builds a fresh chain every recomposition, so a zero angle omits a layer
    // instead of adding an identity one.
    if (rotate != 0f) m = m.rotate(rotate)

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
 * Eight of the twenty-five roles land on something here; the other seventeen are named
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
 *
 * # AccessibilityNestingLevel is not read here either
 *
 * The other half of ARIA's aria-level: how deep a listitem or a row sits
 * inside a nested collection. Compose's nearest property is
 * `collectionItemInfo`, which states an item's index and span *within one
 * collection* rather than its depth *within nested ones* — a different claim,
 * and filling it from this field would tell TalkBack something the app never
 * said. SwiftUI has nothing for it at all, so unlike the heading tier this one
 * is inert on both natives and lives only on the web.
 *
 * Same treatment as above: the key is deliberately not parsed, and
 * mobile/verify/nesting_level_test.go pins both halves.
 */
/**
 * AccessibilityID and AccessibilityControls are not read here either.
 *
 * Go's two IDREF props state that one element points at another — a hand-built
 * tab and the region it shows. Compose semantics has no such relationship, and
 * TalkBack navigates by swiping to the next node rather than by following a
 * reference, so there is nothing for the pair to become.
 *
 * The near miss is `testTag`, and it is the same trap `accessibilityIdentifier`
 * is on the other platform: a test selector, not an accessibility property, and
 * invisible to TalkBack unless an app opts into testTagsAsResourceId. Filling
 * it from an ARIA wiring string would make every hand-built tab a test handle
 * and announce nothing. Same treatment as the two levels above: the key is
 * deliberately not parsed, and mobile/verify/idref_test.go pins both halves.
 *
 * AccessibilitySelectionFollowsFocus is not read here either.
 *
 * Go's flag says a composite widget should choose the member its arrow keys
 * land on. There are no arrow keys here: TalkBack crosses a collection by
 * swipe, and a swipe moves the reader's cursor rather than focus, so the
 * sequence the flag describes does not occur on this platform. It is not a
 * behaviour Compose spells differently — it is one that has no occasion.
 *
 * The near miss is `focusable()` plus a FocusRequester, or acting on
 * onFocusChanged. Both would be wrong the same way: they are about which node
 * has the attention, where this is about what the widget does once it has
 * moved. Taking either would fire the app's onTap on every swipe past a row —
 * a selection nobody asked for, on the platform where the user is least able
 * to see it coming. mobile/verify/followsfocus_test.go pins both halves.
 */
fun SemanticsPropertyReceiver.grMobRole(kind: String) {
    when (kind) {
        // A column header is a heading over its column; TalkBack has one
        // notion of heading and this is the nearest true thing to say.
        "heading", "columnheader" -> heading()
        "button" -> role = Role.Button
        // A node standing in for a picture. Compose has the role, and pairing
        // it with the contentDescription the caller supplied is what makes
        // TalkBack announce "image" and then the alternative rather than
        // reading whatever text happens to be inside.
        "img" -> role = Role.Image
        // One control in a tab strip. Compose has the control half of the tab
        // pair and nothing for the strip around it; SwiftUI has exactly the
        // opposite (.isTabBar and no trait for a tab), which is why core.Role
        // carries both and neither platform could have supplied the pair. The
        // *state* — which tab is showing — is grMobSelected's, below.
        "tab" -> role = Role.Tab
        // The strip. No Compose analog: a Role is a property of a control,
        // and there is no container semantics for "these are tabs".
        "tablist" -> {}
        // The region a tab shows. No analog either, and the loss is smaller
        // than on the web: a tabpanel's whole job there is to be the far end
        // of an aria-controls, and TalkBack navigates by swiping to the next
        // element rather than by following a reference — which is the same
        // reason core.Style.AccessibilityControls is deliberately unparsed
        // here. A core.TabView still announces correctly on this platform
        // because it hands the whole strip to a Material TabRow.
        "tabpanel" -> {}
        // The three live regions. The first two differ in how rudely they
        // interrupt: polite waits for a pause, assertive cuts in.
        "status" -> liveRegion = LiveRegionMode.Polite
        // A log is polite too, and on this platform that is the whole of what
        // can be said about it — the difference from "status" is that a log is
        // appended to and read back in order rather than replaced, which
        // TalkBack has no way to be told. Deliberately the same call as the
        // arm above rather than a gap: collapsing two ARIA roles onto one
        // Compose primitive is the honest mapping, where dropping it would
        // silence a chat transcript entirely.
        "log" -> liveRegion = LiveRegionMode.Polite
        "alert" -> liveRegion = LiveRegionMode.Assertive
        // No Compose analog. See the note above on why they are spelled out.
        "table", "rowgroup", "row", "cell" -> {}
        "list", "listitem" -> {}
        // A listbox and one option in it. Compose's Role has no member for
        // either, and the loss is smaller than the empty arm suggests: what a
        // chosen row most needs said is the *state*, and grMobSelected below
        // sets `selected` on any node without asking what contains it.
        // Missing is only the container's word for what the choice is among —
        // which TalkBack, navigating by swipe rather than by arrow key, does
        // not use the way a browser does.
        "listbox", "option" -> {}
        // A determinate or indeterminate progress bar. The *role* has no
        // Compose member — Role has Button, Checkbox, Switch, RadioButton,
        // Tab, Image and DropdownList — but unlike the empty arms around it
        // this platform is not silent about a progress bar: what it says is
        // the range, through progressBarRangeInfo, and grMobValue below is
        // where that lands. A bar with no range says nothing here, which is
        // the honest rendering of ARIA's indeterminate bar on a platform whose
        // only vocabulary for one is a number.
        "progressbar" -> {}
        "banner", "navigation", "search", "toolbar" -> {}
        // Compose's Role has Button, Checkbox, Switch, RadioButton, Tab,
        // Image and DropdownList, and no Link — the one place SwiftUI's
        // vocabulary is the richer of the two.
        "link" -> {}
        // The naming role. Compose has no member for it, and unlike the arms
        // above that is not the reason this one is empty: a
        // contentDescription is honoured on any node here, so nothing needs
        // unlocking. The role exists because ARIA prohibits a name on a
        // generic element, which is a web problem with a web answer. See
        // core/role.go's RoleGroup.
        "group" -> {}
        else -> {}
    }
}

/**
 * Maps one core.ValueRange onto Compose semantics, inside the same lambda
 * grMobRole and grMobSelected write into.
 *
 * # Two properties, because the range and the words are different claims
 *
 * `progressBarRangeInfo` is the numeric one, and it is one of the better
 * mappings in this file: TalkBack turns it into a percentage it localizes
 * itself, so a bar reports "45 percent" in the user's own language with no
 * string crossing the bridge. That is exactly what Go's ProgressBar could not
 * do while its value lived in the accessible name.
 *
 * `stateDescription` is the words. It is what an app supplies when the digits
 * are not what a listener wants to hear — "step 3 of 5" — and, unlike the
 * range, TalkBack honours it on any node at all.
 *
 * # The role is deliberately not consulted, for the reason grMobSelected's is
 * not
 *
 * Compose honours both of these on any node, so guarding them the way the two
 * web exporters do would drop a value this platform would otherwise have
 * announced. ARIA scopes aria-valuenow to six roles because ARIA scopes
 * things; Compose does not, and the framework is not stricter than the
 * platform it is talking to.
 *
 * # The branch on the numbers lives next door
 *
 * A bar with bounds and no position is *running with no idea how far*, which
 * is a state Compose can only spell as `ProgressBarRangeInfo.Indeterminate`.
 * A bar with a position takes the real range, its bounds defaulting to ARIA's
 * own 0 and 100 so that a bare percentage reads as one. A range that states
 * nothing numeric at all leaves the property alone — a `text` on an ordinary
 * node must not turn it into a progress bar.
 *
 * Those rules are ARIA's rather than Compose's and they are resolved by
 * [grMobProgressOf], in GrMobProgress.kt, which imports nothing and is
 * therefore runnable: android/verify compares it against Go's
 * core.ValueRange.Progress over internal/valuefixture's table. Until it moved
 * there the branch was checked by mobile/verify looking for the word
 * `Indeterminate` somewhere in this file — which would have stayed green for
 * a branch that reached it on the wrong condition.
 *
 * The parameter is `range` and not `value` for the reason grMobRole's is
 * `kind`: the surrounding lambda is a SemanticsPropertyReceiver, and a name
 * that shadows one of its properties stops the assignment compiling.
 */
fun SemanticsPropertyReceiver.grMobValue(range: GrMobStyle.ValueRange) {
    if (range.text.isNotEmpty()) stateDescription = range.text
    // The decision is grMobProgressOf's, in a file that imports nothing, so
    // android/verify can run it on a JVM against core.ValueRange.Progress.
    // What is left here is the assignment, which is the only part that needs
    // a semantics scope — and it is a `when` over four named readings rather
    // than a branch whose conditions have to be read to be understood.
    val p = grMobProgressOf(range.now, range.min, range.max)
    when (p.reading) {
        GRMOB_PROGRESS_DETERMINATE ->
            progressBarRangeInfo = ProgressBarRangeInfo(p.now, p.min..p.max)
        GRMOB_PROGRESS_INDETERMINATE ->
            progressBarRangeInfo = ProgressBarRangeInfo.Indeterminate
        // Unstated and empty-range both leave the property alone, and they do
        // it for different reasons: nothing numeric was claimed, versus a
        // claim Compose cannot hold — ProgressBarRangeInfo throws on an empty
        // range, and crashing a render over an accessibility annotation is
        // the wrong trade. Written as arms rather than as an `else` so the
        // second one is visible here instead of being an absence.
        GRMOB_PROGRESS_UNSTATED, GRMOB_PROGRESS_EMPTY_RANGE -> {}
    }
}

/**
 * Maps one core.SelectedState onto Compose semantics, inside the same lambda
 * grMobRole writes into.
 *
 * Compose has one property where ARIA has two attributes: `selected` covers
 * both aria-selected and aria-pressed, so unlike the two web targets this
 * needs no switch on the role — see the mapping table in Go's
 * core.Style.AccessibilitySelected.
 *
 * Unlike SwiftUI, this platform can state the *off* case: TalkBack announces
 * "not selected" for `selected = false`, which is what a tab strip needs so
 * that the four tabs that are not showing are announced as tabs rather than
 * as furniture. That is why core.SelectedState has three values and not two.
 *
 * The role is deliberately not consulted. Compose honours `selected` on any
 * node, so guarding it the way the web exporters do would drop a state this
 * platform would otherwise have announced — the web is strict because ARIA
 * scopes its attributes, not because the framework does.
 *
 * The parameter is `state` and not `selected` for the reason grMobRole's is
 * `kind`: `selected` inside a SemanticsPropertyReceiver is the property being
 * assigned one line down, and a parameter of that name would shadow it.
 *
 * One arm per line, string literals first, `else ->` last:
 * mobile/verify's TestKotlinSelectedCoversEveryState reads these arms out of
 * the source and holds them against core.SelectedStates().
 */
fun SemanticsPropertyReceiver.grMobSelected(state: String) {
    when (state) {
        "true" -> selected = true
        "false" -> selected = false
        else -> {}
    }
}
