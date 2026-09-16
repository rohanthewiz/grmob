// core.MaxLines against the live DOM: the one-line and clamp spellings, the
// author's own overflow and white-space left standing, and a cap that goes
// away taking every declaration it added with it (the patch path reuses the
// element, so a guarded write would leave a label truncated forever).

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

function mount(children) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: children }));
    rt.drainFrames();
    return { rt, at: (i) => nodeAt(rt.document, `root/${i}`) };
}

const text = (style) => ({ Type: "Text", Props: { content: "a long label" }, Style: style });

test("one line is nowrap, hidden and an ellipsis", () => {
    const { at } = mount([text({ MaxLines: 1 })]);
    const s = at(0).style;
    assert.equal(s.whiteSpace, "nowrap");
    assert.equal(s.overflow, "hidden");
    assert.equal(s.textOverflow, "ellipsis");
    assert.equal(s.webkitLineClamp, "");
    assert.equal(s.display, "block");
});

test("more lines are the -webkit-box clamp", () => {
    const { at } = mount([text({ MaxLines: 3 })]);
    const s = at(0).style;
    assert.equal(s.display, "-webkit-box");
    assert.equal(s.webkitBoxOrient, "vertical");
    assert.equal(s.webkitLineClamp, "3");
    assert.equal(s.overflow, "hidden");
    assert.equal(s.textOverflow, "");
});

test("the author's overflow and white-space win", () => {
    const { at } = mount([text({ MaxLines: 1, Overflow: "clip", WhiteSpace: "pre" })]);
    assert.equal(at(0).style.overflow, "clip");
    assert.equal(at(0).style.whiteSpace, "pre");
});

test("removing the cap clears what it added", () => {
    const { rt, at } = mount([text({ MaxLines: 1 })]);
    rt.GrMob.patch(JSON.stringify([
        { Type: "update-style", TargetID: "root/0", Changes: {} },
    ]));
    rt.drainFrames();
    const s = at(0).style;
    for (const prop of ["whiteSpace", "overflow", "textOverflow", "display", "webkitLineClamp", "webkitBoxOrient"]) {
        assert.equal(s[prop], "", `${prop} stayed after the cap was removed`);
    }
});
