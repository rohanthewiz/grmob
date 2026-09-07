// valuerange.mjs's verdict, with all four of its answers handed over directly.
//
// # The gap this closes
//
// browser.mjs asserts the value-range table in two opposite directions: a row
// whose numbers all parse must agree with core.ValueRange.Progress, and a row
// carrying an unreadable number must *not*. Only the first of those can fail on
// a machine with a shipping browser. The second is a pinned divergence — Chrome
// reads aria-valuenow="half" as 0 and pins the bar at the start where Go and
// Compose both read it as absent — so the branch that reports "the browser now
// agrees" had never executed anywhere, and a mutation of it was invisible to
// every pass in this repository.
//
// That is the shape aria/verify's localCopyGate had before it was extracted,
// and this is the same fix one target over: the decision's inputs are two plain
// objects, so a test can hand it the answer a browser will not give.
//
// # What is deliberately not here
//
// Whether Chrome actually resolves these ranges the way the table says. That is
// browser.mjs's job and it needs a browser; a shim would be this repository
// implementing ARIA's defaulting in order to check its own claim about ARIA's
// defaulting, which is the thing valuerange.mjs's header refuses. What is under
// test here is only the verdict *given* an answer.

import test from "node:test";
import assert from "node:assert/strict";

import {
    VALUE_RANGES, axRange, axAgreesWithGo, valueRangeProblem,
} from "./valuerange.mjs";

// The row the fixture uses for each case, looked up by name rather than
// written out: a row typed here would be a second copy of an answer
// valuerange_test.go pins to Go, and the two would drift.
function row(name) {
    const r = VALUE_RANGES.find((r) => r.name === name);
    assert.ok(r, `internal/valuefixture has no case named "${name}" — this test ` +
        `is written against rows valuerange_test.go pins to Go, and one of them ` +
        `has been renamed or dropped`);
    return r;
}

// What a browser would report for a row it agreed with Go about.
//
// Per reading, because agreement is per reading: an indeterminate bar agrees by
// having no position at all, and a row's resolved numbers are zeros there
// precisely because they are not a position. Building this from `now`, `min`
// and `max` alone would hand the indeterminate rows a bar pinned at 0 — which
// is Chrome's *disagreement*, and the test below would then be asserting that
// the divergence is still open rather than that closing it is caught.
function agreeing(r) {
    switch (r.reading) {
        case "determinate":
            return { value: r.now, min: r.min, max: r.max };
        case "indeterminate":
        case "unstated":
            // ARIA's implicit range, which a browser reports whether or not one
            // was written; what agreement turns on is the absent position.
            return { value: undefined, min: 0, max: 100 };
        case "empty-range":
            return { value: r.min, min: r.min, max: r.max };
    }
    throw new Error(`no agreeing answer for the ${r.reading} reading`);
}

// --------------------------------------------------------------------------
// The two halves of the divergence
// --------------------------------------------------------------------------

test("a row whose numbers parse passes when the browser agrees", () => {
    const r = row("a bare percentage");
    assert.equal(valueRangeProblem(r, agreeing(r)), null);
});

test("a row whose numbers parse fails when the browser disagrees", () => {
    const r = row("a bare percentage");
    const problem = valueRangeProblem(r, { value: 0, min: 0, max: 100 });
    assert.ok(problem, "a browser that pinned a 45% bar at 0 was accepted");
    assert.match(problem, /Every number here parses/);
    assert.match(problem, /value 0, min 0, max 100/);
    assert.match(problem, /value 45, min 0, max 100/);
});

// The half no machine can reach. Chrome reads "half" as 0 and reports the bar
// at the start of its range; Go reads it as no position at all. The row is only
// checked *for* that disagreement, so a browser answering the way Go does is
// the failure — and this is the only place that branch has ever run.
test("a row that does not parse passes when the browser diverges", () => {
    const r = row("a position that is not a number");
    assert.equal(r.parses, false);
    // Chrome's actual answer: a position of 0 where Go says there is none.
    assert.equal(valueRangeProblem(r, { value: 0, min: 0, max: 100 }), null);
});

test("a row that does not parse fails when the browser starts agreeing", () => {
    const r = row("a position that is not a number");
    const problem = valueRangeProblem(r, agreeing(r));
    assert.ok(problem, "the pinned divergence closed and the pass stayed green — " +
        "which is the whole failure this branch exists to report");
    assert.match(problem, /the browser now agrees/);
    assert.match(problem, /good news and it is still a failure/);
});

// The other unparseable shape, and it fails in the opposite direction to the
// one above: an unreadable *bound* leaves the position readable, so Go still
// reads the bar as determinate at 45 and the divergence is in the range Chrome
// inverts around it.
test("an unreadable bound is pinned the same way as an unreadable position", () => {
    const r = row("a bound that is not a number");
    assert.equal(r.reading, "determinate");
    // Chrome reads aria-valuemax="lots" as 0, inverting the range.
    assert.equal(valueRangeProblem(r, { value: 45, min: 0, max: 0 }), null);
    assert.match(valueRangeProblem(r, agreeing(r)), /the browser now agrees/);
});

// --------------------------------------------------------------------------
// The two answers that are neither half
// --------------------------------------------------------------------------

test("a bar the browser did not compute as a progressbar is a failure", () => {
    const problem = valueRangeProblem(row("a bare percentage"), null);
    assert.match(problem, /no progressbar named "a bare percentage"/);
    assert.match(problem, /nothing below was asked about it/);
});

// A fifth core.ProgressReading arrives exactly this way: the row mounts, is
// found, and the switch falls through — which without this arm is a silent
// pass. valuerange_test.go's TestEveryReadingIsOneTheBrowserChecks is what
// stops the reading being added in the first place; this is what the harness
// does if it is.
test("a reading the switch has no arm for is a failure, not a pass", () => {
    const invented = { ...row("a bare percentage"), reading: "half-full" };
    const problem = valueRangeProblem(invented, { value: 45, min: 0, max: 100 });
    assert.match(problem, /has no arm for/);
});

// --------------------------------------------------------------------------
// What each reading compares, which is different per reading
// --------------------------------------------------------------------------

test("an indeterminate bar is judged on having no position at all", () => {
    const r = row("bounds with no position");
    // Chrome's actual answer for this row: ARIA's implicit range around no
    // position at all.
    assert.equal(axAgreesWithGo({ value: undefined, min: 0, max: 100 }, r), true);
    assert.equal(axAgreesWithGo({ value: 0, min: 0, max: 0 }, r), false);
    // And the bounds are outside the comparison, said in the one way the rows
    // themselves cannot say it. A browser reports ARIA's implicit 0..100 for a
    // progressbar whether or not one was written, so comparing bounds here
    // would pass for the wrong reason — but every indeterminate row's own min
    // is 0 and so is Chrome's, which means a comparison added against `min`
    // agrees by coincidence with every case in the fixture. Bounds nothing
    // would ever report are what make the omission observable.
    assert.equal(axAgreesWithGo({ value: undefined, min: 42, max: 99 }, r), true);
});

test("an empty range is judged on its bounds and not on its position", () => {
    const r = row("an inverted range");
    // Chrome clamps the position to whichever end it can reach and Go reports
    // it unclamped; both are honest answers to a range that is not one.
    assert.equal(axAgreesWithGo({ value: 1, min: 9, max: 1 }, r), true);
    assert.equal(axAgreesWithGo({ value: 5, min: 0, max: 100 }, r), false);
});

// --------------------------------------------------------------------------
// axRange
// --------------------------------------------------------------------------

// A node list shaped the way Accessibility.getFullAXTree hands one over.
const axNode = (name, role, value, props) => ({
    name: { value: name },
    role: { value: role },
    ...(value === undefined ? {} : { value: { value } }),
    properties: Object.entries(props).map(([n, v]) => ({ name: n, value: { value: v } })),
});

test("axRange reads a bar's three numbers off the tree", () => {
    const nodes = [
        axNode("a bare percentage", "progressbar", 45, { valuemin: 0, valuemax: 100 }),
    ];
    assert.deepEqual(axRange(nodes, "a bare percentage"), { value: 45, min: 0, max: 100 });
});

// Absent, not zero. The whole indeterminate arm rests on the difference, and a
// reader that defaulted a missing value to 0 would turn every indeterminate row
// into a bar pinned at the start — which is exactly Chrome's wrong answer for
// the unparseable rows, arriving from this side instead.
test("a bar with no position reads as undefined rather than zero", () => {
    const nodes = [axNode("bounds with no position", "progressbar", undefined,
        { valuemin: 0, valuemax: 100 })];
    assert.equal(axRange(nodes, "bounds with no position").value, undefined);
});

test("axRange matches on the role as well as the name", () => {
    const nodes = [axNode("a bare percentage", "generic", 45, {})];
    assert.equal(axRange(nodes, "a bare percentage"), null);
});
