package com.grmob.runtime

import android.icu.text.RelativeDateTimeFormatter
import android.icu.util.ULocale
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
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.draw.rotate
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.layout
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
import androidx.compose.ui.semantics.selectableGroup
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.stateDescription
import androidx.compose.ui.unit.dp
import org.json.JSONObject
import java.util.Locale
import kotlin.math.roundToInt
import androidx.compose.animation.core.withInfiniteAnimationFrameMillis
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.layout.Measurable
import androidx.compose.ui.layout.MeasureResult
import androidx.compose.ui.layout.MeasureScope
import androidx.compose.ui.node.LayoutModifierNode
import androidx.compose.ui.node.ModifierNodeElement
import androidx.compose.ui.unit.Constraints
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

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
    /**
     * core.Spin: milliseconds per revolution, negative for anticlockwise, 0
     * for still. Applied by [SpinElement] beside [rotate]; see core.Spin for
     * why it is a rotation loop and not a general repeating transition.
     */
    val spinMs: Int,
    /**
     * core.Translate, one axis each: a dp amount plus a fraction of the box's
     * own extent, resolved at placement by [TranslateElement]. Leading-relative
     * in x, because placeRelative mirrors under RTL; see core.Translate.
     */
    val translateX: GrMobShift = GrMobShift.Zero,
    val translateY: GrMobShift = GrMobShift.Zero,
    /**
     * core.Overflow. Only "hidden" is read, as a clip to the box in
     * boxModifier: it is what keeps a child translated out of its parent (a
     * Drawer's shut panel) from drawing over whatever sits beside the parent.
     * "scroll"/"auto" stay the web's, as before.
     */
    val overflow: String = "",
    val align: String,
    val display: String,
    val width: String,
    /** core.MaxWidth: CSS `max-width`, applied with Width in widthModifier. */
    val maxWidth: String,
    /** core.MinWidth: CSS `min-width`, px or %; see widthModifier. */
    val minWidth: String = "",
    val height: String,
    /** core.MinHeight: CSS `min-height`, px or %; see heightModifier. */
    val minHeight: String = "",
    val borderColor: Color?,
    val borderWidth: Float,
    /**
     * core.AccentColor: the tint of a platform-drawn control — Switch, Slider,
     * Checkbox — fed into Material's own colour slots by the renderer rather
     * than drawn by boxModifier. Null leaves Material's colours. See
     * core.Style's AccentColor.
     */
    val accentColor: Color? = null,
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
    /**
     * core.MaxLines: the most lines a Text draws, ending in an ellipsis where
     * it is cut; 0 for no limit. Read by GrMobText alone (Renderer.kt).
     */
    val maxLines: Int,
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
     * Go's core.CurrentKind, verbatim: "page", "step", "true", or "" for a node
     * that is not the current item of a set. Folded into `selected` by
     * grMobCurrent below. Defaulted so a GrMobStyle built by hand elsewhere
     * needs no new argument.
     */
    val accessibilityCurrent: String = "",
    /**
     * Go's core.Style.AccessibilityKeyShortcuts, verbatim (aria-keyshortcuts
     * spelling). Not spent in boxModifier: a hardware keyboard's chord reaches
     * the Activity, not a composable, so GrMobRuntime.handleKeyEvent reads it
     * off the tree. Defaulted like accessibilityCurrent.
     */
    val accessibilityKeyShortcuts: String = "",
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

    /**
     * This node's property-change animation spec (callers gate on transitionMs > 0).
     *
     * Reduced motion needs nothing here. Android's reduce-motion switch is
     * "Remove animations", which sets Settings.Global.ANIMATOR_DURATION_SCALE
     * to 0; Compose's window recomposer observes that setting and installs it
     * as the MotionDurationScale of every animation coroutine, and a tween run
     * under scale 0 plays straight to its end value (read in Compose 1.10's
     * WindowRecomposer.android.kt and SuspendAnimation.kt). So the background
     * fade, animateContentSize and item placement all snap already, live, which
     * is core.Transition's rule. Reading the setting again here would only add
     * a second, non-observing copy of it. SpinNode does not use a scaled
     * animation and keeps turning, by design; see core.Spin.
     */
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
                spinMs = obj.optInt("Spin", 0),
                translateX = GrMobShift.parse(obj.optString("TranslateX")),
                translateY = GrMobShift.parse(obj.optString("TranslateY")),
                overflow = obj.optString("Overflow"),
                align = obj.optString("Align"),
                display = obj.optString("Display"),
                width = obj.optString("Width"),
                maxWidth = obj.optString("MaxWidth"),
                minWidth = obj.optString("MinWidth"),
                height = obj.optString("Height"),
                minHeight = obj.optString("MinHeight"),
                borderColor = parseColor(obj.optString("BorderColor")),
                borderWidth = obj.optDouble("BorderWidth", 0.0).toFloat(),
                accentColor = parseColor(obj.optString("AccentColor")),
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
                maxLines = obj.optInt("MaxLines", 0).coerceAtLeast(0),
                accessibilityLabel = obj.optString("AccessibilityLabel"),
                accessibilityHint = obj.optString("AccessibilityHint"),
                accessibilityHidden = obj.optBoolean("AccessibilityHidden", false),
                accessibilityRole = obj.optString("AccessibilityRole"),
                accessibilitySelected = obj.optString("AccessibilitySelected"),
                accessibilityCurrent = obj.optString("AccessibilityCurrent"),
                accessibilityKeyShortcuts = obj.optString("AccessibilityKeyShortcuts"),
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
    // Bound out here beside selectedState, which grMobCurrent reads with it.
    val currentKind = accessibilityCurrent
    // And again: inside the lambda `value` would be nothing in particular, but
    // the range has to be read off `this` before the receiver changes.
    val valueRange = accessibilityValue
    if (accessibilityHidden) {
        m = m.clearAndSetSemantics { }
    } else if (accessibilityLabel.isNotEmpty() || accessibilityHint.isNotEmpty() ||
        isDisabled || kind.isNotEmpty() || selectedState.isNotEmpty() ||
        currentKind.isNotEmpty() || valueRange.stated()
    ) {
        val description = listOf(grMobCurrentLabel(accessibilityLabel, currentKind, valueRange.text), accessibilityHint)
            .filter { it.isNotEmpty() }.joinToString(". ")
        // core.CurrentDate's word, when no stated value holds the state slot.
        val currentState = grMobCurrentState(accessibilityLabel, currentKind, valueRange.text)
        // A named node is ONE element, the way the other two targets already
        // read it.
        //
        // # What two nodes looked like
        //
        // A comps.Calendar day is a single Go node: a Box carrying the label
        // ("14 March 2026"), the gridcell role, the selected state and an
        // onClick, with a Text of the day number inside it. Compose drew it as
        // two accessibility nodes — this element, and the digit's own text
        // node under it — so a reader met the cell twice and the grid's touch
        // exploration had two targets per square.
        //
        // Merging is the same statement SwiftUI's grMobAccessibility makes for
        // a labelled container (`accessibilityElement(children: .combine)`)
        // and the same one aria-label makes on the web, where an accessible
        // name replaces the element's contents rather than joining them.
        //
        // # Why the LABEL is the condition, and not the branch
        //
        // This branch is also entered for a node that declares only a role, or
        // only a disabled or selected state. Merging there would be wrong in a
        // way that is much worse than two nodes: the calendar's own
        // `core.RoleRow` Row would swallow its seven cells and a week would
        // become one element. A label is the author saying "this is one thing
        // and here is its name"; a role on its own says the opposite, that the
        // node is a container of things.
        m = m.semantics(mergeDescendants = accessibilityLabel.isNotEmpty()) {
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
            grMobCurrent(currentKind, selectedState)
            grMobValue(valueRange)
            if (currentState.isNotEmpty()) stateDescription = currentState
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
    // Width, MaxWidth and MinWidth are resolved together, inside the margin:
    // CSS's max-width and min-width limit the border box, and the space
    // reserved around it is not part of what they bound. See widthModifier
    // for why they cannot be independent modifiers. Height and MinHeight
    // likewise, in heightModifier.
    m = m.then(widthModifier(width, maxWidth, minWidth))
    m = m.then(heightModifier(height, minHeight))

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
    // core.Translate, outside both rotations: CSS applies the individual
    // `translate` before `rotate` and `transform`, so a turned box slides
    // along the screen's axes. A placement offset rather than a graphics
    // layer, because placeRelative mirrors x under RTL (Translate is leading-
    // relative) and needs no layer; the size reported to the parent is
    // untouched, so nothing around the node reflows, and Compose hit-tests
    // the box where it is placed. Guarded like rotate: an untranslated node
    // gains no layout node.
    if (!translateX.isZero || !translateY.isZero) {
        m = m.then(TranslateElement(translateX, translateY))
    }
    if (rotate != 0f) m = m.rotate(rotate)
    // core.Spin, at the same layer position as the fixed angle and for the
    // same reasons: it must turn the whole painted box and the touch target
    // with it. Directly inside the fixed rotation; both turn about the layout
    // bounds' centre, so the order between the two draws the same pixels.
    // Guarded like rotate, so a still node gains no layout node.
    if (spinMs != 0) m = m.then(SpinElement(spinMs))

    val shape = if (borderRadius > 0f) RoundedCornerShape(borderRadius.dp) else null
    if (shadow > 0f) {
        m = m.shadow(elevation = shadow.dp, shape = shape ?: RoundedCornerShape(0.dp))
    }
    if (shape != null) {
        m = m.clip(shape)
    } else if (overflow == "hidden") {
        // A rounded box already clips to its shape above; a square one clips
        // only when asked, since clipping every box would cut off overflow
        // (shadows, a translated child) that CSS's default visible allows.
        m = m.clipToBounds()
    }
    background?.let { m = m.background(it) }
    if (borderWidth > 0f && borderColor != null) {
        m = m.border(borderWidth.dp, borderColor, shape ?: RoundedCornerShape(0.dp))
    }
    m = m.then(gestures)
    // The content starts inside the border as well as the padding: CSS's
    // border-box, where a 2px border pushes a box's children 2px in. Compose's
    // Modifier.border only paints — it reserves nothing — so without this a
    // child at the top edge was drawn under the stroke, and inside a round
    // clip it was cut by the rim: comps.Spinner's orbiting dot was clipped in
    // half on Android where the web draws it whole, inset from the ring.
    //
    // The same guard as the border itself (both halves), so a width with no
    // colour, which paints nothing on any target, moves nothing either.
    val borderInset = if (borderWidth > 0f && borderColor != null) borderWidth else 0f
    if (padding != Edges0 || borderInset > 0f) {
        m = m.padding(
            start = (padding.left + borderInset).dp, top = (padding.top + borderInset).dp,
            end = (padding.right + borderInset).dp, bottom = (padding.bottom + borderInset).dp,
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
 * core.Width with core.MaxWidth and core.MinWidth: CSS's
 * `max(min-width, min(width, max-width))`, and a capped box placed at the
 * start of any wider slot its parent forces on it.
 *
 * Without a MaxWidth this is exactly dimensionModifier, so the uncapped chain —
 * nearly every node — is unchanged by the cap's existence.
 *
 * Why not `Modifier.widthIn(max = cap)` beside the size modifier: every
 * ordering of the two loses a case, because Compose's size modifiers enforce
 * the incoming constraints and whichever runs outside wins.
 *
 * ```
 *   chain (outer → inner)                  incoming   result    CSS
 *   ─────────────────────                  ────────   ──────    ───
 *   widthIn(max 520) . width(600)          0..400     400       400  ✓
 *   width(600) . widthIn(max 520)          0..800     600       520  ✗ cap coerced away
 *   widthIn(max 520) . fillMaxWidth(0.5)   0..800     260       400  ✗ half of the CAP
 *   fillMaxWidth() [stretch, from the      800..800   800       520  ✗ fill forces min = max
 *     parent] . widthIn(max 520)
 * ```
 *
 * The last row is the common one: ColumnChildren stretches a child by handing
 * it fillMaxWidth as `extra`, which boxModifier puts outermost, so the child
 * arrives here with minWidth == maxWidth == the column's width. A size
 * modifier cannot relax a minimum; a layout modifier can, by measuring the
 * content at the cap and REPORTING the forced width with the content placed at
 * x = 0. That is CSS's picture of a stretched item stopped by max-width: the
 * box is the cap, at the start of the line, and the rest of the line is empty.
 *
 * The rules, in the order the lambda applies them:
 *
 * ```
 *   limit     = cap in px                       (points × density)
 *             | cap% × incoming maxWidth        (bounded only; else no limit)
 *   width%    → min = max = width% × incoming maxWidth   (dimensionModifier's
 *               fillMaxWidth(fraction), resolved against the SAME incoming width
 *               as the cap rather than against the capped one)
 *   maxWidth  = min(maxWidth, limit);  minWidth = min(minWidth, maxWidth)
 *   measure; report width coerced into the INCOMING constraints; place at 0
 * ```
 *
 * A points Width stays the plain `Modifier.width` and runs inside the layout:
 * that ordering is the first row of the table, which is already right.
 * `placeRelative` so that "start" is the right edge under RTL, as CSS's
 * inline-start is.
 *
 * Not handled: a FlexGrow child in a Row whose cap binds. Modifier.weight
 * fixes the child's share as both minimum and maximum, so the capped box sits
 * at the start of its share and the remainder stays empty, where CSS would
 * redistribute it to the other growers. iOS has the same gap
 * (GrMobMaxWidthLayout).
 */
private fun widthModifier(width: String, maxWidth: String, minWidth: String = ""): Modifier {
    val cap = parseWidthCap(maxWidth)
    // A floor is written in the same forms as a cap (px or %, "auto" and
    // "none" meaning nothing), so it is parsed by the same function.
    val floor = parseWidthCap(minWidth)
    if (cap == null && floor == null) return dimensionModifier(width, horizontal = true)
    val fraction = widthFraction(width)
    val inner = if (fraction != null) Modifier else dimensionModifier(width, horizontal = true)
    return Modifier.layout { measurable, constraints ->
        val bounded = constraints.hasBoundedWidth
        // A percentage of an unbounded width is CSS's percentage against an
        // indefinite containing block: it behaves as `none` for a cap and as
        // `0` for a floor, which is null (no bound) either way.
        val limit: Int? = cap?.let {
            when {
                !it.isFraction -> it.amount.dp.roundToPx()
                bounded -> (constraints.maxWidth * it.amount).roundToInt()
                else -> null
            }
        }
        val least: Int? = floor?.let {
            when {
                !it.isFraction -> it.amount.dp.roundToPx()
                bounded -> (constraints.maxWidth * it.amount).roundToInt()
                else -> null
            }
        }
        var minW = constraints.minWidth
        var maxW = constraints.maxWidth
        if (fraction != null && bounded) {
            val w = (constraints.maxWidth * fraction).roundToInt()
            minW = w
            maxW = w
        }
        if (limit != null) {
            maxW = minOf(maxW, limit)
            minW = minOf(minW, maxW)
        }
        // The floor last, because CSS's min-width beats max-width (and a
        // declared width) when they disagree. It may raise the measurement
        // past the incoming maximum: the box then overflows its slot, as a CSS
        // box with a min-width wider than its container does, and the report
        // below still stays inside the incoming constraints.
        if (least != null) {
            minW = maxOf(minW, least)
            maxW = maxOf(maxW, least)
        }
        val placeable = measurable.measure(constraints.copy(minWidth = minW, maxWidth = maxW))
        // Reported inside the incoming constraints, which a layout must honour:
        // a parent that forced a wider minimum gets that width, and the capped
        // content sits at its start.
        val reported = placeable.width.coerceIn(constraints.minWidth, constraints.maxWidth)
        layout(reported, placeable.height) { placeable.placeRelative(0, 0) }
    }.then(inner)
}

/**
 * core.Height with core.MinHeight: CSS's `max(height, min-height)`.
 *
 * The same shape as widthModifier's floor, for the same reason. A
 * `Modifier.heightIn(min = …)` cannot raise a minimum the parent has already
 * fixed (a weight or a stretch arrives with minHeight == maxHeight), and
 * placed inside a Height it is coerced away. A layout modifier measures the
 * content with the floor folded into the constraints and reports inside the
 * incoming ones.
 *
 * ```
 *   floor     = px                               (points × density)
 *             | % × incoming maxHeight           (bounded only; else none)
 *   minHeight = max(minHeight, floor);  maxHeight = max(maxHeight, floor)
 *   measure; report height coerced into the INCOMING constraints; place at 0
 * ```
 *
 * In a scrolled column (maxHeight infinite) the floor is simply the minimum,
 * which is the case comps.RichTextEditor's MinHeight exists for: an empty
 * document still gives the reader a place to write. The Height itself stays
 * dimensionModifier's and runs inside, so a points Height below the floor is
 * raised to it, as CSS raises it.
 */
private fun heightModifier(height: String, minHeight: String): Modifier {
    val floor = parseWidthCap(minHeight) ?: return dimensionModifier(height, horizontal = false)
    return Modifier.layout { measurable, constraints ->
        val least: Int? = when {
            !floor.isFraction -> floor.amount.dp.roundToPx()
            constraints.hasBoundedHeight -> (constraints.maxHeight * floor.amount).roundToInt()
            else -> null
        }
        val inner = if (least == null) constraints else constraints.copy(
            minHeight = maxOf(constraints.minHeight, least),
            maxHeight = maxOf(constraints.maxHeight, least),
        )
        val placeable = measurable.measure(inner)
        val reported = placeable.height.coerceIn(constraints.minHeight, constraints.maxHeight)
        layout(placeable.width, reported) { placeable.placeRelative(0, 0) }
    }.then(dimensionModifier(height, horizontal = false))
}

/**
 * A parsed core.MaxWidth: points, or a fraction of the incoming max width.
 *
 * The accepted strings are dimensionModifier's for Width, plus the two
 * spellings of "no cap": "none" (CSS's initial value) and "auto". A negative
 * number is invalid CSS and is dropped rather than clamped to zero, which
 * would collapse the box instead of leaving it uncapped. A percentage above
 * 100 is kept; it never binds, as in CSS.
 */
// Internal rather than private: Renderer.kt's rowChildWidth resolves a
// percentage MinWidth, and divides by a percentage MaxWidth, against a Row's
// offer with the same parse.
internal class WidthCap(val amount: Float, val isFraction: Boolean)

internal fun parseWidthCap(value: String): WidthCap? {
    if (value.isEmpty() || value == "none" || value == "auto") return null
    if (value.endsWith("%")) {
        val pct = value.dropLast(1).toFloatOrNull() ?: return null
        return if (pct < 0f) null else WidthCap(pct / 100f, isFraction = true)
    }
    val number = value.removeSuffix("px").toFloatOrNull() ?: return null
    return if (number < 0f) null else WidthCap(number, isFraction = false)
}

/** A percentage Width as dimensionModifier reads it (clamped to 0..1), else null. */
private fun widthFraction(value: String): Float? {
    if (!value.endsWith("%")) return null
    val pct = value.dropLast(1).toFloatOrNull() ?: return null
    return (pct / 100f).coerceIn(0f, 1f)
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
 *
 * AccessibilityHasPopup is not read here either.
 *
 * Go's popup kind says a control opens a dialog before it is pressed. Compose
 * semantics has no property for what a control opens. The near miss is
 * Role.DropdownList, which names a kind of control rather than what it opens,
 * and would have TalkBack announce comps.Menu's "⋯" as a drop-down list. A
 * Modal presents here as a Dialog window, which TalkBack announces as it
 * opens, so the warning arrives one step late rather than not at all. The key
 * is deliberately not parsed; mobile/verify/expanded_test.go pins both halves.
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
        // A radio group and one radio in it — the one choice pair Compose can
        // name at both ends. selectableGroup() is the container semantics
        // Modifier.selectableGroup() writes, which TalkBack reads as "these
        // are one set", and Role.RadioButton is the control. The checked
        // state is grMobSelected's `selected`, which is what Compose's own
        // RadioButton reports through Modifier.selectable.
        "radiogroup" -> selectableGroup()
        "radio" -> role = Role.RadioButton
        // An interactive grid and one cell in it. Compose has no container
        // semantics for a grid — collectionInfo describes row and column
        // counts this prop does not carry — and TalkBack moves through one by
        // swipe rather than by arrow key, so the container's word is the loss.
        "grid" -> {}
        // The cell is a control, and Role.Button is the control word Compose
        // has. It is also what comps.Calendar's days announced as here while
        // they were RoleButton, so moving the widget onto the grid pair
        // changed nothing a TalkBack user hears.
        "gridcell" -> role = Role.Button
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
        // The field that owns a popup list. Role.DropdownList is the near miss
        // and is turned down: TalkBack would announce the text field as a
        // drop-down list, a control you open, where this one is typed into.
        // The field stays the edit box it is, and the list under it is reached
        // by swiping, as every collection is. See core.RoleComboBox.
        "combobox" -> {}
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

/**
 * Folds one core.CurrentKind into Compose's `selected` semantics.
 *
 * Compose has no current property. `selected` is what Material's own
 * NavigationBar sets on the destination it is showing, so TalkBack announces a
 * current bottom-bar cell here as it announces the platform's. The kind itself
 * (page, step, true) has nowhere to go and is not distinguished.
 *
 * A stated core.SelectedState wins: grMobSelected has already written it, and
 * a node that says both is making the more specific claim with that field.
 *
 * "date" is not folded. A calendar's today cell stating selected would be
 * announced as the chosen day, and a calendar has a chosen day of its own; the
 * fact goes into stateDescription instead, through [grMobCurrentState]. See Go's
 * core.CurrentKind, "CurrentDate is the exception to the fold".
 */
fun SemanticsPropertyReceiver.grMobCurrent(kind: String, state: String) {
    if (kind.isNotEmpty() && kind != "date" && state.isEmpty()) selected = true
}

/**
 * The accessible name, with core.CurrentDate spoken into it only when the
 * state channel is already taken.
 *
 * Compose has no current property and "today" must not become `selected`
 * (see [grMobCurrent]). The word goes into `stateDescription` instead
 * ([grMobCurrentState]): TalkBack reads a node's state, name and role as
 * separate parts, in the order the user's "element description order"
 * setting chooses, with its own pauses. The joining is then the screen
 * reader's, in the user's language, rather than a fixed ", " in this file.
 *
 * `stateDescription` holds one text, and a stated core.AccessibilityValue
 * already fills it ([grMobValue]). Only then is the word folded into the name,
 * after a ", " (a pause to every TTS engine TalkBack drives):
 *
 * ```
 *   AccessibilityValue text   contentDescription    stateDescription
 *   none                      label                 "today"
 *   stated                    label + ", today"     the stated text
 * ```
 *
 * The word is the platform's, in the device's language: [grMobTodayWord]
 * asks ICU, as the web's aria-current="date" asks the screen reader. A node
 * with no name gets neither: "today" on its own would announce only the
 * suffix.
 */
fun grMobCurrentLabel(label: String, kind: String, valueText: String = ""): String =
    if (kind == "date" && label.isNotEmpty() && valueText.isNotEmpty()) "$label, ${grMobTodayWord()}" else label

/** The stateDescription core.CurrentDate speaks, or "" (see [grMobCurrentLabel]). */
fun grMobCurrentState(label: String, kind: String, valueText: String): String =
    if (kind == "date" && label.isNotEmpty() && valueText.isEmpty()) grMobTodayWord() else ""

/**
 * ICU's word for the current day ("today", "aujourd’hui", "heute") in the
 * default locale.
 *
 * `RelativeDateTimeFormatter.format(THIS, DAY)` is the named form a date
 * picker's "Today" row uses, and android.icu has carried it since API 24,
 * this app's minSdk. It is cached per locale because the label is rebuilt on
 * every recomposition of a today cell, and cleared by a locale change simply
 * by the key no longer matching.
 */
@Volatile
private var grMobTodayCache: Pair<Locale, String>? = null

private fun grMobTodayWord(): String {
    val locale = Locale.getDefault()
    grMobTodayCache?.let { (cached, word) -> if (cached == locale) return word }
    val word = RelativeDateTimeFormatter.getInstance(ULocale.forLocale(locale))
        .format(RelativeDateTimeFormatter.Direction.THIS, RelativeDateTimeFormatter.AbsoluteUnit.DAY)
    grMobTodayCache = locale to word
    return word
}

/**
 * core.Spin on Compose: the box turns one revolution every [periodMs],
 * clockwise for a positive period, for as long as it is composed.
 *
 * # Why a modifier node and not rememberInfiniteTransition
 *
 * [boxModifier] is a plain function that builds a fresh chain on every
 * recomposition; it has no composition of its own to `remember` an infinite
 * transition in. The alternatives were `Modifier.composed` (discouraged: it
 * re-materialises on every chain rebuild) or threading a @Composable through
 * every caller. A [ModifierNodeElement] carries its own lifecycle instead:
 * the node survives chain rebuilds as long as the element compares equal (a
 * data class on the period), its coroutine starts on attach and is cancelled
 * on detach, so a node that leaves composition — Display none included —
 * stops drawing frames with nothing to clean up.
 *
 * # Why the angle is read in the layer block
 *
 * The angle is snapshot state read only inside `placeWithLayer`'s block.
 * Compose observes reads there per layer, so each frame invalidates the
 * graphics layer's parameters alone: no recomposition, no remeasure, no
 * relayout of anything around the spinner. That is the whole cost of a spin.
 *
 * # Elapsed time, not increments
 *
 * The angle is computed from the frame time since the loop started, modulo
 * the period, as the web's keyframes and SwiftUI's TimelineView are. A
 * dropped frame skips an angle instead of slowing the spin.
 *
 * [withInfiniteAnimationFrameMillis] rather than withFrameMillis: it is the
 * frame source Compose's own infinite animations use, and it tells test
 * clocks and idling machinery that this loop never finishes, so a UI test
 * waiting for idle does not hang on a visible spinner. It does not read the
 * animator duration scale, and that is the decision rather than a gap: a
 * spinner keeps turning under "Remove animations", as core.Spin's "Reduced
 * motion: it keeps turning" argues.
 */
private data class SpinElement(val periodMs: Int) : ModifierNodeElement<SpinNode>() {
    override fun create() = SpinNode(periodMs)

    override fun update(node: SpinNode) {
        // Read by the running loop on its next frame, so a changed period
        // takes effect without restarting the coroutine.
        node.periodMs = periodMs
    }
}

private class SpinNode(var periodMs: Int) : Modifier.Node(), LayoutModifierNode {
    private var angle by mutableFloatStateOf(0f)

    override fun onAttach() {
        coroutineScope.launch {
            var start = -1L
            while (isActive) {
                withInfiniteAnimationFrameMillis { now ->
                    if (start < 0) start = now
                    val period = kotlin.math.abs(periodMs).coerceAtLeast(1)
                    val fraction = ((now - start) % period).toFloat() / period
                    angle = if (periodMs < 0) -360f * fraction else 360f * fraction
                }
            }
        }
    }

    override fun MeasureScope.measure(measurable: Measurable, constraints: Constraints): MeasureResult {
        // Layout-transparent: measured and sized exactly as without the spin,
        // matching core.Rotate's "paint transform, not a layout one". The
        // layer's default transform origin is the centre.
        val placeable = measurable.measure(constraints)
        return layout(placeable.width, placeable.height) {
            placeable.placeWithLayer(0, 0) { rotationZ = angle }
        }
    }
}

/**
 * One axis of core.Translate, resolved from its wire string: an amount in dp
 * and a fraction of the node's own extent on that axis.
 *
 * Two numbers rather than one because the fraction cannot become pixels until
 * the box is measured, which happens after composition, where a Transition
 * animates the values (Renderer.kt's animatedStyle). Animating the pair
 * linearly is also how CSS interpolates "-100%" to none. One core value is
 * either a length or a percentage, so one of the two is always 0 unless a
 * transition is between the two kinds.
 */
data class GrMobShift(val amount: Float, val fraction: Float) {
    val isZero: Boolean get() = amount == 0f && fraction == 0f

    companion object {
        val Zero = GrMobShift(0f, 0f)

        /**
         * "Npx", a bare number (dp) or "N%". Anything else is zero, which is
         * what the web targets and SwiftUI make of it too.
         */
        fun parse(value: String): GrMobShift {
            val v = value.trim()
            if (v.endsWith("%")) {
                val pct = v.dropLast(1).trim().toFloatOrNull()
                return if (pct == null || !pct.isFinite()) Zero else GrMobShift(0f, pct / 100f)
            }
            val n = v.removeSuffix("px").trim().toFloatOrNull()
            return if (n == null || !n.isFinite()) Zero else GrMobShift(n, 0f)
        }
    }
}

/**
 * core.Translate as a layout modifier: the box is measured and sized exactly
 * as without it and placed at the resolved offset. placeRelative, not place,
 * so x is mirrored under RTL; for a child the same size as this layout,
 * mirroring x gives -x, which is Translate's leading-relative rule.
 *
 * A data class so an unchanged offset compares equal and Compose skips the
 * update; a changed one (each frame of a transition) updates the node, and a
 * node's default auto-invalidation re-runs measure, which re-places the box
 * without remeasuring its content under unchanged constraints.
 */
private data class TranslateElement(val x: GrMobShift, val y: GrMobShift) :
    ModifierNodeElement<TranslateNode>() {
    override fun create() = TranslateNode(x, y)

    override fun update(node: TranslateNode) {
        node.x = x
        node.y = y
    }
}

private class TranslateNode(var x: GrMobShift, var y: GrMobShift) : Modifier.Node(), LayoutModifierNode {
    override fun MeasureScope.measure(measurable: Measurable, constraints: Constraints): MeasureResult {
        val placeable = measurable.measure(constraints)
        // Percentages resolve against this box's own size, as CSS translate's
        // do, so "-100%" moves a panel exactly its own width.
        val dx = x.amount.dp.roundToPx() + (x.fraction * placeable.width).roundToInt()
        val dy = y.amount.dp.roundToPx() + (y.fraction * placeable.height).roundToInt()
        return layout(placeable.width, placeable.height) {
            placeable.placeRelative(dx, dy)
        }
    }
}
