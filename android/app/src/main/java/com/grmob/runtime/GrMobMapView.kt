package com.grmob.runtime

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.remember
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import kotlin.math.abs
import kotlin.math.cos
import kotlin.math.pow
import org.osmdroid.config.Configuration
import org.osmdroid.events.DelayedMapListener
import org.osmdroid.events.MapListener
import org.osmdroid.events.ScrollEvent
import org.osmdroid.events.ZoomEvent
import org.osmdroid.tileprovider.tilesource.TileSourceFactory
import org.osmdroid.util.GeoPoint
import org.osmdroid.views.MapView
import org.osmdroid.views.overlay.MapEventsOverlay
import org.osmdroid.events.MapEventsReceiver
import org.osmdroid.views.overlay.Marker
import org.osmdroid.views.overlay.mylocation.GpsMyLocationProvider
import org.osmdroid.views.overlay.mylocation.MyLocationNewOverlay

/**
 * The Android half of core.MapView: osmdroid, wrapped so the declarative tree
 * can hold an imperative map.
 *
 *     core.MapView(region, core.Marker("hall", lat, lng, "The hall"))
 *       ──▶ org.osmdroid.views.MapView, with one Marker overlay per child
 *
 * # Why osmdroid and not Google Maps
 *
 * No API key and no Play Services. A key is a deployment secret every app
 * adopting this framework would have to obtain before a map drew anything at
 * all, and Play Services is absent on a real share of Android devices — so the
 * keyless option is the one that makes the node usable on arrival. Google Maps
 * Compose is a better map on the devices that have both, and it is a second
 * provider this file could grow rather than a reason to start there.
 *
 * It also speaks the framework's own units. osmdroid's zoom is the slippy-tile
 * zoom level, as a double, which is exactly what core.Region carries — so this
 * is the one host with no projection arithmetic in it. (MapKit speaks degrees of
 * span; see GrMobMapView.swift for that conversion and why it is needed.)
 *
 * # AndroidView, and the lifecycle an imperative map needs
 *
 * osmdroid's MapView wants onResume/onPause — it starts and stops its tile
 * downloader from them — and leaks its tile thread pool without onDetach. None
 * of that is Compose's business, so the DisposableEffect below owns it.
 *
 * # The echo guard
 *
 * The holder remembers the region it last handed to osmdroid and compares. Go
 * moving the map is an instruction; Go merely re-rendering is not. Without it
 * any unrelated recomposition would put the map back where Go last said —
 * under the user's finger mid-drag. core.MapView's "The map is controlled, with
 * the echo guard a drag needs" is the contract; the other two hosts implement
 * the same comparison, and in all three the same comparison is what keeps the
 * map's own `setCenter` from being reported back to Go as a gesture.
 *
 * The comparison is exact in the one direction where both numbers came from Go,
 * and tolerant to half a pixel in the three where one of them came back out of
 * osmdroid — which cannot return the doubles it was given. samePlaceOnScreen
 * carries that argument, and it is the difference between the guard working and
 * the guard reporting every programmatic move as a gesture.
 *
 * # Tiles
 *
 * MAPNIK is OpenStreetMap's own tile service, which has a usage policy: the
 * user agent below is what identifies this app to it, and a build that ships to
 * a real user base should point this at a provider it pays for. The user agent
 * is not optional — osmdroid's default is its own package name, and OSM returns
 * 418 to it.
 */
@Composable
internal fun GrMobMapView(node: GrMobNode, extra: Modifier) {
    val runtime = LocalGrMobRuntime.current
    val context = LocalContext.current

    // Per-map state that has to survive recomposition: the echo guard's memory
    // and the marker overlays by id. A remembered holder rather than several
    // mutableStateOf, because nothing here should recompose anything — every
    // field is read and written by the imperative map below.
    val holder = remember { GrMobMapHolder() }

    AndroidView(
        // Margin and size, not the fill: osmdroid draws its own surface, exactly
        // as the Material controls do. Shared with Renderer.kt rather than
        // restated, which is why that function is internal.
        //
        // Then clipped, which is not decoration and is not shared. See
        // mapClip: osmdroid paints past its own bounds, and an emulator run
        // found it painting over the captions laid out beneath it.
        modifier = marginAndSize(node.style, extra).then(mapClip(node.style)),
        factory = { ctx -> createOsmMap(ctx, holder) },
        update = { map ->
            // Re-read on every update, because the callback IDs are positional:
            // a listener holding last pass's node would dispatch last pass's IDs
            // to whatever happens to own them now.
            holder.node = node
            holder.report = { cb, value -> runtime.textChanged(cb, value) }

            applyMapRegion(map, holder, node)
            applyMapMarkers(map, holder, node)
            applyMapUser(map, holder, context, node.boolProp("showUser"))
            map.invalidate()
        },
    )

    DisposableEffect(Unit) {
        onDispose {
            holder.map?.let { map ->
                map.onPause()
                // Releases the tile thread pool and the overlays' own
                // resources. Without it a screen that has shown a map leaks one
                // downloader per visit.
                map.onDetach()
            }
            holder.userOverlay?.disableMyLocation()
            holder.map = null
        }
    }
}

/**
 * The clip that stops osmdroid's tiles at the edge of the node's layout box.
 *
 * # What this fixes
 *
 * A run on a real emulator (Android 16, density 2.625) found lesson 4.12's map
 * painting well outside the slot the layout gave it. Measured against the
 * platform's own `debug.layout` bounds, which draws every View's rectangle:
 *
 *                      layout slot          painted
 *       x            121 .. 959           87 .. 992
 *       y            918 .. 1601         703 .. 1726
 *                    683px = 260dp       1023px = 389dp
 *
 * 683px is exactly the `core.Height("260px")` the lesson asks for, so the View
 * was measured correctly and painted past itself — 215px above the slot and
 * 125px below, over both readout captions laid out beneath. osmdroid
 * over-draws deliberately, so its tiles are ready before a scroll reaches
 * them; that is a good trade inside a clip and a bug without one.
 *
 * Compose does not clip a child's drawing unless a modifier says to.
 * `Modifier.width`/`height` set the layout size and nothing else, which is the
 * same fixed-size fact docs/platforms/native.md's census states from the other
 * side, and so `marginAndSize` alone gives a correctly sized View with no
 * bound on where it paints.
 *
 * # Why here and not in marginAndSize
 *
 * `marginAndSize` is also the Button, Checkbox, Switch, Slider and Progress
 * chain, and those are material3 composables that draw their elevation shadow
 * *inside* themselves — after this modifier chain has run. A `clipToBounds`
 * there would flatten every raised control in the vocabulary to fix one node
 * that wraps a View with an over-draw policy of its own. The over-draw is
 * osmdroid's, so the clip is osmdroid's.
 *
 * # The rounded case
 *
 * A style's corner radius is honored, because the web host already honors it:
 * leaflet.css puts `overflow: hidden` on Leaflet's container, so a MapView with
 * a border radius has round corners in the browser and would have square ones
 * here. `Modifier.clip(shape)` clips to the shape and to the bounds together,
 * so the two cases are one decision with two shapes rather than two clips.
 *
 * `marginAndSize` strips the radius from the chain it builds (the node draws
 * its own box), which is why the radius is read off the style again here rather
 * than left to it.
 */
private fun mapClip(style: GrMobStyle?): Modifier {
    val radius = style?.borderRadius ?: 0f
    if (radius > 0f) return Modifier.clip(RoundedCornerShape(radius.dp))
    return Modifier.clipToBounds()
}

/**
 * The mutable half of a map: what this renderer has told osmdroid, and the
 * listeners' way back to Go.
 *
 * A plain class rather than Compose state on purpose — see the remember above.
 */
private class GrMobMapHolder {
    var map: MapView? = null
    var node: GrMobNode? = null
    var report: ((String, String) -> Unit)? = null

    /**
     * The region GO last asked for, which is also the region this renderer last
     * handed to osmdroid. Written only on the apply path. NaN until the first
     * apply, which is what makes the first region always an instruction: NaN
     * equals nothing, including itself.
     */
    var appliedLat = Double.NaN
    var appliedLng = Double.NaN
    var appliedZoom = Double.NaN

    /**
     * Where the MAP last came to rest, and so also the last thing told to Go.
     * Written only on the report path.
     *
     * Two memories rather than one, because they are two facts. They are equal
     * for as long as nobody touches the map and they diverge the moment
     * somebody does:
     *
     *     apply    want != applied   Go changed its mind — an instruction
     *              want == settled   the map is already there — Go echoing the
     *                                pan back, which must not become a
     *                                setCenter landing a frame late
     *     report   next != applied   not the callback our own setCenter caused
     *              next != settled   not a second event for a map that has not
     *                                moved since the last one
     *
     * These were one slot in all three hosts, and a browser session is what
     * caught it: a pan wrote the user's region into `applied`, so Go's
     * UNCHANGED region then read as a change and the next patch to reach the
     * map — a dropped pin, a marker moving, any unrelated re-render — snapped
     * the map back to where Go last said. That is the exact failure the guard
     * is named for. See the web host's "Two memories, because they are two
     * facts", which carries the long form.
     */
    var settledLat = Double.NaN
    var settledLng = Double.NaN
    var settledZoom = Double.NaN

    /** Markers by their Go id. */
    val markers = HashMap<String, Marker>()

    /**
     * The unnamed markers, in child order. core.Marker allows an empty id — the
     * single "you are here" pin — and those have no identity but their position
     * in the list.
     */
    var anonymous = mutableListOf<Marker>()

    var userOverlay: MyLocationNewOverlay? = null
    var userEnabled = false

    /**
     * Whether Go is asking for the region it last asked for — the apply path's
     * question, and the one comparison here that stays exact.
     *
     * Both numbers came from Go: the argument is this pass's prop and the field
     * is the one the last pass stored, so they are bit-identical whenever Go
     * has not changed its mind. A tolerance would earn nothing and would cost
     * something real — a deliberate small nudge from Go (following a location,
     * stepping a marker along a path) would read as "no change" and never
     * reach the map. See samePlaceOnScreen for the three comparisons that face
     * the *map* and cannot be exact.
     */
    fun hasApplied(lat: Double, lng: Double, zoom: Double): Boolean =
        appliedLat == lat && appliedLng == lng && appliedZoom == zoom

    fun remember(lat: Double, lng: Double, zoom: Double) {
        appliedLat = lat
        appliedLng = lng
        appliedZoom = zoom
    }

    /**
     * Whether the region osmdroid just reported is the one this renderer asked
     * it for — the report path's half of the same question, and it has to
     * tolerate a pixel because osmdroid's answer is not the question.
     *
     * Exact equality here is the bug the emulator run found: every
     * `setCenter` was reported straight back to Go as a region change. See
     * samePlaceOnScreen.
     */
    fun reportMatchesApplied(lat: Double, lng: Double, zoom: Double): Boolean =
        samePlaceOnScreen(appliedLat, appliedLng, appliedZoom, lat, lng, zoom)

    /**
     * Whether this region is where the map is already resting.
     *
     * Tolerant in both directions it is asked in, and for two reasons rather
     * than one. On the report path it compares the map's answer with the map's
     * previous answer, which would match exactly — but on the apply path it
     * compares GO's region with the map's answer, and that is a value the app
     * echoed out of OnRegionChange and may well have rounded on the way
     * through its own state.
     */
    fun hasSettled(lat: Double, lng: Double, zoom: Double): Boolean =
        samePlaceOnScreen(settledLat, settledLng, settledZoom, lat, lng, zoom)

    fun rememberSettled(lat: Double, lng: Double, zoom: Double) {
        settledLat = lat
        settledLng = lng
        settledZoom = zoom
    }
}

/**
 * Whether two regions are the same place *on screen* — the echo guard's
 * comparison wherever one side of it came out of osmdroid.
 *
 * # Why a tolerance is needed at all
 *
 * osmdroid holds its scroll position as integer pixels at the current zoom, so
 * the round trip through the map is lossy by construction:
 *
 *     map.controller.setCenter(GeoPoint(lat, lng))   // exact doubles in
 *     map.mapCenter                                  // quantised doubles out
 *
 * An exact-equality guard can therefore never match on this host, and the
 * emulator run found exactly that: tapping "Show Belém" in lesson 4.12 changed
 * the **Reported** readout to the region Go had just asked for, which it should
 * not have been able to. `applyMapRegion` writes both `applied` and `settled`
 * before moving the map, `reportRegion` returns early on either, and a report
 * getting through at all proves the numbers coming back are not the numbers
 * that went in.
 *
 * It converged and did not jump — the echo landed on `settled` and the next
 * pass skipped — so the cost was not a visible one. The cost was honesty:
 * `OnRegionChange` fired for moves the user did not make, once per
 * programmatic recentre and once on load, which is also why lesson 4.12's
 * "Nothing reported yet — pan or zoom the map." was unreachable on Android. An
 * app that treats the callback as a gesture — a "search this area" fetch, an
 * analytics event — fires it on arrival at every place it sent itself.
 *
 * # In pixels, not in degrees
 *
 * The finest move the map can represent is one pixel, so the tolerance is a
 * number of pixels converted to degrees at the zoom in question, rather than a
 * number of degrees to tune:
 *
 *     256 × 2^zoom pixels span 360° of longitude   (the definition of the
 *                                                   slippy zoom level)
 *     degrees per pixel = 360 / (256 × 2^zoom)
 *
 * A fixed epsilon in degrees cannot work at all: one pixel is 1.7e-4° at zoom
 * 12 and 1.3e-6° at zoom 19, a factor of 128, so a constant coarse enough for
 * the first swallows real movement at the second and one fine enough for the
 * second does nothing at the first.
 *
 * Latitude gets the same pixel count scaled by the cosine of the latitude,
 * which is Mercator's compression — the same factor GrMobMapView.swift's
 * grMobSpan applies for the same reason. A degree of latitude covers more
 * pixels than a degree of longitude away from the equator, so a pixel covers
 * fewer degrees. The latitude is clamped to ±85° first: that is Web Mercator's
 * own limit, and without it the cosine goes to zero at the pole and the
 * tolerance with it.
 *
 * # How wide, which was measured rather than assumed
 *
 * Half a pixel is the number this was first written with, on the argument that
 * rounding to the nearest pixel cannot be off by more than half of one. The
 * emulator disagreed. Logging the deltas on the same "Show Belém" tap, at zoom
 * 15 and 38.7°N:
 *
 *     asked for   38.697,             -9.2065
 *     got back    38.69703256527164,  -9.206500053405762
 *     error       3.26e-5°            5.34e-8°
 *                 0.97 pixels         0.0012 pixels
 *
 * So the two axes are not alike. The longitude comes back essentially intact —
 * osmdroid keeps its scroll X to well under a thousandth of a pixel — and the
 * *latitude* loses nearly a whole one, which is the signature of a Mercator y
 * truncated to an integer pixel rather than rounded to one. Truncation bounds
 * the error below one pixel, not below half, and the measurement sits right
 * under that bound. The same tap at zoom 13 was off by 0.17 pixels, so the
 * error also varies with where in the pixel the target happens to land.
 *
 * GRMOB_MAP_TOLERANCE_PX is therefore 1.5 pixels, which is the measured bound
 * with a little room. What that room cannot do is hide a gesture: Android's
 * touch slop is 8dp — 21 pixels at this emulator's density — before a drag is
 * a drag at all, so the smallest pan a finger can produce is more than an order
 * of magnitude outside this tolerance.
 *
 * # Zoom
 *
 * A small constant, because osmdroid speaks the slippy zoom level natively —
 * `setZoom(14.0)` comes back as 14.0, with no projection round trip to lose
 * anything in. GRMOB_MAP_ZOOM_EPSILON is there so the comparison is uniformly a
 * tolerance rather than a mix of one, and it is far below any pinch. (iOS is
 * the opposite case: MapKit speaks degrees of span, so the zoom is derived
 * through pow/log2 and re-derived by MapKit itself. Its tolerance is computed
 * from the viewport for that reason — see GrMobRegion.isSamePlace.)
 *
 * A zoom osmdroid *clamped* — a request past MAPNIK's maximum of 19 — is a
 * larger mismatch than any tolerance should absorb, and is deliberately not
 * absorbed: the map really is at 19, Go really asked for 25, and reporting 19
 * back is the truth. `hasApplied` then skips Go's unchanged 25 on the next
 * pass, so it converges in one round trip rather than fighting.
 *
 * # NaN
 *
 * "Nothing recorded yet" is NaN, and exact equality gave the right answer for
 * it free — NaN equals nothing, including itself, so the first region was
 * always an instruction. A subtraction does not: every comparison involving
 * NaN is false, so `abs(NaN - x) > eps` is false, which this function would
 * read as "within tolerance" and match. Hence the explicit guard, which is the
 * one line here that a reader has to be told is load-bearing.
 */
private fun samePlaceOnScreen(
    aLat: Double, aLng: Double, aZoom: Double,
    bLat: Double, bLng: Double, bZoom: Double,
): Boolean {
    // See "NaN" above: without this an unrecorded slot matches everything.
    if (aLat.isNaN() || aLng.isNaN() || aZoom.isNaN()) return false
    if (bLat.isNaN() || bLng.isNaN() || bZoom.isNaN()) return false
    if (abs(aZoom - bZoom) > GRMOB_MAP_ZOOM_EPSILON) return false
    // Either zoom will do — they are within GRMOB_MAP_ZOOM_EPSILON of each
    // other by the line above, and that is orders of magnitude below what
    // would change the pixel size. The map's is used because the map's pixels
    // are what the tolerance is about.
    val toleranceLng = 360.0 / (256.0 * 2.0.pow(bZoom)) * GRMOB_MAP_TOLERANCE_PX
    if (abs(aLng - bLng) > toleranceLng) return false
    // Mercator's compression; ±85° is the projection's own limit and keeps the
    // cosine off zero. See above. This is the axis that needed the width — the
    // measured error was 0.97 pixels here and 0.001 on the longitude.
    val clampedLat = bLat.coerceIn(-85.0, 85.0)
    val toleranceLat = toleranceLng * cos(clampedLat * Math.PI / 180.0)
    return abs(aLat - bLat) <= toleranceLat
}

/**
 * Builds the osmdroid MapView and wires the three event paths.
 *
 * The region is NOT set here: the update pass does it, through the echo guard,
 * so there is one place that decides when the map moves.
 */
private fun createOsmMap(ctx: Context, holder: GrMobMapHolder): MapView {
    // osmdroid's configuration is process-global and has to be loaded before a
    // MapView exists. The user agent is what identifies this app to OSM's tile
    // servers, which refuse the library's default.
    val config = Configuration.getInstance()
    config.load(ctx, ctx.getSharedPreferences("osmdroid", Context.MODE_PRIVATE))
    config.userAgentValue = ctx.packageName
    // The tile cache goes in the app's own cache directory, which needs no
    // storage permission on any API level and is cleaned up with the app. The
    // library's default is external storage, which on older releases needs
    // WRITE_EXTERNAL_STORAGE and silently draws nothing without it.
    config.osmdroidBasePath = ctx.cacheDir
    config.osmdroidTileCache = java.io.File(ctx.cacheDir, "tiles")

    val map = MapView(ctx)
    holder.map = map
    map.setTileSource(TileSourceFactory.MAPNIK)
    map.setMultiTouchControls(true)
    // The library's own zoom buttons are a second control beside the pinch, and
    // every other target here draws one too (Leaflet's zoom control, MapKit's
    // implicit gestures) — but osmdroid's are drawn over the map in a fixed
    // corner and cannot be styled, so they are off and the pinch is the
    // gesture. A caller who wants buttons draws them as an overlay child.
    map.zoomController.setVisibility(
        org.osmdroid.views.CustomZoomButtonsController.Visibility.NEVER
    )
    map.onResume()

    // The region report, throttled by the library's own DelayedMapListener: a
    // pan generates a scroll event per frame, and each one that crossed the
    // bridge would be a full Go render pass. The delay is the same number the
    // other two hosts use for the same job (see MAP_REGION_QUIET_MS in
    // grmob-runtime.js), and it also coalesces a pinch's scroll and zoom into
    // one report.
    map.addMapListener(
        DelayedMapListener(
            object : MapListener {
                override fun onScroll(event: ScrollEvent?): Boolean {
                    reportRegion(holder)
                    return true
                }

                override fun onZoom(event: ZoomEvent?): Boolean {
                    reportRegion(holder)
                    return true
                }
            },
            GRMOB_MAP_REGION_QUIET_MS,
        )
    )

    // The map tap. An overlay rather than a touch listener, because osmdroid's
    // own overlays get the touch first — which is exactly the suppression
    // core.OnMapTap promises: a tap that hit a marker is OnMarkerTap's event,
    // and the Marker overlay consumes it before this one is consulted.
    map.overlays.add(
        MapEventsOverlay(object : MapEventsReceiver {
            override fun singleTapConfirmedHelper(p: GeoPoint?): Boolean {
                val cb = holder.node?.stringProp("onMapTap") ?: ""
                if (cb.isEmpty() || p == null) return false
                // The wire form core.ParseLatLng reads. Kotlin's toString for a
                // Double is locale-independent, which is what keeps a decimal
                // comma out of a comma-separated payload.
                holder.report?.invoke(cb, "${p.latitude},${p.longitude}")
                return true
            }

            override fun longPressHelper(p: GeoPoint?): Boolean = false
        })
    )
    return map
}

/** Applies Go's region when it is Go's instruction rather than Go's echo. */
private fun applyMapRegion(map: MapView, holder: GrMobMapHolder, node: GrMobNode) {
    val lat = node.doubleProp("lat")
    val lng = node.doubleProp("lng")
    val zoom = node.doubleProp("zoom")
    // Go has not changed its mind. Nothing to do — and emphatically not a
    // reason to re-centre: the map may be somewhere else entirely because the
    // user put it there, and this is the patch that would yank it back.
    if (holder.hasApplied(lat, lng, zoom)) return
    holder.remember(lat, lng, zoom)
    // Go HAS changed its mind, and to where the map already is: the app echoed
    // OnRegionChange into its own state and this is that value arriving a frame
    // later. Recorded above, because Go is now asking for this region and the
    // next instruction is measured against it — but not applied, because
    // applying it is the round trip an echoing app would otherwise fight.
    if (holder.hasSettled(lat, lng, zoom)) return
    // setZoom before setCenter: osmdroid re-centres on the current zoom's tile
    // grid, and doing it the other way round leaves the centre a fraction of a
    // tile out at the moment both change.
    map.controller.setZoom(zoom)
    map.controller.setCenter(GeoPoint(lat, lng))
    // The map now rests here, so `settled` says so. Without this the slot holds
    // wherever the user last left it, and a later instruction back to that place
    // would be skipped as "already there" while the map sat somewhere else — the
    // invariant is that `settled` is where the map is, however it got there.
    holder.rememberSettled(lat, lng, zoom)
}

/**
 * Reports where the user moved the map to, unless it is where this renderer put
 * it.
 *
 * The comparison is the echo guard read in the other direction, and it is what
 * keeps `applyMapRegion`'s own setCenter from arriving back in Go as a gesture —
 * no timing flag required, which is the same conclusion the web host reached.
 */
private fun reportRegion(holder: GrMobMapHolder) {
    val map = holder.map ?: return
    val center = map.mapCenter
    val lat = center.latitude
    val lng = center.longitude
    val zoom = map.zoomLevelDouble
    // Tolerant, not exact: this compares osmdroid's answer with the doubles
    // this renderer handed it, and osmdroid cannot give those back. See
    // samePlaceOnScreen — an exact comparison here reported every
    // programmatic recentre to Go as a gesture.
    if (holder.reportMatchesApplied(lat, lng, zoom)) return
    // The second half of the comparison, and the reason the user's region does
    // NOT go into `applied`: a map that fires two events without moving between
    // them has one thing to say, and Go's own region is a separate fact a pan
    // must not overwrite. See GrMobMapHolder.settledLat.
    if (holder.hasSettled(lat, lng, zoom)) return
    // Recorded even with no handler attached, so a map that gains one later does
    // not immediately report a pan nobody was listening for.
    holder.rememberSettled(lat, lng, zoom)
    val cb = holder.node?.stringProp("onRegionChange") ?: ""
    if (cb.isEmpty()) return
    holder.report?.invoke(cb, "$lat,$lng,$zoom")
}

/**
 * Reconciles the marker overlays against the node's Marker children.
 *
 * By id rather than by clearing the overlay list, which is the difference
 * between a patch and a rebuild: osmdroid redraws every marker it is handed
 * afresh and closes any open info window, so a list where one pin moved would
 * flicker all of them.
 */
private fun applyMapMarkers(map: MapView, holder: GrMobMapHolder, node: GrMobNode) {
    val seen = HashSet<String>()
    val nextAnonymous = mutableListOf<Marker>()
    var anonIndex = 0

    for (child in node.children) {
        if (child.type != "Marker") continue
        val id = child.stringProp("id")
        val point = GeoPoint(child.doubleProp("lat"), child.doubleProp("lng"))
        val title = child.stringProp("title")

        if (id.isEmpty()) {
            val existing = holder.anonymous.getOrNull(anonIndex)
            anonIndex++
            nextAnonymous.add(placeMarker(map, holder, existing, id, point, title))
            continue
        }
        seen.add(id)
        holder.markers[id] = placeMarker(map, holder, holder.markers[id], id, point, title)
    }

    val gone = holder.markers.keys.filter { it !in seen }
    for (id in gone) {
        holder.markers.remove(id)?.let { map.overlays.remove(it) }
    }
    for (extra in holder.anonymous.drop(nextAnonymous.size)) {
        map.overlays.remove(extra)
    }
    holder.anonymous = nextAnonymous
}

private fun placeMarker(
    map: MapView,
    holder: GrMobMapHolder,
    existing: Marker?,
    id: String,
    point: GeoPoint,
    title: String,
): Marker {
    val marker = existing ?: Marker(map).also { fresh ->
        fresh.setAnchor(Marker.ANCHOR_CENTER, Marker.ANCHOR_BOTTOM)
        fresh.setOnMarkerClickListener { _, _ ->
            val cb = holder.node?.stringProp("onMarkerTap") ?: ""
            if (cb.isNotEmpty()) {
                // The id off the marker's own relatedObject rather than this
                // closure's `id`: an update-props patch can rewrite which place
                // a pin is, and a marker reporting the id it was born with would
                // open the wrong row.
                holder.report?.invoke(cb, fresh.relatedObject as? String ?: "")
            }
            // Consumed, so the MapEventsOverlay below never also sees this tap
            // as a map tap — which is the suppression core.OnMapTap documents.
            true
        }
        map.overlays.add(fresh)
    }
    marker.relatedObject = id
    // Guarded: assigning a position invalidates the overlay, so a pin that has
    // not moved must not be told where it is.
    if (marker.position.latitude != point.latitude ||
        marker.position.longitude != point.longitude
    ) {
        marker.position = point
    }
    marker.title = title.ifEmpty { null }
    return marker
}

/**
 * core.ShowUserLocation: osmdroid's own position overlay, which draws the dot
 * and its accuracy circle.
 *
 * Enabled and disabled rather than created and destroyed, because the overlay
 * owns a location subscription: a watch left running is a GPS left running,
 * which is the one cost here a user can measure. The dot needs
 * ACCESS_COARSE_LOCATION, which is what this app declares — see the note in
 * permission.Location on why one Go constant means the narrower thing
 * everywhere — so the dot is as accurate as a coarse fix, and draws nothing at
 * all if the permission was refused. A map is still a map.
 */
private fun applyMapUser(
    map: MapView,
    holder: GrMobMapHolder,
    ctx: Context,
    want: Boolean,
) {
    if (want == holder.userEnabled) return
    holder.userEnabled = want
    if (!want) {
        holder.userOverlay?.disableMyLocation()
        return
    }
    val overlay = holder.userOverlay ?: MyLocationNewOverlay(GpsMyLocationProvider(ctx), map)
        .also {
            holder.userOverlay = it
            map.overlays.add(it)
        }
    overlay.enableMyLocation()
    // Not enableFollowLocation: the dot is information, not a command to go
    // there. An app that wants to follow the user renders a Region from
    // hooks.UseLocation, which is a decision it makes rather than one this code
    // makes for it.
}

/**
 * How long a gesture has to be quiet before its region is reported, in ms. The
 * same number the other two hosts use (MAP_REGION_QUIET_MS in
 * grmob-runtime.js, GrMobMapRegionQuiet in GrMobMapView.swift) for the same
 * reason.
 */
private const val GRMOB_MAP_REGION_QUIET_MS = 120L

/**
 * How far two zoom levels may differ and still be the same zoom.
 *
 * Small, because osmdroid round-trips the slippy zoom level exactly and there
 * is nothing here to lose precision in — see samePlaceOnScreen's "Zoom". At
 * 1e-6 the viewport scales by a factor of 1.0000007, which moves the edge of a
 * 4000px-wide map by 0.0014px, and no pinch produces a delta within three
 * orders of magnitude of it.
 */
private const val GRMOB_MAP_ZOOM_EPSILON = 1e-6

/**
 * How many of the map's own pixels two regions may differ by and still be the
 * same place.
 *
 * 1.5, which is a measurement and not a taste — see samePlaceOnScreen's "How
 * wide". osmdroid truncates its Mercator y to an integer pixel, so a centre
 * handed to it comes back up to one full pixel north or south of where it went
 * in; the emulator produced 0.97 of one at zoom 15. The remaining half is
 * headroom, and it is affordable because the smallest pan a finger can make is
 * Android's 8dp touch slop — 21 pixels at density 2.625 — so nothing a user
 * does lands inside this.
 */
private const val GRMOB_MAP_TOLERANCE_PX = 1.5
