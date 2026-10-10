// The browser half of Go's keystore package (keystore/keystore.go), which is
// a refusal on purpose: a page has no store that script on the origin cannot
// read. What is worth pinning, since no Go test can see the page:
//
//   - every command is answered — Go holds the caller's callback until its id
//     comes back and has no timeout, so a silent drop is a hung caller
//   - the answer is the reserved reason "unavailable", which Go maps to
//     keystore.ErrUnavailable, and never carries the value a save sent
//   - the answer arrives after the call returns, as it does on the natives
//   - a request without an id, or a page with no GrMobWASM yet, is dropped
//     rather than throwing

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime } from "./load.mjs";

function harness() {
    const rt = loadRuntime();
    const replies = [];
    rt.window.GrMobWASM = {
        HostEvent: (name, json) => replies.push({ name, data: JSON.parse(json) }),
    };
    return { rt, replies, settle: () => new Promise((resolve) => setImmediate(resolve)) };
}

test("every command is refused as unavailable, after the call returns", async () => {
    const h = harness();
    h.rt.GrMob.keystore.handle({ command: "save", id: "1", key: "token", value: "s3cret" });
    h.rt.GrMob.keystore.handle({ command: "get", id: "2", key: "token" });
    h.rt.GrMob.keystore.handle({ command: "delete", id: "3", key: "token" });
    h.rt.GrMob.keystore.handle({ command: "rotate", id: "4", key: "token" });
    assert.equal(h.replies.length, 0, "a reply was delivered inside the call");

    await h.settle();
    assert.deepEqual(
        h.replies.map((r) => [r.name, r.data.id, r.data.ok, r.data.error]),
        [
            ["keystore", "1", false, "unavailable"],
            ["keystore", "2", false, "unavailable"],
            ["keystore", "3", false, "unavailable"],
            ["keystore", "4", false, "unavailable"],
        ],
    );
    for (const r of h.replies) {
        assert.ok(!("value" in r.data), `reply ${r.data.id} carries a value`);
        assert.ok(!("found" in r.data), `reply ${r.data.id} carries found`);
    }
});

test("a request without an id is dropped", async () => {
    const h = harness();
    h.rt.GrMob.keystore.handle({ command: "get", key: "token" });
    await h.settle();
    assert.equal(h.replies.length, 0);
});

test("a page with no GrMobWASM yet does not throw", async () => {
    const h = harness();
    delete h.rt.window.GrMobWASM;
    assert.doesNotThrow(() => h.rt.GrMob.keystore.handle({ command: "get", id: "1", key: "k" }));
    await h.settle();
});
