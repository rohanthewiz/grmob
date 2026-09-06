// core.ZStack against the minimal DOM.
//
// htmlout writes the same layout into a static document and is tested there;
// this is the live half, where the interesting properties are the two a static
// export cannot have — that a stack with no Style at all is still an overlay
// (createElement, not applyStyle, is what plants the chassis), and that a
// layer arriving in a *patch* is placed in the cell like the ones that were
// there at mount.
//
// The second is the one that would have shipped. An unplaced layer is not
// invisible: CSS auto-places it into its own implicit grid row, so it appears
// below the stack rather than on it, which reads as a layout quirk rather than
// as a missing declaration.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

const text = (content) => ({ Type: "Text", Props: { content } });

// Mounts a ZStack as the root's only child and hands back the stack, a layer
// accessor, and the runtime for patching.
function mountStack(style, children) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "Column",
        Children: [{ Type: "ZStack", Style: style, Children: children }],
    }));
    rt.drainFrames();
    return {
        rt,
        stack: nodeAt(rt.document, "root/0"),
        layer: (i) => nodeAt(rt.document, `root/0/${i}`),
    };
}

test("a stack is a single-cell grid and every layer is in the cell", () => {
    const { stack, layer } = mountStack({ Width: "160px" }, [text("under"), text("over")]);

    assert.equal(stack.style.display, "grid");
    // The *items* properties: they place a child inside its cell. The content
    // pair would place the single track inside the container, which on an
    // auto-sized container does nothing at all.
    assert.equal(stack.style.alignItems, "center");
    assert.equal(stack.style.justifyItems, "center");
    assert.equal(layer(0).style.gridArea, "1/1");
    assert.equal(layer(1).style.gridArea, "1/1");
});

test("a stack with no Style is still an overlay", () => {
    // core.ZStack carries no theme base, so this is the shape a stack written
    // with children and nothing else arrives in — and the one that never
    // reaches applyStyle at all.
    const { stack, layer } = mountStack(undefined, [text("under"), text("over")]);

    assert.equal(stack.style.display, "grid");
    assert.equal(stack.style.alignItems, "center");
    assert.equal(layer(0).style.gridArea, "1/1");
    assert.equal(layer(1).style.gridArea, "1/1");
});

test("a container prop does not promote a stack to a flex container", () => {
    // Gap is meaningless on an overlay — there is one cell and nothing to
    // space along — but the flex promotion test would have answered to it,
    // and the stack would have stopped overlaying entirely.
    const { stack } = mountStack({ Gap: 8, AlignItems: "flex-end" }, [text("a"), text("b")]);

    assert.equal(stack.style.display, "grid");
    assert.equal(stack.style.flexDirection, "");
    assert.equal(stack.style.alignItems, "center",
        "an AlignItems on a stack overrode the centring the alignment contract fixes");
});

test("an update-style patch does not cost a stack its grid", () => {
    // styleFromGrMob is total: it reassigns every property it manages on every
    // call, so the chassis planted at creation has to be restated there or the
    // first patch drops the stack into block flow.
    const { rt, stack } = mountStack({ Width: "160px" }, [text("under"), text("over")]);

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-style", TargetID: "root/0", Changes: { Width: "200px" },
    }]));

    assert.equal(stack.style.width, "200px");
    assert.equal(stack.style.display, "grid");
    assert.equal(stack.style.justifyItems, "center");
});

test("a layer added by a patch is placed in the cell", () => {
    const { rt } = mountStack(undefined, [text("under")]);

    rt.GrMob.patch(JSON.stringify([
        { Type: "add-child", TargetID: "root/0", Changes: text("over") },
    ]));

    assert.equal(nodeAt(rt.document, "root/0/1").style.gridArea, "1/1",
        "the new layer was auto-placed into its own grid row, i.e. below the stack");
});

test("a replaced layer is placed in the cell", () => {
    // The patch names the layer, not the stack, so the post-batch pass has to
    // walk up to find the overlay above it — the same walk the TabView and
    // end-reached passes make.
    const { rt } = mountStack(undefined, [text("under"), text("over")]);

    rt.GrMob.patch(JSON.stringify([
        { Type: "replace", TargetID: "root/0/1", Changes: text("over (edited)") },
    ]));

    assert.equal(nodeAt(rt.document, "root/0/1").style.gridArea, "1/1");
});

test("an ordinary container does not place its children in a cell", () => {
    // The stamp is the overlay's, and a container that is not one must leave
    // its children alone — a grid-area on a flex item is inert, but a stray
    // one is the sign the pass is walking further than it should.
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "Column",
        Children: [{ Type: "Box", Children: [text("a"), text("b")] }],
    }));
    rt.drainFrames();

    assert.equal(nodeAt(rt.document, "root/0/0").style.gridArea, undefined);
    assert.equal(nodeAt(rt.document, "root/0").style.display, "flex");
});
