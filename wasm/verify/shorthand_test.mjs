// dom.mjs's inline style, held to what a real CSSStyleDeclaration does with
// shorthands.
//
// cssstyle.mjs models one piece of the CSSOM — shorthands and their longhands
// are two views of the same declarations — because styleFromGrMob's totality
// rule can break it and a plain object could not notice (the gap and TextGrid
// overflow bugs; see that file's header). A model of a browser fact is only as
// good as its agreement with a browser, so the rows below are sequences of
// assignments and the reads a real Chrome returned for them.
//
// # Where the expected values came from
//
// Each row was run as written against document.createElement("div").style in
// Chrome while cssstyle.mjs was being written, and the values here are
// Chrome's. The reads are chosen where Chrome and the model are meant to agree
// exactly: longhand values, the empty string a browser returns for a
// shorthand it can no longer serialize, and serializations after a longhand
// changed. A shorthand read back unchanged is left out on purpose — Chrome
// returns its own shortest spelling and the model returns the author's text,
// a difference cssstyle.mjs states and the other suites depend on.
//
// To re-check a row after changing the model, paste its sets into a browser
// console against a fresh element's style and compare the reads.

import test from "node:test";
import assert from "node:assert/strict";

import { makeStyle, erasedShorthands, isShorthand } from "./cssstyle.mjs";
import { loadRuntime, nodeAt } from "./load.mjs";

const CSSOM = [
    // The two bugs, as the CSSOM sees them.
    { sets: [["gap", "12px"], ["rowGap", ""]], reads: { gap: "", rowGap: "", columnGap: "12px" } },
    { sets: [["gap", "12px"], ["rowGap", ""], ["columnGap", ""]], reads: { gap: "" } },
    { sets: [["overflow", "hidden"], ["overflowX", ""]], reads: { overflow: "", overflowX: "", overflowY: "hidden" } },

    { sets: [["overflow", "hidden"], ["overflowX", "auto"]], reads: { overflow: "auto hidden", overflowY: "hidden" } },
    { sets: [["overflow", "hidden"], ["overflowX", "hidden"]], reads: { overflow: "hidden" } },
    { sets: [["padding", "1px 2px"], ["paddingLeft", "5px"]], reads: { padding: "1px 2px 1px 5px", paddingTop: "1px", paddingRight: "2px", paddingBottom: "1px" } },
    { sets: [["padding", "1px"], ["paddingTop", ""]], reads: { padding: "", paddingRight: "1px" } },
    { sets: [["padding", "1px 2px"], ["paddingLeft", "2px"]], reads: { padding: "1px 2px" } },
    { sets: [["margin", "1px 2px 3px"]], reads: { marginLeft: "2px", marginBottom: "3px", marginRight: "2px" } },
    { sets: [["inset", "1px"], ["left", ""]], reads: { inset: "", top: "1px" } },
    { sets: [["border", "1px solid red"], ["borderBottomColor", "blue"]], reads: { border: "", borderBottom: "1px solid blue", borderTopWidth: "1px", borderTopStyle: "solid", borderBottomColor: "blue" } },
    // The tab strip's own sequence (TAB_STYLE in grmob-runtime.js).
    { sets: [["border", "none"], ["borderBottom", "2px solid transparent"]], reads: { borderTopStyle: "none", borderTopWidth: "medium", borderTopColor: "currentcolor", borderBottomWidth: "2px", border: "" } },
    { sets: [["border", "1px solid red"], ["border", ""]], reads: { borderTop: "", borderLeftColor: "" } },
    { sets: [["borderTop", "2px solid"]], reads: { borderTopColor: "currentcolor", borderTopWidth: "2px" } },
    { sets: [["borderRadius", "8px"], ["borderTopLeftRadius", ""]], reads: { borderRadius: "", borderBottomRightRadius: "8px" } },
    { sets: [["flex", "1 1 0px"], ["flexGrow", "2"]], reads: { flex: "2 1 0px", flexShrink: "1", flexBasis: "0px" } },
    { sets: [["flex", "none"]], reads: { flexGrow: "0", flexShrink: "0", flexBasis: "auto" } },
    // The overlay's stamp (OVERLAY_CHILD_AREA).
    { sets: [["gridArea", "1/1"]], reads: { gridRowStart: "1", gridColumnStart: "1", gridRowEnd: "auto", gridColumnEnd: "auto" } },
    // The code buffer's and the tab's `font: inherit` followed by a longhand.
    { sets: [["font", "inherit"], ["fontWeight", "600"]], reads: { font: "", fontSize: "inherit", fontWeight: "600" } },
];

for (const { sets, reads } of CSSOM) {
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
