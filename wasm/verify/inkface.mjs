// GRMOB_INK_FACE's profile write, apart from the Chrome it is written for.
//
// browser.mjs reads the variable and hands the face here before Chrome starts.
// The write lives in its own module for the reason startup.mjs and fold.mjs do:
// inkface_test.mjs can reach every arm of it with a temporary directory, where
// inline in browser.mjs the only test was a Chrome launch whose grid happened
// to come out in the face asked for — and a face that silently did not take
// looks exactly like a machine whose default serif is Times.
//
// # Why this exists
//
// INK_CALIBRATED_ON's skip, and the calibration report printed under it, were
// only reachable on a machine whose DEFAULT serif is a face nothing here was
// measured on — and every machine this project met resolved Times or
// Liberation Serif. So the report's "WHICH DOES NOT SEPARATE" arm had fired
// once, under fractions since retired, and a third face was a thing to wait
// for. Any installed face can be the third one on demand this way, with the
// same Chrome, flags and grid, so the skip and the report are drills rather
// than accidents.
//
// # Why the profile, and why both families
//
// Chrome has no command-line flag for a default font; the preference is the
// only lever, and the profile is already a fresh directory per run, so nothing
// outside it is touched. The grid asks for core.Theme's typography, which this
// Chrome does not have, and a family list with no generic keyword falls back to
// the STANDARD font, not the serif one — so both are set, or the override would
// hold only for text that happens to say `serif`.
//
// # Why "Zyyy"
//
// Chrome keys each generic family's face by ISO 15924 script, and Zyyy
// ("Common") is the key its own default font preferences use for the fallback
// entry, which is what Latin text — the only script the grid draws — reads.
// It is also the key 94583a3's eight-face table was measured through, so the
// test pins it: the key is part of the fact the table records, not a detail.
//
// # Why a merge and not a fresh file
//
// Today the profile is always a new mkdtemp directory and there is nothing to
// merge with. A profile that is reused, or a second preference written before
// this one, would otherwise lose everything that is not a font — silently, since
// Chrome recreates a missing preference with its default. Merging costs a read
// and keeps this write a change of two keys rather than an ownership claim over
// the whole file. A Preferences file that is not JSON is replaced rather than
// merged: Chrome itself treats an unreadable file as absent, and a verify run
// is not the place to stop over a scratch profile.
//
// Unset (an empty or missing face), nothing is written and the run is the one
// CI takes.

import { mkdirSync, writeFileSync, readFileSync, existsSync } from "node:fs";
import { join } from "node:path";

// The generic families the face is set for, and the script key it is set
// under. Exported so the test asserts against the list rather than a copy.
export const INK_FACE_FAMILIES = ["standard", "serif"];
export const INK_FACE_SCRIPT = "Zyyy";

// inkFacePreferences returns `existing` with the face set for every family in
// INK_FACE_FAMILIES. Pure: the argument is not modified, so a caller holding
// the parsed file can compare before and after.
//
// Only the path webkit.webprefs.fonts.<family>.<script> is created; every
// sibling at every level along it is carried over, including other scripts'
// faces within a family this write touches.
export function inkFacePreferences(existing, face) {
    const prefs = structuredClone(isObject(existing) ? existing : {});
    const webkit = (prefs.webkit = isObject(prefs.webkit) ? prefs.webkit : {});
    const webprefs = (webkit.webprefs = isObject(webkit.webprefs) ? webkit.webprefs : {});
    const fonts = (webprefs.fonts = isObject(webprefs.fonts) ? webprefs.fonts : {});
    for (const family of INK_FACE_FAMILIES) {
        fonts[family] = { ...(isObject(fonts[family]) ? fonts[family] : {}), [INK_FACE_SCRIPT]: face };
    }
    return prefs;
}

// writeInkFaceOverride writes `face` into <profile>/Default/Preferences and
// returns it, or returns null and touches nothing when there is no face.
export function writeInkFaceOverride(profile, face) {
    if (!face) return null;
    const dir = join(profile, "Default");
    const file = join(dir, "Preferences");
    mkdirSync(dir, { recursive: true });
    let existing = {};
    if (existsSync(file)) {
        try {
            existing = JSON.parse(readFileSync(file, "utf8"));
        } catch {
            existing = {};
        }
    }
    writeFileSync(file, JSON.stringify(inkFacePreferences(existing, face)));
    return face;
}

// Plain objects only: an array or null at a step of the path is not something
// a font map can be merged into, and is replaced like a missing key.
function isObject(v) {
    return v !== null && typeof v === "object" && !Array.isArray(v);
}
