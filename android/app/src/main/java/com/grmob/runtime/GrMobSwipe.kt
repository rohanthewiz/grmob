package com.grmob.runtime

import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.gestures.awaitHorizontalTouchSlopOrCancellation
import androidx.compose.foundation.gestures.awaitTouchSlopOrCancellation
import androidx.compose.foundation.gestures.awaitVerticalTouchSlopOrCancellation
import androidx.compose.foundation.gestures.drag
import androidx.compose.foundation.gestures.horizontalDrag
import androidx.compose.foundation.gestures.verticalDrag
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.input.pointer.PointerInputChange
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.unit.dp
import kotlin.math.abs

/**
 * core.SwipeMinDistance, in dp: the travel along the axis, press to release,
 * that makes a drag a swipe. The same number the web runtime and the SwiftUI
 * renderer read, so one swipe takes the same hand on all three.
 */
private val SwipeMinDistance = 40.dp

/**
 * core.OnSwipe and core.OnVerticalSwipe: one gesture per press, reported at
 * release as a direction word on the text channel ("leading", "trailing",
 * "up", "down").
 *
 * Applied by RenderNode, so any node type can carry it, and placed before
 * the node's own modifiers so the whole box, padding included, takes the
 * press.
 *
 * # Which axis is claimed
 *
 * The slop detector picks it. With only onSwipe the press is a swipe once it
 * crosses the touch slop sideways (awaitHorizontalTouchSlopOrCancellation),
 * so a vertical drag is never claimed and a verticalScroll around the node
 * still scrolls. onVerticalSwipe alone is the same turned a quarter. With
 * both, any slop crossing claims it. In every case the release then applies
 * core's rule: at least SwipeMinDistance along an axis the node claims, and
 * strictly further along it than across.
 *
 *	down (unconsumed or not) ──▶ slop on a claimed axis? ── no ─▶ nothing claimed
 *	                                     │ yes: consume
 *	                                     ▼
 *	                       drag, consuming every move
 *	                                     │ up
 *	                                     ▼
 *	              rule on (last − down) ──▶ runtime.textChanged(id, word)
 *
 * # Taps under the swipe
 *
 * The first down is awaited with requireUnconsumed = false. A child's
 * clickable consumes its down (detectTapGestures does), and Gallery's stage
 * is covered by clickable tap zones, so a detector that required an
 * unconsumed down would never see a press that started on one. From the slop
 * on every move is consumed, and a child's tap detector cancels when it sees
 * a consumed change, so the child's OnClick and OnLongPress do not run for a
 * press that became a drag. A child that claims the same axis first (a
 * horizontal Scroll inside a horizontal swipe) consumes the moves before
 * this detector sees them, since the Main pass runs child before parent, and
 * then the slop detector here cancels. The innermost claimant wins, as
 * nested scrolls do.
 *
 * # Direction
 *
 * Pointer positions are physical: x grows to the right in both layout
 * directions. So the horizontal word is resolved here against the node's
 * layout direction. Leading is left in LTR and right in RTL, the rule
 * core.SwipeDirection states.
 *
 * The callback IDs and the direction are read through rememberUpdatedState
 * so a pass that renumbers them does not restart the detector, which would
 * drop a press in progress. The detector is keyed only on which axes are
 * claimed, because that changes which slop detector runs.
 */
@Composable
internal fun swipeGesture(onSwipe: String, onVerticalSwipe: String): Modifier {
    val runtime = LocalGrMobRuntime.current
    val horizontalId by rememberUpdatedState(onSwipe)
    val verticalId by rememberUpdatedState(onVerticalSwipe)
    val rtl by rememberUpdatedState(LocalLayoutDirection.current == LayoutDirection.Rtl)
    val claimsH = onSwipe.isNotEmpty()
    val claimsV = onVerticalSwipe.isNotEmpty()

    return Modifier.pointerInput(claimsH, claimsV) {
        val min = SwipeMinDistance.toPx()
        awaitEachGesture {
            val down = awaitFirstDown(requireUnconsumed = false)
            // Each branch returns the change that crossed the slop, or null
            // when the press ended or another detector took the pointer.
            val slopped = when {
                claimsH && claimsV -> awaitTouchSlopOrCancellation(down.id) { change, _ -> change.consume() }
                claimsH -> awaitHorizontalTouchSlopOrCancellation(down.id) { change, _ -> change.consume() }
                else -> awaitVerticalTouchSlopOrCancellation(down.id) { change, _ -> change.consume() }
            } ?: return@awaitEachGesture
            var last: Offset = slopped.position
            // true when the pointer lifted; false when it was cancelled (a
            // parent took it), in which case nothing is reported.
            val onMove: (PointerInputChange) -> Unit = { change ->
                change.consume()
                last = change.position
            }
            val lifted = when {
                claimsH && claimsV -> drag(slopped.id, onMove)
                claimsH -> horizontalDrag(slopped.id, onMove)
                else -> verticalDrag(slopped.id, onMove)
            }
            if (!lifted) return@awaitEachGesture

            val dx = last.x - down.position.x
            val dy = last.y - down.position.y
            val ax = abs(dx)
            val ay = abs(dy)
            if (claimsH && ax >= min && ax > ay) {
                // Finger toward the left edge is leading in LTR.
                val towardLeft = dx < 0
                val leading = if (rtl) !towardLeft else towardLeft
                if (horizontalId.isNotEmpty()) runtime.textChanged(horizontalId, if (leading) "leading" else "trailing")
            } else if (claimsV && ay >= min && ay > ax) {
                if (verticalId.isNotEmpty()) runtime.textChanged(verticalId, if (dy < 0) "up" else "down")
            }
        }
    }
}
