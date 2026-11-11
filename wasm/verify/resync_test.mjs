// The page asks Go for its whole tree when its own has drifted (N-133, the
// plan's R4).
//
// Patches are positional and incremental, so a page that misses or misapplies
// one stays wrong, and the lookup that would have noticed returned in silence.
// The runtime now asks GrMobWASM.Resync, which answers with one replace of
// "root" carrying the tree Go believes the page holds (render.Manager.Resync).
// What is pinned here:
//
//   - each symptom asks: a path that resolves to nothing, a payload that is
//     not a list, a patch that throws
//   - once per drift: further symptoms wait for the root replace
//   - the request waits for the batch, and for Go's push to return, like any
//     call into Go (dispatchToGo's rule; Resync takes the render lock)
//   - a polling page (Resync returns the replace) is brought back too
//   - a page whose transcript lost a batch ends equal to Go's final tree
//     after the resync

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, loadTranscript, nodeAt } from "./load.mjs";

const settle = () => new Promise((resolve) => setImmediate(resolve));

const TREE = {
    Type: "Column",
    Children: [
        { Type: "Text", Props: { content: "one" } },
        { Type: "Text", Props: { content: "two" } },
    ],
};

// harness mounts TREE and stands in for Go: Resync records each call and,
// like a page with no push listener, returns the replace. `answer` is the
// tree it sends.
function harness(answer = TREE) {
    const rt = loadRuntime();
    rt.window.GoInvokeCallback = () => {};
    const calls = [];
    rt.window.GrMobWASM = {
        Resync() {
            calls.push("Resync");
            return JSON.stringify([{ Type: "replace", TargetID: "root", Changes: answer }]);
        },
    };
    const warned = [];
    rt.sandbox.console = { ...rt.sandbox.console, warn: (...a) => warned.push(a.join(" ")), error: () => {} };
    rt.GrMob.mount(JSON.stringify(TREE));
    rt.drainFrames();
    return { rt, calls, warned };
}

const text = (rt, path) => nodeAt(rt.document, path)?.textContent;

test("a path that resolves to nothing asks once, after the batch, and the replace brings the page back", () => {
    const fixed = { ...TREE, Children: [TREE.Children[0], { Type: "Text", Props: { content: "TWO" } }] };
    const { rt, calls, warned } = harness(fixed);

    // Two patches into a subtree the page does not have: one drift, one ask.
    rt.GrMob.patch(JSON.stringify([
        { Type: "update-props", TargetID: "root/5", Changes: { content: "x" } },
        { Type: "update-props", TargetID: "root/6", Changes: { content: "y" } },
        { Type: "update-props", TargetID: "root/0", Changes: { content: "ONE" } },
    ]));

    assert.deepEqual(calls, ["Resync"], "one request for the whole batch, sent before patch() returned");
    assert.equal(warned.length, 2, "each dangling patch is logged");
    assert.match(warned[0], /update-props root\/5 resolves to nothing/);
    // The patch that did resolve landed before the request, and the replace
    // then put Go's tree in place over everything.
    assert.equal(text(rt, "root/1"), "TWO");
    assert.equal(text(rt, "root/0"), "one", "the replace is Go's tree, not the page's");

    // The replace cleared the flag, so a later drift asks again.
    rt.GrMob.patch(JSON.stringify([{ Type: "remove", TargetID: "root/9" }]));
    assert.deepEqual(calls, ["Resync", "Resync"]);
});

test("a payload that is not a list asks", () => {
    const { rt, calls, warned } = harness();
    rt.GrMob.patch('{"error":"failed to encode JSON"}');
    assert.deepEqual(calls, ["Resync"]);
    assert.match(warned[0], /not a list/);
});

test("a patch that throws asks, and does not skip the batch's post-passes", () => {
    const { rt, calls } = harness();
    let tracked = 0;
    const track = rt.GrMob.windowMetrics.track;
    rt.GrMob.windowMetrics.track = (...a) => { tracked++; return track(...a); };
    // A replace with no node in it: renderNode cannot build nothing.
    rt.GrMob.patch(JSON.stringify([{ Type: "replace", TargetID: "root/0", Changes: null }]));
    assert.deepEqual(calls, ["Resync"]);
    assert.ok(tracked >= 1, "the window pass at the batch's end still ran");
});

test("a drift found inside Go's push is asked about only after the push returns", async () => {
    const { rt, calls } = harness();
    let insidePush = false;
    // A page with the push listener: the replace would come through
    // GrMobApplyPatches inside this call, and the return is "[]".
    rt.window.GrMobWASM.Resync = () => {
        calls.push(insidePush ? "inside the push" : "after it");
        return "[]";
    };
    insidePush = true;
    rt.window.GrMobApplyPatches(JSON.stringify([{ Type: "update-props", TargetID: "root/7", Changes: {} }]));
    insidePush = false;
    assert.deepEqual(calls, [], "Resync takes the render lock Go is holding below the push");
    await settle();
    assert.deepEqual(calls, ["after it"]);
});

test("a Go module with no Resync binding is left alone, and the page can ask again later", () => {
    const { rt, calls } = harness();
    delete rt.window.GrMobWASM.Resync;
    assert.doesNotThrow(() => rt.GrMob.patch(JSON.stringify([{ Type: "remove", TargetID: "root/4" }])));
    rt.window.GrMobWASM.Resync = () => { calls.push("Resync"); return "[]"; };
    rt.GrMob.patch(JSON.stringify([{ Type: "remove", TargetID: "root/4" }]));
    assert.deepEqual(calls, ["Resync"], "the first request's flag did not outlive its call");
});

// The end-to-end claim, over a real transcript: drop one batch, as a host
// that lost a payload would, and the page notices, asks, and ends on Go's
// final tree. The batch dropped is the first one whose loss leaves a later
// patch dangling; a lost batch that leaves nothing dangling is a drift no
// page can see, and that is the stated limit of this mechanism.
test("a replay that lost a batch ends on Go's final tree after the resync", () => {
    const scenarios = loadTranscript().scenarios;
    let tried = 0;
    for (const scenario of scenarios) {
        for (let lost = 0; lost < scenario.steps.length; lost++) {
            tried++;
            const final = JSON.parse(scenario.final);
            const rt = loadRuntime();
            rt.window.GoInvokeCallback = () => {};
            let asked = 0;
            rt.window.GrMobWASM = {
                Resync() {
                    asked++;
                    return JSON.stringify([{ Type: "replace", TargetID: "root", Changes: final }]);
                },
            };
            rt.sandbox.console = { ...rt.sandbox.console, warn: () => {}, error: () => {} };
            rt.GrMob.mount(scenario.initial);
            rt.drainFrames();
            scenario.steps.forEach((batch, i) => {
                if (i === lost) return;
                rt.GrMob.patch(batch);
                rt.drainFrames();
            });
            if (asked === 0) continue;

            // Go's final tree, read back off the page by path and type.
            const want = [];
            (function walk(n, path) {
                want.push(`${path} ${n.Type}`);
                (n.Children || []).forEach((c, i) => walk(c, `${path}/${i}`));
            })(final, "root");
            const got = rt.mountPoint.findAll((el) => el.getAttribute("data-node-path") !== null)
                .map((el) => `${el.getAttribute("data-node-path")} ${el.dataset.nodeType}`);
            assert.deepEqual(got.filter((p) => !want.includes(p)), [],
                `${scenario.name} without batch ${lost}: the page holds nodes Go's final tree does not`);
            assert.deepEqual(want.filter((p) => !got.includes(p)), [],
                `${scenario.name} without batch ${lost}: Go's final tree has nodes the page does not`);
            return;
        }
    }
    assert.fail(`none of ${tried} single-batch losses left a patch dangling, so nothing here exercised a resync`);
});
