// The TabView bar and page selection, in the runtime.
//
// core.TabView's four wire props (tabs, selectedIndex, onTabChange, one child
// per page) went entirely unread here until this pass: a TabView was a bare box
// holding every page at once, with no bar and no way to switch, while both
// natives drew a bar above the selected page alone. These tests hold the
// runtime to the behaviour htmlout/tabview.go states for the pair, plus the two
// things only the live target can get wrong — the click dispatch, and the
// bookkeeping that keeps a bar from throwing the child indices out of step.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

const page = (text) => ({ Type: "Column", Children: [{ Type: "Text", Props: { content: text } }] });

const TABS = [
    { label: "Home", icon: "🏠" },
    { label: "Search", icon: "🔍" },
];

// A TabView at root, with as many pages as it has tabs.
function mountTabs({ selectedIndex = 0, onTabChange = "cb_tab", tabs = TABS } = {}) {
    const rt = loadRuntime();
    const props = { selectedIndex, tabs };
    if (onTabChange) props.onTabChange = onTabChange;
    rt.GrMob.mount(
        JSON.stringify({
            Type: "TabView",
            Props: props,
            Children: tabs.map((t) => page(`${t.label} page`)),
        })
    );
    rt.drainFrames();
    const root = rt.mountPoint.children[0];
    return {
        rt,
        root,
        bar: () => root.children[0],
        // The pages, by *node* index — which is what every path and every
        // patch means, and is one more than the DOM index while a bar exists.
        pageAt: (i) => nodeAt(rt.document, `root/${i}`),
    };
}

// --------------------------------------------------------------------------
// The bar
// --------------------------------------------------------------------------

test("a TabView draws a bar ahead of its pages", () => {
    const { root, bar } = mountTabs();

    assert.equal(root.children.length, 3, "bar plus two pages");
    assert.equal(bar().dataset.grmobChrome, "tabbar");
    assert.equal(bar().getAttribute("role"), "tablist");
    // Chrome is not a node: nothing may be addressed to it.
    assert.equal(bar().getAttribute("data-node-path"), null);
    assert.equal(bar().children.length, 2);
    assert.deepEqual(
        bar().children.map((b) => b.textContent),
        ["Home", "Search"]
    );
    assert.deepEqual(
        bar().children.map((b) => b.getAttribute("role")),
        ["tab", "tab"]
    );
});

// The icon half of a core.TabItem is drawn by no target: Compose's Tab is
// built with `text = { Text(label) }` and the SwiftUI bar with a Text of the
// label. Drawing it here would make the web the outlier instead of closing a
// gap.
test("a tab shows its label and not its icon", () => {
    const { bar } = mountTabs();
    for (const button of bar().children) {
        assert.ok(!button.textContent.includes("🏠"), `icon leaked into ${button.textContent}`);
    }
});

test("a TabView with no tabs draws no bar at all", () => {
    const { root } = mountTabs({ tabs: [] });
    assert.equal(root.children.length, 0);
});

test("the selected tab is the one marked selected", () => {
    const { bar } = mountTabs({ selectedIndex: 1 });
    assert.deepEqual(
        bar().children.map((b) => b.getAttribute("aria-selected")),
        ["false", "true"]
    );
    assert.equal(bar().children[1].style.fontWeight, "600");
    assert.equal(bar().children[0].style.fontWeight, "");
});

// The callback ID is mirrored onto the bar so the live DOM carries the same
// data-ontabchange htmlout writes into an export. The listener is what
// dispatches; this is the record of it.
test("the bar records the callback ID, and drops it when there is none", () => {
    assert.equal(mountTabs().bar().getAttribute("data-ontabchange"), "cb_tab");
    assert.equal(mountTabs({ onTabChange: null }).bar().getAttribute("data-ontabchange"), null);
});

// --------------------------------------------------------------------------
// Selection
// --------------------------------------------------------------------------

test("only the selected page is visible", () => {
    const { pageAt } = mountTabs({ selectedIndex: 1 });
    assert.equal(pageAt(0).style.display, "none");
    // Not "" — a Column is a flex container whether or not its Style asks
    // (stackAxisFor), and a page restored by clearing the declaration would
    // come back in block flow.
    assert.equal(pageAt(1).style.display, "flex");
});

// Both natives guard the page rather than clamping the index
// (`children.indices.contains` in Swift, `getOrNull` in Kotlin), and the Swift
// bar compares `i == selected` with no clamp either.
test("an out-of-range selection shows no page and marks no tab", () => {
    const { bar, pageAt } = mountTabs({ selectedIndex: 7 });
    assert.equal(pageAt(0).style.display, "none");
    assert.equal(pageAt(1).style.display, "none");
    assert.deepEqual(
        bar().children.map((b) => b.getAttribute("aria-selected")),
        ["false", "false"]
    );
});

test("a click on a tab dispatches its index to onTabChange", () => {
    const { rt, bar } = mountTabs();

    bar().children[1].dispatch("click");

    // A number, not a string: ReceiveEventPayload sniffs the value's type, and
    // only a number reaches the int callback map core.OnTabChange registered in.
    assert.deepEqual(rt.dispatched, [{ id: "cb_tab", payload: { value: 1 } }]);
});

test("a bar whose callback was pruned goes inert rather than dispatching a stale ID", () => {
    const { rt, root, bar } = mountTabs();

    // The pass stopped carrying onTabChange. An update-props patch sends the
    // whole new map, so its absence is definitive — pruneStaleListeners drops
    // the ID, and callback IDs are positional, so keeping it would fire some
    // other node's handler.
    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-props", TargetID: "root", Changes: { selectedIndex: 0, tabs: TABS } },
        ])
    );
    bar().children[1].dispatch("click");

    assert.deepEqual(rt.dispatched, []);
    assert.equal(root.children[0].getAttribute("data-ontabchange"), null);
});

// --------------------------------------------------------------------------
// Switching tabs
// --------------------------------------------------------------------------

// core.SelectedIndex is controlled state, so a switch reaches the page as a
// prop patch on the TabView and never as a subtree replacement: the pages are
// never rebuilt, only re-hidden.
test("a selectedIndex patch moves both the indicator and the visible page", () => {
    const { rt, bar, pageAt } = mountTabs({ selectedIndex: 0 });

    rt.GrMob.patch(
        JSON.stringify([
            {
                Type: "update-props",
                TargetID: "root",
                Changes: { selectedIndex: 1, tabs: TABS, onTabChange: "cb_tab" },
            },
        ])
    );

    assert.equal(pageAt(0).style.display, "none");
    assert.equal(pageAt(1).style.display, "flex");
    assert.deepEqual(
        bar().children.map((b) => b.getAttribute("aria-selected")),
        ["false", "true"]
    );
});

// The bar is rebuilt only when the tab strip itself changed. A switch is
// exactly when a keyboard user has one of these buttons focused, and a rebuild
// would throw that focus away on every tab press.
test("switching tabs keeps the same button elements", () => {
    const { rt, bar } = mountTabs({ selectedIndex: 0 });
    const before = bar().children[0];

    rt.GrMob.patch(
        JSON.stringify([
            {
                Type: "update-props",
                TargetID: "root",
                Changes: { selectedIndex: 1, tabs: TABS, onTabChange: "cb_tab" },
            },
        ])
    );

    assert.equal(bar().children[0], before, "the bar was rebuilt on a mere selection change");
});

test("a changed tab strip does rebuild the bar", () => {
    const { rt, bar } = mountTabs();
    const renamed = [{ label: "Home" }, { label: "Explore" }, { label: "Me" }];

    rt.GrMob.patch(
        JSON.stringify([
            {
                Type: "update-props",
                TargetID: "root",
                Changes: { selectedIndex: 0, tabs: renamed, onTabChange: "cb_tab" },
            },
        ])
    );

    assert.deepEqual(
        bar().children.map((b) => b.textContent),
        ["Home", "Explore", "Me"]
    );
    // Still ahead of the pages, so the chrome offset holds.
    assert.equal(bar().dataset.grmobChrome, "tabbar");
});

// --------------------------------------------------------------------------
// The hiding against the rest of the runtime
// --------------------------------------------------------------------------

// styleFromGrMob is *total*: it assigns every property it manages on every
// pass, so an update-style on a hidden page rewrites its display. The
// selection is recomputed at the end of each batch for exactly this reason.
test("an update-style on a hidden page does not reveal it", () => {
    const { rt, pageAt } = mountTabs({ selectedIndex: 0 });

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-style", TargetID: "root/1", Changes: { Background: "#eee" } },
        ])
    );

    assert.equal(pageAt(1).style.display, "none", "a style patch unhid a page");
    assert.equal(pageAt(1).style.background, "#eee", "the style patch itself was lost");
});

// The visible page's display comes back from what the style pass computed, not
// from clearing the declaration.
test("a style patch on the visible page leaves it visible", () => {
    const { rt, pageAt } = mountTabs({ selectedIndex: 0 });

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-style", TargetID: "root/0", Changes: { Padding: { Top: 8, Right: 8, Bottom: 8, Left: 8 } } },
        ])
    );

    assert.equal(pageAt(0).style.display, "flex");
});

// --------------------------------------------------------------------------
// The chrome offset
// --------------------------------------------------------------------------
//
// Patch paths address a child by its *node* index, and the bar makes the DOM
// index one higher. Both places that convert have to shift past the chrome, or
// a page lands in front of the bar and every path after it is out of step.

test("an added page lands after the bar, at the slot its path names", () => {
    const { rt, root, bar } = mountTabs({ selectedIndex: 0 });

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "add", TargetID: "root/0", Changes: page("inserted") },
        ])
    );

    assert.equal(root.children[0], bar(), "the bar was pushed out of first place");
    assert.equal(root.children[1].getAttribute("data-node-path"), "root/0");
    assert.equal(nodeAt(rt.document, "root/0"), root.children[1]);
    assert.equal(root.children[1].children[0].textContent, "inserted");
});

test("add-child numbers the new page from the pages, not from the DOM", () => {
    const { rt, root } = mountTabs({ selectedIndex: 0 });

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "add-child", TargetID: "root", Changes: page("third") },
        ])
    );

    // Two pages existed, so the new one is page 2 — not 3, which is what the
    // DOM child count alone would have said.
    assert.equal(root.children.length, 4);
    assert.equal(root.children[3].getAttribute("data-node-path"), "root/2");
    assert.equal(nodeAt(rt.document, "root/2"), root.children[3]);
});

// A TabView with no bar must not be given a phantom offset — the offset is
// counted from the children, not assumed from the node type.
test("a TabView with no tabs numbers its pages from zero", () => {
    const rt = loadRuntime();
    rt.GrMob.mount(
        JSON.stringify({ Type: "TabView", Props: { selectedIndex: 0 }, Children: [page("only")] })
    );
    rt.drainFrames();
    const root = rt.mountPoint.children[0];

    rt.GrMob.patch(
        JSON.stringify([{ Type: "add-child", TargetID: "root", Changes: page("second") }])
    );

    assert.equal(root.children[1].getAttribute("data-node-path"), "root/1");
});

// --------------------------------------------------------------------------
// The tab/panel wiring
// --------------------------------------------------------------------------
//
// A well-formed tablist still says nothing about which region each tab
// governs. That relationship needs ids, and the ids are derived from the node
// path so they are the same strings htmlout writes — TestTabsAndPanelsPointAtEachOther
// in htmlout/tabview_test.go asserts these exact literals for the same tree.

test("a tab and its page point at each other", () => {
    const { root, bar, pageAt } = mountTabs();

    for (let i = 0; i < 2; i++) {
        const tab = bar().children[i];
        assert.equal(tab.getAttribute("id"), `grmob-root-tab-${i}`);
        assert.equal(tab.getAttribute("aria-controls"), `grmob-root-panel-${i}`);

        const page = pageAt(i);
        assert.equal(page.getAttribute("id"), `grmob-root-panel-${i}`);
        assert.equal(page.getAttribute("role"), "tabpanel");
        assert.equal(page.getAttribute("aria-labelledby"), `grmob-root-tab-${i}`);
    }
    // Every reference resolves inside the tree, which is the property the ids
    // exist for: a dangling IDREF reads as a region that is not there.
    for (const tab of bar().children) {
        const target = root.findAll((el) => el.getAttribute("id") === tab.getAttribute("aria-controls"));
        assert.equal(target.length, 1, `aria-controls of ${tab.textContent} names no single element`);
    }
});

// Ids are document-global. The scope is the node path, so a nested TabView
// cannot collide with the one above it — and does not need a counter to avoid
// it, which is what keeps these ids equal to the exporter's.
test("a nested TabView gets its own id scope", () => {
    const rt = loadRuntime();
    rt.GrMob.mount(
        JSON.stringify({
            Type: "Column",
            Children: [
                { Type: "TabView", Props: { selectedIndex: 0, tabs: TABS }, Children: [page("a"), page("b")] },
                { Type: "TabView", Props: { selectedIndex: 0, tabs: TABS }, Children: [page("c"), page("d")] },
            ],
        })
    );
    rt.drainFrames();
    const root = rt.mountPoint.children[0];

    const ids = root.findAll((el) => el.getAttribute("id") !== null).map((el) => el.getAttribute("id"));
    assert.equal(new Set(ids).size, ids.length, `duplicate ids: ${ids}`);
    assert.ok(ids.includes("grmob-root-0-panel-0"), `scope is not the node path: ${ids}`);
    assert.ok(ids.includes("grmob-root-1-panel-0"), `scope is not the node path: ${ids}`);
});

// role="tabpanel" would replace the role the browser already gives a <button>,
// an <img> or an <input>, so those pages are left alone — and the tab must not
// point at one, since aria-controls naming a non-panel is a lie about the
// document. GENERIC_TAGS is the set, pinned to Go's by
// TestRuntimeGenericTagsMatchGo.
test("a page whose element already has a role is not made a panel", () => {
    const rt = loadRuntime();
    rt.GrMob.mount(
        JSON.stringify({
            Type: "TabView",
            Props: { selectedIndex: 0, tabs: TABS },
            Children: [page("home"), { Type: "Button", Props: { label: "press" } }],
        })
    );
    rt.drainFrames();
    const root = rt.mountPoint.children[0];
    const bar = root.children[0];

    assert.equal(root.children[2].getAttribute("role"), null, "a <button> page was given the panel role");
    assert.equal(root.children[2].getAttribute("id"), null);
    assert.equal(bar.children[1].getAttribute("aria-controls"), null, "its tab still claims to control it");
    // The eligible page beside it keeps its wiring: one page opting out is not
    // the tab set opting out.
    assert.equal(bar.children[0].getAttribute("aria-controls"), "grmob-root-panel-0");
});

// The author took the page out of the accessibility tree on purpose. Read off
// the element rather than out of the Style, because that is where
// applyAccessibility put it and this runs long after.
test("an aria-hidden page is not made a panel", () => {
    const { rt, root, pageAt } = mountTabs();

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-style", TargetID: "root/1", Changes: { AccessibilityHidden: true } },
        ])
    );

    assert.equal(pageAt(1).getAttribute("aria-hidden"), "true");
    assert.equal(pageAt(1).getAttribute("role"), null);
    assert.equal(root.children[0].children[1].getAttribute("aria-controls"), null);
});

// The wiring is derived state like the selection, so it has to survive the
// same three ways a batch can invalidate it. Here: a page becomes eligible
// again after having been ruled out, which a guarded write would never undo.
test("a page that becomes eligible again is rewired", () => {
    const { rt, root, pageAt } = mountTabs();

    const hide = (v) =>
        rt.GrMob.patch(
            JSON.stringify([
                { Type: "update-style", TargetID: "root/1", Changes: { AccessibilityHidden: v } },
            ])
        );
    hide(true);
    assert.equal(pageAt(1).getAttribute("role"), null);

    hide(false);
    assert.equal(pageAt(1).getAttribute("role"), "tabpanel");
    assert.equal(pageAt(1).getAttribute("id"), "grmob-root-panel-1");
    assert.equal(root.children[0].children[1].getAttribute("aria-controls"), "grmob-root-panel-1");
});

// aria-labelledby wins over aria-label in the accessible-name calculation, so
// writing it unconditionally would discard the name the app author chose. The
// tab still points at the panel; only the naming is left alone.
test("a page that names itself keeps its name", () => {
    const { rt, root, pageAt } = mountTabs();

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-style", TargetID: "root/0", Changes: { AccessibilityLabel: "Your home feed" } },
        ])
    );

    assert.equal(pageAt(0).getAttribute("aria-label"), "Your home feed");
    assert.equal(pageAt(0).getAttribute("aria-labelledby"), null, "the tab's name overrode the author's");
    assert.equal(pageAt(0).getAttribute("role"), "tabpanel", "the page stopped being a panel because it had a name");
    assert.equal(root.children[0].children[0].getAttribute("aria-controls"), "grmob-root-panel-0");
});

// A page no tab names is not part of a tab set. The count comes from the bar,
// so a tabs prop that shrinks takes the surplus page's wiring with it.
test("a page with no tab of its own is not a panel", () => {
    const { rt, root, pageAt } = mountTabs();

    rt.GrMob.patch(
        JSON.stringify([
            {
                Type: "update-props",
                TargetID: "root",
                Changes: { selectedIndex: 0, tabs: [{ label: "Home" }], onTabChange: "cb_tab" },
            },
        ])
    );

    assert.equal(root.children[0].children.length, 1, "the bar was not rebuilt from the new strip");
    assert.equal(pageAt(0).getAttribute("role"), "tabpanel");
    assert.equal(pageAt(1).getAttribute("role"), null, "a page no tab names was left wired");
    assert.equal(pageAt(1).getAttribute("id"), null);
});

// No bar, no tab set, nothing to wire — and no ids to collide with a real
// TabView's elsewhere in the document.
test("a TabView with no tabs wires no panels", () => {
    const rt = loadRuntime();
    rt.GrMob.mount(
        JSON.stringify({ Type: "TabView", Props: { selectedIndex: 0 }, Children: [page("only")] })
    );
    rt.drainFrames();
    const root = rt.mountPoint.children[0];

    assert.equal(root.children[0].getAttribute("role"), null);
    assert.equal(root.children[0].getAttribute("id"), null);
});

// The hidden pages are wired too: the relationship is a property of the
// document's structure, not of the current selection, so a switch moves the
// indicator and the visibility and leaves the wiring exactly where it was.
test("a switch leaves the wiring alone", () => {
    const { rt, pageAt } = mountTabs({ selectedIndex: 0 });

    rt.GrMob.patch(
        JSON.stringify([
            {
                Type: "update-props",
                TargetID: "root",
                Changes: { selectedIndex: 1, tabs: TABS, onTabChange: "cb_tab" },
            },
        ])
    );

    for (let i = 0; i < 2; i++) {
        assert.equal(pageAt(i).getAttribute("role"), "tabpanel");
        assert.equal(pageAt(i).getAttribute("aria-labelledby"), `grmob-root-tab-${i}`);
    }
});

// A page added by a patch arrives already wired, because the wiring rides on
// the same end-of-batch pass the selection does.
test("a page added by a patch is wired like the rest", () => {
    const { rt, root } = mountTabs();

    rt.GrMob.patch(
        JSON.stringify([
            {
                Type: "update-props",
                TargetID: "root",
                Changes: {
                    selectedIndex: 0,
                    tabs: [...TABS, { label: "Me" }],
                    onTabChange: "cb_tab",
                },
            },
            { Type: "add-child", TargetID: "root", Changes: page("third") },
        ])
    );

    assert.equal(root.children[3].getAttribute("role"), "tabpanel");
    assert.equal(root.children[3].getAttribute("id"), "grmob-root-panel-2");
    assert.equal(root.children[0].children[2].getAttribute("aria-controls"), "grmob-root-panel-2");
});
