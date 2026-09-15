package com.grmob.app

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import org.json.JSONObject

/**
 * The Android half of core's clipboard (core/clipboard.go).
 *
 *   core.WriteClipboard ──▶ "clipboard" {command: "write", text}
 *   core.ReadClipboard  ──▶ "clipboard" {command: "read", id}
 *                       ◀── host event "clipboard" {id, text, ok}
 *
 * Every read is answered, including on failure, because Go holds the
 * caller's callback until the id comes back and has no timeout to fall back
 * on (see the lifetime note in clipboard.go).
 *
 * Runs on the main thread: [SystemEvents] posts every event there before
 * dispatching, and ClipboardManager is documented as main-thread-safe.
 */
object Clipboard {
    private var manager: ClipboardManager? = null
    private var appContext: Context? = null
    private var report: ((String, String) -> Unit)? = null

    /** Retains the application context only, and the host-event reporter. */
    fun attach(context: Context, out: (String, String) -> Unit) {
        appContext = context.applicationContext
        manager = context.getSystemService(Context.CLIPBOARD_SERVICE) as? ClipboardManager
        report = out
    }

    fun handle(data: JSONObject) {
        when (data.optString("command")) {
            "write" -> write(data.optString("text"))
            "read" -> read(data.optString("id"))
        }
    }

    private fun write(text: String) {
        // The label is what Android 13's clipboard editor overlay shows; the
        // app has no better name for an arbitrary copy than its own.
        // Android 13+ also draws its own "Copied" confirmation for every
        // write, which is why grmob does not toast one.
        manager?.setPrimaryClip(ClipData.newPlainText("grmob", text))
    }

    private fun read(id: String) {
        if (id.isEmpty()) return
        val clipboard = manager
        val context = appContext
        if (clipboard == null || context == null) {
            send(id, "", ok = false)
            return
        }
        // Since Android 10 only the app with input focus may read, and a
        // background read returns null rather than throwing. That null is
        // indistinguishable from an empty clipboard, so both answer ok=true
        // with "": a Paste button is only ever pressed by a focused app, which
        // is the one case that reads successfully.
        //
        // coerceToText rather than item.text: a copied URI or intent has no
        // text field but does have a textual form, and that is what a user
        // pasting it expects to see.
        val item = clipboard.primaryClip?.takeIf { it.itemCount > 0 }?.getItemAt(0)
        val text = item?.coerceToText(context)?.toString() ?: ""
        send(id, text, ok = true)
    }

    private fun send(id: String, text: String, ok: Boolean) {
        val payload = JSONObject()
            .put("id", id)
            .put("text", text)
            .put("ok", ok)
        report?.invoke("clipboard", payload.toString())
    }
}
