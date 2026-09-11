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
        var any = false
        for (provider in listOf(LocationManager.NETWORK_PROVIDER, LocationManager.GPS_PROVIDER)) {
            if (!m.isProviderEnabled(provider)) continue
            try {
                m.requestLocationUpdates(
                    provider, MIN_INTERVAL_MS, MIN_DISTANCE_M, this, Looper.getMainLooper()
                )
                any = true
            } catch (e: SecurityException) {
                // The permission was revoked between the check above and here,
                // which is a real sequence on Android: the user can revoke from
                // the notification shade while the app is running.
                Log.w(TAG, "location updates refused", e)
            }
        }
        if (!any) {
            send(
                JSONObject().put("available", false)
                    .put("error", "location is switched off")
            )
            return
        }
        running = true
        lastSentAt = 0L
    }

    private fun stop() {
        if (!running) return
        running = false
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
     * The deprecated three-argument callbacks. LocationListener's other methods
     * have default implementations only from API 30, and this app's minSdk is
     * 24 — so they are declared here rather than inherited.
     *
     * Nothing is reported from them. A provider going out of service is not the
     * same as location being unavailable (the other provider may be answering),
     * and Go's Available is a fact about the device rather than about one
     * provider's weather.
     */
    override fun onProviderEnabled(provider: String) {}

    override fun onProviderDisabled(provider: String) {
        // Unless both are gone, in which case no fix is coming and a screen
        // waiting on one should be told.
        val m = manager ?: return
        if (!running) return
        val live = listOf(LocationManager.NETWORK_PROVIDER, LocationManager.GPS_PROVIDER)
            .any { m.isProviderEnabled(it) }
        if (!live) {
            running = false
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
