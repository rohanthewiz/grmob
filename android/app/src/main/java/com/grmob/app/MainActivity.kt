package com.grmob.app

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import com.grmob.runtime.GrMobRoot
import com.grmob.runtime.GrMobRuntime

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // Edge to edge: the window stops fitting the system windows itself and
        // hands the insets to the composition, which is what makes them
        // readable at all — WindowInsets.ime and the safeDrawing padding on
        // the SafeArea node both report zero while the decor view is still
        // consuming them. Paired with android:windowSoftInputMode="adjustResize"
        // in the manifest, which is what makes the IME an inset rather than a
        // window pan. See Renderer.keyboardInset.
        enableEdgeToEdge()
        // The runtime mounts the initial Go-rendered tree and opens the push
        // channel; after that the composition tracks the TreeStore on its own.
        // Recreation (rotation, process restore) simply remounts from Go's
        // current state — the Go side is a process-wide singleton.
        val bridge = GomobileBridge(filesDir.absolutePath)
        val runtime = GrMobRuntime(bridge)
        // System events (toasts, external URLs, audio) are wired before
        // start() so an event emitted during the very first render pass has
        // a sink; without a listener Go drops them silently. The runtime is
        // constructed first because the audio player reports back through
        // it, but nothing renders until start(). See SystemEvents.kt.
        SystemEvents.attach(this, bridge, runtime)
        // Foreground/background transitions, reported through the same
        // host-event channel the audio player uses. Attached after start()
        // on purpose: the process observer fires ON_RESUME shortly after
        // this Activity resumes, and that report is only meaningful once
        // the tree exists to render whatever the app does with it. See
        // AppLifecycle.kt for why the process lifecycle and not this
        // Activity's.
        AppLifecycle.attach(runtime)
        runtime.start()
        setContent { GrMobRoot(runtime) }
    }
}
