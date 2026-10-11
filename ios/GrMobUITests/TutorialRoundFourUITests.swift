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
///   N-078  the grid, the range and the swatches are named containers whose
///          cells, sliders and swatches are elements of their own
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

        // The picker is a named container of swatches, not one element: the
        // group used to take the selected swatch's trait onto itself under
        // SwiftUI's combine (N-078).
        let swatches = app.otherElements.matching(NSPredicate(format: "label == 'Label colour'")).firstMatch
        XCTAssertTrue(swatches.exists, "the swatch group is not a container named Label colour")
        XCTAssertFalse(swatches.isSelected, "the group took a swatch's selection onto itself")
        XCTAssertTrue(app.buttons.matching(NSPredicate(format: "label == 'blue'")).firstMatch.isSelected,
                      "the blue swatch is not reported selected")


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
        // Found by name: the group is a container named "Price" and each
        // slider is its own element with its own label (N-078). Before, the
        // combine folded the two into one "Price" slider valued "10%, 40%".
        let group = app.otherElements.matching(NSPredicate(format: "label == 'Price'")).firstMatch
        let low = app.sliders.matching(NSPredicate(format: "label == 'Minimum'")).firstMatch
        let high = app.sliders.matching(NSPredicate(format: "label == 'Maximum'")).firstMatch
        scroll(app, to: high)
        // Back down if the swipes carried the pair off the top: a slider under
        // the status bar still reports itself hittable, and a drag there lands
        // on the page.
        let screen = app.windows.firstMatch.frame
        var back = 0
        while low.frame.minY < screen.minY + 120 && back < 8 {
            app.swipeDown(velocity: .slow)
            back += 1
            sleep(1)
        }
        print("r4-5.9: Minimum at \(low.frame), Maximum at \(high.frame), screen \(screen)")
        XCTAssertTrue(group.exists, "the range is not a container named Price")
        XCTAssertTrue(text(app, beginningWith: "$20 – $80").exists, "the range did not open on $20 – $80")
        dump(app, "r4-5.9-range-open")
        // Each slider's value is its row's readout, not a percentage of the
        // track (N-079: SliderRow states Format as the control's value).
        XCTAssertEqual(low.value as? String, "$20", "Minimum's value is not its readout")
        XCTAssertEqual(high.value as? String, "$80", "Maximum's value is not its readout")
        shot("r4-5.9-range-open")
        // A drag of the thumb from where it sits. Not
        // adjust(toNormalizedSliderPosition:), which finds the thumb by
        // reading the value as a percentage: this value is the readout
        // ("$20"), so it did nothing once Minimum was an element of its own.
        // UISlider insets its track by half a thumb, so the thumb's centre
        // is inset + fraction × (width − 2 × inset); a press that misses it
        // goes to the scroll view and moves the page instead.
        let inset = 14.0 / Double(low.frame.width)
        let thumb = inset + 0.1 * (1 - 2 * inset)
        low.coordinate(withNormalizedOffset: CGVector(dx: thumb, dy: 0.5))
            .press(forDuration: 0.3,
                   thenDragTo: low.coordinate(withNormalizedOffset: CGVector(dx: 0.7, dy: 0.5)),
                   withVelocity: .slow, thenHoldForDuration: 0.2)
        print("r4-5.9: after the drag Minimum is \(low.value as? String ?? "?")")
        sleep(1)
        shot("r4-5.9-range-pushed")
        dump(app, "r4-5.9-range-pushed")
        let lowValue = low.value as? String ?? ""
        let highValue = high.value as? String ?? ""
        XCTAssertTrue(lowValue == highValue && lowValue.hasPrefix("$"),
                      "the pushed Maximum (\(highValue)) did not follow the Minimum (\(lowValue))")
        XCTAssertFalse(text(app, beginningWith: "$20 – $80").exists, "the range line did not move")
    }

    // MARK: 4.37 — EditableGrid's EDIT round trip

    func testGridEditRoundTripAndDiscard() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.37")

        // The cells are reached by name. The grid is a container named
        // "Budget" and every cell is its own element (N-078); SwiftUI's
        // combine used to fold the whole grid into one static text, and this
        // test tapped coordinates off its frame.
        let grid = app.otherElements.matching(NSPredicate(format: "label == 'Budget'")).firstMatch
        let item2 = app.buttons.matching(NSPredicate(format: "label BEGINSWITH 'Item, row 2, '")).firstMatch
        scroll(app, to: item2)
        lift(app)
        shot("r4-4.37-start")
        dump(app, "r4-4.37-start")
        XCTAssertTrue(grid.exists, "the grid is not a container named Budget")
        XCTAssertEqual(item2.label, "Item, row 2, Groceries", "row 2's Item cell is not its own element")
        XCTAssertTrue(app.buttons.matching(NSPredicate(format: "label == 'Amount, row 2, $310.50'")).firstMatch.exists,
                      "row 2's Amount cell is not its own element")
        let undo = app.buttons.matching(NSPredicate(format: "label BEGINSWITH 'Undo'")).firstMatch
        XCTAssertEqual(undo.label, "Undo (0)", "the sheet did not open unedited")

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
        let edited = app.buttons.matching(NSPredicate(format: "label BEGINSWITH 'Item, row 2, '")).firstMatch
        XCTAssertEqual(edited.label, "Item, row 2, GroceriesXY", "the cell does not say its new value")
        // Recorded, not asserted: whether the keyboard stays up once no
        // native acts on the focus command for the row below.
        print("r4-4.37: keyboards after return: \(app.keyboards.count)")

        // The ✕: a tap on it must throw the typed draft away. If the tap blurs
        // the field first and the blur commits, the undo stack grows to two.
        edited.tap()
        sleep(1)
        XCTAssertEqual(app.keyboards.count, 1, "the second tap raised no keyboard")
        app.typeText("ZZ")
        sleep(1)
        shot("r4-4.37-draft")
        let discard = app.buttons.matching(NSPredicate(format: "label == 'Discard edit'")).firstMatch
        XCTAssertTrue(discard.exists, "the ✕ is not its own element")
        discard.tap()
        sleep(1)
        shot("r4-4.37-discarded")
        XCTAssertEqual(undo.label, "Undo (1)", "the ✕ committed the draft instead of throwing it away")
        XCTAssertEqual(app.buttons.matching(NSPredicate(format: "label BEGINSWITH 'Item, row 2, '")).firstMatch.label,
                       "Item, row 2, GroceriesXY", "the discarded draft reached the cell")
    }

    /// The last grid row's editor, opened near the bottom of the screen, must
    /// sit above the software keyboard (N-102). The editor is a UIKit field
    /// that SwiftUI's ScrollView does not scroll into view, and before
    /// GrMobKeyboardReveal the keyboard covered it whole (the editor at y 756
    /// under a keyboard whose top was at 583).
    ///
    /// Only meaningful with the soft keyboard on screen. With the simulator's
    /// hardware keyboard connected, the default, the keyboard element sits
    /// below the screen and there is nothing to cover the field. The test then
    /// records the skip rather than passing for a reason it did not check. To
    /// run it for real: `defaults write com.apple.iphonesimulator
    /// ConnectHardwareKeyboard -bool false`, then relaunch Simulator.app.
    func testTheLastGridRowsEditorIsAboveTheKeyboard() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.37")
        let rows = app.buttons.matching(NSPredicate(format: "label BEGINSWITH 'Item, row '"))
        let screen = app.windows.firstMatch.frame
        // Small drags, so row 4 stops just inside the bottom edge, where the
        // keyboard will rise over it; a slow swipe overshoots to mid-screen.
        var drags = 0
        while drags < 80 {
            let all = rows.allElementsBoundByIndex
            if all.count >= 4, let last = all.last, last.isHittable, last.frame.maxY < screen.maxY - 40 { break }
            app.coordinate(withNormalizedOffset: CGVector(dx: 0.9, dy: 0.75))
                .press(forDuration: 0.05, thenDragTo: app.coordinate(withNormalizedOffset: CGVector(dx: 0.9, dy: 0.68)))
            drags += 1
        }
        let last = rows.allElementsBoundByIndex.last!
        XCTAssertEqual(last.label, "Item, row 4, Dinner out", "the fourth row is not the last")
        last.tap()
        sleep(2)
        // The keyboard's one-time "slide to type" sheet, on a fresh simulator.
        let onboarding = app.buttons["Continue"]
        if onboarding.exists { onboarding.tap(); sleep(2) }
        shot("r4-4.37-last-row-editing")
        let keyboard = app.keyboards.firstMatch.frame
        guard keyboard.minY < screen.maxY else {
            throw XCTSkip("the soft keyboard is not on screen (a hardware keyboard is connected); nothing to cover the field")
        }
        let field = app.textFields.firstMatch
        XCTAssertTrue(field.isHittable, "the last row's editor is not hittable: \(field.frame) under a keyboard at \(keyboard)")
        XCTAssertLessThanOrEqual(field.frame.maxY, keyboard.minY,
                                 "the last row's editor ends at \(field.frame.maxY), under the keyboard's top at \(keyboard.minY)")
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
