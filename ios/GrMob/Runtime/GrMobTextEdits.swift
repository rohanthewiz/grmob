import Foundation

/// The host's half of the text-edit protocol (core/text_edit.go): what a
/// focused text field has sent Go and not yet heard back about, and how to
/// read Go's answer.
///
/// Every edit goes out with a sequence number and the rewrite epoch this
/// field last adopted. Go's answer comes back as three props on the node:
/// `value`, `editSeq` (the last edit Go applied) and `editEpoch` (how many times
/// Go has rewritten the field). An epoch above ours means Go rewrote the
/// text; anything else is an echo of our own typing and is ignored.
///
/// On a rewrite, the keystrokes still in flight were typed on text Go has
/// replaced, and Go will drop them. They are not lost: `upstream` replays
/// them onto Go's text (`rebaseEdit`) and the field sends the result as a
/// new edit at the new epoch.
///
///     anchor   pending (seq, text)                      local text
///     "beta"   (5,"beta,") (6,"beta,g") (7,"beta,ga")   "beta,ga"
///     Go: value "" editSeq 5 editEpoch 1
///       basis = "beta,"  (the text as of seq 5)
///       rebase "beta,ga" from "beta," onto ""  →  "ga", sent as seq 8, epoch 1
///       Go drops 6 and 7 (epoch 0) and applies 8
///
/// A class, held in the field's @State, because it is bookkeeping and not
/// something to draw: mutating it must not invalidate the view. The Android
/// renderer has the same class under the same name (GrMobTextEdits.kt).
///
/// # Who uses it
///
/// GrMobTextField (Input, InputPassword, NumericInput, TextArea) and the two
/// editors' coordinators. The rich-text editor constructs it with
/// `replay: false`: its value is a JSON document, and splicing the text of one
/// JSON string onto another makes something that is not a document. It adopts
/// Go's rewrite as it stands, which is what its value queue did.
///
/// # When Go stamps nothing
///
/// Before this host has sent its first edit, Go stamps no field. `upstream`
/// then falls back to the value queue this class replaced: a value we sent is
/// an echo, and any other value is Go speaking for itself.
///
/// Stamped means the node carries editEpoch at all, and not that editSeq is
/// nonzero. Once this host is sequenced, Go stamps every field from its first
/// render, so a field Go has rewritten before its first edit arrives with
/// editSeq 0 and editEpoch 1, and that is a rewrite (core/text_edit.go,
/// "Every field, once the host is sequenced": comps.PINInput lost keys to it).
final class TextEditLedger {
    /// The rewrite epoch this field has adopted; sent with every edit.
    private(set) var epoch: Int

    /// The text the pending edits were typed on: the value as of the last
    /// edit Go applied, or of the last rewrite or focus, whichever came
    /// later.
    private var anchor: String

    /// Edits sent and not yet acknowledged, oldest first.
    private var pending: [(seq: Int, text: String)] = []

    /// Whether a rewrite replays the typing in flight (`rebaseEdit`), or is
    /// adopted as it stands. See "Who uses it".
    private let replay: Bool

    /// The text the last rewrite was rebased from: what Go had read before it
    /// rewrote. Kept for `rebaseCaret`, which needs the same basis
    /// `rebaseEdit` used to tell typing at the end from typing at the start.
    private(set) var lastBasis: String

    init(anchor: String = "", epoch: Int = 0, replay: Bool = true) {
        self.anchor = anchor
        self.epoch = epoch
        self.replay = replay
        self.lastBasis = anchor
    }

    /// Records an edit just sent under `seq`.
    func sent(_ seq: Int, _ text: String) {
        pending.append((seq, text))
    }

    /// Forgets everything in flight and takes Go's value and epoch as they
    /// stand: focus arriving, or a field that is Go's to draw.
    func reset(_ text: String, epoch: Int) {
        pending.removeAll()
        anchor = text
        self.epoch = epoch
    }

    /// Reads one change to the node's value, editSeq or editEpoch while the
    /// field is focused. Returns the text the field should now show, or nil
    /// to keep its own. When the returned text differs from `value`, the
    /// caller must send it as a new edit: it carries typing Go has not seen.
    func upstream(_ value: String, ack: Int, goEpoch: Int, local: String, stamped: Bool) -> String? {
        if !stamped { return legacy(value) }
        // Let go of every edit Go has applied. The last of them is the text Go
        // read before this render, which is the base of any rewrite.
        var basis = anchor
        while let first = pending.first, first.seq <= ack {
            basis = first.text
            pending.removeFirst()
        }
        anchor = basis
        guard goEpoch > epoch else { return nil }
        epoch = goEpoch
        pending.removeAll()
        anchor = value
        lastBasis = basis
        return replay ? rebaseEdit(basis: basis, local: local, rewrite: value) : value
    }

    /// The value queue, for a node with no stamps. Go may coalesce renders,
    /// so an echo drops the queue through its own entry and not just the
    /// head.
    private func legacy(_ value: String) -> String? {
        if let echo = pending.firstIndex(where: { $0.text == value }) {
            pending.removeSubrange(...echo)
            return nil
        }
        pending.removeAll()
        anchor = value
        return value
    }
}

/// Replays the typing from `basis` to `local` onto Go's `rewrite`.
///
/// Only insertions at either end are replayed. Those are what outruns a
/// round trip: a run of typed characters at the caret, which is almost
/// always at the end. Anything else (a deletion, an edit in the middle) has
/// no single place on the rewritten text it obviously belongs, and Go's text
/// wins, as it always did.
///
///     basis "beta,"   local "beta,gam"   rewrite ""       →  "gam"
///     basis "ab"      local "xab"        rewrite "AB"     →  "xAB"
///     basis "hello"   local "hell"       rewrite "Hello"  →  "Hello"
func rebaseEdit(basis: String, local: String, rewrite: String) -> String {
    if local == basis { return rewrite }
    if local.hasPrefix(basis) { return rewrite + String(local.dropFirst(basis.count)) }
    if local.hasSuffix(basis) { return String(local.dropLast(basis.count)) + rewrite }
    return rewrite
}

/// Where the caret goes after `rebaseEdit` replayed `local` onto `rewrite`,
/// for a host that owns its selection (the code editor). `caret` and the
/// result are UTF-16 offsets, which is what `UITextView.selectedRange` counts.
///
/// It follows the typing that was replayed, since that is where the user was:
///
///     typed at the end    keep the distance from the end
///         basis "beta,"  local "beta,ga|"  rebased "ga|"
///     typed at the start  keep the distance from the start
///         basis "ab"     local "x|ab"      rebased "x|AB"
///     Go's text won       the end of Go's text, which is where a rewrite
///                         always put the caret before there was a rebase
///
/// The cases are `rebaseEdit`'s, in its order, so the two cannot disagree
/// about which end the typing was at. The Android renderer has the same
/// function under the same name.
func rebaseCaret(basis: String, local: String, rewrite: String, caret: Int) -> Int {
    let rebased = rebaseEdit(basis: basis, local: local, rewrite: rewrite).utf16.count
    let at: Int
    if local == basis {
        at = rebased
    } else if local.hasPrefix(basis) {
        at = rebased - (local.utf16.count - caret)
    } else if local.hasSuffix(basis) {
        at = caret
    } else {
        at = rebased
    }
    return min(max(at, 0), rebased)
}
