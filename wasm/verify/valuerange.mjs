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
