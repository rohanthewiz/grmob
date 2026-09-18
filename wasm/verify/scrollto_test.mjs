// core.ScrollIntoView on the web (core/scroll_to.go): the node carrying a
// scroll epoch higher than any the page has applied is brought into view,
// once.
//
// The minimal DOM has no layout, so what is checked is the decision (which
// element is scrolled, how often, with what options), not the scrolling. The
// options are the contract: block "nearest" is what the natives do by
// default, and core defines the command by it.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

// A runtime whose elements record scrollIntoView calls. The stub DOM has no
// isConnected either; every element here is in the document.
function harness() {
    const rt = loadRuntime();
    const calls = [];
    const proto = Object.getPrototypeOf(rt.document.createElement("div"));
    Object.defineProperty(proto, "isConnected", { configurable: true, get() { return true; } });
    proto.scrollIntoView = function (opts) { calls.push({ path: this.dataset.nodePath, opts }); };
    return { rt, calls };
}

const section = (name, epoch) => ({
    Type: "Column",
    Props: epoch ? { scrollEpoch: epoch } : {},
    Children: [{ Type: "Text", Props: { content: name } }],
});

test("a stamped node is scrolled into view after the frame, nearest, once", () => {
    const { rt, calls } = harness();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: [section("a"), section("b", 1)] }));
    assert.equal(calls.length, 0, "scrolled before layout: the element was not in the document yet");
    rt.drainFrames();
    assert.equal(calls.length, 1);
    assert.equal(calls[0].path, "root/1");
    assert.equal(calls[0].opts.block, "nearest");
    assert.equal(calls[0].opts.inline, "nearest");
});

// An update-props patch carries the whole new map, so the target's unchanged
// stamp arrives again with every re-render of it. The high-water mark makes
// that a no-op, and a higher epoch (a second command) acts again.
test("the same epoch again does nothing; a new one acts", () => {
    const { rt, calls } = harness();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: [section("a"), section("b", 1)] }));
    rt.drainFrames();
    rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root/1", Changes: { scrollEpoch: 1 } }]));
    rt.drainFrames();
    assert.equal(calls.length, 1, "a re-sent stamp scrolled again");
    rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root/0", Changes: { scrollEpoch: 2 } }]));
    rt.drainFrames();
    assert.deepEqual(calls.map(c => c.path), ["root/1", "root/0"]);
});

// The mark is app-wide: an element rebuilt later still carrying an old epoch
// (a layout switch rebuilds whole subtrees) must not pull the page back to it.
test("a rebuilt element with an old epoch does not scroll again", () => {
    const { rt, calls } = harness();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: [section("a", 3)] }));
    rt.drainFrames();
    rt.GrMob.patch(JSON.stringify([{ Type: "add", TargetID: "root/1", Changes: section("again", 3) }]));
    rt.drainFrames();
    assert.equal(calls.length, 1);
    assert.ok(nodeAt(rt.document, "root/1"), "the rebuilt element exists");
});

// A fresh mount is a fresh app, whose epochs start at 1 again: the dev
// server's hot reload swaps the Go module under this runtime.
test("a fresh mount starts the count again", () => {
    const { rt, calls } = harness();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: [section("a", 5)] }));
    rt.drainFrames();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: [section("b", 1)] }));
    rt.drainFrames();
    assert.equal(calls.length, 2);
});
