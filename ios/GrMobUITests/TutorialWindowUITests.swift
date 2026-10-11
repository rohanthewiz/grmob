import XCTest

/// The window record on iOS, under rotation (N-028).
///
/// AppWindowReader (App/AppWindow.swift) reports the root GeometryReader's
/// size and safe-area insets to Go as the "window" host event, and
/// hooks.UseWindow re-renders on a change. Its scheme report had been seen
/// live (N-050, N-085), but nothing had watched the size and insets move. A
/// rotation is the change a simulator can make without a person: the size
/// swaps, and the insets move from the top and bottom (the Dynamic Island,
/// the home indicator) to the sides.
///
/// Lesson 4.21 prints both: "Window" as "W × H" and "Insets" as "top T,
/// bottom B, left L, right R", physical edges as core.SafeInsets states
/// them. Split View and Stage Manager stay unseen; they need an iPad window
/// that a script cannot resize.
final class TutorialWindowUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
        XCUIDevice.shared.orientation = .portrait
    }

    /// Rotation persists across tests and apps on the simulator, so every
    /// exit puts the device back upright.
    override func tearDownWithError() throws {
        XCUIDevice.shared.orientation = .portrait
    }

    private struct Record: Equatable {
        var width = 0, height = 0
        var top = 0, bottom = 0, left = 0, right = 0
    }

    /// The two readouts, parsed. Matched by shape rather than position: the
    /// size line is the one label of the form "N × N", the insets line the
    /// one beginning "top ".
    private func record(_ app: XCUIApplication) -> Record? {
        let size = app.staticTexts.matching(NSPredicate(format: "label MATCHES %@", "^[0-9]+ × [0-9]+$")).firstMatch
        let insets = app.staticTexts.matching(NSPredicate(format: "label BEGINSWITH 'top '")).firstMatch
        guard size.exists, insets.exists else { return nil }
        let wh = size.label.components(separatedBy: " × ").compactMap { Int($0) }
        let edges = insets.label.components(separatedBy: CharacterSet.decimalDigits.inverted).compactMap { Int($0) }
        guard wh.count == 2, edges.count == 4 else { return nil }
        return Record(width: wh[0], height: wh[1], top: edges[0], bottom: edges[1], left: edges[2], right: edges[3])
    }

    /// Polls until the readout reports something other than `before`: the
    /// report crosses to Go and back, so the label changes a beat after the
    /// rotation settles.
    private func record(_ app: XCUIApplication, changedFrom before: Record?) -> Record? {
        for _ in 0..<20 {
            if let now = record(app), now != before { return now }
            usleep(250_000)
        }
        return record(app)
    }

    private func shot(_ name: String) {
        guard let dir = ProcessInfo.processInfo.environment["GRMOB_SHOTS_DIR"], !dir.isEmpty else { return }
        let png = XCUIScreen.main.screenshot().pngRepresentation
        try? png.write(to: URL(fileURLWithPath: dir).appendingPathComponent("\(name).png"))
    }

    func testTheWindowRecordFollowsARotation() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/4.21")!)
        XCTAssertTrue(app.staticTexts.matching(NSPredicate(format: "label BEGINSWITH '4.21'")).firstMatch
                        .waitForExistence(timeout: 15), "lesson 4.21 did not open")

        // The size and the insets reach Go in separate reports when the app
        // launches (the probe's reading lands a turn after the size), so the
        // portrait record is the first one whose top inset has arrived.
        var portrait = record(app, changedFrom: nil)
        for _ in 0..<20 where (portrait?.top ?? 0) == 0 {
            usleep(250_000)
            portrait = record(app)
        }
        guard let portrait else {
            return XCTFail("4.21 shows no window size and insets")
        }
        shot("w-4.21-portrait")
        XCTAssertLessThan(portrait.width, portrait.height, "portrait is not taller than wide: \(portrait)")
        XCTAssertGreaterThan(portrait.top, 0, "portrait reports no top inset (the status bar and island): \(portrait)")
        XCTAssertEqual(portrait.left, 0, "portrait reports a left inset: \(portrait)")
        XCTAssertEqual(portrait.right, 0, "portrait reports a right inset: \(portrait)")

        XCUIDevice.shared.orientation = .landscapeLeft
        guard let landscape = record(app, changedFrom: portrait) else {
            return XCTFail("the readouts vanished after the rotation")
        }
        shot("w-4.21-landscape")
        print("TutorialWindowUITests: portrait \(portrait), landscape \(landscape)")
        XCTAssertEqual(landscape.width, portrait.height, "the width did not take the old height: \(portrait) → \(landscape)")
        XCTAssertEqual(landscape.height, portrait.width, "the height did not take the old width: \(portrait) → \(landscape)")
        XCTAssertGreaterThan(landscape.left + landscape.right, 0,
                             "landscape reports no side insets, where the island now is: \(landscape)")
        XCTAssertLessThan(landscape.top, portrait.top, "the top inset did not shrink in landscape: \(portrait) → \(landscape)")

        XCUIDevice.shared.orientation = .portrait
        let back = record(app, changedFrom: landscape)
        XCTAssertEqual(back, portrait, "rotating back did not restore the portrait record")
    }
}
