// Data-layer conformance harness (see run.sh). Reads the transcript gen.go
// produced, replays it through the real runtime files — GrMobNode parsing,
// GrMobStyle decoding, TreeStore patch application — and deep-compares the
// resulting tree against the Go side's final full render. Runs as a plain
// macOS executable: the data layer is UI-free, so no Xcode or simulator is
// needed to prove the Swift store agrees with the Go reconciler.
import Foundation

struct Transcript: Decodable {
    let initial: String
    let steps: [String]
    let final: String
    /// The picker-menu cases, which have nothing to do with the replay and
    /// ride along in the same file because this harness is one executable.
    /// See selectmenu.swift.
    let menuCases: [MenuCase]
}

/// Structural equality, reported as per-path differences so a failure names
/// the exact node that diverged instead of dumping two whole trees.
@MainActor
func diff(_ got: GrMobNode?, _ want: GrMobNode?, path: String, into problems: inout [String]) {
    guard let got, let want else {
        if (got == nil) != (want == nil) {
            problems.append("\(path): one side is missing (got \(got == nil ? "nil" : "node"), want \(want == nil ? "nil" : "node"))")
        }
        return
    }
    if got.type != want.type { problems.append("\(path): type \(got.type) != \(want.type)") }
    if got.key != want.key { problems.append("\(path): key \(got.key) != \(want.key)") }
    // NSDictionary equality gives deep, type-bridged comparison of the
    // JSONSerialization-produced prop values.
    if !(got.props as NSDictionary).isEqual(to: want.props) {
        problems.append("\(path): props \(got.props) != \(want.props)")
    }
    if got.style != want.style {
        problems.append("\(path): style \(String(describing: got.style)) != \(String(describing: want.style))")
    }
    if got.children.count != want.children.count {
        problems.append("\(path): child count \(got.children.count) != \(want.children.count)")
        return
    }
    for (i, pair) in zip(got.children, want.children).enumerated() {
        diff(pair.0, pair.1, path: "\(path)/\(i)", into: &problems)
    }
}

@MainActor
func run() -> Int32 {
    // The flex arithmetic is independent of the transcript, so it is checked
    // first: a layout regression should be reported even if the bridge
    // transcript cannot be read at all.
    let flexProblems = checkFlexSolver() + checkWrapSolver()
    if flexProblems.isEmpty {
        print("OK: flex solver matches the CSS rules")
    } else {
        print("FAIL: \(flexProblems.count) flex solver difference(s)")
        for p in flexProblems { print("  " + p) }
        return 1
    }

    // The overlay arithmetic, on the same footing and for the same reason.
    // This is the pass that pins what a ZStack sizes to, which was the one
    // documented cross-target divergence with nothing measuring it — plus the
    // three proposal decisions the Layout itself makes, which reach the solver
    // through GrMobStackLayer and are run here against a recording fake — and
    // the adapter between SwiftUI's size vocabulary and the solver's, which
    // was three unexecuted expressions in Renderer.swift until it moved into
    // GrMobStackBridge.swift.
    let stackProblems = checkStackSolver()
    if stackProblems.isEmpty {
        print("OK: stack solver sizes, places every anchor, proposes as the Layout "
            + "does, and converts both ways")
    } else {
        print("FAIL: \(stackProblems.count) stack solver difference(s)")
        for p in stackProblems { print("  " + p) }
        return 1
    }

    guard CommandLine.arguments.count == 2,
          let data = FileManager.default.contents(atPath: CommandLine.arguments[1]),
          let transcript = try? JSONDecoder().decode(Transcript.self, from: data)
    else {
        FileHandle.standardError.write(Data("usage: harness <transcript.json>\n".utf8))
        return 2
    }

    // The picker menu, before the replay: it is a pure function of the
    // transcript's own table, so a decomposition regression should be named
    // on its own rather than buried under whatever the tree diff says next.
    let menuProblems = checkSelectMenu(transcript.menuCases)
    if menuProblems.isEmpty {
        print("OK: \(transcript.menuCases.count) picker menus match Go's decomposition")
    } else {
        print("FAIL: \(menuProblems.count) picker menu difference(s)")
        for p in menuProblems { print("  " + p) }
        return 1
    }

    let store = TreeStore()
    store.mount(transcript.initial)
    guard store.root != nil else {
        print("FAIL: initial payload did not mount")
        return 1
    }
    for step in transcript.steps {
        store.applyPatches(step)
    }

    guard let finalData = transcript.final.data(using: .utf8),
          let finalObj = (try? JSONSerialization.jsonObject(with: finalData)) as? [String: Any]
    else {
        print("FAIL: final tree is not valid JSON")
        return 1
    }
    let want = GrMobNode.parse(finalObj)

    var problems: [String] = []
    diff(store.root, want, path: "root", into: &problems)
    if problems.isEmpty {
        print("OK: \(transcript.steps.count) patch batches applied; tree matches Go's final render")
        return 0
    }
    print("FAIL: \(problems.count) difference(s) after replay")
    for p in problems { print("  " + p) }
    return 1
}

exit(MainActor.assumeIsolated { run() })
