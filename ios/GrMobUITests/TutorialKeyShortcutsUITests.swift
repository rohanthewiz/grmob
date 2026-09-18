import XCTest

/// core.AccessibilityKeyShortcuts from a hardware keyboard, on lesson 2.2's
/// "Log from the keyboard" button, which declares "Control+Alt+K F6".
///
/// Both chords have a test, one each, because they reach the app by two different
/// routes:
///
///	Control+Alt+K   the Button's own keyboardShortcut (grMobKeyShortcut in
///	                GrMobStyle.swift)
///	F6              GameController's keyboard (GrMobFunctionKeys), since a
///	                KeyEquivalent has no function keys
///
/// XCUIElement.typeKey sends the key through the simulator's hardware keyboard
/// path, the one an iPad keyboard uses, with no text field focused: the case a
/// page-global shortcut exists for. Each press appends "n · button" to the
/// lesson's event log, which is what is read back.
///
/// # The first key press after launch is lost
///
/// On the iOS 26.5 simulator the first hardware key event an app receives
/// does nothing, whatever it is: Control+Option+K pressed three times in a
/// row, with nothing else between, logged 0, then 1, then 2 entries. So each
/// test primes the keyboard with a chord nothing on the page declares before
/// the press it measures. Both tests failed without it. They had passed on an
/// earlier simulator run, which suggests the lost event depends on how the
/// simulator attaches its keyboard rather than on anything in the app; the
/// web and Compose have no equivalent, and a first press on a real iPad has
/// not been checked.
///
/// It was first read as an on-screen rule for SwiftUI's keyboardShortcut,
/// because a probe that pressed once, scrolled the button into view and
/// pressed again saw only the second press land. Pressing three times
/// without scrolling showed it was the count, not the position.
final class TutorialKeyShortcutsUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    /// Spends the first key event, which the app never sees; see "The first
    /// key press after launch is lost". Control+Option+Z is declared by
    /// nothing in lesson 2.2, so if it does arrive it presses nothing, and
    /// the event log's "No events yet" is checked after it to prove that.
    private func primeKeyboard(_ app: XCUIApplication) {
        app.typeKey("z", modifierFlags: [.control, .option])
        sleep(1)
        XCTAssertTrue(text(app, beginningWith: "No events yet").exists,
                      "the priming chord pressed something")
    }

    private func text(_ app: XCUIApplication, beginningWith prefix: String) -> XCUIElement {
        app.staticTexts.matching(NSPredicate(format: "label BEGINSWITH %@", prefix)).firstMatch
    }

    func testTheModifierChordPressesTheButton() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/2.2")!)
        XCTAssertTrue(text(app, beginningWith: "2.2").waitForExistence(timeout: 10), "lesson 2.2 did not open")
        XCTAssertTrue(text(app, beginningWith: "No events yet").waitForExistence(timeout: 5),
                      "the event log should start empty")
        primeKeyboard(app)

        app.typeKey("k", modifierFlags: [.control, .option])
        XCTAssertTrue(text(app, beginningWith: "1 · button").waitForExistence(timeout: 5),
                      "Control+Option+K did not press \"Log from the keyboard\"")
    }

    /// F6, the button's second chord, which reaches the app through
    /// GameController (GrMobFunctionKeys) rather than SwiftUI.
    ///
    /// A known failure, not a passing test. No F-key from XCUITest has ever
    /// been seen to arrive this way: not on the 2026-09-13 run that added
    /// the route, and not on 2026-09-18, when F6 pressed four times in a row
    /// logged nothing with "Connect Hardware Keyboard" either on or at its
    /// default. The same presses reach Compose (KEYCODE_F6 logged a press on
    /// the emulator). Whether XCUITest's synthetic keys reach GCKeyboard at
    /// all is the open question, and a real iPad keyboard is the next
    /// check. XCTExpectFailure is strict, so the day F6 lands this test
    /// fails and says so, and the wrapper should come off.
    func testFunctionKeyPressesTheButton() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/2.2")!)
        XCTAssertTrue(text(app, beginningWith: "2.2").waitForExistence(timeout: 10), "lesson 2.2 did not open")
        XCTAssertTrue(text(app, beginningWith: "No events yet").waitForExistence(timeout: 5),
                      "the event log should start empty")
        primeKeyboard(app)

        app.typeKey(XCUIKeyboardKey.F6.rawValue, modifierFlags: [])
        XCTExpectFailure("F6 from XCUITest has never reached GCKeyboard on the simulator")
        XCTAssertTrue(text(app, beginningWith: "1 · button").waitForExistence(timeout: 5),
                      "F6 did not press \"Log from the keyboard\"")
    }

    /// The gesture card above the log declares "Control+Alt+J". A card is a
    /// tappable box rather than a Button, so the chord reaches it through
    /// GrMobGestures' invisible Button, and a press logs "tap".
    func testAChordPressesATappableBox() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/2.2")!)
        XCTAssertTrue(text(app, beginningWith: "2.2").waitForExistence(timeout: 10), "lesson 2.2 did not open")
        XCTAssertTrue(text(app, beginningWith: "No events yet").waitForExistence(timeout: 5),
                      "the event log should start empty")
        primeKeyboard(app)

        app.typeKey("j", modifierFlags: [.control, .option])
        XCTAssertTrue(text(app, beginningWith: "1 · tap").waitForExistence(timeout: 5),
                      "Control+Option+J did not press the gesture card")
    }
}
