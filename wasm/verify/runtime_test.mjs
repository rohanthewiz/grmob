// Unit tests for the JavaScript runtime's event and prop paths.
//
// The replay test proves patches land on the right nodes. These prove the
// per-element logic those patches trigger — the parts with no Go counterpart
// to compare against, and the parts written most recently: the Enter filter
// on onSubmit, the void payload it has to send, the keyboard hint, and the
// focus command's frame deferral and epoch guard.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

// mount renders one tree and returns the runtime plus the root's children by
// index, which is how every test below names the element it is driving.
function mount(children, rootProps = {}) {
    const rt = loadRuntime();
    rt.GrMob.mount(
        JSON.stringify({ Type: "Column", Props: rootProps, Children: children })
    );
    rt.drainFrames();
    return {
        rt,
        at: (i) => nodeAt(rt.document, `root/${i}`),
    };
}

const input = (props) => ({ Type: "Input", Props: props });

// --------------------------------------------------------------------------
// onSubmit: the return key
// --------------------------------------------------------------------------

test("Enter on a submitting field dispatches its callback", () => {
    const { rt, at } = mount([input({ value: "", onSubmit: "cb_1" })]);

    const event = at(0).dispatch("keydown", { key: "Enter", shiftKey: false });

    assert.deepEqual(rt.dispatched, [{ id: "cb_1", payload: {} }]);
    // Nothing else may also act on the keypress: inside a <form> the browser
    // would otherwise submit the page out from under the app.
    assert.equal(event.defaultPrevented, true);
});

test("the submit payload is void, not the field's text", () => {
    // Go registered onSubmit through the plain callback channel. A {value}
    // envelope would route it into the *text* callback map, where a void ID
    // does not exist, and the handler would silently never run — the same
    // trap focus and blur already carry a comment about.
    const { rt, at } = mount([input({ value: "typed", onSubmit: "cb_1" })]);

    at(0).dispatch("keydown", { key: "Enter" });

    assert.deepEqual(rt.dispatched[0].payload, {});
});

test("keys other than Enter do not submit", () => {
    const { rt, at } = mount([input({ value: "", onSubmit: "cb_1" })]);

    at(0).dispatch("keydown", { key: "a" });
    at(0).dispatch("keydown", { key: "Tab" });
    at(0).dispatch("keydown", { key: "Escape" });

    assert.deepEqual(rt.dispatched, []);
});

test("Shift+Enter does not submit", () => {
    // So a textarea keeps its "newline without submitting" convention — the
    // same split the native renderers get for free, where a multiline field
    // is given a newline key rather than an action key.
    const { rt, at } = mount([
        { Type: "TextArea", Props: { value: "", onSubmit: "cb_1" } },
    ]);

    const event = at(0).dispatch("keydown", { key: "Enter", shiftKey: true });

    assert.deepEqual(rt.dispatched, []);
    assert.equal(event.defaultPrevented, false);
});

test("a field with no onSubmit has no keydown listener at all", () => {
    const { at } = mount([input({ value: "", onChange: "txt_cb_0" })]);

    assert.equal(at(0).listeners.has("keydown"), false);
});

// --------------------------------------------------------------------------
// The other event payloads, which the submit filter sits alongside
// --------------------------------------------------------------------------

test("focus and blur send void payloads; input sends its value", () => {
    const { rt, at } = mount([
        input({ value: "abc", onChange: "txt_cb_0", onFocus: "cb_1", onBlur: "cb_2" }),
    ]);

    at(0).dispatch("focus");
    at(0).dispatch("blur");
    at(0).value = "abcd";
    at(0).dispatch("input", { target: at(0) });

    assert.deepEqual(rt.dispatched, [
        { id: "cb_1", payload: {} },
        { id: "cb_2", payload: {} },
        { id: "txt_cb_0", payload: { value: "abcd" } },
    ]);
});

// --------------------------------------------------------------------------
// enterkeyhint
// --------------------------------------------------------------------------

test("the keyboard hint reflects what the return key does", () => {
    const { at } = mount([
        input({ value: "", imeAction: "next", onSubmit: "cb_0" }),
        input({ value: "", imeAction: "", onSubmit: "cb_1" }),
        input({ value: "", imeAction: "" }),
        input({ value: "" }),
    ]);

    assert.equal(at(0).getAttribute("enterkeyhint"), "next");
    assert.equal(at(1).getAttribute("enterkeyhint"), "done");
    // No submit action at all: the attribute must be absent, not empty.
    // enterkeyhint has no empty value, and its absence is how HTML spells
    // "the browser's default return key".
    assert.equal(at(2).getAttribute("enterkeyhint"), null);
    assert.equal(at(3).getAttribute("enterkeyhint"), null);
});

test("a field that loses its submit action loses the hint", () => {
    // An update-props patch carries the whole new props map, so the hint has
    // to be recomputed from it — including down to nothing.
    const { rt, at } = mount([input({ value: "", imeAction: "next", onSubmit: "cb_0" })]);
    assert.equal(at(0).getAttribute("enterkeyhint"), "next");

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-props", TargetID: "root/0", Changes: { value: "", imeAction: "" } },
        ])
    );

    assert.equal(at(0).getAttribute("enterkeyhint"), null);
});

test("a field promoted from done to next relabels its key", () => {
    const { rt, at } = mount([input({ value: "", imeAction: "", onSubmit: "cb_0" })]);
    assert.equal(at(0).getAttribute("enterkeyhint"), "done");

    rt.GrMob.patch(
        JSON.stringify([
            {
                Type: "update-props",
                TargetID: "root/0",
                Changes: { value: "", imeAction: "next", onSubmit: "cb_0" },
            },
        ])
    );

    assert.equal(at(0).getAttribute("enterkeyhint"), "next");
});

// --------------------------------------------------------------------------
// Focus commands
// --------------------------------------------------------------------------

test("focus is deferred one frame", () => {
    // On the initial render createElement builds the tree detached — mount
    // appends it only once assembled — and focus() on a node outside the
    // document is a silent no-op. The deferral is what makes a command that
    // arrives with the mount work at all.
    const rt = loadRuntime();
    rt.GrMob.mount(
        JSON.stringify({
            Type: "Column",
            Props: {},
            Children: [input({ value: "", focusEpoch: 1, focusAction: "focus" })],
        })
    );

    assert.equal(rt.document.activeElement, null, "focus was applied inline");
    assert.equal(rt.pendingFrames(), 1);

    rt.drainFrames();
    assert.equal(rt.document.activeElement, nodeAt(rt.document, "root/0"));
});

test("epoch 0 is not a command", () => {
    // Zero is the sentinel for "nothing has ever been issued". Both props
    // always travel together, so a 0 must never be read as an instruction.
    const rt = loadRuntime();
    rt.GrMob.mount(
        JSON.stringify({
            Type: "Column",
            Props: {},
            Children: [input({ value: "", focusEpoch: 0, focusAction: "focus" })],
        })
    );
    rt.drainFrames();

    assert.equal(rt.document.activeElement, null);
});

test("blur only releases the field that holds focus", () => {
    // A dismiss reaches every focusable leaf, and exactly one of them is the
    // one to act. The guard is what stops the others from clearing a focus
    // that has already moved on.
    const { rt, at } = mount([input({ value: "" }), input({ value: "" })]);
    at(1).focus();

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-props", TargetID: "root/0", Changes: { value: "", focusEpoch: 5, focusAction: "blur" } },
        ])
    );
    rt.drainFrames();
    assert.equal(rt.document.activeElement, at(1), "an unrelated field cleared the focus");

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-props", TargetID: "root/1", Changes: { value: "", focusEpoch: 5, focusAction: "blur" } },
        ])
    );
    rt.drainFrames();
    assert.equal(rt.document.activeElement, null);
});

test("the empty action does nothing", () => {
    // Every non-target leaf is told "" on a focus command and must stay put:
    // requesting focus over there already takes it from here, and having both
    // sides act would make the outcome depend on ordering.
    const { rt, at } = mount([input({ value: "" }), input({ value: "" })]);
    at(1).focus();

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-props", TargetID: "root/0", Changes: { value: "", focusEpoch: 5, focusAction: "" } },
        ])
    );
    rt.drainFrames();

    assert.equal(rt.document.activeElement, at(1));
});

test("a repeated epoch does not re-run the command", () => {
    // The epoch is the whole trigger; focusAction is only read once it has
    // moved. An update-props patch carries the entire new props map, so a
    // field re-rendered for its *value* would otherwise re-run whatever
    // command was last issued — stealing focus back on every keystroke
    // elsewhere on the screen.
    const { rt, at } = mount([input({ value: "", focusEpoch: 3, focusAction: "focus" })]);
    assert.equal(rt.document.activeElement, at(0));

    at(0).blur();
    rt.GrMob.patch(
        JSON.stringify([
            {
                Type: "update-props",
                TargetID: "root/0",
                Changes: { value: "typed", focusEpoch: 3, focusAction: "focus" },
            },
        ])
    );
    rt.drainFrames();

    assert.equal(rt.document.activeElement, null, "the stale stamp re-focused the field");
    assert.equal(at(0).value, "typed", "the value in the same patch was dropped");
});

test("a new epoch does re-run the command", () => {
    // The counterpart: focusing the already-focused field has to re-fire — a
    // "try again after a failed submit" — and only a changed value can say so.
    const { rt, at } = mount([input({ value: "", focusEpoch: 3, focusAction: "focus" })]);
    at(0).blur();

    rt.GrMob.patch(
        JSON.stringify([
            {
                Type: "update-props",
                TargetID: "root/0",
                Changes: { value: "", focusEpoch: 4, focusAction: "focus" },
            },
        ])
    );
    rt.drainFrames();

    assert.equal(rt.document.activeElement, at(0));
});

test("a bare focusAction says what but not when, and is ignored", () => {
    const { rt, at } = mount([input({ value: "" })]);

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-props", TargetID: "root/0", Changes: { value: "", focusAction: "focus" } },
        ])
    );
    rt.drainFrames();

    assert.equal(rt.document.activeElement, null);
});

// --------------------------------------------------------------------------
// Listener bookkeeping, which every event path above depends on
// --------------------------------------------------------------------------

test("a changed callback ID reuses the one listener", () => {
    // IDs are per-pass sequence numbers, so they change constantly. The
    // element keeps one listener and reads the current ID out of its dataset
    // at dispatch time; attaching a second would fire the handler twice.
    const { rt, at } = mount([{ Type: "Button", Props: { label: "go", onClick: "cb_0" } }]);

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-props", TargetID: "root/0", Changes: { label: "go", onClick: "cb_9" } },
        ])
    );
    at(0).dispatch("click");

    assert.equal(at(0).listeners.get("click").length, 1, "a second listener was attached");
    assert.deepEqual(rt.dispatched, [{ id: "cb_9", payload: {} }]);
});

// --------------------------------------------------------------------------
// Style, the other thing a patch can put on the wrong element
// --------------------------------------------------------------------------

test("a disabled form control is disabled, not merely aria-disabled", () => {
    // The element *property* is what makes the browser refuse to dispatch the
    // events whose callback IDs are wired alongside it: the Go handler stays
    // registered and the platform, not the app, declines to call it.
    const { rt, at } = mount([input({ value: "", onChange: "txt_cb_0" }), { Type: "Row", Props: {} }]);

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-style", TargetID: "root/0", Changes: { Disabled: true } },
            { Type: "update-style", TargetID: "root/1", Changes: { Disabled: true } },
        ])
    );

    assert.equal(at(0).disabled, true);
    assert.equal(at(0).getAttribute("aria-disabled"), null);
    // A div cannot be disabled, so the state is announced instead.
    assert.equal(at(1).getAttribute("aria-disabled"), "true");
});

test("update-style keeps the node type it needs for the flex axis", () => {
    // An update-style patch carries only the changed Style, not the type —
    // and the Row/everything-else split decides the flex axis. The type was
    // recorded on the element at creation for exactly this.
    const { rt, at } = mount([{ Type: "Row", Props: {} }]);

    rt.GrMob.patch(
        JSON.stringify([{ Type: "update-style", TargetID: "root/0", Changes: { Gap: 8 } }])
    );

    assert.equal(at(0).dataset.nodeType, "Row");
    assert.equal(at(0).style.flexDirection, "row");
});

// --------------------------------------------------------------------------
// The event vocabulary: Go node types, on both listener paths
// --------------------------------------------------------------------------

test("a checkbox reports its checked state, not its value", () => {
    // tagForType collapses Input, InputPassword, NumericInput and Checkbox
    // all onto <input>, so the HTML tag cannot tell a text field from a
    // checkbox. Only the Go node type can, which is why the element carries
    // it and why extractEventPayload is asked in that vocabulary.
    const { rt, at } = mount([{ Type: "Checkbox", Props: { checked: false, onToggle: "bool_cb_0" } }]);

    at(0).checked = true;
    at(0).dispatch("change", { target: at(0) });

    assert.deepEqual(rt.dispatched, [{ id: "bool_cb_0", payload: { value: true } }]);
});

test("both listener paths build the same envelope", () => {
    // The two call sites drifted apart once — one passed the Go node type,
    // the other the HTML tag, and the lookup only matched the tag. A field
    // present at the initial render then sent {} for every keystroke, Go
    // routed the void envelope to the void callback map where a txt_ ID does
    // not exist, and typing did nothing at all. This is the guard.
    const { rt, at } = mount([input({ value: "", onChange: "txt_cb_0" })]);

    // Path one: the listener createElement attached at mount.
    at(0).value = "typed at mount";
    at(0).dispatch("input", { target: at(0) });

    // Path two: the listener an update-props patch attaches to a field that
    // had no onChange before.
    rt.GrMob.patch(
        JSON.stringify([
            { Type: "add-child", TargetID: "root", Changes: { Type: "Input", Props: { value: "" } } },
        ])
    );
    const added = nodeAt(rt.document, "root/1");
    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-props", TargetID: "root/1", Changes: { value: "", onChange: "txt_cb_1" } },
        ])
    );
    added.value = "typed after a patch";
    added.dispatch("input", { target: added });

    assert.deepEqual(rt.dispatched, [
        { id: "txt_cb_0", payload: { value: "typed at mount" } },
        { id: "txt_cb_1", payload: { value: "typed after a patch" } },
    ]);
});

test("a listener attached by a patch also speaks in Go node types", () => {
    // The two listener sites are separate code paths, and only one of them
    // has the Go node in hand — the other has to read the type back off the
    // element. A Checkbox is the case that tells them apart: its tag is
    // <input>, so a path that asked the tag would fetch .value from a control
    // whose state lives in .checked.
    const { rt } = mount([]);

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "add-child", TargetID: "root", Changes: { Type: "Checkbox", Props: { checked: false } } },
        ])
    );
    const box = nodeAt(rt.document, "root/0");
    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-props", TargetID: "root/0", Changes: { checked: false, onToggle: "bool_cb_0" } },
        ])
    );

    box.checked = true;
    box.dispatch("change", { target: box });

    assert.deepEqual(rt.dispatched, [{ id: "bool_cb_0", payload: { value: true } }]);
});

test("a listener attached by a patch follows the ID onward", () => {
    // The listener is attached once and reads the *current* ID out of the
    // element's dataset each time it fires. Capturing the ID in the closure
    // instead would work exactly once — until the next render renumbered it,
    // after which every event would name a callback Go had already purged.
    const { rt, at } = mount([{ Type: "Button", Props: { label: "go" } }]);

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-props", TargetID: "root/0", Changes: { label: "go", onClick: "cb_0" } },
        ])
    );
    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-props", TargetID: "root/0", Changes: { label: "go", onClick: "cb_7" } },
        ])
    );
    at(0).dispatch("click");

    assert.equal(at(0).listeners.get("click").length, 1);
    assert.deepEqual(rt.dispatched, [{ id: "cb_7", payload: {} }]);
});

// --------------------------------------------------------------------------
// The structural patches, which decide where every later patch lands
// --------------------------------------------------------------------------

test("replace keeps the node path and re-paths the subtree", () => {
    // A replaced node inherits the path of the node it replaced — the paths
    // are positional, so anything else strands every patch aimed at that
    // subtree for the rest of the session.
    const { rt, at } = mount([{ Type: "Row", Props: {} }]);

    rt.GrMob.patch(
        JSON.stringify([
            {
                Type: "replace",
                TargetID: "root/0",
                Changes: {
                    Type: "Column",
                    Props: {},
                    Children: [{ Type: "Text", Props: { content: "new" } }],
                },
            },
        ])
    );

    assert.equal(at(0).dataset.nodeType, "Column");
    assert.equal(nodeAt(rt.document, "root/0/0").textContent, "new");
    // And the replacement is in the tree exactly once, in the slot the old
    // node held.
    assert.equal(rt.mountPoint.children[0].children.length, 1);
});

test("remove takes the node it names and nothing else", () => {
    const { rt } = mount([
        { Type: "Text", Props: { content: "a" } },
        { Type: "Text", Props: { content: "b" } },
    ]);

    rt.GrMob.patch(JSON.stringify([{ Type: "remove", TargetID: "root/0" }]));

    const root = rt.mountPoint.children[0];
    assert.equal(root.children.length, 1);
    assert.equal(root.children[0].textContent, "b");
});

test("mount clears whatever was in its root", () => {
    // mount is how a fresh tree arrives, so anything already under the mount
    // point is the previous one. Appending instead of clearing would leave
    // two trees on the page, both matching the same node paths — and
    // querySelector would then start handing patches to the dead one.
    const rt = loadRuntime();
    const stale = rt.document.createElement("div");
    stale.setAttribute("data-node-path", "root");
    rt.mountPoint.appendChild(stale);

    rt.GrMob.mount(JSON.stringify({ Type: "Column", Props: {}, Children: [] }));

    assert.equal(rt.mountPoint.children.length, 1);
    assert.notEqual(rt.mountPoint.children[0], stale);
});

// --------------------------------------------------------------------------
// Checkbox state and TextArea rows: the two props that reached no DOM
// property at all until the harness existed to notice
// --------------------------------------------------------------------------

test("each <input> variant carries the type that decides what is drawn", () => {
    // tagForType sends four Go node types to <input>, and an <input> with no
    // type attribute is a text box. A Checkbox therefore drew as a text field
    // — not a wrong state, an entirely wrong control.
    const { at } = mount([
        input({ value: "" }),
        { Type: "InputPassword", Props: { value: "" } },
        { Type: "NumericInput", Props: { value: "0" } },
        { Type: "Checkbox", Props: { checked: false } },
        { Type: "TextArea", Props: { value: "", rows: 3 } },
        { Type: "Text", Props: { content: "not a control" } },
    ]);

    assert.equal(at(0).getAttribute("type"), "text");
    assert.equal(at(1).getAttribute("type"), "password");
    assert.equal(at(2).getAttribute("type"), "number");
    assert.equal(at(3).getAttribute("type"), "checkbox");
    // A tag that already says what it is gets no discriminator, matching what
    // htmlout emits.
    assert.equal(at(4).tagName, "TEXTAREA");
    assert.equal(at(4).getAttribute("type"), null);
    assert.equal(at(5).getAttribute("type"), null);
});

test("a checkbox renders the state Go gave it", () => {
    const { at } = mount([
        { Type: "Checkbox", Props: { checked: true, onToggle: "bool_cb_0" } },
        { Type: "Checkbox", Props: { checked: false, onToggle: "bool_cb_1" } },
    ]);

    // The live property, not the attribute: the attribute is only a default
    // the browser stops consulting once the user touches the box.
    assert.equal(at(0).checked, true);
    assert.equal(at(1).checked, false);
    assert.equal(at(0).hasAttribute("checked"), false);
});

test("a checkbox's state follows an update-props patch both ways", () => {
    // The path that actually carries a toggle: Go owns the state, so the tick
    // the user sees is the one that came back through a patch, not the one
    // the browser drew on click.
    const { rt, at } = mount([
        { Type: "Checkbox", Props: { checked: false, onToggle: "bool_cb_0" } },
    ]);

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-props", TargetID: "root/0", Changes: { checked: true, onToggle: "bool_cb_0" } },
        ])
    );
    assert.equal(at(0).checked, true);

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-props", TargetID: "root/0", Changes: { checked: false, onToggle: "bool_cb_0" } },
        ])
    );
    assert.equal(at(0).checked, false, "unticking never reached the DOM");
});

test("a text area's height is the rows Go asked for", () => {
    const { rt, at } = mount([
        { Type: "TextArea", Props: { value: "", rows: 5, onChange: "txt_cb_0" } },
    ]);
    assert.equal(at(0).rows, 5);

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-props", TargetID: "root/0", Changes: { value: "", rows: 2, onChange: "txt_cb_0" } },
        ])
    );
    assert.equal(at(0).rows, 2);
});

test("a non-positive rows leaves the browser's own default", () => {
    // rows is limited to positive numbers in the DOM — assigning 0 is an
    // error, not a request for a zero-line box. core.TextArea always supplies
    // a positive count, so this only covers a hand-built core.Node.
    const { at } = mount([
        { Type: "TextArea", Props: { value: "", rows: 0 } },
        { Type: "TextArea", Props: { value: "" } },
    ]);

    assert.equal(at(0).rows, undefined);
    assert.equal(at(1).rows, undefined);
});

// --------------------------------------------------------------------------
// Stale listeners: callback IDs are positional and get reused
// --------------------------------------------------------------------------

test("a node that loses a handler prop stops firing the ID it used to hold", () => {
    // Go re-derives callback IDs from a per-pass counter (core/event.go), so
    // "cb_0" means "the first registration of this pass" and belongs to a
    // different node the moment the tree's handler set changes. The update
    // loop only walked the keys present in Changes, so a dropped onClick left
    // its old ID sitting in the dataset — and the next pass gave that ID to
    // whichever node now registered first.
    const { rt, at } = mount([
        { Type: "Box", Props: { onClick: "cb_0" }, Children: [] },
        { Type: "Box", Props: {}, Children: [] },
    ]);

    at(0).dispatch("click");
    assert.deepEqual(rt.dispatched, [{ id: "cb_0", payload: {} }]);

    // Pass 2: the first box drops its handler, the second picks one up — and
    // because IDs are positional, the second one is now "cb_0".
    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-props", TargetID: "root/0", Changes: {} },
            { Type: "update-props", TargetID: "root/1", Changes: { onClick: "cb_0" } },
        ])
    );

    rt.dispatched.length = 0;
    at(0).dispatch("click");
    assert.deepEqual(
        rt.dispatched,
        [],
        "clicking the node that lost its handler must dispatch nothing"
    );

    at(1).dispatch("click");
    assert.deepEqual(rt.dispatched, [{ id: "cb_0", payload: {} }]);
});

test("a handler prop that comes back is wired again without stacking listeners", () => {
    // Pruning drops the ID but keeps the attached listener, so a prop that
    // returns must dispatch exactly once — not twice.
    const { rt, at } = mount([{ Type: "Box", Props: { onClick: "cb_0" }, Children: [] }]);

    rt.GrMob.patch(
        JSON.stringify([{ Type: "update-props", TargetID: "root/0", Changes: {} }])
    );
    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-props", TargetID: "root/0", Changes: { onClick: "cb_7" } },
        ])
    );

    rt.dispatched.length = 0;
    at(0).dispatch("click");
    assert.deepEqual(rt.dispatched, [{ id: "cb_7", payload: {} }]);
});

test("a field that loses onSubmit outright loses both the listener and the hint", () => {
    // The narrow version of this (imeAction still present in Changes) was
    // already covered; this is the case the old `"onSubmit" in p.Changes`
    // gate missed entirely — neither key survives into the new props map, so
    // nothing re-ran and the field kept advertising a submit it no longer had.
    const { rt, at } = mount([input({ value: "", onSubmit: "cb_0", imeAction: "next" })]);
    assert.equal(at(0).getAttribute("enterkeyhint"), "next");

    rt.GrMob.patch(
        JSON.stringify([
            { Type: "update-props", TargetID: "root/0", Changes: { value: "" } },
        ])
    );

    assert.equal(at(0).getAttribute("enterkeyhint"), null);

    rt.dispatched.length = 0;
    at(0).dispatch("keydown", { key: "Enter" });
    assert.deepEqual(rt.dispatched, [], "a field with no onSubmit must not submit");
});
