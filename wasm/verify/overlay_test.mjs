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

test("a layer names its own corner and the rest stay centred", () => {
    const { layer } = mountStack({ Width: "160px" }, [
        text("under"),
        { Type: "Text", Props: { content: "N" }, Style: { StackAlign: "top" } },
        { Type: "Text", Props: { content: "SE" }, Style: { StackAlign: "bottom-end" } },
    ]);

    // The unplaced layer is placed *explicitly* at the centre rather than left
    // to the chassis. See STACK_PLACEMENTS: the row exists so that a layer's
    // own AlignSelf cannot move it, and so that a layer losing its placement
    // comes back to the middle.
    assert.equal(layer(0).style.justifySelf, "center");
    assert.equal(layer(0).style.alignSelf, "center");

    assert.equal(layer(1).style.justifySelf, "center");
    assert.equal(layer(1).style.alignSelf, "start");

    assert.equal(layer(2).style.justifySelf, "end");
    assert.equal(layer(2).style.alignSelf, "end");
});

test("a layer's own AlignSelf does not move it", () => {
    // Style.AlignSelf is flexbox's, and a grid item honours it too — so
    // before the stack imposed a placement on every layer, this one prop
    // moved a layer on the two DOM targets and nowhere else, in contradiction
    // of the alignment contract core.ZStack documents.
    const { layer } = mountStack({ Width: "160px" }, [
        { Type: "Text", Props: { content: "a" }, Style: { AlignSelf: "flex-end" } },
    ]);

    assert.equal(layer(0).style.alignSelf, "center",
        "a flex item property placed a layer of an overlay");
});

test("a layer that loses its placement returns to the centre", () => {
    // The totality half. applyStyle drops the data attribute and the overlay
    // pass restates both properties, so nothing is left holding the old
    // corner — the failure a pass that only wrote placements it was asked for
    // would have.
    const { rt, layer } = mountStack({ Width: "160px" }, [
        { Type: "Text", Props: { content: "N" }, Style: { StackAlign: "top-start" } },
    ]);
    assert.equal(layer(0).style.justifySelf, "start");

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-style", TargetID: "root/0/0", Changes: { FontSize: 12 },
    }]));

    assert.equal(layer(0).style.justifySelf, "center");
    assert.equal(layer(0).style.alignSelf, "center");
});

test("a placement outside an overlay is inert", () => {
    // The prop is imposed by the stack, so a node that is not a layer never
    // receives it — which is what keeps a StackAlign written on the wrong node
    // from re-placing a Row's children on the web alone.
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "Column",
        Children: [{
            Type: "Row",
            Children: [{ Type: "Text", Props: { content: "a" }, Style: { StackAlign: "top-end" } }],
        }],
    }));
    rt.drainFrames();

    // Falsy rather than "": dom.mjs leaves a property that was never assigned
    // undefined, where a browser's CSSOM reports the empty string, and
    // justify-self is never assigned outside an overlay while align-self is
    // (styleFromGrMob writes it for every node, from Style.AlignSelf).
    const child = nodeAt(rt.document, "root/0/0");
    assert.ok(!child.style.justifySelf, "a StackAlign placed a node that is not a layer");
    assert.ok(!child.style.alignSelf, "a StackAlign placed a node that is not a layer");
});
