// internal/valuefixture, as browser.mjs mounts it.
//
// # Why this table exists
//
// core.ValueRange.Progress decides what three wire strings amount to — a
// determinate bar, ARIA's indeterminate one, an empty range, or no numeric
// claim at all — and until this file its answers had been compared on exactly
// one platform. android/verify runs GrMobProgress.kt against it on a JVM, which
// settles Compose. The web was left out, and not because it was easy: the two
// DOM exporters deliberately *do not implement* these rules. They write
// aria-valuenow, -valuemin and -valuemax verbatim on the argument that a
// browser applies ARIA's rules itself, which is right, and which means nothing
// in this repository had ever watched a browser do it.
//
// So browser.mjs mounts one progressbar per row in a real Chrome and reads the
// value back out of the browser's own accessibility tree — the implicit 0..100
// for a bare position, a single stated bound leaving the other at ARIA's
// default, indeterminate spelled by omitting the position, and a live counter's
// overshoot clamped into its range.
//
// # It is pinned, not transcribed
//
// wasm/verify/valuerange_test.go rebuilds this table from internal/valuefixture
// and core and fails if it has drifted, in either direction. A case added to
// the fixture without a row here is a rule that is still settled on the JVM
// alone; a row here for a case Go does not have is a bar held to an answer no
// authority produced.
//
// `reading`, `now`, `min` and `max` are core.Progress's own answer, carried
// across rather than recomputed — a second implementation of ARIA's defaulting
// in a check *of* ARIA's defaulting would agree with itself and prove nothing.
//
// # parses, and the divergence it marks
//
// `parses` is core.ValueRange.Unparsed finding nothing, and it splits the table
// into the two halves browser.mjs asserts in opposite directions.
//
// A row that parses must agree with Go. A row that does not must not: Chrome
// reads aria-valuenow="half" as 0 and pins the bar at the start where Go and
// Compose both read it as absent, and reads aria-valuemax="lots" as 0, which
// inverts the range and clamps a bar at 45% into announcing as complete. Three
// implementations, three different wrong answers, none of them the one written.
//
// That half is a pinned divergence rather than a wish. If a browser ever starts
// applying ARIA's defaults to an unreadable number, the pass fails — which is
// exactly when somebody should hear about it. core.AuditTree reports the same
// shape in Go, as ConcernUnusableValueRange, so an author is told before it
// reaches any of the three.
export const VALUE_RANGES = [
    {
        name: "a bare percentage",
        wire: { Now: "45", Min: "0", Max: "100", Text: "" },
        reading: "determinate", now: 45, min: 0, max: 100, parses: true,
    },
    {
        name: "a position with no bounds",
        wire: { Now: "45", Min: "", Max: "", Text: "" },
        reading: "determinate", now: 45, min: 0, max: 100, parses: true,
    },
    {
        name: "a position with only a max",
        wire: { Now: "3", Min: "", Max: "5", Text: "" },
        reading: "determinate", now: 3, min: 0, max: 5, parses: true,
    },
    {
        name: "a position with only a min",
        wire: { Now: "3", Min: "1", Max: "", Text: "" },
        reading: "determinate", now: 3, min: 1, max: 100, parses: true,
    },
    {
        name: "a stated zero",
        wire: { Now: "0", Min: "0", Max: "100", Text: "" },
        reading: "determinate", now: 0, min: 0, max: 100, parses: true,
    },
    {
        name: "a non-percentage range",
        wire: { Now: "3", Min: "1", Max: "5", Text: "" },
        reading: "determinate", now: 3, min: 1, max: 5, parses: true,
    },
    {
        name: "a position above the max",
        wire: { Now: "150", Min: "0", Max: "100", Text: "" },
        reading: "determinate", now: 100, min: 0, max: 100, parses: true,
    },
    {
        name: "a position below the min",
        wire: { Now: "-4", Min: "0", Max: "100", Text: "" },
        reading: "determinate", now: 0, min: 0, max: 100, parses: true,
    },
    {
        name: "bounds with no position",
        wire: { Now: "", Min: "0", Max: "100", Text: "" },
        reading: "indeterminate", now: 0, min: 0, max: 0, parses: true,
    },
    {
        name: "a max with no position",
        wire: { Now: "", Min: "", Max: "100", Text: "" },
        reading: "indeterminate", now: 0, min: 0, max: 0, parses: true,
    },
    {
        name: "a min with no position",
        wire: { Now: "", Min: "0", Max: "", Text: "" },
        reading: "indeterminate", now: 0, min: 0, max: 0, parses: true,
    },
    {
        name: "the empty range",
        wire: { Now: "", Min: "", Max: "", Text: "" },
        reading: "unstated", now: 0, min: 0, max: 0, parses: true,
    },
    {
        name: "words alone",
        wire: { Now: "", Min: "", Max: "", Text: "almost done" },
        reading: "unstated", now: 0, min: 0, max: 0, parses: true,
    },
    {
        name: "words beside a position",
        wire: { Now: "45", Min: "", Max: "", Text: "almost done" },
        reading: "determinate", now: 45, min: 0, max: 100, parses: true,
    },
    {
        name: "an empty range",
        wire: { Now: "5", Min: "5", Max: "5", Text: "" },
        reading: "empty-range", now: 5, min: 5, max: 5, parses: true,
    },
    {
        name: "an inverted range",
        wire: { Now: "5", Min: "9", Max: "1", Text: "" },
        reading: "empty-range", now: 5, min: 9, max: 1, parses: true,
    },
    {
        name: "a position that is not a number",
        wire: { Now: "half", Min: "0", Max: "100", Text: "" },
        reading: "indeterminate", now: 0, min: 0, max: 0, parses: false,
    },
    {
        name: "a bound that is not a number",
        wire: { Now: "45", Min: "", Max: "lots", Text: "" },
        reading: "determinate", now: 45, min: 0, max: 100, parses: false,
    },
    {
        name: "a NaN position",
        wire: { Now: "NaN", Min: "0", Max: "100", Text: "" },
        reading: "indeterminate", now: 0, min: 0, max: 0, parses: false,
    },
    {
        name: "an infinite bound",
        wire: { Now: "45", Min: "0", Max: "Inf", Text: "" },
        reading: "determinate", now: 45, min: 0, max: 100, parses: false,
    },
    {
        name: "a fractional position",
        wire: { Now: "45.5", Min: "0", Max: "100", Text: "" },
        reading: "determinate", now: 45.5, min: 0, max: 100, parses: true,
    },
    {
        name: "a negative range",
        wire: { Now: "-5", Min: "-10", Max: "0", Text: "" },
        reading: "determinate", now: -5, min: -10, max: 0, parses: true,
    },
];

// --------------------------------------------------------------------------
// The verdict
// --------------------------------------------------------------------------
//
// # Why the comparison lives here rather than in browser.mjs
//
// Because half of it could never fail on any machine this repository runs on.
//
// The table above splits into two halves asserted in opposite directions: a row
// whose numbers parse must agree with core.Progress, and a row carrying an
// unreadable number must *not*. The first half fires whenever Chrome moves. The
// second is a pinned divergence — it fails only if a browser starts applying
// ARIA's defaults to a value that is not a number, which no shipping browser
// does — so the branch that would report that had never executed, and a
// mutation of it (a dropped `!`, a swapped arm, a message naming the wrong
// half) was invisible to every pass.
//
// That is the same shape aria/verify's localCopyGate had and the same fix: the
// decision is separated from the round trip that feeds it, so its inputs are
// two plain objects and all four of its answers can be handed over directly.
// valuerange_test.mjs does exactly that; browser.mjs supplies the second object
// from a real accessibility tree.

// The numbers a browser resolved one aria-value* family to, or null when it
// has no progressbar of that name at all.
//
// Read off Chrome's own accessibility tree rather than off the DOM, which is
// the entire point: the attributes are what the runtime wrote, and what is
// being asked is what the browser made of them. valuemin and valuemax are
// serialized as node properties and the position is the node's value, which is
// absent — not zero — for a bar with no aria-valuenow.
//
// aria-valuetext is deliberately outside the comparison. It is words rather
// than a number, core.Progress ignores it by design (its reading must not
// depend on a string that is announced instead of the digits), and Chrome does
// not surface it as a node property here in any case.
export function axRange(nodes, name) {
    const n = nodes.find((n) =>
        n.name?.value === name && n.role?.value === "progressbar");
    if (!n) return null;
    const props = Object.fromEntries(
        (n.properties || []).map((p) => [p.name, p.value?.value]));
    return { value: n.value?.value, min: props.valuemin, max: props.valuemax };
}

// Whether the browser's answer is core.Progress's answer, or null for a
// reading this function has no arm for.
//
// What counts as agreement is different per reading, and each difference is
// the reading's own meaning rather than a concession:
//
//	determinate    all three numbers, since that is the whole claim
//	indeterminate  no position at all. The bounds are not compared: ARIA's
//	unstated       0..100 is what a browser reports for a progressbar whether
//	               or not one was written, so a comparison there would pass
//	               for the wrong reason — and core.Progress returns zeros for
//	               both of these readings precisely because they are not a
//	               position.
//	empty-range    the bounds as stated. The position is not compared because
//	               there is nowhere for it to be: Chrome clamps it to whichever
//	               end it can reach and core.Progress reports it unclamped, and
//	               both are honest answers to a range that is not one.
//
// wasm/verify/valuerange_test.go holds this switch to core.ProgressReading, so
// a fifth reading arrives as a Go failure rather than as rows nothing asserts.
export function axAgreesWithGo(ax, row) {
    switch (row.reading) {
        case "determinate":
            return ax.value === row.now && ax.min === row.min && ax.max === row.max;
        case "indeterminate":
        case "unstated":
            return ax.value === undefined;
        case "empty-range":
            return ax.min === row.min && ax.max === row.max;
    }
    return null;
}

export const showRange = (r) =>
    `value ${r.value === undefined ? "(none)" : r.value}, min ${r.min}, max ${r.max}`;

// What is wrong with one browser answer, or null when nothing is.
//
// Four ways a row can be a problem, and the last two are the two halves of the
// divergence pointing opposite ways:
//
//	the bar is missing      it mounted and the browser did not compute it as a
//	                        progressbar, so nothing below was asked about it
//	the reading has no arm  a silent pass otherwise: the row still mounts and
//	                        is still found, and nothing is asserted of it
//	it parses and diverges  a disagreement about ARIA's own defaulting or
//	                        clamping, which the web exporters implement neither
//	                        of and rely on the browser for both
//	it does not parse and   the pinned divergence has closed. Good news, and
//	agrees                  still a failure: three places in this repository
//	                        say something that is no longer true of this browser
//
// `ax` is axRange's answer, so a caller that has one already need not look it
// up twice; null means no such progressbar.
export function valueRangeProblem(row, ax) {
    if (!ax) {
        return `no progressbar named "${row.name}" in the browser's ` +
            `accessibility tree — the row mounted and the browser did not ` +
            `compute it as a progress bar, so nothing below was asked about it`;
    }
    const agrees = axAgreesWithGo(ax, row);
    if (agrees === null) {
        return `"${row.name}" reads as ${row.reading}, which ` +
            `axAgreesWithGo has no arm for — the bar mounted, was found, and ` +
            `had nothing asserted about it`;
    }
    if (row.parses && !agrees) {
        return `"${row.name}": the browser resolved ` +
            `${JSON.stringify(row.wire)} to ${showRange(ax)}, and ` +
            `core.ValueRange.Progress says ${row.reading} at ` +
            `${showRange({ value: row.now, min: row.min, max: row.max })}. ` +
            `Every number here parses, so this is a disagreement about ARIA's ` +
            `own defaulting or clamping — and the web exporters implement ` +
            `neither, they rely on the browser for both`;
    }
    if (!row.parses && agrees) {
        return `"${row.name}": the browser now agrees with ` +
            `core.ValueRange.Progress about a range holding a value that is ` +
            `not a number. That is good news and it is still a failure: the ` +
            `divergence is written down in valuerange.mjs, in core.ValueRange` +
            `.Unparsed and in core.AuditTree's ConcernUnusableValueRange, and ` +
            `all three now say something that is no longer true of this browser`;
    }
    return null;
}
