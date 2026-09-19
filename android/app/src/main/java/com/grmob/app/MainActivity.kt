package com.grmob.app

import android.annotation.SuppressLint
import android.content.Intent
import android.os.Bundle
import android.view.KeyEvent
import android.view.View
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
        // Opt out of force dark (API 29+). The theme is Theme.Material.Light
        // and Go draws every colour itself, so from the platform's view this
        // is a light app it may recolour. When force dark is on — the
        // developer override, or a vendor's system-wide dark mode (MIUI sets
        // debug.hwui.force_dark) — HWUI re-tints draw calls by its own
        // heuristics, and on Compose's output they misfire: the page went
        // dark while Cards stayed light, and black lesson titles were
        // lightened until they vanished on those light Cards (a Mi Max 3 on
        // Android 10). Colours are the Go theme's to choose, so the shell
        // draws them as given.
        //
        // The primary switch is now android:forceDarkAllowed in Theme.GrMob
        // (res/values/themes.xml), which turns force dark off for the window's
        // renderer. This per-view flag alone proved not enough: it is honoured
        // while HWUI walks the render node tree, and on the Mi Max 3 the walk
        // stopped honouring it once 4.12's MapView (an AndroidView) had been
        // attached, fading every later lesson's Card text. It stays as a
        // second statement of the same intent, and is harmless.
        if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.Q) {
            window.decorView.isForceDarkAllowed = false
        }
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
        // Window size and fold posture, as the "window" host event. Per
        // Activity rather than once per process like the lifecycle: a fold
        // or unfold recreates this Activity, and the new instance's window
        // is the one to measure. See AppWindow.kt.
        AppWindow.attach(this, runtime)
        runtime.start()
        this.runtime = runtime
        setContent { GrMobRoot(runtime) }
        // The URL this launch came from, if it came from one. After start(),
        // because the app cannot be navigated before its tree exists — the
        // subscriber lives in a hook slot and the hook has not run yet.
        // See core/deeplink.go.
        reportDeepLink(intent)
        // A tap on one of core's notifications that launched this process.
        // Only on a fresh create: a recreation (rotation) is handed the same
        // launch intent again, and re-reporting it would re-route the reader
        // to the banner they already opened.
        if (savedInstanceState == null) reportNotificationTap(intent)
    }

    // core.AccessibilityKeyShortcuts from a hardware keyboard. Offered to the
    // runtime before the window's own dispatch, so a declared chord is
    // answered wherever focus is, including nowhere; see
    // GrMobRuntime.handleKeyEvent.
    //
    // RestrictedApi is suppressed deliberately. dispatchKeyEvent is public
    // Activity API; androidx's core ComponentActivity overrides it and marks
    // its override @RestrictTo(LIBRARY_GROUP_PREFIX), so lint flags every
    // subclass that overrides it again, and the super call. Overriding it is
    // the documented way to see a key before the focused view does, and the
    // super call keeps androidx's own handling (its KeyEventDispatcher) in
    // the chain.
    @SuppressLint("RestrictedApi")
    override fun dispatchKeyEvent(event: KeyEvent): Boolean {
        if (runtime?.handleKeyEvent(event) == true) return true
        if (restoreFocusForNavigation(event)) return true
        return super.dispatchKeyEvent(event)
    }

    /**
     * Gives the window its focus back when a focus-navigation key arrives and
     * no View holds it, and answers whether the key was spent doing so.
     *
     * # How the window ends up with no focus
     *
     * Compose 1.7 clears the *View's* focus when the focused node leaves the
     * composition: FocusOwnerImpl.invalidateOwnerFocusState calls
     * AndroidComposeView.onClearFocusForOwner, which calls View.clearFocus.
     * Every GrMob navigation does that to whatever had focus, because Push,
     * Replace and Pop swap the screen's subtree: press "Next ›" from the
     * keyboard, or follow a deep link, and the focused button is gone. After
     * that, findFocus() is null and no Compose node hears a key.
     *
     * # Why the framework's own recovery is not enough
     *
     * ViewRootImpl answers an unhandled Tab or arrow with no focused View by
     * calling restoreDefaultFocus(), which is how the first Tab after a
     * navigation normally lands on the first control. With TalkBack running
     * that recovery never happened: on a Galaxy Z Fold6 (Android 16,
     * Samsung TalkBack 16.2) every Tab reached this Activity with no focused
     * View, went unhandled, and the next Tab found the window unchanged, from
     * both an injected key and a USB keyboard. Tab stayed dead until TalkBack
     * was turned off. Calling the same restoreDefaultFocus() here worked in
     * that state, so this does, ahead of the window's dispatch.
     *
     * # Why the key is consumed
     *
     * The framework's recovery consumes the key too (performFocusNavigation
     * returns true), and that is what puts the first Tab on the *first*
     * control. Passing it on as well would move focus a second time and skip
     * that control. Only the keys ViewRootImpl itself navigates with are
     * taken: Tab, with or without Shift, and the four arrows with no
     * modifier. A chord or any other key goes on as before.
     */
    private fun restoreFocusForNavigation(event: KeyEvent): Boolean {
        if (event.action != KeyEvent.ACTION_DOWN) return false
        val navigates = when (event.keyCode) {
            KeyEvent.KEYCODE_TAB ->
                event.hasNoModifiers() || event.hasModifiers(KeyEvent.META_SHIFT_ON)
            KeyEvent.KEYCODE_DPAD_UP, KeyEvent.KEYCODE_DPAD_DOWN,
            KeyEvent.KEYCODE_DPAD_LEFT, KeyEvent.KEYCODE_DPAD_RIGHT -> event.hasNoModifiers()
            else -> false
        }
        if (!navigates || window.decorView.findFocus() != null) return false
        // restoreDefaultFocus arrived in API 26 with android:focusedByDefault.
        // This app declares no default-focus View, and with none declared it
        // is requestFocus(FOCUS_DOWN), which is what API 24 and 25 get.
        if (android.os.Build.VERSION.SDK_INT < android.os.Build.VERSION_CODES.O) {
            return window.decorView.requestFocus(View.FOCUS_DOWN)
        }
        return window.decorView.restoreDefaultFocus()
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
        reportNotificationTap(intent)
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

    /**
     * Forwards a tap on one of core's notifications to Go as the
     * "notification_tap" host event, with the id it was posted under.
     *
     * Here and not in Notifications.kt because the tap's PendingIntent
     * launches this Activity: onCreate (cold launch) and onNewIntent (running
     * app, via singleTop) are the only two places it can be read. An intent
     * that did not come from a notification carries no id and is ignored.
     */
    private fun reportNotificationTap(intent: Intent?) {
        val id = Notifications.tapId(intent) ?: return
        runtime?.hostEvent("notification_tap", JSONObject().put("id", id).toString())
    }
}
