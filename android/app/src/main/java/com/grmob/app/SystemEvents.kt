package com.grmob.app

import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Handler
import android.os.Looper
import android.util.Log
import android.widget.Toast
import com.grmob.runtime.GrMobBridge
import org.json.JSONObject

/**
 * The Android half of grmob's system-event channel: the app→host events that
 * are deliberately not part of the view tree, because what they reach is the
 * platform itself rather than anything the reconciler could diff.
 *
 *   core.ShowToast  ──▶ "toast"     ──▶ android.widget.Toast
 *   core.OpenURL    ──▶ "open_url"  ──▶ Intent(ACTION_VIEW)
 *
 * Before this existed the events were emitted into a nil Go handler and
 * vanished on both natives — only the WASM host had a sink — so an app
 * calling ShowToast worked in the browser preview and silently did nothing on
 * a phone.
 *
 * Unknown event names are dropped, matching the contract every other host
 * applies: a newer app on an older shell degrades to silence rather than to a
 * crash.
 */
object SystemEvents {
    private const val TAG = "GrMobSystemEvents"

    /**
     * Wires Go's event stream to this process's UI.
     *
     * Call before [com.grmob.runtime.GrMobRuntime.start] so an event emitted
     * during the very first render pass has somewhere to land.
     *
     * @param context any Context; only its application context is retained,
     *   so an Activity passed here is not leaked past its own lifetime. The
     *   consequence is that [Intent.FLAG_ACTIVITY_NEW_TASK] is mandatory
     *   below — an application context has no task of its own to launch into.
     */
    fun attach(context: Context, bridge: GrMobBridge) {
        val appContext = context.applicationContext
        val main = Handler(Looper.getMainLooper())
        // The callback runs on the Go goroutine that emitted the event. Both
        // actions below touch the UI (a Toast must be shown from a Looper
        // thread; startActivity from an arbitrary thread is unreliable), so
        // every event is posted to the main thread — the same hop
        // GrMobRuntime makes for patches, and posts run FIFO, so a toast
        // emitted before a navigation still appears first.
        bridge.setSystemEventListener { name, payload ->
            main.post { dispatch(appContext, name, payload) }
        }
    }

    private fun dispatch(context: Context, name: String, payload: String) {
        val data = try {
            JSONObject(payload)
        } catch (e: org.json.JSONException) {
            Log.w(TAG, "malformed payload for system event '$name'", e)
            return
        }
        when (name) {
            "toast" -> showToast(context, data)
            "open_url" -> openUrl(context, data)
        }
    }

    private fun showToast(context: Context, data: JSONObject) {
        val message = data.optString("message")
        if (message.isEmpty()) return
        // Go sends milliseconds; Android offers two fixed buckets and honors
        // nothing in between (a custom duration has been ignored since API
        // 30). 3000ms is the midpoint between SHORT's ~2s and LONG's ~3.5s,
        // so the default 2000 lands on SHORT and an app that deliberately
        // asked for longer gets LONG.
        val duration = if (data.optInt("duration", 2000) > 3000) {
            Toast.LENGTH_LONG
        } else {
            Toast.LENGTH_SHORT
        }
        // A styled toast's "style" key is ignored: since API 30 the platform
        // owns the toast's appearance and setView is a no-op. Honoring it
        // would mean drawing our own overlay, which is a view-tree feature
        // and belongs in the renderer, not here.
        Toast.makeText(context, message, duration).show()
    }

    private fun openUrl(context: Context, data: JSONObject) {
        val url = data.optString("url")
        if (url.isEmpty()) return
        val intent = Intent(Intent.ACTION_VIEW, Uri.parse(url))
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        try {
            context.startActivity(intent)
        } catch (e: ActivityNotFoundException) {
            // No app claims the scheme — a `tel:` link on a tablet with no
            // dialer, or a device with no browser. Logged and swallowed:
            // core.OpenURL is fire-and-forget by contract and a system event
            // has no return channel, so there is nothing to report back to Go.
            Log.w(TAG, "no activity can open $url", e)
        }
    }
}
