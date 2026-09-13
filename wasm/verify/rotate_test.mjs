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

// core.Spin: an animation list entry, the keyframes it names, and the same
// totality rule as Rotate when it stops.

test("a spinning node names grmob-spin and the document gains its keyframes once", () => {
    const { rt, at } = mount([box({ Spin: 1000, Rotate: 30 }), box({ Spin: -400 })]);
    assert.equal(at(0).style.animation, "grmob-spin 1000ms linear infinite");
    assert.equal(at(1).style.animation, "grmob-spin 400ms linear infinite reverse");
    // Composition: the fixed angle stays on transform under the spin.
    assert.equal(at(0).style.transform, "rotate(30deg)");

    const sheets = rt.document.head.children;
    assert.equal(sheets.length, 1, "two spinning nodes should share one stylesheet");
    assert.equal(sheets[0].tagName.toLowerCase(), "style");
    assert.equal(sheets[0].textContent,
        "@keyframes grmob-spin{from{rotate:0deg}to{rotate:360deg}}");
});

// core.Transition: the reduced-motion rule, once, in a sheet of its own so a
// page that only spins keeps exactly the one sheet the test above counts.
test("a transitioning node adds the reduced-motion rule once, beside the spin's", () => {
    const { rt, at } = mount([
        box({ Transition: "250ms ease" }),
        box({ Transition: "100ms linear", Spin: 1000 }),
    ]);
    assert.equal(at(0).style.transition, "all 250ms ease");
    const texts = rt.document.head.children.map((s) => s.textContent);
    assert.equal(texts.length, 2, "one sheet per rule, however many nodes need it");
    assert.ok(texts.includes(
        `@media (prefers-reduced-motion:reduce){[style*="transition"]{transition:none!important}}`));
    assert.ok(texts.includes("@keyframes grmob-spin{from{rotate:0deg}to{rotate:360deg}}"));
});

test("a still page adds no stylesheet", () => {
    const { rt } = mount([box({ Rotate: 10 })]);
    assert.equal(rt.document.head.children.length, 0);
});

test("an author Animation shares the list after the spin", () => {
    const { at } = mount([box({ Spin: 900, Animation: "pulse 2s infinite" })]);
    assert.equal(at(0).style.animation, "grmob-spin 900ms linear infinite, pulse 2s infinite");
});

test("stopping a spin clears the animation on the live element", () => {
    // A Spinner that is re-styled still, or a node whose role style stops
    // spinning, arrives as an update-style patch with no Spin key.
    const { rt, at } = mount([box({ Spin: 1000 })]);
    rt.GrMob.patch(JSON.stringify([
        { Type: "update-style", TargetID: "root/0", Changes: {} },
    ]));
    rt.drainFrames();
    assert.equal(at(0).style.animation, "", "the node kept spinning after its Spin was removed");
});
