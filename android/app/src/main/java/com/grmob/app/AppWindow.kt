package com.grmob.app

import androidx.activity.ComponentActivity
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
        activity.lifecycleScope.launch {
            activity.repeatOnLifecycle(Lifecycle.State.STARTED) {
                tracker.windowLayoutInfo(activity).collect { info ->
                    report(activity, runtime, info)
                }
            }
        }
    }

    private fun report(activity: ComponentActivity, runtime: GrMobRuntime, info: WindowLayoutInfo) {
        val density = activity.resources.displayMetrics.density
        // computeCurrentWindowMetrics, not the display's size: in split
        // screen or a freeform window the app has a fraction of the display,
        // and the window is what the tree is laid out in.
        val bounds = WindowMetricsCalculator.getOrCreate()
            .computeCurrentWindowMetrics(activity).bounds

        val payload = JSONObject()
            .put("width", bounds.width() / density)
            .put("height", bounds.height() / density)

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
