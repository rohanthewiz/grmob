package com.grmob.app

import android.app.AlarmManager
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.BroadcastReceiver
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
 *   core.PostNotification   ──▶ "notification" {command: "post", id, title, body, at?}
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
 *
 * # Scheduled posts ("at")
 *
 * ```
 *   handle ── at > now ──▶ AlarmManager ─ fires ─▶ NotificationAlarmReceiver
 *                          (PendingIntent keyed        │ (maybe a fresh process:
 *                           by the Go id)              ▼  no Go, no attach)
 *                                                  post(context, …)
 * ```
 *
 * The receiver posts with its own context, so nothing here may depend on
 * [attach] having run: the channel is (re)created by [post] itself.
 *
 * Exact where [AlarmManager.canScheduleExactAlarms] allows it (always below
 * 12), via setExactAndAllowWhileIdle, which fires in Doze too. Otherwise
 * setAndAllowWhileIdle, which Doze may push back by minutes — late rather
 * than never. Cancel removes the pending alarm as well as a shown banner.
 * Alarms are forgotten across a reboot and a force stop; re-arming them needs
 * a BOOT_COMPLETED receiver and a store of what was scheduled, which the shell
 * does not keep (core/notifications.go documents the loss).
 */
object Notifications {
    /** The intent extra that carries a tapped notification's Go id. */
    const val EXTRA_ID = "com.grmob.notification.id"
    private const val EXTRA_TITLE = "com.grmob.notification.title"
    private const val EXTRA_BODY = "com.grmob.notification.body"

    private const val TAG = "GrMobNotifications"
    private const val CHANNEL_ID = "grmob"
    private const val NOTIFY_ID = 1

    private var appContext: Context? = null

    fun attach(context: Context) {
        val app = context.applicationContext
        appContext = app
        ensureChannel(app)
    }

    private fun ensureChannel(app: Context) {
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
        val context = appContext ?: return
        when (data.optString("command")) {
            "post" -> {
                val title = data.optString("title")
                val body = data.optString("body")
                // Absent means now; Go leaves the key off for a past time.
                val at = data.optLong("at", 0L)
                if (at > System.currentTimeMillis()) {
                    schedule(context, id, title, body, at)
                } else {
                    post(context, id, title, body)
                }
            }
            "cancel" -> {
                alarmManager(context)?.cancel(alarmIntent(context, id, "", ""))
                NotificationManagerCompat.from(context).cancel(id, NOTIFY_ID)
            }
        }
    }

    private fun alarmManager(context: Context): AlarmManager? =
        context.getSystemService(AlarmManager::class.java)

    /**
     * The broadcast an alarm delivers. Keyed like the tap intent — request
     * code from the id, and the id as data so two ids whose hashes collide are
     * still two PendingIntents (filterEquals compares data, not extras) —
     * which is what makes a re-post under an id replace its alarm and a
     * cancel find it. FLAG_UPDATE_CURRENT carries the newest title and body.
     */
    private fun alarmIntent(context: Context, id: String, title: String, body: String): PendingIntent {
        val intent = Intent(context, NotificationAlarmReceiver::class.java)
            .setData(android.net.Uri.fromParts("grmob-notification", id, null))
            .putExtra(EXTRA_ID, id)
            .putExtra(EXTRA_TITLE, title)
            .putExtra(EXTRA_BODY, body)
        return PendingIntent.getBroadcast(
            context,
            id.hashCode(),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
    }

    private fun schedule(context: Context, id: String, title: String, body: String, at: Long) {
        val alarms = alarmManager(context) ?: return
        val pending = alarmIntent(context, id, title, body)
        // A banner already showing under this id is replaced by the scheduled
        // one, as a re-post replaces it on the other hosts.
        NotificationManagerCompat.from(context).cancel(id, NOTIFY_ID)
        val exact = Build.VERSION.SDK_INT < Build.VERSION_CODES.S || alarms.canScheduleExactAlarms()
        try {
            if (exact) {
                alarms.setExactAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, at, pending)
            } else {
                alarms.setAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, at, pending)
            }
        } catch (e: SecurityException) {
            // Exact-alarm access revoked between the check and the call.
            alarms.setAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, at, pending)
            Log.w(TAG, "notification $id scheduled inexactly", e)
        }
    }

    /** Called by [NotificationAlarmReceiver] when a scheduled post is due. */
    internal fun postScheduled(context: Context, intent: Intent) {
        val id = intent.getStringExtra(EXTRA_ID)?.takeIf { it.isNotEmpty() } ?: return
        post(
            context.applicationContext,
            id,
            intent.getStringExtra(EXTRA_TITLE).orEmpty(),
            intent.getStringExtra(EXTRA_BODY).orEmpty(),
        )
    }

    private fun post(context: Context, id: String, title: String, body: String) {
        ensureChannel(context)
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

/**
 * The target of a scheduled notification's alarm; see "Scheduled posts" on
 * [Notifications]. A manifest receiver, so the system can start the app's
 * process to deliver it after the app was swiped away.
 */
class NotificationAlarmReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        Notifications.postScheduled(context, intent)
    }
}
