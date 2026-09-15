package com.grmob.app

import android.content.Context
import android.os.Build
import android.os.VibrationEffect
import android.os.Vibrator
import android.os.VibratorManager
import org.json.JSONObject

/**
 * The Android half of core.Haptic (core/haptics.go): one named effect per
 * "haptic" system event, fire-and-forget.
 *
 * The Vibrator service rather than View.performHapticFeedback: the view API
 * needs a View (this object deliberately holds only the application context,
 * like the rest of [SystemEvents]) and honours the user's "touch feedback"
 * setting, which is about taps — a buzz for "an agent needs you" is closer to
 * a notification than to a keypress, and should not vanish because the user
 * turned off keyboard clicks. The cost is the VIBRATE permission, a normal
 * install-time grant declared in the manifest; without it vibrate() throws
 * SecurityException, which mobile/verify guards against.
 */
object Haptics {
    private var vibrator: Vibrator? = null

    fun attach(context: Context) {
        val app = context.applicationContext
        vibrator = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            // API 31 moved the service behind VibratorManager; the default
            // vibrator is the one the old service returned.
            (app.getSystemService(Context.VIBRATOR_MANAGER_SERVICE) as? VibratorManager)
                ?.defaultVibrator
        } else {
            @Suppress("DEPRECATION")
            app.getSystemService(Context.VIBRATOR_SERVICE) as? Vibrator
        }
    }

    fun handle(data: JSONObject) {
        val v = vibrator ?: return
        // A tablet or emulator with no motor: nothing to do, and vibrate()
        // would be a silent no-op anyway, but checking skips building effects.
        if (!v.hasVibrator()) return
        val kind = data.optString("kind")
        when {
            Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q -> predefined(v, kind)
            Build.VERSION.SDK_INT >= Build.VERSION_CODES.O -> shaped(v, kind)
            else -> legacy(v, kind)
        }
    }

    /**
     * API 29+: the platform's own tuned effects where one fits, so the buzz
     * feels like the rest of the system. Warning and error have no
     * predefined effect, so they are short waveforms distinct from success's
     * double click — two pulses and three.
     */
    private fun predefined(v: Vibrator, kind: String) {
        val effect = when (kind) {
            "selection", "light" -> VibrationEffect.createPredefined(VibrationEffect.EFFECT_TICK)
            "medium" -> VibrationEffect.createPredefined(VibrationEffect.EFFECT_CLICK)
            "heavy" -> VibrationEffect.createPredefined(VibrationEffect.EFFECT_HEAVY_CLICK)
            "success" -> VibrationEffect.createPredefined(VibrationEffect.EFFECT_DOUBLE_CLICK)
            "warning", "error" -> waveform(kind)
            else -> return
        }
        v.vibrate(effect)
    }

    /** API 26–28: no predefined effects, but amplitude-aware one-shots. */
    private fun shaped(v: Vibrator, kind: String) {
        val effect = when (kind) {
            "selection" -> VibrationEffect.createOneShot(5, VibrationEffect.DEFAULT_AMPLITUDE)
            "light" -> VibrationEffect.createOneShot(10, VibrationEffect.DEFAULT_AMPLITUDE)
            "medium" -> VibrationEffect.createOneShot(20, VibrationEffect.DEFAULT_AMPLITUDE)
            "heavy" -> VibrationEffect.createOneShot(30, VibrationEffect.DEFAULT_AMPLITUDE)
            "success", "warning", "error" -> waveform(kind)
            else -> return
        }
        v.vibrate(effect)
    }

    /** API 24–25 (minSdk): the deprecated millisecond calls are all there is. */
    @Suppress("DEPRECATION")
    private fun legacy(v: Vibrator, kind: String) {
        when (kind) {
            "selection" -> v.vibrate(5)
            "light" -> v.vibrate(10)
            "medium" -> v.vibrate(20)
            "heavy" -> v.vibrate(30)
            "success", "warning", "error" -> v.vibrate(timings(kind), -1)
        }
    }

    // The pulse patterns, shared by every API level that has waveforms and
    // matching the browser's table in grmob-runtime.js: off/on alternating,
    // starting with a zero delay; -1 means play once.
    private fun waveform(kind: String): VibrationEffect =
        VibrationEffect.createWaveform(timings(kind), -1)

    private fun timings(kind: String): LongArray = when (kind) {
        "success" -> longArrayOf(0, 10, 40, 10)
        "warning" -> longArrayOf(0, 20, 60, 20)
        else -> longArrayOf(0, 30, 40, 30, 40, 30) // error
    }
}
