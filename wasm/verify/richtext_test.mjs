// core.RichTextEditor through the real runtime.
//
// The interesting half of this node is the *pure* half — the Doc
// transformations every command is — and that is deliberate: a contenteditable
// is the least predictable surface on the web, so the runtime reads the
// selection, transforms a document, and restores the selection, rather than
// operating on a Range. What is driven here is therefore the whole command
// vocabulary, both serializers, and the echo guard.
//
// What is NOT here is anything about the caret. This DOM has no Selection API,
// which is why the runtime's two selection functions return null rather than
// guessing; a test that invented a caret would pin the invention. The offsets
// used below are handed to the exported transformations directly, which is the
// same thing the browser half does one step later.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

// A document in the wire shape core/richtext.go marshals: short keys, marks as
// 1, an absent kind meaning paragraph.
const doc = (...blocks) => ({ b: blocks });
const block = (k, ...runs) => (k === "p" ? { r: runs } : { k, r: runs });
const run = (t, marks = {}) => ({ t, ...marks });

function mountEditor(value, props = {}) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "Column",
        Children: [{
            Type: "RichTextEditor",
            Props: {
                doc: JSON.stringify(value),
                onChange: "cb-change",
                readOnly: false,
                placeholder: "",
                ...props,
            },
        }],
    }));
    rt.drainFrames();
    return { rt, editor: nodeAt(rt.document, "root/0") };
}

// A compact description of the editor's DOM: tag[text] nested, chrome markers
// dropped. Reads like the markup it is.
function shape(el) {
    return el.children.map(describe).join("");
}

function describe(el) {
    const tag = el.tagName.toLowerCase();
    const inner = el.children.length ? el.children.map(describe).join("") : el.textContent;
    const href = el.getAttribute("href");
    return `<${tag}${href ? `:${href}` : ""}>${inner}</${tag}>`;
}

test("a document mounts as its block elements, and every one is chrome", () => {
    const { editor } = mountEditor(doc(
        block("h2", run("Notes")),
        block("p", run("A "), run("bold", { b: 1 }), run(" word")),
        block("bullet", run("one")),
        block("bullet", run("two")),
        block("numbered", run("first")),
        block("quote", run("said")),
        block("code", run("x := 1\ny := 2")),
    ));

    assert.equal(editor.tagName, "DIV");
    assert.equal(editor.getAttribute("contenteditable"), "true");
    assert.equal(editor.getAttribute("role"), "textbox");
    assert.equal(editor.getAttribute("aria-multiline"), "true");

    assert.equal(shape(editor),
        "<h2><span>Notes</span></h2>" +
        "<p><span>A </span><strong><span>bold</span></strong><span> word</span></p>" +
        // Consecutive items of one kind are gathered into one list, which the
        // model does not express and every renderer has to.
        "<ul><li><span>one</span></li><li><span>two</span></li></ul>" +
        "<ol><li><span>first</span></li></ol>" +
        "<blockquote><span>said</span></blockquote>" +
        "<pre><code>x := 1\ny := 2</code></pre>");

    // None of it is a node: a RichTextEditor's value is one prop, so nothing is
    // ever addressed inside it.
    for (const child of editor.children) {
        assert.equal(child.getAttribute("data-node-path"), null);
        assert.equal(child.dataset.grmobChrome, "richblock");
    }
});

test("marks nest link-outermost, code-innermost — richtext's own order", () => {
    const { editor } = mountEditor(doc(block("p",
        run("x", { b: 1, i: 1, u: 1, s: 1, c: 1, l: "https://example.com" }),
    )));
    assert.equal(shape(editor),
        "<p><a:https://example.com><strong><em><s><u><code><span>x</span>" +
        "</code></u></s></em></strong></a></p>");
});

test("an empty block keeps a line the writer typed", () => {
    const { editor } = mountEditor(doc(block("p", run("a")), block("p"), block("p", run("b"))));
    assert.equal(shape(editor), "<p><span>a</span></p><p><br></br></p><p><span>b</span></p>");
});

test("a read-only editor is not editable and says so", () => {
    const { editor } = mountEditor(doc(block("p", run("x"))), { readOnly: true });
    assert.equal(editor.getAttribute("contenteditable"), "false");
    assert.equal(editor.getAttribute("aria-readonly"), "true");
});

test("the placeholder appears only while the document is empty", () => {
    const { rt, editor } = mountEditor(doc(), { placeholder: "Write something…" });
    const prompt = editor.children.find((c) => c.dataset.grmobChrome === "richplaceholder");
    assert.ok(prompt, "an empty editor shows its prompt");
    assert.equal(prompt.textContent, "Write something…");
    assert.equal(prompt.style.pointerEvents, "none", "a tap on it must reach the editor");

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0",
        Changes: {
            doc: JSON.stringify(doc(block("p", run("now there is text")))),
            placeholder: "Write something…", readOnly: false, onChange: "cb-change",
        },
    }]));
    rt.drainFrames();
    assert.equal(editor.children.filter((c) => c.dataset.grmobChrome === "richplaceholder").length, 0);
});

// The echo guard, over the document's JSON rather than a string of text. Same
// three arms a TextArea's has.
test("Go's echo of our own edit does not rebuild the document", () => {
    const { rt, editor } = mountEditor(doc(block("p", run("a"))));
    editor.focus();

    // The editor sends an edit of its own by way of a paste, which is the one
    // local edit this DOM can drive: it needs no Selection API because the
    // runtime falls back to offset 0 when there is none.
    editor.dispatch("paste", {
        clipboardData: { getData: () => "hi " },
        preventDefault() { this.defaultPrevented = true; },
    });
    assert.equal(rt.dispatched.length, 1);
    const sent = rt.dispatched[0].payload.value;
    assert.equal(shape(editor), "<p><span>hi a</span></p>");

    // Go echoes it back. Nothing should be rebuilt — and the proof is that the
    // element identity of the block survives.
    const paragraph = editor.children[0];
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0",
        Changes: { doc: sent, readOnly: false, placeholder: "", onChange: "cb-change" },
    }]));
    rt.drainFrames();
    assert.equal(editor.children[0], paragraph, "an echo must not rebuild the DOM");

    // And now a document we never sent: Go speaking for itself. That wins even
    // while focused, and does rebuild.
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0",
        Changes: {
            doc: JSON.stringify(doc(block("h1", run("replaced")))),
            readOnly: false, placeholder: "", onChange: "cb-change",
        },
    }]));
    rt.drainFrames();
    assert.equal(shape(editor), "<h1><span>replaced</span></h1>");
});

test("a doc prop that is not a document leaves the editor alone", () => {
    const { rt, editor } = mountEditor(doc(block("p", run("keep me"))));
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0",
        Changes: { doc: "{oops", readOnly: false, placeholder: "", onChange: "cb-change" },
    }]));
    rt.drainFrames();
    assert.equal(shape(editor), "<p><span>keep me</span></p>",
        "one malformed patch must not empty a note");
});

// Paste is the edge foreign markup dies at: whatever was on the clipboard, what
// reaches the document is its text.
test("paste inserts text and never markup", () => {
    const { rt, editor } = mountEditor(doc(block("p", run("x"))));
    let prevented = false;
    editor.dispatch("paste", {
        clipboardData: { getData: (type) => (type === "text/plain" ? "a\nb" : "<b>no</b>") },
        preventDefault() { prevented = true; },
    });

    assert.equal(prevented, true, "the browser's own paste must not run");
    // A multi-line paste splits the block, because a block is a line.
    assert.equal(shape(editor), "<p><span>a</span></p><p><span>bx</span></p>");
    assert.equal(rt.dispatched.length, 1);
    assert.equal(rt.dispatched[0].id, "cb-change");
});

// --- The serializers, in both directions ----------------------------------

// Typing, as the browser delivers it: the DOM has already changed by the time
// `input` fires, and the runtime reads it back rather than rebuilding.
test("typing reads the DOM back into a document without rebuilding it", () => {
    const { rt, editor } = mountEditor(doc(block("p", run("a"))));
    const paragraph = editor.children[0];
    paragraph.children[0].textContent = "abc";
    editor.dispatch("input");

    assert.equal(editor.children[0], paragraph, "a keystroke must not rebuild the document");
    assert.deepEqual(JSON.parse(rt.dispatched[0].payload.value), doc(block("p", run("abc"))));
});

// The serializer normalizes rather than trusting structure, which is the rule
// the whole DOM->Doc direction is written under: a browser under
// contenteditable produces <b> for bold, leaves stray <div>s, and inserts <br>s
// nobody asked for.
test("the serializer reads what a browser leaves behind, not what it was given", () => {
    const { rt, editor } = mountEditor(doc(block("p", run("x"))));
    editor.innerHTML = "";

    const messy = (tag, text, attrs = {}) => {
        const el = rt.document.createElement(tag);
        if (text !== null) el.textContent = text;
        for (const [k, v] of Object.entries(attrs)) el.setAttribute(k, v);
        return el;
    };
    const div = messy("div", null);
    div.appendChild(messy("b", "bold"));
    div.appendChild(messy("i", "italic"));
    div.appendChild(messy("strike", "gone"));
    div.appendChild(messy("br", null));
    const link = messy("a", null, { href: "https://x" });
    link.appendChild(messy("ins", "underlined"));
    div.appendChild(link);
    editor.appendChild(div);
    editor.appendChild(messy("h3", "heading"));
    const marquee = messy("marquee", "still text");
    editor.appendChild(marquee);

    editor.dispatch("input");
    assert.deepEqual(JSON.parse(rt.dispatched[0].payload.value), doc(
        block("p",
            run("bold", { b: 1 }),
            run("italic", { i: 1 }),
            run("gone", { s: 1 }),
            run("underlined", { u: 1, l: "https://x" })),
        block("h3", run("heading")),
        // A tag this model knows nothing about contributes its text and none of
        // its own meaning — the same degradation an unknown block kind gets.
        block("p", run("still text")),
    ));
});

test("a round trip through both serializers is the identity", () => {
    const original = doc(
        block("h1", run("Title")),
        block("p", run("plain "), run("bold", { b: 1 }), run(" and "),
            run("linked", { l: "https://example.com" })),
        block("bullet", run("one")),
        block("bullet", run("two", { i: 1 })),
        block("numbered", run("first")),
        block("quote", run("quoted")),
        block("code", run("verbatim")),
    );
    const { rt, editor } = mountEditor(original);
    editor.dispatch("input");
    assert.deepEqual(JSON.parse(rt.dispatched[0].payload.value), original);
});

// --- Commands --------------------------------------------------------------
//
// Driven through the epoch prop, which is how they reach the page. The DOM here
// has no Selection API, so the runtime's offsets fall back to 0:0 — a bare
// caret at the start — which is exactly the state the block and link commands
// act on and the state the mark commands treat as "set the typing attributes".

// The doc prop carries whatever the editor last told Go, which is what a real
// pass carries: Go re-renders from the value it was given, and the echo guard
// is what keeps that from rebuilding the DOM. Sending anything else here would
// make every command test also a test of a rewrite.
function command(rt, editor, epoch, cmd, extra = {}) {
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0",
        Changes: {
            doc: JSON.stringify(editor.__richDoc),
            readOnly: false, placeholder: "", onChange: "cb-change",
            editorEpoch: epoch, editorCommand: cmd, ...extra,
        },
    }]));
    rt.drainFrames();
}

test("a block command reaches the block the caret is in", () => {
    const { rt, editor } = mountEditor(doc(block("p", run("a")), block("p", run("b"))));
    command(rt, editor, 1, "block:h2");
    assert.equal(shape(editor), "<h2><span>a</span></h2><p><span>b</span></p>");

    // And back to a paragraph, which is the wire's absent kind — so the
    // document that comes out is byte-identical to one Go marshalled.
    command(rt, editor, 2, "block:p");
    assert.deepEqual(JSON.parse(rt.dispatched[rt.dispatched.length - 1].payload.value),
        doc(block("p", run("a")), block("p", run("b"))));
});

test("a block command turns a paragraph into a list item and back", () => {
    const { rt, editor } = mountEditor(doc(block("p", run("a"))));
    command(rt, editor, 1, "block:bullet");
    assert.equal(shape(editor), "<ul><li><span>a</span></li></ul>");
    command(rt, editor, 2, "block:code");
    assert.equal(shape(editor), "<pre><code>a</code></pre>");
});

test("a mark command with nothing selected sets the typing attributes", () => {
    const { rt, editor } = mountEditor(doc(block("p", run("ab"))),
        { onSelectionChange: "cb-sel" });
    rt.dispatched.length = 0;

    command(rt, editor, 1, "bold", { onSelectionChange: "cb-sel" });
    // Nothing is selected, so the document is untouched — and the toolbar is
    // told the mark is pending, because the button has to look pressed or the
    // press looks like it did nothing.
    assert.equal(shape(editor), "<p><span>ab</span></p>");
    const report = rt.dispatched.find((d) => d.id === "cb-sel");
    assert.ok(report, "a pending mark is reported");
    assert.deepEqual(JSON.parse(report.payload.value).marks, ["bold"]);

    // Pressing it again takes it back off.
    command(rt, editor, 2, "bold", { onSelectionChange: "cb-sel" });
    const second = rt.dispatched.filter((d) => d.id === "cb-sel").pop();
    assert.deepEqual(JSON.parse(second.payload.value).marks, []);
});

test("an unknown command is a no-op, and epoch 0 is never an instruction", () => {
    const { rt, editor } = mountEditor(doc(block("p", run("a"))));
    command(rt, editor, 1, "makeItPretty");
    assert.equal(shape(editor), "<p><span>a</span></p>");
    command(rt, editor, 0, "block:h1");
    assert.equal(shape(editor), "<p><span>a</span></p>");
});

test("an editor created under a standing epoch adopts it without running it", () => {
    const { rt, editor } = mountEditor(doc(block("p", run("a"))),
        { editorEpoch: 5, editorCommand: "block:h1" });
    assert.equal(shape(editor), "<p><span>a</span></p>",
        "the command was issued before this editor existed");
    command(rt, editor, 6, "block:h1");
    assert.equal(shape(editor), "<h1><span>a</span></h1>");
});

test("a read-only editor refuses every command", () => {
    const { rt, editor } = mountEditor(doc(block("p", run("a"))), { readOnly: true });
    command(rt, editor, 1, "block:h1", { readOnly: true });
    assert.equal(shape(editor), "<p><span>a</span></p>");
});

test("undo and redo walk a stack of documents", () => {
    const { rt, editor } = mountEditor(doc(block("p", run("a"))));
    command(rt, editor, 1, "block:h1");
    assert.equal(shape(editor), "<h1><span>a</span></h1>");

    command(rt, editor, 2, "undo");
    assert.equal(shape(editor), "<p><span>a</span></p>");
    command(rt, editor, 3, "redo");
    assert.equal(shape(editor), "<h1><span>a</span></h1>");

    // An empty stack is a no-op rather than a crash.
    command(rt, editor, 4, "undo");
    command(rt, editor, 5, "undo");
    assert.equal(shape(editor), "<p><span>a</span></p>");
});

// --- Commands over a real range --------------------------------------------
//
// The editor remembers the selection as it moves, because clicking a toolbar
// button can take focus out of a contenteditable and collapse the live one
// before the handler runs. That memory is what these drive: a test sets it and
// the real command path runs against it, offsets and all.

const select = (editor, start, end) => { editor.__richSelection = { start, end }; };

// The editor's current document, through JSON. The value itself was built
// inside the vm context, so a strict deep-equal against a host-realm literal
// fails on prototype identity alone — the same realm bookkeeping load.mjs
// serializes away at the callback boundary.
const docOf = (editor) => JSON.parse(JSON.stringify(editor.__richDoc));

test("a mark command toggles the whole selection, and toggles it back", () => {
    const { rt, editor } = mountEditor(doc(block("p", run("hello world"))));
    select(editor, 0, 5);

    command(rt, editor, 1, "bold");
    assert.equal(shape(editor),
        "<p><strong><span>hello</span></strong><span> world</span></p>");

    command(rt, editor, 2, "bold");
    assert.equal(shape(editor), "<p><span>hello world</span></p>",
        "a toggle is its own undo, and the runs merge back into one");
});

// All, not any: selecting a sentence with one bold word in it and pressing bold
// makes the sentence bold rather than unbolding the word.
test("a partly-marked selection is marked rather than unmarked", () => {
    const { rt, editor } = mountEditor(doc(block("p", run("a"), run("b", { b: 1 }), run("c"))));
    select(editor, 0, 3);
    command(rt, editor, 1, "bold");
    assert.deepEqual(docOf(editor), doc(block("p", run("abc", { b: 1 }))));
});

test("a mark spanning two blocks reaches both and neither neighbour", () => {
    const { rt, editor } = mountEditor(doc(
        block("p", run("one")), block("p", run("two")), block("p", run("three")),
    ));
    // "e" of one, the newline, "tw" of two.
    select(editor, 2, 6);
    command(rt, editor, 1, "italic");
    assert.deepEqual(docOf(editor), doc(
        block("p", run("on"), run("e", { i: 1 })),
        block("p", run("tw", { i: 1 }), run("o")),
        block("p", run("three")),
    ));
});

test("link and unlink act on the selection", () => {
    const { rt, editor } = mountEditor(doc(block("p", run("see the docs"))));
    select(editor, 8, 12);

    command(rt, editor, 1, "link:https://example.com");
    assert.equal(shape(editor),
        "<p><span>see the </span><a:https://example.com><span>docs</span></a></p>");

    command(rt, editor, 2, "unlink");
    assert.equal(shape(editor), "<p><span>see the docs</span></p>");
});

test("a block command reaches every block the selection touches", () => {
    const { rt, editor } = mountEditor(doc(
        block("p", run("a")), block("p", run("b")), block("p", run("c")),
    ));
    select(editor, 0, 3); // through the start of the second block
    command(rt, editor, 1, "block:bullet");
    assert.equal(shape(editor),
        "<ul><li><span>a</span></li><li><span>b</span></li></ul><p><span>c</span></p>");
});

test("the selection report carries the marks under the caret", () => {
    const { rt, editor } = mountEditor(
        doc(block("h2", run("Title", { b: 1, l: "https://x" })), block("p", run("body"))),
        { onSelectionChange: "cb-sel" });
    rt.dispatched.length = 0;

    select(editor, 0, 5);
    // Any of the events a caret move arrives on; they all report the same thing
    // and the report is deduped.
    editor.dispatch("keyup");
    editor.dispatch("mouseup");

    const reports = rt.dispatched.filter((d) => d.id === "cb-sel");
    assert.equal(reports.length, 1, "an unchanged selection is not news");
    assert.deepEqual(JSON.parse(reports[0].payload.value), {
        s: 0, e: 5, marks: ["bold"], link: "https://x", block: "h2",
    });
});

test("a rewrite from Go forgets the remembered selection", () => {
    const { rt, editor } = mountEditor(doc(block("p", run("hello"))));
    select(editor, 0, 5);
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0",
        Changes: {
            doc: JSON.stringify(doc(block("p", run("something else entirely")))),
            readOnly: false, placeholder: "", onChange: "cb-change",
        },
    }]));
    rt.drainFrames();

    // The offsets described text that is gone, so a command after a rewrite acts
    // on a bare caret rather than on a position in a document that no longer
    // exists.
    command(rt, editor, 1, "bold");
    assert.equal(shape(editor), "<p><span>something else entirely</span></p>");
});
