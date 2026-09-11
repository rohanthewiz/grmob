package com.grmob.app

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.location.Location
import android.location.LocationListener
import android.location.LocationManager
import android.os.Looper
import android.util.Log
import androidx.core.content.ContextCompat
import org.json.JSONObject

/**
 * The Android half of grmob's positioning (core/location.go).
 *
 *   core.StartLocation ──▶ "sensor" {kind:"location", command:"start"} ──▶ here
 *   core.CurrentLocation ◀── "location" host event ◀── here
 *
 * Built on HeadingSensor's shape, one file over, and differing from it in the
 * two places location is more expensive than a compass: it needs a runtime
 * permission, and it costs enough battery that the throttle is about power
 * rather than about render passes.
 *
 * # LocationManager, not the fused provider
 *
 * `FusedLocationProviderClient` is the better API on devices that have Play
 * Services, and it is a dependency on Play Services being present — which it is
 * not on a real share of Android devices, and not on any build that ships
 * outside Google's ecosystem. The platform's own LocationManager is in every
 * Android since API 1, needs no artifact, and gives what this node's consumers
 * ask for: a coordinate, an accuracy and an altitude.
 *
 * That is the same reasoning GrMobMapView.kt applies one package over when it
 * draws with osmdroid rather than Google Maps: the keyless, dependency-free
 * option is the one that works on arrival, and the better-on-some-devices one
 * is a provider this file could grow.
 *
 * # The permission, which this does not ask for
 *
 * ACCESS_COARSE_LOCATION is what the manifest declares — see the note in
 * permission.Location on why one Go constant means the narrower thing on every
 * platform — and a *request* is Permissions.kt's job, from a gesture, because
 * only an Activity can show the dialog and only the app knows when asking is
 * appropriate.
 *
 * So a start with no permission is reported rather than attempted:
 * `available: false` with a reason, which is the shape Go's Location.Available
 * and Location.Error are written against and the only thing that lets a screen
 * stop waiting. That is a deliberate divergence from the iOS host, which *can*
 * prompt from the sensor and does — and it is the platform's difference rather
 * than this framework's: CLLocationManager requests authorization from anywhere,
 * ActivityCompat.requestPermissions needs the Activity that Permissions.kt
 * holds.
 *
 * # And a refusal is not final
 *
 * The grant usually arrives *after* the screen that wants it, because the tap
 * that asks for it is on that screen. So a refused start stays armed and
 * [Permissions] calls [permissionAnswer] when the answer changes — without
 * which the sensor is dead for the life of the context tree, which is what an
 * emulator run found. [awaitingPermission] carries the whole argument.
 *
 * The same is true of location services being switched off system-wide, and
 * for the same reason: the user goes to fix it and comes back. That arm is
 * [awaitingProvider], and it recovers through LocationListener's own
 * `onProviderEnabled` rather than through [Permissions].
 *
 * Either way, a start that gets through says `acquiring: true` before any fix
 * exists, so that Go stops reporting the reason the previous attempt failed.
 */
object LocationSensor : LocationListener {
    private const val TAG = "GrMobLocation"

    /**
     * Throttle floor, in ms. Looser than the compass's 66 — a position is not a
     * needle, and a second is a long time at walking pace — and it is the
     * *power* knob as much as the bridge one: the interval below is what the
     * platform is asked for, and a provider asked for fixes ten times a second
     * keeps the radio awake.
     */
    private const val MIN_INTERVAL_MS = 1000L

    /**
     * Metres of movement before the platform bothers to report, which is a
     * filter that costs nothing because it happens before the callback. Five
     * metres is inside a good fix's own accuracy, so it filters jitter rather
     * than movement — and Go filters again at one metre
     * (locationNotifyEpsilonMeters) for the hosts that have no such control.
     */
    private const val MIN_DISTANCE_M = 5f

    private var appContext: Context? = null
    private var manager: LocationManager? = null
    private var report: ((String, String) -> Unit)? = null

    private var running = false
    private var lastSentAt = 0L

    /**
     * A start is outstanding and cannot proceed until the location permission
     * is granted. Set by [start] when it bails on the permission, cleared by a
     * start that gets through and by [stop].
     *
     * # The bug this exists for
     *
     * An emulator run granted the permission *after* the screen had mounted and
     * the sensor stayed dead for the life of the context tree:
     *
     *     launch with permission revoked     permission: denied    Available: false
     *     grant it through the app           permission: granted   Available: false
     *     12s later, with a fix being fed    permission: granted   Available: false
     *
     * The permission answer reached Go — the readout updated — and nothing
     * reached the sensor. [start] returns early without setting [running], and
     * `core.StartLocation` emits its "sensor" event only on the 0→1 transition
     * of its reference count (and `hooks.UseLocation` guards its own slot with
     * `locationRecord.started`), so the host command that would re-arm the
     * sensor is never sent a second time. The only recovery was a remount.
     *
     * # Why the host and not Go
     *
     * Three reasons, and the third is the decisive one. The host is where the
     * failure is *known* — this object is the only thing that knows the start
     * did not take and why. The host is where the answer *arrives*:
     * [Permissions] already has it in hand and calls [permissionAnswer] with
     * it, so there is no new plumbing. And putting it in `hooks.UseLocation`
     * would mean the hook reading a permission status to decide whether to
     * start the sensor, which is the second authorization policy that hook's
     * own doc argues against having.
     *
     * iOS reaches the same place by a different route: CoreLocation reports
     * authorization changes to the delegate, including ones made in Settings,
     * so LocationSensor.swift re-arms from `locationManagerDidChangeAuthorization`
     * and needs no seam to Permissions at all. LocationManager has no such
     * callback, which is why this flag is here and not there.
     *
     * # And the window it used to leave open
     *
     * Go's last location event used to stay the refusal until the first fix
     * landed, which on cold GPS is tens of seconds — so a screen written to
     * the documented shape (`case !loc.Available: EmptyState{Hint: loc.Error}`)
     * printed "location permission not granted" about a sensor that was
     * working. A start that gets through now reports `acquiring: true`, which
     * is `core.LocationAcquiring`: it withdraws the refusal and puts the record
     * back into Active-with-nothing-received, which is the spinner state. The
     * coordinates are deliberately absent from that payload rather than sent as
     * zeros, which would be a real place off the coast of Ghana.
     */
    private var awaitingPermission = false

    /**
     * Both providers were switched off — location services are off system-wide
     * — and this object is waiting for one of them to come back.
     *
     * The twin of [awaitingPermission], and it exists for the same reason: the
     * user is expected to go and fix the thing that refused the start, and
     * nothing else will tell this object to try again. The difference is which
     * callback carries the good news. A permission answer arrives through
     * [Permissions], because LocationManager has no authorization callback; a
     * provider coming back arrives at [onProviderEnabled], which is
     * LocationListener's own.
     *
     * # Why that callback can fire at all, which it could not before
     *
     * `onProviderEnabled` only reaches a listener that is registered, and this
     * object used to register only with providers that were *already* enabled —
     * so the one case that needed the callback was the one case with nothing to
     * deliver it. [start] now registers with both providers whether or not they
     * are enabled, which is what LocationManager documents the callback pair
     * for, and decides separately whether any of them can actually answer.
     */
    private var awaitingProvider = false

    /**
     * Whether [this] is registered with LocationManager, which is NOT the same
     * as [running].
     *
     * A listener stays registered while [awaitingProvider] waits — that is the
     * whole mechanism — so `running` cannot be the flag [stop] consults before
     * calling removeUpdates, or a screen that unmounts while location services
     * are off would leave this object registered for the life of the process.
     */
    private var registered = false

    fun attach(context: Context, report: (String, String) -> Unit) {
        this.report = report
        appContext = context.applicationContext
        manager = context.applicationContext
            .getSystemService(Context.LOCATION_SERVICE) as? LocationManager
    }

    /**
     * Dispatches one "sensor" system event. Unknown kinds and commands are
     * dropped — both sensors are handed every event and each answers for its
     * own kind, which is what keeps SystemEvents' dispatch one line per sensor.
     */
    fun handle(data: JSONObject) {
        if (data.optString("kind") != "location") return
        when (data.optString("command")) {
            "start" -> start()
            "stop" -> stop()
        }
    }

    private fun start() {
        if (running) return
        val ctx = appContext
        val m = manager
        if (ctx == null || m == null) {
            send(JSONObject().put("available", false).put("error", "no location service"))
            return
        }
        if (!hasPermission(ctx)) {
            // Not a request: see the class comment. One event, and a screen can
            // draw the "ask me" state instead of a spinner that never ends.
            //
            // Armed rather than simply abandoned, which is the difference
            // between "no" and "not yet": the grant may arrive a tap later and
            // nothing else will tell this object to try again. See
            // [awaitingPermission].
            awaitingPermission = true
            send(
                JSONObject().put("available", false)
                    .put("error", "location permission not granted")
            )
            return
        }
        // Both providers, and this is not belt-and-braces. NETWORK_PROVIDER
        // answers in seconds with a coarse fix; GPS_PROVIDER can take a minute
        // outdoors and never answers indoors. Asking for one means either a slow
        // first fix or no fix in a building, and Go's consumer takes whichever
        // arrives — the accuracy field is what tells the two apart, which is
        // exactly what Location.Accuracy is documented for.
        //
        // Registered whether or not the provider is currently enabled, which is
        // the change that makes [awaitingProvider] possible: a disabled provider
        // sends no fixes but does send `onProviderEnabled` when it comes back,
        // and only to a listener that is registered with it. The old shape
        // skipped disabled providers, so the one case that needed the callback
        // was the one case with nothing registered to receive it. Whether any
        // provider can actually answer is a separate question, asked below.
        // A re-registration starts by tearing the old one down, and an
        // emulator run is why. [onProviderEnabled] reaches here with the
        // listener still registered from the original request — that is the
        // whole mechanism — and requesting again on top of it leaves the
        // platform's own MIN_DISTANCE_M filter holding the last fix as its
        // reference point. A device that has not MOVED five metres since
        // location services were switched off therefore gets no callback at
        // all, and the screen sits on "waiting for the first fix" until
        // somebody walks. Observed exactly that way: the recovery re-armed and
        // reported, a stop-and-start in the same session produced a fix
        // instantly, and the only difference between them was removeUpdates.
        if (registered) {
            registered = false
            try {
                m.removeUpdates(this)
            } catch (e: SecurityException) {
                Log.w(TAG, "removing location updates before re-registering", e)
            }
        }
        var live = false
        for (provider in listOf(LocationManager.NETWORK_PROVIDER, LocationManager.GPS_PROVIDER)) {
            try {
                m.requestLocationUpdates(
                    provider, MIN_INTERVAL_MS, MIN_DISTANCE_M, this, Looper.getMainLooper()
                )
                registered = true
                if (m.isProviderEnabled(provider)) live = true
            } catch (e: SecurityException) {
                // The permission was revoked between the check above and here,
                // which is a real sequence on Android: the user can revoke from
                // the notification shade while the app is running.
                Log.w(TAG, "location updates refused", e)
            } catch (e: IllegalArgumentException) {
                // No such provider on this device. Caught rather than guarded
                // against, because the guard that used to make it unreachable —
                // isProviderEnabled — is the one that had to go.
                Log.w(TAG, "no $provider on this device", e)
            }
        }
        if (!live) {
            // Armed, on the same argument as the permission: the user is
            // expected to go and switch location services on, and nothing else
            // would tell this object to try again. It can only be armed if
            // something is registered to hear the callback.
            awaitingProvider = registered
            send(
                JSONObject().put("available", false)
                    .put("error", "location is switched off")
            )
            return
        }
        running = true
        awaitingPermission = false
        awaitingProvider = false
        lastSentAt = 0L
        // The sensor is on and has nothing yet. Said out loud because Go's last
        // event may be a refusal this start has just made untrue, and a cold
        // first fix is tens of seconds away — see core.LocationAcquiring.
        send(JSONObject().put("acquiring", true))
    }

    /**
     * Retries a start that the permission refused, when the permission stops
     * refusing.
     *
     * Called by [Permissions] for every answer it sends to Go — a check's and a
     * request's alike, which is what also covers the user granting the
     * permission in the system settings and coming back, since
     * `hooks.UsePermissionLive` re-checks on every foreground.
     *
     * The [awaitingPermission] guard is what keeps this from being a second way
     * to turn the GPS on: a "granted" answer with no outstanding start is
     * somebody else's business — a camera screen checking its own permissions,
     * a debug readout — and starting the radio from it would be this framework
     * spending battery nobody asked for.
     */
    fun permissionAnswer(kind: String, status: String) {
        if (kind != "location" || status != "granted") return
        if (!awaitingPermission) return
        // start() re-reads the permission itself rather than trusting the
        // status it was handed: the two can disagree, because an answer posted
        // from the launcher crosses a thread hop and the user can revoke from
        // the notification shade inside it.
        start()
    }

    private fun stop() {
        // Cleared even when nothing is running, and that is the point: a screen
        // that unmounts while still waiting for the grant must not leave the
        // sensor armed, or a permission granted later for some other reason
        // would start a GPS with no consumer. core.StopLocation reaches here on
        // the hook's close path whether the sensor ever got going or not.
        awaitingPermission = false
        awaitingProvider = false
        running = false
        // [registered] rather than [running], because the two came apart when
        // the provider wait arrived: a listener registered with switched-off
        // providers is not running and must still be removed, or unmounting
        // that screen leaves this object attached to LocationManager for the
        // life of the process.
        if (!registered) return
        registered = false
        try {
            manager?.removeUpdates(this)
        } catch (e: SecurityException) {
            Log.w(TAG, "removing location updates", e)
        }
    }

    private fun hasPermission(ctx: Context): Boolean {
        // Either one will do: an app that declares the fine permission and is
        // granted it gets fixes, and this framework's manifest declares the
        // coarse one. Checking both means a downstream app that widened its own
        // manifest is not refused by this guard.
        for (permission in listOf(
            Manifest.permission.ACCESS_COARSE_LOCATION,
            Manifest.permission.ACCESS_FINE_LOCATION,
        )) {
            if (ContextCompat.checkSelfPermission(ctx, permission) ==
                PackageManager.PERMISSION_GRANTED
            ) {
                return true
            }
        }
        return false
    }

    // --- LocationListener ---------------------------------------------------

    override fun onLocationChanged(location: Location) {
        if (!running) return
        val now = System.currentTimeMillis()
        // A second throttle behind the platform's own, because two providers are
        // running: each honours the interval separately, so the stream reaching
        // here can be twice what either was asked for.
        if (now - lastSentAt < MIN_INTERVAL_MS) return
        lastSentAt = now

        val payload = JSONObject()
            .put("lat", location.latitude)
            .put("lng", location.longitude)
            .put("ts", now)
        // Omitted rather than sent as a zero, which is Go's missing-key
        // contract: hasAccuracy() is false on a fix from a provider that does
        // not measure one, and a 0-metre radius would read as a perfect fix.
        if (location.hasAccuracy()) {
            payload.put("accuracy", location.accuracy.toDouble())
        }
        if (location.hasAltitude()) {
            payload.put("altitude", location.altitude)
        }
        send(payload)
    }

    /**
     * A provider came back. LocationListener's other methods have default
     * implementations only from API 30, and this app's minSdk is 24 — so both
     * of these are declared here rather than inherited.
     *
     * This one used to report nothing, on the argument that one provider's
     * weather is not a fact about the device — which is right for the ordinary
     * case and left one case dead: [onProviderDisabled] had already given up
     * when BOTH went, and nothing could ever start again. [awaitingProvider] is
     * that case and only that case, so the ordinary enable still reports
     * nothing.
     *
     * [start] is called rather than the flag simply cleared, because the state
     * it has to rebuild is more than a boolean: the registrations, the throttle
     * clock, and the acquiring report that withdraws the "location is switched
     * off" a screen is currently displaying.
     */
    override fun onProviderEnabled(provider: String) {
        if (!awaitingProvider) return
        start()
    }

    override fun onProviderDisabled(provider: String) {
        // Unless both are gone, in which case no fix is coming and a screen
        // waiting on one should be told.
        val m = manager ?: return
        if (!running) return
        val live = listOf(LocationManager.NETWORK_PROVIDER, LocationManager.GPS_PROVIDER)
            .any { m.isProviderEnabled(it) }
        if (!live) {
            running = false
            // Armed for the recovery, which is the half that was missing: the
            // listener stays registered (nothing is removed here), so
            // [onProviderEnabled] will fire when the user switches location
            // services back on, and [awaitingProvider] is what tells it that
            // this object is the one waiting.
            awaitingProvider = true
            send(
                JSONObject().put("available", false)
                    .put("error", "location is switched off")
            )
        }
    }

    private fun send(payload: JSONObject) {
        val r = report
        if (r == null) {
            Log.w(TAG, "location fix with no host-event channel attached")
            return
        }
        r("location", payload.toString())
    }
}
