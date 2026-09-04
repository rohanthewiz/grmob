// core.TextGrid through the real runtime: a <pre> of row <div>s, each row
// rebuilt from its runs prop on create and on update-props. The replay test
// cannot cover this — the example apps carry no grid, and a row's spans are
// deliberately outside the node tree — so the row path is driven directly.

import test from "node:test";
import assert from "node:assert/strict";

import { loadRuntime, nodeAt } from "./load.mjs";

const run = (t, extra = {}) => ({ t, ...extra });

function mountGrid(rows) {
    const rt = loadRuntime();
    rt.GrMob.mount(JSON.stringify({
        Type: "Column",
        Children: [{
            Type: "TextGrid",
            Props: { rows: rows.length },
            Style: { FontSize: 12 },
            Children: rows.map((runs) => ({ Type: "GridRow", Props: { runs } })),
        }],
    }));
    rt.drainFrames();
    return { rt, grid: nodeAt(rt.document, "root/0") };
}

// The run's own white-space:pre is on every span and says nothing about the
// run, so it is dropped here rather than restated in every expectation below.
// It is asserted once, on its own, in the white-space test at the end.
const spans = (row) => row.children.map((s) => {
    const { whiteSpace, ...style } = { ...s.style };
    return { text: s.textContent, style };
});

test("a grid mounts as a <pre> of rows holding one span per run", () => {
    const { grid } = mountGrid([
        [run("$ ls")],
        [run("a.go", { fg: "#00ff00", bg: "#000000" }), run(" b.go", { a: 1 | 8 | 16 })],
        [],
        [run("dim", { a: 2 | 4 })],
    ]);

    assert.equal(grid.tagName, "PRE");
    assert.equal(grid.style.whiteSpace, "normal");
    assert.equal(grid.style.fontSize, "12px", "the author's style still applies");
    assert.equal(grid.children.length, 4);
    for (const row of grid.children) {
        assert.equal(row.tagName, "DIV");
        assert.equal(row.style.minHeight, "1.2em", "an empty row keeps its line");
        assert.equal(row.style.whiteSpace, "nowrap", "a row is one line, whatever its runs");
    }

    assert.deepEqual(spans(grid.children[0]), [{ text: "$ ls", style: {} }]);
    assert.deepEqual(spans(grid.children[1]), [
        { text: "a.go", style: { color: "#00ff00", background: "#000000" } },
        { text: " b.go", style: { fontWeight: "700", textDecoration: "underline line-through" } },
    ]);
    assert.deepEqual(spans(grid.children[2]), []);
    assert.deepEqual(spans(grid.children[3]), [
        { text: "dim", style: { opacity: "0.6", fontStyle: "italic" } },
    ]);
});

test("an update-props on one row rebuilds that row's spans and no other", () => {
    const { rt, grid } = mountGrid([[run("row 0")], [run("before")], [run("row 2")]]);
    const untouched = grid.children[0];

    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props",
        TargetID: "root/0/1",
        Changes: { runs: [run("after", { fg: "#ff0000" }), run("!")] },
    }]));
    rt.drainFrames();

    assert.equal(grid.children[0], untouched, "the unchanged row is the same element");
    assert.deepEqual(spans(grid.children[1]), [
        { text: "after", style: { color: "#ff0000" } },
        { text: "!", style: {} },
    ]);
    assert.deepEqual(spans(grid.children[2]), [{ text: "row 2", style: {} }]);
});

test("runs that are not an array clear the row rather than throwing", () => {
    const { rt, grid } = mountGrid([[run("x")]]);
    rt.GrMob.patch(JSON.stringify([{
        Type: "update-props", TargetID: "root/0/0", Changes: { runs: "nope" },
    }]));
    assert.deepEqual(spans(grid.children[0]), []);
});

// The three white-space levels, which htmlout states the same way
// (textGridChassis). Each says something different: the grid declines the
// <pre>'s own `pre` default because the white space between rows is markup
// formatting, the row is `nowrap` because a row is one line and must not
// break between two runs, and the run is `pre` because a run's spaces are the
// one kind of white space in a grid that is content — an indent, the gap
// between two coloured tokens, a terminal's blank cells.
//
// This runtime has no formatter that could disturb a grid; it carries the
// rules so that a grid drawn here and a grid exported by htmlout are the same
// picture.
test("white space is significant in a run and nowhere else", () => {
    const { grid } = mountGrid([[run("    "), run("func", { fg: "#CC7832" })]]);

    assert.equal(grid.style.whiteSpace, "normal");
    assert.equal(grid.children[0].style.whiteSpace, "nowrap");
    for (const span of grid.children[0].children) {
        assert.equal(span.style.whiteSpace, "pre");
    }
    assert.equal(grid.children[0].children[0].textContent, "    ",
        "a run of nothing but spaces keeps them");
});
