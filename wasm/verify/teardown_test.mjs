// What an element releases when it leaves the document (N-137 e, the plan's
// R9e).
//
// A few elements hold something outside the DOM: a map its Leaflet instance
// and, with core.ShowUser, a geolocation watch; a list with OnEndReached its
// IntersectionObserver; a canvas drawing text its place in the page-long
// ResizeObserver. Removing the element released none of it, so a ShowUser
// map's GPS watch outlived its screen. The runtime now releases them at the
// end of the batch (or mount) that removed them. What is pinned:
//
//   - a map that leaves by a remove, by an ancestor's replace, or by a new
//     mount stops its location watch and is removed from Leaflet
//   - an element that stays, through any number of batches, keeps everything
//   - a list's observer and a canvas's place in the ResizeObserver go too
//   - a map removed before its creation frame is never created, rather than
//     retried every frame for good

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";
import { newLeaflet } from "./leaflet.mjs";

const REGION = { lat: 38.7223, lng: -9.1393, zoom: 14 };

// A Column holding a ShowUser map and a Text beside it, so a test can patch
// the Text and leave the map where it is.
const screen = (title = "Map") => ({
    Type: "Column",
    Children: [
        { Type: "MapView", Props: { ...REGION, showUser: true }, Children: [] },
        { Type: "Text", Props: { content: title } },
    ],
});

function mountMap() {
    const rt = loadRuntime();
    const { L, maps } = newLeaflet();
    rt.window.L = L;
    rt.GrMob.mount(JSON.stringify(screen()));
    rt.drainFrames();
    assert.equal(maps.length, 1);
    assert.equal(maps[0].locates.length, 1, "ShowUser started a location watch");
    return { rt, L, map: maps[0], maps };
}

const released = (map) => map.stops === 1 && map.removed === 1;

test("a map removed by a patch stops its location watch and is released", () => {
    const { rt, map } = mountMap();
    rt.GrMob.patch(JSON.stringify([{ Type: "remove", TargetID: "root/0" }]));

    assert.ok(released(map), `stops ${map.stops}, removed ${map.removed}`);
});

test("a map whose ancestor is replaced is released", () => {
    const { rt, map } = mountMap();
    // Navigation is a replace of the screen, not a remove of the map.
    rt.GrMob.patch(JSON.stringify([{
        Type: "replace", TargetID: "root",
        Changes: { Type: "Column", Children: [{ Type: "Text", Props: { content: "Contents" } }] },
    }]));

    assert.ok(released(map), `stops ${map.stops}, removed ${map.removed}`);
});

test("a map under a tree that a new mount replaces is released", () => {
    const { rt, map } = mountMap();
    rt.GrMob.mount(JSON.stringify({ Type: "Text", Props: { content: "fresh" } }));

    assert.ok(released(map), `stops ${map.stops}, removed ${map.removed}`);
});

test("a map that stays keeps its watch through unrelated batches", () => {
    const { rt, map } = mountMap();
    for (let i = 0; i < 3; i++) {
        rt.GrMob.patch(JSON.stringify([{
            Type: "update-props", TargetID: "root/1", Changes: { content: `Map ${i}` },
        }]));
    }
    assert.equal(map.stops, 0, "a map still on screen stopped its watch");
    assert.equal(map.removed, 0, "a map still on screen was released");

    // And it is released once, however many batches follow its removal.
    rt.GrMob.patch(JSON.stringify([{ Type: "remove", TargetID: "root/0" }]));
    rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root/0", Changes: { content: "after" } }]));
    assert.ok(released(map), `stops ${map.stops}, removed ${map.removed}`);
});

// A map is built a frame after its element is (Leaflet measures it). One
// removed in between used to be retried every frame for the life of the page.
test("a map removed before its creation frame is never created and stops asking", () => {
    const rt = loadRuntime();
    const { L, maps } = newLeaflet();
    rt.window.L = L;
    rt.GrMob.mount(JSON.stringify(screen()));
    rt.GrMob.patch(JSON.stringify([{ Type: "remove", TargetID: "root/0" }]));

    rt.drainFrames();
    assert.equal(maps.length, 0, "a removed map was built");
    assert.equal(rt.pendingFrames(), 0, "the removed map is still asking for frames");
});

test("a list's end-reached observer is disconnected when the list leaves", () => {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "Column",
        Children: [{
            Type: "List", Props: { onEndReached: "cb_0" },
            Children: [{ Type: "Text", Props: { content: "a" } }, { Type: "Text", Props: { content: "b" } }],
        }],
    }));
    rt.drainFrames();
    const watching = rt.observers.filter((o) => o.targets.size);
    assert.equal(watching.length, 1, "the list observes its last row");

    rt.GrMob.patch(JSON.stringify([{ Type: "remove", TargetID: "root/0" }]));
    assert.equal(watching[0].targets.size, 0, "the removed list's observer still holds a row");
});

test("a canvas drawing text leaves the page-long ResizeObserver when it leaves", () => {
    const rt = loadRuntime({ resizeObserver: true });
    rt.GrMob.mount(JSON.stringify({
        Type: "Column",
        Children: [{
            Type: "Canvas", Props: { vw: 100, vh: 50, scale: "fit" },
            Children: [{ Type: "CanvasText", Props: { text: "Jan", at: [10, 20], size: 12, fill: "#111" } }],
        }],
    }));
    rt.drainFrames();
    const svg = nodeAt(rt.document, "root/0");
    const holding = rt.resizeObservers.filter((o) => o.targets.has(svg));
    assert.equal(holding.length, 1, "the canvas is observed while it draws text");

    rt.GrMob.patch(JSON.stringify([{ Type: "remove", TargetID: "root/0" }]));
    assert.equal(holding[0].targets.has(svg), false, "the removed canvas is still held");
});
