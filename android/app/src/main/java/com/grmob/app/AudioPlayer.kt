package com.grmob.app

import android.content.ComponentName
import android.content.Context
import android.net.Uri
import android.os.Handler
import android.os.Looper
import android.util.Log
import androidx.core.content.ContextCompat
import androidx.media3.common.C
import androidx.media3.common.MediaItem
import androidx.media3.common.MediaMetadata
import androidx.media3.common.PlaybackException
import androidx.media3.common.PlaybackParameters
import androidx.media3.common.Player
import androidx.media3.session.MediaController
import androidx.media3.session.SessionToken
import com.google.common.util.concurrent.ListenableFuture
import org.json.JSONObject

/**
 * The Android half of grmob's audio service (core/audio.go): commands in as
 * the "audio" system event, status out as the "audio_status" host event.
 *
 *     core.AudioLoad/Play/... ──"audio" {command}──▶ handle ──▶ MediaController
 *     core.CurrentAudioStatus ◀──"audio_status"──── report ◀── Player.Listener + ticker
 *
 * The player itself lives in [GrMobAudioService]; this object holds a
 * [MediaController] connected to it. The connection is asynchronous, so the
 * first commands after launch — typically a single "load" — queue until it
 * completes and then apply in order. Everything here runs on the main
 * thread: MediaController is bound to the looper it was built on, and
 * [SystemEvents] already hops there before dispatching.
 *
 * Status is reported on every player callback that changes what the app
 * could show (state, error, speed, item) and on a 500ms ticker while
 * playing, which is what moves a seek bar. Each report is a full status,
 * never a delta, so a dropped one costs nothing: the next tick says
 * everything again.
 */
object AudioPlayer {
    private const val TAG = "GrMobAudio"
    private const val TICK_MS = 500L

    private val main = Handler(Looper.getMainLooper())
    private var appContext: Context? = null
    private var report: ((String, String) -> Unit)? = null

    private var controllerFuture: ListenableFuture<MediaController>? = null
    private var controller: MediaController? = null
    private val pending = ArrayDeque<JSONObject>()

    /** The loaded track's URL as Go named it; "" when nothing is loaded. */
    private var url = ""
    private var errorText = ""
    private var ticking = false

    /**
     * Wires the reporter. [report] receives (eventName, payloadJSON) and is
     * expected to be the runtime's host-event dispatcher, which serializes
     * the delivery with render passes and applies the resulting patches.
     */
    fun attach(context: Context, report: (String, String) -> Unit) {
        appContext = context.applicationContext
        this.report = report
    }

    /** Dispatches one "audio" system event. Main thread only. */
    fun handle(data: JSONObject) {
        val c = controller
        if (c == null) {
            pending.addLast(data)
            connect()
            return
        }
        apply(c, data)
    }

    private fun connect() {
        if (controllerFuture != null) return
        val context = appContext ?: return
        val token = SessionToken(context, ComponentName(context, GrMobAudioService::class.java))
        val future = MediaController.Builder(context, token).buildAsync()
        controllerFuture = future
        future.addListener({
            val c = try {
                future.get()
            } catch (e: Exception) {
                // The service could not be bound — a manifest without the
                // declaration, most likely. Every queued command is dropped
                // and the app told, so it does not sit on "loading" forever.
                Log.e(TAG, "could not connect to GrMobAudioService", e)
                controllerFuture = null
                errorText = "audio service unavailable"
                pending.clear()
                emit(null)
                return@addListener
            }
            controller = c
            c.addListener(listener)
            while (pending.isNotEmpty()) apply(c, pending.removeFirst())
        }, ContextCompat.getMainExecutor(context))
    }

    private fun apply(c: MediaController, data: JSONObject) {
        when (data.optString("command")) {
            "load" -> load(c, data)
            "play" -> {
                // After the end, play means "from the top": ExoPlayer stays
                // in STATE_ENDED on play() alone.
                if (c.playbackState == Player.STATE_ENDED) c.seekTo(0)
                c.play()
            }
            "pause" -> c.pause()
            "seek" -> c.seekTo(secondsToMs(data.optDouble("position", 0.0)))
            "skip" -> c.seekTo(clampMs(c, c.currentPosition + secondsToMs(data.optDouble("delta", 0.0))))
            "rate" -> {
                val rate = data.optDouble("rate", 1.0).toFloat()
                if (rate > 0f) c.playbackParameters = PlaybackParameters(rate)
            }
            "stop" -> {
                c.stop()
                c.clearMediaItems()
                // The controller keeps its playback parameters across
                // items; Go's record resets to 1 on stop, so the report
                // that follows must agree with it.
                c.playbackParameters = PlaybackParameters(1f)
                url = ""
                errorText = ""
                emit(c)
            }
        }
    }

    private fun load(c: MediaController, data: JSONObject) {
        val u = data.optString("url")
        if (u.isEmpty()) return
        url = u
        errorText = ""
        val metadata = MediaMetadata.Builder()
            .setTitle(data.optString("title").ifEmpty { null })
            .setArtist(data.optString("artist").ifEmpty { null })
            .setAlbumTitle(data.optString("album").ifEmpty { null })
            .setArtworkUri(data.optString("artwork").ifEmpty { null }?.let(Uri::parse))
            .build()
        val item = MediaItem.Builder()
            .setUri(u)
            .setMediaId(u)
            .setMediaMetadata(metadata)
            .build()
        val rate = data.optDouble("rate", 1.0).toFloat()
        c.setMediaItem(item, secondsToMs(data.optDouble("start", 0.0)))
        c.playbackParameters = PlaybackParameters(if (rate > 0f) rate else 1f)
        c.playWhenReady = data.optBoolean("autoplay", true)
        c.prepare()
        emit(c)
    }

    // ---- Status -----------------------------------------------------------

    private val listener = object : Player.Listener {
        override fun onIsPlayingChanged(isPlaying: Boolean) {
            controller?.let { emit(it) }
            if (isPlaying) startTicking() else stopTicking()
        }

        override fun onPlaybackStateChanged(playbackState: Int) {
            controller?.let { emit(it) }
        }

        override fun onPlayerError(error: PlaybackException) {
            errorText = error.message ?: "playback error ${error.errorCode}"
            Log.w(TAG, "playback error for $url", error)
            controller?.let { emit(it) }
        }

        override fun onPlaybackParametersChanged(playbackParameters: PlaybackParameters) {
            controller?.let { emit(it) }
        }

        override fun onPositionDiscontinuity(
            oldPosition: Player.PositionInfo,
            newPosition: Player.PositionInfo,
            reason: Int,
        ) {
            controller?.let { emit(it) }
        }
    }

    private val tick = object : Runnable {
        override fun run() {
            if (!ticking) return
            controller?.let { emit(it) }
            main.postDelayed(this, TICK_MS)
        }
    }

    private fun startTicking() {
        if (ticking) return
        ticking = true
        main.postDelayed(tick, TICK_MS)
    }

    private fun stopTicking() {
        ticking = false
        main.removeCallbacks(tick)
    }

    /**
     * Folds the player's several flags into core's one AudioState. Buffering
     * (also the state right after prepare) is "loading"; ready splits on
     * whether the player intends to play, so a stall that ExoPlayer is about
     * to recover from still reads as playing rather than flickering.
     */
    private fun state(c: MediaController?): String {
        if (errorText.isNotEmpty()) return "error"
        if (url.isEmpty() || c == null || c.mediaItemCount == 0) return "idle"
        return when (c.playbackState) {
            Player.STATE_BUFFERING -> "loading"
            Player.STATE_ENDED -> "ended"
            Player.STATE_READY -> if (c.playWhenReady) "playing" else "paused"
            else -> "paused" // STATE_IDLE with an item: stopped, or a cleared error
        }
    }

    private fun emit(c: MediaController?) {
        val duration = c?.duration ?: C.TIME_UNSET
        val payload = JSONObject()
            .put("url", url)
            .put("state", state(c))
            .put("position", (c?.currentPosition ?: 0L).coerceAtLeast(0L) / 1000.0)
            .put("duration", if (duration == C.TIME_UNSET) 0.0 else duration / 1000.0)
            .put("rate", (c?.playbackParameters?.speed ?: 1f).toDouble())
            .put("error", errorText)
        report?.invoke("audio_status", payload.toString())
    }

    private fun secondsToMs(seconds: Double): Long = (seconds * 1000).toLong().coerceAtLeast(0L)

    private fun clampMs(c: MediaController, ms: Long): Long {
        val d = c.duration
        val upper = if (d == C.TIME_UNSET) Long.MAX_VALUE else d
        return ms.coerceIn(0L, upper)
    }
}
