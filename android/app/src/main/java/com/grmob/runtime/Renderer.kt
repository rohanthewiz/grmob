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
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
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
import androidx.compose.ui.graphics.Color
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
    val style = node.style
    if (style?.display == "none") return // not composed at all; "hidden" keeps space

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
        "SafeArea" -> Box(style.boxModifier(extra).safeDrawingPadding()) { RenderChildren(node) }

        "TabView" -> GrMobTabView(node, extra)
        "Modal" -> GrMobModal(node)
        "Image" -> AsyncImage(
            model = node.stringProp("src"),
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
    val runtime = LocalGrMobRuntime.current
    return Modifier.combinedClickable(
        onClick = { if (onClick.isNotEmpty()) runtime.click(onClick) },
        onLongClick = if (onLongPress.isEmpty()) null else {
            { runtime.click(onLongPress) }
        },
    )
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
        textAlign = when (s.align) {
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
        shape = RoundedCornerShape((s?.borderRadius ?: 8f).dp),
        colors = ButtonDefaults.buttonColors(
            containerColor = s?.background ?: Color.Unspecified,
            contentColor = s?.textColor ?: Color.Unspecified,
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

    val rows = node.intProp("rows")
    var modifier = s.boxModifier(extra)
    if (multiline && rows > 0) {
        // Approximate a rows-based min height from the line height.
        val line = if (s != null && s.fontSize > 0f) s.fontSize * 1.4f else 24f
        modifier = modifier.heightIn(min = (line * rows).dp)
    }

    BasicTextField(
        value = text,
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
            // reads as "act on this", mirroring the iOS submitLabel.
            imeAction = if (onSubmit.isNotEmpty()) ImeAction.Done else ImeAction.Default,
        ),
        // The IME action dispatches onSubmit as a plain void event — the same
        // channel as a Button tap.
        keyboardActions = KeyboardActions(
            onDone = { if (onSubmit.isNotEmpty()) runtime.click(onSubmit) },
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
        modifier = s.boxModifier(extra, gestureModifier(node)),
        horizontalArrangement = horizontalArrangement(s),
        verticalAlignment = when (s?.alignItems) {
            "center" -> Alignment.CenterVertically
            "flex-end" -> Alignment.Bottom
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
        // ("center"/"end") acts as a fallback when AlignItems is unset.
        horizontalAlignment = when (s?.alignItems?.ifEmpty { s.align }) {
            "center" -> Alignment.CenterHorizontally
            "flex-end", "end" -> Alignment.End
            else -> Alignment.Start
        },
    ) { ColumnChildren(node) }
}

/**
 * The children loops live inside RowScope/ColumnScope because FlexGrow maps
 * onto Modifier.weight, which exists only as a scope extension — the parent
 * computes it and hands it down as the child's `extra` modifier.
 */
@Composable
private fun RowScope.RowChildren(node: GrMobNode) {
    node.children.forEachIndexed { i, child ->
        key(child.key.ifEmpty { i }) {
            val grow = child.style?.flexGrow ?: 0f
            RenderNode(child, if (grow > 0f) Modifier.weight(grow) else Modifier)
        }
    }
}

@Composable
private fun ColumnScope.ColumnChildren(node: GrMobNode) {
    node.children.forEachIndexed { i, child ->
        key(child.key.ifEmpty { i }) {
            val grow = child.style?.flexGrow ?: 0f
            RenderNode(child, if (grow > 0f) Modifier.weight(grow) else Modifier)
        }
    }
}

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
        horizontalAlignment = when (s?.alignItems?.ifEmpty { s.align }) {
            "center" -> Alignment.CenterHorizontally
            "flex-end", "end" -> Alignment.End
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
            val placement = if ((s?.transitionMs ?: 0) > 0) {
                Modifier.animateItemPlacement(s!!.transitionTween())
            } else {
                Modifier
            }
            RenderNode(row, placement)
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

private fun horizontalArrangement(s: GrMobStyle?): Arrangement.Horizontal =
    when (s?.justifyContent) {
        "center" -> Arrangement.Center
        "flex-end" -> Arrangement.End
        "space-between" -> Arrangement.SpaceBetween
        "space-around" -> Arrangement.SpaceAround
        "space-evenly" -> Arrangement.SpaceEvenly
        else -> if ((s?.gap ?: 0f) > 0f) Arrangement.spacedBy(s!!.gap.dp) else Arrangement.Start
    }

private fun verticalArrangement(s: GrMobStyle?): Arrangement.Vertical =
    when (s?.justifyContent) {
        "center" -> Arrangement.Center
        "flex-end" -> Arrangement.Bottom
        "space-between" -> Arrangement.SpaceBetween
        "space-around" -> Arrangement.SpaceAround
        "space-evenly" -> Arrangement.SpaceEvenly
        else -> if ((s?.gap ?: 0f) > 0f) Arrangement.spacedBy(s!!.gap.dp) else Arrangement.Top
    }

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
