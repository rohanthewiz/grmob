package com.grmob.app

import android.content.Intent
import androidx.media3.common.AudioAttributes
import androidx.media3.common.C
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.session.MediaSession
import androidx.media3.session.MediaSessionService

/**
 * The process's one audio player, hosted in a foreground service so that it
 * — and the media notification Media3 draws for it — outlives the activity.
 *
 * This is the half of grmob's audio service (core/audio.go) that Android
 * makes mandatory. A player owned by the activity keeps playing for a while
 * after the app is backgrounded, then dies with the process; a player owned
 * by a [MediaSessionService] is what gets a lock-screen card, headset button
 * handling, and the "audio" foreground-service type that keeps the process
 * alive while something is audibly happening. Everything the app can do to
 * the player goes through a [androidx.media3.session.MediaController]
 * connected to this service's session — see [AudioPlayer], which is the only
 * client.
 *
 * The service is declared in the manifest with the
 * `androidx.media3.session.MediaSessionService` intent filter, which is how
 * the controller finds it, and `foregroundServiceType="mediaPlayback"`,
 * which Android 14 requires before a service may promote itself while
 * playing. Media3 itself calls startForeground with its default media
 * notification; no notification code lives here.
 */
class GrMobAudioService : MediaSessionService() {
    private var session: MediaSession? = null

    override fun onCreate() {
        super.onCreate()
        val player = ExoPlayer.Builder(this)
            // Speech content and audio focus handled by the player: a phone
            // call or another app's playback pauses this one, and it resumes
            // (or ducks) when focus returns — the behavior a listener
            // expects from a podcast app without anyone writing it.
            .setAudioAttributes(
                AudioAttributes.Builder()
                    .setContentType(C.AUDIO_CONTENT_TYPE_SPEECH)
                    .setUsage(C.USAGE_MEDIA)
                    .build(),
                /* handleAudioFocus = */ true,
            )
            // Pause when headphones are unplugged rather than switching to
            // the speaker mid-sermon in a quiet room.
            .setHandleAudioBecomingNoisy(true)
            // Hold the CPU and the network while playing so a streamed file
            // keeps buffering with the screen off. Needs WAKE_LOCK in the
            // manifest.
            .setWakeMode(C.WAKE_MODE_NETWORK)
            .build()
        session = MediaSession.Builder(this, player).build()
    }

    override fun onGetSession(controllerInfo: MediaSession.ControllerInfo): MediaSession? = session

    /**
     * The user swiped the app away from recents. A paused player has nothing
     * to keep alive; a playing one is what the user is still listening to,
     * and the foreground notification is how they stop it.
     */
    override fun onTaskRemoved(rootIntent: Intent?) {
        val player = session?.player
        if (player == null || !player.playWhenReady || player.mediaItemCount == 0) {
            stopSelf()
        }
    }

    override fun onDestroy() {
        session?.run {
            player.release()
            release()
        }
        session = null
        super.onDestroy()
    }
}
