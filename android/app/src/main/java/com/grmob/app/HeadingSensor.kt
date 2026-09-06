package com.grmob.app

import android.content.Context
import android.hardware.Sensor
import android.hardware.SensorEvent
import android.hardware.SensorEventListener
import android.hardware.SensorManager
import android.util.Log
import org.json.JSONObject
import kotlin.math.abs
import kotlin.math.roundToInt

/**
 * The Android half of grmob's compass (core/heading.go).
 *
 *   core.StartHeading ──▶ "sensor" {kind:"heading", command:"start"} ──▶ here
 *   core.CurrentHeading ◀── "heading" host event ◀── here
 *
 * # Which sensor
 *
 * TYPE_ROTATION_VECTOR, not TYPE_MAGNETIC_FIELD or the long-deprecated
 * TYPE_ORIENTATION. The rotation vector is a *fused* sensor: the platform
 * combines the magnetometer with the gyroscope and accelerometer, which is
 * what makes the reading stable while the phone is moving and what keeps it
 * meaningful when the phone is not flat. Reading the magnetometer directly
 * would mean reimplementing that fusion here, badly.
 *
 * TYPE_GEOMAGNETIC_ROTATION_VECTOR is the lower-power alternative (no
 * gyroscope) and is the fallback when the fused one is absent, which it is on
 * some cheap hardware.
 *
 * # No permission
 *
 * Unlike location, the compass needs none: rotation vector is not a
 * runtime-permission sensor on any API level. That is the whole reason Tier C
 * is a smaller job than the location work that reuses its shape.
 */
object HeadingSensor : SensorEventListener {
    private const val TAG = "GrMobHeading"

    /**
     * Throttle floor. The rotation vector delivers at the rate the sensor
     * hardware runs — SENSOR_DELAY_UI is a hint, not a contract, and on some
     * devices means 50 Hz or more. Every event that reaches Go costs a full
     * render pass, so the stream is cut to roughly 15 Hz here, at the host
     * edge, rather than after crossing the bridge.
     */
    private const val MIN_INTERVAL_MS = 66L

    /**
     * Low-pass smoothing factor for the reported bearing, in [0, 1]: the
     * fraction of each new reading that is taken. A magnetometer jitters by a
     * degree or two even on a still desk, and an unsmoothed needle visibly
     * twitches.
     *
     * 0.25 settles a real turn within a few frames while damping the noise.
     * The filter runs on the *unwrapped* delta (see [smooth]) — averaging 359
     * and 1 arithmetically would give 180, sending the needle due south every
     * time the user crossed north.
     */
    private const val SMOOTHING = 0.25f

    private var manager: SensorManager? = null
    private var sensor: Sensor? = null
    private var report: ((String, String) -> Unit)? = null

    private var running = false
    private var lastSentAt = 0L
    private var smoothed = Float.NaN
    private var accuracyDegrees = -1.0

    private val rotation = FloatArray(9)
    private val orientation = FloatArray(3)

    /**
     * Remembers how to reach this device's sensors and how to answer Go.
     * Call once at startup, before [handle] can be reached by an event.
     *
     * @param report the host→app channel, normally GrMobRuntime::hostEvent.
     */
    fun attach(context: Context, report: (String, String) -> Unit) {
        this.report = report
        manager = context.applicationContext
            .getSystemService(Context.SENSOR_SERVICE) as? SensorManager
        sensor = manager?.let {
            it.getDefaultSensor(Sensor.TYPE_ROTATION_VECTOR)
                ?: it.getDefaultSensor(Sensor.TYPE_GEOMAGNETIC_ROTATION_VECTOR)
        }
    }

    /** Dispatches one "sensor" system event. Unknown kinds and commands are dropped. */
    fun handle(data: JSONObject) {
        if (data.optString("kind") != "heading") return
        when (data.optString("command")) {
            "start" -> start()
            "stop" -> stop()
        }
    }

    private fun start() {
        if (running) return
        val m = manager
        val s = sensor
        if (m == null || s == null) {
            // The honest answer for a device with no compass — a tablet, an
            // emulator without sensor emulation. One event, and Go stops
            // waiting for a reading that is never coming.
            send(JSONObject().put("available", false).put("error", "no compass on this device"))
            return
        }
        running = true
        smoothed = Float.NaN
        lastSentAt = 0L
        // SENSOR_DELAY_UI rather than _FASTEST or _GAME: this drives a dial a
        // person is looking at, and the difference between the rates is
        // battery spent on events the throttle above then discards.
        m.registerListener(this, s, SensorManager.SENSOR_DELAY_UI)
    }

    private fun stop() {
        if (!running) return
        running = false
        manager?.unregisterListener(this)
    }

    override fun onSensorChanged(event: SensorEvent) {
        if (!running) return

        SensorManager.getRotationMatrixFromVector(rotation, event.values)
        SensorManager.getOrientation(rotation, orientation)
        // getOrientation answers azimuth in radians over (-pi, pi], measured
        // counter-clockwise-negative from north. Degrees and the fold into
        // [0, 360) both happen on the Go side (core.ReceiveHeading), which is
        // the one place all four hosts funnel through — but the smoothing
        // below needs a continuous number, so the conversion happens here too.
        val degrees = Math.toDegrees(orientation[0].toDouble()).toFloat()
        val next = smooth(degrees)

        val now = System.currentTimeMillis()
        if (now - lastSentAt < MIN_INTERVAL_MS) return
        lastSentAt = now

        val payload = JSONObject()
            .put("magnetic", next.toDouble())
            .put("ts", now)
        if (accuracyDegrees >= 0) payload.put("accuracy", accuracyDegrees)
        send(payload)
    }

    /**
     * Exponential smoothing that respects the seam at north.
     *
     * The filter is applied to the *shortest signed turn* from the last value
     * to the new one rather than to the values themselves, so a step from 359
     * to 1 is treated as the +2 degrees it is instead of as -358. Without
     * this the needle swings the long way round the rose every time the user
     * passes north, which is the single most visible bug a compass can have.
     */
    private fun smooth(degrees: Float): Float {
        if (smoothed.isNaN()) {
            smoothed = degrees
            return smoothed
        }
        var delta = (degrees - smoothed) % 360f
        if (delta > 180f) delta -= 360f
        if (delta <= -180f) delta += 360f
        smoothed += SMOOTHING * delta
        // Keep the accumulator bounded; Go normalises what it is sent, and an
        // unbounded float would lose precision after enough laps.
        if (abs(smoothed) > 720f) smoothed %= 360f
        return smoothed
    }

    override fun onAccuracyChanged(sensor: Sensor?, accuracy: Int) {
        // Android reports a bucket, not an angle. The mapping to degrees is
        // the platform's own documented guidance for what each bucket means
        // for a compass, so Go and the other two hosts can compare like with
        // like — iOS and Safari both answer in degrees.
        accuracyDegrees = when (accuracy) {
            SensorManager.SENSOR_STATUS_ACCURACY_HIGH -> 5.0
            SensorManager.SENSOR_STATUS_ACCURACY_MEDIUM -> 15.0
            SensorManager.SENSOR_STATUS_ACCURACY_LOW -> 30.0
            // UNRELIABLE means the magnetometer needs the figure-eight
            // calibration wave. Reported as a large angle rather than as
            // "unknown": the app should be able to tell its user the reading
            // is bad, and -1 would say only that nobody measured.
            SensorManager.SENSOR_STATUS_UNRELIABLE -> 180.0
            else -> -1.0
        }
    }

    private fun send(payload: JSONObject) {
        val r = report
        if (r == null) {
            Log.w(TAG, "heading reading with no host-event channel attached")
            return
        }
        r("heading", payload.toString())
    }

    /** Rounded degrees, for logging and tests. */
    internal fun debugBearing(): Int =
        if (smoothed.isNaN()) -1 else ((smoothed % 360f + 360f) % 360f).roundToInt()
}
