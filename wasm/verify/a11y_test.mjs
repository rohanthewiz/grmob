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
        control("option", "false"),
    ]);

    assert.equal(at(0).getAttribute("aria-selected"), "true");
    assert.equal(at(0).getAttribute("aria-pressed"), null);
    assert.equal(at(1).getAttribute("aria-selected"), "false");
    assert.equal(at(2).getAttribute("aria-selected"), "true");
    assert.equal(at(3).getAttribute("aria-pressed"), "true");
    assert.equal(at(3).getAttribute("aria-selected"), null);
    // The off case on an option, which is the one a listbox needs every row to
    // answer: a listbox where only the chosen row says anything announces the
    // rest as unselectable furniture. Same argument core.SelectedOff carries.
    assert.equal(at(4).getAttribute("aria-selected"), "false");
    assert.equal(at(4).getAttribute("aria-pressed"), null);
});

test("a state on a role that cannot carry one is dropped", () => {
    // ARIA's scoping, not the framework's. aria-selected is defined for
    // gridcell, option, row, tab, columnheader and rowheader; a cell is not a
    // gridcell and a listitem is not an option, so both write nothing. A
    // reader drops invalid ARIA, so writing it anyway would change nothing a
    // user hears and would put a lie in the document.
    //
    // The listitem case is the sharp one now that `option` is in the
    // vocabulary and does write the attribute: the two roles describe the same
    // visual row, and only one of them is a control.
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

// --------------------------------------------------------------------------
// The expanded state: one field, one attribute, a third role list
// --------------------------------------------------------------------------

const disclosure = (role, expanded) => ({
    Type: "Box",
    Style: { AccessibilityRole: role, AccessibilityExpanded: expanded },
});

test("aria-expanded is scoped to its own roles, which are not the selection's", () => {
    // The third state field and the third role list. ARIA defines
    // aria-expanded for button, link, listbox, row, columnheader and tab among
    // the roles this framework carries — which drops option and adds link and
    // listbox relative to the selection above. The two guards look
    // interchangeable and disagree at four roles, which is why they are two
    // switches and why core.ExpandedState is a type of its own.
    const { at } = mount([
        disclosure("button", "false"),
        disclosure("link", "true"),
        disclosure("listbox", "true"),
        disclosure("row", "false"),
        disclosure("columnheader", "true"),
        disclosure("tab", "true"),
        // The divergence, from both sides.
        disclosure("option", "true"),
        disclosure("listitem", "true"),
        { Type: "Box", Style: { AccessibilityExpanded: "true" } },
    ]);

    assert.equal(at(0).getAttribute("aria-expanded"), "false");
    assert.equal(at(1).getAttribute("aria-expanded"), "true");
    assert.equal(at(2).getAttribute("aria-expanded"), "true");
    assert.equal(at(3).getAttribute("aria-expanded"), "false");
    assert.equal(at(4).getAttribute("aria-expanded"), "true");
    assert.equal(at(5).getAttribute("aria-expanded"), "true");
    // An option takes aria-selected and not this: it is a leaf choice, and the
    // thing that expands is the listbox around it.
    assert.equal(at(6).getAttribute("aria-expanded"), null);
    assert.equal(at(7).getAttribute("aria-expanded"), null);
    // And an unroled Box, which is `generic` — the same drop a name would get
    // if the runtime did not supply `group`. There is no equivalent rescue
    // here, because `group` is not one of the six roles above.
    assert.equal(at(8).getAttribute("aria-expanded"), null);
});

test("a Button node carries an expanded state with no role of its own", () => {
    // ARIA's disclosure pattern *is* a button, so without this arm the
    // attribute would be defined for exactly the node type that could not have
    // it. components.Accordion's header row states the role explicitly; a
    // hand-built disclosure out of core.Button does not have to.
    const { at } = mount([
        { Type: "Button", Props: { label: "What is a hook" }, Style: { AccessibilityExpanded: "false" } },
    ]);
    assert.equal(at(0).getAttribute("aria-expanded"), "false");
});

test("a disclosure that opens and shuts moves the attribute both ways", () => {
    // The totality rule on the fourth field to reach this family, and the case
    // an accordion actually produces: the state is patched, not the tree, so
    // the attribute has to be rewritten rather than added once.
    const { rt, at } = mount([disclosure("button", "false")]);
    assert.equal(at(0).getAttribute("aria-expanded"), "false");

    const restyle = (changes) => {
        rt.GrMob.patch(JSON.stringify([{
            Type: "update-style",
            TargetID: "root/0",
            Changes: changes,
        }]));
        rt.drainFrames();
    };

    restyle({ AccessibilityRole: "button", AccessibilityExpanded: "true" });
    assert.equal(at(0).getAttribute("aria-expanded"), "true");

    // And back to unstated, which is a node that has stopped being a
    // disclosure at all. A guarded write would leave "true" standing and
    // announce a section that is no longer there as open.
    restyle({ AccessibilityRole: "button" });
    assert.equal(at(0).getAttribute("aria-expanded"), null);
    assert.equal(at(0).getAttribute("role"), "button");
});

test("a selection and a disclosure coexist on one node", () => {
    // Two independent facts about one control — the shape that makes these two
    // Go fields rather than one. Nothing in the framework builds it, which is
    // why it is pinned: the two guards are separate switches over the same
    // role, and a merge of them would still pass every other test here.
    const { at } = mount([{
        Type: "Box",
        Style: {
            AccessibilityRole: "tab",
            AccessibilitySelected: "true",
            AccessibilityExpanded: "true",
        },
    }]);
    assert.equal(at(0).getAttribute("aria-selected"), "true");
    assert.equal(at(0).getAttribute("aria-expanded"), "true");
});

test("aria-hidden beats an expanded state too", () => {
    const { at } = mount([{
        Type: "Box",
        Style: {
            AccessibilityHidden: true,
            AccessibilityRole: "button",
            AccessibilityExpanded: "true",
        },
    }]);
    assert.equal(at(0).getAttribute("aria-expanded"), null);
    assert.equal(at(0).getAttribute("role"), null);
});

// --------------------------------------------------------------------------
// The supplied group role, and the two IDREFs
// --------------------------------------------------------------------------

test("a named container is given the group role", () => {
    // ARIA prohibits an accessible name on `generic`, which is the implicit
    // role of every <div> and <span> this runtime creates, and browsers
    // enforce that by dropping the name. `group` is the smallest role that
    // makes it legal. See ariaRole in the runtime and core.RoleGroup.
    const { at } = mount([
        { Type: "Box", Style: { AccessibilityLabel: "Unread messages" } },
        { Type: "Text", Props: { text: "*" }, Style: { AccessibilityLabel: "required" } },
    ]);

    assert.equal(at(0).getAttribute("role"), "group");
    assert.equal(at(0).getAttribute("aria-label"), "Unread messages");
    assert.equal(at(1).getAttribute("role"), "group");
});

test("the group role is withheld where it would take something away", () => {
    const { at } = mount([
        // A <button> can carry a name already; a role here would replace the
        // one the browser gives it.
        { Type: "Button", Props: { label: "x" }, Style: { AccessibilityLabel: "Close" } },
        // An author who said what the node is keeps their word.
        { Type: "Box", Style: { AccessibilityRole: "img", AccessibilityLabel: "Compass" } },
        // Nothing to rescue.
        { Type: "Box", Style: { BorderRadius: 4 } },
        // aria-hidden still wins alone.
        { Type: "Box", Style: { AccessibilityHidden: true, AccessibilityLabel: "Close" } },
    ]);

    assert.equal(at(0).getAttribute("role"), null);
    assert.equal(at(1).getAttribute("role"), "img");
    assert.equal(at(2).getAttribute("role"), null);
    assert.equal(at(3).getAttribute("role"), null);
});

test("a named Modal stays a dialog", () => {
    // The dialog case is answered before ariaRole is consulted, and a dialog
    // is nameable, so there is nothing for the fallback to rescue.
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "Column",
        Children: [{ Type: "Modal", Style: { AccessibilityLabel: "Confirm" } }],
    }));
    rt.drainFrames();
    const el = nodeAt(rt.document, "root/0");
    assert.equal(el.getAttribute("role"), "dialog");
});

test("the supplied role goes away with the name that earned it", () => {
    // The totality rule: an update-style carries the whole new Style, so a
    // name back at its zero value means the role it unlocked has to go too.
    const { rt, at } = mount([{ Type: "Box", Style: { AccessibilityLabel: "Unread" } }]);
    assert.equal(at(0).getAttribute("role"), "group");

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-style",
        TargetID: "root/0",
        Changes: { BorderRadius: 4 },
    }]));
    rt.drainFrames();

    assert.equal(at(0).getAttribute("role"), null);
    assert.equal(at(0).getAttribute("aria-label"), null);
});

test("an id and an aria-controls cross verbatim, in both directions", () => {
    const { rt, at } = mount([{
        Type: "Box",
        Style: {
            AccessibilityRole: "tab",
            AccessibilityID: "home-tab",
            AccessibilityControls: "app-panel",
        },
    }]);

    assert.equal(at(0).getAttribute("id"), "home-tab");
    assert.equal(at(0).getAttribute("aria-controls"), "app-panel");

    // Total, like every other attribute here.
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-style",
        TargetID: "root/0",
        Changes: { AccessibilityRole: "tab" },
    }]));
    rt.drainFrames();

    assert.equal(at(0).getAttribute("id"), null);
    assert.equal(at(0).getAttribute("aria-controls"), null);
});
