package com.grmob.app

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import com.grmob.runtime.GrMobRoot
import com.grmob.runtime.GrMobRuntime
import org.json.JSONObject

class MainActivity : ComponentActivity() {
    /**
     * Kept so [onNewIntent] can report a link, and so [onCreate] can report the
     * one it was launched with. Null until the runtime exists, which is why the
     * launch intent is reported at the end of onCreate rather than at the top.
     */
    private var runtime: GrMobRuntime? = null

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
        // Runtime permissions, which need the Activity SystemEvents does not
        // retain — and need it *here*, because registerForActivityResult
        // throws once the Activity is STARTED. See Permissions.kt.
        Permissions.attach(this, runtime::hostEvent)
        // Foreground/background transitions, reported through the same
        // host-event channel the audio player uses. Attached after start()
        // on purpose: the process observer fires ON_RESUME shortly after
        // this Activity resumes, and that report is only meaningful once
        // the tree exists to render whatever the app does with it. See
        // AppLifecycle.kt for why the process lifecycle and not this
        // Activity's.
        AppLifecycle.attach(runtime)
        runtime.start()
        this.runtime = runtime
        setContent { GrMobRoot(runtime) }
        // The URL this launch came from, if it came from one. After start(),
        // because the app cannot be navigated before its tree exists — the
        // subscriber lives in a hook slot and the hook has not run yet.
        // See core/deeplink.go.
        reportDeepLink(intent)
    }

    /**
     * A link arriving while the app is already running.
     *
     * Reached because the manifest sets launchMode="singleTop": without it the
     * system would start a second MainActivity instead and this would never
     * fire. The Activity's own `intent` is also replaced, so a later
     * recreation (a rotation) does not re-report the intent this launched with
     * — which would navigate the reader away from wherever they had got to.
     */
    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        reportDeepLink(intent)
    }

    /**
     * Forwards one ACTION_VIEW intent's data to Go as the "deeplink" host
     * event, verbatim.
     *
     * Verbatim is the contract: a URL means whatever the app that registered
     * the scheme says it means, so parsing it here would put an app's
     * vocabulary in the shell. core/deeplink.go carries that argument.
     *
     * Everything that is not a VIEW with data is ignored, which is every
     * launch from the home screen (ACTION_MAIN, no data) and every intent some
     * other component might deliver.
     */
    private fun reportDeepLink(intent: Intent?) {
        if (intent?.action != Intent.ACTION_VIEW) return
        val url = intent.data?.toString() ?: return
        if (url.isEmpty()) return
        runtime?.hostEvent("deeplink", JSONObject().put("url", url).toString())
    }
}
