// The text-edit rebase on this target, against internal/rebasefixture.
//
// When a rewrite from Go reaches a focused field, the keys typed since the
// edit Go read are replayed onto Go's text (core/text_edit.go, "The
// protocol"). The rule runs on the host alone, so no Go test executes it:
// GrMobTextEdits.swift and GrMobTextEdits.kt each carry a copy, and this pass
// and android/verify's run both copies against the one table, whose answers
// come from the Go reference.
//
// GrMobTextEdits.swift imports only Foundation, which is what lets it be
// compiled into this macOS executable at all.

/// One rewrite arriving at a focused field; see rebasefixture.Case.
struct RebaseCase: Decodable {
    let name: String
    let basis: String
    let local: String
    let rewrite: String
    let caret: Int
    let want: String
    let wantCaret: Int
}

func checkRebase(_ cases: [RebaseCase]) -> [String] {
    var problems: [String] = []
    if cases.isEmpty { problems.append("the transcript carried no rebase cases") }
    for c in cases {
        let got = rebaseEdit(basis: c.basis, local: c.local, rewrite: c.rewrite)
        if got != c.want {
            problems.append("\(c.name): rebaseEdit gave \(got.debugDescription), Go \(c.want.debugDescription)")
        }
        let caret = rebaseCaret(basis: c.basis, local: c.local, rewrite: c.rewrite, caret: c.caret)
        if caret != c.wantCaret {
            problems.append("\(c.name): rebaseCaret gave \(caret), Go \(c.wantCaret)")
        }
    }
    return problems
}

/// One write into a focused plain field; see rebasefixture.CarryCase.
struct CarryCase: Decodable {
    let name: String
    let before: String
    let after: String
    let caret: Int
    let want: Int
}

/// carryPlan against Carry's table: the caret, and that the span the field's
/// write would replace ends where that caret is whenever the caret follows
/// the change. The second half is the reason the span is extended (a key the
/// keyboard was holding lands at the caret `replace` leaves), and the one
/// part of write's arithmetic Carry itself does not state.
func checkCarry(_ cases: [CarryCase]) -> [String] {
    var problems: [String] = []
    if cases.isEmpty { problems.append("the transcript carried no carry cases") }
    for c in cases {
        let got = carryCaret(before: c.before, after: c.after, caret: c.caret)
        if got != c.want {
            problems.append("\(c.name): carryCaret gave \(got), Go \(c.want)")
        }
        let a = Array(c.before.utf16), b = Array(c.after.utf16)
        let plan = carryPlan(a, b, caret: c.caret)
        let delta = b.count - a.count
        if plan.start > plan.end || plan.end > a.count || plan.end + delta < plan.start {
            problems.append("\(c.name): carryPlan's span \(plan.start)..<\(plan.end) is not a span of the text")
            continue
        }
        // The replacement reproduces `after`.
        let spliced = Array(a[..<plan.start]) + Array(b[plan.start..<(plan.end + delta)]) + Array(a[plan.end...])
        if spliced != b {
            problems.append("\(c.name): replacing \(plan.start)..<\(plan.end) does not make the new text")
        }
        // The differing span's end, found here by plain comparison rather
        // than by the function under test.
        let n = min(a.count, b.count)
        var prefix = 0
        while prefix < n && a[prefix] == b[prefix] { prefix += 1 }
        var suffix = 0
        while suffix < n - prefix && a[a.count - 1 - suffix] == b[b.count - 1 - suffix] { suffix += 1 }
        if c.caret >= a.count - suffix && plan.end + delta != plan.caret {
            problems.append("\(c.name): the caret follows the change, but the replacement ends at "
                + "\(plan.end + delta), not at the caret \(plan.caret), so a key held by the "
                + "keyboard would land in the wrong place")
        }
    }
    return problems
}

