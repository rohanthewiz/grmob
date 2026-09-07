// What browser.mjs does before it does anything: the preconditions, and the two
// different stances it takes toward them.
//
// # Two preconditions that are not the same kind of thing
//
// The browser pass needs three things it does not build itself, and they fall
// into two groups that deserve opposite answers:
//
//	a transcript      gen.go's output, which run.sh generates and points at
//	                  through GRMOB_TRANSCRIPT. It carries the widget swatches —
//	                  real components.Chip and core.Input trees rendered by Go —
//	                  which is the half of the palette check no table of hexes
//	                  in a .mjs file can reach, and the band cases, which are a
//	                  real components.GroupHeader's own geometry.
//
//	a Chrome, and a   the browser to drive and the WebSocket global to drive it
//	Node with         over. Node grew one in v21; before that there is no way to
//	WebSocket         speak CDP without a dependency, and wasm/verify's run.sh
//	                  promises Go and Node and nothing else.
//
// A missing Chrome is a fact about the *machine*. The script's promise is that
// it catches what this machine can catch, and a pass that fails on a machine
// missing an optional tool is a pass people learn to ignore — the same stance
// ios/verify takes toward a missing iPhoneOS SDK. So: SKIP, exit 0.
//
// A missing transcript is a fact about how the script was *invoked*. Nothing
// about the machine changed; somebody ran `node browser.mjs` directly, or a
// caller stopped setting the variable. Skipping there would quietly drop a
// third of the pass — the only half that can see a widget stop drawing its
// boundary tone — and report success. So: FAIL, exit 1, naming run.sh.
//
// # Why the invocation fault is decided first
//
// Because the two can be true at once, and on a machine with no Chrome the
// wrong order hides the mistake: a run with no transcript would print SKIP and
// exit 0, and the person who forgot the variable would never learn they had.
// Reporting the thing they can fix before the thing they cannot is the whole
// reason this is an ordered decision rather than two independent guards.
//
// # Why it is a function
//
// Both guards used to be inline — one at module scope, one at the top of
// main() — which made every answer reachable only by arranging a machine that
// had the fault. Nothing had ever run the SKIP arms on a machine with a Chrome,
// or the FAIL arms on one without, and a swapped stance (a skip where a failure
// belongs) is exactly the mutation that leaves a pass green. Here the inputs are
// three booleans and a string, so startup_test.mjs hands over every combination
// directly. It is aria/verify's localCopyGate, one target over.

/**
 * The decision, given what the machine and the invocation supply.
 *
 * `action` is one of:
 *
 *	"fail"  the invocation is wrong. Print `why` on stderr and exit 1.
 *	"skip"  the machine cannot answer the question. Print `why` and exit 0.
 *	"run"   go ahead; `why` is empty.
 *
 * `widgets` and `bands` are how many of each table the transcript carried, and
 * are only meaningful when it was readable at all — a transcript that exists and
 * carries an empty table is its own failure, because the check that reads it
 * would then pass by having no subject.
 */
export function startupVerdict({
    transcriptPath, transcriptExists, widgets, bands, hasWebSocket, chromePath,
}) {
    // The invocation first. See the note above: on a machine with no Chrome the
    // other order turns a forgotten environment variable into a green run.
    if (!transcriptPath || !transcriptExists) {
        return {
            action: "fail",
            why: "browser.mjs needs the transcript gen.go writes.\n" +
                "  Run wasm/verify/run.sh, which generates it and sets GRMOB_TRANSCRIPT,\n" +
                "  or set GRMOB_TRANSCRIPT to the output of `go run ./wasm/verify`.",
        };
    }
    if (!widgets) {
        return {
            action: "fail",
            why: "the transcript carries no widget swatches — gen.go's widgetCases() " +
                "produced nothing, so the half of the palette check that goes through " +
                "`components` would pass by having no subject.",
        };
    }
    if (!bands) {
        return {
            action: "fail",
            why: "the transcript carries no band cases — internal/bandfixture produced " +
                "nothing, so the check that asks a browser whether the band's insets " +
                "survive overflow has no subject.",
        };
    }
    if (!hasWebSocket) {
        return { action: "skip", why: "this Node has no WebSocket global; v21 or newer has one" };
    }
    if (!chromePath) {
        return { action: "skip", why: "no Chrome or Chromium found; set GRMOB_CHROME to point at one" };
    }
    return { action: "run", why: "" };
}
