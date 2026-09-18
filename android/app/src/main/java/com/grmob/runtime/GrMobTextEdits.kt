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
 * # When Go stamps nothing
 *
 * A field no edit has reached carries no editSeq; nor does anything a host
 * without the protocol renders. [upstream] then falls back to the value
 * queue this class replaced: a value we sent is an echo, and any other value
 * is Go speaking for itself.
 */
internal class TextEditLedger(anchor: String, epoch: Int) {
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
    fun upstream(value: String, ack: Int, goEpoch: Int, local: String): String? {
        if (ack == 0) return legacy(value)
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
        return rebaseEdit(basis, local, value)
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
 * Only insertions at either end are replayed. Those are what outruns a round
 * trip: a run of typed characters at the caret, which is almost always at
 * the end. Anything else (a deletion, an edit in the middle) has no single
 * place on the rewritten text it obviously belongs, and Go's text wins, as
 * it always did.
 *
 *   basis "beta,"   local "beta,gam"   rewrite ""     →  "gam"
 *   basis "ab"      local "xab"        rewrite "AB"   →  "xAB"
 *   basis "hello"   local "hell"       rewrite "Hello" → "Hello"
 */
internal fun rebaseEdit(basis: String, local: String, rewrite: String): String = when {
    local == basis -> rewrite
    local.startsWith(basis) -> rewrite + local.substring(basis.length)
    local.endsWith(basis) -> local.substring(0, local.length - basis.length) + rewrite
    else -> rewrite
}
