// core.OnSwipe and core.OnVerticalSwipe on the DOM runtime: a pointer tracked
// from down to up, reported once at release as a direction word on the text
// channel ({value}). See "Swipes" in grmob-runtime.js.
//
// The shim delivers an event to one element's own listeners and does not
// bubble, so a press "on a child" is driven as the browser would deliver it
// to the swiping ancestor (pointer events reach it by bubbling), and the
// trailing click is dispatched on the child, whose own onClick listener is
// what has to stay quiet.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

function mount(children) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Props: {}, Children: children }));
    rt.drainFrames();
    return { rt, at: (path) => nodeAt(rt.document, `root/${path}`) };
}

// drag presses at (x0, y0), moves to (x1, y1) and releases there, the event
// sequence a browser sends a mouse or an unclaimed touch.
function drag(el, x0, y0, x1, y1, pointerId = 1) {
    el.dispatch("pointerdown", { pointerId, button: 0, clientX: x0, clientY: y0 });
    el.dispatch("pointermove", { pointerId, clientX: (x0 + x1) / 2, clientY: (y0 + y1) / 2 });
    el.dispatch("pointermove", { pointerId, clientX: x1, clientY: y1 });
    el.dispatch("pointerup", { pointerId, clientX: x1, clientY: y1 });
}

const box = (props, children = []) => ({ Type: "Box", Props: props, Children: children });

test("a horizontal swipe reports leading or trailing in a left-to-right page", () => {
    const { rt, at } = mount([box({ onSwipe: "txt_cb_0" })]);

    drag(at(0), 200, 100, 120, 105); // 80px left
    drag(at(0), 100, 100, 160, 90); // 60px right

    assert.deepEqual(rt.dispatched, [
        { id: "txt_cb_0", payload: { value: "leading" } },
        { id: "txt_cb_0", payload: { value: "trailing" } },
    ]);
});

test("leading is the right-hand side under dir=rtl", () => {
    // The host resolves the logical direction, because Go cannot see the
    // direction the node was laid out in.
    const { rt, at } = mount([box({ onSwipe: "txt_cb_0" })]);
    rt.document.body.children[0].setAttribute("dir", "rtl");

    drag(at(0), 200, 100, 120, 100); // finger moved left
    drag(at(0), 100, 100, 200, 100); // finger moved right

    assert.deepEqual(rt.dispatched.map((d) => d.payload.value), ["trailing", "leading"]);
});

test("a short drag or a mostly vertical one is no horizontal swipe", () => {
    const { rt, at } = mount([box({ onSwipe: "txt_cb_0" })]);

    drag(at(0), 100, 100, 70, 100); // 30px: under SwipeMinDistance
    drag(at(0), 100, 100, 150, 160); // 50 across, 60 down: the vertical wins
    drag(at(0), 100, 100, 150, 150); // a tie is neither axis

    assert.deepEqual(rt.dispatched, []);
});

test("exactly the minimum distance is a swipe", () => {
    const { rt, at } = mount([box({ onSwipe: "txt_cb_0" })]);

    drag(at(0), 100, 100, 60, 100);

    assert.deepEqual(rt.dispatched, [{ id: "txt_cb_0", payload: { value: "leading" } }]);
});

test("onVerticalSwipe reports up and down, and ignores sideways travel", () => {
    const { rt, at } = mount([box({ onVerticalSwipe: "txt_cb_0" })]);

    drag(at(0), 100, 200, 105, 100);
    drag(at(0), 100, 100, 100, 150);
    drag(at(0), 100, 100, 200, 100); // horizontal: not this prop's axis

    assert.deepEqual(rt.dispatched.map((d) => d.payload.value), ["up", "down"]);
});

test("touch-action gives the browser the axis the node does not claim", () => {
    const { at } = mount([
        box({ onSwipe: "txt_cb_0" }),
        box({ onVerticalSwipe: "txt_cb_1" }),
        box({ onSwipe: "txt_cb_2", onVerticalSwipe: "txt_cb_3" }),
        box({ onClick: "cb_0" }),
    ]);

    assert.equal(at(0).style.touchAction, "pan-y pinch-zoom");
    assert.equal(at(1).style.touchAction, "pan-x pinch-zoom");
    assert.equal(at(2).style.touchAction, "pinch-zoom");
    assert.equal(at(3).style.touchAction, undefined, "a node with no swipe was given a touch-action");
});

test("a node with both props reports on the axis travelled further", () => {
    const { rt, at } = mount([box({ onSwipe: "txt_cb_0", onVerticalSwipe: "txt_cb_1" })]);

    drag(at(0), 100, 100, 160, 120);
    drag(at(0), 100, 100, 120, 20);

    assert.deepEqual(rt.dispatched, [
        { id: "txt_cb_0", payload: { value: "trailing" } },
        { id: "txt_cb_1", payload: { value: "up" } },
    ]);
});

test("the click after a swipe is swallowed, once, on the child under the finger", () => {
    // comps.Gallery's case: the tap zones are children of the swiping stage,
    // and a mouse's release sends a click to the zone it ended on.
    const { rt, at } = mount([box({ onSwipe: "txt_cb_0" }, [box({ onClick: "cb_0" })])]);

    drag(at(0), 200, 100, 100, 100);
    at("0/0").dispatch("click", { detail: 1 });
    assert.deepEqual(rt.dispatched, [{ id: "txt_cb_0", payload: { value: "leading" } }]);

    // The guard is the tail of that one release, cleared by its timer.
    rt.drainTimers();
    at("0/0").dispatch("click", { detail: 1 });
    assert.deepEqual(rt.dispatched.at(-1), { id: "cb_0", payload: {} });
});

test("a press that does not swipe leaves its click alone", () => {
    const { rt, at } = mount([box({ onSwipe: "txt_cb_0" }, [box({ onClick: "cb_0" })])]);

    drag(at(0), 100, 100, 104, 101); // a tap's jitter
    at("0/0").dispatch("click", { detail: 1 });

    assert.deepEqual(rt.dispatched, [{ id: "cb_0", payload: {} }]);
});

test("pointercancel forgets the press", () => {
    // The browser took the pointer to pan the axis the node left to it.
    const { rt, at } = mount([box({ onSwipe: "txt_cb_0" })]);

    at(0).dispatch("pointerdown", { pointerId: 1, button: 0, clientX: 200, clientY: 100 });
    at(0).dispatch("pointercancel", { pointerId: 1 });
    at(0).dispatch("pointerup", { pointerId: 1, clientX: 100, clientY: 100 });

    assert.deepEqual(rt.dispatched, []);
});

test("a second pointer neither restarts nor ends the first", () => {
    const { rt, at } = mount([box({ onSwipe: "txt_cb_0" })]);

    at(0).dispatch("pointerdown", { pointerId: 1, button: 0, clientX: 200, clientY: 100 });
    at(0).dispatch("pointerdown", { pointerId: 2, button: 0, clientX: 50, clientY: 100 });
    at(0).dispatch("pointerup", { pointerId: 2, clientX: 0, clientY: 100 });
    assert.deepEqual(rt.dispatched, []);
    at(0).dispatch("pointerup", { pointerId: 1, clientX: 120, clientY: 100 });

    assert.deepEqual(rt.dispatched, [{ id: "txt_cb_0", payload: { value: "leading" } }]);
});

test("a secondary-button drag is not a swipe", () => {
    const { rt, at } = mount([box({ onSwipe: "txt_cb_0" })]);

    at(0).dispatch("pointerdown", { pointerId: 1, button: 2, clientX: 200, clientY: 100 });
    at(0).dispatch("pointerup", { pointerId: 1, clientX: 100, clientY: 100 });

    assert.deepEqual(rt.dispatched, []);
});

test("the swipe follows its prop through patches", () => {
    const { rt, at } = mount([box({})]);
    const setProps = (Changes) =>
        rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root/0", Changes }]));

    // Gained on a later pass: wired, and given its touch-action.
    setProps({ onSwipe: "txt_cb_4" });
    assert.equal(at(0).style.touchAction, "pan-y pinch-zoom");
    drag(at(0), 200, 100, 100, 100);

    // Renumbered: the release dispatches the new ID.
    setProps({ onSwipe: "txt_cb_7" });
    drag(at(0), 200, 100, 100, 100);

    // Lost: no dispatch, and the browser gets every pan back.
    setProps({});
    drag(at(0), 200, 100, 100, 100);
    assert.equal(at(0).style.touchAction, "");

    assert.deepEqual(rt.dispatched.map((d) => d.id), ["txt_cb_4", "txt_cb_7"]);
});

test("a style patch keeps the swipe's touch-action", () => {
    // styleFromGrMob restates every property it manages and clears the rest
    // of those; touch-action is not one of them, so a restyle (a Gallery
    // card's flip) must not hand the axis back to the browser.
    const { rt, at } = mount([box({ onSwipe: "txt_cb_0" })]);

    rt.GrMob.patch(JSON.stringify([
        { Type: "update-style", TargetID: "root/0", Changes: { Opacity: 0.5 } },
    ]));

    assert.equal(at(0).style.touchAction, "pan-y pinch-zoom");
});

test("a drag past the slop disarms a long press", () => {
    // Both native recognizers stop treating a press as a long press once it
    // travels; without this a slow swipe across a Gallery card's middle zone
    // zoomed the card as well.
    const { rt, at } = mount([box({ onLongPress: "cb_0" })]);

    at(0).dispatch("pointerdown", { clientX: 100, clientY: 100 });
    at(0).dispatch("pointermove", { clientX: 115, clientY: 100 });
    rt.drainTimers(500);
    assert.deepEqual(rt.dispatched, []);

    // A press that only jitters still is one.
    at(0).dispatch("pointerdown", { clientX: 100, clientY: 100 });
    at(0).dispatch("pointermove", { clientX: 104, clientY: 103 });
    rt.drainTimers(500);
    assert.deepEqual(rt.dispatched, [{ id: "cb_0", payload: {} }]);
});
