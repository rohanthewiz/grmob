// browser.mjs's preconditions, with every answer handed over directly.
//
// Two preconditions with two opposite stances — a missing transcript is a
// failure, a missing Chrome is a skip — and until startup.mjs they were inline
// guards whose arms could only be reached by arranging a machine that had the
// fault. On a developer's laptop the FAIL arms never ran; in a container
// without Chrome the SKIP arms did and the transcript arms did not. A swapped
// stance is the mutation that matters here, and it is the one that leaves the
// pass green: a skip where a failure belongs turns "a third of this check did
// not run" into a line nobody reads.
//
// So the inputs are two booleans, four counts and a string. See startup.mjs
// for why the stances differ and why the invocation fault is decided first.

import test from "node:test";
import assert from "node:assert/strict";

import { startupVerdict } from "./startup.mjs";

// A machine and an invocation with nothing wrong, which each case below
// spoils in exactly one way.
const ok = {
    transcriptPath: "/tmp/grmob-wasm-verify/transcript.json",
    transcriptExists: true,
    widgets: 6,
    bands: 3,
    bandRenders: 9,
    pins: 4,
    hasWebSocket: true,
    chromePath: "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
};

const verdict = (spoil) => startupVerdict({ ...ok, ...spoil });

test("a machine and an invocation with nothing wrong runs", () => {
    assert.equal(verdict({}).action, "run");
});

// --------------------------------------------------------------------------
// The invocation: a failure, because nothing about the machine is wrong
// --------------------------------------------------------------------------

test("no GRMOB_TRANSCRIPT is a failure, not a skip", () => {
    const v = verdict({ transcriptPath: undefined, transcriptExists: false });
    assert.equal(v.action, "fail");
    assert.match(v.why, /GRMOB_TRANSCRIPT/);
    assert.match(v.why, /run\.sh/);
});

test("a GRMOB_TRANSCRIPT naming a file that is not there is a failure", () => {
    assert.equal(verdict({ transcriptExists: false }).action, "fail");
});

// A transcript that parses and carries nothing is the worst of the three,
// because it is the one that looks fine. Every widget assertion would run zero
// times and the pass would print its own OK.
test("a transcript with no widget swatches is a failure", () => {
    const v = verdict({ widgets: 0 });
    assert.equal(v.action, "fail");
    assert.match(v.why, /pass by having no subject/);
});

test("a transcript with no band cases is a failure", () => {
    const v = verdict({ bands: 0 });
    assert.equal(v.action, "fail");
    assert.match(v.why, /bandfixture/);
});

// The two band tables are separate preconditions because they are separate
// subjects: one is the band as arithmetic and the other is the rendered widget,
// and an empty one of either silences a different pair of claims.
test("a transcript with no rendered bands is a failure", () => {
    const v = verdict({ bandRenders: 0 });
    assert.equal(v.action, "fail");
    assert.match(v.why, /bandRenders/);
});

// The pin cases are a third subject again: one overflowing Row in four
// arrangements, which is the only place a browser is asked what
// core.FlexShrink(0) does rather than told by a solver.
test("a transcript with no pinned-Row cases is a failure", () => {
    const v = verdict({ pins: 0 });
    assert.equal(v.action, "fail");
    assert.match(v.why, /pinfixture/);
});

// --------------------------------------------------------------------------
// The machine: a skip, because the promise is what this machine can catch
// --------------------------------------------------------------------------

test("no Chrome is a skip, not a failure", () => {
    const v = verdict({ chromePath: null });
    assert.equal(v.action, "skip");
    assert.match(v.why, /GRMOB_CHROME/);
});

test("a Node with no WebSocket global is a skip", () => {
    const v = verdict({ hasWebSocket: false });
    assert.equal(v.action, "skip");
    assert.match(v.why, /v21/);
});

// --------------------------------------------------------------------------
// The order, which is the only thing the two guards being separate could not
// state
// --------------------------------------------------------------------------

// The case the ordering exists for. Both faults are present; the one the person
// running the script can fix is the one they are told about, because the other
// order prints SKIP and exits 0 and they never learn the variable was missing.
test("a missing transcript is reported ahead of a missing Chrome", () => {
    const v = verdict({ transcriptExists: false, chromePath: null, hasWebSocket: false });
    assert.equal(v.action, "fail");
    assert.match(v.why, /GRMOB_TRANSCRIPT/);
});

test("an empty transcript is reported ahead of a missing Chrome", () => {
    const v = verdict({ widgets: 0, chromePath: null });
    assert.equal(v.action, "fail");
});
