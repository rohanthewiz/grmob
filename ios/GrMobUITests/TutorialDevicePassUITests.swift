import XCTest

/// Simulator pass for the widgets that had only been seen in headless Chrome:
/// D6's range band (4.25), D7's TimePicker menus (4.26), Tier E's small
/// pieces (4.27), Tier F's heat canvases (4.28), and round three's CopyButton,
/// Link, BulletList, AudioPlayer, MessageBubble, ExpandableText, half-star
/// Rating (4.29–4.32) and TagInput (5.8), round two's FAB, QRCode and
/// Countdown/Stopwatch (4.22–4.24), the theme accent on a
/// Toggle and a Slider (2.6, 6.8), and 2.3's TextArea.
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

    // MARK: 4.13, 4.14 — the editors on the text-edit protocol

    /// The rich-text editor's echo path, and the heading's own bold.
    ///
    /// Typing at the end of 4.14's heading goes through the text-edit
    /// protocol (core/text_edit.go): the host's JSON of the document is not
    /// Go's byte for byte, and Go compares the two as documents, so each
    /// keystroke comes back as an echo rather than a rewrite. The Markdown
    /// panel is Go's copy of the document; it must hold the typing.
    ///
    /// And it must hold the heading as a heading, not as bold text. A heading
    /// is drawn in a bold face, and the reverse mapping read that trait as the
    /// user's bold mark, so on Android the first keystroke turned the heading
    /// into `## **A note**` in Go's copy. GrMobRichMapper.boldMark is the fix
    /// here.
    func testRichTextTypingReachesGoWithTheHeadingPlain() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.14")
        let toggle = button(app, "Show the Markdown")
        scroll(app, to: toggle)
        toggle.tap()
        let editor = app.textViews.firstMatch
        XCTAssertTrue(editor.exists, "the rich-text editor has no text view")
        // The top right of the editor is the end of its first line, the
        // heading "A note".
        editor.coordinate(withNormalizedOffset: CGVector(dx: 0.97, dy: 0.04)).tap()
        sleep(1)
        XCTAssertEqual(app.keyboards.count, 1, "tapping the editor raised no keyboard")
        app.typeText("s/ok")
        sleep(1)
        let markdown = app.textViews.element(boundBy: 1)
        XCTAssertTrue(markdown.exists, "the Markdown panel did not open")
        let md = (markdown.value as? String) ?? ""
        shot("dp-4.14-typed")
        XCTAssertTrue(md.hasPrefix("## A notes/ok"), "Go's document lost the typing or the heading: \(md)")
        XCTAssertFalse(md.contains("**A note"), "the heading's face was read as a bold mark: \(md)")
        XCTAssertTrue(md.contains("Some **bold**"), "a real bold mark was lost: \(md)")
    }

    /// The code editor's echo path: typing into 4.13's buffer is echoed by Go
    /// through the text-edit stamps and must stay exactly as typed, with the
    /// caret where the typing is (an echo read as a rewrite would move it to
    /// the end of the buffer).
    func testCodeEditorTypingSurvivesItsEchoes() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.13")
        let editor = app.textViews.matching(NSPredicate(format: "value BEGINSWITH '// Try me'")).firstMatch
        scroll(app, to: editor)
        lift(app)
        XCTAssertTrue(editor.exists, "4.13's editor has no text view holding the snippet")
        // Near the start of the first line. The element's centre is below
        // four short lines, and a tap there puts the caret at the end of the
        // buffer instead. UIKit snaps the caret to a word boundary, so where
        // in the line it lands is not asserted, only that the typing stays
        // together in it.
        editor.coordinate(withNormalizedOffset: CGVector(dx: 0.05, dy: 0.04)).tap()
        sleep(1)
        let focused = app.textViews.matching(NSPredicate(format: "hasKeyboardFocus == true")).firstMatch
        XCTAssertTrue(focused.exists, "tapping the editor did not focus it")
        app.typeText("ok ")
        app.typeText("!")
        sleep(1)
        let value = (focused.value as? String) ?? ""
        shot("dp-4.13-typed")
        let first = value.components(separatedBy: "\n").first ?? ""
        // Both typeText calls in one piece in the first line: an echo read as
        // a rewrite would have moved the caret to the end of the buffer, and
        // the "!" would be there instead.
        XCTAssertTrue(first.contains("ok !"), "the typing was split or lost: \(value)")
        XCTAssertEqual(first.replacingOccurrences(of: "ok !", with: ""),
                       "// Try me: edit, and the colours follow.",
                       "the first line changed beyond the typing: \(first)")
    }

    // MARK: 2.3 — a TextArea on a device

    /// The multiline path of the text field, which no lesson exercised until
    /// 2.3 grew a TextArea: return must insert a newline (not submit), the
    /// newlines must reach Go as characters of the one string, and a rewrite
    /// that changes the *middle* of the value (Tidy lines trims each line and
    /// drops the blank one) must land while the field keeps focus, with
    /// typing after it continuing at the end.
    func testTextAreaTakesLinesAndATidyRewrite() throws {
        let app = XCUIApplication()
        open(app, lesson: "2.3")
        // A text view: a TextArea is a UITextView (GrMobTextInput.swift).
        let area = app.textViews.matching(NSPredicate(format: "value BEGINSWITH 'Milk'")).firstMatch
        scroll(app, to: area)
        lift(app)
        XCTAssertTrue(area.exists, "2.3's TextArea has no text view holding the seeded list")
        // Below the last line: UIKit puts the caret at the end of the text.
        area.coordinate(withNormalizedOffset: CGVector(dx: 0.9, dy: 0.95)).tap()
        sleep(1)
        XCTAssertEqual(app.keyboards.count, 1, "tapping the TextArea raised no keyboard")
        app.typeText("\nBread \n\nJam")
        sleep(1)
        XCTAssertTrue(any(app, beginningWith: "5 lines, 23 characters").waitForExistence(timeout: 3),
                      "Go's value did not hold the typed lines")
        shot("dp-2.3-typed")

        let tidy = button(app, "Tidy lines")
        XCTAssertTrue(tidy.isHittable, "Tidy lines is not reachable with the keyboard up")
        tidy.tap()
        XCTAssertTrue(any(app, beginningWith: "4 lines, 19 characters").waitForExistence(timeout: 3),
                      "the rewrite did not reach state")
        let focused = app.textViews.matching(NSPredicate(format: "hasKeyboardFocus == true")).firstMatch
        let value = (focused.exists ? focused.value : area.value) as? String ?? ""
        XCTAssertEqual(value, "Milk\nEggs\nBread\nJam", "the field did not take Go's rewrite")
        if focused.exists {
            // Typing after a focused rewrite goes where the caret was, clamped
            // to the shorter text: the end.
            app.typeText("Z")
            sleep(1)
            XCTAssertEqual(focused.value as? String, "Milk\nEggs\nBread\nJamZ",
                           "typing after the rewrite landed somewhere other than the end")
        }
        shot("dp-2.3-tidied")
    }

    /// A field whose onChange rewrites every key (2.3's UPPERCASE), typed
    /// into in one `typeText` call, which is fast enough that keys arrive
    /// while their predecessors' rewrites are landing.
    ///
    /// The SwiftUI TextField this runtime used lost keys to it: its binding is
    /// a second copy of the text, and a key reaching UIKit between a rewrite
    /// and SwiftUI pushing it down was overwritten unreported. "hello world"
    /// came out "HELLO WOD". The field is a UITextField now (see
    /// GrMobTextInput.swift), with one buffer.
    ///
    /// Then the caret: typing mid-text must stay where it is typed, not jump
    /// to the end on each rewrite.
        func testUppercaseMidTextKeepsTheCaretWithTheTyping() throws {
        let app = XCUIApplication()
        open(app, lesson: "2.3")
        let upper = app.switches.matching(NSPredicate(format: "label == 'UPPERCASE on the way in'")).firstMatch
        scroll(app, to: upper)
        upper.tap()
        let field = app.textFields.matching(NSPredicate(format: "placeholderValue BEGINSWITH 'Type your name'")).firstMatch
        XCTAssertTrue(field.exists, "2.3's name field is not on screen")
        field.tap()
        sleep(1)
        app.typeText("hello world")
        sleep(1)
        shot("dp-2.3-upper-typed")
        dump(app, "dp-2.3-upper-typed")
        // Re-found by value: the placeholder that found it is gone once the
        // field holds text, and the query re-resolves on every read.
        let typed = app.textFields.matching(NSPredicate(format: "value BEGINSWITH 'HELLO'")).firstMatch
        XCTAssertTrue(typed.waitForExistence(timeout: 3), "the transform did not run on the way in")
        XCTAssertEqual(typed.value as? String, "HELLO WORLD", "the transform did not run on the way in")
        // At the space: UIKit puts a tap's caret on a word boundary, and the
        // space is the only one inside the text.
        typed.coordinate(withNormalizedOffset: CGVector(dx: 0, dy: 0.5))
            .withOffset(CGVector(dx: widthOfHello(typed), dy: 0)).tap()
        sleep(1)
        app.typeText("abc")
        sleep(1)
        let value = (typed.value as? String) ?? ""
        shot("dp-2.3-upper-midtext")
        XCTAssertTrue(value == "HELLOABC WORLD" || value == "HELLO ABCWORLD",
                      "the typing left the caret's place: \(value)")
    }

    /// Where the space in "HELLO WORLD" sits, from the field's left edge:
    /// the field's text starts at its leading padding and the two words are
    /// about the same width, so the gap is a little under half the text.
    private func widthOfHello(_ field: XCUIElement) -> CGFloat {
        // 17pt system caps are ~11.5pt wide; HELLO is five of them, plus the
        // field's 12pt leading inset, plus half a space.
        12 + 5 * 11.5 + 2
    }

    /// 2.3's second transform capitalizes the first letter of every word,
    /// so a key typed mid-text makes Go change text on the far side of the
    /// caret. The caret must stay with the typing all the same.
    ///
    /// Two starts, because they reach different arms of the field's write
    /// (GrMobTextInputCoordinator.write):
    ///
    ///     "hello world", caret 5, type x → Go "Hellox World"
    ///       Go's span [0,8) holds the caret and keeps its length: the caret
    ///       stays at 6. The field used to put it at the span's end, 8.
    ///     "Hello world", caret 5, type x → Go "Hellox World"
    ///       Go's span [7,8) is after the caret, and the replacement leaves the
    ///       caret at 8 until it is moved back: a key queued behind the x
    ///       landed in that window.
    ///
    /// Either way three keys typed at once must read "Helloxyz World". A tap
    /// lands the caret on a word boundary; the other one it can pick, the
    /// start of "world", gives "Hello Xyzworld", which is right too.
    func testCapitalizedWordsKeepTheCaretWithTheTyping() throws {
        for seed in ["Hello world", "hello world"] {
            let app = XCUIApplication()
            open(app, lesson: "2.3")
            let field = app.textFields.matching(NSPredicate(format: "placeholderValue BEGINSWITH 'Type your name'")).firstMatch
            scroll(app, to: field)
            XCTAssertTrue(field.exists, "2.3's name field is not on screen")
            field.tap()
            sleep(1)
            app.typeText(seed)
            sleep(1)
            let words = app.switches.matching(NSPredicate(format: "label == 'Capitalize each word'")).firstMatch
            XCTAssertTrue(words.exists, "2.3 has no Capitalize each word switch")
            words.tap()
            sleep(1)
            let typed = app.textFields.matching(NSPredicate(format: "value BEGINSWITH[c] 'hello'")).firstMatch
            XCTAssertEqual(typed.value as? String, seed, "the seed did not arrive as typed")
            // Inside "hello", nearer its end than its start: the caret goes to 5.
            typed.coordinate(withNormalizedOffset: CGVector(dx: 0, dy: 0.5))
                .withOffset(CGVector(dx: 12 + 30, dy: 0)).tap()
            sleep(1)
            app.typeText("xyz")
            sleep(1)
            let value = (typed.value as? String) ?? ""
            shot("dp-2.3-words-\(seed.first!)")
            XCTAssertTrue(value == "Helloxyz World" || value == "Hello Xyzworld",
                          "from \(seed): the typing left the caret's place: \(value)")
            app.terminate()
        }
    }

    // MARK: 1.5 — core.ScrollIntoView

    /// Lesson 1.5's short Scroll of twelve named rows: "Jump to row 10"
    /// brings row 10 into the box's viewport through the ScrollViewReader
    /// GrMobScroll now provides, and "Back to row 1" brings row 1 back.
    ///
    /// Judged against the box's own frame, re-read after each jump: SwiftUI's
    /// scrollTo also scrolls the lesson's scroll view around the box when the
    /// box itself is not fully showing, so a position measured before the
    /// jump is not where the box is after it.
    func testScrollIntoViewJumpsInsideAScroll() throws {
        let app = XCUIApplication()
        open(app, lesson: "1.5")
        let jump = button(app, "Jump to row 10")
        scroll(app, to: jump)
        lift(app)
        let row1 = any(app, labelled: "Row 1"), row10 = any(app, labelled: "Row 10")
        XCTAssertTrue(row1.exists && row10.exists, "the demo's rows are not on screen")

        // The box: the smallest scroll view holding the rows.
        func box() -> CGRect {
            let views = app.scrollViews
                .containing(NSPredicate(format: "label == 'Row 10'")).allElementsBoundByIndex
            return views.map(\.frame).min { $0.height < $1.height } ?? .zero
        }
        func inside(_ row: XCUIElement) -> Bool {
            let b = box()
            return row.frame.minY >= b.minY - 1 && row.frame.maxY <= b.maxY + 1
        }
        // A known start first: the swipes that brought the demo up can land
        // on the box and scroll it too.
        button(app, "Back to row 1").tap()
        sleep(2)
        XCTAssertTrue(inside(row1), "row 1 did not come into the box: \(row1.frame) in \(box())")
        XCTAssertFalse(inside(row10), "row 10 is in view with row 1: \(row10.frame) in \(box())")
        jump.tap()
        sleep(2)
        shot("dp-1.5-jumped")
        XCTAssertTrue(inside(row10), "row 10 did not come into the box: \(row10.frame) in \(box())")
        XCTAssertFalse(inside(row1), "the box did not move off row 1")
    }

    // MARK: 4.14 — core.Paragraph through comps.RichTextView

    /// The read-only view under 4.14's editor draws each block as one
    /// core.Paragraph: the sentence is one text element holding its marks, and
    /// its link run is a link VoiceOver can reach on its own.
    func testRichTextViewDrawsARunOfMarksWithALink() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.14")
        let caption = any(app, beginningWith: "The same document, read-only")
        scroll(app, to: caption)
        lift(app)
        shot("dp-4.14-richtextview")
        dump(app, "dp-4.14-richtextview")
        let sentence = app.staticTexts.matching(NSPredicate(format: "label BEGINSWITH 'Some bold and some italic'")).firstMatch
        XCTAssertTrue(sentence.exists, "the paragraph is not one text element holding its runs")
        XCTAssertTrue(app.links.matching(NSPredicate(format: "label == 'link'")).firstMatch.exists,
                      "the link run is not exposed as a link")
    }

    // MARK: 4.29 — two links of different colours in one Paragraph

    /// 4.29's sentence holds two link runs: the guide (Link.Span, the theme's
    /// Primary) and "report a problem" (the theme's Error). Each is a link of
    /// its own, and tapping the second runs its own callback. The colours are
    /// the screenshot's to show: SwiftUI draws a link in the tint, and
    /// GrMobParagraph must still draw each run in its own.
    func testParagraphLinksKeepTheirOwnColours() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.29")
        let caption = any(app, beginningWith: "Problem reported")
        scroll(app, to: caption)
        lift(app)
        shot("dp-4.29-two-links")
        dump(app, "dp-4.29-two-links")
        XCTAssertTrue(any(app, beginningWith: "Problem reported 0 times").exists, "the counter did not start at 0")
        let report = app.links.matching(NSPredicate(format: "label == 'report a problem'")).firstMatch
        XCTAssertTrue(report.exists, "the second link run is not a link")
        XCTAssertTrue(app.links.matching(NSPredicate(format: "label == 'guide'")).firstMatch.exists,
                      "the first link run is not a link")
        report.tap()
        XCTAssertTrue(any(app, beginningWith: "Problem reported 1 time").waitForExistence(timeout: 3),
                      "tapping the second link did not run its callback")
    }

    // MARK: 4.33 — comps.MessageThread

    /// 4.33's thread over a pretend server of 48 messages, 12 a page:
    ///
    ///   - it opens at its end, "Message 48" in view;
    ///   - scrolled back to the top, one older page lands ("24 of 48") and
    ///     the message that was at the top stays where it was, so the reader
    ///     is not left at the new top, which would load the next page too;
    ///   - a message sent from the end is shown; one sent while scrolled back
    ///     leaves the reader where they are.
    ///
    /// The bubbles are found by their spoken names ("Ana, Message 37, 09:48"),
    /// which MessageBubble composes; the lesson's caption is Go's count.
    func testMessageThreadOpensAtTheEndAndKeepsThePlace() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.33")
        // Scrolled until the caption under the thread is hittable, and no
        // further: the whole box then sits above it on screen. lift() would
        // push the box off the top, where a drag cannot reach it.
        let caption = any(app, beginningWith: "12 of 48 messages loaded")
        scroll(app, to: caption)
        func bubble(_ n: Int) -> XCUIElement {
            app.descendants(matching: .any)
                .matching(NSPredicate(format: "label CONTAINS %@", "Message \(n),")).firstMatch
        }
        // The thread's box: the smallest scroll view holding a message,
        // re-read on each use because the page around it can move.
        func threadBox() -> CGRect {
            app.scrollViews.containing(NSPredicate(format: "label CONTAINS ', Message '"))
                .allElementsBoundByIndex.map(\.frame).min { $0.height < $1.height } ?? .zero
        }
        let box = threadBox()
        XCTAssertTrue(box.minY >= 0 && box.maxY <= app.frame.maxY, "the thread's box is not on screen: \(box)")
        func inBox(_ e: XCUIElement) -> Bool {
            let b = threadBox()
            return e.exists && e.frame.minY >= b.minY - 1 && e.frame.maxY <= b.maxY + 1
        }
        shot("dp-4.33-opened")
        dump(app, "dp-4.33-opened")
        XCTAssertTrue(inBox(bubble(48)), "the thread did not open on its last message: \(bubble(48).frame) in \(box)")

        // Back to the top, one drag at a time inside the box, until the first
        // loaded message (37) is in view; then the page lands on its own.
        let start = box.origin.y + box.height * 0.25, end = box.origin.y + box.height * 0.85
        var drags = 0
        while !inBox(bubble(37)) && drags < 12 {
            app.coordinate(withNormalizedOffset: .zero)
                .withOffset(CGVector(dx: box.midX, dy: start))
                .press(forDuration: 0.05, thenDragTo: app.coordinate(withNormalizedOffset: .zero)
                    .withOffset(CGVector(dx: box.midX, dy: end)), withVelocity: .slow, thenHoldForDuration: 0.3)
            drags += 1
        }
        dump(app, "dp-4.33-dragged")
        shot("dp-4.33-dragged")
        XCTAssertTrue(inBox(bubble(37)), "never scrolled back to the first loaded message")
        let before = bubble(37).frame.minY
        XCTAssertTrue(any(app, beginningWith: "24 of 48 messages loaded").waitForExistence(timeout: 5),
                      "reaching the top did not load the older page")
        sleep(2)
        shot("dp-4.33-older")
        XCTAssertFalse(any(app, beginningWith: "36 of 48").exists,
                       "one arrival at the top loaded two pages: the reader was left at the new top")
        XCTAssertEqual(bubble(37).frame.minY, before, accuracy: 2,
                       "the message at the top moved when the older page landed")

        // To the end again, then send from there.
        for _ in 0..<12 where !inBox(bubble(48)) {
            app.coordinate(withNormalizedOffset: .zero)
                .withOffset(CGVector(dx: box.midX, dy: end))
                .press(forDuration: 0.05, thenDragTo: app.coordinate(withNormalizedOffset: .zero)
                    .withOffset(CGVector(dx: box.midX, dy: start)), withVelocity: .fast, thenHoldForDuration: 0.1)
        }
        sleep(1)
        let field = app.textFields.matching(NSPredicate(format: "placeholderValue == 'Message…'")).firstMatch
        field.tap()
        app.typeText("hello\n")
        sleep(2)
        let hello = app.descendants(matching: .any).matching(NSPredicate(format: "label BEGINSWITH 'You, hello'")).firstMatch
        shot("dp-4.33-sent")
        XCTAssertTrue(inBox(hello), "a message sent from the end is not shown: \(hello.frame) in \(box)")

        // Scrolled back a little, a send must leave the reader where they are.
        app.coordinate(withNormalizedOffset: .zero)
            .withOffset(CGVector(dx: box.midX, dy: start))
            .press(forDuration: 0.05, thenDragTo: app.coordinate(withNormalizedOffset: .zero)
                .withOffset(CGVector(dx: box.midX, dy: end)), withVelocity: .slow, thenHoldForDuration: 0.3)
        sleep(1)
        let marker = bubble(45)
        XCTAssertTrue(inBox(marker), "message 45 is not in view after scrolling back")
        let at = marker.frame.minY
        field.tap()
        app.typeText("again\n")
        sleep(2)
        shot("dp-4.33-sent-back")
        XCTAssertEqual(bubble(45).frame.minY, at, accuracy: 2,
                       "a message sent while scrolled back moved the reader")
    }

    // MARK: 5.7 — PINInput, one field under the boxes

    /// A tap on the boxes focuses the one field, which brings up the number
    /// pad (core.Keyboard); a code typed in one burst lands whole, and a
    /// backspace deletes the last digit. The six-field version lost digits
    /// typed faster than the caret moved between boxes.
    func testPINInputTakesABurstThroughOneField() throws {
        let app = XCUIApplication()
        open(app, lesson: "5.7")
        let caption = any(app, beginningWith: "Value = ")
        scroll(app, to: caption)
        lift(app)
        dump(app, "dp-5.7-start")
        let field = app.textFields.matching(NSPredicate(format: "label BEGINSWITH 'One-time code,'")).firstMatch
        XCTAssertTrue(field.exists, "the one field is not there")
        // The boxes sit just above the caption; tap the middle of the row.
        caption.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: 0)).withOffset(CGVector(dx: 0, dy: -30)).tap()
        sleep(1)
        XCTAssertEqual(app.keyboards.count, 1, "tapping the boxes raised no keyboard")
        XCTAssertFalse(app.keyboards.buttons["return"].exists && app.keyboards.keys["q"].exists,
                       "the text keyboard came up, not the number pad")
        app.typeText("314159")
        XCTAssertTrue(any(app, beginningWith: "Value = \"314159\"").waitForExistence(timeout: 3),
                      "the burst did not land whole")
        XCTAssertTrue(any(app, beginningWith: "Value = \"314159\"   ·   OnComplete fired 1 time").exists,
                      "a full code should complete once")
        shot("dp-5.7-burst")
        app.typeText(XCUIKeyboardKey.delete.rawValue)
        XCTAssertTrue(any(app, beginningWith: "Value = \"31415\"").waitForExistence(timeout: 3),
                      "backspace did not delete the last digit")
    }

    // MARK: 2.6, 6.8 — the theme accent on platform controls

    /// A Toggle and a Slider in the theme's Primary, not the system green and
    /// blue. core.AccentColor reaches SwiftUI as `.tint`; no assertion can
    /// read a colour, so this drives the screenshots that show it.
    /// Lesson 2.6's Save button sits in a Row beside a caption long enough to
    /// overflow the line. The flex solver's min-content floor put every
    /// Button at 0, so the row squeezed the button instead of the caption and
    /// its label wrapped as "Sav / e". A Button now floors at its widest
    /// word plus its padding (GrMobMinContent), as a <button> does in CSS.
    ///
    /// A one-line label with the default 16/10 padding is wider than it is
    /// tall; "Sav" over "e" was not.
    func testTheSaveButtonKeepsItsLabelOnOneLine() throws {
        let app = XCUIApplication()
        open(app, lesson: "2.6")
        let save = button(app, "Save")
        scroll(app, to: save)
        lift(app)
        XCTAssertTrue(save.exists, "lesson 2.6 draws no Save button")
        shot("dp-2.6-save")
        let frame = save.frame
        XCTAssertGreaterThan(frame.width, frame.height,
                             "Save is \(frame.width)x\(frame.height): its label wrapped")
    }

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

    // MARK: 4.22–4.24 — the round-two widgets, first seen on iOS

    /// The FAB's three shapes and the layer it floats on. All three were
    /// verified on Android only. The sizes are the widget's contract
    /// (comps/fab.go): a regular disc is 56 points square, a small one 40, and
    /// an extended pill keeps the regular height so naming the action does not
    /// move it. The disc is also asserted to sit bottom-end in its stack, which
    /// is what Screen.Floating promises: its bottom-right corner is below and
    /// right of every other layer's content.
    func testFloatingActionButtonShapes() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.22")
        let disc = button(app, "New note")
        scroll(app, to: disc)
        lift(app)
        shot("dp-4.22-fab")
        dump(app, "dp-4.22-fab")

        XCTAssertEqual(disc.frame.width, 56, accuracy: 1, "the regular FAB is \(disc.frame)")
        XCTAssertEqual(disc.frame.height, 56, accuracy: 1, "the regular FAB is \(disc.frame)")
        // The pill's label is the glyph and the word, "✎  Compose".
        let pill = app.buttons.matching(NSPredicate(format: "label ENDSWITH %@", "Compose")).firstMatch
        XCTAssertEqual(pill.frame.height, 56, accuracy: 1, "the extended FAB is \(pill.frame)")
        XCTAssertGreaterThan(pill.frame.width, pill.frame.height, "the extended FAB is not a pill: \(pill.frame)")
        let small = button(app, "Back to top")
        XCTAssertEqual(small.frame.width, 40, accuracy: 1, "the small FAB is \(small.frame)")
        XCTAssertEqual(small.frame.height, 40, accuracy: 1, "the small FAB is \(small.frame)")

        // Bottom-end in the stack: the disc sits above the pill row (the stack
        // comes first in the panel) and at the panel's trailing side, right
        // of the pill row's middle.
        XCTAssertLessThan(disc.frame.maxY, pill.frame.minY, "the disc is not in the stack above the row")
        XCTAssertGreaterThan(disc.frame.midX, app.frame.midX, "the disc is not at the trailing end")

        // The demo seeds three notes, so two taps make five.
        XCTAssertTrue(any(app, labelled: "3 notes").exists, "4.22's caption did not start at 3 notes")
        disc.tap()
        disc.tap()
        XCTAssertTrue(any(app, labelled: "5 notes").waitForExistence(timeout: 3),
                      "two taps on the FAB did not reach Go")
        shot("dp-4.22-tapped")
    }

    /// The QR symbol is one Canvas path; on iOS the risk is seams between
    /// modules (anti-aliased edges of adjacent squares), which only a picture
    /// shows. The geometric half: it is square and the size it was given.
    func testQRCodeIsASquareOfItsSize() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.23")
        let code = any(app, labelled: "Scan to pair this device")
        scroll(app, to: code)
        lift(app)
        shot("dp-4.23-qr")
        XCTAssertTrue(code.exists, "lesson 4.23 draws no QR code")
        XCTAssertEqual(code.frame.width, code.frame.height, accuracy: 1, "the QR code is \(code.frame)")
        XCTAssertEqual(code.frame.width, 200, accuracy: 2, "the QR code is \(code.frame)")
    }

    /// Countdown and Stopwatch own a tick. The countdown is restarted at 10s
    /// and waited out: OnDone must fire once, from the effect, and the caption
    /// counts it. The stopwatch is started, left for two ticks, and paused.
    func testCountdownRunsOutOnceAndTheStopwatchPauses() throws {
        let app = XCUIApplication()
        open(app, lesson: "4.24")
        let restart = button(app, "Restart")
        scroll(app, to: restart)
        lift(app)
        dump(app, "dp-4.24-before")
        let ranOut = any(app, beginningWith: "Ran out ")
        XCTAssertTrue(ranOut.exists, "lesson 4.24 has no ran-out caption")
        let was = ranOut.label

        restart.tap()
        shot("dp-4.24-restarted")
        button(app, "Start").tap()
        sleep(3)
        shot("dp-4.24-running")
        button(app, "Pause").tap()
        XCTAssertTrue(button(app, "Start").waitForExistence(timeout: 3), "Pause did not stop the stopwatch")

        // 10s from the restart, plus a tick's slack.
        sleep(9)
        let now = any(app, beginningWith: "Ran out ").label
        XCTAssertNotEqual(now, was, "the countdown ran out and OnDone did not fire (still \(now))")
        shot("dp-4.24-ran-out")
        dump(app, "dp-4.24-after")
    }
}
