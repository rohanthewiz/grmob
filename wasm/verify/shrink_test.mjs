// A flex-shrink factor of zero, against the minimal DOM.
//
// # The one number in a core.Style whose zero is not its own value
//
// Every optional number in a core.Style means "unset" by being zero, and that
// works because their CSS initial values are zero too — an unset Gap and a Gap
// of 0 lay out identically. flex-shrink's initial value is 1, so zero and unset
// are two different layouts: unset shrinks under pressure, zero keeps its size
// and lets the container overflow.
//
// So `core.FlexShrink(0)` stores core.ShrinkNone (-1), and this runtime is one
// of the three places that has to know it. The guard here used to be a bare
// truthiness test, which turned "do not shrink" into no declaration at all —
// the opposite instruction, delivered silently.
//
// styleFromGrMob is total: every property it manages is assigned on every call,
// because an update-style patch carries the whole new Style and a field back at
// its zero value means "unset now". That is what makes the third test below the
// one worth having — clearing the factor has to restore the default rather than
// leave the pin in place.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

// core.ShrinkNone, spelled here because a mounted tree is JSON and cannot call
// Go. wasm/verify/shrink_test.go holds this number to the constant.
const SHRINK_NONE = -1;

function mount(style) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "Row",
        Children: [{ Type: "Box", Style: style }],
    }));
    rt.drainFrames();
    return { rt, el: nodeAt(rt.document, "root/0") };
}

test("a shrink factor of zero writes flex-shrink:0", () => {
    const { el } = mount({ FlexShrink: SHRINK_NONE });
    assert.equal(el.style.flexShrink, "0",
        "core.FlexShrink(0) arrives as core.ShrinkNone and must be written as a " +
        "factor of 0. A runtime that reads it as unset writes nothing, and the " +
        "browser's own initial value of 1 shrinks the item the author pinned.");
});

test("an unset shrink factor writes no declaration", () => {
    const { el } = mount({});
    assert.equal(el.style.flexShrink, "",
        "a node that never mentioned flex-shrink must not be given one: the CSS " +
        "initial value is 1, and writing it explicitly for every node in the tree " +
        "would be a declaration nobody asked for on every element.");
});

test("an ordinary factor is written as itself", () => {
    const { el } = mount({ FlexShrink: 2 });
    assert.equal(el.style.flexShrink, "2");
});

// The total-reassignment half. An update-style patch carries the whole Style,
// so a factor that goes away has to take the declaration with it — otherwise a
// node pinned once stays pinned for the rest of its life.
test("clearing the factor clears the declaration", () => {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "Row",
        Children: [{ Type: "Box", Style: { FlexShrink: SHRINK_NONE } }],
    }));
    rt.drainFrames();
    assert.equal(nodeAt(rt.document, "root/0").style.flexShrink, "0");

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-style",
        TargetID: "root/0",
        Changes: {},
    }]));
    rt.drainFrames();
    assert.equal(nodeAt(rt.document, "root/0").style.flexShrink, "",
        "the item is no longer pinned and the declaration outlived the Style that " +
        "asked for it");
});
