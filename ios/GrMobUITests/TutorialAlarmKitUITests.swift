import XCTest

// Simulator pass for the AlarmKit route (N-009): a hooks.UseAlarms alarm with
// Notify, in a build that keeps NSAlarmKitUsageDescription, rings as the
// system's alarm screen rather than a banner. Requires the tutorial in the
// framework and an iOS 26+ simulator:
//
//   ios/build.sh ./examples/tutorial
//   xcodebuild test ... -only-testing:GrMobUITests/TutorialAlarmKitUITests
//
// # What it holds
//
//   tap "Ring on the lock screen"   permission.Alarms → AlarmKit's dialog → Allow
//   tap "Ring at the next minute"   alarm.Alarm due at the next :00
//   Home, terminate                 UseAlarms posts LocalNotification{At, Alarm}
//                                   → Notifications.swift → Alarms.schedule
//   wait for the minute             SpringBoard shows the alarm, with Snooze
//   Snooze                          the alarm goes; the system re-rings it after
//                                   the lesson's one-minute snooze, by itself
//   Stop                            the alarm goes for good
//
// The Snooze button is the difference from the banner test next door: a
// banner has none, so finding it proves the post reached AlarmKit.
//
// With TEST_RUNNER_GRMOB_SHOTS_DIR set, each stage is saved as a screenshot.
// It takes about two and a half minutes, nearly all of it waiting.
final class TutorialAlarmKitUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    private func shot(_ name: String) {
        guard let dir = ProcessInfo.processInfo.environment["GRMOB_SHOTS_DIR"], !dir.isEmpty else { return }
        let png = XCUIScreen.main.screenshot().pngRepresentation
        try? png.write(to: URL(fileURLWithPath: dir).appendingPathComponent("\(name).png"))
    }

    func testAlarmRingsAsAnAlarmKitAlarmWithSnooze() throws {
        guard #available(iOS 26.0, *) else { throw XCTSkip("AlarmKit needs iOS 26") }
        let app = XCUIApplication()
        let springboard = XCUIApplication(bundleIdentifier: "com.apple.springboard")

        app.launch()
        app.open(URL(string: "grmob://lesson/4.19")!)
        let ring = app.buttons["Ring at the next minute"]
        XCTAssertTrue(ring.waitForExistence(timeout: 10), "lesson 4.19 did not open")
        var swipes = 0
        while !ring.isHittable && swipes < 20 {
            app.swipeUp(velocity: .slow)
            swipes += 1
        }

        // Notifications first (the lesson's own row), then AlarmKit. Each is
        // a SpringBoard dialog; "Allow" is the affirmative on both.
        for label in ["Allow notifications", "Ring on the lock screen"] {
            let button = app.buttons[label]
            guard button.waitForExistence(timeout: 3) else { continue }
            if !button.isHittable { app.swipeUp(velocity: .slow) }
            button.tap()
            let allow = springboard.buttons["Allow"]
            if allow.waitForExistence(timeout: 5) {
                shot("alarmkit-0-\(label == "Allow notifications" ? "notifications" : "alarmkit")-dialog")
                allow.tap()
            }
            _ = button.waitForNonExistence(timeout: 5)
        }
        XCTAssertFalse(app.buttons["Ring on the lock screen"].exists,
                       "permission.Alarms is not granted; the alarm would be a banner")
        shot("alarmkit-1-granted")

        let second = Calendar.current.component(.second, from: Date())
        if second > 45 {
            Thread.sleep(forTimeInterval: TimeInterval(62 - second))
        }
        ring.tap()
        let due = Calendar.current.date(
            byAdding: .minute, value: 1,
            to: Calendar.current.dateInterval(of: .minute, for: Date())!.start)!

        XCUIDevice.shared.press(.home)
        Thread.sleep(forTimeInterval: 3)
        app.terminate()

        // The alarm screen: its title is the alarm's Label, and it offers the
        // Snooze AlarmKit was given.
        let snooze = springboard.buttons["Snooze"]
        XCTAssertTrue(snooze.waitForExistence(timeout: max(10, due.timeIntervalSinceNow + 20)),
                      "no AlarmKit alarm with Snooze by \(due)")
        XCTAssertGreaterThanOrEqual(Date().timeIntervalSince(due), -1, "the alarm came before its minute")
        let title = springboard.descendants(matching: .any)
            .matching(NSPredicate(format: "label CONTAINS %@", "Try it")).firstMatch
        XCTAssertTrue(title.exists, "the alarm does not show the alarm's Label")
        shot("alarmkit-2-ringing")

        snooze.tap()
        Thread.sleep(forTimeInterval: 2)
        shot("alarmkit-3-snoozed")
        XCTAssertFalse(springboard.buttons["Snooze"].exists, "Snooze did not silence the alarm")

        // The lesson snoozes for one minute; the system rings it again.
        XCTAssertTrue(springboard.buttons["Snooze"].waitForExistence(timeout: 80),
                      "the snoozed alarm did not ring again")
        shot("alarmkit-4-rerung")

        let stop = springboard.buttons["Stop"]
        XCTAssertTrue(stop.exists, "the re-rung alarm has no Stop")
        stop.tap()
        XCTAssertTrue(springboard.buttons["Snooze"].waitForNonExistence(timeout: 5), "Stop did not end the alarm")
        shot("alarmkit-5-stopped")
    }
}
