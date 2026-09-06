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

const nested = (role, level) => ({
    Type: "Box",
    Style: { AccessibilityRole: role, AccessibilityNestingLevel: level },
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

// --------------------------------------------------------------------------
// The nesting level: aria-level's other two roles
// --------------------------------------------------------------------------
//
// ARIA defines aria-level for heading, listitem and row. core carries a
// heading's tier and a collection item's depth as two fields with two ranges,
// and ariaLevel is where they meet at the one attribute both become. The
// static exporter is tested on the ranges; what matters here is the live half
// — that the attribute still comes and goes with the field, and that the two
// fields cannot both reach the slot.

test("a nested item's depth becomes aria-level", () => {
    const { at } = mount([nested("listitem", 2), nested("row", 3)]);
    assert.equal(at(0).getAttribute("aria-level"), "2");
    assert.equal(at(1).getAttribute("aria-level"), "3");
});

test("a nesting depth has no ceiling", () => {
    // 7 is what the heading arm drops. Here it is a perfectly ordinary depth,
    // which is the whole reason the two levels are separate fields rather than
    // one with a single range rule.
    const { at } = mount([nested("listitem", 7)]);
    assert.equal(at(0).getAttribute("aria-level"), "7");
});

test("a depth on a role aria-level does not serve is dropped", () => {
    // A list is not a listitem and a cell is not a row: the nearest misses,
    // and the ones a caller reaching for the prop would most plausibly land on.
    const { at } = mount([nested("list", 2), nested("cell", 2), nested("", 2)]);
    for (const i of [0, 1, 2]) {
        assert.equal(at(i).getAttribute("aria-level"), null);
    }
});

test("the role decides which level is read", () => {
    // Both fields set, one role. The switch in ariaLevel is what makes this
    // structural rather than a precedence rule somebody has to remember.
    const both = (role) => ({
        Type: "Box",
        Style: {
            AccessibilityRole: role,
            AccessibilityHeadingLevel: 2,
            AccessibilityNestingLevel: 5,
        },
    });
    const { at } = mount([both("heading"), both("listitem")]);
    assert.equal(at(0).getAttribute("aria-level"), "2");
    assert.equal(at(1).getAttribute("aria-level"), "5");
});

test("a depth that goes away takes its attribute with it", () => {
    // The totality rule again, on the second field to reach this attribute. A
    // guarded write would leave the old depth standing and the tree would
    // report a shape it no longer has.
    const { rt, at } = mount([nested("listitem", 2)]);
    assert.equal(at(0).getAttribute("aria-level"), "2");

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-style",
        TargetID: "root/0",
        Changes: { AccessibilityRole: "listitem" },
    }]));
    rt.drainFrames();

    assert.equal(at(0).getAttribute("aria-level"), null);
    assert.equal(at(0).getAttribute("role"), "listitem");
});

test("a role change re-reads the level from the other field", () => {
    // The sharpest case the switch has to survive: an item that keeps both
    // levels and changes only its role must swap which one is written, not
    // keep the value the previous role selected.
    const { rt, at } = mount([{
        Type: "Box",
        Style: {
            AccessibilityRole: "heading",
            AccessibilityHeadingLevel: 2,
            AccessibilityNestingLevel: 5,
        },
    }]);
    assert.equal(at(0).getAttribute("aria-level"), "2");

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-style",
        TargetID: "root/0",
        Changes: {
            AccessibilityRole: "listitem",
            AccessibilityHeadingLevel: 2,
            AccessibilityNestingLevel: 5,
        },
    }]));
    rt.drainFrames();

    assert.equal(at(0).getAttribute("aria-level"), "5");
});

test("aria-hidden beats a nesting depth too", () => {
    const { at } = mount([{
        Type: "Box",
        Style: {
            AccessibilityHidden: true,
            AccessibilityRole: "listitem",
            AccessibilityNestingLevel: 2,
        },
    }]);
    assert.equal(at(0).getAttribute("aria-level"), null);
    assert.equal(at(0).getAttribute("role"), null);
});

// --------------------------------------------------------------------------
// The selected state: one field, two attributes
// --------------------------------------------------------------------------

const control = (role, selected) => ({
    Type: "Box",
    Style: { AccessibilityRole: role, AccessibilitySelected: selected },
});

test("the role decides which selection attribute is written", () => {
    // ariaLevel's mirror. That switch resolves two Go fields onto one
    // attribute; this one resolves one Go field onto two, because ARIA has two
    // words and they are not synonyms — selected is one of a set, pressed is a
    // toggle answering only for itself.
    const { at } = mount([
        control("tab", "true"),
        control("row", "false"),
        control("columnheader", "true"),
        control("button", "true"),
    ]);

    assert.equal(at(0).getAttribute("aria-selected"), "true");
    assert.equal(at(0).getAttribute("aria-pressed"), null);
    assert.equal(at(1).getAttribute("aria-selected"), "false");
    assert.equal(at(2).getAttribute("aria-selected"), "true");
    assert.equal(at(3).getAttribute("aria-pressed"), "true");
    assert.equal(at(3).getAttribute("aria-selected"), null);
});

test("a state on a role that cannot carry one is dropped", () => {
    // ARIA's scoping, not the framework's. aria-selected is defined for
    // gridcell, option, row, tab, columnheader and rowheader; a listitem is
    // not an option and a cell is not a gridcell, so both write nothing. A
    // reader drops invalid ARIA, so writing it anyway would change nothing a
    // user hears and would put a lie in the document.
    const { at } = mount([
        control("listitem", "true"),
        control("cell", "true"),
        { Type: "Box", Style: { AccessibilitySelected: "true" } },
    ]);

    for (let i = 0; i < 3; i++) {
        assert.equal(at(i).getAttribute("aria-selected"), null);
        assert.equal(at(i).getAttribute("aria-pressed"), null);
    }
});

test("a Button node carries a pressed state with no role of its own", () => {
    // The node type is the role — the same rule that gives a Modal its dialog
    // role. components.Chip renders as a core.Button and sets no role, so
    // without this the widget that most wants aria-pressed is the one node
    // that could not have it.
    const { at } = mount([
        { Type: "Button", Props: { label: "Active" }, Style: { AccessibilitySelected: "true" } },
    ]);
    assert.equal(at(0).getAttribute("aria-pressed"), "true");
});

test("a state that goes away takes its attribute with it", () => {
    // The totality rule, on the third field to reach this family.
    const { rt, at } = mount([control("tab", "true")]);
    assert.equal(at(0).getAttribute("aria-selected"), "true");

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-style",
        TargetID: "root/0",
        Changes: { AccessibilityRole: "tab" },
    }]));
    rt.drainFrames();

    assert.equal(at(0).getAttribute("aria-selected"), null);
    assert.equal(at(0).getAttribute("role"), "tab");
});

test("a role change swaps which attribute holds the state", () => {
    // The sharpest case the switch has to survive, and the reason both
    // attributes are written on every call rather than only the one the role
    // asks for: a node that had been a tab and becomes a button must stop
    // being aria-selected, or it is announced as a selected tab and a pressed
    // button at once.
    const { rt, at } = mount([control("tab", "true")]);
    assert.equal(at(0).getAttribute("aria-selected"), "true");

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-style",
        TargetID: "root/0",
        Changes: { AccessibilityRole: "button", AccessibilitySelected: "true" },
    }]));
    rt.drainFrames();

    assert.equal(at(0).getAttribute("aria-selected"), null);
    assert.equal(at(0).getAttribute("aria-pressed"), "true");
});

test("aria-hidden beats a selected state too", () => {
    const { at } = mount([{
        Type: "Box",
        Style: {
            AccessibilityHidden: true,
            AccessibilityRole: "tab",
            AccessibilitySelected: "true",
        },
    }]);
    assert.equal(at(0).getAttribute("aria-selected"), null);
    assert.equal(at(0).getAttribute("role"), null);
});
