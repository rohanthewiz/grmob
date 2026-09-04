// Conformance replay: the JavaScript runtime against the Go reconciler.
//
// gen.go drives real example apps through render.Manager and records the
// initial tree, every patch batch, and the final tree. Here the real
// wasm/grmob-runtime.js mounts that initial tree, applies those batches, and
// the resulting DOM is walked back into a flat description that must equal
// the one derived from Go's final render.
//
// This is the check unit tests structurally cannot make: that patches land on
// the nodes they name. An add-child appending at the wrong index, a replace
// losing its data-node-path, a remove taking the wrong sibling — none of
// those are visible from any single function, and all of them show up here as
// a path that is missing, extra, or holding the wrong node type.

import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

import { loadRuntime } from "./load.mjs";

// run.sh generates the transcript and points here. The fallback is the same
// path run.sh writes to, so `node --test wasm/verify` works on its own once
// the suite has been run at least once — without ever writing into the repo.
const TRANSCRIPT =
    process.env.GRMOB_TRANSCRIPT ||
    `${process.env.TMPDIR || "/tmp"}/grmob-wasm-verify/transcript.json`.replace("//", "/");

const scenarios = JSON.parse(readFileSync(TRANSCRIPT, "utf8"));

// The Go node type -> <input type>, restated from the contract for the same
// reason the prop table below is: the runtime has its own copy of this table
// (inputTypeFor), and a conformance test that read the runtime's would only
// prove the runtime agrees with itself. Five Go types share the <input> tag,
// so this attribute is the only thing that makes a checkbox a checkbox.
//
// This copy stays deliberately independent even though the runtime's copy is
// now pinned to Go's (inputtype_test.go). Pinning it here too would close the
// loop back onto the implementation and leave the rule with no witness that
// was written from the contract.
const INPUT_TYPE = {
    Input: "text",
    InputPassword: "password",
    NumericInput: "number",
    Checkbox: "checkbox",
    Slider: "range",
};

// The props the runtime maps onto the DOM, restated here from the contract
// rather than copied from the implementation — two independent statements of
// one rule is the whole point of a conformance test.
function describeGoNode(node, path) {
    const props = node.Props || {};
    // Text carries `content`, Button carries `label`; everything else has no
    // text of its own and takes its content from its children.
    const text = props.content ?? props.label ?? "";
    return {
        path,
        type: node.Type,
        // The <input> discriminator; null for every node whose tag already
        // says what it is.
        inputType: INPUT_TYPE[node.Type] ?? null,
        text: String(text),
        value: props.value === undefined ? undefined : String(props.value),
        placeholder: props.placeholder === undefined ? undefined : String(props.placeholder),
        // A Checkbox's live state. Normalized to a boolean on both sides: Go
        // sends a JSON bool and the DOM holds a property, so neither side has
        // a string to compare.
        checked: props.checked === undefined ? undefined : !!props.checked,
        // A TextArea's height in lines. Only a positive count reaches the DOM
        // — rows is limited to positive numbers there — so anything else
        // leaves the browser's default and reads back as undefined.
        rows: positiveRows(props.rows),
        // The keyboard action hint: "next" when the field advertises
        // traversal, "done" when it merely acts on return, absent otherwise.
        enterkeyhint:
            props.imeAction === "next" ? "next" : props.onSubmit ? "done" : null,
    };
}

function positiveRows(rows) {
    const n = Number(rows);
    return Number.isInteger(n) && n > 0 ? n : undefined;
}

function fromGoTree(node, path = "root", out = []) {
    out.push(describeGoNode(node, path));
    for (const [i, child] of (node.Children || []).entries()) {
        fromGoTree(child, `${path}/${i}`, out);
    }
    return out;
}

// Chrome — an element the runtime draws that no node asked for, currently a
// TabView's bar (buildTabBar). It is skipped whole, subtree and all, because
// Go's tree has nothing to compare it against.
//
// Skipped on the marker *and* checked for a path, rather than skipped on
// "has no data-node-path": the second test would also swallow a real node
// whose path attribute went missing, which is one of the exact failures this
// replay exists to catch (a replace losing its data-node-path). Marked chrome
// carrying a path is a contradiction and fails loudly instead.
function isChrome(el) {
    if (el.dataset.grmobChrome === undefined) return false;
    assert.equal(
        el.getAttribute("data-node-path"),
        null,
        `chrome element <${el.tagName.toLowerCase()}> carries a data-node-path; ` +
            `it is not a node and no patch may be addressed to it`
    );
    return true;
}

function fromDOM(el, out = []) {
    out.push({
        path: el.getAttribute("data-node-path"),
        type: el.dataset.nodeType,
        inputType: el.getAttribute("type"),
        text: el.textContent,
        value: el.value === undefined ? undefined : String(el.value),
        placeholder: el.placeholder === undefined ? undefined : String(el.placeholder),
        checked: el.checked === undefined ? undefined : !!el.checked,
        rows: el.rows,
        enterkeyhint: el.getAttribute("enterkeyhint"),
    });
    for (const child of el.children) {
        if (isChrome(child)) continue;
        fromDOM(child, out);
    }
    return out;
}

for (const scenario of scenarios) {
    test(`replay: ${scenario.name} (${scenario.steps.length} patch batches)`, () => {
        const rt = loadRuntime();

        rt.GrMob.mount(scenario.initial);
        // The initial tree is built detached and appended once assembled, so
        // any focus command it carried is deferred a frame. Drain after every
        // application, exactly as a browser's frame loop would.
        rt.drainFrames();

        for (const [i, batch] of scenario.steps.entries()) {
            rt.GrMob.patch(batch);
            rt.drainFrames();
            // A batch that reached no element at all would otherwise pass
            // silently: the patch handler returns early on a missing target.
            // Every batch in a recorded transcript named live nodes when Go
            // emitted it, so one that matches nothing now is drift.
            const applied = JSON.parse(batch).some((p) =>
                rt.document.querySelector(`[data-node-path="${p.TargetID}"]`)
            );
            assert.ok(applied, `batch ${i} named no element that exists`);
        }

        const root = rt.mountPoint.children[0];
        assert.ok(root, "nothing was mounted");

        const want = fromGoTree(JSON.parse(scenario.final));
        const got = fromDOM(root);

        // Paths first and as a set: a structural mismatch produces a clearer
        // failure than the first differing element of two long lists.
        const wantPaths = want.map((n) => n.path);
        const gotPaths = got.map((n) => n.path);
        assert.deepEqual(
            gotPaths.filter((p) => !wantPaths.includes(p)),
            [],
            "DOM has nodes Go's final tree does not"
        );
        assert.deepEqual(
            wantPaths.filter((p) => !gotPaths.includes(p)),
            [],
            "Go's final tree has nodes the DOM does not"
        );
        // Then order and content, node for node.
        assert.deepEqual(got, want);

        // And the one document-global invariant this replay is in a position
        // to check. The tab/panel wiring writes ids, and an id that is not
        // unique makes both of its ARIA references ambiguous — aria-controls
        // resolves to whichever element the browser found first. The ids are
        // derived from data-node-path rather than from a counter, so this is
        // really the addressing scheme's own uniqueness being asserted at the
        // end of a whole transcript of patches, which nothing else here does.
        //
        // Vacuous for a scenario with no TabView in it — signup has no ids at
        // all — and load-bearing for demo, which ends with eight. That is the
        // honest shape of a document-global invariant: it says nothing about a
        // document that has nothing to say.
        const ids = root
            .findAll((el) => el.getAttribute("id") !== null)
            .map((el) => el.getAttribute("id"));
        assert.equal(new Set(ids).size, ids.length, `duplicate element ids: ${ids}`);
    });
}

// The focus half, end to end through the real runtime: examples/signup's
// server-error path issues a core.Focus at the email field, and the only
// proof the WASM focus path works is that the element actually holds focus
// once the frame it was deferred to has run.
test("replay: a focus command moves the browser's focus", () => {
    const scenario = scenarios.find((s) => s.name === "signup");
    assert.ok(scenario, "no signup scenario in the transcript");

    const rt = loadRuntime();
    rt.GrMob.mount(scenario.initial);
    rt.drainFrames();

    assert.equal(
        rt.document.activeElement,
        null,
        "something took focus before any command was issued"
    );

    for (const batch of scenario.steps) {
        rt.GrMob.patch(batch);
        rt.drainFrames();
    }

    const email = rt.mountPoint.find(
        (el) => el.placeholder === "you@example.com"
    );
    assert.ok(email, "no email field in the mounted tree");
    assert.equal(
        rt.document.activeElement,
        email,
        "the failed submit did not put the cursor back in the email field"
    );
});
