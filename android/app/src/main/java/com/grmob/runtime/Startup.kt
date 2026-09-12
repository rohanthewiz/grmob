package com.grmob.runtime

import android.os.Process
import android.os.SystemClock
import android.util.Log

/**
 * Where a cold launch's time goes, on the stages that scale with the tree.
 *
 * # Why this exists
 *
 * `android/device/launch.sh` established that the tutorial's contents screen
 * cost 2516ms of a 5045ms cold launch beyond the same binary with the cards
 * taken off the screen, and that `core.List` — Compose composing four rows
 * instead of 49 — accounted for only 1017ms of the screen's cost. The other
 * half was everything that happens to 423,472 bytes of JSON *before* Compose
 * gets a say, and laziness cannot touch any of it because the whole tree
 * crosses the bridge either way.
 *
 * "Everything that happens to those bytes" was a phrase, not a measurement.
 * The proposed fix — windowing on the Go side, so the initial payload carries
 * a slice of the rows rather than all of them — is a protocol change, and its
 * size is exactly the size of these stages:
 *
 *     ┌ process start ─────────────────────────────────────────── first frame ┐
 *     │ zygote fork, classload,  │ bridge │ parse │ build │ Compose + measure │
 *     │ Go runtime init, Activity│        │       │       │ + layout + draw   │
 *     └──────────────────────────┴────────┴───────┴───────┴───────────────────┘
 *                                 └──── what windowing would shrink ────┘ ┆
 *                                                       already lazy ─────┘
 *
 *   bridge   Mobile.renderInitial(): the Go render pass, the marshal, and the
 *            gomobile crossing that copies the result into a Java String
 *            (UTF-8 → UTF-16 for every one of those bytes).
 *   parse    org.json turning the string into JSONObject/JSONArray.
 *   build    GrMobNode.parse walking that into the snapshot-state tree
 *            Compose reads — one mutableStateOf per node, plus a
 *            LinkedHashMap per props object.
 *
 * The three are reported separately because they would respond differently to
 * a fix: windowing shrinks all three in proportion to nodes sent, while a
 * different lever (a binary payload, say, or parsing straight into nodes) would
 * move only one of them. Knowing which is the biggest is the point of printing
 * them at all.
 *
 * # What it said the first time, which was not what anyone expected
 *
 * Five cold launches of that screen: bridge 17ms, parse 1666ms, build 427ms.
 * Neither Go nor the FFI was in it — the JNI crossing of 423KB is a rounding
 * error — and windowing was not built, because the same instrument said the
 * payload was 92.4% core.Style fields sitting at their zero values. Those
 * fields are `,omitzero` now (see the note above core.Style), the payload is
 * 53,408 bytes, and the same five launches read bridge 13ms, parse 249ms,
 * build 185ms, for 1320ms off the whole launch.
 *
 * A later pass found the tags had stopped one level too high — a present
 * Padding still wrote all six of core.EdgeInsets' untagged ints — and took the
 * payload to 51,242. Five launches each way say that bought nothing this
 * instrument can see: parse+build 379.8ms -> 377.9ms against run-to-run
 * spreads of 17-52ms. Which is the other thing these clocks are for. A 4%
 * payload cut predicts ~8ms, 8ms is under the floor here, and knowing where
 * the floor is is what stops the next 4% being reported as a win.
 *
 * Two things about that are worth keeping. The parse is still the largest of
 * the three, so this stays the right instrument for the next round. And the
 * identical payload costs **6ms** to parse and build on the iOS simulator
 * (LiveMapUITests records it): the gap between the hosts is org.json, not the
 * hardware, which is why iOS never had this problem to find.
 *
 * # Why it is off unless asked for
 *
 * The measurement itself is four `System.nanoTime()` reads on a path that runs
 * once per process, so its cost is not the concern — the log line is. This is
 * library code in every app built on the runtime, and an app that prints a
 * timing line to logcat on every launch has made a decision on its author's
 * behalf. So the line is gated on the platform's own per-tag switch:
 *
 *     adb shell setprop log.tag.GrMobStartup DEBUG     # then launch
 *     adb logcat -s GrMobStartup:D
 *
 * The property survives until reboot and is read fresh on each launch, which is
 * what makes it usable for an A/B: set it once, run both arms.
 *
 * The stage timings are taken whether or not anyone is listening, because
 * [Log.isLoggable] is itself a property read and doing it once at the end is
 * cheaper than doing it three times around the stages — and because a caller
 * that wants the numbers for something other than logging (a test, a future
 * on-screen readout) gets them from [GrMobRuntime.startupStats] regardless.
 */
internal object Startup {
    /** The logcat tag, and the tag the `log.tag.` property names. */
    const val TAG = "GrMobStartup"

    /**
     * Prints one line attributing the launch, if the tag is enabled.
     *
     * @param bridgeNanos time inside `bridge.renderInitial()` — Go's render
     *   and marshal plus the gomobile string crossing, which cannot be split
     *   from this side.
     * @param stats the mount's two stages, or null if the payload was not a
     *   tree (TreeStore.mount has already logged that case in full).
     */
    fun report(bridgeNanos: Long, stats: MountStats?) {
        if (!Log.isLoggable(TAG, Log.DEBUG)) return

        // Process.getStartUptimeMillis and SystemClock.uptimeMillis are the
        // same clock (uptime, excluding deep sleep), so the difference is the
        // real elapsed launch so far. This is the only number here that covers
        // work this file did not time — zygote fork, class loading, the Go
        // runtime's own init, and Activity.onCreate up to the mount — and it
        // is what makes the printed stages a fraction of something rather than
        // three numbers in isolation.
        val sinceProcessStart = SystemClock.uptimeMillis() - Process.getStartUptimeMillis()

        if (stats == null) {
            Log.d(TAG, "mount abandoned: payload was not a tree " +
                "(bridge ${ms(bridgeNanos)}ms, ${sinceProcessStart}ms since process start)")
            return
        }

        // One line, not four: logcat interleaves, and an A/B is read by
        // eye off two runs. The trailing total is the sum of the stages, so
        // the reader can see at a glance how much of `since process start`
        // the tree accounts for and how much is the platform's own launch.
        val treeNanos = bridgeNanos + stats.parseNanos + stats.buildNanos
        Log.d(TAG, "initial tree: ${stats.chars} chars  " +
            "bridge ${ms(bridgeNanos)}  parse ${ms(stats.parseNanos)}  " +
            "build ${ms(stats.buildNanos)}  = ${ms(treeNanos)} ms of " +
            "${sinceProcessStart}ms since process start")
    }

    /**
     * Nanoseconds as milliseconds with one decimal.
     *
     * String.format rather than integer division because the stages differ by
     * two orders of magnitude on the screens this was built for — a 0ms column
     * next to a 1400ms one reads as "not measured" rather than "fast".
     */
    private fun ms(nanos: Long): String = String.format("%.1f", nanos / 1_000_000.0)
}
