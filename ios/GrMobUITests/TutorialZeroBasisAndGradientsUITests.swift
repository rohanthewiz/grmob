import XCTest

/// Simulator pass for three things the host-side harnesses can build and
/// type-check but not see:
///
/// - lesson 4.9's calendar and lesson 4.7's StatTiles, the two tutorial
///   consumers of core.FlexBasis("0") besides the charts. Since
///   GrMobFlexZeroBasis, an iOS Row gives a zero-basis child its share of the
///   whole axis by weight, not of what is left after content widths. So the
///   seven day columns of a week must be equal to the point whatever their
///   numerals, and the two StatTiles equal whatever their values. The day
///   cells are measured here; the tiles have no element of their own, so
///   their equality is measured through their labels' left edges against
///   the screen, and the screenshots carry the rest.
/// - lesson 4.19's gradient rows: a radial and a linear FillGradient, and a
///   dashed StrokeGradient zigzag beside a radially stroked ring. Stroke
///   gradients go through a different SwiftUI path from fills (the stroked
///   outline is filled, so the width stays in points), and only the
///   screenshot tells whether the dashes kept their gaps and the colour
///   runs the right way.
///
/// Requires the tutorial in the framework (`ios/build.sh ./examples/tutorial`)
/// and runs alone:
///
///   TEST_RUNNER_GRMOB_SHOT_DIR=/some/dir xcodebuild test ... \
///     -only-testing:GrMobUITests/TutorialZeroBasisAndGradientsUITests
///
/// Screenshots are attached to the result, and also written to GRMOB_SHOT_DIR
/// when given.
final class TutorialZeroBasisAndGradientsUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    private func text(_ app: XCUIApplication, beginningWith prefix: String) -> XCUIElement {
        app.staticTexts.matching(NSPredicate(format: "label BEGINSWITH %@", prefix)).firstMatch
    }

    private func any(_ app: XCUIApplication, labelled label: String) -> XCUIElement {
        app.descendants(matching: .any).matching(NSPredicate(format: "label == %@", label)).firstMatch
    }

    private func scroll(_ app: XCUIApplication, to element: XCUIElement, max: Int = 40) {
        var swipes = 0
        while !(element.exists && element.isHittable) && swipes < max {
            app.swipeUp(velocity: .slow)
            swipes += 1
        }
    }

    private func shot(_ app: XCUIApplication, _ name: String) {
        let png = app.screenshot().pngRepresentation
        let attachment = XCTAttachment(data: png, uniformTypeIdentifier: "public.png")
        attachment.name = name
        attachment.lifetime = .keepAlways
        add(attachment)
        if let dir = ProcessInfo.processInfo.environment["GRMOB_SHOT_DIR"], !dir.isEmpty {
            try? png.write(to: URL(fileURLWithPath: dir).appendingPathComponent(name + ".png"))
        }
    }

    func testCalendarDayColumnsAreEqual() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/4.9")!)
        XCTAssertTrue(text(app, beginningWith: "4.9").waitForExistence(timeout: 10), "lesson 4.9 did not open")

        // Lesson 4.9 pins today to Wednesday 11 March 2026. The week of the
        // 8th to the 14th has one- and two-digit numerals side by side, which
        // is where a content-biased basis showed: "8" and "9" are narrower
        // than "10".
        let days = (8...14).map { day -> XCUIElement in
            let weekday = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"][day - 8]
            return app.descendants(matching: .any)
                .matching(NSPredicate(format: "label BEGINSWITH %@", "\(weekday), March \(day), 2026")).firstMatch
        }
        scroll(app, to: days[3])
        shot(app, "zb-4.9")
        for d in days { XCTAssertTrue(d.exists, "a day cell of the week of March 8 is missing: \(d.debugDescription)") }

        // A cell's accessibility frame is its numeral's, not its column's
        // (the first run of this test read 10.7pt for "8" and 18pt for
        // "10"), so widths say nothing. Centres do: each numeral is centred
        // in its column, so equal columns put the centres an equal stride
        // apart, and a content-biased split moves them by half the width
        // difference. Half a point is rounding at @3x.
        let centres = days.map(\.frame.midX)
        let stride = centres[1] - centres[0]
        for i in 1..<centres.count - 1 {
            XCTAssertEqual(centres[i + 1] - centres[i], stride, accuracy: 0.5,
                           "uneven column stride after day \(i + 8): centres \(centres)")
        }
    }

    func testStatTilesShareTheRow() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/4.7")!)
        XCTAssertTrue(text(app, beginningWith: "4.7").waitForExistence(timeout: 10), "lesson 4.7 did not open")

        let showing = text(app, beginningWith: "Showing")
        let filter = text(app, beginningWith: "Filter")
        scroll(app, to: filter)
        XCTAssertTrue(showing.exists && filter.exists, "the StatTile labels are missing")
        shot(app, "zb-4.7")

        // Two Fill tiles, Gap 16, in a row that spans the lesson's content
        // column. Each label sits at its tile's left padding, so the stride
        // from the first label to the second is one tile plus the gap; equal
        // tiles put the second label exactly that far on, and the row's left
        // edge is the first label less the same padding. With the content
        // column symmetric on the screen (its padding is equal on both
        // sides), rowWidth = screen - 2·rowLeft, and a tile is
        // (rowWidth - 16) / 2.
        let screen = app.windows.firstMatch.frame.width
        let stride = filter.frame.minX - showing.frame.minX
        // The tile's own padding is unknown here, so solve for it: with
        // p = tile padding and L = showing.minX, rowLeft = L - p and
        // stride = (screen - 2(L - p) - 16) / 2 + 16. One unknown, one
        // equation: it must come out as a small non-negative padding.
        let p = (2 * (stride - 16) + 16 - screen) / 2 + showing.frame.minX
        XCTAssertGreaterThanOrEqual(p, -0.5, "the second tile starts too early for equal tiles: stride \(stride), screen \(screen), first label at \(showing.frame.minX)")
        XCTAssertLessThanOrEqual(p, 24.5, "the second tile starts too late for equal tiles: stride \(stride), screen \(screen), first label at \(showing.frame.minX)")
    }

    func testGradientRowsDraw() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/4.19")!)
        XCTAssertTrue(text(app, beginningWith: "4.19").waitForExistence(timeout: 10), "lesson 4.19 did not open")

        let fills = any(app, labelled: "A shaded sphere beside a square fading blue to orange to green")
        let strokes = any(app, labelled: "A dashed zigzag fading blue to aqua beside a ring shading orange to red")
        scroll(app, to: strokes)
        XCTAssertTrue(fills.exists, "the fill gradient row is missing")
        XCTAssertTrue(strokes.exists, "the stroke gradient row is missing")
        // One more small swipe so both rows are clear of the bottom edge.
        app.swipeUp(velocity: .slow)
        shot(app, "zb-4.19")
    }
}
