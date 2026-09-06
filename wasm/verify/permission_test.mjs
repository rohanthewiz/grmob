// The browser half of Go's permission package, against the minimal DOM.
//
// This is the host where the two commands are genuinely different operations
// and where one of them mostly cannot be honoured, so almost everything worth
// pinning here is a browser fact that no Go test can reach:
//
//   check     navigator.permissions.query — a real read, no UI, answering in
//             exactly the words Go's Status carries
//   request   no such API. A page obtains camera, microphone or location by
//             *calling the feature*, and the prompt is a side effect of that
//             call — so a request is a getUserMedia whose tracks are stopped
//             the instant it resolves, or a single getCurrentPosition
//
// The interesting failures are all in the seams: a descriptor Firefox has
// never heard of makes query() *throw* rather than resolve, a getUserMedia
// rejection says "denied" and "there is no camera" through the same channel,
// and a granted media request has genuinely opened a capture device that has
// to be closed again.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime } from "./load.mjs";

// A runtime with a place for permission answers to land, plus whatever
// navigator APIs the case is modelling. Returns the events Go would have
// seen, decoded.
function harness(navigator = {}) {
    const rt = loadRuntime();
    const answers = [];
    rt.window.GrMobWASM = {
        HostEvent: (name, json) => {
            if (name === "permission") answers.push(JSON.parse(json));
        },
    };
    Object.assign(rt.sandbox.navigator, navigator);
    return {
        rt,
        answers,
        check: (kind) => rt.GrMob.permission.handle({ command: "check", kind }),
        request: (kind) => rt.GrMob.permission.handle({ command: "request", kind }),
    };
}

// The runtime answers from promise callbacks, so every assertion has to come
// after the microtask queue has drained. Two turns rather than one: the media
// path chains a .then onto the getUserMedia promise, so its report lands one
// tick behind the call.
const settle = async () => { await null; await null; await null; };

// --- check ------------------------------------------------------------------

test("a check answers with the Permissions API's own word", async () => {
    // The whole argument for Go spelling its Status values the way the W3C
    // Permissions API does: this host needs no mapping table, and the state
    // is forwarded unchanged.
    for (const state of ["granted", "denied", "prompt"]) {
        const h = harness({
            permissions: { query: () => Promise.resolve({ state }) },
        });
        h.check("camera");
        await settle();
        assert.deepEqual(h.answers, [{ kind: "camera", status: state }]);
    }
});

test("a check uses the descriptor name, not the Go kind", async () => {
    // "location" is Go's word and "geolocation" is the browser's; a query for
    // the wrong one throws, which the catch below would turn into a plausible
    // "unavailable" rather than an error anybody would see.
    let asked;
    const h = harness({
        permissions: {
            query: (d) => { asked = d.name; return Promise.resolve({ state: "granted" }); },
        },
    });
    h.check("location");
    await settle();
    assert.equal(asked, "geolocation");
    assert.deepEqual(h.answers, [{ kind: "location", status: "granted" }]);
});

test("a descriptor this browser has never heard of is unavailable", async () => {
    // query() rejects — with a TypeError, not a resolved state — for a name it
    // does not know, and browsers disagree about which names those are:
    // Firefox has no "camera" descriptor at all. So this is the common path on
    // some browsers rather than an edge case, and it must not surface as a
    // denial: a denial sends the user to a settings page that has no switch
    // on it.
    const h = harness({
        permissions: { query: () => Promise.reject(new TypeError("bad descriptor")) },
    });
    h.check("camera");
    await settle();
    assert.deepEqual(h.answers, [{ kind: "camera", status: "unavailable" }]);
});

test("a page with no Permissions API at all answers unavailable", async () => {
    const h = harness(); // the default navigator: nothing installed
    h.check("microphone");
    await settle();
    assert.deepEqual(h.answers, [{ kind: "microphone", status: "unavailable" }]);
});

test("storage has no descriptor and says so without asking", async () => {
    // A page reaches files through an <input> or the file-system access API,
    // both of which are a gesture rather than a permission. There is nothing
    // to query and nothing to request, so the honest answer is that this
    // platform cannot grant it — and it must be *sent*, or a screen waiting on
    // it waits forever.
    let queried = false;
    const h = harness({
        permissions: { query: () => { queried = true; return Promise.resolve({ state: "granted" }); } },
    });
    h.check("storage");
    await settle();
    assert.equal(queried, false, "storage must not reach the Permissions API");
    assert.deepEqual(h.answers, [{ kind: "storage", status: "unavailable" }]);
});

// --- request ----------------------------------------------------------------

test("a media request prompts through getUserMedia and closes the device again", async () => {
    // The prompt is the point and the stream is not: a resolved getUserMedia
    // means a live capture device, and leaving it running would keep the
    // browser's recording indicator lit for a page that only wanted an answer.
    let stopped = 0;
    let constraints;
    const h = harness({
        mediaDevices: {
            getUserMedia: (c) => {
                // Serialized at the boundary, as load.mjs does for dispatches
                // and for the same reason: an object built inside the vm
                // context carries that realm's Object.prototype, so a strict
                // deep-equal against a literal here fails on prototype
                // identity alone.
                constraints = JSON.parse(JSON.stringify(c));
                return Promise.resolve({ getTracks: () => [{ stop: () => { stopped++; } }] });
            },
        },
    });
    h.request("camera");
    await settle();

    assert.deepEqual(constraints, { video: true });
    assert.equal(stopped, 1, "the track was left running after the prompt");
    assert.deepEqual(h.answers, [{ kind: "camera", status: "granted" }]);
});

test("the microphone asks for audio, not video", async () => {
    let constraints;
    const h = harness({
        mediaDevices: {
            getUserMedia: (c) => {
                constraints = JSON.parse(JSON.stringify(c));
                return Promise.resolve({ getTracks: () => [] });
            },
        },
    });
    h.request("microphone");
    await settle();
    assert.deepEqual(constraints, { audio: true });
});

test("a refused prompt and an absent device are told apart", async () => {
    // Both arrive as a rejected getUserMedia and they need different words,
    // because only one of them has a fix in the browser's settings. That
    // distinction is the whole reason Go's Status has Unavailable beside
    // Denied.
    for (const [name, status] of [
        ["NotAllowedError", "denied"],
        ["NotFoundError", "unavailable"],
        ["OverconstrainedError", "unavailable"],
        ["NotReadableError", "unavailable"],
        // Anything unrecognised falls to denied: an unknown failure of a
        // prompt is much more likely to be a refusal than a missing camera,
        // and denied is the answer whose remedy is harmless to offer.
        ["SomeFutureError", "denied"],
    ]) {
        const h = harness({
            mediaDevices: {
                getUserMedia: () => Promise.reject(Object.assign(new Error("no"), { name })),
            },
        });
        h.request("camera");
        await settle();
        assert.deepEqual(h.answers, [{ kind: "camera", status }], name);
    }
});

test("a location request prompts through getCurrentPosition", async () => {
    const h = harness({
        geolocation: { getCurrentPosition: (ok) => ok({ coords: {} }) },
    });
    h.request("location");
    await settle();
    assert.deepEqual(h.answers, [{ kind: "location", status: "granted" }]);
});

test("only PERMISSION_DENIED is a denial; a failed fix re-checks", async () => {
    // POSITION_UNAVAILABLE and TIMEOUT are failures of the *fix* and not of
    // the permission — the page may be perfectly authorised and simply
    // indoors — so reporting them as denied would tell a user to change a
    // setting that is already right.
    const denied = harness({
        geolocation: { getCurrentPosition: (_, fail) => fail({ code: 1 }) },
    });
    denied.request("location");
    await settle();
    assert.deepEqual(denied.answers, [{ kind: "location", status: "denied" }]);

    const indoors = harness({
        geolocation: { getCurrentPosition: (_, fail) => fail({ code: 2 }) },
        permissions: { query: () => Promise.resolve({ state: "granted" }) },
    });
    indoors.request("location");
    await settle();
    assert.deepEqual(indoors.answers, [{ kind: "location", status: "granted" }],
        "a position failure must fall back to reading the permission, not assume a refusal");
});

test("a page with no media or geolocation provider answers unavailable", async () => {
    const h = harness();
    h.request("camera");
    h.request("location");
    await settle();
    assert.deepEqual(h.answers, [
        { kind: "camera", status: "unavailable" },
        { kind: "location", status: "unavailable" },
    ]);
});

// --- the dispatcher ---------------------------------------------------------

test("an unknown kind or command is dropped, not answered", async () => {
    // Every host's contract for an unknown event. Answering would be worse
    // than silence here: Go validates the kind on arrival and would drop the
    // reply anyway, and a host that invented a status for a permission it does
    // not know is a host that will keep inventing one after Go learns about it.
    const h = harness({
        permissions: { query: () => Promise.resolve({ state: "granted" }) },
    });
    h.rt.GrMob.permission.handle({ command: "check", kind: "bluetooth" });
    h.rt.GrMob.permission.handle({ command: "revoke", kind: "camera" });
    h.rt.GrMob.permission.handle({});
    await settle();
    assert.deepEqual(h.answers, []);
});

test("an answer with no host channel attached is dropped rather than thrown", async () => {
    // A page that has not finished wiring GrMobWASM, which is a real window
    // during startup. Nothing here may throw into the runtime's own promise
    // chain, where the rejection would be unhandled and invisible.
    const rt = loadRuntime();
    Object.assign(rt.sandbox.navigator, {
        permissions: { query: () => Promise.resolve({ state: "granted" }) },
    });
    rt.GrMob.permission.handle({ command: "check", kind: "camera" });
    await settle();
});
