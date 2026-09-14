import XCTest

/// core.AccessibilityKeyShortcuts from a hardware keyboard, on lesson 2.2's
/// "Log from the keyboard" button, which declares "Control+Alt+K F6".
///
/// Both chords are measured, because they reach SwiftUI by two different
/// routes (see grMobKeyShortcut in GrMobStyle.swift):
///
///	Control+Alt+K   the first chord    the Button's own keyboardShortcut
///	F6              the second chord   an invisible Button behind it, and a
///	                                   KeyEquivalent built from AppKit's
///	                                   function-key character (U+F709)
///
/// XCUIElement.typeKey sends the key through the simulator's hardware keyboard
/// path, the one an iPad keyboard uses, with no text field focused: the case a
/// page-global shortcut exists for. Each press appends "n · button" to the
/// lesson's event log, which is what is read back.
final class TutorialKeyShortcutsUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    private func text(_ app: XCUIApplication, beginningWith prefix: String) -> XCUIElement {
        app.staticTexts.matching(NSPredicate(format: "label BEGINSWITH %@", prefix)).firstMatch
    }

    func testBothDeclaredChordsPressTheButton() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/2.2")!)
        XCTAssertTrue(text(app, beginningWith: "2.2").waitForExistence(timeout: 10), "lesson 2.2 did not open")
        XCTAssertTrue(text(app, beginningWith: "No events yet").waitForExistence(timeout: 5),
                      "the event log should start empty")

        app.typeKey("k", modifierFlags: [.control, .option])
        XCTAssertTrue(text(app, beginningWith: "1 · button").waitForExistence(timeout: 5),
                      "Control+Option+K did not press \"Log from the keyboard\"")

        app.typeKey(XCUIKeyboardKey.F6.rawValue, modifierFlags: [])
        XCTAssertTrue(text(app, beginningWith: "2 · button").waitForExistence(timeout: 5),
                      "F6 did not press \"Log from the keyboard\"")
    }
}
