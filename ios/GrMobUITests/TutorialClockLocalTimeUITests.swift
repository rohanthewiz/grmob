import XCTest

// Simulator pass for two lesson 4.19 facts that only a running app can show
// (examples/tutorial/chapter4.go, lessonClocksAndDrawing). Requires the
// tutorial in the framework:
//
//   ios/build.sh ./examples/tutorial
//
// and runs alone, like the other Tutorial* classes:
//
//   xcodebuild test ... -only-testing:GrMobUITests/TutorialClockLocalTimeUITests
//
// # What it holds
//
//   the clock reads local time   GomobileBridge calls mobile.SetTimeZone before
//                                the first render. On Android that was the fix
//                                for Go running in UTC; iOS already had
//                                /etc/localtime, so here the check is that the
//                                call did not make things worse — the digital
//                                clock's spoken label must match the
//                                simulator's own wall clock, not UTC's.
//
//   no exact-alarm button        permission.ExactAlarms is always Granted on
//                                iOS (calendar triggers are exact), and the
//                                lesson shows "Allow exact alarms" only for
//                                Denied — so the button must never appear.
//
// # Screenshots
//
// With TEST_RUNNER_GRMOB_SHOTS_DIR in xcodebuild's environment each step
// writes a PNG, which is how the canvas's miter corners and edge dots on 4.19
// are checked by eye.
final class TutorialClockLocalTimeUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    private func shot(_ name: String) {
        guard let dir = ProcessInfo.processInfo.environment["GRMOB_SHOTS_DIR"], !dir.isEmpty else { return }
        let png = XCUIScreen.main.screenshot().pngRepresentation
        try? png.write(to: URL(fileURLWithPath: dir).appendingPathComponent("\(name).png"))
    }

    func testTheClockReadsLocalTimeAndExactAlarmsNeedNoButton() throws {
        let app = XCUIApplication()
        app.launch()
        app.open(URL(string: "grmob://lesson/4.19")!)

        // The DigitalClock is one element whose label starts with its digits,
        // "h:mm:ss" and then the AM/PM marker (the lesson starts in 12-hour).
        let clock = app.descendants(matching: .any)
            .matching(NSPredicate(format: "label MATCHES %@", "^[0-9]{1,2}:[0-9]{2}:[0-9]{2} (AM|PM).*"))
            .firstMatch
        XCTAssertTrue(clock.waitForExistence(timeout: 10), "lesson 4.19's digital clock did not appear")
        shot("clock-0")

        // Read the label and the device clock back to back. Minutes are
        // compared, allowing the one-minute turn that can fall between them;
        // a zone error is whole hours, which no tolerance here can absorb.
        let label = clock.label
        let now = Date()
        let parts = label.split(separator: " ")
        let hms = parts[0].split(separator: ":").compactMap { Int($0) }
        XCTAssertEqual(hms.count, 3, "unparseable clock label \(label)")
        var hour = hms[0] % 12
        if parts.count > 1 && parts[1].hasPrefix("PM") { hour += 12 }
        let shown = hour * 60 + hms[1]

        let cal = Calendar.current
        let local = cal.component(.hour, from: now) * 60 + cal.component(.minute, from: now)
        let diff = abs(shown - local) % (24 * 60)
        XCTAssertTrue(diff <= 1 || diff >= 24 * 60 - 1,
                      "clock label \(label) is not local time (\(cal.timeZone.identifier), " +
                      "\(local / 60):\(local % 60)) — Go's time.Local is wrong")

        // Scroll to the alarm demo, where both permission buttons live, and
        // give the permission check time to answer.
        let ring = app.buttons["Ring at the next minute"]
        var swipes = 0
        while !(ring.exists && ring.isHittable) && swipes < 20 {
            app.swipeUp(velocity: .slow)
            swipes += 1
            if swipes % 3 == 0 { shot("clock-scroll-\(swipes)") }
        }
        XCTAssertTrue(ring.isHittable, "the alarm demo was not reached")
        Thread.sleep(forTimeInterval: 2)
        shot("clock-alarms")
        XCTAssertFalse(app.buttons["Allow exact alarms"].exists,
                       "iOS answers ExactAlarms Granted, so the lesson must not offer the button")
    }
}
