// Go's value written into a focused text field (writeFieldValue in
// grmob-runtime.js): the caret stays with the text around it, by the rule the
// native hosts' rewrites follow. Before it, a focused <input> took the
// assignment's default and sent the caret to the end, so a transform in
// onChange broke typing mid-text.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

// Go's value written into the focused field keeps the caret with the text
// around it (writeFieldValue). dom.mjs models the selection as two plain
// properties that nothing else moves, so every case here would read the old
// offset if the runtime did not move it itself.
function focusedField(value, caret) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: [{ Type: "Input", Props: { value } }] }));
    const el = nodeAt(rt.document, "root/0");
    el.focus();
    el.selectionStart = caret;
    el.selectionEnd = caret;
    const write = (v) => rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root/0", Changes: { value: v } }]));
    return { el, write };
}

test("a rewrite before the caret shifts it by the change in length", () => {
    const { el, write } = focusedField("abc", 3);
    write("xxabc");
    assert.equal(el.selectionStart, 5);
    assert.equal(el.selectionEnd, 5);
});

test("a rewrite after the caret leaves it where it is", () => {
    const { el, write } = focusedField("hello world", 2);
    write("hello worlds");
    assert.equal(el.selectionStart, 2);
});

test("a length-kept rewrite around the caret leaves it where it is", () => {
    // UPPERCASE: "HELLOa WORLD" with the caret after the a.
    const { el, write } = focusedField("HELLOa WORLD", 6);
    write("HELLOA WORLD");
    assert.equal(el.selectionStart, 6);
    // Capitalizing words: one span from the h to the w, the caret inside it.
    const second = focusedField("hellox world", 6);
    second.write("Hellox World");
    assert.equal(second.el.selectionStart, 6);
});

test("a rewrite that shortens text around the caret puts it at the end of Go's text", () => {
    // A committed tag: "beta,ga" becomes "ga"; the caret sat in the removed part.
    const { el, write } = focusedField("beta,ga", 3);
    write("ga");
    assert.equal(el.selectionStart, 0);
});

test("a field that is not focused gets the value and no caret move", () => {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: [{ Type: "Input", Props: { value: "abc" } }] }));
    const el = nodeAt(rt.document, "root/0");
    el.selectionStart = 3;
    rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root/0", Changes: { value: "xxabc" } }]));
    assert.equal(el.value, "xxabc");
    assert.equal(el.selectionStart, 3);
});
