import XCTest

/// Simulator pass for lesson 4.42's StockTicker (N-121): the stepping tape,
/// and the gliding one that core.Glide drives (N-120).
///
/// What is asserted is motion and geometry, read off XCUITest's frames: a
/// glide moves the labelled copy toward the leading edge at one width per
/// period, the pass restarts seamlessly, the hidden second copy still takes a
/// tap, Reduce Motion holds the tape at rest, and the direction flips under a
/// right-to-left language. Screenshots go to GRMOB_SHOTS_DIR, as the other
/// tutorial passes do, for the glyphs and the seam.
///
/// Requires the tutorial in the framework (`ios/build.sh ./examples/tutorial`)
/// and runs alone:
///
///   TEST_RUNNER_GRMOB_SHOTS_DIR=/some/dir xcodebuild test ... \
///     -only-testing:GrMobUITests/TutorialStockTickerUITests
///
/// Reduce Motion is the simulator's setting, not the test's:
///
///   xcrun simctl spawn booted defaults write com.apple.Accessibility ReduceMotionEnabled -bool true
///
/// `testGlideFollowsReduceMotion` reads the setting from the runner's own
/// process (same simulator, same preference) and asserts the matching
/// behaviour, so one test serves both runs.
final class TutorialStockTickerUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    // MARK: helpers (TutorialRoundFourUITests' shapes)

    private func any(_ app: XCUIApplication, beginningWith prefix: String) -> XCUIElement {
        app.descendants(matching: .any)
            .matching(NSPredicate(format: "label BEGINSWITH %@", prefix)).firstMatch
    }

    private func scroll(_ app: XCUIApplication, to target: XCUIElement, max: Int = 40) {
        var swipes = 0
        while !(target.exists && target.isHittable) && swipes < max {
            app.swipeUp(velocity: .slow)
            swipes += 1
        }
        sleep(1)
    }

    private func shot(_ name: String) {
        guard let dir = ProcessInfo.processInfo.environment["GRMOB_SHOTS_DIR"], !dir.isEmpty else { return }
        let png = XCUIScreen.main.screenshot().pngRepresentation
        try? png.write(to: URL(fileURLWithPath: dir).appendingPathComponent("\(name).png"))
    }

    private func dump(_ app: XCUIApplication, _ name: String) {
        guard let dir = ProcessInfo.processInfo.environment["GRMOB_SHOTS_DIR"], !dir.isEmpty else { return }
        try? app.debugDescription.write(to: URL(fileURLWithPath: dir).appendingPathComponent("\(name).txt"),
                                        atomically: true, encoding: .utf8)
    }

    /// Opens 4.42 and scrolls until the demo's caption is on screen, then one
    /// more slow swipe so the tape and its three checkboxes sit mid-screen.
    private func openTicker(_ app: XCUIApplication, rtl: Bool = false) {
        if rtl {
            app.launchArguments += ["-AppleLanguages", "(ar)", "-AppleLocale", "ar",
                                    "-AppleTextDirection", "YES", "-NSForceRightToLeftWritingDirection", "YES"]
        }
        app.launch()
        app.open(URL(string: "grmob://lesson/4.42")!)
        XCTAssertTrue(any(app, beginningWith: "4.42").waitForExistence(timeout: 15), "lesson 4.42 did not open")
        // Until the Glide switch is on screen, and no further: one more swipe
        // carried the tape, about 160pt above the switch, off the top.
        scroll(app, to: any(app, beginningWith: "Glide the tape"))
    }

    /// The tape a reader hears: the node labelled "Stock prices" (with OnTap
    /// set the quotes are buttons, so the tape is a group, not a list). A
    /// stepping tape is a Scroll and a gliding one an HBox, so any element
    /// type. When gliding this is the first copy; the second is hidden from
    /// the tree.
    private func tape(_ app: XCUIApplication) -> XCUIElement {
        app.descendants(matching: .any).matching(NSPredicate(format: "label == %@", "Stock prices")).firstMatch
    }

    /// XCUITest's frame queries take tens of milliseconds each, so one sample
    /// can land early or late; the mean over the run is the speed.
    private func mean(_ v: [CGFloat]) -> CGFloat { v.reduce(0, +) / CGFloat(max(v.count, 1)) }

    private func toggle(_ app: XCUIApplication, _ label: String) {
        any(app, beginningWith: label).tap()
        sleep(1)
    }

    /// Samples the tape's leading x at `interval` for `count` samples and
    /// returns the per-second velocity between consecutive samples.
    private func velocities(_ app: XCUIApplication, count: Int = 6, interval: TimeInterval = 0.25) -> [CGFloat] {
        var xs: [(Date, CGFloat)] = []
        for _ in 0..<count {
            xs.append((Date(), tape(app).frame.minX))
            RunLoop.current.run(until: Date().addingTimeInterval(interval))
        }
        // A pass restarting between two samples reads as thousands of points
        // a second the other way (the whole width in a quarter second); that
        // is the loop's seam, not its speed, so such a pair is left out.
        return zip(xs, xs.dropFirst())
            .map { a, b in (b.1 - a.1) / CGFloat(b.0.timeIntervalSince(a.0)) }
            .filter { abs($0) < 1000 }
    }

    // MARK: the stepping tape

    func testTapeStepsEveryThreeSeconds() throws {
        let app = XCUIApplication()
        openTicker(app)
        let strip = tape(app)
        XCTAssertTrue(strip.exists, "no tape labelled Stock prices")
        dump(app, "st-4.42-stepping")
        shot("st-4.42-step-0")
        // Each quote is one element whose label is the spoken sentence.
        let spoken = NSPredicate(format: "label CONTAINS %@ OR label CONTAINS %@", ", up ", ", down ")
        let first = app.descendants(matching: .any).matching(spoken).firstMatch
        XCTAssertTrue(first.waitForExistence(timeout: 5), "no quote with a spoken sentence")
        let before = app.descendants(matching: .any).matching(spoken).allElementsBoundByIndex
            .filter { $0.frame.intersects(strip.frame) }.map(\.label)
        sleep(4)
        shot("st-4.42-step-1")
        let after = app.descendants(matching: .any).matching(spoken).allElementsBoundByIndex
            .filter { $0.frame.intersects(strip.frame) }.map(\.label)
        XCTAssertNotEqual(before.first?.prefix(6), after.first?.prefix(6),
                          "the first quote on the tape did not change in 4s: \(before) vs \(after)")

        // Paused, the tape pans under a finger: a swipe toward the leading
        // edge brings later quotes into the clip. Paused first, so that a step
        // landing mid-swipe is not what changed the quotes on screen.
        toggle(app, "Pause the tape")
        let still = app.descendants(matching: .any).matching(spoken).allElementsBoundByIndex
            .filter { $0.frame.intersects(strip.frame) }.map(\.label)
        strip.swipeLeft()
        sleep(1)
        shot("st-4.42-step-panned")
        let panned = app.descendants(matching: .any).matching(spoken).allElementsBoundByIndex
            .filter { $0.frame.intersects(strip.frame) }.map(\.label)
        XCTAssertNotEqual(still.first?.prefix(6), panned.first?.prefix(6),
                          "a swipe did not pan the paused tape: \(still) vs \(panned)")
    }

    // MARK: the gliding tape

    func testGlideMovesBothCopiesSeamlesslyAndTheHiddenCopyTakesATap() throws {
        let app = XCUIApplication()
        openTicker(app)
        toggle(app, "Glide the tape")
        let strip = tape(app)
        XCTAssertTrue(strip.waitForExistence(timeout: 5))
        dump(app, "st-4.42-glide")
        shot("st-4.42-glide-0")

        // The labelled copy slides toward the leading edge, the left in LTR,
        // at one width per 15s: five quotes, three seconds each.
        // The frame XCUITest reports for the copy is the union of its quotes
        // (its leading padding is not in it), so the width here is the
        // quotes' span, a little under the copy's own width.
        let width = strip.frame.width
        let v = velocities(app)
        let expected = -width / 15
        for each in v {
            XCTAssertLessThan(each, 0, "the tape is not moving left: \(v)")
        }
        XCTAssertEqual(mean(v), expected, accuracy: max(8, abs(expected) * 0.15),
                       "the glide's speed is not one width per 15s (width \(width)): \(v)")

        // The seam: once the first copy's last quote (AMZN) is inside the
        // clip, the second copy's AAPL follows it after the gap, and a tap
        // there reaches that hidden copy. 119pt past AMZN's end is inside a
        // 198pt AAPL whether the 24pt gap is there or not, so the readout,
        // which names the quote that took the tap, is the proof of the tap and
        // the screenshot is the proof of the gap. Start is 0 while gliding, so
        // an AAPL to the right of AMZN can only be the second copy's.
        let amzn = app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Amazon, AMZN")).firstMatch
        let screenRight = app.windows.firstMatch.frame.maxX
        let deadline = Date().addingTimeInterval(20)
        var tapped = false
        while Date() < deadline {
            let f = amzn.frame
            if f.width > 0 && f.maxX + 24 + 190 < screenRight - 8 {
                shot("st-4.42-glide-seam")
                app.coordinate(withNormalizedOffset: .zero)
                    .withOffset(CGVector(dx: f.maxX + 119, dy: f.midY)).tap()
                tapped = true
                break
            }
            RunLoop.current.run(until: Date().addingTimeInterval(0.1))
        }
        XCTAssertTrue(tapped, "the first copy's AMZN never came fully on screen (frame \(amzn.frame))")
        let readout = any(app, beginningWith: "OnTap got")
        XCTAssertTrue(readout.waitForExistence(timeout: 3), "no readout after tapping the second copy")
        XCTAssertEqual(readout.label, "OnTap got Apple (AAPL)",
                       "the tap after the first copy's end did not reach the second copy's AAPL")
        shot("st-4.42-glide-tapped")

        // Pause: the glide is replaced by the stepping tape, which is still.
        toggle(app, "Pause the tape")
        let still = velocities(app, count: 3, interval: 0.4)
        for each in still {
            XCTAssertEqual(each, 0, accuracy: 0.5, "a paused tape still moves: \(still)")
        }
        shot("st-4.42-glide-paused")
    }

    func testGlideFollowsReduceMotion() throws {
        let app = XCUIApplication()
        openTicker(app)
        toggle(app, "Glide the tape")
        let strip = tape(app)
        XCTAssertTrue(strip.waitForExistence(timeout: 5))
        let reduce = UIAccessibility.isReduceMotionEnabled
        let v = velocities(app)
        shot(reduce ? "st-4.42-glide-reduce-motion" : "st-4.42-glide-motion")
        if reduce {
            // At rest: still, with AAPL first, one gap (24pt) inside the
            // clip's leading edge. The clip is the demo panel's content box,
            // whose x the readout caption below the tape shares; the tape's
            // own frame cannot say, since XCUITest reports the union of its
            // quotes, which starts at AAPL.
            for each in v {
                XCTAssertEqual(each, 0, accuracy: 0.5, "under Reduce Motion the tape still moves: \(v)")
            }
            let aapl = app.buttons.matching(NSPredicate(format: "label BEGINSWITH %@", "Apple, AAPL")).firstMatch
            XCTAssertTrue(aapl.exists)
            let caption = any(app, beginningWith: "Tap a quote")
            XCTAssertEqual(aapl.frame.minX, caption.frame.minX + 24, accuracy: 2,
                           "at rest AAPL should sit one gap inside the clip's leading edge")
        } else {
            for each in v {
                XCTAssertLessThan(each, -1, "with motion on the tape should glide left: \(v)")
            }
        }
    }

    func testGlideMovesRightUnderArabic() throws {
        let app = XCUIApplication()
        openTicker(app, rtl: true)
        toggle(app, "Glide the tape")
        let strip = tape(app)
        XCTAssertTrue(strip.waitForExistence(timeout: 5))
        shot("st-4.42-glide-rtl")
        dump(app, "st-4.42-glide-rtl")
        let width = strip.frame.width
        let v = velocities(app)
        let expected = width / 15
        for each in v {
            XCTAssertGreaterThan(each, 0, "under RTL the tape should glide right: \(v)")
        }
        XCTAssertEqual(mean(v), expected, accuracy: max(8, expected * 0.15), "RTL glide speed: \(v)")
    }
}
