package com.grmob.runtime

/**
 * The host's half of the text-edit protocol (core/text_edit.go): what a
 * focused text field has sent Go and not yet heard back about, and how to
 * read Go's answer.
 *
 * Every edit goes out with a sequence number and the rewrite epoch this
 * field last adopted. Go's answer comes back as three props on the node:
 * `value`, `editSeq` (the last edit Go applied) and `editEpoch` (how many times
 * Go has rewritten the field). An epoch above ours means Go rewrote the text;
 * anything else is an echo of our own typing and is ignored.
 *
 * On a rewrite, the keystrokes still in flight were typed on text Go has
 * replaced, and Go will drop them. They are not lost: [upstream] replays
 * them onto Go's text ([rebaseEdit]) and the field sends the result as a new
 * edit at the new epoch.
 *
 *   anchor      pending (seq, text)                 local text
 *   "beta"      (5,"beta,") (6,"beta,g") (7,"beta,ga")   "beta,ga"
 *   Go: value "" editSeq 5 editEpoch 1
 *     basis = "beta,"  (the text as of seq 5)
 *     rebase "beta,ga" from "beta," onto ""  →  "ga", sent as seq 8, epoch 1
 *     Go drops 6 and 7 (epoch 0) and applies 8
 *
 * Plain Kotlin, with no Compose in it, so the rule is one piece of code the
 * composable calls rather than being spread through its body. The iOS
 * renderer has the same class under the same name.
 *
 * # Who uses it
 *
 * GrMobTextField (Input, InputPassword, NumericInput, TextArea),
 * GrMobCodeEditor and GrMobRichTextState. The rich-text editor constructs it
 * with `replay = false`: its value is a JSON document, and splicing the text
 * of one JSON string onto another makes something that is not a document. It
 * adopts Go's rewrite as it stands, which is what its value queue did.
 *
 * # When Go stamps nothing
 *
 * Before this host has sent its first edit, Go stamps no field; nor does
 * anything a host without the protocol renders. [upstream] then falls back to
 * the value queue this class replaced: a value we sent is an echo, and any
 * other value is Go speaking for itself.
 *
 * Stamped means the node carries editEpoch at all, and not that editSeq is
 * nonzero. Once this host is sequenced, Go stamps every field from its first
 * render, so a field Go has rewritten before its first edit arrives with
 * editSeq 0 and editEpoch 1, and that is a rewrite (core/text_edit.go, "Every
 * field, once the host is sequenced": comps.PINInput lost keys to it).
 */
internal class TextEditLedger(anchor: String, epoch: Int, private val replay: Boolean = true) {
    /** The rewrite epoch this field has adopted; sent with every edit. */
    var epoch: Int = epoch
        private set

    /**
     * The text the pending edits were typed on: the value as of the last
     * edit Go applied, or of the last rewrite or focus, whichever came
     * later.
     */
    private var anchor: String = anchor

    /** Edits sent and not yet acknowledged, oldest first. */
    private val pending = ArrayDeque<Pair<Int, String>>()

    /**
     * The text the last rewrite was rebased from: what Go had read before it
     * rewrote. Kept for [rebaseCaret], which must merge from the same basis
     * [rebaseEdit] did, or the two would disagree about where the typing was.
     */
    var lastBasis: String = anchor
        private set

    /** Records an edit just sent under [seq]. */
    fun sent(seq: Int, text: String) {
        pending.addLast(seq to text)
    }

    /**
     * Forgets everything in flight and takes Go's value and epoch as they
     * stand. For a field that is not focused, which is Go's to draw.
     */
    fun reset(text: String, epoch: Int) {
        pending.clear()
        anchor = text
        this.epoch = epoch
    }

    /**
     * Reads one change to the node's value, editSeq or editEpoch while the
     * field is focused. Returns the text the field should now show, or null
     * to keep its own. When the returned text differs from [value], the
     * caller must send it as a new edit: it carries typing Go has not seen.
     */
    fun upstream(value: String, ack: Int, goEpoch: Int, local: String, stamped: Boolean): String? {
        if (!stamped) return legacy(value)
        // Let go of every edit Go has applied. The last of them is the text Go
        // read before this render, which is the base of any rewrite.
        var basis = anchor
        while (pending.isNotEmpty() && pending.first().first <= ack) {
            basis = pending.removeFirst().second
        }
        anchor = basis
        if (goEpoch <= epoch) return null
        epoch = goEpoch
        pending.clear()
        anchor = value
        lastBasis = basis
        return if (replay) rebaseEdit(basis, local, value) else value
    }

    /**
     * The value queue, for a node with no stamps. Go may coalesce renders,
     * so an echo drops the queue through its own entry and not just the
     * head.
     */
    private fun legacy(value: String): String? {
        val echo = pending.indexOfFirst { it.second == value }
        if (echo >= 0) {
            repeat(echo + 1) { pending.removeFirst() }
            return null
        }
        pending.clear()
        anchor = value
        return value
    }
}

/**
 * Replays the typing from [basis] to [local] onto Go's [rewrite].
 *
 * A three-way merge of two single-span edits. internal/rebasefixture is the
 * statement of the rule (read its package doc for the reasoning), and
 * android/verify runs this copy against its case table; ios/verify runs the
 * Swift copy against the same one.
 *
 * Each of basis→local (the user's typing) and basis→rewrite (Go's change) is
 * read as one contiguous span. The user's span is carried across Go's change
 * and spliced into the rewrite:
 *
 *   basis "HELLOaWORLD"  local "HELLOabWORLD"  rewrite "HELLOAWORLD"
 *   user's span [6,6) → "b"     Go's span [5,6) → "A"
 *   6 is at the end of Go's span, so it maps to 6   →   "HELLOAbWORLD"
 *
 * Where the user's span sits inside text Go replaced with text of another
 * length, there is no telling where it belongs, and Go's text wins, as it
 * always did.
 *
 *   basis "beta,"   local "beta,gam"   rewrite ""       →  "gam"
 *   basis "ab"      local "xab"        rewrite "AB"     →  "xAB"
 *   basis "hello"   local "hell"       rewrite "Hello"  →  "Hell"
 *   basis "beta,"   local "bet"        rewrite ""       →  ""   (Go wins)
 *
 * The first version replayed only an insertion at either end of [basis], and
 * two keys typed quickly mid-text under an UPPERCASE transform lost the second.
 */
internal fun rebaseEdit(basis: String, local: String, rewrite: String): String =
    rebaseMerge(basis, local, rewrite)?.text ?: rewrite

/**
 * Where the caret goes after [rebaseEdit], for a host that owns its selection
 * (the code editor; a plain field places its caret from two texts, [carryCaret]).
 * [caret] is the caret in [local].
 *
 * It follows the typing, since that is where the user was:
 *
 *   before the user's span   mapped through Go's change like any basis offset
 *   inside the replayed text keeps its place in it
 *   after the user's span    keeps its place in the text after
 *   Go's text won, or the    the end of Go's text, which is where a rewrite
 *   caret sat in text Go     always put the caret before there was a rebase
 *   replaced
 */
internal fun rebaseCaret(basis: String, local: String, rewrite: String, caret: Int): Int {
    val m = rebaseMerge(basis, local, rewrite) ?: return rewrite.length
    val at = when {
        caret <= m.userStart -> mapOffset(caret, basis.length, m.goStart, m.goEnd, rewrite.length)
        caret < m.userStart + m.replayed.length -> m.spliceStart + (caret - m.userStart)
        else -> {
            val mapped = mapOffset(caret - local.length + basis.length, basis.length,
                m.goStart, m.goEnd, rewrite.length)
            if (mapped < 0) -1 else mapped - m.spliceEnd + m.spliceStart + m.replayed.length
        }
    }
    return if (at < 0 || at > m.text.length || splitsPair(m.text, at)) m.text.length else at
}

/**
 * Where a focused plain field's caret goes when new text is written into it:
 * [before] is what the field showed, [after] what it shows now, [caret] the
 * caret in [before]. internal/rebasefixture's `Carry` is the statement of the
 * rule and android/verify runs this copy against its case table.
 *
 * [rebaseCaret] is not the answer for a plain field. It follows typing that
 * was in flight, and with none (the ordinary case at human speed) it sends
 * the caret to the end, which breaks typing mid-text under an UPPERCASE
 * onChange. So the field places its caret from the two texts alone: the
 * change is read as one differing span and the caret is carried across it by
 * [mapOffset], as the ends of a replayed span are.
 *
 *   "1"     → "(1"       caret 1 → 2   Go put a literal before the caret
 *   "(5556" → "(555) 6"  caret 5 → 7
 *   "HELLOa WORLD" → "HELLOA WORLD"    caret 6 → 6   length kept
 *   "beta," → ""         caret 5 → 0
 *
 * The field used to keep the caret's raw offset, clamped. That is the same
 * answer whenever Go's change is at or after the caret or keeps the length,
 * which is every rewrite there was until comps.MaskedInput: a formatting
 * onChange inserts before the caret, and on the emulator the keys 1 2 3 4 5 6
 * typed a second apart read back "(234) 651". The browser's writeFieldValue
 * and the iOS field's write already carried the caret this way.
 */
internal fun carryCaret(before: String, after: String, caret: Int): Int {
    val (prefix, suffix) = commonSpan(before, after)
    var at = mapOffset(caret, before.length, prefix, before.length - suffix, after.length)
    // Inside a span that changed length: the end of the new span, the nearest
    // place still after the text the caret was after.
    if (at < 0) at = after.length - suffix
    return if (at > after.length || splitsPair(after, at)) after.length else at
}

/** One successful merge and the offsets [rebaseCaret] needs from it. The
 *  fields are internal/rebasefixture's `merged`, under the same names. */
private class RebaseMerge(
    val text: String,
    /** Where the user's span starts, in basis and local alike. */
    val userStart: Int,
    /** The user's text: local's side of the user's span. */
    val replayed: String,
    /** Go's span in basis. */
    val goStart: Int,
    val goEnd: Int,
    /** Where the user's span landed in the rewrite. */
    val spliceStart: Int,
    val spliceEnd: Int,
)

/** [rebaseEdit]'s arithmetic; null where Go's text wins. */
private fun rebaseMerge(basis: String, local: String, rewrite: String): RebaseMerge? {
    // Nothing typed since the edit Go read: nothing to replay.
    if (local == basis) return null
    val (up, us) = commonSpan(basis, local)
    val (gp, gs) = commonSpan(basis, rewrite)
    val goEnd = basis.length - gs
    val spliceStart = mapOffset(up, basis.length, gp, goEnd, rewrite.length)
    val spliceEnd = mapOffset(basis.length - us, basis.length, gp, goEnd, rewrite.length)
    // A splice point between the halves of a surrogate pair is no telling
    // either: the length-kept arm maps one unit at a time.
    if (spliceStart < 0 || spliceEnd < 0 || spliceStart > spliceEnd ||
        splitsPair(rewrite, spliceStart) || splitsPair(rewrite, spliceEnd)
    ) return null
    val replayed = local.substring(up, local.length - us)
    return RebaseMerge(
        text = rewrite.substring(0, spliceStart) + replayed + rewrite.substring(spliceEnd),
        userStart = up, replayed = replayed, goStart = gp, goEnd = goEnd,
        spliceStart = spliceStart, spliceEnd = spliceEnd,
    )
}

/**
 * Carries basis offset [i] across Go's change of basis[goStart, goEnd), which
 * made a [basisLen]-unit basis into a [rewriteLen]-unit rewrite; -1 where there
 * is no telling. "After Go's span" is tested first: where both inserted at one
 * point, the user's text goes after Go's (comps.PINInput's "14").
 */
private fun mapOffset(i: Int, basisLen: Int, goStart: Int, goEnd: Int, rewriteLen: Int): Int = when {
    i >= goEnd -> i + rewriteLen - basisLen
    i <= goStart -> i
    // Inside a change that kept the length: a character-for-character
    // transform such as UPPERCASE, where offset i is still offset i.
    rewriteLen == basisLen -> i
    else -> -1
}

/**
 * The lengths of [a] and [b]'s common prefix and common suffix, the suffix
 * limited so the two never overlap, and neither ending inside a surrogate
 * pair.
 */
private fun commonSpan(a: String, b: String): Pair<Int, Int> {
    val n = minOf(a.length, b.length)
    var prefix = 0
    while (prefix < n && a[prefix] == b[prefix]) prefix++
    if (prefix > 0 && a[prefix - 1].isHighSurrogate()) prefix--
    var suffix = 0
    val limit = n - prefix
    while (suffix < limit && a[a.length - 1 - suffix] == b[b.length - 1 - suffix]) suffix++
    if (suffix > 0 && a[a.length - suffix].isLowSurrogate()) suffix--
    return prefix to suffix
}

/** Whether offset [i] of [s] falls between the two halves of a surrogate pair. */
private fun splitsPair(s: String, i: Int): Boolean =
    i > 0 && i < s.length && s[i - 1].isHighSurrogate() && s[i].isLowSurrogate()
