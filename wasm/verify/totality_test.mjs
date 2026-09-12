// styleFromGrMob's totality rule, and the one exemption from it.
//
// The rule: every CSS property the function manages is assigned on every call
// — a real value, or "", which removes the inline declaration. The wire
// contract forces it. An update-style patch carries the WHOLE new Style
// (reconcile/patch.go), so a zero field means "unset now" and not
// "unmentioned"; and because the patch path reuses the live element, a guarded
// `if (style.X)` leaves the old declaration standing whenever a field returns
// to zero. core.BorderRadius(0) is the canonical victim — the corners stayed
// rounded because nothing ever cleared them.
//
// The exemption: a Modal's `display` is deleted from the result rather than
// assigned, because that property IS the dialog's open/closed state and the
// `visible` prop owns it. Assigning anything would close an open modal on the
// next update-style patch; assigning "" would open a closed one.
//
// # What this file is for
//
// The exemption was pinned by a test that asserted the line exists. What was
// never written down is the test a *second* exemption has to pass — and
// "abstain by deleting the key" is now a technique available to any property,
// which makes the next one likely rather than hypothetical.
//
// So the exemptions are a table here, and three things are checked against it:
// that the source makes exactly the deletions the table names, that each
// exemption actually abstains, and that the prop said to own the property
// writes it in every state. The last is the condition that is easy to miss and
// the one that decides whether an exemption is safe at all: a prop that
// assigns only when it is truthy leaves the value it wrote last standing
// forever, which is the failure totality exists to prevent, moved one channel
// over rather than fixed.
//
// The source scan is load-bearing rather than decorative, and the sweep below
// says why it has to be there. An abstention deletes the key *before* the
// declarations reach the element, so a node built with the property in its
// Style never receives it either — which makes an exemption invisible to any
// test that does not drive the prop that owns it. Reading the source is what
// turns "somebody added a delete" into a failing change; the table is what
// says which deletes are answers rather than accidents.
//
// This is the same shape as knownBoundaryShortfalls' entry-shape test,
// rowsSpec's admission test and TestEverySharedKnobHasItsOwnEffectAssertion:
// what a future addition must look like, written before it arrives.

import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

import { loadRuntime, nodeAt } from "./load.mjs";
import { erasedShorthands } from "./cssstyle.mjs";

const RUNTIME = new URL("../grmob-runtime.js", import.meta.url);

/**
 * Every property styleFromGrMob abstains from, and what owns it instead.
 *
 * - `nodeType` / `property`: the exemption itself. It is keyed on the node
 *   type, so the property stays total for every other node — the third
 *   condition, and the one the last test below holds.
 * - `owner`: the prop that writes the property. Not a Style field: a Style
 *   field would have been assigned by the function in the first place.
 * - `states`: every value the owner can be given, with the CSS it must write.
 *   The list has to be exhaustive, because that is the claim being made —
 *   there is no state in which the property goes unwritten.
 */
const EXEMPTIONS = [
    {
        nodeType: "Modal",
        property: "display",
        owner: "visible",
        // flex, not block: the overlay centres its content.
        states: [[true, "flex"], [false, "none"]],
    },
];

// A Style touching every property the mapping manages, used to prove the pass
// clears what it set. The values are arbitrary; only "not the default" matters.
//
// Overflow was missing, and its absence is how a TextGrid's Overflow went
// half-erased with every sweep here green: the grid chassis followed the
// `overflow` shorthand with an overflowX of "" whenever the author set one.
// The list is now held to styleFromGrMob's source in both directions by
// "FULL_STYLE is every Style field the mapping reads" below. Its first run found
// nine more fields missing (Display to WhiteSpace on the last three lines) and
// two present that nothing reads: Opacity and ObjectFit, which are not
// core.Style fields at all, so the sweep had been exercising neither.
//
// Display is "hidden" rather than "none": "none" replaces the flex display
// every stack container is given, so a stale display:flex would pass under it
// unseen, while "hidden" leaves the display alone and exercises visibility,
// the property this field otherwise never reaches. FlexDirection is "row" for
// the same kind of reason — a value that differs from every node type's own
// axis, so a direction left standing is a visible difference.
const FULL_STYLE = {
    Overflow: "hidden",
    FontSize: 14, FontWeight: 700, TextColor: "#111111", Align: "center",
    Background: "#eeeeee",
    Padding: { Top: 1, Right: 2, Bottom: 3, Left: 4 },
    Margin: { Top: 1, Right: 2, Bottom: 3, Left: 4 },
    BorderRadius: 6, Rotate: 10, Shadow: 4, LineHeight: 20,
    Width: "10px", Height: "11px",
    Gap: 5, JustifyContent: "center", AlignItems: "center",
    FlexGrow: 1, BorderWidth: 1, BorderColor: "#000000",
    Position: "absolute", Top: "1px", Right: "2px", Bottom: "3px", Left: "4px",
    ZIndex: 3, FlexWrap: "wrap", AlignSelf: "center", FlexBasis: "5px",
    FlexShrink: 2, RowGap: 2, ColumnGap: 3,
    Display: "hidden", FlexDirection: "row",
    Transition: "200ms ease", Animation: "pulse 2s infinite",
    MinWidth: "1px", MinHeight: "2px", MaxWidth: "300px", MaxHeight: "400px", WhiteSpace: "nowrap",
};

// Every node type the runtime draws.
//
// Spacer used to be left out, and the exclusion is worth recording because of
// how it ended. Its size is a prop; applySpacerSize wrote width, height and
// flex-shrink from it; and renderNode called that *after* createElement — so a
// Spacer's own Style lost those three properties to a prop that was not exempt
// from anything. That is a write ordering rather than an abstention, so
// reporting it here would have been reporting it where nobody could act on it.
//
// It is fixed: applySpacerChassis writes each of the three only where the
// style pass left the property empty, which is the same "author wins" rule the
// Modal chassis states one file over. So the type belongs in this sweep like
// any other — the sweep mounts it without a size prop, where the chassis is
// inert, and the chassis's own behaviour is checked by name in
// runtime_test.mjs.
//
// CodeEditor was left out too, and its gutter cleared an author's left
// padding on exactly the path this sweep exists for: syncCodeGutter runs after
// the style pass and wrote "" to padding-left whenever line numbers were off.
// It is mounted here with no props, which is line numbers off.
const NODE_TYPES = [
    "Box", "Column", "Row", "Card", "Scroll", "SafeArea", "List", "ZStack",
    "Text", "Button", "Input", "InputPassword", "NumericInput", "Checkbox",
    "Switch", "Slider", "Select", "TextArea", "TextGrid", "GridRow", "CodeEditor",
    "RichTextEditor", "Modal", "TabView", "Image", "CameraView", "MapView",
    "Marker", "Fragment", "Theme", "Spacer",
];

function mountOne(type, style, props = {}) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "Column",
        Children: [{ Type: type, Style: style, Props: props }],
    }));
    rt.drainFrames();
    return { rt, el: nodeAt(rt.document, "root/0") };
}

function patchStyle(rt, changes) {
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-style", TargetID: "root/0", Changes: changes,
    }]));
    rt.drainFrames();
}

test("the source deletes exactly the properties the table names", () => {
    // `out` is styleFromGrMob's local name for the declaration object it
    // builds, so this pattern is the abstention and nothing else — the
    // runtime's other five `delete`s all address a dataset entry on a live
    // element.
    //
    // This is what makes the table binding rather than descriptive: a new
    // `delete out.something` fails here until a row explains it, and a row
    // whose deletion has been removed fails too, so the table cannot outlive
    // what it describes.
    const src = readFileSync(RUNTIME, "utf8");
    const deleted = [...src.matchAll(/delete\s+out\.(\w+)\s*;/g)].map((m) => m[1]);
    const declared = EXEMPTIONS.map((e) => e.property);

    assert.deepEqual(
        [...deleted].sort(),
        [...declared].sort(),
        "styleFromGrMob abstains from a property EXEMPTIONS does not name, or names " +
        "one it no longer abstains from — see the rule at the delete site",
    );
});

test("FULL_STYLE is every Style field the mapping reads", () => {
    // The sweeps below are only as total as the Style they drive: a field
    // FULL_STYLE never sets is a field whose clearing and whose shorthand
    // interactions go untested, with every sweep green. That is how the
    // TextGrid's Overflow bug hid. So the list is held to the source.
    //
    // Both directions. A field the mapping reads and the list lacks is the
    // coverage gap; a key the list has and nothing reads is worse than
    // useless, because it reads as coverage — Opacity and ObjectFit sat here
    // for a long time, and neither is a core.Style field.
    //
    // The function's extent is its declaration to the first line that closes a
    // block at its own four-space indent, and comments are stripped first so a
    // `style.X` mentioned in prose (there are several) is not counted as a read.
    const src = readFileSync(RUNTIME, "utf8");
    const start = src.indexOf("function styleFromGrMob(");
    const end = src.indexOf("\n    }\n", start);
    assert.ok(start >= 0 && end > start, "styleFromGrMob was not found where this scan looks for it");
    const code = src.slice(start, end).split("\n").map((l) => l.replace(/\/\/.*$/, "")).join("\n");
    const read = new Set([...code.matchAll(/\bstyle\.([A-Z]\w*)/g)].map((m) => m[1]));

    // A floor, so a scan broken into finding nothing cannot pass by agreeing
    // with an empty list. The mapping reads over forty fields today.
    assert.ok(read.size >= 30, `the scan found only ${read.size} Style reads in styleFromGrMob`);

    const missing = [...read].filter((f) => !Object.hasOwn(FULL_STYLE, f)).sort();
    assert.deepEqual(missing, [],
        "styleFromGrMob reads Style fields FULL_STYLE does not set, so no sweep here " +
        "checks that they are cleared or that they erase nothing — add each with a " +
        "value that differs from every default");

    const unread = Object.keys(FULL_STYLE).filter((f) => !read.has(f)).sort();
    assert.deepEqual(unread, [],
        "FULL_STYLE sets fields styleFromGrMob never reads — they look like coverage " +
        "and are not; remove them, or find what renamed them");
});

test("after an empty style patch every node type is a freshly built styleless one", () => {
    // Totality, stated as the property that actually matters: whatever a Style
    // put on an element, sending an empty Style takes it all back off, leaving
    // only the type-keyed defaults a node with no Style would have had anyway
    // (the flex axis, the border reset, the overlay grid).
    //
    // This is the *guarded write* half of the rule — `if (style.X) out.X = …`,
    // which leaves the old declaration standing when a field returns to zero.
    // It is deliberately blind to an exemption: a deleted key never reaches
    // the element on either mount, so an abstention looks exactly like a
    // property that was never asked for. Exemptions are held by the source
    // scan above and by the three tests below, which drive the owning prop.
    //
    // No props are set here, so an exempted property has nothing to hold and
    // both mounts agree on it, which is what makes the sweep total over every
    // node type rather than needing a prop table of its own.
    for (const type of NODE_TYPES) {
        const styleless = mountOne(type, null).el;
        const { rt, el } = mountOne(type, FULL_STYLE);
        patchStyle(rt, {});

        const keys = new Set([...Object.keys(styleless.style), ...Object.keys(el.style)]);
        for (const key of keys) {
            assert.equal(
                el.style[key] ?? "", styleless.style[key] ?? "",
                `${type}: ${key} survived an empty Style patch — it is written ` +
                `conditionally somewhere, so a stale declaration stands whenever the ` +
                `field returns to zero (core.BorderRadius(0) is the canonical victim)`,
            );
        }
    }
});

test("no shorthand a Style sets is erased by a declaration after it", () => {
    // Totality's other failure, and the one the sweep above cannot see. That
    // sweep asks whether "" clears what a Style set; this asks whether "" clears
    // what the SAME pass just set. A total mapping writes "" for every unset
    // property, so assigning a shorthand and then a "" longhand of it in one
    // object removes the shorthand's declaration on that axis — `gap` then
    // rowGap/columnGap erased every core.Gap, and `overflow` then a "" overflowX
    // erased half of a TextGrid's Overflow.
    //
    // It needs dom.mjs's style to expand shorthands (cssstyle.mjs), and every
    // path a Style reaches an element by: creation, and an update-style patch
    // from a styled and from a styleless node.
    for (const type of NODE_TYPES) {
        const why = (when) => `${type} ${when}: a shorthand was assigned and then a longhand of it ` +
            `was cleared in the same pass — write one authority per property (see how Gap resolves ` +
            `into rowGap/columnGap in styleFromGrMob)`;

        const styled = mountOne(type, FULL_STYLE);
        assert.deepEqual(erasedShorthands(styled.el.style), [], why("on creation"));
        patchStyle(styled.rt, FULL_STYLE);
        assert.deepEqual(erasedShorthands(styled.el.style), [], why("after a restyling patch"));

        const bare = mountOne(type, null);
        patchStyle(bare.rt, FULL_STYLE);
        assert.deepEqual(erasedShorthands(bare.el.style), [], why("after a patch that styles it"));
    }
});

for (const e of EXEMPTIONS) {
    test(`${e.nodeType}: the style pass abstains from ${e.property}`, () => {
        // The abstention itself. Whatever the prop wrote has to survive a
        // patch that rewrites the whole Style — the case that opens a dialog
        // and then restyles anything inside it.
        for (const [value, css] of e.states) {
            const { rt, el } = mountOne(e.nodeType, FULL_STYLE, { [e.owner]: value });
            assert.equal(el.style[e.property], css,
                `${e.owner}=${value} did not write ${e.property}`);

            patchStyle(rt, FULL_STYLE);
            assert.equal(el.style[e.property], css,
                `a style patch overwrote ${e.property}, which ${e.owner} owns`);

            patchStyle(rt, {});
            assert.equal(el.style[e.property], css,
                `an empty style patch cleared ${e.property}, which ${e.owner} owns`);
        }
    });

    test(`${e.nodeType}: ${e.owner} writes ${e.property} in every state`, () => {
        // The condition an exemption is only safe under, and the one a
        // candidate is most likely to fail.
        //
        // Abstaining hands a property to a prop. If that prop assigns only in
        // its interesting state — sets "flex" when open and says nothing when
        // closed — then the value it wrote last stands forever, and the bug is
        // the same stale-declaration bug totality exists to prevent, moved one
        // channel over. So every state is driven on ONE element, in sequence,
        // which is what a live dialog does and what a fresh mount per state
        // would not catch.
        const { rt, el } = mountOne(e.nodeType, FULL_STYLE, { [e.owner]: e.states[0][0] });
        for (const [value, css] of e.states) {
            rt.GrMob.patch(JSON.stringify([{
                Type: "update-props", TargetID: "root/0", Changes: { [e.owner]: value },
            }]));
            rt.drainFrames();
            assert.equal(el.style[e.property], css,
                `${e.owner}=${value} left ${e.property} at ${JSON.stringify(el.style[e.property])} ` +
                `— the owner does not write this state, so the previous value stands`);
        }
    });

    test(`${e.property} is still total for a node that is not a ${e.nodeType}`, () => {
        // The exemption is keyed on the node type; the property is managed as
        // normally as any other everywhere else. A Box is the plainest node
        // there is, so if the abstention had leaked out of its `if` this is
        // where it would show.
        const { rt, el } = mountOne("Box", { ...FULL_STYLE, Display: "none" });
        const built = el.style[e.property];
        patchStyle(rt, {});
        assert.equal(el.style[e.property], mountOne("Box", null).el.style[e.property],
            `a Box's ${e.property} (${JSON.stringify(built)}) survived an empty Style patch`);
    });
}
