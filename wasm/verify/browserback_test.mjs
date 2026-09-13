// The runtime's browser back: core.OnBack claims the browser's back button by
// keeping one history entry of the runtime's own while a claim is on screen.
//
// The harness has no history, so each test installs a model of one that
// keeps the part the runtime depends on — a stack of entries with state, a
// pushState that truncates Forward, and a back() whose popstate arrives later
// rather than inside the call, which is the browser's timing and the reason
// the runtime has to recognise its own unwind.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

// withHistory installs a history model on rt and returns handles to drive it.
function withHistory(rt, { navigationAPI = false } = {}) {
    const base = "http://page/";
    const entries = [{ state: null, url: base }];
    let index = 0;
    const queued = [];
    const land = (i) => {
        index = i;
        rt.fireWindowEvent("popstate", { state: entries[index].state });
    };
    // An omitted URL keeps the entry's, as pushState and replaceState do.
    const resolve = (url, fallback) => (url === undefined || url === null ? fallback : new URL(url, fallback).href);
    rt.sandbox.location = { get href() { return entries[index].url; } };
    rt.sandbox.history = {
        get state() { return entries[index].state; },
        pushState(state, _title, url) {
            const next = resolve(url, entries[index].url);
            entries.splice(index + 1);
            // Structured-cloned in a browser; a JSON copy is the same here.
            entries.push({ state: JSON.parse(JSON.stringify(state)), url: next });
            index++;
        },
        replaceState(state, _title, url) { entries[index] = { state, url: resolve(url, entries[index].url) }; },
        back() { if (index > 0) queued.push(() => land(index - 1)); },
        // Browsers cap this; the model does not, so the fallback is exercised
        // on the terms it is exact under.
        get length() { return entries.length; },
    };
    // The Navigation API's one field the runtime reads. Installed only when
    // asked, so the history.length fallback has tests of its own.
    if (navigationAPI) rt.sandbox.navigation = { get currentEntry() { return { index }; } };
    return {
        url: () => entries[index].url,
        depth: () => index + 1,
        onRuntimeEntry: () => !!(entries[index].state && entries[index].state.grmobBack),
        // The user's back press: the browser moves first, then reports.
        pressBack() { land(index - 1); },
        pressForward() { land(index + 1); },
        // A hash typed into the address bar: a new entry above the current one,
        // then the popstate a same-document navigation fires.
        typeURL(url) {
            entries.splice(index + 1);
            entries.push({ state: null, url: new URL(url, entries[index].url).href });
            land(index + 1);
        },
        // Delivers the popstate of a history.back() the runtime asked for.
        settle() { while (queued.length) queued.shift()(); },
    };
}

const text = (s) => ({ Type: "Text", Props: { content: s } });

function mount(tree) {
    const rt = loadRuntime();
    const h = withHistory(rt);
    rt.GrMob.mount(JSON.stringify(tree));
    return { rt, h };
}

test("a screen with no claim leaves history alone", () => {
    const { rt, h } = mount({ Type: "Column", Children: [text("home")] });
    assert.equal(h.depth(), 1);
    assert.equal(rt.windowListenerCount("popstate"), 1, "the listener is installed on the first sync regardless");
});

test("a claim on screen pushes one entry, and back runs it", () => {
    const { rt, h } = mount({ Type: "Column", Props: { onBack: "back_cb_0" }, Children: [text("detail")] });
    assert.equal(h.depth(), 2);
    assert.ok(h.onRuntimeEntry());

    h.pressBack();
    assert.deepEqual(rt.dispatched, [{ id: "back_cb_0", payload: {} }]);
    // No patch came back (the harness's "Go" renders nothing), so the claim is
    // still on screen and the entry must be there for the next press.
    assert.equal(h.depth(), 2, "a claim still on screen after back must re-arm");
});

test("a claim removed by a patch takes its entry out without dispatching", () => {
    const { rt, h } = mount({ Type: "Column", Props: { onBack: "back_cb_0" }, Children: [text("detail")] });
    rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root", Changes: {} }]));
    h.settle();

    assert.equal(h.depth(), 1, "an in-app pop must not leave a dead entry for the next back to consume");
    assert.deepEqual(rt.dispatched, [], "the runtime's own unwind is not a back press");
});

test("the innermost claim runs: a descendant, and a later layer", () => {
    const { rt, h } = mount({
        Type: "ZStack",
        Props: { onBack: "back_cb_0" },
        Children: [
            { Type: "Column", Children: [{ Type: "Row", Props: { onBack: "back_cb_1" }, Children: [text("bar")] }] },
            { Type: "Box", Props: { onBack: "back_cb_2" }, Children: [text("drawer")] },
        ],
    });
    h.pressBack();
    assert.deepEqual(rt.dispatched.map((d) => d.id), ["back_cb_2"]);
    assert.ok(nodeAt(rt.document, "root/1"));
});

test("a closed Modal claims nothing, an open one outranks its screen", () => {
    const modal = (visible) => ({
        Type: "Column",
        Props: { onBack: "back_cb_0" },
        Children: [{
            Type: "Modal",
            Props: { visible, onDismiss: "cb_0" },
            Children: [{ Type: "Box", Props: { onBack: "back_cb_1" }, Children: [text("inside")] }],
        }],
    });

    const closed = mount(modal(false));
    closed.h.pressBack();
    assert.deepEqual(closed.rt.dispatched.map((d) => d.id), ["back_cb_0"],
        "a claim inside a closed Modal is hidden, and so is the Modal's dismiss");

    const open = mount(modal(true));
    open.h.pressBack();
    assert.deepEqual(open.rt.dispatched.map((d) => d.id), ["back_cb_1"],
        "inside an open Modal the innermost claim is its content's");
});

test("Forward onto the runtime's entry runs nothing", () => {
    const { rt, h } = mount({ Type: "Column", Props: { onBack: "back_cb_0" }, Children: [text("detail")] });
    rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root", Changes: {} }]));
    h.settle();
    h.pressForward();
    h.settle();

    assert.deepEqual(rt.dispatched, []);
    assert.equal(h.depth(), 1, "with no claim on screen the entry Forward reached is unwound again");
});

test("a page that owns its history opts out", () => {
    const rt = loadRuntime();
    const h = withHistory(rt);
    rt.window.GrMobBrowserBack = false;
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Props: { onBack: "back_cb_0" }, Children: [] }));

    assert.equal(h.depth(), 1);
    assert.equal(rt.windowListenerCount("popstate"), 0);
});

// The reconciler sends a node's whole new props map, which is null when the
// node has none left. The first transition to do that was back itself: a
// lesson root carrying only onBack, replaced by a contents root carrying
// nothing. The batch threw on the null, and the claim stayed on screen.
test("a node that loses its only prop (Changes null) ends its claim", () => {
    const { rt, h } = mount({ Type: "Column", Props: { onBack: "back_cb_0" }, Children: [text("detail")] });
    rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root", Changes: null }]));
    h.settle();

    assert.equal(nodeAt(rt.document, "root").dataset.listener_onBack, undefined);
    assert.equal(h.depth(), 1);
});

// A deep-linked boot starts with the page's entry and the runtime's on the same
// URL. The page clears the lesson from the runtime's entry as ‹ Contents pops,
// and the unwind then lands on the page's entry, which would still name the
// lesson and make a hashchange-routed page open it again.
test("the unwind carries the page's latest URL down to the entry it lands on", () => {
    const { rt, h } = mount({ Type: "Column", Props: { onBack: "back_cb_0" }, Children: [text("lesson")] });
    rt.sandbox.history.replaceState(rt.sandbox.history.state, "", "#2.3"); // the page reports the lesson
    assert.equal(h.url(), "http://page/#2.3");

    rt.sandbox.history.replaceState(rt.sandbox.history.state, "", "/"); // ‹ Contents clears it
    rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root", Changes: null }]));
    h.settle();

    assert.equal(h.depth(), 1);
    assert.equal(h.url(), "http://page/", "the page's entry must not keep the lesson's URL");
});

// A hash typed while a claim is on screen is a new entry above the runtime's,
// and its popstate is not a back press. Before the fold it ran the claim and
// left [page, runtime, typed, runtime], so the screen took an extra back press
// to leave. Run with and without the Navigation API, which are the two ways the
// runtime tells "above" from "below".
for (const navigationAPI of [false, true]) {
    const how = navigationAPI ? "the Navigation API" : "history.length";

    test(`a typed hash runs no claim and folds into the runtime's entry (${how})`, () => {
        const rt = loadRuntime();
        const h = withHistory(rt, { navigationAPI });
        rt.GrMob.mount(JSON.stringify({ Type: "Column", Props: { onBack: "back_cb_0" }, Children: [text("lesson")] }));
        assert.equal(h.depth(), 2);

        h.typeURL("#4.18");
        assert.deepEqual(rt.dispatched, [], "a pushed entry is not a back press");
        h.settle();

        assert.equal(h.depth(), 2, "the runtime's entry must be on top again, with nothing between it and the page");
        assert.ok(h.onRuntimeEntry());
        assert.equal(h.url(), "http://page/#4.18", "the entry left on top names what the address bar showed");

        h.pressBack();
        assert.deepEqual(rt.dispatched.map((d) => d.id), ["back_cb_0"], "one back press runs the claim");
        assert.equal(h.depth(), 2, "the claim is still on screen in the harness, so the entry re-arms");
    });

    test(`Forward onto a folded entry is folded again, not read as back (${how})`, () => {
        const rt = loadRuntime();
        const h = withHistory(rt, { navigationAPI });
        rt.GrMob.mount(JSON.stringify({ Type: "Column", Props: { onBack: "back_cb_0" }, Children: [text("lesson")] }));
        h.typeURL("#4.18");
        h.settle();

        h.pressForward();
        h.settle();
        assert.deepEqual(rt.dispatched, []);
        assert.ok(h.onRuntimeEntry());
        assert.equal(h.depth(), 2);
    });
}

// The mark follows the entry out: after an in-app pop has unwound it, a later
// popstate that lands off it is judged as before, not against a stale mark.
test("an unwound entry leaves no mark behind to misjudge a later back", () => {
    const { rt, h } = mount({ Type: "Column", Props: { onBack: "back_cb_0" }, Children: [text("detail")] });
    rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root", Changes: {} }]));
    h.settle();
    assert.equal(h.depth(), 1);

    h.typeURL("#other");
    h.settle();
    assert.deepEqual(rt.dispatched, []);
    assert.equal(h.depth(), 2, "with no claim on screen a typed hash is the page's own entry, left alone");
});
