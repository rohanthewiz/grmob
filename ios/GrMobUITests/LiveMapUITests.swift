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

    /// Scrolls until a text with `prefix` is on screen, in either direction.
    /// A lesson is much taller than a phone and the map sits in the middle of
    /// it, so a panel can be either side of where a tap left the scroll.
    private func scrollTo(_ app: XCUIApplication, _ prefix: String) -> Bool {
        let match = app.staticTexts
            .matching(NSPredicate(format: "label BEGINSWITH %@", prefix)).firstMatch
        if match.exists && match.isHittable { return true }
        for _ in 0..<12 {
            app.swipeUp()
            if match.exists && match.isHittable { return true }
        }
        for _ in 0..<24 {
            app.swipeDown()
            if match.exists && match.isHittable { return true }
        }
        return false
    }

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
