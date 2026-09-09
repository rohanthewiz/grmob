// Every arm of the tap-target ledger's reading, reached without editing the
// band loop.
//
// The three drift arms — a part with no verdict, a verdict on a part this
// branch does not make, a verdict under a name the table does not have — exist
// because the conjunction the tail recites and the three counts the census
// reports used to be two things that agreed by construction. They are one
// reading now, and the reading's arms are reachable only by writing a ledger
// the loop cannot write. That is the shape fold.mjs was moved out of, and this
// is the same move: as a function of values, every one of them is a call.
import test from "node:test";
import assert from "node:assert/strict";

import { BAND_TARGET_PARTS, bandTargetCensus, bandTargetRead }
    from "./bandtarget.mjs";

const held = (where, target, hasWrapper) => bandTargetRead(where, target, hasWrapper);

// The two ledgers the band loop actually writes: a plain band, which makes two
// of the three claims, and a disclosure band, which makes all three.
const plain = { lead: true, trail: true };
const disclosure = { lead: true, trail: true, stretch: true };

test("the table gives every part a key, a counter and a population", () => {
    // The census reads `counter` and `everyBand` off these rows and browser.mjs
    // derives `declared` from them, so a row missing either is a part that
    // arrives in the claim without a number under it.
    assert.ok(BAND_TARGET_PARTS.length > 0, "the tap-target claim has no parts");
    const keys = new Set(), counters = new Set();
    for (const p of BAND_TARGET_PARTS) {
        assert.equal(typeof p.key, "string", `a part has no key: ${JSON.stringify(p)}`);
        assert.equal(typeof p.counter, "string", `${p.key} has no census counter`);
        assert.equal(typeof p.what, "string", `${p.key} has nothing to call itself`);
        assert.equal(typeof p.everyBand, "boolean",
            `${p.key} does not say which bands make the claim`);
        assert.equal(typeof p.census, "string",
            `${p.key} has nothing for the census line to call it`);
        assert.ok(!keys.has(p.key), `two parts are keyed ${p.key}`);
        assert.ok(!counters.has(p.counter), `two parts count at ${p.counter}`);
        keys.add(p.key);
        counters.add(p.counter);
    }
});

test("a plain band's full ledger is whole and earns its parts' counters", () => {
    const read = held("light/plain", plain, false);
    assert.deepEqual(read.problems, []);
    assert.equal(read.whole, true);
    // Two counters, not three: the stretch is the disclosure branch's
    // mechanism and a plain band is not counted in a population it cannot join.
    assert.deepEqual(read.counters.sort(), ["targetLead", "targetTrail"]);
});

test("a disclosure band's full ledger earns all three", () => {
    const read = held("light/disclosure", disclosure, true);
    assert.deepEqual(read.problems, []);
    assert.equal(read.whole, true);
    assert.deepEqual(read.counters.sort(),
        ["targetLead", "targetStretch", "targetTrail"]);
});

test("a part that failed clears the conjunction and earns no counter", () => {
    // The derivation, in the direction it exists for: `whole` is not a boolean
    // an assertion writes, it is what the parts come to.
    const read = held("light/plain", { lead: true, trail: false }, false);
    assert.equal(read.whole, false);
    assert.deepEqual(read.counters, ["targetLead"]);
    // A false verdict is a decided assertion. The band loop has already
    // reported what it means in pixels, so nothing here says it twice.
    assert.deepEqual(read.problems, []);
});

test("a part with no verdict is reported and counted in neither", () => {
    const read = held("light/plain", { lead: true }, false);
    assert.equal(read.whole, false);
    assert.deepEqual(read.counters, ["targetLead"]);
    assert.equal(read.problems.length, 1);
    assert.ok(read.problems[0].includes("nothing decided"),
        `an undecided part is not reported as one: ${read.problems[0]}`);
    assert.ok(read.problems[0].includes("trailing edge"),
        `the message does not say which part: ${read.problems[0]}`);
});

test("a stretch verdict on a band with no wrapper is reported", () => {
    // The population arm: the stretch is counted over the bands that have a
    // wrapper, so a verdict from outside that set would be a count the tail
    // could not state a denominator for.
    const read = held("light/plain", { ...plain, stretch: true }, false);
    assert.equal(read.whole, false);
    assert.deepEqual(read.counters.sort(), ["targetLead", "targetTrail"]);
    assert.equal(read.problems.length, 1);
    assert.ok(read.problems[0].includes("does not make that claim"),
        `a verdict outside the population is not reported: ${read.problems[0]}`);
});

test("a verdict the table has no row for is reported by name", () => {
    // The arm the whole table exists for: a fourth assertion added to the claim
    // and not to BAND_TARGET_PARTS would drop the conjunction while the census
    // went on reciting three full populations underneath it.
    const read = held("light/disclosure", { ...disclosure, taller: true }, true);
    assert.equal(read.whole, false);
    assert.equal(read.problems.length, 1);
    assert.ok(read.problems[0].includes('"taller"'),
        `the message does not name the unknown verdict: ${read.problems[0]}`);
    assert.ok(read.problems[0].includes("BAND_TARGET_PARTS"),
        `the message does not say where the row belongs: ${read.problems[0]}`);
});

test("a wrapper band missing its stretch verdict is reported", () => {
    // The other side of the population arm: the same absent key is silence on
    // a plain band and a missing assertion on a disclosure one.
    const wanted = held("light/disclosure", plain, true);
    assert.equal(wanted.whole, false);
    assert.equal(wanted.problems.length, 1);
    assert.ok(wanted.problems[0].includes("stretched across the heading wrapper"),
        `the message does not name the missing part: ${wanted.problems[0]}`);

    const notWanted = held("light/plain", plain, false);
    assert.deepEqual(notWanted.problems, []);
});

test("reading a ledger twice reads the same thing", () => {
    // The reason the reading and the spending came apart: this used to
    // increment the census, so asking a second time moved the numbers the tail
    // recites.
    const target = { ...disclosure, taller: true };
    const first = held("light/disclosure", target, true);
    const second = held("light/disclosure", target, true);
    assert.deepEqual(first, second);
});

// The population half of the same table, which used to be a ternary in
// browser.mjs and a phrase in the census table beside it.
//
// The property under test is the join: bandTargetCensus counts a part's
// population with the SAME expression bandTargetRead decides `wanted` with, so
// the set the loop asks for a verdict from and the set the census divides by
// cannot come apart. Both halves are functions of values here, which is what
// makes the property assertable at all.

// Four bands, one of them on the disclosure branch.
const grid = [{ w: false }, { w: true }, { w: false }, { w: false }];
const wrapper = (b) => b.w;

test("the census counts every part over the population its reading wants", () => {
    const rows = bandTargetCensus(grid, wrapper);
    assert.equal(rows.length, BAND_TARGET_PARTS.length);
    for (const row of rows) {
        const part = BAND_TARGET_PARTS.find((p) => p.counter === row.counter);
        assert.ok(part, `${row.counter} is counted and no part earns it`);
        assert.equal(row.subject, part.census);
        // The declared population, recomputed from the reading's own answer:
        // a band earns a place in it exactly when a full ledger for that band
        // hands back this part's counter.
        const wanted = grid.filter((b) => {
            const target = {};
            for (const q of BAND_TARGET_PARTS) {
                if (q.everyBand || wrapper(b)) target[q.key] = true;
            }
            return bandTargetRead("x", target, wrapper(b))
                .counters.includes(part.counter);
        }).length;
        assert.equal(row.declared, wanted,
            `${row.counter} is declared over ${row.declared} bands and the ` +
            `reading hands its counter to ${wanted}`);
    }
});

test("the census names the population it counted", () => {
    const rows = bandTargetCensus(grid, wrapper);
    const by = new Map(rows.map((r) => [r.counter, r]));
    // Both halves of the phrase move with `everyBand` and nothing else: an
    // every-band part is counted over the whole grid and says so, and a
    // wrappered one is counted over the one band with a wrapper and says that.
    for (const part of BAND_TARGET_PARTS) {
        const row = by.get(part.counter);
        assert.equal(row.declared,
            part.everyBand ? grid.length : grid.filter(wrapper).length);
        assert.equal(row.population.includes("wrapper"), !part.everyBand,
            `${part.counter} counts ${row.declared} of ${grid.length} bands and ` +
            `calls them "${row.population}"`);
    }
});

test("an empty grid declares nothing and still gives every part a row", () => {
    // The census is a table of rows and not a table of counts: a run that
    // mounted no bands must still produce a row per part, or a part could
    // vanish from the census by the grid being empty rather than by anybody
    // deciding it should.
    const rows = bandTargetCensus([], wrapper);
    assert.equal(rows.length, BAND_TARGET_PARTS.length);
    for (const row of rows) assert.equal(row.declared, 0);
});
