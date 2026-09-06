// core.Rotate against the live DOM.
//
// htmlout writes the same declaration into a static document and is tested
// there; this is the live half, where the interesting property is the one a
// static export cannot have — that a node which *stops* being rotated has its
// old transform removed rather than left standing on the reused element.
//
// That is the whole reason styleFromGrMob is total: an update-style patch
// carries the entire new Style, so a field back at its zero value means
// "unset now". A compass whose bearing passes through north sends Rotate: 0,
// and a guarded write would stick the dial one frame short and never return.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

function mount(children) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: children }));
    rt.drainFrames();
    return { rt, at: (i) => nodeAt(rt.document, `root/${i}`) };
}

const box = (style) => ({ Type: "Box", Style: style });

test("a rotated node gets a transform", () => {
    const { at } = mount([box({ Rotate: 123.4 })]);
    assert.equal(at(0).style.transform, "rotate(123.4deg)");
});

test("an unrotated node gets no transform", () => {
    // Not an identity rotate(0deg): a transform declaration, even an identity
    // one, makes the element a containing block for fixed and absolutely
    // positioned descendants. "No rotation" has to mean no declaration.
    const { at } = mount([box({})]);
    assert.equal(at(0).style.transform, "");
});

test("the winding survives; the runtime does not fold the angle", () => {
    const { at } = mount([box({ Rotate: -90 }), box({ Rotate: 370 })]);
    assert.equal(at(0).style.transform, "rotate(-90deg)");
    assert.equal(at(1).style.transform, "rotate(370deg)");
});

test("returning to zero clears the transform on the live element", () => {
    // The patch path reuses the element, so this is the case a guarded write
    // gets wrong — and it is the case a compass hits every time it passes
    // north.
    const { rt, at } = mount([box({ Rotate: 45 })]);
    assert.equal(at(0).style.transform, "rotate(45deg)");

    rt.GrMob.patch(JSON.stringify([
        { Type: "update-style", TargetID: "root/0", Changes: { Rotate: 0 } },
    ]));
    rt.drainFrames();
    assert.equal(at(0).style.transform, "",
        "the dial stuck at its last angle instead of returning to north");
});
