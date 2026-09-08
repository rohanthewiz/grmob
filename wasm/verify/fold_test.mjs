// Every arm of the fold guard, reached without bundling more themes.
//
// The guard exists because a grid pushed past the bottom of the window has a
// rect and no pixels; it has never fired, because nine bands and six chips fit
// an 800px window comfortably and the only way to trip the band one is to add
// themes. That is the shape gate.sh and composeSourcesVerdict were both moved
// out of: a decision whose arms are reachable only by owning a machine, or a
// repository, in the state that has the fault.
import test from "node:test";
import assert from "node:assert/strict";

import { foldVerdict, FOLD_EPSILON } from "./fold.mjs";

const call = (bottom, screen) => foldVerdict({
    what: "the swatch grid", bottom, screen, knob: "WIDGETS_PER_ROW",
});

test("a grid well inside the viewport says nothing", () => {
    assert.equal(call(400, 800), "");
});

test("a grid ending exactly at the fold is inside it", () => {
    // The arm the tolerance is for: a rect in CSS pixels against an image
    // height divided by the device pixel ratio, which is the same number
    // arrived at two ways and can differ in the last place.
    assert.equal(call(800, 800), "");
    assert.equal(call(800 + FOLD_EPSILON, 800), "");
});

test("a grid past the fold is reported, with both numbers and the knob", () => {
    const why = call(920, 800);
    assert.notEqual(why, "", "a grid 120px past the bottom was not reported");
    for (const part of ["920px", "800px", "WIDGETS_PER_ROW"]) {
        assert.ok(why.includes(part),
            `the verdict does not mention ${part}, so a reader is told something is ` +
            `wrong and not what to turn: ${why}`);
    }
});

test("a capture that came back empty takes the same arm", () => {
    // Not a hypothetical distinct from the one above so much as the one state
    // where every other assertion in the check fails for the wrong reason:
    // pixelAt answers null for every sample and each one reports a colour.
    assert.notEqual(call(100, 0), "");
});

test("the verdict names the grid it is about", () => {
    const why = foldVerdict({
        what: "the band grid", bottom: 900, screen: 800, knob: "the window size",
    });
    assert.ok(why.startsWith("the band grid:"), why);
    assert.ok(why.includes("the window size"), why);
});
