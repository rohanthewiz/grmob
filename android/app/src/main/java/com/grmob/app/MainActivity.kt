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
        // System events (toasts, external URLs) are wired before start() so
        // an event emitted during the very first render pass has a sink;
        // without a listener Go drops them silently. See SystemEvents.kt.
        SystemEvents.attach(this, bridge)
        val runtime = GrMobRuntime(bridge)
        runtime.start()
        setContent { GrMobRoot(runtime) }
    }
}
