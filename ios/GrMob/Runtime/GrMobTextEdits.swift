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
    /// rewrote. Kept for `rebaseCaret`, which must merge from the same basis
    /// `rebaseEdit` did, or the two would disagree about where the typing was.
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
/// A three-way merge of two single-span edits. internal/rebasefixture is the
/// statement of the rule (read its package doc for the reasoning), and
/// ios/verify runs this copy against its case table; android/verify runs the
/// Kotlin copy against the same one.
///
/// Each of basis→local (the user's typing) and basis→rewrite (Go's change) is
/// read as one contiguous span. The user's span is carried across Go's change
/// and spliced into the rewrite:
///
///     basis "HELLOaWORLD"  local "HELLOabWORLD"  rewrite "HELLOAWORLD"
///     user's span [6,6) → "b"     Go's span [5,6) → "A"
///     6 is at the end of Go's span, so it maps to 6   →   "HELLOAbWORLD"
///
/// Where the user's span sits inside text Go replaced with text of another
/// length, there is no telling where it belongs, and Go's text wins, as it
/// always did.
///
///     basis "beta,"   local "beta,gam"   rewrite ""       →  "gam"
///     basis "ab"      local "xab"        rewrite "AB"     →  "xAB"
///     basis "hello"   local "hell"       rewrite "Hello"  →  "Hell"
///     basis "beta,"   local "bet"        rewrite ""       →  ""   (Go wins)
///
/// The arithmetic is in UTF-16 units, as the Kotlin copy's is (a Kotlin String
/// is UTF-16) and as `rebaseCaret`'s answer must be (UITextView counts them).
/// A Swift Character comparison would disagree with Kotlin about where two
/// strings stop sharing a prefix whenever a combining mark is involved.
///
/// The first version replayed only an insertion at either end of `basis`, and
/// two keys typed quickly mid-text under an UPPERCASE transform lost the
/// second.
func rebaseEdit(basis: String, local: String, rewrite: String) -> String {
    guard let m = rebaseMerge(Array(basis.utf16), Array(local.utf16), Array(rewrite.utf16)) else {
        return rewrite
    }
    return String(decoding: m.text, as: UTF16.self)
}

/// Where the caret goes after `rebaseEdit`, for a host that owns its
/// selection (the code editor). `caret` and the result are UTF-16 offsets,
/// which is what `UITextView.selectedRange` counts.
///
/// It follows the typing, since that is where the user was:
///
///     before the user's span   mapped through Go's change like any basis offset
///     inside the replayed text keeps its place in it
///     after the user's span    keeps its place in the text after
///     Go's text won, or the    the end of Go's text, which is where a rewrite
///     caret sat in text Go     always put the caret before there was a rebase
///     replaced
///
/// The Android renderer has the same function under the same name.
func rebaseCaret(basis: String, local: String, rewrite: String, caret: Int) -> Int {
    let b = Array(basis.utf16), l = Array(local.utf16), r = Array(rewrite.utf16)
    guard let m = rebaseMerge(b, l, r) else { return r.count }
    var at: Int
    if caret <= m.userStart {
        at = rebaseMapOffset(caret, b.count, m.goStart, m.goEnd, r.count)
    } else if caret < m.userStart + m.replayed.count {
        at = m.spliceStart + (caret - m.userStart)
    } else {
        let mapped = rebaseMapOffset(caret - l.count + b.count, b.count, m.goStart, m.goEnd, r.count)
        at = mapped < 0 ? -1 : mapped - m.spliceEnd + m.spliceStart + m.replayed.count
    }
    if at < 0 || at > m.text.count || rebaseSplitsPair(m.text, at) { at = m.text.count }
    return at
}

/// Where a focused plain field's caret goes when new text is written into it:
/// `before` is what the field showed, `after` what it shows now, `caret` the
/// caret in `before`, all in UTF-16 units. internal/rebasefixture's `Carry` is
/// the statement of the rule, the Android renderer has the same function
/// under the same name, and ios/verify runs this copy against `CarryCases`.
///
/// `rebaseCaret` is not the answer for a plain field: it follows typing that
/// was in flight, and with none it sends the caret to the end, which breaks
/// typing mid-text under an UPPERCASE onChange.
func carryCaret(before: String, after: String, caret: Int) -> Int {
    carryPlan(Array(before.utf16), Array(after.utf16), caret: caret).caret
}

/// The one replacement GrMobTextInput's `write` makes, and where it leaves the
/// caret: `a[start..<end]` is replaced by `b[start..<(end + b.count - a.count)]`.
///
/// The caret is `Carry`'s:
///
///     caret at or after the span's end    shifted by the change in length
///     caret at or before its start        left where it is
///     caret inside a span that kept its   left where it is (UPPERCASE)
///     length
///     caret inside a span that changed    the end of the new span
///     length
///     a result that would split a         the end of the text
///     surrogate pair
///
/// The span is the differing one, *extended to the caret* when the caret is at
/// or after it. That is `write`'s business rather than the rule's, and it is
/// here so the two are computed from one span: UITextInput's `replace` leaves
/// the caret at the end of the replacement, and a correction made afterwards
/// is too late, because the keyboard inserts a key it was holding at the
/// caret the replacement left (the "HELL o" log in `write`). Ending the span
/// at the caret makes the replacement leave the caret where `Carry` says,
/// with no move after it.
///
/// It lived inline in `write` until it was pulled out to be run against the
/// table; the arms were already `Carry`'s, and the surrogate guard is the one
/// thing added.
struct CarryPlan: Equatable {
    let start: Int
    let end: Int
    let caret: Int
}

func carryPlan(_ a: [UInt16], _ b: [UInt16], caret was: Int) -> CarryPlan {
    let (prefix, suffix) = rebaseCommonSpan(a, b)
    let spanEnd = a.count - suffix
    var caret = rebaseMapOffset(was, a.count, prefix, spanEnd, b.count)
    // Inside a span that changed length: the end of the new span, the nearest
    // place still after the text the caret was after.
    if caret < 0 { caret = b.count - suffix }
    if caret > b.count || rebaseSplitsPair(b, caret) { caret = b.count }
    // Extended to the caret when the caret follows the change, so `replace`
    // leaves it in place by itself. Both ends then sit in text the two share,
    // the end counted from the back, so it is `end + delta` in `b`.
    return CarryPlan(start: prefix, end: max(spanEnd, min(was, a.count)), caret: caret)
}

/// One successful merge and the offsets `rebaseCaret` needs from it. The
/// fields are internal/rebasefixture's `merged`, under the same names.
private struct RebaseMerge {
    var text: [UInt16]
    /// Where the user's span starts, in basis and local alike.
    var userStart: Int
    /// The user's text: local's side of the user's span.
    var replayed: [UInt16]
    /// Go's span in basis.
    var goStart: Int
    var goEnd: Int
    /// Where the user's span landed in the rewrite.
    var spliceStart: Int
    var spliceEnd: Int
}

/// `rebaseEdit`'s arithmetic; nil where Go's text wins.
private func rebaseMerge(_ b: [UInt16], _ l: [UInt16], _ r: [UInt16]) -> RebaseMerge? {
    // Nothing typed since the edit Go read: nothing to replay.
    if b == l { return nil }
    let (up, us) = rebaseCommonSpan(b, l)
    let (gp, gs) = rebaseCommonSpan(b, r)
    let goEnd = b.count - gs
    let spliceStart = rebaseMapOffset(up, b.count, gp, goEnd, r.count)
    let spliceEnd = rebaseMapOffset(b.count - us, b.count, gp, goEnd, r.count)
    // A splice point between the halves of a surrogate pair is no telling
    // either: the length-kept arm maps one unit at a time.
    if spliceStart < 0 || spliceEnd < 0 || spliceStart > spliceEnd
        || rebaseSplitsPair(r, spliceStart) || rebaseSplitsPair(r, spliceEnd) {
        return nil
    }
    let replayed = Array(l[up..<(l.count - us)])
    return RebaseMerge(
        text: Array(r[..<spliceStart]) + replayed + Array(r[spliceEnd...]),
        userStart: up, replayed: replayed, goStart: gp, goEnd: goEnd,
        spliceStart: spliceStart, spliceEnd: spliceEnd)
}

/// Carries basis offset `i` across Go's change of basis[goStart, goEnd), which
/// made a `basisLen`-unit basis into a `rewriteLen`-unit rewrite; -1 where
/// there is no telling. "After Go's span" is tested first: where both
/// inserted at one point, the user's text goes after Go's (comps.PINInput's
/// "14").
private func rebaseMapOffset(_ i: Int, _ basisLen: Int, _ goStart: Int, _ goEnd: Int,
                             _ rewriteLen: Int) -> Int {
    if i >= goEnd { return i + rewriteLen - basisLen }
    if i <= goStart { return i }
    // Inside a change that kept the length: a character-for-character
    // transform such as UPPERCASE, where offset i is still offset i.
    if rewriteLen == basisLen { return i }
    return -1
}

/// The lengths of `a` and `b`'s common prefix and common suffix, the suffix
/// limited so the two never overlap, and neither ending inside a surrogate
/// pair.
private func rebaseCommonSpan(_ a: [UInt16], _ b: [UInt16]) -> (Int, Int) {
    let n = min(a.count, b.count)
    var prefix = 0
    while prefix < n && a[prefix] == b[prefix] { prefix += 1 }
    if prefix > 0 && UTF16.isLeadSurrogate(a[prefix - 1]) { prefix -= 1 }
    var suffix = 0
    let limit = n - prefix
    while suffix < limit && a[a.count - 1 - suffix] == b[b.count - 1 - suffix] { suffix += 1 }
    if suffix > 0 && UTF16.isTrailSurrogate(a[a.count - suffix]) { suffix -= 1 }
    return (prefix, suffix)
}

/// Whether offset `i` of `u` falls between the two halves of a surrogate pair.
private func rebaseSplitsPair(_ u: [UInt16], _ i: Int) -> Bool {
    i > 0 && i < u.count && UTF16.isLeadSurrogate(u[i - 1]) && UTF16.isTrailSurrogate(u[i])
}
