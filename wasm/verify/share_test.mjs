// The browser half of core.Share (core/share.go): the Web Share API. What is
// worth pinning, since no Go test can see the page:
//
//   - every request is answered, because Go holds done until its id comes
//     back and has no timeout, so a silent drop is a share button that
//     never falls back
//   - ok:false wherever no sheet can be shown (no navigator.share, canShare
//     saying no, a rejection other than the person dismissing the sheet), so
//     the caller's fallback runs exactly there
//   - a dismissal (AbortError) is ok:true: the sheet was shown
//   - only the fields that are set reach navigator.share
//   - a request without an id is dropped rather than throwing

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime } from "./load.mjs";

function harness(share, canShare) {
    const rt = loadRuntime();
    const replies = [];
    const calls = [];
    rt.window.GrMobWASM = {
        HostEvent: (name, json) => replies.push({ name, data: JSON.parse(json) }),
    };
    if (share) {
        rt.sandbox.navigator.share = (payload) => {
            calls.push(JSON.parse(JSON.stringify(payload)));
            return share(payload);
        };
    }
    if (canShare) rt.sandbox.navigator.canShare = canShare;
    return { rt, replies, calls, settle: () => new Promise((resolve) => setImmediate(resolve)) };
}

function rejection(name) {
    const err = new Error(name);
    err.name = name;
    return Promise.reject(err);
}

test("a shown sheet answers ok, and only the set fields are shared", async () => {
    const h = harness(() => Promise.resolve());
    h.rt.GrMob.share.handle({ id: "1", text: "A sunset", url: "https://example.com/p/1" });
    h.rt.GrMob.share.handle({ id: "2", text: "just text", url: "" });
    await h.settle();
    assert.deepEqual(h.calls, [
        { text: "A sunset", url: "https://example.com/p/1" },
        { text: "just text" },
    ]);
    assert.deepEqual(h.replies.map((r) => [r.name, r.data.id, r.data.ok]),
        [["share", "1", true], ["share", "2", true]]);
});

test("no Web Share API answers ok:false, after the call returns", async () => {
    const h = harness(null);
    h.rt.GrMob.share.handle({ id: "1", text: "x" });
    assert.equal(h.replies.length, 0, "a reply was delivered inside the call");
    await h.settle();
    assert.deepEqual(h.replies.map((r) => [r.data.id, r.data.ok]), [["1", false]]);
});

test("canShare saying no answers ok:false without opening a sheet", async () => {
    const h = harness(() => Promise.resolve(), () => false);
    h.rt.GrMob.share.handle({ id: "1", url: "not a url" });
    await h.settle();
    assert.equal(h.calls.length, 0);
    assert.deepEqual(h.replies.map((r) => [r.data.id, r.data.ok]), [["1", false]]);
});

test("a dismissal is ok, any other rejection is a refusal", async () => {
    const outcomes = { "1": "AbortError", "2": "NotAllowedError", "3": "TypeError" };
    // calls already holds this call when share runs, so its length is the id.
    const h = harness(() => rejection(outcomes[String(h.calls.length)]));
    h.rt.GrMob.share.handle({ id: "1", text: "x" });
    h.rt.GrMob.share.handle({ id: "2", text: "x" });
    h.rt.GrMob.share.handle({ id: "3", text: "x" });
    await h.settle();
    assert.deepEqual(h.replies.map((r) => [r.data.id, r.data.ok]),
        [["1", true], ["2", false], ["3", false]]);
});

test("a request without an id is dropped", async () => {
    const h = harness(() => Promise.resolve());
    h.rt.GrMob.share.handle({ text: "x" });
    await h.settle();
    assert.equal(h.calls.length, 0);
    assert.equal(h.replies.length, 0);
});
