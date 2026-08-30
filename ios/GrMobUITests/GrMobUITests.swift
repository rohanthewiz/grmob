import XCTest

// On-device pass for the SwiftUI renderer: drives the bound demo app
// (examples/mobileapp) through every event kind the bridge carries — void
// (Button), int (TabView), text (Input), bool (Checkbox/Toggle) — plus the
// async Go→native push channel (UseInterval). This is the simulator analog of
// examples/mobileapp/app_test.go and ios/verify: those prove the data layer;
// this proves the SwiftUI layer wired to it end to end, real taps included.
//
// Element lookup leans on SwiftUI's default accessibility: Button/TextField/
// Toggle surface as-is; the hand-rolled tab bar wraps each label in a Button,
// so tabs are addressed as buttons.
final class GrMobUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    func testAllEventKindsAndPushChannel() throws {
        let app = XCUIApplication()
        app.launch()

        // -- Initial render (mount JSON -> tree -> SwiftUI) --
        let count = app.staticTexts["Count: 0"]
        XCTAssertTrue(count.waitForExistence(timeout: 10), "initial tree did not render")

        // -- Push channel: UseInterval ticks with no native event in flight.
        // Wait for the elapsed-seconds label to change on its own.
        let running = app.staticTexts.matching(
            NSPredicate(format: "label BEGINSWITH 'App running for '")).firstMatch
        XCTAssertTrue(running.waitForExistence(timeout: 5))
        let before = running.label
        let changed = NSPredicate { _, _ in running.label != before }
        let tick = expectation(for: changed, evaluatedWith: nil)
        wait(for: [tick], timeout: 10)

        // -- Void events: three taps must land as three distinct increments
        // (each round-trips Go -> patch -> tree before the next).
        let increment = app.buttons["Increment"]
        for _ in 0..<3 { increment.tap() }
        XCTAssertTrue(app.staticTexts["Count: 3"].waitForExistence(timeout: 5),
                      "void events lost or coalesced")

        // -- Int event: tab change is a controlled index owned by Go; the
        // Form tab's subtree only appears once Go re-renders with index 1.
        app.buttons["Form"].tap()
        let nameField = app.textFields["Your name"]
        XCTAssertTrue(nameField.waitForExistence(timeout: 5), "tab switch did not reach Go")
        XCTAssertTrue(app.staticTexts["Hello, stranger."].exists)

        // -- Text events: typed text flows Go-ward per keystroke; the greeting
        // is derived state coming back down.
        nameField.tap()
        nameField.typeText("Ada")
        XCTAssertTrue(app.staticTexts["Hello, Ada!"].waitForExistence(timeout: 5),
                      "text events did not round-trip")

        // -- Bool event: the Checkbox renders as a Toggle (iOS has no checkbox).
        let toggle = app.switches.firstMatch
        XCTAssertTrue(toggle.waitForExistence(timeout: 5))
        toggle.tap()
        XCTAssertTrue(app.staticTexts["Subscribed"].waitForExistence(timeout: 5),
                      "bool event did not round-trip")

        // -- State retention: both tab subtrees live in one Go tree, so the
        // counter must survive the round trip to Form and back.
        app.buttons["Counter"].tap()
        XCTAssertTrue(app.staticTexts["Count: 3"].waitForExistence(timeout: 5),
                      "counter state lost across tab switch")
    }

    // The gap-5 surface on a real simulator: List virtualization, container
    // gestures (tap/long-press on Rows), and the accessibility semantics the
    // Go styles declare — the rows are addressed here BY their accessibility
    // label + button trait, so element lookup itself proves the a11y wiring.
    func testFeedListGesturesAndVirtualization() throws {
        let app = XCUIApplication()
        app.launch()

        app.buttons["Feed"].tap()
        XCTAssertTrue(app.staticTexts["Nothing selected"].waitForExistence(timeout: 10),
                      "feed tab did not render")

        // Virtualization probe: the last row must not be in the viewport while
        // the list shows its top. `exists` is deliberately not asserted false —
        // the accessibility snapshot XCUITest queries through is allowed to
        // realize offscreen lazy rows (VoiceOver enumerates them), and does.
        // isHittable is viewport-dependent: false here, true after scrolling.
        let lastRow = app.buttons["Article 30"]
        XCTAssertFalse(lastRow.isHittable,
                       "row 30 is on screen at the top of the list")

        // Tap = select. The row's accessibility label flips to "…, selected".
        app.buttons["Article 3"].tap()
        XCTAssertTrue(app.staticTexts["Selected: Article 3"].waitForExistence(timeout: 5),
                      "row tap did not round-trip to Go")
        XCTAssertTrue(app.buttons["Article 3, selected"].waitForExistence(timeout: 5),
                      "selected row's accessibility label did not update")

        // Long-press = star, asserted through the status line. (The row's
        // starred title "Article 5 ★" is deliberately NOT queried as a static
        // text: the accessibility label on the row combines its children into
        // one element, so inner texts are not individually exposed — that is
        // the a11y design, one swipe stop per row.)
        app.buttons["Article 5"].press(forDuration: 0.8)
        XCTAssertTrue(
            app.staticTexts["Selected: Article 3 · Starred: Article 5"]
                .waitForExistence(timeout: 5),
            "long-press did not round-trip to Go")

        // Scroll far enough and the tail row enters the viewport.
        var swipes = 0
        while !lastRow.isHittable && swipes < 8 {
            app.swipeUp()
            swipes += 1
        }
        XCTAssertTrue(lastRow.isHittable,
                      "row 30 never came on screen after scrolling")
    }
}
