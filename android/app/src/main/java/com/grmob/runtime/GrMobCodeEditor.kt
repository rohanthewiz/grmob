package com.grmob.runtime

import androidx.compose.foundation.horizontalScroll
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
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.input.key.KeyEventType
import androidx.compose.ui.input.key.key
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.input.key.type
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.text.SpanStyle
import androidx.compose.ui.text.TextRange
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.buildAnnotatedString
import androidx.compose.ui.text.withStyle
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
 * # The three rules, as they land here
 *
 *  1. **Echo guard.** Identical bookkeeping to GrMobTextField's
 *     `pendingEchoes`, one type wider: the local state is a `TextFieldValue`
 *     rather than a `String`, so an echo that is dropped leaves the *selection*
 *     alone as well as the text.
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

    val interactions = remember { MutableInteractionSource() }
    val focused by interactions.collectIsFocusedAsState()

    var buffer by remember { mutableStateOf(TextFieldValue(upstream)) }
    val pendingEchoes = remember { mutableListOf<String>() }
    var lastUpstream by remember { mutableStateOf(upstream) }

    // The echo guard. The argument is GrMobTextField's, unchanged: every value
    // this editor sends upstream is queued, an upstream change matching a
    // queued entry is our own edit coming back (drop the queue *through* the
    // match, because Go may coalesce renders and skip intermediates), and one
    // matching nothing we sent can only be Go speaking for itself and wins even
    // mid-typing.
    if (upstream != lastUpstream) {
        lastUpstream = upstream
        if (focused) {
            val echo = pendingEchoes.indexOf(upstream)
            if (echo >= 0) {
                repeat(echo + 1) { pendingEchoes.removeAt(0) }
            } else {
                buffer = TextFieldValue(upstream, TextRange(upstream.length))
                pendingEchoes.clear()
            }
        }
    }
    if (!focused) {
        // Go-owned while blurred; any queued echoes died with the focus session.
        pendingEchoes.clear()
        if (buffer.text != upstream) buffer = TextFieldValue(upstream)
    }

    // One place every local edit leaves by, so the echo ledger and the dispatch
    // can never get out of step with each other.
    val commit: (TextFieldValue) -> Unit = { next ->
        buffer = next
        if (onChange.isNotEmpty()) {
            pendingEchoes.add(next.text)
            runtime.textChanged(onChange, next.text)
        }
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
    )
    val transformation = GrMobCodeRows(node.children, base.color)

    val vertical = rememberScrollState()
    val horizontal = rememberScrollState()

    // The gutter is a sibling of the field, inside the same vertical scroll, so
    // number N stays beside line N with nothing measured. The horizontal scroll
    // wraps the field alone: the buffer pans sideways for a long line and the
    // numbers stay put.
    Row(s.boxModifier(extra).verticalScroll(vertical)) {
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
            Column(Modifier.width(gutterWidth).padding(end = 4.dp)) {
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
        Box(Modifier.horizontalScroll(horizontal)) {
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
                    autoCorrect = false,
                    capitalization = KeyboardCapitalization.None,
                ),
                modifier = Modifier.onPreviewKeyEvent { event ->
                    // Tab, which would otherwise move focus out of the editor
                    // and make indenting impossible. Hardware keyboards only —
                    // a soft keyboard has no Tab — which is why this is the one
                    // key handled here and Return is handled on the value.
                    if (event.type != KeyEventType.KeyDown || event.key != Key.Tab) {
                        return@onPreviewKeyEvent false
                    }
                    if (readOnly) return@onPreviewKeyEvent true
                    commit(insertInCode(buffer, indentUnit(tabSize)))
                    true
                },
            )
        }
    }
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
        // Identity, and it has to be: this is a colouring, so it adds spans and
        // never a character. Visual offset N is buffer offset N, which is what
        // keeps the caret where the user put it.
        return TransformedText(styled, OffsetMapping.Identity)
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
