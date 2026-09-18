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
