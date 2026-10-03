import XCTest

// Simulator pass for the week chart in lesson 4.19, "Clocks, drawing and
// alarms" (examples/tutorial/chapter4.go): canvas text, a per-shape clip and
// per-shape taps (core.Shape.Text, Clip and OnClick). Requires the tutorial in
// the framework:
//
//   ios/build.sh ./examples/tutorial
//
// and runs alone, since GrMobUITests drives the mobileapp demo bound by the
// default build:
//
//   xcodebuild test ... -only-testing:GrMobUITests/TutorialCanvasTapUITests
//
// # What it holds
//
// The hit-test is Go's (core/canvas_hit.go) and has its own tests; what only a
// device can break is the half before it: that a tap on the canvas reaches
// the tap layer rather than the box's own onClick, and that the point it
// reports is in the canvas's box, in points. So the taps are aimed at viewBox
// points mapped the way core.CanvasMapping maps them, and the caption under
// the chart, which Go rewrites from the pick, says whether the right bar was
// hit:
//
//   Tue's bar               "Tue: 42"   a hit
//   under Tue's baseline    "Tap a bar" the pill's clipped-off end: a miss,
//                                       so the canvas's own OnClick clears
//   Thu's bar               "Thu: 36"
//   the empty top corner    "Tap a bar" a miss again
//
// With TEST_RUNNER_GRMOB_SHOTS_DIR set (see TutorialChartsUITests), each step
// writes a PNG, for checking the text and the clip by eye.
final class TutorialCanvasTapUITests: XCTestCase {

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

    func testABarTapIsHitTestedInGo() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/4.19")!)

        let chart = element(app, beginningWith: "Bar chart: Monday")
        XCTAssertTrue(chart.waitForExistence(timeout: 10), "lesson 4.19 has no week chart")
        // Until the caption under it is on screen too, so every tap below
        // lands on a chart that is wholly visible.
        let caption = element(app, beginningWith: "Tap a bar")
        var swipes = 0
        while !(caption.exists && caption.isHittable && chart.isHittable) && swipes < 20 {
            app.swipeUp(velocity: .slow)
            swipes += 1
        }
        XCTAssertTrue(chart.isHittable, "the week chart never came on screen")
        shot("canvas-tap-0")

        // A viewBox point of the 120 × 64 drawing to a point on screen, by
        // CanvasFit's rule: one scale, the slack split evenly.
        let frame = chart.frame
        let k = min(frame.width / 120, frame.height / 64)
        let ox = (frame.width - 120 * k) / 2, oy = (frame.height - 64 * k) / 2
        func tap(_ x: CGFloat, _ y: CGFloat) {
            chart.coordinate(withNormalizedOffset: .zero)
                .withOffset(CGVector(dx: ox + x * k, dy: oy + y * k))
                .tap()
        }
        func expectCaption(_ prefix: String, _ shotName: String) {
            let el = element(app, beginningWith: prefix)
            XCTAssertTrue(el.waitForExistence(timeout: 5), "caption never became \"\(prefix)…\"")
            shot(shotName)
        }

        tap(39, 30)
        expectCaption("Tue: 42", "canvas-tap-1-tue")
        tap(39, 54)
        expectCaption("Tap a bar", "canvas-tap-2-under")
        tap(85, 40)
        expectCaption("Thu: 36", "canvas-tap-3-thu")
        tap(114, 6)
        expectCaption("Tap a bar", "canvas-tap-4-clear")
    }
}
