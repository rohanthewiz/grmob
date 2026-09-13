// The keyboard half of a listbox and a tablist, in the runtime.
//
// core.Role has always been a vocabulary — it says what a node *is* and stops
// there — and for the two structural pairs that are real ARIA *controls* that
// left a gap its own doc names: the container takes keyboard focus, the arrow
// keys move an active member, and a roving tabindex says which one has it.
// None of it existed, and three shipped screens claimed it: an article list
// that is a listbox of divs no keyboard could reach, and two tab strips that a
// keyboard could reach only by tabbing through every member.
//
// Nothing in Go changed to close it. Everything the pattern needs was already
// on the wire — the roles say what contains what, aria-selected says which
// member is chosen, the container's flex-direction says which way the arrows
// go, and the author's onClick says what activation means — so these tests are
// about a target that reads what was already being said.
//
// The natives are deliberately absent from this file. VoiceOver and TalkBack
// navigate a collection by swipe, and neither has a listbox in its semantics
// vocabulary at all; htmlout is absent too, and its absence is asserted in
// Go (wasm/verify/keynav_test.go) rather than here.
//
// # What is checked here and what needs a browser
//
// Everything below runs against dom.mjs, where `tabindex` is a string nobody
// reads, `focus()` is an assignment and `defaultPrevented` is a flag the shim
// set itself. That is the right model for the runtime's own bookkeeping —
// which member the stop is on, which one focus moved to, which keys are
// claimed — and it is the wrong one for the three browser facts the pattern
// rests on: that tabindex="-1" really removes a <button> from the tab order,
// that a disabled control refuses focus, and that preventDefault on ArrowDown
// really stops the scroll. Those are in browser.mjs, against a real Chrome.
//
// The split is worth knowing when a test here fails: this file says the
// runtime made the right decision, that one says the browser honoured it.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

// One member of a composite: a Box carrying the member role, an optional
// selection state, and an onClick — which is what comps.ListRow's
// Selectable rows and examples/social's tab buttons both come across as.
function member(role, { selected, onClick, type = "Box", disabled } = {}) {
    const Style = { AccessibilityRole: role };
    if (selected !== undefined) Style.AccessibilitySelected = selected ? "true" : "false";
    if (disabled) Style.Disabled = true;
    const Props = {};
    if (onClick) Props.onClick = onClick;
    return { Type: type, Style, Props };
}

// A composite container holding n members, the given one selected.
//
// Column for a listbox and Row for a tablist, which is what the two consumers
// actually are and is also what decides the arrow pair — see
// compositeIsVertical.
function composite(role, members, { type } = {}) {
    return {
        Type: type || (role === "listbox" ? "Column" : "Row"),
        Style: { AccessibilityRole: role },
        Children: members,
    };
}

function mountTree(tree) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify(tree));
    rt.drainFrames();
    return {
        rt,
        root: rt.mountPoint.children[0],
        at: (path) => nodeAt(rt.document, path),
        tabindexes: () =>
            rt.mountPoint.children[0].children.map((c) => c.getAttribute("tabindex")),
        focused: () => rt.document.activeElement,
    };
}

// The article list, near enough: five options, the second chosen, each with a
// tap handler.
function listbox({ selected = 1, count = 3 } = {}) {
    const members = [];
    for (let i = 0; i < count; i++) {
        members.push(member("option", { selected: i === selected, onClick: `cb_${i}` }));
    }
    return mountTree(composite("listbox", members));
}

// The bottom bar, near enough: three <button> tabs, the first chosen.
function tablist({ selected = 0, count = 3 } = {}) {
    const members = [];
    for (let i = 0; i < count; i++) {
        members.push(member("tab", {
            selected: i === selected,
            onClick: `cb_${i}`,
            type: "Button",
        }));
    }
    return mountTree(composite("tablist", members));
}

// A toolbar's control: a real <button>, which is what comps.Chip renders
// as and what FOCUSABLE_TAGS is about.
function control(label, { onClick, disabled } = {}) {
    const Style = {};
    if (disabled) Style.Disabled = true;
    const Props = { content: label };
    if (onClick) Props.onClick = onClick;
    return { Type: "Button", Style, Props };
}

// The filter bar, near enough: a Row carrying role="toolbar" over n chips.
//
// comps.ChipStrip is a Row of comps.Chip, each of which renders as a
// core.Button, and the toolbar role is what a caller puts on the strip. So this
// is the shipped shape rather than a shape invented for the test.
function toolbar(children) {
    return mountTree({
        Type: "Row",
        Style: { AccessibilityRole: "toolbar" },
        Children: children,
    });
}

// --------------------------------------------------------------------------
// The roving tabindex
// --------------------------------------------------------------------------

test("a listbox has exactly one tab stop, on the selected option", () => {
    // The whole point of the roving tabindex: a widget is one stop in the
    // page's tab order, and Tab enters it at the thing that is chosen.
    const lb = listbox({ selected: 1 });
    assert.deepEqual(lb.tabindexes(), ["-1", "0", "-1"]);
});

test("a tablist takes its members out of the tab order too", () => {
    // The regression a keyboard user actually felt: three <button> tabs are
    // three tab stops by default, so the bottom bar cost three presses to
    // cross and announced "tab, 1 of 3" while behaving like three unrelated
    // buttons.
    const tl = tablist({ selected: 2 });
    assert.deepEqual(tl.tabindexes(), ["-1", "-1", "0"]);
});

test("a widget with nothing selected is still enterable", () => {
    // SelectedOff on every member is a legal state — a strip sets the state on
    // all of them, not only the live one — and so is a listbox nobody has
    // chosen from yet. Either way there has to be a way in.
    const lb = mountTree(composite("listbox", [
        member("option", { selected: false }),
        member("option", { selected: false }),
    ]));
    assert.deepEqual(lb.tabindexes(), ["0", "-1"]);
});

test("an empty composite is left alone", () => {
    // Nothing to move a stop between, and nothing to write it on.
    const lb = mountTree(composite("listbox", []));
    assert.equal(lb.root.children.length, 0);
    assert.equal(lb.root.getAttribute("tabindex"), null);
});

test("a container that is not a composite writes no tabindex", () => {
    // A `list` is content, not a control: it has no keyboard pattern, and
    // stamping one would take its items out of the page's tab order in
    // exchange for nothing.
    const l = mountTree(composite("list", [
        member("listitem"),
        member("listitem"),
    ], { type: "Column" }));
    assert.deepEqual(l.tabindexes(), [null, null]);
});

// --------------------------------------------------------------------------
// Moving
// --------------------------------------------------------------------------

test("Down and Up move through a listbox and carry the stop with them", () => {
    const lb = listbox({ selected: 0 });
    const items = lb.root.children;
    items[0].focus();

    items[0].dispatch("keydown", { key: "ArrowDown" });
    assert.equal(lb.focused(), items[1]);
    assert.deepEqual(lb.tabindexes(), ["-1", "0", "-1"],
        "focus and the tab stop are two statements of one fact; a focused " +
        "member holding tabindex=-1 sends the next Tab back to the top of the page");

    items[1].dispatch("keydown", { key: "ArrowUp" });
    assert.equal(lb.focused(), items[0]);
    assert.deepEqual(lb.tabindexes(), ["0", "-1", "-1"]);
});

test("a tablist moves on Left and Right, because it is a Row", () => {
    // The arrow pair comes from the container's own resolved flex-direction,
    // which this runtime planted from stackAxisFor. Nobody had to say it
    // twice.
    const tl = tablist();
    const tabs = tl.root.children;
    tabs[0].focus();

    tabs[0].dispatch("keydown", { key: "ArrowRight" });
    assert.equal(tl.focused(), tabs[1]);

    tabs[1].dispatch("keydown", { key: "ArrowLeft" });
    assert.equal(tl.focused(), tabs[0]);
});

test("the arrows of the other axis are left to the page", () => {
    // A widget that swallowed both pairs would stop a vertical page scrolling
    // while a horizontal tab strip held focus.
    const tl = tablist();
    const tabs = tl.root.children;
    tabs[0].focus();

    const e = tabs[0].dispatch("keydown", { key: "ArrowDown" });
    assert.equal(tl.focused(), tabs[0]);
    assert.equal(e.defaultPrevented, false);
});

test("a tablist laid out as a column takes the vertical arrows", () => {
    // A sidebar strip. The author turned it on its side with the layout, and
    // said which way its arrows go by doing so.
    const tl = mountTree({
        Type: "Column",
        Style: { AccessibilityRole: "tablist" },
        Children: [member("tab", { selected: true }), member("tab")],
    });
    const tabs = tl.root.children;
    tabs[0].focus();

    tabs[0].dispatch("keydown", { key: "ArrowDown" });
    assert.equal(tl.focused(), tabs[1]);
});

test("movement wraps at both ends", () => {
    // One rule for both widgets. ARIA makes wrapping optional for a listbox
    // and recommends it for a tablist, and a user who has learned the tab
    // strip should not find the article list behaves differently.
    const lb = listbox({ selected: 0, count: 3 });
    const items = lb.root.children;

    items[0].focus();
    items[0].dispatch("keydown", { key: "ArrowUp" });
    assert.equal(lb.focused(), items[2], "backwards off the front wraps to the end");

    items[2].dispatch("keydown", { key: "ArrowDown" });
    assert.equal(lb.focused(), items[0], "forwards off the end wraps to the front");
});

test("Home and End jump to the ends", () => {
    const lb = listbox({ selected: 1, count: 4 });
    const items = lb.root.children;
    items[1].focus();

    items[1].dispatch("keydown", { key: "End" });
    assert.equal(lb.focused(), items[3]);

    items[3].dispatch("keydown", { key: "Home" });
    assert.equal(lb.focused(), items[0]);
});

test("a movement key is taken from the page", () => {
    // The arrows scroll a page and Home and End jump it to the ends. A widget
    // that moved its own focus while the document scrolled underneath is the
    // same widget twice.
    const lb = listbox();
    const items = lb.root.children;
    items[0].focus();
    for (const key of ["ArrowDown", "ArrowUp", "Home", "End"]) {
        const e = items[0].dispatch("keydown", { key });
        assert.equal(e.defaultPrevented, true, key);
    }
});

test("Tab is never swallowed", () => {
    // A listbox that took Tab would trap a keyboard user inside it, which is
    // the one failure worse than having no pattern at all.
    const lb = listbox();
    const items = lb.root.children;
    const e = items[0].dispatch("keydown", { key: "Tab" });
    assert.equal(e.defaultPrevented, false);
});

// --------------------------------------------------------------------------
// Activation
// --------------------------------------------------------------------------

test("Enter and Space run an option's own handler", () => {
    // The half that makes the pattern worth having for a listbox of divs: the
    // rows were not focusable at all before this, so arrowing to one and
    // having no way to choose it would have been a widget you can look at.
    for (const key of ["Enter", " "]) {
        const lb = listbox({ selected: 0 });
        const items = lb.root.children;
        items[2].focus();
        const e = items[2].dispatch("keydown", { key });

        assert.deepEqual(lb.rt.dispatched, [{ id: "cb_2", payload: {} }], key);
        assert.equal(e.defaultPrevented, true,
            "Space scrolls a page and Enter submits a surrounding form");
    }
});

test("a <button> tab is left to the browser", () => {
    // A <button> fires a real click on both Enter and Space, so synthesizing
    // one here would run the author's handler twice — and a doubled tab change
    // is a screen that switches and switches back.
    const tl = tablist();
    const tabs = tl.root.children;
    tabs[1].focus();
    tabs[1].dispatch("keydown", { key: "Enter" });
    tabs[1].dispatch("keydown", { key: " " });

    assert.deepEqual(tl.rt.dispatched, [],
        "the runtime dispatched a click the browser was already going to send");
});

test("an option with no handler keeps its keys to itself", () => {
    // Nothing to run, so Space must still scroll the page.
    const lb = mountTree(composite("listbox", [member("option"), member("option")]));
    const items = lb.root.children;
    const e = items[0].dispatch("keydown", { key: " " });

    assert.deepEqual(lb.rt.dispatched, []);
    assert.equal(e.defaultPrevented, false);
});

test("activation reads the callback ID back at fire time", () => {
    // The same latest-ID discipline every listener path here follows:
    // callback IDs are minted per render pass, so a handler replaced by a
    // later pass is the one that has to run.
    const lb = listbox({ selected: 0 });
    lb.rt.GrMob.patch(JSON.stringify([{
        Type: "update-props",
        TargetID: "root/1",
        Changes: { onClick: "cb_renamed" },
    }]));
    lb.rt.drainFrames();

    const items = lb.root.children;
    items[1].focus();
    items[1].dispatch("keydown", { key: "Enter" });
    assert.deepEqual(lb.rt.dispatched, [{ id: "cb_renamed", payload: {} }]);
});

// --------------------------------------------------------------------------
// Staying right across patches
// --------------------------------------------------------------------------

test("a patch does not yank the tab stop away from where the user is", () => {
    // The bug the naive version has: sync from aria-selected alone, and a user
    // who has arrowed to the third option without choosing it loses the stop
    // back to the second the moment any unrelated patch lands.
    const lb = listbox({ selected: 1, count: 4 });
    const items = lb.root.children;
    items[1].focus();
    items[1].dispatch("keydown", { key: "ArrowDown" });
    items[2].dispatch("keydown", { key: "ArrowDown" });
    assert.equal(lb.focused(), items[3]);

    lb.rt.GrMob.patch(JSON.stringify([{
        Type: "update-style",
        TargetID: "root/0",
        Changes: { AccessibilityRole: "option", Background: "#eeeeee" },
    }]));
    lb.rt.drainFrames();

    assert.deepEqual(lb.tabindexes(), ["-1", "-1", "-1", "0"],
        "the stop followed the selection instead of the user");
});

test("a selection change moves the way in, once focus has left", () => {
    // The other half of the same rule. With nothing focused inside the widget,
    // the entry point is the chosen member — which is where a click that just
    // changed the selection has moved it.
    const lb = listbox({ selected: 0, count: 3 });
    assert.deepEqual(lb.tabindexes(), ["0", "-1", "-1"]);

    lb.rt.GrMob.patch(JSON.stringify([
        {
            Type: "update-style",
            TargetID: "root/0",
            Changes: { AccessibilityRole: "option", AccessibilitySelected: "false" },
        },
        {
            Type: "update-style",
            TargetID: "root/2",
            Changes: { AccessibilityRole: "option", AccessibilitySelected: "true" },
        },
    ]));
    lb.rt.drainFrames();

    assert.deepEqual(lb.tabindexes(), ["-1", "-1", "0"]);
});

test("a member added by a patch joins the widget", () => {
    // A list that grew a row. The up-walk from the patch's parent is what
    // finds the container: a member cannot see what contains it.
    const lb = listbox({ selected: 0, count: 2 });
    lb.rt.GrMob.patch(JSON.stringify([{
        Type: "add-child",
        TargetID: "root",
        Changes: member("option", { onClick: "cb_new" }),
    }]));
    lb.rt.drainFrames();

    assert.deepEqual(lb.tabindexes(), ["0", "-1", "-1"]);
    const items = lb.root.children;
    items[0].focus();
    items[0].dispatch("keydown", { key: "ArrowUp" });
    assert.equal(lb.focused(), items[2], "the new row is not in the rotation");
});

test("a patch on a buried member still reaches its container", () => {
    // The up-walk, and the case that needs it. A patch reports the element it
    // landed on and that element's parent; when a member sits inside a wrapper
    // — which is what core.For and core.Keyed produce — neither of those is
    // the container, and a walk that only went down from them would never sync
    // the widget at all.
    const lb = mountTree({
        Type: "Column",
        Style: { AccessibilityRole: "listbox" },
        Children: [
            { Type: "Box", Children: [member("option", { selected: true })] },
            { Type: "Box", Children: [member("option")] },
        ],
    });
    const first = nodeAt(lb.rt.document, "root/0/0");
    const second = nodeAt(lb.rt.document, "root/1/0");
    assert.equal(first.getAttribute("tabindex"), "0");

    lb.rt.GrMob.patch(JSON.stringify([
        {
            Type: "update-style",
            TargetID: "root/0/0",
            Changes: { AccessibilityRole: "option", AccessibilitySelected: "false" },
        },
        {
            Type: "update-style",
            TargetID: "root/1/0",
            Changes: { AccessibilityRole: "option", AccessibilitySelected: "true" },
        },
    ]));
    lb.rt.drainFrames();

    assert.equal(first.getAttribute("tabindex"), "-1");
    assert.equal(second.getAttribute("tabindex"), "0");
});

test("a whole composite arriving in a patch is wired", () => {
    // The down-walk. Nothing above the new subtree is a composite, so an
    // up-walk alone would never reach it.
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: [{ Type: "Text" }] }));
    rt.drainFrames();

    rt.GrMob.patch(JSON.stringify([{
        Type: "add-child",
        TargetID: "root",
        Changes: composite("listbox", [
            member("option", { selected: true }),
            member("option"),
        ]),
    }]));
    rt.drainFrames();

    const box = nodeAt(rt.document, "root/1");
    assert.deepEqual(box.children.map((c) => c.getAttribute("tabindex")), ["0", "-1"]);
});

test("a replaced member is wired again", () => {
    // A replace builds a fresh element, which carries neither the stamp nor
    // the listener the old one had — the exact case the stamp exists for.
    const lb = listbox({ selected: 0, count: 2 });
    lb.rt.GrMob.patch(JSON.stringify([{
        Type: "replace",
        TargetID: "root/1",
        Changes: member("option", { onClick: "cb_fresh" }),
    }]));
    lb.rt.drainFrames();

    const items = lb.root.children;
    items[0].focus();
    items[0].dispatch("keydown", { key: "ArrowDown" });
    assert.equal(lb.focused(), items[1]);
    items[1].dispatch("keydown", { key: "Enter" });
    assert.deepEqual(lb.rt.dispatched, [{ id: "cb_fresh", payload: {} }]);
});

test("a member is wired exactly once, however many patches land on it", () => {
    // The stamp's other job. A second listener on the same element would run
    // the handler twice and move focus two places per arrow key.
    const lb = listbox({ selected: 0, count: 3 });
    for (let i = 0; i < 4; i++) {
        lb.rt.GrMob.patch(JSON.stringify([{
            Type: "update-style",
            TargetID: "root/1",
            Changes: { AccessibilityRole: "option" },
        }]));
        lb.rt.drainFrames();
    }

    const items = lb.root.children;
    items[0].focus();
    items[0].dispatch("keydown", { key: "ArrowDown" });
    assert.equal(lb.focused(), items[1], "one arrow key moved more than one place");
});

// --------------------------------------------------------------------------
// What is and is not a member
// --------------------------------------------------------------------------

test("a member found deeper than a direct child still counts", () => {
    // core.For and core.Keyed wrap rows in whatever they wrap them in, and a
    // tab strip may keep its buttons inside a scroller. Nothing on the wire
    // says a member is a direct child.
    const lb = mountTree({
        Type: "Column",
        Style: { AccessibilityRole: "listbox" },
        Children: [
            { Type: "Box", Children: [member("option", { selected: true })] },
            { Type: "Box", Children: [member("option")] },
        ],
    });
    const first = nodeAt(lb.rt.document, "root/0/0");
    const second = nodeAt(lb.rt.document, "root/1/0");

    assert.equal(first.getAttribute("tabindex"), "0");
    assert.equal(second.getAttribute("tabindex"), "-1");

    first.focus();
    first.dispatch("keydown", { key: "ArrowDown" });
    assert.equal(lb.focused(), second);
});

test("a nested composite keeps its own members", () => {
    // Two listboxes, one inside the other. Pooling both sets would let an
    // arrow key in the inner one walk out into the outer one's rows.
    //
    // Not contrived: comps.RadioGroup declares its own listbox, so a
    // RadioGroup placed inside a caller's hand-roled listbox is this tree.
    // RadioGroup and BottomBar are the only widgets that declare a container
    // role, and both are closed (no core.View slot), so two widgets cannot
    // nest on their own; comps/nested_composite_test.go holds that premise.
    const lb = mountTree({
        Type: "Column",
        Style: { AccessibilityRole: "listbox" },
        Children: [
            member("option", { selected: true }),
            composite("listbox", [member("option", { selected: true }), member("option")]),
        ],
    });
    const outerFirst = nodeAt(lb.rt.document, "root/0");
    const inner = nodeAt(lb.rt.document, "root/1");

    // The outer widget's only member is the one option it holds directly, so
    // a move wraps straight back onto itself.
    outerFirst.focus();
    outerFirst.dispatch("keydown", { key: "ArrowDown" });
    assert.equal(lb.focused(), outerFirst);

    // And the inner one has a stop and a rotation of its own.
    assert.deepEqual(inner.children.map((c) => c.getAttribute("tabindex")), ["0", "-1"]);
    inner.children[0].focus();
    inner.children[0].dispatch("keydown", { key: "ArrowDown" });
    assert.equal(lb.focused(), inner.children[1]);
});

test("a nested composite of the OTHER kind does not stop the walk", () => {
    // The other half of the stopping rule, and the one the Go audit used to
    // describe wrongly. compositeMembers stops at a nested container whose
    // members are its own — the test above — and descends through every other
    // one, on the stated grounds that "an option below a tablist is still the
    // listbox's option": the roles say whose it is.
    //
    // So an option buried inside a nested tablist is pooled into the OUTER
    // listbox's rotation. That is deliberate and it is not what a reader would
    // guess from "a nested composite keeps its own members", which is why
    // core.CompositeWalkStopsAt now separates the two cases and the audit's
    // finding says which one an author has built. This is the case that makes
    // the rule discriminate; without it, a runtime that stopped at every
    // composite would pass every other check in this file.
    const lb = mountTree({
        Type: "Column",
        Style: { AccessibilityRole: "listbox" },
        Children: [
            member("option", { selected: true }),
            composite("tablist", [
                member("tab", { selected: true }),
                // A stray option inside the strip. Contrived on purpose: it is
                // the smallest tree that tells the two rules apart.
                member("option"),
            ]),
        ],
    });
    const outerFirst = nodeAt(lb.rt.document, "root/0");
    const strayOption = nodeAt(lb.rt.document, "root/1/1");

    // The stray is in the outer listbox's rotation, not stepped over.
    assert.equal(strayOption.getAttribute("tabindex"), "-1",
        "the option inside the strip was not given a roving tabindex, so the " +
        "outer walk stopped at the tablist after all");
    outerFirst.focus();
    outerFirst.dispatch("keydown", { key: "ArrowDown" });
    assert.equal(lb.focused(), strayOption,
        "the outer listbox's arrows stepped over the tablist whole — which is " +
        "what the audit says only for a pair core.CompositeWalkStopsAt stops at");

    // And the tablist still has a stop and a rotation of its own, so the pair
    // is two tab stops either way. That half of the finding never varies.
    const tab = nodeAt(lb.rt.document, "root/1/0");
    assert.equal(tab.getAttribute("tabindex"), "0");
});

test("a hidden member is not one, and is not one for two reasons", () => {
    // A node pruned from the accessibility tree has no role a reader can see,
    // so it has nothing to be a member of. Upstream of this walk,
    // applyAccessibility already drops a hidden node's role — so this case is
    // covered whether or not the walk checks aria-hidden, and the check below
    // is the half that is not.
    const lb = mountTree(composite("listbox", [
        member("option", { selected: true }),
        { Type: "Box", Style: { AccessibilityRole: "option", AccessibilityHidden: true } },
        member("option"),
    ]));
    const items = lb.root.children;
    assert.equal(items[1].getAttribute("role"), null, "a hidden node kept its role");
    assert.deepEqual(lb.tabindexes(), ["0", null, "-1"]);

    items[0].focus();
    items[0].dispatch("keydown", { key: "ArrowDown" });
    assert.equal(lb.focused(), items[2], "an invisible row was in the rotation");
});

test("a member under a hidden wrapper is not one either", () => {
    // The case the walk's own aria-hidden guard is for, and the only one:
    // a wrapper pruned from the accessibility tree keeps its children's roles,
    // because applyAccessibility answers one node at a time and cannot see an
    // ancestor. The whole subtree is gone as far as a reader is concerned, so
    // arrowing into it would move focus somewhere nothing announces.
    const lb = mountTree({
        Type: "Column",
        Style: { AccessibilityRole: "listbox" },
        Children: [
            member("option", { selected: true }),
            {
                Type: "Box",
                Style: { AccessibilityHidden: true },
                Children: [member("option")],
            },
            member("option"),
        ],
    });
    const buried = nodeAt(lb.rt.document, "root/1/0");
    assert.equal(buried.getAttribute("role"), "option",
        "the wrapper's child lost its own role, so this test proves nothing");
    assert.equal(buried.getAttribute("tabindex"), null);

    const items = lb.root.children;
    items[0].focus();
    items[0].dispatch("keydown", { key: "ArrowDown" });
    assert.equal(lb.focused(), items[2], "a row inside a hidden wrapper was in the rotation");
});

test("a disabled form control is not in the rotation", () => {
    // The browser refuses a disabled <button> focus outright, so arrowing onto
    // it would move the tab stop to a place no focus can follow — and if it
    // were chosen as the entry point the whole widget would be unreachable.
    const tl = mountTree(composite("tablist", [
        member("tab", { selected: true, type: "Button" }),
        member("tab", { type: "Button", disabled: true }),
        member("tab", { type: "Button" }),
    ]));
    const tabs = tl.root.children;
    assert.deepEqual(tl.tabindexes(), ["0", null, "-1"]);

    tabs[0].focus();
    tabs[0].dispatch("keydown", { key: "ArrowRight" });
    assert.equal(tl.focused(), tabs[2]);
});

test("the walk out and the walk in agree about a crossed pair", () => {
    // compositeMembers descends through a composite of the *other* kind — a
    // tablist inside a listbox is not one of the listbox's members and does
    // not close it — so an option below one is still the listbox's member.
    // compositeOf has to walk past the same tablist to say so, which is why it
    // matches on the member's own role rather than stopping at the nearest
    // composite of any kind.
    //
    // Nobody writes this tree on purpose. The two walks still have to answer
    // the same question the same way, and a disagreement between them is a
    // member that is in the rotation but cannot use it.
    const lb = mountTree({
        Type: "Column",
        Style: { AccessibilityRole: "listbox" },
        Children: [
            member("option", { selected: true }),
            {
                Type: "Row",
                Style: { AccessibilityRole: "tablist" },
                Children: [member("option")],
            },
        ],
    });
    const outer = lb.root.children[0];
    const crossed = nodeAt(lb.rt.document, "root/1/0");

    assert.equal(crossed.getAttribute("tabindex"), "-1",
        "the walk in did not count it as one of the listbox's members");

    outer.focus();
    outer.dispatch("keydown", { key: "ArrowDown" });
    assert.equal(lb.focused(), crossed);

    crossed.dispatch("keydown", { key: "ArrowUp" });
    assert.equal(lb.focused(), outer,
        "the walk out stopped at the tablist, so the member the walk in claimed " +
        "cannot move");
});

test("an aria-disabled option stays reachable", () => {
    // The other half of that rule, and it goes the other way. A div carrying
    // aria-disabled is still focusable, and ARIA keeps a disabled option in
    // the rotation on purpose: a user has to be able to find out it is there.
    const lb = mountTree(composite("listbox", [
        member("option", { selected: true }),
        { Type: "Box", Style: { AccessibilityRole: "option", Disabled: true } },
    ]));
    const items = lb.root.children;
    assert.equal(items[1].getAttribute("aria-disabled"), "true");
    assert.deepEqual(lb.tabindexes(), ["0", "-1"]);

    items[0].focus();
    items[0].dispatch("keydown", { key: "ArrowDown" });
    assert.equal(lb.focused(), items[1]);
});

// --------------------------------------------------------------------------
// core.TabView gets it for free
// --------------------------------------------------------------------------

test("a TabView's own bar is navigable, having asked for nothing", () => {
    // The bar is chrome this runtime draws itself, and it writes role=tablist
    // and role=tab from the node type. It is a composite by the same table
    // every hand-built strip goes through, which is what makes this free.
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "TabView",
        Props: { selectedIndex: 1, tabs: [{ label: "A" }, { label: "B" }], onTabChange: "cb_tab" },
        Children: [{ Type: "Column" }, { Type: "Column" }],
    }));
    rt.drainFrames();

    const bar = rt.mountPoint.children[0].children[0];
    assert.deepEqual(bar.children.map((b) => b.getAttribute("tabindex")), ["-1", "0"],
        "the bar's one tab stop is not on the selected tab");

    bar.children[1].focus();
    bar.children[1].dispatch("keydown", { key: "ArrowLeft" });
    assert.equal(rt.document.activeElement, bar.children[0]);
});

test("a TabView's stop follows a selection the app changed", () => {
    // syncTabView writes aria-selected onto the bar from the patch, and the
    // composite pass runs after it — an ordering the tab stop depends on.
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "TabView",
        Props: { selectedIndex: 0, tabs: [{ label: "A" }, { label: "B" }] },
        Children: [{ Type: "Column" }, { Type: "Column" }],
    }));
    rt.drainFrames();

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props",
        TargetID: "root",
        Changes: { selectedIndex: 1, tabs: [{ label: "A" }, { label: "B" }] },
    }]));
    rt.drainFrames();

    const bar = rt.mountPoint.children[0].children[0];
    assert.deepEqual(bar.children.map((b) => b.getAttribute("tabindex")), ["-1", "0"]);
});

// --------------------------------------------------------------------------
// Typeahead
// --------------------------------------------------------------------------
//
// The other half of ARIA's listbox keyboard, and the half that makes a long one
// usable at all: the arrows are fine for three options and useless for a
// hundred. examples/mobileapp's article list is the shape that asked.
//
// It is also the one piece of *state* this section owns. Everything else is
// derived from the DOM on demand, which is what makes the rest survive every
// patch for free; a search string cannot be derived from anything, so there is
// a buffer, and these tests are mostly about the buffer being small and
// expiring correctly.

// A listbox of named options. The name comes from the member's own text, which
// is what a reader would compute — collected leaf by leaf rather than through
// textContent, so it means the same thing here and in a browser.
function namedListbox(names, { selected = 0, labelled = false } = {}) {
    const members = names.map((name, i) => {
        const m = member("option", { selected: i === selected });
        if (labelled) {
            m.Style.AccessibilityLabel = name;
            return m;
        }
        return { ...m, Children: [{ Type: "Text", Props: { content: name } }] };
    });
    return mountTree(composite("listbox", members));
}

const type = (el, key) => el.dispatch("keydown", { key });

test("a printable key jumps to the next member starting with it", () => {
    const lb = namedListbox(["Advent", "Sermons", "Compline"]);
    const items = lb.root.children;
    items[0].focus();

    type(items[0], "s");
    assert.equal(lb.focused(), items[1]);
    // And the tab stop moves with the focus, as it does for the arrows: they
    // are two statements of one fact.
    assert.equal(items[1].getAttribute("tabindex"), "0");
    assert.equal(items[0].getAttribute("tabindex"), "-1");
});

test("the search reads an aria-label ahead of the text", () => {
    // A member that names itself has said what it is called, and the text
    // inside it may be an icon or a count.
    const lb = namedListbox(["Advent", "Sermons"], { labelled: true });
    const items = lb.root.children;
    items[0].focus();

    type(items[0], "s");
    assert.equal(lb.focused(), items[1]);
});

test("a repeated character cycles through the matches", () => {
    // ARIA's own rule: "sss" is not a search for a member called "sss", it is
    // the third press of s. Without it, a listbox with four Sermons entries
    // would be reachable only at its first.
    const lb = namedListbox(["Advent", "Sermons", "Sequence", "Compline"]);
    const items = lb.root.children;
    items[0].focus();

    type(items[0], "s");
    assert.equal(lb.focused(), items[1]);
    type(items[1], "s");
    assert.equal(lb.focused(), items[2]);
    // And round, because there is nothing at either end a stop would protect —
    // the same reason the arrows wrap.
    type(items[2], "s");
    assert.equal(lb.focused(), items[1]);
});

test("a growing string refines rather than cycles", () => {
    const lb = namedListbox(["Advent", "Sermons", "Sequence"]);
    const items = lb.root.children;
    items[0].focus();

    type(items[0], "s");
    assert.equal(lb.focused(), items[1], "s finds Sermons");
    // "se" is a different query from "s" twice: it searches from the current
    // member rather than after it, so a member that still matches stays put.
    type(items[1], "e");
    assert.equal(lb.focused(), items[1], "se still matches Sermons");
    type(items[1], "q");
    assert.equal(lb.focused(), items[2], "seq is Sequence");
});

test("a key that matches nothing is left to the page", () => {
    // The same rule the arrows of the other axis follow. A widget that
    // swallowed every keystroke would break browser shortcuts for a search
    // that found nothing.
    const lb = namedListbox(["Advent", "Sermons"]);
    const items = lb.root.children;
    items[0].focus();

    const e = type(items[0], "z");
    assert.equal(lb.focused(), items[0]);
    assert.equal(e.defaultPrevented, false);
});

test("a modified key belongs to the browser", () => {
    // ctrl-f is find and cmd-l is the address bar. A listbox that consumed
    // either would be worse than one with no typeahead at all.
    const lb = namedListbox(["Advent", "Sermons"]);
    const items = lb.root.children;
    items[0].focus();

    for (const mod of [{ ctrlKey: true }, { metaKey: true }, { altKey: true }]) {
        const e = items[0].dispatch("keydown", { key: "s", ...mod });
        assert.equal(lb.focused(), items[0]);
        assert.equal(e.defaultPrevented, false);
    }
});

test("space stays activation and does not start a search", () => {
    // Handled before the typeahead is reached, which is why a listbox whose
    // options begin with a space is not a shape anyone can type toward.
    const lb = mountTree(composite("listbox", [
        member("option", { selected: true, onClick: "cb_0" }),
        member("option", { onClick: "cb_1" }),
    ]));
    const items = lb.root.children;
    items[0].focus();

    const e = type(items[0], " ");
    assert.equal(lb.focused(), items[0], "space must not move the focus");
    assert.equal(e.defaultPrevented, true, "space is consumed as activation");
    assert.deepEqual(lb.rt.dispatched, [{ id: "cb_0", payload: {} }]);
});

test("a tablist has no typeahead", () => {
    // ARIA's division, not a shortcut: type-to-jump is part of the listbox
    // pattern because a listbox can be a hundred long, and is not part of the
    // tab pattern because a strip's members are all on screen.
    const tl = mountTree(composite("tablist", [
        { ...member("tab", { selected: true }), Children: [{ Type: "Text", Props: { content: "Advent" } }] },
        { ...member("tab"), Children: [{ Type: "Text", Props: { content: "Sermons" } }] },
    ]));
    const tabs = tl.root.children;
    tabs[0].focus();

    const e = type(tabs[0], "s");
    assert.equal(tl.focused(), tabs[0]);
    assert.equal(e.defaultPrevented, false);
});

test("an aria-hidden part of a member is not part of its name", () => {
    // It is pruned from the accessibility tree, so it is not in any name a
    // reader announces — and typing toward it would jump for a reason the user
    // cannot perceive.
    const lb = mountTree(composite("listbox", [
        {
            ...member("option", { selected: true }),
            Children: [{ Type: "Text", Props: { content: "Advent" } }],
        },
        {
            ...member("option"),
            Children: [
                {
                    Type: "Text",
                    Props: { content: "zzz" },
                    Style: { AccessibilityHidden: true },
                },
                { Type: "Text", Props: { content: "Sermons" } },
            ],
        },
    ]));
    const items = lb.root.children;
    items[0].focus();

    // The visible half of the name is what the search sees...
    type(items[0], "s");
    assert.equal(lb.focused(), items[1]);
});

test("and a hidden decoration cannot be typed toward", () => {
    // ...and the hidden half is not there at all. A separate mount because the
    // buffer accumulates within its window: typing z then s in one widget is a
    // search for "zs", not two searches, which is the behaviour the growing
    // string test above is about.
    const lb = mountTree(composite("listbox", [
        {
            ...member("option", { selected: true }),
            Children: [{ Type: "Text", Props: { content: "Advent" } }],
        },
        {
            ...member("option"),
            Children: [
                {
                    Type: "Text",
                    Props: { content: "zzz" },
                    Style: { AccessibilityHidden: true },
                },
                { Type: "Text", Props: { content: "Sermons" } },
            ],
        },
    ]));
    const items = lb.root.children;
    items[0].focus();

    const e = type(items[0], "z");
    assert.equal(lb.focused(), items[0]);
    assert.equal(e.defaultPrevented, false);
});

test("a search in another widget starts over", () => {
    // The buffer records which container it belongs to, so one listbox cannot
    // continue another's search. There is one buffer because only one thing has
    // focus at a time.
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "Column",
        Children: [
            composite("listbox", [
                { ...member("option", { selected: true }), Children: [{ Type: "Text", Props: { content: "Sermons" } }] },
                { ...member("option"), Children: [{ Type: "Text", Props: { content: "Sequence" } }] },
            ]),
            composite("listbox", [
                { ...member("option", { selected: true }), Children: [{ Type: "Text", Props: { content: "Advent" } }] },
                { ...member("option"), Children: [{ Type: "Text", Props: { content: "Easter" } }] },
            ]),
        ],
    }));
    rt.drainFrames();
    const first = nodeAt(rt.document, "root/0").children;
    const second = nodeAt(rt.document, "root/1").children;

    first[0].focus();
    type(first[0], "s");
    // On the member that now holds focus, which is where a browser delivers
    // the next key — the same reason handleCompositeKey reads currentTarget.
    type(first[1], "e");
    assert.equal(rt.document.activeElement, first[1], "se is Sequence");

    // "e" here is a fresh search, not the third character of "see".
    second[0].focus();
    type(second[0], "e");
    assert.equal(rt.document.activeElement, second[1], "Easter, not a continued search");
});

// --------------------------------------------------------------------------
// The toolbar: a composite whose members ARIA does not name
// --------------------------------------------------------------------------
//
// The other two composites are told what their members are by their own role —
// a listbox has options, a tablist has tabs. A toolbar is not: ARIA calls it "a
// collection of commonly used function buttons or controls" and defines no
// `toolbaritem`, which is exactly why it sat in the orientation table with no
// keyboard for two releases. The rule below is what the absence forces: every
// focusable control inside it that is not inside a nested composite.

test("a toolbar is one tab stop over a run of controls", () => {
    // The claim the whole thing exists for. comps.ChipStrip is a Row of
    // core.Buttons, so before this a twelve-chip filter bar was twelve stops in
    // the page's tab order and ARIA promises one.
    const tb = toolbar([
        control("All", { onClick: "cb_0" }),
        control("Sermons", { onClick: "cb_1" }),
        control("Articles", { onClick: "cb_2" }),
    ]);
    assert.deepEqual(tb.tabindexes(), ["0", "-1", "-1"]);
});

test("a toolbar moves on Left and Right, because it is a Row", () => {
    // Same derivation as the tablist: the arrow pair comes off
    // aria-orientation, which applyAccessibility wrote from the container's own
    // resolved flex-direction. A toolbar has been in that table since before it
    // had a keyboard, which is why nothing had to be added for this.
    const tb = toolbar([
        control("All"), control("Sermons"), control("Articles"),
    ]);
    const chips = tb.root.children;
    chips[0].focus();

    chips[0].dispatch("keydown", { key: "ArrowRight" });
    assert.equal(tb.focused(), chips[1]);
    assert.deepEqual(tb.tabindexes(), ["-1", "0", "-1"]);

    chips[1].dispatch("keydown", { key: "ArrowLeft" });
    assert.equal(tb.focused(), chips[0]);

    // Home and End come from the same switch and need no separate rule.
    chips[0].dispatch("keydown", { key: "End" });
    assert.equal(tb.focused(), chips[2]);
});

test("a toolbar laid out as a column takes the vertical arrows", () => {
    // The vertical strip htmlout/orientation.go said the toolbar row was
    // waiting for. Nothing here is toolbar-specific — it is the same
    // compositeIsVertical read the other two composites do.
    const tb = mountTree({
        Type: "Column",
        Style: { AccessibilityRole: "toolbar" },
        Children: [control("Bold"), control("Italic")],
    });
    const chips = tb.root.children;
    chips[0].focus();

    chips[0].dispatch("keydown", { key: "ArrowDown" });
    assert.equal(tb.focused(), chips[1]);
});

test("a control the framework built out of a Box is a member too", () => {
    // The second half of the membership rule. core.RoleButton exists for "a
    // tappable container — a Box or a Row with an OnTap", which every renderer
    // draws as scenery, and a toolbar of icon boxes is an ordinary thing to
    // build. A rule that only knew about <button> would give that toolbar one
    // tab stop and nothing to move it to.
    const tb = toolbar([
        { Type: "Box", Style: { AccessibilityRole: "button" }, Props: { onClick: "cb_0" } },
        { Type: "Box", Style: { AccessibilityRole: "link" }, Props: { onClick: "cb_1" } },
    ]);
    assert.deepEqual(tb.tabindexes(), ["0", "-1"]);
});

test("a Box that says it is a control and has no handler is not one", () => {
    // The `&& listener_onClick` half. A Box carrying role="button" with nothing
    // behind it is a labelling mistake rather than a control, and putting a tab
    // stop on it would send a keyboard user to an element that does nothing.
    const tb = toolbar([
        control("All", { onClick: "cb_0" }),
        { Type: "Box", Style: { AccessibilityRole: "button" } },
    ]);
    assert.deepEqual(tb.tabindexes(), ["0", null]);
});

test("the scenery between a toolbar's controls is not in the rotation", () => {
    // A strip with a label, a separator and a count in it. Every one of those
    // is a div this framework draws as scenery, and a walk that took "every
    // child" would put three dead stops in the middle of the arrow order.
    const tb = toolbar([
        { Type: "Text", Props: { content: "Filter:" } },
        control("All", { onClick: "cb_0" }),
        { Type: "Box" },
        control("Sermons", { onClick: "cb_1" }),
    ]);
    assert.deepEqual(tb.tabindexes(), [null, "0", null, "-1"]);

    const chips = tb.root.children;
    chips[1].focus();
    chips[1].dispatch("keydown", { key: "ArrowRight" });
    assert.equal(tb.focused(), chips[3], "the arrows step over the scenery");
});

test("a control buried inside a wrapper is still a member", () => {
    // Same reason compositeMembers is a subtree walk: nothing says a member is
    // a direct child, and a scrollable strip puts its buttons inside a scroller.
    const tb = toolbar([
        { Type: "Box", Children: [control("All", { onClick: "cb_0" })] },
        { Type: "Box", Children: [control("Sermons", { onClick: "cb_1" })] },
    ]);
    const inner = [tb.root.children[0].children[0], tb.root.children[1].children[0]];
    assert.equal(inner[0].getAttribute("tabindex"), "0");
    assert.equal(inner[1].getAttribute("tabindex"), "-1");

    inner[0].focus();
    inner[0].dispatch("keydown", { key: "ArrowRight" });
    assert.equal(tb.focused(), inner[1]);
});

test("a member is a leaf: a button with an icon inside it is one stop", () => {
    // The walk does not descend into a control it has found. A Button holding a
    // span and a label is one thing to Tab to, not three.
    const tb = toolbar([
        {
            Type: "Button", Props: { onClick: "cb_0" },
            Children: [
                { Type: "Text", Props: { content: "★" } },
                { Type: "Text", Props: { content: "Star" } },
            ],
        },
        control("Sermons", { onClick: "cb_1" }),
    ]);
    assert.deepEqual(tb.tabindexes(), ["0", "-1"]);
    assert.deepEqual(
        tb.root.children[0].children.map((c) => c.getAttribute("tabindex")),
        [null, null]);
});

test("a hidden control is not a member", () => {
    // Same two prunings the other walk makes. aria-hidden is out of the
    // accessibility tree, so arrowing to it would move focus somewhere a
    // screen reader says nothing about.
    const tb = toolbar([
        control("All", { onClick: "cb_0" }),
        { Type: "Button", Style: { AccessibilityHidden: true }, Props: { content: "x" } },
        control("Sermons", { onClick: "cb_1" }),
    ]);
    assert.deepEqual(tb.tabindexes(), ["0", null, "-1"]);
});

test("a disabled control is not in the rotation", () => {
    // The browser refuses a disabled <button> focus outright, so a tab stop on
    // one is a stop no focus can follow — the same reason compositeMembers
    // excludes a disabled form control.
    const tb = toolbar([
        control("All", { onClick: "cb_0" }),
        control("Sermons", { onClick: "cb_1", disabled: true }),
        control("Articles", { onClick: "cb_2" }),
    ]);
    assert.deepEqual(tb.tabindexes(), ["0", null, "-1"]);

    const chips = tb.root.children;
    chips[0].focus();
    chips[0].dispatch("keydown", { key: "ArrowRight" });
    assert.equal(tb.focused(), chips[2]);
});

test("a nested composite keeps its own controls, and keeps its own stop", () => {
    // The one place the two member rules disagree, and the disagreement is
    // deliberate. compositeMembers descends *through* a composite of the other
    // kind, because the roles say whose an option is. Nothing says whose a
    // button is, so the focusable walk stops at a nested composite instead of
    // pooling both sets.
    //
    // The consequence, asserted rather than hidden: the tablist inside keeps
    // its own roving tabindex, so this shape is two tab stops rather than one.
    // Every control stays reachable, which is what the alternatives lose.
    const tb = toolbar([
        control("All", { onClick: "cb_0" }),
        composite("tablist", [
            member("tab", { selected: true, onClick: "cb_1", type: "Button" }),
            member("tab", { onClick: "cb_2", type: "Button" }),
        ]),
        control("Articles", { onClick: "cb_3" }),
    ]);
    const [first, strip, last] = tb.root.children;
    assert.deepEqual([first.getAttribute("tabindex"), last.getAttribute("tabindex")],
        ["0", "-1"], "the toolbar's own two controls are its members");
    assert.deepEqual(strip.children.map((c) => c.getAttribute("tabindex")), ["0", "-1"],
        "and the strip inside still has a stop of its own");

    // The arrows step over the whole strip rather than into it.
    first.focus();
    first.dispatch("keydown", { key: "ArrowRight" });
    assert.equal(tb.focused(), last);
});

test("a button inside a listbox is chrome, not a toolbar member of anything", () => {
    // The walk out has to agree with the walk in here too. A "Load more" button
    // below a listbox's options is not a member of the listbox — that walk
    // steps over it — and compositeOf must not answer "the listbox" for it,
    // which would put the toolbar keyboard on a widget that already has one.
    const lb = mountTree(composite("listbox", [
        member("option", { selected: true, onClick: "cb_0" }),
        member("option", { onClick: "cb_1" }),
        control("Load more", { onClick: "cb_2" }),
    ]));
    const more = lb.root.children[2];
    assert.equal(more.getAttribute("tabindex"), null,
        "the button keeps its own place in the page's tab order");

    // And its keys are its own: no listener was ever attached, so an arrow
    // reaches the page.
    const e = more.dispatch("keydown", { key: "ArrowDown" });
    assert.equal(e.defaultPrevented, false);
});

test("a toolbar has no typeahead", () => {
    // COMPOSITE_TYPEAHEAD is a listbox and nothing else, which is ARIA's own
    // division: type-to-jump is part of the listbox pattern because a listbox
    // can be a hundred long. A toolbar's controls are all on screen, and a
    // toolbar that swallowed printable keys would take them from a page that
    // may have a search shortcut on one.
    const tb = toolbar([
        control("All", { onClick: "cb_0" }),
        control("Sermons", { onClick: "cb_1" }),
    ]);
    const chips = tb.root.children;
    chips[0].focus();

    const e = chips[0].dispatch("keydown", { key: "s" });
    assert.equal(tb.focused(), chips[0]);
    assert.equal(e.defaultPrevented, false);
});

test("an empty toolbar is left alone", () => {
    // A strip whose chips have not arrived yet, and a toolbar of pure scenery.
    // Neither is a widget with a tab stop to place.
    const empty = toolbar([]);
    assert.equal(empty.root.getAttribute("tabindex"), null);

    const scenery = toolbar([{ Type: "Text", Props: { content: "Filter:" } }]);
    assert.deepEqual(scenery.tabindexes(), [null]);
});

test("a control added by a patch joins the toolbar", () => {
    // Same total re-sync every composite gets: syncTouchedComposites walks up
    // from the touched element and down into it, and the whole member list is
    // rewritten rather than appended to.
    const tb = toolbar([control("All", { onClick: "cb_0" })]);
    assert.deepEqual(tb.tabindexes(), ["0"]);

    tb.rt.GrMob.patch(JSON.stringify([{
        Type: "add-child",
        TargetID: "root",
        Changes: control("Sermons", { onClick: "cb_1" }),
    }]));
    tb.rt.drainFrames();
    assert.deepEqual(tb.tabindexes(), ["0", "-1"]);

    const chips = tb.root.children;
    chips[0].focus();
    chips[0].dispatch("keydown", { key: "ArrowRight" });
    assert.equal(tb.focused(), chips[1], "the new control is wired, not just counted");
});

// --------------------------------------------------------------------------
// Selection follows focus
// --------------------------------------------------------------------------
//
// The half of ARIA's listbox and tabs patterns this section left out for two
// releases, and the standing argument against it was that the framework could
// not make the choice: aria-selected is rendered from Go state, and a keystroke
// cannot reach Go state without a render pass.
//
// The conclusion was right and the premise was false. Enter and Space on a
// member have always reached Go — activateCompositeMember calls
// GoInvokeCallback — so a keystroke has had a way to change a selection since
// the day this section was written. What was actually missing was a decision
// about *when*, and that is not one the framework can make for an author:
// ARIA recommends it for tabs over cheap panels and warns about it for anything
// expensive, so it is a prop.

// A composite carrying the flag.
function following(role, members, opts = {}) {
    const tree = composite(role, members, opts);
    tree.Style.AccessibilitySelectionFollowsFocus = true;
    return mountTree(tree);
}

test("the flag is written and is total", () => {
    // It is not an ARIA attribute — ARIA says what a widget is, and this says
    // what its keyboard does — so it rides in the data channel. Total like
    // every attribute applyAccessibility writes: a widget that stops asking
    // must stop getting it, or a strip that turned the behaviour off would keep
    // firing selections for the life of the page.
    const lb = following("listbox", [member("option", { selected: true })]);
    assert.equal(lb.root.getAttribute("data-grmob-selection-follows-focus"), "true");

    lb.rt.GrMob.patch(JSON.stringify([{
        Type: "update-style",
        TargetID: "root",
        Changes: { AccessibilityRole: "listbox" },
    }]));
    lb.rt.drainFrames();
    assert.equal(lb.root.getAttribute("data-grmob-selection-follows-focus"), null);
});

test("without the flag an arrow moves focus and chooses nothing", () => {
    // The default, and it stays the default. A hundred-option list whose
    // selection fires a request must not fire a hundred of them because
    // somebody held ArrowDown.
    const lb = listbox({ selected: 0 });
    const items = lb.root.children;
    items[0].focus();
    items[0].dispatch("keydown", { key: "ArrowDown" });

    assert.equal(lb.focused(), items[1]);
    assert.deepEqual(lb.rt.dispatched, []);
});

test("an arrow chooses the member it lands on", () => {
    const lb = following("listbox", [0, 1, 2].map((i) =>
        member("option", { selected: i === 0, onClick: `cb_${i}` })));
    const items = lb.root.children;
    items[0].focus();

    items[0].dispatch("keydown", { key: "ArrowDown" });
    assert.equal(lb.focused(), items[1]);
    assert.deepEqual(lb.rt.dispatched, [{ id: "cb_1", payload: {} }],
        "the author's own OnTap is what runs — the runtime does not write " +
        "aria-selected and could not, since that is rendered from Go state");
});

test("Home, End and a typeahead match choose too", () => {
    // moveCompositeFocus is the single funnel for every movement here, which
    // is why the hook is in it rather than in each key's arm. A version that
    // handled only the arrows would leave a typed jump selecting nothing,
    // which is the exact shape of the item this closes.
    const lb = following("listbox", ["Alpha", "Beta", "Gamma"].map((label, i) => ({
        Type: "Box",
        Style: {
            AccessibilityRole: "option",
            AccessibilitySelected: i === 0 ? "true" : "false",
        },
        Props: { onClick: `cb_${i}`, content: label },
    })));
    const items = lb.root.children;
    items[0].focus();

    items[0].dispatch("keydown", { key: "End" });
    assert.equal(lb.focused(), items[2]);

    items[2].dispatch("keydown", { key: "b" });
    assert.equal(lb.focused(), items[1], "the typeahead found Beta");

    assert.deepEqual(lb.rt.dispatched.map((c) => c.id), ["cb_2", "cb_1"]);
});

test("a <button> tab is chosen too, which activation would have refused", () => {
    // The one place this differs from activateCompositeMember, and it is the
    // case that matters most. That function returns early for a <button>
    // because Enter and Space already make the browser fire a real click, so
    // synthesizing one would run the handler twice. An arrow key fires nothing
    // on anything — and ARIA recommends selection-follows-focus for tabs above
    // all, where every member is a <button>.
    const tl = following("tablist", [0, 1, 2].map((i) => member("tab", {
        selected: i === 0, onClick: `cb_${i}`, type: "Button",
    })));
    const tabs = tl.root.children;
    tabs[0].focus();

    tabs[0].dispatch("keydown", { key: "ArrowRight" });
    assert.equal(tl.focused(), tabs[1]);
    assert.deepEqual(tl.rt.dispatched, [{ id: "cb_1", payload: {} }]);
});

test("a member with no handler is focused and nothing else", () => {
    // The same rule activation follows. A run of options where only some are
    // tappable is an ordinary shape, and arrowing onto an inert one must not
    // be an error.
    const lb = following("listbox", [
        member("option", { selected: true, onClick: "cb_0" }),
        member("option"),
    ]);
    const items = lb.root.children;
    items[0].focus();
    items[0].dispatch("keydown", { key: "ArrowDown" });

    assert.equal(lb.focused(), items[1]);
    assert.deepEqual(lb.rt.dispatched, []);
});

test("a patch does not fire a selection, however much it moves", () => {
    // The loop this would be. syncComposite runs on every patch and rewrites
    // the whole roving tabindex; a selection fired from there would call back
    // into Go, produce a patch, and fire again. So the hook is in
    // moveCompositeFocus, which only a keystroke reaches, and this is the test
    // that says so.
    const lb = following("listbox", [
        member("option", { selected: true, onClick: "cb_0" }),
        member("option", { onClick: "cb_1" }),
    ]);
    lb.rt.GrMob.patch(JSON.stringify([{
        Type: "update-style",
        TargetID: "root/1",
        Changes: { AccessibilityRole: "option", AccessibilitySelected: "true" },
    }]));
    lb.rt.drainFrames();

    assert.deepEqual(lb.rt.dispatched, [],
        "a selection the app changed must not be echoed back to the app");
});

test("the flag on a container that is not a composite does nothing", () => {
    // applyAccessibility writes the attribute without consulting the composite
    // tables, deliberately: knowing them there would put the tables in two
    // places. The attribute on a `list` is read by nobody, which is the same
    // nothing that happened before it existed.
    const l = mountTree({
        Type: "Column",
        Style: { AccessibilityRole: "list", AccessibilitySelectionFollowsFocus: true },
        Children: [member("listitem", { onClick: "cb_0" })],
    });
    assert.deepEqual(l.tabindexes(), [null]);
    assert.deepEqual(l.rt.dispatched, []);
});
