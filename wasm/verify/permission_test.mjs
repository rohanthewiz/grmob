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
// after the microtask queue has drained. Several turns rather than one: the
// longest path here is three promises deep — a request reads the permission,
// reaches for the device, and reads the permission again to tell a Block from
// a dismissed prompt — and each link lands one tick behind the last. The count
// is generous on purpose; an extra drained turn costs nothing and a missing
// one is a test that fails for a reason that has nothing to do with its
// subject.
const settle = async () => {
    for (let i = 0; i < 8; i++) await null;
};

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

// --- request: the read that comes first -------------------------------------

test("a request for something already granted never opens the device", async () => {
    // The entry this closes: a granted camera request had genuinely opened the
    // camera for a moment — the browser's recording indicator lighting up to
    // answer a question the browser had already written down.
    let opened = 0;
    const h = harness({
        permissions: { query: () => Promise.resolve({ state: "granted" }) },
        mediaDevices: {
            getUserMedia: () => {
                opened++;
                return Promise.resolve({ getTracks: () => [] });
            },
        },
    });
    h.request("camera");
    await settle();

    assert.equal(opened, 0, "the camera was opened to confirm a permission the "
        + "browser had already recorded");
    assert.deepEqual(h.answers, [{ kind: "camera", status: "granted" }]);
});

test("a request for something already denied does not call the feature", async () => {
    // A getUserMedia here rejects immediately with no UI, so the call buys a
    // NotAllowedError and nothing else.
    let opened = 0;
    const h = harness({
        permissions: { query: () => Promise.resolve({ state: "denied" }) },
        mediaDevices: {
            getUserMedia: () => { opened++; return Promise.reject(new Error("no")); },
        },
    });
    h.request("camera");
    await settle();

    assert.equal(opened, 0);
    assert.deepEqual(h.answers, [{ kind: "camera", status: "denied" }]);
});

test("a location request for a granted origin does not take a fix", async () => {
    // The same saving on the other API, where the cost is larger: a
    // getCurrentPosition on a phone can spin up the GPS.
    let fixes = 0;
    const h = harness({
        permissions: { query: () => Promise.resolve({ state: "granted" }) },
        geolocation: { getCurrentPosition: (ok) => { fixes++; ok({ coords: {} }); } },
    });
    h.request("location");
    await settle();

    assert.equal(fixes, 0);
    assert.deepEqual(h.answers, [{ kind: "location", status: "granted" }]);
});

test("an undecided permission still reaches for the device", async () => {
    // The one state where a request has something to do. Opening the camera
    // *is* the prompt; there is no other way to put one on screen.
    let opened = 0;
    const h = harness({
        permissions: { query: () => Promise.resolve({ state: "prompt" }) },
        mediaDevices: {
            getUserMedia: () => {
                opened++;
                return Promise.resolve({ getTracks: () => [] });
            },
        },
    });
    h.request("camera");
    await settle();

    assert.equal(opened, 1, "an undecided permission was never asked for");
    assert.deepEqual(h.answers, [{ kind: "camera", status: "granted" }]);
});

test("a browser with no descriptor asks by asking", async () => {
    // Firefox has no "camera" descriptor at all, so query() throws rather than
    // resolving. The read cannot answer and the feature call is the only thing
    // left — which is what this path always did.
    let opened = 0;
    const h = harness({
        permissions: { query: () => Promise.reject(new TypeError("bad descriptor")) },
        mediaDevices: {
            getUserMedia: () => {
                opened++;
                return Promise.resolve({ getTracks: () => [] });
            },
        },
    });
    h.request("camera");
    await settle();

    assert.equal(opened, 1, "a browser that cannot be read was not asked either");
    assert.deepEqual(h.answers, [{ kind: "camera", status: "granted" }]);
});

test("storage is answered without ever reaching the query", async () => {
    // There is no descriptor for it, so the read would be a round trip to an
    // answer this file already knows.
    let queried = 0;
    const h = harness({
        permissions: { query: () => { queried++; return Promise.resolve({ state: "granted" }); } },
    });
    h.request("storage");
    await settle();

    assert.equal(queried, 0);
    assert.deepEqual(h.answers, [{ kind: "storage", status: "unavailable" }]);
});

// --- request: a Block and a dismissal are different answers -----------------

test("a dismissed prompt is prompt, and a Block is denied", async () => {
    // Both arrive as the same NotAllowedError. The Permissions API is what
    // separates them: a Block is recorded as "denied", a dismissal leaves the
    // state where it was. The words matter to the screen — one of them means
    // "ask again next time the user reaches for this" and the other means
    // "send them to the site settings".
    for (const [after, status] of [["prompt", "prompt"], ["denied", "denied"]]) {
        // The first query has to say "prompt" or the request would never reach
        // getUserMedia at all; the second is the read-back after the refusal.
        const states = ["prompt", after];
        let n = 0;
        const h = harness({
            permissions: { query: () => Promise.resolve({ state: states[n++] || after }) },
            mediaDevices: {
                getUserMedia: () => Promise.reject(
                    Object.assign(new Error("no"), { name: "NotAllowedError" })),
            },
        });
        h.request("camera");
        await settle();
        assert.deepEqual(h.answers, [{ kind: "camera", status }], after);
    }
});

test("a refusal a browser cannot explain stays denied", async () => {
    // No descriptor to read back, so there is still no way to tell the two
    // apart — and denied is the direction whose remedy is harmless to offer.
    const h = harness({
        mediaDevices: {
            getUserMedia: () => Promise.reject(
                Object.assign(new Error("no"), { name: "NotAllowedError" })),
        },
    });
    h.request("camera");
    await settle();
    assert.deepEqual(h.answers, [{ kind: "camera", status: "denied" }]);
});

test("a browser contradicting itself does not hand back a grant", async () => {
    // A query answering "granted" straight after a rejected getUserMedia is a
    // browser disagreeing with itself. Reporting the grant would give the app
    // a camera that had just refused it, so everything except "prompt" stays
    // denied.
    const states = ["prompt", "granted"];
    let n = 0;
    const h = harness({
        permissions: { query: () => Promise.resolve({ state: states[n++] || "granted" }) },
        mediaDevices: {
            getUserMedia: () => Promise.reject(
                Object.assign(new Error("no"), { name: "NotAllowedError" })),
        },
    });
    h.request("camera");
    await settle();
    assert.deepEqual(h.answers, [{ kind: "camera", status: "denied" }]);
});

test("a dismissed location prompt is prompt too", async () => {
    // Same distinction through the other API's code 1, which is equally silent
    // about which of the two happened.
    const states = ["prompt", "prompt"];
    let n = 0;
    const h = harness({
        permissions: { query: () => Promise.resolve({ state: states[n++] || "prompt" }) },
        geolocation: { getCurrentPosition: (_, fail) => fail({ code: 1 }) },
    });
    h.request("location");
    await settle();
    assert.deepEqual(h.answers, [{ kind: "location", status: "prompt" }]);
});

test("a missing device is unavailable and is not read back", async () => {
    // NotFoundError is not a refusal at all, so it must not go through the
    // Block/dismissal read: there is no permission state that would describe
    // a camera the machine does not have.
    let queries = 0;
    const h = harness({
        permissions: {
            query: () => { queries++; return Promise.resolve({ state: "prompt" }); },
        },
        mediaDevices: {
            getUserMedia: () => Promise.reject(
                Object.assign(new Error("no"), { name: "NotFoundError" })),
        },
    });
    h.request("camera");
    await settle();

    assert.equal(queries, 1, "the refusal read-back ran for a device that is simply absent");
    assert.deepEqual(h.answers, [{ kind: "camera", status: "unavailable" }]);
});
