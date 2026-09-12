// dom.mjs's inline style, held to what a real CSSStyleDeclaration does with
// shorthands.
//
// cssstyle.mjs models one piece of the CSSOM — shorthands and their longhands
// are two views of the same declarations — because styleFromGrMob's totality
// rule can break it and a plain object could not notice (the gap and TextGrid
// overflow bugs; see that file's header). A model of a browser fact is only as
// good as its agreement with a browser, so the rows it is held to are
// sequences of assignments and the reads a real Chrome returned for them.
//
// # Where the expected values came from
//
// CSSOM_READS, in cssstyle.mjs. Each row was first run as written against
// document.createElement("div").style in a Chrome console while the model was
// being written; browser.mjs check 14 now replays every row in headless Chrome
// on each run of run.sh, so a row is re-checked by adding it rather than by
// pasting it into a console. The table sits beside the model because both of
// those readers need it and neither is a test file the other could import.

import test from "node:test";
import assert from "node:assert/strict";

import { makeStyle, erasedShorthands, isShorthand, CSSOM_READS } from "./cssstyle.mjs";
import { loadRuntime, nodeAt } from "./load.mjs";

for (const { sets, reads } of CSSOM_READS) {
    const label = sets.map(([p, v]) => `${p}=${JSON.stringify(v)}`).join(", ");
    test(`CSSOM: ${label}`, () => {
        const style = makeStyle();
        for (const [p, v] of sets) style[p] = v;
        for (const [p, want] of Object.entries(reads)) {
            assert.equal(style[p], want, `${p} after ${label}`);
        }
    });
}

test("a property nothing wrote reads undefined, and one that was cleared reads empty", () => {
    // The deliberate difference from a browser, which returns "" for both:
    // the suites tell "never touched" from "cleared".
    const style = makeStyle();
    assert.equal(style.gap, undefined);
    assert.equal(style.gridArea, undefined);
    style.rowGap = "4px";
    assert.equal(style.gap, "", "half a gap cannot be serialized as one");
    style.rowGap = "";
    assert.equal(style.rowGap, "");
});

test("enumeration is what was assigned, not what an assignment expanded into", () => {
    const style = makeStyle();
    style.padding = "1px";
    style.color = "red";
    assert.deepEqual(Object.keys(style), ["padding", "color"]);
});

test("erasedShorthands names a shorthand a later empty longhand cut into, and not an override", () => {
    const erased = makeStyle();
    erased.overflow = "hidden";
    erased.overflowX = "";
    assert.deepEqual(erasedShorthands(erased), ["overflow"]);

    const overridden = makeStyle();
    overridden.border = "none";
    overridden.borderBottom = "2px solid transparent";
    assert.deepEqual(erasedShorthands(overridden), []);

    // A shorthand cleared on purpose is not an erasure either.
    const cleared = makeStyle();
    cleared.gap = "8px";
    cleared.gap = "";
    assert.deepEqual(erasedShorthands(cleared), []);
});

test("a value the model does not cover throws instead of guessing", () => {
    assert.throws(() => { makeStyle().borderRadius = "10px / 20px"; }, /cannot expand borderRadius/);
    assert.throws(() => { makeStyle().font = "bold 14px/1.2 serif"; }, /cannot expand font/);
});

// The second instance of the bug class, by name. totality_test.mjs's erasure
// sweep is what finds the class; this pins the grid's own contract on all
// three paths an Overflow reaches it by.
test("a TextGrid keeps both halves of its Overflow, and scrolls sideways without one", () => {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "Column",
        Children: [{ Type: "TextGrid", Style: { Overflow: "hidden" }, Children: [] }],
    }));
    rt.drainFrames();
    const grid = nodeAt(rt.document, "root/0");
    assert.equal(grid.style.overflowX, "hidden", "the x half of Overflow was erased");
    assert.equal(grid.style.overflowY, "hidden");

    const restyle = (changes) => {
        rt.GrMob.patch(JSON.stringify([{ Type: "update-style", TargetID: "root/0", Changes: changes }]));
        rt.drainFrames();
    };
    restyle({});
    assert.equal(grid.style.overflowX, "auto", "dropping Overflow should restore the grid's sideways scroll");
    assert.equal(grid.style.overflowY, "");

    restyle({ Overflow: "scroll" });
    assert.equal(grid.style.overflow, "scroll");
});

test("an unexpanded shorthand mixed with its own longhand throws", () => {
    const style = makeStyle();
    style.background = "none";
    assert.throws(() => { style.backgroundColor = "red"; }, /both background and its longhand backgroundColor/);
    // Unrelated properties are fine beside it.
    style.color = "red";
    assert.equal(isShorthand("background"), false);
});
