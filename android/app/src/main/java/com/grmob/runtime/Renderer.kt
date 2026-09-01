package com.grmob.runtime

import androidx.compose.animation.Animatable
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsFocusedAsState
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.exclude
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.ime
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawing
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.compositionLocalOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import coil.compose.AsyncImage

/**
 * Node-tree → Compose mapping.
 *
 * The design deliberately leans on Compose's own reconciler for everything the
 * Go reconciler doesn't do: view identity across recompositions (`key()` per
 * child), state retention in unchanged siblings, animation plumbing, and
 * accessibility semantics (via material3 components). The Go side only has to
 * keep the data tree correct; nothing here caches views or paths.
 */
val LocalGrMobRuntime = compositionLocalOf<GrMobRuntime> {
    error("GrMobRoot not mounted")
}

/**
 * Inherited disabled state.
 *
 * A node's own `Style.Disabled` covers the common case — a button, a field —
 * but "disable this whole section while the form submits" is the other half
 * of what the flag is for, and Compose has no subtree-wide equivalent of
 * SwiftUI's `.disabled(true)` or CSS's `pointer-events: none`, both of which
 * propagate. This CompositionLocal is that equivalent, so one Go declaration
 * means the same thing on all three targets rather than "the node only" here
 * and "the node and everything under it" there.
 *
 * It is one-way: nothing re-enables a subtree inside a disabled ancestor,
 * which is also how the two platform mechanisms it mirrors behave.
 */
val LocalGrMobDisabled = compositionLocalOf { false }

/** This node's effective disabled state: its own flag, or an ancestor's. */
@Composable
private fun GrMobNode.isDisabled(): Boolean =
    style?.disabled == true || LocalGrMobDisabled.current

@Composable
fun GrMobRoot(runtime: GrMobRuntime) {
    CompositionLocalProvider(LocalGrMobRuntime provides runtime) {
        runtime.store.root?.let { RenderNode(it) }
    }
}

/**
 * `extra` carries parent-scope modifiers (Row/Column weight) that only the
 * parent can construct; see GrMobStyle.boxModifier.
 */
@Composable
fun RenderNode(node: GrMobNode, extra: Modifier = Modifier) {
    if (node.style?.display == "none") return // not composed at all; "hidden" keeps space

    // core.KeyboardAware, applied at the one funnel every node passes through
    // rather than at each container that might want it. It lands between the
    // parent-scope modifiers and the node's own (boxModifier starts from
    // `extra`), which is exactly where it has to be for a Scroll: constraints
    // flow left to right, so an inset written before verticalScroll shrinks
    // the *viewport*, while one written after would pad the scrolled content
    // and leave the viewport still claiming the rows the keyboard covers.
    val mods = extra.keyboardInset(node)

    // Opening the disabled scope here, once, rather than inside every control
    // means a container's flag reaches leaves it does not know about — and
    // the provider is skipped when nothing changes, so an enabled tree pays
    // nothing for the mechanism.
    if (node.style?.disabled == true && !LocalGrMobDisabled.current) {
        CompositionLocalProvider(LocalGrMobDisabled provides true) {
            RenderNodeContent(node, mods)
        }
    } else {
        RenderNodeContent(node, mods)
    }
}

@Composable
private fun RenderNodeContent(node: GrMobNode, extra: Modifier) {
    val style = node.style

    when (node.type) {
        "Text" -> GrMobText(node, extra)
        "Button" -> GrMobButton(node, extra)

        "Input" -> GrMobTextField(node, extra)
        "InputPassword" -> GrMobTextField(node, extra, password = true)
        "NumericInput" -> GrMobTextField(node, extra, numeric = true)
        "TextArea" -> GrMobTextField(node, extra, multiline = true)
        "Checkbox" -> GrMobCheckbox(node, extra)

        "Row" -> GrMobRow(node, extra)
        "Column", "Card" -> GrMobColumn(node, extra) // Card = Column whose Go theme style carries the card look
        "List" -> GrMobList(node, extra)
        "Box" -> Box(animatedStyle(style).boxModifier(extra, gestureModifier(node))) { RenderChildren(node) }
        "Spacer" -> Spacer(Modifier.size(node.intProp("size").dp))
        "Scroll" -> Column(
            style.boxModifier(extra).verticalScroll(rememberScrollState())
        ) { ColumnChildren(node) }

        // The safe area is the system bars and the display cutout — not the
        // IME. WindowInsets.safeDrawing bundles the keyboard in with the rest,
        // which would make every screen resize whole whenever a field is
        // focused and, worse, consume the IME inset so that a Scroll asking
        // for it with keyboardInset would receive nothing (nested window-inset
        // modifiers consume what they apply). Keeping the keyboard out of this
        // is what leaves it to the region that actually scrolls — the same
        // split SwiftUI makes, where the keyboard is its own safe-area region.
        "SafeArea" -> Box(
            style.boxModifier(extra)
                .windowInsetsPadding(WindowInsets.safeDrawing.exclude(WindowInsets.ime))
        ) { RenderChildren(node) }

        "TabView" -> GrMobTabView(node, extra)
        "Modal" -> GrMobModal(node)
        "Image" -> AsyncImage(
            model = node.stringProp("src"),
            contentScale = contentScaleFor(node.stringProp("contentMode")),
            // The Go-side AccessibilityLabel style prop is the first choice
            // for the description; the legacy "alt" prop remains a fallback.
            // The label is delivered through AsyncImage's own semantics slot,
            // so it is stripped from the style handed to boxModifier — both
            // paths setting contentDescription would double-announce.
            contentDescription = (style?.accessibilityLabel ?: "")
                .ifEmpty { node.stringProp("alt") }.ifEmpty { null },
            modifier = style?.copy(accessibilityLabel = "", accessibilityHint = "")
                .boxModifier(extra, gestureModifier(node)),
        )

        // Camera capture needs a CameraX integration pass of its own; until
        // then render the styled surface and any overlay so layouts hold up.
        "CameraView" -> Box(style.boxModifier(extra)) { RenderChildren(node) }

        // Fragment and Theme are grouping nodes with no visual box of their
        // own: emit the children inline into whatever scope we're in.
        "Fragment", "Theme" -> RenderChildren(node)

        // Unknown node type (newer Go core than this runtime): render the
        // children so the subtree isn't a dead end.
        else -> Column(style.boxModifier(extra)) { ColumnChildren(node) }
    }
}

/** Children of a non-flex container; keyed so reorder/replace keeps sibling state. */
@Composable
private fun RenderChildren(node: GrMobNode) {
    node.children.forEachIndexed { i, child ->
        key(child.key.ifEmpty { i }) { RenderNode(child) }
    }
}

/**
 * The color half of Transition support: when the style declares a
 * Transition, the background is replaced by a value animated toward the
 * current target, so an update-style patch fades the color instead of
 * snapping it. A highlight *appearing* on a previously unpainted node (the
 * feed row selection pattern) fades in as pure alpha rather than popping —
 * see the hue-snap note inside.
 *
 * This lives at the composition layer (not boxModifier) because the animated
 * value is remembered state tied to the node's composition position — the
 * size half of Transition (animateContentSize) is a plain modifier and lives
 * in boxModifier. A `replace` patch swaps the node instance and therefore
 * the composition position, so replaced nodes snap — which matches the Go
 * reconciler's intent (replace = a different thing, not a changed one).
 */
@Composable
private fun animatedStyle(s: GrMobStyle?): GrMobStyle? {
    if (s == null || s.transitionMs <= 0) return s

    // Driven by hand (Animatable rather than animateColorAsState) because a
    // null background must fade as pure alpha, never through gray:
    // Color.Transparent is transparent BLACK, and interpolating straight at
    // it darkens the highlight before it disappears — measured on-device as
    // (169,171,175) mid-fade between white and #E8F0FE. So an appearing
    // color first snaps invisibly to "target at alpha 0" (fixing the hue),
    // and a disappearing one animates to "current hue at alpha 0".
    val bg = remember { Animatable(s.background ?: Color.Transparent) }
    LaunchedEffect(s.background, s.transitionMs) {
        val target = s.background
        if (target != null) {
            if (bg.value.alpha == 0f) bg.snapTo(target.copy(alpha = 0f))
            bg.animateTo(target, s.transitionTween())
        } else if (bg.value.alpha > 0f) {
            bg.animateTo(bg.value.copy(alpha = 0f), s.transitionTween())
        }
    }
    return s.copy(background = bg.value)
}

/**
 * Go's core.KeyboardAware: shrink this node by the software keyboard's height
 * while the keyboard is showing.
 *
 * On a scrolling node that means the viewport ends above the keys, so
 * Compose's own bring-the-focused-field-into-view has somewhere visible to put
 * the field. On a non-scrolling one it means the whole subtree lifts — which
 * is how a screen with a docked composer keeps it reachable.
 *
 * It reads WindowInsets.ime, which only reports a height once the window has
 * stopped fitting the system windows itself:
 *
 *   MainActivity          enableEdgeToEdge()   (insets become the app's job)
 *   AndroidManifest.xml   windowSoftInputMode="adjustResize"
 *
 * Both are set. Were they not, the platform would resize the whole window and
 * deliver the IME inset already consumed, making this a no-op rather than a
 * conflict — the app would still yield to the keyboard, just wholesale.
 */
private fun Modifier.keyboardInset(node: GrMobNode): Modifier =
    if (node.boolProp("keyboardAware")) this.imePadding() else this

/**
 * Tap/long-press wiring for nodes that don't draw their own control (Button
 * and the inputs handle their own interaction). Empty when the node carries
 * neither callback, so plain content pays nothing. combinedClickable also
 * registers the click/long-click accessibility actions, so a gesture-bearing
 * row is activatable from TalkBack, not just by touch.
 */
@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun gestureModifier(node: GrMobNode): Modifier {
    val onClick = node.stringProp("onClick")
    val onLongPress = node.stringProp("onLongPress")
    if (onClick.isEmpty() && onLongPress.isEmpty()) return Modifier
    // A disabled node keeps its callback IDs (Go still has the handlers
    // registered — see core.Style.Disabled) and simply stops being
    // clickable, which also removes the click/long-click accessibility
    // actions. Dropping the modifier rather than passing
    // `combinedClickable(enabled = false)` also drops the ripple, which is
    // what a disabled surface should look like.
    if (node.isDisabled()) return Modifier
    val runtime = LocalGrMobRuntime.current
    return Modifier.combinedClickable(
        onClick = { if (onClick.isNotEmpty()) runtime.click(onClick) },
        onLongClick = if (onLongPress.isEmpty()) null else {
            { runtime.click(onLongPress) }
        },
    )
}

/**
 * core.ContentMode -> Compose ContentScale. An absent or unknown mode is Fit,
 * which is both core.Image's documented default and what this renderer drew
 * before the prop existed, so existing trees are unchanged.
 *
 * ContentScale.None (Center) leaves the bitmap at its intrinsic pixel size;
 * AsyncImage's default alignment is already Center, so "no scaling, centered"
 * needs nothing further.
 *
 * Every mode core declares is listed explicitly, including "fit" — which the
 * `else` arm below would already have handled. That redundancy is the point:
 * `else` swallows an unrecognized mode silently, so a fifth ContentMode added
 * to core would render here as Fit while both DOM targets fell back to the
 * browser default, and nothing anywhere would fail. Listing the modes makes
 * the coverage readable from outside, and mobile/verify/contentmode_test.go
 * reads it — it compares these arms with core.ContentModes() under a plain
 * `go test ./...`. Keep the arms one per line, string-literal first, and keep
 * the `else` arm; that is the shape the parse requires.
 */
private fun contentScaleFor(mode: String): ContentScale = when (mode) {
    "fit" -> ContentScale.Fit
    "fill" -> ContentScale.Crop
    "stretch" -> ContentScale.FillBounds
    "center" -> ContentScale.None
    // Absent (core.imageNode omits the prop entirely) or a mode this build of
    // the runtime predates.
    else -> ContentScale.Fit
}

// ---------------------------------------------------------------------------
// Leaf components
// ---------------------------------------------------------------------------

@Composable
private fun GrMobText(node: GrMobNode, extra: Modifier) {
    val s = animatedStyle(node.style)
    Text(
        text = node.stringProp("content"),
        modifier = s.boxModifier(extra, gestureModifier(node)),
        style = textStyle(s),
    )
}

private fun textStyle(s: GrMobStyle?): TextStyle {
    if (s == null) return TextStyle.Default
    return TextStyle(
        color = s.textColor ?: Color.Unspecified,
        fontSize = if (s.fontSize > 0f) s.fontSize.sp else TextStyle.Default.fontSize,
        // Go's Weight constants are the CSS numeric scale (200/400/700), which
        // FontWeight accepts directly.
        fontWeight = if (s.fontWeight > 0) FontWeight(s.fontWeight) else null,
        lineHeight = if (s.lineHeight > 0) s.lineHeight.sp else TextStyle.Default.lineHeight,
        // core.TextAlignments -> Compose's TextAlign. Every value spelled out,
        // including the one `else` would have produced, because Compose has no
        // "unset" to fall through to: an unlisted value is not left alone, it
        // is silently rendered as Start. Held to core.TextAlignments() by
        // TestKotlinTextAlignCoversEveryTextAlignment in mobile/verify, which
        // parses these arms out of textStyle — one per line, string literals first, `else ->`
        // last, and the arm duplicating `else`'s body stays spelled out.
        //
        // AlignStretch and AlignBaseline are absent by design: they are
        // Alignments naming a cross-axis placement, not a text alignment, and
        // core.TextAlignments() leaves them out for that reason.
        textAlign = when (s.align) {
            "start" -> TextAlign.Start
            "center" -> TextAlign.Center
            "end" -> TextAlign.End
            "justify" -> TextAlign.Justify
            else -> TextAlign.Start
        },
    )
}

@Composable
private fun GrMobButton(node: GrMobNode, extra: Modifier) {
    val runtime = LocalGrMobRuntime.current
    val s = node.style
    val onClick = node.stringProp("onClick")
    // Style properties the Go theme owns are fed into material3's slots
    // instead of boxModifier: Button draws its own container, so background/
    // radius/padding must go through its API to keep ripple + a11y correct.
    Button(
        onClick = { if (onClick.isNotEmpty()) runtime.click(onClick) },
        modifier = marginAndSize(s, extra),
        // The platform disabled state: material3 stops dispatching, drops the
        // ripple, and marks the node disabled for TalkBack.
        enabled = !node.isDisabled(),
        shape = RoundedCornerShape((s?.borderRadius ?: 8f).dp),
        colors = ButtonDefaults.buttonColors(
            containerColor = s?.background ?: Color.Unspecified,
            contentColor = s?.textColor ?: Color.Unspecified,
            // material3 would otherwise paint its own disabled tones (the
            // container color at 12% alpha) over whatever the Go theme chose,
            // so a widget that styles its own disabled look — components.
            // Button spends Surface/TextSecondary on it — would be silently
            // overruled. Feeding the same colors into both slots keeps Go the
            // single source of truth for the palette.
            disabledContainerColor = s?.background ?: Color.Unspecified,
            disabledContentColor = s?.textColor ?: Color.Unspecified,
        ),
        contentPadding = androidx.compose.foundation.layout.PaddingValues(
            start = (s?.padding?.left ?: 16).dp, top = (s?.padding?.top ?: 10).dp,
            end = (s?.padding?.right ?: 16).dp, bottom = (s?.padding?.bottom ?: 10).dp,
        ),
    ) {
        Text(
            node.stringProp("label"),
            fontSize = if ((s?.fontSize ?: 0f) > 0f) s!!.fontSize.sp else 17.sp,
            fontWeight = if ((s?.fontWeight ?: 0) > 0) FontWeight(s!!.fontWeight) else null,
        )
    }
}

/** Margin + explicit dimensions only — for components that draw their own box. */
private fun marginAndSize(s: GrMobStyle?, extra: Modifier): Modifier {
    if (s == null) return extra
    // Reuse boxModifier's ordering by building a margin/size-only style.
    val trimmed = s.copy(
        background = null, borderColor = null, borderWidth = 0f,
        borderRadius = 0f, shadow = 0f,
        padding = GrMobStyle.Edges(0, 0, 0, 0),
    )
    return trimmed.boxModifier(extra)
}

@Composable
private fun GrMobCheckbox(node: GrMobNode, extra: Modifier) {
    val runtime = LocalGrMobRuntime.current
    val cb = node.stringProp("onToggle")
    Checkbox(
        checked = node.boolProp("checked"),
        onCheckedChange = { if (cb.isNotEmpty()) runtime.toggled(cb, it) },
        modifier = marginAndSize(node.style, extra),
        enabled = !node.isDisabled(),
    )
}

/**
 * The controlled-input compromise: Go owns the value, but the IME needs its
 * keystrokes echoed instantly, and the Go round trip is asynchronous. So the
 * field is locally-owned *while focused* (every edit is sent upstream but
 * late echoes never snap the cursor back), and Go-owned when not focused
 * (an async upstream change — validation rewrites, state restores — lands
 * the moment the user isn't mid-typing).
 *
 * The one upstream change that must land mid-focus is a deliberate rewrite —
 * Go clearing the draft after a submit, a validator normalizing the text.
 * Echoes and rewrites are told apart by bookkeeping, not heuristics: every
 * value this field sends upstream is queued, and an upstream change that
 * matches a queued entry is an echo of our own edit (drop the queue through
 * that point — Go may coalesce renders, skipping intermediate values), while
 * one that matches nothing we sent can only be Go speaking for itself, so it
 * wins even while focused. Moving the cursor then is correct: the text under
 * it was replaced.
 */
@Composable
private fun GrMobTextField(
    node: GrMobNode,
    extra: Modifier,
    password: Boolean = false,
    numeric: Boolean = false,
    multiline: Boolean = false,
) {
    val runtime = LocalGrMobRuntime.current
    val s = node.style
    val upstream = node.stringProp("value")
    val onChange = node.stringProp("onChange")
    val onSubmit = node.stringProp("onSubmit")
    // The keyboard's action key, decided in Go from core.UseFocusOrder: "next"
    // on every field of a declared order but the last. Go also wired the
    // onSubmit above to advance the focus, so this prop only has to choose the
    // label — the action itself is an ordinary submit dispatch.
    val imeAction = node.stringProp("imeAction")
    val onFocus = node.stringProp("onFocus")
    val onBlur = node.stringProp("onBlur")
    // The imperative half: core.Focus / core.DismissKeyboard reach the screen
    // as these two props. See the LaunchedEffect below and core/focus.go.
    val focusEpoch = node.intProp("focusEpoch")
    val focusAction = node.stringProp("focusAction")

    val focusRequester = remember { FocusRequester() }
    val focusManager = LocalFocusManager.current
    val interactions = remember { MutableInteractionSource() }
    val focused by interactions.collectIsFocusedAsState()
    var text by remember { mutableStateOf(upstream) }
    val pendingEchoes = remember { mutableListOf<String>() }
    var lastUpstream by remember { mutableStateOf(upstream) }

    if (upstream != lastUpstream) {
        lastUpstream = upstream
        if (focused) {
            val echo = pendingEchoes.indexOf(upstream)
            if (echo >= 0) {
                repeat(echo + 1) { pendingEchoes.removeAt(0) }
            } else {
                text = upstream
                pendingEchoes.clear()
            }
        }
    }
    if (!focused) {
        // Go-owned while blurred; any queued echoes died with the focus session.
        pendingEchoes.clear()
        if (text != upstream) text = upstream
    }

    // The focus edges, dispatched to Go on the void channel like onSubmit.
    //
    // In a LaunchedEffect rather than inline in the composition body: sending
    // an event to Go wakes a render pass, and a composable's body may run any
    // number of times for one logical change (recomposition, and Compose is
    // free to re-run it speculatively). The effect keys on `focused`, so it
    // runs once per actual transition.
    //
    // seenFocus is what keeps mount quiet. collectIsFocusedAsState starts at
    // false, so the effect's first run is an unfocused field that never had
    // focus to lose; without the flag every text field on screen would fire
    // onBlur the moment it appeared. SwiftUI's onChange(of:) skips the initial
    // value for us and needs no equivalent — see the iOS renderer.
    var seenFocus by remember { mutableStateOf(false) }
    LaunchedEffect(focused) {
        if (focused) {
            seenFocus = true
            if (onFocus.isNotEmpty()) runtime.click(onFocus)
        } else if (seenFocus) {
            seenFocus = false
            if (onBlur.isNotEmpty()) runtime.click(onBlur)
        }
    }

    // Go's focus *commands*, the other direction from the edges above.
    //
    // Keyed on the epoch alone, never on the action: the action is what to
    // do, the epoch is when. Two identical prop maps produce no patch and so
    // no recomposition, which is exactly why Go bumps a counter rather than
    // setting a flag — a second core.Focus on the already-focused field has
    // to re-fire, and only a changed key can express that.
    //
    // Epoch 0 means no command has ever been issued (Go stamps nothing at
    // all), so the effect's mandatory first run does nothing on a screen that
    // never touches focus.
    //
    // Unlike the edge effect above this deliberately has NO mount guard. A
    // field that mounts while it is already the target takes focus, which is
    // what makes "push a screen and put the cursor in its search box" work:
    // the command is issued in the handler that navigates, and the field it
    // names does not exist until the pass after. The cost is that returning
    // to a screen re-applies the last command that named its field; issue a
    // core.DismissKeyboard if that is not wanted.
    //
    // "blur" is guarded on this field actually holding focus so that one
    // dismiss does not have every field on screen calling clearFocus(). Only
    // the target acts on "focus"; every other field is told "" and does
    // nothing, because requesting focus over there already takes it from
    // here — having both sides act would make the outcome depend on the
    // order Compose happens to run two effects in.
    LaunchedEffect(focusEpoch) {
        if (focusEpoch == 0) return@LaunchedEffect
        when (focusAction) {
            // requestFocus throws if the requester is not attached to a
            // placed node yet. A LaunchedEffect already runs after
            // composition, which covers the ordinary case; the catch covers
            // the field being composed but not yet placed (inside a lazy
            // list that has not laid out the row), where the honest outcome
            // is "the command missed" rather than a crashed screen.
            "focus" -> runCatching { focusRequester.requestFocus() }
            "blur" -> if (focused) focusManager.clearFocus()
        }
    }

    val rows = node.intProp("rows")
    // Appended after the box modifier, so the requester sits on the innermost
    // element — the field itself — rather than on the padding around it.
    var modifier = s.boxModifier(extra).focusRequester(focusRequester)
    if (multiline && rows > 0) {
        // Approximate a rows-based min height from the line height.
        val line = if (s != null && s.fontSize > 0f) s.fontSize * 1.4f else 24f
        modifier = modifier.heightIn(min = (line * rows).dp)
    }

    BasicTextField(
        value = text,
        // A disabled field refuses focus outright, so the IME never opens and
        // the focused/blurred bookkeeping above simply stays in its blurred
        // branch — Go-owned, which is the correct reading of an inert field.
        enabled = !node.isDisabled(),
        onValueChange = {
            text = it
            if (onChange.isNotEmpty()) {
                pendingEchoes.add(it)
                runtime.textChanged(onChange, it)
            }
        },
        modifier = modifier,
        interactionSource = interactions,
        textStyle = textStyle(s),
        singleLine = !multiline,
        visualTransformation =
            if (password) PasswordVisualTransformation() else VisualTransformation.None,
        keyboardOptions = KeyboardOptions(
            keyboardType = when {
                numeric -> KeyboardType.Number
                password -> KeyboardType.Password
                else -> KeyboardType.Text
            },
            // A submit-carrying field advertises Done so the IME's action key
            // reads as "act on this", mirroring the iOS submitLabel; a field
            // with somewhere to go advertises Next instead, so the key reads
            // "move on" and the platform draws the arrow users expect.
            //
            // Next is tested first because it is the more specific claim: Go
            // only stamps it on a field whose onSubmit it wired itself, so the
            // two can never disagree about what the key does.
            imeAction = when {
                imeAction == "next" -> ImeAction.Next
                onSubmit.isNotEmpty() -> ImeAction.Done
                else -> ImeAction.Default
            },
        ),
        // The IME action dispatches onSubmit as a plain void event — the same
        // channel as a Button tap. Both arms dispatch the same ID: Compose
        // routes the callback by which action the field advertised, so a field
        // showing Next arrives here as onNext and one showing Done as onDone,
        // and Go has already decided which handler that ID points at.
        //
        // Deliberately NOT LocalFocusManager.moveFocus(FocusDirection.Next):
        // that would walk Compose's own focus graph, which is derived from
        // layout and knows nothing about the order the Go code declared.
        //
        // A multiline field never gets here — Compose gives a non-singleLine
        // BasicTextField a return key that inserts a newline, which is the
        // right call. A TextArea in a focus order simply does not advance.
        keyboardActions = KeyboardActions(
            onDone = { if (onSubmit.isNotEmpty()) runtime.click(onSubmit) },
            onNext = { if (onSubmit.isNotEmpty()) runtime.click(onSubmit) },
        ),
        decorationBox = { inner ->
            Box {
                if (text.isEmpty()) {
                    Text(
                        node.stringProp("placeholder"),
                        style = textStyle(s).copy(color = Color(0x993C3C43)),
                    )
                }
                inner()
            }
        },
    )
}

// ---------------------------------------------------------------------------
// Flex containers
// ---------------------------------------------------------------------------

@Composable
private fun GrMobRow(node: GrMobNode, extra: Modifier) {
    val s = animatedStyle(node.style)
    Row(
        modifier = s.boxModifier(extra.then(stretchRowHeight(s)), gestureModifier(node)),
        horizontalArrangement = horizontalArrangement(s),
        // Held to core.AlignItemsValues() by
        // TestKotlinRowAlignmentCoversEveryAlignItems in mobile/verify: one arm
        // per line, string literals first, `else ->` last, and the two arms
        // duplicating `else`'s body stay spelled out.
        //
        // No `align` fallback here, unlike the Column below. Style.Align is a
        // text-alignment concept and has never been read for a Row's vertical
        // axis; Renderer.swift draws the same line (GrMobFlexStack consults it
        // only when the axis is vertical), so the two natives agree.
        verticalAlignment = when (s?.alignItems) {
            "flex-start" -> Alignment.Top
            "center" -> Alignment.CenterVertically
            "flex-end" -> Alignment.Bottom
            // Not placement. A stretched child is given the full height by
            // RowChildren's fillMaxHeight, so nothing is left for the row's own
            // alignment to place. Listed so "handled elsewhere" is
            // distinguishable from "not handled".
            "stretch" -> Alignment.Top
            else -> Alignment.Top
        },
    ) { RowChildren(node) }
}

@Composable
private fun GrMobColumn(node: GrMobNode, extra: Modifier) {
    val s = animatedStyle(node.style)
    Column(
        modifier = s.boxModifier(extra, gestureModifier(node)),
        verticalArrangement = verticalArrangement(s),
        // AlignItems governs cross-axis placement; the DSL's simpler Align
        // ("start"/"center"/"end") acts as a fallback when AlignItems is unset,
        // which is why this dispatch answers for two vocabularies at once — the
        // bare "start"/"end" labels are core.AlignStart and core.AlignEnd, not
        // AlignItems values. Held to core.AlignItemsValues() by
        // TestKotlinColumnAlignmentCoversEveryAlignItems in mobile/verify; one
        // arm per line, string literals first, `else ->` last.
        horizontalAlignment = when (s?.alignItems?.ifEmpty { s.align }) {
            "flex-start", "start" -> Alignment.Start
            "center" -> Alignment.CenterHorizontally
            "flex-end", "end" -> Alignment.End
            // Given the full width by ColumnChildren's fillMaxWidth, so there
            // is nothing left to place. See the Row above.
            "stretch" -> Alignment.Start
            else -> Alignment.Start
        },
    ) { ColumnChildren(node) }
}

/**
 * The children loops live inside RowScope/ColumnScope because FlexGrow maps
 * onto Modifier.weight, which exists only as a scope extension — the parent
 * computes it and hands it down as the child's `extra` modifier.
 *
 * The other parent-computed piece is `AlignItems: "stretch"`. Compose's
 * Alignment.Horizontal/Vertical enums have no stretch member — a stretched
 * item is not *placed* differently, it is *measured* differently — so the
 * container cannot express it and the children carry a fill modifier
 * instead. That is the same shape as weight: a cross-axis decision only the
 * parent knows, applied to the child.
 *
 * Weight and fill compose cleanly because they act on different axes: in a
 * Column, weight distributes height and fillMaxWidth sets width.
 */
@Composable
private fun RowScope.RowChildren(node: GrMobNode) {
    val stretch = isStretch(node.style)
    node.children.forEachIndexed { i, child ->
        key(child.key.ifEmpty { i }) {
            val grow = child.style?.flexGrow ?: 0f
            var m: Modifier = if (grow > 0f) Modifier.weight(grow) else Modifier
            if (stretch) m = m.fillMaxHeight()
            RenderNode(child, m)
        }
    }
}

@Composable
private fun ColumnScope.ColumnChildren(node: GrMobNode) {
    val stretch = isColumnStretch(node.style)
    node.children.forEachIndexed { i, child ->
        key(child.key.ifEmpty { i }) {
            val grow = child.style?.flexGrow ?: 0f
            var m: Modifier = if (grow > 0f) Modifier.weight(grow) else Modifier
            if (stretch) m = m.fillMaxWidth()
            RenderNode(child, m)
        }
    }
}

/**
 * Whether a container stretches its children across the cross axis.
 *
 * Two spellings, because the Style.Align fallback is axis-dependent and this
 * is where Android used to disagree with iOS. GrMobFlexStack in Renderer.swift
 * reads Align as the cross-axis value only when the stack's axis is vertical —
 * Align is a text-alignment concept and has never been read for a Row's
 * vertical axis — so `Align(AlignStretch)` on a Column stretches on iOS and on
 * a Row does not. Compose read `alignItems` alone in both places, so the
 * Column case silently did nothing here while it worked there.
 *
 * The alignment coverage tests in mobile/verify could not have caught this and
 * still cannot: this is an equality test, not a dispatch, so there are no arms
 * to hold up against a list. It is the same class of bug those tests exist for
 * and a reminder that they only reach the switches.
 */
private fun isStretch(s: GrMobStyle?): Boolean = s?.alignItems == "stretch"

private fun isColumnStretch(s: GrMobStyle?): Boolean =
    s?.alignItems?.ifEmpty { s.align } == "stretch"

/**
 * The container half of a stretched Row.
 *
 * CSS stretches a row's items to the container's content height, which for an
 * auto-height row is the tallest item. A bare fillMaxHeight on the children
 * would instead resolve against whatever maximum the *parent* handed down —
 * usually the whole screen — so the row has to be pinned to its tallest child
 * first. IntrinsicSize.Max is exactly that measurement.
 *
 * Skipped when the style names an explicit height: that height is already the
 * definite container size CSS would stretch against, and an intrinsic
 * measurement would fight the frame boxModifier applies for it.
 *
 * Two combinations to avoid, both fixable by giving the Row an explicit
 * Height (which takes the early exit above):
 *
 *  - A List inside a stretched Row. Intrinsic measurement asks every child
 *    for its own intrinsic height, which a lazy list cannot answer —
 *    LazyColumn/LazyRow throw rather than materialize every row to measure.
 *  - A stretched Row that is itself a FlexGrow child of a Column. The parent
 *    hands it a weight-derived height and this modifier overrides it with the
 *    tallest child's, so the row hugs instead of filling its share. The
 *    conflict cannot be detected here: `extra` is opaque, and the same
 *    FlexGrow inside a *Row* parent sets width, where the intrinsic height is
 *    still exactly right.
 */
private fun stretchRowHeight(s: GrMobStyle?): Modifier =
    if (isStretch(s) && s?.height.isNullOrEmpty()) Modifier.height(IntrinsicSize.Max) else Modifier

/**
 * The virtualized sibling of GrMobColumn: LazyColumn composes only the rows
 * on screen and recycles compositions by contentType as the user scrolls, so
 * Go can hand over a thousand-row feed as plain data.
 *
 * Go's For helper wraps generated rows in a Fragment node; those wrappers are
 * flattened here so each row is an individually recycled lazy item rather
 * than one giant Fragment item. Keys come from the row nodes (Keyed in Go);
 * an unkeyed row falls back to positional identity, which behaves like
 * Column but loses row state on reorder — same contract as key() above.
 */
@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun GrMobList(node: GrMobNode, extra: Modifier) {
    val s = animatedStyle(node.style)
    // Reading node.children in composition subscribes this scope to the
    // SnapshotStateList, so structural patches recompose the list.
    val rows = flattenFragments(node.children)
    LazyColumn(
        modifier = s.boxModifier(extra, gestureModifier(node)),
        verticalArrangement = verticalArrangement(s),
        // The lazy sibling of GrMobColumn's dispatch, with the same contract
        // and the same two vocabularies. Held to core.AlignItemsValues() by
        // TestKotlinListAlignmentCoversEveryAlignItems in mobile/verify.
        horizontalAlignment = when (s?.alignItems?.ifEmpty { s.align }) {
            "flex-start", "start" -> Alignment.Start
            "center" -> Alignment.CenterHorizontally
            "flex-end", "end" -> Alignment.End
            // A stretched row is filled by the modifier the item loop applies,
            // not placed by the stack. Same shape as Row and Column above.
            "stretch" -> Alignment.Start
            else -> Alignment.Start
        },
    ) {
        itemsIndexed(
            rows,
            key = { i, row -> row.key.ifEmpty { i } },
            contentType = { _, row -> row.type },
        ) { _, row ->
            // A Transition declared on the List itself animates row
            // *placement*: keyed rows slide to their new positions on
            // reorder/insert/removal instead of teleporting. (A Transition
            // on a row animates that row's own property changes — two
            // declarations, two scopes.) Built here because
            // animateItemPlacement only exists inside LazyItemScope.
            var m: Modifier = if ((s?.transitionMs ?: 0) > 0) {
                Modifier.animateItemPlacement(s!!.transitionTween())
            } else {
                Modifier
            }
            // Same cross-axis stretch as ColumnChildren; a lazy item has no
            // weight to combine it with, since a lazy list's main axis is
            // scrollable and therefore unbounded.
            if (isStretch(s)) m = m.fillMaxWidth()
            RenderNode(row, m)
        }
    }
}

/** Inlines Fragment/Theme grouping nodes so their children become list rows. */
private fun flattenFragments(children: List<GrMobNode>): List<GrMobNode> {
    if (children.none { it.type == "Fragment" || it.type == "Theme" }) return children
    val out = ArrayList<GrMobNode>(children.size)
    for (child in children) {
        if (child.type == "Fragment" || child.type == "Theme") {
            out.addAll(flattenFragments(child.children))
        } else {
            out.add(child)
        }
    }
    return out
}

/**
 * core.JustifyContents -> Compose Arrangement, on each axis.
 *
 * Both are held to core.JustifyContents() by
 * TestKotlinArrangementsCoverEveryJustifyContent in mobile/verify, which
 * parses these arms: one per line, string literals first, `else ->` last. The
 * "flex-start" arm duplicates `else`'s body and stays spelled out — an
 * unlisted value here does not fall back to some neutral arrangement, it packs
 * to the start, which is a rendering and not an abstention.
 *
 * Known divergence, deliberately left alone: the five distributing
 * arrangements drop Style.Gap. CSS treats gap as a minimum that
 * justify-content then adds to, and the iOS solver does the same (it carries
 * `spacing` separately from `justify`), but Compose's Arrangement.Center and
 * friends take no spacing argument, so a Row with both a Gap and a
 * JustifyContent loses the gap here alone. Arrangement.spacedBy(gap, alignment)
 * would fix the three packing values; nothing expresses gap-plus-distribution
 * for the space-* three without a custom Arrangement. Not attempted because it
 * is a rendering change on the one target this repo cannot build.
 */
private fun horizontalArrangement(s: GrMobStyle?): Arrangement.Horizontal =
    when (s?.justifyContent) {
        "flex-start" -> packedHorizontally(s)
        "center" -> Arrangement.Center
        "flex-end" -> Arrangement.End
        "space-between" -> Arrangement.SpaceBetween
        "space-around" -> Arrangement.SpaceAround
        "space-evenly" -> Arrangement.SpaceEvenly
        else -> packedHorizontally(s)
    }

private fun verticalArrangement(s: GrMobStyle?): Arrangement.Vertical =
    when (s?.justifyContent) {
        "flex-start" -> packedVertically(s)
        "center" -> Arrangement.Center
        "flex-end" -> Arrangement.Bottom
        "space-between" -> Arrangement.SpaceBetween
        "space-around" -> Arrangement.SpaceAround
        "space-evenly" -> Arrangement.SpaceEvenly
        else -> packedVertically(s)
    }

/**
 * Children packed against the start edge, which is what flex-start asks for
 * and also what an absent or unrecognized JustifyContent gets.
 *
 * Extracted so the explicit "flex-start" arm and the `else` arm above can
 * share one body rather than repeat it: the body is not a bare Arrangement,
 * because this is the one path where Style.Gap survives (see the divergence
 * note above), and two copies of that conditional would be two chances to
 * change only one of them.
 */
private fun packedHorizontally(s: GrMobStyle?): Arrangement.Horizontal =
    if ((s?.gap ?: 0f) > 0f) Arrangement.spacedBy(s!!.gap.dp) else Arrangement.Start

private fun packedVertically(s: GrMobStyle?): Arrangement.Vertical =
    if ((s?.gap ?: 0f) > 0f) Arrangement.spacedBy(s!!.gap.dp) else Arrangement.Top

// ---------------------------------------------------------------------------
// Composite components
// ---------------------------------------------------------------------------

@Composable
private fun GrMobTabView(node: GrMobNode, extra: Modifier) {
    val runtime = LocalGrMobRuntime.current
    val selected = node.intProp("selectedIndex")
    val onTabChange = node.stringProp("onTabChange")

    @Suppress("UNCHECKED_CAST")
    val tabs = node.props["tabs"] as? List<Map<String, Any?>> ?: emptyList()

    Column(node.style.boxModifier(extra)) {
        TabRow(selectedTabIndex = selected.coerceIn(0, (tabs.size - 1).coerceAtLeast(0))) {
            tabs.forEachIndexed { i, tab ->
                Tab(
                    selected = i == selected,
                    onClick = { if (onTabChange.isNotEmpty()) runtime.intChanged(onTabChange, i) },
                    text = { Text(tab["label"] as? String ?: "") },
                )
            }
        }
        // Go renders every tab's content as a child; selection is presentation
        // state, so only the selected child is composed. key() on the index
        // gives each tab its own composition identity, so per-tab state (input
        // text, scroll) is dropped on switch — matching the replace semantics
        // the Go reconciler would apply anyway.
        node.children.getOrNull(selected)?.let { current ->
            key(selected) { RenderNode(current) }
        }
    }
}

@Composable
private fun GrMobModal(node: GrMobNode) {
    if (!node.boolProp("visible")) return
    val runtime = LocalGrMobRuntime.current
    val onDismiss = node.stringProp("onDismiss")
    Dialog(
        onDismissRequest = { if (onDismiss.isNotEmpty()) runtime.click(onDismiss) },
    ) {
        // The dialog window already scrims with the backdrop; the content gets
        // a card-like surface unless the app styled its children explicitly.
        Column(
            Modifier
                .fillMaxWidth()
                .padding(8.dp)
                .background(Color.White, RoundedCornerShape(12.dp))
                .padding(16.dp)
        ) { ColumnChildren(node) }
    }
}
