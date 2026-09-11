// The runtime's map reconciliation (core.MapView), against a fake Leaflet.
//
// Everything here is bookkeeping rather than rendering — when setView is called
// and when it deliberately is not, which marker moves, which id is reported,
// whether a move the runtime itself caused travels back to Go as a gesture. See
// leaflet.mjs for why a fake is the right instrument for that and what it
// cannot say.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";
import { newLeaflet } from "./leaflet.mjs";

const REGION = { lat: 38.7223, lng: -9.1393, zoom: 14 };

// mapTree builds the node a Go render would send: a MapView with the given
// marker children.
function mapTree(props = {}, markers = []) {
    return {
        Type: "Column",
        Children: [{
            Type: "MapView",
            Props: { lat: REGION.lat, lng: REGION.lng, zoom: REGION.zoom, ...props },
            Children: markers,
        }],
    };
}

const marker = (id, lat, lng, title = "") => ({
    Type: "Marker",
    Key: id ? `marker:${id}` : "",
    Props: { id, lat, lng, title },
});

// mountMap installs the fake Leaflet *before* mounting, because the runtime
// decides between the engine and the placeholder on its first sync.
function mountMap(props, markers) {
    const rt = loadRuntime();
    const { L, maps } = newLeaflet();
    rt.window.L = L;
    rt.GrMob.mount(JSON.stringify(mapTree(props, markers)));
    rt.drainFrames();
    return { rt, maps, el: () => nodeAt(rt.document, "root/0") };
}

test("a map is created at Go's region, with OSM tiles under it", () => {
    const { maps, el } = mountMap();

    assert.equal(maps.length, 1, "one Leaflet map per MapView node");
    const map = maps[0];
    assert.equal(map.el, el(), "the map was given the node's own element");
    assert.deepEqual(map.center, { lat: REGION.lat, lng: REGION.lng });
    assert.equal(map.zoom, REGION.zoom);
    // Creation is not a setView: the region went in as the map's options, so
    // the echo guard starts with nothing to undo.
    assert.deepEqual(map.views, []);

    const tiles = map.layers.filter((l) => l.kind === "tiles");
    assert.equal(tiles.length, 1, "exactly one tile layer");
    assert.match(tiles[0].url, /tile\.openstreetmap\.org/);
    assert.match(tiles[0].opts.attribution, /OpenStreetMap/,
        "the tile licence requires the credit");

    // The region is on the element as data, which is the same place htmlout
    // puts it in a static export and the single source both web targets read.
    assert.equal(el().dataset.lat, String(REGION.lat));
    assert.equal(el().dataset.zoom, String(REGION.zoom));
});

test("a patch restating the same region moves nothing", () => {
    const { rt, maps, el } = mountMap();

    // The user drags: Leaflet's own state moves, and the runtime is told.
    maps[0].center = { lat: 40, lng: -8 };
    maps[0].zoom = 16;

    // Go re-renders for some unrelated reason and restates the region it always
    // had. This is the patch that makes a map unusable if it is obeyed.
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0",
        Changes: { lat: REGION.lat, lng: REGION.lng, zoom: REGION.zoom },
    }]));
    rt.drainFrames();

    assert.deepEqual(maps[0].views, [],
        "Go restating its region yanked the map back from under the finger");
    assert.deepEqual(maps[0].center, { lat: 40, lng: -8 }, "the user's view survived");
    assert.equal(el().dataset.lat, String(REGION.lat), "the dataset still tracks Go");
});

test("a patch with a new region is an instruction and is applied", () => {
    const { rt, maps } = mountMap();

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0",
        Changes: { lat: -33.8688, lng: 151.2093, zoom: 16 },
    }]));
    rt.drainFrames();

    assert.equal(maps[0].views.length, 1, "Go moving the map is an instruction");
    const view = maps[0].views[0];
    // Field by field rather than one deepEqual: the options object was built
    // inside the vm context, so it carries that realm's Object.prototype and a
    // strict deep-equal against a literal here fails on prototype identity
    // alone. load.mjs's GoInvokeCallback notes the same realm bookkeeping.
    assert.equal(view.lat, -33.8688);
    assert.equal(view.lng, 151.2093);
    assert.equal(view.zoom, 16);
    assert.equal(view.options.animate, false,
        "an animated setView lags the state an app is driving the map from");
});

test("a user's pan is reported to Go once the gesture is quiet", () => {
    const { rt, maps } = mountMap({ onRegionChange: "txt_cb_0" });

    maps[0].center = { lat: 40.5, lng: -8.25 };
    maps[0].zoom = 15;
    // A pinch ends as both events a few milliseconds apart; one report is the
    // answer, which is what the quiet timer is for.
    maps[0].fire("moveend", {});
    maps[0].fire("zoomend", {});
    assert.deepEqual(rt.dispatched, [], "reported before the gesture went quiet");

    rt.drainTimers();
    assert.deepEqual(rt.dispatched, [
        { id: "txt_cb_0", payload: { value: "40.5,-8.25,15" } },
    ], "one report, in the wire form core.ParseRegion reads");
});

test("a region the runtime itself applied is not reported back as a gesture", () => {
    const { rt, maps } = mountMap({ onRegionChange: "txt_cb_0" });

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0",
        Changes: { lat: 1, lng: 2, zoom: 10, onRegionChange: "txt_cb_0" },
    }]));
    rt.drainFrames();
    // The real Leaflet fires moveend from setView; the fake leaves that to the
    // test, which is the same sequence.
    maps[0].fire("moveend", {});
    rt.drainTimers();

    assert.deepEqual(rt.dispatched, [],
        "the runtime's own setView came back to Go as a user gesture");
});

test("after a reported pan, Go's old region is no longer an instruction", () => {
    // The subtle half of the echo guard. Go's props still say the original
    // region — the app has not re-rendered yet — and the next unrelated patch
    // carries them again. If the report did not update what the runtime
    // considers "applied", that patch would read as a change and snap the map
    // back to where the user started.
    const { rt, maps } = mountMap({ onRegionChange: "txt_cb_0" });

    maps[0].center = { lat: 40.5, lng: -8.25 };
    maps[0].zoom = 15;
    maps[0].fire("moveend", {});
    rt.drainTimers();

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0",
        Changes: { lat: 40.5, lng: -8.25, zoom: 15, onRegionChange: "txt_cb_0" },
    }]));
    rt.drainFrames();
    assert.deepEqual(maps[0].views, [],
        "the app echoing the region it was just told fought its own round trip");
});

// The test the browser found, and the one the fake's own pan test was one line
// short of. The pan above sets the map's centre and does NOT fire moveend, so
// the report path never runs — which is the one arrangement real Leaflet never
// produces. With moveend fired, the report path used to write the user's region
// into the slot the apply path reads, and the very next patch to reach this map
// read Go's unchanged region as a change.
//
// Every patch in this test is one an ordinary app produces: a pin added to a
// map the user has panned is the demo in tutorial lesson 4.12, and it snapped
// the map back to the opening view on every tap.
test("an unrelated patch after a reported pan leaves the user's view alone", () => {
    const { rt, maps } = mountMap({ onRegionChange: "txt_cb_0" });

    // The user pans. Real Leaflet fires moveend, so the runtime reports it.
    maps[0].center = { lat: 40.5, lng: -8.25 };
    maps[0].zoom = 15;
    maps[0].fire("moveend", {});
    rt.drainTimers();
    assert.equal(rt.dispatched.length, 1, "the pan should have been reported");

    // The app does not echo the region — the ordinary case, and the one
    // core.MapView's doc says needs nothing special. So the next render still
    // carries the region it always had, alongside whatever actually changed.
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0",
        Changes: { lat: REGION.lat, lng: REGION.lng, zoom: REGION.zoom,
                   onRegionChange: "txt_cb_0" },
    }]));
    rt.drainFrames();

    assert.deepEqual(maps[0].views, [],
        "an unrelated patch yanked the map back from under the finger");
    assert.deepEqual(maps[0].center, { lat: 40.5, lng: -8.25 },
        "the user's view survived");
});

// And the same sequence read from the other end: after the app HAS echoed the
// pan into its state, a genuine instruction back to the opening region is still
// obeyed. The fix must not turn "already there" into "never move again".
test("after an echoed pan, Go can still move the map somewhere else", () => {
    const { rt, maps } = mountMap({ onRegionChange: "txt_cb_0" });

    maps[0].center = { lat: 40.5, lng: -8.25 };
    maps[0].zoom = 15;
    maps[0].fire("moveend", {});
    rt.drainTimers();

    // The echo, which moves nothing.
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0",
        Changes: { lat: 40.5, lng: -8.25, zoom: 15, onRegionChange: "txt_cb_0" },
    }]));
    rt.drainFrames();
    assert.deepEqual(maps[0].views, [], "the echo fought its own round trip");

    // And now a real instruction, back to where the map opened.
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0",
        Changes: { lat: REGION.lat, lng: REGION.lng, zoom: REGION.zoom,
                   onRegionChange: "txt_cb_0" },
    }]));
    rt.drainFrames();
    assert.equal(maps[0].views.length, 1,
        "Go moving the map back is an instruction like any other");
    assert.equal(maps[0].views[0].lat, REGION.lat);
});

// A second event for a map that has not moved since the last one says nothing,
// and must not reach Go — which is the other half of keeping the user's region
// in a slot of its own. A pinch that ends as a zoomend and a moveend is already
// coalesced by the quiet timer; this is the pair that arrives further apart.
test("a map that has not moved since its last report reports nothing again", () => {
    const { rt, maps } = mountMap({ onRegionChange: "txt_cb_0" });

    maps[0].center = { lat: 40.5, lng: -8.25 };
    maps[0].zoom = 15;
    maps[0].fire("moveend", {});
    rt.drainTimers();
    assert.equal(rt.dispatched.length, 1);

    maps[0].fire("moveend", {});
    rt.drainTimers();
    assert.equal(rt.dispatched.length, 1,
        "the same region was reported twice");
});

test("a tap on the map reports where, and carries no marker id", () => {
    const { rt, maps } = mountMap({ onMapTap: "txt_cb_1" });

    maps[0].fire("click", { latlng: { lat: 10.5, lng: -20.25 } });

    assert.deepEqual(rt.dispatched, [
        { id: "txt_cb_1", payload: { value: "10.5,-20.25" } },
    ]);
});

test("markers are created per child, and only the one that moved is moved", () => {
    const { rt, maps, el } = mountMap({}, [
        marker("hall", 38.7223, -9.1393, "The hall"),
        marker("annex", 38.7251, -9.1402),
    ]);

    const markers = maps[0].markers();
    assert.equal(markers.length, 2);
    assert.deepEqual(markers[0].latlng, { lat: 38.7223, lng: -9.1393 });
    assert.equal(markers[0].popup, "The hall", "a titled marker binds a popup");
    assert.equal(markers[1].popup, null, "an untitled one binds nothing");

    // The marker elements are real nodes, which is the whole point of them
    // being children: the patch below is addressed to one.
    assert.equal(el().children.length, 2);
    assert.equal(el().children[0].dataset.markerId, "hall");

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0/1",
        Changes: { id: "annex", lat: 38.80, lng: -9.20, title: "" },
    }]));
    rt.drainFrames();

    assert.deepEqual(markers[1].latlng, { lat: 38.80, lng: -9.20 });
    assert.equal(markers[1].calls.setLatLng, 1);
    assert.equal(markers[0].calls.setLatLng, 0,
        "a sibling's move re-placed a marker that had not moved");
    assert.equal(maps[0].markers().length, 2, "the layer was rebuilt rather than patched");
});

test("a removed marker leaves the layer and an added one joins it", () => {
    const { rt, maps } = mountMap({}, [
        marker("hall", 38.7223, -9.1393),
        marker("annex", 38.7251, -9.1402),
    ]);
    const first = maps[0].markers()[0];

    rt.GrMob.patch(JSON.stringify([{ Type: "remove", TargetID: "root/0/1" }]));
    rt.drainFrames();
    assert.equal(maps[0].markers().length, 1, "the removed marker stayed on the map");
    assert.equal(maps[0].markers()[0], first, "the wrong marker was removed");

    rt.GrMob.patch(JSON.stringify([{
        Type: "add-child", TargetID: "root/0",
        Changes: marker("garden", 38.70, -9.10, "The garden"),
    }]));
    rt.drainFrames();
    const after = maps[0].markers();
    assert.equal(after.length, 2);
    assert.deepEqual(after[1].latlng, { lat: 38.70, lng: -9.10 });
    assert.equal(after[1].popup, "The garden");
});

test("a marker tap reports the id the element carries now", () => {
    const { rt, maps } = mountMap({ onMarkerTap: "txt_cb_2" }, [marker("hall", 1, 2)]);
    const pin = maps[0].markers()[0];

    pin.fire("click", {});
    assert.deepEqual(rt.dispatched, [{ id: "txt_cb_2", payload: { value: "hall" } }]);

    // A patch rewrites which place this pin is. The listener must read the id
    // at fire time, not the one it closed over, or the tap opens the old row.
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0/0",
        Changes: { id: "annex", lat: 1, lng: 2, title: "" },
    }]));
    rt.drainFrames();
    pin.fire("click", {});
    assert.deepEqual(rt.dispatched[1], { id: "txt_cb_2", payload: { value: "annex" } });
});

test("ShowUserLocation starts a watch and dropping it stops one", () => {
    const { rt, maps } = mountMap({ showUser: true });

    assert.equal(maps[0].locates.length, 1, "no location watch was started");
    assert.equal(maps[0].locates[0].watch, true, "the dot has to follow the user");
    assert.equal(maps[0].locates[0].setView, false,
        "the dot is information, not a command to go there");

    // A fix arrives: the dot and its accuracy halo. The halo is not decoration
    // — a 2km fix drawn as a point is a confident lie, and on a desktop browser
    // that is the normal case.
    maps[0].fire("locationfound", { latlng: { lat: 5, lng: 6 }, accuracy: 2000 });
    const circles = maps[0].circles();
    assert.equal(circles.length, 2, "want a dot and an accuracy circle");
    assert.equal(circles[1].radius, 2000);

    // A second fix moves them rather than drawing more.
    maps[0].fire("locationfound", { latlng: { lat: 5.5, lng: 6.5 }, accuracy: 30 });
    assert.equal(maps[0].circles().length, 2, "a second fix drew a second dot");
    assert.deepEqual(circles[0].latlng, { lat: 5.5, lng: 6.5 });
    assert.equal(circles[1].radius, 30);

    // The prop goes away: the watch stops, because a watch left running is a
    // GPS left running.
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0",
        Changes: { lat: REGION.lat, lng: REGION.lng, zoom: REGION.zoom },
    }]));
    rt.drainFrames();
    assert.equal(maps[0].stops, 1, "the location watch was left running");
    assert.equal(maps[0].circles().length, 0, "the dot outlived the request for it");
});

test("with no Leaflet on the page, a map is a placeholder and not a crash", () => {
    // The default harness window has no L, which is also a real browser: a host
    // page that did not add the script tag.
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify(mapTree({}, [marker("hall", 1, 2, "The hall")])));
    rt.drainFrames();

    const el = nodeAt(rt.document, "root/0");
    assert.equal(el.dataset.grmobMapPlaceholder, "1");
    // The region is still on the element, which is what makes the placeholder a
    // degraded map rather than an empty div — and is exactly what htmlout
    // exports for the same node.
    assert.equal(el.dataset.lat, String(REGION.lat));
    assert.equal(el.children.length, 1, "the marker node still exists as a child");
    assert.equal(el.children[0].dataset.markerId, "hall");

    // And a patch over a placeholder map is still harmless.
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0",
        Changes: { lat: 1, lng: 2, zoom: 10 },
    }]));
    rt.drainFrames();
    assert.equal(el.dataset.zoom, "10");
});

test("the marker dataset is total: a dropped title leaves no stale attribute", () => {
    // An update-props patch carries the whole new props map, so a key that is
    // absent now means gone. A guarded write would leave the old callout
    // standing on a pin that no longer has one.
    const { rt, maps } = mountMap({}, [marker("hall", 1, 2, "The hall")]);
    const pin = maps[0].markers()[0];
    assert.equal(pin.popup, "The hall");

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0/0",
        Changes: { id: "hall", lat: 1, lng: 2 },
    }]));
    rt.drainFrames();

    assert.equal(nodeAt(rt.document, "root/0/0").dataset.title, undefined);
    assert.equal(pin.calls.unbindPopup, 1, "the popup survived its title");
    assert.equal(pin.popup, null);
});
