package com.grmob.runtime

import android.app.Activity
import android.content.Context
import android.content.ContextWrapper
import android.graphics.drawable.ColorDrawable
import androidx.compose.foundation.LocalIndication
import androidx.compose.foundation.text.selection.LocalTextSelectionColors
import androidx.compose.material3.LocalContentColor
import androidx.compose.material3.LocalTextStyle
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.SideEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
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
 *   the Material scheme = dark over a dark colour, light otherwise (N-086;
 *                        see ShellMaterialTheme)
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

/** The window surface: the root node's Background, else [ShellPage]. */
internal fun shellPage(root: GrMobNode): Color = root.style?.background ?: ShellPage

/**
 * The colour under the bars: the last (innermost) painted SafeArea's claim,
 * else the window surface. The one answer ShellSurface's icons and
 * ShellMaterialTheme's scheme are both taken from.
 */
internal fun barsColor(root: GrMobNode, claims: List<SystemBarClaim>): Color =
    claims.lastOrNull()?.color?.takeIf { it.isSpecified } ?: shellPage(root)

/**
 * Dark means light content over it. Luminance ≤ 0.5, the complement of the
 * > 0.5 the platform's own helpers use to pick dark content for a background.
 */
internal fun isDarkBars(color: Color): Boolean = color.luminance() <= 0.5f

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
    val page = shellPage(root)
    fun underBars(): Color = barsColor(root, claims)
    underBars() // the subscribing read; see above
    SideEffect {
        val window = view.context.findActivity()?.window ?: return@SideEffect
        val argb = page.toArgb()
        if ((window.decorView.background as? ColorDrawable)?.color != argb) {
            window.setBackgroundDrawable(ColorDrawable(argb))
        }
        val lightBars = !isDarkBars(underBars())
        val controller = WindowCompat.getInsetsController(window, view)
        // "Appearance light" names the bar, not the icons: a light bar is
        // one that wants dark icons.
        controller.isAppearanceLightStatusBars = lightBars
        controller.isAppearanceLightNavigationBars = lightBars
    }
}

/**
 * The Material colour scheme under the tree: dark when the colour under the
 * bars is dark, light otherwise (N-086).
 *
 * # Why
 *
 * Material pieces the Go tree does not colour take their colours from
 * MaterialTheme.colorScheme, and text with no stated ink from
 * LocalContentColor. Neither was ever provided, so both stayed at their
 * defaults — lightColorScheme() and Color.Black — whatever the page was. The
 * iOS shell overrides the window's interface style from the same colour that
 * picks the bar icons (GrMobSurface.swift), so SwiftUI's chrome and `.primary`
 * ink turn light-on-dark over a dark page. Seen 2026-10-03 with the demo's
 * Screen painted #1C1C1E on a light system: the TabView's tab row was a pale
 * Material strip and the header's unstated ink black on Android, both dark /
 * white on iOS. This puts Android on the same rule:
 *
 *   isDarkBars(barsColor)   scheme                ink with no TextColor
 *   ─────────────────────   ───────────────────   ─────────────────────
 *   false                   lightColorScheme()    Color.Black
 *   true                    darkColorScheme()     Color.White
 *
 * The light row is exactly what composed before: the defaults of
 * LocalColorScheme and LocalContentColor. Black and white rather than the
 * schemes' onBackground (#1D1B20 / #E6E0E9) because the light row has to stay
 * black to change nothing, and white is what iOS's `.primary` is when dark.
 * Like the iOS override this is window-wide: one answer for the whole tree,
 * not one per subtree. An app that colours everything (the tutorial) sees no
 * difference either way; what changes is only what the Go tree left unstated.
 *
 * # Only the colour scheme
 *
 * material3's LocalColorScheme is internal, so the scheme can only be set
 * through MaterialTheme(). MaterialTheme provides more than the scheme,
 * though, and none of the rest was provided before (checked against the
 * material3 1.3.1 bytecode the BOM resolves):
 *
 *   LocalIndication          the ripple, where clickables drew the foundation
 *                            default
 *   LocalTextSelectionColors the scheme's primary, where selection drew the
 *                            foundation default
 *   LocalTextStyle           typography.bodyLarge (16sp, 24sp line height,
 *                            0.5sp tracking), where Text drew the default style
 *   LocalRippleTheme         read only by the deprecated fallback ripple, which
 *                            nothing here enables; left as MaterialTheme sets it
 *
 * The first three would have changed every app's press feedback and every
 * text's metrics, light ones included. So their values are read from outside
 * MaterialTheme and provided again inside it, which leaves the scheme as the
 * only thing MaterialTheme changes.
 *
 * # Structure and recomposition
 *
 * MaterialTheme and the provider wrap the content in both states, so a flip
 * between light and dark changes values, not the group structure, and the
 * tree's remembered state survives it. `dark` is a derivedStateOf over the
 * root's Background and the claims, so a claim joining or a restyle that does
 * not cross the threshold recomposes nothing. One that does recomposes the
 * whole tree: material3's LocalColorScheme is a static composition local,
 * which invalidates everything below its provider rather than its readers.
 * That is a page turning dark or light, rare enough to pay for in full.
 *
 * # The first frame
 *
 * Unlike ShellSurface's icons, the scheme has to be known while the tree
 * composes, and on the first composition the SafeAreas' claims have not joined
 * yet (they join in a DisposableEffect). So a dark SafeArea over an unpainted
 * root composes light for one frame and then dark. A painted root — what a
 * scheme-following app states — is known from the start and has no such frame.
 */
@Composable
internal fun ShellMaterialTheme(
    root: () -> GrMobNode?,
    claims: List<SystemBarClaim>,
    content: @Composable () -> Unit,
) {
    // `root` is a lambda over TreeStore's state, read inside derivedStateOf so
    // that the derivation, not GrMobRoot, is subscribed to a root swap and to
    // the root's style (both snapshot state). rememberUpdatedState keeps the
    // remembered derivation calling the caller's latest lambda.
    val currentRoot by rememberUpdatedState(root)
    val dark by remember(claims) {
        derivedStateOf { currentRoot()?.let { isDarkBars(barsColor(it, claims)) } ?: false }
    }
    val indication = LocalIndication.current
    val selection = LocalTextSelectionColors.current
    val textStyle = LocalTextStyle.current
    MaterialTheme(colorScheme = if (dark) DarkShellScheme else LightShellScheme) {
        CompositionLocalProvider(
            LocalIndication provides indication,
            LocalTextSelectionColors provides selection,
            LocalTextStyle provides textStyle,
            LocalContentColor provides if (dark) Color.White else Color.Black,
            content = content,
        )
    }
}

/*
 * The two schemes, built once. Material's baseline schemes, uncustomised: the
 * light one is what LocalColorScheme defaulted to before, and the dark one is
 * its counterpart. Shared instances keep MaterialTheme's providers equal from
 * one recomposition to the next.
 */
private val LightShellScheme = lightColorScheme()
private val DarkShellScheme = darkColorScheme()

/**
 * The ink Material draws on a colour the Go tree stated — a Button's
 * Background, a Checkbox's or Switch's AccentColor — when the tree states no
 * ink for it: the tick, the checked thumb, the label.
 *
 * Material takes that ink from the scheme's onPrimary, which pairs with the
 * scheme's own primary, not with a Go colour. In the light scheme it is
 * white, which is what every such control drew before ShellMaterialTheme. In
 * the dark scheme it is a deep purple (#381E72), meant for the dark scheme's
 * pale primary; on a Go accent it drew a dark purple thumb on the demo's blue
 * track. So where the colour underneath is Go's, the ink stays the light
 * scheme's onPrimary in both schemes, and only controls the Go tree leaves
 * wholly uncoloured take the dark scheme's pair.
 */
internal val OnGoColor: Color = LightShellScheme.onPrimary

/** The Activity behind a Compose view's context, which may be wrapped. */
internal tailrec fun Context.findActivity(): Activity? = when (this) {
    is Activity -> this
    is ContextWrapper -> baseContext.findActivity()
    else -> null
}
