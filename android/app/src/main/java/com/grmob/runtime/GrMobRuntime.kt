package com.grmob.runtime

import android.os.Handler
import android.os.Looper
import java.util.concurrent.Executors

/**
 * The surface the Go side exposes, as seen from Kotlin.
 *
 * This mirrors the gomobile-generated `mobile.Mobile` class one-to-one (see
 * mobile/bridge.go); it exists as an interface so the runtime and any JVM
 * tests compile without the generated AAR on the classpath. The app module
 * provides the real implementation (GomobileBridge).
 */
interface GrMobBridge {
    fun renderInitial(): String
    fun triggerCallback(id: String): String
    fun triggerTextCallback(id: String, value: String): String
    fun triggerBoolCallback(id: String, value: Boolean): String
    fun triggerIntCallback(id: String, value: Long): String

    /** Registers the Go→native push target; called from a Go goroutine. */
    fun setListener(listener: (String) -> Unit)

    /**
     * Registers the sink for app→host system events: the transient chrome and
     * OS hand-offs that are deliberately not part of the view tree —
     * `core.ShowToast` and `core.OpenURL` today. See mobile/sysevents.go;
     * `name` is the event kind and `payload` its data as a JSON object.
     *
     * The callback arrives on whichever Go goroutine emitted the event — a
     * tap handler, a timer, a network response — never reliably the Android
     * main thread, so an implementation must post to it before touching UI.
     * The same contract [setListener] carries.
     */
    fun setSystemEventListener(listener: (String, String) -> Unit)

    /**
     * Reports a host→app event that answers no registered callback — the
     * audio player's status ticks today (see mobile/hostevents.go) — and
     * returns the patches of the render it caused, exactly like the
     * Trigger* calls. `name` is the event kind, `payload` its data as a
     * JSON object.
     */
    fun reportHostEvent(name: String, payload: String): String
}

/**
 * Wires the bridge to a TreeStore and owns the threading model.
 *
 * Patches reach Kotlin on two paths — the synchronous return value of a
 * Trigger* call, and asynchronous pushes from Go goroutines (timers, network,
 * State.Set off-thread). The Go side guarantees each render's diff is
 * delivered on exactly one path, in order; our side of the contract is to
 * apply payloads in arrival order on one thread. Both paths therefore funnel
 * into `main.post { store.applyPatches(...) }` — a Handler executes posts in
 * FIFO order, which *is* the ordering guarantee.
 *
 *   UI event ─▶ events executor ─▶ Mobile.trigger*() ─┐  (sync return)
 *                                                     ├─▶ main.post ─▶ TreeStore ─▶ recompose
 *   Go goroutine (timer/State.Set) ── push listener ──┘  (async push)
 *
 * Trigger* calls run on a dedicated single-thread executor, not the main
 * thread: a bridge call spans a full Go render pass and may briefly block on
 * the render mutex, and the single thread keeps events themselves ordered.
 */
class GrMobRuntime(private val bridge: GrMobBridge) {
    val store = TreeStore()

    private val main = Handler(Looper.getMainLooper())
    private val events = Executors.newSingleThreadExecutor { r ->
        Thread(r, "grmob-events").apply { isDaemon = true }
    }

    /**
     * What the initial mount cost, plus the bridge call that fed it, or null
     * until [start] has run.
     *
     * Kept rather than only logged because the numbers answer a question that
     * outlives one logcat session — how much of a cold launch is the tree — and
     * a caller that wants to put them on screen or into an instrumented test
     * should not have to scrape a log line for them. [Startup] is what prints
     * them; this is where they live.
     */
    var startupStats: MountStats? = null
        private set

    /** Time spent inside `bridge.renderInitial()` on the [start] call, in nanoseconds. */
    var startupBridgeNanos: Long = 0
        private set

    /** Mounts the initial tree and opens the push channel. Call once, on the main thread. */
    fun start() {
        // Three clock reads around the two calls that scale with the screen's
        // node count. They cost nothing here — this path runs once per process
        // — and they are the only place the split between "Go produced and
        // handed over the payload" and "we turned it into a tree" is visible.
        // See Startup for what the split is for.
        val t0 = System.nanoTime()
        val json = bridge.renderInitial()
        val t1 = System.nanoTime()
        val stats = store.mount(json)
        startupBridgeNanos = t1 - t0
        startupStats = stats
        Startup.report(bridgeNanos = t1 - t0, stats = stats)
        // Listener attaches after the initial mount so a pre-mount push can
        // never race tree construction; Go re-flushes pending changes on
        // attach, so nothing that happened in between is lost.
        bridge.setListener { patches -> main.post { store.applyPatches(patches) } }
    }

    fun click(callbackId: String) =
        dispatch { bridge.triggerCallback(callbackId) }

    fun textChanged(callbackId: String, value: String) =
        dispatch { bridge.triggerTextCallback(callbackId, value) }

    fun toggled(callbackId: String, value: Boolean) =
        dispatch { bridge.triggerBoolCallback(callbackId, value) }

    fun intChanged(callbackId: String, value: Int) =
        dispatch { bridge.triggerIntCallback(callbackId, value.toLong()) }

    /**
     * Delivers a host event (a player status tick, say) to Go on the same
     * serial executor as UI events, so it can never interleave with one, and
     * applies the patches it produced the same way. Safe to call from any
     * thread — the executor is the serialization point.
     */
    fun hostEvent(name: String, payload: String) =
        dispatch { bridge.reportHostEvent(name, payload) }

    private fun dispatch(call: () -> String) {
        events.execute {
            val patches = call()
            main.post { store.applyPatches(patches) }
        }
    }
}
