import XCTest

/// Tutorial checks of two native-only mappings that no host-side harness can
/// reach, because both are about what UIKit's accessibility tree ends up
/// holding after SwiftUI lays the view out.
///
/// - core.CurrentDate: the today cell of lesson 4.9's calendar is named with
///   the platform's word for today after the day. The label, and not the
///   accessibility value: this test first asserted the value and read it
///   empty, because the cell stays an accessibility container (see
///   grMobCurrentLabel in GrMobStyle.swift).
/// - core.MinWidth: the rich-text link prompt (comps.RichTextEditor's 280pt
///   floor) is at least that wide on a phone, measured through its labelled
///   field, which stretches across it. On iOS the prompt is a sheet, so the
///   floor holds rather than binds.
///
/// Screenshots are attached to the result, and also written to
/// GRMOB_SHOT_DIR when the runner is given one
/// (`TEST_RUNNER_GRMOB_SHOT_DIR=/some/dir xcodebuild test …`), for the
/// by-eye half of a layout check.
final class TutorialNativeFloorsUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    private func text(_ app: XCUIApplication, beginningWith prefix: String) -> XCUIElement {
        app.staticTexts.matching(NSPredicate(format: "label BEGINSWITH %@", prefix)).firstMatch
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

    func testTodayIsSpokenAfterTheDayInItsCellsName() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/4.9")!)
        XCTAssertTrue(text(app, beginningWith: "4.9").waitForExistence(timeout: 10), "lesson 4.9 did not open")

        // The same question grMobTodayWord asks, in the same locale.
        let formatter = RelativeDateTimeFormatter()
        formatter.dateTimeStyle = .named
        formatter.locale = .autoupdatingCurrent
        let today = formatter.localizedString(from: DateComponents(day: 0))

        // Lesson 4.9 pins today to Wednesday 11 March 2026, and
        // comps.Calendar names a day "Monday, January 2, 2006".
        let cell = app.descendants(matching: .any)
            .matching(NSPredicate(format: "label == %@", "Wednesday, March 11, 2026, " + today)).firstMatch
        scroll(app, to: cell)
        XCTAssertTrue(cell.exists, "no element is named \"Wednesday, March 11, 2026, \(today)\": the today cell does not speak the platform's word")
        shot(app, "i_4.9")
    }

    func testTheLinkPromptKeepsItsMinimumWidth() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/4.14")!)
        XCTAssertTrue(text(app, beginningWith: "4.14").waitForExistence(timeout: 10), "lesson 4.14 did not open")

        let add = app.buttons["Add link"]
        scroll(app, to: add, max: 10)
        XCTAssertTrue(add.isHittable, "the editor's Add link button never came on screen")
        shot(app, "i_4.14")
        // The link action is disabled until the editor holds a caret, so the
        // document is focused first.
        let editor = app.textViews.firstMatch
        XCTAssertTrue(editor.exists, "the rich-text editor has no text view")
        editor.tap()
        XCTAssertTrue(add.isEnabled, "Add link stayed disabled with a caret in the editor")
        add.tap()
        sleep(1)
        shot(app, "i_4.14_link")

        // core.AccessibilityLabel on a core.Input names the field's box; the
        // UITextField inside carries only the placeholder.
        let field = app.otherElements["Link address"]
        XCTAssertTrue(field.waitForExistence(timeout: 5), "the link prompt did not open")
        // The prompt's Column is floored at 280pt and padded by the theme's
        // MD step on both sides; the field stretches across what is left, so
        // anything under 240pt means the floor did not bind.
        XCTAssertGreaterThanOrEqual(field.frame.width, 240,
                                    "the link prompt is narrower than its 280pt MinWidth allows")
    }
}
