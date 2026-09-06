// The browser half of core's compass (core/heading.go), against the minimal
// DOM.
//
// Everything interesting about this module is a browser difference, and none
// of it is reachable from Go: which event carries a bearing that means north,
// which direction the number counts in, and what happens on a machine that
// has no compass at all. So it is tested here, where the three browsers can
// be modelled side by side.
//
//   Chrome/Android   deviceorientationabsolute, alpha counter-clockwise
//   Safari/iOS       deviceorientation + webkitCompassHeading, clockwise,
//                    behind DeviceOrientationEvent.requestPermission
//   desktop          the events exist and never fire

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime } from "./load.mjs";

// The availability timer's delay, from the runtime. Used as the drainTimers
// floor so firing it never also fires waitForWasm's 100ms poll, which would
// simply re-queue itself.
const AVAILABILITY_MS = 2000;

// A runtime with a place for heading events to land. Returns the readings Go
// would have seen, decoded.
function harness({ absolute = true, requestPermission } = {}) {
    const rt = loadRuntime();
    const readings = [];
    rt.window.GrMobWASM = {
        HostEvent: (name, json) => {
            if (name === "heading") readings.push(JSON.parse(json));
        },
    };
    // Chrome exposes the absolute event; Safari does not, and the runtime
    // falls back to the plain one.
    if (absolute) rt.window.ondeviceorientationabsolute = null;
    if (requestPermission) {
        rt.sandbox.DeviceOrientationEvent.requestPermission = requestPermission;
    }
    return {
        rt,
        readings,
        start: () => rt.GrMob.heading.handle({ kind: "heading", command: "start" }),
        stop: () => rt.GrMob.heading.handle({ kind: "heading", command: "stop" }),
        fire: (type, event) => rt.fireWindowEvent(type, event),
        listeners: (type) => rt.windowListenerCount(type),
    };
}

test("Chrome's counter-clockwise alpha becomes a clockwise bearing", () => {
    const h = harness();
    h.start();

    // alpha counts counter-clockwise from north, so 90 (a quarter turn
    // anticlockwise) is a device pointing *west*: bearing 270. Getting this
    // complement backwards mirrors the compass and looks entirely plausible
    // until you turn the phone.
    h.fire("deviceorientationabsolute", { absolute: true, alpha: 90 });
    assert.equal(h.readings.length, 1);
    assert.equal(h.readings[0].magnetic, 270);
});

test("a relative orientation event is ignored, not treated as a bearing", () => {
    const h = harness();
    h.start();

    // Android's non-absolute alpha is measured from wherever the device
    // happened to be when the listener attached. It looks exactly like a
    // compass and points somewhere arbitrary, which is worse than no compass.
    h.fire("deviceorientationabsolute", { absolute: false, alpha: 90 });
    assert.equal(h.readings.length, 0);
});

test("Safari's webkitCompassHeading is taken as-is, with its accuracy", () => {
    const h = harness({ absolute: false });
    h.start();

    // No complement here: Safari already answers clockwise from magnetic
    // north. Applying Chrome's 360-alpha to it would mirror the compass on
    // exactly one platform.
    h.fire("deviceorientation", { webkitCompassHeading: 123.5, webkitCompassAccuracy: 12 });
    assert.deepEqual(
        { magnetic: h.readings[0].magnetic, accuracy: h.readings[0].accuracy },
        { magnetic: 123.5, accuracy: 12 },
    );
});

test("a burst of events is throttled to one reading", () => {
    const h = harness();
    h.start();
    for (let i = 0; i < 10; i++) {
        h.fire("deviceorientationabsolute", { absolute: true, alpha: i });
    }
    // Ten events inside one synchronous loop are inside the 66ms window, so
    // nine of them cost nothing. Without this, a 60Hz sensor drives a full Go
    // render pass sixty times a second.
    assert.equal(h.readings.length, 1);
});

test("a device that never reports is called compass-less, once", () => {
    const h = harness();
    h.start();
    assert.equal(h.readings.length, 0, "nothing should be reported before the timeout");

    h.rt.drainTimers(AVAILABILITY_MS);
    assert.equal(h.readings.length, 1);
    assert.deepEqual(h.readings[0], {
        available: false,
        error: "no compass on this device",
    });
});

test("a reading cancels the pending compass-less verdict", () => {
    const h = harness();
    h.start();
    h.fire("deviceorientationabsolute", { absolute: true, alpha: 0 });

    // The verdict must not fire behind a device that has already answered:
    // Go would take the later event as the truth and blank a working compass.
    h.rt.drainTimers(AVAILABILITY_MS);
    assert.equal(h.readings.length, 1);
    assert.equal(h.readings[0].available, undefined,
        "the availability timer fired after a real reading");
});

test("a browser with no orientation API says so instead of waiting", () => {
    const rt = loadRuntime();
    const readings = [];
    rt.window.GrMobWASM = {
        HostEvent: (name, json) => { if (name === "heading") readings.push(JSON.parse(json)); },
    };
    delete rt.sandbox.DeviceOrientationEvent;

    rt.GrMob.heading.handle({ kind: "heading", command: "start" });
    assert.equal(readings.length, 1);
    assert.equal(readings[0].available, false);
});

test("iOS: a granted prompt attaches the listener", async () => {
    let asked = 0;
    const h = harness({
        absolute: false,
        requestPermission: () => { asked++; return Promise.resolve("granted"); },
    });
    h.start();
    await Promise.resolve(); // let the prompt's .then run

    assert.equal(asked, 1);
    assert.equal(h.listeners("deviceorientation"), 1);
    h.fire("deviceorientation", { webkitCompassHeading: 42 });
    assert.equal(h.readings[0].magnetic, 42);
});

test("iOS: a refused prompt reports the reason rather than hanging", async () => {
    const h = harness({
        absolute: false,
        requestPermission: () => Promise.resolve("denied"),
    });
    h.start();
    await Promise.resolve();

    assert.equal(h.listeners("deviceorientation"), 0);
    assert.equal(h.readings.length, 1);
    assert.equal(h.readings[0].available, false);
    assert.match(h.readings[0].error, /not granted/);
});

test("iOS: a prompt outside a user gesture rejects, and the reason survives", async () => {
    // Safari rejects rather than prompting when the call is not attributable
    // to a gesture. The message is what lets an app draw a "tap to enable"
    // button and start again from inside the tap — the only recovery there is.
    const h = harness({
        absolute: false,
        requestPermission: () => Promise.reject(new Error("requires a user gesture")),
    });
    h.start();
    await Promise.resolve();
    await Promise.resolve();

    assert.equal(h.readings.length, 1);
    assert.equal(h.readings[0].available, false);
    assert.match(h.readings[0].error, /user gesture/);
});

test("stop detaches the listener and start is not re-entrant", () => {
    const h = harness();
    h.start();
    h.start(); // core refcounts, but a doubled command must not double-attach
    assert.equal(h.listeners("deviceorientationabsolute"), 1);

    h.stop();
    assert.equal(h.listeners("deviceorientationabsolute"), 0);
    assert.equal(h.fire("deviceorientationabsolute", { absolute: true, alpha: 10 }), 0);
    assert.equal(h.readings.length, 0);
});

test("an unknown sensor kind is dropped", () => {
    const h = harness();
    // core's next sensor is location, and an older page must ignore it rather
    // than treat it as a compass command.
    h.rt.GrMob.heading.handle({ kind: "location", command: "start" });
    assert.equal(h.listeners("deviceorientationabsolute"), 0);
    assert.equal(h.readings.length, 0);
});
