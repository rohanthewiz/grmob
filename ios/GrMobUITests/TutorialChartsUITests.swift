import XCTest

// Simulator pass for lesson 4.20, "Charts" (examples/tutorial/chapter4.go,
// lessonCharts). Requires the tutorial in the framework:
//
//   ios/build.sh ./examples/tutorial
//
// and runs alone, since GrMobUITests drives the mobileapp demo bound by the
// default build:
//
//   xcodebuild test ... -only-testing:GrMobUITests/TutorialChartsUITests
//
// # What it holds
//
// Each chart is one accessibility element (RoleImg) whose label is a summary
// sentence built in Go, and everything drawn is a core.Canvas whose shapes
// change by update-props patches. So the two things a SwiftUI regression could
// break without any Go test noticing are:
//
//   - the element itself: the canvas and axis Text hide under the chart, and
//     the chart node keeps the label (a merge gone wrong reads the tick labels
//     instead, or nothing);
//   - the redraw: GrMobCanvas reads `node.children.map(\.props)`, and a patch
//     that SwiftUI's observation misses leaves the old arc on screen with the
//     new label announced over it. The label is checked here; the arc is what
//     the screenshots are for.
//
// # Screenshots
//
// With TEST_RUNNER_GRMOB_SHOTS_DIR in xcodebuild's environment (the
// TEST_RUNNER_ prefix is how xcodebuild hands a variable to the runner; as a
// `NAME=value` argument it becomes a build setting and never arrives):
//
//   TEST_RUNNER_GRMOB_SHOTS_DIR=/tmp/shots xcodebuild test ...
//
// each step writes a PNG there, for checking the drawing by eye:
// labels on gridlines, round-capped dots whole at the canvas's edges, the
// area tint. Without it the test only asserts.
final class TutorialChartsUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    private func element(_ app: XCUIApplication, beginningWith prefix: String) -> XCUIElement {
        app.descendants(matching: .any)
            .matching(NSPredicate(format: "label BEGINSWITH %@", prefix)).firstMatch
    }

    private func shot(_ name: String) {
        guard let dir = ProcessInfo.processInfo.environment["GRMOB_SHOTS_DIR"], !dir.isEmpty else { return }
        let png = XCUIScreen.main.screenshot().pngRepresentation
        try? png.write(to: URL(fileURLWithPath: dir).appendingPathComponent("\(name).png"))
    }

    /// Swipes until `target` is hittable, taking a screenshot after each swipe.
    private func scroll(_ app: XCUIApplication, to target: XCUIElement, shotPrefix: String) {
        var swipes = 0
        while !(target.exists && target.isHittable) && swipes < 12 {
            app.swipeUp(velocity: .slow)
            swipes += 1
            shot("\(shotPrefix)-\(swipes)")
        }
    }

    func testChartsAreOneElementEachAndRedrawOnUpdate() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/4.20")!)

        // The line chart's summary opens with its Subject.
        let line = element(app, beginningWith: "Traffic: ")
        XCTAssertTrue(line.waitForExistence(timeout: 10), "lesson 4.20 did not open")
        shot("charts-0")
        let before = line.label

        // Shift rotates the data, so the summary must change.
        let shift = app.buttons["Shift"]
        XCTAssertTrue(shift.waitForExistence(timeout: 5))
        shift.tap()
        let changed = NSPredicate(format: "label != %@", before)
        expectation(for: changed, evaluatedWith: line)
        waitForExpectations(timeout: 5)
        shot("charts-shifted")

        // The bar chart and donut are single elements with their summaries.
        let bars = element(app, beginningWith: "Steps this week: ")
        scroll(app, to: bars, shotPrefix: "charts-bars")
        XCTAssertTrue(bars.exists, "the bar chart's summary is missing")

        // The gauge: "Battery, 72%" until "Use 12%" is tapped.
        let use = app.buttons["Use 12%"]
        scroll(app, to: use, shotPrefix: "charts-gauge")
        XCTAssertTrue(use.isHittable, "the gauge's button never came on screen")
        let gauge = element(app, beginningWith: "Battery, ")
        XCTAssertTrue(gauge.exists, "the gauge is not one labelled element")
        XCTAssertEqual(gauge.label, "Battery, 72%")
        XCTAssertTrue(element(app, beginningWith: "Budget: ").exists, "the donut's summary is missing")
        use.tap()
        expectation(for: NSPredicate(format: "label == %@", "Battery, 60%"), evaluatedWith: gauge)
        waitForExpectations(timeout: 5)
        shot("charts-gauge-used")
    }
}
