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

import { loadRuntime, loadTranscript, nodeAt } from "./load.mjs";

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

test("consecutive options sharing a group become one optgroup", () => {
    // Runs, not a gather: the same heading either side of a different one is
    // two groups, in the order written. See core.SelectOption.Group for why
    // reordering the list to suit the headings is not this widget's to do.
    const { el } = mountSelect("pt", [
        opt("none", "Pick one"),
        { value: "pt", label: "Portugal", group: "Europe" },
        { value: "es", label: "Spain", group: "Europe" },
        { value: "us", label: "United States", group: "Americas" },
        opt("zz", "Elsewhere"),
    ]);

    // Top level: the ungrouped option, two groups, the ungrouped option.
    assert.deepEqual(
        el.children.map((c) => c.tagName.toLowerCase()),
        ["option", "optgroup", "optgroup", "option"],
    );
    assert.equal(el.children[1].getAttribute("label"), "Europe");
    assert.deepEqual(el.children[1].children.map((c) => c.textContent), ["Portugal", "Spain"]);
    assert.equal(el.children[2].getAttribute("label"), "Americas");
    assert.deepEqual(el.children[2].children.map((c) => c.textContent), ["United States"]);
    assert.equal(el.children[3].textContent, "Elsewhere");
});

test("the same heading either side of another one is two groups", () => {
    const { el } = mountSelect("a", [
        { value: "a", label: "A", group: "One" },
        { value: "b", label: "B", group: "Two" },
        { value: "c", label: "C", group: "One" },
    ]);

    assert.equal(el.children.length, 3, "the runs were gathered instead of kept in order");
    assert.deepEqual(el.children.map((c) => c.getAttribute("label")), ["One", "Two", "One"]);
});

test("an optgroup is chrome, like a TabView's bar", () => {
    // No node is ever addressed to it, so the conformance replay must be able
    // to tell it from a Go node — the same marker every option carries.
    const { el } = mountSelect("a", [{ value: "a", label: "A", group: "G" }]);

    assert.equal(el.children[0].dataset.grmobChrome, "optgroup");
    assert.equal(el.children[0].children[0].dataset.grmobChrome, "option");
});

test("a disabled option is drawn and not choosable", () => {
    // Drawn is half the point: an option that vanished would take its
    // explanation with it. See core.SelectOption.Disabled.
    const { el } = mountSelect("s", [
        opt("s", "Small"),
        { value: "l", label: "Large", disabled: "true" },
    ]);

    assert.equal(el.children[1].textContent, "Large");
    assert.equal(el.children[1].disabled, true);
    // false, not undefined: dom.mjs starts every element's `disabled` at the
    // browser's own default, unlike `checked` and `value`.
    assert.equal(el.children[0].disabled, false,
        "an option nobody disabled was disabled anyway");
});

test("a group or a disabled flag changing rebuilds the list", () => {
    // The rebuild signature is the list's JSON, so this is really a check that
    // the two new keys are *in* it — a signature computed from values and
    // labels alone would leave a re-grouped picker showing the old headings.
    const { rt, el } = mountSelect("a", [
        { value: "a", label: "A", group: "One" },
        opt("b", "B"),
    ]);
    assert.equal(el.children[0].getAttribute("label"), "One");

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props",
        TargetID: "root/0",
        Changes: {
            value: "a",
            options: [
                { value: "a", label: "A", group: "Two" },
                { value: "b", label: "B", disabled: "true" },
            ],
        },
    }]));

    assert.equal(el.children[0].getAttribute("label"), "Two");
    assert.equal(el.children[1].disabled, true);
});

test("a disabled run is one attribute on the optgroup and one on every option", () => {
    // core.SelectOption.GroupDisabled. The web is the only target of the four
    // that can refuse a whole run in one place, and it still writes both: the
    // group's attribute greys the *heading*, the options' attributes are what
    // core resolved for SwiftUI and Compose, which have no section-level
    // control at all.
    const { el } = mountSelect("free", [
        { value: "free", label: "Free", group: "Free" },
        { value: "pro", label: "Pro", group: "Paid" },
        // On the run's *last* option: a runtime that read the flag as it
        // opened the group would miss it, and would pass every list that
        // happens to declare it first.
        { value: "max", label: "Max", group: "Paid", groupDisabled: "true" },
    ]);

    const [free, paid] = el.children;
    assert.equal(free.disabled, false, "a run nobody disabled was disabled anyway");
    assert.equal(paid.disabled, true, "the run's optgroup carries no disabled attribute");
    assert.deepEqual(paid.children.map((c) => c.disabled), [true, true],
        "the declaration did not reach the option that made none");
    assert.deepEqual(free.children.map((c) => c.disabled), [false]);
});

test("a disabled run with no heading falls back to its options", () => {
    // No <optgroup> exists to carry the attribute, so the declaration degrades
    // to exactly "every option in the run is disabled" — the case the
    // per-option propagation is there for.
    const { el } = mountSelect("a", [
        { value: "a", label: "A", groupDisabled: "true" },
        { value: "b", label: "B" },
    ]);

    assert.deepEqual(el.children.map((c) => c.tagName.toLowerCase()), ["option", "option"]);
    assert.deepEqual(el.children.map((c) => c.disabled), [true, true]);
});

test("a disabled run does not reach the run after it", () => {
    const { el } = mountSelect("free", [
        { value: "pro", label: "Pro", group: "Paid", groupDisabled: "true" },
        { value: "free", label: "Free", group: "Free" },
    ]);

    assert.deepEqual(el.children.map((c) => c.disabled), [true, false],
        "the run state outlived its run");
});

// The fourth transliteration, held to the same statement of the rule as the
// other three.
//
// selectMenuSections in grmob-runtime.js is not reachable from here — the
// runtime exposes mount and patch and nothing else — and that is fine, because
// the interesting subject is not the function. It is the DOM the function
// produced. So this rebuilds the sections back out of the rendered <select>
// and compares them with what Go says: an <optgroup> is a section with a
// heading, and each maximal run of top-level <option> elements is the
// headingless section between two of them.
//
// The reconstruction is unambiguous because two ungrouped runs can never be
// adjacent — a run only ends where the heading changes, so anything between
// two ungrouped options with no <optgroup> between them is one run.
function sectionsFromDOM(el) {
    const sections = [];
    let loose = null;
    const item = (opt, index) => ({
        index,
        value: opt.getAttribute("value") ?? "",
        label: opt.textContent,
        disabled: opt.disabled === true,
    });
    let index = 0;
    for (const child of el.children) {
        if (child.tagName.toLowerCase() === "optgroup") {
            loose = null;
            sections.push({
                heading: child.getAttribute("label") ?? "",
                disabled: child.disabled === true,
                items: child.children.map((o) => item(o, index++)),
            });
            continue;
        }
        if (!loose) {
            // A headingless run has no element of its own, so there is nothing
            // to read a disabled flag off. Go reports one; what the DOM can
            // show is the options, and they carry it — which is the whole
            // reason core propagates it. The comparison below leaves this
            // field out for exactly that reason.
            loose = { heading: "", disabled: null, items: [] };
            sections.push(loose);
        }
        loose.items.push(item(child, index++));
    }
    return sections;
}

test("every menu case decomposes in the DOM the way Go says it does", () => {
    const cases = loadTranscript().menuCases;
    // A guard on the fixture itself: an empty table would make every
    // assertion below vacuous, and the transcript is generated by another
    // program.
    assert.ok(cases.length > 5, `only ${cases.length} menu cases in the transcript`);

    for (const c of cases) {
        const { el } = mountSelect("", c.options);
        const got = sectionsFromDOM(el);
        const want = c.want;

        assert.equal(got.length, want.length,
            `${c.name}: ${got.length} sections in the DOM, Go says ${want.length}`);

        for (let i = 0; i < want.length; i++) {
            const g = got[i];
            const w = want[i];
            assert.equal(g.heading, w.heading, `${c.name} section ${i}: heading`);
            // Only a run with a heading has an element that could carry it;
            // see sectionsFromDOM.
            if (w.heading !== "") {
                assert.equal(g.disabled, w.disabled, `${c.name} section ${i}: optgroup disabled`);
            }
            assert.equal(g.items.length, w.items.length,
                `${c.name} section ${i}: ${g.items.length} options, Go says ${w.items.length}`);
            for (let j = 0; j < w.items.length; j++) {
                const gi = g.items[j];
                const wi = w.items[j];
                // The index is the option's position in the *whole* list, so
                // this also asserts that no option was dropped, duplicated or
                // moved between runs.
                assert.equal(gi.index, wi.index, `${c.name} section ${i} option ${j}: index`);
                assert.equal(gi.value, wi.value, `${c.name} section ${i} option ${j}: value`);
                assert.equal(gi.label, wi.label, `${c.name} section ${i} option ${j}: label`);
                assert.equal(gi.disabled, wi.disabled,
                    `${c.name} section ${i} option ${j}: disabled`);
            }
        }
    }
});
