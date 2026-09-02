import XCTest

// Simulator pass for core's audio service on iOS: load/play, status ticks
// reaching the tree over the host-event channel, skip, speed, the Slider
// seek, pause, play-to-end and stop. Needs the network (the demo streams a
// SoundHelix sample) and takes about a minute. Writes a text dump of every
// label plus a screenshot per checkpoint under GRMOB_AUDIO_OUT (default:
// the simulator's temp dir), because test stdout reaches neither the
// xcodebuild log nor the xcresult. Run with:
//   xcodebuild test ... -only-testing:GrMobUITests/AudioUITests
final class AudioUITests: XCTestCase {
    let out = (ProcessInfo.processInfo.environment["GRMOB_AUDIO_OUT"] ?? NSTemporaryDirectory()) + "/grmob-audio"

    func dump(_ app: XCUIApplication, _ name: String) {
        let texts = app.staticTexts.allElementsBoundByIndex.map { $0.label }.joined(separator: " | ")
        try? texts.write(toFile: "\(out)-\(name).txt", atomically: true, encoding: .utf8)
        let shot = XCUIScreen.main.screenshot().pngRepresentation
        try? shot.write(to: URL(fileURLWithPath: "\(out)-\(name).png"))
    }

    func testAudioTransport() throws {
        let app = XCUIApplication()
        app.launch()
        XCTAssertTrue(app.staticTexts["Count: 0"].waitForExistence(timeout: 15))
        app.buttons["Audio"].tap()
        XCTAssertTrue(app.staticTexts["Nothing loaded"].waitForExistence(timeout: 5))
        dump(app, "1-idle")

        app.buttons["Play"].tap()
        XCTAssertTrue(app.staticTexts["playing"].waitForExistence(timeout: 20), "never reached playing")
        sleep(4)
        dump(app, "2-playing")

        app.buttons["+15s"].tap()
        app.buttons["Speed 1x"].tap()
        sleep(2)
        dump(app, "3-skip-speed")

        // Drag the seek slider to ~90%.
        let slider = app.sliders.firstMatch
        XCTAssertTrue(slider.exists, "no slider")
        slider.adjust(toNormalizedSliderPosition: 0.9)
        sleep(2)
        dump(app, "4-seeked")

        app.buttons["Pause"].tap()
        XCTAssertTrue(app.staticTexts["paused"].waitForExistence(timeout: 5))
        dump(app, "5-paused")

        // Let it run out: from ~90% of a 6:13 track at 1.25x, the end is
        // ~30s away. Play, then wait for "ended".
        app.buttons["Play"].tap()
        XCTAssertTrue(app.staticTexts["ended"].waitForExistence(timeout: 60), "never reached ended")
        dump(app, "6-ended")

        app.buttons["Stop"].tap()
        XCTAssertTrue(app.staticTexts["Nothing loaded"].waitForExistence(timeout: 5))
        dump(app, "7-stopped")
    }
}
