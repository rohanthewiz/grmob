package com.grmob.app

import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.util.Log
import org.json.JSONObject

/**
 * The Android half of core.Share (core/share.go).
 *
 *   core.Share ──▶ "share" {id, text, url}
 *              ◀── host event "share" {id, ok}
 *
 * An ACTION_SEND of text/plain through the system chooser (Intent.createChooser),
 * which is the share sheet: the chooser lists every app that takes plain text,
 * with the platform's direct-share targets on top.
 *
 * ok is whether the chooser started. The chooser does not report what the
 * person picked or whether they backed out (that needs an IntentSender
 * callback, and a dismissal is not something a caller should fall back on
 * anyway; see "What ok means" in share.go). It answers false only when
 * startActivity throws, which with a chooser means no activity at all.
 *
 * Every request is answered, because Go holds the caller's done until its id
 * comes back. Runs on the main thread, like every handler [SystemEvents]
 * dispatches to.
 */
object Share {
    private const val TAG = "GrMobShare"
    private var appContext: Context? = null
    private var report: ((String, String) -> Unit)? = null

    /** Retains the application context only, and the host-event reporter. */
    fun attach(context: Context, out: (String, String) -> Unit) {
        appContext = context.applicationContext
        report = out
    }

    fun handle(data: JSONObject) {
        val id = data.optString("id")
        if (id.isEmpty()) return
        val context = appContext
        val text = data.optString("text")
        val url = data.optString("url")
        if (context == null || (text.isEmpty() && url.isEmpty())) {
            send(id, ok = false)
            return
        }
        // ACTION_SEND has one text slot, so the two join with a newline: the
        // link on its own line is what a messenger turns into a preview.
        // EXTRA_SUBJECT is left out; a mail app would use it, and core.Share
        // has no subject to give.
        val body = listOf(text, url).filter { it.isNotEmpty() }.joinToString("\n")
        val send = Intent(Intent.ACTION_SEND)
            .setType("text/plain")
            .putExtra(Intent.EXTRA_TEXT, body)
        // A null title lets the platform caption the chooser in the device's
        // language ("Share"), which a fixed English string here would not.
        // NEW_TASK because the context is the application's, which has no
        // task to launch into (see SystemEvents.attach).
        val chooser = Intent.createChooser(send, null)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        val ok = try {
            context.startActivity(chooser)
            true
        } catch (e: ActivityNotFoundException) {
            Log.w(TAG, "no activity can show the share chooser", e)
            false
        }
        send(id, ok)
    }

    private fun send(id: String, ok: Boolean) {
        val payload = JSONObject()
            .put("id", id)
            .put("ok", ok)
        report?.invoke("share", payload.toString())
    }
}
