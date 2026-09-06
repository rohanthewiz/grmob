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

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

// One member of a composite: a Box carrying the member role, an optional
// selection state, and an onClick — which is what components.ListRow's
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
