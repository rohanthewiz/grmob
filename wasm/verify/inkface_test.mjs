// GRMOB_INK_FACE's Preferences write, without a Chrome.
//
// The failure this guards is quiet: a face written under the wrong key, or for
// one family only, leaves Chrome on its default serif, and the grid then reads
// like a machine that resolved Times. So each case below asserts the exact
// path Chrome reads, and the one merge case asserts nothing else was lost.
// See inkface.mjs for why those keys.

import test from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, existsSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import {
    INK_FACE_FAMILIES,
    INK_FACE_SCRIPT,
    INK_FACE_SCRIPTS,
    inkFacePreferences,
    writeInkFaceOverride,
} from "./inkface.mjs";

function scratchProfile(t) {
    const dir = mkdtempSync(join(tmpdir(), "grmob-inkface-test-"));
    t.after(() => rmSync(dir, { recursive: true, force: true }));
    return dir;
}

function readPrefs(profile) {
    return JSON.parse(readFileSync(join(profile, "Default", "Preferences"), "utf8"));
}

test("the families are standard and serif, under the Common and Latin scripts", () => {
    // Pinned rather than derived: both are facts about Chrome, and a change to
    // either list is a change someone should have to make on purpose.
    assert.deepEqual(INK_FACE_FAMILIES, ["standard", "serif"]);
    assert.equal(INK_FACE_SCRIPT, "Zyyy");
    // Common first: it is the key the calibration table names.
    assert.deepEqual(INK_FACE_SCRIPTS, ["Zyyy", "Latn"]);
});

test("no face writes nothing and creates no directory", (t) => {
    const profile = scratchProfile(t);
    assert.equal(writeInkFaceOverride(profile, undefined), null);
    assert.equal(writeInkFaceOverride(profile, ""), null);
    assert.equal(existsSync(join(profile, "Default")), false);
});

test("a fresh profile gets Default/Preferences with the face on both families", (t) => {
    const profile = scratchProfile(t);
    assert.equal(writeInkFaceOverride(profile, "Georgia"), "Georgia");
    assert.deepEqual(readPrefs(profile), {
        webkit: { webprefs: { fonts: {
            standard: { Zyyy: "Georgia", Latn: "Georgia" },
            serif: { Zyyy: "Georgia", Latn: "Georgia" },
        } } },
    });
});

test("a Latn face already in the profile is overwritten, since Chrome reads it before Zyyy", (t) => {
    // The shadowing case inkface.mjs measured: with Latn left at Didot, Latin
    // text drew in Didot whatever Zyyy said. So an existing Latn is the one
    // sibling the merge must not keep.
    const profile = scratchProfile(t);
    mkdirSync(join(profile, "Default"));
    writeFileSync(join(profile, "Default", "Preferences"), JSON.stringify({
        webkit: { webprefs: { fonts: {
            standard: { Latn: "Didot" },
            serif: { Latn: "Didot", Cyrl: "Times" },
        } } },
    }));

    writeInkFaceOverride(profile, "Papyrus");

    assert.deepEqual(readPrefs(profile).webkit.webprefs.fonts, {
        standard: { Latn: "Papyrus", Zyyy: "Papyrus" },
        // Other scripts' faces are still carried over.
        serif: { Latn: "Papyrus", Cyrl: "Times", Zyyy: "Papyrus" },
    });
});

test("an existing Preferences file keeps everything the face does not replace", (t) => {
    const profile = scratchProfile(t);
    mkdirSync(join(profile, "Default"));
    writeFileSync(join(profile, "Default", "Preferences"), JSON.stringify({
        browser: { has_seen_welcome_page: true },
        webkit: {
            other_pref: 1,
            webprefs: {
                default_font_size: 16,
                fonts: {
                    standard: { Zyyy: "Times", Hans: "PingFang SC" },
                    sansserif: { Zyyy: "Helvetica" },
                },
            },
        },
    }));

    writeInkFaceOverride(profile, "Didot");

    assert.deepEqual(readPrefs(profile), {
        browser: { has_seen_welcome_page: true },
        webkit: {
            other_pref: 1,
            webprefs: {
                default_font_size: 16,
                fonts: {
                    // Replaced for the scripts the grid draws in, kept for the rest.
                    standard: { Zyyy: "Didot", Hans: "PingFang SC", Latn: "Didot" },
                    // A family this write does not own is left alone.
                    sansserif: { Zyyy: "Helvetica" },
                    serif: { Zyyy: "Didot", Latn: "Didot" },
                },
            },
        },
    });
});

test("a Preferences file that is not JSON is replaced, not a crash", (t) => {
    const profile = scratchProfile(t);
    mkdirSync(join(profile, "Default"));
    writeFileSync(join(profile, "Default", "Preferences"), "{not json");
    writeInkFaceOverride(profile, "Charter");
    assert.equal(readPrefs(profile).webkit.webprefs.fonts.serif.Zyyy, "Charter");
});

test("inkFacePreferences does not modify its argument", () => {
    const existing = { webkit: { webprefs: { fonts: { serif: { Zyyy: "Times" } } } } };
    const before = structuredClone(existing);
    const after = inkFacePreferences(existing, "Bodoni 72");
    assert.deepEqual(existing, before);
    assert.equal(after.webkit.webprefs.fonts.serif.Zyyy, "Bodoni 72");
});

test("a non-object at a step of the path is replaced like a missing key", () => {
    const got = inkFacePreferences({ webkit: { webprefs: { fonts: null } } }, "Baskerville");
    assert.deepEqual(got.webkit.webprefs.fonts, {
        standard: { Zyyy: "Baskerville", Latn: "Baskerville" },
        serif: { Zyyy: "Baskerville", Latn: "Baskerville" },
    });
});
