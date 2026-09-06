package com.grmob.app

import android.Manifest
import android.content.Context
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
 * asked. That flag is in-memory: a process restart forgets it, which turns a
 * permanent refusal back into "prompt" until the next request proves
 * otherwise. Persisting it is the kind of thing an app does with its own
 * preferences, not something a framework shell should be writing to disk
 * unasked.
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

    /** Kinds this process has already requested at least once; see the class doc. */
    private val asked = mutableSetOf<String>()

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
        // Already answered: asking again either shows nothing (a permanent
        // refusal) or is pointless (already granted), and in both cases a
        // screen waiting on a dialog would wait forever. The current status is
        // the truthful reply.
        val current = status(kind)
        if (current != "prompt") {
            send(kind, current)
            return
        }
        asked += kind
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
