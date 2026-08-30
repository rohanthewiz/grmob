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

    /** Mounts the initial tree and opens the push channel. Call once, on the main thread. */
    fun start() {
        store.mount(bridge.renderInitial())
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

    private fun dispatch(call: () -> String) {
        events.execute {
            val patches = call()
            main.post { store.applyPatches(patches) }
        }
    }
}
