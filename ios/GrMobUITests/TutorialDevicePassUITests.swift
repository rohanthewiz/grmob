import XCTest

/// Simulator pass for the widgets that had only been seen in headless Chrome:
/// D6's range band (4.25), D7's TimePicker menus (4.26), Tier E's small
/// pieces (4.27), Tier F's heat canvases (4.28), and round three's CopyButton,
/// Link, BulletList, AudioPlayer, MessageBubble, ExpandableText, half-star
/// Rating (4.29–4.32) and TagInput (5.8), and the theme accent on a
/// Toggle and a Slider (2.6, 6.8).
///
/// Mostly a driver for screenshots: what these widgets can get wrong on a
/// native is how they *draw* (a notch, a menu, a canvas star), which no
/// assertion here can see. Two assertions are geometric, because the Android
/// half of the same pass found both as measurements:
///
///   - today's calendar cell is as tall as the rest of its row. It carried a
///     border no other cell did, and a border adds to an unsized box's size
///     on every target (113px against 107px on the emulator);
///   - a Breadcrumb's plain-text current page is centred on the same line as
///     its button crumbs (a wrapping Row lost AlignItemsCenter on Compose).
///
/// Requires the tutorial in the framework (`ios/build.sh ./examples/tutorial`)
/// and runs alone:
///
///   TEST_RUNNER_GRMOB_SHOTS_DIR=/some/dir xcodebuild test ... \
///     -only-testing:GrMobUITests/TutorialDevicePassUITests
///
/// With that variable set, each step writes a PNG there; without it the test
/// only asserts. The TEST_RUNNER_ prefix is how xcodebuild hands a variable
/// to the runner (see TutorialChartsUITests).
final class TutorialDevicePassUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    // MARK: helpers

    private func any(_ app: XCUIApplication, beginningWith prefix: String) -> XCUIElement {
        app.descendants(matching: .any)
            .matching(NSPredicate(format: "label BEGINSWITH %@", prefix)).firstMatch
    }

    private func any(_ app: XCUIApplication, labelled label: String) -> XCUIElement {
        app.descendants(matching: .any).matching(NSPredicate(format: "label == %@", label)).firstMatch
    }

    private func button(_ app: XCUIApplication, _ label: String) -> XCUIElement {
        app.buttons.matching(NSPredicate(format: "label == %@", label)).firstMatch
    }

    /// Slow swipes until `target` is hittable. Slow, because a fast swipe is a
    /// fling and a pass driven by flings lands somewhere different each run.
    private func scroll(_ app: XCUIApplication, to target: XCUIElement, max: Int = 40) {
        var swipes = 0
        while !(target.exists && target.isHittable) && swipes < max {
            app.swipeUp(velocity: .slow)
            swipes += 1
        }
        settle()
    }

    /// A slow swipe still leaves momentum, and a tap on a scroll view that is
    /// still moving only stops it: the first run's text field "had no
    /// keyboard focus" after a tap that landed on it.
    private func settle() {
        sleep(1)
    }

    /// Scrolls the target up into the middle of the screen, so a screenshot
    /// taken next shows what sits under it too, not just its top edge.
    private func lift(_ app: XCUIApplication) {
        app.swipeUp(velocity: .slow)
        settle()
    }

    private func shot(_ name: String) {
        guard let dir = ProcessInfo.processInfo.environment["GRMOB_SHOTS_DIR"], !dir.isEmpty else { return }
        let png = XCUIScreen.main.screenshot().pngRepresentation
        try? png.write(to: URL(fileURLWithPath: dir).appendingPathComponent("\(name).png"))
    }

    /// The accessibility tree as XCUITest sees it, written beside the
    /// screenshots: the labels a query can match, and every element's frame.
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

    // MARK: 4.25 — the range band

    func testRangeBandAndTodayCell() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.25")

        // The demo's inline calendar pins today to Wednesday 11 March 2026,
        // and names its days "11 March 2026, …" through its own DayLabel.
        let today = any(app, beginningWith: "11 March 2026")
        let before = any(app, beginningWith: "10 March 2026")
        scroll(app, to: any(app, labelled: "Choose your nights"))
        lift(app)
        dump(app, "dp-4.25-inline")
        XCTAssertTrue(today.exists && before.exists, "the inline calendar's days are missing")
        shot("dp-4.25-inline")
        // A cell's accessibility frame is its numeral's (see
        // TutorialZeroBasisAndGradientsUITests), so the cell's extra height
        // shows as today's numeral sitting lower than its neighbour's.
        // The cell's own frame is no measure: today's ring gives its element
        // the cell's visible edge where the others report their numeral. The
        // numerals themselves are: a taller cell drops today's below its
        // neighbour's.
        let todayNumeral = today.staticTexts.firstMatch.frame
        let beforeNumeral = before.staticTexts.firstMatch.frame
        XCTAssertEqual(todayNumeral.minY, beforeNumeral.minY, accuracy: 0.5,
                       "today's numeral is off its row: \(todayNumeral) vs \(beforeNumeral)")

        // Open the sheet and pick 10 → 16: the band crosses a week boundary
        // and holds today inside it.
        let field = any(app, labelled: "Choose your nights")
        scroll(app, to: field)
        field.tap()
        let sheetStart = any(app, beginningWith: "Tuesday, March 10, 2026")
        dump(app, "dp-4.25-sheet")
        XCTAssertTrue(sheetStart.waitForExistence(timeout: 5))
        sheetStart.tap()
        shot("dp-4.25-pending")
        any(app, beginningWith: "Monday, March 16, 2026").tap()
        sleep(1)
        // Back on the page, the inline calendar names its days its own way.
        let end = any(app, beginningWith: "16 March 2026")
        scroll(app, to: end)
        shot("dp-4.25-band")
        dump(app, "dp-4.25-band")
    }

    // MARK: 4.26 — TimePicker's menus

    func testTimePickerMenus() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.26")
        let minute = button(app, "Minute")
        scroll(app, to: minute)
        lift(app)
        shot("dp-4.26-closed")
        minute.tap()
        shot("dp-4.26-minute-menu")
        let quarter = app.buttons["45"].firstMatch
        XCTAssertTrue(quarter.waitForExistence(timeout: 3), "the minute menu has no 45")
        quarter.tap()
        button(app, "Hour").tap()
        shot("dp-4.26-hour-menu")
        app.buttons["10"].firstMatch.tap()
        XCTAssertTrue(any(app, beginningWith: "Holding Wed 11 Mar 2026, 10:45").waitForExistence(timeout: 3),
                      "the picked hour and minute did not reach the readout")
        app.switches.firstMatch.tap()
        sleep(1)
        shot("dp-4.26-24h")
    }

    // MARK: 4.27 — seven small pieces

    func testSmallPieces() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.27")
        let current = any(app, labelled: "#40121")
        scroll(app, to: current)
        // "Orders" is also the bottom bar's first tab; the crumb is the one on
        // the current page's line or near it.
        let crumb = app.buttons.matching(NSPredicate(format: "label == %@", "Orders"))
            .allElementsBoundByIndex.min { abs($0.frame.midY - current.frame.midY) < abs($1.frame.midY - current.frame.midY) }!
        shot("dp-4.27-top")
        XCTAssertEqual(current.frame.midY, crumb.frame.midY, accuracy: 1,
                       "the current crumb is off its line: \(current.frame) vs \(crumb.frame)")

        let show = button(app, "Show password")
        scroll(app, to: show)
        let field = app.secureTextFields.firstMatch
        field.tap()
        field.typeText("hunter2")
        show.tap()
        XCTAssertTrue(app.textFields.matching(NSPredicate(format: "value == %@", "hunter2")).firstMatch
            .waitForExistence(timeout: 3), "Show did not reveal the password")
        shot("dp-4.27-revealed")

        button(app, "View receipt").tap()
        // The image is fetched from the network; give it time to arrive.
        sleep(4)
        shot("dp-4.27-lightbox")
        button(app, "Close").tap()

        let inbox = any(app, beginningWith: "Inbox")
        scroll(app, to: inbox)
        shot("dp-4.27-badge")
        inbox.tap()
        sleep(1)
        shot("dp-4.27-inbox")
    }

    // MARK: 4.28 — heat and spread

    func testHeatCanvases() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.28")
        for (prefix, name) in [("Workouts", "calendar"), ("Orders by hour", "heatmap"), ("Response time", "histogram")] {
            let chart = any(app, beginningWith: prefix)
            scroll(app, to: chart)
            lift(app)
            XCTAssertTrue(chart.exists, "\(prefix) chart missing")
            shot("dp-4.28-\(name)")
        }
    }

    // MARK: 4.29 — copy, link and list

    func testCopyLinkList() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.29")
        shot("dp-4.29-top")
        let copy = button(app, "Copy invite code")
        scroll(app, to: copy)
        copy.tap()
        shot("dp-4.29-copied")
        // The pasteboard is checked from the host (`xcrun simctl pbpaste
        // booted`), not here: a runner reading another app's pasteboard
        // raises iOS's paste-permission prompt, which would stall the pass.

        // Found as links, by name, which is what VoiceOver needs: comps.Link
        // is a labelled RoleLink Box around its own hidden Text, and until
        // GrMobNode.containerStyle such a box was no element at all on this
        // host (app.links.count was 0, and the tap below had nothing to find).
        let hint = any(app, labelled: "One link out of the app, one within it.")
        scroll(app, to: hint)
        lift(app)
        shot("dp-4.29-links")
        dump(app, "dp-4.29-links")
        XCTAssertTrue(app.links["GrMob on GitHub"].exists, "the URL link is missing from the accessibility tree")
        let follow = app.links["Follow an in-app link"]
        XCTAssertTrue(follow.exists, "the in-app link is missing from the accessibility tree")
        follow.tap()
        XCTAssertTrue(any(app, beginningWith: "In-app link followed 1 time.").waitForExistence(timeout: 3),
                      "the in-app link did not follow")
        let list = any(app, beginningWith: "Release steps")
        scroll(app, to: list)
        lift(app)
        shot("dp-4.29-list")
    }

    // MARK: 4.30 — the audio player

    func testAudioPlayer() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.30")
        let play = button(app, "Play")
        scroll(app, to: play)
        lift(app)
        shot("dp-4.30-idle")
        play.tap()
        // Streamed; the duration arrives once the first bytes do.
        XCTAssertTrue(button(app, "Pause").waitForExistence(timeout: 15), "Play did not become Pause")
        sleep(6)
        shot("dp-4.30-playing")
        button(app, "Forward 15 seconds").tap()
        sleep(1)
        shot("dp-4.30-forward")
        button(app, "Stop").tap()
    }

    // MARK: 4.31 — message bubbles

    func testMessageBubbles() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.31")
        let field = app.textFields.firstMatch
        scroll(app, to: field)
        shot("dp-4.31-thread")
        dump(app, "dp-4.31-thread")
        field.tap()
        field.typeText("Canvas stars now too")
        button(app, "Send").tap()
        XCTAssertTrue(any(app, beginningWith: "You, Canvas stars now too").waitForExistence(timeout: 3))
        shot("dp-4.31-sent")
        dump(app, "dp-4.31-sent")
    }

    // MARK: 4.32 — read more, and half a star

    func testReadMoreAndHalfStar() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.32")
        let more = button(app, "Read more")
        scroll(app, to: more)
        lift(app)
        shot("dp-4.32-capped")
        // The toggle keeps its name ("Read more"; aria-expanded says which
        // way it is), so the test reads the paragraph's height instead.
        let review = any(app, beginningWith: "Arrived a day early")
        let capped = review.frame.height
        more.tap()
        sleep(1)
        dump(app, "dp-4.32-open")
        XCTAssertGreaterThan(review.frame.height, capped + 20, "Read more did not open the paragraph")
        shot("dp-4.32-open")

        let minus = button(app, "− 0.5")
        scroll(app, to: minus)
        lift(app)
        shot("dp-4.32-stars-a")
        minus.tap()
        sleep(1)
        shot("dp-4.32-stars-b")
    }

    // MARK: 5.8 — tags

    func testTagInput() throws {
        let app = XCUIApplication()
        open(app, lesson: "5.8")
        let field = app.textFields.firstMatch
        scroll(app, to: field)
        shot("dp-5.8-start")
        dump(app, "dp-5.8-start")
        field.tap()
        sleep(1)
        // Typed through the app, into whatever holds focus. The field does
        // (the keyboard is up and it reports hasKeyboardFocus), but typeText
        // on the element re-resolves the query after each commit re-renders
        // the row above it, and XCUITest then reports no focus on the copy.
        XCTAssertEqual(app.keyboards.count, 1, "tapping the tag field raised no keyboard")
        app.typeText("alpha\n")
        app.typeText("beta,gamma,")
        XCTAssertTrue(button(app, "Remove gamma").waitForExistence(timeout: 3),
                      "a separator did not commit the draft")
        XCTAssertTrue(button(app, "Remove alpha").exists, "return did not commit the draft")
        shot("dp-5.8-pills")
    }

    /// Several tags in one burst, through the text-edit protocol
    /// (core/text_edit.go): every keystroke carries a sequence number and an
    /// epoch, and each comma's rewrite (the draft cleared) is read by epoch.
    ///
    /// This guards the new path; it does not reproduce the race the protocol
    /// was built for. typeText paces its keys and waits for the app between
    /// them, so no keystroke is ever in flight when a rewrite lands, and the
    /// old value queue passed this test too (twice, run for the purpose).
    /// The race was reproduced on the Android emulator with `adb shell input
    /// text`, where the old queue committed "gaba" and "ltad" from
    /// "alpha,beta,gamma,delta".
    ///
    /// Every tag must arrive whole, and nothing else may: the pills are
    /// counted, so a stray fragment fails as surely as a missing tag.
    func testTagInputAtMachineSpeed() throws {
        let app = XCUIApplication()
        open(app, lesson: "5.8")
        let field = app.textFields.firstMatch
        scroll(app, to: field)
        field.tap()
        sleep(1)
        XCTAssertEqual(app.keyboards.count, 1, "tapping the tag field raised no keyboard")
        app.typeText("one,two,three,four,")
        XCTAssertTrue(button(app, "Remove four").waitForExistence(timeout: 5),
                      "the last tag of the burst was not committed")
        for tag in ["one", "two", "three"] {
            XCTAssertTrue(button(app, "Remove \(tag)").exists, "\(tag) was not committed whole")
        }
        // The two the lesson starts with, and the four typed.
        let pills = app.buttons.matching(NSPredicate(format: "label BEGINSWITH 'Remove '")).count
        XCTAssertEqual(pills, 6, "the burst committed fragments beside its tags")
        shot("dp-5.8-burst")
        dump(app, "dp-5.8-burst")
    }

    // MARK: 2.6, 6.8 — the theme accent on platform controls

    /// A Toggle and a Slider in the theme's Primary, not the system green and
    /// blue. core.AccentColor reaches SwiftUI as `.tint`; no assertion can
    /// read a colour, so this drives the screenshots that show it.
    func testAccentOnPlatformControls() throws {
        let app = XCUIApplication()
        open(app, lesson: "2.6")
        let toggle = app.switches.firstMatch
        scroll(app, to: toggle)
        XCTAssertTrue(toggle.exists, "lesson 2.6 draws no switch")
        shot("dp-2.6-accent")

        app.open(URL(string: "grmob://lesson/6.8")!)
        XCTAssertTrue(any(app, beginningWith: "6.8").waitForExistence(timeout: 15), "lesson 6.8 did not open")
        let slider = app.sliders.firstMatch
        scroll(app, to: slider)
        XCTAssertTrue(slider.exists, "lesson 6.8 draws no slider")
        shot("dp-6.8-accent")
    }
}
