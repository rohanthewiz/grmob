import XCTest

// Simulator pass for hooks.AlarmOptions.Notify in lesson 4.19 (examples/
// tutorial/chapter4.go, lessonClocksAndDrawing). Requires the tutorial in the
// framework:
//
//   ios/build.sh ./examples/tutorial
//
// and runs alone, since GrMobUITests drives the mobileapp demo bound by the
// default build:
//
//   xcodebuild test ... -only-testing:GrMobUITests/TutorialAlarmNotifyUITests
//
// # What it holds
//
// The whole Tier E chain, end to end, with the app not running when the alarm
// comes due:
//
//   tap "Ring at the next minute"      alarm.Alarm due at the next :00
//   Home                               "lifecycle" background → UseAlarms
//                                      posts LocalNotification{At}
//                                      → Notifications.swift schedules a
//                                      UNCalendarNotificationTrigger
//   terminate the app                  nothing of Go is left running
//   wait for the minute                SpringBoard draws the banner
//
// A banner that appears proves the request reached the notification center
// with a future trigger: had it been posted without one it would have shown
// at Home, before the app was terminated, and the test waits for the banner
// only after termination and checks the clock against the alarm's minute.
//
// It takes up to about 75 s, most of it waiting for the minute to turn.
final class TutorialAlarmNotifyUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    func testAlarmRingsAsANotificationWithTheAppClosed() throws {
        let app = XCUIApplication()
        let springboard = XCUIApplication(bundleIdentifier: "com.apple.springboard")

        watchForThePermissionAlert()

        app.launch()
        app.open(URL(string: "grmob://lesson/4.19")!)
        let ring = app.buttons["Ring at the next minute"]
        XCTAssertTrue(ring.waitForExistence(timeout: 10), "lesson 4.19 did not open")

        // Scroll the button into reach, then ask for permission if the lesson
        // still offers to.
        var swipes = 0
        while !ring.isHittable && swipes < 20 {
            app.swipeUp(velocity: .slow)
            swipes += 1
        }
        grantNotificationsIfOffered(app, springboard: springboard)

        // Too close to the minute and the banner could come before the app
        // is gone; wait into the next one.
        let second = Calendar.current.component(.second, from: Date())
        if second > 45 {
            Thread.sleep(forTimeInterval: TimeInterval(62 - second))
        }
        ring.tap()
        let due = Calendar.current.date(
            byAdding: .minute, value: 1,
            to: Calendar.current.dateInterval(of: .minute, for: Date())!.start)!

        // Leave the screen, give the background report time to reach Go and
        // the request time to reach the notification center, then kill the app.
        XCUIDevice.shared.press(.home)
        Thread.sleep(forTimeInterval: 3)
        app.terminate()
        XCTAssertEqual(app.state, .notRunning)

        // SpringBoard's banner carries the alarm's Label ("Try it").
        let banner = springboard.descendants(matching: .any)
            .matching(NSPredicate(format: "label CONTAINS %@", "Try it")).firstMatch
        let wait = max(10, due.timeIntervalSinceNow + 20)
        XCTAssertTrue(banner.waitForExistence(timeout: wait),
                      "no notification banner by \(due) with the app closed")
        XCTAssertGreaterThanOrEqual(Date().timeIntervalSince(due), -1,
                                    "the banner came before the alarm's minute")
    }

    // # The relaunch sweep (core.SweepNotifications)
    //
    //   tap "Ring at the next minute", Home   the request is scheduled, as above
    //   terminate                             the record of its id dies with Go
    //   launch, open 4.19                     UseAlarms mounts in the foreground
    //                                         and sweeps "grmob.alarm."
    //   Home, wait past the minute            no banner: the request was removed
    //
    // Without the sweep the dead process's request survives and the banner
    // arrives on time. The second Home matters: it is the app leaving the
    // screen with a fresh alarm list (nothing enabled), so it schedules
    // nothing new that could be mistaken for, or hide, the old request.
    func testRelaunchSweepsWhatTheDeadProcessScheduled() throws {
        let app = XCUIApplication()
        let springboard = XCUIApplication(bundleIdentifier: "com.apple.springboard")
        watchForThePermissionAlert()

        app.launch()
        app.open(URL(string: "grmob://lesson/4.19")!)
        let ring = app.buttons["Ring at the next minute"]
        XCTAssertTrue(ring.waitForExistence(timeout: 10), "lesson 4.19 did not open")
        var swipes = 0
        while !ring.isHittable && swipes < 20 {
            app.swipeUp(velocity: .slow)
            swipes += 1
        }
        // Granted here rather than asserted: this test used to fail on a fresh
        // simulator unless the other one had run first, which made its result
        // depend on test order. A banner that never comes is only evidence of
        // the sweep if one would otherwise have come, so the grant has to be
        // in place and this makes sure it is.
        grantNotificationsIfOffered(app, springboard: springboard)
        XCTAssertFalse(app.buttons["Allow notifications"].exists,
                       "notifications are still not granted; the missing banner would prove nothing")

        // Room for the relaunch before the minute turns.
        let second = Calendar.current.component(.second, from: Date())
        if second > 30 {
            Thread.sleep(forTimeInterval: TimeInterval(62 - second))
        }
        ring.tap()
        let due = Calendar.current.date(
            byAdding: .minute, value: 1,
            to: Calendar.current.dateInterval(of: .minute, for: Date())!.start)!

        XCUIDevice.shared.press(.home)
        Thread.sleep(forTimeInterval: 3)
        app.terminate()

        app.launch()
        app.open(URL(string: "grmob://lesson/4.19")!)
        XCTAssertTrue(ring.waitForExistence(timeout: 10), "lesson 4.19 did not reopen")
        XCTAssertLessThan(Date(), due, "the relaunch took past the alarm's minute; the test proves nothing")
        Thread.sleep(forTimeInterval: 2)
        XCUIDevice.shared.press(.home)

        let banner = springboard.descendants(matching: .any)
            .matching(NSPredicate(format: "label CONTAINS %@", "Try it")).firstMatch
        let wait = max(5, due.timeIntervalSinceNow + 15)
        XCTAssertFalse(banner.waitForExistence(timeout: wait),
                       "the dead process's alarm still rang after a relaunch swept it")
    }

    // MARK: permission

    /// The system's permission alert belongs to SpringBoard; the monitor
    /// answers it whenever the app next receives an interaction.
    private func watchForThePermissionAlert() {
        addUIInterruptionMonitor(withDescription: "notifications") { alert in
            let allow = alert.buttons["Allow"]
            if allow.exists { allow.tap(); return true }
            return false
        }
    }

    /// Taps 4.19's "Allow notifications" when the lesson still offers it, and
    /// answers the system alert that follows. A no-op once granted, since the
    /// lesson stops offering the button.
    private func grantNotificationsIfOffered(_ app: XCUIApplication, springboard: XCUIApplication) {
        let allow = app.buttons["Allow notifications"]
        guard allow.exists else { return }
        if !allow.isHittable { app.swipeUp(velocity: .slow) }
        allow.tap()
        // The alert is SpringBoard's; answer it directly when it is up,
        // and nudge the monitor otherwise.
        let sbAllow = springboard.buttons["Allow"]
        if sbAllow.waitForExistence(timeout: 5) {
            sbAllow.tap()
        } else {
            app.tap()
        }
        // The lesson re-renders once the grant reaches Go.
        _ = allow.waitForNonExistence(timeout: 5)
    }
}
