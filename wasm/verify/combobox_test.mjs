// The combobox keyboard, in the runtime.
//
// A combobox is the one pattern in the runtime whose keyboard must not move
// focus: the field keeps it, and aria-activedescendant names the option the
// arrows reached. comps.SearchableSelect is the widget. Before it adopted the
// pattern its list was a listbox with its own tab stop, so reaching an option
// meant leaving the caret, and a pick made with Enter removed the focused option
// along with the list and dropped focus onto the page.
//
// These tests use the tree SearchableSelect sends: an Input carrying the
// combobox role, aria-expanded and aria-controls, beside a listbox with that id
// whose options carry slot ids. They check the runtime's bookkeeping against
// dom.mjs: which option is active, which keys are claimed, that the popup
// holds no tab stop, and that the dismiss a keyboard pick causes is declined.
// That focus really stays in the field, and that the active option really
// draws its outline, were checked against the tutorial in a real Chrome; this
// file cannot say either.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

const LIST = "country-list";

// A combobox field over a popup of `count` options, the shape
// comps.SearchableSelect renders while its query has matches.
function combobox({ count = 3, ids = true, expanded = true, submit = false } = {}) {
    const fieldStyle = {
        AccessibilityRole: "combobox",
        AccessibilityExpanded: expanded ? "true" : "false",
        AccessibilityLabel: "Country",
    };
    if (expanded) fieldStyle.AccessibilityControls = LIST;
    const field = {
        Type: "Input",
        Props: { value: "an", placeholder: "Search countries", onChange: "txt_query" },
        Style: fieldStyle,
    };
    // The Next action core.UseFocusOrder stamps onto a field in an order, which
    // is what the tutorial's Country field carries.
    if (submit) {
        field.Props.onSubmit = "cb_submit";
        field.Props.imeAction = "next";
    }
    const options = [];
    for (let i = 0; i < count; i++) {
        const Style = { AccessibilityRole: "option", AccessibilityLabel: `Country ${i}` };
        if (ids) Style.AccessibilityID = `${LIST}-option-${i}`;
        options.push({ Type: "Box", Props: { onClick: `cb_${i}` }, Style });
    }
    const list = {
        Type: "Column",
        Style: { AccessibilityRole: "listbox", AccessibilityID: LIST },
        Children: options,
    };
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: expanded ? [field, list] : [field] }));
    rt.drainFrames();
    const f = nodeAt(rt.document, "root/0");
    f.focus();
    return {
        rt,
        field: f,
        option: (i) => nodeAt(rt.document, `root/1/${i}`),
        key: (key) => f.dispatch("keydown", { key }),
        active: () => f.getAttribute("aria-activedescendant"),
    };
}

// The field's focus command, as core.DismissKeyboard stamps it: a new epoch
// with the blur action, patched onto the field.
function dismiss(rt, epoch) {
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props",
        TargetID: "root/0",
        Changes: { focusEpoch: epoch, focusAction: "blur" },
    }]));
    rt.drainFrames();
}

// --------------------------------------------------------------------------
// The popup stands down
// --------------------------------------------------------------------------

test("a combobox's popup holds no tab stop, where a bare listbox holds one", () => {
    // Tab from the field goes on past the list: the options are reached
    // through the field. The same listbox with nothing naming it is a
    // composite as it always was.
    const cb = combobox();
    for (let i = 0; i < 3; i++) {
        assert.equal(cb.option(i).getAttribute("tabindex"), null, `option ${i}`);
    }

    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "Column",
        Style: { AccessibilityRole: "listbox", AccessibilityID: LIST },
        Children: [0, 1].map((i) => ({
            Type: "Box",
            Props: { onClick: `cb_${i}` },
            Style: { AccessibilityRole: "option" },
        })),
    }));
    rt.drainFrames();
    assert.equal(nodeAt(rt.document, "root/0").getAttribute("tabindex"), "0");
});

test("a row rebuilt by a patch stays out of the tab order", () => {
    // Every keystroke re-filters the rows in a patch that never touches the
    // field, which is why the question is asked from the listbox's side.
    const cb = combobox();
    cb.rt.GrMob.patch(JSON.stringify([{
        Type: "replace",
        TargetID: "root/1/0",
        Changes: {
            Type: "Box",
            Props: { onClick: "cb_new" },
            Style: { AccessibilityRole: "option", AccessibilityID: `${LIST}-option-0` },
        },
    }]));
    cb.rt.drainFrames();
    assert.equal(cb.option(0).getAttribute("tabindex"), null);
    assert.equal(cb.option(1).getAttribute("tabindex"), null);
});

// --------------------------------------------------------------------------
// The arrows
// --------------------------------------------------------------------------

test("ArrowDown names the first option, and focus stays in the field", () => {
    const cb = combobox();
    const e = cb.key("ArrowDown");
    assert.equal(cb.active(), `${LIST}-option-0`);
    assert.equal(cb.option(0).dataset.grmobActiveOption, "true");
    assert.equal(cb.rt.document.activeElement, cb.field);
    // Claimed, or the caret would also jump to the end of the text.
    assert.equal(e.defaultPrevented, true);
    // Moving selects nothing: the pick is Go's, through Enter or a tap.
    assert.equal(cb.rt.dispatched.length, 0);
});

test("the arrows wrap both ways, and ArrowUp from none lands on the last", () => {
    const cb = combobox();
    cb.key("ArrowUp");
    assert.equal(cb.active(), `${LIST}-option-2`);
    cb.key("ArrowDown");
    assert.equal(cb.active(), `${LIST}-option-0`);
    cb.key("ArrowDown");
    cb.key("ArrowDown");
    assert.equal(cb.active(), `${LIST}-option-2`);
    cb.key("ArrowDown");
    assert.equal(cb.active(), `${LIST}-option-0`);
});

test("moving on takes the highlight off the option it left", () => {
    const cb = combobox();
    cb.key("ArrowDown");
    cb.key("ArrowDown");
    assert.equal(cb.option(0).dataset.grmobActiveOption, undefined);
    assert.equal(cb.option(1).dataset.grmobActiveOption, "true");
});

test("an option with no id cannot be the active one", () => {
    // aria-activedescendant is an IDREF; an option it cannot name is one a
    // reader would never hear about, so nothing pretends otherwise.
    const cb = combobox({ ids: false });
    cb.key("ArrowDown");
    assert.equal(cb.active(), null);
});

test("with the popup shut the arrows are the field's", () => {
    const cb = combobox({ expanded: false });
    const e = cb.key("ArrowDown");
    assert.equal(e.defaultPrevented, false);
    assert.equal(cb.active(), null);
});

// --------------------------------------------------------------------------
// Picking, and the focus after it
// --------------------------------------------------------------------------

test("Enter picks the active option through its own onClick, once", () => {
    const cb = combobox();
    cb.key("ArrowDown");
    cb.key("ArrowDown");
    const e = cb.key("Enter");
    assert.equal(e.defaultPrevented, true);
    assert.deepEqual(cb.rt.dispatched.map((d) => d.id), ["cb_1"]);
});

test("Enter with no active option is left to the field", () => {
    const cb = combobox();
    const e = cb.key("Enter");
    assert.equal(e.defaultPrevented, false);
    assert.equal(cb.rt.dispatched.length, 0);
});

test("a keyboard pick keeps focus in the field through the dismiss it causes", () => {
    // The bug the pattern closed: the pick's handler calls
    // core.DismissKeyboard, and on the web that blurred the field a keyboard
    // user had never left.
    const cb = combobox();
    cb.key("ArrowDown");
    cb.key("Enter");
    dismiss(cb.rt, 1);
    assert.equal(cb.rt.document.activeElement, cb.field);

    // Once. The next dismiss is an ordinary one.
    dismiss(cb.rt, 2);
    assert.equal(cb.rt.document.activeElement, null);
});

test("without a keyboard pick a dismiss blurs the field as before", () => {
    // A tap on a phone: the choice is made and the keyboard is in the way.
    const cb = combobox();
    dismiss(cb.rt, 1);
    assert.equal(cb.rt.document.activeElement, null);
});

test("a key after the pick cancels the protection", () => {
    const cb = combobox();
    cb.key("ArrowDown");
    cb.key("Enter");
    cb.key("a");
    dismiss(cb.rt, 1);
    assert.equal(cb.rt.document.activeElement, null);
});

// --------------------------------------------------------------------------
// What clears the active option
// --------------------------------------------------------------------------

test("typing clears the active option, since the list is about to change", () => {
    const cb = combobox();
    cb.key("ArrowDown");
    cb.field.dispatch("input", {});
    assert.equal(cb.active(), null);
    assert.equal(cb.option(0).dataset.grmobActiveOption, undefined);
});

test("Escape clears an active option, and is otherwise the page's", () => {
    const cb = combobox();
    assert.equal(cb.key("Escape").defaultPrevented, false);
    cb.key("ArrowDown");
    assert.equal(cb.key("Escape").defaultPrevented, true);
    assert.equal(cb.active(), null);
});

test("Home and End go back to the text", () => {
    const cb = combobox();
    cb.key("ArrowDown");
    const e = cb.key("Home");
    assert.equal(cb.active(), null);
    // Not claimed: moving the caret is the key's own job.
    assert.equal(e.defaultPrevented, false);
});

test("a popup that closes takes the active option with it", () => {
    // A pick's render shuts the list and patches the field's aria-expanded in
    // the same batch; a stale aria-activedescendant would name an option
    // that is gone.
    const cb = combobox();
    cb.key("ArrowDown");
    cb.rt.GrMob.patch(JSON.stringify([{
        Type: "update-style",
        TargetID: "root/0",
        Changes: {
            AccessibilityRole: "combobox",
            AccessibilityExpanded: "false",
            AccessibilityLabel: "Country",
        },
    }]));
    cb.rt.drainFrames();
    assert.equal(cb.active(), null);
});

test("in a focus order, Enter on a reached option picks it and does not also run Next", () => {
    // The tutorial's shape, and what a real Chrome caught: the field carries
    // the Next action as an onSubmit, both listeners are on keydown, and the
    // first version picked France and then moved focus on to the City field.
    const cb = combobox({ submit: true });
    cb.key("ArrowDown");
    cb.key("Enter");
    assert.deepEqual(cb.rt.dispatched.map((d) => d.id), ["cb_0"]);
});

test("in a focus order, Enter with no option reached is still the form's Next", () => {
    const cb = combobox({ submit: true });
    cb.key("Enter");
    assert.deepEqual(cb.rt.dispatched.map((d) => d.id), ["cb_submit"]);
});
