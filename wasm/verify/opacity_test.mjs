// core.Opacity against the live DOM.
//
// htmlout writes the same declaration into a static document and is tested
// there. This is the live half, and it has the two cases a static export
// cannot: the sentinel arriving in a patch, and a node that stops being faded
// having its old declaration removed from the reused element.
//
// The sentinel is the point of the first. An alpha of zero crosses the wire
// as core.OpacityClear (-1), because a plain zero is what an unset field looks
// like. A runtime that wrote the number through would hand CSS "-1", which it
// clamps to 0 and so looks right, until the day the sentinel is read as
// "falsy means unset" and a faded-out node is drawn opaque. So the string is
// pinned, not the pixels. wasm/verify's opacity_test.go holds the number
// itself to core.OpacityClear.

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

test("a faded node gets its alpha, and the sentinel is written as zero", () => {
    const { at } = mount([box({ Opacity: 0.4 }), box({ Opacity: 1 }), box({ Opacity: -1 })]);
    assert.equal(at(0).style.opacity, "0.4");
    assert.equal(at(1).style.opacity, "1");
    assert.equal(at(2).style.opacity, "0",
        "core.OpacityClear must reach the page as an alpha of zero");
});

test("an undeclared opacity writes nothing", () => {
    const { at } = mount([box({})]);
    assert.equal(at(0).style.opacity, "");
});

test("fading out and back in over patches, ending with no declaration", () => {
    // The patch path reuses the element. A node that fades back in by
    // dropping the prop sends a Style with no Opacity, and a guarded write
    // would leave it transparent for good.
    const { rt, at } = mount([box({ Opacity: 0.4, Transition: "200ms ease" })]);
    assert.equal(at(0).style.opacity, "0.4");

    rt.GrMob.patch(JSON.stringify([
        { Type: "update-style", TargetID: "root/0", Changes: { Opacity: -1, Transition: "200ms ease" } },
    ]));
    rt.drainFrames();
    assert.equal(at(0).style.opacity, "0");

    rt.GrMob.patch(JSON.stringify([
        { Type: "update-style", TargetID: "root/0", Changes: { Transition: "200ms ease" } },
    ]));
    rt.drainFrames();
    assert.equal(at(0).style.opacity, "",
        "the node stayed transparent after its Opacity was dropped");
});
