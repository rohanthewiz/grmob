// core.CodeEditor through the real runtime.
//
// The replay suite cannot reach any of this: no example app carries an editor,
// and the parts that matter most are not structural at all — the echo guard,
// the stale-line rule, and what Tab and Enter do — so they are driven directly
// against the harness DOM here.
//
// What is deliberately *not* here is anything about pixels. sizeCodeBuffer
// stretches the transparent textarea over the mirror's scroll size, and this
// DOM has no layout to measure; asserting on a number invented by a shim would
// pin the shim rather than the runtime. That claim belongs in a browser.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

const run = (t, extra = {}) => ({ t, ...extra });

// mountEditor builds a one-editor tree with the props given and the rows
// derived from the value, which is the shape core.CodeEditor always produces:
// one GridRow child per line of the buffer.
function mountEditor(value, props = {}, rows = null) {
    const rt = loadRuntime();
    const lines = value.split("\n");
    rt.GrMob.mount(JSON.stringify({
        Type: "Column",
        Children: [{
            Type: "CodeEditor",
            Props: {
                value,
                onChange: "cb-change",
                lineNumbers: false,
                readOnly: false,
                tabSize: 4,
                commentPrefix: "//",
                ...props,
            },
            Children: (rows ?? lines.map((line) => (line === "" ? [] : [run(line)])))
                .map((runs) => ({ Type: "GridRow", Props: { runs } })),
        }],
    }));
    rt.drainFrames();
    const editor = nodeAt(rt.document, "root/0");
    return { rt, editor, buffer: chrome(editor, "codebuffer"), gutter: chrome(editor, "codegutter") };
}

const chrome = (el, kind) => el.children.find((c) => c.dataset.grmobChrome === kind);
const fillers = (el) => el.children.filter((c) => c.dataset.grmobChrome === "codefiller");
const rowsOf = (el) => el.children.filter((c) => c.getAttribute("data-node-path") !== null);
const textOf = (el) => el.children.map((s) => s.textContent).join("");
const colours = (el) => el.children.map((s) => s.style.color ?? "");

// Typing, as the browser delivers it: the textarea's value has already moved
// by the time `input` fires.
function type(rt, buffer, value, caret = value.length) {
    buffer.value = value;
    buffer.selectionStart = caret;
    buffer.selectionEnd = caret;
    buffer.dispatch("input");
    rt.drainFrames();
}

test("an editor mounts as a <pre> holding its chrome and one row per line", () => {
    const { editor, buffer, gutter } = mountEditor("func f() {\n}");

    assert.equal(editor.tagName, "PRE");
    assert.equal(editor.style.position, "relative", "the gutter and the buffer are positioned against it");
    assert.equal(editor.style.overflow, "auto");
    assert.equal(editor.style.whiteSpace, "normal");

    // The chrome is leading, always both pieces, and carries no path.
    assert.equal(editor.children[0], gutter);
    assert.equal(editor.children[1], buffer);
    assert.equal(buffer.tagName, "TEXTAREA");
    assert.equal(gutter.getAttribute("data-node-path"), null);
    assert.equal(buffer.getAttribute("data-node-path"), null);
    assert.equal(gutter.style.display, "none", "no gutter until lineNumbers asks for one");

    // The four the buffer must refuse, each of which corrupts source.
    assert.equal(buffer.getAttribute("spellcheck"), "false");
    assert.equal(buffer.getAttribute("autocapitalize"), "off");
    assert.equal(buffer.getAttribute("autocorrect"), "off");
    assert.equal(buffer.getAttribute("wrap"), "off", "a code line is one line");

    // The buffer holds the text; the rows hold the picture of it.
    assert.equal(buffer.value, "func f() {\n}");
    const rows = rowsOf(editor);
    assert.equal(rows.length, 2);
    assert.deepEqual(rows.map(textOf), ["func f() {", "}"]);
});

test("lineNumbers draws the gutter and opens the padding it sits in", () => {
    const { editor, gutter } = mountEditor("a\nb\nc", { lineNumbers: true });

    assert.equal(gutter.style.display, "");
    assert.equal(gutter.textContent, "1\n2\n3", "one text node, newline-separated");
    assert.equal(gutter.getAttribute("aria-hidden"), "true", "the numbers are chrome, not code");
    assert.equal(gutter.style.width, "3ch", "one digit plus a column of room");
    assert.equal(editor.style.paddingLeft, "3ch", "the rows begin where the gutter ends");
    assert.equal(gutter.style.userSelect, "none", "copying the buffer must not take the numbers");
});

test("the gutter widens with the line count and disappears when asked to", () => {
    const { rt, editor, gutter } = mountEditor("x", { lineNumbers: true });
    assert.equal(gutter.style.width, "3ch");

    // Ten lines is two digits, so the gutter is one column wider.
    rt.GrMob.patch(JSON.stringify(
        Array.from({ length: 9 }, (_, i) => ({
            Type: "add-child",
            TargetID: "root/0",
            Changes: { Type: "GridRow", Props: { runs: [run(`line ${i}`)] } },
        }))
    ));
    rt.drainFrames();
    assert.equal(rowsOf(editor).length, 10);
    assert.equal(gutter.style.width, "4ch");

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0",
        Changes: { value: "x", lineNumbers: false },
    }]));
    rt.drainFrames();
    assert.equal(gutter.style.display, "none");
    assert.equal(editor.style.paddingLeft, "", "and the padding it opened goes with it");
});

// Rule 2 of the shared editor design: decoration is advisory and *per line*.
test("a line the buffer has moved past is drawn plain until Go catches up", () => {
    const { rt, editor, buffer } = mountEditor("alpha\nbeta", {}, [
        [run("alpha", { fg: "#CC7832" })],
        [run("beta", { fg: "#6A8759" })],
    ]);
    const [first, second] = rowsOf(editor);
    assert.deepEqual(colours(first), ["#CC7832"], "both rows start decorated");
    assert.deepEqual(colours(second), ["#6A8759"]);

    // One keystroke on line 2. Go has not been told yet, so its row for line 2
    // describes text that is no longer there.
    type(rt, buffer, "alpha\nbetas");
    assert.deepEqual(colours(first), ["#CC7832"], "the untouched line keeps its colours");
    assert.equal(textOf(second), "betas", "the edited line shows what was typed");
    assert.deepEqual(colours(second), [""], "...in plain ink, because Go's row disagrees");

    // Go catches up. The row's runs now match the line again, so the colours
    // come back — which is the reversibility the stored runs exist for.
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0/1",
        Changes: { runs: [run("betas", { fg: "#6A8759" })] },
    }]));
    rt.drainFrames();
    assert.deepEqual(colours(second), ["#6A8759"]);
});

// The same rule with the runs standing still: a row that disagrees and then
// agrees again — the user typed a character and deleted it — has to get its
// colours back with no patch from Go at all.
test("a line that comes back into agreement is re-decorated with no new patch", () => {
    const { rt, editor, buffer } = mountEditor("x := 1", {}, [
        [run("x := "), run("1", { fg: "#6897BB" })],
    ]);
    const [row] = rowsOf(editor);

    type(rt, buffer, "x := 12");
    assert.deepEqual(colours(row), [""], "disagrees, so plain");

    type(rt, buffer, "x := 1");
    assert.deepEqual(colours(row), ["", "#6897BB"], "agrees again, so decorated again");
});

test("a new line appears immediately, as a filler, before Go has a row for it", () => {
    const { rt, editor, buffer } = mountEditor("one");
    assert.equal(fillers(editor).length, 0);

    type(rt, buffer, "one\ntwo");
    assert.equal(rowsOf(editor).length, 1, "Go has not sent a second row yet");
    const blanks = fillers(editor);
    assert.equal(blanks.length, 1);
    assert.equal(textOf(blanks[0]), "two", "the glyphs are on screen anyway");
    assert.equal(blanks[0].getAttribute("data-node-path"), null, "a filler is chrome");

    // Go catches up with a real row, and the stand-in goes.
    rt.GrMob.patch(JSON.stringify([{
        Type: "add-child", TargetID: "root/0",
        Changes: { Type: "GridRow", Props: { runs: [run("two")] } },
    }]));
    rt.drainFrames();
    assert.equal(fillers(editor).length, 0);
    assert.deepEqual(rowsOf(editor).map(textOf), ["one", "two"]);
});

// The reason nodeChildCount replaced `children.length - chromeOffset(el)`: an
// editor's fillers trail the node children, so the old formulation would name
// the added row after a slot that is already taken.
test("a row added while a filler stands takes the next node index, not the next DOM slot", () => {
    const { rt, editor, buffer } = mountEditor("one");
    type(rt, buffer, "one\ntwo\nthree");
    assert.equal(fillers(editor).length, 2);

    rt.GrMob.patch(JSON.stringify([{
        Type: "add-child", TargetID: "root/0",
        Changes: { Type: "GridRow", Props: { runs: [run("two")] } },
    }]));
    rt.drainFrames();

    const added = nodeAt(rt.document, "root/0/1");
    assert.ok(added, "the new row answers to root/0/1");
    assert.equal(textOf(added), "two");
    assert.equal(fillers(editor).length, 1, "one line still has no row");
});

test("typing dispatches onChange once, with the whole buffer", () => {
    const { rt, buffer } = mountEditor("a");
    type(rt, buffer, "ab");
    assert.deepEqual(rt.dispatched, [{ id: "cb-change", payload: { value: "ab" } }]);
});

// The echo guard, the same bookkeeping both natives keep.
test("Go's echo of our own edit does not move the caret; a rewrite does", () => {
    const { rt, editor, buffer } = mountEditor("a");
    buffer.focus();

    type(rt, buffer, "ab", 2);
    type(rt, buffer, "abc", 3);

    // Go echoes the *first* of the two back — it coalesced a render — which is
    // still our own text and must not be written over what we have since typed.
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0", Changes: { value: "ab" },
    }]));
    rt.drainFrames();
    assert.equal(buffer.value, "abc", "a late echo never snaps the buffer back");
    assert.equal(buffer.selectionStart, 3, "nor the caret");

    // And now a value we never sent: a validator, a cleared draft. That is Go
    // speaking for itself and it wins even mid-typing.
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0", Changes: { value: "ABC" },
    }]));
    rt.drainFrames();
    assert.equal(buffer.value, "ABC");
    assert.equal(buffer.selectionStart, 3, "the caret lands after the replaced text");
    assert.equal(editor.children.find((c) => c.dataset.grmobChrome === "codefiller"), undefined);
});

test("an unfocused editor is Go's outright", () => {
    const { rt, buffer } = mountEditor("a");
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0", Changes: { value: "from Go" },
    }]));
    rt.drainFrames();
    assert.equal(buffer.value, "from Go");
});

test("an editable editor stays in the tab order", () => {
    const { buffer } = mountEditor("x");
    assert.equal(buffer.getAttribute("tabindex"), "0");
});

test("Tab inserts an indent instead of leaving the editor", () => {
    const { rt, buffer } = mountEditor("x", { tabSize: 2 });
    buffer.value = "x";
    buffer.selectionStart = buffer.selectionEnd = 0;

    const e = buffer.dispatch("keydown", { key: "Tab" });
    rt.drainFrames();
    assert.equal(e.defaultPrevented, true, "or focus would leave the editor");
    assert.equal(buffer.value, "  x");
    assert.equal(buffer.selectionStart, 2);
    assert.deepEqual(rt.dispatched, [{ id: "cb-change", payload: { value: "  x" } }]);
});

test("a zero tab size inserts a literal tab, which is what Go source wants", () => {
    const { rt, buffer } = mountEditor("x", { tabSize: 0 });
    buffer.selectionStart = buffer.selectionEnd = 0;
    buffer.dispatch("keydown", { key: "Tab" });
    rt.drainFrames();
    assert.equal(buffer.value, "\tx");
});

test("Enter continues the previous line's indentation", () => {
    const { rt, buffer } = mountEditor("    if x {");
    buffer.selectionStart = buffer.selectionEnd = buffer.value.length;

    const e = buffer.dispatch("keydown", { key: "Enter" });
    rt.drainFrames();
    assert.equal(e.defaultPrevented, true);
    assert.equal(buffer.value, "    if x {\n    ", "a bare newline would un-indent the block");
    assert.equal(buffer.selectionStart, 15);
});

test("a read-only buffer selects but refuses every edit", () => {
    const { rt, buffer } = mountEditor("x", { readOnly: true });
    assert.equal(buffer.readOnly, true);
    assert.equal(buffer.disabled, false, "read-only is not disabled: it still focuses and selects");
    assert.equal(buffer.getAttribute("tabindex"), "-1",
        "a page of code blocks must not put a tab stop in front of each one");

    buffer.selectionStart = buffer.selectionEnd = 0;
    buffer.dispatch("keydown", { key: "Tab" });
    buffer.dispatch("keydown", { key: "Enter" });
    rt.drainFrames();
    assert.equal(buffer.value, "x");
    assert.deepEqual(rt.dispatched, []);
});

// --- Commands -------------------------------------------------------------

// epochCommand issues one core.RunEditorCommand as it reaches the page.
function epochCommand(rt, epoch, command, extra = {}) {
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0",
        Changes: {
            value: nodeAt(rt.document, "root/0").children[1].value,
            lineNumbers: false, readOnly: false, tabSize: 4, commentPrefix: "//",
            editorEpoch: epoch, editorCommand: command, ...extra,
        },
    }]));
    rt.drainFrames();
}

test("indent and outdent act on every line the selection touches", () => {
    const { rt, buffer } = mountEditor("a\nb\nc");
    // A selection from the middle of line 1 into the middle of line 2: both
    // lines are touched, line 3 is not.
    buffer.selectionStart = 0;
    buffer.selectionEnd = 3;

    epochCommand(rt, 1, "indent");
    assert.equal(buffer.value, "    a\n    b\nc");
    // The rewritten block takes the selection, so a second indent indents the
    // same lines rather than a range that has drifted.
    epochCommand(rt, 2, "indent");
    assert.equal(buffer.value, "        a\n        b\nc");

    epochCommand(rt, 3, "outdent");
    assert.equal(buffer.value, "    a\n    b\nc");
    epochCommand(rt, 4, "outdent");
    epochCommand(rt, 5, "outdent");
    assert.equal(buffer.value, "a\nb\nc", "a line with no indent left is not eaten into");
});

test("commentLine toggles the whole run on the prefix it was given", () => {
    const { rt, buffer } = mountEditor("  a\n\n  b");
    buffer.selectionStart = 0;
    buffer.selectionEnd = buffer.value.length;

    epochCommand(rt, 1, "commentLine");
    assert.equal(buffer.value, "  // a\n\n  // b", "inserted at the indent, and blank lines left blank");

    epochCommand(rt, 2, "commentLine");
    assert.equal(buffer.value, "  a\n\n  b", "the command is its own undo");
});

test("a partly-commented run is commented rather than inverted", () => {
    const { rt, buffer } = mountEditor("// a\nb");
    buffer.selectionStart = 0;
    buffer.selectionEnd = buffer.value.length;
    epochCommand(rt, 1, "commentLine");
    assert.equal(buffer.value, "// // a\n// b");
});

test("an empty comment prefix makes commentLine a no-op", () => {
    const { rt, buffer } = mountEditor("{}", { commentPrefix: "" });
    buffer.selectionStart = 0;
    buffer.selectionEnd = 2;
    epochCommand(rt, 1, "commentLine", { commentPrefix: "" });
    assert.equal(buffer.value, "{}", "JSON has no line comment, and inventing one would invalidate it");
});

test("selectAll selects the buffer, and is allowed on a read-only one", () => {
    const { rt, buffer } = mountEditor("one\ntwo", { readOnly: true, onSelectionChange: "cb-sel" });
    epochCommand(rt, 1, "selectAll", { readOnly: true, onSelectionChange: "cb-sel" });
    assert.equal(buffer.selectionStart, 0);
    assert.equal(buffer.selectionEnd, 7);
    assert.deepEqual(rt.dispatched, [{ id: "cb-sel", payload: { value: "0:7" } }]);
});

test("a command runs once per epoch, and an unknown one does nothing", () => {
    const { rt, buffer } = mountEditor("a\nb");
    buffer.selectionStart = buffer.selectionEnd = 0;

    epochCommand(rt, 1, "indent");
    assert.equal(buffer.value, "    a\nb");

    // The same epoch again — an editor re-rendered for its value carries the
    // whole props map, the stamp included. It must not re-run.
    epochCommand(rt, 1, "indent");
    assert.equal(buffer.value, "    a\nb");

    // A new epoch with the same command must run: that is what makes "indent
    // twice" expressible at all.
    epochCommand(rt, 2, "indent");
    assert.equal(buffer.value, "        a\nb");

    epochCommand(rt, 3, "makeItPretty");
    assert.equal(buffer.value, "        a\nb", "a toolbar that outgrew its editor is a no-op, not a crash");
});

test("epoch 0 is never an instruction", () => {
    const { rt, buffer } = mountEditor("a");
    buffer.selectionStart = buffer.selectionEnd = 0;
    epochCommand(rt, 0, "indent");
    assert.equal(buffer.value, "a");
});

// The one way an editor command differs from a focus command: a focus command
// re-fires on a field that mounts while it is the target, and an editor command
// does not, because it names a moment and an edit rather than a standing state.
// Without this, returning to a screen would re-indent its buffer.
test("an editor created under a standing epoch adopts it without running it", () => {
    const { rt, buffer } = mountEditor("a", { editorEpoch: 7, editorCommand: "indent" });
    assert.equal(buffer.value, "a", "the command was issued before this editor existed");

    // And the stamp was adopted, so the *next* command still lands.
    buffer.selectionStart = buffer.selectionEnd = 0;
    epochCommand(rt, 8, "indent");
    assert.equal(buffer.value, "    a");
});

// --- Selection ------------------------------------------------------------

test("the selection is reported as UTF-8 byte offsets, deduped", () => {
    const { rt, buffer } = mountEditor("héllo → x", { onSelectionChange: "cb-sel" });

    // "héllo " is 7 bytes (é is two); "héllo → " is 7 + 3 + 1 = 11.
    buffer.selectionStart = 6;  // after "héllo " in UTF-16 units
    buffer.selectionEnd = 7;    // after the arrow
    buffer.dispatch("keyup");
    assert.deepEqual(rt.dispatched, [{ id: "cb-sel", payload: { value: "7:10" } }]);

    // The four events this is wired to overlap; an unchanged selection is not
    // news and must not cost a Go render pass.
    buffer.dispatch("mouseup");
    buffer.dispatch("select");
    assert.equal(rt.dispatched.length, 1);

    buffer.selectionStart = buffer.selectionEnd = 0;
    buffer.dispatch("keyup");
    assert.deepEqual(rt.dispatched[1], { id: "cb-sel", payload: { value: "0:0" } });
});

test("an editor with no selection handler reports nothing", () => {
    const { rt, buffer } = mountEditor("abc");
    buffer.selectionStart = 1;
    buffer.selectionEnd = 2;
    buffer.dispatch("keyup");
    assert.deepEqual(rt.dispatched, []);
});
