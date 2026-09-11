import XCTest

// Simulator pass for core.MapView on MapKit, and the first time this host's map
// has been exercised by anything but a type-check.
//
// Requires the tutorial in the framework, since lesson 4.12 is core.MapView's
// only consumer in the repository:
//
//   ios/build.sh ./examples/tutorial
//   cd ios && xcodegen generate
//   xcodebuild test -project GrMobApp.xcodeproj -scheme GrMobApp \
//     -destination 'platform=iOS Simulator,name=iPhone 17 Pro' \
//     -only-testing:GrMobUITests/LiveMapUITests
//
// and runs alone, for the reason TodoAppUITests does: GrMobUITests drives the
// mobileapp demo bound by the default build.
//
// # What this is for, which ios/verify cannot be
//
// ios/verify type-checks the view layer and replays a transcript, so it proves
// the renderer agrees with Go about the tree. The map's correctness is mostly
// not in the tree: it is in whether MKMapView's own `regionDidChangeAnimated`
// comes back with the region it was handed, and whether the echo guard's
// comparison survives the fact that it does not. That question has no answer
// short of a running MapKit.
//
// The Android host answered it on an emulator and the answer was no: osmdroid
// truncates its Mercator y to an integer pixel, so an exact-equality guard
// matched nothing and every programmatic move was reported back to Go as a
// gesture. MapKit adjusts more than a pixel of scroll — it fits the span to the
// view's aspect ratio and to what the tile pyramid can draw — so it should be
// at least as bad. These assertions are that prediction, made checkable.
//
// # Addressing
//
// By the readouts, not by the map. Lesson 4.12 prints the region Go is asking
// for and the region the map last reported as two separate lines, which is the
// lesson's whole subject and happens to be the only place the echo guard is
// observable from outside. The map itself is reached as `app.maps`, which is
// XCUITest's own MKMapView query.
final class LiveMapUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    /// The lesson list is 49 rows of two lines each, and the natives have no
    /// deep-link intent filter — the tutorial's route channel is a host event
    /// the web page speaks and the shells drop — so reaching a lesson is a
    /// scroll on every device run. This is that scroll, bounded.
    private func openLesson(_ app: XCUIApplication, _ label: String) {
        let row = app.descendants(matching: .any)
            .matching(NSPredicate(format: "label BEGINSWITH %@", label)).firstMatch
        for _ in 0..<40 {
            if row.exists && row.isHittable { break }
            app.swipeUp()
        }
        XCTAssertTrue(row.waitForExistence(timeout: 5),
                      "never reached a row labelled '\(label)' in 40 swipes")
        row.tap()
    }

    /// Whether any static text on screen begins with `prefix`, and its label.
    ///
    /// A prefix rather than an equality, because the readouts carry live
    /// coordinates: what is being asserted is which *sentence* is showing, and
    /// then separately what number it carries.
    private func text(_ app: XCUIApplication, startingWith prefix: String) -> String? {
        let match = app.staticTexts
            .matching(NSPredicate(format: "label BEGINSWITH %@", prefix)).firstMatch
        guard match.waitForExistence(timeout: 5) else { return nil }
        return match.label
    }

    /// One page scroll: a long, deliberate drag rather than `app.swipeUp()`.
    ///
    /// Two things are wrong with the flick, and both of them bit.
    ///
    /// **Reach.** A flick travels about a third of the screen. Lesson 4.12's
    /// location panel sits 2989 points down a window 874 points tall, so
    /// twelve flicks came to within a few points of enough — and whether it
    /// arrived depended on how tall the rows above happened to be that run. An
    /// "opened" badge appearing on a row was the difference between the test
    /// passing alone and failing when it ran second. A drag across three
    /// quarters of the screen covers the same distance in a quarter of the
    /// gestures, with room to spare.
    ///
    /// **Where the gesture starts.** A flick starts at the centre of the
    /// screen, and on these lessons the centre of the screen is sometimes a
    /// MapView — which owns the drag that begins on it and pans instead of
    /// scrolling. On a lesson about the echo guard that is not a slow test,
    /// it is a wrong one: a pan reports a region, and the assertion three
    /// lines down says no region has been reported. So the start point is
    /// chosen to miss every map on screen, and only the start point matters —
    /// UIKit hands the whole gesture to the view under the initial touch, so
    /// the path may cross the map freely.
    private func pageDrag(_ app: XCUIApplication, down: Bool) {
        let window = app.windows.firstMatch.frame
        let maps = app.maps.allElementsBoundByIndex.map { $0.frame }
        func clear(_ dy: CGFloat) -> Bool {
            let y = window.minY + window.height * dy
            return !maps.contains { $0.minY <= y && y <= $0.maxY }
        }
        // Preferred first, then progressively further from the map band. One
        // of five positions spread across the screen is clear of a 260-point
        // map in an 874-point window by arithmetic, so the fallback is a
        // formality rather than a hope.
        let preferred: [CGFloat] = down ? [0.90, 0.80, 0.12, 0.22, 0.50]
                                        : [0.12, 0.22, 0.90, 0.80, 0.50]
        let from = preferred.first(where: clear) ?? preferred[0]
        let to: CGFloat = down ? 0.12 : 0.90
        app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: from))
            .press(forDuration: 0.05,
                   thenDragTo: app.coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: to)))
    }

    /// Scrolls until a text with `prefix` is on screen, in either direction.
    /// A lesson is much taller than a phone and the map sits in the middle of
    /// it, so a panel can be either side of where a tap left the scroll.
    private func scrollTo(_ app: XCUIApplication, _ prefix: String) -> Bool {
        let match = app.staticTexts
            .matching(NSPredicate(format: "label BEGINSWITH %@", prefix)).firstMatch
        if match.exists && match.isHittable { return true }
        for _ in 0..<12 {
            pageDrag(app, down: true)
            if match.exists && match.isHittable { return true }
        }
        for _ in 0..<24 {
            pageDrag(app, down: false)
            if match.exists && match.isHittable { return true }
        }
        return false
    }

    /// Brings `element` **entirely** inside the window, rather than merely
    /// within reach.
    ///
    /// `scrollTo` above stops as soon as a label is *hittable*, which for a
    /// one-line readout can mean as little as its own height showing. That is
    /// enough to read a value and enough to tap a button. It is not enough to
    /// drag a map: `swipeLeft()` starts its gesture at the element's CENTRE,
    /// and a 260-point map scrolled to within 72 points of the top of the
    /// screen has its centre at y = -58 — outside the window, where the
    /// gesture lands on nothing and the map never moves.
    ///
    ///     window  (0, 0, 402, 874)
    ///     map     (46, -188, 310, 260)      centre y = -58   ← off-window
    ///
    /// Whether that happens depends on the screen: a taller simulator leaves
    /// both the map and the readout below it on screen at once and a shorter
    /// one does not, so the assertion this precedes used to pass or fail by
    /// device. Scrolling explicitly is what makes it mean the same thing
    /// everywhere.
    ///
    /// The scroll goes through pageDrag, which picks a start point clear of
    /// every map on screen. That is not a nicety here: the element being
    /// scrolled to IS the map, and a gesture that began on it would pan it —
    /// producing the region report the caller is about to assert the
    /// existence of, without the swipe under test having done anything.
    @discardableResult
    private func scrollFullyIntoView(_ app: XCUIApplication, _ element: XCUIElement,
                                     limit: Int = 12) -> Bool {
        func onScreen() -> Bool {
            let w = app.windows.firstMatch.frame, f = element.frame
            return f.minY >= w.minY && f.maxY <= w.maxY
        }
        for _ in 0..<limit {
            if onScreen() { return true }
            // Above the top of the window: pull the content back down. Below
            // the bottom: push it up. pageDrag keeps the gesture off any map,
            // which matters most here — the element being scrolled to IS the
            // map, and a drag that landed on it would move the map instead of
            // the page and satisfy the caller's assertion by accident.
            pageDrag(app, down: element.frame.minY >= app.windows.firstMatch.frame.minY)
        }
        return onScreen()
    }

    /// # The 20-second wait, which is measured rather than generous
    ///
    /// The tutorial's first screen takes **7.3–7.5 seconds** to appear on this
    /// simulator (iPhone 17 Pro, 402×874), over three cold launches each,
    /// measured from the host: install, launch, screenshot on a fixed cadence,
    /// and take the first frame whose title band matches the settled one.
    ///
    ///     Debug,   grMobBox as a chain of `some View`   17.45  17.68  17.62
    ///     Debug,   grMobBox as GrMobBoxModifier          7.40   7.32   7.52
    ///     Release, grMobBox as GrMobBoxModifier          7.50   7.19   7.29
    ///
    /// Two things are in that table.
    ///
    /// **Ten of the eighteen seconds were the opaque-type tower.** Collapsing
    /// grMobBox's chain into one named ViewModifier (see GrMobBoxModifier,
    /// which did it to stop the Release build crashing the compiler) cut the
    /// launch by 58%. The cost was runtime: a distinct tower type per view
    /// type means generic metadata instantiated per node on the way up.
    ///
    /// **Optimisation buys nothing.** Release and Debug are the same reading
    /// to within the spread of either. So the remaining 7.4s is not slow Swift
    /// that `-O` would tighten; it is SwiftUI building a view per node for a
    /// contents screen of 49 two-line rows, and the only lever left is
    /// building fewer of them. The rest of the launch is already accounted
    /// for and is not the framework:
    ///
    ///     examples/mobileapp, same build and simulator      1.8s
    ///     bridge.renderInitial() (Go, across gomobile)      4ms
    ///     JSON parse + GrMobNode tree, 424603 bytes         6ms
    ///     Go's own render of the same tree (a Go program)   ~1ms
    ///
    /// The wait below stays at 20s rather than tracking the reading: it is a
    /// timeout, and its job is to fail on a hang rather than on a slow host.
    func testEchoGuardOnMapKit() throws {
        let app = XCUIApplication()
        app.launch()

        XCTAssertTrue(app.staticTexts["GrMob Interactive Tutorial"].waitForExistence(timeout: 20),
                      "the tutorial never mounted")
        openLesson(app, "Lesson 4.12")

        XCTAssertTrue(app.maps.firstMatch.waitForExistence(timeout: 10),
                      "no MKMapView on lesson 4.12 — core.MapView rendered as something else")

        // 1. On load, nothing is reported.
        //
        // This is the assertion the Android host failed. `applyRegion` records
        // the opening region in both `applied` and `settled` before it moves
        // the map, and `regionDidChangeAnimated` fires straight afterwards, so
        // a guard that cannot recognise its own instruction reports a region
        // change nobody caused — and makes this sentence unreachable.
        XCTAssertTrue(scrollTo(app, "Nothing reported yet"),
                      "could not find the reported-region readout")
        XCTAssertNotNil(text(app, startingWith: "Nothing reported yet"),
                        "the map reported a region on load: the echo guard did not "
                        + "recognise setRegion's own callback, so OnRegionChange fires for "
                        + "a move the user did not make")
        XCTAssertEqual(text(app, startingWith: "Asking for:")?
                        .hasPrefix("Asking for: 38.7139, -9.1394 at zoom 13.00"), true,
                       "the opening region is not the one the lesson asks for")

        // 2. A programmatic move moves the map and reports nothing.
        //
        // "Show Belém" is Go changing its mind, which is an instruction. The
        // callback MapKit fires from inside setRegion is not a gesture, and the
        // app is entitled to tell the two apart — an app that treats
        // OnRegionChange as a gesture (a "search this area" fetch, an
        // analytics event) otherwise fires it on arrival everywhere it sends
        // itself.
        app.buttons["Show Belém"].tap()
        XCTAssertEqual(text(app, startingWith: "Asking for:")?
                        .hasPrefix("Asking for: 38.6970, -9.2065 at zoom 15.00"), true,
                       "the button did not move the region Go is asking for")
        XCTAssertNotNil(text(app, startingWith: "Nothing reported yet"),
                        "a programmatic move was reported back to Go as a region change — "
                        + "MapKit returned a region this renderer did not hand it and the "
                        + "comparison did not absorb the difference")

        // 3. A real gesture IS reported, which is the other half: a tolerance
        // wide enough to hide a drag would be a guard that never fires.
        //
        // Steps 1 and 2 scrolled down to the readout and the button, which on
        // this screen sits the map partly above the top of the window — and a
        // swipe aimed at an element whose centre is off-window moves nothing.
        // See scrollFullyIntoView for the measurement and for why the failure
        // it caused depended on which simulator ran the test.
        XCTAssertTrue(scrollFullyIntoView(app, app.maps.firstMatch),
                      "could not get the whole map on screen, so the swipe below would "
                      + "gesture at a point outside the window and prove nothing")
        app.maps.firstMatch.swipeLeft()
        let reported = app.staticTexts
            .matching(NSPredicate(format: "label BEGINSWITH 'Reported:'")).firstMatch
        XCTAssertTrue(reported.waitForExistence(timeout: 10),
                      "a swipe across the map reported nothing — either the gesture did not "
                      + "reach MapKit or the echo guard's tolerance is swallowing real "
                      + "movement")

        // 4. And a patch that is not about the region leaves the map alone.
        //
        // Dropping a pin re-renders the MapView node with Go's unchanged
        // region. That is the exact patch the guard is named for: without it
        // the map snaps back to where Go last said, out from under the finger.
        let afterPan = reported.label
        app.maps.firstMatch.tap()
        XCTAssertTrue(app.staticTexts
            .matching(NSPredicate(format: "label CONTAINS 'pin-1 selected'"))
            .firstMatch.waitForExistence(timeout: 10),
                      "a tap on the map did not drop a pin, so the no-snap-back assertion "
                      + "below has nothing to prove")
        XCTAssertEqual(reported.label, afterPan,
                       "dropping a pin changed the reported region: the unrelated patch "
                       + "moved the map, which is the failure the two-memory guard exists "
                       + "to prevent")
    }

    /// The location panel, whose two hooks had no consumer anywhere in the
    /// repository until lesson 4.12 grew one.
    ///
    /// The simulator has no position unless one is set, and the authorization
    /// starts undetermined — which is the interesting state, because
    /// CLLocationManager prompts from the sensor itself on iOS and
    /// startUpdatingLocation on an undetermined authorization reports nothing
    /// at all, forever. So what is pinned is that the panel says which state it
    /// is in rather than drawing a spinner, in every state it can be in.
    func testLocationPanelSaysWhichStateItIsIn() throws {
        let app = XCUIApplication()
        app.launch()
        XCTAssertTrue(app.staticTexts["GrMob Interactive Tutorial"].waitForExistence(timeout: 20))
        openLesson(app, "Lesson 4.12")

        XCTAssertTrue(scrollTo(app, "This panel runs the code above"),
                      "the location panel is not on lesson 4.12")

        // One of the four sentences, and never the empty screen. Which one
        // depends on what the simulator's authorization dialog did, so the
        // assertion is that the panel committed to one of them.
        let sentences = ["No position, and none on the way",
                         "Waiting for the first fix",
                         "No fix:",
                         ""]
        let said = sentences.contains { prefix in
            !prefix.isEmpty && app.staticTexts
                .matching(NSPredicate(format: "label BEGINSWITH %@", prefix))
                .firstMatch.exists
        } || app.staticTexts
            .matching(NSPredicate(format: "label CONTAINS 'accurate to about'"))
            .firstMatch.exists
        XCTAssertTrue(said, "the position readout says none of its four sentences, so a "
                      + "reader is shown a blank where the state should be")

        // "Centre the map on me" is present in every state and disabled
        // without a fix: a tap with no position would set the region to 0,0,
        // which is a real place in the Gulf of Guinea and a map that looks like
        // it worked.
        let centre = app.buttons["Centre the map on me"]
        XCTAssertTrue(centre.exists,
                      "the centre-on-me button is missing, so the codeBlock above it shows "
                      + "a line the panel does not run")
        let hasFix = app.staticTexts
            .matching(NSPredicate(format: "label CONTAINS 'accurate to about'"))
            .firstMatch.exists
        if !hasFix {
            XCTAssertFalse(centre.isEnabled,
                           "the centre-on-me button is live with no fix in hand")
        }
    }
}
