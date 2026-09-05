// The user-agent border reset, against the minimal DOM.
//
// htmlout writes the same declaration into a static document and is tested
// there; this is the live half, where the interesting property is the one a
// static export cannot have — that a button which *loses* its border falls
// back to "none" rather than to the browser's own rule.
//
// styleFromGrMob is total: every property it manages is assigned on every
// call, because an update-style patch carries the whole new Style and a field
// back at its zero value means "unset now". For every other property "unset"
// is the empty string, which drops the inline declaration and lets the
// cascade decide. For a <button>'s border, letting the cascade decide is the
// bug — the user-agent stylesheet is what draws the 2px outset rule
// components.Button's EmphasisGhost was documented as not having. So this one
// property has three values rather than two, and the third is what these
// tests hold.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

// One tree, one element under test, plus the handle to patch its style. Same
// shape as a11y_test.mjs's.
function mount(children) {
    const rt = loadRuntime();
    rt.GrMob.mount(
        JSON.stringify({ Type: "Column", Children: children })
    );
    rt.drainFrames();
    return { rt, at: (i) => nodeAt(rt.document, `root/${i}`) };
}

const button = (style) => ({ Type: "Button", Props: { label: "Skip" }, Style: style });

test("a button with no border is talked out of the browser's", () => {
    // The EmphasisGhost case: transparent fill, colored label, nothing said
    // about a border. Both natives draw none; without this the web drew the
    // user agent's, and no core.BorderWidth(0) could remove it because
    // emitting nothing is exactly what leaves the browser in charge.
    const { at } = mount([button({ Background: "#00000000", TextColor: "#007AFF" })]);
    assert.equal(at(0).style.border, "none");
});

test("a button with a border draws the one it was given", () => {
    // The EmphasisOutlined case. The reset must not swallow a real border.
    const { at } = mount([button({ BorderWidth: 1, BorderColor: "#007AFF" })]);
    assert.equal(at(0).style.border, "1px solid #007AFF");
});

test("half a border is no border, and still resets", () => {
    // Both halves are required on all four targets — Compose skips its
    // Modifier.border unless width and color are both set, and SwiftUI's
    // grMobBorder guards the same way — so a width with no color must reach
    // the reset rather than emitting an incomplete declaration.
    const { at } = mount([
        button({ BorderWidth: 1 }),
        button({ BorderColor: "#007AFF" }),
    ]);
    assert.equal(at(0).style.border, "none");
    assert.equal(at(1).style.border, "none");
});

test("a div with no border says nothing about one", () => {
    // The reset is scoped to the tags the browser draws a border on. A <div>
    // has none, so "none" there would be a declaration that means nothing and
    // that an author's own stylesheet would then have to fight.
    const { at } = mount([{ Type: "Box", Style: { Background: "#fff" } }]);
    assert.equal(at(0).style.border, "");
});

test("a button that loses its border does not get the browser's back", () => {
    // The totality rule, and the reason the reset lives in the false arm of
    // the same expression rather than in a guard of its own. A guarded write
    // would leave the old declaration standing; an unguarded "" would hand the
    // element back to the user-agent stylesheet. Neither is what an outlined
    // button turning ghost is asking for.
    const { rt, at } = mount([button({ BorderWidth: 1, BorderColor: "#007AFF" })]);
    assert.equal(at(0).style.border, "1px solid #007AFF");

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-style",
        TargetID: "root/0",
        Changes: { Background: "#00000000" },
    }]));
    rt.drainFrames();

    assert.equal(at(0).style.border, "none");
});
