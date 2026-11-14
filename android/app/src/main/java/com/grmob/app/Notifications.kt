package com.grmob.app

import android.app.AlarmManager
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.media.AudioAttributes
import android.media.AudioManager
import android.media.RingtoneManager
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
 *                       cancel alarm; at ≤ now ─▶ reported as fired; remove
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
 *
 * # Alarms ("alarm" on a post)
 *
 * A post carrying `alarm` ({snooze?, ring_for?} in ms) rings rather than
 * chimes — but only in an app that declares USE_FULL_SCREEN_INTENT, which
 * `grmob android` keeps only for "alarms": true in grmob.json (see
 * [alarmsDeclared]). Everywhere else the field is ignored and the post is the
 * banner above, so Go never has to ask what the build can do.
 *
 * ```
 *   post(alarm) ─▶ "Alarms" channel (alarm stream, the default alarm tone)
 *                  FLAG_INSISTENT: the tone repeats until the notification goes
 *                  setTimeoutAfter(ring_for): …which it does by itself after
 *                  CATEGORY_ALARM, actions Snooze (if snooze > 0) and Stop
 *                  full-screen intent, if [canUseFullScreenIntent]:
 *                      locked / asleep ─▶ MainActivity over the lock screen
 *                      in use          ─▶ a heads-up instead (the system's call)
 *
 *   Snooze ─▶ NotificationActionReceiver: take it down, schedule the same id
 *             again at now + snooze (the store keeps the first ring as "rang")
 *   Stop   ─▶ NotificationActionReceiver: take it down
 *   tap / full-screen launch ─▶ MainActivity: "notification_tap" {id, ringing}
 *             ringing = still inside ring_for of when it was posted (see
 *             [tapRinging]); Go's UseAlarms takes the ring over and cancels
 *             the id, which stops the tone
 * ```
 *
 * Snooze keeps the id so everything keyed by it still works: a cancel or a
 * sweep from Go reaches the snoozed alarm, and the tap of its next ring names
 * the same alarm. The store's "rang" field remembers the first ring, so a
 * sweep reports the alarm as fired even while its snooze is still pending.
 *
 * # Over the lock screen, and back under it
 *
 * The full-screen intent only gets the Activity started; showing it over the
 * keyguard is the Activity's own flag. MainActivity sets it for a ringing
 * alarm launch only ([lockScreen] holds the id that did it), never for an
 * ordinary launch, because everything the app shows while it is set is
 * visible without unlocking. It is cleared when that alarm is answered —
 * Stop on the notification, or Go's core.AlarmAnswered once the app's own
 * ringing screen is dismissed or snoozed ("answered") — so the keyguard comes
 * back over the app rather than leaving it open to whoever holds the phone.
 */
object Notifications {
    /** The intent extra that carries a tapped notification's Go id. */
    const val EXTRA_ID = "com.grmob.notification.id"
    private const val EXTRA_TITLE = "com.grmob.notification.title"
    private const val EXTRA_BODY = "com.grmob.notification.body"

    /**
     * On a tap intent, when the post was an alarm: when it started ringing
     * and for how long it rings (ms), which [tapRinging] turns into the tap's
     * "ringing". On an action intent, the alarm object as JSON, so Snooze can
     * re-post an alarm that was never in the store (one posted for now).
     */
    private const val EXTRA_RANG_AT = "com.grmob.notification.rangAt"
    private const val EXTRA_RING_FOR = "com.grmob.notification.ringFor"
    private const val EXTRA_ALARM = "com.grmob.notification.alarm"

    private const val TAG = "GrMobNotifications"
    private const val CHANNEL_ID = "grmob"
    private const val NOTIFY_ID = 1

    /**
     * The alarms' own channel. Separate from [CHANNEL_ID] because a channel's
     * sound and audio stream are fixed once created and belong to the user
     * after that: an alarm needs the alarm stream (it plays at alarm volume,
     * through a silenced ringer) and the alarm tone, and a banner needs
     * neither. "One channel" above is about categories core has no word for;
     * this is a different kind of notification, which core does name.
     */
    private const val ALARM_CHANNEL_ID = "grmob.alarms"

    /** RingFor when Go sent none; the same default as hooks.AlarmOptions. */
    private const val DEFAULT_RING_FOR_MS = 10L * 60 * 1000

    /** The action intents' actions; see [NotificationActionReceiver]. */
    internal const val ACTION_SNOOZE = "com.grmob.notification.SNOOZE"
    internal const val ACTION_STOP = "com.grmob.notification.STOP"

    /** The permission behind alarms; see [alarmsDeclared]. */
    private const val FULL_SCREEN_PERMISSION = "android.permission.USE_FULL_SCREEN_INTENT"

    /**
     * The id of the alarm that put MainActivity over the lock screen, and the
     * Activity's way of leaving it; see "Over the lock screen". Both are set
     * by MainActivity and read on the main thread, where every command and
     * every receiver runs.
     */
    private var lockScreen: String? = null
    private var releaseLockScreen: (() -> Unit)? = null

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

    /**
     * Stores a scheduled post. [alarm] is kept so a re-arm after a reboot
     * posts an alarm, not a banner; [rang] is the first ring of a snoozed
     * alarm (see "Alarms"), absent for a post that has not rung yet.
     */
    private fun remember(
        context: Context, id: String, title: String, body: String, at: Long,
        alarm: JSONObject? = null, rang: Long = 0L,
    ) {
        val entry = JSONObject().put("title", title).put("body", body).put("at", at).put("fired", false)
        if (alarm != null) entry.put("alarm", alarm)
        if (rang > 0) entry.put("rang", rang)
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
            schedule(
                app, id, entry.optString("title"), entry.optString("body"), entry.optLong("at"),
                entry.optJSONObject("alarm"), entry.optLong("rang", 0L),
            )
        }
        for ((id, entry) in due) {
            post(app, id, entry.optString("title"), entry.optString("body"), entry.optJSONObject("alarm"))
        }
    }

    /**
     * Cancels what is still to come under [prefix] — pending alarms and
     * stored entries — and reports the stored ones whose time had come. See
     * core.SweepNotifications.
     *
     * Banners already on screen stay. The sweep runs when an app mounts, and
     * after a force stop [attach] has just posted the missed ones late (see
     * "What outlives a reboot or a force stop"): taking them down here
     * removed a missed-alarm notice moments after it appeared, sound and all.
     * A banner the OS drew is the user's to dismiss.
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
                // A snoozed alarm's "at" is its next ring, still to come;
                // "rang" is its first, which is what makes it fired.
                val at = try {
                    val entry = JSONObject(raw as? String ?: "")
                    entry.optLong("rang", 0L).takeIf { it > 0 } ?: entry.optLong("at", 0L)
                } catch (e: Exception) {
                    0L
                }
                if (at in 1..now) fired.put(id)
                editor.remove(id)
            }
            editor.apply()
        }
        val alarms = alarmManager(context)
        for (id in ids) {
            alarms?.cancel(alarmIntent(context, id, "", ""))
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

    /**
     * The alarms' channel; see [ALARM_CHANNEL_ID]. Created only when an alarm
     * is posted, so an app that never rings one has no "Alarms" row in its
     * notification settings.
     */
    private fun ensureAlarmChannel(context: Context) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val channel = NotificationChannel(ALARM_CHANNEL_ID, "Alarms", NotificationManager.IMPORTANCE_HIGH).apply {
            setSound(
                RingtoneManager.getDefaultUri(RingtoneManager.TYPE_ALARM),
                AudioAttributes.Builder()
                    .setUsage(AudioAttributes.USAGE_ALARM)
                    .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
                    .build(),
            )
            enableVibration(true)
            lockscreenVisibility = Notification.VISIBILITY_PUBLIC
        }
        context.getSystemService(NotificationManager::class.java)?.createNotificationChannel(channel)
    }

    /**
     * Whether this build opted in to alarms: the manifest asks for
     * USE_FULL_SCREEN_INTENT, which `grmob android` keeps only for "alarms":
     * true. Read from the package rather than from a flag of its own, so the
     * line an app keeps or deletes is the whole switch. Also Permissions.kt's
     * "unavailable" test for permission.Alarms.
     */
    internal fun alarmsDeclared(context: Context): Boolean = try {
        @Suppress("DEPRECATION") // the flags overload is 33+; minSdk is 24
        val info = context.packageManager.getPackageInfo(context.packageName, PackageManager.GET_PERMISSIONS)
        info.requestedPermissions?.contains(FULL_SCREEN_PERMISSION) == true
    } catch (e: PackageManager.NameNotFoundException) {
        false
    }

    /**
     * Whether a full-screen intent will be honoured: declared, and from 14
     * granted (Play grants it at install to alarm and calling apps; for the
     * rest it is a Settings switch). Below 14 the declaration is the grant.
     * Permissions.kt reports this as permission.Alarms.
     */
    internal fun canUseFullScreenIntent(context: Context): Boolean {
        if (!alarmsDeclared(context)) return false
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.UPSIDE_DOWN_CAKE) return true
        return context.getSystemService(NotificationManager::class.java)?.canUseFullScreenIntent() == true
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
                // Absent means a banner; see "Alarms".
                val alarm = data.optJSONObject("alarm")
                // Absent means now; Go leaves the key off for a past time.
                val at = data.optLong("at", 0L)
                if (at > System.currentTimeMillis()) {
                    schedule(context, id, title, body, at, alarm)
                } else {
                    // "Posting again under the same ID replaces the pending
                    // request" (core/notifications.go): an immediate post takes
                    // down an alarm still waiting under the id, as iOS's
                    // same-identifier add does, instead of letting it fire a
                    // second banner later.
                    alarmManager(context)?.cancel(alarmIntent(context, id, "", ""))
                    forget(context, id)
                    post(context, id, title, body, alarm)
                }
            }
            "cancel" -> {
                alarmManager(context)?.cancel(alarmIntent(context, id, "", ""))
                forget(context, id)
                NotificationManagerCompat.from(context).cancel(id, NOTIFY_ID)
            }
            // The app's own ringing screen for this alarm is done (Go's
            // core.AlarmAnswered); see "Over the lock screen".
            "answered" -> leaveLockScreen(id)
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

    private fun schedule(
        context: Context, id: String, title: String, body: String, at: Long,
        alarm: JSONObject? = null, rang: Long = 0L,
    ) {
        val alarms = alarmManager(context) ?: return
        val pending = alarmIntent(context, id, title, body)
        // A banner already showing under this id is replaced by the scheduled
        // one, as a re-post replaces it on the other hosts.
        NotificationManagerCompat.from(context).cancel(id, NOTIFY_ID)
        // Written before the alarm is set, so a process killed between the two
        // leaves an entry the next rearm schedules rather than an alarm the
        // store does not know about.
        remember(context, id, title, body, at, alarm, rang)
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
        // Claimed rather than forgotten: see "Fired entries". The alarm
        // object comes from the entry, which the claim has just proved
        // exists, rather than from an extra: the store is what a re-arm
        // reads too, so the two paths cannot disagree about it.
        val alarm = synchronized(lock) {
            if (!claim(context, id)) return
            entry(context, id)?.optJSONObject("alarm")
        }
        post(
            context.applicationContext,
            id,
            intent.getStringExtra(EXTRA_TITLE).orEmpty(),
            intent.getStringExtra(EXTRA_BODY).orEmpty(),
            alarm,
        )
    }

    private fun post(context: Context, id: String, title: String, body: String, alarm: JSONObject? = null) {
        if (alarm != null && alarmsDeclared(context)) {
            postAlarm(context, id, title, body, alarm)
            return
        }
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
     * Posts [id] as a ringing alarm; see "Alarms" for the whole shape. Same
     * permission gate and tag/NOTIFY_ID keying as [post], so a cancel or a
     * re-post under the id finds it.
     */
    private fun postAlarm(context: Context, id: String, title: String, body: String, alarm: JSONObject) {
        ensureAlarmChannel(context)
        val manager = NotificationManagerCompat.from(context)
        if (!manager.areNotificationsEnabled()) return

        val now = System.currentTimeMillis()
        val ringFor = alarm.optLong("ring_for", 0L).takeIf { it > 0 } ?: DEFAULT_RING_FOR_MS
        val snooze = alarm.optLong("snooze", 0L)

        // One PendingIntent serves the tap and the full-screen launch: both
        // open the app on this alarm, and both are reported as a tap that is
        // ringing while ring_for lasts (see [tapRinging]).
        val open = Intent(context, MainActivity::class.java)
            .putExtra(EXTRA_ID, id)
            .putExtra(EXTRA_RANG_AT, now)
            .putExtra(EXTRA_RING_FOR, ringFor)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP)
        val openPending = PendingIntent.getActivity(
            context,
            id.hashCode(),
            open,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )

        val builder = NotificationCompat.Builder(context, ALARM_CHANNEL_ID)
            .setSmallIcon(smallIcon(context))
            .setContentTitle(title.ifEmpty { null })
            .setContentText(body.ifEmpty { null })
            .setCategory(NotificationCompat.CATEGORY_ALARM)
            // Pre-O has no channels: the priority and sound live here.
            .setPriority(NotificationCompat.PRIORITY_MAX)
            .setSound(RingtoneManager.getDefaultUri(RingtoneManager.TYPE_ALARM), AudioManager.STREAM_ALARM)
            .setVisibility(NotificationCompat.VISIBILITY_PUBLIC)
            .setContentIntent(openPending)
            // A tap answers the ring by opening the app, which takes it over;
            // so the tone stops with the tap rather than playing on under it.
            .setAutoCancel(true)
            // Not swiped away by accident while it rings; Stop is the way.
            .setOngoing(true)
            .setTimeoutAfter(ringFor)
        if (snooze > 0) {
            builder.addAction(0, "Snooze", actionIntent(context, ACTION_SNOOZE, id, title, body, alarm))
        }
        builder.addAction(0, "Stop", actionIntent(context, ACTION_STOP, id, title, body, alarm))
        if (canUseFullScreenIntent(context)) {
            builder.setFullScreenIntent(openPending, true)
        }
        val notification = builder.build()
        // The tone repeats until the notification is cancelled — by Stop,
        // Snooze, a tap, Go's takeover, or the timeout above.
        notification.flags = notification.flags or Notification.FLAG_INSISTENT
        try {
            manager.notify(id, NOTIFY_ID, notification)
        } catch (e: SecurityException) {
            Log.w(TAG, "alarm $id refused", e)
        }
    }

    /**
     * Snooze's or Stop's broadcast for [id]. Keyed by action and id in the
     * data, like [alarmIntent], so each alarm's two buttons are two
     * PendingIntents and neither is another alarm's.
     */
    private fun actionIntent(
        context: Context, action: String, id: String, title: String, body: String, alarm: JSONObject,
    ): PendingIntent {
        val intent = Intent(context, NotificationActionReceiver::class.java)
            .setAction(action)
            .setData(android.net.Uri.fromParts("grmob-notification-action", id, action))
            .putExtra(EXTRA_ID, id)
            .putExtra(EXTRA_TITLE, title)
            .putExtra(EXTRA_BODY, body)
            .putExtra(EXTRA_ALARM, alarm.toString())
        return PendingIntent.getBroadcast(
            context,
            id.hashCode(),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
    }

    /**
     * Snooze or Stop pressed on a ringing alarm; no Go runs for either. See
     * "Alarms". Snooze schedules the same id again, keeping the first ring as
     * "rang" so a sweep still reports the alarm as fired.
     */
    internal fun onAction(context: Context, intent: Intent) {
        val id = intent.getStringExtra(EXTRA_ID)?.takeIf { it.isNotEmpty() } ?: return
        val app = context.applicationContext
        NotificationManagerCompat.from(app).cancel(id, NOTIFY_ID)
        leaveLockScreen(id)
        if (intent.action != ACTION_SNOOZE) return
        val alarm = try {
            JSONObject(intent.getStringExtra(EXTRA_ALARM) ?: return)
        } catch (e: Exception) {
            return
        }
        val snooze = alarm.optLong("snooze", 0L).takeIf { it > 0 } ?: return
        val now = System.currentTimeMillis()
        // The first ring: the stored one if this alarm was scheduled (and
        // perhaps snoozed before), otherwise now — it was posted for now.
        val rang = synchronized(lock) {
            entry(app, id)?.let { it.optLong("rang", 0L).takeIf { r -> r > 0 } ?: it.optLong("at", 0L) }
        }?.takeIf { it > 0 } ?: now
        schedule(
            app, id,
            intent.getStringExtra(EXTRA_TITLE).orEmpty(), intent.getStringExtra(EXTRA_BODY).orEmpty(),
            now + snooze, alarm, rang,
        )
    }

    /**
     * Whether a launch intent is a ringing alarm's: posted as an alarm, and
     * still inside its ring_for. A tap on a banner, or on an alarm whose ring
     * has timed out, is not — which is what keeps a stale tap from starting
     * a ring in the app (core.OnRingingAlarmTap). MainActivity puts this in
     * the tap's "ringing" and shows over the lock screen for it.
     */
    fun tapRinging(intent: Intent?): Boolean {
        val rangAt = intent?.getLongExtra(EXTRA_RANG_AT, 0L) ?: 0L
        if (rangAt <= 0) return false
        val ringFor = intent?.getLongExtra(EXTRA_RING_FOR, DEFAULT_RING_FOR_MS) ?: DEFAULT_RING_FOR_MS
        return System.currentTimeMillis() < rangAt + ringFor
    }

    /**
     * Records that MainActivity is over the lock screen for alarm [id], and
     * how to take it back under; see "Over the lock screen".
     */
    fun showingOverLockScreen(id: String, release: () -> Unit) {
        lockScreen = id
        releaseLockScreen = release
        // One line per alarm, like the sweep's: the log is the only place a
        // person can see the app went over the lock screen, and back under.
        Log.i(TAG, "over the lock screen for alarm $id")
    }

    /** Ends what [showingOverLockScreen] started, if [id] is the alarm that did. */
    private fun leaveLockScreen(id: String) {
        if (lockScreen != id) return
        lockScreen = null
        releaseLockScreen?.invoke()
        releaseLockScreen = null
        Log.i(TAG, "alarm $id answered; back under the lock screen")
    }

    /** The alarm holding the lock screen, for a recreated MainActivity to resume. */
    fun lockScreenAlarm(): String? = lockScreen

    /**
     * Forgets the release of an Activity being destroyed, so a later
     * "answered" does not call into it. The flag is the Activity's own, so
     * once it is gone the lock screen is already back — unless [changing]
     * says a configuration change is replacing it, when the alarm's claim is
     * kept for the new Activity to take up (MainActivity.onCreate).
     */
    fun activityGone(changing: Boolean) {
        releaseLockScreen = null
        if (!changing) lockScreen = null
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
 * Snooze and Stop on a ringing alarm; see "Alarms" on [Notifications]. Not
 * exported: only the PendingIntents [Notifications] builds reach it, and the
 * action is checked for the reason [NotificationBootReceiver] checks its own.
 */
class NotificationActionReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        when (intent.action) {
            Notifications.ACTION_SNOOZE, Notifications.ACTION_STOP -> Notifications.onAction(context, intent)
        }
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
            // alarms set before it become exact, and a running Go app's
            // permission record hears it, with or without an Activity on
            // screen. The literal rather than
            // AlarmManager's constant, which only exists from API 31.
            "android.app.action.SCHEDULE_EXACT_ALARM_PERMISSION_STATE_CHANGED" -> {
                Notifications.rearm(context)
                Permissions.recheckExactAlarms()
            }
        }
    }
}
