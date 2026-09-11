import SwiftUI

/// The iOS half of core.MapView: MapKit, wrapped so the declarative tree can
/// hold an imperative map.
///
///     core.MapView(region, core.Marker("hall", lat, lng, "The hall"))
///       ──▶ MKMapView, with one MKAnnotation per Marker child
///
/// # Why MKMapView and not SwiftUI's Map
///
/// SwiftUI's `Map` is the shorter spelling and it cannot do three of the four
/// things this node promises. It has no tap-location API at all (so
/// core.OnMapTap would be unimplementable), its built-in `MapMarker` is not
/// tappable (so core.OnMarkerTap would need custom annotation content anyway),
/// and its region binding writes back on every frame of a drag, which is the
/// opposite of the throttle core.OnRegionChange documents.
///
/// `MKMapView` answers all three directly: `convert(_:toCoordinateFrom:)` for
/// the tap, `didSelect` for the annotation, and `regionDidChangeAnimated` for
/// the gesture's end. It is also the view the other two hosts are shaped like —
/// osmdroid on Android and Leaflet on the web are both imperative map objects
/// reconciled against the node — so all three renderers read alike.
///
/// # Zoom is not a span, and this is where the two meet
///
/// Every engine in this framework speaks the slippy-tile zoom level. MapKit
/// speaks `MKCoordinateSpan`, in degrees. The bridge between them is Web
/// Mercator's own definition — 256 × 2^zoom pixels around the equator — so a
/// view `w` points wide shows
///
///     longitudeDelta = 360 × w / (256 × 2^zoom)
///
/// and the inverse gives the zoom back. The width is read off the live view, so
/// the conversion is exact rather than assumed: a map at zoom 14 on an iPhone
/// shows the same ground as the same zoom in Leaflet on the web.
///
/// The latitude delta is derived from the longitude one by the view's aspect
/// ratio and the cosine of the latitude, which is Mercator's compression. That
/// half is an approximation — the projection is not linear over a tall span —
/// and it is the right one: MapKit re-derives the region it actually shows
/// anyway, and what matters is that the zoom round-trips, which is a question
/// about longitude alone.
///
/// # The echo guard
///
/// The coordinator remembers the region it last handed to MapKit and compares.
/// Go moving the map is an instruction; Go merely re-rendering is not. Without
/// it, any unrelated re-render would put the map back where Go last said — under
/// the user's finger mid-drag. core.MapView's "The map is controlled, with the
/// echo guard a drag needs" is the contract; the WASM runtime implements the
/// same comparison, and the same comparison suppresses the report MapKit
/// generates from this renderer's own `setRegion`.
///
/// That last part needs a tolerance here, and it is the one place this host
/// diverges from the other two. `setRegion` does not keep the region it is
/// given — MapKit fits the span to the view and to what it can draw — so the
/// comparison is exact in the one direction where both numbers came from Go and
/// pixel-tolerant in the others. See GrMobRegion.
#if canImport(MapKit) && canImport(UIKit)

import MapKit
import UIKit

struct GrMobMapView: View {
    let node: GrMobNode
    let grow: GrMobGrow
    @Environment(\.grMobRuntime) private var runtime

    var body: some View {
        // GeometryReader is not decoration: the zoom/span conversion needs the
        // view's width, and reading it here means the map is sized by the Go
        // style (through grMobBox) and the conversion uses whatever that
        // produced, rather than assuming a screen.
        GeometryReader { geo in
            GrMobMapRepresentable(node: node, runtime: runtime, size: geo.size)
        }
        .grMobBox(node.style, grow: grow)
    }
}

/// The marker annotation, which carries the Go id so a selection can be
/// reported as the thing the app has an index of rather than as a coordinate.
/// See core.OnMarkerTap on why the id and not the position.
private final class GrMobMarkerAnnotation: NSObject, MKAnnotation {
    let markerID: String
    // Must be KVO-observable for MapKit to move the pin when it changes, which
    // is what `@objc dynamic` buys. Reassigning it is how a marker that moved
    // is moved rather than removed and re-added — the whole reason core.Marker
    // is a keyed child node.
    @objc dynamic var coordinate: CLLocationCoordinate2D
    var title: String?

    init(markerID: String, coordinate: CLLocationCoordinate2D, title: String?) {
        self.markerID = markerID
        self.coordinate = coordinate
        self.title = title
    }
}

private struct GrMobMapRepresentable: UIViewRepresentable {
    let node: GrMobNode
    let runtime: GrMobRuntime?
    let size: CGSize

    func makeCoordinator() -> Coordinator { Coordinator() }

    func makeUIView(context: Context) -> MKMapView {
        let map = MKMapView()
        map.delegate = context.coordinator
        context.coordinator.runtime = runtime
        context.coordinator.node = node

        // The tap recognizer for core.OnMapTap. MapKit's own gestures keep
        // working beside it because this one does not cancel them, and the
        // delegate below refuses a touch that landed on an annotation — a tap
        // on a pin is OnMarkerTap's event, and a host that sent both would
        // make every marker tap also drop a pin.
        let tap = UITapGestureRecognizer(
            target: context.coordinator,
            action: #selector(Coordinator.handleTap(_:)))
        tap.delegate = context.coordinator
        map.addGestureRecognizer(tap)
        context.coordinator.map = map
        return map
    }

    func updateUIView(_ map: MKMapView, context: Context) {
        let c = context.coordinator
        // Re-pointed on every update, because the node instance is what the
        // reconciler replaces and the callback IDs are positional: a
        // coordinator holding last pass's node would dispatch last pass's IDs.
        c.node = node
        c.runtime = runtime
        c.viewSize = size

        c.applyRegion(to: map, node: node, width: size.width, height: size.height)
        c.applyMarkers(to: map, node: node)
        // The platform's blue dot, with the platform's own accuracy halo. It
        // needs the location authorization and reports nothing when it has not
        // been granted: a map is still a map. See core.ShowUserLocation.
        map.showsUserLocation = node.boolProp("showUser")
    }

    final class Coordinator: NSObject, MKMapViewDelegate, UIGestureRecognizerDelegate {
        var runtime: GrMobRuntime?
        var node: GrMobNode?
        weak var map: MKMapView?
        var viewSize: CGSize = .zero

        /// The region Go last asked for, which is also the region this
        /// renderer last handed to MapKit. Written only on the apply path. nil
        /// until the first apply, so the first region is always applied.
        private var applied: GrMobRegion?
        /// Where the map last came to rest, and so also the last thing told to
        /// Go. Written only on the report path.
        ///
        /// Two memories rather than one, because they are two facts. They are
        /// equal for as long as nobody touches the map and they diverge the
        /// moment somebody does:
        ///
        ///     apply    want != applied   Go changed its mind — an instruction
        ///              want == settled   the map is already there — Go echoing
        ///                                the pan back, which must not become a
        ///                                setRegion landing a frame late
        ///     report   next != applied   not the callback our own setRegion
        ///                                caused
        ///              next != settled   not a second event for a map that has
        ///                                not moved since the last one
        ///
        /// These were one slot in all three hosts, and a browser session is
        /// what caught it: a pan wrote the user's region into `applied`, so
        /// Go's UNCHANGED region then read as a change and the next patch to
        /// reach the map — a dropped pin, a marker moving, any unrelated
        /// re-render — snapped the map back to where Go last said. That is the
        /// exact failure the guard is named for. The long form is in the web
        /// host, under "Two memories, because they are two facts".
        private var settled: GrMobRegion?
        /// Set while `setRegion` is running, and read by the delegate callback
        /// MapKit fires from inside it. The comparison below would catch that
        /// case on its own; this makes the synchronous one free rather than a
        /// round trip through the throttle.
        private var applying = false
        private var reportWork: DispatchWorkItem?

        /// Annotations by the marker id, so a marker that moved is moved.
        private var markers: [String: GrMobMarkerAnnotation] = [:]
        /// The unnamed markers, in child order. core.Marker allows an empty id
        /// — the single "you are here" pin — and those cannot be kept in a
        /// dictionary, so they are matched positionally, which is the only
        /// identity they have.
        private var anonymous: [GrMobMarkerAnnotation] = []

        // MARK: - Region

        func applyRegion(to map: MKMapView, node: GrMobNode, width: CGFloat, height: CGFloat) {
            guard width > 0, height > 0 else { return }
            let want = GrMobRegion(
                lat: node.doubleProp("lat"),
                lng: node.doubleProp("lng"),
                zoom: node.doubleProp("zoom"))
            // Go has not changed its mind. Nothing to do — and emphatically
            // not a reason to re-centre: the map may be somewhere else
            // entirely because the user put it there, and this is the patch
            // that would yank it back.
            if let applied, applied.isSame(as: want) { return }
            applied = want
            // Go HAS changed its mind, and to where the map already is: the app
            // echoed OnRegionChange into its own state and this is that value
            // arriving a frame later. Recorded above, because Go is now asking
            // for this region and the next instruction is measured against it —
            // but not applied, because applying it is the round trip an echoing
            // app would otherwise fight.
            // Tolerant, because `settled` holds a MapKit-derived region and
            // `want` is what the app echoed out of OnRegionChange — possibly
            // rounded on the way through its own state. See
            // GrMobRegion.isSamePlace.
            if let settled, settled.isSamePlace(as: want, width: width) { return }

            let span = grMobSpan(zoom: want.zoom, width: width, height: height, lat: want.lat)
            let region = MKCoordinateRegion(
                center: CLLocationCoordinate2D(latitude: want.lat, longitude: want.lng),
                span: span)
            applying = true
            // Unanimated, for the reason the web host gives: a map an app is
            // driving from its own state (following a location, stepping
            // through a list) queues animations behind each other and lags the
            // data it is showing.
            map.setRegion(region, animated: false)
            applying = false
            // The map now rests here, so `settled` says so. Without this the
            // slot holds wherever the user last left it, and a later
            // instruction back to that place would be skipped as "already
            // there" while the map sat somewhere else — the invariant is that
            // `settled` is where the map is, however it got there.
            settled = want
        }

        func mapView(_ map: MKMapView, regionDidChangeAnimated animated: Bool) {
            if applying { return }
            guard viewSize.width > 0 else { return }
            // Throttled, because a pan generates a stream of these and each one
            // that crossed the bridge would be a full Go render pass. The
            // delay coalesces a gesture that ends as a pan and a zoom a few
            // milliseconds apart — the same job the web host's quiet timer
            // does, for the same reason.
            reportWork?.cancel()
            let work = DispatchWorkItem { [weak self, weak map] in
                guard let self, let map else { return }
                self.reportRegion(of: map)
            }
            reportWork = work
            DispatchQueue.main.asyncAfter(
                deadline: .now() + GrMobMapRegionQuiet, execute: work)
        }

        private func reportRegion(of map: MKMapView) {
            let region = map.region
            let next = GrMobRegion(
                lat: region.center.latitude,
                lng: region.center.longitude,
                zoom: grMobZoom(longitudeDelta: region.span.longitudeDelta,
                                width: viewSize.width))
            // The echo guard, this direction: the map is sitting where this
            // renderer put it, so there is nothing to tell Go. That covers the
            // callback setRegion itself causes and a gesture that ends where it
            // started.
            //
            // Tolerant, not exact: `next` is derived from the region MapKit
            // settled on, which is never the region it was handed — it fits
            // the span to the view and to the tile pyramid, and the zoom is
            // recovered through log2 on top of that. An exact comparison here
            // reports every programmatic move to Go as a gesture. See
            // GrMobRegion.isSamePlace.
            if let applied, applied.isSamePlace(as: next, width: viewSize.width) { return }
            // The second half of the comparison, and the reason the user's
            // region does NOT go into `applied`: a map that fires two events
            // without moving between them has one thing to say, and Go's own
            // region is a separate fact a pan must not overwrite. See `settled`.
            if let settled, settled.isSamePlace(as: next, width: viewSize.width) { return }
            // Recorded even with no handler attached, so a map that gains one
            // later does not immediately report a pan nobody was listening for.
            settled = next
            let cb = node?.stringProp("onRegionChange") ?? ""
            guard !cb.isEmpty else { return }
            // The wire form core.ParseRegion reads. String(Double) is
            // locale-independent in Swift, which is what keeps a decimal comma
            // out of a comma-separated payload.
            runtime?.textChanged(cb, "\(next.lat),\(next.lng),\(next.zoom)")
        }

        // MARK: - Markers

        func applyMarkers(to map: MKMapView, node: GrMobNode) {
            var seen: Set<String> = []
            var anonIndex = 0
            var nextAnonymous: [GrMobMarkerAnnotation] = []

            for child in node.children where child.type == "Marker" {
                let id = child.stringProp("id")
                let coordinate = CLLocationCoordinate2D(
                    latitude: child.doubleProp("lat"),
                    longitude: child.doubleProp("lng"))
                let title = child.stringProp("title")

                if id.isEmpty {
                    // Positional identity; see `anonymous`.
                    let existing = anonIndex < anonymous.count ? anonymous[anonIndex] : nil
                    anonIndex += 1
                    nextAnonymous.append(
                        place(existing, on: map, id: id, at: coordinate, title: title))
                    continue
                }
                seen.insert(id)
                markers[id] = place(markers[id], on: map, id: id, at: coordinate, title: title)
            }

            // Whatever the node no longer has a child for. Removed by id rather
            // than by clearing the layer, which is the difference between a
            // patch and a rebuild: MapKit drops a selected callout and replays
            // its drop animation for every annotation it is handed afresh.
            for (id, annotation) in markers where !seen.contains(id) {
                map.removeAnnotation(annotation)
                markers.removeValue(forKey: id)
            }
            for extra in anonymous.dropFirst(nextAnonymous.count) {
                map.removeAnnotation(extra)
            }
            anonymous = nextAnonymous
        }

        private func place(
            _ existing: GrMobMarkerAnnotation?, on map: MKMapView,
            id: String, at coordinate: CLLocationCoordinate2D, title: String
        ) -> GrMobMarkerAnnotation {
            if let existing {
                // Guarded: reassigning the coordinate is a KVO change MapKit
                // animates, so a pin that has not moved must not be told where
                // it is.
                if existing.coordinate.latitude != coordinate.latitude
                    || existing.coordinate.longitude != coordinate.longitude {
                    existing.coordinate = coordinate
                }
                existing.title = title.isEmpty ? nil : title
                return existing
            }
            let annotation = GrMobMarkerAnnotation(
                markerID: id, coordinate: coordinate,
                title: title.isEmpty ? nil : title)
            map.addAnnotation(annotation)
            return annotation
        }

        func mapView(_ map: MKMapView, didSelect view: MKAnnotationView) {
            guard let annotation = view.annotation as? GrMobMarkerAnnotation else { return }
            let cb = node?.stringProp("onMarkerTap") ?? ""
            if !cb.isEmpty {
                runtime?.textChanged(cb, annotation.markerID)
            }
            // Deselected immediately so the same pin can be tapped twice.
            // MapKit keeps a selection until something else is chosen, and a
            // second tap on an already-selected annotation reports nothing —
            // which from the app's side looks like a marker that works once.
            map.deselectAnnotation(annotation, animated: false)
        }

        // MARK: - Map taps

        @objc func handleTap(_ gesture: UITapGestureRecognizer) {
            guard let map, gesture.state == .ended else { return }
            let cb = node?.stringProp("onMapTap") ?? ""
            guard !cb.isEmpty else { return }
            let point = gesture.location(in: map)
            let at = map.convert(point, toCoordinateFrom: map)
            runtime?.textChanged(cb, "\(at.latitude),\(at.longitude)")
        }

        func gestureRecognizer(
            _ gestureRecognizer: UIGestureRecognizer,
            shouldReceive touch: UITouch
        ) -> Bool {
            // A touch that landed on a pin belongs to OnMarkerTap. Walked up
            // the view tree rather than tested directly, because an
            // MKAnnotationView's callout and its image are subviews and a tap
            // lands on whichever one is on top.
            var view = touch.view
            while let current = view {
                if current is MKAnnotationView { return false }
                view = current.superview
            }
            return true
        }

        func gestureRecognizer(
            _ gestureRecognizer: UIGestureRecognizer,
            shouldRecognizeSimultaneouslyWith other: UIGestureRecognizer
        ) -> Bool {
            // MapKit's own recognizers must keep working: this one only
            // observes, and refusing to share would break panning.
            true
        }
    }
}

/// How long a gesture has to be quiet before its region is reported, in
/// seconds. See Coordinator.mapView(_:regionDidChangeAnimated:), and the web
/// host's MAP_REGION_QUIET_MS, which is the same number for the same reason.
private let GrMobMapRegionQuiet: TimeInterval = 0.12

/// The slippy zoom level -> MapKit's span. See the type comment for the
/// arithmetic and for which half of it is exact.
private func grMobSpan(zoom: Double, width: CGFloat, height: CGFloat, lat: Double) -> MKCoordinateSpan {
    let w = max(Double(width), 1)
    let h = max(Double(height), 1)
    let lonDelta = 360 * w / (256 * pow(2, zoom))
    // Mercator compresses a degree of latitude by the cosine of the latitude,
    // so the same number of pixels covers fewer degrees away from the equator.
    let latDelta = lonDelta * (h / w) * cos(lat * .pi / 180)
    return MKCoordinateSpan(
        latitudeDelta: min(max(latDelta, 1e-8), 180),
        longitudeDelta: min(max(lonDelta, 1e-8), 360))
}

/// MapKit's span -> the slippy zoom level: the inverse of the longitude half
/// above, which is the half the zoom level is defined by.
private func grMobZoom(longitudeDelta: Double, width: CGFloat) -> Double {
    let w = max(Double(width), 1)
    guard longitudeDelta > 0 else { return 0 }
    return log2(360 * w / (256 * longitudeDelta))
}

#else

/// The non-iOS build. Renderer.swift's dispatch has one arm for this node type
/// on every platform, so the type has to exist on every platform — and the
/// macOS typecheck that runs in ios/verify compiles these files without UIKit.
///
/// A box and no map, which is also the honest rendering: MapKit on macOS is an
/// NSViewRepresentable with different gestures and a different user, and
/// inventing one to satisfy a typecheck would be shipping an unexercised map.
struct GrMobMapView: View {
    let node: GrMobNode
    let grow: GrMobGrow

    var body: some View {
        Color.clear.grMobBox(node.style, grow: grow)
    }
}

#endif

/// How many of the map's own pixels two regions may differ by and still be the
/// same place. See GrMobRegion for the measurement behind the number and for
/// why MapKit's own share of the error is a separate question.
private let GrMobMapTolerancePx: Double = 1.5

/// A region as this renderer compares them: the three numbers, and the two
/// comparisons the echo guard needs — one exact and one to the nearest pixel.
///
/// # Why there are two
///
/// `isSame` is the comparison between a number Go sent and a number this
/// renderer stored from the same source. Those are bit-identical when nothing
/// changed, so it is exact, and it has to stay exact: an epsilon there would
/// make a deliberate small nudge from Go — following a location, stepping a
/// marker along a path — into a no-op that never reaches the map.
///
/// `isSamePlace` is the comparison where one side came back out of MapKit, and
/// that side is never the number that went in. `setRegion` does not store the
/// region it is given: it fits the span to the view's aspect ratio and to the
/// tile pyramid it can actually draw, and `regionDidChangeAnimated` reports
/// what it settled on. This renderer then derives a zoom back out of that span
/// through `log2`, which loses a little more. So the exact comparison can
/// never match on the report path, and every programmatic move would be
/// reported to Go as if the user had made it.
///
/// That is the same bug an Android emulator run found on osmdroid, whose cause
/// is different and whose shape is identical: osmdroid quantises the centre to
/// integer pixels, MapKit re-derives the whole region, and in both cases the
/// echo guard is comparing a question with an answer. GrMobMapView.kt's
/// samePlaceOnScreen carries the long form, including why the browser is the
/// one host that does not need this — Leaflet caches the centre it was given
/// and hands it straight back.
///
/// # In pixels, not in degrees
///
/// The finest move a map can represent is one pixel, so the tolerance is a
/// number of pixels converted to degrees at the zoom in question. Web
/// Mercator's own definition gives the conversion — 256 × 2^zoom pixels span
/// 360° of longitude — and the latitude tolerance is that scaled by the cosine
/// of the latitude, which is the same Mercator compression `grMobSpan` above
/// applies for the same reason.
///
/// A fixed epsilon in degrees could not work: one pixel is 1.7e-4° at zoom 12
/// and 1.3e-6° at zoom 19, a factor of 128.
///
/// # How wide, and the experiment that settled it here
///
/// `GrMobMapTolerancePx` is 1.5, which is the number the Android host arrived
/// at by measurement: osmdroid truncates its Mercator y to an integer pixel and
/// returned a centre 0.97 pixels out, so the bound is one pixel and the rest is
/// headroom. It is affordable because no gesture is that small — a drag has to
/// clear the platform's touch slop before it is a drag.
///
/// MapKit's error is a different quantity, so whether the same number served
/// was a real question. It was answered by running it both ways against a
/// simulator, with GrMobUITests/LiveMapUITests and one line changed:
///
///     isSamePlace   lesson 4.12 shows "Nothing reported yet" on load,
///                   "Show Belém" moves the map and reports nothing,
///                   a swipe reports, a dropped pin does not move the map
///     isSame        the readout never says "Nothing reported yet" at all —
///                   the map reports a region change on load, before anybody
///                   has touched it
///
/// So MapKit has the bug too, the tolerance is what fixes it, and 1.5 pixels is
/// enough. Worth knowing *why* the existing `applying` flag did not already
/// cover it: that flag is set across the `setRegion` call and catches the
/// callback MapKit fires synchronously from inside it, but MapKit fires
/// `regionDidChangeAnimated` again after the next layout pass, outside any
/// flag a caller can hold. The comparison is the only thing standing there.
///
/// # Zoom, which is where iOS differs from Android
///
/// Android compares zooms with a small constant, because osmdroid speaks the
/// slippy zoom level natively and round-trips it exactly. Here the zoom is a
/// derived quantity on both sides of the conversion, and MapKit is free to
/// settle on a span this code never asked for — so the tolerance is derived
/// from the viewport too: changing the zoom by δ scales the view by 2^δ, which
/// moves the edge of a `width`-point map by (width/2)(2^δ − 1) points. Solving
/// that for half a point gives
///
///     δ = log2(1 + 1 / width)
///
/// which is 0.0037 on a 390-point iPhone — a tolerance that means "the map is
/// showing the same pixels", which is the only thing the echo guard is asking.
struct GrMobRegion {
    let lat: Double
    let lng: Double
    let zoom: Double

    /// Exact. For the apply path's "has Go changed its mind" — see the type
    /// comment for why a tolerance there would be a bug.
    func isSame(as other: GrMobRegion) -> Bool {
        lat == other.lat && lng == other.lng && zoom == other.zoom
    }

    /// Whether these two regions put the same pixels on screen, to within half
    /// of one. For every comparison with a MapKit-derived region on either side.
    ///
    /// `width` is the live view's width in points, which is why this takes a
    /// parameter at all: the zoom tolerance is a property of the viewport and
    /// not of the region. A non-positive width — a map that has not been laid
    /// out — falls back to exact equality rather than inventing a screen, and
    /// the callers guard on it anyway.
    func isSamePlace(as other: GrMobRegion, width: CGFloat) -> Bool {
        guard width > 0 else { return isSame(as: other) }
        // Either zoom will do for the pixel size once they are within the zoom
        // tolerance; the map's own is used, because the map's pixels are what
        // is being measured.
        let zoomTolerance = log2(1 + 1 / Double(width))
        guard abs(zoom - other.zoom) <= zoomTolerance else { return false }
        let toleranceLng = 360 / (256 * pow(2, other.zoom)) * GrMobMapTolerancePx
        guard abs(lng - other.lng) <= toleranceLng else { return false }
        // ±85° is Web Mercator's own limit and is what keeps the cosine off
        // zero at the pole, which would collapse this back to exact equality.
        let clampedLat = min(max(other.lat, -85), 85)
        let toleranceLat = toleranceLng * cos(clampedLat * .pi / 180)
        return abs(lat - other.lat) <= toleranceLat
    }
}
