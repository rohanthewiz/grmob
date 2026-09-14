import XCTest

// Simulator pass for the tutorial's lesson-to-lesson navigation: "Next ›"
// opens the next lesson at its top, and a tap inside a lesson keeps its
// scroll. Requires the tutorial in the framework, like TutorialScrollUITests:
//
//   ios/build.sh ./examples/tutorial
//   xcodebuild test ... -only-testing:GrMobUITests/TutorialNextUITests
//
// # What it holds
//
// core.Navigator stamps each frame's root node with a key naming its stack
// entry (withFrameKey), and GrMobRoot gives the root view `.id(root.viewID)`.
// The two halves of that are the two tests:
//
//   - a new frame is a new identity, so SwiftUI discards the old lesson's
//     ScrollView and its offset instead of carrying it into the next lesson
//     (on Android, before the key, 1.4 opened mid-page after "Next ›" from
//     the bottom of 1.3);
//   - within one frame the identity must stay put, or every state change
//     would rebuild the ScrollView and throw the reader back to the top.
final class TutorialNextUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    private func text(_ app: XCUIApplication, beginningWith prefix: String) -> XCUIElement {
        app.staticTexts.matching(NSPredicate(format: "label BEGINSWITH %@", prefix)).firstMatch
    }

    /// Swipes up slowly until the element is on screen and hittable.
    private func scroll(_ app: XCUIApplication, to element: XCUIElement, max: Int = 40) {
        var swipes = 0
        while !(element.exists && element.isHittable) && swipes < max {
            app.swipeUp(velocity: .slow)
            swipes += 1
        }
    }

    func testNextFromTheBottomOfALessonOpensTheNextAtItsTop() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/1.3")!)
        XCTAssertTrue(text(app, beginningWith: "1.3").waitForExistence(timeout: 10), "lesson 1.3 did not open")

        let next = app.buttons["Next ›"]
        scroll(app, to: next)
        XCTAssertTrue(next.isHittable, "lesson 1.3's Next button never came on screen")
        next.tap()

        // The title is the first thing on a lesson page. On screen and in the
        // top half means the page opened at its top, not at 1.3's offset.
        let title = text(app, beginningWith: "1.4")
        XCTAssertTrue(title.waitForExistence(timeout: 10), "lesson 1.4 did not open")
        XCTAssertTrue(title.isHittable, "1.4's title is off screen: the page kept 1.3's scroll offset")
        XCTAssertLessThan(title.frame.minY, app.frame.height / 2,
                          "1.4's title is not near the top of the screen")
    }

    func testATapInsideALessonKeepsItsScroll() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/1.4")!)
        XCTAssertTrue(text(app, beginningWith: "1.4").waitForExistence(timeout: 10), "lesson 1.4 did not open")

        // The FlexGrow check row sits below the demo, a few screens down. Any
        // element type: the row is a labelled control, not a static text.
        let check = app.descendants(matching: .any)
            .matching(NSPredicate(format: "label BEGINSWITH %@", "FlexGrow(1) on B")).firstMatch
        scroll(app, to: check)
        XCTAssertTrue(check.isHittable, "the FlexGrow check row never came on screen")
        let before = check.frame.minY

        // Toggling it is a state change inside the same frame: a render pass
        // and a patch, and no new frame key. The row must not move.
        check.tap()
        sleep(1)
        XCTAssertTrue(check.exists && check.isHittable,
                      "the check row left the screen after a tap: the lesson's scroll was reset")
        XCTAssertEqual(check.frame.minY, before, accuracy: 2,
                       "the check row moved after a tap inside the lesson")
    }
}
