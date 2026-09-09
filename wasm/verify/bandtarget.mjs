// One band's tap-target ledger: what it comes to, and what it says when it has
// drifted from the table the claim is declared in.
//
// # Why this is a module
//
// browser.mjs's band grid writes a verdict per assertion the tap-target claim
// is made of and reduces them at the bottom of the loop. Three of the arms
// that reduction can take — a part with no verdict, a verdict on a part this
// branch does not make, a verdict under a name the table does not have — are
// reachable only by editing the loop that writes the ledger. That is the shape
// fold.mjs, startupVerdict and gate.sh's jvm_harness_verdict were all moved
// out of: a decision whose arms are reachable only by owning the code in the
// state that has the fault. As a function of values they are reachable by
// handing it a ledger, which is what bandtarget_test.mjs does.
//
// # And why the reading does not spend the census
//
// The previous version counted and reduced in one pass and its side effect was
// the census: it took `asked` and incremented it, which is what made the
// conjunction and the three counts one reading — and it also meant a ledger
// could not be asked what it comes to without moving the counters on. A caller
// that wanted to check one twice, or a test that wanted to put a contrived one
// in front of it, paid for the question in the numbers the tail recites.
//
// So the reading is here and returns what it found; the spending is three
// lines in browser.mjs over what this returned. The property that mattered
// survives it: the counters and the conjunction still come out of ONE reading,
// and the conjunction is still derived from the parts rather than carried
// beside them.

// The assertions the tap-target claim is made of, named once.
//
// # A conjunction the census could not see
//
// The tail recites `targets` — how many bands got a tap target that spans the
// band — and the census counts the three assertions that claim is made of
// apart, so a shortfall says which of them cost it. The two agreed by
// construction and nothing said they had to: `targetWhole` was a boolean set
// false at three sites and the three counters were incremented at three
// others, so a fourth assertion added to the claim could clear the conjunction
// while leaving the census reciting three full populations under it.
//
// Naming the parts here is what joins them. The band loop writes a verdict per
// part into a ledger; bandTargetRead derives the conjunction FROM that ledger
// rather than from a boolean an assertion can reach, names each part's own
// counter, and reports a ledger that does not match this table. browser.mjs's
// `declared` takes each part's census population from the same rows, so a part
// added here arrives with a counter and a population and cannot arrive without
// them.
//
// The stretch's population is the bands that HAVE a wrapper — it is the
// disclosure branch's mechanism, and the plain branch has no equivalent — so
// `everyBand` is what the census reads and what says whether a missing verdict
// is a branch that does not make the claim or an assertion that did not run.
//
// `census` is what the tail's census line calls the assertion, and it is here
// for the same reason `counter` is: see bandTargetCensus, which is the one
// place `everyBand` is turned into a population, a count and a phrase.
export const BAND_TARGET_PARTS = [
    {
        key: "lead", counter: "targetLead", everyBand: true,
        what: "the control's leading edge against the band's",
        census: "their tap target's leading edge held to the band's",
    },
    {
        key: "trail", counter: "targetTrail", everyBand: true,
        what: "the control's trailing edge against the band's content",
        census: "its trailing edge held to where the band's content ends",
    },
    {
        key: "stretch", counter: "targetStretch", everyBand: false,
        what: "the button stretched across the heading wrapper",
        census: "the button held to the wrapper it is stretched across",
    },
];

// What the census may say about each part, derived where `everyBand` is read.
//
// # One flag, three readers, and two of them were somewhere else
//
// `everyBand` decides three things: which bands bandTargetRead WANTS a verdict
// from, how many bands the census line divides by, and what that line calls the
// set it is dividing by. The first is here; the other two were in browser.mjs,
// spelled again — a ternary picking between `BAND_RENDERS.length` and the
// wrappered ones, and a literal phrase in the census table beside it.
//
// Three spellings of one predicate is a join nothing makes. A part whose
// `everyBand` is wrong gets a population and a counter that agree with each
// other and with nothing the band loop did, and what a reader meets is a census
// line short by twelve on every run with no message naming the row that is
// wrong. Derived here, the same flag moves all three at once: the loop stops
// wanting the verdict exactly when the count stops expecting it, and the drift
// arrives as bandTargetRead's own message — "nothing decided X", or "X came
// back decided on a band that does not make that claim" — which names the part.
//
//	bands       the band records the grid mounted, in gen.go's order
//	hasWrapper  the predicate that says a band is on the disclosure branch,
//	            passed in because a band record's shape is browser.mjs's and
//	            not this module's
//
// Returns one row per part: the counter the census reads, how many bands are
// declared to earn it, and how the line names that population.
export function bandTargetCensus(bands, hasWrapper) {
    return BAND_TARGET_PARTS.map((part) => ({
        counter: part.counter,
        subject: part.census,
        // The same phrase the population is counted with, so a row cannot say
        // one set and count another.
        population: part.everyBand
            ? "bands in this grid" : "bands mounted behind a heading wrapper",
        // And the same expression bandTargetRead applies per band as `wanted`.
        declared: bands.filter((b) => part.everyBand || hasWrapper(b)).length,
    }));
}

// What one band's ledger comes to.
//
//	where       names the band, as the failure will read
//	target      the ledger: one boolean per BAND_TARGET_PARTS key the loop
//	            decided, and no key for a part this branch does not make
//	hasWrapper  whether this band has a heading wrapper, which is the one
//	            thing that decides which parts are wanted
//
// Returns the messages a drifted ledger produces, the counters the parts that
// held have earned, and whether the whole claim stands. Nothing is incremented
// here: see the note at the top of this file.
export function bandTargetRead(where, target, hasWrapper) {
    const problems = [];
    const counters = [];
    let whole = true;
    for (const part of BAND_TARGET_PARTS) {
        // The same expression bandTargetCensus counts the population with. See
        // there: one flag, and the loop's appetite and the census's divisor
        // move together.
        const wanted = part.everyBand || hasWrapper;
        const held = target[part.key];
        if (held === undefined) {
            if (wanted) {
                whole = false;
                problems.push(`${where}: nothing decided ${part.what}, which is one of ` +
                    `the ${BAND_TARGET_PARTS.length} assertions the tap-target claim ` +
                    `is made of. The tail recites the conjunction of them and the ` +
                    `census counts them apart, and a part with no verdict is a band ` +
                    `counted in neither`);
            }
            continue;
        }
        if (!wanted) {
            whole = false;
            problems.push(`${where}: ${part.what} came back decided on a band that ` +
                `does not make that claim. Its census population is the bands with a ` +
                `heading wrapper, so a verdict from outside that set is a count over ` +
                `a population the tail cannot state`);
            continue;
        }
        if (held) counters.push(part.counter);
        else whole = false;
    }
    for (const key of Object.keys(target)) {
        if (BAND_TARGET_PARTS.some((p) => p.key === key)) continue;
        whole = false;
        problems.push(`${where}: the tap-target ledger came back with a verdict for ` +
            `${JSON.stringify(key)} and BAND_TARGET_PARTS has no row for it.\n\n` +
            `A fourth assertion added to this claim has to be added there, where it ` +
            `gets a counter and a census population. Left out, it would drop the ` +
            `conjunction the tail recites while the census went on reporting three ` +
            `full populations underneath it — which is the shape of shortfall this ` +
            `table exists to make impossible`);
    }
    return { problems, whole, counters };
}
