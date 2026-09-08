// Whether a grid the browser pass mounted is still on screen when the
// screenshot is taken.
//
// # Why this is a decision and not an `if`
//
// browser.mjs mounts three grids of real widgets — swatches, chips, bands —
// and reads pixels out of one screenshot apiece. A screenshot is taken with
// `captureBeyondViewport: false`, so a row the grid pushed past the bottom of
// the window has a perfectly good rect and no pixels at all: `pixelAt` answers
// null for every sample, and the check reports three failures naming colours,
// none of which says the thing was never on screen.
//
// So each grid guards itself first, and both guards were the same four lines
// written twice. They are this now, for the reason internal/gateharness holds
// the shape of two gate tests: one answer to one question.
//
// # And the arm that has never run
//
// Nine bands and six chips fit an 800px window comfortably, so neither guard
// has ever fired — the only way to trip the band one is to bundle more themes.
// It is written out and stated rather than hidden, which is honest, and a guard
// nothing has ever executed is a guard nobody has checked. As a function of
// values its arms are reachable by handing it two numbers, which is what
// fold_test.mjs does. Same move as gate.sh's jvm_harness_verdict, browser.mjs's
// startupVerdict and mobile/verify's composeSourcesVerdict.
//
// # The tolerance
//
// Rects are CSS pixels and the screenshot's height is device pixels divided by
// the device pixel ratio, so a grid that ends exactly at the fold arrives here
// as two numbers that can differ in the last place. Half a pixel is far below
// anything that could hide a row and far above that.
export const FOLD_EPSILON = 0.5;

// foldVerdict returns "" while the grid is on screen, and the whole sentence to
// report when it is not.
//
//	what    names the thing that ran off, as the failure will read
//	bottom  the bottom edge of the last thing measured, in CSS pixels
//	screen  the screenshot's height in CSS pixels (image height / dpr)
//	knob    what a reader is supposed to turn — the constant that decides how
//	        many rows the grid has, or the window size
//
// A zero-height screenshot takes the same arm as an overflowing grid, and
// deliberately: a capture that came back empty is the one state where every
// pixel sample is null and nothing else in the pass would say why.
export function foldVerdict({ what, bottom, screen, knob }) {
    if (!(bottom > screen + FOLD_EPSILON)) return "";
    return `${what}: the grid runs past the bottom of the viewport (this one ends ` +
        `at ${Math.round(bottom)}px of a ${Math.round(screen)}px screenshot), so ` +
        `nothing was painted where its rect says it is — ${knob} needs to grow with ` +
        `what the grid lays out`;
}
