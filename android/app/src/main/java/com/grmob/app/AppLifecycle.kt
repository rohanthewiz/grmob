package com.grmob.app

import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.ProcessLifecycleOwner
import com.grmob.runtime.GrMobRuntime
import org.json.JSONObject

/**
 * The Android half of grmob's app-lifecycle event: tells Go whether the app
 * is on screen, through the "lifecycle" host event (core/lifecycle.go).
 *
 *   ON_RESUME ──▶ "active"
 *   ON_PAUSE  ──▶ "inactive"
 *   ON_STOP   ──▶ "background"
 *
 * The source is [ProcessLifecycleOwner], not the Activity. An Activity's
 * own callbacks fire on every configuration change — a rotation is an
 * onStop on the old instance followed by an onStart on the new one — and an
 * app subscribed to reconnect on resume would tear down and redial its
 * connection every time the phone turned. The process owner reports the
 * lifecycle of the whole app: ON_PAUSE and ON_STOP are delayed by a short
 * grace period and cancelled if another Activity of the app takes over,
 * so they mean "the user left", which is the only thing Go wants to know.
 * ON_START is deliberately not mapped — ON_RESUME follows it immediately
 * and is the one that means input has come back — and ON_CREATE / ON_DESTROY
 * never fire for the process owner.
 *
 * The observer is registered once for the process. The Activity is
 * recreated on rotation and builds a fresh [GrMobRuntime] each time, so
 * [attach] only swaps which runtime the next report goes through; a second
 * observer would deliver every transition twice. Go dedupes repeats of the
 * current state anyway (core.ReceiveLifecycle), so the guard is about not
 * doing the bridge work twice, not about correctness.
 */
object AppLifecycle {
    @Volatile private var runtime: GrMobRuntime? = null
    private var observing = false

    /**
     * Routes lifecycle transitions to [runtime]. Call from the Activity's
     * onCreate, after the runtime exists; main thread only, like the
     * lifecycle it observes.
     */
    fun attach(runtime: GrMobRuntime) {
        this.runtime = runtime
        if (observing) return
        observing = true
        ProcessLifecycleOwner.get().lifecycle.addObserver(
            LifecycleEventObserver { _, event ->
                val state = when (event) {
                    Lifecycle.Event.ON_RESUME -> "active"
                    Lifecycle.Event.ON_PAUSE -> "inactive"
                    Lifecycle.Event.ON_STOP -> "background"
                    else -> return@LifecycleEventObserver
                }
                report(state)
            }
        )
    }

    private fun report(state: String) {
        // The payload is one key, the contract every host writes; Go reads
        // it in core.receiveLifecycle. JSONObject rather than string
        // concatenation so the shape stays a real object if a second key
        // is ever added.
        val payload = JSONObject().put("state", state).toString()
        // hostEvent is safe from any thread and serializes with UI events
        // on the runtime's executor, so a transition can never interleave
        // with a tap that is mid-flight.
        runtime?.hostEvent("lifecycle", payload)
    }
}
