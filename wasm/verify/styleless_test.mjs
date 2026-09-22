// createElement's style pass is total: it runs for every node, Style or not.
//
// styleFromGrMob answers two different questions at once. Most of what it
// writes is a reading of the Style — a font size, a background, a padding —
// and a node with no Style genuinely wants none of that. But some of what it
// writes is a reading of the node TYPE: the flex axis a Row or a Column stacks
// along, the single-cell grid a ZStack draws its layers in, the fixed rules of
// a TextGrid, and the three-valued `border` that turns the user agent's own
// 2px outset rule off for a <button>, an <input> and a <select>. Those are as
// true of a styleless node as of a styled one.
//
// The create path used to call applyStyle only for a node that carried a
// Style, and made up the difference with three special cases (a TextGrid
// branch, a stack branch, an overlay branch) that between them covered every
// type-keyed default except the border. The patch path has always called it
// unconditionally, because reconcile emits the whole new Style and there is no
// "no style" case to guard — so the two paths disagreed, and a styleless
// <button> drew the browser's border right up until something gave it a Style
// and took the border away.
//
// Nothing core builds is ever styleless (every widget reads a theme base), so
// only a hand-assembled tree reaches this. These tests are what says the two
// paths now agree.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

// A tree of styleless children under a styleless root, which is the state
// under test at both levels at once.
function mount(children) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({ Type: "Column", Children: children }));
    rt.drainFrames();
    return { rt, at: (i) => nodeAt(rt.document, `root/${i}`), root: () => nodeAt(rt.document, "root") };
}

test("a styleless form control still resets the browser's border", () => {
    // The gap this file exists for. All six of BORDER_RESET_TYPES go through
    // one expression in styleFromGrMob, so one of each tag is enough to say
    // the expression was reached; border_test.mjs holds what the expression
    // decides.
    const { at } = mount([
        { Type: "Button", Props: { label: "Skip" } },
        { Type: "Input", Props: { value: "" } },
        { Type: "Select", Props: { options: [], value: "" } },
        { Type: "TextArea", Props: { value: "" } },
    ]);
    for (let i = 0; i < 4; i++) {
        assert.equal(at(i).style.border, "none", `child ${i}`);
    }
});

test("a styleless node is drawn the same before and after a patch touches it", () => {
    // The disagreement stated directly: the create path and the update-style
    // path are the same function now, so an empty Style arriving as a patch
    // can find nothing left to change. This is the assertion that would have
    // failed on the old code — not because the patch broke anything, but
    // because it *fixed* something creation had missed.
    const { rt, at } = mount([{ Type: "Button", Props: { label: "Skip" } }]);
    const built = at(0).style.border;

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-style",
        TargetID: "root/0",
        Changes: {},
    }]));
    rt.drainFrames();

    assert.equal(built, "none");
    assert.equal(at(0).style.border, built);
});

test("a styleless stack container still stacks", () => {
    // The branch createElement used to carry. A block-flow div lets inline
    // children run together on one line, which is what a bare Column of Texts
    // looked like before the web opted into the stacking both natives get for
    // free.
    const { at, root } = mount([{ Type: "Row", Children: [] }]);

    assert.equal(root().style.display, "flex");
    assert.equal(root().style.flexDirection, "column");
    assert.equal(at(0).style.display, "flex");
    assert.equal(at(0).style.flexDirection, "row");
    // baseDisplay is what syncTabView restores a page to. A stack page cleared
    // to "" would come back as block flow, which is the case the record exists
    // for and the one a styleless container is most likely to be in.
    assert.equal(at(0).dataset.baseDisplay, "flex");
});

test("a styleless overlay still overlays", () => {
    // core.ZStack carries no theme base, so a stack written with children and
    // no style props is exactly this node — and a box that is not a grid runs
    // its layers down the page instead of over each other.
    const { at } = mount([{ Type: "ZStack", Children: [{ Type: "Text", Props: { text: "over" } }] }]);

    assert.equal(at(0).style.display, "grid");
    assert.equal(at(0).style.alignItems, "center");
    assert.equal(at(0).style.justifyItems, "center");
    assert.equal(at(0).dataset.baseDisplay, "grid");
});

test("a styleless grid still gets the grid chassis", () => {
    // The one type-keyed default that already went through applyStyle on the
    // create path, by way of a branch that called it with an empty Style. The
    // branch is gone; the chassis is not.
    const { at } = mount([{ Type: "TextGrid", Children: [] }]);

    assert.equal(at(0).style.marginBlock, "0");
    assert.equal(at(0).style.marginInline, "0");
    assert.equal(at(0).style.lineHeight, "1.2");
    assert.equal(at(0).style.overflowX, "auto");
});

// The Modal is where the totality rule meets its one exemption. Its chassis —
// the fixed inset-0 box, the centred flex column, the z-index — is a set of
// node-type defaults like the grid's, so it lives in styleFromGrMob and
// survives an update-style patch. Its `display` is not: that IS the open or
// closed state, written by the `visible` prop, and a style pass never sees a
// prop. So the pass abstains from the display and owns everything else.

test("a styleless modal keeps its chassis and its dialog semantics", () => {
    const { at } = mount([{ Type: "Modal", Props: { visible: false }, Children: [] }]);

    assert.equal(at(0).style.position, "fixed");
    assert.equal(at(0).style.display, "none");
    // String()ed on the way out: dom.mjs stores what Object.assign gave it,
    // where a browser stringifies every style value. The chassis writes an
    // integer, so the shim hands one back.
    assert.equal(String(at(0).style.zIndex), "1000");
    // The accessibility half, which used to be written twice — once in the
    // Modal branch (because applyStyle did not run for a node with no Style)
    // and once in applyAccessibility. It is written once now, and this is the
    // path that used to need the second copy.
    assert.equal(at(0).getAttribute("role"), "dialog");
    assert.equal(at(0).getAttribute("aria-modal"), "true");
});

test("a modal that carries a Style keeps the chassis it did not override", () => {
    // core's ModalNode has no Style field, so this is only reachable by hand —
    // and it used to lose the whole chassis. The chassis was assigned at
    // creation, the total style pass ran after it and cleared position,
    // centring and z-index, and nothing put them back. It was silent, too: the
    // overlay still opened and closed, it simply sat in the page's flow.
    const { at } = mount([{
        Type: "Modal",
        Props: { visible: true },
        Style: { Background: "#000000AA", Padding: { Top: 8, Bottom: 8 } },
        Children: [],
    }]);

    assert.equal(at(0).style.position, "fixed");
    assert.equal(String(at(0).style.zIndex), "1000");
    assert.equal(at(0).style.alignItems, "center");
    // safe center and overflow-y:auto together: content taller than the
    // window scrolls from its top instead of being centred past both edges.
    assert.equal(at(0).style.justifyContent, "safe center");
    assert.equal(at(0).style.overflowY, "auto");
    assert.equal(at(0).style.display, "flex");
});

test("a modal with no visible prop at all is closed", () => {
    // The state the create path has to plant for itself, and the one case the
    // prop channel cannot cover: a hand-assembled node that names no `visible`
    // has nothing to drive its display, and the style pass abstains from that
    // property on purpose. Without the initial "none" the dialog's body is
    // laid out inline in the middle of the page — the exact bug htmlout's
    // modalChassis was written for, which defaults the same way.
    const { at } = mount([{ Type: "Modal", Children: [{ Type: "Text", Props: { text: "hi" } }] }]);
    assert.equal(at(0).style.display, "none");
});

test("a modal's author still wins over the chassis, as on the other web target", () => {
    // htmlout writes modalChassis *ahead* of the author's declarations, so the
    // cascade gives the author the last word. The runtime says the same thing
    // with a `||` per line. Anything else would be a divergence between the
    // two web targets on a node nobody can build without meaning to.
    const { at } = mount([{
        Type: "Modal",
        Props: { visible: true },
        Style: { ZIndex: 5, AlignItems: "flex-start", Position: "absolute" },
        Children: [],
    }]);

    assert.equal(String(at(0).style.zIndex), "5");
    assert.equal(at(0).style.alignItems, "flex-start");
    assert.equal(at(0).style.position, "absolute");
});

test("a style patch cannot close an open modal, or open a closed one", () => {
    // The exemption, stated as the thing it protects. `display` is the whole
    // open/closed state and it arrives through the prop channel; a total pass
    // that assigned it would slam the dialog shut the next time anything
    // restyled it. Both directions, because "" would be as wrong as "none".
    const { rt, at } = mount([
        { Type: "Modal", Props: { visible: true }, Children: [] },
        { Type: "Modal", Props: { visible: false }, Children: [] },
    ]);
    assert.equal(at(0).style.display, "flex");
    assert.equal(at(1).style.display, "none");

    rt.GrMob.patch(JSON.stringify([
        { Type: "update-style", TargetID: "root/0", Changes: { Background: "#00000055" } },
        { Type: "update-style", TargetID: "root/1", Changes: { Background: "#00000055" } },
    ]));
    rt.drainFrames();

    assert.equal(at(0).style.display, "flex");
    assert.equal(at(1).style.display, "none");
});

// The patch-path twin of the rule above. A node that loses its whole Style
// sends update-style, and a Go side older than reconcile's wireStyle sent it
// as Changes null, which threw inside styleFromGrMob and abandoned the rest of
// the batch. Both spellings must land the element where a styleless create
// would, and the patch after it in the same batch must still apply.
for (const changes of [null, {}]) {
    test(`an update-style carrying ${JSON.stringify(changes)} clears the style and the batch goes on`, () => {
        const rt = loadRuntime();
        rt.GrMob.mount(JSON.stringify({
            Type: "Column",
            Children: [
                { Type: "Text", Props: { content: "a" }, Style: { FontSize: 16 } },
                { Type: "Text", Props: { content: "b" } },
            ],
        }));
        rt.GrMob.patch(JSON.stringify([
            { Type: "update-style", TargetID: "root/0", Changes: changes },
            { Type: "update-props", TargetID: "root/1", Changes: { content: "after" } },
        ]));
        assert.equal(nodeAt(rt.document, "root/0").style.fontSize, "", "the lost FontSize must be removed");
        assert.equal(nodeAt(rt.document, "root/1").textContent, "after", "the batch must not stop at the style patch");
    });
}
