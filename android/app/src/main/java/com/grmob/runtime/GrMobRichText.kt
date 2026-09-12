package com.grmob.runtime

import android.graphics.Typeface
import android.text.Editable
import android.text.SpannableStringBuilder
import android.text.Spannable
import android.text.Spanned
import android.text.TextWatcher
import android.text.style.LeadingMarginSpan
import android.text.style.QuoteSpan
import android.text.style.RelativeSizeSpan
import android.text.style.StrikethroughSpan
import android.text.style.StyleSpan
import android.text.style.TypefaceSpan
import android.text.style.URLSpan
import android.text.style.UnderlineSpan
import android.util.TypedValue
import android.view.Gravity
import android.widget.EditText
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.viewinterop.AndroidView
import org.json.JSONArray
import org.json.JSONObject

/**
 * core.RichTextEditor: an editable formatted document.
 *
 * # Why an EditText and not a BasicTextField
 *
 * This is the one node in the runtime that reaches past Compose on purpose, and
 * the reason is `Spannable`. A formatted, *editable* document is a string plus
 * a set of spans over it — which is exactly what `Editable` is, and has been
 * since Android 1 — with a span type already in the framework for every mark
 * and every paragraph treatment this model has: StyleSpan, UnderlineSpan,
 * StrikethroughSpan, TypefaceSpan, URLSpan, QuoteSpan, LeadingMarginSpan.
 *
 * Compose's `BasicTextField` holds a plain `String` and produces its styling
 * through a `VisualTransformation`, which is right for a code editor (see
 * GrMobCodeEditor.kt) because the colouring there is *derived* from the text.
 * Here the formatting is not derived from anything: it is part of the value, and
 * the user edits it directly. Building block structure — headings, bullets,
 * hanging indents, a quote bar — into one AnnotatedString rebuilt on every
 * keystroke is the riskiest thing this whole design could ask for, and the
 * classic-view route removes it. AndroidView is how osmdroid is hosted, so the
 * escape hatch is one the runtime already uses.
 *
 * # The mapping
 *
 *	  richtext.Doc                 Spannable
 *	  ────────────────────────────────────────────────────────────────
 *	  Run.Bold / .Italic       StyleSpan(BOLD / ITALIC)
 *	  Run.Underline / .Strike  UnderlineSpan / StrikethroughSpan
 *	  Run.Code                 TypefaceSpan("monospace"), plus GrMobCodeSpan
 *	                           so the reverse mapping reads rather than guesses
 *	  Run.Link                 URLSpan
 *	  Block.Kind               GrMobBlockSpan over the paragraph, plus the
 *	                           size/indent/quote spans that draw it, plus a
 *	                           drawn prefix for the two list kinds
 *
 * GrMobBlockSpan and GrMobPrefixSpan are markers with no drawing of their own.
 * They exist for the same reason iOS's two custom attribute keys do: without
 * them a heading would have to be recognized by its size — a guess that breaks
 * when a theme changes one — and the bullet drawn in front of a list item would
 * come back as part of the user's text.
 *
 * # The compilation gap, stated
 *
 * Nothing in this repository compiles Kotlin that imports Android, so this file
 * is held to the contract textually by `mobile/verify/richtext_test.go` and by a
 * device pass. Same caveat GrMobMapView.kt and GrMobCodeEditor.kt carry.
 */
@Composable
internal fun GrMobRichTextEditor(node: GrMobNode, extra: Modifier) {
    val runtime = LocalGrMobRuntime.current
    val state = remember { GrMobRichTextState() }

    AndroidView(
        factory = { context ->
            EditText(context).apply {
                background = null
                gravity = Gravity.TOP or Gravity.START
                setPadding(0, 0, 0, 0)
                // A document is prose, so it wraps and takes as many lines as it
                // needs — the opposite of the code editor next door.
                isSingleLine = false
                setHorizontallyScrolling(false)
                state.attach(this, runtime)
            }
        },
        update = { view ->
            state.runtime = runtime
            state.onChange = node.stringProp("onChange")
            state.onSelectionChange = node.stringProp("onSelectionChange")
            state.base = GrMobRichStyle(node.style)
            view.setTextColor(state.base.ink)
            view.setTextSize(TypedValue.COMPLEX_UNIT_SP, state.base.size)
            // isFocusable stays true even when read-only: a read-only document
            // is still content the reader is meant to select and copy, which is
            // the whole difference between read-only and disabled.
            val readOnly = node.boolProp("readOnly")
            view.isCursorVisible = !readOnly
            view.setTextIsSelectable(readOnly)
            state.readOnly = readOnly
            view.hint = node.stringProp("placeholder")
            state.applyDoc(node.stringProp("doc"))
            state.runCommand(node.intProp("editorEpoch"), node.stringProp("editorCommand"))
        },
        modifier = node.style.boxModifier(extra),
    )
}

/** The sizes and the ink a document is drawn in, resolved once from the Go style. */
internal class GrMobRichStyle(style: GrMobStyle?) {
    val size: Float = if (style != null && style.fontSize > 0f) style.fontSize else 17f
    val ink: Int = (style?.textColor ?: Color.Black).toArgb()
}

/** A marker for the block kind a paragraph is. Draws nothing; see the file doc. */
internal class GrMobBlockSpan(val kind: String)

/** A marker for a list prefix this renderer drew, which is not the user's text. */
internal class GrMobPrefixSpan

/** A marker for inline code, carried beside the monospace typeface so that the
 *  reverse mapping reads the mark rather than inferring it from a font. */
internal class GrMobCodeSpan

/**
 * Everything about one editor that is a decision rather than a view: the
 * document it holds, the echo ledger, the command epoch, and the two
 * directions of the mapping.
 *
 * Held in a `remember` across recompositions rather than on the view, because
 * the view is created by a factory that Compose may call again and the state has
 * to survive that.
 */
internal class GrMobRichTextState {
    var runtime: GrMobRuntime? = null
    var onChange = ""
    var onSelectionChange = ""
    var readOnly = false
    var base = GrMobRichStyle(null)

    private var view: EditText? = null

    /** The document this editor holds, in the wire's own shape. Kept as the wire
     *  shape rather than as Kotlin data classes so there is one representation
     *  on this host and it is the one that crosses — the same decision the web
     *  runtime and the iOS renderer make. */
    private var doc: JSONObject = JSONObject().put("b", JSONArray())

    /** The JSON of every document sent upstream and not yet seen come back. */
    private val pendingEchoes = ArrayList<String>()
    private var lastEpoch: Int? = null
    private var lastSelection = ""

    /** Set while this class is writing the Editable, so the TextWatcher does not
     *  report the renderer's own edit as the user's. */
    private var applying = false

    fun attach(editText: EditText, runtime: GrMobRuntime?) {
        view = editText
        this.runtime = runtime
        editText.addTextChangedListener(object : TextWatcher {
            override fun beforeTextChanged(s: CharSequence?, start: Int, count: Int, after: Int) {}
            override fun onTextChanged(s: CharSequence?, start: Int, before: Int, count: Int) {}
            override fun afterTextChanged(s: Editable?) {
                if (applying || s == null) return
                doc = GrMobRichMapper.document(s)
                send()
            }
        })
        // The selection report, which is what a toolbar draws its pressed state
        // from. EditText has no selection listener, so it rides the same
        // afterTextChanged plus an explicit poll from the update block; the
        // dedupe below is what keeps that from costing a render pass per poll.
        editText.setOnClickListener { reportSelection() }
        editText.setOnFocusChangeListener { _, _ -> reportSelection() }
    }

    // --- Go -> the document ------------------------------------------------

    /**
     * The echo guard, over the document's JSON rather than a string of text.
     *
     * Same three arms GrMobTextField's has: an echo is dropped, a rewrite lands
     * even mid-typing, and a blurred editor is Go's outright. The comparison is
     * on the JSON string because Go marshals with a fixed key order, so the same
     * document is always the same bytes.
     */
    fun applyDoc(json: String) {
        val editText = view ?: return
        if (editText.isFocused) {
            val echo = pendingEchoes.indexOf(json)
            if (echo >= 0) {
                repeat(echo + 1) { pendingEchoes.removeAt(0) }
                return
            }
        }
        pendingEchoes.clear()
        if (doc.toString() == json) return
        val parsed = try {
            JSONObject(json)
        } catch (e: Exception) {
            // A doc prop that is not a document leaves the editor as it was,
            // rather than emptying a note because one patch was malformed.
            return
        }
        doc = parsed
        rebuild(editText.selectionStart, editText.selectionEnd)
    }

    /**
     * Rebuilds the Editable from the document, restoring a selection afterwards.
     *
     * Only reached when the document's *structure* changed — a rewrite from Go,
     * or a block command — because there is no in-place edit that expresses one.
     * Every mark command avoids it; see runCommand.
     */
    private fun rebuild(selectionStart: Int, selectionEnd: Int) {
        val editText = view ?: return
        applying = true
        editText.text = GrMobRichMapper.spannable(doc, base)
        val length = editText.text?.length ?: 0
        editText.setSelection(selectionStart.coerceIn(0, length), selectionEnd.coerceIn(0, length))
        applying = false
    }

    // --- The document -> Go -------------------------------------------------

    private fun send() {
        val json = doc.toString()
        pendingEchoes.add(json)
        if (onChange.isNotEmpty()) runtime?.textChanged(onChange, json)
        reportSelection()
    }

    /**
     * The caret and the formatting active at it.
     *
     * The offsets are UTF-8 byte offsets into the document's plain text, which is
     * core.RichSelection's promise and the one coordinate system all four hosts
     * can produce — Android counts UTF-16 chars, so the conversion is this
     * method's job beyond the dispatch.
     */
    fun reportSelection() {
        if (onSelectionChange.isEmpty()) return
        val editText = view ?: return
        val text = editText.text ?: return
        val start = editText.selectionStart.coerceIn(0, text.length)
        val end = editText.selectionEnd.coerceIn(0, text.length)
        // A bare caret reports the marks to its left, which is what the next
        // character typed there would inherit.
        val probeStart = if (end > start) start else (start - 1).coerceAtLeast(0)
        val probeEnd = if (end > start) end else start

        val marks = GrMobRichMapper.marksIn(text, probeStart, probeEnd)
        val payload = JSONObject()
            .put("s", GrMobRichMapper.utf8Offset(text, start))
            .put("e", GrMobRichMapper.utf8Offset(text, end))
            .put("marks", JSONArray(marks.names))
            .put("link", marks.link)
            .put("block", GrMobRichMapper.blockKindAt(text, start))
            .toString()
        if (payload == lastSelection) return
        lastSelection = payload
        runtime?.textChanged(onSelectionChange, payload)
    }

    // --- Commands -----------------------------------------------------------

    /**
     * One core.RunEditorCommand.
     *
     * See GrMobCodeEditor.kt for why an editor adopts a standing epoch without
     * running it; the rule is the same on all four hosts.
     */
    fun runCommand(epoch: Int, command: String) {
        val previous = lastEpoch
        lastEpoch = epoch
        if (previous == null || epoch == 0 || epoch == previous) return
        val editText = view ?: return
        if (readOnly) return
        val text = editText.text ?: return

        when {
            command == "undo" || command == "redo" -> {
                // TextView carries the platform's own editing history and
                // exposes it only through the context-menu action ids. That
                // history covers typing, which is the overwhelming majority of
                // what a writer undoes; the span edits this file makes are not
                // in it, which is the one place the three live hosts differ in
                // what undo reaches. The web runtime keeps a stack of Docs
                // instead, because a browser's native history does not survive
                // its rebuilds at all.
                editText.onTextContextMenuItem(
                    if (command == "undo") android.R.id.undo else android.R.id.redo
                )
                doc = GrMobRichMapper.document(text)
                send()
            }
            command.startsWith("block:") -> {
                // A block kind changes a paragraph's prefix and its indentation,
                // so there is no in-place edit that expresses it: read out,
                // transform, rebuild, restore.
                val start = editText.selectionStart
                val end = editText.selectionEnd
                doc = GrMobRichMapper.document(text)
                doc = GrMobRichMapper.setBlockKind(
                    doc, GrMobRichMapper.blockSpan(text, start, end),
                    command.removePrefix("block:"),
                )
                rebuild(start, end)
                doc = GrMobRichMapper.document(editText.text ?: text)
                send()
            }
            else -> {
                if (!GrMobRichMapper.applyMark(text, editText.selectionStart,
                        editText.selectionEnd, command, base)) {
                    // false is "this editor does not know that command", which is
                    // a no-op: a toolbar that outgrew its editor must not break
                    // the screen.
                    return
                }
                doc = GrMobRichMapper.document(text)
                send()
            }
        }
    }
}

/** The mapping, both ways, plus the span surgery the mark commands do. */
internal object GrMobRichMapper {

    /** richtext.Doc -> Spannable. See GrMobRichTextEditor's doc for the table. */
    fun spannable(doc: JSONObject, base: GrMobRichStyle): SpannableStringBuilder {
        val out = SpannableStringBuilder()
        val blocks = doc.optJSONArray("b") ?: JSONArray()
        var ordinal = 0

        for (i in 0 until blocks.length()) {
            val block = blocks.optJSONObject(i) ?: continue
            val kind = block.optString("k", "p").ifEmpty { "p" }
            ordinal = if (kind == "numbered") ordinal + 1 else 0
            val blockStart = out.length

            // The list prefix, drawn as text because an EditText has no list
            // rendering that survives editing — and marked so the reverse
            // mapping drops it rather than handing the user back their bullets.
            if (kind == "bullet" || kind == "numbered") {
                val prefix = if (kind == "bullet") "•\t" else "$ordinal.\t"
                val at = out.length
                out.append(prefix)
                out.setSpan(GrMobPrefixSpan(), at, out.length, Spanned.SPAN_EXCLUSIVE_EXCLUSIVE)
            }

            val runs = block.optJSONArray("r") ?: JSONArray()
            for (j in 0 until runs.length()) {
                val run = runs.optJSONObject(j) ?: continue
                val text = run.optString("t", "")
                if (text.isEmpty()) continue
                val at = out.length
                out.append(text)
                if (truthy(run.opt("b"))) span(out, StyleSpan(Typeface.BOLD), at)
                if (truthy(run.opt("i"))) span(out, StyleSpan(Typeface.ITALIC), at)
                if (truthy(run.opt("u"))) span(out, UnderlineSpan(), at)
                if (truthy(run.opt("s"))) span(out, StrikethroughSpan(), at)
                if (truthy(run.opt("c"))) {
                    span(out, TypefaceSpan("monospace"), at)
                    span(out, GrMobCodeSpan(), at)
                }
                val link = run.optString("l", "")
                if (link.isNotEmpty()) span(out, URLSpan(link), at)
            }

            // The paragraph treatments, applied over the whole block including
            // its prefix, plus the marker the reverse mapping reads.
            out.setSpan(GrMobBlockSpan(kind), blockStart, out.length,
                Spanned.SPAN_EXCLUSIVE_EXCLUSIVE)
            for (paragraph in paragraphSpans(kind, base)) {
                out.setSpan(paragraph, blockStart, out.length, Spanned.SPAN_EXCLUSIVE_EXCLUSIVE)
            }

            if (i < blocks.length() - 1) out.append("\n")
        }
        return out
    }

    /**
     * The spans that *draw* a block kind, as opposed to the marker that records
     * it. Headings step down from the base size rather than naming three sizes,
     * so a theme that changes the body size moves the whole scale with it.
     *
     * The hanging indent on the list kinds is what makes a wrapped item line up
     * under its own text rather than back under its bullet — the one thing a
     * drawn prefix cannot do for itself.
     */
    private fun paragraphSpans(kind: String, base: GrMobRichStyle): List<Any> = when (kind) {
        "h1" -> listOf(RelativeSizeSpan(1.6f), StyleSpan(Typeface.BOLD))
        "h2" -> listOf(RelativeSizeSpan(1.35f), StyleSpan(Typeface.BOLD))
        "h3" -> listOf(RelativeSizeSpan(1.15f), StyleSpan(Typeface.BOLD))
        "bullet", "numbered" -> listOf(LeadingMarginSpan.Standard(0, (base.size * 1.4f).toInt()))
        "quote" -> listOf(QuoteSpan())
        "code" -> listOf(TypefaceSpan("monospace"),
            LeadingMarginSpan.Standard((base.size * 0.5f).toInt()))
        else -> emptyList()
    }

    private fun span(out: SpannableStringBuilder, what: Any, from: Int) {
        out.setSpan(what, from, out.length, Spanned.SPAN_EXCLUSIVE_EXCLUSIVE)
    }

    /**
     * Spannable -> richtext.Doc.
     *
     * A reading rather than an inference for everything this renderer put there
     * — the block kind off GrMobBlockSpan, the drawn prefixes off
     * GrMobPrefixSpan, inline code off GrMobCodeSpan. What is inferred is only
     * what the *platform* owns: the style spans a paste or a system control may
     * have added.
     */
    fun document(text: CharSequence): JSONObject {
        val spanned = text as? Spanned
        val blocks = JSONArray()
        var paragraphStart = 0
        while (paragraphStart <= text.length) {
            var end = text.indexOf('\n', paragraphStart)
            if (end < 0) end = text.length

            val block = JSONObject()
            val runs = JSONArray()
            var kind = "p"
            if (spanned != null && end > paragraphStart) {
                kind = spanned.getSpans(paragraphStart, paragraphStart + 1, GrMobBlockSpan::class.java)
                    .firstOrNull()?.kind ?: "p"
                var at = paragraphStart
                while (at < end) {
                    val next = spanned.nextSpanTransition(at, end, null)
                    if (spanned.getSpans(at, next, GrMobPrefixSpan::class.java).isEmpty()) {
                        val run = JSONObject().put("t", text.subSequence(at, next).toString())
                        for (style in spanned.getSpans(at, next, StyleSpan::class.java)) {
                            if (style.style and Typeface.BOLD != 0) run.put("b", 1)
                            if (style.style and Typeface.ITALIC != 0) run.put("i", 1)
                        }
                        if (spanned.getSpans(at, next, UnderlineSpan::class.java).isNotEmpty()) run.put("u", 1)
                        if (spanned.getSpans(at, next, StrikethroughSpan::class.java).isNotEmpty()) run.put("s", 1)
                        if (spanned.getSpans(at, next, GrMobCodeSpan::class.java).isNotEmpty()) run.put("c", 1)
                        spanned.getSpans(at, next, URLSpan::class.java).firstOrNull()?.let {
                            run.put("l", it.url)
                        }
                        runs.put(run)
                    }
                    at = next
                }
            }
            block.put("r", merge(runs))
            // Paragraph is the wire's absent kind, so writing it explicitly would
            // make a document that round-trips through here differ from one Go
            // marshalled — same bytes, different keys.
            if (kind != "p") block.put("k", kind)
            blocks.put(block)

            if (end >= text.length) break
            paragraphStart = end + 1
        }
        return JSONObject().put("b", blocks)
    }

    /**
     * merge collapses adjacent runs with identical formatting and drops the
     * empty ones — the maximal-run rule, which keeps a document from growing a
     * run boundary on every keystroke while looking identical on screen.
     */
    private fun merge(runs: JSONArray): JSONArray {
        val out = JSONArray()
        for (i in 0 until runs.length()) {
            val run = runs.optJSONObject(i) ?: continue
            val text = run.optString("t", "")
            if (text.isEmpty()) continue
            val previous = if (out.length() > 0) out.optJSONObject(out.length() - 1) else null
            if (previous != null && sameMarks(previous, run)) {
                previous.put("t", previous.optString("t", "") + text)
                continue
            }
            out.put(run)
        }
        return out
    }

    private fun sameMarks(a: JSONObject, b: JSONObject): Boolean {
        for (key in listOf("b", "i", "u", "s", "c")) {
            if (truthy(a.opt(key)) != truthy(b.opt(key))) return false
        }
        return a.optString("l", "") == b.optString("l", "")
    }

    /** The block indices the selection touches, which is what a block command
     *  acts on. */
    fun blockSpan(text: CharSequence, start: Int, end: Int): IntRange {
        val head = text.subSequence(0, start.coerceIn(0, text.length)).count { it == '\n' }
        val tail = text.subSequence(0, end.coerceIn(0, text.length)).count { it == '\n' }
        return head..maxOf(head, tail)
    }

    /** setBlockKind is the one document transformation this host performs on the
     *  model rather than on the text, because a block kind has no in-place
     *  spelling. */
    fun setBlockKind(doc: JSONObject, span: IntRange, kind: String): JSONObject {
        val blocks = doc.optJSONArray("b") ?: JSONArray()
        for (i in span) {
            val block = blocks.optJSONObject(i) ?: continue
            if (kind == "p") block.remove("k") else block.put("k", kind)
        }
        return JSONObject().put("b", blocks)
    }

    /** The marks active over a range, in the shape the selection report wants. */
    class Marks(val names: List<String>, val link: String)

    fun marksIn(text: CharSequence, start: Int, end: Int): Marks {
        val spanned = text as? Spanned ?: return Marks(emptyList(), "")
        if (end <= start) return Marks(emptyList(), "")
        val names = ArrayList<String>()
        for (style in spanned.getSpans(start, end, StyleSpan::class.java)) {
            if (style.style and Typeface.BOLD != 0 && "bold" !in names) names.add("bold")
            if (style.style and Typeface.ITALIC != 0 && "italic" !in names) names.add("italic")
        }
        if (spanned.getSpans(start, end, UnderlineSpan::class.java).isNotEmpty()) names.add("underline")
        if (spanned.getSpans(start, end, StrikethroughSpan::class.java).isNotEmpty()) names.add("strike")
        if (spanned.getSpans(start, end, GrMobCodeSpan::class.java).isNotEmpty()) names.add("code")
        val link = spanned.getSpans(start, end, URLSpan::class.java).firstOrNull()?.url ?: ""
        return Marks(names, link)
    }

    fun blockKindAt(text: CharSequence, at: Int): String {
        val spanned = text as? Spanned ?: return "p"
        val position = at.coerceIn(0, maxOf(0, text.length - 1))
        return spanned.getSpans(position, position + 1, GrMobBlockSpan::class.java)
            .firstOrNull()?.kind ?: "p"
    }

    /**
     * A mark command, applied to the Spannable in place — which preserves the
     * caret and costs no rebuild.
     *
     * An empty selection sets a zero-length span with INCLUSIVE_INCLUSIVE, which
     * is Android's own spelling of typing attributes: the span has no text in it
     * yet and grows to cover whatever is typed at that position. That is what
     * makes "press bold, then type" work here, and it is the reason this host
     * needs nothing like the web runtime's pending-mark bookkeeping.
     *
     * Returns false for a command this editor does not know.
     */
    fun applyMark(text: Editable, start: Int, end: Int, command: String,
                  base: GrMobRichStyle): Boolean {
        val from = minOf(start, end).coerceIn(0, text.length)
        val to = maxOf(start, end).coerceIn(0, text.length)

        if (command.startsWith("link:")) {
            if (to <= from) return true
            for (existing in text.getSpans(from, to, URLSpan::class.java)) text.removeSpan(existing)
            text.setSpan(URLSpan(command.removePrefix("link:")), from, to,
                Spanned.SPAN_EXCLUSIVE_EXCLUSIVE)
            return true
        }
        if (command == "unlink") {
            for (existing in text.getSpans(from, to, URLSpan::class.java)) text.removeSpan(existing)
            return true
        }

        // All, not any: selecting a sentence with one bold word in it and
        // pressing bold makes the sentence bold rather than unbolding the word.
        val on = !marksIn(text, from, to).names.contains(command)
        val flags = if (to > from) Spanned.SPAN_EXCLUSIVE_EXCLUSIVE else Spanned.SPAN_INCLUSIVE_INCLUSIVE

        fun toggle(make: () -> Any, existing: Class<*>, matches: (Any) -> Boolean = { true }) {
            for (span in text.getSpans(from, to, existing)) {
                if (matches(span)) text.removeSpan(span)
            }
            if (on) text.setSpan(make(), from, to, flags)
        }

        when (command) {
            "bold" -> toggle({ StyleSpan(Typeface.BOLD) }, StyleSpan::class.java) {
                (it as StyleSpan).style and Typeface.BOLD != 0
            }
            "italic" -> toggle({ StyleSpan(Typeface.ITALIC) }, StyleSpan::class.java) {
                (it as StyleSpan).style and Typeface.ITALIC != 0
            }
            "underline" -> toggle({ UnderlineSpan() }, UnderlineSpan::class.java)
            "strike" -> toggle({ StrikethroughSpan() }, StrikethroughSpan::class.java)
            "code" -> {
                toggle({ TypefaceSpan("monospace") }, TypefaceSpan::class.java)
                toggle({ GrMobCodeSpan() }, GrMobCodeSpan::class.java)
            }
            else -> return false
        }
        return true
    }

    /** A UTF-16 char offset as a byte offset into the UTF-8 text, which is the
     *  unit core.RichSelection promises. */
    fun utf8Offset(text: CharSequence, offset: Int): Int {
        val clamped = offset.coerceIn(0, text.length)
        return text.subSequence(0, clamped).toString().toByteArray(Charsets.UTF_8).size
    }

    /** truthy reads a wire mark, which core writes as 1 and a hand-written
     *  document may spell as `true`. Both are accepted, for richtext's own
     *  markFlag reason. */
    private fun truthy(value: Any?): Boolean = when (value) {
        null, JSONObject.NULL -> false
        is Number -> value.toInt() != 0
        is Boolean -> value
        is String -> value.isNotEmpty() && value != "0" && value != "false"
        else -> false
    }
}
