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

// --- A keystroke Go refuses ---------------------------------------------
//
// dispatchFromElement in grmob-runtime.js. A controlled field shows what Go
// renders, and a handler that declines an edit changes no state, so the next
// render equals the last and no patch arrives. Without a rule of the page's
// own the refused key stayed drawn; headless Chrome showed comps.MaskedInput
// holding "(555) 123-4567x". The natives' edit ledger catches the same case
// as a rewrite, which is why only this host needed the rule.
//
// `go` stands in for Go: it is called inside GoInvokeCallback, as the real
// bridge applies a pass's patches before the call returns.
function typedField(nodeType, value, go) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "Column",
        Children: [{ Type: nodeType, Props: { value, onChange: "txt_1" } }],
    }));
    const el = nodeAt(rt.document, "root/0");
    const patch = (v) => rt.GrMob.patch(JSON.stringify(
        [{ Type: "update-props", TargetID: "root/0", Changes: { value: v, onChange: "txt_1" } }]));
    rt.window.GoInvokeCallback = (id, payload) => go(payload.value, patch);
    const type = (text) => {
        el.value = text;
        el.dispatch("input", {});
    };
    return { el, type };
}

test("a key Go refuses is taken back out of the field", () => {
    // Go renders nothing new: the handler ignored the edit.
    const { el, type } = typedField("Input", "(555", () => {});
    type("(555x");
    assert.equal(el.value, "(555");
});

test("a key Go accepts stays, whether Go echoes it or rewrites it", () => {
    const echo = typedField("Input", "ab", (v, patch) => patch(v));
    echo.type("abc");
    assert.equal(echo.el.value, "abc");

    // A mask: "(5556" comes back as "(555) 6".
    const mask = typedField("Input", "(555", (v, patch) => patch("(555) 6"));
    mask.type("(5556");
    assert.equal(mask.el.value, "(555) 6");
});

test("a refusal after an accepted key goes back to the accepted text, not the mounted one", () => {
    let accept = true;
    const { el, type } = typedField("Input", "", (v, patch) => { if (accept) patch(v); });
    type("12");
    accept = false;
    type("12x");
    assert.equal(el.value, "12");
});

test("a number field is left alone: its value reads empty for text it has not parsed", () => {
    const { el, type } = typedField("NumericInput", 4, () => {});
    type("-");
    assert.equal(el.value, "-");
});
