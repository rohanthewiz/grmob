// The accessibility attributes the runtime writes, against the minimal DOM.
//
// htmlout writes the same set into a static document and is tested there; this
// is the live half, where the interesting property is not "the attribute
// appears" but "it appears and disappears on the right patches". The runtime's
// applyAccessibility is deliberately *total* — every attribute is set or
// removed on every call — because an update-style patch carries the whole new
// Style, so a field back at its zero value means "unset now" and a guarded
// write would leave the old attribute standing. That is the rule these tests
// exist to hold.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

// One tree, one element under test, plus the handle to patch its style.
function mount(children) {
    const rt = loadRuntime();
    rt.GrMob.mount(
        JSON.stringify({ Type: "Column", Children: children })
    );
    rt.drainFrames();
    return { rt, at: (i) => nodeAt(rt.document, `root/${i}`) };
}

const heading = (level) => ({
    Type: "Box",
    Style: { AccessibilityRole: "heading", AccessibilityHeadingLevel: level },
});

// --------------------------------------------------------------------------
// aria-level
// --------------------------------------------------------------------------

test("a heading's level becomes aria-level", () => {
    const { at } = mount([heading(1), heading(2), heading(6)]);

    assert.equal(at(0).getAttribute("aria-level"), "1");
    assert.equal(at(1).getAttribute("aria-level"), "2");
    assert.equal(at(2).getAttribute("aria-level"), "6");
});

test("a level is dropped when it has no heading to belong to", () => {
    // ARIA's own scoping: aria-level is defined for heading, listitem and row.
    // A columnheader takes the header role and no tier, which is why
    // DataTable's column headers carry one and not the other.
    const { at } = mount([
        { Type: "Box", Style: { AccessibilityHeadingLevel: 2 } },
        { Type: "Box", Style: { AccessibilityRole: "columnheader", AccessibilityHeadingLevel: 2 } },
    ]);

    assert.equal(at(0).getAttribute("aria-level"), null);
    assert.equal(at(1).getAttribute("aria-level"), null);
    // The role itself is unaffected — only the tier is refused.
    assert.equal(at(1).getAttribute("role"), "columnheader");
});

test("an out-of-range level is dropped, not clamped", () => {
    // Rewriting a 7 into a 6 would export a structure the app never described.
    const { at } = mount([heading(0), heading(7), heading(-1)]);

    for (const i of [0, 1, 2]) {
        assert.equal(at(i).getAttribute("aria-level"), null);
        // Still a heading. Losing the tier must not lose the role with it.
        assert.equal(at(i).getAttribute("role"), "heading");
    }
});

test("aria-hidden wins over the level, as it does over the role", () => {
    const { at } = mount([{
        Type: "Box",
        Style: {
            AccessibilityHidden: true,
            AccessibilityRole: "heading",
            AccessibilityHeadingLevel: 2,
        },
    }]);

    assert.equal(at(0).getAttribute("aria-hidden"), "true");
    assert.equal(at(0).getAttribute("aria-level"), null);
    assert.equal(at(0).getAttribute("role"), null);
});

test("a level that goes away takes its attribute with it", () => {
    // The totality rule. A patch carries the whole new Style, so a heading
    // that stops stating its tier must stop carrying aria-level — a guarded
    // write would leave the old one standing and the outline would be a lie.
    const { rt, at } = mount([heading(2)]);
    assert.equal(at(0).getAttribute("aria-level"), "2");

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-style",
        TargetID: "root/0",
        Changes: { AccessibilityRole: "heading" },
    }]));
    rt.drainFrames();

    assert.equal(at(0).getAttribute("aria-level"), null);
    assert.equal(at(0).getAttribute("role"), "heading");
});

// --------------------------------------------------------------------------
// Modal: the role a node type carries for itself
// --------------------------------------------------------------------------

test("a Modal announces as a dialog with no Style at all", () => {
    // core.ModalNode has no Style field, so the applyStyle path never runs for
    // a dialog core built. The chassis in createElement is what covers it.
    const { at } = mount([{ Type: "Modal", Props: { visible: true } }]);

    assert.equal(at(0).getAttribute("role"), "dialog");
    assert.equal(at(0).getAttribute("aria-modal"), "true");
});

test("a Modal's dialog semantics survive an update-style patch", () => {
    // The other route into applyAccessibility. A hand-assembled Modal node
    // that does carry a Style takes the applyStyle path, whose totality would
    // strip an attribute createElement had set if the two disagreed.
    const { rt, at } = mount([{
        Type: "Modal",
        Props: { visible: true },
        Style: { Background: "#fff" },
    }]);
    assert.equal(at(0).getAttribute("role"), "dialog");

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-style",
        TargetID: "root/0",
        Changes: { Background: "#000" },
    }]));
    rt.drainFrames();

    assert.equal(at(0).getAttribute("role"), "dialog");
    assert.equal(at(0).getAttribute("aria-modal"), "true");
});

test("an authored role beats the Modal default", () => {
    // Same precedent the chassis sets for style: the framework's default goes
    // first and the node's own word outranks it. aria-modal is not expressible
    // through core.Role, so it is not the author's to replace.
    const { at } = mount([{
        Type: "Modal",
        Props: { visible: true },
        Style: { AccessibilityRole: "alert" },
    }]);

    assert.equal(at(0).getAttribute("role"), "alert");
    assert.equal(at(0).getAttribute("aria-modal"), "true");
});

test("aria-hidden wins over the Modal semantics", () => {
    const { at } = mount([{
        Type: "Modal",
        Props: { visible: true },
        Style: { AccessibilityHidden: true },
    }]);

    assert.equal(at(0).getAttribute("aria-hidden"), "true");
    assert.equal(at(0).getAttribute("role"), null);
    assert.equal(at(0).getAttribute("aria-modal"), null);
});

test("nothing else gets a dialog role", () => {
    // The role is the node type's, not a default for containers.
    const { at } = mount([{ Type: "Box" }, { Type: "Card" }]);

    assert.equal(at(0).getAttribute("role"), null);
    assert.equal(at(1).getAttribute("aria-modal"), null);
});
