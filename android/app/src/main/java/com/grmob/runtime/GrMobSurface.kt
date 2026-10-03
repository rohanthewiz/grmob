package com.grmob.runtime

import android.app.Activity
import android.content.Context
import android.content.ContextWrapper
import android.graphics.drawable.ColorDrawable
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.SideEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshots.SnapshotStateList
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.isSpecified
import androidx.compose.ui.graphics.luminance
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalView
import androidx.core.view.WindowCompat

/*
 * The shell's surface and the system bars, both taken from the Go tree
 * rather than from the system's dark mode (N-085).
 *
 * The window is the Go app's page. Colours are the Go theme's to choose (see
 * MainActivity's force-dark opt-out), and an app that does not follow
 * core.Window.ColorScheme stays light on a dark phone. Two things here did
 * follow the system all the same:
 *
 *   - enableEdgeToEdge's automatic bar style, which turns the status-bar and
 *     navigation-bar icons white in night mode. The window under them is
 *     still light (Theme.Material.Light), so a light app on a dark phone drew
 *     white icons on a near-white strip (seen on the emulator).
 *     The same automatic style was wrong the other way before N-085: a
 *     SafeArea with a dark background on a phone in light mode drew dark
 *     icons on a dark strip, and the clock and battery vanished. That is
 *     why the SafeArea took the icons over first (below).
 *   - The window background, Theme.Material.Light's #FAFAFA rather than the
 *     page colour the Go theme states.
 *
 * Both now come from one answer, the colour under the bars:
 *
 *   the bars' colour   = the innermost painted SafeArea's Background
 *                        (a SafeArea paints under the bars: GrMobColumn's
 *                        background precedes its inset padding)
 *                        ?: the root node's Background
 *                        ?: ShellPage (core.DefaultTheme's Background)
 *   the window surface = the root node's Background ?: ShellPage
 *   the icons          = dark over a light colour, light over a dark one
 *
 * An app that follows the scheme already paints its root (the tutorial's
 * paintPage does, with DarkTheme's Background), so it gets light icons when
 * dark; one that does not states no root colour and gets the light page and
 * dark icons whatever the system says. The iOS shell answers the same
 * question the same way (GrMobSurface.swift).
 *
 * # Why the SafeArea claims through a list rather than setting the bars
 *
 * The SafeArea arm used to set the icons itself, from a SideEffect, and the
 * last SideEffect to run won. That held while only SafeAreas set them. With
 * the root setting them too, a pass that recomposed the root and skipped an
 * unchanged SafeArea below it would have let the root's colour overwrite the
 * SafeArea's. So a painted SafeArea now registers a claim while it is
 * composed, and one composable, ShellSurface, reads the claims and the root
 * together and writes the window once. Claims join in composition order, so
 * the last one is the innermost or most recently entered SafeArea — the same
 * SafeArea the old last-SideEffect-wins rule picked on a first composition.
 */

/**
 * The page colour behind a tree whose root states none: core.DefaultTheme's
 * Background (#FFFFFF), and the colour of every light bundled theme's page.
 */
internal val ShellPage = Color.White

/** A painted SafeArea's hold on the bars, for as long as it is composed. */
internal class SystemBarClaim {
    var color by mutableStateOf(Color.Unspecified)
}

/**
 * The claims of the SafeAreas composed under the current GrMobRoot. Null
 * outside one (a preview, a test composing a node by itself), where a
 * SafeArea claims nothing and the bars are left alone.
 */
internal val LocalSystemBarClaims = staticCompositionLocalOf<SnapshotStateList<SystemBarClaim>?> { null }

/** A fresh claim list for a GrMobRoot to provide. */
internal fun systemBarClaims(): SnapshotStateList<SystemBarClaim> = mutableStateListOf()

/**
 * Registers a SafeArea's background as the colour under the bars. The claim
 * joins the list when the SafeArea enters the composition and leaves it when
 * the SafeArea does. Its colour is set on joining, and after that follows the
 * background through SideEffect, so a restyled SafeArea updates its claim in
 * place rather than leaving and rejoining (which would move it to the end of
 * the list).
 */
@Composable
internal fun ClaimSystemBars(background: Color) {
    val claims = LocalSystemBarClaims.current ?: return
    val claim = remember { SystemBarClaim() }
    SideEffect { claim.color = background }
    DisposableEffect(claims, claim) {
        claim.color = background
        claims.add(claim)
        onDispose { claims.remove(claim) }
    }
}

/**
 * Writes the window: its background drawable and the bar icon style. A
 * composable of its own, beside the tree rather than around it, so that a
 * claim arriving or the root's Background changing recomposes this alone and
 * not the tree. GrMobRoot composes it *after* the tree, which the first frame
 * depends on (below).
 *
 * SideEffect rather than LaunchedEffect: the window is not Compose state and
 * must follow every successful composition, including the first, with no
 * frame of the wrong colour in between. Cheap to repeat — the controller only
 * touches the window when a value changes, and the background drawable is
 * replaced only when its colour does.
 *
 * The colour is read twice, on purpose. Once during composition, which is
 * what subscribes this scope to the claim list and the last claim's colour,
 * so a SafeArea arriving, leaving or restyling recomposes it. And again
 * inside the SideEffect, which is the value written. On the first
 * composition the composition-time read sees no claims at all: the
 * SafeAreas' DisposableEffects have not run yet. An apply dispatches every
 * remembered effect before any SideEffect, though, and SideEffects in
 * composition order, so by the time this one runs the claims have joined and
 * their colours are set. Reading inside it is what puts the right icons on
 * the first frame rather than the second.
 *
 * Luminance > 0.5 is the threshold the platform's own helpers use to pick
 * light or dark content for a background.
 */
@Composable
internal fun ShellSurface(root: GrMobNode, claims: List<SystemBarClaim>) {
    val view = LocalView.current
    if (view.isInEditMode) return
    val page = root.style?.background ?: ShellPage
    fun underBars(): Color = claims.lastOrNull()?.color?.takeIf { it.isSpecified } ?: page
    underBars() // the subscribing read; see above
    SideEffect {
        val window = view.context.findActivity()?.window ?: return@SideEffect
        val argb = page.toArgb()
        if ((window.decorView.background as? ColorDrawable)?.color != argb) {
            window.setBackgroundDrawable(ColorDrawable(argb))
        }
        val lightBars = underBars().luminance() > 0.5f
        val controller = WindowCompat.getInsetsController(window, view)
        // "Appearance light" names the bar, not the icons: a light bar is
        // one that wants dark icons.
        controller.isAppearanceLightStatusBars = lightBars
        controller.isAppearanceLightNavigationBars = lightBars
    }
}

/** The Activity behind a Compose view's context, which may be wrapped. */
internal tailrec fun Context.findActivity(): Activity? = when (this) {
    is Activity -> this
    is ContextWrapper -> baseContext.findActivity()
    else -> null
}
