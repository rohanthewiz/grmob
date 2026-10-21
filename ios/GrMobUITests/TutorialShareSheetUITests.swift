import XCTest

/// core.Share on iOS (N-114): lesson 4.41's Gallery has a ShareURL on each
/// card and no OnShare, so its Share button calls core.Share, and the iOS
/// host presents a UIActivityViewController over the screen.
///
/// What is measured is that the system sheet comes up. Before core.Share the
/// same tap put the link on the clipboard and showed a toast, so a sheet on
/// screen is the whole difference. The sheet is UIKit's own view hierarchy;
/// XCUITest exposes it as an "ActivityListView" (or, on newer releases, a
/// sheet holding a "Copy" action), so either is accepted.
///
/// Screenshots go to GRMOB_SHOT_DIR as in TutorialNativeFloorsUITests.
final class TutorialShareSheetUITests: XCTestCase {

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

    func testGallerySharePresentsTheSystemShareSheet() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/4.41")!)
        XCTAssertTrue(app.staticTexts.matching(NSPredicate(format: "label BEGINSWITH %@", "4.41"))
            .firstMatch.waitForExistence(timeout: 10), "lesson 4.41 did not open")

        let share = app.buttons.matching(NSPredicate(format: "label == %@", "Share")).firstMatch
        var swipes = 0
        while !(share.exists && share.isHittable) && swipes < 20 {
            app.swipeUp(velocity: .slow)
            swipes += 1
        }
        XCTAssertTrue(share.isHittable, "Gallery's Share button never came on screen")
        share.tap()

        // Springboard-free: the sheet is presented inside the app's own
        // window, so it is in the app's tree.
        let list = app.otherElements["ActivityListView"]
        let copy = app.buttons["Copy"]
        let deadline = Date().addingTimeInterval(10)
        while !(list.exists || copy.exists) && Date() < deadline {
            RunLoop.current.run(until: Date().addingTimeInterval(0.25))
        }
        shot(app, "i_4.41_share_sheet")
        XCTAssertTrue(list.exists || copy.exists, "no share sheet came up after Share")
    }
}
