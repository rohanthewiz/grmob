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
    guard CommandLine.arguments.count == 2,
          let data = FileManager.default.contents(atPath: CommandLine.arguments[1]),
          let transcript = try? JSONDecoder().decode(Transcript.self, from: data)
    else {
        FileHandle.standardError.write(Data("usage: harness <transcript.json>\n".utf8))
        return 2
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
