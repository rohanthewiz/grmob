import XCTest

/// Lesson 4.8's footer: a scrollable ChipStrip holding "Start over", a
/// FlexGrow spacer and the row/fetch count. The strip is shorter than the
/// phone, so the spacer takes the free width and the count sits at the strip's
/// far edge, as it does in a browser and on Compose (GrMobGrowStrip).
///
/// Whether SwiftUI's horizontal ScrollView hands the spacer that width is what
/// is measured: its content is proposed an unbounded width, so a spacer that is
/// given nothing leaves the count right after the chip.
///
/// ```
///   the count at the far edge   count.minX - chip.maxX  >  half the window
///   the count after the chip    count.minX - chip.maxX  ≈  the strip's gap
/// ```
///
/// Screenshots go to GRMOB_SHOT_DIR as in TutorialNativeFloorsUITests.
final class TutorialFooterStripUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
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

    func testTheCountSitsAtTheFooterStripsFarEdge() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/4.8")!)
        XCTAssertTrue(app.staticTexts.matching(NSPredicate(format: "label BEGINSWITH %@", "4.8"))
            .firstMatch.waitForExistence(timeout: 10), "lesson 4.8 did not open")

        let count = app.staticTexts.matching(NSPredicate(format: "label CONTAINS %@", " rows, ")).firstMatch
        let chip = app.descendants(matching: .any).matching(NSPredicate(format: "label == %@", "Start over")).firstMatch
        var swipes = 0
        while !(count.exists && count.isHittable && chip.exists && chip.isHittable) && swipes < 40 {
            app.swipeUp(velocity: .slow)
            swipes += 1
        }
        XCTAssertTrue(count.isHittable && chip.isHittable, "the footer strip never came on screen")
        shot(app, "i_4.8_footer")

        let window = app.windows.firstMatch.frame.width
        let between = count.frame.minX - chip.frame.maxX
        print("GRMOB footer: window \(window) chip \(chip.frame) count \(count.frame) between \(between)")
        XCTAssertGreaterThan(between, window / 2 - count.frame.width,
                             "the count is \(between)pt after the chip: the FlexGrow spacer was given no free width")
    }
}
