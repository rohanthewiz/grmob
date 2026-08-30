import XCTest

// Simulator pass for the todo app (examples/todoapp) — the SwiftUI analog of
// examples/todoapp/app_test.go. Requires the todo app in the framework:
//
//   ios/build.sh ./examples/todoapp
//
// and runs alone, since GrMobUITests drives the mobileapp demo bound by the
// default build:
//
//   xcodebuild test ... -only-testing:GrMobUITests/TodoAppUITests
//
// Beyond the CRUD flow it pins the regression the data-layer test cannot see:
// Go clearing the draft after Add must reach the *focused* TextField, whose
// local buffer otherwise swallows upstream writes (the echo/rewrite split in
// Renderer.swift's GrMobTextField).
final class TodoAppUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    /// Deletes every persisted row so each test starts from a clean slate.
    /// Necessary since the app began persisting (bytdb behind the mutation
    /// helpers): rows now survive both relaunches and previous test runs.
    /// Rows are found by their delete buttons' accessibility labels
    /// ("Delete <title>") — the ✕ glyph itself is not the accessible name.
    private func clearAllTodos(_ app: XCUIApplication) {
        let deletes = app.buttons.matching(NSPredicate(format: "label BEGINSWITH 'Delete '"))
        while deletes.firstMatch.exists {
            deletes.firstMatch.tap()
        }
        XCTAssertTrue(app.staticTexts["0 items left"].waitForExistence(timeout: 5),
                      "clearing leftover todos did not settle")
    }

    func testAddClearsFocusedInput() throws {
        let app = XCUIApplication()
        app.launch()

        XCTAssertTrue(app.staticTexts["Todos"].waitForExistence(timeout: 10),
                      "initial tree did not render")
        clearAllTodos(app)

        let field = app.textFields["What needs doing?"]
        XCTAssertTrue(field.waitForExistence(timeout: 5))
        field.tap()
        field.typeText("Buy milk")

        // The field keeps focus across the button tap — exactly the state in
        // which the clear-on-submit rewrite used to be dropped.
        app.buttons["Add"].tap()
        XCTAssertTrue(app.staticTexts["1 item left"].waitForExistence(timeout: 5),
                      "add did not round-trip")
        XCTAssertTrue(app.staticTexts["Buy milk"].waitForExistence(timeout: 5),
                      "new row did not render")

        // An empty TextField reports its prompt as its value; both spellings
        // of "empty" are accepted, anything else is the stale draft.
        let after = (field.value as? String) ?? ""
        XCTAssertTrue(after.isEmpty || after == "What needs doing?",
                      "input not cleared after add, still shows: \(after)")

        // Consecutive adds must work without manual erasing — the point of
        // clearing the draft in the first place. (Re-tap: the assertion above
        // doesn't depend on focus surviving the Add tap, so neither should
        // this typing.)
        field.tap()
        field.typeText("Walk dog")
        app.buttons["Add"].tap()
        XCTAssertTrue(app.staticTexts["2 items left"].waitForExistence(timeout: 5),
                      "second add did not round-trip")
        XCTAssertTrue(app.staticTexts["Walk dog"].waitForExistence(timeout: 5))

        // The return key is the second commit path: typing "\n" must add the
        // row exactly like the Add button (InputWithSubmit's onSubmit).
        field.tap()
        field.typeText("Feed cat\n")
        XCTAssertTrue(app.staticTexts["3 items left"].waitForExistence(timeout: 5),
                      "return-key submit did not round-trip")
        XCTAssertTrue(app.staticTexts["Feed cat"].waitForExistence(timeout: 5))
    }

    // The persistence path end to end on the real stack: the row must come
    // back after the process is killed and relaunched — i.e. out of the bytdb
    // file under Application Support (GomobileBridge registers the directory
    // before the first render), not out of the previous process's memory.
    func testTodosSurviveRelaunch() throws {
        let app = XCUIApplication()
        app.launch()

        XCTAssertTrue(app.staticTexts["Todos"].waitForExistence(timeout: 10),
                      "initial tree did not render")
        clearAllTodos(app)

        let field = app.textFields["What needs doing?"]
        XCTAssertTrue(field.waitForExistence(timeout: 5))
        field.tap()
        field.typeText("Persist me\n")
        XCTAssertTrue(app.staticTexts["1 item left"].waitForExistence(timeout: 5),
                      "add did not round-trip")

        // terminate() is a hard kill — no lifecycle hook runs on the Go side,
        // so this only passes if the write-through already hit the WAL when
        // the add's render pass returned.
        app.terminate()
        app.launch()

        XCTAssertTrue(app.staticTexts["Persist me"].waitForExistence(timeout: 10),
                      "todo did not survive relaunch")
        XCTAssertTrue(app.staticTexts["1 item left"].waitForExistence(timeout: 5),
                      "count not restored from disk")

        // Deleting after the relaunch proves the restored row carries a live
        // identity (its ID round-tripped, not just its title), and leaves the
        // store clean for the next run.
        app.buttons["Delete Persist me"].tap()
        XCTAssertTrue(app.staticTexts["0 items left"].waitForExistence(timeout: 5),
                      "restored row could not be deleted")
    }
}
