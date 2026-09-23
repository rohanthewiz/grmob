import XCTest

/// Simulator pass for round four's widgets, which had been seen in headless
/// Chrome and on the Android emulator only: 5.9's inputs (ColorSwatchPicker,
/// RangeSlider), 4.37's EditableGrid, 4.34's ReactionBar, 4.36's charts, and
/// 4.35's TreeView under a right-to-left language.
///
/// Where the claim is a value, it is asserted. Where it is a drawing (a
/// swatch's ring and check, a round-capped Waveform bar, a mirrored tree
/// indent), the test is a screenshot driver and a person, or the session that
/// ran it, looks at the PNGs.
///
/// The claims, and the next-list items they answer:
///
///   N-066  a hex typed in its short form commits on return and not before;
///          a Minimum dragged past the Maximum carries it, and the pushed
///          native slider follows Go's value
///   N-072  a tap on a cell opens the field with the keyboard up, return
///          commits and ends EDIT; the ✕ throws a typed draft away rather
///          than the blur committing it first
///   N-062  a chip the reader tapped reports itself selected
///   N-070  the four demos drawn by SwiftUI (screenshots)
///   N-068  TreeView's indent under Arabic (screenshot)
///
/// Requires the tutorial in the framework (`ios/build.sh ./examples/tutorial`)
/// and runs alone:
///
///   TEST_RUNNER_GRMOB_SHOTS_DIR=/some/dir xcodebuild test ... \
///     -only-testing:GrMobUITests/TutorialRoundFourUITests
final class TutorialRoundFourUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    // MARK: helpers (TutorialDevicePassUITests' shapes, which are private there)

    private func any(_ app: XCUIApplication, beginningWith prefix: String) -> XCUIElement {
        app.descendants(matching: .any)
            .matching(NSPredicate(format: "label BEGINSWITH %@", prefix)).firstMatch
    }

    private func text(_ app: XCUIApplication, beginningWith prefix: String) -> XCUIElement {
        app.staticTexts.matching(NSPredicate(format: "label BEGINSWITH %@", prefix)).firstMatch
    }

    /// Slow swipes until `target` is hittable: a fast swipe is a fling, and a
    /// pass driven by flings lands somewhere different each run.
    private func scroll(_ app: XCUIApplication, to target: XCUIElement, max: Int = 40) {
        var swipes = 0
        while !(target.exists && target.isHittable) && swipes < max {
            app.swipeUp(velocity: .slow)
            swipes += 1
        }
        sleep(1)
    }

    /// One more slow swipe, so the target sits mid-screen and a tap on it is
    /// not at the bottom edge, where a field raised no keyboard.
    private func lift(_ app: XCUIApplication) {
        app.swipeUp(velocity: .slow)
        sleep(1)
    }

    private func shot(_ name: String) {
        guard let dir = ProcessInfo.processInfo.environment["GRMOB_SHOTS_DIR"], !dir.isEmpty else { return }
        let png = XCUIScreen.main.screenshot().pngRepresentation
        try? png.write(to: URL(fileURLWithPath: dir).appendingPathComponent("\(name).png"))
    }

    private func dump(_ app: XCUIApplication, _ name: String) {
        guard let dir = ProcessInfo.processInfo.environment["GRMOB_SHOTS_DIR"], !dir.isEmpty else { return }
        try? app.debugDescription.write(to: URL(fileURLWithPath: dir).appendingPathComponent("\(name).txt"),
                                        atomically: true, encoding: .utf8)
    }

    private func open(_ app: XCUIApplication, lesson: String) {
        app.launch()
        app.open(URL(string: "grmob://lesson/\(lesson)")!)
        XCTAssertTrue(any(app, beginningWith: lesson).waitForExistence(timeout: 15), "lesson \(lesson) did not open")
    }

    // MARK: 5.9 — ColorSwatchPicker and RangeSlider

    func testSwatchesHexShortFormAndTheRangeThatCannotCross() throws {
        let app = XCUIApplication()
        open(app, lesson: "5.9")

        let hex = app.textFields.matching(NSPredicate(format: "label == 'Custom colour, hex'")).firstMatch
        scroll(app, to: hex)
        lift(app)
        dump(app, "r4-5.9-swatches")
        shot("r4-5.9-swatches")
        XCTAssertTrue(text(app, beginningWith: "Value = #2A78D6").exists, "the demo did not open on its blue")

        // Every six-digit colour passes through a valid three-digit one, so
        // "#2a7" must not commit while it is being typed.
        hex.tap()
        sleep(1)
        if app.keyboards.count == 0 { hex.tap(); sleep(1) }
        app.typeText("#2a7")
        sleep(1)
        XCTAssertTrue(text(app, beginningWith: "Value = #2A78D6").exists,
                      "a short hex committed while it was typed")
        app.typeText("\n")
        sleep(1)
        shot("r4-5.9-hex-short")
        XCTAssertTrue(text(app, beginningWith: "Value = #22AA77").exists,
                      "the short form did not commit on return as #22AA77")

        // Minimum past Maximum. The demo opens on $20 – $80 of 0…200, so
        // 0.6 of the track is $120: the Maximum is carried, and the pushed
        // native slider has to follow Go's value, not keep its own.
        // Found by position inside the group, not by name: SwiftUI's combine
        // over the labelled group folds the two into one "Price" slider
        // valued "10%, 40%", and the inner two are left with no label of
        // their own (N-078). The queries are what XCUITest can still reach.
        let group = app.sliders.matching(NSPredicate(format: "label == 'Price'")).firstMatch
        let low = group.sliders.element(boundBy: 0)
        let high = group.sliders.element(boundBy: 1)
        scroll(app, to: group)
        lift(app)
        XCTAssertTrue(text(app, beginningWith: "$20 – $80").exists, "the range did not open on $20 – $80")
        dump(app, "r4-5.9-range-open")
        // Each slider's value is its row's readout, not a percentage of the
        // track (N-079: SliderRow states Format as the control's value). Read
        // off the combined group, the one element VoiceOver has here: the
        // inner two XCUITest synthesizes under the combine still report
        // UIKit's percentage, and VoiceOver does not visit them (N-078).
        XCTAssertEqual(group.value as? String, "$20, $80", "the sliders' values are not their readouts")
        low.adjust(toNormalizedSliderPosition: 0.6)
        sleep(1)
        shot("r4-5.9-range-pushed")
        dump(app, "r4-5.9-range-pushed")
        let lowValue = low.value as? String ?? ""
        let highValue = high.value as? String ?? ""
        XCTAssertEqual(lowValue, highValue, "the pushed Maximum (\(highValue)) did not follow the Minimum (\(lowValue))")
        let pair = (group.value as? String ?? "").components(separatedBy: ", ")
        XCTAssertTrue(pair.count == 2 && pair[0] == pair[1] && pair[0].hasPrefix("$"),
                      "the group's value (\(pair)) is not two equal readouts")
        XCTAssertFalse(text(app, beginningWith: "$20 – $80").exists, "the range line did not move")
    }

    // MARK: 4.37 — EditableGrid's EDIT round trip

    func testGridEditRoundTripAndDiscard() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.37")

        // The cells are reached by coordinate inside the grid, not by name:
        // SwiftUI's combine over the labelled grid ("Budget") takes every
        // text cell out of the accessibility tree, leaving only the header
        // texts and the Category menus (N-078). The grid element itself is
        // still there, and its frame is the grid's.
        let grid = app.staticTexts.matching(NSPredicate(format: "label == 'Budget'")).firstMatch
        scroll(app, to: grid)
        lift(app)
        shot("r4-4.37-start")
        dump(app, "r4-4.37-start")
        let undo = app.buttons.matching(NSPredicate(format: "label BEGINSWITH 'Undo'")).firstMatch
        XCTAssertEqual(undo.label, "Undo (0)", "the sheet did not open unedited")

        // Row 2's Item cell: the header row is 24pt and each body row about
        // 34pt, and Item starts 46pt in, after the row header.
        let frame = grid.frame
        let item2 = app.coordinate(withNormalizedOffset: .zero)
            .withOffset(CGVector(dx: frame.minX + 100, dy: frame.minY + 24 + 34 + 17))

        item2.tap()
        sleep(1)
        XCTAssertEqual(app.keyboards.count, 1, "a tap on the cell raised no keyboard (a Focus on an Input made this pass)")
        shot("r4-4.37-editing")
        app.typeText("XY")
        app.typeText("\n")
        sleep(1)
        shot("r4-4.37-committed")
        dump(app, "r4-4.37-committed")
        XCTAssertEqual(undo.label, "Undo (1)", "return did not commit the draft")
        // Recorded, not asserted: whether the keyboard stays up once no
        // native acts on the focus command for the row below.
        print("r4-4.37: keyboards after return: \(app.keyboards.count)")

        // The ✕: a tap on it must throw the typed draft away. If the tap blurs
        // the field first and the blur commits, the undo stack grows to two.
        item2.tap()
        sleep(1)
        XCTAssertEqual(app.keyboards.count, 1, "the second tap raised no keyboard")
        app.typeText("ZZ")
        sleep(1)
        shot("r4-4.37-draft")
        let discard = app.descendants(matching: .any).matching(NSPredicate(format: "label == 'Discard edit'")).firstMatch
        if discard.exists {
            discard.tap()
        } else {
            // Inside the combined grid too: the ✕ sits at the Item cell's
            // trailing edge (a 145pt column from 46pt in).
            app.coordinate(withNormalizedOffset: .zero)
                .withOffset(CGVector(dx: frame.minX + 46 + 145 - 14, dy: frame.minY + 24 + 34 + 17)).tap()
        }
        sleep(1)
        shot("r4-4.37-discarded")
        XCTAssertEqual(undo.label, "Undo (1)", "the ✕ committed the draft instead of throwing it away")
    }

    // MARK: 4.34 — a ReactionBar chip the reader tapped

    func testReactionChipReportsItsSelection() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.34")
        let chip = app.buttons.matching(NSPredicate(format: "label BEGINSWITH 'party popper'")).firstMatch
        scroll(app, to: chip)
        lift(app)
        XCTAssertFalse(chip.isSelected, "party popper opened selected")
        chip.tap()
        sleep(1)
        let after = app.buttons.matching(NSPredicate(format: "label BEGINSWITH 'party popper'")).firstMatch
        shot("r4-4.34-reacted")
        dump(app, "r4-4.34-reacted")
        XCTAssertEqual(after.label, "party popper, 2 reactions", "the count did not follow the tap")
        XCTAssertTrue(after.isSelected, "the chip the reader tapped is not reported selected")
    }

    // MARK: 4.36 — the charts, drawn by SwiftUI

    func testRoundFourChartsDraw() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.36")
        for (target, name) in [("Green rises", "candles"), ("Show rates", "funnel"),
                               ("Player:", "radar"), ("Forward", "waveform")] {
            let el = any(app, beginningWith: target)
            scroll(app, to: el)
            // The radar is tall enough that lift's swipe carries it off the
            // top, and its centring is what the shot is for (N-070).
            if name != "radar" { lift(app) }
            shot("r4-4.36-\(name)")
            dump(app, "r4-4.36-\(name)")
        }
        dump(app, "r4-4.36-end")
    }

    // MARK: 4.35 — TreeView under a right-to-left language

    func testTreeViewUnderArabic() throws {
        let app = XCUIApplication()
        // The language alone does not flip an app that ships no Arabic
        // localization (the first run stayed left-to-right); the two
        // writing-direction defaults force the layout direction itself.
        app.launchArguments += ["-AppleLanguages", "(ar)", "-AppleLocale", "ar",
                                "-AppleTextDirection", "YES", "-NSForceRightToLeftWritingDirection", "YES"]
        open(app, lesson: "4.35")
        let docs = any(app, beginningWith: "docs")
        scroll(app, to: docs)
        lift(app)
        shot("r4-4.35-rtl")
        dump(app, "r4-4.35-rtl")
    }
}
