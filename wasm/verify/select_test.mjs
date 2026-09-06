// core.Select against the minimal DOM.
//
// htmlout writes the same picker into a static document and is tested there;
// this is the live half, where the interesting properties are the ones a
// snapshot has no way to have — that a *change* of value does not rebuild the
// option list, and that a change of the list does.
//
// The first is not a performance note. Replacing a <select>'s <option>
// elements resets the control, so rebuilding on every props patch would close
// an open drop-down mid-choice — and a controlled picker gets a props patch on
// exactly the pass where someone has just opened it.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

const opt = (value, label) => ({ value, label });

// A Style is always passed, even when it is empty. core.Select reads the
// theme's Input base, so a picker reaching this runtime with a null Style is a
// hand-built node and not something the framework produces — and applyStyle,
// which is where the border reset lives, only runs when there is a Style to
// apply.
function mountSelect(value, options, { style = {}, ...extra } = {}) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "Column",
        Children: [{ Type: "Select", Style: style, Props: { value, options, ...extra } }],
    }));
    rt.drainFrames();
    return { rt, el: nodeAt(rt.document, "root/0") };
}

test("a picker builds its options from the prop and shows the chosen one", () => {
    const { el } = mountSelect("pt", [opt("us", "United States"), opt("pt", "Portugal")]);

    assert.equal(el.tagName.toLowerCase(), "select");
    assert.equal(el.children.length, 2);
    assert.equal(el.children[0].getAttribute("value"), "us");
    assert.equal(el.children[1].textContent, "Portugal");
    assert.equal(el.value, "pt");
});

test("an option's label is set as text, never as markup", () => {
    // The labels are as user-originated as anything else here. htmlout escapes
    // the same string through element's TE; this is the property assignment
    // that has to be the safe one.
    const { el } = mountSelect("a", [opt("a", "<b>bold</b>")]);
    assert.equal(el.children[0].textContent, "<b>bold</b>");
});

test("selecting a value patches the value without rebuilding the list", () => {
    const { rt, el } = mountSelect("us", [opt("us", "United States"), opt("pt", "Portugal")]);
    const before = el.children[0];

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props",
        TargetID: "root/0",
        Changes: { value: "pt", options: [opt("us", "United States"), opt("pt", "Portugal")] },
    }]));

    assert.equal(el.value, "pt");
    assert.equal(el.children[0], before,
        "the option elements were replaced, which resets the control and closes an open menu");
});

test("a changed list is rebuilt, and the value re-applied after it", () => {
    const { rt, el } = mountSelect("us", [opt("us", "United States")]);

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props",
        TargetID: "root/0",
        Changes: { value: "pt", options: [opt("us", "United States"), opt("pt", "Portugal")] },
    }]));

    assert.equal(el.children.length, 2);
    assert.equal(el.children[1].getAttribute("value"), "pt");
    assert.equal(el.value, "pt");
});

test("a relabelled option is rebuilt even though the list is the same length", () => {
    // The signature is the list as JSON rather than its length, which is what
    // makes this case work: a translated picker changes every label and no
    // value.
    const { rt, el } = mountSelect("us", [opt("us", "United States")]);

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props",
        TargetID: "root/0",
        Changes: { value: "us", options: [opt("us", "Estados Unidos")] },
    }]));

    assert.equal(el.children[0].textContent, "Estados Unidos");
});

test("a choice dispatches the option's value through the text channel", () => {
    // core.Select takes a func(string), so Go registered the handler in the
    // text callback map. An envelope with no value — or with a number — would
    // be routed to a map where that ID does not exist, and the handler would
    // silently never run.
    const { rt, el } = mountSelect("us", [opt("us", "United States"), opt("pt", "Portugal")],
        { onChange: "txt_cb_0" });

    el.value = "pt";
    el.dispatch("input", { target: el });

    assert.deepEqual(rt.dispatched, [{ id: "txt_cb_0", payload: { value: "pt" } }]);
});

test("a picker with no border in its style is talked out of the browser's", () => {
    // The border decision core.Select was added to settle: on three targets
    // out of four the frame comes from the theme's Input base, so the web must
    // not draw a second one underneath.
    const { el } = mountSelect("a", [opt("a", "A")]);
    assert.equal(el.style.border, "none");

    // And the reset must not swallow a frame the theme does state, which is
    // the case every real app is in.
    const framed = mountSelect("a", [opt("a", "A")],
        { style: { BorderWidth: 1, BorderColor: "#8E8E93" } });
    assert.equal(framed.el.style.border, "1px solid #8E8E93");
});
