package com.grmob.app

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.os.Build
import android.util.Log
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import org.json.JSONObject

/**
 * The Android half of core's local notifications (core/notifications.go).
 *
 *   core.PostNotification   ──▶ "notification" {command: "post", id, title, body}
 *   core.CancelNotification ──▶ "notification" {command: "cancel", id}
 *   core.OnNotificationTap  ◀── "notification_tap" {id}   (from MainActivity)
 *
 * # One channel
 *
 * Android 8+ requires every notification to belong to a channel, and the
 * channel — not the app — is what the user mutes, so its name is what they
 * see in Settings. One channel, created at attach, because core has no
 * vocabulary for categories: a second channel would be a decision this shell
 * made on the app's behalf. An app that needs several extends the shell.
 *
 * # The id is the tag
 *
 * Go ids are strings; NotificationManager keys by (tag, int). The Go id is the
 * tag and the int is a constant, so posting under an existing id replaces that
 * banner and cancel(tag, NOTIFY_ID) takes exactly it down — no id→int table to
 * keep, and none to lose across a process restart.
 *
 * # The tap
 *
 * The content intent launches MainActivity with [EXTRA_ID]; MainActivity
 * reports it (see [tapId]) from onCreate for a cold launch and onNewIntent for
 * a running app, the same two paths a deep link takes.
 */
object Notifications {
    /** The intent extra that carries a tapped notification's Go id. */
    const val EXTRA_ID = "com.grmob.notification.id"

    private const val TAG = "GrMobNotifications"
    private const val CHANNEL_ID = "grmob"
    private const val NOTIFY_ID = 1

    private var appContext: Context? = null

    fun attach(context: Context) {
        val app = context.applicationContext
        appContext = app
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            // Idempotent: creating an existing channel updates its name only,
            // never the importance the user may have changed in Settings.
            val channel = NotificationChannel(
                CHANNEL_ID,
                "Notifications",
                NotificationManager.IMPORTANCE_HIGH,
            )
            app.getSystemService(NotificationManager::class.java)?.createNotificationChannel(channel)
        }
    }

    fun handle(data: JSONObject) {
        val id = data.optString("id")
        if (id.isEmpty()) return
        when (data.optString("command")) {
            "post" -> post(id, data.optString("title"), data.optString("body"))
            "cancel" -> appContext?.let { NotificationManagerCompat.from(it).cancel(id, NOTIFY_ID) }
        }
    }

    private fun post(id: String, title: String, body: String) {
        val context = appContext ?: return
        val manager = NotificationManagerCompat.from(context)
        // False when the user has not granted POST_NOTIFICATIONS (13+) or has
        // turned the app's notifications off (any release). Posting anyway is
        // a silent no-op on the platform, but checking first keeps notify()'s
        // SecurityException path for the case that is genuinely a bug.
        if (!manager.areNotificationsEnabled()) return

        val tap = Intent(context, MainActivity::class.java)
            .putExtra(EXTRA_ID, id)
            // An application context has no task; NEW_TASK is mandatory. The
            // manifest's singleTop delivers this to a running MainActivity's
            // onNewIntent instead of stacking a second one.
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP)
        // A request code per id: PendingIntents that differ only in extras are
        // the *same* PendingIntent to the system, so without it every banner's
        // tap would carry whichever id was posted last. IMMUTABLE is required
        // from API 31 and nothing here needs the intent filled in later.
        val pending = PendingIntent.getActivity(
            context,
            id.hashCode(),
            tap,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )

        val notification = NotificationCompat.Builder(context, CHANNEL_ID)
            .setSmallIcon(smallIcon(context))
            .setContentTitle(title.ifEmpty { null })
            .setContentText(body.ifEmpty { null })
            .setStyle(NotificationCompat.BigTextStyle().bigText(body))
            .setPriority(NotificationCompat.PRIORITY_HIGH) // pre-O heads-up
            .setContentIntent(pending)
            .setAutoCancel(true)
            .build()
        try {
            manager.notify(id, NOTIFY_ID, notification)
        } catch (e: SecurityException) {
            // The permission was revoked between the check above and here.
            Log.w(TAG, "notification $id refused", e)
        }
    }

    /**
     * The status-bar icon. It must be a monochrome drawable — the system tints
     * it — and an app icon is usually neither, so an adopting app supplies
     * `res/drawable/ic_notification`. Until it does, a platform drawable keeps
     * the post from failing outright: a notification with no valid small icon
     * is rejected by the system, not drawn with a blank one.
     */
    private fun smallIcon(context: Context): Int {
        val own = context.resources.getIdentifier("ic_notification", "drawable", context.packageName)
        return if (own != 0) own else android.R.drawable.ic_dialog_info
    }

    /**
     * The Go id a launch intent carries when it came from tapping one of these
     * notifications, or null. MainActivity forwards it as "notification_tap".
     */
    fun tapId(intent: Intent?): String? =
        intent?.getStringExtra(EXTRA_ID)?.takeIf { it.isNotEmpty() }
}
