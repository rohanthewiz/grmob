package com.grmob.app

import android.Manifest
import android.content.Context
import android.content.SharedPreferences
import android.content.pm.PackageManager
import android.os.Build
import android.util.Log
import androidx.activity.ComponentActivity
import androidx.activity.result.ActivityResultLauncher
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.content.ContextCompat
import org.json.JSONObject

/**
 * The Android half of Go's permission package.
 *
 *   permission.Check   ──▶ "permission" {command:"check",   kind:…} ──▶ here
 *   permission.Request ──▶ "permission" {command:"request", kind:…} ──▶ here
 *   permission.Current ◀── "permission" host event ◀── here
 *
 * # Why this one needs the Activity when nothing else here does
 *
 * SystemEvents keeps only an application context, deliberately, so an Activity
 * handed to it is not leaked past its own lifetime. A runtime permission
 * cannot be requested that way: `registerForActivityResult` is an Activity API
 * and the registration has to happen before the Activity is STARTED, which
 * means during onCreate and nowhere else. So MainActivity attaches this
 * directly and the launcher is built there and then.
 *
 * The Activity is held as a plain reference for the process's life, which is
 * safe here for the reason it usually is not: this is a single-Activity app
 * whose Activity is recreated on rotation, and [attach] replaces the reference
 * each time onCreate runs. A multi-Activity shell would want a weak reference
 * and a null check on every use.
 *
 * # The state Android does not have
 *
 * `checkSelfPermission` answers GRANTED or DENIED and nothing else, so the
 * three states Go's Status carries have to be reconstructed:
 *
 *     GRANTED                                        -> granted
 *     DENIED + shouldShowRequestPermissionRationale   -> denied  (asked, refused)
 *     DENIED + no rationale + we have asked before    -> denied  (refused for good)
 *     DENIED + no rationale + we have never asked     -> prompt
 *
 * The middle two are the same word on purpose. They differ in whether a
 * further request will show anything, and the platform will not say which —
 * `shouldShowRequestPermissionRationale` is false both for "never asked" and
 * for "don't ask again", which is the ambiguity the third column resolves.
 * Answering "prompt" for a permanently refused permission would put a screen
 * into a loop offering a button that does nothing, so the tie is broken
 * towards denied and the only thing tracked is whether this install has ever
 * asked.
 *
 * # The flag is on disk, and why that changed
 *
 * That flag used to be in memory only, on the argument that a framework shell
 * should not write to disk unasked. The cost was a real one and it was paid on
 * every cold start: a permanently refused permission read as "prompt" until
 * the next request proved otherwise, so the first thing the user saw was a
 * button offering to ask, and pressing it did nothing at all. One dead press
 * per launch, forever.
 *
 * It is in [SharedPreferences] now. What tipped it: the fact being remembered
 * is not the app's, it is the *platform's* — "has this install ever asked for
 * X" is bookkeeping Android itself keeps and simply will not answer — and a
 * quirk of one platform belongs in the one file that knows about it rather
 * than in every app built on top. The write is a string set in the app's own
 * private preferences, it happens on the request path only (never on a check,
 * which is the path a render pass takes), and it goes away with the app's data
 * like every other thing here.
 *
 * # Auto-reset, which is why a denied request still launches
 *
 * Persisting the flag introduces a failure the in-memory version could not
 * have: Android 11+ revokes permissions for apps the user has not opened in
 * months and *resets* the don't-ask-again state with them. The flag on disk
 * still says "asked", so [status] answers "denied" while the truth is that a
 * request would show the dialog again — and a request path that short-circuits
 * on "denied" would lock the user out of a permission the system had just
 * handed back.
 *
 * So [request] short-circuits on "granted" and "unavailable" only, and lets
 * the launcher decide the rest. That is safe because
 * `registerForActivityResult` always delivers a result: a permanently refused
 * permission comes straight back as DENIED with nothing on screen, which is
 * one no-op round trip and the same answer as before, while an auto-reset one
 * shows the dialog and can be granted.
 *
 * # The manifest is the other half
 *
 * A permission not declared in AndroidManifest.xml is never granted and never
 * prompts — `requestPermissions` returns DENIED immediately, with nothing on
 * screen. That is [PERMISSIONS]'s reason for existing as a table here: a kind
 * whose manifest entry is missing is reported "unavailable" rather than
 * "denied", because it is a build-time fact the user cannot fix in Settings.
 */
object Permissions {
    private const val TAG = "GrMobPermissions"

    /**
     * Go's Permission constants, mapped to the Android permission strings each
     * one asks for. A kind with an empty array is one this platform has no
     * runtime permission for.
     *
     * Storage is the API-level-dependent one. Android 13 split the old
     * READ_EXTERNAL_STORAGE into per-media-type permissions and stopped
     * granting the old one at all, so asking for the wrong one on the wrong
     * release is an immediate silent denial.
     *
     * Location asks for the coarse permission alone. Go has one Location
     * constant that means the narrower thing on every platform (see its doc),
     * and requesting ACCESS_FINE_LOCATION additionally would show the
     * precise/approximate chooser for a promise the API did not make.
     */
    private val PERMISSIONS: Map<String, Array<String>> = mapOf(
        "camera" to arrayOf(Manifest.permission.CAMERA),
        "microphone" to arrayOf(Manifest.permission.RECORD_AUDIO),
        "location" to arrayOf(Manifest.permission.ACCESS_COARSE_LOCATION),
        "storage" to if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            arrayOf(Manifest.permission.READ_MEDIA_IMAGES)
        } else {
            @Suppress("DEPRECATION")
            arrayOf(Manifest.permission.READ_EXTERNAL_STORAGE)
        },
    )

    private var activity: ComponentActivity? = null
    private var launcher: ActivityResultLauncher<Array<String>>? = null
    private var report: ((String, String) -> Unit)? = null

    /** The Go kind whose request is in flight, or null. */
    private var pending: String? = null

    /**
     * The preferences file the asked-before flag lives in, and the key inside
     * it. Its own file rather than the default one, so an app that keeps its
     * own preferences beside this cannot collide with the key and cannot have
     * its own file cleared by something here.
     */
    private const val PREFS_NAME = "grmob-permissions"
    private const val ASKED_KEY = "asked"

    /** Kinds this install has already requested at least once; see the class doc. */
    private val asked = mutableSetOf<String>()

    /** Where [asked] is persisted. Null until [attach] runs. */
    private var prefs: SharedPreferences? = null

    /**
     * Wires this to the Activity that can show a permission dialog.
     *
     * Must be called from [ComponentActivity.onCreate] — `registerForActivityResult`
     * throws if the Activity is already STARTED — and is called again on every
     * recreation, which replaces both the Activity and its launcher. A
     * request that was in flight across a rotation is answered by the new
     * launcher, because the result is delivered to whichever registration is
     * live under the same key.
     */
    fun attach(host: ComponentActivity, report: (String, String) -> Unit) {
        activity = host
        this.report = report
        loadAsked(host)
        launcher = host.registerForActivityResult(
            ActivityResultContracts.RequestMultiplePermissions(),
        ) { granted ->
            val kind = pending ?: return@registerForActivityResult
            pending = null
            // RequestMultiplePermissions answers per permission. Every kind
            // here asks for exactly one today, and `all` is what keeps that
            // from silently becoming "the first one" if a kind ever asks for
            // two — a partial grant is not a grant.
            send(kind, if (granted.values.isNotEmpty() && granted.values.all { it }) {
                "granted"
            } else {
                "denied"
            })
        }
    }

    /**
     * Dispatches one "permission" system event. Unknown kinds and commands are
     * dropped, matching every host's contract for unknown events.
     */
    fun handle(data: JSONObject) {
        val kind = data.optString("kind")
        if (!PERMISSIONS.containsKey(kind)) return
        when (data.optString("command")) {
            "check" -> send(kind, status(kind))
            "request" -> request(kind)
        }
    }

    private fun request(kind: String) {
        val host = activity
        val launch = launcher
        if (host == null || launch == null) {
            Log.w(TAG, "permission request before attach(); nothing can prompt")
            send(kind, "unavailable")
            return
        }
        // The two states where launching cannot tell us anything we do not
        // already know: a granted permission comes straight back granted, and
        // one the manifest never declared comes back DENIED, which would turn
        // "unavailable" into "denied" and send the user to a settings page
        // with no switch on it.
        //
        // "denied" is deliberately not in this list — see "Auto-reset" in the
        // class doc. The flag that produced it may be stale, and the launcher
        // is the only thing that can say so.
        val current = status(kind)
        if (current == "granted" || current == "unavailable") {
            send(kind, current)
            return
        }
        rememberAsked(kind)
        pending = kind
        launch.launch(PERMISSIONS.getValue(kind))
    }

    /** The three-state reconstruction described in the class doc. */
    private fun status(kind: String): String {
        val host = activity ?: return "unavailable"
        val wanted = PERMISSIONS.getValue(kind)
        if (wanted.isEmpty() || !declared(host, wanted)) return "unavailable"

        val allGranted = wanted.all {
            ContextCompat.checkSelfPermission(host, it) == PackageManager.PERMISSION_GRANTED
        }
        if (allGranted) return "granted"

        val rationale = wanted.any { host.shouldShowRequestPermissionRationale(it) }
        if (rationale || kind in asked) return "denied"
        return "prompt"
    }

    /**
     * Reads the asked-before flags back off disk, replacing whatever this
     * process had.
     *
     * Called from [attach], which runs again on every Activity recreation; the
     * clear is what keeps a rotation from being able to widen the set with a
     * stale in-memory entry. A read failure is not fatal — the set simply
     * stays empty, which is the behaviour this had before it was persisted at
     * all.
     *
     * The returned Set is copied rather than kept. SharedPreferences hands
     * back an instance it owns and whose contents are undefined after the next
     * read, and mutating it is explicitly not allowed; a shell holding onto it
     * would be writing into the store's own state.
     */
    private fun loadAsked(host: Context) {
        val store = try {
            host.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        } catch (e: Exception) {
            Log.w(TAG, "cannot open the permission preferences; asked-before is memory only", e)
            null
        }
        prefs = store
        asked.clear()
        val saved = store?.getStringSet(ASKED_KEY, null) ?: return
        asked += saved
    }

    /**
     * Records that this install has now asked for [kind], on disk as well as
     * in memory.
     *
     * The early return keeps a repeated request off the disk entirely: the set
     * only ever grows, so a kind already in it has nothing to write.
     *
     * `apply()` rather than `commit()`: the write is asynchronous and does not
     * block the gesture that started the request, and the in-memory set is
     * already correct for every read between now and the write landing. A
     * process killed in that window forgets one flag, which is exactly the
     * state this whole file used to be in permanently.
     *
     * A fresh Set is passed in, not [asked] itself, for the reason [loadAsked]
     * copies: the store keeps the instance it is given, and handing it a
     * collection this object goes on mutating makes the file's contents depend
     * on when it happened to be flushed.
     */
    private fun rememberAsked(kind: String) {
        if (!asked.add(kind)) return
        prefs?.edit()?.putStringSet(ASKED_KEY, asked.toSet())?.apply()
    }

    /**
     * Whether the manifest declares every permission this kind needs.
     *
     * An undeclared permission is not a refusal — it is a build the user
     * cannot influence — so it maps to "unavailable" and not "denied". Without
     * this a screen would offer an "Open Settings" button leading to a page
     * with no switch on it.
     */
    private fun declared(context: Context, wanted: Array<String>): Boolean {
        val declaredNames = try {
            context.packageManager
                .getPackageInfo(context.packageName, PackageManager.GET_PERMISSIONS)
                .requestedPermissions ?: emptyArray()
        } catch (e: PackageManager.NameNotFoundException) {
            Log.w(TAG, "cannot read this package's own manifest", e)
            return false
        }
        return wanted.all { it in declaredNames }
    }

    private fun send(kind: String, status: String) {
        val out = report
        if (out == null) {
            Log.w(TAG, "permission answer with no host-event channel attached")
            return
        }
        out("permission", JSONObject().put("kind", kind).put("status", status).toString())
    }
}
