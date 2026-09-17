// The browser half of core's window event (core/window.go), against the
// minimal DOM.
//
// What is tested here is the part no Go test can reach: how two viewport
// segments become a hinge, and when the page reports. The rectangles are the
// shapes a real foldable's Chrome hands a page:
//
//   book / dual-screen   two segments side by side    → vertical hinge
//   tabletop             two segments stacked         → horizontal hinge
//   anything unfolded    no segments (null)           → size only

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime } from "./load.mjs";

// A runtime with a place for window events to land, and a viewport.
function harness({ width = 841, height = 673, segments, posture } = {}) {
    const rt = loadRuntime();
    const reports = [];
    rt.window.GrMobWASM = {
        HostEvent: (name, json) => {
            if (name === "window") reports.push(JSON.parse(json));
        },
    };
    rt.window.innerWidth = width;
    rt.window.innerHeight = height;
    if (segments !== undefined) rt.window.viewport = { segments };
    if (posture !== undefined) rt.sandbox.navigator.devicePosture = { type: posture };
    return { rt, reports, wm: rt.GrMob.windowMetrics };
}

const rect = (x, y, width, height) => ({ x, y, width, height });

test("no segments reports the size and no fold", () => {
    const h = harness({ width: 1280, height: 800, segments: null });
    h.wm.report();
    assert.deepEqual(h.reports, [{ width: 1280, height: 800 }]);
});

test("side-by-side segments with no gap are a vertical, non-occluding hinge", () => {
    const h = harness({
        segments: [rect(0, 0, 420, 673), rect(420, 0, 421, 673)],
        posture: "folded",
    });
    h.wm.report();
    assert.deepEqual(h.reports[0].fold, {
        state: "half_opened", orientation: "vertical",
        separating: true, occluding: false,
        x: 420, y: 0, width: 0, height: 673,
    });
});

test("a gap between segments is an occluding seam", () => {
    const fold = loadRuntime().GrMob.windowMetrics.foldFrom(
        [rect(0, 0, 360, 720), rect(388, 0, 360, 720)], "continuous");
    assert.equal(fold.state, "flat");
    assert.equal(fold.occluding, true);
    assert.equal(fold.x, 360);
    assert.equal(fold.width, 28);
});

test("stacked segments are a horizontal hinge — the tabletop shape", () => {
    const fold = loadRuntime().GrMob.windowMetrics.foldFrom(
        [rect(0, 0, 673, 420), rect(0, 420, 673, 421)], "folded");
    assert.equal(fold.orientation, "horizontal");
    assert.equal(fold.y, 420);
    assert.equal(fold.width, 673);
    assert.equal(fold.height, 0);
});

// Posture alone says the device is bent but not where the hinge is; a fold
// with no position is nothing a layout can use, so none is invented.
test("posture without segments reports no fold", () => {
    const h = harness({ segments: null, posture: "folded" });
    h.wm.report();
    assert.equal(h.reports[0].fold, undefined);
});

test("shapes core cannot express are dropped, not guessed", () => {
    const { foldFrom } = loadRuntime().GrMob.windowMetrics;
    assert.equal(foldFrom([rect(0, 0, 100, 100)], "folded"), null);
    assert.equal(foldFrom([rect(0, 0, 100, 100), rect(50, 50, 100, 100)], "folded"), null);
    assert.equal(foldFrom([rect(0, 0, 1, 1), rect(1, 0, 1, 1), rect(2, 0, 1, 1)], "folded"), null);
});

test("a resize reports; nothing is sent before Go is up", () => {
    const h = harness({ width: 400, height: 800 });
    const host = h.rt.window.GrMobWASM;
    delete h.rt.window.GrMobWASM;
    h.rt.fireWindowEvent("resize", {});
    assert.equal(h.reports.length, 0);

    h.rt.window.GrMobWASM = host;
    h.rt.window.innerWidth = 900;
    assert.equal(h.rt.fireWindowEvent("resize", {}) > 0, true, "the runtime listens for resize");
    assert.deepEqual(h.reports, [{ width: 900, height: 800 }]);
});

// The minimal DOM has no innerWidth unless a test sets one; a page with no
// viewport sends nothing rather than a NaN-sized window.
test("no viewport, no report", () => {
    const rt = loadRuntime();
    const reports = [];
    rt.window.GrMobWASM = { HostEvent: (name) => reports.push(name) };
    rt.GrMob.windowMetrics.report();
    assert.deepEqual(reports, []);
});
