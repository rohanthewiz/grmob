// core.Paragraph on the web runtime: a <div> of spans, one per run, redrawn
// whole when the runs change, a link run a focusable span with the link role
// that Enter and a click both press.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

function mountParagraph(runs) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: [{ Type: "Paragraph", Props: { runs } }] }));
    return { rt, el: () => nodeAt(rt.document, "root/0") };
}

test("runs become spans in their marks", () => {
    const { el } = mountParagraph([
        { t: "plain " },
        { t: "marked", b: 1, i: 1, u: 1, s: 1, c: 1, fg: "#123456" },
    ]);
    const spans = el().children;
    assert.equal(el().tagName.toLowerCase(), "div");
    assert.equal(spans.length, 2);
    assert.equal(spans[0].textContent, "plain ");
    const s = spans[1].style;
    assert.equal(s.fontWeight, "700");
    assert.equal(s.fontStyle, "italic");
    assert.equal(s.textDecoration, "underline line-through");
    assert.equal(s.color, "#123456");
    assert.match(s.fontFamily, /monospace/);
});

test("a link run is a focusable link that click and Enter press", () => {
    const { rt, el } = mountParagraph([{ t: "go ", }, { t: "terms", cb: "cb_7" }]);
    const link = el().children[1];
    assert.equal(link.getAttribute("role"), "link");
    assert.equal(link.tabIndex, 0);
    link.dispatch("click");
    link.dispatch("keydown", { key: "Enter" });
    link.dispatch("keydown", { key: "a" });
    assert.deepEqual(rt.dispatched.map(d => d.id), ["cb_7", "cb_7"]);
});

test("a runs patch redraws the paragraph", () => {
    const { rt, el } = mountParagraph([{ t: "one" }]);
    rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root/0", Changes: { runs: [{ t: "two" }, { t: "three" }] } }]));
    assert.deepEqual([...el().children].map(c => c.textContent), ["two", "three"]);
});

// core.Keyboard reaches a field as its inputmode (the keyboard prop shares
// this file with Paragraph only because both landed together).
test("the keyboard prop sets inputmode, on mount and on patch", () => {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: [{ Type: "Input", Props: { value: "", keyboard: "digits" } }] }));
    const el = nodeAt(rt.document, "root/0");
    assert.equal(el.inputMode, "numeric");
    rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root/0", Changes: { value: "", keyboard: "email" } }]));
    assert.equal(el.inputMode, "email");
});

// The patch carries the whole new props map, so a field that drops
// core.Keyboard is back on the text keyboard, not left on the old pad.
test("a patch without the keyboard prop clears inputmode", () => {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: [{ Type: "Input", Props: { value: "", keyboard: "digits" } }] }));
    const el = nodeAt(rt.document, "root/0");
    assert.equal(el.inputMode, "numeric");
    rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root/0", Changes: { value: "12" } }]));
    assert.equal(el.inputMode, "");
});

test("a props patch on a non-field leaves inputmode alone", () => {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: [{ Type: "Text", Props: { content: "a" } }] }));
    const el = nodeAt(rt.document, "root/0");
    rt.GrMob.patch(JSON.stringify([{ Type: "update-props", TargetID: "root/0", Changes: { content: "b" } }]));
    assert.equal(el.inputMode, undefined);
});
