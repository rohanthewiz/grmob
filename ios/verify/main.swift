// Data-layer conformance harness (see run.sh). Reads the transcript gen.go
// produced, replays it through the real runtime files — GrMobNode parsing,
// GrMobStyle decoding, TreeStore patch application — and deep-compares the
// resulting tree against the Go side's final full render. Runs as a plain
// macOS executable: the data layer is UI-free, so no Xcode or simulator is
// needed to prove the Swift store agrees with the Go reconciler.
import CoreGraphics
import Foundation

struct Transcript: Decodable {
    let initial: String
    let steps: [String]
    let final: String
    /// The picker-menu cases, which have nothing to do with the replay and
    /// ride along in the same file because this harness is one executable.
    /// See selectmenu.swift.
    let menuCases: [MenuCase]
    /// The band-inset cases, which have even less to do with the replay: they
    /// are pure geometry solved through GrMobFlexSolver. See band.swift.
    let bandCases: [BandCase]
    /// The pinned-Row cases: one overflowing Row with core.FlexShrink(0) in
    /// three positions, plus the control with none, each carrying the answer a
    /// Compose Row gives it. See pin.swift and internal/pinfixture.
    let pinCases: [PinCase]
}

/// One unweighted child of a pinned Row. `pinned` is core.FlexShrink(0),
/// which reaches this solver as a shrink factor of 0 and reaches Compose as
/// Modifier.pinMainAxis — one declaration, two spellings.
struct PinChild: Decodable {
    let name: String
    let base: CGFloat
    let pinned: Bool
}

/// What internal/pinfixture's transcription of foundation-layout's measure
/// policy produces for the Row. Not measured on this machine and not claimed
/// to be androidx's own code — see internal/pinfixture for the chain that
/// holds it up.
struct PinCompose: Decodable {
    let offered: [CGFloat]
    let mains: [CGFloat]
    let rowMain: CGFloat
}

/// One Row, both targets. See internal/pinfixture.
struct PinCase: Decodable {
    let what: String
    let offer: CGFloat
    let gap: CGFloat
    let children: [PinChild]
    let compose: PinCompose
    /// Whether the two targets land on the same extents for this Row. Asserted
    /// in both directions by checkPinnedRow, for the reason BandCase states
    /// sharesADeficit.
    let mainsAgreeWithCSS: Bool
}

/// One band arrangement: which node carries the chrome.
struct BandArrangement: Decodable {
    let what: String
    let row: BandInsets
    let gap: CGFloat
    let control: BandInsets
    let grow: CGFloat
    let align: String
}

struct BandInsets: Decodable {
    let top: CGFloat
    let right: CGFloat
    let bottom: CGFloat
    let left: CGFloat
}

struct BandSize: Decodable {
    let w: CGFloat
    let h: CGFloat
}

/// One band, both ways. See internal/bandfixture.
struct BandCase: Decodable {
    let what: String
    let now: BandArrangement
    let before: BandArrangement
    let label: BandSize
    let badge: BandSize
    let badgeInsets: BandInsets
    let offers: [Double]
    let sameHeight: Bool
    let sharesADeficit: Bool
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
    // checkFixedSizeContainer is part of the same sum because it is the same
    // solver: it is not a new rule, it is the fixed-size census's SwiftUI
    // main-axis row being measured here instead of reasoned about in a comment.
    let flexProblems = checkFlexSolver() + checkWrapSolver() + checkFixedSizeContainer()
    if flexProblems.isEmpty {
        print("OK: flex solver matches the CSS rules, including the fixed-size census")
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

    // The band insets, before the replay and for the same reason the picker
    // menus are: pure arithmetic over the transcript's own table, so a
    // divergence should be named on its own rather than buried under a tree
    // diff. This is the pass that asks *this* renderer whether moving a band's
    // padding from its Row onto the control inside it is the same band, which
    // was verified on the web by the pixels it did not move and assumed
    // everywhere else.
    let bandProblems = checkBandInsets(transcript.bandCases)
    if bandProblems.isEmpty {
        print("OK: \(transcript.bandCases.count) band arrangements place the same "
            + "pixels with the insets on the Row and on the control, and differ only "
            + "where they are recorded to (a shared deficit, a taller badge)")
    } else {
        print("FAIL: \(bandProblems.count) band inset difference(s)")
        for p in bandProblems { print("  " + p) }
        return 1
    }

    // The pinned Row, before the replay and for the reason the two above are:
    // pure arithmetic over the transcript's own table. This is the pass that
    // asks whether core.FlexShrink(0) means the same thing on this target as
    // on Compose, and where the answers part company.
    let pinProblems = checkPinnedRow(transcript.pinCases)
    if pinProblems.isEmpty {
        print("OK: \(transcript.pinCases.count) pinned Rows keep the pinned child at "
            + "its own size on both targets, and divide what is left between the "
            + "siblings the two different ways they are recorded to")
    } else {
        print("FAIL: \(pinProblems.count) pinned Row difference(s)")
        for p in pinProblems { print("  " + p) }
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
