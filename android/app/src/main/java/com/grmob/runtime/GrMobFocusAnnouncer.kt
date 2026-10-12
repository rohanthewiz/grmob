package com.grmob.runtime

import android.os.Build
import android.os.Handler
import android.os.Looper
import android.os.SystemClock
import android.view.View
import android.view.ViewGroup
import android.view.accessibility.AccessibilityEvent
import android.view.accessibility.AccessibilityManager
import android.view.accessibility.AccessibilityNodeInfo
import android.view.accessibility.AccessibilityNodeProvider
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.node.RootForTest
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.semantics.SemanticsActions
import androidx.compose.ui.semantics.SemanticsNode
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.semantics.getOrNull
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.unit.toSize
import kotlin.math.abs
import kotlin.math.sign

/**
 * Keeps TalkBack on the control keyboard focus is on, where Compose 1.7.6
 * loses it: a Tab, arrow or focus command that lands on a control off
 * screen.
 *
 * # The gap (N-058)
 *
 * TalkBack follows the keyboard by listening for TYPE_VIEW_FOCUSED and
 * putting its own (accessibility) focus on the event's source. Compose sends
 * that event from AndroidComposeViewAccessibilityDelegateCompat's
 * sendSemanticsPropertyChangeEvents. That compares each node with the copy
 * kept from the previous 100ms batch and fires when SemanticsProperties.
 * Focused turns true. A node with no previous copy is skipped. The maps hold
 * only nodes inside the visible region (getAllUncoveredSemanticsNodes), so a
 * control that was off screen has no copy:
 *
 *     Tab ─► focus moves to ☰ (below the fold) ─► bringIntoView scrolls
 *            batch N    ☰ absent from both maps               no event
 *            batch N+1  ☰ in current, absent from previous ─► skipped
 *            batch N+2  ☰ Focused true → true               ─► no change
 *
 * Seen on the emulator (TalkBack 16.0) on lesson 4.18. With TalkBack off, Tab
 * goes ‹ Contents → Copy → ☰ → ‹ Prev → Next ›, and Compose's focus is right
 * at every step. With TalkBack on, Compose sent TYPE_VIEW_FOCUSED for
 * Contents, Copy and Next (already on screen) and nothing for ☰ and ‹ Prev,
 * each scrolled up from below the fold. TalkBack kept the last control it
 * was told about, or read whatever its own scroll handling picked ("12 notes
 * in this section", which no Tab visits). An Enter went to Compose's focus,
 * not to the node just spoken. On a long page (4.34, 4.16) almost every Tab
 * lands below the fold, which is why most Tabs there spoke nothing.
 *
 * # Two more ways TalkBack lost the keyboard
 *
 * - **An event TalkBack ignored.** On 4.34 each poll option sat just past the
 *   bottom edge, so Compose had a copy and did send TYPE_VIEW_FOCUSED. It came
 *   mid-scroll, TalkBack did nothing with it, and the Tab was silent.
 * - **A scroll cut short.** On 4.9 a Tab from the date field to ‹ Prev, about
 *   two screens down, stopped scrolling 745px short (7879 of 8624). TalkBack
 *   moved its own focus to visible prose as its node left the screen, and
 *   that interrupted Compose's bring-into-view. The focused control was left
 *   off screen, out of Compose's tree, where no event can name it.
 *
 * # What this does
 *
 * It watches the events Compose sends, from the Compose view's parent: each
 * batch reaches ViewParent.requestSendAccessibilityEvent, and an
 * AccessibilityDelegate on the parent sees it. After each batch it finds the
 * node holding SemanticsProperties.Focused. When that changes, the node
 * becomes the target, and the target is held until TalkBack is on it:
 *
 *     target off screen, scroll settled ─► scroll it on (showOnScreen)
 *     TalkBack's focus on target        ─► done
 *     still moving (scroll, or < 300ms) ─► wait for it to settle
 *     settled, TalkBack elsewhere       ─► send TYPE_VIEW_FOCUSED,
 *                                          once more 1.2s later if ignored
 *
 * "TalkBack is on it" is asked of Compose's own provider: its findFocus
 * answers with the accessibility-focused node. The event is the one Compose's
 * createEvent builds. Waiting for things to settle means a move that Compose
 * announced and TalkBack followed (the usual case: a control already on
 * screen) is seen as done, and nothing extra is sent.
 *
 * Each target gets at most two scrolls and two events, and lapses
 * [GIVE_UP_MS] after its last step. A control focused off screen and never
 * scrolled in is not announced later, when the reader's own scrolling
 * happens to bring it into view.
 *
 * Still heard: when TalkBack's node scrolls away, TalkBack may first read
 * whatever it moves to. The correction follows once the scroll settles.
 *
 * # When it acts
 *
 * Only with touch exploration on, which is what a screen reader turns on
 * (TalkBack, and Samsung's), and only out of touch mode, which is what a key
 * press puts the window in. Any accessibility service makes Compose send
 * events, but one with no accessibility focus (an autofill service, Switch
 * Access) has nothing to follow. In touch mode Compose's focus is not what
 * the reader is using, and TalkBack's own focus should not be moved to it.
 * It is per window: GrMobRoot and every Modal (its own Dialog window and
 * Compose view) each run one.
 *
 * # Why from the outside
 *
 * The fix belongs in Compose's delegate, which is internal. RootForTest is
 * the public way to reach a root's SemanticsOwner. A semantics node's id is
 * the virtual view id Compose gives the provider, except the root, which is
 * the host view. If a later Compose release announces these moves itself,
 * this sees TalkBack on the target and does nothing.
 */
@Composable
internal fun AnnounceFocusMovesToScreenReader() {
    val view = LocalView.current
    DisposableEffect(view) {
        // The parent sees every event the Compose view sends. It is a
        // ComposeView for the Activity's root and a DialogLayout for a Modal.
        val parent = view.parent as? ViewGroup
        val root = view as? RootForTest
        if (parent == null || root == null) return@DisposableEffect onDispose { }
        val announcer = FocusAnnouncer(view, parent, root)
        announcer.attach()
        onDispose { announcer.detach() }
    }
}

private class FocusAnnouncer(
    private val view: View,
    private val parent: ViewGroup,
    private val root: RootForTest,
) : View.AccessibilityDelegate() {
    private val handler = Handler(Looper.getMainLooper())
    private val accessibility = view.context.getSystemService(AccessibilityManager::class.java)

    /** A delegate already on the parent. Events still go through it. */
    private var previous: View.AccessibilityDelegate? = null

    /** The focused node seen at the last check, or null for none. */
    private var lastFocused: Int? = null

    /** The node TalkBack should be on and is not yet; see the class comment. */
    private var target: Int? = null
    private var targetSince = 0L
    private var announcements = 0
    private var scrolls = 0

    /** When Compose last reported a scroll, which is what "settled" waits on. */
    private var lastScroll = 0L

    /** True while this sends its own event, which comes back through here. */
    private var sending = false

    private var checkPosted = false
    private val check = Runnable { checkPosted = false; checkFocus() }
    private val recheck = Runnable { checkFocus() }

    fun attach() {
        // getAccessibilityDelegate is API 29. Below that nothing in this app
        // sets one on these parents, so there is nothing to keep.
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) previous = parent.accessibilityDelegate
        parent.accessibilityDelegate = this
    }

    fun detach() {
        handler.removeCallbacks(check)
        handler.removeCallbacks(recheck)
        // Put back what was there, unless something has since replaced this
        // (which can be checked only where the getter exists).
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.Q || parent.accessibilityDelegate === this) {
            parent.accessibilityDelegate = previous
        }
    }

    override fun onRequestSendAccessibilityEvent(
        host: ViewGroup,
        child: View,
        event: AccessibilityEvent,
    ): Boolean {
        // A screen reader, driven from a keyboard; see "When it acts".
        if (child === view && !sending && !view.isInTouchMode && accessibility?.isTouchExplorationEnabled == true) {
            if (event.eventType == AccessibilityEvent.TYPE_VIEW_SCROLLED) lastScroll = SystemClock.uptimeMillis()
            // One check per batch: Compose sends a batch's events in one go
            // on the main thread, so a posted check runs after the last one.
            if (!checkPosted) {
                checkPosted = true
                handler.post(check)
            }
        }
        val forward = previous
        return if (forward != null) {
            forward.onRequestSendAccessibilityEvent(host, child, event)
        } else {
            super.onRequestSendAccessibilityEvent(host, child, event)
        }
    }

    private fun checkFocus() {
        val now = SystemClock.uptimeMillis()
        val node = findFocused(root.semanticsOwner.unmergedRootSemanticsNode)
        val focused = node?.id
        if (focused != lastFocused) {
            lastFocused = focused
            target = focused
            targetSince = now
            announcements = 0
            scrolls = 0
        }
        val id = target ?: return
        if (node == null || now - targetSince > GIVE_UP_MS) {
            target = null
            return
        }
        handler.removeCallbacks(recheck)
        // Settled: Compose's own event and TalkBack's answer to it have had
        // time to land (TalkBack is another process), and the scroll that
        // brings the node into view has stopped. Both steps below wait for it.
        val quiet = now - maxOf(targetSince, lastScroll)
        val provider = view.accessibilityNodeProvider
        val virtualId = virtualViewId(id)
        // Compose's provider builds info only for a node in its current map,
        // which is the node TalkBack will ask for when it gets the event.
        val info = provider?.createAccessibilityNodeInfo(virtualId)
        if (info == null) {
            // Still off screen once the scroll has stopped: Compose's
            // bring-into-view was cut short; see "A scroll cut short".
            if (quiet >= SETTLE_MS && scrolls < MAX_SCROLLS) {
                scrolls++
                targetSince = now
                showOnScreen(node)
            }
            handler.postDelayed(recheck, RECHECK_MS)
            return
        }
        // Compose's findFocus (any type) answers with its accessibility-focused
        // node, which is TalkBack's. AccessibilityNodeInfo.equals compares
        // the source id and window, so this is "TalkBack is on the target".
        if (provider.findFocus(AccessibilityNodeInfo.FOCUS_ACCESSIBILITY) == info) {
            target = null
            return
        }
        // An event sent mid-scroll is the one TalkBack was seen ignoring.
        if (quiet < SETTLE_MS) {
            handler.postDelayed(recheck, SETTLE_MS - quiet)
            return
        }
        if (announcements >= MAX_ANNOUNCEMENTS) {
            target = null
            return
        }
        announcements++
        announce(virtualId)
        // Look again: the recheck either sees TalkBack on the node and stops,
        // or tries once more.
        targetSince = now
        handler.postDelayed(recheck, RETRY_MS)
    }

    /**
     * Scrolls the nearest scrolling ancestor until [node] is in view, through
     * the ancestor's ScrollBy semantics action.
     *
     * This is Compose 1.7.6's ACTION_SHOW_ON_SCREEN (performActionHelper),
     * which cannot be asked for here because Compose answers actions only for
     * a node in its current map, and an off-screen node is not in it. The
     * same steps: the viewport is the ancestor's bounds and the child its
     * position and size, both in root coordinates. Each axis moves by the
     * nearer edge's distance when the child is out on one side
     * ([scrollDelta]). The sign is flipped for a reversed scroll, and x is
     * flipped again under RTL. One level, as there: a scroll nested in a
     * scroll gets the inner one, and the next check sees what is left.
     */
    private fun showOnScreen(node: SemanticsNode) {
        var scroller = node.parent
        while (scroller != null && scroller.config.getOrNull(SemanticsActions.ScrollBy) == null) {
            scroller = scroller.parent
        }
        val scrollBy = scroller?.config?.getOrNull(SemanticsActions.ScrollBy)?.action ?: return
        val viewport = Rect(scroller.positionInRoot, scroller.size.toSize())
        val child = Rect(node.positionInRoot, node.size.toSize())
        var dx = scrollDelta(child.left - viewport.left, child.right - viewport.right)
        if (scroller.config.getOrNull(SemanticsProperties.HorizontalScrollAxisRange)?.reverseScrolling == true) dx = -dx
        if (node.layoutInfo.layoutDirection == LayoutDirection.Rtl) dx = -dx
        var dy = scrollDelta(child.top - viewport.top, child.bottom - viewport.bottom)
        if (scroller.config.getOrNull(SemanticsProperties.VerticalScrollAxisRange)?.reverseScrolling == true) dy = -dy
        if (dx != 0f || dy != 0f) scrollBy(dx, dy)
    }

    /**
     * How far to scroll one axis, from the child's leading and trailing edges
     * less the viewport's. Both negative: the child is out at the start, and
     * the smaller distance brings it fully in (or puts its start at the
     * viewport's start when it is the bigger). Both positive: the same at the
     * end. Opposite signs: it is in view, or spans the viewport, so leave it.
     */
    private fun scrollDelta(start: Float, end: Float): Float =
        if (sign(start) == sign(end)) (if (abs(start) < abs(end)) start else end) else 0f

    /** As Compose's createEvent builds it, and sent the way Compose sends it. */
    private fun announce(virtualId: Int) {
        val event = AccessibilityEvent.obtain(AccessibilityEvent.TYPE_VIEW_FOCUSED)
        event.isEnabled = true
        event.className = "android.view.View"
        event.packageName = view.context.packageName
        event.setSource(view, virtualId)
        sending = true
        try {
            parent.requestSendAccessibilityEvent(view, event)
        } finally {
            sending = false
        }
    }

    private fun virtualViewId(id: Int): Int =
        if (id == root.semanticsOwner.unmergedRootSemanticsNode.id) AccessibilityNodeProvider.HOST_VIEW_ID else id

    /**
     * The node whose own config says Focused. Only the focus target itself
     * sets it (focusable's semantics), so there is at most one.
     */
    private fun findFocused(node: SemanticsNode): SemanticsNode? {
        if (node.config.getOrNull(SemanticsProperties.Focused) == true) return node
        for (child in node.children) findFocused(child)?.let { return it }
        return null
    }

    private companion object {
        /** How often a node not yet in Compose's tree is looked for. */
        const val RECHECK_MS = 150L

        /**
         * Quiet time after the move, and after the last scroll, before this
         * speaks up. Longer than Compose's 100ms batch interval plus
         * TalkBack's round trip, so a move Compose announced and TalkBack
         * followed is seen as followed first.
         */
        const val SETTLE_MS = 300L

        /**
         * The wait before the retry. Long enough to outlast TalkBack's own
         * handling of a new window: a Modal opened from the keyboard puts
         * Compose's focus on its first control, and TalkBack, still settling
         * the window, ignored events sent 300ms and 600ms in.
         */
        const val RETRY_MS = 1200L

        /** Sends per move: the first, and one retry if TalkBack ignored it. */
        const val MAX_ANNOUNCEMENTS = 2

        /** Scrolls per move; a second covers a scroll nested in a scroll. */
        const val MAX_SCROLLS = 2

        /** How long a target may go without TalkBack reaching it. */
        const val GIVE_UP_MS = 3000L
    }
}
