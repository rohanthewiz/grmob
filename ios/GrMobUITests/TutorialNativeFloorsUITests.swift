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
///   The same cell is one accessibility element with no Button inside it:
///   the tap's button trait used to be applied inside the label's combine and
///   reached the numeral, which made the cell a container of buttons (see
///   GrMobGestureAccessibility in GrMobStyle.swift).
/// - core.MinWidth: the rich-text link prompt (comps.RichTextEditor's 280pt
///   floor) is at least that wide on a phone, measured through its labelled
///   field, which stretches across it. On iOS the prompt is a sheet, so the
///   floor holds rather than binds.
/// - core.MinWidth("40%"): lesson 1.4's box A, floored at 40% of its row. A
///   Row resolves that floor itself (GrMobFlexSolver.percentFloors): before
///   it did, A kept its content width on this simulator.
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
        // Before GrMobGestureAccessibility this read two Buttons: "11" and an
        // unlabelled 0×0 one.
        XCTAssertEqual(cell.descendants(matching: .button).count, 0,
                       "the today cell holds buttons of its own, so VoiceOver can stop inside it: \(cell.debugDescription)")
        shot(app, "i_4.9")
    }

    /// Lesson 1.4's demo row, before and after its MinWidth("40%") toggle.
    ///
    /// The boxes are unlabelled, but each demoBox's label reports its whole
    /// box's frame to XCUITest, not the text's (measured: B's label started at
    /// 95.3pt, the left edge of B's blue-green box in the screenshot, not 18pt
    /// of padding inside it). So the labels are the boxes here:
    ///
    /// ```
    ///   A's width        a.frame.width                 at least 40% of the row
    ///   the row after    b.frame.minX == a.frame.maxX + 8 (the row's gap)
    /// ```
    ///
    /// The 40% is of the row's content width, which is the window less the
    /// page, panel and row insets on each side (32, 14 and 8pt). Loosely,
    /// because those insets are the theme's: the check is that the floor
    /// binds, not that the theme kept its numbers. That A's letter sits in the
    /// middle of the wider box is not in any frame, so it is left to the
    /// i_1.4_floor screenshot.
    func testAPercentageMinWidthFloorsTheBox() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/1.4")!)
        XCTAssertTrue(text(app, beginningWith: "1.4").waitForExistence(timeout: 10), "lesson 1.4 did not open")

        let toggle = app.descendants(matching: .any)
            .matching(NSPredicate(format: "label BEGINSWITH %@", "MinWidth(")).firstMatch
        scroll(app, to: toggle)
        XCTAssertTrue(toggle.isHittable, "the MinWidth toggle never came on screen")
        let a = app.staticTexts["A"]
        let b = app.staticTexts["B"]
        XCTAssertTrue(a.exists && b.exists, "the demo row's A and B labels are missing")
        let widthBefore = a.frame.width
        shot(app, "i_1.4")

        toggle.tap()
        sleep(1)
        shot(app, "i_1.4_floor")
        let rowWidth = app.windows.firstMatch.frame.width - 2 * (32 + 14 + 8)
        XCTAssertGreaterThan(a.frame.width, widthBefore + 40, "A did not widen: its 40% floor did not bind")
        XCTAssertGreaterThanOrEqual(a.frame.width, rowWidth * 0.4 - 12,
                                    "A is \(a.frame.width)pt wide, under 40% of a row of about \(rowWidth)pt")
        XCTAssertEqual(b.frame.minX, a.frame.maxX + 8, accuracy: 1,
                       "B does not start one gap after the floored A: the row placed A's floor without making room for it")
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

        // core.AccessibilityLabel on a core.Input names the text field itself.
        // It used to name a SwiftUI element wrapped round the field, with the
        // field inside carrying only its placeholder, and this query looked
        // for that wrapper; see GrMobTextField.boxStyle.
        let field = app.textFields["Link address"]
        XCTAssertTrue(field.waitForExistence(timeout: 5), "the link prompt did not open")
        // The prompt's Column is floored at 280pt and padded by the theme's
        // MD step on both sides; the field stretches across what is left, so
        // anything under 240pt means the floor did not bind.
        XCTAssertGreaterThanOrEqual(field.frame.width, 240,
                                    "the link prompt is narrower than its 280pt MinWidth allows")
    }
}
