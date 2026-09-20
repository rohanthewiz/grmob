package com.grmob.runtime

import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsFocusedAsState
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.LayoutDirection
import androidx.compose.ui.text.style.TextDirection
import androidx.compose.ui.platform.LocalLayoutDirection
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusProperties
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.key
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.key.type
import androidx.compose.ui.input.pointer.PointerEventPass
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.semantics.clearAndSetSemantics
import androidx.compose.ui.semantics.text
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.TextRange
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.withStyle
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardCapitalization
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.OffsetMapping
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.text.input.TransformedText
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * core.CodeEditor: an editable monospace buffer with syntax colour, a
 * line-number gutter and the keyboard behaviour a programmer's editor has.
 *
 * # Why a VisualTransformation and not a styled state
 *
 * The temptation is to hold an `AnnotatedString` as the field's value and write
 * the colours into it. Compose will not have it: `BasicTextField`'s state is a
 * `TextFieldValue` whose text is a plain `String`, and the styled view of it is
 * produced by a `VisualTransformation` at layout time. That split is exactly
 * what this node wants and is the sanctioned way to colour an editable buffer:
 * the transformation is a pure function from the buffer to a picture of it, so
 * it can be re-derived on every frame without touching the state the IME is
 * composing into.
 *
 * The mapping is `OffsetMapping.Identity` because the transformation is a
 * *colouring*: it adds spans and never a character, so visual offset N is
 * buffer offset N. Anything else here would misplace the caret.
 *
 * The one exception is a buffer holding a literal tab. Compose's text layout
 * has no tab stops: a "\t" draws about one space wide, so a Go file indented
 * with tabs lost its indentation. Such a buffer gets each tab drawn as the
 * spaces to the next stop, with a real mapping so caret and selection stay in
 * buffer units; see [GrMobTabStops]. A tab-free buffer (the default, since the
 * indent this editor inserts is spaces unless tabSize is 0) keeps Identity.
 *
 * # The three rules, as they land here
 *
 *  1. **Echo guard.** GrMobTextField's TextEditLedger, one type wider: the
 *     local state is a `TextFieldValue` rather than a `String`, so an echo that
 *     is dropped leaves the *selection* alone as well as the text, and a
 *     rewrite that replays in-flight typing puts the caret where that typing
 *     was ([rebaseCaret]).
 *
 *  2. **Decoration is advisory and per line.** The transformation compares each
 *     row's concatenated text with the line under it and paints only where they
 *     agree. Go is a keystroke behind for a few milliseconds after every
 *     keypress, so the line being typed loses its colours for a frame and no
 *     other line does.
 *
 *  3. **Commands are epoch-stamped props.** See the LaunchedEffect below, and
 *     core/editor.go for the mechanism.
 *
 * # The IME's composing region
 *
 * A rows-only patch touches the transformation and never `TextFieldValue`,
 * which is what keeps a composition alive: Compose reports the composing region
 * on the value, and writing a new value mid-composition cancels it. That is the
 * property this design is chosen for, and it is why the colours go through a
 * transformation rather than through the state.
 *
 * # The compilation gap, stated
 *
 * Nothing in this repository compiles Kotlin that imports Compose, so this file
 * is held to the contract textually by `mobile/verify/codeeditor_test.go` and
 * by a device pass. Same caveat GrMobMapView.kt carries.
 */
@Composable
internal fun GrMobCodeEditor(node: GrMobNode, extra: Modifier) {
    val runtime = LocalGrMobRuntime.current
    val s = node.style

    val upstream = node.stringProp("value")
    val onChange = node.stringProp("onChange")
    val onSelectionChange = node.stringProp("onSelectionChange")
    val readOnly = node.boolProp("readOnly")
    val lineNumbers = node.boolProp("lineNumbers")
    val tabSize = node.intProp("tabSize")
    val commentPrefix = node.stringProp("commentPrefix")
    val editorEpoch = node.intProp("editorEpoch")
    val editorCommand = node.stringProp("editorCommand")
    // The imperative half: core.Focus / core.DismissKeyboard reach the screen as
    // these two props, exactly as they do for an ordinary field. See the
    // LaunchedEffect below and core/focus.go.
    val focusEpoch = node.intProp("focusEpoch")
    val focusAction = node.stringProp("focusAction")

    val interactions = remember { MutableInteractionSource() }
    val focused by interactions.collectIsFocusedAsState()
    val focusRequester = remember { FocusRequester() }
    val focusManager = LocalFocusManager.current
    // Whether a read-only buffer may take focus right now; see ReadOnlyFocusGate.
    val gate = remember { ReadOnlyFocusGate() }
    // core.Inert, read here rather than left to RenderNode's head-of-chain
    // modifier, which cannot reach this editor's field.
    //
    // Compose applies a focus target's properties by walking up from the
    // target through the modifier chain — but the walk stops at the first
    // FocusTarget it meets on the way (DelegatableNode.visitSelfAndAncestors
    // takes `untilType = Nodes.FocusTarget`). The field below sits behind two
    // scroll boxes, and a Compose scroll container delegates a focus target of
    // its own, so the walk from the field ends at the horizontal scroll and
    // never reaches the `focusProperties { canFocus = false }` that RenderNode
    // prepended to `extra` on the Row.
    //
    //	Row(extra: canFocus=false)      ← Inert lands here
    //	 └ verticalScroll   ─ FocusTarget
    //	    └ Box + horizontalScroll ─ FocusTarget  ← the walk stops here
    //	       └ BasicTextField ─ FocusTarget        ← never sees it
    //
    // Unnoticed until now because it only bites an *editable* editor: a
    // read-only one answers `false` to a Tab search through its own gate
    // whatever the ancestor says. An editable one inside a shut Drawer panel
    // stayed a hardware-keyboard Tab stop.
    val inert = LocalGrMobInert.current

    var buffer by remember { mutableStateOf(TextFieldValue(upstream)) }
    // Go's edit stamps and this editor's half of them. See core/text_edit.go
    // and GrMobTextEdits.kt; the rule is GrMobTextField's, unchanged.
    val editSeq = node.intProp("editSeq")
    val editEpoch = node.intProp("editEpoch")
    // Whether Go stamped this editor at all; see TextEditLedger.
    val stamped = node.props.containsKey("editEpoch")
    val ledger = remember { TextEditLedger(upstream, editEpoch) }

    // One place every local edit leaves by, so the ledger and the dispatch can
    // never get out of step with each other.
    val commit: (TextFieldValue) -> Unit = { next ->
        buffer = next
        if (onChange.isNotEmpty()) {
            ledger.sent(runtime.textEdited(onChange, next.text, ledger.epoch), next.text)
        }
    }

    // The echo guard. It used to be a queue of the values sent upstream, with
    // any upstream value matching nothing in it read as a rewrite; that is
    // GrMobTextField's old guard, and it had the same hole: a rewrite landing
    // behind keystrokes already sent let Go apply them as new. Now Go stamps
    // the editor with the last edit it applied and its rewrite count, a higher
    // count is a rewrite that wins even mid-typing, and anything else is our
    // own typing coming back.
    //
    // The three together, because each can change alone: an echo moves only
    // the ack, and a refused edit moves the ack and the epoch and leaves the
    // value where it was.
    val seen = Triple(upstream, editSeq, editEpoch)
    var lastSeen by remember { mutableStateOf(seen) }
    if (seen != lastSeen) {
        lastSeen = seen
        if (focused) {
            val local = buffer
            ledger.upstream(upstream, editSeq, editEpoch, local.text, stamped)?.let { next ->
                // The caret follows the typing that was replayed; with nothing
                // replayed it lands at the end of Go's text, as it always did.
                val caret = if (!stamped) next.length
                else rebaseCaret(ledger.lastBasis, local.text, upstream, local.selection.end)
                buffer = TextFieldValue(next, TextRange(caret))
                // Typing Go has not seen yet, replayed onto its rewrite.
                if (next != upstream && onChange.isNotEmpty()) {
                    ledger.sent(runtime.textEdited(onChange, next, ledger.epoch), next)
                }
            }
        }
    }
    if (!focused) {
        // Go-owned while blurred; anything in flight died with the focus session.
        ledger.reset(upstream, editEpoch)
        if (buffer.text != upstream) buffer = TextFieldValue(upstream)
    }

    // The selection, as byte offsets into the UTF-8 value. Compose counts
    // UTF-16 chars and core.OnSelectionChange promises bytes, so the conversion
    // is the whole of what this does beyond the dispatch — the same conversion
    // the other three hosts make from their own unit.
    //
    // Deduped against the last payload because a recomposition can report an
    // unchanged selection, and each dispatch is a Go render pass.
    var lastSelection by remember { mutableStateOf("") }
    LaunchedEffect(buffer.selection, buffer.text, onSelectionChange) {
        if (onSelectionChange.isEmpty()) return@LaunchedEffect
        val payload = "${utf8Offset(buffer.text, buffer.selection.min)}:" +
            "${utf8Offset(buffer.text, buffer.selection.max)}"
        if (payload == lastSelection) return@LaunchedEffect
        lastSelection = payload
        runtime.textChanged(onSelectionChange, payload)
    }

    // Go's commands.
    //
    // # Why an editor adopts a standing epoch instead of running it
    //
    // A focus command deliberately re-fires on a field that mounts while it is
    // the target — that is what makes "push a screen and put the cursor in its
    // search box" work. An editor command is the opposite: it names a *moment*
    // and an edit, and an editor that was not composed when it was issued
    // missed it. Running it at first composition would indent the buffer every
    // time the screen came back. So the first pass records the epoch and runs
    // nothing; only a change after that is an instruction. All four hosts agree.
    var lastEpoch by remember { mutableStateOf<Int?>(null) }
    LaunchedEffect(editorEpoch) {
        val previous = lastEpoch
        lastEpoch = editorEpoch
        if (previous == null || editorEpoch == 0 || editorEpoch == previous) {
            return@LaunchedEffect
        }
        if (editorCommand == "selectAll") {
            // Allowed on a read-only buffer: selecting is reading, which is
            // exactly what read-only permits.
            buffer = buffer.copy(selection = TextRange(0, buffer.text.length))
            return@LaunchedEffect
        }
        if (readOnly) return@LaunchedEffect
        val next = applyCodeCommand(buffer, editorCommand, indentUnit(tabSize), commentPrefix)
        // Null is "this editor does not know that command", which is a no-op
        // rather than an error: a toolbar that outgrew its editor must not
        // crash the screen.
        if (next != null) commit(next)
    }

    val base = textStyle(s).copy(
        fontFamily = FontFamily.Monospace,
        // A code editor is read in columns, so a proportional default size
        // (17sp, what a form field takes) puts about half as much line on
        // screen as it should. 13sp is the pitch both DOM targets and the iOS
        // renderer settle on.
        fontSize = if (s != null && s.fontSize > 0f) s.fontSize.sp else 13.sp,
        // Source reads left to right whatever the locale; see the layout
        // direction pinned around the Row below.
        textDirection = TextDirection.Ltr,
    )
    // How wide a literal tab draws: tabSize columns, or 4 when tabSize is 0
    // (a literal-tab indent), which is the width the web runtime gives the
    // same buffer through CSS tab-size.
    val transformation = GrMobCodeRows(node.children, base.color,
        if (tabSize > 0) tabSize else 4)

    val vertical = rememberScrollState()
    val horizontal = rememberScrollState()

    // Go's focus commands. Keyed on the epoch alone, never on the action: the
    // action is what to do, the epoch is when — and a second core.Focus on an
    // already-focused editor has to re-fire, which only a changed value can
    // express.
    //
    // The rule here is GrMobTextField's, not runCommand's one line up the file,
    // and the difference is deliberate. An editor command names a moment and an
    // edit, so an editor that was off screen when it was issued missed it and
    // adopts the epoch silently. A focus command is the opposite: an editor that
    // mounts while it is already the target should take the caret, because
    // "open a screen with the cursor in its editor" issues the command one pass
    // before the editor exists. LaunchedEffect runs on first composition, which
    // is exactly that behaviour.
    //
    // "blur" is guarded on this editor actually holding focus, so one dismiss
    // does not have every editor on screen calling clearFocus(). Only the target
    // acts on "focus"; everything else is told "" and does nothing, because
    // requesting focus over there already takes it from here.
    LaunchedEffect(focusEpoch) {
        if (focusEpoch == 0) return@LaunchedEffect
        when (focusAction) {
            // requestFocus throws if the requester is not attached to a placed
            // node yet. A LaunchedEffect already runs after composition, which
            // covers the ordinary case; the catch covers the editor being
            // composed but not yet placed — inside a lazy list row that has not
            // laid out — where the honest outcome is "the command missed"
            // rather than a crashed screen. GrMobTextField says the same.
            // The gate opens first: a command is the one keyboard-free way a
            // read-only buffer is still allowed to focus, as a programmatic
            // focus() still reaches the web's tabindex="-1" textarea.
            "focus" -> runCatching {
                gate.open = true
                focusRequester.requestFocus()
            }
            "blur" -> if (focused) focusManager.clearFocus()
        }
    }

    // The gutter is a sibling of the field, inside the same vertical scroll, so
    // number N stays beside line N with nothing measured. The horizontal scroll
    // wraps the field alone: the buffer pans sideways for a long line and the
    // numbers stay put.
    //
    // verticalScrollWhenBounded, not a bare verticalScroll. An editor with no
    // Height sizes to its content (comps.CodeEditor.Height says so), and the
    // ordinary place for one is a scrolled page: every tutorial code block is a
    // read-only editor inside comps.Screen{Scroll: true}. There the page's own
    // scroll hands this Row an infinite maximum height, and Compose's scroll
    // refuses to measure under one — IllegalStateException, "Vertically
    // scrollable component was measured with an infinity maximum height
    // constraints", on the first layout of every lesson. The helper caps the
    // viewport at the content in that case, which is the picture the web and
    // iOS already draw; see it in Renderer.kt.
    //
    // Code is left to right in every locale. Under an RTL app locale the Row
    // put the gutter on the right, and the field right-aligned every line
    // and ran the bidi algorithm over it, so a line that opens with a
    // neutral such as "}" or "(" moved its punctuation to the far end —
    // "}comps.Drawer". An editor is not prose: its columns are what it
    // means. Both DOM targets pin the same thing with dir="ltr" on the
    // editor box, and the iOS view forces left to right; the box's own
    // padding reads left to right with it, as dir does on the web.
    CompositionLocalProvider(LocalLayoutDirection provides LayoutDirection.Ltr) {
        Row(s.boxModifier(extra).verticalScrollWhenBounded(vertical)) {
            if (lineNumbers) {
                val lines = buffer.text.count { it == '\n' } + 1
                // Wide enough for the largest number plus a column of room, in the
                // same units the two DOM targets state as `Nch`: one monospace
                // advance is about 0.6em, so the width is the digit count plus two,
                // times that. Approximate on purpose — measuring the face would
                // need a TextMeasurer and a font load for a column of digits whose
                // only requirement is that it not clip.
                val gutterWidth = (base.fontSize.value * 0.6f * (lines.toString().length + 2)).dp
                val numberStyle = base.copy(
                    color = (base.color.takeIf { it != Color.Unspecified } ?: Color.Gray)
                        .copy(alpha = 0.45f),
                    textAlign = TextAlign.End,
                )
                // Hidden from TalkBack, as the web's gutter is aria-hidden: the
                // numbers are chrome, and each was its own stop reading "1",
                // "2", "3" ahead of the code.
                Column(
                    Modifier.width(gutterWidth).padding(end = 4.dp).clearAndSetSemantics { }
                ) {
                    for (i in 1..lines) {
                        Text(
                            text = "$i",
                            style = numberStyle,
                            maxLines = 1,
                            modifier = Modifier.width(gutterWidth),
                        )
                    }
                }
            }
            // horizontalScrollWhenBounded for the same reason as the Row's vertical
            // helper, on the other axis. The Row hands the field whatever width is
            // left beside the gutter, which is infinite when the editor sits in a
            // sideways Scroll. A bare horizontalScroll throws there.
            //
            // A read-only buffer reads as text, not as a field. Compose 1.7
            // reports every BasicTextField as editable to accessibility:
            // AndroidComposeViewAccessibilityDelegateCompat sets
            // `info.isEditable` from whether IsEditable is *present* in the
            // semantics, not from its value, and readOnly writes it as false.
            // So TalkBack said "Editing" on a code block nobody can edit, and
            // gave it the EditText class. Clearing the field's semantics here
            // and stating the text is what iOS already does (a UITextView that
            // is not editable reads as static text). The scroll's semantics
            // sit on this same node, ahead of the clear, and are kept.
            Box(
                Modifier.horizontalScrollWhenBounded(horizontal).then(
                    if (readOnly) Modifier.clearAndSetSemantics { text = AnnotatedString(buffer.text) }
                    else Modifier
                )
            ) {
                BasicTextField(
                    value = buffer,
                    onValueChange = { next ->
                        // Return, from the soft keyboard as well as a hardware one.
                        // Handled here rather than in onPreviewKeyEvent because an
                        // IME inserts the newline through the text input session and
                        // never as a key event, so a key handler would work on a
                        // tablet with a keyboard and nowhere else.
                        commit(autoIndent(buffer, next, indentUnit(tabSize)))
                    },
                    readOnly = readOnly,
                    // enabled stays true even when readOnly: a read-only buffer
                    // still focuses, still shows a caret and still selects, which is
                    // the whole difference between read-only and disabled.
                    enabled = !node.isDisabled(),
                    textStyle = base,
                    interactionSource = interactions,
                    visualTransformation = transformation,
                    // No wrapping: a code line is one line, and a wrapped one
                    // restarts at column zero, which reads as a new statement at the
                    // outermost indent. The horizontal scroll above is what it pans
                    // in instead.
                    singleLine = false,
                    keyboardOptions = KeyboardOptions(
                        keyboardType = KeyboardType.Ascii,
                        // Both corrupt source: autocorrect rewrites identifiers and
                        // capitalization capitalises the first keyword of every line.
                        autoCorrectEnabled = false,
                        capitalization = KeyboardCapitalization.None,
                    ),
                    // The requester sits on the field and not on the Row above it:
                    // the Row is the scroll box and holds the gutter, which is
                    // chrome the caret must never reach. Compose would happily give
                    // focus to the container, and the keyboard would not come up.
                    modifier = Modifier
                        // Out of the Tab order when read-only; see ReadOnlyFocusGate.
                        // A pointer press opens the gate on the Initial pass, which
                        // runs before the field's own tap handler asks for focus.
                        // Inert wins over both: it is the ancestor's refusal, and
                        // this is the only place it can be applied to the field
                        // (see `inert` above).
                        .focusProperties { canFocus = !inert && (!readOnly || gate.open) }
                        .onFocusChanged { state ->
                            gate.focused = state.isFocused
                            if (!state.isFocused) gate.open = false
                        }
                        .pointerInput(readOnly) {
                            if (!readOnly) return@pointerInput
                            awaitPointerEventScope {
                                while (true) {
                                    val initial = awaitPointerEvent(PointerEventPass.Initial)
                                    if (initial.changes.any { it.pressed }) gate.open = true
                                    // A press that ended without focus (a drag
                                    // that scrolled the page) must not leave the
                                    // buffer as a Tab stop. The Final pass runs
                                    // after the field's tap handler has had its
                                    // chance to focus it.
                                    val last = awaitPointerEvent(PointerEventPass.Final)
                                    if (last.changes.none { it.pressed } && !gate.focused) {
                                        gate.open = false
                                    }
                                }
                            }
                        }
                        .focusRequester(focusRequester).onPreviewKeyEvent { event ->
                        // Tab, which would otherwise move focus out of the editor
                        // and make indenting impossible. Hardware keyboards only —
                        // a soft keyboard has no Tab — which is why this is the one
                        // key handled here and Return is handled on the value.
                        if (event.type != KeyEventType.KeyDown || event.key != Key.Tab) {
                            return@onPreviewKeyEvent false
                        }
                        // Read-only: nothing to indent, so Tab is the platform's
                        // again and moves focus on. This consumed it (returned
                        // true), which made every read-only editor a keyboard
                        // trap (WCAG 2.1.2): the tutorial's code blocks are
                        // read-only CodeEditors, and on the emulator Tab
                        // reached the first one on lesson 4.9 and then went
                        // nowhere for 36 presses — TalkBack said "Editing" once
                        // and nothing after. The web returns early for a
                        // read-only buffer and iOS only intercepts an editable
                        // one; this was the one host that held on to it.
                        if (readOnly) return@onPreviewKeyEvent false
                        commit(insertInCode(buffer, indentUnit(tabSize)))
                        true
                    },
                )
            }
        }
    }
}

/**
 * Whether a read-only CodeEditor may take input focus right now.
 *
 * The web takes a read-only buffer out of the Tab order with tabindex="-1":
 * a page of code blocks would otherwise put a stop in front of each one with
 * nothing to do there. A pointer and a programmatic focus() still reach it,
 * so it can still be selected. Compose has no tabindex; `canFocus` is all or
 * nothing. So the answer is a gate that only a pointer press or Go's focus
 * command opens, and that closes when focus leaves:
 *
 *     Tab / Shift+Tab ──▶ focus search ──▶ canFocus? gate shut ──▶ skipped
 *     press ──▶ gate open ──▶ field's tap handler focuses ──▶ select, copy
 *     core.Focus ──▶ gate open ──▶ requestFocus()
 *     focus leaves ──▶ gate shut
 *
 * Plain fields rather than Compose state, on purpose. Compose observes the
 * reads inside `focusProperties` while a node holds focus and clears focus
 * the moment `canFocus` turns false. Keying the gate on observed state (the
 * input mode, say) would drop the caret out of a tapped code block the
 * instant Tab switched the window to keyboard mode, and the Tab would then
 * start again from the top of the screen. The gate is read at search time
 * and never watched, which is the tabindex behaviour exactly.
 */
internal class ReadOnlyFocusGate {
    var open = false
    var focused = false
}

/**
 * Go's rows as a colouring of the buffer.
 *
 * A row is applied only when its concatenated text is still exactly the line
 * under it — rule 2 of the shared editor design. A row that disagrees leaves
 * its line in the base ink, which is what makes a lexer that is a keystroke
 * behind cost one line's colours for one frame instead of the whole buffer's.
 *
 * The attribute bits are core's Grid* constants, read the same way
 * GrMobGridRow reads them: 1 bold, 2 dim, 4 italic, 8 underline, 16 strike.
 * Dim has no Compose spelling, so it fades the run's colour — which needs a
 * colour to fade, hence the fallback to the editor's own ink.
 */
internal class GrMobCodeRows(
    private val rows: List<GrMobNode>,
    private val ink: Color,
    private val tabStops: Int = 4,
) : VisualTransformation {

    override fun filter(text: AnnotatedString): TransformedText {
        val lines = text.text.split("\n")
        val styled = buildAnnotatedString {
            lines.forEachIndexed { index, line ->
                if (index > 0) append("\n")
                val runs = rows.getOrNull(index)?.props?.get("runs") as? List<*>
                if (runs == null || runsText(runs) != line) {
                    append(line)
                    return@forEachIndexed
                }
                for (raw in runs) {
                    val run = raw as? Map<*, *> ?: continue
                    val bits = (run["a"] as? Number)?.toInt() ?: 0
                    var fg = GrMobStyle.parseColor(run["fg"] as? String)
                    if (bits and 2 != 0) {
                        val dimmed = fg ?: ink.takeIf { it != Color.Unspecified }
                        if (dimmed != null) fg = dimmed.copy(alpha = dimmed.alpha * 0.6f)
                    }
                    val decorations = mutableListOf<TextDecoration>()
                    if (bits and 8 != 0) decorations.add(TextDecoration.Underline)
                    if (bits and 16 != 0) decorations.add(TextDecoration.LineThrough)
                    withStyle(
                        SpanStyle(
                            color = fg ?: Color.Unspecified,
                            background = GrMobStyle.parseColor(run["bg"] as? String)
                                ?: Color.Unspecified,
                            fontWeight = if (bits and 1 != 0) FontWeight.Bold else null,
                            fontStyle = if (bits and 4 != 0) FontStyle.Italic else null,
                            textDecoration =
                                if (decorations.isEmpty()) null
                                else TextDecoration.combine(decorations),
                        )
                    ) { append(run["t"] as? String ?: "") }
                }
            }
        }
        // Identity whenever it can be: this is a colouring, so it adds spans
        // and never a character. Visual offset N is buffer offset N, which is
        // what keeps the caret where the user put it. Only a literal tab needs
        // characters added, and only a buffer that has one pays for a mapping.
        if ('\t' !in text.text) return TransformedText(styled, OffsetMapping.Identity)
        return GrMobTabStops.expand(styled, tabStops)
    }

    private fun runsText(runs: List<*>): String {
        val out = StringBuilder()
        for (raw in runs) {
            val run = raw as? Map<*, *> ?: continue
            out.append(run["t"] as? String ?: "")
        }
        return out.toString()
    }
}

/**
 * Literal tabs drawn to their tab stops, for a text layout that has none.
 *
 * Each "\t" becomes the spaces from its column to the next multiple of
 * [expand]'s `stops`, so a tab after "ab" at 4 stops is two spaces and a tab
 * at column 0 is four. Columns restart at every "\n", and count UTF-16 units,
 * which is exact for the ASCII a code buffer is overwhelmingly made of and a
 * column off per astral character otherwise.
 *
 * The mapping, for "a\tb" at 4 stops (buffer offsets above, drawn below):
 *
 *	  buffer   a  \t          b
 *	           0  1           2  3
 *	  drawn    a  ·  ·  ·     b
 *	           0  1  2  3     4  5
 *
 *	  originalToTransformed: 0→0  1→1  2→4  3→5
 *	  transformedToOriginal: 0→0  1→1  2→1  3→1  4→2  5→3
 *
 * A drawn offset inside a tab's spaces maps back to the tab's own offset, so a
 * tap in the middle of an indent puts the caret before the tab rather than
 * inside a character that does not exist in the buffer. Both tables have an
 * entry for the end offset, which Compose asks for with the caret at the end.
 *
 * The spaces keep whatever span styles covered the tab (a background, say), so
 * the colouring reads the same as it would have on the tab itself.
 */
internal object GrMobTabStops {
    fun expand(styled: AnnotatedString, stops: Int): TransformedText {
        val text = styled.text
        val width = if (stops > 0) stops else 4
        val toDrawn = IntArray(text.length + 1)
        val toBuffer = ArrayList<Int>(text.length + 16)
        val out = AnnotatedString.Builder()
        var column = 0
        var runStart = 0
        for (i in text.indices) {
            toDrawn[i] = toBuffer.size
            val c = text[i]
            if (c != '\t') {
                toBuffer.add(i)
                column = if (c == '\n') 0 else column + 1
                continue
            }
            // Flush the untouched run before the tab with its spans intact.
            if (runStart < i) out.append(styled.subSequence(runStart, i))
            runStart = i + 1
            val spaces = width - column % width
            val covering = styled.spanStyles.filter { it.start <= i && i < it.end }
            covering.forEach { out.pushStyle(it.item) }
            out.append(" ".repeat(spaces))
            repeat(covering.size) { out.pop() }
            repeat(spaces) { toBuffer.add(i) }
            column += spaces
        }
        if (runStart < text.length) out.append(styled.subSequence(runStart, text.length))
        toDrawn[text.length] = toBuffer.size
        toBuffer.add(text.length)

        val mapping = object : OffsetMapping {
            override fun originalToTransformed(offset: Int): Int =
                toDrawn[offset.coerceIn(0, text.length)]
            override fun transformedToOriginal(offset: Int): Int =
                toBuffer[offset.coerceIn(0, toBuffer.size - 1)]
        }
        return TransformedText(out.toAnnotatedString(), mapping)
    }
}

/** One indent: [tabSize] spaces, or a literal tab when it is 0 — which is what
 *  Go source wants. */
internal fun indentUnit(tabSize: Int): String =
    if (tabSize > 0) " ".repeat(tabSize) else "\t"

/**
 * Return continues the previous line's indentation.
 *
 * Detected on the value rather than on a key event, because an IME inserts the
 * newline through the text input session and never as a key. The test is "the
 * new text is the old text plus exactly one newline at the caret", which is
 * deliberately narrow: an Enter that *replaces* a selection changes the length
 * by something else and is left alone, and so is a paste containing newlines.
 * Both are cases where guessing would be worse than doing nothing.
 */
internal fun autoIndent(old: TextFieldValue, next: TextFieldValue, unit: String): TextFieldValue {
    if (next.text.length != old.text.length + 1) return next
    val caret = next.selection.start
    if (caret <= 0 || next.text[caret - 1] != '\n') return next
    val lineStart = next.text.lastIndexOf('\n', caret - 2) + 1
    val indent = next.text.substring(lineStart, caret - 1).takeWhile { it == ' ' || it == '\t' }
    if (indent.isEmpty()) return next
    return TextFieldValue(
        text = next.text.substring(0, caret) + indent + next.text.substring(caret),
        selection = TextRange(caret + indent.length),
    )
}

/** Replaces the selection with [text] and leaves the caret after it — the
 *  platform's own typing behaviour, re-implemented for the one key whose
 *  default had to be refused. */
internal fun insertInCode(value: TextFieldValue, text: String): TextFieldValue {
    val start = value.selection.min
    val end = value.selection.max
    return TextFieldValue(
        text = value.text.substring(0, start) + text + value.text.substring(end),
        selection = TextRange(start + text.length),
    )
}

/**
 * One core.RunEditorCommand, or null for a command this editor does not know.
 *
 * The selection is first widened to whole lines, because all three line
 * commands are line commands — indenting "the middle of line 4" means indenting
 * line 4 — and the rewritten block then takes the selection, so a second indent
 * indents the same lines rather than a range that has drifted under the first
 * one's inserted characters.
 */
internal fun applyCodeCommand(
    value: TextFieldValue,
    command: String,
    unit: String,
    commentPrefix: String,
): TextFieldValue? {
    val text = value.text
    val from = text.lastIndexOf('\n', value.selection.min - 1) + 1
    val nextNewline = text.indexOf('\n', value.selection.max)
    val to = if (nextNewline < 0) text.length else nextNewline
    val lines = text.substring(from, to).split("\n")

    val out: List<String> = when (command) {
        "indent" -> lines.map { unit + it }
        "outdent" -> lines.map { outdentCodeLine(it, unit) }
        "commentLine" -> {
            // A language with no line comment (JSON) sets an empty prefix and
            // gets a command that does nothing, rather than one that inserts a
            // marker making the document invalid.
            if (commentPrefix.isEmpty()) return null
            // The toggle is decided for the whole run, not per line: a
            // partly-commented block becomes fully commented rather than
            // inverting line by line, which is what makes the command its own
            // undo. Blank lines do not vote.
            val allCommented = lines.all {
                it.isBlank() || it.trimStart().startsWith(commentPrefix)
            }
            lines.map {
                if (allCommented) uncommentCodeLine(it, commentPrefix)
                else commentCodeLine(it, commentPrefix)
            }
        }
        else -> return null
    }

    val replaced = out.joinToString("\n")
    return TextFieldValue(
        text = text.substring(0, from) + replaced + text.substring(to),
        selection = TextRange(from, from + replaced.length),
    )
}

/** Removes one indent's worth of leading white space, and leaves a line that
 *  has none alone rather than eating a glyph. A leading tab goes whatever the
 *  unit is, because a buffer mixes them: a tab-indented file outdented by a
 *  four-space unit would otherwise lose nothing at all. */
internal fun outdentCodeLine(line: String, unit: String): String {
    if (line.startsWith(unit)) return line.substring(unit.length)
    if (line.startsWith("\t")) return line.substring(1)
    var i = 0
    while (i < unit.length && i < line.length && line[i] == ' ') i++
    return line.substring(i)
}

/** Inserts the prefix at the start of the line's *indentation*, not at column
 *  zero, so a commented block keeps the shape of the code it came from. A blank
 *  line stays blank: a file of "// " on its empty lines is trailing white space
 *  a formatter strips on the next save. */
internal fun commentCodeLine(line: String, prefix: String): String {
    if (line.isBlank()) return line
    val indent = line.takeWhile { it == ' ' || it == '\t' }
    return indent + prefix + " " + line.substring(indent.length)
}

/** Removes the first prefix and the single space that usually follows it. One
 *  space, not all of them: "//     aligned" is a comment whose own indentation
 *  is part of what it says. */
internal fun uncommentCodeLine(line: String, prefix: String): String {
    val at = line.indexOf(prefix)
    if (at < 0) return line
    var after = at + prefix.length
    if (after < line.length && line[after] == ' ') after++
    return line.substring(0, at) + line.substring(after)
}

/** A UTF-16 char offset as a byte offset into the UTF-8 value, which is the
 *  unit core.OnSelectionChange promises. */
internal fun utf8Offset(text: String, offset: Int): Int {
    val clamped = offset.coerceIn(0, text.length)
    return text.substring(0, clamped).toByteArray(Charsets.UTF_8).size
}
