package com.grmob.app

import androidx.activity.ComponentActivity
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.lifecycleScope
import androidx.lifecycle.repeatOnLifecycle
import androidx.window.layout.FoldingFeature
import androidx.window.layout.WindowInfoTracker
import androidx.window.layout.WindowLayoutInfo
import androidx.window.layout.WindowMetricsCalculator
import com.grmob.runtime.GrMobRuntime
import kotlinx.coroutines.launch
import org.json.JSONObject

/**
 * The Android half of grmob's window event: tells Go how big the window is
 * and whether a fold crosses it, through the "window" host event
 * (core/window.go).
 *
 *   WindowMetricsCalculator ──▶ width, height          (px ÷ density → dp)
 *   FoldingFeature          ──▶ fold { state, orientation, separating,
 *                                      occluding, x, y, width, height }
 *   root WindowInsetsCompat ──▶ insets { top, bottom, left, right }
 *
 * # Why Jetpack WindowManager
 *
 * The platform has no fold API of its own below Android 15 — hinge
 * geometry lives in each OEM's window extensions — and WindowManager is the
 * library that normalizes them, including the emulator's foldable profiles.
 * On a device with no extensions (every non-foldable) windowLayoutInfo
 * simply emits no features, which is the "no fold" report.
 *
 * # When it reports
 *
 * windowLayoutInfo emits once when collection starts and again on every
 * change to the features: a hinge bending from flat to half-opened is an
 * emission with no configuration change. Folding or unfolding the device
 * *is* a configuration change (screenSize), so the Activity is recreated,
 * [attach] runs on the new instance and collection starts over — which is
 * also how a resize in multi-window mode arrives. Go dedupes a window it
 * already has (core.ReceiveWindow), so the recreation's repeat costs a
 * bridge call and nothing else.
 *
 * Collection is scoped to STARTED with repeatOnLifecycle: the tracker holds
 * a listener on the window extensions, and a stopped Activity has no window
 * a report could describe. Coming back to STARTED re-emits the current
 * layout, so a posture change made while backgrounded still lands.
 *
 * The insets have their own trigger, because they move without the fold or
 * the window doing anything — a rotation that carries the cutout to the
 * other edge, a gesture-nav bar that changes height. The natural hook,
 * setOnApplyWindowInsetsListener, is *not* used: it replaces a view's
 * inset handling rather than observing it, and on the decor view that is
 * the chain edge-to-edge and Compose's own WindowInsets depend on.
 * OnGlobalLayoutListener is additive, consumes nothing, and fires on the
 * layout pass an inset change causes anyway; the reported set is remembered
 * so the frequent calls turn into a comparison and nothing more.
 *
 * # Which insets
 *
 * systemBars + displayCutout, which is WindowInsets.safeDrawing minus the
 * IME — the same set Renderer.kt's SafeArea node applies, on purpose, so
 * the numbers Go reads describe the edge its own SafeArea keeps content
 * off. The keyboard is deliberately not in them: it is a transient overlay
 * with its own story (core/keyboard.go), not an edge of the window.
 *
 * # Units and coordinates
 *
 * Both the window bounds and FoldingFeature.bounds are in window pixels with
 * the origin at the window's top-left — under the status bar, since
 * MainActivity is edge to edge. Dividing by density puts them in dp, which
 * is the unit Go lays out in, and changes nothing else: core documents the
 * same window coordinate space for every host.
 */
object AppWindow {
    /**
     * Starts reporting for [activity]. Call from onCreate after the runtime
     * exists; the coroutine is tied to the Activity's lifecycle, so a
     * recreated Activity's collection replaces the destroyed one's rather
     * than stacking beside it.
     */
    fun attach(activity: ComponentActivity, runtime: GrMobRuntime) {
        val tracker = WindowInfoTracker.getOrCreate(activity)
        // The two triggers share what the other one knows. Both run on the
        // main thread — collect resumes on the lifecycleScope's Main
        // dispatcher and a layout pass is by definition on it — so plain
        // vars need no synchronization, and both die with this Activity
        // rather than living on the object across a recreation.
        var latest: WindowLayoutInfo? = null
        var reported: Insets? = null

        activity.lifecycleScope.launch {
            activity.repeatOnLifecycle(Lifecycle.State.STARTED) {
                tracker.windowLayoutInfo(activity).collect { info ->
                    latest = info
                    val now = currentInsets(activity)
                    reported = now
                    report(activity, runtime, info, now)
                }
            }
        }

        activity.window.decorView.viewTreeObserver.addOnGlobalLayoutListener {
            val now = currentInsets(activity)
            if (now == reported) return@addOnGlobalLayoutListener
            reported = now
            // latest is null only before the first emission, which is the
            // one case where the size may not be measured yet either; the
            // collector above will report both together in a moment.
            latest?.let { report(activity, runtime, it, now) }
        }
    }

    /** The window's safe-drawing edges in dp. See "Which insets" above. */
    private fun currentInsets(activity: ComponentActivity): Insets {
        val density = activity.resources.displayMetrics.density
        val root = ViewCompat.getRootWindowInsets(activity.window.decorView)
            ?: return Insets(0.0, 0.0, 0.0, 0.0)
        val i = root.getInsets(
            WindowInsetsCompat.Type.systemBars() or WindowInsetsCompat.Type.displayCutout(),
        )
        return Insets(
            top = i.top / density.toDouble(),
            bottom = i.bottom / density.toDouble(),
            left = i.left / density.toDouble(),
            right = i.right / density.toDouble(),
        )
    }

    /**
     * One report's insets, in dp. A data class so the "did they change?"
     * test above is a value comparison, which is the same thing core's
     * record does with the whole Window.
     */
    private data class Insets(
        val top: Double,
        val bottom: Double,
        val left: Double,
        val right: Double,
    )

    private fun report(
        activity: ComponentActivity,
        runtime: GrMobRuntime,
        info: WindowLayoutInfo,
        insets: Insets,
    ) {
        val density = activity.resources.displayMetrics.density
        // computeCurrentWindowMetrics, not the display's size: in split
        // screen or a freeform window the app has a fraction of the display,
        // and the window is what the tree is laid out in.
        val bounds = WindowMetricsCalculator.getOrCreate()
            .computeCurrentWindowMetrics(activity).bounds

        val payload = JSONObject()
            .put("width", bounds.width() / density)
            .put("height", bounds.height() / density)
            .put(
                "insets",
                JSONObject()
                    .put("top", insets.top)
                    .put("bottom", insets.bottom)
                    .put("left", insets.left)
                    .put("right", insets.right),
            )

        // At most one fold is reported. Every shipping foldable has one
        // hinge, and core's record has room for one; a device that someday
        // reports two gets its first, which still keeps First off a crease.
        val fold = info.displayFeatures.filterIsInstance<FoldingFeature>().firstOrNull()
        if (fold != null) {
            val b = fold.bounds
            payload.put(
                "fold",
                JSONObject()
                    .put("state", if (fold.state == FoldingFeature.State.HALF_OPENED) "half_opened" else "flat")
                    .put(
                        "orientation",
                        if (fold.orientation == FoldingFeature.Orientation.HORIZONTAL) "horizontal" else "vertical",
                    )
                    // isSeparating is WindowManager's own judgement — true
                    // for any half-opened hinge and for a flat seam between
                    // two panels — and is passed through rather than
                    // re-derived, so Go and Android agree on it.
                    .put("separating", fold.isSeparating)
                    .put("occluding", fold.occlusionType == FoldingFeature.OcclusionType.FULL)
                    .put("x", b.left / density)
                    .put("y", b.top / density)
                    .put("width", b.width() / density)
                    .put("height", b.height() / density),
            )
        }
        // hostEvent is safe from any thread and serializes with UI events on
        // the runtime's executor.
        runtime.hostEvent("window", payload.toString())
    }
}
