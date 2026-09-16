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
import org.json.JSONArray
import org.json.JSONObject

/**
 * The Android half of core's local notifications (core/notifications.go).
 *
 *   core.PostNotification   ──▶ "notification" {command: "post", id, title, body, at?}
 *   core.CancelNotification ──▶ "notification" {command: "cancel", id}
 *   core.SweepNotifications ──▶ "notification" {command: "sweep", prefix, request}
 *                           ◀── "notification_swept" {request, fired: [ids]}
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
 *
 * # What outlives a reboot or a force stop
 *
 * AlarmManager forgets every alarm at a reboot, and a force stop cancels them
 * too (and blocks the app's receivers until it is next launched). The OS keeps
 * no list to re-arm from, so the shell keeps one: each scheduled post is
 * written to its own SharedPreferences file, keyed by the Go id, and removed
 * when it fires, is cancelled, or is replaced by an immediate post.
 *
 * ```
 *   schedule ──▶ store[id] = {title, body, at}     fire / cancel / post-now ──▶ remove
 *
 *   BOOT_COMPLETED ─┐
 *   package update ─┼─▶ rearm(context): at > now ─▶ schedule again (same PendingIntent)
 *   attach (launch) ┘                   at ≤ now ─▶ post now, remove  ("late, not never")
 * ```
 *
 * Why [attach] re-arms too: a force stop delivers nothing — no broadcast says
 * the alarms are gone — and the next launch is the first moment this code runs
 * again. Re-scheduling an alarm that did survive is harmless, because the
 * PendingIntent is keyed by id and replaces itself.
 *
 * A post that came due while the device was off is posted at once rather than
 * dropped, matching the inexact fallback's stance above. The store only ever
 * holds what Go scheduled (UseAlarms keeps it to at most 60 entries a week
 * out), so this is never an unbounded replay.
 *
 * # Fired entries, and posting each one exactly once
 *
 * An entry is marked fired when it posts rather than removed, for two
 * readers:
 *
 * ```
 *   entry: {title, body, at, fired}
 *
 *   alarm fires ──┐                    ┌─ fired already ─▶ nothing (the other path posted)
 *   rearm, due  ──┴─▶ [lock] claim() ──┤
 *                                      └─ not fired ─────▶ mark fired, post
 *
 *   sweep(prefix) ──▶ [lock] for each entry under prefix:
 *                       cancel alarm + banner; at ≤ now ─▶ reported as fired; remove
 * ```
 *
 * - The sweep (core.SweepNotifications) reports what fired while the app was
 *   not running, which a removed entry could not tell it.
 * - The claim closes a double post: [attach] re-arms at launch, and an entry
 *   due at that moment could otherwise be posted by rearm and again by its own
 *   alarm arriving a moment later. Both paths claim under one lock, and only
 *   the first posts. An entry that is not in the store at all is not claimed
 *   either: it was cancelled or swept, and its alarm with it.
 *
 * Fired entries are pruned by [rearm] a week after their time (the horizon
 * UseAlarms schedules within), so an app that never sweeps keeps a bounded
 * store. SharedPreferences updates its in-memory map synchronously on
 * `apply()`, so the lock plus the map is the whole consistency story within
 * the one process an app has.
 *
 * # An exact-alarm grant
 *
 * A post scheduled while exact alarms were refused was set inexactly, and
 * stays that way after the user allows them unless something reschedules it.
 * [NotificationBootReceiver] also hears
 * ACTION_SCHEDULE_EXACT_ALARM_PERMISSION_STATE_CHANGED (sent on a grant; a
 * revocation kills the process instead) and re-arms, which re-sets every
 * pending alarm through [schedule]'s exact branch. It also asks Permissions
 * to re-check, so a screen using hooks.UsePermissionLive hears the grant
 * without waiting for the app to come to the foreground.
 */
object Notifications {
    /** The intent extra that carries a tapped notification's Go id. */
    const val EXTRA_ID = "com.grmob.notification.id"
    private const val EXTRA_TITLE = "com.grmob.notification.title"
    private const val EXTRA_BODY = "com.grmob.notification.body"

    private const val TAG = "GrMobNotifications"
    private const val CHANNEL_ID = "grmob"
    private const val NOTIFY_ID = 1

    /**
     * The file the pending-post store lives in. Its own file, like
     * Permissions.kt's, so an app's own preferences can neither collide with an
     * id nor be cleared from here.
     */
    private const val STORE_NAME = "grmob-scheduled-notifications"

    /** How long a fired entry is kept for a sweep to report; see "Fired entries". */
    private const val FIRED_KEEP_MS = 7L * 24 * 60 * 60 * 1000

    private var appContext: Context? = null
    private var report: ((String, String) -> Unit)? = null

    /** Guards the store's read-modify-write sequences; see "Fired entries". */
    private val lock = Any()

    fun attach(context: Context, out: (String, String) -> Unit) {
        val app = context.applicationContext
        appContext = app
        report = out
        ensureChannel(app)
        // A force stop cleared the alarms without telling anyone; see
        // "What outlives a reboot or a force stop".
        rearm(app)
    }

    private fun store(context: Context) =
        context.applicationContext.getSharedPreferences(STORE_NAME, Context.MODE_PRIVATE)

    private fun remember(context: Context, id: String, title: String, body: String, at: Long) {
        val entry = JSONObject().put("title", title).put("body", body).put("at", at).put("fired", false)
        synchronized(lock) {
            store(context).edit().putString(id, entry.toString()).apply()
        }
    }

    /** The stored entry for [id], or null when there is none or it does not parse. */
    private fun entry(context: Context, id: String): JSONObject? {
        val raw = store(context).getString(id, null) ?: return null
        return try {
            JSONObject(raw)
        } catch (e: Exception) {
            null
        }
    }

    /**
     * Marks [id] fired and returns true if this caller should post it: the
     * entry exists and no other path has posted it. Called with [lock] held.
     */
    private fun claim(context: Context, id: String): Boolean {
        val entry = entry(context, id) ?: return false
        if (entry.optBoolean("fired", false)) return false
        store(context).edit().putString(id, entry.put("fired", true).toString()).apply()
        return true
    }

    private fun forget(context: Context, id: String) {
        synchronized(lock) {
            store(context).edit().remove(id).apply()
        }
    }

    /**
     * Re-arms every stored post: the future ones through [schedule], which
     * replaces any alarm still pending under the id, and the ones whose time
     * passed while nothing could fire them posted now (once; see "Fired
     * entries"). Fired entries older than [FIRED_KEEP_MS] are pruned. An
     * entry that does not parse is dropped rather than retried at every boot.
     *
     * The due posts are collected under the lock and posted after it, so
     * NotificationManager is never called with the store locked.
     */
    internal fun rearm(context: Context) {
        val app = context.applicationContext
        val now = System.currentTimeMillis()
        val future = mutableListOf<Pair<String, JSONObject>>()
        val due = mutableListOf<Pair<String, JSONObject>>()
        synchronized(lock) {
            for ((id, raw) in store(app).all) {
                val entry = try {
                    JSONObject(raw as? String ?: "")
                } catch (e: Exception) {
                    Log.w(TAG, "dropping unreadable scheduled notification $id", e)
                    store(app).edit().remove(id).apply()
                    continue
                }
                val at = entry.optLong("at", 0L)
                when {
                    entry.optBoolean("fired", false) ->
                        if (now - at > FIRED_KEEP_MS) store(app).edit().remove(id).apply()
                    at > now -> future += id to entry
                    claim(app, id) -> due += id to entry
                }
            }
        }
        for ((id, entry) in future) {
            schedule(app, id, entry.optString("title"), entry.optString("body"), entry.optLong("at"))
        }
        for ((id, entry) in due) {
            post(app, id, entry.optString("title"), entry.optString("body"))
        }
    }

    /**
     * Cancels everything under [prefix] — pending alarms, stored entries and
     * banners on screen — and reports the stored ones whose time had come.
     * See core.SweepNotifications.
     *
     * Banners are found through getActiveNotifications as well as the store,
     * because an immediate post under the prefix never entered the store.
     */
    private fun sweep(context: Context, prefix: String, request: String) {
        val now = System.currentTimeMillis()
        val fired = JSONArray()
        val ids = mutableListOf<String>()
        synchronized(lock) {
            val editor = store(context).edit()
            for ((id, raw) in store(context).all) {
                if (!id.startsWith(prefix)) continue
                ids += id
                val at = try {
                    JSONObject(raw as? String ?: "").optLong("at", 0L)
                } catch (e: Exception) {
                    0L
                }
                if (at in 1..now) fired.put(id)
                editor.remove(id)
            }
            editor.apply()
        }
        val alarms = alarmManager(context)
        val manager = NotificationManagerCompat.from(context)
        for (id in ids) {
            alarms?.cancel(alarmIntent(context, id, "", ""))
            manager.cancel(id, NOTIFY_ID)
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            val shown = context.getSystemService(NotificationManager::class.java)?.activeNotifications.orEmpty()
            for (n in shown) {
                val tag = n.tag ?: continue
                if (tag.startsWith(prefix)) manager.cancel(tag, NOTIFY_ID)
            }
        }
        // One line per sweep: they are rare (a mount, a first return), and the
        // log is the only place a person can see what a relaunch reported.
        Log.i(TAG, "swept ${ids.size} under $prefix; fired=$fired")
        val payload = JSONObject().put("request", request).put("fired", fired)
        report?.invoke("notification_swept", payload.toString())
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
        val context = appContext ?: return
        // The one command addressed by prefix rather than id.
        if (data.optString("command") == "sweep") {
            val prefix = data.optString("prefix")
            val request = data.optString("request")
            if (prefix.isNotEmpty() && request.isNotEmpty()) sweep(context, prefix, request)
            return
        }
        val id = data.optString("id")
        if (id.isEmpty()) return
        when (data.optString("command")) {
            "post" -> {
                val title = data.optString("title")
                val body = data.optString("body")
                // Absent means now; Go leaves the key off for a past time.
                val at = data.optLong("at", 0L)
                if (at > System.currentTimeMillis()) {
                    schedule(context, id, title, body, at)
                } else {
                    // "Posting again under the same ID replaces the pending
                    // request" (core/notifications.go): an immediate post takes
                    // down an alarm still waiting under the id, as iOS's
                    // same-identifier add does, instead of letting it fire a
                    // second banner later.
                    alarmManager(context)?.cancel(alarmIntent(context, id, "", ""))
                    forget(context, id)
                    post(context, id, title, body)
                }
            }
            "cancel" -> {
                alarmManager(context)?.cancel(alarmIntent(context, id, "", ""))
                forget(context, id)
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
        // Written before the alarm is set, so a process killed between the two
        // leaves an entry the next rearm schedules rather than an alarm the
        // store does not know about.
        remember(context, id, title, body, at)
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
        // Claimed rather than forgotten: see "Fired entries".
        if (!synchronized(lock) { claim(context, id) }) return
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

/**
 * Re-arms scheduled notifications after the events that clear AlarmManager
 * without the app running: a reboot (BOOT_COMPLETED, delivered once the user
 * has unlocked, when the app's credential-protected preferences are readable)
 * and an update of this app (MY_PACKAGE_REPLACED). See "What outlives a reboot
 * or a force stop" on [Notifications]. The action is checked because a
 * manifest receiver can be sent an explicit intent with any action at all.
 */
class NotificationBootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        when (intent.action) {
            Intent.ACTION_BOOT_COMPLETED, Intent.ACTION_MY_PACKAGE_REPLACED ->
                Notifications.rearm(context)
            // A grant of exact alarms (see "An exact-alarm grant"): inexact
            // alarms set before it become exact, and a running app's
            // permission record hears it. The literal rather than
            // AlarmManager's constant, which only exists from API 31.
            "android.app.action.SCHEDULE_EXACT_ALARM_PERMISSION_STATE_CHANGED" -> {
                Notifications.rearm(context)
                Permissions.recheckExactAlarms()
            }
        }
    }
}
