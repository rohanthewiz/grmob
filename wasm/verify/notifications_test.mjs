// The browser half of core's scheduled notifications (LocalNotification.At).
//
// A page cannot ask the browser to post for it later, so "at" is a timer in
// the page. What is worth pinning is the bookkeeping around that timer, since
// no Go test can see it:
//
//   - a future "at" waits, and posts when its time comes
//   - a timer that fires early (the loader's queue fires on demand, and a
//     capped wait for a far-off time fires before it is due) re-arms rather
//     than posting
//   - posting again under an id replaces its pending timer; cancel stops it
//   - permission is read when the banner is due, not when it was scheduled

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime } from "./load.mjs";

// A runtime with a Notification constructor that records what was shown, and
// a clock the test moves. The runtime reads Date.now() through the context's
// global, which the sandbox property shadows.
function harness({ permission = "granted", now = 1_000_000 } = {}) {
    const rt = loadRuntime();
    const shown = [];
    const clock = { now };
    function Notification(title, options) {
        this.title = title;
        this.options = options;
        this.close = () => {};
        shown.push({ title, body: options.body, tag: options.tag });
    }
    Notification.permission = permission;
    rt.sandbox.Notification = Notification;
    const RealDate = Date;
    rt.sandbox.Date = class extends RealDate {
        static now() { return clock.now; }
    };
    return {
        rt,
        shown,
        clock,
        Notification,
        post: (id, extra = {}) =>
            rt.GrMob.notifications.handle({ command: "post", id, title: "T", body: "B", ...extra }),
        cancel: (id) => rt.GrMob.notifications.handle({ command: "cancel", id }),
        // Fires the notification timers only: the page's own 100ms poll is
        // below this floor.
        fire: () => rt.drainTimers(1000),
    };
}

test("a post without at shows at once", () => {
    const h = harness();
    h.post("now");
    assert.deepEqual(h.shown, [{ title: "T", body: "B", tag: "now" }]);
});

test("a future at waits for its time, and an early firing re-arms", () => {
    const h = harness();
    h.post("later", { at: h.clock.now + 60_000 });
    assert.equal(h.shown.length, 0, "shown before its time");

    // The timer fires but the clock has not moved: not due, so it re-arms.
    assert.equal(h.fire(), 1);
    assert.equal(h.shown.length, 0, "an early firing posted");

    h.clock.now += 60_000;
    assert.equal(h.fire(), 1);
    assert.deepEqual(h.shown.map((n) => n.tag), ["later"]);
    assert.equal(h.fire(), 0, "a timer was left behind after posting");
});

test("a re-post replaces the pending timer and a cancel stops it", () => {
    const h = harness();
    h.post("x", { at: h.clock.now + 10_000 });
    h.post("x", { at: h.clock.now + 20_000, title: "second" });
    h.clock.now += 20_000;
    h.fire();
    assert.deepEqual(h.shown.map((n) => n.title), ["second"], "both timers posted, or the old one did");

    h.post("y", { at: h.clock.now + 5_000 });
    h.cancel("y");
    h.clock.now += 5_000;
    assert.equal(h.fire(), 0);
    assert.equal(h.shown.length, 1, "a cancelled notification was shown");
});

test("a wait past setTimeout's range is taken in capped steps", () => {
    const h = harness();
    const month = 31 * 24 * 3600 * 1000;
    h.post("far", { at: h.clock.now + month });
    const delays = [];
    const realSet = h.rt.sandbox.setTimeout;
    h.rt.sandbox.setTimeout = (fn, delay) => { delays.push(delay); return realSet(fn, delay); };
    // The first wait was queued before the spy; fire it with the clock at the
    // cap, which re-arms for the remainder.
    h.clock.now += 2147483647;
    h.fire();
    assert.equal(delays.length, 1);
    assert.equal(delays[0], month - 2147483647);
    assert.equal(h.shown.length, 0);
});

test("permission is read when the banner is due", () => {
    const h = harness({ permission: "default" });
    h.post("p", { at: h.clock.now + 1_000 });
    h.Notification.permission = "granted";
    h.clock.now += 1_000;
    h.fire();
    assert.deepEqual(h.shown.map((n) => n.tag), ["p"]);
});

// A sweep stops every timer under its prefix, leaves banners already shown
// open and other ids alone, and answers with the scheduled ids whose time
// came — once: a second sweep, or a cancel or re-post in between, reports
// nothing more.
test("a sweep cancels by prefix and reports what fired", () => {
    const h = harness();
    const replies = [];
    h.rt.window.GrMobWASM = {
        HostEvent: (name, json) => replies.push({ name, ...JSON.parse(json) }),
    };
    const closed = [];
    const Base = h.rt.sandbox.Notification;
    function Tracking(title, options) {
        Base.call(this, title, options);
        this.close = () => closed.push(options.tag);
    }
    Tracking.permission = "granted";
    h.rt.sandbox.Notification = Tracking;

    h.post("a.1", { at: h.clock.now + 1_000 }); // fires, then is swept
    h.post("a.2", { at: h.clock.now + 1_000 }); // fires, then is re-posted now
    h.post("a.3", { at: h.clock.now + 60_000 }); // still pending at the sweep
    h.post("b.1", { at: h.clock.now + 60_000 }); // another prefix
    h.clock.now += 1_000;
    h.fire();
    h.post("a.2");

    h.rt.GrMob.notifications.handle({ command: "sweep", prefix: "a.", request: "7" });
    assert.deepEqual(replies, [{ name: "notification_swept", request: "7", fired: ["a.1"] }]);
    assert.deepEqual(closed.filter((t) => t.startsWith("a.")).sort(), ["a.2"],
        "only a.2's first banner closed, by its re-post; the sweep closes no shown banner");

    h.clock.now += 60_000;
    h.fire();
    assert.deepEqual(h.shown.map((n) => n.tag), ["a.1", "a.2", "a.2", "b.1"], "a swept timer still fired");

    h.rt.GrMob.notifications.handle({ command: "sweep", prefix: "a.", request: "8" });
    assert.deepEqual(replies[1], { name: "notification_swept", request: "8", fired: [] });
});
