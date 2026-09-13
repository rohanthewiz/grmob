import XCTest

// Simulator pass for the tutorial (examples/tutorial): a List with no Height
// inside a scrolled lesson page. Requires the tutorial in the framework:
//
//   ios/build.sh ./examples/tutorial
//
// and runs alone, since GrMobUITests drives the mobileapp demo bound by the
// default build:
//
//   xcodebuild test ... -only-testing:GrMobUITests/TutorialScrollUITests
//
// # What it holds
//
// GrMobList (Renderer.swift) gives a List with no Height no special arm on
// this host, where Compose needs one: the page's ScrollView proposes the
// List's own ScrollView no height, and a ScrollView answers that with its
// content's height. That was measured once on a simulator — lesson 4.3's
// outline came out 228pt with all six rows inside the frame — and nothing
// re-checked it. The two ways it could stop being true are the two this test
// reads frames for:
//
//   - the List answers the unbounded proposal with less than its content, so
//     its last rows are clipped or drawn over what follows it;
//   - the rows lay out on top of one another (a zero-height stack).
//
// The lesson is reached by deep link (grmob://lesson/4.3), so the test does
// not depend on where 4.3 sits in the contents page.
final class TutorialScrollUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    /// The outline demo's rows, in order (examples/tutorial/chapter4.go,
    /// canonOutline).
    private let rows = ["New Testament", "Gospels", "Matthew", "Mark", "Letters", "Romans"]

    private func text(_ app: XCUIApplication, beginningWith prefix: String) -> XCUIElement {
        app.staticTexts.matching(NSPredicate(format: "label BEGINSWITH %@", prefix)).firstMatch
    }

    func testListWithoutHeightInAScrolledPageKeepsItsRowsInsideIt() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/4.3")!)

        // The demo panel's caption, directly above the List, and the first
        // key point, the first thing after the panel.
        let caption = text(app, beginningWith: "Six rows, one flat list")
        XCTAssertTrue(caption.waitForExistence(timeout: 10), "lesson 4.3 did not open")
        let after = text(app, beginningWith: "Slots are core.View fields")

        // Scroll until the last row is on screen. A slow swipe moves less than
        // the demo's height, so the first row is still materialised when the
        // last one arrives.
        let last = app.staticTexts[rows.last!]
        var swipes = 0
        while !(last.exists && last.isHittable) && swipes < 20 {
            app.swipeUp(velocity: .slow)
            swipes += 1
        }
        XCTAssertTrue(last.isHittable, "the outline's last row never came on screen")

        let frames = rows.map { title -> CGRect in
            let row = app.staticTexts[title]
            XCTAssertTrue(row.exists, "row \(title) is missing")
            return row.frame
        }

        // Rows stack: each starts at or below where the previous one ends. Half
        // a point of slack for rounding between the layout and the snapshot.
        for i in 1..<frames.count {
            XCTAssertGreaterThanOrEqual(frames[i].minY, frames[i - 1].maxY - 0.5,
                                        "\(rows[i]) overlaps \(rows[i - 1])")
        }
        // The List sits between the caption and what follows the panel: its
        // first row below the caption, its last row above the key point. A List
        // that reported less than its content would put the key point over its
        // last rows.
        XCTAssertGreaterThanOrEqual(frames[0].minY, caption.frame.maxY - 0.5,
                                    "the first row is drawn over the demo's caption")
        XCTAssertTrue(after.exists, "the first key point is missing")
        XCTAssertGreaterThanOrEqual(after.frame.minY, frames.last!.maxY - 0.5,
                                    "the key point after the demo is drawn over the List's last row")
    }
}
