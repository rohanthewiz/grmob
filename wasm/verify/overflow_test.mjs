// Every rule of browser check 24's overflow judgement, reached with boxes built
// by hand.
//
// The check itself needs a Chrome, the live tutorial build and eighty lessons
// at three widths, and on today's tree it finds nothing — so a rule that had
// quietly stopped firing would look exactly like a tree with nothing wrong.
// These hold each rule's two sides: the shape it exists to catch, and the
// nearest shape it must let through.
import test from "node:test";
import assert from "node:assert/strict";

import { overflowFindings, overflowLine, OVERFLOW_EPSILON } from "./overflow.mjs";

// box fills in a record the way OVERFLOW_READ_JS would for a plain, visible,
// in-flow box that states nothing.
const box = (over) => ({
    parent: -1, path: null, type: "Box",
    l: 0, t: 0, r: 100, b: 100,
    ox: "v", oy: "v", abs: false, skip: null,
    w: null, h: null, ow: 100, oh: 100, capW: false, capH: false,
    text: "", sw: 0, cw: 0,
    ...over,
});

const kinds = (boxes, viewport = 400) =>
    overflowFindings({ viewport, boxes }).map((f) => f.kind);

test("a child inside its parent says nothing", () => {
    assert.deepEqual(kinds([
        box({ r: 300 }),
        box({ parent: 0, l: 10, r: 290, b: 50 }),
    ]), []);
});

test("an edge within a pixel of its parent's is rounding, not a spill", () => {
    assert.deepEqual(kinds([
        box({ r: 300 }),
        box({ parent: 0, r: 300 + OVERFLOW_EPSILON }),
    ]), []);
});

test("an in-flow child past a parent that shows overflow escapes", () => {
    // Lesson 1.1's stats row before 52ef3ac: the row wider than the card.
    assert.deepEqual(kinds([
        box({ r: 300 }),
        box({ parent: 0, r: 318 }),
    ]), ["escape"]);
});

test("an absolutely positioned child may hang outside its parent", () => {
    // A badge on an avatar's corner.
    assert.deepEqual(kinds([
        box({ l: 50, r: 100 }),
        box({ parent: 0, l: 90, r: 110, abs: true }),
    ]), []);
});

test("a child past an ancestor that hides overflow is clipped", () => {
    // The card's rounded corners cutting "Following" off.
    const found = overflowFindings({ viewport: 400, boxes: [
        box({ r: 300, ox: "c", oy: "c" }),
        box({ parent: 0, r: 280 }),
        box({ parent: 1, l: 250, r: 330 }),
    ] });
    assert.deepEqual(found.map((f) => f.kind).sort(), ["clipped", "escape"]);
    assert.match(found.find((f) => f.kind === "clipped").detail, /cut off by Box's x 0\.\.300/);
});

test("a child past an ancestor that scrolls is not clipped", () => {
    // A horizontal core.Scroll: content wider than the box is the point.
    assert.deepEqual(kinds([
        box({ r: 300, ox: "s" }),
        box({ parent: 0, r: 900 }),
    ]), []);
});

test("the nearest clipping ancestor decides, not an outer one", () => {
    // A scroller inside a clipping card: the card's clip never sees the
    // scroller's content, which scrolls.
    assert.deepEqual(kinds([
        box({ r: 300, ox: "c", oy: "c" }),
        box({ parent: 0, r: 300, ox: "s" }),
        box({ parent: 1, r: 900 }),
    ]), []);
});

test("off the screen is asked sideways only, and only with no clip on the way", () => {
    assert.deepEqual(kinds([box({ r: 420 })], 400), ["offscreen"]);
    assert.deepEqual(kinds([box({ l: -12, r: 100 })], 400), ["offscreen"]);
    // Taller than the window is a page that scrolls.
    assert.deepEqual(kinds([box({ b: 5000 })], 400), []);
});

test("a spill is reported at the outermost box, not at every descendant", () => {
    const found = overflowFindings({ viewport: 400, boxes: [
        box({ r: 500 }),
        box({ parent: 0, l: 380, r: 500 }),
        box({ parent: 1, l: 420, r: 480 }),
    ] });
    assert.deepEqual(found.map((f) => f.kind), ["offscreen"]);
    assert.equal(found[0].box.r, 500);
});

test("a word wider than a box that shows overflow is reported", () => {
    // 6.8's ConcernSelectRowValueNotAnOption before overflow-wrap.
    assert.deepEqual(kinds([
        box({ text: "comps.ConcernSelectRowValueNotAnOption", sw: 318, cw: 300 }),
    ]), ["text"]);
    // Clipped text is an ellipsis or a MaxLines, stated on purpose.
    assert.deepEqual(kinds([
        box({ text: "a long title", sw: 318, cw: 300, ox: "c" }),
    ]), []);
    // An inline box has no clientWidth to hold it to.
    assert.deepEqual(kinds([box({ text: "inline", sw: 318, cw: 0 })]), []);
});

test("a stated px size drawn smaller is squeezed, on either axis", () => {
    // 4.3's 36px avatar drawn 34 wide, and 1.5's 160px Scroll drawn as its
    // border.
    assert.deepEqual(kinds([box({ w: 36, ow: 34 })]), ["squeezed"]);
    assert.deepEqual(kinds([box({ h: 160, oh: 2 })]), ["squeezed"]);
    // Half a pixel short is layout rounding.
    assert.deepEqual(kinds([box({ w: 36, ow: 36 - OVERFLOW_EPSILON / 2 })]), []);
});

test("a stated cap licenses the shrink", () => {
    // comps.StaticMap: Width 320 with MaxWidth 100% in a 296px column.
    assert.deepEqual(kinds([box({ w: 320, ow: 296, capW: true })]), []);
    assert.deepEqual(kinds([box({ h: 320, oh: 296, capH: true })]), []);
});

test("a skipped box takes its whole subtree with it", () => {
    for (const skip of ["editor", "rotated", "svg", "inert"]) {
        assert.deepEqual(kinds([
            box({ r: 300, ox: "c" }),
            box({ parent: 0, l: -200, r: 40, skip }),
            box({ parent: 1, l: -180, r: 20, w: 50, ow: 10 }),
        ]), [], skip);
    }
});

test("a finding reads as where, kind, node and detail", () => {
    const [f] = overflowFindings({ viewport: 400, boxes: [
        box({ path: "root/0/2", type: "Row", text: "Following", r: 420 }),
    ] });
    assert.equal(overflowLine("360 1.1", f),
        `360 1.1: offscreen: Row root/0/2 "Following": x 0..420 past a 400px viewport`);
});
